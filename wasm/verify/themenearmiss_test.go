package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// themeNearMissEdits, derived from the struct it is asked about.
//
// # Why the number is measured and not chosen
//
// themeNearMiss reports what a path that names no leaf of core.Theme was
// probably trying to say. A threshold decides how far "probably" reaches, and a
// threshold somebody picked because it looked sensible is the shape of thing
// this file keeps replacing with a measurement. The measurements are of
// core.Theme's own leaf names, so the number moves when the struct does — and
// this test is where it would say so.
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
func TestThemeNearMissThresholdIsDerivedFromCoreTheme(t *testing.T) {
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
	t.Logf("core.Theme: %d parents, closest sibling pair %d apart (%s); "+
		"within %d edits at most %d sibling, within %d at most %d",
		len(byParent), closest, pair, themeNearMissEdits, at,
		themeNearMissEdits+1, beyond)
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
	leaves := map[string]bool{}
	paths := themeLeafPaths(reflect.ValueOf(*core.DefaultTheme), "")
	for _, path := range paths {
		leaves[path] = true
	}
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
			if leaves[typed] {
				continue // a real leaf: this walk never asks about one
			}
			tried++
			got := themeNearMiss(leaves, typed)
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
