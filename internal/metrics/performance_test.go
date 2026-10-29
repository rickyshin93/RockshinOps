package metrics

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"rockshinops/internal/storage"
)

func TestIncrementalHistoryRetentionAndIsolation(t *testing.T) {
	m := performanceMonitor()
	now := time.Now()
	for i := 0; i < 1800; i++ {
		m.refreshAt(now.Add(time.Duration(i)*2*time.Second), false)
	}
	last := m.History()[1798]
	delta := m.HistorySince(last.At)
	if len(delta) != 1 || !delta[0].At.After(last.At) {
		t.Fatal("delta must include only newer samples")
	}
	fullJSON, _ := json.Marshal(m.History())
	deltaJSON, _ := json.Marshal(delta)
	t.Logf("history payload: full=%d bytes delta=%d bytes", len(fullJSON), len(deltaJSON))
	delta[0].At = time.Time{}
	if m.HistorySince(last.At)[0].At.IsZero() {
		t.Fatal("returned slice changed retained history")
	}
	if len(m.HistorySince(now.Add(2*time.Hour))) != 0 {
		t.Fatal("future cursor returned old data")
	}
}

func performanceMonitor() *Monitor {
	m := NewMonitor(Options{Interval: 2 * time.Second, DiskInterval: 30 * time.Second, ProcessInterval: 5 * time.Second, HistorySeconds: 3600})
	m.read = func() rawSample { return rawSample{memory: Memory{Total: 1000, Used: 500}} }
	m.readVolume = func() (storage.Volume, error) { return storage.Volume{Total: 1000}, nil }
	items := make([]rawProcess, 850)
	for i := range items {
		items[i] = rawProcess{Process: Process{PID: i + 1, App: fmt.Sprintf("/app-%d.app", i%100), Memory: uint64(i * 100)}, start: 1}
	}
	m.readProcesses = func() ([]rawProcess, int, error) { return items, len(items), nil }
	return m
}

func TestPublishedMetricsRemainReadableDuringCollection(t *testing.T) {
	m := performanceMonitor()
	now := time.Now()
	m.refreshAt(now, true)
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	m.read = func() rawSample { close(started); <-release; return rawSample{} }
	go func() { m.refreshAt(now.Add(2*time.Second), false); close(done) }()
	<-started
	defer func() { close(release); <-done }()
	read := make(chan bool, 1)
	go func() {
		read <- m.Current().UpdatedAt.Equal(now) && len(m.History()) == 1 && len(m.Processes().Items) == 850
	}()
	select {
	case ok := <-read:
		if !ok {
			t.Fatal("published data changed before collection completed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("UI reads waited for native sampling to finish")
	}
}

func BenchmarkMonitorBetweenProcessSamples(b *testing.B) {
	m := performanceMonitor()
	now := time.Now()
	m.refreshAt(now, true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Isolate a regular tick without resampling the process list.
		m.history = m.history[:0]
		m.refreshAt(now.Add(time.Second), false)
	}
}

func BenchmarkProcessSampleBookkeeping(b *testing.B) {
	m := performanceMonitor()
	now := time.Now()
	m.refreshProcesses(now)
	b.ResetTimer()
	for b.Loop() {
		now = now.Add(6 * time.Second)
		m.refreshProcesses(now)
	}
}
