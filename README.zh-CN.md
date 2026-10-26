# LocalOps

[English](README.md) · [下载](https://github.com/rickyshin93/local-ops/releases) · [构建状态](https://github.com/rickyshin93/local-ops/actions/workflows/build.yml)

使用 Go、MyGo 和 TypeScript 构建的本机监控工具：菜单栏常驻关键指标，窗口中查看系统活动、进程、运行中的 App 和磁盘空间。

## 平台范围

| 安装包 | 架构 | 功能状态 |
| --- | --- | --- |
| macOS | Apple Silicon / arm64 | 已实现原生采集，并在 Apple Silicon 做过桌面交互验收 |
| macOS | Intel / x86_64 | 使用相同 macOS 采集实现，CI 编译并运行自动测试 |
| Windows | x86_64 / amd64 | 已实现 CPU、内存、磁盘、网络、进程与连接采集，CI 执行真实采样 |
| Linux | x86_64 / amd64 | 已实现 CPU、内存、磁盘、网络、进程与连接采集，CI 执行真实采样 |

四个目标平台的 CI 均执行真实采集冒烟测试。桌面交互已在 macOS Apple Silicon 验收；Windows/Linux 桌面交互和特定硬件遥测仍需要对应设备验证。详见[平台采集契约](docs/platforms.md)。

## 当前能力

- CPU 总量与各逻辑核心、内存分类/压力/交换空间、GPU、磁盘容量与 I/O、网络接口、电池及可用的硬件功率遥测。
- 最近 1 / 5 / 15 分钟和 1 小时的真实历史。暂停仅冻结视图，后台继续采集；历史只保存在本次运行的内存中。
- 进程和运行中 App 的搜索、排序、CPU/物理内存排行；按需读取 TCP/UDP 连接快照。
- 缓存、日志、下载和废纸篓的只读空间分析，明确显示权限不足或扫描上限。不删除文件、不卸载 App、不终止进程。
- 可选菜单栏指标：磁盘剩余、CPU、内存、GPU、上传/下载速度和电池。关闭窗口继续运行，显式「退出」停止。
- **中英文界面**：设置默认「跟随系统」，按系统语言偏好匹配，不支持的语言回退英文。可手动选择「中文 / English」并保存；页面、原生菜单和菜单栏同时切换，重启后保留。
- macOS 风格界面、Lucide 图标、macOS 原生 App 图标（其他平台使用通用可执行文件图标）。

CPU 温度和 App 级瓦特计量尚未实现；GPU 与系统功率取决于硬件和驱动。Windows 支持已安装 NVIDIA 驱动提供的 `nvidia-smi`，Linux 支持 AMD sysfs 和 NVIDIA SMI；GPU 面板显示一张卡。Linux 显示 sysfs 中首个系统电池，Windows 显示系统汇总电池状态。

## 安装

在 [Releases](https://github.com/rickyshin93/local-ops/releases) 下载对应架构：

- macOS：打开 `.dmg`，拖入 Applications。当前为 ad-hoc 签名，未做 Apple 公证；如被系统拦截，参照 [Apple 官方说明](https://support.apple.com/guide/mac-help/open-a-mac-app-from-an-unknown-developer-mh40616/mac)操作。
- Windows：运行 `LocalOps Setup … amd64.exe`，按当前用户安装，需要 WebView2。安装包未签名，Windows 可能显示发布者提示。
- Debian/Ubuntu：`sudo apt install ./localops_…_amd64.deb`，由包管理器处理声明的 GTK/WebKitGTK 依赖。
- 其他 Linux 桌面：解压 `.tar.gz`，使用其中的 `install.sh`；系统需提供 GTK 3、WebKitGTK 4.1 和兼容 AppIndicator 的托盘。安装脚本会报告缺失的 WebKitGTK 依赖。

每次发布附带 `SHA256SUMS.txt`。当前没有自动更新器。

## 开发和验证

需要 **Go 1.27.3、Bun 1.4.2**；macOS 还需 Command Line Tools。macOS 原生接口使用 CGO，其他目标使用纯 Go。

```sh
bun install --frozen-lockfile
bun run generate
bun run dev

bun run test:go
bun run test:web
bun run typecheck
bun run build
```

脚本优先使用项目 `.tools/go/bin/go`，否则使用 PATH 中的 Go。`bun run dev:web` 仅为浏览器预览；实时采集需要桌面应用。

在目标系统/架构的主机上构建，产物位于 `build/<os>-<arch>/`。具体四种构建命令见 [英文 README](README.md#test-and-build)。

## CI 与发布

推送 `main`、提交 PR 或手动触发后，GitHub Actions 会测试并构建 macOS arm64、macOS amd64、Windows amd64 和 Linux amd64。安装包保留为构建产物。

发布时修改 `package.json` 的版本并提交，再推送对应 `v<版本>` 标签。CI 会核对标签版本，等待四个平台全部成功后自动创建 Release、上传安装包和校验文件。发布使用仓库内置 `GITHUB_TOKEN`，无需配置个人令牌或签名密钥。

## 参数、偏好与数据

根目录 `localops.json` 集中配置刷新周期、历史长度、超时、扫描目录与数量上限；嵌入应用，修改后需重新构建。`defaultMenuBarItems` 和 `defaultLanguage` 定义首次运行默认值。`scanTargets` 按 `darwin`、`windows`、`linux` 分别配置；`~/` 展开为当前用户目录。

用户偏好保存在 `os.UserConfigDir()/LocalOps/preferences.json`；macOS 对应 `~/Library/Application Support/LocalOps/preferences.json`。旧设置没有语言字段时默认跟随系统；损坏设置或保存失败会明确报错，不静默重置。

数据仅在本机读取，无分析上报、远端上传、提权辅助程序或 HTTP IPC 服务。权限限制可能导致数据不完整。

- 内存使用二进制 GiB/MiB；macOS 进程使用 `phys_footprint`，Windows/Linux 使用 RSS，共享页可能重复计入。Windows/Linux 系统已用为总内存减去系统报告的可用内存；Windows 交换空间指页面文件。
- 磁盘容量为用户主目录所在卷当前用户可写的空间，使用十进制 MB/GB/TB，不包含潜在可回收空间，不重复累计 APFS 共享卷。
- 进程 CPU 以单逻辑核心 100% 计，可以超过 100%；系统 CPU 按全部核心加权。
- 网络按平台识别硬件接口（macOS `en*`、Windows 硬件标志、Linux sysfs），虚拟接口单列。连接来自 macOS `lsof`、Windows 系统表或 Linux procfs，不提权、不查询远端 DNS、不连接展示的地址。
- 磁盘 I/O 使用 macOS 块驱动、Windows 固定卷、Linux 硬件块设备计数；排除分区与虚拟叠加重复。计数关闭或无权限时显示不可用，不通过注册表启用计数。
- 空间分析读取逻辑文件大小，不跟随符号链接；硬链接或 APFS 克隆可能重复计入，结果不等于可安全释放的空间。

源码结构与完整说明见 [README](README.md#source-layout)，进度和验收边界见 [STATUS.md](STATUS.md)，参与开发见 [CONTRIBUTING.md](CONTRIBUTING.md)。项目使用 [MIT 许可证](LICENSE)，依赖声明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
