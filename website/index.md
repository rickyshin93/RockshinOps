# RockshinOps

A free, open-source desktop toolkit. Monitor your system, find apps and files with Launcher, and bring order to your Mac menu bar.

[Official website](https://rickyshin93.github.io/RockshinOps/) · [GitHub](https://github.com/rickyshin93/RockshinOps) · [MIT](https://github.com/rickyshin93/RockshinOps/blob/main/LICENSE)

## One shortcut. Straight to what you need.

Search apps and local filenames together. Type a name, a few initials or Chinese pinyin, then open the result without hunting through folders.

## A calmer menu bar. Everything still within reach.

Organize your Mac’s menu bar into visible, hidden and always-hidden sections. Drag icons into place and bring back just the one you need.

## See more. Find faster. Stay in control.

Live system monitoring is just the start. Launch what you need, organize your menu bar, inspect busy processes and reclaim disk space.

## Find what’s busy.

Sort processes, group resource use by app, and inspect TCP/UDP connections. Find listening developer ports, save favorites and copy addresses.

## See where the space went.

Scan disk usage, browse large folders and files, and review selected items before moving them to the Trash. On macOS, review apps and their possible associated files too.

## Lives in your menu bar.

Keep the metrics you choose in the menu bar. Closing the window only hides it; native sampling keeps running.

## Stays on your machine.

System readings and file search stay on your computer. No account or hosted monitoring service. Metric history stays in memory.

## A closer look at your hardware.

New in v0.2.12 for macOS: dedicated Sensors and Devices pages for hardware readings and everyday connections.

## Temperature, with context.

CPU, GPU, SSD and battery temperature trends, color-coded readings, system thermal pressure and detailed fan speeds. Missing readings stay explicit; availability depends on the Mac model.

## Your everyday connections, together.

Wi-Fi signal and nearby networks, paired Bluetooth devices, sound outputs and volume in one place. Network names and scans need Location permission; battery levels appear only when available.

## Download RockshinOps.

Free for macOS, Windows and Linux. No account or subscription required.

Enable Launcher and choose a shortcut in the app. It starts off by default.

macOS only. Accessibility allows icon movement; Screen Recording supplies icon previews.

RockshinOps v0.2.12 interface with demonstration data. Features and readings vary by platform and hardware.

Version 0.2.12

- [Apple Silicon](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.12/RockshinOps.0.2.12.arm64.dmg)
- [Intel](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.12/RockshinOps.0.2.12.amd64.dmg)
- [Installer .exe](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.12/RockshinOps.Setup.0.2.12.amd64.exe)
- [Debian / Ubuntu .deb](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.12/rockshinops_0.2.12_amd64.deb)
- [Archive .tar.gz](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.12/rockshinops-0.2.12-linux-amd64.tar.gz)

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
