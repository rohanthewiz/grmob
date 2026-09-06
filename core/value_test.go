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
