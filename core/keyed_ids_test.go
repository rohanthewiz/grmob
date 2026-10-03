package core

import (
	"strings"
	"testing"
)

// Identity-keyed callback IDs (N-073): a keyed subtree names its own
// callbacks, so what happens inside it moves no ID outside it. See
// "Identity-keyed IDs" on callbackRegistry.beginPass.

// clickIDs lists the onClick ID of every node in tree order.
func clickIDs(n *Node) []string {
	var ids []string
	var walk func(*Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		if id, ok := n.Props["onClick"].(string); ok {
			ids = append(ids, id)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return ids
}

// The headline: a keyed subtree that grows from one handler to three moves
// neither its keyed sibling's IDs nor the unkeyed button after both. With
// positional IDs both would have shifted by two.
func TestKeyedSubtreeDoesNotRenumberItsNeighbours(t *testing.T) {
	ctx := NewContext()
	editing := false
	noop := func() {}
	view := ComponentFunc(func(ctx *Context) *Node {
		var first View = Button("a", noop)
		if editing {
			first = Row(Button("a1", noop), Button("a2", noop), Button("a3", noop))
		}
		return Column(
			Keyed("a", first),
			Keyed("b", Button("b", noop)),
			Button("save", noop),
		).Render(ctx)
	})

	before := clickIDs(pass(ctx, view))
	want := []string{"cb_a/0", "cb_b/0", "cb_0"}
	if strings.Join(before, " ") != strings.Join(want, " ") {
		t.Fatalf("IDs = %v, want %v", before, want)
	}

	editing = true
	after := clickIDs(pass(ctx, view))
	want = []string{"cb_a/0", "cb_a/1", "cb_a/2", "cb_b/0", "cb_0"}
	if strings.Join(after, " ") != strings.Join(want, " ") {
		t.Fatalf("IDs while editing = %v, want %v", after, want)
	}
}

// Nested keys spell a path; keys that would forge a boundary are escaped; a
// key repeated in one scope (two keyed lists under one unkeyed column) is
// told apart by occurrence; an empty key opens nothing.
func TestKeyedIDSpelling(t *testing.T) {
	ctx := NewContext()
	noop := func() {}
	view := Column(
		Keyed("row", Keyed("cell", Button("x", noop))),
		Keyed("a/b~c%", Button("x", noop)),
		Column(Keyed("0", Button("x", noop))),
		Column(Keyed("0", Button("x", noop))),
		Keyed("", Button("x", noop)),
	)
	got := clickIDs(pass(ctx, view))
	want := []string{"cb_row/cell/0", "cb_a%2Fb%7Ec%25/0", "cb_0/0", "cb_0~1/0", "cb_0"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("IDs = %v\nwant  %v", got, want)
	}

	// Every one of them dispatches to its own handler.
	ctx2 := NewContext()
	var hits []int
	hit := func(i int) func() { return func() { hits = append(hits, i) } }
	tree := pass(ctx2, Column(
		Column(Keyed("0", Button("x", hit(0)))),
		Column(Keyed("0", Button("x", hit(1)))),
	))
	for _, id := range clickIDs(tree) {
		ctx2.TriggerCallback(id)
	}
	if len(hits) != 2 || hits[0] != 0 || hits[1] != 1 {
		t.Fatalf("repeated keys dispatched to %v, want [0 1]", hits)
	}
}

// A panic inside a keyed subtree is undone exactly: the scope is closed by its
// defer, the abandoned IDs stop dispatching, and the fallback's own use of
// the same key is not mistaken for a repeat.
func TestErrorBoundaryRollsBackAKeyedSubtree(t *testing.T) {
	ctx := NewContext()
	abandoned := false
	var abandonedIDs []string
	child := Keyed("k", ComponentFunc(func(ctx *Context) *Node {
		// Two handlers where the fallback registers one, so the second ID is
		// re-used by nobody and only the rollback's unmarking can purge it.
		abandonedIDs = append(abandonedIDs[:0],
			ctx.registerCallback(func() { abandoned = true }),
			ctx.registerCallback(func() { abandoned = true }))
		panic("mid-subtree")
	}))
	view := Column(
		ErrorBoundary(child, func(error) View {
			return Keyed("k", Button("retry", func() {}))
		}),
		Button("after", func() {}),
	)

	got := clickIDs(pass(ctx, view))
	want := []string{"cb_k/0", "cb_0"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("IDs = %v, want %v (the fallback's key a first use, the root's counter untouched)", got, want)
	}
	if strings.Join(abandonedIDs, " ") != "cb_k/0 cb_k/1" {
		t.Fatalf("abandoned subtree registered %v", abandonedIDs)
	}
	// cb_k/0 now runs the fallback's handler; cb_k/1 must run nothing.
	for _, id := range abandonedIDs {
		ctx.TriggerCallback(id)
	}
	if abandoned {
		t.Fatal("the abandoned subtree's handler is still dispatchable")
	}
}

// A Navigator frame's callbacks carry the frame's key, so the ID a host still
// holds for a popped screen reaches nothing on the screen underneath, even
// when that screen registers a handler at the same position.
func TestStaleIDFromAPoppedFrameReachesNothing(t *testing.T) {
	ctx := NewContext()
	opened := ""
	home := func(ctx *Context) View {
		return Column(Button("open", func() { opened = "home" }))
	}
	detail := func(ctx *Context) View {
		return Column(Button("close", func() { Pop(ctx) }))
	}
	app := Navigator(home)

	pass(ctx, app)
	Push(ctx, detail)
	detailTree := pass(ctx, app)
	closeID := clickIDs(detailTree)[0]

	// The first press pops; the second arrives with the same, now stale, ID.
	ctx.TriggerCallback(closeID)
	homeTree := pass(ctx, app)
	ctx.TriggerCallback(closeID)

	if homeID := clickIDs(homeTree)[0]; homeID == closeID {
		t.Fatalf("both screens spell their first handler %q", homeID)
	}
	if opened != "" {
		t.Fatalf("a stale ID from the popped screen ran %q's handler", opened)
	}
}
