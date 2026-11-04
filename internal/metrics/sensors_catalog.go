package metrics

import (
	"fmt"
	"math"
	"strings"
)

// Read-only key mappings follow Stats' Sensors/values.swift. Keys describe
// temperature channels, not a verified one-to-one mapping to physical cores.
// https://github.com/exelban/stats/blob/master/Modules/Sensors/values.swift
// Match the chip generation explicitly: the same key can change meaning.
type temperatureKeys struct{ efficiency, performance, gpu string }

var appleTemperatureKeys = map[string]temperatureKeys{
	"M1": {"Tp09 Tp0T", "Tp01 Tp05 Tp0D Tp0H Tp0L Tp0P Tp0X Tp0b", "Tg05 Tg0D Tg0L Tg0T"},
	"M2": {"Tp1h Tp1t Tp1p Tp1l", "Tp01 Tp05 Tp09 Tp0D Tp0X Tp0b Tp0f Tp0j", "Tg0f Tg0j"},
	"M3": {"Te05 Te0L Te0P Te0S", "Tf04 Tf09 Tf0A Tf0B Tf0D Tf0E Tf44 Tf49 Tf4A Tf4B Tf4D Tf4E", "Tf14 Tf18 Tf19 Tf1A Tf24 Tf28 Tf29 Tf2A"},
	"M4": {"Te05 Te0S Te09 Te0H", "Tp01 Tp05 Tp09 Tp0D Tp0V Tp0Y Tp0b Tp0e", "Tg0K Tg0L Tg0d Tg0e Tg0j Tg0k"},
}

type temperatureChannel struct{ key, component, kind string }

func temperatureChannels(chip string) []temperatureChannel {
	var channels []temperatureChannel
	add := func(keys, component, kind string) {
		for _, key := range strings.Fields(keys) {
			channels = append(channels, temperatureChannel{key, component, kind})
		}
	}
	fields := strings.Fields(chip)
	if len(fields) >= 2 && fields[0] == "Apple" {
		keys := appleTemperatureKeys[fields[1]]
		add(keys.efficiency, "cpu", "efficiency")
		add(keys.performance, "cpu", "performance")
		add(keys.gpu, "gpu", "gpu")
		if fields[1] == "M4" {
			if len(fields) == 2 {
				add("Tg0G Tg0H", "gpu", "gpu")
			} else if fields[2] == "Pro" || fields[2] == "Max" {
				add("Tg1U Tg1k", "gpu", "gpu")
			}
		}
		add("TH0x", "ssd", "nand")
		add("TaLP", "cooling", "airflowLeft")
		add("TaRF", "cooling", "airflowRight")
	} else if strings.Contains(chip, "Intel") {
		add("TC0D TCAD", "cpu", "cpu")
		add("TCGC TG0D TGDD", "gpu", "gpu")
		add("TH0A TH0B TH0C", "storage", "storage")
	}
	add("TB1T TB2T", "battery", "battery")
	add("TW0P", "cooling", "wireless")
	return channels
}

// Presence is separate from validity: an existing channel remains visible when
// a read fails. Absent keys are not presented as imaginary physical sensors.
type sensorReader func(string) (*float64, bool)

func validTemperature(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 10 && value <= 125
}

func emptySensors() Sensors {
	return Sensors{PowerRails: []SensorReading{}, Fans: []FanReading{}, Temperatures: []TemperatureReading{}, Electrical: []ElectricalReading{}}
}

func collectSMCSensors(chip string, read sensorReader) Sensors {
	result := emptySensors()
	for _, channel := range temperatureChannels(chip) {
		value, present := read(channel.key)
		if !present {
			continue
		}
		if value != nil && !validTemperature(*value) {
			value = nil
		}
		result.Temperatures = append(result.Temperatures, TemperatureReading{channel.key, channel.component, channel.kind, value, "AppleSMC"})
	}
	result.TemperatureSummary = summarizeTemperatures(result.Temperatures)
	for _, key := range []string{"PSTR", "PDTR", "PPBR", "PMVC"} {
		if value, _ := read(key); value != nil && *value >= 0 {
			result.PowerRails = append(result.PowerRails, SensorReading{key, *value, "AppleSMC"})
		}
	}
	for _, key := range []string{"VD0R", "VC0C", "VG0C", "VM0R", "ID0R", "IBAC", "IC0R", "IG0R"} {
		value, present := read(key)
		if !present {
			continue
		}
		kind := "current"
		if key[0] == 'V' {
			kind = "voltage"
			if value != nil && *value < 0 {
				value = nil
			}
		}
		result.Electrical = append(result.Electrical, ElectricalReading{key, kind, value, "AppleSMC"})
	}
	if count, _ := read("FNum"); count != nil && *count >= 0 && *count <= 16 && *count == math.Trunc(*count) {
		for i := 0; i < int(*count); i++ {
			speed := func(suffix string) *float64 {
				value, _ := read(fmt.Sprintf("F%X%s", i, suffix))
				if value != nil && *value < 0 {
					return nil
				}
				return value
			}
			result.Fans = append(result.Fans, FanReading{Index: i, RPM: speed("Ac"), TargetRPM: speed("Tg"), MinRPM: speed("Mn"), MaxRPM: speed("Mx"), Source: fmt.Sprintf("AppleSMC: F%XAc / F%XTg / F%XMn / F%XMx", i, i, i, i)})
		}
	}
	return result
}

func summarizeTemperatures(readings []TemperatureReading) TemperatureSummary {
	var summary TemperatureSummary
	for _, reading := range readings {
		if reading.Celsius == nil {
			continue
		}
		var maximum **float64
		switch reading.Component {
		case "cpu":
			maximum = &summary.CPU
		case "gpu":
			maximum = &summary.GPU
		case "ssd":
			maximum = &summary.SSD
		case "battery":
			maximum = &summary.Battery
		default:
			continue
		}
		if *maximum == nil || **maximum < *reading.Celsius {
			*maximum = reading.Celsius
		}
	}
	return summary
}
