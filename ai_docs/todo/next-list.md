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

**Next ID:** N-077

## Open

- **N-002** · raised `2026-0912-1821-tier-a-comps-landed` · value medium
  **Lessons on hardware, and checks still open.**
  - Done on Compose (2026-0919-1254): radio, StepIndicator and aria-current
    heard; AccessibilityHidden behind a Drawer heard; `core.Focus` on a Button
    and a keyboard reaching a shut panel fixed.
  - Screen readers: combobox active option, "pop-up" triggers, CodeEditor
    toolbar role on Compose, SearchableSelect on the natives.
  - Fixed but unheard (2026-0919-2146): the Stepper group's and the Drawer
    panel's utterances on Compose. The duplicate is gone from the
    accessibility tree, measured on the emulator; a group node is not
    reachable by Tab and TalkBack's reading-order keys ignore `adb input`, so
    hearing the line needs the Fold6's HID keyboard (Meta+Right).
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
  (N-050) would be it. The tutorial's own `darkTheme`
  (examples/tutorial/theme.go, 2026-0921-1419) now spends it and
  `DefaultDarkSequentialColors`, but no core theme does (see N-074).
  (was #22; lapsed@0917-1659)
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
- **N-027** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **Lesson 4.21's TwoPane sets no `Origin`.** No longer waits on N-026:
  4.21 sets `IgnoreHorizontalFold`, so its live axis is the vertical hinge
  and the missing term is `Origin.X` — the demo panel's own left padding, a
  layout constant no host reports. A correct `Origin` there would be a
  hard-coded guess; showing the documented `Origin{Y: insets.Top + bar}`
  case honestly needs a non-scrolling demo. (was #36; lapsed@0917-1659)
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
  send `uiMode` night and core to pick a theme. iOS is unchecked. The web
  tutorial does follow it now: the page resolves System/Light/Dark and sends
  a `theme` host event, and the app swaps to a tutorial-local `darkTheme`
  (2026-0921-1419). The same event is the shape a native host could send.
  (was #67)
- **N-051** · raised `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep` · value low
  **Why the decor-view force-dark flag stopped holding after an AndroidView
  attached is undiagnosed.** Moot for this app. (was #68)
- **N-057** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls` · value low
  **Nothing closes on Escape, on any host** — not Drawer, and not Dialog,
  Menu, ActionSheet or Lightbox either. Read off the source
  (2026-0919-2303), so the item's "check the web and iPad" is answered:
  - Web: `core.Modal` renders as a plain `div`, not `<dialog>`, so there is
    no free browser Escape; the only bare-key listener is the combobox's,
    which clears its active option and explicitly leaves Escape to the page.
  - Android: `dispatchKeyEvent` names the key "Escape" and looks for a
    page-global chord, but `pageGlobal` requires a modifier or an F-key, and
    `findKeyShortcut` only matches nodes with an `onClick` — a Drawer panel
    carries `onBack`. Hence "unhandled".
  - iPad: the chord gate drops every bare key too. A sheet-backed Modal may
    still close for free through SwiftUI's `isPresented` binding, which
    already calls `onDismiss`; a Drawer is a ZStack layer and never can.
  Two shapes if it is ever built: route Escape into the existing back claim
  (`onBackPressedDispatcher`, the web's `innermostBackClaim()`) — no Go API,
  but it would also pop a Navigator route and fire an AppBar back arrow — or
  a layer-only `core.OnEscape` beside `OnBack`. (was #79)
- **N-058** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls` · value low
  **TalkBack does not follow Tab onto 4.18's ☰.** Compose's focus is right
  (TalkBack off shows it), but TalkBack said "Showing Inbox" or stayed on the
  previous node. (was #80)
- **N-061** · raised `2026-0919-2303-safe-insets-record-inert-codeeditor-sse-cleanup` · value low
  **The browser reports no safe-area insets.** `Window.Insets` is zero on the
  web, which is right for a page in a browser window but wrong for an
  installed PWA drawn behind a notch. The values exist only as CSS
  `env(safe-area-inset-*)`; reading them means a probe element and a
  `getComputedStyle` per report. Left undone on purpose (the reason is in
  `wasm/grmob-runtime.js`'s windowMetrics comment), not overlooked.

- **N-062** · raised `2026-0921-0912-comps-round-four-phase-1-chat-family` · value medium
  **Phase 1's chat widgets, unrun on a device.** Lesson 4.34 and
  `examples/chat` were looked at in headless Chrome only.
  - `TypingIndicator` under Reduce Motion on all three live targets: the
    hosts drop the `Transition` and Go still steps the phase, so the dots
    change alpha without the ease. The widget's doc claims that is a quiet
    blink (a 6pt dot, one grey to another) and nobody has seen it.
  - That a `core.Opacity` `Transition` on a 6pt `Box` actually eases on
    Compose and SwiftUI. It is the only thing that moves the dots (they were
    a background-colour ease until the session that added `core.Opacity`;
    see N-064).
  - `TypingIndicator`'s `RoleStatus` appearing from `Display none`: heard on
    TalkBack? VoiceOver is expected to say nothing (the known live-region
    gap, core/role.go).
  - `ReactionBar` chips: the selected state heard with the spelled-out name
    ("thumbs up, 3 reactions, selected"), and emoji glyphs drawn in a `Button`
    label on both natives.
  - `Poll` results: one stop per option, with the hidden `ProgressBar` and
    texts not reachable by swipe.

- **N-064** · raised `2026-0921-0935-core-opacity-and-typing-indicator-fade` · value medium
  **`core.Opacity` is unrun on a device.** It
  compiles on all four targets and its call sites, layer order and sentinel
  are pinned from source, but nobody has seen it fade.
  - Compose: that `Modifier.alpha` outside `Modifier.shadow` keeps the whole
    shadow mid-fade. An alpha below 1 composites the layer offscreen, and an
    offscreen layer may clip what is drawn outside the node's bounds.
  - Compose: `animatedStyle` easing the alpha, and snapping under "Remove
    animations".
  - SwiftUI: the fade under the node's one `.animation`, and whether a view
    at exactly 0 still takes taps and VoiceOver focus (the doc says taps stop;
    that is from SwiftUI's known behaviour, not measured here).
  - Noticed beside it, untouched: Compose's `DisplayHidden` alpha sits at the
    foot of `boxModifier`, inside the background and border, so a hidden node
    with a fill may still draw the fill. iOS and the web hide the whole box.
  - Seen once on Compose (2026-09-21, the emulator): `NumberPad`'s unpainted
    corner key is `Opacity(0)` and draws nothing, so the sentinel reaches
    `Modifier.alpha` as 0. A static zero only; no fade has been watched.
  - Its first consumer is `TypingIndicator`'s dots (looked at in headless
    Chrome only), so N-062's device checks are this field's too. No lesson
    teaches the prop itself.

- **N-065** · raised `2026-0921-1019-comps-round-four-phase-2-inputs` · value medium
  **The caret carry and the mask, unrun on iOS and on hardware.**
  - Android: `carryCaret` replaced "keep the raw offset" in `GrMobTextField`.
    Run on the emulator only (the mask at 1s and machine speed, backspace,
    mid-text, and lesson 2.3's UPPERCASE mid-text). Not on the Fold6, and not
    with an IME that composes (Gboard's suggestions, Samsung's keyboard):
    `adb shell input text` commits whole keys.
  - iOS: `MaskedInput` has not been typed into at all. The iOS field's
    `write` is believed to follow `rebasefixture.Carry`'s rule (read, not
    run), and `ios/verify` does not run the Swift side against `CarryCases`,
    because the rule lives inside `write` and not in a function of its own.
    Extracting it is the way to hold iOS to the table.
  - The known limit (a key typed mid-text at the end of a group leaves the
    caret after the reflow) is the same on all three hosts by construction;
    seen on Android only.
- **N-066** · raised `2026-0921-1019-comps-round-four-phase-2-inputs` · value medium
  **Phase 2's inputs, unrun on a device beyond a look.** Lesson 5.9 was
  looked at in headless Chrome and on the Android emulator (pad and
  swatches drawn right; two pad keys tapped).
  - `NumberPad`: the haptic per key felt on a phone; the unpainted corner
    (`Opacity(0)`, `Disabled`, hidden) skipped by TalkBack and VoiceOver, and
    not a Tab stop with a hardware keyboard.
  - `ColorSwatchPicker`: each radio heard with its name and "selected"; the
    ring and check on SwiftUI; the hex field's return committing the short
    form on both natives.
  - `RangeSlider`: a drag past the other thumb on a touch screen, and that
    the pushed thumb's native position follows Go's value when it was not
    the one dragged.
  - The lock-screen dots' `RoleStatus` line ("Passcode, 2 of 4 entered"):
    heard on TalkBack per key? VoiceOver is expected to say nothing (the
    known live-region gap, core/role.go).
- **N-068** · raised `2026-0921-1035-comps-round-four-phase-3-structure` · value medium
  **Phase 3's structure widgets, unrun on a device.** Lesson 4.35 was looked
  at in headless Chrome only (tree alignment, indents, the wizard's footer).
  - `TreeView`: a branch heard as "docs, collapsed, button" and the chosen
    leaf as selected on TalkBack and VoiceOver; that a `listitem` Box holding
    a button and a nested list is walked in order by swipe; that the level,
    inert on both natives by design, does no harm there. On the web, a
    screen reader saying "level 2" (the DOM was not read for `aria-level`).
  - `TreeView` under RTL: the indent is a Row's leading spacer, which should
    mirror on all three live targets. Unseen.
  - `Wizard`: the `RoleStatus` line heard on a step change on TalkBack and
    in a browser's screen reader; where TalkBack's focus lands after Next
    replaces the body (the keyed body is a replacement, and Compose clears
    View focus when the focused node leaves: 2026-0919-1254 §2).
  - `Wizard.Footer()` in `Screen.Footer` above the keyboard is N-002's
    pinning check with a consumer now; no bundled screen does it yet.
- **N-069** · raised `2026-0921-1035-comps-round-four-phase-3-structure` · value low (API decision)
  **Nothing can move focus to a heading.** `Wizard` wanted the
  focus-after-navigation rule for a step change and could not have it:
  `core.FocusTarget` is read by fields and by a Compose Button, and no
  target focuses a Text. The web half is `tabindex="-1"` plus `focus()` in
  `applyFocusCommand`; the natives need an accessibility-focus request
  (`requestFocus` on a semantics node, `AccessibilityFocusState`), which is
  a different thing from input focus and may deserve its own prop. Until
  then `Wizard` announces the step through a `RoleStatus` line, which iOS
  does not speak.
- **N-070** · raised `2026-0921-1057-comps-round-four-phase-4-charts` · value medium
  **Phase 4's charts, unrun on a device.** Lesson 4.36 was looked at in
  headless Chrome only (all four demos; the look found two defects, fixed).
  - `Waveform`: that a round-capped stroke `BarWidth` px thick really is
    unscaled under `CanvasStretch` on Compose and SwiftUI, as core.Canvas's
    doc says of every stroke. It is the widget's whole premise, and no
    earlier chart stroked anything wider than 2 px, where a scaled stroke
    would not have shown. Also the half-px silent bar drawn as a dot.
  - `RadarChart`: rim labels placed by `core.Translate` px on a centred
    ZStack layer, on both natives (Translate's only other consumer is
    Drawer's "-100%"); the ring-value chips over a filled polygon.
  - `CandlestickChart`: the 1px doji body at a native's pixel density.
  - `AudioPlayer.Waveform` with a real stream: the strip filling as the
    status ticks, and under the finger while scrubbing.
  - Every chart's one spoken sentence on TalkBack and VoiceOver.
- **N-072** · raised `2026-0921-1118-comps-round-four-phase-5-editable-grid` · value medium
  **`EditableGrid`, unrun on a device.** Lesson 4.37 was looked at and driven
  in headless Chrome only (the look found two defects, fixed; the keyboard
  round trip was probed: arrows, Enter into a focused field, Space reaching
  the text, return landing on the cell below with the arrows live).
  - The EDIT round trip on Compose and SwiftUI: a tap opens the field with
    the keyboard up (a `core.Focus` on an Input created in the same pass),
    return commits and the row below is *not* focused (no native acts on a
    focus command on a box), and whether the keyboard then stays up or drops.
  - The ✕ on a touch screen: that a tap on it does not blur the field first
    on either native, and that the 150ms grace is long enough in a real
    browser with a mouse (headless Chrome dispatches callbacks, not pointer
    events, so the race the grace exists for was reasoned, not seen).
  - The soft keyboard covering the active cell near the bottom of the grid:
    the cell is inside a `List` inside (with `MinWidth`) a horizontal scroll
    box, and what each host scrolls to show a focused field there is unknown.
  - A `List` inside a `core.Horizontal()` box on both natives: a lazy column
    under an unbounded width. `MinWidth` is the only thing that builds it.
  - A `core.Select` with its frame stripped (`BorderWidth(0)`, `Padding(0)`,
    transparent fill) inside a cell on both natives: the web obeys; the
    natives' pickers may keep their own chrome and make the row tall again.
  - TalkBack and VoiceOver: a cell heard as "Amount, row 2, $310.50, button",
    the editor's name, the `RoleAlert` message heard on a refused commit
    (TalkBack; VoiceOver is expected to say nothing), and the row menu.
  - The cost on a phone: the doc's "about 5,000 cells" is 4µs a cell measured
    on an M3, times a guess. Type into a 10 × 500 sheet on the Fold6.
- **N-073** · raised `2026-0921-1118-comps-round-four-phase-5-editable-grid` · value low (API decision)
  **Callback IDs are positional, and `EditableGrid` is the widget that pays.**
  IDs are issued in render order (core/event.go, `beginPass`), so entering or
  leaving EDIT, where the editor registers three void callbacks against the
  box's one, re-binds the `onClick` of every cell after it: a patch per later
  cell per transition. It is also the documented stale-event hazard made
  likelier: an event dispatched against the tree before the transition can
  hit a shifted ID. The fix is the one `beginPass`'s comment already names,
  identity-keyed IDs (a keyed node's callbacks named by its key path). Until
  then the grid could pad each cell to a fixed number of registrations, which
  was judged too ugly to do on the way past.
- **N-074** · raised `2026-0921-1419-tutorial-light-dark-theme` · value low (API decision)
  **No bundled `core.DarkTheme`.** The tutorial's `darkTheme` lives in
  examples/tutorial because core's palette censuses (the `*OnLight` tones,
  the control-boundary pairs) assume a light page; in that theme the
  `*OnLight` fields hold light inks. Promoting it means renaming or
  re-arguing those roles and adding it to `BundledThemes`. Its contrast
  figures were computed by hand, not by a census.
- **N-076** · raised `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror` · value medium
  **`PaddingLeft`/`PaddingRight` are leading/trailing on the natives and
  physical on the web.** Compose writes `padding(start = padding.left)` and
  SwiftUI `EdgeInsets(leading: padding.left)`, so both mirror under RTL; the
  runtime and htmlout write `padding-left`, which does not. Found while
  mirroring charts (N-071): `BarChart`'s horizontal value labels kept their
  gap on the wrong side on the web only, and were moved to a spacer Box in a
  Row (Rows mirror everywhere). Every other `PaddingLeft` in comps (TreeView's
  indent is a spacer already; `PaddingLeft(16*depth)` is the prop's own doc
  example) is still a web-only RTL defect. The fix is the web's: write
  `padding-inline-start`/`-end` (both targets, and cssstyle.mjs's shorthand
  table), or a decision that Left means left and the natives change.

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

- **N-071** · raised `2026-0921-1057-comps-round-four-phase-4-charts`
  · closed 2026-09-21, `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror` — `core.CanvasMirrorsRTL`, an opt-in Canvas prop
  (`mirror: true` on the wire, only when set), reflects the drawing about its
  box's centre under RTL. Web and htmlout: `scale: var(--grmob-inline, 1) 1`
  on the `<svg>`, with `core.TranslateDirectionCSS` added on first use.
  Compose and SwiftUI: the viewport mirrored (`core.MirrorCanvasMapping`,
  restated as `CanvasViewport.mirrored` / `GrMobCanvasViewport.mirrored`)
  when the layout direction is RTL, held to `internal/canvasfixture`'s three
  mirrored cases by both native harnesses. Adopted by Line/Area, Bar
  (both orientations), Histogram, Scatter, Candlestick, Radar, Heatmap,
  Sparkline, Waveform and Rating's half star; Donut, Pie, Gauge, Funnel and
  QRCode stay fixed (`comps/canvas_mirror_test.go` is the census). Seen
  right under RTL in headless Chrome (4.20, 4.36) and on the Compose emulator
  with a per-app Arabic locale (4.20, 4.36's radar); browser check 21 pins
  the reflection, mutation-tested. SwiftUI type-checks and its geometry is
  verified, but it is unrun.
- **N-067** · raised `2026-0921-1019-comps-round-four-phase-2-inputs`
  · closed 2026-09-21, `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror` — a `radiogroup` now answers both arrow pairs on the
  web (Down/Right forward, Up/Left back, the horizontal pair mirrored under
  RTL, whatever the group's axis), in `handleCompositeKey`. One-axis
  composites still leave the cross pair to the page. Pinned in
  `wasm/verify/keynav_test.mjs` (dom.mjs only; not tried in a real Chrome on
  lesson 5.9).
- **N-063** · raised `2026-0921-0912-comps-round-four-phase-1-chat-family`
  · closed 2026-09-21, `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror` — README's chapter table restated (chapters 4, 5, 6
  were 14/6/5, are 37/9/8), and `examples/tutorial/readme_counts_test.go`
  now holds the table and both "N lessons across M chapters" sentences to
  `Chapters`. The plan doc's per-corner-radius entry is struck through.
- **N-075** · raised `2026-0921-1419-tutorial-light-dark-theme`
  · closed 2026-09-21, `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror` — browser check 20 (`wasm/verify/browser.mjs`)
  boots the site page at 1280px on lesson 1.2 and holds each pane's
  background and caption ink to the scheme under System on a dark OS, a
  click on Light, a reload with Light remembered (the head script's
  attribute before `<body>`, and RenderInitial's first tree), System again,
  and an OS change with no reload. Mutation-tested: removing boot()'s
  sendTheme and removing the head script's attribute each fail it.

- **N-055** · raised
  `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`
  · closed `2026-0919-2146-labelled-group-echo-on-compose` — the Stepper said its value
  twice on Compose ("2, Guests, 2"). (was #77)
- **N-056** · raised
  `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`
  · closed `2026-0919-2146-labelled-group-echo-on-compose` — a Drawer panel read
  "Notebook, Notebook". (was #78)

  Both were one bug: a labelled node merges its descendants, and Compose emits
  the label as a fake child beside the Texts rather than in place of them, so
  any child Text repeating the label or the stated value was spoken twice.
  `LocalGrMobGroupSaid` in `android/…/Renderer.kt` carries the nearest labelled
  ancestor's label and value text, and `GrMobText` clears the semantics of a
  Text that folds to either. Measured on the emulator's accessibility tree:
  the Stepper group's `'2'` TextView child and the panel's `'Notebook'`
  TextView child are both gone. Not the same fix as `LocalGrMobNamedControl`,
  which silences a *control's* whole content; a group's content stays readable
  apart from the echo.

- **N-026** · raised `2026-0917-0227-foldables-window-record-and-two-pane`
  · closed `2026-0919-2303-safe-insets-record-inert-codeeditor-sse-cleanup` — safe-area insets are a record. `core.SafeInsets`
  (`Top`/`Bottom`/`Left`/`Right`) is a field on `core.Window` rather than a
  record of its own: `Window` is `==`-comparable by design and dedupes on
  that, so insets ride the same `"window"` host event, the same dedupe and
  the existing `hooks.UseWindow` with no second pub/sub. Invalid insets are
  dropped on their own and the size kept, the same stance an unknown fold
  gets. Android reports `systemBars | displayCutout` — `safeDrawing` minus
  the IME, so the numbers match what its own `SafeArea` node applies — on an
  additive `OnGlobalLayoutListener` rather than
  `setOnApplyWindowInsetsListener`, which would displace the inset chain
  edge-to-edge and Compose depend on. iOS reads the root `GeometryReader`'s
  `safeAreaInsets`, which are the window's because the reader already
  ignores the safe area, and watches them as well as the size. The browser
  sends nothing (see N-061). Unrun on a device. (was #35)
- **N-059** · raised
  `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`
  · closed `2026-0919-2303-safe-insets-record-inert-codeeditor-sse-cleanup` — the premise was inverted, the leak was real. On this
  Compose version a nearer `focusProperties` does *not* beat an outer one:
  `fetchFocusProperties` walks up and the outermost wins, so RenderNode's
  head-of-chain placement was already right. What it cannot do is *reach*
  the CodeEditor's field — the walk takes `untilType = Nodes.FocusTarget`
  and stops at the first one it meets, and the field sits behind two scroll
  boxes that each delegate a focus target. An *editable* CodeEditor in a
  shut Drawer panel therefore stayed a hardware-keyboard Tab stop (a
  read-only one was saved by its own gate answering no). Fixed by reading
  `LocalGrMobInert.current` in the editor and conjoining it into the gate:
  `canFocus = !inert && (!readOnly || gate.open)`. Pinned by
  `TestComposeCodeEditorHonoursAnInertAncestor`. Still no bundled Inert
  subtree holds a CodeEditor, so it is unrun on a device. (was #81)
- **N-060** · raised `2026-0919-1421-rweb-serve-and-doctor-dev-server-check`
  · closed `2026-0919-2303-safe-insets-record-inert-codeeditor-sse-cleanup` — no upstream hook was needed after all. The constraint
  was self-imposed: `subscribe` queued the "hello" *into* the channel before
  registering it, which is what ruled out `SSEHub.Handler` and with it
  RWeb's on-close cleanup. Registering first and *broadcasting* the hello
  second keeps the same ordering guarantee — `mu` is held across both steps
  and across every broadcast, so no build can land in between — and lets the
  channel come from `Handler`, which unregisters it the instant the stream
  ends. The lingering channel and the "SSE Channel closed and drained"
  stdout line both go with it (the line came from the eviction closing a
  channel `sendSSE` was still reading; now the close happens after it has
  returned). The price is that hello reaches every open page, so
  `devclient.js` ignores a hello that tells it nothing.

Closures before the seed are written up in the session docs; the
most recent are in `2026-0919-1254-…` (#5, #62, #71, #75) and
`2026-0919-1118-…` (#7, #25, #70, #72, #73).
