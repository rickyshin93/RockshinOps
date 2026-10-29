package metrics

import (
	"time"

	"localops/internal/storage"
)

type CPU struct {
	Usage  float64   `json:"usage"`
	User   float64   `json:"user"`
	System float64   `json:"system"`
	Cores  []float64 `json:"cores"`
	Ready  bool      `json:"ready"`
	Error  string    `json:"error"`
}

type Memory struct {
	Total      uint64 `json:"total"`
	Used       uint64 `json:"used"`
	App        uint64 `json:"app"`
	Wired      uint64 `json:"wired"`
	Compressed uint64 `json:"compressed"`
	Cached     uint64 `json:"cached"`
	Available  uint64 `json:"available"`
	SwapUsed   uint64 `json:"swapUsed"`
	SwapTotal  uint64 `json:"swapTotal"`
	SwapError  string `json:"swapError"`
	Pressure   string `json:"pressure"`
	Error      string `json:"error"`
}

type GPU struct {
	Name      string  `json:"name"`
	Usage     float64 `json:"usage"`
	Memory    uint64  `json:"memory"`
	Available bool    `json:"available"`
	Error     string  `json:"error"`
}

type Disk struct {
	Volume        storage.Volume `json:"volume"`
	CapacityError string         `json:"capacityError"`
	ReadBytes     uint64         `json:"readBytes"`
	WriteBytes    uint64         `json:"writeBytes"`
	ReadRate      float64        `json:"readRate"`
	WriteRate     float64        `json:"writeRate"`
	Ready         bool           `json:"ready"`
	Error         string         `json:"error"`
}

type NetworkInterface struct {
	Name     string  `json:"name"`
	Received uint64  `json:"received"`
	Sent     uint64  `json:"sent"`
	Download float64 `json:"download"`
	Upload   float64 `json:"upload"`
	Up       bool    `json:"up"`
	Included bool    `json:"included"`
	Ready    bool    `json:"ready"`
}

type Network struct {
	Received   uint64             `json:"received"`
	Sent       uint64             `json:"sent"`
	Download   float64            `json:"download"`
	Upload     float64            `json:"upload"`
	Interfaces []NetworkInterface `json:"interfaces"`
	Ready      bool               `json:"ready"`
	Error      string             `json:"error"`
}

type Battery struct {
	Present       bool     `json:"present"`
	Percent       float64  `json:"percent"`
	Charging      bool     `json:"charging"`
	ExternalPower bool     `json:"externalPower"`
	Minutes       int      `json:"minutes"`
	Health        *float64 `json:"health"`
	Cycles        *int     `json:"cycles"`
	Watts         *float64 `json:"watts"`
	SystemWatts   *float64 `json:"systemWatts"`
	Temperature   *float64 `json:"temperature"`
	Error         string   `json:"error"`
}

type Process struct {
	Identity   string  `json:"identity"`
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	Path       string  `json:"path"`
	App        string  `json:"app"`
	CPU        float64 `json:"cpu"`
	CPUReady   bool    `json:"cpuReady"`
	IOReady    bool    `json:"ioReady"`
	IOError    string  `json:"ioError"`
	Memory     uint64  `json:"memory"`
	ReadBytes  uint64  `json:"readBytes"`
	WriteBytes uint64  `json:"writeBytes"`
	ReadRate   float64 `json:"readRate"`
	WriteRate  float64 `json:"writeRate"`
	Threads    int     `json:"threads"`
	UserID     uint32  `json:"userId"`
	Own        bool    `json:"own"`
}

type ProcessList struct {
	Items     []Process `json:"items"`
	Total     int       `json:"total"`
	Skipped   int       `json:"skipped"`
	UpdatedAt time.Time `json:"updatedAt"`
	Error     string    `json:"error"`
}

type AppUsage struct {
	Name      string  `json:"name"`
	Path      string  `json:"path"`
	Processes int     `json:"processes"`
	CPU       float64 `json:"cpu"`
	CPUReady  bool    `json:"cpuReady"`
	Memory    uint64  `json:"memory"`
}

type Snapshot struct {
	UpdatedAt      time.Time  `json:"updatedAt"`
	CPU            CPU        `json:"cpu"`
	Memory         Memory     `json:"memory"`
	GPU            GPU        `json:"gpu"`
	Disk           Disk       `json:"disk"`
	Network        Network    `json:"network"`
	Battery        Battery    `json:"battery"`
	ProcessCount   int        `json:"processCount"`
	ProcessSkipped int        `json:"processSkipped"`
	ProcessError   string     `json:"processError"`
	TopCPU         []Process  `json:"topCPU"`
	TopMemory      []AppUsage `json:"topMemory"`
	UptimeSeconds  float64    `json:"uptimeSeconds"`
}

type Trend struct {
	At        time.Time `json:"at"`
	CPU       *float64  `json:"cpu"`
	Memory    *float64  `json:"memory"`
	GPU       *float64  `json:"gpu"`
	DiskRead  *float64  `json:"diskRead"`
	DiskWrite *float64  `json:"diskWrite"`
	Download  *float64  `json:"download"`
	Upload    *float64  `json:"upload"`
	Battery   *float64  `json:"battery"`
	Power     *float64  `json:"power"`
}

type Machine struct {
	Name             string `json:"name"`
	OSBuild          string `json:"osBuild"`
	Kernel           string `json:"kernel"`
	SerialNumber     string `json:"serialNumber"`
	PerformanceCores *int   `json:"performanceCores"`
	EfficiencyCores  *int   `json:"efficiencyCores"`
	GPUCores         *int   `json:"gpuCores"`
	Hostname         string `json:"hostname"`
	Model            string `json:"model"`
	Chip             string `json:"chip"`
	OS               string `json:"os"`
	Architecture     string `json:"architecture"`
	LogicalCPUs      int    `json:"logicalCPUs"`
	PhysicalCPUs     int    `json:"physicalCPUs"`
	Memory           uint64 `json:"memory"`
	Platform         string `json:"platform"`
	Error            string `json:"error"`
}

type cpuTicks struct{ user, system, idle, nice uint64 }
type ioCounter struct {
	id          string
	read, write uint64
	up          bool
	included    bool
}
type rawProcess struct {
	Process
	start, cpuTime uint64
}
type rawSample struct {
	cpu                               []cpuTicks
	cpuWrap32                         bool
	memory                            Memory
	gpu                               GPU
	disks, networks                   []ioCounter
	battery                           Battery
	uptime                            float64
	cpuError, diskError, networkError string
}
