package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func clipboardTestRecord(t *testing.T, store clipboardStore, content clipboardContent, at time.Time) {
	t.Helper()
	if written, err := store.record(content, at); !written || err != nil {
		t.Fatalf("record clipboard item: written=%t, error=%v", written, err)
	}
}

func clipboardTestSearch(t *testing.T, store clipboardStore, query string, limit int) LauncherSearch {
	t.Helper()
	var result LauncherSearch
	if err := store.search(query, limit, false, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestClipboardHistoryPersistsAndKeepsOriginalRepresentations(t *testing.T) {
	store := clipboardStore{dir: t.TempDir(), config: clipboardHistoryConfig{MaxEntries: 8, MaxHistoryBytes: 4096}}
	text := "  " + strings.Repeat("中", 170) + "\nMiXeD case\t"
	at := time.Now().Add(-time.Minute)
	clipboardTestRecord(t, store, clipboardContent{format: "txt", data: []byte(text)}, at)
	original := clipboardTestSearch(t, store, "", 1).Items[0].ID
	image := []byte{0x89, 0x50, 0x4e, 0x47, 0x00, 0x00, 0xff}
	clipboardTestRecord(t, store, clipboardContent{format: "png", data: image}, at.Add(time.Second))
	clipboardTestRecord(t, store, clipboardContent{format: "txt", data: []byte(text)}, at.Add(2*time.Second))
	if saved, err := store.record(clipboardContent{format: "txt", data: []byte(" \n  ")}, at); saved || err != nil {
		t.Fatalf("whitespace must remain uncaptured: %t %v", saved, err)
	}

	reopened := clipboardStore{dir: store.dir, config: store.config}
	results := clipboardTestSearch(t, reopened, "", 1)
	if len(results.Items) != 1 || results.Items[0].ID != original || results.ClipboardCount != 2 || results.ClipboardBytes != int64(len(image)+len(text)) {
		t.Fatalf("reopened store did not preserve recency, deduplication and usage: %+v", results)
	}
	matching := clipboardTestSearch(t, reopened, " mixed CASE ", 8).Items
	if len(matching) != 1 || matching[0].ID != original || !utf8.ValidString(matching[0].Title) || !strings.HasSuffix(matching[0].Title, "…") {
		t.Fatalf("search must find text beyond the preview: %+v", matching)
	}
	got, err := reopened.entry(original)
	if err != nil || got.format != "txt" || string(got.data) != text {
		t.Fatal("copy must preserve the original Unicode text and whitespace")
	}
	matching = clipboardTestSearch(t, reopened, "截图", 8).Items
	if len(matching) != 1 || !matching[0].ClipboardImage {
		t.Fatalf("expected searchable image history: %+v", matching)
	}
	got, err = reopened.entry(matching[0].ID)
	if err != nil || got.format != "png" || !bytes.Equal(got.data, image) {
		t.Fatal("copy must preserve the original image bytes and format")
	}
}

func TestClipboardPinsSurviveCaptureAndClear(t *testing.T) {
	store := clipboardStore{dir: t.TempDir(), config: clipboardHistoryConfig{MaxEntries: 2, MaxHistoryBytes: 16}}
	at := time.Now().Add(-time.Minute)
	clipboardTestRecord(t, store, clipboardContent{format: "txt", data: []byte("keep")}, at)
	pinned := clipboardTestSearch(t, store, "", 2).Items[0]
	clipboardTestRecord(t, store, clipboardContent{format: "txt", data: []byte("middle")}, at.Add(time.Second))
	if err := store.setPinned(pinned.ID, true); err != nil {
		t.Fatal(err)
	}
	clipboardTestRecord(t, store, clipboardContent{format: "txt", data: []byte("new")}, at.Add(2*time.Second))
	reopened := clipboardStore{dir: store.dir, config: store.config}
	results := clipboardTestSearch(t, reopened, "", 2)
	if len(results.Items) != 2 || results.ClipboardPinnedCount != 1 || !results.Items[0].ClipboardPinned || results.Items[0].ID != pinned.ID || results.Items[0].Subtitle != pinned.Subtitle || results.Items[1].Title != "new" {
		t.Fatalf("pin or eviction was not preserved after reopening: %+v", results)
	}
	if err := reopened.clear(); err != nil {
		t.Fatal(err)
	}
	results = clipboardTestSearch(t, reopened, "keep", 2)
	if results.ClipboardCount != 1 || len(results.Items) != 1 || results.Items[0].ID != pinned.ID {
		t.Fatalf("clear removed a pinned item or kept ordinary history: %+v", results)
	}
	if err := reopened.setPinned(pinned.ID, false); err != nil {
		t.Fatal(err)
	}
	results = clipboardTestSearch(t, reopened, "", 2)
	if results.ClipboardPinnedCount != 0 || results.Items[0].ClipboardPinned {
		t.Fatalf("unpin did not restore ordinary history: %+v", results)
	}
}

func TestClipboardSnippetRetainsFullTextAfterHistoryClear(t *testing.T) {
	store := clipboardStore{dir: t.TempDir(), config: clipboardHistoryConfig{MaxEntries: 3, MaxSnippets: 1, MaxHistoryBytes: 4096}}
	text := "  " + strings.Repeat("中", 170) + "\nLong tail\t"
	at := time.Now().Add(-time.Minute)
	clipboardTestRecord(t, store, clipboardContent{format: "txt", data: []byte(text)}, at)
	item := clipboardTestSearch(t, store, "", 1).Items[0]
	if err := store.saveSnippet(item.ID, "  Sample Name "); err != nil {
		t.Fatal(err)
	}
	if err := store.saveSnippet(item.ID, "Sample Name"); err != nil {
		t.Fatalf("retry should not create another snippet: %v", err)
	}
	if err := store.saveSnippet(item.ID, "Second name"); err == nil {
		t.Fatal("snippet limit was ignored")
	}
	if err := store.clear(); err != nil {
		t.Fatal(err)
	}
	reopened := clipboardStore{dir: store.dir, config: store.config}
	for _, query := range []string{"SAMPLE name", "LONG TAIL"} {
		var results LauncherSearch
		if err := reopened.search(query, 3, true, &results); err != nil {
			t.Fatal(err)
		}
		if len(results.Items) != 0 || len(results.ClipboardSnippets) != 1 || results.ClipboardSnippets[0].Title != "Sample Name" {
			t.Fatalf("expected independently saved snippet for %q: %+v", query, results)
		}
		got, err := reopened.copyEntry(results.ClipboardSnippets[0].ID)
		if err != nil || got.format != "txt" || string(got.data) != text {
			t.Fatal("snippet copy changed its original text or whitespace")
		}
	}
}

func TestClipboardHistoryLimitsAndClearSuppressPendingCaptures(t *testing.T) {
	store := clipboardStore{dir: t.TempDir(), config: clipboardHistoryConfig{MaxEntries: 3, MaxHistoryBytes: 8}}
	at := time.Now().Add(-time.Minute)
	for index, text := range []string{"first", "b", "c", "d", "large"} {
		clipboardTestRecord(t, store, clipboardContent{format: "txt", data: []byte(text)}, at.Add(time.Duration(index)*time.Second))
	}
	history := clipboardTestSearch(t, store, "", 3)
	if len(history.Items) != 3 || history.Items[0].Title != "large" || history.Items[2].Title != "c" || history.ClipboardBytes != 7 {
		t.Fatalf("limits did not keep the newest records: %+v", history)
	}
	// A newer image takes up the entire byte budget, independently of the count cap.
	clipboardTestRecord(t, store, clipboardContent{format: "png", data: []byte("12345678")}, at.Add(5*time.Second))
	if history := clipboardTestSearch(t, store, "", 3); history.ClipboardCount != 1 || history.ClipboardBytes != 8 {
		t.Fatalf("byte cap did not prune oldest entries: %+v", history)
	}
	if _, err := store.entry(history.Items[0].ID); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("evicted item was still available: %v", err)
	}
	unrelated := filepath.Join(store.dir, "unrelated.txt")
	if err := os.WriteFile(unrelated, []byte("leave it alone"), 0600); err != nil {
		t.Fatal(err)
	}
	peer := clipboardStore{dir: store.dir, config: store.config}
	if err := store.clear(); err != nil {
		t.Fatal(err)
	}
	if saved, err := peer.record(clipboardContent{format: "txt", data: []byte("queued")}, at); saved || err != nil {
		t.Fatalf("capture from before clear was replayed: %t %v", saved, err)
	}
	if history := clipboardTestSearch(t, peer, "", 3); history.ClipboardCount != 0 || history.ClipboardBytes != 0 {
		t.Fatalf("clear left active records: %+v", history)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("clear affected a file outside its own history: %v", err)
	}
	marker, err := os.Stat(filepath.Join(store.dir, ".cleared"))
	if err != nil {
		t.Fatal(err)
	}
	freshAt := marker.ModTime().Add(time.Microsecond)
	for !time.Now().After(freshAt) {
		time.Sleep(time.Microsecond)
	}
	clipboardTestRecord(t, peer, clipboardContent{format: "txt", data: []byte("fresh")}, freshAt)
	if err := store.clear(); err != nil {
		t.Fatal(err)
	}
	if saved, err := peer.record(clipboardContent{format: "txt", data: []byte("queued2")}, freshAt); saved || err != nil {
		t.Fatalf("second clear allowed an old capture: %t %v", saved, err)
	}
}
