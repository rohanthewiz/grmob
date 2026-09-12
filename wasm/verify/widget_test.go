package main

import (
	"math"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/palette"
)

// The widget swatches are the census's own pairs, drawn by a real widget.
//
// # What this adds to the browser pass
//
// browser.mjs mounts these trees and reads their pixels back, which settles
// that the colours the *rendered node* declares reach the screen. It has no way
// to know whether those colours are the ones comps/variant_test.go
// measured — it is handed three hexes and it compares three hexes.
//
// That is the gap this closes, and it is the one the whole exercise is about.
// A chip whose ring had drifted off core.ColorPalette.ControlBorder — a Style
// override, a fallback taken, a theme field read instead of the role — would
// render, paint, and pass every pixel comparison in the browser, because the
// browser would be comparing it against its own drifted self. The pairs have to
// be tied back to the authority in Go, which is here.
//
// So each case is held to three things it cannot check for itself: that the
// ring is the palette role, that both backdrops are fills the census measures,
// and that the ratios travelling to the browser are palette.Ratio's.

// widgetWhats are the widgets every bundled theme is expected to draw, and the
// two halves of core.ColorPalette.ControlBorder's job:
//
//	quiet Chip    the role's second spender, reading Colors.ControlBorderColor.
//	              Exports as a <button>.
//	Input frame   the first, reading Components.Input.BorderColor — a literal
//	              each theme states, pinned to the role elsewhere. Exports as
//	              an <input>, a second tag with a second user-agent rule.
//
// Named here rather than derived from widgetCases() because the census's whole
// shape is "every theme times every widget": deriving the list from the cases
// would make a widget silently dropped from gen.go into a smaller census
// rather than a failure, which is the one thing a table that measures things
// cannot afford.
var widgetWhats = []string{"quiet Chip", "Input frame"}

func widgetCaseFor(t *testing.T, theme, what string) widgetCase {
	t.Helper()
	for _, c := range widgetCases() {
		if c.Theme == theme && c.What == what {
			return c
		}
	}
	t.Fatalf("no %s swatch for %s", what, theme)
	return widgetCase{}
}

// Every bundled theme draws every widget, once, and nothing else.
//
// All three directions, for the reason palette_test.go gives about its own
// rows: a theme with no swatch is a palette whose widgets the browser has never
// looked at, a widget with no swatch is half of ControlBorder's job unpainted,
// and a swatch for a theme core does not ship is a check painting a boundary
// nothing draws.
func TestEveryBundledThemeHasAWidgetSwatch(t *testing.T) {
	got := map[string]bool{}
	for _, c := range widgetCases() {
		key := c.Theme + "/" + c.What
		if got[key] {
			t.Errorf("two %s swatches for %s — the browser pass would mount both "+
				"and a failure could not say which", c.What, c.Theme)
		}
		got[key] = true
	}
	for name := range core.BundledThemes() {
		for _, what := range widgetWhats {
			key := name + "/" + what
			if !got[key] {
				t.Errorf("no %s swatch for %s: the browser pass paints the swatch grid "+
					"for this palette and never asks whether that widget draws it",
					what, name)
			}
			delete(got, key)
		}
	}
	for key := range got {
		t.Errorf("a widget swatch for %q, which is not a bundled theme crossed with a "+
			"widget in widgetWhats — the browser is checking a boundary nothing ships "+
			"or nothing here knows how to hold to Go", key)
	}
}

// The ring a real widget draws is the palette role, not a colour that happens
// to look like it.
//
// This is the assertion the browser cannot make. comps.chipRing reads
// Colors.ControlBorderColor rather than Components.Input.BorderColor
// deliberately — the two hold the same hex in every bundled theme today, and
// tying a chip's edge to a text field's is exactly the drift that argument was
// written to prevent. Reading the tone off the rendered node and comparing it
// with the role is what keeps that argument true rather than remembered.
//
// # What it cannot catch on a bundled theme
//
// Precisely that swap. Components.Input.BorderColor holds the same hex as the
// role in all three bundled themes — core/theme_test.go's
// TestBundledFieldFramesAreTheControlBorderRole is what makes that true — so a
// chip that read the field base instead would render an identical tone and
// pass this comparison, every pixel in the browser, and every other test that
// renders a shipped palette. It is a difference in *provenance* with no
// observable consequence until somebody restyles their text fields, which is
// the day the argument in chipRing is about.
//
// A hex comparison cannot see that on a theme where the two agree, and neither
// can a screenshot. TestEachWidgetReadsTheAuthorityItNames below is what does:
// it renders the same two widgets through a theme whose role and field base are
// deliberately different hexes, which is the one arrangement where provenance
// has a pixel. This test keeps the whole of the rest, on the palettes that
// actually ship — a tone that is neither, a fallback taken, a Style override, a
// theme whose role moved and whose widget did not.
func TestTheWidgetSwatchRingIsThePaletteRole(t *testing.T) {
	// RingFrom -> the value it names, read out of the theme. A spelling not in
	// this map is a case whose authority nothing here knows, which is a
	// failure rather than a skip: the alternative is a widget added to gen.go
	// with a blank RingFrom and checked against nothing.
	authority := map[string]func(*core.Theme) string{
		ringFromRole: func(t *core.Theme) string { return t.Colors.ControlBorderColor() },
		// The field base is a literal in the theme, not a resolver call: a
		// core.Style is a value, so a component default cannot read a role.
		// core/theme_test.go's TestBundledFieldFramesAreTheControlBorderRole
		// is what keeps the two equal, and it is a separate claim from this one
		// on purpose — see the note above about provenance.
		ringFromInputBase: func(t *core.Theme) string { return t.Components.Input.BorderColor },
	}

	for name, theme := range core.BundledThemes() {
		for _, what := range widgetWhats {
			c := widgetCaseFor(t, name, what)
			read, known := authority[c.RingFrom]
			if !known {
				t.Errorf("%s/%s names %q as the source of its boundary tone, which is "+
					"not a spelling this test can read. A case whose authority is "+
					"unknown is checked against nothing", name, what, c.RingFrom)
				continue
			}
			if want := read(theme); c.Ring != want {
				t.Errorf("%s/%s: the rendered widget's boundary is %q and %s is %q. The "+
					"browser pass compares the pixel with the first of those, so it "+
					"would go on passing while the widget drew a tone the contrast "+
					"census has never measured", name, what, c.Ring, c.RingFrom, want)
			}
		}
	}
}

// Both of a ring's backdrops are fills the census measures.
//
// A boundary is only as good as the pair, and internal/palette derives the
// list of fills a boundary can land on from core.ComponentDefaults precisely so
// that a new one cannot go unmeasured. A widget drawn on a fill outside that
// list would be a control whose 3:1 floor nobody had ever computed — and it
// would paint perfectly, because painting is not the question.
func TestTheWidgetSwatchBackdropsAreMeasured(t *testing.T) {
	for name, theme := range core.BundledThemes() {
		measured := map[string]bool{}
		for _, b := range palette.Backdrops(theme) {
			measured[b.Hex] = true
		}
		for _, what := range widgetWhats {
			c := widgetCaseFor(t, name, what)
			for _, pair := range []struct{ what, hex string }{
				{"the page behind the widget", c.Page},
				{"the widget's own fill", c.Fill},
			} {
				if pair.hex == "" {
					t.Errorf("%s/%s: %s is unstated — the rendered node declares no "+
						"colour there, so the browser is sampling a pixel against "+
						"nothing", name, what, pair.what)
					continue
				}
				if !measured[pair.hex] {
					t.Errorf("%s/%s: %s is %s, which is not one of the fills "+
						"internal/palette derives from core.ComponentDefaults. The ring "+
						"is drawn on it and its 3:1 floor has never been computed",
						name, what, pair.what, pair.hex)
				}
			}
		}
	}
}

// The ratios that travel to the browser are the census's arithmetic.
//
// Nothing in the browser recomputes them — a second WCAG implementation is the
// last thing a contrast floor needs, which is palette.mjs's own argument — so
// they are only worth carrying if they are right. They appear in every failure
// message the widget check can print, which is the whole reason a wrong one is
// worse than none: it would name a number nobody computed as the thing the
// pixel was supposed to be about.
func TestTheWidgetSwatchRatiosAreThePaletteArithmetic(t *testing.T) {
	for name := range core.BundledThemes() {
		for _, what := range widgetWhats {
			c := widgetCaseFor(t, name, what)
			for _, pair := range []struct {
				what, a, b string
				got        float64
			}{
				{"against the page", c.Ring, c.Page, c.RatioOnPage},
				{"against the fill", c.Ring, c.Fill, c.RatioOnFill},
			} {
				la, oka := palette.Luminance(pair.a)
				lb, okb := palette.Luminance(pair.b)
				if !oka || !okb {
					t.Errorf("%s/%s: %s compares colours that do not parse (%q, %q)",
						name, what, pair.what, pair.a, pair.b)
					continue
				}
				want := math.Round(palette.Ratio(la, lb)*100) / 100
				if pair.got != want {
					t.Errorf("%s/%s: the swatch carries %.2f:1 %s and palette.Ratio says "+
						"%.2f:1", name, what, pair.got, pair.what, want)
				}
			}
		}
	}
}

// Each widget reads the authority its case names, told apart by a theme that
// splits the two.
//
// # The gap this closes, and why it needed a made-up theme
//
// TestTheWidgetSwatchRingIsThePaletteRole above states what it cannot see, and
// this is it: Components.Input.BorderColor holds the same hex as
// Colors.ControlBorder in all three bundled themes (core/theme_test.go's
// TestBundledFieldFramesAreTheControlBorderRole is what makes that true), so
// the comparison passes whichever of the two a case names. A chip that had
// started reading the field base — the exact drift comps.chipRing's
// argument was written to prevent — renders an identical tone, paints an
// identical pixel, and satisfies every check in this file and in browser.mjs.
// The failure only becomes visible on the day somebody restyles their fields,
// which is the day the argument is about.
//
// Both directions are invisible, not one. A field frame that had started
// reading the role would be equally undetectable, and it is the likelier of the
// two: "the role is the thing both of them name" reads like an instruction to
// use it everywhere.
//
// A fourth bundled theme that split them would sharpen the whole family, and it
// would be a palette added to the framework for a test. This does the same work
// with a theme that never leaves this function: the two values are set to two
// hexes nothing else in the repository uses, both widgets are rendered through
// it by gen.go's own builders, and each case's ring must be the value its
// RingFrom names.
//
// # Why it renders rather than reading widgetCases()
//
// widgetCases() is bundled themes only, and every bundled theme is one where
// the two agree. Rendering through gen.go's widgetBuilders is what makes this a
// statement about the widgets the browser pass actually mounts rather than
// about a second pair built here.
func TestEachWidgetReadsTheAuthorityItNames(t *testing.T) {
	// Two hexes chosen only to be different from each other and from anything
	// a theme ships, so a ring that matched one of them matched it by being
	// read from there. Both parse, because renderWidgetCase computes the
	// census's ratios over them.
	const (
		splitRole      = "#010203"
		splitInputBase = "#040506"
	)
	// A copy, not a mutation: core.DefaultTheme is a package-level pointer that
	// every other test in this binary renders through.
	split := *core.DefaultTheme
	split.Colors.ControlBorder = splitRole
	split.Components.Input.BorderColor = splitInputBase

	// The guard the whole test rests on. If the two ever resolved to the same
	// hex again — a resolver that ignored the field, a component default that
	// started deriving from the role — every assertion below would pass for
	// the reason this test exists to remove.
	if got := split.Colors.ControlBorderColor(); got != splitRole {
		t.Fatalf("the split theme's role resolves to %q, not the %q it was set to — "+
			"ControlBorderColor is not reading the field this test moves, so the two "+
			"authorities are not actually split and nothing below discriminates",
			got, splitRole)
	}
	if split.Colors.ControlBorderColor() == split.Components.Input.BorderColor {
		t.Fatal("the split theme's role and Input base hold the same hex, which is " +
			"the state this test exists to escape")
	}

	want := map[string]string{
		ringFromRole:      splitRole,
		ringFromInputBase: splitInputBase,
	}
	// The other spelling, for the failure message: a ring that came from the
	// wrong place is much easier to read as "it took the field base" than as
	// "it was #040506".
	name := map[string]string{splitRole: ringFromRole, splitInputBase: ringFromInputBase}

	for _, w := range widgetBuilders {
		c, err := renderWidgetCase("a theme whose role and field base differ", &split, w)
		if err != nil {
			t.Fatalf("rendering the %s swatch: %v", w.what, err)
		}
		expect, known := want[c.RingFrom]
		if !known {
			t.Errorf("%s names %q as the source of its boundary tone, which is not a "+
				"spelling this test can read", c.What, c.RingFrom)
			continue
		}
		if c.Ring != expect {
			from := name[c.Ring]
			if from == "" {
				from = "neither authority — " + c.Ring
			}
			t.Errorf("%s says its boundary comes from %s and it drew %s. Every bundled "+
				"theme holds the same hex in both, so this widget paints the right "+
				"pixel today and will paint the wrong one for the first theme that "+
				"styles its fields separately — which is the whole reason the two are "+
				"read from different places",
				c.What, c.RingFrom, from)
		}
	}
}
