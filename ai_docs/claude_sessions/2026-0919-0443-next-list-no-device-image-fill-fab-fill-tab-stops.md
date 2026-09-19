# Next list without a phone: the Image's black fill, the FAB's fill, tab stops

**Session:** 4b92f7c3-ae19-4357-b28d-42f943e2183f
**Date:** 2026-09-19 04:43 (follows "mi-max-3-android-10-force-dark-theme-sweep")
**Branch:** master (1148224 → this commit)

## The ask

1. `/sl`: load the 23:10 doc.
2. "Let's work any items remaining in the Next list that does not require a
   physical device."
3. `/sw`.

**Tools used:** emulator `emulator-5554` (Medium_Phone, API 36), the iOS 26.5
simulator (iPhone 17 Pro), Chrome 151 through the extension against
`go run ./serve -addr :8097`. The Mi Max 3 was not touched.

## 1. core.Image drew black letterboxes on every host (item 29)

**Symptom.** A "fit" Image in a box it doesn't fill showed black bars on iOS.

**What it was not.** A long bisect of `grMobBox` on the simulator (clear
background, flexible frames, a `Color.clear` overlay, `drawingGroup`) gave
contradictory rows. It stopped making sense once `drawingGroup(opaque:
false)` was still black: something was *painting* black.

**Cause.** Go. `core.imageNode` starts from `Theme.Components.Camera`, whose
Background is `#000000` in every bundled theme (a viewfinder's). Every Image
carried a black fill on every host. `comps.Avatar` and `comps.StaticMap`
already worked around it by stating Surface.

**Fix.**
- `core/image.go`: the Camera base is kept for its block display, with
  `Background` cleared, and a comment.
- The comments in `comps/static_map.go` and `comps/avatar_test.go` that said
  "the base is black" are reworded; both still state Surface as a loading
  placeholder.
- iOS `GrMobImage` (new, `Renderer.swift`): the image gets
  `frame(maxWidth/maxHeight: .infinity)` on a stated axis, so it is centred in
  its box like CSS object-fit and Compose's Image. It was top-leading,
  because `grMobDimension` aligns a box's content top-leading.
- **Seen:** scratch 150×150 images (a wide PNG, a tall JPEG) in 4.27, centred
  with transparent bars on iOS and Android. The scratch was reverted.

## 2. The FAB's fill on iOS (item 42)

A new `testFloatingActionButtonShapes` found:
- The regular disc was 47×48.7, not 56.
- The extended "Compose" pill was 20pt tall, not 56.

**Cause (iOS `GrMobButton`).**
- `grMobBox` applies Width/Height as a frame outside the Button, while
  `GrMobButtonStyle` paints the fill around the label. So the fill hugged
  the text.
- `Padding(0)` never reaches the host (the field is `omitzero`), so
  `paddingOrDefault` gave the disc 10/16 anyway. 47×48.7 is a 24pt "+" plus
  that padding.

**Fix.**
- The label takes `maxWidth`/`maxHeight: .infinity` on a stated axis
  (`grMobIsStated`), so the fill covers the declared box.
- A box with both sides stated gets no default padding
  (`paddingOrDefault(_, fixedBox:)`).
- **After:** 56 disc, 56-tall pill, 40 small disc. The test asserts all
  three, plus the disc's bottom-end place and two taps making "5 notes".

**Also added and passing:**
- `testQRCodeIsASquareOfItsSize`: 200×200, and no module seams at 2× zoom.
- `testCountdownRunsOutOnceAndTheStopwatchPauses`: OnDone fires, and Pause
  stops the stopwatch.
- 4.27's PasswordField (item 60) looks right in `dp-4.27-revealed`.

## 3. Tab stops (item 57) and bullets (item 58)

**Compose CodeEditor.**
- A literal tab drew one space wide.
- `GrMobTabStops.expand` now draws each tab as the spaces to the next stop,
  through a real `OffsetMapping`. A drawn offset inside a tab maps back to
  the tab.
- Tab-free buffers keep `OffsetMapping.Identity`. `mobile/verify` pins both
  expressions.
- Stops are `tabSize`, or 4 when tabSize is 0, as the web's CSS `tab-size`.
- **Seen on 4.13:** `return` sits at column 4, and an X typed after tapping
  inside the indent lands before the tab.

**iOS CodeEditor.** `paragraph(tabStops:)` sets `tabStops = []` and
`defaultTabInterval` = a "0" advance × stops (UIKit's default is 28pt). Seen:
`return` at column 4.

**Android RichTextEditor bullets.**
- The list indent was `base.size * 1.4` in **sp used as px** (23px ≈ 8dp),
  and the tab fell on Layout's fixed 20px step.
- Now `spToPx` (system metrics) and a `TabStopSpan.Standard` at the hanging
  indent. `isBlockDrawing` knows the new span.
- The code block's 0.5×size indent got the same sp→px fix.
- **Seen:** "•   a bullet".

## 4. Lint (item 66) and a keyboard trap

**Lint.**
- `MainActivity.dispatchKeyEvent` gets `@SuppressLint("RestrictedApi")` with
  the reason: androidx restricts its override of a public Activity method.
- The misplaced `onNewIntent` KDoc moved back above `onNewIntent`.
- The manifest has `<uses-feature android.hardware.camera required=false>`.
- `lintDebug`: 0 errors, 23 warnings.
- New `android/verify/lint.sh`, called from `run.sh` after `sources.sh`:
  - FAIL only when gradle says "Lint found errors".
  - SKIP when gradle can't run offline, or when there's no SDK, JDK or aar.
  - Warnings are reported, not failed.

**Keyboard trap.** Found by the TalkBack pass (§6).
- Compose's CodeEditor consumed Tab when read-only
  (`if (readOnly) return true`). Every tutorial code block is a read-only
  CodeEditor, so Tab went in and never came out.
- Now `false`, with a comment. The web and iOS already let it through.
- **Seen:** Tab then walked all of 4.9.

## 5. Tests

**Item 19.**
- `TutorialAlarmNotifyUITests` has `watchForThePermissionAlert` and
  `grantNotificationsIfOffered`, so the relaunch-sweep test grants
  notifications itself.
- **Verified from a fresh install** (app uninstalled first).
- The whole class passes, which is the re-run item 25 asked for.

**Item 32.** `testStatTilesShareTheRow` now pins the solved padding to 0 ±0.5,
not 0–24. StatTile insets nothing and the row has `Padding(0)`. It passes.

**LiveMapUITests.**
- It was stale: it scrolled the contents list for "Lesson 4.12", which
  collapsed chapters hide.
- `openLesson` now uses `grmob://lesson/<id>`, and both tests pass.

**The full tutorial UI suite passes (40 tests).**
- It ran after the Button, Image and tab-stop changes.
- It skipped `TutorialAlarmNotifyUITests` (run separately, above) and the
  `mobileapp`-build suites `GrMobUITests` and `TodoAppUITests`.
- `AudioUITests` needs the `mobileapp` build, as recorded before, and fails
  on the tutorial build.

## 6. Checks with no code change

- **Chrome, 4.20 (item 31).** A scratch four-series scatter drew squares for
  series 2 and 4, with square swatches the size of the round dots. Reverted.
- **Chrome, item 38 (part).**
  - Chrome 151 has `window.viewport.segments` (1) and
    `navigator.devicePosture` ("continuous").
  - 4.21 reads "336 × 744, compact, medium, normal, none".
  - The two-segment path needs DevTools' foldable emulation, which needs a
    person.
- **Android landscape (item 33).**
  - All 75 lessons' first screens at 914×411dp: all stayed on MainActivity,
    with no GrMob errors.
  - Chapter 4 paged five screens deep: clean.
  - 4.21 reports expanded/compact and puts the panes side by side.
  - Android draws 4.24's "Restart 10s" on one line (compare item 70).
- **Android tablet width (item 61).** `wm size 1920x2400` at 420dpi, about
  731dp. Chapters 1–3 and 5–8, three screens each: clean. The size and
  rotation are restored.
- **iOS ScrollIntoView inside a core.List (item 54).**
  - The scratch made 1.5's jump demo a List, with a scratch UI test.
  - "Back to row 1" put Row 1 in the list's box, and "Jump to row 10" put
    Row 10 in it.
  - It works; both scratches were reverted.
- **TalkBack on the emulator (item 4).**
  - Method: Google TTS disabled so utterances log as "TTS is not ready";
    focus moved with Tab.
  - 4.9's day 11 read "today, Wednesday, 11, Button".
  - Sunday stops logged nothing; that wasn't followed up.
  - After a TalkBack restart, Tab stopped moving focus at all (item 71), so
    item 5 was not reached.
  - TalkBack is off and Google TTS is re-enabled, as before.
- **Chart summaries (item 13).** Every chart (Line, Bar, Scatter, Donut,
  Gauge, Sparkline) already has `AccessibilityLabel`, which replaces the
  generated English summary.

## Pitfalls

- **Bisecting a visual under grMobBox without reading the wire style first.**
  Six rebuilds chased a "compositing" bug that was a `#000000` Background
  from Go. Print the node's style JSON before bisecting the host.
- **XCUITest's accessible label for an icon-plus-label FAB** is "✎  Compose".
  Match with ENDSWITH.
- **A lazy List's off-screen rows don't exist** for XCUITest; the page swipes
  also scroll a List they land on.
- **zsh doesn't word-split `$files`.** Use `${=files}`, or arrays.
- **`simctl terminate` errors when the app isn't running;** use
  `launch --terminate-running-process`.
- **The TalkBack utterance log goes quiet after a service restart,** and Tab
  may stop moving focus.

## Files

- **Go:** `core/image.go`, `comps/static_map.go`, `comps/avatar_test.go`,
  `mobile/verify/codeeditor_test.go`.
- **Android:**
  - `GrMobCodeEditor.kt`: tab stops, and the read-only Tab fix.
  - `GrMobRichText.kt`: sp→px and the tab-stop span.
  - `MainActivity.kt`, `AndroidManifest.xml`.
  - `android/verify/lint.sh` (new) and `run.sh`.
- **iOS:**
  - `Renderer.swift`: `GrMobButton` fill and padding, `grMobIsStated`, and
    `GrMobImage`.
  - `GrMobCodeEditor.swift`: `paragraph(tabStops:)`.
  - UI tests: `TutorialDevicePassUITests`, `TutorialAlarmNotifyUITests`,
    `TutorialZeroBasisAndGradientsUITests`, `LiveMapUITests`.

## Verification

- `go test ./...`: ok. `go vet` on core, comps and mobile: ok.
- `android/verify/run.sh`: all OK, including lint (0 errors).
- `ios/verify/run.sh`: all OK. `wasm/verify/run.sh`: OK.
- iOS UI tests: as §5.
- **Seen on devices:** the emulator (4.13, 4.14, 4.27 scratch, the sweeps),
  the simulator (4.13, 4.22, 4.23, 4.24, 4.27) and Chrome (4.20, 4.21).

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the 25-doc
window). *value* is the payoff, not the effort:
- **high:** worked around today, or a second consumer has arrived.
- **medium:** blocks one named thing, or is a visible defect.
- **low:** nobody has hit the gap yet.

Items are sorted by age, oldest first. `lapsed@<doc>` marks an item that fell
off a list without being done. Numbering carries on from the 23:10 doc.

**Closed this session:**
- 4: heard ("today, Wednesday, 11").
- 19: the test grants notifications itself.
- 29: Go's Camera fill, plus iOS centring.
- 31: Chrome seen; Compose was seen earlier.
- 32: tightened.
- 33: landscape swept.
- 42: seen on iOS; the FAB fill was fixed.
- 54: works, measured.
- 57: Compose and iOS tab stops.
- 58: fixed.
- 60: looks right.
- 61: tablet width swept.
- 66: lint clean, now in verify.

1. **(age ≥27 · value medium · lapsed@0917-1659) Lessons on hardware, and
   checks still open.**
   - Screen readers: radio and StepIndicator semantics, combobox active
     option, "pop-up" triggers, aria-current, CodeEditor toolbar role on
     Compose, SearchableSelect on the natives, AccessibilityHidden behind a
     Drawer. Calendar's today state is now heard on the emulator.
   - iOS: sheet Dialog and ActionSheet filler, Drawer under Reduce Motion,
     RTL for Drawer and CodeEditor, the sideways editor, 4.6's list.
   - Pinning and Drawer: `Screen.Footer`, 100% layers in a pinned ZStack,
     `core.Focus` on a Button, a hardware keyboard reaching a shut panel.
   - Android: predictive back, MaxWidth where it binds on a tablet, RTL
     capped child.
   - Devices: the Fold6 (Android 16); the Mi Max 3 (Android 10, input locked
     without a SIM).
2. **(age ≥27 · value medium · lapsed@0917-1659) `.claude/settings.json`
   cannot be edited from a session** (`[Self-Modification]`). Worth an
   upstream report. Not re-checked since 2026-09-15.
3. **(age ≥27 · value low) F-keys through GameController have never reached
   the app from XCUITest.** Needs a real iPad keyboard.
5. **(age ≥27 · value low · lapsed@0917-1659) Merging a labelled node is
   unheard under TalkBack.**
   - The shape to check is `comps.Stepper` (4.15): a RoleGroup Row labelled
     "Guests" around − and + buttons.
   - Compose keeps a clickable child as its own node, so − and + should
     stay separate stops, but that's unheard.
   - Blocked this session by item 71.
6. **(age ≥27 · value low · lapsed@0917-1659) iOS chords.** Page-global
   chords were verified once, and the chord gate (behind a modal, inside a
   shut Drawer panel) is unheard.
7. **(age ≥27 · value low · lapsed@0917-1659) The cost of a paused
   TimelineView per node** in `GrMobMotion`. Not profiled.
8. **(age ≥27 · non-goal · lapsed@0917-1659)**
   - Rename `docs/components.md` to `comps.md`.
   - Trim the Android shell's permissions; the iOS usage strings.
   - C4 `Carousel` (item 64 declines the scroll offset it needs).
   - Android `onBack` ranking.
   - Forward after browser back.
   - Sticky headers or `OnEndReached` in a List with no viewport on Compose.
   - MaxWidth with a growing sibling.
   - The typed-hash `history.length` fallback; a page's own `pushState`
     during a claim.
   - A Drawer's shut panel is composed on the natives.
   - A List with no Height is not lazy.
   - Unformatted commits already on a remote.
9. **(age 26 · value low · lapsed@0917-1659) Alarm sound and haptics are
   unheard,** and so is the Notify banner's default sound. Needs a person
   holding a phone.
10. **(age 26 · value low · lapsed@0917-1659) Canvas still omits** text,
    clipping and per-shape hit-testing.
11. **(age 26 · non-goal · lapsed@0917-1659)** `DigitalClock` digits shifting
    by a pixel; `AnalogClock{Smooth}` spinning back when the midnight tick is
    skipped.
12. **(age 25 · value low · API decision · lapsed@0917-1659) A chart's hidden
    data table** needs a screen-reader-only primitive.
13. **(age 25 · non-goal? · proposed) Chart summaries are English.** Every
    chart's `AccessibilityLabel` already replaces the summary, which is the
    localisation seam. A phrase table would be an API decision. Proposed as a
    non-goal.
14. **(age 25 · non-goal · lapsed@0917-1659)** A 180° `Gauge` leaves its
    bottom half empty.
15. **(age 24 · value low · lapsed@0917-1659) A Notify alarm is a banner, not
    a ringing screen.** The route is AlarmKit or full-screen intents.
16. **(age 24 · value low · API decision · lapsed@0917-1659)
    `mobile.SetTimeZone` runs once at startup.** Fixing it needs core to hold
    the location.
17. **(age 24 · value low · lapsed@0917-1659) The web's scheduled
    notification and its sweep are unseen in a real browser.** Chrome is
    available; it needs the user to grant notification permission.
18. **(age 23 · value low · user's decision · lapsed@0917-1659) Compose Rows
    don't shrink children in proportion.** Seen on the Fold6's cover screen
    (4.19's "Allow exact alarms" row).
20. **(age 22 · value low · delete?) The double-post claim is unreproduced.**
    Carried eleven times; proposed for deletion.
21. **(age 22 · non-goal · lapsed@0917-1659)** `MaxLines` on the natives
    applies to Text only; a stacked chart counts NaN as 0.
22. **(age 21 · value low · lapsed@0917-1659) `DefaultDarkChartColors` has no
    bundled consumer.** A real dark theme (item 67) would be it.
23. **(age 21 · value low · delete?) `TutorialChartsUITests` failed once,
    reason not captured.** Proposed for deletion.
24. **(age 21 · non-goal · lapsed@0917-1659)** `core.LinearGradient` is
    unused; kept per the no-removal rule.
25. **(age 20 · value low · lapsed@0917-1659) The alarm changes: what is left
    unseen.** Whether `OnRing` switches a one-off alarm off after an Android
    force-stop relaunch. The iOS relaunch test re-run is done (§5).
26. **(age 20 · value medium · blocked · lapsed@0917-1659) The iOS Image floor
    runs high for a narrow image.** Unblocking it needs a px-width box that
    can shrink (`grMobDimension`).
27. **(age 20 · value low → non-goal? · lapsed@0917-1659) A zero basis is
    honoured on iOS only with a definite main extent.** Documented as
    deliberate; proposed as a non-goal.
28. **(age 20 · non-goal · lapsed@0917-1659)** Renaming a `NotifyGroup`
    strands its schedules.
30. **(age 19 · value low · lapsed@0917-1659) `barValueRoom` is still an
    estimate.**
34. **(age 19 · value low → non-goal? · lapsed@0917-1659) Sparkline's `Area`
    is a flat tint.** Proposed as a non-goal.
35. **(age 18 · value medium · API decision · lapsed@0917-1659) Safe-area
    insets are not a record.** A TwoPane under the status bar can't find its
    `Origin.Y`. `core/` has no `SafeInsets`.
36. **(age 18 · value low · lapsed@0917-1659) Lesson 4.21's TwoPane sets no
    `Origin`.** Harmless; waits on item 35.
37. **(age 18 · value low · lapsed@0917-1659) iOS `AppWindowReader` is
    type-checked only.** Not run in Split View or Stage Manager.
38. **(age 18 · value low · lapsed@0917-1659) The browser's segments/posture
    path, two-segment half.**
    - Seen in Chrome 151: the APIs exist, and the one-segment report reaches
      4.21 correctly.
    - Two segments need DevTools' foldable emulation toggled by a person.
39. **(age 18 · value low · lapsed@0917-1659) Folding shut onto the outer
    display** needs Samsung's "Continue apps on cover screen". Unseen.
40. **(age 18 · value low · lapsed@0917-1659) Housekeeping.**
    - The `GrMob_Foldable` AVD is still installed. Delete it? (User's
      call.)
    - The Mi Max 3 still has `stay_on_while_plugged_in` 7 (was 0) and
      auto-rotate off (was on).
41. **(age 18 · non-goal · lapsed@0917-1659)** More than one fold; a static
    HTML export of a TwoPane.
43. **(age 13 · value medium · user's decision) Examples should adopt the
    shipped widgets.**
    - `examples/chat` → `comps.MessageThread`:
      - The example exists to teach `core.For` + `core.Keyed`, which
        MessageThread hides.
      - MessageThread can't grow to fill the space between the header and
        the composer: its outer Column takes no style, and its List defaults
        to 360px. That needs a Grow/Fill option.
    - `examples/signup` → `PINInput`: it has no code step. Adopting it means
      adding a verification step and re-taking `docs/images/signup.png`
      (shotclaims).
44. **(age 11 · non-goal)** A native time wheel; a sheet or Done on
    TimePicker.
45. **(age 10 · non-goal)** Heatmap as a continuous gradient; a Sequential
    ramp interpolated from `Primary`.
46. **(age 9 · value low · delete?) A sixth low-hanging-fruit round.** No
    candidates; proposed for deletion.
47. **(age 9 · non-goal)** A year-wide scrolling `CalendarHeatmap`; a
    caption-flip "Copied ✓"; a separate `Alert` widget.
48. **(age 8 · non-goal)** A two-pane layout on the natives; restructuring
    lesson bodies into guide and demo halves.
49. **(age 7 · non-goal)** Making iOS keep focus on every submit by default.
50. **(age 6 · value medium) The first hardware key after launch is lost on
    the iOS 26.5 simulator.** Worked around by `primeKeyboard`. Unchecked on
    a real iPad.
51. **(age 6 · non-goal)** A right-padding gutter or a toolbar header for
    code blocks.
52. **(age 5 · non-goal)** A smaller minimum for Buttons in general on
    Android.
53. **(age 4 · value low) iOS UIKit field: a queued key during the caret
    correction.** Not observed. Candidate fix: drain through
    `input.inputDelegate` before `replace`.
55. **(age 3 · value low) iOS Paragraph link colours on the iOS 17 floor.**
    No iOS 17 runtime is installed; the `.tint(firstLink)` fallback remains.
56. **(age 3 · value low · by design) The web's thread place-keeping applies
    only when the List is its own scroll box.**
59. **(age 2 · value low · contingent) The secure-field wrapper in
    `Renderer.kt` can go once the Compose BOM reaches foundation 1.10.**
62. **(age 2 · value low) An Android TalkBack route that works on a real
    phone.**
    - Samsung: first-run prompts take focus on every enable.
    - Xiaomi: MIUI's USB-debugging security switch needs a SIM.
    - Unlocks item 5 on hardware; see item 71 for the emulator route.
63. **(age 2 · non-goal)** Continuing apps onto the cover screen by default.
64. **(age 3 · non-goal)** A reported scroll offset from the hosts.
65. **(age 1 · value low) A below-the-fold sweep on Android 10.** Needs item
    62's unlock or a person scrolling. The emulator's landscape and tablet
    sweeps (§6) cover other shapes, not API 29.
67. **(age 1 · value low · API decision) The Android app never follows the
    system's dark mode.** Needs the host to send `uiMode` night and core to
    pick a theme. iOS and the web are unchecked.
68. **(age 1 · value low) Why the decor-view force-dark flag stopped holding
    after an AndroidView attached is undiagnosed.** Moot for this app.
69. **(age 1 · non-goal)** MIUI's `MiuiContrastOverlay` dims screenshots to
    75% in dark mode.
70. **(age 0 · value medium) On iOS, 4.24's "Restart 10s" wraps to two lines
    with room to spare.** Android draws it on one line. It predates this
    session (same frames with the Button change stashed).
    - Frames (pt): Restart 63→154.3 (91.3 × 60.7, the floor of "Restart"
      plus padding), Start 186.7→256, Reset 279.7→323.
    - Worked back as slots, Restart got about 115.7 (under its one-line
      ~122) and Start about 85 (over its 69.3). Proportional shrink doesn't
      do that.
    - Suspect `GrMobFlexLayout.placeSubviews` measuring bases against the
      row's final height. Instrument the solver's inputs before changing
      anything.
71. **(age 0 · value low) The emulator TalkBack harness is flaky.** After
    TalkBack was toggled off and on, Tab stopped moving input focus (no
    focused node in uiautomator) and no utterances logged. Sunday cells on
    4.9 also logged nothing in the good run. Blocks item 5.
72. **(age 0 · value low) An Android read-only CodeEditor is still a Tab stop,
    and TalkBack announces it as "Editing".** The web takes a read-only
    buffer out of the tab order (`tabindex=-1`). The trap is fixed (§4); the
    extra stop and the word "Editing" are not.
73. **(age 0 · value low) Button sizing parity.**
    - iOS now drops the default padding in a box with both sides stated;
      Android still applies 16/10, so FABSmall's 40dp disc leaves 8dp for
      its glyph (it looked fine in the sweep).
    - iOS's fill now covers a stated Width/Height but not a MinWidth or
      MinHeight.
74. **(age 0 · non-goal)** Lint's 23 warnings (NewerVersionAvailable,
    GradleDependency, UseKtx, OldTargetApi, ...). `lint.sh` gates on errors
    only, on purpose.

Read by value instead:
- **high:** none.
- **medium:** 1, 2, 26, 35, 43, 50, 70.
- **low:** 3, 5–7, 9, 10, 12, 15–18, 20, 22, 23, 25, 27, 30, 34, 36–40, 46,
  53, 55, 56, 59, 62, 65, 67, 68, 71–73.
- **non-goal (or proposed):** 8, 11, 13, 14, 21, 24, 28, 41, 44, 45, 47–49,
  51, 52, 63, 64, 69, 74.
