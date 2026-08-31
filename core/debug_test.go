package core

import (
	"strings"
	"testing"
)

// withDebug flips debug mode on with a clean concern slate and restores the
// world afterwards. Debug state is process-wide, so these tests must not run
// in parallel with each other (none call t.Parallel).
func withDebug(t *testing.T) {
	t.Helper()
	ClearConcerns()
	SetDebugMode(true)
	t.Cleanup(func() {
		SetDebugMode(false)
		ClearConcerns()
	})
}

// hasConcern reports whether any recorded concern of the kind mentions every
// given substring in its detail.
func hasConcern(kind string, substrings ...string) bool {
	for _, c := range Concerns() {
		if c.Kind != kind {
			continue
		}
		ok := true
		for _, s := range substrings {
			if !strings.Contains(c.Detail, s) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// renderPass simulates one full render pass the way render.Manager drives it.
func renderPass(ctx *Context, view View) *Node {
	ctx.BeginRenderPass()
	ctx.Reset()
	n := view.Render(ctx)
	ctx.EndRenderPass()
	return n
}

// TestCursorDriftSkippedHook is the canonical conditional-hook bug: a
// NewState that ran on pass 1 is skipped on pass 2, so the pass ends with the
// cursor short of the allocated slots.
func TestCursorDriftSkippedHook(t *testing.T) {
	withDebug(t)
	ctx := NewContext()

	includeFirstHook := true
	view := ComponentFunc(func(ctx *Context) *Node {
		if includeFirstHook {
			NewState(ctx, "conditional")
		}
		NewState(ctx, "always")
		return &Node{Type: "Column"}
	})

	renderPass(ctx, view)
	if got := Concerns(); len(got) != 0 {
		t.Fatalf("pass 1 (consistent hooks) should raise no concerns, got %v", got)
	}

	includeFirstHook = false
	renderPass(ctx, view)
	if !hasConcern(ConcernCursorDrift, "skipped this pass") {
		t.Errorf("skipping a previously-run hook should raise a cursor-drift concern; concerns: %v", Concerns())
	}
}

// TestCursorDriftGrownHook covers the growth direction the slot-count check
// cannot see: a conditional hook turning ON appends its slot at the end, so
// cursor == len(slots) — only the pass-over-pass cursor comparison catches
// that the hook count changed.
func TestCursorDriftGrownHook(t *testing.T) {
	withDebug(t)
	ctx := NewContext()

	includeExtraHook := false
	view := ComponentFunc(func(ctx *Context) *Node {
		NewState(ctx, "always")
		if includeExtraHook {
			NewState(ctx, "appeared later")
		}
		return &Node{Type: "Column"}
	})

	renderPass(ctx, view)
	includeExtraHook = true
	renderPass(ctx, view)
	if !hasConcern(ConcernCursorDrift, "hook count varies") {
		t.Errorf("a hook appearing on a later pass should raise a cursor-drift concern; concerns: %v", Concerns())
	}
}

// TestNoDriftForUnvisitedScope: a Scope that renders on some passes and sits
// out others (the navigation pattern) is not drift — nothing after it reads
// its slots. It must stay clean both while parked and when it comes back.
func TestNoDriftForUnvisitedScope(t *testing.T) {
	withDebug(t)
	ctx := NewContext()

	visitScope := true
	view := ComponentFunc(func(ctx *Context) *Node {
		if visitScope {
			scoped := ctx.Scope("details")
			NewState(scoped, 42)
		}
		return &Node{Type: "Column"}
	})

	renderPass(ctx, view) // scope renders, baseline recorded
	visitScope = false
	renderPass(ctx, view) // scope parked: cursor 0, must not compare stale
	visitScope = true
	renderPass(ctx, view) // scope returns with the same hook count
	if got := Concerns(); len(got) != 0 {
		t.Errorf("an intermittently-rendered scope with a stable hook count is not drift, got %v", got)
	}
}

// TestDuplicateKeyConcern: duplicate sibling keys are flagged with the
// container type and key; unique keys are not.
func TestDuplicateKeyConcern(t *testing.T) {
	withDebug(t)
	ctx := NewContext()

	Column(
		Keyed("row", Text("first")),
		Keyed("row", Text("second")),
	).Render(ctx)
	if !hasConcern(ConcernDuplicateKey, "Column", `"row"`) {
		t.Errorf("duplicate sibling keys in a Column should be flagged; concerns: %v", Concerns())
	}

	ClearConcerns()
	Column(
		Keyed("a", Text("first")),
		Keyed("b", Text("second")),
		Text("unkeyed"), // empty keys never collide
		Text("unkeyed"),
	).Render(ctx)
	if got := Concerns(); len(got) != 0 {
		t.Errorf("unique (or empty) keys should raise nothing, got %v", got)
	}
}

// TestDuplicateKeyConcernThroughFor exercises the realistic path: a For over
// data whose derived keys collide.
func TestDuplicateKeyConcernThroughFor(t *testing.T) {
	withDebug(t)
	ctx := NewContext()

	items := []string{"apple", "apple", "banana"}
	For(items, func(item string, i int) View {
		return Keyed(item, Text(item))
	}).Render(ctx)
	if !hasConcern(ConcernDuplicateKey, "For", `"apple"`) {
		t.Errorf("colliding keys from a For should be flagged; concerns: %v", Concerns())
	}
}

// TestCachedDebugBypass: in debug mode Cached re-renders fresh each pass (so
// checks see the live subtree) and flags callback registrations escaping
// through it — the hard constraint from the Cached doc comment.
func TestCachedDebugBypass(t *testing.T) {
	withDebug(t)
	ctx := NewContext()
	ctx.BeginRenderPass()

	cached := Cached(Button("Tap", func() {}))
	first := cached.Render(ctx)
	second := cached.Render(ctx)
	if first == second {
		t.Errorf("debug mode must bypass the cache and render fresh, got the same *Node twice")
	}
	if !hasConcern(ConcernCachedCallbacks, "Cached(") {
		t.Errorf("a Button inside Cached should raise a cached-callbacks concern; concerns: %v", Concerns())
	}
}

// TestCachedDebugFlagsHooks: hook usage inside a cached view is the other
// forbidden dependency, detected via the parent cursor advancing.
func TestCachedDebugFlagsHooks(t *testing.T) {
	withDebug(t)
	ctx := NewContext()

	cached := Cached(ComponentFunc(func(ctx *Context) *Node {
		NewState(ctx, "forbidden in cached")
		return &Node{Type: "Text"}
	}))
	cached.Render(ctx)
	if !hasConcern(ConcernCachedHooks, "hook slot") {
		t.Errorf("NewState inside Cached should raise a cached-hooks concern; concerns: %v", Concerns())
	}
}

// TestCachedDebugCleanViewIsSilent: a constraint-respecting cached view runs
// through the bypass without raising anything.
func TestCachedDebugCleanViewIsSilent(t *testing.T) {
	withDebug(t)
	ctx := NewContext()

	cached := Cached(Text("static footer"))
	cached.Render(ctx)
	cached.Render(ctx)
	if got := Concerns(); len(got) != 0 {
		t.Errorf("a pure cached view should raise no concerns, got %v", got)
	}
}

// TestDebugModeOffRecordsNothing: with the flag off, the same violations run
// silently (the production behavior) and the collector stays empty — the
// zero-overhead contract, observed from the outside.
func TestDebugModeOffRecordsNothing(t *testing.T) {
	ClearConcerns()
	SetDebugMode(false)
	ctx := NewContext()

	Column(
		Keyed("dup", Text("a")),
		Keyed("dup", Text("b")),
	).Render(ctx)

	includeHook := true
	view := ComponentFunc(func(ctx *Context) *Node {
		if includeHook {
			NewState(ctx, 1)
		}
		return &Node{Type: "Column"}
	})
	renderPass(ctx, view)
	includeHook = false
	renderPass(ctx, view)

	if got := Concerns(); len(got) != 0 {
		t.Errorf("debug mode off must record nothing, got %v", got)
	}
}

// TestConcernDeduplication: the same finding firing across many passes bumps
// one entry's count instead of accumulating duplicates.
func TestConcernDeduplication(t *testing.T) {
	withDebug(t)
	ctx := NewContext()

	view := Column(
		Keyed("dup", Text("a")),
		Keyed("dup", Text("b")),
	)
	view.Render(ctx)
	view.Render(ctx)
	view.Render(ctx)

	list := Concerns()
	if len(list) != 1 {
		t.Fatalf("identical findings should deduplicate into one concern, got %d: %v", len(list), list)
	}
	if list[0].Count != 3 {
		t.Errorf("concern count = %d, want 3 (one per pass)", list[0].Count)
	}
	if dump := DumpConcerns(); !strings.Contains(dump, ConcernDuplicateKey) || !strings.Contains(dump, "×3") {
		t.Errorf("DumpConcerns should mention the kind and count, got:\n%s", dump)
	}
}
