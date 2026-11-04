package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"rockshinops/internal/fileindex"
	"rockshinops/internal/menubar"
	"rockshinops/internal/metrics"
)

// LauncherConfig holds the first-run launcher defaults; the user's choice is saved in launcher.json.
type LauncherConfig struct {
	DefaultEnabled     bool                       `json:"defaultEnabled"`
	DefaultFiles       bool                       `json:"defaultFiles"`
	DefaultShortcut    map[string]string          `json:"defaultShortcut"`
	Files              map[string]fileindex.Rules `json:"files"`
	FileRebuildMinutes int                        `json:"fileRebuildMinutes"`
	WeChatPaths        []string                   `json:"weChatPaths"`
}

//go:embed rockshinops.json
var configJSON []byte

type Config struct {
	DefaultOpenAtLogin              bool                            `json:"defaultOpenAtLogin"`
	ConnectionSampleIntervalSeconds int                             `json:"connectionSampleIntervalSeconds"`
	MenuBarManager                  menubar.Settings                `json:"menuBarManager"`
	DefaultHideDockOnClose          bool                            `json:"defaultHideDockOnClose"`
	DefaultUpdateUseSystemProxy     bool                            `json:"defaultUpdateUseSystemProxy"`
	DefaultLanguage                 string                          `json:"defaultLanguage"`
	UpdateIntervalSeconds           int                             `json:"updateIntervalSeconds"`
	DefaultMenuBarItems             []string                        `json:"defaultMenuBarItems"`
	SampleIntervalSeconds           int                             `json:"sampleIntervalSeconds"`
	MetricsIntervalSeconds          int                             `json:"metricsIntervalSeconds"`
	ProcessIntervalSeconds          int                             `json:"processIntervalSeconds"`
	HistorySeconds                  int                             `json:"historySeconds"`
	ConnectionTimeoutSeconds        int                             `json:"connectionTimeoutSeconds"`
	MachineInfoTimeoutSeconds       int                             `json:"machineInfoTimeoutSeconds"`
	PowerAssertionIntervalSeconds   int                             `json:"powerAssertionIntervalSeconds"`
	ScanTargets                     map[string][]metrics.ScanTarget `json:"scanTargets"`
	Launcher                        LauncherConfig                  `json:"launcher"`
}

func parseConfig(data []byte) (Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("invalid rockshinops.json: %w", err)
	}
	if err := c.MenuBarManager.Validate(); err != nil {
		return c, fmt.Errorf("rockshinops.json: %w", err)
	}
	if c.ConnectionSampleIntervalSeconds <= 0 || c.ConnectionSampleIntervalSeconds >= c.ConnectionTimeoutSeconds {
		return c, fmt.Errorf("rockshinops.json: connectionSampleIntervalSeconds must be positive and less than connectionTimeoutSeconds")
	}
	if c.SampleIntervalSeconds <= 0 {
		return c, fmt.Errorf("rockshinops.json: sampleIntervalSeconds must be positive")
	}
	if c.UpdateIntervalSeconds <= 0 {
		return c, fmt.Errorf("rockshinops.json: updateIntervalSeconds must be positive")
	}
	if c.MachineInfoTimeoutSeconds <= 0 {
		return c, fmt.Errorf("rockshinops.json: machineInfoTimeoutSeconds must be positive")
	}
	if c.MetricsIntervalSeconds <= 0 || c.ProcessIntervalSeconds <= 0 || c.HistorySeconds < 3600 || c.ConnectionTimeoutSeconds <= 0 {
		return c, fmt.Errorf("rockshinops.json: monitoring parameters must be positive; historySeconds must be at least 3600")
	}
	if c.PowerAssertionIntervalSeconds <= 0 {
		return c, fmt.Errorf("rockshinops.json: powerAssertionIntervalSeconds must be positive")
	}
	if c.Launcher.FileRebuildMinutes <= 0 {
		return c, fmt.Errorf("rockshinops.json: launcher.fileRebuildMinutes must be positive")
	}
	if len(c.ScanTargets) == 0 {
		return c, fmt.Errorf("rockshinops.json: scanTargets is required")
	}
	if err := validatePreferences(Preferences{Language: c.DefaultLanguage, MenuBarItems: c.DefaultMenuBarItems}); err != nil {
		return c, fmt.Errorf("rockshinops.json: %w", err)
	}
	for _, platform := range []string{"darwin", "windows", "linux"} {
		if c.Launcher.DefaultShortcut[platform] == "" {
			return c, fmt.Errorf("rockshinops.json: launcher.defaultShortcut.%s is required", platform)
		}
		if _, ok := c.Launcher.Files[platform]; !ok {
			return c, fmt.Errorf("rockshinops.json: launcher.files.%s is required", platform)
		}
		targets := c.ScanTargets[platform]
		if len(targets) == 0 {
			return c, fmt.Errorf("rockshinops.json: scanTargets.%s is required", platform)
		}
		for _, target := range targets {
			if target.Name == "" || !(filepath.IsAbs(target.Path) || strings.HasPrefix(target.Path, "~/")) {
				return c, fmt.Errorf("rockshinops.json: each scan target requires a name and an absolute or ~/ path")
			}
		}
	}

	return c, nil
}
