# RockshinOps

RockshinOps 是免费开源的桌面系统监控工具，支持 macOS、Windows 和 Linux。查看 CPU、内存、磁盘、网络、进程与连接，在菜单栏持续掌握系统活动。

[官网](https://rickyshin93.github.io/RockshinOps/zh/) · [GitHub](https://github.com/rickyshin93/RockshinOps) · [MIT](https://github.com/rickyshin93/RockshinOps/blob/main/LICENSE)

## 看清整个系统。

CPU、内存、磁盘与网络，一处掌握。查看每个核心的活动，回顾本次运行中最近一小时的变化。

## 找到忙碌的源头。

查看进程，按应用归集资源占用，检查当前连接。顺着资源消耗，找到对应的活动。

## 知道空间去了哪里。

查看磁盘空间占用，核对所选项目后可移入系统废纸篓。

## 关上窗口，仍然心中有数。

把关心的指标留在菜单栏。隐藏窗口后，原生采样继续运行；再次打开时，变化尽在眼前。

## 为你的电脑，准备好了。

选择你的平台。无需账户、订阅，也不用配置云端控制台。

当前版本 — V0.2.5

- [Apple Silicon](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.5/RockshinOps.0.2.5.arm64.dmg)
- [Intel](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.5/RockshinOps.0.2.5.amd64.dmg)
- [下载安装程序](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.5/RockshinOps.Setup.0.2.5.amd64.exe)
- [Debian / Ubuntu .deb](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.5/rockshinops_0.2.5_amd64.deb)
- [Linux 压缩包.tar.gz](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.5/rockshinops-0.2.5-linux-amd64.tar.gz)

采用临时签名，尚未公证。

安装包未签名，需要 WebView2。

需要 GTK 3、WebKitGTK 4.1 和桌面托盘支持。

核心采集已在四个构建目标上测试。桌面交互已在 macOS Apple Silicon 验证；Windows/Linux 桌面交互及电池、GPU 实机覆盖仍在进行。GPU 与功耗读数取决于硬件和驱动支持。

### 需要云端账户吗？

不需要。RockshinOps 在本机读取系统指标，历史数据仅保留在本次运行的内存中。无需账户，也无需配置托管监控服务。

### RockshinOps 可以移除文件吗？

分析是只读的。你可以核对所选项目并将其移入系统废纸篓；清空废纸篓需要确认。macOS 还支持核对应用和可能的关联文件。退出和强制退出进程需要单独确认。

### 应用支持中文吗？

支持。应用默认跟随系统语言。也可以在设置中选择“跟随系统”“中文”或“English”，选择后自动保存。

### 可以查看源代码吗？

可以。RockshinOps 采用 MIT 许可证，使用 Go、MyGo 和 TypeScript 构建。
