//go:build !darwin

package metrics

func CollectSensors() Sensors {
	return Sensors{PowerRails: []SensorReading{}, Fans: []FanReading{}}
}
