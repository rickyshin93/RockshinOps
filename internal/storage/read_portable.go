//go:build linux

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
	"rockshinops/internal/localization"
)

func Read() (Volume, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Volume{}, err
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return Volume{}, err
	}
	usage, err := disk.Usage(home)
	if err != nil {
		return Volume{}, err
	}
	var mount string
	partitions, err := disk.Partitions(true)
	if err != nil {
		return Volume{}, err
	}
	longest := 0
	for _, p := range partitions {
		rel, err := filepath.Rel(p.Mountpoint, home)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && len(p.Mountpoint) > longest {
			mount = p.Mountpoint
			longest = len(mount)
		}
	}
	if mount == "" {
		return Volume{}, fmt.Errorf("no volume mount contains the home directory %q", home)
	}
	return Volume{Name: localization.T("用户数据盘"), Mount: mount, Format: strings.ToUpper(usage.Fstype), Total: usage.Total, Available: usage.Free}, nil
}
