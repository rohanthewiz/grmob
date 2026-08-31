# Session: Component gap analysis — what widgets to add next

**Session ID:** session_01UiLqZqPu2rv9U7cYuJVpWb
**Date:** 2026-08-31, ~00:58
**Branch:** master

## Goal

Answer a single question: *what low-hanging fruit, in terms of components, can
be added to the framework?* No code was written — this is a survey of the
current tree plus an evidence scan of the example apps, producing a ranked
backlog. Recording it so the next session can pick an item and build rather
than re-derive the ranking.

## What "low-hanging" was taken to mean

A widget is low-hanging here iff it can be built **purely from the existing
core primitives**, requiring **zero renderer work**. That constraint is real,
not stylistic: the node vocabulary is a wire contract duplicated across three
independent renderers, verified in this session at

- `htmlout/export.go:73-152` (+ `tagForType`)
- `android/app/src/main/java/com/grmob/runtime/Renderer.kt:88-120`
- `ios/GrMob/Runtime/Renderer.swift:63-118`

The shared vocabulary is:

```
Text  Button  Input  InputPassword  NumericInput  TextArea  Checkbox
Row   Column  Card   Box  List  Spacer  Scroll  SafeArea
TabView  Modal  Image  CameraView  Fragment  Theme
```

Anything needing a *new* node type (a real `Switch`, `Slider`, `Picker`,
`DatePicker`, pull-to-refresh, swipe gestures) costs three renderer edits plus
a `styleValue`/`GrMobStyle` round — explicitly **not** low-hanging, and left
out of the ranking below.

## State of the tree as found

`components/` currently holds five widgets, all from Workstream 3 of
`ai_docs/plans/element-lessons-adoption.md`: `Card`, `Badge`, `Chip`,
`FormField`, `Tabs`, `Accordion` (six, counting Accordion — the only one that
owns state). `core/` still holds the three pre-existing widgets the plan said
to leave alone: `modal.go`, `toast.go`, `tabview.go`.

## Method

A subagent read all seven example files (`chat`, `fintechapp`, `mobileapp`,
`social/{app,pages,tab}`, `todoapp`, `layout`) looking only for *recurring
hand-rolled shapes* — no design proposals, just counts and `file:line`. The
ranking below is that count, filtered by the renderer constraint above.

## The ranked backlog

### Tier 1 — already hand-rolled 4-5× each

**1. `ListRow` — the biggest single win (5 instances).**
`fintechapp/main.go:137`, `todoapp/app.go:301`, `todoapp/app.go:243`,
`mobileapp/app.go:146`, `mobileapp/app.go:193`.

Every instance is *leading control → flexible title → trailing action*, and
they **disagree on the mechanism**: fintech pins the trailing element with
`Justify(JustifyBetween)`; todoapp uses `FlexGrow(1)` on the middle `Text`
(twice). Two of them also hand-build the accessibility suffix convention
(`", completed"` at `todoapp/app.go:335-340`, `", selected"` at
`mobileapp/app.go:157`) — the exact convention `components.Chip` already owns
internally. A widget settles all three disagreements in one place.

```go
type ListRow struct {
    Leading  core.View   // checkbox, icon, avatar
    Title    string
    Subtitle string
    Content  core.View   // overrides Title/Subtitle — the Card.Header idiom
    Trailing core.View   // badge, value text, delete button
    OnTap    func()
    Selected bool
    Style    []core.StyleProp
    AccessibilityLabel, AccessibilityHint string // ", selected" appended, like Chip
}
```

**2. `Button` variants (5 instances).**
`chat/main.go:178`, `fintechapp/main.go:95`, `todoapp/app.go:253`,
`todoapp/app.go:326`, `social/tab.go:8`.

`core.Button` offers exactly one look, so every app re-spells
bg + fg + padding + radius by hand. `fintechapp`'s local `MaterialButton`
helper *is* this widget already. The two todoapp destructive buttons are the
same color pair written two different ways, each carrying a near-duplicate
comment explaining why the theme base is overridden.

```go
type Button struct {
    Label string; OnTap func()
    Variant  Variant // Primary (default) | Secondary | Danger | Ghost
    FullWidth, Disabled bool
    Style []core.StyleProp
}
```

**3. `Screen` scaffold (5 instances).**
`SafeArea > (Scroll?) > Column(FlexGrow(1), Gap(n))` at `chat/main.go:77`,
`fintechapp/main.go:32`, `mobileapp/app.go:51`, `todoapp/app.go:188`,
`layout/main.go:12`. Beyond deleting boilerplate, it gives one place to later
hang the keyboard-aware scroll area the ROADMAP wants.

**4. `Separator` (2 byte-identical instances).**
`todoapp/app.go:216`, `mobileapp/app.go:133` — both
`Box(Height("1px"), BackgroundColor("#E5E5EA"), AccessibilityHidden())`, with
the hairline color hardcoded in two separate packages. `core.Divider(h, color)`
exists (`core/layout.go`) but takes a literal color and force-applies
`Margin(8)`, which is presumably why neither example uses it. A themed
replacement is ~8 lines.

### Tier 2 — 2-3 instances, still free

| Widget | Instances |
|---|---|
| `InputRow` / composer — `Row(Gap, InputWithSubmit(…FlexGrow(1)), Button)` | `chat/main.go:168`, `todoapp/app.go:201` (near-identical) |
| `StatTile{Caption, Value, Delta}` | `fintechapp/main.go:70`, `mobileapp/app.go:85`, `todoapp/app.go:245` |
| `EmptyState{Icon, Title, Message, Action}` | `todoapp/app.go:173`+`:228`; near-miss `mobileapp/app.go:114` |
| `ToggleRow{Label, Checked, OnChange}` | `mobileapp/app.go:193`, `todoapp/app.go:318` |
| `SegmentedControl` (a `For` over `Chip`, controlled index) | `fintechapp/main.go:83`, `todoapp/app.go:271` (already on Chip), `social/app.go:31` |

`SegmentedControl` additionally retires `social`'s hand-rolled string-keyed tab
bar (`social/app.go:25-37` + `social/tab.go`), which diverges from
`mobileapp`'s int-keyed `core.TabView` for no reason.

### Tier 3 — conventional, but zero evidence in-tree

`Avatar`, `ProgressBar`, `Alert`/`Banner`, `Skeleton`, and a `Dialog` struct
facade over `core.Modal` (the same wrap-don't-supersede move `Tabs` made over
`TabView`). All are pure composition, but the scan found **zero** avatar or
progress usage — the only `Image` in the tree is `fintechapp/main.go:57`, with
no border-radius. Recommendation: hold until an example needs one.

One implementation note if `ProgressBar` is built: use two `Box`es with
`FlexGrow(v)` / `FlexGrow(1-v)`, **not** percentage widths. Percent widths do
work on both natives (`GrMobStyle.kt:272`, `GrMobStyle.swift:353`) but iOS's
path is an explicit approximation via `containerRelativeFrame` — the comment
there says so.

## Three core gaps that gate the backlog

**1. `UseStyle` silently drops most of `Style`.** `core/style.go:82` merges
only FontSize, FontWeight, TextColor, Background, BorderRadius, Shadow, Align,
Display, Padding, Margin, Bottom/Left/Right, ZIndex. Thrown away: **Width,
Height, Min/Max\*, BorderColor, BorderWidth, Gap, every flex field, Position,
Top, Overflow, WhiteSpace, LineHeight, Transition, Animation, and all three
accessibility fields.** `Style.With` inherits the hole verbatim
(`core/style_props.go:240` is a two-line wrapper over `UseStyle`).

This is the highest-value fix in the list: any widget that sizes itself —
Avatar, ProgressBar, Separator — fails confusingly today, and the failure is
silent. It gates a whole class of widgets, not one.

**2. The palette has no `Border`, `Success`, or `Warning`.** `ColorPalette`
(`core/theme.go`) carries only Primary, Secondary, Background, Surface,
TextPrimary, TextSecondary, Error. That absence is *why* the hairline color is
hardcoded in two packages, and it's what a themed `Separator`, `Alert`, or a
`Badge{Variant:}` would need.

**3. The `[]core.PropsAndChildren`-append idiom.** Appears at
`chat/main.go:147` and `mobileapp/app.go:146`, both with comments explaining
it as a workaround for `core.If` emitting a real child node. A
`core.MaybeProp(cond, prop)` — or a nil-returning `If` — deletes the workaround
from both call sites and from every widget in the backlog above.

## Incidental finding

`ROADMAP.md` lists `UseMemo` and `UseReducer` under "✅ Done → Hooks". Neither
identifier exists anywhere in the tree:

```sh
grep -rn "UseMemo\|UseReducer" --include='*.go' .   # no matches
```

`hooks/` actually contains `UseInterval`, `UseTimeout` (`hooks/interval.go`)
and `UseEffect` (`hooks/effect.go`). Either build them or correct the roadmap.

## Recommendation

Start with **`ListRow`**. It is the most-repeated shape, it settles the
`Justify` vs `FlexGrow` split that currently splits the examples, and it
centralizes the accessibility-suffix convention that is copy-pasted in two
places. Pair it with the `UseStyle` fix (gap 1) if the row is to accept sized
leading slots.

Per `components/doc.go`, whatever lands next: struct with named fields,
`core.View` slots, theme-driven colors and spacing, a `Style []core.StyleProp`
override, a focused test — and if extracted from an example, pin the rendered
output against the original with `htmlout.ExportHTML` (the pattern
`examples/todoapp/chip_migration_test.go` established).

## Files touched

None. Analysis only.
