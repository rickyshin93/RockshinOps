package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"rockshinops/internal/metrics"
)

func TestEmptyTrashKeepsRootAndOutsideSymlinkTarget(t *testing.T) {
	root := t.TempDir()
	trash := filepath.Join(root, "files")
	metadata := filepath.Join(root, "info")
	for _, path := range []string{trash, metadata} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	outside := filepath.Join(root, "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(trash, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metadata, "link.trashinfo"), []byte("info"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := emptyTrash(trash, "linux"); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(trash); err != nil || len(entries) != 0 {
		t.Fatalf("trash: %v %v", entries, err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(metadata, "link.trashinfo")); !os.IsNotExist(err) {
		t.Fatalf("metadata remains: %v", err)
	}
	if err := emptyTrash(filepath.Join(trash, "link"), "linux"); err == nil {
		t.Fatal("missing directory accepted")
	}
}

func TestSpaceCleanupOnlyAllowsReviewedItemsInsideHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	trash := filepath.Join(home, ".Trash")
	caches := filepath.Join(home, "Library", "Caches")
	for _, dir := range []string{trash, filepath.Join(caches, "app"), filepath.Join(home, "Library", "Mail"), filepath.Join(home, "Downloads", "old")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(trash, "junk"), make([]byte, 2_000_000), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "Library", "Mail", "box"), make([]byte, 1_500_000), 0600); err != nil {
		t.Fatal(err)
	}
	s := &System{info: AppInfo{ScanTargets: []metrics.ScanTarget{{Name: "应用缓存", Path: caches}, {Name: trashTargetName, Path: trash}}}}
	for path, want := range map[string]bool{
		filepath.Join(home, "Downloads", "old"): true,
		filepath.Join(caches, "app"):            true,
		filepath.Join(home, "Downloads"):        false,
		filepath.Join(home, "Library", "Mail"):  false,
		caches:                                  false,
		filepath.Join(trash, "junk"):            false,
		home:                                    false,
		filepath.Dir(home):                      false,
	} {
		if got := s.trashable(path); got != want {
			t.Errorf("trashable(%s) = %v, want %v", path, got, want)
		}
	}
	summary, err := s.scanSpace(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Bytes < 3_500_000 || len(summary.Suggestions) != 2 || !summary.Suggestions[1].Trash || summary.Suggestions[1].Bytes < 2_000_000 || len(summary.Largest) != 1 || !summary.Suggestions[0].Found {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	listing, err := s.SpaceListing(filepath.Join(home, "Library"))
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Children) == 0 {
		t.Fatal("Library listing is empty")
	}
	for _, child := range listing.Children {
		if child.Trashable {
			t.Fatalf("app data root offered for removal: %+v", child)
		}
	}
	// A folder replaced after the scan must not be trashed under the old review.
	old := filepath.Join(home, "Downloads", "old")
	if err := os.Rename(old, old+"-moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(old, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TrashSpaceItem(old); err == nil {
		t.Fatal("trashed an item replaced after the scan")
	}
	if summary, err := s.EmptySpaceTrash(); err != nil || summary.Suggestions[1].Bytes != 0 || summary.Bytes >= 2_000_000 || summary.Bytes < 1_500_000 {
		t.Fatalf("empty trash: %+v %v", summary, err)
	}
	if entries, _ := os.ReadDir(trash); len(entries) != 0 {
		t.Fatalf("trash not emptied: %v", entries)
	}
}

func TestCancelledSpaceScanKeepsPreviousResult(t *testing.T) {
	home := t.TempDir()
	s := &System{}
	if _, err := s.scanSpace(context.Background(), home); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.scanSpace(ctx, home); err != errScanCancelled {
		t.Fatalf("cancelled scan returned %v", err)
	}
	if _, err := s.SpaceListing(""); err != nil {
		t.Fatalf("previous result was discarded: %v", err)
	}
	if s.SpaceProgress().Scanning {
		t.Fatal("progress still reports a running scan")
	}
}

func TestCancelScanStopsRunningScan(t *testing.T) {
	s := &System{}
	ctx, done := s.beginScan(context.Background())
	defer done()
	s.CancelScan()
	if ctx.Err() == nil {
		t.Fatal("scan context was not cancelled")
	}
}

func TestScanProgressShowsShallowFolder(t *testing.T) {
	root := filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	deep := filepath.Join(root, "Users", "me", "work", "app", "node_modules", "pkg")
	if got, want := shallowPath(deep, 4), filepath.Join(root, "Users", "me", "work", "app"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := shallowPath(filepath.Join(root, "Users"), 4); got != filepath.Join(root, "Users") {
		t.Fatalf("short path changed: %q", got)
	}
}
