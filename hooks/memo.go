package hooks

import (
	"reflect"

	"github.com/rohanthewiz/grmob/core"
)

// memoRecord is the per-hook-slot memory of one UseMemo: whether a value has
// ever been computed for this slot, the deps it was computed with, and the
// value itself. It lives in the context's hook-slot array (seeded via
// core.NewState) for the same reasons effectRecord does — slot position is
// the hook's identity, so the same call site reads the same record on every
// pass, and two components (or two apps) landing on the same cursor have
// different slots and therefore cannot see each other's cached value.
//
// No mutex: the record is read and written only during render passes, which
// render.Manager serializes. Unlike the interval/timeout records, nothing
// here is ever touched from a background goroutine.
type memoRecord[T any] struct {
	computed bool
	deps     []any
	value    T
}

// UseMemo returns the result of compute, recomputing it only when deps change
// between renders (compared with reflect.DeepEqual). With no deps it computes
// exactly once for the lifetime of the slot — the same mount-once rule
// UseEffect follows for a depless effect.
//
// It exists for work that is expensive relative to a render pass — parsing,
// sorting or filtering a large slice, building a derived index — since a
// render function is re-run in full on every pass:
//
//	visible := hooks.UseMemo(ctx, func() []Todo {
//	    return filterAndSort(todos.Get(), filter.Get())
//	}, todos.Get(), filter.Get())
//
// Two things it deliberately is not:
//
//   - It is not a correctness tool. compute must be pure, and callers must
//     treat the returned value as read-only — the same value is handed back
//     on every cache hit, so mutating it corrupts later renders.
//   - There is no UseCallback counterpart. Memoizing a closure only pays off
//     in a framework that skips subtrees on unchanged prop identity; here the
//     reconciler diffs the rendered tree instead, so a stable closure buys
//     nothing.
//
// compute runs inline on the render goroutine (not on its own goroutine like
// UseEffect) because its result is needed to build this pass's view.
func UseMemo[T any](ctx *core.Context, compute func() T, deps ...any) T {
	// The record is allocated on every render but NewState keeps only the
	// first; Get then returns the slot's live record on this and every later
	// pass. Going through NewState also reserves the cursor slot properly —
	// a bare Cursor++ would leave the slot array one short and make every
	// later hook in the pass read its neighbour's slot.
	slot := core.NewState(ctx, &memoRecord[T]{})
	rec := slot.Get()

	if rec.computed && reflect.DeepEqual(rec.deps, deps) {
		return rec.value
	}

	rec.computed = true
	// Copy the deps rather than retaining the caller's slice. A caller that
	// spreads a slice it owns (UseMemo(ctx, f, args...)) would otherwise have
	// the record alias that slice: a later mutation would change the stored
	// deps in lockstep with the new ones, DeepEqual would report "unchanged",
	// and the memo would go permanently stale.
	rec.deps = append([]any(nil), deps...)
	rec.value = compute()
	return rec.value
}
