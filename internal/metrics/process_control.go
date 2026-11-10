package metrics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"rockshinops/internal/localization"
)

// TerminateProcess addresses one sampled process, never a process group.
// A successful return means the OS accepted the request, not that it exited.
func TerminateProcess(pid int, identity string, force bool) error {
	return terminateProcess(pid, identity, force, nil)
}

func TerminatePortProcess(ctx context.Context, timeout time.Duration, pid int, identity string, port int) error {
	if port < 1 || port > 65535 {
		return errors.New(localization.T("无效的监听端口。"))
	}
	return terminateProcess(pid, identity, true, func() error {
		current, err := readConnections(ctx, timeout, 0)
		if err != nil {
			return err
		}
		if !ownsListeningPort(current.Items, pid, port) {
			return errors.New(localization.T("监听端口已变化，请刷新后重试。"))
		}
		return nil
	})
}

func ownsListeningPort(items []Connection, pid, port int) bool {
	endpoint := ":" + strconv.Itoa(port)
	for _, item := range items {
		if item.PID == pid && item.Protocol == "TCP" && item.State == "LISTEN" && strings.HasSuffix(item.Local, endpoint) {
			return true
		}
	}
	return false
}

func terminateProcess(pid int, identity string, force bool, verifyListener func() error) error {
	if pid <= 1 || pid == os.Getpid() {
		return errors.New(localization.T("不能结束该系统进程或 RockshinOps 自身。"))
	}
	if identity == "" || identity == "0" {
		return errors.New(localization.T("缺少进程身份，请刷新列表后重试。"))
	}
	if verifyListener != nil {
		if err := verifyListener(); err != nil {
			return err
		}
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
