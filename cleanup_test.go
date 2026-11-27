package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"rockshinops/internal/metrics"
	"rockshinops/internal/storage"
)

func TestSpaceSelectsNativeHomeVolume(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native APFS volume selection")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	volume, err := storage.Read()
	if err != nil {
		t.Fatal(err)
	}
	trash := filepath.Join(home, ".Trash")
	scope, err := newSpaceScanScope(home, volume.Mount, []metrics.ScanTarget{{Name: trashTargetName, Path: trash}})
	if err != nil {
		t.Fatal(err)
	}
	if scope.root != filepath.Clean(volume.Mount) || scope.targets[0].Path != filepath.Join(scope.home, ".Trash") {
		t.Fatalf("scope failed to follow the mount or rebase the Trash target: %+v", scope)
	}
	original, err := os.Stat(home)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := os.Stat(scope.home)
	if err != nil || !os.SameFile(original, selected) {
		t.Fatalf("home alias does not identify the same directory: %v", err)
	}
	t.Logf("home and configured Trash resolve within native %s mount %s", volume.Format, scope.root)
}

func TestSpaceSuggestionsDescribeOnlyScannedVolume(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	outside := t.TempDir()
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	s := &System{info: AppInfo{ScanTargets: []metrics.ScanTarget{{Name: "Outside", Path: outside}}}}
	summary, err := s.scanSpace(context.Background(), root)
	if err != nil || len(summary.Suggestions) != 1 || summary.Suggestions[0].Found {
		t.Fatalf("an unscanned target cannot report a scanned size: %+v, %v", summary, err)
	}
}

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
	review, err := reviewTrash(context.Background(), trash, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if outcome, empty, err := review.empty(context.Background()); err != nil || !empty || len(outcome.Outcome.Succeeded) != 1 {
		t.Fatalf("empty reviewed trash: %+v, %t, %v", outcome, empty, err)
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
	if _, err := reviewTrash(context.Background(), filepath.Join(trash, "link"), runtime.GOOS); err == nil {
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
		trash:                                   false,
		filepath.Join(trash, "junk"):            false,
		home:                                    false,
		filepath.Dir(home):                      false,
	} {
		if got := (spaceScanScope{home: home, targets: s.info.ScanTargets}).trashable(path); got != want {
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
	if _, err := s.TrashSpaceItem(summary.ID, old); err == nil {
		t.Fatal("trashed an item replaced after the scan")
	}
	review, err := s.PrepareSpaceTrash(context.Background(), summary.ID)
	if err != nil || review.Count != 1 {
		t.Fatalf("review trash: %+v %v", review, err)
	}
	newItem := filepath.Join(trash, "newly-added")
	if err := os.WriteFile(newItem, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EmptySpaceTrash(context.Background(), review.ID); err == nil {
		t.Fatal("stale review accepted an additional file")
	}
	if entries, _ := os.ReadDir(trash); len(entries) != 2 {
		t.Fatalf("stale review changed trash: %v", entries)
	}
	review, err = s.PrepareSpaceTrash(context.Background(), summary.ID)
	if err != nil || review.Count != 2 {
		t.Fatalf("review updated trash: %+v %v", review, err)
	}
	if result, err := s.EmptySpaceTrash(context.Background(), review.ID); err != nil || result.Summary == nil || result.Summary.Suggestions[1].Bytes != 0 || result.Summary.Bytes >= 2_000_000 || result.Summary.Bytes < 1_500_000 {
		t.Fatalf("empty trash: %+v %v", result, err)
	}
	if entries, _ := os.ReadDir(trash); len(entries) != 0 {
		t.Fatalf("trash not emptied: %v", entries)
	}
}

func TestCancelledSpaceScanKeepsPreviousResult(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
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

func TestPermanentRemovalKeepsNewNestedEntries(t *testing.T) {
	path := t.TempDir()
	folder, older := filepath.Join(path, "folder"), filepath.Join(path, "folder", "existing")
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(older, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	review, err := reviewTrash(context.Background(), path, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	newItem := filepath.Join(folder, "arrived-during-removal")
	if err := os.WriteFile(newItem, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	// Invoke the reviewed removal after simulating an arrival after its final
	// snapshot comparison. A recursive delete would erase the new file too.
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	stage, err := newTrashStage(root)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.finish()
	if err := stage.removeTree(context.Background(), review.entries["folder"]); !errors.Is(err, errTrashReviewChanged) {
		t.Fatalf("expected the new child to block final removal: %v", err)
	}
	if stage.deleted != 1 || stage.bytes == 0 {
		t.Fatalf("missing partial deletion receipt: %d items, %d bytes", stage.deleted, stage.bytes)
	}
	if data, err := os.ReadFile(newItem); err != nil || string(data) != "keep" {
		t.Fatalf("unreviewed item lost: %q %v", data, err)
	}
}

func TestPermanentRemovalPreservesEntryReplacedBeforeClaim(t *testing.T) {
	path := t.TempDir()
	const name = "item"
	item := filepath.Join(path, name)
	if err := os.WriteFile(item, []byte("reviewed"), 0600); err != nil {
		t.Fatal(err)
	}
	review, err := reviewTrash(context.Background(), path, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	stage, err := newTrashStage(root)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.finish()
	modified := review.entries[name].info.ModTime()
	stage.move = func(from, to string) error {
		if err := root.Rename(from, "original"); err != nil {
			return err
		}
		if err := root.WriteFile(from, []byte("supplant"), 0600); err != nil {
			return err
		}
		if err := root.Chtimes(from, modified, modified); err != nil {
			return err
		}
		return root.Rename(from, to)
	}
	if err := stage.removeTree(context.Background(), review.entries[name]); !errors.Is(err, errTrashReviewChanged) {
		t.Fatalf("replacement claim should fail: %v", err)
	}
	replacement, err := os.ReadFile(stage.recovery)
	if stage.deleted != 0 || err != nil || string(replacement) != "supplant" {
		t.Fatalf("replacement was not preserved: deleted=%d, path=%s, content=%q, err=%v", stage.deleted, stage.recovery, replacement, err)
	}
	original, err := root.ReadFile("original")
	if err != nil || string(original) != "reviewed" {
		t.Fatalf("original data was not preserved: %q %v", original, err)
	}
}

func TestTrashReviewDetectsContentChangeWithPreservedMtime(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("supported trash platforms report inode change time")
	}
	path := t.TempDir()
	item := filepath.Join(path, "item")
	modified := time.Unix(1700000000, 0)
	write := func(value string) {
		t.Helper()
		if err := os.WriteFile(item, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(item, modified, modified); err != nil {
			t.Fatal(err)
		}
	}
	write("AAAA")
	review, err := reviewTrash(context.Background(), path, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	write("BBBB")
	if result, _, err := review.empty(context.Background()); !errors.Is(err, errTrashReviewChanged) || result.Deleted != 0 {
		t.Fatalf("content change should reject old review: %+v %v", result, err)
	}
	if data, err := os.ReadFile(item); err != nil || string(data) != "BBBB" {
		t.Fatalf("updated file should remain: %q %v", data, err)
	}
}

func TestTrashReviewAllowsFIFOAndReportsMetadataFailureSeparately(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("uses a POSIX named pipe")
	}
	parent := t.TempDir()
	trash := filepath.Join(parent, "files")
	info := filepath.Join(parent, "info", "regular.trashinfo")
	for _, dir := range []string{trash, info} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(info, "existing"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(trash, "regular"), []byte("reviewed"), 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("mkfifo", filepath.Join(trash, "pipe")).CombinedOutput(); err != nil {
		t.Fatalf("prepare FIFO: %v %s", err, output)
	}
	review, err := reviewTrash(context.Background(), trash, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if len(review.summary.Items) != 2 || review.summary.Items[0].Name != "pipe" {
		t.Fatalf("FIFO missing in manifest: %+v", review.summary)
	}
	result, empty, err := review.empty(context.Background())
	if err != nil || !empty || len(result.Outcome.Succeeded) != 2 || len(result.Outcome.Failed) != 0 || result.Deleted != 2 ||
		result.Bytes == 0 || len(result.PostDeleteFailures) != 1 || result.PostDeleteFailures[0].Path != info {
		t.Fatalf("deletion and metadata receipt: %+v, empty=%t, error=%v", result, empty, err)
	}
}

func TestTrashReviewPagesStayWithReviewVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	trash := filepath.Join(home, ".Trash")
	if err := os.Mkdir(trash, 0700); err != nil {
		t.Fatal(err)
	}
	for i := range trashReviewPageSize + 1 {
		if err := os.WriteFile(filepath.Join(trash, fmt.Sprintf("entry-%03d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	s := &System{info: AppInfo{ScanTargets: []metrics.ScanTarget{{Name: trashTargetName, Path: trash}}}}
	summary, err := s.scanSpace(ctx, home)
	if err != nil {
		t.Fatal(err)
	}
	review, err := s.PrepareSpaceTrash(ctx, summary.ID)
	if err != nil || review.Count != trashReviewPageSize+1 || len(review.Items) != trashReviewPageSize || review.Items[0].Name != "entry-000" {
		t.Fatalf("first page: %+v %v", review, err)
	}
	later, err := s.TrashReviewItems(review.ID, len(review.Items))
	if err != nil || len(later) != 1 || later[0].Name != fmt.Sprintf("entry-%03d", trashReviewPageSize) {
		t.Fatalf("next page: %+v %v", later, err)
	}
	s.DiscardTrashReview(review.ID)
	if _, err := s.TrashReviewItems(review.ID, 0); !errors.Is(err, errTrashReviewChanged) {
		t.Fatalf("discarded review page should expire: %v", err)
	}
	if _, err := s.EmptySpaceTrash(ctx, review.ID); !errors.Is(err, errTrashReviewChanged) {
		t.Fatalf("discarded review must not be executable: %v", err)
	}
	current, err := s.PrepareSpaceTrash(ctx, summary.ID)
	if err != nil {
		t.Fatal(err)
	}
	s.DiscardTrashReview(review.ID)
	if items, err := s.TrashReviewItems(current.ID, 0); err != nil || len(items) != trashReviewPageSize {
		t.Fatalf("discarding the older token must preserve the current review: %v", err)
	}
}

func TestPermanentRemovalHandlesMultipleReviewedHardLinks(t *testing.T) {
	trash := t.TempDir()
	file := filepath.Join(trash, "first")
	if err := os.WriteFile(file, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(file, filepath.Join(trash, "last")); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	review, err := reviewTrash(ctx, trash, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	if review.summary.Items[0].Bytes == 0 || review.summary.Items[0].Bytes != review.summary.Items[1].Bytes {
		t.Fatalf("hard-link sizes should reflect each entry: %+v", review.summary.Items)
	}
	if runtime.GOOS != "windows" && review.summary.Bytes != review.summary.Items[0].Bytes {
		t.Fatalf("review should estimate one shared allocation: %+v", review.summary)
	}
	result, empty, err := review.empty(ctx)
	if err != nil || !empty || result.Deleted != 2 || len(result.Outcome.Failed) != 0 || result.Bytes != review.summary.Bytes {
		t.Fatalf("reviewed hard links: %+v, empty=%t, error=%v", result, empty, err)
	}
}

func TestTrashStageFinishPreservesReplacedStagePath(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	stage, err := newTrashStage(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Rename(stage.path, "actual-stage"); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile(stage.path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := stage.finish(); !errors.Is(err, errTrashReviewChanged) {
		t.Fatalf("unexpected stage should be preserved: %v", err)
	}
	if got, err := root.ReadFile(stage.path); err != nil || string(got) != "keep" {
		t.Fatalf("replacement changed: %q %v", got, err)
	}
	if info, err := root.Stat("actual-stage"); err != nil || !info.IsDir() {
		t.Fatalf("original stage should still exist: %v", err)
	}
}

func TestPermanentRemovalPreservesContentChangedAtClaim(t *testing.T) {
	path := t.TempDir()
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.WriteFile("entry", []byte("old!"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	review, err := reviewTrash(ctx, path, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	stage, err := newTrashStage(root)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.finish()
	modified := review.entries["entry"].info.ModTime()
	stage.move = func(from, to string) error {
		if err := root.WriteFile(from, []byte("new!"), 0600); err != nil {
			return err
		}
		if err := root.Chtimes(from, modified, modified); err != nil {
			return err
		}
		return root.Rename(from, to)
	}
	if err := stage.removeTree(ctx, review.entries["entry"]); !errors.Is(err, errTrashReviewChanged) || stage.deleted != 0 {
		t.Fatalf("modified content should survive the claim: %v, deleted=%d", err, stage.deleted)
	}
	if got, err := os.ReadFile(stage.recovery); err != nil || string(got) != "new!" {
		t.Fatalf("expected changed contents in recovery: %q %v", got, err)
	}
}

func TestPermanentRemovalPreservesChangedRemainingHardLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the trash provider relies on Unix inode identity and link counts")
	}
	path := t.TempDir()
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.WriteFile("first", []byte("old!"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := root.Link("first", "last"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	review, err := reviewTrash(ctx, path, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	stage, err := newTrashStage(root)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.finish()
	if err := stage.removeTree(ctx, review.entries["first"]); err != nil {
		t.Fatal(err)
	}
	modified := review.entries["last"].info.ModTime()
	if err := root.WriteFile("last", []byte("new!"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := root.Chtimes("last", modified, modified); err != nil {
		t.Fatal(err)
	}
	if err := stage.removeTree(ctx, review.entries["last"]); !errors.Is(err, errTrashReviewChanged) || stage.deleted != 1 {
		t.Fatalf("updated hard link should remain: %v, deleted=%d", err, stage.deleted)
	}
	if got, err := os.ReadFile(stage.recovery); err != nil || string(got) != "new!" {
		t.Fatalf("updated data should remain: %q %v", got, err)
	}
}

func TestPermanentRemovalPreservesChangedOrNewLinuxMetadata(t *testing.T) {
	parent := t.TempDir()
	trash := filepath.Join(parent, "files")
	info := filepath.Join(parent, "info")
	for _, path := range []string{trash, info} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"changed", "new", "same"} {
		if err := os.WriteFile(filepath.Join(trash, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	changed := filepath.Join(info, "changed.trashinfo")
	created := filepath.Join(info, "new.trashinfo")
	unchanged := filepath.Join(info, "same.trashinfo")
	for _, path := range []string{changed, unchanged} {
		if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	review, err := reviewTrash(ctx, trash, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(changed, filepath.Join(parent, "original-metadata")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{changed, created} {
		if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	result, empty, err := review.empty(ctx)
	if err != nil || !empty || result.Deleted != 3 || len(result.PostDeleteFailures) != 2 {
		t.Fatalf("unexpected outcome: %+v, empty=%t, err=%v", result, empty, err)
	}
	for _, failure := range result.PostDeleteFailures {
		if !failure.Changed || failure.Error != errTrashMetadataChanged.Error() {
			t.Fatalf("metadata change should be reported: %+v", failure)
		}
	}
	for _, path := range []string{changed, created} {
		if got, err := os.ReadFile(path); err != nil || string(got) != "keep" {
			t.Fatalf("%s should remain: %q %v", path, got, err)
		}
	}
	if _, err := os.Lstat(unchanged); !os.IsNotExist(err) {
		t.Fatalf("unchanged metadata should be removed: %v", err)
	}
}
