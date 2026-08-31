# Session: closing core gap 1, then building Avatar / ProgressBar / Separator

**Session ID:** session_01Ej9aSqY1yefk9am5LZG367
**Date:** 2026-08-31, ~01:38
**Branch:** master
**Commits:** `a1acc1c` (UseStyle), plus the widget commit that follows this doc

## Goal

Two requests in sequence. First: fix core gap 1 from the gap analysis —
`UseStyle` silently dropping most of `Style`. Then, once that was pushed:
build the three widgets it was gating.

## Part 1 — `UseStyle` (commit `a1acc1c`)

### The bug

`core/style.go` merged **14 of Style's 45 fields**. A style value carrying
`Width`, `Height`, `Top`, any flex property, or the accessibility fields
applied cleanly and did nothing — no error, no warning, no node change.

`Top` was the tell that this was rot rather than design: `Bottom`, `Left` and
`Right` were all merged and `Top` alone was not. Nobody designs that.

`Style.With` inherited the hole verbatim (it is a two-line wrapper over
`UseStyle`), though it has no callers in the tree.

### The fix

The merge body moved into `func (s Style) applyTo(target *Style)`; `UseStyle`
is now `styleFunc(s.applyTo)`. Splitting it out is what lets the nested style
fields recurse into the same semantics.

Three decisions worth keeping:

**1. Zero still means "not set".** Widening the merge must not turn it into a
wholesale overwrite, or every role style would blank out the theme defaults
beneath it — which is the entire reason `UseStyle` composes. The cost is that
it cannot *clear* a field (`Style{AccessibilityHidden: false}` does not
un-hide); the direct props remain the way to force a value. Documented on the
function and in `docs/concepts/styling-and-theming.md`.

**2. Nested styles merge rather than replace.** `HoverStyle`, `FocusStyle` and
each `PseudoStates` entry recurse field-by-field, so a value describing only
`":hover"` no longer deletes a `":focus"` the target carries.

**3. Those three fields are merged copy-on-write.** This one is load-bearing.
`containerNode` starts every node at `style := &base` where `base` is a
*shallow copy* of the theme's component `Style` — it shares the theme's
pointer and map. Writing through either would let one render permanently edit
the theme for every render after it. `mergedStylePtr` and the map copy always
allocate, so neither operand is reachable from the result.

`Responsive` (`core/style_props.go`) had that same aliasing bug, writing
`s.PseudoStates[breakpoint]` into a possibly theme-owned map. Latent only
because no theme sets the field today; fixed alongside since the copy-on-write
rule was being established anyway.

### The regression guard

`core/style_merge_test.go`. The bug existed because `Style` grew and a
hand-written field list didn't, so **the guard is reflective**: it walks
`Style`, and for each field builds a value with only that field set, applies it
to an empty target, and asserts it arrived. A new `Style` field that `applyTo`
misses fails the test by name.

Verified it actually bites by deleting the `FlexGrow` branch:

```
--- FAIL: TestUseStyleMergesEveryField/FlexGrow
    style_merge_test.go:82: field FlexGrow: dropped by UseStyle (got 0, want 7.5)
```

Plus four hand-written tests for what reflection can't express: the inverse
contract (an empty style clobbers nothing), the theme-mutation case, nested
merge semantics, and an end-to-end render asserting sizing/flex/a11y reach a
real node.

Blast radius was nil: every `UseStyle` call site in the tree passes a value
whose set fields were already in the old subset (the `Typography` roles carry
only FontSize/FontWeight/TextColor/Display), so nothing changed appearance.

## Part 2 — the three widgets

All three are pure composition on core, zero renderer work.

### `components/separator.go`

The only one of the three with real in-tree demand (2 byte-identical
instances). `core.Divider` exists but force-applies `Margin(8)`, which is why
neither example used it.

```go
type Separator struct {
    Color     string   // default hairlineColor
    Thickness float64  // default 1; fractional values pass through
    Inset     int      // left/right margin, for rules that start under the text
    Style     []core.StyleProp
}
```

- Built on `Box` — the only container with no theme base, so the rule inherits
  no padding to undo.
- **Always** `AccessibilityHidden`. A rule carries no information, and one
  between every pair of rows turns a 20-row feed into 39 utterances.
- `Inset` is left/right **margin**, not `EdgeInsets.Horizontal`: the HTML
  exporter reads only the four per-side fields.
- **No `Vertical` field, deliberately.** A vertical rule needs cross-axis
  stretch, and neither renderer maps `AlignItems: "stretch"` (Compose falls
  through to `Alignment.Top`, SwiftUI to `.top`), so it would collapse to zero
  height on both. Adding the field would advertise something that does not
  work.
- The default tint is a package constant, not a theme lookup, because gap 2 is
  still open — `ColorPalette` has no `Border` role. `Surface` is the nearest
  neutral but it is a *fill* color: on a Surface panel a Surface hairline is
  invisible. One definition is the improvement available today.

**Both call sites migrated:** `examples/todoapp/app.go` and
`examples/mobileapp/app.go`. `todoapp` keeps passing its own `colorHair`
constant (the app deliberately owns its accent palette); `mobileapp` had the
value written out inline and now takes the widget default.

### `components/avatar.go`

```go
type Avatar struct {
    Src, Name, Initials string
    Size                float64  // default 40
    Background, TextColor string
    Style               []core.StyleProp
    AccessibilityLabel  string
}
```

- Both branches are the same square with `BorderRadius = Size/2`. An oversized
  fixed radius (Badge's 999 trick) also yields a circle, but would silently
  keep the old geometry when a caller changed `Size`. Deriving keeps `Size` the
  single knob.
- **The disc is a `Row`, not a `Box`.** `Box` is a ZStack pinned to
  `.topLeading` on iOS and a Compose `Box` pinned to `TopStart` — neither
  centres a child. `Row` honors `JustifyContent` on the main axis and
  `AlignItems` on the cross axis on both platforms. `Padding(0)` undoes the
  theme Row's screen-level padding, which would otherwise inflate the disc past
  `Size`.
- Initials derive from the **first and last** words of `Name` — "Ada King
  Lovelace" is AL, not AK — uppercased, rune-based so a non-Latin name keeps
  whole characters instead of a mangled leading byte. Tested with Cyrillic and
  CJK.
- Initials font size is `Size * 0.4`, proportional so the disc scales as one
  piece.
- **Image avatars default to a Surface background.** `core.Image` bases every
  image on `Components.Camera`, whose background is solid black — right behind
  a viewfinder, wrong behind a portrait that has not downloaded yet, where it
  shows as a black disc until the bytes arrive.
- **Accessibility is synthesized here**, unlike `ListRow`, because an avatar
  has exactly one meaning and `Name` is it: explicit label → `Name` → *hidden*.
  The last case matters most: an unnamed avatar is decoration beside text that
  already names the person, and unlabeled it is announced as "image" or read
  out as its URL.
- Known limitation, documented: non-square images letterbox inside the circle
  (`.scaledToFit` on iOS, Compose's Fit default). Filling needs a `ContentMode`
  prop on `Image` — a two-renderer pass of its own.

### `components/progress_bar.go`

**The gap analysis's implementation note for this widget was wrong, and the
reason is worth recording.** It said: use two boxes with `FlexGrow(v)` /
`FlexGrow(1-v)`, not percentage widths. That is exact on Android, where
`FlexGrow` maps onto Compose's `Modifier.weight` — and silently wrong on iOS,
where it maps onto `frame(maxWidth: .infinity)`. SwiftUI stacks have no weight;
`GrMobGrow`'s own doc comment says multiple growers "split leftover space
equally regardless of their weights". Every bar would render at 50% on iOS, at
every value, with nothing in the tree to suggest a bug.

A percentage width is proportional on all three targets instead:

| target | mapping | accuracy |
|---|---|---|
| Android | `fillMaxWidth(fraction)` | exact |
| HTML | `width:<pct>%` | exact |
| iOS | `containerRelativeFrame` | proportional, but measured against the nearest *container* |

The iOS caveat: a bar that spans its container (the common case — full-width in
a screen column) is exact; one inset inside a narrow card reads wider than it
should. An over-long fill in an uncommon layout beats a permanently-half-full
bar everywhere. When proportional weights land on iOS this can move to flex.

Other decisions:

- **The fill renders at every value**, zero-width included. A constant child
  count keeps advancing progress a *style patch* on one node instead of an
  insert/remove — which is also what lets a `Transition` animate it.
- The fill carries its own `Height`: a Compose Box and a SwiftUI ZStack both
  size to content, and this one has none, so without it the fill is a
  zero-pixel line inside a correct track.
- `Value` is clamped, not rejected; NaN reads as 0. A bar fed a live ratio
  should pin at full and keep rendering.
- The percentage is appended to the accessibility label because no renderer has
  a progress semantic to carry the value natively. Unlabeled → hidden.
- **A test caught real float noise.** `%g` on `0.333*100` prints
  `33.300000000000004`, which would have travelled to every renderer and into
  the exported CSS. Now rounded to hundredths of a percent before formatting —
  0.01% of a 400px track is 0.04px, and `%g` still drops trailing zeros so
  whole values stay short.

## `htmlout` also needed fixing

`htmlout/export.go` emitted no `width` or `height` at all. All three widgets
size themselves, so a 1px `Separator`, an `Avatar` disc and a `ProgressBar`
fill each exported as a zero-height or full-width box — the HTML target
silently disagreed with both natives about the layout. Core's dimension strings
("40px", "45%", "auto") are already CSS lengths, so both fields emit verbatim.

## Verification

`go build ./...`, `go vet ./...`, full `go test ./...` green throughout. The
two `gofmt` complaints (`components/badge.go`, `examples/todoapp/store.go`) are
the same pre-existing noise noted in the last two sessions.

Beyond the suite, all three widgets were rendered together in a real tree (a
throwaway module in the scratchpad with a `replace` directive) and both the
node structure and the HTML export inspected:

```
Row  [A=center Pad=8/16]                          <- ListRow
  Row  [W=40px H=40px Bg=#007AFF R=20 J=center A=center a11y=Ada Lovelace]
    Text {"content": "AL"}
  Column  [FlexGrow=1]
    Text "Ada Lovelace" / Text "Analytical engine"
  Text "3"  [Bg=#007AFF R=999]                    <- Badge
Box  [H=1px Bg=#E5E5EA a11y=hidden Mar=0/56]      <- Separator{Inset: 56}
Row  [H=6px Bg=#F2F2F7 R=3 a11y=Upload, 45 percent]
  Box  [W=45% H=6px Bg=#007AFF R=3]
Row  [H=10px ...] > Box [W=0%   ...]              <- Value 0
Row  [H=10px ...] > Box [W=100% ...]              <- Value 1
```

The HTML export confirms the disc becomes `display:flex` with
`justify-content:center` (so the initials centre), the separator emits
`height:1px; margin:0px 56px`, and the fill emits `width:45%` inside a
block-flow track.

`mobileapp` and `todoapp` both run their tests with `core.SetDebugMode(true)`
and assert an empty concern dump — still clean after the Separator migration.
None of the three widgets calls a hook, so no cursor drift.

## Files touched

**New**
- `core/style_merge_test.go`
- `components/separator.go` + `_test.go` (4 tests)
- `components/avatar.go` + `_test.go` (6 tests)
- `components/progress_bar.go` + `_test.go` (6 tests)

**Modified**
- `core/style.go` — `UseStyle` → `applyTo` + `mergedStylePtr`
- `core/style_props.go` — `Responsive` copy-on-write
- `htmlout/export.go` — emit `width` / `height`
- `examples/todoapp/app.go`, `examples/mobileapp/app.go` — Separator migration
- `docs/components.md` — Separator / Avatar / ProgressBar sections
- `docs/concepts/styling-and-theming.md` — the "subset" warning was accurate
  documentation of the bug; now describes the real contract
- `docs/concepts/views.md` — `Divider` cross-reference to `Separator`
- `docs/index.md` — widget list

## Backlog after this session

- **Core gap 2 is now the blocker in front of everything themed**:
  `ColorPalette` has no `Border`, `Success` or `Warning`. It is why
  `Separator`'s tint is a constant, and it is what a `Badge{Variant:}` or an
  `Alert` would need next.
- **Core gap 3** — the `[]core.PropsAndChildren` append idiom — still open at
  `chat/main.go:147`. A `core.MaybeProp(cond, prop)` deletes it.
- **Tier 1 widgets left:** `Button` variants (5 instances), `Screen` scaffold
  (5). `ListRow` and `Separator` are done.
- **Tier 2:** `InputRow`/composer, `StatTile`, `EmptyState`,
  `SegmentedControl`.
- **Renderer work these widgets identified, none of it blocking:**
  1. Proportional flex weights on iOS (custom SwiftUI `Layout`) — would let
     `ProgressBar` move off percentage widths and fix any future
     multi-grower row.
  2. `AlignItems: "stretch"` on both renderers — unblocks `Separator.Vertical`.
  3. A `ContentMode` prop on `Image` — unblocks avatar images that fill
     rather than letterbox.
- **Still true from two sessions ago:** `ROADMAP.md` lists `UseMemo` and
  `UseReducer` as done; neither identifier exists in the tree.
