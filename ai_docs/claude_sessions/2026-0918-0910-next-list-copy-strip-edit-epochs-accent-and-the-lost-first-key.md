# Next list: copy strip, edit epochs, accent, and the lost first key

**Session:** fb302ba7-8174-49f1-9ccd-fbb62f9ce09d
**Date:** 2026-09-18 09:10 (follows "device-pass-nine-bugs-the-simulators-found")
**Branch:** master (1cf990e → this commit)

## The ask

"Do the new items in the Next list": the five items the previous session
raised, not the carried ones.

The Android emulator (`emulator-5554`) and the iOS simulator (iPhone 17 Pro,
iOS 26.5) were already booted.

## 1. The copy button no longer covers code

The user picked **top padding on the editor** from three options (top
padding, right padding, a toolbar strip).

A fixed padding cannot be right on every target:

- Go has no platform query.
- The button is ~22px tall on the web but 48dp on Android. A material3
  Button is 40dp at minimum and lays out at the 48dp touch minimum.

So `codeBlock` (examples/tutorial/widgets.go) is no longer a ZStack. It is
now:

```
Column (scheme.Bg, radius 10, width 100%)
├ Row (Justify end, PaddingTop 6, PaddingHorizontal 6) → CopyButton
└ CodeEditor (Padding 14, PaddingTop 4)
```

- The strip is the editor's colour with no rule under it, so it reads as top
  padding. Its height is the button's on each platform.
- The ZIndex workaround went with the ZStack.
- A chapter4 test comment that called codeBlocks "two-layer ZStacks" was
  updated.
- Seen on both devices. On Android the band is tall because the button is.

## 2. The text-edit protocol (the Android echo race)

### The protocol

`core/text_edit.go` holds the protocol; the file comment has the full
diagram.

- **Host → Go:** `mobile.TriggerTextEdit(id, value, seq, epoch)`, carried by
  `render.Manager.DispatchTextEdit` and `Context.TriggerTextEdit`.
  - `seq` is a runtime-wide counter, touched only on the main thread.
  - `epoch` is the rewrite count the field last adopted.
- **Go → host:** the props `editSeq` and `editEpoch` on Input, InputPassword,
  NumericInput and TextArea. They are stamped in `leafNode` via
  `stampTextEdit`.
- **Go's ledger:** one per text callback ID, holding `seq`, `epoch` and
  `hostValue`.
  - A render whose value differs from `hostValue` is a rewrite, so the epoch
    is bumped.
  - An edit carrying an older epoch is dropped.
  - Ledgers are purged along with their callbacks.
- **No stamps without the new call:** a field that no `TriggerTextEdit` has
  reached gets none. So the web (synchronous dispatch), exports and every
  existing test are unchanged.
- **The host's half:** `TextEditLedger` plus `rebaseEdit`, in
  `GrMobTextEdits.kt` and `GrMobTextEdits.swift`.
  - A higher epoch means a rewrite.
  - The basis is the pending entry at `editSeq`.
  - `rebaseEdit` replays insertions at either end of the text onto the
    rewrite. Otherwise Go's text wins.
  - With no stamps, it falls back to the old value queue.
- **Both natives' `GrMobTextField`s use it.**
  - Android reacts to a change in the (value, seq, epoch) triple.
  - iOS reacts to `onChange(of: EditStamp)`, which fires even when only the
    ack moves.
- **Bonus:** a refused edit (Go keeps the old value) is now a rewrite too.
  The value queue never saw it, because no patch came.

### Emulator results

These runs used `adb shell input text "alpha,beta,gamma,delta"` into
lesson 5.8.

- **Bug found on the first run:** v1 advanced `editSeq` on *dropped* edits.
  The host then took a dropped edit ("eta,g") as the basis and replayed only
  "a" of "ga".
  - Fix: only applied edits advance `seq`.
  - `TestAKeystrokeTypedBeforeARewriteIsDroppedAndItsRebaseApplies` now pins
    that a dropped edit patches nothing.
- **Old code vs new** (the old code was checked by temporarily restoring
  HEAD's Renderer.kt):
  - Old: scrambled tags such as "gaba", "ltad", "gabma", "eelta".
  - New: every committed tag is exactly the text the field showed.
  - The logs showed each rebase working: "b", "ga", "d".
- **Residual, in both old and new code:** a key or two right after a
  programmatic text replace never reaches `onValueChange`.
  - It happens with every IME disabled, and InputDispatcher logs no drops.
  - A plain field typed at the same speed loses nothing.
  - It is Compose 1.6.8's legacy `BasicTextField` (BOM 2024.06.00). See
    Next.

### iOS

- New `testTagInputAtMachineSpeed` passes.
- **The old code passes it too** (run twice for the purpose). `typeText`
  paces its keys, so no keystroke is ever in flight when a rewrite lands. The
  test's doc says so: it guards the new path and does not reproduce the race.

### Go tests

`render/text_edit_test.go` covers:

- a keystroke dropped and then rebased;
- a refused edit counting as a rewrite;
- the unsequenced (web) path stamping nothing.

## 3. `AccentColor`

- **The field:** `core.Style.AccentColor` plus `core.AccentColor(hex)`.
  - The Switch, Checkbox and Slider builders default it to
    `Colors.Primary` via `accented(ctx, base)` in slider.go. A theme base
    that names its own accent keeps it.
  - It is a builder default rather than a theme field, because Slider reads
    no theme base and the accent is not a new palette role.
- **What each target does with it:**

| target | mapping |
|---|---|
| wasm runtime | `out.accentColor` |
| htmlout | `accent-color:` |
| Compose | `CheckboxDefaults.colors(checkedColor)`; `SwitchDefaults.colors(checkedTrackColor, checkedBorderColor)`; `SliderDefaults.colors(thumb, activeTrack, inactiveTrack at 24% alpha)` |
| SwiftUI | `.tint(node.style?.accentColor)` on both Toggles and the Slider. `tint` takes an optional, so this is not a branch |

- **Tests:**
  - core `accent_test.go` (default, override, only the controls carry it);
  - `TestSwitchReadsTheCheckBoxThemeBase` updated;
  - htmlout `TestAccentColorBecomesCSSAccentColor`;
  - mobile/verify `accent_test.go` (parse, and each control's slot, on both
    natives).
- **Docs:** a new `AccentColor` section in
  docs/concepts/styling-and-theming.md.
- **Seen on devices:**
  - Android: 2.6 Switch and Checkbox, 6.8 Slider, all blue.
  - iOS: new `testAccentOnPlatformControls` screenshots show the 2.6 Switch
    and 6.8 Slider in blue.

## 4. The iOS chord test: the first key press is lost

The test failed on the untouched HEAD (built in a scratch git worktree) as
well as on this session's code. The sibling box test failed too, though it
had passed last session.

**Two probes:**

- **First probe:** press while off screen, scroll, press again. Only the
  second press landed. This was misread as "SwiftUI keyboardShortcut needs
  the Button on screen".
- **Second probe:** press three times with no scrolling. The log went 0, 1,
  2. **The first hardware key event after launch is lost**, whatever it is.

**The fix:**

- `primeKeyboard` presses Control+Option+Z, which lesson 2.2 does not
  declare, and checks that it pressed nothing.
- The test was renamed `testTheModifierChordPressesTheButton`.

**F6:**

- It never arrives: four presses in a row did nothing, with "Connect
  Hardware Keyboard" both on and at its default.
- The 2026-09-13 session never observed it either.
- It now has its own test, `testFunctionKeyPressesTheButton`, wrapped in a
  strict `XCTExpectFailure`. The test fails, and says so, the day F6 starts
  landing.

`ConnectHardwareKeyboard` was written, then deleted again, and Simulator.app
was restarted. The simulator's settings are as they were.

## 5. Why a labelled group with only hidden children still made an element

This was measured with `.combine` forced back on, using temporary probes in
lesson 4.29 and an XCUITest dump. All of it was reverted.

- **One child makes no element:** every labelled container with a single
  hidden child. That held with or without RoleGroup, and with a background,
  border, radius, padding, MaxWidth, an end-aligned Text or a bold Text.
- **Two or more children make one:** a plain Column of two hidden Texts, and
  MessageBubble with a sender, a time or both.
- **Why:** with one child, SwiftUI folds the container into that child, and a
  hidden child takes the element with it.
- **So** a "mine" bubble with no Time had no VoiceOver element before last
  session's `labelOnly` fix.
- **Where it's written down:** the comment in
  `grMobAccessibility` (GrMobStyle.swift).

## Pitfalls

- **`go test` counts tracked Go files.** Four new files moved the census
  from 605 to 609 (`perl -pi -e 's/\b605\b/608/g'`, then 608 → 609).
  `go run ./internal/apidoc/gen` regenerated docs/api.
- **The Android verify harness compiles against `android/app/libs/grmob.aar`.**
  A new bridge function fails there until `android/build.sh
  ./examples/tutorial` has run.
- **A new Swift runtime file needs `cd ios && xcodegen generate`.** The
  project is untracked.
- **XCUITest `dump()` writes only with `TEST_RUNNER_GRMOB_SHOTS_DIR` set.**
  One probe run was wasted without it.
- **The sandbox blocked `sleep N; cmd` chains.** A background `until` loop
  was used instead.
- **PIL is not installed.** `sips -c H W --cropOffset Y X` crops PNGs.

## Files

- **Go:**
  - `core/text_edit.go` (new), `core/event.go` (ledger map, purge),
    `core/layout.go` (stamp hook);
  - `render/manager.go`, `mobile/bridge.go`;
  - `core/style.go`, `style_props.go`, `switch.go`, `input.go`, `slider.go`;
  - `htmlout/export.go`;
  - `examples/tutorial/widgets.go`.
- **Tests:**
  - `render/text_edit_test.go`, `core/accent_test.go`,
    `mobile/verify/accent_test.go` (all new);
  - `core/switch_test.go`, `htmlout/export_test.go`,
    `examples/tutorial/chapter4_test.go` (comment);
  - `wasm/verify/{repowalks,timings}_test.go` (609).
- **Android:**
  - `GrMobTextEdits.kt` (new);
  - `Renderer.kt` (text field, Checkbox/Switch/Slider colours);
  - `GrMobRuntime.kt` (`triggerTextEdit`, `textEdited`);
  - `GrMobStyle.kt` (`accentColor`);
  - `app/GomobileBridge.kt`.
- **iOS:**
  - `GrMobTextEdits.swift` (new);
  - `Renderer.swift` (text field, `.tint` ×3);
  - `GrMobRuntime.swift`, `GrMobStyle.swift` (`accentColor`, the combine
    finding);
  - `App/GomobileBridge.swift`;
  - `verify/gomobile_stub.swift`;
  - UI tests `TutorialDevicePassUITests.swift` (burst, accent) and
    `TutorialKeyShortcutsUITests.swift` (priming, F6 split).
- **Web:** `wasm/grmob-runtime.js` (`accentColor`).
- **Docs:**
  - `docs/concepts/styling-and-theming.md`;
  - `docs/api/*` regenerated.

## Next

- **Android loses keys right after Go replaces a field's text.**
  - Seen with `adb input text`, in both the old and new renderer, and with
    no IME.
  - The keys never reach `onValueChange`.
  - Likely Compose 1.6.8's legacy `BasicTextField`.
  - The candidate fix is `BasicTextField(state: TextFieldState)`
    (foundation 1.7+), which needs a Compose BOM bump.
- **CodeEditor and RichTextEditor still use the value-queue echo guard**, on
  both natives. They would take the same edit stamps: `textEditLeafTypes`
  plus their hosts.
- **F-keys through GameController have never reached the app from
  XCUITest.** `testFunctionKeyPressesTheButton` is a strict expected
  failure. The next check is a real iPad keyboard.
- **The first hardware key after launch is lost on the iOS 26.5 simulator.**
  The tests work around it. It is unchecked on a real iPad.
- **Lesson 2.6's Save button wraps as "Sav / e" on iOS.** Not investigated.
- **The Android copy strip is tall** (a 48dp button plus 6 of padding). It is
  cosmetic, and a smaller Button minimum on Android would be a Button-level
  decision.
- **The guide goes blank-ish while a chapter 6 pushed screen is up**
  (carried). It shows a note, not the lesson.
- **Pointer to demo navigation** (carried). Needs a scroll-to host
  capability.
- **`hooks.UseWindow` in split mode** reports the browser window, not the
  400px phone (carried). The foldables lesson (4.x) is the one to check.
- **A brief phone-layout frame at boot** before the layout patch (carried;
  cosmetic).
- **An inline span node in core** (carried; renderer work, a plan of its
  own). It unblocks `RichTextView`, an inline `Link`, and text decoration.
- **A per-corner radius in core** (carried; renderer). It unblocks the range
  band's endpoint notch and bubble tails.
- **Breadcrumb's ghost buttons draw a faint frame in static exports**
  (carried). This is a Button-level decision.
- **A fourth low-hanging-fruit round** (carried). No candidates gathered.
- *Non-goal, declined:* making iOS keep focus on every submit by default
  (carried).
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
- *Non-goal, declined:* a right-padding gutter or a toolbar header for code
  blocks. The user chose top padding.
