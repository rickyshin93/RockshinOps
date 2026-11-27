package main

import (
	"os"
	"path/filepath"
	"syscall"
)

const trashReadFlags = os.O_RDONLY

func removeTrashDirectory(root *os.Root, name string) error {
	path, err := syscall.UTF16PtrFromString(filepath.Join(root.Name(), name))
	if err != nil {
		return err
	}
	return syscall.RemoveDirectory(path)
}
