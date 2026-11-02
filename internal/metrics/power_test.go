package metrics

import (
	"testing"
	"time"
)

func TestThermalPressureLevels(t *testing.T) {
	for state, want := range []string{"nominal", "moderate", "heavy", "trapping", "sleeping"} {
		if got, ok := thermalPressure(uint64(state)); !ok || got != want {
			t.Errorf("state %d: got %q %v, want %q", state, got, ok, want)
		}
	}
	if got, ok := thermalPressure(5); ok {
		t.Errorf("accepted unknown state 5 as %q", got)
	}
}

// Fixture mirrors IOPMCopyAssertionsByProcess entries seen via `pmset -g assertions`.
func TestPowerAssertionsFromNativeEntries(t *testing.T) {
	start := time.Date(2026, 11, 2, 7, 49, 9, 0, time.UTC)
	cf := float64(start.Unix() - cfAbsoluteEpoch)
	raw := []rawAssertion{
		{pid: 374, level: 255, assertType: "UserIsActive", name: "com.apple.iohideventsystem.queue.tickle", process: "WindowServer",
			path: "/System/Library/PrivateFrameworks/SkyLight.framework/Resources/WindowServer", start: cf + 180, startValid: true},
		{pid: 36432, level: 255, assertType: "PreventUserIdleSystemSleep", name: "caffeinate command-line tool",
			details: "caffeinate asserting for 300 secs", process: "caffeinate", path: "/usr/bin/caffeinate", start: cf, startValid: true},
		{pid: 811, level: 255, assertType: "PreventUserIdleDisplaySleep", name: "Video Wake Lock", process: "Google Chrome Helper",
			path: "/Applications/Google Chrome.app/Contents/Frameworks/Helper.app/Contents/MacOS/Google Chrome Helper", start: cf + 60, startValid: true},
		{pid: 900, level: 0, assertType: "PreventUserIdleSystemSleep", name: "released", process: "Released"},
		{pid: 901, level: 255, assertType: "BackgroundTask", name: "no start", process: "", path: "/usr/libexec/task"},
	}
	got := powerAssertions(raw)
	if len(got) != 4 {
		t.Fatalf("level-0 assertion must be dropped, got %d items: %+v", len(got), got)
	}
	first := got[0]
	if first.PID != 36432 || first.Type != "PreventUserIdleSystemSleep" || first.Name != "caffeinate command-line tool" ||
		first.Details != "caffeinate asserting for 300 secs" || first.Process != "caffeinate" || first.App != "" {
		t.Fatalf("unexpected caffeinate entry: %+v", first)
	}
	if first.Since == nil || !first.Since.Equal(start) {
		t.Fatalf("start time not converted from CFAbsoluteTime: %v", first.Since)
	}
	if got[1].PID != 811 || got[1].App != "/Applications/Google Chrome.app" {
		t.Fatalf("expected oldest-first order with outermost app bundle, got %+v", got[1])
	}
	if got[2].PID != 374 {
		t.Fatalf("expected WindowServer third, got %+v", got[2])
	}
	last := got[3]
	if last.PID != 901 || last.Since != nil || last.Process != "task" {
		t.Fatalf("missing start must stay unknown and name fall back to executable: %+v", last)
	}
}
