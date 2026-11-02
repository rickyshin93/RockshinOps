#ifndef ROCKSHINOPS_SENSORS_H
#define ROCKSHINOPS_SENSORS_H
#include <stdint.h>

// Read-only AppleSMC access. The connection stays open for the process lifetime.
int lo_smc_open(void);
int lo_smc_key_at(uint32_t index, uint32_t *key);
int lo_smc_key_info(uint32_t key, uint32_t *type, uint32_t *size);
int lo_smc_read(uint32_t key, uint32_t size, uint8_t *bytes);

typedef struct { char name[64]; double celsius; int valid; } lo_hid_temperature;
// Returns -1 when the HID event system or its temperature services are unavailable.
int lo_hid_temperatures(lo_hid_temperature **out, int *count);
#endif
