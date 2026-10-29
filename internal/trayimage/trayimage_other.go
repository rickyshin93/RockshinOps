//go:build !darwin

package trayimage

import "errors"

// Render is only available on macOS; other platforms keep the text title.
func Render([]Item) ([]byte, error) { return nil, errors.ErrUnsupported }
