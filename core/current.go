package core

// CurrentKind says this item is the *current* one of a set, and what kind of
// set: the page a navigation bar is showing, the step a wizard is on, the value
// a picker holds.
//
// It is the vocabulary of aria-current, and it closes the gap four widgets used
// to fill with English. comps.BottomBar and comps.Drawer appended ", selected"
// to their current destination's name, comps.StepIndicator appended
// ", current", and comps.ActionSheet's Checked action appended ", selected". A
// name is meant to be stable, a suffix cannot be translated by the platform,
// and each of the four was the weaker place to say a true thing because core
// had no stronger one.
//
// # Why not AccessibilitySelected
//
// SelectedState is *one of these, and choosing one unchooses the rest*, and it
// is scoped by role: aria-selected on option, tab, row and columnheader,
// aria-pressed on a button. None of the four fits. A bottom bar's cells are
// buttons, and aria-pressed announces a toggle a second tap does not turn off.
// RoleTab would claim a tab panel the bar does not control. A drawer row is a
// named group, which takes no selection attribute at all.
//
// aria-current is the attribute ARIA made for exactly this: "the element that
// represents the current item within a container or set of related elements".
// It is a global, so it needs no role beside it and no guard in either web
// exporter.
//
// # Why these values, when ARIA has seven
//
// ARIA's attribute takes page, step, location, date, time, true and false.
// This carries the ones a widget here says, on the rule PopupKind follows: a
// value is added when a widget uses it, not before.
//
//	page      comps.BottomBar's and comps.Drawer's current destination.
//	step      comps.StepIndicator's current step.
//	true      comps.ActionSheet's Checked action: the current choice in a
//	          "Sort by" sheet, which is a current item and not a page or step.
//	date      left out. comps.Calendar's today cell is ARIA's own example,
//	          and it keeps its ", today" suffix because both natives map a
//	          current item to their selected state (below), which would
//	          announce today as the selected day. That is a lie on the two
//	          platforms the suffix is still true on.
//	location  no widget has either.
//	time
//	false     the absence of a claim, which is CurrentNone.
//
// # What each target does with it
//
// Both web targets write aria-current verbatim, on any role, unless the node is
// hidden. Neither native has a current property. Compose's `selected` semantics
// and SwiftUI's `.isSelected` trait are the nearest true thing, and they are
// what Material's navigation bar and UIKit's tab bar use for the current
// destination. So both natives set that state for a node stating any kind,
// unless the node states AccessibilitySelected itself, which wins.
type CurrentKind string

const (
	// CurrentNone is the zero value: this item is not the current one, or says
	// nothing about it. Both web targets write no attribute, which is ARIA's
	// own spelling of aria-current="false".
	CurrentNone CurrentKind = ""

	// CurrentPage — the destination a navigation widget is showing.
	CurrentPage CurrentKind = "page"

	// CurrentStep — the step a process indicator is on.
	CurrentStep CurrentKind = "step"

	// CurrentTrue — the current item of a set that is not pages or steps.
	CurrentTrue CurrentKind = "true"
)

// CurrentKinds returns every stated value, in declaration order. CurrentNone
// is excluded for the reason PopupKinds() excludes PopupNone: it is the absence
// of a claim rather than one of the kinds.
func CurrentKinds() []CurrentKind {
	return []CurrentKind{CurrentPage, CurrentStep, CurrentTrue}
}
