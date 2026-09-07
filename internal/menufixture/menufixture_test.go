package menufixture

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The fixture is a table three harnesses read and none of them owns.
//
// ios/verify, android/verify and wasm/verify each run a different
// transliteration of core.SelectMenuSections against it, and each carries a
// crude guard on the count — enough to catch an empty table, and no help at
// all against the failure that actually matters: a case quietly disappearing
// while the count stays plausible. A dropped case weakens three harnesses at
// once, silently, and none of the three is in a position to notice, because
// none of them knows what the table is *for*.
//
// So the properties the table exists to exercise are named here, each with the
// predicate that finds a case covering it. A case may be rewritten or replaced
// freely; what it may not do is take the last witness of a property with it.
//
// This is the same shape as knownBoundaryShortfalls' entry-shape test and
// rowsSpec's admission test: a statement of what a future edit has to keep
// true, written before the edit arrives.
func TestTheFixtureCoversEveryPropertyItExistsFor(t *testing.T) {
	cases := Cases()

	// A run that ends the list, which nothing follows. The flush is the piece
	// of bookkeeping each transliteration has to remember, and it is what
	// htmlout once shipped without for a release — its fixture's last option
	// happened to be ungrouped, so the flush was never reached.
	endsInARun := func(c Case) bool {
		return len(c.Options) > 0 && c.Options[len(c.Options)-1]["group"] != ""
	}

	// A heading that comes back after a different one: runs, not a gather.
	headingReturns := func(c Case) bool {
		seen := map[string]bool{}
		last := ""
		for i, o := range c.Options {
			g := o["group"]
			if i > 0 && g != last && seen[g] {
				return true
			}
			seen[g] = true
			last = g
		}
		return false
	}

	// A run disabled by an option that is not its first. This is the case that
	// separates "read the declaration when the run is closed" from "read it
	// when the run is opened" — and the second reading passes every list whose
	// first option happens to carry the flag.
	disabledByALaterOption := func(c Case) bool {
		last := ""
		for i, o := range c.Options {
			opensRun := i == 0 || o["group"] != last
			last = o["group"]
			if o["groupDisabled"] == "true" && !opensRun {
				return true
			}
		}
		return false
	}

	// A disabled run with a live one after it: the flag is per-run, so opening
	// the next one has to clear it.
	disabledThenLive := func(c Case) bool {
		sections := core.SelectMenuSections(c.Options)
		for i := 0; i < len(sections)-1; i++ {
			if sections[i].Disabled && !sections[i+1].Disabled {
				return true
			}
		}
		return false
	}

	// A disabled run with no heading, where there is no <optgroup> and no
	// header to carry the state — so it degrades to the per-item propagation,
	// which is the whole reason that propagation exists.
	disabledAndHeadless := func(c Case) bool {
		for _, s := range core.SelectMenuSections(c.Options) {
			if s.Disabled && s.Heading == "" {
				return true
			}
		}
		return false
	}

	for _, want := range []struct {
		property string
		why      string
		holds    func(Case) bool
	}{
		{"a run that ends the list",
			"the flush after the loop is unreached without one",
			endsInARun},
		{"a heading that comes back after a different one",
			"a gather would pass every case where each heading appears once",
			headingReturns},
		{"a run disabled by an option that is not its first",
			"reading the declaration as the run opens passes every other case",
			disabledByALaterOption},
		{"a disabled run followed by a live one",
			"a flag hoisted out of the loop passes when the disabled run is last",
			disabledThenLive},
		{"a disabled run with no heading",
			"nothing else forces the refusal onto the items",
			disabledAndHeadless},
		{"an option whose label falls back to its value",
			"the default lives at core.Select's flattening seam, and a hand-built node skips it",
			func(c Case) bool {
				for _, o := range c.Options {
					if o["label"] == "" && o["value"] != "" {
						return true
					}
				}
				return false
			}},
		{"an empty option list",
			"a menu of no sections has to be rangeable, and to decode as [] rather than null",
			func(c Case) bool { return len(c.Options) == 0 }},
		{"an option disabled on its own",
			"the per-option flag is a different claim from the run's",
			func(c Case) bool {
				for _, o := range c.Options {
					if o["disabled"] == "true" && o["groupDisabled"] != "true" {
						return true
					}
				}
				return false
			}},
	} {
		found := false
		for _, c := range cases {
			if want.holds(c) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no case covers %q — %s", want.property, want.why)
		}
	}
}

// Names are how a failure in any of the three harnesses says which list broke,
// and two cases sharing one makes that message point at the wrong list.
func TestCaseNamesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Cases() {
		if seen[c.Name] {
			t.Errorf("two cases are named %q", c.Name)
		}
		seen[c.Name] = true
	}
}

// The wants are computed, never written, so this checks the computation was
// actually run rather than left as the zero value — an all-empty Want would
// make every harness agree with nothing.
func TestEveryNonEmptyCaseHasAWant(t *testing.T) {
	for _, c := range Cases() {
		if len(c.Options) == 0 {
			if c.Want == nil {
				t.Errorf("%q: Want is nil, which marshals to JSON null and fails to decode "+
					"where a non-optional array is declared", c.Name)
			}
			continue
		}
		if len(c.Want) == 0 {
			t.Errorf("%q: %d options and no sections", c.Name, len(c.Options))
		}
	}
}
