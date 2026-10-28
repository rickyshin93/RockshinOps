package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"localops/internal/localization"
	"localops/internal/metrics"
)

func TestPreferencesPersistAndKeepExplicitEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "LocalOps", "preferences.json")
	defaults := []string{"disk"}
	got, err := readPreferences(path, defaults, "system", false)
	if err != nil || !reflect.DeepEqual(got.MenuBarItems, defaults) {
		t.Fatalf("first run: %+v %v", got, err)
	}
	for _, items := range [][]string{{"cpu", "memory"}, {}} {
		p := Preferences{Language: "system", MenuBarItems: items, UpdateUseSystemProxy: true}
		if err := writePreferences(path, p); err != nil {
			t.Fatal(err)
		}
		got, err = readPreferences(path, defaults, "system", false)
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("reload: %+v %v", got, err)
		}
	}
	if err := os.WriteFile(path, []byte(`{"menuBarItems":["bogus"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPreferences(path, defaults, "system", false); err == nil {
		t.Fatal("invalid persisted value silently accepted")
	}
}

func TestInvalidPreferencesDoNotOverwriteSavedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	valid := Preferences{Language: "system", MenuBarItems: []string{"disk"}}
	if err := writePreferences(path, valid); err != nil {
		t.Fatal(err)
	}
	for _, items := range [][]string{nil, {"unknown"}, {"cpu", "cpu"}} {
		if err := writePreferences(path, Preferences{Language: "system", MenuBarItems: items}); err == nil {
			t.Fatalf("accepted %v", items)
		}
	}
	got, err := readPreferences(path, nil, "system", false)
	if err != nil || !reflect.DeepEqual(got, valid) {
		t.Fatalf("saved state changed: %+v %v", got, err)
	}
	if err := writePreferences(filepath.Join(path, "child"), valid); err == nil {
		t.Fatal("filesystem failure hidden")
	}
}

func TestMenuBarSelectedMetricsAndUnavailableValues(t *testing.T) {
	localization.SetLanguage("zh")
	t.Cleanup(func() { localization.SetLanguage("en") })
	s := metrics.Snapshot{}
	s.CPU.Ready, s.CPU.Usage = true, 42
	s.Memory.Total, s.Memory.Used = 100, 75
	s.Disk.Volume.Total, s.Disk.Volume.Available = 200e9, 80e9
	if got := menuBarTitle(Preferences{Language: "system", MenuBarItems: []string{"disk"}}, s); got != "80.0 GB" {
		t.Fatal(got)
	}
	if got := menuBarTitle(Preferences{Language: "system", MenuBarItems: []string{"cpu", "memory", "gpu"}}, s); got != "CPU 42% · 内存 75% · GPU —" {
		t.Fatal(got)
	}
	if got := menuBarTitle(Preferences{Language: "system", MenuBarItems: []string{}}, s); got != "" {
		t.Fatal(got)
	}
	s.CPU.Error = "failed"
	if got := menuBarTitle(Preferences{Language: "system", MenuBarItems: []string{"cpu"}}, s); strings.Contains(got, "42") {
		t.Fatal("stale CPU value", got)
	}
}

func TestLanguageDefaultMigrationAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	got, err := readPreferences(path, []string{"disk"}, "system", false)
	if err != nil || got.Language != "system" {
		t.Fatalf("first-run language: %+v %v", got, err)
	}
	if err := os.WriteFile(path, []byte(`{"menuBarItems":["cpu"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = readPreferences(path, nil, "system", false)
	if err != nil || got.Language != "system" || !reflect.DeepEqual(got.MenuBarItems, []string{"cpu"}) {
		t.Fatalf("legacy preferences: %+v %v", got, err)
	}
	for _, language := range []string{"en", "zh", "system"} {
		want := Preferences{Language: language, MenuBarItems: []string{"memory"}}
		if err := writePreferences(path, want); err != nil {
			t.Fatal(err)
		}
		got, err = readPreferences(path, nil, "system", false)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("language reload: %+v %v", got, err)
		}
	}
	for _, language := range []string{"", "fr", "zh-Hans", "SYSTEM"} {
		if err := writePreferences(path, Preferences{Language: language, MenuBarItems: []string{"disk"}}); err == nil {
			t.Fatalf("accepted invalid language %q", language)
		}
	}
	got, err = readPreferences(path, nil, "system", false)
	if err != nil || got.Language != "system" || !reflect.DeepEqual(got.MenuBarItems, []string{"memory"}) {
		t.Fatalf("failed save altered preferences: %+v %v", got, err)
	}
}

func TestMenuBarLanguageChangesWithoutChangingReadings(t *testing.T) {
	t.Cleanup(func() { localization.SetLanguage("en") })
	snapshot := metrics.Snapshot{}
	snapshot.Memory.Total, snapshot.Memory.Used = 100, 75
	preferences := Preferences{Language: "system", MenuBarItems: []string{"memory"}}
	localization.SetLanguage("en")
	if got := menuBarTitle(preferences, snapshot); got != "Memory 75%" {
		t.Fatal(got)
	}
	localization.SetLanguage("zh")
	if got := menuBarTitle(preferences, snapshot); got != "内存 75%" {
		t.Fatal(got)
	}
}
