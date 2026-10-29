package metrics

import (
	"errors"
	"os"

	"rockshinops/internal/localization"
)

func signalProcess(pid int, force bool) error {
	if !force {
		return errors.New(localization.T("Windows 暂不支持请求正常退出，请使用强制退出。"))
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	defer process.Release()
	return process.Kill()
}
