package devices

import (
	"errors"
	"math"
	"sync"
	"testing"
	"time"
)

func TestDiscoveryDoesNotSendControlRequests(t *testing.T) {
	m := NewMonitor(time.Hour)
	defer m.Stop()
	var calls []string
	m.call = func(req request) (response, error) {
		calls = append(calls, req.Action)
		return response{State: DeviceState{Supported: true, WiFi: WiFiState{Location: "notDetermined"}, Bluetooth: BluetoothState{Authorization: "notDetermined"}}}, nil
	}
	first := m.Snapshot()
	second := m.Snapshot()
	if len(calls) != 1 || calls[0] != "read" {
		t.Fatalf("discovery sent %v", calls)
	}
	if !first.Supported || second.UpdatedAt != first.UpdatedAt || first.WiFi.Location != "notDetermined" {
		t.Fatalf("invalid cached state: %+v", second)
	}
}
func TestControlsValidateBeforeCrossingNativeBoundary(t *testing.T) {
	m := NewMonitor(time.Hour)
	defer m.Stop()
	m.call = func(request) (response, error) { t.Fatal("invalid request reached native"); return response{}, nil }
	for _, req := range []request{{Action: "join"}, {Action: "output"}, {Action: "connect"}, {Action: "volume", ID: "a", Value: -1}, {Action: "volume", ID: "a", Value: 101}, {Action: "volume", ID: "a", Value: math.NaN()}, {Action: "volume", ID: "a", Value: math.Inf(1)}} {
		if _, err := m.Act(req.Action, req.ID, req.Value); err == nil {
			t.Fatalf("accepted %+v", req)
		}
	}
}
func TestFailedActionReadsActualStateWithoutPretendingSuccess(t *testing.T) {
	m := NewMonitor(time.Hour)
	defer m.Stop()
	var calls []request
	zero := 0.0
	m.call = func(req request) (response, error) {
		calls = append(calls, req)
		if req.Action == "volume" {
			return response{}, errors.New("default output changed")
		}
		return response{State: DeviceState{Supported: true, Audio: AudioState{DefaultID: "new-route", Volume: &zero}}}, nil
	}
	state, err := m.Act("volume", "old-route", 0)
	if err == nil || err.Error() != "default output changed" || state.Audio.DefaultID != "new-route" || state.Audio.Volume == nil || *state.Audio.Volume != 0 {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	if len(calls) != 2 || calls[0].ID != "old-route" || calls[1].Action != "read" {
		t.Fatalf("calls=%+v", calls)
	}
}
func TestBackgroundSamplingContinuesWithoutSnapshotCalls(t *testing.T) {
	m := NewMonitor(10 * time.Millisecond)
	defer m.Stop()
	var mu sync.Mutex
	count := 0
	ready := make(chan struct{})
	m.call = func(req request) (response, error) {
		mu.Lock()
		defer mu.Unlock()
		count++
		if count == 2 {
			close(ready)
		}
		return response{State: DeviceState{Supported: true}}, nil
	}
	m.Snapshot()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("no independent sample")
	}
}
func TestNativeReadErrorRemainsVisible(t *testing.T) {
	m := NewMonitor(time.Hour)
	defer m.Stop()
	m.call = func(request) (response, error) { return response{}, errors.New("native read failed") }
	if state := m.Snapshot(); state.Error != "native read failed" || state.UpdatedAt.IsZero() {
		t.Fatalf("invalid failure state: %+v", state)
	}
}
