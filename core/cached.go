package core

import (
	"fmt"
	"sync"
)

// Cached wraps a view so it renders exactly once; every later pass returns
// the same *Node pointer. The payoff is in the reconciler: Diff short-circuits
// when old and new are the same pointer, so a cached subtree costs zero
// re-render AND zero diff work per pass — a static nav bar or footer drops
// out of the per-frame budget entirely.
//
// The wrapper must be constructed ONCE and reused across passes, typically as
// a package-level var:
//
//	var header = core.Cached(core.Text("My App"))
//
// Constructing it inside a render body is a silent no-op: the app's root
// function (and every ComponentFunc) runs on every pass, so a Cached built
// there is a fresh wrapper each time and caches nothing.
//
// Constraints — a cached view must be a pure function of its construction
// arguments, because it renders on pass 1 and never again:
//
//  1. No hooks (NewState, UseChildContext, hooks.*). Hooks advance the parent
//     context's slot cursor positionally; a view that consumes slots on pass 1
//     but not on pass 2 shifts every later component's slots, bleeding state
//     between unrelated components.
//
//  2. No callbacks — no Button/Input/Checkbox, no OnClick/OnChange/OnToggle
//     behavior props, nothing interactive. Callback IDs are assigned from
//     per-pass sequential counters (see callbackRegistry.beginPass) and any ID
//     not re-registered in a pass is purged after it. A cached subtree with
//     callbacks therefore breaks twice over: its own handlers are purged after
//     the first pass it skips, and by no longer consuming counter slots it
//     shifts the callback IDs of every component registered after it —
//     invalidating handlers well outside the cached subtree.
//
//  3. No dependence on values that change between passes. The first render's
//     ctx (theme, config) is baked into the node forever; a theme switch will
//     not reach a cached subtree.
//
//  4. Nothing may mutate the returned node after render — but that is the
//     framework-wide Node contract (see Node), not a Cached-specific rule.
//     Cached merely raises the stakes: the same pointer is the reconciler's
//     "unchanged" evidence, so a mutation here is invisible to Diff forever
//     rather than for one pass.
type cachedView struct {
	view View
	once sync.Once
	node *Node
}

// Cached returns a View that renders view on first use and replays the same
// *Node on every later pass. See the type comment for the constraints on what
// may be cached.
func Cached(view View) View {
	return &cachedView{view: view}
}

// Render is safe for concurrent use: sync.Once both serializes the single
// underlying render and publishes the node write, so every caller — including
// ones that lost the race — observes the fully built node.
func (c *cachedView) Render(ctx *Context) *Node {
	// Debug bypass, same move as element's Cached: re-render fresh every
	// pass so the debug checks can see the real subtree. The bypass also
	// converts the two Cached constraint violations (hooks, callbacks) from
	// invisible time bombs into direct measurements — see debugRender.
	if IsDebugMode() {
		return c.debugRender(ctx)
	}
	c.once.Do(func() {
		c.node = c.view.Render(ctx)
	})
	return c.node
}

// debugRender renders the wrapped view fresh and flags the two things a
// cached view must never do. Both are detected by sampling around the render
// rather than by instrumenting the hooks/registry themselves, so the check
// costs nothing anywhere but here:
//
//   - hook usage: the parent context's cursor advanced, so the view consumed
//     hook slots that the production cache would stop consuming after pass 1,
//     shifting every later component's slots (ConcernCachedHooks);
//   - callback registration: the registry's per-pass counters advanced, so
//     the view registered handlers that the production cache would let be
//     purged — and whose vacated counter slots would shift every later
//     component's callback IDs (ConcernCachedCallbacks).
//
// Note the debug/production behavior difference is deliberate but real: under
// the bypass the subtree consumes hooks and callback IDs consistently every
// pass, so the drift is *reported* here rather than *exhibited* — an app that
// only misbehaves with debug mode off has likely tripped exactly these
// concerns.
func (c *cachedView) debugRender(ctx *Context) *Node {
	cursorBefore := ctx.Cursor
	cbBefore := ctx.registry.registrationCount()
	node := c.view.Render(ctx)
	if ctx.Cursor != cursorBefore {
		upsertConcern(ConcernCachedHooks, fmt.Sprintf(
			"Cached(%T) consumed %d hook slot(s) during render: cached views must not call NewState/UseChildContext (they render once, then stop consuming slots, shifting every later component's state)",
			c.view, ctx.Cursor-cursorBefore))
	}
	if after := ctx.registry.registrationCount(); after != cbBefore {
		upsertConcern(ConcernCachedCallbacks, fmt.Sprintf(
			"Cached(%T) registered %d callback(s) during render: cached views must not contain interactive components or behavior props (their handlers are purged after the first skipped pass, and later components' callback IDs shift)",
			c.view, after-cbBefore))
	}
	return node
}
