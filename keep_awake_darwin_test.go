package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"rockshinops/internal/localization"
)

func TestIdleSleepAssertionNativeTimeoutAndRelease(t *testing.T) {
	for _, tc := range []struct {
		kind    string
		display bool
	}{
		{"PreventUserIdleSystemSleep", false},
		{"PreventUserIdleDisplaySleep", true},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			id, err := takeIdleSleepAssertion(2*time.Second, tc.display)
			requireAwakeOK(t, err)
			t.Cleanup(func() { requireAwakeOK(t, releaseIdleSleepAssertion(id)) })
			requireAwakeAssertion(t, tc.kind)
			until := time.Now().Add(5 * time.Second)
			for len(awakeAssertions(t)) != 0 {
				if time.Now().After(until) {
					t.Fatal("the operating system did not turn the assertion off")
				}
				time.Sleep(100 * time.Millisecond)
			}
		})
	}
}

func TestKeepAwakeSwitchExtendAndStop(t *testing.T) {
	a := newKeepAwake([]int{30, 60, 120}, 15)
	t.Cleanup(func() { requireAwakeOK(t, a.stop()) })
	requireAwakeOK(t, a.start(time.Hour))
	initialDeadline := a.deadline
	requireAwakeAssertion(t, "PreventUserIdleSystemSleep")
	if !a.extendItem.IsVisible() || !a.displayItem.IsVisible() || a.displayItem.IsChecked() {
		t.Fatal("timed session should show Extend and the unchecked display option")
	}

	requireAwakeOK(t, a.toggleDisplay())
	if !a.deadline.Equal(initialDeadline) || !a.displayItem.IsChecked() {
		t.Fatal("enabling the display option should keep the deadline and check the menu item")
	}
	requireAwakeAssertion(t, "PreventUserIdleDisplaySleep")
	requireAwakeOK(t, a.extend())
	if !a.deadline.Equal(initialDeadline.Add(15*time.Minute)) || !a.displayItem.IsChecked() {
		t.Fatal("extension should add time to the original deadline and preserve the display option")
	}
	requireAwakeAssertion(t, "PreventUserIdleDisplaySleep")
	requireAwakeOK(t, a.toggleDisplay())
	requireAwakeAssertion(t, "PreventUserIdleSystemSleep")

	requireAwakeOK(t, a.start(0))
	requireAwakeOK(t, a.extend())
	if !a.deadline.IsZero() || a.extendItem.IsVisible() || a.displayItem.IsChecked() {
		t.Fatal("a new indefinite session should hide Extend and reset the display option")
	}
	if a.menu.Subtitle != localization.T("保持唤醒中 · 无限期") {
		t.Fatalf("wrong indefinite subtitle: %q", a.menu.Subtitle)
	}
	requireAwakeAssertion(t, "PreventUserIdleSystemSleep")
	requireAwakeOK(t, a.toggleDisplay())
	requireAwakeAssertion(t, "PreventUserIdleDisplaySleep")

	requireAwakeOK(t, a.stop())
	if a.menu.Subtitle != "" || a.stopItem.IsVisible() || a.displayItem.IsVisible() || a.displayItem.IsChecked() {
		t.Fatal("stopping should clear the subtitle and session actions")
	}
	if active := awakeAssertions(t); len(active) != 0 {
		t.Fatalf("stopping left active assertions: %v", active)
	}
}

func awakeAssertions(t *testing.T) []string {
	t.Helper()
	output, err := exec.Command("/usr/bin/pmset", "-g", "assertions").Output()
	requireAwakeOK(t, err)
	pid := fmt.Sprintf("pid %d(", os.Getpid())
	var active []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, pid) && strings.Contains(line, "RockshinOps: Keep Awake") {
			active = append(active, line)
		}
	}
	return active
}

func requireAwakeAssertion(t *testing.T, kind string) {
	t.Helper()
	active := awakeAssertions(t)
	if len(active) != 1 || !strings.Contains(active[0], kind+" named:") {
		t.Fatalf("want exactly one %s assertion, got %v", kind, active)
	}
}

func requireAwakeOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
