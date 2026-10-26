package storage

import (
	"errors"
	"sync"
	"testing"
)

func TestRefreshFailureDoesNotPresentStaleCapacity(t *testing.T) {
	volume := Volume{Name: "test disk", Total: 500_000_000_000, Available: 90_000_000_000}
	failure := false
	m := NewMonitor(func() (Volume, error) {
		if failure {
			return volume, errors.New("read failed")
		}
		return volume, nil
	})
	if m.Current().Error == "" {
		t.Fatal("an unsampled disk must not appear ready")
	}
	first := m.Refresh()
	if first.Error != "" || first.Volume != volume || first.UpdatedAt.IsZero() {
		t.Fatalf("first sample: %+v", first)
	}
	failure = true
	failed := m.Refresh()
	if failed.Error != "read failed" || failed.Volume != (Volume{}) {
		t.Fatalf("stale capacity leaked on failure: %+v", failed)
	}
	if m.Current() != failed {
		t.Fatal("current result did not expose the failure")
	}
	failure = false
	volume.Available = 80_000_000_000
	recovered := m.Refresh()
	if recovered.Error != "" || recovered.Volume.Available != volume.Available {
		t.Fatalf("did not recover: %+v", recovered)
	}
}

func TestConcurrentReadsAreSerialized(t *testing.T) {
	reads := 0
	m := NewMonitor(func() (Volume, error) { reads++; return Volume{Total: 100, Available: uint64(reads)}, nil })
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() { defer wg.Done(); m.Refresh(); m.Current() }()
	}
	wg.Wait()
	if got := m.Current().Volume.Available; got != 20 {
		t.Fatalf("lost sample: %d", got)
	}
}

func TestCapacityUnits(t *testing.T) {
	for _, tc := range []struct {
		bytes uint64
		want  string
	}{
		{0, "0.0 GB"}, {500_000_000, "500.0 MB"}, {1_000_000_000, "1.0 GB"},
		{90_671_276_032, "90.7 GB"}, {1_234_000_000_000, "1.23 TB"},
	} {
		if got := FormatCapacity(tc.bytes); got != tc.want {
			t.Errorf("%d: got %q, want %q", tc.bytes, got, tc.want)
		}
	}
}
