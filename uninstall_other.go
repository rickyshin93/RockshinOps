//go:build !darwin

package main

import (
	"context"
	"fmt"
)

func (s *System) InstalledApps() ([]InstalledApp, error) {
	return nil, fmt.Errorf("app uninstall is available on macOS only")
}
func (s *System) ReviewApp(ctx context.Context, path string, id uint64) (AppReview, error) {
	return AppReview{}, fmt.Errorf("app uninstall is available on macOS only")
}
func (s *System) CancelAppReview(id uint64) error {
	return fmt.Errorf("app uninstall is available on macOS only")
}
func (s *System) ReviewAppQuick(ctx context.Context, path string) (AppReview, error) {
	return AppReview{}, fmt.Errorf("app uninstall is available on macOS only")
}
func (s *System) RemoveAppFiles(ctx context.Context, id uint64, paths []string) (FileOutcome, error) {
	return FileOutcome{}, fmt.Errorf("app uninstall is available on macOS only")
}
