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
//	date      comps.Calendar's today cell, which is ARIA's own example of
//	          the value. It is the one kind the natives do NOT fold into
//	          their selected state; see below.
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
//
// # CurrentDate is the exception to the fold
//
// Folding "today" into selected would be a lie in exactly the widget that
// states it: a calendar has a selected day already, and it is usually not
// today. So neither native sets selected for CurrentDate. They append
// ", today" to the node's accessible name instead — the suffix comps.Calendar
// used to add in Go for every target — because a name is the one channel
// both platforms read that can carry the fact at all.
//
//	target      CurrentDate becomes
//	web (both)  aria-current="date", which the screen reader announces in
//	            its own language; the name is left as the widget wrote it
//	Compose     contentDescription + ", " + ICU's word for today
//	SwiftUI     accessibilityLabel + ", " + Foundation's word for today
//
// The suffix moved rather than disappeared, and on the natives its word is
// now the platform's own, in the user's language: Foundation's
// RelativeDateTimeFormatter on SwiftUI and ICU's on Compose, the named form of
// a zero-day offset ("today", "aujourd’hui", "heute"). Only the ", " that joins
// it to the name is fixed. When it was written in Go it was English
// everywhere.
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

	// CurrentDate — the day a date grid is on, "today". Not folded into the
	// natives' selected state; see "CurrentDate is the exception to the fold".
	CurrentDate CurrentKind = "date"
)

// CurrentKinds returns every stated value, in declaration order. CurrentNone
// is excluded for the reason PopupKinds() excludes PopupNone: it is the absence
// of a claim rather than one of the kinds.
func CurrentKinds() []CurrentKind {
	return []CurrentKind{CurrentPage, CurrentStep, CurrentTrue, CurrentDate}
}
