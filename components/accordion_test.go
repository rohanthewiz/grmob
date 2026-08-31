package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// renderPass drives one framework-shaped pass over a view, the sequence
// render.Manager performs. Accordion owns state via NewState, so its tests
// must respect the pass protocol rather than calling Render bare.
func renderPass(ctx *core.Context, v core.View) *core.Node {
	ctx.BeginRenderPass()
	ctx.Reset()
	return v.Render(ctx)
}

func TestAccordionTogglesContent(t *testing.T) {
	ctx := core.NewContext()
	acc := Accordion{
		Title:   "Details",
		Content: core.Text("hidden treasure"),
	}

	n := renderPass(ctx, acc)
	if findText(n, "hidden treasure") != nil {
		t.Fatal("content should start collapsed")
	}
	header := findFirst(n, func(n *core.Node) bool { return n.Props["onClick"] != nil })
	if header == nil {
		t.Fatal("header should carry an onClick toggle")
	}
	if findText(n, "▸") == nil {
		t.Error("collapsed accordion should show the closed chevron")
	}

	// Tap the header, then re-render: content appears, chevron flips.
	ctx.TriggerCallback(header.Props["onClick"].(string))
	n = renderPass(ctx, acc)
	if findText(n, "hidden treasure") == nil {
		t.Fatal("content should render after expanding")
	}
	if findText(n, "▾") == nil {
		t.Error("expanded accordion should show the open chevron")
	}

	// Tap again: collapses back.
	header = findFirst(n, func(n *core.Node) bool { return n.Props["onClick"] != nil })
	ctx.TriggerCallback(header.Props["onClick"].(string))
	n = renderPass(ctx, acc)
	if findText(n, "hidden treasure") != nil {
		t.Error("content should collapse on the second tap")
	}
}

func TestAccordionInitiallyExpanded(t *testing.T) {
	ctx := core.NewContext()
	n := renderPass(ctx, Accordion{
		Title:             "Open by default",
		Content:           core.Text("visible"),
		InitiallyExpanded: true,
	})
	if findText(n, "visible") == nil {
		t.Error("InitiallyExpanded should seed the state expanded")
	}
}

func TestAccordionHeaderSlot(t *testing.T) {
	ctx := core.NewContext()
	n := renderPass(ctx, Accordion{
		Title:   "For accessibility",
		Header:  core.Text("custom header"),
		Content: core.Text("content"),
	})
	if findText(n, "custom header") == nil {
		t.Error("Header slot should replace the default header content")
	}
	if findText(n, "For accessibility") != nil {
		t.Error("default title text should be suppressed when Header is set")
	}
}
