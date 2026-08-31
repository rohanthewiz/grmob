# Session: `Reset` for the Navigation Stack

**Session ID:** session_01YWwH2ECtdanGaihAf4wtrE
**Date:** 2026-08-31, ~18:01
**Branch:** master
**Follows:** `2026-0831-1733-error-boundaries.md`

## Goal

Close the last open Core Abstractions item: *"`Reset` for the navigation stack
(`Push` / `Pop` are done)."*

## The item was mis-stated, which changed the whole shape of the work

`core.Reset(ctx, route)` already existed — four lines, no doc comment, no
tests, no docs page. It swapped the stack slice for a single entry and stopped
there. That is not a reset in any sense an app needs:

Route state lived in `ctx.Scope(...)` scopes hanging off the **Navigator's own
context**, shared by every route. `Reset` never touched them. So the log-out
operation left the previous session's screens fully intact — every scope, every
slot, every running ticker — waiting to be re-displayed the moment the user
navigated back into a screen with the same scope key.

Worse, the sharing was a documented footgun rather than a bug: routes had to
remember `ctx.Scope("...")` or they would alias the slots of the screen
underneath, and `examples/social/pages.go` carried a 20-line comment teaching
exactly that. The tripwire was debug mode's cursor-drift check.

**Asked the user which reading to build** — ship the existing stack-only Reset
properly, or make it actually discard route state. They chose the latter.

## Per-frame identity — `core/navigation.go`

Each stack entry became a `routeEntry{id, route}` with a process-unique id, and
`Navigator` renders it into `ctx.disposableScope("nav:frame:" + id)`.

```
host context
 ├─ nav:frame:0   ← HomeScreen's hooks
 └─ nav:frame:3   ← DetailsScreen's hooks   (top of stack, rendered)
```

**Why the id and not something more obvious.** Both alternatives fail, and
one of them fails destructively:

| key | failure |
|---|---|
| frame **depth** | pop Details, push Settings — both are depth 1, so Settings inherits Details' slots and panics on the type assertion |
| the **route function** | Go func values are not comparable; and two pushes of one screen (a chat thread opened from within a chat thread) are legitimately two independent states |

The id gives two properties an app can lean on: a frame keeps its state for as
long as it is on the stack (`Push` → `Pop` restores a screen exactly as left),
and a frame that leaves can never have its state resurrected, because the next
`Push` mints a new id. That second property is what makes `Reset` mean what
logout needs it to mean.

### Disposal is deferred to the render pass

`navigatorState.retired []int` collects the ids of frames that left the stack;
`Navigator` drains it and calls `dropScope` at the top of the next pass.

The indirection is not bookkeeping taste — the two halves of "discard a frame"
have different thread-safety requirements. Removing the entry mutates
`navigatorState` under its mutex, on whatever thread the platform dispatched
the event on. Discarding the state means deleting from `ctx.scopes`, an
unsynchronized map that `Reset`, `auditCursor` and `Scope` all walk during a
pass. Doing that from an event goroutine races all three.

A frame pushed and popped between two passes retires an id whose scope never
existed; `dropScope` on a missing key is a no-op, so nothing special-cases it.
(There is a test for exactly this, including that `retired` is still drained.)

### API surface

| call | effect | frames it removes |
|---|---|---|
| `Push` / `Pop` / `Replace` | as before | discarded (was: retained) |
| `Reset(ctx, route)` | whole stack → one **fresh** frame | all, root included |
| `PopToRoot(ctx) bool` | unwind to the bottom frame | all above the root |
| `StackDepth(ctx) int`, `CanPop(ctx) bool` | observation | — |

`PopToRoot` is not redundant with `Reset(ctx, root)`: those look identical on
screen and differ in the one way that matters — `Reset` re-mints the root and
silently wipes its scroll position, selected tab and form contents. Drill-down
escape wants one; ending a session wants the other. `CanPop` exists so a back
button that would do nothing is not rendered at all, and answers Android's
hardware back the same way.

`Pop` and `PopToRoot` now zero the vacated slice elements before truncating —
the backing array is retained, and a route is a closure that can pin a good
deal of app state.

## Nested cleanup registries — `core/cleanup.go`

A frame owns more than hook slots. `hooks.UseInterval` hands its ticker to
`ctx.OnClose`, which went to the single app-wide registry — so a popped screen
polled for the life of the process, refreshing a screen nobody was looking at.
Leaving that in place would have made `Reset` a half-measure: state discarded,
pollers still firing.

`cleanupRegistry` became a tree. `sub()` mints a nested registry for a frame,
`close()` recurses innermost-first, `detach()` unlinks a discarded one.

```
root registry (app)
 ├─ frame 0 registry   ← Pop/Reset closes + detaches one of these
 └─ frame 1 registry
```

Both directions are tested: an app-wide `Close` must still reach a live
frame's resources, and a dropped frame must not stay linked — without `detach`
an app that navigates all day grows one dead registry per pop, re-closed by
every later `Close`.

`ctx.disposableScope(key)` (unexported) is `Scope` plus its own sub-registry;
`ctx.dropScope(key)` deletes, closes and detaches. `disposableScope` stays
unexported deliberately: handing apps a scope they can silently leak is worse
than making them use the one lifetime the framework actually manages.

## Two bugs found on the way

Neither was the task; both were in the blast radius.

**`Navigator` reset the host context mid-pass.** The old body was
`Render(ctx, current(ctx))`, and `core.Render` calls `ctx.Reset()` — rewinding
the hook cursor of that context *and every context below it*, halfway through a
pass. Harmless while `Navigator` was always the root view, and silently
corrupting the moment it is not: a hook rendered **after** the Navigator gets
handed an index already given to one rendered before it. The test presents it
the way it actually presents in the wild — `interface conversion: string, not
int`. Removed; the driver restarts cursors once per pass, which is the only
place that can be correct.

**`Context.dirty` was a per-context bool.** Polling hosts read it from the root
(WASM's `IsDirty` binding), so a state change anywhere below the root — every
child context, every `UseChildContext` subtree, and now every navigation frame
by construction — set a flag nobody looked at, and the screen simply stopped
updating until an unrelated event forced a pass. Push-based hosts hid it
because the render-manager notification travels a separate, already-shared
path. It is now a `*dirtyFlag` shared like `nav` / `registry` / `cleanup`, with
its own mutex so marking the tree dirty never contends with slot access.

Related: nav mutations moved from bare `MarkDirty` to `RequestRender`. Marking
alone suffices only when a pass is already guaranteed — true for a tap, false
for a deep link resolving in an effect goroutine, a timeout dismissing a
splash, a websocket pushing the user to a call screen.

## Tests — 19 in `core/navigation_test.go`

Driven by hand through `Render(ctx, Navigator(...))` rather than
`render.Manager` (core cannot import render), which is also what makes the
*timing* of disposal assertable: retired frames drop at the start of the next
pass, not at the mutation.

Coverage: Reset replaces the stack and Pop cannot walk back past it; Reset
discards every frame scope; Reset to the same route function still yields fresh
state; PopToRoot keeps the root frame intact and reports whether it popped;
Replace drops the outgoing frame and leaves the one below untouched; a covered
frame survives; two routes holding different types in slot 0 do not alias;
push/pop with no render in between; discarded frames stop their resources while
their neighbours' survive; app-wide `Close` still reaches every frame;
registries do not accumulate over five push/pop cycles; depth and `CanPop`
including the unseeded stack; two apps do not reset each other; `Push` from a
frame context marks the **app** dirty; and the sibling-hook rewind.

The `bump` helper writes the frame's slot directly rather than going through
`State.Set` — a Set works through a closure the route already bound, proving
nothing about which context owns the slot, whereas a direct write fails loudly
the moment frames stop getting their own scope.

### Mutation-checked

Twelve deliberate breaks, each confirmed to fail before the test was kept:

| break | failure |
|---|---|
| routes render into the host context | slot aliasing + 2 core failures + 2 social panics |
| no pruning of retired frames | 5 failures |
| `Reset` reuses the old root's id | `TestResetGivesTheNewRootFreshState` |
| `Pop` does not retire | resources never stopped; registries accumulate |
| `dropScope` skips `close` | popped frame's resource never stopped |
| `dropScope` skips `detach` | registries accumulate |
| frames share the app registry | popping stops a neighbour's resource |
| `close` does not recurse | app-wide Close misses a live frame |
| restore `Navigator`'s `ctx.Reset()` | `interface conversion: string, not int` |
| `PopToRoot` re-mints the root | root state wiped |
| `Replace` keeps the frame id | outgoing state survives |
| per-context dirty flag | `TestNavigationMarksTheAppDirty` |

All three backups diffed clean against the originals afterwards.

## Example and docs

`examples/social` existed to teach the old footgun, so it had to change.
`DetailsPage` dropped its defensive `ctx.Scope("details")` — a plain
`NewState(ctx, 0)` is now correct — and its comment became the history of the
guarantee rather than a warning.

`TestScopedStateSurvivesRemount` asserted the *old leak* (counter still 2 after
pop → push) and became `TestPushedRouteStateIsDiscardedOnPop`, asserting
`Contador: 0`. Asserting the exact 0 rather than "not 2" distinguishes a
genuinely fresh frame from one re-initialized part-way.

`App`'s comment now names the escape hatch, which is real and worth stating:
state that must outlive a frame goes **above** the Navigator — `App`'s own
`ctx` parameter *is* the Navigator's host context, so `ctx.Scope("session")`
captured by the route closures survives any `Reset`.

New `docs/concepts/navigation.md` (mkdocs nav entry added). Every snippet in it
was type-checked by compiling them as a scratch package rather than eyeballed —
which caught `components.Button` (a struct, not a function; `core.Button` is
the constructor).

`state-and-hooks.md`'s "`ctx.Scope` — the right tool for routes/screens"
advice is now obsolete and was rewritten: `Scope` is for a branch that comes
and goes *within* one screen; routes get isolation for free.

## Files touched

- `core/navigation.go` — rewritten (frames, ids, deferred disposal, `PopToRoot`,
  `StackDepth`, `CanPop`, full doc comments)
- `core/navigation_test.go` — new, 19 tests
- `core/cleanup.go` — nested registries (`sub`, `close`, `detach`)
- `core/context.go` — `disposableScope`, `dropScope`, app-wide `dirtyFlag`
- `examples/social/{app.go,pages.go,app_test.go}` — updated to the new model
- `docs/concepts/navigation.md` — new; `mkdocs.yml` nav entry
- `docs/concepts/views.md`, `docs/concepts/state-and-hooks.md` — updated
- `ROADMAP.md` — Navigation line expanded; the In-Progress Core Abstractions
  section is now empty and gone

Gate: `gofmt` clean, `go vet ./...` clean, full suite passes, `go test -race
./core ./render ./hooks ./examples/...` clean, `GOOS=js GOARCH=wasm build`
clean. (`examples/todoapp/store.go` remains unformatted — pre-existing,
untouched.)

## Backlog

**Core Abstractions is now empty.** Remaining, unchanged from last session:

- Theme contrast, and `Variant`'s third consumer.
- The four renderer gaps: iOS flex weights, `AlignItems: "stretch"`, `Image`
  `ContentMode`, native disabled state.
- Forms with validation — `FormField` renders an `Error`, nothing validates.
- Packaging (`grmob build --target=…`).

Two things this session noticed but did not act on:

- **Hooks still have no unmount signal.** A frame is the first lifetime the
  framework can actually dispose of; component-level unmount (a `UseEffect`
  cleanup that runs when a component leaves the tree, not when the app closes)
  remains unsolved.
- **`docs/concepts/architecture.md` does not mention frames.** The pass
  sequence there is still accurate; the context-tree picture is now incomplete.
