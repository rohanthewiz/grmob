package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// BundledThemes must name every theme this package ships.
//
// # Why this is derived from the source and not written out
//
// The map is the single list every census in the repository loops over, so the
// question "does this rule hold for all our themes" is only as good as the
// map's completeness — and a theme missing from it fails nothing, anywhere. It
// simply is not asked. That is the exact failure the *value* of a shared list
// is supposed to prevent, so it would be a poor joke to guard it with a second
// hand-written list.
//
// So the declarations are read out of theme.go: every package-level
// `var X = &Theme{...}` has to appear in BundledThemes() under its own
// identifier. Adding a fourth palette and forgetting the map fails here by
// name, which is the same trick TestBundledThemesSetEveryColorRole plays one
// level down with reflection over ColorPalette's fields.
//
// The parse is deliberately narrow — the file, the top level, `&Theme{` — so
// that a *Theme built inside a function (a test fixture, a WithTheme example)
// is not swept in. A bundled theme is a package-level var by definition: it is
// the thing an app names.
func TestBundledThemesListIsExhaustive(t *testing.T) {
	declared := themeVarsDeclaredIn(t, "theme.go")
	if len(declared) == 0 {
		t.Fatal("no package-level *Theme vars found in theme.go — the parse below has " +
			"stopped matching how the themes are declared, so this test now proves nothing")
	}

	listed := BundledThemes()
	for _, name := range declared {
		theme, ok := listed[name]
		if !ok {
			t.Errorf("theme.go declares %s and BundledThemes() does not list it. Every "+
				"census in this repository loops over that map, so an unlisted theme is "+
				"never asked whether it keeps a single rule — it does not fail them, it "+
				"is absent from them", name)
			continue
		}
		if theme == nil {
			t.Errorf("BundledThemes()[%q] is nil", name)
		}
	}
	for name := range listed {
		found := false
		for _, d := range declared {
			if d == name {
				found = true
			}
		}
		if !found {
			t.Errorf("BundledThemes() lists %q, which theme.go does not declare as a "+
				"package-level theme var — a renamed or deleted theme leaves the map "+
				"pointing at something else", name)
		}
	}
}

// Two entries pointing at one theme would pass the exhaustiveness check above
// (every declaration is listed, every listing is declared) and would still
// mean a census runs one palette twice and another not at all.
func TestBundledThemesAreDistinct(t *testing.T) {
	seen := map[*Theme]string{}
	for name, theme := range BundledThemes() {
		if prev, dup := seen[theme]; dup {
			t.Errorf("BundledThemes() maps both %q and %q to the same *Theme", prev, name)
		}
		seen[theme] = name
	}
}

// themeVarsDeclaredIn returns the names of the package-level `var X = &Theme{}`
// declarations in one file of this package, sorted.
func themeVarsDeclaredIn(t *testing.T, file string) []string {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}

	var names []string
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, value := range vs.Values {
				if !isThemeLiteral(value) || i >= len(vs.Names) {
					continue
				}
				name := vs.Names[i].Name
				// An unexported theme var would be a fixture rather than a
				// shipped palette; nothing declares one today, and if one
				// arrives the map should not be forced to carry it.
				if strings.HasPrefix(name, strings.ToLower(name[:1])) {
					continue
				}
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names
}

// isThemeLiteral matches `&Theme{...}`, which is how every bundled palette is
// spelled. A composite literal of any other type, or a call, is not one.
func isThemeLiteral(x ast.Expr) bool {
	unary, ok := x.(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return false
	}
	lit, ok := unary.X.(*ast.CompositeLit)
	if !ok {
		return false
	}
	ident, ok := lit.Type.(*ast.Ident)
	return ok && ident.Name == "Theme"
}
