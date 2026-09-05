package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The exporter's half of roadmap tier B. Two of the three props needed no
// code here at all, and that is the claim worth holding: core.Horizontal and
// core.StickyHeader were designed to spell themselves in Style fields this
// package already emits, so if the export ever stops carrying them the design
// premise has failed and the natives are the only targets left implementing
// the feature.

// core.Horizontal on a Scroll: the axis overrides the node's stacking
// default, and the overflow is what makes the strip pan instead of clip.
func TestHorizontalScrollExportsAsARowThatOverflows(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := core.Scroll(core.Horizontal(), core.Gap(8), core.Text("chip")).Render(ctx)

	out := ExportHTML(n)
	for _, want := range []string{"flex-direction:row", "overflow:auto", "display:flex", "gap:8px"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	// The stack table says a Scroll is a column; an explicit FlexDirection is
	// the documented override, and this is the first prop in the tree to use
	// it in anger.
	if strings.Contains(out, "flex-direction:column") {
		t.Errorf("the node's stacking default won over core.Horizontal:\n%s", out)
	}
}

// core.StickyHeader on a List child: the three declarations that pin a band
// in a browser. Top and z-index matter as much as position — a sticky box
// with no offset never sticks, and one at the default layer is painted over
// by the rows scrolling under it.
func TestStickyHeaderExportsThePinningDeclarations(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := core.List(
		core.Keyed("group:2026-01", core.Row(core.StickyHeader(), core.Text("January 2026"))),
		core.Keyed("s1", core.Text("a sermon")),
	).Render(ctx)

	out := ExportHTML(n)
	for _, want := range []string{"position:sticky", "top:0", "z-index:1"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// core.OnEndReached is recorded, not wired: a static document has no observer
// to report a scroll position with. What the attribute buys is that a loader
// which does have one knows which callback the bottom of this list belongs to
// — the same deal every other data-on* attribute here strikes.
func TestOnEndReachedExportsItsCallbackID(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := core.List(core.OnEndReached(func() {}), core.Text("row")).Render(ctx)

	id, _ := n.Props["onEndReached"].(string)
	if id == "" {
		t.Fatal("the List carries no onEndReached prop to export")
	}
	out := ExportHTML(n)
	if want := `data-onendreached="` + id + `"`; !strings.Contains(out, want) {
		t.Errorf("missing %q:\n%s", want, out)
	}
}
