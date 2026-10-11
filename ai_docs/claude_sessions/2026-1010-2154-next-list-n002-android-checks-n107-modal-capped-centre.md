# Next list, part 8: N-002's Android checks, and N-107 (a capped Modal child left at the start)

Session: `1e5d7494-420f-4e59-bc6e-2acf4044d880`
**Date:** 2026-10-10 21:54 · **Branch:** master (6635023 → 96f6154, plus this doc)

## Ask

The standing ask from part 1: do everything in the Next list that does not
need the user, commit and push per item, and wrap every two or three items.
This is the eighth wrap.

## 1. N-002, Android: predictive back

The emulator uses gesture navigation (`navigation_mode` 2). An
`adb shell input swipe 2 1300 420 1300 3000` is a slow edge swipe, with a
screencap taken 1.4s in.

| from | mid-swipe | after |
|---|---|---|
| Contents | the window shrinks and slides, showing what is behind (the previously used app, com.gonotes.mobile, rather than home) | the app is left |
| a lesson (1.4) | only the back arrow, no exit preview | Contents, in the app |
| 4.18 with the drawer open | only the back arrow | the drawer closed, still in the lesson |

## 2. N-002, Android: MaxWidth where it binds, which found N-107

- **Landscape instead of a tablet.** `user_rotation 1`, which needed
  `accelerometer_rotation 0`. Auto-rotate read 1 at this point, although part
  4 had set it to 0, so it was set back to 1 at the end.
- **6.7's ActionSheet:** the card was 1365px, which is 520dp, so the cap held.
  But it sat at the bottom-left, where the web centres it.
- **The cause.** ColumnChildren's `centreCapped` centres with
  `wrapContentWidth`. That centres only inside a minimum width the parent
  forced, and the minimum comes from `fillMaxWidth`, which `hugsContent`
  withholds from a child with a Width. The ActionSheet card is
  `Width("100%")` under the caller's `MaxWidth("520px")`. The DatePicker's
  card has no Width, which is why the comment's own example worked.
- **The fix.** For a capped child, `align(Alignment.CenterHorizontally)` as
  well, unless it states its own AlignSelf (N-100).
- **After:** the ActionSheet and 4.17's Menu panel both span 517–1882 of
  2400px, centred. 4.9's DatePicker card (360) stays centred and scrolls.
- **Pin:** `TestComposeModalCentresACappedChildWithAWidth` (in
  alignself_test.go, so no new Go file).
- android/verify passes.
- Commit `96f6154`.

## 3. N-002, Android: an RTL capped child

5.7's PIN row (`MaxWidth("320px")`), in landscape:
- LTR: [123..963], at the start;
- per-app Arabic: [1437..2277], at the start, which is the right.

## Restored

- Rotation is back to 0 and auto-rotate to 1, the value found.
- The per-app locale is reset (`[]`).

## What went wrong

- **My first landscape swipes went off screen.** Auto-rotate was on, so
  `user_rotation` had no effect, and the swipes used landscape coordinates on
  a portrait screen. Reading `dumpsys window displays` first would have shown
  it.

## Files touched

- `android/app/src/main/java/com/grmob/runtime/Renderer.kt`
- `mobile/verify/alignself_test.go`
- `ai_docs/todo/next-list.md`, this doc

## Next

Closed: N-107. Declined: None. Raised: N-107. Deferred: None. Promoted: None.
Moved: None. Updated: N-002. Full list: `ai_docs/todo/next-list.md`.
