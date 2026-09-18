# Device pass: nine bugs the simulators found

**Session:** 219e6e32-9207-462f-be28-24dc62b00220
**Date:** 2026-09-18 07:13 (follows "two-pane-tutorial")
**Branch:** master (c843625 → this commit)

## The ask

"Do what you can from the Next list. Use the simulators as necessary."

The Android emulator (`emulator-5554`, Medium Phone API 36) and the iOS
simulator (iPhone 17 Pro, iOS 26.5) were already booted.

## 1. Code-only items from the list

- **`core.AuditTree` in the comps widget tests.** It now runs in
  `renderDebug` (stepper_test.go) and in `rowHarness.render`
  (select_row_test.go). The two other hand-rolled loops that assert concerns
  run it too: `rangeHarness` and the searchable-select focus test. **Nothing
  surfaced.**
- **htmlout box-sizing.** The export writes
  `*,::before,::after{box-sizing:border-box}` (`borderBoxCSS`), and only when
  a node needs it (`needsBorderBox`).
  - A node needs it when it declares a size (Width, Height, or any Min/Max)
    and something sits inside that size: Padding, a BorderWidth, or a form
    control's own UA padding and border (input, textarea, select, button).
  - The rule rides the existing `motionStylesheet` head, as `motion.borderBox`.
  - It is conditional because `TestStillExportWritesNoHead` pins "no head
    unless needed".
  - Stale comments and docs updated: `comps/chip.go`, `comps/avatar_stack.go`
    and docs/components.md, `comps/lightbox.go`, `comps/data_table.go`.
  - Left alone: `docs/platforms/wasm.md:1510`. It describes a verify harness
    with no hosting page, and is still true.
- **CalendarHeatmap ties.** The earliest day wins, because the grid is walked
  oldest first with a strict `>`. Now documented on the type, at the loop, and
  in docs/components.md. `TestCalendarHeatmapTieNamesTheEarliestDay` checks
  both entry orders.

## 2. The device pass

- **Android** was driven with a scratch helper (`ad.py`). It matches text *or*
  content-desc, with exact matches preferred. Without that preference, a tap
  on "Play" hit the hint "Press Play…".
- **iOS** was driven by a new XCUITest,
  `ios/GrMobUITests/TutorialDevicePassUITests.swift`.
  - Nine tests cover lessons 4.25–4.32 and 5.8.
  - With `TEST_RUNNER_GRMOB_SHOTS_DIR` set, each step writes a PNG and a
    `.txt` accessibility-tree dump (`app.debugDescription`).
  - The dumps were what found the real labels and frames.

The bugs, in the order found:

| # | Target | Bug | Fix |
|---|---|---|---|
| 1 | all | Today's calendar cell was 2 units taller than its row. Its ring was a border on an unsized box, and every target puts that border outside the content. Android measured 113px against 107px, with the numeral 3px low | Every day cell carries `BorderWidth(1)`: `ColorTransparent`, or Primary for today. This is Chip's invisible-ring trick. `TestCalendarEveryCellCarriesTheRingsWidth` |
| 2 | Android | The FlowRow branch of `GrMobRow` ignored `AlignItems`. Breadcrumb's "#40121" and its chevrons sat 14dp above the button crumbs | `RowChildren(lineAlign:)` applies `Modifier.align` per child. FlowRowScope is a RowScope |
| 3 | iOS | `GrMobWrapLayout` placed every child at the top of its line, so the same Breadcrumb sat 10pt off | `crossAlign` parameter, using `GrMobFlexSolver.crossOffset` per line |
| 4 | iOS | **comps.Link was absent from the accessibility tree** (`app.links.count == 0`) | See "What `.combine` does with nothing" below |
| 5 | iOS | A tappable link announced as a button: `.isButton` beat `.isLink` | `GrMobGestureAccessibility(isLink:)` adds no button trait for a RoleLink |
| 6 | iOS | MessageBubble's `MaxWidth("80%")` compounded, and "Canvas stars now too" wrapped over three lines | See "Percentage caps" below |
| 7 | iOS | **A FlexGrow box painted its background around its content only.** The range band drew as separate strips, a selected day as a thin pill, and a tap beside the numeral missed | `grMobGrow` moved inside the background/border/gestures in `GrMobBoxModifier`. The margin is now outside it (CSS order) |
| 8 | iOS | TagInput lost focus on Return. SwiftUI resigns a TextField on submit; web and Compose keep it | TagInput takes a `UseFocusRef` and calls `core.Focus` in its submit handler. Nothing changes on web and Compose. `TestTagInputKeepsFocusAcrossReturn` |
| 9 | web | In the split layout, the toast landed below the phone once the window was taller than ~900px: 114px under the glass at 1300px | `bottom: calc(max(1.2rem, (100% - 860px) / 2) + 36px)`, which follows the capped, centred phone |

Plus two small ones:

- **The page header said "60 lessons".** It had since the foldables commit,
  and there are 74. Fixed, and `examples/tutorial/pagecount_test.go` now
  counts lessons and chapters from `Chapters`.
- **"In-app link followed 1 times"** now uses the tutorial's `plural`.

### What `.combine` does with nothing

`grMobAccessibility` gives every labelled container
`accessibilityElement(children: .combine)`. When every child is hidden, there
is nothing to combine, and **SwiftUI creates no element at all**. The label,
the role and the tap's accessibility action all go with it.

An experiment built four labelled boxes in lesson 4.29, then reverted them:

- (a) no role, hidden text, OnClick;
- (b) RoleLink, *visible* text, OnClick;
- (c) RoleLink, hidden text, no OnClick;
- (d) RoleLink, hidden text, OnClick.

Only (b) existed. OnClick and the role are irrelevant.

The fix, `GrMobNode.containerStyle`:

- It sets an unparsed `GrMobStyle.labelOnly` when every child is
  AccessibilityHidden (or there are none).
- It is computed at render time, from the live children, because patches
  change them.
- `grMobAccessibility` then passes `.ignore` instead of `.combine` as an
  *argument*, not a branch. A branch would add a `_ConditionalContent` layer,
  which the file warns crashes swiftc.
- It is used by GrMobRow, GrMobColumn/Card/Box and GrMobZStack.

MessageBubble had looked like a counter-example, since its texts are hidden
too, yet its elements existed. Why the group role differs was not pursued.

### Percentage caps

The bug:

- `GrMobMaxWidthLayout` resolves a percentage against its proposal.
- On a Row's main axis, that proposal is the child's own slot.
- So the cap came out as 80% of an already-narrowed width.

It was the same shape as percentage *floors*, which `percentFloors` had
already solved in the flex solver. The fix mirrors that:

- **`GrMobFlexSolver.percentCaps` / `capped`** (GrMobFlex.swift, and checked
  by ios/verify): cap = fraction × container extent + margin. Bases and final
  mains are clamped to it.
- **FlexChildren** passes `GrMobFlexPercentCap` / `GrMobFlexCapMargin` for a
  horizontal axis. It also sets the environment value
  `grMobPercentCapResolved`, so the child's `GrMobMaxWidthModifier` stands
  down. The modifier resets the value for its own subtree.
- **The wrap layout** has no solver, so it opts out
  (`resolvesPercentCaps: false`).
- **A start-justified Row still compounded, one level up.** It hugged its
  bubble, and the parent re-proposed the hugged width. So `containerMain` has
  a third claimant: `percentCapped` makes the Row fill a definite offer, which
  is CSS's rule anyway.

After the fix, the bubbles measure exactly 220.7pt, which is 80% of the 276pt
row. "Bubbles are a widget now." still wraps on the iPhone, legitimately: the
text needs slightly more than 196pt.

### The grow-frame move (bug 7)

This touches every FlexGrow and stretched node on iOS. It was invisible for
content that fills its slot, and most rows contain a growing title, which
hides it. A day cell centres one numeral, which is what exposed it.

After the move:

- all nine tutorial UI-test classes were re-run;
- every test passed except `testBothDeclaredChordsPressTheButton`;
- that test also fails with every iOS runtime change stashed, so it is
  pre-existing (see Next).

## 3. Verified on the devices, and not a bug

- **Android:**
  - TimePicker menus;
  - AvatarStack;
  - the BarItem badge, which clears on select;
  - Lightbox (the image just took ~2s to load);
  - the PasswordField swap;
  - the Heatmap, CalendarHeatmap and Histogram canvases;
  - CopyButton's toast and the system clipboard preview;
  - Link and BulletList;
  - AudioPlayer: plays, reaches 0:06 / 6:13, Pause, ±15s;
  - MessageBubble at 80%;
  - ExpandableText;
  - the canvas half-star at 3.5;
  - TagInput's wrapping pills and its Return-then-type.
- **iOS:** the same list, plus:
  - `simctl pbpaste` read "CATS-4721-QX";
  - the TimePicker readout reached 10:45 and the 24-hour conversion;
  - the half-star at 3.5.
- **Web, at a real 1400px viewport.** A same-origin iframe 1400 CSS px wide
  inside the zoomed tab, scaled down with `transform`, stood in for a wider
  window. Its media queries see 1400.
  - The split turns on by itself.
  - The switch hides below 900px and returns above it.
  - The text column is capped at 820px and centred.
  - Both palettes are right: light was checked by injecting the light tokens.
- **Pitfall:** Chrome was occluded, so `requestAnimationFrame` never fired and
  media-query changes were not evaluated until a frame was forced. A
  screenshot forces one. The first reading ("still split at 850px") was this
  artifact, not a page bug.

## 4. Other pitfalls

- **Test-driver false alarms** before the real bugs:
  - "Orders" matched the bottom bar's tab, not the crumb.
  - Calendar days in this demo are named "11 March 2026, summary", not
    "Wednesday, March 11".
  - ExpandableText's toggle keeps its name, "Read more", by design.
  - An element tapped during a slow swipe's momentum only stops the scroll,
    hence the `settle()` sleep.
  - `typeText` on a re-queried element fails after a re-render, so the test
    uses `app.typeText`.
- **zsh does not word-split `$args`.** The xcodebuild flags need an array:
  `"${args[@]}"`.
- **`mobile/verify`'s source checks mask string literals**
  (`maskSwiftNonCode`), so a pinned substring must avoid quoted text.
- **The `wasm/verify` census** went 604 → 605 because of the new test file.
  Bumped with `perl -pi -e 's/\b604\b/605/g'`.
- **The dev server was stopped by its task ID** (TaskStop), not by an
  lsof-found PID, as last session's lesson said.

## Files

- **Go:**
  - `htmlout/export.go`, `htmlout/export_test.go`;
  - `comps/calendar.go`, `calendar_test.go`;
  - `comps/tag_input.go`, `tag_input_test.go`;
  - `comps/heatmap.go`, `heatmap_test.go`;
  - comment-only: `comps/{chip,avatar_stack,lightbox,data_table}.go`;
  - harness audit: `comps/{stepper,select_row,date_range_picker,searchable_select}_test.go`;
  - `examples/tutorial/chapter4.go` (the plural);
  - `examples/tutorial/pagecount_test.go` (new).
- **Android:** `android/app/src/main/java/com/grmob/runtime/Renderer.kt`
  (FlowRow `lineAlign`).
- **iOS:**
  - `ios/GrMob/Runtime/GrMobFlex.swift` (`percentCaps`, `capped`,
    `containerMain(percentCapped:)`);
  - `GrMobNode.swift` (`containerStyle`);
  - `GrMobStyle.swift`: `labelOnly`; `.ignore`/`.combine`; `isLink`; the
    MaxWidth stand-down and its environment key; the grow move;
  - `Renderer.swift`: containerStyle at three containers; the wrap layout's
    `crossAlign`; FlexChildren's cap values; the layout's caps;
  - `ios/verify/flex.swift` (cap checks);
  - `ios/GrMobUITests/TutorialDevicePassUITests.swift` (new).
- **Web:** `wasm/index.html` (toast bottom, header count).
- **Checks:**
  - `mobile/verify/maxwidth_test.go` (pins updated to the new modifier);
  - `wasm/verify/{repowalks,timings}_test.go` (605).
- **Docs:**
  - `docs/components.md` (AvatarStack ring, CalendarHeatmap tie, TagInput
    focus);
  - `docs/api/*` regenerated.

## Next

- **Copy buttons cover code on the natives.** The codeBlock's Copy layer has
  an opaque background in the top-right corner. It hides the end of line 1 on
  iOS and web, and lines 1–2 on Android (48dp touch minimum). The options all
  change the look, so this is a user decision:
  - top padding on the editor;
  - right padding;
  - a toolbar strip above the code.
- **Android text-field echo race.** At machine typing speed, Go's rewrite
  after a separator commit (draft → "") arrives after later keystrokes were
  sent. It matches no queued echo, wins, and drops them: ",gamma," committed
  "mma". It was not reproducible at human pace (~100ms/key). A real fix needs
  a sequence-numbered ack, which is a protocol change. iOS has the same
  bookkeeping and was not tested at speed.
- **Native controls ignore the theme accent.** Switch and Slider are Material
  purple on Android and system green on iOS. Needs an accent/tint property on
  all targets: CSS `accent-color`, SwiftUI `.tint`, Compose
  `SwitchDefaults`/`SliderDefaults`.
- **`TutorialKeyShortcutsUITests.testBothDeclaredChordsPressTheButton` fails
  on iOS 26.5.** Control+Option+K does not press the Button. It is
  pre-existing (it fails with this session's iOS changes stashed). The sibling
  chord-on-a-box test passes. Not diagnosed.
- **Why a labelled RoleGroup with all-hidden children still made an element
  under `.combine`** (MessageBubble), when a labelled Box did not. The fix
  doesn't depend on the answer, but the rule isn't understood.
- **The guide goes blank-ish while a chapter 6 pushed screen is up**
  (carried). It shows a note, not the lesson.
- **Pointer to demo navigation** (carried). Needs a scroll-to host capability.
- **`hooks.UseWindow` in split mode** reports the browser window, not the
  400px phone (carried). The foldables lesson (4.x) is the one to check.
- **A brief phone-layout frame at boot** before the layout patch (carried;
  cosmetic).
- **An inline span node in core** (carried; renderer work, a plan of its
  own). It unblocks `RichTextView`, an inline `Link`, and text decoration.
- **A per-corner radius in core** (carried; renderer). It unblocks the range
  band's endpoint notch, which is still visible on both natives as expected,
  and bubble tails.
- **Breadcrumb's ghost buttons draw a faint frame in static exports**
  (carried). This is a Button-level decision.
- **A fourth low-hanging-fruit round** (carried). No candidates gathered.
- *Non-goal, declined:* making iOS keep focus on every submit by default. iOS
  convention is that Done dismisses; widgets that want to continue ask with
  `core.Focus`, as TagInput now does.
- *Non-goal, declined:* a two-pane layout on the natives (carried).
- *Non-goal, declined:* restructuring lesson bodies into guide and demo halves
  (carried).
- *Non-goal, declined:* a year-wide scrolling `CalendarHeatmap` (carried).
- *Non-goal, declined:* a `MessageList` / thread widget until a host can
  report or accept a scroll offset (carried).
- *Non-goal, declined:* a caption-flip "Copied ✓" on CopyButton (carried).
- *Non-goal, declined:* a separate `Alert` widget. Banner is it (carried).
- *Non-goal, declined:* Heatmap as a continuous gradient (carried).
- *Non-goal, declined:* a Sequential ramp interpolated from `Primary`
  (carried).
- *Non-goal, declined:* a native time wheel, and a sheet or Done on
  TimePicker (carried).
