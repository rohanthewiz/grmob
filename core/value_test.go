package core

import "testing"

// ValueOf's formatting, which is the reason the numbers are strings at all.
func TestValueOfFormatsWithoutScientificNotation(t *testing.T) {
	for _, tc := range []struct {
		name          string
		now, min, max float64
		wantNow       string
		wantMin       string
		wantMax       string
	}{
		{"a whole percentage stays short", 45, 0, 100, "45", "0", "100"},
		{"a stated zero is not the zero value", 0, 0, 100, "0", "0", "100"},
		{"a fraction keeps its digits", 45.6, 0, 100, "45.6", "0", "100"},
		// The reason the formatting is 'f' and not %g: a byte counter past six
		// digits would emit "1.048576e+06", which is not a number ARIA accepts
		// and which no reader announces.
		{"a large range does not go scientific", 1048576, 0, 2097152,
			"1048576", "0", "2097152"},
		{"a negative range is a range", -5, -10, 10, "-5", "-10", "10"},
	} {
		got := ValueOf(tc.now, tc.min, tc.max)
		if got.Now != tc.wantNow || got.Min != tc.wantMin || got.Max != tc.wantMax {
			t.Errorf("%s: ValueOf(%v, %v, %v) = %#v, want %q/%q/%q",
				tc.name, tc.now, tc.min, tc.max, got, tc.wantNow, tc.wantMin, tc.wantMax)
		}
	}
}

// The whole reason the numbers are not float64 fields: a bar at the start of an
// upload and a bar that is not a bar have to be different values, and Style
// merges on "non-zero wins".
func TestAStatedZeroSurvivesAMerge(t *testing.T) {
	base := Style{AccessibilityRole: RoleProgressBar}
	over := Style{AccessibilityValue: ValueOf(0, 0, 100)}
	over.applyTo(&base)

	if !base.AccessibilityValue.Stated() {
		t.Fatal("a bar at 0 percent merged as though it had said nothing — which is " +
			"the failure a float64 Now would have had, and the reason these are strings")
	}
	if base.AccessibilityValue.Now != "0" {
		t.Errorf("Now = %q, want %q", base.AccessibilityValue.Now, "0")
	}
}

// Merged as a unit, unlike the two level ints beside it. Now, Min and Max are
// one fact in three parts — "45" is 45% out of 0..100 and step 45 out of 1..50
// — so two Styles each contributing half a range would state something neither
// of them said.
func TestARangeMergesWhole(t *testing.T) {
	base := Style{AccessibilityValue: ValueOf(3, 1, 5)}
	over := Style{AccessibilityValue: ValueRange{Now: "45"}}
	over.applyTo(&base)

	got := base.AccessibilityValue
	if got.Min != "" || got.Max != "" {
		t.Errorf("value = %#v, want the bounds replaced along with the position: a "+
			"45 left inside 1..5 is a claim neither Style made", got)
	}
	if got.Now != "45" {
		t.Errorf("Now = %q, want %q", got.Now, "45")
	}

	// And the zero value leaves a stated range alone, which is what makes the
	// field safe on every Style in a chain that is not about the value.
	unstated := Style{BorderRadius: 4}
	unstated.applyTo(&base)
	if base.AccessibilityValue.Now != "45" {
		t.Errorf("an unstated range cleared a stated one: %#v", base.AccessibilityValue)
	}
}

func TestWithTextLeavesTheNumbersAlone(t *testing.T) {
	v := ValueOf(3, 1, 5).WithText("step 3 of 5")
	if v.Now != "3" || v.Min != "1" || v.Max != "5" {
		t.Errorf("WithText disturbed the range: %#v", v)
	}
	if v.Text != "step 3 of 5" {
		t.Errorf("Text = %q", v.Text)
	}
}

// The zero value says nothing, which is what every node in every tree carried
// before this type existed — and, beside RoleProgressBar, is ARIA's own
// spelling of an indeterminate bar rather than an omission.
func TestTheZeroRangeIsUnstated(t *testing.T) {
	if (ValueRange{}).Stated() {
		t.Error("the zero ValueRange claims to state something")
	}
	for _, v := range []ValueRange{
		{Now: "0"},
		{Min: "0"},
		{Max: "0"},
		{Text: "loading"},
	} {
		if !v.Stated() {
			t.Errorf("%#v reads as unstated", v)
		}
	}
}

// --- Progress: what the three numbers amount to ----------------------------

// The authority, pinned by hand.
//
// Everything else that checks this reading — internal/valuefixture,
// android/verify's JVM pass — compares a transliteration *against* Progress,
// so this is the one place the answers themselves are written down rather than
// derived. A wrong rule here would propagate silently to every one of them and
// they would all agree.
func TestProgressReadsTheThreeNumbers(t *testing.T) {
	for _, c := range []struct {
		name      string
		in        ValueRange
		want      ProgressReading
		n, lo, hi float64
	}{
		// ARIA's implicit bounds, which is what makes a bare position
		// announce as a percentage on every target.
		{"a bare position defaults to 0..100", ValueRange{Now: "45"},
			ProgressDeterminate, 45, 0, 100},
		{"only a max still defaults the min", ValueRange{Now: "3", Max: "5"},
			ProgressDeterminate, 3, 0, 5},
		{"only a min still defaults the max", ValueRange{Now: "3", Min: "1"},
			ProgressDeterminate, 3, 1, 100},
		// A stated zero is a bar at the start of an upload. The whole reason
		// these fields are strings.
		{"a stated zero is a position", ValueRange{Now: "0", Min: "0", Max: "100"},
			ProgressDeterminate, 0, 0, 100},
		// Clamped, both ways: a live counter that overshot must not announce
		// a number outside its own range.
		{"a position above the max is clamped", ValueRange{Now: "150", Min: "0", Max: "100"},
			ProgressDeterminate, 100, 0, 100},
		{"a position below the min is clamped", ValueRange{Now: "-4", Min: "0", Max: "100"},
			ProgressDeterminate, 0, 0, 100},
		// Bounds and no position: ARIA's indeterminate bar, which is a state
		// rather than a missing one.
		{"bounds with no position", ValueRange{Min: "0", Max: "100"},
			ProgressIndeterminate, 0, 0, 0},
		{"one bound with no position", ValueRange{Max: "100"},
			ProgressIndeterminate, 0, 0, 0},
		// Nothing numeric. The text must not turn an ordinary node into a bar.
		{"the zero range", ValueRange{}, ProgressUnstated, 0, 0, 0},
		{"words alone", ValueRange{Text: "almost done"}, ProgressUnstated, 0, 0, 0},
		{"words beside a position", ValueRange{Now: "45", Text: "almost done"},
			ProgressDeterminate, 45, 0, 100},
		// A range that is not one, kept apart from "nothing was stated"
		// because a platform may need to tell them apart — Compose throws on
		// an empty range and so has to catch this before it assigns.
		{"an empty range", ValueRange{Now: "5", Min: "5", Max: "5"},
			ProgressEmptyRange, 5, 5, 5},
		{"an inverted range", ValueRange{Now: "5", Min: "9", Max: "1"},
			ProgressEmptyRange, 5, 9, 1},
		// Unparseable and non-finite are both "no position", which is a state
		// ARIA already has a meaning for.
		{"an unparseable position", ValueRange{Now: "half", Min: "0", Max: "100"},
			ProgressIndeterminate, 0, 0, 0},
		{"an unparseable bound falls back to the default",
			ValueRange{Now: "45", Max: "lots"}, ProgressDeterminate, 45, 0, 100},
		{"NaN is not a position", ValueRange{Now: "NaN", Min: "0", Max: "100"},
			ProgressIndeterminate, 0, 0, 0},
		{"an infinite bound is not a bound", ValueRange{Now: "45", Min: "0", Max: "Inf"},
			ProgressDeterminate, 45, 0, 100},
		{"fractions survive", ValueRange{Now: "45.5", Min: "0", Max: "100"},
			ProgressDeterminate, 45.5, 0, 100},
		{"a negative range", ValueRange{Now: "-5", Min: "-10", Max: "0"},
			ProgressDeterminate, -5, -10, 0},
	} {
		got := c.in.Progress()
		want := Progress{Reading: c.want, Now: c.n, Min: c.lo, Max: c.hi}
		if got != want {
			t.Errorf("%s: %#v.Progress() = %+v, want %+v", c.name, c.in, got, want)
		}
	}
}

// Stated and Progress answer different questions, and the gap between them is
// deliberate: a range carrying only words has said something (so it merges,
// and its text is announced) and has claimed no number.
func TestAWordOnlyRangeIsStatedAndUnreadable(t *testing.T) {
	v := ValueRange{Text: "almost done"}
	if !v.Stated() {
		t.Error("a range with words says nothing at all")
	}
	if got := v.Progress().Reading; got != ProgressUnstated {
		t.Errorf("reading = %q, want %q — a text on an ordinary node must not turn it "+
			"into a progress bar", got, ProgressUnstated)
	}
}
