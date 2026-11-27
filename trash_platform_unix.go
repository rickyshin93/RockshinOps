//go:build unix

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

const trashReadFlags = os.O_RDONLY | unix.O_NOFOLLOW | unix.O_NONBLOCK

func removeTrashDirectory(root *os.Root, name string) error {
	parent, err := root.Open(".")
	if err != nil {
		return err
	}
	defer parent.Close()
	return unix.Unlinkat(int(parent.Fd()), name, unix.AT_REMOVEDIR)
}
