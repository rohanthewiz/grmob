package permission

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// Re-checking on foreground: the half of a permission that no platform
// announces.
//
// # The fact this file exists for
//
// A user who is refused something taps "Open Settings", grants it there, and
// comes back. Nothing tells the app that happened. There is no permission-
// changed callback on iOS, none on Android, and the browser's
// PermissionStatus.onchange covers only the three descriptors it has and only
// while the page is alive. The one signal every platform does send is the app
// coming back to the foreground, which core.OnLifecycle already carries — so
// the re-check is a lifecycle subscription plus a Check, and it has always
// been writable by hand:
//
//	state := hooks.UseLifecycle(ctx)
//	hooks.UseEffect(ctx, func() {
//	    if state == core.LifecycleActive { permission.Check(permission.Camera) }
//	}, state)
//
// Every consumer wrote that. This file is the same thing written once.
//
// # Who owns the re-check, which is the question that kept this open
//
// hooks.UsePermission deliberately does not re-check, and its doc says why: a
// hook cannot see whether its screen is still the one on top, and an app with
// five screens in a stack would fire five checks per resume. That is a real
// objection to putting the subscription in the hook, and for a while it read
// as an objection to the feature.
//
// It is not, because it assumes a *screen* has to own the re-check. Nothing
// about the question is per-screen: there is one device with one camera, and
// "is the camera permission still what it was" has one answer no matter how
// many screens are asking. So the owner is the permission, and the shape is
// core.StartHeading's — a reference count, one subscription behind it, and a
// check per *permission* per resume rather than per watcher:
//
//	five screens watching Camera        one "permission" system event per resume
//	one screen watching Camera+Location two, one for each kind
//	no screens watching anything        no lifecycle subscription at all
//
// The last line is the one that makes this cheap. An app that never asks for
// a live permission never subscribes to the lifecycle here, so it renders and
// resumes exactly as it did before this file existed.
//
// # Why a check on resume is nearly free when nothing changed
//
// Check is a question, not a sensor: it costs one system event out and one
// host event back, and there is no device to power up. And permission.set
// notifies only on a *change*, so the overwhelmingly common case — the user
// switched apps and came back having changed nothing — reaches Go, matches
// the recorded status and stops there. No subscriber runs and no render is
// requested. The cost of watching is a round trip per resume per permission,
// and the benefit is that the one resume that did change something is seen.
//
// # Only into active, and only on a transition
//
// core.OnLifecycle notifies on transitions and dedupes repeats (the hosts
// report their platform callbacks verbatim, and some of those restate the
// current state), so a resume fires this once. "inactive" and "background"
// are ignored: the system permission dialog itself puts an iOS app into
// inactive, and re-checking there would race the answer that is about to
// arrive through the request path anyway.

var (
	fgMu sync.Mutex
	// fgWatchers is the reference count per permission — how many live
	// watchers want this kind re-checked. A kind at zero is deleted rather
	// than left holding a 0, so the resume loop iterates only what is
	// actually watched.
	fgWatchers = map[Permission]int{}
	// fgCancel releases the one lifecycle subscription. Non-nil exactly while
	// some kind is watched; see the "no screens watching" line above.
	fgCancel func()
)

// WatchForeground asks for p to be re-checked every time the app returns to
// the foreground, and returns the function that stops asking.
//
// Balanced rather than idempotent, exactly as core.StartHeading/StopHeading
// are and for the same failure: two screens both watching the camera, one of
// them closing, and a plain on/off flag turning the re-check off under the
// other one with nothing in any log. Calling the returned cancel more than
// once is harmless — the second call finds the watch already released and
// does nothing, so a component that both defers a cancel and calls it on an
// error path cannot drive the count negative.
//
// It does not check anything itself. A watcher registering says what should
// happen on the *next* resume; the opening check is the caller's, which for
// almost every caller means hooks.UsePermission doing it on mount.
// hooks.UsePermissionLive is the two together.
//
// Safe from any goroutine.
func WatchForeground(p Permission) (cancel func()) {
	if !valid(p) {
		// An unspelled permission has no host arm and would make the resume
		// loop emit an event no shell answers. Refused here rather than
		// counted, and the cancel is still a real function so a caller's
		// defer needs no nil check.
		return func() {}
	}

	fgMu.Lock()
	fgWatchers[p]++
	if fgCancel == nil {
		// First watcher of any kind: take the one subscription. Inside the
		// lock so two goroutines registering at once cannot both decide they
		// are first and leave one subscription unreferenced forever.
		fgCancel = core.OnLifecycle(onLifecycle)
	}
	fgMu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() { unwatchForeground(p) })
	}
}

// unwatchForeground releases one watch and drops the lifecycle subscription
// with the last one.
//
// The cancel call happens outside the lock: core.OnLifecycle's cancel takes
// core's own lifecycle mutex, and a resume being delivered on another
// goroutine at that moment holds that mutex while running onLifecycle, which
// wants this one. Releasing here first is what keeps the two from meeting
// head on.
func unwatchForeground(p Permission) {
	fgMu.Lock()
	n := fgWatchers[p]
	if n <= 1 {
		delete(fgWatchers, p)
	} else {
		fgWatchers[p] = n - 1
	}
	var release func()
	if len(fgWatchers) == 0 && fgCancel != nil {
		release, fgCancel = fgCancel, nil
	}
	fgMu.Unlock()

	if release != nil {
		release()
	}
}

// WatchedForeground reports how many watchers p has. Bookkeeping for tests
// and a debug overlay, the same job core.HeadingActive does for the sensor;
// nothing in an app should need it.
func WatchedForeground(p Permission) int {
	fgMu.Lock()
	defer fgMu.Unlock()
	return fgWatchers[p]
}

// onLifecycle is the subscription's body: one Check per watched permission on
// the way back to active.
//
// The snapshot under the lock, checked outside it, is the pattern set uses
// for subscribers and for the same reason — Check reaches the host, a
// headless host answers synchronously inside the call, and an answer running
// a subscriber that registers another watch would deadlock on a held fgMu.
func onLifecycle(state core.LifecycleState) {
	if state != core.LifecycleActive {
		return
	}
	fgMu.Lock()
	// Declaration order rather than map order: the events leave in the order
	// Permissions() lists, so a shell's log and a test's expectations read the
	// same way twice running. Four constants, so the scan costs nothing.
	kinds := make([]Permission, 0, len(fgWatchers))
	for _, p := range Permissions() {
		if fgWatchers[p] > 0 {
			kinds = append(kinds, p)
		}
	}
	fgMu.Unlock()

	for _, p := range kinds {
		Check(p)
	}
}

// resetForegroundForTest drops every watch and the subscription behind them,
// so one test's watchers cannot re-check inside the next.
func resetForegroundForTest() {
	fgMu.Lock()
	release := fgCancel
	fgCancel = nil
	fgWatchers = map[Permission]int{}
	fgMu.Unlock()

	if release != nil {
		release()
	}
}
