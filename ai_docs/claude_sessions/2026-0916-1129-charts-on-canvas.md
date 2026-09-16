# Charts on core.Canvas: seven widgets and lesson 4.20

**Session:** c5e1f70f-c2c8-47b8-9d54-88a4a301474c
**Date:** 2026-09-16 11:29 (follows "clocks-canvas-alarm")
**Branch:** master (4991c48 → this commit)

## 1. The ask

"Start Tier D: charts on core.Canvas." That's Tier D of
`ai_docs/plans/clocks-canvas-charts.md`, carried as Next item 30 of the
previous doc.

## 2. What landed

No renderer changes. Everything is pure Go in `comps`, plus a tutorial lesson.

### `comps/chart.go`: the shared pieces

- **`ChartSeries{Name, Values, Color}`.** NaN is a missing value.
- **`niceScale(lo, hi, maxTicks, zero)`** uses Heckbert's nice numbers.
  - Steps are 1, 2 or 5 × 10ⁿ, and the ends are snapped outwards.
  - When rounding overshoots the tick budget, the step moves up to the next
    nice number. For example, 12..45 zero-based with 5 ticks gives 0..60 by
    20, not 0..50 by 10.
  - A flat or empty range is widened first.
- **`formatTick(v, step, extent)`:**
  - Decimals come from the step, so 0.5 is followed by 1.0, not 1.
  - The k/M unit is chosen once from the axis extent (≥10 000 → k,
    ≥1 000 000 → M), so every label on an axis shares a unit.
  - Zero prints as "0".
- **`chartPalette(theme)`:**
  - Order: Primary, Secondary, Warning, Success, Error, with duplicates
    dropped (DefaultTheme's Secondary and Success are both #34C759).
  - Error comes last because a red series reads as a problem.
  - After the roles, each colour repeats with alpha `99`.
  - `withAlpha` handles `#rgb` and `#rrggbb`, and leaves anything else
    unchanged.
- **Layout of a cartesian chart** (diagram in the file comment):
  - **Row:** a y-axis Column, then a plot Column (FlexGrow 1, FlexBasis 0).
  - **Plot Column:** a spacer of half a label line, the Canvas
    (CanvasStretch, viewBox 100×100, Height H px), another half-line spacer,
    then the x-label Row.
  - **Legend:** a wrapping Row under the chart, only when there are two or
    more series.
- **Y labels:**
  - Each of the N+1 labels sits in a box 14 px tall (`chartLabelLine`).
  - The column is H+14 tall, with Gap = (H − N·14)/N.
  - So box i starts at i·H/N, and its centre lands on gridline i thanks to the
    half-line spacer above the canvas.
  - No negative margins are used.
  - `chartMaxTicks(h)` = min(6, h/28 + 1).
- **X labels under a line** (`pointLabels`):
  - Point i sits at x = i/(n−1).
  - Stride s = ceil((n−1)/4).
  - The first box has weight s/2 and is start-aligned.
  - Interior boxes have weight s and are centred on their point.
  - A label on the last point gets s/2 and is end-aligned.
  - A final labelled point whose centred box won't fit is dropped, and the
    rest of the row becomes an empty filler.
  - Weights are FlexGrow plus FlexBasis("0"): StatTile's recipe, which makes
    CSS agree with the natives.
- **X labels under bars** (`bandLabels`):
  - n equal boxes, so labels are exactly under their slots.
  - Past 5 labels, only every s-th box has text, but all boxes are kept.
- **Accessibility:**
  - Each chart is a single `RoleImg` with a summary sentence.
  - Axis labels and the legend are hidden.
  - The canvas has no label, so it hides itself.

### The widgets

| Widget | File | Notes |
|---|---|---|
| `Sparkline` | `sparkline.go` | A bare labelled Canvas, 32 px tall. The domain is the data's own range, padded so the stroke and dot stay inside (`sparkScale`: margin = 1.5·sw/h). `Area` fills to the bottom edge, not to zero. `ShowLast` adds a dot. Always four shape slots. |
| `LineChart` / `AreaChart` | `line_chart.go` | `AreaChart` is `type AreaChart LineChart` and sets Area. `linePaths(values, n, scale, baseY)` makes one subpath per run of values; a lone value between gaps becomes a dot. All fills are drawn under all lines. `ZeroBased` is off for lines and forced on for areas. `Points` adds a dot on each value. The summary gives, per series, the count, first and last, low and high, each with its x label. A lone series' name is dropped when `Subject` is set. |
| `BarChart` | `bar_chart.go` | Equal slots; `BarFill` (default 0.7) is the group's share of its slot, with a 12% sliver between bars in a group. One path per series, holding each bar as its own subpath. The axis always includes zero, and negative bars hang below it. The zero line is drawn in `ControlBorderColor`. The summary reads every value up to 8 categories, and low and high beyond that. |
| `DonutChart` / `PieChart` | `donut_chart.go` | `PieChart` sets an unexported `pie` flag. Slices start at 12 o'clock in the caller's order. `Gap` (1.5° for donuts, 0 for pies) is taken out of both ends of a slice, unless the slice is alone or too thin to spare it. The gray track is drawn only when the total is 0; its slot is always kept. `CenterValue` and `CenterLabel` overlay the hole via a ZStack. The legend's percentages use `percentages()`, the largest-remainder method, with ties going to the earlier slice. |
| `Gauge` | `gauge.go` | The track and fill are strokes, round-capped (butt at 360°), in a square CanvasFit box. r = 50 − (thickness/2)·(100/size), so the caps stay inside. start = 90 + (360 − sweep)/2; sweep defaults to 240 and is clamped to 60..360. The value and label are centred via a ZStack. The fill's slot is kept even at 0. Label: "Battery, 72%", or "3 of 4" when there's no ValueText. |

### Design choices worth remembering

- **Dots and points are zero-length strokes with round caps**
  (`MoveTo(x,y).LineTo(x,y)`), not `core.Circle`. Under CanvasStretch the
  viewBox scales differently on each axis, so a circle becomes an ellipse;
  a stroke's width never scales.
- **Shape slots stay fixed** (empty `core.Shape{}` placeholders), so toggling
  a feature or a zero value patches one slot instead of shifting later ones.
  This follows the canvas's "one child per shape" rule.
- **The gauge box is square**, so its text centres on the arc's centre. The
  cost is empty space under a 180° gauge.
- **A `Subject` field** says what the chart measures and opens the spoken
  summary. It isn't drawn, since a screen heading usually says it already.

## 3. Tutorial lesson 4.20 "Charts"

- **Placement:** appended to chapter 4 (`lessonCharts` in `chapter4.go`), so
  no existing lesson ID moved.
- **Data:** fixed package vars rotated by a Shift button through
  `rotated[T]`, a generic helper. `chartSignups` has a NaN, which shows a gap.
- **Demos:**
  - LineChart with two series and points, then an AreaChart (Height 110)
  - BarChart (Height 120), then "N visits" beside a Sparkline
  - DonutChart with centre text, then a Gauge with a "Use 12%" button
- **Hints** were shortened to "Shift the data." and "Bars and a sparkline."
  because the longer ones wrapped the TRY IT badge onto two lines.

## 4. Knock-on edits the repo's tests required

- **Lesson count 58 → 59** in:
  - `README.md` (twice)
  - `core/deeplink.go`
  - `wasm/index.html`
  - `docs/tutorial-interactive.md`
  - `internal/shotclaims/shotclaims.go` (twice) and its test
  - `examples/tutorial/app_test.go` and `screenshot_test.go`
- **Screenshot:** retook `docs/images/tutorial-contents.png` with
  `wasm/shots/shoot.sh tutorial-contents`. It now reads "0 of 59 lessons
  opened".
- **Go-file count 534 → 541** in `wasm/verify/repowalks_test.go` (3 places)
  and `timings_test.go` (2). `sharedparse_test.go` counts untracked files
  too, so it caught the new ones before they were committed.
- **API docs:**
  - `internal/apidoc/packages.go` has a new comps topic, `charts` ("Charts"),
    listing the six widget files.
  - `mkdocs.yml` nav gained `api/comps-charts.md`.
  - `go run ./internal/apidoc/gen` regenerated 32 pages.
- **Plan doc:** the Status block in `ai_docs/plans/clocks-canvas-charts.md`
  now records Tier D and where it differs from the sketch.

## 5. Verification

- **Unit tests** in `comps/chart_test.go`:
  - `niceScale`: the data lies within the ticks, the budget is kept, steps
    are nice, and zero is included when asked.
  - `formatTick` units.
  - The palette: duplicates dropped, tints after the roles.
  - `withAlpha`.
  - `pointLabels`: the weights add up to the whole axis, and each interior
    box's centre is its point.
  - The y-axis Gap arithmetic.
  - LineChart's summary and hidden canvas, gaps splitting paths, the legend
    only for multiple series.
  - AreaChart includes zero; LineChart doesn't by default.
  - The BarChart path is two rectangles, with the negative bar's top on the
    zero line.
  - `percentages` adds up to 100.
  - Donut slots, summary and CanvasFit; the pie wedge starts at the centre.
  - Gauge label, clamping, empty fill slot, and radius.
  - Sparkline margins and the last dot.
- **Chrome, static htmlout export** (scratchpad test file, served with
  `python3 -m http.server`):
  - Every widget checked by eye; labels on gridlines; summaries read through
    `aria-label`.
  - Found the donut track showing through the gaps, and fixed it.
  - Found "Visits: Visits, …" repeating the name, and fixed it.
- **Chrome, live WASM tutorial at `#4.20`** (built into the scratchpad; the
  tracked `wasm/main.wasm` was untouched):
  - Shift and "Use 12%" re-rendered the charts.
  - Every summary changed, and the gauge went to 60%.
- **At the end:** `gofmt -l` clean, `go vet ./...` clean, `go test ./...`
  passes.
- **Not run this session:** `android/verify`, `ios/verify`, and the WASM Node
  suite beyond `go test ./wasm/verify`. No native code changed.

## 6. Harness notes

- **The page wouldn't scroll to the bottom** of the static export
  (viewport-limited). Moving the gauge row to the top of `body` with JS and
  zooming in worked.
- **The zoom screenshot action** was handy for checking the gauges up close.
- **gofmt re-indents box-drawing diagrams** in doc comments (the gauge
  diagram lost its leading spaces). It still reads fine.
- **One Bash call failed with a transient classifier outage**; retrying later
  worked.
- **Both http.server tasks were stopped** by task ID, and the Chrome tab was
  closed.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

**Closed or changed this session** (previous numbering):

- 30 (Tier D charts) is done.
- 38 (TRY IT badge wraps) came up again in 4.20. It's still worked around, so
  its value rises to medium.

1. **(age ≥18 · value high) Lessons on hardware, and checks still open.**
   - Real devices for everything (deferred by the user).
   - **Screen readers:**
     - radio and StepIndicator done-step semantics
     - combobox active option
     - "pop-up" on Menu/DatePicker triggers
     - aria-current/selected on current items
     - Calendar's grid on the web (needs a person with VoiceOver)
     - CodeEditor toolbar role on Compose
     - SearchableSelect natives reading field + list + status
     - AccessibilityHidden confining TalkBack/VoiceOver behind a Drawer
     - VoiceOver on the calendar cell now that it is one element
   - **iOS:**
     - sheet `Dialog` and ActionSheet filler
     - Drawer slide by eye and under Reduce Motion
     - RTL for Drawer and CodeEditor
     - editing in the sideways code editor
     - `banded` and a drag on 4.6's list
   - **Pinning and Drawer:**
     - `Screen.Footer` pinning
     - 100% layers in a pinned-height ZStack
     - the panel's `Height 100%` in an HStack
     - `core.Focus` on a Button
     - a hardware keyboard reaching a shut panel
   - **Android:**
     - predictive back from contents
     - MaxWidth where it binds (tablet)
     - a capped stretched child in RTL
2. **(age ≥18 · non-goal) Rename `docs/components.md` to `comps.md`.**
3. **(age ≥18 · non-goal) Rewrite `components` in the older plans.**
4. **(age ≥18 · non-goal) Trim the copied Android shell's permissions.**
5. **(age ≥18 · non-goal) Replace the iOS usage strings further.**
6. **(age ≥18 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
7. **(age 17 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android.
8. **(age 17 · non-goal) An AppBar outside the Navigator** with a custom
   `OnBack` is outranked by the Navigator's pop on Android.
9. **(age 16 · non-goal) Forward does not re-open a screen left by browser
   back.**
10. **(age 15 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose.
11. **(age 13 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain. Not profiled.
12. **(age 13 · non-goal) MaxWidth with a growing sibling on the natives.**
13. **(age 13 · non-goal) The typed-hash fold's `history.length` fallback.**
14. **(age 13 · non-goal) A page's own `pushState` while a claim is on
    screen.**
15. **(age 13 · non-goal) A Drawer's shut panel is composed on the natives.**
16. **(age 11 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android.
17. **(age 5 · value low) Compose's today `stateDescription` not heard** under
    TalkBack on the emulator.
18. **(age 5 · value medium) F-keys on iOS unverified; key delivery to a
    simulator is unreliable.**
19. **(age 5 · value low) Page-global chords on iOS are verified once.**
20. **(age 4 · non-goal) GameController cannot take a key.**
21. **(age 4 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
22. **(age 3 · value low) The Compose Row hug is unswept in `comps` widgets'
    own internal Rows.**
23. **(age 2 · value medium) The min-content floor has not been seen on a
    device.** Lessons 1.1 and 8.2 are the screens that should change.
24. **(age 2 · value low) `content ÷ N%` arithmetic unwatched** for a Row child
    with a percentage MaxWidth.
25. **(age 2 · value low) A plain horizontal strip in a Row now hugs.** Unseen.
26. **(age 2 · value low) The iOS chord gate is unheard.**
27. **(age 2 · value low) Merging a labelled node is unheard under TalkBack.**
28. **(age 2 · value medium) `.claude/settings.json` cannot be edited from a
    session.** Worth an upstream report.
29. **(age 2 · value low) `android/verify/sources.sh` needs `ANDROID_HOME` and
    a filled gradle cache**, and SKIP reads like a pass.
30. **(age 1 · value high) `core.Canvas`, lesson 4.19 and now 4.20 have not
    run on an Android emulator or iOS simulator.**
    - Compiled and held to Go's geometry on a JVM and macOS only.
    - Look for:
      - aspect-ratio sizing with and without Width/Height
      - stroke width under `CanvasStretch`
      - an update-props patch redrawing (the Compose draw-phase read, the
        SwiftUI observation of `children.map(\.props)`)
      - dashes
      - the analog clock's hands pivoting at the centre and `Smooth`
        animating forward
      - **new for charts:**
        - round-capped zero-length strokes drawing as dots (Skia and
          CoreGraphics)
        - y labels landing on gridlines, given each platform's text line
          height inside a 14 px box
        - FlexGrow weights of 1.5/3/0.5 splitting the x-label row in
          proportion
        - the `#rrggbb33` area tint
31. **(age 1 · value medium) Tier E: an alarm that rings with the app
    closed.**
    - `LocalNotification.At` (UNCalendarNotificationTrigger / AlarmManager;
      Android 12+ needs `SCHEDULE_EXACT_ALARM`).
    - Or iOS AlarmKit / Android `AlarmClock`.
    - Browsers cannot schedule a notification.
32. **(age 1 · value low) Alarm sound and haptics are unheard.**
    - The lesson passes no `Sound`.
    - Ringing replaces whatever core's single audio player was playing and
      doesn't resume it (documented).
    - Haptic pulses were not felt on a device.
33. **(age 1 · value low) A snoozed alarm coming back was not watched live**
    (unit-tested only).
34. **(age 1 · non-goal) `DigitalClock` digits can shift width by a pixel.**
    There's no font family in `core.Style`; centred to hide it.
35. **(age 1 · non-goal) `AnalogClock{Smooth}` spins back once if the tick at
    00:00:00 is skipped** (an app suspended over midnight).
36. **(age 1 · non-goal) Canvas v1 omits** text inside the drawing, gradients,
    clipping, per-shape hit-testing, and the even-odd fill rule.
37. **(age 1 · value medium) `demoPanel`'s TRY IT badge wraps under a hint
    longer than about 25 characters.** Worked around twice now (4.19, 4.20)
    by shortening hints; other lessons' hints were not checked. The badge
    probably wants `FlexShrink(0)`.
38. **(age 0 · value low) A chart's hidden data table** (the plan's "may add a
    hidden data table") was not built. The one-sentence summary is all a
    screen reader gets; a BarChart with more than 8 categories only gives the
    low and high.
39. **(age 0 · value low) Chart types not built:** smoothed (monotone cubic)
    lines, stacked areas and bars, horizontal bars, scatter, and value labels
    on bars.
40. **(age 0 · value low) An x label wider than its slot** widens the slot and
    pulls neighbours off their points. Capped at 5 labels, which is enough
    for "Jan" at phone width. Long labels aren't truncated.
41. **(age 0 · non-goal) A 180° `Gauge` leaves its bottom half empty.** The box
    is square so the value centres on the arc; trimming it needs a text
    offset the framework avoids.
42. **(age 0 · value low) The chart palette has only 3–5 distinct hues** per
    bundled theme before it repeats as 60%-alpha tints. A 6-slice donut
    therefore has look-alike pairs; callers can pass `Colors`.
43. **(age 0 · value low) Chart summaries are English** ("points", "from",
    "low", "high", "of", "no data"), the same issue other widgets' spoken
    strings have. `AccessibilityLabel` overrides them.
