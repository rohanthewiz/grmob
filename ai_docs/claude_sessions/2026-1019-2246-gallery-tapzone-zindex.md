# Gallery tap zones under the cards on the web; v0.7.0

Session: `ada40c4b-9595-4b2b-bcbc-e2e6ecae2cf7`
**Date:** 2026-10-19 22:46 · **Branch:** master (1bb8b57 → dcf9274, plus this doc) · **Tag:** `v0.7.0` on the commit that adds this doc

## Goal

The session started in a separate app workspace (`~/projs/go/grmob_x2/catadopt`),
building a cat-adoption app with `grmob new` and `comps.Gallery` in dark
mode. In Chrome, clicking the photo did nothing. The user then asked for:

1. the bug fixed upstream, here;
2. a session wrap and a release;
3. catadopt updated to the release.

## The bug (N-115)

`comps.Gallery` puts three invisible tap zones over the stage. The leading
quarter steps back, the middle flips (a long press zooms), and the trailing
quarter steps forward. They are a hidden `Row`, the stage `ZStack`'s last
child. On the natives that is enough, because they paint by child order.

On the web every card layer also carries `core.ZIndex(zOrder(d))`, which
ranges from 1 to n+2. That was added so the deck's fanned cards stop
painting over the current one (see `zOrder`'s doc). The tap-zone Row had no
z-index, and in CSS z-index beats document order, so every card painted
over the zones. Hit testing follows paint order, so the picture took every
click. The arrows, dots, thumbnails and Info button still worked, which is
why lesson 4.41's headless shots never showed it: none of them tapped the
photo.

How it was found in the browser:

- `document.elementFromPoint` at the middle zone's centre returned the
  current card's `<img>`.
- The stage's children computed as `z-index` 4…11 for the cards and `auto`
  for the Row.

## Fix

- `dcf9274`, `comps/gallery.go`: a new `galleryRender.tapZOrder()` returns
  `r.n + 3`, one above `zOrder`'s maximum. `tapZones()` now adds
  `core.ZIndex(r.tapZOrder())` to the Row. The doc comment says why child
  order is not enough on the web.
- `comps/gallery_test.go`: the new `TestGalleryTapZonesPaintAboveTheCards`
  checks that the Row's ZIndex is above every card's. It covers all four
  transitions, with the first card current and with the last, because the
  deck lifts the cards already left behind (d < 0) highest. It fails on
  1bb8b57 and passes now.
- `go test ./...`, `go vet ./comps` and `gofmt -l comps` are clean locally.
- Browser check: catadopt was built against this checkout through a
  temporary `replace`, and its CSS workaround was deleted in the tab. The
  Row then computed `z-index: 12` (9 cards + 3), the middle zone was the hit
  target, and a click flipped the card to its Details. The `replace` was
  reverted afterwards.

## Side finding: Dark Reader in the user's Chrome

The user's Chrome runs the Dark Reader extension in filter mode on
localhost, and it ignored `<meta name="darkreader-lock">`. It inverts an
already-dark page to cream. Its reverse filter on `<img>` also makes each
photo paint above its unpositioned siblings, which hid Gallery's caption
band. This is not a grmob bug. When judging a local screenshot, check
`data-darkreader-mode` on `<html>` first.

## Release v0.7.0

- **API diff** against v0.6.0 (`apidiff -m -w` on each side, with v0.6.0 in
  a temporary worktree that was removed afterwards): 32 declarations added
  and no incompatible changes.
  - 30 are in comps: `Gallery`, `GalleryItem`, `GalleryState`,
    `GalleryLabels`, `GalleryMotion`, the Look, Transition, Flip, Zoom and
    Indicator enums with their values, and `ConcernGalleryFrozen`.
  - 2 are in highlight: `Languages` and `LineComment`.
- **Version:** a minor bump, because API was added and none removed.
- **CI:** both workflows are still red on master, for the known reasons
  only.
  - CI: `TestEditableGrid*` under `-race`, the harness race (N-110).
  - site: check 24's overflow on lessons 2.2, 4.29, 5.3, 6.5, 7.2 and 7.4
    (N-109).
  - The tag lists both under Known issues, as v0.6.0 did.

## Next

Closed: N-115. Declined: None. Raised: N-115. Deferred: None.
Promoted: None. Moved: None. Updated: N-111 (the tap-zone check now notes
that the web case was N-115 and is fixed, and that the natives are still
untapped). Full list: `ai_docs/todo/next-list.md`.
