package comps

import (
	"strconv"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// treeFiles is the fixture: two branches (one nested two deep), an empty
// branch, and a top-level leaf.
//
//	docs/            branch
//	  guide.md       leaf
//	  api/           branch
//	    core.md      leaf
//	src/             branch
//	  main.go        leaf
//	empty/           branch by flag, no children
//	README.md        leaf
var treeFiles = []TreeNode{
	{ID: "docs", Label: "docs", Children: []TreeNode{
		{ID: "docs/guide.md", Label: "guide.md"},
		{ID: "docs/api", Label: "api", Children: []TreeNode{
			{ID: "docs/api/core.md", Label: "core.md"},
		}},
	}},
	{ID: "src", Label: "src", Children: []TreeNode{
		{ID: "src/main.go", Label: "main.go"},
	}},
	{ID: "empty", Label: "empty", Branch: true},
	{ID: "README.md", Label: "README.md"},
}

// nodesWhere collects every node satisfying pred, depth-first.
func nodesWhere(n *core.Node, pred func(*core.Node) bool) []*core.Node {
	var out []*core.Node
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if pred(n) {
			out = append(out, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

// treeButton finds the row button with the given spoken name.
func treeButton(n *core.Node, label string) *core.Node {
	return findFirst(n, func(n *core.Node) bool {
		return n.Style.AccessibilityRole == core.RoleButton && n.Style.AccessibilityLabel == label
	})
}

// A nil Expanded is a collapsed tree: the four top-level items and nothing
// under them, in a named list.
func TestTreeViewCollapsedRendersOnlyTheTopLevel(t *testing.T) {
	_, n := renderDebug(t, TreeView{Label: "Project files", Nodes: treeFiles, OnToggle: func(string) {}})

	if n.Style.AccessibilityRole != core.RoleList || n.Style.AccessibilityLabel != "Project files" {
		t.Errorf("role %q label %q", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	if len(n.Children) != 4 {
		t.Fatalf("want 4 top-level items, got %d", len(n.Children))
	}
	for i, c := range n.Children {
		if c.Style.AccessibilityRole != core.RoleListItem || c.Style.AccessibilityNestingLevel != 1 {
			t.Errorf("item %d: role %q level %d", i, c.Style.AccessibilityRole, c.Style.AccessibilityNestingLevel)
		}
		if c.Key != treeFiles[i].ID {
			t.Errorf("item %d keyed %q, want %q", i, c.Key, treeFiles[i].ID)
		}
	}
	if findText(n, "guide.md") != nil {
		t.Error("a shut branch's children should not be rendered")
	}
	if lists := nodesWhere(n, func(n *core.Node) bool { return n.Style.AccessibilityRole == core.RoleList }); len(lists) != 1 {
		t.Errorf("a collapsed tree is one list, got %d", len(lists))
	}
}

// Every branch states its expansion, open or shut, and a leaf states none.
// The empty branch is a branch because of its flag.
func TestTreeViewBranchesStateTheirExpansion(t *testing.T) {
	_, n := renderDebug(t, TreeView{
		Nodes:    treeFiles,
		Expanded: map[string]bool{"docs": true},
		OnToggle: func(string) {},
		OnSelect: func(string) {},
	})

	var none core.ExpandedState // a leaf says nothing, which is not "collapsed"
	want := map[string]core.ExpandedState{
		"docs":      core.ExpandedWhen(true),
		"api":       core.ExpandedWhen(false),
		"src":       core.ExpandedWhen(false),
		"empty":     core.ExpandedWhen(false),
		"guide.md":  none,
		"README.md": none,
	}

	for label, state := range want {
		b := treeButton(n, label)
		if b == nil {
			t.Errorf("no button named %q", label)
			continue
		}
		if b.Style.AccessibilityExpanded != state {
			t.Errorf("%q: expanded %q, want %q", label, b.Style.AccessibilityExpanded, state)
		}
	}
}

// Depth is said twice: by the level on each item and by the nesting itself,
// a list inside the parent's item, named by the parent.
func TestTreeViewNestsListsAndCountsLevels(t *testing.T) {
	_, n := renderDebug(t, TreeView{
		Nodes:    treeFiles,
		Expanded: map[string]bool{"docs": true, "docs/api": true},
		OnToggle: func(string) {},
	})

	levels := map[string]int{}
	for _, item := range nodesWhere(n, func(n *core.Node) bool { return n.Style.AccessibilityRole == core.RoleListItem }) {
		levels[item.Key] = item.Style.AccessibilityNestingLevel
	}
	for id, want := range map[string]int{"docs": 1, "docs/guide.md": 2, "docs/api": 2, "docs/api/core.md": 3, "README.md": 1} {
		if levels[id] != want {
			t.Errorf("%s: level %d, want %d", id, levels[id], want)
		}
	}
	if _, drawn := levels["src/main.go"]; drawn {
		t.Error("src is shut, so main.go should not be an item")
	}

	docs := n.Children[0]
	sub := findFirst(docs, func(n *core.Node) bool { return n.Style.AccessibilityRole == core.RoleList })
	if sub == nil || sub.Style.AccessibilityLabel != "docs" {
		t.Fatalf("the docs item should hold a list named docs, got %+v", sub)
	}
	// A list's children are its items and nothing else: the structural rule
	// in core/role.go, which no audit checks for this pair.
	for _, list := range nodesWhere(n, func(n *core.Node) bool { return n.Style.AccessibilityRole == core.RoleList }) {
		for _, c := range list.Children {
			if c.Style.AccessibilityRole != core.RoleListItem {
				t.Errorf("list %q has a %s child with role %q", list.Style.AccessibilityLabel, c.Type, c.Style.AccessibilityRole)
			}
		}
	}
}

// The indent is a spacer of Indent points leading each nested list, and the
// theme's Spacing.LG when Indent is zero.
func TestTreeViewIndent(t *testing.T) {
	spacerOf := func(n *core.Node) *core.Node {
		// The Row that pairs the spacer with a nested list.
		return findFirst(n, func(n *core.Node) bool {
			return n.Type == "Row" && len(n.Children) == 2 &&
				n.Children[1].Style.AccessibilityRole == core.RoleList
		})
	}
	open := map[string]bool{"docs": true}

	ctx, n := renderDebug(t, TreeView{Nodes: treeFiles, Expanded: open, OnToggle: func(string) {}})
	row := spacerOf(n)
	if row == nil {
		t.Fatal("no indent row")
	}
	if got, want := row.Children[0].Style.Width, strconv.Itoa(ctx.Theme().Spacing.LG)+"px"; got != want {
		t.Errorf("default indent is %q, want %q", got, want)
	}

	_, n = renderDebug(t, TreeView{Nodes: treeFiles, Expanded: open, OnToggle: func(string) {}, Indent: 12})
	if w := spacerOf(n).Children[0].Style.Width; w != "12px" {
		t.Errorf("Indent 12 drew a %q spacer", w)
	}
}

// A tapped branch reports its ID to OnToggle, once, and not to OnSelect. A
// tapped leaf does the reverse.
func TestTreeViewTapsReportIDs(t *testing.T) {
	var toggled, selected []string
	ctx, n := renderDebug(t, TreeView{
		Nodes:    treeFiles,
		Expanded: map[string]bool{"docs": true},
		OnToggle: func(id string) { toggled = append(toggled, id) },
		OnSelect: func(id string) { selected = append(selected, id) },
	})

	ctx.TriggerCallback(treeButton(n, "api").Props["onClick"].(string))
	if len(toggled) != 1 || toggled[0] != "docs/api" || len(selected) != 0 {
		t.Errorf("after a branch tap: toggled %v selected %v", toggled, selected)
	}
	ctx.TriggerCallback(treeButton(n, "guide.md").Props["onClick"].(string))
	if len(selected) != 1 || selected[0] != "docs/guide.md" || len(toggled) != 1 {
		t.Errorf("after a leaf tap: toggled %v selected %v", toggled, selected)
	}
}

// The chosen leaf states aria-current and no other row does. A Selected that
// names a branch marks nothing.
func TestTreeViewSelectedIsCurrent(t *testing.T) {
	tv := TreeView{
		Nodes:    treeFiles,
		Expanded: map[string]bool{"docs": true},
		OnToggle: func(string) {},
		OnSelect: func(string) {},
		Selected: "docs/guide.md",
	}
	_, n := renderDebug(t, tv)
	current := nodesWhere(n, func(n *core.Node) bool { return n.Style.AccessibilityCurrent != core.CurrentNone })
	if len(current) != 1 || current[0].Style.AccessibilityLabel != "guide.md" ||
		current[0].Style.AccessibilityCurrent != core.CurrentTrue {
		t.Errorf("want guide.md alone as current, got %d nodes", len(current))
	}

	tv.Selected = "docs"
	_, n = renderDebug(t, tv)
	if c := nodesWhere(n, func(n *core.Node) bool { return n.Style.AccessibilityCurrent != core.CurrentNone }); len(c) != 0 {
		t.Errorf("a branch cannot be chosen, got %d current nodes", len(c))
	}
}

// With no OnSelect a leaf is text: no role, no handler, no current.
func TestTreeViewLeavesAreTextWithoutOnSelect(t *testing.T) {
	_, n := renderDebug(t, TreeView{Nodes: treeFiles, OnToggle: func(string) {}, Selected: "README.md"})

	if treeButton(n, "README.md") != nil {
		t.Error("a leaf with no OnSelect should not be a button")
	}
	label := findText(n, "README.md")
	if label == nil {
		t.Fatal("the leaf's label is missing")
	}
	if c := nodesWhere(n, func(n *core.Node) bool { return n.Style.AccessibilityCurrent != core.CurrentNone }); len(c) != 0 {
		t.Error("current is stated on a button only")
	}
}

// Leading is drawn and hidden, whatever the caller passed.
func TestTreeViewLeadingIsHidden(t *testing.T) {
	nodes := []TreeNode{{ID: "a", Label: "notes.txt", Leading: core.Text("📄")}}
	_, n := renderDebug(t, TreeView{Nodes: nodes})

	icon := findText(n, "📄")
	if icon == nil {
		t.Fatal("Leading was not drawn")
	}
	wrapper := findFirst(n, func(w *core.Node) bool {
		return len(w.Children) == 1 && w.Children[0] == icon
	})
	if wrapper == nil || !wrapper.Style.AccessibilityHidden {
		t.Error("Leading should sit in a hidden wrapper")
	}
}

// A leaf keeps the chevron column empty, so its label lines up with a
// branch's: both rows lead with a 20pt cell.
func TestTreeViewLeafKeepsTheChevronColumn(t *testing.T) {
	_, n := renderDebug(t, TreeView{Nodes: treeFiles, OnToggle: func(string) {}, OnSelect: func(string) {}})

	for _, label := range []string{"docs", "README.md"} {
		b := treeButton(n, label)
		if b == nil || len(b.Children) == 0 {
			t.Fatalf("%q: no row", label)
		}
		if w := b.Children[0].Style.Width; w != "20px" {
			t.Errorf("%q leads with a %q cell, want 20px", label, w)
		}
	}
}

func TestTreeViewConcerns(t *testing.T) {
	cases := []struct {
		name string
		tv   TreeView
		want string
	}{
		{"a branch and no OnToggle", TreeView{Nodes: treeFiles}, ConcernTreeViewInert},
		{"a duplicate ID at different depths", TreeView{
			OnToggle: func(string) {},
			Nodes: []TreeNode{
				{ID: "a", Label: "A", Children: []TreeNode{{ID: "b", Label: "B"}}},
				{ID: "b", Label: "B again"},
			},
		}, ConcernTreeViewDuplicateID},
		{"a duplicate inside a shut branch is still found", TreeView{
			OnToggle: func(string) {},
			Nodes: []TreeNode{
				{ID: "a", Label: "A", Children: []TreeNode{{ID: "x", Label: "X"}, {ID: "x", Label: "X too"}}},
			},
		}, ConcernTreeViewDuplicateID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			core.SetDebugMode(true)
			core.ClearConcerns()
			t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
			ctx := core.NewContext()
			ctx.BeginRenderPass()
			c.tv.Render(ctx)
			ctx.EndRenderPass()
			if dump := core.DumpConcerns(); !strings.Contains(dump, c.want) {
				t.Errorf("want %s, got:\n%s", c.want, dump)
			}
		})
	}

	// Leaves only: nothing to toggle, so no OnToggle is no concern.
	renderDebug(t, TreeView{Nodes: []TreeNode{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}})
}

// A branch tapped with no OnToggle is a reported bug and not a panic.
func TestTreeViewNilOnToggleDoesNotPanic(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := TreeView{Nodes: treeFiles}.Render(ctx)
	ctx.EndRenderPass()
	ctx.TriggerCallback(treeButton(n, "docs").Props["onClick"].(string))
}

// The widget holds no hook, which is what lets a branch open and shut (and a
// caller render the tree conditionally) without shifting a neighbour's slot.
// Pinned the way debug mode pins it: a state declared after the tree keeps
// its value across passes that draw different amounts of tree, and no
// cursor-drift concern is raised.
func TestTreeViewOpeningABranchMovesNoHookSlot(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })

	ctx := core.NewContext()
	pass := func(open map[string]bool, initial string) string {
		// The driver's sequence (see renderPass): the cursor is rewound by
		// Reset, which a pass over a bare context would otherwise skip.
		ctx.BeginRenderPass()
		ctx.Reset()
		defer ctx.EndRenderPass()
		TreeView{Nodes: treeFiles, Expanded: open, OnToggle: func(string) {}}.Render(ctx)
		after := core.NewState(ctx, initial)
		return after.Get()
	}

	pass(nil, "first")
	if got := pass(map[string]bool{"docs": true, "docs/api": true}, "second"); got != "first" {
		t.Errorf("the slot after the tree moved when branches opened: %q", got)
	}
	if got := pass(nil, "third"); got != "first" {
		t.Errorf("the slot after the tree moved when branches shut: %q", got)
	}
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("concerns raised:\n%s", dump)
	}
}
