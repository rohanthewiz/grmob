package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"math"
	"os"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
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
	wide := themeLeafSetOf([]string{"Ramp.Aaaaaa", "Ramp.Bbbbbb", "Ramp.Cccccc"})
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
	pair := themeLeafSetOf([]string{"Duo.Alpha", "Duo.Zulu"})
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

	// And that the message names what capped the threshold, on the arm where
	// it matters: a reader told "nothing is within three edits" will wonder
	// about four, and whether four was available is the difference between a
	// struct that is not crowded and a judgement that refused to look further.
	miss := themeNearMiss(sparse, "Palette.Zulu")
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
	if wideMiss := themeNearMiss(wide, "Ramp.Zzzzzz"); !strings.Contains(
		wideMiss, fmt.Sprintf("would carry %d", wide.afforded)) {
		t.Errorf("themeNearMiss over a set that affords %d does not say so.\n\n"+
			"It said: %s\n\n"+
			"This is the arm where the ceiling is genuinely costing something, and "+
			"the width is what a reader needs to decide whether raising it is worth "+
			"doing. \"These names could carry a wider threshold\" without the number "+
			"is the state this replaced.", wide.afforded, wideMiss)
	}
	if pairMiss := themeNearMiss(pair, "Duo.Mike"); !strings.Contains(
		pairMiss, "no width crowds these names") {
		t.Errorf("themeNearMiss over a two-leaf parent does not say that no width "+
			"crowds it.\n\n"+
			"It said: %s\n\n"+
			"A number there would read as a measurement of where these names stop "+
			"being distinguishable, and there is no such distance: a parent with two "+
			"leaves never produces a crowd. The shape of the set is the answer.",
			pairMiss)
	}
	if crowdedMiss := themeNearMiss(crowded, "Weight.Zzz"); strings.Contains(
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
	n := 0
	for w := 1; w <= window && w <= leaves; w++ {
		n += 2 * (leaves - w + 1)
	}
	return n
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
// messages. See TestNoFloatThisFileComparesIsDerivedTwice for what now stops a
// fifth appearing.
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
// Measured over one 78-name population on an M3, `go test -bench`:
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
// not leave thirteen million allocations behind for four numbers. What the
// measurement settles is the OTHER question — that `-short` is not carrying an
// unexamined cost this could have removed, because the cost is not here. The
// lever that would actually move it is not allocation at all: a k-drop leaves
// most of its windows identical to the full population's, so all but about
// thirty of the 906 classifications per population are re-asked answers. That
// is a second walk to hold against this one, which is a different piece of
// work from a buffer.
func affordedEachSet(names []string, window int, yield func([]string)) {
	if window <= 0 || len(names) == 0 {
		return
	}
	one := make([]string, len(names))
	alt := [2][]string{make([]string, len(names)), make([]string, len(names))}
	for k, name := range names {
		one[k] = "One." + name
		left, right := "Left."+name, "Right."+name
		alt[k%2][k], alt[1-k%2][k] = left, right
	}
	for width := 1; width <= window && width <= len(names); width++ {
		for i := 0; i+width <= len(names); i++ {
			yield(one[i : i+width])
			yield(alt[i%2][i : i+width])
		}
	}
}

// affordedEndingOf is which of the four sentences a set reaches.
//
// The switch has a default arm, so every set reaches exactly one — which is
// the partition every prediction in this file rests on. Lifted out of the test
// loop for the same reason affordedSets was: the band walks eighty populations
// and has to classify them the way the census does, and two copies of a switch
// is two classifications that can drift apart while both look right.
func affordedEndingOf(set themeLeafSet) string {
	switch {
	case !set.cappedByReach:
		return "its own crowding stopped the search"
	case set.affordedOpen:
		return "no width crowds these names at all"
	case set.afforded > set.edits:
		return "the ceiling cost this set a wider threshold"
	default:
		return "the ceiling and the crowding stop in the same place"
	}
}

// affordedCensusOf is the whole census — the walk, the derivation and the
// classification — over a given list of distinct leaf names.
func affordedCensusOf(names []string, window int) map[string]int {
	census := map[string]int{}
	affordedCensusInto(census, names, window)
	return census
}

// affordedCensusInto is that census taken into a map the caller owns.
//
// The band's walk takes three thousand of these and keeps none of them, so the
// map is cleared and refilled rather than allocated per population — the same
// move the `smaller` buffer in affordedKLeafBand makes, for the same reason.
// Cleared rather than assumed empty: a caller that passed a full map would
// otherwise get a census of two populations added together, which is a number
// that looks exactly like a count.
func affordedCensusInto(census map[string]int, names []string, window int) {
	clear(census)
	affordedEachSet(names, window, func(set []string) {
		census[affordedEndingOf(themeLeafSetOf(set))]++
	})
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
func affordedOneLeafBand(names []string, endings []string) map[string]affordedBand {
	return affordedKLeafBand(names, endings, 1)
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
// So the band for a step is measured for that step. This is what makes it
// possible: the family is still enumerable at k = 2 (3160 populations against
// 80), so the two-leaf arms are assertions on the same footing as the one-leaf
// ones and nothing has been scaled.
//
// # And where it stops
//
// C(80, k) walks: 80, 3160, 82160, 1.6M. Three leaves is minutes and four is
// an hour, so k >= 3 is not enumerable here at any price a test can pay, and
// the log line's sentence stands for those — with a reason now attached to it
// rather than an absence: the one-leaf band cannot be scaled to reach them and
// the family cannot be walked.
//
// # Walked in parallel, and deterministically
//
// 3160 walks is eight seconds on one core and about one across the machine,
// which is the difference between a reading that runs on every green run and
// one nobody would keep. The reduction is by (value, then lowest combination
// index), which is what the sequential scan produced — the first combination
// to reach the maximum — so the recorded `losingAt` does not depend on how the
// work was split.
func affordedKLeafBand(names, endings []string, k int) map[string]affordedBand {
	bands := map[string]affordedBand{}
	for _, ending := range endings {
		bands[ending] = affordedBand{}
	}
	if k <= 0 || k >= len(names) {
		return bands
	}
	full := affordedCensusOf(names, affordedWindowMax)
	whole := affordedSetCount(len(names), affordedWindowMax)
	part := affordedSetCount(len(names)-k, affordedWindowMax)
	if whole == 0 || part == 0 {
		return bands
	}
	// Every k-drop is the same size, so the two scales are constants rather
	// than a division inside the walk. Through affordedRatioOf, which is also
	// what affordedScale divides with: `down` and the scale the test reads a
	// residual against are the same number, and the k-leaf assertions are only
	// proofs while they are.
	down := affordedRatioOf(part, whole)
	up := affordedRatioOf(whole, part)

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
	type result struct{ losing, gaining map[string]best }
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
			r := result{losing: map[string]best{}, gaining: map[string]best{}}
			for _, ending := range endings {
				r.losing[ending] = best{at: -1}
				r.gaining[ending] = best{at: -1}
			}
			// Strided rather than blocked, so a family whose expensive
			// populations sit together does not land on one worker.
			smaller := make([]string, 0, len(names)-k)
			// And one census map for the whole share, refilled per population.
			// See affordedCensusInto: three thousand maps allocated and thrown
			// away is the same cost as the sets were, one level up.
			census := map[string]int{}
			for c := w; c < len(combos); c += workers {
				drop := combos[c]
				// Rebuilt into a buffer this goroutine owns. affordedSets
				// takes sub-slices of what it is given, so a walk over a
				// shared backing array would read names its population does
				// not have.
				smaller = smaller[:0]
				at := 0
				for i, name := range names {
					if at < k && drop[at] == i {
						at++
						continue
					}
					smaller = append(smaller, name)
				}
				affordedCensusInto(census, smaller, affordedWindowMax)
				for _, ending := range endings {
					if off, _, ok := affordedResidualOf(
						full[ending], census[ending], down); ok {
						r.losing[ending] = worst(r.losing[ending], best{off, c})
					}
					if off, _, ok := affordedResidualOf(
						census[ending], full[ending], up); ok {
						r.gaining[ending] = worst(r.gaining[ending], best{off, c})
					}
				}
			}
			results[w] = r
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
	for _, ending := range endings {
		losing, gaining := best{at: -1}, best{at: -1}
		for _, r := range results {
			if r.losing == nil {
				continue // a worker with no share of the combinations
			}
			losing = worst(losing, r.losing[ending])
			gaining = worst(gaining, r.gaining[ending])
		}
		bands[ending] = affordedBand{
			losing: losing.value, losingAt: spell(losing.at),
			gaining: gaining.value, gainingAt: spell(gaining.at),
		}
	}
	return bands
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
func affordedChainBoundOf(names, endings []string, k int) map[string]affordedBand {
	bound := map[string]affordedBand{}
	for _, ending := range endings {
		bound[ending] = affordedBand{}
	}
	if k <= 0 || k >= len(names) {
		return bound
	}
	// The running products, started at 1 because ∏(1+b_i) is what composes and
	// the bound is that product less one. Kept as (1+b) rather than as b so
	// the multiply below is the identity above and not a rearrangement of it.
	losing := make(map[string]float64, len(endings))
	gaining := make(map[string]float64, len(endings))
	losingAt := make(map[string][]string, len(endings))
	gainingAt := make(map[string][]string, len(endings))
	for _, ending := range endings {
		losing[ending], gaining[ending] = 1, 1
	}
	for step := 0; step < k; step++ {
		// The representative population of size n−step: these names with the
		// first `step` dropped. A rule, so that the chain is reproducible and
		// the same on every run — and the whole of what makes this a bound
		// over one chain rather than over the family. See the note above.
		at := affordedKLeafBand(names[step:], endings, 1)
		for _, ending := range endings {
			b := at[ending]
			losing[ending] *= 1 + b.losing
			gaining[ending] *= 1 + b.gaining
			losingAt[ending] = append(losingAt[ending], b.losingAt)
			gainingAt[ending] = append(gainingAt[ending], b.gainingAt)
		}
	}
	for _, ending := range endings {
		bound[ending] = affordedBand{
			losing:    losing[ending] - 1,
			gaining:   gaining[ending] - 1,
			losingAt:  strings.Join(losingAt[ending], " then "),
			gainingAt: strings.Join(gainingAt[ending], " then "),
		}
	}
	return bound
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
// One and two, and the reason it stops there is a cost rather than a shape:
// C(80,k) is 80, 3160, 82160, 1.6M, so three leaves is minutes and four is an
// hour. See affordedKLeafBand for what fails when a bigger step is scaled from
// a smaller one instead, and affordedChainBoundOf for the composition that was
// supposed to get past that and does not.
//
// A list rather than two spelled-out blocks because every reading below is the
// same three arms per step, and two copies of them is how the two-leaf record
// came to state a population the one-leaf record also stated.
var affordedBandSteps = []int{1, 2}

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
	},
	// The composed two-step bound, recorded so the finding is a pair of
	// numbers a run re-derives rather than a sentence in a note. Read against
	// step[2] by the arm that says the composition does not cover it.
	chain: map[string]affordedBand{
		"its own crowding stopped the search":                 {losing: 0.5613, gaining: 0.7755},
		"no width crowds these names at all":                  {losing: 0.0004, gaining: 0.0004},
		"the ceiling cost this set a wider threshold":         {losing: 0.0597, gaining: 0.0580},
		"the ceiling and the crowding stop in the same place": {losing: 0.4792, gaining: 0.6279},
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
	step map[int]map[string]affordedBand, chain map[string]affordedBand) string {
	var b strings.Builder
	b.WriteString("\n\nThis run's walk, in the shape the records are written in " +
		"— paste it once the derivation has been read and the new sort is the " +
		"one that was wanted:\n\n")
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
	if len(step) == 0 {
		return b.String()
	}
	b.WriteString("\nand the bands over those names:\n\n\tstep: " +
		"map[int]map[string]affordedBand{\n")
	for _, k := range affordedBandSteps {
		bands, taken := step[k]
		if !taken {
			fmt.Fprintf(&b, "\t\t// %d: not re-derived on this run (-short)\n", k)
			continue
		}
		fmt.Fprintf(&b, "\t\t%d: {\n", k)
		for _, ending := range keys {
			band := bands[ending]
			fmt.Fprintf(&b, "\t\t\t%q: {losing: %.4f, gaining: %.4f},\n",
				ending, affordedBandRounded(band.losing),
				affordedBandRounded(band.gaining))
		}
		b.WriteString("\t\t},\n")
	}
	b.WriteString("\t},\n")
	if len(chain) > 0 {
		b.WriteString("\tchain: map[string]affordedBand{\n")
		for _, ending := range keys {
			band := chain[ending]
			fmt.Fprintf(&b, "\t\t%q: {losing: %.4f, gaining: %.4f},\n",
				ending, affordedBandRounded(band.losing),
				affordedBandRounded(band.gaining))
		}
		b.WriteString("\t},\n")
	}
	return b.String()
}

func TestTheAffordedWidthHoldsItsTwoRelations(t *testing.T) {
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
	endings := []string{
		"its own crowding stopped the search",
		"no width crowds these names at all",
		"the ceiling cost this set a wider threshold",
		"the ceiling and the crowding stop in the same place",
	}
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
		recordCensus := affordedCensusOf(affordedMeasuredOn.names,
			affordedMeasuredOn.window)
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
	bands := affordedOneLeafBand(names, endings)
	// Which of the two one-leaf families this run and the record are in, decided
	// by the NAMES rather than by how many of them there are. See
	// affordedMissingLeaf: an assertion against a band is only a proof while
	// the bracketed population is a member of the family the band is the
	// maximum over, and "one apart by count" does not say that — a leaf added
	// and another renamed is +1 with no subset anywhere.
	//
	//	added, oneLong    the record's names are this run's with one dropped.
	//	                  The band over THIS run's names is the family.
	//	removed, oneShort this run's names are the record's with one dropped.
	//	                  The band over the RECORD's names is the family, and
	//	                  this run cannot walk it — which is why the removal had
	//	                  no assertion until the record carried its names.
	added, oneLong := affordedMissingLeaf(names, affordedMeasuredOn.names)
	removed, oneShort := affordedMissingLeaf(affordedMeasuredOn.names, names)
	// And the same question for a step of two, which is the largest step whose
	// family can still be walked. See affordedKLeafBand: 3160 populations at
	// two and eighty-two thousand at three, and the one-leaf band cannot be
	// scaled to stand in for either because the drift is not linear in the
	// number of leaves moved.
	addedTwo, twoLong := affordedMissingLeaves(names, affordedMeasuredOn.names)
	twoLong = twoLong && len(addedTwo) == 2
	removedTwo, twoShort := affordedMissingLeaves(affordedMeasuredOn.names, names)
	twoShort = twoShort && len(removedTwo) == 2
	// And the bands over the RECORD's own names, one per step, which are pure
	// functions of them and therefore measurable on every run — the same
	// argument the record's census re-derivation rests on, applied to the
	// record one level up. Each buys two things at once: the family a run that
	// many leaves SHORT of the record belongs to, and a reading on
	// affordedBandMeasuredOn that does not need core.Theme to have stood still.
	//
	// The two-leaf step is the one reading here with a cost worth naming —
	// about five seconds against the rest of this test's half — so `-short`
	// gives it up, and gives up exactly three things with it: the four numbers
	// affordedKLeafBand's note argues from stop being re-derived, the widest of
	// the readings on themeLeafSetOf stops running, and the chain bound below
	// has nothing to be held against. All three are re-taken by the next full
	// run and the log line says so. A run that is actually two leaves from the
	// record measures it either way, because that is the run it is the family
	// for.
	twoBandTaken := !testing.Short() || twoShort || twoLong
	recordStep := map[int]map[string]affordedBand{}
	for _, k := range affordedBandSteps {
		if k > 1 && !twoBandTaken {
			continue // -short; said in the log line
		}
		recordStep[k] = affordedKLeafBand(affordedMeasuredOn.names, endings, k)
	}
	// Named for the two arms that read them: a run one leaf short of the record
	// is bracketed by the first and a run two leaves short by the second.
	recordBands, recordTwoBands := recordStep[1], recordStep[2]

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
		if got == nil {
			continue // -short; said in the log line
		}
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
	// One ulp apart on this machine, against a tolerance of a part in 1e12.
	// Asserted rather than assumed for the reason the FMA hazard taught: this
	// file has already been caught once by two roundings of one number, and it
	// was found in a break-test's odd output rather than by anything looking.
	{
		n, w := affordedMeasuredOn.leaves, affordedMeasuredOn.window
		down := affordedRatioOf(affordedSetCount(n-1, w), affordedSetCount(n, w))
		step := affordedRatioOf(affordedSetCount(n-2, w), affordedSetCount(n-1, w))
		whole := affordedRatioOf(affordedSetCount(n-2, w), affordedSetCount(n, w))
		composed := down * step
		if off := math.Abs(composed-whole) / whole; off > affordedScaleCompose {
			t.Errorf("the two-leaf scale is %.17g taken directly and %.17g taken as "+
				"one leaf then another — %g apart, against a tolerance of %g.\n\n"+
				"Those are the same rational number: the set count at %d cancels. The "+
				"chain bound below multiplies the one-leaf bands on exactly that "+
				"identity, and the band it is compared against took its residuals "+
				"against the direct spelling, so a difference here is the two of them "+
				"dividing by different numbers and the comparison being between two "+
				"readings rather than one. If the walk's set count has stopped being "+
				"the product it is here, the composition is not an identity any more "+
				"and the chain bound is not even the right shape of wrong.",
				whole, composed, off, affordedScaleCompose, n-1)
		}
	}

	// Which of the eight composed numbers came in UNDER the measured band,
	// which is the finding the composition exists here to carry. Empty under
	// -short, where nothing was composed and the log line says so.
	chainSaid := []string{}
	// # And the composition that was supposed to reach past the enumerable
	// steps
	//
	// The steps stop at two because C(80,3) is 82160 walks. The obvious way
	// past that is not to enumerate at all: dropping k leaves is k one-leaf
	// drops, the residuals multiply exactly (see affordedChainBoundOf), so
	// ∏(1+b_i)−1 is a bound wherever the b_i bound each step — and those are
	// one-leaf bands at n, n−1, …, which is k×80 walks rather than C(80,k).
	//
	// It does not work, and k = 2 is the one step where the composed number and
	// a measured one can be put side by side. Both are re-derived here and the
	// comparison is the finding: the arm below fires when the composition
	// covers all eight, because covering is what would make it usable and
	// today it does not.
	var chain map[string]affordedBand
	if recordTwoBands != nil {
		chain = affordedChainBoundOf(affordedMeasuredOn.names, endings, 2)
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
			for _, d := range []struct {
				what            string
				chain, measured float64
			}{
				{"losing", g.losing, m.losing},
				{"gaining", g.gaining, m.gaining},
			} {
				if affordedBandRounded(d.chain) >= affordedBandRounded(d.measured) {
					continue
				}
				under = append(under, fmt.Sprintf(
					"%q %s composes to %.2f%% against a measured %.2f%%",
					ending, d.what, affordedPercent(d.chain), affordedPercent(d.measured)))
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

	// # And the arm that runs when somebody has just added a field
	//
	// The exact comparison above is the strongest reading in this file and it
	// runs only while the population is untouched — which is the state a
	// re-measure leaves and NOT the state anybody editing core.Theme is in. The
	// moment the struct gains a leaf that arm goes silent and what is left is
	// four floors that cannot see a re-sort by construction, which is the state
	// affordedTakers and the exact arm were both written because of.
	//
	// A leaf ADDED is the one move where the gap can be closed without a
	// tolerance anybody chose, and the reason is the band's construction. The
	// band drops each of THIS run's names in turn, so when this run has one
	// leaf more than the record, the record's own population is a member of
	// that family — these names minus the added leaf — and its residual is one
	// of the numbers the band is the maximum of. Not "about the same size as":
	// one of them. So a residual over the band is not a population that moved
	// unusually far; it is arithmetic that cannot happen while themeLeafSetOf
	// sorts the walk the way it did when the record was taken.
	//
	// # And the other direction, which used to have no argument
	//
	// A run with one leaf FEWER cannot walk the record's population from its
	// own names — the removed name is not here to put back — so the band this
	// run can measure is the 78↔79 step standing in for the 79↔80 step in
	// force, and a bound whose family does not contain the case it brackets is
	// a tolerance wearing a proof's clothes. That was the reason the removal
	// stayed a reading in the log line, and the reason was sound.
	//
	// What it rested on was the record being four counts and an integer. With
	// the names recorded the record's population is walkable on any run, so the
	// band over THOSE names is measurable — and this run's population, being
	// the record's names with one dropped, is a member of that family by the
	// same argument the addition uses in the other direction. The arm below is
	// therefore an assertion and not a tolerance, and its premise is checked by
	// name rather than inferred from a count.
	//
	// The premise of the ADDED arm is checked the same way, and it was not
	// before: `len(names) == leaves+1` is true of a run that added one leaf and
	// renamed another, where the record's population is not this run's names
	// minus anything and the band is a bound over the wrong family.
	if oneLong && affordedWindowMax == affordedMeasuredOn.window {
		for _, ending := range endings {
			measured, known := affordedMeasuredOn.ending[ending]
			if !known {
				continue // reported by the key-set arms above
			}
			off, predicted, ok := affordedResidualOf(measured, reached[ending], scale)
			if !ok {
				continue
			}
			band, why := affordedBandFor(bands, 1, len(names), "this run's names",
				ending, true)
			if off <= band {
				continue
			}
			t.Errorf("%q was reached by %d of the %d sets, and a population that had "+
				"only gained a leaf predicts about %.0f — %.1f%% out, against a band "+
				"of %.2f%%: %s.\n\n"+
				"This run has %d leaf names and affordedMeasuredOn was taken over %d "+
				"at the same window, and the record's names are this run's with %q "+
				"dropped — which is exactly one of the %d "+
				"populations the band was measured over. The residual above is "+
				"therefore a member of the family the band is the largest of, and it "+
				"cannot exceed it while themeLeafSetOf sorts the walk the way it did "+
				"when the record was taken. The whole census is %v against a record "+
				"of %v.\n\n"+
				"So this is the re-sort the per-ending floors cannot see, arriving on "+
				"the population an edit to core.Theme actually leaves. If the new "+
				"sort is what was wanted, re-take affordedMeasuredOn and "+
				"affordedBandMeasuredOn together from the log line — and read the "+
				"derivation first, because a record re-taken over a classification "+
				"that moved by accident records the accident as the baseline.",
				ending, reached[ending], len(sets), predicted, affordedPercent(off),
				affordedPercent(band), why, len(names), affordedMeasuredOn.leaves, added, len(names),
				reached, affordedMeasuredOn.ending)
		}
	}

	// And the same arm the other way round, over the band the record's own
	// names carry. See affordedOneLeafBand: dropping each of the RECORD's
	// eighty names in turn produces eighty populations of seventy-nine, and
	// this run — the record's names with the removed one gone — is one of them.
	// Its residual
	// against the record is therefore one of the numbers `losing` is the
	// maximum of, and a residual above it cannot happen while themeLeafSetOf
	// sorts the walk the way it did.
	if oneShort && affordedWindowMax == affordedMeasuredOn.window {
		for _, ending := range endings {
			measured, known := affordedMeasuredOn.ending[ending]
			if !known {
				continue // reported by the key-set arms above
			}
			off, predicted, ok := affordedResidualOf(measured, reached[ending], scale)
			if !ok {
				continue
			}
			band, why := affordedBandFor(recordBands, 1, affordedMeasuredOn.leaves,
				"the record's names", ending, false)
			if off <= band {
				continue
			}
			t.Errorf("%q was reached by %d of the %d sets, and a population that had "+
				"only lost a leaf predicts about %.0f — %.1f%% out, against a band "+
				"of %.2f%%: %s.\n\n"+
				"This run has %d leaf names and affordedMeasuredOn was taken over %d "+
				"at the same window, and this run's names are the record's with %q "+
				"dropped — which is exactly one of the %d populations that band was "+
				"measured over. The residual above is therefore a member of the "+
				"family the band is the largest of, and it cannot exceed it while "+
				"themeLeafSetOf sorts the walk the way it did when the record was "+
				"taken. The whole census is %v against a record of %v.\n\n"+
				"This is the reading a run one leaf SHORT did not have. The band this "+
				"run can measure from its own names is the 78↔79 step and the step in "+
				"force is 79↔80, so until the record carried the names it was taken "+
				"over there was no family here to be a member of. If the new sort is "+
				"what was wanted, re-take affordedMeasuredOn and "+
				"affordedBandMeasuredOn together from the log line — and read the "+
				"derivation first, because a record re-taken over a classification "+
				"that moved by accident records the accident as the baseline.",
				ending, reached[ending], len(sets), predicted, affordedPercent(off),
				affordedPercent(band), why, len(names), affordedMeasuredOn.leaves, removed,
				affordedMeasuredOn.leaves, reached, affordedMeasuredOn.ending)
		}
	}

	// # And the same pair of arms for a step of two
	//
	// The step past one leaf used to have nothing: the log line said an ending
	// outside the band was "the absence of a finding rather than one", which is
	// honest and is not a measurement. What was missing was a family, and the
	// obvious one — the one-leaf band times two — turned out not to be a bound
	// at all. See affordedKLeafBand: gaining runs to 2.6× the one-leaf figure
	// on the two small endings, so a doubled band would have called an honest
	// two-field edit a re-sort, which is the failure the measured one-leaf band
	// replaced in the first place, arriving in its own extension.
	//
	// So two leaves are measured rather than scaled, over their own 3160
	// populations, and these arms are assertions on the same footing as the
	// one-leaf pair: the premise is the subset, checked by name, and this run's
	// population is then a member of the family the band is the maximum of.
	if twoLong && affordedWindowMax == affordedMeasuredOn.window {
		// The family is drops of two from THIS run's names, one of which is
		// the record's population. Measured here rather than above because it
		// is a band about this run and nothing else reads it.
		runTwoBands := affordedKLeafBand(names, endings, 2)
		for _, ending := range endings {
			measured, known := affordedMeasuredOn.ending[ending]
			if !known {
				continue // reported by the key-set arms above
			}
			off, predicted, ok := affordedResidualOf(measured, reached[ending], scale)
			if !ok {
				continue
			}
			band, why := affordedBandFor(runTwoBands, 2,
				affordedDropCount(len(names), 2), "this run's names", ending, true)
			if off <= band {
				continue
			}
			t.Errorf("%q was reached by %d of the %d sets, and a population that had "+
				"only gained two leaves predicts about %.0f — %.1f%% out, against a "+
				"band of %.2f%%: %s.\n\n"+
				"This run has %d leaf names and affordedMeasuredOn was taken over %d "+
				"at the same window, and the record's names are this run's with %s "+
				"dropped — which is exactly one of the %d populations that band was "+
				"measured over. The residual is therefore a member of the family the "+
				"band is the largest of, and it cannot exceed it while themeLeafSetOf "+
				"sorts the walk the way it did when the record was taken. The whole "+
				"census is %v against a record of %v.\n\n"+
				"Note that this band is MEASURED for a two-leaf step and is not twice "+
				"the one-leaf one: the drift is not linear in the number of leaves "+
				"moved, and in the gaining direction it is worse than linear. If the "+
				"new sort is what was wanted, re-take affordedMeasuredOn and both of "+
				"affordedBandMeasuredOn's steps from the log line — and read the "+
				"derivation first.",
				ending, reached[ending], len(sets), predicted, affordedPercent(off),
				affordedPercent(band), why, len(names), affordedMeasuredOn.leaves,
				strings.Join(addedTwo, " and "), affordedDropCount(len(names), 2),
				reached, affordedMeasuredOn.ending)
		}
	}
	if twoShort && affordedWindowMax == affordedMeasuredOn.window {
		for _, ending := range endings {
			measured, known := affordedMeasuredOn.ending[ending]
			if !known {
				continue // reported by the key-set arms above
			}
			off, predicted, ok := affordedResidualOf(measured, reached[ending], scale)
			if !ok {
				continue
			}
			band, why := affordedBandFor(recordTwoBands, 2,
				affordedDropCount(affordedMeasuredOn.leaves, 2), "the record's names",
				ending, false)
			if off <= band {
				continue
			}
			t.Errorf("%q was reached by %d of the %d sets, and a population that had "+
				"only lost two leaves predicts about %.0f — %.1f%% out, against a "+
				"band of %.2f%%: %s.\n\n"+
				"This run has %d leaf names and affordedMeasuredOn was taken over %d "+
				"at the same window, and this run's names are the record's with %s "+
				"dropped — which is exactly one of the %d populations that band was "+
				"measured over. The residual is therefore a member of the family the "+
				"band is the largest of, and it cannot exceed it while themeLeafSetOf "+
				"sorts the walk the way it did when the record was taken. The whole "+
				"census is %v against a record of %v.\n\n"+
				"Note that this band is MEASURED for a two-leaf step and is not twice "+
				"the one-leaf one. If the new sort is what was wanted, re-take "+
				"affordedMeasuredOn and both of affordedBandMeasuredOn's steps from "+
				"the log line — and read the derivation first.",
				ending, reached[ending], len(sets), predicted, affordedPercent(off),
				affordedPercent(band), why, len(names), affordedMeasuredOn.leaves,
				strings.Join(removedTwo, " and "),
				affordedDropCount(affordedMeasuredOn.leaves, 2), reached,
				affordedMeasuredOn.ending)
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
	// And whose names that band was measured over. A run one leaf SHORT of the
	// record is bracketed by the band over the RECORD's names — that is the
	// family it belongs to and the one the arm above asserts against — so the
	// sentence has to read the same numbers the assertion did, or a green run
	// would be reporting a different bound from the one in force.
	bandsFor, bandOver, bandWhose := bands, len(names), "this run's names"
	bandStep := 1
	if oneShort {
		bandsFor, bandOver, bandWhose =
			recordBands, affordedMeasuredOn.leaves, "the record's names"
	}
	// A run two leaves SHORT is bracketed by the two-leaf band over the
	// record's names, which is the family it belongs to and the one the arm
	// above asserted against. The two-leaf LONG case has its own band measured
	// inside that arm and not here: it is a band about this run's names and
	// nothing else reads it, so the sentence stays with the one the readings
	// share.
	if twoShort && recordTwoBands != nil {
		bandsFor, bandOver, bandWhose, bandStep =
			recordTwoBands, affordedDropCount(affordedMeasuredOn.leaves, 2),
			"the record's names", 2
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
	case oneLong:
		reading = fmt.Sprintf("the census against the record within the band a "+
			"single ADDED leaf is worth to each ending, asserted — the record's own "+
			"population is this run's names with %q dropped, so its residual is a "+
			"member of the family that band is the largest of", added)
	case oneShort:
		reading = fmt.Sprintf("the census against the record within the band a "+
			"single REMOVED leaf is worth to each ending, asserted over the band "+
			"measured on the RECORD's names — this run's population is those names "+
			"with %q dropped, so its residual is a member of the family that band "+
			"is the largest of. This run cannot walk that family from its own "+
			"names; it is walkable because the record carries the names it was "+
			"taken over", removed)
	case distance == 0:
		reading = fmt.Sprintf("the record's re-walk and the four floors: this run "+
			"has the record's %d leaf names by COUNT and not by name — %d of them "+
			"are not here — so a leaf was renamed, this run's counts are a walk "+
			"over different strings, and comparing them against the record exactly "+
			"would report a re-sort nobody made",
			affordedMeasuredOn.leaves, strangers)
	case twoLong:
		reading = fmt.Sprintf("the census against the record within the band TWO "+
			"ADDED leaves are worth to each ending, asserted — the record's own "+
			"population is this run's names with %s dropped, so its residual is a "+
			"member of the family that band is the largest of. The band is measured "+
			"for a two-leaf step over all %d of them and is not twice the one-leaf "+
			"one, which is not a bound",
			strings.Join(addedTwo, " and "), affordedDropCount(len(names), 2))
	case twoShort:
		reading = fmt.Sprintf("the census against the record within the band TWO "+
			"REMOVED leaves are worth to each ending, asserted over the band "+
			"measured on the RECORD's names — this run's population is those names "+
			"with %s dropped, so its residual is a member of the family that band "+
			"is the largest of, measured over all %d of them",
			strings.Join(removedTwo, " and "),
			affordedDropCount(affordedMeasuredOn.leaves, 2))
	case distance == 1 || distance == -1 || distance == 2 || distance == -2:
		reading = fmt.Sprintf("the record's re-walk and the four floors: this run "+
			"is %d leaves from the record by COUNT and not by name — %d of the "+
			"record's %d names are not here — so neither population is the other "+
			"with leaves merely dropped, and a band whose family does not contain "+
			"the case it brackets is a tolerance wearing a proof's clothes",
			distance, strangers, affordedMeasuredOn.leaves)
	default:
		reading = fmt.Sprintf("the record's re-walk and the four floors: this run "+
			"is %d leaves from the record's %d, and the bands measured here are the "+
			"ONE- and TWO-leaf steps. Neither can be scaled to reach %d — the drift "+
			"is sub-linear losing and worse than linear gaining — and the family "+
			"cannot be walked either: dropping %d of %d names is %d populations "+
			"against the 3160 two of them make. So an ending outside the band below "+
			"is the absence of a finding rather than one",
			distance, affordedMeasuredOn.leaves, distance, distance,
			affordedMeasuredOn.leaves,
			affordedDropCount(affordedMeasuredOn.leaves, distance))
	}
	// And what the two-leaf reading came to, or that it was skipped. The four
	// ratios are the argument against `band × k` and the eight comparisons
	// below them are the argument against composing one, so a green run carries
	// both rather than leaving them in comments nobody re-derives.
	twoBandNote := ""
	if !twoBandTaken {
		twoBandNote = ". The two-leaf band was NOT re-derived on this run " +
			"(-short), so affordedBandMeasuredOn.step[2], the composed chain bound " +
			"beside it and the 3160-population reading on themeLeafSetOf are the " +
			"previous full run's"
	} else {
		ratios := make([]string, 0, len(endings))
		for _, ending := range endings {
			one, two := recordBands[ending], recordTwoBands[ending]
			losing := affordedStepRatio(two.losing, one.losing)
			gaining := affordedStepRatio(two.gaining, one.gaining)
			ratios = append(ratios, fmt.Sprintf("%q ±%.2f%%/%.2f%% (%.2f×/%.2f×)",
				ending, affordedPercent(two.losing), affordedPercent(two.gaining),
				losing, gaining))
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
		twoBandNote = fmt.Sprintf(". Two leaves over all %d pairs of the record's "+
			"names move each ending by %s — losing under twice the one-leaf figure "+
			"and gaining over it, which is why the two-leaf band is measured rather "+
			"than scaled from the one-leaf one. Multiplying the one-leaf bands along "+
			"a chain instead — the construction that would cost 240 walks rather "+
			"than %d and is the only one that could reach k = 3 — %s",
			affordedDropCount(affordedMeasuredOn.leaves, 2), strings.Join(ratios, ", "),
			affordedDropCount(affordedMeasuredOn.leaves, 2), composed)
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
		fresh := map[int]map[string]affordedBand{}
		for _, k := range affordedBandSteps {
			fresh[k] = affordedKLeafBand(names, endings, k)
		}
		t.Log(affordedRetakeSource(names, reached, fresh,
			affordedChainBoundOf(names, endings, 2)))
	}

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
		"%d-leaf step, measured over %s%s%s",
		len(sets), len(names), onShare, affordedEndingFloor, scale,
		len(sets), census, affordedPercent(worst), worstEnding, resorted,
		len(affordedMeasuredOn.names), reading,
		strings.Join(held, ", "), strings.Join(bandSaid, ", "), bandStep, bandWhose,
		twoBandNote, affordedMeasuredNote(len(names), len(sets)))
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
// throughout — `down * step`, `1 + b.losing`, `losing[ending] - 1` — so under
// the old reading none of the composition would have been censused at all.
// That is the point rather than a bonus: the rule is only worth holding over
// the arithmetic there is.
//
// The rule the table enforces is the one the FMA hazard taught: TWO FLOATS
// THAT ARE COMPARED MUST COME FROM ONE EVALUATION. What that reduces to
// syntactically is that no derivation is spelled in two functions — a
// derivation in one place cannot disagree with itself.
var affordedFloatDerivations = []struct{ expr, what string }{
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
	{"1 + b.losing",
		"one step's factor in the chain composition, losing. The identity is " +
			"∏(1+b_i)−1, so the band is carried as a factor and not as a residual " +
			"until the product is finished"},
	{"1 + b.gaining", "the same, gaining — see affordedChainBoundOf"},
	{"losing[ending] - 1",
		"the finished chain product back to a residual, losing. COMPARED: this " +
			"is the number held against the measured two-leaf band, and against " +
			"affordedBandMeasuredOn.chain"},
	{"gaining[ending] - 1", "the same, gaining"},
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
//	         declared float64. Methods are not read, and neither is anything
//	         in another package: `palette.Ratio(a, b) * 2` is invisible unless
//	         the expression carries one of the three text marks, which the two
//	         entries that use it do.
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
	field   map[string]bool
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
		field:   map[string]bool{},
	}
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
				if !affordedFloatType(f.Type) {
					continue
				}
				for _, name := range f.Names {
					src.field[name.Name] = true
				}
			}
			return true
		})
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue // methods are not read; see the note above
			}
			var results []bool
			if fn.Type.Results != nil {
				for _, f := range fn.Type.Results.List {
					n := len(f.Names)
					if n == 0 {
						n = 1
					}
					for i := 0; i < n; i++ {
						results = append(results, affordedFloatType(f.Type))
					}
				}
			}
			src.returns[fn.Name.Name] = results
		}
	}
	return src
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
			id, ok := x.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if id.Name == "float64" || id.Name == "float32" {
				found = true
			}
			if r, known := src.returns[id.Name]; known && len(r) > 0 && r[0] {
				found = true
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
			if !affordedFloatType(f.Type) {
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
func affordedFloatDerivedIn(t *testing.T) map[string][]string {
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
	return in
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
	derived := affordedFloatDerivedIn(t)

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
		"name rather than by anybody remembering it",
		len(derived), compared)
}
