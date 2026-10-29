package appicon

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
void *localops_app_icon_png(const char *path, int pixels, size_t *length);
*/
import "C"

import (
	"errors"
	"os"
	"unsafe"
)

func render(path string, pixels int) ([]byte, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return renderPNG(cpath, pixels)
}

func renderComputer() ([]byte, error) { return renderPNG(nil, 256) }

func renderPNG(path *C.char, pixels int) ([]byte, error) {
	var length C.size_t
	data := C.localops_app_icon_png(path, C.int(pixels), &length)
	if data == nil {
		return nil, errors.New("icon unavailable")
	}
	defer C.free(data)
	return C.GoBytes(data, C.int(length)), nil
}
