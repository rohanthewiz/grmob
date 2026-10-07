package docsnippets

import (
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The component docs show whole widgets and a whole test harness, and
// SKILL-component.md says its widget "compiles as written". That was last
// true by hand: on 2026-10-03 the blocks were pasted into a throwaway package
// and vetted and tested. Nothing re-checked them after that, so a renamed
// core or comps symbol would leave every copy of Tally and Spoiler broken
// in the one place a reader copies from (N-087).
//
// This test is that throwaway package, made on every run:
//
//	README.md ─────────────┐  go fences under the headings in the docs table
//	components.md ─────────┼─────────────────────────────────────────────┐
//	SKILL-component.md ────┘                                             ▼
//	  t.TempDir()/go.mod     replace github.com/rohanthewiz/grmob => <this checkout>
//	  t.TempDir()/readme/    fence01.go
//	  t.TempDir()/concepts/  fence01.go … fence04_test.go, given_test.go
//	  t.TempDir()/skill/     fence01.go … fence03_test.go
//	                                   │
//	                                   ▼
//	                       go vet ./...   then   go test ./...
//
// # Why build and run, rather than type-check in process
//
// go/types with a source importer would answer "does it compile" without a
// subprocess. But the testing fences contain a real test of the Spoiler
// beside them, and the claim a reader relies on is that it passes, not only
// that it compiles. Running
// the go command is also how cmd/grmob's TestNewScaffoldsAnAppThatBuildsAndPasses
// checks the scaffold, with the same replace-to-this-checkout module.
//
// # Why each doc is its own package
//
// All three docs declare a Tally, and two declare a Spoiler. They are
// separate copies of one example, so they cannot share a package. Within a
// doc, though, the fences are meant to be read together: the concept page's
// test exercises the Spoiler shown above it, and its concern snippet reads
// the Tally's Label. So one doc is one package, and one fence is one file.
//
// # Why errors point into the markdown
//
// Each fence's body is preceded by a //line directive naming the doc and the
// fence's first line. A compile or vet error then reads
// ".../docs/concepts/components.md:251:5: undefined: core.Foo", which is the
// line to fix. Only a synthesized header (package clause and imports, for a
// fence that has none) is reported against the scratch file name.

// section names one heading whose ```go fences are claimed to be whole code.
type section struct {
	// heading is the heading line exactly as written, hashes included.
	heading string
	// fences is how many ```go fences sit under the heading. It is checked,
	// because a selector that silently stopped matching would pass by
	// compiling nothing, and a fence added under a listed heading should
	// be a decision to include it.
	fences int
	// wrap, when set, is a fmt format with one %s. The fence is a fragment
	// written in the voice of a function body, and is pasted into that
	// format rather than compiled at file level.
	wrap string
}

// doc is one markdown file and the package its fences become.
type doc struct {
	path     string // relative to the repository root
	dir      string // package directory in the scratch module
	pkg      string // package name for fences that state none
	sections []section
	// given is source for names that the prose describes and that the
	// fences use but never define. It is written as given_test.go.
	given string
}

// docs is the whole selection. Fences elsewhere in these files are left out
// deliberately: they are fragments that lean on names from another program,
// such as todoRow's Todo, the Callbacks paragraph's w.OnTap, or the View
// interface quoted from core.
var docs = []doc{
	{
		path: "README.md",
		dir:  "readme",
		pkg:  "ui",
		sections: []section{
			{heading: "## Your own components", fences: 1},
		},
	},
	{
		path: "docs/concepts/components.md",
		dir:  "concepts",
		pkg:  "ui",
		sections: []section{
			{heading: "## Writing a leaf widget", fences: 1},
			{heading: "## A widget that owns state", fences: 1},
			// The concern snippet is a const and an if statement, shown as
			// they would sit at the top of Tally's Render. A local const is
			// legal Go, so the whole fence goes into a method body on Tally,
			// which is where its w comes from.
			{heading: "## Accessibility", fences: 1, wrap: "func (w Tally) reportsMisuse() {\n%s\n}\n"},
			{heading: "## Testing a component", fences: 1},
		},
		// The page shows renderDebug and renderPass but describes these two
		// in prose ("findFirst walks the tree with a predicate, and findText
		// matches a Text node's content"), as SKILL-component.md §8 defines
		// them.
		given: `package ui

import "github.com/rohanthewiz/grmob/core"

func findFirst(n *core.Node, pred func(*core.Node) bool) *core.Node {
	if n == nil || pred(n) {
		return n
	}
	for _, c := range n.Children {
		if f := findFirst(c, pred); f != nil {
			return f
		}
	}
	return nil
}

func findText(n *core.Node, s string) *core.Node {
	return findFirst(n, func(x *core.Node) bool { return x.Type == "Text" && x.Props["content"] == s })
}
`,
	},
	{
		path: "ai_docs/SKILL-component.md",
		dir:  "skill",
		pkg:  "ui",
		sections: []section{
			{heading: "## 2. Anatomy of a struct widget", fences: 1},
			{heading: "## 6. Hooks inside a widget", fences: 1},
			{heading: "## 8. Testing", fences: 1},
		},
	},
}

// knownImports maps a package qualifier to its import path, for fences that
// carry no import block. Only qualifiers the fence actually uses are
// imported, because an unused import is a compile error.
var knownImports = map[string]string{
	"core":    "github.com/rohanthewiz/grmob/core",
	"comps":   "github.com/rohanthewiz/grmob/comps",
	"hooks":   "github.com/rohanthewiz/grmob/hooks",
	"fmt":     "fmt",
	"strconv": "strconv",
	"strings": "strings",
	"testing": "testing",
	"time":    "time",
}

func TestComponentDocSnippetsBuildAndPass(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	mod := t.TempDir()
	writeModule(t, root, mod)
	// Every test function the selected fences declare, with how many docs
	// declare it. Read off the fences rather than listed here, so a doc that
	// ships a second test is held to it without an edit to this file.
	shipped := map[string]int{}

	for _, d := range docs {
		abs := filepath.Join(root, d.path)
		raw, err := os.ReadFile(abs)
		if err != nil {
			t.Fatalf("read %s: %v", d.path, err)
		}
		all := goFences(string(raw))

		pkgDir := filepath.Join(mod, d.dir)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, s := range d.sections {
			var picked []fence
			for _, f := range all {
				if f.heading == s.heading {
					picked = append(picked, f)
				}
			}
			if len(picked) != s.fences {
				t.Errorf("%s: %d ```go fence(s) under %q, the table expects %d.\n"+
					"A heading renamed or a fence moved leaves this check compiling nothing; "+
					"update the docs table in this file to match what the doc now claims.",
					d.path, len(picked), s.heading, s.fences)
				continue
			}
			for _, f := range picked {
				n++
				for _, m := range testFunc.FindAllStringSubmatch(f.body, -1) {
					shipped[m[1]]++
				}
				name, src := assemble(d, s, f, abs, n)
				if err := os.WriteFile(filepath.Join(pkgDir, name), []byte(src), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
		if d.given != "" {
			if err := os.WriteFile(filepath.Join(pkgDir, "given_test.go"), []byte(d.given), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if t.Failed() {
		return // the selection is wrong; a build of part of it says nothing
	}

	// vet first, because it compiles the test files too and reports every
	// package's errors in one go; then test, which runs what the docs ship.
	var testOut string
	for _, args := range [][]string{{"vet", "./..."}, {"test", "-count=1", "-v", "./..."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = mod
		// GOWORK=off: a go.work above the temp dir must not pull this
		// checkout in as a second main module. -mod=mod: the scratch go.mod
		// lists only grmob, and the go command may add what it infers.
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("go %s over the component docs' snippets: %v\n%s\n"+
				"Positions name the markdown line through //line directives. "+
				"A position in fenceNN.go is a synthesized header: a qualifier "+
				"missing from knownImports, or a fence that now needs one.",
				strings.Join(args, " "), err, out)
			return
		}
		testOut = string(out)
	}

	// A green "go test" proves nothing if the shipped tests were not in it:
	// a fence filed under fenceNN.go rather than _test.go compiles its Test
	// function as an ordinary one, and nothing runs it. So the run has to
	// name each shipped test, once per doc that ships it. Both testing
	// sections ship one today, so an empty map means the pattern below
	// stopped matching, not that the tests went away.
	if len(shipped) == 0 {
		t.Fatal("no test function found in the selected fences; testFunc has stopped matching how the docs declare one")
	}
	for name, want := range shipped {
		if got := strings.Count(testOut, "--- PASS: "+name+" "); got != want {
			t.Errorf("%s passed %d time(s) in the snippets' run, want %d (once per doc that ships it).\n%s",
				name, got, want, testOut)
		}
	}
}

// testFunc matches a test declaration at the start of a fence line.
var testFunc = regexp.MustCompile(`(?m)^func (Test\w*)\(t \*testing\.T\)`)

// writeModule writes the scratch module's go.mod, pointing grmob at this
// checkout, and copies go.sum so that grmob's own requirements verify
// without a download.
func writeModule(t *testing.T, root, mod string) {
	t.Helper()
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	// The same go version as the repository, so a fence may use whatever the
	// repository's own code may.
	goLine := regexp.MustCompile(`(?m)^go \S+$`).Find(gomod)
	if goLine == nil {
		t.Fatal("go.mod has no go line")
	}
	src := fmt.Sprintf("module example.com/docsnippets\n\n%s\n\n"+
		"require github.com/rohanthewiz/grmob v0.0.0\n\n"+
		"replace github.com/rohanthewiz/grmob => %s\n", goLine, root)
	if err := os.WriteFile(filepath.Join(mod, "go.mod"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if sum, err := os.ReadFile(filepath.Join(root, "go.sum")); err == nil {
		if err := os.WriteFile(filepath.Join(mod, "go.sum"), sum, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// fence is one ```go block: the nearest heading above it, the 1-based line
// of its first body line, and its body with the fence's indentation removed.
type fence struct {
	heading string
	line    int
	body    string
}

// goFences returns every ```go fence in the markdown with the heading it
// sits under. Other fences (```, ```sh) are tracked only so that a "#" line
// inside one is not mistaken for a heading and its closing ``` is not taken
// for the end of a Go block.
//
// A fence may be indented, as the concern snippet is inside a list item.
// Its body is dedented by the opening fence's indentation, which is what a
// markdown renderer strips too.
func goFences(md string) []fence {
	var (
		out     []fence
		heading string
		inFence bool
		isGo    bool
		indent  int
		cur     fence
		body    []string
	)
	for i, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		if !inFence {
			if strings.HasPrefix(trimmed, "```") {
				inFence = true
				isGo = strings.TrimSpace(strings.TrimPrefix(trimmed, "```")) == "go"
				indent = len(line) - len(strings.TrimLeft(line, " "))
				cur = fence{heading: heading, line: i + 2}
				body = body[:0]
				continue
			}
			if strings.HasPrefix(line, "#") {
				heading = trimmed
			}
			continue
		}
		if trimmed == "```" {
			inFence = false
			if isGo {
				cur.body = strings.Join(body, "\n")
				out = append(out, cur)
			}
			continue
		}
		// Strip at most the fence's own indentation, never code indentation.
		strip := min(indent, len(line)-len(strings.TrimLeft(line, " ")))
		body = append(body, line[strip:])
	}
	return out
}

// assemble returns the file name and source for the n-th fence of a doc.
//
// A fence with its own package clause is used as written. One without gets a
// package clause and an import block for the qualifiers it uses. Either way
// the body is introduced by a //line directive, so diagnostics name the
// markdown. A fence that uses testing goes in a _test.go file, since it is
// either a test or a helper only tests call.
func assemble(d doc, s section, f fence, absDoc string, n int) (string, string) {
	quals := qualifiers(f.body)
	name := fmt.Sprintf("fence%02d.go", n)
	if quals["testing"] {
		name = fmt.Sprintf("fence%02d_test.go", n)
	}
	directive := fmt.Sprintf("//line %s:%d\n", absDoc, f.line)

	if hasPackageClause(f.body) {
		return name, directive + f.body + "\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", d.pkg)
	var paths []string
	for q := range quals {
		paths = append(paths, knownImports[q])
	}
	sort.Strings(paths)
	if len(paths) > 0 {
		b.WriteString("import (\n")
		for _, p := range paths {
			fmt.Fprintf(&b, "\t%q\n", p)
		}
		b.WriteString(")\n\n")
	}
	if s.wrap != "" {
		fmt.Fprintf(&b, s.wrap, directive+f.body)
	} else {
		b.WriteString(directive)
		b.WriteString(f.body)
		b.WriteString("\n")
	}
	return name, b.String()
}

// qualifiers reports which knownImports names the source uses as a package
// qualifier: an identifier immediately followed by a dot. The scanner skips
// comments, so "// core.AuditTree reports …" in a fence does not import core.
func qualifiers(src string) map[string]bool {
	found := map[string]bool{}
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, []byte(src), nil, 0)
	prevIdent := ""
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			return found
		}
		if tok == token.PERIOD && prevIdent != "" {
			if _, ok := knownImports[prevIdent]; ok {
				found[prevIdent] = true
			}
		}
		prevIdent = ""
		if tok == token.IDENT {
			prevIdent = lit
		}
	}
}

// hasPackageClause reports whether the first token of the source is
// "package".
func hasPackageClause(src string) bool {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, []byte(src), nil, 0)
	_, tok, _ := s.Scan()
	return tok == token.PACKAGE
}
