package verify

import (
	"strings"
	"testing"
)

// core.BorderWidth/BorderColor on a core.Button, held against both native
// renderers.
//
// # What was wrong
//
// A Button is one of the few node types that draws its own container, so both
// renderers hand it a style stripped of the box-drawing fields — Compose's
// marginAndSize, SwiftUI's marginAndSizeOnly, each of which clears background,
// border, radius, shadow and padding and feeds them into the platform
// control's own slots instead. Background, radius and padding were fed back in.
// The border was not: it was stripped and then dropped, so the one modifier
// that would have drawn it (Modifier.border in boxModifier, grMobBorder in
// grMobBox) never ran for a Button on either platform.
//
// The visible cost was components.Button's EmphasisOutlined, documented as "a
// transparent fill, a 1px rule and a label both in the variant's color" and
// drawing its rule on the two web targets and nothing at all on either phone.
// It is the mirror image of the gap borderResetTypes (htmlout/tag.go) closes in
// the other direction, and the two were fixed together because they are one
// disagreement: whether "BorderWidth > 0 && BorderColor != nil" decides the
// border on all four targets or only on two.
//
// # Why this is a source check
//
// A stripped-then-dropped field is not a type error in either language, and
// neither native runs under `go test ./...` — the same reason gap_test.go and
// switchlabels_test.go read source. What is pinned is that each renderer
// *consults* the border for a Button, since a renderer that compiles and
// silently flattens every outlined button is exactly the failure this package
// exists to notice.

// The Kotlin half. material3's Button and the Surface the long-press path
// rebuilds it out of both take a `border` slot, and both have to use it: an
// outlined button must not lose its rule the moment it grows an OnLongPress.
func TestKotlinGivesAButtonItsBorder(t *testing.T) {
	src := readNative(t, kotlinRenderer)

	if !strings.Contains(src, "private fun borderStroke(s: GrMobStyle?): BorderStroke?") {
		t.Fatalf("%s: no borderStroke helper — if it was renamed, update this test rather "+
			"than deleting it", kotlinRenderer)
	}
	// Both button paths, counted rather than merely found: one match would
	// mean the material3 path took the border and the long-press rebuild did
	// not, which is the divergence that would be hardest to notice.
	if n := strings.Count(src, "border = borderStroke(s),"); n != 2 {
		t.Errorf("%s: border = borderStroke(s) appears %d times, want 2 (GrMobButton's "+
			"material3 Button and GrMobLongPressButton's Surface) — a Button that draws its "+
			"own container gets no border from boxModifier, so each path has to ask", kotlinRenderer, n)
	}

	// The guard, which is boxModifier's restated: both halves are required on
	// every target, so a Button must not be the one place where a width with
	// no color draws something.
	body := declSource(t, kotlinRenderer, "private fun borderStroke(s: GrMobStyle?): BorderStroke?")
	for _, expr := range []string{
		"if (s == null) return null",
		"val color = s.borderColor ?: return null",
		"if (s.borderWidth <= 0f) return null",
	} {
		if !strings.Contains(body, expr) {
			t.Errorf("%s: borderStroke is missing %q — half a border draws nothing on the "+
				"other three targets and must draw nothing here", kotlinRenderer, expr)
		}
	}
}

// The Swift half. GrMobButtonStyle is where a Button's container is actually
// drawn, so the border has to travel into it alongside the background and the
// radius rather than through grMobBox, which this view is handed a stripped
// style for.
func TestSwiftGivesAButtonItsBorder(t *testing.T) {
	src := readNative(t, swiftRenderer)
	for _, pin := range []struct{ expr, why string }{
		{"borderColor: s?.borderColor,",
			"the color reaching the button style. marginAndSizeOnly clears it before " +
				"grMobBox ever sees it, so this is the only route left"},
		{"borderWidth: s?.borderWidth ?? 0",
			"the width, which grMobBorder needs as well — either half missing is no border"},
		{".grMobBorder(RoundedRectangle(cornerRadius: radius), color: borderColor, width: borderWidth)",
			"the stroke itself, on the same shape the fill was clipped to, so the rule lands " +
				"on the edge rather than inside or outside it"},
	} {
		if !strings.Contains(src, pin.expr) {
			t.Errorf("%s: %q not found — %s", swiftRenderer, pin.expr, pin.why)
		}
	}

	// grMobBorder lives in GrMobStyle.swift and had been fileprivate, which
	// put it out of reach of the file that needs it. Swift's access control
	// is per *file*, so this is not a formality: making it fileprivate again
	// would break the build, but making a second copy of the strokeBorder
	// logic here would not, and that is the outcome worth pinning against.
	style := readNative(t, swiftStyle)
	if strings.Contains(style, "fileprivate func grMobBorder(") {
		t.Errorf("%s: grMobBorder is fileprivate again — Renderer.swift's GrMobButtonStyle "+
			"needs the same stroke, and a second copy of it would be a second border rule "+
			"to keep in step with Compose's", swiftStyle)
	}
	if !strings.Contains(style, "strokeBorder(color, lineWidth: width)") {
		t.Errorf("%s: grMobBorder no longer insets its stroke — strokeBorder is what matches "+
			"Compose's Modifier.border placement, where a plain stroke straddles the edge",
			swiftStyle)
	}
}
