LocalOps is a local desktop system monitor with persistent menu bar metrics, English/Chinese UI, process and app inspection, and read-only disk space analysis.

### Downloads

- **macOS Apple Silicon:** the `arm64.dmg` file.
- **macOS Intel:** the `amd64.dmg` file.
- **Windows x86_64:** the `Setup … amd64.exe` installer.
- **Linux x86_64:** the `.deb` package or `.tar.gz` archive (includes a local install script).
- `SHA256SUMS.txt` contains checksums for all packages.

### Platform status

System collectors are implemented for macOS, Windows and Linux. All four native CI runners sample CPU, memory, disk, network, processes and a known TCP socket. Interactive desktop acceptance has been performed on macOS Apple Silicon; interactive Windows/Linux desktop verification remains device-dependent.

GPU support: macOS IOKit, Linux AMD sysfs or NVIDIA SMI, Windows NVIDIA SMI. Drivers must expose the relevant counters. Missing GPU, battery or power telemetry is shown as unavailable. Linux displays the first system battery; Windows uses system battery status.

macOS packages are ad-hoc signed and not notarized. Windows packages are unsigned. Operating systems may require explicit approval to open downloaded apps. No automatic updater is included.

Language defaults to the system preference. Choose **System default / 中文 / English** in Settings and save to persist an override. File analysis is read-only; LocalOps does not delete files or terminate processes.

See the repository README for installation, dependencies and measurement details.
