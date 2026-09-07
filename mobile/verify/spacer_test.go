package verify

import (
	"regexp"
	"strings"
	"testing"
)

// core.Spacer's own Style, on the two targets that used to drop it.
//
// A Spacer is the one node type in the vocabulary whose *size* arrives as a
// prop rather than as a Style declaration, and that is what made it the one
// node type whose Style went missing. Both natives spelled the arm as a single
// expression built from the prop — `Color.clear.frame(width:height:)` and
// `Spacer(Modifier.size(n.dp))` — with no call to the renderer's own box
// helper anywhere in it, so a hand-assembled Spacer carrying a Background, a
// Margin, an AccessibilityLabel or an OnTap got none of them. htmlout had the
// same hole and closed it a session earlier by moving its branch into the
// shared attribute assembly; the WASM runtime never had it.
//
// Nothing could notice. The arm compiles, the node draws, and the props it
// silently ignores are exactly the ones nobody sets on a Spacer — because
// core.Spacer(n) cannot set them, which is the argument core.Modal's chassis
// sits under too. That leaves a source check as the only thing that can hold
// the two renderers to it, and it is worth having for the reason the whole
// package is: the next person to add a leaf node type reaches for the arm that
// is shortest to write.

// Both arms must route the node through the renderer's box helper.
//
// The claim is deliberately "the box is built" rather than "the background is
// applied": grMobBox and boxModifier are each a single funnel — margin, size,
// shadow, clip, fill, border, padding, the accessibility semantics and the
// platform disabled state — so reaching one is what makes all of it arrive,
// and naming any single declaration here would be a check that passes while
// the other ten stay dropped.
func TestBothNativeSpacersBuildTheNodesOwnBox(t *testing.T) {
	swift := dispatchArm(t, swiftRenderer, `case "Spacer":`,
		regexp.MustCompile(`\n\s+case "`))
	if !strings.Contains(swift, "GrMobSpacer(node: node") {
		t.Errorf("%s: the Spacer arm does not route through GrMobSpacer — %s",
			swiftRenderer, spacerWhy)
	}
	if !strings.Contains(spacerView(t), ".grMobBox(s, grow: grow,") {
		t.Errorf("%s: GrMobSpacer does not call grMobBox — %s", swiftRenderer, spacerWhy)
	}

	// The Compose arm reaches boxModifier through GrMobColumn now rather than
	// spelling it — the same indirection the Swift half has always had, and
	// for the same reason: the arm names a composite, and the composite builds
	// the box. Both ends are checked, exactly as the Swift half checks the arm
	// and then GrMobSpacer.
	kotlin := dispatchArm(t, kotlinRenderer, `"Spacer" ->`,
		regexp.MustCompile(`\n\s+"[A-Za-z]+"(, "[A-Za-z]+")* ->`))
	if !strings.Contains(kotlin, "GrMobColumn(") {
		t.Errorf("%s: the Spacer arm does not route through GrMobColumn — %s",
			kotlinRenderer, spacerWhy)
	}
	if !strings.Contains(
		dispatchArm(t, kotlinRenderer, "private fun GrMobColumn", kotlinCompositeStart),
		"s.boxModifier(extra",
	) {
		t.Errorf("%s: GrMobColumn does not call boxModifier — %s",
			kotlinRenderer, spacerWhy)
	}
}

const spacerWhy = "a hand-assembled Spacer's Background, Margin, " +
	"AccessibilityLabel and OnTap are dropped on this target and honoured on " +
	"both DOM targets"

// The size prop is the node type's fixed look and must go *underneath* the
// author's declarations, which on both natives means innermost.
//
// This is the half a "calls the box helper" check cannot see. An arm that
// applied the chassis outside the box would compile, would honour the
// Background, and would silently make a Spacer the one node in the framework
// where a type default outranks an author — the exact inversion htmlout's
// spacerChassis and the WASM runtime's applySpacerChassis were both written to
// undo, and which the runtime's own history records as the thing that shipped
// first.
//
// The two languages read in opposite directions and so do their constraint
// rules, which is why the pin is per-target rather than one shared string:
//
//	SwiftUI   later in the chain is further out, and an *outer* frame wins.
//	          So the chassis frame is written before .grMobBox — and on an
//	          axis the Style claims it is not written at all (nil), which is
//	          what keeps Color.clear flexible there so the background fills
//	          the frame grMobBox puts around it.
//	Compose   constraints flow outside-in and an inner size() coerces itself
//	          into what it was handed, so the author's dimension modifiers
//	          winning is exactly "boxModifier first, .size() after".
func TestTheSpacerChassisSitsUnderTheAuthorsStyle(t *testing.T) {
	swift := spacerView(t)
	frameAt := strings.Index(swift, ".frame(width: spacerExtent(size, stated:")
	boxAt := strings.Index(swift, ".grMobBox(s, grow: grow,")
	if frameAt < 0 || boxAt < 0 {
		t.Fatalf("%s: GrMobSpacer was restructured; update this test rather than "+
			"deleting it (frame=%d box=%d)", swiftRenderer, frameAt, boxAt)
	}
	if frameAt > boxAt {
		t.Errorf("%s: the size frame is applied outside grMobBox, so the prop "+
			"outranks a Style that stated a Width or a Height", swiftRenderer)
	}
	// nil on a claimed axis is the whole mechanism, and it is one `guard` away
	// from being dropped: `return CGFloat(size)` unconditionally still
	// compiles and still passes the ordering check above.
	if !strings.Contains(swift, `guard stated.isEmpty || stated == "auto" else { return nil }`) {
		t.Errorf("%s: spacerExtent no longer yields the axis the Style claims — a "+
			"fixed clear inside a stated frame paints the chassis's square in the "+
			"author's hole", swiftRenderer)
	}

	// The Compose half is now read in two pieces, because it is written in
	// two. The arm used to spell the whole chain itself
	// (`boxModifier(...).size(...)`); it hands the node to GrMobColumn so that
	// a hand-assembled Spacer's children are stacked rather than dropped, and
	// the chassis rides in as that composite's `outer` modifier. The ordering
	// claim is unchanged and is still one `.then` — it just lives one call
	// down, so both ends are checked or the pin would be satisfied by an arm
	// that passed the size to a composite which applied it first.
	kotlin := dispatchArm(t, kotlinRenderer, `"Spacer" ->`,
		regexp.MustCompile(`\n\s+"[A-Za-z]+"(, "[A-Za-z]+")* ->`))
	if !strings.Contains(kotlin, `outer = Modifier.size(node.intProp("size").dp)`) {
		t.Fatalf("%s: the Spacer arm no longer hands the size in as GrMobColumn's "+
			"outer modifier; update this test rather than deleting it:\n%s",
			kotlinRenderer, kotlin)
	}
	column := dispatchArm(t, kotlinRenderer, "private fun GrMobColumn", kotlinCompositeStart)
	boxIdx := strings.Index(column, "s.boxModifier(extra")
	sizeIdx := strings.Index(column, ".then(outer)")
	if boxIdx < 0 || sizeIdx < 0 {
		t.Fatalf("%s: GrMobColumn was restructured; update this test rather "+
			"than deleting it (box=%d then=%d)", kotlinRenderer, boxIdx, sizeIdx)
	}
	if sizeIdx < boxIdx {
		t.Errorf("%s: GrMobColumn applies its outer modifier before boxModifier, so "+
			"a Spacer's size prop fixes the constraints and a Style that stated a "+
			"Width is coerced into the chassis rather than the other way round",
			kotlinRenderer)
	}
}

// The GrMobSpacer declaration and the spacerExtent helper below it, cut out of
// Renderer.swift so the ordering check reads this view's chain and not the
// nineteen other `.grMobBox(` calls in the file. `private func spacerExtent`
// is the next top-level declaration after the struct, so the cut ends there
// and still contains both halves of the mechanism.
func spacerView(t *testing.T) string {
	t.Helper()
	src := readNative(t, swiftRenderer)
	at := strings.Index(src, "private struct GrMobSpacer: View {")
	if at < 0 {
		t.Fatalf("%s: no GrMobSpacer — if it was renamed, update this test rather "+
			"than deleting it", swiftRenderer)
	}
	rest := src[at:]
	end := strings.Index(rest, "\nprivate struct ")
	if end < 0 {
		t.Fatalf("%s: GrMobSpacer is the last declaration in the file; this cut "+
			"assumed another followed it", swiftRenderer)
	}
	return rest[:end]
}
