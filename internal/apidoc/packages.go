package apidoc

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ModulePath is grmob's module path. Import paths on the generated pages are
// built from it, and repoRoot uses it to tell the module root apart from the
// nested module under cmd/docs.
const ModulePath = "github.com/rohanthewiz/grmob"

// SourceBaseURL is where a "declared in" line points. It is pinned to a branch
// rather than a tag or a commit on purpose: the pages are regenerated from the
// working tree and committed alongside it, so a line number is only ever
// correct for the same revision the pages were built from, and master is the
// revision a reader arriving from the docs site is looking at.
const SourceBaseURL = "https://github.com/rohanthewiz/grmob/blob/master/"

// Pkg is one documented package: where its source lives, what the nav calls
// it, and which group it sits in on the overview page.
type Pkg struct {
	// Dir is the package directory relative to the module root ("core",
	// "aria/spec"). It is also the key everything else is derived from: the
	// import path, the page filename, the nav entry.
	Dir string

	// Group is the overview page's section heading. Packages with the same
	// Group are listed together, in the order they appear in Packages.
	Group string

	// Blurb is one line of editorial orientation shown on the overview page
	// next to the package name. The package's own doc comment is the authority
	// on what the package *is*; this says when you reach for it, which is the
	// question an index answers and a doc comment does not.
	Blurb string
}

// ImportPath is the path a caller writes in an import statement.
func (p Pkg) ImportPath() string { return ModulePath + "/" + p.Dir }

// Name is the package's clause name, which for every package here is the last
// element of its directory. Nothing in grmob's public surface renames a package
// away from its directory, and the loader verifies that rather than trusting it
// (see load).
func (p Pkg) Name() string { return path.Base(p.Dir) }

// Page is the generated file's name under docs/api/. Nested directories are
// flattened with a dash ("aria/spec" -> "aria-spec.md") so every page is a
// sibling of every other one, which is what lets an in-page cross-reference be
// a bare relative link with no "../" to get wrong.
func (p Pkg) Page() string { return strings.ReplaceAll(p.Dir, "/", "-") + ".md" }

// Packages is the documented public surface, in nav order.
//
// It is a hand-kept list rather than a walk of the tree, because "every package
// that compiles" is the wrong set and the difference is not mechanical. Left out
// deliberately:
//
//	internal/...        not importable, by construction
//	examples/...        programs, and the tutorial's lessons are prose already
//	*/verify, aria/gen  test harnesses and generators — developer tooling whose
//	                    audience reads the source, not a reference page
//	serve, wasm         package main
//
// TestPackagesCoversEveryPublicPackage holds the list to that rule: it walks
// the tree and fails if an importable non-main package appears that is neither
// listed here nor matched by one of the exclusions above. A new public package
// therefore cannot be added without either documenting it or saying, in that
// test's terms, why not.
var Packages = []Pkg{
	{
		Dir:   "core",
		Group: "Core",
		Blurb: "Views, nodes, state, styling, events — everything an app builds its UI out of.",
	},
	{
		Dir:   "hooks",
		Group: "Core",
		Blurb: "Effects, timers, memos, reducers and the other hooks layered on core's state slots.",
	},
	{
		Dir:   "render",
		Group: "Rendering",
		Blurb: "The render loop: mount a root view, dispatch host events, drain pushed updates.",
	},
	{
		Dir:   "reconcile",
		Group: "Rendering",
		Blurb: "The tree diff and the patch vocabulary every host applies.",
	},
	{
		Dir:   "comps",
		Group: "Widgets",
		Blurb: "The widget library — cards, tabs, accordions and friends, built on the public core API.",
	},
	{
		Dir:   "forms",
		Group: "Widgets",
		Blurb: "Validation rules, the form hook that owns values and error visibility, and bound inputs.",
	},
	{
		Dir:   "richtext",
		Group: "Widgets",
		Blurb: "The document model behind core.RichTextEditor: a formatted document as Go values.",
	},
	{
		Dir:   "highlight",
		Group: "Widgets",
		Blurb: "Go syntax highlighting, for the code editor and the tutorial's listings.",
	},
	{
		Dir:   "htmlout",
		Group: "Exporters",
		Blurb: "Render a node tree to a standalone HTML document.",
	},
	{
		Dir:   "jsonout",
		Group: "Exporters",
		Blurb: "Render a node tree to JSON, for host renderers and for inspecting a tree in a test.",
	},
	{
		Dir:   "mobile",
		Group: "Platform",
		Blurb: "The gomobile-bindable bridge the Android and iOS shells call into.",
	},
	{
		Dir:   "permission",
		Group: "Platform",
		Blurb: "Asking the platform for the camera, the microphone, location, the media store.",
	},
	{
		Dir:   "aria/spec",
		Group: "Tools",
		Blurb: "Reads the W3C ARIA specification's own role definitions; feeds the accessibility fixture.",
	},
}

// Loaded is one package's parsed documentation, plus the fileset its positions
// are relative to.
type Loaded struct {
	Pkg  Pkg
	Doc  *doc.Package
	FSet *token.FileSet
}

// load parses one package and builds its documentation.
//
// The file list comes from go/build rather than from a directory listing so
// that build constraints and the _test.go suffix are honoured by the same rules
// the compiler uses. It resolves against build.Default, i.e. the host GOOS and
// GOARCH, which is exact for grmob because no package in Packages has a single
// constrained file — the only //go:build lines in the tree are in wasm's
// package main and in tests. Should that ever change, the omission would be
// silent here, so TestNoBuildConstraintsInDocumentedPackages fails on the first
// constrained file to appear in a documented package rather than letting a page
// quietly lose a declaration.
func load(root string, p Pkg) (*Loaded, error) {
	dir := filepath.Join(root, filepath.FromSlash(p.Dir))

	bp, err := build.ImportDir(dir, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p.Dir, err)
	}
	if bp.Name != p.Name() {
		// The generator derives the page title and every "package x" mention
		// from the directory; a package whose clause disagrees would be
		// documented under a name no import statement produces.
		return nil, fmt.Errorf("%s: package clause is %q, want %q",
			p.Dir, bp.Name, p.Name())
	}

	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(bp.GoFiles))
	for _, name := range bp.GoFiles {
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.Dir, err)
		}
		files = append(files, f)
	}

	// Default mode: unexported declarations are dropped, which is the whole
	// point — this is a reference for callers. doc.NewFromFiles also does the
	// grouping a reader expects, attaching each constructor and method to the
	// type it belongs to instead of listing it among the package functions.
	dp, err := doc.NewFromFiles(fset, files, p.ImportPath())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p.Dir, err)
	}

	return &Loaded{Pkg: p, Doc: dp, FSet: fset}, nil
}

// LoadAll parses every documented package.
func LoadAll(root string) ([]*Loaded, error) {
	out := make([]*Loaded, 0, len(Packages))
	for _, p := range Packages {
		l, err := load(root, p)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

// RepoRoot walks up from the working directory to grmob's module root.
//
// It cannot stop at the first go.mod it finds, the way a single-module
// repository's generator can: cmd/docs is a module of its own (so that the docs
// server's dependencies stay out of the framework's go.mod), and a run started
// from inside it would otherwise take that directory for the root. So the
// module line has to match, and the walk continues past a go.mod that belongs to
// somebody else.
func RepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	start := dir
	for {
		if isModuleRoot(filepath.Join(dir, "go.mod")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod declaring %s above %s — "+
				"run this from inside the repository", ModulePath, start)
		}
		dir = parent
	}
}

func isModuleRoot(goMod string) bool {
	data, err := os.ReadFile(goMod)
	if err != nil {
		return false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.TrimSpace(line) == "module "+ModulePath {
			return true
		}
	}
	return false
}

// symbolIndex maps an import path to the anchors of that package's exported
// symbols, keyed by the name a doc link would use ("Node", "Node.Clone").
//
// It exists because a doc link does not say what kind of thing it points at.
// `[Node]` arrives as a bare name, and the anchor for a type is "type-node"
// while the anchor for a function of the same name is "func-node" — so
// resolving a link needs to know which one the target package actually declares.
// Building the index over every loaded package first, then rendering, is what
// makes a link from core's doc comment into components resolvable at all.
type symbolIndex map[string]map[string]string

func newSymbolIndex(pkgs []*Loaded) symbolIndex {
	idx := make(symbolIndex, len(pkgs))
	for _, l := range pkgs {
		syms := map[string]string{}
		for _, f := range l.Doc.Funcs {
			syms[f.Name] = symAnchor("func", "", f.Name)
		}
		for _, t := range l.Doc.Types {
			syms[t.Name] = symAnchor("type", "", t.Name)
			for _, f := range t.Funcs { // constructors: documented under the type
				syms[f.Name] = symAnchor("func", "", f.Name)
			}
			for _, m := range t.Methods {
				syms[t.Name+"."+m.Name] = symAnchor("func", t.Name, m.Name)
			}
		}
		for _, v := range l.Doc.Consts {
			for _, name := range v.Names {
				if ast.IsExported(name) {
					syms[name] = "constants"
				}
			}
		}
		for _, v := range l.Doc.Vars {
			for _, name := range v.Names {
				if ast.IsExported(name) {
					syms[name] = "variables"
				}
			}
		}
		idx[l.Pkg.ImportPath()] = syms
	}
	return idx
}

// sortedGroups returns the overview page's group headings in the order their
// first package appears in Packages, so the nav and the overview agree.
func sortedGroups() []string {
	var groups []string
	seen := map[string]bool{}
	for _, p := range Packages {
		if !seen[p.Group] {
			seen[p.Group] = true
			groups = append(groups, p.Group)
		}
	}
	return groups
}

// stableNames returns names sorted, for the one place go/doc does not sort for
// us and a stable page still matters.
func stableNames(names []string) []string {
	out := append([]string(nil), names...)
	sort.Strings(out)
	return out
}
