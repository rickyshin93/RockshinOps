# LocalOps

[![Build and release](https://github.com/rickyshin93/local-ops/actions/workflows/build.yml/badge.svg)](https://github.com/rickyshin93/local-ops/actions/workflows/build.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A local desktop system monitor built with **Go, MyGo and TypeScript**. Keep useful metrics in the menu bar, inspect live activity, and see what takes up disk space.

[Website](https://rickyshin93.github.io/local-ops/) · [简体中文](README.zh-CN.md) · [Download](https://github.com/rickyshin93/local-ops/releases) · [Project status](STATUS.md)

## Platform status

| Package | Architecture | Monitoring status |
| --- | --- | --- |
| macOS | Apple Silicon / arm64 | Native collectors; interactive desktop acceptance on Apple Silicon |
| macOS | Intel / x86_64 | Same macOS collectors; CI build and automated tests |
| Windows | x86_64 / amd64 | Native CPU, memory, disk, network, process and connection collectors; CI sampling checks |
| Linux | x86_64 / amd64 | Native CPU, memory, disk, network, process and connection collectors; CI sampling checks |

All four targets run real collector smoke tests in CI. Interactive desktop acceptance has been performed on macOS Apple Silicon; Windows/Linux desktop interaction and hardware-specific telemetry need device coverage. [Platform collection contract](docs/platforms.md).

## Features

- CPU usage by core, memory composition/pressure/swap, GPU counters, disk capacity and I/O, physical network traffic, battery and available hardware power telemetry.
- Actual history for the last minute, 5 minutes, 15 minutes or hour. Pausing freezes the view while native sampling continues. History lives only in memory for the current session.
- Searchable process and running-app lists with CPU and physical-memory usage; on-demand TCP/UDP connection snapshots.
- Space analysis of configured caches, logs, downloads and trash. Move selected scanned items to the system trash, or confirm permanent emptying of the configured trash directory. Incomplete directories cannot be cleaned; process termination is unavailable.
- Explore the system disk from its root by logical size and navigate its directories. Disk exploration counts items before showing item-based progress and has no time or entry limit; inaccessible paths are reported, symbolic links are not followed, and mounted paths may be counted more than once. On macOS, review installed app bundles and possible associated files found by identifier across home, Library and system temporary/cache directories, and by app name or declared bundle name at filename boundaries at any depth throughout those locations and application directories, then move selected items to the system trash. Common locations appear first while the remaining search runs independently of disk exploration. Apps without a bundle identifier can still be searched by name; canonically equivalent Unicode names match while original paths are preserved. Search has no time or item limit but can be incomplete due to permissions, skipped symbolic links, application bundles, and files whose names do not identify the app; matches may be shared and must be reviewed. Inaccessible locations and OS errors appear in expandable coverage details, separately from app candidates. Running apps must be quit first. System components, other users, arbitrary names and privileged locations may be missed.
- macOS app-uninstall scans show a visible permission guide when access is denied. **Open Full Disk Access** goes directly to the system authorization page and is also available in the installed-app list. Enable LocalOps there (use **+** to add it if missing), then quit and reopen LocalOps before searching again. Some system directories may remain restricted.
- Persistent menu bar metrics: disk space, CPU, memory, GPU, network speed and battery. Close the window to keep sampling; use **Quit** to stop.
- English and Simplified Chinese. Settings defaults to **System default** and matches the OS language preferences; unsupported languages fall back to English. Save **中文** or **English** to override. The UI, native menus and tray labels change together.
- A compact macOS-style interface with Lucide icons and native app icons on macOS (generic executable icons elsewhere).

CPU temperature and per-app watt measurements are not implemented. GPU and system power availability depends on hardware and drivers. Windows supports NVIDIA telemetry through an installed `nvidia-smi`; Linux supports AMD sysfs counters and NVIDIA SMI. The GPU panel shows one device. Linux battery details cover the first system battery in sysfs; Windows uses aggregate system battery status.

## Install

Download the package for your architecture from [Releases](https://github.com/rickyshin93/local-ops/releases).

- **macOS:** open the `.dmg` and drag LocalOps to Applications. Packages are ad-hoc signed, not notarized. If macOS blocks the downloaded app, follow Apple's [instructions for opening an app from an unidentified developer](https://support.apple.com/guide/mac-help/open-a-mac-app-from-an-unknown-developer-mh40616/mac).
- **Windows:** run the `LocalOps Setup … amd64.exe` installer. It installs for the current user and requires WebView2. Packages are unsigned, so Windows may display a publisher warning.
- **Debian/Ubuntu:** install the `.deb` with `sudo apt install ./localops_…_amd64.deb`; the package declares its GTK/WebKitGTK dependencies. A desktop tray implementation is needed to expose the tray menu.
- **Other Linux desktops:** extract the `.tar.gz` and use its included `install.sh`. Install GTK 3, WebKitGTK 4.1 and an AppIndicator-compatible tray through your distribution. The archive's installer reports missing WebKitGTK dependencies.

Each release includes `SHA256SUMS.txt`. Starting with v0.2.1, LocalOps checks GitHub Releases after launch and daily while running. Use **Settings → Check for Updates…**, the app menu or the tray menu to check manually. The update window shows release notes, downloads with progress, verifies the signature and lets you choose when to relaunch. Installation requires confirmation unless you explicitly opt into automatic installation.

Install v0.2.1 manually once if you currently use v0.2.0. Updates require a writable install location; Linux system packages must be upgraded with the package manager. GitHub must be reachable. Update signing does not change the OS signing/notarization status above.

## Develop

Requirements: **Go 1.27.3**, **Bun 1.4.2**, and macOS Command Line Tools for macOS builds. Windows and Linux builds use pure Go; macOS collectors use C/Objective-C system APIs.

```sh
bun install --frozen-lockfile
bun run generate
bun run dev
```

Scripts prefer `.tools/go/bin/go` when present, otherwise the Go on `PATH`. The toolchain is not a runtime dependency.

`bun run dev:web` starts a browser preview. Live native data is available only inside the MyGo desktop app; the ordinary browser preview explicitly reports this limitation.

## Test and build

```sh
bun run test:go
bun run test:web
bun run typecheck
bun run build
```

Build on the target OS; macOS builds require that architecture's native C toolchain. Packages are written under `build/<os>-<arch>/`.

```sh
# Examples, each on its matching runner/host
sh scripts/mygo.sh build -platform darwin/arm64
sh scripts/mygo.sh build -platform darwin/amd64
sh scripts/mygo.sh build -platform windows/amd64
sh scripts/mygo.sh build -platform linux/amd64
```

[GitHub Actions](.github/workflows/build.yml) runs tests and builds all four targets on pushes to `main`, pull requests and manual dispatches. Download per-platform packages from each run's artifacts.

To publish, update `package.json`'s version, commit it, then push the matching `v<version>` tag. Add the matching `## <version>` entry to `CHANGELOG.md`. CI checks the tag, builds and tests every target, verifies signed update manifests/archives, and publishes installers, updates and SHA-256 checksums only after all builds pass. The release job uses the repository's `GITHUB_TOKEN`; tagged builds require the `MYGO_UPDATER_PRIVATE_KEY` repository secret. Pull requests and untagged builds do not receive the signing key.

The public key and release repository are in `mygo.config.ts`. Keep the original private key in a secure backup outside Git: replacing it strands existing installations. For a local signed build, supply it through MyGo's `MYGO_UPDATER_PRIVATE_KEY` credential input; never commit it. Update manifests use the standard `releases/latest/download/update-<platform>.json` feed, with full archives and no delta patches. See [MyGo's update documentation](https://github.com/egoist/mygo/blob/v0.6.4/docs/updates.md).

## Configuration and data

`localops.json` is the single file for the update check interval (`updateIntervalSeconds`, default 86400 seconds), sampling intervals, history retention, connection/scan timeouts, scan scope and resource limits. It is embedded in the app, so changes require a rebuild. `defaultMenuBarItems` and `defaultLanguage` define first-run preferences. `scanTargets` has separate `darwin`, `windows` and `linux` lists; `~/` paths resolve against the current user home.

User settings are stored at `os.UserConfigDir()/LocalOps/preferences.json` (on macOS: `~/Library/Application Support/LocalOps/preferences.json`). Existing settings without a language field follow the system. Invalid settings or failed saves report errors; saved preferences are not silently reset.

Data is read locally. No analytics, remote upload, privileged helper or HTTP IPC server is included. Permission restrictions can result in partial data.

| Metric | Meaning |
| --- | --- |
| CPU | Native counter deltas; system usage is weighted across cores. Process usage uses 100% per logical core and may exceed 100%. |
| Memory | Binary GiB/MiB. macOS uses Mach/sysctl and process `phys_footprint`. Windows/Linux use total minus OS-available memory and process RSS; shared pages can be counted multiple times. Windows swap means page files. |
| Disk capacity | Space writable by the current user on the home directory's volume, decimal MB/GB/TB. Reclaimable space is excluded; APFS shared volumes are not summed. |
| Disk I/O | macOS block-driver counters, Windows fixed-volume counters, Linux hardware block devices excluding partitions and virtual layers. Disabled or inaccessible counters are unavailable; LocalOps does not enable them in the registry. |
| Network | 64-bit interface counter deltas. macOS sums `en*`; Windows uses the hardware-interface flag; Linux uses sysfs hardware device identity. Virtual interfaces remain separate to avoid duplicate traffic. |
| Connections | macOS `lsof`, Windows connection tables or Linux procfs, limited to current permissions; no remote DNS lookup or outgoing connection to displayed addresses. |
| Space analysis | Logical file sizes, without following symlinks. Hard links/APFS clones may count more than once; results are not guaranteed reclaimable space. |

## Source layout

- `website/`: bilingual static promotional site, deployed to GitHub Pages by `pages.yml`. English is the default; `?lang=zh` selects Chinese. Preview with `python3 -m http.server 4177 --directory website` and open `http://localhost:4177/`. No build step, CDN or runtime dependencies are required.
- `internal/metrics/`, `internal/storage/`: native collection, rates, history and read-only inspection.
- `internal/localization/`: shared English/Chinese catalogs and native translations.
- `main.go`, `preferences.go`: lifecycle, native menus, preferences and typed IPC.
- `src/`: TypeScript views and React Settings using react-i18next. `src/mygo.ts` is generated.
- `scripts/`, `.github/`: development, packaging and release automation.
- [DESIGN.md](DESIGN.md): interface conventions; [STATUS.md](STATUS.md): current progress and verification limits.

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md). Licensed under [MIT](LICENSE); dependency licenses are in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
