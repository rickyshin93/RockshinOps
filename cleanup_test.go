package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestCleanupRejectsUnscannedPath(t *testing.T) {
	s := &System{}
	if err := s.TrashCleanupItem(t.TempDir()); err == nil {
		t.Fatal("accepted unscanned item")
	}
	if err := s.EmptyCleanupTrash(t.TempDir()); err == nil {
		t.Fatal("accepted unscanned trash")
	}
}

func TestScanDiskBrowsesOnlyRealDirectories(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	folder := filepath.Join(home, "folder")
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "data"), []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(home, "outside")); err != nil {
		t.Fatal(err)
	}
	s := &System{config: Config{ScanMaxEntries: 100, ScanTimeoutSeconds: int(time.Second.Seconds())}}
	result, err := s.scanDisk(context.Background(), home, "")
	if err != nil || len(result.Groups) != 1 || len(result.Groups[0].Items) != 1 || !result.Groups[0].Items[0].Directory {
		t.Fatalf("unexpected scan: %+v %v", result, err)
	}
	child, err := s.scanDisk(context.Background(), home, folder)
	if err != nil || len(child.Groups[0].Items) != 1 || child.Groups[0].Items[0].Name != "data" {
		t.Fatalf("unexpected nested scan: %+v %v", child, err)
	}
	// Restore the parent scan before verifying protection against replacement.
	if _, err := s.scanDisk(context.Background(), home, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.scanDisk(context.Background(), home, filepath.Join(home, "outside")); err == nil {
		t.Fatal("followed symlink")
	}
	if _, err := s.scanDisk(context.Background(), home, filepath.Dir(home)); err == nil {
		t.Fatal("accepted path outside home")
	}
	if err := os.Rename(folder, filepath.Join(home, "original")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	if err := s.TrashCleanupItem(folder); err == nil {
		t.Fatal("accepted a replacement after scanning")
	}
}
