# Phase 3, style half: the web pair catches up to the natives

Session: https://claude.ai/code/session_011WASqH3Z74UCj6VWGcQKAV

Loaded `2026-0901-2355-changelog-removal-v0.1.0-tag-ci.md` as context, then
started Phase 3 renderer parity, DOM/htmlout first. One commit:

| | |
|---|---|
| `b281070` | Phase 3: the web pair catches up to the natives on style |

## The shape of the gap

Every item below had the same failure mode, which is why none of them was
caught by any existing test: the field was declared in Go, both natives read
it, the web targets dropped it — and the style still applied cleanly, the
render still succeeded, and the only symptom was that the browser disagreed
with the device. The conformance replay (`wasm/verify`) compares structure and
props, not CSS, so it never had an opinion.

Ten rows of the plan's Phase 3 table, closed in one pass across four files:
`htmlout/edges.go` (new), `htmlout/export.go`'s `styleValue` and `renderNode`,
and `grmob-runtime.js`'s `styleFromGrMob` / `applyStyle` / `renderNode`.

| Item | was | now |
|---|---|---|
| `Padding`/`Margin` `Horizontal`/`Vertical` | dropped on both | resolved the natives' way |
| `Shadow` | dropped on both | `box-shadow` |
| `LineHeight` | dropped on both | `line-height` in px |
| `AccessibilityLabel/Hint/Hidden` | dropped on both | `aria-label` / `aria-description` / `aria-hidden` |
| `Display none` | dropped on DOM | beats the flex declaration |
| `Display hidden`/`visible` | dropped on DOM; **invalid CSS** in htmlout | `visibility` |
| `Spacer` axis | height only on both | `size × size` + `flex-shrink:0` |
| `Spacer size` on the patch path | missing on DOM | `applySpacerSize` on both paths |
| `MinWidth`…`ColumnGap`, `Animation` | read by nothing, anywhere | emitted on the web pair |
| `Modal visible` | htmlout laid out closed dialogs inline | overlay chassis |

## Decisions worth keeping

### The edge shorthand rule, and why it is deliberately lossy

`core.EdgeInsets` has six fields, not four. Both natives resolve the shorthand
identically (`parseEdges` in `GrMobStyle.swift` and `GrMobStyle.kt`):

```
top    = Top    != 0 ? Top    : Vertical
bottom = Bottom != 0 ? Bottom : Vertical
left   = Left   != 0 ? Left   : Horizontal
right  = Right  != 0 ? Right  : Horizontal
```

"Set" means non-zero, so `PaddingHorizontal(16)` + `PaddingLeft(0)` cannot ask
for a zero left inset. That was reproduced rather than fixed: a Go zero value
carries no "was it set?" bit, both natives are lossy the same way, and an inset
that resolves one way on device and another on the web is worse than one that
is uniformly lossy. `htmlout/edges.go` states the rule and says this in its doc
comment; `edgeToCSS` restates it in JS.

`components.Separator`'s `Inset` was spelled out as a `Left`/`Right` pair
*because of* this gap, with a comment saying so. It now uses `Horizontal`, and
the test that pinned the workaround was updated with the reason it is gone.

### `Display` splits across two CSS properties

The mode has five values and only three are CSS `display` keywords. That is why
the runtime emitted **none** of them (assigning `"hidden"` through `el.style`
overwrites the flex display and is then rejected, leaving the container in
block flow) and why htmlout emitted `display:hidden`, which the browser
discards.

Resolved by giving each value the treatment it needs:

- `none` → `display:none`, assigned after the flex block so it wins. Hiding
  beats layout on every target that reads Display at all.
- `hidden`/`visible` → `visibility`. That is what the natives do with the mode
  (SwiftUI `.opacity(0)`, Compose alpha 0): keep the space, drop the pixels.
  `display:none` is the other thing — no pixels *and* no space — which is why
  the two cannot share a property.
- `block` stays unemitted on the runtime (a block-level flex container is
  exactly `display:flex`; outside one the div is block already).
- `inline` keeps its existing `width:fit-content` translation.

`visible` is emitted rather than treated as a no-op default: a node inside a
hidden ancestor inherits hidden, and an explicit `DisplayVisible` is the only
way to override it. The natives get this free — opacity does not inherit.

### Shadow arithmetic, and rounding

One elevation number on every target against a CSS property that wants offsets,
a blur and a color. Took SwiftUI's `grMobShadow` as the authority (blur =
elevation/2, y = elevation/3) and its default black at a third alpha, because
CSS has no default to fall back on.

First cut printed `box-shadow:0 1.3333333333333333px …` for `Shadow(4)`. Both
targets now `round2`, so they emit the same string for the same elevation.

### `aria-hidden` wins alone

Matching `clearAndSetSemantics` on Android and `accessibilityHidden` on iOS: it
prunes the node *and its subtree*, so a name on the same node is contradictory
rather than additive. The label and description are dropped when hidden is set.

The hint maps to `aria-description`, not `aria-describedby` — the latter takes
an ID reference and a static export has no stable IDs to point at. Support for
`aria-description` is thinner than the rest of ARIA; the alternative was
dropping the author's hint entirely, which is the same call `enterkeyhint`
already made.

### Which fields are *not* in the isFlex decision

Only `Gap`, `JustifyContent`, `AlignItems` and `FlexDirection` promote a
block-flow box to a flex container. `FlexWrap`, `RowGap` and `ColumnGap`
deliberately do not: they only mean anything once the box already is one, so
promoting for them alone would change the layout to no purpose. Pinned by a
test on both sides.

## The bug that was not on the plan

`showToast` applied `styleFromGrMob`'s **total** declaration map onto a toast
that had already been given its default look. `styleFromGrMob` returns every
property it manages on every call, `""` included, so that an `update-style`
patch reusing a live element clears what the new Style dropped — correct for
the patch path, wrong for a throwaway element layered over defaults. A
`core.UseToastStyle` setting only a background blanked the padding, the radius
and the drop shadow.

It predated this session and would have widened with the new `boxShadow` and
`maxWidth`. `definedDecls` strips the `""` entries at that one call site;
`styleFromGrMob` keeps its totality everywhere else. First toast test in the
suite.

## What was refused, and why

The plan filed `HoverStyle` / `FocusStyle` / `PseudoStates` in the same row as
the rest, as "cheap: direct CSS". They are not. An inline style cannot express
a pseudo-state — the web targets need a generated stylesheet and class names,
which is a different piece of work. They are read by nothing today (verified:
`core/style.go` merges them, nothing else touches them) and now have their own
entry in the ROADMAP's new **Styling gaps** section rather than an implied
promise.

## Docs corrected rather than left to Phase 6

Two claims were made false-or-truer by this change, so they were fixed here:

- `ROADMAP.md` listed `PositionSticky, Absolute, Relative` as Done and "hover
  styles" among shipped features. Position is now real but **web-only**;
  hover styles are real nowhere. Both re-stated accurately, with the whole
  web-only group named explicitly.
- `docs/concepts/styling-and-theming.md` gained a per-target support table
  (three rows: everywhere / web-only / nowhere), the shorthand rule, the
  `Display` split, and the web half of the accessibility mapping.

## Tests

21 new, each written from the contract rather than read off the implementation
— the same discipline the conformance replay's prop table follows.

- `htmlout/export_test.go` — 10, grouped under a "Phase 3 parity" banner that
  names the shared failure mode. Includes an escaping test for
  `AccessibilityLabel` (it is user-originated like any other string).
- `wasm/verify/runtime_test.mjs` — 11, including the totality checks that a
  patch back to an empty Style clears `box-shadow`, `visibility` and the ARIA
  attributes.

Baseline green: `gofmt`, `go vet`, `go test`, `go test -race`, WASM build
(5.7 MB), `wasm/verify` (54 tests), `ios/verify` (3 checks).

## Something else is writing to this tree

`git status` was clean at session start. During the session — timestamps
interleaved with my own edits, 00:08 to 00:18 — a `core.OpenURL` / system-events
feature appeared that I did not create:

```
?? core/openurl.go            ?? mobile/sysevents.go
?? core/openurl_test.go       ?? mobile/sysevents_test.go
?? android/…/app/SystemEvents.kt
?? ios/GrMob/App/SystemEvents.swift
 M android/…/GomobileBridge.kt, MainActivity.kt, GrMobRuntime.kt
 M ios/…/GomobileBridge.swift, GrMobApp.swift, GrMobRuntime.swift
```

All of it was left unstaged; `b281070` contains only my ten files. **Check
whether another session is running here before committing anything broadly** —
a `git commit -a` would sweep another session's half-finished work into this
history.

## Carried forward

1. **Phase 3's behavioral half is open**: `TabView` (a bare div showing every
   child on DOM, missing entirely in htmlout — it also unblocks plan item 1.9's
   `onTabChange`) and `CameraView` (`camera.js` is never instantiated).
2. **Two native-side Phase 3 rows** remain: `Gap` with a non-start
   `JustifyContent` is dropped by `Renderer.kt:913-922`, and the Modal backdrop
   is ignored by both natives.
3. `onLongPress` was already done — Phase 1.10 landed `attachLongPress` before
   this session, so the plan's row was stale. Struck through with a note.
4. The `core.OpenURL` work above is uncommitted and not mine.
5. Everything from the last session that is still open: CI has never run;
   `android/build.sh` + two docs still document `go install …gomobile@latest`
   against the `tool` pin; neither native long-press path has run on a device;
   `wasm/main.go`'s `renderInitial` stale-slot re-mount (plan 2.2); 6.3's dated
   prose; **6.2's 182 undocumented `core` exports**; v0.1.0's `bytdb` dependency
   surface.

## Where the plan stands

- **Phase 1 — done. Phase 6.1 — done. Phase 7 CI — done. Phase 3 style half —
  done.**
- Open: Phase 2 (stability), Phase 3's behavioral half, 4 (performance),
  5 (quick wins), 6.2 / 6.3 / 6.4 (documentation), Phase 7's remaining hygiene.

Next per the plan: `TabView` on both web targets, which is the last structural
item in Phase 3 and unblocks 1.9.
