package core

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Debug mode: opt-in runtime checks for the failure modes that positional
// hooks make silent. Production apps mis-render mysteriously when a hook is
// called conditionally (cursor drift) or when sibling keys collide (keyed
// reconciliation quietly degrades to positional); with debug mode on, those
// conditions are detected during the render pass and recorded as concerns
// that tests can assert on and humans can dump.
//
// The flag is process-wide (an atomic.Bool, mirroring element's
// DebugSet/IsDebugMode) rather than per-context. Debug mode is a development
// switch flipped once at startup, not per-app state — and keeping it global
// lets hot paths guard with a single atomic load instead of threading a
// context into places that don't otherwise need one (renderAll, Cached).
// If two apps in one process ever need different settings, this can move
// onto Context like the callback registry did; the concerns collector below
// would move with it.
var debugMode atomic.Bool

// SetDebugMode turns the debug checks on or off. Zero overhead when off:
// every check site guards with IsDebugMode before doing any work.
func SetDebugMode(on bool) {
	debugMode.Store(on)
}

// IsDebugMode reports whether debug checks are active.
func IsDebugMode() bool {
	return debugMode.Load()
}

// Concern kinds. Each names one class of silent bug the debug checks detect.
const (
	// ConcernCursorDrift: a context's hook cursor ended a pass out of step
	// with its slot count or with the previous pass — some NewState /
	// UseChildContext call is conditional or loop-varying, so later slots
	// are (or will be) read by the wrong component.
	ConcernCursorDrift = "cursor-drift"

	// ConcernDuplicateKey: two siblings in one container carry the same
	// non-empty Key, defeating keyed reconciliation for that sibling list.
	ConcernDuplicateKey = "duplicate-key"

	// ConcernCachedHooks: a Cached view consumed hook slots during its
	// render. In production the view renders once and never again, so those
	// slots vanish on later passes and shift every component after it.
	ConcernCachedHooks = "cached-hooks"

	// ConcernCachedCallbacks: a Cached view registered event callbacks. In
	// production its handlers are purged after the first pass it skips, and
	// the un-consumed counter slots shift the callback IDs of everything
	// registered after it.
	ConcernCachedCallbacks = "cached-callbacks"

	// ConcernUnknownItem: a container (Row, Column, Card, Box, List) was
	// handed an argument that is neither a StyleProp, a BehaviorProp nor a
	// View. PropsAndChildren is an alias for any, so the compiler accepts
	// anything and containerNode drops what it cannot classify — the symptom
	// is a style or handler that simply never took effect. An untyped nil is
	// exempt: that is MaybeProp's false path, not a mistake.
	ConcernUnknownItem = "unknown-container-item"

	// ConcernRenderPanic: an ErrorBoundary caught a panic escaping a
	// component's Render and swapped in its fallback. The app kept running —
	// that is the boundary doing its job — which is exactly why this needs
	// reporting: a boundary placed high in the tree can hide a component that
	// has been dead for weeks behind a plausible-looking "unavailable" panel.
	// The detail carries the panic value; the full stack goes to the
	// fallback, not here.
	ConcernRenderPanic = "render-panic"

	// ConcernHandlerPanic: an event handler panicked and the render driver
	// recovered it. Distinct from ConcernRenderPanic because the failure is
	// in a different phase with a different blast radius: a render panic
	// costs a subtree's frame, while a handler panic abandons the handler
	// partway, so the app's state may be half-updated in a way no fallback
	// can describe.
	ConcernHandlerPanic = "handler-panic"
)

// Concern is one detected issue. Kind is one of the Concern* constants;
// Detail is human-readable specifics; Count is how many times this exact
// (Kind, Detail) pair fired — checks run every pass, so a persistent bug
// increments its count rather than flooding the collector with duplicates.
type Concern struct {
	Kind   string
	Detail string
	Count  int
}

// concernCollector deduplicates concerns by (Kind, Detail), the same idea as
// element's concerns map. Package-level for the same reason as the flag; the
// mutex is uncontended in practice (writes only happen in debug mode).
type concernCollector struct {
	mu    sync.Mutex
	items map[string]*Concern // key: Kind + "|" + Detail
}

var concerns = &concernCollector{items: make(map[string]*Concern)}

// upsertConcern records a finding, bumping the count if the identical
// finding was already recorded. Callers gate on IsDebugMode before building
// the detail string, so the off path pays nothing; the guard here is a
// backstop so a stray call can never grow the map with debug off.
func upsertConcern(kind, detail string) {
	if !IsDebugMode() {
		return
	}
	key := kind + "|" + detail
	concerns.mu.Lock()
	defer concerns.mu.Unlock()
	if c, ok := concerns.items[key]; ok {
		c.Count++
		return
	}
	concerns.items[key] = &Concern{Kind: kind, Detail: detail, Count: 1}
}

// ReportConcern records a concern from outside this package.
//
// The detection sites for most concern kinds live in core, so they call
// upsertConcern directly; the panic guards in the render driver do not, and a
// recovered panic is exactly the sort of silent-by-design event the concern
// list exists to surface. Callers should gate on IsDebugMode themselves —
// detail strings usually cost a Sprintf to build, and there is no reason to
// pay for one in a release build.
//
// Deduplicated on kind+detail like every other concern, so a failure that
// repeats every frame occupies one entry with a rising count.
func ReportConcern(kind, detail string) {
	upsertConcern(kind, detail)
}

// Concerns returns a snapshot of all recorded concerns, sorted by kind then
// detail so test assertions and dumps are deterministic.
func Concerns() []Concern {
	concerns.mu.Lock()
	out := make([]Concern, 0, len(concerns.items))
	for _, c := range concerns.items {
		out = append(out, *c)
	}
	concerns.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Detail < out[j].Detail
	})
	return out
}

// ClearConcerns drops all recorded concerns. Tests call it between cases;
// apps can call it after acting on a dump.
func ClearConcerns() {
	concerns.mu.Lock()
	defer concerns.mu.Unlock()
	clear(concerns.items)
}

// DumpConcerns renders the recorded concerns as a human-readable block, one
// line per finding. Empty string when there is nothing to report.
func DumpConcerns() string {
	list := Concerns()
	if len(list) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "grmob debug: %d concern(s)\n", len(list))
	for _, c := range list {
		fmt.Fprintf(&b, "  [%s] ×%d %s\n", c.Kind, c.Count, c.Detail)
	}
	return b.String()
}

// ---- Check 1: cursor drift ----

// EndRenderPass closes a render pass for debug purposes: in debug mode it
// walks the context tree and flags any context whose hook usage this pass is
// inconsistent. Hosts that drive passes through render.Manager get this for
// free (the manager calls it after each pass); hand-rolled pass loops should
// call it after rendering, paired with BeginRenderPass/Reset before.
//
// The pairing with the rest of the pass boundary:
//
//	BeginRenderPass()  — callback ID counters restart
//	Reset()            — hook cursors restart
//	root.Render(ctx)   — components consume slots, cursor advances
//	EndRenderPass()    — cursors audited against slots + previous pass  ← here
//
// A no-op (single atomic load) when debug mode is off.
func (ctx *Context) EndRenderPass() {
	if !IsDebugMode() {
		return
	}
	ctx.auditCursor()
}

// auditCursor applies the two drift detections to this context and recurses
// over every derived context, mirroring Reset's traversal (direct children,
// slot-held child contexts, named scopes).
//
// The two detections, and why each comparison is what it is:
//
//  1. Within-pass: 0 < Cursor < len(slots). Slots only grow (NewState appends
//     when the cursor runs past the end), so a cursor that stopped short of
//     the slot count means a hook that allocated those trailing slots on an
//     earlier pass did not run this pass — the classic conditional-hook bug.
//     Cursor == 0 is exempt: a context that rendered nothing this pass is
//     normal life for a navigated-away Scope, and skipping the whole context
//     shifts nothing because nothing after it read its slots.
//
//  2. Across passes: Cursor differs from the previous pass's ending cursor,
//     both non-zero. This catches the growth direction that check 1 cannot:
//     a conditional hook turning ON appends its slot at the end (cursor ==
//     len(slots), check 1 passes) but has still changed the context's hook
//     count — the moment the conditional flips back off, existing state is
//     misread. Requiring both cursors non-zero keeps navigated-away scopes
//     (which record 0-cursor passes) from comparing against a stale count.
func (ctx *Context) auditCursor() {
	cursor := ctx.Cursor

	// Snapshot slot length and the slot values under the lock, same
	// discipline as Reset: the slice header and elements may be written by a
	// concurrent State.Set. Child contexts are only ever appended, never
	// replaced, so recursing outside the lock on the copies is safe.
	ctx.lock.Lock()
	slotCount := len(ctx.slots)
	slots := make([]any, slotCount)
	copy(slots, ctx.slots)
	ctx.lock.Unlock()

	if cursor > 0 && cursor < slotCount {
		upsertConcern(ConcernCursorDrift, fmt.Sprintf(
			"context %p ended the pass at cursor %d with %d slots allocated: a hook (NewState/UseChildContext) that ran on an earlier pass was skipped this pass — hooks must be called unconditionally, in the same order, every render",
			ctx, cursor, slotCount))
	}
	if cursor > 0 && ctx.debugPassSeen && ctx.debugLastCursor > 0 && ctx.debugLastCursor != cursor {
		upsertConcern(ConcernCursorDrift, fmt.Sprintf(
			"context %p consumed %d hook slots this pass but %d last pass: hook count varies between renders — a hook is inside a conditional or a variable-length loop",
			ctx, cursor, ctx.debugLastCursor))
	}
	// Record for the next pass's across-pass comparison, but never overwrite
	// a real count with 0: a scope that sat out some passes should be judged
	// against its last *rendered* pass when it comes back.
	if cursor > 0 {
		ctx.debugLastCursor = cursor
		ctx.debugPassSeen = true
	}

	for _, child := range ctx.children {
		child.auditCursor()
	}
	for _, slot := range slots {
		if c, ok := slot.(*Context); ok {
			c.auditCursor()
		}
	}
	for _, scope := range ctx.scopes {
		scope.auditCursor()
	}
}

// ---- Check 2: duplicate sibling keys ----

// ---- Check 3 (sketch): provenance ----
//
// Not implemented — the element-lessons plan calls for a sketch only.
// The idea, from element's data-ele-id: in debug mode, annotate each node
// with the component that produced it, so a concern (or a rendered page) can
// say "this came from TodoRow" instead of "context 0xc000123". The cheap
// version: ComponentFunc.Render, under IsDebugMode, stamps the returned
// node's Props["debugSource"] with the component function's name via
// runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(); htmlout then
// surfaces it as a data-grmob-src attribute. Two reasons it is deferred:
// Props mutation must happen inside the render (before the Node freezes per
// the immutability contract), and anonymous ComponentFuncs — most of the
// framework — all report unhelpful names like "core.Row.func1", so the
// stamping is only useful once user-defined named components are common.

// checkDuplicateKeys flags non-empty Keys that repeat within one sibling
// list. Keyed reconciliation matches old and new children by Key; with a
// duplicate, one of the two matches wins arbitrarily and the other child is
// diffed against the wrong counterpart — state and row identity silently
// attach to the wrong data. Called from renderAll (the shared child-render
// path for containers) in debug mode only; containerType names the caller
// in the concern so the finding is locatable.
func checkDuplicateKeys(containerType string, nodes []*Node) {
	seen := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if n == nil || n.Key == "" {
			continue
		}
		if seen[n.Key] {
			upsertConcern(ConcernDuplicateKey, fmt.Sprintf(
				"%s has multiple children with key %q: sibling keys must be unique for keyed reconciliation to track identity",
				containerType, n.Key))
			continue
		}
		seen[n.Key] = true
	}
}
