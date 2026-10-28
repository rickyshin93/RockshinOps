LocalOps is a local desktop system monitor with persistent menu bar metrics, English/Chinese UI, process and app inspection, disk space analysis, selected cleanup and macOS app removal.

### New in v0.2.2

Explore disk folders and review selected items before moving them to the system trash. On macOS, review installed apps and possible associated files before moving selected items to the trash. Search results can be incomplete; name matches may be shared with other apps. Permission limitations are shown with an entry to macOS Full Disk Access settings.

**中文：**新增磁盘浏览和所选项目移入废纸篓；macOS 可核对应用及可能的关联文件后选择移入废纸篓。搜索可能不完整，名称匹配可能属于共享数据；受权限限制时可查看说明并打开完全磁盘访问设置。

LocalOps 0.2.1 can update to this release through the signed updater. Users of 0.2.0 should install this version manually once.

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

Language defaults to the system preference. Choose **System default / 中文 / English** in Settings and save to persist an override. Review is required before moving selected items to the system trash. Emptying the trash requires explicit confirmation; LocalOps does not terminate processes.

See the repository README for installation, dependencies and measurement details.
