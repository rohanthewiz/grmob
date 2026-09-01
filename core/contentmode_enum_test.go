package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// ContentModes is a hand-written list of constants that are declared a few
// lines above it, which is the shape of duplication that quietly goes stale:
// nothing about adding a fifth ContentMode forces anyone to scroll down. So
// the list is checked against the declaration rather than trusted.
//
// This reads image.go's syntax tree rather than matching text, because the
// question being asked is a syntactic one — "which constants of type
// ContentMode does this file declare?" — and a regexp over source would also
// match the same words inside a doc comment or a test fixture. The parse is
// source-only (no type checking, no imports resolved), so it costs a
// millisecond and needs nothing built.
//
// Every renderer's coverage check ultimately rests on this list — see
// htmlout.ObjectFits — so it is the one link in that chain that has to be
// pinned to something other than another list.
func TestContentModesMatchTheDeclaredConstants(t *testing.T) {
	declared := declaredContentModes(t)

	listed := map[string]bool{}
	for _, m := range ContentModes() {
		if listed[string(m)] {
			t.Errorf("ContentModes lists %q twice", m)
		}
		listed[string(m)] = true
	}

	for name, value := range declared {
		if !listed[value] {
			t.Errorf("%s = %q is declared but missing from ContentModes()", name, value)
		}
		delete(listed, value)
	}
	// Anything left is a value ContentModes hands out that no constant backs —
	// a renderer table checked against it would then be required to handle a
	// mode core cannot produce.
	for value := range listed {
		t.Errorf("ContentModes() yields %q, which no ContentMode constant declares", value)
	}
}

// declaredContentModes returns the ContentMode constants image.go declares, as
// name -> value.
//
// Failure at every step rather than an empty map: a check that reads nothing
// must not be able to read as a pass. If this file is renamed or the constants
// move, the test is meant to say so, not to agree with whatever it was given.
func declaredContentModes(t *testing.T) map[string]string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "image.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing image.go: %v — if the ContentMode constants moved, update this test", err)
	}

	out := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			// Each constant carries its type explicitly (no iota carry-over),
			// so the type is read off the spec itself.
			if id, ok := vs.Type.(*ast.Ident); !ok || id.Name != "ContentMode" {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					t.Fatalf("%s has no value; this test assumes each ContentMode constant declares one", name.Name)
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					t.Fatalf("%s is not declared as a string literal; this test cannot read it", name.Name)
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("%s: unquoting %s: %v", name.Name, lit.Value, err)
				}
				out[name.Name] = value
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("image.go declares no ContentMode constants — the parse found nothing, which is not a pass")
	}
	return out
}
