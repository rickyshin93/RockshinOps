package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"rockshinops/internal/fileindex"
)

// A subprocess runs the real AppKit loop on its main thread. A unit test of
// a close predicate would miss MyGo asking hidden windows to close on Quit.
func TestMain(m *testing.M) {
	lazyStartup := os.Getenv("ROCKSHINOPS_TEST_LAUNCHER_STARTUP") == "1"
	if !lazyStartup && os.Getenv("ROCKSHINOPS_TEST_LAUNCHER_QUIT") != "1" {
		os.Exit(m.Run())
	}
	mygo.App.SetName("RockshinOps Lifecycle Test")
	mygo.App.WhenReady(func() {
		l := &launcher{}
		if lazyStartup {
			var err error
			l, err = newLauncher(os.Getenv("ROCKSHINOPS_TEST_LAUNCHER_PREFS"), launcherPrefs{Enabled: true, Shortcut: "Ctrl+Alt+Shift+F18"}, "", fileindex.Rules{}, nil, time.Hour)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			l.start()
			if l.lastError != "" {
				fmt.Fprintln(os.Stderr, l.lastError)
				os.Exit(2)
			}
			if l.panel.Load() != nil || len(mygo.Windows()) != 0 {
				fmt.Fprintln(os.Stderr, "enabling Launcher allocated a hidden WebView before first use")
				os.Exit(2)
			}
			l.toggle()
		} else {
			l.ensurePanel()
		}
		panel := l.panel.Load()
		if panel == nil {
			fmt.Fprintln(os.Stderr, "first invocation did not create the panel")
			os.Exit(2)
		}
		panel.Show()
		panel.Close()
		mygo.RunOnMain(func() {
			if panel.IsVisible() || len(mygo.Windows()) != 1 {
				fmt.Fprintln(os.Stderr, "ordinary close must hide and retain the launcher")
				os.Exit(2)
			}
			mygo.App.Quit()
		})
	})
	go func() {
		time.Sleep(20 * time.Second)
		fmt.Fprintln(os.Stderr, "Quit was blocked by the hidden launcher")
		os.Exit(2)
	}()
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func TestLauncherCreatesWindowOnlyOnFirstUse(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "ROCKSHINOPS_TEST_LAUNCHER_STARTUP=1", "ROCKSHINOPS_TEST_LAUNCHER_PREFS="+filepath.Join(t.TempDir(), "launcher.json"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native launcher startup: %v\n%s", err, output)
	}
}

func TestQuitWithHiddenLauncher(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "ROCKSHINOPS_TEST_LAUNCHER_QUIT=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native launcher lifecycle: %v\n%s", err, output)
	}
}
