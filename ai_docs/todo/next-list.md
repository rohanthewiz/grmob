# Next list

The project's open follow-ups, kept in one place. Sessions edit this file in
place (see `/sess-save`) instead of copying a list forward from doc to doc, so
an item that leaves Open without a line in Closed or Non-goals shows up as a
deletion in git history. Seeded 2026-09-19 by `/next-list seed` from the
`## Next` of `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`,
with each item's `raised` traced back through all session docs.

## Conventions

- **IDs** (`N-001`…) are permanent and never reused. The old per-doc numbers
  are kept as `(was #n)` for the seed only.
- **`raised`** is the session-doc stem the item first appeared in, traced past
  any window.
- **Age** is computed (session docs since `raised`), never stored.
- **Value** is the payoff, not the effort:
  - `high`: worked around today, or a second consumer has arrived.
  - `medium`: blocks one named thing, or is a visible defect.
  - `low`: nobody has hit the gap yet, or contingent on something that
    does not exist.
  - A qualifier in parentheses (`API decision`, `user's decision`, `blocked`,
    `delete?`, `non-goal?`) says what the item waits on.
- **Nothing leaves Open** without a line in Closed (done or merged) or
  Non-goals (declined, with the reason).
- **Open stays in ID order.** New items are appended with **Next ID**, which is
  then bumped.
- `lapsed@<doc>` marks an item that once fell off a hand-carried list without
  being done; kept as history.
- In the seed, a non-goal's `declined` stem is where the item was first
  raised; the decision itself may have come in a later doc.

**Next ID:** N-060

## Open

- **N-002** · raised `2026-0912-1821-tier-a-comps-landed` · value medium
  **Lessons on hardware, and checks still open.**
  - Done on Compose (2026-0919-1254): radio, StepIndicator and aria-current
    heard; AccessibilityHidden behind a Drawer heard; `core.Focus` on a Button
    and a keyboard reaching a shut panel fixed.
  - Screen readers: combobox active option, "pop-up" triggers, CodeEditor
    toolbar role on Compose, SearchableSelect on the natives.
  - iOS: sheet Dialog and ActionSheet filler, Drawer under Reduce Motion, RTL
    for Drawer and CodeEditor, the sideways editor, 4.6's list; `core.Focus`
    on a Button and Inert (an iPad keyboard still reaches a shut panel).
  - Pinning: `Screen.Footer`, 100% layers in a pinned ZStack.
  - Android: predictive back, MaxWidth where it binds on a tablet, RTL capped
    child.
  - Devices: the Mi Max 3 (Android 10, input locked without a SIM). (was #1;
    lapsed@0917-1659)
- **N-003** · raised `2026-0913-2140-next-list-hooks-shortcuts-strips-bidi-and-floors` · value medium
  **`.claude/settings.json` cannot be edited from a session**
  (`[Self-Modification]`). Worth an upstream report. Not re-checked since
  2026-09-15. (was #2; lapsed@0917-1659)
- **N-004** · raised `2026-0913-2250-next-list-hook-blame-cell-elements-f-keys-and-a-row-that-fills` · value low
  **F-keys through GameController have never reached the app from XCUITest.**
  Needs a real iPad keyboard. (was #3)
- **N-005** · raised `2026-0914-2319-next-list-row-hug-box-chords-and-a-strip-that-divides` · value low
  **iOS chords.** Page-global chords were verified once, and the chord gate
  (behind a modal, inside a shut Drawer panel) is unheard. (was #6;
  lapsed@0917-1659)
- **N-006** · raised `2026-0916-1032-clocks-canvas-alarm` · value low
  **Alarm sound and haptics are unheard,** and so is the Notify banner's
  default sound. Needs a person holding a phone. (was #9; lapsed@0917-1659)
- **N-007** · raised `2026-0916-1032-clocks-canvas-alarm` · value low
  **Canvas still omits** text, clipping and per-shape hit-testing. (was #10;
  lapsed@0917-1659)
- **N-009** · raised `2026-0916-1032-clocks-canvas-alarm` · value low
  **A Notify alarm is a banner, not a ringing screen.** The route is AlarmKit
  or full-screen intents. (was #15; lapsed@0917-1659)
- **N-010** · raised `2026-0916-1129-charts-on-canvas` · value low (API decision)
  **A chart's hidden data table** needs a screen-reader-only primitive. (was
  #12; lapsed@0917-1659)
- **N-011** · raised `2026-0916-1129-charts-on-canvas` · value low (non-goal?, proposed)
  **Chart summaries are English.** Every chart's `AccessibilityLabel` already
  replaces the summary. Proposed as a non-goal. (was #13)
- **N-013** · raised `2026-0916-1157-next-list-charts-on-devices-tier-e-alarms-timezone` · value low (API decision)
  **`mobile.SetTimeZone` runs once at startup.** Fixing it needs core to hold
  the location. (was #16; lapsed@0917-1659)
- **N-014** · raised `2026-0916-1157-next-list-charts-on-devices-tier-e-alarms-timezone` · value low
  **The web's scheduled notification and its sweep are unseen in a real
  browser.** Chrome is available; it needs the user to grant notification
  permission. (was #17; lapsed@0917-1659)
- **N-015** · raised `2026-0916-1229-next-list-exact-alarms-boot-rearm-and-sweeps` · value low (user's decision)
  **Compose Rows don't shrink children in proportion.** Seen on the Fold6's
  cover screen. (was #18; lapsed@0917-1659)
- **N-016** · raised `2026-0916-1331-next-list-sweep-maxlines-chart-types-evenodd` · value low (delete?)
  **The double-post claim is unreproduced.** Carried thirteen times; proposed
  for deletion. (was #20)
- **N-018** · raised `2026-0916-1410-chart-palette-and-canvas-gradients` · value low
  **`DefaultDarkChartColors` has no bundled consumer.** A real dark theme
  (N-050) would be it. (was #22; lapsed@0917-1659)
- **N-019** · raised `2026-0916-1410-chart-palette-and-canvas-gradients` · value low (delete?)
  **`TutorialChartsUITests` failed once, reason not captured.** Proposed for
  deletion. (was #23)
- **N-021** · raised `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups` · value medium (blocked)
  **The iOS Image floor runs high for a narrow image.** Needs a px-width box
  that can shrink (`grMobDimension`). (was #26; lapsed@0917-1659)
- **N-022** · raised `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups` · value low → non-goal?
  **A zero basis is honoured on iOS only with a definite main extent.**
  Proposed as a non-goal. (was #27; lapsed@0917-1659)
- **N-024** · raised `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups` · value low
  **`barValueRoom` is still an estimate.** Exact needs host measurement of the
  plot. (was #30)
- **N-025** · raised `2026-0916-1643-zero-basis-floor-area-fades-scatter-squares-android-look` · value low → non-goal?
  **Sparkline's `Area` is a flat tint.** Proposed as a non-goal. (was #34;
  lapsed@0917-1659)
- **N-026** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value medium (API decision)
  **Safe-area insets are not a record.** `core/` has no `SafeInsets`. (was
  #35; lapsed@0917-1659)
- **N-027** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **Lesson 4.21's TwoPane sets no `Origin`.** Waits on N-026. (was #36;
  lapsed@0917-1659)
- **N-028** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **iOS `AppWindowReader` is type-checked only.** Not run in Split View or
  Stage Manager. (was #37; lapsed@0917-1659)
- **N-029** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **The browser's segments/posture path, two-segment half.** Needs DevTools'
  foldable emulation toggled by a person. (was #38; lapsed@0917-1659)
- **N-030** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **Folding shut onto the outer display** needs Samsung's "Continue apps on
  cover screen". Unseen. (was #39; lapsed@0917-1659)
- **N-031** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **Housekeeping.**
  - The `GrMob_Foldable` AVD is still installed. Delete it? (User's call.)
  - The Mi Max 3 still has `stay_on_while_plugged_in` 7 (was 0) and
    auto-rotate off (was on).
  - The emulator's GrMob app has POST_NOTIFICATIONS and SCHEDULE_EXACT_ALARM
    granted.
  - New: the Fold6's Samsung TalkBack now has READ_PHONE_STATE granted (was
    denied), kept for the harness; "Display speech output" is still on. (was
    #40; lapsed@0917-1659)
- **N-034** · raised `2026-0917-1935-pin-input-and-a-code-with-no-gaps` · value medium (user's decision)
  **Examples should adopt the shipped widgets.**
  - `examples/chat` → `comps.MessageThread`: the example teaches `core.For` +
    `core.Keyed`, which MessageThread hides, and MessageThread can't grow to
    fill (needs a Grow/Fill option).
  - `examples/signup` → `PINInput`: needs a verification step and a re-taken
    `docs/images/signup.png`. (was #43)
- **N-036** · raised `2026-0918-0130-round-three-eight-widgets-and-what-a-text-node-cannot-do` · value low (delete?)
  **A sixth low-hanging-fruit round.** No candidates; proposed for deletion.
  (was #46)
- **N-040** · raised `2026-0918-0910-next-list-copy-strip-edit-epochs-accent-and-the-lost-first-key` · value medium
  **The first hardware key after launch is lost on the iOS 26.5 simulator.**
  Worked around by `primeKeyboard`. Unchecked on a real iPad. (was #50)
- **N-043** · raised `2026-0918-1310-next-list-paragraph-corners-scroll-to-one-field-pin` · value low
  **iOS UIKit field: a queued key during the caret correction.** Not observed.
  Candidate fix: drain through `input.inputDelegate` before `replace`. (was
  #53)
- **N-044** · raised `2026-0918-2002-next-list-thread-widget-caret-fixes-boot-frame-proof` · value low
  **iOS Paragraph link colours on the iOS 17 floor.** No iOS 17 runtime is
  installed. (was #55)
- **N-045** · raised `2026-0918-2002-next-list-thread-widget-caret-fixes-boot-frame-proof` · value low (by design)
  The web's thread place-keeping applies only when the List is its own scroll
  box. (was #56)
- **N-047** · raised `2026-0918-2203-android-device-pass-fold6-secure-field-weight` · value low (contingent)
  **The secure-field wrapper in `Renderer.kt` can go once the Compose BOM
  reaches foundation 1.10.** Check then:
  - whether Compose still reports read-only fields as editable;
  - whether it still clears the View's focus when the focused node leaves
    (2026-0919-1254 §2);
  - whether it still splits a merged node's label into a fake child
    (2026-0919-1254 §4). (was #59)
- **N-049** · raised `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep` · value low
  **A below-the-fold sweep on Android 10.** Needs the Mi Max 3 unlocked, or a
  person scrolling (or an API 29 emulator image). The HID keyboard
  (2026-0919-1254 §1) may drive it. (was #65)
- **N-050** · raised `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep` · value low (API decision)
  **The Android app never follows the system's dark mode.** Needs the host to
  send `uiMode` night and core to pick a theme. iOS and the web are unchecked.
  (was #67)
- **N-051** · raised `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep` · value low
  **Why the decor-view force-dark flag stopped holding after an AndroidView
  attached is undiagnosed.** Moot for this app. (was #68)
- **N-055** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls` · value low
  **The Stepper says its value twice on Compose:** "2, Guests, 2". The group's
  `stateDescription` plus its merged "2" Text. The web drops the value on a
  group, so the Text is needed there; a fix is Compose-side (hide a labelled
  group's Text that equals its value text?). (was #77)
- **N-056** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls` · value low
  **A Drawer's panel reads "Notebook, Notebook"** : the navigation group's
  label plus its title Text. Same shape as N-055. (was #78)
- **N-057** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls` · value low
  **Escape does not close an open Drawer on Android.** It reaches the app
  unhandled; Back closes it. Check what the web and iPad do. (was #79)
- **N-058** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls` · value low
  **TalkBack does not follow Tab onto 4.18's ☰.** Compose's focus is right
  (TalkBack off shows it), but TalkBack said "Showing Inbox" or stayed on the
  previous node. (was #80)
- **N-059** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls` · value low
  **A control that sets its own `canFocus` later in its chain may override
  Inert.** The read-only CodeEditor's focus gate does. Unchecked; no bundled
  Inert subtree holds a CodeEditor. (was #81)

## Non-goals

- **N-001** · declined `2026-0912-1744-the-widget-library-answers-to-comps` —
  a bundle of small declines, each recorded where it was raised: (was #8;
  lapsed@0917-1659)
  - Rename `docs/components.md` to `comps.md`.
  - Trim the Android shell's permissions; the iOS usage strings.
  - C4 `Carousel` (N-046 declines the scroll offset it needs).
  - Android `onBack` ranking.
  - Forward after browser back.
  - Sticky headers or `OnEndReached` in a List with no viewport on Compose.
  - MaxWidth with a growing sibling.
  - The typed-hash `history.length` fallback; a page's own `pushState` during
    a claim.
  - A Drawer's shut panel is composed on the natives (unfocusable on Compose
    since 2026-0919-1254).
  - A List with no Height is not lazy.
  - Unformatted commits already on a remote.
- **N-008** · declined `2026-0916-1032-clocks-canvas-alarm` — `DigitalClock`
  digits shifting by a pixel; `AnalogClock{Smooth}` spinning back when the
  midnight tick is skipped. (was #11; lapsed@0917-1659)
- **N-012** · declined `2026-0916-1129-charts-on-canvas` — A 180° `Gauge`
  leaves its bottom half empty. (was #14; lapsed@0917-1659)
- **N-017** · declined
  `2026-0916-1331-next-list-sweep-maxlines-chart-types-evenodd` — `MaxLines`
  on the natives applies to Text only; a stacked chart counts NaN as 0. (was
  #21; lapsed@0917-1659)
- **N-020** · declined `2026-0916-1410-chart-palette-and-canvas-gradients` —
  `core.LinearGradient` is unused; kept per the no-removal rule. (was #24;
  lapsed@0917-1659)
- **N-023** · declined
  `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups` —
  Renaming a `NotifyGroup` strands its schedules. (was #28; lapsed@0917-1659)
- **N-032** · declined `2026-0917-0227-foldables-window-record-and-two-pane` —
  More than one fold; a static HTML export of a TwoPane. (was #41;
  lapsed@0917-1659)
- **N-033** · declined `2026-0917-1659-fab-screen-floating-and-round-two-plan`
  — A native time wheel; a sheet or Done on TimePicker. (was #44)
- **N-035** · declined
  `2026-0917-2336-tier-e-and-f-small-pieces-and-a-scale-for-quantities` —
  Heatmap as a continuous gradient; a Sequential ramp interpolated from
  `Primary`. (was #45)
- **N-037** · declined
  `2026-0918-0130-round-three-eight-widgets-and-what-a-text-node-cannot-do` —
  A year-wide scrolling `CalendarHeatmap`; a caption-flip "Copied ✓"; a
  separate `Alert` widget. (was #47)
- **N-038** · declined `2026-0918-0150-two-pane-tutorial` — A two-pane layout
  on the natives; restructuring lesson bodies into guide and demo halves. (was
  #48)
- **N-039** · declined
  `2026-0918-0713-device-pass-nine-bugs-the-simulators-found` — Making iOS
  keep focus on every submit by default. (was #49)
- **N-041** · declined
  `2026-0918-0910-next-list-copy-strip-edit-epochs-accent-and-the-lost-first-key`
  — A right-padding gutter or a toolbar header for code blocks. (was #51)
- **N-042** · declined
  `2026-0918-1046-next-list-textfieldstate-editor-stamps-button-floor-compact-copy`
  — A smaller minimum for Buttons in general on Android. (was #52)
- **N-046** · declined
  `2026-0918-2002-next-list-thread-widget-caret-fixes-boot-frame-proof` — A
  reported scroll offset from the hosts. (was #64)
- **N-048** · declined
  `2026-0918-2203-android-device-pass-fold6-secure-field-weight` — Continuing
  apps onto the cover screen by default. (was #63)
- **N-052** · declined
  `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep` — MIUI's
  `MiuiContrastOverlay` dims screenshots to 75% in dark mode. (was #69)
- **N-053** · declined
  `2026-0919-0443-next-list-no-device-image-fill-fab-fill-tab-stops` — Lint's
  23 warnings. `lint.sh` gates on errors only, on purpose. (was #74)
- **N-054** · declined
  `2026-0919-1118-next-list-readonly-code-tab-stop-button-min-fill-onring-relaunch`
  — 4.19 can't show OnRing after a force-stop. Its alarms live in memory. The
  path works. (was #76)

## Closed

Nothing yet. Closures before the seed are written up in the session docs; the
most recent are in `2026-0919-1254-…` (#5, #62, #71, #75) and
`2026-0919-1118-…` (#7, #25, #70, #72, #73).
