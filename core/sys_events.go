package core

import "sync"

// System events are app→host notifications that are not part of the view
// tree: transient chrome the platform draws itself, of which the toast
// (core.ShowToast) is the first. They flow one way — nothing about them is
// reconciled, patched, or diffed — so they travel as a named payload handed
// to whatever host is driving the app, rather than as nodes.
//
// The handler is package-level, unlike the callback registry and navigation
// stack (which moved onto the context tree so two apps in one process stay
// independent). That is deliberate: ShowToast takes no Context by design —
// a toast is fire-and-forget and callable from any goroutine an app owns —
// so there is no tree to hang the handler on. And the resource a system
// event ultimately reaches is the physical screen's overlay layer, of which
// a process has exactly one; the host that owns it registers here once at
// startup (the WASM host forwards to the page, a native shell to its own
// toast view). Tests register a recorder and restore nil when done.
var (
	sysEventsMu     sync.RWMutex
	sysEventHandler func(name string, data map[string]any)
)

// SetSystemEventHandler installs the host's sink for system events. Passing
// nil detaches it, after which SendSystemEvent drops events silently — the
// right behavior for a headless run, where there is no screen to draw on.
func SetSystemEventHandler(fn func(name string, data map[string]any)) {
	sysEventsMu.Lock()
	defer sysEventsMu.Unlock()
	sysEventHandler = fn
}

// SendSystemEvent delivers one event to the host, synchronously on the
// caller's goroutine. The read is under RLock so senders never contend with
// each other, only with the (rare) handler swap.
func SendSystemEvent(name string, data map[string]any) {
	sysEventsMu.RLock()
	h := sysEventHandler
	sysEventsMu.RUnlock()
	if h != nil {
		h(name, data)
	}
}
