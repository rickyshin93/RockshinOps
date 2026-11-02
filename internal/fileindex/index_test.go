package fileindex

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const wechat = "Library/Containers/com.tencent.xinWeChat/Data/Documents/xwechat_files/u1/msg/file"

var testRules = Rules{
	SkipNames: []string{"node_modules", ".git"},
	SkipPaths: []string{"Library"},
	Include:   []string{"Library/Containers/com.tencent.xinWeChat/Data/Documents/xwechat_files/*/msg/file"},
}

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	now := time.Now()
	for path, age := range map[string]time.Duration{
		"Documents/作业/outputs/x/第一题复核成绩.xlsx":                time.Minute,
		"Documents/作业/outputs/x/第一题复核成绩.xlsx.inspect.ndjson": 2 * time.Minute,
		wechat + "/2026-11/第一题复核成绩.xlsx":                      3 * time.Minute,
		"Library/Caches/junk/第一题复核成绩.xlsx":                     time.Minute,
		"project/node_modules/pkg/第一题复核成绩.js":                  time.Minute,
		"project/STATUS.md":                                    time.Hour,
		"old/STATUS.md":                                        2 * 365 * 24 * time.Hour,
		".codex/worktrees/r/STATUS.md":                         time.Minute,
		".zshrc":                                               24 * time.Hour,
	} {
		write(t, filepath.Join(root, path), now.Add(-age))
	}
	return root
}

func write(t *testing.T, path string, mod time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func built(t *testing.T, root string) *Index {
	t.Helper()
	ix := New(root, testRules)
	if err := ix.Build(context.Background()); err != nil {
		t.Fatal(err)
	}
	return ix
}

func paths(root string, matches []Match) []string {
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i], _ = filepath.Rel(root, m.Path)
		out[i] = filepath.ToSlash(out[i])
	}
	return out
}

func TestSearchRanksAndScopes(t *testing.T) {
	root := fixture(t)
	ix := built(t, root)

	got := paths(root, ix.Search("复核成绩", 10))
	want := []string{
		"Documents/作业/outputs/x/第一题复核成绩.xlsx",
		wechat + "/2026-11/第一题复核成绩.xlsx",
		"Documents/作业/outputs/x/第一题复核成绩.xlsx.inspect.ndjson",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("复核成绩:\n got %q\nwant %q", got, want)
	}
	if got := paths(root, ix.Search("第一题", 1)); !slices.Equal(got, want[:1]) {
		t.Errorf("第一题: %q", got)
	}
	// Recent first, but copies inside hidden folders (tool caches, worktrees) come last.
	if got := paths(root, ix.Search("status", 10)); !slices.Equal(got, []string{"project/STATUS.md", "old/STATUS.md", ".codex/worktrees/r/STATUS.md"}) {
		t.Errorf("status: %q", got)
	}
	if got := paths(root, ix.Search("zshrc", 10)); !slices.Equal(got, []string{".zshrc"}) {
		t.Errorf("hidden files themselves are found: %q", got)
	}
	// Earlier words narrow by the folders above the name, including the app folders of included places.
	if got := paths(root, ix.Search("作业 复核", 10)); !slices.Equal(got, []string{want[0], want[2]}) {
		t.Errorf("作业 复核: %q", got)
	}
	if got := paths(root, ix.Search("fhcj", 10)); !slices.Equal(got, want) {
		t.Errorf("pinyin initials: %q", got)
	}
	if got := paths(root, ix.Search("zuoye diyi", 10)); !slices.Equal(got, []string{want[0], want[2]}) {
		t.Errorf("pinyin folder words: %q", got)
	}
	if got := paths(root, ix.Search("wechat 复核", 10)); !slices.Equal(got, want[1:2]) {
		t.Errorf("wechat 复核: %q", got)
	}
	// The folders leading to an included place are not results themselves.
	for _, q := range []string{"library", "containers", "caches", "node_modules", "pkg"} {
		if got := ix.Search(q, 10); len(got) != 0 {
			t.Errorf("%s: %q", q, paths(root, got))
		}
	}
	if got := ix.Search("  ", 10); len(got) != 0 {
		t.Errorf("blank query: %q", paths(root, got))
	}
}

func TestRefreshFollowsChanges(t *testing.T) {
	root := fixture(t)
	ix := built(t, root)
	before := ix.Len()

	write(t, filepath.Join(root, "project", "新建报告.docx"), time.Now())
	ix.Refresh(filepath.Join(root, "project"))
	if got := paths(root, ix.Search("新建报告", 10)); !slices.Equal(got, []string{"project/新建报告.docx"}) {
		t.Errorf("new file: %q", got)
	}

	// A hint below folders the index has not seen yet refreshes the nearest known one.
	write(t, filepath.Join(root, "project", "a", "b", "deep.txt"), time.Now())
	ix.Refresh(filepath.Join(root, "project", "a", "b"))
	if got := paths(root, ix.Search("deep", 10)); !slices.Equal(got, []string{"project/a/b/deep.txt"}) {
		t.Errorf("new folders: %q", got)
	}

	if err := os.RemoveAll(filepath.Join(root, "old")); err != nil {
		t.Fatal(err)
	}
	ix.Refresh(root)
	if got := paths(root, ix.Search("status", 10)); !slices.Equal(got, []string{"project/STATUS.md", ".codex/worktrees/r/STATUS.md"}) {
		t.Errorf("removed folder: %q", got)
	}
	// Refreshing the root keeps the included place and still skips the rest of Library.
	if got := paths(root, ix.Search("wechat 复核", 10)); len(got) != 1 {
		t.Errorf("included place lost: %q", got)
	}
	if got := ix.Search("caches", 10); len(got) != 0 {
		t.Errorf("skipped folder indexed: %q", paths(root, got))
	}

	write(t, filepath.Join(root, wechat, "2026-11", "合同.pdf"), time.Now())
	ix.Refresh(filepath.Join(root, wechat, "2026-11"))
	if got := paths(root, ix.Search("合同", 10)); len(got) != 1 || !strings.HasPrefix(got[0], wechat) {
		t.Errorf("included place refresh: %q", got)
	}
	// new file + a + b + deep.txt + 合同.pdf - (old + old/STATUS.md)
	if got := ix.Len(); got != before+3 {
		t.Errorf("entries %d, want %d", got, before+3)
	}
}
