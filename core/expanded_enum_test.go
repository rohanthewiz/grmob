package core

import "testing"

// ExpandedStates(), pinned to the const block in expanded.go that it restates.
// See enum_pin_test.go for the parse and why it is one, and expanded.go for
// what the list obliges each renderer to do.
//
// The same arrangement Roles() and SelectedStates() have, and it hangs the
// same weight: mobile/verify holds Compose's expand/collapse dispatch against
// this list, so a census that could quietly stop listing a state would quietly
// stop requiring an arm for it.

// Checked against the declarations *plus the zero value*, because
// ExpandedUnset is a declared ExpandedState that ExpandedStates() deliberately
// omits — it is the absence of a claim, and no renderer has an arm for
// "unstated". Adding it back here is what keeps the check exact in both
// directions.
func TestExpandedStatesMatchTheDeclaredConstants(t *testing.T) {
	requireExactEnum(t, "expanded.go", "ExpandedState", "ExpandedStates() plus ExpandedUnset",
		append(ExpandedStates(), ExpandedUnset))
}

// ExpandedUnset must stay out of the census and must stay the empty string: it
// is Style.AccessibilityExpanded's zero value, so every node in every tree
// carries it, and a spelling would make every one of them claim to be a
// disclosure.
func TestExpandedStatesOmitTheZeroValue(t *testing.T) {
	if ExpandedUnset != "" {
		t.Errorf("ExpandedUnset = %q, want the empty string so an unset "+
			"Style.AccessibilityExpanded is it", ExpandedUnset)
	}
	for _, state := range ExpandedStates() {
		if state == ExpandedUnset {
			t.Error("ExpandedStates() lists ExpandedUnset; the census is the states that say something")
		}
	}
}

// The values are ARIA's own, which is why neither web exporter needs a mapping
// table — the state crosses the bridge and becomes the attribute value
// verbatim. A constant respelled here (to "open"/"closed", say) would still
// compile, still merge, still reach all four renderers, and write
// aria-expanded="open", which no screen reader honours.
//
// The Kotlin renderer compares against these literals too, so the strings are
// a three-way contract rather than a detail of the DOM half.
func TestExpandedStatesAreSpelledAsARIAWritesThem(t *testing.T) {
	if ExpandedOpen != "true" {
		t.Errorf("ExpandedOpen = %q, want %q — the value is written into "+
			"aria-expanded verbatim", ExpandedOpen, "true")
	}
	if ExpandedClosed != "false" {
		t.Errorf("ExpandedClosed = %q, want %q", ExpandedClosed, "false")
	}
}

// ExpandedWhen exists so the two-state conversion is written once, and the
// half that matters is the false one. The hand-rolled version sets ExpandedOpen
// and leaves the shut case at the zero value, which announces a collapsed
// section as a plain button with nothing behind it.
func TestExpandedWhenStatesBothHalves(t *testing.T) {
	if got := ExpandedWhen(true); got != ExpandedOpen {
		t.Errorf("ExpandedWhen(true) = %q, want %q", got, ExpandedOpen)
	}
	if got := ExpandedWhen(false); got != ExpandedClosed {
		t.Errorf("ExpandedWhen(false) = %q, want %q — a shut disclosure has to say so, "+
			"not go quiet", got, ExpandedClosed)
	}
}

// The two state types carry identical values and are deliberately not the same
// type, so a caller cannot transpose them. This is the assertion that keeps
// somebody from "simplifying" one into the other: the values below are equal
// as strings and the compiler must still refuse the assignment, which it does
// only while the named types are distinct.
//
// Written as a value comparison rather than a compile-fail fixture because Go
// has no in-tree spelling for the latter. What it pins is the premise — that
// the two are interchangeable by value and must not be by type — so a future
// `type ExpandedState = SelectedState` alias fails here by name.
func TestTheTwoStateTypesShareValuesAndNotAType(t *testing.T) {
	if string(ExpandedOpen) != string(SelectedOn) || string(ExpandedClosed) != string(SelectedOff) {
		t.Fatal("the two state types no longer share ARIA's spellings; this test's premise is gone")
	}
	if any(ExpandedOpen) == any(SelectedOn) {
		t.Error("ExpandedOpen and SelectedOn compare equal as interface values, which means " +
			"they are one type — a selection and a disclosure are independent facts about a " +
			"node and are scoped to different roles; see core.ExpandedState")
	}
}
