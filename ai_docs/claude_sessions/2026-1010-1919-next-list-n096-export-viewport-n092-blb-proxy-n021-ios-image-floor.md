# Next list, part 1: N-096 export viewport, N-092 BLB proxy, N-021 iOS image floor

Session: `1e5d7494-420f-4e59-bc6e-2acf4044d880`
**Date:** 2026-10-10 19:19 · **Branch:** master (7aea91b → dfd2ee6, plus this doc)

## Ask

"Do all in the Next list that does not require my direct input. If there are
any minor decisions to be made use your best judgement. Commit and push
between each item and do a sess-wrap after every 2–3 items and at the end."

This is the first wrap of the session. Each item was committed and pushed on
its own; the next-list edits ride with this doc, per `/sess-save`.

### Triage

- **Attached:** the API 36 emulator (`emulator-5554`) and the iPhone 17 Pro
  simulator (iOS 26.5). No physical phone was connected. No other grmob
  session was running (ListAgents).
- **Doable without the user:** the code items N-096, N-092, N-021, N-099,
  N-043 and N-098, plus the checks that run on the emulator, the simulator or
  in Chrome.
- **Waits on the user:**
  - decisions: N-013, N-015, N-027, N-045, N-051;
  - hardware or a person: N-006, N-030, N-040, the Fold6 halves.
- **Contingent:** N-003 (the next hook change), N-047 (a Compose BOM past
  foundation 1.10), N-009 (AlarmKit or full-screen intents).

## 1. N-096: every export has a viewport meta

- `ExportHTML` wrote a `<head>` only when the tree needed a motion or
  border-box rule. So no export had a viewport meta, and a phone laid it out
  at the 980px desktop fallback.
- `documentHead` now always writes the head, in this order:
  - `<meta charset="utf-8">`;
  - a viewport of `width=device-width, initial-scale=1`;
  - motionStylesheet's `<style>`s, which stay conditional.
- `motionStylesheet` now writes only the rules, into the head
  `documentHead` opens.
- **Decision:** `viewport-fit=cover` is left out, although the scaffold has it.
  Cover lets a page run under the notch, which only a page that pads itself by
  `env(safe-area-inset-*)` can afford. wasm/index.html does; an export does
  not.
- **Decision:** charset added too. The head is now unconditional, and exports
  carry non-ASCII (‹, ▸, emoji labels).
- **Tests:**
  - The three "no head" tests now assert "no `<style>`", which was their
    point.
  - `TestEveryExportCarriesCharsetAndViewport` pins the order.
  - `TestExportViewportStaysInsideTheSafeArea` keeps viewport-fit out.
  - `docs/api/htmlout.md` regenerated (line numbers).
- **Measured:** a scratch CDP probe on `examples/layout`'s export at 420px
  under mobile emulation. Before: innerWidth 980, `visualViewport.scale`
  0.43. After: 420 at scale 1.
- Commit `2c50935`.

## 2. N-092: `blb.Proxy`

- **New in `blb/proxy.go`:**
  - `EndpointPath`;
  - `Proxy`, an `http.Handler`;
  - `Proxy.Forward(ctx, rawQuery) (ProxyReply, error)`, the same without
    net/http, for rweb.
- **`blb.go`:** `Fetch` uses `EndpointPath` and a shared `userAgent`. The
  package doc's "Where it works" names the proxy.
- **Narrow on purpose**, since it fronts a third party:
  - The upstream and the path are fixed. The request's own path is ignored.
  - GET and HEAD only. 405 otherwise, with `Allow`.
  - Only `id`, `style` and `target` are forwarded, each at most 256 bytes.
    No id means a 400 before any upstream call.
  - None of the caller's headers go upstream (no Cookie, Authorization or
    X-Forwarded-For). The upstream sees Fetch's User-Agent.
  - Only the status, the Content-Type and a body of at most 1 MiB come back.
    A longer reply is a 502, not a cut passage.
- rweb's `Server.Proxy` was not used: it forwards every header, method and
  parameter.
- **`serve`:** `mountBLBProxy` is mounted in both modes. The fixed route
  beats `/*path` whatever the registration order. A refusal is logged once.
- **Docs:** lesson 4.39's prose, its `tutorialPassages` comment and
  `docs/components.md` now say to mount `blb.Proxy` and set `BaseURL` to the
  page's origin. The tutorial stays canned, since GitHub Pages is static.
- **Tests:**
  - `blb/proxy_test.go`: a round trip from a Client, the narrowing,
    path-ignoring, refusals before the upstream, HEAD, oversize (exactly
    1 MiB passes, one byte more is refused), and a pass-through 503 and a
    502 when the upstream is down.
  - `serve/main_test.go`: `TestBLBProxyRoute`.
- **Live:**
  - The built `serve`, with curl: 200 and "John 3:16 (KJV)". With no id, a
    400. `/` still served.
  - A scratch wasm module with `BaseURL = location.origin`, served by `serve`
    and loaded in headless Chrome: "OK Psalms 23:1-2 | 2 verses |
    https://www.blueletterbible.org/kjv/psa/23/1/ | A Psalm of David…".
- Commit `2be6c39`.

## 3. N-021: an iOS Image floors at its natural size

### Cause

`GrMobImage` was an `AsyncImage`, which hands its content a SwiftUI `Image`
and no size. So `GrMobMinContent` floored every Image with a src at its
declared width. A browser floors an `<img>` flex item at min(declared width,
natural width carried through a declared height).

### Fix

- **The loader.** `GrMobImage` loads with `URLSession.shared` and decodes a
  `UIImage` itself (scale 1, EXIF upright). It records the size in
  `GrMobImageSizes`.
  - `GrMobImageSizes` is @Observable and keyed by src.
  - Kept from AsyncImage: the spinner (also after a failure or with no src),
    URLCache, an unlabeled image labelled from outside, and the cancel on a
    src change.
- **The floor.** `GrMobMinContent.imageFloor(declared:height:natural:)`
  takes:
  - the natural width, or
  - with a points Height, height × natural aspect;
  - capped at the declared width.
  - Unknown or zero-sided sizes keep the declared width.
  - Observation re-runs the Row body that read the store when a bitmap
    lands, so the floor is not stale.
- **The frame.** `GrMobGrow.squeezesWidth` is set by FlexChildren for a
  Row's image child. `grMobDimension(squeezable:)` then takes the flexible
  arm `relativeCap` uses, clamped by a points MaxWidth.
  - The base is unchanged: there is no main-axis proposal, so the frame
    answers with its ideal, the declared width.
  - Images only. Making every points Width squeezable would move rows no
    check has looked at.
- `GrMobPlatformImage` is UIImage, with an NSImage arm for ios/verify's
  macOS typecheck.

### Tests

- `ios/verify/mincontent.swift`: six cases.
  - Chrome's 400×100 → 110 and 50×50 → 40.
  - Unsized and percentage heights → the natural width (50).
  - A zero side → 110.
  - Margins outside the floor.
  - Still loading → 110.
  - A mutant that returns the declared width fails three of them.
- `mobile/verify/imagefloor_test.go` pins the wiring:
  - the record;
  - FlexChildren's flag;
  - grMobBox's hand-off;
  - the flexible arm;
  - the walk's read;
  - the @Observable store;
  - no `AsyncImage(` in the renderer.
- `maxwidth_test.go`'s pin of the `relativeCap` arm follows the new spelling.

### Seen on the simulator

- **Setup:** a temporary demo at the top of lesson 1.4 (reverted, never
  committed). Three 200px Rows, each with 16px padding by the theme, holding
  an Image (110×40, a 50×50 PNG data URL) beside a `FlexShrink(0)` box.
- **Results:**

  | pinned box | iOS after | Chrome (export) | iOS before |
  |---|---|---|---|
  | 180 | 40 (floor; the row overflows) | 40 | 110 |
  | 130 | 40 (168 − 130 = 38, floored) | 40 | 110 |
  | 60 | 108 | 108 | 110 |

  "Before" is read off the row extents (306 and 256 = 16 + 110 + the pin).
  The old runtime's images were not found by label at all.
- **Regression:**
  - `TutorialNativeFloorsUITests` 6/6, including the 4.11 `relativeCap`
    check.
  - 4.27's Lightbox draws the dummyimage receipt through the new loader.
  - 4.11's map is a spinner under both loaders: `staticmap.openstreetmap.de`
    is NXDOMAIN, which the lesson's prose already says.

Commit `dfd2ee6`.

## What went wrong

- **The first probe run measured an error page.** `go run
  /path/examples/layout` outside the module failed. The probe then loaded a
  missing file and reported 420 for both modes. Caught because the old/new
  split was missing, and rebuilt properly.
- **The scratch wasm module said `go 1.25`.** The repo needs 1.26.8, so the
  first build failed until the directive matched.
- **Misread the first simulator numbers** as 40/40/108 against an expected
  40/70/110. The Row has 16px of theme padding a side, so the content box is
  168. Chrome confirmed the iOS numbers exactly.

## Files touched

- `htmlout/export.go`, `htmlout/export_test.go`, `docs/api/htmlout.md`
- `blb/blb.go`, `blb/proxy.go`, `blb/proxy_test.go`
- `serve/main.go`, `serve/main_test.go`
- `examples/tutorial/chapter4.go`
- `docs/components.md`, `docs/api/blb.md`, `docs/api/index.md`
- `ios/GrMob/Runtime/{GrMobMinContent,GrMobStyle,Renderer}.swift`,
  `ios/verify/mincontent.swift`
- `mobile/verify/maxwidth_test.go`, `mobile/verify/imagefloor_test.go`
- `ai_docs/todo/next-list.md`, this doc

## Next

Closed: N-021, N-092, N-096. Declined: None. Raised: None. Deferred: None.
Promoted: None. Moved: None. Updated: None. Full list:
`ai_docs/todo/next-list.md`.
