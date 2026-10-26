package metrics

import (
	"errors"
	"testing"
	"time"

	"localops/internal/storage"
)

func TestCounterDeltasBaselineResetAndSleep(t *testing.T) {
	m := NewMonitor(Options{Interval: 2 * time.Second, DiskInterval: 30 * time.Second, ProcessInterval: 5 * time.Second, HistorySeconds: 60})
	raw := rawSample{cpu: []cpuTicks{{user: 100, system: 50, idle: 850}}, disks: []ioCounter{{id: "one", read: 1000, write: 2000}}, networks: []ioCounter{{id: "en0", included: true, read: 3000, write: 4000}, {id: "utun0", read: 9000, write: 9000}}, memory: Memory{Total: 1000, Used: 600}}
	m.read = func() rawSample { return raw }
	m.readProcesses = func() ([]rawProcess, int, error) { return nil, 0, nil }
	m.readVolume = func() (storage.Volume, error) { return storage.Volume{Total: 10000, Available: 5000}, nil }
	now := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	first := m.refreshAt(now, false)
	if first.CPU.Ready || first.Disk.Ready || first.Network.Ready {
		t.Fatal("first counters must be a baseline")
	}
	raw.cpu = []cpuTicks{{user: 120, system: 60, idle: 920}}
	raw.disks = []ioCounter{{id: "one", read: 1400, write: 2200}}
	raw.networks = []ioCounter{{id: "en0", included: true, read: 3400, write: 4200}, {id: "utun0", read: 9500, write: 9500}}
	second := m.refreshAt(now.Add(2*time.Second), false)
	if second.CPU.Usage != 30 || second.Disk.ReadRate != 200 || second.Network.Download != 200 || second.Network.Received != 3400 {
		t.Fatalf("bad delta or duplicate tunnel traffic: %+v", second)
	}
	raw.disks = []ioCounter{{id: "one", read: 1, write: 1}}
	reset := m.refreshAt(now.Add(4*time.Second), false)
	if reset.Disk.Ready || reset.Disk.ReadRate != 0 {
		t.Fatal("counter reset produced throughput")
	}
	sleep := m.refreshAt(now.Add(time.Minute), false)
	if sleep.CPU.Ready || sleep.Network.Ready {
		t.Fatal("sleep gap must rebaseline rates")
	}
}

func TestProcessPIDReuseAndFailuresClearCurrentValues(t *testing.T) {
	m := NewMonitor(Options{Interval: 2 * time.Second, DiskInterval: time.Second, ProcessInterval: 5 * time.Second, HistorySeconds: 60})
	raw := rawSample{cpuError: "no cpu"}
	m.read = func() rawSample { return raw }
	proc := rawProcess{Process: Process{PID: 10, App: "/Applications/Test.app", Memory: 400}, start: 100, cpuTime: 1e9}
	var failure error
	m.readProcesses = func() ([]rawProcess, int, error) {
		if failure != nil {
			return nil, 0, failure
		}
		return []rawProcess{proc}, 2, nil
	}
	m.readVolume = func() (storage.Volume, error) {
		if failure != nil {
			return storage.Volume{}, failure
		}
		return storage.Volume{Total: 1000, Available: 100}, nil
	}
	now := time.Now()
	m.refreshAt(now, false)
	proc.cpuTime += 5e9
	m.refreshAt(now.Add(5*time.Second), false)
	if p := m.Processes(); p.Items[0].CPU != 100 || p.Skipped != 1 {
		t.Fatalf("wrong process percent/count: %+v", p)
	}
	proc.start = 200
	m.refreshAt(now.Add(10*time.Second), false)
	if m.Processes().Items[0].CPUReady {
		t.Fatal("reused PID inherited another process's counters")
	}
	failure = errors.New("denied")
	s := m.refreshAt(now.Add(15*time.Second), true)
	if s.Disk.Volume.Available != 0 || s.Disk.CapacityError != "denied" || len(s.TopCPU) != 0 || s.ProcessError != "denied" {
		t.Fatalf("stale value survived failure: %+v", s)
	}
}

func TestHistoryRetentionKeepsGapsAndGroupsApps(t *testing.T) {
	m := NewMonitor(Options{Interval: time.Second, DiskInterval: time.Second, ProcessInterval: time.Second, HistorySeconds: 2})
	m.read = func() rawSample { return rawSample{cpuError: "denied", memory: Memory{Error: "denied"}} }
	m.readVolume = func() (storage.Volume, error) { return storage.Volume{}, errors.New("denied") }
	m.readProcesses = func() ([]rawProcess, int, error) { return nil, 0, nil }
	now := time.Now()
	for i := 0; i < 5; i++ {
		m.refreshAt(now.Add(time.Duration(i)*time.Second), false)
	}
	history := m.History()
	if len(history) != 3 || history[0].CPU != nil || history[0].Memory != nil {
		t.Fatalf("invalid retained history: %+v", history)
	}
	apps := GroupApps([]Process{{App: "/A.app", Memory: 200, CPU: 10, CPUReady: true}, {App: "/A.app", Memory: 300, CPU: 20, CPUReady: true}, {App: "/B.app", Memory: 100}, {Name: "daemon", Memory: 999}})
	if len(apps) != 2 || apps[0].Memory != 500 || apps[0].CPU != 30 || apps[0].Processes != 2 {
		t.Fatalf("invalid app grouping: %+v", apps)
	}
}

func TestCPUCounterWidthAndProcessIOAvailability(t *testing.T) {
	m := NewMonitor(Options{Interval: time.Second, DiskInterval: time.Second, ProcessInterval: time.Second, HistorySeconds: 60})
	raw := rawSample{cpu: []cpuTicks{{user: 1<<32 - 5, idle: 100}}, cpuWrap32: true}
	p := rawProcess{Process: Process{PID: 1, IOError: "denied"}, start: 1, cpuTime: 1}
	m.read = func() rawSample { return raw }
	m.readProcesses = func() ([]rawProcess, int, error) { return []rawProcess{p}, 1, nil }
	m.readVolume = func() (storage.Volume, error) { return storage.Volume{}, nil }
	now := time.Now()
	m.refreshAt(now, true)
	raw.cpu = []cpuTicks{{user: 5, idle: 110}}
	p.cpuTime += 1e9
	s := m.refreshAt(now.Add(time.Second), true)
	if !s.CPU.Ready || s.CPU.Usage != 50 {
		t.Fatalf("Mach wrap: %+v", s.CPU)
	}
	if !m.Processes().Items[0].CPUReady || m.Processes().Items[0].IOReady {
		t.Fatal("optional I/O failure must preserve CPU and hide I/O")
	}
	raw.cpuWrap32 = false
	raw.cpu = []cpuTicks{{user: 2, idle: 120}}
	s = m.refreshAt(now.Add(2*time.Second), true)
	if s.CPU.Ready {
		t.Fatal("64-bit counter reset must establish a new baseline")
	}
	p.IOError = ""
	p.ReadBytes = 100
	p.WriteBytes = 200
	p.cpuTime += 1e9
	m.refreshAt(now.Add(3*time.Second), true)
	if m.Processes().Items[0].IOReady {
		t.Fatal("I/O recovery needs a new baseline")
	}
	p.ReadBytes = 200
	p.WriteBytes = 300
	p.cpuTime += 1e9
	m.refreshAt(now.Add(4*time.Second), true)
	if got := m.Processes().Items[0]; !got.IOReady || got.ReadRate != 100 {
		t.Fatalf("I/O recovered: %+v", got)
	}
}
