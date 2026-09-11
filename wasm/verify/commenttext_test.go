package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

// The comment text in this repository, held to being text somebody typed.
//
// # What this is about
//
// repowalks_test.go:395 was, for at least a session:
//
//	// Counted in WALKS and not in functions: a helper called twice is two	// Counted in WALKS and not in functions: a helper called twice is two
//	// walks, and the budget is about what a run pays.
//
// One comment concatenated with itself by a botched edit. gofmt is clean on
// it, go vet is clean on it, and every census in this package — six of which
// walk the whole repository — is clean on it, because not one of them reads a
// comment as TEXT. They read declarations, call sites, string constants and
// now figures; the line above is none of those, and it survived a session of
// arms being written two files away from it.
//
// # Deciding the set, which is the work
//
// A rule here is only worth having if nothing already holds it, and two of
// the obvious candidates turned out to fail that on measurement rather than
// on argument:
//
//	trailing whitespace   gofmt already strips it from comment lines —
//	                      checked, not assumed — and `gofmt -l` is one of this
//	                      repository's verification paths. A rule for it would
//	                      be a second statement of something already held
//	a doubled word        measured over every Go comment in the repository:
//	                      nine hits and not one of them a defect. Six are
//	                      correct English — `what it is is ARIA's`, `a subtest
//	                      that fails fails its parent` — two are deliberate
//	                      data — `a part in a million million`, `"sermons
//	                      sermons sermons"` — and the one that read like a
//	                      real duplication, `a List of rows rows`, is a
//	                      sentence naming the `rows` parameter beside it. A
//	                      rule that needs a nine-row exemption table on the
//	                      day it is written, for a shape that has never once
//	                      been wrong here, is a rule whose finding is noise
//
// What is left is two shapes that nothing holds and that nobody types on
// purpose. Both are at zero across every Go file in this repository today, so
// neither arrives with an exemption table.
//
// # Why it parses, which the first design did not
//
// These are rules about RAW lines — a tab, a repeated marker — so the obvious
// implementation is a byte scan, and that is what this was going to be. It
// cannot work. A line beginning with `//` is not necessarily a comment: it
// can be a line inside a raw string literal, and this package writes plenty
// of those. Nor can a scanner track raw strings by counting backquotes,
// because the comments here are full of backquoted names and a comment
// containing an odd number of them puts the count into a string that is not
// there.
//
// So go/parser says WHERE the comments are and ast.Comment.Text says what
// each one says — which is the source as written, tabs and all, because the
// parser does not normalise a comment's interior. The count of lines read is
// logged on every run rather than written here, because a figure in prose is
// a figure that drifts — and the parse is not this check's to pay at all: see
// below.
//
// # Why this file is not read by it
//
// The example above is a real one, quoted as it stood, and it breaks both
// rules — which makes this file a finding about itself twice over. So it is
// skipped by name, which is what gitquoting_test.go and copies_test.go do for
// the same reason: a check that reads its own explanation as a finding is a
// check nobody can leave a comment in.
//
// The cost is real and worth saying out loud rather than burying in the skip.
// This is the file somebody editing these rules will be editing, and it is
// the one file the rules do not cover. The alternative was to describe the
// incident instead of quoting it, and a shape described in words is a shape
// the next reader has to reconstruct — which for two rules about invisible
// characters is most of what the header is for.
//
// # Why it is the repository and rides somebody else's walk
//
// It was this directory first, on the argument that the incident was here and
// that this package is mostly prose. Measured, that argument does not survive:
// wasm/verify is 12,326 of the repository's 50,190 comment lines — a reading
// taken when this was widened, and one that drifts upward the way every other
// count here does — so a rule scoped to it reads a QUARTER of what it is
// about. core and components carry another third between them and are written
// the same way. The live number is in the log line on every run.
//
// Widening it cannot be a walk of its own. Both budgets are full —
// repositoryWalkBudget is 7 against 7 walks, repositoryParseBudget is 4
// against 4 parses — and an eighth walk would pay its own `git ls-files` and
// its own read of every tracked file to reach files that four existing walks
// have already parsed. That is precisely what those budgets exist to stop, so
// the answer they force is the one taken: this rides the shared parse.
//
// # Why the copies walk, whose subject this is not
//
// Because it is the shared parse, and has been since the budget made it one.
// That walk's unifying principle was never its subject — its own header says
// the three shapes are one arm BECAUSE they are one repository-wide parse —
// and two of its four existing questions already have their checks in
// importnames_test.go rather than in the file the walk is named after. The
// arrangement is: one parse, and each question's check owned by the file that
// owns its subject.
//
// So this is the fifth question and checkCommentText lives here, beside the
// rules and the argument for them, which is where somebody changing either
// would look. What that costs is a walk whose name describes its largest
// question rather than all of them; see the header of copies_test.go, which
// now says so.
// checkCommentText holds the repository's comment text to the two rules
// above, off the syntax trees the copies walk has already built.
//
// Everything here is a reading of what that walk collected. Nothing in it
// parses, reads or enumerates anything.
func checkCommentText(t *testing.T, filesRead, lines int,
	twice, tabbed []commentLine) {

	t.Helper()
	// The scan reaching anything, for the reason every other question on that
	// walk says it — and this one has a version of the failure that is easy
	// to arrive at: go/parser discards comments unless asked for them with
	// parser.ParseComments, and a file parsed without it comes back with an
	// empty Comments slice and no error at all. Every rule below would then
	// pass over nothing, on every run, for as long as it took somebody to
	// notice.
	if lines == 0 {
		t.Fatalf("no comment line was found in %d Go file(s).\n\n"+
			"This repository is nearly fifty thousand comment lines, so the "+
			"reading is wrong rather than the tree being bare. The likeliest "+
			"cause is the parse mode on the walk above: comments are only "+
			"built when parser.ParseComments is asked for.", filesRead)
	}

	if len(twice) > 0 {
		t.Errorf("%d comment line(s) are one comment written twice: %s.\n\n"+
			"A botched edit, and nothing else produces this shape: the text "+
			"before the second `//` and the text after it are the same "+
			"sentence. gofmt is clean on it, go vet is clean on it, and no "+
			"other check in this repository reads a comment as text — the "+
			"last one survived at least a session in a file that is itself "+
			"six censuses.\n\n"+
			"Only EQUAL halves are a finding. A comment line carrying a "+
			"second, different comment is ordinary — this repository has "+
			"forty-odd of them, mostly annotated code samples inside doc "+
			"comments — so the rule is the shape the incident actually had "+
			"and not the general one, which would be unusable.",
			len(twice), commentLineList(twice))
	}

	if len(tabbed) > 0 {
		t.Errorf("%d comment line(s) contain a tab inside the text: %s.\n\n"+
			"A tab that is not a table indent is not something anybody "+
			"types. It is a join, a paste out of aligned output, or an "+
			"editor — and the last one to appear here was the separator "+
			"holding two copies of one sentence together on a single "+
			"line.\n\n"+
			"The indent is exempt and nothing else is: a comment whose text "+
			"begins with tabs is the gofmt-rendered code block or table this "+
			"repository uses everywhere, and those are stripped before the "+
			"rule looks. Leading SPACES are not stripped, so a space-then-tab "+
			"indent is a finding — that is mixed indentation, which renders "+
			"differently in every viewer and is the artifact rather than the "+
			"convention.\n\n"+
			"If a tab is genuinely wanted mid-line, it is being used to align "+
			"something, and alignment inside a comment is a thing this "+
			"repository spells with spaces so that it survives being read "+
			"anywhere.",
			len(tabbed), commentLineList(tabbed))
	}

	t.Logf("%d comment line(s) in %d Go file(s), held to 2 rule(s). %s is the "+
		"one file not read; see its header.", lines, filesRead,
		commentRulesFile)
}

// commentFindingsIn is the two rules applied to one file's comments.
//
// Returned as two slices rather than reported here because the walk that
// calls this is collecting, not judging: every question on it reports in its
// own subtest, with its own failure boundary. See the header of
// copies_test.go.
func commentFindingsIn(fset *token.FileSet, rel string,
	file *ast.File) (lines int, twice, tabbed []commentLine) {

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			for _, line := range commentLinesOf(fset, rel, c) {
				lines++
				if line.writtenTwice() {
					twice = append(twice, line)
				}
				if line.hasInteriorTab() {
					tabbed = append(tabbed, line)
				}
			}
		}
	}
	return lines, twice, tabbed
}

// The file the rules are written in, and the one file they are not applied
// to. A repository path, because the walk that applies them enumerates the
// repository. Named as a constant so that the skip and the sentence
// explaining it cannot come apart.
//
// Renaming the file does not break the constant — it is a string, and nothing
// checks that it names anything. What it does is make the comment question
// fail loudly on this file's own two examples, which is the right end of
// that: the skip goes missing in a way somebody sees on the next run rather
// than one that widens coverage silently and correctly until it does not.
const commentRulesFile = "wasm/verify/commenttext_test.go"

// commentLine is one line of one comment, as written.
type commentLine struct {
	file string
	line int
	// The text after the `//`, or the line as it stands inside a `/* */`.
	// Kept raw: the rules here are about characters the parser does not
	// normalise, so anything trimmed on the way in is a rule that cannot fire.
	text string
}

// commentLineList renders comment lines for a failure message. Beside the
// type for the reason the other named renderers are — what it encodes is
// which fields a reader needs in order to go and find the line. See
// messagelists_test.go for the convention.
func commentLineList(in []commentLine) string {
	return listOf(in, func(c commentLine) string {
		return fmt.Sprintf("%s:%d (%q)", c.file, c.line, c.text)
	})
}

// writtenTwice is the shape repowalks_test.go:395 had: an interior `//` with
// the same sentence on both sides of it.
//
// Both halves have to be non-empty. `// // still a comment` is a commented-out
// comment, which is a thing people do on purpose, and its first half is empty.
func (c commentLine) writtenTwice() bool {
	at := strings.Index(c.text, "//")
	if at < 0 {
		return false
	}
	before := strings.TrimSpace(c.text[:at])
	after := strings.TrimSpace(c.text[at+2:])
	return before != "" && before == after
}

// hasInteriorTab is a tab anywhere but in the leading indent.
//
// Only TABS are stripped from the front. A comment indented with spaces and
// then a tab is mixed indentation rather than this package's table idiom, and
// that is the shape worth reporting rather than exempting.
func (c commentLine) hasInteriorTab() bool {
	return strings.Contains(strings.TrimLeft(c.text, "\t"), "\t")
}

// commentLinesOf is one comment node as the lines a reader sees.
//
// A `//` comment is one line and one node. A `/* */` comment is one node
// holding as many lines as it spans, and each of those is a line somebody
// typed and can garble on its own — so it is split, and each piece carries
// the line it is actually on rather than the line the block starts at. There
// is no block comment in this repository today; handling one is three lines,
// and a rule that silently skipped the first one written would be a rule
// whose coverage quietly depended on a style nobody has stated.
func commentLinesOf(fset *token.FileSet, file string,
	c *ast.Comment) []commentLine {

	at := fset.Position(c.Pos()).Line
	if strings.HasPrefix(c.Text, "//") {
		return []commentLine{{file: file, line: at, text: c.Text[2:]}}
	}
	body := strings.TrimSuffix(strings.TrimPrefix(c.Text, "/*"), "*/")
	var out []commentLine
	for i, line := range strings.Split(body, "\n") {
		out = append(out, commentLine{file: file, line: at + i, text: line})
	}
	return out
}
