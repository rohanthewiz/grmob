package render_test

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// Manager.Resync (N-133, the plan's R4): the whole tree Go believes the host
// holds, as one replace of "root", for a host whose tree has drifted.

// resyncPayload decodes a Resync answer into its one patch's tree.
func resyncPayload(t *testing.T, out string) jsonNode {
	t.Helper()
	var patches []struct {
		Type     string
		TargetID string
		Changes  jsonNode
	}
	if err := json.Unmarshal([]byte(out), &patches); err != nil {
		t.Fatalf("resync payload is not a patch list: %v\n%s", err, out)
	}
	if len(patches) != 1 || patches[0].Type != "replace" || patches[0].TargetID != "root" {
		t.Fatalf("resync payload = %s, want exactly one replace of root", out)
	}
	return patches[0].Changes
}

func TestResyncBeforeTheMountSendsNothing(t *testing.T) {
	m := render.New(core.NewContext(), counterApp)
	defer m.Close()
	if out := m.Resync(); out != "[]" {
		t.Fatalf("Resync before RenderInitial = %s, want []", out)
	}
}

// Without a listener the replace is returned, and it is the tree the host was
// last brought to: the mount plus every pass since, re-rendering nothing.
func TestResyncReturnsTheTreeTheHostWasLastBroughtTo(t *testing.T) {
	ctx := core.NewContext()
	m := render.New(ctx, counterApp)
	defer m.Close()
	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)
	m.DispatchCallback(onClick) // count: 1, returned (no listener)

	got := resyncPayload(t, m.Resync())
	if got.Children[0].Props["content"] != "count: 1" {
		t.Fatalf("the resync tree reads %v, want the committed pass's \"count: 1\"", got.Children[0].Props["content"])
	}
	// The same tree a fresh mount of that state would send, callback IDs
	// included: the IDs in it are the ones still registered.
	again := decodeTree(t, m.RenderInitial())
	if !reflect.DeepEqual(got, again) {
		t.Errorf("resync tree %+v differs from the current render %+v", got, again)
	}
	// A write that has not been rendered is not in it: Resync answers "what
	// should the host hold", not "what would a render say".
	ctx.TriggerCallback(onClick)
	if got := resyncPayload(t, m.Resync()); got.Children[0].Props["content"] != "count: 1" {
		t.Errorf("resync rendered the pending write: %v", got.Children[0].Props["content"])
	}
}

// With a listener the replace goes through it, in order with every other
// pass, and the call returns "[]", as a dispatch does (N-132).
func TestResyncWithAListenerDeliversThroughIt(t *testing.T) {
	m := render.New(core.NewContext(), counterApp)
	defer m.Close()
	m.RenderInitial()
	l := newChanListener()
	m.SetListener(l)

	if out := m.Resync(); out != "[]" {
		t.Fatalf("Resync with a listener returned %s, want []", out)
	}
	select {
	case p := <-l.ch:
		if got := resyncPayload(t, p); got.Children[0].Props["content"] != "count: 0" {
			t.Errorf("the pushed resync reads %v", got.Children[0].Props["content"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the resync never reached the listener")
	}
}

// After a dropped pass (N-133's other half) the host still holds the last good
// tree, and that is what a resync sends, not the pass that would not encode.
func TestResyncAfterADroppedPassSendsTheLastGoodTree(t *testing.T) {
	quietLog(t)
	app := &encodeApp{}
	m := render.New(core.NewContext(), app.view)
	defer m.Close()
	m.RenderInitial()
	app.label.Set("dropped")
	app.value.Set(math.NaN())
	m.RenderAgain()

	out := m.Resync()
	if strings.Contains(out, "dropped") || strings.Contains(out, "error") {
		t.Fatalf("the resync carries the dropped pass: %s", out)
	}
	if got := resyncPayload(t, out); got.Children[0].Props["content"] != "before" {
		t.Errorf("the resync reads %v, want the last good \"before\"", got.Children[0].Props["content"])
	}
}
