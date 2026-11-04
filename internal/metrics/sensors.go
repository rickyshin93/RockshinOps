package metrics

import (
	"encoding/binary"
	"math"
	"strconv"
)

// decodeSMC converts an AppleSMC payload of a four-character data type to a number.
// Fixed-point types spXY/fpXY are 16-bit big-endian with Y (hex) fraction bits;
// flt is a host-order float32 (little-endian on every supported Mac).
func decodeSMC(kind string, data []byte) (float64, bool) {
	if len(kind) != 4 {
		return 0, false
	}
	var value float64
	switch {
	case kind == "flt " && len(data) == 4:
		value = float64(math.Float32frombits(binary.LittleEndian.Uint32(data)))
	case kind == "ui8 " && len(data) == 1:
		value = float64(data[0])
	case kind == "ui16" && len(data) == 2:
		value = float64(binary.BigEndian.Uint16(data))
	case kind == "ui32" && len(data) == 4:
		value = float64(binary.BigEndian.Uint32(data))
	case kind == "si8 " && len(data) == 1:
		value = float64(int8(data[0]))
	case kind == "si16" && len(data) == 2:
		value = float64(int16(binary.BigEndian.Uint16(data)))
	case (kind[:2] == "sp" || kind[:2] == "fp") && len(data) == 2:
		fraction, err := strconv.ParseUint(kind[3:], 16, 8)
		if err != nil {
			return 0, false
		}
		raw := float64(binary.BigEndian.Uint16(data))
		if kind[0] == 's' {
			raw = float64(int16(binary.BigEndian.Uint16(data)))
		}
		value = raw / float64(uint(1)<<fraction)
	default:
		return 0, false
	}
	return value, !math.IsNaN(value) && !math.IsInf(value, 0)
}

// Sensors contains read-only hardware readings. Missing values are unavailable,
// rather than measurements of zero.
type Sensors struct {
	DieTemperature     *float64             `json:"dieTemperature"`
	TemperatureSource  string               `json:"temperatureSource"`
	PowerRails         []SensorReading      `json:"powerRails"`
	Fans               []FanReading         `json:"fans"`
	Temperatures       []TemperatureReading `json:"temperatures"`
	TemperatureSummary TemperatureSummary   `json:"temperatureSummary"`
	Electrical         []ElectricalReading  `json:"electrical"`
}

type TemperatureReading struct {
	Key       string   `json:"key"`
	Component string   `json:"component"`
	Kind      string   `json:"kind"`
	Celsius   *float64 `json:"celsius"`
	Source    string   `json:"source"`
}

// Summary values are the maximum available mapped channel for each component.
type TemperatureSummary struct {
	CPU     *float64 `json:"cpu"`
	GPU     *float64 `json:"gpu"`
	SSD     *float64 `json:"ssd"`
	Battery *float64 `json:"battery"`
}

type ElectricalReading struct {
	Key    string   `json:"key"`
	Kind   string   `json:"kind"`
	Value  *float64 `json:"value"`
	Source string   `json:"source"`
}

type SensorReading struct {
	Key    string  `json:"key"`
	Value  float64 `json:"value"`
	Source string  `json:"source"`
}

type FanReading struct {
	Index     int      `json:"index"`
	RPM       *float64 `json:"rpm"`
	TargetRPM *float64 `json:"targetRPM"`
	MinRPM    *float64 `json:"minRPM"`
	MaxRPM    *float64 `json:"maxRPM"`
	Source    string   `json:"source"`
}
