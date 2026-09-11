package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/themeleaves"
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
	// And that the derivation SAYS so, which is the half the number cannot
	// carry. The line above reads the answer and infers what stopped the
	// search from its size; that inference is only available while the ceiling
	// is slack, and it is the reader of a struct sitting ON the ceiling who
	// needs to be told. See cappedByReach.
	if set.cappedByReach {
		t.Errorf("themeLeaves measures core.Theme's threshold at %d under a ceiling "+
			"of %d and reports that the ceiling is what stopped the search.\n\n"+
			"It is not: crowding is, and the reading that says so is %d siblings "+
			"within %d edits of %s. A set that reports the wrong stopping condition "+
			"puts a sentence about themeNearMissReach into a message whose number came "+
			"from these names, which sends a reader to raise a constant that is not "+
			"binding.", set.edits, themeNearMissReach, beyond, set.edits+1,
			beyondWorst)
	}
	// And that a set the ceiling did not stop says nothing about what it would
	// have afforded. The crowding already answered that: crowdBeyond is the
	// reading that ended the search, and a second number claiming these names
	// carry more would be two measurements of one thing disagreeing. See
	// afforded, which is only searched for when the ceiling is what stopped it.
	if set.afforded != set.edits {
		t.Errorf("core.Theme's threshold is %d, measured against its own crowding, "+
			"and the set reports it would afford %d.\n\n"+
			"That number exists for the sets the ceiling stopped, to say what the "+
			"ceiling is costing them. For a set that stopped on its own crowding the "+
			"answer is the threshold itself, and anything else is a second search "+
			"contradicting the one that produced the number.", set.edits, set.afforded)
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
	// The four fixtures, from the one place their names are written.
	//
	// They used to be four literals here, and affordedEndingWitness needed the
	// same four populations for a different question — which branch of the
	// classification each reaches. Two copies of a fixture list is how this
	// file came to have two copies of the ending list, and the second copy is
	// worse here than there: a set edited in one place and not the other goes
	// on passing BOTH tests, because each would be asserting about whatever it
	// was handed. The names are next door, indexed by the ending they reach.
	//
	// Four siblings, pairwise far apart: no two of these are within three edits
	// of each other, so nothing crowds at any distance the reach allows and the
	// search runs to the ceiling.
	sparse := themeLeafSetOf(affordedEndingWitness[3].names)
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

	if !sparse.cappedByReach {
		t.Errorf("the sparse set measures %d, which is the ceiling, and reports that "+
			"its own crowding stopped the search.\n\n"+
			"Nothing in it crowds within the reach, so what stopped the search is "+
			"themeNearMissReach — the one judgement left in this derivation. A set "+
			"that does not know which of the two bound it cannot say so in its "+
			"message, and \"nothing is within %d edits of it\" then invites a reader "+
			"to try four with no answer about whether four was available.",
			sparse.edits, sparse.edits)
	}
	// And what four would have been worth here, which is the question the flag
	// alone cannot answer.
	//
	// These four names look sparse and are not, one step out: Palette.Echo has
	// two siblings within four edits. So the ceiling and the crowding stop this
	// set in the same place, and the sentence the message used to print for
	// every capped set — "these names are far enough apart to carry a wider
	// one" — was false about the very set this test was written around. See
	// afforded.
	if sparse.afforded != sparse.edits {
		t.Errorf("the sparse set measures %d and reports that it would afford %d.\n\n"+
			"Its own crowding is %d siblings within %d edits (%s), which is the same "+
			"place the ceiling stops it. A capped set that reports a wider afforded "+
			"threshold than it has is the message telling a reader to raise "+
			"themeNearMissReach and find something, when raising it finds this crowd.",
			sparse.edits, sparse.afforded, sparse.crowdBeyond, sparse.edits+1,
			sparse.beyondWorst)
	}
	if sparse.affordedOpen {
		t.Errorf("the sparse set reports that no width crowds it at all, and %s has "+
			"%d siblings within %d edits.\n\n"+
			"That flag is for a set with no parent holding three leaves, where the "+
			"search runs out of distances to try rather than finding a crowd. This "+
			"parent holds four.", sparse.beyondWorst, sparse.crowdBeyond,
			sparse.edits+1)
	}

	// A set that really is far enough apart to carry more, so the arm that
	// states a width is reached by something.
	wide := themeLeafSetOf(affordedEndingWitness[2].names)
	if !wide.cappedByReach || wide.afforded <= wide.edits {
		t.Errorf("three sibling names six edits apart measure %d, capped=%v, and "+
			"afford %d.\n\n"+
			"Nothing under this parent is within five edits of anything else, so the "+
			"ceiling is what stops the search and the width these names would carry "+
			"is what the ceiling is costing. A set that cannot report that leaves the "+
			"message saying a wider threshold was available without saying how much "+
			"wider, which is the state this measurement replaced.",
			wide.edits, wide.cappedByReach, wide.afforded)
	}

	// And a set nothing crowds at any width, which is a different sentence and
	// not a bigger number: two leaves under one parent are never a crowd,
	// however far the search goes, so there is no width to report.
	pair := themeLeafSetOf(affordedEndingWitness[1].names)
	if !pair.affordedOpen {
		t.Errorf("two sibling names under one parent report an afforded width of %d "+
			"(open=%v).\n\n"+
			"A crowd is a leaf with more than one sibling inside the threshold, and a "+
			"parent with two leaves cannot produce one at any distance. The search is "+
			"supposed to run out of distances to try and say so, because a number "+
			"there would read as a measurement of where these names stop being "+
			"distinguishable — and they never do.", pair.afforded, pair.affordedOpen)
	}

	// And the other direction: names one edit apart in a crowd cannot widen,
	// whatever the ceiling says.
	crowded := themeLeafSetOf(affordedEndingWitness[0].names)
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
	//
	// The probe is built under the witness's own parent and its distance is
	// read rather than assumed: both halves of the sentence above are premises
	// about this fixture, and a respelling that broke either would leave the
	// arm below reporting the wrong finding. See affordedWitnessProbe.
	alfa, alfaD, alfaAt := affordedWitnessProbe(t, 3, "Alfa")
	if alfaD > sparse.edits {
		t.Fatalf("%s is %d edits from the nearest leaf of the sparse set (%s), "+
			"which measures a threshold of %d.\n\n"+
			"This arm is about a probe INSIDE the measured threshold — the whole "+
			"claim is that the wider number the sparse names carry is the one in "+
			"force. A probe outside it would be answered with silence, and the arm "+
			"would then be reading the not-found branch of the function while "+
			"saying it was reading the other one.",
			alfa, alfaD, alfaAt, sparse.edits)
	}
	if alfaD <= themeNearMissEdits {
		t.Fatalf("%s is %d edits from %s, and core.Theme's threshold is %d.\n\n"+
			"The point of this arm is that the sparse set's own wider threshold is "+
			"what answers: a probe this close would be reported at core.Theme's "+
			"number too, so finding it says nothing about whose threshold was used.",
			alfa, alfaD, alfaAt, themeNearMissEdits)
	}
	got := themeNearMiss(sparse, alfa)
	if !strings.Contains(got, alfaAt) {
		t.Errorf("themeNearMiss over the sparse set does not name %s for "+
			"%s, which is %d edits from it under a threshold of %d.\n\n"+
			"It said: %s\n\n"+
			"The function reads the threshold off the set it is given. If this answer "+
			"is core.Theme's, the number is travelling with the code rather than with "+
			"the names.", alfaAt, alfa, alfaD, sparse.edits, got)
	}

	// And that the message names what capped the threshold, on the arm where
	// it matters: a reader told "nothing is within three edits" will wonder
	// about four, and whether four was available is the difference between a
	// struct that is not crowded and a judgement that refused to look further.
	zulu, zuluD, zuluAt := affordedWitnessProbe(t, 3, "Zulu")
	if zuluD <= sparse.edits {
		t.Fatalf("%s is %d edits from %s and the sparse set's threshold is %d, so "+
			"this probe is a HIT.\n\n"+
			"The three arms below are about the sentence themeNearMiss prints when "+
			"it finds nothing — what capped the threshold, and what the cap is "+
			"costing. A probe inside the threshold gets a list of candidates "+
			"instead, and none of those arms is then asking about anything.",
			zulu, zuluD, zuluAt, sparse.edits)
	}
	miss := themeNearMiss(sparse, zulu)
	if !strings.Contains(miss, "themeNearMissReach") {
		t.Errorf("themeNearMiss over the sparse set finds nothing for Palette.Zulu "+
			"and does not say the threshold was the ceiling.\n\n"+
			"It said: %s\n\n"+
			"The number in that sentence is a judgement about typing accidents and "+
			"not a reading of these names. A message that reports it without its "+
			"origin reads as a measurement over these leaves, which it is not.", miss)
	}
	// And that it says what the ceiling is costing, which for THIS set is
	// nothing: its crowding stops in the same place. A reader sent to raise a
	// constant that would find a crowd is the failure this arm exists to
	// prevent.
	if !strings.Contains(miss, "stops in the same place") {
		t.Errorf("themeNearMiss over the sparse set does not say that raising the "+
			"ceiling would find nothing.\n\n"+
			"It said: %s\n\n"+
			"%s has %d siblings within %d edits, so the ceiling and this struct's own "+
			"crowding bound the threshold at the same number. The message names the "+
			"constant, which invites a reader to raise it; what it owes them is that "+
			"the names do not carry more.", miss, sparse.beyondWorst,
			sparse.crowdBeyond, sparse.edits+1)
	}

	// The other two arms of the same sentence, each reached by a set with the
	// shape it is about.
	zzzzzz, zzzzzzD, zzzzzzAt := affordedWitnessProbe(t, 2, "Zzzzzz")
	if zzzzzzD <= wide.edits {
		t.Fatalf("%s is %d edits from %s and the wide set's threshold is %d, so "+
			"this probe is a hit and the arm below is not about the sentence it "+
			"names.", zzzzzz, zzzzzzD, zzzzzzAt, wide.edits)
	}
	if wideMiss := themeNearMiss(wide, zzzzzz); !strings.Contains(
		wideMiss, fmt.Sprintf("would carry %d", wide.afforded)) {
		t.Errorf("themeNearMiss over a set that affords %d does not say so.\n\n"+
			"It said: %s\n\n"+
			"This is the arm where the ceiling is genuinely costing something, and "+
			"the width is what a reader needs to decide whether raising it is worth "+
			"doing. \"These names could carry a wider threshold\" without the number "+
			"is the state this replaced.", wide.afforded, wideMiss)
	}
	mike, mikeD, mikeAt := affordedWitnessProbe(t, 1, "Mike")
	if mikeD <= pair.edits {
		t.Fatalf("%s is %d edits from %s and the two-leaf set's threshold is %d, "+
			"so this probe is a hit and the arm below is not about the sentence it "+
			"names.", mike, mikeD, mikeAt, pair.edits)
	}
	if pairMiss := themeNearMiss(pair, mike); !strings.Contains(
		pairMiss, "no width crowds these names") {
		t.Errorf("themeNearMiss over a two-leaf parent does not say that no width "+
			"crowds it.\n\n"+
			"It said: %s\n\n"+
			"A number there would read as a measurement of where these names stop "+
			"being distinguishable, and there is no such distance: a parent with two "+
			"leaves never produces a crowd. The shape of the set is the answer.",
			pairMiss)
	}
	zzz, zzzD, zzzAt := affordedWitnessProbe(t, 0, "Zzz")
	if zzzD <= crowded.edits {
		t.Fatalf("%s is %d edits from %s and the crowded set's threshold is %d, so "+
			"this probe is a hit.\n\n"+
			"The arm below is about what the not-found sentence does NOT say, and a "+
			"probe that is found prints no such sentence — it would pass for the "+
			"reason an assertion over an empty population always passes.",
			zzz, zzzD, zzzAt, crowded.edits)
	}
	if crowdedMiss := themeNearMiss(crowded, zzz); strings.Contains(
		crowdedMiss, "themeNearMissReach") {
		t.Errorf("themeNearMiss over the crowded set says the ceiling capped its "+
			"threshold.\n\n"+
			"It said: %s\n\n"+
			"That set is at the floor because five names sit one edit apart, which is "+
			"a fact about the names. Citing the ceiling there sends a reader to raise "+
			"a constant that would change nothing.", crowdedMiss)
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

// The two relations `afforded` rests on, over sets nobody chose.
//
// # A number with one reader and no assertion
//
// afforded is what themeNearMissReach costs a struct: the width its own names
// would have carried with the ceiling taken off, measured only when the ceiling
// is what stopped the search. affordedOpen is the other answer that search can
// end with — it ran out of distances to try rather than finding a crowd, which
// is a fact about the SHAPE of the set and not a width.
//
// Both reach exactly one reader, themeNearMiss's `why` clause, and the test
// above holds them over four sets somebody wrote down: a sparse one, a wide
// one, a pair and a crowd. Four sets are four cases an author thought of, which
// is the evidence this file has twice decided is not evidence — see the note
// above TestEveryThemeLeafIsFoundAgainAfterOneSlip. What is not held anywhere
// is the pair of relations the number rests on, and they are properties of the
// derivation rather than of core.Theme:
//
//	afforded >= edits, always
//	    afforded is a width the threshold could have been, so a set that
//	    affords LESS than it measures is a message telling a reader the
//	    ceiling cost them something while the ceiling was raising the answer.
//	    Where the crowding stopped the search the two are equal by
//	    construction, and that half is asserted too, because "equal by
//	    construction" is a claim about an assignment somebody can move.
//
//	affordedOpen exactly when no parent holds three leaves
//	    a crowd is a leaf with more than one sibling inside the threshold, so
//	    a parent with two leaves cannot make one at any distance. The flag is
//	    the search running out of distances, and running out is available
//	    only to a set no width crowds. Both directions matter: reporting "no
//	    width crowds these names" about a set some width does crowd sends a
//	    reader to stop looking, and failing to report it about a set nothing
//	    crowds prints a number that reads as a measurement of where the names
//	    stop being distinguishable, which is nowhere.
//
// # Where the sets come from
//
// core.Theme's own leaf names, re-parented into shapes it does not have.
// Windows of one to six consecutive names are taken over the sorted leaves and
// mounted under a synthetic parent — and, for the halves that need two parents
// to be interesting, split across two. The names are real and every shape is an
// accident of where the window fell, which is the same move the slip walk
// makes: the inputs are the struct's, and the cases are nobody's.
// How wide a window of consecutive leaf names is taken, and the floor each of
// the four endings is held to.
//
// The window bound is the knob the whole population hangs from: at 6 it puts
// 930 sets in front of the relations, and the four endings come to 27, 473, 53
// and 377 on core.Theme's current leaves. Neither number is asserted anywhere
// and the thinnest arm is the one that matters — an ending reached by 27 sets
// is separating the relations, an ending reached by 1 is a relation passing
// over a population that can no longer break it, and the two look identical
// from outside a census that only fires at zero.
//
// So a floor is stated. Ten, which is well under what every ending scores today
// and far enough above 1 that a population losing an arm has to lose it
// noticeably rather than silently. This is INK_ROW_ROUNDING's argument in
// another directory: a margin that is silently spent is a margin nobody notices
// leaving, and the way this one gets spent is somebody narrowing the window —
// or core.Theme gaining or losing leaves — with the test still green.
//
// # And one floor is the wrong shape for four endings this far apart
//
// Ten is a real bracket under 27 and is nothing at all under 473. An ending
// that fell from 473 to 11 would be a population that had lost nineteen
// twentieths of itself — a relation asked of an arm that has all but gone —
// and the floor above would say nothing about it. The two failures this census
// exists to notice are "the ending is gone" and "the ending is going", and a
// constant can only see the first for the thinnest arm.
//
// So the floor moves with the ending, in the shape INK_OWN_MEASURED_ON uses one
// directory over: a number taken against ONE build, recorded as such, and the
// bracket derived from it rather than typed in beside it. See
// affordedMeasuredOn.
const (
	affordedWindowMax   = 6
	affordedEndingFloor = 10
	// What fraction of its own measurement an ending may fall to before it is
	// reported.
	//
	// A third. What this has to separate is a population that moved — core.Theme
	// gains a leaf, the window bound is nudged, and every ending moves with the
	// count of sets — from a population that lost an ARM, which is one ending
	// collapsing while the others hold. The first is proportional and small; the
	// second takes two thirds of one number away.
	//
	// The sets scale with the leaf count (930 is 2×(6n−15) at n=80), so a third
	// is roughly core.Theme shrinking to a quarter of its leaves before an
	// ending that is holding its share says anything. That is a re-measure and
	// the message asks for one, with the leaf count then and now beside it, so
	// the reading is never "a relation broke" when what happened is a struct
	// changing size.
	//
	// # And a fraction is still a fraction of one population
	//
	// One number, chosen once, applied to four endings — which is the shape
	// complaint above arriving one level up. INK_OWN_SHAPE_FLOOR's note makes
	// the same admission about its quarter and then says what keeps it a
	// measurement: the pair of populations it has to separate, both in hand on
	// every run, and both asserted rather than argued. A third had neither. It
	// is under every ending today and nothing said what it was above.
	//
	// So the two edges are named and held, and they are the two findings this
	// census exists to tell apart:
	//
	//	a population that MOVED   affordedScale — every ending is a count of
	//	                         sets off one walk, so a struct that changed
	//	                         size moves all four together and by that much.
	//	                         The floor has to stay UNDER what a merely-moved
	//	                         ending would score, or a re-measure arrives
	//	                         looking exactly like a collapse.
	//	a population on the       affordedBracket.share — an ending whose share
	//	stated floor              falls under the constant is held to a number
	//	                         that is not a measurement of IT. That is right
	//	                         for a thin arm and it is the derivation going
	//	                         quiet when it is true of all four, which is the
	//	                         state this replaced with nobody deciding it.
	//
	// Neither edge is protected by the arithmetic. max(constant, measured/3) is
	// the constant the moment a measurement is small, and it is above a moved
	// population the moment the population moved by more than the share.
	affordedEndingShare = 3
	// How close to 1.00 the scale has to be before the population is called
	// unchanged.
	//
	// A ratio of two integer set counts, so an untouched walk gives exactly 1
	// and the tolerance is not carrying a rounding — it is carrying the reader.
	// The sentence this gates says "same leaves, same window, same 930 sets",
	// which is a claim about the walk and would be a lie at 0.999; a thousandth
	// is under any change the walk can actually make (one leaf moves 930 by 12,
	// which is 1.3%) and above every double this division can produce.
	affordedScaleSame = 0.001
	// And how far the two spellings of a multi-leaf scale may sit apart before
	// the composition they license is called a different arithmetic.
	//
	// A part in a million million. The k-step scale is S(n−k)/S(n) and the
	// product of the one-step scales is the same rational number with the
	// intermediate counts cancelled — so the difference is rounding and
	// nothing else, and on this machine it is one ulp. The tolerance is far
	// above that and far below anything the bands compare at, which is four
	// decimal places. See the arm that reads it: what it protects is the
	// chain bound and the measured band dividing by one number rather than by
	// two roundings of one.
	affordedScaleCompose = 1e-12
	// And how far an ending may sit from its own prediction before the log line
	// stops calling the classification unmoved — for an ending nothing has
	// measured.
	//
	// The floors are asked one ending at a time and a redistribution that stays
	// above all four of them passes without a word — see affordedTakers — so
	// the residual is the only place a green run can say the sort has moved.
	// What it needs is a band, and this constant used to be it: a tenth, on the
	// argument that a leaf added mid-name shifts which windows crowd "by a
	// percent or two" and that a tenth was well above that. Neither number was
	// in hand, and when the measurement was taken (affordedOneLeafBand) the
	// argument turned out to be wrong in the direction that matters — a tenth
	// is well above the drift for the two large endings and roughly a third of
	// it for the two small ones, so an honest one-leaf edit would have made the
	// log line call a standing-still walk re-sorted.
	//
	// So the band is now per ending and measured on every run, and this is what
	// is left: the fallback for an ending affordedBandMeasuredOn carries no
	// reading for, which is a reworded ending — the same state
	// affordedEndingBracket falls back to the stated floor in. It is a stated
	// number and the arm that uses it says so rather than printing it like a
	// measurement, because "one number is not a band for four endings this far
	// apart" is precisely the finding, and an ending with no measurement is the
	// one case where a reader cannot tell that from the number.
	affordedResidualQuiet = 0.10
)

// What each ending scored when this census was written, and over what.
//
// The counts are the population's own measurement and the only one anybody has
// taken. INK_OWN_MEASURED_ON's note argues the shape at length: a bracket
// derived from a recorded measurement moves when somebody re-measures, and a
// bracket typed in beside it stays where it was written while everything it was
// about moves out from under it.
//
// `leaves` and `window` are here because a count is meaningless without them.
// The sets are windows of one to `window` consecutive names over core.Theme's
// distinct leaf names, so both numbers are inputs to every count in the map,
// and a run that disagrees about either is a run whose endings cannot be
// compared with these. The message says which changed rather than reporting a
// number that moved.
//
// # And the names themselves, which is what a count of them could not do
//
// `leaves: 80` was the whole of what this record said about its population,
// and every reading built on it had to argue from that one integer. Three of
// them were arguing about set MEMBERSHIP:
//
//	the exact arm     ran whenever this run had 80 leaves, on the premise that
//	                  the walk is a function of the names — over a population
//	                  it only knew had the same COUNT of names. A leaf renamed
//	                  keeps the count and changes the strings the walk is over,
//	                  and that arm would have reported themeLeafSetOf re-sorting
//	                  a walk nobody re-sorted.
//	the added-leaf    is a proof because the record's population is this run's
//	arm               names with the new leaf dropped — which is true when the
//	                  record's names are a SUBSET of this run's and was assumed
//	                  from the count being one lower. A leaf added and another
//	                  renamed is +1 with no subset, and the assertion is a
//	                  tolerance again with nothing saying so.
//	the removed-leaf  had no arm at all, and the reason given was that the
//	reading           removed name is not in this run to put back — so the band
//	                  in force was the 78↔79 step standing in for the 79↔80 one.
//
// All three are the same missing fact: which names. So they are recorded, and
// the walk is a pure function of them — which means the record's own census can
// be RE-DERIVED on every run, whatever core.Theme has become since. That is the
// strongest reading in this file and it no longer needs the struct to have
// stood still: see the arm that walks these names directly.
//
// The order is the order affordedLeafNames produces (paths sorted, leaf taken
// off each, first occurrence kept) and not alphabetical, because the walk takes
// windows of CONSECUTIVE names — a re-ordering of this list is a different
// population. Nothing asserts the order separately; the re-derivation above
// asserts it along with everything else, because a list in a different order
// produces different counts.
//
// # And re-taking it is a paste, not a transcription
//
// Eighty names and four counts is a lot to copy by hand, and nothing generated
// them: the obvious fix — a `-update` flag that rewrites this literal — is the
// one thing this record cannot have, because a record a test can rewrite
// re-baselines an accident. So a failing run prints the whole of it in source
// shape instead and a person pastes it. See affordedRetakeSource: what is
// automated is the copying, and what stays with a reader is the decision.
var affordedMeasuredOn = struct {
	leaves int
	window int
	names  []string
	ending map[string]int
}{
	leaves: 80,
	window: 6,
	names: []string{
		"Background", "Border", "ControlBorder", "Error", "ErrorOnLight", "Primary",
		"PrimaryOnLight", "Secondary", "Success", "SuccessOnLight", "Surface",
		"TextPrimary", "TextSecondary", "Warning", "WarningOnLight",
		"AccessibilityControls", "AccessibilityExpanded",
		"AccessibilityHeadingLevel", "AccessibilityHidden", "AccessibilityHint",
		"AccessibilityID", "AccessibilityLabel", "AccessibilityNestingLevel",
		"AccessibilityRole", "AccessibilitySelected",
		"AccessibilitySelectionFollowsFocus", "Max", "Min", "Now", "Text", "Align",
		"AlignItems", "AlignSelf", "Animation", "BorderColor", "BorderRadius",
		"BorderWidth", "Bottom", "ColumnGap", "Disabled", "Display", "FlexBasis",
		"FlexDirection", "FlexGrow", "FlexShrink", "FlexWrap", "FocusStyle",
		"FontSize", "FontWeight", "Gap", "Height", "HoverStyle", "JustifyContent",
		"Left", "LineHeight", "Horizontal", "Right", "Top", "Vertical", "MaxHeight",
		"MaxWidth", "MinHeight", "MinWidth", "Overflow", "Position", "PseudoStates",
		"Rotate", "RowGap", "Shadow", "StackAlign", "TextColor", "Transition",
		"WhiteSpace", "Width", "ZIndex", "LG", "MD", "SM", "XL", "XS",
	},
	ending: map[string]int{
		"its own crowding stopped the search":                 27,
		"no width crowds these names at all":                  473,
		"the ceiling cost this set a wider threshold":         377,
		"the ceiling and the crowding stop in the same place": 53,
	},
}

// affordedSetCount is how many sets the walk below produces from a given number
// of distinct leaf names at a given window.
//
// Both arrangements of every window of one to `window` consecutive names, which
// is exactly what the loop does: 2×(leaves−w+1) at each width. A function
// rather than the 930 it comes to today, because a count of this population is
// unreadable without what it was counted over — and the scale below divides
// this run's population by the one affordedMeasuredOn was taken over, which is
// a number no constant can be asked for after the fact.
//
// The walk is asserted against it. A formula that has come apart from the loop
// it describes makes the scale a ratio between a real population and an
// imaginary one, and every reading derived from it a fiction stated in numbers.
func affordedSetCount(leaves, window int) int {
	return 2 * affordedWindowCount(leaves, window)
}

// affordedScale is this run's population as a fraction of the one
// affordedMeasuredOn was taken over.
//
// One number for the whole census. Every ending is a count of sets and all the
// sets come off one walk, so a struct that gained or lost leaves — or a window
// somebody narrowed — moves all four endings together and by this much. That is
// the difference between the two findings the floor exists to tell apart, and
// until it was computed the distinction was made in prose by a note that could
// only say "may be".
//
// Zero when the recorded population is empty, which is a record that cannot be
// scaled against and is reported where it is read rather than divided by.
func affordedScale(leaves int) float64 {
	return affordedRatioOf(affordedSetCount(leaves, affordedWindowMax),
		affordedSetCount(affordedMeasuredOn.leaves, affordedMeasuredOn.window))
}

// affordedRatioOf is one population as a fraction of another.
//
// One division, in one place, for every ratio between two set counts in this
// file — affordedScale's, and the two the band's walk takes at each step. It
// is not tidiness: the k-leaf arms are assertions BECAUSE the residual they
// bracket is one of the numbers the band is the maximum of, and that is only
// true while the scale the test divides by and the scale the band divided by
// are the same number. They are two spellings of one division today and the
// argument that they agree is an argument about IEEE rounding — which is
// exactly the shape of argument that failed the last time this file trusted
// one: see affordedResidualOf, where a multiply and the subtract after it were
// fused into a single rounding at one spelling and not at the other, and one
// ulp was the whole distance between a band held and a green run reporting a
// re-sort nobody made.
//
// Zero when there is nothing to divide by, which is a population that cannot be
// scaled against and is reported where it is read rather than divided by.
func affordedRatioOf(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

// affordedComposeSpread is how far the two spellings of a two-leaf scale sit
// apart, over every population size this walk can produce.
//
// The chain composition needs S(n−2)/S(n) to be S(n−1)/S(n) times
// S(n−2)/S(n−1). That is exact in the rationals — the middle count cancels —
// and in floats it is four roundings against one. affordedScaleCompose is the
// bracket, and until this existed it was a tolerance with one number behind
// it: the difference at n = 80, on one machine, called "one ulp" in a comment.
//
// A bracket is a share of a measurement in this file, and this is the
// measurement. Every n from three to `leaves` — eighty divisions, which is
// nothing beside the walk that reads them — and the worst of them is what the
// constant has to sit above. The size that produced it comes back too, because
// a worst case at n = 4 and a worst case at n = 80 are different findings:
// the first is the small-population edge and the second is the walk this file
// actually takes.
func affordedComposeSpread(leaves, window int) (worst float64, at int) {
	for n := 3; n <= leaves; n++ {
		sets := affordedSetCount(n, window)
		mid := affordedSetCount(n-1, window)
		small := affordedSetCount(n-2, window)
		if sets == 0 || mid == 0 || small == 0 {
			continue
		}
		down := affordedRatioOf(mid, sets)
		step := affordedRatioOf(small, mid)
		whole := affordedRatioOf(small, sets)
		composed := down * step
		if whole == 0 {
			continue
		}
		if off := math.Abs(composed-whole) / whole; off > worst {
			worst, at = off, n
		}
	}
	return worst, at
}

// What that spread came to, and the margin the bracket keeps over it.
//
// Recorded rather than left in the arm for the reason every other measurement
// here is: a number the run computes and nobody holds is a number that has
// already moved. Held to within affordedComposeDrift rather than exactly,
// because this is the one recorded number in the file that is not a count or a
// rounded band — Go may fuse the multiply and the subtract that reads it into
// a single rounding, and it is free to do so on one architecture and not
// another, which is the hazard this whole census exists about arriving inside
// the measurement OF it.
var affordedComposeMeasuredOn = struct {
	worst  float64
	at     int
	margin float64
}{
	worst: 1.4803e-16,
	// Five names, which is the smallest population this spread is measured
	// over and not the eighty the walk takes. The counts are smallest there
	// and the relative rounding largest, which is the edge somebody would have
	// guessed at and nobody had looked at — and it is still one ulp, so the
	// answer is that the size does not matter. That is worth having as a
	// measurement rather than as the assumption the constant used to carry.
	at: 5,
	// How far affordedScaleCompose has to sit above the worst spread before it
	// is a bracket rather than a number somebody liked. Three decades, and the
	// measurement clears it by a factor of seven: the spread is a rounding of
	// a division and the tolerance is meant to catch an arithmetic that has
	// stopped being one, and there is nothing in between those two that a
	// tighter margin would separate.
	margin: 1e3,
}

// How far the two spellings of a multi-leaf scale may be from each other
// before the composition they license is called a different arithmetic. See
// affordedComposeMeasuredOn for what it is a bracket over.
const affordedComposeDrift = 10.0

// affordedPopulationUnchanged is whether the scale says this run walked the
// same population the record was taken over.
//
// One spelling, because it was two: affordedShortfallCause decided which of
// three findings to name with it and the log line decided which sentence to
// print with it, and both compared the same derived float against the same
// constant. Two evaluations of that absolute difference are two numbers the
// compiler is free to round differently — see affordedPredictedFrom, where
// exactly that cost a green run a sentence about a re-sort that did not happen
// — and here the two would disagree about whether the walk stood still, in a
// failure message and in the line printed beside it.
//
// The tolerance is affordedScaleSame and its note says why a thousandth: the
// scale is a ratio of two integer set counts, so an untouched walk gives
// exactly 1 and what the tolerance carries is the reader rather than a
// rounding.
func affordedPopulationUnchanged(scale float64) bool {
	return math.Abs(scale-1) < affordedScaleSame
}

// affordedRecordTotal is what affordedMeasuredOn's four counts add up to.
//
// Every set the walk produces reaches exactly one of the four endings — the
// switch that classifies them has a default arm, so the counts are a partition
// of the population and not a sample of it. That makes this sum the population
// the record was taken over, and it is asserted against affordedSetCount rather
// than assumed: a re-measure that mistyped one ending would leave every
// prediction below scaled against a total nobody walked, and the error would be
// invisible because each ending's own number still looks like a count.
//
// It is also the premise the residual reading rests on. Because the record is a
// partition and this run's counts are a partition of the same walk scaled, the
// four residuals sum to zero — so an ending that came in under its prediction
// has, necessarily, a partner that came in over one. That is what lets
// affordedShortfallCause name where the sets went instead of asserting that
// they left.
func affordedRecordTotal() int {
	n := 0
	for _, count := range affordedMeasuredOn.ending {
		n += count
	}
	return n
}

// affordedPrediction is how many sets an ending would reach if the only thing
// that had happened were the population moving.
//
// The ending's own recorded count times the whole census's scale. One walk
// produces all four endings, so a struct that gained or lost leaves moves every
// ending by the same ratio and leaves each of them at its own share of the new
// total — which is exactly this number, and is the null hypothesis every
// shortfall below is read against.
//
// Zero-with-false when there is no reading for the ending, because a prediction
// derived from a measurement nobody took is unavailable rather than zero.
func affordedPrediction(ending string, scale float64) (float64, bool) {
	measured, known := affordedMeasuredOn.ending[ending]
	if !known || scale == 0 {
		return 0, false
	}
	return affordedPredictedFrom(measured, scale), true
}

// affordedPredictedFrom is that multiply, in one place.
//
// The one arithmetic rule this file has learned the hard way: two floats that
// are COMPARED have to come from one evaluation. affordedResidualOf's note
// says what happens otherwise — Go may fuse a multiply and the subtract after
// it into a single rounding, it did so at one spelling of this product and not
// at another, and against a band that is the maximum of a family the residual
// belongs to, one ulp was the whole distance between "inside its band" and a
// green run reporting a re-sort that did not happen.
//
// That was fixed at the residual and prevented nowhere. This is the same
// product, and it had four spellings: affordedPrediction's, the one
// affordedShortfallCause decided a floor with, the one the share-edge arm
// decided the SAME floor with in the opposite direction, and the four failure
// messages. See TestNoFloatAComparisonRestsOnIsDerivedTwice for what now
// stops a fifth appearing.
func affordedPredictedFrom(recorded int, scale float64) float64 {
	return float64(recorded) * scale
}

// affordedTaker is one ending that came in ABOVE what a merely-moved population
// predicts for it, and by how much.
type affordedTaker struct {
	ending string
	got    int
	over   float64
}

// affordedTakers is every such ending, the largest surplus first.
//
// # The reading the scale on its own cannot make
//
// affordedScale is one number for the whole census, and that is right for the
// change it was written for: a window nudged or core.Theme gaining leaves moves
// all four endings together, and the ratio is the whole of what happened. It is
// wrong for the change that actually threatens this census. A rewrite of
// themeLeafSetOf that reshapes WHICH ending a set reaches moves the four
// against each other while the population is identical — same leaves, same
// window, same 930 sets, scale 1.00 — and "this ending should be where it was"
// is then the constant argument wearing a measurement's clothes. Every
// shortfall reads as an arm having gone, and the sentence sends a reader after
// a relation that is being asked as often as ever, of different sets.
//
// The residual is what separates the two, and it is available because both
// censuses are partitions of the same walk: the four predictions add up to this
// run's own population (see affordedRecordTotal), so the surpluses and the
// shortfalls are the same pixels counted twice. An ending that fell did not
// lose its sets to the walk — they are in this list, under another name.
//
//	the population moved      every residual near zero; the ratio is the whole
//	                          of what happened, and a shortfall under the floor
//	                          is the record needing a re-take
//	the classification moved  one ending down and a named ending up by about as
//	                          much; the walk produced the same sets and
//	                          themeLeafSetOf sorted them somewhere else
//
// Endings the record does not name are left out rather than predicted at zero:
// they have no measurement to scale, the key-set arms above report them, and a
// surplus computed against nothing would be the largest number in the list.
func affordedTakers(reached map[string]int, endings []string, scale float64) []affordedTaker {
	takers := []affordedTaker{}
	for _, ending := range endings {
		predicted, ok := affordedPrediction(ending, scale)
		if !ok {
			continue
		}
		if over := float64(reached[ending]) - predicted; over > 0 {
			takers = append(takers, affordedTaker{
				ending: ending, got: reached[ending], over: over})
		}
	}
	sort.Slice(takers, func(i, j int) bool { return takers[i].over > takers[j].over })
	return takers
}

// affordedBracket is the floor one ending is held to, why it is that number,
// and which of the two arms produced it.
//
// The arm is here because it is the half no message could see. The larger of a
// constant and a share reads as one number from outside, and the two say
// different things about the ending: on the share it is held to a fraction of
// its OWN measurement, and on the constant it is held to a number that is not a
// measurement of it at all. Which of those is in force decides what a shortfall
// means and whether the derivation is doing anything for this ending.
type affordedBracket struct {
	floor int
	// What the ending scored when the record was taken, and whether there is
	// such a reading at all. A reworded ending has neither, and everything
	// derived from a measurement is unavailable rather than zero.
	measured int
	known    bool
	share    bool
	why      string
}

// affordedEndingBracket is that bracket, for one ending.
//
// The larger of the stated floor and a share of what this ending scored when it
// was measured: the constant is what keeps a small arm from sliding to 1, and
// the share is what keeps a large one from losing most of itself unremarked.
// Carries the reason as well as the number, because a message that prints a
// bound without saying which of the two produced it leaves a reader unable to
// tell "this ending is thin" from "this ending has collapsed".
func affordedEndingBracket(ending string) affordedBracket {
	measured, known := affordedMeasuredOn.ending[ending]
	if !known {
		return affordedBracket{floor: affordedEndingFloor, why: fmt.Sprintf(
			"the stated floor of %d — affordedMeasuredOn carries no reading for this "+
				"ending, so there is no share of a measurement to take",
			affordedEndingFloor)}
	}
	share := measured / affordedEndingShare
	if share <= affordedEndingFloor {
		return affordedBracket{floor: affordedEndingFloor, measured: measured,
			known: true, why: fmt.Sprintf(
				"the stated floor of %d, which is above the %d that a third of this "+
					"ending's own measurement (%d) comes to",
				affordedEndingFloor, share, measured)}
	}
	return affordedBracket{floor: share, measured: measured, known: true,
		share: true, why: fmt.Sprintf(
			"a third of the %d this ending scored when affordedMeasuredOn was taken",
			measured)}
}

// affordedShortfallCause is which of the three findings a shortfall actually
// is, decided rather than hedged.
//
// A count under its floor is one of three things, and the first two look
// identical from the count alone:
//
//	the population moved          the walk got smaller — a window narrowed,
//	                              core.Theme lost leaves — and every ending
//	                              came down with it. affordedMeasuredOn is
//	                              what needs re-taking, not the relation.
//	the classification moved      the walk produced the same sets and
//	                              themeLeafSetOf sorted them into different
//	                              endings. The relation is still asked as
//	                              often; it is asked of other sets, and some
//	                              named ending is holding the difference.
//	the arm went                  neither: the sets that used to reach this
//	                              ending are not on the walk and are not
//	                              anywhere else either.
//
// affordedMeasuredNote can only say the first "may be" the case. The scale says
// whether it is, because a moved population predicts a number and the number is
// in hand — and affordedTakers says whether it is the second, because the four
// counts are a partition of one walk and a shortfall therefore has to have a
// partner. The third is the one that cannot happen while affordedRecordTotal's
// assertion holds, and it is still spelled out: an arithmetic that has come
// apart deserves a sentence saying which arithmetic, not a fallthrough into one
// of the other two readings.
//
// `reached` is the whole census rather than this ending's count, because the
// question "did these sets leave, or did they go somewhere" cannot be asked of
// one number.
func affordedShortfallCause(b affordedBracket, scale float64, got string,
	reached map[string]int, endings []string) string {
	count := reached[got]
	if !b.known || scale == 0 {
		return "\n\nWhich of the three this is cannot be said here: " +
			"affordedMeasuredOn has no reading for this ending, so there is no " +
			"count to scale and no prediction to hold this one against."
	}
	// Through affordedPrediction rather than spelled here, for the reason
	// affordedResidualOf carries: this comparison and the one in the test's own
	// share-edge arm are the same number against the same floor, read in
	// opposite directions, and two spellings of a float multiply are two
	// numbers the compiler is free to round differently.
	moved, _ := affordedPrediction(got, scale)
	if moved < float64(b.floor) {
		return fmt.Sprintf("\n\nThis is the population having moved rather than "+
			"this arm having gone: the census is at %.2f× the one the record was "+
			"taken over, which puts this ending's own %d at about %.0f — under the "+
			"floor of %d before any relation is asked. Every ending is scaled by "+
			"that same number, so the thing to re-take is affordedMeasuredOn and "+
			"not this walk.", scale, b.measured, moved, b.floor)
	}
	// Where the missing sets went. The four endings partition one walk, so a
	// count under its prediction is matched, set for set, by counts over
	// theirs; naming them is the difference between "this relation has stopped
	// being asked" and "this relation is being asked of the sets that used to
	// answer some other question".
	takers := affordedTakers(reached, endings, scale)
	if len(takers) > 0 {
		said := make([]string, 0, len(takers))
		for _, t := range takers {
			predicted, _ := affordedPrediction(t.ending, scale)
			said = append(said, fmt.Sprintf("%q is at %d against a prediction of "+
				"%.0f, +%.0f", t.ending, t.got, predicted, t.over))
		}
		// The population is the thing that decides how to read the surplus: at
		// 1.00× the walk is untouched and the only thing that can have moved is
		// the sort, and at any other ratio the two changes are on top of each
		// other and the residual is what is left after the scale is taken off.
		where := fmt.Sprintf("The population is at %.2f× the one the record was "+
			"taken over, so the scale is not the whole of it", scale)
		if affordedPopulationUnchanged(scale) {
			where = "The population is the one the record was taken over — same " +
				"leaves, same window, same %d sets — so this is not the walk at all"
			where = fmt.Sprintf(where, affordedSetCount(affordedMeasuredOn.leaves,
				affordedMeasuredOn.window))
		}
		return fmt.Sprintf("\n\nAnd these sets did not leave the walk. %s: this "+
			"ending predicts about %.0f and %d arrived, and %s. That is "+
			"themeLeafSetOf reshaping WHICH ending a set reaches rather than an arm "+
			"going — the relation is still being asked, of the sets now answering "+
			"somewhere else, and the ending to look at is the one holding them. "+
			"Read the two derivations against each other before re-taking "+
			"affordedMeasuredOn: a record re-taken over a classification that has "+
			"moved records the new shape as the baseline.",
			where, moved, count, strings.Join(said, "; "))
	}
	return fmt.Sprintf("\n\nAnd it is this arm and not the census: the population "+
		"is at %.2f× the one the record was taken over, which predicts about %.0f "+
		"sets for this ending and %d arrived, and no other ending is above its own "+
		"prediction.\n\nThe four endings partition one walk, so a shortfall with no "+
		"partner is arithmetic that has come apart rather than a finding about the "+
		"derivation — affordedRecordTotal's assertion above is the one that should "+
		"have fired first, and reading this as an arm going would be a sentence "+
		"about core.Theme built on a division nobody can stand behind.",
		scale, moved, count)
}

// affordedMeasuredNote is the sentence a failure adds when this run is not the
// run those numbers were taken on.
//
// The same move inkOwnBuildNote makes: the bracket is a share of a measurement,
// so the first thing a reader needs is whether the measurement still describes
// the population. A struct that lost half its leaves halves every ending, and
// that is a re-measure rather than a relation coming apart — but it arrives
// looking exactly like one.
func affordedMeasuredNote(leaves, sets int) string {
	if leaves == affordedMeasuredOn.leaves && affordedWindowMax == affordedMeasuredOn.window {
		return ""
	}
	return fmt.Sprintf("\n\naffordedMeasuredOn was taken over %d distinct leaf names "+
		"at a window of %d; this run has %d at a window of %d, and produced %d sets. "+
		"Every ending scales with that population, so the shortfall above may be the "+
		"struct having changed size rather than an arm having gone — in which case "+
		"the census in the log line is the new measurement and affordedMeasuredOn is "+
		"what needs re-taking.",
		affordedMeasuredOn.leaves, affordedMeasuredOn.window,
		leaves, affordedWindowMax, sets)
}

// affordedLeafNames is core.Theme's distinct leaf names, sorted.
//
// Deduplicated because two parents can hold the same leaf name
// (Colors.Surface and Colors.Overlay.Surface), and a window carrying it twice
// would mount one parent with two identical children — a shape the walk below
// can never produce, and one whose zero distance is a different check's
// business.
//
// This is the SECOND of the three copies of the rule "recurse on a struct,
// take the last dotted segment, keep each name once" — the deduplication on
// top of gen.go's themeLeafPaths, and the population everything in this file
// is measured over. internal/themeleaves' reflectLeafNames is the third, which
// is the two of these written again over reflect with nothing shared. The
// table naming all three is in its comment; every claim in this file about
// "the population" is about the answer THIS one gives.
func affordedLeafNames() []string {
	paths := themeLeafPaths(reflect.ValueOf(*core.DefaultTheme), "")
	sort.Strings(paths)
	names := []string{}
	seen := map[string]bool{}
	for _, path := range paths {
		name := path[strings.LastIndex(path, ".")+1:]
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

// The two expansions of core.Theme are one population.
//
// # What rests on this, and what was holding it
//
// affordedBandSteps' table of what the struct has actually done — eight
// one-leaf commits, two of two, one of four, one of twenty-eight — comes out of
// internal/themehistory, which expands core.Theme by PARSING core/'s sources at
// a revision. affordedLeafNames expands the same struct over reflect. Two
// mechanisms, and the table is a fact about this file's population only while
// they produce the same eighty names.
//
// That was verified once, by hand, with a diff: run the command with -names,
// eyeball it against this file's list, believe the table from then on. It is
// the same shape as a number written into a note, and this file's whole
// convention is that such a number has already moved.
//
// # Why it can be an arm when the table cannot
//
// The command needs git and a test must not: a shallow clone, a source tarball
// or a build container has no history in it. But the git half is only where the
// SOURCES come from. themeleaves.InDir takes them from the working tree
// instead, and the expansion it runs is the identical code path — so the half
// that has to shell out stays a command, and the half that can be checked is
// checked on every run.
//
// It is also the half that already went wrong: the first walker came back with
// ninety-one names, because its recursion guard was reassigned inside the field
// loop and Typography's three TextStyle fields stopped being three subtrees.
// Eleven names too many, in the mechanism the whole table is derived by.
//
// # What each direction of a disagreement means
//
// They are not symmetric, and the failure says which happened:
//
//	only the source walk has it   a field whose type it could not resolve to a
//	                              struct declared in core/ — an alias, a
//	                              generic, a type from another package. It
//	                              stopped there and kept the FIELD's name;
//	                              reflect went in and kept the children.
//	only reflect has it           the children behind exactly that stop, unless
//	                              some other parent happens to hold the same
//	                              name too.
//
// So the ninety-one-name bug reads as eleven names the source walk has and
// reflect does not, which is the first line of the failure below.
func TestTheHistoryWalkersExpansionIsTheOneThisFileMeasures(t *testing.T) {
	// Relative to this package's directory, which is where `go test` runs.
	// core/ is the package Theme is declared in and nothing outside it
	// contributes a field, which is the same premise internal/themehistory
	// reads only core/ on.
	const coreDir = "../../core"
	exp, err := themeleaves.InDir(coreDir, "Theme")
	if err != nil {
		t.Fatalf("could not read %s to expand Theme from its sources: %v\n\n"+
			"This is the working-tree half of internal/themehistory's reading — "+
			"no git, just the .go files that are there — so a failure here is the "+
			"path having moved rather than anything about the struct.", coreDir, err)
	}
	if len(exp.Unparsed) > 0 {
		t.Fatalf("go/parser returned nothing for %s.\n\n"+
			"The expansion below is over whatever the other files declared, so "+
			"every count it produces is short by however much lives in these. In a "+
			"HISTORY that is ordinary — a revision caught mid-refactor — and "+
			"internal/themehistory says so and carries on. In a working tree it "+
			"means this checkout does not compile, and no reading taken over it is "+
			"worth comparing to anything.", strings.Join(exp.Unparsed, ", "))
	}
	if !exp.Found {
		t.Fatalf("no struct named Theme is declared in %s.\n\n"+
			"reflect finds one — affordedLeafNames is expanding core.DefaultTheme "+
			"right now — so the type has been moved to another package, or renamed, "+
			"and internal/themehistory is walking the whole history looking for a "+
			"name that is not there any more. Its table would come back empty "+
			"rather than wrong, which is the failure that looks like nothing "+
			"happened.", coreDir)
	}

	// Sorted, because the claim is about the POPULATION and not about the
	// order. affordedLeafNames returns its names in the order the sorted PATHS
	// produce them — Colors.Background before Colors.Border before
	// Spacing.LG — which is the order the windows are cut in and is a fact
	// about the paths rather than about the names. themeleaves has no paths to
	// sort by, and an arm that failed over the two orderings would be a
	// failure about nothing.
	want := slices.Clone(affordedLeafNames())
	sort.Strings(want)
	if slices.Equal(exp.Names, want) {
		t.Logf("core.Theme expands to the same %d leaf names both ways: parsed "+
			"out of %s by internal/themeleaves, and walked over reflect by "+
			"affordedLeafNames. That is what the edit-size table under "+
			"affordedBandSteps rests on — it is a reading of the parsed "+
			"population at sixteen revisions, and it is a fact about the "+
			"population this file measures only while these two agree. The git "+
			"half stays a command (go run ./internal/themehistory -names); this "+
			"is the half that does not need one.", len(want), coreDir)
		return
	}

	inSource := affordedNamesNotIn(exp.Names, want)
	inReflect := affordedNamesNotIn(want, exp.Names)
	t.Errorf("core.Theme expands to %d leaf names parsed out of %s and %d walked "+
		"over reflect.\n\n"+
		"only the source walk has: %s\nonly reflect has:        %s\n\n"+
		"These two are supposed to be one population, and the whole of "+
		"internal/themehistory's table — the edit-size distribution "+
		"affordedBandSteps stops at three leaves because of — is the parsed one "+
		"read at sixteen revisions.\n\n"+
		"A name only the SOURCE walk has is a field whose type it could not "+
		"resolve to a struct declared in core/: an alias, a generic, a type from "+
		"another package. It stopped there and kept the field's own name where "+
		"reflect recursed and kept the children — so those children are the names "+
		"only reflect has, and the two lists above are usually one edit read from "+
		"both sides. That is the ninety-one-name shape the walker shipped with "+
		"once.\n\n"+
		"A name only REFLECT has, with nothing beside it in the other list, is "+
		"the other direction: a field this walk never saw at all, which means "+
		"Theme is now assembled from more than %s.",
		len(exp.Names), coreDir, len(want),
		affordedNameList(inSource), affordedNameList(inReflect), coreDir)
}

// affordedNamesNotIn is the names of one sorted list that the other does not
// hold, as its own list.
//
// Both directions are wanted and each is read as a different finding, so the
// difference is taken twice rather than returned as a pair — see the failure
// above, which says what each side means.
func affordedNamesNotIn(these, those []string) []string {
	have := make(map[string]bool, len(those))
	for _, name := range those {
		have[name] = true
	}
	out := []string{}
	for _, name := range these {
		if !have[name] {
			out = append(out, name)
		}
	}
	return out
}

// affordedSets is the population: every window of one to `window` consecutive
// names, mounted under one parent and under two.
//
// The second arrangement is what puts leaves in front of the derivation that
// are NOT candidates for one another — the only comparison the measurement
// makes is within a parent, and a set with one parent cannot show that `span`
// and the crowd search read different populations.
//
// A function rather than a loop inside the test because the band measured
// below has to walk populations this run does not have: every population one
// leaf away from this one (see affordedOneLeafBand). A band measured with a
// second copy of this loop would be a reading about a walk nobody asserts,
// which is what affordedSetCount's own assertion exists to prevent one level
// down.
//
// The whole list, for the one caller that wants it: the test's own walk, which
// counts what it produced against affordedSetCount. Every other caller takes
// the sets one at a time through affordedEachSet, which is the same loop —
// this is a collector around it and not a second copy.
func affordedSets(names []string, window int) [][]string {
	sets := make([][]string, 0, affordedSetCount(len(names), window))
	affordedEachSet(names, window, func(set []string) {
		sets = append(sets, set)
	})
	return sets
}

// affordedEachSet is that walk, one set at a time and without building any of
// them.
//
// # Why this is not the loop it replaced
//
// The k-leaf band walks C(80,2) = 3160 populations of 78 names and takes a
// census of each, which is about 2.9 million sets — and the loop that made
// them allocated a fresh pair of slices per window and a fresh string per leaf
// in it, roughly thirteen million small allocations for a reading that keeps
// four numbers.
//
// What makes the allocation avoidable is that the prefixes are a function of
// the NAME and not of the window it appears in:
//
//	one   "One."+name at every position, so a window is a sub-slice of the
//	      prefixed list — no copy, no strings built per set
//	two   "Left." on the even positions OF THE WINDOW and "Right." on the odd
//	      ones, which is the absolute position's parity XORed with the window
//	      start's. Two prefixed lists cover both cases, and a window starting
//	      at i takes the one for i's parity.
//
// So three lists of n strings are built per population and every set is a
// sub-slice of one of them: n allocations rather than the 2(6n−15) pairs the
// windows come to. The sets are identical, in the same order, and they are
// read-only on every path — themeLeafSetOf takes its input as map keys and
// sub-strings and mutates nothing — so sharing the backing arrays is safe and
// affordedSets can hand the sub-slices straight out.
//
// The parity trick is worth checking on paper. A window starting at i puts the
// name at absolute index k = i+j at window position j, so j is even exactly
// when k and i have the same parity. `alt[0]` spells the even absolute indices
// "Left." and is therefore right for an even i; `alt[1]` is its mirror and is
// right for an odd one.
//
// # And what it bought, which is not what it was expected to buy
//
// Measured over one 78-name population on verifyTimingsTakenOn's machine,
// `go test -bench`:
//
//	the loop this replaced   96.7us   4057 allocations
//	this walk                 7.4us    243 allocations
//	the whole census        2530.0us  60556 allocations
//
// So the sets were 3.8% of the census's time and 6.7% of its allocations, and
// end to end the two-leaf band barely moved: three runs of 5.5s, 6.0s and 6.8s
// before against 5.5s, 5.6s and 5.7s after — a few percent, and most of what
// is visible is the spread narrowing rather than the mean falling. The five
// seconds are themeLeafSetOf, which allocates two maps and a distance matrix
// per set and is called 2.9 million times — and that is the derivation under
// test rather than test-support code, so it is not this file's to make
// cheaper.
//
// The walk is kept anyway, and the reason is the one this file gives for the
// buffer in affordedKLeafBand: a reading that runs on every green run should
// not leave thirteen million allocations behind for four numbers.
//
// What the measurement pointed at was the real lever, and it is not allocation:
// a k-drop leaves most of its windows identical to the full population's, so
// all but about thirty of the 906 classifications per population are re-asked
// answers. That is affordedWhole now, and it is where the five seconds went.
// This walk is what the base population still goes through — and what the
// direct census the corrections are held against is still taken with.
func affordedEachSet(names []string, window int, yield func([]string)) {
	if window <= 0 || len(names) == 0 {
		return
	}
	mount := affordedPrefixOf(names)
	for width := 1; width <= window && width <= len(names); width++ {
		for i := 0; i+width <= len(names); i++ {
			yield(mount.one[i : i+width])
			yield(mount.alt[i%2][i : i+width])
		}
	}
}

// affordedParent is which of the two parents a leaf sits under at position j
// of a set.
//
// The whole of the second arrangement's rule, in one place, because there are
// now two ways of applying it: affordedPrefixOf builds a list per window-start
// parity and takes sub-slices of it, and the drop census below gathers by
// index because its windows are not contiguous in the list they came from.
// Two mechanics for one rule — and the rule is the thing that has to agree,
// since a set with the parents swapped is the mirror of this one and every
// reading in this file is blind to the difference. See the arrangement arm in
// the test, which is what actually holds the outcome.
func affordedParent(j int) string {
	if j%2 == 0 {
		return "Left."
	}
	return "Right."
}

// affordedPrefixed is a population's names under the three mountings a window
// can need, built once so that a window is an index rather than a string.
//
//	one       "One."+name, the single-parent arrangement
//	alt[p]    the two-parent arrangement for a window whose START has parity
//	          p: the name at absolute index k sits at window position k−i, so
//	          its parent is affordedParent((k+p)%2) — the absolute parity
//	          XORed with the start's, which is what makes two lists enough.
type affordedPrefixed struct {
	one []string
	alt [2][]string
}

// affordedPrefixOf builds them.
func affordedPrefixOf(names []string) affordedPrefixed {
	m := affordedPrefixed{one: make([]string, len(names))}
	m.alt[0] = make([]string, len(names))
	m.alt[1] = make([]string, len(names))
	for k, name := range names {
		m.one[k] = "One." + name
		for parity := 0; parity < 2; parity++ {
			m.alt[parity][k] = affordedParent((k+parity)%2) + name
		}
	}
	return m
}

// affordedMountAt is the two arrangements of a window that is NOT contiguous
// in the population it came from: the leaves at `at`, in that order.
//
// The same rule as the sub-slices above and reached the other way round. A
// window position j holds the name at absolute index q, so its parent must be
// affordedParent(j) — and the prefixed list that spells it that way is the one
// for parity (j+q)%2, because alt[p][q] is affordedParent((q+p)%2) and
// (q + (j+q)) ≡ j (mod 2).
//
// Filled into buffers the caller owns, because the census below calls this
// about thirty times per population and three thousand populations of fresh
// slices is the allocation this file already took out of the walk once.
func affordedMountAt(mount affordedPrefixed, at []int, one, two []string) (
	[]string, []string) {
	one, two = one[:0], two[:0]
	for j, q := range at {
		one = append(one, mount.one[q])
		two = append(two, mount.alt[(j+q)%2][q])
	}
	return one, two
}

// affordedWhole is one population's walk, kept so that the walks over its
// sub-populations do not have to repeat it.
//
// # The reading that made this worth writing
//
// The two-leaf band walks 3160 populations of 78 names and takes a census of
// each, which is 2.9 million classifications and about five seconds. The
// previous attempt at that cost went after the allocation and found it was
// 3.8% of the walk: the time is themeLeafSetOf, which is the derivation under
// test and not this file's to make cheaper.
//
// What IS this file's is how often it asks. Dropping a leaf leaves almost
// every window of the population exactly where it was — the windows are runs
// of consecutive names, so only the ones that SPAN the gap are new, and there
// are about w−1 of those per width. Thirty of the 906 sets per population are
// questions nobody has already answered.
//
// So the whole population is classified once, per window, and a sub-population
// is that census with two corrections:
//
//	take out   every window of the whole that holds a dropped name. It is not
//	           a window of the smaller population at all.
//	put in     every window of the smaller population whose names are NOT
//	           consecutive in the whole. It is a set nothing has classified.
//
// Everything else is a window of the whole that survived, in the same order,
// holding the same names — so it reaches the same ending, because the ending
// is a function of the set and the set is a function of the names.
//
// # Why the two corrections are exactly complementary
//
// A window of the whole holds no dropped name exactly when its names are all
// kept, and a run of consecutive kept names is exactly a window of the smaller
// population whose kept indices are consecutive. So the windows taken out are
// the complement of the windows kept, and the windows put in are the
// complement of those within the smaller population. The two tests below are
// the two halves of that one sentence, and each is written against its own
// definition rather than derived from the other.
//
// # And what holds it
//
// Not an argument. The census this produces is compared against the direct
// walk over every one of the record's eighty one-drop populations, and over
// every drop of one to four names from a smaller list — where the family is
// enumerable at each of those sizes, and the index arithmetic is the same
// arithmetic because it does not depend on the names. See the two arms in the
// test. Beyond that, affordedBandMeasuredOn.step[2]'s eight numbers were
// measured by the direct walk before this existed and are re-derived by this
// one on every run.
type affordedWhole struct {
	names  []string
	window int
	mount  affordedPrefixed
	// base[width] is where that width's windows start in the flat arrays.
	base []int
	// Which ending each window reached, under one parent and under two.
	one, two []uint8
	census   affordedCensus
	windows  int
	// Every window shape a drop can have to put BACK, classified once. See
	// affordedWhole.memoise, which is where the remaining cost of this file's
	// widest walks went.
	spanning map[uint64][2]uint8
}

// affordedDropScratch is the working room one goroutine needs to take a drop
// census, reused across populations.
type affordedDropScratch struct {
	// Which windows of the whole have already been taken out, so a window
	// holding two dropped names is not subtracted twice. Reset through
	// `touched` rather than by clearing 906 bools per population.
	spent   []bool
	touched []int
	kept    []int
	one     []string
	two     []string
	// How many windows this goroutine had to classify because the memo did not
	// hold them. Counted rather than prevented: a miss is still the right
	// answer, so this is a reading about the enumeration and not a correctness
	// gate. See the arm in the test — it is zero today and the enumeration is
	// the sort of thing that stops covering silently.
	missed int
}

// affordedScratchFor is that room, sized for one population.
func affordedScratchFor(w *affordedWhole) *affordedDropScratch {
	return &affordedDropScratch{
		spent: make([]bool, w.windows),
		kept:  make([]int, 0, len(w.names)),
		one:   make([]string, 0, w.window),
		two:   make([]string, 0, w.window),
	}
}

// affordedSpanKey is an index list as one integer, for the memo below.
//
// The width in the low byte and up to seven indices above it, one byte each.
// False when the shape does not fit — a window wider than seven or a
// population of more than 256 names — and the caller then classifies the
// window directly, which is what it did before this table existed. A memo that
// silently mapped two different windows onto one key would be a census of a
// population nobody walked, so the encoding refuses rather than truncates.
func affordedSpanKey(at []int) (uint64, bool) {
	if len(at) > 7 {
		return 0, false
	}
	key := uint64(len(at))
	for _, q := range at {
		if q < 0 || q > 255 {
			return 0, false
		}
		key = key<<8 | uint64(q)
	}
	return key, true
}

// memoise classifies every window shape a drop of up to k names can put back,
// once, so that the walks below stop re-asking.
//
// # The reading that made this worth writing, which is not the one that was
// expected
//
// affordedWhole already took the two-leaf band from five seconds to under
// half of one by classifying the whole population once and correcting it per
// drop: about thirty of the 906 sets per population are questions nobody has
// already answered. The next thing to try was the census's own bookkeeping —
// the corrections key their counts by the ending's own sentence, which is a
// map lookup on a fifty-character string, and an ending index looked like it
// would take the per-population cost from 150us to about 80.
//
// It is not where the time is. Profiled over the k = 2 walk, the string map is
// 0.3% of it and themeEditDistance plus the allocation under it is most of the
// rest — the thirty windows a drop puts back are thirty calls to
// themeLeafSetOf, and that is the derivation under test rather than this
// file's to make cheaper. (The index is here anyway: it costs nothing, and
// once the classifications below are memoised away the arithmetic per
// population is all that is left.)
//
// What IS this file's is, again, how often it asks — one level further down
// than last time. The windows a drop puts back are not new sets. A window of
// kept indices {a, b, c} is put back by EVERY population that drops something
// between a and c and keeps all three, and there are many of those:
//
//	k = 2   3160 populations put back about 30 windows each, which is 190
//	        thousand classifications over 3665 distinct index lists
//	k = 3   82160 populations, 7.4 million classifications, 8705 lists
//
// So the census is a correction of the whole population's, and now the
// corrections are themselves a lookup. The shapes are a function of the
// POSITIONS and the window bound — which is the same fact that lets a
// fourteen-name family stand as evidence for an eighty-name one — so they can
// be enumerated before the walk starts and the table is read-only while the
// workers run, with no lock between them.
//
// # What is enumerated
//
// A window put back is a run of w consecutive KEPT indices that is not
// consecutive in the whole, so it is an increasing list i_1 < … < i_w whose
// span exceeds its width and whose interior holes are all dropped. With at
// most k names dropped, the total hole is between 1 and k:
//
//	width w    1 … window
//	gap g      1 … k, distributed over the w−1 interior slots
//	start      anywhere the whole list still fits
//
// Width one has no interior slot and therefore no spanning window, which is
// the arithmetic saying what the sentence above says: one kept index is always
// a run of one.
//
// # And it is not trusted to be complete
//
// A window the table does not hold is classified directly and counted (see
// affordedDropScratch.missed), so an enumeration that stopped covering
// produces the same numbers more slowly rather than the wrong numbers. The
// count is carried out of the walk and asserted at zero, because "the two
// enumerations agree" is exactly the sort of claim that goes quiet.
func (w *affordedWhole) memoise(k int) {
	n := len(w.names)
	if k < 1 || w.window < 2 || n < 2 {
		return
	}
	w.spanning = map[uint64][2]uint8{}
	at := make([]int, 0, w.window)
	one := make([]string, 0, w.window)
	two := make([]string, 0, w.window)
	// The interior gaps of one shape, filled slot by slot. `left` is how much
	// of the budget is unspent and `slots` how many interior slots remain, so
	// the recursion below places a hole count in each and the last slot may
	// take anything left including nothing.
	var place func(start, slots, left int)
	place = func(start, slots, left int) {
		if slots == 0 {
			if left != 0 {
				return // budget not spent exactly; a wider gap is its own shape
			}
			key, ok := affordedSpanKey(at)
			if !ok {
				return
			}
			one, two = affordedMountAt(w.mount, at, one, two)
			w.spanning[key] = [2]uint8{
				uint8(affordedEndingAt(themeLeafSetOf(one))),
				uint8(affordedEndingAt(themeLeafSetOf(two))),
			}
			return
		}
		for hole := 0; hole <= left; hole++ {
			next := start + hole + 1
			if next >= n {
				break
			}
			at = append(at, next)
			place(next, slots-1, left-hole)
			at = at[:len(at)-1]
		}
	}
	for width := 2; width <= w.window && width <= n; width++ {
		for gap := 1; gap <= k && width+gap <= n; gap++ {
			for first := 0; first+width+gap <= n; first++ {
				at = append(at[:0], first)
				place(first, width-1, gap)
			}
		}
	}
}

// affordedSpanningAt is what a window put back reaches, from the memo where it
// is held and from the derivation where it is not.
func (w *affordedWhole) spanningAt(at []int, sc *affordedDropScratch) (uint8, uint8) {
	if w.spanning != nil {
		if key, ok := affordedSpanKey(at); ok {
			if reached, held := w.spanning[key]; held {
				return reached[0], reached[1]
			}
		}
	}
	sc.missed++
	one, two := affordedMountAt(w.mount, at, sc.one, sc.two)
	sc.one, sc.two = one, two
	return uint8(affordedEndingAt(themeLeafSetOf(one))),
		uint8(affordedEndingAt(themeLeafSetOf(two)))
}

// affordedWholeOf walks a population once and keeps what each window reached.
func affordedWholeOf(names []string, window int) *affordedWhole {
	w := &affordedWhole{
		names:  names,
		window: window,
		mount:  affordedPrefixOf(names),
		base:   make([]int, window+1),
	}
	n := len(names)
	for width := 1; width <= window && width <= n; width++ {
		w.base[width] = w.windows
		w.windows += n - width + 1
	}
	w.one = make([]uint8, w.windows)
	w.two = make([]uint8, w.windows)
	for width := 1; width <= window && width <= n; width++ {
		for s := 0; s+width <= n; s++ {
			at := w.base[width] + s
			w.one[at] = uint8(affordedEndingAt(themeLeafSetOf(w.mount.one[s : s+width])))
			w.two[at] = uint8(affordedEndingAt(themeLeafSetOf(w.mount.alt[s%2][s : s+width])))
			w.census[w.one[at]]++
			w.census[w.two[at]]++
		}
	}
	return w
}

// censusDropping is the census of this population with `drop` — sorted,
// distinct positions — taken out, written into a map the caller owns.
//
// It returns how many windows it took out and how many it put in, which is
// the accounting the identity below holds: the whole has W(n) windows and the
// smaller population W(n−k), so out − in has to come to W(n) − W(n−k) whatever
// the drop was. An off-by-one in either range moves one of the two and not the
// other.
func (w *affordedWhole) censusDropping(drop []int, sc *affordedDropScratch,
	into *affordedCensus) (out, in int) {
	n := len(w.names)
	*into = w.census
	// Out: every window of the whole that holds a dropped name.
	sc.touched = sc.touched[:0]
	for _, d := range drop {
		for width := 1; width <= w.window && width <= n; width++ {
			lo, hi := d-width+1, d
			if lo < 0 {
				lo = 0
			}
			if hi > n-width {
				hi = n - width
			}
			for s := lo; s <= hi; s++ {
				at := w.base[width] + s
				if sc.spent[at] {
					continue // already taken out for another dropped name
				}
				sc.spent[at] = true
				sc.touched = append(sc.touched, at)
				into[w.one[at]]--
				into[w.two[at]]--
				out++
			}
		}
	}
	for _, at := range sc.touched {
		sc.spent[at] = false
	}
	// In: every window of the smaller population that spans a gap.
	sc.kept = sc.kept[:0]
	next := 0
	for i := 0; i < n; i++ {
		if next < len(drop) && drop[next] == i {
			next++
			continue
		}
		sc.kept = append(sc.kept, i)
	}
	m := len(sc.kept)
	for width := 1; width <= w.window && width <= m; width++ {
		for i := 0; i+width <= m; i++ {
			if sc.kept[i+width-1]-sc.kept[i] == width-1 {
				continue // consecutive in the whole, so already counted
			}
			in++
			// Through the memo, which is where all but a few thousand of these
			// classifications went. See affordedWhole.memoise.
			one, two := w.spanningAt(sc.kept[i:i+width], sc)
			into[one]++
			into[two]++
		}
	}
	return out, in
}

// affordedWindowCount is how many windows a population of that many names has
// at that window bound — the same walk affordedSetCount counts sets over, and
// exactly half of it, because every window is mounted twice.
func affordedWindowCount(leaves, window int) int {
	n := 0
	for w := 1; w <= window && w <= leaves; w++ {
		n += leaves - w + 1
	}
	return n
}

// affordedEndingNames is the four sentences a set can reach, in the order the
// census below counts them.
//
// One list, because there were two: the switch that classifies a set spelled
// them and the test declared its own slice of the same four to iterate. The
// pair was held together by an arm comparing the record's keys against the
// test's list, which says nothing about the SWITCH — a fifth sentence added to
// the classification and not to the test's slice would go uncounted, and every
// partition arm in the file would still add up because it only ever asks about
// the four it was told about.
//
// An array rather than a slice so that its length is a constant the census
// below can be sized by.
var affordedEndingNames = [...]string{
	"its own crowding stopped the search",
	"no width crowds these names at all",
	"the ceiling cost this set a wider threshold",
	"the ceiling and the crowding stop in the same place",
}

// affordedCensus is a count per ending, by index rather than by sentence.
//
// The census a walk produces used to be a map[string]int keyed by the ending's
// own sentence, which is how every record and message in this file still reads
// it. Inside the walks it is an array: the drop census copies a whole census
// per population and adds and subtracts about a hundred and twenty counts into
// it, and a copy of an array of four ints is a register move where a copy of a
// map is an allocation and four hashes of a fifty-character string.
//
// It is also what makes a census comparable with `==`. Two of the readings
// below are "these two walks produced the same census", and maps.Equal over a
// map nobody can be sure has no zero-valued key is a comparison with a corner;
// an array has no such corner, because an ending no set reached is a 0 in a
// fixed slot rather than a key that may or may not be there.
type affordedCensus [len(affordedEndingNames)]int

// affordedEndingAt is which of the four sentences a set reaches, as an index.
//
// The switch has a default arm, so every set reaches exactly one — which is
// the partition every prediction in this file rests on. Lifted out of the test
// loop for the same reason affordedSets was: the band walks eighty populations
// and has to classify them the way the census does, and two copies of a switch
// is two classifications that can drift apart while both look right.
//
// An index and not the sentence, because the walks below classify millions of
// sets and then do arithmetic per ending: the sentence is what a reader and a
// record need, and it is one array lookup away (see affordedEndingOf).
func affordedEndingAt(set themeLeafSet) int {
	switch {
	case !set.cappedByReach:
		return 0
	case set.affordedOpen:
		return 1
	case set.afforded > set.edits:
		return 2
	default:
		return 3
	}
}

// affordedEndingOf is that sentence.
func affordedEndingOf(set themeLeafSet) string {
	return affordedEndingNames[affordedEndingAt(set)]
}

// affordedCensusMap is a census in the shape the records and the messages read
// it: keyed by the ending's own sentence.
//
// Every ending is present, including one no set reached. A census printed with
// a key missing reads as a walk that was never asked the question rather than
// as an ending nothing reached, and the two are different findings — the
// floors arm exists for the second.
func affordedCensusMap(c affordedCensus) map[string]int {
	out := make(map[string]int, len(c))
	for e, count := range c {
		out[affordedEndingNames[e]] = count
	}
	return out
}

// affordedEndingWitness is one population per ending: entry e is a set of leaf
// names that reaches affordedEndingAt's branch e, together with the sentence
// affordedEndingNames stands at that index and the shape the names have.
//
// # What holds the switch to the list, which was nothing
//
// affordedEndingAt returns 0, 1, 2 and 3 as literals and affordedEndingNames
// lists four sentences in that order, and the two were held together by
// position alone. That coupling was INTRODUCED when the classification became
// an index: before it the switch returned the sentence itself, and the order
// of the test's list of four was harmless.
//
// A reordering of the array now silently re-labels every census, band and
// record in this file. Most of the way there, something does fire — the
// record's census is keyed by the sentence and re-walked on every run, so the
// counts arrive under the wrong headings and that comparison fails. What it
// does not survive is a re-take: a person who reorders the array, sees the
// census arm fail and pastes the new record has a file that is internally
// consistent, states the wrong sentence for every population it counts, and
// has no arm left that would say so.
//
// So the four sentences are written down a second time HERE, beside a set
// that reaches each, and that second copy is the assertion rather than a
// duplicate to be removed. It is the one copy the retake paste does not
// rewrite (see affordedRetakeSource, which prints affordedMeasuredOn and
// affordedBandMeasuredOn and nothing else), which is exactly the property
// that makes it worth having.
//
// # Why these four sets
//
// They are the four branches, reached by the smallest thing that reaches
// each, and they are the same populations TestTheNearMissThresholdMovesWith-
// TheNamesItIsMeasuredOver builds its four fixtures out of — which is why the
// names live here rather than in that test. Two copies of a fixture list is
// how this file came to have two copies of the ending list.
var affordedEndingWitness = [len(affordedEndingNames)]struct {
	// The sentence at this index, typed rather than read off the array. See
	// above: reading it from affordedEndingNames would make every arm below
	// a comparison of the array with itself.
	sentence string
	names    []string
	// What about these names puts them on this branch, for a failure that
	// has to say whether the derivation moved or the fixture stopped being
	// the shape it was chosen for.
	shape string

	// # And the countable half of that sentence, so the fixture is held to
	// # its shape and not only to the branch it happens to reach
	//
	// `shape` above is prose, and the arms below asserted which ending each
	// set reaches and nothing else. A fixture that drifted into reaching the
	// RIGHT ending by the wrong route went on passing with the sentence
	// beside it describing a set that no longer exists: Duo.Alpha/Duo.Zulu
	// edited to three names under one parent still reaches ending 1 — a
	// parent of three CAN crowd, so it would be reaching it because nothing
	// happens to be within any width rather than because no width can — and
	// nothing here would have said so.
	//
	// Every field below is re-derived from `names` on every run by
	// affordedWitnessShapeOf. Between them they pin the whole of each shape
	// sentence:
	//
	//	perParent   how many leaves sit under each parent, most first. This
	//	            is "five siblings", "one parent holding two leaves", and
	//	            it is also the WHOLE of "cannot produce a crowd at any
	//	            distance": a crowd is a leaf with more than one sibling
	//	            inside the threshold, so a parent of two can never make
	//	            one however far the search goes.
	//	closestD    the closest sibling pair, and
	//	farthestD   the farthest. Equal when the sentence says the names are
	//	            all one distance apart ("each one edit from the other
	//	            four", "six edits apart"); apart when it bounds them from
	//	            one side only ("no two within three edits").
	//	crowd*      the crowd reading the sentence names: within one width the
	//	            most OTHER siblings any leaf has is `crowdSiblings`, and
	//	            `crowdAt` is every leaf tied at that maximum. The tie
	//	            structure is part of the shape — "Palette.Echo has two
	//	            within four" is a claim that Echo is the ONLY one, and a
	//	            list of three there would be a different set.
	//
	// # And the width that reading is taken at is a choice, not a number
	//
	// Two of the four sentences name a width: "no two of which are within
	// three edits, and Palette.Echo has two within four" is a claim at four,
	// and "six edits apart" is one at six. Those are `crowdWithin`.
	//
	// The other two are claims about EVERY width — "no width above the floor
	// keeps an answer to a pair", "cannot produce a crowd at any distance" —
	// and there is no number in them to type. The reading for those is taken
	// at the farthest sibling pair, because past it the answer cannot move:
	// every pair is already inside the threshold, so one more edit of width
	// admits nobody. `crowdAtFarthest` says so, and the width comes from
	// affordedWitnessShapeOf's own farthestD on the run.
	//
	// It used to be a literal that HAPPENED to equal the farthest pair, with
	// the reasoning in a comment beside it. Both are true of these fixtures
	// today and only one of them survives a respelling: names edited three
	// apart instead of five leave a recorded 5 reading a width the set no
	// longer reaches, where the derived one follows. The arms below also make
	// the "cannot move past it" half checkable rather than asserted in
	// prose — see the saturation arm in affordedHoldWitnessShape.
	//
	// The distances are the derivation's own metric (themeEditDistance, an
	// adjacent swap counted once), because that is what the threshold is
	// measured in and a shape stated in some other metric would be a fact
	// about no search.
	perParent           []int
	closestD, farthestD int
	// The width the crowd reading is taken at, as one of two things: a width
	// the sentence names, or the farthest pair for a sentence that names none.
	// Exactly one of them per witness — a literal beside the flag is a number
	// nothing reads, which the arm says.
	crowdWithin     int
	crowdAtFarthest bool
	crowdSiblings   int
	crowdAt         []string
}{
	0: {
		sentence: "its own crowding stopped the search",
		names: []string{
			"Weight.Aaa", "Weight.Aab", "Weight.Aac", "Weight.Aad", "Weight.Aae",
		},
		shape: "five siblings each one edit from the other four, so no width " +
			"above the floor keeps an answer to a pair and the crowding stops " +
			"the search below themeNearMissReach",
		perParent: []int{5},
		// Every pair is one apart, which is what "each one edit from the
		// other four" says and what closest == farthest is the reading of.
		closestD:  1,
		farthestD: 1,
		// "no width above the floor keeps an answer to a pair" is a claim
		// about every width, so the reading is taken where it stops moving:
		// at the farthest pair, which for a set all one edit apart is one.
		// Typed as 1 it would have been indistinguishable from a sentence
		// that names a width, and a respelling that pushed two of these
		// names three apart would leave it reading at a width the set no
		// longer has anything at.
		crowdAtFarthest: true,
		crowdSiblings:   4,
		crowdAt: []string{
			"Weight.Aaa", "Weight.Aab", "Weight.Aac", "Weight.Aad", "Weight.Aae",
		},
	},
	1: {
		sentence: "no width crowds these names at all",
		names:    []string{"Duo.Alpha", "Duo.Zulu"},
		shape: "one parent holding two leaves, which cannot produce a crowd at " +
			"any distance, so the upward search runs out of widths to try",
		perParent: []int{2},
		closestD:  5,
		farthestD: 5,
		// Read at the farthest pair, which is where the reading stops
		// moving: past it every sibling is inside the threshold, so one
		// sibling here is one sibling at every wider width. The claim that
		// no width crowds these names is perParent's — a parent of two
		// cannot make a crowd — and this is that claim at the only distance
		// where it could have failed. Derived rather than typed, because
		// "at any distance" names no distance to type.
		crowdAtFarthest: true,
		crowdSiblings:   1,
		crowdAt:         []string{"Duo.Alpha", "Duo.Zulu"},
	},
	2: {
		sentence: "the ceiling cost this set a wider threshold",
		names:    []string{"Ramp.Aaaaaa", "Ramp.Bbbbbb", "Ramp.Cccccc"},
		shape: "three siblings six edits apart, so the ceiling stops the search " +
			"and the width these names would carry is what it is costing",
		perParent: []int{3},
		closestD:  6,
		farthestD: 6,
		// Six is where all three become each other's neighbours at once, so
		// the crowd appears in one step and every leaf is tied at it. A
		// width the sentence names ("six edits apart") rather than a
		// saturating one, even though these names make the two the same
		// number: the claim being held is about six.
		crowdWithin:   6,
		crowdSiblings: 2,
		crowdAt:       []string{"Ramp.Aaaaaa", "Ramp.Bbbbbb", "Ramp.Cccccc"},
	},
	3: {
		sentence: "the ceiling and the crowding stop in the same place",
		names: []string{
			"Palette.Alpha", "Palette.Bravo", "Palette.CharlieDelta", "Palette.Echo",
		},
		shape: "four siblings no two of which are within three edits, and " +
			"Palette.Echo has two within four — so the ceiling stops the search " +
			"at the same width the crowding would have",
		perParent: []int{4},
		// Bounded from one side only: the closest pair is four apart, which
		// is what "no two within three edits" means, and the farthest is
		// eleven. These are the names that look sparse and are not one step
		// out, and closestD is the reading that says where "one step out"
		// is.
		closestD:  4,
		farthestD: 11,
		// And Echo alone. A second name in this list would be a set the
		// ceiling costs nothing for a different reason than the one the
		// sentence gives.
		crowdWithin:   4,
		crowdSiblings: 2,
		crowdAt:       []string{"Palette.Echo"},
	},
}

// affordedWitnessProbe is a path under a witness's own parent, with the
// distance from it to the nearest leaf that witness has and which leaf that
// is.
//
// # The probes were literals beside a table they were not read from
//
// TestTheNearMissThresholdMovesWithTheNamesItIsMeasuredOver builds its four
// populations out of affordedEndingWitness and then asked about them with
// "Palette.Alfa", "Palette.Zulu", "Ramp.Zzzzzz", "Duo.Mike" and "Weight.Zzz"
// typed into the test. themeNearMiss only ever compares within a parent, so a
// witness renamed out of `Palette.` would leave every probe naming a parent no
// set has — and the arms would have reported "themeNearMiss found nothing" and
// meant "this probe is not about this set". Those are different findings and
// only one of them is about the function.
//
// So the parent comes from the witness. The distance comes back with it
// because every arm has a PREMISE about it and none of them stated it: an arm
// that expects a name to be reported needs the nearest leaf inside the set's
// threshold, and an arm that expects silence needs it outside. Neither is a
// fact about the probe alone — both move when the fixture is respelled — and
// an unstated premise that has quietly stopped holding is an arm asserting
// about the wrong branch of the function.
//
// Fatal rather than Errorf on the two structural failures: a probe under the
// wrong parent, or one that names a leaf the set actually has, makes every arm
// downstream of it a reading of nothing.
func affordedWitnessProbe(t *testing.T, e int, leaf string) (path string,
	nearest int, nearestAt string) {
	t.Helper()
	w := affordedEndingWitness[e]
	first := w.names[0]
	parent := first[:strings.LastIndex(first, ".")+1]
	// Every witness sits under exactly one parent — affordedHoldWitnessShape
	// asserts it, as len(perParent) — and the probe is only well defined while
	// that holds.
	for _, name := range w.names {
		if got := name[:strings.LastIndex(name, ".")+1]; got != parent {
			t.Fatalf("witness %d holds %s under %q and %s under %q, so there is no "+
				"one parent to hang a probe off.\n\n"+
				"themeNearMiss answers a slip with that parent's own names and "+
				"nothing else, so a probe has to name the parent whose leaves the arm "+
				"is about. These fixtures are one parent each by construction; a "+
				"second one here means the set has been rebuilt and the arms below "+
				"are asking about whichever half the probe happens to land in.",
				e, first, parent, name, got)
		}
	}
	path = parent + leaf
	nearest = -1
	for _, name := range w.names {
		if name == path {
			t.Fatalf("the probe %s IS a leaf of witness %d.\n\n"+
				"themeNearMiss is about paths that name no leaf: handed one that "+
				"does, there is no near miss to report and the arm below is reading "+
				"the wrong branch of the function. Pick a spelling this set does not "+
				"have.", path, e)
		}
		d := themeEditDistance(strings.ToLower(leaf),
			strings.ToLower(name[strings.LastIndex(name, ".")+1:]))
		if nearest < 0 || d < nearest {
			nearest, nearestAt = d, name
		}
	}
	return path, nearest, nearestAt
}

// affordedWitnessShapeOf is the countable half of a witness's `shape`
// sentence, read off a list of names and nothing else.
//
// # Why this is a second implementation of a search themeLeafSetOf already has
//
// themeLeafSetOf groups by parent and runs the same crowd search, and its
// answers are on the set every arm below already holds. Reading the shape off
// THAT would make the witness's numbers a comparison of the derivation with
// itself — the objection the threshold test states one screen up and recomputes
// its own readings for. The witness's numbers are a claim about the FIXTURE:
// that these names still have the shape they were picked for. So they are held
// against a walk that shares nothing with the derivation but the metric.
//
// Returns the crowd search as a closure rather than a table because the
// distance each witness's sentence names is a different one, and a table would
// have to guess how far out to go.
func affordedWitnessShapeOf(names []string) (perParent []int,
	closestD, farthestD int, crowd func(within int) (int, []string)) {
	byParent := map[string][]string{}
	for _, path := range names {
		cut := strings.LastIndex(path, ".") + 1
		byParent[path[:cut]] = append(byParent[path[:cut]], path[cut:])
	}
	parents := make([]string, 0, len(byParent))
	for parent := range byParent {
		parents = append(parents, parent)
	}
	sort.Strings(parents)
	for _, parent := range parents {
		sort.Strings(byParent[parent])
		perParent = append(perParent, len(byParent[parent]))
	}
	// Most first, so the shape reads as "one parent of four" rather than as an
	// ordering of parent names nobody chose — and so a fixture that moved a
	// leaf between two parents of the same size does not read as a change.
	sort.Sort(sort.Reverse(sort.IntSlice(perParent)))

	// One distance function, used by all three readings below, for the reason
	// themeLeafSetOf builds its matrix once: the closest pair, the farthest
	// pair and every crowd width are three questions about one set of numbers.
	dist := func(parent string, i, j int) int {
		ns := byParent[parent]
		return themeEditDistance(strings.ToLower(ns[i]), strings.ToLower(ns[j]))
	}

	closestD, farthestD = -1, -1
	for _, parent := range parents {
		ns := byParent[parent]
		for i := range ns {
			for j := i + 1; j < len(ns); j++ {
				d := dist(parent, i, j)
				if closestD < 0 || d < closestD {
					closestD = d
				}
				if d > farthestD {
					farthestD = d
				}
			}
		}
	}

	// The most OTHER siblings any leaf has within a distance, and EVERY leaf
	// tied at that maximum. themeLeafSetOf keeps only the first, because a
	// message wants one name; the tie structure is part of a shape, so this
	// keeps them all. Sorted, so the answer does not depend on map iteration.
	crowd = func(within int) (int, []string) {
		most, at := 0, []string{}
		for _, parent := range parents {
			ns := byParent[parent]
			for i := range ns {
				n := 0
				for j := range ns {
					if i != j && dist(parent, i, j) <= within {
						n++
					}
				}
				switch {
				case n > most:
					most, at = n, []string{parent + ns[i]}
				case n == most && n > 0:
					at = append(at, parent+ns[i])
				}
			}
		}
		sort.Strings(at)
		return most, at
	}
	return perParent, closestD, farthestD, crowd
}

// affordedHoldWitnessShape asks whether one witness's names still have the
// shape the sentence beside them describes.
//
// Split out of affordedHoldEndingWitnesses so the two findings stay two: this
// one says the FIXTURE moved, and the branch arms next door say the
// CLASSIFICATION moved. A set edited into a different shape usually trips both,
// and a reader who is told only "it reaches branch 2 rather than 3" has to go
// and work out which of the two happened.
func affordedHoldWitnessShape(t *testing.T, e int) {
	t.Helper()
	w := affordedEndingWitness[e]
	perParent, closestD, farthestD, crowd := affordedWitnessShapeOf(w.names)

	if !slices.Equal(perParent, w.perParent) {
		t.Errorf("%s sits %v leaves to a parent and this witness says %v.\n\n"+
			"That is the shape it was chosen for — %s — and the branch of the "+
			"classification it reaches is a consequence of it. A set that reaches "+
			"the right branch with the wrong number of leaves under a parent is "+
			"reaching it for a different reason than the one written here, which "+
			"is the state the ending arms alone cannot see: they ask which branch "+
			"and this asks why.",
			affordedNameList(w.names), perParent, w.perParent, w.shape)
	}
	if closestD != w.closestD || farthestD != w.farthestD {
		t.Errorf("%s has its closest sibling pair %d edits apart and its farthest "+
			"%d; this witness says %d and %d.\n\n"+
			"Those two bound the shape from each side: equal means every pair is "+
			"the same distance apart (\"each one edit from the other four\", \"six "+
			"edits apart\") and apart means the sentence bounds them from one side "+
			"only. A fixture whose names were respelled keeps its leaf count and "+
			"loses this, and the ending it reaches can survive both.\n\n"+
			"The shape it was picked for: %s",
			affordedNameList(w.names), closestD, farthestD,
			w.closestD, w.farthestD, w.shape)
	}
	// The width, which is either the one the sentence names or the one the
	// names themselves put the last pair at. See the witness struct: a
	// sentence about EVERY width has no number in it to type, and a literal
	// that happens to equal farthestD is a reading that stops following the
	// fixture the moment somebody respells it.
	within, widthSaid := w.crowdWithin, fmt.Sprintf("within %d edits", w.crowdWithin)
	if w.crowdAtFarthest {
		within = farthestD
		widthSaid = fmt.Sprintf("at this set's farthest sibling pair, %d edits",
			farthestD)
		if w.crowdWithin != 0 {
			t.Errorf("witness %s carries crowdAtFarthest and a crowdWithin of %d.\n\n"+
				"The two are exclusive: the reading is taken at the farthest pair "+
				"(%d here, derived from these names) and the literal beside it is "+
				"read by nothing. If the sentence does name a width, drop the flag "+
				"and the width is asserted; if it does not, drop the number and "+
				"nothing can go stale.",
				affordedNameList(w.names), w.crowdWithin, farthestD)
		}
	}
	siblings, at := crowd(within)
	if siblings != w.crowdSiblings || !slices.Equal(at, w.crowdAt) {
		t.Errorf("%s the most siblings any leaf of %s has is %d, at "+
			"%v; this witness says %d at %v.\n\n"+
			"A crowd reading is what every branch of the classification turns on, "+
			"and WHICH leaves are tied at it is part of the shape rather than "+
			"decoration: \"Palette.Echo has two within four\" is a claim that Echo is "+
			"the only one, and three names there would be a set the ceiling costs "+
			"nothing for a different reason than the one written down.\n\n"+
			"The shape it was picked for: %s",
			widthSaid, affordedNameList(w.names), siblings, at,
			w.crowdSiblings, w.crowdAt, w.shape)
	}

	// # And the argument for reading at the farthest pair, as an arm
	//
	// "past it the answer cannot move" is the whole reason a sentence about
	// every width can be checked at one width, and it was a comment. Here it
	// is the assertion: the same reading taken at a width no pair of these
	// names can exceed — an edit distance is at most the longer of the two
	// spellings — has to come back identical.
	//
	// It cannot fail while the crowd search is monotone in `within`, which is
	// the point: it is the search's own property that makes the one reading
	// stand for all of them, and a search that acquired a ceiling of its own
	// would break the inference silently everywhere else.
	if w.crowdAtFarthest {
		beyond := within
		for _, name := range w.names {
			beyond += len(name)
		}
		wider, widerAt := crowd(beyond)
		if wider != siblings || !slices.Equal(widerAt, at) {
			t.Errorf("%s's crowd reading is %d siblings at %v within %d edits and "+
				"%d at %v within %d.\n\n"+
				"This witness is read at its farthest sibling pair because its "+
				"sentence is about every width — %s — and that only stands while "+
				"the reading cannot move past that pair. Every pair is inside the "+
				"threshold at %d edits and %d is wider than any two of these names "+
				"can be apart, so the two readings are over the same comparisons "+
				"and a difference means the crowd search is not monotone in its "+
				"width: something in it now REFUSES at a distance it used to "+
				"admit, and every sentence in this file that reasons \"past here "+
				"nothing changes\" is reasoning about a search that no longer works "+
				"that way.",
				affordedNameList(w.names), siblings, at, within, wider, widerAt,
				beyond, w.shape, within, beyond)
		}
	}

	// And the one join between the two walks: the derivation reads the same
	// names this shape was taken over. Everything above is a second
	// implementation deliberately kept apart from themeLeafSetOf, and that
	// separation is worth nothing if the two are being handed different lists.
	if set := themeLeafSetOf(w.names); set.closestD != closestD {
		t.Errorf("themeLeafSetOf puts %s's closest sibling pair %d apart (%s) and "+
			"the walk beside it makes them %d apart.\n\n"+
			"These are two implementations of one metric over one list of names, "+
			"kept apart on purpose so the witness's shape is a claim about the "+
			"fixture rather than about the derivation. Two answers means the metric "+
			"itself has moved — themeEditDistance is what both of them call — or "+
			"one of the two groupings by parent is wrong.",
			affordedNameList(w.names), set.closestD, set.closest, closestD)
	}
}

// affordedHoldEndingWitnesses asks each of those four populations which ending
// it reaches, by index and by sentence.
//
// Both, because they are two different findings. The index says the switch's
// branches still stand where the array's entries are; the sentence says the
// array has not been reordered underneath them. Neither implies the other:
// affordedEndingOf is affordedEndingNames[affordedEndingAt(set)], so a
// reordering that moves the array alone leaves every index right and every
// sentence wrong, and an edit to the switch's literals leaves them both
// wrong in the other direction.
func affordedHoldEndingWitnesses(t *testing.T) {
	t.Helper()
	for e, w := range affordedEndingWitness {
		// The shape first, because it is the diagnostic for the two arms
		// below: a set that reaches the wrong branch has either been edited
		// or the classification has moved underneath it, and this is the
		// half that says which. See affordedHoldWitnessShape.
		affordedHoldWitnessShape(t, e)

		set := themeLeafSetOf(w.names)
		at := affordedEndingAt(set)
		if at != e {
			t.Errorf("%s is %s, and affordedEndingAt puts it at %d rather than "+
				"%d.\n\n"+
				"That set was chosen to reach exactly one branch of the "+
				"classification, and the branch it reaches is what every census, "+
				"band and record in this file is a partition by. Either the switch's "+
				"arms have moved — the ceiling flag, the open flag and the "+
				"afforded/edits comparison are read in an order and the order is the "+
				"partition — or these names no longer have the shape they were "+
				"picked for, and those are different repairs. It measures "+
				"threshold %d, afforded %d, capped=%v, open=%v.",
				affordedNameList(w.names), w.shape, at, e,
				set.edits, set.afforded, set.cappedByReach, set.affordedOpen)
			continue
		}
		if got := affordedEndingOf(set); got != w.sentence {
			t.Errorf("%s reaches branch %d of the classification and "+
				"affordedEndingOf calls it %q, where this witness says %q.\n\n"+
				"The switch returns an index and affordedEndingNames turns it into a "+
				"sentence by POSITION, so those two are held together by nothing but "+
				"the order of a literal — and a reordering re-labels every census, "+
				"band and record in this file at once. The census re-walked from "+
				"affordedMeasuredOn would fail too, and it is the arm a person "+
				"silences by pasting a fresh record; this one is not printed by the "+
				"paste and does not move with it.\n\n"+
				"If the wording of an ending was what changed, change it here as "+
				"well and re-take both records — affordedMeasuredOn is keyed by the "+
				"sentence. If the ORDER changed, put it back: the index is what the "+
				"walks carry.\n\nThe list reads %v.",
				affordedNameList(w.names), e, got, w.sentence,
				affordedEndingNames)
		}
	}
}

// affordedCensusOf is the whole census — the walk, the derivation and the
// classification — over a given list of distinct leaf names.
//
// The direct walk, which is what the corrected census is held against. It
// builds every set and classifies it, and nothing here is memoised: a
// shortcut in the reading that holds the shortcut would hold nothing.
func affordedCensusOf(names []string, window int) affordedCensus {
	var census affordedCensus
	affordedEachSet(names, window, func(set []string) {
		census[affordedEndingAt(themeLeafSetOf(set))]++
	})
	return census
}

// affordedResidualOf is how far one ending sits from what a merely-moved
// population predicts for it, as a fraction of that prediction.
//
// The prediction is the recorded count times the scale (see
// affordedPrediction), and the residual is what is LEFT after the scale has
// been taken off: zero when the only thing that happened was the population
// changing size, and large when the sets were sorted somewhere else. One
// definition, used by the band below and by the log line, because a band and
// the number it brackets computed by two expressions is a comparison between
// two different readings.
//
// False when there is nothing to divide by: an ending with no recorded count
// has no prediction, and a residual against zero is not a large number, it is
// no number.
// The prediction it divided by comes back with it, so a message that names the
// number and an arm that compares against it cannot be reading two evaluations
// of one product. That was the remaining half of the FMA hazard: the residual
// had one definition and the prediction beside it in the sentence did not.
func affordedResidualOf(recorded, got int, scale float64) (residual, predicted float64,
	ok bool) {
	predicted = affordedPredictedFrom(recorded, scale)
	if predicted == 0 {
		return 0, 0, false
	}
	return math.Abs(float64(got)-predicted) / predicted, predicted, true
}

// affordedBand is how far one ending's count moves, after the scale is taken
// off, when core.Theme gains or loses a single leaf.
//
// Two numbers rather than one because the walk is not symmetric: the
// populations either side of a one-leaf step are different sizes, so the same
// step read forwards and backwards divides by different predictions. Both are
// kept because both are the band for a real edit — a field removed from the
// struct and a field added to it.
type affordedBand struct {
	losing  float64
	gaining float64
	// The leaf whose absence produced each number. Not recorded (see
	// affordedBandMeasuredOn) and not asserted; carried so a failure can say
	// WHICH single leaf moves an ending that much, because "a one-leaf change
	// moves this ending by a third" is a fact a reader can only check by being
	// told which leaf.
	losingAt  string
	gainingAt string
}

// affordedOneLeafBand is that band, per ending, measured over every population
// this one is a single leaf away from.
//
// # Why this is measured and not stated
//
// The residual reading below is the only thing in this file that can see a
// re-sort while every floor is still cleared, and it needs a band: how far an
// ending may sit from its own prediction before the run is no longer a walk
// standing still. Until this was measured that band was affordedResidualQuiet,
// a tenth, whose note argued that a leaf added mid-name shifts which windows
// crowd "by a percent or two" and that a tenth was well above it. Neither
// number was in hand and the argument was wrong in the direction that matters:
//
//	no width crowds these names at all             0.02%   (473 sets)
//	the ceiling cost this set a wider threshold      2.9%   (377 sets)
//	the ceiling and the crowding stop in the same   27.6%   ( 53 sets)
//	its own crowding stopped the search             33.3%   ( 27 sets)
//
// A tenth is well above the drift for the two large endings and well UNDER it
// for the two small ones, so on an honest one-leaf edit the log line would
// have called a standing-still walk re-sorted — the reading firing at exactly
// the moment somebody is editing the struct, which is the moment it is read.
//
// The shape is not a surprise once it is in front of you: a one-leaf change
// moves a handful of SETS between endings, and a handful is a third of 27 and
// nothing at all of 473. That is the same complaint the per-ending floors were
// introduced for, arriving one level up for the third time in this file: one
// number is not a bracket for four populations this far apart.
//
// # Where the populations come from
//
// Not from a synthetic name. Every population here is core.Theme's own leaves
// with one of them dropped, and the pair is read in both directions:
//
//	losing    the record is this run and the run is the smaller population —
//	          the struct having lost that leaf
//	gaining   the record is the smaller population and the run is this one —
//	          the struct having gained it back
//
// So the leaf that moves is every one of the struct's in turn rather than one
// somebody picked, and both directions are real edits with real names on both
// sides. This is also why the gaining half can be an assertion and not just a
// reading: see the arm in the test below. When this run has one leaf MORE than
// affordedMeasuredOn, the record's population is exactly one of the smaller
// populations walked here — the run's names minus the added leaf — so the
// record-to-run residual is a member of this family and cannot exceed the band
// unless themeLeafSetOf sorted the walk differently than it did when the
// record was taken.
func affordedOneLeafBand(names []string) (map[string]affordedBand, string) {
	return affordedKLeafBand(names, 1)
}

// affordedKLeafBand is that band for a step of k leaves rather than one:
// every population these names are exactly k away from, enumerated.
//
// # Why k at all, and why it is not the one-leaf band times k
//
// The one-leaf arms are assertions because the family is enumerable — eighty
// populations, every one of them walked, no case anybody chose. Past one leaf
// the log line said an ending outside the band is "the absence of a finding
// rather than one", and the obvious next move was to check whether the drift
// grows about linearly and use band × k. It does not:
//
//	                                                one leaf   two leaves  ×
//	its own crowding stopped the search    losing     24.96%      46.77%  1.87
//	                                      gaining     33.26%      87.88%  2.64
//	no width crowds these names at all     losing      0.02%       0.04%  2.03
//	                                      gaining      0.02%       0.04%  2.03
//	the ceiling cost this set a wider ...  losing      2.92%       5.10%  1.75
//	                                      gaining      2.84%       4.85%  1.71
//	the ceiling and the crowding stop ...  losing     21.63%      38.02%  1.76
//	                                      gaining     27.60%      61.35%  2.22
//
// The ratios run from 1.71× to 2.64× and four of the eight are ABOVE two, so
// `band × k` is not a bound: twice the one-leaf gaining band for the crowding
// ending is 66.5% and two leaves are worth 87.9%, which would have called an
// honest two-field edit a re-sort — the same failure the stated tenth had,
// arriving in the fix for it. (The percentages are the recorded four decimal
// places and the ratios are taken on the two MEASUREMENTS, which is why 0.02%
// over 0.02% comes to 2.03 rather than 1 — two numbers that differ by more
// than the last place they are shown at. The ratio was printed as 2.24 while
// the log line divided the measurement by the ROUNDED record, which is a
// rounding wearing a measurement's clothes and is the reason both ends of the
// division are now live numbers.)
//
// It is not linear in either direction and the two directions are not even
// alike: gaining divides by the SMALLER population's count, so a handful of
// sets is a larger share of 27 than of 29 and the same step read backwards is
// worth more. Losing is under two on the three endings with sets to spare and
// over it on the one whose drift is a rounding — which is "one number is not a
// bracket for four populations this far apart" holding at the next step out.
//
// So the band for a step is measured for that step, and the ratios above go
// on saying so at each one: three leaves gain 2.55× what two do, which is the
// same shape again a step out. This is what makes it possible: the family is
// enumerable at every k this file takes, so the arms are assertions on the
// same footing as the one-leaf ones and nothing has been scaled.
//
// # And where it stops, which has moved twice and is now a judgement
//
// C(80, k) populations: 80, 3160, 82160, 1.6M, 24M. Measured on
// verifyTimingsTakenOn's machine, with the family split across its cores, at
// each of the two things that have
// happened to the cost — the census of a drop taken as a correction of the
// whole population's rather than as a fresh walk (affordedWhole), and the
// windows those corrections put back classified once for the walk rather than
// once per population (affordedWhole.memoise):
//
//	          direct walk    corrected    and memoised
//	k = 1            80ms          8ms             3ms
//	k = 2           5.3s         463ms            38ms
//	k = 3        ~2 hours        12.4s           112ms
//	k = 4                    ~4 minutes         685ms
//	k = 5                                         8.4s
//
// The first was worth eleven times the direct walk and is what took the
// `-short` gate out of the test. The second is worth another twelve, and it is
// what moved the wall: twelve seconds was not a reading that belonged on every
// green run and 112ms is, so k = 3 has a band and the arms that read it (see
// affordedBandSteps, which is where the judgement about k = 4 is written down
// with its number).
//
// What has NOT moved is the composition. affordedChainBoundOf is the
// construction that was supposed to reach past whatever the enumerable limit
// was, and it is not a bound; the sound version is one and needs a census of
// every population of size n−k to build, which is the family the direct
// measurement walks. The direct measurement is now the cheaper of the two at
// every k, so there is no longer a step at which the chain would even be the
// affordable answer.
//
// # Walked in parallel, and deterministically
//
// The family is split across runtime.GOMAXPROCS and the reduction is by
// (value, then lowest combination index), which is what the sequential scan
// produced — the first combination to reach the maximum — so the recorded
// `losingAt` does not depend on how the work was split.
// # And measured once per (names, k) in a run, not once per caller
//
// The walk is a pure function of a list of names and a step, and a run asks
// for the same (names, k) more than once. The one-leaf band over the record's
// names was taken three times on every green run: once by the steps loop,
// once by affordedTwoStepBands for the composition's b₁, and a third time by
// affordedChainBoundOf's first step, which is `names[0:]` and therefore the
// same list. About 15ms — a tenth of what the three-leaf step costs, which is
// why nobody noticed while the test was 2.9 seconds and does notice now that
// it is under one.
//
// So the answer is memoised on the argument list rather than the walk being
// changed. This is the same trade affordedWhole.memoise makes one level down,
// with none of its difficulty: the key is the whole of the input, the value
// is the whole of the output, and there is no enumeration that has to stay
// covering. Guarded because the walk itself is parallel and a caller may sit
// inside another goroutine; keyed on the names joined by a byte no leaf path
// can hold, so two different lists cannot key the same entry.
//
// The maps are copied out. Four entries is nothing to copy, and handing the
// same map to two callers would make a caller that wrote into it edit an
// answer somebody else is about to read — a bug that would look exactly like
// a re-sort.
//
// Counted, and the counts are in the affordance test's log line. A memo that
// stops being hit is a caller that has quietly started asking for a different
// list — `names[1:]` where it used to ask for `names`, or this run's own
// population having moved away from the record's — and that is a change in
// what is being walked rather than in how fast. The numbers say which:
// `walks` is populations actually censused and `reused` is askings that found
// one already there.
//
// # The counters are the PROCESS's and the sentence reading them says "this
// # run", so the reader takes a baseline
//
// These two accumulate for the life of the test binary, and the one place
// that prints them is inside a single test. The numbers happened to be that
// test's own, because it is the only caller of the band walks in the package —
// which is a fact nobody asserted and which the next test to want a band would
// quietly end, leaving a sentence about "this run" reporting a total that
// includes somebody else's walks. Rather than assert exclusivity, which would
// be a rule about what may be written next door, the reader snapshots the
// counters before it starts and reports the difference: see
// affordedBandMemoRead and affordedBandMemoSince. A baseline that is not zero
// is then a finding the log line states rather than an error, because a second
// caller is a perfectly good thing to be.
//
// # And the baseline says THAT there is another caller, not which
//
// The difference from a baseline is this test's own share, and a non-zero
// baseline is a sentence saying somebody else walked a band first. Somebody
// else is where the reader has to stop: the note could say two walks happened
// before this test started and not what asked for them, in a package where the
// interesting version of that finding is "the composition test is now taking
// its own census" and the uninteresting one is "a fixture warmed the memo".
//
// So the counters are also kept per caller. The key is the outermost Test
// function on the stack at the asking (see affordedBandCaller), which is the
// unit a reader can go and look at — the helper it came through is this file's
// own plumbing and would only ever name affordedTwoStepBands or
// affordedChainBoundOf. One stack walk per asking, on a path that either walks
// 80 names for 13ms or copies a four-entry map; the walk is 32 frames of
// runtime.Callers and does not show up against either.
var affordedBandMemo = struct {
	sync.Mutex
	at map[string]struct {
		bands map[string]affordedBand
		off   string
	}
	walks, reused int
	// The same two, split by who asked. Totals are kept alongside rather than
	// summed out of this map: they are what the memo is FOR, and a sum over a
	// map is a second way to get a number that has to agree with the first.
	by map[string]affordedBandCallerCount
}{at: map[string]struct {
	bands map[string]affordedBand
	off   string
}{}, by: map[string]affordedBandCallerCount{}}

// affordedBandCallerCount is one caller's share of the memo.
type affordedBandCallerCount struct{ walks, reused int }

// affordedBandCaller is the name a band walk is attributed to: the outermost
// Test function on the stack.
//
// Outermost rather than immediate, because the immediate caller is always one
// of this file's own helpers — affordedOneLeafBand, affordedTwoStepBands,
// affordedChainBoundOf — and "affordedChainBoundOf asked for a band" is not a
// fact anybody needs. Which TEST asked is: the whole point of the tally is to
// tell a reader that the band walks have a second reader and to name it.
//
// The fallback is the immediate caller's file and line, for an asking with no
// Test frame above it at all — from a goroutine started by a test, or from
// TestMain. That is a worse name and it is a name; an asking counted under ""
// would be one the tally cannot describe.
//
// # And "the outermost Test frame" is a convention, so it is asserted
//
// Every caller in this file today is a test function calling a helper on its
// own goroutine, which is the one shape this walk reads correctly. A band
// asked for inside t.Run gets the fallback — Go runs a subtest closure on a
// new goroutine, so the parent's frame is not on the stack and the closure's
// own name is `TestFoo.func1`, whose last segment is `func1`. So does a band
// asked for from a goroutine a test started.
//
// affordedHoldBandAttribution holds this against t.Name(), which is the same
// attribution from the side that cannot be wrong about it, and against the
// memo's own totals. Without it the tally is a heuristic producing a sentence
// that reads identically whether the heuristic worked or not.
func affordedBandCaller() string {
	// 2 skips runtime.Callers and this function, so the first frame is
	// whatever asked for the band.
	pcs := make([]uintptr, 32)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	outermost, fallback := "", ""
	for {
		f, more := frames.Next()
		short := f.Function[strings.LastIndex(f.Function, ".")+1:]
		if fallback == "" {
			fallback = fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line)
		}
		// Frames arrive innermost first, so the last Test seen is the one
		// furthest out — which is the test itself rather than a subtest
		// closure or a helper that happens to start with Test.
		if strings.HasPrefix(short, "Test") {
			outermost = short
		}
		if !more {
			break
		}
	}
	if outermost != "" {
		return outermost
	}
	return fallback
}

// affordedCallSite is the file:line of whoever called it, in the spelling
// affordedBandCaller's fallback uses.
//
// The other end of that fallback, and it exists so the two can be taken on ONE
// SOURCE LINE — `got, want := affordedBandCaller(), affordedCallSite()`. A
// runtime.Caller(1) written out at the asking would be a second line and the
// answers would then differ by one, so the assertion would either be
// approximate or would break the day somebody inserted a blank line between
// them.
func affordedCallSite() string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// The three stacks affordedBandCaller's doc comment describes, read.
//
// # A branch a green run never takes
//
// affordedHoldBandAttribution asserts that every key in the tally is a test
// name, which is the right thing to assert and which makes the file:line
// fallback the branch no passing run has ever entered. It is three lines, and
// it is the branch that fires on the day the convention breaks — so it is the
// one that will be read under pressure, by somebody looking at an unfamiliar
// key in a failure message, having never once been run.
//
// It is also the whole content of a claim made in prose next door. The doc
// comment says a band asked for inside t.Run or from a goroutine a test
// started lands on the fallback, and the tally arm says such a key would be
// reported — both of them describing a stack neither of them produces.
//
//	affordedBandCaller() called from        frames above it        answers
//	  the test body                         TestFoo                TestFoo
//	  a goroutine the test started          TestFoo.funcN, goexit  file:line
//	  a t.Run closure                       TestFoo.funcN, tRunner file:line
//
// Nothing here touches the memo: affordedBandCaller is a stack walk and does
// not count, so the tally this reads about is not moved by reading it.
func TestTheBandCallerFallsBackToALineWhenNoTestIsAboveIt(t *testing.T) {
	// The shape every caller in this file has, which is the one the tally is
	// built on. Asserted here as well as through the memo because this test is
	// where the three stacks are compared, and "the fallback fired" only means
	// something beside a case where it did not.
	if who, want := affordedBandCaller(), t.Name(); who != want {
		t.Errorf("called straight from a test body, affordedBandCaller answers "+
			"%q and the test is %q.\n\n"+
			"This is the ordinary stack — a test calling a helper on its own "+
			"goroutine — and it is the one every band walk in this file is asked "+
			"for on. If it does not come back as the test's own name, the tally "+
			"affordedHoldBandAttribution holds against t.Name() is keyed on "+
			"something else entirely.", who, want)
	}

	// A goroutine the test started. tRunner's frame is not on it and the
	// closure's own name is TestFoo.funcN, whose last segment is funcN — so
	// no frame in the top 32 starts with "Test" and the fallback is the whole
	// answer.
	type asking struct{ got, want string }
	fromGoroutine := make(chan asking, 1)
	go func() {
		got, want := affordedBandCaller(), affordedCallSite()
		fromGoroutine <- asking{got, want}
	}()
	a := <-fromGoroutine

	// And a subtest closure, which Go also runs on a new goroutine. The same
	// stack for the same reason, and it is listed separately because it is the
	// one a reader is most likely to write by accident: t.Run looks like a
	// call, not like a goroutine.
	var b asking
	t.Run("in a subtest closure", func(t *testing.T) {
		got, want := affordedBandCaller(), affordedCallSite()
		b = asking{got, want}
	})

	for _, c := range []struct {
		where string
		asking
	}{
		{"a goroutine this test started", a},
		{"a t.Run closure", b},
	} {
		if c.got != c.want {
			t.Errorf("asked from %s, affordedBandCaller answers %q; the frame it "+
				"was called on is %q.\n\n"+
				"Both are the same call site — they are taken on one source line "+
				"— so this is the walk having found something it took for the "+
				"asking. An answer starting with \"Test\" is a Test frame having "+
				"become visible from a goroutine, which would make the fallback "+
				"dead code and the tally's keys mean something new; any other "+
				"difference is the fallback naming a frame that is not the "+
				"caller's.", c.where, c.got, c.want)
			continue
		}
		// And that it is the fallback and not a name, which is the property
		// affordedHoldBandAttribution's last arm reads: a key that does not
		// start with "Test" is what that arm reports, and this is where such a
		// key comes from.
		if strings.HasPrefix(c.got, "Test") {
			t.Errorf("asked from %s, affordedBandCaller answers %q, which starts "+
				"with \"Test\".\n\n"+
				"It is the file:line fallback — it matches this call site exactly "+
				"— so a test file whose name begins with Test would make the two "+
				"branches indistinguishable to affordedHoldBandAttribution's "+
				"notTests arm, which sorts keys by that prefix alone.",
				c.where, c.got)
		}
		file, line, found := strings.Cut(c.got, ":")
		if !found || file != "themenearmiss_test.go" {
			t.Errorf("asked from %s, the fallback answers %q, which is not "+
				"file:line in this file.\n\n"+
				"It is built from filepath.Base of the frame's file and the "+
				"frame's line, and a reader who meets it in a tally is meant to be "+
				"able to open it. A bare line number, an absolute path, or a name "+
				"with no colon in it is not that.", c.where, c.got)
			continue
		}
		if n, err := strconv.Atoi(line); err != nil || n <= 0 {
			t.Errorf("asked from %s, the fallback answers %q, whose line number "+
				"is %q.\n\nA frame with no line is a stack the walk read past "+
				"the end of.", c.where, c.got, line)
		}
	}

	if t.Failed() {
		return
	}
	t.Logf("affordedBandCaller answers %q from this test's own body and %q from "+
		"a goroutine it started (%q from a t.Run closure, for the same reason: "+
		"Go runs a subtest on a new goroutine, so no Test frame is above the "+
		"asking). The second is the file:line fallback, which is the branch a "+
		"green run never takes — affordedHoldBandAttribution asserts every key "+
		"in the tally IS a test name — and which is the one a reader meets on "+
		"the day the convention breaks.", t.Name(), a.got, b.got)
}

// affordedBandKey is the identity of a walk: its names and its step.
//
// 0x00 as the separator because a leaf path is a Go identifier and dots, so
// no name can hold one — which is what stops {"a\x00b"} and {"a","b"} from
// being one key.
//
// # And what a run would and would not notice about this key
//
// A key that drops the step conflates the one-leaf and three-leaf bands over
// the record's names, and a key that drops the names conflates the chain's
// two steps; both are caught, loudly, by the recorded bands they would move.
//
// A key of (step, len(names)) is NOT caught, and the reason is worth writing
// down: the three populations a green run censuses here are 80 names at k=1,
// 80 at k=3 and 79 at k=1, and no two of them share a size. It would go wrong
// on the first run where a leaf was RENAMED — this run's names and the
// record's would then be two different lists of eighty, and the second asking
// would get the first's answer. That is the same premise-by-count the exact
// census arm carried before it started reading the record's own strings, and
// it is why the key is the whole list rather than a summary of it.
func affordedBandKey(names []string, k int) string {
	return fmt.Sprintf("%d\x00%s", k, strings.Join(names, "\x00"))
}

// affordedBandCopy is a band map handed to a caller, so the memo's own is
// never the one anybody holds.
func affordedBandCopy(bands map[string]affordedBand) map[string]affordedBand {
	out := make(map[string]affordedBand, len(bands))
	for ending, b := range bands {
		out[ending] = b
	}
	return out
}

// affordedBandMemoRead is the two counters, taken together under the lock.
//
// Together, because they are read into one sentence and a walk in progress
// between two separate reads would make that sentence describe a state the
// memo was never in. Cheap enough to be unremarkable — the band walks hold
// this lock for the length of a map lookup.
func affordedBandMemoRead() (walks, reused int) {
	affordedBandMemo.Lock()
	defer affordedBandMemo.Unlock()
	return affordedBandMemo.walks, affordedBandMemo.reused
}

// affordedBandMemoCount adds one asking to a caller's tally.
//
// Called with the memo already locked, by the two places that move the totals,
// so a caller's share and the total it is part of cannot disagree — the
// alternative is a second lock acquisition between the two increments and a
// window in which they do.
func affordedBandMemoCount(who string, walked bool) {
	had := affordedBandMemo.by[who]
	if walked {
		had.walks++
	} else {
		had.reused++
	}
	affordedBandMemo.by[who] = had
}

// affordedBandMemoCallers is the per-caller tally as a sentence, most work
// first, or empty when nothing has asked for a band at all.
//
// Sorted by walks and then by name so the line does not depend on map
// iteration, and so the caller that actually cost something leads it.
func affordedBandMemoCallers() string {
	affordedBandMemo.Lock()
	who := make([]string, 0, len(affordedBandMemo.by))
	tally := make(map[string]affordedBandCallerCount, len(affordedBandMemo.by))
	for name, count := range affordedBandMemo.by {
		who = append(who, name)
		tally[name] = count
	}
	affordedBandMemo.Unlock()

	sort.Slice(who, func(i, j int) bool {
		a, b := tally[who[i]], tally[who[j]]
		if a.walks != b.walks {
			return a.walks > b.walks
		}
		if a.reused != b.reused {
			return a.reused > b.reused
		}
		return who[i] < who[j]
	})
	said := make([]string, 0, len(who))
	for _, name := range who {
		said = append(said, fmt.Sprintf("%s (%d walk(s), %d reuse(s))",
			name, tally[name].walks, tally[name].reused))
	}
	return strings.Join(said, ", ")
}

// affordedBandMemoSince is the work done since a baseline, with a sentence
// about the baseline itself.
//
// The counters are the process's (see affordedBandMemo). A caller takes a
// baseline at the top of the test that prints them and the difference is that
// test's own share; the note is what says whether there WAS anything else,
// which is the assertion the old sentence rested on without making.
//
// Go runs a package's tests one after another unless they ask otherwise, so a
// non-zero baseline is work done by a test that ran before this one — not by
// one running alongside it.
func affordedBandMemoSince(baseWalks, baseReused int) (walks, reused int,
	note string) {
	nowWalks, nowReused := affordedBandMemoRead()
	walks, reused = nowWalks-baseWalks, nowReused-baseReused
	if baseWalks == 0 && baseReused == 0 {
		return walks, reused, ""
	}
	// And WHICH other test, which is the half a baseline cannot say. See
	// affordedBandMemo: the tally is kept per caller precisely so this
	// sentence names the reader rather than reporting that one exists.
	return walks, reused, fmt.Sprintf(" — and these two are this test's own "+
		"share: %d walk(s) and %d reuse(s) were already on the memo when it "+
		"started, so the band walks have a caller that is not this test and the "+
		"process totals are %d and %d. Per caller: %s",
		baseWalks, baseReused, nowWalks, nowReused, affordedBandMemoCallers())
}

// affordedBandMemoState is everything the tally can be held to, taken in one
// acquisition of the lock.
//
// One acquisition because the four readings below are compared with each
// other: a caller's share, the totals it is part of, the same totals added up
// out of the per-caller map, and whether every key names a test. Taken
// separately they could describe four moments, and a mismatch between two of
// them would then be a walk that happened in between rather than a finding.
type affordedBandMemoState struct {
	// The caller asked about, and whether it has asked for anything at all.
	mine  affordedBandCallerCount
	known bool
	// The counters the memo keeps alongside the map — what the walks and
	// reuses at the bottom of the log line are.
	total affordedBandCallerCount
	// And those same two added up out of the per-caller map. The memo keeps
	// the totals separately on purpose (see affordedBandMemo: they are what it
	// is FOR, and a sum over a map is a second way to get a number that has to
	// agree with the first) — which makes this the arm that says the second
	// way still gets the first answer.
	summed affordedBandCallerCount
	// Keys that are not a Go test function's name. affordedBandCaller falls
	// back to file:line for an asking with no Test frame above it, and that is
	// deliberate — an asking counted under "" would be one the tally cannot
	// describe — but the fallback firing means the attribution did not work
	// for that asking, which is a thing to say out loud rather than to print
	// in a sentence about who asked.
	notTests []string
}

// affordedBandMemoStateFor is that reading, for one caller.
func affordedBandMemoStateFor(who string) affordedBandMemoState {
	affordedBandMemo.Lock()
	defer affordedBandMemo.Unlock()
	st := affordedBandMemoState{total: affordedBandCallerCount{
		walks: affordedBandMemo.walks, reused: affordedBandMemo.reused}}
	st.mine, st.known = affordedBandMemo.by[who]
	for name, count := range affordedBandMemo.by {
		st.summed.walks += count.walks
		st.summed.reused += count.reused
		if !strings.HasPrefix(name, "Test") {
			st.notTests = append(st.notTests, name)
		}
	}
	sort.Strings(st.notTests)
	return st
}

// This test's own askings are filed under this test's own name.
//
// # A heuristic with nothing holding it to its own convention
//
// affordedBandCaller walks 32 frames and attributes an asking to the OUTERMOST
// function whose name starts with "Test". That is right for every caller in
// this file today and it is a convention rather than a fact: Go runs a subtest
// closure on its own goroutine, so a band asked for inside t.Run has no Test
// frame above it at all and lands on the file:line fallback. So does a band
// asked for from a goroutine a test started, and so does one asked for from
// TestMain. Nothing said the tally's keys were test names, and the sentence
// the tally exists to produce — "the band walks have a caller that is not this
// test, and it is X" — reads exactly the same whether X is a test or a line
// number.
//
// t.Name() is the other end of that attribution and it is right here. Holding
// the memo's idea of who asked against the runtime's makes the convention a
// claim: this test asked for every band it asked for, and they are all filed
// under it.
//
// # Why the two numbers can be compared at all
//
// baseWalks and baseReused are snapshotted at the top of this test, before it
// has asked for anything, so the difference from them is this test's own
// share. affordedBandMemo.by[t.Name()] is that same share from the other
// direction — nothing else in this package can be filed under this test's
// name, because the key IS the name and Go runs it once. The two are the same
// number counted by the caller and by the callee, and they agree only while
// every asking on this test's stack was attributed to this test.
func affordedHoldBandAttribution(t *testing.T, mineWalks, mineReused int) {
	t.Helper()
	st := affordedBandMemoStateFor(t.Name())
	if !st.known {
		t.Errorf("the band memo has no entry at all for %q, and this test asked "+
			"for %d walk(s) and %d reuse(s).\n\n"+
			"affordedBandCaller attributes an asking to the outermost Test "+
			"function on the stack, and the tally it feeds is what tells a reader "+
			"the band walks have a second caller and names it. With this test "+
			"missing from it, that naming is not working for the one caller "+
			"there certainly is: the askings went to the file:line fallback, "+
			"which happens when no frame in the top 32 starts with \"Test\" — a "+
			"band asked for inside t.Run (its closure runs on its own goroutine), "+
			"from a goroutine this test started, or from a helper deeper than the "+
			"stack the walk reads.\n\nPer caller: %s",
			t.Name(), mineWalks, mineReused, affordedBandMemoCallers())
		return
	}
	if st.mine.walks != mineWalks || st.mine.reused != mineReused {
		t.Errorf("this test's own share of the band memo is %d walk(s) and %d "+
			"reuse(s) counted from the baseline it took, and %d and %d filed "+
			"under %q.\n\n"+
			"Those are one number counted twice: the baseline is snapshotted "+
			"before this test asks for anything, and nothing else in this package "+
			"can be filed under this test's name. A difference is askings that "+
			"happened on this test's watch and were attributed to somebody else "+
			"— which is affordedBandCaller reading a stack it did not expect, "+
			"most likely a band asked for from a goroutine.\n\nPer caller: %s",
			mineWalks, mineReused, st.mine.walks, st.mine.reused, t.Name(),
			affordedBandMemoCallers())
	}
	if st.summed != st.total {
		t.Errorf("the band memo's own counters are %d walk(s) and %d reuse(s), "+
			"and the per-caller tally adds up to %d and %d.\n\n"+
			"The two are kept separately on purpose — the totals are what the "+
			"memo is for, and summing a map is a second way to get a number that "+
			"has to agree with the first — and they move under one lock, in "+
			"affordedBandMemoCount, called with the memo already held by the two "+
			"places that move the totals. They can only disagree if an asking "+
			"moved a total without moving a caller's share, which is a walk "+
			"nobody can attribute sitting inside a number everybody reads.\n\n"+
			"Per caller: %s",
			st.total.walks, st.total.reused, st.summed.walks, st.summed.reused,
			affordedBandMemoCallers())
	}
	if len(st.notTests) > 0 {
		t.Errorf("the band memo has %d caller(s) that are not test names: %s.\n\n"+
			"affordedBandCaller falls back to file:line when no frame above the "+
			"asking starts with \"Test\", which is better than counting it under "+
			"\"\" and is not what the tally is for: the unit a reader can go and "+
			"look at is a test, and \"themenearmiss_test.go:4612 walked a band\" "+
			"names a line that will have moved by the time anybody reads it.\n\n"+
			"The three ways to get here are a band asked for inside t.Run, from a "+
			"goroutine a test started, or from TestMain. If the new caller is one "+
			"of those on purpose, the attribution is what needs widening — "+
			"t.Name() is available at every one of them and the stack is not.\n\n"+
			"Per caller: %s", len(st.notTests), strings.Join(st.notTests, ", "),
			affordedBandMemoCallers())
	}
}

func affordedKLeafBand(names []string, k int) (map[string]affordedBand, string) {
	key := affordedBandKey(names, k)
	// Taken outside the lock: a stack walk under a mutex the parallel band
	// walks also take is a stack walk every other goroutine waits for.
	who := affordedBandCaller()
	affordedBandMemo.Lock()
	if had, seen := affordedBandMemo.at[key]; seen {
		affordedBandMemo.reused++
		affordedBandMemoCount(who, false)
		affordedBandMemo.Unlock()
		return affordedBandCopy(had.bands), had.off
	}
	affordedBandMemo.Unlock()
	// Walked outside the lock, so two callers asking for two different bands
	// at once are not serialised behind each other. The cost of both asking
	// for the SAME one is that it is walked twice and stored twice, which is
	// the state before the memo existed rather than a wrong answer.
	bands, off := affordedKLeafBandWalk(names, k)
	affordedBandMemo.Lock()
	if _, seen := affordedBandMemo.at[key]; !seen {
		affordedBandMemo.walks++
		affordedBandMemoCount(who, true)
	}
	affordedBandMemo.at[key] = struct {
		bands map[string]affordedBand
		off   string
	}{affordedBandCopy(bands), off}
	affordedBandMemo.Unlock()
	return bands, off
}

// affordedKLeafBandWalk is the walk itself. See affordedKLeafBand, which is
// the memo in front of it.
func affordedKLeafBandWalk(names []string, k int) (map[string]affordedBand, string) {
	bands := map[string]affordedBand{}
	for _, ending := range affordedEndingNames {
		bands[ending] = affordedBand{}
	}
	if k <= 0 || k >= len(names) {
		return bands, ""
	}
	// The whole population, walked once and kept per window. Every population
	// below is this one with k names taken out, and a census of it is this
	// census corrected — see affordedWhole, which is why 3160 populations cost
	// about thirty classifications each instead of nine hundred — and the
	// windows those corrections put back are classified once for the whole
	// walk rather than once per population. See memoise.
	walked := affordedWholeOf(names, affordedWindowMax)
	walked.memoise(k)
	full := walked.census
	sets := affordedSetCount(len(names), affordedWindowMax)
	smaller := affordedSetCount(len(names)-k, affordedWindowMax)
	if sets == 0 || smaller == 0 {
		return bands, ""
	}
	// The accounting every drop census below has to come to: the whole has
	// this many windows and each smaller population has that many, so what a
	// drop takes out less what it puts in is the difference, whatever the drop
	// was. Held on every one of the populations rather than argued for.
	moved := affordedWindowCount(len(names), affordedWindowMax) -
		affordedWindowCount(len(names)-k, affordedWindowMax)
	// Every k-drop is the same size, so the two scales are constants rather
	// than a division inside the walk. Through affordedRatioOf, which is also
	// what affordedScale divides with: `down` and the scale the test reads a
	// residual against are the same number, and the k-leaf assertions are only
	// proofs while they are.
	down := affordedRatioOf(smaller, sets)
	up := affordedRatioOf(sets, smaller)

	// The combinations, enumerated up front so the work can be split by index
	// and the reduction can tie-break on it.
	combos := [][]int{}
	pick := make([]int, k)
	var build func(pos, from int)
	build = func(pos, from int) {
		if pos == k {
			combos = append(combos, append([]int(nil), pick...))
			return
		}
		for i := from; i <= len(names)-(k-pos); i++ {
			pick[pos] = i
			build(pos+1, i+1)
		}
	}
	build(0, 0)

	// A candidate maximum and the combination that reached it. The index is
	// carried so the merge below can reproduce the sequential answer.
	type best struct {
		value float64
		at    int
	}
	worst := func(a, b best) best {
		if b.value > a.value || (b.value == a.value && b.at >= 0 && a.at >= 0 &&
			b.at < a.at) {
			return b
		}
		return a
	}
	type result struct {
		// The widest residual this worker saw in each direction, per ending.
		//
		// Named for the direction and not `losing`/`gaining`, which is what
		// they were called: affordedBand's two fields are floats with those
		// names, and the field table this package's float census reads a type
		// off is keyed by the field name alone. Two structs disagreeing about
		// a name is a conflation that census can only report, so every
		// selector called `.losing` anywhere in the package read as float
		// arithmetic — including these, which are arrays. See
		// affordedFloatSource: renaming the pair here is the half of that
		// finding this file owns, and it takes the census from five conflated
		// names to three.
		widestLosing, widestGaining [len(affordedEndingNames)]best
		// The first population whose window bookkeeping did not add up, or
		// empty. Carried out of the walk rather than reported inside it.
		off string
		// And how many windows this worker had to classify itself. See
		// affordedWhole.memoise: zero is the enumeration covering, and any
		// other number is the same answers reached more slowly.
		missed int
		// Whether this worker had a share of the combinations at all, which
		// the reduction below has to tell from a worker that found nothing.
		ran bool
	}
	workers := runtime.GOMAXPROCS(0)
	if workers > len(combos) {
		workers = len(combos)
	}
	results := make([]result, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			r := result{ran: true}
			defer func() { results[w] = r }()
			for e := range affordedEndingNames {
				r.widestLosing[e] = best{at: -1}
				r.widestGaining[e] = best{at: -1}
			}
			// Strided rather than blocked, so a family whose expensive
			// populations sit together does not land on one worker.
			//
			// The working room is this goroutine's own: the drop census marks
			// which windows of the whole it has taken out and gathers the
			// leaves of the ones it puts in, and two workers sharing either
			// would be two populations written over one another.
			sc := affordedScratchFor(walked)
			defer func() { r.missed = sc.missed }()
			var census affordedCensus
			for c := w; c < len(combos); c += workers {
				drop := combos[c]
				out, in := walked.censusDropping(drop, sc, &census)
				if out-in != moved && r.off == "" {
					// Reported through the result rather than raised here:
					// this is a walk and not a test, and the caller is the
					// one that can say which band was being measured.
					r.off = fmt.Sprintf("dropping %v took out %d windows and put "+
						"in %d, a difference of %d against the %d a population of "+
						"%d has fewer than one of %d", drop, out, in, out-in, moved,
						len(names)-k, len(names))
				}
				for e := range affordedEndingNames {
					if off, _, ok := affordedResidualOf(full[e], census[e], down); ok {
						r.widestLosing[e] = worst(r.widestLosing[e], best{off, c})
					}
					if off, _, ok := affordedResidualOf(census[e], full[e], up); ok {
						r.widestGaining[e] = worst(r.widestGaining[e], best{off, c})
					}
				}
			}
		}(w)
	}
	wg.Wait()

	// The name a combination is reported under: the leaves it dropped, in the
	// order they stand in the struct.
	spell := func(c int) string {
		if c < 0 {
			return ""
		}
		dropped := make([]string, 0, k)
		for _, i := range combos[c] {
			dropped = append(dropped, names[i])
		}
		return strings.Join(dropped, ", ")
	}
	// The first bookkeeping complaint any worker had, in worker order so the
	// sentence does not depend on which goroutine got there first — and the
	// windows the memo did not hold, which is a complaint of the same kind:
	// both are the walk having quietly stopped being the walk it describes.
	off, missed := "", 0
	for _, r := range results {
		if off == "" {
			off = r.off
		}
		missed += r.missed
	}
	if off == "" && missed > 0 {
		off = fmt.Sprintf("%d of the windows a drop put back were not in the "+
			"table of every shape a drop of up to %d names can produce, so they "+
			"were classified one at a time", missed, k)
	}
	for e, ending := range affordedEndingNames {
		losing, gaining := best{at: -1}, best{at: -1}
		for _, r := range results {
			if !r.ran {
				continue // a worker with no share of the combinations
			}
			losing = worst(losing, r.widestLosing[e])
			gaining = worst(gaining, r.widestGaining[e])
		}
		bands[ending] = affordedBand{
			losing: losing.value, losingAt: spell(losing.at),
			gaining: gaining.value, gainingAt: spell(gaining.at),
		}
	}
	return bands, off
}

// affordedComposedOf is what a chain of per-step bands composes to.
//
// ∏(1+b_i) − 1, which is the identity both chain readings rest on: dropping k
// leaves is k one-leaf drops, the scales telescope, and the residuals
// multiply. One spelling because there are two callers — the chain over one
// representative population and the chain over every one of them — and a
// composition written twice is two arithmetics that can round differently
// against a band that is the maximum of a family one of them belongs to. That
// is the hazard this file was actually caught by, and the census that watches
// for it could not see either spelling until it learned to read a product of
// two variables.
func affordedComposedOf(steps ...float64) float64 {
	product := 1.0
	for _, b := range steps {
		product *= 1 + b
	}
	return product - 1
}

// affordedChainBoundOf is what the one-leaf band composes to over a step of k,
// and the reading that says a composed bound is not one.
//
// # The identity, which is exact
//
// Dropping k leaves is k successive one-leaf drops through intermediate
// populations, and the residuals compose. Write c_i for an ending's count over
// the i-th population of a chain P_0 ⊃ P_1 ⊃ … ⊃ P_k, each one leaf smaller
// than the last, and s_i for that step's scale — the set count at n−i−1 over
// the set count at n−i. The one-step residual is
//
//	1 + r_i = c_(i+1) / (c_i · s_i)
//
// and the scales telescope: s_0·s_1·…·s_(k−1) = S(n−k)/S(n), which is exactly
// the k-step scale affordedScale divides by. So
//
//	1 + r_total = ∏ (1 + r_i)
//
// with nothing approximated — the intermediate counts cancel. That much is
// algebra, and the only empirical part is whether the two spellings of the
// k-step scale agree in floating point, which the test asserts.
//
// From it, |r_total| = |∏(1+r_i) − 1| ≤ ∏(1+b_i) − 1 whenever b_i ≥ |r_i| at
// each step. That is the chain bound, and it is what this function computes.
//
// # Why it is measured here and not used
//
// The attraction was cost: the bands the product needs are one-leaf bands at
// sizes n, n−1, …, which is k×80 walks against the C(80,k) the direct
// measurement costs — 160 against 3160 at k = 2, and 240 against 82160 at
// k = 3, where the family stops being enumerable at all. Two things are wrong
// with it and both are visible at k = 2, which is the one step where the
// composed number and a measured one can be put side by side.
//
// The first is that the b_i have to bound the steps of EVERY chain, and the
// cheap construction does not. This function walks one representative chain —
// P_i is these names with the first i dropped, a rule rather than a choice —
// so b_1 is the band over one population of n−1 names, and a two-drop whose
// first leaf came from somewhere else steps out of a population that band was
// never measured over. The sound version takes b_1 as the largest one-leaf
// residual out of ANY population of n−1, which is every ordered pair of drops:
// 80 walks at n−1 plus the 3160 at n−2 that the direct measurement already
// costs. So the sound chain bound is more expensive than the thing it was
// meant to replace, and the cheap one is not a bound at all.
//
// The second is that it is loose where it does hold. A product of maxima is
// the case where every step is simultaneously at its worst, and the endings
// this file has do not do that — see the log line, which prints both numbers
// so the slack is a fact and not a claim.
//
// Both are asserted rather than argued: the composed numbers are recorded in
// affordedBandMeasuredOn.chain, re-derived here, and compared against the
// measured two-leaf band by the arm that says at least one of the eight is
// UNDER it. A run where that stops being true is a run where somebody should
// look again at whether the composition has become usable — it would be the
// only route to a band at k = 3 — and the arm says so.
func affordedChainBoundOf(names []string, k int) (map[string]affordedBand, string) {
	bound := map[string]affordedBand{}
	for _, ending := range affordedEndingNames {
		bound[ending] = affordedBand{}
	}
	if k <= 0 || k >= len(names) {
		return bound, ""
	}
	// Each step's band, kept per ending until the whole chain is walked and
	// then composed in one place. See affordedComposedOf.
	n := len(affordedEndingNames)
	losing := make(map[string][]float64, n)
	gaining := make(map[string][]float64, n)
	losingAt := make(map[string][]string, n)
	gainingAt := make(map[string][]string, n)
	off := ""
	for step := 0; step < k; step++ {
		// The representative population of size n−step: these names with the
		// first `step` dropped. A rule, so that the chain is reproducible and
		// the same on every run — and the whole of what makes this a bound
		// over one chain rather than over the family. See the note above.
		at, wrong := affordedKLeafBand(names[step:], 1)
		if wrong != "" && off == "" {
			off = fmt.Sprintf("at step %d of the chain, %s", step, wrong)
		}
		for _, ending := range affordedEndingNames {
			b := at[ending]
			losing[ending] = append(losing[ending], b.losing)
			gaining[ending] = append(gaining[ending], b.gaining)
			losingAt[ending] = append(losingAt[ending], b.losingAt)
			gainingAt[ending] = append(gainingAt[ending], b.gainingAt)
		}
	}
	for _, ending := range affordedEndingNames {
		bound[ending] = affordedBand{
			losing:    affordedComposedOf(losing[ending]...),
			gaining:   affordedComposedOf(gaining[ending]...),
			losingAt:  strings.Join(losingAt[ending], " then "),
			gainingAt: strings.Join(gainingAt[ending], " then "),
		}
	}
	return bound, off
}

// affordedTwoStepBands is one walk of every two-drop and the three bands it
// produces: the measured two-leaf step, the widest SECOND step of any chain,
// and the sound composition of that with the one-leaf band.
//
// # What affordedChainBoundOf could not say
//
// The composition is an identity: two drops are two steps and the residuals
// multiply. It is a BOUND only if each b_i bounds the step it stands for, and
// affordedChainBoundOf measures b_1 over one representative population of n−1
// — these names with the first dropped — while a two-drop can step out of any
// of the eighty. So its failure to cover the measured two-leaf band has two
// possible causes and it cannot separate them:
//
//	the representative was unlucky   another population of n−1 has a wider
//	                                 one-leaf step, and a b_1 that saw it
//	                                 would compose to something that covers
//	the product is the wrong shape   even the largest step at each size
//	                                 composes to less than the step taken
//	                                 whole, because the two steps of a real
//	                                 two-drop are not independent
//
// This is the reading that separates them. b_1 here is the largest one-leaf
// residual out of ANY population of n−1: every ordered pair of drops, which is
// each of the 3160 two-drops reached from each of the two populations it can
// be reached from. If the composed number covers, the representative was
// unlucky and the construction is worth its cost; if it still does not, the
// product is the wrong shape and no cheaper version of the chain will do.
//
// It covers. So the answer is the first, and what the sound bound is FOR is no
// longer the finding — it is a check: the bound dominates every two-drop
// residual by construction, so it dominates the measured two-leaf band by
// construction, and a run where it does not is one of the two walks being
// wrong.
//
// # And the two walks are now one, which is why the check is affordable
//
// The sound bound and the measured two-leaf band were two functions, and each
// censused all 3160 two-drops of the record's names: the band read each
// population against the WHOLE and the bound read it against the two
// populations of n−1 it can be reached from. Same populations, same censuses,
// twice — about 450ms of a test that had just been brought to 1.8s, spent
// re-deriving a census that had already been taken in the same run.
//
// So the census is taken once and read three ways. The two readings still
// share nothing but themeLeafSetOf in the sense that matters — the residual a
// population is judged by is against a different denominator on each side, and
// the domination arm compares the two — but the population itself is walked
// once, because walking it twice was never part of what the check was holding.
//
// The tie-break is the pair's own index, which is the order affordedKLeafBand
// enumerates its combinations in for k = 2 (i < j, lexicographically), so the
// measured band this produces names the same widest drop that one does.
// # And why there is no three-step version, though affordedBandSteps runs to 3
//
// Declined, and written down rather than left as a gap somebody re-derives.
// Bounding the LAST step over every chain needs a census of every population
// of size n−k — which is the family the direct measurement already walks, and
// the direct measurement is the cheaper of the two at every k this file takes.
// The composition at k = 3 would cost more than the number it bounds and
// produce a looser one: a slower route to a weaker answer, not an
// approximation that buys anything. This is kept at k = 2 for the SHAPE above
// — the reading that separated the two causes of affordedChainBoundOf failing
// to cover — and not as a step towards three.
//
// See ai_docs/plans/non_goals.md, which carries the argument in full and what
// would have to change for it to be reconsidered.
func affordedTwoStepBands(names []string) (
	measured, second, bound map[string]affordedBand, off string) {
	measured = map[string]affordedBand{}
	second = map[string]affordedBand{}
	bound = map[string]affordedBand{}
	for _, ending := range affordedEndingNames {
		measured[ending] = affordedBand{}
		second[ending] = affordedBand{}
		bound[ending] = affordedBand{}
	}
	n := len(names)
	if n < 3 {
		return measured, second, bound, ""
	}
	first, off := affordedKLeafBand(names, 1)
	walked := affordedWholeOf(names, affordedWindowMax)
	walked.memoise(2)
	full := walked.census
	// Two pairs of scales, because two different steps are being read off one
	// census. The two-leaf pair takes a population of n−2 against the whole;
	// the second-step pair takes it against a population of n−1, which is what
	// the composition's b_2 is a band over. Both through affordedRatioOf, so
	// every residual in this file divides with one division.
	whole := affordedSetCount(n, affordedWindowMax)
	one := affordedSetCount(n-1, affordedWindowMax)
	two := affordedSetCount(n-2, affordedWindowMax)
	if whole == 0 || one == 0 || two == 0 {
		return measured, second, bound, off
	}
	down, up := affordedRatioOf(two, one), affordedRatioOf(one, two)
	downTwo, upTwo := affordedRatioOf(two, whole), affordedRatioOf(whole, two)
	moved := affordedWindowCount(n, affordedWindowMax) -
		affordedWindowCount(n-1, affordedWindowMax)
	movedTwo := affordedWindowCount(n, affordedWindowMax) -
		affordedWindowCount(n-2, affordedWindowMax)

	// Every population of n−1, censused once, because each is the middle of 79
	// of the chains below.
	middles := make([]affordedCensus, n)
	sc := affordedScratchFor(walked)
	for d := 0; d < n; d++ {
		out, in := walked.censusDropping([]int{d}, sc, &middles[d])
		if out-in != moved && off == "" {
			off = fmt.Sprintf("the middle population dropping %d took out %d and "+
				"put in %d, a difference of %d against %d", d, out, in, out-in, moved)
		}
	}

	type best struct {
		value float64
		at    int
		from  int
	}
	worst := func(a, b best) best {
		if b.value > a.value || (b.value == a.value && b.at >= 0 && a.at >= 0 &&
			(b.at < a.at || (b.at == a.at && b.from < a.from))) {
			return b
		}
		return a
	}
	// The pairs, enumerated so the work splits by index and the reduction can
	// tie-break on it — the same shape affordedKLeafBand uses, for the same
	// reason: the widest chain a run reports must not depend on how the work
	// was divided.
	type pair struct{ i, j int }
	pairs := make([]pair, 0, affordedDropCount(n, 2))
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			pairs = append(pairs, pair{i, j})
		}
	}
	type result struct {
		// The second step of a chain, and the two-leaf step taken whole. Two
		// readings of one census: `step` divides by a population of n−1 and
		// `whole` by the population of n.
		stepLosing, stepGaining   [len(affordedEndingNames)]best
		wholeLosing, wholeGaining [len(affordedEndingNames)]best
		off                       string
		missed                    int
		ran                       bool
	}
	workers := runtime.GOMAXPROCS(0)
	if workers > len(pairs) {
		workers = len(pairs)
	}
	results := make([]result, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			r := result{ran: true}
			defer func() { results[w] = r }()
			for e := range affordedEndingNames {
				r.stepLosing[e] = best{at: -1, from: -1}
				r.stepGaining[e] = best{at: -1, from: -1}
				r.wholeLosing[e] = best{at: -1, from: -1}
				r.wholeGaining[e] = best{at: -1, from: -1}
			}
			sc := affordedScratchFor(walked)
			defer func() { r.missed = sc.missed }()
			var leaf affordedCensus
			for c := w; c < len(pairs); c += workers {
				p := pairs[c]
				out, in := walked.censusDropping([]int{p.i, p.j}, sc, &leaf)
				if out-in != movedTwo && r.off == "" {
					r.off = fmt.Sprintf("dropping %v took out %d windows and put in "+
						"%d, a difference of %d against %d", []int{p.i, p.j}, out, in,
						out-in, movedTwo)
				}
				// The two-leaf step itself: this population against the whole.
				for e := range affordedEndingNames {
					if o, _, ok := affordedResidualOf(full[e], leaf[e], downTwo); ok {
						r.wholeLosing[e] = worst(r.wholeLosing[e], best{o, c, -1})
					}
					if o, _, ok := affordedResidualOf(leaf[e], full[e], upTwo); ok {
						r.wholeGaining[e] = worst(r.wholeGaining[e], best{o, c, -1})
					}
				}
				// And the second step of both chains that reach it: drop i
				// first, or j.
				for _, mid := range []int{p.i, p.j} {
					for e := range affordedEndingNames {
						if o, _, ok := affordedResidualOf(middles[mid][e], leaf[e], down); ok {
							r.stepLosing[e] = worst(r.stepLosing[e], best{o, c, mid})
						}
						if o, _, ok := affordedResidualOf(leaf[e], middles[mid][e], up); ok {
							r.stepGaining[e] = worst(r.stepGaining[e], best{o, c, mid})
						}
					}
				}
			}
		}(w)
	}
	wg.Wait()
	missed := 0
	for _, r := range results {
		if off == "" {
			off = r.off
		}
		missed += r.missed
	}
	if off == "" && missed > 0 {
		off = fmt.Sprintf("%d of the windows a drop put back were not in the table "+
			"of every shape a drop of up to two names can produce, so they were "+
			"classified one at a time", missed)
	}
	// How a chain is named — the middle it went through, then the other leaf.
	spellChain := func(b best) string {
		if b.at < 0 {
			return ""
		}
		p := pairs[b.at]
		other := p.i
		if b.from == p.i {
			other = p.j
		}
		return names[b.from] + " then " + names[other]
	}
	// And how a two-drop is named: both leaves, in the order they stand in the
	// struct. The same sentence affordedKLeafBand's `spell` produces, because
	// the two bands are compared with each other and a reader checking one
	// against the other has only the names to go on.
	spellPair := func(b best) string {
		if b.at < 0 {
			return ""
		}
		return names[pairs[b.at].i] + ", " + names[pairs[b.at].j]
	}
	for e, ending := range affordedEndingNames {
		step := [2]best{{at: -1, from: -1}, {at: -1, from: -1}}
		took := [2]best{{at: -1, from: -1}, {at: -1, from: -1}}
		for _, r := range results {
			if !r.ran {
				continue // a worker with no share of the pairs
			}
			step[0] = worst(step[0], r.stepLosing[e])
			step[1] = worst(step[1], r.stepGaining[e])
			took[0] = worst(took[0], r.wholeLosing[e])
			took[1] = worst(took[1], r.wholeGaining[e])
		}
		measured[ending] = affordedBand{
			losing: took[0].value, losingAt: spellPair(took[0]),
			gaining: took[1].value, gainingAt: spellPair(took[1]),
		}
		second[ending] = affordedBand{
			losing: step[0].value, losingAt: spellChain(step[0]),
			gaining: step[1].value, gainingAt: spellChain(step[1]),
		}
		b := first[ending]
		bound[ending] = affordedBand{
			losing:    affordedComposedOf(b.losing, step[0].value),
			gaining:   affordedComposedOf(b.gaining, step[1].value),
			losingAt:  b.losingAt + " then any of " + strconv.Itoa(n-1),
			gainingAt: b.gainingAt + " then any of " + strconv.Itoa(n-1),
		}
	}
	return measured, second, bound, off
}

// affordedMissingLeaf is the one name `bigger` has that `smaller` does not,
// when that is the whole of the difference between them.
//
// # What this is the premise of
//
// Two of the readings below are assertions rather than tolerances, and both
// rest on the same fact: that one of the two populations is the other one with
// a single name dropped. When that holds, the smaller population is a MEMBER
// of the family affordedOneLeafBand walks over the larger — so the residual
// between them is one of the numbers that band is the maximum of, and a
// residual above it is arithmetic that cannot happen while themeLeafSetOf
// sorts the walk the way it did.
//
// Until the names were recorded this was inferred from the leaf COUNT being
// one apart, which is a different claim. A leaf added and another renamed is
// +1 with no subset anywhere, and the band would then be a bound over a family
// that does not contain the case it brackets — a tolerance wearing a proof's
// clothes, which is the exact thing the removal reading was left out to avoid.
//
// False when the sizes are wrong, when more than one name is missing, or when
// nothing is: each of those is a population that cannot be reached from the
// other by dropping one leaf, and the caller falls back to the reading it had.
//
// Both lists are distinct names — affordedLeafNames deduplicates and the
// record's list is asserted distinct — so exactly one missing name plus a size
// difference of one is enough to make the smaller a subset of the bigger. With
// n+1 names in `bigger`, n of which are in `smaller`, and n distinct names in
// `smaller`, the two sets of n are the same set.
func affordedMissingLeaf(bigger, smaller []string) (string, bool) {
	missing, ok := affordedMissingLeaves(bigger, smaller)
	if !ok || len(missing) != 1 {
		return "", false
	}
	return missing[0], true
}

// affordedMissingLeaves is the same question for a step of any size: the names
// `bigger` has and `smaller` does not, when `smaller` is a subset of it.
//
// The subset is what the caller needs and the count is how it is established.
// With `smaller` distinct, |bigger \ smaller| equals |bigger| − |smaller|
// exactly when every name in `smaller` is in `bigger` — a name that is not
// leaves one of `bigger`'s unmatched, so the difference comes out too large.
// So the size test below is the subset test, and it is the whole premise the
// k-leaf arms rest on: the population is one of the ones the band was measured
// over, rather than one that is merely the same size as them.
//
// False on a repeat in `smaller`, which breaks that counting argument.
func affordedMissingLeaves(bigger, smaller []string) ([]string, bool) {
	if len(bigger) < len(smaller) {
		return nil, false
	}
	have := make(map[string]bool, len(smaller))
	for _, name := range smaller {
		if have[name] {
			return nil, false
		}
		have[name] = true
	}
	missing := make([]string, 0, len(bigger)-len(smaller))
	for _, name := range bigger {
		if !have[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) != len(bigger)-len(smaller) {
		return nil, false
	}
	return missing, true
}

// affordedDropCount is how many populations a k-leaf step has to choose from:
// C(n, k), the number of ways k of n names can be dropped.
//
// Written out because it is the size of the family every k-leaf assertion
// rests on, and a sentence that says "one of the 3160 populations that band
// was measured over" with 3160 typed into it is a sentence that goes on saying
// 3160 after core.Theme changes size. Multiplied before dividing at each step
// so the running value stays an integer — C(n, k) is whole at every k and the
// product of k consecutive integers is divisible by k!.
func affordedDropCount(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	c := 1
	for i := 1; i <= k; i++ {
		c = c * (n - k + i) / i
	}
	return c
}

// affordedPercent is a fraction as the sentences here print it.
//
// One multiply, in one place, for the nine spellings and eighteen places this
// file had it in.
// None of them is compared — they are all inside a message — so this is not
// the FMA hazard itself; it is the census that watches for it, which had no
// way to see any of them. `off * 100` over a float variable carries no
// `float64(`, no `math.` and no decimal point, so the reading that looks for
// those three was silent about every one, including the two that were the same
// expression in two functions. See affordedFloatSource for the inference that
// made them visible, and this is the first thing it found.
func affordedPercent(v float64) float64 {
	return v * 100
}

// affordedStepRatio is one step's band as a multiple of a smaller step's.
//
// The ×-column of the log line, and the numbers affordedKLeafBand's note
// argues from: two leaves are worth 1.7 to 2.6 times what one is, which is
// what says `band × k` is not a bound. Zero when there is nothing to divide
// by, which is a band of zero — an ending a step does not move at all — and is
// reported as a ratio nobody can take rather than as infinity.
func affordedStepRatio(bigger, smaller float64) float64 {
	if smaller == 0 {
		return 0
	}
	return bigger / smaller
}

// affordedBandRounded is a band number as it is recorded and compared.
//
// Four decimal places. The measurement is deterministic — the same names in
// the same order through the same arithmetic — so the full float would compare
// equal too, and it would be a number nobody can read in a record whose whole
// purpose is to be read. A fifth decimal place is a ten-thousandth of a
// prediction, which is under one set at every ending here.
func affordedBandRounded(v float64) float64 {
	return math.Round(v*10000) / 10000
}

// The steps this file measures a band for, smallest first.
//
// # Where this stops, which has moved twice and is now a decision
//
// C(80,k) populations: 80, 3160, 82160, 1.6M, 24M. The bound was a cost and
// not a shape, and the cost has fallen twice — once when a drop census became
// a correction of the whole population's rather than a fresh walk (see
// affordedWhole) and once when the windows those corrections put back stopped
// being re-classified per population (see affordedWhole.memoise):
//
//	          direct walk    corrected    and memoised
//	k = 1            80ms          8ms             3ms
//	k = 2           5.3s         463ms            38ms
//	k = 3        ~2 hours        12.4s           112ms
//	k = 4                    ~4 minutes         685ms
//	k = 5                                         8.4s
//
// So three leaves is on this list, and the reason is that 112ms buys the
// three-field edit the same footing every other step has: an assertion against
// a family this run's population is a MEMBER of, rather than a sentence saying
// an ending outside the band is the absence of a finding.
//
// # Four is left off, and the history says how much that costs
//
// This used to read "the fourth simultaneous field edit is a rarer thing than
// the third by about the margin the cost says", which was a guess. Nobody had
// asked how often core.Theme actually moves four leaves at once, and the git
// history has the answer.
//
// It was read like this: for every commit that touches core/, the package's
// sources at that revision are parsed and core.Theme is expanded to its
// distinct leaf NAMES — the same population affordedLeafNames produces, and
// the reading reproduces affordedMeasuredOn's eighty exactly at HEAD — and
// each commit is diffed against its predecessor. Taken over the whole history
// to 6d3bad1 (2026-09-07), the commit that brought the population to eighty.
//
// Which is also how a reader tells whether it is current: this reading ends
// where affordedMeasuredOn's population begins, so while that record says 80
// leaves the table below covers every edit there has been. The run that
// re-takes the record for an eighty-first leaf is the run that has one more
// row to add here, and it knows its own size — the failure prints it.
//
// # And the walker is in the repository, so the next reading is a command
//
// The first version of this table came out of a throwaway AST walker in a
// scratch directory, and what survived into the file was the numbers, the
// method in this paragraph, and the commit it was taken at. That is the
// convention the rest of this file has been moving AWAY from: the record's
// census is re-walked on every run precisely because a number in a note is a
// number that has already moved.
//
// It cannot become an arm — a test that shells out to git is a test that
// fails in a shallow clone, a source tarball or a build container, none of
// which have a history to read — but it can stop being a re-derivation:
//
//	go run ./internal/themehistory
//	go run ./internal/themehistory -names
//
// prints the per-commit listing this table is a histogram of, and the
// population at HEAD to check it against affordedMeasuredOn's. The second
// form prints the names, which is the stronger check: two different sets of
// eighty print the same 80.
//
// And that check is no longer a thing somebody remembers to run. The command
// needs git; the EXPANSION under it does not, and
// TestTheHistoryWalkersExpansionIsTheOneThisFileMeasures holds
// internal/themeleaves over a working-tree core/ against affordedLeafNames()
// name for name on every run. So what a reader still has to take on trust here
// is that the sixteen revisions were read by that same expansion — not that
// the expansion produces this file's population.
//
// Fifteen commits moved the population after the one that created it with
// twenty-five leaves in it:
//
//	leaves moved   commits   what they were
//	           1         8   a role, a state, a flag: one field, one decision
//	           2         2
//	           3         2   three Accessibility fields; Border/Success/Warning
//	           4         1   Max/Min/Now/Text — a slider's value range
//	           5         1   a second tone per palette role
//	          28         1   the style-props surface arriving at once
//
// Two things fall out of that, and neither is what the guess said.
//
// The fourth simultaneous edit is NOT rare against the third: three leaves
// moved on two commits and four on one, which is a margin of two to one
// against a cost margin of six. And there is no k at which the family becomes
// complete — a five-leaf edit happened as often as a four-leaf one, and the
// twenty-eight-leaf one is past anything enumerable at all. Field edits arrive
// FEATURE-sized: Max/Min/Now/Text is one slider, and the five is one decision
// about palette roles. A feature's size is not bounded by what this test can
// afford to walk.
//
// So the list stops at three, and now for a reason with a distribution under
// it rather than a guess: k = 1..3 covers 12 of the 14 non-bootstrap edits,
// k = 4 would cover 13 for 685ms — six times what the whole rest of this test
// costs — and would still leave the sentence about an ending outside the band
// being the absence of a finding in place for the fourteenth. Five is where
// the wall is, and past five the question stops being about cost.
//
// # And one thing the history says that no band here is measuring
//
// Every one of those sixteen commits ADDED leaves. Not one of them removed or
// renamed a leaf name in the whole history of the struct. So affordedBand's
// `losing` number — the residual of a population read against a LARGER one —
// is a band over a direction core.Theme has never actually gone, and every
// arm that has ever fired against a real edit fired against `gaining`. It is
// still measured, because the direction is a real edit somebody can make and
// the day they make it is the day the band has to be there; but a reader
// weighing what these numbers have caught should know the two halves have not
// had equal exposure.
//
// See affordedKLeafBand for what fails when a bigger step is scaled from a
// smaller one instead, and affordedChainBoundOf for the composition that was
// supposed to get past all of this and does not.
//
// A list rather than a spelled-out block per step because every reading below
// is the same arms per step, and copies of them is how the two-leaf record
// came to state a population the one-leaf record also stated — and how the
// four residual arms came to be two pairs of near-identical prose. Adding a
// step here adds its record arms, its assertion arms and its line in the log.
var affordedBandSteps = []int{1, 2, 3}

// What a step of k leaves is worth to each ending, measured, and what the
// one-leaf band composes to over the same step.
//
// The same shape as affordedMeasuredOn one level up: a reading taken against
// one population, recorded as such, with the bracket derived from it rather
// than typed in beside it.
//
// # One record, one population
//
// This was two records — one per step — and each of them stated the population
// it was taken over: `leaves: 80` and `window: 6`, asserted equal to
// affordedMeasuredOn's, which is one population written down three times. The
// two-leaf record had already dropped both fields and said why, so the file
// carried two conventions for one fact.
//
// So the population is stated once, in affordedMeasuredOn, and these are
// measurements OF it: every band here is re-derived over affordedMeasuredOn's
// own names on every run, which is what makes the leaves-and-window fields
// unnecessary rather than merely redundant. A band is a function of a list of
// names; the list is next door; there is nothing left for a second copy of the
// count to protect.
//
// # What each step is
//
//	step[1]  eighty populations — the record's names with each one dropped in
//	         turn, read in both directions. The family a run one leaf from the
//	         record belongs to.
//	step[2]  3160 populations, every pair dropped. Not twice step[1]: gaining
//	         runs to 2.6× the one-leaf figure, which is the measurement that
//	         says `band × k` is not a bound.
//	step[3]  82160 populations, every triple. The step that used to be twelve
//	         seconds and is now 112ms — see affordedWhole.memoise — and the
//	         measurement that says `band × k` is not merely wrong at two but
//	         getting worse: the crowding ending gains 224.44% over three
//	         leaves, which is 6.75× its one-leaf figure and 2.55× its two-leaf
//	         one. Three times the one-leaf band is 99.8%, so a scaled bound
//	         would call a three-field edit a re-sort by a factor of two.
//	chain    what the one-leaf band COMPOSES to over a two-leaf step, which is
//	         the other construction that was supposed to reach past the
//	         enumerable steps. See affordedChainBoundOf: the identity is exact
//	         and the bound is not one — the crowding ending's gaining figure
//	         composes to 77.55% against a measured 87.88%, so a chain bound
//	         would call an honest two-field edit a re-sort, which is the same
//	         failure `band × k` has and the same failure the stated tenth had
//	         before it.
//
// It is also a second reading on themeLeafSetOf, and a much wider one than the
// census: the census is one walk over 80 names, step[1] is eighty walks over
// 79, and step[2] is 3160 walks over 78. A re-sort that happened to leave the
// census where it was still has three thousand populations to get past.
var affordedBandMeasuredOn = struct {
	step  map[int]map[string]affordedBand
	chain map[string]affordedBand
	sound map[string]affordedBand
}{
	step: map[int]map[string]affordedBand{
		1: {
			"its own crowding stopped the search":                 {losing: 0.2496, gaining: 0.3326},
			"no width crowds these names at all":                  {losing: 0.0002, gaining: 0.0002},
			"the ceiling cost this set a wider threshold":         {losing: 0.0292, gaining: 0.0284},
			"the ceiling and the crowding stop in the same place": {losing: 0.2163, gaining: 0.2760},
		},
		2: {
			"its own crowding stopped the search":                 {losing: 0.4677, gaining: 0.8788},
			"no width crowds these names at all":                  {losing: 0.0004, gaining: 0.0004},
			"the ceiling cost this set a wider threshold":         {losing: 0.0510, gaining: 0.0485},
			"the ceiling and the crowding stop in the same place": {losing: 0.3802, gaining: 0.6135},
		},
		3: {
			"its own crowding stopped the search":                 {losing: 0.6918, gaining: 2.2444},
			"no width crowds these names at all":                  {losing: 0.0007, gaining: 0.0007},
			"the ceiling cost this set a wider threshold":         {losing: 0.0734, gaining: 0.0684},
			"the ceiling and the crowding stop in the same place": {losing: 0.4897, gaining: 0.9596},
		},
	},
	// The composed two-step bound over ONE chain, recorded so the finding is a
	// pair of numbers a run re-derives rather than a sentence in a note. Read
	// against step[2] by the arm that says the composition does not cover it.
	chain: map[string]affordedBand{
		"its own crowding stopped the search":                 {losing: 0.5613, gaining: 0.7755},
		"no width crowds these names at all":                  {losing: 0.0004, gaining: 0.0004},
		"the ceiling cost this set a wider threshold":         {losing: 0.0597, gaining: 0.0580},
		"the ceiling and the crowding stop in the same place": {losing: 0.4792, gaining: 0.6279},
	},
	// And the same composition over EVERY chain, which is a bound and is
	// therefore not a finding but a check. See affordedTwoStepBands: with
	// b_i taken over all intermediate populations the product dominates every
	// two-drop residual by construction, so these eight numbers standing above
	// step[2]'s eight is an agreement between two walks that share nothing but
	// the classification — and a disagreement is one of them being wrong.
	//
	// They are also what says the construction is dead rather than merely
	// unlucky: sound and 1.85x the measurement it would replace on the ending
	// that matters, and never cheaper at any k, because bounding the last step
	// over all chains needs a census of every population of size n−k — which
	// is exactly the family the direct measurement walks.
	sound: map[string]affordedBand{
		"its own crowding stopped the search":                 {losing: 0.8661, gaining: 1.6303},
		"no width crowds these names at all":                  {losing: 0.0004, gaining: 0.0004},
		"the ceiling cost this set a wider threshold":         {losing: 0.0654, gaining: 0.0633},
		"the ceiling and the crowding stop in the same place": {losing: 0.5363, gaining: 0.7316},
	},
}

// affordedStepLeaves is a step's size with its noun, so a sentence about it
// does not read "a step of 1 leaves".
func affordedStepLeaves(k int) string {
	if k == 1 {
		return "1 leaf"
	}
	return fmt.Sprintf("%d leaves", k)
}

// affordedNameList is a handful of leaf names as a sentence names them.
//
// "SM, XL and XS" rather than "SM and XL and XS", which is what joining on
// " and " produced the moment a step of three arrived — the two-leaf arms had
// exactly two names and the join read correctly by accident.
func affordedNameList(names []string) string {
	switch len(names) {
	case 0:
		return "nothing"
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// affordedStepWord is how a step of k reads in a sentence: which of the
// record's names each population dropped.
func affordedStepWord(k int) string {
	switch k {
	case 1:
		return "each one"
	case 2:
		return "every pair"
	}
	return fmt.Sprintf("every %d", k)
}

// affordedStepMissing is what an ending with no recorded band at this step
// loses by not having one.
//
// Two different findings: at one leaf the residual falls back to
// affordedResidualQuiet, a number the measurement says is three times the
// drift of the large endings and a third of the drift of the small ones; past
// one leaf there is no fallback that means anything, because a smaller step's
// band cannot be scaled up to stand in for a larger one.
func affordedStepMissing(k int) string {
	if k <= 1 {
		return fmt.Sprintf("The residual for this ending is then read against the "+
			"stated fallback of %.0f%% — which the measurement says is three times "+
			"the drift of the large endings and a third of the drift of the small "+
			"ones — and nothing says it happened.", affordedResidualQuiet*100)
	}
	return fmt.Sprintf("A run %d leaves from the record then has no family to be a "+
		"member of for this ending, and a smaller step's band cannot stand in for "+
		"it: the drift is not linear in the number of leaves moved.", k)
}

// affordedStepRetake is what a drifted band at this step means for the
// arguments built on it, and what re-taking it involves.
func affordedStepRetake(k int) string {
	if k <= 1 {
		return "If the new sort is what was wanted, re-take affordedMeasuredOn and " +
			"affordedBandMeasuredOn together from the log line — and read the " +
			"derivation first, because a record re-taken over a classification that " +
			"moved by accident records the accident as the baseline."
	}
	return "These numbers are also two arguments. They are what says `band × k` is " +
		"not a bound — gaining runs to 2.6× the one-leaf figure on the small " +
		"endings — and they are what the composed chain bound is held against, so " +
		"a change here changes both. Re-take the record and read affordedKLeafBand's " +
		"note with it."
}

// affordedStepWider is what a step's band reads that the step below it does
// not — the sentence that says why a drift here can be invisible there.
func affordedStepWider(k int) string {
	if k <= 1 {
		return "which the census re-derivation above can miss: it reads one " +
			"population and this reads eighty"
	}
	return fmt.Sprintf("which the %d-leaf band can miss: it reads %d of them",
		k-1, affordedDropCount(affordedMeasuredOn.leaves, k-1))
}

// affordedBandFor is the band one ending is read against, and where it came
// from.
//
// The measurement when there is one for this ending, and affordedResidualQuiet
// when there is not — which is a reworded ending, the same state
// affordedEndingBracket falls back to the stated floor in. The fallback is
// named in the string rather than left to look like a reading, because the
// whole finding above is that one number is not a band for four endings this
// far apart, and an ending arriving with no measurement is exactly the case
// where that matters and nobody can tell from the number.
// `whose` names the population the band was measured over, because there are
// now two: the band over THIS run's names, which is the family a run one leaf
// LONG than the record belongs to, and the band over the RECORD's names, which
// is the family a run one leaf SHORT belongs to. They are different numbers
// measured over different populations, and a sentence that did not say which
// would be the one thing a reader needs to check the claim.
func affordedBandFor(bands map[string]affordedBand, step, over int,
	whose, ending string, gaining bool) (float64, string) {
	b, known := bands[ending]
	if !known {
		return affordedResidualQuiet, fmt.Sprintf(
			"the stated fallback of %.0f%% — the band is measured per ending and "+
				"there is no measurement for this one", affordedResidualQuiet*100)
	}
	// How many leaves moved, spelled as the sentence needs it. This used to be
	// "a single leaf" whatever the band was, so the two-leaf arms cited a
	// two-leaf band as a one-leaf one — and beside it a population of 80 where
	// the family has C(80,2) = 3160 members. Neither was in the arithmetic:
	// both arms asserted against the right numbers and described them wrongly,
	// which is the failure this file keeps naming in the other direction.
	moved := "a single leaf"
	if step != 1 {
		moved = fmt.Sprintf("%d leaves", step)
	}
	if gaining {
		return b.gaining, fmt.Sprintf(
			"the %.2f%% %s ADDED to %s moves this ending, "+
				"measured over all %d of them (the widest being %q)",
			affordedPercent(b.gaining), moved, whose, over, b.gainingAt)
	}
	return b.losing, fmt.Sprintf(
		"the %.2f%% %s REMOVED from %s moves this ending, "+
			"measured over all %d of them (the widest being %q)",
		affordedPercent(b.losing), moved, whose, over, b.losingAt)
}

// affordedRetakeSource is what a re-take of these records would look like in
// the source, printed from what this run actually walked.
//
// # The flag this is instead of
//
// Re-taking affordedMeasuredOn means putting eighty names and four counts into
// a struct literal, and re-taking the bands means eight numbers beside them.
// Nothing generated any of it, so it was transcription — and the obvious fix,
// a `-update` flag that rewrites the literals, is the one this file cannot
// have: a record a test can rewrite is a record that re-baselines an accident,
// which is the failure every message here warns about. A green run after an
// accidental re-sort is exactly what the whole file is built to prevent.
//
// So the run prints the record and a person pastes it. The difference is not
// cosmetic — the decision to accept a new classification stays with whoever
// reads the derivation, which is the thing being protected, and what goes away
// is the copying, which protects nothing. Every message that says "re-take
// from the log line" ends in this.
//
// Printed from THIS run's walk rather than from the record, because the whole
// occasion for reading it is that the two disagree.
func affordedRetakeSource(names []string, census map[string]int,
	step map[int]map[string]affordedBand, chain, sound map[string]affordedBand) string {
	record, bands := affordedRetakeParts(names, census, step, chain, sound)
	out := "\n\nThis run's walk, in the shape the records are written in — paste " +
		"it once the derivation has been read and the new sort is the one that " +
		"was wanted:\n\n" + record
	if bands != "" {
		out += "\nand the bands over those names:\n\n" + bands
	}
	return out
}

// affordedRetakeParts is the two pastes themselves, without the sentences
// around them.
//
// Separate from the prose because they are parsed as well as printed: see
// TestTheRetakePasteIsTheShapeTheRecordsAreIn, which reads the real
// declarations out of this file and checks that what a failing run offers a
// reader would actually go where it says. A printer nobody parses is a
// printer that goes on emitting a field somebody renamed — silently, on the
// one run where it is needed.
func affordedRetakeParts(names []string, census map[string]int,
	step map[int]map[string]affordedBand, chain, sound map[string]affordedBand) (
	record, bands string) {
	var b strings.Builder
	fmt.Fprintf(&b, "\tleaves: %d,\n\twindow: %d,\n\tnames: []string{\n",
		len(names), affordedWindowMax)
	// Wrapped the way the literal is, so what comes out is a paste and not a
	// line a hundred names long.
	line := "\t\t"
	for _, name := range names {
		item := fmt.Sprintf("%q, ", name)
		if len(line)+len(item) > 78 {
			b.WriteString(strings.TrimRight(line, " ") + "\n")
			line = "\t\t"
		}
		line += item
	}
	if strings.TrimSpace(line) != "" {
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}
	b.WriteString("\t},\n\tending: map[string]int{\n")
	// Sorted, because a map printed in range order is a different paste every
	// run and a reader comparing two of them would be reading the shuffle.
	keys := make([]string, 0, len(census))
	for key := range census {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "\t\t%q: %d,\n", key, census[key])
	}
	b.WriteString("\t},\n")
	record = b.String()
	if len(step) == 0 {
		return record, ""
	}
	b.Reset()
	b.WriteString("\tstep: map[int]map[string]affordedBand{\n")
	for _, k := range affordedBandSteps {
		measured, taken := step[k]
		if !taken {
			fmt.Fprintf(&b, "\t\t// %d: not measured on this run\n", k)
			continue
		}
		fmt.Fprintf(&b, "\t\t%d: {\n", k)
		for _, ending := range keys {
			band := measured[ending]
			fmt.Fprintf(&b, "\t\t\t%q: {losing: %.4f, gaining: %.4f},\n",
				ending, affordedBandRounded(band.losing),
				affordedBandRounded(band.gaining))
		}
		b.WriteString("\t\t},\n")
	}
	b.WriteString("\t},\n")
	// The two compositions, in the same shape and under their own names. They
	// are re-taken with the bands rather than separately: the chain is the
	// composition over one representative chain and the sound one over all of
	// them, and both move when either one-leaf band does.
	for _, named := range []struct {
		key   string
		bands map[string]affordedBand
	}{{"chain", chain}, {"sound", sound}} {
		if len(named.bands) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\t%s: map[string]affordedBand{\n", named.key)
		for _, ending := range keys {
			band := named.bands[ending]
			fmt.Fprintf(&b, "\t\t%q: {losing: %.4f, gaining: %.4f},\n",
				ending, affordedBandRounded(band.losing),
				affordedBandRounded(band.gaining))
		}
		b.WriteString("\t},\n")
	}
	return record, b.String()
}

func TestTheAffordedWidthHoldsItsTwoRelations(t *testing.T) {
	// Which branch of the classification each of the four sentences names,
	// asked first because everything below is a partition BY those sentences
	// and the index-to-sentence map is held by the order of a literal. See
	// affordedEndingWitness.
	affordedHoldEndingWitnesses(t)

	// The band memo's counters before this test has asked for anything, so
	// the sentence at the bottom reporting them can say "this run" and mean
	// it. They are the process's and this test is only their busiest reader.
	// See affordedBandMemoSince.
	baseWalks, baseReused := affordedBandMemoRead()

	names := affordedLeafNames()
	sets := affordedSets(names, affordedWindowMax)

	// Reported once at the end rather than per set: 20-odd thousand sets can
	// break a relation in the same way twenty thousand times, and a failure
	// list that long is a failure nobody reads. The first of each is the one
	// that gets the sentence.
	told := map[string]bool{}
	// And which of the derivation's four endings each set reached.
	//
	// A relation asserted over a population that never produces the state it is
	// about passes for free, and both relations above are biconditionals: the
	// open flag's half is worth nothing unless sets on each side of it are in
	// the population. This is the census that says they were — the same
	// argument the ink scan's counts are kept for, one directory over.
	reached := map[string]int{}
	for _, leaves := range sets {
		set := themeLeafSetOf(leaves)
		where := strings.Join(leaves, ", ")
		reached[affordedEndingOf(set)]++

		if set.afforded < set.edits && !told["under"] {
			told["under"] = true
			t.Errorf("%s measures a threshold of %d and reports affording %d.\n\n"+
				"afforded is the width these names would carry with "+
				"themeNearMissReach taken off, so it is the same search continued and "+
				"can only go up. A value under the threshold is the message telling a "+
				"reader the ceiling cost them something while the ceiling was the "+
				"thing raising the answer — and the sentence it prints, \"these names "+
				"would carry %d\", would be citing a width narrower than the one in "+
				"force.", where, set.edits, set.afforded, set.afforded)
		}

		if !set.cappedByReach && (set.afforded != set.edits || set.affordedOpen) &&
			!told["uncapped"] {
			told["uncapped"] = true
			t.Errorf("%s was stopped by its own crowding at %d and reports affording "+
				"%d (open=%v).\n\n"+
				"When the crowding is what stopped the search there is nothing for "+
				"the ceiling to have cost: raising themeNearMissReach finds the same "+
				"crowd one step out. afforded is assigned edits for exactly that "+
				"reason and the open flag is never reached, so a set that disagrees "+
				"is the assignment having moved out from under the message that "+
				"reads it.", where, set.edits, set.afforded, set.affordedOpen)
		}

		// # And the arrangement itself, which nothing else here can see
		//
		// The two-parent sets alternate Left/Right and start with Left. That is
		// a property of the WALK and not of the derivation, and the difference
		// matters: swapping the two parent names gives the mirror set — the
		// same partition of names into two groups, the same distances inside
		// each, the same ending — so every reading in this file is invariant
		// under it. A walk that got the alternation wrong would be measuring a
		// population no census, band or record could tell from this one, and
		// all of them would go on holding.
		//
		// It is asserted because affordedEachSet no longer builds the
		// arrangement per window. It prefixes the names once and picks one of
		// two lists by the window start's parity, which is arithmetic that can
		// be off by one — and this arm is the only thing that would say so.
		if !strings.HasPrefix(leaves[0], "One.") {
			for j, leaf := range leaves {
				want := "Left."
				if j%2 == 1 {
					want = "Right."
				}
				if strings.HasPrefix(leaf, want) || told["arrangement"] {
					continue
				}
				told["arrangement"] = true
				t.Errorf("%s is the two-parent arrangement and its leaf at position "+
					"%d is %q, where the walk mounts them under %q.\n\n"+
					"The sets alternate and begin with Left, and nothing downstream "+
					"can see that they do: the mirror of this set is the same two "+
					"groups under swapped names, which every distance, every ending "+
					"and therefore every band and record here is blind to. So a walk "+
					"that lost the alternation would keep producing a population that "+
					"passes all of them and is not the one affordedMeasuredOn was "+
					"taken over.", where, j, leaf, want)
			}
		}

		// The shape the open flag claims: a parent holding three leaves is the
		// smallest thing that can crowd, because a crowd is a leaf with more
		// than one sibling inside the threshold.
		crowdable := false
		byParent := map[string]int{}
		for _, leaf := range leaves {
			byParent[leaf[:strings.LastIndex(leaf, ".")+1]]++
		}
		for _, n := range byParent {
			if n >= 3 {
				crowdable = true
			}
		}
		if set.affordedOpen != !crowdable && !told["open"] {
			told["open"] = true
			// Said in the direction the set is actually in: the flag is wrong
			// both ways and the two are different mistakes, so the sentence
			// names the shape rather than printing a boolean beside it.
			shape := "has a parent holding three leaves"
			if !crowdable {
				shape = "has no parent holding three leaves"
			}
			t.Errorf("%s %s, and reports "+
				"open=%v (afforded %d, threshold %d, capped=%v).\n\n"+
				"The flag says the upward search ran out of distances to try rather "+
				"than finding a crowd, and past the longest name every sibling pair "+
				"is inside the threshold — so a parent with three leaves crowds at "+
				"SOME width and the search cannot run out. The two answers are a "+
				"width and a shape, and they are printed as different sentences: one "+
				"tells a reader how much wider a threshold these names would carry, "+
				"and the other tells them there is no such width to look for.",
				where, shape, set.affordedOpen, set.afforded, set.edits,
				set.cappedByReach)
		}
	}
	// Every ending, and each of them by more than an accident.
	//
	// Zero is the failure this started as and it is the last state of a slide,
	// not the first: an ending that 27 sets reach is separating the relations,
	// an ending that 1 reaches is a relation passing over a population that
	// cannot break it, and both of those pass a census that only asks about
	// zero. See affordedEndingFloor — the two arms say different things
	// because they are different findings, and the thin one names the knob
	// that moves it.
	// The four sentences, from the one place they are written. This used to be
	// a second copy of the list beside the switch that produces them, held to
	// it by nothing: an ending added to affordedEndingAt and not to this slice
	// would be counted by no arm here, and every partition below would still
	// add up over the four it was told about.
	endings := affordedEndingNames[:]
	// And the record's keys against them, because the bracket below is looked
	// up BY the ending's own sentence. A reworded ending finds nothing in the
	// map, falls back to the stated floor, and goes on passing with its share
	// of a measurement quietly gone — which is this whole census's failure
	// mode arriving through the lookup that fixes it.
	for _, ending := range endings {
		if _, known := affordedMeasuredOn.ending[ending]; !known {
			t.Errorf("affordedMeasuredOn carries no reading for %q.\n\n"+
				"The floor each ending is held to is a share of what that ending "+
				"scored when the record was taken, looked up by the sentence itself. "+
				"An ending the record does not name falls back to the stated floor of "+
				"%d — which is a bracket under 27 and nothing at all under 473 — and "+
				"nothing says it happened. Re-take the reading under the new wording, "+
				"or leave the wording alone.", ending, affordedEndingFloor)
		}
	}
	for key := range affordedMeasuredOn.ending {
		if !slices.Contains(endings, key) {
			t.Errorf("affordedMeasuredOn carries a reading for %q and the derivation "+
				"has no such ending.\n\n"+
				"Every set reaches exactly one of the four sentences above, so a fifth "+
				"in the record is a reading for an ending that was renamed or removed "+
				"— and its partner is an ending running on the stated floor with "+
				"nobody having decided that.", key)
		}
	}
	// And the walk against the formula the scale is taken with.
	//
	// affordedSetCount describes the loop above and the loop is what actually
	// produced these counts. A formula that has come apart from it would leave
	// every reading below a ratio between this population and an imaginary one
	// — stated to two decimal places, which is the shape a number nobody
	// derived arrives in.
	if want := affordedSetCount(len(names), affordedWindowMax); len(sets) != want {
		t.Fatalf("the walk produced %d sets over %d distinct leaf names at a window "+
			"of %d, and affordedSetCount says %d.\n\n"+
			"That function is the same walk written as arithmetic, and it is what "+
			"scales this run's population against the one affordedMeasuredOn was "+
			"taken over — both ends of that division go through it. A disagreement "+
			"makes every reading below a comparison with a population that was never "+
			"walked, so nothing further is asked.",
			len(sets), len(names), affordedWindowMax, want)
	}
	// And both censuses as partitions, which is what every prediction below
	// rests on.
	//
	// The switch that classifies a set has a default arm, so each set reaches
	// exactly one ending and the four counts add up to the walk. That is what
	// makes an ending's own share of the population a thing the scale can
	// predict — and, one step further, what makes the four residuals sum to
	// zero, so a shortfall in one ending is necessarily a surplus in another
	// and affordedShortfallCause can name where the sets went instead of
	// asserting that they left. See affordedTakers.
	//
	// Asked of both ends because either can come apart on its own: a fifth
	// ending would leave this run's counts short of the walk, and a re-measure
	// that mistyped one number would leave the RECORD short of the population
	// it says it was taken over — and that one is invisible from the record,
	// because each ending's number still looks like a count.
	census := 0
	for _, ending := range endings {
		census += reached[ending]
	}
	if census != len(sets) {
		t.Fatalf("the walk produced %d sets and the four endings account for %d of "+
			"them.\n\n"+
			"Every set reaches exactly one of the four sentences — the switch above "+
			"has a default arm — and that partition is what every prediction below "+
			"rests on: an ending's floor is its share of a population, and the "+
			"reading that tells a redistribution from an arm going is that the "+
			"residuals have to sum to zero. With sets unaccounted for, a shortfall "+
			"could be either and nothing here could say which, so nothing further is "+
			"asked. The census is %v.", len(sets), census, reached)
	}
	if total, want := affordedRecordTotal(),
		affordedSetCount(affordedMeasuredOn.leaves, affordedMeasuredOn.window); total != want {
		t.Fatalf("affordedMeasuredOn's four counts add up to %d and the walk it says "+
			"it was taken over produces %d sets.\n\n"+
			"The record is a partition of that population, so those two are the same "+
			"number by construction and a disagreement means one of the counts was "+
			"mistyped when it was written down — which is invisible from the record "+
			"itself, because the wrong number still looks like a count. Everything "+
			"below scales this run against that total: the floors are shares of "+
			"these numbers and the redistribution reading is the residuals summing "+
			"to zero, so a record that is not a partition makes every one of them a "+
			"comparison with a population nobody walked. Re-take the reading (the "+
			"census in the log line) rather than adjusting one entry to make the sum "+
			"come out. The record reads %v.", total, want, affordedMeasuredOn.ending)
	}
	// # The record's own census, re-derived — the reading that needs nothing
	//
	// Everything else in this test compares THIS run's walk against the record
	// and therefore has to argue about how far the two populations are apart.
	// This one does not: the record now carries the names it was taken over
	// (see affordedMeasuredOn), the walk is a function of a list of names, and
	// so the record's four counts can simply be produced again.
	//
	// That makes it the strongest reading here and the only one that runs on
	// every population. The exact arm below is silent the moment core.Theme
	// gains, loses or renames a leaf — which is the moment somebody is editing
	// it, which is the moment it would be read — and this one is not, because
	// it is not about core.Theme at all. It is about themeLeafSetOf: the record
	// is 80 strings and 4 counts, and a run that cannot reproduce the counts
	// from the strings has changed how a set is classified.
	//
	// Asked before anything that divides by the record, because a record whose
	// own census cannot be reproduced is a record every scaled reading below
	// would be comparing against a population nobody can walk.
	if len(affordedMeasuredOn.names) != affordedMeasuredOn.leaves {
		t.Fatalf("affordedMeasuredOn says it was taken over %d leaf names and "+
			"carries %d of them.\n\n"+
			"The two are the same fact written twice and the count is the one every "+
			"scaled reading below divides by, so a disagreement makes the scale a "+
			"ratio against a population that is neither of them. Re-take the record "+
			"as a whole — the census and the names in the log line — rather than "+
			"adjusting one of the two to agree with the other.",
			affordedMeasuredOn.leaves, len(affordedMeasuredOn.names))
	}
	{
		seen := map[string]bool{}
		for _, name := range affordedMeasuredOn.names {
			if seen[name] {
				t.Fatalf("affordedMeasuredOn's names carry %q twice.\n\n"+
					"affordedLeafNames deduplicates, because two parents can hold the "+
					"same leaf name and a window carrying it twice would mount one "+
					"parent with two identical children — a shape the walk cannot "+
					"produce. A repeat here is a record of a population the walk was "+
					"never taken over, and it is also what makes affordedMissingLeaf's "+
					"counting argument unsound: the two arms that are assertions rather "+
					"than tolerances both rest on it.", name)
			}
			seen[name] = true
		}
		recordCensus := affordedCensusMap(affordedCensusOf(affordedMeasuredOn.names,
			affordedMeasuredOn.window))
		for _, ending := range endings {
			want, known := affordedMeasuredOn.ending[ending]
			if !known {
				continue // reported by the key-set arms above
			}
			if recordCensus[ending] == want {
				continue
			}
			t.Errorf("walking affordedMeasuredOn's own %d names at its own window of "+
				"%d puts %d sets at %q, and the record says %d.\n\n"+
				"The walk is a function of the names and the record carries them, so "+
				"this comparison is exact and it does not depend on what core.Theme "+
				"is today: the same strings in the same order through the same "+
				"arithmetic. A disagreement is themeLeafSetOf classifying a set "+
				"differently than it did when the record was taken — which is the "+
				"finding every other reading in this test is trying to make from a "+
				"distance, arriving without a scale, a band or a population to argue "+
				"about.\n\n"+
				"This run's own census is %v over %d names; the record's re-walk is "+
				"%v against a record of %v.\n\n"+
				"If the new sort is what was wanted, re-take affordedMeasuredOn and "+
				"affordedBandMeasuredOn together — and read the derivation first, "+
				"because a record re-taken over a classification that moved by "+
				"accident records the accident as the baseline.",
				len(affordedMeasuredOn.names), affordedMeasuredOn.window,
				recordCensus[ending], ending, want,
				reached, len(names), recordCensus, affordedMeasuredOn.ending)
		}
	}

	// What this whole census comes to against the one the record was taken
	// over. See affordedScale: one number, because one walk produced all four
	// endings.
	scale := affordedScale(len(names))

	for _, ending := range endings {
		if reached[ending] == 0 {
			t.Errorf("not one of the %d generated sets ended with %q.\n\n"+
				"The relations above are biconditionals and this is the reading that "+
				"says they were asked of both sides. An ending no set reaches is a "+
				"relation passing over a population that cannot break it — which is "+
				"the same failure as a table of four cases somebody wrote down, "+
				"arriving with a bigger number in front of it.", len(sets), ending)
			continue
		}
		b := affordedEndingBracket(ending)
		if reached[ending] < b.floor {
			t.Errorf("%d of the %d generated sets ended with %q, against a floor of "+
				"%d — %s.%s\n\n"+
				"The ending is still reached, so the relations above are still being "+
				"asked of both sides — and by a population thin enough that the next "+
				"change to either end of it decides whether they are asked at all. "+
				"The whole census is %v.\n\n"+
				"Neither end of this is an assertion anybody has made. The sets are "+
				"windows of one to %d consecutive names over core.Theme's own leaves "+
				"(affordedWindowMax), so this number moves when that bound moves and "+
				"when the struct gains or loses leaves — and it moves without a "+
				"message, because a relation that holds over 1 set holds. That is the "+
				"margin this floor is here to notice leaving.%s",
				reached[ending], len(sets), ending, b.floor, b.why,
				affordedShortfallCause(b, scale, ending, reached, endings), reached,
				affordedWindowMax, affordedMeasuredNote(len(names), len(sets)))
		}
	}

	// # And the redistribution itself, where there is nothing to argue about
	//
	// Everything above is a bound on ONE ending, and a re-sort of the walk
	// clears all four of them: 377 sets moving from one ending to another
	// leaves both above their floors, both above what the scale predicts to
	// within the floors' own slack, and the census reading as healthy. That is
	// the failure the per-ending floors were introduced for, arriving one level
	// up — the same move this file has now made three times.
	//
	// The residual is what sees it, and on an unmoved population it needs no
	// tolerance at all. Same leaf names, same window bound: the walk produces
	// the same sets in the same order, themeLeafSetOf is a function of a set,
	// and therefore every ending's count is the one that was recorded. Not
	// "about the one" — the one. So the comparison is exact, and anything that
	// is not equal is the classification having moved, with no re-measure to
	// hedge about.
	//
	// Asked only in that state on purpose. When the population HAS moved the
	// counts are expected to differ and by how much is a question about which
	// windows crowd, which is a shape the scale does not model — that reading
	// stays in the log line, under affordedResidualQuiet, where it is a
	// sentence rather than a bound nobody measured.
	//
	// # And "the same population" is now the names and not the count of them
	//
	// This arm used to run whenever the leaf COUNT matched, on a premise about
	// the names: "the walk is a function of those two numbers and they have not
	// moved". A leaf RENAMED keeps the count and changes the strings the walk
	// is over — so the sets are different sets, the counts may legitimately
	// differ, and this arm would have reported themeLeafSetOf re-sorting a walk
	// nobody re-sorted. The premise was checkable the moment the record carried
	// its names, and it is checked.
	//
	// With the names equal this is the same claim the re-derivation above makes
	// over the record's own list, reached through the test's own walk rather
	// than through affordedCensusOf. It is kept because its message is the one
	// about the population in force, and because the pair covers the whole
	// question between them: this arm goes quiet exactly when the names differ,
	// and that is when the re-derivation is the only reading left.
	sameNames := slices.Equal(names, affordedMeasuredOn.names)
	if sameNames && affordedWindowMax == affordedMeasuredOn.window {
		for _, ending := range endings {
			measured, known := affordedMeasuredOn.ending[ending]
			if !known || reached[ending] == measured {
				continue
			}
			t.Errorf("%q was reached by %d of the %d sets and affordedMeasuredOn "+
				"records %d, over the same %d leaf NAMES at the same window of %d.\n\n"+
				"The walk is a function of the names and the record carries them, so "+
				"this run walked the record's own list — the same %d sets in the same "+
				"order — and which ending a set reaches is a function of the set. "+
				"These two counts are therefore the same number or themeLeafSetOf is "+
				"sorting the walk differently than it was when the record was "+
				"taken.\n\n"+
				"That is the finding the per-ending floors cannot make. A re-sort "+
				"moves the four endings against each other while the population is "+
				"identical: every floor stays clear, every relation goes on being "+
				"asked, and the census reads healthy while the sets answering each "+
				"question are not the ones the record was about. The whole census is "+
				"%v against a record of %v.\n\n"+
				"If the new sort is what was wanted, re-take affordedMeasuredOn from "+
				"the census in the log line — and read the derivation first, because "+
				"a record re-taken over a classification that moved by accident "+
				"records the accident as the baseline.",
				ending, reached[ending], len(sets), measured, len(names),
				affordedWindowMax, len(sets), reached, affordedMeasuredOn.ending)
		}
	}

	// # The band a single leaf is worth, re-derived and held to its record
	//
	// Everything below this point reads a residual against a band, and the band
	// is a measurement of this run's own names (see affordedOneLeafBand): every
	// population one leaf away from this one, read in both directions. Taken
	// here, once, because the two readings that use it — the arm that runs when
	// a leaf has been ADDED and the log line's per-ending quiet check — have to
	// be reading the same numbers.
	// Every walk's own window accounting, collected and asserted once below.
	// A walk that took out and put back numbers that do not add up has
	// produced a census of a population nobody can name, and the band it feeds
	// would look exactly like a band.
	bookkeeping := []string{}
	noteWalk := func(what, off string) {
		if off != "" {
			bookkeeping = append(bookkeeping, what+": "+off)
		}
	}
	bands, off := affordedOneLeafBand(names)
	noteWalk("the one-leaf band over this run's names", off)
	// Which step this run and the record are apart by, and in which direction,
	// is decided further down — by the NAMES rather than by how many of them
	// there are. See affordedMissingLeaves and the block that reads it: an
	// assertion against a band is only a proof while the bracketed population
	// is a member of the family the band is the maximum over, and "k apart by
	// count" does not say that — a leaf added and another renamed is +1 with no
	// subset anywhere.
	//
	// And the bands over the RECORD's own names, one per step, which are pure
	// functions of them and therefore measurable on every run — the same
	// argument the record's census re-derivation rests on, applied to the
	// record one level up. Each buys two things at once: the family a run that
	// many leaves SHORT of the record belongs to, and a reading on
	// affordedBandMeasuredOn that does not need core.Theme to have stood still.
	//
	// # And the lever that used to be here
	//
	// The two-leaf step was the one reading in this test with a cost worth
	// naming — about five seconds — so `-short` gave it up, and gave up the
	// widest reading on themeLeafSetOf with it. It is 38ms now, and the
	// three-leaf step that could not be taken at all is 112ms: the census of a
	// k-drop is the whole population's census corrected rather than a fresh
	// walk (affordedWhole), and the windows the correction puts back are
	// classified once for the walk rather than once per population
	// (affordedWhole.memoise). So the gate is gone and every step is taken on
	// every run, including the short ones.
	//
	// A lever is worth keeping when what it saves is worth the reading it
	// costs. A tenth of a second against the widest reading in the file is
	// not, and leaving it in would have meant a log line explaining that a
	// green `-short` run had skipped eighty thousand populations to save it.
	// # And the walk that used to be taken twice
	//
	// The two-leaf band and the sound chain bound both censused all 3160
	// two-drops of the record's names — the first reading each population
	// against the whole and the second against the two populations of n−1 it
	// can be reached from — and they did it in the same run, one after the
	// other, for about 450ms. Same populations, same censuses. So the census
	// is taken once and read three ways: see affordedTwoStepBands. What the
	// domination check between them holds is unchanged, because it was never
	// holding that the population had been walked twice.
	recordTwoLeaf, recordSecond, recordSound, twoStepOff :=
		affordedTwoStepBands(affordedMeasuredOn.names)
	noteWalk("the two-leaf and sound chain walk over the record's names", twoStepOff)
	recordStep := map[int]map[string]affordedBand{}
	for _, k := range affordedBandSteps {
		if k == 2 {
			recordStep[k] = recordTwoLeaf
			continue // taken above, off the same census the sound bound reads
		}
		step, off := affordedKLeafBand(affordedMeasuredOn.names, k)
		recordStep[k] = step
		noteWalk(fmt.Sprintf("the %s band over the record's names",
			affordedStepLeaves(k)), off)
	}
	// Named for the readings that still call it out by step: the two-leaf band
	// is what the chain bound and the sound bound are both held against, and
	// every other use goes through recordStep by the step in force.
	recordTwoBands := recordStep[2]

	// # The record's own bands, re-derived and held, one step at a time
	//
	// A band derived on every run cannot go stale, but the notes cite these
	// numbers and argue from them, and a number cited in prose and never
	// re-derived is a number that has already moved.
	//
	// # Over the RECORD's names, and therefore on every run
	//
	// This comparison used to be guarded on `len(names) == leaves`, which is
	// the same premise-by-count the exact arm carried: a leaf RENAMED keeps the
	// count and changes the strings, so the band this run measures is a band
	// over a different eighty populations and the comparison reports a re-sort
	// that is a rename. It fired that way the first time somebody tried it.
	//
	// The fix is the one the census re-derivation makes: the band is a function
	// of a list of names, the record carries the list, so the record's own band
	// is re-measurable whatever core.Theme has become. That removes the guard
	// entirely — this is asserted on every run, including the runs where
	// somebody is editing the struct.
	//
	// # And one loop rather than one block per step
	//
	// These were two blocks of three arms with the same shape and different
	// prose, and the second was written by copying the first. Every step's
	// population is affordedMeasuredOn's names with k of them dropped, so the
	// numbers that vary — how many populations, what a step is called, what a
	// drift here means that a smaller step would miss — are three sentences and
	// an integer. What the record no longer carries is a population of its own:
	// the leaves-and-window fields were a second copy of affordedMeasuredOn's
	// asserted equal to it, which is what a re-derivation over its names makes
	// unnecessary rather than merely redundant.
	for _, k := range affordedBandSteps {
		want, recorded := affordedBandMeasuredOn.step[k]
		if !recorded {
			t.Errorf("this file measures a band for a step of %s and "+
				"affordedBandMeasuredOn records none.\n\n"+
				"The band is re-derived on every run either way, so what is missing is "+
				"not the number — it is anything holding the number still. A step "+
				"measured and not recorded drifts silently, which is the state every "+
				"record in this file exists to replace. Take it from the log line.",
				affordedStepLeaves(k))
			continue
		}
		got := recordStep[k]
		for _, ending := range endings {
			w, known := want[ending]
			if !known {
				t.Errorf("affordedBandMeasuredOn carries no %d-leaf band for %q.\n\n"+
					"%s Re-take the band under the new wording, or leave the wording "+
					"alone. The record's own names measure it at %.4f losing and %.4f "+
					"gaining.",
					k, ending, affordedStepMissing(k),
					got[ending].losing, got[ending].gaining)
				continue
			}
			g := got[ending]
			if affordedBandRounded(g.losing) == w.losing &&
				affordedBandRounded(g.gaining) == w.gaining {
				continue
			}
			t.Errorf("a step of %s moves %q by %.4f losing and %.4f gaining "+
				"over affordedMeasuredOn's own %d names, and affordedBandMeasuredOn "+
				"records %.4f and %.4f.\n\n"+
				"Both numbers come off the same %d walks over the same %d populations "+
				"— the record's names with %s dropped in turn — and the walk is a "+
				"function of the names, so this comparison is exact and does not "+
				"depend on what core.Theme is today. A disagreement is themeLeafSetOf "+
				"sorting one of those populations differently than it did when the "+
				"band was taken, %s. The widest are %q losing and %q gaining.\n\n%s",
				affordedStepLeaves(k), ending, g.losing, g.gaining,
				affordedMeasuredOn.leaves,
				w.losing, w.gaining,
				affordedDropCount(affordedMeasuredOn.leaves, k),
				affordedDropCount(affordedMeasuredOn.leaves, k),
				affordedStepWord(k), affordedStepWider(k), g.losingAt, g.gainingAt,
				affordedStepRetake(k))
		}
		for key := range want {
			if !slices.Contains(endings, key) {
				t.Errorf("affordedBandMeasuredOn carries a %d-leaf band for %q and "+
					"the derivation has no such ending.\n\n"+
					"Every set reaches exactly one of the four sentences, so a fifth in "+
					"the band record is a band for an ending that was renamed or removed "+
					"— and its partner is an ending being read against the stated "+
					"fallback with nobody having decided that.", k, key)
			}
		}
	}
	for k := range affordedBandMeasuredOn.step {
		if !slices.Contains(affordedBandSteps, k) {
			t.Errorf("affordedBandMeasuredOn records a band for a step of %s "+
				"and nothing measures one.\n\n"+
				"A recorded band nothing re-derives is four numbers that cannot be "+
				"wrong, which is the shape this whole file replaces: it goes on "+
				"describing a walk after the walk has moved. Either measure the step "+
				"(add it to affordedBandSteps and read what C(%d,%d) costs first) or "+
				"drop the record.", affordedStepLeaves(k), affordedMeasuredOn.leaves, k)
		}
	}

	// # The drop census against the walk it stands in for
	//
	// Every band above is measured over populations whose census was NOT
	// walked: affordedWhole classifies the whole population once and corrects
	// that census per drop, which is what takes the two-leaf band from five
	// seconds to a fifth of one. It is the file's own hazard — a second way of
	// producing a number that already had one — and the defence is not an
	// argument about the index arithmetic. It is the direct walk, over
	// families nobody chose.
	//
	//	the record's own eighty   every one-drop of the record's names,
	//	one-drops                 censused both ways. The family the one-leaf
	//	                          band is measured over, so this is the walk
	//	                          those numbers would have come from.
	//	every drop of one to      over the first fourteen of this run's names,
	//	four names                which is where the arithmetic gets exercised
	//	                          at the sizes the band cannot afford to walk:
	//	                          C(20,4) is 4845 populations and C(80,4) is
	//	                          1.6 million. The bookkeeping is a function of
	//	                          the positions and the window bound and not of
	//	                          the names — that is the whole reason a
	//	                          smaller list says anything — and adjacent
	//	                          drops, drops at both ends and drops closer
	//	                          together than the window are all in it.
	//
	// Beyond these two: affordedBandMeasuredOn.step[2]'s eight numbers were
	// measured by the direct walk before any of this existed, and are
	// re-derived through the corrected census on every run.
	{
		// # And the base census over THIS run's names, which nothing held
		//
		// Every band this run measures over its own population is a correction
		// of affordedWholeOf(names)'s census, and that census was compared
		// with nothing. The record's was — twice, below — but the record's
		// names are the record's; a run that has edited core.Theme measures
		// its bands over a population whose base walk had no second reading at
		// all, and the base is what every correction starts from.
		//
		// It costs nothing to close, because the test has already walked those
		// sets itself: `reached` is the census of this run's population taken
		// set by set through affordedEachSet, and this is the same census taken
		// window by window through affordedWholeOf. Two walks of one
		// population, and the arms above have already held `reached` against
		// the record.
		//
		// Read into an array rather than compared as maps because an ending no
		// set reached is absent from `reached` and a 0 in the census — the same
		// corner affordedCensus exists to remove, and comparing the two shapes
		// directly would reintroduce it here.
		var walked affordedCensus
		for e, ending := range affordedEndingNames {
			walked[e] = reached[ending]
		}
		if got := affordedWholeOf(names, affordedWindowMax).census; got != walked {
			t.Errorf("this run's population classified per window comes to %v and "+
				"walked as sets it comes to %v.\n\n"+
				"affordedWholeOf takes the same windows in the same order and mounts "+
				"them the same two ways, and every band this run measures over its "+
				"own names is a correction of the first of those. A disagreement "+
				"before a single leaf has been dropped is the two walks having come "+
				"apart at the base — so the band the assertion arms read is a maximum "+
				"over populations that are corrections of a census of nothing, while "+
				"the census those arms compare against the record is the other walk "+
				"and is fine.",
				affordedCensusMap(got), affordedCensusMap(walked))
		}
		recordWhole := affordedWholeOf(affordedMeasuredOn.names, affordedMeasuredOn.window)
		recordWhole.memoise(1)
		sc := affordedScratchFor(recordWhole)
		var corrected affordedCensus
		if direct := affordedCensusOf(affordedMeasuredOn.names,
			affordedMeasuredOn.window); recordWhole.census != direct {
			t.Errorf("the record's population classified per window comes to %v and "+
				"walked as sets it comes to %v.\n\n"+
				"affordedWholeOf takes the same windows in the same order and mounts "+
				"them the same two ways; a disagreement before a single leaf has been "+
				"dropped is the two walks having come apart at the base, and every "+
				"band in this file is a correction of the first one.",
				affordedCensusMap(recordWhole.census), affordedCensusMap(direct))
		}
		for d := range affordedMeasuredOn.names {
			recordWhole.censusDropping([]int{d}, sc, &corrected)
			smaller := make([]string, 0, len(affordedMeasuredOn.names)-1)
			for i, name := range affordedMeasuredOn.names {
				if i != d {
					smaller = append(smaller, name)
				}
			}
			direct := affordedCensusOf(smaller, affordedMeasuredOn.window)
			if corrected == direct {
				continue
			}
			t.Errorf("dropping %q from the record's names gives %v when the whole "+
				"population's census is corrected for it and %v when the smaller "+
				"population is walked.\n\n"+
				"These are the same population reached two ways, and the corrected "+
				"one is what every band in this file is measured with — the direct "+
				"walk is here only to hold it. A disagreement is either a window "+
				"taken out that should have stayed, or one put in that was already "+
				"counted, and the census it produces is of no population at all.",
				affordedMeasuredOn.names[d], affordedCensusMap(corrected),
				affordedCensusMap(direct))
			break // one is the finding; eighty of them is the same finding
		}
		// And whether the table of window shapes covered them. A miss is the
		// right answer reached the slow way, so this is a reading about the
		// enumeration rather than about the census — and it is the sort of
		// claim that goes quiet, which is why it is carried out to the
		// bookkeeping arm with the walks' own.
		if sc.missed > 0 {
			noteWalk("the one-drop comparison over the record's names",
				fmt.Sprintf("%d of the windows a drop put back were not in the table "+
					"of every shape a one-name drop can produce", sc.missed))
		}
	}
	{
		// Fourteen: twice the window bound plus two, which is the shortest
		// list in which a drop can be adjacent to another, at either end, and
		// further apart than the widest window — the three shapes the
		// bookkeeping has separate arms for. Twenty would take five seconds
		// and say the same thing, because what is being held does not depend
		// on how many names there are beyond that.
		short := names
		if len(short) > 2*affordedWindowMax+2 {
			short = short[:2*affordedWindowMax+2]
		}
		whole := affordedWholeOf(short, affordedWindowMax)
		whole.memoise(4)
		sc := affordedScratchFor(whole)
		var corrected affordedCensus
		smaller := make([]string, 0, len(short))
		told := false
		for k := 1; k <= 4 && k < len(short); k++ {
			drop := make([]int, k)
			var walk func(pos, from int)
			walk = func(pos, from int) {
				if told {
					return
				}
				if pos == k {
					whole.censusDropping(drop, sc, &corrected)
					smaller = smaller[:0]
					at := 0
					for i, name := range short {
						if at < k && drop[at] == i {
							at++
							continue
						}
						smaller = append(smaller, name)
					}
					direct := affordedCensusOf(smaller, affordedWindowMax)
					if corrected == direct {
						return
					}
					told = true
					t.Errorf("over the first %d of this run's names, dropping the %d "+
						"at %v gives %v corrected and %v walked.\n\n"+
						"The correction is what the bands are measured with and the "+
						"walk is what they claim to be. This list is short so that "+
						"every drop of one to four names can be taken — %d of them at "+
						"this size — which is the arithmetic at the step sizes the "+
						"real family cannot afford: C(%d,%d) is %d populations. What "+
						"is being held is the window bookkeeping, and that is a "+
						"function of the positions and the window bound rather than "+
						"of the names, which is why a shorter list is evidence about "+
						"the longer one.",
						len(short), k, drop, affordedCensusMap(corrected),
						affordedCensusMap(direct),
						affordedDropCount(len(short), k), affordedMeasuredOn.leaves, k,
						affordedDropCount(affordedMeasuredOn.leaves, k))
					return
				}
				for i := from; i <= len(short)-(k-pos); i++ {
					drop[pos] = i
					walk(pos+1, i+1)
				}
			}
			walk(0, 0)
		}
		if sc.missed > 0 {
			noteWalk("the one-to-four-drop comparison over the first names",
				fmt.Sprintf("%d of the windows a drop put back were not in the table "+
					"of every shape a drop of up to four names can produce", sc.missed))
		}
	}
	// # And the mounting the corrected census reaches by index
	//
	// The windows it puts back are not contiguous in the list they came from,
	// so they are gathered rather than sub-sliced — a second way of applying
	// the one rule affordedParent states. It needs its own reading for the
	// reason the arrangement arm above exists: a set with the two parents
	// swapped is the mirror of this one, every ending is the same, and no
	// census, band or record in this file can tell them apart. The rule is
	// held here and the outcome is held there, and neither would notice the
	// other going wrong.
	//
	// Over every index list of one to six positions drawn from the same
	// fourteen names — 6475 of them, which is every window shape the gather
	// can ever be asked for at this window bound.
	{
		short := names
		if len(short) > 2*affordedWindowMax+2 {
			short = short[:2*affordedWindowMax+2]
		}
		mount := affordedPrefixOf(short)
		one := make([]string, 0, affordedWindowMax)
		two := make([]string, 0, affordedWindowMax)
		at := make([]int, affordedWindowMax)
		lists, told := 0, false
		for width := 1; width <= affordedWindowMax && width <= len(short); width++ {
			var walk func(pos, from int)
			walk = func(pos, from int) {
				if told {
					return
				}
				if pos == width {
					lists++
					gotOne, gotTwo := affordedMountAt(mount, at[:width], one, two)
					for j := 0; j < width; j++ {
						wantOne := "One." + short[at[j]]
						wantTwo := affordedParent(j) + short[at[j]]
						if gotOne[j] == wantOne && gotTwo[j] == wantTwo {
							continue
						}
						told = true
						t.Errorf("mounting the leaves at %v puts %q and %q at "+
							"position %d, where the rule says %q and %q.\n\n"+
							"affordedParent is the whole of the second "+
							"arrangement's rule and this is the gather that "+
							"applies it to a window the drop census had to put "+
							"back. Nothing downstream can see it go wrong: the "+
							"mirror set is the same two groups under swapped "+
							"names, so every ending, every band and every record "+
							"here would go on holding over a population that is "+
							"not the one the record was taken over.",
							at[:width], gotOne[j], gotTwo[j], j, wantOne, wantTwo)
						return
					}
					return
				}
				for i := from; i <= len(short)-(width-pos); i++ {
					at[pos] = i
					walk(pos+1, i+1)
				}
			}
			walk(0, 0)
		}
		if lists == 0 {
			t.Errorf("no index list was mounted at all, so the gather the drop " +
				"census puts its windows back with is held by nothing.")
		}
	}

	if len(bookkeeping) > 0 {
		t.Errorf("%d of the walks above took windows out and put windows back in "+
			"numbers that do not add up:\n\t%s\n\n"+
			"A population of n−k names has a fixed number of windows fewer than one "+
			"of n, whatever k names were dropped, so what a drop census removes "+
			"less what it adds is that difference and nothing else. A walk where it "+
			"is not has classified some window twice or none at all, and the band "+
			"it produced is a maximum over populations that are not the ones it "+
			"names. This is the arithmetic the two comparisons above hold for k up "+
			"to four; it is asserted on every population of every walk because "+
			"those two cannot be.",
			len(bookkeeping), strings.Join(bookkeeping, "\n\t"))
	}

	// # The scales the composition below rests on, which is one multiply
	//
	// Dropping two leaves is two one-leaf drops, and the residuals compose
	// because the SCALES do: S(n−2)/S(n) is S(n−1)/S(n) times S(n−2)/S(n−1),
	// with the intermediate count cancelling. That is exact in the rationals.
	// It is not exact in floats — three divisions and a multiply are four
	// roundings against one — and the chain bound is compared against a band
	// whose residuals divided by the direct spelling, so the two have to be the
	// same number to the precision anything here reads.
	//
	// Asserted rather than assumed for the reason the FMA hazard taught: this
	// file has already been caught once by two roundings of one number, and it
	// was found in a break-test's odd output rather than by anything looking.
	//
	// # And over every population size rather than the one in force
	//
	// The tolerance used to be a stated 1e-12 with one number behind it — the
	// difference at n = 80, on one machine, called "one ulp" in a comment.
	// affordedComposeSpread is the population: every size from three to the
	// record's, which is where the small-population edge would show up if the
	// error grew as the counts got smaller. It does show up there — the worst
	// is at five names and not at eighty — and it is still one ulp, so what
	// the measurement says is that the size does not matter. The constant used
	// to assume that from one reading at one size.
	{
		worst, at := affordedComposeSpread(affordedMeasuredOn.leaves,
			affordedMeasuredOn.window)
		if worst > affordedScaleCompose {
			t.Errorf("over the %d population sizes this walk can produce, the "+
				"two-leaf scale taken directly and taken as one leaf then another "+
				"differ by as much as %g — at %d names — against a tolerance of "+
				"%g.\n\n"+
				"Those are the same rational number: the set count in the middle "+
				"cancels. The chain bound multiplies the one-leaf bands on exactly "+
				"that identity, and the band it is compared against took its "+
				"residuals against the direct spelling, so a difference here is the "+
				"two of them dividing by different numbers and every comparison "+
				"between them being between two readings rather than one. At this "+
				"size the difference is arithmetic that has stopped being a rounding.",
				affordedMeasuredOn.leaves-2, worst, at, affordedScaleCompose)
		}
		// And the record, which is what makes the tolerance a bracket over a
		// measurement rather than a number somebody liked. Held to a decade
		// rather than exactly: this is the one recorded number here that is
		// not a count or a rounded band, and the multiply that produces it is
		// the one the compiler is free to fuse on one machine and not another.
		if worst > affordedComposeMeasuredOn.worst*affordedComposeDrift ||
			worst*affordedComposeDrift < affordedComposeMeasuredOn.worst {
			t.Errorf("the worst spread between the two spellings of a two-leaf "+
				"scale is %g on this run, at %d names, and affordedComposeMeasuredOn "+
				"records %g at %d — more than %.0fx apart.\n\n"+
				"A rounding of a division does not move by a decade. What does move "+
				"it is the arithmetic changing shape: a set count that is no longer "+
				"the sum it was, a ratio taken in a different order, or a multiply "+
				"the compiler has started fusing with the subtract that reads it. "+
				"The tolerance above is a bracket over THIS number and would not "+
				"notice any of them, which is why the measurement is recorded and "+
				"not only bracketed.",
				worst, at, affordedComposeMeasuredOn.worst,
				affordedComposeMeasuredOn.at, affordedComposeDrift)
		}
		// And the margin, which is the whole of what makes 1e-12 a bracket:
		// four decades above a rounding and far under anything that is not one.
		if worst > 0 &&
			affordedScaleCompose < worst*affordedComposeMeasuredOn.margin {
			t.Errorf("affordedScaleCompose is %g and the worst measured spread is "+
				"%g, which leaves less than the %.0fx this file asks of a bracket "+
				"over a measurement.\n\n"+
				"A tolerance that sits just above what it brackets is a tolerance "+
				"that will fire on the next machine rather than on the next mistake. "+
				"Either the spread has grown — read the arm above, which says by how "+
				"much — or the constant was tightened without the measurement being "+
				"re-read.",
				affordedScaleCompose, worst, affordedComposeMeasuredOn.margin)
		}
	}

	// Which of the eight composed numbers came in UNDER the measured band,
	// which is the finding the composition exists here to carry. Empty under
	// -short, where nothing was composed and the log line says so.
	chainSaid := []string{}
	// And how far the SOUND composition sits above the band it has to
	// dominate, which is the looseness that keeps it from being useful.
	soundSaid := []string{}
	// # And the composition that was supposed to reach past the enumerable
	// steps
	//
	// The steps stop somewhere, and wherever that is, the obvious way past it
	// is not to enumerate at all: dropping k leaves is k one-leaf drops, the
	// residuals multiply exactly (see affordedChainBoundOf), so ∏(1+b_i)−1 is
	// a bound wherever the b_i bound each step — and those are one-leaf bands
	// at n, n−1, …, which is k×80 walks rather than C(80,k).
	//
	// It does not work, and k = 2 is the step where the composed number and a
	// measured one can be put side by side most cheaply. Both are re-derived
	// here and the comparison is the finding: the arm below fires when the
	// composition covers all eight, because covering is what would make it
	// usable and today it does not.
	//
	// The construction has since been overtaken rather than merely rejected.
	// Its whole attraction was cost, and the direct measurement is now cheaper
	// at every k this file takes — 240 walks against 112ms of corrected
	// censuses — so even a version that covered would be the slower way to a
	// number the walk already has. The comparison is kept because what it
	// holds is the SHAPE: two drops are two steps, and a run where the product
	// suddenly covers is a run where the drift changed or the measurement
	// shrank, and either is worth knowing.
	chain, chainOff := affordedChainBoundOf(affordedMeasuredOn.names, 2)
	noteWalk("the chain bound over the record's names", chainOff)
	{
		for _, ending := range endings {
			w, known := affordedBandMeasuredOn.chain[ending]
			if !known {
				t.Errorf("affordedBandMeasuredOn carries no chain bound for %q.\n\n"+
					"The chain is the construction this file rejected in favour of "+
					"measuring the step, and the rejection is these numbers rather than "+
					"the sentence about them. An ending with no recorded composition is "+
					"an ending the rejection is not held over. This run composes it to "+
					"%.4f losing and %.4f gaining, against a measured %.4f and %.4f.",
					ending, chain[ending].losing, chain[ending].gaining,
					recordTwoBands[ending].losing, recordTwoBands[ending].gaining)
				continue
			}
			g := chain[ending]
			if affordedBandRounded(g.losing) == w.losing &&
				affordedBandRounded(g.gaining) == w.gaining {
				continue
			}
			t.Errorf("the one-leaf band composes to %.4f losing and %.4f gaining for "+
				"%q over a two-leaf step, and affordedBandMeasuredOn.chain records "+
				"%.4f and %.4f.\n\n"+
				"The composition walks one chain of populations — the record's names, "+
				"then the record's names with the first dropped — so this number moves "+
				"when either of those two one-leaf bands moves, and the %d-leaf band "+
				"above may not have moved with it. The widest at each step were %q "+
				"losing and %q gaining. Re-take the composition and the two-leaf band "+
				"together: what the pair is for is the comparison between them.",
				g.losing, g.gaining, ending, w.losing, w.gaining, 2,
				g.losingAt, g.gainingAt)
		}
		for key := range affordedBandMeasuredOn.chain {
			if !slices.Contains(endings, key) {
				t.Errorf("affordedBandMeasuredOn.chain carries a composition for %q "+
					"and the derivation has no such ending.\n\n"+
					"Its partner is an ending whose composed bound is not being compared "+
					"with anything, which is the rejection above going quiet for one of "+
					"the four.", key)
			}
		}
		// # And the same composition over every chain, which is a proof
		//
		// affordedChainBoundOf walks one chain and its failure to cover has
		// two possible causes: the representative population was unlucky, or
		// the product is the wrong shape. This separates them, and the answer
		// is the first — with b_i taken over ALL intermediate populations the
		// product does cover, every one of the eight.
		//
		// Which turns it from a finding into a check. The sound bound
		// dominates every two-drop residual by construction, so it dominates
		// the measured two-leaf band by construction, and the two walks share
		// nothing but the classification: one takes 3160 populations of n−2
		// against the whole, the other takes each of them against the two
		// populations of n−1 it can be reached from. A run where the bound
		// does NOT cover is one of the two being wrong, and this arm says so
		// in the direction the arithmetic guarantees — which is the opposite
		// direction from the cheap chain's arm above, because one is a
		// heuristic that was tried and the other is a theorem.
		//
		// It does not rescue the construction. The sound bound is 1.85x the
		// measurement on the ending that matters, and it is not cheaper at any
		// k: bounding the LAST step over every chain needs a census of every
		// population of size n−k, which is the family the direct measurement
		// walks. There is no k at which the chain gets there first.
		sound, second := recordSound, recordSecond
		soundSaid = make([]string, 0, len(endings))
		for _, ending := range endings {
			w, known := affordedBandMeasuredOn.sound[ending]
			if !known {
				t.Errorf("affordedBandMeasuredOn carries no sound chain bound for "+
					"%q.\n\n"+
					"That number is the one that says the cheap chain fails because "+
					"its representative population is one of eighty and not because "+
					"the product is the wrong shape — and it is the check that the "+
					"two-leaf band and the chain of one-leaf bands describe the same "+
					"walk. This run composes it to %.4f losing and %.4f gaining.",
					ending, sound[ending].losing, sound[ending].gaining)
				continue
			}
			g := sound[ending]
			if affordedBandRounded(g.losing) != w.losing ||
				affordedBandRounded(g.gaining) != w.gaining {
				t.Errorf("the one-leaf bands compose over every chain to %.4f losing "+
					"and %.4f gaining for %q, and affordedBandMeasuredOn.sound "+
					"records %.4f and %.4f.\n\n"+
					"This is the widest walk in the file: 80 populations of %d and "+
					"%d of %d, every one of the latter read against both of the "+
					"former it can be reached from. The widest chains were %q losing "+
					"and %q gaining. A number that moved here moved in the one-leaf "+
					"band, in the second step, or in themeLeafSetOf, and the arm "+
					"below says which by whether the bound still covers.",
					g.losing, g.gaining, ending, w.losing, w.gaining,
					affordedMeasuredOn.leaves-1,
					affordedDropCount(affordedMeasuredOn.leaves, 2),
					affordedMeasuredOn.leaves-2, g.losingAt, g.gainingAt)
			}
			m := recordTwoBands[ending]
			// `band` and not `measured`, which is what this field was called:
			// affordedBracket declares `measured` as an int and this
			// package's float census reads a field's type off the name
			// alone, so the two spellings conflated every `.measured` in the
			// package into one answer. See affordedFloatSource — a local
			// table's field name is free to move and the conflation is not
			// free, so it moves.
			for _, d := range []struct {
				what        string
				bound, band float64
				at          string
			}{
				{"losing", g.losing, m.losing, second[ending].losingAt},
				{"gaining", g.gaining, m.gaining, second[ending].gainingAt},
			} {
				if affordedBandRounded(d.bound) >= affordedBandRounded(d.band) {
					soundSaid = append(soundSaid, fmt.Sprintf("%q %s %.2f%% over a "+
						"measured %.2f%% (%.2fx)", ending, d.what,
						affordedPercent(d.bound), affordedPercent(d.band),
						affordedStepRatio(d.bound, d.band)))
					continue
				}
				t.Errorf("the sound chain bound for %q %s is %.4f and the measured "+
					"two-leaf band is %.4f, which is larger.\n\n"+
					"That cannot happen. Every two-drop is two one-leaf steps, the "+
					"residuals multiply exactly, and the sound bound multiplies the "+
					"LARGEST residual available at each step — over all %d "+
					"populations at the first and every chain into all %d at the "+
					"second. So it dominates every member of the family the measured "+
					"band is the maximum of, this one included.\n\n"+
					"The two walks share only themeLeafSetOf: one reads each "+
					"population of %d against the whole, the other reads it against "+
					"the two populations of %d it can be reached from. A "+
					"disagreement is a census that is not of the population it says "+
					"— the drop bookkeeping, the scales, or the composition — and it "+
					"is not a fact about core.Theme. The widest second step was %q.",
					ending, d.what, d.bound, d.band, affordedMeasuredOn.leaves,
					affordedDropCount(affordedMeasuredOn.leaves, 2),
					affordedMeasuredOn.leaves-2, affordedMeasuredOn.leaves-1, d.at)
			}
		}
		for key := range affordedBandMeasuredOn.sound {
			if !slices.Contains(endings, key) {
				t.Errorf("affordedBandMeasuredOn.sound carries a composition for %q "+
					"and the derivation has no such ending.\n\n"+
					"Its partner is an ending whose composed bound is not being "+
					"checked against the band it has to dominate.", key)
			}
		}

		// And the finding itself, which is a comparison and not a number.
		//
		// Rounded on both sides, because both are recorded at four decimal
		// places and an ORDERING between two independently derived floats is
		// the FMA hazard's own shape: the small ending's two numbers agree to
		// twelve places and differ in the last bit, and calling that "under"
		// would be reporting a rounding as a finding.
		under := []string{}
		for _, ending := range endings {
			g, m := chain[ending], recordTwoBands[ending]
			// `composed` and `band` rather than `chain` and `measured`, for
			// the reason the table above carries: affordedBandMeasuredOn
			// declares `chain` as a map and affordedBracket declares
			// `measured` as an int, and the float census's field table is
			// keyed by the name with no struct in front of it.
			for _, d := range []struct {
				what           string
				composed, band float64
			}{
				{"losing", g.losing, m.losing},
				{"gaining", g.gaining, m.gaining},
			} {
				if affordedBandRounded(d.composed) >= affordedBandRounded(d.band) {
					continue
				}
				under = append(under, fmt.Sprintf(
					"%q %s composes to %.2f%% against a measured %.2f%%",
					ending, d.what, affordedPercent(d.composed), affordedPercent(d.band)))
			}
		}
		if len(under) == 0 {
			t.Errorf("the composed one-leaf bound covers all eight of the measured "+
				"two-leaf figures on this run, and this file's reason for measuring a "+
				"step rather than composing it is that it does not.\n\n"+
				"The identity is exact — two drops are two steps and the residuals "+
				"multiply — and the bound is not, because the b_i it multiplies are "+
				"measured over ONE representative population per size while a two-drop "+
				"can step out of any of them. What said so was a number: the crowding "+
				"ending's gaining figure composed to 77.55%% against a measured "+
				"87.88%%, so a chain bound would have called an honest two-field edit "+
				"a re-sort — the same failure `band × k` has and the stated tenth had "+
				"before it.\n\n"+
				"With all eight covered that argument is no longer visible in the "+
				"numbers, and it is worth finding out which happened: the drift "+
				"changed shape, or the two-leaf measurement shrank. If the "+
				"composition really does cover, it is the only route this file has to "+
				"a band at k = 3 — C(%d,3) is %d walks and a chain is 240 — and this "+
				"is where somebody would find that out. Composed %v against measured "+
				"%v.",
				affordedMeasuredOn.leaves,
				affordedDropCount(affordedMeasuredOn.leaves, 3), chain, recordTwoBands)
		}
		chainSaid = under
	}

	// # And the arms that run when somebody has just edited core.Theme
	//
	// The exact comparison above is the strongest reading in this file and it
	// runs only while the population is untouched — which is the state a
	// re-measure leaves and NOT the state anybody editing core.Theme is in. The
	// moment the struct gains or loses a leaf that arm goes silent and what is
	// left is four floors that cannot see a re-sort by construction, which is
	// the state affordedTakers and the exact arm were both written because of.
	//
	// An edit of k fields is the one move where the gap can be closed without a
	// tolerance anybody chose, and the reason is the band's construction. A
	// k-leaf band drops every k of a population's names in turn, so:
	//
	//	this run is k LONG    the record's own population is this run's names
	//	                      with the k added leaves dropped — a member of the
	//	                      family the band over THIS run's names is the
	//	                      maximum of.
	//	this run is k SHORT   this run's population is the record's names with
	//	                      the k removed leaves dropped — a member of the
	//	                      family the band over the RECORD's names is the
	//	                      maximum of. This run cannot walk that family from
	//	                      its own names; it is walkable because the record
	//	                      carries the names it was taken over, which is what
	//	                      the removal reading had no way to do before.
	//
	// Not "about the same size as": one of them. So a residual over the band is
	// not a population that moved unusually far; it is arithmetic that cannot
	// happen while themeLeafSetOf sorts the walk the way it did.
	//
	// # And the premise is checked by NAME
	//
	// `len(names) == leaves+k` is true of a run that added k leaves and renamed
	// another, where the record's population is not this run's names minus
	// anything and the band is a bound over the wrong family — a tolerance
	// wearing a proof's clothes. affordedMissingLeaves establishes the subset
	// rather than the size; see its note for why the counting argument holds.
	//
	// # One loop rather than a block per step
	//
	// These were four blocks — long and short, at one leaf and at two — with
	// the same eight lines of arithmetic and four sets of prose, and the second
	// pair was written by copying the first. What actually varies is the band,
	// the population it was measured over, and the names that moved; everything
	// else is the same sentence with a different number in it. So the step is
	// looked up rather than spelled out, and adding one to affordedBandSteps
	// adds its pair of arms with it — which is how the three-leaf step arrived
	// with no new arms written for it at all.
	//
	// At most one step can match: the size difference decides which, and a run
	// whose names are the record's exactly is handled by the exact arm above.
	stand, standLong := 0, false
	standMoved := []string{}
	for _, k := range affordedBandSteps {
		if moved, ok := affordedMissingLeaves(names, affordedMeasuredOn.names); ok &&
			len(moved) == k {
			stand, standLong, standMoved = k, true, moved
			break
		}
		if moved, ok := affordedMissingLeaves(affordedMeasuredOn.names, names); ok &&
			len(moved) == k {
			stand, standLong, standMoved = k, false, moved
			break
		}
	}
	// The band that family is the maximum of, and what it was measured over.
	// Nil when this run is not a step away from the record by name, or when the
	// window bound has moved — the band is over windows of one to
	// affordedWindowMax consecutive names, so a run at a different bound is not
	// walking the family at all.
	var standBands map[string]affordedBand
	standOver, standWhose := 0, ""
	if stand > 0 && affordedWindowMax == affordedMeasuredOn.window {
		if standLong {
			// Drops of `stand` from THIS run's names, one of which is the
			// record's population. The one-leaf band is already in hand;
			// anything wider is measured here, because it is a band about this
			// run and nothing else reads it.
			standBands = bands
			if stand != 1 {
				var off string
				standBands, off = affordedKLeafBand(names, stand)
				noteWalk(fmt.Sprintf("the %s band over this run's names",
					affordedStepLeaves(stand)), off)
			}
			standOver, standWhose =
				affordedDropCount(len(names), stand), "this run's names"
		} else {
			standBands = recordStep[stand]
			standOver, standWhose = affordedDropCount(affordedMeasuredOn.leaves, stand),
				"the record's names"
		}
	}
	if standBands != nil {
		// Said in the direction the edit actually went, because the two are
		// different edits and a reader checking the claim has to walk the
		// subset the right way round.
		way, whose := "lost", "this run's names are the record's"
		if standLong {
			way, whose = "gained", "the record's names are this run's"
		}
		for _, ending := range endings {
			measured, known := affordedMeasuredOn.ending[ending]
			if !known {
				continue // reported by the key-set arms above
			}
			off, predicted, ok := affordedResidualOf(measured, reached[ending], scale)
			if !ok {
				continue
			}
			band, why := affordedBandFor(standBands, stand, standOver, standWhose,
				ending, standLong)
			if off <= band {
				continue
			}
			t.Errorf("%q was reached by %d of the %d sets, and a population that had "+
				"only %s %s predicts about %.0f — %.1f%% out, against a band of "+
				"%.2f%%: %s.\n\n"+
				"This run has %d leaf names and affordedMeasuredOn was taken over %d "+
				"at the same window, and %s with %s dropped — which is exactly one of "+
				"the %d populations that band was measured over. The residual above is "+
				"therefore a member of the family the band is the largest of, and it "+
				"cannot exceed it while themeLeafSetOf sorts the walk the way it did "+
				"when the record was taken. The whole census is %v against a record "+
				"of %v.\n\n"+
				"So this is the re-sort the per-ending floors cannot see, arriving on "+
				"the population an edit to core.Theme actually leaves. Note that the "+
				"band is MEASURED for a step of %s and is not the one-leaf band times "+
				"%d: the drift is not linear in the number of leaves moved, and in the "+
				"gaining direction it is worse than linear.\n\n%s",
				ending, reached[ending], len(sets), way, affordedStepLeaves(stand),
				predicted, affordedPercent(off), affordedPercent(band), why,
				len(names), affordedMeasuredOn.leaves, whose,
				affordedNameList(standMoved), standOver,
				reached, affordedMeasuredOn.ending, affordedStepLeaves(stand), stand,
				affordedStepRetake(stand))
		}
	}

	// # The share's two edges, and which endings it is doing work for
	//
	// See affordedEndingShare. A third is a fraction of one population and
	// what makes it a measurement rather than a taste is the pair it has to
	// separate — both in hand here, and neither protected by max(constant,
	// measured/3).
	//
	// The first edge is the population that MOVED. Every ending is scaled by
	// affordedScale, so a merely-moved ending scores about `measured × scale`;
	// a floor above that is a floor no healthy census can clear, and the next
	// failure it produces is a re-measure wearing a collapse's message. The
	// second is the derivation itself: an ending whose share falls under the
	// stated constant is held to a number that is not a measurement of it, and
	// when that is true of all four the per-ending floor has quietly become the
	// one constant for four endings it was written to replace.
	onShare, brackets := 0, map[string]affordedBracket{}
	for _, ending := range endings {
		b := affordedEndingBracket(ending)
		brackets[ending] = b
		if b.share {
			onShare++
		}
	}
	if onShare == 0 {
		t.Errorf("not one of the four endings is held to a share of its own "+
			"measurement: every floor here is the stated %d.\n\n"+
			"A third of an ending's own count is what brackets a large arm — 473 "+
			"sets losing four fifths of themselves clears a constant of %d without a "+
			"word — and the constant is what keeps a small one from sliding to 1. "+
			"With every share under the constant this is one number holding four "+
			"endings, which is the state affordedMeasuredOn was recorded to replace, "+
			"arriving back through the arithmetic rather than through an edit. The "+
			"record reads %v against a population of %d sets%s",
			affordedEndingFloor, affordedEndingFloor, affordedMeasuredOn.ending,
			len(sets), affordedMeasuredNote(len(names), len(sets)))
	}
	for _, ending := range endings {
		b := brackets[ending]
		if !b.known || scale == 0 {
			continue // reported by the key-set arms above
		}
		// The same number affordedShortfallCause reads, through the same
		// function, and read the other way. See affordedPrediction.
		moved, _ := affordedPrediction(ending, scale)
		if moved >= float64(b.floor) {
			continue
		}
		t.Errorf("this ending's floor is %d and a population that had only MOVED "+
			"would put %q at about %.0f — under its own floor before any relation "+
			"is asked.\n\n"+
			"The census is at %.2f× the population affordedMeasuredOn was taken "+
			"over (%d sets then, %d now), and the floor is %s. The two findings "+
			"this bracket exists to separate are a struct that changed size and an "+
			"arm that went; at this scale the first of them trips the second's "+
			"message, so the next failure here says a relation has stopped being "+
			"asked when what happened is that somebody narrowed a window or "+
			"core.Theme lost leaves. Re-take affordedMeasuredOn against this walk — "+
			"the census in the log line is the new measurement.",
			b.floor, ending, moved, scale,
			affordedSetCount(affordedMeasuredOn.leaves, affordedMeasuredOn.window),
			len(sets), b.why)
	}
	// And what each ending was actually held to, which arm set it, and how much
	// room there was. A passing run's whole evidence that the per-ending
	// derivation is doing anything is this line: "against its own floor" was
	// true of an ending on the stated constant too.
	held := make([]string, 0, len(endings))
	// And how far the four are from where a moved population puts them, which
	// is the reading that says the SORT has not moved.
	//
	// The floors above are about one ending at a time and they pass over a
	// redistribution that stayed above them — the case affordedTakers exists
	// for. This is that case in a passing run's own line: the residual is what
	// is left of an ending's count after the scale is taken off, so four
	// residuals near zero is the classification standing still, and a pair of
	// them at ±40% with the population unchanged is themeLeafSetOf having
	// re-sorted the walk with every floor still cleared.
	// Started under zero rather than at it, so the first ending always names
	// itself: on an unmoved walk every residual is exactly 0 and a strict
	// comparison would leave the sentence with no noun.
	worst, worstEnding := -1.0, ""
	// And the endings that are outside the band a single leaf is worth to
	// THEM, which is the per-ending version of the same reading. See
	// affordedOneLeafBand: a tenth is three times the drift of the large
	// endings and a third of the drift of the small ones, so one number here
	// either misses a re-sort of 473 sets or reports a standing-still walk as
	// a moved one, depending which ending it is asked about.
	gaining := len(names) > affordedMeasuredOn.leaves
	// And whose names that band was measured over, which is whatever the arm
	// above asserted against. A run k leaves SHORT of the record is bracketed
	// by the k-leaf band over the RECORD's names and a run k LONG by the band
	// over this run's — that is the family each belongs to — so the sentence
	// reads the same numbers the assertion did, or a green run would be
	// reporting a different bound from the one in force. This used to be two
	// special cases and a default, and the two-leaf LONG case fell through to
	// the one-leaf default: the arm asserted against a band the line did not
	// print.
	bandsFor, bandOver, bandWhose := bands, len(names), "this run's names"
	bandStep := 1
	if standBands != nil {
		bandsFor, bandOver, bandWhose, bandStep =
			standBands, standOver, standWhose, stand
		gaining = standLong
	}
	outside := []string{}
	for _, ending := range endings {
		b := brackets[ending]
		arm := "the stated floor"
		if b.share {
			arm = "a third of its own"
		}
		held = append(held, fmt.Sprintf("%q %d/%d (%s)",
			ending, reached[ending], b.floor, arm))
		// Through affordedResidualOf rather than spelled out here, and the
		// reason is not tidiness. This line used to compute the residual
		// inline while the band computed it in that function, and on arm64 the
		// two disagreed in the last bit — Go may fuse a multiply and the
		// subtract that follows it into one rounding, and it did so at one of
		// the two spellings and not the other. The band is the MAXIMUM over a
		// family this number is a member of, so a difference of one ulp is the
		// whole distance between "inside its band" and a green run reporting a
		// re-sort that did not happen.
		off, _, ok := affordedResidualOf(b.measured, reached[ending], scale)
		if !ok {
			continue
		}
		if off > worst {
			worst, worstEnding = off, ending
		}
		if band, why := affordedBandFor(
			bandsFor, bandStep, bandOver, bandWhose, ending, gaining); off > band {
			outside = append(outside, fmt.Sprintf("%q is %.1f%% out against %s",
				ending, affordedPercent(off), why))
		}
	}
	// Said in the direction the residuals are actually in. "Re-sorted by
	// nothing" is a claim, and a run where an ending is half again its own
	// prediction while every floor is still cleared is the redistribution
	// affordedTakers is about, passing — which is worth a sentence in a green
	// run's line rather than the same words that describe a walk standing
	// still. The floors are one ending at a time and cannot see it.
	resorted := "every ending inside the band a single leaf is worth to it"
	if affordedPopulationUnchanged(scale) {
		resorted = "the walk having been re-sorted by nothing"
	}
	if len(outside) > 0 {
		resorted = fmt.Sprintf("outside what a one-leaf change moves them, so some "+
			"ending is holding sets another one used to reach — every floor is still "+
			"clear and themeLeafSetOf has moved under them: %s",
			strings.Join(outside, "; "))
	}
	// # And which of the three readings actually ran
	//
	// The three are not equally strong and only one of them runs on any given
	// population, so a green run that does not say which is a green run whose
	// evidence a reader has to reconstruct from the leaf count. The strongest
	// is silent exactly when somebody is editing core.Theme, which is when it
	// would be read — see the arms above.
	distance := len(names) - affordedMeasuredOn.leaves
	// How many of the record's names this run does not have, for the branch
	// where the two are one apart by count and not by name.
	strangers := 0
	{
		here := make(map[string]bool, len(names))
		for _, name := range names {
			here[name] = true
		}
		for _, name := range affordedMeasuredOn.names {
			if !here[name] {
				strangers++
			}
		}
	}
	reading := ""
	switch {
	case affordedWindowMax != affordedMeasuredOn.window:
		reading = fmt.Sprintf("only the four floors and the sentence above: the "+
			"window is %d against the record's %d, so neither the exact comparison "+
			"nor the one-leaf band is about this population",
			affordedWindowMax, affordedMeasuredOn.window)
	case sameNames:
		reading = "the census against the record EXACTLY, ending by ending — same " +
			"names, same window, so which ending a set reaches is the only thing " +
			"left that can move"
	case standBands != nil && standLong:
		reading = fmt.Sprintf("the census against the record within the band "+
			"ADDING %s is worth to each ending, asserted — the record's own "+
			"population is this run's names with %s dropped, so its residual is a "+
			"member of the family that band is the largest of, measured over all %d "+
			"of them and not scaled from a smaller step",
			affordedStepLeaves(stand), affordedNameList(standMoved), standOver)
	case standBands != nil:
		reading = fmt.Sprintf("the census against the record within the band "+
			"REMOVING %s is worth to each ending, asserted over the band measured "+
			"on the RECORD's names — this run's population is those names with %s "+
			"dropped, so its residual is a member of the family that band is the "+
			"largest of, measured over all %d of them. This run cannot walk that "+
			"family from its own names; it is walkable because the record carries "+
			"the names it was taken over",
			affordedStepLeaves(stand), affordedNameList(standMoved), standOver)
	case distance == 0:
		reading = fmt.Sprintf("the record's re-walk and the four floors: this run "+
			"has the record's %d leaf names by COUNT and not by name — %d of them "+
			"are not here — so a leaf was renamed, this run's counts are a walk "+
			"over different strings, and comparing them against the record exactly "+
			"would report a re-sort nobody made",
			affordedMeasuredOn.leaves, strangers)
	case distance >= -affordedBandSteps[len(affordedBandSteps)-1] &&
		distance <= affordedBandSteps[len(affordedBandSteps)-1]:
		reading = fmt.Sprintf("the record's re-walk and the four floors: this run "+
			"is %d leaves from the record by COUNT and not by name — %d of the "+
			"record's %d names are not here — so neither population is the other "+
			"with leaves merely dropped, and a band whose family does not contain "+
			"the case it brackets is a tolerance wearing a proof's clothes",
			distance, strangers, affordedMeasuredOn.leaves)
	default:
		widest := affordedBandSteps[len(affordedBandSteps)-1]
		reading = fmt.Sprintf("the record's re-walk and the four floors: this run "+
			"is %d leaves from the record's %d, and the widest step measured here "+
			"is %s. It cannot be scaled to reach %d — the drift is sub-linear "+
			"losing and worse than linear gaining — and the family cannot be walked "+
			"either: dropping %d of %d names is %d populations against the %d a "+
			"step of %s makes. So an ending outside the band below is the absence "+
			"of a finding rather than one",
			distance, affordedMeasuredOn.leaves, affordedStepLeaves(widest),
			distance, distance, affordedMeasuredOn.leaves,
			affordedDropCount(affordedMeasuredOn.leaves, distance),
			affordedDropCount(affordedMeasuredOn.leaves, widest),
			affordedStepLeaves(widest))
	}
	// And what each step past the first came to, against the step below it.
	// The ratios are the argument against `band × k` and the eight comparisons
	// after them are the argument against composing one, so a green run carries
	// both rather than leaving them in comments nobody re-derives.
	//
	// Against the step BELOW rather than always against the one-leaf band,
	// because that is the number `band × k` is wrong by at each step and it is
	// the one that says whether the drift is settling down. It is not: the
	// crowding ending gains 2.64× from one leaf to two and 2.55× again from
	// two to three.
	twoBandNote := ""
	{
		ratios := make([]string, 0, len(endings)*len(affordedBandSteps))
		for _, k := range affordedBandSteps {
			if k == 1 {
				continue // nothing below it to be a multiple of
			}
			for _, ending := range endings {
				below, at := recordStep[k-1][ending], recordStep[k][ending]
				ratios = append(ratios, fmt.Sprintf(
					"%s %q ±%.2f%%/%.2f%% (%.2f×/%.2f× the %s figure)",
					affordedStepLeaves(k), ending,
					affordedPercent(at.losing), affordedPercent(at.gaining),
					affordedStepRatio(at.losing, below.losing),
					affordedStepRatio(at.gaining, below.gaining),
					affordedStepLeaves(k-1)))
			}
		}
		// And how the composed bound came out against them. The whole reason
		// the step is measured rather than multiplied is that this list is not
		// empty, so a passing run says which of the eight it is.
		composed := "covers all eight, which the arm above reports"
		if len(chainSaid) > 0 {
			composed = fmt.Sprintf("comes in UNDER the measurement on %d of the "+
				"eight — %s, so a composed bound would call an honest two-field edit "+
				"a re-sort", len(chainSaid), strings.Join(chainSaid, "; "))
		}
		widest := affordedBandSteps[len(affordedBandSteps)-1]
		twoBandNote = fmt.Sprintf(". Each step past the first, over every drop of "+
			"that size from the record's names, moves the endings by %s — losing "+
			"under linear and gaining over it at every step, which is why each band "+
			"is measured for its own step rather than scaled from a smaller one. "+
			"Multiplying the one-leaf bands along ONE chain instead — the "+
			"construction that would cost 240 walks rather than %d — %s. Over EVERY "+
			"chain it does cover, which is what says the cheap one fails on its "+
			"representative population rather than on its shape, and is a check "+
			"between two walks that share only themeLeafSetOf: %s. It is still not "+
			"a way past %s, because bounding the last step over every chain needs a "+
			"census of every population of size n−k, which is the family the direct "+
			"measurement already walks — and the direct measurement is now the "+
			"cheaper of the two at every k this file takes",
			strings.Join(ratios, ", "),
			affordedDropCount(affordedMeasuredOn.leaves, 2), composed,
			strings.Join(soundSaid, ", "), affordedStepLeaves(widest))
	}

	// The band itself, so the numbers the note above argues from are in a
	// green run's line rather than only in a failure's.
	bandSaid := make([]string, 0, len(endings))
	for _, ending := range endings {
		b := bandsFor[ending]
		bandSaid = append(bandSaid, fmt.Sprintf("%q ±%.2f%%/%.2f%%",
			ending, affordedPercent(b.losing), affordedPercent(b.gaining)))
	}
	// And, on a run that found something, the records as this walk would write
	// them.
	//
	// Measured over THIS run's names rather than the record's, because that is
	// what the record would become — the bands above are over the population
	// being compared AGAINST, and pasting those would re-baseline the old
	// population under the new name. It costs the two-leaf walk a second time
	// and it is only paid by a run that has already failed, which is the run
	// where somebody is about to do the transcription by hand.
	//
	// See affordedRetakeSource for why this is printed rather than written.
	if t.Failed() {
		freshTwo, _, freshSound, _ := affordedTwoStepBands(names)
		fresh := map[int]map[string]affordedBand{}
		for _, k := range affordedBandSteps {
			if k == 2 {
				fresh[k] = freshTwo
				continue
			}
			fresh[k], _ = affordedKLeafBand(names, k)
		}
		freshChain, _ := affordedChainBoundOf(names, 2)
		t.Log(affordedRetakeSource(names, reached, fresh, freshChain, freshSound))
	}

	// Read here rather than at the Logf's argument list so the retake walk
	// above — which asks for three more bands, and only on a run that has
	// already failed — is inside the count. What the sentence reports is the
	// work this test caused, whichever branch it took.
	memoWalks, memoReused, memoNote := affordedBandMemoSince(baseWalks, baseReused)

	// And that those two are filed under this test's own name, which is the
	// half the per-caller tally asserted about nobody. See
	// affordedHoldBandAttribution: the attribution is a convention about stack
	// frames, and t.Name() is the same fact from the side that cannot be wrong
	// about it.
	affordedHoldBandAttribution(t, memoWalks, memoReused)

	t.Logf("%d generated leaf sets over %d distinct leaf names, none of them a "+
		"shape anybody chose, hold afforded >= edits and the open flag's "+
		"parent-of-three — each ending against its own floor, %d of the four on a "+
		"share of its own measurement and the rest on the stated %d, over a "+
		"population at %.2f× the one the record was taken on and partitioning it "+
		"(%d sets, %d accounted for), every ending within %.1f%% of what that scale "+
		"predicts for it (the furthest being %q), which is %s. The record's own %d "+
		"names were re-walked and hold its four counts exactly, which is the one "+
		"reading here that does not need this run's population to have stood still "+
		"— the walk is a function of a list of strings and the record carries the "+
		"list. Against THIS run the reading in force is %s. Floors: %s. One leaf "+
		"lost/gained moves each ending by %s. The band in force here is the "+
		"%d-leaf step, measured over %s%s. %d population(s) were censused for a "+
		"band and %d further asking(s) found one already taken — the one-leaf band "+
		"over the record's names alone is asked for by the steps loop, by the "+
		"composition's b1, by the chain's first step and, while core.Theme still "+
		"has the record's population, by this run's own band. A walk of 80 names is "+
		"13ms and a reuse is under 3µs; this test measured 0.78s without the memo "+
		"and 0.72s with it. The numbers are here because a `reused` that drops is a "+
		"caller asking about a different population — which is a change in what is "+
		"being read rather than in what it costs. Every one of those askings is "+
		"filed under this test's own name: affordedBandCaller reads the outermost "+
		"Test frame on the stack and t.Name() is the same fact from the side that "+
		"cannot be wrong about it, so the per-caller tally is a claim here rather "+
		"than a convention%s%s",
		len(sets), len(names), onShare, affordedEndingFloor, scale,
		len(sets), census, affordedPercent(worst), worstEnding, resorted,
		len(affordedMeasuredOn.names), reading,
		strings.Join(held, ", "), strings.Join(bandSaid, ", "), bandStep, bandWhose,
		twoBandNote, memoWalks, memoReused,
		memoNote, affordedMeasuredNote(len(names), len(sets)))
}

// affordedDeclaredFields is the field names of a package-level var's struct
// type, read out of this file's own source.
//
// The paste below has to fit a declaration, and the declaration is in the file
// rather than in anybody's memory of it. Reading it is the same move
// TestTheTwoFoldsDropTheSameCharacters makes about browser.mjs and
// TestNoFloatAComparisonRestsOnIsDerivedTwice makes about this package's
// arithmetic: compare the two implementations, not two copies of one list.
func affordedDeclaredFields(t *testing.T, name string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "themenearmiss_test.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing this file to find %s's declaration: %v", name, err)
	}
	var fields []string
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 || spec.Names[0].Name != name {
			return true
		}
		found = true
		// `var x = struct{...}{...}` carries the type inside the literal and
		// `var x struct{...} = ...` carries it beside the name. Both are
		// declarations of the same shape and this file uses the first, so the
		// second is read too rather than assumed absent.
		typed := spec.Type
		if typed == nil && len(spec.Values) == 1 {
			if lit, ok := spec.Values[0].(*ast.CompositeLit); ok {
				typed = lit.Type
			}
		}
		st, ok := typed.(*ast.StructType)
		if !ok || st.Fields == nil {
			return false
		}
		for _, f := range st.Fields.List {
			for _, id := range f.Names {
				fields = append(fields, id.Name)
			}
		}
		return false
	})
	if !found {
		t.Fatalf("%s is not declared in themenearmiss_test.go as a package-level "+
			"var this test can read. It is the record a failing run offers a reader "+
			"a paste for, and the paste is checked against the declaration rather "+
			"than against a copy of it.", name)
	}
	return fields
}

// affordedPasteKeys is the keys of a composite literal written as the body of
// one, parsed back.
//
// The paste is a field list without its braces, so it is given the braces and
// a type and handed to the parser. A key the declaration does not have parses
// perfectly well — Go's parser does not type-check — which is why the caller
// compares the keys rather than trusting the parse.
func affordedPasteKeys(t *testing.T, what, body string) map[string]ast.Expr {
	t.Helper()
	src := "package p\n\nvar _ = struct{}{\n" + body + "}\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "paste.go", src, 0)
	if err != nil {
		t.Fatalf("the %s paste a failing run prints does not parse: %v\n\n"+
			"It is offered to a reader as something to put in the source, so a run "+
			"that cannot parse its own offer is printing a paste nobody can use — "+
			"and it would print it on exactly the run where somebody needs it. The "+
			"text was:\n\n%s", what, err, body)
	}
	keys := map[string]ast.Expr{}
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if id, ok := kv.Key.(*ast.Ident); ok {
				keys[id.Name] = kv.Value
			}
		}
		return false
	})
	return keys
}

// The paste a failing run offers, against the declarations it is meant to go
// into.
//
// # What this is for
//
// affordedRetakeParts writes out the records in source shape so that re-taking
// them is a paste rather than eighty names transcribed by hand. Nothing read
// it back. A field renamed in affordedMeasuredOn, or one added to
// affordedBandMeasuredOn — which is what happened when the sound chain bound
// was recorded — leaves the printer emitting a paste that no longer fits, and
// it emits it silently on the one run where somebody needs it.
//
// The paste is not a record of anything, so there is nothing here to hold it
// still. What there is instead is the declaration, in this file, read out of
// the source: every key the printer writes has to be a field of the struct it
// says it is for, and every field of that struct has to be one the printer
// writes. Both directions, because the two failures are different — a paste
// that will not compile, and a paste that quietly leaves a field at whatever
// it was.
func TestTheRetakePasteIsTheShapeTheRecordsAreIn(t *testing.T) {
	names := affordedLeafNames()
	census := affordedCensusMap(affordedCensusOf(names, affordedWindowMax))
	// The recorded bands rather than re-measured ones: what is under test is
	// the SHAPE of the paste, and the numbers in it are held by the arms in
	// TestTheAffordedWidthHoldsItsTwoRelations. Re-measuring here would be two
	// seconds spent proving something about a printer.
	record, bands := affordedRetakeParts(names, census,
		affordedBandMeasuredOn.step, affordedBandMeasuredOn.chain,
		affordedBandMeasuredOn.sound)

	for _, part := range []struct {
		what, body, decl string
	}{
		{"census record", record, "affordedMeasuredOn"},
		{"band record", bands, "affordedBandMeasuredOn"},
	} {
		keys := affordedPasteKeys(t, part.what, part.body)
		declared := affordedDeclaredFields(t, part.decl)
		for _, field := range declared {
			if _, written := keys[field]; !written {
				t.Errorf("%s declares a field %q and the paste a failing run prints "+
					"does not write it.\n\n"+
					"A paste that leaves a field out does not fail to compile — it "+
					"leaves that field at whatever it was, which for a record is the "+
					"previous measurement sitting beside the new ones with nothing "+
					"saying so. Every other reading in this file would then be "+
					"comparing part of one walk against part of another.",
					part.decl, field)
			}
		}
		for key := range keys {
			if !slices.Contains(declared, key) {
				t.Errorf("the paste for %s writes a key %q and the declaration has "+
					"no such field.\n\n"+
					"That paste is offered to a reader as something to put in the "+
					"source and it would not compile. The printer was written against "+
					"a shape this record no longer has: %v.",
					part.decl, key, declared)
			}
		}
	}

	// And the values, on the two that can be checked against this run without
	// re-measuring anything: the paste says what it walked, so it has to say
	// what this run walked.
	keys := affordedPasteKeys(t, "census record", record)
	if lit, ok := keys["leaves"].(*ast.BasicLit); !ok ||
		lit.Value != strconv.Itoa(len(names)) {
		t.Errorf("the paste says leaves: %s and this run has %d distinct leaf "+
			"names.\n\n"+
			"The count and the list are the same fact written twice in that record, "+
			"and the arm that reads them is the first thing every scaled reading "+
			"rests on. A paste that disagrees with itself would be pasted and then "+
			"reported as a record whose names and count do not match.",
			affordedNodeText(keys["leaves"]), len(names))
	}
	list, ok := keys["names"].(*ast.CompositeLit)
	if !ok {
		t.Fatalf("the paste's names are not a list at all: %s",
			affordedNodeText(keys["names"]))
	}
	if len(list.Elts) != len(names) {
		t.Fatalf("the paste writes %d names and this run walked %d.",
			len(list.Elts), len(names))
	}
	for i, elt := range list.Elts {
		lit, ok := elt.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			t.Errorf("the paste's name at %d is not a string literal: %s",
				i, affordedNodeText(elt))
			continue
		}
		got, err := strconv.Unquote(lit.Value)
		if err != nil || got != names[i] {
			t.Errorf("the paste's name at %d is %s and this run walked %q.\n\n"+
				"The order is the population: the walk takes windows of CONSECUTIVE "+
				"names, so a list in a different order is a record of a different "+
				"set of sets.", i, lit.Value, names[i])
			break
		}
	}
	// And the census's own four counts, which are the rest of what that record
	// is. The names were checked above and the numbers beside them were not:
	// a printer that wrote the endings sorted and the counts in walk order
	// would produce a paste that compiles, fits the declaration, carries the
	// right names, and records the population under the wrong sentences.
	counts, ok := keys["ending"].(*ast.CompositeLit)
	if !ok {
		t.Fatalf("the paste's census is not a map literal: %s",
			affordedNodeText(keys["ending"]))
	}
	written := map[string]int{}
	for _, elt := range counts.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, keyOK := kv.Key.(*ast.BasicLit)
		val, valOK := kv.Value.(*ast.BasicLit)
		if !keyOK || !valOK || key.Kind != token.STRING || val.Kind != token.INT {
			t.Errorf("the paste's census writes %s: %s, which is not a sentence and "+
				"a count.", affordedNodeText(kv.Key), affordedNodeText(kv.Value))
			continue
		}
		ending, err := strconv.Unquote(key.Value)
		if err != nil {
			continue
		}
		n, err := strconv.Atoi(val.Value)
		if err != nil {
			continue
		}
		written[ending] = n
	}
	for ending, want := range census {
		if written[ending] != want {
			t.Errorf("the paste puts %d sets at %q and this run walked %d there.\n\n"+
				"The paste is offered as the record this run would become, so its "+
				"counts are this run's census or it is a record of some other walk.",
				written[ending], ending, want)
		}
	}
	for ending := range written {
		if _, walked := census[ending]; !walked {
			t.Errorf("the paste's census carries %q and this run's walk reached no "+
				"such ending.", ending)
		}
	}

	// # And the band paste's numbers, which nothing read
	//
	// The arms above compare KEYS in both directions and the census's own
	// values. The bands' values had nobody: the printer is handed three maps
	// and writes them out, and every failure in that transport produces a
	// paste that parses, fits the declaration and carries the wrong
	// measurement — a step's band written under another step's key, losing
	// printed where gaining goes, the chain composition written under `sound`.
	// Each of those pastes compiles. Each of them re-baselines a number
	// against a walk that did not produce it, which is the one thing every
	// message in this file tells a reader not to do.
	//
	// # Why this is not measured here
	//
	// The two honest ways to check the values were to measure the bands in
	// this test and compare — two seconds spent proving something about a
	// printer — or to check a failing run's own paste against the numbers that
	// run measured, which only ever runs on a failing run.
	//
	// Neither is needed, because what is under test is transport. The printer
	// takes the numbers as arguments, so it is handed a set in which every
	// slot is DIFFERENT — one value per step, per ending, per direction, and a
	// separate range for the chain and the sound composition — and every one
	// has to come back where it was put. A printer that transports whatever it
	// is given faithfully, given this run's measurements by the failing run
	// above, writes this run's measurements; and any mix-up between two slots
	// is two values that are not equal here, where the recorded bands (four
	// endings at 0.0004 apiece on one of them) would have hidden it.
	{
		slot := 0
		next := func() float64 {
			slot++
			// A distinct four-decimal value per slot, which is the precision
			// the printer writes at — so a number that comes back unequal came
			// back from the wrong slot rather than from rounding.
			return float64(slot) / 10000
		}
		step := map[int]map[string]affordedBand{}
		for _, k := range affordedBandSteps {
			at := map[string]affordedBand{}
			for _, ending := range affordedEndingNames {
				at[ending] = affordedBand{losing: next(), gaining: next()}
			}
			step[k] = at
		}
		chain := map[string]affordedBand{}
		sound := map[string]affordedBand{}
		for _, named := range []map[string]affordedBand{chain, sound} {
			for _, ending := range affordedEndingNames {
				named[ending] = affordedBand{losing: next(), gaining: next()}
			}
		}
		_, printed := affordedRetakeParts(names, census, step, chain, sound)
		back := affordedPasteKeys(t, "band record", printed)
		lit, ok := back["step"].(*ast.CompositeLit)
		if !ok {
			t.Fatalf("the band paste's steps are not a map literal: %s",
				affordedNodeText(back["step"]))
		}
		got := map[int]map[string]affordedBand{}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.BasicLit)
			if !ok || key.Kind != token.INT {
				t.Errorf("the band paste's steps are keyed by %s, which is not a "+
					"step size.", affordedNodeText(kv.Key))
				continue
			}
			k, err := strconv.Atoi(key.Value)
			if err != nil {
				continue
			}
			got[k] = affordedPasteBands(t, fmt.Sprintf("band for a step of %d", k),
				kv.Value)
		}
		for what, pair := range map[string][2]map[string]affordedBand{
			"chain": {chain, affordedPasteBands(t, "chain composition", back["chain"])},
			"sound": {sound, affordedPasteBands(t, "sound composition", back["sound"])},
		} {
			for _, ending := range affordedEndingNames {
				if pair[0][ending] == pair[1][ending] {
					continue
				}
				t.Errorf("the printer was given %.4f losing and %.4f gaining for %q's "+
					"%s composition and wrote %.4f and %.4f.\n\n"+
					"Every slot in this reading holds a different number, so what came "+
					"back is not a rounding — it is another slot's measurement under "+
					"this key. A failing run hands this printer the bands it just "+
					"measured and a reader pastes what comes out, so a transport that "+
					"crosses two slots records one walk's number against another "+
					"walk's name and every reading in this file goes on comparing "+
					"against it.",
					pair[0][ending].losing, pair[0][ending].gaining, ending, what,
					pair[1][ending].losing, pair[1][ending].gaining)
			}
		}
		for _, k := range affordedBandSteps {
			for _, ending := range affordedEndingNames {
				if got[k][ending] == step[k][ending] {
					continue
				}
				t.Errorf("the printer was given %.4f losing and %.4f gaining for %q at "+
					"a step of %s and wrote %.4f and %.4f.\n\n"+
					"Every slot here holds a different number, so this is another "+
					"step's band, another ending's, or the two directions swapped — "+
					"and all three produce a paste that compiles. The whole point of "+
					"the paste is that a reader does not have to check the numbers by "+
					"eye.",
					step[k][ending].losing, step[k][ending].gaining, ending,
					affordedStepLeaves(k), got[k][ending].losing, got[k][ending].gaining)
			}
		}
		for k := range got {
			if !slices.Contains(affordedBandSteps, k) {
				t.Errorf("the band paste writes a step of %s and nothing measures "+
					"one.", affordedStepLeaves(k))
			}
		}
	}
}

// affordedPasteBands reads a `map[string]affordedBand` literal back out of a
// paste, as the numbers it holds.
//
// The two directions only. `losingAt` and `gainingAt` are carried through a
// walk so a failure can name which leaf moved an ending; they are not fields
// of the record and the printer does not write them, which the key-set arms
// above are what hold.
func affordedPasteBands(t *testing.T, where string, n ast.Expr) map[string]affordedBand {
	t.Helper()
	out := map[string]affordedBand{}
	lit, ok := n.(*ast.CompositeLit)
	if !ok {
		t.Errorf("the paste's %s is not a map literal at all: %s",
			where, affordedNodeText(n))
		return out
	}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.BasicLit)
		if !ok || key.Kind != token.STRING {
			t.Errorf("the paste's %s is keyed by %s, which is not an ending's own "+
				"sentence.", where, affordedNodeText(kv.Key))
			continue
		}
		ending, err := strconv.Unquote(key.Value)
		if err != nil {
			t.Errorf("the paste's %s has a key that will not unquote: %s",
				where, key.Value)
			continue
		}
		band := ast.Expr(kv.Value)
		inner, ok := band.(*ast.CompositeLit)
		if !ok {
			t.Errorf("the paste's %s writes %s for %q, which is not a band.",
				where, affordedNodeText(band), ending)
			continue
		}
		var got affordedBand
		for _, f := range inner.Elts {
			fkv, ok := f.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			name, ok := fkv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			text := affordedNodeText(fkv.Value)
			v, err := strconv.ParseFloat(text, 64)
			if err != nil {
				t.Errorf("the paste's %s writes %s for %q's %s, which is not a "+
					"number.", where, text, ending, name.Name)
				continue
			}
			switch name.Name {
			case "losing":
				got.losing = v
			case "gaining":
				got.gaining = v
			default:
				t.Errorf("the paste's %s writes a field %q inside %q's band, and a "+
					"band is losing and gaining.", where, name.Name, ending)
			}
		}
		out[ending] = got
	}
	return out
}

// affordedNodeText is an expression as it was written, for a message about it.
func affordedNodeText(n ast.Node) string {
	if n == nil {
		return "nothing"
	}
	var b strings.Builder
	if err := printer.Fprint(&b, token.NewFileSet(), n); err != nil {
		return "an expression that will not print"
	}
	return b.String()
}

// Every float this package's checks derive, and where each one is derived.
//
// # What this is for
//
// The residual and the band it is read against were once computed by two
// expressions that were the same arithmetic written twice. On arm64 they
// disagreed in the last bit — Go may fuse a multiply and the subtract after it
// into a single rounding, and it did so at one spelling and not at the other —
// and against a band that is the MAXIMUM of a family the residual is a member
// of, one ulp is the whole distance between "inside its band" and a green run
// reporting a re-sort that did not happen. It surfaced as a 0.02%-band ending
// reported 0.0% out, in a break-test's odd output, which is the only reason
// anybody saw it.
//
// That was fixed at the one site and prevented nowhere, and nothing had looked
// for a second. Looking found three more of the same shape:
//
//	float64(measured) * scale   four spellings — the prediction. Two of them
//	                            DECIDED a floor, in opposite directions, at two
//	                            sites; two printed it beside a residual taken
//	                            from a third.
//	math.Abs(scale - 1)         two spellings, both comparing against
//	                            affordedScaleSame — one choosing which of three
//	                            findings a failure names, the other choosing
//	                            which sentence a green run prints.
//	float64(part) / float64(whole)
//	                            the band's step scale and affordedScale's, which
//	                            the k-leaf assertions require to be the same
//	                            number: the residual is only a member of the
//	                            band's family while the two divisions agree.
//
// Each is now one function — affordedPredictedFrom, affordedPopulationUnchanged,
// affordedRatioOf — and this census is what stops a fifth spelling appearing.
//
// # What a derivation is here
//
// Read off the package's test sources: every `* / + -` between operands and
// every call into `math`, where the expression is float arithmetic and no
// string literal appears inside it. Sub-expressions count as well as their
// parents, because the hazard is arithmetic and not statements.
//
// "Is float arithmetic" is asked two ways, and the second is new. The first is
// the printed text — a `float64(` conversion, a `math.` call or a decimal
// literal (affordedLooksFloat). The second is a small inference over the
// syntax (affordedFloatSource, affordedFloatLocals): which locals hold floats,
// which struct fields are declared as floats, which functions return them. It
// is what makes `down * step` a derivation — a product of two variables that
// carries none of the three text marks, and is exactly the arithmetic the
// chain composition rests on.
//
// The text test is still asked first and the inference only widens, so nothing
// this census used to count can stop being counted by an inference that
// decided otherwise.
//
// # What the inference found the day it was turned on
//
// Nine expressions in eighteen places, all of them one multiply: a fraction
// printed as a percentage. Two of the nine were the SAME expression in two
// functions — `b.losing * 100` and `b.gaining * 100`, in affordedBandFor and
// in the log line — which is exactly the shape this census exists to report
// and about which it could say nothing at all, because a float variable times
// an integer literal carries none of the three text marks. They are
// affordedPercent now: one function, one entry.
//
// Two more were the inner halves of entries already here. `scale - 1` sits
// under its own math.Abs and `v * 10000` under its own math.Round, and both
// were invisible on their own because the float-ness was in the wrapper.
//
// And the arithmetic added with the chain bound is float over float
// throughout — `down * step`, `1 + b`, `product - 1` — so under the old
// reading none of the composition would have been censused at all. That is the
// point rather than a bonus: the rule is only worth holding over the
// arithmetic there is.
//
// # And what it found the second time it was widened
//
// Reading methods, this repository's other packages, variadic floats and the
// value of a range took the census from 25 derivations to 28 and found two
// more things, one of them in code written the same afternoon:
//
//	a duplicate                the chain composition was spelled in
//	                           affordedChainBoundOf and again in
//	                           affordedTwoStepBands — `1 + b.losing` in
//	                           both — which is the identity two readings rest
//	                           on, written twice, against bands one of them
//	                           has to dominate. It is affordedComposedOf now.
//	the palette multiply       `palette.Ratio(edge, lum) * 100` is float
//	                           arithmetic because internal/palette's own
//	                           source says Ratio returns one. The wrapper
//	                           around it was already an entry; the multiply
//	                           inside it was not, and could not be.
//
// The duplicate is the answer to whether this is worth its complexity. It was
// found by the census, in new code, on the day the census learned to see it.
//
// The rule the table enforces is the one the FMA hazard taught: TWO FLOATS
// THAT ARE COMPARED MUST COME FROM ONE EVALUATION. What that reduces to
// syntactically is that no derivation is spelled in two functions — a
// derivation in one place cannot disagree with itself.
var affordedFloatDerivations = []struct{ expr, what string }{
	{"f * float64(unit)",
		"a recorded band's endpoint, turned from the decimal the record " +
			"writes into a Duration. NOT COMPARED as a float: the " +
			"multiplication happens once per endpoint, the result is a " +
			"Duration immediately, and every comparison downstream of it is " +
			"integer. A rounding difference here moves a band edge by less " +
			"than a nanosecond, which is smaller than the clock that feeds " +
			"the other side. See recordedBand"},
	{"float64(recorded) * scale",
		"the prediction: a recorded count scaled by this run's population. " +
			"COMPARED — the share-edge arm and affordedShortfallCause both decide a " +
			"floor with it, and the residual is taken against it"},
	{"float64(got) - predicted",
		"the gap between a count and its prediction, inside the residual"},
	{"math.Abs(float64(got) - predicted)",
		"the same gap without its sign — the sub-expression the fusing happened " +
			"at"},
	{"math.Abs(float64(got)-predicted) / predicted",
		"the residual itself. COMPARED against every band in this file"},
	{"math.Abs(scale - 1)",
		"how far the population moved. COMPARED against affordedScaleSame, which " +
			"decides whether a walk is called unchanged"},
	{"float64(part) / float64(whole)",
		"one population as a fraction of another. COMPARED by construction: the " +
			"k-leaf arms are proofs only while the scale the residual is taken with " +
			"and the scale the band was measured with are the same number"},
	{"float64(reached[ending]) - predicted",
		"how far above its prediction an ending came in, for naming which ending " +
			"holds another's sets"},
	{"v * 100",
		"a fraction as the sentences here print it — affordedPercent. Eleven " +
			"spellings before this census could see any of them, two of which were " +
			"one expression in two functions"},
	{"v * 10000", "the band record's four decimal places: the multiply"},
	{"math.Round(v * 10000)", "the same, rounded"},
	{"bigger / smaller",
		"one step's band as a multiple of a smaller step's — affordedStepRatio, " +
			"the ×-column of the log line. Not compared: what the ratio argues is " +
			"held by the two recorded bands themselves, and this is the sentence " +
			"that carries the argument to a reader"},
	{"scale - 1",
		"the population's distance from unchanged, inside affordedPopulationUnchanged"},
	{"1 + b",
		"one step's factor in the chain composition. The identity is ∏(1+b_i)−1, " +
			"so each band is carried as a factor and the product is finished in one " +
			"place — see affordedComposedOf, which is what both chain readings " +
			"compose through"},
	{"product - 1",
		"the finished chain product back to a residual. COMPARED: it is held " +
			"against the measured two-leaf band, against " +
			"affordedBandMeasuredOn.chain and, in the sound version, against the " +
			"band it has to dominate"},
	{"affordedComposeMeasuredOn.worst * affordedComposeDrift",
		"the decade the recorded compose spread is allowed to move within. " +
			"COMPARED against this run's worst"},
	{"worst * affordedComposeDrift",
		"the same decade taken the other way round, so a spread that SHRANK by " +
			"one is reported too. COMPARED against the record"},
	{"worst * affordedComposeMeasuredOn.margin",
		"the floor affordedScaleCompose has to clear to be a bracket over that " +
			"spread rather than a number somebody liked. COMPARED against the " +
			"constant"},
	{"palette.Ratio(edge, lum) * 100",
		"a contrast ratio at two decimal places: the multiply, which this census " +
			"could not see until it learned to read another package's signatures"},
	{"palette.Ratio(la, lb) * 100", "the same over the widget swatches' colours"},
	{"float64(slot) / 10000",
		"one slot's distinguishing value in the band paste's transport check. " +
			"COMPARED, and that is the whole of what it is for: every step, ending " +
			"and direction the printer is handed gets a different number at the " +
			"precision the printer writes, so a value that comes back unequal came " +
			"back from another slot rather than from a rounding. The divisor is the " +
			"four decimal places affordedBandRounded records at"},
	{"down * step",
		"the two-leaf scale reached as one leaf then another. COMPARED against " +
			"the same ratio taken directly, which is the premise the chain " +
			"composition rests on — and the derivation this census could not see " +
			"before it read local float-ness: a product of two variables carries no " +
			"conversion, no math call and no decimal point"},
	{"composed - whole",
		"how far the two spellings of that scale sit apart"},
	{"math.Abs(composed - whole)",
		"the same without its sign"},
	{"math.Abs(composed-whole) / whole",
		"the same as a fraction of the scale. COMPARED against affordedScaleCompose"},
	{"math.Round(v*10000) / 10000",
		"the same, back to a fraction. COMPARED against affordedBandMeasuredOn " +
			"and its composed chain bound"},
	{"math.Round(palette.Ratio(edge, lum) * 100)",
		"a contrast ratio at two decimal places, in the palette census"},
	{"math.Round(palette.Ratio(edge, lum)*100) / 100",
		"the same, back to a ratio. COMPARED against the recorded palette rows"},
	{"math.Round(palette.Ratio(la, lb) * 100)",
		"the same recipe over the widget swatches' own two colours — one rounding " +
			"written in two files, which textual identity cannot see and this table " +
			"can"},
	{"math.Round(palette.Ratio(la, lb)*100) / 100",
		"the same, back to a ratio. COMPARED against the swatch's own reported " +
			"figure"},
}

// affordedFloatSource is what this census reads off the package's own test
// sources before it looks for arithmetic: which functions hand back a float,
// and which struct fields hold one.
//
// Both are what let the reading see a derivation over two float VARIABLES.
// affordedLooksFloat is a text test — a `float64(` conversion, a `math.` call
// or a decimal literal — and `a * b` where both operands came from somewhere
// else carries none of the three. That was the census's stated limit and it is
// the same shape of argument the FMA note exists to replace: every hazard the
// file had actually taken carried one of the three, which is a fact about the
// hazards found rather than about the ones there.
//
// # What this is not
//
// It is not go/types. The honest fix is to run the type checker over the
// package, and the cost is a second toolchain inside a test suite that imports
// half the repository — so this is a small inference over the syntax instead,
// and it is deliberately coarse in the direction that costs a false ENTRY
// rather than a missed derivation:
//
//	returns  a package-level function whose result at that position is
//	         declared float64.
//	method   the same for a method, by NAME and without its receiver — so a
//	         `changed()` that returns a float on one type makes every
//	         `x.changed()` read as one. Two methods with a name and different
//	         result types would be conflated, and there are none today.
//	from     the same for a function in another package of THIS repository,
//	         keyed by the name the import is used under: `palette.Ratio(a, b)`
//	         is a float here because internal/palette says so in its own
//	         source. Packages outside the module — the standard library
//	         included, `math` apart, which is read as a name — are still
//	         invisible, and that is the gap this cannot close without the type
//	         checker.
//	field    a struct field declared float64, by NAME and not by type — so
//	         `b.losing` reads as a float wherever `losing` is a float field
//	         anywhere in these sources. Two structs with one field name would
//	         be conflated, and there are none today.
//
// The locals are scoped to a function and not to a block: an identifier that
// holds a float anywhere in a function marks every expression in it that
// mentions the name. Shadowing would be read wrongly, and the direction is
// again the safe one — an entry somebody has to write rather than arithmetic
// nobody sees.
type affordedFloatSource struct {
	fset    *token.FileSet
	files   map[string]*ast.File
	returns map[string][]bool
	method  map[string][]bool
	from    map[string][]bool
	field   map[string]bool
	// Names more than one declaration answers differently, which is the whole
	// of what three of those tables cannot represent — split by which way the
	// disagreement goes, because they are two different findings.
	//
	//	collide   an answer was REPLACED. The method and import tables are
	//	          assigned into, so the surviving answer is whichever
	//	          declaration was parsed last and the order is the directory
	//	          listing. That can go either way, and one of the two ways is
	//	          real float arithmetic silently stopping being counted.
	//	conflate  the FIELD table, which is a union rather than an assignment:
	//	          a name declared float in any struct marks every selector with
	//	          that name. Order-independent, and always in the direction
	//	          this census calls safe — an entry somebody has to write
	//	          rather than arithmetic nobody sees.
	//
	// See the two arms in the test: the first fails and the second is counted
	// in the log line, and both are reported rather than resolved, because
	// resolving them is go/types.
	collide  []string
	conflate []string
}

// affordedFloatResults is a signature's results as this census reads them, for
// a message about two that disagree.
func affordedFloatResults(results []bool) string {
	if len(results) == 0 {
		return "nothing"
	}
	said := make([]string, 0, len(results))
	for _, isFloat := range results {
		if isFloat {
			said = append(said, "a float")
			continue
		}
		said = append(said, "something else")
	}
	return strings.Join(said, " and ")
}

// affordedReceiverName is the type a method is declared on, pointer or not.
func affordedReceiverName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return "nothing"
	}
	t := recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return affordedNodeText(t)
}

// affordedFloatType is whether a type expression is one of Go's floats,
// written out.
func affordedFloatType(t ast.Expr) bool {
	id, ok := t.(*ast.Ident)
	return ok && (id.Name == "float64" || id.Name == "float32")
}

// affordedFloatSourceOf parses this package's test sources and reads the two
// tables above off them.
func affordedFloatSourceOf(t *testing.T) affordedFloatSource {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading this package's directory: %v", err)
	}
	src := affordedFloatSource{
		fset:    token.NewFileSet(),
		files:   map[string]*ast.File{},
		returns: map[string][]bool{},
		method:  map[string][]bool{},
		from:    map[string][]bool{},
		field:   map[string]bool{},
	}
	// Where each table entry's kept answer came from, and the field names some
	// struct declares as something other than a float — the two halves of the
	// collision census below.
	where := map[string]string{}
	notFloat := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(src.fset, e.Name(), nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", e.Name(), err)
		}
		src.files[e.Name()] = file
	}
	for _, file := range src.files {
		// Every struct in the sources, named or anonymous — affordedBand is a
		// type and affordedMeasuredOn's shape is a literal, and both hold
		// numbers this census is about.
		ast.Inspect(file, func(n ast.Node) bool {
			st, ok := n.(*ast.StructType)
			if !ok || st.Fields == nil {
				return true
			}
			for _, f := range st.Fields.List {
				isFloat := affordedFloatType(f.Type)
				for _, name := range f.Names {
					// Both answers are kept, because the table can only hold
					// one: a field name declared float in one struct and an
					// int in another marks EVERY selector with that name as
					// float arithmetic, and the census then asks for an entry
					// describing a derivation that is not one.
					if isFloat {
						src.field[name.Name] = true
					} else {
						notFloat[name.Name] = affordedNodeText(f.Type)
					}
				}
			}
			return true
		})
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			into, table := src.returns, "function"
			whose := "the function"
			if fn.Recv != nil {
				// By name and without the receiver, which is the same
				// approximation the field table makes and is named where it
				// is taken.
				into, table = src.method, "method"
				whose = "the method on " + affordedReceiverName(fn.Recv)
			}
			results := affordedResultsOf(fn.Type)
			key := table + " " + fn.Name.Name
			// An assignment into a name-keyed table is a REPLACEMENT, and the
			// one it replaces is gone with no record that there were two. That
			// is the direction that matters: whichever of the two is parsed
			// last decides how every call by that name reads, and the file
			// order is the directory listing.
			if had, seen := into[fn.Name.Name]; seen && !slices.Equal(had, results) {
				src.collide = append(src.collide, fmt.Sprintf(
					"%s %q — %s returns %s and %s returns %s", table, fn.Name.Name,
					where[key], affordedFloatResults(had), whose,
					affordedFloatResults(results)))
			}
			into[fn.Name.Name], where[key] = results, whose
		}
	}
	for name, was := range notFloat {
		if src.field[name] {
			src.conflate = append(src.conflate, fmt.Sprintf(
				"%q (also declared as %s)", name, was))
		}
	}
	// And the packages of this repository these checks import, which is the
	// half of "another package" that is readable without the type checker: the
	// sources are on disk, in the module, and their exported signatures say
	// what comes back.
	var imported []string
	src.from, imported = affordedImportedFloats(t, src.files)
	src.collide = append(src.collide, imported...)
	slices.Sort(src.collide)
	slices.Sort(src.conflate)
	return src
}

// affordedResultsOf is which of a signature's results are floats, one entry
// per result, counting a named group as one entry per name.
func affordedResultsOf(sig *ast.FuncType) []bool {
	var results []bool
	if sig.Results == nil {
		return results
	}
	for _, f := range sig.Results.List {
		n := len(f.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			results = append(results, affordedFloatType(f.Type))
		}
	}
	return results
}

// affordedImportedFloats reads the signatures of the functions these checks
// call in other packages OF THIS REPOSITORY, keyed by the name the import is
// used under — `palette.Ratio`.
//
// The module root is found by walking up for go.mod rather than stated, and
// an import inside it is a directory whose .go files can be parsed. Anything
// outside the module is skipped: the sources may not be on disk at all, and a
// census that quietly read a stale copy of one would be worse than one that
// says it cannot see it.
//
// A parse that fails is not a failure here. This reading only ever ADDS
// derivations to the census, so a package it cannot read is the census back
// where it was — which is why this returns what it managed rather than
// stopping the test on a directory somebody moved.
func affordedImportedFloats(t *testing.T, files map[string]*ast.File) (
	map[string][]bool, []string) {
	t.Helper()
	root, err := filepath.Abs(".")
	if err != nil {
		return nil, nil
	}
	module := ""
	for dir := root; ; {
		if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if after, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
					module, root = strings.TrimSpace(after), dir
				}
			}
			break
		}
		up := filepath.Dir(dir)
		if up == dir {
			break
		}
		dir = up
	}
	if module == "" {
		return nil, nil
	}
	out := map[string][]bool{}
	seen := map[string]bool{}
	// This table is keyed by the name the import is USED under, so two
	// packages whose paths end in the same segment and neither of which is
	// aliased write over one another — the same replacement the method table
	// makes, one level out.
	collide := []string{}
	from := map[string]string{}
	fset := token.NewFileSet()
	for _, file := range files {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			rest, inside := strings.CutPrefix(path, module+"/")
			if !inside || seen[path] {
				continue
			}
			seen[path] = true
			name := path[strings.LastIndex(path, "/")+1:]
			if imp.Name != nil {
				name = imp.Name.Name
			}
			entries, err := os.ReadDir(filepath.Join(root, rest))
			if err != nil {
				continue // named rather than read; see the note above
			}
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") ||
					strings.HasSuffix(e.Name(), "_test.go") {
					continue
				}
				parsed, err := parser.ParseFile(fset,
					filepath.Join(root, rest, e.Name()), nil, 0)
				if err != nil {
					continue
				}
				for _, d := range parsed.Decls {
					fn, ok := d.(*ast.FuncDecl)
					if !ok || fn.Recv != nil {
						continue
					}
					key := name + "." + fn.Name.Name
					results := affordedResultsOf(fn.Type)
					if had, seen := out[key]; seen && !slices.Equal(had, results) {
						collide = append(collide, fmt.Sprintf(
							"import %q — %s returns %s and %s returns %s", key,
							from[key], affordedFloatResults(had), path,
							affordedFloatResults(results)))
					}
					out[key], from[key] = results, path
				}
			}
		}
	}
	return out, collide
}

// affordedFloatish is whether an expression MENTIONS a float: a decimal
// literal, a conversion, a call into math, a call to a function declared to
// return one, a field declared as one, or a local this pass has already
// decided holds one.
//
// A mention rather than a type, and that is the whole of what a syntactic
// reading can offer. It is the same generosity affordedLooksFloat has —
// anything with `float64(` anywhere inside it counts — carried to the two
// cases text cannot reach.
func affordedFloatish(n ast.Node, floats map[string]bool, src affordedFloatSource) bool {
	found := false
	ast.Inspect(n, func(m ast.Node) bool {
		if found {
			return false
		}
		switch x := m.(type) {
		case *ast.BasicLit:
			if x.Kind == token.FLOAT {
				found = true
			}
		case *ast.Ident:
			if floats[x.Name] {
				found = true
			}
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok && id.Name == "math" {
				found = true
			}
			if src.field[x.Sel.Name] {
				found = true
			}
		case *ast.CallExpr:
			switch fn := x.Fun.(type) {
			case *ast.Ident:
				if fn.Name == "float64" || fn.Name == "float32" {
					found = true
				}
				if r, known := src.returns[fn.Name]; known && len(r) > 0 && r[0] {
					found = true
				}
			case *ast.SelectorExpr:
				// `pkg.Func` when the package is one of this repository's,
				// and `x.Method` otherwise — the two are told apart by which
				// table has the name, because the syntax cannot tell them
				// apart at all.
				if id, ok := fn.X.(*ast.Ident); ok {
					if r, known := src.from[id.Name+"."+fn.Sel.Name]; known &&
						len(r) > 0 && r[0] {
						found = true
					}
				}
				if r, known := src.method[fn.Sel.Name]; known && len(r) > 0 && r[0] {
					found = true
				}
			}
		// A container of floats, which reaches this through the type inside a
		// `make` or a composite literal: `map[string]float64{}` marks the name
		// it is assigned to, so indexing it is float arithmetic. Without it the
		// chain's own subtraction — `losing[ending] - 1`, the last step of the
		// composition — was arithmetic over a map value and invisible to both
		// halves of this reading.
		case *ast.MapType:
			if affordedFloatType(x.Value) {
				found = true
			}
		case *ast.ArrayType:
			if affordedFloatType(x.Elt) {
				found = true
			}
		}
		return !found
	})
	return found
}

// affordedFloatLocals is every identifier in one function that holds a float,
// as far as the syntax can say.
//
// Three sources: a signature that declares one, a `var` that declares or
// initialises one, and an assignment whose right-hand side mentions one. Taken
// to a fixpoint because Go does not require the assignment to come before the
// use in source order — a value assigned inside a loop and read at the top of
// the next iteration is a float on the second pass and not the first.
func affordedFloatLocals(fn *ast.FuncDecl, src affordedFloatSource) map[string]bool {
	floats := map[string]bool{}
	mark := func(e ast.Expr) {
		if id, ok := e.(*ast.Ident); ok && id.Name != "_" {
			floats[id.Name] = true
		}
	}
	for _, list := range []*ast.FieldList{fn.Type.Params, fn.Type.Results} {
		if list == nil {
			continue
		}
		for _, f := range list.List {
			// A variadic float is a float container under the same rule the
			// slices and maps below go by: what comes out of it is a float, so
			// the name marks the expressions that reach into it.
			held := f.Type
			if v, ok := held.(*ast.Ellipsis); ok {
				held = v.Elt
			}
			if !affordedFloatType(held) {
				continue
			}
			for _, name := range f.Names {
				floats[name.Name] = true
			}
		}
	}
	for {
		before := len(floats)
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ValueSpec:
				if affordedFloatType(x.Type) {
					for _, name := range x.Names {
						floats[name.Name] = true
					}
				}
				for i, name := range x.Names {
					if i < len(x.Values) && affordedFloatish(x.Values[i], floats, src) {
						floats[name.Name] = true
					}
				}
			case *ast.RangeStmt:
				// `for _, b := range steps` — the value of a range over
				// something holding floats holds one. The key does not, and
				// the two are separate here because a map of floats is ranged
				// by a string in this file more often than by an index.
				if affordedFloatish(x.X, floats, src) {
					mark(x.Value)
				}
			case *ast.AssignStmt:
				// `a, b := f()` — one call, several names, so the result
				// positions decide which of them hold floats.
				if len(x.Rhs) == 1 && len(x.Lhs) > 1 {
					call, ok := x.Rhs[0].(*ast.CallExpr)
					if !ok {
						return true
					}
					id, ok := call.Fun.(*ast.Ident)
					if !ok {
						return true
					}
					for i, lhs := range x.Lhs {
						if r := src.returns[id.Name]; i < len(r) && r[i] {
							mark(lhs)
						}
					}
					return true
				}
				for i, lhs := range x.Lhs {
					if i < len(x.Rhs) && affordedFloatish(x.Rhs[i], floats, src) {
						mark(lhs)
					}
				}
			}
			return true
		})
		if len(floats) == before {
			return floats
		}
	}
}

// affordedFloatDerivedIn is every float derivation in this package's test
// sources, mapped to the functions it is spelled in.
//
// See affordedFloatDerivations for what counts as one and why the reading is
// syntactic. A string literal anywhere inside an expression takes it out: a
// concatenated message carrying "%.4f" is an ADD between strings and is not
// arithmetic, and there is no cheaper way to tell the two apart without
// running the type checker over a package that imports half the repository.
func affordedFloatDerivedIn(t *testing.T) (map[string][]string, []string, []string) {
	t.Helper()
	src := affordedFloatSourceOf(t)
	in := map[string][]string{}
	for name, file := range src.files {
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			where := name + ":" + fn.Name.Name
			// What holds a float in THIS function, decided before the
			// arithmetic is read. See affordedFloatLocals: this is the half of
			// the reading that text cannot make, and it is why `down * step`
			// over two ratios is a derivation here and was invisible before.
			floats := affordedFloatLocals(fn, src)
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if !affordedIsDerivation(n) {
					return true
				}
				var b strings.Builder
				if err := printer.Fprint(&b, src.fset, n); err != nil {
					t.Fatalf("printing an expression of %s: %v", where, err)
				}
				text := b.String()
				// The text test first, so nothing this census used to see can
				// stop being seen by an inference that decided otherwise.
				if !affordedLooksFloat(text) && !affordedFloatish(n, floats, src) {
					return true
				}
				if affordedHoldsString(n) {
					return true
				}
				if !slices.Contains(in[text], where) {
					in[text] = append(in[text], where)
				}
				return true
			})
		}
	}
	if len(src.files) == 0 {
		t.Fatalf("no _test.go files in this package.\n\n" +
			"This census reads the checks' own arithmetic off their source. A run " +
			"that finds nothing to read reports every derivation single-spelled, " +
			"which is a green run saying the thing it was written to look for is " +
			"absent.")
	}
	return in, src.collide, src.conflate
}

// affordedIsDerivation is whether a node is arithmetic this census counts:
// a binary `* / + -`, or a call into the math package.
func affordedIsDerivation(n ast.Node) bool {
	switch x := n.(type) {
	case *ast.BinaryExpr:
		switch x.Op {
		case token.MUL, token.QUO, token.ADD, token.SUB:
			return true
		}
	case *ast.CallExpr:
		sel, ok := x.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		id, ok := sel.X.(*ast.Ident)
		return ok && id.Name == "math"
	}
	return false
}

// affordedLooksFloat is the first of the two tests for "this is float
// arithmetic": a float64 conversion, a call into math, or a decimal literal in
// the text.
//
// It is asked of the printed expression, so it sees anything float-shaped
// anywhere inside — and it misses an expression over two float VARIABLES,
// `a * b` where both came from somewhere else. That was this census's stated
// limit and it is now the fallback rather than the whole reading: what the
// text cannot say, affordedFloatLocals infers from the assignments. The two
// together are still not go/types — see affordedFloatSource for what they
// cannot reach — but the gap is now the one nobody has a cheap answer for
// rather than the one nobody had looked at.
func affordedLooksFloat(text string) bool {
	if strings.Contains(text, "float64(") || strings.Contains(text, "math.") {
		return true
	}
	for i := 1; i+1 < len(text); i++ {
		if text[i] == '.' && text[i-1] >= '0' && text[i-1] <= '9' &&
			text[i+1] >= '0' && text[i+1] <= '9' {
			return true
		}
	}
	return false
}

// affordedHoldsString is whether a string literal appears anywhere inside an
// expression, which is how a concatenated message is told from arithmetic.
func affordedHoldsString(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(m ast.Node) bool {
		if lit, ok := m.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			found = true
		}
		return !found
	})
	return found
}

// The rule the FMA hazard taught, held over the package's own checks.
//
// Three arms, and they are three different findings:
//
//	spelled twice   a derivation in two functions. This is the hazard itself:
//	                two evaluations of one expression are two numbers the
//	                compiler may round differently, and a comparison between
//	                them is a comparison nobody wrote.
//	not in the      a derivation the source has and this table does not. Not a
//	table           failure of the arithmetic — a decision nobody has taken.
//	                Every entry says whether a comparison rests on it, and that
//	                is the sentence somebody has to write.
//	not in the      a table entry the source no longer has, which is an entry
//	source          describing arithmetic that was reworded or deleted.
func TestNoFloatAComparisonRestsOnIsDerivedTwice(t *testing.T) {
	derived, collide, conflate := affordedFloatDerivedIn(t)

	// # And the tables themselves, which are keyed by a name and nothing else
	//
	// Three of the four tables this census reads a type from conflate anything
	// sharing a name: a method is keyed without its receiver, an imported
	// function by the name the import is used under, and a struct field by the
	// field name alone. The note on affordedFloatSource said so and called it
	// an approximation. What it did not say is whether the approximation was
	// currently costing anything, and nothing asked.
	//
	// It was. Five field names in this package were declared as a float in
	// one struct and as something else in another, so the reading a comment
	// described as a limit was a limit in force, on names this file uses
	// everywhere:
	//
	//	losing    affordedBand's number, and the k-leaf walk's array of
	//	gaining   candidate maxima
	//	chain     affordedBandMeasuredOn's map, and a float in the arm that
	//	          asks whether the composition covers
	//	measured  affordedBracket's int, and a float in two comparison tables
	//	got       affordedTaker's int, and a float64 in widget_test.go
	//
	// # And four of the five were this file's own to rename
	//
	// The census reports and does not resolve, which is right for the general
	// case — telling two same-named declarations apart is what go/types is
	// for. It is not right for a collision where both declarations are in
	// this package and one of them is a local table nobody reads by name: the
	// walk's arrays are widestLosing/widestGaining, and the two comparison
	// tables spell their fields `band` and `composed`. Four renames, no
	// change to a single number, and the approximation narrows from five
	// names to one.
	//
	// What is left is the honest residue and the shape the census cannot fix:
	// `got` is affordedTaker's int here and a float64 in widget_test.go —
	// two files with nothing to do with each other, where neither name is
	// wrong and there is no local edit that separates them. That one stays
	// counted.
	//
	// # And the two tables fail differently, which is why this is two arms
	//
	// The field table is a UNION: a name declared float in any struct marks
	// every selector with it. That is order-independent and always in the
	// direction this census calls safe — an expression that is not arithmetic
	// gets asked for an entry, which is a sentence somebody writes and a
	// reader can see. So it is counted, and the count is in the log line where
	// a reader can watch it grow.
	//
	// The method and import tables are ASSIGNED into. A `changed()` returning
	// a float on one type and an int on another leaves one answer in the table
	// and the other gone, and which one depends on the order the directory
	// listed the files in. One of those two orders makes real float arithmetic
	// invisible, which is the failure this whole test exists for — so that one
	// fails.
	//
	// Reported and not resolved. Telling two same-named declarations apart is
	// what go/types is for, and running it over a package that imports half
	// this repository is the cost this census was written to avoid.
	if len(collide) > 0 {
		t.Errorf("%d name(s) in this census's method or import tables are answered "+
			"differently by more than one declaration:\n\t%s\n\n"+
			"Those tables are keyed by the name alone — a method without its "+
			"receiver, an import by the segment it is used under — and an assignment "+
			"into them REPLACES what was there, so only one of the two answers "+
			"survives and which one is the order the files were read in. If the "+
			"surviving answer is the float, this census asks for an entry describing "+
			"arithmetic that is not there; if it is the other, real float arithmetic "+
			"stops being counted and this test goes quiet about it. Unlike the field "+
			"table below, neither direction is the safe one and neither is stable. "+
			"Rename one of the two.",
			len(collide), strings.Join(collide, "\n\t"))
	}

	said := map[string]string{}
	for _, d := range affordedFloatDerivations {
		if _, ok := derived[d.expr]; !ok {
			t.Errorf("this package's checks no longer derive `%s`, which the census "+
				"describes as %s.\n\n"+
				"An entry for arithmetic that is not there describes nothing, and it "+
				"is the shape that goes stale silently: the expression it was written "+
				"about was reworded or removed, and what is left is a sentence a "+
				"reader will one day take as covering something else.", d.expr, d.what)
		}
		if was, dup := said[d.expr]; dup {
			t.Errorf("the census carries `%s` twice — as %s and as %s.\n\n"+
				"One derivation, one entry: the second is either a copy or a "+
				"disagreement about what the arithmetic is for, and both are worse "+
				"than either sentence alone.", d.expr, was, d.what)
		}
		said[d.expr] = d.what
	}

	twice, missing := []string{}, []string{}
	for expr, where := range derived {
		if len(where) > 1 {
			slices.Sort(where)
			twice = append(twice, fmt.Sprintf("`%s` in %s", expr,
				strings.Join(where, " and ")))
		}
		if _, known := said[expr]; !known {
			missing = append(missing, fmt.Sprintf("`%s` in %s", expr,
				strings.Join(where, ", ")))
		}
	}
	slices.Sort(twice)
	slices.Sort(missing)
	if len(twice) > 0 {
		t.Errorf("%d float derivation(s) are spelled in more than one function:\n"+
			"\t%s\n\n"+
			"Two floats a comparison rests on have to come from one evaluation. Go "+
			"may fuse a multiply and the subtract after it into a single rounding "+
			"and is free to do so at one spelling and not at another — it did, in "+
			"this file, and against a band that is the maximum of a family the "+
			"residual belongs to, one ulp was the whole distance between a bound "+
			"held and a green run reporting a re-sort nobody made. It surfaced in a "+
			"break-test's odd output and nothing would have surfaced it otherwise.\n\n"+
			"Give the expression one name and call it from both places. If the two "+
			"really are independent arithmetic over different populations, they are "+
			"still two spellings a reader has to check by eye — say so in "+
			"affordedFloatDerivations and give one of them its own name anyway.",
			len(twice), strings.Join(twice, "\n\t"))
	}
	if len(missing) > 0 {
		t.Errorf("%d float derivation(s) in this package's checks have no entry in "+
			"affordedFloatDerivations:\n\t%s\n\n"+
			"The census is the place a reader finds out whether a comparison rests "+
			"on a number — which is the question the FMA hazard turned on and the "+
			"question nobody could answer about the second spelling, because nobody "+
			"had looked for one. A new derivation is a decision: say what it is and "+
			"whether anything compares it, or route it through arithmetic that "+
			"already has an entry.",
			len(missing), strings.Join(missing, "\n\t"))
	}

	compared := 0
	for _, d := range affordedFloatDerivations {
		if strings.Contains(d.what, "COMPARED") {
			compared++
		}
	}
	t.Logf("%d float derivations across this package's checks, each spelled in "+
		"exactly one function — %d of them carrying a comparison, which is the "+
		"class the FMA hazard belongs to: two evaluations of one expression are "+
		"two numbers the compiler may round differently, and the residual and its "+
		"band were once exactly that. The rule is held by the arithmetic having one "+
		"name rather than by anybody remembering it.\n\nThe types behind them are "+
		"read off four name-keyed tables, three of which cannot tell two "+
		"declarations sharing a name apart. No method or import name is answered "+
		"two ways — that arm fails, because the surviving answer there is whichever "+
		"file was read last and one of the two orders hides arithmetic. %d FIELD "+
		"name(s) are answered two ways: %s. That table is a union rather than an "+
		"assignment, so every selector with one of those names reads as float "+
		"wherever it appears, which is this census asking for an entry it may not "+
		"need rather than missing one — the direction it is built to err in. It was "+
		"five, and four of those were collisions inside this package between a "+
		"record and a local table; those are renamed. What a number here now means "+
		"is a name two FILES disagree about, which is the case this census cannot "+
		"resolve without go/types. The count is here so a reader can see it move; it "+
		"was a sentence in a comment and nobody was counting",
		len(derived), compared, len(conflate), strings.Join(conflate, ", "))
}
