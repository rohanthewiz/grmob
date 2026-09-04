package tutorial

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// Deep links, both directions (deeplink.go). Host events are delivered the
// way wasm/main.go delivers them — core.ReceiveHostEvent with a decoded
// payload — and system events are caught by a recorder standing where the
// page stands, so these tests exercise the exact seam the browser host uses.

// recordRoutes installs a system-event recorder that keeps only the "route"
// events, restored on cleanup because the handler is process-wide.
func recordRoutes(t *testing.T) *[]string {
	t.Helper()
	var routes []string
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name == routeEvent {
			id, _ := data["lesson"].(string)
			routes = append(routes, id)
		}
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })
	return &routes
}

// newAppWithContext is newApp keeping hold of the root context, for the
// tests below that assert on the navigation stack's depth: the Manager does
// not expose its context, and core.StackDepth needs one.
func newAppWithContext(t *testing.T) (*render.Manager, *core.Context) {
	t.Helper()
	core.ClearConcerns()
	ctx := core.NewContext().WithTheme(core.DefaultTheme)
	mgr := render.New(ctx, App)
	t.Cleanup(mgr.Close)
	return mgr, ctx
}

// route delivers what the page sends at boot and on hashchange.
func route(id string) {
	core.ReceiveHostEvent(routeEvent, map[string]any{"lesson": id})
}

// --- Outbound: every door reports the lesson on screen ----------------------

func TestNavigationReportsRoutes(t *testing.T) {
	routes := recordRoutes(t)
	mgr := newApp(t)

	openLesson(t, mgr, flatLessons[0].Title) // contents row → Push
	tap(t, mgr, "Next ›")                    // → Replace
	tap(t, mgr, "‹ Prev")                    // → Replace
	tap(t, mgr, "‹ Contents")                // → Pop

	want := []string{flatLessons[0].ID, flatLessons[1].ID, flatLessons[0].ID, ""}
	if len(*routes) != len(want) {
		t.Fatalf("expected routes %v, got %v", want, *routes)
	}
	for i := range want {
		if (*routes)[i] != want[i] {
			t.Fatalf("expected routes %v, got %v", want, *routes)
		}
	}

	// Rendering reports nothing: the address bar only moves on navigation.
	tree(t, mgr)
	if len(*routes) != len(want) {
		t.Fatalf("a render pass must not report a route, got %v", *routes)
	}
	assertNoConcerns(t)
}

func TestFinishReportsContents(t *testing.T) {
	routes := recordRoutes(t)
	mgr := newApp(t)
	last := flatLessons[len(flatLessons)-1]

	openLesson(t, mgr, last.Title)
	tap(t, mgr, "Finish ✓")
	if n := len(*routes); n != 2 || (*routes)[0] != last.ID || (*routes)[1] != "" {
		t.Fatalf("expected [%s \"\"], got %v", last.ID, *routes)
	}
	assertNoConcerns(t)
}

// --- Inbound: a link opens its lesson --------------------------------------

func TestRouteOpensLessonFromContents(t *testing.T) {
	routes := recordRoutes(t)
	mgr, ctx := newAppWithContext(t)
	tree(t, mgr) // the first render is what subscribes the app

	route("2.3")
	cur := tree(t, mgr)
	if !hasTextContaining(cur, "2.3  ") {
		t.Fatal("route 2.3 should open lesson 2.3")
	}
	if core.StackDepth(ctx) != 2 {
		t.Fatalf("a link from the contents should Push, depth %d", core.StackDepth(ctx))
	}
	// Opening by link counts as opening: the contents row is badged.
	tap(t, mgr, "‹ Contents")
	if !hasText(tree(t, mgr), "opened") {
		t.Fatal("a lesson opened by link should be marked opened")
	}
	// The inbound hop never echoes back — the page's hash is already right.
	// Only the ‹ Contents tap above reported.
	if len(*routes) != 1 || (*routes)[0] != "" {
		t.Fatalf("a route host event must not report a route back, got %v", *routes)
	}
	assertNoConcerns(t)
}

func TestRouteBetweenLessonsReplaces(t *testing.T) {
	mgr, ctx := newAppWithContext(t)
	tree(t, mgr)

	route("1.1")
	route("3.2")
	if !hasTextContaining(tree(t, mgr), "3.2  ") {
		t.Fatal("route 3.2 should show lesson 3.2")
	}
	if d := core.StackDepth(ctx); d != 2 {
		t.Fatalf("a link from a lesson should Replace, keeping depth 2, got %d", d)
	}

	// "" is the contents.
	route("")
	if !hasText(tree(t, mgr), "GrMob Interactive Tutorial") {
		t.Fatal("an empty route should land on the contents")
	}
	if d := core.StackDepth(ctx); d != 1 {
		t.Fatalf("expected depth 1 on the contents, got %d", d)
	}
	assertNoConcerns(t)
}

func TestRouteIgnoresUnknownAndCurrent(t *testing.T) {
	mgr, ctx := newAppWithContext(t)
	tree(t, mgr)

	route("9.9")
	if !hasText(tree(t, mgr), "GrMob Interactive Tutorial") {
		t.Fatal("an unknown lesson must be ignored")
	}

	// A hashchange to the lesson already showing must not rebuild its
	// frame: the demo's state has to survive. Lesson 2.1's counter is the
	// probe.
	route("2.1")
	tap(t, mgr, "+1")
	tap(t, mgr, "+1")
	route("2.1")
	if !hasText(tree(t, mgr), "2") {
		t.Fatal("re-routing to the current lesson must keep its demo state")
	}
	if d := core.StackDepth(ctx); d != 2 {
		t.Fatalf("expected depth 2, got %d", d)
	}
	assertNoConcerns(t)
}

// A chapter link is its first lesson (deeplink.go, resolveRoute).
func TestChapterRouteOpensFirstLesson(t *testing.T) {
	mgr, ctx := newAppWithContext(t)
	tree(t, mgr)

	route("3")
	if !hasTextContaining(tree(t, mgr), "3.1  ") {
		t.Fatal("route 3 should open lesson 3.1")
	}
	if d := core.StackDepth(ctx); d != 2 {
		t.Fatalf("expected depth 2, got %d", d)
	}

	// The chapter spelling of the lesson already showing is a no-op: the
	// demo's state survives.
	toggleCheckbox(t, mgr, 0, true) // "Pause the count"
	route("3")
	if !hasTextContaining(tree(t, mgr), "Paused —") {
		t.Fatal("re-routing to the current chapter must keep the lesson's state")
	}

	// Past the end, and the malformed spellings Atoi would have accepted.
	for _, bad := range []string{"9", "0", "03", "+3", " 3", "3."} {
		route(bad)
		if !hasTextContaining(tree(t, mgr), "3.1  ") {
			t.Fatalf("route %q should be ignored", bad)
		}
	}
	assertNoConcerns(t)
}

// Subscriptions are process-wide; a closed app must let go of the channel
// so the next app (a browser re-mount, the next test) is the only listener.
func TestClosedAppStopsListening(t *testing.T) {
	first, firstCtx := newAppWithContext(t)
	tree(t, first)
	first.Close()

	second := newApp(t)
	tree(t, second)
	route("1.2")
	if d := core.StackDepth(firstCtx); d != 1 {
		t.Fatalf("a closed app must not navigate, depth %d", d)
	}
	if !hasTextContaining(tree(t, second), "1.2  ") {
		t.Fatal("the live app should have navigated")
	}
}
