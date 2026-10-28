package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// emptyTrash removes entries, leaving the trash directory itself in place.
func emptyTrash(path, platform string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("trash path is not a directory")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	var failures []error
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(path, entry.Name())); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", entry.Name(), err))
			continue
		}
		if platform == "linux" {
			if err := os.Remove(filepath.Join(filepath.Dir(path), "info", entry.Name()+".trashinfo")); err != nil && !os.IsNotExist(err) {
				failures = append(failures, fmt.Errorf("%s metadata: %w", entry.Name(), err))
			}
		}
	}
	return errors.Join(failures...)
}
