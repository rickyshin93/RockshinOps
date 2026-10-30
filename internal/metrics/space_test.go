package metrics

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeSize(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSpaceScanMeasuresEachFileOnceWithoutFollowingLinks(t *testing.T) {
	root := t.TempDir()
	writeSize(t, filepath.Join(root, "big", "video"), 3_000_000)
	writeSize(t, filepath.Join(root, "big", "small"), 10)
	writeSize(t, filepath.Join(root, "skipped", "data"), 2_000_000)
	outside := t.TempDir()
	writeSize(t, filepath.Join(outside, "elsewhere"), 4_000_000)
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		writeSize(t, filepath.Join(root, "pair", "a"), 1_200_000)
		if err := os.Link(filepath.Join(root, "pair", "a"), filepath.Join(root, "pair", "b")); err != nil {
			t.Fatal(err)
		}
		// Sparse files only occupy the blocks that were written.
		sparse, err := os.Create(filepath.Join(root, "sparse"))
		if err != nil {
			t.Fatal(err)
		}
		if err := sparse.Truncate(500_000_000); err != nil {
			t.Fatal(err)
		}
		sparse.Close()
	}
	var progress SpaceProgress
	tree, err := ScanSpaceTree(context.Background(), root, []string{filepath.Join(root, "skipped")}, &progress)
	if err != nil {
		t.Fatal(err)
	}
	want := int64(3_000_000)
	if runtime.GOOS != "windows" {
		want += 1_200_000
	}
	if tree.Bytes() < want || tree.Bytes() > want+200_000 {
		t.Fatalf("total %d counted links, sparse or skipped data", tree.Bytes())
	}
	listing, err := tree.Listing(root, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Children) == 0 || listing.Children[0].Name != "big" || !listing.Children[0].Directory {
		t.Fatalf("largest child should be big: %+v", listing.Children)
	}
	for _, child := range listing.Children {
		if child.Name == "link" || child.Name == "skipped" {
			t.Fatalf("unexpected child %+v", child)
		}
	}
	inner, err := tree.Listing(filepath.Join(root, "big"), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(inner.Children) != 2 || inner.Children[0].Name != "video" || !inner.Children[1].Other || inner.Children[1].Files != 1 {
		t.Fatalf("small files should be grouped after large ones: %+v", inner.Children)
	}
	if len(inner.Ancestors) != 1 || inner.Ancestors[0].Path != root {
		t.Fatalf("unexpected breadcrumb: %+v", inner.Ancestors)
	}
	wantFiles := int64(2)
	if runtime.GOOS != "windows" {
		wantFiles++
	}
	if progress.Files.Load() < wantFiles || progress.Bytes.Load() < 3_000_000 {
		t.Fatalf("progress not reported: %d files %d bytes", progress.Files.Load(), progress.Bytes.Load())
	}
	if largest := tree.Largest(5, ""); len(largest) == 0 || largest[0].Name != "video" {
		t.Fatalf("largest files: %+v", largest)
	}
}

func TestSpaceTreeMoveAndClearKeepTotals(t *testing.T) {
	root := t.TempDir()
	writeSize(t, filepath.Join(root, "work", "cache", "blob"), 2_000_000)
	writeSize(t, filepath.Join(root, "trash", "old"), 1_500_000)
	tree, err := ScanSpaceTree(context.Background(), root, nil, &SpaceProgress{})
	if err != nil {
		t.Fatal(err)
	}
	before := tree.Bytes()
	cache, trash := filepath.Join(root, "work", "cache"), filepath.Join(root, "trash")
	if !tree.Move(cache, trash) {
		t.Fatal("move failed")
	}
	if tree.Bytes() != before {
		t.Fatalf("moving into trash changed the total: %d → %d", before, tree.Bytes())
	}
	if work, _ := tree.Find(filepath.Join(root, "work")); work.Bytes != 0 {
		t.Fatalf("source folder still counts moved item: %+v", work)
	}
	if moved, ok := tree.Find(filepath.Join(trash, "cache")); !ok || moved.Bytes < 2_000_000 {
		t.Fatalf("moved item missing from trash: %+v", moved)
	}
	if largest := tree.Largest(5, trash); len(largest) != 0 {
		t.Fatalf("trash contents listed as large files: %+v", largest)
	}
	tree.Clear(trash)
	if entry, _ := tree.Find(trash); entry.Bytes != 0 || tree.Bytes() >= before-3_000_000 {
		t.Fatalf("clearing trash kept bytes: %+v total %d", entry, tree.Bytes())
	}
}

func TestSpaceScanStopsWhenCancelled(t *testing.T) {
	root := t.TempDir()
	writeSize(t, filepath.Join(root, "a", "b"), 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ScanSpaceTree(ctx, root, nil, &SpaceProgress{}); err == nil {
		t.Fatal("cancelled scan returned a tree")
	}
}
