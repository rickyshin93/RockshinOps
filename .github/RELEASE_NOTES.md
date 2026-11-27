RockshinOps is a local desktop system monitor with persistent menu bar metrics, English/Chinese UI, process and app inspection, disk space analysis, selected cleanup and macOS app removal.

### Changes in v0.3.5

- **Quick Cleanup** finds older caches, logs and temporary files. Search and review your selection before moving items to the system Trash. Downloads and Xcode build caches require manual selection.
- **Empty Trash** shows a paged review with per-item usage and a space estimate that accounts for hard links. Permanent deletion verifies file contents after staging, preserves changed items and Linux restore information, and reports incomplete operations with recovery paths when available.
- Preference changes keep tray updates in order. Calculator input-source switching runs when entering or leaving the page.

### 中文

- **快速清理**可筛选较旧的缓存、日志和临时文件，复核所选项目后移入系统废纸篓。下载文件和 Xcode 构建缓存需手动选中。
- **清空废纸篓**前可分页复核每项占用和预计释放空间，合计按共享硬链接去重。永久删除在暂存后核对文件内容，保留发生变化的条目和 Linux 恢复信息，并报告部分失败和可用的恢复路径。
- 设置变更期间保持菜单栏更新顺序。计算器只在进入或离开页面时切换输入源。

### Upgrade notice / 升级说明

Old LocalOps installations require a manual installation of RockshinOps because their updater relaunches the old executable path. Linux system packages should be updated through the package manager.

旧 LocalOps 用户需手动安装 RockshinOps；Linux 系统软件包请通过包管理器更新。

macOS builds remain ad-hoc signed and not notarized; Accessibility or Screen Recording permissions may need to be re-granted after replacement.

### Downloads

- **macOS Apple Silicon:** the `arm64.dmg` file.
- **macOS Intel:** the `amd64.dmg` file.
- **Windows x86_64:** the `rockshinops-setup-0.3.5-amd64.exe` installer.
- **Linux x86_64:** the `.deb` package or `.tar.gz` archive (extract and run `./rockshinops`; no install script is included).
- `SHA256SUMS.txt` contains checksums for all packages. `update-*.json` and the macOS/Windows `.tar.gz` files are consumed by the updater; choose the installers above for manual installation.

### Platform status

System collectors are implemented for macOS, Windows and Linux. All four native CI runners sample CPU, memory, disk, network, processes and a known TCP socket. Interactive desktop acceptance has been performed on macOS Apple Silicon; interactive Windows/Linux desktop verification remains device-dependent.

This release has not completed interactive Windows/Linux acceptance, multi-display/fullscreen and sleep/wake coverage, or every macOS tray/third-party menu interaction.

GPU support: macOS IOKit, Linux AMD sysfs or NVIDIA SMI, Windows NVIDIA SMI. Drivers must expose the relevant counters. Missing GPU, battery or power telemetry is shown as unavailable. Linux displays the first system battery; Windows uses system battery status.

macOS packages are ad-hoc signed and not notarized. Windows packages are unsigned. Operating systems may require explicit approval to open downloaded apps. Update archives are signed independently of OS code signing.

Language defaults to the system preference. Choose **System default / 中文 / English** in Settings; changes are saved automatically. Review is required before moving selected items to the system trash. Emptying the trash requires explicit confirmation; process termination requires a separate confirmation naming the process and PID.

See the repository README for installation, dependencies and measurement details.
