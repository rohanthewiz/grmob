package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The variant is spent on the edges, never on the fill: the strip keeps the
// theme's Surface and the primary ink whatever it is saying, which is what
// makes its contrast independent of the role.
func TestBannerVariantTintsTheEdgesNotTheFill(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	theme := core.DefaultTheme

	n := Banner{Text: "Could not refresh.", Variant: VariantError}.Render(ctx)

	if n.Style.Background != theme.Colors.Surface {
		t.Errorf("background = %q, want the theme Surface %q — an error banner must not fill with red",
			n.Style.Background, theme.Colors.Surface)
	}
	if n.Style.BorderColor != theme.Colors.Error {
		t.Errorf("border = %q, want the Error role %q", n.Style.BorderColor, theme.Colors.Error)
	}
	if n.Style.BorderWidth != 1 {
		t.Errorf("border width = %v, want 1", n.Style.BorderWidth)
	}
	msg := findText(n, "Could not refresh.")
	if msg == nil {
		t.Fatal("no message")
	}
	if msg.Style.TextColor != theme.Colors.TextPrimary {
		t.Errorf("message ink = %q, want TextPrimary %q", msg.Style.TextColor, theme.Colors.TextPrimary)
	}
}

func TestBannerDefaultGlyphPerVariant(t *testing.T) {
	cases := []struct {
		variant Variant
		glyph   string
	}{
		{VariantDefault, "ⓘ"},
		{VariantSuccess, "✓"},
		{VariantWarning, "⚠"},
		{VariantError, "⊗"},
	}
	for _, c := range cases {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		n := Banner{Text: "note", Variant: c.variant}.Render(ctx)

		g := findText(n, c.glyph)
		if g == nil {
			t.Errorf("variant %q should default to glyph %q", c.variant, c.glyph)
			continue
		}
		if g.Style.TextColor != c.variant.Color(core.DefaultTheme) {
			t.Errorf("variant %q glyph ink = %q, want the role color", c.variant, g.Style.TextColor)
		}
		// Decoration. A reader announcing "circled times" ahead of the message
		// is noise, which is why the text has to carry the meaning.
		if !g.Style.AccessibilityHidden {
			t.Errorf("variant %q glyph should be hidden from assistive tech", c.variant)
		}
	}
}

func TestBannerGlyphOverrideAndSuppression(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	if n := (Banner{Text: "note", Glyph: "🔔"}).Render(ctx); findText(n, "🔔") == nil {
		t.Error("Glyph should replace the variant default")
	}

	n := Banner{Text: "note", NoGlyph: true}.Render(ctx)
	if findText(n, "ⓘ") != nil {
		t.Error("NoGlyph should drop the mark entirely")
	}
	// NoGlyph must win over an explicit Glyph too, or the two fields fight.
	if n2 := (Banner{Text: "note", Glyph: "🔔", NoGlyph: true}).Render(ctx); findText(n2, "🔔") != nil {
		t.Error("NoGlyph should win over an explicit Glyph")
	}
}

func TestBannerActionAndDismiss(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	retried, dismissed := false, false
	n := Banner{
		Text:        "Offline.",
		Variant:     VariantWarning,
		ActionLabel: "Retry",
		OnAction:    func() { retried = true },
		OnDismiss:   func() { dismissed = true },
	}.Render(ctx)

	retry := findButton(n, "Retry")
	if retry == nil {
		t.Fatal("no action button")
	}
	// A *default* ghost, not one tinted with the banner's variant: the strip
	// already says what it is twice, and warning-on-surface is the least
	// legible combination this package has.
	if retry.Style.TextColor != core.DefaultTheme.Colors.Primary {
		t.Errorf("action ink = %q, want the theme Primary — the action does not take the variant",
			retry.Style.TextColor)
	}
	ctx.TriggerCallback(retry.Props["onClick"].(string))

	dismiss := findButton(n, "✕")
	if dismiss == nil {
		t.Fatal("no dismiss button")
	}
	if dismiss.Style.AccessibilityLabel != "Dismiss" {
		t.Errorf("dismiss label = %q, want %q", dismiss.Style.AccessibilityLabel, "Dismiss")
	}
	ctx.TriggerCallback(dismiss.Props["onClick"].(string))

	if !retried || !dismissed {
		t.Errorf("callbacks fired: retry=%v dismiss=%v, want both", retried, dismissed)
	}
}

// An ActionLabel with no handler, or a handler with no label, is an
// incompletely configured action and draws nothing — a labelled button that
// does nothing is worse than no button.
func TestBannerHalfConfiguredActionDrawsNothing(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	if n := (Banner{Text: "x", ActionLabel: "Retry"}).Render(ctx); findButton(n, "Retry") != nil {
		t.Error("an ActionLabel with no OnAction should draw no button")
	}
	if n := (Banner{Text: "x", OnAction: func() {}}).Render(ctx); len(n.Children) != 2 {
		t.Error("an OnAction with no ActionLabel should draw no button")
	}
}

func TestBannerActionSlotWinsOverTheBuiltButton(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Banner{
		Text:        "Offline.",
		ActionLabel: "Retry",
		OnAction:    func() {},
		Action:      Button{Label: "Reconnect", OnTap: func() {}},
	}.Render(ctx)

	if findButton(n, "Reconnect") == nil {
		t.Error("Action should be rendered")
	}
	if findButton(n, "Retry") != nil {
		t.Error("Action should replace the built button, not sit beside it")
	}
}

func TestBannerContentSlotReplacesTheText(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Banner{Text: "plain", Content: core.Text("rich")}.Render(ctx)

	if findText(n, "plain") != nil {
		t.Error("Content should replace Text")
	}
	if findText(n, "rich") == nil {
		t.Error("Content should be rendered")
	}
}

// The frame is the caller's to remove — this is the documented recipe for the
// edge-to-edge strip a banner under an AppBar wants.
func TestBannerStyleCanRemoveTheFrame(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Banner{Text: "Reconnecting…", Style: []core.StyleProp{
		core.BorderWidth(0), core.BorderRadius(0),
	}}.Render(ctx)

	if n.Style.BorderWidth != 0 || n.Style.BorderRadius != 0 {
		t.Errorf("Style should override the frame, got width %v radius %v",
			n.Style.BorderWidth, n.Style.BorderRadius)
	}
}

// The message takes the row's slack, which is what pins the action and the
// dismiss to the trailing edge whether or not either is present.
func TestBannerMessageTakesTheSlack(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Banner{Text: "note"}.Render(ctx)

	middle := findFirst(n, func(n *core.Node) bool { return n.Style.FlexGrow == 1 })
	if middle == nil {
		t.Fatal("no growing middle")
	}
	if findText(middle, "note") == nil {
		t.Error("the message should be inside the growing slot")
	}
}
