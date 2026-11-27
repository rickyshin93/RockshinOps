package metrics

import "os"

// ScanTarget is a configured cleanup location shown with its scanned size.
type ScanTarget struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Cleanup string `json:"cleanup,omitempty"`
}

// CleanupFileBytes omits hard links because another path may retain their blocks.
func CleanupFileBytes(info os.FileInfo) int64 {
	if linkCount(info) > 1 {
		return 0
	}
	return allocatedBytes(info)
}

// CleanupLinkCount reports the available number of links to an entry.
func CleanupLinkCount(info os.FileInfo) uint64 { return linkCount(info) }

// FileAllocatedBytes includes each directory entry’s allocated size.
func FileAllocatedBytes(info os.FileInfo) int64 { return allocatedBytes(info) }
