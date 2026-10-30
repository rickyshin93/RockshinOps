package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStartupDefaultAndSystemOptOut(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings", "login-default-applied")
	enabled, calls := false, 0
	s := &startupSettings{marker: path, read: func() bool { return enabled }, set: func(value bool) error { enabled = value; calls++; return nil }}
	s.initialize(true)
	if !enabled || calls != 1 || s.status().Error != "" {
		t.Fatalf("default was not applied: %+v, calls=%d", s.status(), calls)
	}
	// Simulate disabling in OS settings and restarting the app.
	enabled = false
	s.initialize(true)
	if enabled || calls != 1 {
		t.Fatal("overrode an OS-level opt-out")
	}
	system := &System{startup: s}
	if got := system.SetOpenAtLogin(true); !got.Enabled || got.Error != "" {
		t.Fatalf("enable: %+v", got)
	}
	if got := system.SetOpenAtLogin(false); got.Enabled || got.Error != "" {
		t.Fatalf("disable: %+v", got)
	}
	s.initialize(true)
	if enabled || calls != 3 {
		t.Fatal("overrode an app-level opt-out")
	}
}

func TestStartupFailureIsVisibleAndNotRepeated(t *testing.T) {
	calls := 0
	s := &startupSettings{marker: filepath.Join(t.TempDir(), "marker"), read: func() bool { return false }, set: func(bool) error { calls++; return errors.New("approval required") }}
	s.initialize(true)
	if got := s.status(); got.Enabled || got.Error != "approval required" {
		t.Fatalf("failure hidden: %+v", got)
	}
	s.initialize(true)
	if calls != 1 {
		t.Fatal("automatically retried registration")
	}
	s.set = func(bool) error { return nil }
	if got := (&System{startup: s}).SetOpenAtLogin(true); got.Error == "" || got.Enabled {
		t.Fatalf("reported an unapplied setting as successful: %+v", got)
	}
}

func TestStartupDevelopmentAndMarkerFailure(t *testing.T) {
	root := t.TempDir()
	s := &startupSettings{dev: true, marker: filepath.Join(root, "marker"), read: func() bool { t.Fatal("read OS in dev"); return false }, set: func(bool) error { t.Fatal("changed OS in dev"); return nil }}
	s.initialize(true)
	if got := (&System{startup: s}).SetOpenAtLogin(true); got.Available || got.Error == "" {
		t.Fatalf("dev enabled: %+v", got)
	}
	if _, err := os.Stat(s.marker); !os.IsNotExist(err) {
		t.Fatalf("dev consumed default: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "file"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	s.dev, s.marker = false, filepath.Join(root, "file", "marker")
	s.read = func() bool { return false }
	s.initialize(true)
	if got := s.status(); got.Error == "" {
		t.Fatal("marker failure hidden")
	}
	if got := (&System{startup: s}).SetOpenAtLogin(false); got.Error == "" {
		t.Fatal("changed setting without retaining opt-out")
	}
}

func TestStartupConfiguredDefaultOff(t *testing.T) {
	s := &startupSettings{marker: filepath.Join(t.TempDir(), "marker"), read: func() bool { return false }, set: func(bool) error { t.Fatal("unneeded OS mutation"); return nil }}
	s.initialize(false)
	if s.status().Error != "" {
		t.Fatal(s.status())
	}
}
