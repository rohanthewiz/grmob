# Components

A **component** is a Go value that implements `core.View` and builds its
node tree entirely out of other views. It has no registration step, no
lifecycle interface and no per-platform code. It needs none of these because
it ends in primitives that every host already knows how to draw.

This page settles the vocabulary, sets out the contract a component keeps,
and walks through writing one: first a leaf, then one that owns state, then
its tests. The canonical statement is the doc comment on
[`core.View`](../api/core-views.md); this page explains it.

## Four words, one interface

Everything on screen implements one interface:

```go
type View interface {
    Render(ctx *Context) *Node
}
```

The docs use four words for the things that implement it, always in these
senses:

| Word | Means | Examples |
|---|---|---|
| **View** | the interface itself. Each of the other three is a View. | any child, slot or root |
| **Primitive** | a `core` constructor whose node type a host draws itself. Adding one touches Compose, SwiftUI, the wasm runtime and `htmlout`. | `core.Text`, `core.Row`, `core.Button`, `core.TextInput`, `core.Canvas` |
| **Component** | a View written in Go from other Views: no new node type, no host code | an app's screens, a `todoRow` helper, `core.If` |
| **Widget** | a component built for reuse: a struct with named fields that keeps the whole [contract](#the-contract) | every type in [`comps`](../components.md) |

Apps are made almost entirely of components. Primitives are core's business,
and a missing one is [a gap in core](#when-it-is-not-a-component), not
something an app works around.

## Three shapes

Pick the smallest shape that does the job:

```
needs named knobs or slots, reused across screens? ──no──▶ a function returning a View
        │ yes
needs a node type, Style field or role core lacks? ──yes──▶ a core gap, not a component
        │ no
a struct with a Render method: a widget
```

**A function returning a View** suits a screen-local piece. It holds no
hooks, so it is safe to create and discard inside `core.For` or `core.If`:

```go
func todoRow(t Todo, setDone func(int, bool)) core.View {
    return core.Keyed(fmt.Sprintf("todo-%d", t.ID), core.Row(
        core.Checkbox(t.Done, func(v bool) { setDone(t.ID, v) }),
        core.Text(t.Title, core.FlexGrow(1)),
    ))
}
```

**A struct with a `Render` method** suits anything reused. Named fields let
it grow a knob without breaking a call site, and a `core.View` field is a
composition slot. This is the `comps` idiom.

**`core.ComponentFunc`** adapts a render function to a View, the way
`http.HandlerFunc` adapts a function to a `Handler`. Most of core's own
constructors are ComponentFuncs closing over their arguments. Reach for it
when you need a render function as a value. A struct is a View already and
needs no adapter.

## The contract

Rules 1–3 are about correctness. Breaking one makes the app misbehave, so
every component keeps them, screens included. Rules 4–6 are what make a
component reusable, and every widget keeps all six.

| # | Rule | What breaking it looks like |
|---|---|---|
| 1 | `Render` builds a fresh tree and never mutates a Node after returning it. | The diff takes a reused pointer as proof that nothing changed, so the change never reaches the screen. |
| 2 | A component uses no hooks, or it takes every hook before any branch and is rendered on every pass. It owns state only when that state is purely presentational. | State jumps between components ([cursor drift](state-and-hooks.md)). |
| 3 | It never registers a nil callback, and calls its caller's callbacks nil-safely. | A panic on the first native tap. |
| 4 | Its look comes from `ctx.Theme()`, and its caller's `Style` props are applied after its own. | Unreadable under `DarkTheme`; an override that has no effect. |
| 5 | It states its role, name and state on its nodes, and reports misuse with `core.ReportConcern` under debug mode. | Silent to a screen reader; a mistake that surfaces only on a device. |
| 6 | Every field's zero value means something sensible. | A zero `Max` that clamps every value to 0. (`comps.Stepper` applies its bounds only when `Max > Min` for this reason.) |

The rest of this page covers each rule as it comes up.

## Writing a leaf widget

A complete widget that holds no state. It compiles as written:

```go
package ui

import (
    "strconv"

    "github.com/rohanthewiz/grmob/comps"
    "github.com/rohanthewiz/grmob/core"
)

// Tally is a labelled count: "Unread  12" with the number in a pill.
//
//	ui.Tally{Label: "Unread", Count: n, Variant: comps.VariantWarning}
//
// No hooks; may be rendered conditionally.
type Tally struct {
    // Label is drawn before the count and names it for screen readers.
    Label string
    // Count is the number in the pill.
    Count int
    // Variant colours the pill. The zero value is the theme's primary.
    Variant comps.Variant
    // Format renders the count. Nil uses strconv.Itoa.
    Format func(int) string
    // Style is applied to the row after the widget's own props.
    Style []core.StyleProp
}

func (w Tally) Render(ctx *core.Context) *core.Node {
    t := ctx.Theme()

    text := strconv.Itoa(w.Count)
    if w.Format != nil {
        text = w.Format(w.Count)
    }
    fill := w.Variant.Color(t)

    // Defaults first, the caller's Style last, so a caller override wins.
    items := make([]core.PropsAndChildren, 0, len(w.Style)+6)
    items = append(items,
        // The theme pads every Row for use as a screen band. A control sits
        // inside a padded row or field already, so it clears that inset.
        core.Padding(0),
        core.Gap(float64(t.Spacing.SM)),
        core.AlignItemsProp(core.AlignItemsCenter),
        // One stable phrase for readers: "Unread, 12".
        core.AccessibilityLabel(w.Label+", "+text),
    )
    for _, sp := range w.Style {
        items = append(items, sp)
    }
    items = append(items,
        core.Text(w.Label, core.UseStyle(t.Typography.Body),
            core.TextColor(t.Colors.TextPrimary), core.FlexGrow(1)),
        core.Text(text,
            core.UseStyle(t.Typography.Caption),
            core.BackgroundColor(fill),
            core.TextColor(w.Variant.Ink(t, fill)),
            core.PaddingHorizontal(t.Spacing.SM),
            core.PaddingVertical(t.Spacing.XS),
            core.BorderRadius(999),
        ),
    )
    return core.Row(items...).Render(ctx)
}
```

Points to notice:

- **Every colour and size comes from the theme** (rule 4): `t.Colors`,
  `t.Spacing`, `t.Typography`, and `Variant.Color` / `Variant.Ink` for a
  fill and the ink drawn on it. A literal colour breaks under `DarkTheme`
  and under every custom theme. See [Styling & Theming](styling-and-theming.md).
- **The caller's `Style` comes after the defaults**, so
  `Style: []core.StyleProp{core.PaddingLeft(0)}` beats the widget's own
  inset. The only exception is behaviour that must not be overridden:
  `comps.Button` appends `core.Disabled(true)` after the caller's Style, so
  a styling tweak cannot re-enable a disabled button.
- **`core.Padding(0)` on a nested Row or Column.** The theme insets both for
  use as screen bands. A control nested in something already padded would
  otherwise be indented twice. `core.Box` carries no theme base at all.
- **The zero value is the existing look** (rule 6). `Variant`'s zero is
  primary, and a nil `Format` falls back to `strconv.Itoa`. When a zero
  would be a trap, rename the field so that zero is safe: `comps.Poll` has
  `Mine bool` rather than `Voted int`.
- **The doc comment is part of the widget.** It carries a usage line and
  says whether the widget takes hooks. The `comps` widgets also draw their
  node tree in ASCII and list the theme roles they read.

### Field conventions

The `comps` library uses the same field names throughout. Use them too, so
that a reader who knows one widget can predict the next:

| Field | Convention |
|---|---|
| `Value` + `OnChange func(T)` | a controlled pair; the widget never keeps its own copy |
| `OnTap func()` | a tap (core's prop is `core.OnClick`) |
| `Header`, `Body`, `Footer`, `Leading`, `Trailing` | `core.View` slots; when a plain field and a slot are both set (`Title` vs `Header`), the slot wins |
| `Style []core.StyleProp` | caller overrides, applied last; the doc comment names the node they land on |
| `Variant comps.Variant` | a semantic colour role whose zero value is the default look |
| `Disabled bool` | `core.Disabled(true)`, with the handler still registered and guarded |
| `Initially*` | seeds widget-owned state on the first pass only |
| `Format func(T) string` | display formatting; nil means a sensible default |

## Callbacks

Rule 3 exists because a native tap arrives by callback ID. If the function
registered under that ID is nil, the dispatch panics:

```go
onTap := w.OnTap
if w.Disabled || onTap == nil {
    onTap = func() {}
}
```

Call your caller's callbacks nil-safely, and only on a real change:
`if next != w.Value && w.OnChange != nil { w.OnChange(next) }`. Clamp and
validate inside the widget, so that the caller can pass a plain setter such
as `OnChange: qty.Set`. Give rows in a list stable identities with
`core.Keyed`. That keeps the reconciler from mismatching them, and it scopes
their callback IDs too. See [Events & Callbacks](events.md).

## A widget that owns state

A widget's `Render` receives its **caller's** context, so every hook it
calls takes a positional slot in the caller's sequence (rule 2). That has
two consequences:

- Take every hook **first**, before any branch, so the number of slots never
  changes between passes.
- A widget that holds hooks must itself be rendered **unconditionally**:
  never inside `core.If` or a `For` whose length varies. Say so in its doc
  comment. A hook-free widget says "No hooks; may be rendered
  conditionally."

Own state only when it is purely about the widget's own presentation, and
no app would want to read, drive or persist it. An accordion's open/closed
state qualifies. A calendar's visible month does not, because a screen may
need to open on the month of its next event. So `comps.Calendar` takes no
hooks, and the month arrives as a value plus an `OnChange`.

```go
// Spoiler hides Content behind a tappable title until the user reveals it.
//
// It takes one hook (revealed), so it must be rendered unconditionally,
// every pass; Content is rendered only while revealed and must be hook-free.
type Spoiler struct {
    Title   string
    Content core.View
    // InitiallyRevealed seeds the state on the first pass only.
    InitiallyRevealed bool
    // Style is applied to the outer column.
    Style []core.StyleProp
}

func (s Spoiler) Render(ctx *core.Context) *core.Node {
    // Before any branch: the slot is claimed on every pass.
    revealed := core.NewState(ctx, s.InitiallyRevealed)
    t := ctx.Theme()

    chevron := "▸"
    if revealed.Get() {
        chevron = "▾"
    }
    header := core.Row(
        core.Padding(0),
        core.Gap(float64(t.Spacing.SM)),
        // Role, state and handler on the same node: a stated expansion with
        // no click handler is announced on the web and inert on Android.
        core.OnClick(func() { revealed.Set(!revealed.Get()) }),
        core.AccessibilityRole(core.RoleButton),
        core.AccessibilityExpanded(core.ExpandedWhen(revealed.Get())),
        core.AccessibilityLabel(s.Title),
        core.Text(chevron, core.UseStyle(t.Typography.Body)),
        core.Text(s.Title, core.UseStyle(t.Typography.Body), core.FontWeight(core.Bold)),
    )

    items := []core.PropsAndChildren{core.Padding(0), core.Gap(float64(t.Spacing.XS))}
    for _, sp := range s.Style {
        items = append(items, sp)
    }
    items = append(items, header)
    if s.Content != nil {
        items = append(items, core.If(revealed.Get(), s.Content))
    }
    return core.Column(items...).Render(ctx)
}
```

For timers inside a widget, use `hooks.UseIntervalWhile` or
`hooks.UseTimeoutWhile`, so that a stopped widget stops ticking.

## Accessibility

Rule 5 asks each component to state what it is on its own nodes. The rules
that matter most often:

- **Don't restate what a node type already says.** `core.Button` is already
  a button, and `core.Disabled` is already announced as unavailable. Adding
  ", disabled" to the label gets it heard twice.
- **A tappable Row or Box** needs `core.OnClick`,
  `core.AccessibilityRole(core.RoleButton)` and a label. Without the role,
  every renderer treats it as inert scenery.
- **Glyph buttons need spoken names.** "−" should be read as "Decrease", and
  "×" as "Close".
- **A toggle keeps one stable name** and reports its state separately, with
  `core.AccessibilitySelected` or `core.AccessibilityExpanded`, on the same
  node as its `OnClick`.
- **Report misuse; don't panic.** Export a concern code and report it under
  debug mode:

  ```go
  const ConcernTallyUnnamed = "tally-unnamed"

  if core.IsDebugMode() && w.Label == "" {
      core.ReportConcern(ConcernTallyUnnamed, "a Tally with no Label is read as a bare number")
  }
  ```

The full set of roles and states is on the
[accessibility reference](../api/core-accessibility.md).

## Testing a component

Render it under debug mode **and** run the whole-tree audit. A bare
`Render` call skips `core.AuditTree`, which is where unusable value ranges,
duplicate IDs and inert disclosures are reported:

```go
// renderDebug renders v the way render.Manager does and fails on any concern.
func renderDebug(t *testing.T, v core.View) (*core.Context, *core.Node) {
    t.Helper()
    core.SetDebugMode(true)
    core.ClearConcerns()
    t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
    ctx := core.NewContext()
    ctx.BeginRenderPass()
    n := v.Render(ctx)
    ctx.EndRenderPass()
    core.AuditTree(n)
    if dump := core.DumpConcerns(); dump != "" {
        t.Errorf("concerns raised:\n%s", dump)
    }
    return ctx, n
}

// renderPass drives a later pass on the same context, for widgets with hooks.
func renderPass(ctx *core.Context, v core.View) *core.Node {
    ctx.BeginRenderPass()
    ctx.Reset()
    return v.Render(ctx)
}

func TestSpoilerRevealsOnTap(t *testing.T) {
    sp := Spoiler{Title: "Ending", Content: core.Text("they were the butler")}
    ctx, n := renderDebug(t, sp)
    if findText(n, "they were the butler") != nil {
        t.Fatal("content starts hidden")
    }
    header := findFirst(n, func(x *core.Node) bool { return x.Style.AccessibilityRole == core.RoleButton })
    ctx.TriggerCallback(header.Props["onClick"].(string)) // a tap, through the real registry

    n = renderPass(ctx, sp)
    if findText(n, "they were the butler") == nil {
        t.Fatal("a tap reveals the content")
    }
}
```

`findFirst` walks the tree with a predicate, and `findText` matches a Text
node's `content`. Locate nodes by what they are rather than by child index,
so a test survives a layout change. Cover these:

- structure and accessibility: the node type, role, label, value, and the
  spoken names of glyph buttons
- behaviour: fire callbacks with `ctx.TriggerCallback` and its typed
  siblings, and assert what `OnChange` received, including that it was
  *not* called on a no-op
- the zero-value configuration renders with no concerns
- a caller's `Style` outranks the widget's defaults
- every bundled theme (`core.BundledThemes()`), when the widget computes
  colours
- when extracting a component from app code, pin `htmlout.ExportHTML` of the
  tree before the extraction and compare it after

For a whole screen, drive the component through `render.Manager` as in
[Debug Mode](debug-mode.md).

## When it is not a component

A component never needs Kotlin, Swift, JavaScript or exporter changes. If
yours seems to, what is missing is a **primitive**: a node type, a
`core.Style` field or an accessibility role. Adding one touches every host
and is guarded by tests that fail when any host misses it. Before reaching
for one, check whether an existing primitive already covers it.
`core.Canvas` draws anything Text cannot, and `comps.Rating` draws its
half-star on a canvas for exactly that reason. In an app, report the gap
upstream rather than working around it in platform code.

## Contributing a widget to `comps`

A widget for the library follows everything above. It also has to be
registered where the repository keeps its lists:

1. `comps/<snake_name>.go` and a `_test.go` beside it, reusing the package's
   helpers (`orDefault`, `headingProps`, `disclosure`, `appendRows`, the
   chart scaffolding) and the shared test harness.
2. The census tests that apply. Canvas widgets go in
   `canvas_mirror_test.go`. Charts go in `chart_data_test.go`. Composite
   roles go in `nested_composite_test.go`. Widgets with their own insets go
   in `caller_style_insets_test.go`.
3. Its file name goes in the right `comps` topic in
   `internal/apidoc/packages.go`, followed by `go run ./internal/apidoc/gen`.
   `go test ./...` fails on an unlisted file or a stale `docs/api/`.
4. A section in the [widget library](../components.md) page.
5. `gofmt -l .` must print nothing, and `go test ./...` must pass.

The full checklist, with every file and the failure each one prevents, is in
the component skill described below.

## For AI coding agents

[`ai_docs/SKILL-component.md`](https://github.com/rohanthewiz/grmob/blob/master/ai_docs/SKILL-component.md)
is this page as an agent skill, named `grmob-component`. It holds the same
contract and examples, plus the full contribution checklist for `comps`. To
install it, copy it to `~/.claude/skills/grmob-component/SKILL.md`. It builds
on the app-level skill, `ai_docs/SKILL.md` (`grmob-native-mobile-go`).
