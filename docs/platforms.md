# Platform collection contract

RockshinOps reads operating-system counters without privilege escalation. macOS uses its existing Mach, IOKit and libproc collectors. Windows/Linux use gopsutil v4 for CPU, memory, processes and sockets. Windows disk and network counters use direct Win32 calls. In particular, the gopsutil Windows disk module is not imported: its init attempts to enable counters through a registry write, and its capacity return is total free space rather than per-user available space.

Implementation scope:

- CPU, memory, swap, disk capacity/I/O, interfaces/traffic, uptime and machine details.
- Accessible processes, CPU/RSS/I/O and grouping by executable on Windows/Linux; macOS retains bundle grouping and phys_footprint.
- TCP/UDP sockets and ownership visible with current permissions.
- macOS per-socket `nettop` receive/send counters, two-sample rates and positive TCP RTT, joined to `lsof` by unique PID/protocol/endpoints. Process aggregates are never assigned to sockets. Missing/ambiguous matches remain unavailable; UDP and listening sockets have no TCP RTT. Windows/Linux per-socket traffic metrics are not implemented.
- Native battery status and supported GPU counters; missing hardware or unsupported driver interfaces stay unavailable.
- Platform-specific default scan directories, with the existing read-only analysis and resource limits.

Data contracts:

- Rates need two compatible samples. Counter reset, process-ID reuse and sleep gaps reset the baseline.
- CPU counters retain their source width: Mach 32-bit rollover is distinct from a 64-bit counter reset.
- Do not sum both a disk and its partitions, or both physical network traffic and a VPN/bridge copy. Windows sums fixed-volume I/O; Linux sums /sys/block devices with a hardware device link (excluding partition, device-mapper, RAID-layer and loop copies). Windows network totals require the MIB_IF_ROW2 hardware flag; Linux requires a sysfs device link. Virtual interfaces remain individually visible.
- Windows/Linux memory composition uses their own definitions; do not present zero-valued macOS wired/compressed fields as measurements.
- A denied optional process counter must remain unavailable without hiding its other accessible information.
- App icons are native on macOS; executable groups on other platforms use the generic app icon.

Acceptance: unit tests for counter transitions and parsers, real collector smoke tests on both macOS architectures plus Windows/Linux CI runners, macOS package/UI regression checks, and all four installable packages. No release tag is pushed until the implementation and required checks pass. GPU/battery hardware coverage and interactive Windows/Linux desktop verification are reported with their actual limits.

Dependency decision: [gopsutil v4.26.9](https://github.com/shirou/gopsutil/releases/tag/v4.26.9) was released in October 2026 and is actively maintained. It avoids duplicating Linux procfs parsing and Windows process/system calls; platform-specific behavior is still validated by RockshinOps tests.

Hardware adapters:

- Windows battery: GetSystemPowerStatus. Unknown charge is unavailable; absent batteries remain absent.
- Linux battery: first system battery in sorted sysfs name order, excluding device/peripheral batteries. Kernel micro-units are converted to watts/Wh; missing optional readings stay null.
- GPU: Linux first AMD sysfs device with counters, otherwise NVIDIA GPU index 0; Windows NVIDIA GPU index 0. NVIDIA SMI must already be supplied by the driver. Queries are bounded by the configured sampling interval; no driver or utility is installed by RockshinOps.
- CPU temperature, multiple-GPU aggregation, multi-battery aggregation on Linux and per-app watts are not implemented.

Primary references: [MIB_IF_ROW2](https://learn.microsoft.com/en-us/windows/win32/api/netioapi/ns-netioapi-mib_if_row2), [SYSTEM_POWER_STATUS](https://learn.microsoft.com/en-us/windows/win32/api/winbase/ns-winbase-system_power_status), [DISK_PERFORMANCE](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntdddisk/ns-ntdddisk-_disk_performance), [Linux power supply class](https://cdn.kernel.org/doc/html/latest/power/power_supply_class.html), [AMDGPU counters](https://kernel.org/doc/html/latest/gpu/amdgpu/thermal.html), [NVIDIA SMI](https://docs.nvidia.com/deploy/nvidia-smi/index.html).

Verification evidence: [four native CI jobs and the v0.2.0 release](https://github.com/rickyshin93/RockshinOps/actions/runs/40373814709) passed on 2026-10-27. The smoke test read real CPU/memory/disk/network counters, found its own process, and matched its own TCP listening socket on every runner. Hosted runners had no battery or usable GPU counters; those adapters have parser/unit coverage, not physical Windows/Linux GPU or battery acceptance. macOS Apple Silicon desktop checks covered language switching, persistence, native App icons and hidden-window history.
