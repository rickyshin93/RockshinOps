package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"localops/internal/menubar"
	"localops/internal/metrics"
)

//go:embed localops.json
var configJSON []byte

type Config struct {
	MenuBarManager              menubar.Settings                `json:"menuBarManager"`
	DefaultHideDockOnClose      bool                            `json:"defaultHideDockOnClose"`
	DefaultUpdateUseSystemProxy bool                            `json:"defaultUpdateUseSystemProxy"`
	DefaultLanguage             string                          `json:"defaultLanguage"`
	UpdateIntervalSeconds       int                             `json:"updateIntervalSeconds"`
	DefaultMenuBarItems         []string                        `json:"defaultMenuBarItems"`
	SampleIntervalSeconds       int                             `json:"sampleIntervalSeconds"`
	MetricsIntervalSeconds      int                             `json:"metricsIntervalSeconds"`
	ProcessIntervalSeconds      int                             `json:"processIntervalSeconds"`
	HistorySeconds              int                             `json:"historySeconds"`
	ConnectionTimeoutSeconds    int                             `json:"connectionTimeoutSeconds"`
	MachineInfoTimeoutSeconds   int                             `json:"machineInfoTimeoutSeconds"`
	ScanTimeoutSeconds          int                             `json:"scanTimeoutSeconds"`
	ScanMaxEntries              int                             `json:"scanMaxEntries"`
	ScanMaxItems                int                             `json:"scanMaxItems"`
	ScanTargets                 map[string][]metrics.ScanTarget `json:"scanTargets"`
}

func parseConfig(data []byte) (Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("invalid localops.json: %w", err)
	}
	if err := c.MenuBarManager.Validate(); err != nil {
		return c, fmt.Errorf("localops.json: %w", err)
	}
	if c.SampleIntervalSeconds <= 0 {
		return c, fmt.Errorf("localops.json: sampleIntervalSeconds must be positive")
	}
	if c.UpdateIntervalSeconds <= 0 {
		return c, fmt.Errorf("localops.json: updateIntervalSeconds must be positive")
	}
	if c.MachineInfoTimeoutSeconds <= 0 {
		return c, fmt.Errorf("localops.json: machineInfoTimeoutSeconds must be positive")
	}
	if c.MetricsIntervalSeconds <= 0 || c.ProcessIntervalSeconds <= 0 || c.HistorySeconds < 3600 || c.ConnectionTimeoutSeconds <= 0 || c.ScanTimeoutSeconds <= 0 || c.ScanMaxEntries <= 0 || c.ScanMaxItems <= 0 {
		return c, fmt.Errorf("localops.json: monitoring and scan parameters must be positive; historySeconds must be at least 3600")
	}
	if len(c.ScanTargets) == 0 {
		return c, fmt.Errorf("localops.json: scanTargets is required")
	}
	if err := validatePreferences(Preferences{Language: c.DefaultLanguage, MenuBarItems: c.DefaultMenuBarItems}); err != nil {
		return c, fmt.Errorf("localops.json: %w", err)
	}
	for _, platform := range []string{"darwin", "windows", "linux"} {
		targets := c.ScanTargets[platform]
		if len(targets) == 0 {
			return c, fmt.Errorf("localops.json: scanTargets.%s is required", platform)
		}
		for _, target := range targets {
			if target.Name == "" || !(filepath.IsAbs(target.Path) || strings.HasPrefix(target.Path, "~/")) {
				return c, fmt.Errorf("localops.json: each scan target requires a name and an absolute or ~/ path")
			}
		}
	}

	return c, nil
}
