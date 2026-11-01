# N-121: StockTicker and the glide on the natives, with three defects found

Session: `1d79c591-5129-4e15-8027-a2d38e7f6b2c`
**Date:** 2026-10-31 23:36 · **Branch:** master (3ca7640, plus this commit)

## Goal

The user asked for N-121, the Validate item for `comps.StockTicker` and
lesson 4.42 on the natives, which had grown the glide checks when N-120
closed. Everything on its list was run: the web through a scratch CDP probe,
Compose on the API 36 emulator (with the TalkBack logcat harness), SwiftUI on
the iOS 26.5 simulator through a new XCUITest class. Three defects came out of
it and were fixed in the same session; two follow-ups were raised.

## The web (headless Chrome, scratch probe on wasm/shots/host)

A scratch Node script copied shot.mjs's launch sequence, routed to 4.42 with
a `route` host event (`tap("A stock ticker")` finds nothing on the contents
screen), and read the DOM through the prelude's helpers.

- Stepping: the Scroll is `role=group` labelled "Stock prices", five
  `button` children labelled with the spoken sentence, `overflow-x: auto`.
- Gliding: two copies, `15s linear infinite grmob-glide`, the second
  `aria-hidden="true"` with no roles or labels but with click handlers.
  Both at −73 px/s (copy width 1101px / 15s), seam 0px, gap 24px within a
  copy and 24px across the seam. A real `Input.dispatchMouseEvent` click on
  the hidden copy's AAPL read "OnTap got Apple (AAPL)".
- RTL (`dir="rtl"` on `<html>`): `--grmob-inline` is −1 and the tape moves
  right at +72 px/s. One sample jittered where the copy's width changed at
  a tick.
- `prefers-reduced-motion: reduce` through `Emulation.setEmulatedMedia`:
  both copies still, copy 0 at the clip's x, copy 1 at copy 0's right; they
  move again when the feature is cleared.
- Pause: the glide copies go, the Scroll is back (scrollWidth 1081 of 296).
- **A step reuses no element.** With the interval paused, Step replaced all
  five quote elements. `reconcile/doc.go` says there are no move patches: a
  keyed slot whose key changed is replaced. The widget doc, lesson 4.42 and
  `docs/components.md` had claimed a step reorders the nodes on screen; all
  three now say a step never patches one quote's figures into another's slot
  and name the replace. Raised as N-129.

## Compose (API 36 emulator)

- Stepping: TSLA 499px wide, AMZN cut at the clip's edge; with Pause on, a
  swipe pans the tape (NVDA came to the left, TSLA appeared).
- **Gliding, the two copies were drawn over each other.** The band showed
  two layers of quotes 417px apart, and under a zero animator scale the
  tape stopped at the same off-rest spot every time (MSFT partly off the
  left). Cause, confirmed in ui 1.7.6's `Placeable.kt`: Compose coerces a
  Placeable's width into the constraints the child was measured with and
  centres the surplus (`recalculateWidthAndHeight`, `apparentToRealOffset`).
  So `pinMainAxis`'s unclamped report never reaches the Row: copy 0 (2835px
  in an 834px clip) was placed as 834 wide with its content centred, starting
  1000px left, and copy 1, offered 0, was placed at 834 and centred on
  nothing, 417px after copy 0. Fix: `overflowsAtContentWidth` in
  Renderer.kt, applied by GrMobRow when every child is pinned and none
  weighted: measure the Row with no width limit, report the offer, place the
  content with `placeRelative(0, 0)`. Pinned by
  `TestTheComposePinnedStripIsMeasuredUnboundedAndPlacedFromTheStart`
  (mobile/verify). The general case (a lone pinned child among unpinned
  siblings) is N-130, and `pinMainAxis`'s doc and
  `docs/platforms/native.md` now say what the transcription left out.
- After the fix: one layer; gap 63px (24dp) within and across the seam
  (uiautomator bounds follow the layer translation); both copies at
  189 px/s (2835 / 15); a tap predicted from AMZN's edge read "OnTap got
  Apple (AAPL)"; animator scale 0 holds the tape at rest with AAPL at
  clip + 63 (so `MotionDurationScale` does reach the node's coroutine
  context, the question N-120 left); scale 1 moves it again; per-app Arabic
  locale: +181 to +190 px/s rightward, quotes mirrored.
- TalkBack (TTS disabled, utterances off logcat): "Amazon, AMZN, 189.10, up
  1.65, 0.88 percent, Button", every quote in a Shift+Tab walk, and "Stock
  prices" for the tape. A forward Tab walk of the stepping tape visited two
  of five quotes before leaving the tape; not diagnosed. A Space on the
  Pause row under TalkBack did not toggle it.

## SwiftUI (iOS 26.5 simulator)

New `ios/GrMobUITests/TutorialStockTickerUITests.swift`, four tests, all
passing (`xcodegen generate` first; the project is untracked):

- `testTapeStepsEveryThreeSeconds`: the first quote changes within 4s;
  paused, a `swipeLeft` pans it.
- `testGlideMovesBothCopiesSeamlesslyAndTheHiddenCopyTakesATap`: the
  labelled copy moves left at one quotes-span per 15s (mean of samples,
  wrap samples over 1000 pt/s dropped); a tap 119pt past AMZN's end reads
  "OnTap got Apple (AAPL)"; Pause leaves it still.
- `testGlideFollowsReduceMotion`: reads `UIAccessibility.isReduceMotionEnabled`
  in the runner. Run with the simulator setting off (glides left) and on,
  via `simctl spawn … defaults write com.apple.Accessibility
  ReduceMotionEnabled -bool true` (still, AAPL 24pt inside the clip, measured
  against the caption's x since a container's XCUITest frame is the union of
  its quotes). The default was deleted after.
- `testGlideMovesRightUnderArabic`: forced RTL, +72 pt/s.
- **The stepping tape was one combined Button.** The tree showed `Button
  "Stock prices" > ScrollView > Other` with no quotes: a labelled Scroll was
  not in `GrMobNode.labelledContainerTypes`, so GrMobScroll applied its label
  with `.combine`, which merged the quotes' button trait up. Fix: `Scroll`
  joined the set, GrMobScroll reads `containerStyle`, and
  `ios/verify/childmode.swift`'s own type list gained it. The census then
  listed three labelled Scrolls: StockTicker (both forms, contain), Wizard's
  body ("Setup, step 1 of 2: Name", contain) and StepIndicator's strip
  ("Step 2 of 3: Pay", contain), all added to `childmode.go`; the latter two
  were merged elements before. The tape is now a ScrollView holding one
  Button per quote. VoiceOver's own speech is still unheard.

## Verification

- `gofmt -l .` empty; `go test ./...` green; `go vet` on the touched
  packages clean; `docs/api` regenerated.
- Kotlin: `compileDebugKotlin` exit 0 after both edits; `android/verify`
  OK; the APK reinstalled on the emulator for the re-check.
- Swift: `ios/verify/run.sh` exit 0 (23 widgets in the child-mode census);
  the UI tests built and ran the changed runtime.
- Devices left as found: animator scale 1, no app locale, TalkBack off and
  Google TTS enabled, no Reduce Motion default; emulator and simulator shut
  down (both were off at the start).
- Not done: a real finger on either native; VoiceOver's speech; the
  forward-Tab walk's two-of-five quotes.

## Lessons

- Compose's `Placeable.width` is coerced into the measurement constraints
  and the surplus centred; a layout modifier's "report what was measured"
  reaches the parent only under unbounded constraints. internal/pinfixture's
  transcription sums real widths and is right only there (N-130).
- uiautomator bounds follow a graphics-layer translation, so a glide can be
  measured off dumps as well as off screencaps; a 2-pixel-row band
  cross-correlation (scratch Go tool) read the shift between timed caps.
- XCUITest's frame for a labelled SwiftUI container is the union of its
  children (no padding); a wrapping pass reads as thousands of pt/s in one
  sample; assert on the mean. Saved to memory.

## Next

Closed: N-121. Declined: None. Raised: N-129, N-130.
Deferred: None. Promoted: None. Moved: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
