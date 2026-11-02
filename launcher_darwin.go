package main

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/egoist/mygo"
)

func spotlightHoldsShortcut(shortcut string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	plist := filepath.Join(home, "Library", "Preferences", "com.apple.symbolichotkeys.plist")
	data, err := exec.Command("/usr/bin/plutil", "-extract", "AppleSymbolicHotKeys.64", "json", "-o", "-", plist).Output()
	return spotlightConflict(shortcut, data, err)
}

// OpenShortcutSettings opens the Keyboard settings, where Spotlight's shortcut is turned off.
func (s *System) OpenShortcutSettings() error {
	return mygo.Shell.OpenExternal("x-apple.systempreferences:com.apple.preference.keyboard?Shortcuts")
}
