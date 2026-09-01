package render_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// jsonNode mirrors the wire shape of core.Node for decoding RenderInitial output.
type jsonNode struct {
	Type     string
	Props    map[string]any
	Children []jsonNode
}

// jsonPatch mirrors the wire shape of reconcile.Patch for decoding RenderAgain output.
type jsonPatch struct {
	Type     string
	TargetID string
	Changes  any
}

// counterApp is the canonical stateful UI: a Text bound to state plus a Button
// that mutates it. It exercises the full loop the native bridges rely on:
// render -> event by callback ID -> re-render -> minimal patch set.
func counterApp(ctx *core.Context) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		count := core.NewState(ctx, 0)
		return core.Column(
			core.Text(fmt.Sprintf("count: %d", count.Get())),
			core.Button("increment", func() {
				count.Set(count.Get() + 1)
			}),
		).Render(ctx)
	})
}

func decodeTree(t *testing.T, s string) jsonNode {
	t.Helper()
	var n jsonNode
	if err := json.Unmarshal([]byte(s), &n); err != nil {
		t.Fatalf("failed to decode tree JSON: %v\n%s", err, s)
	}
	return n
}

func decodePatches(t *testing.T, s string) []jsonPatch {
	t.Helper()
	var p []jsonPatch
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		t.Fatalf("failed to decode patches JSON: %v\n%s", err, s)
	}
	return p
}

func TestNoOpReRenderProducesNoPatches(t *testing.T) {
	// The flagship regression test for callback ID churn. Before per-pass ID
	// sequencing, the Button's onClick prop carried a freshly minted ID every
	// render, so even a no-change re-render diffed as update-props on every
	// interactive node. A correct render loop must produce an empty patch set.
	m := render.New(core.NewContext(), counterApp)
	m.RenderInitial()

	out := m.RenderAgain()
	if out != "[]" {
		t.Fatalf("no-op re-render must serialize as \"[]\" (renderers iterate it unchecked), got: %s", out)
	}
}

func TestStateChangePatchesOnlyTheAffectedNode(t *testing.T) {
	m := render.New(core.NewContext(), counterApp)
	tree := decodeTree(t, m.RenderInitial())

	if len(tree.Children) != 2 {
		t.Fatalf("expected Column with 2 children, got %+v", tree)
	}
	if got := tree.Children[0].Props["content"]; got != "count: 0" {
		t.Fatalf("initial text = %v, want \"count: 0\"", got)
	}
	onClick, ok := tree.Children[1].Props["onClick"].(string)
	if !ok || onClick == "" {
		t.Fatalf("button is missing its onClick callback ID: %+v", tree.Children[1].Props)
	}

	// Dispatch the event exactly as a native bridge does: by callback ID,
	// through the manager, which returns the resulting patches directly.
	patches := decodePatches(t, m.DispatchCallback(onClick))
	if len(patches) != 1 {
		t.Fatalf("expected exactly 1 patch (the Text update), got %d: %+v", len(patches), patches)
	}
	p := patches[0]
	if p.Type != "update-props" || p.TargetID != "root/0" {
		t.Errorf("patch = %+v, want update-props targeting root/0", p)
	}
	changed, ok := p.Changes.(map[string]any)
	if !ok || changed["content"] != "count: 1" {
		t.Errorf("patched props = %v, want content \"count: 1\"", p.Changes)
	}
}

func TestCallbackIDSurvivesReRenderAndKeepsWorking(t *testing.T) {
	// The ID handed to the native side at initial render must stay valid and
	// dispatch to the freshest closure after any number of re-renders —
	// renderers only re-bind listeners when an update-props patch tells them to.
	m := render.New(core.NewContext(), counterApp)
	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	for i := 1; i <= 3; i++ {
		patches := decodePatches(t, m.DispatchCallback(onClick))
		if len(patches) != 1 || patches[0].TargetID != "root/0" {
			t.Fatalf("click %d: expected single Text patch, got %+v", i, patches)
		}
		if got := patches[0].Changes.(map[string]any)["content"]; got != fmt.Sprintf("count: %d", i) {
			t.Fatalf("click %d: content = %v — stale closure? IDs must dispatch to the latest registration", i, got)
		}
	}
}

// TestRenderAndGetPatchesMountsThenDiffs pins the delegating rewrite.
//
// The hand-rolled version reset only the root cursor (`ctx.Cursor = 0` rather
// than `Reset()`), so every child scope resumed from the previous pass's
// cursor and appended a fresh slot each render: child state was lost every
// pass and the slot array grew without bound. It also skipped
// PurgeUnusedCallbacks and ClearDirty, so handlers for vanished nodes stayed
// dispatchable and a polling host re-rendered forever.
func TestRenderAndGetPatchesMountsThenDiffs(t *testing.T) {
	ctx := core.NewContext()

	// The state lives in a *child* scope, which is where the cursor bug bit:
	// the root's cursor was zeroed each pass, every scope's was not.
	var (
		once sync.Once
		st   core.State[int]
	)
	app := func(ctx *core.Context) core.View {
		return core.ComponentFunc(func(ctx *core.Context) *core.Node {
			child := core.UseChildContext(ctx)
			n := core.NewState(child, 0)
			once.Do(func() { st = n })
			return core.Column(core.Text(fmt.Sprintf("n: %d", n.Get()))).Render(ctx)
		})
	}

	m := render.New(ctx, app)
	defer m.Close()

	// First call mounts: a full tree, not a patch list.
	tree := decodeTree(t, m.RenderAndGetPatches())
	if tree.Type != "Column" {
		t.Fatalf("first call returned %q, want the mounted tree", tree.Type)
	}

	// Later calls diff. Each write must land on the slot the previous pass
	// used, which is only true if the child scope was reset along with the
	// root.
	for pass := 1; pass <= 3; pass++ {
		st.Set(pass)
		out := m.RenderAndGetPatches()
		decodePatches(t, out) // must be a patch list, not a tree
		want := fmt.Sprintf("n: %d", pass)
		if !strings.Contains(out, want) {
			t.Fatalf("pass %d patches = %s, want them to carry %q", pass, out, want)
		}
	}

	// ClearDirty ran: nothing has changed since the last pass consumed it, so
	// a polling host must not be told to render again.
	if ctx.IsDirty() {
		t.Error("the tree is still marked dirty after a completed pass")
	}
}
