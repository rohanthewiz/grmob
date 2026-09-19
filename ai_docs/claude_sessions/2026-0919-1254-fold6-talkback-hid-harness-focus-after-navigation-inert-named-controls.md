# Fold6 under TalkBack: a HID harness, focus after navigation, Inert and named controls

**Session:** 5d161579-c1e2-4099-9d7c-6afbcf6115fb
**Date:** 2026-09-19 12:54 (follows "next-list-readonly-code-tab-stop-button-min-fill-onring-relaunch")
**Branch:** master (cd12896 → this commit)

## The ask

1. `/sl`: load the 11:18 doc.
2. "Continue working items in the Next list. My ZFold 6 is attached." (Mid-way:
   "I touched the screen to keep the device awake.")
3. `/sw`.

**Device:** Galaxy Z Fold6 (SM-F956U1, `RFCX70FJXZN`), Android 16, Samsung
TalkBack 16.2, folded (cover display `4630947194243491972`). The emulator was
also attached, so every command set `ANDROID_SERIAL`.

## 1. A TalkBack harness on the Samsung phone (item 62)

- **Prompts gone.** `pm grant com.samsung.android.accessibility.talkback
  android.permission.READ_PHONE_STATE` (it was denied, USER_SET). After that,
  enabling TalkBack opened no tutorial and no phone prompt.
- **Reading it.** "Display speech output" (left on from 0918-2203) shows each
  utterance at the bottom; screenshots read it. It lags while the page
  scrolls, so allow about 2s a key.
- **Driving it.**
  - `input keyevent KEYCODE_TAB` moves input focus, and TalkBack follows.
  - TalkBack's own shortcuts ignore `input`. A virtual USB keyboard from the
    `hid` shell command works: a boot-keyboard descriptor, 8-byte reports,
    kept registered by `adb shell hid - < kfifo` in the background with
    `sleep 36000 > kfifo` holding the FIFO open. Scratch helpers:
    `hidkb.py` (JSON), `kb.sh` (press and capture).
  - The keymap is "Enhanced", modifier "Action" = Meta: Meta+Right/Left is
    next/previous item. Meta+Left through `input` is system Back.
  - Injected taps and swipes do not reach the touch explorer in the app.
- Saved as memory `samsung-talkback-hid-harness`.

## 2. Fix: Tab dead after any navigation under TalkBack (item 71)

**Symptom.** After a deep link, or after pressing "Next ›" from the keyboard,
every Tab did nothing until TalkBack was turned off. Reproduced with `input`
and with the HID keyboard, so it was the app, not the harness.

**Cause (logged, then read in the Compose 1.7.6 source).**
- Every GrMob navigation swaps the screen subtree, removing the focused node.
- Compose then clears the *View's* focus:
  `FocusOwnerImpl.invalidateOwnerFocusState` →
  `AndroidComposeView.onClearFocusForOwner` → `View.clearFocus()`.
- With no focused View, `dispatchKeyEvent` saw `focus=null handled=false`.
  - Without TalkBack, ViewRootImpl's fallback (`restoreDefaultFocus`) gave
    focus back on the first Tab.
  - With TalkBack it never did.

**Fix (`MainActivity.kt`).**
- New `restoreFocusForNavigation`, called after the chord lookup and before
  `super.dispatchKeyEvent`.
- On a Tab (optionally with Shift) or an unmodified arrow, with no focused
  View, it calls `restoreDefaultFocus()` and consumes the key, as the
  framework does.
- API 24–25 use `requestFocus(View.FOCUS_DOWN)` (lint NewApi).

**Seen.** A Replace (4.15 → 4.9) then Tab gives "‹ Contents, Button" → Copy →
"‹". On 1.1, Enter on "Next ›" opens 1.2, then Tab gives Contents → Copy.

## 3. Fix: groups announced "Progress bar" (item 75)

- **Heard before.**
  - Stepper group: "2, Progress bar, Guests, 2".
  - Rating group: "0 of 5, Progress bar, Your rating".
- **Fix.** `grMobValue(range, kind)` sets `stateDescription` on any node and
  `progressBarRangeInfo` only for `progressbar`, the one role core allows
  aria-valuenow on. `contentSemantics` passes `""`.
- **Heard after:** "2, Guests, 2" and "0 of 5, Your rating".

## 4. Fix: a labelled control read its text after its name (new)

- **Heard.**
  - A star: "1 of 5, White star, Button".
  - A calendar day: "Not selected, Monday, March 2, 2026, 2, Button".
- **Cause.** A merging node's label is not set on the node itself. Compose
  1.7.6 emits it as a fake child
  (`AndroidComposeViewAccessibilityDelegateCompat` around line 843), and each
  Text child stays in the tree, so TalkBack reads both.
- **Fix (`Renderer.kt`).**
  - `LocalGrMobNamedControl` is opened by `namesItsContent(node)`: a label
    plus an onClick, or a role in button, img, tab, radio, option,
    progressbar, link, gridcell.
  - `GrMobText` under it gets `clearAndSetSemantics { }`, unless it is the
    named node itself.
  - A labelled group keeps its content, as on the web.
- **Heard after:** "3 of 5, Button"; "Not selected, Monday, March 2, 2026,
  Button".

## 5. Fix: Inert on Compose, and core.Focus on a Button (item 1)

**Found.**
- With the 4.18 Drawer shut, Tab went through the hidden panel's ✕ and rows
  one by one: nothing drawn, nothing spoken. This is the gap
  `comps/drawer.go` recorded ("natives do not read Inert").
- `GrMobButton` never read `focusEpoch`, so `Button.FocusRef` did nothing on
  Android. That covers both the Drawer's CloseRef (✕) and the ☰ that
  OnDismiss focuses.

**Fix.**
- **Inert.**
  - `GrMobStyle.inert` is parsed.
  - `LocalGrMobInert` is opened at an Inert node.
  - `RenderNode` puts `focusProperties { canFocus = false }` at the head of
    *every* node's chain under it, because the property covers only the
    focus targets after it.
- **Button focus.**
  - `GrMobButton` now runs the text field's epoch contract: a
    `LaunchedEffect(focusEpoch)`, one `withFrameNanos` first, then
    `requestFocus`.
  - The frame lets the focus tree take in the panel turning non-inert in the
    same pass.
  - The requester and the `onFocusChanged` observer ride `extra`, so the
    long-press path is covered too. The body moved to `GrMobButtonControl`.
- **Docs.** `core/style.go`, `core/style_props.go` and `comps/drawer.go`
  now say Compose reads Inert's keyboard half and SwiftUI does not. The API
  docs were regenerated.

**Seen (TalkBack off, Compose's focus highlight):**
- Tab goes Copy → ☰ → "‹ Prev".
- ☰ + Enter focuses ✕; Tab goes to Inbox; Back returns focus to ☰.

## 6. Heard and correct (item 1's screen-reader list)

- **RadioGroup (4.16).**
  - "Selected, Standard, 3–5 days · free, Radio button, 1 of 3"; …
  - "Not selected, Pick up in store. Unavailable at this address, Radio
    button, disabled, 3 of 3, In list, 3 items".
  - The group reads "Shipping".
- **StepIndicator.**
  - "Step 1: Account, done, Button"; "Selected, Step 2: Shipping" (the
    aria-current step reads as selected, as designed).
  - "Step 3: Payment"; "Step 4: Review". The strip scrolls with focus.
- **Open Drawer.** TalkBack goes from the panel title straight to the demo's
  instructions, skipping the hidden screen (☰, Inbox, 12 notes).
- **Stepper reading order (item 5).** Guests → group → − → + → "Booking for
  2"; the value is said twice (item 77).

## Pitfalls

- **Deep links need `-a android.intent.action.VIEW`.** Without it `am start
  -d` is ignored (`reportDeepLink` checks the action). The memory recipe was
  corrected.
- **TalkBack's caption and frame lag input focus** during scrolls. They can
  show Copy while Compose's focus is already on ☰. An Enter "on Copy" opened
  the drawer that way. Judge Compose's focus with TalkBack off.
- **TalkBack reads notifications** over the caption. The user's messages
  arrived mid-walk and hid two StepIndicator utterances.
- **Meta+Left via `input` is system Back.** It took the app back twice.
- **Adding a Go file moves the counts** in `wasm/verify` prose (625 → 628);
  the test says which lines to edit.
- **`codeOf`/`codeIn` blank string literals.** Use `valuesOf`/`valuesIn` for
  pins that quote a literal.
- **`stay_on_while_plugged_in`** was already 15; the screen needed no
  touching.

## Files

- **Android:**
  - `MainActivity.kt`: focus recovery.
  - `GrMobStyle.kt`: `inert`; `grMobValue` role gate.
  - `Renderer.kt`: `LocalGrMobNamedControl`, `namesItsContent`,
    `LocalGrMobInert`, the Button focus command.
- **Go:**
  - `core/style.go`, `core/style_props.go`, `comps/drawer.go` (prose only).
  - `docs/api/*` regenerated.
- **Pins:**
  - New: `mobile/verify/focus_restore_test.go`, `named_control_test.go`,
    `inert_focus_test.go`.
  - Changed: `value_test.go`; counts in `wasm/verify/repowalks_test.go` and
    `timings_test.go`.

## Verification

- `go test ./...`: ok. `go vet` on core, comps and mobile: ok.
- `android/verify/run.sh`: all OK, lint 0 errors, 23 warnings.
- `ios/verify/run.sh`: all OK, including the Release check.
- Fold6: final build installed; TalkBack off (`null`/`0`); the virtual
  keyboard unregistered.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the 25-doc
window). *value* is the payoff, not the effort:
- **high:** worked around today, or a second consumer has arrived.
- **medium:** blocks one named thing, or is a visible defect.
- **low:** nobody has hit the gap yet.

Items are sorted by age, oldest first. `lapsed@<doc>` marks an item that fell
off a list without being done. Numbering carries on from the 11:18 doc.

**Closed this session:**
- 5: heard on the Fold6 (§6); the duplicate value is item 77.
- 62: HID keyboard plus TalkBack's phone permission (§1).
- 71: an app bug, fixed (§2).
- 75: range only on a progress bar (§3).

1. **(age ≥29 · value medium · lapsed@0917-1659) Lessons on hardware, and
   checks still open.**
   - Done this session on Compose: radio, StepIndicator and aria-current
     heard; AccessibilityHidden behind a Drawer heard; `core.Focus` on a
     Button and a keyboard reaching a shut panel fixed (§5, §6).
   - Screen readers: combobox active option, "pop-up" triggers, CodeEditor
     toolbar role on Compose, SearchableSelect on the natives.
   - iOS: sheet Dialog and ActionSheet filler, Drawer under Reduce Motion,
     RTL for Drawer and CodeEditor, the sideways editor, 4.6's list;
     `core.Focus` on a Button and Inert (an iPad keyboard still reaches a
     shut panel).
   - Pinning: `Screen.Footer`, 100% layers in a pinned ZStack.
   - Android: predictive back, MaxWidth where it binds on a tablet, RTL
     capped child.
   - Devices: the Mi Max 3 (Android 10, input locked without a SIM).
2. **(age ≥29 · value medium · lapsed@0917-1659) `.claude/settings.json`
   cannot be edited from a session** (`[Self-Modification]`). Worth an
   upstream report. Not re-checked since 2026-09-15.
3. **(age ≥29 · value low) F-keys through GameController have never reached
   the app from XCUITest.** Needs a real iPad keyboard.
6. **(age ≥29 · value low · lapsed@0917-1659) iOS chords.** Page-global
   chords were verified once, and the chord gate (behind a modal, inside a
   shut Drawer panel) is unheard.
8. **(age ≥29 · non-goal · lapsed@0917-1659)**
   - Rename `docs/components.md` to `comps.md`.
   - Trim the Android shell's permissions; the iOS usage strings.
   - C4 `Carousel` (item 64 declines the scroll offset it needs).
   - Android `onBack` ranking.
   - Forward after browser back.
   - Sticky headers or `OnEndReached` in a List with no viewport on Compose.
   - MaxWidth with a growing sibling.
   - The typed-hash `history.length` fallback; a page's own `pushState`
     during a claim.
   - A Drawer's shut panel is composed on the natives (now unfocusable on
     Compose, §5).
   - A List with no Height is not lazy.
   - Unformatted commits already on a remote.
9. **(age 28 · value low · lapsed@0917-1659) Alarm sound and haptics are
   unheard,** and so is the Notify banner's default sound. Needs a person
   holding a phone.
10. **(age 28 · value low · lapsed@0917-1659) Canvas still omits** text,
    clipping and per-shape hit-testing.
11. **(age 28 · non-goal · lapsed@0917-1659)** `DigitalClock` digits shifting
    by a pixel; `AnalogClock{Smooth}` spinning back when the midnight tick is
    skipped.
12. **(age 27 · value low · API decision · lapsed@0917-1659) A chart's hidden
    data table** needs a screen-reader-only primitive.
13. **(age 27 · non-goal? · proposed) Chart summaries are English.** Every
    chart's `AccessibilityLabel` already replaces the summary. Proposed as a
    non-goal.
14. **(age 27 · non-goal · lapsed@0917-1659)** A 180° `Gauge` leaves its
    bottom half empty.
15. **(age 26 · value low · lapsed@0917-1659) A Notify alarm is a banner, not
    a ringing screen.** The route is AlarmKit or full-screen intents.
16. **(age 26 · value low · API decision · lapsed@0917-1659)
    `mobile.SetTimeZone` runs once at startup.** Fixing it needs core to hold
    the location.
17. **(age 26 · value low · lapsed@0917-1659) The web's scheduled
    notification and its sweep are unseen in a real browser.** Chrome is
    available; it needs the user to grant notification permission.
18. **(age 25 · value low · user's decision · lapsed@0917-1659) Compose Rows
    don't shrink children in proportion.** Seen on the Fold6's cover screen.
20. **(age 24 · value low · delete?) The double-post claim is unreproduced.**
    Carried thirteen times; proposed for deletion.
21. **(age 24 · non-goal · lapsed@0917-1659)** `MaxLines` on the natives
    applies to Text only; a stacked chart counts NaN as 0.
22. **(age 23 · value low · lapsed@0917-1659) `DefaultDarkChartColors` has no
    bundled consumer.** A real dark theme (item 67) would be it.
23. **(age 23 · value low · delete?) `TutorialChartsUITests` failed once,
    reason not captured.** Proposed for deletion.
24. **(age 23 · non-goal · lapsed@0917-1659)** `core.LinearGradient` is
    unused; kept per the no-removal rule.
26. **(age 22 · value medium · blocked · lapsed@0917-1659) The iOS Image floor
    runs high for a narrow image.** Needs a px-width box that can shrink
    (`grMobDimension`).
27. **(age 22 · value low → non-goal? · lapsed@0917-1659) A zero basis is
    honoured on iOS only with a definite main extent.** Proposed as a
    non-goal.
28. **(age 22 · non-goal · lapsed@0917-1659)** Renaming a `NotifyGroup`
    strands its schedules.
30. **(age 21 · value low) `barValueRoom` is still an estimate.** Exact needs
    host measurement of the plot.
34. **(age 21 · value low → non-goal? · lapsed@0917-1659) Sparkline's `Area`
    is a flat tint.** Proposed as a non-goal.
35. **(age 20 · value medium · API decision · lapsed@0917-1659) Safe-area
    insets are not a record.** `core/` has no `SafeInsets`.
36. **(age 20 · value low · lapsed@0917-1659) Lesson 4.21's TwoPane sets no
    `Origin`.** Waits on item 35.
37. **(age 20 · value low · lapsed@0917-1659) iOS `AppWindowReader` is
    type-checked only.** Not run in Split View or Stage Manager.
38. **(age 20 · value low · lapsed@0917-1659) The browser's segments/posture
    path, two-segment half.** Needs DevTools' foldable emulation toggled by a
    person.
39. **(age 20 · value low · lapsed@0917-1659) Folding shut onto the outer
    display** needs Samsung's "Continue apps on cover screen". Unseen.
40. **(age 20 · value low · lapsed@0917-1659) Housekeeping.**
    - The `GrMob_Foldable` AVD is still installed. Delete it? (User's call.)
    - The Mi Max 3 still has `stay_on_while_plugged_in` 7 (was 0) and
      auto-rotate off (was on).
    - The emulator's GrMob app has POST_NOTIFICATIONS and
      SCHEDULE_EXACT_ALARM granted.
    - New: the Fold6's Samsung TalkBack now has READ_PHONE_STATE granted (was
      denied), kept for the harness; "Display speech output" is still on.
41. **(age 20 · non-goal · lapsed@0917-1659)** More than one fold; a static
    HTML export of a TwoPane.
43. **(age 15 · value medium · user's decision) Examples should adopt the
    shipped widgets.**
    - `examples/chat` → `comps.MessageThread`: the example teaches
      `core.For` + `core.Keyed`, which MessageThread hides, and MessageThread
      can't grow to fill (needs a Grow/Fill option).
    - `examples/signup` → `PINInput`: needs a verification step and a
      re-taken `docs/images/signup.png`.
44. **(age 13 · non-goal)** A native time wheel; a sheet or Done on
    TimePicker.
45. **(age 12 · non-goal)** Heatmap as a continuous gradient; a Sequential
    ramp interpolated from `Primary`.
46. **(age 11 · value low · delete?) A sixth low-hanging-fruit round.** No
    candidates; proposed for deletion.
47. **(age 11 · non-goal)** A year-wide scrolling `CalendarHeatmap`; a
    caption-flip "Copied ✓"; a separate `Alert` widget.
48. **(age 10 · non-goal)** A two-pane layout on the natives; restructuring
    lesson bodies into guide and demo halves.
49. **(age 9 · non-goal)** Making iOS keep focus on every submit by default.
50. **(age 8 · value medium) The first hardware key after launch is lost on
    the iOS 26.5 simulator.** Worked around by `primeKeyboard`. Unchecked on
    a real iPad.
51. **(age 7 · non-goal)** A right-padding gutter or a toolbar header for
    code blocks.
52. **(age 6 · non-goal)** A smaller minimum for Buttons in general on
    Android.
53. **(age 6 · value low) iOS UIKit field: a queued key during the caret
    correction.** Not observed. Candidate fix: drain through
    `input.inputDelegate` before `replace`.
55. **(age 5 · value low) iOS Paragraph link colours on the iOS 17 floor.**
    No iOS 17 runtime is installed.
56. **(age 5 · value low · by design) The web's thread place-keeping applies
    only when the List is its own scroll box.**
59. **(age 4 · value low · contingent) The secure-field wrapper in
    `Renderer.kt` can go once the Compose BOM reaches foundation 1.10.**
    Check then:
    - whether Compose still reports read-only fields as editable;
    - whether it still clears the View's focus when the focused node leaves
      (§2);
    - whether it still splits a merged node's label into a fake child (§4).
63. **(age 4 · non-goal)** Continuing apps onto the cover screen by default.
64. **(age 5 · non-goal)** A reported scroll offset from the hosts.
65. **(age 3 · value low) A below-the-fold sweep on Android 10.** Needs item
    62's unlock on the Mi Max 3, or a person scrolling (or an API 29
    emulator image). The HID keyboard (§1) may drive it.
67. **(age 3 · value low · API decision) The Android app never follows the
    system's dark mode.** Needs the host to send `uiMode` night and core to
    pick a theme. iOS and the web are unchecked.
68. **(age 3 · value low) Why the decor-view force-dark flag stopped holding
    after an AndroidView attached is undiagnosed.** Moot for this app.
69. **(age 3 · non-goal)** MIUI's `MiuiContrastOverlay` dims screenshots to
    75% in dark mode.
74. **(age 2 · non-goal)** Lint's 23 warnings. `lint.sh` gates on errors
    only, on purpose.
76. **(age 1 · non-goal) 4.19 can't show OnRing after a force-stop.** Its
    alarms live in memory. The path works.
77. **(age 0 · value low) The Stepper says its value twice on Compose:**
    "2, Guests, 2". The group's `stateDescription` plus its merged "2" Text.
    The web drops the value on a group, so the Text is needed there; a fix
    is Compose-side (hide a labelled group's Text that equals its value
    text?).
78. **(age 0 · value low) A Drawer's panel reads "Notebook, Notebook"**: the
    navigation group's label plus its title Text. Same shape as item 77.
79. **(age 0 · value low) Escape does not close an open Drawer on Android.**
    It reaches the app unhandled; Back closes it. Check what the web and
    iPad do.
80. **(age 0 · value low) TalkBack does not follow Tab onto 4.18's ☰.**
    Compose's focus is right (TalkBack off shows it), but TalkBack said
    "Showing Inbox" or stayed on the previous node.
81. **(age 0 · value low) A control that sets its own `canFocus` later in its
    chain may override Inert.** The read-only CodeEditor's focus gate does.
    Unchecked; no bundled Inert subtree holds a CodeEditor.

Read by value instead:
- **high:** none.
- **medium:** 1, 2, 26, 35, 43, 50.
- **low:** 3, 6, 9, 10, 12, 15–18, 20, 22, 23, 27, 30, 34, 36–40, 46, 53,
  55, 56, 59, 65, 67, 68, 77–81.
- **non-goal (or proposed):** 8, 11, 13, 14, 21, 24, 28, 41, 44, 45, 47–49,
  51, 52, 63, 64, 69, 74, 76.
