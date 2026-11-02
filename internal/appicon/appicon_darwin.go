package appicon

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework UniformTypeIdentifiers
#include <stdlib.h>
void *rockshinops_app_icon_png(const char *path, int pixels, size_t *length);
void *rockshinops_type_icon_png(const char *ext, int folder, int pixels, size_t *length);
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

// renderType draws the generic icon of a folder, or of files with extension ext.
func renderType(ext string, folder bool, pixels int) ([]byte, error) {
	var length C.size_t
	var cext *C.char
	if ext != "" {
		cext = C.CString(ext)
		defer C.free(unsafe.Pointer(cext))
	}
	isFolder := C.int(0)
	if folder {
		isFolder = 1
	}
	return goPNG(C.rockshinops_type_icon_png(cext, isFolder, C.int(pixels), &length), length)
}

func renderPNG(path *C.char, pixels int) ([]byte, error) {
	var length C.size_t
	return goPNG(C.rockshinops_app_icon_png(path, C.int(pixels), &length), length)
}

func goPNG(data unsafe.Pointer, length C.size_t) ([]byte, error) {
	if data == nil {
		return nil, errors.New("icon unavailable")
	}
	defer C.free(data)
	return C.GoBytes(data, C.int(length)), nil
}
