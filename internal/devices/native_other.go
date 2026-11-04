//go:build !darwin

package devices

import "fmt"

func native(req request) (response, error) {
	if req.Action != "read" {
		return response{}, fmt.Errorf("device connectivity is available on macOS only")
	}
	return response{State: DeviceState{WiFi: WiFiState{Location: "unsupported"}, Bluetooth: BluetoothState{Authorization: "unsupported", Devices: []BluetoothDevice{}}, Audio: AudioState{Outputs: []AudioOutput{}}}}, nil
}
