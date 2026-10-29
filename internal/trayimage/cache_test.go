package trayimage

import (
	"errors"
	"sync"
	"testing"
)

func TestSharedTrayRenderingAndInvalidation(t *testing.T) {
	var cache imageCache
	var calls int
	draw := func([]Item) ([]byte, error) { calls++; return []byte("png"), nil }
	items := []Item{{"cpu", "20%"}}
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() { defer group.Done(); cache.get(items, draw) }()
	}
	group.Wait()
	if calls != 1 {
		t.Fatalf("unchanged readings drew %d times", calls)
	}
	items[0].Text = "21%"
	cache.get(items, draw)
	items[0].Symbol = "memorychip"
	cache.get(items, draw)
	if calls != 3 {
		t.Fatal("a changed value or symbol reused stale pixels")
	}
	failed := []Item{{"invalid", "1"}}
	if _, err := cache.get(failed, func([]Item) ([]byte, error) { return nil, errors.New("draw failed") }); err == nil {
		t.Fatal("draw failure was hidden")
	}
	cache.get(failed, draw)
	if calls != 4 {
		t.Fatal("a failed draw poisoned later rendering")
	}
}
