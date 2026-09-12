package comps

import (
	"strings"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/palette"
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
//	          a mid-tone works and the pair clears AA on every bundled theme.
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
// anything about and wrong for one the theme has. DefaultTheme's Primary was
// the case that showed the difference: against iOS systemBlue #007AFF, white
// measures 4.02:1 and black 5.23:1, so the maximum picks *black* — and a
// comps.Button one screen up paints white on the same blue, because the
// theme's Button base states the pair outright. The framework was drawing two
// different answers for one colour, and the calendar's selected day was the
// visible half: a black numeral on iOS system blue, which reads as a rendering
// fault rather than as a selection.
//
// No third ink role was added to the palette to settle it, and the reason is
// arithmetic rather than taste: nothing a theme could name would outscore
// black on a mid-tone, so any fix expressed as another *candidate* would have
// lost the same comparison. What had to change is the question. The theme is
// the authority on its own colours — the same rule the on-light tones landed
// under, where a widget was said to have "the number and not the authority" —
// so the pairing is read, not recomputed, wherever one exists.
//
// # The cost the theme then paid
//
// The declared answer was white on #007AFF at 4.02:1, below WCAG AA's 4.5:1
// for body text, and this function returned it where pure measurement returned
// a passing black. That was not a regression waved through — it was the number
// every filled comps.Button had always painted — and it was recorded as
// owed by the palette rather than by the widgets, because raising it here
// would have meant one widget quietly disagreeing with its theme again.
//
// DefaultTheme has since paid it: Colors.Primary is Apple's accessible blue
// #0040DD and the declared pair is 7.56:1. Only the *role* could, since the
// failing thing was a pairing and a pairing has no second tone to reach for.
//
// # What that leaves here, and what a third palette put back
//
// DefaultTheme and MaterialTheme both pair white with a fill dark enough that
// the maximum would pick white anyway, so neither can show this function
// choosing the declaration over the measurement. For two sessions the evidence
// lived entirely in a fixture — components' midTonePrimaryTheme, which is
// DefaultTheme as it stood before the move.
//
// core.AmberTheme now carries it. Its button declares MD brown 900 over an
// amber fill and the measurement would pick the page's near-black instead, so
// deleting the declaredInk call below repaints every Primary fill in that
// theme. That is a shipped palette rather than a fixture, and it got there by
// making an ordinary brand decision rather than by being tuned to a gap.
//
// One half stayed with the fixture, and provably: the case where the
// declaration picks the *opposite ink pole* — white where measurement says
// black, which is the pairing this function was written for. A declared ink
// that loses the measurement is the lower-contrast of the theme's two, and the
// two ratios multiply to a palette constant of at most 21, so the loser can
// never exceed sqrt(21) = 4.58:1 while the package's AA check asks 4.5:1 of it.
// The band is [4.50, 4.58]. See TestThePoleFlipBandIsTooNarrowToShip, which
// also finds that two of the three bundled palettes could not host such a fill
// at any brand colour, because their near-blacks are soft.
//
// All of that is recorded per rule and per theme by
// TestEveryPaletteRuleStillHasAWitness rather than left as a note, because
// losing a witness is silent: DefaultTheme's move to the accessible blue was a
// straightforward improvement and it quietly left this check with nothing to
// observe.
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
// This is also why comps.Chip reads its accent off the Button base rather
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
//
// A forwarder since a second consumer arrived. wasm/verify pins the census's
// (tone, backdrop, ratio) table into browser.mjs so a real Chrome paints the
// pairs, and it cannot reach an unexported function in this package — so the
// arithmetic itself moved to internal/palette rather than being written twice.
// The name stays because forty-odd call sites and their failure messages use
// it, and because "the widget package computes contrast" is still true.
func contrastRatio(a, b float64) float64 { return palette.Ratio(a, b) }

// relativeLuminance implements the WCAG definition for an #RGB, #RRGGBB or
// #RRGGBBAA color, reporting false for anything it cannot parse.
//
// Alpha is parsed but ignored; see palette.Luminance, which this forwards to
// and which carries the argument.
func relativeLuminance(hex string) (float64, bool) { return palette.Luminance(hex) }
