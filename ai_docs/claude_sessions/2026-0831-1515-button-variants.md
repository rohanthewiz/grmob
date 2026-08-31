# Session: `components.Button` — two color axes, and four things it uncovered

**Session ID:** session_01Ej9aSqY1yefk9am5LZG367
**Date:** 2026-08-31, ~15:15
**Branch:** master
**Follows:** `2026-0831-1500-maybeprop-core-gap-3.md`

## Goal

Tier 1, item 2 from the component gap analysis: `Button` variants, 5 hand-rolled
instances (`chat/main.go:178`, `fintechapp/main.go:95`, `todoapp/app.go:253`,
`todoapp/app.go:326`, `social/tab.go:8`). `core.Button` offers exactly one look,
so every app re-spells bg + fg + padding + radius by hand.

## The design decision: two axes, not one enum

The gap analysis sketched a single enum — `Primary | Secondary | Danger | Ghost`.
It was dropped, because it conflates two independent questions: **which** color
(the meaning) and **how much** of it (the visual weight). A flat enum cannot
express an outlined destructive button — the ordinary shape of a "Delete"
confirmation — without a fifth value, and then a ghost destructive needs a sixth.

So:

- **`Variant`** — reused *verbatim* from last session's Badge work
  (Default/Success/Warning/Error). A danger Button and an error Badge are now
  the same red by construction rather than by two palettes agreeing.
- **`Emphasis`** — new, in `components/button.go`: `EmphasisFilled` (zero) /
  `EmphasisOutlined` / `EmphasisGhost`.

| emphasis | fill | label | rule |
|---|---|---|---|
| Filled (zero) | the variant's color | contrast-picked against the fill | — |
| Outlined | transparent | the variant's color | 1px, the variant's color |
| Ghost | transparent | the variant's color | — |

Both are string enums with empty zero values, matching `Variant`, `Alignment`,
`DisplayMode`.

### Secondary is deliberately not a Variant

Last session established the distinction on the palette: `Secondary` is a brand
slot a theme may set to anything, `Success`/`Warning`/`Error` carry meaning. So
there is no `VariantSecondary`, and fintech's "Recharge" — which wants the
brand's teal — goes through `Style`, the documented escape hatch. That call site
is now the worked example in both the type doc and `docs/components.md`.

## Finding 1 — the zero value must apply *nothing*

Not "re-derive Primary from the palette". A theme's `Components.Button` is
allowed to differ from `Colors.Primary`, and re-deriving would silently
overwrite that choice. So with both axes at zero, `colorProps` returns `nil` and
the theme base carries through untouched:

```go
if b.Variant == VariantDefault {
    return nil
}
```

**The first version of the guard could not tell the two implementations apart.**
`TestButtonZeroValueIsExactlyCoreButton` compares against a rendered
`core.Button`, which is right, but both bundled themes pair `Components.Button`
with `Colors.Primary` + `Colors.Background` — so a re-deriving implementation
produces a byte-identical Style and the test passes. Mutation testing caught it:
deleting the early return changed nothing.

The fix is a second test that renders under a purpose-built theme where the two
disagree — palette says blue-on-white, `Components.Button` says
charcoal-on-amber. That mutation now fails loudly. A first attempt at a
"self-guard" (asserting the theme base carries padding/radius) was written and
then *deleted*: it proved a different fact and would have read as protection it
did not provide.

## Finding 2 — `examples/fintechapp` has no `Components` block at all

Finding 1 surfaced it immediately: "Transfer" rendered with **zero** styling
once it moved onto the widget. Its `MaterialTheme()` sets `Colors`, `Typography`
and `Spacing`, and stops. Every widget in that app was hand-styled at the call
site, so the omission had been invisible.

This is pre-existing — `core.Button` behaves identically there — but the widget's
"apply nothing" contract is what made it visible. Fixed by giving the theme the
values the local `MaterialButton` helper used to hardcode per call, so the
rendered result is unchanged and they now live in one place.

Worth generalizing, and now in `styling-and-theming.md`: the palette's
resolver/fallback reasoning **does not extend to `Theme.Components`**. It is a
plain struct literal, so any field may simply be unset, and the zero `Style`
that results is genuinely no styling rather than a default. The rest of
fintech's `ComponentDefaults` (Card, Input, Row, Column) is still empty —
deliberately left, with a comment, since nothing reads them yet.

## Finding 3 — transparent has to be a real color

Outlined and Ghost cannot *omit* the fill: an empty `Background` means "inherit
the theme's Button base", which is a solid Primary. `core.Style` has no clear /
unset for a color, so:

```go
const ColorTransparent = "#00000000"
```

Verified rather than assumed: both natives' `parseColor` handle the 8-digit
form with **alpha last** (CSS byte order) — `GrMobStyle.kt:176` recomposes the
channels for android.graphics, `GrMobStyle.swift:143` reads it directly — and
htmlout emits the string verbatim as a CSS Color 4 hex.

## Finding 4 — htmlout never emitted borders

`BorderColor` / `BorderWidth` are in `core.Style` and merge correctly, and
**both natives already honor them** (Compose `Modifier.border`, SwiftUI
`.grMobBorder`). Only the HTML exporter dropped them, so an outlined button
would have had an edge on device and none in the export — exactly the class of
silent target disagreement the `Width`/`Height` emission fixed earlier.

Both halves are required, matching Compose's own guard
(`borderWidth > 0f && borderColor != null`): a color with no width draws nothing
there, so it must draw nothing here. `TestBorderNeedsBothWidthAndColor` pins all
four combinations.

## A bug fixed on the way — `examples/social`

`TabButton` set a `TextColor` and no `Background`, so it inherited the theme's
solid blue pill. The rendered truth, captured before touching it:

```
ACTIVE:   color:#007AFF; background:#007AFF   ← invisible glyph, 1:1
INACTIVE: color:#555;    background:#007AFF   ← ~2.6:1
```

Setting half a color pair is precisely what the emphasis axis exists to prevent.
`EmphasisGhost` punches the fill out instead of leaving it to whatever the theme
put there. The inactive tint also moved off the literal `#555` onto the
palette's `TextSecondary`, which needed a `ComponentFunc` wrapper — `TabButton`
returns a View before any render, so there is no theme to read until a Context
arrives. The now-unused local `ifThen` helper went with it.

## Contrast — where the widget stops promising

Filled owns both the fill and the label, so it contrast-picks the ink
(`Variant.Ink`) and is tested to clear WCAG AA on both bundled themes. Outlined
and Ghost own neither: the fill is transparent, so the label's real backdrop is
whatever the caller placed the button on, which the widget cannot see. Their
label is the variant's color **verbatim** — the theme author's hex, not a
synthesized shade.

Measured (not estimated — the first draft of this table was invented and wrong,
which is why it was computed and re-checked) against each theme's own
`Background`, both `#FFFFFF`:

|  | Default | Material |
|---|---|---|
| default | 4.02:1 | 7.63:1 |
| success | 2.22:1 | 5.13:1 |
| warning | 2.20:1 | 3.08:1 |
| error | 3.55:1 | 7.33:1 |

**Auto-darkening the role color until it passes was considered and rejected.**
It would fire on DefaultTheme's own brand blue (4.02:1 — the value Apple ships
and the one both themes pair with Button), i.e. the *default* case, and a widget
silently altering a hex the theme author chose is worse than a documented
number. The durable fix is a second palette value per role (an "on-light" tone,
as Material carries alongside each container color) — a palette decision, not a
Button one.

### The AA test excludes VariantDefault, and that is the finding

`TestButtonFilledStatusVariantsAreLegibleOnEveryTheme` covers success/warning/
error only. Including default fails:

```
Default/: ink "#FFFFFF" on "#007AFF" is 4.02:1, below WCAG AA
```

That pairing is **not the widget's** — Button applies no color props at all in
that case; it is DefaultTheme's own `Components.Button`. Failing it here would
report a palette decision as a widget defect, and "fixing" it would mean the
zero value silently repaints every button in every tree. Backlogged against the
theme instead. Badge's equivalent test exempts `VariantDefault` for the same
reason, so the two are consistent.

## Disabled

Shipped, with three deliberate choices:

- **The handler is replaced with a no-op, not dropped.** `core.Button` registers
  whatever it is given, and `registerVoid` stores a nil `func()` happily — then
  `TriggerCallback` finds it (`ok == true`) and calls it, panicking on the late
  native tap that races a purge. Verified: reverting to a pass-through nil
  panics the test.
- **One muted treatment for every variant** (Surface fill, TextSecondary ink). A
  disabled control has no meaning left to signal, and a dimmed red still reads
  as "danger" to a user who cannot tell it is inert.
- **The state rides the accessibility label** as `", disabled"`, following the
  `", selected"` convention Chip and ListRow already use, because no renderer
  carries a platform disabled state. When no explicit label was given the
  visible one is promoted, so the suffix has something to attach to. Accessibility
  props are appended *after* caller `Style`, so a caller cannot clobber the
  announcement — there is a test for that specifically.

`FullWidth` sets both `Width("100%")` and a block `Display`: both bundled themes
give Button an inline display, and width does nothing to an inline box. The
natives read `Display` only for `"hidden"`, so the block half is inert there and
the width alone does the work. The test skips itself if the theme base stops
being inline, so it cannot quietly assert a redundant thing.

## Verification

Every guard mutation-tested:

| mutation | caught by |
|---|---|
| ink → `t.Colors.Background` | 4 AA failures (2.22 / 2.20 / 3.55 / 3.08) |
| transparent fill → `""` | outlined + ghost fill assertions |
| nil handler passed through | `nil OnTap is also safe to dispatch` **panics** |
| zero value re-derives palette | the discriminating-theme test |

Rendered HTML diffed before/after for every migrated example:

- **chat** — colors identical; padding moves from a hand-typed 12 to the theme's
  10/16, which is the point.
- **fintech "Transfer"** — byte-identical (after finding 2).
- **fintech "Recharge"** — intentional: opaque `#FFF` fill becomes transparent
  and gains `border:1px solid #03DAC6`. A white fill on a white page was a fake
  transparent; this is the real one, and it is end-to-end proof of finding 4.
- **todoapp filter chips / Add** — identical.
- **todoapp delete `✕`** — **intentional appearance change**: white on the
  pinned `#B3261E` becomes contrast-picked black on the theme's Error `#FF3B30`.
  Both pairings clear AA (7.7:1 and 5.92:1). This is the point of moving off a
  literal — the app's destructive red is now the same red its error Badge would
  use — but it is visible, so it is called out in the code and here. The
  now-unused `colorDanger` constant was removed.
- **social tabs** — the invisible-glyph bug above.

`go build ./...`, `go vet ./...`, full `go test ./...` green.
`examples/todoapp/store.go` remains the only `gofmt` offender.

## Files touched

**New**
- `components/button.go` — `Button`, `Emphasis`, `ColorTransparent`
- `components/button_test.go` — 9 tests / 19 subtests

**Modified**
- `htmlout/export.go` + `_test.go` — border emission, both-halves guard
- `examples/chat/main.go` — four hand-spelled props → the zero value
- `examples/fintechapp/main.go` — `MaterialButton` helper deleted; theme gains
  a `Components.Button`; "Recharge" is the Secondary escape-hatch example
- `examples/todoapp/app.go` — Add / Clear completed / delete `✕`;
  `colorDanger` dropped
- `examples/social/tab.go` — rewritten as a ghost button, `ifThen` dropped
- `docs/components.md` — a full `## Button` section
- `docs/concepts/styling-and-theming.md` — the `ComponentDefaults` hazard
- `docs/index.md` — widget list and the role blurb

## Backlog after this session

- **Tier 1 is down to the `Screen` scaffold** (5 instances):
  `SafeArea > (Scroll?) > Column(FlexGrow(1), Gap(n))` at `chat/main.go:77`,
  `fintechapp/main.go:32`, `mobileapp/app.go:51`, `todoapp/app.go:188`,
  `layout/main.go:12`. Beyond deleting boilerplate it gives one place to hang
  the keyboard-aware scroll area the ROADMAP wants.
- **New, against the theme rather than a widget:** `DefaultTheme`'s
  `Components.Button` is white on `#007AFF` at **4.02:1**, below WCAG AA for
  17px text. Changing it repaints every button in every tree, so it is a
  deliberate palette decision, not a drive-by fix.
- **Also new:** a second palette value per status role (an "on-light" tone)
  would make Outlined/Ghost legible for Success and Warning. This is the same
  request as above from the other direction.
- **Tier 2:** `InputRow`/composer, `StatTile`, `EmptyState`, `SegmentedControl`.
- **`Variant` now has two consumers** (Badge, Button). `Alert`/banner is the
  obvious third and needs no new core work; `Chip{Variant:}` is the fourth —
  and Chip is now the odd one out, since it hand-rolls a selected treatment
  that `Emphasis` could express.
- **`MaybeProp` has one consumer** (`chat`). Button did not need it: its
  conditional props are style props resolved inside `colorProps`, not items in
  a container argument list.
- **A `Neutral`/muted variant** mapping to Surface is still the one addition a
  status set usually wants next — and `Disabled` now hand-rolls exactly that
  pair, so it would have a second caller immediately.
- **Renderer work, none of it blocking, unchanged:**
  1. Proportional flex weights on iOS (custom SwiftUI `Layout`) — lets
     `ProgressBar` move off percentage widths.
  2. `AlignItems: "stretch"` on both renderers — unblocks `Separator.Vertical`.
  3. A `ContentMode` prop on `Image` — unblocks avatar images that fill rather
     than letterbox.
  4. **New:** no renderer carries a disabled state. `Button.Disabled` is honest
     without it (no-op handler, muted fill, announced via the label), but a
     native `enabled=false` would get the platform's own press/focus behavior.
- **Still true from five sessions ago:** `ROADMAP.md` lists `UseMemo` and
  `UseReducer` as done; neither identifier exists in the tree.
