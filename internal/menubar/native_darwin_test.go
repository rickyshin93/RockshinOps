package menubar

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativeItemsExcludeDisplayReplicas(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "items")
	cmd := exec.Command("clang", "-fobjc-arc", "testdata/items_darwin.m", "events_darwin.m", "-framework", "Cocoa", "-framework", "ApplicationServices", "-framework", "ScreenCaptureKit", "-o", binary)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile native item regression: %v\n%s", err, output)
	}
	if output, err := exec.Command(binary).CombinedOutput(); err != nil {
		t.Fatalf("native item regression: %v\n%s", err, output)
	}
}

func TestNativeCaptureDeliveryAndRetry(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "capture")
	cmd := exec.Command("clang", "-fobjc-arc", "testdata/capture_darwin.m", "events_darwin.m", "-framework", "Cocoa", "-framework", "ApplicationServices", "-framework", "ScreenCaptureKit", "-o", binary)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile native capture regression: %v\n%s", err, output)
	}
	if output, err := exec.Command(binary).CombinedOutput(); err != nil {
		t.Fatalf("native capture regression: %v\n%s", err, output)
	}
}

func TestNativeStateReadPreservesActionErrorWithoutBlockingLayout(t *testing.T) {
	response := `{"supported":true,"error":"move failed","items":[{"id":7,"name":"Flux"}]}`
	state, err := decodeNativeState("state", response)
	if err != nil || state.Error != "move failed" || len(state.Items) != 1 {
		t.Fatalf("state must remain readable with its error: %#v, %v", state, err)
	}
	if _, err := decodeNativeState("move", response); err == nil {
		t.Fatal("failed actions must still return an error")
	}
	if _, err := decodeNativeState("state", "invalid"); err == nil {
		t.Fatal("invalid responses must still return an error")
	}
}
