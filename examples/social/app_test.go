package social

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// TestMain turns on debug mode for the whole package. Debug mode is
// process-wide, so one switch covers every test here, and every render pass
// driven below is audited for cursor drift and duplicate keys. The tests then
// assert the audit found nothing — see assertNoConcerns.
func TestMain(m *testing.M) {
	core.SetDebugMode(true)
	m.Run()
}

// node mirrors the JSON shape render.Manager emits, which is all a test needs
// to locate widgets and read their callback IDs.
type node struct {
	Type     string
	Props    map[string]any
	Children []*node
}

func findNode(n *node, pred func(*node) bool) *node {
	if n == nil {
		return nil
	}
	if pred(n) {
		return n
	}
	for _, c := range n.Children {
		if found := findNode(c, pred); found != nil {
			return found
		}
	}
	return nil
}

func buttonLabeled(label string) func(*node) bool {
	return func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == label
	}
}

// tree renders the current state and parses the full tree. Callback IDs are
// per-pass sequence numbers, so every dispatch reads its ID from a freshly
// rendered tree rather than reusing one from an earlier pass — the same
// discipline the native shells follow.
func tree(t *testing.T, mgr *render.Manager) *node {
	t.Helper()
	var root node
	if err := json.Unmarshal([]byte(mgr.RenderInitial()), &root); err != nil {
		t.Fatalf("tree is not valid JSON: %v", err)
	}
	return &root
}

// tap locates a button by its label in the live tree and dispatches its
// onClick, returning the resulting patch JSON.
func tap(t *testing.T, mgr *render.Manager, label string) string {
	t.Helper()
	n := findNode(tree(t, mgr), buttonLabeled(label))
	if n == nil {
		t.Fatalf("no Button labeled %q in the current tree", label)
	}
	id, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatalf("Button %q has no onClick callback: %#v", label, n.Props)
	}
	return mgr.DispatchCallback(id)
}

func assertNoConcerns(t *testing.T) {
	t.Helper()
	if cs := core.Concerns(); len(cs) != 0 {
		t.Fatalf("debug concerns raised:\n%s", core.DumpConcerns())
	}
}

// TestNavigationKeepsRouteAndTabStateSeparate is the executable form of the
// comment on DetailsPage: the pushed route's counter and the shell's tab state
// must not alias the same positional slot, which they would if both routes
// rendered into one context — the counter would read the tab's string, and the
// popped shell would find an int where its tab name was.
//
// The isolation is Navigator's per-frame scope, so removing it fails this test
// in both of those visible ways before the concerns assertion even runs.
//
// The pop half also pins the other side of the contract: a frame that is still
// on the stack keeps its state. The root frame is never popped, so the shell
// comes back exactly as it was left.
func TestNavigationKeepsRouteAndTabStateSeparate(t *testing.T) {
	// The concern collector is process-wide too, so clear it up front rather
	// than inheriting a finding from another test in the package.
	core.ClearConcerns()

	mgr := render.New(core.NewContext().WithTheme(core.DefaultTheme), App)
	defer mgr.Close()

	initial := mgr.RenderInitial()
	if !strings.Contains(initial, "Página Inicial") {
		t.Fatalf("initial route is not the home tab:\n%s", initial)
	}

	// Push the details route: it replaces the whole tabbed shell, tab bar
	// included, because Navigator renders only the top of the stack.
	pushed := tap(t, mgr, "Abrir Detalhes")
	if !strings.Contains(pushed, "Contador: 0") {
		t.Fatalf("pushed route missing its scoped counter:\n%s", pushed)
	}

	// Two increments, so the assertion after the pop distinguishes "state
	// survived" from "state was re-initialized to its first-increment value".
	tap(t, mgr, "➕ Incrementar")
	if after := tap(t, mgr, "➕ Incrementar"); !strings.Contains(after, "Contador: 2") {
		t.Fatalf("counter did not reach 2:\n%s", after)
	}

	// Pop: the shell must come back intact. A tab state clobbered by an
	// aliased slot would leave Match with no matching case — an empty screen.
	popped := tap(t, mgr, "⬅️ Voltar")
	if !strings.Contains(popped, "Página Inicial") {
		t.Fatalf("popping did not restore the home tab:\n%s", popped)
	}

	// The tab bar still works after the round trip, which is the other half of
	// "the routes never touched each other's slots".
	searched := tap(t, mgr, "🔍")
	if !strings.Contains(searched, "Pesquisa") {
		t.Fatalf("tab state broken after the push/pop round trip:\n%s", searched)
	}

	assertNoConcerns(t)
}

// TestPushedRouteStateIsDiscardedOnPop is the counterpart to the test above:
// state belonging to a frame that LEFT the stack must be gone, not waiting.
//
// The same route function is pushed twice with a Pop in between. Each push
// mints a new frame id and therefore a new scope, so the counter starts at 0
// the second time. Before per-frame scopes this test asserted the opposite —
// DetailsPage's ctx.Scope("details") hung off the shared context and outlived
// every pop, so a screen's state silently leaked into its own next visit.
//
// Asserting "Contador: 0" rather than "not 2" is deliberate: it distinguishes
// a genuinely fresh frame from one that was merely re-initialized part-way.
func TestPushedRouteStateIsDiscardedOnPop(t *testing.T) {
	core.ClearConcerns()

	mgr := render.New(core.NewContext().WithTheme(core.DefaultTheme), App)
	defer mgr.Close()
	mgr.RenderInitial()

	tap(t, mgr, "Abrir Detalhes")
	tap(t, mgr, "➕ Incrementar")
	if after := tap(t, mgr, "➕ Incrementar"); !strings.Contains(after, "Contador: 2") {
		t.Fatalf("counter did not reach 2 before the pop:\n%s", after)
	}
	tap(t, mgr, "⬅️ Voltar")

	again := tap(t, mgr, "Abrir Detalhes")
	if !strings.Contains(again, "Contador: 0") {
		t.Fatalf("re-pushed route inherited the popped frame's state:\n%s", again)
	}

	assertNoConcerns(t)
}
