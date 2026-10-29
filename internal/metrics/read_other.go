//go:build !darwin && !linux && !windows

package metrics

import (
	"context"
	"errors"
	"runtime"
	"time"

	"rockshinops/internal/localization"
)

func readNative(timeout time.Duration) rawSample {
	err := localization.T("当前平台尚未实现本机监控")
	return rawSample{cpuError: err, memory: Memory{Error: err}, gpu: GPU{Error: err}, diskError: err, networkError: err, battery: Battery{Error: err}, uptime: -1}
}
func readProcesses() ([]rawProcess, int, error) {
	return nil, 0, errors.New(localization.T("当前平台尚未实现进程采集"))
}
func ReadMachine(timeout time.Duration) Machine {
	return Machine{Platform: runtime.GOOS, Architecture: runtime.GOARCH, Error: localization.T("当前平台尚未实现本机信息采集")}
}

func readConnections(ctx context.Context, timeout, interval time.Duration) (Connections, error) {
	return Connections{Items: []Connection{}, UpdatedAt: time.Now().UTC()}, errors.New(localization.T("当前平台尚未实现连接采集"))
}
