package core

import "testing"

// ContentMode travels as a node prop rather than a Style field because every
// renderer expresses it through the image view's own API. These tests pin the
// prop's presence and spelling, which is the contract the three renderers
// decode against.

func TestImageOmitsContentModeByDefault(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := Image("a.png").Render(ctx)

	if _, ok := n.Props["contentMode"]; ok {
		t.Fatalf("plain Image emitted a contentMode prop: %#v", n.Props)
	}
	// The prop map staying exactly {src} is what keeps existing snapshots and
	// the reconciler's update-props diffs unchanged by this feature.
	if len(n.Props) != 1 || n.Props["src"] != "a.png" {
		t.Fatalf("props = %#v, want only src", n.Props)
	}
}

func TestImageWithModeEmitsTheMode(t *testing.T) {
	for _, mode := range []ContentMode{
		ContentModeFit, ContentModeFill, ContentModeStretch, ContentModeCenter,
	} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := NewContext()
			ctx.BeginRenderPass()
			n := ImageWithMode("a.png", mode).Render(ctx)

			// A string, not a ContentMode: the node is serialized to JSON for
			// the native bridges, and a named type that survived into Props
			// would still marshal as a string but would not compare equal to
			// one in a renderer test.
			if got := n.Props["contentMode"]; got != string(mode) {
				t.Errorf("contentMode = %#v, want %q", got, string(mode))
			}
		})
	}
}

func TestImageWithModeStillTakesStyleProps(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := ImageWithMode("a.png", ContentModeFill, Width("80px"), BorderRadius(40)).Render(ctx)

	if n.Style.Width != "80px" || n.Style.BorderRadius != 40 {
		t.Fatalf("style props were dropped: %#v", n.Style)
	}
}

// --- Disabled ---------------------------------------------------------------

// Disabled rides Style so every builder supports it without a signature
// change — the same argument the accessibility fields are on Style for.
func TestDisabledAppliesToAnyBuilder(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	for name, view := range map[string]View{
		"Button":   Button("Send", func() {}, Disabled(true)),
		"Input":    Input("", "name", func(string) {}, Disabled(true)),
		"Checkbox": Checkbox(false, func(bool) {}, Disabled(true)),
		"Text":     Text("inert", Disabled(true)),
	} {
		if n := view.Render(ctx); !n.Style.Disabled {
			t.Errorf("%s did not carry Disabled", name)
		}
	}
}

// The handler must stay registered: a nil func in the registry panics when a
// native tap races the patch that disabled the control, so disabling is the
// renderer's job, never a matter of dropping the callback.
func TestDisabledButtonStillRegistersItsHandler(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := Button("Send", func() {}, Disabled(true)).Render(ctx)

	if id, ok := n.Props["onClick"].(string); !ok || id == "" {
		t.Fatalf("disabled button dropped its callback: %#v", n.Props)
	}
}

// Disabled(false) has to be able to force a node back to enabled, which is why
// the prop takes a value instead of being a no-arg flag like
// AccessibilityHidden: UseStyle's "a zero value means unset" rule leaves no
// other way to clear it.
func TestDisabledFalseClearsAnInheritedFlag(t *testing.T) {
	var s Style
	UseStyle(Style{Disabled: true}).Apply(&s)
	if !s.Disabled {
		t.Fatal("UseStyle did not merge Disabled")
	}

	UseStyle(Style{Disabled: false}).Apply(&s)
	if !s.Disabled {
		t.Fatal("a false in a Style value cleared Disabled; it must be ignored as unset")
	}

	Disabled(false).Apply(&s)
	if s.Disabled {
		t.Fatal("Disabled(false) did not clear the flag")
	}
}
