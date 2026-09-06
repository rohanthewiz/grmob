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

// The guarantee the on-light tones exist to make: every one of them clears
// WCAG AA against its own theme's Background.
//
// This is the check that could not live in core. The tones are a palette
// decision and the palette declares them, but the arithmetic that says whether
// a declaration is any good is here — relativeLuminance and contrastRatio, the
// same WCAG 2.x implementation Variant.Ink picks with. So this package is
// where a retint of either bundled theme gets caught, and it is caught by
// number rather than by eye.
//
// 4.5:1 is the body-text floor, and body text is what these are for: an
// outlined button's label and a loud chip's caption are read, not glanced at.
// The looser 3:1 large-text allowance is deliberately not used — a chip's
// caption is 13pt in the one app that has one.
func TestBundledOnLightTonesClearWCAGAA(t *testing.T) {
	const floor = 4.5

	for name, theme := range map[string]*core.Theme{
		"DefaultTheme":  core.DefaultTheme,
		"MaterialTheme": core.MaterialTheme,
	} {
		bg, ok := relativeLuminance(theme.Colors.Background)
		if !ok {
			t.Fatalf("%s: Background %q does not parse", name, theme.Colors.Background)
		}
		for _, role := range []struct {
			what string
			tone string
		}{
			{"Primary", theme.Colors.PrimaryOnLightColor()},
			{"Success", theme.Colors.SuccessOnLightColor()},
			{"Warning", theme.Colors.WarningOnLightColor()},
			{"Error", theme.Colors.ErrorOnLightColor()},
		} {
			lum, ok := relativeLuminance(role.tone)
			if !ok {
				t.Errorf("%s: %s on-light tone %q does not parse", name, role.what, role.tone)
				continue
			}
			if r := contrastRatio(bg, lum); r < floor {
				t.Errorf("%s: %s on-light tone %q is %.2f:1 against Background %q, want at "+
					"least %.1f:1 — the tone exists precisely to clear this",
					name, role.what, role.tone, r, theme.Colors.Background, floor)
			}
		}
	}
}

// The other half of the same story: the role colours these replace mostly do
// *not* clear the floor, which is why the second value exists at all.
//
// Written as a census rather than as four assertions, because the point is the
// count. If a later retint made every role ink-weight on its own, the tones
// would be redundant and this would say so; a test that only checked the tones
// would let that pass in silence and leave eight fields with nothing to do.
func TestTheRoleColoursAreWhyTheOnLightTonesExist(t *testing.T) {
	const floor = 4.5
	failing := 0

	for _, theme := range []*core.Theme{core.DefaultTheme, core.MaterialTheme} {
		bg, _ := relativeLuminance(theme.Colors.Background)
		for _, raw := range []string{
			theme.Colors.Primary,
			theme.Colors.SuccessColor(),
			theme.Colors.WarningColor(),
			theme.Colors.Error,
		} {
			if lum, ok := relativeLuminance(raw); ok && contrastRatio(bg, lum) < floor {
				failing++
			}
		}
	}

	if failing == 0 {
		t.Error("every bundled role colour now clears AA as ink on its own Background — " +
			"the on-light tones have nothing left to fix, and eight palette fields plus " +
			"their plumbing should be reconsidered rather than left standing")
	}
}

// --- The field frame -------------------------------------------------------

// The two text-field bases each state a border, and it clears WCAG 1.4.11's
// 3:1 floor for a control boundary against both the page behind the field and
// the field's own fill.
//
// It lives beside the on-light census for the same reason that one is here
// rather than in core: the palette declares the tone and this package owns the
// arithmetic. What is different is the floor. The on-light tones are *ink* and
// take AA's 4.5:1 body-text floor; this is a *boundary* — Non-text Contrast,
// where 3:1 is the whole requirement, because the question is only whether a
// reader can see that a control is there.
//
// Both backdrops are checked because a field has two. Under DefaultTheme the
// fill and the page are the same white and the border is the only thing
// drawing the control; under MaterialTheme the fill is a shade off the page,
// so the outer edge and the inner edge sit on different colours.
//
// The frame has to exist at all, and that is asserted rather than assumed: a
// theme with no Input border would sail through a contrast loop with nothing
// in it, and the web target would then reset the browser's border and draw
// none of its own — the "levelling down" borderResetTypes was held back for.
func TestBundledFieldFramesClearNonTextContrast(t *testing.T) {
	const floor = 3.0

	for name, theme := range map[string]*core.Theme{
		"DefaultTheme":  core.DefaultTheme,
		"MaterialTheme": core.MaterialTheme,
	} {
		for _, base := range []struct {
			what  string
			style core.Style
		}{
			{"Input", theme.Components.Input},
			{"TextArea", theme.Components.TextArea},
		} {
			if base.style.BorderWidth == 0 || base.style.BorderColor == "" {
				t.Errorf("%s: Components.%s states no frame (%v/%q) — every renderer guards "+
					"the border on both halves, so the field goes out unmarked on all four "+
					"targets", name, base.what, base.style.BorderWidth, base.style.BorderColor)
				continue
			}
			edge, ok := relativeLuminance(base.style.BorderColor)
			if !ok {
				t.Errorf("%s: Components.%s border %q does not parse",
					name, base.what, base.style.BorderColor)
				continue
			}
			for _, backdrop := range []struct {
				what string
				hex  string
			}{
				{"Background", theme.Colors.Background},
				{"its own fill", base.style.Background},
			} {
				lum, ok := relativeLuminance(backdrop.hex)
				if !ok {
					t.Errorf("%s: Components.%s backdrop %s = %q does not parse",
						name, base.what, backdrop.what, backdrop.hex)
					continue
				}
				if r := contrastRatio(edge, lum); r < floor {
					t.Errorf("%s: Components.%s border %q is %.2f:1 against %s (%q), want at "+
						"least %.1f:1 — WCAG 1.4.11 puts that floor under the boundary that "+
						"identifies a control", name, base.what, base.style.BorderColor, r,
						backdrop.what, backdrop.hex, floor)
				}
			}
		}
	}
}

// The other half of the same story: the palette's Border role does *not* clear
// that floor on either bundled theme, which is why the field frames are not
// spent out of it.
//
// A census rather than two assertions, and pointed the same way as the on-light
// one. Border is a divider — a rule between list rows, the outline of a card —
// and a pale rule is a legitimate choice there; the fault would be reaching for
// it as a control boundary. If a later retint darkened it past 3:1 the two jobs
// would collapse back into one hex and the split ColorPalette.Border documents
// would be worth reconsidering, which nothing would otherwise say.
func TestTheDividerRoleIsWhyTheFieldFrameIsSeparate(t *testing.T) {
	const floor = 3.0
	failing := 0

	for _, theme := range []*core.Theme{core.DefaultTheme, core.MaterialTheme} {
		bg, _ := relativeLuminance(theme.Colors.Background)
		if lum, ok := relativeLuminance(theme.Colors.BorderColor()); ok &&
			contrastRatio(bg, lum) < floor {
			failing++
		}
	}

	if failing == 0 {
		t.Error("both bundled Border roles now clear 3:1 as a control boundary — the field " +
			"frames could read the palette role instead of stating their own hex, and " +
			"ColorPalette.Border's divider/boundary split should be revisited")
	}
}
