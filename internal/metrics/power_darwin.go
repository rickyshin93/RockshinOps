package metrics

/*
#cgo LDFLAGS: -framework CoreFoundation -framework IOKit
#include <stdlib.h>
#include <string.h>
#include <notify.h>
#include <libproc.h>
#include <libkern/OSThermalNotification.h>
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/pwr_mgt/IOPMLib.h>

static int lo_thermal_token = -1;

static uint32_t lo_thermal_pressure(uint64_t *level) {
	if (lo_thermal_token == -1) {
		uint32_t status = notify_register_check(kOSThermalNotificationPressureLevelName, &lo_thermal_token);
		if (status != NOTIFY_STATUS_OK) { lo_thermal_token = -1; return status; }
	}
	return notify_get_state(lo_thermal_token, level);
}

typedef struct {
	int pid, level, start_valid;
	double start;
	char *type, *name, *details, *process;
	char path[PROC_PIDPATHINFO_MAXSIZE];
} lo_assertion;

static char *lo_copy_string(CFDictionaryRef d, CFStringRef key) {
	CFTypeRef v = CFDictionaryGetValue(d, key);
	if (!v || CFGetTypeID(v) != CFStringGetTypeID()) return NULL;
	CFIndex size = CFStringGetMaximumSizeForEncoding(CFStringGetLength(v), kCFStringEncodingUTF8) + 1;
	char *out = malloc(size);
	if (out && !CFStringGetCString(v, out, size, kCFStringEncodingUTF8)) { free(out); out = NULL; }
	return out;
}

static int lo_copy_int(CFDictionaryRef d, CFStringRef key, int *out) {
	CFTypeRef v = CFDictionaryGetValue(d, key);
	return v && CFGetTypeID(v) == CFNumberGetTypeID() && CFNumberGetValue(v, kCFNumberIntType, out);
}

// Flattens IOPMCopyAssertionsByProcess: {pid: [assertion dictionary, ...]}.
static IOReturn lo_read_assertions(lo_assertion **out, int *count) {
	*out = NULL; *count = 0;
	CFDictionaryRef byPID = NULL;
	IOReturn status = IOPMCopyAssertionsByProcess(&byPID);
	if (status != kIOReturnSuccess) return status;
	if (!byPID) return kIOReturnSuccess;
	CFIndex pids = CFDictionaryGetCount(byPID), total = 0;
	const void **keys = malloc(sizeof(void *) * (pids + 1)), **values = malloc(sizeof(void *) * (pids + 1));
	CFDictionaryGetKeysAndValues(byPID, keys, values);
	for (CFIndex i = 0; i < pids; i++)
		if (CFGetTypeID(values[i]) == CFArrayGetTypeID()) total += CFArrayGetCount(values[i]);
	lo_assertion *items = calloc(total + 1, sizeof(lo_assertion));
	int n = 0;
	for (CFIndex i = 0; i < pids; i++) {
		int pid = 0;
		if (CFGetTypeID(keys[i]) != CFNumberGetTypeID() || !CFNumberGetValue(keys[i], kCFNumberIntType, &pid)) continue;
		if (CFGetTypeID(values[i]) != CFArrayGetTypeID()) continue;
		char path[PROC_PIDPATHINFO_MAXSIZE] = "";
		if (proc_pidpath(pid, path, sizeof(path)) <= 0) path[0] = 0;
		for (CFIndex j = 0; j < CFArrayGetCount(values[i]); j++) {
			CFDictionaryRef a = CFArrayGetValueAtIndex(values[i], j);
			if (CFGetTypeID(a) != CFDictionaryGetTypeID()) continue;
			lo_assertion *item = &items[n++];
			item->pid = pid;
			if (!lo_copy_int(a, kIOPMAssertionLevelKey, &item->level)) item->level = -1;
			item->type = lo_copy_string(a, kIOPMAssertionTypeKey);
			item->name = lo_copy_string(a, kIOPMAssertionNameKey);
			item->details = lo_copy_string(a, kIOPMAssertionDetailsKey);
			item->process = lo_copy_string(a, CFSTR("Process Name"));
			CFTypeRef when = CFDictionaryGetValue(a, CFSTR("AssertStartWhen"));
			if (when && CFGetTypeID(when) == CFDateGetTypeID()) {
				item->start = CFDateGetAbsoluteTime(when);
				item->start_valid = 1;
			}
			memcpy(item->path, path, sizeof(path));
		}
	}
	free(keys); free(values);
	CFRelease(byPID);
	*out = items; *count = n;
	return kIOReturnSuccess;
}
*/
import "C"

import (
	"fmt"
	"unsafe"

	"rockshinops/internal/localization"
)

const thermalSource = "notify com.apple.system.thermalpressurelevel"

func readThermal() Thermal {
	t := Thermal{Source: thermalSource}
	var state C.uint64_t
	if status := C.lo_thermal_pressure(&state); status != 0 {
		t.Error = fmt.Sprintf(localization.T("热压力读取失败（notify %d）"), status)
		return t
	}
	level, ok := thermalPressure(uint64(state))
	if !ok {
		t.Error = fmt.Sprintf(localization.T("系统返回未知热压力级别 %d"), uint64(state))
		return t
	}
	t.Pressure = level
	return t
}

func readPowerAssertions() PowerAssertions {
	result := PowerAssertions{Items: []PowerAssertion{}, Source: "IOPMCopyAssertionsByProcess"}
	var items *C.lo_assertion
	var count C.int
	if status := C.lo_read_assertions(&items, &count); status != 0 {
		result.Error = fmt.Sprintf(localization.T("电源断言读取失败（IOReturn 0x%x）"), uint32(status))
		return result
	}
	defer C.free(unsafe.Pointer(items))
	native := unsafe.Slice(items, int(count))
	raw := make([]rawAssertion, 0, len(native))
	take := func(s *C.char) string {
		if s == nil {
			return ""
		}
		defer C.free(unsafe.Pointer(s))
		return C.GoString(s)
	}
	for i := range native {
		a := &native[i]
		raw = append(raw, rawAssertion{pid: int(a.pid), level: int(a.level), assertType: take(a._type), name: take(a.name), details: take(a.details),
			process: take(a.process), path: C.GoString(&a.path[0]), start: float64(a.start), startValid: a.start_valid != 0})
	}
	result.Items = powerAssertions(raw)
	return result
}
