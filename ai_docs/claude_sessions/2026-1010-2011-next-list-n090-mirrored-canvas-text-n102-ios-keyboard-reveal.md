# Next list, part 3: N-090 mirrored canvas text, N-072's iOS keyboard check and N-102

Session: `1e5d7494-420f-4e59-bc6e-2acf4044d880`
**Date:** 2026-10-10 20:11 · **Branch:** master (72fd384 → e945bc9, plus this doc)

## Ask

The standing ask from part 1: do everything in the Next list that does not
need the user, commit and push per item, and wrap every two or three items.
This is the third wrap.

## 1. N-090: mirrored canvas text on the natives

### Setup

- **The demo.** A temporary canvas at the top of lesson 1.4, reverted and
  never committed. It is 200×60 with `CanvasMirrorsRTL` and holds:
  - ticks at x 20, 100 and 180;
  - a 30×6 bar at the start corner;
  - three 14pt texts: "start" (Align start, on 20), "mid" (middle, on 100)
    and "end" (end, on 180).
- **The web reference.** A scratch module exported the same canvas with
  htmlout. Headless Chrome screenshotted it as written and with
  `dir="rtl"`.
- **Android.** The API 36 emulator, deep-linked to 1.4. RTL came from
  `cmd locale set-app-locales com.grmob.app --locales ar`, reset after.
- **iOS.** A scratch XCUITest on the iOS 26.5 simulator. RTL came from the
  launch arguments in [[ios-simulator-rtl-and-xcuitest]]:
  `-AppleLanguages (ar)`, `-AppleTextDirection YES` and
  `-NSForceRightToLeftWritingDirection YES`.

### Result

All three targets agree, and they agree with core's contract for
`CanvasText` in a mirrored canvas:
- The bar moves to the top-right.
- "start" ends at its tick: Start is the text's right-hand end.
- "mid" stays centred.
- "end" begins at its tick.
- The glyphs are never mirrored.

Closed.

## 2. N-072's iOS keyboard bullet, which found N-102

### Getting a soft keyboard

- The simulator's default is a connected hardware keyboard. With it, the soft
  keyboard element sits off screen (y 952 of 874), which is why this bullet
  was unjudged on 2026-09-28.
- `defaults write com.apple.iphonesimulator ConnectHardwareKeyboard -bool
  false` takes effect only when Simulator.app relaunches.
- I sent TERM to Simulator.app alone. Quitting it **shut down the iPhone 17
  Pro**, and reopening it booted a different device (iPhone 17). I booted
  the 17 Pro again and shut the 17 down. I did the same dance after
  restoring the preference (`defaults delete`).
- End state: the iPhone 17 Pro booted, the hardware keyboard on, as at the
  start.
- The keyboard's first-run "slide to type" sheet appeared once and was
  dismissed with Continue.

### The defect (N-102)

- **Setup.** Lesson 4.37, with row 4 brought to just inside the bottom edge
  by small drags (a slow swipe overshoots to mid-screen). A tap on its Item
  cell opened the editor.
- **What happened.** The soft keyboard's top was at y 583, and the editor
  was at y 756: covered whole and not hittable. The page did not move.
- **Cause.** core.KeyboardAware's doc counts on SwiftUI's ScrollView to
  scroll the focused field into view. SwiftUI does that only for its own
  TextField. GrMob's field is a UIKit view in a representable (for the
  reasons GrMobTextField gives), so nothing scrolled for it. Compose's
  BasicTextField does bring itself into view; the 2026-09-28 Compose check
  found the editor above Gboard.

### The fix

`GrMobKeyboardReveal` lives in GrMobTextInput.swift, UIKit only.

- It remembers the focused GrMob field (the coordinator's `began` and
  `ended`) and the keyboard's frame (did-show and did-change-frame; did-hide
  clears it).
- **"Did" rather than "will".** By then SwiftUI has applied the keyboard's
  safe-area inset to the page.
- A field taking focus with the keyboard already up reveals on the next turn
  of the main loop.
- **The reveal.** It walks the field's ancestors. Each `UIScrollView` with
  more content than height is scrolled the least amount that puts the field,
  plus a 16pt margin, inside the part of it not covered by its
  `adjustedContentInset` or the keyboard.
  - The offset is clamped to the content.
  - A horizontal strip (EditableGrid's) is skipped.
  - A field already in view moves nothing.

### Checks

- **After the fix:** the editor at y 502 above the keyboard, text, caret and
  ✕ whole.
- **New test:** `TutorialRoundFourUITests.testTheLastGridRowsEditorIsAboveTheKeyboard`.
  - With the soft keyboard on, it passes.
  - With the hardware keyboard connected (the default), it skips: "the soft
    keyboard is not on screen…". It does not pass for a reason it did not
    check.
- **Soft keyboard on:** all six TutorialRoundFourUITests and the 2.3 caret
  test pass.
- **Hardware keyboard restored:** the grid round trip passes and the new test
  skips.
- **Source pins:** `mobile/verify/keyboardreveal_test.go`.
- **Docs:** KeyboardAware's doc now names the iOS reveal.
- **Not wired:** GrMobCodeEditor and GrMobRichTextEditor. They are
  UITextViews with no begin-editing hook. A global
  `textDidBeginEditing` observer would also catch SwiftUI's own internal
  fields.

Commit `e945bc9`. It also bumps wasm/verify's tracked-Go-file figure to 699.

## What went wrong

- **Quitting Simulator.app shut down the booted device and booted another.**
  That was not expected. The original state is restored, but a Simulator
  restart is not a free operation, and that belongs in memory.
- **wasm/verify's file count bit twice.** Each new Go file moves the figure
  that five sentences quote. `go test ./...` before each commit now catches
  it.

## Files touched

- `ios/GrMob/Runtime/GrMobTextInput.swift`
- `ios/GrMobUITests/TutorialRoundFourUITests.swift`
- `mobile/verify/keyboardreveal_test.go`
- `core/keyboard.go`, `docs/api/core-layout.md`
- `wasm/verify/{repowalks,timings}_test.go`
- `ai_docs/todo/next-list.md`, this doc

## Next

Closed: N-090, N-102. Declined: None. Raised: N-102. Deferred: None.
Promoted: None. Moved: None. Updated: N-072. Full list:
`ai_docs/todo/next-list.md`.
