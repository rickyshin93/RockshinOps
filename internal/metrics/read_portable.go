//go:build linux || windows

package metrics

import (
	"fmt"
	"rockshinops/internal/localization"
	"os/user"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

func readNative(timeout time.Duration) rawSample {
	r := rawSample{uptime: -1}
	times, err := cpu.Times(true)
	if err != nil {
		r.cpuError = err.Error()
	} else if len(times) == 0 {
		r.cpuError = localization.T("CPU counters unavailable")
	} else {
		for _, c := range times {
			// Linux guest time is already included in user/nice. I/O wait is idle.
			r.cpu = append(r.cpu, cpuTicks{user: uint64(c.User * 1e9), nice: uint64(c.Nice * 1e9), system: uint64((c.System + c.Irq + c.Softirq + c.Steal) * 1e9), idle: uint64((c.Idle + c.Iowait) * 1e9)})
		}
	}
	vm, err := mem.VirtualMemory()
	if err != nil {
		r.memory.Error = err.Error()
	} else {
		r.memory = Memory{Total: vm.Total, Used: vm.Total - vm.Available, Available: vm.Available, Cached: vm.Cached, Pressure: "unavailable"}
	}
	swap, err := mem.SwapMemory()
	if err != nil {
		r.memory.SwapError = err.Error()
	} else {
		r.memory.SwapTotal = swap.Total
		r.memory.SwapUsed = swap.Used
	}
	r.disks, err = readDisks()
	if err != nil {
		r.diskError = err.Error()
	}
	r.networks, err = readNetworks()
	if err != nil {
		r.networkError = err.Error()
	}
	if uptime, err := host.Uptime(); err == nil {
		r.uptime = float64(uptime)
	}
	r.battery = readBattery()
	r.gpu = readGPU(timeout)
	return r
}

func readProcesses() ([]rawProcess, int, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, 0, err
	}
	current, currentErr := user.Current()
	items := make([]rawProcess, 0, len(pids))
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			continue
		}
		start, err := p.CreateTime()
		if err != nil {
			continue
		}
		times, err := p.Times()
		if err != nil {
			continue
		}
		memory, err := p.MemoryInfo()
		if err != nil {
			continue
		}
		name, err := p.Name()
		if err != nil {
			continue
		}
		item := rawProcess{Process: Process{PID: int(pid), Name: name, Memory: memory.RSS}, start: uint64(start), cpuTime: uint64((times.User + times.System) * 1e9)}
		if path, err := p.Exe(); err == nil {
			item.Path = path
			item.App = path
		}
		if threads, err := p.NumThreads(); err == nil {
			item.Threads = int(threads)
		}
		if owner, err := p.Username(); err == nil && currentErr == nil {
			item.Own = strings.EqualFold(owner, current.Username)
		}
		if io, err := p.IOCounters(); err == nil {
			item.ReadBytes = io.ReadBytes
			item.WriteBytes = io.WriteBytes
		} else {
			item.IOError = err.Error()
		}
		items = append(items, item)
	}
	return items, len(pids), nil
}

func ReadMachine(timeout time.Duration) Machine {
	m := Machine{Platform: runtime.GOOS, Architecture: runtime.GOARCH}
	var failures []string
	if info, err := host.Info(); err == nil {
		m.Kernel = info.KernelVersion
		m.Hostname = info.Hostname
		m.OS = strings.TrimSpace(info.Platform + " " + info.PlatformVersion)
		m.Model = info.VirtualizationSystem
	} else {
		failures = append(failures, err.Error())
	}
	if info, err := cpu.Info(); err == nil && len(info) > 0 {
		m.Chip = info[0].ModelName
	} else if err != nil {
		failures = append(failures, err.Error())
	}
	var err error
	m.LogicalCPUs, err = cpu.Counts(true)
	if err != nil {
		failures = append(failures, err.Error())
	}
	m.PhysicalCPUs, err = cpu.Counts(false)
	if err != nil {
		failures = append(failures, err.Error())
	}
	if v, err := mem.VirtualMemory(); err == nil {
		m.Memory = v.Total
	} else {
		failures = append(failures, err.Error())
	}
	if model := machineModel(); model != "" {
		m.Model = model
	}
	if m.Model == "" {
		m.Model = "—"
	}
	m.Name = m.Model
	m.Error = strings.Join(failures, "; ")
	return m
}

func sortedCounters(c []ioCounter) []ioCounter {
	slices.SortFunc(c, func(a, b ioCounter) int { return strings.Compare(a.id, b.id) })
	return c
}

func noCounters(kind string) error {
	return fmt.Errorf(localization.T("%s counters unavailable"), kind)
}
