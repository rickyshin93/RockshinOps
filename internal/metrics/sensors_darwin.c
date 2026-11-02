#include "sensors_darwin.h"
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/hidsystem/IOHIDEventSystemClient.h>
#include <IOKit/hidsystem/IOHIDServiceClient.h>
#include <stdlib.h>
#include <string.h>

// AppleSMC user client layout shared by Intel and Apple Silicon Macs.
typedef struct {
    uint32_t key;
    struct { uint8_t major, minor, build, reserved; uint16_t release; } version;
    struct { uint16_t version, length; uint32_t cpu, gpu, memory; } limits;
    struct { uint32_t size, type; uint8_t attributes; } info;
    uint8_t result, status, command;
    uint32_t index;
    uint8_t bytes[32];
} smc_param;
_Static_assert(sizeof(smc_param) == 80, "AppleSMC parameter layout");

enum { SMC_CALL = 2, SMC_READ_BYTES = 5, SMC_READ_INDEX = 8, SMC_READ_INFO = 9 };
static io_connect_t smc;

int lo_smc_open(void) {
    if (smc) return 0;
    io_service_t service = IOServiceGetMatchingService(kIOMainPortDefault, IOServiceMatching("AppleSMC"));
    if (!service) return -1;
    kern_return_t result = IOServiceOpen(service, mach_task_self(), 0, &smc);
    IOObjectRelease(service);
    if (result != KERN_SUCCESS) smc = 0;
    return result;
}

static int smc_call(smc_param *input, smc_param *output) {
    size_t size = sizeof(*output);
    memset(output, 0, size);
    kern_return_t result = IOConnectCallStructMethod(smc, SMC_CALL, input, sizeof(*input), output, &size);
    if (result != KERN_SUCCESS) return result;
    return output->result;  // Non-zero SMC result codes mean the key is missing or unreadable.
}

int lo_smc_key_at(uint32_t index, uint32_t *key) {
    smc_param input = {0}, output;
    input.command = SMC_READ_INDEX;
    input.index = index;
    int result = smc_call(&input, &output);
    *key = output.key;
    return result;
}

int lo_smc_key_info(uint32_t key, uint32_t *type, uint32_t *size) {
    smc_param input = {0}, output;
    input.key = key;
    input.command = SMC_READ_INFO;
    int result = smc_call(&input, &output);
    *type = output.info.type;
    *size = output.info.size;
    return result;
}

int lo_smc_read(uint32_t key, uint32_t size, uint8_t *bytes) {
    if (size > 32) return -1;
    smc_param input = {0}, output;
    input.key = key;
    input.info.size = size;
    input.command = SMC_READ_BYTES;
    int result = smc_call(&input, &output);
    memcpy(bytes, output.bytes, size);
    return result;
}

// Private but exported IOKit HID event-system calls used for Apple vendor temperature services.
typedef struct __IOHIDEvent *IOHIDEventRef;
IOHIDEventSystemClientRef IOHIDEventSystemClientCreate(CFAllocatorRef allocator);
int IOHIDEventSystemClientSetMatching(IOHIDEventSystemClientRef client, CFDictionaryRef matching);
IOHIDEventRef IOHIDServiceClientCopyEvent(IOHIDServiceClientRef service, int64_t type, int32_t options, int64_t timestamp);
double IOHIDEventGetFloatValue(IOHIDEventRef event, int32_t field);

enum { HID_VENDOR_PAGE = 0xff00, HID_TEMPERATURE_USAGE = 5, HID_TEMPERATURE_EVENT = 15 };
static IOHIDEventSystemClientRef hid;
static CFArrayRef services;
static CFIndex service_count;

static CFNumberRef number(int value) { return CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &value); }

static int hid_open(void) {
    if (services) return 0;
    if (!hid) hid = IOHIDEventSystemClientCreate(kCFAllocatorDefault);
    if (!hid) return -1;
    CFNumberRef page = number(HID_VENDOR_PAGE), usage = number(HID_TEMPERATURE_USAGE);
    const void *keys[] = {CFSTR("PrimaryUsagePage"), CFSTR("PrimaryUsage")}, *values[] = {page, usage};
    CFDictionaryRef matching = CFDictionaryCreate(kCFAllocatorDefault, keys, values, 2, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    IOHIDEventSystemClientSetMatching(hid, matching);
    CFRelease(matching);
    CFRelease(page);
    CFRelease(usage);
    services = IOHIDEventSystemClientCopyServices(hid);
    if (!services) return -1;
    service_count = CFArrayGetCount(services);
    return 0;
}

int lo_hid_temperatures(lo_hid_temperature **out, int *count) {
    *out = NULL;
    *count = 0;
    if (hid_open() != 0 || service_count == 0) return -1;
    lo_hid_temperature *items = calloc((size_t)service_count, sizeof(*items));
    if (!items) return -1;
    for (CFIndex i = 0; i < service_count; i++) {
        IOHIDServiceClientRef service = (IOHIDServiceClientRef)CFArrayGetValueAtIndex(services, i);
        CFTypeRef product = IOHIDServiceClientCopyProperty(service, CFSTR("Product"));
        if (product && CFGetTypeID(product) == CFStringGetTypeID()) {
            CFStringGetCString(product, items[i].name, sizeof(items[i].name), kCFStringEncodingUTF8);
        }
        if (product) CFRelease(product);
        IOHIDEventRef event = IOHIDServiceClientCopyEvent(service, HID_TEMPERATURE_EVENT, 0, 0);
        if (event) {
            items[i].celsius = IOHIDEventGetFloatValue(event, HID_TEMPERATURE_EVENT << 16);
            items[i].valid = 1;
            CFRelease(event);
        }
    }
    *out = items;
    *count = (int)service_count;
    return 0;
}
