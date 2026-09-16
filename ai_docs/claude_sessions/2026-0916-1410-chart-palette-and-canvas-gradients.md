# A default chart palette and canvas gradient fills

**Session:** c3973f7f-0e58-430e-aba5-a08e0421f6d9
**Date:** 2026-09-16 14:10 (follows "next-list-sweep-maxlines-chart-types-evenodd")
**Branch:** master (af52275 → this commit)

## 1. The ask

"Pick a default theme color set and also let add gradients". Those are two
items from the last Next list:

- **32:** the chart palette had only 3–5 distinct hues per bundled theme.
- **29, in part:** Canvas v1 had no gradients.

## 2. The chart palette

### Choosing it

- The `dataviz` skill's reference categorical palette was taken verbatim:
  eight hues in a fixed order.

  | Slot | Hue | Hex | Contrast on #FFFFFF |
  |---|---|---|---|
  | 1 | blue | `#2A78D6` | 4.42:1 |
  | 2 | orange | `#EB6834` | 3.20:1 |
  | 3 | aqua | `#1BAF7A` | 2.82:1 |
  | 4 | yellow | `#EDA100` | 2.17:1 |
  | 5 | magenta | `#E87BA4` | 2.69:1 |
  | 6 | green | `#008300` | 4.95:1 |
  | 7 | violet | `#4A3AA7` | 8.56:1 |
  | 8 | red | `#E34948` | 3.95:1 |

- **Validator** (`scripts/validate_palette.js`), run against `#FFFFFF`,
  `#F2F2F7`, `#F5F5F5` and `#FFF8E1`:
  - Lightness band and chroma floor pass for all eight.
  - Adjacent pairs, CVD: worst ΔE 9.1 (yellow↔aqua, protan); the target is 8.
  - Adjacent pairs, normal vision: worst ΔE 19.6; the floor is 15.
  - Contrast WARN: aqua, yellow and magenta, plus orange on the two grey
    Surfaces. The chart legends and spoken summaries are the relief.
  - `--pairs all` (scatter): only the first three slots pass.
- The contrast figures were computed by hand with WCAG luminance. The first
  draft's numbers were wrong and were corrected.

### Wiring it

- **`core/theme.go`:**
  - New field `ColorPalette.Chart []string`, the only slice role.
  - `DefaultChartColors()` and `DefaultDarkChartColors()` are functions that
    return fresh slices, not package vars, so no importer can write to them.
  - The resolver `ChartColors()` returns a copy, drops blank entries, and
    falls back to the default list.
  - All three bundled themes set `Chart: DefaultChartColors()`.
- **`comps/chart.go`:** `chartPalette` now reads `ChartColors()`, not
  Primary/Secondary/Warning/Success/Error. It still drops duplicates
  (case-insensitively) and repeats the list as 60% tints.
- **Tests:**
  - `TestBundledThemesSetEveryColorRole` now handles a `[]string` role: each
    entry must be a hex, and an empty list counts as unset.
  - New `TestChartColorsResolve` (copy, fallback, blanks, dark list length).
  - `TestChartPaletteDedupesRolesAndTints` became
    `TestChartPaletteIsTheChartRoleThenTints`, which also asserts no series
    colour is a status role.
  - New `TestChartPaletteDedupesAThemesList`.
- **Lesson 4.19:** the donut uses `hue(0..2)` from `ChartColors()` instead of
  Primary/Success/Warning. `hue` indexes modulo the list's length, so a
  theme with fewer colours cannot panic.

## 3. Canvas gradient fills

### API (`core/canvas.go`)

- `Shape.FillGradient *Gradient`. It wins over `Fill`.
- `Gradient{Kind, X1, Y1, X2, Y2, CX, CY, R, Stops}`, with `GradientStop`
  and `Stop(offset, color)`.
- Constructors are `LinearGradientFill` and `RadialGradientFill`.
  `core.LinearGradient` already existed: an unused CSS-string helper in
  `style_props.go`. It was left in place, per CLAUDE.md.
- **Geometry** is in viewBox units (SVG's userSpaceOnUse), so it stretches
  with the shapes: a radial gradient becomes an ellipse under CanvasStretch.
- **Wire** (`Gradient.wire`), written only for a gradient that paints:
  - `gradient`: `"linear"` | `"radial"`
  - `gradientAt`: `[x1,y1,x2,y2]` or `[cx,cy,r]`, rounded to the canvas's
    places
  - `gradientStops`: offsets clamped to [0,1], non-decreasing, 4 places
  - `gradientColors`: one per offset
  - `fill` and the gradient keys are exclusive. `fillRule` goes with either.
- **Degenerate cases:** no stops paints nothing, so a flat `Fill` still
  stands. One stop, a zero-length line or r ≤ 0 is sent as a flat fill in the
  last colour.
- **Test:** `TestGradientWire`.

### Web: htmlout and the live runtime

- **Finding:** Chrome does not resolve `url(#id)` to a gradient nested inside
  its own `<path>`, though SVG 2 allows it (checked headless: nothing
  painted). A sibling or a `<defs>` works.
- **Design:** one leading `<defs data-grmob-chrome="gradients">` in the
  `<svg>`.
  - The runtime's `chromeOffset` already skips leading chrome, so add-child
    indexes still line up.
  - ids come from `CanvasGradientID(canvasPath, i)` =
    `grmob-<path-with-dashes>-fill-<i>` (tabScope style).
- **htmlout** (`htmlout/canvas.go`):
  - `renderCanvas` writes the `<defs>` first, and only when some shape has a
    gradient.
  - `CanvasShapeAttrs(props, gradientID)` has a new second parameter.
  - New `CanvasGradient(props, id)` returns the tag, attributes and stops.
    Malformed keys give no gradient and `fill="none"`.
  - New `strs` helper.
  - `renderCanvasShape` now takes the path.
  - The element builder lowercases tags (`lineargradient`); an HTML parser
    restores the camel case, and Chrome renders it.
- **Runtime** (`wasm/grmob-runtime.js`):
  - `canvasGradientId`, `canvasGradient` and `canvasShapeAttrs(props, id)`
    restate the Go side.
  - Each shape's props are kept on its element (`__grmobShapeProps`).
  - `syncCanvasGradients(svg)` rebuilds the `<defs>` and every gradient
    shape's fill from the children as they stand. It runs from `renderNode`
    after a Canvas's children are built, and from `syncTouchedCanvases` at
    the end of each patch batch.
  - `applyCanvasShapeAttrs` was factored out.
  - `firstChild` isn't in `dom.mjs`, so the code uses `children[0]`.
- **Tests:**
  - `wasm/verify/gen.go` gained a `Gradients` field and a case covering a
    flat shape first, a stretched linear, an even-odd radial and a degenerate
    gradient.
  - `canvas_test.mjs` compares the `<defs>` against htmlout and has a new
    add/update/lose test.
  - htmlout: `TestCanvasGradientExport`.

### Natives

- **Compose** (`GrMobCanvas.kt`): `canvasGradientBrush(props, vp)` builds a
  `LinearGradientShader` or `RadialGradientShader` in viewBox units with
  `TileMode.Clamp`, sets the viewport's scale-then-translate with
  `setLocalMatrix`, and wraps it in a `ShaderBrush`.
- **SwiftUI** (`Renderer.swift`):
  - `grMobCanvasGradient(props)` decodes the keys into a
    `GrMobCanvasGradientSpec`.
  - `grMobFillGradient` copies the `GraphicsContext`, applies
    `concatenate(toBox)`, maps the path back with `toBox.inverted()`, and
    fills with `.linearGradient` or `.radialGradient`.
  - A zero scale is skipped.
- **Why a transform and not mapped end points:** a stretched radial must
  become an ellipse, and a stretched diagonal's bands must stay parallel in
  viewBox space.
- **Test:** `mobile/verify/gradient_test.go` pins the key names and the
  matrix/transform lines on both natives.

### Lesson 4.19

A new row in the canvas panel: a radial-shaded sphere and a diagonal
three-colour linear fill, captioned
"FillGradient: RadialGradientFill and LinearGradientFill, in the drawing's
own units."

## 4. Seen

- **Throwaway scene** (`internal/zzgrad` for htmlout, `examples/zzgrad` bound
  to both shells; both deleted afterwards):
  - A stretched radial in a rect, with the circle stroked over it.
  - A diagonal linear with a hard stop at 0.5, the diagonal stroked over it.
  - An 8-series stacked BarChart.
- **Chrome, emulator and simulator agree:**
  - The radial is an ellipse that fills the stroked ellipse exactly.
  - The hard stop runs through the centre on the same slant.
  - The ends pad on iOS.
  - The eight hues are distinct.
- **Emulator, tutorial 4.19:** the donut is in blue, orange and aqua; the
  sphere and square render.
- **The 4.19 gradient row was not viewed on the simulator.** The zzgrad
  scene covered the SwiftUI path.

## 5. Verification

- `gofmt -l` clean; `go vet ./...` clean; `go test ./...` passes.
- `node --test wasm/verify/*_test.mjs`: 561 pass, 0 fail. `wasm/verify/run.sh`
  prints its OK lines and exits 0.
- `ios/verify/run.sh`, `android/verify/run.sh` and
  `android/verify/sources.sh`: all OK.
- **`TutorialChartsUITests`:** the first run printed only "TEST FAILED", and
  the filtered output kept no reason. The rerun passed (24 s).
- **Knock-on:**
  - `docs/api/*` regenerated.
  - The tracked-Go-file count went 546 → 547 in `repowalks_test.go` and
    `timings_test.go` (the new `gradient_test.go`).
  - Plan doc `ai_docs/plans/clocks-canvas-charts.md`: the palette note and
    the non-goals now record gradient fills.
- **Device state:**
  - The emulator had the tutorial build reinstalled, then was killed by the
    system for low memory (a background task from an earlier session).
    `adb devices` is empty.
  - The simulator has the tutorial build from the UITest run.

## 6. Harness notes

- **Headless Chrome for SVG questions:** write a tiny HTML file to the
  scratchpad and run
  `"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new --window-size=W,H --screenshot=… file://…`.
- **iOS build to a scratch derived-data dir:**
  `xcodebuild -project ios/GrMobApp.xcodeproj -scheme GrMobApp -destination 'id=<sim>' -derivedDataPath $S/dd build`,
  then `simctl install booted` and `simctl launch booted com.grmob.demo`.
- `ad.sh` was copied from the last session's scratchpad. `lesson 4.19` and
  `find` then scroll work as before.
- **Stale LSP diagnostics** flagged `FillRule` and `core.FillEvenOdd` as
  unknown in `chapter4.go`. The build was fine, so ignore them.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here). *value*: **high** = worked around
today or a second consumer has arrived; **medium** = blocks one named thing
or is a visible defect; **low** = nobody has hit the gap yet. Sorted by age,
oldest first; new items last.

**Closed this session** (previous numbering):

- **32** (chart palette had only 3–5 hues): `ColorPalette.Chart` and
  `DefaultChartColors`, validated.
- **29, in part:** gradient fills. Text, gradient strokes, clipping and
  hit-testing stay open below.

1. **(age ≥22 · value high) Lessons on hardware, and checks still open.**
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
2. **(age ≥22 · non-goal) Rename `docs/components.md` to `comps.md`.**
3. **(age ≥22 · non-goal) Rewrite `components` in the older plans.**
4. **(age ≥22 · non-goal) Trim the copied Android shell's permissions.**
5. **(age ≥22 · non-goal) Replace the iOS usage strings further.**
6. **(age ≥22 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
7. **(age 21 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android.
8. **(age 21 · non-goal) An AppBar outside the Navigator** with a custom
   `OnBack` is outranked by the Navigator's pop on Android.
9. **(age 20 · non-goal) Forward does not re-open a screen left by browser
   back.**
10. **(age 19 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose.
11. **(age 17 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain. Not profiled.
12. **(age 17 · non-goal) MaxWidth with a growing sibling on the natives.**
13. **(age 17 · non-goal) The typed-hash fold's `history.length` fallback.**
14. **(age 17 · non-goal) A page's own `pushState` while a claim is on
    screen.**
15. **(age 17 · non-goal) A Drawer's shut panel is composed on the natives.**
16. **(age 15 · non-goal) A List with no Height in a scrolled page is not
    lazy** on iOS or Android.
17. **(age 9 · value low) Compose's today `stateDescription` not heard** under
    TalkBack on the emulator.
18. **(age 9 · value medium) F-keys on iOS unverified; key delivery to a
    simulator is unreliable.**
19. **(age 9 · value low) Page-global chords on iOS are verified once.**
20. **(age 8 · non-goal) GameController cannot take a key.**
21. **(age 8 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
22. **(age 7 · value low) The Compose Row hug is unswept in `comps` widgets'
    own internal Rows.**
23. **(age 6 · value low) The iOS chord gate is unheard.**
24. **(age 6 · value low) Merging a labelled node is unheard under
    TalkBack.**
25. **(age 6 · value medium) `.claude/settings.json` cannot be edited from a
    session.** Worth an upstream report.
26. **(age 5 · value low) Alarm sound and haptics are unheard.**
    - The lesson passes no `Sound`.
    - Ringing replaces core's single audio player and doesn't resume it
      (documented).
    - Haptic pulses were not felt on a device.
    - The Notify banner's default sound is unheard.
27. **(age 5 · non-goal) `DigitalClock` digits can shift width by a pixel.**
    There's no font family in `core.Style`.
28. **(age 5 · non-goal) `AnalogClock{Smooth}` spins back once if the tick at
    00:00:00 is skipped.**
29. **(age 5 · value low) Canvas still omits** text inside the drawing,
    gradient *strokes*, clipping and per-shape hit-testing. Gradient fills
    and the even-odd rule are in. A gradient stroke is the cheapest next
    piece: the same brush or shading on the stroke call, plus an SVG `stroke`
    reference.
30. **(age 4 · value low) A chart's hidden data table** was not built.
    - The one-sentence summary is all a screen reader gets.
    - A BarChart with more than 8 categories only gives the low and high.
    - It needs a screen-reader-only primitive core doesn't have: an API
      decision.
31. **(age 4 · non-goal) A 180° `Gauge` leaves its bottom half empty.**
32. **(age 4 · value low) Chart summaries are English**, like other widgets'
    spoken strings. `AccessibilityLabel` overrides them.
33. **(age 3 · value low) A Notify alarm is a banner, not a ringing screen.**
    - There's no full-screen intent, no looping sound, and no Snooze action.
    - iOS AlarmKit and Android `AlarmClock` / full-screen intents are the
      route to a real system alarm. Large, and only a device will tell.
34. **(age 3 · value low) `mobile.SetTimeZone` runs once at startup.** A fix
    needs core to hold the current location: an API decision.
35. **(age 3 · value low) The web's scheduled notification and its sweep are
    unseen in a real browser.** Chrome's permission prompt needs a person to
    grant it once.
36. **(age 2 · value low) Compose Rows don't shrink children in
    proportion.** Documented in `Renderer.kt` (pinMainAxis) and pinned by
    `internal/pinfixture`.
37. **(age 2 · value low) Android's exact-alarm grant is only re-checked for
    a running Activity.**
38. **(age 1 · value low) The end tick labels have half an interval of
    width.** A wide `Format` is cut on a horizontal BarChart or a
    ScatterChart; documented on `BarChart`.
39. **(age 1 · value low) Value labels sit along the plot's edge, not at each
    bar's tip.** Documented under "Values".
40. **(age 1 · non-goal) A stacked chart counts NaN as 0 and stacks negatives
    through the band below.** Documented under "Stacked".
41. **(age 1 · value low) Only one `UseAlarms` with Notify per app.** The
    `grmob.alarm.` sweep prefix belongs to the hook. Documented.
42. **(age 1 · value low) The relaunch sweep also takes down shown banners.**
    A missed alarm posted late by `attach` is removed moments later.
43. **(age 1 · value low) The double-post claim (old item 46) is
    unreproduced.**
44. **(age 1 · value low) iOS's floor for replaced elements covers Canvas
    only.** Image and MapView still floor at 0.
45. **(age 1 · value low) `MaxLines` on the natives applies to Text only.**
    Documented on `core.MaxLines`.
46. **(age 1 · value low) `testRelaunchSweepsWhatTheDeadProcessScheduled`
    needs notifications already granted.**
47. **(age 0 · value low) Charts don't use gradients yet.** An area chart's
    fade under its line is the obvious consumer (`LineChart`/`AreaChart`
    fills are still flat `33`/`66` alpha). It is a visual change, so it
    should be put to the user.
48. **(age 0 · value low) Chart palette slots 3–5 are under 3:1 on light
    pages.** Legends and spoken summaries are the relief. A ScatterChart with
    more than three series has pairs the validator's all-pairs check fails.
    Documented on `DefaultChartColors`; nothing in comps caps or warns.
49. **(age 0 · value low) `DefaultDarkChartColors` has no bundled consumer.**
    No dark theme ships. It was checked by the skill's published validation,
    not re-run here.
50. **(age 0 · value low) Gradient colour interpolation is not compared
    pixel-for-pixel across targets.** Premultiplied vs straight alpha can
    differ on a fade to transparent. The docs advise fading to the same hue
    at zero alpha; seen only by eye on opaque stops.
51. **(age 0 · value low) htmlout writes lowercased SVG tag names**
    (`lineargradient`), which an HTML parser fixes. An export consumed as
    XML/XHTML would not paint its gradients. Pre-existing builder behaviour;
    `<svg>` and `<path>` are unaffected because they're already lowercase.
52. **(age 0 · value low) The 4.19 gradient row is unseen on the
    simulator.** SwiftUI gradients were seen only in the throwaway scene.
53. **(age 0 · value low) `TutorialChartsUITests` failed once, reason not
    captured.** The rerun passed; watch for flakiness.
54. **(age 0 · value low) The emulator was killed for low memory.** Restart
    it and reinstall the tutorial (`android/build.sh ./examples/tutorial`,
    then `./gradlew installDebug`) before the next Android check.
55. **(age 0 · non-goal) `core.LinearGradient` (a CSS string helper) is
    unused and sits beside the new canvas gradient names.** Kept per
    CLAUDE.md's no-removal rule.
