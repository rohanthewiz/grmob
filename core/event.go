package core

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
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

	// scopes is the stack of ID namespaces open at this point of the pass:
	// scopes[0] is the pass's root, and each keyed subtree being rendered
	// pushes one more (openScope). Every registration draws its number from
	// the top scope's counters and its spelling from the top scope's path.
	// See "Identity-keyed IDs" on beginPass.
	scopes []*idScope

	// total counts registrations of every kind, in every scope, since
	// beginPass. registrationCount reads it; ErrorBoundary snapshots it.
	total int

	// trail records, in order, every ID registered and every child-key
	// occurrence counted since beginPass, so ErrorBoundary can undo exactly
	// what an abandoned subtree did (rollbackCounters). Reset, not freed, at
	// each beginPass, so a steady app reuses one backing array.
	trail []trailEntry

	// edits holds the text-edit ledger of each text field a native host
	// edits, by callback ID; see text_edit.go. Nil until the first
	// TriggerTextEdit, which is every web build and every test that never
	// dispatches an edit.
	edits map[string]*textEditLedger

	// sequenced is set by the first TriggerTextEdit: the host speaks the
	// text-edit protocol, so from then on every text field is stamped from
	// its first render rather than from its first edit. See "Every field,
	// once the host is sequenced" in text_edit.go.
	sequenced bool

	// used marks IDs touched (registered or triggered) since the last
	// beginPass; purge drops everything unmarked, so handlers for nodes that
	// vanished from the tree cannot fire from a stale native event.
	used map[string]bool
}

// idScope is one ID namespace: the root of a pass, or one keyed subtree.
//
// The counters are per kind, as they always were. backCounter numbers
// core.OnBack's handlers ("back_cb_N"): they are void callbacks and live in
// voidCBs, so every host dispatches them through the ordinary void path, but
// they take IDs from a sequence of their own. See registerBack for why.
type idScope struct {
	// path is "" for the root and "k1/k2/" below it: each enclosing key,
	// escaped (escapeIDKey) and followed by '/'. An ID is its kind's prefix,
	// the path, then the counter, so the counter is always the text after
	// the last '/' and no two scopes can spell the same ID.
	path string

	voidCounter int
	textCounter int
	boolCounter int
	intCounter  int
	backCounter int

	// seen counts the child scopes opened under this one this pass, by key.
	// A key is unique among siblings, not within a scope: two lists that are
	// both children of one unkeyed column, each keyed "0", "1", ..., open the
	// same keys in the same scope. The second "0" is spelled "0~1". Nil until
	// the first child scope, which most scopes never open.
	seen map[string]int
}

// trailEntry is one undoable step of a pass: an ID registered (id set) or a
// child key counted in a scope (scope set). See rollbackCounters.
type trailEntry struct {
	id    string
	scope *idScope
	key   string
}

func newCallbackRegistry() *callbackRegistry {
	return &callbackRegistry{
		voidCBs: make(map[string]func()),
		textCBs: make(map[string]func(string)),
		boolCBs: make(map[string]func(bool)),
		intCBs:  make(map[string]func(int)),
		used:    make(map[string]bool),
		// A root scope from the start, so a view rendered outside any pass
		// (a test calling Render directly) still has somewhere to register.
		scopes: []*idScope{{}},
	}
}

// beginPass resets the ID counters so IDs are assigned by render-pass
// sequence: the Nth callback registered in a scope is always "cb_N" with that
// scope's path (or "txt_cb_"/"bool_cb_"/"int_cb_"/"back_cb_" for its kind).
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
// # Identity-keyed IDs
//
// A sequence alone is positional: a subtree that registers one more callback
// than last pass shifts the ID of every callback after it, anywhere in the
// tree. That cost a patch per later node and, worse, let an event dispatched
// against the tree before the change land on a neighbour's handler.
// EditableGrid paid both: its editor registers three handlers where the cell
// it replaces registers one, so every later cell's onClick moved on each
// entry to and exit from EDIT (N-073).
//
// So a keyed node names its own callbacks. core.Keyed, and the root of each
// Navigator frame, open a scope while their subtree renders (Context.keyScope);
// the subtree numbers its callbacks from zero under the key's path, and the
// enclosing scope's counters do not move at all:
//
//	Column                         cb_0          (root scope)
//	├─ Keyed "r1" ─ Row
//	│   ├─ Keyed "c0" ─ cell       cb_r1/c0/0
//	│   └─ Keyed "c1" ─ cell       cb_r1/c1/0
//	├─ Keyed "r2" ─ Row
//	│   ├─ Keyed "e0" ─ editor     cb_r2/e0/0 … cb_r2/e0/2
//	│   └─ Keyed "c1" ─ cell       cb_r2/c1/0    (unmoved by the editor)
//	└─ Button "Save"               cb_1          (unmoved by every row)
//
// IDs at the root keep the spelling they always had, so an app with no keys
// sees no change. Within a scope, IDs are still positional, with the same
// granularity as the reconciler's positional paths: a change shifts its
// unkeyed later siblings and nothing past the nearest keyed ancestor.
//
// What remains of the stale-ID window is narrower in kind, not just in
// reach. An ID now belongs to a key path; a late event can land on another
// handler only inside the same keyed node, or at an unkeyed position the
// change shifted. A Navigator frame's IDs carry the frame's key, so an event
// from a screen that has been navigated away from cannot reach the screen
// that replaced it at all.
//
// Must be called exactly once at the start of each render pass, before any
// component builders run. Renderers do this via render.Manager, not directly.
func (r *callbackRegistry) beginPass() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// A fresh root rather than zeroed counters: a pass that panicked through
	// a keyed subtree unwinds its scopes with defers, but a fresh root makes
	// the pass boundary the one place the stack is known to be right.
	r.scopes = append(r.scopes[:0], &idScope{})
	r.total = 0
	r.trail = r.trail[:0]
	// Fresh liveness marks for this pass: only callbacks re-registered below
	// survive the post-render purge.
	r.used = make(map[string]bool)
}

// idKeyEscaper keeps a key from forging a scope boundary. '/' ends a path
// segment and '~' introduces an occurrence suffix, so a key holding either
// could otherwise spell another key path's ID; '%' is escaped so the escape
// itself is unambiguous. Navigator's frame keys ("nav:frame:3") and
// EditableGrid's ("c2") pass through unchanged.
var idKeyEscaper = strings.NewReplacer("%", "%25", "/", "%2F", "~", "%7E")

// openScope pushes the ID scope of a keyed child of the current scope. It is
// paired with closeScope by Context.keyScope, under a defer, so a panic
// inside the subtree cannot leave the stack deep.
func (r *callbackRegistry) openScope(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	parent := r.scopes[len(r.scopes)-1]
	if parent.seen == nil {
		parent.seen = make(map[string]int)
	}
	n := parent.seen[key]
	parent.seen[key] = n + 1
	r.trail = append(r.trail, trailEntry{scope: parent, key: key})

	seg := idKeyEscaper.Replace(key)
	if n > 0 {
		// A repeat of a key already opened in this scope. Its IDs are
		// positional among the repeats, which is no worse than before keyed
		// IDs and needs two keyed lists without a keyed container to arise.
		seg += "~" + strconv.Itoa(n)
	}
	r.scopes = append(r.scopes, &idScope{path: parent.path + seg + "/"})
}

func (r *callbackRegistry) closeScope() {
	r.mu.Lock()
	defer r.mu.Unlock()
	// The root is never popped: beginPass owns it.
	if len(r.scopes) > 1 {
		r.scopes = r.scopes[:len(r.scopes)-1]
	}
}

// nextIDLocked spells the next ID of one kind in the current scope and records it.
// counter points into the top scope; prefix is the kind's ("cb_", ...).
// Callers hold r.mu.
func (r *callbackRegistry) nextIDLocked(prefix string, counter func(*idScope) *int) string {
	s := r.scopes[len(r.scopes)-1]
	c := counter(s)
	id := prefix + s.path + strconv.Itoa(*c)
	*c++
	r.total++
	r.trail = append(r.trail, trailEntry{id: id})
	r.used[id] = true
	return id
}

func (r *callbackRegistry) registerVoid(fn func()) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextIDLocked("cb_", func(s *idScope) *int { return &s.voidCounter })
	r.voidCBs[id] = fn // overwrites last pass's closure at this position, keeping the freshest captures
	return id
}

// registerBack registers a system-back handler (core.OnBack) as a void
// callback with an ID from its own sequence, "back_cb_N".
//
// # Why back gets a namespace
//
// A back press is the one event a user reliably sends twice in quick
// succession, and the second press is dispatched with the ID the host still
// holds, which is the one from before the first press's patches landed. With
// back handlers numbered among all void callbacks, that ID is re-assigned by
// the pass the first press caused, and it usually lands on whatever the new
// screen registered at that position:
//
//	pass 1  lesson screen   cb_7 = Navigator's Pop (onBack)
//	press 1 → cb_7 → Pop → pass 2
//	pass 2  contents screen cb_7 = the eighth row's onClick
//	press 2 → cb_7 (stale, Compose has not recomposed yet) → opens a lesson
//
// With a sequence of their own, a stale back ID can only resolve to another
// back handler registered at the same position, or to nothing:
//
//	pass 2  contents screen no back handlers, back_cb_0 is purged
//	press 2 → back_cb_0 → unknown ID → silent no-op
//
// Keyed IDs (see beginPass) would close that example on their own, since the
// two screens' callbacks now carry different frame keys. The namespace stays
// for what keys do not cover: Navigator's own Pop is registered outside the
// frame's scope, on purpose, so that "back" meeting a different back handler
// is still a back, as when two quick presses on a three-deep stack pop twice.
func (r *callbackRegistry) registerBack(fn func()) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextIDLocked("back_cb_", func(s *idScope) *int { return &s.backCounter })
	r.voidCBs[id] = fn
	return id
}

func (r *callbackRegistry) registerText(fn func(string)) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextIDLocked("txt_cb_", func(s *idScope) *int { return &s.textCounter })
	r.textCBs[id] = fn
	return id
}

func (r *callbackRegistry) registerBool(fn func(bool)) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextIDLocked("bool_cb_", func(s *idScope) *int { return &s.boolCounter })
	r.boolCBs[id] = fn
	return id
}

func (r *callbackRegistry) registerInt(fn func(int)) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextIDLocked("int_cb_", func(s *idScope) *int { return &s.intCounter })
	r.intCBs[id] = fn
	return id
}

// registrationCount is the total callbacks registered so far in the current
// pass, across every kind (back handlers included) and every scope. The
// debug-mode Cached bypass samples it before and after rendering a cached
// subtree: any advance means the subtree registers callbacks, which the
// production cache would break (see ConcernCachedCallbacks).
func (r *callbackRegistry) registrationCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.total
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

// hasVoid reports whether a void callback ID is still registered. Its one
// caller is endReachedState.purge, which needs the registry's answer rather
// than a second liveness rule of its own.
func (r *callbackRegistry) hasVoid(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.voidCBs[id]
	return ok
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

	// A ledger lives exactly as long as its text callback, for the same
	// reason the debounce ledger in PurgeUnusedCallbacks does.
	r.purgeEditsLocked(newText)

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

func (ctx *Context) registerBackCallback(fn func()) string {
	return ctx.registry.registerBack(fn)
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

// keyScope renders a keyed subtree inside its own callback-ID scope, so the
// subtree's IDs are spelled under key and nothing it registers moves an ID
// outside it (see "Identity-keyed IDs" on callbackRegistry.beginPass). An
// empty key is no key, as it is on Node.Key, and opens nothing.
//
// The close is deferred: a panic inside the subtree is recovered by an
// ErrorBoundary above it, and the pass carries on in the enclosing scope.
func (ctx *Context) keyScope(key string, render func() *Node) *Node {
	if key == "" {
		return render()
	}
	ctx.registry.openScope(key)
	defer ctx.registry.closeScope()
	return render()
}

// BeginRenderPass starts a callback ID pass for this context tree; see
// callbackRegistry.beginPass for the stability contract.
func (ctx *Context) BeginRenderPass() {
	ctx.registry.beginPass()
}

// PurgeUnusedCallbacks drops handlers not re-registered in the current pass;
// see callbackRegistry.purge.
//
// OnEndReached's debounce ledger is trimmed in the same breath and against
// the registry's own survivors, so the two can never disagree about which
// lists are still on screen — a guard outliving its handler would silently
// suppress the first page fetch of whatever list next inherits the ID.
func (ctx *Context) PurgeUnusedCallbacks() {
	ctx.registry.purge()
	ctx.endReached.purge(ctx.registry.hasVoid)
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
			if f, ok := parsed["value"].(float64); ok {
				ctx.TriggerIntCallback(id, int(f))
				return
			}
		}

		// Fallback: treat as a plain string value.
		ctx.TriggerTextCallback(id, val)

	case bool:
		ctx.TriggerBoolCallback(id, val)

	case float64:
		// JSON has one number type, so every numeric event — tab selection
		// (onTabChange), a slider index, a stepper — arrives here as a
		// float64 no matter how Go declared the handler. Without this case a
		// numeric envelope fell through to the void branch below, missed the
		// int callback map entirely, and did nothing at all: the control was
		// simply dead, with no error on either side of the bridge.
		ctx.TriggerIntCallback(id, int(val))

	case int:
		// Not reachable from a JSON host, but ReceiveEventPayload is exported
		// and a Go-side caller (a test, an embedder driving events directly)
		// naturally writes an int.
		ctx.TriggerIntCallback(id, val)

	case nil:
		ctx.TriggerCallback(id)
	default:
		ctx.TriggerCallback(id)
	}
}

// ---- Counter snapshot / rollback (ErrorBoundary) ----

// counterSnapshot is the registry's position at one instant: the current
// scope's counters, the pass's registration total, and how much of the trail
// existed. It is only ever produced and consumed inside a single render pass,
// in the scope it was taken in — the counters restart at every beginPass, so
// a snapshot has no meaning across passes.
type counterSnapshot struct {
	scope   *idScope
	void    int
	text    int
	boolean int
	integer int
	back    int
	total   int
	trail   int
}

// snapshotCounters records where the next callback ID of each kind would be
// assigned. ErrorBoundary takes one before rendering a child it might have to
// abandon.
func (r *callbackRegistry) snapshotCounters() counterSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.scopes[len(r.scopes)-1]
	return counterSnapshot{
		scope:   s,
		void:    s.voidCounter,
		text:    s.textCounter,
		boolean: s.boolCounter,
		integer: s.intCounter,
		back:    s.backCounter,
		total:   r.total,
		trail:   len(r.trail),
	}
}

// rollbackCounters rewinds the counters to a snapshot and un-marks the IDs
// registered in between, undoing the registration side-effects of a render
// that panicked partway through.
//
// Both halves are needed and they fix different problems:
//
//   - Rewinding the counters keeps ID assignment positional. A panicking
//     subtree registers a number of handlers that depends on how far it got —
//     which can vary with data between passes — so without the rewind every
//     component rendered after the boundary would see its IDs shift whenever
//     the failure point moved, and taps would land on the wrong handlers.
//     After the rewind the boundary's footprint is just its fallback's. The
//     same goes for child keys counted: a key the abandoned subtree opened
//     would otherwise make the fallback's own use of it a "~1" repeat.
//
//   - Un-marking makes purge collect the abandoned handlers. purge keeps
//     every ID marked used since beginPass; the abandoned subtree marked its
//     own, and those nodes are not on screen, so leaving the marks would keep
//     dead handlers dispatchable for as long as the failure persists.
//
// The trail, not the counters, says what to undo: the subtree may have
// registered inside keyed scopes of its own, whose IDs no range of the
// current scope's counters describes. Those scopes were popped by their
// defers during the panic, so only the current scope's counters need
// rewinding.
//
// The entries in the four callback maps are deliberately left alone: purge
// removes exactly the unmarked ones at the end of the pass, and any ID in the
// rolled-back range that the fallback or a later sibling re-uses is
// overwritten and re-marked on registration, as it would be normally.
func (r *callbackRegistry) rollbackCounters(snap counterSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := len(r.trail) - 1; i >= snap.trail; i-- {
		e := r.trail[i]
		if e.scope != nil {
			e.scope.seen[e.key]--
			continue
		}
		delete(r.used, e.id)
	}
	r.trail = r.trail[:snap.trail]
	r.total = snap.total

	s := snap.scope
	s.voidCounter = snap.void
	s.textCounter = snap.text
	s.boolCounter = snap.boolean
	s.intCounter = snap.integer
	s.backCounter = snap.back
}
