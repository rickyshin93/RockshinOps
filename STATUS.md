# RockshinOps status

## Doing

- 2026-10-29: Preparing v0.2.5 with all current source changes authorized for commit and publication. Go tests, metrics race checks, 64 frontend tests, typecheck, frontend build and bilingual website asset/version checks passed. macOS arm64 app/DMG packaging passed. First CI passed Windows/Linux/macOS Intel but exposed macOS QUIC rows breaking all connection metrics; reproduced with a failing regression, fixed unsupported QUIC flow isolation. Revalidation, signed release, asset checks and website deployment pending.
- 2026-10-29: Whole-app performance, independent sidebar feedback, idle preloading, incremental history and retained tables are implemented. Prior browser and macOS page/cache/close-reopen checks passed; current-release native tray and full lifecycle acceptance remain open. Controlled warm navigation averaged 231 → 33 ms, not native end-to-end timing. See [measurements](docs/performance.md).
- 2026-10-29: macOS connection rates, byte totals and TCP RTT use uniquely matched native sockets. Parser, reset, IPv6, long-RTT and real localhost-transfer checks passed; prior native sorting/search/pause acceptance passed. Windows/Linux per-socket metrics remain unsupported.
- 2026-10-29: Menu selection and pointer dragging now provide live neighbor reflow, cross-section settling and rollback. Browser move/cancel/error/wrapped-row/keyboard/Reduced Motion checks passed; native continuous-drag frame pacing, third-party menus, shortcuts, multi-display/fullscreen and sleep/wake remain open. Icon cache retention, overflow placement and permission recovery were checked locally. Installed Accessibility and Screen Recording stale entries were restored with explicit authorization; user confirmed dragging and actual icon previews returned. Ad-hoc signing may invalidate permissions on replacement. See [implementation and evidence](docs/menu-bar-management.md).
- 2026-10-29: Production reload/devtools menu removal, display-synced window resizing, incremental About refresh, icon sizing, shared page headers/backgrounds and Dock visibility preferences are implemented and have scoped automated/native/browser checks. Final-package native shortcut, Dock/tray appearance and close/reopen regression remain open; older macOS timing path and other desktops are unverified.
- 2026-10-29: Cleanup/uninstall candidate discovery, Full Disk Access guidance, disk browsing and confirmed process controls have automated and scoped native checks. Destructive cleanup/uninstall acceptance, complete real-app residual coverage and Windows/Linux desktop behavior remain open. Protected paths and shared-name candidates remain explicit limitations.

## Next

- None.

## Backlog

- Windows/Linux interactive desktop acceptance beyond CI native collector smoke tests.

## Done

- 2026-10-29: Published [RockshinOps v0.2.4](https://github.com/rickyshin93/local-ops/releases/tag/v0.2.4) with all current UI, process-control and macOS menu-bar changes. [Four-target signed release CI](https://github.com/rickyshin93/local-ops/actions/runs/40843411356) and main CI passed. Downloaded all assets: 12 SHA-256 checksums, four update signatures, version/branding and installed executable identities verified. Website deployed and live 0.2.4 links checked. LocalOps filenames preserve existing updater relaunch paths; a real installed-version upgrade and remaining desktop/lifecycle cases below were not revalidated.

- 2026-10-28: Released [v0.2.1](https://github.com/rickyshin93/local-ops/releases/tag/v0.2.1) with signed updates. Four-target CI and a loopback fixture upgrade passed. Public 0.2.1 → 0.2.2 upgrade passed with an explicitly supplied proxy; the normal macOS system proxy was not automatically used.

- 2026-10-29: Renamed current product surfaces to RockshinOps, added the blue R/green-dot logo and rounded graphite/bright-blue brand-and-product wordmark with a 2px gap, and tightened sidebar top spacing. Native Dev window/branding, 900px layouts, bilingual website, identity/config preservation and Go/web checks passed. Production builds, installed upgrades, Dock/tray appearance and Windows/Linux desktop branding remain unverified; no publication.

- 2026-10-29: Removed the search input focus border and added ascending/descending header sorting for the five populated connection columns, including numeric ports and sort retention after filtering/refresh. Native Dev search, sort directions and resumed refresh verified; Go tests, frontend tests and typecheck passed. Dev-only; no production build or packaging, and no new tray/lifecycle or Windows/Linux acceptance.

- 2026-10-29: Unified desktop colors around the original bright-green sidebar tile (#24b73f), with coral/orange/blue/slate accents shared by navigation, charts and About sections; action controls and sidebar selection use softer blue, and headings match icon base colors. Native Dev sidebar/CPU rendering and action-text contrast checked; typecheck and frontend tests passed. No production build or packaging per user request.

- 2026-10-29: Increased right-pane body/labels/helper text to 15/14/13px, darkened secondary text and strengthened card separators. Dev native settings/overview/process-table and 900×600 English fixture layouts checked; typecheck passed. Right-pane heading/settings icons now share the sidebar’s Apple-style colored tiles, highlights and shadows. Removed the redundant Energy footer notice; native settings icons and Energy bottom spacing checked, frontend tests passed. Production build and packaging skipped per user request.

- 2026-10-29: Rebuilt About This Computer with a native device image and hardware/software/display/storage groups using real metadata. Chinese/English layouts, narrow-window scrolling and native CPU/GPU core values verified; display dimensions explicitly use logical resolution. Refined this page with gray panels, semibold field labels and coordinated coral/blue/orange/green classification colors (no purple); native Dev appearance and typecheck passed.

- 2026-10-27: Published the [bilingual website](https://rickyshin93.github.io/local-ops/) with English default, responsive layouts and released installer links. Public rendering, language switching and asset integrity verified; [Pages deployment passed](https://github.com/rickyshin93/local-ops/actions/runs/40378884753). Acceptance: goals/2026-10-27-public-website.md.
- 2026-10-27: Released [v0.2.0](https://github.com/rickyshin93/local-ops/releases/tag/v0.2.0) with five installers after all four native build/test jobs passed. Downloaded packages match their SHA-256 hashes; corrected the manifest for GitHub's filename normalization and verified all platform collection paths preserve bytes with stable names.
- 2026-10-27: Implemented Windows/Linux CPU, memory, disk, network, process and connection collection, platform directories and optional battery/GPU adapters. Native smoke tests and installable packages passed on macOS arm64/amd64, Windows amd64 and Linux amd64 ([CI evidence](https://github.com/rickyshin93/local-ops/actions/runs/40372365187)); interactive Windows/Linux desktop acceptance remains below.
- 2026-10-27: Added English/Chinese localization and persisted system/en/zh selection. Native system-language default, live UI/menu switching and restart persistence verified; restored the local app to system default.
- 2026-10-27: Published the source with MIT licensing, dependency notices, English/Chinese READMEs and GitHub Actions for four-target builds and tagged releases.

- 2026-10-27: Removed the sidebar footer's hide-to-menu-bar button and its event/style references; native Dev rendering and required build checks passed.
- 2026-10-27: Fixed title-bar double-click restore by sharing AppKit zoom state with title regions and system controls. Native repeated/mixed zoom and maximized restart restoration, close/reopen, tray text and required build checks passed.
- 2026-10-26: Added persisted menu-bar Settings, custom sorting controls, redesigned the sidebar, and implemented native dragging and animated double-click zoom/restore, including restore after maximized restart. Native acceptance and independent review passed.
- 2026-10-26: Fixed narrow-window dot charts by retaining circular geometry at every width. Browser rendering measurements, frontend checks and updated macOS package passed; native post-fix verification is not included.
- 2026-10-26: Selected project name LocalOps, and MyGo + Vite for future cross-platform expansion.
- 2026-10-26: Expanded LocalOps 0.2 into a Pulse-style two-pane monitor with real hardware metrics/history, processes, connections, running Apps, read-only space analysis and machine information. Native navigation, search/sort, minimum-size scrolling, hidden-window sampling, reopen and quit verified; app rebuilt and left running. Automated checks and finish review passed. Verification boundaries remain below.

## Known issues

- CPU temperature and App-level watt measurements are not implemented; GPU/system power availability depends on hardware.
- Space and uninstall scans can be partial when macOS denies directory access; the current uninstall comparison encountered 371 protected locations. Full Disk Access may improve coverage but does not guarantee access to all system directories. Name matches do not prove ownership or complete residual detection; REA coverage is representative, not exhaustive. Space results are logical file sizes, not guaranteed reclaimable space.
- Actual sleep/wake and direct tray-menu interaction remain untested; tray capacity and configurable text/icon-only states were observed in earlier builds through Ice; its cached preview cannot establish current release appearance. Error recovery used a browser fixture, not native fault injection. The interface now uses fixed light styling.
- Direct GitHub download connections timed out on the macOS test machine during v0.2.1 acceptance; authenticated API downloads succeeded and all four update archives passed signature verification. Native public-source upgrade acceptance remains open.
- Windows/Linux interactive desktop behavior and battery/GPU hardware remain outside local acceptance. macOS packages are ad-hoc signed and not notarized; Windows packages are unsigned.
