# Performance work — 2026-10-29

Sampling intervals and metric definitions are unchanged. The frontend still consumes native Go samples; missing metrics and sleep/reset gaps remain explicit.

## First pass

- Native sampling uses a separate lock from published snapshots. Current metrics, processes and history remain readable while a collector is busy; settings no longer share the sampling lock.
- Process rankings and app aggregation are recomputed only when the process sample changes. Snapshots expose that sample timestamp so the frontend can avoid fetching an unchanged process list.
- macOS process conversion reads C records in place rather than copying each large record onto the Go heap. Returned strings and values remain owned by Go.
- Bootstrap reuses an available native snapshot. Regular history updates request only samples newer than the last accepted timestamp; retention and gaps are preserved. Hidden views skip history requests and catch up on visibility changes. Updates arriving during a request coalesce to the newest snapshot.
- Sidebar selection stays independent. Page renders are batched per animation frame. Live metric pages and inspection tables patch changed DOM nodes; process rows keep their identity across sorting and use a longest ordered subsequence to minimize moves. React forms retain their separate renderer.
- Chart timestamps are parsed once per immutable sample. Connection sorting reuses an Intl.Collator, empty queries avoid search-string construction, and connection row labels are translated once per render.
- React settings/menu-bar code is split out of the initial script and loaded during startup idle time, alongside the existing process/connection preloads.
- The native tray renderer retains one successful PNG for unchanged symbols and displayed values. Changed values redraw; rendering errors are not cached.

## First-pass measurements

Local Apple M4 Pro. These are component measurements, not a claim about every native click or whole-app CPU usage. Native process counts and machine load can vary. Browser results use synthetic but fully rendered 850-process / 1,000-connection / 1,800-history fixtures and the actual view code.

| Measurement | Before | After |
| --- | ---: | ---: |
| Go allocation per native process collection | 3,311,022 B | 183,735 B |
| Native process collection (one benchmark run) | 4.21 ms | 3.68 ms |
| Derived monitor work between process samples, native reads stubbed | 44.2 µs / 158,897 B | 0.111 µs / 8 B |
| History JSON, 1,800 retained samples / one new sample | 291,601 B | 163 B |
| 1,500 connection host sort, Bun fixture | 14.50 ms | 2.34 ms |
| Six dot charts over one hour of samples, Bun fixture | 1.20 ms | 0.39 ms |
| Two line charts, Bun fixture | 0.38 ms | 0.09 ms |
| Repeated identical native tray PNG | 441.5 µs / 4,968 B | 0.0075 µs / 0 B |
| Initial JavaScript | approximately 445 KiB | 210.66 kB (205.7 KiB) |

The 248.99 kB settings/menu-bar chunk is still loaded, but after the first screen; this reduces the startup critical path, not total script bytes. The second pass below adds table virtualization; no result guarantees 60 fps under every workload.

## Second pass: page switching and collection

- Processes and Connections retain their DOM and scroll position between visits. Their tables render the visible rows plus one viewport on either side, while search and sorting still operate on the complete snapshot. Row positions, selection details and keyboard scrolling remain available.
- Table markup is reused when its snapshot, filtering, sorting, language, selection and visible window are unchanged. Other pages retain their original mount lifecycle.
- Process and connection reads have independent in-flight requests. A slow connection collection no longer delays metric/history rendering or process refresh. Hiding the window during a history request now queues a catch-up render for return to the foreground.
- Process sampling retains only previous counters, rather than full process records, and allocates the next result list once. Connection parsing streams fields instead of allocating a complete split array.
- On macOS, lsof enumeration runs concurrently with nettop's two-sample measurement. Cancellation waits for the child collector to exit. The one-second rate measurement, unavailable states and metric definitions are unchanged.

| Second-pass measurement | Before | After |
| --- | ---: | ---: |
| Actual browser navigation: click to destination content plus next frame, mean of 10 warm switches | 230.82 ms | 33.33 ms |
| Same navigation test, slowest measured switch | 342.80 ms | 34.20 ms |
| Synchronous sidebar click handler, maximum | 0.20 ms | 0.30 ms |
| Process rows mounted at the end of that fixture | 850 | 66 |
| Warm table DOM insertion/layout component, mean | 84.17 ms | 4.85 ms |
| Unchanged process table markup, mean | 0.493 ms | 0.009 ms |
| Unchanged connection table markup, mean | 1.386 ms | <0.01 ms |
| Process bookkeeping for 850 records, native reads stubbed | 90.59 µs / 514,522 B / 865 allocations | 34.99 µs / 196,697 B / 5 allocations |
| Parsing 1,500 connections | 432.85 µs / 1,146,853 B | 295.72 µs / 851,941 B |
| Native connection collection, one local benchmark run | 1,078.73 ms | 1,024.83 ms |

Navigation measurements use the actual frontend with a synthetic bridge and 850 processes / 1,000 connections. They exclude native IPC and are not native end-to-end latency claims. The first two of twelve alternating switches are discarded. Native collection varies with current sockets and machine load; nettop still needs its second sample. Final initial JavaScript is 213.41 kB, plus the deferred 248.99 kB React chunk.

Second-pass evidence is under ignored `work/app-performance-2/`: `switch-bench.html?mode=before` and `?mode=after` measure actual navigation against the saved baseline; `second-bench.html` measures view markup and DOM layout; `app-regression.html` runs 31 integrated checks. Run each page's test button with the local Vite server. JSON outputs and benchmark logs record before/after results. Added Go benchmarks are `BenchmarkProcessSampleBookkeeping`, `BenchmarkParseConnections`, and macOS `BenchmarkNativeConnectionCollection`; `TestNativeConnectionCancellation` verifies cancellation, and the real localhost transfer test verifies native counters.

All 64 frontend tests, full Go tests, metrics race tests, TypeScript checking and final macOS arm64 packaging passed. Integrated browser checks cover bounded rows, complete-list sorting, bottom rows/details, searching after scrolling, keyboard focus, scroll extent, cached nodes, paused/hidden states, language changes, and metrics updating during a deliberately blocked connection read. Native final-package acceptance covered real process details and scrolling, preloaded connections followed by fresh data, search and bottom scrolling, Settings/Menu manager switching, and close/reopen with PID 23582 unchanged and samples advancing from 21:26:45 to 21:27:22. Tray readout and Windows/Linux remain outside verified acceptance.

## Verification and reproduction

- `bun run test:web`, `bun run typecheck`, `bun run test:go`, `bun run build`.
- `sh scripts/go.sh test -race ./internal/metrics ./internal/trayimage`.
- `sh scripts/go.sh test ./internal/metrics -run 'Test(PublishedMetrics|IncrementalHistory)' -v` reproduces readable snapshots during blocked collection and measures incremental history.
- `sh scripts/go.sh test ./internal/metrics -run '^$' -bench 'Benchmark(MonitorBetweenProcessSamples|NativeCollectors)' -benchmem`.
- `sh scripts/go.sh test ./internal/trayimage -run '^$' -bench BenchmarkRepeatedTrayImage -benchmem`.
- Local browser harnesses and raw logs are retained under ignored `work/app-performance/`. With `bun run dev:web --host 127.0.0.1`, open `/work/app-performance/render-bench.html` and `/work/app-performance/app-regression.html`, then run their check buttons. The first compares saved pre-change views against current views, checks exact DOM parity and 100 deterministic row permutations. The second exercises actual main.ts with a synthetic native bridge: coalesced events, incremental history, cached process lists, focus/scroll, pause/resume, hidden-view catch-up, connection entry/search, rapid navigation lazy React pages, and language changes overlapping a history request.

Native macOS arm64 acceptance covered Overview live data/charts, process selection/details, preloaded Connections followed by fresh collection, connection search, lazy Settings/Menu manager, and close/reopen retaining the same process with advancing samples. Actual system-tray readout remains unverified because system UI inspection timed out; Windows/Linux desktop acceptance was not performed. Final package evidence is recorded in STATUS.md. These changes are local; no commit, push or release is implied.
