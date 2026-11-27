package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hash/maphash"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"rockshinops/internal/metrics"
)

// Review state is local to each scanner. File writes share one lock so every
// operation either updates the volume tree or invalidates it before returning.
type fileOperations struct {
	writes  sync.Mutex
	reviews atomic.Uint64
	space   spaceState
	quick   quickCleanup
	appMu   sync.Mutex
	app     appRemoval
}

type spaceState struct {
	mu                sync.Mutex
	tree              *metrics.SpaceTree
	scope             spaceScanScope
	id, revision      uint64
	skipped           int64
	started, finished time.Time
	progress          atomic.Pointer[metrics.SpaceProgress]
	cancelMu          sync.Mutex
	cancel            context.CancelFunc
	trash             *trashReview
}

func (ops *fileOperations) invalidateSpace() {
	ops.space.mu.Lock()
	defer ops.space.mu.Unlock()
	ops.space.revision++
	ops.space.tree = nil
	ops.space.trash = nil
}

type FileOutcome struct {
	Succeeded []string         `json:"succeeded"`
	Failed    []CleanupFailure `json:"failed"`
}

func checkSelection[T any](review map[string]T, paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no items selected")
	}
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		if _, ok := review[path]; !ok || seen[path] {
			return fmt.Errorf("selection does not match the review")
		}
		seen[path] = true
	}
	return nil
}

func applyReviewed(ctx context.Context, paths []string, verify, apply func(string) error) FileOutcome {
	outcome := FileOutcome{Succeeded: []string{}, Failed: []CleanupFailure{}}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			outcome.Failed = append(outcome.Failed, CleanupFailure{Path: path, Error: err.Error()})
		} else if err := verify(path); err != nil {
			outcome.Failed = append(outcome.Failed, CleanupFailure{Path: path, Changed: ctx.Err() == nil, Error: err.Error()})
		} else if err := apply(path); err != nil {
			outcome.Failed = append(outcome.Failed, CleanupFailure{Path: path, Error: err.Error()})
		} else {
			outcome.Succeeded = append(outcome.Succeeded, path)
		}
	}
	return outcome
}

type reviewedFile struct {
	path    string
	info    fs.FileInfo
	content [sha256.Size]byte
}

type fileTree struct {
	info   fs.FileInfo
	bytes  int64
	newest time.Time
	stamp  uint64
	files  []reviewedFile
}

type fileTreePurpose uint8

const (
	cleanupCandidateTree fileTreePurpose = iota
	trashVerificationTree
	trashManifestTree
)

var fileStampSeed = maphash.MakeSeed()

func inspectFileTree(ctx context.Context, root *os.Root, name string, purpose fileTreePurpose) (fileTree, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return fileTree{}, err
	}
	base, err := root.Stat(".")
	if err != nil {
		return fileTree{}, err
	}
	parentID, identified := metrics.FileIdentity(base)
	tree := fileTree{info: info, newest: info.ModTime()}
	var stamp maphash.Hash
	stamp.SetSeed(fileStampSeed)
	add := func(path string, info fs.FileInfo) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		id, _ := metrics.FileIdentity(info)
		if identified && id.Device != parentID.Device {
			return fmt.Errorf("item spans multiple volumes")
		}
		if info.ModTime().After(tree.newest) {
			tree.newest = info.ModTime()
		}
		if info.Mode().IsRegular() {
			if purpose == trashManifestTree {
				tree.bytes += metrics.FileAllocatedBytes(info)
			} else {
				tree.bytes += metrics.CleanupFileBytes(info)
			}
		}
		if purpose == trashManifestTree {
			tree.files = append(tree.files, reviewedFile{path: filepath.FromSlash(path), info: info})
		}
		fmt.Fprintf(&stamp, "%q %d %d %d %d %d %d\n", path, id.Device, id.Inode, info.Mode(), info.Size(), info.ModTime().UnixNano(), fileChangeTime(info))
		return nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		err = add(name, info)
	} else {
		err = fs.WalkDir(root.FS(), name, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			metadata, err := entry.Info()
			if err != nil {
				return err
			}
			if path == name && !os.SameFile(info, metadata) {
				return errTrashReviewChanged
			}
			if purpose == cleanupCandidateTree && !metadata.IsDir() && !metadata.Mode().IsRegular() && metadata.Mode()&os.ModeSymlink == 0 {
				return fmt.Errorf("item contains a special file")
			}
			return add(path, metadata)
		})
	}
	tree.stamp = stamp.Sum64()
	return tree, err
}

func (tree fileTree) matches(current fileTree) bool {
	return tree.stamp == current.stamp && os.SameFile(tree.info, current.info)
}
