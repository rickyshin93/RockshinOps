//go:build linux || windows

package metrics

import (
	"context"
	"encoding/csv"
	"fmt"
	"rockshinops/internal/localization"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func readGPU(timeout time.Duration) GPU {
	if gpu, found := readDriverGPU(); found {
		return gpu
	}
	path, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return GPU{Error: localization.T("当前系统未提供可读取的 GPU 性能计数")}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--id=0", "--query-gpu=name,utilization.gpu,memory.used", "--format=csv,noheader,nounits")
	hideCommandWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return GPU{Error: fmt.Sprintf("nvidia-smi: %v", err)}
	}
	return parseNvidiaGPU(string(out))
}

func parseNvidiaGPU(output string) GPU {
	rows, err := csv.NewReader(strings.NewReader(output)).ReadAll()
	if err != nil || len(rows) != 1 || len(rows[0]) != 3 {
		return GPU{Error: localization.T("Invalid nvidia-smi response")}
	}
	row := rows[0]
	g := GPU{Name: strings.TrimSpace(row[0])}
	usage, err := strconv.ParseFloat(strings.TrimSpace(row[1]), 64)
	if err != nil || math.IsNaN(usage) || math.IsInf(usage, 0) || usage < 0 || usage > 100 {
		g.Error = localization.T("GPU utilization unavailable")
		return g
	}
	memory, err := strconv.ParseUint(strings.TrimSpace(row[2]), 10, 64)
	if err != nil {
		g.Error = localization.T("GPU memory unavailable")
		return g
	}
	g.Usage = usage
	g.Memory = memory * 1024 * 1024
	g.Available = true
	return g
}
