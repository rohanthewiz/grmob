package verify

import (
	"strings"
	"testing"
)

// core.Opacity on both native renderers.
//
// The same three halves Rotate has, because it fails the same three ways and
// neither native runs under `go test ./...`: a key that is parsed and handed
// to nothing, a modifier at the wrong layer, and (Opacity's own) a Transition
// that does not reach it. wasm/verify's opacity_test.go holds the sentinel's
// number; these hold the call sites.

// Both parsers must read the key off the wire.
func TestBothNativeParsersReadOpacity(t *testing.T) {
	for _, pin := range []struct{ file, key string }{
		{swiftStyle, `num("Opacity")`},
		{kotlinStyle, `alphaOf(obj.optDouble("Opacity"`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: does not parse %s — core.Opacity crosses the bridge and "+
				"is dropped on this target", pin.file, pin.key)
		}
	}
}

// Both renderers must hand the READING to the platform's alpha, never the raw
// field: the raw field is 0 for unset and -1 for transparent, which as an
// alpha draws every node invisible and crashes Compose on the one node that
// asked to fade out.
func TestBothNativeRenderersApplyTheOpacityReading(t *testing.T) {
	kotlin := codeIn(t, kotlinStyle)
	if !strings.Contains(kotlin, "if (layerAlpha < 1f) m = m.alpha(layerAlpha)") {
		t.Errorf("%s: no Modifier.alpha for core.Opacity in boxModifier — the "+
			"alpha parses and never fades anything", kotlinStyle)
	}

	swift := codeIn(t, swiftStyle)
	if !strings.Contains(swift, "(s?.alpha ?? 1)") {
		t.Errorf("%s: grMobBox's .opacity does not read s?.alpha. The raw "+
			"`opacity` field is the JSON as written, and as an alpha it hides "+
			"every node that never mentioned one", swiftStyle)
	}
}

// The layer order. An alpha is a drawing layer on both platforms, as a
// rotation is, and fades only what is painted inside it: below the background
// it would fade the content over a fill that stayed solid. CSS's opacity
// always fades the whole border box, so the natives are the only targets
// where the mistake is available.
func TestOpacityWrapsThePaintedBoxOnBothNatives(t *testing.T) {
	kotlin := codeIn(t, kotlinStyle)
	alphaAt := strings.Index(kotlin, "if (layerAlpha < 1f) m = m.alpha(layerAlpha)")
	shadowAt := strings.Index(kotlin, "m = m.shadow(elevation")
	bgAt := strings.Index(kotlin, "background?.let { m = m.background(it) }")
	if alphaAt < 0 || shadowAt < 0 || bgAt < 0 {
		t.Fatalf("%s: boxModifier was restructured; update this test rather than "+
			"deleting it (alpha=%d shadow=%d background=%d)", kotlinStyle, alphaAt, shadowAt, bgAt)
	}
	// Earlier in a Compose chain is further out.
	if alphaAt > shadowAt || alphaAt > bgAt {
		t.Errorf("%s: the Opacity layer is applied after the shadow or the "+
			"background, so the box's own paint does not fade with its content",
			kotlinStyle)
	}

	swift := codeIn(t, swiftStyle)
	opacityAt := strings.Index(swift, "(s?.alpha ?? 1)")
	shadowAt = strings.Index(swift, ".grMobShadow(s?.shadow ?? 0)")
	transitionAt := strings.Index(swift, ".grMobTransition(s, reduceMotion: reduceMotion)")
	if opacityAt < 0 || shadowAt < 0 || transitionAt < 0 {
		t.Fatalf("%s: grMobBox was restructured; update this test rather than "+
			"deleting it (opacity=%d shadow=%d transition=%d)", swiftStyle, opacityAt, shadowAt, transitionAt)
	}
	// SwiftUI is the mirror image: later in the chain is further out. The
	// alpha must be outside the painted box, and inside the .animation that
	// eases it.
	if opacityAt < shadowAt {
		t.Errorf("%s: .opacity is applied inside the shadow, so the box's own "+
			"paint does not fade with its content", swiftStyle)
	}
	if opacityAt > transitionAt {
		t.Errorf("%s: .opacity is applied outside grMobTransition, so a changed "+
			"alpha snaps under a Transition", swiftStyle)
	}
}

// DisplayHidden's alpha is the Opacity layer's, not a second alpha at the
// foot of boxModifier. At the foot it sat inside the shadow, the background
// and the border, and faded only the content, so a hidden node with a fill
// still drew the fill on Compose alone (N-064): the web's visibility:hidden
// and SwiftUI's opacity hide the whole box.
func TestComposeHidesTheWholePaintedBox(t *testing.T) {
	// valuesIn, not codeIn: the mode is named by a string literal.
	kotlin := valuesIn(t, kotlinStyle)
	if !strings.Contains(kotlin, `val layerAlpha = if (display == "hidden") 0f else opacity`) {
		t.Errorf("%s: DisplayHidden no longer feeds the Opacity layer's alpha", kotlinStyle)
	}
	if strings.Contains(kotlin, `if (display == "hidden") m = m.alpha(0f)`) {
		t.Errorf("%s: a DisplayHidden alpha is back inside the painted box, where "+
			"it fades the content and leaves the fill, shadow and border drawn", kotlinStyle)
	}
}

// Compose animates nothing it is not told to. animatedStyle eases the alpha
// by hand, and without it a fade is the one thing core.Opacity was added for
// and the one thing this target would not do.
func TestComposeEasesOpacityUnderATransition(t *testing.T) {
	body := codeOf(t, kotlinRenderer, "private fun animatedStyle(")
	for _, want := range []string{
		"remember { FloatAnimatable(s.opacity) }",
		"alpha.animateTo(s.opacity, s.transitionTween())",
		// Clamped: an overshooting easing steps outside [0, 1], and
		// Modifier.alpha throws there.
		"opacity = alpha.value.coerceIn(0f, 1f)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("%s: animatedStyle is missing %q, so Opacity snaps where the "+
				"other three targets fade", kotlinRenderer, want)
		}
	}
}
