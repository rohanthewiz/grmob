package hooks

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// Async is what UseAsync returns on each render: the state of the async
// work for the current deps.
//
// Data and Err describe the last run that finished for these deps; Loading
// says a newer run is in flight. They change together, so a screen picks
// one branch per render:
//
//	switch {
//	case a.Err != nil:  // the last run failed (Reload clears it)
//	case a.Loading:     // nothing finished yet, or a reload is running
//	default:            // a.Data is the answer
//	}
//
// Data is kept through a Reload, so a list can stay on screen under a
// refresh indicator, and dropped when the deps change, because it was the
// answer to a different question.
type Async[T any] struct {
	Data    T
	Err     error
	Loading bool

	// Reload starts the work again with the same deps, cancelling a run in
	// flight. It is safe to call from any goroutine (a handler, a timer, a
	// pull-to-refresh) and does nothing once the hook's tree has closed. It
	// clears Err at once, so a Retry button's tap shows Loading rather than
	// the error it is retrying.
	Reload func()
}

// ErrAsyncPanicked is the Err of a run whose function panicked. The panic
// itself is recovered and logged with its stack (and reported as a
// handler-panic concern under debug mode), as core.Safely does for every
// hook goroutine; the screen gets an error it can show and a Reload it can
// offer.
var ErrAsyncPanicked = errors.New("hooks.UseAsync: the function panicked (the panic and its stack are in the log)")

// asyncRecord is one UseAsync slot's memory, kept in the context's hook slot
// like effectRecord.
//
// Unlike effectRecord it has a mutex: the run's goroutine writes the
// outcome, Reload may be called from any goroutine, and renders read it.
//
// # Generations
//
// Every start (a mount, a deps change, a Reload) increments gen and gives
// the run its own context. A finishing run writes its outcome only if its
// generation is still the current one, so a slow first fetch that returns
// after a fast second one is dropped instead of overwriting it:
//
//	render deps=1 ──▶ run g1 ──────────────────────────────▶ done (g1 ≠ g2: dropped)
//	render deps=2 ──▶ cancel g1, run g2 ──▶ done (g2: Data, render)
//
// Close also increments gen and cancels, so nothing lands on a closed tree.
// The cancellation is advice to fn (a well-behaved fetch returns early); the
// generation check is what makes a result that ignores it harmless.
type asyncRecord[T any] struct {
	mu sync.Mutex

	// started is true from the first render of a mount until the tree
	// closes. Close resets it, so a render after a remount starts again, as
	// the timer hooks do (TestUseIntervalResumesAfterCloseAndRemount).
	started bool
	deps    []any
	gen     uint64
	cancel  context.CancelFunc

	// fn is the latest render's closure, so a Reload runs what the screen
	// last asked for, with the values it last captured.
	fn  func(context.Context) (T, error)
	ctx *core.Context

	data    T
	err     error
	loading bool

	// reload is built once per slot so every render hands out the same
	// func value.
	reload func()
}

// UseAsync runs fn on its own goroutine when the hook first mounts and again
// whenever deps change between renders (compared with reflect.DeepEqual, as
// UseEffect does), and returns where that work stands.
//
//	verse := hooks.UseAsync(ctx, func(c context.Context) (blb.Passage, error) {
//		return client.Passage(c, ref)
//	}, ref)
//
//	comps.BibleVerse{
//		Passage: verse.Data,
//		Loading: verse.Loading,
//		Error:   errText(verse.Err),
//		OnRetry: verse.Reload,
//	}
//
// It is the fetch pattern UseEffect leaves to the caller, done once:
//
//   - fn's context is cancelled when the deps change, on Reload, and when
//     the tree closes (render.Manager.Close, a popped route), so a fetch
//     built on it stops;
//   - only the latest run's outcome is kept: a run that finishes after a
//     newer one started is dropped, whatever order the network answers in;
//   - the outcome reaches the screen with a render request, like any state
//     written off the render pass;
//   - a panic in fn is recovered and logged, and becomes ErrAsyncPanicked.
//
// The first render of a mount already reports Loading, with no extra render.
//
// Capture what fn needs at render time, as with UseEffect: the closure runs
// later, on another goroutine. A dep is anything fn's answer depends on; a
// value fn reads but the deps omit will not trigger a new run when it
// changes.
func UseAsync[T any](ctx *core.Context, fn func(context.Context) (T, error), deps ...any) Async[T] {
	slot := core.NewState(ctx, &asyncRecord[T]{})
	rec := slot.Get()

	rec.mu.Lock()
	defer rec.mu.Unlock()

	rec.fn = fn
	rec.ctx = ctx
	if rec.reload == nil {
		rec.reload = rec.reloadFromAnywhere
	}

	switch {
	case !rec.started:
		rec.started = true
		// Once per mount: Close drains its registrations, and the reset of
		// started is what makes the next mount register again.
		ctx.OnClose(rec.close)
		rec.restartLocked(deps, true)
	case !reflect.DeepEqual(rec.deps, deps):
		rec.restartLocked(deps, true)
	}
	return Async[T]{Data: rec.data, Err: rec.err, Loading: rec.loading, Reload: rec.reload}
}

// restartLocked cancels any run in flight and starts one for deps. fresh
// says the deps are new (a mount or a change), so the outcome of the old
// ones is dropped; a Reload keeps Data and clears only Err.
func (r *asyncRecord[T]) restartLocked(deps []any, fresh bool) {
	if r.cancel != nil {
		r.cancel()
	}
	// Copied, not retained: see UseEffect's note on a caller's spread slice.
	r.deps = append([]any(nil), deps...)
	r.gen++
	gen := r.gen
	if fresh {
		var zero T
		r.data = zero
	}
	r.err = nil
	r.loading = true

	runCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	fn, owner := r.fn, r.ctx
	go r.run(gen, runCtx, cancel, fn, owner)
}

// run is one generation's goroutine.
func (r *asyncRecord[T]) run(gen uint64, runCtx context.Context, cancel context.CancelFunc,
	fn func(context.Context) (T, error), owner *core.Context) {
	defer cancel() // release the context's resources whatever happens

	var (
		v   T
		err error
	)
	if !core.Safely("UseAsync", func() { v, err = fn(runCtx) }) {
		err = ErrAsyncPanicked
	}

	r.mu.Lock()
	if gen != r.gen {
		// Superseded by a newer run, or the tree closed.
		r.mu.Unlock()
		return
	}
	if err == nil {
		r.data = v
	}
	r.err = err
	r.loading = false
	r.cancel = nil
	r.mu.Unlock()

	owner.RequestRender()
}

// reloadFromAnywhere is Async.Reload.
func (r *asyncRecord[T]) reloadFromAnywhere() {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return
	}
	r.restartLocked(r.deps, false)
	owner := r.ctx
	r.mu.Unlock()

	// Loading is now true and Err cleared; the screen should say so before
	// the run finishes, so ask for the render rather than wait for it.
	owner.RequestRender()
}

// close is the slot's OnClose: cancel the run, make its outcome stale, and
// let the next mount start over.
func (r *asyncRecord[T]) close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.gen++
	r.started = false
	r.loading = false
}
