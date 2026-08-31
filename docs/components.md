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
  └─ Scroll            (only when Scroll is true)
       └─ Column       ← Gap / Fill / Style land here
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
Of the five app roots in `examples/`, exactly one (`fintechapp`) scrolls as a
whole; `chat` scrolls its message list and `todoapp` scrolls a virtualized
`core.List`.

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

`Disabled` renders the palette's muted pair (Surface fill, TextSecondary ink)
and swallows taps. There is no platform disabled state to set — no renderer
carries one — so the announcement rides the accessibility label as
`", disabled"`, the same convention Chip and ListRow use for `", selected"`.

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

- Unselected: the theme Button base. Selected: `SelectedStyle` if given,
  else the theme default (Surface background, Primary ink).
- When selected, `", selected"` is appended to the accessibility label so
  screen readers announce state with the name.

The todoapp filter bar is built on Chip; `examples/todoapp/chip_migration_test.go`
pins its rendered HTML to the pre-extraction hand-rolled markup byte for
byte — the pattern to copy when extracting your own widgets.

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

**Horizontal only, deliberately.** A vertical rule has to stretch to its
row's height, and neither renderer maps `AlignItems: "stretch"` (Compose
falls through to `Alignment.Top`, SwiftUI to `.top`), so it would collapse
to zero height on both. The field can land with the renderer support.

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
`.scaledToFit` / Compose's Fit default. Filling needs a `ContentMode` prop
on `Image`, which is a two-renderer pass of its own.

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
two boxes weighted `FlexGrow(v)` / `FlexGrow(1-v)`. That is exact on
Android, where `FlexGrow` maps onto Compose's `Modifier.weight`, and
silently wrong on iOS, where it maps onto `frame(maxWidth: .infinity)`:
SwiftUI stacks have no weight, so two growers split free space *equally
regardless of their values*. Every bar would sit at 50% on iOS. A percentage
width is proportional on all three targets instead:

| target | mapping | accuracy |
|---|---|---|
| Android | `fillMaxWidth(fraction)` | exact |
| HTML | `width:<pct>%` | exact |
| iOS | `containerRelativeFrame` | proportional; measured against the nearest *container*, so a bar that spans its container is exact and one inset in a narrow card reads wide |

When proportional weights land on iOS (a custom `Layout`), this can move to
flex.

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

## Writing your own

The package doc (`components/doc.go`) is the reference for the idiom. In
short:

1. A struct with named fields; `core.View`-typed fields for slots.
2. `Render(ctx)` builds on core containers/widgets and returns their node.
3. Read the theme; accept `Style []core.StyleProp` for overrides.
4. If the widget owns state, document its hook obligations (see Accordion).
5. Give it a focused test — and if it's extracted from app code, pin the
   rendered output against the original with `htmlout.ExportHTML`.
