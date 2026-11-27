package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

var errTrashMetadataChanged = errors.New("trash_metadata_changed")
var errTrashMetadataInvalid = errors.New("trash_metadata_invalid")

type trashMetadataEntry struct {
	file reviewedFile
	err  error
}

type trashMetadataReview struct {
	path    string
	root    fs.FileInfo
	entries map[string]trashMetadataEntry
	err     error
}

func reviewTrashMetadata(ctx context.Context, path string, names []string, reader *trashContentReader) *trashMetadataReview {
	metadata := &trashMetadataReview{path: path}
	root, info, err := openTrashRoot(path)
	if err != nil {
		if !os.IsNotExist(err) {
			metadata.err = err
		}
		return metadata
	}
	defer root.Close()
	metadata.root = info
	metadata.entries = make(map[string]trashMetadataEntry, len(names))
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			metadata.err = err
			return metadata
		}
		path := name + ".trashinfo"
		info, err := root.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		entry := trashMetadataEntry{file: reviewedFile{path: path, info: info}, err: err}
		if err == nil {
			if info.Mode().IsRegular() {
				entry.file.content, entry.err = reader.hash(ctx, root, path, info)
			} else {
				entry.err = errTrashMetadataInvalid
			}
		}
		metadata.entries[name] = entry
	}
	return metadata
}

func (metadata *trashMetadataReview) cleanup(ctx context.Context, names []string, failures []CleanupFailure) []CleanupFailure {
	if metadata.err != nil {
		return append(failures, CleanupFailure{Path: metadata.path, Error: metadata.err.Error()})
	}
	root, info, err := openTrashRoot(metadata.path)
	if os.IsNotExist(err) && metadata.root == nil {
		return failures
	}
	if os.IsNotExist(err) || errors.Is(err, errTrashReviewChanged) {
		err = errTrashMetadataChanged
	}
	if err != nil {
		return append(failures, CleanupFailure{Path: metadata.path, Changed: errors.Is(err, errTrashMetadataChanged), Error: err.Error()})
	}
	defer root.Close()
	if metadata.root == nil || !os.SameFile(metadata.root, info) {
		return append(failures, CleanupFailure{Path: metadata.path, Changed: true, Error: errTrashMetadataChanged.Error()})
	}
	var stage *trashStage
	for _, name := range names {
		path := name + ".trashinfo"
		entry, reviewed := metadata.entries[name]
		current, err := root.Lstat(path)
		if os.IsNotExist(err) && !reviewed {
			continue
		}
		if os.IsNotExist(err) || err == nil && !reviewed {
			err = errTrashMetadataChanged
		}
		if err == nil {
			err = entry.err
		}
		if err == nil && !sameFileState(entry.file.info, current) {
			err = errTrashMetadataChanged
		}
		recovery := ""
		if err == nil {
			if stage == nil {
				stage, err = newTrashStage(root)
			}
			if err == nil {
				previousRecovery := stage.recovery
				err = stage.remove(ctx, entry.file)
				if previousRecovery == "" {
					recovery = stage.recovery
				}
			}
		}
		if errors.Is(err, errTrashReviewChanged) {
			err = errTrashMetadataChanged
		}
		if err != nil {
			failures = append(failures, CleanupFailure{Path: filepath.Join(metadata.path, path), RecoveryPath: recovery, Changed: errors.Is(err, errTrashMetadataChanged), Error: err.Error()})
		}
	}
	if stage != nil {
		if err := stage.finish(); err != nil {
			failures = append(failures, CleanupFailure{Path: filepath.Join(metadata.path, stage.path), Error: err.Error()})
		}
	}
	return failures
}
