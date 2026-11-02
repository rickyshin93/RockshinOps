package fileindex

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"runtime"
	"slices"
	"testing"
	"time"
)

var home = flag.Bool("home", false, "measure indexing and search on the current user's home folder")

// go test ./internal/fileindex -run TestHome -home -v
func TestHome(t *testing.T) {
	if !*home {
		t.Skip("pass -home to measure the real home folder")
	}
	root, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../rockshinops.json")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Launcher struct {
			Files map[string]Rules `json:"files"`
		} `json:"launcher"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	rules := config.Launcher.Files[runtime.GOOS]
	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	ix := New(root, rules)
	start := time.Now()
	if err := ix.Build(context.Background()); err != nil {
		t.Fatal(err)
	}
	built := time.Since(start)
	var after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&after)
	t.Logf("entries %d, build %v, heap %.1f MB, included %v", ix.Len(), built.Round(time.Millisecond), float64(after.HeapAlloc-before.HeapAlloc)/1e6, ix.Roots()[1:])

	var took []time.Duration
	for _, q := range []string{"复核成绩", "第一题", "status", "sta", "s", "a", "readme", "main.go", "作业 复核", "localops status", "xlsx", "zzzzqqq", "diyi", "fhcj", "fuhecj", "zuoye fhcj", "dy"} {
		start := time.Now()
		got := ix.Search(q, 20)
		took = append(took, time.Since(start))
		if len(got) > 0 {
			t.Logf("%-16q %6v  %d  top %s", q, took[len(took)-1].Round(time.Microsecond), len(got), got[0].Path)
		}
	}
	slices.Sort(took)
	t.Logf("search p50 %v, max %v", took[len(took)/2].Round(time.Microsecond), took[len(took)-1].Round(time.Microsecond))
}
