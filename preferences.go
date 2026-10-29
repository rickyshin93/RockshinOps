package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"rockshinops/internal/localization"
	"rockshinops/internal/metrics"
	"rockshinops/internal/storage"
	"rockshinops/internal/trayimage"
)

type Preferences struct {
	HideDockOnClose      bool     `json:"hideDockOnClose"`
	UpdateUseSystemProxy bool     `json:"updateUseSystemProxy"`
	Language             string   `json:"language"`
	MenuBarItems         []string `json:"menuBarItems"`
}

// Copy only the two settings files from the former app directory. Keep the
// originals for older installations and never replace newer saved settings.
func migrateConfigDirectory(root string) (string, error) {
	directory := filepath.Join(root, "RockshinOps")
	for _, name := range []string{"preferences.json", "menu-bar.json"} {
		target := filepath.Join(directory, name)
		if _, err := os.Stat(target); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect settings %s: %w", name, err)
		}
		data, err := os.ReadFile(filepath.Join(root, "LocalOps", name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("read legacy settings %s: %w", name, err)
		}
		if !json.Valid(data) {
			return "", fmt.Errorf("migrate settings %s: invalid JSON", name)
		}
		if err := os.MkdirAll(directory, 0700); err != nil {
			return "", fmt.Errorf("create settings directory: %w", err)
		}
		if err := os.WriteFile(target+".tmp", data, 0600); err != nil {
			return "", fmt.Errorf("copy settings %s: %w", name, err)
		}
		if err := os.Rename(target+".tmp", target); err != nil {
			return "", fmt.Errorf("install settings %s: %w", name, err)
		}
	}
	return directory, nil
}

func validatePreferences(p Preferences) error {
	switch p.Language {
	case "system", "en", "zh":
	default:
		return fmt.Errorf("unknown language: %q", p.Language)
	}
	if p.MenuBarItems == nil {
		return fmt.Errorf("menuBarItems must be an array")
	}
	seen := map[string]bool{}
	for _, item := range p.MenuBarItems {
		switch item {
		case "disk", "cpu", "memory", "gpu", "network", "battery":
		default:
			return fmt.Errorf("unknown menu-bar item: %q", item)
		}
		if seen[item] {
			return fmt.Errorf("duplicate menu-bar item: %s", item)
		}
		seen[item] = true
	}
	return nil
}

func readPreferences(path string, defaults []string, defaultLanguage string, defaultProxy, defaultHideDock bool) (Preferences, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Preferences{HideDockOnClose: defaultHideDock, UpdateUseSystemProxy: defaultProxy, Language: defaultLanguage, MenuBarItems: append([]string{}, defaults...)}, nil
	}
	if err != nil {
		return Preferences{}, fmt.Errorf("read preferences: %w", err)
	}
	p := Preferences{HideDockOnClose: defaultHideDock, UpdateUseSystemProxy: defaultProxy, Language: defaultLanguage}
	if err = json.Unmarshal(data, &p); err == nil {
		err = validatePreferences(p)
	}
	if err != nil {
		return Preferences{}, fmt.Errorf("invalid preferences: %w", err)
	}
	return p, nil
}

func writePreferences(path string, p Preferences) error {
	if err := validatePreferences(p); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create preferences directory: %w", err)
	}
	if err := os.WriteFile(path+".tmp", append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("write preferences: %w", err)
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		return fmt.Errorf("save preferences: %w", err)
	}
	return nil
}

func (s *System) Preferences() Preferences {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Preferences{HideDockOnClose: s.preferences.HideDockOnClose, UpdateUseSystemProxy: s.preferences.UpdateUseSystemProxy, Language: s.preferences.Language, MenuBarItems: append([]string{}, s.preferences.MenuBarItems...)}
}

func (s *System) SavePreferences(p Preferences) (Preferences, error) {
	s.mu.Lock()
	if err := writePreferences(s.preferencesPath, p); err != nil {
		s.mu.Unlock()
		return Preferences{}, err
	}
	s.preferences = Preferences{HideDockOnClose: p.HideDockOnClose, UpdateUseSystemProxy: p.UpdateUseSystemProxy, Language: p.Language, MenuBarItems: append([]string{}, p.MenuBarItems...)}
	s.update(s.monitor.Current())
	s.mu.Unlock()
	applyLanguage(p.Language)
	return p, nil
}

type menuBarReading struct{ label, value, symbol string }

// menuBarReadings is shared by the text title and the macOS icon image, which omits labels.
func menuBarReadings(items []string, s metrics.Snapshot) []menuBarReading {
	readings := make([]menuBarReading, 0, len(items))
	for _, item := range items {
		r := menuBarReading{value: "—"}
		switch item {
		case "disk":
			r.symbol = "internaldrive"
			if s.Disk.CapacityError == "" && s.Disk.Volume.Total > 0 {
				r.value = storage.FormatCapacity(s.Disk.Volume.Available)
			}
			if len(items) > 1 {
				r.label = localization.T("磁盘 ")
			}
		case "cpu":
			r.symbol, r.label = "cpu", "CPU "
			if s.CPU.Ready && s.CPU.Error == "" {
				r.value = fmt.Sprintf("%.0f%%", s.CPU.Usage)
			}
		case "memory":
			r.symbol, r.label = "memorychip", localization.T("内存 ")
			if s.Memory.Error == "" && s.Memory.Total > 0 {
				r.value = fmt.Sprintf("%.0f%%", float64(s.Memory.Used)/float64(s.Memory.Total)*100)
			}
		case "gpu":
			r.symbol, r.label = "cube", "GPU "
			if s.GPU.Available && s.GPU.Error == "" {
				r.value = fmt.Sprintf("%.0f%%", s.GPU.Usage)
			}
		case "network":
			r.symbol, r.label = "network", localization.T("网速 ")
			if s.Network.Ready && s.Network.Error == "" {
				r.value = "↓" + menuBarRate(s.Network.Download) + " ↑" + menuBarRate(s.Network.Upload)
			}
		case "battery":
			r.symbol, r.label = batterySymbol(s.Battery), localization.T("电池 ")
			if s.Battery.Present && s.Battery.Error == "" {
				r.value = fmt.Sprintf("%.0f%%", s.Battery.Percent)
			}
		}
		readings = append(readings, r)
	}
	return readings
}

func batterySymbol(b metrics.Battery) string {
	if b.Charging {
		return "battery.100.bolt"
	}
	if !b.Present || b.Error != "" {
		return "battery.0"
	}
	return fmt.Sprintf("battery.%.0f", math.Min(100, math.Round(b.Percent/25)*25))
}

func menuBarTitle(p Preferences, s metrics.Snapshot) string {
	parts := make([]string, 0, len(p.MenuBarItems))
	for _, r := range menuBarReadings(p.MenuBarItems, s) {
		parts = append(parts, r.label+r.value)
	}
	return strings.Join(parts, " · ")
}

// menuBarImage draws the readings as SF Symbols with values; nil where no image renderer exists.
func menuBarImage(items []string, s metrics.Snapshot) []byte {
	readings := menuBarReadings(items, s)
	tray := make([]trayimage.Item, len(readings))
	for i, r := range readings {
		tray[i] = trayimage.Item{Symbol: r.symbol, Text: r.value}
	}
	png, err := trayimage.Render(tray)
	if err != nil {
		return nil
	}
	return png
}

// MenuBarPreview is what the menu bar would show for the latest sample.
type MenuBarPreview struct {
	Image string `json:"image"`
	Text  string `json:"text"`
}

// MenuBarPreview renders items against the latest snapshot without saving them.
func (s *System) MenuBarPreview(items []string) MenuBarPreview {
	snapshot := s.monitor.Current()
	preview := MenuBarPreview{Text: menuBarTitle(Preferences{MenuBarItems: items}, snapshot)}
	if png := menuBarImage(items, snapshot); png != nil {
		preview.Image = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	}
	return preview
}

func menuBarRate(v float64) string {
	if v >= 1e9 {
		return fmt.Sprintf("%.1fGB/s", v/1e9)
	}
	if v >= 1e6 {
		return fmt.Sprintf("%.1fMB/s", v/1e6)
	}
	if v >= 1e3 {
		return fmt.Sprintf("%.0fKB/s", v/1e3)
	}
	return fmt.Sprintf("%.0fB/s", v)
}

func applyLanguage(language string) {
	if language == "system" {
		language = ""
	}
	mygo.App.SetLocale(language)
}
