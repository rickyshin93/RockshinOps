//go:build !darwin

package main

import "errors"

func spotlightHoldsShortcut(string) bool { return false }

func (s *System) OpenShortcutSettings() error {
	return errors.New("shortcut settings are available on macOS only")
}
