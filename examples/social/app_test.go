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
// share a context, and only the ctx.Scope in DetailsPage keeps them from
// aliasing the same positional slot.
//
// If the scope were removed, this test fails in two visible ways before the
// concerns assertion even runs — the counter reads the tab's string, and the
// popped shell finds an int where its tab name was.
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

// TestScopedStateSurvivesRemount pushes the same route twice. The scope is
// created on first use and stable forever after, so the counter is still 2 the
// second time Details appears — a positional child context would have been
// re-created and reset.
func TestScopedStateSurvivesRemount(t *testing.T) {
	core.ClearConcerns()

	mgr := render.New(core.NewContext().WithTheme(core.DefaultTheme), App)
	defer mgr.Close()
	mgr.RenderInitial()

	tap(t, mgr, "Abrir Detalhes")
	tap(t, mgr, "➕ Incrementar")
	tap(t, mgr, "➕ Incrementar")
	tap(t, mgr, "⬅️ Voltar")

	again := tap(t, mgr, "Abrir Detalhes")
	if !strings.Contains(again, "Contador: 2") {
		t.Fatalf("scoped counter did not survive the pop/push cycle:\n%s", again)
	}

	assertNoConcerns(t)
}
