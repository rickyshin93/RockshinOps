package main

import (
	"fmt"
	"runtime"

	"github.com/egoist/mygo"
	"rockshinops/internal/devices"
)

func (s *System) Devices() devices.DeviceState             { return s.devices.Snapshot() }
func (s *System) RefreshDevices() devices.DeviceState      { s.devices.Start(); return s.devices.Refresh() }
func (s *System) ScanWiFi() ([]devices.WiFiNetwork, error) { return s.devices.Scan() }
func (s *System) DeviceAction(action, id string, value float64) (devices.DeviceState, error) {
	return s.devices.Act(action, id, value)
}
func (s *System) OpenDeviceSettings(section string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("device settings are available on macOS only")
	}
	urls := map[string]string{
		"wifi":             "x-apple.systempreferences:com.apple.wifi-settings-extension",
		"bluetooth":        "x-apple.systempreferences:com.apple.BluetoothSettings",
		"audio":            "x-apple.systempreferences:com.apple.Sound-Settings.extension",
		"location":         "x-apple.systempreferences:com.apple.settings.PrivacySecurity.extension?Privacy_LocationServices",
		"bluetoothPrivacy": "x-apple.systempreferences:com.apple.settings.PrivacySecurity.extension?Privacy_Bluetooth",
	}
	url, ok := urls[section]
	if !ok {
		return fmt.Errorf("unknown device settings section %q", section)
	}
	return mygo.Shell.OpenExternal(url)
}
