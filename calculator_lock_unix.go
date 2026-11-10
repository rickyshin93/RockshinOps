//go:build darwin || linux

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockCalculatorFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX)
}
