# iOS: a filled Image keeps its box; Screen Fill grows its Scroll; v0.7.2

Session: `ada40c4b-9595-4b2b-bcbc-e2e6ecae2cf7`
**Date:** 2026-10-19 23:43 · **Branch:** master (955bf89 → e5b0e80, plus this doc) · **Tag:** `v0.7.2` on the commit that adds this doc

## Goal

This continues `2026-1019-2323-android-image-agent-and-scroll-strip-floor`.
The user asked for catadopt (`~/projs/go/grmob_x2/catadopt`, a
`comps.Gallery` of adoptable cats under `core.AmberTheme`) to be built for
iOS and run in the simulator. `grmob ios -run` vendored the v0.7.1 shell,
bound the app, and launched it on the booted iPhone 17 Pro (iOS 26.5). It
rendered with the layout broken, and the user asked for an upstream fix,
then a wrap, v0.7.2, and a catadopt update.

## What the simulator showed

- The whole screen was laid out much wider than the 402pt window and
  centred. The AppBar title was cut on the left, the description was one
  line running off both edges, the ‹ › arrows and the Info button were off
  screen, and the thumbnail strip spilled past them.
- White showed below the content.

## Bisecting

Each experiment was a catadopt rebuild with `grmob ios -run`, reverted
afterwards.

1. **Thumbnails → dots:** still too wide, so not the strip. (iOS's
   `GrMobMinContent` already gives a Scroll a minimum of 0, unlike Compose
   before N-117.)
2. **`Screen.Fill` off:** still too wide, so not Fill.
3. **Reading the renderer:**
   - `GrMobStackLayout` passes its proposal through.
   - Each `Width("100%")` is `frame(maxWidth: .infinity)`.
   - `GrMobImage` wraps the scaled bitmap in the same frame on each stated
     axis, and `grMobScaled`'s fill arm is `resizable().scaledToFill()
     .clipped()`.

   A flexible frame with no minimum reports `max(proposal, child)`, and a
   filled image reports its filled size. Gallery keeps every card in the
   tree, and Pepper's 500×306 photo filled to the 424pt stage is about
   693pt wide. That matches the roughly double-width screen.

## N-118: a filled Image widened its box (379ddca)

- `GrMobImage`'s frame is now `frame(minWidth: stated ? 0 : nil, maxWidth:
  stated ? .infinity : nil, …)` on both axes, with `.clipped()` after it.
  - The clip moved because `grMobScaled`'s clip is to the image's own,
    filled bounds, not to the box.
  - With a nil proposal (a flex basis query) the frame still reports its
    child, so measurement is unchanged.
  - An unstated axis keeps no frame at all.
- New doc section "A filled image is the size of its box", with the 370 vs
  693 arithmetic.
- No mobile/verify pin was added. There is no textual contract that would
  distinguish this from the old frame better than a device does. The
  simulator is the evidence.

## N-119: Screen{Scroll, Fill} left white below a short page (e5b0e80)

- `Screen{Scroll, Fill}` is `SafeArea > Scroll > Column(FlexGrow 1)`. The
  Scroll got FlexGrow only with a Footer.
  - The browser makes every Scroll full height through the host page CSS.
  - Compose's vertical Scroll takes Fill's minimum from the height it is
    offered.
  - SwiftUI's vertical body reads its viewport from its own GeometryReader,
    and the flex column measured the Scroll at its content height, so the
    minimum was circular.
- Fixed in `comps/screen.go`: `core.MaybeProp(s.Footer != nil || s.Fill,
  core.FlexGrow(1))` on the Scroll.
  - Screens without Fill or a Footer render the same tree as before.
  - Fill's doc comment no longer says it "does not make a scroll view fill
    anything", and `docs/api/comps-structure.md` was regenerated
    (`TestGeneratedPagesAreUpToDate` asked for it).
  - `TestScreenFillWithScrollGrowsTheScrollRegion` fails without the change.

## Verification

- **iOS 26.5 simulator**, catadopt built with a temporary `replace` to this
  checkout and `grmob ios -refresh -run`:
  - the screen fits;
  - the description wraps to three lines;
  - the photo is cropped to the card;
  - the strip sits between the arrows;
  - the AmberTheme Surface reaches the bottom.
- **Android:** the phone had been unplugged, so the regression check for the
  Screen change ran on the `Medium_Phone_API_36.1` emulator, which was
  stopped afterwards. It renders as before, with the background full height
  and the strip between the arrows.
- `go test ./...` passes.
- `sh ios/verify/run.sh` is all OK: the view layer type-checks and
  survives whole-module optimisation, and the app layer type-checks against
  the iOS SDK.
- catadopt was restored to its commit afterwards.

## Release v0.7.2

- **API diff** against v0.7.1: no changes.
- **Version:** a patch.
- **CI** is still red for the known reasons only: N-110 (the
  `TestEditableGrid*` race under -race) and N-109 (check 24 on lessons 2.2,
  4.29, 5.3, 6.5, 7.2 and 7.4).

## Next

Closed: N-118, N-119. Declined: None. Raised: N-118, N-119.
Deferred: None. Promoted: None. Moved: None. Updated: N-111 (SwiftUI front
face, caption and strip seen at the screen's width; nothing tapped there).
Full list: `ai_docs/todo/next-list.md`.
