RockshinOps is a local desktop system monitor with persistent menu bar metrics, English/Chinese UI, process and app inspection, disk space analysis, selected cleanup and macOS app removal.

### New in v0.2.12

- A dedicated macOS Sensors page shows available CPU, GPU, SSD and battery temperature trends, color-coded readings and fan speeds. Energy keeps battery, power and sleep information.
- A new macOS Devices page shows Wi-Fi information and requested scans, paired Bluetooth devices with connect/disconnect and available battery readings, plus audio output selection and volume.
- Keep new volume edits intact when an earlier adjustment is still waiting for the device response.
- English navigation is shorter: Space, MenuBar and About. Launcher now appears before About.

**中文：**新增独立传感器页，展示可用的 CPU、GPU、SSD 和电池温度趋势、分段颜色及风扇转速，能耗页保留电池、功耗和睡眠信息。新增设备连接页，支持 Wi-Fi 信息和按需扫描、已配对蓝牙设备连接及可用电量、声音输出和音量控制。英文导航缩短为 Space、MenuBar、About，启动器移到关于本机之前。

Sensors and Devices are macOS-only. Temperature channels depend on the hardware and do not identify individual physical cores. Wi-Fi names/scans and Bluetooth access depend on macOS permissions. Real permission recovery, Bluetooth/audio control writes, other Mac models and installed-app upgrades remain outside this release's local interactive acceptance.

### Upgrade notice / 升级说明

Old LocalOps installations require a manual installation of RockshinOps because their updater relaunches the old executable path. Linux system packages should be updated through the package manager.

旧 LocalOps 用户需手动安装 RockshinOps；Linux 系统软件包请通过包管理器更新。

macOS builds remain ad-hoc signed and not notarized; Accessibility or Screen Recording permissions may need to be re-granted after replacement.

### Downloads

- **macOS Apple Silicon:** the `arm64.dmg` file.
- **macOS Intel:** the `amd64.dmg` file.
- **Windows x86_64:** the `Setup … amd64.exe` installer.
- **Linux x86_64:** the `.deb` package or `.tar.gz` archive (extract and run `./rockshinops`; no install script is included).
- `SHA256SUMS.txt` contains checksums for all packages. `update-*.json` and the macOS/Windows `.tar.gz` files are consumed by the updater; choose the installers above for manual installation.

### Platform status

System collectors are implemented for macOS, Windows and Linux. All four native CI runners sample CPU, memory, disk, network, processes and a known TCP socket. Interactive desktop acceptance has been performed on macOS Apple Silicon; interactive Windows/Linux desktop verification remains device-dependent.

This release has not completed interactive Windows/Linux acceptance, multi-display/fullscreen and sleep/wake coverage, or every macOS tray/third-party menu interaction.

GPU support: macOS IOKit, Linux AMD sysfs or NVIDIA SMI, Windows NVIDIA SMI. Drivers must expose the relevant counters. Missing GPU, battery or power telemetry is shown as unavailable. Linux displays the first system battery; Windows uses system battery status.

macOS packages are ad-hoc signed and not notarized. Windows packages are unsigned. Operating systems may require explicit approval to open downloaded apps. Update archives are signed independently of OS code signing.

Language defaults to the system preference. Choose **System default / 中文 / English** in Settings; changes are saved automatically. Review is required before moving selected items to the system trash. Emptying the trash requires explicit confirmation; process termination requires a separate confirmation naming the process and PID.

See the repository README for installation, dependencies and measurement details.
