package verify

import (
	"strings"
	"testing"
)

// A Compose Row measures an unweighted child against everything its earlier
// siblings left, as a maximum, and a stretched Column fills that maximum: the
// first demoBox of lesson 1.3 took the whole Row on the emulator and B and C
// were measured against nothing. CSS sizes the same flex item by its content.
// RowChildren hands such a child hugRowOffer, which measures it within
// min(offer, maxIntrinsicWidth), with a percentage MinWidth resolved against
// the Row rather than the content.
//
// Held in source because the measure itself runs only on a device; the
// before/after screenshots live in the session doc that introduced it.
func TestComposeUnweightedRowChildHugsItsContent(t *testing.T) {
	for _, c := range []struct{ decl, expr, why string }{
		{"private fun RowScope.RowChildren(",
			"if (grow <= 0f && !unboundedWidth && child.style?.shrinkPinned != true && hugsRowOffer(child))",
			"an unweighted, unpinned child in a bounded Row is the one CSS sizes by its content"},
		{"private fun Modifier.hugRowOffer(", "val content = measurable.maxIntrinsicWidth(constraints.maxHeight)",
			"the content width is asked without measuring, so the child is measured once"},
		{"private fun Modifier.hugRowOffer(", "val maxW = maxOf(minOf(constraints.maxWidth, content), minW)",
			"content width, shrunk to the offer, never below the floor"},
		{"private fun Modifier.hugRowOffer(",
			"if (it.isFraction) (constraints.maxWidth * it.amount).roundToInt() else it.amount.dp.roundToPx()",
			"a percentage floor resolves against the Row's offer; widthModifier further in only sees the content"},
		{"private fun hugsRowOffer(", "return child.type in RowOfferFillers && answersIntrinsicWidth(child)",
			"a List or vertical Scroll inside would throw on the intrinsic query"},
		{"private fun hugsRowOffer(", "s.width.isNotEmpty() || s.maxWidth.endsWith(",
			"a Width is already the item's size, and a percentage cap would resolve against the content"},
	} {
		if !strings.Contains(codeOf(t, kotlinRenderer, c.decl), c.expr) {
			t.Errorf("%s: %s has no %q — %s", kotlinRenderer, c.decl, c.expr, c.why)
		}
	}

	// The filler set, with its string literals intact. A horizontal Scroll
	// must stay out: a grow strip captures its viewport width during measure,
	// and an intrinsic query would capture an unbounded one.
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
