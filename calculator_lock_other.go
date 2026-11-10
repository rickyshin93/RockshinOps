//go:build !darwin && !linux && !windows

package main

import (
	"errors"
	"os"
)

func lockCalculatorFile(*os.File) error {
	return errors.New("calculator storage locking is unavailable on this platform")
}
