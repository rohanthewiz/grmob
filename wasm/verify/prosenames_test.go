package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"testing"
)

// A test named in a comment is a test this repository has.
//
// # What this is about
//
// One test was renamed and five sentences went on naming the old one — two in
// timings_test.go, two in versionorder_test.go, one of those inside a string
// constant another census prints. gofmt, go vet, go build, `go test ./...`,
// `-race` and all four verify scripts were green with every one of them in
// place, because a name in a comment is text and the only things in this
// repository that read comments as text are two rules about tabs.
//
// A stale name is not cosmetic. These comments are how this repository
// explains itself: "see TestX for the other half of the pair" is an
// instruction, and a reader who follows it to nothing has to guess whether the
// check was renamed, deleted, or never existed — and the third possibility is
// the one that makes them stop trusting the sentence.
//
// # What was measured before this was written
//
// The general form of this rule — an identifier in prose resolves to
// something declared — was not written, because prose is full of
// identifier-shaped words that are not declarations and of qualified names
// from the standard library. A Go TEST name is the narrow case that can be
// recognised without resolving anything: `Test` followed by an upper-case
// letter is a shape nothing else in English has.
//
// Measured over every Go comment and string constant in the repository,
// outside ai_docs:
//
//	336 mentions       of a Test-shaped name
//	25 unresolved      before a comment group is joined
//	20 unresolved      after joining, hyphenated line-splits included
//	6 unresolved       after accepting a PREFIX (see below)
//	4 of those 6       real: comments pointing at tests that do not exist, in
//	                   four packages, none of them the one being worked on
//
// The four were `TestGroupHeaderLabelIsASecondLevelHeading` (the test is
// `…IsAHeading`), `TestNativeBoxIsAVerticalStackNotAnOverlay` (renamed to
// `TestNativeContainersStackTheirChildrenAndDoNotOverlay`),
// `TestNoFloatThisFileComparesIsDerivedTwice` (`TestNoFloatAComparisonRestsOn…`)
// and `TestTheGobindPinIsTheOneInGoMod` (`TestTheGobindReadingsArePinnedTo…`).
// That is the opposite result from the doubled-word rule two sessions ago,
// which scored nine hits and no defects and was declined on the number.
//
// # Why a prefix resolves
//
// Because `go test -run X` matches every test whose name begins with X, so a
// prefix is a mention that still takes a reader where they are going. That is
// not a concession — it is what a test name in a comment is FOR, and this
// repository writes `-run` command lines into comments for exactly that use.
//
// It is also what makes a name wrapped across two comment lines with no
// hyphen readable: the fragment on the first line is a prefix of the whole,
// so it resolves, and the rule does not need to reassemble prose nobody
// hyphenated.
//
// What it costs is that renaming `TestFoo` to `TestFooAndBar` leaves every
// mention of `TestFoo` resolving. That is the right answer rather than a
// missed one: those mentions still run the test they name.
//
// # Why this file's own prose is not read
//
// The four names above are dead, quoted as the evidence for the rule, and
// they are the most useful thing in this header — a reader deciding whether
// the rule is worth its exemption wants to see what it found. So the prose
// here is skipped, by name, the way commenttext_test.go's and
// prosefigures_test.go's are.
//
// The skip is narrower than theirs: this file declares no test, so what is
// exempt is its MENTIONS and not the file. Nothing about it is invisible to
// the other five questions on that walk.
//
// # Where this runs
//
// On the shared repository parse — see sharedparse_test.go. It needs every Go
// file's declarations and every Go file's comments, which is exactly and only
// what that walk has. It is the question that took walkQuestionBudget from
// five to six, one session after that budget was written, and the argument
// for the raise is in the constant's own doc.
var testNameInProse = regexp.MustCompile(`\bTest[A-Z][A-Za-z0-9_]{3,}\b`)

// mightNameATest is the cheap half of the filter above.
//
// The pattern begins with the literal `Test`, so a line without those four
// bytes cannot match it and does not need the regexp run over it. That is the
// whole optimisation and it is worth stating because of the size of what it
// is filtering: this question reads every comment in the repository, and
// running the compiled pattern over each of fifty thousand lines cost 0.07s —
// two and a half per cent of this package, and enough to take it out of the
// band verifyTimingsTakenOn records. With this in front it is 0.01s.
//
// It cannot produce a false negative: `strings.Contains` is exactly the
// necessary condition for the pattern, not an approximation of it.
func mightNameATest(s string) bool {
	return strings.Contains(s, "Test")
}

// The file this rule is written in, and the one file whose prose it does not
// read. A repository path, because the walk that applies it enumerates the
// repository. See the header.
const proseNameRulesFile = "wasm/verify/prosenames_test.go"

// proseName is one mention: the name, where it is written, and whether it
// stands in a comment or in a string constant.
//
// The last matters for the same reason it does for a figure: a string
// constant here can be something another package's test reads or prints, so
// editing one has a second consequence that editing a comment does not.
type proseName struct {
	name, rel, in string
	line          int
}

// proseNameList renders mentions for a failure message. Beside the type for
// the reason the other named renderers are — see messagelists_test.go.
func proseNameList(in []proseName) string {
	return listOf(in, func(m proseName) string {
		return fmt.Sprintf("%s:%d names %s, in %s", m.rel, m.line, m.name, m.in)
	})
}

// The tests this repository deliberately names after they are gone.
//
// A rename is worth recording — "it said three, naming the copies census,
// then called X, now Y" is a sentence that explains a number's history, and
// deleting the dead name from it would leave a reader unable to match the
// record to anything. So these are exempt, by name, with the reason.
//
// The bar is that the mention is ABOUT the name having changed. A sentence
// that merely uses a dead name to point at a live test is the finding this
// rule exists for, and belongs in neither this table nor the file.
//
// It is a graveyard and it should stay small. An entry here whose sentence
// has since been rewritten is dead weight that reads as a guard, which is the
// failure citationExempt's own assertion catches one file over; if this table
// ever gets long enough to be worth an arm of its own, that is the same
// check.
var renamedTestsStillNamed = map[string]string{
	"TestEveryTimingsRecordIsTheSameShape": "wasm/verify/timings_test.go " +
		"tells the history of the number it records — the copies census was " +
		"called this, then something else, and is now a third thing. The " +
		"sentence is about the renaming",
	"TestTheShapesThisRepositoryKeepsTwoCopiesOfAreInStep": "the middle name " +
		"in that same history, and the one sharedparse_test.go's header " +
		"explains itself by: the walk moved out of copies_test.go because a " +
		"name describing its first question had stopped describing the set",
}

// testNamesIn is every test this file declares, and every Test-shaped name
// its prose mentions.
//
// Both off one file, because the walk that calls this is visiting each file
// once and the question needs both halves of the repository: what exists, and
// what is pointed at.
func testNamesIn(fset *token.FileSet, rel string, file *ast.File) (
	declared []string, mentioned []proseName) {

	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		// A method called TestSomething is not a test, and a test is never a
		// method — `go test` runs package-level functions only.
		if !ok || fn.Recv != nil {
			continue
		}
		if testNameInProse.MatchString(fn.Name.Name) {
			declared = append(declared, fn.Name.Name)
		}
	}

	add := func(p *prose, in string) {
		text := string(p.text)
		for _, loc := range testNameInProse.FindAllStringIndex(text, -1) {
			mentioned = append(mentioned, proseName{
				name: text[loc[0]:loc[1]], rel: rel,
				line: p.lineAt(loc[0]), in: in,
			})
		}
	}
	for _, cg := range file.Comments {
		// The pattern is run over the raw lines FIRST, and the group is only
		// assembled when one of them hits. That is what keeps this question
		// affordable: it visits every comment in the repository, fifty
		// thousand lines of them, and building a joined run of text with a
		// line number per word for each group cost about six hundredths of a
		// second — two per cent of this package — where the filter costs
		// nothing measurable.
		//
		// It cannot produce a false negative on anything but a name
		// hyphenated within its first few characters: a wrapped name leaves a
		// FRAGMENT on the first line, and a fragment of a Test-shaped name is
		// Test-shaped. `Test-` on its own is the one spelling this would
		// miss, and nobody wraps a line there.
		hit := false
		for _, c := range cg.List {
			if mightNameATest(c.Text) {
				hit = true
				break
			}
		}
		if !hit {
			continue
		}
		// Joined by prose rather than by hand, so that a name wrapped across
		// two lines — hyphenated or not — is one name here and one name to a
		// reader. See prose.add.
		var p prose
		for _, c := range cg.List {
			at := fset.Position(c.Pos()).Line
			if strings.HasPrefix(c.Text, "//") {
				p.add(at, c.Text[2:])
				continue
			}
			body := strings.TrimSuffix(strings.TrimPrefix(c.Text, "/*"), "*/")
			for i, line := range strings.Split(body, "\n") {
				p.add(at+i, line)
			}
		}
		add(&p, "a comment")
	}
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		// Same filter, and one literal at a time rather than the `+` chain
		// stringLiteralValue would join. A literal is one line, so this
		// attributes a mention to the line it is written on exactly — and
		// what it gives up is a name split across a concatenation
		// (`"TestFoo" + "Bar"`), which is the same limit the figures rule
		// states one file over and has no instance here either.
		if !mightNameATest(lit.Value) {
			return false
		}
		var p prose
		p.add(fset.Position(lit.Pos()).Line, lit.Value)
		add(&p, "a string constant")
		return false
	})
	return declared, mentioned
}

// checkProseNamesResolve holds every test named in this repository's prose to
// being a test the repository has.
//
// Everything here is a reading of what the shared walk collected. Nothing in
// it parses, reads or enumerates anything.
func checkProseNamesResolve(t *testing.T, from string, declared []string,
	mentioned []proseName) {

	t.Helper()
	// The walk reaching the DECLARATIONS, which is this question's own
	// reaching arm and not the comment question's. A run that collected
	// mentions and no declarations would report every sentence in the
	// repository as a stale name, which is a check failing loudly in the
	// shape of a thousand findings rather than of one.
	if len(declared) == 0 {
		t.Fatalf("no `func Test…` was found by %s, and this repository is "+
			"most of a thousand of them.\n\n"+
			"Nothing is being claimed about the prose here: the half of this "+
			"question that says what EXISTS came back empty, so every "+
			"mention would read as stale. The reading is wrong rather than "+
			"the repository being bare.", from)
	}

	exact := make(map[string]bool, len(declared))
	for _, d := range declared {
		exact[d] = true
	}
	resolves := func(name string) bool {
		if exact[name] || renamedTestsStillNamed[name] != "" {
			return true
		}
		// `go test -run X` matches every test X is a prefix of, so a prefix
		// is a mention that still takes a reader to the test. See the header.
		for _, d := range declared {
			if strings.HasPrefix(d, name) {
				return true
			}
		}
		return false
	}

	var stale []proseName
	for _, m := range mentioned {
		if !resolves(m.name) {
			stale = append(stale, m)
		}
	}
	if len(stale) > 0 {
		t.Errorf("%d sentence(s) name a test this repository does not "+
			"have: %s.\n\n"+
			"Each of these was a real test when it was written about. A "+
			"comment saying \"see TestX\" is an instruction, and one that "+
			"leads nowhere makes a reader guess whether the check was "+
			"renamed, deleted or never written — and it is the third "+
			"possibility that costs, because it is the one that makes them "+
			"stop following the next sentence.\n\n"+
			"A PREFIX resolves: `go test -run TestFoo` runs "+
			"`TestFooAndBar`, so a mention that still takes a reader to the "+
			"test is not a finding. These do not. Either the test was "+
			"renamed — point the sentence at the new name — or it is gone, "+
			"and the sentence is describing a guard this repository no "+
			"longer has, which is worth more than a broken link.\n\n"+
			"If the sentence is ABOUT the renaming, which is a thing several "+
			"records here do on purpose, put the name in "+
			"renamedTestsStillNamed with the reason.",
			len(stale), proseNameList(stale))
	}

	t.Logf("%d test(s) declared, %d mention(s) in prose, %d name(s) "+
		"deliberately kept after renaming. Enumerated by %s.",
		len(declared), len(mentioned), len(renamedTestsStillNamed), from)
}
