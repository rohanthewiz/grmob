package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Everything this repository deliberately keeps two copies of, held to being
// two copies of the same thing.
//
// # The three shapes, and why they are in one walk
//
//	the timings record    five machine fields, a reporting arm and a standing
//	                      sentence about what a core count is worth, in
//	                      wasm/verify/timings_test.go and
//	                      internal/themehistory/timings_test.go
//	the import resolver   the five functions every census resolves a qualifier
//	                      with, in the importnames_test.go of both packages.
//	                      See importResolverShapes
//	the cores note        held to being COMPLETE rather than merely present:
//	                      every place in a record's package that reads the
//	                      core count is named in that package's note
//
// They are one arm because they are one repository-wide parse, and a fifth of
// those is the decision repositoryParseBudget exists to force. None of these
// questions needs a walk of its own — each is a reading of declarations the
// walk has already built — so taking one would be spending the budget on the
// arrangement of this file rather than on a question.
//
// # What this is about, which is a duplication that is currently correct
//
// wasm/verify/timings_test.go and internal/themehistory/timings_test.go hold
// the same struct twice: `machine` for a reader, `goos`/`goarch`/`goVersion`/
// `cores` for an arm, and then whatever timings that package has. Each carries
// the same reporting test, which says whether this run is standing on the
// machine the numbers came from.
//
// The second one says out loud why it is a copy: the two are separate
// `package main` programs, one under wasm/ and one under internal/, and an
// `internal/timingrecord` imported by both would be a package existing so that
// two structs could be one. That is a good reason. It is a good reason for
// TWO — at three it stops being an argument about a shared package's cost and
// becomes an argument for keeping a shape in step by remembering to, which is
// exactly the kind of thing this repository writes arms instead of.
//
// So the trigger is written down rather than left as a judgement somebody has
// to make again from scratch when a fourth package grows a wall clock in a
// comment. Two: the copy is the cheaper option, and this check holds the two
// to being the same shape. Three: the check fails, and says what the decision
// is.
//
// # Why the shape and not the values
//
// The values are DIFFERENT NUMBERS about different machines-at-different-
// moments and were never meant to match; each record's own arm holds it to
// being filled in. What has to stay in step is the five fields and the arm
// over them, because that is what makes a number in this repository
// attributable at all — a record with no `cores` in it is a record a reader on
// another machine cannot use.
//
// And what a `cores` is WORTH, which is the third thing held here and was the
// one place the two records had drifted apart: one of them carried a measured
// table of which term scales and the other said nothing at all. See
// timingsCoresNote — and checkCoresNoteNamesEveryScaledTerm, which is what
// makes that note a claim about this package rather than a paragraph.
func TestTheShapesThisRepositoryKeepsTwoCopiesOfAreInStep(t *testing.T) {
	root := filepath.Join("..", "..")
	_, considered, from, err := citingFiles(root)
	if err != nil {
		t.Fatalf("enumerating the repository (%s): %v", from, err)
	}
	paths := make([]string, 0, len(considered))
	for p := range considered {
		paths = append(paths, p)
	}
	// Sorted for the reason every other walk in this package sorts: findings
	// that arrive in map order cannot be diffed against the last run.
	sort.Strings(paths)

	// dir -> the records declared in it, and whether the reporting arm is
	// there. Keyed by directory rather than by file because the arm and the
	// record need not share one, and the thing that has to hold is a property
	// of the package.
	type found struct {
		name string
		rel  string
		line int
	}
	var records []found
	// The other two shapes, read off the same parse. Kept beside the record
	// rather than in walks of their own because a fifth repository-wide parse
	// is the decision repositoryParseBudget exists to force, and neither of
	// these needs one.
	var resolverDecls []importResolverDecl
	var asks []importPathAsk
	var coreSites []coreCountSite
	// dir -> what its cores note actually says, for the completeness check
	// below. The presence of the note is `notes`; this is its text.
	noteText := map[string]string{}
	arms := map[string]bool{}
	// dir -> whether the package says what its core count is worth. Kept
	// beside the arm because it is the same kind of thing: a property of the
	// package rather than of the file the record happens to sit in.
	notes := map[string]bool{}

	fset := token.NewFileSet()
	for _, rel := range paths {
		if !strings.HasSuffix(rel, ".go") {
			continue
		}
		// This file names the fields and the arm in its own prose and declares
		// neither. Skipped by name for the reason gitquoting_test.go skips
		// itself: a check that reads its own explanation as a finding is a
		// check nobody can leave a comment in.
		if rel == "wasm/verify/copies_test.go" {
			continue
		}
		src := filepath.Join(root, filepath.FromSlash(rel))
		file, parseErr := parser.ParseFile(fset, src, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			// A file go/parser cannot read is not this check's business — the
			// build says so first, and every other reading here would be
			// failing too.
			continue
		}
		dir := path.Dir(rel)
		// Which of the import-resolving shapes this file declares, and which
		// import paths its censuses name. One walk over the declarations, two
		// questions — see importResolverDeclarationsIn.
		fileDecls, fileAsks := importResolverDeclarationsIn(fset, rel, file)
		resolverDecls = append(resolverDecls, fileDecls...)
		asks = append(asks, fileAsks...)
		// And every place this file reads the core count, which is what makes
		// a cores note checkable rather than merely present.
		coreSites = append(coreSites, coreCountSitesIn(t, fset, rel, file)...)
		for _, d := range file.Decls {
			switch decl := d.(type) {
			case *ast.FuncDecl:
				if decl.Recv == nil && decl.Name.Name == timingsArm {
					arms[dir] = true
				}
			case *ast.GenDecl:
				// The cores note, which may be a const or a var — what
				// matters is that the package declares one, not how.
				for _, sp := range decl.Specs {
					vs, ok := sp.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, n := range vs.Names {
						if n.Name != timingsCoresNote {
							continue
						}
						notes[dir] = true
						if i < len(vs.Values) {
							noteText[dir] = stringLiteralValue(vs.Values[i])
						}
					}
				}
				if decl.Tok != token.VAR {
					continue
				}
				for _, sp := range decl.Specs {
					vs, ok := sp.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, n := range vs.Names {
						if !strings.HasSuffix(n.Name, "TimingsTakenOn") {
							continue
						}
						at := fset.Position(n.Pos())
						records = append(records, found{name: n.Name, rel: rel,
							line: at.Line})
						if i < len(vs.Values) {
							checkRecordShape(t, rel, at.Line, n.Name, vs.Values[i])
						}
					}
				}
			}
		}
	}

	// The other two shapes, before the record's own checks: each is a whole
	// question with its own reaching-anything arm, and a record that has gone
	// missing should not take them with it.
	checkImportResolverCopies(t, resolverDecls)
	checkImportPathsAreImportable(t, root, asks)

	// The walk reaching anything. Every arm in this package that walks the
	// repository says this, and for the same reason: a walk over nothing
	// passes silently and reads as a clean result — and this one is looking
	// for a NAME, which a rename would take away without touching a line of
	// the thing it names.
	if len(records) == 0 {
		t.Fatalf("no `…TimingsTakenOn` record was found in %d file(s) "+
			"enumerated by %s, and this repository has two: "+
			"wasm/verify/timings_test.go and "+
			"internal/themehistory/timings_test.go. Either the walk is not "+
			"reaching them or they have been renamed, and in both cases this "+
			"check is over nothing.", len(paths), from)
	}

	// The arm beside each record. A record with no reporting test is a struct
	// nothing reads: the numbers are attributed in the source and nothing puts
	// the attribution in front of the person holding a different reading.
	seen := map[string]bool{}
	for _, r := range records {
		dir := path.Dir(r.rel)
		seen[dir] = true
		if !arms[dir] {
			t.Errorf("%s:%d declares %s and %s declares no `func %s`.\n\n"+
				"A timings record with no reporting arm beside it is a struct "+
				"nothing reads. The record is how a number in a comment is "+
				"attributed; the arm is what puts that attribution in front of "+
				"the person holding a reading that disagrees with it, which is "+
				"the only moment either one is worth anything.",
				r.rel, r.line, r.name, dir, timingsArm)
		}
		if !notes[dir] {
			t.Errorf("%s:%d declares %s and %s declares no `%s`.\n\n"+
				"`cores` is the one machine field that changes a recorded "+
				"number by a term a reader can NAME, and the arm above reports "+
				"it as `8 cores against 4` — which says that the two "+
				"computers differ and not what the difference is worth. "+
				"Whichever the answer is, it has to be written down: a record "+
				"whose figures do not move with the core count is as useful to "+
				"a reader as one whose figures do, and silence reads as "+
				"\"nobody measured that\" rather than as \"that is not where "+
				"the difference is\".\n\n"+
				"This was two records disagreeing about it rather than a "+
				"missing feature: one carried a measured table and the other "+
				"said nothing, so a reader holding both had one attribution "+
				"and one gap. Declare a `%s` saying which of this package's "+
				"terms scale and by how much — or that none of them do, and "+
				"why — and print it from %s when the two counts differ.",
				r.rel, r.line, r.name, dir, timingsCoresNote,
				timingsCoresNote, timingsArm)
		}
	}

	// And what each of those notes actually claims, against what its package
	// does. `seen` is the set of directories that hold a record, which is the
	// set the notes are about.
	checkCoresNoteNamesEveryScaledTerm(t, seen, coreSites, noteText)

	names := make([]string, 0, len(records))
	for _, r := range records {
		names = append(names, fmt.Sprintf("%s (%s:%d)", r.name, r.rel, r.line))
	}
	sort.Strings(names)

	// The trigger. See the header: two is a copy with a written reason, three
	// is a shape being kept in step by memory.
	if len(records) > timingsRecordCopies {
		t.Errorf("this repository has %d timings record(s) and the argument "+
			"for copying the struct was written for %d: %s.\n\n"+
			"internal/themehistory/timings_test.go says why it is a copy — two "+
			"separate `package main` programs, and an `internal/timingrecord` "+
			"imported by both would be a package existing so that two structs "+
			"could be one. That reasoning is about the cost of a package "+
			"against the cost of one duplicate. At %d it is no longer that "+
			"trade: the five machine fields and the reporting arm are being "+
			"kept in step across %d files by whoever remembers to, and this "+
			"check is the memory.\n\n"+
			"Extract the record and the arm into a package both can import, or "+
			"— if there is a reason the third is different — raise "+
			"timingsRecordCopies and write that reason down where the copy "+
			"argument is, so the next person reads a decision rather than a "+
			"number.",
			len(records), timingsRecordCopies, strings.Join(names, ", "),
			len(records), len(records))
	}

	t.Logf("%d timings record(s), each with the %d machine field(s), a `%s` "+
		"and a `%s` in its package: %s. Enumerated by %s.",
		len(records), len(timingsMachineFields), timingsArm, timingsCoresNote,
		strings.Join(names, ", "), from)
}

// How many copies of the record the written argument covers.
//
// Not a magic number and not a limit for its own sake: it is the number the
// reasoning in internal/themehistory/timings_test.go was written about. Moving
// it is a decision, which is why the failure above asks for the reason to be
// written beside that reasoning rather than just for the constant to go up.
//
// # The direction this fails in, which is chosen rather than overlooked
//
// A third timings record is a package that has grown a wall clock and written
// down which machine it came from — the right thing to do — and this check
// fails it. That is a CORRECT addition failing an arm, which is the shape
// hookSchemaReadFrom exists to apologise for: a key a later Claude Code adds
// is a key the tables have never heard of, so a correct settings.json fails a
// check meant to catch an incorrect one.
//
// The two are the same direction and they are not the same decision, because
// what a reader can do about them is not the same:
//
//	the hook tables    the failure is a MISTAKE about another program's
//	                   schema. Nothing here can tell it from a release that
//	                   moved, and softening the check would let `if` sit in
//	                   settings.json again — so the finding stays and arrives
//	                   with the two versions, which is all this side can
//	                   honestly offer
//	this constant      the failure IS the prompt. The third record is the
//	                   moment the trade changes, the change wanted is in THIS
//	                   repository, and the message asks for an extraction or a
//	                   written reason
//
// So the failing green run is the mechanism here and the residue there. The
// cost is the same in both: somebody reads a message and edits a file. The
// difference is that this one is asking for a decision it can name, and that
// is worth stating rather than leaving to be rediscovered as an inconsistency
// between two arms in the same repository.
const timingsRecordCopies = 2

// The reporting arm's name, which is the same in both packages and is what
// makes the two a pair rather than two structs that happen to look alike.
const timingsArm = "TestTheTimingsInThisPackageSayWhichMachineTheyCameFrom"

// The name of the standing sentence about what a core count is worth here.
//
// # Why this is part of the shape and the other four fields are not
//
// The record's five fields are what makes a number ATTRIBUTABLE. Four of them
// are answered by runtime and differ or do not; `cores` is the only one that
// moves a recorded figure by an amount somebody could subtract, and it is
// therefore the only one where "these two machines differ" is an unfinished
// sentence.
//
// Both packages have an answer and they are different answers —
// internal/themehistory's largest term is 88 git processes and improves to
// eight workers; wasm/verify's is one program's own goroutines and is flat
// from two cores up. Neither is guessable from the other, which is exactly why
// having one and not the other was worse than having neither: a reader
// comparing the two records found an attribution beside a silence and no way
// to tell a package that does not vary from one nobody measured.
const timingsCoresNote = "coresAttribution"

// The fields every record has, and the type each one is.
//
// The four machine fields are read by the arm — runtime answers all of them —
// and `machine` is the one for a person, which nothing checks and which is the
// only one that says what the computer actually was. The per-package timings
// are deliberately NOT listed: they are different numbers about different
// things, and a list of them here would be this check having an opinion about
// what a package is allowed to measure.
var timingsMachineFields = map[string]string{
	"machine":   "string",
	"goos":      "string",
	"goarch":    "string",
	"goVersion": "string",
	"cores":     "int",
}

// checkRecordShape holds one record's declaration to carrying the five machine
// fields, at the right types.
//
// The value is an anonymous struct literal in both records, which is what
// makes the shape readable from a parse: the fields are right there in the
// composite literal's type. A record that became a NAMED type elsewhere would
// fail this, and the message says so rather than reporting a missing field —
// the two are different findings and only one of them is a mistake.
func checkRecordShape(t *testing.T, rel string, line int, name string, val ast.Expr) {
	t.Helper()
	lit, ok := val.(*ast.CompositeLit)
	if !ok {
		t.Errorf("%s:%d declares %s and its value is %T, not a struct "+
			"literal. This check reads the five machine fields out of the "+
			"literal's own type; a record built some other way needs this "+
			"check taught how to read it.", rel, line, name, val)
		return
	}
	st, ok := lit.Type.(*ast.StructType)
	if !ok {
		t.Errorf("%s:%d declares %s as a %T rather than an anonymous struct.\n\n"+
			"If the record has been given a named type — which is what "+
			"sharing one between packages looks like — this check has been "+
			"overtaken by the change it exists to prompt, and should be "+
			"rewritten against that type instead of deleted.",
			rel, line, name, lit.Type)
		return
	}

	got := map[string]string{}
	for _, f := range st.Fields.List {
		typ := types(f.Type)
		for _, n := range f.Names {
			got[n.Name] = typ
		}
	}
	for _, field := range keysOf(timingsMachineFields) {
		want := timingsMachineFields[field]
		have, present := got[field]
		if !present {
			t.Errorf("%s:%d declares %s and it has no `%s %s` field.\n\n"+
				"The five machine fields are what makes a number in this "+
				"package's prose attributable: a reading a few percent off one "+
				"of them is a difference between two computers before it is "+
				"anything else, and a record short a field cannot say which "+
				"difference. The other record is the reference — see %s.",
				rel, line, name, field, want, timingsArm)
			continue
		}
		if have != want {
			t.Errorf("%s:%d declares %s with `%s %s` and the other record has "+
				"`%s %s`. The arm compares this against what runtime answers, "+
				"so the type is part of the comparison rather than a detail of "+
				"the declaration.", rel, line, name, field, have, field, want)
		}
	}
}

// types is an expression's type as it is written, for the field comparison
// above.
//
// Only the shapes a record's fields actually take are spelled out, which is
// identifiers and nothing else. Anything else comes back as a description
// rather than as a type name, so the message says what it found instead of
// comparing "" against "string" and reporting a field that is not there.
func types(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return fmt.Sprintf("%T", e)
}

// coreCountSite is one place a package reads how many cores it is running on.
type coreCountSite struct {
	dir, rel string
	line     int
	// The declaration it is in — the function, or the variable whose
	// initialiser it is. This is the name the note has to contain, because it
	// is the only handle a reader has on the term.
	in string
	// How it is spelled, for the message: `runtime.NumCPU` or
	// `runtime.GOMAXPROCS`.
	how string
}

// checkCoresNoteNamesEveryScaledTerm holds each package's cores note to naming
// every term in that package that scales with the core count.
//
// # What was wrong with holding the note to EXISTING
//
// The record check above makes each package declare a `coresAttribution`, and
// both of them contain a measured table today. Nothing re-derived either one.
// A note saying "nothing in this package moves with the core count", left
// standing after somebody adds a worker pool, passes exactly as well as one
// that is right — which is the state the record itself was in before it had an
// arm, arriving one level along in the thing the arm asks for.
//
// # What a parse can hold it to, and what it cannot
//
// Not the numbers. `0.22s at eight workers` is a wall clock and an assertion
// over one fails on a busy laptop, which is the reason both records exist
// instead of being tests. What a parse CAN settle is the INVENTORY: which
// declarations in this package read the core count, which is exactly the set
// of terms a note about core counts is a claim about.
//
// So the rule is that every such declaration is NAMED in the note. That is a
// finding a reader can act on in one edit — the term is there in front of them
// — and it fires on the change that makes a note wrong, which is a new term
// arriving rather than an old measurement drifting.
//
// The measurement drifting is still not covered and cannot be from here. What
// changes is that the note now goes stale LOUDLY in the one way that is
// somebody's fault, and the figures are what they always were: a reading of a
// machine, attributed.
//
// # Why the reporting arm is not one of these
//
// It reads runtime.NumCPU to compare this machine against the record, which is
// the arm doing its job rather than a term that scales — so a site inside the
// declaration named by timingsArm is skipped. Every other reading is work
// being divided, or a decision made on the core count, and both are things a
// note about core counts has to have an opinion about.
func checkCoresNoteNamesEveryScaledTerm(t *testing.T, recordDirs map[string]bool,
	sites []coreCountSite, notes map[string]string) {

	t.Helper()
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].rel != sites[j].rel {
			return sites[i].rel < sites[j].rel
		}
		return sites[i].line < sites[j].line
	})
	named := map[string][]string{}
	for _, site := range sites {
		if !recordDirs[site.dir] {
			// A package with no timings record has no note to be complete,
			// and nothing here is asking every package in the repository to
			// grow one.
			continue
		}
		note, ok := notes[site.dir]
		if !ok || note == "" {
			// The record check above already says this package has no note,
			// or has one this cannot read. Saying it a second time per site
			// is the wall.
			continue
		}
		if strings.Contains(note, site.in) {
			named[site.dir] = append(named[site.dir], site.in)
			continue
		}
		t.Errorf("%s:%d reads the core count in %s (`%s`), and %s's `%s` "+
			"does not mention %s.\n\n"+
			"That note is what a reader on a different machine is handed when "+
			"the arm reports `8 cores against 4`, and it is a claim about "+
			"WHICH of this package's terms move with the count. A term the "+
			"note has never heard of is the note being wrong in the direction "+
			"nobody can see: the figures still read as attributed and the "+
			"sentence under them is about a package that no longer exists.\n\n"+
			"The numbers in it cannot be checked from here — a wall clock is "+
			"a reading of a machine, which is why these are records and not "+
			"assertions — but the inventory can, and this is the half that "+
			"goes stale by somebody adding code rather than by a measurement "+
			"drifting.\n\n"+
			"Measure what %s is worth at one core and at this machine's "+
			"count, and say so in `%s`; or say that it does not move and why, "+
			"which is as useful and is also an answer.",
			site.rel, site.line, site.in, site.how, site.dir,
			timingsCoresNote, site.in, site.in, timingsCoresNote)
	}
	dirs := make([]string, 0, len(named))
	for dir := range named {
		dirs = append(dirs, dir)
		sort.Strings(named[dir])
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		t.Logf("%s's `%s` names all %d of its core-scaled term(s): %s.", dir,
			timingsCoresNote, len(named[dir]), strings.Join(named[dir], ", "))
	}
}

// coreCountSitesIn is every place this file reads the core count, and what
// declaration each reading is in.
//
// `runtime` is resolved off the file's own import block for the reason
// importnames_test.go gives — an `import rt "runtime"` and an `rt.NumCPU()` is
// a term this check would otherwise not see, which is the silent direction.
//
// A package-level variable counts as much as a function does: enumWorkers is
// `min(runtime.NumCPU(), 8)` and is the largest single term in one of the two
// records. What is wanted is the name a note would have to use, and for a
// `var` that is the variable.
func coreCountSitesIn(t *testing.T, fset *token.FileSet, rel string,
	file *ast.File) []coreCountSite {

	t.Helper()
	runtimeNames := qualifiersFor(t, rel, file, "runtime")
	if len(runtimeNames) == 0 {
		return nil
	}
	dir := path.Dir(rel)
	read := func(in string, n ast.Node) []coreCountSite {
		var out []coreCountSite
		ast.Inspect(n, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || !runtimeNames[pkg.Name] {
				return true
			}
			switch sel.Sel.Name {
			case "NumCPU", "GOMAXPROCS":
				out = append(out, coreCountSite{dir: dir, rel: rel,
					line: fset.Position(call.Pos()).Line, in: in,
					how: pkg.Name + "." + sel.Sel.Name})
			}
			return true
		})
		return out
	}
	var sites []coreCountSite
	for _, d := range file.Decls {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if decl.Body == nil {
				continue
			}
			// The reporting arm's own comparison is not a term — see the
			// header above.
			if decl.Recv == nil && decl.Name.Name == timingsArm {
				continue
			}
			sites = append(sites, read(decl.Name.Name, decl.Body)...)
		case *ast.GenDecl:
			for _, sp := range decl.Specs {
				vs, ok := sp.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, n := range vs.Names {
					if i < len(vs.Values) {
						sites = append(sites, read(n.Name, vs.Values[i])...)
					}
				}
			}
		}
	}
	return sites
}

// stringLiteralValue is the value of a string constant written as literals and
// `+`, or "" for anything else.
//
// Both cores notes are one long sentence built by concatenation, which is what
// a string constant that has to fit in a column of source looks like. Nothing
// here evaluates anything: a note assembled by a function is a note this
// returns "" for, and the caller reads that as "no text to check against"
// rather than as an empty note.
func stringLiteralValue(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.BasicLit:
		if x.Kind != token.STRING {
			return ""
		}
		if v, err := strconv.Unquote(x.Value); err == nil {
			return v
		}
		return x.Value
	case *ast.BinaryExpr:
		if x.Op != token.ADD {
			return ""
		}
		left, right := stringLiteralValue(x.X), stringLiteralValue(x.Y)
		if left == "" || right == "" {
			return ""
		}
		return left + right
	case *ast.ParenExpr:
		return stringLiteralValue(x.X)
	}
	return ""
}
