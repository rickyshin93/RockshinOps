package menubar

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework ScreenCaptureKit
#include <stdlib.h>
char *rockshin_menubar_request(const char *request);
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"
)

//export rockshin_menubar_action
func rockshin_menubar_action(action *C.char) {
	command := C.GoString(action)
	// Return to AppKit before the handler dispatches window work to its main queue.
	go actionHandler(command)
}

// Native serializes AppKit work on the main queue; it must not be called while holding a UI lock.
func Native(action string, payload any) (State, error) {
	data, err := json.Marshal(map[string]any{"action": action, "payload": payload})
	if err != nil {
		return State{}, err
	}
	input := C.CString(string(data))
	defer C.free(unsafe.Pointer(input))
	result := C.rockshin_menubar_request(input)
	defer C.free(unsafe.Pointer(result))
	return decodeNativeState(action, C.GoString(result))
}

func decodeNativeState(action, result string) (State, error) {
	var state State
	if err := json.Unmarshal([]byte(result), &state); err != nil {
		return state, err
	}
	// A state read carries the last action error for display; it is still a valid snapshot.
	if state.Error != "" && action != "state" {
		return state, fmt.Errorf("%s", state.Error)
	}
	return state, nil
}
