package core

import "testing"

// renderStartList builds a List of n rows carrying both edges and StartAtEnd,
// as a real pass would, and returns the node.
func renderStartList(ctx *Context, rows int, onStart, onEnd func()) *Node {
	ctx.BeginRenderPass()
	items := []PropsAndChildren{StartAtEnd(), OnStartReached(onStart), OnEndReached(onEnd)}
	for range rows {
		items = append(items, Text("row"))
	}
	n := List(items...).Render(ctx)
	ctx.PurgeUnusedCallbacks()
	return n
}

func TestOnStartReachedAndStartAtEndReachTheWire(t *testing.T) {
	ctx := NewContext()
	n := renderStartList(ctx, 3, func() {}, func() {})
	start, _ := n.Props["onStartReached"].(string)
	end, _ := n.Props["onEndReached"].(string)
	if start == "" || end == "" || start == end {
		t.Fatalf("want two distinct callback IDs, got start %q end %q", start, end)
	}
	if n.Props["startAtEnd"] != true {
		t.Fatalf("StartAtEnd should set startAtEnd, props %v", n.Props)
	}
}

// The same guard as the bottom edge: one load per row count, reopened when an
// older page grows the list, and each edge guarded on its own.
func TestOnStartReachedFiresOnceUntilTheRowCountChanges(t *testing.T) {
	ctx := NewContext()
	var starts, ends int
	n := renderStartList(ctx, 20, func() { starts++ }, func() { ends++ })
	start := n.Props["onStartReached"].(string)
	ctx.TriggerCallback(start)
	ctx.TriggerCallback(start)
	if starts != 1 {
		t.Fatalf("two reports of the same top ran the handler %d times, want 1", starts)
	}
	// The bottom has its own guard: the top having fired at 20 rows does not
	// shut it.
	ctx.TriggerCallback(n.Props["onEndReached"].(string))
	if ends != 1 {
		t.Fatalf("the end edge ran %d times, want 1: the two edges share a guard", ends)
	}

	// The older page landed: 20 rows became 40, so the next top is a new one.
	n = renderStartList(ctx, 40, func() { starts++ }, func() { ends++ })
	ctx.TriggerCallback(n.Props["onStartReached"].(string))
	if starts != 2 {
		t.Fatalf("after the row count grew, the handler ran %d times in total, want 2", starts)
	}

	// A page that came back empty leaves the count where it was: shut.
	n = renderStartList(ctx, 40, func() { starts++ }, func() { ends++ })
	ctx.TriggerCallback(n.Props["onStartReached"].(string))
	if starts != 2 {
		t.Fatalf("an empty page reopened the guard: %d runs, want 2", starts)
	}
}

// A List without the props carries neither key, so every existing list is
// byte-identical on the wire.
func TestAListWithoutTheStartPropsCarriesNeither(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := List(Text("a")).Render(ctx)
	if _, ok := n.Props["onStartReached"]; ok {
		t.Fatal("a plain List carries onStartReached")
	}
	if _, ok := n.Props["startAtEnd"]; ok {
		t.Fatal("a plain List carries startAtEnd")
	}
}
