package core

// The top edge of a List, for the one kind of list that grows upward: a
// conversation, which opens on its newest message and loads older ones as the
// reader scrolls back.
//
//	core.List(
//	    core.StartAtEnd(),
//	    core.OnStartReached(thread.LoadOlder),
//	    messages...,
//	)
//
// # Why two props and not a scroll offset
//
// A thread needs three things from the host: to open at the end, to say when
// the reader has scrolled back to the start, and to keep the reader's place
// when the older page lands above them. None of them needs Go to know a
// number. An offset reported on every scroll would be a stream of events
// across the bridge whose only consumer would compare it with zero, and the
// place-keeping cannot be done from Go at all: by the time Go could answer a
// reported offset with a corrected one, the host would have drawn a frame at
// the wrong place. So the edge is an event, like OnEndReached, and the rest
// is the host's, declared once.
//
// # What the host does
//
//	                 web                     Compose                SwiftUI
//	StartAtEnd       scrollTop to the end    initial index: last    defaultScrollAnchor
//	                 after mount; kept at    row; kept at the end   (.bottom); a new
//	                 the end on an append    on an append while     last row scrolled to
//	                 while there             there                  while near the end
//	OnStartReached   the first row enters    first visible index    the row at the top
//	                 the viewport            within 2 of 0          edge (scrollPosition)
//	                                                                within 2 of the first
//	older rows       the first visible row   LazyColumn's own       scrollPosition(id:)
//	prepended        kept at its offset,     keyed anchoring        holds the top row
//	                 found by its Keyed key
//
// Each was measured on its platform with comps.MessageThread (tutorial lesson
// 4.33): wasm/verify's browser check 19, the emulator, and
// TutorialDevicePassUITests.testMessageThreadOpensAtTheEndAndKeepsThePlace.
// SwiftUI needed the most: its first row "appears" as soon as the lazy stack
// materializes it, ahead of the viewport, which loaded every page from one
// arrival at the top, so the top edge there is read from the row the scroll
// view reports at its top edge instead.
//
// Rows must be Keyed for the place to survive a prepend: the reconciler pairs
// children by position, so to every host a prepend is a new row in every
// slot, and the key is the only thing that says which row the reader was
// looking at. comps.MessageThread keys its bubbles.

// OnStartReached fires when the reader scrolls to within a few rows of the
// top of a List: the "load older messages" edge. The mirror of OnEndReached,
// with the same guard: it runs once per row count, so the host reporting the
// same top several times (it will) loads one page, and a page that comes back
// empty leaves the guard shut.
//
// Both edges share OnEndReached's ledger. It is keyed by callback ID, and a
// List carrying both registers two IDs, so the two guards never meet.
func OnStartReached(handler func()) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		// As in OnEndReached: id is assigned before the closure can run, and
		// the row count is read at dispatch time, when n holds the children
		// the pass rendered.
		var id string
		id = ctx.registerCallback(func() {
			if !ctx.endReached.shouldFire(id, len(n.Children)) {
				return
			}
			handler()
		})
		n.Props["onStartReached"] = id
	})
}

// StartAtEnd opens a List scrolled to its last row, and keeps it there while
// rows are appended as long as the reader has not scrolled away from the end:
// a conversation's newest message, and the next one arriving under it.
//
// A prop rather than a ScrollIntoView of the last row, because the natives'
// List is windowed and its off-screen rows do not exist to be scrolled to
// (see ScrollTarget), and because a command issued after the first render is
// a frame too late: the list would draw its top first.
func StartAtEnd() BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["startAtEnd"] = true
	})
}
