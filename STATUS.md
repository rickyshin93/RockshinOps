# LocalOps status

## Doing

- 2026-10-27: English/Chinese localization is implemented with persisted system/en/zh preferences; native acceptance remains. Windows/Linux collectors and platform-specific UI/source descriptions are implemented; local macOS tests and Windows/Linux cross-compilation pass. GitHub CI and open-source documents are prepared; release waits for native four-platform verification.
- 2026-10-27: Replacing hand-drawn interface icons with Lucide; native rendering and package verification pending.
- 2026-10-27: Restyled to macOS-native per DESIGN.md (sidebar vibrancy via MyGo VibrancySidebar, blue selection pill, Health-style metric cards in Apple system colors); fixed WebKit button-flex children not stretching in metric cards. Checks, package and native overview rendering verified; close/reopen and tray text not re-verified (System Events automation blocked).
- 2026-10-27: App page is now a grouped list with real bundle icons served by Go via the `appicon://` scheme (NSWorkspace, 64px PNG, cached); overview App ranking uses the same icons. Native icon loading verified on overview; native App-page rendering checked only in browser (synthetic clicks blocked).

## Next

- None.

## Backlog

- Windows/Linux interactive desktop acceptance beyond CI native collector smoke tests.

## Done

- 2026-10-27: Removed the sidebar footer's hide-to-menu-bar button and its event/style references; native Dev rendering and required build checks passed.
- 2026-10-27: Fixed title-bar double-click restore by sharing AppKit zoom state with title regions and system controls. Native repeated/mixed zoom and maximized restart restoration, close/reopen, tray text and required build checks passed.
- 2026-10-26: Added persisted menu-bar Settings, custom sorting controls, redesigned the sidebar, and implemented native dragging and animated double-click zoom/restore, including restore after maximized restart. Native acceptance and independent review passed.
- 2026-10-26: Fixed narrow-window dot charts by retaining circular geometry at every width. Browser rendering measurements, frontend checks and updated macOS package passed; native post-fix verification is not included.
- 2026-10-26: Selected project name LocalOps, and MyGo + Vite for future cross-platform expansion.
- 2026-10-26: Expanded LocalOps 0.2 into a Pulse-style two-pane monitor with real hardware metrics/history, processes, connections, running Apps, read-only space analysis and machine information. Native navigation, search/sort, minimum-size scrolling, hidden-window sampling, reopen and quit verified; app rebuilt and left running. Automated checks and finish review passed. Verification boundaries remain below.

## Known issues

- CPU temperature and App-level watt measurements are not implemented; GPU/system power availability depends on hardware.
- Space analysis can be partial when macOS denies directory access. Results are logical file sizes, not guaranteed reclaimable space; no deletion is implemented.
- Actual sleep/wake and direct tray-menu interaction remain untested; tray capacity and configurable text/icon-only states were observed through Ice. Error recovery used a browser fixture, not native fault injection. The interface now uses fixed light styling.
- The final dot-geometry fix was verified in browser rendering. Native automation was unavailable for this last change, so the running instance was not restarted and its post-fix rendering remains unverified.
- Windows/Linux interactive desktop behavior and battery/GPU hardware remain outside local acceptance. macOS packages are ad-hoc signed and not notarized; Windows packages are unsigned.
