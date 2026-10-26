package metrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxBatteryUnitsAndAbsentHardware(t *testing.T) {
	root := t.TempDir()
	if b := readLinuxBattery(root); b.Present || b.Error != "" {
		t.Fatalf("empty sysfs: %+v", b)
	}
	dir := filepath.Join(root, "BAT0")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	data := "POWER_SUPPLY_TYPE=Battery\nPOWER_SUPPLY_PRESENT=1\nPOWER_SUPPLY_CAPACITY=50\nPOWER_SUPPLY_STATUS=Discharging\nPOWER_SUPPLY_ENERGY_NOW=20000000\nPOWER_SUPPLY_ENERGY_FULL=40000000\nPOWER_SUPPLY_ENERGY_FULL_DESIGN=50000000\nPOWER_SUPPLY_POWER_NOW=10000000\nPOWER_SUPPLY_TEMP=325\nPOWER_SUPPLY_CYCLE_COUNT=20\n"
	if err := os.WriteFile(filepath.Join(dir, "uevent"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	b := readLinuxBattery(root)
	if !b.Present || b.Percent != 50 || b.Minutes != 120 || b.Watts == nil || *b.Watts != -10 || b.Health == nil || *b.Health != 80 || b.Temperature == nil || *b.Temperature != 32.5 {
		t.Fatalf("battery units: %+v", b)
	}
	if err := os.WriteFile(filepath.Join(dir, "uevent"), []byte("POWER_SUPPLY_TYPE=Battery\nPOWER_SUPPLY_CAPACITY=unknown\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if b := readLinuxBattery(root); b.Error == "" {
		t.Fatal("unknown charge must not become zero")
	}
}
