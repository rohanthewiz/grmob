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
//
// # What "SKIPS" is asked of, which used to be the whole function
//
// The first spelling of this check asked whether ANYTHING in the enclosing
// function skipped. That is a question about the wrong scope, and it passes
// the exact shape the rule is against:
//
//	if runtime.GOOS == "js" { t.Skip(…) }
//	samples := 1000
//	if testing.Short() { samples = 10 }
//
// The function skips, so the old check was satisfied; what `-short` does in it
// is shrink a loop. So the lever is now tied to the `if` it is the condition
// OF — see shortLeverSkip — and the branch that condition guards is what has
// to skip.
//
// The receiver is checked too, and for the same reason one level down: `.Skip`
// on anything at all used to count. It is now held to the test's own
// *testing.T (or *testing.B, or *testing.F), read off the function's
// signature, so a `someOther.Skip()` in the guarded branch is not read as this
// test skipping. That is a syntactic question about one parameter name and not
// the dataflow question gitquoting_test.go declines: nothing here follows a
// value, it compares two identifiers.
//
// # And which `testing` is the testing package
//
// Both of those readings were spelled against the qualifier `testing`, which
// is what every file in this repository happens to call it. An alias —
// `import gotesting "testing"` — is a lever this census does not see, and an
// unseen lever is not a near miss: it is `-short` meaning something nobody
// wrote down, which is the failure this whole arm is about. So the qualifier
// comes off each file's own import block, which also lets the walk skip a file
// that does not import testing before looking at a single declaration. See
// importnames_test.go.
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
		// Which identifier THIS FILE binds to the testing package. A lever
		// spelled `import gotesting "testing"` and `gotesting.Short()` is a
		// second lever the census would not see, and an unseen lever is the
		// exact failure this arm exists for: `-short` quietly means something
		// nobody wrote down. Read off the import block; see
		// importnames_test.go.
		testingNames := qualifiersFor(t, rel, file, "testing")
		if len(testingNames) == 0 {
			// A file that does not import testing has no lever in it and no
			// *testing.T either. Skipped before the declarations rather than
			// inside them, because that is most of the repository.
			continue
		}
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			// The test's own *testing.T / *testing.B / *testing.F, by name.
			// A function that reads testing.Short() without one is not a test
			// at all, and `recv` is "" for it — which shortLeverSkip reports
			// as its own finding rather than treating as a missing skip.
			recv := testParamName(fn, testingNames)
			at, guarded := shortLeverCall(fset, fn.Body, testingNames)
			if at == 0 {
				continue
			}
			why := shortLeverSkip(guarded, recv)
			found = append(found, shortLever{file: rel, fn: fn.Name.Name,
				line: at, skips: why == "", why: why})
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
			t.Errorf("%s:%d reads testing.Short() in %s and %s.\n\n"+
				"A `-short` that shrinks a loop, lowers a sample count or "+
				"takes a smaller input is a different thing wearing the same "+
				"name: the test passes, its name is in the run's output, and "+
				"what it asserted is not what it asserts on a full run. This "+
				"repository's one lever SKIPS, and says in the skip message "+
				"what is not being asserted — which is the half a reader of a "+
				"green `-short` run has to be able to see.\n\n"+
				"The question is asked of the BRANCH testing.Short() guards "+
				"and not of the function around it: a test that skips for an "+
				"unrelated reason elsewhere and shrinks a loop here is exactly "+
				"the shape this rule is against, and it satisfies a check "+
				"that only asks whether the function skips somewhere.",
				l.file, l.line, l.fn, l.why)
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
	// Why not, when it does not — as the half of a sentence the message puts
	// after "reads testing.Short() in F and ". Empty when it does skip.
	//
	// Carried rather than re-derived in the message, because there are four
	// ways to fail this and "nothing skips" describes one of them: see
	// shortLeverSkip.
	why string
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

// shortLeverCall finds this body's testing.Short() and the branch it guards.
//
// # Why the branch and not the function
//
// A lever is a CONDITION. `if testing.Short() { t.Skip(…) }` is the shape this
// repository decided on, and what makes it that shape is the branch: the call
// is a question, and the answer is what the `if` does with it. Reading the
// enclosing function instead answers a looser question — "does anything here
// skip" — which a test that skips for one reason and shrinks for another
// passes while being the thing the rule refuses.
//
// So the call is found first, and then the innermost `if` whose CONDITION
// contains it. `guarded` is that if's body.
//
//	if testing.Short() { … }              guarded is the body
//	if testing.Short() && x { … }         the same: the call is in the cond
//	if !testing.Short() { … }             the same body, and shortLeverSkip
//	                                      fails it — the branch a short run
//	                                      does NOT take is not where a skip
//	                                      belongs, and a lever spelled this way
//	                                      is a full run taking a second path
//	short := testing.Short(); if short …  no enclosing cond, guarded is nil,
//	                                      and shortLeverSkip says so. A lever
//	                                      held in a variable is one this parse
//	                                      cannot follow, which is a finding
//	                                      rather than a pass
//
// Returns the call's line, which is what every message here points at.
func shortLeverCall(fset *token.FileSet, body *ast.BlockStmt, testingPkg map[string]bool) (line int, guarded *ast.BlockStmt) {
	// The chain of `if` statements this walk is currently inside, innermost
	// last. ast.Inspect calls back with nil on the way out of a node, which is
	// what pops it.
	var ifs []*ast.IfStmt
	ast.Inspect(body, func(n ast.Node) bool {
		if line != 0 {
			return false
		}
		if n == nil {
			if len(ifs) > 0 {
				ifs = ifs[:len(ifs)-1]
			}
			return false
		}
		if in, ok := n.(*ast.IfStmt); ok {
			ifs = append(ifs, in)
			return true
		}
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 0 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Short" {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || !testingPkg[pkg.Name] {
			return true
		}
		line = fset.Position(call.Pos()).Line
		// The innermost enclosing `if` whose CONDITION this call is inside.
		// Walking outwards rather than taking the last one, because the call
		// may be in a nested if's BODY rather than its condition — in which
		// case the lever belongs to whichever ancestor's condition holds it,
		// and to none of them if no ancestor's does.
		for i := len(ifs) - 1; i >= 0; i-- {
			if containsPos(ifs[i].Cond, call.Pos()) {
				guarded = ifs[i].Body
				break
			}
		}
		return false
	})
	return line, guarded
}

// containsPos is whether an expression's source range covers this position.
//
// Position arithmetic rather than a second walk: the condition and the call
// come out of one parse and one FileSet, so a call inside a condition is
// exactly a call whose Pos lies between the condition's ends. Nothing here
// needs to know what the condition is made of, which is the point — `a &&
// testing.Short()`, `!testing.Short()` and a call buried in a parenthesised
// expression all answer the same way.
func containsPos(e ast.Expr, pos token.Pos) bool {
	return e != nil && e.Pos() <= pos && pos < e.End()
}

// shortLeverSkip is why this lever is not a skip, or "" when it is one.
//
// Four ways to fail, and they are different findings — which is why this
// returns a reason rather than a bool. Each is a half-sentence the caller puts
// after "reads testing.Short() in F and ".
func shortLeverSkip(guarded *ast.BlockStmt, recv string) string {
	if recv == "" {
		return "that function takes no *testing.T, *testing.B or *testing.F, " +
			"so there is nothing there that could skip. A `-short` read " +
			"outside a test is a lever on something else"
	}
	if guarded == nil {
		return "the call is not the condition of an `if`. A lever assigned to " +
			"a variable, passed to a helper or read inside a larger " +
			"expression is one this parse cannot follow to the branch it " +
			"decides, and what a short run then stops asserting is not " +
			"readable off the source"
	}
	if !skipsVia(guarded, recv) {
		return "the branch it guards does not call " + recv + ".Skip, " +
			recv + ".Skipf or " + recv + ".SkipNow"
	}
	return ""
}

// testParamName is the name of the test's own *testing.T, *testing.B or
// *testing.F parameter.
//
// # Why the receiver is worth reading at all
//
// The skip check used to match any `.Skip` selector on anything, on the
// reasoning that these are methods on the testing types and nothing else here
// declares such a name. That is true today and it is an argument about the
// repository rather than about the code being read, and it costs nothing to
// stop making: the parameter is right there in the signature.
//
// What this deliberately does NOT do is prove the receiver at the call site IS
// that parameter in any deeper sense — a `t` shadowed by a local of another
// type would still match. That is the dataflow question gitquoting_test.go
// explains this package does not ask, and the answer here is the same: this
// compares two identifiers, which is a real narrowing over matching every
// selector in the language, and the remaining looseness is in the direction of
// ACCEPTING a lever, which is the safe direction for a census whose failure is
// a prompt.
//
// "" when there is no such parameter, which shortLeverSkip reports as its own
// finding: a testing.Short() outside a test is not a lever this convention is
// about.
func testParamName(fn *ast.FuncDecl, testingPkg map[string]bool) string {
	if fn.Type.Params == nil {
		return ""
	}
	for _, f := range fn.Type.Params.List {
		star, ok := f.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sel, ok := star.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || !testingPkg[pkg.Name] {
			continue
		}
		switch sel.Sel.Name {
		case "T", "B", "F":
		default:
			continue
		}
		for _, id := range f.Names {
			if id.Name != "_" {
				return id.Name
			}
		}
	}
	return ""
}

// skipsVia is whether this block calls Skip, Skipf or SkipNow on `recv`.
func skipsVia(body *ast.BlockStmt, recv string) bool {
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
		default:
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == recv {
			hit = true
			return false
		}
		return true
	})
	return hit
}
