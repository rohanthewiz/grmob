package core

import (
	"reflect"
	"strings"
	"testing"
)

// Every bundled theme must set every role in ColorPalette.
//
// This is written reflectively, for the same reason core/style_merge_test.go
// is: the failure mode being guarded against is *growth*. A role added to the
// struct and wired into DefaultTheme but forgotten in MaterialTheme produces
// no compile error and no runtime error — a struct literal is happy to leave a
// field zero — it just renders that one role as an empty color under that one
// theme. A hand-written list of role names would have the same hole the struct
// grew past. Walking the type by reflection means a new role fails here by
// name until every bundled theme carries it.
func TestBundledThemesSetEveryColorRole(t *testing.T) {
	themes := map[string]*Theme{
		"DefaultTheme":  DefaultTheme,
		"MaterialTheme": MaterialTheme,
	}

	paletteType := reflect.TypeOf(ColorPalette{})
	for themeName, theme := range themes {
		palette := reflect.ValueOf(theme.Colors)
		for i := 0; i < paletteType.NumField(); i++ {
			field := paletteType.Field(i)
			got := palette.Field(i).String()
			if got == "" {
				t.Errorf("%s.Colors.%s is empty: a bundled theme must define every "+
					"palette role, or widgets reading that role render no color at all",
					themeName, field.Name)
				continue
			}
			// Catch a typo'd literal too — every value here is a hex color,
			// either #RRGGBB or #RRGGBBAA (TextSecondary uses the alpha form).
			if !strings.HasPrefix(got, "#") || (len(got) != 7 && len(got) != 9) {
				t.Errorf("%s.Colors.%s = %q, want a #RRGGBB or #RRGGBBAA hex color",
					themeName, field.Name, got)
			}
		}
	}
}

// The fallback constants are documented as "DefaultTheme's own values", so a
// theme omitting a role looks like the default theme in that one place. If
// DefaultTheme's palette is retinted and the constants are not, that promise
// quietly stops holding — and only under a *third-party* theme, which is
// exactly where nobody looks.
func TestFallbacksTrackDefaultTheme(t *testing.T) {
	cases := []struct {
		role     string
		fallback string
		themed   string
	}{
		{"Border", FallbackBorder, DefaultTheme.Colors.Border},
		{"Success", FallbackSuccess, DefaultTheme.Colors.Success},
		{"Warning", FallbackWarning, DefaultTheme.Colors.Warning},
	}
	for _, c := range cases {
		if c.fallback != c.themed {
			t.Errorf("Fallback%s = %q but DefaultTheme.Colors.%s = %q — the two must "+
				"stay in step, or a theme that omits %s stops matching the default",
				c.role, c.fallback, c.role, c.themed, c.role)
		}
	}
}

// A theme written before these roles existed leaves them empty. Empty is not
// "use the default", it is *no color*: an invisible hairline, a transparent
// status chip. The resolvers are what turn that into a visible default.
func TestResolversFallBackOnAPaletteMissingTheNewRoles(t *testing.T) {
	// Deliberately shaped like a pre-existing custom theme: the seven
	// original roles, nothing else. examples/fintechapp had exactly this
	// until this change.
	legacy := ColorPalette{
		Primary:       "#6200EE",
		Secondary:     "#03DAC6",
		Background:    "#FFFFFF",
		Surface:       "#F5F5F5",
		TextPrimary:   "#000000",
		TextSecondary: "#666666",
		Error:         "#B00020",
	}

	if got := legacy.BorderColor(); got != FallbackBorder {
		t.Errorf("BorderColor() = %q, want the fallback %q", got, FallbackBorder)
	}
	if got := legacy.SuccessColor(); got != FallbackSuccess {
		t.Errorf("SuccessColor() = %q, want the fallback %q", got, FallbackSuccess)
	}
	if got := legacy.WarningColor(); got != FallbackWarning {
		t.Errorf("WarningColor() = %q, want the fallback %q", got, FallbackWarning)
	}
}

// The other half of the contract: a theme that *does* set a role must win over
// the fallback, or theming these three roles would be a no-op.
func TestResolversPreferTheThemedValue(t *testing.T) {
	themed := ColorPalette{
		Border:  "#111111",
		Success: "#222222",
		Warning: "#333333",
	}

	if got := themed.BorderColor(); got != "#111111" {
		t.Errorf("BorderColor() = %q, want the themed value", got)
	}
	if got := themed.SuccessColor(); got != "#222222" {
		t.Errorf("SuccessColor() = %q, want the themed value", got)
	}
	if got := themed.WarningColor(); got != "#333333" {
		t.Errorf("WarningColor() = %q, want the themed value", got)
	}

	// MaterialTheme is the live proof that a theme can diverge from the
	// fallbacks — if it ever stopped, the test above would pass vacuously.
	if MaterialTheme.Colors.BorderColor() == FallbackBorder {
		t.Error("MaterialTheme.Border matches the fallback; this test no longer proves " +
			"the themed value is preferred")
	}
}

// Success and Secondary are separate roles that DefaultTheme happens to tint
// the same green. The risk is someone "simplifying" Success away to Secondary
// — which is wrong under MaterialTheme, where Secondary is a teal brand color
// and a teal "saved" badge is a bug.
func TestSuccessIsNotAnAliasForSecondary(t *testing.T) {
	if MaterialTheme.Colors.SuccessColor() == MaterialTheme.Colors.Secondary {
		t.Error("MaterialTheme's Success must not be its Secondary: Secondary is a brand " +
			"slot, Success carries meaning")
	}
}
