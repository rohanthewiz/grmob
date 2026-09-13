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
// background color, size, padding, list placement — should animate over the
// given duration instead of snapping. This is the "declare in Go, drive
// natively" model: Go only ships the declaration in the style; each frame of
// the animation is produced by the platform's animation system (Compose,
// SwiftUI, CSS transitions), never by patches over the bridge.
//
// The canonical serialized form is "<ms>ms <easing>" (e.g. "250ms
// ease-in-out"), which the native parsers read; they also tolerate the CSS
// longhand ("all 0.3s ease") for styles written by hand.
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
// # Reduced motion
//
// No target reads the platform's reduce-motion setting today, for Transition
// or for Spin. Android's "Remove animations" (animator duration scale 0) is not
// consulted either, because the frame loop is not a scaled Compose animation
// spec. core has no signal for the setting yet; that is recorded here rather
// than faked on one target.
func Spin(periodMs int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Spin = periodMs
	})
}
