package menubar

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func testSettings() Settings {
	a := Appearance{Tint: "none", Color: "#34465C", Shape: "none"}
	return Settings{UsePanel: true, HoverDelay: 0.2, RehideDelay: 15, RevealCheckInterval: 0.5, TemporaryRevealSeconds: 15, ShowAllOnDrag: true, Divider: "line", Light: a, Dark: a}
}
func TestSettingsRoundTripAndUpgrade(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences", "menu-bar.json")
	defaults := testSettings()
	got, err := Load(path, defaults)
	if err != nil || !reflect.DeepEqual(got, defaults) {
		t.Fatalf("first run: %#v %v", got, err)
	}
	got.Enabled = true
	got.SearchHotkey = "Cmd+Shift+M"
	got.Dark.Tint = "solid"
	if err = Save(path, got); err != nil {
		t.Fatal(err)
	}
	saved, err := Load(path, defaults)
	if err != nil || !reflect.DeepEqual(got, saved) {
		t.Fatalf("round trip: %#v %v", saved, err)
	}
	if err = os.WriteFile(path, []byte(`{"enabled":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	saved, err = Load(path, defaults)
	if err != nil || !saved.Enabled || saved.HoverDelay != defaults.HoverDelay || saved.RevealCheckInterval != defaults.RevealCheckInterval || saved.TemporaryRevealSeconds != defaults.TemporaryRevealSeconds {
		t.Fatalf("missing new fields: %#v %v", saved, err)
	}
	if err = os.WriteFile(path, []byte(`{"divider":"invalid"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(path, defaults); err == nil {
		t.Fatal("corrupt settings accepted")
	}
}
func TestRejectInvalidSettingsWithoutReplacingSavedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "menu-bar.json")
	valid := testSettings()
	if err := Save(path, valid); err != nil {
		t.Fatal(err)
	}
	cases := []func(*Settings){func(s *Settings) { s.TemporaryRevealSeconds = 4 }, func(s *Settings) { s.TemporaryRevealSeconds = 61 }, func(s *Settings) { s.TemporaryRevealSeconds = math.NaN() }, func(s *Settings) { s.RevealCheckInterval = 0 }, func(s *Settings) { s.RevealCheckInterval = math.Inf(1) }, func(s *Settings) { s.RehideDelay = -1 }, func(s *Settings) { s.HoverDelay = 6 }, func(s *Settings) { s.HoverDelay = math.NaN() }, func(s *Settings) { s.Light.Color = "#12x45g" }, func(s *Settings) { s.Dark.Shape = "circle" }, func(s *Settings) { s.Light.Tint = "blur" }, func(s *Settings) { s.ToggleHotkey = "Cmd+M"; s.SearchHotkey = "Cmd+M" }}
	for _, change := range cases {
		s := valid
		change(&s)
		if err := Save(path, s); err == nil {
			t.Fatalf("accepted %#v", s)
		}
	}
	got, err := Load(path, valid)
	if err != nil || !reflect.DeepEqual(got, valid) {
		t.Fatal("failed save changed persisted settings", err)
	}
}

func TestTemporaryRevealDurationBoundsAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "menu-bar.json")
	for _, seconds := range []float64{5, 15, 60} {
		settings := testSettings()
		settings.TemporaryRevealSeconds = seconds
		if err := Save(path, settings); err != nil {
			t.Fatal(err)
		}
		loaded, err := Load(path, testSettings())
		if err != nil || loaded.TemporaryRevealSeconds != seconds {
			t.Fatalf("duration %v: %#v %v", seconds, loaded, err)
		}
	}
}
