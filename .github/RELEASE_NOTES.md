LocalOps is a local desktop system monitor with persistent menu bar metrics, English/Chinese UI, process and app inspection, and read-only disk space analysis.

### New in v0.2.1

Signed updates from GitHub Releases, checked after launch and daily. Use Settings, the app menu or the tray menu to check manually. Review release notes, install with progress, skip a version or choose a reminder, then relaunch when ready. Installation requires confirmation unless you opt in to automatic installation.

**v0.2.0 users: install v0.2.1 manually once to enable future updates.** Updates need a writable installation; upgrade Linux system packages through the package manager. GitHub must be reachable.

**中文：**新增 GitHub 签名更新，启动后和持续运行期间每天检查；设置、应用菜单、托盘菜单均可手动检查。默认确认后安装，支持进度、跳过版本及稍后提醒，重启时机由你决定。v0.2.0 用户需先手动安装一次本版本；Linux 系统软件包通过包管理器升级。

### Downloads

- **macOS Apple Silicon:** the `arm64.dmg` file.
- **macOS Intel:** the `amd64.dmg` file.
- **Windows x86_64:** the `Setup … amd64.exe` installer.
- **Linux x86_64:** the `.deb` package or `.tar.gz` archive (includes a local install script).
- `SHA256SUMS.txt` contains checksums for all packages. `update-*.json` and the macOS/Windows `.tar.gz` files are consumed by the updater; choose the installers above for manual installation.

### Platform status

System collectors are implemented for macOS, Windows and Linux. All four native CI runners sample CPU, memory, disk, network, processes and a known TCP socket. Interactive desktop acceptance has been performed on macOS Apple Silicon; interactive Windows/Linux desktop verification remains device-dependent.

GPU support: macOS IOKit, Linux AMD sysfs or NVIDIA SMI, Windows NVIDIA SMI. Drivers must expose the relevant counters. Missing GPU, battery or power telemetry is shown as unavailable. Linux displays the first system battery; Windows uses system battery status.

macOS packages are ad-hoc signed and not notarized. Windows packages are unsigned. Operating systems may require explicit approval to open downloaded apps. Update archives are signed independently of OS code signing.

Language defaults to the system preference. Choose **System default / 中文 / English** in Settings and save to persist an override. File analysis is read-only; LocalOps does not delete files or terminate processes.

See the repository README for installation, dependencies and measurement details.
