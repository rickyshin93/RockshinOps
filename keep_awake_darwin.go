package main

/*
#cgo LDFLAGS: -framework CoreFoundation -framework IOKit
#include <IOKit/pwr_mgt/IOPMLib.h>

static IOReturn take_idle_sleep_assertion(double seconds, IOPMAssertionID *id) {
	return IOPMAssertionCreateWithDescription(kIOPMAssertionTypePreventUserIdleSystemSleep,
		CFSTR("RockshinOps: Keep Awake"), NULL, NULL, NULL, seconds,
		kIOPMAssertionTimeoutActionTurnOff, id);
}
*/
import "C"

import (
	"fmt"
	"time"
)

func takeIdleSleepAssertion(duration time.Duration) (uint32, error) {
	var id C.IOPMAssertionID
	status := C.take_idle_sleep_assertion(C.double(duration.Seconds()), &id)
	if status != C.kIOReturnSuccess {
		return 0, fmt.Errorf("IOPMAssertionCreateWithDescription: 0x%08x", uint32(status))
	}
	return uint32(id), nil
}

func releaseIdleSleepAssertion(id uint32) error {
	if status := C.IOPMAssertionRelease(C.IOPMAssertionID(id)); status != C.kIOReturnSuccess {
		return fmt.Errorf("IOPMAssertionRelease: 0x%08x", uint32(status))
	}
	return nil
}
