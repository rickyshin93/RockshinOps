# RockshinOps

RockshinOps is a personal system-monitoring desktop app built with Go, MyGo, and a Vite TypeScript frontend. Native collectors target macOS, Windows and Linux. CI builds macOS amd64/arm64, Windows amd64 and Linux amd64. Desktop acceptance and hardware-specific availability must be reported separately from compilation.

- Read `STATUS.md` before implementation. Keep it as the single progress tracker.
- System sampling and tray updates belong in Go. The frontend consumes generated MyGo bindings; do not duplicate native sampling in JavaScript or introduce an HTTP server for IPC.
- Keep platform-specific sampling in Go files with OS suffixes. Add platform implementations when needed; do not fake unsupported metrics.
- Disk capacity means bytes available to the current user. Display decimal units and label the metric explicitly. Do not sum APFS volumes sharing the same container.
- Metrics must carry their actual source and unavailable state. CPU and throughput use differences between native counters; the first sample is a baseline, not a zero reading. Per-process CPU uses 100% per logical core.
- Connection/App inspection and space analysis are read-only. Process Quit/Force Quit is explicitly user-authorized: require confirmation naming the process and PID, revalidate its sampled identity, and use current-user OS permissions without elevation. Selected cleanup and macOS app-uninstall items may be moved to the system trash after review, and the trash may be permanently emptied after explicit confirmation. Do not add process termination, privilege escalation, or login-item changes without explicit authorization.
- Root `rockshinops.json` defines sampling and other adjustable app parameters. Embed it for packaged builds; no environment-based parameter overrides.
- UI strings belong in the shared English/Chinese catalogs in `internal/localization/`. Frontend translation uses i18next; React Settings uses react-i18next. Keep metric status codes independent of display language.
- User language and menu-bar preferences are saved in RockshinOps' user configuration directory. Root `rockshinops.json` owns their first-run defaults. In-page selectors must be custom themed controls with keyboard support; do not use native HTML selects.
- Closing the main window hides it; explicit Quit terminates the app. Native sampling must continue independently of WebView timers.
- Updates use MyGo's signed GitHub Releases feed. Tagged releases must include verified update manifests and archives for every target; keep the signing private key outside the repository. Framework update-window translations remain owned by MyGo.
- Use a bounded full-height shell, normal Flex/Grid layout, and one main content scroller. No sticky positioning or document scrolling. The native title bar is hidden; traffic lights live above the left navigation. Do not add a full-width top bar.
- Before delivery run Go tests, frontend type checking, frontend build, and a macOS package build. Verify actual native tray text and close/reopen behavior in the desktop app. Report unverified platforms and lifecycle cases explicitly.
