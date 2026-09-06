package core

import "strconv"

// ValueRange is where a valued control sits inside its range — the fraction an
// upload has finished, the step a wizard is on.
//
// It is the fourth of core's accessibility state vocabularies, after
// SelectedState (is this control on), ExpandedState (is this disclosure open)
// and the two level ints (how deep does this sit). Those three answer yes/no or
// a single integer; this one answers "how far along, out of what", which is
// three numbers and cannot be one field.
//
// # What asked for it
//
// components.ProgressBar, which had no way to say any of it. Its accessible
// name was built as "Upload, 45 percent" — the value spelled into the *name*
// channel, which is the exact move components.Chip's ", selected" suffix was
// deleted for. A name is meant to be stable: a reader that re-announces a
// control says the whole altered name rather than the changed part, so a bar
// ticking from 44 to 45 re-announced "Upload, 45 percent" instead of "45
// percent", and no platform could act on the number because no platform could
// find it.
//
// # Why the numbers are strings
//
// For the reason Role's values and SelectedState's are ARIA's own spellings:
// the two DOM targets write them into the attribute verbatim and need no
// mapping table. The Style struct already carries numbers this way wherever
// zero is a legal value — Width, Height and MaxWidth are all strings — and
// that is the deciding reason here rather than a stylistic one:
//
//	a float64 Now of 0 is a bar at the start of an upload, and it is also the
//	zero value of the field. Style merges on "non-zero wins", so a stated 0
//	would be indistinguishable from an unstated one and would be dropped by
//	every merge in the chain.
//
// SelectedState solved the same problem the same way — SelectedOff is the
// stated string "false" and SelectedUnset is "" — and ValueOf is the
// constructor that keeps a caller from having to think about it.
//
// # Why it is one field and not four
//
// Style's two level ints merge independently *on purpose*: they answer
// disjoint questions, and dropping one because the other was set would make
// the result depend on which Style in the chain happened to name the role.
// This is the opposite case. Now, Min and Max are one fact in three parts —
// "45" means 45% under ARIA's implicit 0..100 and means nothing at all without
// knowing whether the range is 0..100 or 1..5 — so two Styles each merging half
// a range would produce a claim neither of them made. Merging as a unit is what
// makes that unwritable: a stated range replaces a stated range whole.
//
// # Text is the value channel, and it is not only for a range
//
// Text becomes aria-valuetext on the web, Compose's stateDescription and
// SwiftUI's accessibilityValue. Those last two are honoured on any node, which
// makes this the value channel GrMobStyle.swift's AccessibilityExpanded note
// says the framework does not have. The difference that makes it safe now is
// whose words they are: a renderer emitting the literal "expanded" would be
// inventing English for every app in every locale, where this string is the
// app's own — the same line AccessibilityLabel and AccessibilityHint sit on.
//
// The web is stricter than the natives here, exactly as it is for a selection:
// ARIA scopes aria-valuetext to the range roles, so a Text on an unroled
// container is dropped by a browser and announced by both phones. Each platform
// says the truest thing it can.
type ValueRange struct {
	// Now is the current position, as ARIA's aria-valuenow. Empty means
	// unstated, which for a progressbar is ARIA's own spelling of
	// "indeterminate" — a bar that is running with no idea how far.
	Now string

	// Min and Max are the ends of the range, aria-valuemin and aria-valuemax.
	// Empty on both means ARIA's defaults, which are 0 and 100 — so a bare
	// Now reads as a percentage, which is what a progress fraction wants and
	// is why ValueOf's two-argument sibling would have been a trap: "3" with
	// no range announces as 3%, not as step 3.
	Min string
	Max string

	// Text replaces the number in the announcement when the digits are not
	// what a listener wants to hear — "3 of 5", "medium", "£12.50". ARIA says
	// a reader announces this *instead of* Now, so a Text that disagrees with
	// the number is the version the user gets.
	Text string
}

// ValueOf states a range from the three numbers a caller is holding.
//
// The formatting is 'f' with the shortest round-tripping precision rather than
// %g, and that is not cosmetic: %g switches to scientific notation past six
// digits, so a byte counter would emit aria-valuenow="1.048576e+06" — which is
// not a number ARIA accepts and which no reader announces. -1 precision is what
// keeps a whole value short ("45", not "45.000000").
func ValueOf(now, min, max float64) ValueRange {
	return ValueRange{
		Now: formatValue(now),
		Min: formatValue(min),
		Max: formatValue(max),
	}
}

// WithText adds the spoken form to a range, for a control whose number is not
// what a listener wants to hear.
//
//	core.ValueOf(3, 1, 5).WithText("step 3 of 5")
//
// A method rather than a fourth argument to ValueOf, because most ranges do not
// want one: a percentage announces perfectly well as a percentage, in whatever
// language the reader is set to, and supplying an English string would take that
// localization away.
func (v ValueRange) WithText(text string) ValueRange {
	v.Text = text
	return v
}

// Stated reports whether this range says anything at all. The zero value says
// nothing, which is what every node in every tree carried before this type
// existed, and is the condition Style.Merge and both web exporters test.
func (v ValueRange) Stated() bool {
	return v != ValueRange{}
}

func formatValue(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
