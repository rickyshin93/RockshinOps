package metrics

import (
	"github.com/shirou/gopsutil/v4/disk"
	psnet "github.com/shirou/gopsutil/v4/net"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func readDisks() ([]ioCounter, error) {
	counters, err := disk.IOCounters()
	if err != nil {
		return nil, err
	}
	result := []ioCounter{}
	for name, c := range counters {
		// Whole hardware devices only: excludes partitions, dm, md and loop copies.
		if _, err := os.Stat(filepath.Join("/sys/block", name, "device")); err == nil {
			result = append(result, ioCounter{id: name, read: c.ReadBytes, write: c.WriteBytes})
		}
	}
	if len(result) == 0 {
		return nil, noCounters("Disk I/O")
	}
	return sortedCounters(result), nil
}

func readNetworks() ([]ioCounter, error) {
	counters, err := psnet.IOCounters(true)
	if err != nil {
		return nil, err
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	up := map[string]bool{}
	for _, i := range interfaces {
		up[i.Name] = i.Flags&net.FlagUp != 0
	}
	result := []ioCounter{}
	for _, c := range counters {
		_, err := os.Stat(filepath.Join("/sys/class/net", c.Name, "device"))
		result = append(result, ioCounter{id: c.Name, read: c.BytesRecv, write: c.BytesSent, up: up[c.Name], included: err == nil})
	}
	return sortedCounters(result), nil
}

func machineModel() string {
	data, err := os.ReadFile("/sys/class/dmi/id/product_name")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
