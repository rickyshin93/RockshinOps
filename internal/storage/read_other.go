//go:build !darwin && !linux && !windows

package storage

import (
	"fmt"
	"runtime"

	"localops/internal/localization"
)

func Read() (Volume, error) {
	return Volume{}, fmt.Errorf(localization.T("当前版本尚未实现 %s 的磁盘采集"), runtime.GOOS)
}
