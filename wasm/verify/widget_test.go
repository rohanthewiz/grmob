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
// to know whether those colours are the ones components/variant_test.go
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

func widgetCaseFor(t *testing.T, theme string) widgetCase {
	t.Helper()
	for _, c := range widgetCases() {
		if c.Theme == theme {
			return c
		}
	}
	t.Fatalf("no widget swatch for %s", theme)
	return widgetCase{}
}

// One swatch per bundled theme, and no others.
//
// Both directions, for the reason palette_test.go gives about its own rows: a
// theme with no swatch is a palette whose widgets the browser has never looked
// at, and a swatch for a theme core does not ship is a check painting a
// boundary nothing draws.
func TestEveryBundledThemeHasAWidgetSwatch(t *testing.T) {
	got := map[string]bool{}
	for _, c := range widgetCases() {
		if got[c.Theme] {
			t.Errorf("two widget swatches for %s — the browser pass would mount both "+
				"and a failure could not say which", c.Theme)
		}
		got[c.Theme] = true
	}
	for name := range core.BundledThemes() {
		if !got[name] {
			t.Errorf("no widget swatch for %s: the browser pass paints the swatch grid "+
				"for this palette and never asks whether a widget draws it", name)
		}
		delete(got, name)
	}
	for name := range got {
		t.Errorf("a widget swatch for %q, which core.BundledThemes() does not have — "+
			"the browser is checking a boundary nothing ships", name)
	}
}

// The ring a real widget draws is the palette role, not a colour that happens
// to look like it.
//
// This is the assertion the browser cannot make. components.chipRing reads
// Colors.ControlBorderColor rather than Components.Input.BorderColor
// deliberately — the two hold the same hex in every bundled theme today, and
// tying a chip's edge to a text field's is exactly the drift that argument was
// written to prevent. Reading the tone off the rendered node and comparing it
// with the role is what keeps that argument true rather than remembered.
//
// # What it cannot catch, and why nothing here can
//
// Precisely that swap. Components.Input.BorderColor holds the same hex as the
// role in all three bundled themes — core/theme_test.go's
// TestBundledFieldFramesAreTheControlBorderRole is what makes that true — so a
// chip that read the field base instead would render an identical tone and
// pass this comparison, every pixel in the browser, and every other test here.
// It is a difference in *provenance* with no observable consequence until
// somebody restyles their text fields, which is the day the argument in
// chipRing is about.
//
// A hex comparison cannot see that, and neither can a screenshot. What this
// test does catch is the whole of the rest: a tone that is neither, a fallback
// taken, a Style override, a theme whose role moved and whose widget did not.
func TestTheWidgetSwatchRingIsThePaletteRole(t *testing.T) {
	for name, theme := range core.BundledThemes() {
		c := widgetCaseFor(t, name)
		if want := theme.Colors.ControlBorderColor(); c.Ring != want {
			t.Errorf("%s: the rendered widget's boundary is %q and the theme's "+
				"ControlBorder role is %q. The browser pass compares the pixel with "+
				"the first of those, so it would go on passing while the widget drew "+
				"a tone the contrast census has never measured", name, c.Ring, want)
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
		c := widgetCaseFor(t, name)
		measured := map[string]bool{}
		for _, b := range palette.Backdrops(theme) {
			measured[b.Hex] = true
		}
		for _, pair := range []struct{ what, hex string }{
			{"the page behind the widget", c.Page},
			{"the widget's own fill", c.Fill},
		} {
			if pair.hex == "" {
				t.Errorf("%s: %s is unstated — the rendered node declares no colour "+
					"there, so the browser is sampling a pixel against nothing", name,
					pair.what)
				continue
			}
			if !measured[pair.hex] {
				t.Errorf("%s: %s is %s, which is not one of the fills "+
					"internal/palette derives from core.ComponentDefaults. The ring is "+
					"drawn on it and its 3:1 floor has never been computed", name,
					pair.what, pair.hex)
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
		c := widgetCaseFor(t, name)
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
				t.Errorf("%s: %s compares colours that do not parse (%q, %q)",
					name, pair.what, pair.a, pair.b)
				continue
			}
			want := math.Round(palette.Ratio(la, lb)*100) / 100
			if pair.got != want {
				t.Errorf("%s: the swatch carries %.2f:1 %s and palette.Ratio says "+
					"%.2f:1", name, pair.got, pair.what, want)
			}
		}
	}
}
