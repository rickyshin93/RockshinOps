//go:build !darwin

package main

import (
	"context"
	"fmt"
)

func (s *System) InstalledApps() ([]InstalledApp, error) {
	return nil, fmt.Errorf("app uninstall is available on macOS only")
}
func (s *System) ReviewApp(context.Context, string, uint64) (AppReview, error) {
	return AppReview{}, fmt.Errorf("app uninstall is available on macOS only")
}
func (s *System) CancelAppReview(uint64) error {
	return fmt.Errorf("app uninstall is available on macOS only")
}
func (s *System) ReviewAppQuick(string) (AppReview, error) {
	return AppReview{}, fmt.Errorf("app uninstall is available on macOS only")
}
func (s *System) RemoveAppFiles([]string) error {
	return fmt.Errorf("app uninstall is available on macOS only")
}
