package comps

import (
	"fmt"
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernTreeViewDuplicateID is raised, in debug builds only, when two nodes
// of one TreeView share an ID, at any depth. Expanded, OnToggle, OnSelect and
// Selected all speak in IDs, so a shared one opens two branches with one tap
// and marks two rows as the chosen one.
const ConcernTreeViewDuplicateID = "tree-view-duplicate-id"

// ConcernTreeViewInert is raised, in debug builds only, when a TreeView has a
// branch and no OnToggle. The chevron promises that the row opens, and with
// no handler nothing a user does can open it.
const ConcernTreeViewInert = "tree-view-inert"

// TreeNode is one row of a TreeView, and everything under it.
type TreeNode struct {
	// ID names the node to Expanded, OnToggle, OnSelect and Selected. It must
	// be unique in the whole tree and not only among siblings: the caller's
	// Expanded map has one key space.
	ID string

	// Label is the row's text and its spoken name.
	Label string

	// Leading is drawn between the chevron column and the label: a folder or
	// file icon. It is decoration and is hidden from accessibility; say what
	// it means in Label if it means something.
	Leading core.View

	// Children are the nodes under this one. A node with any is a branch.
	Children []TreeNode

	// Branch makes a node with no Children a branch all the same: an empty
	// folder, or one whose children are fetched when it first opens. Without
	// it such a node would draw as a leaf and could never be asked to open.
	Branch bool
}

// isBranch reports whether the node is drawn with a chevron and toggles.
func (n TreeNode) isBranch() bool { return n.Branch || len(n.Children) > 0 }

// TreeView is an indented, expandable hierarchy: a file browser, a document
// outline, a category picker.
//
//	comps.TreeView{
//	    Label:    "Project files",
//	    Nodes:    files,
//	    Expanded: open.Get(),                  // map[string]bool, the caller's
//	    OnToggle: func(id string) { … },       // flip open[id], Set a copy
//	    Selected: chosen.Get(),
//	    OnSelect: chosen.Set,
//	}
//
//	┌ Column  role=list  "Project files" ────────────────────────────────┐
//	│ ┌ Box  role=listitem  level=1 ───────────────────────────────────┐ │
//	│ │ Row  role=button  expanded   "docs"          ▾ 📁 docs         │ │
//	│ │ ┌ Column  role=list  (after an Indent-wide spacer) ──────────┐ │ │
//	│ │ │ Box  role=listitem  level=2                                │ │ │
//	│ │ │   Row  role=button  current   "guide.md"      📄 guide.md  │ │ │
//	│ │ └────────────────────────────────────────────────────────────┘ │ │
//	│ └────────────────────────────────────────────────────────────────┘ │
//	│ ┌ Box  role=listitem  level=1 ───────────────────────────────────┐ │
//	│ │ Row  role=button  collapsed  "src"           ▸ 📁 src          │ │
//	│ └────────────────────────────────────────────────────────────────┘ │
//	└────────────────────────────────────────────────────────────────────┘
//
// # A branch toggles, a leaf selects
//
// One row is one target, so a tap has one meaning: a branch row calls
// OnToggle and a leaf row calls OnSelect. A branch that could also be chosen
// would need two targets in one row (the chevron and the words), which is two
// buttons with the same name for a screen reader and a 20pt chevron for a
// thumb. A picker whose categories can themselves be chosen can say so in
// data: give the branch a first child that stands for it ("All of Fiction").
//
// # The caller owns what is open
//
// Expanded is the caller's map and OnToggle only reports an ID. Which folders
// are open is navigation state: an app wants to restore it, to open the path
// to a search hit, to collapse everything. A widget that kept the map in a
// hook could offer none of those, and would also have to hold a hook, which
// would forbid rendering a TreeView conditionally. It holds none.
//
// The map must be replaced and not mutated in place (copy, flip, Set), as any
// state value must.
//
// # Shut branches are not rendered
//
// A shut branch's children are absent from the tree, not hidden in it, so a
// tree of ten thousand nodes costs what its open part costs. The price is
// that a node's own transient native state does not survive its parent
// closing; a TreeNode is a label, so it has none.
//
// # The roles: nested lists now, a tree later
//
// ARIA's pattern for this widget is `tree` and `treeitem`, and core leaves
// the pair out on purpose (core/role.go, "The depth question"): it is a
// third collection pattern with its own keyboard contract (one tab stop,
// Up/Down to walk, Right/Left to open and shut), and that contract is the
// web runtime's to supply. Stamping the roles without it would announce a
// tree and then not behave like one.
//
// So the structure is the one HTML itself uses for an outline, a list whose
// items hold lists: RoleList, RoleListItem with core.AccessibilityNestingLevel
// and, inside each item, a button. The button is inside the item rather than
// being it, because a list's child must be a listitem and a listitem is not a
// control. A branch's button is comps' shared disclosure shape, so it states
// expanded or collapsed on every pass; a reader hears "docs, collapsed,
// button, level 1". Every branch and leaf is a tab stop, which is slower
// than a tree's arrows and is never wrong.
//
// The roles are not in the API. Swapping them for `tree` and `treeitem`, if
// core gains the pair, changes no caller.
//
// On the two phones the difference does not exist: VoiceOver and TalkBack
// walk by swipe. Neither has a nesting-depth property, so the level reaches
// the web only (see core.Style's AccessibilityNestingLevel), and the indent
// is what says depth on a phone.
//
// # The chosen leaf
//
// Selected's row states core.CurrentTrue, which is aria-current on the web
// and the selected state on both natives. It is not aria-selected, which ARIA
// allows on an option, a row, a tab or a gridcell, and never on a button.
//
// # Theme roles read
//
//	Label          Typography.Body, Colors.TextPrimary
//	Chosen leaf    bold, Colors.PrimaryOnLightColor(), on Colors.Surface
//	Chevron        Typography.Body, Colors.TextSecondary
//	Row padding    Spacing.XS by Spacing.SM; gap Spacing.SM
//	Indent         Spacing.LG per level, unless Indent is set
type TreeView struct {
	// Nodes are the top-level nodes, in order.
	Nodes []TreeNode

	// Expanded says which branches are open, by ID. A missing ID is shut, so
	// a nil map is a fully collapsed tree.
	Expanded map[string]bool

	// OnToggle receives the ID of a tapped branch. The caller flips its entry
	// in Expanded. Nil with any branch in Nodes reports ConcernTreeViewInert.
	OnToggle func(id string)

	// OnSelect receives the ID of a tapped leaf. Nil draws leaves as text,
	// which is right for an outline that is only read.
	OnSelect func(id string)

	// Selected is the ID of the chosen leaf, or "" for none. A Selected that
	// names a branch, or a leaf inside a shut branch, marks nothing.
	Selected string

	// Indent is how far each level sits to the right of its parent, in
	// points. Zero takes the theme's Spacing.LG. An int because it is spent
	// as a padding, and core's paddings are whole points.
	Indent int

	// Label names the outer list for a screen reader ("Project files").
	Label string

	// Style is applied to the outer column after the widget's own props.
	Style []core.StyleProp
}

// treeChevronWidth is the width of the chevron column, in points. It is
// fixed, and a leaf keeps the column empty, so a leaf's label starts where
// its sibling branches' labels do. Sized from the glyph: "▸" and "▾" differ
// in width in most fonts, and a column that took the glyph's width would
// shift the label by a pixel or two on every toggle.
const treeChevronWidth = 20

// treeRowMinHeight is the least height of a row, in points: the 44pt target
// both phone platforms ask of anything tappable. Rows that are only read take
// it too, so a tree does not change its rhythm when OnSelect is set.
const treeRowMinHeight = 44

// Render builds the nested lists. It takes no hook slot.
func (tv TreeView) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	if core.IsDebugMode() {
		tv.audit()
	}

	items := make([]core.PropsAndChildren, 0, len(tv.Nodes)+len(tv.Style)+4)
	items = append(items,
		// A Column arrives with the theme's screen inset. The tree is a part
		// of a screen and not one, so the inset is the caller's to add.
		core.Padding(0),
		core.Gap(0),
		core.AccessibilityRole(core.RoleList),
	)
	if tv.Label != "" {
		items = append(items, core.AccessibilityLabel(tv.Label))
	}
	items = append(items, asProps(tv.Style)...)
	for _, n := range tv.Nodes {
		items = append(items, tv.item(t, n, 1))
	}
	return core.Column(items...).Render(ctx)
}

// audit reports the two concerns. It walks every node, shut branches
// included: a duplicate inside a shut branch is a bug that waits for the
// branch to open, and debug mode exists to find it before a user does.
func (tv TreeView) audit() {
	seen := map[string]bool{}
	branches := false
	var walk func(nodes []TreeNode)
	walk = func(nodes []TreeNode) {
		for _, n := range nodes {
			if seen[n.ID] {
				core.ReportConcern(ConcernTreeViewDuplicateID, fmt.Sprintf(
					"TreeView has two nodes with ID %q; Expanded, Selected and both callbacks name a node by ID, so IDs must be unique in the whole tree",
					n.ID))
			}
			seen[n.ID] = true
			branches = branches || n.isBranch()
			walk(n.Children)
		}
	}
	walk(tv.Nodes)

	if branches && tv.OnToggle == nil {
		core.ReportConcern(ConcernTreeViewInert,
			"TreeView has a branch and no OnToggle, so no branch can be opened or shut; set OnToggle and flip the ID in Expanded")
	}
}

// item builds one listitem: the node's row and, for an open branch, the list
// of its children. level is the node's depth, 1 at the top.
//
// The item is keyed by the node's ID. Siblings come and go with the caller's
// data (a file is added, a folder is fetched), and a key keeps an insertion
// from re-propping every row after it.
func (tv TreeView) item(t *core.Theme, n TreeNode, level int) core.View {
	item := []core.PropsAndChildren{
		core.AccessibilityRole(core.RoleListItem),
		core.AccessibilityNestingLevel(level),
	}

	if !n.isBranch() {
		item = append(item, tv.leafRow(t, n))
		return core.Keyed(n.ID, core.Box(item...))
	}

	open := tv.Expanded[n.ID]
	item = append(item, tv.branchRow(t, n, open))
	if open && len(n.Children) > 0 {
		indent := tv.Indent
		if indent <= 0 {
			indent = t.Spacing.LG
		}
		// The nested list sits inside its parent's item, as a <ul> sits in an
		// <li>, so the depth is in the structure as well as in the level. The
		// indent is spent once per list, and so it accumulates: level 3 is
		// inside two indented lists without any row knowing its own depth.
		sub := make([]core.PropsAndChildren, 0, len(n.Children)+5)
		sub = append(sub,
			core.Padding(0),
			core.Gap(0),
			core.FlexGrow(1),
			core.AccessibilityRole(core.RoleList),
			// Named by its branch, so a reader that lists the lists on a
			// page does not meet a run of unnamed ones.
			core.AccessibilityLabel(n.Label),
		)
		for _, c := range n.Children {
			sub = append(sub, tv.item(t, c, level+1))
		}
		// The indent is a spacer leading a Row and not a left padding. A Row
		// lays out from the reading direction's start on every target, so
		// under RTL the tree indents from the right with nothing said here;
		// core.PaddingLeft is a physical side and would indent the wrong one.
		//
		//	Row ─┬─ Box (Indent wide, hidden)
		//	     └─ Column role=list, grows
		item = append(item, core.Row(
			core.Padding(0),
			core.Gap(0),
			core.Box(
				core.Width(strconv.Itoa(indent)+"px"),
				core.FlexShrink(0),
				core.AccessibilityHidden(),
			),
			core.Column(sub...),
		))
	}
	return core.Keyed(n.ID, core.Box(item...))
}

// rowStyle is the geometry a branch row and a leaf row share.
func (tv TreeView) rowStyle(t *core.Theme) []core.StyleProp {
	return []core.StyleProp{
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.PaddingVertical(t.Spacing.XS),
		core.PaddingHorizontal(t.Spacing.SM),
		core.MinHeight(strconv.Itoa(treeRowMinHeight) + "px"),
	}
}

// branchRow is the shared disclosure button: chevron, Leading, label.
func (tv TreeView) branchRow(t *core.Theme, n TreeNode, open bool) core.View {
	id := n.ID // captured per node: the handler outlives this call
	toggle := func() {
		// Guarded here and not only by the concern: a nil OnToggle is a
		// reported bug in debug builds and must not be a panic in release.
		if tv.OnToggle != nil {
			tv.OnToggle(id)
		}
	}
	return disclosure{
		Label:    n.Label,
		Hint:     "Expands or collapses the branch",
		Expanded: open,
		OnToggle: toggle,
		// No heading wrapper. A tree of forty folders would put forty
		// entries in the heading outline, and the list structure already
		// says what the rows are.
		Heading: false,
		ChevronStyle: []core.StyleProp{
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextSecondary),
			core.Width(strconv.Itoa(treeChevronWidth) + "px"),
			core.FlexShrink(0),
		},
		ControlStyle: tv.rowStyle(t),
		Control:      tv.rowContent(t, n, false),
	}.view()
}

// leafRow is a leaf: a button when OnSelect is set, plain text when not.
func (tv TreeView) leafRow(t *core.Theme, n TreeNode) core.View {
	chosen := n.ID == tv.Selected && tv.Selected != ""

	row := make([]core.PropsAndChildren, 0, 12)
	row = append(row, asProps(tv.rowStyle(t))...)
	if chosen {
		row = append(row,
			core.BackgroundColor(t.Colors.Surface),
			core.BorderRadius(float64(t.Spacing.XS)),
		)
	}
	if tv.OnSelect != nil {
		id := n.ID // captured per node: the handler outlives this call
		row = append(row,
			core.OnClick(func() { tv.OnSelect(id) }),
			core.AccessibilityRole(core.RoleButton),
			core.AccessibilityLabel(n.Label),
		)
		// Stated only on a row that is a button: the natives fold "current"
		// into the selected state of a control, and a selected run of plain
		// text is announced as nothing in particular.
		if chosen {
			row = append(row, core.AccessibilityCurrent(core.CurrentTrue))
		}
	}
	// The empty chevron column; see treeChevronWidth.
	row = append(row, core.Box(
		core.Width(strconv.Itoa(treeChevronWidth)+"px"),
		core.FlexShrink(0),
		core.AccessibilityHidden(),
	))
	for _, v := range tv.rowContent(t, n, chosen) {
		row = append(row, v)
	}
	return core.Row(row...)
}

// rowContent is what follows the chevron column in either kind of row:
// Leading, if any, and the label.
func (tv TreeView) rowContent(t *core.Theme, n TreeNode, chosen bool) []core.View {
	color, weight := t.Colors.TextPrimary, core.Normal
	if chosen {
		color, weight = t.Colors.PrimaryOnLightColor(), core.Bold
	}
	out := make([]core.View, 0, 2)
	if n.Leading != nil {
		// Wrapped so the hiding does not depend on what the caller passed:
		// an icon Text would otherwise be read before every label.
		out = append(out, core.Box(core.FlexShrink(0), core.AccessibilityHidden(), n.Leading))
	}
	out = append(out, core.Text(n.Label,
		core.UseStyle(t.Typography.Body),
		core.TextColor(color),
		core.FontWeight(weight),
		core.FlexGrow(1),
	))
	return out
}
