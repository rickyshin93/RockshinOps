package main

import (
	"context"
	"crypto/rand"
	"io/fs"
	"os"
	"path/filepath"

	"rockshinops/internal/metrics"
)

type trashStage struct {
	root         *os.Root
	dir          *os.Root
	path         string
	move         func(string, string) error
	recovery     string
	deleted      int
	bytes        int64
	touchedLinks map[metrics.FileID]bool
	reader       trashContentReader
}

func newTrashStage(root *os.Root) (*trashStage, error) {
	path := ".rockshinops-trash-" + rand.Text()
	if err := root.Mkdir(path, 0700); err != nil {
		return nil, err
	}
	dir, err := root.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &trashStage{root: root, dir: dir, path: path, move: root.Rename, touchedLinks: make(map[metrics.FileID]bool)}, nil
}

func (stage *trashStage) removeTree(ctx context.Context, tree fileTree) error {
	for i := len(tree.files) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := stage.remove(ctx, tree.files[i]); err != nil {
			return err
		}
	}
	return nil
}

func sameFileMetadata(before, after fs.FileInfo) bool {
	return os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

// The atomic rename claims the actual directory entry. Only the private path
// is unlinked after its identity has been verified. Failed claims stay in the
// staging directory to avoid overwriting a replacement at the original path.
func (stage *trashStage) remove(ctx context.Context, file reviewedFile) error {
	if stage.recovery != "" {
		return errTrashRecoveryPending
	}
	current, err := stage.root.Lstat(file.path)
	if err != nil {
		return err
	}
	if !os.SameFile(file.info, current) {
		return errTrashReviewChanged
	}
	if current.IsDir() {
		// Reviewed children were processed first. Check for new entries in
		// the original location before moving it. Remove still requires
		// emptiness if another entry arrives after this check.
		entries, err := fs.ReadDir(stage.root.FS(), filepath.ToSlash(file.path))
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			return errTrashReviewChanged
		}
	} else {
		id, _ := metrics.FileIdentity(current)
		// The prior rename/unlink of a reviewed hard link changes the
		// shared inode's ctime for every other link to it.
		if !sameFileMetadata(file.info, current) || !stage.touchedLinks[id] && fileChangeTime(file.info) != fileChangeTime(current) {
			return errTrashReviewChanged
		}
	}

	name := filepath.Base(file.path)
	slot := filepath.Join(stage.path, name)
	if err := stage.move(file.path, slot); err != nil {
		return err
	}
	stage.recovery = filepath.Join(stage.root.Name(), slot)
	staged, err := stage.dir.Lstat(name)
	if err != nil {
		return err
	}
	// Renaming the inode itself may change ctime.
	if !sameFileMetadata(current, staged) {
		return errTrashReviewChanged
	}
	if staged.Mode().IsRegular() {
		content, err := stage.reader.hash(ctx, stage.dir, name, staged)
		if err != nil {
			return err
		}
		if content != file.content {
			return errTrashReviewChanged
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	latest, err := stage.dir.Lstat(name)
	if err != nil {
		return err
	}
	if !sameFileState(staged, latest) {
		return errTrashReviewChanged
	}
	if err := stage.dir.Remove(name); err != nil {
		return err
	}
	stage.recovery = ""
	stage.deleted++
	if staged.Mode().IsRegular() {
		stage.bytes += metrics.CleanupFileBytes(staged)
	}
	if id, ok := metrics.FileIdentity(staged); ok && !staged.IsDir() && metrics.CleanupLinkCount(staged) > 1 {
		stage.touchedLinks[id] = true
	}
	return nil
}

func (stage *trashStage) finish() error {
	defer stage.dir.Close()
	if stage.recovery != "" {
		return nil
	}
	created, err := stage.dir.Stat(".")
	if err != nil {
		return err
	}
	current, err := stage.root.Lstat(stage.path)
	if err != nil {
		return err
	}
	if !current.IsDir() || !os.SameFile(created, current) {
		return errTrashReviewChanged
	}
	return removeTrashDirectory(stage.root, stage.path)
}
