package devices

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework CoreWLAN -framework CoreLocation -framework CoreBluetooth -framework IOBluetooth -framework CoreAudio -framework AudioToolbox -framework IOKit
#include <stdlib.h>
char *rockshin_devices_request(const char *input);
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"
)

func native(req request) (response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return response{}, err
	}
	input := C.CString(string(data))
	defer C.free(unsafe.Pointer(input))
	output := C.rockshin_devices_request(input)
	defer C.free(unsafe.Pointer(output))
	var result response
	if err = json.Unmarshal([]byte(C.GoString(output)), &result); err != nil {
		return result, fmt.Errorf("device response: %w", err)
	}
	if result.Error != "" {
		return result, fmt.Errorf("%s", result.Error)
	}
	return result, nil
}
