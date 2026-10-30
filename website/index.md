# RockshinOps

A free, open-source system monitor for macOS, Windows and Linux. CPU, memory, disk, network, processes and connections, with live readings in your menu bar.

[Official website](https://rickyshin93.github.io/RockshinOps/) · [GitHub](https://github.com/rickyshin93/RockshinOps) · [MIT](https://github.com/rickyshin93/RockshinOps/blob/main/LICENSE)

## One app for the whole system.

Start with six hardware panels at a glance, then go deeper into per-core activity and up to an hour of in-session history.

## Find what’s busy.

Sort processes, group usage by app, and see every open TCP and UDP connection.

## See where the space went.

Analyze disk usage, review what you select, then move it to the Trash.

## Lives in your menu bar.

Keep the metrics you choose in the menu bar. Closing the window only hides it; native sampling keeps running.

## Stays on your machine.

Reads native system counters locally. No account, no cloud. History is kept in memory only.

## Download RockshinOps.

Free for macOS, Windows and Linux. No account or subscription required.

Version 0.2.6

- [Apple Silicon](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.6/RockshinOps.0.2.6.arm64.dmg)
- [Intel](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.6/RockshinOps.0.2.6.amd64.dmg)
- [Installer .exe](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.6/RockshinOps.Setup.0.2.6.amd64.exe)
- [Debian / Ubuntu .deb](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.6/rockshinops_0.2.6_amd64.deb)
- [Archive .tar.gz](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.6/rockshinops-0.2.6-linux-amd64.tar.gz)

Ad-hoc signed. Not notarized.

Unsigned installer. Requires WebView2.

GTK 3, WebKitGTK 4.1 and a desktop tray required.

Core collectors are tested on all four build targets. Desktop interaction is verified on macOS Apple Silicon; Windows/Linux desktop and battery/GPU hardware coverage are still in progress. Available GPU and power readings depend on your hardware and drivers.

### Does RockshinOps need a cloud account?

No. RockshinOps reads system metrics locally. History is held in memory for the current session. There is no account or hosted monitoring service to configure.

### Can RockshinOps remove files?

Analysis is read-only. You can review and move selected items to the system trash; emptying the trash requires confirmation. On macOS you can also review apps and possible associated files. Process Quit and Force Quit require separate confirmation.

### Can I use it in Chinese?

Yes. The app follows your system language by default. Choose System default, 中文 or English in Settings; changes are saved automatically.

### Is the source code available?

Yes. RockshinOps is MIT-licensed and built with Go, MyGo and TypeScript.
