package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"rockshinops/internal/metrics"
)

type CleanupItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Bytes       int64  `json:"bytes"`
	Recommended bool   `json:"recommended"`
	Directory   bool   `json:"directory"`
}

type CleanupCategory struct {
	Name    string        `json:"name"`
	Mode    string        `json:"mode"`
	Path    string        `json:"path"`
	Items   []CleanupItem `json:"items"`
	Skipped int           `json:"skipped"`
	Error   string        `json:"error"`
}

type CleanupScan struct {
	ID         uint64            `json:"id"`
	IdleDays   int               `json:"idleDays"`
	Categories []CleanupCategory `json:"categories"`
}

type CleanupProgress struct {
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	Category  string `json:"category"`
}

type CleanupFailure struct {
	Path         string `json:"path"`
	Changed      bool   `json:"changed"`
	Error        string `json:"error"`
	RecoveryPath string `json:"recoveryPath,omitempty"`
}

type CleanupOutcome struct {
	Moved  []CleanupItem    `json:"moved"`
	Failed []CleanupFailure `json:"failed"`
}

type cleanupCandidate struct {
	item  CleanupItem
	base  string
	stamp uint64
	info  fs.FileInfo
}

type quickCleanup struct {
	mu         sync.Mutex
	cancel     context.CancelFunc
	progress   CleanupProgress
	id         uint64
	candidates map[string]cleanupCandidate
}

func (s *System) CleanupProgress() CleanupProgress {
	s.files.quick.mu.Lock()
	defer s.files.quick.mu.Unlock()
	return s.files.quick.progress
}

func (s *System) CancelCleanupScan() {
	s.files.quick.mu.Lock()
	defer s.files.quick.mu.Unlock()
	if s.files.quick.cancel != nil {
		s.files.quick.cancel()
	}
}

func (s *System) ScanCleanup(ctx context.Context) (CleanupScan, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return CleanupScan{}, err
	}
	home, err := filepath.EvalSymlinks(userHome)
	if err != nil {
		return CleanupScan{}, err
	}
	var targets []metrics.ScanTarget
	for _, target := range s.info.ScanTargets {
		if target.Cleanup != "" {
			targets = append(targets, target)
		}
	}
	scan := &s.files.quick
	scan.mu.Lock()
	if scan.cancel != nil {
		scan.mu.Unlock()
		return CleanupScan{}, fmt.Errorf("a cleanup scan is already running")
	}
	ctx, cancel := context.WithCancel(ctx)
	scan.cancel = cancel
	scan.progress.Total = len(targets)
	scan.mu.Unlock()
	defer func() {
		scan.mu.Lock()
		defer scan.mu.Unlock()
		cancel()
		scan.cancel = nil
		scan.progress = CleanupProgress{}
	}()

	categories := make([]CleanupCategory, 0, len(targets))
	candidates := make(map[string]cleanupCandidate)
	idleBefore := time.Now().AddDate(0, 0, -s.config.CleanupIdleDays)
	for i, target := range targets {
		scan.mu.Lock()
		scan.progress.Category, scan.progress.Completed = target.Name, i
		scan.mu.Unlock()
		categories = append(categories, scanCleanupCategory(ctx, userHome, home, target, idleBefore, s.config.CleanupDownloadMinBytes, candidates))
		if ctx.Err() != nil {
			return CleanupScan{}, errScanCancelled
		}
	}
	scan.mu.Lock()
	defer scan.mu.Unlock()
	if ctx.Err() != nil {
		return CleanupScan{}, errScanCancelled
	}
	scan.id = s.files.reviews.Add(1)
	scan.candidates = candidates
	return CleanupScan{ID: scan.id, IdleDays: s.config.CleanupIdleDays, Categories: categories}, nil
}

func openCleanupRoot(path string) (*os.Root, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	if rel, err := filepath.Rel(path, resolved); err != nil || rel != "." {
		return nil, fmt.Errorf("cleanup folder was redirected")
	}
	return os.OpenRoot(path)
}

func scanCleanupCategory(ctx context.Context, userHome, home string, target metrics.ScanTarget, idleBefore time.Time, downloadMin int64, candidates map[string]cleanupCandidate) CleanupCategory {
	category := CleanupCategory{Name: target.Name, Mode: target.Cleanup, Path: target.Path, Items: []CleanupItem{}}
	base, within := rebaseSpacePath(target.Path, userHome, home)
	if !within || base == home {
		category.Error = "cleanup folder is outside the home directory"
		return category
	}
	root, err := openCleanupRoot(base)
	if errors.Is(err, fs.ErrNotExist) {
		return category
	}
	if err != nil {
		category.Error = err.Error()
		return category
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		category.Error = err.Error()
		return category
	}
	for _, entry := range entries {
		if ctx.Err() != nil {
			break
		}
		if target.Cleanup == "downloads" && !entry.Type().IsRegular() {
			continue
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			continue
		}
		candidate, newest, err := inspectCleanupItem(ctx, root, entry.Name())
		if err != nil {
			category.Skipped++
			continue
		}
		item := &candidate.item
		if item.Bytes == 0 || target.Cleanup == "downloads" && item.Bytes < downloadMin {
			continue
		}
		item.Recommended = target.Cleanup == "idle" && newest.Before(idleBefore)
		candidates[item.Path] = candidate
		category.Items = append(category.Items, *item)
	}
	slices.SortFunc(category.Items, func(a, b CleanupItem) int {
		if a.Bytes != b.Bytes {
			if a.Bytes > b.Bytes {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Path, b.Path)
	})
	return category
}

func inspectCleanupItem(ctx context.Context, root *os.Root, name string) (cleanupCandidate, time.Time, error) {
	tree, err := inspectFileTree(ctx, root, name, cleanupCandidateTree)
	if err != nil {
		return cleanupCandidate{}, time.Time{}, err
	}
	if !tree.info.IsDir() && !tree.info.Mode().IsRegular() {
		return cleanupCandidate{}, time.Time{}, fmt.Errorf("item cannot be moved")
	}
	return cleanupCandidate{
		item: CleanupItem{Name: name, Path: filepath.Join(root.Name(), name), Directory: tree.info.IsDir(), Bytes: tree.bytes},
		base: root.Name(), info: tree.info, stamp: tree.stamp,
	}, tree.newest, nil
}

func (s *System) RevealCleanupItem(id uint64, path string) error {
	scan := &s.files.quick
	scan.mu.Lock()
	defer scan.mu.Unlock()
	candidate, exists := scan.candidates[path]
	if id != scan.id || !exists {
		return fmt.Errorf("item is not in the cleanup review")
	}
	root, err := openCleanupRoot(candidate.base)
	if err != nil {
		return err
	}
	defer root.Close()
	current, err := root.Lstat(candidate.item.Name)
	if err != nil {
		return err
	}
	if !os.SameFile(candidate.info, current) {
		return fmt.Errorf("item changed since the scan")
	}
	mygo.Shell.ShowItemInFolder(path)
	return nil
}

func (s *System) TrashCleanup(ctx context.Context, id uint64, paths []string) (CleanupOutcome, error) {
	return s.trashCleanup(ctx, id, paths, mygo.Shell.TrashItem)
}

func (s *System) trashCleanup(ctx context.Context, id uint64, paths []string, move func(string) error) (CleanupOutcome, error) {
	s.files.writes.Lock()
	defer s.files.writes.Unlock()
	scan := &s.files.quick
	scan.mu.Lock()
	defer scan.mu.Unlock()
	if scan.cancel != nil || id != scan.id {
		return CleanupOutcome{}, fmt.Errorf("no matching cleanup review; scan again")
	}
	if err := checkSelection(scan.candidates, paths); err != nil {
		return CleanupOutcome{}, err
	}

	result := applyReviewed(ctx, paths, func(path string) error {
		return scan.candidates[path].revalidate(ctx)
	}, move)
	outcome := CleanupOutcome{Moved: []CleanupItem{}, Failed: result.Failed}
	for _, path := range result.Succeeded {
		outcome.Moved = append(outcome.Moved, scan.candidates[path].item)
		delete(scan.candidates, path)
	}
	for _, failure := range result.Failed {
		if failure.Changed {
			delete(scan.candidates, failure.Path)
		}
	}
	if len(outcome.Moved) > 0 {
		s.files.invalidateSpace()
	}
	return outcome, nil
}

func (candidate cleanupCandidate) revalidate(ctx context.Context) error {
	root, err := openCleanupRoot(candidate.base)
	if err != nil {
		return err
	}
	defer root.Close()
	current, _, err := inspectCleanupItem(ctx, root, candidate.item.Name)
	if err != nil {
		return err
	}
	final, err := os.Lstat(candidate.item.Path)
	if err != nil {
		return err
	}
	if candidate.stamp != current.stamp || !os.SameFile(candidate.info, current.info) || !os.SameFile(current.info, final) {
		return fmt.Errorf("item changed since the scan")
	}
	return nil
}
