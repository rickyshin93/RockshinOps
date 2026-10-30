package metrics

import "os"

// FileID is unavailable from os.FileInfo on Windows; scans fall back to logical sizes.
type FileID struct{ Device, Inode uint64 }

func FileIdentity(os.FileInfo) (FileID, bool) { return FileID{}, false }
func linkCount(os.FileInfo) uint64            { return 1 }
func allocatedBytes(info os.FileInfo) int64   { return info.Size() }
