package verify

import (
	"strings"
	"testing"
)

// A Compose Row measures an unweighted child against everything its earlier
// siblings left, as a maximum, and a stretched Column fills that maximum: the
// first demoBox of lesson 1.3 took the whole Row on the emulator and B and C
// were measured against nothing. CSS sizes the same flex item by its content.
// RowChildren hands such a child rowChildWidth, which measures it within
// min(offer, maxIntrinsicWidth), with a percentage MinWidth resolved against
// the Row rather than the content.
//
// Held in source because the measure itself runs only on a device; the
// before/after screenshots live in the session doc that introduced it.
func TestComposeUnweightedRowChildHugsItsContent(t *testing.T) {
	for _, c := range []struct{ decl, expr, why string }{
		{"private fun RowScope.RowChildren(",
			"if (grow <= 0f && !unboundedWidth && child.style?.shrinkPinned != true && sizedByRow(child))",
			"an unweighted, unpinned child in a bounded Row is the one CSS sizes by its content"},
		{"private fun Modifier.rowChildWidth(", "val content = measurable.maxIntrinsicWidth(constraints.maxHeight)",
			"the content width is asked without measuring, so the child is measured once"},
		{"private fun Modifier.rowChildWidth(", "maxW = minOf(offer, content)",
			"content width, shrunk to the offer"},
		{"private fun Modifier.rowChildWidth(",
			"if (it.isFraction) (offer * it.amount).roundToInt() else it.amount.dp.roundToPx()",
			"a percentage floor resolves against the Row's offer; widthModifier further in only sees the content"},
		{"private fun sizedByRow(", "return answersIntrinsicWidth(child)",
			"a List or vertical Scroll inside would throw on the intrinsic query"},
		{"private fun sizedByRow(", "if (s != null && s.width.isNotEmpty()) return false",
			"a Width is already the item's size"},
	} {
		if !strings.Contains(codeOf(t, kotlinRenderer, c.decl), c.expr) {
			t.Errorf("%s: %s has no %q — %s", kotlinRenderer, c.decl, c.expr, c.why)
		}
	}

	// The filler set, with its string literals intact. A horizontal Scroll
	// must stay out of the SET: a strip with a grower captures its viewport
	// width during measure, and an intrinsic query would capture an unbounded
	// one. isPlainStrip is the narrower door it comes in by; see
	// TestAPlainScrollingStripInARowHugsItsContent.
	fillers := valuesOf(t, kotlinRenderer, "private val RowOfferFillers =")
	for _, want := range []string{`"Column"`, `"Card"`, `"Box"`, `"Row"`} {
		if !strings.Contains(fillers, want) {
			t.Errorf("%s: RowOfferFillers has no %s, so that container would still fill a Row", kotlinRenderer, want)
		}
	}
	if strings.Contains(fillers, `"Scroll"`) {
		t.Errorf("%s: RowOfferFillers holds \"Scroll\"; a grow strip's viewport capture would run unbounded", kotlinRenderer)
	}
}

// CSS's `min-width: auto`: a flex item is never measured below its min-content
// width, and the line overflows instead.
//
// # What it is for
//
// Compose's Row hands a child whatever its predecessors left and treats that
// as binding, so a Text there breaks INSIDE a word — lesson 1.1's third stat
// read "Fo llo wi ng" on the emulator, and 8.2's "Repair (set B = A)" button
// the same way. No browser does that: min-content is the width of the longest
// unbreakable run, and a browser overflows at the word rather than through it.
//
// # The two halves, and why each fails silently alone
//
//	minIntrinsicWidth   asked of EVERY unweighted child, not only the
//	                    containers the cap applies to. 8.2's button is a leaf;
//	                    a floor that skipped leaves would leave half the
//	                    reported defect exactly as it was, and the lesson that
//	                    still broke would look like a second unrelated bug.
//
//	report unclamped    a `coerceIn(constraints…)` on the way out compiles and
//	                    looks better behaved, and it tells the Row a width that
//	                    fits. The Row's running total then never exceeds its
//	                    own maximum: nothing overflows, and the next sibling is
//	                    laid out on top of a child that is drawn wider than the
//	                    parent believes. That is the mistake
//	                    TestTheComposePinMeasuresUnboundedAndReportsWhatItMeasured
//	                    refuses for pinMainAxis, for the same reason.
func TestComposeFloorsARowChildAtItsMinContentWidth(t *testing.T) {
	body := codeOf(t, kotlinRenderer, "private fun Modifier.rowChildWidth(")
	for _, c := range []struct{ expr, why string }{
		{"val minContent = measurable.minIntrinsicWidth(constraints.maxHeight)",
			"min-content is asked of the subtree, not derived from a measurement"},
		{"val minW = maxOf(constraints.minWidth, declared, minContent)",
			"the floor is the widest of the incoming minimum, a declared MinWidth and min-content"},
		{"val reported = maxOf(placeable.width, constraints.minWidth)",
			"the parent learns the child's real extent, so an overflowing Row overflows"},
	} {
		if !strings.Contains(body, c.expr) {
			t.Errorf("%s: rowChildWidth has no %q — %s.\n\nWithout it a Row whose "+
				"children together exceed it breaks a word in half where every "+
				"browser overflows at the word.", kotlinRenderer, c.expr, c.why)
		}
	}
	if strings.Contains(body, "placeable.width.coerceIn(") {
		t.Errorf("%s: rowChildWidth coerces the width it reports back into the "+
			"incoming constraints. The Row then believes every child fits, so a "+
			"child held at its min-content floor is drawn over the sibling after "+
			"it instead of pushing the row past its own maximum. See pinMainAxis, "+
			"which refuses the same spelling.", kotlinRenderer)
	}
	// The gate, which is what makes the floor reach a Button at all.
	gate := codeOf(t, kotlinRenderer, "private fun sizedByRow(")
	if strings.Contains(gate, "RowOfferFillers") {
		t.Errorf("%s: sizedByRow consults RowOfferFillers, so only containers are "+
			"floored. The floor is a rule about flex items and 8.2's button is a "+
			"leaf; RowOfferFillers decides the CAP (hugsRowOffer), not the floor.",
			kotlinRenderer)
	}
}

// A percentage MaxWidth is a share of the ROW, and widthModifier resolves it
// against whatever maximum rowChildWidth passes down.
//
// This is why such a child used to be skipped outright: capping here as well
// would apply the share twice, and handing down min(offer, content) makes
// widthModifier compute a share of the CONTENT. The fix is neither — the
// maximum handed down is `content ÷ N%`, chosen so that widthModifier's own
// arithmetic lands on `min(content, N% × offer)`.
//
//	M = offer          → N% × offer          when content ≥ N% × offer
//	M = content ÷ N%   → N% × content ÷ N%   when content < N% × offer
//	M = min(the two)   → both, in one line
func TestAPercentageMaxWidthInARowIsAShareOfTheRow(t *testing.T) {
	body := codeOf(t, kotlinRenderer, "private fun Modifier.rowChildWidth(")
	if !strings.Contains(body, "maxW = minOf(offer, ceil(content / cap.amount).toInt())") {
		t.Errorf("%s: rowChildWidth does not divide the content width by a "+
			"percentage MaxWidth.\n\nWithout it the child is either skipped (and "+
			"fills the Row) or handed a content-width maximum that widthModifier "+
			"takes its percentage of, which is a share of the content rather than "+
			"of the Row.", kotlinRenderer)
	}
	if !strings.Contains(body, "cap.amount > 0f") {
		t.Errorf("%s: rowChildWidth divides by a percentage MaxWidth without "+
			"checking it is positive. `MaxWidth(\"0%%\")` is a division by zero.",
			kotlinRenderer)
	}
	// And the gate no longer turns such a child away.
	if strings.Contains(codeOf(t, kotlinRenderer, "private fun sizedByRow("), "maxWidth.endsWith(") {
		t.Errorf("%s: sizedByRow still refuses a percentage MaxWidth, so the "+
			"division above is unreachable and the child fills the Row as before.",
			kotlinRenderer)
	}
}

// A horizontal Scroll in a Row hugs its content when — and only when — it
// holds no grower.
//
// With a grower it renders through GrMobGrowStrip, whose measure policy reads
// a viewport width that a layout modifier records during the same pass; an
// intrinsic query runs that capture with an unbounded width and the strip
// sizes itself to infinity. Without one it is a plain Row under
// horizontalScrollWhenBounded, which forwards intrinsics to its own content.
func TestAPlainScrollingStripInARowHugsItsContent(t *testing.T) {
	body := valuesOf(t, kotlinRenderer, "private fun isPlainStrip(")
	for _, c := range []struct{ expr, why string }{
		{`child.type == "Scroll"`, "only a Scroll takes this door"},
		{`child.style?.flexDirection == "row"`, "a vertical Scroll cannot answer intrinsics at all"},
		{"children.none { (it.style?.flexGrow ?: 0f) > 0f }",
			"a grower means GrMobGrowStrip, whose viewport capture an intrinsic query would run unbounded"},
	} {
		if !strings.Contains(body, c.expr) {
			t.Errorf("%s: isPlainStrip has no %q — %s", kotlinRenderer, c.expr, c.why)
		}
	}
	if !strings.Contains(codeOf(t, kotlinRenderer, "private fun hugsRowOffer("), "isPlainStrip(child)") {
		t.Errorf("%s: hugsRowOffer does not admit a plain strip, so a horizontal "+
			"Scroll in a Row still fills the offer.", kotlinRenderer)
	}
}
