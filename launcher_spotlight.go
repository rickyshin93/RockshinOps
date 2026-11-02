package main

import (
	"encoding/json"
	"strings"
)

// spotlightConflict reports whether Spotlight's "Show Spotlight search"
// hot key (symbolic hot key 64) still takes Cmd+Space. data is that entry
// as JSON; a missing entry means the system default, which is enabled.
func spotlightConflict(shortcut string, data []byte, err error) bool {
	if !strings.EqualFold(shortcut, "Cmd+Space") {
		return false
	}
	if err != nil {
		return true
	}
	var entry struct {
		Enabled bool `json:"enabled"`
		Value   struct {
			Parameters []int `json:"parameters"`
		} `json:"value"`
	}
	if json.Unmarshal(data, &entry) != nil || len(entry.Value.Parameters) < 3 {
		return false
	}
	const spaceKeyCode, commandModifier = 49, 1 << 20
	return entry.Enabled && entry.Value.Parameters[1] == spaceKeyCode && entry.Value.Parameters[2] == commandModifier
}
