package metrics

import (
	"flag"
	"testing"
	"time"
)

var powerProbe = flag.Bool("power", false, "print this Mac's thermal pressure and power assertions")

// go test ./internal/metrics -run TestPowerProbe -power -v
func TestPowerProbe(t *testing.T) {
	if !*powerProbe {
		t.Skip("pass -power to read this Mac")
	}
	thermal := readThermal()
	if thermal.Error != "" {
		t.Fatalf("thermal: %s", thermal.Error)
	}
	t.Logf("thermal pressure: %s (source: %s)", thermal.Pressure, thermal.Source)
	assertions := readPowerAssertions()
	if assertions.Error != "" {
		t.Fatalf("assertions: %s", assertions.Error)
	}
	for _, a := range assertions.Items {
		held := "unknown"
		if a.Since != nil {
			held = time.Since(*a.Since).Truncate(time.Second).String()
		}
		t.Logf("pid %d %s [%s] held %s name=%q details=%q app=%q", a.PID, a.Process, a.Type, held, a.Name, a.Details, a.App)
	}
	t.Logf("%d assertions (source: %s)", len(assertions.Items), assertions.Source)
}
