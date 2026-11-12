// Package hooks is the layer of conveniences built on core's state slots:
// effects, timers, memoisation, a reducer, and read-only views of the things
// the host reports asynchronously (audio status, location, lifecycle,
// permission decisions).
//
// Everything here is built from [core.NewState] and [core.Context.OnClose] and
// could be written by an app instead. What the package adds is the part that is
// easy to get wrong — deciding when to re-run, and cleaning up when the context
// closes.
//
// # Slots, and therefore order
//
// Every hook here occupies one or more of the calling context's slots, and
// slots are positional (see [core.NewState]). So the rules of hooks apply to
// this package exactly as they do to core's: call them unconditionally, in the
// same order, on every pass. A hook behind an `if` does not merely skip itself;
// it shifts every hook after it onto the wrong slot.
//
// # Dependencies
//
//	UseEffect(ctx, fn)             once, when the slot first mounts
//	UseEffect(ctx, fn, a, b)       on mount, and when a or b changes from the
//	                               last pass
//	UseAsync(ctx, fetch, a, b)     the same schedule, with the run before it
//	                               cancelled and its answer dropped
//
// Comparison is by value equality on the dependency list. A dependency that is
// a function value or a freshly built slice compares unequal every pass, which
// turns a conditional effect into an unconditional one — silently, since the
// effect still does the right thing, just far more often than intended.
//
// # Timers hold the latest closure
//
// [UseInterval] and [UseTimeout] re-bind their function on every pass, so the
// callback that eventually fires is the one from the most recent render and
// sees current state rather than the state of the pass that started the timer.
// Both stop themselves when the context closes, which is what makes a ticker
// inside a screen safe to leave to the navigator.
//
// # Async work
//
// [UseEffect] runs its function on a goroutine and forgets it: nothing
// cancels a fetch whose deps have moved on, and a slow answer for the old
// deps can land after a fast one for the new. [UseAsync] is the fetch shape
// with both handled. It hands fn a context that is cancelled when the deps
// change, on Reload and on close, keeps only the latest run's answer, and
// returns Data, Err and Loading for the screen to branch on.
package hooks
