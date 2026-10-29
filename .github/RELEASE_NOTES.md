RockshinOps is a local desktop system monitor with persistent menu bar metrics, English/Chinese UI, process and app inspection, disk space analysis, selected cleanup and macOS app removal.

### New in v0.2.4

- RockshinOps branding, refreshed navigation, settings and hardware pages, and smoother macOS window resizing.
- Rebuilt Processes and Connections tables with search, sorting, pause/resume and process detail charts from real samples.
- Confirmed process Quit (SIGTERM on macOS/Linux) and Force Quit, with process identity checks and visible permission errors. Windows supports Force Quit only.
- macOS menu-bar management with visible/hidden sections, icon previews, reordering, temporary reveal and appearance settings. Moving other apps' icons requires Accessibility access; previews require Screen Recording access.
- Automatically saved settings and an optional “Hide Dock icon when closing” preference.

**中文：**更新 RockshinOps 品牌与整体界面，重做进程、连接列表和进程详情；新增确认后退出/强制退出进程、macOS 菜单栏图标分区与整理、自动保存设置及关闭窗口时隐藏 Dock 图标选项。

Installer filenames, executable paths and the preferences directory retain the **LocalOps** name so older versions can continue to update and relaunch. The app identifier and update signing key are unchanged. Versions 0.2.1–0.2.3 use the signed updater; 0.2.0 requires a manual install first. Linux system packages should be updated through the package manager.

### Downloads

- **macOS Apple Silicon:** the `arm64.dmg` file.
- **macOS Intel:** the `amd64.dmg` file.
- **Windows x86_64:** the `Setup … amd64.exe` installer.
- **Linux x86_64:** the `.deb` package or `.tar.gz` archive (includes a local install script).
- `SHA256SUMS.txt` contains checksums for all packages. `update-*.json` and the macOS/Windows `.tar.gz` files are consumed by the updater; choose the installers above for manual installation.

### Platform status

System collectors are implemented for macOS, Windows and Linux. All four native CI runners sample CPU, memory, disk, network, processes and a known TCP socket. Interactive desktop acceptance has been performed on macOS Apple Silicon; interactive Windows/Linux desktop verification remains device-dependent.

This release has not completed interactive Windows/Linux acceptance, multi-display/fullscreen and sleep/wake coverage, or every macOS tray/third-party menu interaction.

GPU support: macOS IOKit, Linux AMD sysfs or NVIDIA SMI, Windows NVIDIA SMI. Drivers must expose the relevant counters. Missing GPU, battery or power telemetry is shown as unavailable. Linux displays the first system battery; Windows uses system battery status.

macOS packages are ad-hoc signed and not notarized. Windows packages are unsigned. Operating systems may require explicit approval to open downloaded apps. Update archives are signed independently of OS code signing.

Language defaults to the system preference. Choose **System default / 中文 / English** in Settings; changes are saved automatically. Review is required before moving selected items to the system trash. Emptying the trash requires explicit confirmation; process termination requires a separate confirmation naming the process and PID.

See the repository README for installation, dependencies and measurement details.
