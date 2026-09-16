# Small fixes: stroke gradients, bar values at the tips, alarm groups, and a zero flex-basis on iOS

**Session:** afcc8901-a665-4c3b-b834-94b8cc5bd888
**Date:** 2026-09-16 15:57 (follows "chart-palette-and-canvas-gradients")
**Branch:** master (00f0c08 → this commit)

## 1. The ask

The session was loaded with `/sl`. Asked "what remains in the Next list that
we can do now?", the answer grouped the open items, and the user said "Do the
small code fixes". That group was (previous numbering):

- **29** gradient strokes, **51** lowercased SVG gradient tags
- **39** bar values at each bar's tip, **38** end tick labels half an interval wide
- **44** iOS replaced-element floor for Image/MapView
- **42** relaunch sweep takes down shown banners, **37** Android exact-alarm
  re-check needs an Activity, **41** only one `UseAlarms` with Notify per app

All eight are done. A ninth fix came from checking 39 on the simulator: iOS
ignored `core.FlexBasis`.

## 2. Item 29: `Shape.StrokeGradient`

- **Checked first:** headless Chrome resolves a `userSpaceOnUse` gradient on a
  `vector-effect="non-scaling-stroke"` stroke in viewBox space, including a
  radial under `preserveAspectRatio="none"`, and the stroke width stays
  unscaled. That is the split the natives had to match.
- **core (`core/canvas.go`):**
  - New field `Shape.StrokeGradient *Gradient`. It wins over `Stroke`, and a
    degenerate one is sent as the flat stroke it reduces to.
  - `Gradient.wire(props, places, prefix)`. The stroke keys are
    `strokeGradient`, `strokeGradientAt`, `strokeGradientStops` and
    `strokeGradientColors`. The fill keys stay unprefixed.
  - New exported `GradientKey(prefix, name)`.
  - The "Not in v1" doc no longer lists gradient strokes.
- **htmlout (`htmlout/canvas.go`):**
  - New `CanvasStrokeGradientID` (`…-stroke-i`) and `CanvasStrokeGradient`,
    both over a shared `canvasPaintServer(props, prefix, id)`.
  - **API change:** `CanvasShapeAttrs(props, fillID, strokeID)`.
  - `<defs>` order: shape by shape, the fill server then the stroke server.
- **Runtime (`wasm/grmob-runtime.js`):**
  - `canvasGradientId(path, i, kind)`, `canvasGradientKey` and
    `canvasGradient(props, id, prefix)`.
  - `canvasShapeAttrs(props, fillId, strokeId)`.
  - `syncCanvasGradients` pushes both servers.
- **Compose (`GrMobCanvas.kt`):**
  - `canvasGradientBrush(props, vp, prefix = "stroke")`.
  - The `Stroke` style is built once, then drawn with the brush or the colour.
- **SwiftUI (`Renderer.swift`):**
  - `grMobCanvasGradient(props, prefix:)`.
  - A gradient stroke fills `path.strokedPath(strokeStyle)` (box points)
    through `grMobFillGradient`. Stroking inside the transformed context would
    scale the width.
  - The outset calculation counts `strokeGradient`.
- **Tests:**
  - `TestStrokeGradientWire` (core)
  - `TestCanvasStrokeGradientExport` (htmlout)
  - a stroke case in `wasm/verify/gen.go`
  - `canvas_test.mjs`: "a stroke gradient's server follows its slot"
  - `mobile/verify/gradient_test.go` pins for both natives

## 3. Item 51: camelCase SVG tags

- The element builder lowercases tag names (`lowerName` in element v0.7.0,
  the user's library, left alone).
- `renderCanvas` now writes the gradient server's open and close tags through
  `writeSVGOpen` / `b.WriteString`, with attribute values
  `html.EscapeString`'d. `<stop>` still goes through the builder.
- `Pretty()` keeps tag case, and attribute names were never lowercased.
- **Tests:**
  - `TestCanvasGradientExport` now compares case-sensitively.
  - New `TestCanvasGradientExportIsXMLCased` parses the `<svg>` with
    `encoding/xml`.

## 4. Item 38: two thirds of a stride per label (`pointLabels`)

- The old tiling split at midpoints: interior boxes a full stride `s`, end
  boxes `s/2`.
- The new one gives every labelled box `2s/3`: interior boxes are centred with
  half-width `r = s/3`, end boxes extend `2r` inwards, and empty fillers fill
  the gaps.
- A two-label row still meets at the midpoint.
- Used by line/area x labels and by the tick rows of horizontal BarChart and
  ScatterChart.
- **Tests:**
  - `TestPointLabelsTileTheAxis` rewritten: weights `[2 2 1 2 1 2 1]` for 12
    points.
  - New `TestPointLabelsGiveEndTicksTwoThirds`.
- **Docs:** the BarChart doc and the tutorial's `Format` comment ("two thirds
  of an interval") were updated.

## 5. Item 39: bar values at the tips (`comps/bar_chart.go`)

- **The model:** values are a layer of Text over the canvas in a `core.ZStack`.
  - New `barTip` and `barValueTips(n, scale, colors)`, cell for cell with
    `barValueCells`.
  - A stack's tip is its positive total, or its negative total when it has no
    positive part. Its colour is the outermost segment's.
- **Vertical bars:**
  - `valueLayer` places each label with a px spacer: top =
    `1.5·line + tip·h/100 − line` for a positive bar, and no `− line` for a
    negative one.
  - `bottomExtra` adds half a line of foot when any value is negative.
  - `cartesianFrameWithTop` became
    `cartesianFrameWithValues(…, values, bottomExtra)`. The plot is a ZStack of
    height `2·line + h + bottomExtra`, and the y axis keeps its one-line
    spacer.
- **Horizontal bars:**
  - `bandValueLayer`: in each px band, a Row of weighted segments puts the
    label past the tip. The right-hand value column is gone.
  - A bar leaving less than `barValueRoom` (0.25) of the plot past its tip
    carries its value inside, end-aligned, in
    `contrastInk(bar, TextPrimary, Background)`. Negatives mirror this.
- New helper `chartLabelText`.
- **Tests:**
  - `TestBarValueTipsMeetTheBars`
  - `TestVerticalValuesSitAtTheTips` (it parses px with `strconv`, because
    Sscanf's `%g` reads the "p" of "px" as a hex exponent)
  - `TestHorizontalValuesFollowOrEnterTheBar`
- **Seen:**
  - Headless Chrome, on a throwaway `internal/zzvals` page (deleted).
  - The iOS simulator, on lesson 4.20 via `TutorialChartsUITests` screenshots.

## 6. Found checking 39 on iOS: `FlexBasis("0")` was ignored

- **Symptom:** on the simulator, horizontal values sat 3–5 pt off their bars,
  and the error grew with the label's text width. The arithmetic matched
  `base = text width`, not `base = 0`.
- **Cause:** `GrMobFlexLayout` modelled every item as `flex-basis: auto`, and
  `GrMobStyle` never decoded `FlexBasis`.
  - core's 11 `FlexBasis` calls are all `"0"`, beside `FlexGrow`, in
    `chart.go`, `bar_chart.go`, `stat_tile.go`, `donut_chart.go`,
    `calendar.go` and the tutorial.
  - So every weighted row on iOS was slightly content-biased. Compose's
    `weight()` and CSS divide exactly.
- **Fix:**
  - `GrMobStyle.flexBasis` and `zeroBasis` ("0", "0px", "0%").
  - A new `GrMobFlexZeroBasis` layout value carries the child's main-axis
    padding, or −1 for the default basis (helper `zeroBasisPadding`; inline,
    the expression took the type-checker past its time limit).
  - `baseMains(…, definite:)`: when the main extent is definite, a zero-basis
    child's base is `max(padding, automatic min, percent floor)`. An
    ideal-size query keeps measured bases, and so does a Column child whose
    verdict is `.infinity`.
- **Seen:**
  - Values now touch their bar ends.
  - Category labels centre on their slots ("First quarter" no longer cut).
- **Test:** `TestIOSFlexHonoursAZeroBasis` pins it in `mobile/verify`.
- **UI tests passing after the change:**
  - `TutorialChartsUITests`
  - `TutorialNativeFloorsUITests` (3)
  - `TutorialFooterStripUITests`
  - `TutorialScrollUITests`
  - `TutorialClockLocalTimeUITests`
- **Not viewed:** Calendar (4.9) or StatTile screens after the change. The
  simulator screenshots from a deep link showed only the top of each lesson.

## 7. Item 44: iOS floor for an Image

- **Measured** in headless Chrome, a flex item `Width 110px × Height 40px`
  beside an unbreakable word:

  | Image | Floors at |
  |---|---|
  | 400×100 | 110 |
  | 50×50 | 40 (natural aspect at the declared height) |
  | missing file | 110 |
  | no src | 0 |

- **`GrMobMinContent.width(of:)`:** an `Image` with a non-empty `src` and a px
  width floors at the declared width plus margins. It runs high only for an
  image narrower in proportion than its box.
- **MapView** is a `<div>` in htmlout, so it stays at 0.
- **Test:** four cases in `ios/verify/mincontent.swift`.

## 8. Alarms: items 41, 42 and 37

### 41: `AlarmOptions.NotifyGroup`

- `notifyPrefix()`:
  - `""` gives `grmob.alarm.` (unchanged, so older schedules still sweep)
  - a named group gives `grmob.alarms.<group>.`, with `%` and `.` escaped
    (`%25`, `%2E`) so no group's prefix begins another's
- `notifications()` ids and `sweepPrevious` use the prefix, and
  `firedAlarms(prefix, fired)` reads with it.
- New `notifyInstant(id)` parses the trailing unix seconds.
- The hook doc now says to give two Notify hooks different groups. Renaming a
  group strands what it scheduled (documented).
- **Tests:** `TestAlarmNotifyGroupsKeepTheirOwnIDs` (a pairwise prefix check
  over "", work, workday, a, a.b, a%2Eb, a_b) and
  `TestUseAlarmsSweepsItsOwnGroup`.

### 42: shown banners stay (a behaviour change)

- **`setAway(false)`** cancels only scheduled ids whose instant is still
  ahead. An id without a readable instant is still cancelled.
  - `rangAway` is gated on `hadScheduled`, not on the cancel list.
- **Host sweeps:**
  - **Android:** no `manager.cancel` and no `activeNotifications` loop; alarms
    and store entries are still cancelled.
  - **iOS:** `removePendingNotificationRequests` only; the
    `getDeliveredNotifications` pass is gone.
  - **Web:** timers are cleared, and open banners are not closed.
- **Docs:** `core.SweepNotifications` and "Sweeping by prefix" now say banners
  stay. The reason: after a force stop, Android's `attach` posts missed
  alarms late, and the mount sweep removed them moments later, sound and all.
- **Tests:**
  - `TestAlarmsNotifyScheduleWhileAwayAndCancelOnReturn` now expects only the
    five future ids cancelled.
  - `notifications_test.mjs` now expects only the re-post's close.
  - New `TestSweepsLeaveShownBanners` in `mobile/verify`. For the JS file the
    declaration cut is bounded at the next `function`, because the runtime's
    functions share one closure.

### 37: exact-alarm re-check without an Activity (`Permissions.kt`)

- New `appContext`, set in `attach`.
- `recheckExactAlarms` is gated on `appContext` and `report`, and answers
  `exactAlarmStatus(appContext)`.
- `status()` answers `notifications` below Android 13 and `exact_alarms` from
  the application context.
- The old `activity == null` gate never fired in practice: `activity` is a
  plain reference kept for the process's life.
- **Test:** `TestExactAlarmRecheckNeedsNoActivity`.

## 9. Verification

- `gofmt -l` clean, `go vet ./...` clean, and `go test ./...` passes.
- `docs/api/*` was regenerated (`go run ./internal/apidoc/gen`).
- `wasm/verify/run.sh` exits 0, and all its node tests pass.
- `ios/verify/run.sh` is all OK, including the view-layer and app-layer
  type-checks.
- `android/verify/run.sh` and `sources.sh` are OK, and
  `./gradlew compileDebugKotlin --offline` has no errors.
- **iOS simulator:** `ios/build.sh ./examples/tutorial` plus xcodebuild into
  a scratch derived-data dir. `TutorialChartsUITests` passed twice (before
  and after the flex-basis fix), and so did the four other UI test classes
  listed in section 6.
- **Android:** nothing seen. The emulator is still down from the last
  session.

## 10. Harness notes

- **UITest screenshots:**
  `TEST_RUNNER_GRMOB_SHOTS_DIR=<dir> xcodebuild test … -only-testing:GrMobUITests/TutorialChartsUITests`
  writes `charts-more-*.png`. Crop with
  `sips -c H W --cropOffset Y X in.png --out out.png`.
- **zsh:** an unmatched glob (`rm -f dir/*` on an empty dir) aborts the whole
  command. A bare `cat > file` with no heredoc hangs on stdin; stop it with
  TaskStop.
- **Python edit scripts** with a `\'` inside a single-quoted literal fail to
  parse, and then nothing runs. Put the script in a file with a quoted
  heredoc.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here). *value*: **high** = worked around
today or a second consumer has arrived; **medium** = blocks one named thing
or is a visible defect; **low** = nobody has hit the gap yet. Sorted by age,
oldest first; new items last.

**Closed this session** (previous numbering):

- **29, in part:** gradient strokes (`Shape.StrokeGradient`). Text, clipping
  and hit-testing stay open below.
- **37:** the exact-alarm re-check reads the application context.
- **38:** each label gets 2/3 of a stride.
- **39:** bar values at the tips.
- **41:** `AlarmOptions.NotifyGroup`.
- **42:** sweeps and the return leave shown banners.
- **44:** the iOS Image floor. MapView matches the web at 0.
- **51:** camelCase gradient tags in the export.

1. **(age ≥23 · value high) Lessons on hardware, and checks still open.**
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
2. **(age ≥23 · non-goal) Rename `docs/components.md` to `comps.md`.**
3. **(age ≥23 · non-goal) Rewrite `components` in the older plans.**
4. **(age ≥23 · non-goal) Trim the copied Android shell's permissions.**
5. **(age ≥23 · non-goal) Replace the iOS usage strings further.**
6. **(age ≥23 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
7. **(age 22 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android.
8. **(age 22 · non-goal) An AppBar outside the Navigator** with a custom
   `OnBack` is outranked by the Navigator's pop on Android.
9. **(age 21 · non-goal) Forward does not re-open a screen left by browser
   back.**
10. **(age 20 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose.
11. **(age 18 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain. Not profiled.
12. **(age 18 · non-goal) MaxWidth with a growing sibling on the natives.**
13. **(age 18 · non-goal) The typed-hash fold's `history.length` fallback.**
14. **(age 18 · non-goal) A page's own `pushState` while a claim is on
    screen.**
15. **(age 18 · non-goal) A Drawer's shut panel is composed on the natives.**
16. **(age 16 · non-goal) A List with no Height in a scrolled page is not
    lazy** on iOS or Android.
17. **(age 10 · value low) Compose's today `stateDescription` not heard**
    under TalkBack on the emulator.
18. **(age 10 · value medium) F-keys on iOS unverified; key delivery to a
    simulator is unreliable.**
19. **(age 10 · value low) Page-global chords on iOS are verified once.**
20. **(age 9 · non-goal) GameController cannot take a key.**
21. **(age 9 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
22. **(age 8 · value low) The Compose Row hug is unswept in `comps` widgets'
    own internal Rows.**
23. **(age 7 · value low) The iOS chord gate is unheard.**
24. **(age 7 · value low) Merging a labelled node is unheard under
    TalkBack.**
25. **(age 7 · value medium) `.claude/settings.json` cannot be edited from a
    session.** Worth an upstream report.
26. **(age 6 · value low) Alarm sound and haptics are unheard.**
    - The lesson passes no `Sound`.
    - Ringing replaces core's single audio player and doesn't resume it
      (documented).
    - Haptic pulses were not felt on a device.
    - The Notify banner's default sound is unheard.
27. **(age 6 · non-goal) `DigitalClock` digits can shift width by a pixel.**
    There's no font family in `core.Style`.
28. **(age 6 · non-goal) `AnalogClock{Smooth}` spins back once if the tick at
    00:00:00 is skipped.**
29. **(age 6 · value low) Canvas still omits** text inside the drawing,
    clipping and per-shape hit-testing. Gradient fills, gradient strokes and
    the even-odd rule are in.
30. **(age 5 · value low) A chart's hidden data table** was not built.
    - The one-sentence summary is all a screen reader gets.
    - A BarChart with more than 8 categories only gives the low and high.
    - It needs a screen-reader-only primitive core doesn't have: an API
      decision.
31. **(age 5 · non-goal) A 180° `Gauge` leaves its bottom half empty.**
32. **(age 5 · value low) Chart summaries are English**, like other widgets'
    spoken strings. `AccessibilityLabel` overrides them.
33. **(age 4 · value low) A Notify alarm is a banner, not a ringing screen.**
    - There's no full-screen intent, no looping sound, and no Snooze action.
    - iOS AlarmKit and Android `AlarmClock` / full-screen intents are the
      route to a real system alarm. Large, and only a device will tell.
34. **(age 4 · value low) `mobile.SetTimeZone` runs once at startup.** A fix
    needs core to hold the current location: an API decision.
35. **(age 4 · value low) The web's scheduled notification and its sweep are
    unseen in a real browser.** Chrome's permission prompt needs a person to
    grant it once.
36. **(age 3 · value low) Compose Rows don't shrink children in
    proportion.** Documented in `Renderer.kt` (pinMainAxis) and pinned by
    `internal/pinfixture`.
37. **(age 2 · non-goal) A stacked chart counts NaN as 0 and stacks negatives
    through the band below.** Documented under "Stacked".
38. **(age 2 · value low) The double-post claim (old item 46) is
    unreproduced.**
39. **(age 2 · value low) `MaxLines` on the natives applies to Text only.**
    Documented on `core.MaxLines`.
40. **(age 2 · value low) `testRelaunchSweepsWhatTheDeadProcessScheduled`
    needs notifications already granted.**
41. **(age 1 · value low) Charts don't use gradients yet.** An area chart's
    fade under its line is the obvious consumer (`LineChart`/`AreaChart`
    fills are still flat `33`/`66` alpha). It is a visual change, so it
    should be put to the user.
42. **(age 1 · value low) Chart palette slots 3–5 are under 3:1 on light
    pages.** Legends and spoken summaries are the relief. A ScatterChart with
    more than three series has pairs the validator's all-pairs check fails.
    Documented on `DefaultChartColors`; nothing in comps caps or warns.
43. **(age 1 · value low) `DefaultDarkChartColors` has no bundled consumer.**
    No dark theme ships. It was checked by the skill's published validation,
    not re-run here.
44. **(age 1 · value low) Gradient colour interpolation is not compared
    pixel-for-pixel across targets.** Premultiplied vs straight alpha can
    differ on a fade to transparent. Now applies to strokes too.
45. **(age 1 · value low) The 4.19 gradient row is unseen on the
    simulator.** SwiftUI gradients were seen only in a throwaway scene.
46. **(age 1 · value low) `TutorialChartsUITests` failed once, reason not
    captured.** It passed on both runs this session; watch for flakiness.
47. **(age 1 · value medium) The emulator is down** (killed for low memory).
    Restart it and reinstall the tutorial (`android/build.sh
    ./examples/tutorial`, then `./gradlew installDebug`) before any Android
    check, including item 49 below.
48. **(age 1 · non-goal) `core.LinearGradient` (a CSS string helper) is
    unused and sits beside the canvas gradient names.** Kept per CLAUDE.md's
    no-removal rule.
49. **(age 0 · value medium) None of this session's changes were seen on
    Android:** stroke gradients (Compose brush), bar values at the tips in a
    ZStack, the kept banners after a force stop, and the exact-alarm
    re-check.
50. **(age 0 · value medium) The iOS zero flex-basis change is unseen on
    Calendar (4.9) and StatTile.** Both use `FlexBasis("0")`. Their boxes
    should now be exactly equal by weight; the UI tests that touch the
    calendar pass, but nobody looked.
51. **(age 0 · value low) A zero basis is honoured on iOS only when the main
    extent is definite.** An ideal-size query keeps content bases, and a
    Column child with the `.infinity` height verdict keeps its measured base.
    Both are deliberate (see `GrMobFlexZeroBasis`).
52. **(age 0 · value low) `barValueRoom` (0.25) is a guess at label width.**
    A long `Format` on a narrow horizontal plot can still be cut outside a
    bar, or overflow a short bar inside. Grouped horizontal bands can be
    thinner than a label line (as before).
53. **(age 0 · value low) Stroke gradients are unseen on both natives.** They
    are built and type-checked, and Chrome was checked. No lesson uses one.
54. **(age 0 · value low) The iOS Image floor runs high for an image narrower
    in proportion than its box.** Chrome would let a squeezed row take it
    down to its natural aspect at the declared height; there is no natural
    size in a tree walk.
55. **(age 0 · value low) Kept banners are unseen on a device.** The relaunch
    UI test (`TutorialAlarmNotifyUITests`) was not re-run. An Android alarm
    Doze delayed past its time is no longer cancelled on return, so its
    banner may post after `OnRing` already reported it (documented in the
    hook).
56. **(age 0 · non-goal) Renaming a `NotifyGroup` strands what the old name
    scheduled** until the OS fires it. Documented; groups are meant to be
    constants.
