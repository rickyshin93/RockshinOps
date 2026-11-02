package metrics

import (
	"math"
	"testing"
)

func TestHottestDie(t *testing.T) {
	readings := []hidReading{{"PMU tdie1", 40, true}, {"PMU tdie2", 57, true}, {"PMU tdev1", 90, true}, {"PMU tdie3", -9200, true}, {"PMU tdie4", 130, true}, {"PMU tdie5", math.NaN(), true}, {"PMU tdie6", 100, false}}
	if got, ok := hottestDie(readings); !ok || got != 57 {
		t.Fatalf("got %v %v", got, ok)
	}
	if _, ok := hottestDie(readings[2:]); ok {
		t.Fatal("invalid sensors must be unavailable")
	}
}

func TestSensorReadSmoke(t *testing.T) {
	s := CollectSensors()
	if s.DieTemperature != nil {
		t.Logf("die=%.2f °C source=%s", *s.DieTemperature, s.TemperatureSource)
	}
	t.Logf("power=%+v fans=%+v", s.PowerRails, s.Fans)
}
