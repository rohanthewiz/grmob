package core

import "testing"

// statefulProbe is a view that owns one hook slot of type T and records what
// it read on each pass. It exists so a test can place independent stateful
// components on both sides of a WithTheme boundary and see whether their slots
// stayed independent.
type statefulProbe[T comparable] struct {
	initial T
	seen    *[]T
	// setTo, when non-nil, is written back into the slot on the first pass.
	setTo *T
	done  *bool
}

func (p statefulProbe[T]) Render(ctx *Context) *Node {
	st := NewState(ctx, p.initial)
	*p.seen = append(*p.seen, st.Get())
	if p.setTo != nil && !*p.done {
		*p.done = true
		st.Set(*p.setTo)
	}
	return &Node{Type: "Text", Props: map[string]any{}}
}

// TestWithThemeSharesParentHookScope pins the fix for the WithTheme /
// WithConfig slot-aliasing bug: the themed copy used to carry a *value* copy
// of the parent's slot slice and cursor, so a stateful child inside the theme
// and one outside it both landed on slot 0 — pass 2 then read a string out of
// an int slot and panicked.
//
// Three passes, because the failure needs one pass to seed the aliased slot
// and a second to read it back through the wrong type.
func TestWithThemeSharesParentHookScope(t *testing.T) {
	ctx := NewContext()

	var insideSeen []int
	var outsideSeen []string
	setTo := 42
	setDone := false

	app := ComponentFunc(func(c *Context) *Node {
		return Column(
			WithTheme(MaterialTheme, statefulProbe[int]{
				initial: 0, seen: &insideSeen, setTo: &setTo, done: &setDone,
			}),
			statefulProbe[string]{initial: "s", seen: &outsideSeen},
		).Render(c)
	})

	for pass := 0; pass < 3; pass++ {
		ctx.Reset()
		if err := Guard(func() { app.Render(ctx) }); err != nil {
			t.Fatalf("pass %d panicked: %v", pass, err)
		}
	}

	// The themed child's Set on pass 1 must be visible on passes 2 and 3 —
	// under the bug it was written to a slot the parent then overwrote.
	wantInside := []int{0, 42, 42}
	if len(insideSeen) != len(wantInside) {
		t.Fatalf("themed child rendered %d times, want %d (%v)", len(insideSeen), len(wantInside), insideSeen)
	}
	for i, want := range wantInside {
		if insideSeen[i] != want {
			t.Errorf("themed child pass %d read %d, want %d (all: %v)", i, insideSeen[i], want, insideSeen)
		}
	}

	// The unthemed sibling must keep its own slot untouched.
	for i, got := range outsideSeen {
		if got != "s" {
			t.Errorf("sibling pass %d read %q, want \"s\" (all: %v)", i, got, outsideSeen)
		}
	}

	// Two hooks were called, so the owning context must have advanced two
	// slots — the bug left the parent cursor at 1 with one shared slot.
	if got := len(ctx.slots); got != 2 {
		t.Errorf("root context allocated %d slots, want 2", got)
	}
	if ctx.Cursor != 2 {
		t.Errorf("root cursor ended at %d, want 2", ctx.Cursor)
	}
}

// TestWithConfigSharesParentHookScope is the same pin for WithConfig, which
// had the identical copy-by-value bug and is the path render.New takes.
func TestWithConfigSharesParentHookScope(t *testing.T) {
	ctx := NewContext()
	cfg := &AppConfig{Name: "probe"}

	var insideSeen, outsideSeen []int

	app := ComponentFunc(func(c *Context) *Node {
		themed := c.WithConfig(cfg)
		if themed.Config().Name != "probe" {
			t.Errorf("WithConfig copy lost the config")
		}
		inside := statefulProbe[int]{initial: 1, seen: &insideSeen}.Render(themed)
		outside := statefulProbe[int]{initial: 2, seen: &outsideSeen}.Render(c)
		return &Node{Type: "Column", Props: map[string]any{}, Children: []*Node{inside, outside}}
	})

	for pass := 0; pass < 3; pass++ {
		ctx.Reset()
		app.Render(ctx)
	}

	for i, got := range insideSeen {
		if got != 1 {
			t.Errorf("configured child pass %d read %d, want 1 (all: %v)", i, got, insideSeen)
		}
	}
	for i, got := range outsideSeen {
		if got != 2 {
			t.Errorf("sibling pass %d read %d, want 2 (all: %v)", i, got, outsideSeen)
		}
	}
	if got := len(ctx.slots); got != 2 {
		t.Errorf("root context allocated %d slots, want 2", got)
	}
}

// TestWithThemeChildContextSharesParentScope covers the UseChildContext half
// of the same fix: a child scope opened inside a theme must consume a slot in
// the parent's numbering, not a duplicate of one the parent hands out again.
func TestWithThemeChildContextSharesParentScope(t *testing.T) {
	ctx := NewContext()

	var childSeen, siblingSeen []int

	app := ComponentFunc(func(c *Context) *Node {
		themed := c.WithTheme(MaterialTheme)
		child := UseChildContext(themed)
		if child.Theme() != MaterialTheme {
			t.Errorf("child context did not inherit the themed copy's theme")
		}
		inner := statefulProbe[int]{initial: 7, seen: &childSeen}.Render(child)
		sib := statefulProbe[int]{initial: 9, seen: &siblingSeen}.Render(c)
		return &Node{Type: "Column", Props: map[string]any{}, Children: []*Node{inner, sib}}
	})

	for pass := 0; pass < 3; pass++ {
		ctx.Reset()
		app.Render(ctx)
	}

	for i, got := range siblingSeen {
		if got != 9 {
			t.Errorf("sibling pass %d read %d, want 9 (all: %v)", i, got, siblingSeen)
		}
	}
	for i, got := range childSeen {
		if got != 7 {
			t.Errorf("child-scope hook pass %d read %d, want 7 (all: %v)", i, got, childSeen)
		}
	}
	// One child context + one sibling state = 2 slots on the root.
	if got := len(ctx.slots); got != 2 {
		t.Errorf("root context allocated %d slots, want 2", got)
	}
}
