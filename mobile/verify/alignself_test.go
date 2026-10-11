package verify

import (
	"strings"
	"testing"
)

// core.Style.AlignSelf on both native renderers (N-100).
//
// For as long as the field existed only the two DOM targets read it, and the
// style reference said so. comps leaned on it anyway, in two places where the
// natives' silence was visible:
//
//   - comps.Discussion's thread line is an empty 2px Box with
//     AlignSelf(stretch) beside the replies. Neither phone drew it: the
//     Box had no height of its own and nothing stretched it (the emulator
//     and the iOS 26.5 simulator, lesson 4.40, 2026-10-10).
//   - comps.Link hugs its text with AlignSelf(start), so that a tap beside
//     "Terms" is not a tap on the link. On both phones the stretching
//     Column spread it across the card (lesson 4.39's "Read on Blue Letter
//     Bible", 708px of tap target on Android for about 485px of text).
//
// The rule both natives now apply is CSS's: a child's own value overrides the
// container's AlignItems for it alone, and "" defers. Its arithmetic runs on
// macOS in ios/verify (flex.swift's selfAlign and selfStretches cases); these
// checks hold the parts only a device could otherwise show — that the key is
// parsed, that each dispatch answers every value, and that each container
// reads the child's value through the shared rule on both of its channels.

// Both parsers must read the key off the wire.
func TestBothNativeParsersReadAlignSelf(t *testing.T) {
	for _, pin := range []struct{ file, key string }{
		{swiftStyle, `str("AlignSelf")`},
		{kotlinStyle, `optString("AlignSelf")`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: does not parse %s — core.AlignSelf crosses the bridge and is dropped on this target",
				pin.file, pin.key)
		}
	}
}

// Compose's two placements, one per axis, each held to core.AlignItemsValues().
// "stretch" maps to null in both on purpose (it is a fill, not a placement),
// and the arm must still be spelled out so that a fifth value added to core
// is a failing test rather than a silent `else`.
func TestKotlinAlignSelfCoversEveryAlignItems(t *testing.T) {
	for _, c := range []struct{ anchor, fn, consequence string }{
		{"fun rowSelfAlignment(", "rowSelfAlignment",
			"leaves a self-aligned Row child where the row puts every child"},
		{"fun columnSelfAlignment(", "columnSelfAlignment",
			"leaves a self-aligned Column child where the column puts every child"},
	} {
		syntax := kotlinWhen.with(kotlinRenderer, c.anchor, "when (alignSelf) {")
		coverage{
			file:        "Renderer.kt",
			fn:          c.fn,
			required:    alignItemsValues(),
			consequence: c.consequence,
		}.check(t, syntax.labels(t))
	}
}

// Each container must read the child's own value through the shared rule, on
// both channels. On iOS the two channels are FlexChildren (the fill frame the
// child accepts) and GrMobFlexLayout (the cross extent it is proposed and
// where it is placed). If either read the container's value alone, the
// layout would propose a fill no frame accepts, or the reverse, and the
// child would draw at a size its slot does not have. On Android the fill and
// the placement are both RowChildren's or ColumnChildren's, and the Row must
// also pin its height when a child stretches itself (rowPinsHeight), or
// fillMaxHeight has nothing definite to fill in a scrolled page.
func TestEachContainerReadsTheChildsAlignSelf(t *testing.T) {
	for _, pin := range []struct{ file, anchor, code, why string }{
		{swiftRenderer, "struct FlexChildren",
			"GrMobFlexSolver.selfStretches(own, containerStretches: stretch)",
			"the fill frame must follow the child's own AlignSelf"},
		{swiftRenderer, "struct FlexChildren",
			".layoutValue(key: GrMobFlexAlignSelf.self, value: own)",
			"the layout cannot see the node, so the value must reach it as a layout value"},
		{swiftRenderer, "",
			"let childCross = stretches && !subview[GrMobFlexHugs.self]",
			"GrMobFlexLayout's proposal must follow the same rule the fill frame does"},
		{swiftRenderer, "",
			"GrMobFlexSolver.selfAlign(own, container: crossAlign)",
			"GrMobFlexLayout's cross offset must place the child by its own value"},
		{swiftRenderer, "",
			"GrMobFlexSolver.selfAlign(subviews[i][GrMobFlexAlignSelf.self], container: crossAlign)",
			"a wrapped child must sit in its line by its own value"},
		{kotlinRenderer, "fun RowScope.RowChildren(", "selfStretches(own, stretch)",
			"a Row child's fill must follow its own AlignSelf"},
		{kotlinRenderer, "fun RowScope.RowChildren(", "rowSelfAlignment(own)",
			"a Row child must be placed by its own AlignSelf"},
		{kotlinRenderer, "fun ColumnScope.ColumnChildren(", "selfStretches(own, stretch)",
			"a Column child's fill must follow its own AlignSelf"},
		{kotlinRenderer, "fun ColumnScope.ColumnChildren(", "columnSelfAlignment(own)",
			"a Column child must be placed by its own AlignSelf"},
		{kotlinRenderer, "fun rowPinsHeight(", `it.style?.alignSelf == "stretch"`,
			"a Row must pin its height when a child stretches itself, or the fill has no height to fill"},
		{kotlinRenderer, "fun rowPinsHeight(", "answersIntrinsicWidth(node)",
			"the pin's intrinsic query must be skipped when a child is a List or a vertical Scroll, which throw"},
		{kotlinRenderer, "fun ColumnScope.ColumnChildren(", "m = m.fitContentWidth()",
			"a self-placed container child must be fit-content, or its stretched children fill the column through it"},
		{swiftRenderer, "", "proposed(main: childMain, cross: fitCross(subview, main: childMain, bound: containerCross))",
			"a self-placed child must be proposed its fit-content cross size, or a container child fills the extent"},
	} {
		// An anchor of "" reads the whole file: a Layout's placeSubviews has
		// no anchor of its own (every Layout declares one, and the cut stops
		// at a struct's first member), and these expressions occur once. A
		// pin with a string literal in it reads the literals intact.
		var src string
		switch {
		case pin.anchor == "":
			src = codeIn(t, pin.file)
		case strings.Contains(pin.code, `"`):
			src = valuesOf(t, pin.file, pin.anchor)
		default:
			src = codeOf(t, pin.file, pin.anchor)
		}
		if !strings.Contains(src, pin.code) {
			t.Errorf("%s: %s lacks %q — %s", pin.file, orWholeFile(pin.anchor), pin.code, pin.why)
		}
	}
}

func orWholeFile(anchor string) string {
	if anchor == "" {
		return "the file"
	}
	return anchor
}

// The rule is run where it can run: ios/verify's flex section asks both
// halves of it, so a change to either is a failing macOS pass.
func TestAlignSelfRuleIsCheckedOnMacOS(t *testing.T) {
	src := valuesIn(t, maxWidthHarnessFlex)
	for _, call := range []string{"GrMobFlexSolver.selfAlign(", "GrMobFlexSolver.selfStretches("} {
		if !strings.Contains(src, call) {
			t.Errorf("%s: never calls %s — the AlignSelf rule is unchecked on macOS", maxWidthHarnessFlex, call)
		}
	}
}
