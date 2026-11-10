RockshinOps is a local desktop system monitor with persistent menu bar metrics, English/Chinese UI, process and app inspection, disk space analysis, selected cleanup and macOS app removal.

### New in v0.2.14

- Calculator adds editable calculation history, linked answers, and grouping and precision settings shared with Launcher.
- Evaluate expressions in Launcher, copy the answer or open it in Calculator.
- Developer port closing rechecks listener ownership and process identity before signalling.

**中文：**新增计算器，支持编辑历史、引用结果，以及与启动器共享的数字分组和精度设置。启动器支持计算表达式、复制结果或打开计算器。关闭开发者端口前重新验证监听和进程身份。

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
