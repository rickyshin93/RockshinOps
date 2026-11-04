//go:build !darwin

package metrics

func CollectSensors() Sensors {
	return emptySensors()
}
