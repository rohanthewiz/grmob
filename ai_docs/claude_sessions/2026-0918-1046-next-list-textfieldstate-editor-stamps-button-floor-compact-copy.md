# Next list: TextFieldState, editor stamps, the Button floor, a compact Copy

**Session:** f58e78e2-0816-43db-be6f-39caee0ed101
**Date:** 2026-09-18 10:46 (follows "next-list-copy-strip-edit-epochs-accent-and-the-lost-first-key")
**Branch:** master (6ef8089 → this commit)

## The ask

"Do the new items in the Next list except for any requiring hardware. As much
as possible use the emulator / simulators." Then: "Go with option 2" for the
copy strip, and `/sw`.

- **Skipped as hardware:** F-keys through GameController, and the first
  hardware key lost after launch. Both need a real iPad.
- **Devices:** Android emulator `emulator-5554`, iOS simulator iPhone 17 Pro
  (iOS 26.5, `7572B953-…`), both already booted.

## 1. Android lost keys after Go replaced a field's text

### Measured

`adb shell input text "alpha,beta,gamma,delta,"` into lesson 5.8's TagInput:

| build | tags committed |
|---|---|
| Compose 1.6.8, value/onValueChange field | `bea gmma dlta` |
| Compose 1.7.6, same field | `bta amma dlta`, `eta gmma dlta`, `bta gamma dlta` |
| Compose 1.7.6, TextFieldState field | `alpha beta gamma delta` ×3 |

- **The bump alone did not fix it.**
- **Cause:** the legacy field keeps the text twice: the caller's value, and
  the buffer the key handler and the IME write into. It reconciles the two
  when a new value arrives, so a key landing between a rewrite and that
  reconciliation is overwritten.

### The fix

- **Compose BOM** `2024.06.00` → `2024.12.01`: foundation 1.7.6, material3
  1.3.1. It is the newest line that still builds against compileSdk 34.
- **`GrMobTextField` (Renderer.kt) now uses a `TextFieldState`,** one buffer.
  - **User edits:** read by `snapshotFlow { field.text }`. The flow
    conflates, which is fine because Go needs only whole values.
    `LastText` keeps Go's own writes from being read back as typing.
  - **Upstream handling:** moved from the composition body into
    `LaunchedEffect(seen, focused)`, because a state is edited from outside
    composition.
  - **Password:** `BasicSecureTextField` with `TextObfuscationMode.Hidden`.
    `KeyboardActionHandler` replaces `KeyboardActions`, and
    `TextFieldLineLimits` replaces `singleLine`.
  - **The file comment** has the table above.
- **A regression caught and fixed:** a focused rewrite first used
  `setTextAndPlaceCursorAtEnd`. That sent the caret to the end on every
  keystroke of an UPPERCASE transform. A focused rewrite now keeps the caret
  offset, clamped to the new length, which is the legacy field's behaviour.

### Bump fallout

- **Deprecations:**
  - `KeyboardOptions(autoCorrect=)` → `autoCorrectEnabled` (the pin in
    `mobile/verify/codeeditor_test.go` was updated).
  - `animateItemPlacement` → `animateItem(fadeInSpec = null, placementSpec,
    fadeOutSpec = null)`. The fades are off so a List Transition still means
    placement only.
- **The census check:** foundation 1.7 moved the Row measure loop from
  `RowColumnMeasurementHelper.kt` to `RowColumnMeasurePolicy.kt`. The
  arithmetic is the same, now in plain Ints.
  - `TestTheComposeCensusClaimsAreWhatTheSourceSays` got the new spelling,
    plus a 1900-byte window for the loop claim: a flow-layout branch pushed
    its last line to 1726 bytes.
  - `internal/pinfixture` quotes 1.7.6.
- **Present-tense version statements** in `docs/platforms/native.md`,
  `build.gradle`, `composelayout_test.go` and `banddistribution_test.go`
  are now past tense.

### Checked on the emulator

- Placeholder.
- UPPERCASE transform, with typing mid-text at human pace:
  `HELLOXYZWORLD`.
- Clear while focused.
- Password masking and the password keyboard (5.4).
- Enter moves Country → City (4.17).

## 2. Every field once the host is sequenced (core/text_edit.go)

Lesson 5.7's PINInput lost digits under a burst. A field's ledger was made by
its first edit, so Go rewriting a never-edited cell (a fast "paste" into cell
0 spilling into cell 1) was never counted as a rewrite.

- **The rule:** the first `TriggerTextEdit` sets
  `callbackRegistry.sequenced`. From then on every text field gets a ledger
  at its first render, with `hostValue` = the rendered value.
- **Hosts:** "stamped" now means the node *has* `editEpoch`, not
  `editSeq != 0`. `TextEditLedger.upstream(..., stamped)` on both natives,
  six call sites.
- **Go test:** `TestAFieldGoRewroteBeforeItsFirstEditDropsTheStaleKey` replays
  the emulator's sequence. It fails without the change.

### What still loses PIN digits (logged, not fixed)

| on-device spacing | old field | new field |
|---|---|---|
| ~530ms, ~330ms | clean | clean |
| ~230ms | `3 1 4 1 5 _` | clean |
| ~130ms and faster | lossy | lossy |

Temporary logs (since removed) showed two mechanisms:

- **Keys that never reach any field.** They arrive while focus moves between
  cells (the "5" never sent).
- **A cell blurring before its rebase.** Cell 0 sent "3", "31", "314". Go
  dropped "314" at the old epoch, and the cell had lost focus, so nothing
  replayed it. Per-cell replay could not place it correctly anyway: only a
  single hidden field owning the whole code could.

## 3. CodeEditor and RichTextEditor joined the protocol

- **Go:** `textEditFields` (it was `textEditLeafTypes`) maps a node type to its
  value prop and an optional `canon`.
  - `RichTextEditor` uses `"doc"` with `canonicalDocJSON`. Go compares the
    host's JSON with its own render as documents, because org.json escapes
    `/` and neither host writes encoding/json's key order.
  - `stampEdit` parses only when the bytes differ.
- **Hosts:**
  - `TextEditLedger(replay:)`: rich text uses `replay = false`, since
    splicing JSON strings is not a document.
  - `rebaseCaret`: the code editor puts the caret where the replayed typing
    was.
  - Both editors dedupe on the (value, seq, epoch) triple.
  - Rich text keeps `lastGoJson` so a blurred editor tells Go's document from
    its own despite the spelling difference.
- **Tests:**
  - `render/text_edit_test.go`: CodeEditor stamps, a host-spelled echo, an
    unreadable payload.
  - `mobile/verify` pins rewritten from the value queue to the ledger.
  - iOS UI tests `testRichTextTypingReachesGoWithTheHeadingPlain` and
    `testCodeEditorTypingSurvivesItsEchoes`.
- **Emulator:**
  - 4.13: typing is intact and the caret report agrees.
  - 4.14: `a/b,c\<d\>\&e,alpha/beta,gamma` matches between the editor and Go's
    Markdown.

## 4. Found along the way

- **Rich-text headings became bold marks, on both hosts.** A heading is drawn
  bold, and the reverse mapping read that as the user's mark, so the first
  keystroke made `## **A note**`.
  - Android: `GrMobBlockBoldSpan` and `GrMobBlockMonospaceSpan` are
    subclasses, skipped by `document`, `marksIn` and the toggles.
  - iOS: `GrMobRichMapper.boldMark(attrs, blockKind:)` judges bold by the
    paragraph's kind, because text typed at the end of a heading inherits the
    bold font but not the custom block-kind key.
- **The iOS code editor gutter.** A UILabel centres its lines in a frame
  taller than they are, and the frame was at least the box's height. In
  4.13, "1" sat beside line 4. The label is now fitted to its own height.
  The trailing-gap line was `insetBy(dx: 0, dy: 0)`; it now gives up a
  column.

## 5. "Sav / e" on iOS (lesson 2.6)

- **Cause:** `GrMobMinContent` floored every Button at 0, so a long caption
  in the same Row squeezed the button.
- **Fix:** `buttonWidth` floors a Button at its widest word plus the padding
  `GrMobButton` draws (16 a side by default), plus its margin, clamped by a
  points MaxWidth.
- **Tests:** `ios/verify/mincontent.swift` checks the default padding, widest
  word, own padding and margin, cap, empty label and declared width.
  `testTheSaveButtonKeepsItsLabelOnOneLine` failed at 59.7×60.7 and passes
  after.

## 6. The compact Copy button (option 2)

- **Go:** `CopyButton.Render` sets `touchTarget: "compact"` on the Button node
  it returns. No public core API, so the opt-out is this widget's alone.
  `TestOnlyCopyButtonIsCompact` covers it.
- **Android:** `GrMobButton` wraps `GrMobMaterialButton` in
  `LocalMinimumInteractiveComponentSize provides Dp.Unspecified`, and gives
  the modifier `heightIn(min = 1.dp)`. `defaultMinSize` applies only when the
  incoming minimum is 0, which disables ButtonDefaults.MinHeight.
  - Result: 126px (48dp) → 63px (24dp). The rest of the height is material3's
    20sp label line height.
  - **Touch is still 48dp:** Compose widens a small clickable's touch bounds
    without laying it out larger. The first try, the Surface path, measured a
    126×118 clickable around a 47px button.
- **Why not the Surface path:** it split the label and the click into
  separate accessibility nodes. Reverted.
- **Verified:** copying 5.8's snippet and pasting it into the tag field
  committed `comps.FormField{ Label: "Labels"`.

## Pitfalls

- **Harness taps were the "bugs" twice.**
  - A key-point bullet containing "the Clear button" matched my Clear search.
  - With the keyboard up, a tap at y=1657 typed "w".
  - Match `text="Clear"` exactly, and hide the keyboard first.
- **uiautomator writes multiline text as `text='…'`**, not `text="…"`.
- **The bundle ID is `com.grmob.demo`,** not `com.grmob.GrMobApp`.
- **On-device timing:** `adb shell "input text 3; sleep 0.1; …"`. One `input`
  call costs about 32ms.
- **`println` in Go shows in logcat** (twice per call).
- **`docs/api` goes stale on doc-comment edits:** run
  `go run ./internal/apidoc/gen`.
- **The `mobile/verify` failures were cut off by `tail -5`.** Read the whole
  output.
- **XCUITest:**
  - `element.value(forKey: "hasKeyboardFocus")` failed to snapshot. Query with
    `hasKeyboardFocus == true` instead.
  - UIKit snaps a tap's caret to a word boundary.

## Files

- **Go:**
  - `core/text_edit.go`, `core/event.go`, `core/codeeditor.go`,
    `core/richtext.go`;
  - `comps/copy_button.go`;
  - `internal/pinfixture/pinfixture.go`.
- **Tests:**
  - `render/text_edit_test.go`, `comps/copy_button_test.go`;
  - `mobile/verify/{codeeditor,richtext,composelayout,banddistribution}_test.go`;
  - `ios/verify/mincontent.swift`;
  - `ios/GrMobUITests/TutorialDevicePassUITests.swift` (Save, two editor
    tests).
- **Android:**
  - `app/build.gradle` (BOM);
  - `Renderer.kt` (TextFieldState field, compact button, `animateItem`);
  - `GrMobTextEdits.kt`, `GrMobCodeEditor.kt`, `GrMobRichText.kt`.
- **iOS:**
  - `GrMobTextEdits.swift`, `GrMobCodeEditor.swift` (ledger, gutter);
  - `GrMobRichText.swift`, `GrMobMinContent.swift`, `Renderer.swift`.
- **Docs:** `docs/platforms/native.md`, `docs/api/*` regenerated.

## Next

- **PINInput loses digits on Android at ≤130ms per key** (new). The causes are
  keys dropped during a focus move, and per-cell paste semantics. The real
  fix is a single hidden field owning the code; that is a widget redesign.
- **No TextArea (multiline) is exercised on a device** (new). No tutorial
  lesson has one; the multiline path of the TextFieldState field is
  unverified.
- **Mid-text typing at machine speed under a rewriting transform drops keys**
  (new; the documented rebase limit). Correct at human pace.
- **Android: text typed at the end of a rich-text heading draws in the body
  face** until the next rebuild (new, cosmetic). The heading spans are
  EXCLUSIVE_EXCLUSIVE. The document is right.
- **Android: Button accessibility nodes are split** (new). The labelled node
  is not clickable, and the clickable node has no label. This applies to every
  material3 Button carrying `core.AccessibilityLabel`. Check it with TalkBack.
- **F-keys through GameController have never reached the app from
  XCUITest** (carried; needs hardware). `testFunctionKeyPressesTheButton` is
  a strict expected failure.
- **The first hardware key after launch is lost on the iOS 26.5 simulator**
  (carried; needs a real iPad).
- **The guide goes blank-ish while a chapter 6 pushed screen is up**
  (carried).
- **Pointer to demo navigation** (carried). Needs a scroll-to host
  capability.
- **`hooks.UseWindow` in split mode** reports the browser window, not the
  400px phone (carried).
- **A brief phone-layout frame at boot** before the layout patch (carried;
  cosmetic).
- **An inline span node in core** (carried; renderer work, a plan of its
  own).
- **A per-corner radius in core** (carried; renderer).
- **Breadcrumb's ghost buttons draw a faint frame in static exports**
  (carried). This is a Button-level decision.
- **A fourth low-hanging-fruit round** (carried). No candidates gathered.
- *Non-goal, declined:* a smaller minimum for Buttons in general on Android.
  Only CopyButton opts out (option 2).
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
  blocks (carried).
