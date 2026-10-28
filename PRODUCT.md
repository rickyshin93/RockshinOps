# LocalOps

<!-- impeccable:product-schema 1 -->

## Platform

web

The interface runs inside MyGo's desktop WebView. macOS, Windows and Linux have system collectors. CI builds four target packages and runs native sampling checks; interactive desktop acceptance is tracked separately in STATUS.md.

## Users

A developer who wants to see remaining disk capacity in the Mac menu bar and inspect live system activity in a quiet two-pane monitor.

## Product Purpose

Keep current system activity understandable at a glance. Overview cards lead to detailed hardware, process, connection and App views. Disk capacity remains visible in the menu bar.

## Capabilities and Constraints

- Stack: MyGo + Vite + TypeScript, with React Settings and shared i18next catalogs. Native sampling and menu-bar updates are owned by Go.
- The accepted scope includes overview/history, CPU, memory, GPU, disk capacity/I/O, network, battery/energy, processes, connections, Apps, space analysis and machine information.
- Navigation is on the left and content on the right, matching the user's 2026-10-26 reference screenshot. No standalone top bar; keep native traffic lights over the sidebar.
- Space exploration scans from the system disk root without time or entry limits and allows navigation into its folders; inaccessible paths are reported. Cleanup moves selected items to the system trash and can permanently empty the trash after confirmation. OS configuration changes remain outside the authorization.
- On macOS, installed apps and possible related files can be reviewed and individually selected for removal to the system trash; matching does not guarantee discovery of all residual files.
- Capacity means space available for writing on the filesystem containing the user's home. Units are decimal; reclaimable space is not included.
- All production readings come from the system. Failures replace the displayed value with an explicit unavailable state.

## Brand Commitments

LocalOps follows a macOS-native direction: real sidebar vibrancy, system blue for selection and actions, Apple system colors for metric identity, Health-style cards and soft shadows. `DESIGN.md` is the interface guide. Keep live data readable, fixed light styling, left navigation, native window controls and one content scroller. Do not invent system-health scores.

## Evidence on Hand

The Go collector, its native snapshots, and the verified Pulse research. No performance benchmark claims are made. Platform measurement definitions are recorded in docs/platforms.md.
