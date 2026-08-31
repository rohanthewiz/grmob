# Session: closing core gap 2, then giving the new roles a consumer

**Session ID:** session_01Ej9aSqY1yefk9am5LZG367
**Date:** 2026-08-31, ~14:45
**Branch:** master
**Commits:** `93f1b01` (palette roles), `34c8637` (Badge.Variant)

## Goal

Two requests in sequence. First: close core gap 2 from the gap analysis —
`ColorPalette` has no `Border`, `Success` or `Warning`. Then, once that
landed: `Badge{Variant: Success|Warning|Error}`, the widget the new roles
were gating.

## Part 1 — the palette roles (commit `93f1b01`)

### What was missing and why it mattered

`ColorPalette` carried seven roles: Primary, Secondary, Background, Surface,
TextPrimary, TextSecondary, Error. The absence showed up as two concrete
symptoms already in the tree:

- `components/separator.go` pinned its hairline as a package constant, with a
  doc comment explaining it as temporary until a `Border` role existed.
  `examples/todoapp` had the same value again as its own `colorHair`.
- `examples/fintechapp` used `t.Colors.Secondary` to mean "money in".

Values chosen:

| role | DefaultTheme | MaterialTheme |
|---|---|---|
| `Border` | `#E5E5EA` (iOS systemGray5) | `#E0E0E0` (MD grey 300) |
| `Success` | `#34C759` | `#2E7D32` (green 800) |
| `Warning` | `#FF9500` | `#EF6C00` (orange 800) |

`Border` deliberately keeps `#E5E5EA` rather than iOS's truer separator
`#C6C6C8`: it is the value both examples had independently picked, so nothing
changes appearance. MaterialTheme's values stay inside the official Material
palette, matching the theme's existing `#6200EE` / `#03DAC6` / `#B00020`.

### The one design decision: resolvers for three, not for ten

The new three get `BorderColor()` / `SuccessColor()` / `WarningColor()`; the
original seven deliberately get nothing.

The reason is compatibility, and it is specific to *when* a role was added. A
theme written before these roles existed leaves them empty, and an empty color
is not "the default" — it is *no color*, which renders as an invisible rule or
a transparent chip. The resolvers fall back to `FallbackBorder` /
`FallbackSuccess` / `FallbackWarning`, which are DefaultTheme's own values, so
a stale theme looks like the default theme in that one place instead of
disappearing.

The original seven predate every theme in existence, bundled or user-written,
so none can be missing one. Resolvers there would be API surface for nothing.
`examples/fintechapp` was the live proof that the hazard is real: a
hand-written seven-role palette with no `Components` block at all.

`VariantError` later reads `t.Colors.Error` *directly* for exactly this
reason — it is one of the original seven.

### Two distinctions the role names do not make obvious

Both are documented on the struct and in the styling doc, because both are the
kind of thing a later reader would "simplify" away:

**`Border` is not `Surface`.** Surface is a *fill*. On a light theme the two
are near neighbors, so a Surface-tinted hairline drawn on a Surface panel is
invisible. This was already written down in the old Separator comment as the
reason the constant existed.

**`Success` is not `Secondary`,** even though DefaultTheme happens to tint
both `#34C759`. Secondary is a brand slot a theme may set to anything —
MaterialTheme makes it teal — while Success carries meaning. A magenta "saved"
badge is a bug. There is a test asserting the two differ under MaterialTheme
specifically so the coincidence under DefaultTheme cannot mislead.

`components/list_row.go` had a stale comment saying the palette "has no
dedicated Border/Selected entry yet". Corrected rather than acted on: Surface
is *still* the right tint for a selected row, because Border is a stroke role
and there is no Selected role.

### The regression guard

`core/theme_test.go`, reflective for the same reason `style_merge_test.go` is:
the failure mode is **growth**. A role added to the struct and wired into
DefaultTheme but forgotten in MaterialTheme produces no compile error and no
runtime error — a struct literal is happy to leave a field zero — it just
renders that role as an empty color under that one theme.

Verified it bites by adding a temporary `Info` field:

```
--- FAIL: TestBundledThemesSetEveryColorRole
    theme_test.go:32: MaterialTheme.Colors.Info is empty: ...
    theme_test.go:32: DefaultTheme.Colors.Info is empty: ...
```

Plus `TestFallbacksTrackDefaultTheme`, which guards a promise that would
otherwise rot silently and only under a third-party theme: the fallback
constants are documented as DefaultTheme's own values, so retinting the theme
without the constants breaks that where nobody looks.

### Consumers wired

- `Separator` reads `ctx.Theme().Colors.BorderColor()`; its `hairlineColor`
  constant is gone.
- `todoapp` dropped `colorHair` and renders `components.Separator{}` —
  byte-identical output, but now retintable.
- `fintechapp` gained the three roles and moved credits off Secondary.

## Part 2 — `Badge.Variant` (commit `34c8637`)

### Shape

`components/variant.go` is new. `Variant` is a **package-level** string enum,
not Badge's own type, so a future Alert or banner resolves the same four roles
identically, and a caller can pass one value around.

```go
type Variant string
const (
    VariantDefault Variant = ""        // zero value
    VariantSuccess Variant = "success"
    VariantWarning Variant = "warning"
    VariantError   Variant = "error"
)
```

String-typed with an empty zero value, matching core's `Alignment` and
`DisplayMode`. That is load-bearing, not just idiom: the zero value has to be
the existing look or adding the field would restyle every Badge already in a
tree. `TestBadgeThemeDefaults` predates the field and still passes untouched,
which is the real proof.

`Variant.Color(theme)` and `Variant.Ink(theme, bg)` are exported so a
hand-built status surface gets the same mapping.

### The ink has to be computed — this is the finding of the session

The obvious implementation reuses Badge's existing ink default (the theme's
`Background`, i.e. white, chosen to read on Primary). That would have shipped
this, on the framework's own default theme:

| DefaultTheme fill | white ink | dark ink |
|---|---|---|
| Success `#34C759` | **2.22:1** | 9.46:1 |
| Warning `#FF9500` | **2.20:1** | 9.55:1 |
| Error `#FF3B30` | **3.55:1** | 5.92:1 |

Badge text is caption-sized (13px / 12px), so the bar is WCAG AA at 4.5:1.
Two of those are below even the 3:1 large-text floor — a badge nobody can
read, with nothing in the tree to suggest a bug.

**A fixed per-variant pairing does not fix it either**, which is the part
worth remembering. The correct ink *flips direction between the bundled
themes for the same variant*: DefaultTheme's Success is a light green wanting
dark ink, MaterialTheme's is a dark green wanting light ink. Any hardcoded
pairing is wrong under one theme or the other.

So `Variant.Ink` picks, per color at render time, whichever of the theme's two
ink roles (`Background` / `TextPrimary`) has more WCAG contrast against the
resolved fill.

**`VariantDefault` is exempt** and keeps the theme's `Background`. That is the
pairing both themes chose for Primary and use for Button, and preserving it is
what makes the zero value a no-op. The exemption is *observable* — on
`#007AFF` white is 4.02:1 and black 5.23:1, so the contrast rule would flip it
— which is why it gets its own test with a self-guard.

The end-to-end HTML export, all four variants under both themes:

```
Default:  #FFFFFF/#007AFF · #000000/#34C759 · #000000/#FF9500 · #000000/#FF3B30
Material: #FFFFFF/#6200EE · #FFFFFF/#2E7D32 · #212121/#EF6C00 · #FFFFFF/#B00020
```

Material's Warning is the only one that flips to dark ink — the single case a
"just use white on Material" shortcut would have gotten wrong.

### Contrast helpers

`contrastInk` / `contrastRatio` / `relativeLuminance`, WCAG 2.x with proper
sRGB linearization. The linearization is not optional: a naive channel mean
puts `#007AFF` at ~0.50 instead of 0.211, which is the classic mistake that
makes mid-tones read as far brighter than they are.

- Parses `#RGB`, `#RRGGBB`, `#RRGGBBAA`, trimmed and case-insensitive.
- **Alpha is parsed and ignored**, documented: compositing needs the backdrop,
  and a widget resolving its own ink does not know what it will be drawn over.
  A translucent fill therefore reads as its opaque form, overestimating
  contrast — acceptable, since every palette *fill* role is opaque and the one
  translucent value in the bundled themes (`TextSecondary`) is an ink.
- Unparseable input degrades to the caller's **first** candidate, which is why
  `Ink` passes `Background` first: a bad color falls back to pre-variant
  behavior rather than something arbitrary. Palette values come from user code
  and are never validated.

### Precedence

Explicit `Color` beats `Variant` — but the ink is still resolved against
whichever fill *won*, so an override cannot silently produce an illegible
pill. Explicit `TextColor` beats the computed ink.

### No accessibility synthesis, unlike Avatar

Avatar synthesizes a label because it has exactly one thing to say and `Name`
is it. A badge's meaning is already in its `Text`. The variant is WCAG 1.4.1
*reinforcement* — nothing announces "warning" to a screen reader, and a reader
who cannot distinguish the tints sees only the label, so the label has to say
it ("Overdue", not "!"). Documented on the type rather than coded.

### The legibility guard

Asserts every variant x both themes clears 4.5:1. Verified it bites by
reverting `Ink` to the naive version:

```
--- FAIL: TestVariantInkIsLegibleOnEveryThemeAndVariant
    DefaultTheme/success: ink "#FFFFFF" on "#34C759" is 2.22:1, below WCAG AA
    DefaultTheme/warning: ink "#FFFFFF" on "#FF9500" is 2.20:1, below WCAG AA
    DefaultTheme/error:   ink "#FFFFFF" on "#FF3B30" is 3.55:1, below WCAG AA
    MaterialTheme/warning: ink "#FFFFFF" on "#EF6C00" is 3.08:1, below WCAG AA
```

Two tests carry explicit self-guards (`TestVariantDefaultKeepsTheThemePairing`,
`TestSeparatorTakesTintFromTheme`) that fail if the themes ever converge, so
they cannot start passing vacuously.

### Example migration

`fintechapp`'s `TransactionItem(label, amount, color, ink)` became
`TransactionItem(label, amount, variant)`. It had been passing a background
*and* a hand-picked matching ink — exactly the two obligations the variant
removes from every call site. Rendered appearance is unchanged (its dark green
and dark red both still take white).

## Committing as two

The two tasks shared two files — `examples/fintechapp/main.go` and
`docs/components.md`. Rather than fold the variant migration into the palette
commit, those two were rolled back to their gap-2-only state for `93f1b01` and
restored for `34c8637`.

To confirm the split was clean rather than merely plausible, `93f1b01` was
checked out into a throwaway `git worktree` and built + vetted + fully tested
there. Green, so the first commit stands on its own rather than only compiling
in the presence of the second. Worth repeating whenever a commit is split by
hand.

## Verification

`go build ./...`, `go vet ./...`, full `go test ./...` green throughout both
parts. `components/badge.go` was one of the two long-standing `gofmt`
offenders and got formatted while being edited, so **only
`examples/todoapp/store.go` remains** on that list.

## Files touched

**New**
- `core/theme_test.go` (5 tests)
- `components/variant.go` + `_test.go` (8 tests)

**Modified**
- `core/theme.go` — three roles, three fallback constants, three resolvers,
  both bundled palettes
- `components/badge.go` + `_test.go` — the `Variant` field (6 new tests)
- `components/separator.go` + `_test.go` — themed tint (3 new tests)
- `components/list_row.go` — stale palette comment corrected
- `examples/todoapp/app.go` — `colorHair` dropped
- `examples/fintechapp/main.go` — roles added, then `TransactionItem`
  migrated to `Variant`
- `docs/components.md`, `docs/concepts/styling-and-theming.md`,
  `docs/index.md`

## Backlog after this session

- **Core gap 3** — the `[]core.PropsAndChildren` append idiom — still open at
  `chat/main.go:147` and `mobileapp/app.go:146`. A `core.MaybeProp(cond, prop)`
  deletes it. This is now the last of the three core gaps.
- **Tier 1 widgets left:** `Button` variants (5 instances), `Screen` scaffold
  (5). `ListRow` and `Separator` are done.
- **Tier 2:** `InputRow`/composer, `StatTile`, `EmptyState`,
  `SegmentedControl`.
- **`Variant` now has one consumer.** `Alert`/banner is the obvious second,
  and it needs no new core work — it can call `Variant.Color` and
  `Variant.Ink` exactly as Badge does. A `Chip{Variant:}` is the third.
- **A `Neutral`/muted variant** was considered and left out as out of scope.
  It would map to Surface and is the one addition a status set usually wants
  next.
- **Renderer work, none of it blocking, unchanged from last session:**
  1. Proportional flex weights on iOS (custom SwiftUI `Layout`) — lets
     `ProgressBar` move off percentage widths.
  2. `AlignItems: "stretch"` on both renderers — unblocks
     `Separator.Vertical`.
  3. A `ContentMode` prop on `Image` — unblocks avatar images that fill
     rather than letterbox.
- **Still true from three sessions ago:** `ROADMAP.md` lists `UseMemo` and
  `UseReducer` as done; neither identifier exists in the tree.
