package comps

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
// # The answer, first version
//
// For a long time nothing in `comps` declared a composite CONTAINER role.
// core.RoleOption is set by ListRow, core.RoleTab is documented for a
// SegmentedControl's segments — those are member roles, and a member role is a
// fact about a node's parent. ListRow's doc says "set core.RoleListBox on the
// container yourself" and SegmentedControl's says the same for tablist, each
// with the recipe written out. So a nested composite needed an author to
// declare BOTH container roles, by hand, on two containers they built.
//
// # What changed, and the answer now
//
// Three widgets declare a container role for themselves:
//
//	BottomBar    toolbar, when Selected < 0 (Tier A). It chose the role through
//	             a variable, which the first version of this test did not see:
//	             it only matched a role constant written directly as the
//	             argument of core.AccessibilityRole. The scan below looks at
//	             every reference to a container constant in code, so a role
//	             reaching the call by any route is found.
//	RadioGroup   radiogroup (Tier B; a listbox until core carried the radio
//	             pair). The widget is the container; making every caller role
//	             it by hand would be the recipe the widget exists to replace.
//	RichTextEditor
//	             toolbar, on its formatting strip. It carried no role while
//	             the rule was "no widget declares one"; the strip's buttons are
//	             built from RichToolItem data, so it met the closed rule.
//
// Both are CLOSED: every member is built by the widget from data (BarItem,
// RadioOption), and no struct the widget declares holds a core.View. A closed
// composite cannot have another composite put inside it, so composing widgets
// still cannot nest two composites on its own. The reachable shape is one step
// shorter than before: an author places a closed widget inside a container
// they roled by hand — a RadioGroup inside their own listbox. That is one
// deliberate declaration rather than two, and core.AuditTree's finding names
// it. The five descending pairs remain worth their reasoning.
//
// # What this test holds
//
// The premise, which is the half that can change:
//
//   - A file that refers to a container role in code must be on the
//     closedComposites list, with its reason.
//   - A listed file must declare no struct holding a core.View, in a field or
//     a slice. An open widget declaring a container would let two of it nest
//     by composition, and the answer above would be stale with nothing to say
//     so.
//   - A listed file must still refer to a container role, so the list cannot
//     outlive the declarations it excuses.
//
// Parsed rather than grepped, for the reason mobile/verify's Swift anchors are:
// every one of these roles is named in a doc comment somewhere in this package
// (the recipes are the point), and a substring search would report each of
// those as a declaration.
func TestOnlyClosedWidgetsDeclareACompositeContainerRole(t *testing.T) {
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

	// The closed widgets allowed to declare a container role, by file.
	closedComposites := map[string]string{
		"bottom_bar.go":       "toolbar when Selected < 0; cells are built from BarItem data",
		"code_editor.go":      "toolbar; its three buttons are built by the widget",
		"radio_group.go":      "radiogroup; rows are built from RadioOption data",
		"rich_text_editor.go": "toolbar; buttons are built from RichToolItem data",
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("listing this package: %v", err)
	}
	fset := token.NewFileSet()
	declares := map[string][]string{}
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

		// Every reference, not only a direct AccessibilityRole argument: a
		// role chosen into a variable first is still declared by the widget.
		ast.Inspect(file, func(n ast.Node) bool {
			sel, isSel := n.(*ast.SelectorExpr)
			if isSel && isCorePkg(sel.X) && container[sel.Sel.Name] {
				declares[name] = append(declares[name], "core."+sel.Sel.Name)
			}
			return true
		})

		if _, listed := closedComposites[name]; listed {
			if fields := viewFields(file); len(fields) > 0 {
				t.Errorf("%s declares a composite container role and holds caller views in %v.\n\n"+
					"A widget on the closed list must build every member itself. One that "+
					"takes a core.View can hold another composite, so two of it nest by "+
					"composition alone. Re-read this test's header and the note in "+
					"keynav_test.mjs, then either close the widget or update both.",
					name, fields)
			}
		}
	}

	var open []string
	for name, roles := range declares {
		if _, listed := closedComposites[name]; !listed {
			open = append(open, name+": "+strings.Join(roles, ", "))
		}
	}
	sort.Strings(open)
	if len(open) > 0 {
		t.Errorf("a widget not on the closed list declares a composite container role: %v\n\n"+
			"A widget that declares one makes nesting reachable by composing widgets "+
			"unless it is closed (it builds every member and holds no core.View). "+
			"Re-read this test's header and the note in keynav_test.mjs, then add the "+
			"file to closedComposites with its reason, or leave the role to the caller.", open)
	}
	for name := range closedComposites {
		if len(declares[name]) == 0 {
			t.Errorf("%s is on the closed list but no longer refers to a container role; "+
				"remove it so the list only excuses declarations that exist", name)
		}
	}
}

// viewFields returns "Type.Field" for every struct field declared in the file
// whose type is core.View or a slice of it.
func viewFields(file *ast.File) []string {
	var out []string
	ast.Inspect(file, func(n ast.Node) bool {
		spec, isSpec := n.(*ast.TypeSpec)
		if !isSpec {
			return true
		}
		st, isStruct := spec.Type.(*ast.StructType)
		if !isStruct {
			return true
		}
		for _, f := range st.Fields.List {
			typ := f.Type
			if arr, isArr := typ.(*ast.ArrayType); isArr {
				typ = arr.Elt
			}
			sel, isSel := typ.(*ast.SelectorExpr)
			if !isSel || !isCorePkg(sel.X) || sel.Sel.Name != "View" {
				continue
			}
			for _, id := range f.Names {
				out = append(out, spec.Name.Name+"."+id.Name)
			}
		}
		return true
	})
	sort.Strings(out)
	return out
}

// roleConstName maps a core.Role value back to the identifier core declares it
// under, which is what a source scan can see.
//
// A switch rather than a lookup, because the mapping is exactly the four
// containers and a fourth added to core.KeyboardComposites() should fail here
// rather than be silently skipped by a scan that cannot name it.
func roleConstName(t *testing.T, r core.Role) string {
	t.Helper()
	switch r {
	case core.RoleListBox:
		return "RoleListBox"
	case core.RoleRadioGroup:
		return "RoleRadioGroup"
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
