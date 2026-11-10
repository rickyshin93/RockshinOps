package main

import (
	"errors"
	"testing"
)

func TestCalculatorKeyboardFollowsPageAndWindow(t *testing.T) {
	selected, restored := 0, 0
	c := calculatorInput{selectKeyboard: func() (func() error, error) {
		selected++
		return func() error { restored++; return nil }, nil
	}}
	for _, step := range []struct {
		name               string
		page, focus        bool
		selected, restored int
	}{
		{"overview", false, true, 0, 0},
		{"calculator", true, true, 1, 0},
		{"duplicate focus", true, true, 1, 0},
		{"other app or launcher", true, false, 1, 1},
		{"return", true, true, 2, 1},
		{"leave calculator", false, true, 2, 2},
		{"navigate while hidden", true, false, 2, 2},
		{"reopen", true, true, 3, 2},
		{"quit", false, false, 3, 3},
	} {
		c.page, c.focused = step.page, step.focus
		if err := c.sync(); err != nil {
			t.Fatal(err)
		}
		if selected != step.selected || restored != step.restored {
			t.Fatalf("%s: selected/restored = %d/%d, want %d/%d", step.name, selected, restored, step.selected, step.restored)
		}
	}
}

func TestCalculatorKeyboardPreservesFailedRestoration(t *testing.T) {
	failure := errors.New("source unavailable")
	c := calculatorInput{page: true, focused: true, selectKeyboard: func() (func() error, error) { return nil, failure }}
	if err := c.sync(); !errors.Is(err, failure) || c.restore != nil {
		t.Fatalf("selection error lost: %v", err)
	}
	c.selectKeyboard = func() (func() error, error) {
		return func() error { return failure }, nil
	}
	if err := c.sync(); err != nil {
		t.Fatal(err)
	}
	c.focused = false
	if err := c.sync(); !errors.Is(err, failure) || c.restore == nil {
		t.Fatalf("restoration error lost: %v", err)
	}
	failure = nil
	if err := c.sync(); err != nil || c.restore != nil {
		t.Fatalf("restoration did not complete: %v", err)
	}
}
