# LocalOps status

## Doing

- 2026-10-28: Preparing v0.2.2 PR and release for cleanup and macOS app removal, merged with signed updates. Go and frontend checks and the signed macOS package passed; the packaged cleanup page opened. Native destructive-action, tray lifecycle and public-feed upgrade acceptance remain pending.

- 2026-10-27: Added macOS Full Disk Access settings access in the installed-app list and visible permission-error guidance, including add-app and quit/reopen instructions. Native System Settings navigation verified; Go tests, frontend tests/typecheck/build and generated binding passed. LocalOps button interaction remains unverified after the prior automatic approval rejection; continuing in dev mode without packaging.
- 2026-10-27: REA verified AppDelete's selected-app branches, protection guards, auxiliary modes, 15 mode/sort combinations and nine special-filename cases. LocalOps fixes include unlimited name depth, missing-ID/cache detection and canonical Unicode matching. Shared permission failures now appear as expandable path/error coverage details, separately from candidates. Current real-app scans retain all 27 captured AppDelete paths; quick results take 49–73 ms and complete scans 11.3–13.5 s. Automated validation passed; LocalOps native validation remains blocked by the prior control rejection.
- 2026-10-27: Implemented unlimited system-disk exploration with recursive folder navigation, stable scan animation, item-based progress, removed page footer, selected-item cleanup, confirmed permanent trash emptying, and macOS app uninstall with quick candidate display, unlimited parallel background search, select all, an independent operation lock, and a denser installed-app list with larger icons. Desktop validation remains blocked because computer control was not approved for LocalOps Dev.
- 2026-10-28: Released [v0.2.1](https://github.com/rickyshin93/local-ops/releases/tag/v0.2.1) with signed automatic updates and localized Settings/menu checks. [Four-target CI](https://github.com/rickyshin93/local-ops/actions/runs/40654891951), downloaded checksums/signatures, and a macOS fixture upgrade/relaunch passed. Real GitHub-source native acceptance remains blocked by this machine’s download-host connection timeout; this is separate from the successful loopback test.
- 2026-10-27: Final native tray appearance regression remains unverified: the menu-bar preview tool returns an old cached image. Sampling, preferences, language persistence, window close/reopen, Lucide rendering and native App icons are verified.

## Next

- None.

## Backlog

- Windows/Linux interactive desktop acceptance beyond CI native collector smoke tests.

## Done

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
