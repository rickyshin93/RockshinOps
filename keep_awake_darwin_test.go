package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestIdleSleepAssertionNativeTimeoutAndRelease(t *testing.T) {
	assertion, err := takeIdleSleepAssertion(2 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := releaseIdleSleepAssertion(assertion); err != nil {
			t.Error(err)
		}
	}()
	const name = "RockshinOps: Keep Awake"
	pid := fmt.Sprintf("pid %d(", os.Getpid())
	active := func() bool {
		t.Helper()
		output, err := exec.Command("/usr/bin/pmset", "-g", "assertions").Output()
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(output), "\n") {
			if strings.Contains(line, pid) && strings.Contains(line, "PreventUserIdleSystemSleep") && strings.Contains(line, name) {
				return true
			}
		}
		return false
	}
	if !active() {
		t.Fatal("pmset did not report the native assertion")
	}
	until := time.Now().Add(5 * time.Second)
	for active() {
		if time.Now().After(until) {
			t.Fatal("the operating system did not turn the assertion off")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
