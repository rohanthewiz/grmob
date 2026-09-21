package comps

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/reconcile"
)

// The harness is select_row_test.go's: one context and one pass per render,
// because EditableGrid owns hooks (the editor, the landing cell, the menu).

// gridChange is one OnChange call.
type gridChange struct {
	row, col int
	value    string
}

// gridHarness holds the caller's half of the contract: the rows, their ids,
// and a record of what the grid reported. OnChange writes back, so the next
// pass draws what a real caller would draw.
type gridHarness struct {
	*rowHarness
	rows    [][]string
	ids     []string
	changes []gridChange
	inserts []int
	deletes []int
}

// budgetColumns is the fixture every test shares: one column of each kind.
//
//	Item (text) │ Category (choice) │ Amount (number, ≥ 0, "$") │ Paid (bool)
func budgetColumns() []GridColumn {
	return []GridColumn{
		{Title: "Item", Weight: 2},
		{Title: "Category", Kind: GridChoice, Options: []string{"Home", "Food"}},
		{Title: "Amount", Kind: GridNumber,
			Format: func(v string) string { return "$" + v },
			Validate: func(v string) string {
				if strings.HasPrefix(v, "-") {
					return "Amount cannot be negative"
				}
				return ""
			}},
		{Title: "Paid", Kind: GridBool, Width: 56},
	}
}

func newGridHarness(t *testing.T, tweak func(*EditableGrid)) *gridHarness {
	t.Helper()
	g := &gridHarness{
		rows: [][]string{
			{"Rent", "Home", "1200", "true"},
			{"Bread", "Food", "4.5", "false"},
			{"Milk", "Food", "2", "false"},
		},
		ids: []string{"a", "b", "c"},
	}
	g.rowHarness = newRowHarness(t, func() core.View {
		grid := EditableGrid{
			Label:   "Budget",
			Columns: budgetColumns(),
			Rows:    g.rows,
			Key:     func(r int) string { return g.ids[r] },
			OnChange: func(r, c int, v string) {
				g.changes = append(g.changes, gridChange{r, c, v})
				// Replaced, not mutated: the caller's half of the state rule.
				next := make([][]string, len(g.rows))
				for i := range g.rows {
					next[i] = append([]string(nil), g.rows[i]...)
				}
				next[r][c] = v
				g.rows = next
			},
		}
		if tweak != nil {
			tweak(&grid)
		}
		return grid
	})
	return g
}

// cell finds a body cell by the start of its spoken name.
func (g *gridHarness) cell(prefix string) *core.Node {
	g.t.Helper()
	n := findFirst(g.node, func(n *core.Node) bool {
		role := n.Style.AccessibilityRole
		return (role == core.RoleGridCell || role == core.RoleCell) &&
			strings.HasPrefix(n.Style.AccessibilityLabel, prefix)
	})
	if n == nil {
		g.t.Fatalf("no cell named %q…", prefix)
	}
	return n
}

// editor is the one Input in the tree, or nil.
func (g *gridHarness) editor() *core.Node {
	return findFirst(g.node, func(n *core.Node) bool { return n.Type == "Input" })
}

func (g *gridHarness) tap(n *core.Node) {
	g.t.Helper()
	id, ok := n.Props["onClick"].(string)
	if !ok {
		g.t.Fatalf("%q has no onClick", n.Style.AccessibilityLabel)
	}
	g.ctx.TriggerCallback(id)
	g.render()
}

func (g *gridHarness) typeText(s string) {
	g.t.Helper()
	g.ctx.TriggerTextCallback(g.editor().Props["onChange"].(string), s)
	g.render()
}

func (g *gridHarness) fire(prop string) {
	g.t.Helper()
	g.ctx.TriggerCallback(g.editor().Props[prop].(string))
	g.render()
}

// alert is the text of the message line under the grid.
func (g *gridHarness) alert() (text string, hidden bool) {
	n := findFirst(g.node, func(n *core.Node) bool { return n.Style.AccessibilityRole == core.RoleAlert })
	if n == nil {
		g.t.Fatal("no alert line")
	}
	return n.Props["content"].(string), n.Style.Display == core.DisplayNone
}

// The structure is the one ARIA describes: a grid that owns a header row and
// a rowgroup, a rowgroup that owns rows, rows that own cells. The message and
// the sheet are outside it.
func TestEditableGridStructureAndRoles(t *testing.T) {
	g := newGridHarness(t, func(e *EditableGrid) { e.RowHeaders = true })

	grid := findFirst(g.node, func(n *core.Node) bool { return n.Style.AccessibilityRole == core.RoleGrid })
	if grid == nil || grid.Style.AccessibilityLabel != "Budget" {
		t.Fatalf("no grid named Budget: %+v", grid)
	}
	if len(grid.Children) != 2 {
		t.Fatalf("a grid owns a header row and a rowgroup, got %d children", len(grid.Children))
	}
	header, body := grid.Children[0], grid.Children[1]
	if header.Style.AccessibilityRole != core.RoleRow || len(header.Children) != 5 {
		t.Fatalf("header: role %q, %d cells (want the corner and four titles)",
			header.Style.AccessibilityRole, len(header.Children))
	}
	for i, h := range header.Children {
		if h.Style.AccessibilityRole != core.RoleColumnHeader {
			t.Errorf("header cell %d is a %q", i, h.Style.AccessibilityRole)
		}
	}
	if body.Type != "List" || body.Style.AccessibilityRole != core.RoleRowGroup || len(body.Children) != 3 {
		t.Fatalf("body: %s, role %q, %d rows", body.Type, body.Style.AccessibilityRole, len(body.Children))
	}
	for i, row := range body.Children {
		if row.Key != g.ids[i] || row.Style.AccessibilityRole != core.RoleRow {
			t.Errorf("row %d: key %q role %q", i, row.Key, row.Style.AccessibilityRole)
		}
		// Every child of a row is a cell of one kind or the other.
		for j, c := range row.Children {
			if r := c.Style.AccessibilityRole; r != core.RoleGridCell && r != core.RoleCell {
				t.Errorf("row %d child %d is a %q", i, j, r)
			}
		}
	}

	// The role says whether the cell is the control or holds one.
	for prefix, want := range map[string]core.Role{
		"Item, row 1, Rent":       core.RoleGridCell, // press to edit
		"Amount, row 1, $1200":    core.RoleGridCell, // drawn through Format
		"Paid, row 1, checked":    core.RoleGridCell, // press to toggle
		"Paid, row 2, not checke": core.RoleGridCell,
		"Row 2":                   core.RoleCell, // a number, no menu
	} {
		if got := g.cell(prefix).Style.AccessibilityRole; got != want {
			t.Errorf("%q is a %q, want %q", prefix, got, want)
		}
	}
	sel := findFirst(g.node, func(n *core.Node) bool { return n.Type == "Select" })
	if sel == nil || sel.Style.AccessibilityLabel != "Category, row 1" {
		t.Fatalf("the choice cell holds a named Select, got %+v", sel)
	}
	if g.editor() != nil {
		t.Error("no cell is a field until one is edited")
	}
	if _, hidden := g.alert(); !hidden {
		t.Error("the alert line is hidden with nothing to say")
	}
}

// A column is one width down the sheet only if the header cell and every body
// cell take the same share.
func TestEditableGridHeaderAndBodyShareSizing(t *testing.T) {
	g := newGridHarness(t, nil)
	grid := findFirst(g.node, func(n *core.Node) bool { return n.Style.AccessibilityRole == core.RoleGrid })
	header, row := grid.Children[0], grid.Children[1].Children[0]
	for c := range header.Children {
		h, b := header.Children[c].Style, row.Children[c].Style
		if h.FlexGrow != b.FlexGrow || h.Width != b.Width || h.FlexBasis != b.FlexBasis {
			t.Errorf("column %d: header grow %v width %q, body grow %v width %q",
				c, h.FlexGrow, h.Width, b.FlexGrow, b.Width)
		}
	}
	if w := header.Children[3].Style.Width; w != "56px" {
		t.Errorf("a Width column is fixed, got %q", w)
	}
	if got := header.Children[1].Style.FlexGrow; got != 1 {
		t.Errorf("a column with neither Width nor Weight shares at 1, got %v", got)
	}
}

// Tap, type, return: one OnChange, with the draft, and the editor gone. The
// keystrokes in between reach nobody.
func TestEditableGridEditCommitsOnceOnReturn(t *testing.T) {
	g := newGridHarness(t, nil)
	g.tap(g.cell("Item, row 1"))

	ed := g.editor()
	if ed == nil || ed.Props["value"] != "Rent" {
		t.Fatalf("the editor opens on the stored value, got %+v", ed)
	}
	if ed.Props["focusAction"] != "focus" {
		t.Errorf("entering EDIT focuses the field, got %v", ed.Props["focusAction"])
	}
	g.typeText("Ren")
	g.typeText("Rent 2")
	if len(g.changes) != 0 {
		t.Fatalf("a keystroke is not a commit: %+v", g.changes)
	}
	if got := g.editor().Props["value"]; got != "Rent 2" {
		t.Fatalf("the field draws the draft, got %v", got)
	}

	g.fire("onSubmit")
	if len(g.changes) != 1 || g.changes[0] != (gridChange{0, 0, "Rent 2"}) {
		t.Fatalf("changes = %+v", g.changes)
	}
	if g.editor() != nil {
		t.Error("the editor closes on commit")
	}
	// The return key moves down a row: the cell below carries the landing
	// focus command.
	if got := g.cell("Item, row 2").Props["focusAction"]; got != "focus" {
		t.Errorf("row 2's cell should be focused after the commit, got %v", got)
	}
	if got := g.cell("Item, row 1, Rent 2").Props["focusAction"]; got == "focus" {
		t.Error("the committed cell is not the landing cell")
	}
}

// A number column opens on the raw value, not the formatted one, asks for the
// decimal pad, and reports what was typed.
func TestEditableGridNumberEditsTheRawValue(t *testing.T) {
	g := newGridHarness(t, nil)
	g.tap(g.cell("Amount, row 2"))
	ed := g.editor()
	if ed.Props["value"] != "4.5" {
		t.Errorf("editor value = %v, want the stored 4.5 and not $4.5", ed.Props["value"])
	}
	if ed.Props["keyboard"] != string(core.KeyboardDecimal) {
		t.Errorf("keyboard = %v", ed.Props["keyboard"])
	}
	g.typeText(" 5.25 ")
	g.fire("onSubmit")
	if len(g.changes) != 1 || g.changes[0] != (gridChange{1, 2, "5.25"}) {
		t.Fatalf("changes = %+v", g.changes)
	}
	if g.cell("Amount, row 2, $5.25") == nil {
		t.Error("the committed value is drawn through Format")
	}
}

// A refused commit keeps the cell in EDIT with the message in the alert, and
// the caller hears nothing. Typing clears the message; a good value commits.
func TestEditableGridValidateKeepsTheEditor(t *testing.T) {
	g := newGridHarness(t, nil)
	g.tap(g.cell("Amount, row 1"))

	g.typeText("12x")
	g.fire("onSubmit")
	if msg, hidden := g.alert(); hidden || !strings.Contains(msg, "not a number") {
		t.Fatalf("alert = %q hidden=%v", msg, hidden)
	}
	if g.editor() == nil || len(g.changes) != 0 {
		t.Fatalf("a refused commit stays in EDIT and reports nothing: %+v", g.changes)
	}

	g.typeText("-3")
	if _, hidden := g.alert(); !hidden {
		t.Error("typing answers the message, which clears")
	}
	g.fire("onSubmit")
	if msg, _ := g.alert(); msg != "Amount cannot be negative" {
		t.Fatalf("the column's Validate runs after the kind's: %q", msg)
	}

	g.typeText("3")
	g.fire("onSubmit")
	if len(g.changes) != 1 || g.changes[0].value != "3" || g.editor() != nil {
		t.Fatalf("changes = %+v, editor = %v", g.changes, g.editor())
	}
}

// ✕ discards: no OnChange, the stored value back, focus on the same cell.
func TestEditableGridDiscard(t *testing.T) {
	g := newGridHarness(t, nil)
	g.tap(g.cell("Item, row 2"))
	g.typeText("Baguette")

	x := findFirst(g.node, func(n *core.Node) bool { return n.Style.AccessibilityLabel == "Discard edit" })
	if x == nil || x.Style.AccessibilityRole != core.RoleButton {
		t.Fatalf("the editing cell ends in a discard button, got %+v", x)
	}
	g.tap(x)
	if len(g.changes) != 0 || g.editor() != nil {
		t.Fatalf("discard reported %+v", g.changes)
	}
	if got := g.cell("Item, row 2, Bread").Props["focusAction"]; got != "focus" {
		t.Errorf("focus returns to the discarded cell, got %v", got)
	}
}

// A blur commits after the grace, and not before: a ✕ pressed inside it still
// discards. See "Why a blur waits".
func TestEditableGridBlurCommitsAfterTheGrace(t *testing.T) {
	old := gridBlurGrace
	gridBlurGrace = 20 * time.Millisecond
	t.Cleanup(func() { gridBlurGrace = old })

	g := newGridHarness(t, nil)
	g.tap(g.cell("Item, row 3"))
	g.typeText("Oat milk")
	g.fire("onBlur")
	if len(g.changes) != 0 || g.editor() == nil {
		t.Fatalf("a blur does not commit at once: %+v", g.changes)
	}
	time.Sleep(80 * time.Millisecond)
	g.render()
	if len(g.changes) != 1 || g.changes[0] != (gridChange{2, 0, "Oat milk"}) || g.editor() != nil {
		t.Fatalf("after the grace: changes %+v, editor %v", g.changes, g.editor() != nil)
	}
	if got := g.cell("Item, row 3").Props["focusAction"]; got == "focus" {
		t.Error("a blur's commit does not take focus back: the reader put it elsewhere")
	}

	// Second edit: blur, then ✕ inside the grace.
	g.tap(g.cell("Item, row 3"))
	g.typeText("discard me")
	g.fire("onBlur")
	g.tap(findFirst(g.node, func(n *core.Node) bool { return n.Style.AccessibilityLabel == "Discard edit" }))
	time.Sleep(80 * time.Millisecond)
	g.render()
	if len(g.changes) != 1 {
		t.Fatalf("the ✕ won the race and the timer found nothing to commit: %+v", g.changes)
	}
}

// Tapping another cell commits the open one and edits the new one, in one
// dispatch.
func TestEditableGridTapAnotherCellCommitsFirst(t *testing.T) {
	g := newGridHarness(t, nil)
	g.tap(g.cell("Item, row 1"))
	g.typeText("Mortgage")
	g.tap(g.cell("Item, row 2"))

	if len(g.changes) != 1 || g.changes[0] != (gridChange{0, 0, "Mortgage"}) {
		t.Fatalf("changes = %+v", g.changes)
	}
	if ed := g.editor(); ed == nil || ed.Props["value"] != "Bread" {
		t.Fatalf("the editor moved to row 2, got %+v", ed)
	}

	// Unless the open one is refused, which holds the reader there.
	g.tap(g.cell("Amount, row 3"))
	g.typeText("nope")
	g.tap(g.cell("Item, row 1"))
	if ed := g.editor(); ed == nil || ed.Props["value"] != "nope" {
		t.Fatalf("a refused commit keeps its editor, got %+v", ed)
	}
}

// An unchanged commit is not a change.
func TestEditableGridUnchangedCommitIsSilent(t *testing.T) {
	g := newGridHarness(t, nil)
	g.tap(g.cell("Item, row 1"))
	g.fire("onSubmit")
	if len(g.changes) != 0 {
		t.Fatalf("changes = %+v", g.changes)
	}
}

// Bool toggles with one tap and no editor; Choice reports the pick.
func TestEditableGridBoolAndChoice(t *testing.T) {
	g := newGridHarness(t, nil)
	g.tap(g.cell("Paid, row 2"))
	if len(g.changes) != 1 || g.changes[0] != (gridChange{1, 3, "true"}) || g.editor() != nil {
		t.Fatalf("changes = %+v", g.changes)
	}
	if g.cell("Paid, row 2, checked") == nil {
		t.Error("the toggled cell says so")
	}

	sel := findFirst(g.node, func(n *core.Node) bool {
		return n.Type == "Select" && n.Style.AccessibilityLabel == "Category, row 3"
	})
	g.ctx.TriggerTextCallback(sel.Props["onChange"].(string), "Home")
	g.render()
	if len(g.changes) != 2 || g.changes[1] != (gridChange{2, 1, "Home"}) {
		t.Fatalf("changes = %+v", g.changes)
	}
}

// A value outside Options is offered as itself, so the picker does not draw
// the first option over it.
func TestEditableGridChoiceKeepsAForeignValue(t *testing.T) {
	g := newGridHarness(t, nil)
	g.rows = append(g.rows, []string{"", "", "", ""})
	g.ids = append(g.ids, "d")
	g.render()
	sel := findFirst(g.node, func(n *core.Node) bool {
		return n.Type == "Select" && n.Style.AccessibilityLabel == "Category, row 4"
	})
	opts := sel.Props["options"].([]map[string]string)
	if len(opts) != 3 || opts[0]["value"] != "" || opts[0]["label"] != "—" {
		t.Fatalf("options = %+v", opts)
	}
}

// Read only, by column and by cell: a plain cell, no handler, and it says so.
func TestEditableGridReadOnly(t *testing.T) {
	g := newGridHarness(t, func(e *EditableGrid) {
		e.Columns[0].ReadOnly = true
		e.ReadOnly = func(r, c int) bool { return r == 0 && c == 3 }
	})
	for _, name := range []string{"Item, row 1, Rent, read only", "Paid, row 1, checked, read only"} {
		c := g.cell(name)
		if c.Style.AccessibilityRole != core.RoleCell || c.Props["onClick"] != nil {
			t.Errorf("%q: role %q onClick %v", name, c.Style.AccessibilityRole, c.Props["onClick"])
		}
	}
	if g.cell("Paid, row 2").Props["onClick"] == nil {
		t.Error("the per-cell flag closes one cell, not the column")
	}
}

// The row menu: the header is the trigger, the caller does the work, and the
// open editor follows its row by key across the change.
func TestEditableGridRowMenuAndKeys(t *testing.T) {
	var g *gridHarness
	g = newGridHarness(t, func(e *EditableGrid) {
		e.OnInsertRow = func(after int) { g.inserts = append(g.inserts, after) }
		e.OnDeleteRow = func(row int) {
			g.deletes = append(g.deletes, row)
			g.rows = append(append([][]string(nil), g.rows[:row]...), g.rows[row+1:]...)
			g.ids = append(append([]string(nil), g.ids[:row]...), g.ids[row+1:]...)
		}
	})
	head := g.cell("Row 1")
	if head.Style.AccessibilityRole != core.RoleGridCell {
		t.Fatalf("a row header with a menu is a control, got %q", head.Style.AccessibilityRole)
	}
	sheet := func() *core.Node {
		return findFirst(g.node, func(n *core.Node) bool { return n.Type == "Modal" })
	}
	if sheet().Props["visible"] == true {
		t.Fatal("the menu starts shut")
	}

	// Edit row 3, then delete row 1 from its menu. Opening the menu commits.
	g.tap(g.cell("Item, row 3"))
	g.typeText("Oat milk")
	g.tap(head)
	if len(g.changes) != 1 || g.editor() != nil {
		t.Fatalf("opening the menu commits the open edit: %+v", g.changes)
	}
	if sheet().Props["visible"] != true {
		t.Fatal("the header opens the menu")
	}
	del := findFirst(sheet(), func(n *core.Node) bool { return n.Type == "Button" && n.Props["label"] == "Delete" })
	if del == nil {
		t.Fatal("no Delete in the menu")
	}
	g.tap(del)
	if len(g.deletes) != 1 || g.deletes[0] != 0 {
		t.Fatalf("deletes = %v", g.deletes)
	}
	if sheet().Props["visible"] == true {
		t.Error("an item shuts the menu")
	}

	// An editor open on "c" stays on "c" when the row above it goes.
	g.tap(g.cell("Item, row 2")) // "c" is row 2 now
	g.rows, g.ids = g.rows[1:], g.ids[1:]
	g.render()
	if ed := g.editor(); ed == nil || ed.Style.AccessibilityLabel != "Item, row 1" {
		t.Fatalf("the editor followed its row's key, got %+v", ed)
	}
}

func TestEditableGridConcerns(t *testing.T) {
	cases := []struct {
		name string
		grid EditableGrid
		want string
	}{
		{"inert", EditableGrid{Columns: []GridColumn{{Title: "A"}}, Rows: [][]string{{"x"}}},
			ConcernEditableGridInert},
		{"ragged", EditableGrid{Columns: []GridColumn{{Title: "A"}, {Title: "B"}}, Rows: [][]string{{"x"}},
			OnChange: func(int, int, string) {}}, ConcernEditableGridRagged},
		{"no key", EditableGrid{Columns: []GridColumn{{Title: "A"}}, Rows: [][]string{{"x"}},
			OnChange: func(int, int, string) {}, OnDeleteRow: func(int) {}}, ConcernEditableGridNoKey},
		{"no options", EditableGrid{Columns: []GridColumn{{Title: "A", Kind: GridChoice}}, Rows: [][]string{{"x"}},
			OnChange: func(int, int, string) {}}, ConcernEditableGridChoiceNoOptions},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newQuietRowHarness(t, func() core.View { return tc.grid })
			_ = h
			if dump := core.DumpConcerns(); !strings.Contains(dump, tc.want) {
				t.Errorf("want %s in:\n%s", tc.want, dump)
			}
		})
	}

	// A wholly read-only grid with no OnChange is a viewer, and quiet.
	newRowHarness(t, func() core.View {
		return EditableGrid{Columns: []GridColumn{{Title: "A", ReadOnly: true}}, Rows: [][]string{{"x"}}}
	})
}

// The cost claim: a changed value patches that cell and nothing else. Two
// patches, the text and the cell's spoken name, both under one cell's path.
func TestEditableGridChangedValuePatchesOneCell(t *testing.T) {
	g := newGridHarness(t, nil)
	before := g.node
	next := [][]string{g.rows[0], {"Sourdough", "Food", "4.5", "false"}, g.rows[2]}
	g.rows = next
	g.render()

	patches := reconcile.Diff(before, g.node, "root")
	if len(patches) == 0 || len(patches) > 2 {
		t.Fatalf("got %d patches, want the text and the label of one cell:\n%+v", len(patches), patches)
	}
	cell := ""
	for _, p := range patches {
		if !strings.HasPrefix(p.Type, "update-") {
			t.Errorf("a changed value is an update, got %s at %s", p.Type, p.TargetID)
		}
		// The cell is five levels under the root: root/grid/list/row/cell.
		parts := strings.Split(p.TargetID, "/")
		if len(parts) < 5 {
			t.Fatalf("patch at %s is above the cells", p.TargetID)
		}
		at := strings.Join(parts[:5], "/")
		if cell != "" && at != cell {
			t.Errorf("patches touch two cells: %s and %s", cell, at)
		}
		cell = at
	}
}

// The size the doc states, measured: build and diff a 30 × 1000 sheet.
func BenchmarkEditableGrid30x1000(b *testing.B) {
	const cols, rows = 30, 1000
	columns := make([]GridColumn, cols)
	for c := range columns {
		columns[c] = GridColumn{Title: "C" + strconv.Itoa(c), Width: 80}
	}
	data := make([][]string, rows)
	for r := range data {
		data[r] = make([]string, cols)
		for c := range data[r] {
			data[r][c] = fmt.Sprintf("%d:%d", r, c)
		}
	}
	grid := EditableGrid{Columns: columns, Rows: data, OnChange: func(int, int, string) {}}
	ctx := core.NewContext()
	pass := func() *core.Node {
		ctx.BeginRenderPass()
		ctx.Reset()
		n := grid.Render(ctx)
		ctx.EndRenderPass()
		return n
	}
	prev := pass()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		next := pass()
		reconcile.Diff(prev, next, "root")
		prev = next
	}
}
