package metrics

import (
	"rockshinops/internal/localization"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func hideCommandWindow(cmd *exec.Cmd) {}

func readDriverGPU() (GPU, bool) {
	devices, err := filepath.Glob("/sys/class/drm/card[0-9]*/device/gpu_busy_percent")
	if err != nil || len(devices) == 0 {
		return GPU{}, false
	}
	device := filepath.Dir(devices[0])
	g := GPU{Name: "AMD " + filepath.Base(filepath.Dir(device))}
	if name, err := os.ReadFile(filepath.Join(device, "product_name")); err == nil {
		g.Name = strings.TrimSpace(string(name))
	}
	usage, err := readSysNumber(devices[0])
	if err != nil {
		g.Error = err.Error()
		return g, true
	}
	memory, err := readSysNumber(filepath.Join(device, "mem_info_vram_used"))
	if err != nil {
		g.Error = err.Error()
		return g, true
	}
	if usage < 0 || usage > 100 || memory < 0 {
		g.Error = localization.T("Invalid GPU counters")
		return g, true
	}
	g.Usage = usage
	g.Memory = uint64(memory)
	g.Available = true
	return g, true
}
