package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The repository's one shared parse, and the questions that ride on it.
//
// # What this is
//
// Four arms in this package hand every Go file in the tree to go/parser, and
// repositoryParseBudget says four is the number. This is the fourth, and it
// is the one that ANSWERS that budget: when a question needs a repository-wide
// parse and does not need a walk of its own, it becomes a question here
// rather than a fifth parse.
//
//	the timings records   five machine fields, a reporting arm, a standing
//	                      sentence about what a core count is worth, and a
//	                      taking command for every band, in
//	                      wasm/verify/timings_test.go and
//	                      internal/themehistory/timings_test.go. See
//	                      checkTimingsRecordCopies
//	the two-copy shapes   every declaration this repository deliberately
//	                      keeps two copies of — the functions each census
//	                      resolves a qualifier with, the state they keep, the
//	                      band reader and the verdict levers — held to being
//	                      the same declaration in both packages. See
//	                      checkTwoCopyDecls; and, asked the other way round,
//	                      every name both record packages declare held to
//	                      being in one of the two lists at all. See
//	                      checkSharedNamesAreAccountedFor
//	the paths it is asked the import paths those censuses name, held to being
//	                      ones this module could actually import. See
//	                      checkImportPathsAreImportable
//	the file counts       every sentence in the repository that prices
//	                      something against how many Go files the tree holds.
//	                      See checkProseFileCounts
//	the comment text      two rules over every comment there is: a line that
//	                      is one comment written twice, and a tab anywhere but
//	                      the leading indent. See checkCommentText
//	the names in prose    every Go test named in a comment, in a string
//	                      constant, or anywhere in a tracked file this
//	                      repository does not parse — the docs, the scripts,
//	                      the browser pass's .mjs — held to being a test this
//	                      repository has. See checkProseNamesResolve
//
// # Why this file exists, which is a name that had stopped being true
//
// The walk was in copies_test.go and was called
// `TestTheShapesThisRepositoryKeepsTwoCopiesOfAreInStep` — backquoted
// because the sentence is about the name having changed, which is the
// convention quotedprose_test.go states — because the first
// three questions on it are about shapes this repository keeps two copies of
// and the parse was built for them. The last two are not. A figure in prose
// is a copy of a fact about the TREE, which is already a stretch; a garbled
// comment is not a copy of anything.
//
// What never changed is what actually unifies them, and it was written down
// from the start: they are one arm BECAUSE they are one repository-wide
// parse. That is a fact about cost, not about subject, and a file named for
// the subject of its first question was going to keep being wrong as
// questions arrived. Two arrived in one session.
//
// So the walk is here and each question's check is owned by the file that
// owns its subject — which was already true of three of them before this
// file existed. What is left in copies_test.go is the copies questions, which
// is what that name was always about.
//
// # Why they are subtests and not sections of one function
//
// Each of these ends in a reaching-anything arm and every one of those is a
// t.Fatalf — a walk over nothing passes silently and reads as a clean result,
// so the only honest thing to do about it is to stop. A Fatalf stops the
// GOROUTINE, so N questions in one function is N questions any one of which
// can end the others: a repository where the records had been renamed
// reported that and said nothing about whether the import resolver's two
// copies were still in step.
//
// t.Run gives each question its own goroutine and its own failure boundary at
// no cost to the walk — the parse has already happened and the subtests read
// what it built. So the unit of the WALK is the repository and the unit of a
// FINDING is the question.
//
// The order is the cheap-to-say first: the two that read declarations the
// walk collected, then the file counts, which are one integer comparison,
// then the comment text, then the records, whose own checks are the longest.
//
// # What stops this becoming a place to put things
//
// walkQuestionBudget, which is six. Every question here arrived with a good
// individual argument — the parse budget is full, this walk has the parse,
// the question needs no walk of its own — and that argument will keep being
// true of every future question, which is exactly what makes it dangerous. A
// reason that always wins is not a reason.
//
// So the number is written down and the census in repowalks_test.go holds it,
// beside repositoryWalkBudget and repositoryParseBudget, which bound the two
// other ways this cost can grow. The seventh question is a decision: either
// the questions here are no longer one thing and some of them want a walk, or
// the budget moves and the reason goes beside it.
//
// That is not hypothetical and it has happened once. The budget was five, and
// the question that made it six — the tests named in prose — arrived in the
// session after it was written. It did what it was for: the raise is argued
// in the constant's own doc rather than having been a line nobody noticed.
//
// # The two files this walk cannot read
//
// Per QUESTION and not per walk, which is the correction this file made on
// the way out of the old one. There used to be a single skip — copies_test.go
// was not parsed at all — written for the record question's sake, and it was
// over-broad: measured by taking it out, the only findings copies_test.go
// produces are against the file-count rule, from figures quoted as examples.
// Every other question reads it clean, and now does.
//
//	proseFigureRulesFile   quotes counts as examples of the form it holds
//	commentRulesFile       quotes the incident verbatim, which breaks both of
//	                       its own two rules
//	proseNameRulesFile     quotes four dead test names as the evidence for
//	                       its rule. The exemption is on its PROSE and not on
//	                       the file: it declares no test, so skipping it
//	                       outright would exempt nothing and lose nothing
//
// Three of the six questions carry one, and it is the same shape every time:
// a rule worth writing down is worth showing an example of, and an example of
// a rule is a thing the rule catches. The fourth time this happens it is
// worth asking whether the exemption should be a convention rather than a
// list — a marker in the prose, say — but three lines with three reasons is
// cheaper than a mechanism, and each of these says what it is protecting.
//
// A check that reads its own explanation as a finding is a check nobody can
// leave a comment in. That is the same reason gitquoting_test.go skips
// itself; what is new is that the exemption belongs to the rule rather than
// to the walk, so a question that needs none does not inherit one.
func TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn(t *testing.T) {
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
	// Every package-level name each directory declares, which is what makes
	// the two-copy lists answerable in the other direction: not "is every
	// registered shape the same in both packages" but "is every name both
	// packages declare in one of the lists at all". Names only — the text
	// comparison stays where it was, over the shapes the lists name. See
	// checkSharedNamesAreAccountedFor.
	declaredIn := map[string]map[string]bool{}
	// The other two shapes, read off the same parse. Kept beside the record
	// rather than in walks of their own because a fifth repository-wide parse
	// is the decision repositoryParseBudget exists to force, and neither of
	// these needs one.
	var resolverDecls []twoCopyDecl
	var asks []importPathAsk
	var coreSites []coreCountSite
	// Every sentence in the tree that quotes the tracked-Go-file count, and
	// the count itself. Both are readings of this same walk: the figures come
	// off the parse below and the number comes off the enumeration above, so
	// the fourth question costs the walk nothing it was not already paying.
	var figures []proseFigure
	goFiles := 0
	// And the comment text, which is the one question here that is about
	// neither a declaration nor a copy. Collected as two finding lists and a
	// count, because the walk collects and the subtests judge — see
	// commentFindingsIn.
	var twice, tabbed []commentLine
	commentLines, commentFiles := 0, 0
	// And what the prose points AT: every test this repository declares, and
	// every Test-shaped name its comments and string constants mention. Both
	// halves come off this one parse, which is why the question is here.
	var declaredTests []string
	var namedTests []proseName
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
			// The tests this repository names outside its Go sources: the
			// documentation, a hook script, the browser pass's own .mjs.
			// Read rather than parsed, because a test name in a shell
			// comment is a word in a file — see testNamesInText, including
			// what 136 files cost.
			raw, readErr := os.ReadFile(filepath.Join(root,
				filepath.FromSlash(rel)))
			if readErr == nil {
				namedTests = append(namedTests,
					testNamesInText(rel, raw)...)
			}
			continue
		}
		// Counted BEFORE the skip below, because what the prose figures claim
		// is the size of the tree and copies_test.go is a file in it. Counted
		// before the parse too: a file go/parser cannot read is still a Go
		// file git is tracking.
		goFiles++
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
		// Which of the two-copy shapes this file declares, and which
		// import paths its censuses name. One walk over the declarations, two
		// questions — see twoCopyDeclarationsIn.
		fileDecls, fileAsks := twoCopyDeclarationsIn(fset, rel, file)
		resolverDecls = append(resolverDecls, fileDecls...)
		asks = append(asks, fileAsks...)
		// And every place this file reads the core count, which is what makes
		// a cores note checkable rather than merely present.
		coreSites = append(coreSites, coreCountSitesIn(t, fset, rel, file)...)
		// The two comment rules, off the comment groups this parse has just
		// built. No filter and no second pass: the rules are about every
		// comment line there is, which is the whole of what makes them worth
		// having over a directory's worth.
		if rel != commentRulesFile {
			commentFiles++
			n, fileTwice, fileTabbed := commentFindingsIn(fset, rel, file)
			commentLines += n
			twice = append(twice, fileTwice...)
			tabbed = append(tabbed, fileTabbed...)
		}
		// The tests this file declares, and the ones its prose names. The
		// exemption is on the MENTIONS only: prosenames_test.go's header
		// quotes four dead names as the evidence for the rule, and declares
		// no test of its own, so skipping the file outright would be
		// exempting nothing and skipping the prose is the whole of it.
		fileTests, fileNames := testNamesIn(fset, rel, file)
		declaredTests = append(declaredTests, fileTests...)
		if rel != proseNameRulesFile {
			namedTests = append(namedTests, fileNames...)
		}
		// And every sentence in it that quotes the tracked-Go-file count,
		// read off the comments and string constants the parse just built —
		// which is why this is here and not in a scan of its own.
		//
		// Filtered by a byte scan first, which is the same move the walk
		// census makes one file over and cannot produce a false negative: a
		// figure in the held form contains this word, in a comment or in a
		// string, whatever else the file says. It matters because the scan
		// below walks EVERY node of a syntax tree and builds a run of text
		// out of every comment group and every string constant in it, and
		// eleven files in the tree contain the word at all. Measured: 0.09s
		// over all of them and 0.00s with the filter, which is 3% of this
		// package's wall clock for a question with five answers in it.
		//
		// The eleven is deliberately not written as a tracked-Go-file count.
		// It is a reading like the ones the question below holds, and one
		// this file's own skip would keep unheld — so it says what it is
		// worth (almost every file is skipped) rather than a number that
		// would drift with nothing watching it.
		if bytes.Contains(raw, []byte("tracked")) {
			if rel != proseFigureRulesFile {
				figures = append(figures, fileCountFiguresIn(fset, rel, file)...)
			}
		}
		for _, d := range file.Decls {
			// The names first, whatever kind of declaration this is: the
			// question the inversion asks is about a NAME being declared in
			// two packages, and it has to see every declaration to ask it.
			// Methods are left out — a method on a type is not a
			// package-level name, and two packages declaring the same method
			// on their own types is not a copy.
			for _, name := range namesDeclaredBy(d) {
				if declaredIn[name] == nil {
					declaredIn[name] = map[string]bool{}
				}
				declaredIn[name][dir] = true
			}
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
							line: at.Line, doc: recordDoc(decl, vs)}
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

	// One walk, six questions, six failure boundaries. See the header for
	// why this is t.Run and not six sections of one function: each of these
	// ends in a t.Fatalf over a walk that reached nothing, and a Fatalf ends
	// the goroutine it is on.
	// Which directories the two-copy questions are ABOUT, derived rather than
	// written down: they are the packages carrying a timings record, which is
	// what both lists' doc comments describe — two separate `package main`
	// programs, neither able to import the other's tests. Taken from the walk
	// so that a record moving to a third package is a finding there rather
	// than a silent change of subject here.
	recordDirs := map[string]bool{}
	for _, r := range records {
		recordDirs[path.Dir(r.rel)] = true
	}

	t.Run("the declarations kept in two copies", func(t *testing.T) {
		checkTwoCopyDecls(t, resolverDecls)
		checkSharedNamesAreAccountedFor(t, keysOf(recordDirs), declaredIn,
			resolverDecls)
	})
	t.Run("the paths those helpers are asked about", func(t *testing.T) {
		checkImportPathsAreImportable(t, root, asks)
	})
	t.Run("the file counts in this repository's prose", func(t *testing.T) {
		checkProseFileCounts(t, from, goFiles, figures)
	})
	t.Run("the comment text", func(t *testing.T) {
		checkCommentText(t, commentFiles, commentLines, twice, tabbed)
	})
	t.Run("the tests named in prose", func(t *testing.T) {
		checkProseNamesResolve(t, from, declaredTests, namedTests)
	})
	t.Run("the timings records", func(t *testing.T) {
		checkTimingsRecordCopies(t, root, from, len(paths), records, arms,
			notes, coreSites, noteText)
	})
}
