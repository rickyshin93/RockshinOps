# RockshinOps

RockshinOps is a free, open-source desktop system monitor for macOS, Windows and Linux. Track CPU, memory, disk, network, processes and connections, with persistent menu bar metrics.

[Official website](https://rickyshin93.github.io/RockshinOps/) · [GitHub](https://github.com/rickyshin93/RockshinOps) · [MIT](https://github.com/rickyshin93/RockshinOps/blob/main/LICENSE)

## See the whole picture.

CPU, memory, disk and network, together. Explore per-core activity and up to an hour of in-session history.

## Find what is busy.

Inspect processes, group activity by app, and see open connections. Follow the resource usage back to its source.

## Know where it goes.

Explore disk usage, then review selected items before moving them to the system trash.

## Close the window. Keep the signal.

Keep your chosen metrics in the menu bar. Native sampling continues when the window is hidden, ready when you come back.

## Make yourself at home.

Choose your platform. No account, subscription or cloud dashboard to set up.

CURRENT RELEASE — V0.2.5

- [Apple Silicon](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.5/RockshinOps.0.2.5.arm64.dmg)
- [Intel](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.5/RockshinOps.0.2.5.amd64.dmg)
- [Download installer](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.5/RockshinOps.Setup.0.2.5.amd64.exe)
- [Debian / Ubuntu .deb](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.5/rockshinops_0.2.5_amd64.deb)
- [Linux archive.tar.gz](https://github.com/rickyshin93/RockshinOps/releases/download/v0.2.5/rockshinops-0.2.5-linux-amd64.tar.gz)

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

### Can I look under the hood?

Absolutely. RockshinOps is MIT-licensed and built with Go, MyGo and TypeScript.
