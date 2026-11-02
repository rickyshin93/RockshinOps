package metrics

import (
	"testing"
	"time"

	"rockshinops/internal/storage"
)

func TestThermalEveryRefreshAssertionsOnTheirOwnInterval(t *testing.T) {
	m := NewMonitor(Options{Interval: 2 * time.Second, DiskInterval: time.Minute, ProcessInterval: time.Minute, PowerAssertionInterval: 10 * time.Second, HistorySeconds: 60})
	m.read = func() rawSample { return rawSample{} }
	m.readProcesses = func() ([]rawProcess, int, error) { return nil, 0, nil }
	m.readVolume = func() (storage.Volume, error) { return storage.Volume{}, nil }
	thermalReads, assertionReads := 0, 0
	m.readThermal = func() Thermal { thermalReads++; return Thermal{Pressure: "nominal"} }
	m.readAssertions = func() PowerAssertions {
		assertionReads++
		return PowerAssertions{Items: []PowerAssertion{{PID: assertionReads}}}
	}
	now := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	first := m.refreshAt(now, false)
	second := m.refreshAt(now.Add(2*time.Second), false)
	if thermalReads != 2 || assertionReads != 1 || second.Thermal.Pressure != "nominal" {
		t.Fatalf("thermal %d, assertions %d", thermalReads, assertionReads)
	}
	if second.PowerAssertions.Items[0].PID != 1 || !second.PowerAssertions.UpdatedAt.Equal(first.PowerAssertions.UpdatedAt) {
		t.Fatalf("assertions must be reused within their interval: %+v", second.PowerAssertions)
	}
	third := m.refreshAt(now.Add(10*time.Second), false)
	if assertionReads != 2 || third.PowerAssertions.Items[0].PID != 2 || !third.PowerAssertions.UpdatedAt.Equal(now.Add(10*time.Second)) {
		t.Fatalf("assertions not re-read after interval: %+v", third.PowerAssertions)
	}
	m.refreshAt(now.Add(11*time.Second), true)
	if assertionReads != 3 {
		t.Fatal("forced refresh must re-read assertions")
	}
}
