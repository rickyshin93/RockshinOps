# RockshinOps

[![Build and release](https://github.com/rickyshin93/RockshinOps/actions/workflows/build.yml/badge.svg)](https://github.com/rickyshin93/RockshinOps/actions/workflows/build.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**RockshinOps is a free, open-source desktop system monitor for macOS, Windows and Linux**, built with Go, MyGo and TypeScript. Monitor CPU, memory, disk, network, processes and connections. Keep useful metrics in the menu bar, inspect live activity, and see what takes up disk space.

[Website](https://rickyshin93.github.io/RockshinOps/) · [简体中文](README.zh-CN.md) · [Download](https://github.com/rickyshin93/RockshinOps/releases) · [Project status](STATUS.md)

Live pages reuse DOM nodes, large process/connection tables render visible rows, history updates transfer only new samples, and slow inspection reads stay independent of live metrics. Native collection remains independent of rendering. See [performance design and measurements](docs/performance.md).

## Platform status

| Package | Architecture | Monitoring status |
| --- | --- | --- |
| macOS | Apple Silicon / arm64 | Native collectors; interactive desktop acceptance on Apple Silicon |
| macOS | Intel / x86_64 | Same macOS collectors; CI build and automated tests |
| Windows | x86_64 / amd64 | Native CPU, memory, disk, network, process and connection collectors; CI sampling checks |
| Linux | x86_64 / amd64 | Native CPU, memory, disk, network, process and connection collectors; CI sampling checks |

All four targets run real collector smoke tests in CI. Interactive desktop acceptance has been performed on macOS Apple Silicon; Windows/Linux desktop interaction and hardware-specific telemetry need device coverage. [Platform collection contract](docs/platforms.md).

## Features

- Process details support confirmed Quit (SIGTERM on macOS/Linux) and Force Quit (SIGKILL; Windows uses process termination), with process-identity checks and normal OS permissions. A sent request does not guarantee a process has exited.
- **Settings → General → Open at login** controls the system login item. It defaults to enabled on the first launch of the packaged app after installation (also when upgrading from a version without this setting). Login launches stay in the menu bar; manual launches show the window. Later launches respect changes made in system settings. Development builds do not register login items. If macOS requires approval, allow RockshinOps under System Settings → General → Login Items and retry. Dragging a DMG app into Applications alone does not run registration; open the installed app once.
- CPU usage by core, memory composition/pressure/swap, GPU counters, disk capacity and I/O, physical network traffic, battery and available hardware power telemetry.
- Actual history for the last minute, 5 minutes, 15 minutes or hour. Pausing freezes the view while native sampling continues. History lives only in memory for the current session.
- Searchable process and running-app lists with CPU and physical-memory usage; on-demand TCP/UDP connection snapshots. On macOS, matched sockets include received/sent bytes, sampled download/upload rates and available TCP RTT, with sortable columns. Missing measurements remain unavailable.
- **Connections → Developer ports** lists TCP listeners by port, with exact numeric port/PID search, binding-scope filters, persistent favorites, address copy and process details. Favorite chips filter only their port (`:3000`); plain `3000` searches both port and PID. IPv4/IPv6 bindings remain visible, including dual-stack wildcard addresses. Favorites are stored in `ports.json` beside `preferences.json` and start empty. Only currently visible listeners are shown: no result does not establish that a port is free, and a binding does not prove external reachability.
- Space analysis scans the volume holding the home directory once, in parallel, by allocated size: hard links count once, symbolic links and other volumes are skipped, and progress compares bytes read with the used space. The result shows a capacity breakdown by top-level folder, a drill-down folder browser, the 50 largest files and common cleanup locations (caches, logs, downloads, trash) without rescanning. Scans can be stopped. Items at least two levels inside the home directory can be moved to the system trash after confirmation, and the configured trash can be emptied permanently after confirmation; process termination requires separate confirmation.
- On macOS, review installed app bundles and possible associated files found by identifier across home, Library and system temporary/cache directories, and by app name or declared bundle name at filename boundaries at any depth throughout those locations and application directories, then move selected items to the system trash. Common locations appear first while the remaining search runs independently of disk exploration. Apps without a bundle identifier can still be searched by name; canonically equivalent Unicode names match while original paths are preserved. Search has no time or item limit but can be incomplete due to permissions, skipped symbolic links, application bundles, and files whose names do not identify the app; matches may be shared and must be reviewed. Inaccessible locations and OS errors appear in expandable coverage details, separately from app candidates. Running apps must be quit first. System components, other users, arbitrary names and privileged locations may be missed.
- macOS app-uninstall scans show a visible permission guide when access is denied. **Open Full Disk Access** goes directly to the system authorization page and is also available in the installed-app list. Enable RockshinOps there (use **+** to add it if missing), then quit and reopen RockshinOps before searching again. Some system directories may remain restricted.
- Persistent menu bar metrics: disk space, CPU, memory, GPU, network speed and battery. Close the window to keep sampling; use **Quit** to stop. On macOS, **Settings → General → Hide Dock icon when closing** optionally removes the Dock icon while the window is closed; reopening restores it.
- macOS **Menu Bar Management** adds visible, hidden and always-hidden sections, a native overflow/search panel, icon movement, appearance settings and global shortcuts. RockshinOps’ own arrow and metric item can also be dragged in the layout, with positions preserved across restarts. The overflow panel centers under its arrow and shifts inward at screen edges; selecting an item reveals only that icon for you to click manually. **General → Temporary icon duration** defaults to 15 seconds, accepts 5–60 seconds, and keeps items visible while their menu is open. Right-click the arrow to open the app, jump to menu-bar or app settings, refresh icons, search, manage sections or quit. Quit Ice before enabling it. Accessibility permits movement/opening; Screen Recording captures only status-item windows for previews. App-specific accessibility and multi-display limitations are tracked in [STATUS.md](STATUS.md); implementation and local Ice inspection evidence are in [the integration note](docs/menu-bar-management.md).
- English and Simplified Chinese. Settings defaults to **System default** and matches the OS language preferences; unsupported languages fall back to English. Save **中文** or **English** to override. The UI, native menus and tray labels change together.
- A compact macOS-style interface built with Tailwind CSS v4, shared panel/control styles, a disk-space cleanup dashboard, Lucide icons and native app icons on macOS (generic executable icons elsewhere).

CPU temperature and per-app watt measurements are not implemented. GPU and system power availability depends on hardware and drivers. Windows supports NVIDIA telemetry through an installed `nvidia-smi`; Linux supports AMD sysfs counters and NVIDIA SMI. The GPU panel shows one device. Linux battery details cover the first system battery in sysfs; Windows uses aggregate system battery status.

## Install

Download the package for your architecture from [Releases](https://github.com/rickyshin93/RockshinOps/releases).

- **macOS:** open the `.dmg` and drag the app to Applications. Packages are ad-hoc signed, not notarized. If macOS blocks the downloaded app, follow Apple's [instructions for opening an app from an unidentified developer](https://support.apple.com/guide/mac-help/open-a-mac-app-from-an-unknown-developer-mh40616/mac).
- **Windows:** run the `RockshinOps Setup … amd64.exe` installer. It installs for the current user and requires WebView2. Packages are unsigned, so Windows may display a publisher warning.
- **Debian/Ubuntu:** install the `.deb` with `sudo apt install ./rockshinops_…_amd64.deb`; the package declares its GTK/WebKitGTK dependencies. A desktop tray implementation is needed to expose the tray menu.
- **Other Linux desktops:** extract the `.tar.gz` and run `./rockshinops` from the extracted directory. Install GTK 3, WebKitGTK 4.1 and an AppIndicator-compatible tray through your distribution. The archive does not include an installation script.

Each release includes `SHA256SUMS.txt`. Starting with v0.2.1, RockshinOps checks GitHub Releases after launch and daily while running. Use **Settings → Check for Updates…**, the app menu or the tray menu to check manually. The update window shows release notes, downloads with progress, verifies the signature and lets you choose when to relaunch. Installation requires confirmation unless you explicitly opt into automatic installation.

Install v0.2.1 manually once if you currently use v0.2.0. Updates require a writable install location; Linux system packages must be upgraded with the package manager. On macOS, **Settings → Use system proxy for updates** uses the system network proxy after saving; PAC scripts and authenticated proxies are not supported. GitHub must be reachable. Update signing does not change the OS signing/notarization status above.

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

`rockshinops.json` is the single file for the update check interval (`updateIntervalSeconds`, default 86400 seconds), sampling intervals, history retention, connection/scan timeouts, scan scope and resource limits. It is embedded in the app, so changes require a rebuild. `defaultMenuBarItems`, `defaultLanguage`, and `defaultUpdateUseSystemProxy` define first-run preferences. `scanTargets` has separate `darwin`, `windows` and `linux` lists; `~/` paths resolve against the current user home.

User settings are stored at `os.UserConfigDir()/RockshinOps/preferences.json` (on macOS: `~/Library/Application Support/RockshinOps/preferences.json`). Existing settings without a language field follow the system. Invalid settings or failed saves report errors; saved preferences are not silently reset.

Data is read locally. No analytics, remote upload, privileged helper or HTTP IPC server is included. Permission restrictions can result in partial data.

| Metric | Meaning |
| --- | --- |
| CPU | Native counter deltas; system usage is weighted across cores. Process usage uses 100% per logical core and may exceed 100%. |
| Memory | Binary GiB/MiB. macOS uses Mach/sysctl and process `phys_footprint`. Windows/Linux use total minus OS-available memory and process RSS; shared pages can be counted multiple times. Windows swap means page files. |
| Disk capacity | Space writable by the current user on the home directory's volume, decimal MB/GB/TB. Reclaimable space is excluded; APFS shared volumes are not summed. |
| Disk I/O | macOS block-driver counters, Windows fixed-volume counters, Linux hardware block devices excluding partitions and virtual layers. Disabled or inaccessible counters are unavailable; RockshinOps does not enable them in the registry. |
| Network | 64-bit interface counter deltas. macOS sums `en*`; Windows uses the hardware-interface flag; Linux uses sysfs hardware device identity. Virtual interfaces remain separate to avoid duplicate traffic. |
| Connections | Processes and connections preload once after the initial view is idle; page entry shows cached connections while refreshing stale data. Failed refreshes retain the previous list with an error. macOS `lsof` plus two `nettop` samples (`connectionSampleIntervalSeconds`, default 1 second); metrics match unique PID/protocol/local/remote tuples. TCP RTT is the OS-reported round-trip time, not a ping probe. New/reset counters have no rate baseline. Windows connection tables and Linux procfs currently provide socket metadata only. No elevation, remote DNS lookup or outgoing probe. |
| Space analysis | Allocated sizes, without following symlinks or leaving the volume; hard links count once. APFS clones may count more than once and snapshots/purgeable space are not attributed to folders; results are not guaranteed reclaimable space. Windows uses logical sizes. |

## Source layout

- `website/`: bilingual static promotional site, deployed to GitHub Pages by `pages.yml`. English is the default; `/zh/` serves static Chinese HTML. Preview with `python3 -m http.server 4177 --directory website` and open `http://localhost:4177/`. Run `bun run build:website` after editing the website; deployed pages have no CDN or server runtime dependencies.
- `internal/metrics/`, `internal/storage/`: native collection, rates, history and read-only inspection.
- `internal/localization/`: shared English/Chinese catalogs and native translations.
- `main.go`, `preferences.go`: lifecycle, native menus, preferences and typed IPC.
- `src/`: TypeScript views and React Settings using react-i18next. `src/mygo.ts` is generated.
- `scripts/`, `.github/`: development, packaging and release automation.
- [DESIGN.md](DESIGN.md): interface conventions; [STATUS.md](STATUS.md): current progress and verification limits.

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md). Licensed under [MIT](LICENSE); dependency licenses are in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

## Website and machine-readable documentation

The [English website](https://rickyshin93.github.io/RockshinOps/) and [Chinese website](https://rickyshin93.github.io/RockshinOps/zh/) are static HTML and work without JavaScript. [Product facts in Markdown](https://rickyshin93.github.io/RockshinOps/index.md) and [llms.txt](https://rickyshin93.github.io/RockshinOps/llms.txt) provide concise entry points for AI search and agents. They describe the same features and platform limits shown to people.

Run `bun run build:website` after editing `website/index.html` or `scripts/website-copy.ts`; it regenerates metadata, Chinese HTML, Markdown and `sitemap.xml`. Preview with `python3 -m http.server 4177 --directory website`. GitHub Pages serves this project below `/RockshinOps/`; a project-level `robots.txt` cannot control the host root. The sitemap can be submitted in Google Search Console after ownership verification. These files do not guarantee indexing or AI citations.

## Naming transition (unreleased)

New builds use RockshinOps for the app, executable and installer, with application ID `dev.rockshinops.desktop`. Install the first renamed release manually: older updaters relaunch the former executable path. Existing `LocalOps/preferences.json` and `LocalOps/menu-bar.json` are copied to the RockshinOps settings directory only when their destination is absent; originals are retained. macOS permissions may need to be granted to the renamed app again. Published 0.2.4 artifacts retain their original filenames; use the files actually listed on the release page. The update signing key is unchanged.
