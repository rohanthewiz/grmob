package components

import (
	"math"
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// Variant selects a widget's semantic color role — what a piece of UI *means*
// rather than what it looks like. It is shared across the package rather than
// owned by Badge so a future Alert, Banner or status Chip resolves the same
// four roles the same way, and so a caller can pass one value around.
//
// It is a string enum with an empty zero value, matching core's Alignment and
// DisplayMode. That is load-bearing here: the zero value must be the existing
// look, or adding the field would restyle every Badge already in a tree.
type Variant string

const (
	// VariantDefault is the zero value: the theme's Primary, the badge look
	// that predates variants.
	VariantDefault Variant = ""
	VariantSuccess Variant = "success"
	VariantWarning Variant = "warning"
	VariantError   Variant = "error"
)

// Color resolves the variant to a background from the theme's palette.
//
// Success and Warning go through their resolver methods so a theme predating
// those roles falls back to a visible default rather than to no color; Error
// is one of the palette's original seven and is read directly, since no theme
// can be missing it.
func (v Variant) Color(t *core.Theme) string {
	switch v {
	case VariantSuccess:
		return t.Colors.SuccessColor()
	case VariantWarning:
		return t.Colors.WarningColor()
	case VariantError:
		return t.Colors.Error
	default:
		return t.Colors.Primary
	}
}

// OnLight resolves the variant to the ink-weight tone of its role — the value
// to spend when the color *is* the ink, rather than the fill something else is
// laid over.
//
// Color and this are the two halves of one role, and which one a widget wants
// is decided by what it does with it:
//
//	Color     a fill. The ink over it is chosen by contrast (Ink, below), so
//	          a mid-tone works and the pair clears AA on both bundled themes.
//	OnLight   ink itself — an outlined button's label and rule, a loud chip's
//	          outline. The backdrop is whatever the widget was placed on,
//	          which the widget cannot see, so the value has to be dark enough
//	          to be read against a light surface on its own.
//
// VariantDefault resolves through the palette's Primary tone here, with no
// special arm, and the reason is that there is nothing for one to preserve.
// Ink's answer for the default is a *pairing* the theme itself declares
// (Background over Primary, which is what Button already paints, and which Ink
// now reads back rather than assuming); no theme declares anything about a
// role spent as ink on an unknown backdrop, because before these tones existed
// every caller spent the role colour raw — which is exactly what the unset
// fallback still returns.
func (v Variant) OnLight(t *core.Theme) string {
	switch v {
	case VariantSuccess:
		return t.Colors.SuccessOnLightColor()
	case VariantWarning:
		return t.Colors.WarningOnLightColor()
	case VariantError:
		return t.Colors.ErrorOnLightColor()
	default:
		return t.Colors.PrimaryOnLightColor()
	}
}

// Ink returns the label color to lay over bg.
//
// # Why this is computed rather than a fixed pairing
//
// The palette names one color per role and no ink to go with it, so a status
// fill arrives without a partner. Picking one badly is not a cosmetic problem:
// under DefaultTheme, white on Success (#34C759) is 2.22:1 and white on
// Warning (#FF9500) is 2.20:1 — below even the 3:1 large-text floor, i.e. a
// badge nobody can read. Black on those two is ~9.5:1.
//
// A fixed per-variant pairing would not survive a theme swap either, because
// the correct ink *flips direction* between the two bundled themes: Success is
// a light green under DefaultTheme (wants dark ink) and a dark green under
// MaterialTheme (wants light ink). So the choice is made per color, against
// the theme's own two ink roles, at render time.
//
// # The variant is not consulted, and used to be
//
// VariantDefault had an arm of its own here that returned the theme's
// Background whatever bg was, to keep the Primary/Background pairing both
// bundled themes chose and Button paints. That is still the answer it gets —
// but it is now reached by asking the theme rather than by exempting a
// constant, which is inkOn's whole subject. Two things fall out of the swap:
//
//   - Badge{Color: "#FFF9C4"} with no variant used to get white ink on pale
//     yellow, because the exemption ignored bg entirely. Badge's own doc
//     already promised the opposite ("resolved against bg, so an explicit
//     Color still gets a legible ink picked for it"); it is true now.
//   - A theme that states no Components.Button base at all — examples exist,
//     see the Components note in examples/fintechapp — has declared no
//     pairing, so its default variant is measured like any other. That is the
//     one case whose pixels move, and towards the more legible ink.
func (v Variant) Ink(t *core.Theme, bg string) string {
	return inkOn(t, bg)
}

// inkOn returns the ink to lay over fill: the pairing the theme has already
// declared for that fill, and the best-contrast choice among the theme's two
// ink roles when it has declared none.
//
// # Why measurement alone is the wrong rule for a palette role
//
// contrastInk maximises contrast, which is right for a colour nobody has said
// anything about and wrong for one the theme has. DefaultTheme's Primary is
// the case that shows the difference: against #007AFF, white measures 4.02:1
// and black 5.23:1, so the maximum picks *black* — and a components.Button one
// screen up paints white on the same blue, because the theme's Button base
// states the pair outright. The framework was drawing two different answers
// for one colour, and the calendar's selected day was the visible half: a
// black numeral on iOS system blue, which reads as a rendering fault rather
// than as a selection.
//
// No third ink role was added to the palette to settle it, and the reason is
// arithmetic rather than taste: nothing a theme could name would outscore
// black on a mid-tone, so any fix expressed as another *candidate* would have
// lost the same comparison. What had to change is the question. The theme is
// the authority on its own colours — the same rule the on-light tones landed
// under, where a widget was said to have "the number and not the authority" —
// so the pairing is read, not recomputed, wherever one exists.
//
// # The honest cost
//
// White on #007AFF is 4.02:1, below WCAG AA's 4.5:1 for body text, and this
// function now returns it where the old rule returned a passing black. That is
// not a contrast regression being waved through: it is the number every
// filled components.Button in the framework has always painted, because it is
// the pair DefaultTheme declares. Raising it is the theme's move (a darker
// Button base, or Apple's own accessible blue #0040DD, which this palette
// already carries as PrimaryOnLight) and it would lift the buttons and the
// calendar together. One widget quietly disagreeing with the theme fixed
// nothing and hid the question.
func inkOn(t *core.Theme, fill string) string {
	if ink := declaredInk(t, fill); ink != "" {
		return ink
	}
	// Background first so it wins ties and so an unreadable fill degrades to
	// the pre-variant behavior rather than to something arbitrary.
	return contrastInk(fill, t.Colors.Background, t.Colors.TextPrimary)
}

// declaredInk returns the ink a theme has itself paired with fill, or "" if it
// has paired none.
//
// Components.Button is the only place a theme states a fill and an ink
// *together*, which is what makes it a declaration rather than two colours
// that happen to sit near each other. A filled button is the one control a
// palette cannot describe without answering the question — Card names no ink,
// Input's pair is a field's own text on its own white — so the button base is
// where the answer lives, and reading it back is the same reverse lookup
// ColorPalette.OnLight performs one property over.
//
// This is also why components.Chip reads its accent off the Button base rather
// than off Colors.Primary: a theme whose buttons are not primary-coloured has
// said something, and the button is where it said it.
//
// Both halves must be present. A base that sets a background and no text
// colour has not named a pair, and returning its empty TextColor would paint
// an invisible label; a base that sets a text colour and no background has
// nothing to match fill against.
//
// Comparison is case-insensitive, as OnLight's is: "#007AFF" and "#007aff" are
// one colour to every renderer, and a theme is hand-written. It is not
// format-insensitive — "#FFF" and "#FFFFFF" are the same colour and different
// strings — which costs nothing here, since the miss falls through to
// measurement rather than to an error.
func declaredInk(t *core.Theme, fill string) string {
	base := t.Components.Button
	if base.Background == "" || base.TextColor == "" {
		return ""
	}
	if !strings.EqualFold(fill, base.Background) {
		return ""
	}
	return base.TextColor
}

// contrastInk returns whichever candidate has the highest WCAG contrast ratio
// against bg. The first candidate is the fallback: it is returned when bg
// cannot be parsed, and it wins any tie.
//
// This can only choose between the inks the theme offers. A theme whose two
// ink roles are both poor against a status color gets the better of two bad
// options — the fix for that is the theme's, not the widget's.
func contrastInk(bg string, candidates ...string) string {
	bgLum, ok := relativeLuminance(bg)
	if !ok || len(candidates) == 0 {
		if len(candidates) == 0 {
			return ""
		}
		return candidates[0]
	}

	best, bestRatio := candidates[0], -1.0
	for _, c := range candidates {
		lum, ok := relativeLuminance(c)
		if !ok {
			continue
		}
		if r := contrastRatio(bgLum, lum); r > bestRatio {
			best, bestRatio = c, r
		}
	}
	return best
}

// contrastRatio is the WCAG 2.x formula: (lighter + 0.05) / (darker + 0.05),
// ranging from 1 (identical) to 21 (black on white).
func contrastRatio(a, b float64) float64 {
	hi, lo := a, b
	if lo > hi {
		hi, lo = lo, hi
	}
	return (hi + 0.05) / (lo + 0.05)
}

// relativeLuminance implements the WCAG definition for an #RGB, #RRGGBB or
// #RRGGBBAA color, reporting false for anything it cannot parse.
//
// Alpha is parsed but ignored: compositing needs the backdrop, and a widget
// resolving its own ink does not know what it will be drawn over. A
// translucent fill therefore reads as its opaque form, which overestimates
// contrast — acceptable, since every palette fill role is opaque and the one
// translucent value in the bundled themes (TextSecondary) is an ink.
func relativeLuminance(hex string) (float64, bool) {
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
		channels[i] = linearizeChannel(float64(v) / 255)
	}
	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2], true
}

// linearizeChannel undoes the sRGB transfer function, converting a gamma
// encoded 0..1 channel to linear light. Luminance is a sum of *linear*
// intensities; averaging the encoded values instead is the classic mistake
// that makes mid-tones look far brighter than they are.
func linearizeChannel(c float64) float64 {
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}
