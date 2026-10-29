package storage

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"rockshinops/internal/localization"
)

func Read() (Volume, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Volume{}, fmt.Errorf(localization.T("无法确定用户数据盘：%w"), err)
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(home, &stat); err != nil {
		return Volume{}, fmt.Errorf(localization.T("无法读取磁盘容量：%w"), err)
	}
	if stat.Bsize <= 0 || stat.Blocks == 0 || stat.Bavail > stat.Blocks {
		return Volume{}, errors.New(localization.T("系统返回了无效的磁盘容量"))
	}
	return Volume{
		Name:      localization.T("用户数据盘"),
		Mount:     nullString(stat.Mntonname[:]),
		Format:    strings.ToUpper(nullString(stat.Fstypename[:])),
		Total:     stat.Blocks * uint64(stat.Bsize),
		Available: stat.Bavail * uint64(stat.Bsize),
	}, nil
}

func nullString(chars []int8) string {
	b := make([]byte, 0, len(chars))
	for _, c := range chars {
		if c == 0 {
			break
		}
		b = append(b, byte(c))
	}
	return string(b)
}
