package counter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/grmobtest"
	"github.com/rohanthewiz/grmob/internal/shotclaims"
)

// TestMain turns debug mode on for the package, so every pass driven below
// is audited for hook drift and the tests assert the audit came back empty.
// It is cheap insurance on the one app in this repository a reader is most
// likely to copy: NewState here is a positional hook, and this file is the
// example of calling it correctly.
func TestMain(m *testing.M) {
	core.SetDebugMode(true)
	m.Run()
}

// The tests below drive the app through grmobtest, which renders the same
// tree a host receives and dispatches the callback IDs it finds there, so a
// tap here takes the path a tap in the browser does.

// --- The app -------------------------------------------------------------

func TestCounterCountsBothWays(t *testing.T) {
	app := grmobtest.New(t, App)

	if !app.HasText("Count: 0") {
		t.Fatal("the counter does not open on zero")
	}
	app.Tap("+")
	app.Tap("+")
	if !app.HasText("Count: 2") {
		t.Fatal("two taps of + did not reach 2")
	}
	// The minus is a U+2212 MINUS SIGN and not a hyphen, here and in both
	// documents that quote this file. Worth a sentence because the two are
	// indistinguishable in most fonts, and a test that typed the hyphen
	// would fail with `no tappable node on screen is a Button labelled "-"`,
	// which reads as the button having gone.
	app.Tap("−")
	if !app.HasText("Count: 1") {
		t.Fatal("a tap of − did not come back to 1")
	}
	app.NoConcerns()
}

// --- The screenshot ------------------------------------------------------

// TestTheCounterScreenshotStillShowsWhatItClaims drives this app to the
// state docs/images/counter.png was taken in and asserts the picture's
// strings are still on the screen.
//
// See internal/shotclaims for why a screenshot needs holding at all. What
// this arm adds to the manifest's own checks is the only thing a manifest
// cannot do for itself: it renders the app.
func TestTheCounterScreenshotStillShowsWhatItClaims(t *testing.T) {
	claims := shotclaims.For("examples/counter")
	if len(claims) == 0 {
		t.Fatal("no claim names examples/counter, and docs/images/counter.png " +
			"is a picture of it. Either the shot has gone or the manifest has " +
			"lost its row; both leave this test asserting nothing.")
	}

	for _, c := range claims {
		app := grmobtest.New(t, App)

		// The state in the claim, driven rather than posed: three taps of
		// the same button a person would press, through the same dispatch
		// path the browser host uses.
		app.Tap("+")
		app.Tap("+")
		app.Tap("+")

		// Shows rather than HasText: "on the screen" is not one node type.
		// The label of a Button and the content of a Text are both words a
		// reader of the picture sees, and a check that read only one of them
		// would answer a narrower question than the one a screenshot raises.
		root := app.Tree()
		for _, want := range c.Shows {
			if !grmobtest.Shows(root, want) {
				t.Errorf("docs/images/%s shows %q and this app no longer "+
					"does, %s.\n\n"+
					"The picture is now a statement about a version of this "+
					"screen that does not exist. Re-take it (see wasm/shots) "+
					"and move the claim and the README's caption with it.",
					c.File, want, c.State)
			}
		}
		app.NoConcerns()
	}
}

// --- The two documents that quote this file ------------------------------

// The documents that show this package's source, and the one rule they are
// held to.
//
// README.md opens with this app and docs/getting-started.md opens with it
// again. Before this package existed they were two snippets of a file that
// did not exist, and they had already drifted apart — one had
// core.Padding(24) and the other did not, which is a difference a reader
// only discovers by typing both in.
//
// The rule is the one examples/todoapp's tutorial excerpts are held to,
// and the reasoning is written out there: not byte equality, because a
// fence legitimately re-indents, drops the source's comments and elides.
// Every non-blank, non-comment line of the fence appears in app.go, in
// order.
var quotingDocs = []string{
	filepath.Join("..", "..", "README.md"),
	filepath.Join("..", "..", "docs", "getting-started.md"),
}

// counterMarkers are lines only this package's source has. A fence in
// those documents carrying one is quoting this file; a fence carrying none
// is illustrating something else and is not this test's business.
//
// Narrow on purpose. `func App(ctx *core.Context) core.View` appears in
// eight documents in this repository and names a different app in most of
// them, so it is exactly the wrong marker — what identifies this one is
// the state it declares and the name it binds.
var counterMarkers = []string{
	"count := core.NewState(ctx, 0)",
	`func AppName() string { return "Counter" }`,
}

func TestTheDocumentedCounterIsThisPackage(t *testing.T) {
	source := sourceLines(t, "app.go")

	found := 0
	for _, doc := range quotingDocs {
		raw, err := os.ReadFile(doc)
		if err != nil {
			t.Fatalf("read %s: %v", doc, err)
		}
		for _, fence := range goFences(string(raw)) {
			if !quotesCounter(fence) {
				continue
			}
			found++
			if missing, ok := traces(fence, source); !ok {
				t.Errorf("%s quotes this package and the first line that is "+
					"not in app.go, in order, is:\n\t%s\n\nfull excerpt:\n%s",
					doc, missing, fence)
			}
		}
	}
	// Both documents open with this app, so two is the floor and the arm is
	// about the markers still selecting anything at all: a rename in app.go
	// would quietly make every fence "not about this package", and a test
	// that checks nothing passes.
	if found < 2 {
		t.Fatalf("only %d fence(s) in %v were recognised as quoting this "+
			"package. Both documents open with it, so the markers have "+
			"stopped matching rather than the quotations having gone.",
			found, quotingDocs)
	}
}

func quotesCounter(fence string) bool {
	for _, m := range counterMarkers {
		if strings.Contains(fence, m) {
			return true
		}
	}
	return false
}

// goFences returns the body of every ```go block in the markdown.
func goFences(md string) []string {
	var out []string
	rest := md
	for {
		i := strings.Index(rest, "```go\n")
		if i < 0 {
			return out
		}
		rest = rest[i+len("```go\n"):]
		j := strings.Index(rest, "```")
		if j < 0 {
			return out
		}
		out = append(out, rest[:j])
		rest = rest[j+3:]
	}
}

// meaningful strips a block to the lines that carry code: no blanks, no
// comment-only lines, since a document paraphrases the source's comments
// in prose rather than repeating them.
func meaningful(block string) []string {
	var out []string
	for _, l := range strings.Split(block, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "//") {
			continue
		}
		out = append(out, l)
	}
	return out
}

func sourceLines(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return meaningful(string(b))
}

// traces reports whether every meaningful line of the excerpt appears in
// the source in order, returning the first line that did not.
//
// Anchored at every possible start, because a line like `}` recurs
// throughout the file and a greedy first match would wander off it.
func traces(fence string, source []string) (string, bool) {
	want := meaningful(fence)
	if len(want) == 0 {
		return "", true
	}
	var first string
	for start := range source {
		line, ok := matchFrom(want, source[start:])
		if ok {
			return "", true
		}
		if start == 0 {
			first = line
		}
	}
	return first, false
}

func matchFrom(want, source []string) (string, bool) {
	i := 0
	for _, w := range want {
		if w == "..." {
			continue // the gap itself; the next literal resyncs
		}
		j := i
		for j < len(source) && source[j] != w {
			j++
		}
		if j == len(source) {
			return w, false
		}
		i = j + 1
	}
	return "", true
}
