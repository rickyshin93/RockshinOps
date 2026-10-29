//go:build darwin || linux || windows

package metrics

import (
	"context"
	"net"
	"os"
	"runtime"
	"testing"
	"time"
)

// Real OS calls, including a known listening socket and the current process.
// This runs on each native CI runner, not through a cross-compilation wrapper.
func TestNativeCollectorSmoke(t *testing.T) {
	m := NewMonitor(Options{Interval: time.Second, DiskInterval: time.Second, ProcessInterval: time.Second, HistorySeconds: 60})
	first := m.Refresh(true)
	if first.CPU.Ready || first.Network.Ready || first.Disk.Ready {
		t.Fatal("first native sample must establish a baseline")
	}
	time.Sleep(200 * time.Millisecond)
	second := m.Refresh(true)
	if second.CPU.Error != "" || !second.CPU.Ready || second.CPU.Usage < 0 || second.CPU.Usage > 100 {
		t.Fatalf("CPU: %+v", second.CPU)
	}
	if second.Memory.Error != "" || second.Memory.Total == 0 || second.Memory.Used > second.Memory.Total || second.Memory.Available > second.Memory.Total {
		t.Fatalf("memory: %+v", second.Memory)
	}
	if second.Disk.CapacityError != "" || second.Disk.Volume.Total == 0 || second.Disk.Volume.Available > second.Disk.Volume.Total {
		t.Fatalf("capacity: %+v", second.Disk)
	}
	if second.Disk.Error != "" || !second.Disk.Ready {
		t.Fatalf("disk rates: %+v", second.Disk)
	}
	if second.Network.Error != "" || len(second.Network.Interfaces) == 0 {
		t.Fatalf("network: %+v", second.Network)
	}
	ready := false
	for _, n := range second.Network.Interfaces {
		ready = ready || n.Ready
	}
	if !ready {
		t.Fatal("no interface rate baseline")
	}
	own := false
	for _, p := range m.Processes().Items {
		if p.PID == os.Getpid() {
			own = true
			if !p.CPUReady || p.Memory == 0 {
				t.Fatalf("current process: %+v", p)
			}
		}
	}
	if !own {
		t.Fatalf("current process missing; %s", m.Processes().Error)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	connections, err := ReadConnections(context.Background(), 10*time.Second, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range connections.Items {
		if c.Local == listener.Addr().String() && c.PID == os.Getpid() {
			found = true
		}
	}
	if !found {
		t.Fatalf("known listening socket missing on %s", runtime.GOOS)
	}
	machine := ReadMachine(5 * time.Second)
	if machine.Error != "" || machine.LogicalCPUs == 0 || machine.Memory == 0 || machine.OS == "" {
		t.Fatalf("machine: %+v", machine)
	}
	t.Logf("%s/%s: CPU %.2f%%, RAM %d, volume %d, disk read %.0f B/s, %d interfaces, %d processes, %d sockets; GPU available=%t; battery present=%t", runtime.GOOS, runtime.GOARCH, second.CPU.Usage, second.Memory.Total, second.Disk.Volume.Total, second.Disk.ReadRate, len(second.Network.Interfaces), len(m.Processes().Items), len(connections.Items), second.GPU.Available, second.Battery.Present)
}
