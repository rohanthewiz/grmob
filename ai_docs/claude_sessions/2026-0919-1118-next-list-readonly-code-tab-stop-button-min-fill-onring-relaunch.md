# Next list: read-only code off the Tab order, Button min fill, OnRing after a relaunch

**Session:** d5ac964f-b520-4a7f-8cf6-541c65e9a82b
**Date:** 2026-09-19 11:18 (follows "next-list-no-device-image-fill-fab-fill-tab-stops")
**Branch:** master (301530b → this commit)

## The ask

1. `/sl`: load the 04:43 doc.
2. "Item 70 is done. Continue working items in the Next list not requiring
   real hardware." (301530b had shortened 4.24's label to "Restart".)
3. `/sw`.

**Tools used:** emulator `emulator-5554` (Medium_Phone, API 36), the iOS 26.5
simulator (iPhone 17 Pro), and a scratch macOS Swift harness. No phone.

## 1. Android read-only CodeEditor (item 72)

**Cause of "Editing".** Compose 1.7.6 (BOM 2024.12.01) reports every
BasicTextField as editable:
`AndroidComposeViewAccessibilityDelegateCompat.android.kt:879` sets
`info.isEditable` from whether `IsEditable` is *present*, not its value, and
`readOnly` writes it as false. The node also gets the EditText class.
(Read from the downloaded `ui-android-1.7.6-sources.jar`.)

**Fix (`GrMobCodeEditor.kt`).**
- **Out of the Tab order.** New `ReadOnlyFocusGate`, read by
  `focusProperties { canFocus = !readOnly || gate.open }`.
  - A pointer press opens it (Initial pass, before the field's tap handler
    asks for focus), and so does Go's `core.Focus` command.
  - It shuts when focus leaves, or when a press ends without focus (Final
    pass of the same event; pointer passes resume synchronously).
  - Plain fields, not Compose state, on purpose: Compose observes
    `focusProperties` reads while a node holds focus and force-clears focus
    when `canFocus` turns false. Keying it on the input mode would drop the
    caret out of a tapped block the moment Tab switched to keyboard mode, and
    the Tab would restart from the top of the screen.
- **Reads as text.** A read-only buffer's Box gets
  `clearAndSetSemantics { text = AnnotatedString(buffer.text) }`, as iOS's
  non-editable UITextView reads. The horizontal scroll's semantics stay on
  the node.
- **Gutter hidden** from TalkBack (`clearAndSetSemantics { }`), and on iOS
  `gutter.isAccessibilityElement = false`, as the web's gutter is
  `aria-hidden`. Each number had been its own stop.

**Seen on the emulator.**
- No TalkBack, 4.9: Tab goes Contents → Copy → Previous month. A tap in the
  block then Tab goes to Previous month; Shift+Tab ×2 from day 1 goes back to
  Copy. Long-press still selects (Copy / Select all menu).
- The uiautomator class is HorizontalScrollView, not EditText.
- TalkBack, 4.9:
  - Baseline build (change stashed): Copy → "Editing." → Previous month.
  - This build: Copy → Previous month, both directions.

**Pins.** New `TestReadOnlyCodeEditorLeavesTheTabOrderOnCompose`; gutter
pins added on both hosts; two existing pins respelled for the new modifier
chain.

## 2. Button sizing parity (item 73)

- **Android half: not a defect.** `GrMobStyle.padding` is a non-null `Edges`
  parsed to zeros when absent, so `(s?.padding?.left ?: 16)` defaults only
  when the Button has no style at all. FABSmall's `Padding(0)` is already
  0dp. (iOS differs for a styled Button with no padding — 10/16 there, 0
  here — but every theme Button states padding.)
- **iOS half: fixed.** `GrMobButton`'s label now takes
  `.grMobMinimum(width:height:alignment: .center)` after its padding, so the
  fill grows to a MinWidth/MinHeight as it does to Width/Height.
  `grMobMinimum` went from fileprivate to internal for it. The outer floor in
  grMobBox is kept (never larger than the Button now).
- **Seen:** a scratch `comps.Button{Label: "Floor", MinWidth 200px,
  MinHeight 80px}` in 4.22 measured 200×80 with the fill covering it and the
  label centred. Reverted. No bundled widget sets a minimum on a Button.

## 3. OnRing after an Android force-stop (item 25)

The tutorial's "Ring at the next minute" alarm lives in memory, so after a
force-stop it's gone before the sweep reply arrives, and OnRing (which
reports only alarms still in the list) can't be seen. So:
- A scratch one-off `{ID: "scratch-once", 07:55, Enabled: true}` was put in
  4.19's initial list, so it survives the relaunch.
- Granted the app POST_NOTIFICATIONS (`pm grant`) and SCHEDULE_EXACT_ALARM
  (`appops set ... allow`) on the emulator.
- Home → `dumpsys alarm` showed the RTC_WAKEUP at 07:55:00 → `am force-stop`
  at 07:52:37 → nothing posted while stopped.
- Relaunch at 07:55:41: the late banner
  `grmob.alarm.scratch-once.1789822500` posted and stayed; the switch was
  **off** (it started on); no in-app ringing panel.
- Reverted.

## 4. Paused Spin cost per node (item 7)

Scratch macOS harness (`-O`, NSHostingView, VStack of N Texts, with and
without `GrMobSpin(periodMs: 0)`):

| rows | first render plain | with paused spin | update plain | with paused spin |
|---|---|---|---|---|
| 200 | 28 ms | 29 ms | 24.0 ms | 24.6 ms |
| 1000 | 143 ms | 155–162 ms | 124 ms | 126–129 ms |

About 10–20 µs a node (8–13%) on first render, 1–3% on an update, nothing
between updates. Written into the `GrMobSpin` comment in place of
"Unmeasured".

## 5. The TalkBack harness (item 71) and the Stepper (item 5)

**Found.**
- `uiautomator dump` turns TalkBack off and back on (UiAutomation suppresses
  accessibility services while connected). The previous harness dumped
  between Tabs, which explains part of the flakiness.
- After a reboot, TalkBack binds only if `accessibility_enabled 1` is set
  *after* `enabled_accessibility_services`.
- Sunday cells on 4.9 do speak ("Sunday, March 1, 2026"); the earlier
  silence was the harness.

**Still flaky.** In the app, keys sometimes do nothing after a launch or deep
link, with TalkBack's frame stuck on "‹ Contents". One dump cycle fixed it on
4.9 twice, but not on 4.15 twice. Same with the change stashed, so not
caused by item 72. Injected swipes moved TalkBack on the launcher but not in
the app. Time-boxed.

**Item 5 from the tree instead (uiautomator, TalkBack off).**
- − and + are separate clickable nodes ("Decrease", "Increase"); "2" is its
  own TextView; the name "Guests" is a fake child node of the group.
- **New:** the Stepper's group, and the Rating's, have class
  **ProgressBar**: `grMobValue` maps a bounded `AccessibilityValue` to
  `progressBarRangeInfo` without consulting the role (deliberately, per its
  comment). Unheard whether TalkBack says "progress bar". Item 75.

## 6. Item 30 looked at

`barValueRoom` guesses a 220px plot. An exact answer needs the host to
measure the plot at layout time; no change.

## Pitfalls

- **`uiautomator dump` cycles TalkBack.** Read TalkBack through logcat only
  ("TTS is not ready" lines with Google TTS disabled).
- **zsh `[ a \< b ]` fails** ("condition expected: <") and a `while` using it
  exits at once. Compare numbers with `(( ))`.
- **A background wait that relaunched too early** spoiled the first OnRing
  run; check the device clock (`adb shell date`) in the loop.
- **Rebuild the AAR after reverting Go scratch**: gradle alone reinstalls
  whatever `android/build.sh` last bound.
- **The Tab-walk script's `focused` node is the Compose wrapper View**, with
  no text; use bounds and a screenshot to tell which control it is.

## Files

- **Android:** `GrMobCodeEditor.kt` (focus gate, read-only semantics, gutter).
- **iOS:** `GrMobCodeEditor.swift` (gutter), `Renderer.swift` (Button min
  fill), `GrMobStyle.swift` (`grMobMinimum` internal; Spin cost measured).
- **Go:** `mobile/verify/codeeditor_test.go`.

## Verification

- `go test ./...`: ok. `go vet` on core, comps and mobile: ok.
- `android/verify/run.sh`: all OK, lint 0 errors, 23 warnings.
- `ios/verify/run.sh`: all OK, including the Release (WMO) check.
- iOS tutorial UI suite: **42 passed**, skipping GrMobUITests,
  TodoAppUITests, AudioUITests (need the `mobileapp` build) and
  TutorialAlarmNotifyUITests.
- Emulator: clean tutorial reinstalled; TalkBack off, Google TTS re-enabled.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the 25-doc
window). *value* is the payoff, not the effort:
- **high:** worked around today, or a second consumer has arrived.
- **medium:** blocks one named thing, or is a visible defect.
- **low:** nobody has hit the gap yet.

Items are sorted by age, oldest first. `lapsed@<doc>` marks an item that fell
off a list without being done. Numbering carries on from the 04:43 doc.

**Closed this session:**
- 7: measured (§4).
- 25: OnRing switched the one-off off after a force-stop relaunch (§3).
- 70: done by 301530b (user).
- 72: off the Tab order, reads as text, seen under TalkBack (§1).
- 73: Android half not a defect; iOS min fill fixed (§2).

1. **(age ≥28 · value medium · lapsed@0917-1659) Lessons on hardware, and
   checks still open.**
   - Screen readers: radio and StepIndicator semantics, combobox active
     option, "pop-up" triggers, aria-current, CodeEditor toolbar role on
     Compose, SearchableSelect on the natives, AccessibilityHidden behind a
     Drawer.
   - iOS: sheet Dialog and ActionSheet filler, Drawer under Reduce Motion,
     RTL for Drawer and CodeEditor, the sideways editor, 4.6's list.
   - Pinning and Drawer: `Screen.Footer`, 100% layers in a pinned ZStack,
     `core.Focus` on a Button, a hardware keyboard reaching a shut panel.
   - Android: predictive back, MaxWidth where it binds on a tablet, RTL
     capped child.
   - Devices: the Fold6 (Android 16); the Mi Max 3 (Android 10, input locked
     without a SIM).
2. **(age ≥28 · value medium · lapsed@0917-1659) `.claude/settings.json`
   cannot be edited from a session** (`[Self-Modification]`). Worth an
   upstream report. Not re-checked since 2026-09-15.
3. **(age ≥28 · value low) F-keys through GameController have never reached
   the app from XCUITest.** Needs a real iPad keyboard.
5. **(age ≥28 · value low · lapsed@0917-1659) Merging a labelled node is
   unheard under TalkBack.**
   - `comps.Stepper` (4.15) seen in the tree: − and + separate, "2" separate,
     "Guests" a fake child of the group (§5).
   - Still unheard: the reading order and what the group says; see item 75.
   - Blocked by item 71 on 4.15.
6. **(age ≥28 · value low · lapsed@0917-1659) iOS chords.** Page-global
   chords were verified once, and the chord gate (behind a modal, inside a
   shut Drawer panel) is unheard.
8. **(age ≥28 · non-goal · lapsed@0917-1659)**
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
9. **(age 27 · value low · lapsed@0917-1659) Alarm sound and haptics are
   unheard,** and so is the Notify banner's default sound. Needs a person
   holding a phone.
10. **(age 27 · value low · lapsed@0917-1659) Canvas still omits** text,
    clipping and per-shape hit-testing.
11. **(age 27 · non-goal · lapsed@0917-1659)** `DigitalClock` digits shifting
    by a pixel; `AnalogClock{Smooth}` spinning back when the midnight tick is
    skipped.
12. **(age 26 · value low · API decision · lapsed@0917-1659) A chart's hidden
    data table** needs a screen-reader-only primitive.
13. **(age 26 · non-goal? · proposed) Chart summaries are English.** Every
    chart's `AccessibilityLabel` already replaces the summary. Proposed as a
    non-goal.
14. **(age 26 · non-goal · lapsed@0917-1659)** A 180° `Gauge` leaves its
    bottom half empty.
15. **(age 25 · value low · lapsed@0917-1659) A Notify alarm is a banner, not
    a ringing screen.** The route is AlarmKit or full-screen intents.
16. **(age 25 · value low · API decision · lapsed@0917-1659)
    `mobile.SetTimeZone` runs once at startup.** Fixing it needs core to hold
    the location.
17. **(age 25 · value low · lapsed@0917-1659) The web's scheduled
    notification and its sweep are unseen in a real browser.** Chrome is
    available; it needs the user to grant notification permission.
18. **(age 24 · value low · user's decision · lapsed@0917-1659) Compose Rows
    don't shrink children in proportion.** Seen on the Fold6's cover screen.
20. **(age 23 · value low · delete?) The double-post claim is unreproduced.**
    Carried twelve times; proposed for deletion.
21. **(age 23 · non-goal · lapsed@0917-1659)** `MaxLines` on the natives
    applies to Text only; a stacked chart counts NaN as 0.
22. **(age 22 · value low · lapsed@0917-1659) `DefaultDarkChartColors` has no
    bundled consumer.** A real dark theme (item 67) would be it.
23. **(age 22 · value low · delete?) `TutorialChartsUITests` failed once,
    reason not captured.** Passed again this session. Proposed for deletion.
24. **(age 22 · non-goal · lapsed@0917-1659)** `core.LinearGradient` is
    unused; kept per the no-removal rule.
26. **(age 21 · value medium · blocked · lapsed@0917-1659) The iOS Image floor
    runs high for a narrow image.** Needs a px-width box that can shrink
    (`grMobDimension`).
27. **(age 21 · value low → non-goal? · lapsed@0917-1659) A zero basis is
    honoured on iOS only with a definite main extent.** Proposed as a
    non-goal.
28. **(age 21 · non-goal · lapsed@0917-1659)** Renaming a `NotifyGroup`
    strands its schedules.
30. **(age 20 · value low) `barValueRoom` is still an estimate.** Looked at
    (§6): exact needs host measurement of the plot.
34. **(age 20 · value low → non-goal? · lapsed@0917-1659) Sparkline's `Area`
    is a flat tint.** Proposed as a non-goal.
35. **(age 19 · value medium · API decision · lapsed@0917-1659) Safe-area
    insets are not a record.** `core/` has no `SafeInsets`.
36. **(age 19 · value low · lapsed@0917-1659) Lesson 4.21's TwoPane sets no
    `Origin`.** Waits on item 35.
37. **(age 19 · value low · lapsed@0917-1659) iOS `AppWindowReader` is
    type-checked only.** Not run in Split View or Stage Manager.
38. **(age 19 · value low · lapsed@0917-1659) The browser's segments/posture
    path, two-segment half.** Needs DevTools' foldable emulation toggled by a
    person.
39. **(age 19 · value low · lapsed@0917-1659) Folding shut onto the outer
    display** needs Samsung's "Continue apps on cover screen". Unseen.
40. **(age 19 · value low · lapsed@0917-1659) Housekeeping.**
    - The `GrMob_Foldable` AVD is still installed. Delete it? (User's call.)
    - The Mi Max 3 still has `stay_on_while_plugged_in` 7 (was 0) and
      auto-rotate off (was on).
    - New: the emulator's GrMob app has POST_NOTIFICATIONS and
      SCHEDULE_EXACT_ALARM granted (for §3).
41. **(age 19 · non-goal · lapsed@0917-1659)** More than one fold; a static
    HTML export of a TwoPane.
43. **(age 14 · value medium · user's decision) Examples should adopt the
    shipped widgets.**
    - `examples/chat` → `comps.MessageThread`: the example teaches
      `core.For` + `core.Keyed`, which MessageThread hides, and MessageThread
      can't grow to fill (needs a Grow/Fill option).
    - `examples/signup` → `PINInput`: needs a verification step and a
      re-taken `docs/images/signup.png`.
44. **(age 12 · non-goal)** A native time wheel; a sheet or Done on
    TimePicker.
45. **(age 11 · non-goal)** Heatmap as a continuous gradient; a Sequential
    ramp interpolated from `Primary`.
46. **(age 10 · value low · delete?) A sixth low-hanging-fruit round.** No
    candidates; proposed for deletion.
47. **(age 10 · non-goal)** A year-wide scrolling `CalendarHeatmap`; a
    caption-flip "Copied ✓"; a separate `Alert` widget.
48. **(age 9 · non-goal)** A two-pane layout on the natives; restructuring
    lesson bodies into guide and demo halves.
49. **(age 8 · non-goal)** Making iOS keep focus on every submit by default.
50. **(age 7 · value medium) The first hardware key after launch is lost on
    the iOS 26.5 simulator.** Worked around by `primeKeyboard`. Unchecked on
    a real iPad.
51. **(age 6 · non-goal)** A right-padding gutter or a toolbar header for
    code blocks.
52. **(age 5 · non-goal)** A smaller minimum for Buttons in general on
    Android.
53. **(age 5 · value low) iOS UIKit field: a queued key during the caret
    correction.** Not observed. Candidate fix: drain through
    `input.inputDelegate` before `replace`.
55. **(age 4 · value low) iOS Paragraph link colours on the iOS 17 floor.**
    No iOS 17 runtime is installed.
56. **(age 4 · value low · by design) The web's thread place-keeping applies
    only when the List is its own scroll box.**
59. **(age 3 · value low · contingent) The secure-field wrapper in
    `Renderer.kt` can go once the Compose BOM reaches foundation 1.10.**
    Check then whether Compose still reports read-only fields as editable
    (§1); the CodeEditor's semantics override could go too.
62. **(age 3 · value low) An Android TalkBack route that works on a real
    phone.** Samsung's first-run prompts; Xiaomi's SIM-gated USB debugging.
63. **(age 3 · non-goal)** Continuing apps onto the cover screen by default.
64. **(age 4 · non-goal)** A reported scroll offset from the hosts.
65. **(age 2 · value low) A below-the-fold sweep on Android 10.** Needs item
    62's unlock or a person scrolling (or an API 29 emulator image).
67. **(age 2 · value low · API decision) The Android app never follows the
    system's dark mode.** Needs the host to send `uiMode` night and core to
    pick a theme. iOS and the web are unchecked.
68. **(age 2 · value low) Why the decor-view force-dark flag stopped holding
    after an AndroidView attached is undiagnosed.** Moot for this app.
69. **(age 2 · non-goal)** MIUI's `MiuiContrastOverlay` dims screenshots to
    75% in dark mode.
71. **(age 1 · value low) The emulator TalkBack harness is still flaky.**
    - Known now: `uiautomator dump` cycles TalkBack; enable order matters
      after a reboot (§5).
    - Open: in the app, keys sometimes do nothing after a launch or deep
      link; a dump cycle fixed 4.9 but not 4.15. Injected swipes work on the
      launcher, not in the app.
74. **(age 1 · non-goal)** Lint's 23 warnings. `lint.sh` gates on errors
    only, on purpose.
75. **(age 0 · value low · design decision) Compose reports a Stepper's and a
    Rating's group as a ProgressBar.** `grMobValue` sets
    `progressBarRangeInfo` for any bounded `AccessibilityValue`, whatever the
    role (deliberate, per its comment). Unheard whether TalkBack adds
    "progress bar". The web drops aria-value* on a group. Scoping it to
    progressbar/slider roles would change that decision.
76. **(age 0 · non-goal) 4.19 can't show OnRing after a force-stop.** Its
    alarms live in memory, so a one-off set from the demo is gone before the
    sweep reply. The path works (§3); showing it would need persistence in a
    lesson.

Read by value instead:
- **high:** none.
- **medium:** 1, 2, 26, 35, 43, 50.
- **low:** 3, 5, 6, 9, 10, 12, 15–18, 20, 22, 23, 27, 30, 34, 36–40, 46, 53,
  55, 56, 59, 62, 65, 67, 68, 71, 75.
- **non-goal (or proposed):** 8, 11, 13, 14, 21, 24, 28, 41, 44, 45, 47–49,
  51, 52, 63, 64, 69, 74, 76.
