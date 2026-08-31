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
// VariantDefault is deliberately exempt and keeps the theme's Background. That
// is the pairing both bundled themes chose for Primary and use for Button, and
// preserving it is what keeps this field's zero value a no-op for every badge
// that already exists.
func (v Variant) Ink(t *core.Theme, bg string) string {
	if v == VariantDefault {
		return t.Colors.Background
	}
	// Background first so it wins ties and so an unreadable bg degrades to
	// the pre-variant behavior rather than to something arbitrary.
	return contrastInk(bg, t.Colors.Background, t.Colors.TextPrimary)
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
