package metrics

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConnectionRecordsKeepIPv6AndProcessOwnership(t *testing.T) {
	items, err := parseConnections([]byte("p42\x00cExample Helper\x00u501\x00\nf7\x00tIPv6\x00PTCP\x00n[::1]:8000->[::1]:5000\x00TST=ESTABLISHED\x00\nf8\x00tIPv4\x00PUDP\x00n*:5353\x00\n"))
	if err != nil || len(items) != 2 {
		t.Fatalf("%+v %v", items, err)
	}
	if items[0].Remote != "[::1]:5000" || items[0].Process != "Example Helper" || items[1].PID != 42 || items[1].State != "" {
		t.Fatalf("incorrect records: %+v", items)
	}
	if _, err := parseConnections([]byte("pinvalid\x00")); err == nil {
		t.Fatal("accepted invalid process")
	}
}

func TestScanDoesNotFollowSymlinksOrModifyFiles(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "cache", "item")
	if err := os.WriteFile(file, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	result := ScanSpace(context.Background(), []ScanTarget{{Name: "Test", Path: root}}, 100, 10, time.Second)
	if !result.Complete || result.Groups[0].Bytes != 7 || result.Groups[0].Files != 1 {
		t.Fatalf("unexpected scan: %+v", result)
	}
	if data, err := os.ReadFile(file); err != nil || string(data) != "content" {
		t.Fatal("scan modified a file")
	}
	limited := ScanSpace(context.Background(), []ScanTarget{{Name: "Test", Path: root}}, 1, 10, time.Second)
	if limited.Complete || limited.Warning == "" {
		t.Fatal("limited scan falsely reported complete")
	}
}
