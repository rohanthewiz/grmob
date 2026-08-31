# Views & Composition

Everything on screen is a `View`:

```go
type View interface {
    Render(ctx *Context) *Node
}

type ComponentFunc func(ctx *Context) *Node // adapts a function to a View
```

This is the `http.HandlerFunc` pattern applied to rendering: a component is
either a function (`ComponentFunc`) or any struct implementing `Render` (the
idiom the [widget library](../components.md) uses). Components compose by
calling each other; there is no registration step.

## Containers

The flex-style containers — `Row`, `Column`, `Card`, `Box`, `List` — share
one argument contract: a mixed variadic list of **style props**, **behavior
props**, and **child views**, in any order.

```go
core.Row(
    core.Gap(8),                       // style prop
    core.OnClick(func() { ... }),      // behavior prop → applies to the Row
    core.Text("left"),                 // child
    core.Text("right", core.FlexGrow(1)),
)
```

- `Row` / `Column` — horizontal / vertical flex stacks, themed base padding.
- `Card` — a themed surface (background, radius, shadow).
- `Box` — the unopinionated container: no theme base by design.
- `List` — the virtualized sibling of `Column`: children are laid out lazily
  by the native renderer (Compose `LazyColumn`, SwiftUI `LazyVStack`), so a
  thousand-row feed composes only what is on screen. Use `Column` + `Scroll`
  for short content, `List` for long data-driven collections — and give
  every row a stable identity with `Keyed`.
- `Scroll`, `SafeArea`, `Spacer(px)`, `Divider(height, color)`, `Fragment`.
  (`Divider` force-applies `Margin(8)`; for a rule inside a list use
  `components.Separator`, which leaves spacing to the caller and defaults
  its own hairline tint.)

Behavior props (`OnClick`, `OnTouch`, `OnLongPress`, or the generic
`On(event, fn)`) register their callbacks in argument order, before any child
renders — a container's callback IDs always precede its children's.

## Leaves

| Widget | Signature (abridged) |
|---|---|
| `Text` | `Text(content, styleProps...)` |
| `Button` | `Button(label, onClick, styleProps...)` — also `ButtonWithEvent(label, event, fn, ...)` |
| `Input` | `Input(value, placeholder, onChange, ...)` — also `InputWithSubmit`, `InputPassword`, `NumericInput`, `TextArea` |
| `Checkbox` | `Checkbox(checked, onToggle, ...)` |
| `Image` | `Image(src, styleProps...)` |
| `CameraView` | `CameraView(props...)` with `OnCapture`, `WithOverlay`, `SetFacing`, ... |
| `Modal` | `Modal(Visible(b), OnDismiss(fn), Backdrop(color), ...)` |
| `TabView` | native tab bar — prefer the [`components.Tabs`](../components.md#tabs) facade |

Inputs are **controlled**: you pass the current value in and receive changes
through `onChange`; the value on screen is whatever your state says it is.
The [tutorial](../tutorial-todo.md) covers the echo/rewrite contract in depth.

## Conditional rendering

Render logic stays declarative with the conditional helpers:

```go
core.If(user != "", core.Text("Welcome, "+user))

core.IfElse(loading.Get(),
    core.Text("Loading…"),
    core.Text("Ready"))

core.Match(status.Get(),
    core.Case("ok",    core.Text("✅")),
    core.Case("error", core.Text("❌")),
    core.Default[string](core.Text("…")))

core.MatchBool(
    core.When(isGuest, guestBanner),
    core.When(isAdmin, adminPanel),
    core.Otherwise(memberView))
```

!!! warning "Conditionals hide views, not hooks"
    `If`/`Match` decide which **view** renders — but a view that calls
    `NewState` and renders only sometimes violates the
    [rules of hooks](state-and-hooks.md#the-rules-of-hooks). Keep hook calls
    unconditional; let conditionals choose between hook-free views, or give
    each branch its own `ctx.Scope`.

## Lists and keys

`For` renders a slice into a sibling list:

```go
core.List(
    core.For(todos, func(t Todo, i int) core.View {
        return core.Keyed(fmt.Sprintf("todo-%d", t.ID), todoRow(t))
    }),
)
```

`Keyed(key, child)` stamps an identity onto the rendered node. The
reconciler uses it to detect when the occupant of a list slot changed —
diffing across different keys would leak state between logically distinct
rows (the classic un-keyed-list bug). Keys must be **unique among siblings**;
[debug mode](debug-mode.md) flags duplicates.

Derive keys from data identity (an ID), never from the index — deletion
shifts indices.

## Navigation

`Navigator` renders a route stack; the stack lives on the context tree, so
one app has exactly one:

```go
core.Navigator(HomeScreen)          // root view

core.Push(ctx, DetailsScreen)       // push a route
core.Pop(ctx)                       // back
core.Replace(ctx, OtherScreen)      // swap the top
core.Reset(ctx, HomeScreen)         // clear the stack to one route
```

Routes are `func(*core.Context) core.View` — plain view functions.

!!! note
    `core.Reset(ctx, route)` (navigation) is unrelated to `ctx.Reset()`
    (the internal hook-cursor reset performed at each pass boundary).

## Composition patterns

Build screens out of small functions of their inputs:

```go
func todoRow(t Todo, setDone func(int, bool)) core.View {
    return core.Keyed(fmt.Sprintf("todo-%d", t.ID), core.Row(
        core.Checkbox(t.Done, func(v bool) { setDone(t.ID, v) }),
        core.Text(t.Title, core.FlexGrow(1)),
    ))
}
```

Rows like this are **pure functions of their data** — no hooks — which keeps
them safe to create and discard freely inside `For`. State lives higher up
and flows down as values + closures; see
[State & Hooks](state-and-hooks.md#where-state-should-live).

For reusable widgets with many optional knobs, prefer the struct idiom of the
[widget library](../components.md) — named fields scale where positional
arguments do not, and `View`-typed fields make natural slots.
