package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Everything this repository deliberately keeps two copies of, held to being
// two copies of the same thing.
//
// # The four shapes, and why they are in one walk
//
//	the timings record    five machine fields, a reporting arm and a standing
//	                      sentence about what a core count is worth, in
//	                      wasm/verify/timings_test.go and
//	                      internal/themehistory/timings_test.go
//	the import resolver   the functions every census resolves a qualifier
//	                      with, and the package-level state they keep, in the
//	                      importnames_test.go of both packages. See
//	                      importResolverShapes and importResolverStateShapes
//	the cores note        held to being COMPLETE rather than merely present,
//	                      in both directions: every place in a record's
//	                      package that reads the core count is named in that
//	                      package's note, and every term the note names is
//	                      still something that package has
//	the file counts       every sentence in this repository that prices
//	                      something against how many Go files the tree holds,
//	                      held to the count this walk has just made. The
//	                      second copy of that number is the TREE — the one
//	                      shape here whose other copy is not source. See
//	                      checkProseFileCounts
//
// They are one arm because they are one repository-wide parse, and a fifth of
// those is the decision repositoryParseBudget exists to force. None of these
// questions needs a walk of its own — each is a reading of declarations the
// walk has already built — so taking one would be spending the budget on the
// arrangement of this file rather than on a question.
//
// # Why they are four subtests and not four sections
//
// One walk is what the budget requires; one FUNCTION was what the first
// arrangement made of it, and those are not the same thing. Each of these
// questions ends in a reaching-anything arm, and every one of those is a
// t.Fatalf — a walk over nothing passes silently and reads as a clean result,
// so the only honest thing to do about it is to stop. A Fatalf stops the
// GOROUTINE, so three questions in one function is three questions one of
// which can end the other two: a repository where the records had been renamed
// reported that and said nothing about whether the import resolver's two
// copies were still in step.
//
// t.Run gives each question its own goroutine and its own failure boundary at
// no cost to the walk — the parse has already happened and the subtests read
// what it built. So the unit of the WALK is the repository and the unit of a
// FINDING is the question, which is what it was before these were folded
// together.
//
// The order is the cheap-to-say first: the two that read declarations the walk
// collected, then the file counts, which are one integer comparison, then the
// records, whose own checks are the longest.
//
// # Why a figure in prose is a copy, which is the fourth shape
//
// A sentence that says "386 tracked Go files" is holding a number that lives
// somewhere else — the tree — and nothing joins the two. That is this file's
// subject exactly: a copy kept in step by somebody remembering to. It had
// already drifted by four, across five sentences, in the package whose whole
// argument is that a number nobody can attribute is worth nothing, and what
// found it was a reader who had not written it rather than any of the eleven
// verification paths.
//
// It is an ARM and not a report, which the wall clocks in the same sentences
// are not allowed to be. The difference is what the number is a fact about: a
// timing is a fact about one computer, so asserting it would fail on every
// machine that is not that one, and a file count is a fact about the TREE, so
// a count that disagrees here disagrees for everybody. See
// verifyTimingsTakenOn for the other half of that argument.
//
// The cost is that adding a Go file to this repository now fails a test until
// those sentences are edited. That is the point and not a side effect — the
// alternative is the five sentences nobody could trust, which is what they
// were — and the message names every one of them with its line so the fix is
// one pass rather than a search.
//
// Two limits, both stated rather than resolved. This walk skips
// copies_test.go by name, so a figure written HERE is not held — which is
// load-bearing rather than incidental, because the paragraph above quotes one
// as an example and would otherwise be a finding about itself. It is the same
// trade the skip already makes, and this file prices nothing against the
// count. And what is held is the FORM — see trackedGoFileFigure — not the
// practice: a sentence that quotes the count some other way is not a finding,
// it is a sentence this cannot read.
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
// timingsCoresNote, checkCoresNoteNamesEveryScaledTerm and
// checkCoresNoteNamesNothingThatIsGone, which between them make that note a
// claim about this package rather than a paragraph.
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

	var records []timingsRecord
	// The other two shapes, read off the same parse. Kept beside the record
	// rather than in walks of their own because a fifth repository-wide parse
	// is the decision repositoryParseBudget exists to force, and neither of
	// these needs one.
	var resolverDecls []importResolverDecl
	var asks []importPathAsk
	var coreSites []coreCountSite
	// Every sentence in the tree that quotes the tracked-Go-file count, and
	// the count itself. Both are readings of this same walk: the figures come
	// off the parse below and the number comes off the enumeration above, so
	// the fourth question costs the walk nothing it was not already paying.
	var figures []proseFigure
	goFiles := 0
	// dir -> what its cores note says and what it names. The PRESENCE of the
	// note is `notes`, which is what the record check holds each package to;
	// this is the note itself, which is what the two checks over its content
	// read. See coresNote.
	noteText := map[string]coresNote{}
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
		// Counted BEFORE the skip below, because what the prose figures claim
		// is the size of the tree and copies_test.go is a file in it. Counted
		// before the parse too: a file go/parser cannot read is still a Go
		// file git is tracking.
		goFiles++
		// This file names the fields and the arm in its own prose and declares
		// neither. Skipped by name for the reason gitquoting_test.go skips
		// itself: a check that reads its own explanation as a finding is a
		// check nobody can leave a comment in.
		if rel == "wasm/verify/copies_test.go" {
			continue
		}
		src := filepath.Join(root, filepath.FromSlash(rel))
		// Read here rather than left to go/parser so that the bytes can be
		// used twice: once as the source of the parse, and once as the filter
		// that decides whether this file's prose is worth walking at all.
		// ParseFile reads the file itself when handed nil, so this is the
		// same read moved rather than a second one.
		raw, readErr := os.ReadFile(src)
		if readErr != nil {
			// Same reasoning as the parse failure below: git listed it, the
			// stat in citingFiles found it, and a file that has gone away
			// between then and now is not this check's business.
			continue
		}
		// ParseComments because one of the four questions here is about the
		// PROSE and go/parser throws comments away unless asked. Measured
		// over this walk's own 386 files on the machine verifyTimingsTakenOn
		// names, five runs apiece: 0.040s without the comments and 0.044s
		// with them.
		file, parseErr := parser.ParseFile(fset, src, raw,
			parser.SkipObjectResolution|parser.ParseComments)
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
		// And every sentence in it that quotes the tracked-Go-file count,
		// read off the comments and string constants the parse just built —
		// which is why this is here and not in a scan of its own.
		//
		// Filtered by a byte scan first, which is the same move the walk
		// census makes one file over and cannot produce a false negative: a
		// figure in the held form contains this word, in a comment or in a
		// string, whatever else the file says. It matters because the scan
		// below walks EVERY node of a syntax tree and builds a run of text
		// out of every comment group and every string constant in it, and 11
		// of these 386 files contain the word at all. Measured: 0.09s over
		// all of them, 0.00s with the filter, which is 3% of this package's
		// wall clock for a question with five answers in it.
		if bytes.Contains(raw, []byte("tracked")) {
			figures = append(figures, fileCountFiguresIn(fset, rel, file)...)
		}
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
							text := stringLiteralValue(vs.Values[i])
							noteText[dir] = coresNote{
								text:  text,
								terms: termsNamedIn(text),
							}
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
						rec := timingsRecord{name: n.Name, rel: rel,
							line: at.Line}
						// The literal is kept rather than read here: the
						// shape check belongs to the records question, which
						// is a subtest of its own — see the header.
						if i < len(vs.Values) {
							rec.val = vs.Values[i]
						}
						records = append(records, rec)
					}
				}
			}
		}
	}

	// One walk, four questions, four failure boundaries. See the header for
	// why this is t.Run and not four sections of one function: each of these
	// ends in a t.Fatalf over a walk that reached nothing, and a Fatalf ends
	// the goroutine it is on.
	t.Run("the import-resolving helpers", func(t *testing.T) {
		checkImportResolverCopies(t, resolverDecls)
	})
	t.Run("the paths those helpers are asked about", func(t *testing.T) {
		checkImportPathsAreImportable(t, root, asks)
	})
	t.Run("the file counts in this repository's prose", func(t *testing.T) {
		checkProseFileCounts(t, from, goFiles, figures)
	})
	t.Run("the timings records", func(t *testing.T) {
		checkTimingsRecordCopies(t, root, from, len(paths), records, arms,
			notes, coreSites, noteText)
	})
}

// checkTimingsRecordCopies is the timings-record question: every record
// carrying the five machine fields, an arm and a note beside it, the note
// being complete, and there being no more copies than the written argument
// covers.
//
// # Why this is a function and not the tail of the walk
//
// It is one of three questions the walk above answers, and the only one whose
// reaching-anything arm can silence the others: `records` being empty is a
// t.Fatalf, which ends the goroutine, and a repository where the records had
// been renamed used to report that and say nothing at all about whether the
// import resolver's two copies were still in step. Splitting the questions
// into subtests is what fixes that; splitting this one out into a function of
// its own is what keeps the walk readable now that its result is read in three
// places.
//
// Everything here is a reading of what the walk collected. Nothing in it
// parses, reads or enumerates anything.
func checkTimingsRecordCopies(t *testing.T, root, from string, filesSeen int,
	records []timingsRecord, arms, notes map[string]bool,
	coreSites []coreCountSite, noteText map[string]coresNote) {

	t.Helper()
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
			"check is over nothing.", filesSeen, from)
	}

	// The shape of each record's literal, which is the one question here that
	// is about a RECORD. Read here rather than in the walk so that every
	// reading of a record is in one place — and so that a record declared with
	// no value is a record this says nothing about, which is what the
	// `var x T` case is.
	seen := map[string]bool{}
	byDir := map[string][]timingsRecord{}
	for _, r := range records {
		dir := path.Dir(r.rel)
		seen[dir] = true
		byDir[dir] = append(byDir[dir], r)
		if r.val != nil {
			checkRecordShape(t, r.rel, r.line, r.name, r.val)
		}
	}

	// # Everything else here is about a PACKAGE, and is counted per package
	//
	// The arm and the note are declarations of a directory, not of a record:
	// `arms` and `notes` are keyed by directory, and a second record in the
	// same package does not need a second arm. Asking these once per record
	// meant a package with two records reported each missing thing twice —
	// two findings, one fact, and one edit that answers both — which is the
	// wall this repository writes one-per-row rules against everywhere else.
	//
	// Nothing forbids a second record in one package below
	// timingsRecordCopies, so this is a shape that is reachable rather than a
	// hypothetical. The finding names every record in the directory, because
	// what a reader wants is which numbers are unattributed, and that is all
	// of them.
	for _, dir := range keysOf(seen) {
		here := recordList(byDir[dir])
		if !arms[dir] {
			t.Errorf("%s declares %s and no `func %s`.\n\n"+
				"A timings record with no reporting arm beside it is a struct "+
				"nothing reads. The record is how a number in a comment is "+
				"attributed; the arm is what puts that attribution in front of "+
				"the person holding a reading that disagrees with it, which is "+
				"the only moment either one is worth anything.",
				dir, here, timingsArm)
		}
		if !notes[dir] {
			t.Errorf("%s declares %s and no `%s`.\n\n"+
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
				dir, here, timingsCoresNote, timingsCoresNote, timingsArm)
			continue
		}
		// And the note being something the two checks below can READ. That
		// check holds each package to declaring a `coresAttribution`; this
		// holds the declaration to being evaluable without running the
		// package.
		//
		// A note assembled by a function, or built from anything
		// stringLiteralValue does not evaluate, is declared and unreadable —
		// it passes the check above, comes back as "", and both directions
		// then skip it in silence. That is a note claiming an attribution
		// with nothing whatever holding it, which is the state the note itself
		// was in two sessions ago.
		if noteText[dir].text != "" {
			continue
		}
		t.Errorf("%s declares a `%s` and this check could not read its "+
			"text.\n\n"+
			"Both things held over that note read the note: one asks whether "+
			"it names every term in this package that scales with the core "+
			"count, and the other whether everything it names still exists. A "+
			"note whose text cannot be read is exempt from both, silently, "+
			"and goes on reading to a person as a measured attribution — "+
			"which is exactly the state a note with no arm over it was in.\n\n"+
			"stringLiteralValue reads a string constant written as literals "+
			"joined by `+`, which is what a sentence that has to fit in a "+
			"column of source looks like. A note assembled by a function, or "+
			"built out of other constants, is one this cannot evaluate "+
			"without running the package. Write it as literals, or teach "+
			"stringLiteralValue the shape and say what it now costs.",
			dir, timingsCoresNote)
	}

	// Two directions, because a note can be wrong in two ways: it can fail to
	// name a term that exists, and it can name one that does not. Both read
	// the same coresNote, so there is one answer to "what does this note name"
	// rather than two parameters that could disagree.
	checkCoresNoteNamesEveryScaledTerm(t, seen, coreSites, noteText)
	checkCoresNoteNamesNothingThatIsGone(t, root, seen, noteText)

	// The same rendering the per-package findings above use, over every record
	// rather than one package's — see recordList.
	all := recordList(records)

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
			len(records), timingsRecordCopies, all, len(records), len(records))
	}

	t.Logf("%d timings record(s), each with the %d machine field(s), a `%s` "+
		"and a `%s` in its package: %s. Enumerated by %s.",
		len(records), len(timingsMachineFields), timingsArm, timingsCoresNote,
		all, from)
}

// recordList is the records one package declares, for a message.
//
// Named with their file and line because the finding is about the package and
// the edit is in a file: a reader told "internal/foo declares no arm" needs to
// know which numbers are the ones going unattributed.
func recordList(in []timingsRecord) string {
	return listOf(in, func(r timingsRecord) string {
		return fmt.Sprintf("%s (%s:%d)", r.name, r.rel, r.line)
	})
}

// timingsRecord is one `…TimingsTakenOn` declaration the walk found.
//
// The declaration's VALUE travels with it rather than being read where it was
// found. The walk's job is to collect; the three questions are answered
// afterwards, each in its own subtest, and a shape check made during the walk
// would be a finding belonging to one question raised on another's goroutine.
//
// `val` is nil for a record declared without one — `var x T` — which is a
// declaration this check has no opinion about: the five machine fields are
// read out of the composite literal's own type, and there is no literal there
// to read.
type timingsRecord struct {
	name string
	rel  string
	line int
	val  ast.Expr
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

// coresNote is one package's standing sentence about what its core count is
// worth, as the two checks over it need it.
//
// # Why one value and not a text map beside a terms map
//
// They were two parameters carrying one fact, which is how a caller comes to
// pass a stale one: the text is what says whether the note could be READ at
// all, and the terms are what it names, and the terms are derived from the
// text. Deriving them once, where the text is found, means there is no
// arrangement in which the two disagree.
//
// `text` is "" for a note this could not evaluate — see stringLiteralValue,
// and the arm in checkTimingsRecordCopies that reports it rather than letting
// both directions skip in silence.
type coresNote struct {
	text string
	// The identifiers the note spells as code — see termsNamedIn. Empty for a
	// note that names none, which is a legitimate note for a package with
	// nothing to name.
	terms []string
}

// names is the note's terms as a set, for the membership test the forward
// check makes once per core-count site.
func (n coresNote) names() map[string]bool {
	set := make(map[string]bool, len(n.terms))
	for _, term := range n.terms {
		set[term] = true
	}
	return set
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
	sites []coreCountSite, notes map[string]coresNote) {

	t.Helper()
	// What each note names, as a set. This used to be a strings.Contains over
	// the note's whole text, which is loose in a way nothing would have
	// noticed: a note mentioning `enumWorkersPool` satisfied a check asking
	// about `enumWorkers`, because one name is a prefix of the other. A term
	// is now a term because the note spelled it as code — see termsNamedIn —
	// and the membership is exact.
	namesAsCode := map[string]map[string]bool{}
	for dir, note := range notes {
		namesAsCode[dir] = note.names()
	}
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
		if !ok || note.text == "" {
			// A package with no note, or with one nothing could read. Both
			// are said once by the arms in checkTimingsRecordCopies — the
			// record check for the first and the readability check for the
			// second — and saying either a second time per core site is the
			// wall this repository writes one-per-row rules against.
			continue
		}
		if namesAsCode[site.dir][site.in] {
			named[site.dir] = append(named[site.dir], site.in)
			continue
		}
		t.Errorf("%s:%d reads the core count in %s (`%s`), and %s's `%s` "+
			"does not name %s as code.\n\n"+
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
			"A term counts as named when the note spells it inside "+
			"backquotes, which is the note saying it means a declaration "+
			"rather than a word — the same rule the other direction reads, so "+
			"a name mentioned in prose is neither claimed nor checked.\n\n"+
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

// checkCoresNoteNamesNothingThatIsGone holds each note to the other
// direction: every term it names still being something its package has.
//
// # The half the completeness check could not see
//
// checkCoresNoteNamesEveryScaledTerm reads the note for a substring, once per
// core-count site. That makes a note that has never heard of a new term a
// finding, and it says nothing whatever about a term the note names that no
// longer exists — the check only ever asks in the direction of the code. A
// function deleted leaves its name in the sentence, the sentence still reads
// as a measured attribution, and nothing anywhere says otherwise. It is the
// same silence one step round from the one the completeness check closed.
//
// # Which words in a paragraph are a claim about code
//
// The note is prose, so something has to decide what in it is meant as a name,
// and the author is the only one who can. What is read as a term is what the
// note spells inside BACKQUOTES — see termsNamedIn. That is a convention both
// notes already half-followed and it is now the whole rule, which is what
// makes checkCoresNoteNamesEveryScaledTerm exact as well: a term is named when
// it is named AS CODE, and not when its letters happen to appear in a word.
//
// The rule this replaced was camelCase — an identifier-shaped word with a
// lowercase letter somewhere before an uppercase one — chosen because it is
// what this repository's declarations look like and what English words never
// do. It was right about most of a note and wrong at both ends:
//
//	workers                  a real local the note quotes, invisible to the
//	                         rule, because nothing in a lowercase word tells a
//	                         variable from a noun
//	a prose word in camel    reported as a term that has gone, about a
//	                         sentence that was never a claim about code
//
// Backquotes have neither end. They also cost nothing to comply with: a note
// that spells its terms as code is a note that reads better, and the arm's
// finding when one is missed says so in one line rather than asking the author
// to guess at a spelling rule.
//
// # What counts as the term still existing
//
// Any identifier anywhere in the package — a declaration, a field, a use.
// Deliberately the widest reading: the finding this exists for is a name that
// has gone from the package ENTIRELY, and asking for a declaration in
// particular would report a note that names a field of a struct declared
// elsewhere, or a method, as though the term had been deleted.
//
// The scan is packageLevelNames's shape and for the same reasons — read the
// directory, scan the bytes, parse the hits, and let the PARSE decide, so that
// a name appearing only inside the note's own string literal is read and
// discarded rather than counted. A substring search over the source would
// match every note against itself and find nothing, ever.
//
// # The cross-reference, which is not a finding
//
// These two notes are written as a pair: each ends by comparing its package
// against the other one, because the whole point of the field is that the two
// answers are different. A note naming the other record package's term is
// therefore ordinary and correct, and it is logged rather than reported. What
// is left as a finding is a name no record-carrying package has at all, which
// is exactly the deletion this is about.
//
// There is no reaching-anything arm here. A note that names no terms is a
// legitimate note — "nothing in this package moves with the core count" names
// nothing and is an answer — and the case where that is WRONG is already the
// completeness check's finding, one per site it could not find in the note.
func checkCoresNoteNamesNothingThatIsGone(t *testing.T, root string,
	recordDirs map[string]bool, notes map[string]coresNote) {

	t.Helper()
	dirs := keysOf(recordDirs)

	wanted := map[string][]string{}
	terms := 0
	for _, dir := range dirs {
		wanted[dir] = notes[dir].terms
		terms += len(wanted[dir])
	}
	if terms == 0 {
		return
	}

	// # Each package for its own note first, and the others only if it has to
	//
	// A note names its own package's terms. That is what it is for, and it is
	// true of both notes today for every term either of them names. Scanning
	// every record package for the UNION of every note's terms — which is what
	// this did — pays the cross-reference case on every green run, and that
	// case has never happened.
	//
	// So the first pass asks each package about its own note, and the second
	// runs only over what the first could not find. On a repository where the
	// notes are right, that second pass does not happen at all.
	//
	// # What that was worth, which is less than it looks
	//
	// Half the scan, and about a thousandth of a second: 0.011s before,
	// 0.010s after, which is at the edge of what seven takings of sixty runs
	// can see. The reason is in identifiersIn — it stops looking for a term
	// once it has found one, and the terms the two notes share are words like
	// `runtime` and `min`, which the first file answers. Scanning the wrong
	// package for them was never expensive; it was just work for a case that
	// has not happened.
	//
	// Worth doing and worth saying what it bought, because a structural
	// argument that predicts a saving and delivers a thousandth is a
	// structural argument somebody should be able to check.
	// One reader for both passes, so the second does not re-open and re-parse
	// what the first has already read. See packageSource.
	src := newPackageSource()
	has := map[string]map[string]bool{}
	unresolved := map[string]bool{}
	for _, dir := range dirs {
		if len(wanted[dir]) == 0 {
			continue
		}
		has[dir] = identifiersIn(t, src,
			filepath.Join(root, filepath.FromSlash(dir)), wanted[dir])
		for _, term := range wanted[dir] {
			if !has[dir][term] {
				unresolved[term] = true
			}
		}
	}
	if len(unresolved) > 0 {
		// The pair: each note ends by comparing its package with the other,
		// so a term one of them names may belong to the other. Asked for only
		// the terms that need it, which is none on a green run.
		missing := keysOf(unresolved)
		for _, dir := range dirs {
			found := identifiersIn(t, src,
				filepath.Join(root, filepath.FromSlash(dir)), missing)
			if has[dir] == nil {
				has[dir] = map[string]bool{}
			}
			for term := range found {
				has[dir][term] = true
			}
		}
	}

	for _, dir := range dirs {
		var held, elsewhere []string
		for _, term := range wanted[dir] {
			if has[dir][term] {
				held = append(held, term)
				continue
			}
			// The pair: each note compares its package with the other one, so
			// a term belonging to the other record package is the note doing
			// what it was written to do.
			from := ""
			for _, other := range dirs {
				if other != dir && has[other][term] {
					from = other
					break
				}
			}
			if from != "" {
				elsewhere = append(elsewhere, term+" ("+from+")")
				continue
			}
			t.Errorf("%s's `%s` names %s, and no package in this repository "+
				"that carries a timings record has such an identifier.\n\n"+
				"That note is a claim about which of this package's terms "+
				"move with the core count, and it is read by somebody the arm "+
				"has just told they are on a different machine. A term that "+
				"has been deleted leaves its name in the sentence, and the "+
				"sentence goes on reading as a measured attribution — which "+
				"is the note being wrong in the direction nobody can see, one "+
				"step round from the direction %s already covers.\n\n"+
				"Either the term has been renamed, in which case the note "+
				"needs the new name and probably a new number; or it has gone, "+
				"in which case the sentence about it has gone too and what is "+
				"left is the measurement the remaining terms account for.\n\n"+
				"A word is read as a term when the note spells it inside "+
				"backquotes, which is the note saying it means a declaration "+
				"rather than a word. If this one is prose, take the "+
				"backquotes off it and nothing here will have an opinion "+
				"about it.",
				dir, timingsCoresNote, term,
				"checkCoresNoteNamesEveryScaledTerm")
		}
		if len(held) == 0 && len(elsewhere) == 0 {
			continue
		}
		also := ""
		if len(elsewhere) > 0 {
			also = fmt.Sprintf(", and %d belonging to the package it compares "+
				"itself with: %s", len(elsewhere), strings.Join(elsewhere, ", "))
		}
		t.Logf("%s's `%s` names %d term(s) this package still has: %s%s.", dir,
			timingsCoresNote, len(held), strings.Join(held, ", "), also)
	}
}

// termsNamedIn is every identifier a note spells inside backquotes, sorted and
// without repeats.
//
// # Why the backquotes are the rule
//
// They are the author saying "this is code". Every other way of deciding is
// this arm guessing at English, and the guess it used to make — camelCase —
// missed `workers` and would have reported a prose word that happened to be
// spelled that way. A convention the note follows is exact in both directions
// and costs nothing, because a note that spells its terms as code is a note
// that reads better anyway.
//
// # What is taken out of a backquoted span
//
// Every identifier in it, not the span itself. A note writes
// `workers := runtime.GOMAXPROCS(0)` as one quoted phrase because that is how
// somebody says what a declaration DOES, and what is being claimed to exist is
// each of the three names in it. So the span is split on everything that
// cannot be part of a Go identifier and what is left that starts like one is a
// term — which makes `0` not a term, and `:=` not a term, without either being
// a special case.
//
// A backquoted span with no identifier in it contributes nothing, which is
// what makes quoting a number or a flag harmless.
func termsNamedIn(note string) []string {
	seen := map[string]bool{}
	var out []string
	for i := 0; ; {
		open := strings.Index(note[i:], "`")
		if open < 0 {
			break
		}
		open += i
		close := strings.Index(note[open+1:], "`")
		if close < 0 {
			// An unpaired backquote is the end of the quoted spans, not the
			// start of one running to the end of the note.
			break
		}
		close += open + 1
		for _, word := range strings.FieldsFunc(note[open+1:close], func(r rune) bool {
			return !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
				r >= '0' && r <= '9')
		}) {
			if seen[word] {
				continue
			}
			// A Go identifier starts with a letter or an underscore, which is
			// what separates a name from the `0` in `GOMAXPROCS(0)`.
			if c := word[0]; !(c == '_' || c >= 'a' && c <= 'z' ||
				c >= 'A' && c <= 'Z') {
				continue
			}
			seen[word] = true
			out = append(out, word)
		}
		i = close + 1
	}
	sort.Strings(out)
	return out
}

// packageSource is one or more directories' Go files, read and parsed at most
// once each however many questions are asked of them.
//
// # What it exists for
//
// identifiersIn used to take a directory and a term list, read the directory,
// read every file that might hold a term, parse it, and throw all of it away.
// The cores-note check asks twice — once for each package's own terms, and
// again over the other packages for whatever the first pass could not find —
// so a repository where a note is WRONG did the os.ReadDir, the os.ReadFile
// and the parser.ParseFile a second time, over files whose trees had been in
// memory a microsecond earlier.
//
// It cost nothing while the second pass did not run, which is every green run,
// and that is exactly the argument that stops being true at the moment
// something breaks: the run already failing is the one that pays twice. A
// census whose cost depends on whether it is about to report is a census
// nobody measures under the conditions it matters in.
//
// # What is cached and what is not
//
// The listing, the bytes and the syntax trees, all keyed by path, and each
// file's parse attempted once. The ANSWER is not cached: a second question
// about a different term list re-reads the trees it already has, which is an
// ast.Inspect over a handful of files and is not what was expensive.
//
// Errors are reported once per file for the same reason — a file that cannot
// be opened is one finding about that file, not one per asking.
//
// # What bounds it
//
// A cache with no stated limit is a cache nobody notices growing, and the
// thing that would grow this one is named right here in the file: a third
// package carrying a timings record, which timingsRecordCopies explicitly
// contemplates and which the trigger above exists to prompt a decision about.
//
// So the directories this may hold is coresNoteScanDirs, and asking about one
// more is a finding rather than a quiet doubling. It is the same arrangement
// repositoryParseBudget has — a number that forces a decision rather than a
// limit for its own sake — and it fires at the same moment the record's own
// trigger does, which is the moment somebody is already reading about the
// trade.
type packageSource struct {
	fset *token.FileSet
	// dir -> its .go file names, sorted.
	listed map[string][]string
	// path -> its bytes, and whether they could be read.
	raw  map[string][]byte
	read map[string]bool
	// path -> its tree, nil for a file that would not parse. Presence in
	// `parsed` is what says the attempt has been made.
	trees  map[string]*ast.File
	parsed map[string]bool
	// Files already reported as unreadable or unparseable.
	told map[string]bool
}

// How many directories the cores-note scan may hold the source of at once.
//
// # Why it is its own name and not timingsRecordCopies spelled again
//
// The two are the same number today and they answer different questions.
// timingsRecordCopies is how many copies of the record the written
// copy-argument covers — a statement about a duplication somebody reasoned
// about once, in internal/themehistory/timings_test.go. This is how much
// source one check may keep in memory at once.
//
// Sharing the identifier made the second question invisible. A reader raising
// timingsRecordCopies because a third package legitimately grew a record —
// exactly what that trigger asks them to consider, and a decision the message
// there walks them through — would have doubled this cache on the way past,
// with no reason written for the second half and nothing prompting them to
// supply one.
//
// # Why it is still defined as timingsRecordCopies rather than as 2
//
// Because the coupling is real, and a bare 2 here would be a second number to
// keep in step by hand. The set this scan reads IS the set of packages
// carrying a record: each record's package is listed once for its own terms,
// and once more for the cross-package pass — over directories already held, so
// costing nothing. Growing that set is the only thing that grows this.
//
// What the separate name buys is a place for the second answer. Somebody who
// decides the copy count should go to three and this cache should not can say
// so here in one edit, and the two numbers part without either of them
// becoming a literal nobody can trace.
const coresNoteScanDirs = timingsRecordCopies

// listedDirs is the directories already held, as a set, for the message above.
func (p *packageSource) listedDirs() map[string]bool {
	out := make(map[string]bool, len(p.listed))
	for dir := range p.listed {
		out[dir] = true
	}
	return out
}

func newPackageSource() *packageSource {
	return &packageSource{
		fset:   token.NewFileSet(),
		listed: map[string][]string{},
		raw:    map[string][]byte{},
		read:   map[string]bool{},
		trees:  map[string]*ast.File{},
		parsed: map[string]bool{},
		told:   map[string]bool{},
	}
}

// identifiersIn is which of these names appear as an identifier somewhere in
// the Go source of one directory.
//
// # The shape, which is packageLevelNames's
//
//	read the directory     one os.ReadDir, the .go files in it
//	scan the bytes         a file that does not contain the name cannot use
//	                       it. No false negative: a use of `foldWalk` contains
//	                       the bytes `foldWalk`
//	parse the hits         and the PARSE is what decides, which is the whole
//	                       reason this is not a grep — the note being checked
//	                       is itself a string in one of these files, and every
//	                       term in it would match its own text
//
// A read or a parse that fails is reported rather than skipped, for
// packageLevelNames's reason: the answer it would otherwise give is "this name
// is not here", which is the direction that invents a finding about somebody's
// note out of a file this could not open.
//
// # Why this is a function and not a method on packageSource
//
// repositoryWalkRow.besides names it as the function this read goes through,
// and the arm over that field finds a call the way every census in this
// repository finds one: a BARE identifier, because a `pkg.Fn(…)` is another
// package's and an `x.Fn(…)` is a method call, and neither is what an
// unqualified call resolves to. A method here would be a read that arm cannot
// see — the row would say nothing calls it and be right.
//
// That is the watched code being shaped by the watcher, which is worth saying
// out loud rather than leaving as a puzzle. The cache is still a value with
// methods; what stays a function is the one thing something else is holding.
func identifiersIn(t *testing.T, p *packageSource, dir string,
	want []string) map[string]bool {

	t.Helper()
	found := map[string]bool{}
	wanted := map[string]bool{}
	for _, w := range want {
		wanted[w] = true
	}
	for _, name := range p.filesIn(t, dir) {
		full := filepath.Join(dir, name)
		raw, ok := p.bytesOf(t, full)
		if !ok {
			continue
		}
		hit := false
		for _, w := range want {
			if !found[w] && bytes.Contains(raw, []byte(w)) {
				hit = true
				break
			}
		}
		if !hit {
			continue
		}
		file := p.treeOf(t, full, raw)
		if file == nil {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && wanted[id.Name] {
				found[id.Name] = true
			}
			return true
		})
	}
	return found
}

// filesIn is a directory's .go files, sorted, listed once.
//
// The first listing of a directory is what counts against the budget: a
// question about a directory already held costs nothing, and a question about
// a new one is the cache growing.
//
// # The budget is advisory, deliberately
//
// Over it, this reports with t.Errorf and then lists the directory anyway.
// Nothing about the reading is refused, and the cache really does grow past
// coresNoteScanDirs on that run.
//
// The alternative is to return nil, and the cost of that is the wrong finding
// in front of the wrong person. A check whose scan was truncated reports what
// the truncation did — every term the note names reads as one this package no
// longer has, because the files that declare them were never listed — so the
// run would carry one true finding about a cache and a dozen invented ones
// about a cores note, with nothing saying which was which. That is the same
// argument the ReadDir failure below makes, and it ends the same way: say
// what is wrong with the scan, and let the scan still give its real answer.
//
// So this is a TRIGGER and not a limit. What it bounds is how long a growing
// cache can go unremarked, which is one run — not how much memory this may
// hold, which is bounded by the repository.
func (p *packageSource) filesIn(t *testing.T, dir string) []string {
	t.Helper()
	if names, done := p.listed[dir]; done {
		return names
	}
	if len(p.listed) >= coresNoteScanDirs {
		t.Errorf("the cores-note scan has been asked about %s, which makes %d "+
			"directory(ies) this run, and coresNoteScanDirs is %d — the "+
			"number of package(s) this repository keeps a timings record in: "+
			"%s.\n\n"+
			"This scan holds every listing, every file's bytes and every "+
			"syntax tree it reads until the check returns, and the set it "+
			"reads is the set of packages that carry a record. A third "+
			"directory is either that set having grown — in which case the "+
			"record's own trigger is firing too, and the decision to make is "+
			"the one it describes — or this scan being used for something "+
			"else, in which case what bounds it is no longer the size of that "+
			"set.\n\n"+
			"coresNoteScanDirs is defined as timingsRecordCopies and is not "+
			"the same question: raising the copy count because a third "+
			"package legitimately grew a record says nothing about what this "+
			"cache should be allowed to hold. Decide both, and write the "+
			"second reason beside coresNoteScanDirs.",
			dir, len(p.listed)+1, coresNoteScanDirs,
			strings.Join(keysOf(p.listedDirs()), ", "))
	}
	p.listed[dir] = nil
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Errorf("reading %s to check what its `%s` names: %v.\n\n"+
			"Without it, every term that note names reads as one this "+
			"package no longer has, which would be this arm inventing a "+
			"finding out of a directory it could not open.", dir,
			timingsCoresNote, err)
		return nil
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
	p.listed[dir] = names
	return names
}

// bytesOf is a file's contents, read once.
func (p *packageSource) bytesOf(t *testing.T, full string) ([]byte, bool) {
	t.Helper()
	if done := p.read[full]; done {
		raw, ok := p.raw[full]
		return raw, ok
	}
	p.read[full] = true
	raw, err := os.ReadFile(full)
	if err != nil {
		if !p.told[full] {
			p.told[full] = true
			t.Errorf("%s could not be read while checking what `%s` names: "+
				"%v.\n\nA file this cannot open is a file that might use "+
				"every term in the note, and the answer it would otherwise "+
				"give is that none of them are here.", full, timingsCoresNote,
				err)
		}
		return nil, false
	}
	p.raw[full] = raw
	return raw, true
}

// treeOf is a file's syntax tree, parsed once, nil when go/parser refused it.
func (p *packageSource) treeOf(t *testing.T, full string, raw []byte) *ast.File {
	t.Helper()
	if p.parsed[full] {
		return p.trees[full]
	}
	p.parsed[full] = true
	file, err := parser.ParseFile(p.fset, full, raw, parser.SkipObjectResolution)
	if err != nil {
		// Only for a file the byte scan kept: this one contains a term
		// somewhere, and whether that is an identifier or a sentence is
		// precisely what the parse was going to decide.
		if !p.told[full] {
			p.told[full] = true
			t.Errorf("%s contains one of the terms `%s` names and go/parser "+
				"could not read it: %v.\n\nIf the build is failing this is "+
				"the same failure said twice and the other one is more "+
				"useful; if the build is green, this file is not what the "+
				"build compiles, and the term may be in it.", full,
				timingsCoresNote, err)
		}
		return nil
	}
	p.trees[full] = file
	return file
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

// A sentence's claim about how many Go files this repository tracks.
//
// # Why the form is fixed
//
// The five sentences that quote this number wrote it three ways — `386
// tracked Go files`, `386 of them`, `386 where` — and the last two are only
// numbers because of the clause before them says what "them" is. That is a
// reading no regexp makes, and guessing how far back a pronoun reaches, in
// paragraphs that also quote wall clocks, core counts and budgets, is a check
// that invents findings.
//
// So the convention is that the number and what it counts go in one breath.
// It is the same rule coresAttribution states for the terms it names — those
// go in backquotes, and a term named in prose is neither claimed nor checked
// — applied to a figure instead of a name: a number written in this form is a
// reading of the tree and is held to it, and a number written any other way
// is English.
//
// Singular is matched as well as plural. `1 tracked Go file` is the same
// claim, and a rule that could not read it would be a rule with a hole in it
// at exactly the size where somebody would notice.
var trackedGoFileFigure = regexp.MustCompile(`\b(\d+) tracked Go files?\b`)

// proseFigure is one such claim: where it is written, what it says, and
// whether it stands in a comment or in a string constant.
//
// The last is not decoration. A string constant in this repository can be
// something another package's test reads without running this one —
// coresAttribution is exactly that, evaluated out of the source by
// stringLiteralValue — so editing one has a second consequence that editing a
// comment does not, and a person handed a list of lines to fix wants to know
// which kind each one is before they start.
type proseFigure struct {
	rel   string
	line  int
	count int
	in    string
}

// figureList renders figures for a failure message. Beside the type rather
// than in messagelists_test.go for the reason the other named renderers are:
// what it encodes is which fields a reader needs in order to go and find the
// sentence, which is a fact about this type. See messagelists_test.go for the
// convention it follows.
func figureList(in []proseFigure) string {
	return listOf(in, func(f proseFigure) string {
		return fmt.Sprintf("%s:%d says %d, in %s", f.rel, f.line, f.count, f.in)
	})
}

// prose is a run of text assembled out of pieces that each have a line of
// their own — the lines of a comment group, or the literals a `+` chain joins
// — collapsed to single spaces, with enough kept to say which line any part
// of the result was written on.
//
// # Why the collapse is necessary
//
//	// … a reading of a repository on a day — 386
//	// tracked Go files where verifyTimingsTakenOn was taken — and it …
//
// That is one phrase to the person reading it and two to anything matching
// raw lines, and it is how the first of the five figures is actually written.
// Joining the group and matching the join is the only thing that reads it the
// way a reader does.
//
// # Why the lines are kept
//
// A comment group here is routinely forty lines long. Reporting a figure at
// the line the GROUP starts on would point a person at the top of a section
// and leave them to find which sentence, which is most of the work the
// message exists to save them. So each word carries the line it came from and
// a match is attributed to the word it starts at.
type prose struct {
	text  strings.Builder
	at    []int // byte offset within text of each word
	lines []int // the line that word was written on
}

// add appends one piece — a comment line, or a string literal — written on
// the given line. Words are separated by exactly one space whatever the piece
// held, which is the collapse; a piece that splits a word in half (`"38" +
// "6 tracked Go files"`) is therefore not found, and that is the one shape
// this declines to read rather than guessing at.
func (p *prose) add(line int, s string) {
	for _, word := range strings.Fields(s) {
		if p.text.Len() > 0 {
			p.text.WriteByte(' ')
		}
		p.at = append(p.at, p.text.Len())
		p.lines = append(p.lines, line)
		p.text.WriteString(word)
	}
}

// lineAt is the line the word covering this byte offset was written on.
func (p *prose) lineAt(off int) int {
	// The last word that starts at or before the offset. SearchInts finds the
	// first start strictly greater, so one back from it is the word the
	// offset is inside.
	i := sort.SearchInts(p.at, off+1) - 1
	if i < 0 || len(p.lines) == 0 {
		return 0
	}
	return p.lines[i]
}

// figures is every tracked-Go-file claim in this run of text.
func (p *prose) figures(rel, in string) []proseFigure {
	text := p.text.String()
	var out []proseFigure
	for _, m := range trackedGoFileFigure.FindAllStringSubmatchIndex(text, -1) {
		n, err := strconv.Atoi(text[m[2]:m[3]])
		if err != nil {
			// Unreachable while the pattern is `\d+`, and cheaper to handle
			// than to argue about: a figure nobody can turn into a number is
			// not a figure this can hold either way.
			continue
		}
		out = append(out, proseFigure{rel: rel, line: p.lineAt(m[0]),
			count: n, in: in})
	}
	return out
}

// fileCountFiguresIn is every tracked-Go-file figure in one file's prose.
//
// Comments and string constants both, because the count is quoted in both:
// four of the five sentences are doc comments and the fifth is inside
// coresAttribution, which is a paragraph that happens to be a constant so
// that another package's test can read it without running this one.
func fileCountFiguresIn(fset *token.FileSet, rel string,
	file *ast.File) []proseFigure {

	var out []proseFigure
	for _, cg := range file.Comments {
		var p prose
		for _, c := range cg.List {
			line := fset.Position(c.Pos()).Line
			switch {
			case strings.HasPrefix(c.Text, "//"):
				p.add(line, c.Text[2:])
			case strings.HasPrefix(c.Text, "/*"):
				// One node holding many lines. Split so that each word still
				// carries the line it is on, which is the whole point of
				// prose keeping them.
				body := strings.TrimSuffix(strings.TrimPrefix(c.Text, "/*"), "*/")
				for i, l := range strings.Split(body, "\n") {
					p.add(line+i, l)
				}
			}
		}
		out = append(out, p.figures(rel, "a comment")...)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		e, ok := n.(ast.Expr)
		if !ok {
			return true
		}
		// The gate is stringLiteralValue's own answer, so that what is read
		// here is exactly the set of expressions this package already treats
		// as a string constant. A `+` chain it can evaluate is taken whole
		// and not descended into — otherwise the chain would be read once as
		// itself and again as each of its halves, and one sentence would
		// arrive as three findings.
		if stringLiteralValue(e) == "" {
			return true
		}
		var p prose
		stringLiteralProse(fset, e, &p)
		out = append(out, p.figures(rel, "a string constant")...)
		return false
	})
	return out
}

// stringLiteralProse is the expression stringLiteralValue evaluates, kept as
// its pieces instead of as one string, so that a figure is attributed to the
// literal it is written in rather than to the head of the chain.
//
// The two walk the same shapes deliberately: a sentence this can read and
// that cannot, or the other way round, would be a finding whose line number
// pointed somewhere else.
func stringLiteralProse(fset *token.FileSet, e ast.Expr, p *prose) {
	switch x := e.(type) {
	case *ast.BasicLit:
		if x.Kind != token.STRING {
			return
		}
		v, err := strconv.Unquote(x.Value)
		if err != nil {
			v = x.Value
		}
		p.add(fset.Position(x.Pos()).Line, v)
	case *ast.BinaryExpr:
		if x.Op != token.ADD {
			return
		}
		stringLiteralProse(fset, x.X, p)
		stringLiteralProse(fset, x.Y, p)
	case *ast.ParenExpr:
		stringLiteralProse(fset, x.X, p)
	}
}

// checkProseFileCounts holds every sentence that quotes this repository's
// tracked-Go-file count to the count the walk above has just made.
//
// Everything here is a reading of what that walk collected. Nothing in it
// parses, reads or enumerates anything.
func checkProseFileCounts(t *testing.T, from string, goFiles int,
	figures []proseFigure) {

	t.Helper()
	// The walk reaching anything, for the reason every other question here
	// says it: this one is looking for a PHRASE, and a rewrite that keeps
	// every sentence and changes how each one says the number would leave a
	// check over nothing that passes and reads as a clean result.
	if len(figures) == 0 {
		t.Fatalf("no sentence in this repository quotes a tracked-Go-file "+
			"count, and %s found %d of them.\n\n"+
			"Five sentences did, in wasm/verify/repowalks_test.go and "+
			"wasm/verify/timings_test.go, each pricing a repository-wide "+
			"walk against how many files it hands to go/parser. Either they "+
			"are gone — in which case delete this question, because a walk "+
			"nobody prices needs no arm over the price — or they now say the "+
			"number some other way, in which case see trackedGoFileFigure "+
			"for the form this reads and why it is the only one it can.",
			from, goFiles)
	}

	var wrong []proseFigure
	for _, f := range figures {
		if f.count != goFiles {
			wrong = append(wrong, f)
		}
	}
	if len(wrong) > 0 {
		t.Errorf("%d sentence(s) quote a tracked-Go-file count this "+
			"repository does not have. %s enumerated %d: %s.\n\n"+
			"Every one of these is a figure somebody wrote down while it was "+
			"true, in a sentence pricing a walk against the number of files "+
			"it parses, and the tree has moved under it since. That is not a "+
			"bug in the walks and there is nothing to fix in the code: edit "+
			"each line above to say %d.\n\n"+
			"If this is firing on a commit that added or removed Go files "+
			"and nothing else, that is this arm working. The figures used to "+
			"drift silently and did, by four, over five sentences at once — "+
			"which in a package whose argument is that an unattributed "+
			"number is worth nothing left five numbers with nothing behind "+
			"them. The trade is written up in this file's header.",
			len(wrong), from, goFiles, figureList(wrong), goFiles)
	}

	// Logged whether or not anything failed, because the useful moment for
	// this number is BEFORE somebody edits prose: `go test -v -run
	// TestTheShapesThisRepositoryKeepsTwoCopiesOfAreInStep` is how a person
	// adding a walk finds the count to write into the sentence they are
	// about to add.
	t.Logf("%d tracked Go file(s) by %s; %d sentence(s) quote that count: %s.",
		goFiles, from, len(figures), figureList(figures))
}
