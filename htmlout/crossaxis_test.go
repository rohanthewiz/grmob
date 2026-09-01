package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The census, in the only form this table can have one. There is no
// core.CrossAxisAlignments() list to hold the keys to — the fallback's source
// vocabulary is a subset of Alignment that exists nowhere else — but the
// table's whole contract is that it translates onto the AlignItems vocabulary
// and invents nothing, so the census runs through the *values*: every
// core.AlignItemsValues() must be produced by exactly one row, and every key
// must be a declared core.Alignment. A deleted row, a duplicated value, or a
// row mapping onto some spelling AlignItems never emits all land here.
func TestCrossAxisAlignsMapOntoExactlyTheAlignItemsVocabulary(t *testing.T) {
	table := CrossAxisAligns()

	declared := map[string]bool{}
	for _, a := range core.Alignments() {
		declared[string(a)] = true
	}
	for key := range table {
		if !declared[key] {
			t.Errorf("table has a row for %q, which core declares no Alignment for", key)
		}
	}

	produced := map[string]string{}
	for key, value := range table {
		if prev, dup := produced[value]; dup {
			t.Errorf("%q is produced by both %q and %q; the fallback must be one-to-one "+
				"with the AlignItems vocabulary", value, prev, key)
		}
		produced[value] = key
	}
	for _, items := range core.AlignItemsValues() {
		if _, ok := produced[string(items)]; !ok {
			t.Errorf("no Alignment falls back to %q; that AlignItems value is reachable "+
				"directly but not through Align", items)
		}
		delete(produced, string(items))
	}
	for value, key := range produced {
		t.Errorf("%q maps to %q, which is not an AlignItems value either DOM renderer emits", key, value)
	}
}

// The two Alignments that are deliberately absent, asserted as absent rather
// than left to the census's silence — the same treatment
// TestTextAlignsOmitTheCrossAxisOnlyAlignments gives the mirror-image pair.
//
// AlignJustify names no cross-axis placement anywhere. AlignBaseline is the
// row someone "completing" the table would add, because CSS align-items
// genuinely has a baseline keyword — but no native dispatch answers for it
// (it falls through to start-packing on both), so the row would baseline-align
// on exactly the two DOM targets. "" is what leaves the container as if Align
// were unset.
func TestCrossAxisAlignsOmitTheTextOnlyAndUnansweredAlignments(t *testing.T) {
	for _, align := range []core.Alignment{core.AlignJustify, core.AlignBaseline} {
		if got := CrossAxisAlignFor(string(align)); got != "" {
			t.Errorf("CrossAxisAlignFor(%q) = %q; no native cross-axis dispatch answers for %s, "+
				"so the DOM must not either", align, got, align)
		}
	}
}

// The gate is exactly the containers the natives read the fallback for. This
// is a pin on a decision, not a census — there is no core list of
// "vertical-stacking container types" to hold it to — so the expected set is
// spelled here once, beside the reasoning: Column and Card route to
// GrMobColumn on both natives, List to GrMobList, and all three read the
// fallback; Row deliberately declines it on every target, and leaf types were
// never containers. Growing the set is legitimate (a new vertical container
// would belong in it); the test makes the growth a visible decision on both
// DOM targets at once, via the wasm conformance test that compares the
// runtime's copy to the same table.
func TestAlignFallbackGateMatchesTheNativeContainers(t *testing.T) {
	want := map[string]string{"Column": "column", "Card": "column", "List": "column"}
	got := AlignFallbackAxes()
	for typ, axis := range want {
		if got[typ] != axis {
			t.Errorf("%s: axis %q, want %q — the natives read the fallback for this type", typ, got[typ], axis)
		}
		delete(got, typ)
	}
	for typ := range got {
		t.Errorf("gate includes %q, which no native reads the fallback for", typ)
	}
}

// End-to-end through the exporter, for each type in the gate: Align alone
// must make the container flex and place its children, exactly as if the
// corresponding AlignItems had been set. This was the visible half of the
// gap — Align: "center" on a Column centered children on device and only the
// text on the web.
func TestAlignFallbackBecomesAlignItemsOnTheVerticalContainers(t *testing.T) {
	for _, typ := range AlignFallbackTypes() {
		n := &core.Node{Type: typ, Style: &core.Style{Align: core.AlignCenter}}
		out := ExportHTML(n)
		for _, want := range []string{"display:flex", "flex-direction:column", "align-items:center"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s with Align center: missing %q:\n%s", typ, want, out)
			}
		}
	}
}

// The stretch value, end to end: the one the natives fill with rather than
// place, and the one whose absence on the web was masked wherever block flow
// or the flex align-items default happened to produce the same picture. The
// fallback makes the agreement declared rather than coincidental.
func TestAlignStretchFallbackFillsTheListRows(t *testing.T) {
	n := &core.Node{Type: "List", Style: &core.Style{Align: core.AlignStretch}}
	out := ExportHTML(n)
	for _, want := range []string{"display:flex", "align-items:stretch"} {
		if !strings.Contains(out, want) {
			t.Errorf("List with Align stretch: missing %q:\n%s", want, out)
		}
	}
	// Stretch is cross-axis-only; it must not leak into the text role.
	if strings.Contains(out, "text-align") {
		t.Errorf("Align stretch emitted a text-align, which has no such keyword:\n%s", out)
	}
}

// AlignItems set means the fallback is never consulted — the same precedence
// crossAxisValue gives it on iOS and `alignItems.ifEmpty { align }` on
// Android.
func TestAlignItemsWinsOverTheAlignFallback(t *testing.T) {
	n := &core.Node{Type: "Column", Style: &core.Style{
		Align:      core.AlignCenter,
		AlignItems: core.AlignItemsEnd,
	}}
	out := ExportHTML(n)
	if !strings.Contains(out, "align-items:flex-end") {
		t.Errorf("explicit AlignItems lost to the fallback:\n%s", out)
	}
	if strings.Contains(out, "align-items:center") {
		t.Errorf("the fallback was emitted alongside the explicit AlignItems:\n%s", out)
	}
}

// The declines, each of which is a line every target draws in the same place:
//
//   - Row: Align has never been read for a vertical cross axis.
//   - Text: not a container; its Align is the ordinary text role, and turning
//     the span into a flex container would be the gate failing at its job.
//   - Column flipped to a row by FlexDirection: the node still becomes flex
//     (FlexDirection alone triggers that), but its cross axis is vertical
//     now, so the fallback stays out of it.
//   - Column with a text-only Align: no cross-axis value, no flex container.
func TestAlignFallbackDeclines(t *testing.T) {
	cases := []struct {
		name string
		node *core.Node
		// The declaration that must NOT appear; "" means no flex at all.
		forbidden string
	}{
		{"Row", &core.Node{Type: "Row", Style: &core.Style{Align: core.AlignCenter}},
			"align-items"},
		{"Text", &core.Node{Type: "Text", Props: map[string]any{"content": "hi"},
			Style: &core.Style{Align: core.AlignCenter}}, "display:flex"},
		{"flipped Column", &core.Node{Type: "Column", Style: &core.Style{
			Align: core.AlignCenter, FlexDirection: "row"}}, "align-items"},
		{"Column with justify", &core.Node{Type: "Column", Style: &core.Style{
			Align: core.AlignJustify}}, "display:flex"},
	}
	for _, c := range cases {
		if out := ExportHTML(c.node); strings.Contains(out, c.forbidden) {
			t.Errorf("%s: emitted %q, which this case must decline:\n%s", c.name, c.forbidden, out)
		}
	}
}

// A copy, for the reason TextAligns hands out one: the conformance and census
// tests delete from what they are given as they match rows.
func TestCrossAxisTablesReturnCopies(t *testing.T) {
	aligns := CrossAxisAligns()
	delete(aligns, "center")
	aligns["start"] = "center"
	if got := CrossAxisAlignFor("center"); got != "center" {
		t.Errorf("deleting from the returned map changed CrossAxisAlignFor(center): %q", got)
	}
	if got := CrossAxisAlignFor("start"); got != "flex-start" {
		t.Errorf("writing to the returned map changed CrossAxisAlignFor(start): %q", got)
	}

	axes := AlignFallbackAxes()
	delete(axes, "List")
	if got := AlignFallbackAxisFor("List"); got != "column" {
		t.Errorf("deleting from the returned map changed AlignFallbackAxisFor(List): %q", got)
	}
}
