//go:build js && wasm

// Package webhost is the browser host for a GrMob app, as a library: the
// GrMobWASM bindings, the render.Manager, the event and host-event bridges,
// the push channel and the hot-reload Shutdown hook, behind one call.
//
//	func main() { webhost.Run(nil, app.App) }
//
// # Why it exists
//
// The wiring was a file you copied. wasm/main.go carries it for the tutorial
// site and wasm/shots/host/main.go carries a second copy for the screenshot
// harness, and both of those live inside this module, where a dot-import can
// name an example. An app in its own module has no such option: it had to copy
// the file a third time, and the copy it made was of whichever version it
// happened to read — which is how an app ends up without Shutdown (so serve
// -dev falls back to page reloads) or without HostEvent (so lifecycle and audio
// status never arrive). A package is versioned with the runtime JS and the
// dev client it talks to; a copy is not.
//
// wasm/main.go and the shots host are left as they are rather than migrated
// onto this. Each is a program with its own documented reasons to differ (the
// shots host resolves its app from the query string; the tutorial host is what
// ./build.sh ships and what several prose checks quote), and
// webhost_test.go holds the three to exporting the same GrMobWASM surface, so
// the copies cannot drift from the library without a failing test.
//
// # The contract with the page
//
// Unchanged from docs/platforms/wasm.md: Go installs GrMobWASM.{RenderInitial,
// RenderAgain, ReceiveEvent, IsDirty, HostEvent, Shutdown}; the page may define
// GrMobApplyPatches (async push) and GrMobSystemEvent (toasts, audio, routes),
// and grmob-runtime.js defines both before the module is instantiated.
package webhost

import (
	"encoding/json"
	"log"
	"sync"
	"syscall/js"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// host is the state of one running module. A struct rather than the package
// globals wasm/main.go uses so that nothing about Run depends on being called
// once per process — though on js/wasm, where main returning exits the
// module, it is.
type host struct {
	ctx     *core.Context
	app     func(*core.Context) core.View
	manager *render.Manager

	done     chan struct{}
	doneOnce sync.Once
}

// Run installs the GrMobWASM bindings for app and blocks until the page calls
// GrMobWASM.Shutdown. Call it as the last line of main.
//
// ctx may be nil, which means a fresh context carrying core.DefaultTheme — the
// same root the tutorial mounts on. Pass one to start from another theme or
// with context values of your own.
//
// Returning from main is how a Go program exits on js/wasm, so Run returning
// is what lets serve -dev's client await the old module's exit before it
// instantiates the new one; see the Shutdown binding below.
func Run(ctx *core.Context, app func(*core.Context) core.View) {
	if ctx == nil {
		ctx = core.NewContext().WithTheme(core.DefaultTheme)
	}
	h := &host{ctx: ctx, app: app, done: make(chan struct{})}

	funcs := map[string]js.Func{
		"RenderInitial": js.FuncOf(h.renderInitial),
		"RenderAgain":   js.FuncOf(h.renderAgain),
		"ReceiveEvent":  js.FuncOf(h.receiveEvent),
		"IsDirty":       js.FuncOf(h.isDirty),
		"HostEvent":     js.FuncOf(h.hostEvent),
		"Shutdown":      js.FuncOf(h.shutdown),
	}
	global := map[string]any{}
	for name, fn := range funcs {
		global[name] = fn
	}
	js.Global().Set("GrMobWASM", global)
	h.registerSystemEvents()

	println("GrMob WASM ready.")
	<-h.done

	// Release the js.Funcs only after main is on its way out: a page calling
	// into a released Func gets a "call to released function" panic instead of
	// a quiet no-op, and nothing may call in once Shutdown has run.
	for _, fn := range funcs {
		fn.Release()
	}
	println("GrMob WASM stopped.")
}

// renderInitial mounts (or re-mounts) the app and returns the full tree JSON.
//
// A re-mount closes the previous manager first, which stops its pump and every
// hook resource (interval tickers, timeouts) registered on the shared context,
// so a second RenderInitial cannot leave the first tree's timers running.
func (h *host) renderInitial(this js.Value, args []js.Value) any {
	if h.manager != nil {
		h.manager.Close()
	}
	h.manager = render.New(h.ctx, h.app)
	// Push path: attached only when the page defines the sink. A page without
	// it polls IsDirty/RenderAgain instead, and the manager never consumes a
	// diff unless a listener is attached, so neither path loses one.
	if js.Global().Get("GrMobApplyPatches").Type() == js.TypeFunction {
		h.manager.SetListener(jsPatchListener{})
	}
	return js.ValueOf(h.manager.RenderInitial())
}

// jsPatchListener forwards pushed patches to the page. ApplyPatches runs on
// the pump goroutine, which js/wasm schedules cooperatively on the single JS
// thread, so calling into JS from it needs no marshalling.
type jsPatchListener struct{}

func (jsPatchListener) ApplyPatches(patches string) {
	js.Global().Call("GrMobApplyPatches", patches)
}

func (h *host) renderAgain(this js.Value, args []js.Value) any {
	if h.manager == nil {
		return js.ValueOf("[]") // nothing mounted yet: an empty patch list, not a nil deref
	}
	return js.ValueOf(h.manager.RenderAgain())
}

func (h *host) isDirty(this js.Value, args []js.Value) any {
	return js.ValueOf(h.ctx.IsDirty())
}

// receiveEvent delivers a user event: args are the callback ID and a JSON
// payload of the form {"value": ...}, whose value type selects the callback
// kind (void, text, bool, int) inside core.
//
// The dispatch is guarded because this host calls the context directly rather
// than through render.Manager's Dispatch* (which carry the same guard). A
// panic escaping a handler would unwind into the js.Func callback and abort
// the Go runtime, killing the app for the rest of the page's life; recovered,
// the handler's work is abandoned where it stopped and the next render shows
// whatever state actually exists.
func (h *host) receiveEvent(this js.Value, args []js.Value) any {
	if len(args) < 2 || args[0].Type() != js.TypeString || args[1].Type() != js.TypeString {
		log.Printf("grmob: dropping an event with %d arguments", len(args))
		return nil
	}
	id := args[0].String()
	var payload map[string]any
	if err := json.Unmarshal([]byte(args[1].String()), &payload); err != nil {
		log.Printf("grmob: dropping malformed payload for %s: %v", id, err)
		return nil
	}
	if rerr := core.Guard(func() {
		h.ctx.ReceiveEventPayload(map[string]any{"callback": id, "value": payload["value"]})
	}); rerr != nil {
		log.Printf("grmob: recovered panic in handler %s: %v\n%s", id, rerr.Value, rerr.Stack)
	}
	return nil
}

// hostEvent carries host→app traffic that answers no callback: lifecycle
// (visibilitychange), audio status ticks, permission answers, and any name an
// app and its page agree on. Consumers' state writes reach the screen through
// the push channel, so nothing is rendered here.
func (h *host) hostEvent(this js.Value, args []js.Value) any {
	if len(args) < 2 || args[0].Type() != js.TypeString || args[1].Type() != js.TypeString {
		log.Printf("grmob: dropping a host event with %d arguments", len(args))
		return nil
	}
	name := args[0].String()
	var payload map[string]any
	if err := json.Unmarshal([]byte(args[1].String()), &payload); err != nil {
		log.Printf("grmob: dropping malformed host event %q: %v", name, err)
		return nil
	}
	if rerr := core.Guard(func() { core.ReceiveHostEvent(name, payload) }); rerr != nil {
		log.Printf("grmob: recovered panic in host event %s: %v\n%s", name, rerr.Value, rerr.Stack)
	}
	return nil
}

// shutdown is the hot-reload hook. Order matters: closing the manager first
// stops every hook-owned timer, so the old module cannot keep calling
// GrMobApplyPatches with paths into a tree the new module has replaced; then
// releasing Run lets main return, which makes wasm_exec.js drop the instance's
// memory and settle go.run's promise — the signal serve/devclient.js awaits
// before booting the next build. Without the second step every reload would
// park another full Go heap in the tab.
func (h *host) shutdown(this js.Value, args []js.Value) any {
	if h.manager != nil {
		h.manager.Close()
		h.manager = nil
	}
	h.doneOnce.Do(func() { close(h.done) })
	return nil
}

// registerSystemEvents wires core's app→host channel (toasts, open_url, audio
// commands, permission requests) to the page, when the page has a sink for it.
// The payload crosses as JSON text because js.ValueOf cannot marshal nested Go
// maps or the *core.Style a styled toast carries.
func (h *host) registerSystemEvents() {
	if js.Global().Get("GrMobSystemEvent").Type() != js.TypeFunction {
		return
	}
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		payload, err := json.Marshal(data)
		if err != nil {
			log.Printf("grmob: dropping system event %q: %v", name, err)
			return
		}
		js.Global().Call("GrMobSystemEvent", name, string(payload))
	})
}
