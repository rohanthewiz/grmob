package core

import "testing"

// A column of three named sections, rendered in a fresh pass: the props a
// host would receive.
func renderSections(ctx *Context) (a, b, c *Node) {
	ctx.BeginRenderPass()
	n := Column(
		Column(ScrollTarget("a"), Text("A")),
		Column(ScrollTarget("b"), Text("B")),
		Column(ScrollTarget(""), Text("C")),
	).Render(ctx)
	return n.Children[0], n.Children[1], n.Children[2]
}

func epochOf(n *Node) (int, bool) {
	v, ok := n.Props["scrollEpoch"].(int)
	return v, ok
}

// An app that never scrolls renders what it always did: no node carries a
// stamp, named or not.
func TestNoScrollCommandStampsNothing(t *testing.T) {
	a, b, c := renderSections(NewContext())
	for _, n := range []*Node{a, b, c} {
		if _, ok := epochOf(n); ok {
			t.Fatalf("a stamp before any command: %#v", n.Props)
		}
	}
}

// Only the latest command's target carries the stamp, and each command,
// including a second one to the same target, is a new epoch a host can see.
func TestScrollIntoViewStampsItsTargetOnly(t *testing.T) {
	ctx := NewContext()
	ScrollIntoView(ctx, "b")
	a, b, c := renderSections(ctx)
	if e, ok := epochOf(b); !ok || e != 1 {
		t.Fatalf("the target should carry epoch 1, got %#v", b.Props)
	}
	if _, ok := epochOf(a); ok {
		t.Fatal("a node the command did not name was stamped")
	}
	if _, ok := epochOf(c); ok {
		t.Fatal("an empty name is no target")
	}

	ScrollIntoView(ctx, "b")
	_, b, _ = renderSections(ctx)
	if e, _ := epochOf(b); e != 2 {
		t.Fatalf("a second command to the same target is a new epoch, got %d", e)
	}

	ScrollIntoView(ctx, "a")
	a, b, _ = renderSections(ctx)
	if e, _ := epochOf(a); e != 3 {
		t.Fatalf("the new target should carry epoch 3, got %d", e)
	}
	if _, ok := epochOf(b); ok {
		t.Fatal("the stamp stays only on the latest command's target")
	}
}

// A command for a name nothing carries takes an epoch and stamps nothing,
// and an empty name is not a command at all.
func TestScrollIntoViewWithoutATarget(t *testing.T) {
	ctx := NewContext()
	ScrollIntoView(ctx, "")
	ScrollIntoView(ctx, "nowhere")
	a, b, _ := renderSections(ctx)
	if _, ok := epochOf(a); ok {
		t.Fatal("stamped a node for a name it does not carry")
	}
	if _, ok := epochOf(b); ok {
		t.Fatal("stamped a node for a name it does not carry")
	}
	ScrollIntoView(ctx, "a")
	a, _, _ = renderSections(ctx)
	if e, _ := epochOf(a); e != 2 {
		t.Fatalf("the missed command should still have taken epoch 1, got %d", e)
	}
}

// A derived context (a navigation frame's scope, a themed subtree) shares the
// app's record: a command issued through one is seen through every other.
func TestScrollCommandsAreAppWide(t *testing.T) {
	root := NewContext()
	child := root.Scope("frame").WithTheme(DefaultTheme)
	ScrollIntoView(child, "a")
	a, _, _ := renderSections(root)
	if _, ok := epochOf(a); !ok {
		t.Fatal("a command through a derived context did not reach the root's render")
	}
}
