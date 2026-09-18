# Next list: Paragraph, corners, scroll-to, a UIKit field, and a one-field PIN

**Session:** 309f8e63-ebe4-4638-85ac-fbd5b7623ce9
**Date:** 2026-09-18 13:10 (follows "next-list-textfieldstate-editor-stamps-button-floor-compact-copy")
**Branch:** master (1ae9528 → this commit)

## The ask

"Do all items in the Next list except hardware dep ones. Use the emulator /
simulator as necessary." Then `/sess-wrap` when completely done.

- **Skipped as hardware:** F-keys through GameController, and the first
  hardware key lost after launch.
- **Devices:** Android emulator `emulator-5554` (API 36), iOS simulator
  iPhone 17 Pro (iOS 26.5, `7572B953-…`), and Chrome for the web tutorial.

## 1. Android: text typed at a heading's end drew in the body face

- **Cause:** the block spans in `GrMobRichText.kt` are EXCLUSIVE_EXCLUSIVE,
  so text typed at a heading's very end fell outside them.
- **Fix:** `GrMobRichMapper.stretchBlocks`, called in `afterTextChanged`
  before the document is read. A block's marker and drawing spans are
  stretched to the end of the paragraph they end in.
  - INCLUSIVE flags were rejected: they would also swallow the `\n` of Enter
    at a heading's end, and the new paragraph would become a heading.
- **Emulator (4.14):** "A notes HERE" draws in the heading face; Enter then
  "body" draws plain. The Markdown reads `## A notes HERE` then `body`.

## 2. Android Button accessibility: not a split, a doubled caption

The previous doc said a labelled Button's nodes were split. Measured with
TalkBack, that was a misreading.

- **Method:** TalkBack's utterances are logged when it has no TTS engine.
  Google TTS was disabled (`pm disable-user com.google.android.tts`), focus
  was moved with Tab, and TTS was re-enabled afterwards.
  - Injected `adb input` taps and Alt/Meta+arrow did not drive TalkBack.
- **The dump's "labelled node":** Compose's *fake* child, which publishes a
  merging node's contentDescription. It is never a TalkBack stop, so there
  was no split.
- **The real defect:** the caption was read after the name.
  - Before: "Copy code. Copies to the clipboard, Copy, Button".
  - After: "Copy code. Copies to the clipboard, Button".
- **Fix:** `GrMobStyle.contentSemantics(caption)` puts
  `clearAndSetSemantics` on the label Text. `buttonBox` strips the a11y
  fields from the outer modifier. Both the material3 and long-press paths
  use it.

## 3. TextArea on devices (lesson 2.3)

- **Demo:** 2.3 gained a TextArea with a line and character count and a
  "Tidy lines" rewrite (`lineSummary`, `tidyLines`), plus Go tests.
- **Emulator:** multiline typing works; the rewrite lands while the field is
  focused; the caret is clamped to the end; mid-text typing works.
- **iOS:** `testTextAreaTakesLinesAndATidyRewrite`.

## 4. Typing replay: a three-way merge (`internal/rebasefixture`)

- **The rule:** `rebaseEdit` and `rebaseCaret` read basis→local and
  basis→rewrite as one span each, and carry the user's span across Go's
  change.
  - The mapping is exact outside Go's span, and identity inside a span that
    kept its length (UPPERCASE).
  - Where both inserted at one point, the user's text goes after Go's, which
    keeps PINInput's "14".
  - Surrogate pairs are never split.
- **Reference and table:** `internal/rebasefixture`, a Go reference with 16
  cases.
  - `android/verify` (Harness.kt, gen.go) and `ios/verify` (rebase.swift,
    gen.go) run both hosts' copies against it.
  - All 16 cases agree on both hosts.
- **Emulator (2.3, UPPERCASE):** a mid-text burst via `adb input text`
  landed whole: `HELLOABCDEFGHWORLD`, then two 12-key bursts.

## 5. iOS text fields are UIKit now (`GrMobTextInput.swift`)

- **Found by a new test:** under 2.3's UPPERCASE, one XCUITest
  `typeText("hello world")` gave "HELLO WOD".
  - Logging showed the SwiftUI `TextField`'s binding never reported
    h/i/j/k. It is a second copy of the text, and a key arriving during a
    rewrite was overwritten.
- **The replacement:** `GrMobTextField` hosts a `UITextField`, or a
  `GrMobTextAreaView` (a `UITextView` with a placeholder) for TextArea. The
  old private struct is removed from Renderer.swift.
  - Focus commands go through `GrMobEditorFocus`, now taking any `UIView`.
  - Also carried over: submit via `textFieldShouldReturn` (it resigns, as
    SwiftUI did), `returnKeyType`, disabled from `environment.isEnabled`,
    `GrMobTextLook` for font, colour and alignment, and a macOS stub.
- **Rewrites are written with `UITextInput.replace` of the differing span.**
  Assigning `text` made UIKit deliver queued keys inside the setter, with no
  editingChanged.
- **The replaced span ends at the caret when the caret follows the change.**
  `replace` leaves the caret at the end of the replacement. A correcting move
  afterwards let a queued space land at the old spot: "HELL o", then "HEL
  LWOORLD".
  - Keys delivered during a write are detected by reading the text back and
    are sent to Go.
  - Five of five runs pass after the fix.
- **Labels go on the UIKit view** (`applyAccessibility`), and
  `GrMobTextField.boxStyle` strips them from grMobBox. grMobBox's
  `.accessibilityElement` + label had wrapped the field in an unlabelled
  "Other".
  - `TutorialNativeFloorsUITests` was querying that wrapper; it now queries
    `textFields["Link address"]`.
- **`grMobValueText` is now conditional** (`GrMobValueTextModifier`).
  `accessibilityValue("")` blanked every text field's value, so VoiceOver
  read every Input as empty, the old SwiftUI field included. The pin in
  `mobile/verify/value_test.go` is updated.

## 6. Web tutorial items

- **Boot frame:** a package-level `OnHostEvent` records the layout sent
  before any tree subscribes, and `App` seeds `split` from `bootSplit()`.
  - The page now sends the layout before `RenderInitial`.
  - The Go test passes. Proving the absence of the frame in Chrome failed on
    harness quirks: occluded-tab rAF, and hooks that never saw the iframe's
    document.
- **Chapter 6 pushed screen:** the guide shows the last lesson tree
  (`coveredLesson`), made `inert`.
  - Callbacks are stripped (the runs' `cb` included), focus commands are
    dropped, and nodes that had callbacks are marked Disabled.
  - A bar (`#tutorial-guide-note`) goes *after* the lesson, so the lesson
    keeps its slot.
  - **Chrome:** the same element before and after, scroll 400 kept across
    both the push and the pop, Next disabled while covered.
- **`hooks.UseWindow`:** the page names its window with
  `window.GrMobViewport`. The runtime measures that element, watches it with
  a ResizeObserver, and re-tracks it after each mount and patch; a named
  element reports no fold.
  - **Chrome 4.21:** 336×723 compact in the split, 406×792 in phone-only.
    Both used to report the browser window.
- **Pointer to demo:** the pointers are buttons that call
  `core.ScrollIntoView`.
  - **Chrome 4.3:** the phone smooth-scrolled to 169 of 183.

## 7. `core.ScrollTarget` / `core.ScrollIntoView` (new capability)

- **Wire:** the latest command's target carries `scrollEpoch`. Each host
  acts once per epoch, keeping an app-wide high-water mark (the web runtime
  resets it on mount).
- **Semantics:** the platform's own bring-into-view. The web uses
  `scrollIntoView({block:"nearest"})` (smooth unless reduced motion);
  Compose uses a `BringIntoViewRequester` after one frame; SwiftUI uses
  `ScrollViewReader.scrollTo(viewID)`.
- **iOS plumbing:** GrMobScroll provides the proxy through
  `\.grMobScrollProxy`, and `GrMobBringIntoView` is applied unconditionally
  in RenderNode.
  - These are named to avoid the `private struct GrMobScroll` prefix that
    mobile/verify pins match on.
- **Demo:** lesson 1.5 has a 160px Scroll of twelve rows with "Jump to row
  10" and "Back to row 1".
  - Verified on the emulator and in `testScrollIntoViewJumpsInsideAScroll`.
  - The iOS test judges against the box's own frame, re-read, because
    SwiftUI also scrolls the lesson around the box.

## 8. `core.CornerRadii` (per-corner radius)

- **Core:** `Style.Corners`, four physical radii in CSS order, with
  `Style.Radii()` stating the precedence. `BorderRadius` clears it, and the
  merge takes all four corners whole.
- **Hosts:**
  - htmlout and the web runtime write four values (`radiusCSS`);
    `FULL_STYLE` gained Corners and the missing AccentColor.
  - Android: `grMobShape` → `AbsoluteRoundedCornerShape`.
  - iOS: `grMobShape` → `UnevenRoundedRectangle(.circular)`, with
    leading/trailing swapped under RTL.
- **Users:**
  - Calendar range endpoints are square toward the band, so the notch is
    gone.
  - MessageBubble has a tail: the bottom corner on the sender's side at 4
    against 16.
  - Verified on the emulator and the simulator.

## 9. `core.Paragraph` (the inline span node)

- **Node:** `Paragraph([]core.Span, props…)`, one node whose runs travel in
  the `runs` prop. Keys: `t b i u s c fg cb`.
  - A link run with no colour gets the theme's Primary, resolved in Go.
- **Hosts:**
  - Web: a `<div>` of `<span>`s; a link is `role=link` with tabindex 0,
    pressed by click or Enter.
  - htmlout: runs are written on one line, because spans broken onto their
    own lines would read as spaces.
  - Android: an AnnotatedString with `LinkAnnotation.Clickable`.
  - iOS: an AttributedString; links are `grmob-run:` URLs caught by an
    `OpenURLAction`, and `.tint` is the first link's colour.
  - iOS min-content floors a Paragraph like a Text.
- **Built on it:**
  - `comps.RichTextView`: blocks → Paragraphs; list runs grouped as RoleList;
    an empty block becomes one space.
  - `Link.Span(ctx)`: an inline link, underlined.
  - Lesson 4.14 shows a live RichTextView under the editor.
  - **Verified:** the emulator draws it and the link opens Chrome; iOS
    exposes one text plus a link element; Chrome draws it.

## 10. PINInput is one field

- **Design:** the boxes are drawing. One `core.Input` (`InputPassword` when
  Secure) holds the code: 1px, transparent, the ZStack's first layer (under
  the boxes), bottom-start, inset 8.
  - The row takes taps and calls `core.Focus`.
  - The next box is bordered in Primary while the field has focus.
  - The row and the ZStack are `Width(100%)`, because the web's grid had
    shrunk them.
- **Tests:** the widget tests are rewritten; lesson 5.7 (renamed "six
  boxes, one field") and its test are rewritten; docs/components.md is
  updated.
  - The render ledger test now builds its own three-box field.
- **Emulator:** "314159" at 130ms and "271828" at about 32ms per key land
  whole, where the six-field version lost digits.
- **iOS:** `testPINInputTakesABurstThroughOneField` (number pad, a one-call
  burst, backspace). Chrome works too.

## 11. Low-hanging round 4

- **`core.Keyboard(kind)`** (`core/keyboard_kind.go`): digits, decimal,
  phone, email, url.
  - Web and htmlout write `inputmode` (`htmlout.InputModeFor`).
  - Android sets KeyboardOptions (digits on a password field is
    NumberPassword).
  - iOS sets `keyboardType`, and a digits text field gets
    `.oneTimeCode`.
  - PINInput uses it.
- **Transparent buttons drop the theme's elevation** (`Shadow(0)` on
  Outlined and Ghost). That was the Breadcrumb's faint frame in exports.
- **Tutorial lesson screens are `KeyboardAware`.** On Android, 5.7's boxes
  sat under the keyboard with nothing able to reveal them.
- **Already done before this session:** htmlout box-sizing and
  CalendarHeatmap ties.

## Pitfalls

- **Write/`cat >` overwrote tracked `core/keyboard.go`** (KeyboardAware). It
  was restored with `git checkout` and the new code moved to
  `keyboard_kind.go`. Check `git ls-files` before creating a file with a
  plausible name.
- **uiautomator dumps Compose's unmerged tree**, fake nodes included. Judge
  TalkBack by its utterances instead.
- **The Chrome tab is occluded:** rAF-gated work (focus commands,
  scrollIntoView) runs only when a screenshot forces a frame. Long awaits
  can hang CDP and lose typed keys.
- **The iOS Xcode project is generated** (`cd ios && xcodegen generate`) for
  new files. Temporary probe tests belong inside an existing class.
- **XCUITest:**
  - A SwiftUI field's `value` was blank (the accessibilityValue bug).
  - Swipes also scroll inner Scrolls.
  - `identifier:` matches identifiers, not labels.
- **`grep -o` of a class name** matched `GrMobScroll…` prefixes in
  mobile/verify pins.
- **The wasm/verify file-count census** went 609 → 621.
- **Android's package is `com.grmob.app`;** iOS's bundle is `com.grmob.demo`.

## Files

- **Go core:**
  - `corners.go`, `scroll_to.go`, `paragraph.go`, `keyboard_kind.go`
    (+ tests);
  - `style.go`, `style_props.go`, `context.go`, `text_edit.go`.
- **comps:**
  - `rich_text_view.go` (+ test), `pin_input.go`, `link.go`, `button.go`,
    `calendar.go`, `message_bubble.go`;
  - tests updated.
- **htmlout:** `export.go`, `tag.go`.
- **internal:** `rebasefixture/` (new), `apidoc/packages.go`.
- **render:** `text_edit_test.go`.
- **Tutorial:** `app.go`, `split.go`, `widgets.go`, `lesson_screen.go`, and
  chapters 1, 2, 4 and 5, with their tests.
- **Web:** `wasm/index.html`, `wasm/grmob-runtime.js`; new
  `wasm/verify/{scrollto,corners,paragraph}_test.mjs`, plus
  `window_test.mjs` and `totality_test.mjs`.
- **Android:**
  - `Renderer.kt`, `GrMobStyle.kt`, `GrMobRichText.kt`,
    `GrMobTextEdits.kt`, `GrMobRuntime.kt`, `GrMobMapView.kt`;
  - `android/verify/{Harness.kt,gen.go,run.sh}`.
- **iOS:**
  - `GrMobTextInput.swift` (new), `Renderer.swift`, `GrMobStyle.swift`,
    `GrMobTextEdits.swift`, `GrMobEditorFocus.swift`,
    `GrMobMinContent.swift`, `GrMobRuntime.swift`;
  - `ios/verify/{rebase.swift,gen.go,main.swift,run.sh}`;
  - UI tests.
- **mobile/verify:** `button_border_test.go`, `value_test.go`.
- **Docs:** `docs/components.md`, `docs/api/*` regenerated.

## Verification

- `go test ./...` and `go vet` are clean.
- android/verify, ios/verify, wasm/verify and mobile/verify pass.
- **iOS UI tests:**
  - All tutorial classes pass after the two fixes.
  - TodoApp passes.
  - GrMobUITests' feed test and AudioUITests fail identically at 1ae9528
    (checked in a worktree), so both predate this session.

## Next

- **GrMobUITests.testFeedListGesturesAndVirtualization fails at HEAD**
  (new, pre-existing). "Article 3, selected" never appears as the row's
  label after a tap.
- **AudioUITests.testAudioTransport never reaches "playing"** on the
  simulator (new, pre-existing; possibly network or environment).
- **The web runtime's keyboard prop is not cleared** when a patch drops the
  key (new, minor). An update-props patch without `keyboard` leaves the old
  inputMode.
- **iOS link runs in one Paragraph share the first link's tint** (new; a
  SwiftUI limit). Differently coloured links draw alike there.
- **iOS UIKit field: a rewrite that changes text after the caret** still
  sets the caret after `replace` (new). A key arriving in between lands
  early. That case is rare; caret-at-or-after is the common one and is fixed.
- **Prove the web boot frame is gone in a real browser** (new). The Go test
  covers it; the Chrome probes failed on harness quirks.
- **A thread widget:** ScrollIntoView can open at the newest message, but
  loading older ones needs a *reported* scroll offset (updated from
  "needs a scroll offset").
- **F-keys through GameController have never reached the app from
  XCUITest** (carried; needs hardware).
- **The first hardware key after launch is lost on the iOS 26.5 simulator**
  (carried; needs a real iPad).
- **A fifth low-hanging-fruit round** (carried shape). No candidates
  gathered yet.
- *Non-goal, declined:* a smaller minimum for Buttons in general on Android.
  Only CopyButton opts out (carried).
- *Non-goal, declined:* making iOS keep focus on every submit by default
  (carried). The UIKit field keeps SwiftUI's resign-on-return.
- *Non-goal, declined:* a two-pane layout on the natives (carried).
- *Non-goal, declined:* restructuring lesson bodies into guide and demo
  halves (carried).
- *Non-goal, declined:* a year-wide scrolling `CalendarHeatmap` (carried).
- *Non-goal, declined:* a caption-flip "Copied ✓" on CopyButton (carried).
- *Non-goal, declined:* a separate `Alert` widget. Banner is it (carried).
- *Non-goal, declined:* Heatmap as a continuous gradient (carried).
- *Non-goal, declined:* a Sequential ramp interpolated from `Primary`
  (carried).
- *Non-goal, declined:* a native time wheel, and a sheet or Done on
  TimePicker (carried).
- *Non-goal, declined:* a right-padding gutter or a toolbar header for code
  blocks (carried).
