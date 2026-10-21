package core

import "testing"

// OnSwipe and OnVerticalSwipe ride the text channel: the host reports a
// direction word, and Go hands the handler a SwipeDirection. These pin the
// wire key per axis, the payload filter, the Disabled guard, and that a nil
// handler claims nothing.

func TestOnSwipeDispatchesItsAxis(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	var got []SwipeDirection
	n := Box(OnSwipe(func(d SwipeDirection) { got = append(got, d) })).Render(ctx)

	id, ok := n.Props["onSwipe"].(string)
	if !ok || id == "" {
		t.Fatalf("no onSwipe prop: %#v", n.Props)
	}
	if _, ok := n.Props["onVerticalSwipe"]; ok {
		t.Fatalf("OnSwipe claimed the vertical axis too: %#v", n.Props)
	}

	// The horizontal pair passes through; the vertical pair, an unknown
	// word, an empty payload and a physical "left" are all dropped rather
	// than guessed at.
	for _, p := range []string{"leading", "trailing", "up", "down", "left", "", "LEADING"} {
		ctx.TriggerTextCallback(id, p)
	}
	want := []SwipeDirection{SwipeLeading, SwipeTrailing}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestOnVerticalSwipeDispatchesItsAxis(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	var got []SwipeDirection
	n := Column(OnVerticalSwipe(func(d SwipeDirection) { got = append(got, d) })).Render(ctx)

	id, ok := n.Props["onVerticalSwipe"].(string)
	if !ok || id == "" {
		t.Fatalf("no onVerticalSwipe prop: %#v", n.Props)
	}
	for _, p := range []string{"up", "leading", "down", "trailing"} {
		ctx.TriggerTextCallback(id, p)
	}
	if len(got) != 2 || got[0] != SwipeUp || got[1] != SwipeDown {
		t.Fatalf("got %v, want [up down]", got)
	}
}

func TestBothSwipeAxesOnOneNode(t *testing.T) {
	// Two props, two IDs: a host that sees both claims both axes and reports
	// each release to the axis it travelled along.
	ctx := NewContext()
	ctx.BeginRenderPass()

	var h, v SwipeDirection
	n := Box(
		OnSwipe(func(d SwipeDirection) { h = d }),
		OnVerticalSwipe(func(d SwipeDirection) { v = d }),
	).Render(ctx)

	hID, _ := n.Props["onSwipe"].(string)
	vID, _ := n.Props["onVerticalSwipe"].(string)
	if hID == "" || vID == "" || hID == vID {
		t.Fatalf("want two distinct IDs, got %q and %q", hID, vID)
	}
	ctx.TriggerTextCallback(hID, "trailing")
	ctx.TriggerTextCallback(vID, "down")
	if h != SwipeTrailing || v != SwipeDown {
		t.Fatalf("h=%q v=%q", h, v)
	}
}

func TestSwipeOnDisabledNodeIsDropped(t *testing.T) {
	// Disabled is written after OnSwipe on purpose: props apply in argument
	// order, so the guard has to read the finished node at dispatch time
	// rather than the half-built one at registration.
	ctx := NewContext()
	ctx.BeginRenderPass()

	fired := false
	n := Box(OnSwipe(func(SwipeDirection) { fired = true }), Disabled(true)).Render(ctx)

	id, _ := n.Props["onSwipe"].(string)
	if id == "" {
		t.Fatalf("no onSwipe prop: %#v", n.Props)
	}
	ctx.TriggerTextCallback(id, "leading")
	if fired {
		t.Fatal("a Disabled node's swipe ran its handler")
	}
}

func TestNilSwipeHandlerClaimsNothing(t *testing.T) {
	// No prop at all, so no host takes the axis away from a Scroll around
	// the node for a handler that would do nothing.
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Box(OnSwipe(nil), OnVerticalSwipe(nil)).Render(ctx)
	if _, ok := n.Props["onSwipe"]; ok {
		t.Fatalf("nil OnSwipe wrote a prop: %#v", n.Props)
	}
	if _, ok := n.Props["onVerticalSwipe"]; ok {
		t.Fatalf("nil OnVerticalSwipe wrote a prop: %#v", n.Props)
	}
}
