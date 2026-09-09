package main

import (
	"fmt"
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
	affordedEndingShare = 3
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

// affordedEndingBracket is the floor one ending is held to, and why it is that
// number.
//
// The larger of the stated floor and a share of what this ending scored when it
// was measured: the constant is what keeps a small arm from sliding to 1, and
// the share is what keeps a large one from losing most of itself unremarked.
// Returns the reason as well as the number, because a message that prints a
// bound without saying which of the two produced it leaves a reader unable to
// tell "this ending is thin" from "this ending has collapsed".
func affordedEndingBracket(ending string) (int, string) {
	measured, known := affordedMeasuredOn.ending[ending]
	if !known {
		return affordedEndingFloor, fmt.Sprintf(
			"the stated floor of %d — affordedMeasuredOn carries no reading for this "+
				"ending, so there is no share of a measurement to take",
			affordedEndingFloor)
	}
	share := measured / affordedEndingShare
	if share <= affordedEndingFloor {
		return affordedEndingFloor, fmt.Sprintf(
			"the stated floor of %d, which is above the %d that a third of this "+
				"ending's own measurement (%d) comes to",
			affordedEndingFloor, share, measured)
	}
	return share, fmt.Sprintf(
		"a third of the %d this ending scored when affordedMeasuredOn was taken",
		measured)
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
		floor, why := affordedEndingBracket(ending)
		if reached[ending] < floor {
			t.Errorf("%d of the %d generated sets ended with %q, against a floor of "+
				"%d — %s.\n\n"+
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
				reached[ending], len(sets), ending, floor, why, reached,
				affordedWindowMax, affordedMeasuredNote(len(names), len(sets)))
		}
	}
	t.Logf("%d generated leaf sets over %d distinct leaf names, none of them a "+
		"shape anybody chose, hold afforded >= edits and the open flag's "+
		"parent-of-three: %v — each ending against its own floor%s",
		len(sets), len(names), reached, affordedMeasuredNote(len(names), len(sets)))
}
