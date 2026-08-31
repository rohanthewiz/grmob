# Session: `components.Screen` — the scaffold, and what a zero value has to *not* do

**Session ID:** session_01Ej9aSqY1yefk9am5LZG367
**Date:** 2026-08-31, ~15:33
**Branch:** master
**Follows:** `2026-0831-1515-button-variants.md`

## Goal

The last Tier 1 item from the component gap analysis: the root scaffold, 5
instances — `chat/main.go:77`, `fintechapp/main.go:32`, `mobileapp/app.go:51`,
`todoapp/app.go:188`, `layout/main.go:12`.

**A note on how the session opened.** The first instruction was "close core gap
3 — add `core.MaybeProp`", which was already done: it landed in `d4f90dc`, one
commit behind `HEAD`, with `core/conditionals.go:125`, the nil-item contract in
`containerNode`, the debug-mode exemption, and 4 tests. Reported as already
closed with the evidence rather than re-implemented, and the session moved to
the real remaining item.

## What the 5 sites actually had in common

Less than the gap analysis's sketch (`SafeArea > (Scroll?) > Column(FlexGrow(1),
Gap(n))`) implied. Surveyed before writing anything:

| site | Scroll | Gap | FlexGrow(1) |
|---|---|---|---|
| chat | — (its `MessageList` scrolls) | — | — |
| fintechapp | **yes** | — (non-uniform Spacers) | — |
| mobileapp | — | — | — |
| todoapp | — (its `List` scrolls) | 12 | **yes** |
| layout | — | 8 | — |

No two agreed, and only one site wanted each of the three optional pieces. The
shape is not hard to type — the value of naming it is that there is now one
place to hang what a screen root will eventually need (the keyboard-aware
scroll area the ROADMAP wants, pull-to-refresh, per-platform inset behavior)
instead of five.

```
SafeArea
  └─ Scroll            (only when Scroll is true)
       └─ Column       ← Gap / Fill / Style land here
            ├─ Children[0]
            └─ …
```

## The design decision: every field defaults to contributing nothing

Same discipline as last session's Button, and for the same reason — it is what
let all five migrations come out byte-identical. A field only speaks when the
caller sets it; with none set, `Screen` renders exactly `SafeArea(Column(children...))`
and the theme's `Column` base carries through untouched.

### Where that is load-bearing rather than tidy: `Gap`

Style props **set** rather than merge (`core/style_props.go:91` is
`s.Gap = px`, applied directly to the style struct). So an implementation that
always calls `core.Gap(s.Gap)` would overwrite a gap the theme's `Column` base
had set, whenever the caller left the field alone. Unset and zero therefore
have to mean the same thing here — *the absence of a gap, not the imposition of
one*:

```go
if s.Gap != 0 {
    items = append(items, core.Gap(s.Gap))
}
```

### `Fill` is a field, not a default

Defaulting `FlexGrow(1)` on would have been the "nicer" scaffold and would have
changed 4 of the 5 trees. It is genuinely load-bearing at exactly one site:
todoapp's `List` asks to grow into the leftover space, and a flex child can only
grow inside a parent that has height to give. Elsewhere the content is
content-height on purpose.

### `Style` goes last

`containerNode` applies style props in argument order, so appending caller
`Style` after `Gap`/`Fill` is what makes it an actual escape hatch rather than a
suggestion. Reversing the order is one of the mutations below.

## Finding — the zero-value test has the same blind spot Button's had

`TestScreenZeroValueIsExactlySafeAreaColumn` compares whole trees against a
hand-built `SafeArea(Column(...))` under both bundled themes. It is the right
test and it cannot see the `Gap` guard at all: neither bundled theme sets a
`Column` gap, so `Gap(0)` writes a zero over a zero and a
mutant that always applies the prop produces an identical tree.

The fix is the same shape as last session's discriminating theme — a
purpose-built theme whose `Column` base carries `Gap: 7`, where "apply nothing"
and "apply zero" finally disagree. That mutant now fails with
`zero Gap overwrote the theme's Column gap: got 0, want 7`.

Two sessions running, the honest zero-value test has needed a second theme to
be worth anything. Worth remembering as the default move, not a special case.

## Finding — `Scroll` + `Fill` earns documentation, not a debug concern

The first instinct was a `ConcernScreenScrollAndFill` in debug mode: a growing
column inside a scroll view looks contradictory. It was dropped, because the
combination is legitimate — it makes the scrolled content *at least* as tall as
the viewport, which is how a footer gets bottom-anchored on a short page
(`min-height:100%` in CSS terms). There is no clean predicate that separates
that from a mistake, and a concern that fires on correct code is worse than a
paragraph. So both halves are documented instead: `Scroll` is for screens with
no scrolling region inside them (a scroll view nested in a scroll view fights
for the same drag on both natives), and `Fill` with `Scroll` is unusual but
means something specific.

## Nil children come for free, and are pinned anyway

`Children` flows into `core.Column`'s argument list, which skips nil items — the
contract `MaybeProp` established last session. So the conditional-region idiom
costs the tree no node at all:

```go
var banner core.View
if offline {
    banner = OfflineBanner()
}
components.Screen{Children: []core.View{banner, body}}
```

`TestScreenSkipsNilChildren` pins it here rather than leaving it to core's
tests, because what it actually guards is that `Screen` passes children through
**untouched**: wrapping each one — in a `Fragment`, a `Box`, anything — turns
the absent child back into a real node and reopens the stray-flex-slot bug
`MaybeProp` exists to close. (A version that filtered the nils out first would
also pass, and that is fine — same tree. The failure guarded against is a node
appearing, not how it fails to.) The test comment initially claimed it would
catch pre-filtering; that was wrong and was corrected before commit.

## Verification

Six mutations, each caught by exactly one test:

| mutation | caught by |
|---|---|
| `Gap` applied unconditionally | the discriminating-theme test (`got 0, want 7`) |
| `Fill` ignored | `Fill = true gave FlexGrow = 0, want 1` |
| `Scroll` ignored | shape assertion — `want SafeArea > Scroll` |
| `Style` applied before Gap/Fill | both overrides fail (`Style Gap did not win: got 12, want 3`) |
| each child wrapped in a `Fragment` | nil child → **panic** |
| scaffold fills by default | the zero-value test, both themes |

Rendered output diffed before/after for every migrated root — **all five
byte-identical**. Baselines were captured before touching anything; `mobileapp`
and `todoapp` have no `main`, so a throwaway `zz_baseline_dump_test.go` was
added in each package to export HTML under an env var and **deleted after**.

`go build ./...`, `go vet ./...`, full `go test ./...` green.
`examples/todoapp/store.go` remains the only `gofmt` offender.

## Migration notes worth keeping

- **chat** — `Scroll: false`. The scrolling region is `MessageList`'s own
  `Scroll`, which is what keeps the header and composer put while only the
  thread moves.
- **fintechapp** — the only root that scrolls as a whole. `Gap` deliberately
  unset: spacing is 24/24/28, and the Gap-for-uniform-runs / Spacer-for-the-
  exception rule (established when that file was written) still holds.
- **mobileapp** — the pure zero value; four lines of nesting become the
  scaffold with no props at all.
- **todoapp** — the only `Fill: true`, with a comment saying *why* it is
  load-bearing rather than decorative.
- **layout** — `Gap: 8`, still wrapped in the outer `core.WithTheme`.

## Files touched

**New**
- `components/screen.go` — `Screen{Children, Scroll, Gap, Fill, Style}`
- `components/screen_test.go` — 7 tests / 2 subtests, plus a `describe`
  tree-printer for the shape assertions (a `%+v` of a `*Node` is unreadable at
  that depth)

**Modified**
- `examples/chat/main.go`, `examples/fintechapp/main.go`,
  `examples/mobileapp/app.go`, `examples/todoapp/app.go`,
  `examples/layout/main.go` — all five roots
- `docs/components.md` — a full `## Screen` section, placed **ahead of**
  `## Button`: it is the scaffold a reader building an app meets first
- `docs/index.md` — widget list

## Backlog after this session

- **Tier 1 is done.** The gap analysis's Tier 1 (Badge variants, Button
  variants, Screen scaffold) is fully closed.
- **New, small:** chat's scaffold sets no `Fill` even though its middle region
  scrolls. Pre-existing and left alone to keep this session's diff honest, but
  it is likely why that layout depends on the composer's intrinsic height —
  worth a look when the keyboard-aware scroll area lands.
- **Tier 2:** `InputRow`/composer, `StatTile`, `EmptyState`, `SegmentedControl`.
- **Still open against the theme, unchanged from last session:**
  `DefaultTheme.Components.Button` is white on `#007AFF` at **4.02:1**, below
  WCAG AA for 17px text; and a second palette value per status role (an
  "on-light" tone) would make Button's Outlined/Ghost legible for Success and
  Warning. Both are palette decisions that repaint every tree, not drive-by
  fixes.
- **`Variant` has two consumers** (Badge, Button). `Alert`/banner is the
  obvious third and needs no new core work; `Chip{Variant:}` is the fourth —
  and Chip is still the odd one out, hand-rolling a selected treatment that
  `Emphasis` could express.
- **A `Neutral`/muted variant** mapping to Surface is still the addition a
  status set usually wants next; `Button.Disabled` already hand-rolls exactly
  that pair.
- **Renderer work, none of it blocking, unchanged:**
  1. Proportional flex weights on iOS (custom SwiftUI `Layout`) — lets
     `ProgressBar` move off percentage widths.
  2. `AlignItems: "stretch"` on both renderers — unblocks `Separator.Vertical`.
  3. A `ContentMode` prop on `Image` — unblocks avatar images that fill rather
     than letterbox.
  4. No renderer carries a disabled state. `Button.Disabled` is honest without
     it, but a native `enabled=false` would get the platform's own press/focus
     behavior.
- **Still true from six sessions ago:** `ROADMAP.md` lists `UseMemo` and
  `UseReducer` as done; neither identifier exists in the tree.
