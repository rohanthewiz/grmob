package core

import (
	"strconv"
	"strings"
	"sync"
	"testing"
)

// --- helpers -------------------------------------------------------------
//
// Routes are rendered through Render(ctx, Navigator(...)) rather than through
// render.Manager: core cannot import render, and every property under test
// here (which frame is on top, whose slots a route reads, when a scope is
// dropped) is visible in the returned node tree. Passes are driven by hand
// because the timing of the drop is part of the contract — retired frames are
// disposed of at the START of the next pass, not at the moment of the
// mutation.

// counterRoute is a route with one hook slot, rendering its own value. Reading
// the rendered text is how the tests tell a preserved frame from a fresh one.
func counterRoute(label string) func(*Context) View {
	return func(ctx *Context) View {
		n := NewState(ctx, 0)
		return Text(label + ":" + strconv.Itoa(n.Get()))
	}
}

// bump adds times to the counter of whichever route is on top, by writing the
// frame's slot directly.
//
// Going through the slot rather than through State.Set is what keeps the test
// honest about *where* the state lives: a Set would work through a closure the
// route already bound, proving nothing about which context owns the slot,
// whereas this fails loudly (index out of range, or a wrong-typed slot) the
// moment frames stop getting their own scope. It renders first, because the
// slot does not exist until the route that claims it has run.
func bump(ctx *Context, app View, times int) {
	Render(ctx, app)
	top := topFrameCtx(ctx)
	top.lock.Lock()
	top.slots[0] = top.slots[0].(int) + times
	top.lock.Unlock()
}

// topFrameCtx returns the scope Navigator is currently rendering into. It
// mirrors Navigator's own key derivation rather than guessing, so a change to
// the key scheme breaks the helper loudly instead of silently testing nothing.
func topFrameCtx(ctx *Context) *Context {
	ctx.nav.mu.Lock()
	id := ctx.nav.stack[len(ctx.nav.stack)-1].id
	ctx.nav.mu.Unlock()
	return ctx.scopes[routeScopeKey(id)]
}

func rendered(t *testing.T, ctx *Context, app View) string {
	t.Helper()
	n := Render(ctx, app)
	content, _ := n.Props["content"].(string)
	return content
}

// frameScopes lists the nav frame scopes currently held on the context, which
// is the direct observation of "was the discarded frame's state released".
func frameScopes(ctx *Context) []string {
	var out []string
	for k := range ctx.scopes {
		if strings.HasPrefix(k, "nav:frame:") {
			out = append(out, k)
		}
	}
	return out
}

// --- stack behavior ------------------------------------------------------

func TestResetReplacesTheWholeStack(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("home"))

	Render(ctx, app)
	Push(ctx, counterRoute("a"))
	Push(ctx, counterRoute("b"))
	if got := StackDepth(ctx); got != 3 {
		t.Fatalf("stack depth before reset = %d, want 3", got)
	}

	Reset(ctx, counterRoute("login"))
	if got := rendered(t, ctx, app); got != "login:0" {
		t.Errorf("Reset did not render the new root, got %q", got)
	}
	if got := StackDepth(ctx); got != 1 {
		t.Errorf("stack depth after Reset = %d, want 1", got)
	}
	if CanPop(ctx) {
		t.Errorf("CanPop is true after Reset: the new root must be the only frame")
	}

	// Pop must not be able to walk back into the discarded history.
	Pop(ctx)
	if got := rendered(t, ctx, app); got != "login:0" {
		t.Errorf("Pop after Reset escaped the new root, got %q", got)
	}
}

// TestResetDiscardsEveryFrameScope is the heart of the feature: Reset is the
// log-out operation, so the state of every frame it removed must be released,
// not merely unreachable through the stack.
func TestResetDiscardsEveryFrameScope(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("home"))

	Render(ctx, app)
	Push(ctx, counterRoute("a"))
	Render(ctx, app)
	Push(ctx, counterRoute("b"))
	Render(ctx, app)
	if got := len(frameScopes(ctx)); got != 3 {
		t.Fatalf("expected 3 live frame scopes before reset, got %d", got)
	}

	Reset(ctx, counterRoute("login"))
	// The drop happens on the next pass, by design — see navigatorState.retired.
	Render(ctx, app)

	scopes := frameScopes(ctx)
	if len(scopes) != 1 {
		t.Fatalf("Reset left %d frame scopes (%v), want only the new root", len(scopes), scopes)
	}
}

// TestResetGivesTheNewRootFreshState guards the property that makes Reset
// usable for logging out: resetting to the same route function the old root
// ran must not resurrect the old root's state.
func TestResetGivesTheNewRootFreshState(t *testing.T) {
	ctx := NewContext()
	root := counterRoute("home")
	app := Navigator(root)

	Render(ctx, app)
	bump(ctx, app, 3)
	if got := rendered(t, ctx, app); got != "home:3" {
		t.Fatalf("setup failed, counter = %q", got)
	}

	Reset(ctx, root)
	if got := rendered(t, ctx, app); got != "home:0" {
		t.Errorf("Reset to the same route kept the old frame's state: %q", got)
	}
}

// TestPopToRootKeepsTheRootFrameIntact is the distinction PopToRoot exists to
// draw. Reset(ctx, root) would look identical on screen and silently wipe the
// root's state; PopToRoot returns to the frame that is already there.
func TestPopToRootKeepsTheRootFrameIntact(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("home"))

	Render(ctx, app)
	bump(ctx, app, 2)
	Push(ctx, counterRoute("a"))
	Push(ctx, counterRoute("b"))
	Render(ctx, app)

	if !PopToRoot(ctx) {
		t.Fatalf("PopToRoot reported nothing to pop from depth 3")
	}
	if got := rendered(t, ctx, app); got != "home:2" {
		t.Errorf("PopToRoot did not restore the root frame's state, got %q", got)
	}
	if got := len(frameScopes(ctx)); got != 1 {
		t.Errorf("PopToRoot left %d frame scopes, want 1", got)
	}
	if PopToRoot(ctx) {
		t.Errorf("PopToRoot at the root reported a pop")
	}
}

// TestReplaceSwapsTheTopAndDropsItsState covers the login-form case: the
// replaced screen must not be reachable, and its half-filled state must not be
// waiting for whatever lands on that depth next.
func TestReplaceSwapsTheTopAndDropsItsState(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("home"))

	Render(ctx, app)
	Push(ctx, counterRoute("form"))
	Render(ctx, app)
	bump(ctx, app, 4)

	Replace(ctx, counterRoute("welcome"))
	if got := rendered(t, ctx, app); got != "welcome:0" {
		t.Errorf("Replace produced %q, want a fresh welcome frame", got)
	}
	if got := StackDepth(ctx); got != 2 {
		t.Errorf("Replace changed the stack depth to %d, want 2", got)
	}
	if got := len(frameScopes(ctx)); got != 2 {
		t.Errorf("Replace left %d frame scopes, want 2 (root + the new top)", got)
	}

	// The frame underneath was never touched.
	Pop(ctx)
	if got := rendered(t, ctx, app); got != "home:0" {
		t.Errorf("Replace disturbed the frame below it, got %q", got)
	}
}

// TestFrameStateSurvivesWhileOnTheStack is the other half of the contract: a
// frame that is merely covered, not removed, comes back as it was. Without it
// "discard on pop" could be implemented by discarding everything.
func TestFrameStateSurvivesWhileOnTheStack(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("home"))

	Render(ctx, app)
	bump(ctx, app, 5)

	Push(ctx, counterRoute("detail"))
	if got := rendered(t, ctx, app); got != "detail:0" {
		t.Fatalf("pushed route did not render, got %q", got)
	}

	Pop(ctx)
	if got := rendered(t, ctx, app); got != "home:5" {
		t.Errorf("covered frame lost its state, got %q", got)
	}
}

// TestRoutesDoNotShareHookSlots is the aliasing bug per-frame scopes exist to
// prevent. The two routes hold different types in slot 0; sharing a context
// would make the second route's Get panic on the type assertion, which is
// exactly how this failed in the wild.
func TestRoutesDoNotShareHookSlots(t *testing.T) {
	stringRoute := func(ctx *Context) View {
		s := NewState(ctx, "shell")
		return Text(s.Get())
	}
	intRoute := func(ctx *Context) View {
		n := NewState(ctx, 7)
		return Text(strconv.Itoa(n.Get()))
	}

	ctx := NewContext()
	app := Navigator(stringRoute)

	if got := rendered(t, ctx, app); got != "shell" {
		t.Fatalf("root route rendered %q", got)
	}
	Push(ctx, intRoute)
	if got := rendered(t, ctx, app); got != "7" {
		t.Errorf("pushed route read the wrong slot, got %q", got)
	}
	Pop(ctx)
	if got := rendered(t, ctx, app); got != "shell" {
		t.Errorf("popping found the root's slot overwritten, got %q", got)
	}
}

// TestPushPopBetweenPassesDropsNothingUnexpected exercises the deferred-drop
// path's awkward case: a frame created and retired without ever rendering. Its
// scope never existed, so the drop must be a no-op rather than an error, and
// the retired list must still be drained so it cannot leak or double-fire.
func TestPushPopBetweenPassesDropsNothingUnexpected(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("home"))
	Render(ctx, app)
	bump(ctx, app, 1)

	Push(ctx, counterRoute("ghost"))
	Pop(ctx)

	if got := rendered(t, ctx, app); got != "home:1" {
		t.Errorf("a push/pop with no render in between disturbed the root: %q", got)
	}
	ctx.nav.mu.Lock()
	leftover := len(ctx.nav.retired)
	ctx.nav.mu.Unlock()
	if leftover != 0 {
		t.Errorf("retired list still holds %d ids after the pass that should have drained it", leftover)
	}
	if got := len(frameScopes(ctx)); got != 1 {
		t.Errorf("%d frame scopes survive, want 1", got)
	}
}

// --- resource ownership --------------------------------------------------

// TestDiscardedFrameStopsItsBackgroundResources is why cleanup registries
// nest. A screen that started a ticker and was then popped must stop ticking;
// otherwise Reset ends a session while the previous session's pollers keep
// firing into a tree that no longer contains their screen.
func TestDiscardedFrameStopsItsBackgroundResources(t *testing.T) {
	var mu sync.Mutex
	stopped := map[string]bool{}
	resourceRoute := func(name string) func(*Context) View {
		return func(ctx *Context) View {
			// Registered through a hook slot so it happens exactly once per
			// frame, the way UseInterval registers its ticker.
			started := NewState(ctx, false)
			if !started.Get() {
				ctx.OnClose(func() {
					mu.Lock()
					stopped[name] = true
					mu.Unlock()
				})
				// Written directly rather than through Set: Set triggers a
				// render request, and this is bookkeeping, not app state.
				ctx.lock.Lock()
				ctx.slots[0] = true
				ctx.lock.Unlock()
			}
			return Text(name)
		}
	}

	ctx := NewContext()
	app := Navigator(resourceRoute("root"))
	Render(ctx, app)
	Push(ctx, resourceRoute("pushed"))
	Render(ctx, app)

	Pop(ctx)
	Render(ctx, app)

	mu.Lock()
	defer mu.Unlock()
	if !stopped["pushed"] {
		t.Errorf("popped frame's resource was never stopped")
	}
	if stopped["root"] {
		t.Errorf("popping a frame stopped a resource belonging to the frame below it")
	}
}

// TestCloseStopsResourcesInEveryFrame guards the other direction: nesting the
// registries must not hide a live frame's resources from the app-wide
// shutdown that render.Manager.Close performs.
func TestCloseStopsResourcesInEveryFrame(t *testing.T) {
	var mu sync.Mutex
	var stopped []string
	resourceRoute := func(name string) func(*Context) View {
		return func(ctx *Context) View {
			started := NewState(ctx, false)
			if !started.Get() {
				ctx.OnClose(func() {
					mu.Lock()
					stopped = append(stopped, name)
					mu.Unlock()
				})
				ctx.lock.Lock()
				ctx.slots[0] = true
				ctx.lock.Unlock()
			}
			return Text(name)
		}
	}

	ctx := NewContext()
	app := Navigator(resourceRoute("root"))
	Render(ctx, app)
	Push(ctx, resourceRoute("pushed"))
	Render(ctx, app)

	ctx.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(stopped) != 2 {
		t.Fatalf("app-wide Close stopped %v, want both frames", stopped)
	}
}

// TestDroppedFrameRegistriesDoNotAccumulate pins the detach half of dropScope.
// A closed-but-still-linked sub-registry would be retained (and re-closed) by
// every later Close, so an app that navigates all day would grow one dead
// registry per pop.
func TestDroppedFrameRegistriesDoNotAccumulate(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("home"))
	Render(ctx, app)

	for i := 0; i < 5; i++ {
		Push(ctx, counterRoute("x"))
		Render(ctx, app)
		Pop(ctx)
		Render(ctx, app)
	}

	ctx.cleanup.mu.Lock()
	n := len(ctx.cleanup.children)
	ctx.cleanup.mu.Unlock()
	if n != 1 {
		t.Errorf("app registry holds %d sub-registries after 5 push/pop cycles, want 1 (the root frame)", n)
	}
}

// --- observation and isolation -------------------------------------------

func TestStackDepthAndCanPop(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("home"))

	// Before the first render the stack is unseeded: depth 0, nothing to pop.
	if got := StackDepth(ctx); got != 0 {
		t.Errorf("depth before the first render = %d, want 0", got)
	}
	if CanPop(ctx) {
		t.Errorf("CanPop is true on an unseeded stack")
	}

	Render(ctx, app)
	if got := StackDepth(ctx); got != 1 {
		t.Errorf("depth after seeding = %d, want 1", got)
	}
	if CanPop(ctx) {
		t.Errorf("CanPop is true at the root")
	}

	Push(ctx, counterRoute("a"))
	if got, want := StackDepth(ctx), 2; got != want {
		t.Errorf("depth after Push = %d, want %d", got, want)
	}
	if !CanPop(ctx) {
		t.Errorf("CanPop is false with a pushed route on the stack")
	}
}

// TestNavigationIsPerApp re-states the isolation the per-context stack exists
// for, now including Reset: two apps in one process must not reset each other.
func TestNavigationIsPerApp(t *testing.T) {
	ctxA, ctxB := NewContext(), NewContext()
	appA := Navigator(counterRoute("home-A"))
	appB := Navigator(counterRoute("home-B"))

	Render(ctxA, appA)
	Render(ctxB, appB)
	Push(ctxB, counterRoute("detail-B"))
	Render(ctxB, appB)

	Reset(ctxA, counterRoute("reset-A"))
	if got := rendered(t, ctxA, appA); got != "reset-A:0" {
		t.Errorf("app A did not reset, got %q", got)
	}
	if got := rendered(t, ctxB, appB); got != "detail-B:0" {
		t.Errorf("app A's Reset disturbed app B, got %q", got)
	}
}

// TestNavigationMarksTheAppDirty covers the polling hosts (WASM reads
// ctx.IsDirty on the root). A route calls Push with its FRAME context, so the
// flag has to be app-wide state rather than a bool on whichever context the
// mutation happened to be handed.
func TestNavigationMarksTheAppDirty(t *testing.T) {
	ctx := NewContext()
	var frame *Context
	app := Navigator(func(c *Context) View {
		frame = c
		return Text("home")
	})
	Render(ctx, app)
	ctx.ClearDirty()

	Push(frame, counterRoute("detail"))
	if !ctx.IsDirty() {
		t.Errorf("Push from a route context did not mark the app dirty: a polling host would never re-render")
	}
}

// TestNavigatorDoesNotRewindSiblingHooks covers the removal of Navigator's
// internal ctx.Reset(). That call rewound the host context's cursor mid-pass,
// so slots already handed to components rendered BEFORE the Navigator were
// handed out again to components rendered after it.
//
// The two siblings hold different types on purpose. With the rewind, `after`
// claims index 0 — the slot holding `before`'s string — and NewState's type
// assertion panics, which is how this corruption actually presents rather than
// as a quiet wrong value.
func TestNavigatorDoesNotRewindSiblingHooks(t *testing.T) {
	ctx := NewContext()
	nav := Navigator(counterRoute("home"))
	app := ComponentFunc(func(c *Context) *Node {
		before := NewState(c, "sibling")
		beforeText := Text(before.Get())
		navNode := nav.Render(c)
		after := NewState(c, 42)
		return Column(
			beforeText,
			ComponentFunc(func(*Context) *Node { return navNode }),
			Text(strconv.Itoa(after.Get())),
		).Render(c)
	})

	// Three passes: a rewind that happens to be harmless on the mount pass can
	// still corrupt the ones after it.
	for pass := 0; pass < 3; pass++ {
		root := Render(ctx, app)
		if got := root.Children[0].Props["content"]; got != "sibling" {
			t.Fatalf("pass %d: slot before the Navigator was clobbered, got %v", pass, got)
		}
		if got := root.Children[1].Props["content"]; got != "home:0" {
			t.Fatalf("pass %d: Navigator rendered %v", pass, got)
		}
		if got := root.Children[2].Props["content"]; got != "42" {
			t.Fatalf("pass %d: slot after the Navigator was handed a recycled index, got %v", pass, got)
		}
	}
}

// --- navigation before the Navigator's first render ----------------------

// TestPushBeforeFirstRenderKeepsTheInitialRoute pins the deep-link case. A
// handler that runs before the first pass (a launch URL, a restored session,
// a notification tap that opens the app) used to *become* the whole stack:
// takeTop only seeded `initial` into an empty stack, so the initial screen
// never existed, CanPop was false, and the platform Back gesture quit the app
// instead of revealing home.
func TestPushBeforeFirstRenderKeepsTheInitialRoute(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("home"))

	Push(ctx, counterRoute("detail"))

	if got := rendered(t, ctx, app); got != "detail:0" {
		t.Errorf("first render showed %q, want the pushed route", got)
	}
	if got := StackDepth(ctx); got != 2 {
		t.Errorf("stack depth = %d, want 2 (initial beneath the pushed route)", got)
	}
	if !CanPop(ctx) {
		t.Error("CanPop = false: Back would exit the app instead of revealing home")
	}

	Pop(ctx)
	if got := rendered(t, ctx, app); got != "home:0" {
		t.Errorf("after Pop the screen is %q, want the initial route", got)
	}
}

// TestMultiplePushesBeforeFirstRenderStayOrdered checks the splice puts the
// initial route at the *bottom*, not merely somewhere on the stack.
func TestMultiplePushesBeforeFirstRenderStayOrdered(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("home"))

	Push(ctx, counterRoute("one"))
	Push(ctx, counterRoute("two"))

	if got := rendered(t, ctx, app); got != "two:0" {
		t.Fatalf("first render showed %q, want the last pushed route", got)
	}
	if got := StackDepth(ctx); got != 3 {
		t.Fatalf("stack depth = %d, want 3", got)
	}
	Pop(ctx)
	if got := rendered(t, ctx, app); got != "one:0" {
		t.Errorf("after one Pop the screen is %q", got)
	}
	Pop(ctx)
	if got := rendered(t, ctx, app); got != "home:0" {
		t.Errorf("after two Pops the screen is %q, want the initial route", got)
	}
	if CanPop(ctx) {
		t.Error("CanPop = true at the bottom of the stack")
	}
}

// TestReplaceBeforeFirstRenderSupersedesTheInitialRoute is the other half of
// the seeding rule: Replace states what the bottom of the stack should be, so
// the Navigator's own initial route must not be spliced underneath it. This
// is the "restore straight into the logged-in screen" shape, where showing a
// login screen one Back away would be wrong.
func TestReplaceBeforeFirstRenderSupersedesTheInitialRoute(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("login"))

	Replace(ctx, counterRoute("session"))

	if got := rendered(t, ctx, app); got != "session:0" {
		t.Errorf("first render showed %q, want the replacement route", got)
	}
	if got := StackDepth(ctx); got != 1 {
		t.Errorf("stack depth = %d, want 1 — Replace does not change depth", got)
	}
	if CanPop(ctx) {
		t.Error("CanPop = true: the replaced initial route is still reachable")
	}
}

// TestResetBeforeFirstRenderSupersedesTheInitialRoute mirrors the Replace case
// for Reset, which likewise declares the whole stack.
func TestResetBeforeFirstRenderSupersedesTheInitialRoute(t *testing.T) {
	ctx := NewContext()
	app := Navigator(counterRoute("login"))

	Reset(ctx, counterRoute("session"))

	if got := rendered(t, ctx, app); got != "session:0" {
		t.Errorf("first render showed %q, want the reset route", got)
	}
	if got := StackDepth(ctx); got != 1 {
		t.Errorf("stack depth = %d, want 1", got)
	}
}
