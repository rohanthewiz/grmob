package core

// ExpandedState is whether a disclosure is *open* — the accordion section
// showing its body, the twisty that has been turned.
//
// It is the third of the state types, after SelectedState, and it exists
// because a control can be on and open at the same time. components.Accordion
// is the widget that asked for it: it is the one stateful widget in the
// components package, its header is the only thing on screen that knows
// whether the section is showing, and until this type existed the only thing
// that said so was a chevron glyph — which a screen reader announces as
// "black right-pointing small triangle" or, more often, not at all.
//
// # Why this is a second type and not SelectedState reused
//
// The two carry the same three values for the same reason (see below), and
// reusing the type would have compiled, exported and rendered. It is turned
// down on two grounds.
//
// *They are independent facts about one node.* A control can be selected and
// expanded at once — a menu button that is both the current tab and showing
// its submenu — so they are two fields, and two fields typed the same are two
// fields a caller can transpose. core.AccessibilitySelected(ExpandedOpen) is
// nonsense that would type-check.
//
// *They are scoped differently.* ARIA defines aria-selected for four of this
// framework's roles and aria-pressed for a fifth; aria-expanded is defined for
// a sixth set that overlaps but does not match. A shared type would suggest a
// shared guard, and the guards are what make each state legal where it is
// written.
//
// # Why three values and not a bool
//
// The same argument SelectedState makes, arriving from the opposite end. There
// "off" and "not selectable" were two facts a bool has one spelling for; here
// it is "closed" and "not a disclosure".
//
//	ExpandedUnset    a Box, a heading, a row of text. Makes no claim, which is
//	                 what every node in every tree was before this field, so
//	                 the zero value is a no-op.
//	ExpandedOpen     the section that is showing.
//	ExpandedClosed   a disclosure that could be open and is not.
//
// The third is again the one that would be lost, and losing it is worse here
// than it is for a selection. A closed accordion that says nothing is
// announced as an ordinary button: a reader is told they can press it and not
// that there is anything behind it. "Collapsed" is the whole of what invites
// the press.
//
// # Why the values are ARIA's own spellings
//
// Because both web targets write them into the attribute verbatim and need no
// mapping table, which is the trade core.Role and SelectedState both made. The
// two natives map what they can, which here is one of them — see
// Style.AccessibilityExpanded.
type ExpandedState string

const (
	// ExpandedUnset is the zero value: this node is not a disclosure and says
	// nothing about being open. Every renderer writes no attribute and offers
	// no action.
	ExpandedUnset ExpandedState = ""

	// ExpandedOpen — the disclosure is showing its content.
	ExpandedOpen ExpandedState = "true"

	// ExpandedClosed — the disclosure is a disclosure and is shut. See the
	// type doc for why this is not the same as ExpandedUnset.
	ExpandedClosed ExpandedState = "false"
)

// ExpandedWhen turns the bool a disclosure already holds into the stated pair.
//
// The twin of SelectedWhen, and it earns its place the same way: the widget
// owns a `expanded bool` (components.Accordion holds one in NewState), so the
// conversion would otherwise be written by hand at each call site, and the
// tempting hand-rolled version — set ExpandedOpen when open, leave it alone
// otherwise — is exactly the silence the third value exists to prevent.
func ExpandedWhen(open bool) ExpandedState {
	if open {
		return ExpandedOpen
	}
	return ExpandedClosed
}

// ExpandedStates returns both stated values, in declaration order.
//
// ExpandedUnset is excluded for the reason SelectedStates() excludes its own
// zero value: it is the absence of a claim rather than one of the states, and
// a coverage check that demanded a renderer arm for it would be asking each
// renderer to implement "unstated".
//
// Pinned to the const block above by expanded_enum_test.go, and consumed by
// the exporters' round-trip checks the way SelectedStates() is.
func ExpandedStates() []ExpandedState {
	return []ExpandedState{ExpandedOpen, ExpandedClosed}
}
