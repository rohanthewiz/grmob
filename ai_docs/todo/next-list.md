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

**Next ID:** N-098

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
  - 2026-09-29: bare Escape joins it (N-057): XCUITest's Escape reaches no
    SwiftUI keyboardShortcut on the simulator, so core.OnEscape on iOS, and
    whether a Modal closes on it there, wait on the same keyboard.
- **N-005** · raised `2026-0914-2319-next-list-row-hug-box-chords-and-a-strip-that-divides` · value low
  **iOS chords.** Page-global chords were verified once, and the chord gate
  (behind a modal, inside a shut Drawer panel) is unheard. (was #6;
  lapsed@0917-1659)
- **N-006** · raised `2026-0916-1032-clocks-canvas-alarm` · value low
  **Alarm sound and haptics are unheard,** and so is the Notify banner's
  default sound. Needs a person holding a phone. (was #9; lapsed@0917-1659)
- **N-009** · raised `2026-0916-1032-clocks-canvas-alarm` · value low
  **A Notify alarm is a banner, not a ringing screen.** The route is AlarmKit
  or full-screen intents. (was #15; lapsed@0917-1659)
- **N-013** · raised `2026-0916-1157-next-list-charts-on-devices-tier-e-alarms-timezone` · value low (API decision)
  **`mobile.SetTimeZone` runs once at startup.** Fixing it needs core to hold
  the location. (was #16; lapsed@0917-1659)
  - Recommendation (2026-09-29, not acted on): leave it. The fix is a zone
    core holds (`core.Local()`, set from a "timezone" host event) and every
    widget reading it instead of `time.Local`, but app code calling
    `time.Now()` would still see the launch zone, so the result is two clocks
    that can disagree. A zone change mid-run is rare; relaunch picks it up.
    Propose as a non-goal.
- **N-015** · raised `2026-0916-1229-next-list-exact-alarms-boot-rearm-and-sweeps` · value low (user's decision)
  **Compose Rows don't shrink children in proportion.** Seen on the Fold6's
  cover screen. (was #18; lapsed@0917-1659)
  - Recommendation (2026-09-29, not acted on): leave Compose as it is.
    Proportional shrink needs a custom Row measure policy (the flex solver
    iOS already has), a large change for one cover-screen look; comps that
    must fit already pin with `FlexShrink(0)` or wrap.
- **N-021** · raised `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups` · value medium (blocked)
  **The iOS Image floor runs high for a narrow image.** Needs a px-width box
  that can shrink (`grMobDimension`). (was #26; lapsed@0917-1659)
- **N-024** · raised `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups` · value low
  **`barValueRoom` is still an estimate.** Exact needs host measurement of the
  plot. (was #30)
- **N-027** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **Lesson 4.21's TwoPane sets no `Origin`.** No longer waits on N-026:
  4.21 sets `IgnoreHorizontalFold`, so its live axis is the vertical hinge
  and the missing term is `Origin.X` — the demo panel's own left padding, a
  layout constant no host reports. A correct `Origin` there would be a
  hard-coded guess; showing the documented `Origin{Y: insets.Top + bar}`
  case honestly needs a non-scrolling demo. Measured in the browser
  (2026-09-22, browser check 23's setup): under a vertical hinge at x 390 the
  seam lands at x 445, the demo panel's 55px of inset. (was #36;
  lapsed@0917-1659)
  - Recommendation (2026-09-29, not acted on): decline. The lesson's prose
    already names the missing term; a hard-coded Origin.X would teach a guess.
- **N-028** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **iOS `AppWindowReader` is type-checked only.** Not run in Split View or
  Stage Manager. (was #37; lapsed@0917-1659)
- **N-030** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **Folding shut onto the outer display** needs Samsung's "Continue apps on
  cover screen". Unseen. (was #39; lapsed@0917-1659)
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
  - whether `setCollectionItemInfo` still overwrites a stated
    `collectionItemInfo` under a `selectableGroup` (N-077's second defect).
  - Re-checked 2026-09-25: BOM `2024.12.01` resolves foundation and ui to
    **1.7.6** (`gradlew :app:dependencies`, debugRuntimeClasspath), so the
    premise holds. N-077's closure and the comment at `Renderer.kt:349`
    call the overwrite a "Compose 1.10" defect, but it was heard on the app's
    1.7.6. The 1.10.0 jars in the Gradle cache are not on the classpath.
- **N-049** · raised `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep` · value low
  **A below-the-fold sweep on Android 10.** Needs the Mi Max 3 unlocked, or a
  person scrolling (or an API 29 emulator image). The HID keyboard
  (2026-0919-1254 §1) may drive it. (was #65)
- **N-051** · raised `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep` · value low
  **Why the decor-view force-dark flag stopped holding after an AndroidView
  attached is undiagnosed.** Moot for this app. (was #68)
- **N-058** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls` · value medium
  (Raised from low 2026-09-25: it now blocks the emulator half of N-062's
  Poll, N-068's TreeView and Wizard checks, and N-072's TalkBack pass. The
  Fold6's HID harness is the known route round it.)
  **TalkBack does not follow Tab onto 4.18's ☰.** Compose's focus is right
  (TalkBack off shows it), but TalkBack said "Showing Inbox" or stayed on the
  previous node. (was #80)
  - Seen again on the emulator (2026-09-22): on 4.34 most Tabs spoke nothing
    and Enter under TalkBack pressed a different node from the one just
    spoken (a Copy button); on 4.16 TalkBack's Tab read prose and code that
    Compose's own focus never visits. Controls were reachable by moving
    Compose's focus with TalkBack off, turning TalkBack on and stepping once.
  - Blocked the Wizard check (2026-09-23, 4.35): with Compose's focus on the
    wizard's Skip and TalkBack then turned on, Enter pressed the page's
    "‹ Contents"; an earlier Enter on Next advanced the step with nothing
    logged, so whether it was Next that Enter pressed is unknown. Injected
    taps do not reach the emulator's touch explorer either.
- **N-062** · raised `2026-0921-0912-comps-round-four-phase-1-chat-family` · value medium
  **Phase 1's chat widgets, unrun on a device.** Lesson 4.34 and
  `examples/chat` were looked at in headless Chrome only.
  - `TypingIndicator` under Reduce Motion: the hosts drop the `Transition`
    and Go still steps the phase. Seen on Compose (2026-09-23, the emulator
    with all three animation scales at 0): 30 raw screencaps of the three
    dots' centres read exactly two values, one dark dot at a time, so the
    quiet blink the doc claims is what draws. The web, seen 2026-09-28 in
    headless Chrome on 4.34 (a scratch CDP probe, per-frame computed opacity
    for 3s): each dot passed through about 20 values with a 0.3s transition,
    and exactly two, 0.4 and 1.0, with the transition at 0s under an emulated
    `prefers-reduced-motion: reduce`. SwiftUI, seen 2026-09-28 on the iPhone
    17 Pro simulator (a scratch XCUITest taking about 75 screenshots in 6s,
    the dots' centre pixels read offline): 17–18 luminance values between
    rest (145) and full (0) normally, and exactly two, 145 and 0, with
    `com.apple.Accessibility ReduceMotionEnabled` set in the simulator
    (turned back off after). All three hosts now draw the quiet blink.
  - That a `core.Opacity` `Transition` on a 6pt `Box` eases: seen on
    Compose (2026-09-23): at normal scale the dots' centre pixels passed
    through 34, 47, 59, 73, 87, 111… between rest (145) and full (4–9).
    SwiftUI unseen. Lesson 4.34's prose still said the fade was the
    background colour because "core.Style has no opacity"; it now names
    `core.Opacity` and the reduced-motion blink.
  - `TypingIndicator`'s `RoleStatus` appearing from `Display none`: heard on
    TalkBack? VoiceOver is expected to say nothing (the known live-region
    gap, core/role.go).
  - `ReactionBar` chips: emoji glyphs drawn in a `Button` label on both
    natives. The selected state is done: heard on the emulator's TalkBack
    (2026-09-22) as "Selected, thumbs up, 3 reactions, Button" and, after a
    tap, "Selected, party popper, 2 reactions, Button"; on the iOS simulator
    the tapped chip reads "party popper, 2 reactions" and isSelected
    (`TutorialRoundFourUITests.testReactionChipReportsItsSelection`).
  - `Poll` results: one stop per option, with the hidden `ProgressBar` and
    texts not reachable by swipe. Not reached by the emulator's Tab sweep of
    4.34 (N-058's gap), so still unheard.

- **N-064** · raised `2026-0921-0935-core-opacity-and-typing-indicator-fade` · value medium
  **`core.Opacity` is unrun on a device.** It
  compiles on all four targets and its call sites, layer order and sentinel
  are pinned from source, but nobody has seen it fade.
  - Compose: that `Modifier.alpha` outside `Modifier.shadow` keeps the whole
    shadow mid-fade. An alpha below 1 composites the layer offscreen, and an
    offscreen layer may clip what is drawn outside the node's bounds.
  - Compose: `animatedStyle` easing the alpha, and snapping under "Remove
    animations": both seen on the emulator (2026-09-23) through
    TypingIndicator's dots; see N-062. The offscreen-layer shadow question
    above is still open.
  - SwiftUI: the fade under the node's one `.animation` is seen
    (2026-09-28): TypingIndicator's dots pass through 17–18 values on the
    simulator, and snap under Reduce Motion (see N-062). Whether a view at
    exactly 0 still takes taps and VoiceOver focus is still unmeasured (the
    doc says taps stop; that is from SwiftUI's known behaviour).
  - Fixed (2026-09-21): Compose's `DisplayHidden` alpha used to sit at the
    foot of `boxModifier`, inside the background and border, so a hidden node
    with a fill still drew the fill. It now shares the Opacity layer
    (`layerAlpha`), as iOS's `.opacity` already did; pinned from source by
    `TestComposeHidesTheWholePaintedBox`. Compiled, not watched.
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
  - iOS: done on the simulator (2026-09-21). `write`'s arithmetic is
    `carryPlan` in GrMobTextEdits.swift, run by `ios/verify` against
    `CarryCases` (the caret, and that the span ends at the caret when the
    caret follows the change; both mutation-tested). It gained Carry's
    surrogate guard. `testMaskedInputFormatsAsItIsTyped` types 5.9's phone
    mask in one burst ("(555) 123-4567"), refuses a letter, backspaces
    through the group break, and types the card mask a key a second. Not on
    a real iPhone.
  - The known limit (a key typed mid-text at the end of a group leaves the
    caret after the reflow) is the same on all three hosts by construction;
    seen on Android only.
- **N-066** · raised `2026-0921-1019-comps-round-four-phase-2-inputs` · value medium
  **Phase 2's inputs, unrun on a device beyond a look.** Lesson 5.9 was
  looked at in headless Chrome and on the Android emulator (pad and
  swatches drawn right; two pad keys tapped).
  - `NumberPad`: the haptic per key felt on a phone; the unpainted corner
    (`Opacity(0)`, `Disabled`, hidden) skipped by VoiceOver. XCUITest's tree on
    the iOS 26.5 simulator (2026-09-29) lists that corner as a disabled
    Button labelled " " despite Go sending `AccessibilityHidden` (and the
    Button's box keeps the flag through `marginAndSizeOnly`); XCUITest's tree
    is not VoiceOver's order (it also lists the lock screen's hidden dots),
    so whether VoiceOver stops there is still Accessibility Inspector's
    question. On Compose it
    is not a Tab stop (2026-09-21) and has no node at all in the
    accessibility tree (2026-09-23, uiautomator: the bottom row holds 0 and
    Delete only), so TalkBack cannot land on it.
  - `ColorSwatchPicker`: the ring and check on SwiftUI, seen 2026-09-28 on
    the simulator (the selected blue swatch ringed with a gap and a white
    check, the other seven plain, the custom well and hex field below);
    VoiceOver still unheard. Done (2026-09-22): TalkBack
    says "Selected, blue, Radio button, 1 of 8" and every position right
    since N-077; the hex field's return commits the short form on both
    natives ("#2a7" leaves Value at #2A78D6 while typed and commits #22AA77
    on return: the emulator, and `testSwatchesHexShortFormAndTheRangeThatCannotCross`
    on the simulator).
  - `RangeSlider`: done on both simulators' input paths (2026-09-22): an
    injected drag of Minimum from $20 to past $80 on the emulator left both
    thumbs and readouts at $115; XCUITest's adjust on the simulator left the
    two sliders' values equal. A real finger is untried. Its values are now
    the readouts (N-079, closed); the combine over its group is N-078.
  - The lock-screen dots' `RoleStatus` line: heard on the emulator's TalkBack
    per key (2026-09-22), "Passcode, 1 of 4 entered", then 2 and 3.
    VoiceOver is expected to say nothing (the known live-region gap,
    core/role.go).
  - Heard on the emulator's TalkBack (2026-09-21, Tab sweep): pad keys as
    "2, Button" … "Delete, Button"; swatches as "Not selected, orange, Radio
    button" (the selected one was not reached). Compose's Tab order goes
    9 → 0 → Delete, so the unpainted corner is not a Tab stop. The swatches'
    positions are wrong: see N-077.
- **N-068** · raised `2026-0921-1035-comps-round-four-phase-3-structure` · value medium
  **Phase 3's structure widgets, unrun on a device.** Lesson 4.35 was looked
  at in headless Chrome only (tree alignment, indents, the wizard's footer).
  - `TreeView`: a branch heard as "docs, collapsed, button" and the chosen
    leaf as selected on TalkBack and VoiceOver; that a `listitem` Box holding
    a button and a nested list is walked in order by swipe; that the level,
    inert on both natives by design, does no harm there. On the web, done
    as far as the browser (2026-09-23): headless Chrome's accessibility tree
    on 4.35 gives guide.md and api `listitem` level 2 inside docs' level 1
    (scratch CDP probe, not a browser check). A screen reader's speech is
    unheard.
  - `TreeView` under RTL: seen mirrored in headless Chrome and on the Compose
    emulator (per-app Arabic locale), 2026-09-21, and on the iOS simulator
    (2026-09-22, `testTreeViewUnderArabic`; `-AppleLanguages (ar)` alone did
    not flip the app, the two forced writing-direction defaults did). On all
    three the collapsed chevron "▸" still points right, against the
    reading direction; a glyph has no way to mirror (an API decision: a
    mirror-under-RTL prop for Text, like `CanvasMirrorsRTL`).
  - Heard on the emulator's TalkBack: "expanded. docs. Expands or collapses
    the branch, Button", "Selected, guide.md, Button", "collapsed. api. …".
    TalkBack skipped src, assets and README.md on Tab although Compose's own
    focus visits all six rows (N-058's gap).
  - `Wizard`: the `RoleStatus` line heard on a step change on TalkBack (tried
    on the emulator 2026-09-23 and blocked by N-058; the Fold6's HID harness
    is the route) and in a browser's screen reader; where TalkBack's focus lands after Next (on the web and iOS it is
  now the title, N-069; Compose cannot move it)
    replaces the body (the keyed body is a replacement, and Compose clears
    View focus when the focused node leaves: 2026-0919-1254 §2).
  - `Wizard.Footer()` in `Screen.Footer` above the keyboard is N-002's
    pinning check with a consumer now; no bundled screen does it yet.
- **N-070** · raised `2026-0921-1057-comps-round-four-phase-4-charts` · value medium
  **Phase 4's charts, unrun on a device.** Lesson 4.36 was looked at in
  headless Chrome only (all four demos; the look found two defects, fixed).
  - Seen on the emulator and the iOS simulator (2026-09-22,
    `testRoundFourChartsDraw`): `Waveform`'s bars are round-capped and even
    on both (the stroke is unscaled under `CanvasStretch`); the doji draws;
    the funnel draws. The half-px silent bar as a dot was not looked for.
  - `RadarChart` on SwiftUI was drawn 62pt left of centre. Fixed
    (2026-09-23): the iOS ZStack layout reported its largest layer (the
    180pt canvas) and the `.frame(width: 304)` round it placed that at its
    leading edge, 62pt = (304 − 180) / 2 short. `GrMobStackSolver.containerSize`
    now takes `fills:` and reports the offer on an axis the box is sized on
    (a stated Width/Height or a fill), as Compose and CSS size the box
    first; checked by ios/verify's stack section and pinned by
    `TestNativeZStackOverlaysItsChildren`. On the simulator the chart's
    image is centred on the panel (x 82.8 + 236.7/2 = 201) and "Vision"
    reads whole (`testRoundFourChartsDraw`'s shot). All 48 Tutorial UI
    tests pass with it.
  - `AudioPlayer.Waveform` with a real stream: the strip filling as the
    status ticks, and under the finger while scrubbing.
  - Every chart's one spoken sentence on TalkBack and VoiceOver.
  - 2026-10-03: iOS charts now carry an `AXChartDescriptor` (N-010), which
    type-checks and survives the Release build but has not been opened in
    VoiceOver's chart details or played as an Audio Graph.
- **N-072** · raised `2026-0921-1118-comps-round-four-phase-5-editable-grid` · value medium
  **`EditableGrid`, unrun on a device.** Lesson 4.37 was looked at and driven
  in headless Chrome only (the look found two defects, fixed; the keyboard
  round trip was probed: arrows, Enter into a focused field, Space reaching
  the text, return landing on the cell below with the arrows live).
  - Done on the emulator and the simulator (2026-09-22): a tap opens the
    field with the keyboard up, return commits (Undo (1)), and the ✕ throws
    a typed draft away without the blur committing it first (Undo stays
    (1)). After return the keyboard drops on Compose and stays up on iOS.
    `testGridEditRoundTripAndDiscard` holds the simulator half. The 150ms
    grace, seen in headless Chrome with CDP's real mouse events (2026-09-28):
    a 60ms and a 120ms press on the ✕ discard the draft; a 300ms and a 600ms
    press commit it. That is N-084.
  - The horizontal box was a `core.Box(core.Horizontal())`, which neither
    native scrolls (only a browser reads `overflow: auto`): on the simulator
    the lesson page grew to 490pt on a 402pt screen with every paragraph cut
    at both edges, and on the emulator the grid was squeezed with Amount
    clipped. Now a `core.Scroll`, and Compose's strip gives a grower with a
    points MinWidth a definite width (`GrMobGrowStrip`, pinned by
    `TestComposeStripGrowerWithAFloorIsMeasuredAtAWidth`), so the rows keep
    their weights: seen aligned and scrolling sideways on the emulator, the
    simulator and in headless Chrome.
  - The frameless `core.Select` in a cell keeps the row at the other cells'
    height on both natives. The iOS row in EDIT that drew its Category text
    about 3pt left is fixed (2026-09-23): a zero-basis child's base was its
    padding alone (`zeroBasisPadding`), and the editing cell trades 2pt of
    padding a side for its 2pt ring, so it started 4pt short and the weights
    moved the rest of the row. It now reads `contentInsets` (padding plus a
    drawn border, as CSS counts a basis); pinned in
    `TestIOSFlexHonoursAZeroBasis`, and seen aligned on the simulator.
  - The soft keyboard covering the active cell near the bottom of the grid:
    judged on Compose (2026-09-28, the emulator with
    `show_ime_with_hard_keyboard` 1, restored to 0 after): a tap on row 4's
    Item cell, 1870–1967px on a 2400px screen, brought the editor up to sit
    just above Gboard's suggestion strip, text and ✕ whole. On the iOS
    simulator row 2's editor sits clear of the keyboard. Row 4 (the last) was
    tried 2026-09-28 and opens its editor, but that simulator had a hardware
    keyboard connected, so the soft keyboard was off screen (its frame at y
    952 of 874) and covered nothing: still unjudged on iOS.
  - On iOS the grid is one static text "Budget" to VoiceOver, and its text
    cells are not in the accessibility tree at all: N-078.
  - TalkBack and VoiceOver (VoiceOver now waits on N-078): a cell heard as "Amount, row 2, $310.50, button",
    the editor's name, the `RoleAlert` message heard on a refused commit
    (TalkBack; VoiceOver is expected to say nothing), and the row menu.
    Heard on the emulator (2026-09-21): "Amount, row 2, $310.50. Edits the
    cell, Button" and "Row 1. Opens the row's menu, Button". A Category cell
    (the frameless `core.Select`) says "Category, row 1, Home" with no role.
  - The cost on a phone: the doc's "about 5,000 cells" is 4µs a cell measured
    on an M3, times a guess. Type into a 10 × 500 sheet on the Fold6.
- **N-089** · raised `2026-1003-1659-n007-canvas-text-clip-shape-taps` · value medium
  **`ios/verify/run.sh` does not compile.** `GrMobSurface.swift` (N-085,
  `6fda8ec`) imports UIKit, and the harness builds the runtime's files with
  the macOS `swiftc`, so it stops at `no such module 'UIKit'`. Fails the same
  way on a clean checkout of HEAD. Every check the harness holds (canvas
  mapping, stack, text edits, the widget census) is unrun until it builds.
- **N-090** · raised `2026-1003-1659-n007-canvas-text-clip-shape-taps` · value low
  **Mirrored canvas text under RTL is unseen on the natives.** Compose and
  SwiftUI swap start and end for a `CanvasMirrorsRTL` canvas laid out right
  to left; only Chrome was checked (a static page with `dir="rtl"`). Lesson
  4.19's week chart does not mirror, so the tutorial never reaches the path.
- **N-091** · raised `2026-1003-1852-stripe-checkout-bible-verse-discussion-widgets` · value medium
  **StripeCheckout, BibleVerse and Discussion are unseen on the natives.**
  They were checked with debug-mode tests, `AuditTree` and one headless
  Chrome screenshot of an `htmlout` export. Nobody has looked at them on
  Android or iOS. The parts most likely to differ: Discussion's labelled
  header rows ("Ben, reply to Ana, 1h"), which exist for the natives' sake;
  the 2px thread line stretched with `AlignSelf`; and the lock glyph and Like
  chip sizing.
- **N-092** · raised `2026-1003-1852-stripe-checkout-bible-verse-discussion-widgets` · value low
  **`blb` has no proxy for browser builds.** BLB's ScriptTagger feed sends no
  CORS headers, so a wasm app must set `blb.Client.BaseURL` to a same-origin
  proxy that it writes itself. Candidate: a small handler in `webhost` or
  `serve` that forwards `/remoteExtensions/toolTip/toolTipRemote.cfm` to
  www.blueletterbible.org. Contingent on a browser app wanting verses.
- **N-096** · raised `2026-1007-0210-n094-headless-window-minimum-not-overflow` · value low
  **`htmlout.ExportHTML` writes no viewport `<meta>`.** Opened on a phone,
  or under CDP mobile emulation, an export lays out at the 980px desktop
  fallback and is drawn zoomed out. Measured 2026-10-07: `innerWidth` 980
  for the export, 420 for the same tree in the `grmob new` host page, which
  has `width=device-width, initial-scale=1, viewport-fit=cover`. The export
  writes a `<head>` only when the tree needs a motion or border-box rule
  (`motionStylesheet`), so adding the meta changes every export's bytes and
  makes the head unconditional. Candidate: an always-present head with the
  same meta the scaffold uses.

## Validate

- **N-097** · raised `2026-1010-1728-keystore-binding` · value low
  **The keystore is unrun on physical hardware.** Checked with a throwaway
  probe on the API 36 emulator and the iOS 26.5 simulator. Covered: save,
  read, overwrite, a non-ASCII value, an empty value, a double delete, a
  20-call burst, persistence across relaunch, a tampered entry, a simulated
  lost key (a renamed alias), and iOS items outliving uninstall until the
  fresh install's wipe. Unchecked:
  - A hardware-backed key: the Fold6's TEE/StrongBox, and the Mi Max 3 on
    Android 10, the oldest Keystore GCM-with-AAD in reach.
  - A real Auto Backup restore (`bmgr`) instead of the renamed alias.
  - An iOS background launch before first unlock, which should answer
    errSecInteractionNotAllowed (-25308) rather than hang.

  church_mobile's N-004 swap is the natural first real consumer.

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
- **N-011** · declined `2026-0916-1129-charts-on-canvas` — Chart summaries
  are English. Every chart's `AccessibilityLabel` already replaces the
  summary, so an app that needs another language states its own. Declined by
  the user 2026-09-25. (was #13)
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
- **N-031** · declined 2026-10-04 · raised `2026-0917-0227-foldables-window-record-and-two-pane`
  — **Housekeeping.**
  - The `GrMob_Foldable` AVD is still installed. Delete it? (User's call.)
  - The Mi Max 3 still has `stay_on_while_plugged_in` 7 (was 0) and
    auto-rotate off (was on).
  - The emulator's GrMob app has POST_NOTIFICATIONS and SCHEDULE_EXACT_ALARM
    granted.
  - New: the Fold6's Samsung TalkBack now has READ_PHONE_STATE granted (was
    denied), kept for the harness; "Display speech output" is still on. (was
    #40; lapsed@0917-1659)
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
- **N-061** · raised `2026-0919-2303-safe-insets-record-inert-codeeditor-sse-cleanup`
  · declined 2026-10-03, `2026-1003-1559-n061-n088-safe-area-insets-on-the-web` — the browser reports no safe-area insets, by design.
  Zero is the true answer for a page in a browser window; the one case it is
  wrong (a page drawn behind a notch with `viewport-fit=cover`, typically an
  installed PWA) would cost a probe element and a `getComputedStyle` per
  report, and that page owns its HTML: one CSS padding of
  `env(safe-area-inset-*)` on the element hosting the app keeps the content
  clear with no number reaching Go. Recorded in `core.SafeInsets`' doc
  comment (and docs/api/core-device.md) and in the runtime's windowMetrics
  comment. The tutorial page that does set cover is N-088.

## Closed

- **N-095** · raised `2026-1003-2008-n093-gallery-lessons-checkout-verse-discussion`
  · closed 2026-10-07, `2026-1007-0216-n095-progress-track-border-role` — confirmed and fixed in `comps.ProgressBar`, not
  the tutorial. The default track was `Colors.Surface`, and DarkTheme's
  Card fill *is* Surface (#2C2C2E), so a stock bar in a stock dark Card had
  a 1:1 track: absent, not faint. The default is now
  `Colors.BorderColor()`, which `comps.Gauge`'s track already used. Measured
  against every `palette.Backdrops` fill of every bundled theme, the old
  default was 1.00:1 on each theme's Surface, on DarkTheme's Card and on
  Amber's Input and TextArea. Border's lowest is 1.13:1 (DefaultTheme on
  Surface). `TestProgressBarDefaultTrackShowsOnEveryBackdrop` holds the
  default to a 1.1:1 floor on every pair; it fails 7 pairs with the old
  default. Seen before and after in headless Chrome at a pinned 420px: the
  tutorial's card shape under DarkTheme, runtime-mounted. At 0% the old
  default drew the caption over nothing, and the new one draws a full-width
  #38383A groove. Light themes' tracks get darker too (DefaultTheme #F2F2F7
  → #E5E5EA), so `docs/images/tutorial-contents.png` is now a shade off; it
  was not retaken. `wasm/verify/run.sh` and `go test ./...` pass.

- **N-094** · raised `2026-1003-1852-stripe-checkout-bible-verse-discussion-widgets`
  · closed 2026-10-07, `2026-1007-0210-n094-headless-window-minimum-not-overflow` — not a bug: the premise was a headless Chrome
  artifact. `--headless=new --window-size=420,900` lays the page out at
  Chrome's minimum window width, 500px (`innerWidth` 500), while
  `--screenshot` saves a 420px-wide PNG of it. That crop cuts off the right
  80px, which is what looked like overflow. Re-run on 2026-10-07 with one
  `comps.Screen{Scroll: true}` holding a stock `comps.Card` and
  `comps.InputRow`. It was rendered both as `htmlout.ExportHTML` and as the
  runtime's JSON mounted by `GrMob.mount` in the `grmob new` host page, then
  measured over CDP with `Emulation.setDeviceMetricsOverride` pinning a true
  420px viewport. Both reported `scrollWidth` 420 and no element past the
  right edge, and the screenshot shows both widgets inside the frame. Real
  differences seen, neither an overflow: the export keeps `<body>`'s
  default 8px margin, and it has no viewport `<meta>` (raised as N-096). A
  check at a phone width should pin the viewport over CDP, as
  `wasm/shots/shot.mjs` does with its clip, not rely on `--window-size`.

- **N-087** · raised `2026-1003-0746-component-defined-readme-docs-skill`
  · closed 2026-10-07, `2026-1007-0204-n087-docsnippets-compile-component-docs` — `internal/docsnippets`. Its test takes the ```go
  fences under a fixed table of headings in the README ("Your own
  components"), `docs/concepts/components.md` (leaf widget, owned state,
  accessibility, testing) and `ai_docs/SKILL-component.md` (§2, §6, §8). It
  writes one package per doc into a scratch module that `replace`s grmob with
  this checkout, then runs `go vet` and `go test -v` there. A fence with no
  package clause gets one plus imports for the qualifiers it uses. The concern
  snippet is pasted into a method body on Tally. The concept page's
  `findFirst`/`findText`, which its prose describes but does not show, are
  supplied as `given_test.go`. `//line` directives make errors name the
  markdown line. The table's fence counts fail a renamed heading, and every
  `func Test…` a fence declares must PASS once per doc. Mutation-checked:
  a renamed symbol failed at `components.md:310`, an inverted reveal at
  `SKILL-component.md:436`, and a renamed README heading failed the count. It
  takes about a second. Not behind `-short`. Both docs now say the check
  exists. Also bumped wasm/verify's tracked-Go-file figure from 688 to 691;
  the N-083 commit's `hbox_test.go` had already left it stale.

- **N-083** · raised `2026-0924-1211-tutorial-overflow-sweep-web-floors-nested-scroll`
  · closed 2026-10-07, `2026-1007-0157-n083-hbox-paddingless-row` — added `core.HBox`, a Row with no theme base,
  as the horizontal twin of `core.Box`. The 2026-09-29 recommendation was to
  decline, on the grounds that a paddingless constructor would double the
  stack API. But `Box` has always been the paddingless Column, so only the Row
  half was missing. `HBox` emits an ordinary `Row` node: the theme base is
  applied in Go by the constructor and no renderer looks one up by type, so no
  target changed. The theme default stays as it is, so no app's layout moves.
  The existing `Padding(0)` sites were not migrated. An AST count found 66
  `core.Row`/`core.Column` calls with a literal `Padding(0)` argument (22 in
  examples/tutorial, 42 in comps) out of about 435, plus slice-built ones it
  does not see. The grep counts are larger (42 lines in examples/tutorial,
  256 in non-test Go) because they include other nodes. Pinned by
  `TestHBoxIsARowNode` and `TestHBoxCarriesNoThemeBase`. Row and Column
  gained doc comments pointing at HBox and Box. `docs/concepts/views.md`,
  `docs/concepts/components.md` and both `ai_docs/SKILL*.md` mention it, and
  `docs/api` is regenerated. Not seen on a device, since no renderer changed.

- **N-022** · raised `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups`
  · closed 2026-10-04, `2026-1004-2022-n022-zero-basis-two-pass-measure-ios` — the two-pass measure, in `GrMobFlexLayout`
  (Renderer.swift). The real gap at placement was a zero-basis Column child
  whose `min-height: auto` is the `.infinity` verdict: it kept its measured
  content height as its base, so a weighted Column was content-biased.
  `minMains` now measures that child's content and passes it as the minimum,
  so `baseMains` can start it at its padding. Under an ideal-size query,
  `sizeThatFits` still sizes the container from content (pass 1), then lays
  the children out inside that length from zero bases (pass 2, `zeroBased`),
  so the cross size is measured at the mains placement draws. Seen on the
  iPhone 17 Pro simulator with a throwaway app (deleted): a 300pt Column
  with two weight-1 zero-basis cells, 1 and 4 lines tall, went from 247/384px
  to 315/316px. Pass 2 made no visible difference there, because the outer
  Row measures the inner one again at a definite width; no screen was found
  that shows it. `TestIOSFlexHonoursAZeroBasis` pins both halves.
  `TutorialZeroBasisAndGradientsUITests` (calendar, StatTiles, gradients)
  and `TutorialChartsUITests` pass. Not seen on Android or the web, which
  were not affected. (was #27; lapsed@0917-1659)

- **N-093** · raised `2026-1003-1852-stripe-checkout-bible-verse-discussion-widgets`
  · closed 2026-10-03, `2026-1003-2008-n093-gallery-lessons-checkout-verse-discussion` — lessons 4.38 "Checkout with Stripe", 4.39 "Quoting
  a Bible passage" and 4.40 "Threaded comments", each with a test driving it
  through the app. 4.38 plays Stripe's page and the server's answer with two
  buttons (paid / declined) and shows the same minor units in USD, JPY and
  KWD. 4.39 uses three canned KJV passages in `blb.Passage`'s shape, with
  segments playing the fetch (loaded / loading / failed); the tutorial does
  not import `blb`, which would add net/http to the wasm build. 4.40 is a
  working thread with MaxDepth 2. Counts moved to 83 lessons and 40 in
  chapter 4 (README, `docs/tutorial-interactive.md`, `wasm/index.html`,
  `internal/shotclaims`), and `tutorial-contents.png` was retaken. Two
  fixes the work turned up: `comps.Discussion`'s indented replies column
  now has `MinWidth("0")` (a Composer one level down pushed the whole level
  past a 414px phone's edge on the web), and `wasm/shots/shot.mjs` emulates
  `prefers-color-scheme: light`, since the tutorial follows the system scheme
  and a dark-mode Mac took a dark shot. Seen in headless Chrome at 414px.
  Not seen on Android or iOS.

- **N-007** · raised `2026-0916-1032-clocks-canvas-alarm`
  · closed 2026-10-03, `2026-1003-1659-n007-canvas-text-clip-shape-taps` — all three added to `core.Canvas`:
  `Shape.Text` (`core.CanvasText`: anchor in viewBox units, size in layout
  units, start/middle/end × top/middle/bottom, start following a mirrored
  drawing's reading direction), `Shape.Clip` (a path, nonzero) and
  `Shape.OnClick`. Taps are hit-tested in Go (`core/canvas_hit.go`): hosts
  report "x,y,w,h" in the box to one `onShapeTap` text callback, shapes are
  tried topmost first and only those with a handler take part, a miss runs
  the canvas's own OnClick. Web: `<clipPath>` in the leading `<defs>`, text
  as `<g clip-path><text>` counter-scaled by `--grmob-canvas-ix/-iy` from a
  ResizeObserver (htmlout's static export leaves them at 1). Seen in lesson
  4.19's new week chart on Chrome (headless CDP), the iPhone 17 Pro simulator
  (new `TutorialCanvasTapUITests`) and the Android emulator: the same four
  taps picked Tue, missed on the clipped-off pill end, picked Thu, cleared.
  Not seen on a device: a mirrored canvas's text under RTL on the natives
  (checked on the web only). Text is not hit-testable.

- **N-088** · raised `2026-1003-1559-n061-n088-safe-area-insets-on-the-web`
  · closed 2026-10-03, `2026-1003-1559-n061-n088-safe-area-insets-on-the-web` — kept `viewport-fit=cover` and padded `body` with the
  four `env(safe-area-inset-*)` values (`wasm/index.html`). Measured on the
  iPhone 17 Pro simulator (iOS 26.5, Safari): portrait reports 0/0/0/0, so
  the bottom-edge worry was unfounded there and the rule is a no-op;
  landscape reports 0/62/20/62 (t/r/b/l), and before the fix the Dynamic
  Island hid the header's "Docs" link. After it the header clears the island
  and the page clears the home indicator; portrait screenshots unchanged.
  Dropping cover was rejected: Safari reports the 20px bottom inset without
  cover too, so the padding was needed either way. The cost is that the
  header's panel colour no longer reaches the side edges in landscape (the
  strips are the page background, as Safari's own letterboxing would be).
  Rotation was driven by a throwaway XCUITest setting `XCUIDevice`
  orientation; osascript has no assistive access here. `wasm/verify/run.sh`
  passes.
- **N-086** · raised `2026-1003-0453-n085-shell-surface-follows-go-tree`
  · closed 2026-10-03 — the candidate: a dark MaterialTheme scheme below
  GrMobRoot when the bars' colour is dark. `ShellMaterialTheme`
  (`GrMobSurface.kt`) wraps the tree in `MaterialTheme(darkColorScheme())`
  or `lightColorScheme()` and provides LocalContentColor white or black,
  from the same answer as the bar icons (`barsColor`, now shared with
  ShellSurface); a derivedStateOf, so only crossing the threshold
  recomposes. The light row is exactly the old defaults. MaterialTheme also
  provides a ripple, selection colours and bodyLarge as LocalTextStyle
  (read from the material3 1.3.1 bytecode), which would have changed every
  light app's press feedback and text metrics, so their outer values are
  provided again inside it. A Go accent's ink (Checkbox tick, Switch
  thumb) and a Button label on a Go Background with no TextColor are pinned
  to `OnGoColor` (the light scheme's onPrimary, white): the dark scheme's
  onPrimary drew a deep purple thumb on the demo's blue track
  (`TestAndroidInkOnTheAccentIsNotTheSchemes`). Seen on the emulator, demo
  with Screen painted #1C1C1E on a light system: dark tab row, white header
  and "Hello, stranger." ink, white tick and thumb. The unpainted demo by
  day is pixel-identical to before apart from the status-bar icons; the
  tutorial at night is unchanged except that an unchecked checkbox's
  outline is the dark scheme's (light grey, was near-invisible dark grey).
  `core.ColorScheme`'s doc names the chrome. Not covered: a dark SafeArea
  over an unpainted root composes light for one frame (claims join after
  the first composition), and the window theme (Theme.Material.Light)
  still styles the classic views (the rich-text EditText, Toasts).

- **N-085** · raised `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard`
  · closed 2026-10-03, `2026-1003-0453-n085-shell-surface-follows-go-tree` — the first
  candidate: the shells take their surface and bar style from the Go tree,
  not the system. One rule on both (`GrMobSurface.kt`, `GrMobSurface.swift`):
  the bars' colour is the innermost painted SafeArea's Background, else the
  root's, else the shell page (#FFFFFF, DefaultTheme's Background); the
  surface is the root's Background or that page; dark bars' colour (WCAG
  luminance ≤ 0.5) means light icons. Android writes the window background
  and both bars' icon appearance from one `ShellSurface` composable composed
  after the tree; a painted SafeArea registers a claim instead of setting the
  icons itself (`SystemBarIcons`, which a root-level writer would have raced
  on a recomposition that skipped the SafeArea). The claim is read inside
  the SideEffect, which runs after the claims' DisposableEffects in the same
  apply, so the first frame is right. iOS paints the surface under the bars
  behind the root, carries the SafeArea's colour up as a preference
  (`transformPreference`, so the innermost wins) and sets the window's
  `overrideUserInterfaceStyle` from it — status bar and native chrome
  together, since an App cannot subclass SwiftUI's hosting controller for
  `preferredStatusBarStyle`. The override hides the system scheme from the
  colorScheme environment, so `AppWindowReader` now reads it from the window
  scene's traits (`GrMobSystemScheme`, watched with
  `registerForTraitChanges`), which overrides do not reach. Seen: the demo
  app (follows no scheme) on a dark iPhone 17 Pro simulator (iOS 26.5) and a
  night-mode emulator draws a white page with dark status-bar text / icons;
  the tutorial on iOS is dark with light status text on a dark system and
  switches live to light and back (so the scene read reaches Go through the
  override); on Android it is dark with light icons at night and light with
  dark icons by day; the demo with its Screen painted #1C1C1E on a light
  system gets light icons on both. `core.ColorScheme`'s doc says the scheme
  is the system's, not the shell's. The Material chrome left light on
  Android is N-086.
  - Note: the iOS half of the Renderer.swift wiring landed inside
    `c43e3bc` (another session's commit swept the working tree); the rest
    lands with this item's commit.

- **N-073** · raised `2026-0921-1118-comps-round-four-phase-5-editable-grid`
  · closed 2026-10-03, `2026-1003-0208-keyed-callback-ids-dark-theme-chart-data` — identity-keyed IDs. `core.Keyed`
  and each Navigator frame open an ID scope while their subtree renders
  (`Context.keyScope`), so its callbacks are numbered under the key path
  (`cb_r2/e0/1`) and the enclosing counters do not move; root IDs keep `cb_N`.
  Keys are escaped (`/`, `~`, `%`) and a key repeated in one scope is told
  apart by occurrence (`0~1`). ErrorBoundary's rollback now walks a trail of
  registrations and key counts. EditableGrid's EDIT transitions re-bind no
  other cell (`TestEditableGridEditMovesNoOtherCellsHandler`, which shows
  `cb_1`→`cb_6` without the scope); a stale ID from a popped frame reaches
  nothing; Navigator's own Pop stays outside the frame so two quick backs pop
  twice. No host parses IDs. Mutation-tested (`core/keyed_ids_test.go`).
- **N-074** · raised `2026-0921-1419-tutorial-light-dark-theme`
  · closed 2026-10-03, `2026-1003-0208-keyed-callback-ids-dark-theme-chart-data` — `core.DarkTheme` (a recoloured copy of
  DefaultTheme, `newDarkTheme`) is in `BundledThemes` and every census passes
  over it; palette.mjs gained its rows and the browser paints them. The four
  `*OnLight` fields are now `PrimaryInk`/`SuccessInk`/`WarningInk`/`ErrorInk`
  with no field alias (a deprecated duplicate would make "clear the tone when
  re-branding" a silent no-op); the resolvers and `OnLight` keep deprecated
  wrappers, and the reverse lookup is `AsInk` (`Variant.Ink` was taken). The
  tutorial's `darkTheme` is `core.DarkTheme`. Census figures replaced the
  hand-computed ones (Border 1.45:1, not 1.5).
- **N-018** · raised `2026-0916-1410-chart-palette-and-canvas-gradients`
  · closed 2026-10-03, `2026-1003-0208-keyed-callback-ids-dark-theme-chart-data` — `core.DarkTheme` spends
  `DefaultDarkChartColors` and `DefaultDarkSequentialColors`.
- **N-010** · raised `2026-0916-1129-charts-on-canvas`
  · closed 2026-10-03, `2026-1003-0208-keyed-callback-ids-dark-theme-chart-data` — per host, as recommended: no
  screen-reader-only primitive. `core.AccessibilityChart(core.ChartData)`
  puts the numbers on the chart's labelled RoleImg node (series of points,
  categorical or numeric x; NaN dropped). Web runtime and htmlout write a
  visually hidden `<table>` as leading chrome and the role as `figure` (an
  img's children are presentational); iOS builds an `AXChartDescriptor`
  (`GrMobChartAccessibility` on Row and Column); Android keeps the sentence.
  Every summarising chart carries it except Sparkline and Gauge. Held across
  languages by gen.go `chartCases` + `chart_test.mjs`; Chrome's own AX tree
  (CDP `getFullAXTree`) shows each figure's table with row headers and cells
  on lessons 4.20 and 4.36. The descriptor is unheard (N-070).
- **N-069** · raised `2026-0921-1035-comps-round-four-phase-3-structure`
  · closed 2026-09-29, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — `core.Focus` now reaches a node that
  takes no input focus, per host, with no new core API: the web makes it a
  programmatic target (`tabindex="-1"`, out of the Tab order) and focuses it;
  iOS moves VoiceOver's focus to a stamped Text (`AccessibilityFocusState`,
  once per epoch, attached only to Texts that carry a stamp); Compose has no
  way for a node to take TalkBack's focus, so it reads nothing. A Text takes
  style props only, so it is named by applying `FocusTarget` to the node it
  renders. `comps.Wizard` gained `TitleRef` (the caller's ref, as Drawer's
  CloseRef is) and focuses the title after every step change; lesson 4.35
  passes one. Seen in headless Chrome on 4.35: after Next the focused
  element is the "Gift note" heading and after Back "Your name"; without the
  tabindex it is the page body. iOS builds and is unheard (no VoiceOver on
  the simulator). Tests: `TestWizardTitleRefTakesFocusOnEveryChange`, a
  runtime test for the programmatic target.

- **N-057** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`
  · closed 2026-09-29, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — the layer-only shape: `core.OnEscape`, a
  claim of its own beside OnBack (Escape closes what is open over a screen
  and does not navigate, so a Navigator route and an AppBar's back arrow
  claim back and not Escape), with an open Modal's OnDismiss counting as a
  claim. `comps.Drawer`'s open panel carries `OnEscape(OnDismiss)`, so every
  Modal-based widget and the Drawer close on Escape.
  - Web: one window keydown listener (shared with the page shortcuts, which
    keynav_test pins to one) runs the last claimant in document order; a
    handler that consumed the key first (a combobox clearing its active
    option) wins. Seen in headless Chrome with CDP keys: 4.18's drawer and
    6.6's dialog close; both stay open with the listener removed. Four
    tests in browserback_test.mjs.
  - Android: the Activity walks the tree for the last claimant
    (`GrMobRuntime.lastEscapeClaim`). A Modal's Dialog is its own window and
    did *not* close on Escape (seen on the emulator: the platform maps
    Escape to back only with predictive back off, and this shell opts in),
    so the Dialog's content wraps its window's callback (`DialogEscape`).
    Seen: `input keyevent KEYCODE_ESCAPE` closes 4.18's drawer and 6.6's
    dialog, and back still closes the dialog.
  - iOS: unverified. The claim is an invisible Button with Escape as its
    keyboardShortcut (`GrMobEscapeClaim`). XCUITest's Escape reached neither
    it nor `.cancelAction`, while the same claim bound to Control+Option+E
    closed the drawer; the Mac's own Escape could not be sent (osascript has
    no keystroke permission here). `testEscapeClosesTheDrawerAndTheDialog` is
    a strict XCTExpectFailure, like F6's. A Modal has no Escape claim on iOS
    yet. Carried by N-004's real-iPad-keyboard check.

- **N-050** · raised `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep`
  · closed 2026-09-29, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — the hosts report the system's scheme and
  the app picks. `core.Window` gained `ColorScheme` ("light", "dark", or
  empty when no host said) and `Dark()`, carried as `scheme` on the existing
  "window" report and deduped with it: Android reads the configuration's
  night bit (a switch recreates the Activity, which reports again), iOS the
  reader's `colorScheme` environment (watched, so a switch re-reports), the
  web `prefers-color-scheme` (with a change listener). Core still bundles no
  dark theme (N-074). The tutorial follows the window's scheme until a page
  sends its own "theme" choice, so the web's Light/Dark/System switch still
  wins there; on a native it paints the root in darkTheme's Background and
  ink, the colours the web page's CSS supplies. That exposed a real
  divergence: a container's TextColor was inherited on the web (CSS `color`)
  and not on the natives, so lesson 1.1's plain `core.Text` name and numbers
  drew Compose's black on the dark card. Now Compose provides
  `LocalContentColor` below a container with TextColor and SwiftUI sets the
  container's foreground style (`grMobInk`), documented on `core.TextColor`.
  Seen: the emulator (`cmd uimode night yes`) draws 1.1, 4.37 and 5.9
  readable dark and returns to light on `no`; the iOS simulator
  (`simctl ui appearance dark`) draws 1.1 dark and comes back live to light.
  Tests: `TestWindowHostEventDecodesTheColorScheme`,
  `TestTheSystemSchemeRulesUntilAPageSays` (mutation-tested), two
  window_test.mjs cases, and the four-shell spelling census gained the key.
  The surfaces behind apps that do not follow the scheme are N-085.

- **N-078** · raised `2026-0922-0440-next-list-radio-positions-browser-notify-fold-grid-scroll-ios-round-four`
  · closed 2026-09-29, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — decided by content, not role. A labelled
  container is `.ignore` when every child is hidden (as before), `.contain`
  when two or more *members* sit below it, and `.combine` otherwise. A member
  is a control type, a pressable node with something to announce (a label or
  a child not hidden), or a nested labelled container, which is Compose's
  merge boundary per clickable and per labelled node; the walk does not
  enter a member and stops at the second (`GrMobNode.holdsControls`,
  `grMobChildMode`). Nested named containers count so that Poll's results
  (named rows, no controls) are one stop per option and Wizard (one Next,
  a step's body) does not merge its step. PINInput's tap-catching row
  (unnamed, all children hidden) is not a member, so it still combines.
  `ios/verify`'s new census renders 21 bundled widgets in Go and holds every
  labelled container's shape to a table (`childmode.go` /
  `childmode.swift`; mutation-tested by removing the contain arm).
  On the simulator (iOS 26.5): "Budget" is a container whose every cell is
  a Button named "Amount, row 1, $1200.00" etc.; "Price" holds Slider
  "Minimum" $20 and "Maximum" $80; "Label colour" is not itself Selected,
  "blue" is. The round-four tests now reach cells, sliders and the ✕ by
  name; the range test drags the thumb, because XCUITest's
  `adjust(toNormalizedSliderPosition:)` reads the value as a percentage and
  did nothing on "$20". All 51 Tutorial UI tests pass (`testAudioPlayer`,
  which streams, failed once in the full run and passed alone). VoiceOver
  itself is still unheard: Accessibility Inspector or a device. Open
  questions left with it: Stepper's value is now read as its visible number
  between the buttons, the web's reading; a SliderRow's title Text and its
  slider both say "Minimum".

- **N-034** · raised `2026-0917-1935-pin-input-and-a-code-with-no-gaps`
  · closed 2026-09-29, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — half done, half declined on the session's
  recommendation (the user's decision; reopen to overrule).
  - `examples/signup` → `PINInput`: done. A passing submit now sends a code
    (the example states it, 246810, having no mail server) and a verification
    step takes six digits in `comps.PINInput` before "Account created". A
    wrong code is cleared with "That code is not the one we sent"; "Use a
    different address" returns to the form with its values kept. The step
    renders in `ctx.Scope("verify")` because PINInput holds hooks.
    `TestTheCodeStepRefusesAWrongCodeAndGoesBack`, and three tests now pass
    through the step. `docs/images/signup.png` is the form's mismatch error,
    which the step does not touch, so it was not retaken (its claims test
    passes).
  - `examples/chat` → `comps.MessageThread`: declined. The example exists to
    teach `core.For` and `core.Keyed`, which MessageThread hides, and the
    widget is taught by lessons 4.33 and 4.34 already; it also cannot grow to
    fill yet.

- **N-084** · raised `2026-0928-1917-next-list-overflow-check-ios-width-cap-sparkline-fade-reduce-motion`
  · closed 2026-09-28, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — the ✕ carries a new
  `core.PressKeepsFocus()`, a no-arg prop (`pressKeepsFocus: true` on the
  wire) that the web runtime turns into a `mousedown` preventDefault, so a
  press never blurs the field; the natives need nothing. Picked over a blur
  payload naming where focus went, which would put a node path in every blur
  for one widget. The ✕ also reports its own focus and blur as the field's, so
  Tab onto it is not a blur and Enter there discards. The fix exposed a second
  defect: the field was now focused when the discard removed it, and Chrome
  fires `blur` synchronously inside the removal, which re-entered Go with the
  removed field's positional ID (N-073's hazard, now firing): the next tap on
  the cell opened nothing. The runtime now drops any element event fired while
  it is mounting or applying a batch (`applyingTree`). Measured in headless
  Chrome with CDP's real mouse path on 4.37: 60, 120, 300 and 600ms holds all
  discard (before: 300 and 600 committed); Tab onto the ✕ outlasts the grace
  and Enter discards; return commits and lands on row 2; a 300ms tap on
  another cell commits and opens it. Pinned by
  `TestEditableGridDiscardIsOneFocusUnitWithTheField` and three runtime tests
  (the guard's mutation-tested).

- **N-081** · raised `2026-0924-1211-tutorial-overflow-sweep-web-floors-nested-scroll`
  · closed 2026-09-28, `2026-0928-1917-next-list-overflow-check-ios-width-cap-sparkline-fade-reduce-motion` — browser check 24. `wasm/verify/overflow.mjs` holds the
  sweep's reading (in the page) and its judgement (a pure function), and
  `overflow_test.mjs` reaches every rule with hand-built boxes. The check opens
  every lesson gen.go numbers from `tutorial.Chapters` (a new `lessons` field in
  the transcript) through the real site page's hash deep link, in the split at
  1280px and in the phone layout at 390 and 360, and fails on a box that
  escapes a parent that shows overflow, is cut off by one that hides it, runs
  past the screen's side, holds a word wider than itself, or is drawn smaller
  than its stated px size. Skipped: code editors, rotated layers, SVG
  internals, and inert subtrees (4.18's shut Drawer panel was the only finding
  on today's tree). 240 views in about a minute. Mutation-tested by reverting
  each fix of `2026-0924-1211` in turn: 1.1's stats row (escape), Avatar's
  FlexShrink(0) (squeezed 34 of 36 at 360), the field floors (Send and Show
  escape), the mount's overflow-wrap (6.8's and 4.11's text) and the nested
  Scroll rule (1.5 squeezed to 2px at all three views). Still first state only.
- **N-082** · raised `2026-0924-1211-tutorial-overflow-sweep-web-floors-nested-scroll`
  · closed 2026-09-28, `2026-0928-1917-next-list-overflow-check-ios-width-cap-sparkline-fade-reduce-motion` — `docs/images/tutorial-lesson.png` retaken with
  `wasm/shots/shoot.sh tutorial-lesson`; the card shows "Following" whole.
- **N-080** · raised `2026-0924-1211-tutorial-overflow-sweep-web-floors-nested-scroll`
  · closed 2026-09-28, `2026-0928-1917-next-list-overflow-check-ios-width-cap-sparkline-fade-reduce-motion` — seen on both natives, and SwiftUI had a defect.
  - Compose (the emulator at 411dp and, via `wm density 480`, at 360dp, reset
    after): 4.11's StaticMap frame takes a 318dp and a 266dp column and keeps
    its 180dp height; 1.1's avatar is round and the three stats sit inside
    the card; 1.5's short Scroll is 474px (158dp) inside its 1dp border; 4.3's
    avatars are round beside rows whose subtitle wraps.
  - SwiftUI (iPhone 17 Pro simulator, 402pt): 1.5's viewport is 158pt, 1.1's
    avatar 40×40, 4.3's 36×36. 4.11's map was **320pt wide at x 47, 13pt past
    its 306pt column** — `grMobDimension` drew a points Width as a rigid
    `frame(width:)`, which `GrMobMaxWidthLayout`'s narrowed proposal cannot
    shrink, and a percentage cap has no length to fold in. A points Width
    under a percentage cap is now a flexible frame (`minWidth: 0,
    idealWidth: w, maxWidth: w`, the `relativeCap` argument): 306pt, even
    margins. StaticMap is the only bundled node with that pair. Pinned in
    `TestSwiftCapsOutsideTheGrowFrame`; three new tests in
    `TutorialNativeFloorsUITests` (the map, the nested Scroll, the avatars),
    the map one seen failing on the old Swift at x 367 against 354.
  - The even crop under ContentModeFill is still unseen with a real picture:
    4.11's provider host is gone, which is the lesson's own subject.
- **N-025** · raised `2026-0916-1643-zero-basis-floor-area-fades-scatter-squares-android-look`
  · closed 2026-09-28, `2026-0928-1917-next-list-overflow-check-ios-width-cap-sparkline-fade-reduce-motion` — Sparkline's `Area` fades as AreaChart's does, 4D under
  the line's highest point to 0A at the bottom edge its area closes on
  (`areaFadeTo`, which `areaShape` now calls with the scale's zero), and keeps
  the flat 33 tint for a non-hex colour or no finite value.
  `TestSparklineAreaFadesTowardItsBottomEdge`; seen in headless Chrome on
  4.20. The natives already draw `LinearGradientFill` for AreaChart. (was #34)

- **N-016** · raised `2026-0916-1331-next-list-sweep-maxlines-chart-types-evenodd`
  · closed 2026-09-26 by the user, not done — the double-post claim was never
  reproduced in forty sessions of carrying it. A recurrence is a new item.
  (was #20)
- **N-036** · raised `2026-0918-0130-round-three-eight-widgets-and-what-a-text-node-cannot-do`
  · closed 2026-09-26 by the user, not done — a sixth low-hanging-fruit round
  with no candidates behind it. (was #46)

- **N-019** · raised `2026-0916-1410-chart-palette-and-canvas-gradients`
  · closed 2026-09-25, `/next-list` re-check. The one failure never came
  back: `2026-0923-1149` ran all Tutorial* classes, 48 tests with 0 failures,
  and that includes `TutorialChartsUITests.testChartsAreOneElementEachAndRedrawOnUpdate`
  (48 is the count of `func test` in ios/GrMobUITests/Tutorial*.swift). The
  original failure's reason was never captured, so a recurrence is a new item.

- **N-079** · raised `2026-0922-0440-next-list-radio-positions-browser-notify-fold-grid-scroll-ios-round-four`
  · closed 2026-09-23, `2026-0923-1149-next-list-slider-values-radar-centre-zero-basis-border-opacity-seen` — `comps.SliderRow` now states its readout as the
  slider's `core.AccessibilityValue` text (Format, or the default precision;
  nothing when Format returns ""). Both natives already honoured the text on
  any node (Compose's stateDescription, SwiftUI's accessibilityValue); the
  two web exporters' `ariaValue` gained a Slider arm that writes
  `aria-valuetext` alone, the input keeping its own value/min/max. Heard on
  the emulator's TalkBack: "$20, Minimum, Slider" and "$80, Maximum,
  Slider". On the simulator the combined "Price" element reads "$20, $80"
  and "$130, $130" after the push (`testSwatchesHexShortFormAndTheRangeThatCannotCross`
  asserts both); the inner sliders XCUITest synthesizes under the combine
  still say 10%/40%, which is N-078. Documented on `core.Slider`,
  `core.Style.AccessibilityValue` and docs/components.md.

- **N-077** · raised `2026-0922-0204-n076-logical-insets-hidden-fill-ios-carry-talkback-sweep`
  · closed 2026-09-22, `2026-0922-0440-next-list-radio-positions-browser-notify-fold-grid-scroll-ios-round-four` — Compose numbers a
  `selectableGroup`'s members by `layoutNode.placeOrder`, which restarts in
  every layout parent, so a grid of Rows was counted per Row (every heard
  number was one plus the count of lower places across both rows). Compose
  1.10's `setCollectionItemInfo` also overwrites a stated item info whenever
  the parent is a `selectableGroup`, so the radiogroup role no longer writes
  one: `RenderNode` states the group's `collectionInfo` and each radio's
  `collectionItemInfo`, counted in document order from the Go tree
  (`LocalGrMobRadioPositions`). Heard on the emulator: 5.9's swatches 1…8 of
  8 in order, and 4.16's RadioGroup "2 of 3". Pinned by
  `TestComposeStatesEachRadiosPlaceInItsGroup`.
- **N-029** · raised `2026-0917-0227-foldables-window-record-and-two-pane`
  · closed 2026-09-22, `2026-0922-0440-next-list-radio-positions-browser-notify-fold-grid-scroll-ios-round-four` — no person needed: CDP's
  `setDeviceMetricsOverride` takes a `displayFeature` and
  `setDevicePostureOverride` a posture, and browser check 23 drives lesson
  4.21 through a vertical hinge, the book posture, a tabletop hinge and none.
  It found that a segment change at an unchanged window size fires no
  `resize`, so the fold reached Go only on the next posture change; the
  runtime now also reports on the `(horizontal|vertical)-viewport-segments: 2`
  media queries. Mutation-tested by removing those listeners.
- **N-014** · raised `2026-0916-1157-next-list-charts-on-devices-tier-e-alarms-timezone`
  · closed 2026-09-22, `2026-0922-0440-next-list-radio-positions-browser-notify-fold-grid-scroll-ios-round-four` — the permission is granted over
  CDP (`Browser.grantPermissions`), so no person is needed. Browser check 22
  posts four notifications through the real runtime in headless Chrome and
  hears each real `Notification`'s "show": an immediate one at once, a
  scheduled one at its time, a sweep answering with that id alone, a
  swept timer never firing, another prefix's firing, and a second sweep
  reporting nothing. Mutation-tested by removing the sweep's `clearTimer` (the
  first version scheduled the swept post at +60s, which could never fire
  inside the check, and passed the mutant; it is now due between the sweep
  and the read).

- **N-076** · raised `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror`
  · closed 2026-09-22, `2026-0922-0204-n076-logical-insets-hidden-fill-ios-carry-talkback-sweep` — Left
  means leading, everywhere: the web now writes `padding-block`/`padding-inline`
  and `margin-block`/`margin-inline` (the runtime's `edgeLogicalCSS`,
  `htmlout.EdgeLogicalCSS`), matching Compose's `start` and SwiftUI's
  `leading`, and never a physical side for an inset; the code editor's gutter
  inset is `padding-inline-start` (the editor is `dir="ltr"`). The field names
  stay Left/Right, documented on `core.EdgeInsets` and the four side props.
  cssstyle.mjs models the logical pairs and refuses an element given both a
  physical and a logical name for one box; its CSSOM table gained six rows,
  replayed against Chrome by check 14. Browser check 21 now also reads a Left
  inset's computed side under each direction (mutation-tested by restoring
  the physical shorthand). BarChart's spacer Row is kept, its comment updated.

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
