package verify

import (
	"strings"
	"testing"
)

// core.Gradient on both native canvases. Like the fill rule, it is read inside
// the drawing loop, which neither geometry harness runs, so each renderer is
// held to the wire keys it must read and to the one decision that makes its
// gradient agree with SVG's userSpaceOnUse under a stretched viewBox: the
// gradient's space is transformed by the viewport, not just its end points.
// Stroke gradients read the same keys under the "stroke" prefix
// (core.GradientKey), which each renderer builds from the unprefixed names,
// so the check for them is the call site and the width-preserving draw.
func TestBothNativeCanvasesPaintGradients(t *testing.T) {
	keys := []string{`"gradient"`, `"gradientAt"`, `"gradientStops"`, `"gradientColors"`, `"linear"`, `"radial"`}

	swift := valuesIn(t, swiftRenderer)
	for _, k := range keys {
		if !strings.Contains(swift, k) {
			t.Errorf("%s: the gradient reader never names %s", swiftRenderer, k)
		}
	}
	for _, want := range []string{
		"if let gradient = grMobCanvasGradient(props)",
		"local.concatenate(toBox)",
		"path.applying(toBox.inverted())",
		// A gradient stroke fills the stroke's box-space outline, which is
		// what keeps its width in points while the colours follow the viewBox.
		`grMobCanvasGradient(props, prefix: "stroke")`,
		"path: path.strokedPath(strokeStyle)",
		`props["strokeGradient"] != nil`,
	} {
		if !strings.Contains(swift, want) {
			t.Errorf("%s: missing %q", swiftRenderer, want)
		}
	}

	kotlinPath := nativeFile("android", "app", "src", "main", "java", "com", "grmob", "runtime", "GrMobCanvas.kt")
	kotlin := valuesIn(t, kotlinPath)
	for _, k := range keys {
		if !strings.Contains(kotlin, k) {
			t.Errorf("GrMobCanvas.kt: the gradient reader never names %s", k)
		}
	}
	for _, want := range []string{
		"val gradient = canvasGradientBrush(props, vp)",
		"shader.setLocalMatrix(matrix)",
		"tileMode = TileMode.Clamp",
		`canvasGradientBrush(props, vp, prefix = "stroke")`,
		"drawPath(path, strokeBrush, style = strokeStyle)",
	} {
		if !strings.Contains(kotlin, want) {
			t.Errorf("GrMobCanvas.kt: missing %q", want)
		}
	}
}
