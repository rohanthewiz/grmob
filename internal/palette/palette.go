// Package palette holds the facts about a theme's colours that more than one
// of this repository's harnesses needs: the fills a control boundary can be
// drawn on, and the WCAG contrast arithmetic that measures a pair.
//
// # The backdrops, and what they are for
//
// core.ColorPalette.ControlBorder has a 3:1 floor under it (WCAG 1.4.11), and
// a floor is meaningless without the other half of the pair: the fill the
// boundary is drawn *on*. The census in comps/variant_test.go crosses the
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
// to that struct is a decision somebody has to make, and every way it can go
// wrong is a failure rather than a silence: an unclassified field fails
// Untagged, a fill under the floor fails the census, and either classification
// has to carry its argument.
//
// # Reachable, unreachable, and the state between them
//
// The derivation started with one tag. `notbackdrop` said a fill is never
// drawn on; the absence of one said nothing, so the census was the boundary
// tone crossed with whatever fills the struct happened to carry. IsABackdrop
// is the other half and its doc comment carries the argument for why the
// missing half mattered — the short version is that a pair nothing builds was
// measured at the same weight as a pair three widgets build, and that a tag
// could be deleted with no consequence but a pair silently joining the census.
//
// The cost of deriving is that both claims have to survive as data, which is
// what the tags are. That is a fair trade: as a line missing from a slice
// literal an exclusion was invisible, and as a tag it is a claim a test can
// hold (see the three exported readings, and
// TestEveryComponentFillIsClassified).
//
// # The contrast arithmetic
//
// Luminance and Ratio were in comps/variant.go, where Variant.inkOn uses
// them to pick the readable ink for a fill. They moved here when a second
// consumer arrived that could not reach them: wasm/verify pins a table of
// (tone, backdrop, ratio) into browser.mjs so a real Chrome paints the pairs
// the census measures, and computing the ratio there would have been a second
// WCAG implementation in one module — the exact failure a contrast floor is
// least able to survive, since both copies would look right and only one would
// be.
//
// comps/variant.go keeps its own spellings as one-line forwarders, so
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
// failure reports it by and the reason it is in the list at all.
//
// What is the census key — knownBoundaryShortfalls in comps/variant_test.go
// is keyed by "<theme>/<What>" — so the spelling is part of the contract and
// not a label. The component rows are "<field> fill", which is what the
// hand-written list said before this package derived them, so the existing
// keys did not move.
type Backdrop struct {
	What string
	Hex  string

	// Why is the reachability claim: what actually draws a control boundary on
	// this fill. For a component row it is the field's `backdrop` tag verbatim;
	// for the two palette roles it is stated in Backdrops, which is where those
	// two are named.
	//
	// # Why a measured pair carries an argument
	//
	// The census used to be a full cross product — the boundary tone against
	// every fill in the theme — and a cross product cannot tell a pair three
	// widgets build from a pair nothing builds. Both appear as a row with a
	// number under it, both have to clear 3:1, and a shortfall in either one
	// reads the same way to whoever has to fix it. The `notbackdrop` tag said
	// which fills were *not* reachable; nothing said why the rest were.
	//
	// So the claim travels with the pair. A failing row can now name the thing
	// that builds it, which is the difference between "ControlBorder is 2.9:1
	// on Card" and "a FormField's frame inside a Card is 2.9:1" — the second
	// is a screen somebody can go and look at.
	//
	// Empty only for a field carrying neither tag, which is a state Untagged
	// reports and the census refuses to run in.
	Why string
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
// Both fields it names fail the 3:1 floor in all three bundled themes, which is
// exactly why the exclusion has to be argued rather than assumed — an entry
// that merely made a failure go away would be the census defeating itself.
//
// # Why it is read off the struct
//
// It used to be a map literal here, and that put the exclusion in the one place
// a theme author never opens. Everything else about this list is derived from
// core.ComponentDefaults precisely so that adding a field cannot silently leave
// a pair unmeasured — and the exception to that derivation was a name in an
// internal package, invisible in the diff that adds the field beside it.
//
// The `notbackdrop` struct tag is the fact in its own place: the argument sits
// next to the field it is about, where the person adding a Sheet or a Popover
// is already looking, and this function is the reading of it. That also makes
// the two failure modes the old map had impossible rather than checked — an
// entry naming a field that does not exist cannot be written, and neither can
// one that has drifted from the field it names.
//
// A function rather than a var because it is derived, and because a map that
// looked like data would go on looking editable here.
func NotABackdrop() map[string]string { return tagged(notABackdropTag) }

// tagged is the reflection both readings share: field name -> the tag's value,
// for every field carrying key. One walk rather than two near-copies, so the
// two readings cannot disagree about what "carries a tag" means.
func tagged(key string) map[string]string {
	t := reflect.TypeOf(core.ComponentDefaults{})
	out := map[string]string{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if why, ok := f.Tag.Lookup(key); ok {
			out[f.Name] = why
		}
	}
	return out
}

// Untagged returns the core.ComponentDefaults fields carrying neither tag, in
// declaration order — the fields nobody has said anything about.
//
// This is the state the two tags exist to make impossible. A new field is a new
// fill, and until somebody says whether a control boundary can be drawn on it
// there is no honest thing for the census to do: measuring it asserts a pair
// nothing has claimed exists, and skipping it drops a pair silently, which is
// the failure the derivation was built to prevent in the first place. So the
// answer is neither — the census refuses to run and names the field.
//
// A field carrying *both* tags is reported here too. It is the same absence of
// a decision wearing two hats, and the alternative (letting one win) would make
// which one silently load-bearing.
func Untagged() []string {
	is, not := IsABackdrop(), NotABackdrop()
	var out []string
	for _, name := range ComponentFields() {
		_, yes := is[name]
		_, no := not[name]
		if yes == no {
			out = append(out, name)
		}
	}
	return out
}

// The two struct tag keys, spelled once. core.ComponentDefaults carries the
// values; this package is their only reader.
//
// They are exclusive and, together, total: every field must carry exactly one,
// and Untagged is what refuses the state where a field carries neither. See
// IsABackdrop for what the pair buys over the single tag it replaced.
const (
	isABackdropTag  = "backdrop"
	notABackdropTag = "notbackdrop"
)

// IsABackdrop names the core.ComponentDefaults fields whose fill a control
// boundary really is drawn on, each with the thing that draws it.
//
// # The half NotABackdrop could not state
//
// A `notbackdrop` tag says a fill is unreachable. The absence of one said
// nothing at all: everything else was measured because it was left over, so
// the census was the boundary tone crossed with every fill the struct
// happened to carry. That is a set with no claim behind it, and it has two
// costs a contrast census can ill afford.
//
// The first is that a pair nothing builds is measured beside a pair three
// widgets build, at the same weight, and a shortfall in either reads
// identically. The second is worse and is what decided this: a `notbackdrop`
// tag could be *deleted* and the only consequence would be a pair quietly
// joining the census. Removing core.ComponentDefaults.Camera's tag adds a pair
// that clears 6:1, so the whole run stays green while a geometry claim has
// been thrown away. (Button's is load-bearing by accident — Primary as a fill
// is 2.17:1 and the census fails — so half the exclusions were defended by
// their own numbers and half by nothing.)
//
// Making the tags total closes both. A field carrying neither is Untagged,
// which is a hard failure rather than a silent measurement, so deleting either
// tag is now the same kind of event as deleting a field's name.
//
// # Why the argument and not a bare marker
//
// The value is what draws there, in the words of whoever knew. `backdrop:""`
// would make the tags total and would say nothing — and the reason to prefer a
// tag over a list in this package was never the totality, it was that the
// claim sits next to the field it is about, where the person adding a Sheet or
// a Popover is already looking.
func IsABackdrop() map[string]string { return tagged(isABackdropTag) }

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
	excluded, reachable := NotABackdrop(), IsABackdrop()
	out := []Backdrop{
		// The two roles' reachability claims are stated here rather than in a
		// tag because these are core.ColorPalette roles and there is no field
		// on ComponentDefaults to hang them off. They are also the two claims
		// least likely to need revisiting: a page and a raised panel are what
		// a control is drawn on by definition.
		{"Background", theme.Colors.Background,
			"the page. Every screen is drawn on it, so every control that is not " +
				"inside a Card, a panel or a field is drawn on it too — this is the " +
				"pair a bare comps.Input on a Screen produces"},
		{"Surface", theme.Colors.Surface,
			"the raised fill: a quiet comps.Chip's interior, a " +
				"comps.GroupHeader band, a Card in the themes that give the two " +
				"the same hex. The quiet Chip is the one that decided " +
				"core.ColorPalette.ControlBorder's own value — its ring is drawn on " +
				"this fill and on the page at once"},
	}
	for _, name := range ComponentFields() {
		if _, skip := excluded[name]; skip {
			continue
		}
		if hex := Fill(theme, name); hex != "" {
			// reachable[name] is "" for an untagged field, which is the one
			// way a Backdrop can come back with no argument. Measured anyway
			// rather than skipped: a field nobody has classified must not
			// silently leave the census, and Untagged is what fails the run.
			out = append(out, Backdrop{name + " fill", hex, reachable[name]})
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
// order, so a test can walk the same fields Backdrops and NotABackdrop do.
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
