# Session: tutorial layout & widget audit

Session: https://claude.ai/code/session_01Mv1iFwxr7pCAdxCFMKT8Pk
Date: 2026-09-04 (follows "tutorial-chapter-links" in this directory)
Local: `go run ./serve -addr :8099` after `./build.sh`

## Ask

Go over each of the 40 lessons and check layout and widget functionality —
"at first glance some things seemed not to work".

## Method

Driving the live browser build rather than reading code, because every bug
found here lives *below* the Go tree: `go test ./examples/tutorial` was green
throughout, and stayed green while the tutorial rendered wrong.

Three harnesses, installed into the page over CDP:

- **Layout audit** — per lesson, walk the DOM and flag: the Scroll node's
  horizontal overflow, any element whose right edge passes the phone screen's,
  text under 3:1 contrast against its nearest painted ancestor, and code-block
  lines taller than ~1.6 line-heights (i.e. wrapped).
- **Interaction probe** — click every `[data-has_listener_on-click]` and
  exercise every input, watching a MutationObserver on `.screen` for a
  response. Selecting on `button` alone was wrong: an Accordion header is a
  clickable `<div data-node-type="Row">`, so 4.3/4.4 first read as having zero
  controls.
- **Synchronous navigation** — `HostEvent("route", …)` + `RenderAgain()` +
  `GrMob.patch()` instead of `location.hash`. The tab is backgrounded while
  Claude drives it, so Chrome throttles `setTimeout` to ~1/s and `rAF` never
  fires; an early sweep timed out at 45 s and a click looked like it took a
  second. Nothing was slow — a real click is 4–7 ms, `RenderAgain` 2 ms.

## Two false alarms worth remembering

- **Colors in screenshots are inverted.** Dark Reader runs in `filter` mode
  (`data-darkreader-mode=filter`), which inverts paint and leaves computed
  styles alone. Screenshots are still exact for geometry; trust
  `getComputedStyle` for color, never the image.
- **Most "dead" controls are correct.** Clicking an already-selected segment,
  Reset at zero, "Clear log" with an empty log, or 8.2's "Repair (set B = A)"
  before the counters skew all legitimately change nothing.

## The main bug: core.Gap never rendered on the web

`gap` is the CSS shorthand for `row-gap` + `column-gap`. `styleFromGrMob` is
*total* — it restates every property it manages on every pass so an
update-style patch clears what the new Style dropped — so it wrote

```js
out.gap = style.Gap ? `${style.Gap}px` : "";      // line ~667
...
out.rowGap = style.RowGap ? … : "";               // line ~795
out.columnGap = style.ColumnGap ? … : "";
```

`Object.assign(el.style, out)` applies these in insertion order, so setting the
two longhands to `""` — which is what happens whenever `RowGap`/`ColumnGap` are
unset, i.e. almost always — **erased the shorthand that had just been set**.

Measured: **0 of 671** Row/Column containers across the 40 lessons carried any
gap. Every `core.Gap()` in every GrMob web app rendered as no spacing.

Go was innocent (the patch JSON carries `"Gap":16`), and htmlout was innocent —
it appends declaration *strings* and simply omits absent ones, so shorthand and
longhand never collide there. The bug was CSSOM-specific, which is also why
`wasm/verify` missed it: dom.mjs's `style` is a plain object with no
shorthand/longhand semantics.

Fix: resolve Gap into the two longhands and never write the shorthand.

```js
const rowGap = style.RowGap || style.Gap;
const columnGap = style.ColumnGap || style.Gap;
out.rowGap = rowGap ? `${rowGap}px` : "";
out.columnGap = columnGap ? `${columnGap}px` : "";
```

Axis values still win over the isotropic one — the same result htmlout gets
from the cascade. After: **607/671** containers gapped.

## Changes

- `wasm/grmob-runtime.js`
  - The gap fix above.
  - Toast layer: appended to the app root's *parent* with `position: absolute`
    when that box establishes a containing block, else the old
    `document.body` + `fixed`. Toasts were stretching across the whole browser
    window instead of the phone. Parent, not the mount point itself, because
    `mount()` clears the mount point's `innerHTML` and would detach the layer
    on the next `RenderInitial`; `ensureToastLayer` now tests `isConnected`
    for the same reason. `getComputedStyle` is `typeof`-guarded so the
    verify shim keeps working. Its own `gap: "8px"` became `rowGap`.
- `wasm/index.html` — `.screen` gains `color: #000000` and
  `transform: translateZ(0)`.
  - The ink: `body { color: var(--fg) }` is a near-white in the dark palette
    and cascaded into the hard-coded-white phone, so the ~20 `core.Text` calls
    that name no color (Style.TextColor is only emitted when set) drew at
    1.36:1. Invisible, but only for readers whose OS is in dark mode.
  - The transform makes the screen the containing block for `position: fixed`
    descendants, i.e. `core.Modal`'s chassis — a modal used to dim the entire
    site, header included. `overflow: hidden` then clips it to the bezel.
- `core/style_props.go` — added `core.WhiteSpace(value)`. `Style.WhiteSpace`
  was already carried by both web targets but nothing could set it; this is
  the missing constructor, not a new capability.
- `examples/tutorial/widgets.go`
  - `codeBlock`: `core.Overflow("auto")` on the Column, `WhiteSpace("nowrap")`
    on each line. Code lines wrapped in **37 of 40 lessons** (7 of 9 lines in
    7.2), and a wrap restarts the overflow at column zero — destroying exactly
    the indentation the NBSP substitution exists to preserve. Now scrolls
    sideways; 0 wrapped lines.
  - `segWrap` — the shared `[]core.StyleProp{core.FlexWrap(true)}` applied to
    all 15 tutorial SegmentedControls.
- `examples/tutorial/chapter{1,2,3,4,5,7}.go` — `Style: segWrap` on each
  SegmentedControl; `core.FlexWrap(true)` on 4.2's hand-built chip Row.
- `wasm/verify/runtime_test.mjs` — three gap regression tests.

## Gotcha: the parity contract

Putting `FlexWrap` inside `components.SegmentedControl` broke
`examples/todoapp`'s `TestFilterBarMatchesLegacyMarkup`, which pins the
component to byte-for-byte equality with the hand-rolled bar it replaced. That
contract is deliberate, so the wrap moved out to the tutorial's call sites
through the `Style` field the component already exposes. Long captions are the
tutorial's problem, not every app's.

## Gotcha: caching

`location.reload(true)` is ignored by modern Chrome and a `?v=` on the page
does not bust `grmob-runtime.js`. Runtime edits need a real hard reload
(`cmd+shift+r`) or the page keeps executing the old script — this cost one
round of "the fix didn't work".

## Verification

- `go test ./...`, `sh wasm/verify/run.sh`, gofmt — all clean.
- The three new runtime tests were confirmed to *fail* against the pre-fix
  runtime (3 failed / 51 passed), so they actually pin the bug.
- Final browser sweep of all 40 lessons: **0 wrapped code lines, 0 overflow,
  0 low-contrast text, 607/671 containers gapped**, every interactive control
  responsive, modal rect exactly equal to the screen rect, toast layer inside
  `.screen`.
