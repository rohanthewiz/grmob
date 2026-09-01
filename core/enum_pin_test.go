package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"testing"
)

// Shared machinery for the tests that pin a hand-written enumeration function
// to the const block it enumerates: ContentModes, Alignments, TextAlignments,
// JustifyContents, AlignItemsValues.
//
// Every one of those functions restates a list of constants declared a few
// lines (or a file) away, which is the shape of duplication that quietly goes
// stale: nothing about adding a seventh JustifyContent forces anyone to scroll
// down. So the lists are checked against their declarations rather than
// trusted.
//
// This reads the syntax tree rather than matching text, because the question
// is a syntactic one — "which constants of type T does this file declare?" —
// and a regexp over source would also match the same words inside a doc
// comment or a test fixture. Both files here contain exactly that hazard:
// image.go's comments spell out "fit"/"fill"/"stretch"/"center" in prose, and
// alignment.go's tables list every Alignment twice. The parse is source-only
// (no type checking, no imports resolved), so it costs a millisecond and needs
// nothing built.
//
// Every renderer coverage check downstream — in htmlout, wasm/verify and
// mobile/verify — ultimately rests on one of these lists, so these are the
// links in that chain that have to be pinned to something other than another
// list.

// declaredConstants returns the constants of the named type that the named
// file declares, as name -> value.
//
// Failure at every step rather than an empty map: a check that reads nothing
// must not be able to read as a pass. If a file is renamed or a const block
// moves, the test is meant to say so, not to agree with whatever it was given.
//
// Only specs that name their type explicitly are collected. That is not a
// limitation in practice and it is deliberately not worked around: a spec with
// no type and no value inherits the previous spec's type through iota
// carry-over, which this would have to simulate, while a spec with a value and
// no type (style.go's `DisplayFlex = "flex"`, sitting in the same block as the
// FlexDirection constants) is an untyped constant that genuinely is not a
// member of the type. Skipping the untyped ones is correct; carry-over is not
// used by any of these blocks, and a block that started using it would fail
// the exact-match check by coming up short rather than pass quietly.
func declaredConstants(t *testing.T, file, typeName string) map[string]string {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v — if the %s constants moved, update this test", file, err, typeName)
	}

	out := map[string]string{}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			// A nil Type (the untyped case above) fails this assertion and is
			// skipped, which is why it is written as an assertion rather than
			// a nil check followed by one.
			if id, ok := vs.Type.(*ast.Ident); !ok || id.Name != typeName {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					t.Fatalf("%s: %s has no value; this test assumes each %s constant declares one",
						file, name.Name, typeName)
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					t.Fatalf("%s: %s is not declared as a string literal; this test cannot read it",
						file, name.Name)
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("%s: %s: unquoting %s: %v", file, name.Name, lit.Value, err)
				}
				out[name.Name] = value
			}
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s declares no %s constants — the parse found nothing, which is not a pass",
			file, typeName)
	}
	return out
}

// enumStrings drops the named string type off an enumeration so it can be
// compared with values read out of a syntax tree, which have already lost it.
func enumStrings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// requireExactEnum checks a list against the declarations in both directions:
// every constant must be listed, and every listed value must be backed by a
// constant.
//
// Both directions matter and they fail for different reasons. A constant that
// is not listed shrinks the set every downstream coverage check runs against,
// so a renderer could stop handling it and no test would notice — the exact
// failure these pins exist to prevent. A listed value with no constant behind
// it is the mirror image: the renderers would be *required* to keep handling
// something core can no longer produce, which reads to the next person as
// support the framework has.
func requireExactEnum[T ~string](t *testing.T, file, typeName, listName string, values []T) {
	t.Helper()

	declared := declaredConstants(t, file, typeName)

	listed := map[string]bool{}
	for _, value := range enumStrings(values) {
		if listed[value] {
			t.Errorf("%s lists %q twice", listName, value)
		}
		listed[value] = true
	}

	for _, name := range sortedNames(declared) {
		if !listed[declared[name]] {
			t.Errorf("%s = %q is declared in %s but missing from %s",
				name, declared[name], file, listName)
		}
		delete(listed, declared[name])
	}
	for _, value := range sortedKeys(listed) {
		t.Errorf("%s yields %q, which no %s constant in %s declares",
			listName, value, typeName, file)
	}
}

// requireSubsetEnum checks a list that is deliberately narrower than the type:
// every listed value must still be a declared constant, but constants may be
// left out.
//
// Only one list is like this — TextAlignments, which omits the two Alignments
// that name no text alignment — and the direction it keeps is the one that
// still bites. A typo, or a constant renamed out from under it, would produce
// a list entry that no renderer arm could ever match; the resulting coverage
// check would then demand an arm for a value core cannot produce. What it
// cannot check is the omissions, which is exactly why the omissions are
// argued for in TextAlignments' own doc comment rather than left implicit.
func requireSubsetEnum[T ~string](t *testing.T, file, typeName, listName string, values []T) {
	t.Helper()

	declared := declaredConstants(t, file, typeName)
	backed := map[string]bool{}
	for _, value := range declared {
		backed[value] = true
	}

	seen := map[string]bool{}
	for _, value := range enumStrings(values) {
		if !backed[value] {
			t.Errorf("%s yields %q, which no %s constant in %s declares",
				listName, value, typeName, file)
		}
		if seen[value] {
			t.Errorf("%s lists %q twice", listName, value)
		}
		seen[value] = true
	}
}

// Deterministic iteration, so a run that reports several problems reports them
// in the same order twice — map order would otherwise shuffle the failure list
// between runs of the same broken state.
func sortedNames(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
