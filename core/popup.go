package core

// PopupKind is what a control opens: the kind of element that appears when it
// is activated, stated so a reader can say so before the press.
//
// It is the vocabulary of aria-haspopup, and it exists for the near miss
// Style.AccessibilityExpanded names and turns down. A trigger that opens a
// modal looks exactly like a disclosure — comps.DatePicker's even flips a
// glyph — and it is not one: aria-expanded says the content is here, in the
// page, and can be shown or hidden, where a popup is a new surface the reader
// is moved into. Before this type the honest thing was to say nothing, and a
// reader pressing comps.Menu's "⋯" was told "button" and then found itself in
// a dialog with no warning.
//
// # Why one value, when ARIA has seven
//
// ARIA's attribute takes false, true, menu, listbox, tree, grid and dialog.
// This carries the one a widget here can honestly say:
//
//	dialog    comps.Menu and comps.DatePicker both present through core.Modal,
//	          which both web targets write as role="dialog". The value names
//	          what the reader actually lands in.
//	menu      also `true`, its synonym. comps.Menu is *called* a menu and is
//	          not one to a reader: its sheet is a dialog of buttons, and
//	          core.Role has no `menu`/`menuitem` pair
//	          (aria/verify/refusals_test.go records why). Saying menu would
//	          announce a menu keyboard — arrows between items, Escape back to
//	          the trigger — that nothing supplies.
//	listbox   a combobox's popup is a listbox implicitly, so the one widget
//	          with a listbox popup (comps.SearchableSelect, through
//	          RoleComboBox) needs no attribute, and no other trigger opens one.
//	tree      no widget has either, and both are refused patterns.
//	grid
//	false     the absence of a claim, which is PopupNone.
//
// A value is added when a widget opens that kind of surface, not before: a
// constant naming a popup nothing presents would be a claim the framework
// cannot keep.
//
// # Why the values are ARIA's own spellings
//
// The same trade core.Role, SelectedState and ExpandedState made: both web
// targets write the value verbatim and need no table.
//
// # What each target does with it
//
// The two web targets write aria-haspopup, scoped to the roles ARIA defines it
// on (see Style.AccessibilityHasPopup). Neither native has anything to put it
// in — Compose's semantics and SwiftUI's traits have no popup property — and
// both present a Modal through a platform dialog that announces itself when it
// opens, which is the half of the warning that matters most there. The key
// crosses the bridge and is deliberately unparsed on both.
type PopupKind string

const (
	// PopupNone is the zero value: this control opens nothing, or says
	// nothing about what it opens. Both web targets write no attribute.
	PopupNone PopupKind = ""

	// PopupDialog — activating this control opens a dialog. See the type doc
	// for why this is the only kind a widget here can say.
	PopupDialog PopupKind = "dialog"
)

// PopupKinds returns every stated value, in declaration order. PopupNone is
// excluded for the reason ExpandedStates() excludes its own zero value: it is
// the absence of a claim rather than one of the kinds.
func PopupKinds() []PopupKind {
	return []PopupKind{PopupDialog}
}
