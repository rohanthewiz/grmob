package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// This repository has ONE `-short` lever, it is in the place this file says it
// is, and it skips rather than shrinks.
//
// # What this is about
//
// internal/themehistory's whole-walk arm is the only test here that runs the
// real program over the real history. It costs a couple of seconds on a plain
// run and about five under `-race`, and it was given a `-short` skip so that
// anybody who does not want to pay it has a lever — with the default left ON,
// because a claim nobody checks on a green run is the state that arm was
// written to end.
//
// That made `-short` load-bearing and left it asserted by nothing. Two things
// go wrong quietly from there, and both of them look like a fast green run:
//
//	a second arm under the same skip   `-short` now means something different
//	                                   from what its one comment says, and
//	                                   nobody chose that
//	the lever moved or renamed         the expensive arm runs on every short
//	                                   run, or a cheap one stops running on
//	                                   them, and either way the message a
//	                                   reader gets is about the wrong test
//
// So the lever becomes a rule with an arm rather than staying a call with a
// comment beside it. The table below is the whole convention: which tests read
// testing.Short(), and what a short run therefore stops asserting.
//
// # What is checked
//
//	where       every testing.Short() in the repository is one of the rows
//	            below, and every row is still there
//	how         each one SKIPS. A `-short` that shrinks a loop or lowers a
//	            sample count is a different thing wearing the same name: the
//	            test still passes and it is no longer the test whose name is
//	            in the run's output
//	how many    shortLeverBudget. One lever is a lever; a second is a
//	            convention, and a convention needs the rule written down
func TestTheShortLeversAreTheOnesThisRepositoryHasDecidedOn(t *testing.T) {
	root := filepath.Join("..", "..")
	_, considered, from, err := citingFiles(root)
	if err != nil {
		t.Fatalf("enumerating the repository (%s): %v", from, err)
	}
	paths := make([]string, 0, len(considered))
	for p := range considered {
		if strings.HasSuffix(p, ".go") {
			paths = append(paths, p)
		}
	}
	// Sorted for the reason every walk in this package sorts: findings that
	// arrive in map order cannot be diffed against the last run.
	sort.Strings(paths)

	var found []shortLever

	fset := token.NewFileSet()
	for _, rel := range paths {
		src := filepath.Join(root, filepath.FromSlash(rel))
		file, parseErr := parser.ParseFile(fset, src, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			// A file go/parser cannot read is not this check's business — the
			// build says so first.
			continue
		}
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			at := 0
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) != 0 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Short" {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "testing" {
					at = fset.Position(call.Pos()).Line
					return false
				}
				return true
			})
			if at == 0 {
				continue
			}
			found = append(found, shortLever{file: rel, fn: fn.Name.Name,
				line: at, skips: callsSkip(fn.Body)})
		}
	}

	// The walk reaching anything. A walk over nothing passes silently and
	// reads as a clean result, and this one is looking for a CALL that a
	// rewrite could take away without anybody meaning to — which is one of the
	// two failure shapes this arm exists for.
	if len(found) == 0 {
		t.Fatalf("no `testing.Short()` was found in %d Go file(s) enumerated "+
			"by %s, and this repository has %d: %s.\n\n"+
			"Either the walk is not reaching them, or the lever has been "+
			"removed — in which case the arm it was guarding now runs on every "+
			"short run, and this check should go in the same commit rather "+
			"than be left passing over nothing.",
			len(paths), from, len(shortLevers), shortLeverList())
	}

	// Which levers exist, against which ones this repository has decided on.
	// Both directions: an unlisted one is a convention nobody wrote down, and
	// a listed one that has gone is a row describing nothing.
	want := map[string]string{}
	for _, l := range shortLevers {
		want[l.file+" "+l.fn] = l.stops
	}
	seen := map[string]bool{}
	for _, l := range found {
		key := l.file + " " + l.fn
		seen[key] = true
		if _, ok := want[key]; !ok {
			t.Errorf("%s:%d reads testing.Short() in %s, and this repository's "+
				"`-short` convention is one lever: %s.\n\n"+
				"A second one is not a mistake — it is the moment `-short` "+
				"stops meaning one named thing and starts meaning whatever the "+
				"set of tests reading it happens to skip. What a short run "+
				"still asserts is then a question nobody can answer from the "+
				"flag, which is the state a fast green run hides.\n\n"+
				"Add a row to shortLevers saying what this one stops "+
				"asserting, and raise shortLeverBudget with the reason beside "+
				"it, so the next person reads a decision rather than a number.",
				l.file, l.line, l.fn, shortLeverList())
		}
		if !l.skips {
			t.Errorf("%s:%d reads testing.Short() in %s and nothing in that "+
				"function skips.\n\n"+
				"A `-short` that shrinks a loop, lowers a sample count or "+
				"takes a smaller input is a different thing wearing the same "+
				"name: the test passes, its name is in the run's output, and "+
				"what it asserted is not what it asserts on a full run. This "+
				"repository's one lever SKIPS, and says in the skip message "+
				"what is not being asserted — which is the half a reader of a "+
				"green `-short` run has to be able to see.",
				l.file, l.line, l.fn)
		}
	}
	for key, stops := range want {
		if seen[key] {
			continue
		}
		t.Errorf("shortLevers says %s reads testing.Short(), and %s found no "+
			"such call there.\n\n"+
			"On a short run that arm is meant to stop asserting: %s. If the "+
			"lever has moved, this row is describing nothing; if it has gone, "+
			"the arm now runs on every short run and the reason it had a lever "+
			"is worth re-reading before this row is deleted.", key, from, stops)
	}

	if len(found) > shortLeverBudget {
		t.Errorf("%d test(s) in this repository read testing.Short() and the "+
			"lever was written as one: %s.\n\n"+
			"See shortLeverBudget — the question a second one asks is not "+
			"whether it is worth skipping, it is what `-short` is now "+
			"guaranteed to still cover.",
			len(found), strings.Join(leverList(found), ", "))
	}

	// Reported as what was found rather than as what should have been found:
	// this line is printed on a failing run too, and "each skipping, each with
	// a row" is a claim the errors above may have just contradicted.
	t.Logf("%d `-short` lever(s) against %d decided on: %s. The rows say what "+
		"a short run stops asserting; see shortLevers. Enumerated by %s over "+
		"%d Go file(s).", len(found), len(shortLevers),
		strings.Join(leverList(found), ", "), from, len(paths))
}

// How many `-short` levers this repository has decided on.
//
// One, and the second is a decision rather than a number to raise — the same
// shape of constant as gitWrapperAcceptRules and timingsRecordCopies, for the
// same kind of reason.
//
// # What the second one actually costs
//
// Not the skip: skipping a slow arm is usually right. It is that `-short` has
// no meaning of its own here — it means "whatever the tests reading it decided
// to skip" — so with one lever a reader can be told exactly what a short run
// does not assert, and with several the honest answer is a list somebody has
// to assemble by grepping.
//
// So the second lever is the moment to write the rule down: which run is the
// supported one, what `-short` is guaranteed to still cover, and where a
// reader finds that out. Until then the table below IS the rule, and it is
// short enough to read.
const shortLeverBudget = 1

// Every test in this repository that reads testing.Short(), and what a short
// run therefore stops asserting.
//
// `stops` is the part worth having. A row that only named the test would say
// where the lever is; what a reader of a green `-short` run needs is which
// claims did not get made.
var shortLevers = []struct {
	file, fn string
	stops    string
}{{
	file: "internal/themehistory/timings_test.go",
	fn:   "TestTheWholeWalkGoesRoundOneBatchProcess",
	stops: "that a real walk over the real history goes round ONE " +
		"`git cat-file --batch`, fetches exactly the objects themeSourcesAt " +
		"names across the history, and prints the table — and no wall clock " +
		"is taken for themehistoryTimingsTakenOn.wholeRun",
}}

// shortLeverList is the decided levers, for a message.
func shortLeverList() string {
	out := make([]string, 0, len(shortLevers))
	for _, l := range shortLevers {
		out = append(out, fmt.Sprintf("%s (%s)", l.fn, l.file))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// shortLever is one testing.Short() the walk found: where it is, and whether
// the function it is in skips.
type shortLever struct {
	file, fn string
	line     int
	skips    bool
}

// leverList is what the walk found, for a message.
func leverList(found []shortLever) []string {
	out := make([]string, 0, len(found))
	for _, l := range found {
		out = append(out, fmt.Sprintf("%s (%s:%d)", l.fn, l.file, l.line))
	}
	sort.Strings(out)
	return out
}

// callsSkip is whether this body calls Skip, Skipf or SkipNow on anything.
//
// The receiver is not checked. Every one of these is a method on *testing.T or
// *testing.B and nothing else in this repository declares such a name, so
// matching the selector alone is the whole question — and a check that tried
// to prove the receiver was the test's own `t` would be the dataflow question
// gitquoting_test.go explains this package does not ask.
func callsSkip(body *ast.BlockStmt) bool {
	hit := false
	ast.Inspect(body, func(n ast.Node) bool {
		if hit {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch sel.Sel.Name {
		case "Skip", "Skipf", "SkipNow":
			hit = true
			return false
		}
		return true
	})
	return hit
}
