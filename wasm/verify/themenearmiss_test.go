package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// themeNearMissEdits, and the derivation that produces it.
//
// # Why the number is measured and not chosen
//
// themeNearMiss reports what a path that names no leaf of core.Theme was
// probably trying to say. A threshold decides how far "probably" reaches, and a
// threshold somebody picked because it looked sensible is the shape of thing
// this file keeps replacing with a measurement. The measurements are of a
// struct's own leaf names, so the number moves when the struct does — and this
// test is where it would say so.
//
// Three readings, and each settles something different:
//
//	the smallest distance between two sibling leaves
//	    1 today, and the message's SHAPE turns on it. A struct whose closest
//	    two siblings were three apart could report one candidate and call it
//	    the answer at a threshold of one. This one cannot: a path one edit
//	    from Spacing.XS is one edit from Spacing.XL, so the honest answer at
//	    any threshold is a list.
//
//	the most siblings within themeNearMissEdits of any leaf
//	    at most one other, so the answer stays a pair of names.
//
//	the most siblings within one more than that
//	    more than one other, so the threshold is the largest that keeps that
//	    promise rather than a number with slack in it. Held from below as
//	    well as above, for the reason every tolerance in this repository is.
//
// # And the readings are the function's own now
//
// All three used to be computed here, over core.DefaultTheme, while
// themeNearMiss took a bare leaf map and read a constant. That held the
// derivation for the one caller there is and said nothing about the signature:
// a second caller with a different struct would have got core.Theme's number
// measured against somebody else's field names, and the message would have gone
// on citing Spacing.XS and Spacing.XL as its reason for hedging about a struct
// that has neither.
//
// themeLeaves does the measuring now and carries the answer with the leaves, so
// what is left for this test is the other direction: that the derivation, run
// over core.DefaultTheme, comes to the number the note above themeNearMissEdits
// explains, for the reasons it gives.
func TestThemeNearMissThresholdIsDerivedFromCoreTheme(t *testing.T) {
	set := themeLeaves(reflect.ValueOf(*core.DefaultTheme))

	// The derivation is the thing under test, so the readings are recomputed
	// here rather than read back off the struct that produced them. A crowd
	// count that agrees with itself is not evidence about anything.
	byParent := map[string][]string{}
	for _, path := range themeLeafPaths(reflect.ValueOf(*core.DefaultTheme), "") {
		cut := strings.LastIndex(path, ".") + 1
		byParent[path[:cut]] = append(byParent[path[:cut]], path[cut:])
	}

	// The most other siblings any leaf has within a given distance.
	crowd := func(within int) (int, string) {
		most, worst := 0, ""
		for parent, names := range byParent {
			for i, from := range names {
				n := 0
				for j, to := range names {
					if i != j && themeEditDistance(
						strings.ToLower(from), strings.ToLower(to)) <= within {
						n++
					}
				}
				if n > most {
					most, worst = n, parent+from
				}
			}
		}
		return most, worst
	}

	closest, pair := 1<<30, ""
	for parent, names := range byParent {
		for i := range names {
			for j := i + 1; j < len(names); j++ {
				d := themeEditDistance(
					strings.ToLower(names[i]), strings.ToLower(names[j]))
				if d < closest {
					closest, pair = d, parent+names[i]+" and "+parent+names[j]
				}
			}
		}
	}
	if closest > themeNearMissEdits {
		t.Errorf("no two sibling leaves of core.Theme are within %d edits of each "+
			"other — the closest pair is %s, %d apart.\n\n"+
			"themeNearMiss names every candidate rather than choosing one, and the "+
			"reason written above themeNearMissEdits is that this struct HAS a pair "+
			"close enough to be ambiguous. If it no longer does, the function can say "+
			"\"this is what was meant\" again and the note explaining why it cannot is "+
			"describing a struct that has moved on.", themeNearMissEdits, pair, closest)
	}
	if set.closestD != closest {
		t.Errorf("themeLeaves reports core.Theme's closest sibling pair as %d apart "+
			"(%s) and it is %d apart (%s).\n\n"+
			"That number is what the ambiguous answer cites as its reason for refusing "+
			"to choose between candidates, so a wrong one is a hedge justified by a "+
			"fact about no struct.", set.closestD, set.closest, closest, pair)
	}

	at, atWorst := crowd(themeNearMissEdits)
	if at > 1 {
		t.Errorf("%s has %d siblings within %d edits of it.\n\n"+
			"themeNearMiss prints every candidate, so the threshold is only usable "+
			"while that list is short: a name and one alternative is an answer, and a "+
			"column of them is the wall this function exists not to produce. Either "+
			"the threshold comes down or the message stops listing.",
			atWorst, at, themeNearMissEdits)
	}
	beyond, beyondWorst := crowd(themeNearMissEdits + 1)
	if beyond <= 1 {
		t.Errorf("no leaf of core.Theme has more than one sibling within %d edits "+
			"either (the most is %s, with %d), so themeNearMissEdits is %d for no "+
			"reason this test can see.\n\n"+
			"The threshold is supposed to be the largest that keeps the answer to a "+
			"pair of names. If the next one out keeps that promise too, this bound is "+
			"slack: it rules out typos — a transposition plus a case slip, two dropped "+
			"letters — that could be reported without any loss of usefulness.",
			themeNearMissEdits+1, beyondWorst, beyond, themeNearMissEdits)
	}

	// And that the derivation arrives at the constant the note explains, over
	// exactly those readings. This is the join between the measurement and the
	// prose: a themeLeaves that started answering 2 would leave the note above
	// themeNearMissEdits describing a threshold nothing uses.
	if set.edits != themeNearMissEdits {
		t.Errorf("themeLeaves measures core.Theme's near-miss threshold at %d and "+
			"themeNearMissEdits is %d.\n\n"+
			"The constant is core.Theme's answer pinned, and the note above it is the "+
			"argument for a number of that size. A derivation that disagrees means one "+
			"of the two is describing a struct that has moved: within %d edits the "+
			"most crowded leaf has %d siblings (%s), and within %d it has %d (%s).",
			set.edits, themeNearMissEdits, themeNearMissEdits, at, atWorst,
			themeNearMissEdits+1, beyond, beyondWorst)
	}
	if set.crowdAt != at || set.crowdBeyond != beyond {
		t.Errorf("themeLeaves reports %d siblings within %d edits and %d within %d; "+
			"measured here they are %d and %d.\n\n"+
			"Those two readings are what bound the threshold from each side, and a "+
			"derivation whose own numbers are wrong can land on the right answer for "+
			"the wrong reason.", set.crowdAt, set.edits, set.crowdBeyond,
			set.edits+1, at, beyond)
	}
	if set.edits >= themeNearMissReach {
		t.Errorf("the threshold measured over core.Theme is %d and themeNearMissReach "+
			"caps it at %d, so what stopped the search is the ceiling rather than the "+
			"struct.\n\n"+
			"That ceiling is the one judgement left in this derivation — where a slip "+
			"stops being one slip — and it is supposed to be slack over any struct this "+
			"repository asks about. A set that reaches it is one whose names are far "+
			"enough apart that the measurement has stopped being the binding "+
			"constraint, and the number reported is then a decision nobody made about "+
			"this struct.", set.edits, themeNearMissReach)
	}

	t.Logf("core.Theme: %d parents, closest sibling pair %d apart (%s); "+
		"within %d edits at most %d sibling, within %d at most %d",
		len(byParent), closest, pair, themeNearMissEdits, at,
		themeNearMissEdits+1, beyond)
}

// A threshold measured over names that are not core.Theme's.
//
// # The claim the signature used to make and could not keep
//
// themeLeaves' whole point is that the number is a fact about the names it is
// handed. Run over core.Theme alone that is unfalsifiable: one struct gives one
// answer, and a constant with a walk in front of it gives the same one. So two
// other sets of names go through the same derivation, chosen for a property
// each, and the property is what is asserted.
//
// These names are contrived and that is not the objection it is elsewhere in
// this file. The generated test below refuses a table of misspellings because
// the interesting inputs are the ones nobody thought of; here the input is a
// struct SHAPE — sparse in one case, crowded in the other — and what is being
// asked is whether the derivation reads the shape it is given.
func TestTheNearMissThresholdMovesWithTheNamesItIsMeasuredOver(t *testing.T) {
	// Four siblings, pairwise far apart: no two of these are within three edits
	// of each other, so nothing crowds at any distance the reach allows and the
	// search runs to the ceiling.
	sparse := themeLeafSetOf([]string{
		"Palette.Alpha", "Palette.Bravo", "Palette.CharlieDelta", "Palette.Echo",
	})
	if sparse.edits <= themeNearMissEdits {
		t.Errorf("four sibling names no two of which are within three edits of each "+
			"other measure a threshold of %d, and core.Theme's is %d.\n\n"+
			"themeLeaves is supposed to widen the threshold for a struct that can "+
			"afford it — that is the whole difference between a measurement and the "+
			"constant it replaced. A set this sparse coming back at core.Theme's "+
			"number means the derivation is not reading the names it was handed.",
			sparse.edits, themeNearMissEdits)
	}
	if sparse.edits != themeNearMissReach {
		t.Errorf("the sparse set measures %d and themeNearMissReach is %d.\n\n"+
			"Nothing in this set crowds at any distance, so the search is supposed to "+
			"run to the ceiling and stop there. A different answer means the crowd "+
			"reading is finding neighbours among names that have none.",
			sparse.edits, themeNearMissReach)
	}

	// And the other direction: names one edit apart in a crowd cannot widen,
	// whatever the ceiling says.
	crowded := themeLeafSetOf([]string{
		"Weight.Aaa", "Weight.Aab", "Weight.Aac", "Weight.Aad", "Weight.Aae",
	})
	if crowded.edits != 1 {
		t.Errorf("five sibling names each one edit from the other four measure a "+
			"threshold of %d, want the floor of 1.\n\n"+
			"Every one of them has four siblings within a single edit, so no distance "+
			"above the floor keeps an answer to a pair. A wider threshold here is a "+
			"message that prints the whole parent.", crowded.edits)
	}
	if crowded.closestD != 1 {
		t.Errorf("the crowded set's closest sibling pair measures %d apart (%s), "+
			"want 1 — these names differ in one character", crowded.closestD,
			crowded.closest)
	}

	// And that the message follows the number rather than the constant. The
	// sparse set's threshold is three, so a name two edits from one of its
	// leaves has to be reported — which at core.Theme's threshold it would not
	// be.
	got := themeNearMiss(sparse, "Palette.Alfa")
	if !strings.Contains(got, "Palette.Alpha") {
		t.Errorf("themeNearMiss over the sparse set does not name Palette.Alpha for "+
			"Palette.Alfa, which is two edits from it under a threshold of %d.\n\n"+
			"It said: %s\n\n"+
			"The function reads the threshold off the set it is given. If this answer "+
			"is core.Theme's, the number is travelling with the code rather than with "+
			"the names.", sparse.edits, got)
	}
}

// Every leaf of core.Theme, mistyped, is found again.
//
// # Why the inputs are generated and not listed
//
// The obvious fixture here is a table of misspellings — Lineheight,
// LetterSpaceing, Colours — and a table of misspellings is a list of things
// somebody thought of. This repository has twice decided that is not evidence:
// it tests the cases the author imagined, and the ones that matter are the ones
// they did not.
//
// So the inputs come from the struct. Every leaf is perturbed by each of the
// edits themeEditDistance is defined over — a doubled letter, a dropped one, a
// pair swapped — and the perturbed path is handed to themeNearMiss, which has
// to name the leaf it came from. That is 872 leaves times three slips, and none
// of them is a name anybody chose.
//
// A perturbation that lands on a real leaf is skipped: the caller only reaches
// themeNearMiss for a path core.Theme does not have.
func TestEveryThemeLeafIsFoundAgainAfterOneSlip(t *testing.T) {
	// Measured once and reused. The derivation is a per-parent quadratic over
	// 872 names and this walk asks 2616 questions of it — which is the reason
	// the measurement lives on a value rather than inside themeNearMiss.
	set := themeLeaves(reflect.ValueOf(*core.DefaultTheme))
	paths := themeLeafPaths(reflect.ValueOf(*core.DefaultTheme), "")
	sort.Strings(paths)

	tried := 0
	for _, path := range paths {
		cut := strings.LastIndex(path, ".") + 1
		parent, name := path[:cut], path[cut:]
		if len(name) < 2 {
			continue
		}
		for _, slip := range []struct{ what, typo string }{
			{"a doubled letter", name[:1] + name},
			{"a dropped letter", name[:len(name)-1]},
			{"two adjacent letters swapped", string(name[1]) + string(name[0]) + name[2:]},
		} {
			typed := parent + slip.typo
			if set.leaves[typed] {
				continue // a real leaf: this walk never asks about one
			}
			tried++
			got := themeNearMiss(set, typed)
			if !strings.Contains(got, path) {
				t.Errorf("%s with %s is %q, and themeNearMiss does not name %s.\n\n"+
					"It said: %s\n\n"+
					"One edit is what themeNearMissEdits covers and what its note "+
					"lists — a letter added, dropped or changed, or two adjacent "+
					"letters swapped. A leaf that cannot be found again after one of "+
					"them is a reader sent to look for a rename that did not happen.",
					path, slip.what, typed, path, got)
			}
		}
	}
	if tried < len(paths) {
		t.Fatalf("only %d perturbations were asked of %d leaves — this test is not "+
			"reaching the struct", tried, len(paths))
	}
	t.Logf("%d leaves, %d one-edit slips, every one traced back", len(paths), tried)
}
