package metrics

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestNativeSnapshot(t *testing.T) {
	raw := readNative(time.Second)
	if raw.cpuError != "" || len(raw.cpu) == 0 {
		t.Fatalf("CPU: %s", raw.cpuError)
	}
	if raw.memory.Error != "" || raw.memory.Total == 0 || raw.memory.Used > raw.memory.Total {
		t.Fatalf("Memory: %+v", raw.memory)
	}
	if raw.diskError != "" || len(raw.disks) == 0 {
		t.Fatalf("Disk: %s", raw.diskError)
	}
	if raw.networkError != "" {
		t.Fatalf("Network: %s", raw.networkError)
	}
	if raw.battery.Error != "" {
		t.Fatalf("Battery: %+v", raw.battery)
	}
	processes, total, err := readProcesses()
	if err != nil || len(processes) == 0 || total < len(processes) {
		t.Fatalf("Processes: %d/%d %v", len(processes), total, err)
	}
	info := ReadMachine()
	if info.Error != "" || info.LogicalCPUs == 0 {
		t.Fatalf("Machine: %+v", info)
	}
	t.Logf("native CPU cores=%d, memory=%d/%d, GPU=%v, disks=%d, interfaces=%d, battery=%v, processes=%d/%d", len(raw.cpu), raw.memory.Used, raw.memory.Total, raw.gpu, len(raw.disks), len(raw.networks), raw.battery.Present, len(processes), total)
}

func TestProcessCPUTimeMatchesGetrusage(t *testing.T) {
	readOwn := func() uint64 {
		items, _, err := readProcesses()
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range items {
			if p.PID == os.Getpid() {
				return p.cpuTime
			}
		}
		t.Fatal("own process missing")
		return 0
	}
	cpu := func() float64 {
		var r syscall.Rusage
		if err := syscall.Getrusage(syscall.RUSAGE_SELF, &r); err != nil {
			t.Fatal(err)
		}
		return float64(r.Utime.Sec+r.Stime.Sec) + float64(r.Utime.Usec+r.Stime.Usec)/1e6
	}
	before := readOwn()
	beforeCPU := cpu()
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
	}
	expected := cpu() - beforeCPU
	actual := float64(readOwn()-before) / 1e9
	t.Logf("native CPU seconds=%.6f getrusage=%.6f ratio=%.3f", actual, expected, actual/expected)
	if actual < expected*.8 || actual > expected*1.3 {
		t.Fatalf("CPU counter unit mismatch: %f vs %f", actual, expected)
	}
}
