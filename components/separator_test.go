package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestSeparatorDefaults(t *testing.T) {
	ctx := core.NewContext()
	n := Separator{}.Render(ctx)

	if n.Type != "Box" {
		t.Fatalf("root type = %q, want Box — the only container with no theme padding to undo", n.Type)
	}
	if len(n.Children) != 0 {
		t.Errorf("a rule has no children, got %d", len(n.Children))
	}
	if n.Style.Height != "1px" {
		t.Errorf("Height = %q, want the 1px default", n.Style.Height)
	}
	if n.Style.Background != core.DefaultTheme.Colors.Border {
		t.Errorf("Background = %q, want the default theme's Border role %q",
			n.Style.Background, core.DefaultTheme.Colors.Border)
	}
	if n.Style.Margin != (core.EdgeInsets{}) {
		t.Errorf("no Inset must mean no margin at all — core.Divider's forced Margin(8) is the "+
			"reason neither example used it; got %+v", n.Style.Margin)
	}
}

// A rule carries no information, and one between every pair of rows would turn
// a 20-row feed into 39 screen-reader utterances.
func TestSeparatorIsAlwaysHiddenFromAssistiveTech(t *testing.T) {
	ctx := core.NewContext()
	for _, s := range []Separator{
		{},
		{Color: "#000000", Thickness: 4, Inset: 16},
	} {
		if n := s.Render(ctx); !n.Style.AccessibilityHidden {
			t.Errorf("%+v: separator must be hidden from assistive tech", s)
		}
	}
}

func TestSeparatorThicknessAndInset(t *testing.T) {
	ctx := core.NewContext()
	n := Separator{Thickness: 0.5, Inset: 16, Color: "#FF0000"}.Render(ctx)

	// Fractional thickness must survive as a fraction: a half-pixel hairline
	// is a real request on a 2x display and both renderers parse a float.
	if n.Style.Height != "0.5px" {
		t.Errorf("Height = %q, want %q", n.Style.Height, "0.5px")
	}
	if n.Style.Background != "#FF0000" {
		t.Errorf("Background = %q, want the override", n.Style.Background)
	}
	// Per-side, not EdgeInsets.Horizontal: the HTML exporter reads only the
	// four sides, so Horizontal would vanish from that target.
	want := core.EdgeInsets{Left: 16, Right: 16}
	if n.Style.Margin != want {
		t.Errorf("Margin = %+v, want %+v", n.Style.Margin, want)
	}
}

func TestSeparatorStyleOverridesDefaults(t *testing.T) {
	ctx := core.NewContext()
	n := Separator{Style: []core.StyleProp{
		core.Height("2px"),
		core.BackgroundColor("#123456"),
	}}.Render(ctx)

	if n.Style.Height != "2px" || n.Style.Background != "#123456" {
		t.Errorf("caller Style must be applied last and win; got Height=%q Background=%q",
			n.Style.Height, n.Style.Background)
	}
}

// The rule is themed, not hardcoded: a theme swap must retint every separator
// in the tree without any call site changing.
func TestSeparatorTakesTintFromTheme(t *testing.T) {
	ctx := core.NewContext().WithTheme(core.MaterialTheme)
	n := Separator{}.Render(ctx)

	if n.Style.Background != core.MaterialTheme.Colors.Border {
		t.Errorf("Background = %q, want MaterialTheme's Border %q",
			n.Style.Background, core.MaterialTheme.Colors.Border)
	}
	// Guard against the test passing because both themes agree.
	if core.MaterialTheme.Colors.Border == core.DefaultTheme.Colors.Border {
		t.Error("the two bundled themes now share a Border tint; this test no longer " +
			"proves the theme is being read")
	}
}

// A custom theme predating the Border role leaves it empty, and an empty
// Background is not a default — it is no color, i.e. an invisible rule. The
// resolver is what keeps such a theme drawing a visible hairline.
func TestSeparatorFallsBackOnAThemeWithoutABorderRole(t *testing.T) {
	legacy := &core.Theme{Colors: core.ColorPalette{
		Primary:    "#6200EE",
		Background: "#FFFFFF",
		Surface:    "#F5F5F5",
	}}
	n := Separator{}.Render(core.NewContext().WithTheme(legacy))

	if n.Style.Background != core.FallbackBorder {
		t.Errorf("Background = %q, want the fallback hairline %q — a theme missing the "+
			"Border role must not render an invisible separator",
			n.Style.Background, core.FallbackBorder)
	}
}

// Color is still the per-instance escape hatch, above whatever the theme says.
func TestSeparatorColorOverridesTheTheme(t *testing.T) {
	ctx := core.NewContext().WithTheme(core.MaterialTheme)
	n := Separator{Color: "#FF0000"}.Render(ctx)

	if n.Style.Background != "#FF0000" {
		t.Errorf("Background = %q, want the explicit override", n.Style.Background)
	}
}
