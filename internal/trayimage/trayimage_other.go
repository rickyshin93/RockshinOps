//go:build !darwin

package trayimage

import "errors"

// renderNative is only available on macOS; other platforms keep the text title.
func renderNative([]Item) ([]byte, error) { return nil, errors.ErrUnsupported }
