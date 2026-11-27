//go:build !darwin && !linux

package main

import "io/fs"

func fileChangeTime(fs.FileInfo) int64 { return 0 }
