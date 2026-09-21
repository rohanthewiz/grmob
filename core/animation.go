package core

import "strconv"

// Easing names the timing curve of a transition. The values are the CSS
// keywords — the Style.Transition field predates the native renderers and is
// CSS-shaped, so the DSL keeps that vocabulary and each renderer maps it onto
// its own curve type (Compose CubicBezierEasing, SwiftUI Animation). The
// cubic-bezier control points the CSS spec defines for each keyword are what
// the native mappings reproduce, so one Go declaration animates identically
// on Android, iOS, and the web backends.
type Easing string

const (
	EaseLinear Easing = "linear"
	Ease       Easing = "ease"
	EaseIn     Easing = "ease-in"
	EaseOut    Easing = "ease-out"
	EaseInOut  Easing = "ease-in-out"
)

// Transition declares that changes to this node's animatable properties —
// background color, opacity, size, padding, list placement — should animate over the
// given duration instead of snapping. This is the "declare in Go, drive
// natively" model: Go only ships the declaration in the style; each frame of
// the animation is produced by the platform's animation system (Compose,
// SwiftUI, CSS transitions), never by patches over the bridge.
//
// The canonical serialized form is "<ms>ms <easing>" (e.g. "250ms
// ease-in-out"), which the native parsers read; they also tolerate the CSS
// longhand ("all 0.3s ease") for styles written by hand.
//
// # Reduced motion
//
// When the platform's reduce-motion setting is on, a Transition snaps: the
// change lands on the next frame, exactly as it would with no Transition
// declared. The setting is read by each target rather than passed from Go,
// so turning it on mid-session applies to the next change without a render.
//
//	CSS       ReducedMotionCSS: under prefers-reduced-motion: reduce, any
//	          element whose inline style declares a transition gets
//	          transition: none !important
//	Compose   nothing to add: Android's "Remove animations" sets the
//	          animator duration scale to 0, which Compose's frame clock
//	          already reads (MotionDurationScale), and a tween under scale 0
//	          plays straight to its end value
//	SwiftUI   @Environment(\.accessibilityReduceMotion) swaps the node's
//	          Animation for nil
//
// All properties snap, colour included, rather than only the ones that move
// (size, placement, a translation). A colour fade is not the motion the
// setting is about, and the web could keep it, but SwiftUI's Animation is
// scoped to a value and not to a property, so keeping fades there means
// splitting the box chain into per-property animations. One rule that every
// target implements the same way beats a finer one that holds on two.
func Transition(durationMs int, easing Easing) StyleProp {
	return styleFunc(func(s *Style) {
		if durationMs <= 0 {
			s.Transition = ""
			return
		}
		if easing == "" {
			easing = Ease
		}
		s.Transition = strconv.Itoa(durationMs) + "ms " + string(easing)
	})
}

// SpinKeyframes is the stylesheet rule both web targets pair with Style.Spin.
// It animates the individual `rotate` property rather than `transform`, so the
// spin composes with Style.Rotate's `transform: rotate()` instead of replacing
// it (see Spin). One constant, read by htmlout and restated in the WASM
// runtime, because the two web targets must name and shape it identically for
// an export and a live page to turn the same way.
const SpinKeyframes = "@keyframes grmob-spin{from{rotate:0deg}to{rotate:360deg}}"

// ReducedMotionCSS is the stylesheet rule both web targets pair with
// Style.Transition: under the reduce-motion media query, every transition a
// node declares inline is switched off. See Transition, "Reduced motion".
//
// A stylesheet rule rather than a check in the runtime, for three reasons.
// The media query is live, so a reader who turns the setting on mid-session
// is honoured on the next change without a render pass or a listener. It
// works in an htmlout export, which has no script. And it is the only way to
// reach an inline declaration from outside: `!important` in a sheet beats a
// normal inline style.
//
// The selector matches on the inline style attribute rather than on `*`, so
// the rule reaches only elements a grmob renderer gave a transition (a hosting
// page's own transitions are the page's to manage). Both web targets write the
// property inline as "transition", and an element with no transition has
// nothing for the rule to switch off anyway, so the match is exact in the only
// direction that matters.
const ReducedMotionCSS = `@media (prefers-reduced-motion:reduce){[style*="transition"]{transition:none!important}}`

// Spin turns the node one full revolution every periodMs milliseconds, round
// and round, for as long as it is displayed. A negative period turns it
// anticlockwise; zero (the default) holds it still. It is the looping
// counterpart of Transition, and it exists so that a continuously moving
// widget — the comps.Spinner ring — is driven by the platform's frame clock
// rather than by state changes stepped from Go.
//
//	CSS       animation: grmob-spin <ms>ms linear infinite [reverse], with
//	          SpinKeyframes on the individual `rotate` property
//	Compose   a layout modifier node placing the box in a graphics layer whose
//	          rotationZ follows withInfiniteAnimationFrameMillis
//	SwiftUI   TimelineView(.animation) turning the box with .rotationEffect,
//	          the angle computed from the timeline's date
//
// All three compute the angle from elapsed time modulo the period rather than
// accumulating per-frame increments, so a dropped frame costs a skipped angle,
// never a spinner that drifts slower than declared.
//
// # Why a spin and not a general looping transition
//
// Transition animates *between* two values, and the patch supplies both ends:
// the old style and the new one. A loop has no second patch to supply the other
// end, so a general loop would need vocabulary of its own — a from-style, a
// to-style, a repeat count, a direction — mapped onto three animation systems
// that do not agree on what repeating an arbitrary property means (Compose's
// infiniteRepeatable is per animated value, SwiftUI's repeatForever rides a
// transaction, CSS keyframes are a named rule). Rotation is the one property
// whose cycle closes on its own: 360 degrees draws exactly what 0 does, so the
// restart is invisible and the loop needs no second endpoint and no
// alternate-direction mode. It is also the only paint transform core has (see
// Style.Rotate). A pulse or a shimmer would be the second consumer that
// justifies the general form; until one exists, the narrow prop is the one all
// three targets implement identically.
//
// # Linear, always
//
// There is no easing parameter. An eased revolution slows to a stop at the same
// angle every turn, which reads as a stutter rather than a spin, and the
// restart seam that is invisible under linear motion becomes a visible jolt.
// The period is the only knob.
//
// # Composition with Rotate
//
// Spin is added to Rotate, not substituted for it. Both turn the box about its
// own centre, and rotations about one point commute, so each target applies
// them as two layers and draws the same pixels whichever is outermost.
//
// # What it costs
//
// Nothing crosses the bridge after the style that declares it: no patches and
// no render passes. A node with Display none is not composed on either native
// and runs no CSS animation on the web, so a hidden spinning node draws no
// frames either.
//
// # Reduced motion: it keeps turning
//
// A spin is not stopped or slowed when the platform's reduce-motion setting is
// on, on any target, while a Transition under the same setting snaps (see
// Transition). The choice, and what it was weighed against:
//
//   - What a spin says. comps.Spinner is the one consumer, and its motion is
//     the message: "still working". A ring frozen at an angle reads as a hung
//     screen or as decoration, and nothing else on the widget says busy.
//     WCAG 2.3.3 (Animation from Interactions) exempts motion that is
//     essential to the information conveyed; an activity indicator is the
//     textbook case of that.
//   - What the setting is for. Reduce motion targets vestibular triggers:
//     content sliding across the screen, zooms, parallax, large surfaces
//     moving. A small glyph turning in place is none of those.
//   - What the platforms do with their own spinner. UIActivityIndicatorView
//     and SwiftUI's ProgressView keep spinning under Reduce Motion. The web
//     has no built-in spinner, and Bootstrap's slows rather than stops.
//     Android's indeterminate ProgressBar does freeze under "Remove
//     animations", but that switch removes every animator in the system,
//     including the ones apps rely on to show progress, and a frozen ring is
//     the known cost of it rather than a design.
//   - Why not slow it. A slower period is the web-library compromise, but the
//     factor would be invented (twice? four times?), it would need a runtime
//     read of the setting on Compose, where the loop is not a scaled
//     animation, and a slow spin is still a spin to anyone it bothers.
//
// So the frame loops stay as they are: Compose's withInfiniteAnimationFrameMillis
// does not read the animator duration scale, SwiftUI's TimelineView does not
// read the environment, and ReducedMotionCSS touches `transition` only, never
// `animation`. A caller for whom the motion is decoration rather than a status
// should not use Spin for it.
func Spin(periodMs int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Spin = periodMs
	})
}
