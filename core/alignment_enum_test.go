package core

import "testing"

// The four alignment enumerations, pinned to the const blocks in style.go that
// they restate. See enum_pin_test.go for the parse and why it is one, and
// alignment.go for what each list obliges a renderer to do.
//
// The lists are in alignment.go and the constants are in style.go, so unlike
// ContentModes these pins span two files. That is the arrangement's one real
// hazard — the list and the declaration are not on the same screen, so a
// constant added to style.go is *more* likely to be forgotten here, not less —
// which is the argument for the pin rather than against the arrangement.

// Alignment carries two roles (text alignment and cross-axis placement) and
// this is the census of both: the list no renderer is held to directly, and
// the one every Alignment subset has to be a subset of.
func TestAlignmentsMatchTheDeclaredConstants(t *testing.T) {
	requireExactEnum(t, "style.go", "Alignment", "Alignments()", Alignments())
}

// TestTextAlignmentsAreDeclaredAlignments only checks the direction a subset
// can be checked in: every value it hands out must be a real Alignment.
//
// It cannot check that the omissions are the right omissions — that AlignStretch
// and AlignBaseline belong outside and AlignJustify belongs inside is a
// judgment about what text alignment means, argued in TextAlignments' doc
// comment. What it does catch is the mechanical failure: a renamed constant,
// or a value invented here, would make every text dispatch downstream answer
// for a value core cannot produce.
func TestTextAlignmentsAreDeclaredAlignments(t *testing.T) {
	requireSubsetEnum(t, "style.go", "Alignment", "TextAlignments()", TextAlignments())
}

// The two subsets must together stay inside the census, and the census must
// stay inside the declarations — checked above. This adds the middle link:
// nothing may be a TextAlignment that is not an Alignment *as Alignments()
// reports them*, which is what stops the census and the subset from being
// pinned to the same const block but to different readings of it.
func TestTextAlignmentsAreASubsetOfAlignments(t *testing.T) {
	all := map[Alignment]bool{}
	for _, a := range Alignments() {
		all[a] = true
	}
	for _, a := range TextAlignments() {
		if !all[a] {
			t.Errorf("TextAlignments() yields %q, which Alignments() does not list", a)
		}
	}
}

func TestJustifyContentsMatchTheDeclaredConstants(t *testing.T) {
	requireExactEnum(t, "style.go", "JustifyContent", "JustifyContents()", JustifyContents())
}

// AlignItems shares its const block with FlexDirection and with the untyped
// DisplayFlex, so this is also the case that exercises the parse's rule about
// reading the type off each spec rather than off the block.
func TestAlignItemsValuesMatchTheDeclaredConstants(t *testing.T) {
	requireExactEnum(t, "style.go", "AlignItems", "AlignItemsValues()", AlignItemsValues())
}
