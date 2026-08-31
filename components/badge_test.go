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

// The field's zero value must be a no-op: adding Variant to the struct cannot
// restyle a badge written before it existed. TestBadgeThemeDefaults above is
// the real proof (it predates the field and still passes untouched); this
// states the equivalence directly.
func TestBadgeZeroVariantMatchesThePreVariantLook(t *testing.T) {
	ctx := core.NewContext()
	before := Badge{Text: "3"}.Render(ctx)
	after := Badge{Text: "3", Variant: VariantDefault}.Render(ctx)

	if before.Style.Background != after.Style.Background ||
		before.Style.TextColor != after.Style.TextColor {
		t.Errorf("VariantDefault must be the zero value's look: %q/%q vs %q/%q",
			before.Style.Background, before.Style.TextColor,
			after.Style.Background, after.Style.TextColor)
	}
	if before.Style.Background != core.DefaultTheme.Colors.Primary {
		t.Errorf("the default fill is still theme Primary, got %q", before.Style.Background)
	}
}

func TestBadgeVariantsTakeTheirColorsFromTheTheme(t *testing.T) {
	ctx := core.NewContext()
	theme := core.DefaultTheme

	cases := []struct {
		variant Variant
		wantBg  string
	}{
		{VariantSuccess, theme.Colors.Success},
		{VariantWarning, theme.Colors.Warning},
		{VariantError, theme.Colors.Error},
	}
	for _, c := range cases {
		n := Badge{Text: "status", Variant: c.variant}.Render(ctx)
		if n.Style.Background != c.wantBg {
			t.Errorf("%s badge background = %q, want %q", c.variant, n.Style.Background, c.wantBg)
		}
		// The pill shape and caption sizing are variant-independent.
		if n.Style.BorderRadius < 100 {
			t.Errorf("%s badge lost its pill shape, radius = %v", c.variant, n.Style.BorderRadius)
		}
		if n.Style.FontSize != theme.Typography.Caption.FontSize {
			t.Errorf("%s badge font = %v, want caption %v",
				c.variant, n.Style.FontSize, theme.Typography.Caption.FontSize)
		}
	}
}

// Under DefaultTheme every status fill is light enough that the ink must go
// dark — the opposite of the Primary default. A badge rendering white-on-green
// is the regression this guards.
func TestBadgeVariantInkIsReadable(t *testing.T) {
	ctx := core.NewContext()
	for _, v := range []Variant{VariantSuccess, VariantWarning, VariantError} {
		n := Badge{Text: "status", Variant: v}.Render(ctx)
		if n.Style.TextColor != core.DefaultTheme.Colors.TextPrimary {
			t.Errorf("%s badge ink = %q, want the dark ink %q on DefaultTheme's light fills",
				v, n.Style.TextColor, core.DefaultTheme.Colors.TextPrimary)
		}
	}
}

// An explicit color is the more specific instruction and beats the variant —
// but the ink is still resolved against whatever fill actually won, so an
// override does not silently produce an unreadable pill.
func TestBadgeExplicitColorOverridesVariant(t *testing.T) {
	ctx := core.NewContext()
	n := Badge{Text: "custom", Variant: VariantSuccess, Color: "#101010"}.Render(ctx)

	if n.Style.Background != "#101010" {
		t.Errorf("explicit Color must beat Variant, got %q", n.Style.Background)
	}
	if n.Style.TextColor != core.DefaultTheme.Colors.Background {
		t.Errorf("ink = %q, want the light ink on a near-black override — the ink is "+
			"resolved against the fill that won, not against the variant's role color",
			n.Style.TextColor)
	}
}

func TestBadgeExplicitTextColorOverridesTheComputedInk(t *testing.T) {
	ctx := core.NewContext()
	n := Badge{Text: "status", Variant: VariantWarning, TextColor: "#445566"}.Render(ctx)

	if n.Style.TextColor != "#445566" {
		t.Errorf("explicit TextColor must win, got %q", n.Style.TextColor)
	}
}

func TestBadgeVariantsRetintWithTheTheme(t *testing.T) {
	ctx := core.NewContext().WithTheme(core.MaterialTheme)
	n := Badge{Text: "Paid", Variant: VariantSuccess}.Render(ctx)

	if n.Style.Background != core.MaterialTheme.Colors.Success {
		t.Errorf("background = %q, want MaterialTheme's Success %q",
			n.Style.Background, core.MaterialTheme.Colors.Success)
	}
	// Material's green is dark where the default theme's is light, so the ink
	// must have flipped along with the fill.
	if n.Style.TextColor != core.MaterialTheme.Colors.Background {
		t.Errorf("ink = %q, want the light ink %q on Material's dark green",
			n.Style.TextColor, core.MaterialTheme.Colors.Background)
	}
}
