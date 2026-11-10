package inputsource

/*
#cgo LDFLAGS: -framework Carbon
#include <stdint.h>
int32_t rockshinops_use_system_keyboard(void **selection);
int32_t rockshinops_restore_keyboard(void *selection);
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// UseSystemKeyboard selects an enabled Apple ASCII keyboard layout. Call this
// and its returned restore function on the native UI thread.
func UseSystemKeyboard() (func() error, error) {
	var selection unsafe.Pointer
	if status := C.rockshinops_use_system_keyboard(&selection); status != 0 {
		return nil, fmt.Errorf("select system keyboard: OSStatus %d", status)
	}
	return func() error {
		if status := C.rockshinops_restore_keyboard(selection); status != 0 {
			return fmt.Errorf("restore input source: OSStatus %d", status)
		}
		selection = nil
		return nil
	}, nil
}
