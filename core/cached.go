package core

import "sync"

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
	// Debug-mode hook point (debug workstream in the element-lessons plan):
	// when a debug flag lands, bypass the cache here and re-render fresh —
	//	if IsDebugMode() { return c.view.Render(ctx) }
	// — so the concern checks (cursor drift, callback registration escaping
	// through a cached render) can see the real subtree every pass.
	c.once.Do(func() {
		c.node = c.view.Render(ctx)
	})
	return c.node
}
