package hooks_test

import (
	"sync"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

func TestUseMemoRecomputesOnlyWhenDepsChange(t *testing.T) {
	ctx := core.NewContext()
	computes := 0

	// The memo's value is derived from dep so a stale cache is visible in the
	// returned value, not only in the compute count.
	render := func(dep int) string {
		var got string
		renderPass(ctx, func(ctx *core.Context) {
			got = hooks.UseMemo(ctx, func() string {
				computes++
				return string(rune('a' + dep))
			}, dep)
		})
		return got
	}

	if got := render(0); got != "a" || computes != 1 {
		t.Fatalf("mount: value %q after %d computes, want \"a\" after 1", got, computes)
	}
	if got := render(0); got != "a" || computes != 1 {
		t.Fatalf("unchanged deps: value %q after %d computes, want \"a\" after 1", got, computes)
	}
	if got := render(1); got != "b" || computes != 2 {
		t.Fatalf("changed deps: value %q after %d computes, want \"b\" after 2", got, computes)
	}
	// Changing back is still a change from the *last* deps, so it recomputes:
	// the record holds one generation of deps, not a history of them.
	if got := render(0); got != "a" || computes != 3 {
		t.Fatalf("deps back to 0: value %q after %d computes, want \"a\" after 3", got, computes)
	}
}

func TestUseMemoWithNoDepsComputesOnce(t *testing.T) {
	ctx := core.NewContext()
	computes := 0

	body := func(ctx *core.Context) {
		hooks.UseMemo(ctx, func() int {
			computes++
			return computes
		})
	}

	for range 3 {
		renderPass(ctx, body)
	}
	if computes != 1 {
		t.Fatalf("depless memo computed %d times, want 1", computes)
	}
}

func TestUseMemoSlotsAreIndependent(t *testing.T) {
	// Two memos in one component, plus the same shape in a second app: four
	// distinct slots. A memo keyed on cursor alone (or on any package-level
	// store) would let these overwrite each other's cached values.
	type harness struct {
		ctx      *core.Context
		computes [2]int
	}
	render := func(h *harness, depA, depB int) (string, string) {
		var a, b string
		renderPass(h.ctx, func(ctx *core.Context) {
			a = hooks.UseMemo(ctx, func() string {
				h.computes[0]++
				return "A" + string(rune('0'+depA))
			}, depA)
			b = hooks.UseMemo(ctx, func() string {
				h.computes[1]++
				return "B" + string(rune('0'+depB))
			}, depB)
		})
		return a, b
	}

	one := &harness{ctx: core.NewContext()}
	two := &harness{ctx: core.NewContext()}
	render(one, 0, 0)
	render(two, 0, 0)

	// Change only app one's second dep: exactly that memo recomputes.
	a, b := render(one, 0, 1)
	if a != "A0" || b != "B1" {
		t.Fatalf("app one values = %q/%q, want A0/B1", a, b)
	}
	if one.computes != [2]int{1, 2} {
		t.Fatalf("app one computes = %v, want [1 2]", one.computes)
	}

	a, b = render(two, 0, 0)
	if a != "A0" || b != "B0" {
		t.Fatalf("app two values = %q/%q, want A0/B0", a, b)
	}
	if two.computes != [2]int{1, 1} {
		t.Fatalf("app two computes = %v, want [1 1] (app one's change leaked)", two.computes)
	}
}

func TestUseMemoDoesNotAliasCallerDeps(t *testing.T) {
	// A caller spreading a slice it owns: the record must hold a copy, or the
	// caller's mutation would rewrite the stored deps too and DeepEqual would
	// report "unchanged" forever.
	ctx := core.NewContext()
	deps := []any{1}
	computes := 0

	render := func() int {
		var got int
		renderPass(ctx, func(ctx *core.Context) {
			got = hooks.UseMemo(ctx, func() int {
				computes++
				return deps[0].(int)
			}, deps...)
		})
		return got
	}

	render()
	deps[0] = 2
	if got := render(); got != 2 || computes != 2 {
		t.Fatalf("after mutating the caller's deps slice: value %d after %d computes, want 2 after 2", got, computes)
	}
}

// counterAction is the action type for the reducer tests below.
type counterAction int

const (
	increment counterAction = iota
	reset
)

func countReducer(state int, action counterAction) int {
	switch action {
	case increment:
		return state + 1
	case reset:
		return 0
	}
	return state
}

func TestUseReducerDispatchUpdatesStateAndRequestsRender(t *testing.T) {
	ctx := core.NewContext()

	var count int
	var dispatch func(counterAction)

	renderPass(ctx, func(ctx *core.Context) {
		count, dispatch = hooks.UseReducer(ctx, countReducer, 5)
	})
	if count != 5 {
		t.Fatalf("mount state = %d, want the initial 5", count)
	}

	ctx.ClearDirty()
	dispatch(increment)
	dispatch(increment)
	if !ctx.IsDirty() {
		t.Fatal("dispatch did not request a render")
	}

	// A re-render re-reads the live record — and passes a different initial,
	// which must be ignored now that the slot exists.
	renderPass(ctx, func(ctx *core.Context) {
		count, dispatch = hooks.UseReducer(ctx, countReducer, 99)
	})
	if count != 7 {
		t.Fatalf("state after two increments = %d, want 7", count)
	}

	dispatch(reset)
	renderPass(ctx, func(ctx *core.Context) {
		count, dispatch = hooks.UseReducer(ctx, countReducer, 99)
	})
	if count != 0 {
		t.Fatalf("state after reset = %d, want 0", count)
	}
}

func TestUseReducerConcurrentDispatchesLoseNothing(t *testing.T) {
	// The reason UseReducer keeps its state behind its own mutex instead of
	// doing slot.Set(reducer(slot.Get(), a)): the naive form is a
	// read-modify-write across two atomic operations, so concurrent
	// dispatches drop updates. Run with -race.
	ctx := core.NewContext()

	var dispatch func(counterAction)
	renderPass(ctx, func(ctx *core.Context) {
		_, dispatch = hooks.UseReducer(ctx, countReducer, 0)
	})

	const n = 200
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			dispatch(increment)
		}()
	}
	wg.Wait()

	var count int
	renderPass(ctx, func(ctx *core.Context) {
		count, _ = hooks.UseReducer(ctx, countReducer, 0)
	})
	if count != n {
		t.Fatalf("state after %d concurrent increments = %d, want %d", n, count, n)
	}
}

func TestUseReducerSlotsAreIndependent(t *testing.T) {
	// Same-cursor reducers in two apps must not share state.
	newApp := func() (*core.Context, func(counterAction), func() int) {
		ctx := core.NewContext()
		var dispatch func(counterAction)
		read := func() int {
			var c int
			renderPass(ctx, func(ctx *core.Context) {
				c, dispatch = hooks.UseReducer(ctx, countReducer, 0)
			})
			return c
		}
		read()
		return ctx, func(a counterAction) { dispatch(a) }, read
	}

	_, dispatchOne, readOne := newApp()
	_, _, readTwo := newApp()

	dispatchOne(increment)
	if got := readOne(); got != 1 {
		t.Fatalf("app one state = %d, want 1", got)
	}
	if got := readTwo(); got != 0 {
		t.Fatalf("app two state = %d, want 0 (app one's dispatch leaked)", got)
	}
}
