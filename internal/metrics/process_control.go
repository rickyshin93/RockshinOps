package metrics

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"localops/internal/localization"
)

// TerminateProcess addresses one sampled process, never a process group.
// A successful return means the OS accepted the request, not that it exited.
func TerminateProcess(pid int, identity string, force bool) error {
	if pid <= 1 || pid == os.Getpid() {
		return errors.New(localization.T("不能结束该系统进程或 RockshinOps 自身。"))
	}
	if identity == "" || identity == "0" {
		return errors.New(localization.T("缺少进程身份，请刷新列表后重试。"))
	}
	items, _, err := readProcesses()
	if err != nil {
		return fmt.Errorf("%s: %w", localization.T("无法核对进程身份"), err)
	}
	for _, item := range items {
		if item.PID != pid {
			continue
		}
		if strconv.FormatUint(item.start, 10) != identity {
			return errors.New(localization.T("进程身份已变化，请重新选择进程。"))
		}
		if err := signalProcess(pid, force); err != nil {
			return fmt.Errorf("%s: %w", localization.T("无法结束进程（可能无权限或进程已退出）"), err)
		}
		return nil
	}
	return errors.New(localization.T("进程已退出或当前权限无法读取，请刷新列表。"))
}
