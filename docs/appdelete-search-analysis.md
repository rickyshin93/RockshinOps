# AppDelete search investigation

Local target: AppDelete 4.3.3, x86_64, SHA-256 `a80d5dff43c1cb5b5224af4da36c966239a1ac1912b64435bd22be0ea3c405ca`. REA 6.7.0, Ghidra 12.1.4 and Apple's LLDB. This records reconstructed behavior of this binary, not original source or proof of file ownership.

## Window identity repair

The isolated REA copy is `/Users/crh/.local/share/rea-lab-tools/rea-runtime-identity`; its wrapper is `/Users/crh/.local/share/rea-lab-tools/rea-runtime`. The pinned package in `rea-lab/node_modules` is unchanged.

Two independent faults were reproduced:

1. LLDB-launched AppDelete does not expose an AppKit launch date. The helper uses the kernel's PID/start-time pair and validates the executable's canonical path and digest. It rechecks identity before an action and after observation.
2. Core Graphics' `.optionIncludingWindow` returned no entry for a debugger-owned window that was present in `.optionAll`. The helper now enumerates window metadata and selects the exact requested ID, requiring one result, the expected PID, layer zero and valid bounds. AX ownership/bounds checks remain. Screen capture still targets that one window through `SCContentFilter(desktopIndependentWindow:)`; desktop-wide pixels are not captured.

The Core Graphics SDK contract is in `CGWindow.h:132–157` under the local macOS SDK. The repaired helper accepted the owned window and rejected an unknown window and another process's window as `window-mismatch`, a stale start time as `process-replaced`, and a wrong digest as `target-mismatch`. Evidence: `/private/tmp/localops-rea-window-validation-results.json`. The isolated source change is preserved in [rea-native-ui-identity.patch](rea-native-ui-identity.patch).

Legacy protection dialogs expose a Core Graphics window but do not appear in the target's AX window/child/focused-window lists. The helper correctly refuses AX actions on those dialogs as `ambiguous-window`. Their screenshots were captured using the exact dialog ID, owning PID, start time, executable and digest, with accessibility disabled. No geometry-only click or foreground/global-event fallback was added.

All actions targeted debugger-owned AppDelete processes. Temporary fixture bundles were created under `/private/tmp/localops-rea-branch-fixtures`, without installation or execution. Fixtures were opened using an Open Documents Apple event addressed to the exact validated PID. Search results were inspected; no Delete, Archive, Log Only or cleanup action was used. Other foreground applications and LocalOps Dev were not controlled.

## Selected-app algorithm

REA native dispatch maps `ADController.searchFiles:` to `0x10001eec1` (`___lldb_unnamed_symbol407`), `appAdjustments:` to `0x100025d0b` (421), and `appAdjustments2:` to `0x100025f0b` (422). Selector-only breakpoints do not resolve these stripped functions; function breakpoints do.

The selected-app path performs these steps:

1. Check ignored paths and the `ProtectRunningApps` / `ProtectDefaultApps` preferences. Running-app detection uses `JAProcessInfo`; the default-app check tests whether the bundle identifier contains `com.apple`. All three blocking paths now have runtime evidence. Per-process launch arguments supplied the test preferences through `NSArgumentDomain`; persistent user preferences were not edited.
2. Read `NSBundle.bundleIdentifier` and `CFBundleName`. Missing identifiers become empty strings. A missing or space-only declared name falls back to the file manager's display name without its extension. A declared name equal to `Library`, ignoring case, also uses the display name. Trim the chosen name and derive a second variant with spaces removed. The command builder escapes double quotes in the selected path; LocalOps uses Go string matching rather than a shell command.
3. Select one of **three** command templates, decoded from the binary's `__cfstring` section and cross-checked against runtime child commands:

| Template address | Condition | Difference |
| --- | --- | --- |
| `0x10003fbe0` | No bundle identifier | Name and compact-name searches; duplicate exact-name terms replace identifier terms; no recursive identifier search in `/private/var`. An empty ID does not become a match-all wildcard. |
| `0x10003fc00` | Identifier present; missing/space-only declared name | Uses display name; Library search additionally excludes `.*(null)(null).tiff`. |
| `0x10003fc20` | Identifier and usable declared name | Normal identifier/name search. The special `Library` name substitution also reaches this template. |

4. Run `/bin/sh -c` with the following search areas. The app bundle itself is emitted first. Searches prune matched directories and some nonmatching directories whose names contain a dot. Errors are redirected to `/dev/null`.

| Area | Match | AppDelete depth limit |
| --- | --- | --- |
| `~/Library`, `/Library`, plus explicit SDMHelpData locations below user Preferences/helpd caches | Name, compact name and identifier | 4 |
| Desktop, `/Applications`, `/Applications/Utilities`, Finder workflows, Growl tickets, generated help | Name, compact name and identifier | 1 |
| `/private/var` and ApplicationRecentDocuments | Identifier, only when present | No explicit limit |

For either name variant `N`, the case-insensitive patterns are `N`, `N.*`, `*.N`, `*.N.*`, and `N_*`. Identifier matching is `*identifier*`. This is filename matching, not a content search or a complete disk scan.

5. Pipe NUL-delimited matches through `xargs -0 du -hsc`, then `awk` to split size/path output. Parse UTF-8 newline pairs, apply the two special adjustments below, remove the trailing total/empty output, deduplicate paths and arrange the display arrays. The intermediate newline representation is not a reliable design for arbitrary filenames; LocalOps retains native Go path values.
6. When the identifier contains `com.theescapers.flux`, call `appAdjustments:` and remove entries containing `herf.Flux` or `justgetflux.flux`, together with their preceding sizes. When it contains `org.herf.Flux`, call `appAdjustments2:` and remove entries whose final path component is exactly `Flux`, together with their sizes. Both calls were observed at runtime. These are hard-coded disambiguation rules, not a general ownership algorithm.

## Runtime coverage

| Branch | Input | Evidence / result |
| --- | --- | --- |
| Declared name + identifier | Real Agent HUD | Search and task-launch hits; actual command captured; 8 UI candidates. Earlier trace `ev_4154e2c382ed170fc939188a30fed569f2f81f3fbcc8364db7196655e6c27be1`. |
| f.lux adjustment | Real `/Applications/Flux.app`, `org.herf.Flux` | Trace `ev_f9be357e26b10315bad24936fc7e88c8673bc9bbdd48ffb4b2de7831b9926d3c` hits 407 and 422; REA UI scenario `ev_730e4807ca3b525d670a2ded2573911a249eb7b8b8e7e148824be386f4d8ba1a` completes; 7 candidates. |
| Missing declared name | Real 震旦安心印, `com.Aurora.Client` | Executed command contains the null-TIFF exclusion; later AX observation shows the bundle, user preference and system cache (3 candidates). REA scenario `ev_393324cd468de242755bdd0aade531ba0e97d16e9b73e88125e65ecfe55efca0`; final scenario capture preceded the asynchronous result replacement. |
| Missing identifier | Temporary `No Identifier.app`, declared `REA No Identifier` | Actual command uses name-only template, without `/private/var`; one candidate (the fixture bundle). |
| Declared `Library` name | Temporary `Library Name.app` | Actual command searches `Library Name` / `LibraryName`, not the generic `Library`; one candidate. |
| Other Flux adjustment | Temporary `Flux Web Designer.app`, `com.theescapers.flux` | Trace hits 407 and 421; 3 results include `Application Support/Flux` **and the different installed `/Applications/Flux.app`**. LocalOps skips other application bundles. |
| Running-app protection | Real AppDelete itself; `ProtectRunningApps=YES` | Trace `ev_94d5f61d0489141a10d9bdce87f6dd2e0a7ac1f57cd94d5f64707cac70b51988`, event 11 enters search without launching a residual-search task; exact-window screenshot shows “发现被保护项目”. |
| Default-app protection | Uninstalled `Apple Guard.app`, ID `com.apple.localops.rea.guard`; running protection disabled | Trace `ev_e7371684c3056e7e7a6a51d64443969b72e7401203797e09637cf0b30c67884b`; same protection dialog captured. This validates identifier-based classification, not a check of Apple's signing identity. |
| Ignored-path protection | Real Agent HUD passed in `IgnoredApps`; both other protections disabled | Trace `ev_bfbbe2abbf6b491940f88909ffeb2f723fef16d5c372d4eb80d10c29220de244`; same protection dialog captured. The argument array was first checked in an independent Foundation process. |

The missing-name, missing-ID, `Library` and designer-Flux cases share trace `ev_f81796b42d50b8fd4a4afb10e61f1d83f896b1274f82e4f48ce32071e9e173fb`. Its 18 events include the inventory stage and four selected-app searches; no breakpoint was unresolved and no event/resource limit was reached. A duration-bounded trace still reports partial program coverage: this is not a claim that every AppDelete branch was executed.

For the real f.lux search, task launch → `appAdjustments2:` was 4.362 seconds under LLDB. The designer fixture's task launch → `appAdjustments:` was 4.915 seconds. These intervals include debugger overhead and AppDelete's smaller search scope; they are not equivalent to untraced LocalOps backend benchmarks.

Raw traces and process commands: `/private/tmp/localops-rea-branches-trace.json`, `/private/tmp/localops-rea-branches-subprocesses.jsonl`, `/private/tmp/localops-rea-extra-branches-trace.json`, `/private/tmp/localops-rea-extra-subprocesses.jsonl`. Fixture observations: `/private/tmp/localops-rea-{noid,libraryname,escapers}-ui.json`. A compact durable index is [appdelete-runtime-evidence.json](appdelete-runtime-evidence.json).

## Other modes in the same function

`firstPanel` dispatches inventory commands separately from selected-app search. Its application mode enumerates `/Applications`, user Applications and Utilities, pruning bundles; other modes enumerate widgets, preference panes, Internet plug-ins, screen savers and Mail bundles, or combine those inventories. Size-sorted variants add `du`; name/creation date/last-used/kind preferences change display sorting. `findAll:` maps toolbar tags 80/81/82 through the checked sorting-menu items (initialized from stored preferences); tag 100 selects `ORPHANS`. Compiled English/Chinese Interface Builder resources confirm those tags and target/action connections (REA `ev_bcb6cb70bbd7a67940753d5165b7402f6f76c1741c3bfe285fd45a229c4229c2`). The resource-only analysis slice contains unchanged copies of four nib files and the same executable; it was not launched.

`fakeDeletePanel` and `fakeDeletePanelTwo` are informational notices for Genius and Orphans, respectively, with OK/Cancel and a suppression checkbox. They are not license or permission checks. Their per-process `AlertSuppress` / `AlertSuppressTwo` defaults allow a test process to skip the notices without editing persistent preferences. Static evidence: `ev_f2f9a947ade0f73e90aa55b5b8e40baf4bded5b0f148ef7df3d9ebbd0fa559a1` and `ev_e76ba094b23392c39bfd6d8fb6d4ba2cfee8c7ddb1a205dee62c9d978a66ce2a`.

Genius applies `geniusFixNew:` (`0x10001200c`): read each inventory path's Spotlight `kMDItemLastUsedDate` through `MDItemCreate` / `MDItemCopyAttribute`, fall back to file modification time, and retain dates at or before now minus 15,552,000 seconds (180 days). Unreadable file attributes produce an `unknown` marker, which is excluded. The size-sort variant preserves the preceding size with each retained path. Evidence: `ev_72c506d3306add5dab128d438c433d17228ec4f76a59b57604c7d12728b3e470`. This identifies old inventory items; it does not establish that their data is abandoned.

The orphan heuristic is narrower than a full leftover scan: enumerate `~/Library/Preferences/com.*.plist` at depth one, excluding `com.apple*`; derive the filename's domain, require no application registered for that identifier, exactly three dot-separated components, and membership in persistent preference domains. Exclude `com.microsoft.office`, `com.binaryage.totalfinder`, `com.osx.iffmpeg` and `com.ragingmenace.MenuMeters`. Read Spotlight last-used dates with `mdls`, falling back to file modification time, then sort. The final date filter also uses 180 days; it parses the displayed medium-style date before comparison. These conditions are static reconstruction; they do not prove ownership or absence of an installed application outside Launch Services' records.

Earlier auxiliary runs (`ev_94d5f61d0489141a10d9bdce87f6dd2e0a7ac1f57cd94d5f64707cac70b51988`, `ev_b137965417b710247da69d97deee0b8f78f227c6ffdd3ca606735433923485ef`) had incomplete observations and some AX timeouts. The final run uses the two transient notice-suppression defaults and restricts task breakpoints to `NSConcreteTask`. Trace `ev_cc1f120b9748d887b2ec0951af1328aab588c259ddad5e713022f1a9ebe1775a` records all three searches, 19 events, no unresolved breakpoint/event/resource limit, and normal termination at the observation deadline. Every toolbar press succeeded. Exact-window captures now establish these outcomes:

| Mode | Observed result | Independent check |
| --- | --- | --- |
| Others | One row named `localops` with an empty path; earlier runs launched from `rea-lab` showed that name instead | The decoded name-inventory command returned no paths. This is a working-directory-dependent empty-result anomaly, not an established add-on or residual. Its precise formatting cause is not claimed. |
| Genius | One path, `/Applications/CleanMyMac_5.app` | The decoded combined inventory returned 63 paths. A separate Foundation/CoreServices reader applied the same metadata/fallback/180-day rules and selected exactly that path, using its modification date. |
| Orphans | Exact owned dialog says “没有找到碎片！” | The legacy preference-domain and Launch Services checks left four domains; independent `mdls`/modification-date checks selected none older than 180 days. The trace contains the preference-list task followed by four metadata tasks. |

A second trace, `ev_e28ef54b7b858e42741c029cc949ebf83b777a1a3115df216aa25b14de24540d`, covers positive Orphans and add-on results. Its child-only `HOME` / `CFFIXED_USER_HOME` point to `/private/tmp/localops-rea-isolated-home-fixture`; Foundation's home path was checked before launch. This changes the searched file tree, not the system's preference-domain registry. Existing domain names were read, and ten synthetic plist files were written only under that temporary home. Two passed the initial orphan predicates; only the 400-day-old `com.gwork.picker-acceptance.plist` passed the date filter. AppDelete displayed that one file, unselected. An independent reader selected the same canonical path. Recent, registered-app, excluded, extra-component and nonpersistent-domain examples were absent.

Orphans has a second search stage: `initiateOrphansSearch:` (`0x100005d75`) copies the filtered paths into `pasteboardFiles`, clears `firstPanel` and dispatches `searchFiles:` again. The positive trace observes this second entry and its task launch (events 9–11). A plain plist has no `NSBundle` metadata, so the normal missing-ID/name template can expand its basename into associated candidates; the UI is not merely a direct rendering of the initial preference list. The final Orphans path clears the selection array before display. Static handoff evidence: `ev_2c673df6477ae5e734e46048a1da8be3c324b49336b64ab6563dc571e7f3dec1`.

The same owned process then found all five inert temporary add-on directories: a widget, preference pane, Internet plug-in, screen saver and Mail bundle. AX exposed all five paths, even though only the first row fit the compact window. Neither fixture bundles nor plug-ins were installed or executed. Evidence: `/private/tmp/localops-rea-aux-isolated-{trace,summary}.json`, `/private/tmp/localops-rea-aux-isolated-{orphans,addons}-screen.json`, `/private/tmp/localops-rea-isolated-home-plan.json` and `/private/tmp/localops-rea-isolated-orphan-reference-result.json`. The trace had 15 events, no unresolved breakpoint or event/resource limit, and terminated its owned process at the observation deadline.

The Orphans result dialog is another legacy alert without an AX window. It was captured by its exact CG window ID with accessibility disabled; no dialog click was attempted. Evidence files: `/private/tmp/localops-rea-aux-narrow-{trace,summary}.json`, mode-specific state/screenshot files, and `/private/tmp/localops-rea-aux-narrow-orphan-dialog.{json,png}`. Independent references are `/private/tmp/localops-rea-{genius,orphan}-reference.swift` and their result JSON files; inventory commands and outputs are in `/private/tmp/localops-rea-{others,genius}-reference-command.json`. Inventory commands returned status 1 because some configured roots are absent; AppDelete suppresses their stderr. Available paths were checked, not presented as a successful full-root enumeration.

One preceding launch stopped before producing search events after repeated `EXC_BAD_INSTRUCTION` stops. Its observation is retained as `/private/tmp/localops-rea-aux-verified-trace.json`; it is not coverage evidence. The successful narrower trace does not establish why that launch failed. AppDelete's archive/delete/login behavior is outside this search reconstruction and was not invoked.

Static evidence: dispatch `ev_2ea520a5cd35c2433ad4727a6385777f5fd4ab35ddec85d225442dfd7241837d`; search function `ev_60d41dee4e6226a9a03b7765b23fca96e88179c22af598f5e0a7903f3d94ea2d`; adjustments `ev_df14ae0d8f8619d07b9944fa57df34d33d1a14f261cb80dedbd08cc8863dbca8` and `ev_2355f24ae0d741cefc9c444a7fe15e204b786d59160825fe89226f53a6589074`. Detailed local pseudocode and decoded constants: `/private/tmp/localops-appdelete-search-pseudocode.c` and `/private/tmp/localops-appdelete-cfstrings.json`.

## Sorting branches and special filenames

REA decompilation of `findAllMenu:` (`0x100016883`), launch preference initialization, `whatKind:` (`0x10000dbdb`) and the size comparator (`0x100024ee9`) resolves the five sort choices. The per-mode preferences use 1=Name, 2=Size, 3=Creation, 4=Last Used, 5=Kind; menu tags route separately to Apps, Others and Genius. The menu order differs from the preference numbering.

| Sort | Reconstructed behavior |
| --- | --- |
| Name | Preserve the native `find` enumeration order; no alphabetical sort is applied in this branch. |
| Size | Descending comparator over displayed sizes, parsed as floats with binary KB/MB/GB multipliers. Rounded size ties are not exact-byte ordering. |
| Kind | Ascending kind descriptions from `LSCopyKindStringForURL`. App descriptions add the legacy `lipo -info` classification: PowerPC, Intel or Universal for PPC plus Intel. ARM-only output has no recognized architecture label. |
| Creation | Ascending `NSFileCreationDate`, formatted for display afterward. Equal-date order is not specified. |
| Last Used | Ascending `NSDate` parsed from `mdls -name kMDItemLastUsedDate -raw` output, falling back to `NSFileModificationDate`. Spotlight dates therefore have whole-second precision; modification-date fallbacks retain their precision. Equal-date order is not specified. |

Apps, Others and Genius each completed all five runtime sort choices. All five choices preserved each mode's candidate path set: 62 apps for Apps, five inert add-on fixtures for Others, and those fixtures plus CleanMyMac for Genius. Displayed-size, kind and date ordering agree with the reconstructed rules. A separate Foundation/Spotlight snapshot provides the date reference, including modification-date fallbacks; its Spotlight dates must be reduced to `mdls` whole-second precision. Comparing unrounded `MDItem` dates initially produced a false submillisecond ordering discrepancy. A later `mdls` probe also observed changed dates for running apps, so it is not treated as the earlier snapshot. The equal old fixture dates do not establish tie stability.

An early harness incorrectly treated stable empty AX rows as loaded results; those observations are excluded. Completion now requires populated rows, an untruncated snapshot and the ready status. A bounded bulk-child probe was used to inspect the populated Apps cells without changing identity checks. It remains a temporary diagnostic, rather than a change to the installed REA child-enumeration implementation. The isolated helper reports `AXEnabled` and rejects actions on explicitly disabled controls.

Nine inert bundles and matching/nonmatching directory fixtures test filename handling without uninstalling or executing the bundles. A function-breakpoint-only REA trace completed all nine searches. LocalOps scanned the same filename cases in separate temporary roots; comparison uses fixture basenames, not an assertion that the two products searched identical root paths.

| Filename case | AppDelete observed result | LocalOps scanner |
| --- | --- | --- |
| Spaces, apostrophe, tab, leading dash | All six intended related entries found in each case | All intended entries found; negative fixtures excluded |
| Double quote | Five name matches missed; identifier match retained | All six intended entries found |
| Square brackets | Five name matches missed; identifier match retained | All six intended entries found |
| Literal `*` and `?` | Intended entries plus one unrelated glob match | All six intended entries found; glob-like negative excluded |
| NFC/NFD accented name | Five NFC name matches missed; NFD and identifier entries retained | All seven intended entries found after NFC correction |
| Embedded newline | Displayed path/size pairs were split and corrupted; none of the six intended related paths remained intact | All six intended paths preserved and found |

These cases establish concrete filename failures in AppDelete's command/output pipeline; they are not reasons to reproduce those failures in LocalOps. Literal matching already avoids shell/glob expansion and preserves newline paths. The new correction addresses the reproduced Unicode miss. An earlier broad-breakpoint run with the original empty fixtures did not yield a usable result window, and another mixed trace stopped on repeated `EXC_BAD_ACCESS`. Their causes are not established; neither is counted as successful coverage. Raw observations, hashes and exclusions are retained in the evidence index.

## LocalOps behavior and validation

### Inaccessible locations shared by every app

The reported “370 items could not be read” was the count of failed directory reads across the common search roots, not a count of residuals belonging to the selected app. A fresh AlDente scan found 13 candidates and 371 inaccessible locations; every failure was `EPERM` or `EACCES`. Counts can vary as the filesystem changes. The same roots explain why the warning appeared for unrelated apps. AppDelete redirects its command errors to `/dev/null`; its absence of a warning does not establish better access.

LocalOps now returns each failed path, a permission/change/read classification, and the actual OS error. Optional roots that do not exist are not reported as failed reads. The interface folds these details under **Scan coverage is limited**, explains the distinction from candidate counts, and keeps reviewed candidates available for removal. No search scope, depth, duration or result limit was introduced. The permission regression uses a real unreadable directory and verifies that an accessible candidate is retained.

The UI explains [macOS Full Disk Access](https://support.apple.com/en-au/guide/mac-help/mchl211c911f/mac) and restarting before rescanning. No system permissions were changed. Full Disk Access does not promise access to every protected system directory; the OS errors remain visible.

- Search roots are user Library, `/Library`, `/Applications`, `/private/var` and the user's home, plus the resolved temporary directory if outside these roots. Home traversal also includes user Applications. It skips the Library subtree already searched separately, the user's trash, symbolic links and other app bundles. An `.app` suffix alone does not identify a bundle: a `Contents` directory is required, so cache identifiers ending in `.app` remain searchable in quick and complete passes.
- Both identifier and filename-boundary name matching operate at **every depth throughout those roots**. There is no time, entry, result or name-depth cap. A matching directory represents its entire subtree, so its children are not listed separately. Permission failures remain visible.
- Four workers per root, a bounded scheduling queue and local fallback avoid a queue becoming a search limit. Matching is literal Go string comparison: metacharacters in a name are not shell/glob expressions. Results stay unselected until reviewed.
- Name comparisons now use canonical Unicode NFC normalization after lowercasing. The existing pinned `golang.org/x/text` dependency supplies normalization; returned paths retain their original filesystem spelling. A real decomposed-name directory reproduced the miss before this correction. Compatibility characters and punctuation are not broadened into equivalent names.
- Applications without a bundle identifier can now be searched by name. The empty identifier is explicitly excluded from substring matching. Malformed nonempty identifiers remain errors; `Library` remains a display-name fallback.
- Name-only results are labeled “Name match only; verify ownership.” Unlimited name search intentionally also discovers similarly named personal folders, models and source packages. They are candidates, not established residuals. AppDelete's app-specific exclusions are documented rather than copied as automatic ownership assertions.
- Regression tests reproduce the missing-ID failures before fixing them, cover name-only files deep in Library and personal folders, filename boundaries, cancellation, wide/deep trees, symlinks, and the system cache's `C` sibling. Tests do not remove real application data.

### Performance evidence

CPU sampling over three real apps attributed 78.8% of sampled CPU time to `os.Open`, versus 0.3% to matching. The complete searches in that profiling run took 10.823 / 11.771 / 11.708 seconds; quick results took 35 / 58 / 65 ms. Log/profile: `/private/tmp/localops-unlimited-profile.log`, `/private/tmp/localops-unlimited-cpu.pprof` and `/private/tmp/localops-unlimited-profile.test`.

Measured alternatives did not improve this workload: eight workers took 10.599 / 12.484 / 10.692 seconds, root-relative `openat` took 12.065 / 11.595 / 11.830, and queued parent-descriptor traversal took 12.162 / 11.912 / 12.183. Native `find` was slower still. These experiments were not merged. The Go experiments returned candidate counts 15 / 8 / 17 for LocalOps / Agent HUD / f.lux; changing temporary test-binary paths prevent claiming identical path sets for those trials. Native-find prototypes also needed filtering of other application bundles and were not equivalent production implementations.

The final comparison uses the current unlimited name rules and all bundle/cache fixes. Two fixed binaries were compiled before timing; only workers per root differ (one versus four). Run order was serial / parallel / parallel / serial. Every candidate path and reason was identical across all four runs; every run reported 370 unreadable items. This is an existing concurrency improvement under the expanded search scope, not a claim that expanding scope itself makes searching faster.

| Real app | Serial complete, seconds (two runs) | Parallel complete, seconds (two runs) | Quick results, ms across runs | Candidates |
| --- | --- | --- | --- | --- |
| LocalOps | 29.326 / 30.280 | 11.139 / 12.089 | 46–62 | 14 |
| Agent HUD | 32.706 / 30.703 | 11.765 / 11.674 | 50–113 | 8 |
| f.lux | 33.374 / 41.647 | 11.569 / 10.889 | 53–78 | 17 |

The machine also ran development/analysis tools; these are wall-clock measurements on the local filesystem, not isolated lab throughput. Reproduction: the commands, fixed source snapshots and per-run logs are recorded in `/private/tmp/localops-unlimited-final-benchmark.py` and `/private/tmp/localops-unlimited-final-results.json`. Compact results are preserved in the durable evidence index.

A second fixed-binary ABBA comparison isolates the NFC correction, with four workers and structured coverage reporting in both binaries. All candidate paths and reasons were identical in all four runs, and all runs reported 371 permission failures. This correctness fix is not presented as an additional speedup:

| Real app | Before NFC, seconds | With NFC, seconds | Quick results with NFC, ms |
| --- | --- | --- | --- |
| LocalOps | 11.367 / 11.480 | 11.651 / 11.362 | 52 / 49 |
| Agent HUD | 11.911 / 11.722 | 11.490 / 11.331 | 55 / 58 |
| f.lux | 11.743 / 11.406 | 13.455 / 11.765 | 73 / 54 |

Reproduction: `/private/tmp/localops-scan-nfc-benchmark.py`; raw results: `/private/tmp/localops-scan-nfc-results.json`. The final real-app path sets still include all 27 previously captured AppDelete paths. No fresh AppDelete search timings are implied by this comparison.

### Real-app result comparison

| Real app | AppDelete displayed paths | LocalOps candidates | AppDelete paths missing in LocalOps |
| --- | --- | --- | --- |
| LocalOps | 12 | 14 | 0 |
| Agent HUD | 8 | 8 | 0 |
| f.lux | 7 | 17 | 0 |

This compares captured path sets, not ground-truth ownership. LocalOps' extra matches include its source checkout and downloaded installer, plus unrelated Flux model/source packages and an archive. Unlimited name matching keeps these discoverable as unselected candidates labeled for ownership review. It does not silently classify them as application residuals. The evidence index retains exact missing/extra path sets.

Validation: `sh scripts/go.sh test ./...` passed before the NFC correction; `sh scripts/go.sh test -race ./...` passed after it, together with `bun run typecheck`, `bun run test:web` (23 tests), and `bun run build:web`. `bun run generate` regenerated the changed MyGo contract. Logs are `/private/tmp/localops-coverage-{go-tests,race-tests,typecheck,web-tests,web-build,generate}.log`. The coverage UI detector reported no findings. These checks validate the scanner and generated HTML behavior, not native rendering or actual removal of user data.

### Remaining limits

The three main selected-app templates, both Flux adjustments, all three protection guards, Genius results, positive/empty Orphans/add-on outcomes, 15 mode/sort combinations and nine special-filename cases now have representative runtime evidence. This is not exhaustive program/input coverage or proof of equal-key sort stability. Static pseudocode or handler-entry hits alone are not counted as complete results. Neither product can guarantee all residuals or ownership when macOS denies access or names do not identify the application. LocalOps native rendering/lifecycle revalidation remains unperformed following the earlier automatic approval rejection for controlling LocalOps Dev; this work uses dev mode without packaging.
