package menubar

import "testing"

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
