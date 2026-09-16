# Next list: a notification sweep, core.MaxLines, new chart types and the even-odd rule

**Session:** 4649ae8f-d316-48bd-b800-17eecd35cce3
**Date:** 2026-09-16 13:31 (follows "next-list-exact-alarms-boot-rearm-and-sweeps")
**Branch:** master (46d6eb2 → this commit)

## 1. The ask

"Keep going through the Next list doing what we can including the large items -
just process them in reasonable batches." Three batches:

1. The alarm and notification items (39, 42, 43, 46, 47, 48).
2. Charts (33, 34), which needed a new text line cap first.
3. DataTable widths (45), the even-odd fill rule (part of 31), and two layout
   watches (23, 24).

Items that need a design decision (36, 32, 40, 38), a person or hardware were
left and are listed in Next.

## 2. Batch 1: alarms and notifications

### `core.SweepNotifications` (items 43 and 39)

`UseAlarms` could only cancel the ids in its in-memory list, so a dead
process's alarms stayed with the OS, and an alarm rung while the app was
closed never reached `OnRing`. The shells already know both answers.

```
app ──"notification" {command: "sweep", prefix, request}──────▶ host
app ◀──"notification_swept" {request, fired: [ids]}──────────── host
```

- **core** (`core/notifications.go`): `SweepNotifications(prefix, fn)`. It uses
  a correlation id in `ReadClipboard`'s mould and is dispatched from
  `ReceiveHostEvent`. An empty prefix, or no host, answers at once with no ids.
- **Android** (`Notifications.kt`):
  - The store now marks entries `fired` instead of removing them.
  - `sweep` cancels the alarm and banner for every stored id under the prefix,
    plus shown banners found by `activeNotifications` tag, and reports the
    stored ids whose `at ≤ now`.
  - It logs one line: `swept N under P; fired=[…]`.
  - `attach` now takes the host-event reporter.
- **iOS** (`Notifications.swift`):
  - A UserDefaults map `grmob.scheduled-notifications` (id → ms), written on
    schedule and removed on cancel or an immediate post.
  - The sweep takes its ids from that map synchronously, not from
    `getPendingNotificationRequests`. That call answers later, and a sweep
    built from its answer could remove requests the app scheduled in between.
  - Delivered banners under the prefix are removed asynchronously, only if
    they were delivered before the sweep began.
- **Web**: a `fired` map beside the timers. It dies with the tab, as the
  timers do.
- **Hook** (`hooks/alarms.go`):
  - `UseAlarms` sweeps `grmob.alarm.` at mount when in the foreground.
  - If it mounted in the background, `sweepOnReturn` defers the sweep to the
    first return. That check sits before `setAway`'s Notify-off early return.
  - `firedAlarms` parses `grmob.alarm.<id>.<unix>`, splitting at the last dot
    so an id containing dots still parses. It skips ids no longer in the list
    and reports each alarm once, soonest first.
  - The sweep is sent before the lifecycle subscription, so it reaches the
    host before any of this record's own posts.
- **Spelling pin:** `TestNotificationEventSpellingsAgree` now also requires
  `"sweep"`, `"notification_swept"`, `"prefix"`, `"request"` and `"fired"`.

### Item 46: posting a missed banner twice

- Both `postScheduled` (the alarm) and `rearm` (attach/boot) now **claim** an
  entry under one lock: an entry already fired, or not in the store, is not
  posted.
- Fired entries are pruned by `rearm` a week after their time.
- The race itself was not reproduced; the fix is argued from the code.

### Item 47: an exact-alarm grant

- `NotificationBootReceiver` also filters
  `android.app.action.SCHEDULE_EXACT_ALARM_PERMISSION_STATE_CHANGED`.
- On it, the receiver re-arms (inexact alarms become exact) and calls
  `Permissions.recheckExactAlarms()`, which sends nothing without an Activity.

### Item 42

4.19's notification status now uses `hooks.UsePermissionLive`.

### Item 48

- `skipped()` moved into `internal/gateharness/skip.sh`, sourced by both
  `android/verify/gate.sh` and `ios/verify/gate.sh`.
- `ios/verify/run.sh`'s app-layer skip uses it.

### Seen

- **Android, `am kill` then relaunch:**

  | Relaunch | Log | Alarms left | Store |
  |---|---|---|---|
  | After the minute | `fired=["grmob.alarm.soon-1.1789580760"]`; the entry had been `fired:true` and the banner up before relaunch | 0 | empty |
  | Before the minute | `fired=[]` | 0 (1 before) | empty |

- **Android, exact-alarm grant:** revoked via appops (both modes), scheduled
  (`window=+23s507ms`), then granted: `window=0 exactAllowReason=permission`
  with no app action.
- **Android, backgrounded (not killed):** the alarm fired and was marked
  fired. The return cancelled it through the old per-id path, with no sweep
  logged.
- **iOS:** new `testRelaunchSweepsWhatTheDeadProcessScheduled` in
  `TutorialAlarmNotifyUITests`. It schedules, goes Home, terminates,
  relaunches 4.19, goes Home, and asserts no "Try it" banner by the minute.
  - It passes (74 s), beside the existing Allow test (64 s).
  - **Mutation:** with the Swift sweep call replaced it fails ("the dead
    process's alarm still rang"). Restored and re-run: passes.
- **Strict drill:** a fake `xcrun` failing only for `iphoneos`. Strict mode
  exited 3 with the FAIL line; lenient mode exited 0.

## 3. Batch 2: `core.MaxLines` and charts

### `core.MaxLines(n)`, needed by item 34

- **Style field:** `MaxLines int`, placed inside the aligned field block with
  a trailing comment. A doc comment there made gofmt re-align five
  neighbours.
- **Each target:**

  | Target | One line | More than one |
  |---|---|---|
  | CSS, both web targets | `white-space:nowrap; overflow:hidden; text-overflow:ellipsis`, display `block` when unset | `display:-webkit-box; -webkit-box-orient:vertical; -webkit-line-clamp:n; overflow:hidden` |
  | Compose, `GrMobText` | `maxLines = n`, `TextOverflow.Ellipsis` | same |
  | SwiftUI, `GrMobText` | `.lineLimit(n).truncationMode(.tail)` | same |

  The author's own overflow and white-space win. The runtime clears
  `textOverflow`, `webkitLineClamp` and `webkitBoxOrient` when the cap goes
  (totality).
- **Why one line uses nowrap:** a clamp cuts only between lines, so a single
  word wider than its box would overflow uncut.
- **iOS floor:** `GrMobMinContent` floors a Text with `maxLines > 0` at 0,
  matching CSS, where `overflow:hidden` zeroes the automatic minimum.
  - Harness: `ios/verify/mincontent.swift`. Mutation: removing the guard fails
    it with "got 67.02".
- **Tests:**
  - `wasm/verify/maxlines_test.mjs` (4 tests)
  - `htmlout` `TestMaxLinesExportsTruncation`
  - `mobile/verify/maxlines_test.go` (parse and apply pins)
  - `totality_test.mjs`'s `FULL_STYLE` gains `MaxLines: 2`

### Item 34: x labels

- `weightedLabel` gives each slot `MinWidth("0px")` and its Text
  `MaxLines(1)`.
- Why the slot needs its own zero minimum on CSS: a flex item's automatic
  minimum is its content's min-content contribution, and for a nowrap run
  that is the whole run, however it clips.
- The spoken summary still reads full labels.

### Item 33: chart types

- **`LineChart.Smooth`:** a Fritsch–Carlson monotone cubic (`traceRun`,
  `monotoneTangents`). Monotone between points, so it invents no peak. It
  survives CanvasStretch, since each axis is scaled separately.
- **`LineChart.Stacked`** (`AreaChart{Stacked}`):
  - `stackSeries` draws running totals, counting a NaN as 0.
  - Each band's floor is the previous total, traced backwards on the same
    curve.
  - Band alpha is `66` (unstacked areas `33`).
  - The summary reads each series' own values.
- **`BarChart.Stacked`:** positives stack up and negatives down, each with
  its own total (`stackRange`, `barRects`).
- **`BarChart.Horizontal`:**
  - Bands in px (`bandColumn`); `Height` defaults to 28 px a category, at
    least 56.
  - A name column capped at `LabelWidth` (default 96 px, fixed; a percentage
    was avoided because Compose and CSS resolve it differently).
  - Ticks on the point axis via `pointLabels`, with a budget of 5 so every
    tick is labelled.
- **`BarChart.ShowValues`:**
  - A value row above vertical bars. `cartesianFrameWithTop` gives the y axis
    a matching spacer.
  - A value column beside horizontal bars.
  - Cells are weighted by the same arithmetic as `barRects` (pad, bar(s),
    pad), so each cell centres on its bar.
  - A stack shows its total.
- **`ScatterChart`** (`comps/scatter_chart.go`): `ScatterSeries` of
  `ChartPoint{X, Y}`. The x axis is nice ticks on the point axis, dots are
  zero-length round strokes, and the summary gives the count and x and y
  ranges.
- **Tests:** `TestSmoothLineIsMonotone` (samples each Bézier and checks the
  reversed controls), `TestStackedAreaDrawsTotals`,
  `TestStackedBarsRunEndToEnd`, `TestHorizontalBarsLieAlongX`,
  `TestBarValueCellsCentreOnBars`, `TestScatterChart`,
  `TestXLabelsAreCappedInTheirSlots`.
- **Lesson 4.20:** a third panel, "Stacks, sideways bars and a scatter". The
  demo's money `Format` is compact ("$1.2k"), because the end ticks have half
  an interval each and "$1500" was cut to "$15…".
- **`TutorialChartsUITests`:** scrolls to the new panel, asserts the four
  summaries, and checks that the cut name is spoken whole.

### Seen

- **Emulator:** a smooth stacked area, horizontal bars with "Groceries and
  hou…" and "Subscriptions and …", stacked bars with totals, and the scatter.
  After Shift, the stacked area follows its data.
- **Simulator:** the same; quarter labels are cut ("First quart…"). The test
  passes (26 s).
- **Chrome** (htmlout export, headless screenshot): cuts and alignment match.
- **A false alarm, recorded so it is not chased again:** downscaled simulator
  screenshots looked as if area fills stopped following their lines after
  Shift. Go's patches were correct and the decoder was fine. At full
  resolution (`sips --cropToHeightWidth` + `--cropOffset`) every fill follows
  its line. Crop before believing a downscaled chart.

## 4. Batch 3

### Item 45: `Column.Width`

- `DataTable`'s `Column.Width` is a px content width on an inner, unpadded
  Row. It sits there because the targets disagree on whether a padded box's
  width includes the padding (the web has no global `box-sizing`).
- Without a Weight the cell gets `FlexShrink(0)` and the inner box an exact
  `Width`. With a Weight, `MinWidth` plus grow.
- `Align` justifies inside the inner box. Test: `TestDataTableColumnWidth`.
- **4.6:** the Date column has `Width: 56`. Seen on the emulator: Speaker now
  starts at the same x on every row.

### Part of item 31: `Shape.FillRule` / `core.FillEvenOdd`

- **Wire:** `fillRule: "evenodd"`, written only with a fill.
- **Targets:** SVG `fill-rule` (htmlout, and the runtime's
  `CANVAS_SHAPE_ATTRS`), Compose `PathFillType.EvenOdd`, SwiftUI
  `FillStyle(eoFill:)`.
- **Tests:**
  - `core` `TestFillRuleIsWrittenOnlyWhenItPaints`
  - a `wasm/verify/gen.go` canvas case (runtime held to htmlout)
  - `mobile/verify/fillrule_test.go`, which uses `valuesIn` because
    `codeIn` blanks string literals
- **4.19:** a ring pair (even-odd hole beside a solid nonzero disc). Seen on
  the emulator, simulator and Chrome.

### Found on the way: an iOS canvas squeezed in a Row

- **The bug:** on the simulator, 4.19's donut (existing) and the new rings
  overlapped their captions. The Row shrank the `Width("110px")` canvas and
  the drawing kept its size. Chrome and the emulator keep 110.
- **The cause:** a canvas is an `<svg>`, a replaced element, whose content
  size suggestion is its natural width (300 px without size attributes). So
  CSS floors it at min(declared, 300). `GrMobMinContent` floored every
  declared width at 0.
- **The fix:** a Canvas with a px width floors at `min(w, 300)` plus margins.
  A percentage, or any other node type, still floors at 0.
- **Checked:** four cases in `mincontent.swift`, and the simulator shows both
  rows clear of their captions.

### Items 23 and 24: watched

A throwaway `examples/zzwatch` (deleted) was bound to the Android shell and
exported through htmlout into headless Chrome at 411 px:

| Case | Compose and CSS |
|---|---|
| 50% `MaxWidth` where the content binds | the box hugs "Short" |
| 50% `MaxWidth` where the cap binds | half the row |
| A plain horizontal strip in `Row(Justify(End))` | hugs its chips at the end |
| The strip beside a sibling | hugs, sibling after |

Compose and Chrome agree in every case; only line breaks differ, by font.

## 5. Knock-on

- `docs/api/*` regenerated. `internal/apidoc/packages.go` adds
  `scatter_chart.go` to Charts and a new blurb.
- `wasm/verify/repowalks_test.go` and `timings_test.go`: the tracked-Go-file
  count 543 → 546, as `sharedparse_test` requires when files are added.
- **Plan doc** `ai_docs/plans/clocks-canvas-charts.md`: the sweep, the
  exact-alarm grant, the second chart round, and even-odd moved out of
  non-goals.

## 6. Verification

- `gofmt -l` clean; `go vet ./...` clean; `go test ./...` passes.
- `node --test wasm/verify/*_test.mjs`: 559 pass, 0 fail. `wasm/verify/run.sh`
  exits 0.
- `ios/verify/run.sh`, `android/verify/run.sh` and
  `android/verify/sources.sh`: all OK.
- **XCUITests:**
  - `TutorialAlarmNotifyUITests`, both tests
  - `TutorialChartsUITests`
  - `TutorialClockLocalTimeUITests` (twice, before and after the canvas
    floor)
- **Left running:** the emulator (tutorial APK with this session's code) and
  the booted simulator (tutorial build).

## 7. Harness notes

- **`scratchpad/ad.sh`** was rewritten; the last session's copy was in its own
  scratchpad. Its `find` and `tap` parse the uiautomator dump with python.
  - `alarms` counts pending grmob alarms between `pending alarms:` and
    `LazyAlarmStore` in `dumpsys alarm`. The per-uid history further down
    otherwise gets counted too.
  - `store` cats the prefs XML with `run-as`.
- **A "System UI isn't responding" dialog** appeared on the emulator once;
  tapping Wait cleared it.
- **zsh has lowercase `pipestatus`** (the previous doc's note about
  `PIPESTATUS` is right about the uppercase one).
- **Relative paths from `cd android`:** `android/build.sh ../examples/tutorial`
  fails silently. Run it from the repo root with `./examples/tutorial`.
- **`xcodegen generate`** is needed after adding or removing a UITest file.
- **Headless Chrome:** `"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new --window-size=W,H --screenshot=out.png file://…`
  works for htmlout exports.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

**Closed this session** (previous numbering):

- **23** (`content ÷ N%` unwatched) and **24** (strip hug unseen): watched on
  the emulator beside Chrome; they agree.
- **33** (chart types): Smooth, Stacked areas and bars, Horizontal,
  ShowValues, ScatterChart.
- **34** (x label widens its slot): `core.MaxLines` on four targets, applied
  to axis labels.
- **39** (OnRing misses alarms rung with the app closed) and **43** (a dead
  process's alarms can't be cancelled): `core.SweepNotifications` and the
  mount sweep.
- **42** (4.19 Allow notifications stale): `UsePermissionLive`.
- **45** (DataTable columns drift): `Column.Width`.
- **46** (re-arm can post twice): a claim under one lock. The race was not
  reproduced.
- **47** (exact-alarm grant read only on foreground): the grant broadcast
  re-arms and re-checks.
- **48** (`GRMOB_VERIFY_STRICT` Android-only): shared `skip.sh`, used by
  `ios/verify`.
- **31, in part:** the even-odd fill rule. The rest stays open below.

1. **(age ≥21 · value high) Lessons on hardware, and checks still open.**
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
2. **(age ≥21 · non-goal) Rename `docs/components.md` to `comps.md`.**
3. **(age ≥21 · non-goal) Rewrite `components` in the older plans.**
4. **(age ≥21 · non-goal) Trim the copied Android shell's permissions.**
5. **(age ≥21 · non-goal) Replace the iOS usage strings further.**
6. **(age ≥21 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
7. **(age 20 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android.
8. **(age 20 · non-goal) An AppBar outside the Navigator** with a custom
   `OnBack` is outranked by the Navigator's pop on Android.
9. **(age 19 · non-goal) Forward does not re-open a screen left by browser
   back.**
10. **(age 18 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose.
11. **(age 16 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain. Not profiled.
12. **(age 16 · non-goal) MaxWidth with a growing sibling on the natives.**
13. **(age 16 · non-goal) The typed-hash fold's `history.length` fallback.**
14. **(age 16 · non-goal) A page's own `pushState` while a claim is on
    screen.**
15. **(age 16 · non-goal) A Drawer's shut panel is composed on the natives.**
16. **(age 14 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android.
17. **(age 8 · value low) Compose's today `stateDescription` not heard** under
    TalkBack on the emulator.
18. **(age 8 · value medium) F-keys on iOS unverified; key delivery to a
    simulator is unreliable.**
19. **(age 8 · value low) Page-global chords on iOS are verified once.**
20. **(age 7 · non-goal) GameController cannot take a key.**
21. **(age 7 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
22. **(age 6 · value low) The Compose Row hug is unswept in `comps` widgets'
    own internal Rows.** Items 23 and 24 were watched on a scene built for
    them, not on the widgets.
23. **(age 5 · value low) The iOS chord gate is unheard.**
24. **(age 5 · value low) Merging a labelled node is unheard under TalkBack.**
25. **(age 5 · value medium) `.claude/settings.json` cannot be edited from a
    session.** Worth an upstream report.
26. **(age 4 · value low) Alarm sound and haptics are unheard.**
    - The lesson passes no `Sound`.
    - Ringing replaces whatever core's single audio player was playing and
      doesn't resume it (documented).
    - Haptic pulses were not felt on a device.
    - The Notify banner uses the platform's default sound, also unheard.
27. **(age 4 · non-goal) `DigitalClock` digits can shift width by a pixel.**
    There's no font family in `core.Style`; centred to hide it.
28. **(age 4 · non-goal) `AnalogClock{Smooth}` spins back once if the tick at
    00:00:00 is skipped** (an app suspended over midnight).
29. **(age 4 · value low) Canvas v1 still omits** text inside the drawing,
    gradients, clipping and per-shape hit-testing. The even-odd rule was
    added this session. Gradients are the cheapest next piece (SVG `<defs>`,
    `Brush.linearGradient`, SwiftUI `.linearGradient`).
30. **(age 3 · value low) A chart's hidden data table** was not built.
    - The one-sentence summary is all a screen reader gets.
    - A BarChart with more than 8 categories only gives the low and high.
    - It needs a screen-reader-only primitive that core doesn't have: an API
      decision.
31. **(age 3 · non-goal) A 180° `Gauge` leaves its bottom half empty.**
32. **(age 3 · value low) The chart palette has only 3–5 distinct hues** per
    bundled theme.
    - A fix is a new theme role (for example a categorical list on
      `ColorPalette`). That is a theme-API and colour decision for each
      bundled theme, put to the user and not yet answered.
33. **(age 3 · value low) Chart summaries are English**, like other widgets'
    spoken strings. `AccessibilityLabel` overrides them.
34. **(age 2 · value low) A Notify alarm is a banner, not a ringing screen.**
    - There's no full-screen intent, no looping sound, and no Snooze action on
      the notification.
    - iOS AlarmKit and Android `AlarmClock` / full-screen intents are the
      route to a real system alarm. Large, and only a device will tell.
35. **(age 2 · value low) `mobile.SetTimeZone` runs once at startup.**
    - A time-zone change while the app runs is picked up on the next launch,
      because `time.Local` can't be written safely while goroutines read it.
    - A fix needs core to hold the current location for hooks and widgets:
      an API decision.
36. **(age 2 · value low) The web's scheduled notification is unseen in a real
    browser** (Node tests only), and now so is its sweep.
    - It needs a granted permission in Chrome and a tab left open.
    - Chrome's permission prompt is browser UI the automation can't click, so
      a person has to grant it once.
37. **(age 1 · value low) Compose Rows don't shrink children in proportion.**
    - 8.2's "Advance both counters" keeps one line and Repair takes three;
      CSS squeezes both.
    - Documented in `Renderer.kt` (pinMainAxis) and pinned by
      `internal/pinfixture`.
38. **(age 1 · value low) Android's exact-alarm grant is only re-checked for a
    running Activity.** The broadcast re-arms in any process, but a process
    started for the receiver sends no permission update; the next launch
    checks afresh.
39. **(age 0 · value low) The end tick labels have half an interval of width.**
    On a horizontal BarChart or a ScatterChart, a wide `Format` ("$1500") is
    cut with an ellipsis. The box cannot extend past the plot's edge;
    documented on `BarChart`. A shorter format is the workaround.
40. **(age 0 · value low) Value labels sit along the plot's edge, not at each
    bar's tip.**
    - Text cannot go inside `core.Canvas`.
    - Following a tip would need the drawn plot size, which no target reports
      to Go.
    - On a stacked chart whose total sits well under the top tick, the row
      reads a little detached. Documented under "Values".
41. **(age 0 · non-goal) A stacked chart counts NaN as 0 and stacks
    negatives through the band below.** A stack is for parts of a whole;
    documented under "Stacked".
42. **(age 0 · value low) Only one `UseAlarms` with Notify per app.** The
    sweep prefix `grmob.alarm.` belongs to the hook, so a second instance
    would sweep the first one's notifications. Documented.
43. **(age 0 · value low) The relaunch sweep also takes down shown banners.**
    - After a force stop, `attach` posts a missed alarm late, and the mount
      sweep removes that banner moments later.
    - The in-process return already did the same through per-id cancel.
      Consistent, but the late banner barely shows.
44. **(age 0 · value low) Item 46's double-post claim is unreproduced.** The
    window is the moment of launch.
45. **(age 0 · value low) iOS's floor for replaced elements covers Canvas
    only.** An Image or MapView with a declared px width still floors at 0,
    where CSS floors an `<img>` at min(declared, natural). Not seen to bite.
46. **(age 0 · value low) `MaxLines` on the natives applies to Text only.**
    The web targets write the declarations on any node. Documented on
    `core.MaxLines`.
47. **(age 0 · value low) `testRelaunchSweepsWhatTheDeadProcessScheduled`
    needs notifications already granted.** It asserts the Allow button is
    absent and says to run the other test first; alphabetical order does
    that when both run.
