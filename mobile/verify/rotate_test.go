package verify

import (
	"strings"
	"testing"
)

// core.Rotate on both native renderers.
//
// The prop is one float and its three platform spellings agree on both things
// that usually need a mapping table — degrees, and clockwise-positive — which
// is exactly what makes it easy to half-implement. A parser can read the key
// and the modifier chain can never apply it; nothing about that is a type
// error in either language, and neither native runs under `go test ./...`.
// Same class of silent inertness the gap longhands had, and the same kind of
// source check catches it.

// Both parsers must read the key off the wire.
func TestBothNativeParsersReadRotate(t *testing.T) {
	for _, pin := range []struct{ file, key string }{
		{swiftStyle, `num("Rotate")`},
		{kotlinStyle, `optDouble("Rotate"`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: does not parse %s — core.Rotate crosses the bridge and "+
				"is dropped on this target", pin.file, pin.key)
		}
	}
}

// Reading it is half the job; each renderer must also hand it to the
// platform's own rotation modifier.
func TestBothNativeRenderersApplyRotate(t *testing.T) {
	swift := codeIn(t, swiftStyle)
	if !strings.Contains(swift, "rotationEffect(.degrees(degrees), anchor: .center)") {
		t.Errorf("%s: no .rotationEffect — the angle parses and never turns anything",
			swiftStyle)
	}
	if !strings.Contains(swift, ".grMobRotate(s?.rotate ?? 0)") {
		t.Errorf("%s: grMobRotate is not in grMobBox's chain", swiftStyle)
	}

	kotlin := codeIn(t, kotlinStyle)
	if !strings.Contains(kotlin, "import androidx.compose.ui.draw.rotate") {
		t.Errorf("%s: Modifier.rotate is not imported", kotlinStyle)
	}
	if !strings.Contains(kotlin, "m = m.rotate(rotate)") {
		t.Errorf("%s: no Modifier.rotate in boxModifier — the angle parses and "+
			"never turns anything", kotlinStyle)
	}
}

// The layer order, which is the one thing about this prop that is genuinely
// easy to get wrong and impossible to see in a type check.
//
// A rotation is a drawing layer on both platforms, and a layer turns only what
// is painted inside it. Put it below the background and the content spins
// inside a square that stays put — which still compiles, still animates, and
// still looks like a bug nobody can name. CSS has no equivalent trap
// (`transform` always turns the whole border box), so the natives are the only
// two targets where the mistake is even available.
func TestRotationWrapsThePaintedBoxOnBothNatives(t *testing.T) {
	kotlin := codeIn(t, kotlinStyle)
	rotateAt := strings.Index(kotlin, "m = m.rotate(rotate)")
	bgAt := strings.Index(kotlin, "background?.let { m = m.background(it) }")
	borderAt := strings.Index(kotlin, "m = m.border(borderWidth.dp")
	if rotateAt < 0 || bgAt < 0 || borderAt < 0 {
		t.Fatalf("%s: boxModifier was restructured; update this test rather than "+
			"deleting it (rotate=%d background=%d border=%d)",
			kotlinStyle, rotateAt, bgAt, borderAt)
	}
	// Earlier in a Compose chain is further *out*, so the rotation must come
	// before the fill and the stroke it is supposed to turn.
	if rotateAt > bgAt || rotateAt > borderAt {
		t.Errorf("%s: Modifier.rotate is applied after the background/border, so "+
			"only the content turns and the box stays square", kotlinStyle)
	}

	swift := codeIn(t, swiftStyle)
	rotAt := strings.Index(swift, ".grMobRotate(")
	marginAt := strings.Index(swift, ".padding((s?.margin ?? .zero).insets)")
	shadowAt := strings.Index(swift, ".grMobShadow(s?.shadow ?? 0)")
	if rotAt < 0 || marginAt < 0 || shadowAt < 0 {
		t.Fatalf("%s: grMobBox was restructured; update this test rather than "+
			"deleting it (rotate=%d margin=%d shadow=%d)",
			swiftStyle, rotAt, marginAt, shadowAt)
	}
	// SwiftUI is the mirror image: later in the chain is further out. The
	// rotation must come after the painted box and before the margin, which
	// is the CSS rule — a transform turns the border box and leaves the space
	// reserved around it alone.
	if rotAt < shadowAt {
		t.Errorf("%s: .grMobRotate is applied inside the shadow, so the box's own "+
			"paint does not turn with it", swiftStyle)
	}
	if rotAt > marginAt {
		t.Errorf("%s: .grMobRotate is applied outside the margin, so an "+
			"asymmetrically-spaced node turns about a point that is not its centre",
			swiftStyle)
	}
}

// core.Spin on both natives: the same two halves as Rotate, and the same layer
// position, since a spin is a rotation that keeps going.
//
// Neither native runs under `go test ./...`, so a Spin that is parsed and never
// applied — or applied inside the background, spinning the dot in a square
// that stays put — would be invisible here without these pins.
func TestBothNativesParseAndApplySpin(t *testing.T) {
	for _, pin := range []struct{ file, key string }{
		{swiftStyle, `int("Spin")`},
		{kotlinStyle, `optInt("Spin"`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: does not parse %s — core.Spin crosses the bridge and "+
				"the ring never turns on this target", pin.file, pin.key)
		}
	}

	kotlin := codeIn(t, kotlinStyle)
	for _, want := range []string{
		"if (spinMs != 0) m = m.then(SpinElement(spinMs))",
		// The frame loop and the layer read that make it a paint-only spin.
		"withInfiniteAnimationFrameMillis",
		"placeWithLayer(0, 0) { rotationZ = angle }",
	} {
		if !strings.Contains(kotlin, want) {
			t.Errorf("%s: missing %q", kotlinStyle, want)
		}
	}
	spinAt := strings.Index(kotlin, "m = m.then(SpinElement(spinMs))")
	bgAt := strings.Index(kotlin, "background?.let { m = m.background(it) }")
	if spinAt < 0 || bgAt < 0 || spinAt > bgAt {
		t.Errorf("%s: the spin layer must come before the background in "+
			"boxModifier (spin=%d background=%d), or only the content turns",
			kotlinStyle, spinAt, bgAt)
	}

	swift := codeIn(t, swiftStyle)
	for _, want := range []string{
		".modifier(GrMobSpin(periodMs: s?.spin ?? 0))",
		"TimelineView(.animation(minimumInterval: nil, paused: periodMs == 0))",
		"content.rotationEffect(.degrees(angle(at: timeline.date)), anchor: .center)",
	} {
		if !strings.Contains(swift, want) {
			t.Errorf("%s: missing %q", swiftStyle, want)
		}
	}
	spinAt = strings.Index(swift, ".modifier(GrMobSpin(")
	shadowAt := strings.Index(swift, ".grMobShadow(s?.shadow ?? 0)")
	marginAt := strings.Index(swift, ".padding((s?.margin ?? .zero).insets)")
	if spinAt < 0 || spinAt < shadowAt || spinAt > marginAt {
		t.Errorf("%s: GrMobSpin must sit between the painted box and the margin "+
			"in grMobBox (spin=%d shadow=%d margin=%d)",
			swiftStyle, spinAt, shadowAt, marginAt)
	}
}
