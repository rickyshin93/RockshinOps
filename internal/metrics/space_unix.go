//go:build unix

package metrics

import (
	"os"
	"syscall"
)

type FileID struct{ Device, Inode uint64 }

func FileIdentity(info os.FileInfo) (FileID, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return FileID{}, false
	}
	return FileID{Device: uint64(stat.Dev), Inode: uint64(stat.Ino)}, true
}

func linkCount(info os.FileInfo) uint64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(stat.Nlink)
	}
	return 1
}

// allocatedBytes counts blocks on disk, so sparse and compressed files are not inflated.
func allocatedBytes(info os.FileInfo) int64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return int64(stat.Blocks) * 512
	}
	return info.Size()
}
