package todoapp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// docs/tutorial-todo.md opens by promising that "every excerpt below is taken
// from [examples/todoapp] verbatim". That promise rotted once: the app moved
// onto the components widgets (Screen, InputRow, SegmentedControl, ListRow,
// Separator) and the tutorial kept showing the hand-rolled core.Row/core.Button
// version it replaced, down to a colorDanger constant this package no longer
// declares. A reader following along would have written code that does not
// compile against the source they were told they were reading.
//
// This test is the pin. It re-reads the tutorial's Go fences and checks that
// each one that talks about this app's symbols actually traces back to the
// source — so the next time app.go moves, the tutorial has to move with it or
// the build says so.
//
// # What "traces back" means
//
// Not byte equality. An excerpt is legitimately allowed to:
//
//   - re-indent, because a fragment lifted out of App's body is shown at the
//     indentation the fence reads best at, not the one it had;
//   - drop blank lines and the source's inline comments, which are written for
//     someone reading the file, not someone reading the tutorial;
//   - elide with a literal "..." line, the tutorial's own convention for "and
//     the rest of this list continues".
//
// So the rule is: every non-elided line, stripped, appears in the source in
// the same order, and a "..." permits a gap. That is strong enough to have
// caught all four of the stale excerpts and weak enough to leave the fences
// readable.
func TestTutorialExcerptsMatchTheSource(t *testing.T) {
	source := readLines(t,
		filepath.Join("app.go"),
		filepath.Join("store.go"),
	)

	md, err := os.ReadFile(filepath.Join("..", "..", "docs", "tutorial-todo.md"))
	if err != nil {
		t.Fatalf("read tutorial: %v", err)
	}

	for _, fence := range goFences(string(md)) {
		if !aboutThisApp(fence) {
			// Fences that illustrate the framework in general (a bare
			// core.Button, the package skeleton in section 2) are not claims
			// about this package and are not this test's business.
			continue
		}
		if missing, ok := traces(fence, source); !ok {
			t.Errorf("tutorial excerpt is not in examples/todoapp — first line that\n"+
				"does not appear in order is:\n\t%s\n\nfull excerpt:\n%s",
				missing, fence)
		}
	}
}

// goFences returns the body of every ```go block in the markdown.
var fenceRe = regexp.MustCompile("(?s)```go\n(.*?)```")

func goFences(md string) []string {
	var out []string
	for _, m := range fenceRe.FindAllStringSubmatch(md, -1) {
		out = append(out, m[1])
	}
	return out
}

// appSymbols are names that only this package's own code uses. A fence
// mentioning one is making a claim about examples/todoapp; a fence mentioning
// none is illustrating the framework and is out of scope.
var appSymbols = []string{
	"todos", "draft", "filterBar", "todoRow", "addTodo",
	"clearButton", "setDone", "visible",
	"components.Screen", "components.InputRow",
}

func aboutThisApp(fence string) bool {
	for _, sym := range appSymbols {
		if strings.Contains(fence, sym) {
			return true
		}
	}
	return false
}

// traces reports whether every meaningful line of the excerpt appears in the
// source in order. On failure it returns the first line that did not, which is
// the only part of a 20-line fence a reader needs to see.
func traces(fence string, source []string) (string, bool) {
	want := meaningful(fence)
	if len(want) == 0 {
		return "", true
	}
	// Anchored at every possible start, because the same line ("}", say) recurs
	// throughout the file and a greedy first match would wander off.
	for start := range source {
		if line, ok := matchFrom(want, source[start:]); ok {
			return "", true
		} else if start == len(source)-1 {
			return line, false
		}
	}
	return want[0], false
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

// meaningful strips a fence to the lines that carry code: no blanks, no
// comment-only lines (the tutorial paraphrases the source's comments in prose).
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

func readLines(t *testing.T, paths ...string) []string {
	t.Helper()
	var out []string
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		for _, l := range strings.Split(string(b), "\n") {
			if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "//") {
				out = append(out, l)
			}
		}
	}
	return out
}
