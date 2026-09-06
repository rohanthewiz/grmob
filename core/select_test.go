package core

import "testing"

// The wire shape of a picker: the value, the flattened option list, and a
// *text* callback ID.
//
// The callback channel is the part worth pinning. core.Select takes a
// func(string), which registerTextCallback puts in the text map; a renderer
// that sent the choice through the void or int channel would reach Go with an
// ID that map has no entry for, and the handler would silently never run —
// the failure mode extractEventPayload's doc describes from the other side.
func TestSelectSendsTheValueThroughTheTextChannel(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	var got string
	n := Select("pt", []SelectOption{
		{Value: "us", Label: "United States"},
		{Value: "pt", Label: "Portugal"},
	}, func(v string) { got = v }).Render(ctx)

	if n.Type != "Select" {
		t.Fatalf("node type = %q, want Select", n.Type)
	}
	if n.Props["value"] != "pt" {
		t.Errorf("value = %v, want pt", n.Props["value"])
	}

	id, ok := n.Props["onChange"].(string)
	if !ok {
		t.Fatal("no onChange callback ID")
	}
	ctx.TriggerTextCallback(id, "us")
	if got != "us" {
		t.Errorf("the handler received %q, want us", got)
	}
}

// The label defaults to the value at the core seam, once, so no renderer has
// to carry the fallback and none of them can disagree about it.
func TestSelectOptionsFlattenWithTheLabelDefaulted(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Select("", []SelectOption{
		{Value: "s", Label: "Small"},
		{Value: "M"}, // no label: the value is readable enough
	}, nil).Render(ctx)

	opts, ok := n.Props["options"].([]map[string]string)
	if !ok {
		t.Fatalf("options prop is %T, want []map[string]string — the wire shape every "+
			"renderer reads", n.Props["options"])
	}
	if len(opts) != 2 {
		t.Fatalf("option count = %d, want 2", len(opts))
	}
	if opts[0]["value"] != "s" || opts[0]["label"] != "Small" {
		t.Errorf("first option = %v, want {value: s, label: Small}", opts[0])
	}
	if opts[1]["value"] != "M" || opts[1]["label"] != "M" {
		t.Errorf("unlabelled option = %v, want the value used as the label", opts[1])
	}
}

// A picker reads the theme's Input base, which is how it inherits the frame,
// the radius and the fill the text fields around it have — and the reason it
// needs no palette role of its own. It is also what makes the <select> border
// reset safe on the web: there is something to reset *to*.
func TestSelectWearsTheThemesFieldStyle(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	base := ctx.Theme().Components.Input
	if base.BorderWidth == 0 || base.BorderColor == "" {
		t.Fatal("the theme's Input base carries no frame; this test cannot tell inheritance " +
			"from an empty style")
	}

	n := Select("", nil, nil).Render(ctx)
	if n.Style.BorderColor != base.BorderColor || n.Style.BorderWidth != base.BorderWidth {
		t.Errorf("picker frame = %gpx %q, want the Input base's %gpx %q",
			n.Style.BorderWidth, n.Style.BorderColor, base.BorderWidth, base.BorderColor)
	}
	// The control: a Box in the same context inherits nothing, so the check
	// above is about the Select and not about a theme that styles everything.
	if b := Box().Render(ctx); b.Style.BorderColor != "" {
		t.Errorf("the control Box arrived with a frame %q", b.Style.BorderColor)
	}
}

// A picker takes no keyboard focus stamp. It looks like a field and reads a
// field's theme base, so the omission is the kind that reads as an oversight;
// see focusableLeafTypes for why it is not one.
func TestSelectIsNotStampedWithTheFocusCommand(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	Focus(UseFocusRef(ctx))

	sel := Select("", nil, nil).Render(ctx)
	if _, stamped := sel.Props["focusAction"]; stamped {
		t.Error("a picker carries the focus command; neither native gives a menu keyboard focus, " +
			"so the patch travels to two renderers with nowhere to put it")
	}
	// The control: a text field in the same pass does take it, so the absence
	// above is about the node type rather than about a command nobody issued.
	field := Input("", "", nil).Render(ctx)
	if _, stamped := field.Props["focusAction"]; !stamped {
		t.Fatal("no focus command was in flight; this test proves nothing")
	}
}
