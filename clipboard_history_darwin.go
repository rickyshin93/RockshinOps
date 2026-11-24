package main

/*
#cgo LDFLAGS: -framework AppKit -framework ImageIO -framework CoreGraphics
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

int64_t rockshin_clipboard_change_count(void);
char *rockshin_clipboard_read(int64_t *change, int maxText, int maxImage, int *length,
                             char **format, bool *accessRequired, bool *tooLarge);
bool rockshin_clipboard_write(const char *format, const void *bytes, int length);
char *rockshin_clipboard_preview(const char *path, int pixels, int *length);
*/
import "C"

import (
	"errors"
	"unicode/utf8"
	"unsafe"
)

func clipboardChangeCount() int64 { return int64(C.rockshin_clipboard_change_count()) }

func readClipboardChange(previous int64, maxText, maxImage int) clipboardChange {
	change := C.int64_t(previous)
	var length C.int
	var format *C.char
	var accessRequired, tooLarge C.bool
	data := C.rockshin_clipboard_read(&change, C.int(maxText), C.int(maxImage), &length, &format, &accessRequired, &tooLarge)
	result := clipboardChange{count: int64(change), accessRequired: bool(accessRequired), tooLarge: bool(tooLarge)}
	if data == nil {
		return result
	}
	defer C.free(unsafe.Pointer(data))
	content := clipboardContent{format: C.GoString(format), data: C.GoBytes(unsafe.Pointer(data), length)}
	if content.format != "txt" || utf8.Valid(content.data) {
		result.content = content
	}
	return result
}

func writeClipboardContent(content clipboardContent) error {
	if len(content.data) == 0 {
		return errors.New("clipboard item is empty")
	}
	format := C.CString(content.format)
	defer C.free(unsafe.Pointer(format))
	if !C.rockshin_clipboard_write(format, unsafe.Pointer(unsafe.SliceData(content.data)), C.int(len(content.data))) {
		return errors.New("macOS did not write the clipboard item")
	}
	return nil
}

func clipboardImagePreview(path string, pixels int) ([]byte, error) {
	name := C.CString(path)
	defer C.free(unsafe.Pointer(name))
	var length C.int
	data := C.rockshin_clipboard_preview(name, C.int(pixels), &length)
	if data == nil {
		return nil, errors.New("could not generate a clipboard image preview")
	}
	defer C.free(unsafe.Pointer(data))
	return C.GoBytes(unsafe.Pointer(data), length), nil
}
