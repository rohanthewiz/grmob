package core

import (
	"strconv"
	"sync"
)

// routeEntry is one frame of the navigation stack: the route function plus an
// id that is unique for the life of the app.
//
// The id is what makes route state disposable. Hook slots are positional, so
// a route needs a hook namespace of its own or it aliases whatever the route
// below it allocated; Navigator gives each frame one by rendering it into
// ctx.disposableScope(routeScopeKey(id)). Deriving the key from the id rather
// than from the route function or the frame's depth buys two properties an
// app can rely on:
//
//   - a frame keeps its state for as long as it is on the stack, so
//     Push → Pop returns to a screen exactly as it was left; and
//   - a frame that leaves the stack can never have its state resurrected,
//     because a later Push allocates a fresh id and therefore a fresh scope.
//
// Keying by depth fails the second property in a way that corrupts rather
// than merely surprises: pop DetailsPage, push SettingsPage, and both are
// depth 1 — Settings would inherit Details' slots and read an int where it
// expects a string, which panics on the type assertion. Keying by the route
// function is no better: Go function values are not comparable, and two
// pushes of the same screen (a chat thread opened from within a chat thread)
// are legitimately two frames with two independent states.
type routeEntry struct {
	id    int
	route func(*Context) View
}

// navigatorState is the per-app route stack. It hangs off the context tree
// (one instance per NewContext root, shared by all derived contexts) rather
// than living in a package variable, so two apps in one process each navigate
// independently.
//
// The mutex matters because mutation and consumption run on different
// goroutines: Push/Pop/Replace/Reset are called from event handlers
// (dispatched on the native event path or under the render manager's dispatch
// lock) while Navigator reads the top of the stack during render passes on the
// pump goroutine. The old global slice had no synchronization at all.
type navigatorState struct {
	mu     sync.Mutex
	stack  []routeEntry
	nextID int

	// retired holds the ids of frames that have left the stack since the last
	// render pass collected them; Navigator drops their scopes and drains this
	// on the next pass.
	//
	// The indirection exists because the two halves of "discard a frame" have
	// different thread-safety requirements. Removing the entry is a mutation
	// of this struct, guarded by mu, and happens wherever the event handler
	// runs. Discarding the frame's state means deleting from ctx.scopes, an
	// unsynchronized map that Reset, auditCursor and Scope all walk during a
	// pass — doing that from an event goroutine would race all three. So the
	// mutation records an intent and the render goroutine carries it out.
	//
	// A frame pushed and popped between two passes lands here without its
	// scope ever having been created; dropScope on a missing key is a no-op,
	// which is why nothing needs to special-case that.
	retired []int
}

func newNavigatorState() *navigatorState {
	return &navigatorState{stack: make([]routeEntry, 0)}
}

// routeScopeKey names the scope holding one frame's hook state. The prefix is
// namespaced so it cannot collide with an app's own ctx.Scope("...") keys on
// the same context — the Navigator's host context is ordinary app territory.
func routeScopeKey(id int) string {
	return "nav:frame:" + strconv.Itoa(id)
}

// newEntryLocked mints a frame for route. Callers must hold n.mu.
func (n *navigatorState) newEntryLocked(route func(*Context) View) routeEntry {
	e := routeEntry{id: n.nextID, route: route}
	n.nextID++
	return e
}

// retireLocked marks a frame's state for disposal on the next render pass.
// Callers must hold n.mu.
func (n *navigatorState) retireLocked(entries ...routeEntry) {
	for _, e := range entries {
		n.retired = append(n.retired, e.id)
	}
}

// takeTop seeds the stack with initial on first use, then returns the frame to
// render along with the ids whose scopes the caller must now drop.
//
// Seeding lazily (rather than in newNavigatorState) is what lets the initial
// route be a property of the Navigator view instead of the context: a context
// is constructed by the host, which has no idea what the app's root screen is.
func (n *navigatorState) takeTop(initial func(*Context) View) (routeEntry, []int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.stack) == 0 {
		n.stack = append(n.stack, n.newEntryLocked(initial))
	}
	retired := n.retired
	n.retired = nil
	return n.stack[len(n.stack)-1], retired
}

// Navigator renders the top of the route stack, seeding the stack with initial
// the first time it renders. It emits no wrapper node of its own — the tree it
// returns is the route's tree — so a Navigator can sit anywhere a view can.
//
// Each frame renders into its own scope of the host context, which has three
// consequences worth knowing:
//
//   - Routes may use hooks freely. NewState in a pushed route claims slot 0 of
//     that frame, not slot 0 of whatever screen is underneath it.
//   - A route's state, and any background resource its hooks started, is
//     discarded when its frame leaves the stack (Pop, Replace, Reset).
//   - State that must outlive a frame belongs above the Navigator. Routes are
//     closures, so the usual move is to capture the context the Navigator
//     itself renders into and keep the state in a scope of that.
//
// Note that Navigator does not call ctx.Reset(): cursors are restarted once
// per pass by the render driver, before the root render. A second, partial
// Reset from inside the tree would rewind the cursor of every context at or
// below this one mid-pass — harmless when the Navigator is the root view and
// silently corrupting when it is not, since siblings rendered before it have
// already consumed slots that the rewind hands out again.
func Navigator(initial func(*Context) View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		entry, retired := ctx.nav.takeTop(initial)

		// Deferred disposal, executed here because this is the render
		// goroutine — see navigatorState.retired for why the mutations could
		// not do it themselves.
		for _, id := range retired {
			ctx.dropScope(routeScopeKey(id))
		}

		frame := ctx.disposableScope(routeScopeKey(entry.id))
		return entry.route(frame).Render(frame)
	})
}

// Push adds a route on top of the stack. The screen underneath keeps its state
// and is restored intact by the matching Pop.
//
// Like every mutation here it ends in RequestRender rather than a bare
// MarkDirty, which these used to do. Marking alone is enough only when a pass
// is already guaranteed to follow — true for a tap, since the native dispatch
// path re-renders on the way out, and false for a navigation triggered from
// anywhere else: an effect goroutine resolving a deep link, a timeout
// dismissing a splash screen, a websocket pushing the user to a call screen.
// Those marked the tree dirty and then waited for an unrelated event to
// notice.
func Push(ctx *Context, route func(*Context) View) {
	ctx.nav.mu.Lock()
	ctx.nav.stack = append(ctx.nav.stack, ctx.nav.newEntryLocked(route))
	ctx.nav.mu.Unlock()
	ctx.RequestRender()
}

// Pop removes the top route, discarding its state, and reveals the one below.
// It is a no-op at the root — the stack is never left empty, because Navigator
// has nothing to render then.
func Pop(ctx *Context) {
	n := ctx.nav
	n.mu.Lock()
	popped := len(n.stack) > 1 // the root route is never popped
	if popped {
		last := len(n.stack) - 1
		n.retireLocked(n.stack[last])
		// Zero the vacated element before truncating: the slice keeps its
		// backing array, and a route is a closure that can capture a good deal
		// of app state. Leaving it there pins that state until the slot is
		// overwritten by a later Push, which may be never.
		n.stack[last] = routeEntry{}
		n.stack = n.stack[:last]
	}
	n.mu.Unlock()
	if popped {
		ctx.RequestRender()
	}
}

// Replace swaps the top route for another without changing the stack depth,
// discarding the outgoing route's state. Use it for a step that should not be
// returned to — the "logged in" screen after a login form, so Back skips the
// form rather than showing it again.
func Replace(ctx *Context, route func(*Context) View) {
	n := ctx.nav
	n.mu.Lock()
	replaced := len(n.stack) > 0
	if replaced {
		old := n.stack[len(n.stack)-1]
		n.stack[len(n.stack)-1] = n.newEntryLocked(route)
		n.retireLocked(old)
	}
	n.mu.Unlock()
	if replaced {
		ctx.RequestRender()
	}
}

// Reset discards the entire stack and starts over with route as the only
// frame. This is the log-out / onboarding-complete operation: every frame's
// hook state is thrown away and every background resource its hooks started is
// stopped, so nothing from the previous session survives to be re-displayed.
//
// The new root is a fresh frame even when route is the same function the old
// root ran, which is the point — resetting to the login screen must not show
// the previous tenant's half-filled form.
//
// What Reset does not touch is state the app deliberately kept outside the
// stack: hooks on the context hosting the Navigator, package-level stores, the
// database. Those outlive navigation by construction, and clearing them is the
// app's call, not the router's.
func Reset(ctx *Context, route func(*Context) View) {
	n := ctx.nav
	n.mu.Lock()
	n.retireLocked(n.stack...)
	n.stack = []routeEntry{n.newEntryLocked(route)}
	n.mu.Unlock()
	ctx.RequestRender()
}

// PopToRoot unwinds to the bottom of the stack, discarding the state of every
// frame above it, and returns whether anything was popped.
//
// It differs from Reset in exactly one way, and it is the way that matters:
// the root frame is the one already there, state and all. Reset(ctx, root)
// would look identical on screen and quietly reset the root's scroll position,
// selected tab and form contents. Reach for PopToRoot to escape a deep
// drill-down ("Done" out of a five-level settings tree), and for Reset to end
// a session.
func PopToRoot(ctx *Context) bool {
	n := ctx.nav
	n.mu.Lock()
	popped := len(n.stack) > 1
	if popped {
		n.retireLocked(n.stack[1:]...)
		// Same reason as Pop: clear the vacated elements so the truncation
		// actually releases the route closures they held.
		for i := 1; i < len(n.stack); i++ {
			n.stack[i] = routeEntry{}
		}
		n.stack = n.stack[:1]
	}
	n.mu.Unlock()
	if popped {
		ctx.RequestRender()
	}
	return popped
}

// StackDepth reports how many frames are on the stack. It is 0 before the
// Navigator's first render, because the stack is seeded lazily at that point,
// and at least 1 afterwards.
func StackDepth(ctx *Context) int {
	ctx.nav.mu.Lock()
	defer ctx.nav.mu.Unlock()
	return len(ctx.nav.stack)
}

// CanPop reports whether there is a screen to go back to, which is what a back
// button or a hardware-back handler needs in order to decide between popping
// and exiting the app. Pop is a safe no-op when this is false; the point of
// asking first is to avoid rendering a control that does nothing.
func CanPop(ctx *Context) bool {
	return StackDepth(ctx) > 1
}

// Render renders view into ctx after restarting ctx's hook cursors. It is the
// entry point for a host driving passes by hand; render.Manager does the same
// two steps itself (with the debug pass boundary around them) and does not
// call this.
func Render(ctx *Context, view View) *Node {
	ctx.Reset()
	return view.Render(ctx)
}
