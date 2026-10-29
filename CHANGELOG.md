# Changelog

## 0.2.5

- Unify app, executable, installer and configuration names as RockshinOps; preserve existing settings during the transition. Older LocalOps installations require one manual installation.
- Improve page switching, idle preloading, retained tables, incremental history, native collection concurrency and cached rendering.
- Add macOS per-connection transfer rates, byte counters and TCP RTT, including long-latency parsing fixes.
- Refine menu-bar selection and pointer dragging with live reordering, settling animations and failure rollback; retain successful icon previews.
- Remove reload and developer-tool actions from the production View menu; improve macOS resize pacing and About-page refreshes.
- Publish bilingual website metadata, structured data, sitemap and Agent-readable Markdown/llms.txt.
- macOS builds remain ad-hoc signed: installing an updated build can require removing and re-adding stale Accessibility or Screen Recording permissions.

### 中文

- App、可执行文件、安装包和配置统一为 RockshinOps，保留旧设置；旧 LocalOps 安装需要手动安装一次。
- 优化切页、空闲预加载、列表复用、增量历史、原生并行采集和渲染缓存。
- 新增 macOS 连接流量、速率及 TCP RTT，并修复大延迟数值解析。
- 优化菜单栏选中态和拖动动画，支持实时让位与失败回退，保留已捕获的图标。
- 正式版本移除刷新和开发者工具入口，优化窗口缩放与关于本机刷新。
- 发布双语官网 SEO、结构化数据、站点地图和 Agent 可读内容。
- macOS 仍采用临时签名；覆盖安装后可能需要移除并重新添加辅助功能或屏幕录制授权。

## 0.2.4

- Refresh RockshinOps branding, navigation, settings and hardware views, with smoother macOS resizing.
- Rebuild Processes and Connections with compact tables, sorting, search, pause/resume and real process history charts.
- Add confirmed Quit and Force Quit with process identity and OS permission checks. Windows supports Force Quit only.
- Add macOS menu-bar icon sections, previews, reordering, temporary reveal and appearance controls.
- Save settings automatically and optionally hide the Dock icon when closing the main window.
- Keep LocalOps installer and executable names for compatibility with existing signed updates and relaunch paths.

### 中文

- 更新 RockshinOps 品牌、导航、设置和硬件页面，优化 macOS 窗口缩放。
- 重做进程与连接列表，支持排序、搜索、暂停和真实进程历史图表。
- 新增确认后退出与强制退出，核对进程身份并遵循系统权限；Windows 仅支持强制退出。
- 新增 macOS 菜单栏图标分区、预览、排序、临时显示和外观设置。
- 设置自动保存，可选择关闭窗口时隐藏 Dock 图标。
- 安装包和可执行文件保留 LocalOps 名称，兼容旧版签名更新与重启路径。

## 0.2.3

- On macOS, optionally use the system proxy for update checks and downloads. PAC scripts and authenticated proxies are not supported.

### 中文

- macOS 新增更新使用系统代理的选项，适用于检查和下载更新；不支持 PAC 自动代理脚本及需要认证的代理。

## 0.2.2

- Explore disk folders and review selected files before moving them to the system trash.
- Review installed macOS apps and possible associated files before moving selected items to the trash. Search results may be incomplete and name matches may be shared.
- Show scan permission limits and provide access to macOS Full Disk Access settings.
- Continue to receive signed updates from GitHub Releases.

### 中文

- 浏览磁盘文件夹，核对所选项目后可移入系统废纸篓。
- 查看已安装的 macOS 应用及可能的关联文件，核对后将所选项目移入废纸篓。搜索可能不完整，名称匹配的文件可能由其他应用共享。
- 显示搜索权限限制，并提供前往 macOS 完全磁盘访问权限设置的入口。
- 继续通过 GitHub Releases 接收签名更新。

## 0.2.1

- Check for signed updates automatically after launch and daily while LocalOps is running.
- Check manually from Settings, the app menu or the tray menu. Read release notes, install, skip a version or choose a reminder; relaunch when ready.
- Updates are downloaded from GitHub Releases and verified with the public key embedded in the app. Automatic installation requires opting in.
- macOS and Windows installations must be writable by the current user. Linux system packages should be upgraded through the package manager.
- Existing v0.2.0 users must install v0.2.1 manually once to enable future updates. macOS notarization and Windows code-signing status are unchanged.

### 中文

- 启动后及持续运行期间每天自动检查签名更新。
- 设置、应用菜单和托盘菜单新增“检查更新”。支持查看更新说明、安装、跳过版本或稍后提醒，重启时机由你决定。
- 从 GitHub Releases 下载，并用应用内置公钥验签；默认确认后安装，自动安装需要主动勾选。
- macOS / Windows 安装目录须对当前用户可写；Linux 系统软件包请通过包管理器升级。
- v0.2.0 用户需手动安装一次 v0.2.1，此后才能使用自动更新。系统代码签名及公证状态保持不变。
