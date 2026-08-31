package components

import (
	"math"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The variants map onto palette roles, not literals — so a theme swap
// restyles every status pill in the tree.
func TestVariantResolvesPaletteRoles(t *testing.T) {
	for _, theme := range []*core.Theme{core.DefaultTheme, core.MaterialTheme} {
		cases := []struct {
			v    Variant
			want string
		}{
			{VariantDefault, theme.Colors.Primary},
			{VariantSuccess, theme.Colors.Success},
			{VariantWarning, theme.Colors.Warning},
			{VariantError, theme.Colors.Error},
		}
		for _, c := range cases {
			if got := c.v.Color(theme); got != c.want {
				t.Errorf("Variant(%q).Color() = %q, want %q", c.v, got, c.want)
			}
		}
	}
}

// A theme predating the Success/Warning roles leaves them empty, and an empty
// background is not "the default" — it is no fill at all, i.e. an invisible
// pill. Error needs no such treatment: it is one of the palette's original
// seven, so no theme can be missing it.
func TestVariantFallsBackOnAThemeWithoutTheStatusRoles(t *testing.T) {
	legacy := &core.Theme{Colors: core.ColorPalette{
		Primary:    "#6200EE",
		Background: "#FFFFFF",
		Error:      "#B00020",
	}}

	if got := VariantSuccess.Color(legacy); got != core.FallbackSuccess {
		t.Errorf("Success on a legacy theme = %q, want the fallback %q", got, core.FallbackSuccess)
	}
	if got := VariantWarning.Color(legacy); got != core.FallbackWarning {
		t.Errorf("Warning on a legacy theme = %q, want the fallback %q", got, core.FallbackWarning)
	}
	if got := VariantError.Color(legacy); got != "#B00020" {
		t.Errorf("Error should read the palette field directly, got %q", got)
	}
}

// The reason Ink computes rather than hardcodes.
//
// Badge text is caption-sized (13px under DefaultTheme, 12 under Material) —
// "normal text" by WCAG's reckoning, so the bar is AA at 4.5:1. A naive
// implementation reusing Badge's pre-variant ink default (the theme's
// Background, i.e. white) would put Success at 2.22:1 and Warning at 2.20:1
// under DefaultTheme: a badge nobody can read, on the framework's own default
// theme, with nothing in the tree to suggest a bug.
func TestVariantInkIsLegibleOnEveryThemeAndVariant(t *testing.T) {
	const wcagAA = 4.5

	for themeName, theme := range map[string]*core.Theme{
		"DefaultTheme":  core.DefaultTheme,
		"MaterialTheme": core.MaterialTheme,
	} {
		for _, v := range []Variant{VariantSuccess, VariantWarning, VariantError} {
			bg := v.Color(theme)
			ink := v.Ink(theme, bg)

			bgLum, ok := relativeLuminance(bg)
			if !ok {
				t.Fatalf("%s/%s: unparseable background %q", themeName, v, bg)
			}
			inkLum, ok := relativeLuminance(ink)
			if !ok {
				t.Fatalf("%s/%s: unparseable ink %q", themeName, v, ink)
			}

			if ratio := contrastRatio(bgLum, inkLum); ratio < wcagAA {
				t.Errorf("%s/%s: ink %q on %q is %.2f:1, below WCAG AA %.1f:1",
					themeName, v, ink, bg, ratio, wcagAA)
			}
		}
	}
}

// The ink must be picked per color, not per variant: the correct answer flips
// direction between the bundled themes for the *same* variant, because
// DefaultTheme's Success is a light green and MaterialTheme's is a dark one.
// Any fixed per-variant pairing is wrong under one theme or the other.
func TestVariantInkFlipsDirectionBetweenThemes(t *testing.T) {
	def := VariantSuccess.Ink(core.DefaultTheme, VariantSuccess.Color(core.DefaultTheme))
	mat := VariantSuccess.Ink(core.MaterialTheme, VariantSuccess.Color(core.MaterialTheme))

	if def != core.DefaultTheme.Colors.TextPrimary {
		t.Errorf("DefaultTheme Success ink = %q, want the dark ink %q (its green is light)",
			def, core.DefaultTheme.Colors.TextPrimary)
	}
	if mat != core.MaterialTheme.Colors.Background {
		t.Errorf("MaterialTheme Success ink = %q, want the light ink %q (its green is dark)",
			mat, core.MaterialTheme.Colors.Background)
	}
}

// VariantDefault is exempt from the contrast rule on purpose: it keeps the
// Primary/Background pairing the themes chose and Button uses, so the zero
// value stays a no-op for every badge that already exists.
//
// The exemption is observable under DefaultTheme, which is what makes this
// worth a test: on Primary (#007AFF) white is 4.02:1 and black 5.23:1, so the
// contrast rule *would* flip the ink to black if it applied here.
func TestVariantDefaultKeepsTheThemePairing(t *testing.T) {
	theme := core.DefaultTheme
	bg := VariantDefault.Color(theme)

	if got := VariantDefault.Ink(theme, bg); got != theme.Colors.Background {
		t.Errorf("VariantDefault ink = %q, want the theme's Background %q", got, theme.Colors.Background)
	}
	// Prove the exemption is doing work rather than agreeing by luck.
	if contrastInk(bg, theme.Colors.Background, theme.Colors.TextPrimary) == theme.Colors.Background {
		t.Error("the contrast rule now agrees with the theme pairing on Primary; this test " +
			"no longer proves VariantDefault is exempt")
	}
}

func TestContrastInkPicksTheHigherRatio(t *testing.T) {
	// Near-white fill: the dark candidate must win.
	if got := contrastInk("#FAFAFA", "#FFFFFF", "#000000"); got != "#000000" {
		t.Errorf("on a near-white fill, ink = %q, want #000000", got)
	}
	// Near-black fill: the light candidate must win.
	if got := contrastInk("#050505", "#FFFFFF", "#000000"); got != "#FFFFFF" {
		t.Errorf("on a near-black fill, ink = %q, want #FFFFFF", got)
	}
}

// An unparseable background must degrade to the first candidate — the theme's
// documented default ink — rather than to an arbitrary pick or an empty color.
// Colors reach the palette from user code and are never validated.
func TestContrastInkFallsBackOnUnparseableInput(t *testing.T) {
	for _, bad := range []string{"", "red", "#12", "#ZZZZZZ", "rgb(1,2,3)"} {
		if got := contrastInk(bad, "#FFFFFF", "#000000"); got != "#FFFFFF" {
			t.Errorf("contrastInk(%q) = %q, want the first candidate #FFFFFF", bad, got)
		}
	}
	// A bad *candidate* is skipped rather than returned.
	if got := contrastInk("#FFFFFF", "#FFFFFF", "nonsense"); got != "#FFFFFF" {
		t.Errorf("unparseable candidate should be skipped, got %q", got)
	}
}

func TestRelativeLuminance(t *testing.T) {
	cases := []struct {
		hex  string
		want float64
	}{
		{"#FFFFFF", 1.0},
		{"#000000", 0.0},
		{"#FFF", 1.0},        // 3-digit shorthand doubles each digit
		{"#FFFFFF00", 1.0},   // alpha parsed and ignored
		{"  #ffffff  ", 1.0}, // trimmed, case-insensitive
		{"#808080", 0.2158},  // mid grey, the value that exposes a missing
		{"#007AFF", 0.2114},  // sRGB linearization (a naive mean gives ~0.50)
	}
	for _, c := range cases {
		got, ok := relativeLuminance(c.hex)
		if !ok {
			t.Errorf("relativeLuminance(%q) failed to parse", c.hex)
			continue
		}
		if math.Abs(got-c.want) > 0.001 {
			t.Errorf("relativeLuminance(%q) = %.4f, want %.4f", c.hex, got, c.want)
		}
	}

	for _, bad := range []string{"", "#", "#1234", "#GGGGGG", "blue"} {
		if _, ok := relativeLuminance(bad); ok {
			t.Errorf("relativeLuminance(%q) should report failure", bad)
		}
	}
}
