#include "native_darwin.h"
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/ps/IOPowerSources.h>
#include <IOKit/ps/IOPSKeys.h>
#include <mach/mach.h>
#include <mach/mach_time.h>
#include <mach/processor_info.h>
#include <libproc.h>
#include <sys/proc_info.h>
#include <sys/resource.h>
#include <sys/sysctl.h>
#include <sys/time.h>
#include <net/if.h>
#include <net/route.h>
#include <errno.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

void lo_free(void *ptr) { free(ptr); }

int lo_sysctl_string(const char *name, char *out, size_t size) {
    return sysctlbyname(name, out, &size, NULL, 0);
}
uint64_t lo_sysctl_number(const char *name) {
    uint64_t value = 0;
    size_t size = sizeof(value);
    if (sysctlbyname(name, &value, &size, NULL, 0) != 0) return 0;
    return value;
}

int lo_read_cpu(lo_cpu **out, int *count) {
    natural_t cores = 0;
    processor_info_array_t info;
    mach_msg_type_number_t length = 0;
    mach_port_t host = mach_host_self();
    kern_return_t result = host_processor_info(host, PROCESSOR_CPU_LOAD_INFO, &cores, &info, &length);
    mach_port_deallocate(mach_task_self(), host);
    if (result != KERN_SUCCESS) return result;
    lo_cpu *items = calloc(cores, sizeof(lo_cpu));
    if (!items) { vm_deallocate(mach_task_self(), (vm_address_t)info, length * sizeof(integer_t)); return ENOMEM; }
    for (unsigned int i = 0; i < cores; i++) {
        items[i] = (lo_cpu){info[i * CPU_STATE_MAX + CPU_STATE_USER], info[i * CPU_STATE_MAX + CPU_STATE_SYSTEM],
                           info[i * CPU_STATE_MAX + CPU_STATE_IDLE], info[i * CPU_STATE_MAX + CPU_STATE_NICE]};
    }
    vm_deallocate(mach_task_self(), (vm_address_t)info, length * sizeof(integer_t));
    *out = items; *count = cores;
    return 0;
}

lo_memory lo_read_memory(void) {
    lo_memory m = {0};
    vm_statistics64_data_t v;
    mach_msg_type_number_t count = HOST_VM_INFO64_COUNT;
    mach_port_t host = mach_host_self();
    m.error = host_statistics64(host, HOST_VM_INFO64, (host_info64_t)&v, &count);
    mach_port_deallocate(mach_task_self(), host);
    if (m.error) return m;
    m.total = lo_sysctl_number("hw.memsize");
    if (!m.total) { m.error = EINVAL; return m; }
    uint64_t page = vm_kernel_page_size;
    m.active = v.active_count * page; m.inactive = v.inactive_count * page;
    m.speculative = v.speculative_count * page; m.wired = v.wire_count * page;
    m.compressed = v.compressor_page_count * page; m.purgeable = v.purgeable_count * page;
    m.external = v.external_page_count * page;
    struct xsw_usage swap;
    size_t size = sizeof(swap);
    if (sysctlbyname("vm.swapusage", &swap, &size, NULL, 0) == 0) {
        m.swap_used = swap.xsu_used; m.swap_total = swap.xsu_total;
    } else { m.swap_error = errno; }
    size = sizeof(m.pressure);
    if (sysctlbyname("kern.memorystatus_vm_pressure_level", &m.pressure, &size, NULL, 0)) m.pressure = 0;
    return m;
}

static int number(CFDictionaryRef dictionary, CFStringRef key, int64_t *value) {
    if (!dictionary || CFGetTypeID(dictionary) != CFDictionaryGetTypeID()) return 0;
    CFTypeRef field = CFDictionaryGetValue(dictionary, key);
    return field && CFGetTypeID(field) == CFNumberGetTypeID() && CFNumberGetValue(field, kCFNumberSInt64Type, value);
}
static int boolean(CFDictionaryRef dictionary, CFStringRef key) {
    return dictionary && CFDictionaryGetValue(dictionary, key) == kCFBooleanTrue;
}
static void string(CFDictionaryRef dictionary, CFStringRef key, char *out, size_t size) {
    CFTypeRef field = dictionary ? CFDictionaryGetValue(dictionary, key) : NULL;
    if (field && CFGetTypeID(field) == CFStringGetTypeID()) CFStringGetCString(field, out, size, kCFStringEncodingUTF8);
}

lo_gpu lo_read_gpu(void) {
    lo_gpu gpu = {0};
    io_iterator_t iterator;
    if (IOServiceGetMatchingServices(kIOMainPortDefault, IOServiceMatching("IOAccelerator"), &iterator)) return gpu;
    io_object_t service;
    while ((service = IOIteratorNext(iterator))) {
        CFMutableDictionaryRef props = NULL;
        if (IORegistryEntryCreateCFProperties(service, &props, kCFAllocatorDefault, 0) == KERN_SUCCESS) {
            CFDictionaryRef stats = CFDictionaryGetValue(props, CFSTR("PerformanceStatistics"));
            int64_t utilization, memory;
            if (number(stats, CFSTR("Device Utilization %"), &utilization) && utilization >= 0 && utilization <= 100 &&
                number(stats, CFSTR("In use system memory"), &memory) && memory >= 0) {
                gpu.usage = utilization; gpu.memory = memory; gpu.available = 1;
                string(props, CFSTR("model"), gpu.name, sizeof(gpu.name));
            }
            CFRelease(props);
        }
        IOObjectRelease(service);
        if (gpu.available) break;
    }
    IOObjectRelease(iterator);
    return gpu;
}

int lo_read_disk(lo_io **out, int *count) {
    io_iterator_t iterator;
    int result = IOServiceGetMatchingServices(kIOMainPortDefault, IOServiceMatching("IOBlockStorageDriver"), &iterator);
    if (result) return result;
    lo_io *items = NULL;
    int length = 0;
    io_object_t service;
    while ((service = IOIteratorNext(iterator))) {
        CFTypeRef stats = IORegistryEntryCreateCFProperty(service, CFSTR("Statistics"), kCFAllocatorDefault, 0);
        int64_t read, write;
        if (stats && number(stats, CFSTR("Bytes (Read)"), &read) && number(stats, CFSTR("Bytes (Write)"), &write)) {
            lo_io *grown = realloc(items, (length + 1) * sizeof(lo_io));
            if (!grown) { CFRelease(stats); IOObjectRelease(service); IOObjectRelease(iterator); free(items); return ENOMEM; }
            items = grown;
            uint64_t id = 0;
            IORegistryEntryGetRegistryEntryID(service, &id);
            items[length] = (lo_io){.read = read, .write = write, .up = 1};
            snprintf(items[length].id, sizeof(items[length].id), "%llu", id);
            length++;
        }
        if (stats) CFRelease(stats);
        IOObjectRelease(service);
    }
    IOObjectRelease(iterator);
    *out = items; *count = length;
    return length ? 0 : ENODEV;
}

int lo_read_network(lo_io **out, int *count) {
    int mib[] = {CTL_NET, PF_ROUTE, 0, 0, NET_RT_IFLIST2, 0};
    size_t length = 0;
    if (sysctl(mib, 6, NULL, &length, NULL, 0)) return errno;
    char *buffer = malloc(length);
    if (!buffer) return ENOMEM;
    if (sysctl(mib, 6, buffer, &length, NULL, 0)) { int err = errno; free(buffer); return err; }
    lo_io *items = calloc(length / sizeof(struct if_msghdr2) + 1, sizeof(lo_io));
    if (!items) { free(buffer); return ENOMEM; }
    int n = 0;
    for (char *cursor = buffer; cursor < buffer + length;) {
        struct if_msghdr *header = (struct if_msghdr *)cursor;
        if (!header->ifm_msglen || cursor + header->ifm_msglen > buffer + length) { free(items); free(buffer); return EINVAL; }
        if (header->ifm_type == RTM_IFINFO2) {
            struct if_msghdr2 *info = (struct if_msghdr2 *)cursor;
            if (!(info->ifm_flags & IFF_LOOPBACK)) {
                if_indextoname(info->ifm_index, items[n].id);
                items[n].read = info->ifm_data.ifi_ibytes;
                items[n].write = info->ifm_data.ifi_obytes;
                items[n].up = !!(info->ifm_flags & IFF_UP);
                n++;
            }
        }
        cursor += header->ifm_msglen;
    }
    free(buffer); *out = items; *count = n;
    return 0;
}

lo_battery lo_read_battery(void) {
    lo_battery b = {.minutes = -1};
    CFTypeRef sources = IOPSCopyPowerSourcesInfo();
    if (!sources) { b.error = EIO; return b; }
    CFArrayRef list = IOPSCopyPowerSourcesList(sources);
    if (!list) { CFRelease(sources); b.error = EIO; return b; }
    for (CFIndex i = 0; i < CFArrayGetCount(list); i++) {
        CFDictionaryRef info = IOPSGetPowerSourceDescription(sources, CFArrayGetValueAtIndex(list, i));
        CFTypeRef type = info ? CFDictionaryGetValue(info, CFSTR(kIOPSTypeKey)) : NULL;
        if (!type || !CFEqual(type, CFSTR(kIOPSInternalBatteryType)) || !boolean(info, CFSTR(kIOPSIsPresentKey))) continue;
        int64_t current = 0, maximum = 0, minutes = -1;
        b.present = 1;
        if (!number(info, CFSTR(kIOPSCurrentCapacityKey), &current) || !number(info, CFSTR(kIOPSMaxCapacityKey), &maximum) || maximum <= 0) {
            b.error = EINVAL; break;
        }
        b.percent = (double)current / maximum * 100;
        b.charging = boolean(info, CFSTR(kIOPSIsChargingKey));
        CFTypeRef power = CFDictionaryGetValue(info, CFSTR(kIOPSPowerSourceStateKey));
        b.external = power && CFEqual(power, CFSTR(kIOPSACPowerValue));
        number(info, b.charging ? CFSTR(kIOPSTimeToFullChargeKey) : CFSTR(kIOPSTimeToEmptyKey), &minutes);
        b.minutes = minutes;
        break;
    }
    CFRelease(list); CFRelease(sources);
    if (!b.present || b.error) return b;
    io_service_t service = IOServiceGetMatchingService(kIOMainPortDefault, IOServiceMatching("AppleSmartBattery"));
    if (!service) return b;
    CFMutableDictionaryRef props = NULL;
    if (IORegistryEntryCreateCFProperties(service, &props, kCFAllocatorDefault, 0) == KERN_SUCCESS) {
        int64_t maximum, design, value, voltage, current;
        if (number(props, CFSTR("AppleRawMaxCapacity"), &maximum) && number(props, CFSTR("DesignCapacity"), &design) && design > 0) {
            b.health_valid = 1; b.health = (double)maximum / design * 100;
        }
        if (number(props, CFSTR("CycleCount"), &value)) { b.cycles_valid = 1; b.cycles = value; }
        if (number(props, CFSTR("Voltage"), &voltage) && number(props, CFSTR("InstantAmperage"), &current)) {
            b.watts_valid = 1; b.watts = (double)voltage * (int32_t)current / 1000000;
        }
        if (number(props, CFSTR("Temperature"), &value)) { b.temperature_valid = 1; b.temperature = (double)value / 100; }
        CFDictionaryRef telemetry = CFDictionaryGetValue(props, CFSTR("PowerTelemetryData"));
        if (number(telemetry, CFSTR("SystemLoad"), &value) && value >= 0) { b.system_watts_valid = 1; b.system_watts = (double)value / 1000; }
        CFRelease(props);
    }
    IOObjectRelease(service);
    return b;
}

int lo_read_processes(lo_process **out, int *count, int *total) {
    mach_timebase_info_data_t timebase;
    kern_return_t timing = mach_timebase_info(&timebase);
    if (timing != KERN_SUCCESS) return timing;
    int estimate = proc_listallpids(NULL, 0);
    if (estimate <= 0) return errno ? errno : EIO;
    int capacity = estimate * 2;
    int *pids = calloc(capacity, sizeof(int));
    lo_process *items = calloc(capacity, sizeof(lo_process));
    if (!pids || !items) { free(pids); free(items); return ENOMEM; }
    int length = proc_listallpids(pids, capacity * sizeof(int));
    if (length <= 0) { int err = errno ? errno : EIO; free(pids); free(items); return err; }
    int n = 0;
    *total = 0;
    for (int i = 0; i < length; i++) {
        if (pids[i] <= 0) continue;
        (*total)++;
        struct rusage_info_v2 usage;
        struct proc_taskallinfo info;
        if (proc_pid_rusage(pids[i], RUSAGE_INFO_V2, (rusage_info_t *)&usage) != 0 ||
            proc_pidinfo(pids[i], PROC_PIDTASKALLINFO, 0, &info, sizeof(info)) != sizeof(info)) continue;
        lo_process *p = &items[n++];
        p->pid = pids[i]; p->uid = info.pbsd.pbi_uid; p->threads = info.ptinfo.pti_threadnum;
        p->start = usage.ri_proc_start_abstime;
        // proc_pid_rusage CPU counters are Mach absolute ticks on Apple Silicon.
        p->cpu_time = (uint64_t)(((__uint128_t)usage.ri_user_time + usage.ri_system_time) * timebase.numer / timebase.denom);
        p->memory = usage.ri_phys_footprint; p->read = usage.ri_diskio_bytesread; p->write = usage.ri_diskio_byteswritten;
        proc_name(pids[i], p->name, sizeof(p->name));
        proc_pidpath(pids[i], p->path, sizeof(p->path));
        if (!p->name[0]) snprintf(p->name, sizeof(p->name), "%s", info.pbsd.pbi_name[0] ? info.pbsd.pbi_name : info.pbsd.pbi_comm);
    }
    free(pids); *out = items; *count = n;
    return 0;
}

double lo_uptime(void) {
    struct timeval boot, now;
    size_t length = sizeof(boot);
    if (sysctlbyname("kern.boottime", &boot, &length, NULL, 0) || gettimeofday(&now, NULL)) return -1;
    return (double)(now.tv_sec - boot.tv_sec) + (double)(now.tv_usec - boot.tv_usec) / 1000000;
}
