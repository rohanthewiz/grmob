package core

// SelectedState is whether a control is *on* — the tab that is showing, the
// filter chip that is applied, the calendar day that is chosen.
//
// It is the state half of core.Role. A role says what a control is and the
// label says what it is called; neither can say that this one of five chips
// is the one in effect, and until this type existed nothing in the framework
// could. What a widget did instead was write the state into the name —
// components.Chip appended ", selected" to its AccessibilityLabel — which
// announces once, in the wrong place (a name is meant to be stable, and a
// reader that re-announces the control after a tap says the whole altered
// name rather than the changed state), and which no platform can act on.
//
// # Why three values and not a bool
//
// Because "off" and "not a thing that can be on" are different facts and a
// bool has one spelling for both.
//
//	SelectedUnset   a Box, a heading, a row of text. The node makes no claim,
//	                which is the state every node in every tree was in before
//	                this field existed, so the zero value is a no-op.
//	SelectedOn      the chip that is applied, the tab that is showing.
//	SelectedOff     a control that *could* be on and is not.
//
// The third is the one a bool loses, and losing it is not cosmetic. A tablist
// in which only the live tab carries a state is malformed: a reader counting
// "tab 2 of 5" needs all five to say something, and the four that stay quiet
// are announced as plain tabs while the fifth is announced as selected — a
// strip that reads as though it has one tab and four pieces of furniture. So
// a strip sets the state on every control it draws, and SelectedOff is a
// value with a job rather than a way of saying nothing.
//
// # Why the values are ARIA's own spellings
//
// For the reason core.Role's are: the two DOM targets write the value into
// the attribute verbatim and need no mapping table, and the two natives map
// what they can — which is the same trade the vocabulary made and the same
// place it pays off. `mixed`, ARIA's third value for aria-pressed and
// aria-checked, is deliberately absent: it describes a control governing a
// partially-selected set, no widget here has one, and a value nobody can
// produce is a fourth arm on every renderer for nothing.
type SelectedState string

const (
	// SelectedUnset is the zero value: this node says nothing about being on
	// or off. Every renderer writes no attribute, adds no trait and sets no
	// semantics property, which is what every node did before this type.
	SelectedUnset SelectedState = ""

	// SelectedOn — the control is on.
	SelectedOn SelectedState = "true"

	// SelectedOff — the control can be on and is not. See the type doc for
	// why this is not the same as SelectedUnset.
	SelectedOff SelectedState = "false"
)

// SelectedWhen turns a widget's plain bool into the stated pair.
//
// Every widget that has one of these holds a `Selected bool` — the caller
// owns the selection and the widget renders it — so the conversion would
// otherwise be three lines at each of them, and the tempting two-line version
// (`if on { … SelectedOn }` and nothing else) is exactly the bug the type doc
// warns about: it leaves the unselected controls silent.
//
// It is a function rather than a method so that the bool reads as the
// subject: core.AccessibilitySelected(core.SelectedWhen(c.Selected)).
func SelectedWhen(on bool) SelectedState {
	if on {
		return SelectedOn
	}
	return SelectedOff
}

// SelectedStates returns both stated values, in declaration order.
//
// SelectedUnset is excluded for the reason RoleNone is excluded from Roles():
// it is the field's zero value rather than one of the states, no renderer has
// an arm for it, and a coverage check that demanded one would be asking each
// renderer to implement "unstated".
//
// Pinned to the const block above by selected_enum_test.go, and consumed by
// the exporters' round-trip checks the way Roles() is.
func SelectedStates() []SelectedState {
	return []SelectedState{SelectedOn, SelectedOff}
}
