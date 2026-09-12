---
name: grmob-native-mobile-go
description: Build native mobile apps (Android, iOS), browser apps and HTML exports in pure Go with the GrMob framework — declarative views, positional hook state, and a tree diff that every host applies as patches.
---

# GrMob — AI Agent Documentation

> Native mobile UI in pure Go. No templates, no XML, no Swift, no Kotlin, no JS.

```go
import "github.com/rohanthewiz/grmob/core"
```

## The mental model

An app is a **function from a context to a view**, called again on every pass.
It returns a description of the UI; it never touches the screen.

```
App(ctx) ──▶ View.Render(ctx) ──▶ *core.Node tree
                                        │
                              reconcile.Diff(old, new)
                                        │
                                   []reconcile.Patch
                          ┌─────────────┼─────────────┬──────────────┐
                      Compose        SwiftUI     JS runtime      htmlout
                      (Android)       (iOS)      (browser)     (HTML file)
```

Four consequences worth holding on to:

1. **The same app function runs on every target**, including inside a plain Go
   test. There is nothing platform-specific in application code.
2. **Nodes are immutable once rendered.** The diff treats pointer equality as
   proof a subtree is unchanged, so mutating a node after its pass has returned
   makes the change invisible rather than merely late.
3. **State lives in the context, addressed by call order** — not by name. This
   is the single biggest source of bugs; see the rules below.
4. **`State.Set` is the whole update path.** Write state, and the next pass's
   diff reaches the screen. Never try to "refresh" anything by hand.

## A complete app

```go
package counter

import (
    "fmt"

    "github.com/rohanthewiz/grmob/core"
    "github.com/rohanthewiz/grmob/mobile"
)

// Registers with the bridge at link time. This is the whole integration
// contract for the native shells and the browser host.
func init() { mobile.Register(core.NewContext(), App) }

// gomobile cannot bind a func-typed symbol, so without an exported function of
// a bindable shape the linker drops this package — and the init above with it.
// Every app package needs one such symbol.
func AppName() string { return "Counter" }

func App(ctx *core.Context) core.View {
    count := core.NewState(ctx, 0)

    return core.Column(
        core.Padding(24),
        core.Gap(12),
        core.Text(fmt.Sprintf("Count: %d", count.Get()), core.FontSize(28)),
        core.Row(
            core.Gap(8),
            core.Button("−", func() { count.Set(count.Get() - 1) }),
            core.Button("+", func() { count.Set(count.Get() + 1) }),
        ),
    )
}
```

## Views and containers

A view is anything with `Render(ctx *core.Context) *core.Node`. Containers take
a single variadic `...core.PropsAndChildren` (which is `any`) and sort styles
from children at runtime — so **props and children are interleaved freely**:

```go
core.Column(
    core.Padding(16),          // a StyleProp
    core.Text("Title"),        // a View
    core.Gap(8),               // another StyleProp, after a child: fine
    core.Text("Body"),
)
```

| Containers | Leaves | Structure |
|---|---|---|
| `Column` `Row` `Box` `Card` `Scroll` `List` `ZStack` `SafeArea` | `Text` `Image` `Divider` `Spacer` | `Fragment` `Keyed` `For` `If` `IfElse` `Match` |

Leaf widgets take **typed positional arguments first**, then styles:

```go
core.Text("hello", core.FontSize(16), core.TextColor("#333"))
core.Button("Save", onSave)
core.Input(value, "you@example.com", onChange)
core.Checkbox(checked, func(b bool) { /* ... */ })
core.Slider(v, 0, 100, func(f float64) { /* ... */ })
core.Select(value, opts, onChange)
core.Image("logo.png", core.Width("100%"))
```

### Composition

Extract a screen into a function; there is no component registration step.

```go
func header(title string) core.View {
    return core.Row(core.Padding(12), core.Text(title, core.FontWeight(core.Bold)))
}
```

Use `core.ComponentFunc` when a piece needs its **own** render function value
(and therefore its own hook slots relative to where it is called):

```go
core.ComponentFunc(func(ctx *core.Context) *core.Node {
    n := core.NewState(ctx, 0)
    return core.Text(fmt.Sprint(n.Get())).Render(ctx)
})
```

## State and the rules of hooks

`core.NewState[T](ctx, initial) State[T]` claims the slot at the context's
current cursor. Slots are **positional**: identified by the order the calls
happen in, so the Nth `NewState` call of a pass is the Nth slot.

```go
name  := core.NewState(ctx, "")   // slot 0
email := core.NewState(ctx, "")   // slot 1
```

**The rules, and what breaks when they are broken:**

```go
// WRONG — the state call is conditional, so every hook after it shifts onto
// a neighbour's slot on the pass where the condition flips.
if expanded {
    detail := core.NewState(ctx, "")
}

// RIGHT — call unconditionally, use conditionally.
detail := core.NewState(ctx, "")
if expanded { /* read detail */ }
```

- Call every hook on **every** pass, in the **same order**.
- Never call a hook inside `if`, `for`, `switch`, or after an early return.
- A conditional **view** is fine (`core.If`) — it is a conditional **hook** that
  is not. Widgets that hold hooks (`comps.Accordion`, `DatePicker`) count
  as hook callers and must be rendered unconditionally too.
- `core.SetDebugMode(true)` detects the drift and reports it; see Debug mode.

**Updating state** — treat values as immutable and replace them:

```go
items := core.NewState(ctx, []Todo{})

// WRONG: mutating the slice in place. The slot still holds the same header,
// so a pointer-equality fast path can conclude nothing changed.
list := items.Get()
list[0].Done = true

// RIGHT: copy, change the copy, Set it.
list := append([]Todo(nil), items.Get()...)
list[0].Done = true
items.Set(list)
```

`Set` is safe from **any goroutine** — a timer, an HTTP response, a channel
reader. It writes the slot and nudges the render loop, which coalesces a burst
of writes into one pass over the settled state.

### State lives high

State that must outlive a screen belongs above the thing that comes and goes.
Routes are closures, so the usual move is to capture the context the
`Navigator` renders into:

```go
func App(ctx *core.Context) core.View {
    cart := core.NewState(ctx, []Item{})       // survives every navigation
    return core.Navigator(func(*core.Context) core.View {
        return productList(ctx, cart)          // capture, don't re-declare
    })
}
```

### Scoping

`ctx.Scope(key)` gives a stable named sub-context (cursor independent of its
parent's), and `core.UseChildContext(ctx)` gives a positional one. Reach for
`Scope` when a subtree's hook count varies and you want it not to disturb
siblings.

## Hooks

```go
import "github.com/rohanthewiz/grmob/hooks"

hooks.UseEffect(ctx, fn)                       // every pass
hooks.UseEffect(ctx, fn, userID)               // when userID changes
hooks.UseInterval(ctx, tick, time.Second)      // self-cancelling on close
hooks.UseTimeout(ctx, fire, 300*time.Millisecond)
v   := hooks.UseMemo(ctx, expensive, dep)
s, dispatch := hooks.UseReducer(ctx, reduce, initial)
d   := hooks.UseDebounce(ctx, 250*time.Millisecond)

loc  := hooks.UseLocation(ctx)                 // read-only host reports
life := hooks.UseLifecycle(ctx)
st   := hooks.UsePermission(ctx, permission.Camera)
```

**Dependency comparison is value equality.** A func value, or a slice built
fresh each pass, compares unequal every time — which turns a conditional effect
into an unconditional one silently, since it still does the right thing, just
far more often than intended.

`UseInterval` / `UseTimeout` re-bind their function every pass, so the callback
that fires sees **current** state, not the state of the pass that started it.

Cleanup: `ctx.OnClose(fn)`. Contexts close when their frame leaves the
navigation stack or when `Manager.Close` runs, which is what makes a ticker
inside a screen safe to walk away from.

## Styling and theming

Individual props, or a whole `Style` value:

```go
core.Box(
    core.Padding(16), core.BorderRadius(8),
    core.BackgroundColor("#fff"), core.Gap(8),
    core.Width("100%"), core.Justify(core.JustifyBetween),
)

core.Text("Body", core.UseStyle(ctx.Theme().Typography.Body))
```

**`UseStyle`'s one trap:** the merge rule is "a set field wins, an unset field
is ignored", and a zero value is indistinguishable from unset. So `UseStyle`
can **add** but never **clear**:

```go
core.UseStyle(core.Style{FontSize: 0})   // does NOT reset the font size
core.FontSize(0)                          // this does
```

Theme a subtree, and read the theme rather than hard-coding colors:

```go
core.WithTheme(core.MaterialTheme, screen())
ctx.Theme().Colors.Primary    // roles, not hex, in widget code
ctx.Theme().Typography.Body   // Title / Subtitle / Body / Caption
ctx.Theme().Spacing           // never a magic number in widget code
```

Bundled: `core.DefaultTheme`, `core.MaterialTheme`, `core.AmberTheme`.
Override by copying a base and editing the copy.

### Accessibility

Accessibility props are ordinary `StyleProp`s, so they compose like any other:

```go
core.Button("Close", onClose, core.AccessibilityLabel("Close dialog"))
```

**Do not restate a role the node type already owns.** `core.Button` exports as
a `<button>` and builds a real control on both natives; `core.Modal` writes
`role="dialog"` itself. `RoleButton` exists for the *other* case — a `Box` or
`Row` with an `OnClick`, which every renderer otherwise draws as inert scenery:

```go
core.Box(core.OnClick(open), core.AccessibilityRole(core.RoleButton),
    core.AccessibilityLabel("Open settings"))
```

Landmark, live-region and content roles (`RoleBanner`, `RoleNavigation`,
`RoleStatus`, `RoleHeading`, `RoleLink`, `RoleImg`, `RoleTab`) describe the node
itself and are always the author's to state. Four role families make claims
about their *children* — the tabular set, both collection pairs, and
`RoleTabList` — so a container carrying one has to actually contain what it
says (a `core.List` with a "Load more" footer inside it is not a list).

## Events

There are three prop families, and they differ in what they need:

| Family | Interface | Example |
|---|---|---|
| `StyleProp` | `Apply(*Style)` | `core.Padding(8)`, `core.AccessibilityLabel("x")` |
| `BehaviorProp` | `Apply(*Context, *Node)` | `core.OnClick(fn)` — needs the context's callback registry |
| child views | `core.View` | `core.Text("hi")` |

```go
core.Box(
    core.OnClick(open),                    // tap
    core.OnLongPress(showMenu),
    core.OnTouch(highlight),
    core.OnFocus(onFocus), core.OnBlur(onBlur),
    core.On("custom-event", handler),      // the general form
    core.Text("Settings"),
)
```

Handlers are registered per pass and get an ID; IDs are purged and reissued
every pass, which is why **a callback inside a `Cached` subtree dangles** and
why nothing outside a render may hold one. The host sends the ID back, the
context dispatches, and the pass that follows carries the result.

Handler callbacks run under the render mutex, so a handler observes a settled
tree and every write it makes is rendered together by the next pass.

## Conditionals and lists

```go
core.If(loading, core.Text("Loading…"))
core.IfElse(signedIn, dashboard(), loginForm())

core.For(todos, func(t Todo, i int) core.View {
    return core.Keyed(t.ID, todoRow(ctx, t))
})

core.Match(status,
    core.Case(StatusOK, core.Text("ok")),
    core.Case(StatusErr, core.Text("failed")),
)
core.MaybeProp(compact, core.Padding(4))   // a conditional prop
```

**Key children of a dynamic list.** Positional diffing means an unkeyed
insertion at the top re-props every row below it. Keys also have a limit worth
knowing: there are no move patches yet, so a key that changes slot is
*replaced*, which is visually right but costs that subtree its transient native
state (focus, scroll offset).

## Navigation

```go
core.Navigator(func(ctx *core.Context) core.View { return home(ctx) })

core.Push(ctx, detail)       // screen underneath keeps its state
core.Pop(ctx)
core.Replace(ctx, other)
core.Reset(ctx, home)        // clear the stack
core.PopToRoot(ctx)
core.CanPop(ctx)             // e.g. whether to draw a back button
```

Each frame renders in its own scope, so routes may use hooks freely, and a
route's state is discarded when its frame leaves the stack.

**Two different `Reset`s — do not confuse them:**

| Call | Meaning |
|---|---|
| `core.Reset(ctx, route)` | navigation: clear the stack, seed with `route` |
| `ctx.Reset()` | rewind the context's hook cursor — **render-driver business.** Calling it from inside a tree corrupts sibling slots mid-pass |

## Forms

```go
import "github.com/rohanthewiz/grmob/forms"

form := forms.UseForm(ctx, forms.Spec{
    Fields: []forms.Field{
        {Name: "email", Rules: []forms.Rule{
            forms.Required("We need an address to reach you"),
            forms.Email(""),                      // "" takes the default message
        }},
        {Name: "password", Rules: []forms.Rule{forms.MinLen(8, "")}},
        {Name: "confirm"},
        {Name: "terms", Rules: []forms.Rule{forms.Accepted("Please accept")}},
    },
    // Cross-field checks see every value at once.
    Validate: func(v forms.Values) map[string]string {
        if v["confirm"] != v["password"] {
            return map[string]string{"confirm": "The two passwords differ"}
        }
        return nil
    },
})

comps.FormField{
    Label: "Email",
    Error: form.Error("email"),                   // "" until the user should see it
    Input: form.Input("email", "you@example.com"),
}
```

`form.Input(name, placeholder)` binds value **and** onChange to the same name in
one call. Also: `form.Checkbox`, `form.InputWithSubmit`, `form.Error(name)`,
`form.Errors()`, `form.Blurred(name)`.

Rules run in order and **the first complaint wins**, because a field shows one
line of feedback — so `Required` belongs first; it is the only rule that speaks
about an empty value. Field rules beat `Validate` messages for the same field,
being the more specific complaint. A `Validate` key that names no field is a
form-level error, readable with `form.Error("form")`. `Field.Initial` seeds a
value once and is not re-applied, so a field the user cleared stays cleared.
Rules and `Validate` must be pure and must not call back into the `Form`.
The full rule set: `Required` `MinLen` `MaxLen` `Email` `Pattern` `Accepted`.

The form — not the widget — decides *when* an error becomes visible. Do not
build that logic yourself.

## The widget library

```go
import "github.com/rohanthewiz/grmob/comps"
```

Struct widgets configured by named field, with `core.View` composition slots:

```go
comps.Screen{
    Children: []core.View{
        comps.AppBar{Title: "Account", Subtitle: "4 cards"},
        comps.Card{
            Title:  "Balance",
            Body:   core.Text("$42.00"),
            Footer: comps.Badge{Text: "verified"},
        },
        comps.SegmentedControl{Labels: tabs, Selected: sel, OnSelect: pick},
    },
}
```

Available: `Screen` `AppBar` `Card` `Button` `InputRow` `SearchField`
`SegmentedControl` `ListRow` `GroupedList[T]` `DataTable[T]` `Separator`
`Avatar` `ProgressBar` `Chip` `ChipStrip` `Badge` `Banner` `FormField`
`Accordion` `Collapse` `CollapseBand` `Tabs` `Calendar` `DatePicker`
`EmptyState` `Skeleton` `Pagination` `LoadMore` `StatTile` `MapPanel`
`StaticMap` `CodeEditor` `RichTextEditor` `RichToolbar` `Compass`.

`Screen` is the scaffold: `Children`, `Scroll`, `KeyboardAware`, `Gap`, `Fill`,
`Style`. Leave `Scroll` false when the screen already contains its own
scrolling region — a scroll view inside a scroll view fights for the same drag
on both natives.

Widgets take their look from `ctx.Theme()` and accept `Style` overrides (a
`[]core.StyleProp` field). When a widget offers both a simple path and a slot
(`Card.Title` vs `Card.Header`), the **slot wins** if both are set.

## Caching

```go
core.Cached(expensiveStaticSubtree())
```

`Cached` renders once and replays the same `*Node` forever, which the diff
short-circuits on pointer identity. The constraints are strict, and debug mode
enforces them:

- Construct it **once** — `core.Cached(x)` inside a render call caches nothing.
- No hooks inside.
- No callbacks inside (their IDs are purged each pass and would dangle).
- Nothing that reads changing state.

Reach for it as a profiling response, not a default.

## Debug mode

```go
core.SetDebugMode(true)   // development builds and tests; costs nothing when off
```

It flags exactly the failures that are otherwise silent: cursor drift, duplicate
sibling keys, dangling callback references, handler panics, misused caches.

```go
core.ClearConcerns()
// ... drive some passes ...
if c := core.DumpConcerns(); c != "" {
    t.Fatalf("debug concerns raised:\n%s", c)
}
```

Asserting the concern list is **empty** is the standard test shape.

## Testing — the fastest loop there is

No simulator, no browser, no device. Drive the real engine from a Go test:

```go
func TestMain(m *testing.M) {
    core.SetDebugMode(true)   // audit every pass this package drives
    m.Run()
}

func TestIncrement(t *testing.T) {
    mgr := render.New(core.NewContext(), App)
    defer mgr.Close()

    tree := mgr.RenderInitial()             // the whole tree, as JSON
    id := findCallbackID(tree, "+")
    patches := mgr.DispatchCallback(id)     // tap it; get the diff back

    if !strings.Contains(patches, "Count: 1") {
        t.Fatalf("got %s", patches)
    }
    if c := core.DumpConcerns(); c != "" {
        t.Fatalf("concerns:\n%s", c)
    }
}
```

`render.Manager` is the app-lifetime owner: `New`, `SetListener`,
`RenderInitial`, `Dispatch(label, fn)`, `DispatchCallback` /
`DispatchTextCallback` / `DispatchBoolCallback` / `DispatchIntCallback`,
`RenderAgain`, `RenderAndGetPatches`, `Close`.

Every pass is serialised under one mutex, so a handler always observes a settled
tree and its writes are rendered together by the pass that follows.

For a snapshot you can pin byte for byte:

```go
html := htmlout.ExportHTML(node)   // standalone HTML document
json := jsonout.Export(node)       // the same shape the bridges send
```

## Targets

| Target | How |
|---|---|
| Go test | `render.New` — start here for everything |
| Browser | `GOOS=js GOARCH=wasm go build -o main.wasm ./wasm`, then `./build.sh` |
| Android | gomobile bind → `.aar`, Compose renderer applies patches |
| iOS | gomobile bind → framework, SwiftUI renderer applies patches |
| HTML | `htmlout.ExportHTML` |

The native bridge (`package mobile`) narrows everything to strings, bools and a
one-method interface, because gomobile cannot bind functions, generics or maps:

```go
mobile.Register(core.NewContext(), App)
mobile.SetListener(l)          // Go → native push channel for async updates
mobile.RenderInitial()         // string
mobile.TriggerCallback(id)     // string of patches, synchronously
mobile.SetDataDir(path)        // call before anything opens a store
```

`SetDataDir` timing matters: `init` runs before the host can call it, so open
persistent stores lazily, not in `init`.

Patches arrive on two paths — the synchronous return of a `Trigger*` call, and
`PatchListener` pushes for timers and goroutines. Each pass delivers its diff on
exactly one of them, so a renderer applies everything it receives in arrival
order and stays consistent. `ApplyPatches` is called from a background
goroutine: hop to the UI thread before touching views.

## Pitfalls, in the order they bite

1. **A hook inside a conditional.** Shifts every later slot. Turn on debug mode.
2. **Mutating state in place** instead of replacing the value.
3. **Mutating a `*Node` after render.** The diff will skip what you changed.
4. **Unkeyed dynamic lists.** Key rows by a stable ID.
5. **`core.Reset` vs `ctx.Reset`.** The second is not yours to call.
6. **`UseStyle` to clear a field.** It cannot; use the individual setter.
7. **Effect deps that are funcs or fresh slices.** They never compare equal.
8. **`Cached` constructed inside a render**, or containing a callback or a hook.
9. **No bindable exported symbol** in an app package, so the linker drops it.
10. **Opening a data store in `init`**, before the host has called `SetDataDir`.

## Where the full documentation is

Narrative guides, and a generated reference of every exported symbol:

```sh
./docs.sh     # serves mkdocs.yml + docs/ with gkdocs at :8000
```

- `docs/getting-started.md`, `docs/tutorial-todo.md`,
  `docs/tutorial-interactive.md`
- `docs/concepts/` — architecture, views, state & hooks, events, forms,
  navigation, styling, reconciliation, caching, error boundaries, debug mode
- `docs/api/` — one page per package, generated from the doc comments by
  `go run ./internal/apidoc/gen` and committed
