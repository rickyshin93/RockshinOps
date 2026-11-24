//go:build !darwin

package main

import "errors"

func clipboardChangeCount() int64 { return 0 }
func readClipboardChange(previous int64, _, _ int) clipboardChange {
	return clipboardChange{count: previous}
}
func writeClipboardContent(clipboardContent) error      { return errors.ErrUnsupported }
func clipboardImagePreview(string, int) ([]byte, error) { return nil, errors.ErrUnsupported }
