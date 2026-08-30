// Package mobile is the gomobile-bindable bridge surface for native shells.
//
// The rest of the framework's API is not bind-safe (gomobile cannot bind
// function parameters, generics, or map-typed exports), so this package
// narrows everything the native side needs to strings, bools, and a
// single-method interface. An app author writes their root view in Go, wires
// it in an init step, and binds this package plus their own:
//
//	// app's Go code
//	func init() { mobile.Register(core.NewContext(), App) }
//
//	// Kotlin (via the gomobile-generated classes)
//	Mobile.setListener { patches -> runOnUiThread { renderer.applyPatches(patches) } }
//	val tree = Mobile.renderInitial()
//	...
//	val patches = Mobile.triggerCallback(id) // event path, synchronous
//
// Event delivery contract: patches reach the native side on two paths — the
// synchronous return value of the Trigger* functions (the event path) and
// PatchListener pushes (the async path: timers, goroutines calling State.Set).
// Each render pass produces its diff exactly once and delivers it on exactly
// one of the two paths, so the native renderer applies everything it receives
// from either path, in arrival order, and stays consistent.
package mobile

import (
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// PatchListener is re-declared here (rather than aliased to
// render.PatchListener) so this package binds standalone; see that type for
// the threading contract — ApplyPatches arrives on a background goroutine and
// implementations must hop to their UI thread.
type PatchListener interface {
	ApplyPatches(patches string)
}

var mgr *render.Manager

// dataDir is the app's writable directory, provided by the native shell.
// Go code cannot discover this itself: on iOS and Android the writable
// sandbox path is an OS-level fact only the shell knows (Documents on iOS,
// Context.filesDir on Android), and there is no portable env var for it.
var dataDir string

// SetDataDir records the app's writable directory for Go-side persistence
// (see examples/todoapp's bytdb store). The shell must call it before
// RenderInitial so the first render can already read persisted state; the
// bound app package's init runs earlier still, which is why apps open their
// stores lazily on first render rather than in init. Left unset, DataDir
// returns "" and persistence-aware apps run in-memory — the right behavior
// for the web preview targets and for tests that don't care about storage.
func SetDataDir(path string) {
	dataDir = path
}

// DataDir returns the writable directory the shell registered, or "" if none.
func DataDir() string {
	return dataDir
}

// Register installs the app's root view and context. It must be called from
// Go (typically the bound app package's init) before the native shell invokes
// any other function here. Calling it again replaces the app — the previous
// manager's push pump is shut down so it cannot keep rendering the old tree.
func Register(ctx *core.Context, root func(*core.Context) core.View) {
	if mgr != nil {
		mgr.Close()
	}
	mgr = render.New(ctx, root)
}

// RenderInitial returns the full initial tree as JSON for the first mount.
func RenderInitial() string {
	return mgr.RenderInitial()
}

// RenderAgain re-renders and returns the diff against the last rendered tree.
// Native shells normally don't call this directly — the Trigger* functions
// already fold it into the event path — but it is the escape hatch for shells
// that drive rendering themselves.
func RenderAgain() string {
	return mgr.RenderAgain()
}

// SetListener attaches the native push target for async updates.
func SetListener(l PatchListener) {
	mgr.SetListener(listenerAdapter{l})
}

// TriggerCallback dispatches a void event (e.g. a button tap) by callback ID
// and returns the resulting patches. Dispatch goes through the manager so the
// handler and its follow-up render run under the render mutex — an event can
// never interleave with a push-pump render pass.
func TriggerCallback(id string) string {
	return mgr.DispatchCallback(id)
}

// TriggerTextCallback dispatches a string-carrying event (e.g. input change).
func TriggerTextCallback(id string, value string) string {
	return mgr.DispatchTextCallback(id, value)
}

// TriggerBoolCallback dispatches a bool-carrying event (e.g. checkbox toggle).
func TriggerBoolCallback(id string, value bool) string {
	return mgr.DispatchBoolCallback(id, value)
}

// TriggerIntCallback dispatches an int-carrying event (e.g. tab selection).
func TriggerIntCallback(id string, value int) string {
	return mgr.DispatchIntCallback(id, value)
}

// listenerAdapter bridges the locally declared interface onto the render
// package's without exposing that package to the bind tool.
type listenerAdapter struct{ l PatchListener }

func (a listenerAdapter) ApplyPatches(patches string) { a.l.ApplyPatches(patches) }
