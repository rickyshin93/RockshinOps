package trayimage

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
void *localops_tray_png(const char **symbols, const char **texts, int count, size_t *length);
*/
import "C"

import (
	"errors"
	"unsafe"
)

// Render returns a PNG 36 pixels tall (18pt at 2x) for a template status item image.
func Render(items []Item) ([]byte, error) {
	if len(items) == 0 {
		return nil, errors.New("no menu bar items")
	}
	symbols := make([]*C.char, len(items))
	texts := make([]*C.char, len(items))
	for i, item := range items {
		symbols[i] = C.CString(item.Symbol)
		texts[i] = C.CString(item.Text)
		defer C.free(unsafe.Pointer(symbols[i]))
		defer C.free(unsafe.Pointer(texts[i]))
	}
	var length C.size_t
	data := C.localops_tray_png(&symbols[0], &texts[0], C.int(len(items)), &length)
	if data == nil {
		return nil, errors.New("menu bar image unavailable")
	}
	defer C.free(data)
	return C.GoBytes(data, C.int(length)), nil
}
