# LocalOps

[官网](https://rickyshin93.github.io/local-ops/?lang=zh) · [English](README.md) · [下载](https://github.com/rickyshin93/local-ops/releases) · [构建状态](https://github.com/rickyshin93/local-ops/actions/workflows/build.yml)

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
- 分析缓存、日志、下载和废纸篓；可将选中的扫描项目移入系统废纸篓，或确认永久清空配置的废纸篓目录。扫描不完整的目录不能清理；不终止进程。
- 从系统磁盘根目录按逻辑大小透视并逐层浏览，先统计项目再显示按项目数计算的进度，透视没有时间或条目数上限；无法读取的位置会被标注，不跟随符号链接，多次挂载的路径可能重复计入。macOS 可检查已安装的应用包及按标识符在主目录、Library 和系统临时及缓存目录发现的关联候选文件；上述目录及应用目录中的各级子目录均按应用名称或声明名称的文件名边界匹配，不限制深度。选中项目可移入系统废纸篓。常见位置的候选项会先显示，其余搜索继续进行，且不受磁盘透视锁阻塞。没有 Bundle ID 的应用也支持按名称搜索，等价 Unicode 编码的名称可以匹配，返回路径保留原样。搜索不设时间或数量上限，但受权限、符号链接、其他应用包和无法通过名称识别的文件影响，结果仍可能不完整，匹配结果可能属于共享数据，需逐项核对；无法读取的位置和系统错误归入可展开的覆盖范围详情，与 App 候选项目分开显示。操作前须退出应用。系统组件、其他用户、任意名称或受限位置中的残留可能遗漏。
- macOS 卸载搜索遇到权限限制时会显示授权引导。「打开完全磁盘访问权限」直接跳到系统授权页，已安装 App 列表也保留此入口。在系统设置中允许 LocalOps（若没有则点「+」添加），退出并重新打开 LocalOps 后再搜索；部分系统目录仍可能受限。
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

每次发布附带 `SHA256SUMS.txt`。从 v0.2.1 开始，应用启动后及持续运行期间每天检查 GitHub Releases；也可通过**设置 → 检查更新…**、应用菜单或托盘菜单手动检查。更新窗口提供说明、下载进度、签名验证和重启选择；默认确认后安装，主动勾选后才启用自动安装。

v0.2.0 用户需先手动安装一次 v0.2.1。自动更新要求安装位置可写；Linux 系统软件包通过包管理器升级。检查和下载需要能访问 GitHub；更新签名不改变上述操作系统签名和公证状态。

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

宣传网站位于 `website/`，默认英文，通过 `?lang=zh` 打开中文。运行 `python3 -m http.server 4177 --directory website` 后访问 `http://localhost:4177/` 即可预览；不需要构建或 CDN。推送网站改动后，`pages.yml` 自动部署到 [GitHub Pages](https://rickyshin93.github.io/local-ops/)。

推送 `main`、提交 PR 或手动触发后，GitHub Actions 会测试并构建 macOS arm64、macOS amd64、Windows amd64 和 Linux amd64。安装包保留为构建产物。

发布时修改 `package.json` 的版本并提交，再推送对应 `v<版本>` 标签。同时在 `CHANGELOG.md` 增加对应的 `## <版本>` 条目。CI 核对标签版本，等待四个平台的测试、构建和更新包验签全部通过后，再创建 Release、上传安装包、更新包和校验文件。发布使用仓库内置 `GITHUB_TOKEN`；标签构建还需要仓库 Secret `MYGO_UPDATER_PRIVATE_KEY`。PR 和非标签构建不注入签名密钥。

更新公钥和发布仓库位于 `mygo.config.ts`；原始私钥须在 Git 之外安全备份，更换密钥会使已安装版本无法接受后续更新。本机签名构建通过 MyGo 的 `MYGO_UPDATER_PRIVATE_KEY` 凭据入口提供私钥，禁止提交私钥。检查间隔由 `localops.json` 的 `updateIntervalSeconds` 定义，默认 86400 秒。首版通过 GitHub 的 `releases/latest/download/update-<平台>.json` 分发完整更新包，不生成增量补丁。

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
