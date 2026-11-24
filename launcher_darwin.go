package main

/*
#cgo LDFLAGS: -framework AppKit
#include <stdint.h>
#include <stdlib.h>

void rockshin_launcher_open_app(const char *path, uintptr_t token);
*/
import "C"

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/cgo"
	"unsafe"

	"github.com/egoist/mygo"
)

func (l *launcher) launchApp(app mygo.InstalledApp) error {
	panel := l.panel.Load()
	var version uint64
	var started, wasVisible bool
	result := make(chan error, 1)
	handle := cgo.NewHandle(result)
	path := C.CString(app.Path)
	defer C.free(unsafe.Pointer(path))
	mygo.RunOnMain(func() {
		started = true
		version = l.panelVersion
		wasVisible = panel != nil && panel.IsVisible()
		if wasVisible {
			panel.Hide()
		}
		C.rockshin_launcher_open_app(path, C.uintptr_t(handle))
	})
	if !started {
		handle.Delete()
		return errors.New("launcher is shutting down")
	}
	err := <-result
	if err != nil && wasVisible {
		mygo.RunOnMain(func() {
			// Leave a later invocation in charge if the user reopened the launcher.
			if l.panelVersion == version {
				panel.Show()
				panel.Focus()
			}
		})
	}
	return err
}

//export rockshin_launcher_open_app_completed
func rockshin_launcher_open_app_completed(token C.uintptr_t, message *C.char) {
	handle := cgo.Handle(token)
	result := handle.Value().(chan error)
	handle.Delete()
	var err error
	if message != nil {
		err = errors.New(C.GoString(message))
	}
	result <- err
}

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
