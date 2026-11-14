package core

import "testing"

// The dirty flag's generation (N-138, plan P1): a pass clears only the writes
// it could have seen, and the mark never moves backwards.
func TestClearDirtyThroughLeavesALaterWriteStanding(t *testing.T) {
	ctx := NewContext()
	if ctx.IsDirty() {
		t.Fatal("a fresh context is dirty")
	}
	ctx.MarkDirty()
	gen := ctx.DirtyGeneration() // a pass starts here
	ctx.MarkDirty()              // and something writes while it builds
	ctx.ClearDirtyThrough(gen)
	if !ctx.IsDirty() {
		t.Fatal("the write made during the pass was cleared with the ones before it")
	}

	// A newer pass consumes it; an older pass finishing late must not
	// re-dirty what the newer one drew.
	newer := ctx.DirtyGeneration()
	ctx.ClearDirtyThrough(newer)
	ctx.ClearDirtyThrough(gen)
	if ctx.IsDirty() {
		t.Fatal("a stale ClearDirtyThrough moved the mark backwards")
	}

	// The derived contexts share the flag, as the bool did.
	child := ctx.WithTheme(DefaultTheme)
	child.MarkDirty()
	if !ctx.IsDirty() {
		t.Fatal("a write in a derived context did not dirty the app")
	}
	ctx.ClearDirty()
	if child.IsDirty() {
		t.Fatal("ClearDirty did not clear the shared flag")
	}
}

// A Set of the value a scalar slot already holds requests no pass; any other
// write, and any write of a non-scalar, still does.
func TestSettingAScalarToItsOwnValueRequestsNoRender(t *testing.T) {
	ctx := NewContext()
	var nudges int
	ctx.OnStateChange(func() { nudges++ })

	open := NewState(ctx, false)
	name := NewState(ctx, "Ana")
	list := NewState(ctx, []int{1})

	open.Set(false)
	name.Set("Ana")
	if nudges != 0 || ctx.IsDirty() {
		t.Fatalf("unchanged scalar writes nudged %d times (dirty %v), want none", nudges, ctx.IsDirty())
	}
	open.Set(true)
	name.Set("Ben")
	if nudges != 2 || !ctx.IsDirty() {
		t.Fatalf("two changed scalar writes nudged %d times, want 2", nudges)
	}
	list.Set([]int{1}) // not comparable with ==: always a change
	if nudges != 3 {
		t.Fatalf("a slice write nudged %d times in all, want 3", nudges)
	}
	if !open.Get() || name.Get() != "Ben" {
		t.Fatal("the changed writes did not land")
	}
}
