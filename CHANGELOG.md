# Changelog

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
