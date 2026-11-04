# Performance work — 2026-10-29

Sampling intervals and metric definitions are unchanged. The frontend still consumes native Go samples; missing metrics and sleep/reset gaps remain explicit.

## First pass

- Native sampling uses a separate lock from published snapshots. Current metrics, processes and history remain readable while a collector is busy; settings no longer share the sampling lock.
- Process rankings and app aggregation are recomputed only when the process sample changes. Snapshots expose that sample timestamp so the frontend can avoid fetching an unchanged process list.
- macOS process conversion reads C records in place rather than copying each large record onto the Go heap. Returned strings and values remain owned by Go.
- Bootstrap reuses an available native snapshot. Regular history updates request only samples newer than the last accepted timestamp; retention and gaps are preserved. Hidden views skip history requests and catch up on visibility changes. Updates arriving during a request coalesce to the newest snapshot.
- Sidebar selection stays independent. Page renders are batched per animation frame. Live metric pages and inspection tables patch changed DOM nodes; process rows keep their identity across sorting and use a longest ordered subsequence to minimize moves. React forms retain their separate renderer.
- Chart timestamps are parsed once per immutable sample. Connection sorting reuses an Intl.Collator, empty queries avoid search-string construction, and connection row labels are translated once per render.
- React settings/menu-bar code is split out of the initial script. This first pass preloaded it during startup idle time; the runtime-memory follow-up below now loads forms on first navigation while retaining process/connection preloads.
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


## Launcher startup memory — 2026-11-03

Launcher now registers its shortcut and starts the file index without creating a hidden WebView. The first invocation creates the panel; the frontend focuses and queries on initial load because the first shown event may precede JavaScript loading. Later invocations reuse the same panel. After each full index build, `debug.FreeOSMemory()` collects scan temporaries and returns unused Go heap pages to the OS. Sampling intervals, file-search scope, index representation and result ranking are unchanged by this optimization.

Local Apple M4 Pro component measurements, using macOS `footprint` physical footprint (MiB), not RSS or a whole-app Activity Monitor comparison:

| Measurement | Before | After |
| --- | ---: | ---: |
| Real home index, 579,832 entries, immediate post-build footprint | 111 MiB | 67 MiB after memory release |
| Independent native Launcher probe, main process before first invocation | 29 MiB with eager hidden panel | 17 MiB with lazy panel |
| Native Launcher windows before first invocation | 1 | 0 |

The direct memory-release call took 6.14 ms, including garbage collection; the index still returned the real file-search results afterward. Its retained Go heap was 41.8 MB (decimal), so this does not eliminate the index's live storage or reduce its scan-time peak. The earlier staged experiment measured 116 → 73 MiB after a separate GC, consistent with the direct probe. Real-home search before the change had a 1.29 ms median and 5.72 ms maximum; indexing/ranking code was not changed.

The native probe loads the built Launcher frontend through MyGo and binds the real Go service. It verified initial DOM focus, a Safari query over real installed apps, close-to-hide, reopening with query preservation and the same window, and application Quit. Two lazy first invocations focused in approximately 0.29–0.65 s; the already-created panel focused in 0.058 s in one comparison. These are local observations, not a latency guarantee. Global-shortcut registration is exercised; invocation is called programmatically in the probe.

Reproduction and local evidence:

- `sh scripts/go.sh test . -run 'Test(LauncherCreatesWindowOnlyOnFirstUse|QuitWithHiddenLauncher)' -count=1`; the added startup regression failed before the lazy-window change and passes afterward.
- `sh scripts/go.sh run work/startup-memory/index_probe.go`; the ignored probe records heap stats and physical footprint before/after direct memory release, retaining the live index through the measurement.
- `sh scripts/go.sh test -overlay work/startup-memory/probe-overlay.json -c -o work/startup-memory/native-probe .`; run once with `ROCKSHINOPS_PROBE_EAGER=1` and once without it. The ignored overlay substitutes only the test entry point; it uses an isolated test preference path and shortcut and does not replace the installed app.
- Raw logs: `work/startup-memory/index-direct-release.log`, `native-eager.log`, `native-lazy.log`, `lazy-before.log`, `lazy-after.log`, `launcher-race.log`, `go-tests.log`, `web-tests.log`, `frontend-build.log`, `package-build.log`.

Go tests, focused launcher race tests, frontend tests, type checking, frontend build and macOS arm64 package build passed. The installed app/preferences were preserved. Whole-app before/after startup footprint, packaged tray text, main-window close/reopen and Windows/Linux desktop behavior remain unverified. The component savings must not be added to claim that a 300 MB application now uses a particular total.


## File-index storage follow-up — 2026-11-03

Each entry now occupies 16 rather than 24 bytes: a folded name is stored directly after its original spelling, with its UTF-8 byte length only when needed. A completed full scan then shares identical name bytes across entries. Paths, parent links, timestamps, spelling/pinyin data and ranking stay per entry; equal filenames in separate directories remain separate results. New filesystem updates still append normally, and the next full rebuild compacts again. The temporary name lookup is discarded after compaction.

Measurements on the same real home and configured scope (approximately 579,870 entries; the live filesystem changed by a few entries between runs):

| Measurement | Previous optimization | Compact entries and shared names |
| --- | ---: | ---: |
| Retained index Go heap, excluding pre-build baseline | 39.8 MB | 29.5 MB |
| Independent index-process physical footprint after returning idle pages | 68 MiB | 57 MiB |
| Immediate post-build process footprint, before returning idle pages | 111 MiB | 118 MiB |
| Full build, one observation | 3.107 s | 3.327 s |
| Search median / maximum across the same 17 queries | 1.312 / 6.255 ms | 1.254 / 6.110 ms |

The heap reduction is approximately 26%. The name-compaction step took 33.8 ms and holds the index lock during that step. Startup/rebuild transient memory was not reduced: the temporary deduplication map requires memory until the existing post-build collection releases it. Timing differences are single local observations; they establish no search speedup. These remain component measurements, not a before/after measurement of the installed desktop app.

Tests cover preserved original names and parent-folder queries when Unicode lowercase encoding is shorter or longer (`İ`, `K`, `Ⱥ`), duplicate filenames, arena growth, pinyin ranking and filesystem additions/removals. Go tests, index race tests, frontend tests, type checking, frontend build and macOS arm64 packaging passed. Local evidence is in ignored `work/index-memory/`: `home-before.log`, `home-packed.log`, `footprint-before.log`, `footprint-packed.log`, `index-race.log`, `go-tests.log`, `web-tests.log`, `package-build.log`. The existing real-home and memory probes above reproduce the final measurement; `timing-overlay.json` adds timing output without modifying production code.

A native WKWebView probe bound the real Launcher service and built the live home index (579,905 entries): Safari app search, the real 第一题复核成绩.xlsx file result, first-load focus and panel close/reopen with query retention passed (`work/index-memory/native-file-search.log`). The installed app/preferences were preserved; packaged tray text, main-window lifecycle and Windows/Linux desktop acceptance remain open.


## Runtime memory and rolling history — 2026-11-03

History eviction now reuses the existing slice storage instead of allocating and copying the whole retained hour on every sample. Public History/HistorySince reads still return copies. A regression covers repeated eviction, the inclusive retention boundary, previously returned snapshots and long gaps. A full-history benchmark with 850 synthetic processes measured 298,643–298,661 B per sampling tick before and 118,417–118,418 B afterward (three 1,000-tick runs each), approximately 60% less allocation. This is reduced allocation churn after the history window fills, not a 60% reduction in resident application memory. Timing was affected by concurrent machine activity; no CPU-speedup claim is made.

The Settings/Menu Bar/Launcher/Cleanup React bundle is now loaded on first navigation. Process and connection data still preload as before, preserving cached table entry. An isolated native instance uses the real main window, samplers and configured home index, with its own preference directory and shortcut; login-item initialization and updater polling are omitted. It starts on Overview, samples memory 20 seconds after the main window is created, then enters Settings and checks main-window close/reopen. It preserves the installed app and user preferences.

| Native probe observation | Startup React preload | Forms loaded on demand |
| --- | ---: | ---: |
| Main page WebKit physical footprint before visiting forms | 74–75 MiB | 70–71 MiB |
| First Settings render, click through next animation frame | 29 ms | 52 ms |
| Go heap after collection, approximately 580k indexed entries | 33.68 MB | 33.68–33.69 MB |

The WebKit saving lasts until a form page loads the bundle. The main process varied around 89–92 MiB; its footprint plus the main WebContent process was approximately 165–166 MiB before and 159–162 MiB afterward. These totals exclude GPU/network processes and do not represent the user's installed instance with its existing settings. Disabling all inspection preloads saved more WebKit memory in a separate probe, but was not adopted because first connection entry would need to wait for collection. No sampling interval, history duration or file-search scope changed.

Evidence and reproduction:

- `sh scripts/go.sh test ./internal/metrics -run '^Test(RollingHistory|IncrementalHistory)' -bench '^BenchmarkMonitorFullHistory$' -benchtime=1000x -count=3 -benchmem`.
- Ignored `work/runtime-memory/history-before.log` and `history-after.log` hold the three-run allocation comparison; `metrics-race.log` and `go-tests.log` record passing race/full Go tests.
- `baseline.log`, `preload-repeat.log`, `no-react-preload.log` and `final.log` record native memory and interaction checks. `overlay.json` and `profile_test.go.txt` define the isolated test entry point; the WebKit PID is read through a private selector only in this diagnostic harness, never production code. `no-preload.log` is the unshipped broader experiment.
- `web-tests.log` and `package-build.log`: frontend tests, TypeScript check, frontend build and macOS arm64 app/DMG packaging passed. The final production frontend is used by the `final` native probe.

Native probe first Settings rendering and main-window close/reopen passed. Installed-app replacement, packaged tray text and Windows/Linux desktop behavior remain unverified. No installation, commit, push or release was performed.


## Name-compaction allocation follow-up — 2026-11-03

The lookup now converts the name bytes directly in the map lookup, so Go can avoid allocating a string for an already-seen name. Only new names acquire an owned string before in-place packing changes the backing bytes. The algorithm, stored index and search results are unchanged.

`BenchmarkNameCompaction` uses 200,000 entries with 80,000 distinct names, including uppercase names with folded storage. Three runs of ten iterations, excluding fixture cloning:

| Per compaction | Before | After |
| --- | ---: | ---: |
| Allocated bytes | 11,655,864–11,655,876 | 9,735,865–9,736,928 |
| Allocations | 200,531 | 80,531–80,532 |
| Elapsed time | 6.20–6.23 ms | 5.31–5.49 ms |

This reduces allocation bytes by approximately 16% and allocation count by approximately 60% in this fixture. It does not reduce the retained index size. Reproduce with `sh scripts/go.sh test ./internal/fileindex -run '^$' -bench '^BenchmarkNameCompaction$' -benchtime=10x -count=3`.

A sorting alternative was rejected: its fixture allocations were lower but compaction was about three times slower, and three real-home process runs had overlapping footprint peaks (116–128 MiB for the original hash lookup, 111–125 MiB for sorting). Phase probes of the adopted lookup still varied: the pre-compaction scan already reached 97–113 MiB peak, and the full build reached 108–120 MiB. These are separate local observations with garbage-collection timing and a live filesystem, not evidence of a peak-memory improvement. No GC tuning or sampling/search-scope change was adopted.

Evidence: ignored `work/index-peaks/compaction-before.log`, `compaction-final.log`, `real-{before,sorted}-{1,2,3}.log` and `phase-{before,lookup}.log`. Go and frontend tests, index race tests, TypeScript check, frontend build and macOS arm64 packaging passed (`go-tests.log`, `web-tests.log`, `index-race.log`, `package-build.log`).

The first native Launcher probe timed out waiting for the Safari result after input focus (`native-file-search.log`). An instrumented rerun confirmed backend results, rendered Safari and the real 第一题复核成绩.xlsx result from 580,052 indexed entries, and passed panel close/reopen reuse (`native-diagnostic.log`). The timeout did not reproduce and its cause remains undetermined; this is a passing rerun, not evidence that the intermittent failure is resolved. Packaged tray text, installed-app lifecycle and Windows/Linux desktop acceptance remain open. The installed application and user preferences were preserved; no commit, push or release was performed.


## Scan path allocation — 2026-11-03

Full scans now construct the relative child path only for directories, where it is used by skip rules and the traversal queue. Files already store their parent index and filename; their temporary relative path was unused. Traversal scope, ordering, timestamps, search scoring and watch updates are unchanged.

Three alternating independent-process runs on the configured real home, approximately 580,064–580,069 entries:

| Full scan measurement | Before | After |
| --- | ---: | ---: |
| Cumulative allocated bytes | 784.8–786.5 MB | 729.0–729.5 MB |
| Allocation count | 5,057,224–5,057,390 | 4,569,857–4,570,234 |
| Elapsed time | 3.31–3.48 s | 3.42–3.56 s |
| Peak process footprint | 116–123 MiB | 110–122 MiB |
| Post-collection process Go heap, including baseline | 31.6 MB | 31.6 MB |

Each scan avoided approximately 56 MB of cumulative allocation (~7%) and 487,000 allocations (~10%). This is less allocation churn, not a 56 MB decrease in simultaneous or resident memory. Peak ranges overlap and elapsed time did not improve in these runs; neither a peak reduction nor a speedup is claimed.

A repeatable fixture of 100 folders with 50 files each measured 4.057–4.073 MB/op before and 3.898–3.910 MB/op afterward, with approximately 5,000 fewer allocations per build. Reproduce with `sh scripts/go.sh test ./internal/fileindex -run '^$' -bench '^BenchmarkBuild$' -benchtime=10x -count=3`. Fixture setup is excluded from measurement.

Local evidence: ignored `work/index-scan/build-{before,after}.log` and `real-{before,after}-{1,2,3}.log`; `probe-overlay.json` adds cumulative-allocation measurements to the existing real-home probe. Index race tests, full Go tests and frontend tests passed (`index-race.log`, `go-tests.log`, `web-tests.log`).

Type checking, frontend build and macOS arm64 app/DMG packaging passed (`package-build.log`). The native Launcher probe passed first focus, Safari search, real 第一题复核成绩.xlsx search from 580,094 indexed entries, and panel close/reopen reuse (`native.log`). Packaged tray text, installed main-window lifecycle and Windows/Linux desktop acceptance remain unverified. No installation, commit, push or release occurred.


## Temporary scan records — 2026-11-03

Temporary directory entries now carry the same unsigned Unix seconds already used by the retained index, converting once at the filesystem boundary. On arm64, the record shrinks from 48 to 24 bytes. Timestamp precision, pre-epoch clamping, ranking and directory-change detection remain unchanged. The existing directory-reading API is retained.

Three alternating real-home runs (approximately 580,121–580,126 entries) measured cumulative allocation of 728.84–729.04 MB before and 714.45–714.89 MB afterward: approximately 14.4 MB (~2%) less temporary allocation per full scan. Build times were 3.48–3.60 s before and 3.41–3.49 s afterward; these local runs do not establish a speedup. Collected process Go heap remained 31.6 MB including baseline. Peak physical footprint was 105–128 MiB before and 105–120 MiB afterward; overlapping ranges do not establish a peak-memory improvement.

Two directory-reading alternatives were not adopted. Reading FileInfo directly lowered fixture allocations but changes handling of individual stat failures. Reading names then performing Lstat preserved skip handling and lowered real-home allocation to approximately 682 MB, but took 3.79–3.83 s versus 3.42–3.52 s for the baseline, without lowering retained memory. Those experiments remain outside production code.

Evidence is in ignored `work/index-list/final-{before,compact}-{1,2,3}.log`; the existing real-home probe is instrumented through `probe-before-overlay.json` to report cumulative allocations. Full Go/frontend tests and index race tests passed (`go-tests.log`, `web-tests.log`, `race.log`), including timestamp precision, ranking, Unicode/pinyin and filesystem refresh regressions.

Type checking, frontend build and macOS arm64 packaging passed (`work/index-list/package-build.log`). The native Launcher probe passed initial input focus, Safari search, the real 第一题复核成绩.xlsx result from 580,151 entries, and panel close/reopen reuse (`native.log`). Installed-app replacement, packaged tray text/main-window lifecycle and Windows/Linux desktop acceptance remain open. No commit, push or release occurred.
