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
// # And the files that are not Go
//
// This repository names its tests outside its Go sources too — in the docs,
// in a hook script, in the browser pass's own .mjs — and three of those were
// stale when this was widened to reach them: a shell comment naming
// `TestEveryGitListingInAScriptAsksForZ`, a page naming
// `TestNoRoleCollidesWithTheTabPanelWiring`, and browser.mjs pointing at a
// mobile/verify census that had become two tests.
//
// Those files are read rather than parsed, because there is nothing to parse:
// a test name in a shell comment is a word in a file. The walk's enumeration
// already lists them and the cost of reading them again is 136 files and 2.5
// megabytes, measured at 0.003s — which is why this is a read here rather
// than a change to what citingFiles hands back to its six callers.
//
// One shape is skipped: a name written as `func TestX(`. The documentation
// contains example tests a reader is meant to WRITE — `func TestCounter(t
// *testing.T)` in the getting-started page, `func TestNoHookDrift` in the
// debug-mode one — and those are declarations in a sample rather than
// citations of anything. A citation in prose is never spelled with `func` in
// front of it.
//
// # Where this runs
//
// On the shared repository parse — see sharedparse_test.go. It needs every Go
// file's declarations and every Go file's comments, which is exactly and only
// what that walk has. It is the question that took walkQuestionBudget from
// five to six, one session after that budget was written, and the argument
// for the raise is in the constant's own doc.
var testNameInProse = regexp.MustCompile(`\bTest[A-Z][A-Za-z0-9_]{3,}\b`)

// testNamesInText is every test named in a file this repository does not
// parse: documentation, scripts, the browser pass's JavaScript.
//
// Line by line rather than joined, because none of these is Go and none of
// them wraps a name across lines the way a gofmt'd comment does. What that
// gives up is a name split across a line break in a markdown paragraph, and
// there is no instance of one.
func testNamesInText(rel string, raw []byte) []proseName {
	var out []proseName
	for i, line := range strings.Split(string(raw), "\n") {
		if !mightNameATest(line) {
			continue
		}
		// Per line rather than over the file, because none of these is Go and
		// a backquote pairs within a line in all three of markdown, shell and
		// JavaScript. Over the whole file a fenced block's ``` would pair
		// with something a screen away.
		runs := quotedRuns(line)
		for _, loc := range testNameInProse.FindAllStringIndex(line, -1) {
			// A sample somebody is meant to write, not a citation. See the
			// header: the docs declare example tests, and a declaration is
			// not a claim that this repository has one.
			if strings.HasSuffix(line[:loc[0]], "func ") {
				continue
			}
			// A name the sentence is ABOUT rather than pointing at. See
			// quotedprose_test.go.
			if quotedAt(runs, loc[0]) {
				continue
			}
			out = append(out, proseName{
				name: line[loc[0]:loc[1]], rel: rel, line: i + 1,
				in: "text",
			})
		}
	}
	return out
}

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

// The tests this repository deliberately names after they are gone used to
// be listed here, by name, with the reason for each.
//
// They are not listed any more, and nothing replaced the list with a smaller
// list: the two sentences that needed it now say what they mean in the
// prose itself, by putting the dead name in backquotes. See
// quotedprose_test.go for the convention and for what it was measured to
// cost, which is one mention repository-wide and that one a code sample.
//
// Worth recording because the table was the shape of a real argument and the
// argument survives: a rename IS worth writing down — "it said three, naming
// the copies census, then called X, now Y" explains a number's history and
// deleting the dead name from it leaves a reader unable to match the record
// to anything — and the bar for saying so has not moved. The mention has to
// be ABOUT the name having changed. What changed is only where that is
// stated: in the sentence, where a reader is, rather than in a table one
// file over that a reader has to be told exists.
//
// The table's own doc predicted its failure mode — an entry whose sentence
// had since been rewritten, reading as a guard over nothing — and a
// convention carried in the sentence cannot have that failure, because
// rewriting the sentence takes the marker with it.

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
		// Over the JOINED group, so that a backquoted name wrapped across two
		// comment lines is one quoted run here as it is to a reader. The
		// pairing is left to right and a stray backquote opens nothing, which
		// means a malformed comment gets checked rather than skipped — see
		// quotedRuns.
		runs := quotedRuns(text)
		for _, loc := range testNameInProse.FindAllStringIndex(text, -1) {
			// A name the sentence is ABOUT rather than pointing at. See
			// quotedprose_test.go.
			if quotedAt(runs, loc[0]) {
				continue
			}
			// A sample somebody is meant to write, not a citation — the same
			// skip testNamesInText applies to the documentation, and for the
			// same reason the header gives: a citation in prose is never
			// spelled with `func` in front of it. It is needed on this side
			// too because this repository keeps code samples in raw string
			// literals, and examples/tutorial/chapter8.go's is a
			// `func TestMain(` that resolves today only by prefix accident.
			if strings.HasSuffix(text[:loc[0]], "func ") {
				continue
			}
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
		// Unwrapped first, because a raw literal is delimited with the same
		// byte the quoting convention uses and its own delimiters would
		// otherwise pair around everything inside it. See unwrapRawLiteral.
		var p prose
		p.add(fset.Position(lit.Pos()).Line, unwrapRawLiteral(lit.Value))
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
		if exact[name] {
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
			"records here do on purpose, put the dead name in backquotes: a "+
			"token in backquotes is quoted rather than claimed, and this "+
			"question does not read it. That is the same convention "+
			"coresAttribution states for the terms it names. The bar is that "+
			"the sentence is about the name having CHANGED — using a dead "+
			"name to point at a live test and backquoting it hides this "+
			"finding rather than recording anything.",
			len(stale), proseNameList(stale))
	}

	t.Logf("%d test(s) declared, %d mention(s) in prose held to resolving. "+
		"Enumerated by %s.\n\n"+
		"A mention in backquotes is not counted here and not checked: it is "+
		"a name the sentence is about rather than one it points at. See "+
		"quotedprose_test.go.", len(declared), len(mentioned), from)
}
