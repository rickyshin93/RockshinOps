#ifndef LOCALOPS_NATIVE_H
#define LOCALOPS_NATIVE_H
#include <stdint.h>
#include <stddef.h>

typedef struct { uint32_t user, system, idle, nice; } lo_cpu;
typedef struct {
    uint64_t total, active, inactive, speculative, wired, compressed, purgeable, external;
    uint64_t swap_used, swap_total;
    int pressure, error, swap_error;
} lo_memory;
typedef struct { char id[128]; uint64_t read, write; int up; } lo_io;
typedef struct { char name[128]; double usage; uint64_t memory; int available; } lo_gpu;
typedef struct {
    int present, charging, external, minutes, health_valid, cycles_valid, watts_valid, system_watts_valid, temperature_valid, error;
    double percent, health, watts, system_watts, temperature;
    int cycles;
} lo_battery;
typedef struct {
    int pid, threads; uint32_t uid;
    uint64_t start, cpu_time, memory, read, write;
    char name[256], path[4096];
} lo_process;

int lo_read_cpu(lo_cpu **out, int *count);
lo_memory lo_read_memory(void);
lo_gpu lo_read_gpu(void);
int lo_read_disk(lo_io **out, int *count);
int lo_read_network(lo_io **out, int *count);
lo_battery lo_read_battery(void);
int lo_read_processes(lo_process **out, int *count, int *total);
double lo_uptime(void);
int lo_sysctl_string(const char *name, char *out, size_t size);
uint64_t lo_sysctl_number(const char *name);
void lo_free(void *ptr);
#endif
