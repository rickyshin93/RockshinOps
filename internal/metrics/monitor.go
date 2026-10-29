package metrics

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"rockshinops/internal/storage"
)

type Options struct {
	Interval        time.Duration
	DiskInterval    time.Duration
	ProcessInterval time.Duration
	HistorySeconds  int
}

type Monitor struct {
	mu              sync.RWMutex
	sampleMu        sync.Mutex
	options         Options
	read            func() rawSample
	readProcesses   func() ([]rawProcess, int, error)
	readVolume      func() (storage.Volume, error)
	previous        rawSample
	previousAt      time.Time
	capacityAt      time.Time
	processAt       time.Time
	processPrevious map[int]processCounters
	processes       ProcessList
	snapshot        Snapshot
	history         []Trend
}

func NewMonitor(options Options) *Monitor {
	return &Monitor{options: options, read: func() rawSample { return readNative(options.Interval) }, readProcesses: readProcesses, readVolume: storage.Read,
		processPrevious: make(map[int]processCounters), history: []Trend{}, processes: ProcessList{Items: []Process{}},
		snapshot: Snapshot{CPU: CPU{Cores: []float64{}}, TopCPU: []Process{}, TopMemory: []AppUsage{}}}
}

func (m *Monitor) Current() Snapshot { m.mu.RLock(); defer m.mu.RUnlock(); return m.snapshot }
func (m *Monitor) History() []Trend {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Clone(m.history)
}
func (m *Monitor) Processes() ProcessList { m.mu.RLock(); defer m.mu.RUnlock(); return m.processes }

// HistorySince returns only samples newer than the caller's last accepted point.
func (m *Monitor) HistorySince(after time.Time) []Trend {
	m.mu.RLock()
	defer m.mu.RUnlock()
	first := len(m.history)
	for first > 0 && m.history[first-1].At.After(after) {
		first--
	}
	return slices.Clone(m.history[first:])
}

func ptr(value float64) *float64 { return &value }

func (m *Monitor) Refresh(force bool) Snapshot { return m.refreshAt(time.Now(), force) }

func (m *Monitor) refreshAt(now time.Time, force bool) Snapshot {
	// Native I/O is serialized separately from reads of the last published data.
	m.sampleMu.Lock()
	defer m.sampleMu.Unlock()
	raw := m.read()
	seconds := now.Sub(m.previousAt).Seconds()
	continuous := !m.previousAt.IsZero() && seconds > 0 && now.Sub(m.previousAt) <= m.options.Interval*3
	s := Snapshot{UpdatedAt: now.UTC(), Memory: raw.memory, GPU: raw.gpu, Battery: raw.battery, UptimeSeconds: raw.uptime}
	s.CPU = CPU{Cores: []float64{}, Error: raw.cpuError}
	if continuous && raw.cpuError == "" && m.previous.cpuError == "" && len(raw.cpu) == len(m.previous.cpu) {
		var user, system, total uint64
		for i, current := range raw.cpu {
			prev := m.previous.cpu[i]
			if !raw.cpuWrap32 && (current.user < prev.user || current.system < prev.system || current.idle < prev.idle || current.nice < prev.nice) {
				total = 0
				s.CPU.Cores = []float64{}
				break
			}
			delta := func(a, b uint64) uint64 {
				if raw.cpuWrap32 {
					return uint64(uint32(a) - uint32(b))
				}
				return a - b
			}
			u, sys := delta(current.user, prev.user)+delta(current.nice, prev.nice), delta(current.system, prev.system)
			all := u + sys + delta(current.idle, prev.idle)
			usage := float64(0)
			if all > 0 {
				usage = float64(u+sys) / float64(all) * 100
			}
			s.CPU.Cores = append(s.CPU.Cores, usage)
			user += u
			system += sys
			total += all
		}
		if total > 0 {
			s.CPU.Ready = true
			s.CPU.User = float64(user) / float64(total) * 100
			s.CPU.System = float64(system) / float64(total) * 100
			s.CPU.Usage = s.CPU.User + s.CPU.System
		}
	}
	s.Disk = Disk{Volume: m.snapshot.Disk.Volume, CapacityError: m.snapshot.Disk.CapacityError, Error: raw.diskError}
	if force || m.capacityAt.IsZero() || now.Sub(m.capacityAt) >= m.options.DiskInterval {
		volume, err := m.readVolume()
		s.Disk.Volume = volume
		s.Disk.CapacityError = ""
		if err != nil {
			s.Disk.CapacityError = err.Error()
		}
		m.capacityAt = now
	}
	previousDisk := indexIO(m.previous.disks)
	for _, device := range raw.disks {
		s.Disk.ReadBytes += device.read
		s.Disk.WriteBytes += device.write
		if continuous && raw.diskError == "" && m.previous.diskError == "" {
			if read, write, ok := rate(device, previousDisk, seconds); ok {
				s.Disk.ReadRate += read
				s.Disk.WriteRate += write
				s.Disk.Ready = true
			}
		}
	}
	s.Network = Network{Interfaces: []NetworkInterface{}, Error: raw.networkError}
	previousNetwork := indexIO(m.previous.networks)
	for _, device := range raw.networks {
		n := NetworkInterface{Name: device.id, Received: device.read, Sent: device.write, Up: device.up, Included: device.included}
		if continuous && raw.networkError == "" && m.previous.networkError == "" {
			n.Download, n.Upload, n.Ready = rate(device, previousNetwork, seconds)
		}
		s.Network.Interfaces = append(s.Network.Interfaces, n)
		if n.Included {
			s.Network.Received += n.Received
			s.Network.Sent += n.Sent
			s.Network.Download += n.Download
			s.Network.Upload += n.Upload
			s.Network.Ready = s.Network.Ready || n.Ready
		}
	}
	processes := m.processes
	processChanged := force || m.processAt.IsZero() || now.Sub(m.processAt) >= m.options.ProcessInterval
	if processChanged {
		processes = m.refreshProcesses(now)
	}
	s.ProcessUpdatedAt = processes.UpdatedAt
	s.ProcessCount = processes.Total
	s.ProcessSkipped = processes.Skipped
	s.ProcessError = processes.Error
	s.TopCPU, s.TopMemory = m.snapshot.TopCPU, m.snapshot.TopMemory
	if processChanged {
		s.TopCPU = slices.Clone(processes.Items)
		slices.SortFunc(s.TopCPU, func(a, b Process) int {
			if a.CPU > b.CPU {
				return -1
			}
			if a.CPU < b.CPU {
				return 1
			}
			return a.PID - b.PID
		})
		s.TopCPU = s.TopCPU[:min(6, len(s.TopCPU))]
		s.TopMemory = GroupApps(processes.Items)
		s.TopMemory = s.TopMemory[:min(6, len(s.TopMemory))]
	}
	m.previous, m.previousAt = raw, now
	t := Trend{At: s.UpdatedAt}
	if s.CPU.Ready {
		t.CPU = ptr(s.CPU.Usage)
	}
	if s.Memory.Error == "" && s.Memory.Total > 0 {
		t.Memory = ptr(float64(s.Memory.Used) / float64(s.Memory.Total) * 100)
	}
	if s.GPU.Available {
		t.GPU = ptr(s.GPU.Usage)
	}
	if s.Disk.Ready {
		t.DiskRead, t.DiskWrite = ptr(s.Disk.ReadRate), ptr(s.Disk.WriteRate)
	}
	if s.Network.Ready {
		t.Download, t.Upload = ptr(s.Network.Download), ptr(s.Network.Upload)
	}
	if s.Battery.Present && s.Battery.Error == "" {
		t.Battery = ptr(s.Battery.Percent)
		t.Power = s.Battery.SystemWatts
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshot, m.processes = s, processes
	m.history = append(m.history, t)
	cutoff := now.Add(-time.Duration(m.options.HistorySeconds) * time.Second)
	first := 0
	for first < len(m.history) && m.history[first].At.Before(cutoff) {
		first++
	}
	if first > 0 {
		m.history = slices.Clone(m.history[first:])
	}
	return s
}

func indexIO(items []ioCounter) map[string]ioCounter {
	result := make(map[string]ioCounter, len(items))
	for _, item := range items {
		result[item.id] = item
	}
	return result
}

func rate(current ioCounter, previous map[string]ioCounter, seconds float64) (float64, float64, bool) {
	old, found := previous[current.id]
	if !found || current.read < old.read || current.write < old.write {
		return 0, 0, false
	}
	return float64(current.read-old.read) / seconds, float64(current.write-old.write) / seconds, true
}

type processCounters struct {
	start, cpuTime, read, write uint64
	ioValid                     bool
}

func (m *Monitor) refreshProcesses(now time.Time) ProcessList {
	items, total, err := m.readProcesses()
	result := ProcessList{Items: make([]Process, 0, len(items)), Total: total, UpdatedAt: now.UTC()}
	seconds := now.Sub(m.processAt).Seconds()
	continuous := !m.processAt.IsZero() && seconds > 0 && now.Sub(m.processAt) <= m.options.ProcessInterval*3
	previous := make(map[int]processCounters, len(items))
	if err != nil {
		result.Error = err.Error()
	}
	for _, raw := range items {
		p := raw.Process
		p.Identity = strconv.FormatUint(raw.start, 10)
		if old, exists := m.processPrevious[p.PID]; exists && old.start == raw.start && continuous && raw.cpuTime >= old.cpuTime {
			p.CPU = float64(raw.cpuTime-old.cpuTime) / 1e9 / seconds * 100
			p.CPUReady = true
			if raw.IOError == "" && old.ioValid && raw.ReadBytes >= old.read && raw.WriteBytes >= old.write {
				p.ReadRate = float64(raw.ReadBytes-old.read) / seconds
				p.WriteRate = float64(raw.WriteBytes-old.write) / seconds
				p.IOReady = true
			}
		}
		result.Items = append(result.Items, p)
		previous[p.PID] = processCounters{raw.start, raw.cpuTime, raw.ReadBytes, raw.WriteBytes, raw.IOError == ""}
	}
	result.Skipped = total - len(items)
	m.processAt, m.processPrevious = now, previous
	return result
}

func GroupApps(processes []Process) []AppUsage {
	groups := map[string]*AppUsage{}
	for _, p := range processes {
		if p.App == "" {
			continue
		}
		app := groups[p.App]
		if app == nil {
			app = &AppUsage{Name: strings.TrimSuffix(filepath.Base(p.App), ".app"), Path: p.App, CPUReady: true}
			groups[p.App] = app
		}
		app.Processes++
		app.CPU += p.CPU
		app.Memory += p.Memory
		app.CPUReady = app.CPUReady && p.CPUReady
	}
	result := make([]AppUsage, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b AppUsage) int {
		if a.Memory > b.Memory {
			return -1
		}
		if a.Memory < b.Memory {
			return 1
		}
		return strings.Compare(a.Path, b.Path)
	})
	return result
}
