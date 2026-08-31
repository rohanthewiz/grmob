package core

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"
)

// callbackRegistry holds every event handler registered during render, keyed
// by the string IDs the native side dispatches with. One registry exists per
// context tree (created in NewContext, shared by every derived context), so
// two apps in one process — or two Managers in one test binary — cannot see
// or purge each other's handlers. This replaces the former package-level maps,
// which were exactly that kind of cross-app shared state.
//
// Four maps rather than one map[string]any: each callback kind has its own
// value signature, and separate maps keep dispatch type-safe without
// assertions at trigger time. IDs are namespaced per kind ("cb_N",
// "txt_cb_N", ...) so the counters are independent too.
type callbackRegistry struct {
	mu sync.Mutex

	voidCBs map[string]func()
	textCBs map[string]func(string)
	boolCBs map[string]func(bool)
	intCBs  map[string]func(int)

	voidCounter int
	textCounter int
	boolCounter int
	intCounter  int

	// used marks IDs touched (registered or triggered) since the last
	// beginPass; purge drops everything unmarked, so handlers for nodes that
	// vanished from the tree cannot fire from a stale native event.
	used map[string]bool
}

func newCallbackRegistry() *callbackRegistry {
	return &callbackRegistry{
		voidCBs: make(map[string]func()),
		textCBs: make(map[string]func(string)),
		boolCBs: make(map[string]func(bool)),
		intCBs:  make(map[string]func(int)),
		used:    make(map[string]bool),
	}
}

// beginPass resets the ID counters so IDs are assigned by render-pass
// sequence: the Nth callback registered in a pass is always "cb_N" (or
// "txt_cb_N"/"bool_cb_N"/"int_cb_N" for its kind).
//
// This is what makes callback IDs stable across renders. Component trees are
// rebuilt from scratch on every render, and with monotonically increasing
// counters every button received a brand-new onClick ID each time — so the
// reconciler saw every interactive node's props as changed on every render,
// and renderers re-bound every listener. With per-pass sequence IDs, an
// unchanged UI re-registers the same IDs in the same order and produces zero
// prop diffs; registration simply overwrites the map entry with the latest
// closure, which is required for correctness anyway (the new closure captures
// the current state slots).
//
// The IDs have the same stability granularity as the reconciler's positional
// TargetID paths: a structural change that shifts later siblings also shifts
// their callback IDs, and the same nodes get update-props patches the
// positional differ would emit regardless. An event dispatched against a
// stale tree can therefore hit a re-used ID and run the wrong handler in the
// brief window around a structural re-render; identity-keyed IDs (planned
// with stable node identity) are the eventual fix.
//
// Must be called exactly once at the start of each render pass, before any
// component builders run. Renderers do this via render.Manager, not directly.
func (r *callbackRegistry) beginPass() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.voidCounter = 0
	r.textCounter = 0
	r.boolCounter = 0
	r.intCounter = 0
	// Fresh liveness marks for this pass: only callbacks re-registered below
	// survive the post-render purge.
	r.used = make(map[string]bool)
}

func (r *callbackRegistry) registerVoid(fn func()) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := "cb_" + strconv.Itoa(r.voidCounter)
	r.voidCounter++
	r.voidCBs[id] = fn // overwrites last pass's closure at this position, keeping the freshest captures
	r.used[id] = true
	return id
}

func (r *callbackRegistry) registerText(fn func(string)) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := "txt_cb_" + strconv.Itoa(r.textCounter)
	r.textCounter++
	r.textCBs[id] = fn
	r.used[id] = true
	return id
}

func (r *callbackRegistry) registerBool(fn func(bool)) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := "bool_cb_" + strconv.Itoa(r.boolCounter)
	r.boolCounter++
	r.boolCBs[id] = fn
	r.used[id] = true
	return id
}

func (r *callbackRegistry) registerInt(fn func(int)) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := "int_cb_" + strconv.Itoa(r.intCounter)
	r.intCounter++
	r.intCBs[id] = fn
	r.used[id] = true
	return id
}

// registrationCount is the total callbacks registered so far in the current
// pass, across all four kinds. The debug-mode Cached bypass samples it before
// and after rendering a cached subtree: any advance means the subtree
// registers callbacks, which the production cache would break (see
// ConcernCachedCallbacks).
func (r *callbackRegistry) registrationCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.voidCounter + r.textCounter + r.boolCounter + r.intCounter
}

// lookupVoid (and the sibling lookups below) fetch the handler and mark the
// ID live under the lock, but return the function for the caller to invoke
// OUTSIDE the lock. Handlers are app code: they may run for a while, and they
// may legitimately dispatch another callback (a handler programmatically
// "clicking" something). Invoking under the registry lock would serialize
// unrelated registrations behind app code and would deadlock on any nested
// dispatch — the old package-global implementation had exactly that trap.
func (r *callbackRegistry) lookupVoid(id string) (func(), bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn, ok := r.voidCBs[id]
	if ok {
		r.used[id] = true
	}
	return fn, ok
}

func (r *callbackRegistry) lookupText(id string) (func(string), bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn, ok := r.textCBs[id]
	if ok {
		r.used[id] = true
	}
	return fn, ok
}

func (r *callbackRegistry) lookupBool(id string) (func(bool), bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn, ok := r.boolCBs[id]
	if ok {
		r.used[id] = true
	}
	return fn, ok
}

func (r *callbackRegistry) lookupInt(id string) (func(int), bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn, ok := r.intCBs[id]
	if ok {
		r.used[id] = true
	}
	return fn, ok
}

// purge drops every callback not marked live since the last beginPass. Called
// after each diff so IDs the pass did not re-register — nodes that left the
// tree — become silent no-ops instead of firing handlers for dead UI.
func (r *callbackRegistry) purge() {
	r.mu.Lock()
	defer r.mu.Unlock()

	newVoid := make(map[string]func())
	newText := make(map[string]func(string))
	newBool := make(map[string]func(bool))
	newInt := make(map[string]func(int))

	for id, fn := range r.voidCBs {
		if r.used[id] {
			newVoid[id] = fn
		}
	}
	for id, fn := range r.textCBs {
		if r.used[id] {
			newText[id] = fn
		}
	}
	for id, fn := range r.boolCBs {
		if r.used[id] {
			newBool[id] = fn
		}
	}
	for id, fn := range r.intCBs {
		if r.used[id] {
			newInt[id] = fn
		}
	}

	r.voidCBs = newVoid
	r.textCBs = newText
	r.boolCBs = newBool
	r.intCBs = newInt
	r.used = make(map[string]bool)
}

// ---- Context-facing surface ----
//
// Registration is internal (component builders inside this package call it
// with the ctx they already receive); dispatch and the render-pass hooks are
// exported methods because they are called from other packages — render's
// Manager for the pass boundary, and hosts without a Manager event path (the
// WASM runtime, tests) for dispatch. Native mobile shells should dispatch via
// render.Manager's Dispatch* methods instead, which serialize the handler
// with render passes under the manager's render mutex.

func (ctx *Context) registerCallback(fn func()) string {
	return ctx.registry.registerVoid(fn)
}

func (ctx *Context) registerTextCallback(fn func(string)) string {
	return ctx.registry.registerText(fn)
}

func (ctx *Context) registerBoolCallback(fn func(bool)) string {
	return ctx.registry.registerBool(fn)
}

func (ctx *Context) registerIntCallback(fn func(int)) string {
	return ctx.registry.registerInt(fn)
}

// BeginRenderPass starts a callback ID pass for this context tree; see
// callbackRegistry.beginPass for the stability contract.
func (ctx *Context) BeginRenderPass() {
	ctx.registry.beginPass()
}

// PurgeUnusedCallbacks drops handlers not re-registered in the current pass;
// see callbackRegistry.purge.
func (ctx *Context) PurgeUnusedCallbacks() {
	ctx.registry.purge()
}

// TriggerCallback dispatches a void event (e.g. a button tap) by callback ID.
// Unknown IDs are silent no-ops: a late native event racing a purge is
// expected traffic, not an error.
func (ctx *Context) TriggerCallback(id string) {
	if fn, ok := ctx.registry.lookupVoid(id); ok {
		fn()
	}
}

// TriggerTextCallback dispatches a string-carrying event (e.g. input change).
func (ctx *Context) TriggerTextCallback(id string, val string) {
	if fn, ok := ctx.registry.lookupText(id); ok {
		fn(val)
	}
}

// TriggerBoolCallback dispatches a bool-carrying event (e.g. a toggle).
func (ctx *Context) TriggerBoolCallback(id string, val bool) {
	if fn, ok := ctx.registry.lookupBool(id); ok {
		fn(val)
	}
}

// TriggerIntCallback dispatches an int-carrying event (e.g. tab selection).
func (ctx *Context) TriggerIntCallback(id string, val int) {
	if fn, ok := ctx.registry.lookupInt(id); ok {
		fn(val)
	}
}

// ReceiveEventPayload dispatches a loosely typed event envelope
// ({"callback": id, "value": ...}) by sniffing the value's type — the shape
// the WASM host sends. Typed hosts should call the Trigger* methods directly.
func (ctx *Context) ReceiveEventPayload(payload map[string]any) {
	id, ok := payload["callback"].(string)
	if !ok {
		log.Println("event payload has no callback ID")
		return
	}

	switch val := payload["value"].(type) {
	case string:
		// The value may itself be a JSON envelope ({"value": ...}) carrying
		// the real payload; unwrap it if so.
		var parsed map[string]any
		if err := json.Unmarshal([]byte(val), &parsed); err == nil {
			if v, ok := parsed["value"].(string); ok {
				ctx.TriggerTextCallback(id, v)
				return
			}
			if b, ok := parsed["value"].(bool); ok {
				ctx.TriggerBoolCallback(id, b)
				return
			}
		}

		// Fallback: treat as a plain string value.
		ctx.TriggerTextCallback(id, val)

	case bool:
		ctx.TriggerBoolCallback(id, val)
	case nil:
		ctx.TriggerCallback(id)
	default:
		ctx.TriggerCallback(id)
	}
}
