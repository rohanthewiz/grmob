// Command themehistory reads how core.Theme's leaf population has moved,
// commit by commit, and prints the distribution of edit sizes.
//
// # Why this is a program and not a test
//
// wasm/verify/themenearmiss_test.go measures a band for each step in
// affordedBandSteps — every population one leaf from the record's, every
// population two from it, three — and stops at three because four costs
// 685ms. That bound used to be argued from a guess ("the fourth simultaneous
// field edit is a rarer thing than the third"), and the argument it has now is
// a distribution: how often core.Theme has actually moved one, two, three or
// more leaves in a single commit. Both halves of the guess turned out to be
// wrong.
//
// The reading was taken by a throwaway AST walker in a scratch directory, and
// what survived into the test was the numbers, the method in a paragraph, and
// the commit it was taken at. That is weaker than the convention the rest of
// that file has moved to: the record's census is re-walked on every run
// precisely because a number in a note is a number that has already moved. A
// test cannot shell out to git reliably — a shallow clone, a source tarball or
// a build container has no history to read — so the reading cannot become an
// arm. It can stop being a re-derivation:
//
//	go run ./internal/themehistory
//	go run ./internal/themehistory -names   (and the population at HEAD)
//
// # What it reads
//
// For every commit that touches core/, the package's sources at that revision
// are parsed and Theme is expanded to its distinct leaf NAMES — the same
// population wasm/verify's affordedLeafNames produces, by the same rule
// (recurse into struct-typed fields, take the last dotted segment, keep each
// name once). Each commit is then diffed against its predecessor.
//
// The expansion is syntactic and affordedLeafNames' is reflective, which is
// the one place the two could disagree: a field whose type this walker cannot
// resolve to a struct declared in core/ is treated as a leaf, where reflect
// would have recursed into it. That is not hypothetical — the first version
// of this walker stopped at Typography.Caption because it had already
// descended through Typography.Body and both are TextStyle, and it came back
// with ninety-one names rather than eighty (see walk's `next`). So the tool
// prints its own reading of HEAD, and a reader checks it against
// affordedMeasuredOn before believing anything else it says.
//
// The counts agreeing is the weak form of that check: two different sets of
// eighty print the same 80. -names prints the population itself, which is the
// form worth running — it is name-for-name identical to affordedLeafNames'
// today.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// The directory whose commits are read and whose sources are parsed. The
// struct is core.Theme and nothing outside that package contributes a field to
// it, so a commit that does not touch core/ cannot move the population.
const themePkg = "core"

// The struct the population is an expansion of.
const themeType = "Theme"

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "themehistory:", err)
		os.Exit(1)
	}
}

// Whether to print the leaf names at HEAD. The counts below are only worth
// reading while this walker's population is the one wasm/verify measures, and
// the count agreeing is weaker evidence than the names agreeing: two
// different sets of eighty would print the same 80.
var showNames = flag.Bool("names", false,
	"also print the distinct leaf names at HEAD, one per line")

func run() error {
	// Oldest first, so a commit is diffed against the state before it. --follow
	// is deliberately not used: it is a per-file heuristic and this is a
	// directory.
	out, err := git("log", "--reverse", "--format=%H %ad", "--date=short",
		"--", themePkg)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return fmt.Errorf("no commits touch %s/", themePkg)
	}

	// One commit that moved the population, with what it moved.
	type step struct {
		sha, date      string
		added, removed []string
		// The size AFTER this commit, so the listing reads as a running total
		// and the last row is HEAD.
		leaves int
	}
	var steps []step
	var prev map[string]bool
	for _, line := range lines {
		sha, date, _ := strings.Cut(line, " ")
		names, err := leavesAt(sha)
		if err != nil {
			return fmt.Errorf("%s: %w", sha[:8], err)
		}
		// A commit that touched core/ without moving the population — a
		// comment, a method, a rename of something that is not a leaf — is not
		// a step. Only the commits that MOVED it are what the distribution is
		// over.
		var added, removed []string
		for name := range names {
			if !prev[name] {
				added = append(added, name)
			}
		}
		for name := range prev {
			if !names[name] {
				removed = append(removed, name)
			}
		}
		sort.Strings(added)
		sort.Strings(removed)
		if len(added) > 0 || len(removed) > 0 {
			steps = append(steps, step{sha: sha, date: date, added: added,
				removed: removed, leaves: len(names)})
		}
		prev = names
	}

	// The per-commit reading, which is what makes the histogram checkable.
	fmt.Printf("%d commit(s) touching %s/ moved %s.%s's leaf population:\n\n",
		len(steps), themePkg, themePkg, themeType)
	for i, s := range steps {
		what := fmt.Sprintf("+%d", len(s.added))
		if len(s.removed) > 0 {
			what += fmt.Sprintf(" -%d", len(s.removed))
		}
		moved := append(append([]string{}, s.added...), s.removed...)
		note := ""
		if i == 0 {
			note = "  (the commit that created it)"
		}
		fmt.Printf("  %s  %s  %-7s -> %3d leaves%s\n", s.sha[:8], s.date, what,
			s.leaves, note)
		fmt.Printf("            %s\n", wrap(moved, 66, "            "))
	}

	// And the distribution the band steps are argued from. The bootstrap
	// commit is counted like any other and marked, because "28 leaves at once"
	// is a real row and also obviously not a field edit.
	fmt.Printf("\nleaves moved   commits\n")
	by := map[int]int{}
	for _, s := range steps {
		by[len(s.added)+len(s.removed)]++
	}
	sizes := make([]int, 0, len(by))
	for n := range by {
		sizes = append(sizes, n)
	}
	sort.Ints(sizes)
	for _, n := range sizes {
		fmt.Printf("%12d   %7d\n", n, by[n])
	}

	// The direction, which is a finding of its own: affordedBand measures a
	// `losing` number — a population read against a LARGER one — and if
	// nothing has ever removed a leaf, that half of every band is over a
	// direction the struct has never gone. Counted as commits and not as
	// leaves, because one commit that removed thirty would still be one
	// occasion on which anybody did it.
	grew, shrank := 0, 0
	for _, s := range steps {
		if len(s.added) > 0 {
			grew++
		}
		if len(s.removed) > 0 {
			shrank++
		}
	}
	fmt.Printf("\n%d of the %d added leaves and %d removed any.\n",
		grew, len(steps), shrank)

	if *showNames {
		out := []string{}
		for n := range prev {
			out = append(out, n)
		}
		sort.Strings(out)
		fmt.Println(strings.Join(out, "\n"))
	}
	// And HEAD, which is what says the reading is measuring the population the
	// test measures. Compare against affordedMeasuredOn's `leaves`.
	if n := len(prev); n > 0 {
		fmt.Printf("HEAD expands to %d distinct leaf names — compare "+
			"affordedMeasuredOn.leaves in wasm/verify/themenearmiss_test.go.\n", n)
	}
	return nil
}

// leavesAt is the distinct leaf names of core.Theme at one revision.
//
// The same rule affordedLeafNames uses over reflect: recurse into a field
// whose type is a struct declared in this package, take the last segment of
// the dotted path, and keep each name once however many parents hold it.
func leavesAt(sha string) (map[string]bool, error) {
	files, err := git("ls-tree", "-r", "--name-only", sha, "--", themePkg)
	if err != nil {
		return nil, err
	}
	structs := map[string]*ast.StructType{}
	fset := token.NewFileSet()
	for _, path := range strings.Split(strings.TrimSpace(files), "\n") {
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := git("cat-file", "-p", sha+":"+path)
		if err != nil {
			return nil, err
		}
		// Errors are not fatal: a revision mid-refactor can hold a file that
		// does not parse, and go/parser returns the declarations it did manage.
		// A struct missing from a partial parse shows up as a leaf count that
		// jumps, which is visible in the per-commit listing.
		f, _ := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if f == nil {
			continue
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if st, ok := ts.Type.(*ast.StructType); ok {
					structs[ts.Name.Name] = st
				}
			}
		}
	}
	names := map[string]bool{}
	// Guarded against a struct that holds itself, directly or through another:
	// reflect cannot build such a value so affordedLeafNames never meets one,
	// and this walker would recurse forever.
	var walk func(st *ast.StructType, seen map[string]bool)
	walk = func(st *ast.StructType, seen map[string]bool) {
		if st == nil {
			return
		}
		for _, field := range st.Fields.List {
			// An embedded field has no name and its type is the name.
			fieldNames := []string{}
			for _, n := range field.Names {
				fieldNames = append(fieldNames, n.Name)
			}
			if len(fieldNames) == 0 {
				if id, ok := unwrap(field.Type).(*ast.Ident); ok {
					fieldNames = append(fieldNames, id.Name)
				}
			}
			// The set the CHILD descends with, which is this one plus the
			// type being entered. Kept separate from `seen` because two
			// sibling fields of the same struct type are not recursion:
			// Typography holds Body, Caption and Subtitle, all TextStyle, and
			// a `seen` that grew as the loop ran would descend into the first
			// and treat the other two as leaves.
			inner, next := (*ast.StructType)(nil), seen
			switch t := unwrap(field.Type).(type) {
			case *ast.Ident:
				if st, ok := structs[t.Name]; ok && !seen[t.Name] {
					inner, next = st, withName(seen, t.Name)
				}
			case *ast.StructType:
				inner = t
			}
			for _, name := range fieldNames {
				if inner != nil {
					walk(inner, next)
					continue
				}
				names[name] = true
			}
		}
	}
	walk(structs[themeType], map[string]bool{themeType: true})
	return names, nil
}

// unwrap strips the type constructors a struct-typed field can be wrapped in
// without stopping being one for reflect's purposes. A pointer is NOT one:
// themeLeafPaths recurses on reflect.Struct only, so a *T field is a leaf
// there and has to be a leaf here.
func unwrap(t ast.Expr) ast.Expr {
	if p, ok := t.(*ast.ParenExpr); ok {
		return unwrap(p.X)
	}
	return t
}

// withName is `seen` plus one name, copied so that two sibling fields of the
// same struct type do not see each other's descent as recursion.
func withName(seen map[string]bool, name string) map[string]bool {
	out := make(map[string]bool, len(seen)+1)
	for k := range seen {
		out[k] = true
	}
	out[name] = true
	return out
}

// wrap is a list of names as indented lines, so a 28-name commit does not
// print as one unreadable row.
func wrap(names []string, width int, indent string) string {
	var b strings.Builder
	line := 0
	for _, name := range names {
		if line > 0 && line+len(name)+2 > width {
			b.WriteString("\n" + indent)
			line = 0
		}
		if line > 0 {
			b.WriteString(", ")
			line += 2
		}
		b.WriteString(name)
		line += len(name)
	}
	return b.String()
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err,
			strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}
