package core

import (
	"math"
	"strconv"
)

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

// ProgressReading is what a ValueRange's three numbers amount to once
// somebody has to act on them.
//
// # Why core owns this and the web exporters do not use it
//
// The two DOM targets hand aria-valuenow/-min/-max to a browser verbatim, and
// a browser applies ARIA's rules itself: the implicit 0..100, the reading of a
// missing position as an indeterminate bar. That pass-through is right and it
// is also why this reading had nowhere to live — the only code in the
// repository applying those rules was Kotlin, in a three-way branch inside a
// Compose semantics lambda, checked by looking for substrings in the file.
//
// The rules are ARIA's and this type's, though, not Compose's: both of them are
// already stated in prose on the fields below, and the Kotlin is a
// transliteration of that prose. Naming them here makes the transliteration
// comparable — android/verify runs GrMobProgress.kt against this function over
// internal/valuefixture's table — which is the same relationship
// core.SelectMenuSections has with the four picker menus.
//
// Text has no part in it. The words are a separate claim on a separate
// property (aria-valuetext, stateDescription, accessibilityValue) and they are
// announced on nodes that carry no range at all, so a reading about the numbers
// must not depend on them.
type ProgressReading string

const (
	// Nothing numeric was stated. A `text` on an ordinary node must not turn
	// it into a progress bar, so this is the reading that leaves a platform's
	// range property untouched.
	ProgressUnstated ProgressReading = "unstated"

	// Bounds and no position: a bar that is running with no idea how far.
	// ARIA spells it by omitting aria-valuenow; Compose has a name for it.
	ProgressIndeterminate ProgressReading = "indeterminate"

	// A position inside a real range. Min and Max carry ARIA's own defaults
	// of 0 and 100 when unstated, which is what makes a bare Now announce as
	// a percentage, and Now is clamped into the range.
	ProgressDeterminate ProgressReading = "determinate"

	// A position inside a range that is not one — Max at or below Min. It is
	// separated from Unstated because the two are different mistakes and a
	// platform may want to treat them differently: Compose cannot express it
	// at all (ProgressBarRangeInfo requires a non-empty range and throws), so
	// it drops the property rather than crashing a render over an
	// accessibility annotation.
	ProgressEmptyRange ProgressReading = "empty-range"
)

// Progress is a ValueRange's numbers, resolved.
//
// Now, Min and Max are meaningful when Reading is ProgressDeterminate. For
// ProgressEmptyRange they are the numbers as stated, unclamped, so a caller
// reporting the problem can name them; for the other two readings they are
// zero, which is not a position.
type Progress struct {
	Reading       ProgressReading
	Now, Min, Max float64
}

// Progress resolves the three numbers into the one claim they make.
//
// The parse is deliberately strict about what counts as a number: an empty
// string is unstated, and so is anything that does not parse — including the
// non-finite spellings ("NaN", "Inf") that Go's and Kotlin's parsers both
// accept and that no range property on any platform can hold. A bar whose
// position failed to parse is a bar with no position, which is a state ARIA
// already has a meaning for.
func (v ValueRange) Progress() Progress {
	now, hasNow := parseValue(v.Now)
	min, hasMin := parseValue(v.Min)
	max, hasMax := parseValue(v.Max)

	if !hasNow {
		if hasMin || hasMax {
			return Progress{Reading: ProgressIndeterminate}
		}
		return Progress{Reading: ProgressUnstated}
	}
	// ARIA's own defaults for an unstated bound, which is what makes a bare
	// position announce as a percentage on every target.
	if !hasMin {
		min = 0
	}
	if !hasMax {
		max = 100
	}
	if max <= min {
		return Progress{Reading: ProgressEmptyRange, Now: now, Min: min, Max: max}
	}
	if now < min {
		now = min
	}
	if now > max {
		now = max
	}
	return Progress{Reading: ProgressDeterminate, Now: now, Min: min, Max: max}
}

// Unparsed names the stated numeric fields that are not numbers this
// vocabulary can use, in the order the type declares them.
//
// # Why the reading alone is not enough to report one
//
// Progress answers what the three strings amount to, and it answers it in
// ARIA's own terms: a position that does not parse is a bar with no position,
// which is a state ARIA already has a meaning for, and a bound that does not
// parse falls back to ARIA's default. Both are the right resolution and
// neither is what the author wrote — and from the outside the two cases are
// indistinguishable from the ones where nothing was stated at all.
//
//	ValueRange{Now: "half", Min: "0", Max: "100"}   reads as indeterminate
//	ValueRange{Min: "0", Max: "100"}                reads as indeterminate
//
// The first is a mistake and the second is a bar that is genuinely running
// with no idea how far. So the fields are named separately from the reading,
// and core.AuditTree is what reports the first without reporting the second.
//
// # And it is not a harmless mistake
//
// The three targets that parse these strings do not agree about a string that
// is not a number. Compose and this package read it as absent; a browser reads
// aria-valuenow="half" as 0 and pins the bar at the start of its range, and
// reads aria-valuemax="lots" as 0 and then clamps the position down to it. So
// an unparseable number is not "no number" — it is a different wrong answer on
// each platform, with nothing anywhere saying so. wasm/verify's browser pass
// holds Chrome to that divergence case by case.
//
// Text has no part in it: it is words by design, and any string is a legal one.
func (v ValueRange) Unparsed() []string {
	var out []string
	for _, f := range []struct{ name, value string }{
		{"Now", v.Now}, {"Min", v.Min}, {"Max", v.Max},
	} {
		if f.value == "" {
			continue
		}
		if _, ok := parseValue(f.value); !ok {
			out = append(out, f.name)
		}
	}
	return out
}

// parseValue is the wire-string-to-number rule, in one place because three
// callers have to agree on it: this file, GrMobStyle.kt's parser (through
// grMobProgressNumber) and GrMobStyle.swift's.
func parseValue(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
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
