package core

import (
	"errors"
	"strings"
	"testing"
)

// pass runs one full render pass over ctx the way render.Manager does — the
// callback-ID reset, the cursor reset, the render, the purge — because most of
// what ErrorBoundary repairs is per-pass bookkeeping, and a test that skipped
// the boundaries would not exercise any of it.
func pass(ctx *Context, view View) *Node {
	ctx.BeginRenderPass()
	ctx.Reset()
	n := view.Render(ctx)
	ctx.PurgeUnusedCallbacks()
	return n
}

// boom is a View whose Render always panics with the given value.
func boom(v any) View {
	return ComponentFunc(func(ctx *Context) *Node {
		panic(v)
	})
}

// findText walks a node tree and reports whether any Text node's content
// contains sub. The trees here are tiny, so a linear walk beats threading
// indices through every assertion.
func findText(n *Node, sub string) bool {
	if n == nil {
		return false
	}
	if n.Type == "Text" {
		if s, ok := n.Props["content"].(string); ok && strings.Contains(s, sub) {
			return true
		}
	}
	for _, c := range n.Children {
		if findText(c, sub) {
			return true
		}
	}
	return false
}

func TestErrorBoundaryRendersFallbackInsteadOfPanicking(t *testing.T) {
	ctx := NewContext()
	var got error
	view := ErrorBoundary(boom("kaboom"), func(err error) View {
		got = err
		return Text("fallback")
	})

	node := pass(ctx, view)

	if got == nil {
		t.Fatal("fallback was never called")
	}
	if !strings.Contains(got.Error(), "kaboom") {
		t.Errorf("error = %q, want it to mention the panic value", got.Error())
	}
	var re *RenderError
	if !errors.As(got, &re) {
		t.Fatalf("fallback got %T, want *RenderError", got)
	}
	// The stack is the reason the value is wrapped at all: it must still name
	// the frame that panicked, not just the recover site.
	if !strings.Contains(string(re.Stack), "boom.func") {
		t.Errorf("stack does not name the panicking component:\n%s", re.Stack)
	}
	if !findText(node, "fallback") {
		t.Error("fallback content is not in the rendered tree")
	}
}

func TestErrorBoundaryUnwrapsAPanickedError(t *testing.T) {
	// A panic often carries a real error. Routing it through Unwrap is what
	// lets a fallback branch on it rather than string-match the message.
	sentinel := errors.New("db unavailable")
	ctx := NewContext()
	var got error
	pass(ctx, ErrorBoundary(boom(sentinel), func(err error) View {
		got = err
		return Text("fallback")
	}))

	if !errors.Is(got, sentinel) {
		t.Fatalf("errors.Is(%v, sentinel) = false, want true", got)
	}
}

func TestErrorBoundaryDoesNotShiftSiblingHookSlots(t *testing.T) {
	// The headline guarantee. A panic partway through a child's hooks must not
	// move the slot positions of anything rendered after the boundary — and it
	// must hold even when the child's hook count VARIES between passes, which
	// is what a data-dependent panic looks like.
	ctx := NewContext()
	hooksBeforePanic := 0

	build := func() View {
		return ComponentFunc(func(c *Context) *Node {
			before := NewState(c, "before")
			child := ComponentFunc(func(cc *Context) *Node {
				for i := 0; i < hooksBeforePanic; i++ {
					NewState(cc, i)
				}
				panic("child failed")
			})
			fb := ErrorBoundary(child, func(error) View { return Text("fb") })
			// Bound to locals so the hook order reads top-to-bottom: before,
			// the boundary's two contexts, after.
			beforeText := Text(before.Get())
			after := NewState(c, "after")
			return Column(beforeText, fb, Text(after.Get())).Render(c)
		})
	}

	for i, n := range []int{3, 1, 5, 0} {
		hooksBeforePanic = n
		node := pass(ctx, build())
		if !findText(node, "before") || !findText(node, "after") {
			t.Fatalf("pass %d (child used %d hooks): sibling state was corrupted; tree = %+v", i, n, node)
		}
	}

	// The boundary's own footprint in the parent context is fixed at two
	// slots (child context + fallback context) whatever the child did.
	if len(ctx.slots) != 4 {
		t.Errorf("parent allocated %d slots, want 4 (before, childCtx, fallbackCtx, after)", len(ctx.slots))
	}
}

func TestErrorBoundaryRewindsCallbackIDsOfTheFailedSubtree(t *testing.T) {
	// Callback IDs are positional within a pass, so a child that registers a
	// varying number of handlers before panicking would otherwise shift every
	// later component's IDs — and taps would land on the wrong handler.
	ctx := NewContext()
	buttonsBeforePanic := 0
	siblingClicks := 0

	build := func() View {
		child := ComponentFunc(func(cc *Context) *Node {
			for i := 0; i < buttonsBeforePanic; i++ {
				Button("doomed", func() { t.Error("a handler from the failed subtree ran") }).Render(cc)
			}
			panic("child failed")
		})
		return Column(
			ErrorBoundary(child, func(error) View { return Text("fb") }),
			Button("sibling", func() { siblingClicks++ }),
		)
	}

	var siblingID string
	for _, n := range []int{2, 0, 4} {
		buttonsBeforePanic = n
		node := pass(ctx, build())

		btn := node.Children[1]
		id := btn.Props["onClick"].(string)
		if siblingID == "" {
			siblingID = id
		} else if id != siblingID {
			t.Fatalf("sibling callback ID moved to %q (was %q) when the child registered %d handlers before panicking", id, siblingID, n)
		}

		// The rewind is only worth anything if the ID still dispatches to the
		// sibling's handler and the abandoned handlers are gone.
		want := siblingClicks + 1
		ctx.TriggerCallback(id)
		if siblingClicks != want {
			t.Fatalf("sibling handler did not run for ID %q", id)
		}
		// The failed subtree's handlers occupied cb_0..cb_(n-1); the rewind
		// hands cb_0 back to the sibling, so cb_1 upward must have been
		// un-marked and collected by the purge rather than left dispatchable.
		if n >= 2 {
			if _, ok := ctx.registry.lookupVoid("cb_1"); ok {
				t.Errorf("an abandoned handler (cb_1) from the failed subtree survived the purge")
			}
		}
	}
}

func TestErrorBoundaryDoesNotLatch(t *testing.T) {
	// Deliberately unlike React: the tree is rebuilt every pass, so a
	// transient failure must heal on its own rather than pin a dead panel.
	ctx := NewContext()
	failing := true
	view := ErrorBoundary(
		ComponentFunc(func(c *Context) *Node {
			if failing {
				panic("transient")
			}
			return Text("real content").Render(c)
		}),
		func(error) View { return Text("fallback") },
	)

	if node := pass(ctx, view); !findText(node, "fallback") {
		t.Fatal("first pass should have rendered the fallback")
	}
	failing = false
	node := pass(ctx, view)
	if !findText(node, "real content") {
		t.Fatal("boundary latched: it kept the fallback after the child recovered")
	}
	if findText(node, "fallback") {
		t.Error("fallback is still in the tree alongside the recovered child")
	}
}

func TestErrorBoundaryContainsAPanickingFallback(t *testing.T) {
	// A fallback that panics must not defeat the boundary that exists to stop
	// panics. Both the construction call and the render are covered.
	ctx := NewContext()
	for _, tc := range []struct {
		name     string
		fallback func(error) View
	}{
		{"panics while building", func(error) View { panic("fallback build failed") }},
		{"panics while rendering", func(error) View { return boom("fallback render failed") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := pass(ctx, ErrorBoundary(boom("child failed"), tc.fallback))
			if node == nil {
				t.Fatal("boundary returned a nil node")
			}
			if !findText(node, "Something went wrong") {
				t.Errorf("want the last-resort placeholder, got %+v", node)
			}
		})
	}
}

func TestDefaultErrorFallbackHidesDetailOutsideDebugMode(t *testing.T) {
	// A panic message is developer-facing text. It identifies the bug in a
	// debug build and leaks internals into a user's screenshot in a release
	// one, so the detail line is gated.
	ctx := NewContext()
	view := SafeRender(boom("index out of range [7]"))

	node := pass(ctx, view)
	if findText(node, "index out of range") {
		t.Error("release build exposed the raw panic message")
	}
	if !findText(node, "Something went wrong") {
		t.Error("release build showed no error message at all")
	}

	SetDebugMode(true)
	defer SetDebugMode(false)
	defer ClearConcerns()
	node = pass(ctx, view)
	if !findText(node, "index out of range") {
		t.Error("debug build hid the panic message")
	}
}

func TestErrorBoundaryReportsAConcernInDebugMode(t *testing.T) {
	SetDebugMode(true)
	defer SetDebugMode(false)
	ClearConcerns()
	defer ClearConcerns()

	ctx := NewContext()
	pass(ctx, ErrorBoundary(boom("kaboom"), func(error) View { return Text("fb") }))

	var found *Concern
	for i, c := range Concerns() {
		if c.Kind == ConcernRenderPanic {
			found = &Concerns()[i]
		}
	}
	if found == nil {
		t.Fatalf("no %s concern recorded; got %+v", ConcernRenderPanic, Concerns())
	}
	if !strings.Contains(found.Detail, "kaboom") {
		t.Errorf("concern detail = %q, want it to name the panic value", found.Detail)
	}
}

func TestErrorBoundaryLeavesNoPhantomCursorDrift(t *testing.T) {
	// A truncated child context ends the pass with a cursor short of its slot
	// count, which is the exact signature of the conditional-hook bug the
	// drift audit hunts for. Zeroing the cursor on recovery is what keeps the
	// audit from reporting a second, imaginary problem on top of the real one.
	SetDebugMode(true)
	defer SetDebugMode(false)
	ClearConcerns()
	defer ClearConcerns()

	ctx := NewContext()
	hooksBeforePanic := 3
	view := ErrorBoundary(
		ComponentFunc(func(cc *Context) *Node {
			for i := 0; i < hooksBeforePanic; i++ {
				NewState(cc, i)
			}
			panic("child failed")
		}),
		func(error) View { return Text("fb") },
	)

	pass(ctx, view)
	ctx.EndRenderPass()
	hooksBeforePanic = 1
	pass(ctx, view)
	ctx.EndRenderPass()

	for _, c := range Concerns() {
		if c.Kind == ConcernCursorDrift {
			t.Errorf("phantom cursor-drift concern from a recovered panic: %s", c.Detail)
		}
	}
}

func TestGuardReturnsNilWhenNothingPanics(t *testing.T) {
	ran := false
	if rerr := Guard(func() { ran = true }); rerr != nil {
		t.Fatalf("Guard reported %v for a clean call", rerr)
	}
	if !ran {
		t.Error("Guard did not run fn")
	}
}
