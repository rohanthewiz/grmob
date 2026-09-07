package components

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Whether any screen this framework builds can produce a nested composite.
//
// # The finding that had no example
//
// core.CompositeWalkStopsAt sorts the nine ordered pairs of keyboard composites
// into four that stop and five that descend, and a descending pair is the
// interesting one: the outer widget's arrows can land *inside* a widget they
// are not steering. wasm/verify's keynav_test.mjs exercises it with a
// `listbox > tablist > option` tree, and that test's own doc comment calls the
// tree contrived — which left the obvious question unasked. If no real screen
// can produce a descending pair, the five findings are ones nobody will ever
// see, and core.AuditTree carries reasoning for a shape that cannot exist.
//
// # The answer
//
// Nothing in `components` declares a composite CONTAINER role. Not one widget.
// core.RoleOption is set by ListRow, core.RoleTab is documented for a
// SegmentedControl's segments — those are member roles, and a member role is a
// fact about a node's parent. The containers that give a widget a keyboard —
// listbox, tablist, toolbar — are always the caller's own: ListRow's doc says
// "set core.RoleListBox on the container yourself" and SegmentedControl's says
// the same for tablist, each with the recipe written out.
//
// So a nested composite needs an author to declare BOTH container roles, by
// hand, on two containers they built, one inside the other. It is reachable —
// nothing prevents it and the recipes are published — and it is not something a
// screen falls into by composing widgets. That is a different thing from
// contrived, and it is why the five descending pairs are worth their reasoning:
// the author who writes that tree wrote two deliberate declarations and will
// read the finding that names them.
//
// # What this test holds
//
// The premise, which is the half that can change. A widget that started
// declaring a container role for itself would make nested composites something
// composition alone can produce — two of that widget nested, or one inside a
// caller's own listbox — and the reachability answer above would be stale with
// nothing to say so.
//
// Parsed rather than grepped, for the reason mobile/verify's Swift anchors are:
// every one of these roles is named in a doc comment somewhere in this package
// (the recipes are the point), and a substring search would report each of
// those as a declaration.
func TestNoWidgetDeclaresACompositeContainerRole(t *testing.T) {
	// The container roles, from core rather than transcribed: a fourth keyboard
	// composite added there has to be looked for here too, and a list written
	// out would silently stop covering it.
	container := map[string]bool{}
	for _, r := range core.KeyboardComposites() {
		container[roleConstName(t, r)] = true
	}
	if len(container) == 0 {
		t.Fatal("core.KeyboardComposites() is empty, so this test has no subject")
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("listing this package: %v", err)
	}
	fset := token.NewFileSet()
	var found []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		// Comments are dropped: the recipes in the doc comments name these
		// roles on purpose, and they are instructions to a caller rather than
		// declarations by a widget.
		file, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, isCall := n.(*ast.CallExpr)
			if !isCall || len(call.Args) != 1 {
				return true
			}
			if !isCoreCall(call.Fun, "AccessibilityRole") {
				return true
			}
			sel, isSel := call.Args[0].(*ast.SelectorExpr)
			if !isSel || !isCorePkg(sel.X) || !container[sel.Sel.Name] {
				return true
			}
			found = append(found, name+": core."+sel.Sel.Name)
			return true
		})
	}

	sort.Strings(found)
	if len(found) > 0 {
		t.Errorf("a widget in this package declares a composite container role: %v\n\n"+
			"Until now none did, and that is the whole of the answer to \"can a real "+
			"screen produce a nested composite\": a caller has to declare both "+
			"container roles by hand, so the pair is deliberate rather than something "+
			"composition falls into. A widget that declares one makes nesting reachable "+
			"by composing widgets — two of this one, or one inside a caller's own "+
			"listbox — and core.AuditTree's five descending pairs become findings a "+
			"screen can produce by accident. Re-read this test's header and the note "+
			"in keynav_test.mjs, then update both.", found)
	}
}

// roleConstName maps a core.Role value back to the identifier core declares it
// under, which is what a source scan can see.
//
// A switch rather than a lookup, because the mapping is exactly the three
// containers and a fourth added to core.KeyboardComposites() should fail here
// rather than be silently skipped by a scan that cannot name it.
func roleConstName(t *testing.T, r core.Role) string {
	t.Helper()
	switch r {
	case core.RoleListBox:
		return "RoleListBox"
	case core.RoleTabList:
		return "RoleTabList"
	case core.RoleToolbar:
		return "RoleToolbar"
	}
	t.Fatalf("core.KeyboardComposites() names %q and this test cannot name the "+
		"constant it is declared under, so a widget declaring it would not be "+
		"looked for. Add the arm.", r)
	return ""
}

// isCoreCall reports whether an expression is a call of core.<name>.
func isCoreCall(fun ast.Expr, name string) bool {
	sel, isSel := fun.(*ast.SelectorExpr)
	return isSel && sel.Sel.Name == name && isCorePkg(sel.X)
}

// isCorePkg reports whether an expression is the identifier `core`.
//
// The package is imported under its own name everywhere in this package, and a
// rename would make this stop finding declarations — so the totality guard is
// the count check in the caller rather than anything here.
func isCorePkg(x ast.Expr) bool {
	ident, isIdent := x.(*ast.Ident)
	return isIdent && ident.Name == "core"
}
