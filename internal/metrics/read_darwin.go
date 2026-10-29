package metrics

/*
#cgo LDFLAGS: -framework CoreFoundation -framework IOKit
#include "native_darwin.h"
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"rockshinops/internal/localization"
)

func readNative(timeout time.Duration) rawSample {
	raw := rawSample{cpuWrap32: true}
	var cpus *C.lo_cpu
	var count C.int
	if code := C.lo_read_cpu(&cpus, &count); code != 0 {
		raw.cpuError = fmt.Sprintf(localization.T("CPU 采集失败（Mach %d）"), code)
	} else {
		for _, c := range unsafe.Slice(cpus, int(count)) {
			raw.cpu = append(raw.cpu, cpuTicks{uint64(c.user), uint64(c.system), uint64(c.idle), uint64(c.nice)})
		}
		C.lo_free(unsafe.Pointer(cpus))
	}
	m := C.lo_read_memory()
	if m.error != 0 {
		raw.memory.Error = fmt.Sprintf(localization.T("内存采集失败（%d）"), m.error)
	} else {
		// Anonymous application pages, wired pages and the compressed pager.
		app := int64(m.active) + int64(m.inactive) + int64(m.speculative) - int64(m.purgeable) - int64(m.external)
		used := app + int64(m.wired) + int64(m.compressed)
		if app < 0 || used < 0 || uint64(used) > uint64(m.total) {
			raw.memory.Error = localization.T("系统返回了不一致的内存计数")
		} else {
			raw.memory = Memory{Total: uint64(m.total), Used: uint64(used), App: uint64(app), Wired: uint64(m.wired),
				Compressed: uint64(m.compressed), Cached: uint64(m.purgeable + m.external), Available: uint64(m.total) - uint64(used),
				SwapUsed: uint64(m.swap_used), SwapTotal: uint64(m.swap_total)}
			switch m.pressure {
			case 1:
				raw.memory.Pressure = "normal"
			case 2:
				raw.memory.Pressure = "elevated"
			case 4:
				raw.memory.Pressure = "critical"
			default:
				raw.memory.Pressure = "unavailable"
			}
			if m.swap_error != 0 {
				raw.memory.SwapError = fmt.Sprintf(localization.T("交换空间采集失败（%d）"), m.swap_error)
			}
		}
	}
	gpu := C.lo_read_gpu()
	raw.gpu = GPU{Name: C.GoString(&gpu.name[0]), Usage: float64(gpu.usage), Memory: uint64(gpu.memory), Available: gpu.available != 0}
	if !raw.gpu.Available {
		raw.gpu.Error = localization.T("当前系统未提供可读取的 GPU 性能计数")
	}
	var disks *C.lo_io
	count = 0
	if code := C.lo_read_disk(&disks, &count); code != 0 {
		raw.diskError = fmt.Sprintf(localization.T("磁盘 I/O 采集失败（%d）"), code)
	}
	for _, d := range unsafe.Slice(disks, int(count)) {
		raw.disks = append(raw.disks, ioCounter{id: C.GoString(&d.id[0]), read: uint64(d.read), write: uint64(d.write), up: true})
	}
	C.lo_free(unsafe.Pointer(disks))
	var nets *C.lo_io
	count = 0
	if code := C.lo_read_network(&nets, &count); code != 0 {
		raw.networkError = fmt.Sprintf(localization.T("网络接口采集失败（%d）"), code)
	}
	for _, n := range unsafe.Slice(nets, int(count)) {
		raw.networks = append(raw.networks, ioCounter{id: C.GoString(&n.id[0]), read: uint64(n.read), write: uint64(n.write), up: n.up != 0, included: strings.HasPrefix(C.GoString(&n.id[0]), "en")})
	}
	C.lo_free(unsafe.Pointer(nets))
	b := C.lo_read_battery()
	raw.battery = Battery{Present: b.present != 0, Percent: float64(b.percent), Charging: b.charging != 0, ExternalPower: b.external != 0, Minutes: int(b.minutes)}
	if b.error != 0 {
		raw.battery.Error = fmt.Sprintf(localization.T("电池采集失败（%d）"), b.error)
	}
	if b.health_valid != 0 {
		raw.battery.Health = ptr(float64(b.health))
	}
	if b.cycles_valid != 0 {
		value := int(b.cycles)
		raw.battery.Cycles = &value
	}
	if b.watts_valid != 0 {
		raw.battery.Watts = ptr(float64(b.watts))
	}
	if b.system_watts_valid != 0 {
		raw.battery.SystemWatts = ptr(float64(b.system_watts))
	}
	if b.temperature_valid != 0 {
		raw.battery.Temperature = ptr(float64(b.temperature))
	}
	raw.uptime = float64(C.lo_uptime())
	return raw
}

func readProcesses() ([]rawProcess, int, error) {
	var items *C.lo_process
	var count, total C.int
	if code := C.lo_read_processes(&items, &count, &total); code != 0 {
		return nil, 0, fmt.Errorf(localization.T("进程采集失败（%d）"), code)
	}
	defer C.lo_free(unsafe.Pointer(items))
	result := make([]rawProcess, 0, int(count))
	native := unsafe.Slice(items, int(count))
	for i := range native {
		p := &native[i]
		path := C.GoString(&p.path[0])
		app := ""
		if index := strings.Index(path, ".app/"); index >= 0 {
			app = path[:index+4]
		}
		result = append(result, rawProcess{Process: Process{PID: int(p.pid), Name: C.GoString(&p.name[0]), Path: path, App: app,
			Memory: uint64(p.memory), ReadBytes: uint64(p.read), WriteBytes: uint64(p.write), Threads: int(p.threads), UserID: uint32(p.uid), Own: uint32(p.uid) == uint32(os.Getuid())},
			start: uint64(p.start), cpuTime: uint64(p.cpu_time)})
	}
	return result, int(total), nil
}

func ReadMachine(timeout time.Duration) Machine {
	info := Machine{Platform: runtime.GOOS, Architecture: runtime.GOARCH}
	var failures []string
	var err error
	if info.Hostname, err = os.Hostname(); err != nil {
		failures = append(failures, err.Error())
	}
	if info.Model, err = syscall.Sysctl("hw.model"); err != nil {
		failures = append(failures, err.Error())
	}
	if info.Chip, err = syscall.Sysctl("machdep.cpu.brand_string"); err != nil {
		failures = append(failures, err.Error())
	}
	if out, err := exec.Command("/usr/bin/sw_vers", "-productVersion").Output(); err == nil {
		info.OS = "macOS " + strings.TrimSpace(string(out))
	} else {
		failures = append(failures, err.Error())
	}
	info.LogicalCPUs = int(sysNumber("hw.logicalcpu"))
	info.PhysicalCPUs = int(sysNumber("hw.physicalcpu"))
	info.Memory = sysNumber("hw.memsize")
	if info.LogicalCPUs == 0 || info.Memory == 0 {
		failures = append(failures, localization.T("本机硬件信息读取不完整"))
	}
	info.OSBuild, err = syscall.Sysctl("kern.osversion")
	if err != nil {
		failures = append(failures, err.Error())
	}
	if release, err := syscall.Sysctl("kern.osrelease"); err == nil {
		info.Kernel = "Darwin " + release
	} else {
		failures = append(failures, err.Error())
	}
	for level := 0; level < int(sysNumber("hw.nperflevels")); level++ {
		prefix := fmt.Sprintf("hw.perflevel%d.", level)
		name, err := syscall.Sysctl(prefix + "name")
		if err != nil {
			continue
		}
		count := int(sysNumber(prefix + "physicalcpu"))
		if count <= 0 {
			continue
		}
		switch name {
		case "Performance":
			info.PerformanceCores = &count
		case "Efficiency":
			info.EfficiencyCores = &count
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, "/usr/sbin/system_profiler", "-json", "SPHardwareDataType", "SPDisplaysDataType").Output()
	if err != nil {
		failures = append(failures, "system_profiler: "+err.Error())
	} else if err := decodeMachineProfile(output, &info); err != nil {
		failures = append(failures, err.Error())
	}
	info.Error = strings.Join(failures, "; ")
	return info
}

func sysNumber(key string) uint64 {
	name := C.CString(key)
	defer C.lo_free(unsafe.Pointer(name))
	return uint64(C.lo_sysctl_number(name))
}

func decodeMachineProfile(data []byte, info *Machine) error {
	var profile struct {
		Hardware []struct {
			Name   string `json:"machine_name"`
			Serial string `json:"serial_number"`
		} `json:"SPHardwareDataType"`
		Displays []struct {
			Cores string `json:"sppci_cores"`
		} `json:"SPDisplaysDataType"`
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return fmt.Errorf("system_profiler: %w", err)
	}
	if len(profile.Hardware) == 0 {
		return fmt.Errorf("system_profiler: missing hardware information")
	}
	info.Name, info.SerialNumber = profile.Hardware[0].Name, profile.Hardware[0].Serial
	if len(profile.Displays) > 0 && profile.Displays[0].Cores != "" {
		cores, err := strconv.Atoi(profile.Displays[0].Cores)
		if err != nil || cores <= 0 {
			return fmt.Errorf("system_profiler: invalid GPU core count")
		}
		info.GPUCores = &cores
	}
	return nil
}
