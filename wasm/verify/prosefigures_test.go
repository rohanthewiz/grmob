package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The figures in this repository's prose that are readings of the tree, and
// the rule that makes them checkable.
//
// A sentence that says "387 tracked Go files" holds a number that lives
// somewhere else — the tree — and nothing joins the two. Five sentences did,
// and all five had drifted by four at once before a reader who had not
// written them noticed. What holds them now is the file-count question on the
// shared repository parse; see sharedparse_test.go for why it rides that walk
// rather than taking one.
//
// It can be an arm where the wall clocks in the same paragraphs cannot. A
// timing is a fact about one computer, so asserting one would fail on every
// machine that is not it; a file count is a fact about the TREE, so a figure
// that disagrees here disagrees for everybody. See verifyTimingsTakenOn for
// the other half of that argument.
//
// The cost is that adding a Go file to this repository fails a test until
// those sentences are edited. That is the point and not a side effect — the
// alternative is the five sentences nobody could trust — and the message
// names every one with its line so the fix is one pass rather than a search.
//
// This file is the one the rule does not read, because the paragraphs here
// quote counts as examples and would otherwise be findings about themselves.
// See the exemption list in sharedparse_test.go.

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
//	// … a reading of a repository on a day — 387
//	// tracked Go files where verifyTimingsTakenOn was taken — and it …
//
// That is one phrase to the person reading it and two to anything matching
// raw lines, and it is how the first of the five figures is actually written.
// The number in the quotation is not itself held — this is the file the walk
// skips — and it is here to show where the line break falls rather than to
// state a count.
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
	text  []byte
	at    []int // byte offset within text of each word
	lines []int // the line that word was written on
}

// add appends one piece — a comment line, or a string literal — written on
// the given line. Words are separated by exactly one space whatever the piece
// held, which is the collapse; a piece that splits a word in half (`"38" +
// "6 tracked Go files"`) is therefore not found, and that is the one shape
// this declines to read rather than guessing at.
//
// # The one exception, which is a hyphen at the end of a piece
//
// A long name wrapped across two comment lines is written here the way it is
// written in print:
//
//	// because TestEveryGitListingAsksForNul-
//	// SeparatedPaths has parsed the whole tree since…
//
// That is one word to a reader and the hyphen is the split rather than part
// of it, so a piece beginning where the last one ended in `-` is joined with
// the hyphen removed and no space. Measured over every Go comment in this
// repository: twelve lines end in a hyphen and every one of them is a word
// split across lines. There is no counterexample to exempt.
//
// The rule is applied at a PIECE boundary and not between words, because
// within a line a trailing `-` is punctuation somebody typed and the next
// word is a different word.
func (p *prose) add(line int, s string) {
	for i, word := range strings.Fields(s) {
		if len(p.text) > 0 {
			if i == 0 && p.text[len(p.text)-1] == '-' {
				p.text = p.text[:len(p.text)-1]
			} else {
				p.text = append(p.text, ' ')
			}
		}
		p.at = append(p.at, len(p.text))
		p.lines = append(p.lines, line)
		p.text = append(p.text, word...)
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
	text := string(p.text)
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
	// TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn` is how a
	// person adding a walk finds the count to write into the sentence they
	// are about to add.
	t.Logf("%d tracked Go file(s) by %s; %d sentence(s) quote that count: %s.",
		goFiles, from, len(figures), figureList(figures))
}

// The file the file-count rule is written in, and the one file it is not
// applied to. A repository path, because the walk that applies it enumerates
// the repository.
//
// Renaming the file does not break the constant — it is a string, and nothing
// checks that it names anything. What it does is make the file-count question
// fail loudly on this file's own examples, which is the right end of that:
// the skip goes missing in a way somebody sees on the next run rather than
// one that widens coverage silently and correctly until it does not.
const proseFigureRulesFile = "wasm/verify/prosefigures_test.go"
