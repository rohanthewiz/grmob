package htmlout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The defect this table closed: a container with no Style, or with a Style
// that says nothing about layout, exported as a block-flow <div>. Text
// exports as a <span>, so a bare Column of texts ran its children together on
// one line in the browser and stacked them on every other target.
//
// Both halves are checked per type — the promotion and the axis — because
// getting the promotion right with the wrong axis is the same bug wearing a
// different hat: a Row that stacks vertically is exactly what block flow was
// already doing.
func TestStackContainersAreFlexWithNoStyleAtAll(t *testing.T) {
	for _, nodeType := range StackTypes() {
		out := ExportHTML(&core.Node{Type: nodeType})
		want := fmt.Sprintf("display:flex; flex-direction:%s", StackAxisFor(nodeType))
		if !strings.Contains(out, want) {
			t.Errorf("%s with a nil Style: missing %q:\n%s", nodeType, want, out)
		}
	}
}

// The same, for a Style that exists but declares nothing about layout. This
// is the shape the bug was actually found in — examples/layout's BodySection
// is a core.Row carrying only padding, and it exported as
// `<div style="padding:…">` — and it is a genuinely different path from the
// nil case above, since styleValue's early return is what the nil case
// exercises.
func TestStackContainersAreFlexWithANonLayoutStyle(t *testing.T) {
	for _, nodeType := range StackTypes() {
		out := ExportHTML(&core.Node{Type: nodeType, Style: &core.Style{Padding: core.EdgeInsets{Top: 12, Right: 12, Bottom: 12, Left: 12}}})
		want := fmt.Sprintf("display:flex; flex-direction:%s", StackAxisFor(nodeType))
		if !strings.Contains(out, want) {
			t.Errorf("%s with padding only: missing %q:\n%s", nodeType, want, out)
		}
	}
}

// The gate's other side: the default belongs to the container types and to no
// one else. styleValue serializes every node through one function, so a
// missing gate would turn a Text or a Button into a flex container, and its
// children — a Button's label is its text content — would be laid out as flex
// items rather than as the inline content they are.
//
// Spacer and Modal are here as the two container-ish types the table
// deliberately omits: Spacer is a sized void with no children, and Modal's
// overlay chassis sets display itself and toggles it through the visible prop,
// which a default here would fight.
func TestNonStackTypesStayInBlockFlow(t *testing.T) {
	cases := []*core.Node{
		{Type: "Text", Props: map[string]any{"content": "hi"}},
		{Type: "Button", Props: map[string]any{"label": "go"}},
		{Type: "Image", Props: map[string]any{"src": "a.png"}},
		{Type: "Spacer", Props: map[string]any{"size": 8}},
		{Type: "Modal", Props: map[string]any{"visible": false}},
		{Type: "TabView"},
	}
	for _, n := range cases {
		if out := ExportHTML(n); strings.Contains(out, "display:flex") {
			t.Errorf("%s was promoted to a flex container:\n%s", n.Type, out)
		}
	}
}

// An explicit FlexDirection still wins over the type's own axis. The default
// is a default, not an override: core.FlexDirection("row") on a Column is the
// author saying the node stacks the other way, and both natives honor it
// (GrMobFlexStack switches on the same field).
func TestExplicitFlexDirectionBeatsTheStackAxis(t *testing.T) {
	out := ExportHTML(&core.Node{Type: "Column", Style: &core.Style{FlexDirection: "row"}})
	if !strings.Contains(out, "flex-direction:row") {
		t.Errorf("FlexDirection did not override the Column's own axis:\n%s", out)
	}
	if strings.Contains(out, "flex-direction:column") {
		t.Errorf("both axes emitted:\n%s", out)
	}
}

// The stacking default must not swallow the one Display that outranks it.
// DisplayNone hides a node on both natives (each Renderer bails before any
// layout) and it has to here too, so it is emitted after the flex block and
// wins the browser's last-declaration-wins parse. Before the default existed
// this was only reachable on a container that had asked for flex some other
// way; now every container reaches it.
func TestDisplayNoneStillBeatsTheStackDefault(t *testing.T) {
	out := ExportHTML(&core.Node{Type: "Column", Style: &core.Style{Display: core.DisplayNone}})
	flex := strings.Index(out, "display:flex")
	none := strings.Index(out, "display:none")
	if flex < 0 || none < 0 {
		t.Fatalf("want both display:flex and display:none:\n%s", out)
	}
	if none < flex {
		t.Fatalf("display:none precedes display:flex, so the container is not hidden:\n%s", out)
	}
}

// A copy, for the reason Tags hands out one: the conformance test deletes
// from what it is given as it matches rows, and a package-level map is
// writable by any importer.
func TestStackAxesReturnsACopy(t *testing.T) {
	axes := StackAxes()
	delete(axes, "Row")
	axes["Column"] = "row"
	if got := StackAxisFor("Row"); got != "row" {
		t.Errorf("deleting from the returned map changed StackAxisFor(Row): %q", got)
	}
	if got := StackAxisFor("Column"); got != "column" {
		t.Errorf("writing to the returned map changed StackAxisFor(Column): %q", got)
	}
}

// Every stack container is a real node type with a real tag. A row here for a
// type tags does not know about would be a stacking default planted on a box
// nothing builds — the kind of drift the census tables exist to surface.
func TestStackTypesAreKnownTags(t *testing.T) {
	known := Tags()
	for _, nodeType := range StackTypes() {
		if _, ok := known[nodeType]; !ok {
			t.Errorf("%s stacks but is not in the tag table", nodeType)
		}
		if IsTransparent(nodeType) {
			t.Errorf("%s stacks but is transparent here — it has no box to stack in", nodeType)
		}
	}
}
