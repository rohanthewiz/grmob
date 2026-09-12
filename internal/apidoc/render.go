// Package apidoc renders grmob's public API as mkdocs-style markdown: one page
// per documented package, written from the packages' own doc comments, plus an
// overview page indexing them.
//
// # Why generate instead of write
//
// A reference page is a restatement of a declaration, and a restatement that
// lives in a different file from the thing it restates goes stale on its own
// schedule. The narrative docs under docs/ are written by hand because they say
// things the code cannot; docs/api says only what the code already says, so it
// is derived from the code and committed, in the same shape as every other
// generated artifact in this repository (see aria/gen): the output is checked
// in so that reading the docs needs no toolchain, and a test regenerates it in
// memory and fails when the committed copy has drifted.
//
//	go run ./internal/apidoc/gen        write docs/api/
//	go test ./internal/apidoc           fail if docs/api/ is stale
//
// # Page shape
//
// The layout follows godoc's, bent to fit gkdocs' table of contents, which is
// built from h2 and h3 headings only (h1 is the page title, h4 and deeper are
// judged too noisy). So the levels are spent where they buy the most
// navigation:
//
//	#      package name                the page title
//	##     Index / Constants / ...     the four fixed sections
//	###    type Node, func Text        one per exported symbol — the useful TOC
//	####   func (*Node) Clone          methods, reachable from the page index
//
// That puts every top-level symbol of a package in the sidebar and keeps
// methods off it, which for a package the size of core is the difference
// between a usable list and a wall.
package apidoc

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/printer"
	"go/token"
	"path/filepath"
	"strings"
)

// DocsSubdir is where the generated pages live, relative to the mkdocs
// docs_dir. It is also the prefix mkdocs.yml's nav entries carry, and
// TestNavListsEveryPage checks the two agree.
const DocsSubdir = "api"

// hiddenFieldsMarker is the line godoc puts where it elided a type's
// unexported members.
//
// It reaches the output through a deliberate abuse of go/printer: the marker is
// injected into the filtered field list as a field whose type is an identifier
// whose *name* is the comment text. go/printer writes an identifier's name
// verbatim, so the line comes out as written, on its own line, indented with
// the fields around it. The alternative — attaching a real *ast.CommentGroup —
// does not work, because go/printer only emits comments it can find in the
// comment map of a whole *ast.File, and what is being printed here is a single
// synthesised declaration.
const hiddenFieldsMarker = "// contains filtered or unexported fields"

const hiddenMethodsMarker = "// contains filtered or unexported methods"

// Generate renders every page and returns them keyed by their path relative to
// the mkdocs docs_dir ("api/core.md").
//
// Nothing is written to disk here: the generator command writes the returned
// map, and the staleness test compares it against what is committed. One
// function serving both is what makes the test meaningful — it exercises the
// same code path the command does, not a reimplementation of it.
func Generate(root string) (map[string][]byte, error) {
	pkgs, err := LoadAll(root)
	if err != nil {
		return nil, err
	}

	g := &gen{
		root:   root,
		syms:   newSymbolIndex(pkgs),
		byName: map[string]string{},
		byPath: map[string]Pkg{},
	}
	for _, l := range pkgs {
		g.byName[l.Pkg.Name()] = l.Pkg.ImportPath()
		g.byPath[l.Pkg.ImportPath()] = l.Pkg
	}

	out := make(map[string][]byte, len(pkgs)+1)
	for _, l := range pkgs {
		page, err := g.page(l)
		if err != nil {
			return nil, err
		}
		out[DocsSubdir+"/"+l.Pkg.Page()] = page
	}
	out[DocsSubdir+"/index.md"] = g.overview(pkgs)
	return out, nil
}

// gen carries the whole-run state a single page needs: the module root, so
// source links can be made repo-relative, and the cross-package indexes, so a
// doc link can be resolved to a page and an anchor.
type gen struct {
	root   string
	syms   symbolIndex
	byName map[string]string // package clause name -> import path
	byPath map[string]Pkg    // import path -> package
}

// ---------------------------------------------------------------------------
// Pages
// ---------------------------------------------------------------------------

func (g *gen) page(l *Loaded) ([]byte, error) {
	var b strings.Builder

	p := l.Pkg
	fmt.Fprintf(&b, "# Package %s\n\n", p.Name())
	fmt.Fprintf(&b, "```go\nimport \"%s\"\n```\n\n", p.ImportPath())

	// Level 2, the same as the Index and Types headings below: a heading in a
	// package comment is a section of the page, not a subsection of anything,
	// and h2 is also the level the sidebar shows at full weight.
	if d := strings.TrimSpace(l.Doc.Doc); d != "" {
		b.WriteString(g.comment(l, d, 2))
		b.WriteString("\n")
	}

	b.WriteString(g.index(l))

	g.values(&b, l, "Constants", l.Doc.Consts)
	g.values(&b, l, "Variables", l.Doc.Vars)

	if len(l.Doc.Funcs) > 0 {
		b.WriteString("## Functions\n\n")
		for _, fn := range l.Doc.Funcs {
			g.function(&b, l, fn, 3)
		}
	}

	if len(l.Doc.Types) > 0 {
		b.WriteString("## Types\n\n")
		for _, t := range l.Doc.Types {
			g.typ(&b, l, t)
		}
	}

	return []byte(b.String()), nil
}

// index is the per-page symbol list.
//
// It is plain nested lists under one h2 rather than a heading per kind, because
// every heading here would also land in the sidebar TOC and the sidebar already
// lists these symbols — a second copy of the same names, interleaved with the
// first, is worse than no index at all. What the list adds over the sidebar is
// the methods, which the sidebar stops short of.
func (g *gen) index(l *Loaded) string {
	var b strings.Builder
	b.WriteString("## Index\n\n")

	if n := countNames(l.Doc.Consts); n > 0 {
		fmt.Fprintf(&b, "- [Constants](#constants) — %s\n", strings.Join(codeList(allNames(l.Doc.Consts)), ", "))
	}
	if n := countNames(l.Doc.Vars); n > 0 {
		fmt.Fprintf(&b, "- [Variables](#variables) — %s\n", strings.Join(codeList(allNames(l.Doc.Vars)), ", "))
	}
	for _, fn := range l.Doc.Funcs {
		fmt.Fprintf(&b, "- [`func %s`](#%s)\n", fn.Name, symAnchor("func", "", fn.Name))
	}
	for _, t := range l.Doc.Types {
		fmt.Fprintf(&b, "- [`type %s`](#%s)\n", t.Name, symAnchor("type", "", t.Name))
		for _, fn := range t.Funcs {
			fmt.Fprintf(&b, "    - [`func %s`](#%s)\n", fn.Name, symAnchor("func", "", fn.Name))
		}
		for _, m := range t.Methods {
			fmt.Fprintf(&b, "    - [`func (%s) %s`](#%s)\n",
				recvName(m), m.Name, symAnchor("func", t.Name, m.Name))
		}
	}

	b.WriteString("\n")
	return b.String()
}

// overview is docs/api/index.md: what each package is for, and the way in.
func (g *gen) overview(pkgs []*Loaded) []byte {
	var b strings.Builder
	b.WriteString("# API Reference\n\n")
	b.WriteString("One page per public package, generated from the packages' own doc\n")
	b.WriteString("comments. The [Concepts](../concepts/architecture.md) pages explain how the\n")
	b.WriteString("pieces fit together; these pages are the exact surface.\n\n")

	for _, group := range sortedGroups() {
		fmt.Fprintf(&b, "## %s\n\n", group)
		b.WriteString("| Package | Import | What it is for |\n")
		b.WriteString("| --- | --- | --- |\n")
		for _, p := range Packages {
			if p.Group != group {
				continue
			}
			fmt.Fprintf(&b, "| [%s](%s) | `%s` | %s |\n",
				p.Name(), p.Page(), p.ImportPath(), p.Blurb)
		}
		b.WriteString("\n")
	}

	// A count is the one fact a reader cannot get by looking, and it is also
	// the fact that makes a stale page obvious.
	var types, funcs int
	for _, l := range pkgs {
		types += len(l.Doc.Types)
		funcs += len(l.Doc.Funcs)
		for _, t := range l.Doc.Types {
			funcs += len(t.Funcs) + len(t.Methods)
		}
	}
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "%d packages, %d exported types, %d exported functions and methods.\n\n",
		len(pkgs), types, funcs)
	b.WriteString("These pages are generated by `go run ./internal/apidoc/gen` and committed.\n")
	b.WriteString("`go test ./internal/apidoc` fails if they have drifted from the source.\n")

	return []byte(b.String())
}

// ---------------------------------------------------------------------------
// Sections
// ---------------------------------------------------------------------------

// values renders the Constants or Variables section. Both are blocks of
// declarations rather than named symbols, so they get one shared h2 and no
// heading of their own — a const block's members are found by searching the
// page or through the index line above, and giving forty enum members forty
// sidebar entries would bury every type in the package.
func (g *gen) values(b *strings.Builder, l *Loaded, title string, vals []*doc.Value) {
	if len(vals) == 0 {
		return
	}
	fmt.Fprintf(b, "## %s\n\n", title)
	for _, v := range vals {
		if d := strings.TrimSpace(v.Doc); d != "" {
			b.WriteString(g.comment(l, d, 4))
			b.WriteString("\n")
		}
		fmt.Fprintf(b, "```go\n%s\n```\n\n", g.print(l, valueDecl(v)))
		fmt.Fprintf(b, "%s\n\n", g.source(l, v.Decl.Pos()))
	}
}

func (g *gen) function(b *strings.Builder, l *Loaded, fn *doc.Func, level int) {
	heading := "func " + fn.Name
	if fn.Recv != "" {
		heading = fmt.Sprintf("func (%s) %s", recvName(fn), fn.Name)
	}
	fmt.Fprintf(b, "%s %s\n\n", strings.Repeat("#", level), heading)
	fmt.Fprintf(b, "```go\n%s\n```\n\n", g.print(l, funcDecl(fn)))
	if d := strings.TrimSpace(fn.Doc); d != "" {
		b.WriteString(g.comment(l, d, level+1))
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "%s\n\n", g.source(l, fn.Decl.Pos()))
}

func (g *gen) typ(b *strings.Builder, l *Loaded, t *doc.Type) {
	fmt.Fprintf(b, "### type %s\n\n", t.Name)
	fmt.Fprintf(b, "```go\n%s\n```\n\n", g.print(l, typeDecl(t)))
	if d := strings.TrimSpace(t.Doc); d != "" {
		b.WriteString(g.comment(l, d, 4))
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "%s\n\n", g.source(l, t.Decl.Pos()))

	// Constants and variables of this type, which go/doc has already moved off
	// the package lists and onto the type.
	for _, v := range t.Consts {
		g.typeValue(b, l, v)
	}
	for _, v := range t.Vars {
		g.typeValue(b, l, v)
	}

	for _, fn := range t.Funcs { // constructors
		g.function(b, l, fn, 4)
	}
	for _, m := range t.Methods {
		g.function(b, l, m, 4)
	}
}

func (g *gen) typeValue(b *strings.Builder, l *Loaded, v *doc.Value) {
	if d := strings.TrimSpace(v.Doc); d != "" {
		b.WriteString(g.comment(l, d, 5))
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "```go\n%s\n```\n\n", g.print(l, valueDecl(v)))
}

// ---------------------------------------------------------------------------
// Doc comments
// ---------------------------------------------------------------------------

// comment renders a doc comment to markdown.
//
// go/doc/comment does the work, which is the reason the doc comments in this
// repository read the way they do on a docs site at all: it understands Go's
// conventions — an indented block is code, a "# " line is a heading, a bare
// "[Node]" is a link to a symbol — and none of those survive being pasted into
// markdown raw. Two things are configured on top of it:
//
//   - headingLevel, so that a heading inside a doc comment nests under the
//     symbol's own heading instead of competing with it.
//   - the two lookups, which decide what counts as a doc link. The parser's
//     default only recognises packages the *documented file* imports, which
//     would make "[components.Tabs]" a literal bracket in any package that does
//     not import components. Widening it to every documented package makes a
//     symbol reference resolvable from anywhere in the module.
func (g *gen) comment(l *Loaded, text string, headingLevel int) string {
	p := l.Doc.Parser()
	base := p.LookupPackage
	p.LookupPackage = func(name string) (string, bool) {
		if ip, ok := g.byName[name]; ok {
			return ip, true
		}
		if base != nil {
			return base(name)
		}
		return "", false
	}

	pr := l.Doc.Printer()
	pr.HeadingLevel = headingLevel
	pr.DocLinkURL = func(dl *comment.DocLink) string { return g.docLinkURL(l, dl) }

	// The markdown printer's default is to give every heading inside a doc
	// comment an explicit anchor, written as `## Text {#hdr-Text}`. That
	// syntax is goldmark's attribute extension, which gkdocs does not enable,
	// so the braces would reach the page as part of the heading's own text.
	// Returning an empty id drops the suffix and leaves the heading to
	// goldmark's automatic one — which is what every other heading on the page
	// gets anyway.
	pr.HeadingID = func(*comment.Heading) string { return "" }

	return string(pr.Markdown(p.Parse(text)))
}

// docLinkURL turns a resolved doc link into a URL on this site where it can be,
// and into a pkg.go.dev URL where it cannot.
//
// The in-module case relies on the pages being flat siblings under docs/api/:
// "components.md#type-tabs" resolves correctly from every other page in the
// directory, and gkdocs' link rewriter strips the ".md" on the way out. A link
// whose target exists but whose *kind* is unknown to the symbol index — an
// unexported symbol, or one in a package deliberately left undocumented —
// degrades to the package page rather than to a dead anchor.
func (g *gen) docLinkURL(l *Loaded, dl *comment.DocLink) string {
	target := dl.ImportPath
	if target == "" {
		target = l.Pkg.ImportPath()
	}

	pkg, documented := g.byPath[target]
	if !documented {
		return "https://pkg.go.dev/" + dl.ImportPath + "#" + dl.Name
	}

	key := dl.Name
	if dl.Recv != "" {
		key = dl.Recv + "." + dl.Name
	}
	anchor, known := g.syms[target][key]

	// Same page: an anchor on its own, so the link does not depend on the
	// page's own URL.
	base := pkg.Page()
	if target == l.Pkg.ImportPath() {
		base = ""
	}
	if !known {
		return base
	}
	return base + "#" + anchor
}

// ---------------------------------------------------------------------------
// Declarations
// ---------------------------------------------------------------------------

// print renders a synthesised declaration as Go source.
//
// Tabs are turned into spaces because the output lands inside a markdown code
// fence, where a tab is whatever width the reader's browser decides and struct
// fields stop lining up.
func (g *gen) print(l *Loaded, node ast.Node) string {
	var buf bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 4}
	if err := cfg.Fprint(&buf, l.FSet, node); err != nil {
		// A printer failure on an AST that just parsed is not a condition a
		// caller can do anything about, and swallowing it would ship a page
		// with a blank code fence.
		return "// unprintable declaration: " + err.Error()
	}
	return strings.TrimRight(buf.String(), "\n")
}

// source is the "declared in" line under a declaration.
func (g *gen) source(l *Loaded, pos token.Pos) string {
	p := l.FSet.Position(pos)
	rel, err := filepath.Rel(g.root, p.Filename)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	return fmt.Sprintf("<small>[%s:%d](%s%s#L%d)</small>", rel, p.Line, SourceBaseURL, rel, p.Line)
}

// funcDecl strips a function down to what a reference page shows: the
// signature. The copy is so that nilling Doc and Body does not edit the AST the
// rest of the run is still reading.
func funcDecl(fn *doc.Func) ast.Node {
	d := *fn.Decl
	d.Doc = nil
	d.Body = nil
	return &d
}

// valueDecl strips the doc comment off a const or var block. The block itself
// is printed whole, including any unexported names declared alongside exported
// ones: within a single ValueSpec the names and the values are positional, and
// dropping a name without dropping its value would print a declaration that
// does not compile.
func valueDecl(v *doc.Value) ast.Node {
	d := *v.Decl
	d.Doc = nil
	return &d
}

// typeDecl strips the doc comment off a type declaration and elides the
// unexported members of a struct or interface, the way godoc does. A caller
// cannot name an unexported field, so listing them is noise; saying that some
// were left out is not, because it tells the reader the zero value is not
// simply the zero value of what is shown.
func typeDecl(t *doc.Type) ast.Node {
	d := *t.Decl
	d.Doc = nil

	specs := make([]ast.Spec, 0, len(d.Specs))
	for _, s := range d.Specs {
		ts, ok := s.(*ast.TypeSpec)
		if !ok {
			specs = append(specs, s)
			continue
		}
		c := *ts
		c.Doc = nil
		c.Comment = nil
		c.Type = exportedOnly(ts.Type)
		specs = append(specs, &c)
	}
	d.Specs = specs
	return &d
}

// exportedOnly rewrites a struct or interface type with its unexported members
// removed. Anything else is returned unchanged, so a named type, a map, a func
// type or a type parameter list prints exactly as written.
//
// The recursion stops at one level on purpose: an anonymous struct nested inside
// an exported field is part of that field's type, and a caller constructing the
// field needs every one of its members, exported or not.
func exportedOnly(e ast.Expr) ast.Expr {
	switch t := e.(type) {
	case *ast.StructType:
		fields, _ := filterFields(t.Fields, hiddenFieldsMarker)
		c := *t
		c.Fields = fields
		return &c
	case *ast.InterfaceType:
		methods, _ := filterFields(t.Methods, hiddenMethodsMarker)
		c := *t
		c.Methods = methods
		return &c
	}
	return e
}

// filterFields keeps the exported members of a field list and appends marker as
// a synthetic member when anything was dropped. See hiddenFieldsMarker for why
// the marker is an identifier and not a comment.
func filterFields(fl *ast.FieldList, marker string) (*ast.FieldList, bool) {
	if fl == nil {
		return nil, false
	}

	kept := make([]*ast.Field, 0, len(fl.List))
	var hidden bool
	for _, f := range fl.List {
		names := exportedFieldNames(f)
		switch {
		case len(f.Names) == 0:
			// Embedded, or an interface's embedded constraint. Its visibility
			// is the visibility of the type's own name; an embedded type from
			// another package (pkg.T) is exported if T is.
			if embeddedIsExported(f.Type) {
				kept = append(kept, f)
			} else {
				hidden = true
			}
		case len(names) == 0:
			hidden = true
		case len(names) == len(f.Names):
			kept = append(kept, f)
		default:
			// A mixed "A, b int" line: keep the exported names, drop the rest.
			// Unlike a ValueSpec there are no values to keep in step with, so
			// this rewrite is safe.
			c := *f
			c.Names = names
			c.Doc = nil
			c.Comment = f.Comment
			kept = append(kept, &c)
			hidden = true
		}
	}

	if hidden {
		kept = append(kept, &ast.Field{Type: ast.NewIdent(marker)})
	}

	c := *fl
	c.List = kept
	return &c, hidden
}

func exportedFieldNames(f *ast.Field) []*ast.Ident {
	out := make([]*ast.Ident, 0, len(f.Names))
	for _, n := range f.Names {
		if n.IsExported() {
			out = append(out, n)
		}
	}
	return out
}

// embeddedIsExported reports whether an embedded field or constraint is part of
// the public surface. It looks through the pointer, qualifier and type-argument
// wrappers an embedded type can wear and asks whether the name it arrives at is
// exported. A union constraint ("int | string") has no name and is always part
// of the interface's meaning, so it is kept.
func embeddedIsExported(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.Ident:
		return t.IsExported()
	case *ast.StarExpr:
		return embeddedIsExported(t.X)
	case *ast.SelectorExpr: // pkg.T — T decides
		return t.Sel.IsExported()
	case *ast.IndexExpr: // T[int]
		return embeddedIsExported(t.X)
	case *ast.IndexListExpr: // T[int, string]
		return embeddedIsExported(t.X)
	}
	return true
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

// recvName is the receiver as a heading shows it.
//
// doc.Func.Recv arrives as "Node", "*Manager" or "*State[T]" — the type, with
// no receiver variable, which is already what a heading wants. The one edit is
// the type parameter list, dropped here for the sake of the anchor: slug() has
// no case for '[' or ']' so it deletes them, and "State[T]" would slug to
// "statet" while a doc link to that same method arrives as ("State", "Get")
// and can only produce "state". The list is a poor loss to take — the code
// fence directly under the heading carries the full signature, type parameters
// and receiver variable included — and it buys a method on a generic type an
// anchor that cross-references can actually reach.
//
// The pointer star stays. slug() drops it too, so a pointer and a value
// receiver land on the same anchor, and a method set cannot contain both.
func recvName(fn *doc.Func) string {
	r := fn.Recv
	if i := strings.IndexByte(r, '['); i >= 0 {
		r = r[:i]
	}
	return r
}

func allNames(vals []*doc.Value) []string {
	var out []string
	for _, v := range vals {
		for _, n := range v.Names {
			if ast.IsExported(n) {
				out = append(out, n)
			}
		}
	}
	return stableNames(out)
}

func countNames(vals []*doc.Value) int { return len(allNames(vals)) }

// codeList wraps each name in backticks, and caps the run so that a package
// with a hundred enum members does not put a hundred of them on one index line.
func codeList(names []string) []string {
	const max = 12
	out := make([]string, 0, max+1)
	for i, n := range names {
		if i == max {
			out = append(out, fmt.Sprintf("and %d more", len(names)-max))
			break
		}
		out = append(out, "`"+n+"`")
	}
	return out
}
