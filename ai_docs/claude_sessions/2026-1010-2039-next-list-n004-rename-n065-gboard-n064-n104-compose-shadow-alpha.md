# Next list, part 5: N-004 test rename, N-065 Gboard composing, N-064 opacity checks and N-104

Session: `1e5d7494-420f-4e59-bc6e-2acf4044d880`
**Date:** 2026-10-10 20:39 · **Branch:** master (9dd9f38 → c864d21, plus this doc)

## Ask

The standing ask from part 1: do everything in the Next list that does not
need the user, commit and push per item, and wrap every two or three items.
This is the fifth wrap.

## 1. N-004: the Escape test's name

- `TutorialKeyShortcutsUITests.testEscapeClosesTheDrawerAndTheDialog` only
  ever opened 4.18's Drawer. It is now `testEscapeClosesTheDrawer`.
- Its doc says why the Dialog is not this test's check: an iOS Modal has no
  Escape claim of its own, so SwiftUI's own sheet dismissal under a real
  Escape is N-004's question for a real iPad keyboard.
- It still passes, with its `XCTExpectFailure` (XCUITest's Escape does not
  reach the claim).
- Commit `421ff1b`.

## 2. N-065: a composing IME on Android

- **Setup.** The emulator's `show_ime_with_hard_keyboard` was 0, set to 1 for
  the run, and restored after. Lesson 2.3: UPPERCASE switched on, the field
  tapped, Gboard up.
- **Driving Gboard.** Its key centres were measured from a screenshot on the
  1080×2400 screen; a scratchpad script (`keys.sh`) taps letters by
  coordinate. Unlike `adb shell input text`, which commits whole keys, this
  composes the way a person's typing does.
- **Results:**
  - "hello world" read `HELLO WORLD`, with Gboard composing (its strip
    offered "world" and "would").
  - "abc" typed with the caret at the start of "WORLD" read `HELLO
    ABCWORLD`, with the caret after the C.
  - " wor", then a tap on the "world" suggestion, committed `WORLD ` (with
    Gboard's auto-space). A following "x" read `X`.
- N-065's Android bullet records it. The Fold6 and Samsung's keyboard remain.

## 3. N-064: opacity checks, which found N-104

### The demo

Temporary, at the top of lesson 1.4, reverted and never committed. On a
`#DDE3EA` panel:
- two 120×60 white cards with `Shadow(16)`, at Opacity 1 and 0.5;
- an Opacity(0) Box with an OnClick counter and an AccessibilityLabel;
- the count beside it.

Pixels were read with a scratch PNG reader (no PIL here), 12px below each
card. The panel is 221, 227, 234.

### Compose (API 36 emulator)

- **The Opacity(0) box** is in the uiautomator tree and clickable, and a tap
  counted (1). That is core's documented behaviour for Compose.
- **The 0.5 card cast no shadow at all.** Below it the pixel was the panel
  exactly, where the opaque card darkened it to 198, 204, 210. Its corners
  were squared at the buffer edge. That is **N-104**.

### N-104: the cause and the fix

- **Cause.** `boxModifier` applied `Modifier.alpha` before `Modifier.shadow`.
  An alpha below 1 renders its layer offscreen into a node-sized buffer, and
  the shadow inside it was cut.
- **First try.** The alpha on the shadow's own graphics layer. The shadow
  came back but fades by alpha squared (Compose sets the layer's and the
  outline's alpha together). It darkened the panel by about 5 against the
  opaque card's 23.
- **Kept for API 28+.** `m.shadow(…, ambientColor = tint, spotColor =
  tint).alpha(layerAlpha)`, where
  `tint = DefaultShadowColor.copy(alpha = layerAlpha)`. The darkening is 11
  against 23 near the card and 5 against 9 further out: linear, as CSS is.
  Below API 28 (minSdk 24, no shadow colours) the single-layer form stays.
- **Interior of the 0.5 card** (white over the panel should be 238):

  | build | inside |
  |---|---|
  | original | 238 |
  | single layer | 232 |
  | tinted outer shadow | 228 |

  An elevation shadow is cast under the whole outline, so it shows through a
  translucent box; CSS clips a box-shadow to outside the box. This is the
  platform's model, and the comment records it. The tinted version is the
  closer of the two overall.
- **Pins.** `mobile/verify/opacity_test.go`: two pins follow the new alpha
  line, and `TestComposeShadowSurvivesAnOpacity` holds the guard, the tint,
  the order and the API<28 arm. `android/verify` passes.
- Commit `c864d21`.

### iOS (26.5 simulator)

- **The Opacity(0) box** is in XCUITest's tree, but the tap did not reach it
  (0 taps). That is SwiftUI's exception, as core's doc says. VoiceOver
  itself is unheard.
- **The 0.5 card** keeps its shadow, and its interior is exact (238, 241,
  245): SwiftUI composites the shadow with the content.

N-064 stays in Validate for VoiceOver at opacity 0, and for watching a
DisplayHidden fill vanish on Compose.

## Cleanup

- The probe was reverted and the scratch UI test removed.
- Both apps were rebuilt and reinstalled from `./examples/tutorial`.
- The emulator's IME setting is back to 0.

## What went wrong

- **The emulator still sleeps between looks.** Every pass starts with
  `KEYCODE_WAKEUP` and two cold starts of the deep link.
- **The count's AccessibilityLabel hid its text from XCUITest.** The iOS tap
  count had to be read off the screenshot.

## Files touched

- `ios/GrMobUITests/TutorialKeyShortcutsUITests.swift`
- `android/app/src/main/java/com/grmob/runtime/GrMobStyle.kt`
- `mobile/verify/opacity_test.go`
- `ai_docs/todo/next-list.md`, this doc

## Next

Closed: N-104. Declined: None. Raised: N-104. Deferred: None. Promoted: None.
Moved: None. Updated: N-004, N-064, N-065. Full list:
`ai_docs/todo/next-list.md`.
