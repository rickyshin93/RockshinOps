package metrics

import (
	"math"
	"rockshinops/internal/storage"
	"testing"
	"time"
)

func TestMappedSensorsKeepUnavailableAndZeroReadings(t *testing.T) {
	values := map[string]*float64{"Te05": ptr(51), "Tp01": ptr(55), "Tp05": ptr(-127), "Tg1U": ptr(48), "TH0x": ptr(34), "TB1T": ptr(31), "TB2T": ptr(32), "TaLP": nil, "FNum": ptr(2), "F0Ac": ptr(0), "F0Tg": ptr(1350), "F0Mn": ptr(1200), "F0Mx": ptr(5777), "F1Ac": nil, "VD0R": ptr(0), "ID0R": ptr(-0.2)}
	s := collectSMCSensors("Apple M4 Pro", func(key string) (*float64, bool) { v, ok := values[key]; return v, ok })
	summary := s.TemperatureSummary
	if *summary.CPU != 55 || *summary.GPU != 48 || *summary.SSD != 34 || *summary.Battery != 32 {
		t.Fatalf("wrong maxima: %+v", summary)
	}
	for _, r := range s.Temperatures {
		if (r.Key == "Tp05" || r.Key == "TaLP") && r.Celsius != nil {
			t.Fatalf("invalid read became valid: %+v", r)
		}
	}
	if len(s.Temperatures) != 8 {
		t.Fatalf("absent keys must be omitted: %+v", s.Temperatures)
	}
	if len(s.Fans) != 2 || *s.Fans[0].RPM != 0 || s.Fans[1].RPM != nil || *s.Fans[0].TargetRPM != 1350 || *s.Fans[0].MaxRPM != 5777 || s.Fans[1].MinRPM != nil {
		t.Fatalf("fan details lost: %+v", s.Fans)
	}
	if len(s.Electrical) != 2 || *s.Electrical[0].Value != 0 || *s.Electrical[1].Value != -0.2 {
		t.Fatalf("electrical zero/sign lost: %+v", s.Electrical)
	}
}

func TestTemperatureMappingsRespectChipGeneration(t *testing.T) {
	for _, tc := range []struct{ chip, key, kind string }{
		{"Apple M1", "Tp09", "efficiency"}, {"Apple M4 Pro", "Tp09", "performance"},
		{"Apple M4", "Tg0G", "gpu"}, {"Apple M4 Pro", "Tg0G", ""},
		{"Apple M40", "Tp09", ""}, {"", "Tp09", ""}, {"Intel Core i7", "TCAD", "cpu"},
	} {
		got := ""
		for _, c := range temperatureChannels(tc.chip) {
			if c.key == tc.key {
				got = c.kind
			}
		}
		if got != tc.kind {
			t.Errorf("%s %s = %q, want %q", tc.chip, tc.key, got, tc.kind)
		}
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), -9200, -127, 0, 1, 126} {
		if validTemperature(value) {
			t.Errorf("invalid temperature accepted: %v", value)
		}
	}
}

func TestTemperatureHistoryUsesActualSamplesAndMissingValues(t *testing.T) {
	m := NewMonitor(Options{Interval: time.Second, HistorySeconds: 60})
	m.read = func() rawSample { return rawSample{} }
	m.readProcesses = func() ([]rawProcess, int, error) { return nil, 0, nil }
	m.readVolume = func() (storage.Volume, error) { return storage.Volume{}, nil }
	m.readThermal = func() Thermal { return Thermal{} }
	m.readAssertions = func() PowerAssertions { return PowerAssertions{} }
	m.readSensors = func() Sensors { return Sensors{TemperatureSummary: TemperatureSummary{CPU: ptr(55), SSD: ptr(34)}} }
	now := time.Now()
	m.refreshAt(now, false)
	m.readSensors = emptySensors
	m.refreshAt(now.Add(time.Second), false)
	history := m.History()
	if len(history) != 2 || *history[0].CPUTemperature != 55 || *history[0].SSDTemperature != 34 || history[0].GPUTemperature != nil || history[1].CPUTemperature != nil {
		t.Fatalf("temperature history fabricated or stale: %+v", history)
	}
	if len(m.HistorySince(now)) != 1 || m.HistorySince(now)[0].SSDTemperature != nil {
		t.Fatal("incremental history lost missing sample")
	}
}
