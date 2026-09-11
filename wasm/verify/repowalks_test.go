package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// How many times a run of this package walks the whole repository, and what
// each walk is for.
//
// # What this is about
//
// citingFiles asks git for every tracked file, stats it, reads it and hands
// back the lot. Six tests here call it, and four of those then parse every Go
// file in the tree with go/parser. None of them shares anything with the
// others: each pays its own `git ls-files`, its own 381 file reads and its own
// parse.
//
// That is about a second of a 2.7-second package (see verifyTimingsTakenOn),
// and it is the right call at this size. The walks are INDEPENDENT by design:
// each asks a different question, each fails on its own, and a shared cache
// between them would be a fixture with lifetime rules — built once, invalidated
// never, and read by tests that no longer say what they read. Four cheap
// duplicates beat one clever thing nobody can reason about.
//
// What was missing is that this was a standing assumption. The number was
// written into verifyTimingsTakenOn's prose by hand — as THREE, which was
// already wrong when it was typed, because TestEveryGitListingAsksForNul-
// SeparatedPaths has parsed the whole tree since before the other three
// existed. A count kept by hand in a comment is the shape this package writes
// arms instead of, and it was keeping the one number that decides when the
// trade changes.
//
// # What is checked
//
//	which       every function in this package that reaches an enumeration is
//	            one of the rows below, and every row is still there
//	how deep    the row's `depth` is what the source actually does. A walk
//	            listed as reading files that has grown a go/parser pass is a
//	            walk that costs three times what its row says
//	how many    repositoryWalkBudget, and repositoryParseBudget under it. The
//	            second is the one that would move first: a fifth parse is
//	            where one shared parse stops being a fixture nobody can reason
//	            about and starts being the cheaper of two bad options
//
// # Why the unit is a CALL and not a declaration
//
// One of these walks is not in a test at all: checkCitationsResolve is a
// helper, and it walks once per test that calls it. Counting declarations
// would price a helper called from three tests as one walk, which is the
// number being wrong in the direction that hides the cost. So a walker whose
// name does not begin with `Test` is counted by its call sites in this
// directory, and a row that says which test drives it.
//
// A SITE is still not a call, and that is the same understatement one
// construct along: a walk inside a `for`, or inside a subtest closure that a
// `range` drives, is one site and as many walks as the loop is long. A parse
// cannot price that — the loop's length is a run-time fact — so it is
// REPORTED instead of counted, which is the honest version of the same
// finding. There is no such site in this package; see callsTo for how one
// would be recognised, and why it would otherwise read exactly like a row
// that is right.
//
// # Why this arm is not itself a repository walk
//
// It reads THIS DIRECTORY, which is where every caller of citingFiles is and
// has to be — citingFiles is unexported, so nothing outside package main here
// can reach it. Files are filtered by a byte scan for the name before being
// parsed, which cannot produce a false negative: a call to citingFiles(…)
// contains those bytes, whatever else the file says. The parse is still what
// DECIDES, so the prose in this file — which names citingFiles a dozen times —
// is read and discarded rather than counted, which is the failure a grep would
// have.
func TestTheRepositoryWideWalksInThisPackageAreTheOnesDecidedOn(t *testing.T) {
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
	// Sorted for the reason every walk in this package sorts: findings that
	// arrive in directory order cannot be diffed against the last run.
	sort.Strings(names)

	var found []repositoryWalk
	fset := token.NewFileSet()
	// Read once, parsed at most once, and only when something in the bytes
	// says it might matter. Two passes want overlapping subsets of these
	// files — the enumeration callers, and then the callers of whatever
	// helpers those turn out to be — and re-reading the directory for the
	// second would be this arm doing the thing it counts.
	sources := map[string][]byte{}
	trees := map[string]*ast.File{}
	parse := func(name string) *ast.File {
		if file, done := trees[name]; done {
			return file
		}
		// A file go/parser cannot read is not this check's business — the
		// build says so first. Cached as nil so it is not retried.
		file, _ := parser.ParseFile(fset, name, sources[name],
			parser.SkipObjectResolution)
		trees[name] = file
		return file
	}
	for _, name := range names {
		raw, readErr := os.ReadFile(name)
		if readErr != nil {
			t.Fatalf("reading %s: %v", name, readErr)
		}
		sources[name] = raw
	}
	for _, name := range names {
		// The filter. Every enumeration entry point is named here, so a file
		// mentioning none of them cannot call one.
		if !mentionsAnEnumeration(sources[name]) {
			continue
		}
		file := parse(name)
		if file == nil {
			continue
		}
		// Which identifier THIS FILE binds to go/parser. The deepest of the
		// three depths is "runs go/parser over every Go file", and reading
		// that off the conventional qualifier would miss `import goparser
		// "go/parser"` — a walk that had grown its parse and a row that still
		// said it only reads bytes, which is the cost being understated in
		// exactly the direction this arm exists to stop. See
		// importnames_test.go.
		parserNames := qualifiersFor(t, name, file, "go/parser")
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			// The enumerations themselves, which reach each other and are not
			// walks anybody chose to pay for.
			if enumerationEntryPoints[fn.Name.Name] {
				continue
			}
			depth := walkDepth(fn.Body, parserNames)
			if depth == "" {
				continue
			}
			found = append(found, repositoryWalk{
				fn:    fn.Name.Name,
				file:  name,
				line:  fset.Position(fn.Pos()).Line,
				depth: depth,
				// A test runs once. A helper runs as often as it is called,
				// and the second pass below is what says how often.
				runs: 1,
			})
		}
	}

	// The second pass: how many times each non-test walker is actually called.
	// A helper that walks the repository and is called from two tests is two
	// walks, and the budget is about walks.
	for i, w := range found {
		if strings.HasPrefix(w.fn, "Test") {
			continue
		}
		calls, in := 0, ""
		var looped []string
		for _, name := range names {
			if !bytes.Contains(sources[name], []byte(w.fn)) {
				continue
			}
			file := parse(name)
			if file == nil {
				continue
			}
			for _, d := range file.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Body == nil || fn.Name.Name == w.fn {
					continue
				}
				n, inLoops := callsTo(fset, fn.Body, w.fn)
				calls += n
				for _, line := range inLoops {
					looped = append(looped, fmt.Sprintf("%s:%d, in %s",
						name, line, fn.Name.Name))
				}
				if n > 0 && in == "" {
					in = fn.Name.Name
				}
			}
		}
		found[i].runs = calls
		found[i].drivenBy = in
		found[i].looped = looped
	}

	// The walk reaching anything. A walk over nothing passes silently and
	// reads as a clean result, and this one is looking for CALLS that a
	// rewrite could move out of this package without anybody meaning to.
	if len(found) == 0 {
		t.Fatalf("no call to any of %s was found in %d Go file(s) in this "+
			"directory, and this package has %d such walk(s): %s.\n\n"+
			"Either the scan is not reaching them or they have moved. Either "+
			"way the number verifyTimingsTakenOn's prose quotes is now kept "+
			"by nothing, which is the state this arm was written to end.",
			enumerationList(), len(names), len(repositoryWalks),
			repositoryWalkList())
	}

	// Which walks exist, against which ones this package has decided on. Both
	// directions: an unlisted one is a cost nobody chose, and a listed one
	// that has gone is a row describing nothing.
	want := map[string]repositoryWalkRow{}
	for _, w := range repositoryWalks {
		want[w.file+" "+w.fn] = w
	}
	seen := map[string]bool{}
	for _, w := range found {
		key := w.file + " " + w.fn
		seen[key] = true
		row, ok := want[key]
		if !ok {
			t.Errorf("%s:%d walks the whole repository in %s (%s), and it is "+
				"not one of the %d this package has decided on: %s.\n\n"+
				"Each of these is its own `git ls-files`, its own read of every "+
				"tracked file, and — at the deepest — its own go/parser pass "+
				"over every Go file in the tree. They are independent on "+
				"purpose, and the price of that is paid on every green run.\n\n"+
				"Add a row to repositoryWalks saying what this one asks the "+
				"repository, and raise the budget with the reason beside it, "+
				"so the next person reads a decision rather than a number.",
				w.file, w.line, w.fn, w.depth, len(repositoryWalks),
				repositoryWalkList())
			continue
		}
		// A call SITE is not a call. Every row here is priced by counting
		// sites, which is the same number only while each site is reached
		// once per run — see callsTo, and the limit this closes one construct
		// along from the helper case it was written for.
		if len(w.looped) > 0 {
			t.Errorf("%s is called from inside a loop: %s.\n\n"+
				"The count beside it is a count of call SITES, and a site "+
				"inside a `for` or a `range` is one site and as many walks as "+
				"the loop has iterations — a `git ls-files`, a read of every "+
				"tracked file and possibly a go/parser pass, each time round. "+
				"So `runs: %d` in its row understates the cost by whatever "+
				"that loop's length turns out to be, and the budget below is "+
				"counting the wrong thing.\n\n"+
				"This is the same shape as the helper case this arm does "+
				"close — a unit that runs more often than it is written — one "+
				"construct further along, and a parse cannot price it: the "+
				"loop's length is a run-time fact. Hoist the walk out of the "+
				"loop and pass its result in, which is what makes the cost a "+
				"number again, or teach this arm how to read the bound.",
				w.fn, strings.Join(w.looped, "; "), row.runs)
		}
		// Only for helpers: a test runs once by definition, and its row says
		// nothing about how often.
		if !strings.HasPrefix(w.fn, "Test") &&
			(row.runs != w.runs || row.drivenBy != w.drivenBy) {
			t.Errorf("%s:%d is listed as running %d time(s) per run, driven by "+
				"%q, and the source has %d call site(s), the first in %q.\n\n"+
				"A helper that walks the repository costs its walk once per "+
				"caller. A second caller is a second `git ls-files`, a second "+
				"read of every tracked file, and a row that still says one — "+
				"which is the cost being understated in the direction nobody "+
				"notices. Zero call sites is the other end: a walk nothing "+
				"makes, and a row describing a dead function.",
				w.file, w.line, row.runs, row.drivenBy, w.runs, w.drivenBy)
		}
		if row.depth != w.depth {
			t.Errorf("%s:%d is listed as a walk that %s and it %s.\n\n"+
				"The depths are not decoration: `%s` is one git process and "+
				"a stat per file, `%s` reads every one of them, and `%s` runs "+
				"go/parser over every Go file on top of that — roughly 0.09s, "+
				"0.16s and 0.18s respectively where verifyTimingsTakenOn was "+
				"taken. A walk that has grown a parse has roughly doubled, and "+
				"the count that decides when a shared parse is worth building "+
				"is repositoryParseBudget, not the total.",
				w.file, w.line, row.depth, w.depth,
				walkEnumerates, walkReads, walkParses)
		}
	}
	for key, row := range want {
		if seen[key] {
			continue
		}
		t.Errorf("repositoryWalks says %s walks the repository (%s) and no "+
			"such call was found in this directory.\n\n"+
			"What it asked: %s. If the walk has moved, this row describes "+
			"nothing; if it has gone, the cost recorded in "+
			"verifyTimingsTakenOn is now high by that walk's share and the "+
			"question it was asking is unasked.", key, row.depth, row.asks)
	}

	// Counted in WALKS and not in functions: a helper called twice is two
	// walks, and the budget is about what a run pays.
	walks, parses := 0, 0
	for _, w := range found {
		walks += w.runs
		if w.depth == walkParses {
			parses += w.runs
		}
	}

	if walks > repositoryWalkBudget {
		t.Errorf("this package walks the whole repository %d time(s) per run "+
			"and the budget is %d: %s.\n\n"+
			"See repositoryWalkBudget. The question a new one asks is not "+
			"whether it is worth having — each of these is — it is whether "+
			"they are still cheap enough to keep independent.",
			walks, repositoryWalkBudget, walkList(found))
	}
	if parses > repositoryParseBudget {
		t.Errorf("%d of this package's %d repository walks parse every Go "+
			"file in the tree, and the budget is %d: %s.\n\n"+
			"This is the number that decides, not the total. Each parse is "+
			"about 0.18s where verifyTimingsTakenOn was taken and none of it "+
			"is shared: the same 381 files go through go/parser once per arm, "+
			"and every one of them throws the syntax trees away.\n\n"+
			"A shared parse is a fixture with a lifetime — built once, "+
			"invalidated never, read by tests that no longer say what they "+
			"read — which is why it was not built at four. Either build it "+
			"and say what each arm now depends on, or raise "+
			"repositoryParseBudget with the reading that says the walks are "+
			"still cheap enough to keep separate.",
			parses, walks, repositoryParseBudget, walkList(found))
	}

	// Reported as what was found rather than as what should have been found:
	// this line is printed on a failing run too.
	t.Logf("%d repository-wide walk(s) per run in this package, %d of them "+
		"parsing every Go file, from %d function(s): %s. Found by scanning %d "+
		"Go file(s) in this directory and parsing the %d that named "+
		"something. Their cost is part of verifyTimingsTakenOn.wholeFile.",
		walks, parses, len(found), walkList(found), len(names), len(trees))
}

// The three depths a repository walk comes in, cheapest first.
//
// Named rather than spelled inline because the rows below and the message
// above both quote them, and a row whose depth is a string nothing else knows
// about is a row that can never match.
const (
	walkEnumerates = "asks git for the file list"
	walkReads      = "reads every tracked file"
	walkParses     = "parses every Go file"
)

// How many repository-wide walks this package has decided to pay for.
//
// Seven, and the eighth is a decision rather than a number to raise — the same
// shape of constant as gitWrapperAcceptRules, shortLeverBudget and
// timingsRecordCopies, for the same kind of reason.
//
// # What the number is actually about
//
// Not the wall clock on its own. Seven independent walks is seven arms that
// each fail for their own reason and name their own finding, which is worth
// about a second here; the same seven sharing one cached enumeration would be
// arms whose failures are entangled with a fixture's lifetime. The budget is
// where
// that trade is re-examined, and it is repositoryParseBudget below that will
// reach it first.
const repositoryWalkBudget = 7

// How many of them may parse every Go file in the tree.
//
// Four, and this is the half that costs. The other two walks read bytes and
// stop; these four hand all 381 Go files to go/parser, build the syntax trees,
// ask one question each and drop them.
//
// A fifth is where a shared parse becomes the cheaper of two bad options —
// which is a real trade and not an obvious one, so it is written down here
// rather than decided in advance.
const repositoryParseBudget = 4

// The enumeration entry points, and the functions that ARE them.
//
// citingFiles reaches repositoryFiles and walkedFiles, so the three of them
// would count each other as walks; they are the thing being counted, not
// callers of it.
var enumerationEntryPoints = map[string]bool{
	"citingFiles":     true,
	"repositoryFiles": true,
	"walkedFiles":     true,
}

// Every function in this package that walks the whole repository, and what it
// asks of it.
//
// `asks` is the part worth having. A row that only named the test would say
// where the cost is; what a reader deciding whether to share a parse needs is
// what each walk would then be sharing.
var repositoryWalks = []repositoryWalkRow{{
	fn:       "checkCitationsResolve",
	file:     "checknumbering_test.go",
	depth:    walkReads,
	runs:     1,
	drivenBy: "TestTheBrowserChecksAreOneNumberedSequence",
	asks: "every `check N` citation in the repository, in one numbered " +
		"sequence — which is why citingFiles reads every file rather than " +
		"every Go file. A HELPER and not a test: it walks once per caller, " +
		"and it also asks repositoryFiles separately first, on purpose, so " +
		"that a git which succeeds and lists nothing is told apart from a " +
		"machine with no git",
}, {
	fn:    "TestTheCitationSkipsGitAlreadyMakes",
	file:  "checknumbering_test.go",
	depth: walkEnumerates,
	asks: "which of citationSkipDirs git's own exclude rules already make, " +
		"which is a question about the FILE LIST and not about any file's " +
		"contents",
}, {
	fn:    "TestEveryGitListingAsksForNulSeparatedPaths",
	file:  "gitquoting_test.go",
	depth: walkParses,
	asks: "every git invocation in Go source that lists paths, and whether " +
		"it asks for them NUL-separated",
}, {
	fn:    "TestEveryGitListingInAScriptAsksForNulSeparatedPaths",
	file:  "gitscript_test.go",
	depth: walkReads,
	asks: "the same rule in shell scripts, which are lexed rather than " +
		"parsed — so this one reads bytes and never reaches go/parser",
}, {
	fn:    "TestTheShortLeversAreTheOnesThisRepositoryHasDecidedOn",
	file:  "shortlever_test.go",
	depth: walkParses,
	asks: "every testing.Short() there is, and whether the branch it guards " +
		"skips",
}, {
	fn:    "TestEveryTimingsRecordIsTheSameShape",
	file:  "timingsrecords_test.go",
	depth: walkParses,
	asks: "every `…TimingsTakenOn` record, and whether each carries the five " +
		"machine fields and a reporting arm in its own package",
}, {
	fn:    "TestTheDottedVersionParsersAreTheOnesTheReasonCovers",
	file:  "versionorder_test.go",
	depth: walkParses,
	asks:  "every function that orders dotted versions, by either construction",
}}

// repositoryWalkRow is one decided walk.
type repositoryWalkRow struct {
	fn, file string
	depth    string
	asks     string
	// For a helper: how many calls a run makes, and which test drives it. Zero
	// and "" for a test, which runs once and drives itself.
	runs     int
	drivenBy string
}

// repositoryWalk is one the scan found.
type repositoryWalk struct {
	fn, file string
	line     int
	depth    string
	// How many times a run makes this walk: 1 for a test, and the number of
	// call sites for a helper. Zero for a helper nothing calls, which is a
	// walk that costs nothing and a row describing a dead function.
	runs int
	// The first function found calling it, for a helper. "" for a test.
	drivenBy string
	// Call sites that are inside a loop, as "file:line, in F". Empty for
	// every walk in this package today; a site here is one whose cost `runs`
	// cannot state, because the number of walks it makes is the number of
	// times the loop goes round.
	looped []string
}

// walkDepth is how far into the repository this body goes, or "" for a body
// that does not go there at all.
//
// Deepest wins: a function that calls citingFiles AND go/parser is paying for
// both, and what its row has to say is the larger number.
//
// The go/parser pass is recognised by the CALL and not by the import, because
// an import is a fact about a FILE and several files here parse something
// small — a fragment, their own source — without walking anything. What makes
// a parse a repository-wide parse is that the same body also enumerated the
// repository, which is what the ordering below says.
//
// The import block is still what says which qualifier is go/parser's. Those
// are two different questions and only the first one was ever about the
// import: `parsers` is the calling file's own binding, so an aliased import
// resolves and a `parser.ParseFoo` in a file that imports no such package
// does not. See importnames_test.go.
func walkDepth(body *ast.BlockStmt, parsers map[string]bool) string {
	reads, enumerates, parses := false, false, false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			switch fn.Name {
			case "citingFiles":
				reads = true
			case "repositoryFiles", "walkedFiles":
				enumerates = true
			}
		case *ast.SelectorExpr:
			if pkg, ok := fn.X.(*ast.Ident); ok && parsers[pkg.Name] &&
				strings.HasPrefix(fn.Sel.Name, "Parse") {
				parses = true
			}
		}
		return true
	})
	switch {
	case reads && parses:
		return walkParses
	case reads:
		return walkReads
	case enumerates:
		return walkEnumerates
	}
	return ""
}

// callsTo is how many times this body calls a package-level function by name,
// and the lines of those calls that are inside a loop.
//
// Bare identifiers only, for the reason the git-wrapper census gives: a
// `pkg.Fn(…)` is another package's, and a method call is `x.Fn(…)`, which is
// not what an unqualified call in this package resolves to.
//
// # Why the loops are found as well as the calls
//
// The count is of SITES. That is the same number as the count of walks
// exactly while every site is reached once, and there are two ways it is not:
//
//	for _, c := range cases { checkCitationsResolve(t, c) }   one site, N walks
//	t.Run(c.name, func(t *testing.T){ … })  inside that range — the same, with
//	                                        the site one function literal down
//
// Both are the same construct from this walk's point of view: the call sits
// inside a `for` or a `range` in the same function body, wherever the closures
// between them are, because ast.Inspect descends into a FuncLit like any other
// node and the position arithmetic does not care what it descended through.
//
// There is no such site in this package. What is here is the reporting of one
// if it arrives, because the number it would break is the one the budget above
// is made of, and it would break it silently: a row saying `runs: 1` beside a
// walk that runs eleven times reads exactly like a row that is right.
//
// The loops are collected first and the calls tested against them by POSITION,
// for the reason shortlever_test.go's containsPos gives: one parse, one
// FileSet, and a call inside a loop is exactly a call whose Pos lies between
// that statement's ends. Nothing here has to know what the loop is made of.
func callsTo(fset *token.FileSet, body *ast.BlockStmt, name string) (calls int, looped []int) {
	// Every loop in the body, as a source range. `for {}`, a three-clause
	// `for` and a `range` are one question here — how many times does what is
	// inside this run — so both statement kinds go in.
	type span struct{ from, to token.Pos }
	var loops []span
	ast.Inspect(body, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			loops = append(loops, span{from: node.Pos(), to: node.End()})
		}
		return true
	})
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok || id.Name != name {
			return true
		}
		calls++
		for _, l := range loops {
			if l.from <= call.Pos() && call.Pos() < l.to {
				looped = append(looped, fset.Position(call.Pos()).Line)
				break
			}
		}
		return true
	})
	return calls, looped
}

// mentionsAnEnumeration is whether this source names any entry point, as a
// filter before the parse.
//
// A byte scan cannot produce a false negative here — a call to citingFiles(…)
// contains the bytes `citingFiles` — and its false positives are exactly what
// the parse is for: this file names all three of them in prose and declares
// none of the calls.
func mentionsAnEnumeration(raw []byte) bool {
	for name := range enumerationEntryPoints {
		if bytes.Contains(raw, []byte(name)) {
			return true
		}
	}
	return false
}

// enumerationList is the entry points, for a message.
func enumerationList() string {
	out := make([]string, 0, len(enumerationEntryPoints))
	for name := range enumerationEntryPoints {
		out = append(out, name)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// repositoryWalkList is the decided walks, for a message.
func repositoryWalkList() string {
	out := make([]string, 0, len(repositoryWalks))
	for _, w := range repositoryWalks {
		out = append(out, fmt.Sprintf("%s (%s)", w.fn, w.file))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// walkList is what the scan found, for a message.
func walkList(found []repositoryWalk) string {
	out := make([]string, 0, len(found))
	for _, w := range found {
		runs := ""
		if w.runs != 1 {
			runs = fmt.Sprintf(", ×%d", w.runs)
		}
		out = append(out, fmt.Sprintf("%s (%s:%d, %s%s)", w.fn, w.file, w.line,
			w.depth, runs))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}
