package core

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestCachedRendersExactlyOnce(t *testing.T) {
	ctx := NewContext()
	var renders atomic.Int32
	cached := Cached(ComponentFunc(func(ctx *Context) *Node {
		renders.Add(1)
		return &Node{Type: "Text", Props: map[string]any{"content": "static"}}
	}))

	// Simulate consecutive render passes: same wrapper, fresh calls.
	first := cached.Render(ctx)
	second := cached.Render(ctx)
	third := cached.Render(ctx)

	if got := renders.Load(); got != 1 {
		t.Errorf("wrapped view rendered %d times, want exactly 1", got)
	}
	// Pointer identity across passes is the whole point: it is the signal
	// reconcile.Diff short-circuits on.
	if first != second || second != third {
		t.Errorf("Cached returned distinct nodes across passes: %p, %p, %p", first, second, third)
	}
	if first.Type != "Text" || first.Props["content"] != "static" {
		t.Errorf("cached node content mangled: %+v", first)
	}
}

// TestCachedConcurrentRender drives Render from many goroutines at once;
// under -race this verifies sync.Once both serializes the single underlying
// render and publishes the node to callers that lost the race.
func TestCachedConcurrentRender(t *testing.T) {
	ctx := NewContext()
	var renders atomic.Int32
	cached := Cached(ComponentFunc(func(ctx *Context) *Node {
		renders.Add(1)
		return &Node{Type: "Text", Props: map[string]any{"content": "static"}}
	}))

	const goroutines = 16
	nodes := make([]*Node, goroutines)
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Go(func() {
			nodes[i] = cached.Render(ctx)
		})
	}
	wg.Wait()

	if got := renders.Load(); got != 1 {
		t.Errorf("wrapped view rendered %d times under contention, want exactly 1", got)
	}
	for i, n := range nodes {
		if n == nil {
			t.Fatalf("goroutine %d observed a nil node — Once published before the render completed", i)
		}
		if n != nodes[0] {
			t.Errorf("goroutine %d got a different node pointer (%p vs %p)", i, n, nodes[0])
		}
	}
}
