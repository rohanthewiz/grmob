package verify

import (
	"strings"
	"testing"
)

// core.MaxWidth on both native renderers.
//
// For two releases the field crossed the bridge and nothing read it: the web
// pair wrote `max-width` and Compose and SwiftUI dropped it, so a dialog card
// capped at 480px for a wide browser spanned a tablet, and comps documented
// the gap ("MaxWidth is read by the web targets only"). Neither native runs
// under `go test ./...`, so a parser that stops reading the key, or a chain
// that stops applying it, is a silent regression — the same class the Rotate
// and gap checks catch, caught the same way.
//
// The arithmetic (which strings are a cap, what a percentage resolves against,
// how a cap meets a rigid Width, how it clamps the min-content floor) is run on
// macOS by ios/verify; these checks hold the parts that can only be read: that
// the value is parsed, that it is applied, and that it is applied at the layer
// where the constraint model lets it bind.

var (
	maxWidthHarnessMain = nativeFile("ios", "verify", "main.swift")
	maxWidthHarnessFlex = nativeFile("ios", "verify", "flex.swift")
	maxWidthMinContent  = nativeFile("ios", "GrMob", "Runtime", "GrMobMinContent.swift")
)

// Both parsers must read the key off the wire.
func TestBothNativeParsersReadMaxWidth(t *testing.T) {
	for _, pin := range []struct{ file, key string }{
		{swiftStyle, `str("MaxWidth")`},
		{kotlinStyle, `optString("MaxWidth")`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: does not parse %s — core.MaxWidth crosses the bridge and "+
				"is dropped on this target", pin.file, pin.key)
		}
	}
}

// Compose: Width and MaxWidth go through ONE layout modifier, placed where
// Width used to be.
//
// The shape is the point. Two independent size modifiers lose a case in every
// ordering (widthModifier's table), and the case lost by the obvious order —
// a stretched Column child, which arrives with minWidth == maxWidth — is the
// commonest capped node there is: a dialog card. So the check is not only that
// the cap is applied but that it relaxes the incoming minimum, reports inside
// the incoming constraints, and places at the start.
func TestComposeResolvesWidthAndMaxWidthTogether(t *testing.T) {
	kotlin := codeIn(t, kotlinStyle)
	call := strings.Index(kotlin, "m = m.then(widthModifier(width, maxWidth, minWidth))")
	if call < 0 {
		t.Fatalf("%s: boxModifier does not call widthModifier(width, maxWidth, minWidth) — "+
			"core.MaxWidth parses and caps nothing", kotlinStyle)
	}
	if strings.Contains(kotlin, "m = m.then(dimensionModifier(width, horizontal = true))") {
		t.Errorf("%s: boxModifier still applies Width on its own beside widthModifier, "+
			"so the width is sized twice and the cap loses to whichever runs outside",
			kotlinStyle)
	}

	// Inside the margin (CSS caps the border box) and alongside the height.
	margin := strings.Index(kotlin, "start = margin.left.dp")
	height := strings.Index(kotlin, "m = m.then(heightModifier(height, minHeight))")
	if margin < 0 || height < 0 {
		t.Fatalf("%s: boxModifier was restructured; update this test rather than "+
			"deleting it (margin=%d height=%d)", kotlinStyle, margin, height)
	}
	if call < margin {
		t.Errorf("%s: widthModifier runs outside the margin, so the reserved space "+
			"counts against the cap", kotlinStyle)
	}
	if call > height {
		t.Errorf("%s: widthModifier has moved below the height modifier; the two "+
			"dimension layers are meant to sit together", kotlinStyle)
	}

	start := strings.Index(kotlin, "private fun widthModifier(")
	if start < 0 {
		t.Fatalf("%s: no widthModifier declaration", kotlinStyle)
	}
	end := strings.Index(kotlin[start:], "\n}\n")
	if end < 0 {
		t.Fatalf("%s: widthModifier has no closing brace at column 0", kotlinStyle)
	}
	body := kotlin[start : start+end]
	for _, want := range []struct{ code, why string }{
		{"return dimensionModifier(width, horizontal = true)",
			"an uncapped node must take the plain size modifier, unchanged"},
		{"Modifier.layout {", "the cap must be a layout modifier; a size modifier cannot relax a minimum"},
		{"minW = minOf(minW, maxW)", "a forced minimum (a stretched child) must be relaxed to the cap"},
		{"placeable.width.coerceIn(constraints.minWidth, constraints.maxWidth)",
			"the reported width must stay inside the incoming constraints"},
		{"placeRelative(0, 0)", "a capped box sits at the start of a wider slot, mirrored under RTL"},
		{"constraints.hasBoundedWidth", "a percentage cap of an unbounded width must behave as none"},
	} {
		if !strings.Contains(body, want.code) {
			t.Errorf("%s: widthModifier lacks %q — %s", kotlinStyle, want.code, want.why)
		}
	}
}

// SwiftUI: the cap is a Layout outside grMobGrow, and it is folded into the
// rigid Width frames.
//
// Outside grMobGrow because a stretched child's fill is a flexible frame that
// takes whatever it is proposed: capped inside it, the child fills the whole
// extent and draws its capped content somewhere within. A Layout, not a
// `.frame(maxWidth:)`, because a flexible frame with only a maximum is greedy
// and grows a hugging box to the cap. Folded into grMobDimension because a
// rigid frame ignores the narrowed proposal and would spill past it.
func TestSwiftCapsOutsideTheGrowFrame(t *testing.T) {
	swift := codeIn(t, swiftStyle)
	grow := strings.Index(swift, ".grMobGrow(grow, alignment: alignment)")
	if grow < 0 {
		t.Fatalf("%s: grMobBox was restructured; update this test rather than deleting it",
			swiftStyle)
	}
	rest := swift[grow:]
	capAt := strings.Index(rest, ".modifier(GrMobMaxWidthModifier(")
	opacityAt := strings.Index(rest, ".opacity(")
	if capAt < 0 {
		t.Fatalf("%s: no GrMobMaxWidthModifier after grMobGrow — core.MaxWidth parses and "+
			"caps nothing, or it was moved inside the grow frame where a stretched "+
			"child ignores it", swiftStyle)
	}
	if opacityAt >= 0 && capAt > opacityAt {
		t.Errorf("%s: GrMobMaxWidthModifier has drifted past the sizing layers", swiftStyle)
	}

	for _, want := range []struct{ code, why string }{
		{"struct GrMobMaxWidthLayout: Layout", "the cap must be a Layout, not a greedy flexible frame"},
		{"min(size.width, bound)", "the layout must report what the child took, never more than the cap"},
		{"cap: GrMobMaxWidth.fixedLimit(", "a rigid Width frame must be clamped by a points cap"},
		{"GrMobMaxWidthLayout(value: value, margin: margin) { content }",
			"the modifier must wrap its concrete content in the layout (see its doc for why it is a modifier)"},
	} {
		if !strings.Contains(swift, want.code) {
			t.Errorf("%s: lacks %q — %s", swiftStyle, want.code, want.why)
		}
	}
}

// The arithmetic lives where ios/verify can run it, and ios/verify runs it.
func TestMaxWidthArithmeticIsCheckedOnMacOS(t *testing.T) {
	if !strings.Contains(codeIn(t, swiftFlex), "enum GrMobMaxWidth") {
		t.Errorf("%s: GrMobMaxWidth has left the CoreGraphics-only file ios/verify "+
			"compiles into its harness", swiftFlex)
	}
	if !strings.Contains(codeIn(t, maxWidthHarnessFlex), "func checkMaxWidth()") {
		t.Errorf("%s: no checkMaxWidth — the cap's arithmetic is unchecked", maxWidthHarnessFlex)
	}
	if !strings.Contains(codeIn(t, maxWidthHarnessMain), "checkMaxWidth()") {
		t.Errorf("%s: checkMaxWidth is declared and never run", maxWidthHarnessMain)
	}
	if !strings.Contains(codeIn(t, maxWidthMinContent), "GrMobMaxWidth.fixedLimit(") {
		t.Errorf("%s: the min-content floor ignores core.MaxWidth, so a capped box "+
			"holds a row open wider than it can be drawn", maxWidthMinContent)
	}
}

// core.MinWidth and core.MinHeight on both natives.
//
// Documented as web-only for as long as MaxWidth was, while three widgets
// leaned on them: comps.DatePicker's sheet (MinWidth 300, so the grid's
// flex-basis-0 cells have a width to divide), the rich-text link prompt (280)
// and comps.RichTextEditor's MinHeight (an empty editor still a place to
// write). The same three links as MaxWidth: parsed, applied, and applied where
// the constraint model lets a floor bind.
func TestBothNativesApplyMinWidthAndMinHeight(t *testing.T) {
	for _, pin := range []struct{ file, key string }{
		{swiftStyle, `str("MinWidth")`},
		{swiftStyle, `str("MinHeight")`},
		{kotlinStyle, `optString("MinWidth")`},
		{kotlinStyle, `optString("MinHeight")`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: does not parse %s — the floor crosses the bridge and is dropped",
				pin.file, pin.key)
		}
	}

	// Compose: the width floor is folded into widthModifier's constraints
	// after the cap (CSS's min-width beats max-width), and the height floor is
	// a layout modifier of its own for the same reason the cap is one: a size
	// modifier cannot raise a minimum a weight or stretch already fixed.
	kotlin := codeIn(t, kotlinStyle)
	body := func(decl string) string {
		start := strings.Index(kotlin, decl)
		if start < 0 {
			t.Fatalf("%s: no %s", kotlinStyle, decl)
		}
		end := strings.Index(kotlin[start:], "\n}\n")
		if end < 0 {
			t.Fatalf("%s: %s has no closing brace at column 0", kotlinStyle, decl)
		}
		return kotlin[start : start+end]
	}
	width := body("private fun widthModifier(")
	limitAt := strings.Index(width, "maxW = minOf(maxW, limit)")
	floorAt := strings.Index(width, "minW = maxOf(minW, least)")
	if floorAt < 0 || strings.Index(width, "maxW = maxOf(maxW, least)") < 0 {
		t.Errorf("%s: widthModifier does not fold MinWidth into the measured constraints", kotlinStyle)
	} else if limitAt < 0 || floorAt < limitAt {
		t.Errorf("%s: widthModifier applies the floor before the cap; min-width must win", kotlinStyle)
	}
	height := body("private fun heightModifier(")
	for _, want := range []string{
		"return dimensionModifier(height, horizontal = false)",
		"Modifier.layout {",
		"minHeight = maxOf(constraints.minHeight, least)",
		"placeable.height.coerceIn(constraints.minHeight, constraints.maxHeight)",
	} {
		if !strings.Contains(height, want) {
			t.Errorf("%s: heightModifier lacks %q", kotlinStyle, want)
		}
	}

	// SwiftUI: a flexible frame's minimums, inside the background so the fill
	// and the border cover the floored box.
	// valuesIn, not codeIn: the anchors hold string literals ("").
	swift := valuesIn(t, swiftStyle)
	dim := strings.Index(swift, `.grMobDimension(s?.height ?? "", axis: .vertical, alignment: alignment)`)
	floor := strings.Index(swift, `.grMobMinimum(width: s?.minWidth ?? "", height: s?.minHeight ?? "", alignment: alignment)`)
	bg := strings.Index(swift, `.background(s?.background ?? .clear)`)
	if dim < 0 || floor < 0 || bg < 0 {
		t.Fatalf("%s: grMobBox does not apply grMobMinimum beside its dimensions "+
			"(dimension=%d minimum=%d background=%d)", swiftStyle, dim, floor, bg)
	}
	if !(dim < floor && floor < bg) {
		t.Errorf("%s: grMobMinimum must sit between the declared dimensions and the "+
			"background", swiftStyle)
	}
	if !strings.Contains(swift, "frame(minWidth: w, minHeight: h, alignment: alignment)") {
		t.Errorf("%s: grMobMinimum no longer applies a flexible frame's minimums", swiftStyle)
	}
}
