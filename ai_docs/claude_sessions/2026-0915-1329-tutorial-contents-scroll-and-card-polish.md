# Session: tutorial contents — scroll fix and chapter card polish

Session ID: b24b2f4e-57ff-4f54-b0fd-629f29c5aaeb
Date: 2026-09-15 (follows "tutorial-layout-widget-audit" in this directory)
Local: `./build.sh && go run ./serve -addr :8099`

## Ask

1. "On the interactive tutorial in the browser the initial page is not
   scrollable. Navigation seems broken there."
2. Spin it up in Chrome and show it.
3. Fix the cosmetic issues the screenshots showed.

## Part 1 — the contents screen could not scroll

### Cause

3b24f40 ("The contents screen as a List") turned `Home` from a
`Screen{Scroll: true}` over a Column into `comps.Screen{Children: {core.List(…)}}`.
Right on both natives (List is LazyColumn / SwiftUI List), wrong on the web
host: `grmob-runtime.js` gives neither Scroll nor List any `overflow`, and
`wasm/index.html` only ever made **Scroll** the viewport:

```css
#app [data-node-type="Scroll"] { flex: 1 1 0; min-height: 0; overflow-y: auto; … }
```

Measured in headless Chrome: the List was **2226px tall inside a 709px
screen**, `overflow-y: visible` on it and every ancestor, and `.screen`'s
`overflow: hidden` clipped it. A mouse wheel scrolled nothing.

"Navigation broken" was the same bug, not a second one: tapping 1.1 / 1.5 set
the hash, opened the lesson and ‹ Contents came back — but only because the
probe used `scrollIntoView`, which a reader cannot. Everything below the fold
(chapters 2–8) was unreachable.

### Fix

```css
#app [data-node-type="List"]:not([data-node-type="Scroll"] [data-node-type="List"]) {
  flex: 1 1 0; min-height: 0; overflow-y: auto; overscroll-behavior: contain;
}
```

- The `:not(…)` matters. Inside a Scroll the column has no definite height, so
  a zero basis would collapse the list — the horizontal-Scroll bug on the other
  axis. Lesson 4.3's `outlineDemo` is a List inside the lesson Scroll and
  measured 254/254px after the fix.
- A first candidate (`min-height: 0; overflow-y: auto` without the zero basis)
  did nothing: the List kept the runtime's `flex: 1 1 auto` and its parent
  Column still grew to content height.
- Applied to all three host pages carrying the Scroll rule:
  `wasm/index.html`, `cmd/grmob/templates/wasm/index.html.tmpl` (so
  `grmob new` scaffolds it), `wasm/shots/index.html`; `docs/platforms/wasm.md`
  describes it.

After: List 709/2226, wheel scrolls it 1200px, taps and back still work.

## Part 2 — chapter card cosmetics

Found from screenshots, diagnosed with a DOM geometry dump.

| Symptom | Cause | Fix (examples/tutorial/home.go) |
|---|---|---|
| Chapter title 48px inside the card, wrapping ("Views & / Layout") | Theme `Components.Row` base 8/16 on the header `core.Row` **plus** the disclosure button's own 8/16, inside the Card's 16 | `core.Padding(0)` on the header Row; `ControlStyle: PaddingHorizontal(0)` on the band (vertical 8 kept for the press target) |
| "5 lessons" folded onto two lines | Caption shrank to min-content when the title took the slack | `lessonCount` — caption type with `FlexShrink(0)` |
| Rows inset twice | Theme Column base 12/16 on `core.Column(rows...)` inside the Card, and ListRow insets 16 itself | `core.Padding(0)` on that Column |
| 32px between cards, cards 8px in from the title | List `Gap(16)` + the theme Card base's 8px margin on every side | `core.Margin(0)` on chapter cards and `progressCard` |
| Chevron a 6px speck | Band chevron at Caption (13px); ▸/▾ are the *small* triangles, ~½ em | `ChevronStyle: UseStyle(Body), FontSize(20)` and `ControlStyle: Gap(4)` |

### The chevron needed an API

`comps.CollapseBand` had no way to reach the glyph — `disclosure.ChevronStyle`
is unexported and CollapseBand hard-coded Caption. Added
`CollapseBand.ChevronStyle []core.StyleProp`, applied **after** the Caption
default, so an empty value builds exactly what the default band builds
(`TestACollapseBandIsTheSameDisclosureTheDefaultBandBuilds` still compares like
with like). Ignored on an inactive Collapse, which draws no chevron. New test:
`TestACollapseBandsChevronTakesTheCallersType`.

### 22px was one step too far

At `FontSize(22)` with the default 8px gap, "Chapter 6 — Navigation & Overlays"
wrapped by **1.5px** (need 343.5, have 342). Every other chapter had ≥8px
slack. 20px + `Gap(4)` gives ~3.5px on the longest title; all eight headers
measured one line, 45px tall. It is still tight — a longer chapter title or a
wider font on another host will wrap it, which degrades gracefully.

Known leftover: at 20px the ▸ sits slightly below the title's optical centre
(glyph metrics). Left alone.

## Harness notes

- Probes were throwaway Node scripts in the session scratchpad driving headless
  Chrome over CDP with Node's built-in WebSocket (same approach as
  `wasm/verify/browser.mjs`): measure boxes, dispatch real `mouseWheel` and
  `mousePressed`/`mouseReleased`, `Page.captureScreenshot`.
- `js()` helpers that return an "EXC …" string on exception are truthy — a
  wait loop on `!!querySelector(...)` must compare explicitly or it exits early.
- A screenshot taken ~1s after the List appeared caught the boot spinner once;
  3.5s settled. Running a probe concurrently with `wasm/verify/run.sh` (its own
  Chrome) made `captureScreenshot` return no data.
- Don't patch probe scripts with `sed` substitutions containing nested quotes —
  one produced a silently broken expression (`tap: undefined`).

## Verification

- `go vet ./comps ./examples/tutorial`; `go test ./comps/... ./examples/...
  ./core/... ./wasm/... ./cmd/...` — all ok. `TestHomeTreeSize` is an
  order-of-magnitude bound, so the added props do not trip it.
- `sh wasm/verify/run.sh` — exit 0.
- gofmt clean.
- Browser: contents scrolls; lesson taps / ‹ Contents work; 4.3's nested List
  keeps its height; eight one-line chapter headers; 16px gaps; progress and
  chapter cards aligned with the title at x=379 content edge.

## Not done

- The deployed GitHub Pages site stays broken until this is pushed and the site
  workflow redeploys.
- Natives were not re-run for the home.go spacing changes; the insets removed
  are theme bases that apply on every target, so they should tighten there too.
