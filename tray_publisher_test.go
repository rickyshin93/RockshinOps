package main

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"rockshinops/internal/metrics"
)

func TestTrayPresentationKeepsNewestSnapshot(t *testing.T) {
	var pending []func()
	var painted []uint64
	publish := newTrayPublisher(func(draw func()) { pending = append(pending, draw) }, func(frame trayFrame) {
		painted = append(painted, frame.sequence)
	})
	publish(trayFrame{sequence: 1})
	publish(trayFrame{sequence: 2})
	pending[1]()
	pending[0]()
	if !reflect.DeepEqual(painted, []uint64{2}) {
		t.Fatalf("painted %v; expected the newest snapshot only", painted)
	}
}

func TestSavePreferencesReleasesLockBeforeTrayPresentation(t *testing.T) {
	system := &System{
		preferencesPath: filepath.Join(t.TempDir(), "preferences.json"),
		monitor:         metrics.NewMonitor(metrics.Options{}),
	}
	painted := make(chan Preferences, 1)
	system.publishTray = func(frame trayFrame) {
		prefs := system.Preferences()
		painted <- prefs
	}
	done := make(chan error, 1)
	go func() {
		_, err := system.SavePreferences(Preferences{Language: "en", MenuBarItems: []string{"memory"}, HideDockOnClose: true})
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("tray presentation could not read preferences while the save was waiting")
	}
	if prefs := <-painted; prefs.Language != "en" || !prefs.HideDockOnClose || prefs.MenuBarItems[0] != "memory" {
		t.Fatalf("tray saw the wrong saved preferences: %+v", prefs)
	}
}
