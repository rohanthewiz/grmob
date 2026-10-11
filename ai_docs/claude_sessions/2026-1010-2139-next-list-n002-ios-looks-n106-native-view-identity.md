# Next list, part 7: N-002's iOS looks, and N-106 (both natives rebuilt a node whose flag flipped)

Session: `1e5d7494-420f-4e59-bc6e-2acf4044d880`
**Date:** 2026-10-10 21:39 · **Branch:** master (43b8fd6 → 3f9e9c1, plus this doc)

## Ask

The standing ask from part 1: do everything in the Next list that does not
need the user, commit and push per item, and wrap every two or three items.
This is the seventh wrap.

## 1. N-002: the iOS looks

A scratch XCUITest (removed after) visited each look twice: once in LTR, and
once with the forced right-to-left launch arguments.

| look | result |
|---|---|
| sheet Dialog (6.6) | a medium-detent sheet with the card at its top and white below; GrMobModal's doc describes this placement |
| ActionSheet filler (6.7) | the same sheet, and the filler zero-height, as documented |
| Drawer under RTL (4.18) | the panel opens from the right; title right-aligned, ✕ at the left, icons on the right |
| CodeEditor under RTL (4.13) | code stays left to right with its gutter on the left; the toolbar mirrors |
| the sideways editor | a swipe pans the long lines and the gutter stays put |
| 4.6's list | the banded GroupedList draws its "March 2026" band and rows |
| Drawer under Reduce Motion | snaps, but it also snapped with motion on: N-106 below |

`core.Focus` on a Button is still open.

## 2. N-106: the Drawer never slid on either native

### Measuring a 250ms slide

- **iOS.** `xcrun simctl io … recordVideo` around a scratch test that opens
  4.18's Drawer, with Reduce Motion off and then on (the simulator's
  `com.apple.Accessibility ReduceMotionEnabled`, restored to 0). Two scratch
  Swift tools read the movie with AVFoundation: `edges` tracks a panel edge
  along one row per 60fps frame, and `frame` dumps a frame. There is no
  ffmpeg here.
  - First pass: the edge jumped in both modes. The frames confirmed it:
    shut at 18.300s, fully open at 18.317s.
- **Android.** `screenrecord` output would not open in AVFoundation, and a
  `screencap` takes 0.5–1s, too slow for 250ms. Setting
  `animator_duration_scale` to 10 (restored to 1) stretches the slide to
  2.5s.
  - First pass: fully open in the first frame after the tap.

### Causes and fixes

A container flag flips on every open and close, and each renderer expressed
that flag as a branch around the content. A branch is part of the content's
identity, so the subtree was built again instead of updated. Its animation
state restarted at the target, and the screen behind the drawer lost any
state it held, on every toggle.

| host | branch | flag |
|---|---|---|
| iOS | `grMobAccessibility`'s `if hidden … else if label …` | AccessibilityHidden, which both layers flip |
| iOS | `GrMobEscapeClaim`'s `if id.isEmpty { content } else { content.background … }` | the panel layer's OnEscape, open only |
| Android | RenderNode calling `RenderNodeContent` directly or inside `CompositionLocalProvider` | Inert, which both layers flip, plus the other locals |

- **iOS:**
  - hidden is now `accessibilityHidden(hidden)` around the label branch, and
    the hidden environment is only ever raised (`transformEnvironment`);
  - the Escape claim's condition sits inside the background.
- **Android:** one provider call site, always, with nothing provided when
  nothing changes.
- **Ruled out first.** The reconciler sends `update-style` and `update-props`
  for an open, with no replace (a scratch Go program printed the patches).
  Go keeps identity.

### Verified

- **iOS, each fix alone:** with the accessibility change only, or the Escape
  change only, the recording still jumped.
- **iOS, both:** seven intermediate edge positions over about 300ms
  opening (18.533–18.667s), and fewer closing. A mid-slide frame shows
  "ebook" with the title cut off at the left. Under Reduce Motion it is one
  jump.
- **Android:** at 10× the ✕ travels right and the panel's edge advances
  across six frames, while the scrim darkens.
- **Suites:**
  - all 13 Tutorial UI test classes, 55 tests, pass on the simulator
    (driven through `sh`, since zsh does not split a flags variable);
  - ios/verify passes, with its Release WMO guard;
  - android/verify passes;
  - `go test ./...` passes.
- **Pins:** `mobile/verify/viewidentity_test.go` holds all three shapes, and
  `keyshortcut_test.go` follows the new environment spelling.
- Commit `3f9e9c1`.

## Housekeeping

- **N-105 was never written.** It was earmarked in part 6 for the
  TypingIndicator "appearance" defect, which turned out to be TalkBack's
  repeat suppression. The next ID went from N-105 to N-107 when N-106 was
  raised, so N-105 stays unused. IDs are never reused, and nothing refers to
  it.
- **The emulator's storage filled** with my raw `screencap` burst (about
  140MB in /sdcard/cap). Deleted; `installDebug` then worked.

## What went wrong

- **The first iOS fix was only half.** Without the edge measurement after
  each change, I would have committed the accessibility change alone as the
  fix.
- **Two tool limits found the hard way:** zsh's unsplit `$ARGS` made the
  first full UI run fail at once, and the emulator's `screenrecord` file
  could not be read.

## Files touched

- `ios/GrMob/Runtime/GrMobStyle.swift`, `ios/GrMob/Runtime/Renderer.swift`
- `android/app/src/main/java/com/grmob/runtime/Renderer.kt`
- `mobile/verify/viewidentity_test.go` (new), `mobile/verify/keyshortcut_test.go`
- `wasm/verify/{repowalks,timings}_test.go` (700)
- `ai_docs/todo/next-list.md`, this doc

## Next

Closed: N-106. Declined: None. Raised: N-106. Deferred: None. Promoted: None.
Moved: None. Updated: N-002. Full list: `ai_docs/todo/next-list.md`.
