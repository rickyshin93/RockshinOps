RockshinOps is a local desktop system monitor with persistent menu bar metrics, English/Chinese UI, process and app inspection, disk space analysis, selected cleanup and macOS app removal.

### New in v0.3.0

- On macOS, record and search local clipboard history for text and images. Pin records, save text as reusable snippets, and access history from the sidebar, menu bar, or Launcher with `⌘4`.
- Add timed Keep Awake to the macOS menu bar and sidebar visibility settings for Calculator and Clipboard.
- Improve Launcher app activation, thumbnail reuse and keyboard navigation. Refine Calculator wrapping, leading-zero handling and layout.
- Upgrade to MyGo 0.9.0 and calculate total CPU usage directly from busy ticks.

### 中文

- macOS 新增本地剪贴板历史，可搜索文本和图片、置顶记录、将文本另存为片段，并从侧栏、菜单栏或启动器 `⌘4` 打开历史。
- macOS 菜单栏新增定时保持唤醒。设置中可显示或隐藏侧栏的计算器和剪贴板入口。
- 优化启动器打开应用、缩略图复用和键盘切换，改进计算器长算式换行、前导零和布局。
- 升级 MyGo 0.9.0，并直接使用忙碌计数计算总 CPU 使用率。

**Known issue / 已知问题：** Calculator can drop continuation input while saving history. / 计算器保存历史期间可能丢失连续输入。

### Upgrade notice / 升级说明

Old LocalOps installations require a manual installation of RockshinOps because their updater relaunches the old executable path. Linux system packages should be updated through the package manager.

旧 LocalOps 用户需手动安装 RockshinOps；Linux 系统软件包请通过包管理器更新。

macOS builds remain ad-hoc signed and not notarized; Accessibility or Screen Recording permissions may need to be re-granted after replacement.

### Downloads

- **macOS Apple Silicon:** the `arm64.dmg` file.
- **macOS Intel:** the `amd64.dmg` file.
- **Windows x86_64:** the `rockshinops-setup-0.3.0-amd64.exe` installer.
- **Linux x86_64:** the `.deb` package or `.tar.gz` archive (extract and run `./rockshinops`; no install script is included).
- `SHA256SUMS.txt` contains checksums for all packages. `update-*.json` and the macOS/Windows `.tar.gz` files are consumed by the updater; choose the installers above for manual installation.

### Platform status

System collectors are implemented for macOS, Windows and Linux. All four native CI runners sample CPU, memory, disk, network, processes and a known TCP socket. Interactive desktop acceptance has been performed on macOS Apple Silicon; interactive Windows/Linux desktop verification remains device-dependent.

This release has not completed interactive Windows/Linux acceptance, multi-display/fullscreen and sleep/wake coverage, or every macOS tray/third-party menu interaction.

GPU support: macOS IOKit, Linux AMD sysfs or NVIDIA SMI, Windows NVIDIA SMI. Drivers must expose the relevant counters. Missing GPU, battery or power telemetry is shown as unavailable. Linux displays the first system battery; Windows uses system battery status.

macOS packages are ad-hoc signed and not notarized. Windows packages are unsigned. Operating systems may require explicit approval to open downloaded apps. Update archives are signed independently of OS code signing.

Language defaults to the system preference. Choose **System default / 中文 / English** in Settings; changes are saved automatically. Review is required before moving selected items to the system trash. Emptying the trash requires explicit confirmation; process termination requires a separate confirmation naming the process and PID.

See the repository README for installation, dependencies and measurement details.
