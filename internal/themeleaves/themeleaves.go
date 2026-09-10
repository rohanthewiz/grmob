// Package themeleaves expands a struct declared in a package's SOURCES to the
// distinct names of its leaf fields.
//
// # What this is for, and why it is not simply reflect
//
// wasm/verify's affordedLeafNames produces the same population — core.Theme's
// distinct leaf names — over reflect, and everything that test measures is
// measured over it. This walker exists because one reader of that population
// cannot use reflect at all: internal/themehistory asks what the population
// was at a git revision, and a revision is source text. There is no value to
// reflect over without building the code.
//
// So there are two expansions of one struct by two mechanisms, and the whole
// worth of the history table rests on them agreeing. They can disagree, and
// the disagreement is one-sided: this one resolves a field's type by NAME
// against the struct declarations it parsed, so a field whose type it cannot
// find is a leaf here where reflect would have recursed into it. A type
// declared in another package, a generic instantiation, an alias chain — all
// of them stop this walk early and none of them stop reflect's.
//
// That is not hypothetical. The first version of this code came back with
// NINETY-ONE names rather than eighty, because its recursion guard was
// reassigned inside the field loop: it descended through Typography.Body,
// marked TextStyle seen, and then treated Caption and Subtitle — the same
// type — as leaves. Eleven names too many, in a walker whose output was about
// to become a table nobody would re-derive.
//
//	Typography ──┬── Body      TextStyle ──┬── Size    ← descends, marks
//	             │                         ├── Weight    TextStyle seen
//	             │                         └── ...
//	             ├── Caption   TextStyle ── (seen!) ← WRONG: "Caption" is a leaf
//	             └── Subtitle  TextStyle ── (seen!) ← WRONG: "Subtitle" too
//
// The guard has to be the path from the root to the field being entered, not
// the set of types the loop has entered so far: two SIBLING fields of the same
// struct type are not recursion. See walk's `next`.
//
// # Which is why the expansion is a package and the git reading is not
//
// A test cannot shell out to git — a shallow clone, a source tarball or a
// build container has no history to read — so the history table stays a
// command. But the expansion underneath it does not need git at all: given a
// working tree it is a pure parse of a directory, and
// TestTheHistoryWalkersExpansionIsTheOneThisFileMeasures in wasm/verify holds
// InDir("../../core", "Theme") against affordedLeafNames() on every run. The
// half that can be an arm is the half that already went wrong once.
package themeleaves

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Expansion is one reading of a struct's leaf population.
//
// Unparsed and Found are returned rather than folded into an error because the
// two callers want opposite things from them. themehistory walks revisions,
// and a revision mid-refactor can hold a file that does not parse or can
// predate the struct entirely — both are ordinary there, and what it wants is
// the names go/parser did manage. A test over the working tree wants either of
// those to be a failure naming the file. Handing back the fact lets each
// decide.
type Expansion struct {
	// The distinct leaf names, sorted. Distinct because two parents can hold
	// the same leaf name (Colors.Surface and Colors.Overlay.Surface), and the
	// population this is a reading of counts each name once — see
	// affordedLeafNames, which deduplicates for the same reason.
	Names []string
	// The files go/parser returned nothing for. A struct declared only in one
	// of these is missing from the walk, which shows up as a leaf count that
	// jumps rather than as an error.
	Unparsed []string
	// Whether the root struct was declared at all in what was parsed. False
	// with an empty Names is "this revision has no such type"; false is never
	// the same finding as "it has no fields".
	Found bool
}

// Of expands the struct named root, declared somewhere in sources, to its
// distinct leaf names.
//
// sources is path → file text, and only .go files that are not _test.go
// contribute — a test file can declare a struct of the same name in the same
// package, and reflect over the built package would never see it.
//
// The rule is affordedLeafNames' rule over reflect: recurse into a field whose
// type is a struct declared in this package, take the last segment of the
// dotted path, and keep each name once however many parents hold it.
func Of(sources map[string]string, root string) Expansion {
	structs := map[string]*ast.StructType{}
	var unparsed []string
	fset := token.NewFileSet()
	paths := make([]string, 0, len(sources))
	for path := range sources {
		paths = append(paths, path)
	}
	// Sorted so two runs over the same sources build `structs` in one order.
	// It only matters where a package declares one name twice — which does not
	// compile — but a walker whose answer depends on map iteration is a walker
	// whose disagreement with reflect is unreproducible.
	sort.Strings(paths)
	for _, path := range paths {
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			continue
		}
		// Errors are not fatal here: go/parser returns the declarations it did
		// manage, and which files failed goes back to the caller.
		f, _ := parser.ParseFile(fset, path, sources[path],
			parser.SkipObjectResolution)
		if f == nil {
			unparsed = append(unparsed, path)
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
			// The set the CHILD descends with, which is this one plus the type
			// being entered. Kept separate from `seen` because two sibling
			// fields of the same struct type are not recursion — the
			// ninety-one names in this package's doc comment are what a `seen`
			// that grew as the loop ran produces.
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
	st, found := structs[root]
	walk(st, map[string]bool{root: true})

	out := make([]string, 0, len(names))
	for name := range names {
		out = append(out, name)
	}
	sort.Strings(out)
	return Expansion{Names: out, Unparsed: unparsed, Found: found}
}

// InDir is Of over the .go files of one directory on disk.
//
// The working-tree form of the reading, which is the form a test can take: no
// git, no revision, just the sources that are there. Not recursive — the
// struct and everything it holds are declared in one package, and a
// subdirectory is a different one whose types this walk could not resolve
// anyway.
func InDir(dir, root string) (Expansion, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Expansion{}, err
	}
	sources := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			return Expansion{}, fmt.Errorf("%s: %w", path, err)
		}
		sources[path] = string(src)
	}
	return Of(sources, root), nil
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
