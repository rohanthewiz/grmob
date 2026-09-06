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
		{"ControlBorder", FallbackControlBorder, DefaultTheme.Colors.ControlBorder},
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
	if got := legacy.ControlBorderColor(); got != FallbackControlBorder {
		t.Errorf("ControlBorderColor() = %q, want the fallback %q", got, FallbackControlBorder)
	}
}

// The boundary must not degrade into the divider.
//
// ControlBorder and Border are neighbours in the struct and opposites in
// intent, and the tempting implementation of the resolver — fall back to
// BorderColor(), since a theme that has one probably meant the other — returns
// exactly the 1.26:1 hairline the roles were split apart to stop a control
// from being drawn in. A legacy palette with a *stated* Border is the case
// that would hide it: the fallback would look like it was working.
func TestTheControlBoundaryDoesNotFallBackToTheDivider(t *testing.T) {
	dividerOnly := ColorPalette{Border: "#E5E5EA"}

	got := dividerOnly.ControlBorderColor()
	if got == dividerOnly.BorderColor() {
		t.Errorf("ControlBorderColor() = %q, the same as BorderColor() — a control drawn "+
			"in the divider role is the bug this role exists to fix", got)
	}
	if got != FallbackControlBorder {
		t.Errorf("ControlBorderColor() = %q, want the boundary fallback %q",
			got, FallbackControlBorder)
	}
}

// The other half of the contract: a theme that *does* set a role must win over
// the fallback, or theming these three roles would be a no-op.
func TestResolversPreferTheThemedValue(t *testing.T) {
	themed := ColorPalette{
		Border:        "#111111",
		Success:       "#222222",
		Warning:       "#333333",
		ControlBorder: "#444444",
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
	if got := themed.ControlBorderColor(); got != "#444444" {
		t.Errorf("ControlBorderColor() = %q, want the themed value", got)
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

// A theme installed after a scope's first render must reach inside it.
//
// This is the shape an app whose palette arrives from the network has: the
// first frame renders with the default theme, the config lands, and every
// later frame is themed. Before Scope re-inherited theme and config, the
// cached child kept the theme it was built with, so everything inside a
// navigation frame (which is a scope by construction) stayed unthemed forever
// while everything outside it repainted.
func TestScopeReInheritsALaterTheme(t *testing.T) {
	root := NewContext()
	branded := &Theme{Colors: ColorPalette{Primary: "#1B5E20"}}

	// First render: no theme installed, so the scope is built with none and
	// falls back to the default.
	first := root.Scope("frame")
	if got := first.Theme().Colors.Primary; got != DefaultTheme.Colors.Primary {
		t.Fatalf("unthemed scope primary = %q, want the default", got)
	}

	// Second render, now under a themed parent — as core.WithTheme produces.
	themed := root.WithTheme(branded)
	second := themed.Scope("frame")

	if second != first {
		t.Fatal("Scope returned a different context, losing the subtree's hook slots")
	}
	if got := second.Theme().Colors.Primary; got != "#1B5E20" {
		t.Errorf("scope primary = %q, want the theme installed since it was created", got)
	}
}

// The hook slots a scope owns are its own and must survive the re-inheritance
// above — that is the whole reason a scope exists.
func TestScopeKeepsItsHookSlotsAcrossAThemeChange(t *testing.T) {
	root := NewContext()

	scope := root.Scope("frame")
	counter := NewState(scope, 7)
	counter.Set(42)

	themed := root.WithTheme(&Theme{Colors: ColorPalette{Primary: "#1B5E20"}})
	again := themed.Scope("frame")
	again.Reset() // what a render pass does before re-reading the slots

	restored := NewState(again, 7)
	if got := restored.Get(); got != 42 {
		t.Errorf("scope state = %d after a theme change, want the 42 it held", got)
	}
}

// The on-light tones fall back to their own role rather than to a constant,
// which is what makes them a no-op for a theme written before they existed:
// every widget spending one lands on exactly the colour it spent before.
//
// Each role is checked with the *other* three set, so a resolver that read the
// wrong field would fail here rather than pass by coincidence.
func TestOnLightTonesFallBackToTheirOwnRole(t *testing.T) {
	full := ColorPalette{
		Primary: "#111111", Error: "#222222", Success: "#333333", Warning: "#444444",
		PrimaryOnLight: "#AAAAAA", ErrorOnLight: "#BBBBBB",
		SuccessOnLight: "#CCCCCC", WarningOnLight: "#DDDDDD",
	}

	for _, c := range []struct {
		role  string
		tone  func(ColorPalette) string
		clear func(*ColorPalette)
		want  string
	}{
		{"Primary", ColorPalette.PrimaryOnLightColor, func(p *ColorPalette) { p.PrimaryOnLight = "" }, "#111111"},
		{"Error", ColorPalette.ErrorOnLightColor, func(p *ColorPalette) { p.ErrorOnLight = "" }, "#222222"},
		{"Success", ColorPalette.SuccessOnLightColor, func(p *ColorPalette) { p.SuccessOnLight = "" }, "#333333"},
		{"Warning", ColorPalette.WarningOnLightColor, func(p *ColorPalette) { p.WarningOnLight = "" }, "#444444"},
	} {
		if got := c.tone(full); got == c.want {
			t.Errorf("%sOnLightColor read the role while the tone was set", c.role)
		}
		bare := full
		c.clear(&bare)
		if got := c.tone(bare); got != c.want {
			t.Errorf("%sOnLightColor with no tone = %q, want the role's own %q — a theme "+
				"that predates the field must render as it always did", c.role, got, c.want)
		}
	}
}

// A theme missing both halves of Success or Warning still lands somewhere
// visible: the on-light resolver defers to the role's *resolver*, not to the
// raw field, so the documented fallback colour comes through rather than an
// empty string. Primary and Error need no such step — they are two of the
// original seven and no theme can be missing them.
func TestOnLightTonesInheritTheRoleFallbacks(t *testing.T) {
	empty := ColorPalette{}
	if got := empty.SuccessOnLightColor(); got != FallbackSuccess {
		t.Errorf("SuccessOnLightColor on an empty palette = %q, want %q", got, FallbackSuccess)
	}
	if got := empty.WarningOnLightColor(); got != FallbackWarning {
		t.Errorf("WarningOnLightColor on an empty palette = %q, want %q", got, FallbackWarning)
	}
}

// OnLight is the reverse lookup, for a widget holding a colour rather than a
// role — components.Chip's accent, read off the theme's Button base.
func TestOnLightResolvesAColourToItsRolesTone(t *testing.T) {
	p := DefaultTheme.Colors

	for _, c := range []struct{ from, want string }{
		{p.Primary, p.PrimaryOnLightColor()},
		{p.Error, p.ErrorOnLightColor()},
		{p.Success, p.SuccessOnLightColor()},
		{p.Warning, p.WarningOnLightColor()},
	} {
		if got := p.OnLight(c.from); got != c.want {
			t.Errorf("OnLight(%q) = %q, want %q", c.from, got, c.want)
		}
	}

	// Case-insensitively, because a hand-written theme may spell either way
	// and every renderer treats the two as one colour.
	//
	// Read off the palette and lower-cased rather than written as a literal,
	// and Error rather than Primary: Primary's tone is now the role itself
	// (see the theme), so a case-sensitive implementation would answer that
	// lookup with the *input* — a different string, so still a failure, but
	// one that no longer shows a wrong colour. Error's tone is a genuinely
	// different hex, so this checks the miss the way a widget would feel it.
	lower := strings.ToLower(p.Error)
	if got := p.OnLight(lower); got != p.ErrorOnLightColor() {
		t.Errorf("OnLight(%q) = %q, want the lookup to ignore hex case and return %q",
			lower, got, p.ErrorOnLightColor())
	}

	// A colour that is not one of the four toned roles comes back unchanged.
	// That is the honest answer for a reverse lookup and it is also what makes
	// the function safe to call unconditionally: a theme whose Button base is
	// some fifth colour keeps it rather than being snapped to a role it never
	// named.
	for _, other := range []string{"#8E44AD", p.Surface, p.TextSecondary, ""} {
		if got := p.OnLight(other); got != other {
			t.Errorf("OnLight(%q) = %q, want it unchanged", other, got)
		}
	}
}

// Secondary is deliberately not one of the toned roles, and DefaultTheme is
// the fixture that proves it matters: it paints Secondary and Success the same
// green. A lookup that consulted Secondary would answer for a brand slot with
// a status role's tone.
func TestOnLightDoesNotTintTheBrandSlot(t *testing.T) {
	p := DefaultTheme.Colors
	if p.Secondary != p.Success {
		t.Skipf("fixture assumed DefaultTheme paints Secondary and Success alike; "+
			"they are now %q and %q", p.Secondary, p.Success)
	}
	// The shared hex resolves through Success, which is the documented
	// first-match rule — what is pinned is that Secondary has no tone of its
	// own to disagree with it.
	if got := p.OnLight(p.Secondary); got != p.SuccessOnLightColor() {
		t.Errorf("OnLight(Secondary) = %q, want the Success tone %q it shares a hex with",
			got, p.SuccessOnLightColor())
	}
}

// Each bundled theme's field frames are its ControlBorder role.
//
// Components.Input and Components.TextArea state their border as a literal
// hex, and they have to: a component default is a Style *value*, so it cannot
// call ControlBorderColor() the way a widget does. The role and the two
// literals are therefore three copies of one decision that the type system
// cannot hold together, and this is what holds them instead.
//
// The failure being guarded is a retint. Somebody darkens a theme's field
// frames, leaves the role alone, and every chip in that theme keeps the old
// edge while every text field beside it moves — two controls on one screen
// disagreeing about where a boundary sits, with nothing failing anywhere.
func TestBundledFieldFramesAreTheControlBorderRole(t *testing.T) {
	for themeName, theme := range map[string]*Theme{
		"DefaultTheme":  DefaultTheme,
		"MaterialTheme": MaterialTheme,
	} {
		role := theme.Colors.ControlBorderColor()
		for _, base := range []struct {
			what  string
			style Style
		}{
			{"Input", theme.Components.Input},
			{"TextArea", theme.Components.TextArea},
		} {
			if base.style.BorderColor != role {
				t.Errorf("%s.Components.%s.BorderColor = %q but Colors.ControlBorder is %q — "+
					"the frame and the role are the same decision and must not drift",
					themeName, base.what, base.style.BorderColor, role)
			}
		}
	}
}

// Each bundled theme's Button base is filled with its own Primary role.
//
// Components.Button is the one place a palette states a fill and an ink
// together, and components.declaredInk reads that pair back as "the ink for
// this fill" — so every widget that paints Colors.Primary (Badge, Avatar,
// ProgressBar, Calendar's selected day, Chip's accent) gets its label colour
// from this base, and gets it only while the two hexes match.
//
// The failure being guarded is a half-move. Somebody darkens the button
// because its label is illegible and leaves Primary where it was: the button
// is fixed, the pair is broken, and every *other* Primary fill silently falls
// through to measurement — which on a mid-tone blue picks black, the exact
// disagreement inkOn was written to end. Nothing about that fails to compile
// and nothing about it looks wrong until two widgets are on screen together.
//
// It is a literal against a literal, like the field-frame pin below, because a
// Style is a value and a component default cannot call anything.
func TestBundledButtonFillsAreThePrimaryRole(t *testing.T) {
	for themeName, theme := range map[string]*Theme{
		"DefaultTheme":  DefaultTheme,
		"MaterialTheme": MaterialTheme,
	} {
		fill := theme.Components.Button.Background
		if !strings.EqualFold(fill, theme.Colors.Primary) {
			t.Errorf("%s.Components.Button.Background = %q but Colors.Primary is %q — "+
				"the declared pair only reaches the role's other spenders while the two "+
				"are one colour", themeName, fill, theme.Colors.Primary)
		}
		if theme.Components.Button.TextColor == "" {
			t.Errorf("%s.Components.Button states a fill and no ink, which is half a "+
				"declaration: declaredInk reads both halves or neither", themeName)
		}
	}
}

// The two border roles must stay different colours in every bundled theme.
//
// They were one role until a second kind of consumer arrived, and the whole of
// the split is that a divider may be pale and a boundary may not. A theme that
// tints them the same has un-split them — silently, since every call site still
// compiles and every widget still draws a rule.
func TestTheDividerAndTheBoundaryAreDifferentTones(t *testing.T) {
	for themeName, theme := range map[string]*Theme{
		"DefaultTheme":  DefaultTheme,
		"MaterialTheme": MaterialTheme,
	} {
		if theme.Colors.BorderColor() == theme.Colors.ControlBorderColor() {
			t.Errorf("%s paints Border and ControlBorder the same %q: a hairline between "+
				"rows and the edge that identifies a control carry different floors",
				themeName, theme.Colors.BorderColor())
		}
	}
}
