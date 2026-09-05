# Widget Library — the `components` package

`components` is GrMob's higher-level widget library, built **entirely on the
public core API** — a deliberate dogfooding discipline: if a widget can't be
built out here, that is a gap in core's primitives, not a reason to reach
inside.

```go
import "github.com/rohanthewiz/grmob/components"
```

## The struct-widget idiom

Every widget is a struct implementing `core.View`, configured through named
fields:

```go
components.Card{
    Title:  "Account",
    Body:   balanceSummary,
    Footer: components.Badge{Text: "verified"},
}
```

Structs — rather than more constructor functions in core — for two reasons:

- **Named fields scale.** A widget can grow an optional knob without
  breaking a single call site, where positional arguments cannot.
- **A `core.View`-typed field is a composition slot.** `Card.Header`,
  `FormField.Input`, `Accordion.Content` accept any view. Where a widget
  offers a simple path *and* a slot (`Card.Title` vs `Card.Header`), the
  slot wins when both are set.

All widgets take their look from `ctx.Theme()` — palette colors, spacing and
typography scales, never hard-coded values — and accept `Style` overrides
for per-use adjustment.

## Screen

The root scaffold: safe-area inset, an optional scroll region, and the
vertical column that holds the screen's content.

```
SafeArea
  └─ Scroll            (only when Scroll is true; KeyboardAware lands here)
       └─ Column       ← Gap / Fill / Style land here
                         (and KeyboardAware, when there is no Scroll)
            ├─ Children[0]
            └─ …
```

```go
// The zero value: SafeArea(Column(children...)), nothing else.
components.Screen{
    Children: []core.View{header, body, composer},
}

// A screen that scrolls as a whole.
components.Screen{
    Scroll:   true,
    Children: []core.View{hero, section1, section2},
}

// A screen whose list fills the space and pushes a footer down.
components.Screen{
    Fill: true,
    Gap:  12,
    Children: []core.View{title, entryRow, list, footer},
}
```

| field | effect |
|---|---|
| `Children` | laid out top to bottom in the column; a `nil` entry is skipped |
| `Scroll` | wraps the column in `core.Scroll` |
| `KeyboardAware` | the region yields to the software keyboard instead of being covered |
| `Gap` | uniform vertical spacing, in points; zero means *unset* |
| `Fill` | `FlexGrow(1)` on the column — claim the full safe-area height |
| `Style` | applied to the column **after** `Gap` and `Fill`, so it overrides both |

**Every field defaults to contributing nothing**, so the zero value renders
the bare scaffold with no style props at all and the theme's `Column` base
carries through untouched. That is what let all five migrations stay
byte-identical. It matters specifically for `Gap`: style props *set* rather
than merge, so an unconditional `core.Gap(0)` would overwrite a gap the
theme's `Column` base had set. Unset and zero therefore have to mean the same
thing — the absence of a gap, not the imposition of one.

**`Scroll` is for screens with no scrolling region inside them.** Leave it
false when the screen already contains a `core.List` or its own `Scroll` — a
scroll view nested in a scroll view fights for the same drag on both natives.
Of the nine app packages in `examples/`, two scroll at the root
(`fintechapp`, `signup`), and every screen the `tutorial`'s navigator pushes
(`home`, `lesson_screen`, chapter 6) scrolls as a whole. The rest do not:
`chat` scrolls its message list, `todoapp` scrolls a virtualized `core.List`,
and `mobileapp` and `layout` are short enough to need neither.

**`KeyboardAware` lands on the scroll region, or on the column when there is
none** — the two halves of what it means. A scrolling screen wants its
*viewport* shortened, so the platform's scroll-the-focused-field-into-view has
somewhere visible to put the field; a fixed screen wants the whole column
lifted, so whatever is docked at its bottom — a chat composer, a checkout bar,
the thing the keyboard actually covers — stays reachable. Either way the
`SafeArea` itself stays put, so a header does not slide off the top. See
[`core.KeyboardAware`](concepts/views.md#the-software-keyboard) for what each
platform does with it.

```go
// A form: the viewport ends where the keyboard begins.
components.Screen{Scroll: true, KeyboardAware: true, Children: fields}

// A chat: no scroll here, so the column lifts and the composer rides up.
components.Screen{KeyboardAware: true, Children: []core.View{header, thread, composer}}
```

**`Fill` is load-bearing wherever a child grows.** A `FlexGrow` child can only
grow inside a parent that has height to give, so a screen whose list should
expand into the leftover space needs `Fill: true` on the scaffold —
`examples/todoapp` is the worked example. `Fill` with `Scroll` is legal but
unusual: it makes the scrolled content at least as tall as the viewport, which
is how you bottom-anchor a footer on a short page. It does not make a scroll
view fill anything.

Because `Children` flows into `core.Column`'s argument list, which skips nil
items, the conditional-region idiom costs the tree nothing when the condition
is false:

```go
var banner core.View
if offline {
    banner = OfflineBanner()
}
components.Screen{Children: []core.View{banner, body}}
```

That is the same contract behind [`core.MaybeProp`](concepts/views.md) — no
node, no flex slot, no stray `Gap` — where a `core.If` would leave an empty
`Fragment` for the column to space against.

## Button

A themed action button with **two orthogonal color axes** and no per-call hex.

```go
components.Button{Label: "Save",   OnTap: save}                                   // theme Button base
components.Button{Label: "Delete", OnTap: rm,   Variant: components.VariantError}
components.Button{Label: "Cancel", OnTap: back, Emphasis: components.EmphasisOutlined}
components.Button{Label: "Skip",   OnTap: skip, Emphasis: components.EmphasisGhost}
```

`Variant` says **which** color (the meaning) and is the same enum
[Badge](#variants) uses, so a danger button and an error badge are the same red
by construction. `Emphasis` says **how much** of it:

| emphasis | fill | label | rule |
|---|---|---|---|
| `EmphasisFilled` (zero value) | the variant's color | contrast-picked against the fill | — |
| `EmphasisOutlined` | transparent | the variant's color | 1px, the variant's color |
| `EmphasisGhost` | transparent | the variant's color | — |

!!! note "Why two fields instead of one `Primary | Secondary | Danger | Ghost`"
    A flat enum conflates the two questions, so it cannot express an outlined
    destructive button — the ordinary shape of a "Delete" confirmation —
    without a fifth value, and then a ghost destructive needs a sixth.

**The zero value applies nothing.** `Button{Label: l, OnTap: f}` renders
exactly `core.Button(l, f)` — the theme's own `Components.Button` carries the
look through untouched, rather than being re-derived from the palette. A theme
whose Button base deliberately differs from `Colors.Primary` keeps that choice.

The flip side: a theme with *no* `Components.Button` gets a button with no
styling. That is not new — `core.Button` has always behaved that way — but the
widget makes it visible, which is how `examples/fintechapp` turned out to have
no `Components` block at all.

### Secondary is deliberately not a variant

`Variant` carries *meaning*; `Secondary` is a brand slot a theme may set to
anything. `Style` is the escape hatch for the brand case:

```go
components.Button{
    Label: "Recharge", Emphasis: components.EmphasisOutlined,
    Style: []core.StyleProp{
        core.TextColor(t.Colors.Secondary),
        core.BorderColor(t.Colors.Secondary),
    },
}
```

### Contrast, and what the widget can promise

`EmphasisFilled` owns both the fill and the label, so it picks the label by
contrast and is tested to clear WCAG AA under both bundled themes.

Outlined and Ghost own neither — the fill is transparent, so the label's real
backdrop is whatever you placed the button on. Their label is the variant's
color **verbatim**. Measured against each theme's own `Background` (both
`#FFFFFF`):

| variant | DefaultTheme | MaterialTheme |
|---|---|---|
| default | 4.02:1 | 7.63:1 |
| success | 2.22:1 | 5.13:1 |
| warning | 2.20:1 | 3.08:1 |
| error | 3.55:1 | 7.33:1 |

Prefer `EmphasisFilled` for a status action; override `TextColor` where those
numbers do not hold. Darkening the role color until it passes was rejected — it
would repaint DefaultTheme's own brand blue, i.e. the default case. The durable
fix is a second palette value per role, which is a palette decision.

### FullWidth and Disabled

`FullWidth` sets both a `100%` width and a **block** display: the bundled
themes give Button an inline display, and width does nothing to an inline box.

`Disabled` does three separate things, and all three are needed. It renders
the palette's muted pair (Surface fill, TextSecondary ink); it replaces the
handler with a no-op (dropping it would leave a nil func in the callback
registry for a tap that races the disabling patch to panic on); and it sets
[`core.Disabled`](concepts/styling-and-theming.md#disabled), which is what
makes the *platform* refuse to dispatch and announce the state to a screen
reader.

That last part is why the accessibility label is left alone. The widget used
to append `", disabled"` to it — the `", selected"` convention Chip and
ListRow still use — because no renderer carried a disabled state. Now that
they all do, appending it as well would announce the state twice.

## InputRow

The composer: a text field that fills the row, and an optional trailing button
that commits it.

```
Row (Gap)
  ├─ Input   ← FlexGrow(1), value / placeholder / onChange / onSubmit
  └─ Button  ← only when Button.Label is set; OnTap defaults to OnSubmit
```

```go
// A field and a Send button, one commit action.
components.InputRow{
    Value:       draft.Get(),
    Placeholder: "What needs doing?",
    OnChange:    func(v string) { draft.Set(v) },
    OnSubmit:    addTodo,
    Button:      components.Button{Label: "Add"},
}

// A search field with no button: the return key commits it.
components.InputRow{
    Value:       query.Get(),
    Placeholder: "Search",
    OnChange:    func(v string) { query.Set(v) },
    OnSubmit:    runSearch,
}

// A docked composer, with the bar treatment this widget has no opinion about.
components.InputRow{
    Value: draft.Get(), Placeholder: "Mensagem…",
    OnChange: func(v string) { draft.Set(v) },
    OnSubmit: send,
    Button:   components.Button{Label: "Enviar"},
    Style: []core.StyleProp{
        core.BackgroundColor("#FFFFFF"),
        core.Padding(12),
    },
}
```

| field | effect |
|---|---|
| `Value` | the field's text — fully controlled, so `OnChange` is the only way it changes |
| `Placeholder` | empty-state text; the field's only label |
| `OnChange` | fires on every keystroke |
| `OnSubmit` | the commit action: keyboard return/done **and** the button's tap |
| `Button` | trailing commit button; the zero value renders **no node at all** |
| `Gap` | horizontal spacing, in points; zero means the theme's `SM` step |
| `Style` | applied to the row **after** `Gap`, so it overrides it |

**One commit action, three paths.** The button *is* the field's submit,
rendered as a tap target for the case where the keyboard's return key is not
obvious or not reachable — so `OnSubmit` drives both and the two cannot drift
apart. Both hand-written call sites this replaced named the same helper twice.
Setting `Button.OnTap` explicitly still wins, but it has to be said out loud.

**`Gap` defaults to the theme's step, unlike `Screen`'s.** `Screen` treats a
zero `Gap` as *unset*, because the spacing between a screen's sections is the
app's decision. The gap here is the opposite kind of thing — it is the widget's
internal layout, and the field and the button must not touch — so `InputRow`
owns it the way `FormField` owns the spacing between its label and its input.
Zero means the theme's `SM` step (8pt in both bundled themes). To ask for no
gap at all, say it through `Style`:

```go
Style: []core.StyleProp{core.Gap(0)}
```

**A nil `OnSubmit` builds the field without a submit path**, rather than with
one wired to a no-op: the renderers read the `onSubmit` prop to decide whether
to show a submit affordance on the keyboard, and a registered no-op would
advertise an action the row ignores.

**The input is owned, not slotted.** Unlike [`FormField`](#formfield), which
takes whatever input it is given, `InputRow` builds its own — the wiring *is*
the widget, and a slot would hand it back to the caller. So the field itself
takes no per-call styling; a composer that needs to restyle its input has
outgrown this and should go back to `core.Row` + `core.InputWithSubmit`. Wrap
the row in a `FormField` when it needs a caption or an error line.

## Card

A surface with optional header, body, and footer regions, on the theme's
Card base (background, padding, radius, shadow).

```go
components.Card{
    Title: "Recent activity",        // simple path: themed bold subtitle
    Body:  activityList,
    Footer: core.Text("Updated 2m ago"),
}

components.Card{
    Header: customHeaderRow,          // escape hatch — overrides Title
    Body:   content,
    Style:  []core.StyleProp{core.Margin(0)},
}
```

## ListRow

The leading-control / flexible-title / trailing-action shape every list
needs: a checkbox and a task with a delete button, an avatar and a name with
a chevron, a label and an amount.

```go
components.ListRow{
    Leading:  core.Checkbox(t.Done, func(v bool) { setDone(id, v) }),
    Title:    t.Title,
    Subtitle: "Due today",
    Trailing: core.Button("✕", func() { remove(id) }),
    OnTap:    func() { open(id) },

    Selected:           id == selectedID,
    AccessibilityLabel: t.Title,
    AccessibilityHint:  "Opens the task",
}
```

**How the trailing slot gets pinned.** The examples that hand-rolled this
shape disagreed: some used `Justify(JustifyBetween)` on the row, others
`FlexGrow(1)` on the middle text. They are not equivalent —
`JustifyBetween` distributes slack between *every* pair of children, so a
row with no trailing slot pushes leading and title apart. ListRow settles it
on `FlexGrow`: a middle column takes all the slack, so leading and trailing
sit hard against the row's edges in every configuration.

```
┌ Row ─────────────────────────────────────────────────────────┐
│ [Leading]  ┌ Column FlexGrow(1) ────────────┐    [Trailing]  │
│            │ Title                          │                │
│            │ Subtitle                       │                │
│            └────────────────────────────────┘                │
└──────────────────────────────────────────────────────────────┘
            └──────── takes all the slack ────┘
```

The middle column is rendered **even when empty** — unlike `Card`, which
omits empty regions. Here the middle is structure, not content: making it
conditional would make the pinning conditional too.

Other notes:

- `Content` is the escape hatch for the middle, overriding `Title`/
  `Subtitle` — the same simple-path-plus-slot idiom as `Card.Header`.
- `OnTap` / `OnLongPress` make the whole row a target and are wired only
  when non-nil, so a presentational row registers no callback at all. Both
  may be set at once; the renderers wire them as one gesture recognizer, so
  a long press never also fires the tap.
- `Selected` tints the row with the theme's `Surface` (override with
  `SelectedStyle`) and appends `", selected"` to the accessibility label —
  the same convention Chip owns.
- No label is synthesized from `Title`. A row is a compound control whose
  slots carry meaning the widget cannot see, and labelling the container
  overrides how its children are announced, so naming the row is the
  caller's call.

All five hand-rolled instances in the examples are now built on it:
`todoapp`'s task row and footer, `mobileapp`'s feed row and subscribe
toggle, and `fintechapp`'s transaction row. Two of them lost workarounds in
the process — `mobileapp`'s feed row no longer appends to a
`[]core.PropsAndChildren` to add a conditional style prop (`SelectedStyle`
is that conditional, declared), and `todoapp`'s footer no longer wraps its
bulk-clear button in `core.If` (a nil `Trailing` emits no node at all).

## Badge

A small **non-interactive** status pill — a count, a "verified" mark, a
state label. Defaults: theme Primary background, theme Background ink,
caption-sized, stadium-shaped.

```go
components.Badge{Text: "3"}
components.Badge{Text: "beta", Color: "#7B1FA2"}
```

### Variants

`Variant` names what the badge *means* and takes its colors from the
palette's status roles, so a status pill carries no literal hex:

```go
components.Badge{Text: "Paid",     Variant: components.VariantSuccess}
components.Badge{Text: "Expiring", Variant: components.VariantWarning}
components.Badge{Text: "Failed",   Variant: components.VariantError}
```

| variant | fill |
|---|---|
| `VariantDefault` (zero value) | `Primary` |
| `VariantSuccess` | `Success` |
| `VariantWarning` | `Warning` |
| `VariantError` | `Error` |

The zero value is the look every badge had before the field existed, so
adding it restyles nothing.

**The ink is computed, not fixed.** The palette pairs no ink with a status
role, and the right answer flips between themes — `DefaultTheme`'s Success is
a *light* green wanting dark ink, `MaterialTheme`'s is a dark one wanting
light ink. So for a status variant the label takes whichever of the theme's
two ink roles has more contrast against the fill. Reusing the default
variant's white ink would render DefaultTheme's Success and Warning badges at
~2.2:1 — unreadable. `VariantDefault` is exempt and keeps the theme's
Primary/Background pairing, which is what makes the zero value a no-op.

Explicit `Color` beats `Variant`, and the ink is still resolved against
whichever fill won, so an override cannot silently produce an illegible pill.
Explicit `TextColor` beats the computed ink.

**A variant reinforces the text, it does not replace it.** Nothing announces
"warning" to a screen reader, and a reader who cannot tell the tints apart
sees only the label — so the label has to say it ("Overdue", not "!").

`Variant` is a package-level type, not Badge's own, so a future Alert or
banner resolves the same four roles the same way. `Variant.Color(theme)` and
`Variant.Ink(theme, bg)` are exported for building your own status surface.

For a *selectable* pill, use Chip.

## Chip

A selectable pill — filter toggles, tag pickers. **Controlled**: the chip
holds no state; it renders `Selected` and reports taps through `OnTap`, so a
chip group is one piece of parent state plus a loop.

```go
components.Chip{
    Label:    label,
    Selected: i == active,
    OnTap:    func() { onSelect(i) },
    AccessibilityLabel: "Show " + strings.ToLower(label) + " tasks",
    AccessibilityHint:  "Filters the task list",
}
```

- Selected is the loud state: the theme's Button base, untouched (plus a ring
  painted in the fill, so both states hold the same box). Unselected is the
  quiet one: Surface fill, `TextPrimary` ink, a hairline rule.
- `Prominence` tunes *how* quiet the unselected state is — see below.
- `SelectedStyle` and `UnselectedStyle` replace their state's default. Both
  read `nil` as "use the default" and an allocated-but-empty slice as "apply
  nothing", which is how you drop a default instead of overriding it.
- `Style` applies to both states and the state wins where they collide —
  otherwise one `Style` shared across a strip would flatten the distinction
  the strip is drawing.
- When selected, `", selected"` is appended to the accessibility label so
  screen readers announce state with the name.

The two state defaults used to be the other way round — the selected chip was
the quiet one — which read as an inverted filter row and is the one thing that
has changed here. `SelectedStyle: {Surface fill, Primary ink}` plus
`UnselectedStyle: []core.StyleProp{}` restores it.

### Prominence: quiet or loud

Which state is louder is settled — the selected one — and that is not what
this field touches. *How much* quieter the other one is has two right answers:

| | |
|---|---|
| `ProminenceQuiet` (zero) | Surface fill, `TextPrimary` ink, hairline rule. Right for a **filter** row, which is chrome above the content it filters: a loud row of years competes with the archive it is filtering. |
| `ProminenceLoud` | The chip's accent as ink and as a 1px rule over a transparent fill — the outlined treatment. Right for a row of **suggestions** the reader is meant to reach into: grey pills over an empty amount field do not read as "tap one of these". |

```go
components.Chip{Label: "$25", Prominence: components.ProminenceLoud,
    Selected: cents == 2500, OnTap: func() { set(2500) }}
```

Material draws the same distinction (filter chip vs. suggestion chip) with a
different default prominence for each.

Loud is **not** the pre-inversion look. That one gave every unselected chip a
solid fill and left the chosen one pale; here the fill is transparent, so the
selected chip is still the only solid pill in the row.

The accent is the theme's own `Components.Button` background — the fill the
selected chip paints — so the outline and what it becomes when tapped are the
same hue on any theme. A theme with no Button fill falls back to
`Colors.Primary`. Legibility over a transparent fill is the palette's, exactly
as it is for an outlined [`Button`](#button) — and the numbers are that
widget's `default` row, since it is the same colour on the same backdrop:
**4.02:1** under `DefaultTheme` (`#007AFF` on white), **7.63:1** under
`MaterialTheme`. Only the second clears WCAG AA at the theme's Button font
size, so a screen leaning on loud chips under a `DefaultTheme`-like palette
should override `TextColor` through `UnselectedStyle`.

`UnselectedStyle` wins where both are set — it replaces the treatment,
`Prominence` picks between them. And because `SegmentedControl.Segment` is a
whole `Chip`, `Segment: components.Chip{Prominence: components.ProminenceLoud}`
carries it to every segment.

The todoapp filter bar is built on Chip; `examples/todoapp/chip_migration_test.go`
pins its rendered HTML against the same bar written out by hand — the pattern
to copy when extracting your own widgets.

## SegmentedControl

A controlled single-select rendered as a row of chips — a filter bar, a mode
switcher, a scope picker.

```
Row (Gap)
  ├─ Chip "All"     ← Selected == 0
  ├─ Chip "Active"
  └─ Chip "Done"
```

```go
components.SegmentedControl{
    Labels:   []string{"All", "Active", "Done"},
    Selected: filter.Get(),
    OnSelect: func(i int) { filter.Set(i) },
}
```

| field | effect |
|---|---|
| `Labels` | segment captions, left to right; `Selected` indexes this slice |
| `Selected` | index of the active segment; out of range selects **nothing** |
| `OnSelect` | fires with the tapped segment's index |
| `Segment` | the `Chip` template every segment is rendered from |
| `SegmentLabel` | derives a segment's accessibility name from its caption |
| `KeyPrefix` | prepended to each segment's reconciler key (default: the caption) |
| `Gap` | spacing between segments; zero means the theme's `SM` step |
| `Style` | applied to the row **after** `Gap`, so it overrides it |

**Selection is an index and the caller owns it.** The control holds no state:
it renders `Selected` and reports taps — the same contract [`Chip`](#chip) has,
one level up. That is what lets the selected index *be* the app's own filter
enum; `examples/todoapp` declares `filterAll`/`filterActive`/`filterDone` as
indices into its label slice, so there is no mapping in between.

An out-of-range `Selected` selects nothing. That is a legal state, not a
defensive check: a scope picker that starts with no scope chosen says so with
`-1` rather than by growing a fourth "none" segment.

**`Segment` is a template, not a set of pass-through fields.** Everything a
`Chip` can do — `Style`, `SelectedStyle`, `AccessibilityHint` — is set once and
applies to all of them:

```go
components.SegmentedControl{
    Labels:    filterLabels,
    Selected:  active,
    OnSelect:  onSelect,
    KeyPrefix: "filter-",
    Segment: components.Chip{
        Style:             []core.StyleProp{core.FontSize(13)},
        SelectedStyle:     []core.StyleProp{core.BackgroundColor(colorAccent)},
        AccessibilityHint: "Filters the task list",
    },
    SegmentLabel: func(label string, _ int) string {
        return "Show " + strings.ToLower(label) + " tasks"
    },
}
```

`Label`, `Selected` and `OnTap` on the template are ignored — those three are
exactly what the control computes. The alternative was re-exporting Chip's
surface as `SegmentStyle`, `SelectedSegmentStyle`, `SegmentHint` and so on,
which grows a field every time `Chip` does. It is the same move
[`InputRow`](#inputrow) makes with `Button`.

**`SegmentLabel` is a function because the name is the one thing that varies
per segment and is not derivable from the caption** — todoapp announces "Show
active tasks" for a chip captioned "Active". A parallel `[]string` would have
to be kept in step with `Labels` by hand. `Chip` still appends `", selected"`
to whichever name it ends up with, so state and name are announced together;
return the name only. A nil `SegmentLabel` leaves `Chip` to announce the
caption itself.

**Segments are keyed**, by `KeyPrefix` + caption. Keys never appear in exported
HTML but they drive reconciler matching and native view recycling, so captions
are assumed distinct — two identical captions collide, which debug mode reports
rather than silently mismatching segments.

## Separator

The hairline rule between rows and between sections. The zero value is the
common case:

```go
components.Separator{}
components.Separator{Inset: 56}          // starts under the text, not the avatar
components.Separator{Thickness: 0.5}     // sub-pixel hairline on a 2x display
```

- Always hidden from assistive technology. A rule carries no information,
  and one between every pair of rows turns a 20-row feed into 39
  utterances.
- No forced margin, which is what makes it usable inside a list —
  `core.Divider` force-applies `Margin(8)`, and that is why neither example
  that wanted a rule used it.
- `Inset` is applied as left/right **margin**, not `EdgeInsets.Horizontal`:
  the HTML exporter reads only the four per-side fields.

**Horizontal only, for now.** A vertical rule has to stretch to its row's
height, which is cross-axis stretch, and that used to be the blocker: neither
renderer mapped `AlignItems: "stretch"`, so the rule would have collapsed to
zero height on both. Both map it today — Compose pins a stretched `Row` to
`IntrinsicSize.Max` and gives each child `fillMaxHeight()`, and SwiftUI's
`GrMobFlexStack` proposes the full cross extent to a stretched child — so
adding a `Vertical` field is now a widget change rather than a renderer one.
It has simply not been added. Note the one asymmetry if you hand-roll it
meanwhile: a `Row` reads `AlignItems` only, never the simpler `Align`
fallback (`Align` is a text-alignment concept and has never applied to a
row's vertical axis), so the containing row needs `AlignItems: "stretch"`
spelled out.

The default tint is the theme's `Border` role, read through
`ColorPalette.BorderColor()` rather than off the field, so a theme written
before that role existed falls back to `core.FallbackBorder` (`#E5E5EA`)
instead of rendering an invisible rule. `Color` still overrides per instance.

## Avatar

The circular portrait: a remote image when there is one, initials on a
colored disc when there is not.

```go
components.Avatar{Src: user.PhotoURL, Name: user.Name}  // image, labelled
components.Avatar{Name: "Ada Lovelace"}                 // "AL" on a disc
components.Avatar{Name: "Ada Lovelace", Size: 64}
```

- Both branches are the same square with `BorderRadius = Size/2`, so `Size`
  is the single knob. (An oversized fixed radius would also give a circle,
  but would silently keep the old geometry when `Size` changed.)
- Initials derive from the **first and last** words of `Name` — "Ada King
  Lovelace" is AL, not AK — uppercased, rune-based so non-Latin names keep
  whole characters. `Initials` overrides when the rule gets it wrong.
- The disc is a `Row` with `JustifyCenter` + `AlignItemsCenter`, because
  `Box` is pinned to the top-leading corner on both platforms and cannot
  centre a child.
- Image avatars default to a Surface background: `core.Image`'s theme base
  is `Components.Camera`, whose background is solid black — right behind a
  viewfinder, wrong behind a portrait that has not downloaded yet.

**Accessibility**, unlike `ListRow`, *is* synthesized here, because an
avatar has exactly one meaning and `Name` is it:

| state | result |
|---|---|
| `AccessibilityLabel` set | used verbatim |
| `Name` set | used as the label |
| neither | the node is hidden from assistive tech |

The last row is the important one: an unnamed avatar is decoration beside
text that already names the person, and unlabeled it is announced as
"image" or read out as its URL.

*Non-square images letterbox* inside the circle — the renderers scale with
`.scaledToFit` / Compose's Fit default. `Avatar` does not expose a way to
change that, though the underlying prop now exists:
`core.ImageWithMode(src, core.ContentModeFill, ...)` covers the four modes
(`Fit`, `Fill`, `Stretch`, `Center`) on every renderer. Threading it through
`Avatar` is a widget change waiting for a caller that wants it.

## ProgressBar

The determinate track-and-fill bar.

```go
components.ProgressBar{Value: 0.45, AccessibilityLabel: "Upload"}
components.ProgressBar{Value: done / total, Thickness: 10, Color: "#34C759"}
```

- `Value` is clamped to 0–1 rather than rejected (NaN reads as 0): a bar fed
  a live ratio should pin at full and keep rendering.
- The percentage is appended to the accessibility label, because no renderer
  has a progress semantic to carry the value natively. With no label the bar
  is hidden — an unlabeled bar announces a bare number with nothing to
  attach it to.
- The fill renders at every value, zero-width included. A constant child
  count keeps advancing progress a *style patch* on one node instead of an
  insert/remove, which is also what lets a `Transition` animate it.

**Why a percentage width and not two flex weights.** The obvious build is
two boxes weighted `FlexGrow(v)` / `FlexGrow(1-v)`. That was exact on
Android, where `FlexGrow` maps onto Compose's `Modifier.weight`, and
silently wrong on iOS, where it mapped onto `frame(maxWidth: .infinity)`:
SwiftUI stacks have no weight, so two growers split free space *equally
regardless of their values*. Every bar would have sat at 50% on iOS. A
percentage width is proportional on all three targets instead:

| target | mapping | accuracy |
|---|---|---|
| Android | `fillMaxWidth(fraction)` | exact |
| HTML | `width:<pct>%` | exact |
| iOS | `containerRelativeFrame` | proportional; measured against the nearest *container*, so a bar that spans its container is exact and one inset in a narrow card reads wide |

Proportional weights have since landed on iOS: `GrMobFlexStack` is that
custom `Layout`, and `GrMobFlexSolver` resolves `FlexGrow` by value on all
three targets. The bar has not been migrated, so the caveat above still
describes what it does today — but the blocker is gone, and moving it to flex
would remove the caveat, since a flex child is measured against its immediate
parent rather than the nearest container.

## FormField

The label / input / hint-or-error frame around any input:

```go
components.FormField{
    Label: "Email",
    Hint:  "We never share it",
    Input: core.Input(email.Get(), "you@example.com", func(v string) { email.Set(v) }),
}
```

`Error`, when non-empty, **replaces** `Hint` (a field shows one line of
feedback; an error outranks guidance) and inks with the theme's Error color.
The `Input` slot keeps it agnostic to what is wrapped — `Input`, `TextArea`,
`NumericInput`, a custom picker.

The widget renders feedback; it does not produce any. What fills `Error` is
[`forms`](concepts/forms.md), which also decides *when* a message should be
visible:

```go
components.FormField{
    Label:    "Email",
    Required: form.Required("email"),
    Hint:     "We never share it",
    Error:    form.Error("email"),
    Input:    form.Input("email", "you@example.com"),
}
```

`Required` draws the conventional asterisk after the label, inked in the theme's
Error color and announced to screen readers as "required". It is annotation
only — the widget still validates nothing — which is why it is worth asking
[`form.Required(name)`](concepts/forms.md#the-required-marker) rather than
writing `true`: the form derives its answer from the field's own rules, so the
marker cannot outlive the rule that justified it. It is ignored when `Label` is
empty, there being nothing to mark.

Because the `Input` slot takes any view, wrapping is also how a control with
no error line of its own gets one — a checkbox row, for instance, with the
`ListRow` title standing in for the label:

```go
components.FormField{
    Error: form.Error("terms"),
    Input: components.ListRow{
        Leading: form.Checkbox("terms"),
        Title:   "I accept the terms of service",
    },
}
```

## Accordion

A collapsible section — tappable chevron header, content shown while
expanded.

```go
components.Accordion{
    Title:   "Advanced options",
    Content: advancedPanel,
    InitiallyExpanded: false,
}
```

!!! warning "Accordion owns state"
    Accordion is the one widget in the package that calls `NewState`, so the
    [rules of hooks](concepts/state-and-hooks.md#the-rules-of-hooks) apply to
    it: render it unconditionally, in a stable position, every pass. And
    because `Content` only renders while expanded, it must be **hook-free**
    (interactive, hook-free content is fine — its callbacks re-register on
    every visible pass). [Debug mode](concepts/debug-mode.md) flags
    violations as cursor drift.

`Header` replaces the default title text (the tap target and toggle stay
with the widget); `InitiallyExpanded` seeds the first pass only.

## Tabs

The named-field facade over `core.TabView`:

```go
components.Tabs{
    Items:    []core.TabItem{core.Tab("Home", "🏠"), core.Tab("Search", "🔍")},
    Selected: tab.Get(),
    OnChange: func(i int) { tab.Set(i) },
    Content:  []core.View{homePage, searchPage},
}
```

Tabs *wraps* rather than supersedes `TabView`: the `"TabView"` node type is a
wire contract the native renderers consume, and node-type contracts live in
core — this struct only supplies the field names. All pages are children of
the node; the native side shows the selected one. With no `OnChange`, no
callback is registered, keeping static tab strips diff-stable.

All four targets draw a bar above the selected page. The two DOM ones hide the
other pages rather than dropping them, and wire each tab to its page as an ARIA
tab set — `role="tabpanel"`, `aria-controls`, `aria-labelledby` — so a screen
reader on the web knows which region each tab governs (see
[WASM — Tab views](platforms/wasm.md#tab-views)); the natives compose only the
selected one, and give it a fresh identity on each switch, so per-tab view state
is dropped there. `Selected` is controlled state on every target: a switch
arrives as a prop change, never as a rebuilt subtree. The `Icon` of a
`core.TabItem` is drawn by no target.

## Collections — GroupedList & DataTable

Keyed, lazily-composed collections over `core.List`, with run-length group
headers, controlled sort, compact mode and client- or server-side paging.
`Pagination` (numbered pages) and `LoadMore` (the four-state tail: nothing /
Load more / Loading… / error + Retry) are the footers.

```go
components.DataTable[Entry]{
    Columns: []components.Column[Entry]{
        {Title: "Title", Weight: 2, Text: title, Less: byTitle},
        {Title: "Speaker", Narrow: true, Text: speaker},
    },
    Rows: entries, Key: entryKey,
    Sort: sortBy.Get(), OnSort: func(s components.Sort) { sortBy.Set(&s) },
    Pagination: &components.Pagination{Page: page.Get(), PageSize: 20, OnChange: page.Set},
}
```

Both are hook-free and fully controlled. Rows are sorted, then paged, then
grouped. Give a column `Less` only when `Rows` is the whole set: `Less` sorts
all of `Rows` and only `Rows`, so sorting an accumulated window yields the
first rows *of the window* under a header claiming the first rows of the
table. When the server pages, set `Sortable` without `Less` and put the sort
in the query — [debug mode](concepts/debug-mode.md) reports the detectable
half of that as a `partial-sort` concern. `HideTrailingCount` suppresses the
last group's badge while a pager still has pages to fetch, since a closed
group's count is final and an open one's is about to change.

### Sticky bands and infinite feeds

`StickyHeaders` pins each group band to the top of the viewport while its run
scrolls underneath (both widgets); `OnEndReached` fires when the reader gets
within a few rows of the bottom, so the next page arrives without a tap
(`GroupedList`).

```go
components.GroupedList[Entry]{
    Items:         pager.Items,
    GroupBy:       byMonth,
    StickyHeaders: true,
    OnEndReached:  pager.LoadNext,   // the scroll
    Footer: components.LoadMore{     // and the tap, and the states
        HasMore: pager.HasMore, Loading: pager.Loading,
        Err: pager.Err, OnLoadMore: pager.LoadNext},
}
```

**Keep the footer.** Auto-loading replaces the tap, not the tail: `LoadMore`
is still where "Loading…" and a failed page's Retry live, and it is the
manual fallback wherever the edge cannot be reported (a static export, a
browser with no `IntersectionObserver`). Handing the same load function to
both is the intended shape — `core.OnEndReached` will not re-ask until the row
count changes, so a tap and a scroll cannot double-load.

`StickyHeaders` pins the *default* `GroupHeader`. A `Header` override builds
its own view, which the widget cannot reach into; such a header pins itself
with `core.StickyHeader()` in its own `Style`. On `DataTable` the flag pins
the group bands and not the column header — the column header is a sibling of
the body list rather than a row inside it, which is what keeps it on screen in
the first place.

`DataTable` states its structure as well as drawing it:
`RoleTable` on the table, `RoleRowGroup` on the body list, `RoleRow` on the
header and body rows, `RoleColumnHeader` and `RoleCell` on their cells. The
rowgroup is load-bearing rather than decorative — ARIA reads the rows a table
*owns*, and the body list is a container between the two, so without it the
other four describe a table with no rows. A busy or empty table withholds the
rowgroup, since what the body holds then is one placeholder and not rows.

See lessons 4.6 and 4.8 of the [interactive tutorial](tutorial-interactive.md)
and the godoc for the full field list.

## AppBar

The title strip at the top of a screen: an optional back affordance, the
screen's name, and trailing actions.

```go
components.AppBar{
    Title:    "Sermon",
    Subtitle: "22 March 2026",
    Actions:  []core.View{shareButton},
}
```

- **The back control appears only when there is somewhere to go.** With no
  `Leading` and no `HideBack`, the arrow is drawn exactly when
  `core.CanPop(ctx)` is true — so a tab root gets none without asking and a
  pushed screen gets one without wiring. `OnBack` *replaces* `core.Pop`
  rather than running before it, which is what makes a confirm-before-leaving
  handler possible.
- `Leading` replaces the automatic control entirely — a close button on a
  modally presented screen, where `CanPop` is false. `Content` replaces the
  Title/Subtitle stack.
- The title takes the theme's **Subtitle** size with the primary ink and a
  bold weight. `Typography.Title` is the screen's large heading (28pt under
  `DefaultTheme`) and does not fit in a bar.
- A hairline rule is drawn under the bar unless `HideSeparator` is set,
  because an unstyled bar sits on the same Background as the content below it.

The bar carries `RoleBanner` and its `Title` carries `RoleHeading`, so a
reader navigating by landmark or by heading lands where they expect. The
banner sits on the bar row (not on the box the separator adds around it) so
`Style` can override it — a second bar on a screen that already has a banner
should say something else.

It is an ordinary `Row`, not a platform navigation bar: nothing floats,
collapses on scroll, or claims the status bar. `Screen`'s `SafeArea` is what
keeps it clear of the notch.

## Banner

The inline strip that tells the user something about the screen they are on:
a failed refresh over content that is still good, an offline notice, a
"Reconnecting…".

```go
components.Banner{
    Text:        "Could not refresh. Showing a saved copy.",
    Variant:     components.VariantWarning,
    ActionLabel: "Retry", OnAction: reload,
    OnDismiss:   func() { notice.Set(false) },
}
```

**The variant is a tint, not a fill.** A hairline border and the leading
glyph take the role color; the strip keeps the theme's Surface and the primary
ink. A saturated Error red across the width of a screen reads as a failure of
the app rather than of one fetch, and the palette carries no muted container
tone to fill with instead. The upshot is that a banner's contrast does not
depend on which variant it is.

Default glyphs are `ⓘ ✓ ⚠ ⊗` for the four roles, overridable with `Glyph` and
droppable with `NoGlyph`. They are **decoration** and are hidden from
assistive technology, so `Text` has to carry the meaning — "Could not
refresh", not "Something went wrong" beside a red edge.

**It announces itself when it appears.** A banner shows up because something
changed, usually while the reader is elsewhere on the screen, so the strip is
a live region: `RoleAlert` when the variant is Error, `RoleStatus` otherwise —
error interrupts, everything else waits for a pause. Override it through
`Style` for a strip that is really static content.

It is not a toast: `core.ShowToast` disappears on a timer, a Banner stays
until the state that produced it changes. For an edge-to-edge strip with no
frame, pass `core.BorderWidth(0)` and `core.BorderRadius(0)` in `Style`.

## EmptyState

The centered placeholder for content a screen does not have — and for the
other two moments with the same shape:

```go
empty   components.EmptyState{Glyph: "📭", Title: "No messages yet"}
busy    components.EmptyState{Title: "Loading sermons…"}
failed  components.EmptyState{Glyph: "☁", Title: "Could not reach the server.",
            ActionLabel: "Retry", OnAction: reload}
```

Three states, one widget, so their wording and spacing cannot drift apart.
The busy case is a line of text rather than a spinner because core has no
indeterminate progress node — and naming what is loading is more useful than
an animation anyway.

The column sets `Width: 100%`, which looks redundant and is not: on both
natives a column hugs its widest child, so without it the block sits at the
leading edge with its children centered inside a box only as wide as the
longest line. The DOM targets fill the line already, so the bug is invisible
on the target you are most likely to be looking at.

The built action is **outlined**, not filled: an empty state is a dead end,
and a solid Primary button in the middle of an empty screen is the loudest
thing on it.

## SearchField

A text field dressed as a search box: a leading magnifier, a flexible input,
and a clear button that appears once there is something to clear.

```go
d := hooks.UseDebounce(ctx, 250*time.Millisecond)

components.SearchField{
    Value: query.Get(),
    OnChange: func(s string) {
        query.Set(s)                      // now: the field is controlled
        d.Call(func() { runSearch(s) })   // in 250ms, if the typing stopped
    },
    OnSubmit: func() { d.Cancel(); runSearch(query.Get()) },
}
```

**It holds no state and calls no hook**, which is what lets a search box live
in a header that appears and disappears — a hook-slot consumer could not.
That is also why it cannot debounce its own `OnChange`: a controlled field's
value has to reach state on the keystroke or the characters do not appear.
What wants delaying is the *reaction*, and that lives in the caller — see
[`hooks.UseDebounce`](concepts/state-and-hooks.md#hooksusedebouncectx-delay).

The row paints the theme's Surface at the theme's own field radius and the
input inside it is flattened (transparent, no radius, no padding), so there is
one box rather than two. `AccessibilityLabel` falls back to the resolved
placeholder, because a placeholder is not a label on any platform — it
vanishes on the first keystroke.

The row carries `RoleSearch` — the landmark is the whole region, so a reader
jumping to it arrives before the clear button rather than past it. The input
keeps its own name; a role says what a region is and a label says what a
control is called.

## ChipStrip

A run of `Chip`s that wraps onto as many lines as it needs — a filter bar,
tags on an article, quick amounts on a form.

```go
components.ChipStrip{Chips: []components.Chip{
    {Label: "All",      Selected: f == "",        OnTap: func() { filter.Set("") }},
    {Label: "Sermons",  Selected: f == "sermon",  OnTap: func() { filter.Set("sermon") }},
}}
```

The field is `[]Chip` rather than a parallel vocabulary of labels and
callbacks, so a chip in a strip is configured exactly like a chip anywhere
else. `Children` is the escape hatch for a strip mixing chips with something
else.

**ChipStrip is not SegmentedControl.** The segmented control is one-of-N: a
fixed, exhaustive set drawn as one joined control. ChipStrip is the loose
case — any number selected including none, a set that comes from data.

`Scrollable` makes the strip one line that pans sideways instead of a block
that wraps:

```go
components.ChipStrip{Scrollable: true, Chips: years}
```

Wrapping is right for a set the reader should see all of — the tags on an
article, the references on a sermon. Panning is right for a long filter bar,
where a strip growing to three lines pushes the content it filters off the
screen and the chips past the fold read as "there is more" rather than as a
queue. The two are exclusive: a scrolling strip is one line, so there is
nothing to wrap.

Under the hood it becomes a `core.Scroll` carrying `core.Horizontal()`, not a
`Row` with an overflow — the natives implement sideways panning in their
scroll composites alone, so the node type has to change. `Style` still lands
on the strip itself either way.

The strip claims **no** role of its own. A filter bar is a toolbar and can say
so with `Style: []core.StyleProp{core.AccessibilityRole(core.RoleToolbar)}`,
but the tags on an article are not one, and the widget cannot tell them
apart — nor does it implement the roving focus a toolbar implies.

## Skeleton

The grey placeholder that holds a screen's shape while its content loads.

```go
components.Skeleton{}                                        // one line
components.Skeleton{Lines: 3}                                // a paragraph
components.Skeleton{Width: "44px", Height: 44, Radius: 999}  // an avatar
```

A stack's last bar is short (`LastLineWidth`, 60% by default), which is what
makes it read as a paragraph — applied only when `Lines` is 2 or more, since
on a single bar the last line is the only line.

The bars take the palette's **Border** role, not Surface: Surface is a
*panel's* fill, so a Surface bar inside a card disappears — the same trap
`Separator` documents. They are hidden from assistive technology and the
container carries the label (`"Loading"` by default).

**No shimmer.** A moving highlight is a repeating keyframe animation, and
`core.Transition` animates a property between two declared values. Looping it
from Go would push a render pass and a bridge patch per frame of a
decoration, which is the one thing the "declare in Go, animate natively" model
exists to avoid.

Skeleton and EmptyState answer different questions: a skeleton says content is
coming and will look roughly like this (worth saying when the layout is
known); an empty state says there is nothing here and why.

## StatTile

One figure with its name and, optionally, its movement.

```go
core.Row(core.Gap(12),
    components.StatTile{Label: "Attendance", Value: "412", Fill: true,
        Delta: "+18 vs last week", DeltaVariant: components.VariantSuccess},
    components.StatTile{Label: "Giving", Value: "MZN 42,750", Fill: true},
)
```

**It has no frame.** "Tile" names the content, not a card: the widget paints
and insets nothing, which is what lets three tiles share one `core.Card` or
take one each, rather than a `Framed` bool that is wrong half the time.

**The delta's zero variant is neutral, not Primary** — the one place in this
package where `VariantDefault` is not the theme's brand color. A delta is a
measurement, and whether a number going up is good is the caller's domain:
attendance up is a success, spend up is not, latency up is an incident. So
the default says nothing. The [outlined-Button contrast caveat](#contrast-and-what-the-widget-can-promise)
applies to a colored delta as well — under `DefaultTheme`, Success is 2.22:1
and Warning 2.20:1 against the Background.

`Fill` sets `FlexGrow` **and** a zero `FlexBasis`, which is what makes the
four targets agree: Compose and SwiftUI divide the whole axis by weight, CSS
divides only the leftover space. The natives ignore `FlexBasis`, so the prop
that is inert on two targets is exactly the one that converges the other two.

## Writing your own

The package doc (`components/doc.go`) is the reference for the idiom. In
short:

1. A struct with named fields; `core.View`-typed fields for slots.
2. `Render(ctx)` builds on core containers/widgets and returns their node.
3. Read the theme; accept `Style []core.StyleProp` for overrides.
4. If the widget owns state, document its hook obligations (see Accordion).
5. Give it a focused test — and if it's extracted from app code, pin the
   rendered output against the original with `htmlout.ExportHTML`.
