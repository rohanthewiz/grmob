package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
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
// others: each pays its own `git ls-files`, its own read of every tracked file
// and its own parse.
//
// # About the file counts quoted below, which are readings and ARE held
//
// Several sentences here and in timings_test.go price a walk against how many
// files it touches. That number is a reading of a repository on a day — 485
// tracked Go files where verifyTimingsTakenOn was taken — and it goes up with
// every file anybody adds, silently, exactly like the wall clocks beside it.
//
// Unlike the wall clocks, something now checks it. They drifted by four,
// across five sentences at once, before a reader who had not written them
// noticed — and the reason the clocks beside them cannot be armed the same
// way does not apply here: a timing is a fact about one computer and a file
// count is a fact about the TREE, so a figure that disagrees with the tree
// disagrees for everybody. checkProseFileCounts in copies_test.go holds every
// sentence in this repository that quotes the count, off a walk that was
// already enumerating and already parsing.
//
// What that costs a person is that adding a Go file fails a test until these
// sentences are edited, and the failure names each one with its line. The
// form it reads is fixed — see trackedGoFileFigure — so a count written some
// other way is prose rather than a claim, which is the same distinction
// coresAttribution draws with backquotes one file over.
//
// That is getting on for half the package where verifyTimingsTakenOn was
// taken, and it is still the right call at this size. The figure is four
// parse walks at the record's walkParse, two reads at its walkRead and one
// enumeration at its walkEnumerate, summed at both ends and set against its
// wholeFile; the arm below prints the counts on every run.
//
// What is stated here is the PROPORTION and not the two readings it is a
// proportion of, which is a change from how this sentence read for several
// sessions. It carried `1.18–1.40s of a 2.88–2.97s package`, and the record's
// own re-taking list named it as the last figure in this repository that a
// re-taking had to move by hand. A proportion does not drift and a reading
// does: that is the resolution internal/themehistory/main.go reached for the
// same shape, and the general lesson verifyTimingsTakenOn's header states
// three lines under the entry that used to point here — a figure quoted in
// two places is a copy, and attributing the copy does not make it one thing. The walks are INDEPENDENT by design:
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
// `range` drives, is one site and as many walks as the loop is long. There is
// no such site in this package, and the two kinds it would come in are not the
// same finding:
//
//	for range 2 { … }          two walks, and the two is in the source. Read
//	                           off the loop and multiplied into the count, so
//	                           the row states the real number — see loopBound
//	for _, c := range cases    len(cases), which is a run-time fact no parse
//	                           has. REPORTED, which is the honest version of
//	                           a number that does not exist yet
//
// Reporting both identically was one finding doing the work of two: the first
// is a cost this arm declines to compute rather than one it cannot, and a row
// forced to say `runs: 1` beside it is a row that cannot be right. See
// priceCalls
// for how a site is recognised, and why one would otherwise read exactly like
// a row that is right.
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
				fn:       fn.Name.Name,
				file:     name,
				line:     fset.Position(fn.Pos()).Line,
				depth:    depth,
				subtests: subtestSitesIn(fn),
				// A test runs once. A helper runs as often as it is called,
				// and the second pass below is what says how often.
				runs: 1,
			})
		}
	}

	// A bound written as a name, read off this package's own declarations —
	// lazily, over the files already read, using the same memoised parse. See
	// packageLevelInts and loopBound.
	packageInts := packageLevelInts(names, sources, parse)
	// And what each function binds, remembered. The loop below visits every
	// function once per walk NAME — seven of them — and what a function binds
	// does not depend on which walk is being asked about. See boundNamesIn:
	// the answer is an ast.Inspect of the whole body, which is the most
	// expensive thing in this pass and was being recomputed six times out of
	// seven. The parse is memoised, so the same *ast.FuncDecl comes back each
	// time and is the key.
	binds := map[*ast.FuncDecl]map[string]bool{}
	boundNames := func(fn *ast.FuncDecl) map[string]bool {
		if known, done := binds[fn]; done {
			return known
		}
		known := boundNamesIn(fn)
		binds[fn] = known
		return known
	}

	// The second pass: how many times each non-test walker is actually called.
	// A helper that walks the repository and is called from two tests is two
	// walks, and the budget is about walks.
	for i, w := range found {
		if strings.HasPrefix(w.fn, "Test") {
			continue
		}
		calls, in := 0, ""
		// Two lists and not one: a loop whose length is written in the source
		// is a site this pass has already PRICED into `calls`, and one whose
		// length is a run-time fact is a site nothing can price. Only the
		// second is a finding — see priceCalls.
		var looped, priced []string
		// The sites and the package-level declarations: what this pass is
		// counting is how many times the walk runs, and a call in a `var`
		// initializer runs too. Whether the package also declares a method of
		// the name is a question the `besides` pass asks, not this one.
		scan := callSitesOf(w.fn, names, sources, parse)
		for _, c := range scan.in {
			n, sites := priceCalls(fset, c.fn, c.calls, packageInts, boundNames)
			calls += n
			for _, site := range sites {
				where := fmt.Sprintf("%s:%d, in %s (%s)", c.file, site.line,
					c.fn.Name.Name, strings.Join(site.how, " · "))
				if site.known {
					priced = append(priced,
						fmt.Sprintf("%s ×%d", where, site.times))
					continue
				}
				looped = append(looped, where)
			}
			if n > 0 && in == "" {
				in = c.fn.Name.Name
			}
		}
		found[i].runs = calls
		found[i].drivenBy = in
		found[i].looped = looped
		found[i].priced = priced
		found[i].atInit = scan.atPackageLevel
	}

	// Every read declared `besides` a walk still being there, in a subtest of
	// its own.
	//
	// # Why it is not a section of this one
	//
	// A `besides` row is defined as a read that is NOT a repository walk, and
	// its findings were arriving under a test called
	// TestTheRepositoryWideWalksInThisPackageAreTheOnesDecidedOn, sharing its
	// log line and its failure. That is the one thing the field means, said
	// under the one name that contradicts it.
	//
	// t.Run costs nothing here for the reason it costs nothing in
	// copies_test.go: the parse has happened and the subtest reads what it
	// built. What it buys is a name that says what failed and a boundary that
	// keeps this question alive when the walk census stops.
	//
	// And it is BEFORE the reaching-anything arm below for the same reason
	// copies_test.go orders its three the way it does. That arm is a t.Fatalf,
	// which ends the goroutine, so a repository where the walks had been
	// renamed used to report that and say nothing about whether the reads
	// beside them still happen — the exact fault the subtests in
	// copies_test.go were written to end, still open here because this one was
	// added afterwards. Nothing in here reads `found`.
	//
	// # Why it is not a test of its own, given that
	//
	// Reading nothing the census produced is the definition of a question that
	// could stand alone, and this one nearly does. What keeps it here is the
	// three arguments it is handed: `names`, `sources` and the memoised
	// `parse` — a listing of the directory, its files' bytes, and their trees,
	// all of which exist because the census above built them.
	//
	// A top-level test would rebuild all three. That is a second read of this
	// directory and a second parse of the files that mention a `through`, paid
	// on every run to buy a name that t.Run already gives. And it would be
	// spent against repositoryParseBudget, which is four — a number this
	// package treats as a decision rather than a limit, so the trade would not
	// be a quiet one either.
	//
	// The boundary the subtest was moved for is the one that matters: a name
	// of its own, and a failure that does not end this question. Sharing a
	// parse is not what was wrong with sharing a log line.
	t.Run("the reads besides those walks", func(t *testing.T) {
		checkReadsBesidesWalks(t, names, sources, parse)
	})

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
		// once per run — see priceCalls, and the limit this closes one construct
		// along from the helper case it was written for.
		if len(w.looped) > 0 {
			t.Errorf("%s is called from inside a loop whose length this arm "+
				"cannot read: %s.\n\n"+
				"The count beside it is a count of call SITES, and a site "+
				"inside a `for` or a `range` is one site and as many walks as "+
				"the loop has iterations — a `git ls-files`, a read of every "+
				"tracked file and possibly a go/parser pass, each time round. "+
				"So `runs: %d` in its row is a floor rather than a total, and "+
				"the budget below is counting the wrong thing.\n\n"+
				"A loop whose length is WRITTEN DOWN is not this finding: "+
				"`for range 2` and `for i := 0; i < 3; i++` are read off the "+
				"source and multiplied into the count, so the row states the "+
				"real number and nothing is reported. This one is the other "+
				"case — a range over something whose length is decided at run "+
				"time, which no parse has. Hoist the walk out of the loop and "+
				"pass its result in, which is what makes the cost a number "+
				"again, or give the loop a bound loopBound can read.",
				w.fn, strings.Join(w.looped, "; "), row.runs)
		}
		// A call in no function at all. Reported before the count, and
		// instead of it: `runs` is built out of call sites inside function
		// bodies, so a walk reached only from a `var` initializer comes back
		// as zero — and the message the count would print calls it a dead
		// function, which is the scan's limit dressed up as a fact about the
		// code.
		if len(w.atInit) > 0 {
			t.Errorf("%s is called from a package-level declaration: %s.\n\n"+
				"That call is in no function, so nothing here attributes it "+
				"to a caller or prices it against a loop, and `runs: %d` in "+
				"its row is a number this census did not compute. It is also "+
				"not free: a `var` initializer runs when the test binary "+
				"starts, before any test does, so this walk's `git ls-files` "+
				"— and its read of every tracked file, and possibly its "+
				"go/parser pass — is paid on every run including `-run "+
				"NoSuchTest`.\n\n"+
				"This arm cannot tell an initializer that runs at start from "+
				"a call inside a function value that a `var` happens to hold. "+
				"Either is a walk the budgets below are not counting. Move "+
				"the call into the function that needs the result and pass it "+
				"in, which is what makes the cost a number again.",
				w.fn, strings.Join(w.atInit, ", "), row.runs)
		}
		// Only for helpers: a test runs once by definition, and its row says
		// nothing about how often. And only when the count means something —
		// the arm above has just said it does not, and printing a second
		// finding about a number nothing computed would be the same fact
		// twice with the useful half in only one of them. The depth check
		// below still runs either way: what a walk READS is read off its body
		// and does not depend on who calls it.
		if !strings.HasPrefix(w.fn, "Test") && len(w.atInit) == 0 &&
			(row.runs != w.runs || row.drivenBy != w.drivenBy) {
			looped := ""
			if len(w.priced) > 0 {
				looped = fmt.Sprintf("\n\nSome of that count is a loop "+
					"rather than a second call: %s. A bound written in the "+
					"source is read and multiplied in, so the row states "+
					"walks and not sites.", strings.Join(w.priced, "; "))
			}
			t.Errorf("%s:%d is listed as running %d time(s) per run, driven by "+
				"%q, and the source makes %d call(s), the first in %q.%s\n\n"+
				"A helper that walks the repository costs its walk once per "+
				"caller. A second caller is a second `git ls-files`, a second "+
				"read of every tracked file, and a row that still says one — "+
				"which is the cost being understated in the direction nobody "+
				"notices. Zero calls is the other end: a walk nothing makes, "+
				"and a row describing a dead function.",
				w.file, w.line, row.runs, row.drivenBy, w.runs, w.drivenBy,
				looped)
		}
		// What the row SAYS it asks, against how many questions the body
		// actually opens. See subtestSitesIn.
		if want := w.subtests; want > 0 && len(row.asks) != want {
			t.Errorf("%s:%d opens %d subtest(s) and its row lists %d "+
				"question(s): %s.\n\n"+
				"A question here is a subtest. That is not a coincidence in "+
				"the arrangement — it is what copies_test.go's header argues "+
				"for and what the incident behind it was: three questions in "+
				"one function is three questions any one of which can end the "+
				"other two, because each ends in a t.Fatalf over a walk that "+
				"reached nothing, and a Fatalf stops the goroutine it is "+
				"on.\n\n"+
				"So `asks` is a count and not a paragraph, and this is what "+
				"counts it. A row with fewer entries than the body has "+
				"subtests is a walk that has quietly grown a census nobody "+
				"reading the table would see; a row with more is a row "+
				"describing a question that is no longer asked. Both were "+
				"kept by hand until this arm, and the hand missed within one "+
				"session of the field being written.\n\n"+
				"If the subtests are STEPS of one question rather than "+
				"questions of their own, that is the finding and not a false "+
				"one: say so in the row by listing what each actually asks, "+
				"or give the steps one boundary by folding them back into the "+
				"body.",
				w.file, w.line, want, len(row.asks),
				strings.Join(row.asks, "; "))
		}
		if len(row.asks) > walkQuestionBudget {
			t.Errorf("%s:%d answers %d question(s) and the budget is %d: "+
				"%s.\n\n"+
				"See walkQuestionBudget. The question to ask is not whether "+
				"the newest one belongs — it does, and so did the one before "+
				"it, because a walk that already holds the repository's parse "+
				"is the cheapest home for anything needing that parse and "+
				"always will be. It is whether the things on this walk are "+
				"still ONE thing.\n\n"+
				"If they are not, some of them want a walk of their own, and "+
				"that is repositoryWalkBudget's conversation rather than this "+
				"one. If they are, raise this number and write the reason "+
				"beside it.",
				w.file, w.line, len(row.asks), walkQuestionBudget,
				strings.Join(row.asks, "; "))
		}
		if row.depth != w.depth {
			t.Errorf("%s:%d is listed as a walk that %s and it %s.\n\n"+
				"The depths are not decoration: `%s` is one git process and "+
				"a stat per file, `%s` reads every one of them, and `%s` runs "+
				"go/parser over every Go file on top of that — %s, %s and %s "+
				"respectively where verifyTimingsTakenOn was taken. A walk "+
				"that has grown a parse has roughly doubled, and the count "+
				"that decides when a shared parse is worth building is "+
				"repositoryParseBudget, not the total.",
				w.file, w.line, row.depth, w.depth,
				walkEnumerates, walkReads, walkParses,
				verifyTimingsTakenOn.walkEnumerate,
				verifyTimingsTakenOn.walkRead,
				verifyTimingsTakenOn.walkParse)
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
			"question(s) it was asking are unasked.", key, row.depth,
			strings.Join(row.asks, "; "))
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
			"%s where verifyTimingsTakenOn was taken and none of it is "+
			"shared: every Go file in the tree goes through go/parser once "+
			"per arm — 485 tracked Go files where that record was taken — "+
			"and every one of them throws the syntax trees away.\n\n"+
			"A shared parse is a fixture with a lifetime — built once, "+
			"invalidated never, read by tests that no longer say what they "+
			"read — which is why it was not built at four. Either build it "+
			"and say what each arm now depends on, or raise "+
			"repositoryParseBudget with the reading that says the walks are "+
			"still cheap enough to keep separate.",
			parses, walks, repositoryParseBudget, walkList(found),
			verifyTimingsTakenOn.walkParse)
	}

	// Reported as what was found rather than as what should have been found:
	// this line is printed on a failing run too.
	// Any site whose count came from a loop bound rather than from a second
	// call site, said out loud: a `runs: 2` that a reader cannot find two
	// calls for is a row that looks wrong, and the loop it came from is the
	// answer.
	bounded := ""
	for _, w := range found {
		if len(w.priced) > 0 {
			bounded += fmt.Sprintf("\n%s is called inside a loop of a length "+
				"this arm could read, and the count above includes it: %s.",
				w.fn, strings.Join(w.priced, "; "))
		}
	}
	// And how many QUESTIONS those walks are between them answering. The two
	// numbers used to be the same and no longer are: copies_test.go is one
	// parse and three censuses, which is what the parse budget bought. A row
	// growing a fourth entry is a walk that has become a place to put things,
	// and that is worth a number rather than a long field — see repositoryWalks.
	questions := 0
	for _, w := range repositoryWalks {
		questions += len(w.asks)
	}
	t.Logf("%d repository-wide walk(s) per run in this package, %d of them "+
		"parsing every Go file, from %d function(s), asking %d question(s) "+
		"between them: %s. Found by scanning %d Go file(s) in this directory "+
		"and parsing the %d that named something. Their cost is part of "+
		"verifyTimingsTakenOn.wholeFile.%s",
		walks, parses, len(found), questions, walkList(found), len(names),
		len(trees), bounded)
}

// checkReadsBesidesWalks holds every `besides` row to describing a read that
// still happens, and to attributing the figure beside it.
//
// # What is held, and what cannot be
//
//	the function     declared in this package, and called by something. Which
//	                 is what `runs` already asks of a helper, applied to a
//	                 read instead of a walk
//	the attribution  the cost naming the timings record, because every other
//	                 wall-clock figure in this package's prose does
//	the number       nothing. A wall clock is a reading of a machine, which is
//	                 the whole reason this repository keeps records rather
//	                 than asserting timings
//
// Two of the three are exact and the third is impossible, which is worth
// stating in that order: the field is not half-checked by oversight.
func checkReadsBesidesWalks(t *testing.T, names []string,
	sources map[string][]byte, parse func(string) *ast.File) {

	t.Helper()
	rows := 0
	for _, w := range repositoryWalks {
		for _, b := range w.besides {
			rows++
			scan := callSitesOf(b.through, names, sources, parse)
			callers := make([]string, 0, len(scan.in))
			for _, c := range scan.in {
				callers = append(callers, c.fn.Name.Name)
			}
			// A `var x = through(…)` is a caller too. This pass asks only
			// whether the read still happens — it does not price it, which is
			// what the walk census needs a function for — so a package-level
			// initializer answers the question and belongs in the list rather
			// than being the difference between "called" and "dead".
			for _, at := range scan.atPackageLevel {
				callers = append(callers, "the package-level "+at)
			}
			sort.Strings(callers)
			if !scan.declared {
				// The limit named rather than described. A method of this
				// name is a real declaration this scan cannot find calls to,
				// and telling a reader "nothing declares it" would point them
				// at code that is right.
				method := ""
				if scan.asMethod {
					method = fmt.Sprintf("\n\nThis package DOES declare a "+
						"method `%s`, and that is the finding: `through` has "+
						"to name a function, because this scan reads bare "+
						"identifiers. A call spelled `x.%s(…)` needs the "+
						"receiver's type to resolve, which is the cost every "+
						"walk in this package declines — so a method here is "+
						"a read nothing can hold. Point `through` at whatever "+
						"is called without a receiver, or say in the row why "+
						"there is nothing to point it at.", b.through,
						b.through)
				}
				t.Errorf("%s's row says it reads %s through %s, and nothing "+
					"in this directory declares a `func %s`.\n\n"+
					"That row is the only place this read is counted: the "+
					"budgets are about repository walks and this is not one, "+
					"so a row describing a function that has gone is a read "+
					"nobody is watching — or a row for a read that no longer "+
					"happens, and a figure in the log that is about "+
					"nothing.\n\nIf the read has moved, move the row with "+
					"it; if it has gone, take the row out.%s",
					w.fn, b.reads, b.through, b.through, method)
				continue
			}
			if len(callers) == 0 {
				t.Errorf("%s's row says it reads %s through %s, and nothing "+
					"in this directory calls `%s`.\n\n"+
					"A declared function nothing calls is a read that does "+
					"not happen, and the row beside it is a cost being "+
					"reported on every green run for work this package no "+
					"longer does — the opposite failure from the one this "+
					"field exists for, and as invisible.\n\nEither the "+
					"call has gone and the row should go with it, or it has "+
					"moved behind a spelling this scan cannot see: bare "+
					"identifiers only, so a call through a method or a "+
					"function value is one `through` cannot name. A call in a "+
					"`var` initializer IS seen — see packageLevelCallsTo — so "+
					"it is not that.",
					w.fn, b.reads, b.through, b.through)
				continue
			}
			// And the figure beside it being attributed. The NUMBER cannot
			// be held — a wall clock is a reading of a machine, which is why
			// this repository keeps records instead of asserting timings —
			// but which machine it was taken on can, and every other
			// wall-clock figure in this package's prose says so. A cost with
			// no record behind it is a number a reader on a different
			// computer has no way to place, which is the whole thing
			// verifyTimingsTakenOn exists to end.
			if !strings.Contains(b.costs, timingsRecordName) {
				t.Errorf("%s's row prices its read through %s at %q, and that "+
					"figure does not name %s.\n\n"+
					"A wall clock is a reading of a machine. This repository "+
					"does not assert one — that is why the record exists — "+
					"and the thing it does instead is attribute it, so that a "+
					"reader holding a different number knows whether they are "+
					"looking at a regression or at a different computer.\n\n"+
					"Every other figure in this package's prose says where it "+
					"was taken. Say it here: the cost is part of the same "+
					"run, and a number in a field with no machine behind it "+
					"is the state every timing in this repository was in "+
					"before the record.",
					w.fn, b.through, b.costs, timingsRecordName)
			}
			t.Logf("%s reads %s through %s, called from %s. %s", w.fn, b.reads,
				b.through, strings.Join(callers, ", "), b.costs)
		}
	}
	t.Logf("%d read(s) besides the repository walks, which the budgets there "+
		"do not govern and which are therefore the ones worth naming.", rows)
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

// The record every wall-clock figure in this package is attributed to.
//
// Named rather than spelled inline because the `besides` arm quotes it in a
// message and compares against it, and a constant one of those two knows about
// and the other does not is a check that passes on the wrong string.
const timingsRecordName = "verifyTimingsTakenOn"

// How many of them may parse every Go file in the tree.
//
// Four, and this is the half that costs. The other two walks read bytes and
// stop; these four hand every Go file in the tree — 485 tracked Go files
// where verifyTimingsTakenOn was taken — to go/parser, build the syntax
// trees, ask one question each and drop them.
//
// A fifth is where a shared parse becomes the cheaper of two bad options —
// which is a real trade and not an obvious one, so it is written down here
// rather than decided in advance.
const repositoryParseBudget = 4

// How many questions one walk may answer.
//
// # Why there has to be a number
//
// repositoryParseBudget is full, which means a new question needing a
// repository-wide parse has exactly one cheap home: the walk that already has
// one. That is the right answer and it is the answer EVERY TIME — the parse
// budget is full whoever is asking, the shared walk always has the parse, and
// the question always needs no enumeration of its own.
//
// A reason that always wins is not a reason. Two questions arrived on that
// walk in a single session, each with that argument, each correct, and
// nothing in the repository would have objected at nine. What the arrangement
// was quietly becoming is the thing every row in this table exists to stop: a
// walk that is a place to put things.
//
// # Why six, and why it was five
//
// It is the same kind of number as timingsRecordCopies: not a measured limit
// but a line drawn at the current state, so that the next step past it is
// taken deliberately. The three budgets bound the three ways this cost grows
// — how many walks, how many of them parse, and how much any one of them is
// carrying — and this is the one that had no number at all.
//
// It was five, and it was raised the session after it was written, which is
// the thing worth recording about it. The question that raised it holds every
// test named in this repository's prose to being a test that exists, and it
// was written because a rename had left five sentences naming a function that
// was gone, through every verification path clean.
//
// The raise is the first of the two honest answers this constant's doc
// offered, and the argument had to be made rather than assumed:
//
//	it needs both halves of one parse   what tests EXIST comes off the
//	                                    declarations, what prose POINTS AT
//	                                    comes off the comments, and no other
//	                                    walk has both
//	a walk of its own is not available  repositoryWalkBudget is 7 against 7,
//	                                    and that budget's conversation is the
//	                                    more expensive one
//	it is not the comment question      that question is two rules about
//	                                    characters in a line and has its own
//	                                    reaching arm; this one's reaching arm
//	                                    is that the DECLARATIONS were found,
//	                                    which is a different failure and a
//	                                    much louder one
//
// The last row is the one that mattered. Folding this into the comment
// question would have kept the number at five, and a number kept at five by
// arranging the questions to suit it is worth nothing at all.
//
// The seventh question is the same decision again. Either the questions on
// that walk are no longer one thing and some of them want a walk of their
// own, which is repositoryWalkBudget's conversation; or they are, and this
// moves with the reason written beside it.
const walkQuestionBudget = 6

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
//
// # Why it is a list and not a sentence
//
// The field was written to hold what ONE walk asks, because that was the unit:
// one walk, one question, one row. copies_test.go stopped being that — it is
// one parse answering three questions, which is the arrangement the parse
// budget forced and is the right one — and the row it left behind was three
// sentences run together in a field built for one.
//
// A paragraph is not countable. A list is: a row with four entries is a walk
// that has quietly become four censuses sharing a parse, and the number is
// printed on every green run rather than being something a reader notices by
// finding the field long. That is the same move as `runs` — the cost of a walk
// was a sentence until something counted it.
var repositoryWalks = []repositoryWalkRow{{
	fn:       "checkCitationsResolve",
	file:     "checknumbering_test.go",
	depth:    walkReads,
	runs:     1,
	drivenBy: "TestTheBrowserChecksAreOneNumberedSequence",
	asks: []string{
		"every `check N` citation in the repository, in one numbered " +
			"sequence — which is why citingFiles reads every file rather " +
			"than every Go file. A HELPER and not a test: it walks once per " +
			"caller, and it also asks repositoryFiles separately first, on " +
			"purpose, so that a git which succeeds and lists nothing is told " +
			"apart from a machine with no git",
	},
}, {
	fn:    "TestTheCitationSkipsGitAlreadyMakes",
	file:  "checknumbering_test.go",
	depth: walkEnumerates,
	asks: []string{
		"which of citationSkipDirs git's own exclude rules already make, " +
			"which is a question about the FILE LIST and not about any " +
			"file's contents",
	},
}, {
	fn:    "TestEveryGitListingAsksForNulSeparatedPaths",
	file:  "gitquoting_test.go",
	depth: walkParses,
	asks: []string{
		"every git invocation in Go source that lists paths, and whether it " +
			"asks for them NUL-separated",
	},
}, {
	fn:    "TestEveryGitListingInAScriptAsksForNulSeparatedPaths",
	file:  "gitscript_test.go",
	depth: walkReads,
	asks: []string{
		"the same rule in shell scripts, which are lexed rather than parsed " +
			"— so this one reads bytes and never reaches go/parser",
	},
}, {
	fn:    "TestTheShortLeversAreTheOnesThisRepositoryHasDecidedOn",
	file:  "shortlever_test.go",
	depth: walkParses,
	asks: []string{
		"every testing.Short() there is, and whether the branch it guards " +
			"skips",
	},
}, {
	// Six questions and one parse, because a fifth repository-wide parse is
	// the decision repositoryParseBudget exists to force — and this walk is
	// the answer that decision has, which is why questions land here rather
	// than becoming walks. Each is a reading of what the one walk has already
	// built, and each is a subtest with its own failure boundary. What stops
	// that arrangement absorbing every future question is walkQuestionBudget;
	// see sharedparse_test.go.
	fn:    "TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn",
	file:  "sharedparse_test.go",
	depth: walkParses,
	besides: []besidesRow{{
		through: "identifiersIn",
		reads: "the two directories that carry a timings record, for the " +
			"second direction of the cores-note check — which asks whether " +
			"the terms a note names still exist, and cannot be answered off " +
			"the repository walk because the walk throws its trees away. " +
			"os.ReadDir per directory, the bytes scanned first, and only the " +
			"files that hold one of the terms parsed",
		costs: "0.010s where verifyTimingsTakenOn was taken, measured by " +
			"taking the call out and putting it back over seven takings of " +
			"sixty runs — 12.19–12.28s against 11.57–11.99s",
	}},
	asks: []string{
		"the `…TimingsTakenOn` records: whether each carries the five " +
			"machine fields, a reporting arm, and a cores note that names " +
			"every term in its package that scales and nothing that has gone",
		"the declarations this repository keeps two copies of — the " +
			"import-resolving helpers, the registry they share, the band " +
			"reader and the verdict levers both timings records use — held " +
			"to being the same declarations in both packages, and every " +
			"other name both packages declare held to being in one of the " +
			"two lists",
		"every import path those helpers are asked about, held to being one " +
			"this module could actually import",
		"every sentence in the repository quoting how many Go files the tree " +
			"holds, held to the count this walk's own enumeration just made " +
			"— the one shape in that file whose second copy is not source",
		"every comment line in the repository, held to two rules about text " +
			"nothing else reads as text: a line that is one comment written " +
			"twice, and a tab anywhere but the leading indent. See " +
			"commenttext_test.go, which owns the rules and the check",
		"every Go test named in a comment or a string constant, held to " +
			"being a test this repository has — the declarations and the " +
			"prose being the two halves of this one parse. See " +
			"prosenames_test.go",
	},
}, {
	fn:    "TestTheDottedVersionParsersAreTheOnesTheReasonCovers",
	file:  "versionorder_test.go",
	depth: walkParses,
	asks: []string{
		"every function that orders dotted versions, by either construction",
	},
}}

// repositoryWalkRow is one decided walk.
type repositoryWalkRow struct {
	fn, file string
	depth    string
	// What this walk asks the repository, one entry per question. See the
	// list above for why the unit is a question rather than a row.
	asks []string
	// Reads this walk makes that are NOT repository-wide, and what each one
	// costs. Empty for a walk that only walks the repository, which is every
	// row here but one. See besidesRow.
	besides []besidesRow
	// For a helper: how many calls a run makes, and which test drives it. Zero
	// and "" for a test, which runs once and drives itself.
	runs     int
	drivenBy string
}

// besidesRow is one read a walk makes that is not repository-wide.
//
// # Why this exists as a row at all
//
// The budgets above are about repository walks, and a read of two directories
// is not one — which is a correct exemption and is also exactly how four
// repository-wide parses came to exist before anything counted them. Something
// small enough not to be worth a row is something nothing is watching, and the
// next one is as easy to add as the first.
//
// # And why it names a FUNCTION
//
// The first version of this was a sentence, and a sentence about a read is a
// claim nothing can be wrong about: the call deleted, the function renamed,
// the cost changed by a factor — every one of those leaves a row that reads
// exactly as it did. `asks` does not have that problem, because it describes a
// walk the census FINDS, so a row for a walk that has gone is a finding and an
// unlisted walk is another.
//
// `through` is the handle that gives this the same property. It names the
// function the read goes through, and the pass below holds it to being
// declared in this package and to being called by something — which is what
// `runs` already does for a helper, applied to a read instead of a walk. The
// COST is still a measurement nobody re-takes automatically, and that part is
// a written figure like every other number in this repository's prose; what is
// no longer possible is the row outliving the read.
type besidesRow struct {
	// The function the read goes through. Held to being declared here and to
	// being called — see the pass over `besides` below.
	through string
	// What it reads, for a reader deciding whether it should have been a walk.
	reads string
	// What it costs, and how that was measured. Held to naming the timings
	// record, which is the only thing a wall-clock figure in this repository
	// can be held to — see the pass over `besides`.
	costs string
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
	// Call sites inside a loop whose length no parse can read, as
	// "file:line, in F (how)". Empty for every walk in this package today; a
	// site here is one whose cost `runs` cannot state, because the number of
	// walks it makes is the number of times the loop goes round.
	looped []string
	// And the ones inside a loop that DOES say how long it is, which are
	// already multiplied into `runs`. Not a finding — they are carried so the
	// log line can say that a count above one came from a bound in the source
	// rather than from a second call site.
	priced []string
	// How many subtests the body opens at its own level, which is how many
	// QUESTIONS this walk asks — see the asks check for why those are the
	// same number, and for the one case where they are not.
	subtests int
	// The `var`/`const` declarations whose value calls this walk, as
	// "name (file)". Empty for every walk in this package today. A call here
	// is not in `runs`: it is in no function, so there is no caller to
	// attribute it to and no loop to price it against — and it is the one
	// place a repository walk can hide from this census while still costing a
	// run. See packageLevelCallsTo.
	atInit []string
}

// subtestSitesIn is how many subtests this function opens at its own level.
//
// # Why this is the count of a walk's QUESTIONS
//
// copies_test.go is the argument, and it is not a stylistic one. Each of its
// questions ends in a t.Fatalf over a walk that reached nothing — the only
// honest thing to do about a census that passed because it found nothing to
// look at — and a Fatalf ends the GOROUTINE. Three questions in one function
// was three questions any one of which could silence the other two, and it
// did: a repository where the timings records had been renamed reported that
// and said nothing at all about whether the import resolver's two copies were
// still in step.
//
// So a question gets a t.Run, and the number of them is a number this census
// can read instead of one the table keeps by hand. Which it needed: the
// fourth question was added to that walk and its row still said three, in the
// same session that wrote the field to be countable.
//
// # What is counted, and what is not
//
//	t.Run at the body's own level      one question
//	t.Run inside a function literal    not counted. That is a subtest of a
//	                                   subtest, and the `t` it is called on is
//	                                   the inner one whatever it is spelled
//	t.Run inside a `for`               one SITE and as many subtests as the
//	                                   loop is long, counted as one — the same
//	                                   understatement `runs` makes for a walk
//	                                   in a loop, and for the same reason: the
//	                                   length is a run-time fact no parse has
//
// The receiver has to be the function's own *testing.T, found by its
// parameter name rather than by assuming it is spelled `t`. A method called
// `Run` on something else — a fixture, an exec.Cmd — is not a subtest, and
// counting one would make a row look short by a question that does not exist.
func subtestSitesIn(fn *ast.FuncDecl) int {
	name := testingTParamOf(fn)
	if name == "" || fn.Body == nil {
		return 0
	}
	sites := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		// Function literals are where a subtest's own body lives, and a
		// t.Run in there belongs to that subtest and not to this function.
		if _, isLit := n.(*ast.FuncLit); isLit {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Run" {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == name {
			sites++
		}
		return true
	})
	return sites
}

// testingTParamOf is the name this function calls its *testing.T by, or "" if
// it takes none.
//
// Read off the parameter rather than assumed, for the reason
// importnames_test.go reads a qualifier off the import: `t` is a convention
// and not a rule, and a census that only recognises the convention reports a
// function written the other way as having no subtests at all — which is a
// row that looks right while counting nothing.
func testingTParamOf(fn *ast.FuncDecl) string {
	if fn.Type.Params == nil {
		return ""
	}
	for _, field := range fn.Type.Params.List {
		star, ok := field.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sel, ok := star.X.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "T" {
			continue
		}
		pkg, ok := sel.X.(*ast.Ident)
		// The package qualifier is `testing` as this repository imports it —
		// every file here takes the plain import, and a renamed one would be
		// a finding importnames_test.go is the right place for rather than
		// a second resolver in this file.
		if !ok || pkg.Name != "testing" {
			continue
		}
		if len(field.Names) == 0 {
			// An unnamed parameter cannot be called, so nothing in the body
			// can open a subtest on it.
			return ""
		}
		return field.Names[0].Name
	}
	return ""
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

// loopSite is one call site that sits inside a loop, and what a parse can say
// about how many times it runs.
//
// `times` is the product of the bounds of every loop around it and `known`
// says whether a parse could read them all. The two are separate because the
// finding is different in each case — a bound this arm CAN read is a cost it
// declines to compute if it only reports it, and a bound it cannot read is a
// number that does not exist until the run happens.
type loopSite struct {
	line  int
	times int
	known bool
	// How the bounds were read, as "for range 2" or "for range cases", so a
	// message can say which loop it could not price rather than only that
	// there was one.
	how []string
}

// priceCalls is what these call sites cost: how many times this body actually
// makes them, and the ones inside a loop whose length no parse can read.
//
// The sites are found by callSitesOf and handed in. Two things ask this file
// where a function is called and only one of them cares what it costs, so
// finding is shared and pricing is here.
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
//
// # Which of those loops is a number and which is a question
//
// Reporting every loop identically was one finding doing the work of two.
// `for range 2 { … }` is two walks and the two is IN THE SOURCE; `for _, c :=
// range cases` is len(cases), which is a run-time fact a parse cannot have.
// The first is a cost this arm was declining to compute rather than one it
// could not, and a row that has to say `runs: 1` beside a bounded loop is a
// row that cannot be right.
//
// So a bound a parse can read is READ — see loopBound — and the site is priced
// rather than reported: `calls` comes back as the product, the row states it,
// and the budget counts it. A bound it cannot read comes back as a site with
// `known` false, which is the finding that was there before.
//
// Nested loops multiply, and one unreadable bound anywhere in the nest makes
// the whole product unknown: two known loops around one unknown one is still
// "as many as that range is long", times a constant nobody needs to be told.
//
// # What the pricing is loose about, which is the reporting direction
//
// A call inside a loop whose enclosing FuncLit is never invoked — stored in a
// variable, passed somewhere that drops it — is priced as though the loop ran
// it. That overstates the cost, which is the direction this arm is allowed to
// be wrong in: its whole purpose is to stop a walk's price being understated
// where nobody looks. There is no such site here, and `t.Run` and `defer` —
// the two ways a closure in a test actually reaches a call — both run it.
func priceCalls(fset *token.FileSet, fn *ast.FuncDecl, found []*ast.CallExpr,
	packageInts func(string) (int, bool),
	boundNames func(*ast.FuncDecl) map[string]bool) (calls int,
	sites []loopSite) {

	body := fn.Body
	// A bound written as a name is read off this package's declarations — but
	// only when the name is not bound inside this function. See boundNamesIn:
	// a local `n` shadowing a package-level `n` would otherwise be priced at
	// the package's number, which is a count invented out of a coincidence.
	//
	// Handed in rather than computed, because this is called once per walk
	// name over the same functions and the answer does not vary with the walk.
	shadowed := boundNames(fn)
	bound := func(id string) (int, bool) {
		if shadowed[id] || packageInts == nil {
			return 0, false
		}
		return packageInts(id)
	}
	// Every loop in the body, as a source range with whatever bound could be
	// read off it. `for {}`, a three-clause `for` and a `range` are one
	// question here — how many times does what is inside this run — so both
	// statement kinds go in.
	type span struct {
		from, to token.Pos
		times    int
		known    bool
		how      string
	}
	var loops []span
	ast.Inspect(body, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			times, known, how := loopBound(node, bound)
			loops = append(loops, span{from: node.Pos(), to: node.End(),
				times: times, known: known, how: how})
		}
		return true
	})
	// The sites themselves were found by callSitesOf and are handed in, which
	// is the whole of the split: finding a bare call to a name is one
	// question this file asks in two places, and pricing it is a question only
	// this one asks.
	for _, call := range found {
		// Every loop around this call, not the innermost: a site two `range`s
		// deep runs the product of the two, and a row that quoted only the
		// inner one would be wrong by the outer one's length.
		times, known, inside := 1, true, false
		var how []string
		for _, l := range loops {
			if l.from > call.Pos() || call.Pos() >= l.to {
				continue
			}
			inside = true
			how = append(how, l.how)
			if !l.known {
				known = false
				continue
			}
			times *= l.times
		}
		if !inside {
			calls++
			continue
		}
		if known {
			// Priced. The site costs what the loops make it cost, and the row
			// beside it is expected to say so — which is the whole difference
			// between a number this arm computes and one it hands back as a
			// question.
			calls += times
		} else {
			// Still one site as far as the count goes, because the alternative
			// is inventing a number. The finding below is what says the count
			// is a floor rather than a total.
			calls++
		}
		sites = append(sites, loopSite{line: fset.Position(call.Pos()).Line,
			times: times, known: known, how: how})
	}
	return calls, sites
}

// loopBound is how many times this loop's body runs, when the source says so.
//
// # Why only these shapes
//
// The question is not "what does this loop do" — that is the halting problem
// with extra steps — it is "is the number of iterations written down here". It
// is written down in exactly two constructions, and both of them are the ones
// somebody actually writes around a call they meant to make a fixed number of
// times:
//
//	for range 3                 the count is the literal
//	for range []T{a, b, c}      the count is the number of elements. An array
//	                            type with a written length is that length
//	                            instead, because `[8]int{}` ranges eight times
//	                            and has one element
//	for i := 0; i < 3; i++      the count is the difference, with `<=` one
//	                            more. Only a written start, a written bound
//	                            and a ++ post — anything else is arithmetic
//	                            this is not going to do
//
// Everything else comes back unknown, and unknown is not a failure: it is the
// case the report above exists for. In particular `for _, c := range cases` —
// the shape that actually threatens the walk count — is a slice whose length
// is decided somewhere else, and pretending to read it by finding the `cases`
// declaration would be a dataflow walk in a census that declines those for the
// reason gitquoting_test.go writes down.
//
// # A name is written down too, when it is a package-level integer
//
// `for i := 0; i < enumWorkers; i++` is a bound this repository HAS written
// down. It is not a run-time fact and it is not dataflow: enumWorkers is a
// package-level declaration with an integer on the right of it, sitting in the
// same directory this walk has already read, which is how packageLevelNames
// answers a question of exactly this shape for a hundredth of a second.
//
// Reporting it was the arm declining to compute a cost rather than being
// unable to — the same thing `for range 2` was before the literal case was
// read — so the `bound` resolver is handed every identifier that appears where
// a number would do, and a name it can settle is priced like a literal.
//
// Two things it will not do. A name whose initialiser is anything but an
// integer literal comes back unknown, because `min(NumCPU, 8)` is a bound that
// depends on the machine and 8 is its ceiling rather than its value — and
// pricing a walk at its ceiling is the overstating direction on a number the
// budget is made of. And a name bound anywhere inside the function is refused
// outright: see boundNamesIn.
//
// `how` says which name and what it resolved to, because a row reading
// `runs: 8` beside a loop written `i < enumWorkers` is a number a reader
// cannot check without opening another file.
//
// A string range is deliberately unknown: `for range "héllo"` goes round once
// per RUNE and the literal's length is in bytes, and a census that got that
// wrong by one on a non-ASCII literal would be worse than one that asks.
//
// `how` is the loop as a reader would say it, for the message: "for range
// cases" names the thing that could not be priced, and "there is a loop" does
// not.
func loopBound(n ast.Node, bound func(string) (int, bool)) (times int,
	known bool, how string) {

	switch loop := n.(type) {
	case *ast.RangeStmt:
		how = "for range " + exprText(loop.X)
		switch x := loop.X.(type) {
		case *ast.Ident:
			// `for range n`, which ranges over an integer. A name here is the
			// same question as a name in a `<` — see the header.
			if v, ok := bound(x.Name); ok {
				return v, true, fmt.Sprintf("%s, %s = %d", how, x.Name, v)
			}
			return 0, false, how
		case *ast.BasicLit:
			if x.Kind == token.INT {
				if v, err := strconv.Atoi(x.Value); err == nil && v >= 0 {
					return v, true, how
				}
			}
			// A string literal ranges by rune and its length is in bytes.
			return 0, false, how
		case *ast.CompositeLit:
			// `[N]T{…}` is N iterations whatever the literal fills in; a
			// slice or a map is one per element.
			if arr, ok := x.Type.(*ast.ArrayType); ok && arr.Len != nil {
				if lit, ok := arr.Len.(*ast.BasicLit); ok && lit.Kind == token.INT {
					if v, err := strconv.Atoi(lit.Value); err == nil && v >= 0 {
						return v, true, how
					}
				}
				return 0, false, how
			}
			return len(x.Elts), true, how
		}
		return 0, false, how
	case *ast.ForStmt:
		how = "for " + forHeaderText(loop)
		if loop.Cond == nil {
			// `for {}` and `for … ;; …` run until something breaks, which is
			// not a number in this file.
			return 0, false, how
		}
		cond, ok := loop.Cond.(*ast.BinaryExpr)
		if !ok {
			return 0, false, how
		}
		// The counter, its start and its bound all have to be a number this
		// can read for the difference to mean anything.
		name, from, ok := counterStart(loop.Init, bound)
		if !ok {
			return 0, false, how
		}
		id, ok := cond.X.(*ast.Ident)
		if !ok || id.Name != name {
			return 0, false, how
		}
		to, ok := intValue(cond.Y, bound)
		if !ok {
			return 0, false, how
		}
		// Which name the number came from, if it came from one. A row saying
		// `runs: 8` beside `i < enumWorkers` is otherwise a figure a reader
		// has to go and look up.
		if named, isName := cond.Y.(*ast.Ident); isName {
			how = fmt.Sprintf("%s, %s = %d", how, named.Name, to)
		}
		inc, ok := loop.Post.(*ast.IncDecStmt)
		if !ok || inc.Tok != token.INC {
			return 0, false, how
		}
		if incID, ok := inc.X.(*ast.Ident); !ok || incID.Name != name {
			return 0, false, how
		}
		switch cond.Op {
		case token.LSS:
			// Fewer than none is none, not a negative number of walks.
			return max(to-from, 0), true, how
		case token.LEQ:
			return max(to-from+1, 0), true, how
		}
		return 0, false, how
	}
	return 0, false, ""
}

// counterStart is the name and starting value of an `i := 0` init statement.
//
// The starting value goes through the same resolver the bound does, so
// `for i := firstCheck; i < lastCheck; i++` is as readable as `0` and `3` are.
// Named counterStart rather than literalInit because the start no longer has
// to be a literal — only a number this arm can settle without running
// anything.
func counterStart(init ast.Stmt, bound func(string) (int, bool)) (name string,
	from int, ok bool) {

	assign, ok := init.(*ast.AssignStmt)
	if !ok || assign.Tok != token.DEFINE || len(assign.Lhs) != 1 ||
		len(assign.Rhs) != 1 {
		return "", 0, false
	}
	id, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		return "", 0, false
	}
	from, ok = intValue(assign.Rhs[0], bound)
	if !ok {
		return "", 0, false
	}
	return id.Name, from, true
}

// intValue is an expression's value when the source says what it is: a
// non-negative integer literal, or a name the resolver can settle.
//
// The resolver is the only place a name is read, and it is what decides
// whether `enumWorkers` is a number or a question — see loopBound's header and
// packageLevelInts.
func intValue(e ast.Expr, bound func(string) (int, bool)) (int, bool) {
	if v, ok := intLit(e); ok {
		return v, true
	}
	id, ok := e.(*ast.Ident)
	if !ok || bound == nil {
		return 0, false
	}
	return bound(id.Name)
}

// callsIn is every bare call to one name inside one function.
type callsIn struct {
	// The file the function is in, for a message.
	file string
	fn   *ast.FuncDecl
	// The call expressions themselves, in source order. Kept rather than
	// counted, because what a caller does with them differs: the walk census
	// prices each one against the loops around it, and the `besides` pass only
	// wants to know that somebody calls the function at all.
	calls []*ast.CallExpr
}

// callSitesOf is every bare call to this name in this directory, grouped by
// the function it is in, and whether this directory declares it.
//
// # Why the finding and the pricing are separate
//
// There used to be two of these — one that counted calls and priced the loops
// around them, and one that only asked whether a function was declared and
// reached. Two scans of the same trees looking for the same construct, kept
// apart by an argument about what the ANSWER means rather than about the walk.
// That argument is real and it is about priceCalls, which is now the only
// thing that prices: this finds sites, and each caller decides what a site is
// worth to it.
//
// So the walk census takes these and multiplies the loop bounds in, because
// what it is counting is repository walks and the budget is made of that
// number. The `besides` pass takes the same sites and reads off the names,
// because a read the budgets do not govern needs to be known to happen and not
// to be priced — and deciding what a `besides` number means when the read is
// in a loop is a question nothing has yet had to ask.
//
// # The scan
//
// The one everything in this file uses: the bytes say which files could
// mention the name, and the memoised parse decides. A file that does not
// contain the name cannot declare or call it, and this file's own prose —
// which names every walk and every `through` there is — is read and discarded
// rather than counted.
//
// Bare identifiers only, for the reason the git-wrapper census gives: a
// `pkg.Fn(…)` is another package's, and a method call is `x.Fn(…)`, which is
// not what an unqualified call in this package resolves to.
//
// The declaration itself is never one of the callers, which would otherwise
// report a recursive helper as its own caller and a non-recursive one as
// having none.
//
// # What `asMethod` is for
//
// Bare identifiers are what this finds, so a METHOD of the same name is a
// declaration it cannot find calls to. Reporting that as "nothing declares a
// func of this name" is true and useless — it points a reader at the code when
// the answer is about the scan. `asMethod` says which of the two it is, so the
// finding can name the limit instead of describing its symptom.
//
// # Why one value and not three results
//
// Neither caller wants all three. The walk census takes only `in`; the
// `besides` pass takes `declared`, `asMethod` and the names off `in`. A
// signature that is the union of two callers' needs grows by one every time
// something asks a new question of the same scan — which is how this reached
// three, `asMethod` having been the last — and every growth edits both call
// sites, including the one that did not want the answer.
//
// A struct moves that: a fourth thing this scan can notice is a field, the
// caller that cares reads it, and the other one is not touched. The cost is
// that a caller must name what it wants, which is the same thing the blank
// identifiers were doing less legibly.
func callSitesOf(name string, names []string, sources map[string][]byte,
	parse func(string) *ast.File) callSites {

	var out callSites

	for _, file := range names {
		if !bytes.Contains(sources[file], []byte(name)) {
			continue
		}
		tree := parse(file)
		if tree == nil {
			continue
		}
		for _, d := range tree.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if fn.Name.Name == name {
				if fn.Recv == nil {
					out.declared = true
					continue
				}
				// A METHOD of the same name. Not a declaration this can find
				// calls to — those are spelled `x.Fn(…)` and resolving the
				// receiver means type information this walk declines — but
				// knowing it is there turns a misleading finding into an
				// exact one. See the `besides` pass: a row pointed at a
				// method gets told that, rather than being told its function
				// does not exist.
				out.asMethod = true
				continue
			}
			var calls []*ast.CallExpr
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == name {
					calls = append(calls, call)
				}
				return true
			})
			if len(calls) > 0 {
				out.in = append(out.in, callsIn{file: file, fn: fn,
					calls: calls})
			}
		}
		out.atPackageLevel = append(out.atPackageLevel,
			packageLevelCallsTo(name, file, tree)...)
	}
	sort.Strings(out.atPackageLevel)
	return out
}

// packageLevelCallsTo is the `var`/`const` declarations in one file whose
// value contains a bare call to this name, rendered for a message.
//
// # Why this is looked for at all
//
// Everything above walks FuncDecl bodies, which is where a call to a helper
// normally is. A `var x = someWalk(root)` is not there, and a scan that only
// reads bodies gives the same answer for it as for a function nothing calls at
// all — while the Go runtime runs that initializer before any test does.
//
// Both callers then report something false. The walk census says `runs: 0` and
// calls the walk dead when it in fact runs once per binary; the `besides` pass
// says nothing calls the read when something does. That is the shape asMethod
// was added for, one construct along: a limit of the scan, described as a fact
// about the code, pointing a reader at something that is right.
//
// # What it does not decide
//
// Whether the call RUNS at init. `var f = func() { someWalk() }` puts the call
// inside a function value, which runs when something invokes f, and telling
// the two apart is a data-flow question this scan is not. Both are reported
// the same way and the message says so — because either way the count beside
// the row is not a number this pass computed, which is the thing a reader
// needs to know.
func packageLevelCallsTo(name, file string, tree *ast.File) []string {
	var out []string
	for _, d := range tree.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, sp := range gen.Specs {
			vs, ok := sp.(*ast.ValueSpec)
			if !ok {
				continue
			}
			hit := false
			for _, v := range vs.Values {
				ast.Inspect(v, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if id, ok := call.Fun.(*ast.Ident); ok && id.Name == name {
						hit = true
					}
					return true
				})
			}
			if !hit {
				continue
			}
			// Named by the declaration rather than by a line number: the
			// thing a reader has to go and look at is the `var`, and every
			// other message in this file addresses a declaration by name.
			for _, n := range vs.Names {
				out = append(out, fmt.Sprintf("%s (%s)", n.Name, file))
			}
		}
	}
	return out
}

// callSites is what one callSitesOf scan of one name found.
//
// Three answers to one walk of the trees, and no caller reads all three — see
// callSitesOf's note on why they arrive together in a value rather than as
// results a caller has to spell blanks for.
type callSites struct {
	// Whether this directory declares a `func name` with no receiver: the
	// only spelling a bare call resolves to.
	declared bool
	// Whether it declares a METHOD of that name. Not a declaration this scan
	// can find calls to, and the difference between a finding that names its
	// own limit and one that sends a reader to correct code.
	asMethod bool
	// The bare calls, grouped by the function they are in, in source order.
	in []callsIn
	// The `var`/`const` declarations whose value calls the name, sorted. Not
	// in `in` because they are in no function: there is no body to price a
	// loop against and no caller name to attribute the run to. See
	// packageLevelCallsTo.
	atPackageLevel []string
}

// boundNamesIn is every identifier this function binds: its receiver, its
// parameters and results, and everything declared anywhere in its body.
//
// # Why the shadow is refused rather than resolved
//
// A bound written as a name is read off the package's declarations, and that
// is only sound while the name in the loop IS the package's. A local `n`,
// a parameter `n`, a `for _, n := range …` — each of them makes
// `for i := 0; i < n; i++` a run-time fact spelled exactly like a written-down
// one, and pricing it at the package-level `n` would be a number invented out
// of two declarations sharing a name.
//
// Resolving it properly means scope, which means a walk that knows which
// block a call is in and what each of them binds — dataflow, which this census
// declines for the reason gitquoting_test.go writes down. Refusing every name
// the function binds ANYWHERE costs a real bound now and then — a body that
// happens to declare `enumWorkers` somewhere unrelated makes a loop elsewhere
// in it unreadable — and what it costs is a finding, which is the direction
// this arm is allowed to be wrong in. It cannot cost a wrong number.
//
// The body is read with ast.Inspect and therefore includes every FuncLit
// inside it, which is right: a call in a closure is a call in this function as
// far as the position arithmetic goes, so a name the closure binds is a name
// that could be the one in the loop.
//
// # Called once per function and not once per asking
//
// This is the most expensive thing the second pass does — a full traversal of
// a body — and that pass visits every function once per walk name. The answer
// does not depend on which walk is being counted, so the caller memoises it on
// the *ast.FuncDecl and hands the result in. Seven walk names over the
// functions of this package is the difference between one traversal each and
// seven.
func boundNamesIn(fn *ast.FuncDecl) map[string]bool {
	bound := map[string]bool{}
	add := func(e ast.Expr) {
		if id, ok := e.(*ast.Ident); ok && id.Name != "_" {
			bound[id.Name] = true
		}
	}
	fields := func(list *ast.FieldList) {
		if list == nil {
			return
		}
		for _, f := range list.List {
			for _, n := range f.Names {
				add(n)
			}
		}
	}
	fields(fn.Recv)
	if fn.Type != nil {
		fields(fn.Type.Params)
		fields(fn.Type.Results)
	}
	if fn.Body == nil {
		return bound
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				for _, lhs := range node.Lhs {
					add(lhs)
				}
			}
		case *ast.RangeStmt:
			add(node.Key)
			add(node.Value)
		case *ast.ValueSpec:
			for _, name := range node.Names {
				add(name)
			}
		case *ast.TypeSpec:
			add(node.Name)
		case *ast.FuncLit:
			if node.Type != nil {
				fields(node.Type.Params)
				fields(node.Type.Results)
			}
		}
		return true
	})
	return bound
}

// packageLevelInts is a resolver for names this directory declares as an
// integer, built over the files the walk above has already read.
//
// # What it will settle, and what it will not
//
// A package-level `const` or `var` whose value is a non-negative integer
// LITERAL. That is the whole rule, and everything about it is deliberate:
//
//	const enumWorkers = 8            settled. The source says 8
//	var retries = 3                  settled. A var is as written down as a
//	                                 const for this purpose — nothing here
//	                                 runs, so what matters is that the number
//	                                 is in the file
//	const n = min(NumCPU, 8)         unknown. That is a bound on the value and
//	                                 not the value, and pricing a walk at its
//	                                 ceiling overstates a number the budget is
//	                                 made of
//	const n = other + 1              unknown. Arithmetic is the halting problem
//	                                 with extra steps, one expression in
//	iota                             unknown, because a ValueSpec in an iota
//	                                 block often has no value of its own and
//	                                 the ones that do are counted from a
//	                                 position rather than written
//
// # The cost, which is the reason it is lazy
//
// The walk above reads every .go file in this directory once and parses only
// the ones that name an enumeration entry point. A resolver that parsed them
// all to build a table would double that parse for a question asked, today,
// zero times — so a name is looked up when one is asked for, over the files
// whose BYTES contain it, using the same memoised parse. That is
// packageLevelNames's shape, and its argument: a file that does not contain
// the name cannot declare it, and the parse is what decides.
//
// Both answers are remembered, the absent one included, so a bound in a loop
// that is asked about once per call site is read once.
func packageLevelInts(names []string, sources map[string][]byte,
	parse func(string) *ast.File) func(string) (int, bool) {

	known := map[string]int{}
	absent := map[string]bool{}
	return func(name string) (int, bool) {
		if v, ok := known[name]; ok {
			return v, true
		}
		if absent[name] {
			return 0, false
		}
		for _, file := range names {
			if !bytes.Contains(sources[file], []byte(name)) {
				continue
			}
			tree := parse(file)
			if tree == nil {
				continue
			}
			for _, d := range tree.Decls {
				gen, ok := d.(*ast.GenDecl)
				if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR) {
					continue
				}
				for _, sp := range gen.Specs {
					vs, ok := sp.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, n := range vs.Names {
						if n.Name != name || i >= len(vs.Values) {
							continue
						}
						if v, ok := intLit(vs.Values[i]); ok {
							known[name] = v
							return v, true
						}
						// Declared here and not a written-down integer, which
						// is an answer: nothing further along will make it
						// one.
						absent[name] = true
						return 0, false
					}
				}
			}
		}
		absent[name] = true
		return 0, false
	}
}

// intLit is an expression's value when it is a non-negative integer literal.
func intLit(e ast.Expr) (int, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}
	v, err := strconv.Atoi(lit.Value)
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}

// exprText is an expression as a reader would say it, for a message.
//
// Only the spellings a range clause actually takes are rendered; anything else
// comes back as a placeholder, because a message saying "for range …" is still
// pointing at a line and a half-printed expression is a message that looks
// like a bug in the census.
func exprText(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.BasicLit:
		return x.Value
	case *ast.SelectorExpr:
		return exprText(x.X) + "." + x.Sel.Name
	case *ast.CallExpr:
		return exprText(x.Fun) + "(…)"
	case *ast.CompositeLit:
		return "a composite literal"
	}
	return "…"
}

// forHeaderText is a three-clause `for`'s header, as much of it as a message
// needs to point at the right line.
func forHeaderText(loop *ast.ForStmt) string {
	if loop.Cond == nil {
		return "{…}"
	}
	if cond, ok := loop.Cond.(*ast.BinaryExpr); ok {
		return fmt.Sprintf("…; %s %s %s; …", exprText(cond.X), cond.Op,
			exprText(cond.Y))
	}
	return "…; …; …"
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
	return listOf(keysOf(enumerationEntryPoints), func(name string) string {
		return name
	})
}

// repositoryWalkList is the decided walks, for a message.
func repositoryWalkList() string {
	return listOf(repositoryWalks, func(w repositoryWalkRow) string {
		return fmt.Sprintf("%s (%s)", w.fn, w.file)
	})
}

// walkList is what the scan found, for a message.
func walkList(found []repositoryWalk) string {
	return listOf(found, func(w repositoryWalk) string {
		runs := ""
		if w.runs != 1 {
			runs = fmt.Sprintf(", ×%d", w.runs)
		}
		return fmt.Sprintf("%s (%s:%d, %s%s)", w.fn, w.file, w.line, w.depth,
			runs)
	})
}
