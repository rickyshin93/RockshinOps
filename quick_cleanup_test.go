package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"rockshinops/internal/metrics"
)

func cleanupTestFile(t *testing.T, path string, modified time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, 8192), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
}

func TestQuickCleanupReviewAndPartialMove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	now := time.Now()
	old := now.AddDate(0, 0, -35)
	cache := filepath.Join(home, "Caches")
	downloads := filepath.Join(home, "Downloads")
	derived := filepath.Join(home, "DerivedData")
	stale, current, locked := filepath.Join(cache, "stale"), filepath.Join(cache, "current"), filepath.Join(cache, "locked")
	personal, build := filepath.Join(downloads, "installer.dmg"), filepath.Join(derived, "build.db")
	for path, date := range map[string]time.Time{stale: old, current: now, locked: old, personal: old, build: old} {
		cleanupTestFile(t, path, date)
	}
	changed := filepath.Join(cache, "changed")
	cleanupTestFile(t, filepath.Join(changed, "cache"), old)
	if err := os.Chtimes(changed, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(downloads, "small"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	s := &System{
		config: Config{CleanupIdleDays: 30, CleanupDownloadMinBytes: 4096},
		info: AppInfo{ScanTargets: []metrics.ScanTarget{
			{Name: "Caches", Path: cache, Cleanup: "idle"}, {Name: "Downloads", Path: downloads, Cleanup: "downloads"},
			{Name: "Build caches", Path: derived, Cleanup: "review"}, {Name: "Trash", Path: filepath.Join(home, ".Trash")},
		}},
	}
	ctx := context.Background()
	review, err := s.ScanCleanup(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Categories) != 3 || len(review.Categories[0].Items) != 4 || len(review.Categories[1].Items) != 1 || len(review.Categories[2].Items) != 1 {
		t.Fatalf("unexpected categories: %+v", review.Categories)
	}
	items := make(map[string]CleanupItem)
	for _, category := range review.Categories {
		for _, item := range category.Items {
			items[item.Name] = item
		}
	}
	for name, recommended := range map[string]bool{"stale": true, "current": false, "locked": true, "changed": true, "installer.dmg": false, "build.db": false} {
		item, exists := items[name]
		if !exists || item.Recommended != recommended || item.Bytes <= 0 {
			t.Fatalf("incorrect candidate %s: %+v", name, item)
		}
	}
	stale, current, locked, changed = items["stale"].Path, items["current"].Path, items["locked"].Path, items["changed"].Path
	var moved []string
	move := func(path string) error {
		if path == locked {
			return errors.New("permission denied")
		}
		moved = append(moved, path)
		return nil
	}
	if _, err := s.trashCleanup(ctx, review.ID, []string{stale, current, cache}, move); err == nil || len(moved) > 0 {
		t.Fatal("an unreviewed path was allowed to start a batch")
	}
	cleanupTestFile(t, filepath.Join(changed, "cache"), now)
	result, err := s.trashCleanup(ctx, review.ID, []string{stale, changed, locked}, move)
	if err != nil || !slices.Equal(moved, []string{stale}) || len(result.Moved) != 1 || len(result.Failed) != 2 ||
		!result.Failed[0].Changed || result.Failed[1].Changed || result.Failed[1].Error != "permission denied" {
		t.Fatalf("unexpected batch outcome: %+v, %v", result, err)
	}
	if _, err := s.trashCleanup(ctx, review.ID, []string{stale}, move); err == nil {
		t.Fatal("already moved path was accepted again")
	}
	if _, err := s.trashCleanup(ctx, review.ID, []string{changed}, move); err == nil {
		t.Fatal("changed path was accepted without rescanning")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.ScanCleanup(cancelled); err != errScanCancelled {
		t.Fatalf("cancellation returned %v", err)
	}
	if _, err := s.trashCleanup(ctx, review.ID, []string{locked}, move); err != nil {
		t.Fatalf("cancelled scan discarded the previous review: %v", err)
	}
}

func TestQuickCleanupRejectsRedirectedRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks may require Windows Developer Mode")
	}
	home := t.TempDir()
	outside := t.TempDir()
	t.Setenv("HOME", home)
	cleanupTestFile(t, filepath.Join(outside, "data"), time.Now())
	target := filepath.Join(home, "Caches")
	if err := os.Symlink(outside, target); err != nil {
		t.Fatal(err)
	}
	s := &System{info: AppInfo{ScanTargets: []metrics.ScanTarget{{Name: "Caches", Path: target, Cleanup: "idle"}}}}
	result, err := s.ScanCleanup(context.Background())
	if err != nil || len(result.Categories) != 1 || result.Categories[0].Error == "" || len(result.Categories[0].Items) != 0 {
		t.Fatalf("redirected cleanup root accepted: %+v, %v", result, err)
	}
}
