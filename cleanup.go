package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"rockshinops/internal/metrics"
)

const trashReviewPageSize = 24

var errTrashReviewChanged = errors.New("trash_review_changed")
var errTrashRecoveryPending = errors.New("trash_recovery_pending")

type TrashReviewItem struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

type TrashReview struct {
	ID    uint64            `json:"id"`
	Path  string            `json:"path"`
	Bytes int64             `json:"bytes"`
	Count int               `json:"count"`
	Items []TrashReviewItem `json:"items"`
}

type SpaceTrashOutcome struct {
	Summary            *SpaceSummary    `json:"summary,omitempty"`
	Bytes              int64            `json:"bytes"`
	Deleted            int              `json:"deleted"`
	Outcome            FileOutcome      `json:"outcome"`
	PostDeleteFailures []CleanupFailure `json:"postDeleteFailures"`
}

type trashReview struct {
	summary  TrashReview
	root     fs.FileInfo
	names    []string
	entries  map[string]fileTree
	metadata *trashMetadataReview
}

func openTrashRoot(path string) (*os.Root, fs.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, errTrashReviewChanged
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, nil, err
	}
	current, err := root.Stat(".")
	if err != nil || !os.SameFile(info, current) {
		root.Close()
		return nil, nil, errTrashReviewChanged
	}
	return root, current, nil
}

func reviewTrash(ctx context.Context, path, platform string) (*trashReview, error) {
	root, info, err := openTrashRoot(path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return nil, err
	}
	review := &trashReview{summary: TrashReview{Path: path, Count: len(entries)}, root: info, entries: make(map[string]fileTree, len(entries))}
	var reader trashContentReader
	type reviewedLink struct {
		file  reviewedFile
		count uint64
	}
	links := make(map[metrics.FileID]reviewedLink)
	for _, entry := range entries {
		name := entry.Name()
		item, err := inspectFileTree(ctx, root, name, trashManifestTree)
		if os.IsNotExist(err) || errors.Is(err, errTrashReviewChanged) {
			return nil, errTrashReviewChanged
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		for i := range item.files {
			file := &item.files[i]
			if !file.info.Mode().IsRegular() {
				continue
			}
			id, identified := metrics.FileIdentity(file.info)
			linkCount := metrics.CleanupLinkCount(file.info)
			linked := identified && linkCount > 1
			if old, ok := links[id]; linked && ok {
				if !sameFileState(file.info, old.file.info) {
					return nil, errTrashReviewChanged
				}
				file.content = old.file.content
				old.count++
				links[id] = old
				continue
			}
			digest, err := reader.hash(ctx, root, file.path, file.info)
			if os.IsNotExist(err) || errors.Is(err, errTrashReviewChanged) {
				return nil, errTrashReviewChanged
			}
			if err != nil {
				return nil, fmt.Errorf("%s: %w", file.path, err)
			}
			file.content = digest
			if linked {
				links[id] = reviewedLink{file: *file, count: 1}
			} else {
				review.summary.Bytes += metrics.CleanupFileBytes(file.info)
			}
		}
		review.names = append(review.names, name)
		review.entries[name] = item
	}
	for _, link := range links {
		if link.count == metrics.CleanupLinkCount(link.file.info) {
			review.summary.Bytes += metrics.FileAllocatedBytes(link.file.info)
		}
	}
	if platform == "linux" {
		review.metadata = reviewTrashMetadata(ctx, filepath.Join(filepath.Dir(path), "info"), review.names, &reader)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	review.summary.Items = review.items(0)
	return review, nil
}

func (review *trashReview) items(offset int) []TrashReviewItem {
	names := review.names[offset:min(offset+trashReviewPageSize, len(review.names))]
	items := make([]TrashReviewItem, 0, len(names))
	for _, name := range names {
		items = append(items, TrashReviewItem{Name: name, Bytes: review.entries[name].bytes})
	}
	return items
}

func (review *trashReview) empty(ctx context.Context) (SpaceTrashOutcome, bool, error) {
	path := review.summary.Path
	root, info, err := openTrashRoot(path)
	if os.IsNotExist(err) {
		err = errTrashReviewChanged
	}
	if err != nil {
		return SpaceTrashOutcome{}, false, err
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return SpaceTrashOutcome{}, false, err
	}
	if !os.SameFile(review.root, info) || len(entries) != len(review.names) {
		return SpaceTrashOutcome{}, false, errTrashReviewChanged
	}
	for _, entry := range entries {
		name := entry.Name()
		old, ok := review.entries[name]
		if !ok {
			return SpaceTrashOutcome{}, false, errTrashReviewChanged
		}
		current, err := inspectFileTree(ctx, root, name, trashVerificationTree)
		if os.IsNotExist(err) || errors.Is(err, errTrashReviewChanged) {
			return SpaceTrashOutcome{}, false, errTrashReviewChanged
		}
		if err != nil {
			return SpaceTrashOutcome{}, false, err
		}
		if !old.matches(current) {
			return SpaceTrashOutcome{}, false, errTrashReviewChanged
		}
	}
	stage, err := newTrashStage(root)
	if err != nil {
		return SpaceTrashOutcome{}, false, err
	}
	result := SpaceTrashOutcome{
		Outcome:            FileOutcome{Succeeded: []string{}, Failed: []CleanupFailure{}},
		PostDeleteFailures: []CleanupFailure{},
	}
	for _, name := range review.names {
		previousRecovery := stage.recovery
		if err := stage.removeTree(ctx, review.entries[name]); err != nil {
			failure := CleanupFailure{Path: name, Changed: errors.Is(err, errTrashReviewChanged), Error: err.Error()}
			if previousRecovery == "" {
				failure.RecoveryPath = stage.recovery
			}
			result.Outcome.Failed = append(result.Outcome.Failed, failure)
			continue
		}
		result.Outcome.Succeeded = append(result.Outcome.Succeeded, name)
	}
	result.Deleted, result.Bytes = stage.deleted, stage.bytes
	if err := stage.finish(); err != nil {
		result.PostDeleteFailures = append(result.PostDeleteFailures, CleanupFailure{Path: filepath.Join(path, stage.path), Error: err.Error()})
	}
	if review.metadata != nil && len(result.Outcome.Succeeded) > 0 {
		result.PostDeleteFailures = review.metadata.cleanup(ctx, result.Outcome.Succeeded, result.PostDeleteFailures)
	}
	remaining, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		result.PostDeleteFailures = append(result.PostDeleteFailures, CleanupFailure{Path: path, Error: err.Error()})
	}
	return result, err == nil && len(remaining) == 0, nil
}
