package metrics

import (
	"fmt"
	"rockshinops/internal/localization"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readBattery() Battery { return readLinuxBattery("/sys/class/power_supply") }

func readLinuxBattery(root string) Battery {
	b := Battery{Minutes: -1}
	supplies, err := os.ReadDir(root)
	if err != nil {
		b.Error = err.Error()
		return b
	}
	var battery map[string]string
	for _, supply := range supplies {
		data, err := os.ReadFile(filepath.Join(root, supply.Name(), "uevent"))
		if err != nil {
			b.Error = err.Error()
			continue
		}
		fields := map[string]string{}
		for _, line := range strings.Split(string(data), "\n") {
			key, value, ok := strings.Cut(line, "=")
			if ok {
				fields[strings.TrimPrefix(key, "POWER_SUPPLY_")] = value
			}
		}
		if fields["TYPE"] == "Battery" && fields["SCOPE"] != "Device" && fields["PRESENT"] != "0" {
			// One battery panel: select the first system battery in sysfs name order.
			if battery == nil {
				battery = fields
			}
		} else if fields["ONLINE"] == "1" {
			b.ExternalPower = true
		}
	}
	if battery == nil {
		return b
	}
	b.Present = true
	percent, err := strconv.ParseFloat(battery["CAPACITY"], 64)
	if err != nil || percent < 0 || percent > 100 {
		b.Error = localization.T("Battery charge unavailable")
		return b
	}
	b.Percent = percent
	b.Charging = battery["STATUS"] == "Charging"
	if battery["STATUS"] == "" {
		b.Error = localization.T("Battery status unavailable")
	}
	optional := func(key string) *float64 {
		value, err := strconv.ParseFloat(battery[key], 64)
		if err != nil || value < 0 {
			return nil
		}
		return &value
	}
	if value := optional("TEMP"); value != nil {
		b.Temperature = ptr(*value / 10)
	}
	if value := optional("CYCLE_COUNT"); value != nil {
		v := int(*value)
		b.Cycles = &v
	}
	if power := optional("POWER_NOW"); power != nil {
		b.Watts = ptr(*power / 1e6)
	} else if current, voltage := optional("CURRENT_NOW"), optional("VOLTAGE_NOW"); current != nil && voltage != nil {
		b.Watts = ptr(*current * *voltage / 1e12)
	}
	for _, unit := range []string{"ENERGY", "CHARGE"} {
		full, design := optional(unit+"_FULL"), optional(unit+"_FULL_DESIGN")
		if full != nil && design != nil && *design > 0 {
			b.Health = ptr(*full / *design * 100)
			break
		}
	}
	if seconds := optional("TIME_TO_EMPTY_NOW"); seconds != nil && !b.Charging {
		b.Minutes = int(*seconds / 60)
	} else if energy := optional("ENERGY_NOW"); energy != nil && b.Watts != nil && *b.Watts > 0 && battery["STATUS"] == "Discharging" {
		b.Minutes = int(*energy / 1e6 / *b.Watts * 60)
	}
	if b.Watts != nil && battery["STATUS"] == "Discharging" {
		*b.Watts = -*b.Watts
	}
	return b
}

func readSysNumber(path string) (float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", path, err)
	}
	return value, nil
}
