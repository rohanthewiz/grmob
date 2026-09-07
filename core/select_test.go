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

// The two optional fields are written to the wire only when they say
// something, so an ordinary option crosses in exactly the shape it always did.
//
// That is not tidiness. The WASM runtime decides whether to rebuild a picker's
// <option> elements by comparing the option list's JSON, and rebuilding closes
// an open drop-down mid-choice — so a key that appeared on every option with
// an empty value would have changed every signature the first time this
// shipped, for nothing.
func TestSelectWritesGroupAndDisabledOnlyWhenSet(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Select("", []SelectOption{
		{Value: "a", Label: "A"},
		{Value: "b", Label: "B", Group: "Letters"},
		{Value: "c", Label: "C", Disabled: true},
	}, nil).Render(ctx)

	opts, ok := n.Props["options"].([]map[string]string)
	if !ok {
		t.Fatalf("options prop is %T, want []map[string]string", n.Props["options"])
	}
	if len(opts) != 3 {
		t.Fatalf("option count = %d, want 3", len(opts))
	}

	// The plain option carries the two required keys and nothing else, which
	// is the assertion the signature argument rests on.
	if len(opts[0]) != 2 {
		t.Errorf("a plain option flattened to %v, want the value and label alone", opts[0])
	}
	if _, present := opts[0]["group"]; present {
		t.Error("an ungrouped option carries a group key")
	}
	if _, present := opts[0]["disabled"]; present {
		t.Error("an enabled option carries a disabled key")
	}

	if opts[1]["group"] != "Letters" {
		t.Errorf("group = %q, want Letters", opts[1]["group"])
	}
	if _, present := opts[1]["disabled"]; present {
		t.Error("a grouped option picked up a disabled key")
	}

	// "true", the spelling core.SelectedState uses: it is what the DOM writes,
	// so the web half needs no translation. A bool cannot travel in this map.
	if opts[2]["disabled"] != "true" {
		t.Errorf("disabled = %q, want %q", opts[2]["disabled"], "true")
	}
	if _, present := opts[2]["group"]; present {
		t.Error("a disabled option picked up a group key")
	}
}

// A disabled option is still an option: it keeps its value and its label, and
// it stays in the list at its own index.
//
// The alternative implementation — flatten it away — is the thing this rules
// out, and it is a tempting one, since nothing can choose it. It would be
// wrong for the reason the field exists: an option that vanishes takes its
// explanation with it, and a list that changes length between renders is one a
// person has to re-read.
func TestADisabledOptionKeepsItsPlaceInTheList(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Select("s", []SelectOption{
		{Value: "s", Label: "Small"},
		{Value: "m", Label: "Medium", Disabled: true},
		{Value: "l", Label: "Large"},
	}, nil).Render(ctx)

	opts := n.Props["options"].([]map[string]string)
	if len(opts) != 3 {
		t.Fatalf("option count = %d, want 3 — a disabled option was dropped", len(opts))
	}
	if opts[1]["value"] != "m" || opts[1]["label"] != "Medium" {
		t.Errorf("the disabled option flattened to %v, want its own value and label", opts[1])
	}
}

// The label default applies to a grouped option too. The fallback lives at
// this seam so no renderer carries it, and a second path through the loop is
// exactly where a second copy would appear.
func TestAGroupedOptionStillDefaultsItsLabel(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Select("", []SelectOption{{Value: "XL", Group: "Sizes"}}, nil).Render(ctx)

	opts := n.Props["options"].([]map[string]string)
	if opts[0]["label"] != "XL" {
		t.Errorf("label = %q, want the value %q", opts[0]["label"], "XL")
	}
	if opts[0]["group"] != "Sizes" {
		t.Errorf("group = %q, want Sizes", opts[0]["group"])
	}
}

// GroupDisabled crosses the wire on the same terms as the other two optional
// keys: written only when it says something, so an ordinary option's flattened
// map is unchanged and the WASM runtime's rebuild signature — the list as JSON
// — does not shift the first time this ships.
func TestSelectWritesGroupDisabledOnlyWhenSet(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Select("", []SelectOption{
		{Value: "free", Label: "Free", Group: "Plans"},
		{Value: "pro", Label: "Pro", Group: "Paid", GroupDisabled: true},
	}, nil).Render(ctx)

	opts := n.Props["options"].([]map[string]string)
	if _, present := opts[0]["groupDisabled"]; present {
		t.Error("an option in a live run carries a groupDisabled key")
	}
	if opts[1]["groupDisabled"] != "true" {
		t.Errorf("groupDisabled = %q, want %q", opts[1]["groupDisabled"], "true")
	}
	// The declaration is about the run, so it does not imply the option's own
	// Disabled. Resolving the two is SelectMenuSections' job, once, for all
	// four renderers — this seam only carries what the caller wrote.
	if _, present := opts[1]["disabled"]; present {
		t.Error("GroupDisabled wrote the option's own disabled key at the flattening seam; " +
			"the resolution belongs in SelectMenuSections, where every renderer gets it")
	}
}
