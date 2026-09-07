// Package palette holds the facts about a theme's colours that more than one
// of this repository's harnesses needs: the fills a control boundary can be
// drawn on, and the WCAG contrast arithmetic that measures a pair.
//
// # The backdrops, and what they are for
//
// core.ColorPalette.ControlBorder has a 3:1 floor under it (WCAG 1.4.11), and
// a floor is meaningless without the other half of the pair: the fill the
// boundary is drawn *on*. The census in components/variant_test.go crosses the
// tone with every such fill, and wasm/verify/browser.mjs paints the same pairs
// in a real browser and measures what Chrome actually painted. Both need the
// same list, and a list written out twice proves only that one person made the
// same mistake twice — the argument internal/menufixture already makes.
//
// # Why it is derived rather than named
//
// The list used to be a slice literal: Background, Surface, Card, Input,
// TextArea, with Camera left out by name. Every entry was correct and the
// shape was wrong, because the set is not a decision — it is a *consequence*
// of core.ComponentDefaults. A new field carrying a Background (a Sheet, a
// Popover) is a fill a control can sit on, and a hand-written list would not
// know: the pair would simply go unmeasured, which is the failure the census
// exists to make impossible.
//
// So the two palette roles are named — they are roles, not components — and
// everything else is read off ComponentDefaults by reflection. Adding a field
// to that struct adds a backdrop, and the two things that could go wrong when
// it does are both failures rather than silences: a fill under the floor
// fails the census, and a field that is deliberately not a backdrop has to be
// written into NotABackdrop with a reason.
//
// The cost of deriving is that the Camera exclusion had to survive as data,
// which is the map below. That is a fair trade: as a line missing from a slice
// literal it was invisible, and as a map entry it is a claim a test can hold
// (see the two exported helpers, and TestTheBackdropExclusionsNameRealFills).
//
// # The contrast arithmetic
//
// Luminance and Ratio were in components/variant.go, where Variant.inkOn uses
// them to pick the readable ink for a fill. They moved here when a second
// consumer arrived that could not reach them: wasm/verify pins a table of
// (tone, backdrop, ratio) into browser.mjs so a real Chrome paints the pairs
// the census measures, and computing the ratio there would have been a second
// WCAG implementation in one module — the exact failure a contrast floor is
// least able to survive, since both copies would look right and only one would
// be.
//
// components/variant.go keeps its own spellings as one-line forwarders, so
// every call site and every test that named them is untouched.
//
// It lives under internal/ for the reason menufixture does: it is not part of
// the framework's API, it is a fact this repository's own harnesses share.
package palette

import (
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// Backdrop is one fill a control boundary can land on, with the name a
// failure reports it by.
//
// What is the census key — knownBoundaryShortfalls in components/variant_test.go
// is keyed by "<theme>/<What>" — so the spelling is part of the contract and
// not a label. The component rows are "<field> fill", which is what the
// hand-written list said before this package derived them, so the existing
// keys did not move.
type Backdrop struct {
	What string
	Hex  string
}

// NotABackdrop names the core.ComponentDefaults fields whose fill no control
// boundary is ever drawn on, each with the reason it is exempt.
//
// An entry here is a claim about the *geometry* of the framework, not about
// the numbers: it says no widget can put a bordered control on top of this
// fill, so measuring the pair would be measuring something nothing draws. That
// is a different kind of statement from knownBoundaryShortfalls, which says a
// pair is real, falls short, and is allowed to.
//
// Both fields below fail the 3:1 floor in all three bundled themes, which is
// exactly why the exclusion has to be argued rather than assumed — an entry
// that merely made a failure go away would be the census defeating itself.
var NotABackdrop = map[string]string{
	"Camera": "a viewfinder: its fill is black in every theme because it is what " +
		"shows for the frame before the first camera frame arrives, and nothing " +
		"draws a control boundary on top of a preview. Excluded by name rather " +
		"than by a lightness test, because a rule that skipped dark fills would " +
		"also skip a dark theme's page",
	"Button": "a control's own fill, not a surface: Colors.Primary. A bordered " +
		"control is never drawn on top of a filled button — an outline Button " +
		"draws its own edge over whatever is behind it, which is the page or a " +
		"panel, and both of those are already measured. Excluded because the " +
		"pair is unreachable, not because it is close",
}

// Backdrops returns every fill a control boundary can be drawn on in theme, in a
// stable order: the two palette roles first, then the ComponentDefaults fields
// in declaration order.
//
// A component that states no fill is skipped rather than reported. That is not
// the same silence the palette roles get — an empty Colors.Background is a
// theme with no page colour, which the census reports — because a component
// default legitimately declares nothing and inherits whatever is behind it.
// Column, Row and (in two of the three themes) CheckBox are all in that
// position today.
func Backdrops(theme *core.Theme) []Backdrop {
	out := []Backdrop{
		{"Background", theme.Colors.Background},
		{"Surface", theme.Colors.Surface},
	}
	for _, name := range ComponentFields() {
		if _, skip := NotABackdrop[name]; skip {
			continue
		}
		if hex := Fill(theme, name); hex != "" {
			out = append(out, Backdrop{name + " fill", hex})
		}
	}
	return out
}

// Fill returns the Background a named core.ComponentDefaults field states in
// theme, or "" if it states none or there is no such field.
//
// It ignores NotABackdrop, which is the whole reason it is separate from For:
// the test that holds the exclusions to the struct has to be able to ask what
// an *excluded* field carries, and asking through Backdrops — which skips
// them — could only ever answer "nothing".
func Fill(theme *core.Theme, field string) string {
	f := reflect.ValueOf(theme.Components).FieldByName(field)
	if !f.IsValid() || f.Kind() != reflect.Struct {
		return ""
	}
	// Every ComponentDefaults field is a core.Style today. A field of some
	// other shape contributes nothing rather than panicking, and the test one
	// package over is what notices: an exclusion for such a field reports
	// itself as exempting nothing.
	bg := f.FieldByName("Background")
	if !bg.IsValid() || bg.Kind() != reflect.String {
		return ""
	}
	return bg.String()
}

// ComponentFields returns core.ComponentDefaults' field names in declaration
// order, so a test can hold NotABackdrop to the struct it names.
//
// Exported for that one purpose: an exclusion for a field that no longer
// exists exempts nothing, and it reads as though the census had considered a
// component it has never heard of.
func ComponentFields() []string {
	t := reflect.TypeOf(core.ComponentDefaults{})
	out := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		out = append(out, t.Field(i).Name)
	}
	return out
}

// --- The contrast arithmetic ------------------------------------------------

// Ratio is the WCAG 2.x formula over two relative luminances:
// (lighter + 0.05) / (darker + 0.05), ranging from 1 (identical) to 21 (black
// on white).
func Ratio(a, b float64) float64 {
	hi, lo := a, b
	if lo > hi {
		hi, lo = lo, hi
	}
	return (hi + 0.05) / (lo + 0.05)
}

// Luminance implements the WCAG definition of relative luminance for an #RGB,
// #RRGGBB or #RRGGBBAA color, reporting false for anything it cannot parse.
//
// Alpha is parsed but ignored: compositing needs the backdrop, and a widget
// resolving its own ink does not know what it will be drawn over. A
// translucent fill therefore reads as its opaque form, which overestimates
// contrast — acceptable, since every palette fill role is opaque and the one
// translucent value in the bundled themes (TextSecondary) is an ink.
func Luminance(hex string) (float64, bool) {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	switch len(h) {
	case 3: // #RGB shorthand — each digit doubles, as in CSS
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	case 6, 8:
		h = h[:6]
	default:
		return 0, false
	}

	channels := [3]float64{}
	for i := range channels {
		v, err := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
		if err != nil {
			return 0, false
		}
		channels[i] = linearize(float64(v) / 255)
	}
	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2], true
}

// linearize undoes the sRGB transfer function, converting a gamma encoded
// 0..1 channel to linear light. Luminance is a sum of *linear* intensities;
// averaging the encoded values instead is the classic mistake that makes
// mid-tones look far brighter than they are.
func linearize(c float64) float64 {
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}
