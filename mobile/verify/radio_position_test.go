package verify

import (
	"strings"
	"testing"
)

// A radio's "3 of 8" is its place in the group in document order, however the
// group lays its radios out.
//
// Left to itself, Compose numbers a selectableGroup's members by
// layoutNode.placeOrder, which restarts at zero in every layout parent. A
// group whose radios sit in more than one Row (comps.ColorSwatchPicker's
// grid) was heard on the emulator's TalkBack with orange "3 of 8", teal 5,
// yellow 6, pink 7, green 8, purple 1 and red 3: each number one plus the
// count of lower places across both rows (N-077).
//
//	RenderNode(radiogroup)
//	  radioPositions(node) ─► collectionInfo(n rows, 1 column)
//	                          LocalGrMobRadioPositions provides positions
//	    RenderNode(radio, any depth below)
//	      positions.index[node] ─► collectionItemInfo(rowIndex = place)
//
// Stating it is only half the repair: Compose 1.10 overwrites a stated
// collectionItemInfo with its own count whenever the item's parent carries
// SelectableGroup, so the radiogroup role must no longer write
// selectableGroup().
func TestComposeStatesEachRadiosPlaceInItsGroup(t *testing.T) {
	if role := codeOf(t, kotlinStyle, "fun SemanticsPropertyReceiver.grMobRole("); strings.Contains(role, "selectableGroup()") {
		t.Errorf("%s: grMobRole writes selectableGroup() again — Compose then "+
			"overwrites every radio's stated place with a placeOrder count, "+
			"which is per Row (N-077)", kotlinStyle)
	}

	render := codeOf(t, kotlinRenderer, "fun RenderNode(")
	for _, pin := range []struct{ expr, why string }{
		{"remember(node) { derivedStateOf { radioPositions(node) } }.value",
			"the walk re-runs on a selection without re-providing an unchanged answer"},
		{"collectionInfo = CollectionInfo(rowCount = n, columnCount = 1)",
			"the group states its own size, one column: a radio group is a list to TalkBack"},
		{"val place = LocalGrMobRadioPositions.current?.index?.get(node)",
			"a radio looks itself up by identity in the nearest group's walk"},
		{"rowIndex = place, rowSpan = 1, columnIndex = 0, columnSpan = 1",
			"the place is stated, so Compose's placeOrder count is never consulted"},
		{"if (group) add(LocalGrMobRadioPositions provides positions)",
			"provided with the other subtree locals"},
	} {
		if !strings.Contains(render, pin.expr) {
			t.Errorf("%s: RenderNode has lost %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}

	// String literals matter in the walk (the role names), so it is read
	// with them kept. Over the whole file: the walk is a local fun, and a
	// declaration cut stops at the first nested declaration it meets.
	walk := valuesIn(t, kotlinRenderer)
	for _, pin := range []struct{ expr, why string }{
		{`if (s?.display == "none" || s?.accessibilityHidden == true) continue`,
			"a radio that is not in the accessibility tree takes no place"},
		{`"radio" -> index[child] = index.size`,
			"document order: the place is how many radios came before"},
		{`"radiogroup" -> {}`,
			"a nested group's radios are its own set, as aria-posinset is scoped"},
		{`else -> walk(child)`,
			"any layout between the group and its radios is looked through"},
	} {
		if !strings.Contains(walk, pin.expr) {
			t.Errorf("%s: radioPositions has lost %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}
}
