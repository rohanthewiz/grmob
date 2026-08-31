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
