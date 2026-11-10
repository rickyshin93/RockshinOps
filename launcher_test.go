package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"rockshinops/internal/fileindex"
)

func TestLauncherAppRanking(t *testing.T) {
	apps := []mygo.InstalledApp{
		{Name: "Visual Studio Code", Path: "/Applications/Visual Studio Code.app"},
		{Name: "Safari", Path: "/Applications/Safari.app"},
		{Name: "System Settings", Path: "/System/Applications/System Settings.app"},
		{Name: "计算器", Path: "/System/Applications/Calculator.app"},
		{Name: "Calendar", Path: "/System/Applications/Calendar.app"},
		{Name: "Activity Monitor", Path: "/System/Applications/Utilities/Activity Monitor.app"},
		{Name: "TextEdit", Path: "/System/Applications/TextEdit.app"},
	}
	for _, c := range []struct{ query, top string }{
		{"safari", "Safari"},
		{"saf", "Safari"},
		{"vsc", "Visual Studio Code"},
		{"code", "Visual Studio Code"},
		{"set", "System Settings"},
		{"calc", "计算器"}, // localized name, matched by the bundle's file name
		{"计算", "计算器"},
		{"jsq", "计算器"},
		{"jisuan", "计算器"},
		{"cal", "Calendar"},  // shorter prefix match first
		{"edit", "TextEdit"}, // camel-case word start
		{"actmon", "Activity Monitor"},
	} {
		got := rankApps(apps, c.query, 8)
		if len(got) == 0 || got[0].app.Name != c.top {
			t.Errorf("%q: want %q first, got %v", c.query, c.top, names(got))
		}
	}
	if got := rankApps(apps, "zzz", 8); len(got) != 0 {
		t.Errorf("unrelated query matched %v", names(got))
	}
	if got := rankApps(apps, "  ", 8); len(got) != 0 {
		t.Errorf("blank query matched %v", names(got))
	}
	if got := rankApps(apps, "a", 2); len(got) != 2 {
		t.Errorf("limit ignored: %v", names(got))
	}
}

func TestLauncherCalculation(t *testing.T) {
	for _, tc := range []struct{ query, want string }{
		{"11851-64", "11787"},
		{"2 + 3 * 4", "14"},
		{"−(2 + 3) × .4 ÷ 2", "-1"},
		{"24 / 3 / 2", "4"},
		{"10 - 3 - 2", "5"},
		{"0.1 + 0.2", "0.3"},
		{"9007199254740992 + 1", "9007199254740993"},
		{"1 / 3", "0.3333"},
		{"-42", ""},
		{"1 +", ""},
		{"1 + (2", ""},
		{"1 / (2 - 2)", ""},
		{"1+2.pdf", ""},
		{"2026-11-09 report", ""},
	} {
		t.Run(tc.query, func(t *testing.T) {
			got, err := evaluateCalculation(tc.query, true)
			if got.Value != tc.want || (err == nil) != (tc.want != "") {
				t.Fatalf("evaluateCalculation(%q) = %+v, %v; want %q", tc.query, got, err, tc.want)
			}
		})
	}
}

func TestLauncherQueryPutsCalculationBeforeSearchMatches(t *testing.T) {
	s := calculatorTestSystem(t, filepath.Join(t.TempDir(), "preferences.json"))
	s.launcher = &launcher{apps: []mygo.InstalledApp{
		{Name: "11851-64", Path: "/Applications/11851-64.app"},
		{Name: "1+2.pdf", Path: "/Applications/1+2.pdf.app"},
	}}
	got := s.LauncherQuery("11851-64").Items
	if len(got) != 2 || got[0].Kind != "calculation" || got[0].Title != "11787" || got[1].Kind != "app" {
		t.Fatalf("expected calculation followed by the matching app, got %+v", got)
	}
	if got := s.LauncherQuery("1+2.pdf").Items; len(got) != 1 || got[0].Kind != "app" {
		t.Fatalf("expected ordinary search for a filename, got %+v", got)
	}
}

func TestLauncherSearchKeepsAppMatchesOnCalculatorFormatError(t *testing.T) {
	s := calculatorTestSystem(t, filepath.Join(t.TempDir(), "preferences.json"))
	s.launcher = &launcher{apps: []mygo.InstalledApp{{Name: "1+2", Path: "/Applications/1+2.app"}}}
	if err := os.WriteFile(s.calculatorFormatPath(), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	search := s.LauncherQuery("1+2")
	if search.CalculatorError == "" || len(search.Items) != 1 || search.Items[0].Kind != "app" {
		t.Fatalf("calculator settings failure must preserve the app match: %+v", search)
	}
}

func TestCalculatorOperatorGlyphsUseTheSameLengthLimit(t *testing.T) {
	ascii := strings.Repeat("1-", 64) + "1"
	for _, expression := range []string{ascii, strings.ReplaceAll(ascii, "-", "−")} {
		value, err := evaluateCalculation(expression, true)
		if err != nil || value.Value != "-63" {
			t.Fatalf("operator length depends on glyph: %+v, %v", value, err)
		}
	}
}

func names(apps []scoredApp) []string {
	out := make([]string, len(apps))
	for i, a := range apps {
		out[i] = a.app.Name
	}
	return out
}

func TestLauncherPrefsPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "RockshinOps", "launcher.json")
	defaults := launcherPrefs{Enabled: false, Shortcut: "Cmd+Space"}
	got, err := readLauncherPrefs(path, defaults)
	if err != nil || got != defaults {
		t.Fatalf("first run: %+v %v", got, err)
	}
	want := launcherPrefs{Enabled: true, Files: true, Shortcut: "Alt+Shift+Space"}
	if err := writeLauncherPrefs(path, want); err != nil {
		t.Fatal(err)
	}
	if got, err = readLauncherPrefs(path, defaults); err != nil || got != want {
		t.Fatalf("reload: %+v %v", got, err)
	}
	if err := writeLauncherPrefs(path, launcherPrefs{Enabled: true}); err == nil {
		t.Fatal("saved an empty shortcut")
	}
}

type fakeShortcuts struct {
	registered []string
	taken      string
}

func (f *fakeShortcuts) register(acc string) error {
	if acc == f.taken {
		return errors.New("taken")
	}
	f.registered = append(f.registered, acc)
	return nil
}

func (f *fakeShortcuts) unregister(acc string) {
	for i, r := range f.registered {
		if r == acc {
			f.registered = append(f.registered[:i], f.registered[i+1:]...)
			return
		}
	}
}

func TestLauncherShortcutSwitchKeepsOldOnFailure(t *testing.T) {
	f := &fakeShortcuts{taken: "Cmd+Space"}
	l := &launcher{path: filepath.Join(t.TempDir(), "launcher.json"), register: f.register, unregister: f.unregister}
	if _, err := l.set(launcherPrefs{Enabled: true, Shortcut: "Alt+Space"}); err != nil {
		t.Fatal(err)
	}
	if _, err := l.set(launcherPrefs{Enabled: true, Shortcut: "Cmd+Space"}); err == nil {
		t.Fatal("accepted a taken shortcut")
	}
	if !reflect.DeepEqual(f.registered, []string{"Alt+Space"}) || l.prefs.Shortcut != "Alt+Space" {
		t.Fatalf("lost the working shortcut: %v %+v", f.registered, l.prefs)
	}
	if saved, _ := readLauncherPrefs(l.path, launcherPrefs{}); saved.Shortcut != "Alt+Space" {
		t.Fatalf("saved the failed shortcut: %+v", saved)
	}
	if _, err := l.set(launcherPrefs{Enabled: false, Shortcut: "Alt+Space"}); err != nil || len(f.registered) != 0 {
		t.Fatalf("disable kept %v: %v", f.registered, err)
	}
	if _, err := l.set(launcherPrefs{Enabled: false, Shortcut: "Cmd+Space"}); err != nil {
		t.Fatalf("changing a disabled shortcut must not register it: %v", err)
	}
}

func TestSpotlightConflict(t *testing.T) {
	missing := errors.New("no value")
	for _, c := range []struct {
		shortcut string
		data     string
		err      error
		want     bool
	}{
		{"Cmd+Space", `{"enabled":true,"value":{"parameters":[65535,49,1048576]}}`, nil, true},
		{"Cmd+Space", `{"enabled":false,"value":{"parameters":[65535,49,1048576]}}`, nil, false},
		{"Cmd+Space", `{"enabled":true,"value":{"parameters":[65535,49,1572864]}}`, nil, false}, // rebound to Cmd+Option+Space
		{"Cmd+Space", ``, missing, true}, // never changed: the system default holds Cmd+Space
		{"Alt+Space", `{"enabled":true,"value":{"parameters":[65535,49,1048576]}}`, nil, false},
	} {
		if got := spotlightConflict(c.shortcut, []byte(c.data), c.err); got != c.want {
			t.Errorf("%s %s %v: got %v", c.shortcut, c.data, c.err, got)
		}
	}
}

func TestLauncherMergesAppsAndFiles(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Safari 书签导出.html", "notes/safari-bugs.md", "a.txt"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ix := fileindex.New(root, fileindex.Rules{})
	if err := ix.Build(context.Background()); err != nil {
		t.Fatal(err)
	}
	l := &launcher{home: root, apps: []mygo.InstalledApp{{Name: "Safari", Path: "/Applications/Safari.app"}}}
	l.index.Store(ix)

	got, _ := l.query("safari")
	if len(got) != 3 || got[0].Kind != "app" || got[1].Title != "safari-bugs.md" || got[1].Subtitle != filepath.Join("~", "notes") || got[2].Subtitle != "~" {
		t.Fatalf("merged results: %+v", got)
	}
	if got, _ := l.query("a"); len(got) != 1 || got[0].Kind != "app" {
		t.Fatalf("one letter searches apps only: %+v", got)
	}
	// Only results the launcher returned can be opened.
	for _, id := range []string{"file:/etc/passwd", "file:" + filepath.Join(root, "a.txt"), "app:/Applications/Evil.app", "other"} {
		if err := l.open(id, "open"); err == nil {
			t.Errorf("opened %s", id)
		}
	}
}

func TestLauncherOnlyAllowsReturnedFilesInCurrentScope(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "1+2.txt")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	ix := fileindex.New(root, fileindex.Rules{})
	if err := ix.Build(context.Background()); err != nil {
		t.Fatal(err)
	}
	s := calculatorTestSystem(t, filepath.Join(root, "preferences.json"))
	s.launcher = &launcher{home: root}
	s.launcher.index.Store(ix)
	for i := range launcherResults - 1 {
		s.launcher.apps = append(s.launcher.apps, mygo.InstalledApp{Name: "1+2", Path: filepath.Join(root, strconv.Itoa(i)+".app")})
	}
	fileID := "file:" + path
	s.LauncherQuery("1+2")
	if err := s.LauncherOpen(fileID, "fixture"); err == nil || !strings.Contains(err.Error(), "unknown launcher item") {
		t.Fatalf("trimmed result was actionable: %v", err)
	}
	s.LauncherQuery("1+2.txt")
	if err := s.LauncherOpen(fileID, "fixture"); err == nil || !strings.Contains(err.Error(), "unknown launcher action") {
		t.Fatalf("visible file failed result validation: %v", err)
	}
	s.launcher.stopIndexing()
	if err := s.LauncherOpen(fileID, "fixture"); err == nil || !strings.Contains(err.Error(), "unknown launcher item") {
		t.Fatalf("expired file result was actionable: %v", err)
	}
}
