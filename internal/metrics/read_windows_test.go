package metrics

import (
	"testing"
	"unsafe"
)

func TestWindowsBatteryStatusAndLayouts(t *testing.T) {
	b := batteryFromStatus(powerStatus{AC: 1, Flags: 8, Percent: 65, Seconds: 0xffffffff})
	if !b.Present || !b.Charging || !b.ExternalPower || b.Percent != 65 || b.Minutes != -1 {
		t.Fatalf("battery: %+v", b)
	}
	if b := batteryFromStatus(powerStatus{Flags: 128}); b.Present || b.Error != "" {
		t.Fatalf("desktop without battery: %+v", b)
	}
	if b := batteryFromStatus(powerStatus{Flags: 255}); b.Error == "" {
		t.Fatal("unknown battery status hidden")
	}
	var row mibIfRow2
	if unsafe.Offsetof(row.InOctets) != 1208 || unsafe.Offsetof(row.OutOctets) != 1280 || unsafe.Sizeof(row) != 1352 {
		t.Fatalf("MIB_IF_ROW2 ABI mismatch: in=%d out=%d size=%d", unsafe.Offsetof(row.InOctets), unsafe.Offsetof(row.OutOctets), unsafe.Sizeof(row))
	}
	if unsafe.Sizeof(powerStatus{}) != 12 || unsafe.Sizeof(diskPerformance{}) != 88 {
		t.Fatal("Win32 ABI mismatch")
	}
}
