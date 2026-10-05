# N-022: a zero flex-basis on iOS without a definite main extent

Session: `e1a42c56-7bea-4f71-83eb-65b1602fa975`
**Date:** 2026-10-04 20:22 · **Branch:** master (c0a8e68 → one commit with this doc)

## Ask

N-022 from the next-list, pasted in from the cats-todo backlog: "A zero basis
is honoured on iOS only with a definite main extent." The 2026-09-29
recommendation had named the fix, a two-pass measure around
`GrMobFlexZeroBasis`, and said no bundled screen needed it.

## What the gap actually was

Reading `GrMobFlexLayout` (`ios/GrMob/Runtime/Renderer.swift`) showed the item
was narrower than its title:

- `placeSubviews` always passes `definite: true`, so at placement a Row's
  zero-basis children already started from their padding.
- The guard in `baseMains` was `if definite, padding >= 0, automatic.isFinite`.
  A **Column** child whose `GrMobFlexMin` is the `.infinity` verdict ("floor
  at your content height") failed `automatic.isFinite`. So it kept its
  measured content height as its base even inside a Column of definite height,
  and a weighted Column of such children was content-biased.
- Under an ideal-size query (`sizeThatFits` with no main offer), the bases
  were measured content. The container's length is right that way (CSS sizes
  a container from content contributions). But the cross size was then
  measured at content mains, while `placeSubviews` draws at zero-basis mains.

## What landed

### Definite extent: the `.infinity` verdict is measured

- `baseMains`: the guard is now `if definite, padding >= 0 {`. Every zero-basis
  child starts at its padding when the extent is definite.
- The measurement moved into a helper, `contentMain(_:crossBound:floor:)`.
- `minMains` gained a `crossBound` parameter. For a zero-basis child it passes
  `automatic` when finite. Otherwise (the Column verdict) it measures the
  content height with `contentMain` and passes that as the minimum. Non-zero-
  basis children keep `max(min(automatic, bases[i]), floors[i])`.
- Both call sites pass their cross bound: `crossBound` in `sizeThatFits`,
  `containerCross` in `placeSubviews`.
- The measurement runs only for that one combination (zero basis plus Column
  verdict). Every other child costs nothing new.

### Ideal-size query: a second pass

`sizeThatFits` now keeps two arrays:

```
  pass 1   measured = content bases   -> main = containerMain(measured)
  pass 2   bases = zeroBased(measured) (padding for zero-basis entries)
           resolved = resolve(main, bases, mins)
           cross measured at resolved mains
```

With a definite offer, `bases == measured` and there is one pass. `zeroBased`
only swaps entries, so nothing is measured twice. No cap is re-applied, since
a padding is never above the content it replaces.

### Docs and pins

- The doc comments on `GrMobFlexZeroBasis`, `baseMains` and `minMains` now
  describe the two passes. `minMains` has a worked table (Column 300, contents
  40/100: before 120/180, now 150/150).
- `TestIOSFlexHonoursAZeroBasis` (`mobile/verify/shrink_test.go`) pins the new
  guard, the `minMains` non-zero-basis line, the `contentMain` measurement and
  the `zeroBased` second pass. It fails if the old
  `automatic.isFinite` guard comes back.

## Seen

On the iPhone 17 Pro simulator (iOS 26.5), with a throwaway
`examples/zzbasis` app (deleted afterwards) built through
`ios/build.sh ./examples/zzbasis`:

| Case | Before (HEAD renderer) | After |
|---|---|---|
| Column `Height("300px")`, two `FlexGrow(1) FlexBasis("0")` cells, 1 and 4 lines | red 247px / blue 384px | 315px / 316px |
| Inner Row sized by content, cells "hi" and a long label | equal cells, label wraps to 3 lines, green box encloses it | same |

The second case looked the same both ways. The outer Row measures the inner
one again at a definite width, so it never reaches the ideal-size path that
pass 2 changes. No screen was found that shows pass 2. It was kept because it
makes `sizeThatFits` agree with placement and costs no extra measurement.

The "before" screenshot was taken by putting HEAD's Renderer.swift back
temporarily, rebuilding only the app, and restoring the fixed file (checked
with `cmp`).

## Checks

- `go test ./mobile/verify/`: ok.
- `xcrun --sdk iphonesimulator swiftc -typecheck -target
  arm64-apple-ios17.0-simulator ios/GrMob/Runtime/*.swift`: clean.
- `xcodebuild test -only-testing:GrMobUITests/TutorialZeroBasisAndGradientsUITests
  -only-testing:GrMobUITests/TutorialChartsUITests`: all 4 tests passed
  (calendar day columns equal, StatTiles share the row, gradient rows, charts).
- The tutorial framework was rebuilt and the app reinstalled after the
  throwaway app was removed, so the simulator holds the tutorial again.

## Found, not fixed

- `ios/verify/run.sh`'s macOS type-check
  (`swiftc -typecheck -target arm64-apple-macos14.0 ../GrMob/Runtime/*.swift`)
  fails with `no such module 'UIKit'` in `GrMobSurface.swift` (from commit
  6fda8ec). It was already failing before this session and is tracked as N-089. Type-checking against
  the iOS simulator SDK works.

## Notes

- Another session's uncommitted edit to `ai_docs/todo/next-list.md` (N-031
  moved to Non-goals) was in the tree. It was left as it was, and only this
  session's N-022 hunk is committed.
- Not checked on Android or the web, which this item did not touch.

## Next

Closed: N-022. Declined: None. Raised: None.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
