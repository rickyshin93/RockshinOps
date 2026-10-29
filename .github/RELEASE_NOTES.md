RockshinOps is a local desktop system monitor with persistent menu bar metrics, English/Chinese UI, process and app inspection, disk space analysis, selected cleanup and macOS app removal.

### New in v0.2.5

- Faster navigation with idle preloading, cached tables, incremental history and concurrent native collection.
- macOS connection download/upload rates, received/sent bytes and TCP RTT, including long-latency parsing fixes.
- Refined menu-bar selection, pointer dragging, live reordering, settling animations, rollback and retained icon previews.
- Production builds no longer expose reload or developer-tool menu actions; smoother macOS resizing and incremental About-page updates.
- RockshinOps app/executable/installer naming and bilingual website SEO with Agent-readable product documentation.

**中文：**优化全 App 切页与采集性能，补齐 macOS 连接流量和延迟，优化菜单栏拖动与选中动画，关闭生产版刷新入口，统一 RockshinOps 命名并发布双语官网与 Agent 可读内容。

### Upgrade notice / 升级说明

**Install v0.2.5 manually when upgrading from the old LocalOps app.** Older updaters relaunch the old executable path; this version renames the app, executable and installers to RockshinOps. Existing preferences are copied only when the new configuration is absent. The application identifier is now `dev.rockshinops.desktop`; the update signing key is unchanged. Linux system packages should be updated through the package manager.

**旧 LocalOps 用户请手动下载安装 v0.2.5。** 本版更改 App、可执行文件和安装包名称，旧更新器的重启路径不兼容。已有设置会在新配置不存在时迁移，原文件保留。

macOS builds remain ad-hoc signed and not notarized. If Accessibility or Screen Recording appears enabled in System Settings but denied in the app after installation, remove only the stale RockshinOps entry, add `/Applications/RockshinOps.app` again, and quit/reopen the app. This is a known local signing limitation, not a guarantee that upgrades retain permissions.

macOS 覆盖安装后，如系统开关已开但 App 提示未授权，请在相应权限列表移除旧 RockshinOps 条目，重新添加安装版并退出重开。本版仍采用临时签名，无法保证跨构建保留授权。

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
