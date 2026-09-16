# Zero-basis floor fix, area fades, square scatter dots, and a look at Android

**Session:** d575a54d-8762-454f-8fed-5c871cdd69d1
**Date:** 2026-09-16 16:43 (follows "small-fixes-stroke-gradients-bar-values-alarm-groups")
**Branch:** master (81e9961 → 48f3d17 → 879e61c → this commit)

## 1. The ask

The session was loaded with `/sl`. Asked what remained on the Next list, the
answer grouped it. The user said: "Check 50, 45 and 53 on the simulator. Then
follow up by the code work items. Do all in reasonable batches committing and
pushing in-between."

- **Simulator items:** 50 (the iOS zero basis on Calendar and StatTile), 45
  (the 4.19 gradient row), 53 (stroke gradients on the natives).
- **Code items:** 41, 42, 52, 54, 22, 36, 39, 44 (previous numbering).

## 2. Batch 1 (48f3d17): simulator checks, and a bug they found

### A stroke-gradient row in 4.19 (item 53)

- No lesson used `Shape.StrokeGradient`, so 4.19 gained a row after the
  FillGradient row:
  - a dashed zigzag with round caps and joins, fading `hue(0)` → `hue(2)`
    (blue → aqua);
  - a ring whose outline takes an off-centre radial gradient, `hue(1)` →
    `hue(7)` (orange → red).
- Accessibility label: "A dashed zigzag fading blue to aqua beside a ring
  shading orange to red".

### Item 50: the calendar was content-biased on iOS

- **Symptom:** the first screenshot of 4.9 showed uneven columns. The row
  "29 30 31 1 2 3 4" was visibly spread, and today's ring hugged "11".
  Weekday captions were uneven too.
- **Cause:** last session's `baseMains` returned
  `max(padding, automatic, floors[i])` for a zero-basis child. The automatic
  minimum of a day cell is its numeral's width, so the base was the content
  width again and the grow split only the leftover. The bar-chart spacers
  last session checked have no content (min 0), which is why that check
  passed.
- **Fix (CSS 9.7, grow half with min violations):**
  - `baseMains`: a zero-basis child's base is its padding, nothing more.
  - `minMains`: a zero-basis child's min is passed whole (not clamped to its
    base) when finite.
  - `GrMobFlexSolver.resolve`: computes `hypothetical = max(base, min)` and
    decides grow-or-shrink on those. Shrink starts from the hypothetical
    sizes; the "nothing grew" arm returns them.
  - New `GrMobFlexSolver.grow`: share free space by weight on top of the
    BASES, raise any child below its minimum to it and freeze it, repeat.
    Children with no weight are frozen at their hypothetical size.
  - Every other caller passes mins ≤ bases, so nothing else changes.
- **Docs:** the `GrMobFlexZeroBasis` comment, the `baseMains` and `minMains`
  docs, and `TestIOSFlexHonoursAZeroBasis` now describe the new shape. The pin
  rejects the old `return max(padding, automatic, floors[i])`.
- **Tests (`ios/verify/flex.swift`):** six grow-with-minimum cases.
  - Seven zero bases, weight 1 each, mins 8–18, in 350 give 50 each.
  - A share short of a floor: `[70, 30]`.
  - A padded zero basis grows from its padding: `[65, 55]`. (First written
    as `[60, 60]`, which was the test's mistake, not the solver's.)
  - A raised child lands on its floor: `[80, 40]`.
  - Minima that overflow overflow: `[90, 60]`.
  - A zero-basis child with no weight is its minimum.

### New UI test: `TutorialZeroBasisAndGradientsUITests`

- `testCalendarDayColumnsAreEqual`: centres of March 8–14 an equal stride
  apart, ±0.5 pt. Widths could not be used: a cell's accessibility frame is
  its numeral (the first run read 10.7 pt for "8" and 18 pt for "10").
- `testStatTilesShareTheRow`: solves for the tile padding from the two label
  x positions and the screen width; it must come out between 0 and 24. A weak
  assertion; the screenshot carries the check. Measured on the screenshot,
  "Filter" starts at 209 pt, where equal tiles put it.
- `testGradientRowsDraw`: both rows exist, then a screenshot.
- Screenshots go to `GRMOB_SHOT_DIR` (note: SHOT, singular, as in
  TutorialNativeFloorsUITests; TutorialChartsUITests uses `GRMOB_SHOTS_DIR`).

### Seen on the iOS simulator

- 4.9: all columns aligned, today's ring centred on 11.
- 4.7: tiles equal.
- 4.19: dashes kept their gaps, the colour runs the right way on both shapes.
- Re-run after the solver change, all passing: `TutorialChartsUITests`,
  `TutorialNativeFloorsUITests` (3), `TutorialFooterStripUITests`,
  `TutorialScrollUITests`, `TutorialClockLocalTimeUITests`. Chart screenshots
  unchanged (values at tips, cut category labels, stacked totals).

## 3. Batch 2 (879e61c): chart code items

### Item 41: area fades (`comps/line_chart.go`)

- New `areaShape(area, color, values, scale)`. An unstacked area is a
  vertical `LinearGradientFill` from the series' extreme point (furthest from
  the base line, so a negative series fades from its trough) to the base,
  `areaFadeTop` `4D` (30%) → `areaFadeBase` `0A` (4%), same hue.
- **Falls back to the old flat `33`** for a colour `withAlpha` cannot tint
  (already has alpha, e.g. the palette's `99` repeats) and for a series with
  no finite value off the base.
- Stacked bands keep flat `66`.
- The `Area` field doc says so.
- **Test:** `TestAreaFadesTowardTheBaseLine` (geometry, stops, the negative
  case, both fallbacks, and the rendered canvas: one gradient unstacked, one
  flat `66` stacked).
- **Seen:** iOS simulator (4.20's second chart) and the Android emulator.
- **Sparkline's `Area`** is still a flat tint (not changed).

### Item 42: square scatter dots past three series (`comps/scatter_chart.go`)

- `scatterSquare(i, n)`: `n > 3 && i%2 == 1`. Those series stroke with
  `CapSquare` along a segment `2 × scatterSquareHalf` (0.01 viewBox units)
  long.
- **Why not zero-length:** a local SwiftUI `ImageRenderer` check drew nothing
  for a zero-length subpath with square caps (round caps drew a dot). With a
  0.01 segment the square is solid to its corners.
- New `legendWithMarks(t, names, colors, squares)` and `markSwatch`: a
  circle swatch (radius 5) for round dots, a sharp square (radius 0) for
  square ones. `legend` is `legendWithMarks(…, nil)`, unchanged for lines and
  bars.
- The "Square dots past three series" section is on the type doc;
  `DefaultChartColors`' doc names the cue.
- **Test:** `TestScatterSquaresPastThreeSeries` (caps for 3 and 4 series,
  square subpaths have length, swatch radii `[5 0 5 0]`).
- **Unseen on any device:** the lesson's scatter has one series.

### Item 52: `barValueRoom(text)` (`comps/bar_chart.go`)

- The flat `barValueRoom = 0.25` became a function:
  `(runes × barValueEm × chartLabelSize + barValueGap) / barPlotGuess`,
  clamped to `[barValueRoomMin, barValueRoomMax]` = `[0.08, 0.5]`.
  - `barPlotGuess` 220 px, `barValueEm` 0.6.
  - `bandValueLayer`'s `gap` is now `barValueGap`.
- "85" → 0.08, "$1.5k" → 0.17, "$1,234,567" → 0.32.
- **Test:** `TestBarValueRoomFollowsTheLabel` (ordering, floor, cap, runes
  not bytes, and 85-of-100 outside for "85" but inside for a 10-rune label).
  The existing `TestHorizontalValuesFollowOrEnterTheBar` still passes.
- **Seen:** on iOS "$1.2k" of $1.5k now sits just past its bar and fits (it
  was inside before). Android shows the same.

## 4. Item 54: built, seen to be wrong, reverted

- **What was built:**
  - `GrMobImageSizes`, an `@Observable` store of natural sizes by `src`.
  - `GrMobRemoteImage`, replacing `AsyncImage`: `URLSession` fetch, ImageIO
    decode with EXIF transform off the main actor, records the size.
  - `GrMobMinContent.imageFloor`: `min(declared, height × aspect)`, or the
    natural width with no px height.
  - Verify cases. `ios/verify` passed.
- **How it was seen:** `examples/fintechapp` (the only app with an image) is
  a `main` package, which gomobile cannot bind. So two scratch rows went into
  4.19 (a dummyimage.com 50×50 and 400×100 in a 110×40 px box beside an
  unbreakable word) with a scratch UI test. Both were removed afterwards.
- **Result:** images loaded and drew. The 50×50 row's floor dropped to 40,
  the row placed the text at 40 + gap, but the image box still drew 110 pt
  wide, so the word ran UNDER the image. The committed code (floor 110) wraps
  the word beside the box, no overlap.
- **Conclusion:** a px-width box on iOS does not accept a narrower proposal,
  so a floor below the declared width only produces overlap (the Canvas bug
  the floor was written for). All item-54 Swift changes were reverted; nothing
  of it is committed.

### Found along the way: black letterbox on iOS Images

- A "fit" Image (the default mode) with no background paints BLACK where the
  bitmap does not fill its box.
- Present with the committed `AsyncImage` code too, and for JPEG as well as
  colormap PNG.
- `core.BackgroundColor("#FFFFFF")` on the image makes the letterbox white.
- Not traced: nothing in `grMobBox`'s chain (`.background(.clear)`,
  `grMobClip`, `grMobBorder`) paints black.

## 5. Items closed without code

- **39 (`MaxLines` on non-Text natives):** now a non-goal. No `comps` widget
  sets it on anything but Text, and carrying it to containers would differ
  from CSS (SwiftUI's `lineLimit` caps each Text separately; CSS clamps the
  container's lines).
- **44 (gradient interpolation across targets):** premultiplied and straight
  interpolation agree whenever every stop has the same hue (only alpha
  varies). Every shipped gradient is either opaque stops (4.19) or a
  same-hue alpha fade (the area fill), so no shipped drawing can differ. The
  risk remains only for an app fading between hues and alphas at once,
  which `core.Gradient`'s doc already warns against.
- **36 (Compose proportional shrink):** not attempted. It means replacing
  Compose's Row measure policy for every Row on Android; put to the user as
  their call.

## 6. Android emulator: up, and looked at

- The emulator (`Medium_Phone_API_36.1`, emulator-5554) was running.
  `android/build.sh ./examples/tutorial` then
  `./gradlew installDebug --offline` installed.
- **Seen:**
  - 4.9: calendar columns aligned.
  - 4.7: StatTiles equal.
  - 4.19: dashed zigzag blue → aqua and ring orange → red (full-resolution
    crop).
  - 4.20: area fade, horizontal values past tips, stacked totals, scatter,
    donut, gauge.
- This closes the item-22 sweep at phone size (comps widgets in their
  tutorial demos lay out as on iOS and the web) and most of item 49.

## 7. Verification

- `gofmt -l` clean, `go vet` clean, `go test ./...` passes, `docs/api/*`
  regenerated.
- `ios/verify/run.sh`, `wasm/verify/run.sh` and `android/verify/run.sh` all
  OK.
- iOS UI tests listed in section 2 pass.

## 8. Harness notes

- **Android screenshots of a lesson:** a deep link to a running app keeps
  its scroll, so force-stop first:
  `adb shell am force-stop com.grmob.app; adb shell am start -a android.intent.action.VIEW -d grmob://lesson/4.19`,
  then `adb shell input swipe 540 1600 540 1000 400` per step and
  `adb exec-out screencap -p > shot.png`. On 4.19, 6 short swipes reach the
  drawing rows; on 4.9, 2 long ones reach the grid.
- **Montage:** `montage a.png b.png -tile 2x1 -geometry 600x1333+6+0 m.png`
  (ImageMagick is installed) keeps a multi-shot look to one image read.
- **Xcode project:** a new UI test file needs `xcodegen generate` in `ios/`
  (the `.xcodeproj` is git-ignored).
- **Swift checks without a simulator:** a `swiftc -parse-as-library` file
  with `@main` and `ImageRenderer` answers "does SwiftUI draw X" in seconds.
  Top-level `MainActor.assumeIsolated` fails to compile under
  `-parse-as-library`.
- **Python edits into Swift:** inserting before a function can split it from
  its `@ViewBuilder` attribute (the error is "branches have mismatching
  types"); check what precedes the anchor.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here). *value*: **high** = worked around
today or a second consumer has arrived; **medium** = blocks one named thing
or is a visible defect; **low** = nobody has hit the gap yet. Sorted by age,
oldest first; new items last.

**Closed this session** (previous numbering):

- **22:** comps widgets' internal Rows seen on Android at phone size (4.7,
  4.9, 4.19, 4.20).
- **39:** now a non-goal (item 36 below).
- **41:** unstacked areas fade.
- **42:** ScatterChart's shape cue past three series.
- **44:** no shipped gradient can differ (same-hue or opaque stops).
- **45:** the 4.19 gradient row seen on the simulator.
- **47:** the emulator is up and the tutorial reinstalled.
- **50:** seen, found wrong, fixed (grow with minimums).
- **52:** each label's own room (estimate remains, item 49 below).
- **53:** stroke gradients seen on both natives.

1. **(age ≥24 · value high) Lessons on hardware, and checks still open.**
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
2. **(age ≥24 · non-goal) Rename `docs/components.md` to `comps.md`.**
3. **(age ≥24 · non-goal) Rewrite `components` in the older plans.**
4. **(age ≥24 · non-goal) Trim the copied Android shell's permissions.**
5. **(age ≥24 · non-goal) Replace the iOS usage strings further.**
6. **(age ≥24 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
7. **(age 23 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android.
8. **(age 23 · non-goal) An AppBar outside the Navigator** with a custom
   `OnBack` is outranked by the Navigator's pop on Android.
9. **(age 22 · non-goal) Forward does not re-open a screen left by browser
   back.**
10. **(age 21 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose.
11. **(age 19 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain. Not profiled.
12. **(age 19 · non-goal) MaxWidth with a growing sibling on the natives.**
13. **(age 19 · non-goal) The typed-hash fold's `history.length` fallback.**
14. **(age 19 · non-goal) A page's own `pushState` while a claim is on
    screen.**
15. **(age 19 · non-goal) A Drawer's shut panel is composed on the natives.**
16. **(age 17 · non-goal) A List with no Height in a scrolled page is not
    lazy** on iOS or Android.
17. **(age 11 · value low) Compose's today `stateDescription` not heard**
    under TalkBack on the emulator.
18. **(age 11 · value medium) F-keys on iOS unverified; key delivery to a
    simulator is unreliable.**
19. **(age 11 · value low) Page-global chords on iOS are verified once.**
20. **(age 10 · non-goal) GameController cannot take a key.**
21. **(age 10 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
22. **(age 8 · value low) The iOS chord gate is unheard.**
23. **(age 8 · value low) Merging a labelled node is unheard under
    TalkBack.**
24. **(age 8 · value medium) `.claude/settings.json` cannot be edited from a
    session.** Worth an upstream report.
25. **(age 7 · value low) Alarm sound and haptics are unheard.**
    - The lesson passes no `Sound`.
    - Ringing replaces core's single audio player and doesn't resume it
      (documented).
    - Haptic pulses were not felt on a device.
    - The Notify banner's default sound is unheard.
26. **(age 7 · non-goal) `DigitalClock` digits can shift width by a pixel.**
    There's no font family in `core.Style`.
27. **(age 7 · non-goal) `AnalogClock{Smooth}` spins back once if the tick at
    00:00:00 is skipped.**
28. **(age 7 · value low) Canvas still omits** text inside the drawing,
    clipping and per-shape hit-testing. Gradient fills, gradient strokes and
    the even-odd rule are in.
29. **(age 6 · value low) A chart's hidden data table** was not built.
    - The one-sentence summary is all a screen reader gets.
    - A BarChart with more than 8 categories only gives the low and high.
    - It needs a screen-reader-only primitive core doesn't have: an API
      decision.
30. **(age 6 · non-goal) A 180° `Gauge` leaves its bottom half empty.**
31. **(age 6 · value low) Chart summaries are English**, like other widgets'
    spoken strings. `AccessibilityLabel` overrides them.
32. **(age 5 · value low) A Notify alarm is a banner, not a ringing screen.**
    - There's no full-screen intent, no looping sound, and no Snooze action.
    - iOS AlarmKit and Android `AlarmClock` / full-screen intents are the
      route to a real system alarm. Large, and only a device will tell.
33. **(age 5 · value low) `mobile.SetTimeZone` runs once at startup.** A fix
    needs core to hold the current location: an API decision.
34. **(age 5 · value low) The web's scheduled notification and its sweep are
    unseen in a real browser.** Chrome's permission prompt needs a person to
    grant it once.
35. **(age 4 · value low · user's decision) Compose Rows don't shrink
    children in proportion.** Documented in `Renderer.kt` (pinMainAxis) and
    pinned by `internal/pinfixture`. A fix replaces Compose's Row measure
    policy for every Row on Android (a flex solver like iOS's); not attempted,
    put to the user.
36. **(age 3 · non-goal) `MaxLines` on the natives applies to Text only.**
    Documented on `core.MaxLines`. No comps widget sets it elsewhere, and
    SwiftUI's inherited `lineLimit` would cap each Text rather than clamp the
    container as CSS does.
37. **(age 3 · non-goal) A stacked chart counts NaN as 0 and stacks negatives
    through the band below.** Documented under "Stacked".
38. **(age 3 · value low) The double-post claim (old item 46) is
    unreproduced.**
39. **(age 3 · value low) `testRelaunchSweepsWhatTheDeadProcessScheduled`
    needs notifications already granted.**
40. **(age 2 · value low) `DefaultDarkChartColors` has no bundled consumer.**
    No dark theme ships. It was checked by the skill's published validation,
    not re-run here.
41. **(age 2 · value low) `TutorialChartsUITests` failed once, reason not
    captured.** It passed on all three runs this session.
42. **(age 2 · non-goal) `core.LinearGradient` (a CSS string helper) is
    unused and sits beside the canvas gradient names.** Kept per CLAUDE.md's
    no-removal rule.
43. **(age 1 · value medium) Last session's alarm changes are unseen on
    Android:** the kept banners after a force stop, and the exact-alarm
    re-check. (Stroke gradients and bar values were seen this session.)
44. **(age 1 · value low) A zero basis is honoured on iOS only when the main
    extent is definite.** An ideal-size query keeps content bases, and a
    Column child with the `.infinity` height verdict keeps its measured base.
    Both are deliberate (see `GrMobFlexZeroBasis`).
45. **(age 1 · value medium · blocked) The iOS Image floor runs high for an
    image narrower in proportion than its box.**
    - A natural-size store and a floor matching Chrome were built and
      reverted this session.
    - On the simulator a floor below the declared width made the row overlap
      the text under the image: a px-width box does not narrow when offered
      less.
    - Unblocking it needs a px-width box that can shrink on iOS (CSS lets an
      `<img>` flex item shrink to its content suggestion), which touches
      `grMobDimension` for every node.
46. **(age 1 · value low) Kept banners are unseen on a device.** The relaunch
    UI test (`TutorialAlarmNotifyUITests`) was not re-run. An Android alarm
    Doze delayed past its time is no longer cancelled on return, so its
    banner may post after `OnRing` already reported it (documented in the
    hook).
47. **(age 1 · non-goal) Renaming a `NotifyGroup` strands what the old name
    scheduled** until the OS fires it. Documented; groups are meant to be
    constants.
48. **(age 0 · value medium) An iOS Image with no background shows a black
    letterbox.** A "fit" image narrower or shorter in proportion than its box
    paints black in the rest. Seen with `AsyncImage` (committed) for PNG and
    JPEG; a `BackgroundColor` hides it. Not traced to a modifier.
49. **(age 0 · value low) `barValueRoom` is still an estimate.** It assumes a
    220 px plot and 0.6 em per rune, so a wide screen puts some labels inside
    that would fit outside, and a narrow plot can still cut one with an
    ellipsis.
50. **(age 0 · value low) Square scatter dots are unseen on any device.** No
    lesson has four series; they rest on a Go test and a SwiftUI render check
    of the 0.01-unit square cap. Compose and Chrome are unchecked.
51. **(age 0 · value low) `testStatTilesShareTheRow` asserts weakly.** It
    solves for a padding in 0–24 pt, which a content-biased split could also
    satisfy; the screenshot was the real check.
52. **(age 0 · value low) Sparkline's `Area` is still a flat tint**, unlike
    LineChart's new fade. Left alone: a sparkline's area runs to its bottom
    edge, not zero, and it has no axis to fade toward.
53. **(age 0 · value low) Comps on Android were seen at phone size only.**
    Tablet widths and landscape are unswept.
