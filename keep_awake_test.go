package main

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"rockshinops/internal/localization"
)

func TestAwakeCountdownShowsRemainingAndLocalEnd(t *testing.T) {
	t.Cleanup(func() { localization.SetLanguage("en") })
	now := time.Date(2026, 11, 24, 21, 30, 0, 100_000_000, time.Local)
	for _, tc := range []struct {
		language string
		end      time.Time
		want     string
	}{
		{"en", now.Add(time.Hour + 30*time.Minute + 30*time.Second), "1h 30min 30s remaining · Ends at 23:00"},
		{"zh", now.Add(time.Hour), "剩余 1h 0min 0s · 22:30 结束"},
		{"zh", now.Add(59*time.Minute + 59*time.Second), "剩余 59min 59s · 22:29 结束"},
		{"en", now.Add(59 * time.Second), "59s remaining · Ends at 21:30"},
		{"zh", now.Add(200 * time.Millisecond), "剩余 1s · 21:30 结束"},
		{"zh", now.Add(3 * time.Hour), "剩余 3h 0min 0s · 11月25日 00:30 结束"},
		{"en", now.Add(3 * time.Hour), "3h 0min 0s remaining · Ends at Nov 25, 00:30"},
	} {
		localization.SetLanguage(tc.language)
		if got := awakeCountdownLabel(now, tc.end); got != tc.want {
			t.Errorf("%s countdown = %q, want %q", tc.language, got, tc.want)
		}
	}
}

func TestKeepAwakeGroupsHourPresets(t *testing.T) {
	localization.SetLanguage("en")
	cfg, err := parseConfig(configJSON)
	if err != nil {
		t.Fatal(err)
	}
	awake := newKeepAwake(cfg.KeepAwakeMinutes, cfg.KeepAwakeExtendMinutes)
	awake.localize()
	var entries []string
	var hours *mygo.MenuItem
	for _, item := range awake.menu.Submenu {
		if item.IsVisible() {
			entries = append(entries, item.Label)
			if item.Label == "唤醒小时菜单" {
				hours = item
			}
		}
	}
	if want := []string{"30 min", "唤醒小时菜单", "无限期"}; !slices.Equal(entries, want) {
		t.Fatalf("idle menu labels = %v, want %v", entries, want)
	}

	entries = nil
	for _, item := range hours.Submenu {
		if item.Type == mygo.MenuItemSeparator {
			entries = append(entries, "|")
		} else {
			entries = append(entries, item.Label)
		}
	}
	want := []string{"1 hour", "2 hours", "3 hours", "4 hours", "5 hours", "6 hours", "7 hours", "8 hours",
		"9 hours", "|", "10 hours", "11 hours", "12 hours", "24 hours"}
	if !slices.Equal(entries, want) {
		t.Errorf("hour menu labels = %v, want %v", entries, want)
	}
}
