# Navigation

`core.Navigator` renders a stack of routes and shows the top one. A route is
just a view function — `func(*core.Context) core.View` — so there is no route
table, no registration step, and no string identifiers to keep in sync.

```go
func App(ctx *core.Context) core.View {
    return core.Navigator(HomeScreen)
}

func HomeScreen(ctx *core.Context) core.View {
    return core.Button("Details", func() {
        core.Push(ctx, DetailsScreen)
    })
}
```

The stack lives on the context tree, so one app has exactly one — and two apps
in one process navigate independently.

| Call | Effect | State of the frames it removes |
|---|---|---|
| `Push(ctx, route)` | adds a frame on top | — |
| `Pop(ctx)` | removes the top frame | discarded |
| `Replace(ctx, route)` | swaps the top frame | discarded |
| `PopToRoot(ctx) bool` | unwinds to the bottom frame | discarded |
| `Reset(ctx, route)` | replaces the whole stack | discarded, including the old root |
| `StackDepth(ctx) int` | frames on the stack | — |
| `CanPop(ctx) bool` | `StackDepth > 1` | — |

`Pop` is a no-op at the root: the stack is never left empty, because a
Navigator with nothing on the stack has nothing to render.

!!! note
    `core.Reset(ctx, route)` (navigation) is unrelated to `ctx.Reset()`, the
    internal hook-cursor restart performed at each pass boundary.

## Every frame owns its state

Each frame renders into its own hook namespace — a scope of the context the
Navigator itself renders into, keyed by an id minted when the frame is pushed.

```
host context
 ├─ nav:frame:0   ← HomeScreen's hooks
 └─ nav:frame:3   ← DetailsScreen's hooks   (top of stack, rendered)
```

Two consequences follow, and both are things you can rely on:

**A route may use hooks freely.** `core.NewState` in a pushed route claims slot
0 *of that frame*, not slot 0 of the screen underneath it. Routes cannot alias
each other's slots, so nothing is required of you to keep them apart.

**A frame that leaves the stack takes its state with it.** Push a screen, pop
it, push the same screen again, and it starts fresh — a new frame, a new id, a
new scope. This is what makes `Reset` mean what an app needs it to mean at
logout: the previous session's screens are gone, not merely covered.

Frames still on the stack are untouched. Push a detail screen over a tabbed
shell and pop back, and the shell returns with its selected tab, scroll
position and form contents intact, because its frame never left.

### Background resources go too

A frame owns more than hook slots. Anything its hooks registered through
`ctx.OnClose` — `hooks.UseInterval`'s ticker, `hooks.UseTimeout`'s pending
timer — is stopped when the frame is discarded, rather than lingering until the
app shuts down.

```go
func LiveScoresScreen(ctx *core.Context) core.View {
    hooks.UseInterval(ctx, func() { refresh() }, 5*time.Second)
    return ...
}
```

Pop that screen and the polling stops. Before per-frame ownership it would have
kept firing for the life of the process, refreshing a screen nobody was
looking at.

## Reset vs. PopToRoot

These look identical on screen and differ in exactly the way that matters.

```go
core.Reset(ctx, HomeScreen)   // a NEW home frame — home's own state is wiped
core.PopToRoot(ctx)           // the EXISTING home frame — state intact
```

Use `PopToRoot` to escape a deep drill-down ("Done" out of a five-level
settings tree). Use `Reset` to end a session — logging out, finishing
onboarding, switching accounts — where showing the previous user's half-filled
form would be a bug, not a convenience.

## Keeping state across a Reset

`Reset` discards frames. State that must outlive them belongs **above** the
Navigator, on the context the Navigator renders into. Routes are closures, so
capturing it is the whole technique:

```go
func App(ctx *core.Context) core.View {
    // ctx here is the Navigator's host context: outside every frame.
    session := ctx.Scope("session")

    return core.Navigator(func(*core.Context) core.View {
        return LoginScreen(session)
    })
}
```

Package-level stores, a `bytdb` handle, and anything else the app owns are
unaffected by navigation by construction. Clearing those at logout is the app's
job — the router does not know what they mean.

## Rendering a back button

`CanPop` exists so a control that would do nothing is not rendered at all:

```go
func Header(ctx *core.Context, title string) core.View {
    return core.Row(
        core.MaybeProp(core.CanPop(ctx),
            core.Button("Back", func() { core.Pop(ctx) })),
        core.Text(title),
    )
}
```

The same check answers Android's hardware back button: pop if `CanPop`,
otherwise let the platform close the app.

## Where the Navigator sits

`Navigator` emits no wrapper node — the tree it returns is the route's tree —
so it can sit anywhere a view can, including inside a screen that draws
chrome around it.

```go
core.Column(
    Header(ctx, "My App"),
    core.Navigator(HomeScreen),   // only this region swaps
)
```

Sibling hooks around it are unaffected: `Navigator` does not restart hook
cursors, which the render driver already does once per pass.

## Re-render behavior

Every mutation ends in `RequestRender`, the same notification `State.Set`
fires. Navigating from a tap works, and so does navigating from anywhere
else — an effect goroutine resolving a deep link, a timeout dismissing a
splash screen, a websocket pushing the user to a call screen.

Frame disposal is deferred to the next render pass rather than done inside the
mutation. Navigation is called from event handlers on whatever thread the
platform dispatches on, while the scope table is walked by every render pass;
recording the intent and acting on it during the pass is what keeps the two
from racing.
