package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/egoist/mygo"
	"rockshinops/internal/localization"
)

type StartupSettings struct {
	Enabled   bool   `json:"enabled"`
	Available bool   `json:"available"`
	Error     string `json:"error"`
}

// The OS owns the setting. The marker only records that the first-run default
// was attempted, so an OS-level opt-out is never overwritten on later launches.
type startupSettings struct {
	mu        sync.Mutex
	marker    string
	dev       bool
	read      func() bool
	set       func(bool) error
	lastError string
}

func (s *startupSettings) initialize(enabled bool) {
	if s.dev {
		return
	}
	fresh, err := s.markDefault()
	if err != nil {
		s.lastError = err.Error()
		return
	}
	if fresh && s.read() != enabled {
		s.apply(enabled)
	}
}

func (s *startupSettings) markDefault() (bool, error) {
	if err := os.MkdirAll(filepath.Dir(s.marker), 0700); err != nil {
		return false, err
	}
	file, err := os.OpenFile(s.marker, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, file.Close()
}

func (s *startupSettings) status() StartupSettings {
	if s.dev {
		return StartupSettings{Error: localization.T("请在安装后的应用中设置开机自启动。")}
	}
	return StartupSettings{Enabled: s.read(), Available: true, Error: s.lastError}
}

func (s *System) StartupSettings() StartupSettings {
	s.startup.mu.Lock()
	defer s.startup.mu.Unlock()
	return s.startup.status()
}

func (s *System) SetOpenAtLogin(enabled bool) StartupSettings {
	s.startup.mu.Lock()
	defer s.startup.mu.Unlock()
	if s.startup.dev {
		return s.startup.status()
	}
	if _, err := s.startup.markDefault(); err != nil {
		s.startup.lastError = err.Error()
		return s.startup.status()
	}
	s.startup.apply(enabled)
	return s.startup.status()
}

func (s *startupSettings) apply(enabled bool) {
	s.lastError = ""
	if err := s.set(enabled); err != nil {
		s.lastError = err.Error()
	} else if s.read() != enabled {
		s.lastError = localization.T("系统尚未应用开机自启动设置，请检查系统的登录项权限后重试。")
	}
}

func newStartupSettings(directory string) *startupSettings {
	return &startupSettings{
		marker: filepath.Join(directory, "login-default-applied"), dev: mygo.IsDev(),
		read: mygo.App.OpenAtLogin, set: func(enabled bool) error {
			if err := mygo.App.SetOpenAtLogin(enabled); err != nil {
				return fmt.Errorf("%s %w", localization.T("无法设置开机自启动："), err)
			}
			return nil
		},
	}
}
