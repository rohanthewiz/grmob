package components

import (
	"math"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/palette"
)

// The variants map onto palette roles, not literals — so a theme swap
// restyles every status pill in the tree.
func TestVariantResolvesPaletteRoles(t *testing.T) {
	for _, theme := range core.BundledThemes() {
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
//
// # VariantDefault is in the loop, and used to be exempt
//
// Its pairing is not this package's: the default variant's fill is the
// palette's Primary and its ink is whatever the theme's own Components.Button
// declares over it, so a failure here is a *palette* defect and the widgets
// can only report it. That is exactly why it sat outside the loop — under
// DefaultTheme it was white on systemBlue at 4.02:1, and a test that failed on
// it would have been reporting a theme decision as a widget defect while
// "fixing" it would have meant the zero value silently repainting every button
// in every tree.
//
// The palette answered instead: Primary moved to Apple's accessible blue and
// the pair is 7.56:1. So the exemption has nothing left to protect, and
// including the variant turns the bundled themes' own button colours into
// something a retint cannot quietly break.
func TestVariantInkIsLegibleOnEveryThemeAndVariant(t *testing.T) {
	const wcagAA = 4.5

	for themeName, theme := range core.BundledThemes() {
		for _, v := range []Variant{VariantDefault, VariantSuccess, VariantWarning, VariantError} {
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

// midTonePrimaryTheme is DefaultTheme as it stood before its Primary role was
// darkened: iOS systemBlue as the brand colour, white declared over it by the
// Button base, and Apple's accessible blue as the separate on-light tone.
//
// It is the fixture for the two properties a mid-tone role has and a dark one
// does not. core.AmberTheme now carries the second of them in a shipped
// palette, and the first in its weak form; what stays here alone is the
// *strong* form — a declaration that picks the opposite ink pole from the
// measurement — which no bundled theme can carry without placing its button
// label in a band 1.8% wide at the AA floor. See
// TestThePoleFlipBandIsTooNarrowToShip for that arithmetic.
//
//	the declaration splits from the measurement, at the poles
//	    white on #007AFF is 4.02:1 and black is 5.23:1, so declaredInk and
//	    contrastInk give different answers *and* the two answers are the
//	    theme's own two ink roles. That is the pairing inkOn was written for,
//	    and it is unshippable in a bundled palette: the declared ink here is
//	    below AA, and every legal version of the same disagreement sits between
//	    4.50:1 and 4.58:1.
//
//	the role splits from its on-light tone
//	    #007AFF cannot be read as ink on white and #0040DD can, which is the
//	    whole reason the tones exist. core.AmberTheme now shows this too, so
//	    this half of the fixture is a second witness rather than the only one.
//
// The blue is not invented for the test — it is the framework's own former
// default, kept as a fixture precisely because it stopped being the default.
//
// The first property is asserted from the outside by
// TestTheFixtureStillCarriesWhatNoBundledThemeCan, so an edit here that looked
// like tidying a helper — rounding the blue, dropping the separate on-light
// tone — fails rather than leaving that rule with no evidence in the repository
// at all.
func midTonePrimaryTheme() *core.Theme {
	return &core.Theme{
		Colors: core.ColorPalette{
			Primary:        "#007AFF", // iOS systemBlue: white 4.02:1, black 5.23:1
			PrimaryOnLight: "#0040DD", // Apple accessible blue — 7.56:1 on white
			Background:     "#FFFFFF",
			Surface:        "#F2F2F7",
			TextPrimary:    "#000000",
			Error:          "#FF3B30",
			ErrorOnLight:   "#D70015",
		},
		Components: core.ComponentDefaults{
			Button: core.Style{Background: "#007AFF", TextColor: "#FFFFFF"},
		},
	}
}

// The default variant paints whatever pairing the theme declared over its own
// Primary fill, so the zero value stays a no-op for every badge that already
// exists — and it reaches that by reading the declaration rather than by
// exempting itself from the rule.
//
// # Why the assertion is the declaration and not the Background
//
// It used to be "the ink is the theme's Colors.Background", which was true of
// both palettes at the time and true for a reason neither of them states:
// DefaultTheme and MaterialTheme each spend white on both, so the declared ink
// and the page colour are one hex and there was no way to tell which one this
// test was actually checking. AmberTheme is what separates them — its button
// label is MD brown 900 over amber and its page is white — so the assertion has
// to name the thing the rule is about.
//
// The two spellings are kept apart below rather than merged: the general rule
// holds for every bundled theme, and "this theme's declaration happens to be
// its Background" is a fact about two of the three that is worth stating where
// somebody can see it stop being universal.
func TestVariantDefaultKeepsTheThemePairing(t *testing.T) {
	for name, theme := range core.BundledThemes() {
		bg := VariantDefault.Color(theme)
		want := declaredInk(theme, bg)
		if want == "" {
			t.Errorf("%s: Components.Button declares no ink over the Primary fill %q, so "+
				"the default variant falls through to measurement — see "+
				"TestBundledButtonFillsAreThePrimaryRole", name, bg)
			continue
		}
		if got := VariantDefault.Ink(theme, bg); got != want {
			t.Errorf("%s: VariantDefault ink = %q, want the theme's declared %q",
				name, got, want)
		}
	}

	// The two palettes whose declaration *is* their Background, named. A third
	// theme that joined them would not be a failure; a fourth palette written
	// on the assumption that this is what "the default variant" means would be,
	// and the comment above is the only place that assumption is contradicted.
	for _, name := range []string{"DefaultTheme", "MaterialTheme"} {
		theme := core.BundledThemes()[name]
		if theme.Components.Button.TextColor != theme.Colors.Background {
			t.Errorf("%s no longer states its Background as the button's ink (%q vs %q) — "+
				"that is fine, and the comment above saying two of the bundled themes do "+
				"is now wrong", name, theme.Components.Button.TextColor,
				theme.Colors.Background)
		}
	}

	// Prove the declaration is doing work rather than agreeing by luck.
	//
	// This used to read DefaultTheme, whose Primary was systemBlue and so
	// split the two rules on its own. It no longer does — both bundled themes
	// pair white with a fill dark enough that measurement picks white too —
	// which is a *palette* improvement that would have quietly cost this
	// assertion its teeth. The fixture carries the split instead, and the
	// guard below is what says the fixture still has it.
	theme := midTonePrimaryTheme()
	bg := VariantDefault.Color(theme)
	if contrastInk(bg, theme.Colors.Background, theme.Colors.TextPrimary) == theme.Colors.Background {
		t.Fatal("the fixture's fill no longer splits the two rules; this test no longer " +
			"proves the declaration is consulted")
	}
	if got := VariantDefault.Ink(theme, bg); got != theme.Colors.Background {
		t.Errorf("on a fill where measurement disagrees, VariantDefault ink = %q, want the "+
			"theme's declared %q", got, theme.Colors.Background)
	}
}

// The declaration is keyed on the *fill*, not on the variant, which is what
// makes the rule one rule. A colour the theme has paired nothing with — an
// explicit Badge.Color, a status role — is measured, and a fill that matches
// the button base is read even when it arrives through a variant.
func TestInkOnReadsTheThemeDeclarationAndMeasuresEverythingElse(t *testing.T) {
	theme := core.DefaultTheme

	// The declared pair, spelled in the other case: a theme is hand-written
	// and "#0040dd" is the same blue to every renderer.
	//
	// Asserted through declaredInk rather than through inkOn, because inkOn
	// can no longer see the difference under a bundled theme: measurement
	// picks white over this fill as well, so a case-blind lookup would fall
	// through to the same answer. declaredInk returns "" on a miss, which is
	// the observation the case-folding is actually about.
	lower := strings.ToLower(theme.Components.Button.Background)
	if got := declaredInk(theme, lower); got != theme.Components.Button.TextColor {
		t.Errorf("declaredInk on a lower-case Primary %q = %q, want the declared %q",
			lower, got, theme.Components.Button.TextColor)
	}
	// A pale fill nobody declared anything about. This is the case the old
	// VariantDefault arm got wrong: it returned white on pale yellow because
	// it never looked at bg at all.
	if got := VariantDefault.Ink(theme, "#FFF9C4"); got != theme.Colors.TextPrimary {
		t.Errorf("ink on an undeclared pale fill = %q, want the dark %q", got, theme.Colors.TextPrimary)
	}
	// A status fill is undeclared too, and stays measured.
	if got := inkOn(theme, theme.Colors.SuccessColor()); got != theme.Colors.TextPrimary {
		t.Errorf("ink on Success = %q, want the measured %q", got, theme.Colors.TextPrimary)
	}
}

// A theme that declares no pair has nothing to read, and must not be handed an
// empty ink. Half a declaration is not a declaration: a background with no
// text colour would otherwise return "", which paints an invisible label.
func TestDeclaredInkNeedsBothHalves(t *testing.T) {
	for name, base := range map[string]core.Style{
		"no Components.Button at all": {},
		"a fill and no ink":           {Background: "#007AFF"},
		"an ink and no fill":          {TextColor: "#FFFFFF"},
	} {
		theme := &core.Theme{
			Colors:     core.ColorPalette{Background: "#FFFFFF", TextPrimary: "#000000", Primary: "#007AFF"},
			Components: core.ComponentDefaults{Button: base},
		}
		if got := declaredInk(theme, "#007AFF"); got != "" {
			t.Errorf("%s: declaredInk = %q, want no declaration", name, got)
		}
		// And the fallback is the measurement, not the empty string.
		if got := inkOn(theme, "#007AFF"); got != "#000000" {
			t.Errorf("%s: inkOn = %q, want the measured #000000", name, got)
		}
	}
}

// A theme whose buttons are not primary-coloured has declared a pair for that
// other colour and nothing for Primary. Both halves of that are checked here,
// because reading the base as "the ink for Primary" rather than "the ink for
// the base's own fill" would pass the first and fail the second.
func TestDeclarationFollowsTheButtonFillNotTheRole(t *testing.T) {
	theme := &core.Theme{
		Colors: core.ColorPalette{
			Primary:     "#FFFFFF", // a white brand colour, deliberately absurd
			Background:  "#FFFFFF",
			TextPrimary: "#111111",
		},
		Components: core.ComponentDefaults{
			Button: core.Style{Background: "#101820", TextColor: "#F2AA4C"},
		},
	}
	if got := inkOn(theme, "#101820"); got != "#F2AA4C" {
		t.Errorf("ink on the declared button fill = %q, want %q", got, "#F2AA4C")
	}
	if got := inkOn(theme, theme.Colors.Primary); got != "#111111" {
		t.Errorf("ink on the undeclared Primary = %q, want the measured %q", got, "#111111")
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

	for name, theme := range core.BundledThemes() {
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

	for _, theme := range core.BundledThemes() {
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

	for name, theme := range core.BundledThemes() {
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
			// The field frame's own two backdrops, which are deliberately
			// not the census's: this test asks about one widget on the two
			// fills that widget lands on. palette.Backdrop is borrowed for
			// the field names alone.
			for _, backdrop := range []palette.Backdrop{
				{What: "Background", Hex: theme.Colors.Background},
				{What: "its own fill", Hex: base.style.Background},
			} {
				lum, ok := relativeLuminance(backdrop.Hex)
				if !ok {
					t.Errorf("%s: Components.%s backdrop %s = %q does not parse",
						name, base.what, backdrop.What, backdrop.Hex)
					continue
				}
				if r := contrastRatio(edge, lum); r < floor {
					t.Errorf("%s: Components.%s border %q is %.2f:1 against %s (%q), want at "+
						"least %.1f:1 — WCAG 1.4.11 puts that floor under the boundary that "+
						"identifies a control", name, base.what, base.style.BorderColor, r,
						backdrop.What, backdrop.Hex, floor)
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

	for _, theme := range core.BundledThemes() {
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

// --- The boundary census ---------------------------------------------------

// Colors.ControlBorder, measured against every fill a bundled theme actually
// paints under a control.
//
// # What this is for
//
// The field-frame test above asks whether *two named widgets* clear the floor
// on *their own* backdrops. That is the right question for those two and it
// answers nothing about the next widget, which is how the one shortfall in
// this repository came to be argued in two prose blocks and asserted nowhere:
// components.Chip drew its quiet ring in this role on a Surface fill, where
// DefaultTheme's tone measured 2.92:1 rather than 3.26:1. A second widget
// drawing a boundary on Surface would have inherited the shortfall without
// inheriting the argument — it would simply have been under the floor, with
// nothing anywhere saying so.
//
// So the census is over the *palette role and the theme's own fills* rather
// than over widgets. Every pair a widget could reach is measured whether or
// not one reaches it today, which is what turns "the chip happens to be fine"
// into "these are the pairs, and this is the one that falls short".
//
// # What the census did next
//
// It closed the shortfall it was built to record, and by an unplanned route.
// The pair had stood for three sessions because the honest fixes were both
// palette decisions, and a palette decision was expensive: to know whether a
// candidate tone was an improvement or a trade you had to go and find every
// fill a boundary lands on. That is exactly what this table is, so the cost
// fell to one test run — and #89898E, five steps darker than systemGray,
// clears every DefaultTheme backdrop. (Four of them at the time; the list is
// derived now, and the two the derivation added clear it too.) The retint is recorded on
// core.ColorPalette.ControlBorder and the argument it retired on chipRing.
//
// The table below is therefore empty, and that is its resting state rather
// than a defect. It exists so that the *next* pair under the floor has to be
// either fixed or defended in writing, and an empty exemption list is what
// "no pair is currently under the floor" looks like.
// TestKnownBoundaryShortfallsIsEmptyOrJustified is what stops it filling up
// quietly.
//
// # The backdrops
//
// Derived rather than named. Every fill a control can be drawn on is either
// one of the two palette roles — the page and the Surface panel — or a
// core.ComponentDefaults field that states a Background, and the second half
// is read off the struct by reflection in internal/palette.
//
// That package carries the argument. The short version is that the set is a
// consequence of ComponentDefaults rather than a decision, so a new field
// carrying a fill (a Sheet, a Popover) is a backdrop a hand-written list would
// silently not measure — and the census exists precisely so that an unmeasured
// pair is impossible. The two exclusions that used to be a line missing from a
// slice literal are now data with reasons attached, held to the struct by the
// test below.
func boundaryBackdrops(theme *core.Theme) []palette.Backdrop {
	return palette.Backdrops(theme)
}

// Every exclusion names a real ComponentDefaults field that really carries a
// fill.
//
// Both halves matter and they fail differently. An exclusion for a field that
// no longer exists exempts nothing — the census would measure a pair the
// exclusion's author believed was skipped, and the reason attached to it reads
// as though somebody had considered it. An exclusion for a field that states
// no Background exempts nothing either, because For skips empty fills anyway;
// such an entry is an argument about a pair that does not exist, and the next
// person to give that component a fill inherits an exemption nobody made for
// them.
func TestTheBackdropExclusionsNameRealFills(t *testing.T) {
	fields := map[string]bool{}
	for _, name := range palette.ComponentFields() {
		fields[name] = true
	}
	for name, reason := range palette.NotABackdrop {
		if reason == "" {
			t.Errorf("palette.NotABackdrop[%q] has no reason — an exclusion without "+
				"an argument is a pair that was quietly dropped", name)
		}
		if !fields[name] {
			t.Errorf("palette.NotABackdrop[%q] names no core.ComponentDefaults field. "+
				"The exclusion exempts nothing and the census is measuring the pair "+
				"anyway, or the field was renamed and the reason travelled with the "+
				"old spelling", name)
			continue
		}
		fills := false
		for _, theme := range core.BundledThemes() {
			if palette.Fill(theme, name) != "" {
				fills = true
			}
		}
		if !fills {
			t.Errorf("palette.NotABackdrop[%q] names a component that states no "+
				"Background in any bundled theme, so it was never a backdrop to "+
				"exclude. Delete the entry, or the next theme to give it a fill "+
				"inherits an exemption written for a different reason", name)
		}
	}
}

// Every fill a bundled theme states is either measured or excluded.
//
// This is the closure the derivation buys, and it is deliberately computed a
// different way from palette.Backdrops: it walks ComponentFields and asks
// palette.Fill directly, so a Backdrops that quietly stopped reflecting — an
// early return, a skipped field, a Kind check that no longer matches — leaves
// pairs unmeasured and this reports them by name.
//
// Without it the census can shrink silently. Fewer pairs is fewer assertions
// and a green run, which is the one failure mode a table that measures things
// cannot afford.
func TestEveryStatedFillIsMeasuredOrExcluded(t *testing.T) {
	for name, theme := range core.BundledThemes() {
		measured := map[string]bool{}
		for _, b := range boundaryBackdrops(theme) {
			measured[b.What] = true
		}
		for _, field := range palette.ComponentFields() {
			hex := palette.Fill(theme, field)
			if hex == "" {
				continue
			}
			_, excluded := palette.NotABackdrop[field]
			if excluded == measured[field+" fill"] {
				verb := "is measured and excluded at once"
				if !excluded {
					verb = "states " + hex + " and is neither measured nor excluded"
				}
				t.Errorf("%s: Components.%s %s — every fill a control could be drawn "+
					"on is either a pair with a number under it or an entry in "+
					"palette.NotABackdrop with an argument attached", name, field, verb)
			}
		}
		// The two palette roles are not components and would not be caught
		// above; a census that dropped them would be measuring the widgets
		// and not the page.
		for _, role := range []string{"Background", "Surface"} {
			if !measured[role] {
				t.Errorf("%s: the census no longer measures Colors.%s, which is the "+
					"fill most controls in the theme are actually drawn on", name, role)
			}
		}
	}
}

// The pairs that do not clear 3:1, stated once with the reason attached.
// Currently none — see the census comment above for why empty is the resting
// state and not a gap.
//
// Transcribed rather than derived, and that is the point: an entry here is a
// decision someone made and can defend, where a pair that merely happens to
// fall short is a bug. A new pair that falls short fails the census until
// somebody either fixes the theme or writes down why it is allowed.
//
// Keyed by theme name and backdrop, spelled exactly as boundaryBackdrops
// names them.
var knownBoundaryShortfalls = map[string]struct {
	ratio  float64 // to 2dp, the number the argument was made about
	reason string
}{}

// Every (theme, backdrop) pair either clears WCAG 1.4.11's 3:1 floor or is a
// recorded shortfall whose number has not moved.
//
// The recorded ratio is compared, not just the key. A retint that left the
// pair failing but changed how badly would slip past a bare exemption, and
// the argument in knownBoundaryShortfalls is about a specific distance from
// the floor — "the last 0.08" is a sentence that stops being true at 2.4:1.
func TestEveryControlBoundaryPairIsAccountedFor(t *testing.T) {
	const floor = 3.0

	for name, theme := range core.BundledThemes() {
		edge, ok := relativeLuminance(theme.Colors.ControlBorderColor())
		if !ok {
			t.Errorf("%s: ControlBorder %q does not parse", name, theme.Colors.ControlBorderColor())
			continue
		}
		for _, backdrop := range boundaryBackdrops(theme) {
			if backdrop.Hex == "" {
				t.Errorf("%s: %s states no fill — a control boundary drawn on it lands on "+
					"whatever is behind, which no test can measure", name, backdrop.What)
				continue
			}
			lum, ok := relativeLuminance(backdrop.Hex)
			if !ok {
				t.Errorf("%s: %s = %q does not parse", name, backdrop.What, backdrop.Hex)
				continue
			}
			r := contrastRatio(edge, lum)
			known, exempt := knownBoundaryShortfalls[name+"/"+backdrop.What]
			switch {
			case r >= floor:
				// Clears. If it is also recorded as a shortfall, the sibling
				// test below is the one that reports it.
			case !exempt:
				t.Errorf("%s: ControlBorder %q is %.2f:1 against %s (%q), want at least "+
					"%.1f:1 — WCAG 1.4.11 puts that floor under the boundary that "+
					"identifies a control. Either retint, or record it in "+
					"knownBoundaryShortfalls with the argument for why this pair is "+
					"allowed to fall short", name, theme.Colors.ControlBorderColor(),
					r, backdrop.What, backdrop.Hex, floor)
			case round2(r) != known.ratio:
				t.Errorf("%s: ControlBorder against %s is %.2f:1, recorded as %.2f:1 — the "+
					"exemption's argument was made about the recorded number (%s)",
					name, backdrop.What, r, known.ratio, known.reason)
			}
		}
	}
}

// The other direction: a recorded shortfall that has stopped being one is an
// exemption with nothing to exempt, and it must be deleted rather than left
// standing.
//
// Same shape as TestTheDividerRoleIsWhyTheFieldFrameIsSeparate one section
// up, and for the same reason. An exemption nobody revisits is how a fixed
// theme keeps carrying the paragraph explaining why it was not fixed — and
// worse, how the next real shortfall gets waved through by an entry that
// looks like precedent.
//
// This is the test that fired. The DefaultTheme/Surface entry was retired by
// the retint that closed its pair, and this loop is what would have reported
// it had the retint come from someone who had not read chipRing.
func TestNoRecordedBoundaryShortfallHasQuietlyBeenFixed(t *testing.T) {
	const floor = 3.0

	byName := core.BundledThemes()
	for key := range knownBoundaryShortfalls {
		name, what, found := strings.Cut(key, "/")
		theme, ok := byName[name]
		if !found || !ok {
			t.Errorf("knownBoundaryShortfalls[%q] names no bundled theme", key)
			continue
		}
		var hex string
		for _, b := range boundaryBackdrops(theme) {
			if b.What == what {
				hex = b.Hex
			}
		}
		if hex == "" {
			t.Errorf("knownBoundaryShortfalls[%q] names no backdrop the census measures — "+
				"a spelling the loop above never reaches exempts nothing", key)
			continue
		}
		edge, _ := relativeLuminance(theme.Colors.ControlBorderColor())
		lum, _ := relativeLuminance(hex)
		if contrastRatio(edge, lum) >= floor {
			t.Errorf("%s clears %.1f:1 now — delete the entry from "+
				"knownBoundaryShortfalls, and the argument in chipRing and "+
				"ColorPalette.ControlBorder with it", key, floor)
		}
	}
}

// What a new entry has to look like.
//
// The table is empty, which is the state it is hardest to add a bad entry to
// safely: there is no neighbour to match against, and the two tests above
// both pass vacuously over nothing. Each requirement below is a way the one
// entry this table has ever held could have been written badly.
//
//   - A reason long enough to be an argument. The retired entry's was six
//     lines and carried the whole case; "the chip" would have exempted the
//     same pair, passed both sibling tests, and told the next reader nothing.
//     The floor is arbitrary and deliberately low — it separates a sentence
//     from a label, and no more.
//   - A recorded ratio actually under 3:1. The sibling test compares the
//     recorded number against the measured one, but only for pairs the census
//     reaches; an entry recording 3.2 is exempting something that clears, and
//     is unfalsifiable rather than wrong.
//   - A pointer to where the argument lives in full. The reason here is a
//     précis by design — the retired one pointed at chipRing and
//     ColorPalette.ControlBorder — and a précis with no referent is the only
//     copy, which is the shape that drifts.
func TestKnownBoundaryShortfallsIsEmptyOrJustified(t *testing.T) {
	const floor = 3.0
	// Long enough to be a sentence. Chosen against the shortest defensible
	// argument, not against the retired one, which was six times this.
	const minReason = 60

	for key, entry := range knownBoundaryShortfalls {
		if len(entry.reason) < minReason {
			t.Errorf("knownBoundaryShortfalls[%q] gives %d characters of reason, want at "+
				"least %d — an exemption is worth what its argument is worth, and this "+
				"table is the place the argument is stated as a fact",
				key, len(entry.reason), minReason)
		}
		if entry.ratio >= floor {
			t.Errorf("knownBoundaryShortfalls[%q] records %.2f:1, which is not below the "+
				"%.1f:1 floor — an entry for a pair that clears exempts nothing and "+
				"cannot be checked by TestNoRecordedBoundaryShortfallHasQuietlyBeenFixed",
				key, entry.ratio, floor)
		}
		if !strings.Contains(entry.reason, "see ") {
			t.Errorf("knownBoundaryShortfalls[%q] points nowhere — end the reason with "+
				"\"see <identifier>\" naming where the argument is made in full, so this "+
				"précis has something to be a précis of", key)
		}
	}
}

// round2 rounds to the two decimal places the recorded ratios are written to.
// Comparing floats directly would make the table a transcription of this
// package's float64 arithmetic rather than of a number a human can check
// against a contrast tool.
func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
