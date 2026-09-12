// Package valuefixture is the one table of accessibility value ranges every
// harness that has to resolve one is held to.
//
// core.ValueRange.Progress (core/value.go) decides what three wire strings
// amount to: a determinate bar, ARIA's indeterminate one, an empty range, or
// no numeric claim at all. The web targets never ask — they hand the strings
// to a browser, which applies the same rules itself — so the only code
// applying them outside Go is Kotlin's, in GrMobProgress.kt, and the only
// thing that had ever looked at that branch was a substring search over the
// file it used to live in.
//
//	android/verify  runs grMobProgressOf on a JVM against Go's answers
//	core            asserts the authority itself (core/value_test.go)
//
// The expected value is never written here. Cases carry only the input, and
// Wants computes the answer with core.ValueRange.Progress, so what a harness
// compares is its target against the authority rather than against a
// hand-copied transcription of what the authority was believed to say — the
// same arrangement internal/menufixture makes for the picker menus, and for
// the same reason.
//
// It lives under internal/ because it is fixture data for this repository's
// own harnesses, not part of the framework's API.
package valuefixture

import "github.com/rohanthewiz/grmob/core"

// Case is one range and the name a failure reports it by.
//
// The three numbers are carried as the strings they cross the wire as, which
// is what a renderer's parser actually receives: the parse is part of the
// rule, and a table of float64s would check the half of it that cannot go
// wrong.
type Case struct {
	Name string
	// Range is the input, exactly as core.Style.AccessibilityValue holds it.
	Range core.ValueRange
}

// Cases is the table. Each entry names the decision it is the witness for, so
// a case deleted because it "looked like the one above" is a decision going
// unchecked with nothing to say so.
func Cases() []Case {
	return []Case{
		// The ordinary bar: a bare position, under ARIA's implicit 0..100.
		// This is what comps.ProgressBar emits.
		{"a bare percentage", core.ValueRange{Now: "45", Min: "0", Max: "100"}},
		// The defaults, actually defaulted. A renderer that read a missing
		// bound as 0 on both sides would produce an empty range here.
		{"a position with no bounds", core.ValueRange{Now: "45"}},
		{"a position with only a max", core.ValueRange{Now: "3", Max: "5"}},
		{"a position with only a min", core.ValueRange{Now: "3", Min: "1"}},
		// A stated zero. The reason every number on this type is a string:
		// a bar at the start of an upload must not read as an unstated one.
		{"a stated zero", core.ValueRange{Now: "0", Min: "0", Max: "100"}},
		// A real range that is not a percentage — step 3 of 5.
		{"a non-percentage range", core.ValueRange{Now: "3", Min: "1", Max: "5"}},
		// Clamping, both ways. A position outside its range is a live counter
		// that overshot, and it must not crash a render or announce a number
		// the range does not contain.
		{"a position above the max", core.ValueRange{Now: "150", Min: "0", Max: "100"}},
		{"a position below the min", core.ValueRange{Now: "-4", Min: "0", Max: "100"}},
		// ARIA's indeterminate bar: bounds, no position. A real state rather
		// than a missing one, and the arm most likely to be collapsed into
		// "nothing was stated" by a renderer that defaults its numbers.
		{"bounds with no position", core.ValueRange{Min: "0", Max: "100"}},
		{"a max with no position", core.ValueRange{Max: "100"}},
		{"a min with no position", core.ValueRange{Min: "0"}},
		// Nothing numeric at all. The text must not turn an ordinary node
		// into a progress bar.
		{"the empty range", core.ValueRange{}},
		{"words alone", core.ValueRange{Text: "almost done"}},
		// The words never touch the numeric reading, in either direction.
		{"words beside a position", core.ValueRange{Now: "45", Text: "almost done"}},
		// An empty and an inverted range. Compose throws on both, so the
		// reading has to be distinguishable from a determinate one before the
		// property is ever assigned.
		{"an empty range", core.ValueRange{Now: "5", Min: "5", Max: "5"}},
		{"an inverted range", core.ValueRange{Now: "5", Min: "9", Max: "1"}},
		// Unparseable numbers. Every one of these is a bar with no position,
		// which is a state ARIA already has a meaning for.
		{"a position that is not a number", core.ValueRange{Now: "half", Min: "0", Max: "100"}},
		{"a bound that is not a number", core.ValueRange{Now: "45", Max: "lots"}},
		// The non-finite spellings both parsers accept and no range property
		// can hold.
		{"a NaN position", core.ValueRange{Now: "NaN", Min: "0", Max: "100"}},
		{"an infinite bound", core.ValueRange{Now: "45", Min: "0", Max: "Inf"}},
		// Decimals and negatives, which are legal ARIA values and are where a
		// renderer that parsed with an integer parser would part company.
		{"a fractional position", core.ValueRange{Now: "45.5", Min: "0", Max: "100"}},
		{"a negative range", core.ValueRange{Now: "-5", Min: "-10", Max: "0"}},
	}
}

// Want is the authority's answer for a case.
func Want(c Case) core.Progress { return c.Range.Progress() }
