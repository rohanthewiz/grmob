# Android: images get a User-Agent, a Scroll in a Row scrolls; v0.7.1

Session: `ada40c4b-9595-4b2b-bcbc-e2e6ecae2cf7`
**Date:** 2026-10-19 23:23 · **Branch:** master (be15af0 → 6e820c8, plus this doc) · **Tag:** `v0.7.1` on the commit that adds this doc

## Goal

This continues `2026-1019-2246-gallery-tapzone-zindex`. The catadopt app
(`~/projs/go/grmob_x2/catadopt`, a `comps.Gallery` of adoptable cats) was
switched to `core.AmberTheme`, then built with `grmob android` and installed
on the user's Galaxy Z Fold6 (SM-F956U1). It was viewed on the cover screen,
968×2376. Two Android-only defects showed up, and the user asked for both
to be fixed upstream, then a wrap, v0.7.1, and a catadopt update.

## N-116: Commons photos blank on Android

- **Symptom:** every Gallery picture and thumbnail was an empty box.
  Nothing in logcat, and INTERNET was declared.
- **Cause:** Coil 2 fetches through OkHttp, whose default User-Agent is
  `okhttp/4.12.0`. Tested with curl against an upload.wikimedia.org thumb:
  - `okhttp/4.12.0` → 403 `Please set a user-agent and respect our robot
    policy https://w.wiki/4wJS`;
  - a CFNetwork-shaped agent (what iOS's URLSession sends) → 200;
  - a bare application ID → 200.

  The browser sends its own agent. So only Android was affected.
  `GrMobMapView.createOsmMap` already sets osmdroid's `userAgentValue` to the
  package name for OSM's tile servers; Coil never got the equivalent.
- **Fix (c2e98ad):**
  - `android/app/src/main/java/com/grmob/app/GrMobApplication.kt` is new.
    It is the shell's `Application` and Coil's `ImageLoaderFactory`. Its
    OkHttp client has an application interceptor (which runs before
    BridgeInterceptor fills in the default) that sets
    `<label>/<versionName> (<applicationId>; Android <release>) grmob`, only
    when the request carries no User-Agent.
  - The manifest names it with `android:name`.
  - `mobile/verify/image_agent_test.go` pins the manifest, the factory and
    the interceptor (through `valuesIn`/`valuesOf`; `TestEveryNativeRead
    NamesItsQuestion` closes the raw readers).
  - The new test file moved wasm/verify's tracked-Go-file figures from 708
    to 709 in five sentences, as `TestTheQuestionsOnTheSharedRepository
    ParseAreTheOnesDecidedOn` requires.
- catadopt had carried the same file locally, from before this fix, so its
  next `-refresh` loses nothing.

## N-117: the thumbnail strip drawn over the arrows on Compose

- **Symptom:** Gallery's nav row is
  `Row[‹, Row(FlexGrow 1, basis 0, MinWidth 0)[Scroll(Horizontal, MaxWidth
  100%)[Row of thumbs]], ›]`. On the Fold6 the strip spanned the whole
  screen, centred and unclipped, with the first thumbnails off the leading
  edge and the arrows covered. On the web it sits between them.
- **Cause:** `RowChildren` gives an unweighted child `rowChildWidth`, which
  floors it at `minIntrinsicWidth`, CSS's `min-width: auto`.
  - CSS computes that to 0 for a scroll container (overflow not visible).
  - Compose's `horizontalScroll` forwards intrinsic queries to its content,
    so the strip's min-content is every thumbnail laid end to end.
  - `rowChildWidth` reports its measured width unclamped, by design, so an
    overflowing row overflows. So the strip was reported past the grower's
    share and drawn over its siblings.
- **Fix (6e820c8):**
  - `rowChildWidth` takes `scrollContainer: Boolean = false`. When set,
    `minContent` is 0 and the intrinsic query is skipped. `RowChildren`
    passes `scrollContainer = child.type == "Scroll"`.
  - A plain strip then measures at `min(offer, content)` (`hugsRowOffer`) and
    scrolls. A declared MinWidth still floors.
  - New doc section: "A scroll container's minimum is zero".
  - `TestComposeFloorsARowChildAtItsMinContentWidth`'s pin follows the new
    spelling, and the new `TestComposeDoesNotFloorAScrollContainerInARow`
    pins the wiring. Both fail on be15af0's Renderer.kt.
- **Not changed:** an `Overflow("hidden")` Box also gets CSS's zero minimum,
  but no case needed it.

## Verification

- `go test ./...`, gofmt and vet are clean. `sh android/verify/run.sh` is
  all OK: `com.grmob.runtime` and `com.grmob.app` compile against the
  gradle classpath, and lint shows 0 errors (23 warnings).
- **Device:** catadopt was built with a temporary `replace` to this checkout,
  then `grmob android -refresh -install`.
  - Photos loaded.
  - The strip sat between ‹ and ›, starting at the ringed current thumbnail.
  - A swipe scrolled it.
  - A middle tap flipped the card to its Details.

  catadopt was restored to its commit afterwards.
- The Gallery parts of N-111 seen on Compose are recorded in that item.

## Release v0.7.1

- **API diff** against v0.7.0 (apidiff, with v0.7.0 in a temporary worktree
  that was removed afterwards): no changes.
- **Version:** a patch, two native fixes.
- **CI** is still red for the known reasons only: N-110 (the
  `TestEditableGrid*` race under -race) and N-109 (check 24 on lessons 2.2,
  4.29, 5.3, 6.5, 7.2 and 7.4). The tag lists both as Known issues.

## Next

Closed: N-116, N-117. Declined: None. Raised: N-116, N-117.
Deferred: None. Promoted: None. Moved: None. Updated: N-111 (Compose
results: the thumbnail strip, the middle-zone tap and the fade flip seen;
the edge zones, long press and SwiftUI still open). Full list:
`ai_docs/todo/next-list.md`.
