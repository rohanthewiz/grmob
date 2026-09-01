package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// renderRow renders r on a fresh context under the given theme and returns
// both the Row node and the context, because half of what this widget does is
// register callbacks — the tests below fire them through the context rather
// than inspecting the funcs, which is the only way to prove the button and
// the keyboard reach the same handler.
func renderRow(t *testing.T, theme *core.Theme, r InputRow) (*core.Node, *core.Context) {
	t.Helper()
	ctx := core.NewContext().WithTheme(theme)
	ctx.BeginRenderPass()
	return r.Render(ctx), ctx
}

func inputNode(t *testing.T, root *core.Node) *core.Node {
	t.Helper()
	n := findFirst(root, func(n *core.Node) bool { return n.Type == "Input" })
	if n == nil {
		t.Fatalf("no Input in the rendered row: %s", describe(root))
	}
	return n
}

func buttonNode(root *core.Node) *core.Node {
	return findFirst(root, func(n *core.Node) bool { return n.Type == "Button" })
}

// The shape, and the one style prop that makes it a composer rather than two
// controls side by side. A field that does not grow leaves the button floating
// mid-row on every screen wider than the placeholder.
func TestInputRowShapeAndGrowingField(t *testing.T) {
	root, _ := renderRow(t, core.DefaultTheme, InputRow{
		Placeholder: "Say something",
		OnSubmit:    func() {},
		Button:      Button{Label: "Send"},
	})

	if root.Type != "Row" || len(root.Children) != 2 {
		t.Fatalf("want Row with 2 children, got %s", describe(root))
	}
	// Order is load-bearing beyond appearance: callback IDs are handed out in
	// render order, and examples/chat's event simulation names them.
	if root.Children[0].Type != "Input" || root.Children[1].Type != "Button" {
		t.Fatalf("want Row > [Input, Button], got %s", describe(root))
	}
	if g := inputNode(t, root).Style.FlexGrow; g != 1 {
		t.Errorf("field FlexGrow = %v, want 1 — the button would not be pushed to the end", g)
	}
	if g := buttonNode(root).Style.FlexGrow; g != 0 {
		t.Errorf("button FlexGrow = %v, want 0 — it should keep its intrinsic width", g)
	}
}

// Gap defaults to the theme's SM step. Both bundled themes set SM to 8, which
// is also the literal both migrated call sites used to write, so a hardcoded 8
// would pass under either of them — this uses a theme whose SM is something
// else, where "read the theme" and "hardcode the number they happen to share"
// finally disagree.
func TestInputRowGapDefaultsToTheThemeSMStep(t *testing.T) {
	theme := &core.Theme{
		Colors:     core.DefaultTheme.Colors,
		Typography: core.DefaultTheme.Typography,
		Spacing:    core.SpacingScale{XS: 2, SM: 5, MD: 11, LG: 19, XL: 27},
	}

	root, _ := renderRow(t, theme, InputRow{OnSubmit: func() {}})
	if g := root.Style.Gap; g != 5 {
		t.Fatalf("default Gap = %v, want the theme's SM step 5", g)
	}

	// And the bundled themes still produce the 8 the migrations depend on.
	for name, bundled := range map[string]*core.Theme{
		"Default":  core.DefaultTheme,
		"Material": core.MaterialTheme,
	} {
		root, _ := renderRow(t, bundled, InputRow{OnSubmit: func() {}})
		if g := root.Style.Gap; g != 8 {
			t.Errorf("%s theme: default Gap = %v, want 8", name, g)
		}
	}
}

func TestInputRowExplicitGapWinsOverTheDefault(t *testing.T) {
	root, _ := renderRow(t, core.DefaultTheme, InputRow{Gap: 20, OnSubmit: func() {}})
	if g := root.Style.Gap; g != 20 {
		t.Fatalf("explicit Gap = %v, want 20", g)
	}
}

// Style is appended after the gap, so it is a real escape hatch. Gap(0) is the
// case that matters most: it is the only way to ask for no spacing at all,
// since the Gap field's zero means "the theme's step". If the order were
// reversed the widget's own gap would silently win.
func TestInputRowStyleOverridesTheWidgetsOwnGap(t *testing.T) {
	root, _ := renderRow(t, core.DefaultTheme, InputRow{
		Gap:      20,
		OnSubmit: func() {},
		Style:    []core.StyleProp{core.Gap(0), core.Padding(12)},
	})

	if g := root.Style.Gap; g != 0 {
		t.Errorf("Style Gap did not win: got %v, want 0", g)
	}
	if root.Style.Padding.Top != 12 {
		t.Errorf("Style Padding not applied: got %+v", root.Style.Padding)
	}
}

// The wiring claim in the doc comment, proved by dispatch rather than by
// reading the struct: both the keyboard's submit and the button's tap must
// land on the one handler the caller named.
func TestInputRowButtonAndKeyboardShareOneHandler(t *testing.T) {
	commits := 0
	root, ctx := renderRow(t, core.DefaultTheme, InputRow{
		OnSubmit: func() { commits++ },
		Button:   Button{Label: "Send"},
	})

	ctx.TriggerCallback(inputNode(t, root).Props["onSubmit"].(string))
	if commits != 1 {
		t.Fatalf("keyboard submit ran the handler %d times, want 1", commits)
	}

	ctx.TriggerCallback(buttonNode(root).Props["onClick"].(string))
	if commits != 2 {
		t.Fatalf("button tap did not reach OnSubmit: %d commits, want 2", commits)
	}
}

// The inheritance is a default, not a rule — a caller who names OnTap keeps it.
func TestInputRowExplicitButtonOnTapWins(t *testing.T) {
	submits, taps := 0, 0
	root, ctx := renderRow(t, core.DefaultTheme, InputRow{
		OnSubmit: func() { submits++ },
		Button:   Button{Label: "Send", OnTap: func() { taps++ }},
	})

	ctx.TriggerCallback(buttonNode(root).Props["onClick"].(string))
	if taps != 1 || submits != 0 {
		t.Fatalf("taps = %d, submits = %d; want the explicit OnTap alone (1, 0)", taps, submits)
	}
}

// OnChange is what keeps the field controlled: without it the next render
// paints Value back over whatever was typed. Both halves are pinned — the
// value reaches the node, and the keystroke reaches the caller.
func TestInputRowFieldIsControlled(t *testing.T) {
	var got string
	root, ctx := renderRow(t, core.DefaultTheme, InputRow{
		Value:       "draft so far",
		Placeholder: "Mensagem…",
		OnChange:    func(v string) { got = v },
		OnSubmit:    func() {},
	})

	in := inputNode(t, root)
	if in.Props["value"] != "draft so far" {
		t.Errorf("value = %v, want %q", in.Props["value"], "draft so far")
	}
	if in.Props["placeholder"] != "Mensagem…" {
		t.Errorf("placeholder = %v", in.Props["placeholder"])
	}

	ctx.TriggerTextCallback(in.Props["onChange"].(string), "typed")
	if got != "typed" {
		t.Fatalf("OnChange got %q, want %q", got, "typed")
	}
}

// A zero Button contributes no node at all, rather than an empty one: a search
// field that commits on the return key is this widget with one less field set,
// and an empty Button would take a slot in the row's flex layout and open a
// gap where nothing is drawn.
func TestInputRowZeroButtonRendersNoNode(t *testing.T) {
	root, _ := renderRow(t, core.DefaultTheme, InputRow{OnSubmit: func() {}})

	if len(root.Children) != 1 {
		t.Fatalf("want the field alone, got %s", describe(root))
	}
	if b := buttonNode(root); b != nil {
		t.Fatalf("zero Button still produced a node: %s", describe(root))
	}
}

// With no OnSubmit the field is built without a submit path rather than with
// one wired to a no-op — the renderers read the prop to decide whether to show
// a submit affordance on the keyboard, so a registered no-op would advertise
// an action the row ignores.
func TestInputRowWithoutOnSubmitOmitsTheSubmitProp(t *testing.T) {
	root, _ := renderRow(t, core.DefaultTheme, InputRow{Placeholder: "Filter"})

	in := inputNode(t, root)
	if _, ok := in.Props["onSubmit"]; ok {
		t.Fatalf("nil OnSubmit still registered a submit callback: %+v", in.Props)
	}

	// A button with no handler to inherit must still be inert rather than
	// panic when a late tap dispatches to it — Button's own nil-OnTap
	// contract, exercised through the path that produces the nil.
	root, ctx := renderRow(t, core.DefaultTheme, InputRow{Button: Button{Label: "Go"}})
	ctx.TriggerCallback(buttonNode(root).Props["onClick"].(string))
}

// Button is passed through whole, not re-spelled: everything the widget can do
// elsewhere keeps working inside the row. Variant is the representative case —
// if InputRow rebuilt the button from a Label it would be silently dropped.
func TestInputRowPassesTheButtonThrough(t *testing.T) {
	root, _ := renderRow(t, core.DefaultTheme, InputRow{
		OnSubmit: func() {},
		Button: Button{
			Label:             "Send",
			Variant:           VariantError,
			AccessibilityHint: "Sends the message",
		},
	})

	b := buttonNode(root)
	if b.Props["label"] != "Send" {
		t.Fatalf("label = %v", b.Props["label"])
	}
	if want := core.DefaultTheme.Colors.Error; b.Style.Background != want {
		t.Errorf("Variant dropped: background = %q, want %q", b.Style.Background, want)
	}
	if b.Style.AccessibilityHint != "Sends the message" {
		t.Errorf("AccessibilityHint dropped: %q", b.Style.AccessibilityHint)
	}
}

// TestInputRowNilOnChangeDoesNotPanic pins the nil-handler guard. The doc on
// OnChange describes an omitted handler as leaving the field "read-only in
// practice"; before the guard it instead panicked on the first keystroke,
// because core.Input registers the nil func and the registry calls it
// unguarded.
func TestInputRowNilOnChangeDoesNotPanic(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := InputRow{Value: "hello", Placeholder: "Say something"}.Render(ctx)
	field := n.Children[0]
	id, ok := field.Props["onChange"].(string)
	if !ok {
		t.Fatalf("input row's field should register an onChange callback, props: %v", field.Props)
	}
	if err := core.Guard(func() { ctx.TriggerTextCallback(id, "typed") }); err != nil {
		t.Fatalf("typing into an InputRow with no OnChange panicked: %v", err.Value)
	}
}

// TestInputRowNilOnChangeWithSubmitDoesNotPanic covers the other branch:
// InputWithSubmit takes the same handler down a different core builder.
func TestInputRowNilOnChangeWithSubmitDoesNotPanic(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := InputRow{Value: "hello", OnSubmit: func() {}}.Render(ctx)
	field := n.Children[0]
	id, ok := field.Props["onChange"].(string)
	if !ok {
		t.Fatalf("input row's field should register an onChange callback, props: %v", field.Props)
	}
	if err := core.Guard(func() { ctx.TriggerTextCallback(id, "typed") }); err != nil {
		t.Fatalf("typing into a submit-wired InputRow with no OnChange panicked: %v", err.Value)
	}
}
