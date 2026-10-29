package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"rockshinops/internal/localization"
	"rockshinops/internal/metrics"
)

func TestPreferencesPersistAndKeepExplicitEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "RockshinOps", "preferences.json")
	defaults := []string{"disk"}
	got, err := readPreferences(path, defaults, "system", false, false)
	if err != nil || !reflect.DeepEqual(got.MenuBarItems, defaults) {
		t.Fatalf("first run: %+v %v", got, err)
	}
	for _, items := range [][]string{{"cpu", "memory"}, {}} {
		p := Preferences{Language: "system", MenuBarItems: items, UpdateUseSystemProxy: true}
		if err := writePreferences(path, p); err != nil {
			t.Fatal(err)
		}
		got, err = readPreferences(path, defaults, "system", false, false)
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("reload: %+v %v", got, err)
		}
	}
	if err := os.WriteFile(path, []byte(`{"menuBarItems":["bogus"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPreferences(path, defaults, "system", false, false); err == nil {
		t.Fatal("invalid persisted value silently accepted")
	}
}

func TestConfigDirectoryMigration(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "LocalOps")
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"preferences.json", "menu-bar.json"} {
		if err := os.WriteFile(filepath.Join(legacy, name), []byte(`{"saved":true}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	directory, err := migrateConfigDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"preferences.json", "menu-bar.json"} {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || string(data) != `{"saved":true}` {
			t.Fatalf("migration %s: %s, %v", name, data, err)
		}
		if _, err := os.Stat(filepath.Join(legacy, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "preferences.json"), []byte(`{"newer":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := migrateConfigDirectory(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(directory, "preferences.json"))
	if err != nil || string(data) != `{"newer":true}` {
		t.Fatalf("existing settings overwritten: %s, %v", data, err)
	}
}

func TestConfigDirectoryMigrationAbsentOrInvalidLegacy(t *testing.T) {
	root := t.TempDir()
	if got, err := migrateConfigDirectory(root); err != nil || got != filepath.Join(root, "RockshinOps") {
		t.Fatalf("fresh installation: %q, %v", got, err)
	}
	legacy := filepath.Join(root, "LocalOps")
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "preferences.json"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := migrateConfigDirectory(root); err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("invalid legacy settings: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "RockshinOps", "preferences.json")); !os.IsNotExist(err) {
		t.Fatalf("invalid settings copied: %v", err)
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
	got, err := readPreferences(path, nil, "system", false, false)
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
	got, err := readPreferences(path, []string{"disk"}, "system", false, false)
	if err != nil || got.Language != "system" {
		t.Fatalf("first-run language: %+v %v", got, err)
	}
	if err := os.WriteFile(path, []byte(`{"menuBarItems":["cpu"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = readPreferences(path, nil, "system", false, false)
	if err != nil || got.Language != "system" || !reflect.DeepEqual(got.MenuBarItems, []string{"cpu"}) {
		t.Fatalf("legacy preferences: %+v %v", got, err)
	}
	for _, language := range []string{"en", "zh", "system"} {
		want := Preferences{Language: language, MenuBarItems: []string{"memory"}}
		if err := writePreferences(path, want); err != nil {
			t.Fatal(err)
		}
		got, err = readPreferences(path, nil, "system", false, false)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("language reload: %+v %v", got, err)
		}
	}
	for _, language := range []string{"", "fr", "zh-Hans", "SYSTEM"} {
		if err := writePreferences(path, Preferences{Language: language, MenuBarItems: []string{"disk"}}); err == nil {
			t.Fatalf("accepted invalid language %q", language)
		}
	}
	got, err = readPreferences(path, nil, "system", false, false)
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

func TestMenuBarReadingsPairSymbolsWithBareValues(t *testing.T) {
	s := metrics.Snapshot{}
	s.CPU.Ready, s.CPU.Usage = true, 42
	s.Memory.Total, s.Memory.Used = 100, 75
	s.Disk.Volume.Total, s.Disk.Volume.Available = 200e9, 80e9
	s.Battery.Present, s.Battery.Percent = true, 60
	got := menuBarReadings([]string{"disk", "cpu", "memory", "gpu", "battery"}, s)
	want := [][2]string{{"internaldrive", "80.0 GB"}, {"cpu", "42%"}, {"memorychip", "75%"}, {"cube", "—"}, {"battery.50", "60%"}}
	for i, w := range want {
		if got[i].symbol != w[0] || got[i].value != w[1] {
			t.Fatalf("reading %d = %+v, want %v", i, got[i], w)
		}
	}
	s.Battery.Charging = true
	if got := menuBarReadings([]string{"battery"}, s)[0].symbol; got != "battery.100.bolt" {
		t.Fatal(got)
	}
	s.Battery.Charging, s.Battery.Percent = false, 90
	if got := menuBarReadings([]string{"battery"}, s)[0].symbol; got != "battery.100" {
		t.Fatal(got)
	}
}

func TestHideDockPreferenceDefaultAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	for _, enabled := range []bool{false, true} {
		got, err := readPreferences(path, []string{"disk"}, "system", false, enabled)
		if err != nil || got.HideDockOnClose != enabled {
			t.Fatalf("first-run Dock preference: %+v %v", got, err)
		}
	}
	if err := os.WriteFile(path, []byte(`{"menuBarItems":["disk"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readPreferences(path, nil, "system", false, false)
	if err != nil || got.HideDockOnClose {
		t.Fatalf("legacy Dock preference: %+v %v", got, err)
	}
	for _, enabled := range []bool{true, false} {
		got.HideDockOnClose = enabled
		if err := writePreferences(path, got); err != nil {
			t.Fatal(err)
		}
		restored, err := readPreferences(path, nil, "system", false, !enabled)
		if err != nil || restored.HideDockOnClose != enabled {
			t.Fatalf("restored Dock preference: %+v %v", restored, err)
		}
	}
}
