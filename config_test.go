package main

import (
	"encoding/json"
	"testing"
)

func TestConfigRequiresExplicitPositiveRefreshInterval(t *testing.T) {
	for _, input := range []string{`{}`, `{"sampleIntervalSeconds":0}`, `{"sampleIntervalSeconds":-1}`, `invalid`} {
		if _, err := parseConfig([]byte(input)); err == nil {
			t.Errorf("accepted invalid config: %s", input)
		}
	}
	c, err := parseConfig(configJSON)
	if err != nil {
		t.Fatal(err)
	}
	if c.SampleIntervalSeconds != 30 {
		t.Fatalf("expected documented refresh interval, got %d", c.SampleIntervalSeconds)
	}
}

func TestConfigRequiresUpdateInterval(t *testing.T) {
	var values map[string]any
	if err := json.Unmarshal(configJSON, &values); err != nil {
		t.Fatal(err)
	}
	for _, interval := range []any{nil, 0, -1} {
		values["updateIntervalSeconds"] = interval
		input, err := json.Marshal(values)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parseConfig(input); err == nil {
			t.Fatalf("accepted update interval %v", interval)
		}
	}
}
