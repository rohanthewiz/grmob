package apidoc

import (
	"bytes"
	"go/build"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The generated reference is a committed artifact, and a committed artifact has
// exactly one failure mode worth a test: it can disagree with the thing it was
// derived from. Everything here is a guard against a page that renders
// perfectly and says something untrue —
//
//	stale          a declaration changed and nobody reran the generator
//	dead anchor    a link points at an id no heading produces
//	shifted anchor two headings slug the same, so goldmark renames one and
//	               every link aimed at it silently lands on the other
//	unlisted       a page exists that the nav does not reach
//	undocumented   a public package exists that the reference does not cover
//	invisible      a build-constrained file drops declarations off a page
//
// None of the six is visible by looking at the rendered site, which is what
// makes them worth a test rather than a review.

func root(t *testing.T) string {
	t.Helper()
	r, err := RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestGeneratedPagesAreUpToDate is the staleness gate: it runs the generator in
// memory and compares the result with what is committed under docs/api/.
func TestGeneratedPagesAreUpToDate(t *testing.T) {
	r := root(t)

	pages, err := Generate(r)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	for name, want := range pages {
		path := filepath.Join(r, "docs", filepath.FromSlash(name))
		got, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("docs/%s is missing — run `go run ./internal/apidoc/gen`", name)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("docs/%s is stale (%d bytes committed, %d generated) — "+
				"run `go run ./internal/apidoc/gen`", name, len(got), len(want))
		}
	}

	// The other direction: a page on disk that the generator no longer writes.
	// gen prunes these, so one surviving means gen has not been run since a
	// package left Packages.
	apiDir := filepath.Join(r, "docs", DocsSubdir)
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		t.Fatalf("read %s: %v", apiDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		if _, ok := pages[DocsSubdir+"/"+e.Name()]; !ok {
			t.Errorf("docs/%s/%s is orphaned — run `go run ./internal/apidoc/gen`",
				DocsSubdir, e.Name())
		}
	}
}

// headingRe matches an ATX heading, capturing its level and its text. Setext
// headings are not used by the generator, and a fenced block's contents are
// skipped by the caller.
var headingRe = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)

// heading is one heading of a rendered page, with the id goldmark will give it.
type heading struct {
	Level  int
	Text   string
	ID     string // the id actually assigned, duplicate counter included
	Wanted string // the id the text alone would produce
}

// headings returns a page's headings in order, each with the id goldmark will
// assign it — duplicate counter included, so a heading whose id was taken by an
// earlier one shows up here as the "-1" it really becomes rather than as the id
// it was aiming for.
func headings(md string) []heading {
	var out []heading
	seen := map[string]bool{}
	inFence := false

	for line := range strings.SplitSeq(md, "\n") {
		// Both fence flavours, and an opening fence may carry an info string
		// ("```go"), so only the prefix is matched.
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		m := headingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		h := heading{Level: len(m[1]), Text: m[2]}
		h.Wanted = slug(h.Text)
		h.ID = h.Wanted
		for i := 1; seen[h.ID]; i++ {
			h.ID = h.Wanted + "-" + strconv.Itoa(i)
		}
		seen[h.ID] = true
		out = append(out, h)
	}
	return out
}

// structural reports whether a heading is one the generator writes and links
// to, as opposed to prose lifted out of a doc comment.
//
// Prose headings are allowed to repeat — two types can both document an
// "Accessibility" section, and goldmark renaming the second one costs nothing
// because nothing links to either. A structural heading repeating is a
// different matter, and so is a prose heading landing on an id a structural one
// wanted; both are what TestLinkedAnchorsAreNotStolen is looking for.
func structural(text string) bool {
	switch text {
	case "Index", "Constants", "Variables", "Functions", "Types":
		return true
	}
	return strings.HasPrefix(text, "type ") || strings.HasPrefix(text, "func ")
}

// TestLinkedAnchorsAreNotStolen is the guard slug() deliberately does not
// implement.
//
// Two headings that slug alike are not an error to goldmark: it appends "-1" to
// the second and renders both. The cost falls entirely on the links, which keep
// pointing at the un-suffixed id and so quietly arrive at the wrong section —
// a page that looks right and navigates wrong. Every id the generator links to
// belongs to a structural heading, so the property worth asserting is that each
// structural heading kept the id its own text produces.
func TestLinkedAnchorsAreNotStolen(t *testing.T) {
	pages, err := Generate(root(t))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	for name, body := range pages {
		for _, h := range headings(string(body)) {
			if structural(h.Text) && h.ID != h.Wanted {
				t.Errorf("docs/%s: heading %q should anchor at #%s but an "+
					"earlier heading took that id, so it became #%s",
					name, h.Text, h.Wanted, h.ID)
			}
		}
	}
}

// linkRe matches a markdown inline link's destination.
var linkRe = regexp.MustCompile(`\]\(([^)\s]+)\)`)

// TestEveryGeneratedLinkResolves follows every link the generator writes that
// stays inside docs/api/, and checks that its page exists and that its fragment
// is an id some heading on that page produces.
//
// Links out of the directory (../concepts/..., pkg.go.dev, the source links on
// every declaration) are not followed: the first are checked by the docs link
// walk, and the last two point off this site.
func TestEveryGeneratedLinkResolves(t *testing.T) {
	pages, err := Generate(root(t))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	anchors := map[string]map[string]bool{}
	for name, body := range pages {
		set := map[string]bool{}
		for _, h := range headings(string(body)) {
			set[h.ID] = true
		}
		anchors[name] = set
	}

	for name, body := range pages {
		for _, m := range linkRe.FindAllStringSubmatch(string(body), -1) {
			dest := m[1]
			if strings.Contains(dest, "://") || strings.HasPrefix(dest, "../") {
				continue
			}

			page, frag := name, ""
			if i := strings.IndexByte(dest, '#'); i >= 0 {
				frag = dest[i+1:]
				dest = dest[:i]
			}
			if dest != "" {
				page = DocsSubdir + "/" + dest
				if _, ok := anchors[page]; !ok {
					t.Errorf("docs/%s: link to %q, which is not a generated page", name, dest)
					continue
				}
			}
			if frag != "" && !anchors[page][frag] {
				t.Errorf("docs/%s: link to %q#%s — no heading on that page has that anchor",
					name, dest, frag)
			}
		}
	}
}

// TestNavListsEveryPage checks that mkdocs.yml reaches every generated page.
//
// A page absent from the nav is still served — gkdocs resolves any path under
// docs_dir — but nothing links to it, so it is documentation that exists and
// cannot be found, which is the failure a reference site can least afford.
func TestNavListsEveryPage(t *testing.T) {
	r := root(t)

	cfg, err := os.ReadFile(filepath.Join(r, "mkdocs.yml"))
	if err != nil {
		t.Fatal(err)
	}
	pages, err := Generate(r)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	for name := range pages {
		if !bytes.Contains(cfg, []byte(name)) {
			t.Errorf("mkdocs.yml nav does not mention %q", name)
		}
	}
}

// TestPackagesCoversEveryPublicPackage walks the tree and fails on an
// importable, non-main package that the reference neither documents nor
// deliberately excludes. It is what keeps the hand-kept Packages list from
// quietly falling behind the module.
func TestPackagesCoversEveryPublicPackage(t *testing.T) {
	r := root(t)

	documented := map[string]bool{}
	for _, p := range Packages {
		documented[p.Dir] = true
	}

	err := filepath.WalkDir(r, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		name := d.Name()
		if path != r && (strings.HasPrefix(name, ".") || name == "testdata" || name == "node_modules") {
			return filepath.SkipDir
		}

		rel, err := filepath.Rel(r, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		// Exclusions are tested per directory rather than pruned as subtrees:
		// "wasm" is excluded but a documented package could one day live under
		// it, and SkipDir would hide that.
		if rel == "." || excluded(rel) {
			return nil
		}

		bp, err := build.ImportDir(path, 0)
		if err != nil {
			return nil // no buildable Go files here
		}
		if bp.Name == "main" || len(bp.GoFiles) == 0 {
			return nil
		}
		if !documented[rel] {
			t.Errorf("package %s is importable and undocumented: add it to "+
				"apidoc.Packages, or to the exclusion rule in excluded()", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// excluded states, in one place, which directories the reference leaves out and
// why. The reasons are in Packages' doc comment; this is the executable form.
func excluded(rel string) bool {
	switch {
	case rel == "internal" || strings.HasPrefix(rel, "internal/"):
		return true // not importable by construction
	case rel == "examples" || strings.HasPrefix(rel, "examples/"):
		return true // programs, documented as prose by the tutorial
	case rel == "cmd" || strings.HasPrefix(rel, "cmd/"):
		return true // the docs server, a module of its own
	case strings.HasSuffix(rel, "/verify") || strings.HasSuffix(rel, "/gen"):
		return true // conformance harnesses and generators
	case rel == "android" || rel == "ios" || rel == "wasm" || rel == "serve":
		return true // native shells and package main
	case rel == "webhost":
		// Importable, but everything it does is behind //go:build js && wasm,
		// which load reads against the host GOOS and would render as a page
		// holding one variable. Documented in docs/platforms/wasm.md instead.
		return true
	case strings.HasPrefix(rel, "wasm/"):
		return true // the JS runtime's harness and its shot host
	case rel == "docs" || strings.HasPrefix(rel, "docs/"):
		return true
	case rel == "ai_docs" || strings.HasPrefix(rel, "ai_docs/"):
		return true
	}
	return false
}

// TestNoBuildConstraintsInDocumentedPackages guards the one assumption load
// makes that could cost a page a declaration without any visible sign.
//
// Files are collected through go/build against the host GOOS and GOARCH, so a
// file behind a //go:build line that the host does not satisfy is simply not
// read, and whatever it declared is absent from the page. No documented package
// has such a file today. Should one appear, this fails and names it, instead of
// the reference losing a function on whichever machine ran the generator.
func TestNoBuildConstraintsInDocumentedPackages(t *testing.T) {
	r := root(t)

	for _, p := range Packages {
		dir := filepath.Join(r, filepath.FromSlash(p.Dir))
		bp, err := build.ImportDir(dir, 0)
		if err != nil {
			t.Fatalf("%s: %v", p.Dir, err)
		}
		for _, f := range bp.IgnoredGoFiles {
			// A leading "_" or "." also lands here, and is equally invisible.
			t.Errorf("%s/%s is excluded by a build constraint, so its "+
				"declarations are missing from docs/%s/%s",
				p.Dir, f, DocsSubdir, p.Page())
		}
	}
}

// TestEveryDocumentedPackageHasAPackageComment. The page's opening paragraph is
// the package's doc comment, and a package without one opens on the import line
// and an index — a reference that says what is there and never what it is for.
func TestEveryDocumentedPackageHasAPackageComment(t *testing.T) {
	pkgs, err := LoadAll(root(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range pkgs {
		if strings.TrimSpace(l.Doc.Doc) == "" {
			t.Errorf("package %s has no package comment", l.Pkg.Dir)
		}
	}
}
