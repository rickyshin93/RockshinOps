// Package devices reads and controls local Wi-Fi, paired Bluetooth and audio outputs.
package devices

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type WiFiState struct {
	Available bool     `json:"available"`
	Powered   bool     `json:"powered"`
	Connected bool     `json:"connected"`
	Interface string   `json:"interface"`
	SSID      string   `json:"ssid"`
	Location  string   `json:"location"`
	Signal    *int     `json:"signal"`
	Noise     *int     `json:"noise"`
	Rate      *float64 `json:"rate"`
	Channel   *int     `json:"channel"`
	Band      string   `json:"band"`
	Width     *int     `json:"width"`
	PHY       string   `json:"phy"`
	Security  string   `json:"security"`
	Source    string   `json:"source"`
}
type WiFiNetwork struct {
	SSID    string `json:"ssid"`
	Signal  int    `json:"signal"`
	Channel int    `json:"channel"`
	Band    string `json:"band"`
	Secure  bool   `json:"secure"`
}
type BluetoothDevice struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Connected bool   `json:"connected"`
	Battery   *int   `json:"battery"`
}
type BluetoothState struct {
	Authorization string            `json:"authorization"`
	Available     bool              `json:"available"`
	Powered       bool              `json:"powered"`
	Devices       []BluetoothDevice `json:"devices"`
	Source        string            `json:"source"`
}
type AudioOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type AudioState struct {
	Outputs        []AudioOutput `json:"outputs"`
	DefaultID      string        `json:"defaultId"`
	Volume         *float64      `json:"volume"`
	VolumeSettable bool          `json:"volumeSettable"`
	Error          string        `json:"error"`
	Source         string        `json:"source"`
}
type DeviceState struct {
	Supported bool           `json:"supported"`
	UpdatedAt time.Time      `json:"updatedAt"`
	WiFi      WiFiState      `json:"wifi"`
	Bluetooth BluetoothState `json:"bluetooth"`
	Audio     AudioState     `json:"audio"`
	Error     string         `json:"error"`
}
type request struct {
	Action string  `json:"action"`
	ID     string  `json:"id"`
	Value  float64 `json:"value"`
}
type response struct {
	State    DeviceState   `json:"state"`
	Networks []WiFiNetwork `json:"networks"`
	Error    string        `json:"error"`
}

// Monitor starts on first page access and samples independently of WebView timers.
type Monitor struct {
	once      sync.Once
	stopOnce  sync.Once
	stop      chan struct{}
	interval  time.Duration
	operation sync.Mutex
	mu        sync.RWMutex
	state     DeviceState
	call      func(request) (response, error)
}

func NewMonitor(interval time.Duration) *Monitor {
	return &Monitor{stop: make(chan struct{}), interval: interval, call: native}
}
func (m *Monitor) Start() {
	m.once.Do(func() {
		m.Refresh()
		go func() {
			ticker := time.NewTicker(m.interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					m.Refresh()
				case <-m.stop:
					return
				}
			}
		}()
	})
}
func (m *Monitor) Stop() { m.stopOnce.Do(func() { close(m.stop) }) }
func (m *Monitor) Snapshot() DeviceState {
	m.Start()
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}
func (m *Monitor) refreshLocked() DeviceState {
	result, err := m.call(request{Action: "read"})
	state := result.State
	state.UpdatedAt = time.Now()
	if err != nil {
		state.Error = err.Error()
	}
	m.mu.Lock()
	m.state = state
	m.mu.Unlock()
	return state
}
func (m *Monitor) Refresh() DeviceState {
	m.operation.Lock()
	defer m.operation.Unlock()
	return m.refreshLocked()
}
func (m *Monitor) Scan() ([]WiFiNetwork, error) {
	m.operation.Lock()
	defer m.operation.Unlock()
	result, err := m.call(request{Action: "scan"})
	return result.Networks, err
}
func validateAction(action, id string, value float64) error {
	switch action {
	case "location", "bluetoothPermission":
	case "connect", "disconnect", "output":
		if id == "" {
			return fmt.Errorf("device ID is required")
		}
	case "volume":
		if id == "" || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
			return fmt.Errorf("volume requires a device ID and a value from 0 to 100")
		}
	default:
		return fmt.Errorf("unknown device action %q", action)
	}
	return nil
}
func (m *Monitor) Act(action, id string, value float64) (DeviceState, error) {
	if err := validateAction(action, id, value); err != nil {
		return DeviceState{}, err
	}
	m.operation.Lock()
	defer m.operation.Unlock()
	_, err := m.call(request{Action: action, ID: id, Value: value})
	state := m.refreshLocked()
	return state, err
}
