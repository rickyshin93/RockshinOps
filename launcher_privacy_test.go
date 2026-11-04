package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"rockshinops/internal/fileindex"
)

func TestLauncherStartupSkipsPrivateAppFiles(t *testing.T) {
	cfg, home := launcherPrivacyFixture(t)
	ix := fileindex.New(home, cfg.Launcher.Files["darwin"])
	if err := ix.Build(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := ix.Search("private-report", 10); len(got) != 0 {
		t.Fatalf("startup indexed private App data: %+v", got)
	}
	// A filesystem hint under an excluded directory must not bring it back.
	ix.Refresh(filepath.Join(home, "Library/Containers"))
	if got := ix.Search("private-report", 10); len(got) != 0 {
		t.Fatalf("watch hint indexed private App data: %+v", got)
	}
	for _, query := range []string{"public-report", "cloud-report"} {
		if len(ix.Search(query, 10)) != 1 {
			t.Fatalf("lost ordinary file search: %s", query)
		}
	}
}

func launcherPrivacyFixture(t *testing.T) (Config, string) {
	t.Helper()
	cfg, err := parseConfig(configJSON)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	for _, path := range []string{
		"Documents/public-report.txt",
		"Library/Mobile Documents/com~apple~CloudDocs/cloud-report.txt",
		"Library/Containers/com.tencent.xinWeChat/Data/Documents/xwechat_files/user/msg/file/private-report.txt",
		"Library/Group Containers/another-app/private-report.txt",
	} {
		abs := filepath.Join(home, path)
		if err := os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return cfg, home
}

func TestLauncherWeChatRequiresSessionOptIn(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS WeChat integration")
	}
	cfg, home := launcherPrivacyFixture(t)
	prefsPath := filepath.Join(t.TempDir(), "launcher.json")
	prefs := launcherPrefs{Enabled: true, Files: true, Shortcut: "Alt+Space"}
	if err := writeLauncherPrefs(prefsPath, prefs); err != nil {
		t.Fatal(err)
	}
	makeLauncher := func() *launcher {
		l, err := newLauncher(prefsPath, launcherPrefs{}, home, cfg.Launcher.Files["darwin"], cfg.Launcher.WeChatPaths, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		f := &fakeShortcuts{}
		l.register, l.unregister = f.register, f.unregister
		t.Cleanup(func() { l.mu.Lock(); defer l.mu.Unlock(); l.stopIndexing() })
		l.start()
		waitLauncherIndex(t, l)
		return l
	}
	l := makeLauncher()
	assertPrivateResults := func(want int) {
		t.Helper()
		waitLauncherIndex(t, l)
		if got := l.query("private-report"); len(got) != want {
			t.Fatalf("private results: got %+v, want %d", got, want)
		}
	}
	assertPrivateResults(0)
	if err := l.setWeChat(true); err != nil {
		t.Fatal(err)
	}
	assertPrivateResults(1) // only WeChat, never another app's group container
	if !l.settings().WeChat {
		t.Fatal("enabled session not reported")
	}
	// A preference write failure must restore the active session as well.
	blockedPath := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blockedPath, nil, 0600); err != nil {
		t.Fatal(err)
	}
	l.path = filepath.Join(blockedPath, "launcher.json")
	if _, err := l.set(launcherPrefs{Shortcut: prefs.Shortcut}); err == nil {
		t.Fatal("expected preference write failure")
	}
	l.path = prefsPath
	assertPrivateResults(1)
	if !l.settings().WeChat {
		t.Fatal("failed save lost session opt-in")
	}
	if _, err := l.set(prefs); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(prefsPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "weChat") {
		t.Fatal("persisted session opt-in")
	}
	restarted := makeLauncher()
	if restarted.settings().WeChat || len(restarted.query("private-report")) != 0 {
		t.Fatal("restart resumed private indexing")
	}
	if err := l.setWeChat(false); err != nil {
		t.Fatal(err)
	}
	assertPrivateResults(0)
	// Cancel before the new index finishes, then ensure it cannot publish later.
	for range 5 {
		if err := l.setWeChat(true); err != nil {
			t.Fatal(err)
		}
		if err := l.setWeChat(false); err != nil {
			t.Fatal(err)
		}
	}
	assertPrivateResults(0)
	for _, disabled := range []launcherPrefs{
		{Enabled: true, Files: false, Shortcut: prefs.Shortcut},
		{Enabled: false, Files: true, Shortcut: prefs.Shortcut},
	} {
		if err := l.setWeChat(true); err != nil {
			t.Fatal(err)
		}
		if _, err := l.set(disabled); err != nil {
			t.Fatal(err)
		}
		if l.settings().WeChat || l.index.Load() != nil {
			t.Fatal("disable retained private search")
		}
		if err := l.setWeChat(true); err == nil {
			t.Fatal("enabled WeChat without file search")
		}
		if _, err := l.set(prefs); err != nil {
			t.Fatal(err)
		}
		assertPrivateResults(0)
	}
}

func TestLauncherWeChatUnavailableDoesNotEnable(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS WeChat integration")
	}
	l := &launcher{home: t.TempDir(), prefs: launcherPrefs{Enabled: true, Files: true}, weChatPaths: []string{"Library/Containers/wechat/*/file"}}
	if err := l.setWeChat(true); err == nil {
		t.Fatal("missing WeChat directory reported success")
	}
	if l.weChat || l.stopFiles != nil {
		t.Fatal("failed opt-in changed indexing")
	}
	denied := filepath.Join(l.home, "Library/Containers/wechat")
	if err := os.MkdirAll(denied, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(denied, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(denied, 0700) })
	if err := l.setWeChat(true); err == nil {
		t.Fatal("denied WeChat directory reported success")
	}
	if l.weChat || l.stopFiles != nil {
		t.Fatal("denied opt-in changed indexing")
	}
}

func waitLauncherIndex(t *testing.T, l *launcher) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for l.indexing.Load() {
		if time.Now().After(deadline) {
			t.Fatal("file index did not finish")
		}
		time.Sleep(time.Millisecond)
	}
}
