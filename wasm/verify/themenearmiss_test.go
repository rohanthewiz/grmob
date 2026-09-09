package main

import (
	"fmt"
	"math"
	"reflect"
	"slices"
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
	// And how far an ending may sit from its own prediction before the log line
	// stops calling the classification unmoved.
	//
	// A tenth. The floors are asked one ending at a time and a redistribution
	// that stays above all four of them passes without a word — see
	// affordedTakers — so this is the only place a green run can say the sort
	// has moved. It is a reading rather than an assertion because the residual
	// moves for an honest reason too: the scale is exact only while the walk's
	// shape is unchanged, and a leaf added mid-name shifts which windows crowd
	// by a percent or two. A tenth is well above that and well under the 600%
	// a re-sorted ending shows.
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
var affordedMeasuredOn = struct {
	leaves int
	window int
	ending map[string]int
}{
	leaves: 80,
	window: 6,
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
	then := affordedSetCount(affordedMeasuredOn.leaves, affordedMeasuredOn.window)
	if then == 0 {
		return 0
	}
	return float64(affordedSetCount(leaves, affordedWindowMax)) / float64(then)
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
	return float64(measured) * scale, true
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
	moved := float64(b.measured) * scale
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
		if math.Abs(scale-1) < affordedScaleSame {
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

func TestTheAffordedWidthHoldsItsTwoRelations(t *testing.T) {
	paths := themeLeafPaths(reflect.ValueOf(*core.DefaultTheme), "")
	sort.Strings(paths)
	// The bare names, deduplicated: two parents can hold the same leaf name
	// (Colors.Surface and Colors.Overlay.Surface), and a window carrying it
	// twice would mount one parent with two identical children — a shape the
	// walk that produces these sets can never see, and one whose zero distance
	// is a different check's business.
	names := []string{}
	seen := map[string]bool{}
	for _, path := range paths {
		name := path[strings.LastIndex(path, ".")+1:]
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	// Every window, at every width, under one parent and under two. The second
	// arrangement is what puts leaves in front of the derivation that are NOT
	// candidates for one another — the only comparison the measurement makes is
	// within a parent, and a set with one parent cannot show that `span` and
	// the crowd search read different populations.
	sets := [][]string{}
	for width := 1; width <= affordedWindowMax && width <= len(names); width++ {
		for i := 0; i+width <= len(names); i++ {
			window := names[i : i+width]
			one := make([]string, 0, width)
			two := make([]string, 0, width)
			for j, name := range window {
				one = append(one, "One."+name)
				if j%2 == 0 {
					two = append(two, "Left."+name)
				} else {
					two = append(two, "Right."+name)
				}
			}
			sets = append(sets, one, two)
		}
	}

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
		switch {
		case !set.cappedByReach:
			reached["its own crowding stopped the search"]++
		case set.affordedOpen:
			reached["no width crowds these names at all"]++
		case set.afforded > set.edits:
			reached["the ceiling cost this set a wider threshold"]++
		default:
			reached["the ceiling and the crowding stop in the same place"]++
		}

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
	if len(names) == affordedMeasuredOn.leaves &&
		affordedWindowMax == affordedMeasuredOn.window {
		for _, ending := range endings {
			measured, known := affordedMeasuredOn.ending[ending]
			if !known || reached[ending] == measured {
				continue
			}
			t.Errorf("%q was reached by %d of the %d sets and affordedMeasuredOn "+
				"records %d, over the same %d leaf names at the same window of %d.\n\n"+
				"The walk is a function of those two numbers and they have not moved, "+
				"so it produced the same %d sets in the same order — and which ending "+
				"a set reaches is a function of the set. These two counts are "+
				"therefore the same number or themeLeafSetOf is sorting the walk "+
				"differently than it was when the record was taken.\n\n"+
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
		moved := float64(b.measured) * scale
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
	for _, ending := range endings {
		b := brackets[ending]
		arm := "the stated floor"
		if b.share {
			arm = "a third of its own"
		}
		held = append(held, fmt.Sprintf("%q %d/%d (%s)",
			ending, reached[ending], b.floor, arm))
		predicted, ok := affordedPrediction(ending, scale)
		if !ok || predicted == 0 {
			continue
		}
		if off := math.Abs(float64(reached[ending])-predicted) / predicted; off > worst {
			worst, worstEnding = off, ending
		}
	}
	// Said in the direction the residuals are actually in. "Re-sorted by
	// nothing" is a claim, and a run where an ending is half again its own
	// prediction while every floor is still cleared is the redistribution
	// affordedTakers is about, passing — which is worth a sentence in a green
	// run's line rather than the same words that describe a walk standing
	// still. The floors are one ending at a time and cannot see it.
	resorted := "the walk having been re-sorted by nothing"
	if worst > affordedResidualQuiet {
		resorted = fmt.Sprintf("more than the %.0f%% a walk standing still leaves, "+
			"so some ending is holding sets another one used to reach — every floor "+
			"is still clear and themeLeafSetOf has moved under them",
			affordedResidualQuiet*100)
	}
	t.Logf("%d generated leaf sets over %d distinct leaf names, none of them a "+
		"shape anybody chose, hold afforded >= edits and the open flag's "+
		"parent-of-three — each ending against its own floor, %d of the four on a "+
		"share of its own measurement and the rest on the stated %d, over a "+
		"population at %.2f× the one the record was taken on and partitioning it "+
		"(%d sets, %d accounted for), every ending within %.1f%% of what that scale "+
		"predicts for it (the furthest being %q), which is %s: %s%s",
		len(sets), len(names), onShare, affordedEndingFloor, scale,
		len(sets), census, worst*100, worstEnding, resorted,
		strings.Join(held, ", "), affordedMeasuredNote(len(names), len(sets)))
}
