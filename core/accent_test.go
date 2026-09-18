package core

import "testing"

// The three platform-drawn controls carry the theme's Primary as their
// accent, and a caller's core.AccentColor wins over it. See Style.AccentColor.
func TestPlatformControlsTakeTheThemeAccent(t *testing.T) {
	ctx := NewContext()
	primary := ctx.Theme().Colors.Primary
	for name, v := range map[string]View{
		"Switch":   Switch(true, nil),
		"Checkbox": Checkbox(true, nil),
		"Slider":   Slider(0.5, 0, 1, nil),
	} {
		n := v.Render(ctx)
		if n.Style == nil || n.Style.AccentColor != primary {
			t.Errorf("%s: AccentColor = %v, want the theme's Primary %s", name, n.Style, primary)
		}
	}
}

func TestAccentColorOverridesTheTheme(t *testing.T) {
	ctx := NewContext()
	for name, v := range map[string]View{
		"Switch":   Switch(true, nil, AccentColor("#34C759")),
		"Checkbox": Checkbox(true, nil, AccentColor("#34C759")),
		"Slider":   Slider(0.5, 0, 1, nil, AccentColor("#34C759")),
	} {
		if n := v.Render(ctx); n.Style.AccentColor != "#34C759" {
			t.Errorf("%s: AccentColor = %q, want the caller's #34C759", name, n.Style.AccentColor)
		}
	}
}

// Nothing else takes an accent: a Button or a Text with one would be a
// field some target spends on a surface the others do not draw.
func TestOnlyTheControlsCarryAnAccent(t *testing.T) {
	ctx := NewContext()
	for name, v := range map[string]View{
		"Text":  Text("x"),
		"Input": Input("", "", nil),
	} {
		if n := v.Render(ctx); n.Style != nil && n.Style.AccentColor != "" {
			t.Errorf("%s carries an accent %q", name, n.Style.AccentColor)
		}
	}
}
