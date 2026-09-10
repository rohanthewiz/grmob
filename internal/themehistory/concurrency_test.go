package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// What in this package runs concurrently, and what none of it is allowed to
// touch.
//
// # What this is about
//
// Two claims elsewhere in this package rest on nothing here being parallel,
// and both of them are load-bearing:
//
//	blobsMu's comment      "the mutex is not for this program, which is
//	                       single-threaded. It is for the tests"
//	the whole-walk arm     it swaps os.Stdout for a file for the duration of
//	                       run(). os.Stdout is a process-wide variable and
//	                       fmt.Printf reads it at CALL TIME, so a redirect is
//	                       safe exactly as long as nothing else is printing
//
// Then themeSourcesAcross grew a pool of eight goroutines. The argument that
// this is fine — it runs before the walk, touches no package state, and joins
// first — was written down at enumWorkers at some length, and a paragraph is
// not a check. What it does not survive is an ordinary later edit: a fetch
// moved inside the pool to save a second, a print added to see what a worker
// is doing. `-race` catches the first only if two goroutines happen to overlap
// on the run that matters, and nothing at all catches the second.
//
// So the exception becomes a rule with an arm. This file is the same shape of
// census wasm/verify writes for the same kind of reason, one directory over.
//
// # What is checked
//
//	which       every `go` statement in this package is one of the rows below,
//	            and every row is still there — both directions
//	what        no goroutine reaches anything in goroutineMustNotTouch, by
//	            naming it or by calling something in this package that does.
//	            Transitively; see reachesFrom
//	how many    goroutineBudget. Three is a number somebody chose; a fourth is
//	            where "this package is single-threaded, apart from" stops being
//	            a sentence anybody can finish
//
// A row is per FUNCTION and the budget is per STATEMENT, which is not a
// mismatch: what a reader wants named is the place that decided to be
// concurrent, and what the budget is about is how many goroutines a run
// starts. A second `go` inside an already-listed function is therefore not an
// unlisted site — it is over budget, which is the finding that fits it.
//
// # Why the call graph and not just the body
//
// A goroutine body is usually three lines and a call. `go func(){ blobs.read(…)
// }()` is the shape nobody writes; `go func(){ leavesAt(sha) }()` is the same
// bug spelled the way somebody would, and only a walk that follows leavesAt to
// blob to `blobs` can tell. The walk is over THIS PACKAGE's declarations only,
// which is the whole call graph that matters: everything below it is the
// standard library and os/exec.
func TestTheGoroutinesInThisPackageAreTheOnesDecidedOn(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading this package's directory: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			names = append(names, e.Name())
		}
	}
	// Sorted for the reason every census in this repository sorts: findings
	// that arrive in directory order cannot be diffed against the last run.
	sort.Strings(names)

	fset := token.NewFileSet()
	// name -> what its body calls, and whether it names something guarded.
	// Methods are keyed by their own name with no receiver: two `read`s in one
	// package would be conflated, which over-approximates and is the safe
	// direction for a check whose failure is a question.
	calls := map[string][]string{}
	touches := map[string]string{}
	var found []goroutineSite

	for _, name := range names {
		file, parseErr := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
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
			calls[fn.Name.Name] = append(calls[fn.Name.Name], calleeNames(fn.Body)...)
			if what := touchesGuarded(fn.Body); what != "" {
				touches[fn.Name.Name] = what
			}
			// The `go` statements themselves. Reported against the function
			// they are IN, which is what a row names: a goroutine is a cost
			// its enclosing function decided to pay.
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				stmt, ok := n.(*ast.GoStmt)
				if !ok {
					return true
				}
				found = append(found, goroutineSite{
					file: name,
					fn:   fn.Name.Name,
					line: fset.Position(stmt.Pos()).Line,
					body: stmt,
				})
				return true
			})
		}
	}

	// The walk reaching anything. A census over nothing passes silently and
	// reads as a clean result, and this one is looking for a STATEMENT that a
	// rewrite could take away without anybody meaning to — which would be the
	// pool going, and the figures in themehistoryTimingsTakenOn with it.
	if len(found) == 0 {
		t.Fatalf("no `go` statement was found in %d Go file(s) in this "+
			"package, and it has %d: %s.\n\n"+
			"Either the walk is not reaching them or they have gone. The "+
			"enumeration's pool going would put 0.7s back into every green "+
			"run of this package (see enumWorkers), and this arm should be "+
			"retired in the same commit rather than left passing over nothing.",
			len(names), len(decidedGoroutines), decidedGoroutineList())
	}

	// Which goroutines exist, against which ones this package has decided on.
	want := map[string]decidedGoroutine{}
	for _, g := range decidedGoroutines {
		want[g.file+" "+g.fn] = g
	}
	seen := map[string]bool{}
	for _, g := range found {
		key := g.file + " " + g.fn
		seen[key] = true
		if _, ok := want[key]; !ok {
			t.Errorf("%s:%d starts a goroutine in %s, and this package's "+
				"concurrency is %d place(s): %s.\n\n"+
				"Two things here are correct only because almost nothing in "+
				"this package is parallel: blobsMu's comment says the program "+
				"is single-threaded, and the whole-walk arm swaps os.Stdout "+
				"for the duration of run(). Neither survives an unannounced "+
				"third party.\n\n"+
				"Add a row to decidedGoroutines saying what this one is and "+
				"what joins it, and raise goroutineBudget with the reason "+
				"beside it, so the next person reads a decision rather than a "+
				"number.", g.file, g.line, g.fn, len(decidedGoroutines),
				decidedGoroutineList())
		}

		// And the thing the rows cannot say for themselves: what it reaches.
		// Directly first, so the message can name the line.
		if what := touchesGuarded(g.body); what != "" {
			t.Errorf("the goroutine at %s:%d names %s.\n\n%s\n\n"+
				"This is the assumption the two claims above are made of, and "+
				"it is not a style rule: the redirect and the mutex comment "+
				"are both correct today because the goroutines here touch "+
				"none of this.", g.file, g.line, what,
				goroutineMustNotTouch[what])
			continue
		}
		// Then through the package's own call graph.
		if via, what := reachesFrom(g.body, calls, touches); what != "" {
			t.Errorf("the goroutine at %s:%d calls %s, which reaches %s.\n\n"+
				"%s\n\n"+
				"The call chain is followed through this package's own "+
				"declarations, which is where it can go: a `go func(){ "+
				"leavesAt(sha) }()` is one hop from blob and two from the "+
				"shared reader, and it is the spelling somebody would "+
				"actually write.", g.file, g.line, via, what,
				goroutineMustNotTouch[what])
		}
	}
	for key, g := range want {
		if seen[key] {
			continue
		}
		t.Errorf("decidedGoroutines says %s starts a goroutine and none was "+
			"found there.\n\n"+
			"What it was: %s. Joined by: %s. If it has moved, this row "+
			"describes nothing; if it has gone, the reason it was worth a "+
			"goroutine is worth re-reading before this row is deleted.",
			key, g.what, g.joinedBy)
	}

	if len(found) > goroutineBudget {
		t.Errorf("this package starts a goroutine in %d place(s) and the "+
			"budget is %d: %s.\n\n"+
			"See goroutineBudget — the question a fourth one asks is not "+
			"whether it is worth starting, it is whether anybody can still "+
			"finish the sentence \"this package is single-threaded, apart "+
			"from\".", len(found), goroutineBudget, goroutineList(found))
	}

	// Reported as what was found rather than as what should have been found.
	// This line is printed on a failing run too, and "none of them reaches"
	// is a claim the errors above may have just contradicted — so it is only
	// made when nothing above made the opposite one.
	reach := fmt.Sprintf("None of them reaches %s, directly or through this "+
		"package's own calls", mustNotTouchList())
	if t.Failed() {
		reach = fmt.Sprintf("Held against %s, directly and through this "+
			"package's own calls — see the finding(s) above", mustNotTouchList())
	}
	t.Logf("%d goroutine(s) in %d Go file(s) in this package: %s. %s; see "+
		"decidedGoroutines for what joins each.", len(found), len(names),
		goroutineList(found), reach)
}

// How many places in this package may start a goroutine.
//
// Three, and the fourth is a decision rather than a number to raise — the same
// shape of constant as wasm/verify's shortLeverBudget and gitWrapperAcceptRules,
// for the same kind of reason.
//
// # What the fourth one actually costs
//
// Not the goroutine. It is that the two claims in the header stop being
// checkable by reading one paragraph: with three, a reader asking "is the
// os.Stdout redirect safe" reads three rows and knows; past that the honest
// answer is a call graph somebody has to walk. The arm above walks it either
// way — what the budget buys is that a person can too.
const goroutineBudget = 3

// What no goroutine in this package may reach, and why each one.
//
// The reason is the message: an arm that says "do not touch this" and not why
// is one the next person routes around.
//
// batchesStarted is deliberately NOT here. It is an atomic.Int64, which is the
// one piece of package state that means the same thing from two goroutines,
// and listing it would be this table asserting a style rather than a premise.
var goroutineMustNotTouch = map[string]string{
	"blobs": "`blobs` is the process-wide `git cat-file --batch`. Two " +
		"goroutines driving one batch reader interleave two requests down " +
		"one pipe and hand each caller the other's file — which is not a " +
		"crash, it is a table built out of the wrong bytes. blobsMu exists " +
		"for exactly this and its comment says the program never needed it.",
	"os.Stdout": "`os.Stdout` is a variable, and " +
		"TestTheWholeWalkGoesRoundOneBatchProcess swaps it for a file for the " +
		"duration of run(). Anything printing from another goroutine during " +
		"that window writes into the captured table or misses the swap " +
		"entirely, depending on when it read the variable.",
	"fmt.Print": "`fmt.Print`, `fmt.Printf` and `fmt.Println` read os.Stdout " +
		"at CALL TIME, so they are the redirect above wearing a different " +
		"name. A print added to a worker to see what it is doing is the " +
		"whole failure.",
	"t.Fatal": "`Fatal`, `Fatalf`, `FailNow` and `SkipNow` call " +
		"runtime.Goexit, which ends the CALLING goroutine. From a worker that " +
		"kills the worker and leaves the test running, so the run either " +
		"deadlocks on a WaitGroup that never finishes or passes having " +
		"silently done less. themeSourcesAcross collects its errors into a " +
		"slice and reports them after the join for this reason.",
}

// decidedGoroutine is one place this package has decided to start one.
type decidedGoroutine struct {
	file, fn string
	// What it is, as the reason it exists.
	what string
	// And what makes it finish. A goroutine nobody joins is the other half of
	// the question this census asks: not what it touches, but for how long.
	joinedBy string
}

// Every `go` statement in this package.
var decidedGoroutines = []decidedGoroutine{{
	file: "main.go",
	fn:   "retire",
	what: "the DRAIN. `git cat-file --batch` exits when its stdin reaches " +
		"EOF, and a git blocked writing into a full pipe never sees one — so " +
		"the pipe is emptied and the process waited for, off the calling " +
		"goroutine so that a wedged git can be given a deadline",
	joinedBy: "`<-done` on both arms of the select, including after the Kill. " +
		"retire does not return until this goroutine has",
}, {
	file: "timings_test.go",
	fn:   "themeSourcesAcross",
	what: "the enumeration pool. One `git ls-tree` per commit, 88 of them, " +
		"which is 0.90s serially and 0.22s over eight workers — see " +
		"enumWorkers for the bound and for what it deliberately leaves serial",
	joinedBy: "wg.Wait(), before the results are read and before either arm " +
		"measures anything. Results go into a slice indexed by commit, so no " +
		"two workers touch one element and the order is the same at any " +
		"worker count",
}, {
	file: "main_test.go",
	fn:   "TestRetiringAWedgedProcessDoesNotWaitForever",
	what: "the retire being timed. It is the call under test, run off the " +
		"test's own goroutine so that a retire which never returns fails the " +
		"test instead of hanging the binary",
	joinedBy: "a select on `done` against a timeout, which is the assertion " +
		"itself — the test fails rather than blocking if this goroutine does " +
		"not finish",
}}

// goroutineSite is one `go` statement the walk found.
type goroutineSite struct {
	file, fn string
	line     int
	body     *ast.GoStmt
}

// decidedGoroutineList is the decided sites, for a message.
func decidedGoroutineList() string {
	out := make([]string, 0, len(decidedGoroutines))
	for _, g := range decidedGoroutines {
		out = append(out, fmt.Sprintf("%s (%s)", g.fn, g.file))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// goroutineList is what the walk found, for a message.
func goroutineList(found []goroutineSite) string {
	out := make([]string, 0, len(found))
	for _, g := range found {
		out = append(out, fmt.Sprintf("%s (%s:%d)", g.fn, g.file, g.line))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// mustNotTouchList is the guarded names, for a message.
func mustNotTouchList() string {
	out := make([]string, 0, len(goroutineMustNotTouch))
	for name := range goroutineMustNotTouch {
		out = append(out, name)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// touchesGuarded is the first thing in goroutineMustNotTouch this node
// reaches directly, or "" for one that reaches none.
//
// # How each is recognised, and what that is loose about
//
//	blobs          a bare identifier. There is one such name in this package
//	               and nothing shadows it
//	os.Stdout      the selector, whatever is done with it. Reading it to save
//	               it is as much a use as writing it
//	fmt.Print*     the three that write to os.Stdout. fmt.Fprintf(os.Stderr, …)
//	               is not one of them and is not a hazard: stderr is never
//	               swapped here
//	t.Fatal…       the selector NAME, with no receiver check — the same
//	               approximation wasm/verify's git-wrapper census makes and
//	               for the same reason. Nothing else in this package declares
//	               a Fatalf, and a check that tried to prove the receiver was
//	               a *testing.T would be a dataflow question worth more than
//	               the answer
//
// Every one of these is loose in the direction of REPORTING, which is the safe
// direction for a census whose failure is a question rather than a verdict.
func touchesGuarded(n ast.Node) string {
	hit := ""
	ast.Inspect(n, func(node ast.Node) bool {
		if hit != "" {
			return false
		}
		switch e := node.(type) {
		case *ast.Ident:
			if e.Name == "blobs" {
				hit = "blobs"
				return false
			}
		case *ast.SelectorExpr:
			pkg, ok := e.X.(*ast.Ident)
			if ok && pkg.Name == "os" && e.Sel.Name == "Stdout" {
				hit = "os.Stdout"
				return false
			}
			if ok && pkg.Name == "fmt" {
				switch e.Sel.Name {
				case "Print", "Printf", "Println":
					hit = "fmt.Print"
					return false
				}
			}
			switch e.Sel.Name {
			case "Fatal", "Fatalf", "FailNow", "SkipNow":
				hit = "t.Fatal"
				return false
			}
		}
		return true
	})
	return hit
}

// calleeNames is every function this body calls by a name this package could
// have declared.
//
// Bare identifiers and method names both, because a goroutine reaching the
// shared reader is as likely to do it through `b.read(…)` as through
// `blob(…)`. A qualified `pkg.Fn(…)` is somebody else's package and cannot
// name anything guarded — except for the four selectors touchesGuarded looks
// for itself, which is why that runs first.
func calleeNames(body *ast.BlockStmt) []string {
	var out []string
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			out = append(out, fn.Name)
		case *ast.SelectorExpr:
			// `x.Method(…)`, but not `pkg.Fn(…)` — told apart by nothing, so
			// both are collected and only names this package declares are
			// ever looked up.
			out = append(out, fn.Sel.Name)
		}
		return true
	})
	return out
}

// reachesFrom follows this node's calls through the package's own declarations
// until one of them touches something guarded.
//
// Breadth-first from the goroutine's own callees, with a seen-set, so a
// recursive or mutually recursive pair terminates. Returns the name that was
// called and what it reached, so the message can say both — "calls leavesAt,
// which reaches blobs" is a finding somebody can act on and "reaches blobs" is
// not.
//
// Only names this package declares are followed. Everything else is the
// standard library, which cannot name a package-level variable of ours; the
// four selectors that ARE hazards regardless of package are checked by
// touchesGuarded at every step rather than followed.
func reachesFrom(from ast.Node, calls map[string][]string,
	touches map[string]string) (via, what string) {

	// The goroutine's own callees, each carrying itself as the name to blame.
	type step struct{ name, blame string }
	var work []step
	seen := map[string]bool{}
	ast.Inspect(from, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var name string
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			name = fn.Name
		case *ast.SelectorExpr:
			name = fn.Sel.Name
		default:
			return true
		}
		if _, ours := calls[name]; ours && !seen[name] {
			seen[name] = true
			work = append(work, step{name: name, blame: name})
		}
		return true
	})
	for len(work) > 0 {
		s := work[0]
		work = work[1:]
		if guarded, bad := touches[s.name]; bad {
			return s.blame, guarded
		}
		for _, next := range calls[s.name] {
			if _, ours := calls[next]; !ours || seen[next] {
				continue
			}
			seen[next] = true
			work = append(work, step{name: next, blame: s.blame})
		}
	}
	return "", ""
}
