package hooks

import (
	"reflect"

	"github.com/rohanthewiz/grmob/core"
)

// effectRecord is the per-hook-slot memory of UseEffect: whether the effect
// has ever run for this slot, and the deps it last ran with. It lives in the
// context's own hook-slot array (seeded via core.NewState), which is what
// makes the hook correct without any package-level storage:
//
//   - identity comes from the slot position, so the same UseEffect call site
//     reads the same record on every render — the former global index only
//     ever counted upward, so every render saw fresh indices and re-ran every
//     effect regardless of deps;
//   - two components (or two whole apps) at the same cursor position have
//     different slots, hence different records — no cross-talk;
//   - no per-pass reset step is needed, so nothing depends on hosts
//     remembering to call it (the old ResetEffects had no callers).
//
// No mutex: the record is only touched during render passes, which the render
// manager serializes; the spawned effect goroutine never sees the record.
type effectRecord struct {
	ran  bool
	deps []any
}

// UseEffect runs effect when the hook first mounts and again whenever deps
// change between renders (compared with reflect.DeepEqual). With no deps it
// runs exactly once for the lifetime of the slot.
//
// The effect runs on its own goroutine so a slow effect cannot stall the
// render pass; anything it changes via State.Set reaches the screen through
// the normal RequestRender → push-channel path.
func UseEffect(ctx *core.Context, effect func(), deps ...any) {
	// The record is allocated every render but NewState only keeps the first;
	// Get then returns the slot's live record on this and every later pass.
	slot := core.NewState(ctx, &effectRecord{})
	rec := slot.Get()

	if rec.ran && reflect.DeepEqual(rec.deps, deps) {
		return
	}
	rec.ran = true
	// Copy the deps rather than retaining the caller's slice. A caller that
	// spreads a slice it owns (UseEffect(ctx, fn, args...)) would otherwise
	// have the record alias that slice: a later mutation would rewrite the
	// stored deps in lockstep with the new ones, DeepEqual would report
	// "unchanged", and the effect would never run again. Mirrors UseMemo.
	rec.deps = append([]any(nil), deps...)
	go effect()
}
