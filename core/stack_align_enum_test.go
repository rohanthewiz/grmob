package core

import "testing"

// StackAlignments(), pinned to the const block in stack_align.go that it
// restates. See enum_pin_test.go for the parse and why it is one, and
// stack_align.go for what the list obliges each renderer to do.
//
// This census hangs more weight than most: three renderers are held to it —
// htmlout's placement table by number and both natives' dispatches by
// mobile/verify's coverage check — so a value that quietly fell off this list
// would quietly stop requiring an arm in three places at once, and a layer
// asking for it would land in each renderer's default and render centred.

// Checked against the declarations *plus the zero value*, because
// StackAlignCenter is a declared StackAlignment that StackAlignments()
// deliberately omits: it is Style.StackAlign's zero value and no renderer has
// an arm for "the default". The same shape SelectedStates() has.
func TestStackAlignmentsMatchTheDeclaredConstants(t *testing.T) {
	requireExactEnum(t, "stack_align.go", "StackAlignment",
		"StackAlignments() plus StackAlignCenter",
		append(StackAlignments(), StackAlignCenter))
}

// StackAlignCenter must stay out of the census and must stay the empty string.
//
// It is Style.StackAlign's zero value, so every node in every tree carries it;
// a spelling would make each of them a placement claim, and would oblige every
// renderer to answer for a value that means "do what you already do".
func TestStackAlignCenterIsTheUnspelledZeroValue(t *testing.T) {
	if StackAlignCenter != "" {
		t.Errorf("StackAlignCenter = %q, want the empty string so an unset "+
			"Style.StackAlign is it", StackAlignCenter)
	}
	for _, a := range StackAlignments() {
		if a == StackAlignCenter {
			t.Error("StackAlignments() lists StackAlignCenter; the census is the " +
				"placements that ask for something")
		}
	}
}

// The eight stated values must be the eight cells of a 3x3 grid with the
// centre removed — one per direction, none repeated.
//
// Written as an arithmetic check rather than as a list, because a list here
// would be a third copy of the const block (the pin above is the second) and
// would agree with a typo as readily as with a value. What this catches is the
// failure a spelling test cannot: a ninth value, a duplicate, or a name whose
// two halves do not decompose — "topstart", say, or "middle-start" — any of
// which reaches the renderers as a string none of their tables answer for.
func TestStackAlignmentsAreTheEightCellsOfTheGrid(t *testing.T) {
	// Every stated value is exactly "<vertical>", "<horizontal>", or
	// "<vertical>-<horizontal>", where each half is one of the two named
	// edges. The centre is what is left when both halves are omitted, which
	// is why it is the empty string.
	vertical := map[string]bool{"top": true, "bottom": true}
	horizontal := map[string]bool{"start": true, "end": true}

	seen := map[[2]string]bool{}
	for _, a := range StackAlignments() {
		var v, h string
		s := string(a)
		switch {
		case vertical[s]:
			v = s
		case horizontal[s]:
			h = s
		default:
			i := -1
			for j := 0; j < len(s); j++ {
				if s[j] == '-' {
					i = j
					break
				}
			}
			if i < 0 || !vertical[s[:i]] || !horizontal[s[i+1:]] {
				t.Errorf("StackAlignment %q is not a vertical edge, a horizontal edge, "+
					"or a hyphenated pair of the two", a)
				continue
			}
			v, h = s[:i], s[i+1:]
		}
		cell := [2]string{v, h}
		if cell == [2]string{} {
			t.Errorf("StackAlignment %q decodes to the centre cell, which is "+
				"StackAlignCenter's and must have no second spelling", a)
		}
		if seen[cell] {
			t.Errorf("StackAlignment %q names a cell already claimed", a)
		}
		seen[cell] = true
	}

	if len(seen) != 8 {
		t.Errorf("the census fills %d of the 3x3 grid's 8 non-centre cells; every "+
			"renderer's placement table is sized by this list", len(seen))
	}
}
