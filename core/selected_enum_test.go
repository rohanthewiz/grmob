package core

import "testing"

// SelectedStates(), pinned to the const block in selected.go that it restates.
// See enum_pin_test.go for the parse and why it is one, and selected.go for
// what the list obliges each renderer to do.
//
// The same arrangement Roles() has, and it hangs the same weight: the native
// coverage check in mobile/verify holds Compose's dispatch against this list,
// so a census that could quietly stop listing a state would quietly stop
// requiring an arm for it.

// Checked against the declarations *plus the zero value*, because
// SelectedUnset is a declared SelectedState that SelectedStates() deliberately
// omits — it is the absence of a claim, and no renderer has an arm for
// "unstated". Adding it back here is what keeps the check exact in both
// directions.
func TestSelectedStatesMatchTheDeclaredConstants(t *testing.T) {
	requireExactEnum(t, "selected.go", "SelectedState", "SelectedStates() plus SelectedUnset",
		append(SelectedStates(), SelectedUnset))
}

// SelectedUnset must stay out of the census and must stay the empty string: it
// is Style.AccessibilitySelected's zero value, so every node in every tree
// carries it, and a spelling would make every one of them a claim.
func TestSelectedStatesOmitTheZeroValue(t *testing.T) {
	if SelectedUnset != "" {
		t.Errorf("SelectedUnset = %q, want the empty string so an unset "+
			"Style.AccessibilitySelected is it", SelectedUnset)
	}
	for _, state := range SelectedStates() {
		if state == SelectedUnset {
			t.Error("SelectedStates() lists SelectedUnset; the census is the states that say something")
		}
	}
}

// The values are ARIA's own, which is the whole reason neither web exporter
// needs a mapping table — the state crosses the bridge and becomes the
// attribute value verbatim. A constant respelled here (to "on"/"off", say)
// would still compile, still merge, still reach all four renderers, and write
// aria-selected="on", which no screen reader honours.
//
// Both natives compare against these literals too, so the strings are a
// four-way contract rather than a detail of the DOM half.
func TestSelectedStatesAreSpelledAsARIAWritesThem(t *testing.T) {
	if SelectedOn != "true" {
		t.Errorf("SelectedOn = %q, want %q — the value is written into "+
			"aria-selected/aria-pressed verbatim", SelectedOn, "true")
	}
	if SelectedOff != "false" {
		t.Errorf("SelectedOff = %q, want %q", SelectedOff, "false")
	}
}

// SelectedWhen exists so that the two-state conversion is written once. The
// half that matters is the false one: the tempting hand-rolled version sets
// SelectedOn and leaves everything else at the zero value, which is exactly
// the silence a tablist cannot afford.
func TestSelectedWhenStatesBothHalves(t *testing.T) {
	if got := SelectedWhen(true); got != SelectedOn {
		t.Errorf("SelectedWhen(true) = %q, want %q", got, SelectedOn)
	}
	if got := SelectedWhen(false); got != SelectedOff {
		t.Errorf("SelectedWhen(false) = %q, want %q — an unselected control has to "+
			"say so, not go quiet", got, SelectedOff)
	}
}
