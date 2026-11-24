//go:build !darwin

package main

import (
	"errors"

	"github.com/egoist/mygo"
)

func (l *launcher) launchApp(app mygo.InstalledApp) error {
	if err := mygo.Shell.LaunchApp(app); err != nil {
		return err
	}
	l.hide()
	return nil
}

func spotlightHoldsShortcut(string) bool { return false }

// OpenShortcutSettings opens the Keyboard settings, where Spotlight's shortcut is turned off.
func (s *System) OpenShortcutSettings() error {
	return errors.New("shortcut settings are available on macOS only")
}
