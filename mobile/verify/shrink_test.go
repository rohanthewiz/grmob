package verify

import (
	"strings"
	"testing"
)

// core.FlexShrink(0) must reach a layout decision on both natives.
//
// # What this is the other half of
//
// wasm/verify's shrink_test.go pins the NUMBER: core.ShrinkNone is -1, and all
// four spellings outside Go — two JavaScript, one Swift, one Kotlin — must read
// that same number. A parser that reads the sentinel correctly and hands the
// answer to nothing passes every one of those checks, which is exactly the
// state both DOM targets were in before the sentinel existed: the prop
// compiled, applied, serialised, and did nothing.
//
// So this file pins the call site. It is the same division of labour the gap
// longhands have one file over (TestBothNativeParsersReadTheGapLonghands reads
// the parser, TestNativeContainersSpaceAlongTheirOwnAxis reads the container),
// and it is here rather than in wasm/verify because a call site is native
// source, which is this package's whole subject.
//
// # The two targets do different things with it, and both are the contract
//
//	SwiftUI   GrMobFlexSolver takes a per-item shrink factor and implements
//	          CSS's scaled-base rule over it, so every factor means what CSS
//	          says it means. The renderer's job is to hand the factor over.
//
//	Compose   a Row has no proportional shrink at all — it measures each
//	          unweighted child against the main-axis space the ones before it
//	          did not take — so a fractional factor has nothing to map onto.
//	          Zero is not a proportion but a refusal, and a refusal IS
//	          expressible: measure the child unbounded and report its own
//	          size. That is Modifier.pinMainAxis, and it is what makes
//	          core.FlexShrink(0) mean the same thing on the fourth target as
//	          on the other three.

// The SwiftUI renderer must hand the solver the reading, not the raw field.
//
// `shrinkFactor` and `flexShrink` are one character apart at a call site and
// mean opposite things for the two values that matter: an unset field is 0 and
// must become a factor of 1, and core.ShrinkNone is -1 and must become 0. A
// renderer that passed the field straight through would pin every node in
// every layout (0 read as a factor) and let the one node that asked to be
// pinned shrink harder than its siblings (-1 read as a factor), and both
// layouts would still be produced by code that compiles.
func TestTheSwiftUIRowHandsTheSolverTheShrinkReading(t *testing.T) {
	body := codeOf(t, swiftRenderer, "private struct FlexChildren: View {")
	if !strings.Contains(body, "GrMobFlexShrink.self") {
		t.Errorf("%s: FlexChildren no longer attaches a GrMobFlexShrink layout value. "+
			"GrMobFlexSolver's shrink arm takes a per-item factor and defaults it to 1; "+
			"with nothing attached, core.FlexShrink is inert on this target and the "+
			"only thing saying otherwise is core.ShrinkNone's doc comment.", swiftRenderer)
	}
	if !strings.Contains(body, "child.style?.shrinkFactor") {
		t.Errorf("%s: FlexChildren does not read child.style?.shrinkFactor. The raw "+
			"flexShrink field is the JSON as written — 0 for unset, -1 for "+
			"core.ShrinkNone — and passing it as a factor pins every node in every "+
			"layout while unpinning the one node that asked.", swiftRenderer)
	}
}

// The Compose renderer must apply pinMainAxis, on the right axis, from both
// children loops.
//
// The axis argument is the half worth pinning by value. A Row's main axis is
// its width and a Column's is its height; transposing the two leaves a pinned
// child unbounded across the axis nothing was measuring it on, so it is
// squeezed exactly as before and the declaration goes back to doing nothing —
// with the modifier still applied, still named, and still compiling.
func TestTheComposeChildrenLoopsPinOnTheirOwnAxis(t *testing.T) {
	for _, loop := range []struct {
		anchor, want, axis string
	}{
		{"private fun RowScope.RowChildren(", "pinMainAxis(horizontal = true)",
			"a Row stacks along its width"},
		{"private fun ColumnScope.ColumnChildren(", "pinMainAxis(horizontal = false)",
			"a Column stacks along its height"},
	} {
		body := codeOf(t, kotlinRenderer, loop.anchor)
		if !strings.Contains(body, "shrinkPinned") {
			t.Errorf("%s: %s never reads shrinkPinned — core.FlexShrink(0) parses on "+
				"this target and reaches no layout decision, which is the state the "+
				"other three targets were in before core.ShrinkNone existed",
				kotlinRenderer, loop.anchor)
		}
		if !strings.Contains(body, loop.want) {
			t.Errorf("%s: %s does not apply %s (%s). A pin on the cross axis is not a "+
				"weaker version of a pin on the main one — it is no pin at all, since "+
				"nothing was constraining that axis, and the child is squeezed exactly "+
				"as it was.", kotlinRenderer, loop.anchor, loop.want, loop.axis)
		}
	}
}

// And pinMainAxis must do the two things that make it a pin.
//
// Both halves are load-bearing and each fails silently on its own:
//
//	measure unbounded   without it the child is measured against the space that
//	                    is left, which is the squeeze the declaration exists to
//	                    refuse
//
//	report the measured a `layout(constraints.constrain(...))` compiles, looks
//	size                more correct — a well-behaved layout respects its
//	                    constraints — and clamps the size the parent is told
//	                    about, so the Row's running total never exceeds its own
//	                    maximum, nothing overflows, and the child is drawn
//	                    clipped to a box the parent still believes fits
func TestTheComposePinMeasuresUnboundedAndReportsWhatItMeasured(t *testing.T) {
	body := codeOf(t, kotlinRenderer, "private fun Modifier.pinMainAxis(")
	for _, want := range []string{
		"maxWidth = Constraints.Infinity",
		"maxHeight = Constraints.Infinity",
		"layout(placeable.width, placeable.height)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("%s: pinMainAxis does not contain %q. Without it the modifier is "+
				"applied, named after what it no longer does, and core.FlexShrink(0) is "+
				"inert on this target again.", kotlinRenderer, want)
		}
	}
	if strings.Contains(body, "constraints.constrain(") {
		t.Errorf("%s: pinMainAxis constrains the size it reports. The parent then "+
			"learns a size that fits, so its running total never overflows and the "+
			"pinned child is drawn spilling out of a box the Row believes it fits in "+
			"— which is neither the squeeze nor the overflow, and matches no target.",
			kotlinRenderer)
	}
}

// codeOf is declSource with the prose taken out.
//
// Every check in this file asks whether a renderer *does* something, and
// declSource's cut is deliberately coarse — it runs from an anchor to the next
// declaration, so it carries that declaration's doc comment along with it. In
// this file that coarseness was not merely untidy: Modifier.pinMainAxis's own
// doc comment sits between RowChildren and pinMainAxis, it explains that the
// renderer reads `shrinkPinned`, and a `strings.Contains(body, "shrinkPinned")`
// was satisfied by the explanation. Deleting the call and keeping the comment
// passed.
//
// That is the same failure swiftDeclIndices was written for one file over — an
// anchor matching a mention rather than a declaration — and it has the same
// answer: mask the comments and the string literals first, and ask the question
// of what is left. maskSwiftNonCode is named for Swift and is not specific to
// it; Kotlin spells line comments, block comments and both kinds of string
// literal identically, which is the same argument matchingBrace makes for
// serving both languages with one scanner.
func codeOf(t *testing.T, file, anchor string) string {
	t.Helper()
	return maskSwiftNonCode(declSource(t, file, anchor))
}
