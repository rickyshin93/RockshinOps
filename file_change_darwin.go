package main

import (
	"io/fs"
	"syscall"
)

func fileChangeTime(info fs.FileInfo) int64 {
	return info.Sys().(*syscall.Stat_t).Ctimespec.Nano()
}
