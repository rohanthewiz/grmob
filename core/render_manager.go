package core

import (
	"sync"
)

// RenderManager is the app's "state changed" notification point: one
// registered handler (see OnStateChange), invoked whenever anything in the
// context tree calls RequestRender. One instance per NewContext root, shared
// by pointer with every derived context.
//
// It is keyed by string rather than holding a bare func because it once
// carried a second, parallel registration API — RegisterRender, which minted
// "render_N" ids, and SubscribeRender, which called it and threw the id away.
// Nothing ever triggered those ids: State.Set has always notified the
// hardcoded "default" key, so every SubscribeRender handler was unreachable
// while the map grew by one entry per call. Both are gone; OnStateChange is
// the registration side that actually completes the circuit.
type RenderManager struct {
	mu   sync.Mutex
	subs map[string]func()
}

func NewRenderManager() *RenderManager {
	return &RenderManager{
		subs: make(map[string]func()),
	}
}

// TriggerRender invokes the handler registered under id, if any.
func (r *RenderManager) TriggerRender(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if fn, ok := r.subs[id]; ok {
		go fn() // async
	}
}

// defaultRenderTarget is the subscription key state mutations notify; see
// RenderManager for the history behind it being a key at all.
const defaultRenderTarget = "default"

// OnStateChange registers fn to run whenever state anywhere in this context
// tree is written (State.Set, or anything else calling RequestRender). Only
// one handler is held: a render driver like render.Manager owns re-rendering
// for the whole app, so later registrations replace earlier ones rather than
// fanning out duplicate render passes.
//
// fn is invoked on a fresh goroutine per notification (see TriggerRender), so
// it must be safe to call concurrently and should be cheap — the intended
// pattern is a non-blocking nudge into a coalescing channel, not a render.
func (ctx *Context) OnStateChange(fn func()) {
	ctx.renderManager.mu.Lock()
	defer ctx.renderManager.mu.Unlock()
	ctx.renderManager.subs[defaultRenderTarget] = fn
}

// RequestRender marks the tree dirty and notifies the registered render
// driver. This is the one entry point for "state changed, the UI should
// re-render" — used by State.Set and by async sources such as timers, so
// changes that happen outside a native event (where no bridge call is pending
// a response) can still reach the screen via the push channel.
func (ctx *Context) RequestRender() {
	ctx.MarkDirty()
	ctx.renderManager.TriggerRender(defaultRenderTarget)
}
