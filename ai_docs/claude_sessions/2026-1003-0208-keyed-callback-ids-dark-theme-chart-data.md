# Keyed callback IDs, a bundled DarkTheme, chart data per host

Session: `352537a5-111c-4a35-801f-8c2b8864140a`

## Ask

Do N-010 (a chart's hidden data table, pasted from the cats-todo backlog)
along with N-073 and N-074, the three API-decision items whose 2026-09-29
recommendations had not been acted on. Then `/sw`.

## Commits

| Commit | Item |
| --- | --- |
| c43e3bc | N-073, N-074 (closing N-018), N-010 |

One commit. The generated API pages and the file-count sentences span all
three items, and splitting would leave intermediate commits with stale docs.

## N-073: identity-keyed callback IDs

- **Shape:** the registry keeps a stack of ID scopes. `core.Keyed` and each
  Navigator frame push one while their subtree renders (`Context.keyScope`,
  with a deferred close).
  - An ID is the kind's prefix, the scope's path, then the counter:
    `cb_r2/e0/1`.
  - The enclosing scope's counters do not move.
  - Root IDs keep `cb_N`, so an app with no keys sees no change.
- **Collisions:**
  - Keys are unique among siblings, not within a scope. Two keyed lists under
    one unkeyed column open the same keys, so a repeat is spelled `0~1`.
  - `/`, `~` and `%` are escaped (`idKeyEscaper`), so a key cannot forge a
    boundary.
- **ErrorBoundary:** the counter snapshot alone cannot undo registrations made
  in nested keyed scopes. `rollbackCounters` now walks a trail of registered
  IDs and child-key counts.
- **Back:** Navigator's own Pop (`withSystemBackPop`) stays outside the frame
  scope, so two quick presses on a three-deep stack still pop twice.
  `back_cb_` keeps its namespace for that reason.
- **Results:**
  - EditableGrid's EDIT transitions re-bind no other cell. The new test shows
    `cb_1`→`cb_6` with the scope removed.
  - A stale ID from a popped frame reaches nothing.
  - `OnEndReached`'s cross-navigation guard edge closed as a side effect; its
    doc was rewritten.
- **Checks:**
  - No host parses or selects on IDs (they are opaque strings).
  - The full Go suite and `wasm/verify/run.sh` passed unchanged apart from the
    generated docs.
  - Mutation-tested: Keyed scope, frame scope, escaping, the occurrence
    suffix, and both halves of the rollback. The marks half first survived,
    because the fallback re-registered the same ID; the test now has the
    abandoned subtree register two handlers.

## N-074: core.DarkTheme

- **The theme:** the tutorial's palette moved into core as `newDarkTheme()`,
  a recoloured copy of DefaultTheme, and is in `BundledThemes`.
- **The source-derived list test:** `TestBundledThemesListIsExhaustive` now
  also accepts a package-level var built by a no-argument `*Theme`
  constructor declared in theme.go. The alternative, a second full literal,
  could drift from DefaultTheme's geometry. A new core test pins the geometry
  and checks there is no write-through.
- **Censuses:**
  - Every contrast census passed on first contact.
  - The palette witness table gained DarkTheme in four rows.
  - palette.mjs gained its seven rows, and the browser paints them.
  - Census figures replaced the hand ones; the divider is 1.45:1, not 1.5.
- **Rename:**
  - `*OnLight` → `*Ink` (`PrimaryInk`, …).
  - The reverse lookup is `AsInk`, because `Variant.Ink` already meant the
    ink over a fill.
  - Methods keep deprecated wrappers. Fields could not: Go has no field alias,
    and a deprecated duplicate field would make
    `theme.Colors.PrimaryOnLight = ""` a silent no-op. That is the
    half-branded-app bug the release idiom exists for. So a compile error is
    the deprecation for a field.
- **Prose:** "on-light tone" became "ink tone" through comps, the theming
  guide, components.md (table gains a DarkTheme column) and lessons 7.3/7.4.
- **Mistake caught:** the blanket rename also rewrote
  `themenearmiss_test.go`'s recorded name list, a snapshot the test replays
  exactly. It was reverted.

## N-010: chart data per host

- **Core:** `core.ChartData` (title, x/y axes, series of points; categorical
  or numeric x), attached by `core.AccessibilityChart` as the `chartData`
  prop.
  - The stored value is a deep copy with non-finite values dropped, because
    encoding/json refuses NaN.
- **comps:** `seriesData` builds it for the categorical charts.
  - Candlestick: four price series. Histogram: bins × counts.
  - Scatter: numeric x, with the column heads "x"/"y" coming from comps
    rather than being invented by a renderer.
  - Donut and funnel: shares or rates in the cell text.
  - Heatmap: the grid's own shape.
  - A stacked chart's data is its input, not the totals it draws.
  - Sparkline and Gauge carry none.
- **Web (runtime + htmlout):** a visually hidden `<table>` as *leading*
  chrome (`data-grmob-chrome="charttable"`).
  - Leading because `chromeOffset` counts only leading chrome.
  - The element's role becomes `figure`: an img's children are
    presentational. `data-grmob-chart` lets `applyAccessibility` keep the
    swap across update-style patches.
- **iOS:** `GrMobChartAccessibility`, one concrete `.modifier` after
  `grMobBox` on Row and Column (FunnelChart's root is a Row). It wraps a
  `GrMobChartDescriptor: AXChartDescriptorRepresentable`, kept in
  GrMobStyle.swift so no xcodegen is needed.
- **Android:** nothing.
- **Checks:**
  - htmlout unit tests.
  - gen.go `chartCases` (seven real charts) against `chart_test.mjs`: table,
    style, roles, paths unmoved, data removed and re-added, add past the
    table.
  - The full `wasm/verify/run.sh`, overflow sweep included.
  - `ios/verify/run.sh`, including the Release build and the iOS 17 SDK
    typecheck.
  - A scratch CDP script reading Chrome's accessibility tree on lessons 4.20
    and 4.36. Every chart is a `figure` named by its summary, holding a table
    with row headers and cells (`rowheader:Rent | cell:$1.2k`).

## Also

- The tracked-Go-file count sentences: 672 → 678.
- `chart_data.go` was added to apidoc's accessibility topic.
- A Mach-O `verify` binary, left at the repo root by `go build ./wasm/verify`,
  was removed.
- **Not this session's:** `GrMobSurface.kt`, `GrMobSurface.swift`,
  `Renderer.kt` and `AppWindow.swift` appeared in the tree at 02:05. They are
  apparently session grmob-54's N-085 work, and `git add -A` swept them into
  the commit. They were taken back out before pushing and left in the working
  tree untouched.

## Next

Closed: N-010, N-018, N-073, N-074. Declined: None. Raised: None.
Deferred: None. Promoted: None.
Updated: N-070. Full list: `ai_docs/todo/next-list.md`.
