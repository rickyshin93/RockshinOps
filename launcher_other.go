//go:build !darwin

package main

import "errors"

func spotlightHoldsShortcut(string) bool { return false }

// OpenShortcutSettings opens the Keyboard settings, where Spotlight's shortcut is turned off.
func (s *System) OpenShortcutSettings() error {
	return errors.New("shortcut settings are available on macOS only")
}
