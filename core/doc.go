// Package core is the vocabulary a grmob app is written in: views, the tree
// they render to, the state they read, and the styling and accessibility
// properties they carry. Everything else in the framework is layered on this
// package's public surface — the widget library, the validation layer, the two
// web exporters and both native bridges import core and add nothing to it.
//
// # The shape of an app
//
// A view is anything with a Render method ([View]), and what it returns is a
// [Node] — a plain data description of one element and its children. An app is
// a function from a [Context] to a view, and a frame is one call of it:
//
//	func App(ctx *core.Context) core.View {
//	    return core.Column(
//	        core.Text("Hello"),
//	        core.Button("Tap", func() { /* ... */ }),
//	    )
//	}
//
// Nothing here draws anything. Render produces a tree; the render loop in
// package render diffs that tree against the previous one and hands the
// difference to whichever host is attached. The same app function therefore
// runs unchanged on Android, on iOS, in the browser and in a test.
//
// # State and the cursor
//
// [NewState] allocates a slot on the context and returns typed accessors for
// it. Slots are identified by the order the calls happen in, not by name:
//
//	pass 1:  NewState(ctx, 0)   NewState(ctx, "")   ->  slot 0, slot 1
//	pass 2:  NewState(ctx, 0)   NewState(ctx, "")   ->  slot 0, slot 1
//	         ^ the same two slots, because the same two calls in the same order
//
// That is what makes the rules of hooks rules rather than advice: a state call
// inside a conditional shifts every later call onto a neighbour's slot. Debug
// mode detects the shift and reports it as a concern (see [SetDebugMode] and
// [DumpConcerns]) rather than leaving it to be found as a display bug.
//
// [State.Set] is safe to call from any goroutine. It writes the slot and nudges
// the render loop, which coalesces a burst of writes into a single pass.
//
// # Nodes are frozen
//
// A [Node] is immutable once the pass that produced it returns. The reconciler
// treats pointer equality as proof that a subtree is unchanged, which is what
// makes [Cached] worth having and what makes a post-render mutation invisible
// rather than merely unusual — the diff would skip the subtree it changed.
//
// # Where to read next
//
// The narrative documentation covers the parts a reference cannot: the
// architecture's pass boundaries, the rules of hooks in full, the styling
// resolution order, and the reconciler's cost model. This page is the exact
// surface; those pages are why it has the shape it does.
package core
