RockshinOps is a local desktop system monitor with persistent menu bar metrics, English/Chinese UI, process and app inspection, disk space analysis, selected cleanup and macOS app removal.

### Fixes in v0.3.3

- Clipboard history preserves existing items if a write fails, reports when pinned items fill the budget, and keeps copy order after a system clock change. Search results and preview focus stay in sync with the current view.
- Launcher refreshes early app searches, keeps excluded files out, and handles overlapping folders, replaced paths and held shortcuts correctly.
- Calculator keeps continuation input while saving and through a retry. Canceling a calculation discards its queued input, and deletion preserves row references.
- Space analysis scans the volume containing the home directory, preserves Trash accounting, and keeps the latest folder or tab selection.
- Live metrics keep updating across system clock and language changes. Keep Awake uses elapsed time for timed sessions, and Linux keeps a readable battery result when another power supply disappears.

### 中文

- 剪贴板写入失败时保留旧记录，置顶占满容量时显示提示。系统时间回拨后仍按复制顺序排列，搜索结果和预览焦点随当前视图更新。
- 启动器修复了首次应用搜索、排除目录、重叠索引、路径替换以及长按快捷键的问题。
- 计算器在保存和失败重试期间保留后续输入。取消算式时清除对应的待续输入，删除操作保留行引用的指向。
- 空间分析按主目录所在卷扫描，保持废纸篓容量统计正确，并保留最新的目录或标签页选择。
- 切换语言或调整系统时间后，实时监控继续更新。保持唤醒按实际经过的时长计时。Linux 上其他供电设备消失时，仍会显示可用的电池读数。

### Upgrade notice / 升级说明

Old LocalOps installations require a manual installation of RockshinOps because their updater relaunches the old executable path. Linux system packages should be updated through the package manager.

旧 LocalOps 用户需手动安装 RockshinOps；Linux 系统软件包请通过包管理器更新。

macOS builds remain ad-hoc signed and not notarized; Accessibility or Screen Recording permissions may need to be re-granted after replacement.

### Downloads

- **macOS Apple Silicon:** the `arm64.dmg` file.
- **macOS Intel:** the `amd64.dmg` file.
- **Windows x86_64:** the `rockshinops-setup-0.3.3-amd64.exe` installer.
- **Linux x86_64:** the `.deb` package or `.tar.gz` archive (extract and run `./rockshinops`; no install script is included).
- `SHA256SUMS.txt` contains checksums for all packages. `update-*.json` and the macOS/Windows `.tar.gz` files are consumed by the updater; choose the installers above for manual installation.

### Platform status

System collectors are implemented for macOS, Windows and Linux. All four native CI runners sample CPU, memory, disk, network, processes and a known TCP socket. Interactive desktop acceptance has been performed on macOS Apple Silicon; interactive Windows/Linux desktop verification remains device-dependent.

This release has not completed interactive Windows/Linux acceptance, multi-display/fullscreen and sleep/wake coverage, or every macOS tray/third-party menu interaction.

GPU support: macOS IOKit, Linux AMD sysfs or NVIDIA SMI, Windows NVIDIA SMI. Drivers must expose the relevant counters. Missing GPU, battery or power telemetry is shown as unavailable. Linux displays the first system battery; Windows uses system battery status.

macOS packages are ad-hoc signed and not notarized. Windows packages are unsigned. Operating systems may require explicit approval to open downloaded apps. Update archives are signed independently of OS code signing.

Language defaults to the system preference. Choose **System default / 中文 / English** in Settings; changes are saved automatically. Review is required before moving selected items to the system trash. Emptying the trash requires explicit confirmation; process termination requires a separate confirmation naming the process and PID.

See the repository README for installation, dependencies and measurement details.
