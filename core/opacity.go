package core

import "math"

// OpacityClear is what core.Opacity(0) stores, and what every renderer must
// read as an opacity of zero: fully transparent.
//
// # Why a sentinel, and why the same one as ShrinkNone
//
// Every optional number in Style means "unset" by being zero, and that is
// right wherever the property's own initial value is zero. Opacity is the
// second field where it is not (flex-shrink was the first; see ShrinkNone).
// Its initial value is 1, so zero and unset are opposite pictures:
//
//	unset   the node is drawn as it always was, fully opaque
//	zero    the node is not drawn at all
//
// A plain zero would be dropped three times over before it reached a screen:
// by `omitzero` on the wire, by applyTo's "non-zero wins" merge, and by every
// renderer's `if Opacity != 0` guard. "Fade this out completely" is the one
// value a fade most needs, and it would have compiled and done nothing.
//
// The alternative was to store the complement (a Fade of 0 for opaque), whose
// zero is its own value and needs no sentinel. It was declined for the wire:
// Style's field names are CSS's and ARIA's own spellings on purpose (see the
// note on Style), a dumped tree should say Opacity:0.4 where the page says
// opacity:0.4, and 1-(1-0.7) is 0.7000000000000001 in every renderer that has
// to undo the complement.
//
// -1 because CSS's opacity is an <alpha-value> clamped to [0, 1], so no author
// can mean a negative one, and because core.Opacity is the only door into the
// field and it clamps before it stores. It is deliberately ShrinkNone's
// number: one rule, "-1 is this property's zero", in each of the four
// runtimes.
//
// # What each target does with it
//
//	htmlout        writes opacity:0
//	WASM runtime   writes opacity "0"
//	Compose        Modifier.alpha(0f), animated under a Transition
//	SwiftUI        .opacity(0), animated under a Transition
const OpacityClear = -1

// Opacity sets how opaque the node and everything inside it is drawn: 1 is
// fully opaque (and what an unset node is), 0 is fully transparent, 0.4 lets
// 60% of what is behind the node show through.
//
//	core.Opacity(0.4)                                   // dimmed
//	core.Opacity(0), core.Transition(200, core.EaseOut) // fades out
//
// # Group opacity, on every target
//
// The node is composited as one picture and that picture is faded, which is
// what CSS's `opacity` means: a child overlapping its parent's fill does not
// show the fill through itself. Compose's Modifier.alpha and SwiftUI's
// .opacity are both layer alphas of the same kind. This is the difference
// from an alpha byte in Background or TextColor, which fades one paint and
// lets overlapping paints add up.
//
// It multiplies down the tree, as layers do: a 0.5 child of a 0.5 parent is
// drawn at 0.25, and no child can be more opaque than its parent.
//
// # Paint only
//
// The box keeps its size and its place, like Rotate and Translate, so fading
// a node never reflows its siblings. A faded node also keeps its semantics:
// a screen reader still reads a node at Opacity(0), and on the web and
// Compose it still takes taps. SwiftUI alone stops hit-testing a view at
// exactly zero, so do not lean on either behaviour. A node that should be
// out of reach says so: Display(DisplayNone) or DisplayHidden, Inert, or
// AccessibilityHidden, alongside the fade.
//
// # Under a Transition
//
// Opacity is animatable everywhere, which is what it was added for: a fade
// is the one motion a background-colour ease cannot stand in for when the
// thing fading is text, an image or a whole subtree.
//
//	CSS       the transition the node declares covers `opacity` with the rest
//	Compose   animatedStyle (Renderer.kt) eases the alpha by hand, beside the
//	          background and the translation
//	SwiftUI   the node's one .animation(value: style) covers it
//
// Under reduced motion it snaps with everything else; see core.Transition.
//
// # The values at the edges
//
// The argument is clamped to [0, 1], as CSS clamps it. Zero is stored as
// OpacityClear (see that constant for why), 1 is stored as 1 so that a later
// Opacity(1), or a merged style carrying one, can bring a dimmed node back to
// opaque: the clear that Rotate's merge cannot do, this one can. NaN is
// stored as unset, which is opaque.
func Opacity(alpha float64) StyleProp {
	return styleFunc(func(s *Style) {
		switch {
		case math.IsNaN(alpha):
			s.Opacity = 0
		case alpha <= 0:
			s.Opacity = OpacityClear
		default:
			s.Opacity = math.Min(alpha, 1)
		}
	})
}

// OpacityFactor returns the effective opacity and whether one was declared.
//
// The two returns are the two questions a renderer has, the same pair
// ShrinkFactor answers:
//
//	declared == false   nothing was set. Write no declaration; alpha is 1.
//	declared == true    write alpha, which may be 0.
//
// It is the one statement of the OpacityClear rule for the Go-side renderer
// (htmlout), and the reference the other three transliterate. A stored value
// outside the sentinel and (0, 1], which only a hand-built Style can hold, is
// clamped here rather than trusted, so a renderer never receives an alpha it
// would have to range-check.
func (s Style) OpacityFactor() (alpha float64, declared bool) {
	switch {
	case s.Opacity == 0 || math.IsNaN(s.Opacity):
		return 1, false
	case s.Opacity < 0: // OpacityClear, and any other negative a literal held
		return 0, true
	default:
		return math.Min(s.Opacity, 1), true
	}
}
