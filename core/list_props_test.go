package core

import "testing"

// The three collection props of roadmap tier B: core.Horizontal (B1),
// core.OnEndReached (B2) and core.StickyHeader (B3).
//
// Two of them are StyleProps that set fields the web targets already emit, so
// what is worth pinning here is the *mapping* — which Style fields each one
// writes — because that mapping is the whole contract the four renderers were
// written against. The third is a BehaviorProp with real logic behind it (the
// debounce), and most of this file is about that.

// --- B1: Horizontal ---------------------------------------------------------

func TestHorizontalSetsTheRowAxisAndOverflow(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Scroll(Horizontal(), Text("chip")).Render(ctx)

	if n.Style.FlexDirection != FlexRow {
		t.Errorf("FlexDirection = %q, want %q — the natives read this field to pick their "+
			"scroll axis, and both web targets read it to override the node's stacking default",
			n.Style.FlexDirection, FlexRow)
	}
	if n.Style.Overflow != "auto" {
		t.Errorf("Overflow = %q, want \"auto\" — without it a sideways strip is clipped in the "+
			"browser rather than scrolled, since a Scroll emits no overflow of its own",
			n.Style.Overflow)
	}
}

// The overflow is a default, not a decision, so a caller who states one keeps
// it — and keeps it whichever side of Horizontal() they wrote it on. (The
// axis is not defaulted: it is what the prop means.)
func TestHorizontalDefersToAnExplicitOverflow(t *testing.T) {
	for _, order := range []struct {
		name  string
		items []PropsAndChildren
	}{
		{"prop first", []PropsAndChildren{Horizontal(), Overflow("hidden")}},
		{"prop last", []PropsAndChildren{Overflow("hidden"), Horizontal()}},
	} {
		t.Run(order.name, func(t *testing.T) {
			ctx := NewContext()
			ctx.BeginRenderPass()

			n := Scroll(order.items...).Render(ctx)
			if n.Style.Overflow != "hidden" {
				t.Errorf("Overflow = %q, want \"hidden\"", n.Style.Overflow)
			}
			if n.Style.FlexDirection != FlexRow {
				t.Errorf("FlexDirection = %q, want %q", n.Style.FlexDirection, FlexRow)
			}
		})
	}
}

// --- B3: StickyHeader -------------------------------------------------------

func TestStickyHeaderSetsPositionTopAndZIndex(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Row(StickyHeader(), Text("January 2026")).Render(ctx)

	if n.Style.Position != PositionSticky {
		t.Errorf("Position = %q, want %q — this is the marker both natives look for in their "+
			"lazy list, and the declaration the browser pins on", n.Style.Position, PositionSticky)
	}
	// A sticky box with no offset never sticks, and one at the default
	// stacking level is painted over by the rows sliding under it. Both are
	// web-only concerns and both are silent failures, which is why the prop
	// supplies them rather than leaving them to the caller.
	if n.Style.Top != "0" {
		t.Errorf("Top = %q, want \"0\"", n.Style.Top)
	}
	if n.Style.ZIndex != 1 {
		t.Errorf("ZIndex = %d, want 1", n.Style.ZIndex)
	}
}

func TestStickyHeaderDefersToExplicitOffsetAndLayer(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	// A band pinned below a toolbar, above everything.
	n := Row(
		StickyHeader(),
		UseStyle(Style{Top: "44px", ZIndex: 5}),
	).Render(ctx)

	if n.Style.Top != "44px" || n.Style.ZIndex != 5 {
		t.Errorf("Top/ZIndex = %q/%d, want 44px/5 — the supplied defaults must not overwrite "+
			"a caller's own offset", n.Style.Top, n.Style.ZIndex)
	}
}

// --- B2: OnEndReached -------------------------------------------------------

func TestOnEndReachedRegistersAVoidCallback(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	var fired int
	n := List(OnEndReached(func() { fired++ }), Text("row")).Render(ctx)

	id, ok := n.Props["onEndReached"].(string)
	if !ok || id == "" {
		t.Fatalf("List node has no onEndReached prop: %#v", n.Props)
	}
	ctx.TriggerCallback(id)
	if fired != 1 {
		t.Fatalf("handler ran %d times, want 1", fired)
	}
}

// The debounce, which is the whole reason this prop is not a bare On("EndReached").
//
// Every renderer reports the edge more than once for the same bottom — an
// IntersectionObserver re-fires on resize, .onAppear re-fires on recycle, a
// snapshot flow emits per visible-index change — so a slow fetch would see
// two or three calls before its first page landed and an offset pager would
// load page 2 twice.
func TestOnEndReachedFiresOnceUntilTheRowCountChanges(t *testing.T) {
	ctx := NewContext()

	var fired int
	// render builds a List of n rows carrying the prop, as a real pass would,
	// and returns the callback ID the pass assigned it.
	render := func(rows int) string {
		ctx.BeginRenderPass()
		items := []PropsAndChildren{OnEndReached(func() { fired++ })}
		for range rows {
			items = append(items, Text("row"))
		}
		n := List(items...).Render(ctx)
		ctx.PurgeUnusedCallbacks()
		return n.Props["onEndReached"].(string)
	}

	id := render(20)
	ctx.TriggerCallback(id)
	ctx.TriggerCallback(id)
	ctx.TriggerCallback(id)
	if fired != 1 {
		t.Fatalf("three reports of the same bottom ran the handler %d times, want 1", fired)
	}

	// The page landed: 20 rows became 40, so the next bottom is a new one.
	id = render(40)
	ctx.TriggerCallback(id)
	if fired != 2 {
		t.Fatalf("after the row count grew, handler ran %d times in total, want 2", fired)
	}
	ctx.TriggerCallback(id)
	if fired != 2 {
		t.Fatalf("the new bottom re-fired: handler ran %d times, want 2", fired)
	}
}

// A page that comes back with nothing — the feed is exhausted, or the fetch
// failed — leaves the count where it was, and the guard therefore stays shut.
// Scrolling at the bottom of a list that just came back empty must not re-ask
// forever; components.LoadMore's error arm is where the retry lives.
func TestOnEndReachedStaysQuietWhenAPageAddsNothing(t *testing.T) {
	ctx := NewContext()

	var fired int
	render := func(rows int) string {
		ctx.BeginRenderPass()
		items := []PropsAndChildren{OnEndReached(func() { fired++ })}
		for range rows {
			items = append(items, Text("row"))
		}
		n := List(items...).Render(ctx)
		ctx.PurgeUnusedCallbacks()
		return n.Props["onEndReached"].(string)
	}

	id := render(12)
	ctx.TriggerCallback(id)
	id = render(12) // the fetch returned no rows
	ctx.TriggerCallback(id)
	ctx.TriggerCallback(id)
	if fired != 1 {
		t.Fatalf("handler ran %d times, want 1", fired)
	}
}

// The ledger is trimmed against the callback registry's own survivors, so a
// List that leaves the tree does not leave a guard behind for whichever node
// next inherits its positional ID.
func TestOnEndReachedGuardIsPurgedWithItsCallback(t *testing.T) {
	ctx := NewContext()

	ctx.BeginRenderPass()
	n := List(OnEndReached(func() {}), Text("a"), Text("b")).Render(ctx)
	ctx.PurgeUnusedCallbacks()
	id := n.Props["onEndReached"].(string)
	ctx.TriggerCallback(id)

	if _, ok := ctx.endReached.firedAt[id]; !ok {
		t.Fatalf("no guard recorded for %s after it fired", id)
	}

	// A pass in which the list is gone entirely.
	ctx.BeginRenderPass()
	Column(Text("some other screen")).Render(ctx)
	ctx.PurgeUnusedCallbacks()

	if _, ok := ctx.endReached.firedAt[id]; ok {
		t.Errorf("guard for %s outlived its callback — the next list to be handed this ID "+
			"would have its first page fetch swallowed", id)
	}
}

// The ledger is per app tree, like the callback registry and the navigation
// stack it sits beside. Two Managers in one test binary must not silence each
// other's feeds.
func TestOnEndReachedGuardsAreNotSharedBetweenApps(t *testing.T) {
	var firedA, firedB int
	build := func(ctx *Context, counter *int) string {
		ctx.BeginRenderPass()
		n := List(OnEndReached(func() { *counter++ }), Text("row")).Render(ctx)
		return n.Props["onEndReached"].(string)
	}

	a, b := NewContext(), NewContext()
	idA := build(a, &firedA)
	idB := build(b, &firedB)
	if idA != idB {
		t.Fatalf("the two apps took different IDs (%s, %s) — this test only says something "+
			"while they collide", idA, idB)
	}

	a.TriggerCallback(idA)
	b.TriggerCallback(idB)
	if firedA != 1 || firedB != 1 {
		t.Fatalf("firedA=%d firedB=%d, want 1 and 1 — one app's guard suppressed the other's",
			firedA, firedB)
	}
}

// A context derived for a theme or a config override is the same app wearing
// a different hat, so it must share the ledger rather than open a second one.
func TestDerivedContextsShareTheEndReachedLedger(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	var fired int
	themed := ctx.WithTheme(DefaultTheme)
	n := List(OnEndReached(func() { fired++ }), Text("row")).Render(themed)
	id := n.Props["onEndReached"].(string)

	ctx.TriggerCallback(id)
	ctx.TriggerCallback(id)
	if fired != 1 {
		t.Fatalf("handler ran %d times, want 1 — the themed copy kept its own guard", fired)
	}
}
