package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"localops/internal/localization"
	"localops/internal/metrics"
	"localops/internal/storage"
)

type Preferences struct {
	Language     string   `json:"language"`
	MenuBarItems []string `json:"menuBarItems"`
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

func readPreferences(path string, defaults []string, defaultLanguage string) (Preferences, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Preferences{Language: defaultLanguage, MenuBarItems: append([]string{}, defaults...)}, nil
	}
	if err != nil {
		return Preferences{}, fmt.Errorf("read preferences: %w", err)
	}
	p := Preferences{Language: defaultLanguage}
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
	return Preferences{Language: s.preferences.Language, MenuBarItems: append([]string{}, s.preferences.MenuBarItems...)}
}

func (s *System) SavePreferences(p Preferences) (Preferences, error) {
	s.mu.Lock()
	if err := writePreferences(s.preferencesPath, p); err != nil {
		s.mu.Unlock()
		return Preferences{}, err
	}
	s.preferences = Preferences{Language: p.Language, MenuBarItems: append([]string{}, p.MenuBarItems...)}
	s.update(s.monitor.Current())
	s.mu.Unlock()
	applyLanguage(p.Language)
	return p, nil
}

func menuBarTitle(p Preferences, s metrics.Snapshot) string {
	parts := make([]string, 0, len(p.MenuBarItems))
	for _, item := range p.MenuBarItems {
		value := "—"
		switch item {
		case "disk":
			if s.Disk.CapacityError == "" && s.Disk.Volume.Total > 0 {
				value = storage.FormatCapacity(s.Disk.Volume.Available)
			}
			if len(p.MenuBarItems) > 1 {
				value = localization.T("磁盘 ") + value
			}
		case "cpu":
			if s.CPU.Ready && s.CPU.Error == "" {
				value = fmt.Sprintf("%.0f%%", s.CPU.Usage)
			}
			value = "CPU " + value
		case "memory":
			if s.Memory.Error == "" && s.Memory.Total > 0 {
				value = fmt.Sprintf("%.0f%%", float64(s.Memory.Used)/float64(s.Memory.Total)*100)
			}
			value = localization.T("内存 ") + value
		case "gpu":
			if s.GPU.Available && s.GPU.Error == "" {
				value = fmt.Sprintf("%.0f%%", s.GPU.Usage)
			}
			value = "GPU " + value
		case "network":
			if s.Network.Ready && s.Network.Error == "" {
				value = "↓" + menuBarRate(s.Network.Download) + " ↑" + menuBarRate(s.Network.Upload)
			}
			value = localization.T("网速 ") + value
		case "battery":
			if s.Battery.Present && s.Battery.Error == "" {
				value = fmt.Sprintf("%.0f%%", s.Battery.Percent)
			}
			value = localization.T("电池 ") + value
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, " · ")
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
