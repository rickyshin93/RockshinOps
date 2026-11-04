package metrics

/*
#include "sensors_darwin.h"
#include "native_darwin.h"
*/
import "C"

import (
	"log"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

func smcKey(name string) uint32 {
	return uint32(name[0])<<24 | uint32(name[1])<<16 | uint32(name[2])<<8 | uint32(name[3])
}

func smcName(key uint32) string {
	return string([]byte{byte(key >> 24), byte(key >> 16), byte(key >> 8), byte(key)})
}

type smcEntry struct {
	key, size uint32
	kind      string
}

func smcInfo(key uint32) (smcEntry, bool) {
	var kind, size C.uint32_t
	if C.lo_smc_key_info(C.uint32_t(key), &kind, &size) != 0 || size == 0 || size > 32 {
		return smcEntry{}, false
	}
	return smcEntry{key: key, size: uint32(size), kind: smcName(uint32(kind))}, true
}

func smcValue(entry smcEntry) (float64, bool) {
	var bytes [32]byte
	if C.lo_smc_read(C.uint32_t(entry.key), C.uint32_t(entry.size), (*C.uint8_t)(unsafe.Pointer(&bytes[0]))) != 0 {
		return 0, false
	}
	return decodeSMC(entry.kind, bytes[:entry.size])
}

type hidReading struct {
	name    string
	celsius float64
	valid   bool
}

func hidTemperatures() ([]hidReading, bool) {
	var items *C.lo_hid_temperature
	var count C.int
	if C.lo_hid_temperatures(&items, &count) != 0 {
		return nil, false
	}
	defer C.lo_free(unsafe.Pointer(items))
	result := make([]hidReading, 0, int(count))
	for _, item := range unsafe.Slice(items, int(count)) {
		result = append(result, hidReading{C.GoString(&item.name[0]), float64(item.celsius), item.valid != 0})
	}
	return result, true
}

var sensorMu sync.Mutex
var sensorInfo = map[string]smcEntry{}

var sensorChip = sync.OnceValue(func() string {
	chip, err := syscall.Sysctl("machdep.cpu.brand_string")
	if err != nil {
		log.Printf("sensor chip identification: %v", err)
	}
	return chip // An unknown model leaves component temperatures unavailable.
})

func readSMC(name string) (*float64, bool) {
	entry, known := sensorInfo[name]
	if !known {
		var present bool
		entry, present = smcInfo(smcKey(name))
		if !present {
			return nil, false
		}
		sensorInfo[name] = entry
	}
	value, ok := smcValue(entry)
	if !ok {
		return nil, true
	}
	return &value, true
}

// CollectSensors reads the SMC and HID clients serially; these clients are
// shared for the process lifetime and are not safe for concurrent access.
func CollectSensors() Sensors {
	sensorMu.Lock()
	defer sensorMu.Unlock()
	result := emptySensors()
	if C.lo_smc_open() == 0 {
		result = collectSMCSensors(sensorChip(), readSMC)
	}
	if readings, ok := hidTemperatures(); ok {
		if value, ok := hottestDie(readings); ok {
			result.DieTemperature = &value
			result.TemperatureSource = "IOHIDEventSystem: PMU tdie (max)"
		}
	}

	return result
}

func hottestDie(readings []hidReading) (float64, bool) {
	var max float64
	found := false
	for _, r := range readings {
		if !r.valid || !strings.HasPrefix(r.name, "PMU tdie") || !validTemperature(r.celsius) {
			continue
		}
		if !found || r.celsius > max {
			max, found = r.celsius, true
		}
	}
	return max, found
}
