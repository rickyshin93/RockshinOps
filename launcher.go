package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"rockshinops/internal/fileindex"
	"rockshinops/internal/localization"
)

// LauncherShown tells the launcher page that the panel was summoned.
var LauncherShown = mygo.NewEvent[bool]("launcherShown")

const (
	launcherWidth     = 680
	launcherMaxHeight = 600
	launcherResults   = 9
	appBoost          = 100 // apps outrank files that match as well
)

type LauncherSettings struct {
	Enabled           bool   `json:"enabled"`
	Files             bool   `json:"files"`
	Shortcut          string `json:"shortcut"`
	SpotlightConflict bool   `json:"spotlightConflict"`
	FileCount         int    `json:"fileCount"`
	Indexing          bool   `json:"indexing"`
	Error             string `json:"error"`
}

type LauncherResult struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"` // app, file or folder
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Path     string `json:"path"`
}

type launcherPrefs struct {
	Enabled  bool   `json:"enabled"`
	Files    bool   `json:"files"`
	Shortcut string `json:"shortcut"`
}

type launcher struct {
	mu         sync.Mutex
	path       string
	prefs      launcherPrefs
	active     string // the registered accelerator, "" when none
	lastError  string
	register   func(string) error
	unregister func(string)

	panelOnce sync.Once
	panel     atomic.Pointer[mygo.Window]
	appsMu    sync.Mutex
	apps      []mygo.InstalledApp

	home      string // "" leaves file search off, as in tests
	rules     fileindex.Rules
	rebuild   time.Duration
	index     atomic.Pointer[fileindex.Index]
	indexing  atomic.Bool
	stopFiles context.CancelFunc
	shownMu   sync.Mutex
	shown     map[string]bool // files the last query returned, which open may act on
}

func readLauncherPrefs(path string, defaults launcherPrefs) (launcherPrefs, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return launcherPrefs{}, fmt.Errorf("read launcher settings: %w", err)
	}
	p := defaults
	if err := json.Unmarshal(data, &p); err != nil || p.Shortcut == "" {
		return launcherPrefs{}, fmt.Errorf("invalid launcher settings: %s", path)
	}
	return p, nil
}

func writeLauncherPrefs(path string, p launcherPrefs) error {
	if p.Shortcut == "" {
		return errors.New("launcher shortcut is required")
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create launcher settings directory: %w", err)
	}
	if err := os.WriteFile(path+".tmp", append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("write launcher settings: %w", err)
	}
	return os.Rename(path+".tmp", path)
}

// apply registers the shortcut p asks for before releasing the old one, so
// a taken shortcut leaves the working one in place, then starts or stops
// file indexing. Call with l.mu held.
func (l *launcher) apply(p launcherPrefs) error {
	want := ""
	if p.Enabled {
		want = p.Shortcut
	}
	if want != l.active {
		if want != "" {
			if err := l.register(want); err != nil {
				return fmt.Errorf("%s%s (%v)", localization.T("无法注册快捷键："), want, err)
			}
		}
		if l.active != "" {
			l.unregister(l.active)
		}
		l.active = want
	}
	switch files := p.Enabled && p.Files && l.home != ""; {
	case files && l.stopFiles == nil:
		ctx, cancel := context.WithCancel(context.Background())
		l.stopFiles = cancel
		go l.indexFiles(ctx)
	case !files && l.stopFiles != nil:
		l.stopFiles()
		l.stopFiles = nil
		l.index.Store(nil) // free the names
	}
	return nil
}

func (l *launcher) set(p launcherPrefs) (launcherPrefs, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if p.Shortcut == "" {
		return l.prefs, errors.New("launcher shortcut is required")
	}
	old := l.prefs
	if err := l.apply(p); err != nil {
		return old, err
	}
	if err := writeLauncherPrefs(l.path, p); err != nil {
		if rollback := l.apply(old); rollback != nil {
			log.Printf("restore launcher shortcut: %v", rollback)
		}
		return old, err
	}
	l.prefs, l.lastError = p, ""
	return p, nil
}

func (l *launcher) settings() LauncherSettings {
	l.mu.Lock()
	defer l.mu.Unlock()
	s := LauncherSettings{Enabled: l.prefs.Enabled, Files: l.prefs.Files, Shortcut: l.prefs.Shortcut, SpotlightConflict: spotlightHoldsShortcut(l.prefs.Shortcut), Indexing: l.indexing.Load(), Error: l.lastError}
	if ix := l.index.Load(); ix != nil {
		s.FileCount = ix.Len()
	}
	return s
}

// start applies the saved settings once the app is ready; a shortcut taken
// by another app is reported in Settings rather than stopping the app.
func (l *launcher) start() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.apply(l.prefs); err != nil {
		l.lastError = err.Error()
	}
}

func newLauncher(path string, defaults launcherPrefs, home string, rules fileindex.Rules, rebuild time.Duration) (*launcher, error) {
	p, err := readLauncherPrefs(path, defaults)
	if err != nil {
		return nil, err
	}
	l := &launcher{path: path, prefs: p, unregister: mygo.GlobalShortcut.Unregister, home: home, rules: rules, rebuild: rebuild}
	l.register = func(acc string) error {
		l.ensurePanel()
		return mygo.GlobalShortcut.Register(acc, l.toggle)
	}
	return l, nil
}

// ensurePanel creates the hidden panel the first time the launcher is
// enabled, so that it shows at once on the shortcut.
func (l *launcher) ensurePanel() {
	l.panelOnce.Do(func() {
		panel := mygo.NewWindow(mygo.WindowOptions{
			URL: "/launcher.html", Width: launcherWidth, Height: 72,
			Panel: true, Hidden: true, Frameless: true, Transparent: true, DisableResize: true,
		})
		panel.SetHasShadow(false) // the page draws the card's shadow; the window grows with results
		quitting := false         // both callbacks run on MyGo's main thread
		mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })
		panel.OnClose(func(e *mygo.CloseEvent) {
			if !quitting {
				e.PreventDefault()
				panel.Hide()
			}
		})
		panel.OnBlur(panel.Hide)
		l.panel.Store(panel)
		go l.loadApps()
	})
}

// toggle runs on the main thread from the global shortcut.
func (l *launcher) toggle() {
	panel := l.panel.Load()
	if panel.IsVisible() {
		panel.Hide()
		return
	}
	d := mygo.Screen.DisplayNearestPoint(mygo.Screen.CursorScreenPoint())
	w, _ := panel.Size()
	panel.SetPosition(d.WorkArea.X+(d.WorkArea.Width-w)/2, d.WorkArea.Y+d.WorkArea.Height/5)
	panel.Show()
	panel.Focus()
	if err := LauncherShown.Broadcast(true); err != nil {
		log.Printf("launcher shown: %v", err)
	}
	go l.loadApps()
}

func (l *launcher) loadApps() {
	apps, err := mygo.Shell.InstalledApps()
	if err != nil {
		log.Printf("launcher apps: %v", err)
		return
	}
	l.appsMu.Lock()
	l.apps = apps
	l.appsMu.Unlock()
}

type scoredResult struct {
	LauncherResult
	score int
}

// query ranks apps and, from two letters on, files together; apps win ties
// by appBoost.
func (l *launcher) query(q string) []LauncherResult {
	l.appsMu.Lock()
	apps := l.apps
	l.appsMu.Unlock()
	var found []scoredResult
	for _, a := range rankApps(apps, q, launcherResults) {
		found = append(found, scoredResult{LauncherResult{ID: "app:" + a.app.Path, Kind: "app", Title: a.app.Name, Subtitle: a.app.Path, Path: a.app.Path}, a.score + appBoost})
	}
	shown := map[string]bool{}
	if ix := l.index.Load(); ix != nil && utf8.RuneCountInString(strings.TrimSpace(q)) >= 2 {
		for _, m := range ix.Search(q, launcherResults) {
			kind := "file"
			if m.Dir {
				kind = "folder"
			}
			shown[m.Path] = true
			found = append(found, scoredResult{LauncherResult{ID: "file:" + m.Path, Kind: kind, Title: m.Name, Subtitle: l.shortPath(filepath.Dir(m.Path)), Path: m.Path}, m.Score})
		}
	}
	l.shownMu.Lock()
	l.shown = shown
	l.shownMu.Unlock()
	slices.SortStableFunc(found, func(a, b scoredResult) int { return cmp.Compare(b.score, a.score) })
	results := make([]LauncherResult, 0, min(len(found), launcherResults))
	for _, f := range found[:min(len(found), launcherResults)] {
		results = append(results, f.LauncherResult)
	}
	return results
}

// shortPath writes the home folder as ~.
func (l *launcher) shortPath(p string) string {
	if l.home != "" {
		if rest, ok := strings.CutPrefix(p, l.home); ok && (rest == "" || rest[0] == filepath.Separator) {
			return "~" + rest
		}
	}
	return p
}

// open acts on a result the launcher returned: open it, show it in the
// file manager, or copy its path.
func (l *launcher) open(id, action string) error {
	var app *mygo.InstalledApp
	path, isApp := strings.CutPrefix(id, "app:")
	if isApp {
		l.appsMu.Lock()
		if i := slices.IndexFunc(l.apps, func(a mygo.InstalledApp) bool { return a.Path == path }); i >= 0 {
			app = &l.apps[i]
		}
		l.appsMu.Unlock()
	} else if p, ok := strings.CutPrefix(id, "file:"); ok {
		l.shownMu.Lock()
		if l.shown[p] {
			path = p
		} else {
			path = ""
		}
		l.shownMu.Unlock()
	} else {
		path = ""
	}
	if path == "" || isApp && app == nil {
		return fmt.Errorf("unknown launcher item: %s", id)
	}
	// Hide first so the app the user was in keeps the keyboard until the opened one takes it.
	if panel := l.panel.Load(); panel != nil {
		panel.Hide()
	}
	switch action {
	case "reveal":
		mygo.Shell.ShowItemInFolder(path)
		return nil
	case "copy":
		mygo.Clipboard.WriteText(path)
		return nil
	case "open":
		if app != nil {
			return mygo.Shell.LaunchApp(*app)
		}
		return mygo.Shell.OpenPath(path)
	}
	return fmt.Errorf("unknown launcher action: %s", action)
}

type scoredApp struct {
	app   mygo.InstalledApp
	score int
}

// rankApps orders the apps matching q by how well their shown name, or the
// name of their bundle file (English for a localized system app), matches.
func rankApps(apps []mygo.InstalledApp, q string, limit int) []scoredApp {
	lq := []byte(strings.ToLower(strings.TrimSpace(q)))
	if len(lq) == 0 {
		return nil
	}
	var found []scoredApp
	for _, a := range apps {
		file := strings.TrimSuffix(filepath.Base(a.Path), filepath.Ext(a.Path))
		s := max(fileindex.ScoreText(lq, a.Name), fileindex.ScoreText(lq, file))
		if s > 0 {
			found = append(found, scoredApp{a, s})
		}
	}
	slices.SortStableFunc(found, func(a, b scoredApp) int {
		return cmp.Or(cmp.Compare(b.score, a.score), strings.Compare(strings.ToLower(a.app.Name), strings.ToLower(b.app.Name)))
	})
	return found[:min(limit, len(found))]
}

func (s *System) LauncherSettings() LauncherSettings { return s.launcher.settings() }

func (s *System) SetLauncher(enabled, files bool, shortcut string) (LauncherSettings, error) {
	if _, err := s.launcher.set(launcherPrefs{Enabled: enabled, Files: files, Shortcut: shortcut}); err != nil {
		return s.launcher.settings(), err
	}
	return s.launcher.settings(), nil
}

func (s *System) LauncherQuery(query string) []LauncherResult { return s.launcher.query(query) }

// LauncherOpen acts on a result: action is "open", "reveal" or "copy".
func (s *System) LauncherOpen(id, action string) error { return s.launcher.open(id, action) }

// LauncherResize fits the panel to the page's card, growing downwards.
func (s *System) LauncherResize(height int) {
	if p := s.launcher.panel.Load(); p != nil {
		p.SetSize(launcherWidth, min(max(height, 60), launcherMaxHeight))
	}
}

func (s *System) LauncherHide() {
	if p := s.launcher.panel.Load(); p != nil {
		p.Hide()
	}
}
