package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// A subprocess runs the real AppKit loop on its main thread. A unit test of
// a close predicate would miss MyGo asking hidden windows to close on Quit.
func TestMain(m *testing.M) {
	if os.Getenv("ROCKSHINOPS_TEST_LAUNCHER_QUIT") != "1" {
		os.Exit(m.Run())
	}
	mygo.App.SetName("RockshinOps Lifecycle Test")
	mygo.App.WhenReady(func() {
		l := &launcher{}
		l.ensurePanel()
		panel := l.panel.Load()
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

func TestQuitWithHiddenLauncher(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "ROCKSHINOPS_TEST_LAUNCHER_QUIT=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native launcher lifecycle: %v\n%s", err, output)
	}
}
