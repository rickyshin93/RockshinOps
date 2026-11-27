package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"rockshinops/internal/metrics"
	"rockshinops/internal/storage"
)

const trashTargetName = "废纸篓"

// Direct children of these home folders hold app data that must not be trashed wholesale.
var protectedHomeFolders = map[string]bool{"Library": true, "AppData": true, ".local": true, ".config": true}

var errScanCancelled = errors.New("scan cancelled")

type SpaceSuggestion struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Files int64  `json:"files"`
	Found bool   `json:"found"`
	Trash bool   `json:"trash"`
}

type SpaceSummary struct {
	ID          uint64               `json:"id"`
	Root        string               `json:"root"`
	Bytes       int64                `json:"bytes"`
	Files       int64                `json:"files"`
	Skipped     int64                `json:"skipped"`
	StartedAt   time.Time            `json:"startedAt"`
	FinishedAt  time.Time            `json:"finishedAt"`
	Top         []metrics.SpaceEntry `json:"top"`
	Largest     []metrics.SpaceEntry `json:"largest"`
	Suggestions []SpaceSuggestion    `json:"suggestions"`
}

type SpaceProgress struct {
	Scanning bool   `json:"scanning"`
	Files    int64  `json:"files"`
	Bytes    int64  `json:"bytes"`
	Skipped  int64  `json:"skipped"`
	Path     string `json:"path"`
}

// beginScan registers the running scan so CancelScan can stop it.
func (s *System) beginScan(ctx context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	s.files.space.cancelMu.Lock()
	s.files.space.cancel = cancel
	s.files.space.cancelMu.Unlock()
	return ctx, func() {
		cancel()
		s.files.space.cancelMu.Lock()
		s.files.space.cancel = nil
		s.files.space.cancelMu.Unlock()
	}
}

func (s *System) CancelScan() {
	s.files.space.cancelMu.Lock()
	defer s.files.space.cancelMu.Unlock()
	if s.files.space.cancel != nil {
		s.files.space.cancel()
	}
}

// ScanSpace measures the volume holding the home directory once; browsing and
// cleanup then use the in-memory result until the next scan.
func (s *System) ScanSpace(ctx context.Context) (SpaceSummary, error) {
	volume, err := storage.Read()
	if err != nil {
		return SpaceSummary{}, err
	}
	return s.scanSpace(ctx, volume.Mount)
}

type spaceScanScope struct {
	root, home, originalHome, resolvedHome string
	targets                                []metrics.ScanTarget
}

// rebaseSpacePath preserves the suffix of a path within a root, including the root itself.
func rebaseSpacePath(path, from, to string) (string, bool) {
	rel, err := filepath.Rel(from, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.Join(to, rel), true
}

func newSpaceScanScope(home, mount string, targets []metrics.ScanTarget) (spaceScanScope, error) {
	if !filepath.IsAbs(mount) {
		return spaceScanScope{}, fmt.Errorf("invalid home volume mount: %q", mount)
	}
	root := filepath.Clean(mount)
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return spaceScanScope{}, err
	}
	resolvedHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		return spaceScanScope{}, err
	}
	scannedHome, within := rebaseSpacePath(resolvedHome, resolvedRoot, root)
	if !within && runtime.GOOS == "darwin" && root == "/System/Volumes/Data" {
		// The Data volume exposes home through a firmlink such as /Users.
		candidate := filepath.Join(root, resolvedHome)
		source, sourceErr := os.Stat(resolvedHome)
		destination, destinationErr := os.Stat(candidate)
		if sourceErr == nil && destinationErr == nil && os.SameFile(source, destination) {
			scannedHome, within = candidate, true
		}
	}
	if !within {
		return spaceScanScope{}, fmt.Errorf("home directory %q is outside the reported volume %q", home, root)
	}
	scope := spaceScanScope{root: root, home: scannedHome, originalHome: home, resolvedHome: resolvedHome}
	for _, target := range targets {
		target.Path = scope.treePath(target.Path)
		scope.targets = append(scope.targets, target)
	}
	return scope, nil
}

func (s spaceScanScope) treePath(path string) string {
	for _, home := range []string{s.originalHome, s.resolvedHome} {
		if mapped, ok := rebaseSpacePath(path, home, s.home); ok {
			return mapped
		}
	}
	return path
}

func (s *System) scanSpace(ctx context.Context, mount string) (SpaceSummary, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return SpaceSummary{}, err
	}
	scope, err := newSpaceScanScope(home, mount, s.info.ScanTargets)
	if err != nil {
		return SpaceSummary{}, err
	}
	progress := &metrics.SpaceProgress{}
	if !s.files.space.progress.CompareAndSwap(nil, progress) {
		return SpaceSummary{}, fmt.Errorf("a scan is already running")
	}
	defer s.files.space.progress.Store(nil)
	s.files.space.mu.Lock()
	revision := s.files.space.revision
	s.files.space.mu.Unlock()
	ctx, done := s.beginScan(ctx)
	defer done()
	started := time.Now()
	keep := make([]string, 0, len(scope.targets))
	for _, target := range scope.targets {
		keep = append(keep, target.Path)
	}
	tree, err := metrics.ScanSpaceTree(ctx, scope.root, nil, progress, keep...)
	if ctx.Err() != nil {
		return SpaceSummary{}, errScanCancelled
	}
	if err != nil {
		return SpaceSummary{}, err
	}
	s.files.space.mu.Lock()
	defer s.files.space.mu.Unlock()
	if revision != s.files.space.revision {
		return SpaceSummary{}, fmt.Errorf("cleanup changed the disk during the scan; scan again")
	}
	s.files.space.scope = scope
	s.files.space.id = s.files.reviews.Add(1)
	s.files.space.trash = nil
	s.files.space.tree, s.files.space.skipped, s.files.space.started, s.files.space.finished = tree, progress.Skipped.Load(), started, time.Now()
	return s.spaceSummary(), nil
}

func (s *System) SpaceProgress() SpaceProgress {
	progress := s.files.space.progress.Load()
	if progress == nil {
		return SpaceProgress{}
	}
	result := SpaceProgress{Scanning: true, Files: progress.Files.Load(), Bytes: progress.Bytes.Load(), Skipped: progress.Skipped.Load()}
	if path := progress.Path.Load(); path != nil {
		result.Path = shallowPath(*path, 4)
	}
	return result
}

// shallowPath keeps depth folders below the volume root. Parallel walkers each
// report their own deep folder, so the full path would jump on every poll.
func shallowPath(path string, depth int) string {
	root := filepath.VolumeName(path) + string(filepath.Separator)
	parts, ok := relativeParts(root, path)
	if !ok || len(parts) <= depth {
		return path
	}
	return filepath.Join(append([]string{root}, parts[:depth]...)...)
}

// SpaceListing returns a scanned folder; an empty path means the volume root.
func (s *System) SpaceListing(path string) (metrics.SpaceListing, error) {
	s.files.space.mu.Lock()
	defer s.files.space.mu.Unlock()
	if s.files.space.tree == nil {
		return metrics.SpaceListing{}, fmt.Errorf("no scan result; scan again")
	}
	if path == "" {
		path = s.files.space.tree.Root()
	}
	listing, err := s.files.space.tree.Listing(path, 200)
	if err != nil {
		return listing, fmt.Errorf("folder is not in the scan result; scan again")
	}
	for i, child := range listing.Children {
		listing.Children[i].Trashable = !child.Other && s.files.space.scope.trashable(child.Path)
	}
	return listing, nil
}

func (s *System) TrashSpaceItem(id uint64, path string) (SpaceSummary, error) {
	s.files.writes.Lock()
	defer s.files.writes.Unlock()
	s.files.space.mu.Lock()
	defer s.files.space.mu.Unlock()
	if s.files.space.tree == nil || id != s.files.space.id || !s.files.space.scope.trashable(path) {
		return SpaceSummary{}, fmt.Errorf("this item cannot be moved to the trash")
	}
	if err := s.matchesScan(path); err != nil {
		return SpaceSummary{}, err
	}
	var destination string
	var err error
	if runtime.GOOS == "darwin" {
		destination, err = mygo.Shell.TrashItemResult(path)
		destination = s.files.space.scope.treePath(destination)
	} else {
		err = mygo.Shell.TrashItem(path)
		if trash := s.trashPath(); trash != "" {
			destination = filepath.Join(trash, filepath.Base(path))
		}
	}
	if err != nil {
		return SpaceSummary{}, err
	}
	s.files.space.tree.Move(path, destination)
	s.files.space.revision++
	s.files.space.trash = nil
	return s.spaceSummary(), nil
}

func (s *System) PrepareSpaceTrash(ctx context.Context, id uint64) (TrashReview, error) {
	space := &s.files.space
	space.mu.Lock()
	if space.tree == nil || id != space.id {
		space.mu.Unlock()
		return TrashReview{}, fmt.Errorf("no matching space scan; scan again")
	}
	trash := s.trashPath()
	if err := s.matchesScan(trash); err != nil {
		space.mu.Unlock()
		return TrashReview{}, err
	}
	revision := space.revision
	space.mu.Unlock()

	review, err := reviewTrash(ctx, trash, runtime.GOOS)
	if err != nil {
		return TrashReview{}, err
	}
	space.mu.Lock()
	defer space.mu.Unlock()
	if id != space.id || revision != space.revision {
		return TrashReview{}, errTrashReviewChanged
	}
	review.summary.ID = s.files.reviews.Add(1)
	space.trash = review
	return review.summary, nil
}

func (s *System) DiscardTrashReview(reviewID uint64) {
	space := &s.files.space
	space.mu.Lock()
	defer space.mu.Unlock()
	if space.trash != nil && space.trash.summary.ID == reviewID {
		space.trash = nil
	}
}

func (s *System) TrashReviewItems(reviewID uint64, offset int) ([]TrashReviewItem, error) {
	space := &s.files.space
	space.mu.Lock()
	defer space.mu.Unlock()
	review := space.trash
	if review == nil || review.summary.ID != reviewID {
		return nil, errTrashReviewChanged
	}
	if offset < 0 || offset > len(review.names) {
		return nil, fmt.Errorf("invalid trash review offset")
	}
	return review.items(offset), nil
}

func (s *System) EmptySpaceTrash(ctx context.Context, reviewID uint64) (SpaceTrashOutcome, error) {
	s.files.writes.Lock()
	defer s.files.writes.Unlock()
	space := &s.files.space
	space.mu.Lock()
	defer space.mu.Unlock()
	review := space.trash
	if review == nil || review.summary.ID != reviewID {
		return SpaceTrashOutcome{}, errTrashReviewChanged
	}
	space.trash = nil
	result, empty, err := review.empty(ctx)
	if err != nil {
		return SpaceTrashOutcome{}, err
	}
	space.revision++
	if empty {
		space.tree.Clear(review.summary.Path)
		summary := s.spaceSummary()
		result.Summary = &summary
	} else {
		space.tree = nil
	}
	return result, nil
}

// matchesScan rejects paths replaced since the scan, where the platform reports identities.
func (s *System) matchesScan(path string) error {
	scanned, ok := s.files.space.tree.Find(path)
	if !ok {
		return fmt.Errorf("item is not in the scan result; scan again")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	old, _ := s.files.space.tree.Identity(path)
	current, identified := metrics.FileIdentity(info)
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() != scanned.Directory || identified && current != old {
		return fmt.Errorf("item changed since the scan; scan again")
	}
	return nil
}

// trashable allows items at least two levels inside the home directory, except
// configured cleanup locations themselves, trash contents and app-data roots.
func (s spaceScanScope) trashable(path string) bool {
	path = filepath.Clean(path)
	parts, ok := relativeParts(s.home, path)
	if !ok || len(parts) < 2 || len(parts) == 2 && protectedHomeFolders[parts[0]] {
		return false
	}
	for _, target := range s.targets {
		if path == target.Path {
			return false
		}
		if _, inside := relativeParts(target.Path, path); target.Name == trashTargetName && (path == target.Path || inside) {
			return false
		}
	}
	return true
}

func relativeParts(base, path string) ([]string, bool) {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, false
	}
	return strings.Split(rel, string(filepath.Separator)), true
}

func (s *System) trashPath() string {
	for _, target := range s.files.space.scope.targets {
		if target.Name == trashTargetName {
			if _, ok := s.files.space.tree.Find(target.Path); ok {
				return target.Path
			}
		}
	}
	return ""
}

func (s *System) spaceSummary() SpaceSummary {
	largest := s.files.space.tree.Largest(50, s.trashPath())
	for i, file := range largest {
		largest[i].Trashable = s.files.space.scope.trashable(file.Path)
	}
	suggestions := make([]SpaceSuggestion, 0, len(s.files.space.scope.targets))
	for _, target := range s.files.space.scope.targets {
		entry, found := s.files.space.tree.Find(target.Path)
		suggestions = append(suggestions, SpaceSuggestion{Name: target.Name, Path: target.Path, Bytes: entry.Bytes, Files: entry.Files, Found: found, Trash: target.Name == trashTargetName})
	}
	top, _ := s.files.space.tree.Listing(s.files.space.tree.Root(), 5)
	return SpaceSummary{ID: s.files.space.id, Root: s.files.space.tree.Root(), Bytes: s.files.space.tree.Bytes(), Files: s.files.space.tree.Files(), Skipped: s.files.space.skipped,
		StartedAt: s.files.space.started, FinishedAt: s.files.space.finished, Top: top.Children, Largest: largest, Suggestions: suggestions}
}
