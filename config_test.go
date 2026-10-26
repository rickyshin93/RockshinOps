package main

import "testing"

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
