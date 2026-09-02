package mobile

import (
	"encoding/json"
	"log"

	"github.com/rohanthewiz/grmob/core"
)

// SystemEventListener is the native shell's sink for app→host system events:
// the transient, non-tree things a platform draws or performs itself —
// core.ShowToast and core.OpenURL today.
//
// It mirrors PatchListener exactly, and for the same reason: gomobile cannot
// bind a function parameter, but it can bind a single-method interface, so an
// interface is how a callback crosses the FFI at all. Both halves of the
// payload are strings for the same reason — core's handler signature is
// (string, map[string]any), and a Go map is not a bindable type — so the data
// crosses as JSON text, the same envelope discipline patches and events
// already use.
//
// # Threading
//
// OnSystemEvent arrives synchronously on whichever goroutine called ShowToast
// or OpenURL. That may be the render goroutine (a tap handler), a timer, or
// any goroutine the app spawned — never, reliably, the platform's UI thread.
// Implementations must therefore hop to their own main thread before touching
// UI, exactly as PatchListener implementations do.
type SystemEventListener interface {
	// OnSystemEvent delivers one event. name is the event kind ("toast",
	// "open_url"); payload is its data as a JSON object.
	OnSystemEvent(name string, payload string)
}

// SetSystemEventListener installs the shell's sink. Passing nil detaches it,
// after which system events are dropped silently — the correct behavior for a
// headless run, and the state a shell that never calls this stays in.
//
// It exists because core.SetSystemEventHandler takes a func, which gobind
// cannot bind, so before this there was no way for a native shell to receive
// a toast at all: the events were emitted into a nil handler and vanished.
// (The WASM host had its own path and did not; see wasm/main.go.)
//
// Registering replaces any previous listener rather than fanning out. A
// process has one screen, so a second sink would mean one gesture producing
// two toasts.
func SetSystemEventListener(l SystemEventListener) {
	if l == nil {
		core.SetSystemEventHandler(nil)
		return
	}
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		payload, err := json.Marshal(data)
		if err != nil {
			// Dropped rather than delivered half-formed: the shell parses the
			// payload, and an unmarshalable one would only fail there, further
			// from the cause. A styled toast (*core.Style in the map) is the
			// only payload with any real marshalling surface.
			log.Printf("grmob/mobile: dropping system event %q: %v", name, err)
			return
		}
		l.OnSystemEvent(name, string(payload))
	})
}
