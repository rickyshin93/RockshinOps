//go:build !darwin && !linux && !windows

package metrics

import (
	"errors"
	"rockshinops/internal/localization"
)

func signalProcess(pid int, force bool) error {
	return errors.New(localization.T("当前平台不支持结束进程。"))
}
