package core

import "testing"

// Each test builds its own context: the callback registry is per-context-tree
// state, so tests are isolated by construction (which is itself one of the
// behaviors under test — see TestContextsHaveIsolatedRegistries).

func TestCallbackIDsStableAcrossRenderPasses(t *testing.T) {
	ctx := NewContext()

	ctx.BeginRenderPass()
	first := ctx.registerCallback(func() {})
	second := ctx.registerTextCallback(func(string) {})
	third := ctx.registerBoolCallback(func(bool) {})

	// Simulate the next render of the same UI: same registration order must
	// reproduce the same IDs, otherwise every interactive node's props would
	// differ between renders and the reconciler would patch all of them.
	ctx.BeginRenderPass()
	if got := ctx.registerCallback(func() {}); got != first {
		t.Errorf("plain callback ID changed across passes: %q then %q", first, got)
	}
	if got := ctx.registerTextCallback(func(string) {}); got != second {
		t.Errorf("text callback ID changed across passes: %q then %q", second, got)
	}
	if got := ctx.registerBoolCallback(func(bool) {}); got != third {
		t.Errorf("bool callback ID changed across passes: %q then %q", third, got)
	}
}

func TestCallbackKindsUseDistinctIDSpaces(t *testing.T) {
	// Each kind counts independently; IDs must never collide across the
	// registries since the Trigger* methods look up in separate maps keyed by
	// these strings.
	ctx := NewContext()
	ctx.BeginRenderPass()
	ids := map[string]bool{
		ctx.registerCallback(func() {}):           true,
		ctx.registerTextCallback(func(string) {}): true,
		ctx.registerBoolCallback(func(bool) {}):   true,
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 distinct IDs across kinds, got %v", ids)
	}
}

func TestReRegistrationRunsLatestClosure(t *testing.T) {
	// Correctness requirement behind the overwrite semantics: the closure from
	// the most recent render captures the current state slots, so a stable ID
	// must always dispatch to the newest registration.
	var ran string
	ctx := NewContext()

	ctx.BeginRenderPass()
	id := ctx.registerCallback(func() { ran = "stale" })

	ctx.BeginRenderPass()
	if id2 := ctx.registerCallback(func() { ran = "fresh" }); id2 != id {
		t.Fatalf("expected re-registration at same position to reuse %q, got %q", id, id2)
	}

	ctx.TriggerCallback(id)
	if ran != "fresh" {
		t.Errorf("triggered closure = %q, want the latest registration", ran)
	}
}

func TestPurgeDropsCallbacksNotReRegistered(t *testing.T) {
	// Pass 1 renders two buttons; pass 2 renders one. After the purge, the
	// orphaned tail ID must be gone so a late event against it is a no-op
	// instead of firing a handler for a node that no longer exists.
	staleRan := false
	ctx := NewContext()

	ctx.BeginRenderPass()
	keep := ctx.registerCallback(func() {})
	stale := ctx.registerCallback(func() { staleRan = true })

	ctx.BeginRenderPass()
	ctx.registerCallback(func() {})
	ctx.PurgeUnusedCallbacks()

	ctx.registry.mu.Lock()
	_, staleExists := ctx.registry.voidCBs[stale]
	_, keepExists := ctx.registry.voidCBs[keep]
	ctx.registry.mu.Unlock()

	if staleExists {
		t.Errorf("callback %q should have been purged after not re-registering", stale)
	}
	if !keepExists {
		t.Errorf("callback %q was re-registered this pass and must survive the purge", keep)
	}

	ctx.TriggerCallback(stale) // must be a silent no-op
	if staleRan {
		t.Errorf("purged callback still executed")
	}
}

func TestIntCallbackIDsStableAndPurged(t *testing.T) {
	// Int callbacks (TabView's onTabChange) originally lived in a separate
	// registry with a never-resetting counter, so their IDs churned every
	// render and stale entries accumulated forever. This locks in their
	// membership in the render-pass system alongside the other three kinds.
	ctx := NewContext()
	ctx.BeginRenderPass()
	first := ctx.registerIntCallback(func(int) {})

	ctx.BeginRenderPass()
	got := 0
	if id := ctx.registerIntCallback(func(v int) { got = v }); id != first {
		t.Errorf("int callback ID changed across passes: %q then %q", first, id)
	}

	ctx.TriggerIntCallback(first, 3)
	if got != 3 {
		t.Errorf("TriggerIntCallback delivered %d, want 3", got)
	}

	// A pass that does not re-register the callback must purge it.
	ctx.BeginRenderPass()
	ctx.PurgeUnusedCallbacks()
	ctx.TriggerIntCallback(first, 9) // must be a silent no-op
	if got != 3 {
		t.Errorf("purged int callback still executed (got %d)", got)
	}
}

func TestContextsHaveIsolatedRegistries(t *testing.T) {
	// The consolidation guarantee: two apps (two NewContext roots) in one
	// process share nothing. Identical registration order yields identical ID
	// strings in both — and each ID must dispatch only its own app's handler.
	ctxA, ctxB := NewContext(), NewContext()

	var ranA, ranB bool
	ctxA.BeginRenderPass()
	idA := ctxA.registerCallback(func() { ranA = true })
	ctxB.BeginRenderPass()
	idB := ctxB.registerCallback(func() { ranB = true })

	if idA != idB {
		t.Fatalf("independent apps should mint the same pass-sequenced IDs, got %q and %q", idA, idB)
	}

	ctxA.TriggerCallback(idA)
	if !ranA || ranB {
		t.Errorf("dispatch crossed context boundaries: ranA=%v ranB=%v", ranA, ranB)
	}

	// One app's purge must not evict the other's live handlers.
	ctxA.BeginRenderPass()
	ctxA.PurgeUnusedCallbacks()
	ctxB.TriggerCallback(idB)
	if !ranB {
		t.Errorf("context A's purge removed context B's callback")
	}
}

func TestDerivedContextsShareOneRegistry(t *testing.T) {
	// Child and With* contexts must all write into the root's registry: a
	// handler registered while rendering inside a scoped/themed subtree is
	// dispatched by ID at the app level, with no knowledge of which subtree
	// registered it.
	root := NewContext()
	root.BeginRenderPass()

	ran := ""
	id1 := root.Scope("tab-0").registerCallback(func() { ran = "scoped" })
	id2 := root.WithTheme(DefaultTheme).registerCallback(func() { ran = "themed" })

	root.TriggerCallback(id1)
	if ran != "scoped" {
		t.Errorf("scoped child's callback not reachable from root, ran=%q", ran)
	}
	root.TriggerCallback(id2)
	if ran != "themed" {
		t.Errorf("WithTheme copy's callback not reachable from root, ran=%q", ran)
	}
}

func TestNestedDispatchDoesNotDeadlock(t *testing.T) {
	// A handler that itself dispatches another callback (programmatic
	// "click"). The registry must invoke handlers outside its lock or this
	// deadlocks — the old package-global implementation did.
	ctx := NewContext()
	ctx.BeginRenderPass()

	innerRan := false
	inner := ctx.registerCallback(func() { innerRan = true })
	outer := ctx.registerCallback(func() { ctx.TriggerCallback(inner) })

	ctx.TriggerCallback(outer)
	if !innerRan {
		t.Errorf("nested dispatch did not run the inner handler")
	}
}

func TestNavigationStackIsPerContext(t *testing.T) {
	// The navigator stack moved off a package global for the same isolation
	// reason as the registry: pushing a route in one app must not navigate
	// another. Screens render distinct text so the rendered tree identifies
	// which route is on top.
	screen := func(name string) func(*Context) View {
		return func(ctx *Context) View { return Text(name) }
	}

	ctxA, ctxB := NewContext(), NewContext()
	appA := Navigator(screen("home-A"))
	appB := Navigator(screen("home-B"))

	// Mount both apps, then navigate only app A. Navigator renders the top
	// route directly (no wrapper node), so the returned node IS the screen's
	// Text node.
	Render(ctxA, appA)
	Render(ctxB, appB)
	Push(ctxA, screen("details-A"))

	if got := Render(ctxA, appA).Props["content"]; got != "details-A" {
		t.Errorf("app A should show its pushed route, got %v", got)
	}
	if got := Render(ctxB, appB).Props["content"]; got != "home-B" {
		t.Errorf("app B was navigated by app A's Push, got %v", got)
	}

	Pop(ctxA)
	if got := Render(ctxA, appA).Props["content"]; got != "home-A" {
		t.Errorf("app A should be back at its root after Pop, got %v", got)
	}
}

// TestReceiveEventPayloadRoutesEveryValueKind pins the envelope dispatcher's
// type sniff against all four callback kinds.
//
// The float64 row is the one that was broken: JSON has a single number type,
// so a numeric event (onTabChange, a slider index) always arrives as float64.
// With no float64 case it fell through to the void branch, looked the ID up
// in the *void* map where an "int_cb_N" never exists, and did nothing — the
// control was dead with no error on either side of the bridge.
func TestReceiveEventPayloadRoutesEveryValueKind(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	var (
		voidRan bool
		gotText string
		gotBool bool
		gotInt  = -1
	)
	voidID := ctx.registerCallback(func() { voidRan = true })
	textID := ctx.registerTextCallback(func(s string) { gotText = s })
	boolID := ctx.registerBoolCallback(func(b bool) { gotBool = b })
	intID := ctx.registerIntCallback(func(i int) { gotInt = i })

	ctx.ReceiveEventPayload(map[string]any{"callback": voidID})
	ctx.ReceiveEventPayload(map[string]any{"callback": textID, "value": "typed"})
	ctx.ReceiveEventPayload(map[string]any{"callback": boolID, "value": true})
	// A JSON-decoded number, which is what the host actually sends.
	ctx.ReceiveEventPayload(map[string]any{"callback": intID, "value": float64(2)})

	if !voidRan {
		t.Error("a value-less envelope did not reach the void callback")
	}
	if gotText != "typed" {
		t.Errorf("text callback got %q, want \"typed\"", gotText)
	}
	if !gotBool {
		t.Error("bool callback did not receive true")
	}
	if gotInt != 2 {
		t.Errorf("int callback got %d, want 2 — a numeric envelope was not routed", gotInt)
	}
}

// TestReceiveEventPayloadAcceptsAGoInt covers the exported-API case:
// ReceiveEventPayload is public, and a Go caller driving events by hand
// writes a plain int rather than a JSON float.
func TestReceiveEventPayloadAcceptsAGoInt(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	got := -1
	id := ctx.registerIntCallback(func(i int) { got = i })
	ctx.ReceiveEventPayload(map[string]any{"callback": id, "value": 3})

	if got != 3 {
		t.Errorf("int callback got %d, want 3", got)
	}
}
