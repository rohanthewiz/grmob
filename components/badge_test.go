package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestBadgeThemeDefaults(t *testing.T) {
	ctx := core.NewContext()
	n := Badge{Text: "3"}.Render(ctx)

	if n.Type != "Text" || n.Props["content"] != "3" {
		t.Fatalf("badge should be a Text pill, got %q %v", n.Type, n.Props)
	}
	theme := core.DefaultTheme
	if n.Style.Background != theme.Colors.Primary {
		t.Errorf("default background = %q, want theme Primary %q", n.Style.Background, theme.Colors.Primary)
	}
	if n.Style.TextColor != theme.Colors.Background {
		t.Errorf("default ink = %q, want theme Background %q", n.Style.TextColor, theme.Colors.Background)
	}
	if n.Style.BorderRadius < 100 {
		t.Errorf("badge should be pill-shaped, radius = %v", n.Style.BorderRadius)
	}
}

func TestBadgeOverrides(t *testing.T) {
	ctx := core.NewContext()
	n := Badge{
		Text:      "beta",
		Color:     "#112233",
		TextColor: "#445566",
		Style:     []core.StyleProp{core.FontSize(9)},
	}.Render(ctx)

	if n.Style.Background != "#112233" || n.Style.TextColor != "#445566" {
		t.Errorf("explicit colors should win: %+v", n.Style)
	}
	if n.Style.FontSize != 9 {
		t.Errorf("Style props should apply after badge defaults, FontSize = %v", n.Style.FontSize)
	}
}
