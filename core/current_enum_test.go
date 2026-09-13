package core

import "testing"

// CurrentKinds(), pinned to the const block in current.go that it restates,
// plus the zero value it deliberately omits. See expanded_enum_test.go for the
// same arrangement and enum_pin_test.go for the parse.
func TestCurrentKindsMatchTheDeclaredConstants(t *testing.T) {
	requireExactEnum(t, "current.go", "CurrentKind", "CurrentKinds() plus CurrentNone",
		append(CurrentKinds(), CurrentNone))
}

// The values are ARIA's own spellings, written into aria-current verbatim by
// both web targets. CurrentNone must be the empty string: it is
// Style.AccessibilityCurrent's zero value, carried by every node, and a
// spelling would make every node claim to be current.
func TestCurrentKindsAreSpelledAsARIAWritesThem(t *testing.T) {
	if CurrentNone != "" {
		t.Errorf("CurrentNone = %q, want the empty string", CurrentNone)
	}
	for kind, want := range map[CurrentKind]string{
		CurrentPage: "page", CurrentStep: "step", CurrentTrue: "true",
	} {
		if string(kind) != want {
			t.Errorf("CurrentKind %q, want %q — the value is written into aria-current verbatim", kind, want)
		}
	}
	for _, kind := range CurrentKinds() {
		if kind == CurrentNone {
			t.Error("CurrentKinds() lists CurrentNone; the census is the kinds that say something")
		}
	}
}
