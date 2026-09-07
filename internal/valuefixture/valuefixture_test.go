package valuefixture

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// A comparison harness is only as good as the arms its table reaches.
//
// android/verify runs GrMobProgress.kt over this table and reports every
// difference — which says nothing at all about a reading no case produces. The
// arm most at risk is the one that reads as an absence on the far side:
// Compose assigns nothing for both ProgressUnstated and ProgressEmptyRange, so
// a table with no empty-range case would let a Kotlin branch collapse the two
// and stay green.
func TestTheTableReachesEveryReading(t *testing.T) {
	seen := map[core.ProgressReading]int{}
	for _, c := range Cases() {
		seen[Want(c).Reading]++
	}
	for _, reading := range []core.ProgressReading{
		core.ProgressUnstated,
		core.ProgressIndeterminate,
		core.ProgressDeterminate,
		core.ProgressEmptyRange,
	} {
		if seen[reading] == 0 {
			t.Errorf("no case reads as %q — a harness comparing this table cannot see a "+
				"transliteration that gets that arm wrong", reading)
		}
	}
}

// Every case is named, and named once. A failure in the JVM pass reports the
// name and nothing else, so two cases sharing one is a report that points at
// the wrong input.
func TestEveryCaseIsNamedOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Cases() {
		if c.Name == "" {
			t.Errorf("an unnamed case (%#v) — the name is the whole of what a harness "+
				"failure can point at", c.Range)
		}
		if seen[c.Name] {
			t.Errorf("two cases named %q", c.Name)
		}
		seen[c.Name] = true
	}
}

// The clamping and the defaulting are the two rules a transliteration is most
// likely to get subtly right-looking, so the table has to hold a case where
// each of them actually changes the answer. A case list that drifted into
// only stating full, in-range triples would compare two implementations of
// pass-through.
func TestTheTableExercisesTheRulesThatTransform(t *testing.T) {
	var clamps, defaults int
	for _, c := range Cases() {
		w := Want(c)
		if w.Reading != core.ProgressDeterminate {
			continue
		}
		if c.Range.Now != "" && (c.Range.Min == "" || c.Range.Max == "") {
			defaults++
		}
		// The position moved: the stated number is not the resolved one.
		if core.ValueOf(w.Now, w.Min, w.Max).Now != c.Range.Now {
			clamps++
		}
	}
	if defaults == 0 {
		t.Error("no case leaves a bound unstated — ARIA's implicit 0..100 is what makes " +
			"a bare position announce as a percentage, and nothing here would notice a " +
			"renderer that defaulted both bounds to 0")
	}
	if clamps == 0 {
		t.Error("no case has its position clamped — a live counter that overshoots is " +
			"the reason the rule exists, and a renderer that dropped the clamp would " +
			"crash Compose rather than mis-announce")
	}
}
