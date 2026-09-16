# Clocks, a vector canvas on four targets, and an in-app alarm

**Session:** a48b745b-edc9-42d8-a559-401545367bf0
**Date:** 2026-09-16 10:32 (follows "next-list-githooks-census-row-floor-and-two-a11y-fixes")
**Branch:** master (d710f91 → this commit)

## 1. The asks

1. "Add analog and digital clock components, possibly with an alarm
   sub-component" and "charts — what are my options? Can we do something decent
   from scratch?"
2. After the options answer: "First create a plan doc under ai_docs/plans/, and
   yes, start with clocks and core.Canvas; in-app alarm first."

## 2. The options answer (for the record)

Charts were weighed four ways:

- **A. Charts from flex boxes.** Bars and gauges work; lines, pies and scatter
  plots do not.
- **B. A vector primitive in core** (chosen).
- **C. Rasterise charts in Go and show them with `Image`.** Static, no dark
  mode, no accessibility.
- **D. A JS charting library in a webview.** Web only.

The alarm had three levels: in-app, a scheduled local notification, and the
system alarm APIs (iOS AlarmKit, Android `AlarmClock`). The user chose in-app
first.

## 3. Plan doc

`ai_docs/plans/clocks-canvas-charts.md` has five tiers:

- **A:** clocks
- **B:** `core.Canvas`
- **C:** in-app alarm
- **D:** charts
- **E:** scheduled or system alarms

Its Status block now records what landed and how that differs from the plan.

## 4. Tier A: clocks (no renderer changes)

- **`hooks/now.go`:** `UseNow(ctx, every)`.
  - A timer re-aims at the next wall-clock boundary on every tick.
    `UseInterval`'s ticker keeps the phase it had at mount, which can be up to
    999 ms stale next to the status bar.
  - Restarts after a close and re-mount, like `useInterval`.
- **`comps/clock.go`:** `DigitalClock` and `AnalogClock`.
  - Both take a `time.Time` and hold no hooks, the same split as `Compass` and
    `UseHeading`.
  - Each announces once, as `RoleImg` with a sentence label.
  - The analog face is a `ZStack` of dial-sized layers turned with `Rotate`.
    A layer turns about its own centre, which is the dial's centre, so a stick
    in its top half pivots there. This is the workaround `Style.Rotate`'s doc
    prescribes.
  - Angles are unwrapped across the local day (the second hand reaches at most
    518 400°), so `Smooth`'s Transition never sweeps backwards.
  - The pass in the first second of the day omits the Transition, so the hands
    jump at midnight.
- **Dropped from the plan:** the DigitalClock width floor.
- **Checked:** by eye through htmlout in Chrome.

## 5. Tier B: `core.Canvas`

### API

- `core/canvas.go` adds `Canvas(w, h, shapes, props...)`, `Shape`,
  `CanvasScale` (`CanvasFit`/`CanvasStretch`, a BehaviorProp that writes the
  `scale` prop), `LineCap` and `LineJoin`.
- The path builder has `MoveTo`, `LineTo`, `QuadTo`, `CubicTo`, `Arc` and
  `Close`.
- Helpers: `Line`, `Polyline`, `Rect`, `Circle`, `Sector`.
- `CanvasMapping` is the Go authority for how a viewBox maps onto a box.

### Wire and node shape

- **Four opcodes on the wire** (0 M, 1 L, 2 C, 3 Z), as one flat `[]float64`.
  - Quadratics are raised to cubics exactly.
  - Arcs become cubic segments of at most 90°, with k = 4/3·tan(θ/4).
  - Angles are degrees clockwise from three o'clock, the same sense as `Rotate`.
- **Coordinates** are rounded to `4 − floor(log10(max(w,h)))` decimal places.
- **Node shape** follows TextGrid: a `Canvas` with one `CanvasShape` child per
  shape, so the diff patches only the shapes that changed.
- **Accessibility:** an unlabelled canvas is hidden; a labelled one gets
  `RoleImg`.
- **Strokes** are in layout units and never scaled.
- **Sizing:** with no Height, the canvas takes the viewBox's aspect ratio; with
  no Width, it fills its parent.

### Renderers

| Target | Where | How |
|---|---|---|
| htmlout | `htmlout/canvas.go`; `tag.go` rows | `<svg viewBox preserveAspectRatio>` + `<path>`. The chassis (`display:block; width:100%; overflow:visible; aspect-ratio`) goes ahead of the author's style. `fill="none"` is written because SVG's default fill is black. `vector-effect=non-scaling-stroke`. Exports `PathData` and `CanvasShapeAttrs`. |
| WASM runtime | canvas section of `grmob-runtime.js` | `createElementNS` for `svg`/`path`. `applyCanvasProps` runs on create and on update-props, and removes attributes a shape no longer carries. The `styleFromGrMob` chassis has no aspect-ratio (a prop-driven value, written as `el.style.aspectRatio`). |
| Compose | `GrMobCanvas.kt`, `GrMobCanvasGeometry.kt` | Foundation `Canvas`. Shape props are read in the draw phase. Points are mapped before they reach the `Path`; strokes are in dp. `fillMaxWidth`/`aspectRatio` only when Width/Height are unset. An odd dash list is doubled. |
| SwiftUI | `GrMobCanvas` in `Renderer.swift`, `GrMobCanvasGeometry.swift` | `Canvas {}`. `.aspectRatio` is applied **conditionally**: `aspectRatio(nil, .fit)` means "the child's ideal ratio", which would squash a canvas that has a Height. |

Both geometry files import nothing from Compose or SwiftUI, so the harnesses can
run them.

### Verification

- **core:** arc points on the radius within 3e-4, clockwise on screen, the
  quad→cubic curve identical, `CanvasMapping`.
- **htmlout:** golden SVG attributes; the decoded (`[]any`) form of props draws
  the same as the typed form.
- **wasm/verify:**
  - `gen.go` generates `canvasCases`.
  - `canvas_test.mjs` compares the runtime's attributes to htmlout's, plus two
    patch tests.
  - A deliberate break in `grmob-runtime.js` made the test fail, which confirms
    it can.
  - `dom.mjs` gained `createElementNS`.
- **internal/canvasfixture:** Go answers from `CanvasMapping` and `Decode`,
  checked by:
  - `android/verify` (`Harness.kt`, `gen.go`, `run.sh` SRC): "OK: 7 canvas
    drawings…"
  - `ios/verify` (`canvas.swift`, `main.swift`, `gen.go`, `run.sh`): same.
- **Compile checks:** `android/verify/sources.sh` compiled `GrMobCanvas.kt`
  against the Compose classpath. iOS type-checks, survives `-O -wmo`, and
  type-checks against the iOS SDK.
- **`nodetypes_test.go`:** dispatch arms for both types in both natives.
- **Live Chrome:**
  - htmlout export: pie, donut, a round-capped gauge arc, a stretched area chart.
  - WASM runtime: the transcript's trees.

## 6. Tier C: in-app alarm

### `alarm/`, pure time math

- `Alarm{ID, Hour, Minute, Label, Days, Enabled}`.
- Methods: `Next`, `Due(from, to)` over the interval (from, to], `Once`,
  `Valid`, `DaysLabel`, `TimeLabel`.
- **DST finding:** Go's `time.Date` resolved a nonexistent 02:30 (New York, 8
  March 2026) to 01:30 EST, an hour early. `Alarm.on` recomputes as midnight
  plus elapsed time, only when the wall clock doesn't match, so it rings at
  03:30. Tests cover spring-forward and fall-back (05:00 stays 05:00; the
  repeated 01:30 rings once).

### `hooks/alarms.go`

- `UseAlarms(ctx, alarms, AlarmOptions)` returns `AlarmRinger`, with
  `Ringing`, `Snoozed`, `Snooze` and `Dismiss`.
- Options: `OnRing`, `Sound`, `Haptics`, `Snooze` (default 9m), `RingFor`
  (default 10m).
- **Checking:**
  - One wall-aligned check a second over the interval (last, now].
  - Snoozes are checked before alarms.
  - Alarms due while one rings are queued.
  - A ringing alarm times out after `RingFor`.
  - A render is requested only when ringing state changes.
- **Ringing:**
  - Sound goes through `core.AudioLoad`, and `AudioPlay` restarts it when it
    ends (the player does not loop).
  - Haptic pulse every other second.
  - `check(now)` is split out from the goroutine so tests can drive it with
    chosen instants.
- **Tests:** tests of the ringer pass under `-race`.

### `comps/alarm.go`

- **`AlarmRow`:** a SwitchRow with the time as title and "Label · Days" as
  subtitle.
- **`AlarmRinging`:**
  - `RoleAlert` panel with the label "Alarm, 6:30 AM, Wake up".
  - Snooze is filled and Dismiss outlined, so the accidental press is the cheap
    one.

## 7. Tutorial lesson 4.19 "Clocks, drawing and alarms"

- **Demos:**
  - live clocks with a 24-hour toggle
  - a stretched area and line chart with a Shift button
  - a donut built from `Sector`
  - alarm rows, a "Ring at the next minute" button, and the ringing panel or
    snooze caption
- **Placement:** appended to chapter 4, so no existing lesson ID moved.
- **Run live** in the WASM build (served from the scratchpad; tracked
  `wasm/main.wasm` untouched):
  - The alarm rang at 10:08:00.
  - `OnRing` switched the one-time alarm off.
  - `AlarmRinging` showed.
  - Snooze produced "snoozed until 10:09:24".
  - I didn't wait for the snooze to come back; a unit test covers it.
- **Hint shortened** to "Set one, then wait on this screen.": the longer one
  wrapped the TRY IT badge onto two lines.

## 8. Knock-on edits the repo's tests required

- **Lesson count 57 → 58** in `README.md` (twice), `core/deeplink.go`,
  `wasm/index.html`, `docs/tutorial-interactive.md`,
  `internal/shotclaims/shotclaims.go` and its test, and
  `examples/tutorial/app_test.go` / `screenshot_test.go`. Re-took
  `docs/images/tutorial-contents.png` with `wasm/shots/shoot.sh
  tutorial-contents`. The hero composite does not include it.
- **Tracked-Go-file count 518 → 534** in `wasm/verify/repowalks_test.go` (3
  places) and `timings_test.go` (2). `sharedparse_test.go` fails when these
  drift, so the previous doc's "noted, not enforced" is out of date. The
  sentences describe when a timing was taken, so the new number is less true
  as history, but the test demands the current count.
- **`internal/apidoc/packages.go`:**
  - `canvas.go` goes under core "Controls".
  - `clock.go` and `alarm.go` go under comps "Data display".
  - New `alarm` package entry.
  - `mkdocs.yml` nav gained `api/alarm.md`.
  - `go run ./internal/apidoc/gen` regenerated 31 pages.

## 9. Harness notes

- **Chrome MCP can't open `file://`:** served the scratchpad with
  `python3 -m http.server`, then killed that one PID by its port.
- **Chrome renders `<input switch>` as a checkbox.** This predates the
  session.
- **Two coordinate clicks from scaled screenshots missed.** A JS
  `button.click()` was reliable. The first miss looked at first like a bug
  (the rows read "all off"); they were the 24-hour toggle and the two preset
  alarms.
- **`javascript_tool` times out at 45 s:** keep awaits at or under 30 s.
- **`go build .` in `wasm/verify`** left a `verify` binary; deleted.

## 10. Verification at the end

- `gofmt -l` is clean.
- `go vet ./...` is clean.
- `go test ./...` passes.
- `-race` passes on `hooks`, `alarm`, `comps` and `core`.
- WASM Node suite: exit 0.
- `android/verify/run.sh`: all OK, including sources.sh.
- `ios/verify/run.sh`: all OK.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

Closed or changed this session (previous numbering): 2 is now enforced by
`sharedparse_test.go`, so it is no longer a non-goal. Nothing else from the
previous list was worked.

1. **(age ≥17 · value high) Lessons on hardware, and checks still open.**
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
2. **(age ≥17 · non-goal) Rename `docs/components.md` to `comps.md`.**
3. **(age ≥17 · non-goal) Rewrite `components` in the older plans.**
4. **(age ≥17 · non-goal) Trim the copied Android shell's permissions.**
5. **(age ≥17 · non-goal) Replace the iOS usage strings further.**
6. **(age ≥17 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
7. **(age 16 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android.
8. **(age 16 · non-goal) An AppBar outside the Navigator** with a custom
   `OnBack` is outranked by the Navigator's pop on Android.
9. **(age 15 · non-goal) Forward does not re-open a screen left by browser
   back.**
10. **(age 14 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose.
11. **(age 12 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain. Not profiled.
12. **(age 12 · non-goal) MaxWidth with a growing sibling on the natives.**
13. **(age 12 · non-goal) The typed-hash fold's `history.length` fallback.**
14. **(age 12 · non-goal) A page's own `pushState` while a claim is on
    screen.**
15. **(age 12 · non-goal) A Drawer's shut panel is composed on the natives.**
16. **(age 10 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android.
17. **(age 4 · value low) Compose's today `stateDescription` not heard** under
    TalkBack on the emulator.
18. **(age 4 · value medium) F-keys on iOS unverified; key delivery to a
    simulator is unreliable.**
19. **(age 4 · value low) Page-global chords on iOS are verified once.**
20. **(age 3 · non-goal) GameController cannot take a key.**
21. **(age 3 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
22. **(age 2 · value low) The Compose Row hug is unswept in `comps` widgets'
    own internal Rows.**
23. **(age 1 · value medium) The min-content floor has not been seen on a
    device.** Lessons 1.1 and 8.2 are the screens that should change.
24. **(age 1 · value low) `content ÷ N%` arithmetic unwatched** for a Row child
    with a percentage MaxWidth.
25. **(age 1 · value low) A plain horizontal strip in a Row now hugs.** Unseen.
26. **(age 1 · value low) The iOS chord gate is unheard.**
27. **(age 1 · value low) Merging a labelled node is unheard under TalkBack.**
28. **(age 1 · value medium) `.claude/settings.json` cannot be edited from a
    session.** Worth an upstream report.
29. **(age 1 · value low) `android/verify/sources.sh` needs `ANDROID_HOME` and
    a filled gradle cache**, and SKIP reads like a pass. It compiled for real
    this session.
30. **(age 0 · value high) Tier D: charts on `core.Canvas`.**
    - `Sparkline`, `LineChart`/`AreaChart`, `BarChart`, `DonutChart`/`PieChart`,
      `Gauge`.
    - Go-side scales and "nice" ticks; axes and legends as Text; series colours
      from theme roles; one spoken summary per chart.
    - `Sector` and `Polyline` are already in core for this.
31. **(age 0 · value high) `core.Canvas` and lesson 4.19 have not run on an
    Android emulator or iOS simulator.**
    - Compiled and held to Go's geometry on a JVM and macOS only.
    - Look for:
      - aspect-ratio sizing with and without Width/Height
      - stroke width under `CanvasStretch`
      - an update-props patch redrawing (the Compose draw-phase read, the
        SwiftUI observation of `children.map(\.props)`)
      - dashes
      - the analog clock's hands pivoting at the centre and `Smooth`
        animating forward
32. **(age 0 · value medium) Tier E: an alarm that rings with the app
    closed.**
    - `LocalNotification.At` (UNCalendarNotificationTrigger / AlarmManager;
      Android 12+ needs `SCHEDULE_EXACT_ALARM`).
    - Or iOS AlarmKit / Android `AlarmClock`.
    - Browsers cannot schedule a notification.
33. **(age 0 · value low) Alarm sound and haptics are unheard.**
    - The lesson passes no `Sound`.
    - Ringing replaces whatever core's single audio player was playing and
      doesn't resume it (documented).
    - Haptic pulses were not felt on a device.
34. **(age 0 · value low) A snoozed alarm coming back was not watched live**
    (unit-tested only).
35. **(age 0 · non-goal) `DigitalClock` digits can shift width by a pixel.**
    There's no font family in `core.Style`; centred to hide it.
36. **(age 0 · non-goal) `AnalogClock{Smooth}` spins back once if the tick at
    00:00:00 is skipped** (an app suspended over midnight).
37. **(age 0 · non-goal) Canvas v1 omits** text inside the drawing, gradients,
    clipping, per-shape hit-testing, and the even-odd fill rule.
38. **(age 0 · value low) `demoPanel`'s TRY IT badge wraps under a long
    hint.** Worked around in 4.19 by shortening the hint; other lessons'
    hints were not checked.
