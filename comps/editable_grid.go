package comps

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// ConcernEditableGridInert is raised, in debug builds only, when a grid has no
// OnChange and at least one cell that is not read only. Such a cell opens an
// editor, takes a draft, and throws it away at the commit: the grid looks
// editable and is not. A grid that is meant to be looked at sets ReadOnly (or
// every column's ReadOnly) and says so.
const ConcernEditableGridInert = "editable-grid-inert"

// ConcernEditableGridRagged is raised, in debug builds only, for a row whose
// length differs from Columns. A short row is drawn with empty cells and a
// long one loses its tail, so nothing crashes; but an edit to a cell past a
// short row's end reports a column the caller's row does not have.
const ConcernEditableGridRagged = "editable-grid-ragged"

// ConcernEditableGridNoKey is raised, in debug builds only, when OnInsertRow
// or OnDeleteRow is set without Key. Rows are then keyed by index, so a delete
// re-pairs every row below it with its neighbour's node, and an open editor
// stays at its index, which is now a different row. See "Rows" on the type.
const ConcernEditableGridNoKey = "editable-grid-no-key"

// ConcernEditableGridChoiceNoOptions is raised, in debug builds only, for a
// GridChoice column with no Options: a picker with nothing to pick.
const ConcernEditableGridChoiceNoOptions = "editable-grid-choice-no-options"

// GridCellKind chooses a column's editor, its soft keyboard and its default
// alignment. The value is a string whatever the kind: see "Cells are strings"
// on EditableGrid.
type GridCellKind int

const (
	// GridText is a free text cell, edited in a text field.
	GridText GridCellKind = iota

	// GridNumber is edited in a text field with the decimal keyboard, is
	// right-aligned unless Align says otherwise, and refuses a draft that
	// strconv.ParseFloat refuses ("" is allowed: an empty cell is not a bad
	// number). The decimal pad has no minus key on iOS, so a signed column
	// sets GridColumn.Keyboard to core.KeyboardText.
	GridNumber

	// GridBool is a checkbox glyph that toggles on tap and never opens an
	// editor. Its value is "true" or "false"; anything strconv.ParseBool
	// refuses is drawn unchecked.
	GridBool

	// GridChoice is a core.Select over Options: the platform's own picker,
	// always present in the cell, so choosing is one tap and not two.
	GridChoice
)

// GridColumn describes one column of an EditableGrid.
type GridColumn struct {
	// Title is the header cell's text and the stem of every cell's spoken
	// name in the column.
	Title string

	// Kind chooses the editor. The zero value is GridText.
	Kind GridCellKind

	// Options are a GridChoice column's values, in order.
	Options []string

	// Width fixes the column in px; Weight shares the row's slack. A column
	// with neither gets Weight 1, because a grid's header and body are
	// separate rows and a column that hugged its content would be a different
	// width on each of them. With both set, Width is the least the column
	// takes.
	Width, Weight float64

	// Align positions the cell's content on the row axis. The zero value is
	// the leading edge, except for GridNumber (the trailing edge) and
	// GridBool (the centre).
	Align core.JustifyContent

	// Format turns the stored value into the drawn one: "1234.5" to
	// "$1,234.50". Display only. The editor opens on the stored value, and
	// OnChange reports what was typed.
	Format func(string) string

	// Validate returns "" for a draft that may be committed, or the message
	// to show. It runs at the commit, not per key: a half-typed value is
	// allowed to be wrong.
	Validate func(string) string

	// Keyboard overrides the soft keyboard the Kind asks for.
	Keyboard core.KeyboardKind

	// ReadOnly makes every cell of the column a value and not a control.
	ReadOnly bool
}

// gridEdit is the open editor: which cell, the text typed so far, and the
// message of a refused commit. The zero value is "no editor".
//
// The row is held twice. key is the identity (EditableGrid.Key), which is
// what survives an insert or delete above; row is the index it was opened at,
// tried first so that the common case costs one Key call rather than a scan.
type gridEdit struct {
	on      bool
	key     string
	row     int
	col     int
	draft   string
	err     string
	blurred bool
}

// gridSpot names the cell that carries the landing FocusRef: the one a
// core.Focus goes to when an edit ends.
type gridSpot struct {
	key string
	col int
}

// gridMenu is the open row menu (N3). The row is held by key and index for
// the reason gridEdit's is.
type gridMenu struct {
	open bool
	key  string
	row  int
}

// gridBlurGrace is how long a blurred editor waits before committing itself.
// See "Why a blur waits" on EditableGrid. A variable so a test need not sleep
// a human's reaction time.
var gridBlurGrace = 150 * time.Millisecond

// EditableGrid is a spreadsheet-like table: a header over a windowed body of
// cells, where the unit is the cell and the point is editing it.
//
//	comps.EditableGrid{
//	    Label:   "Budget",
//	    Columns: []comps.GridColumn{
//	        {Title: "Item", Weight: 2},
//	        {Title: "Amount", Kind: comps.GridNumber, Format: dollars},
//	        {Title: "Paid", Kind: comps.GridBool, Width: 56},
//	    },
//	    Rows:     rows.Get(),
//	    Key:      func(i int) string { return ids.Get()[i] },
//	    OnChange: func(r, c int, v string) { rows.Set(with(rows.Get(), r, c, v)) },
//	}
//
// # Against DataTable
//
// DataTable[T] is a read-only view of typed rows: it sorts, groups and pages,
// and a row is the tap target. Cell editing bolted onto it would be one widget
// with two selection models (row and cell) and two role sets (table and grid).
// This borrows its column sizing and its keyed, windowed body and nothing
// else.
//
// # Cells are strings
//
// DataTable is generic because it reads rows through accessors. An editable
// cell needs a setter per column too, and a pair of closures per column is a
// heavy API for what a text field produces anyway. So Rows is [][]string,
// Kind chooses the editor, and the caller parses. A typed adapter can be
// layered over this without changing it.
//
// # One editor at a time
//
// Every cell is a box showing text, and only the cell being edited becomes a
// text field. A 50×10 sheet of real fields would be 500 native inputs, each
// with its own text-edit ledger and its own tab stop, and the grid's arrow
// keys would fight the caret's in every one.
//
//	          tap / Enter / Space (the cell's onClick)
//	┌──────────┐ ─────────────────────────────▶ ┌──────────┐
//	│ NAVIGATE │                                │   EDIT   │
//	│ cell is  │ ◀───────────────────────────── │ cell is  │
//	│ a button │   return key  → commit, move ↓ │ an Input │
//	└──────────┘   blur        → commit, stay   └──────────┘
//	               ✕           → discard draft
//	               another cell tapped → commit, edit that one
//
// The draft is the widget's: no application wants a half-typed cell, so
// OnChange fires once per commit and not per key, and only when the value
// changed. A commit Validate refuses keeps the cell in EDIT, tints its border
// with Error and puts the message under the grid in a core.RoleAlert line; the
// caller never receives a refused value.
//
// Cancel has no key. Key events do not reach Go (the PINInput finding), so
// Escape cannot discard a draft; the editing cell ends in a ✕ that does.
//
// # Why a blur waits
//
// A blur commits after gridBlurGrace, not at once. In a browser a press on
// the ✕ blurs the field before the click is delivered, and a commit in
// between would remove the ✕ from under the pointer: the click would never
// arrive and the discard would have committed. So the blur only marks the
// editor, and the ✕, a tap on another cell, the return key or the field
// taking focus again each settle it first. If none does, the timer commits.
// That one commit reaches OnChange from a timer goroutine and not from an
// event handler; State.Set is safe from either.
//
// # Focus
//
// Entering EDIT focuses the field (core.Focus). Ending it focuses a cell: the
// one below after the return key, the same one after ✕. Only the web acts on
// the second, where it is what hands the arrow keys back to the grid; both
// natives ignore a focus command on a box, and neither has arrow keys to give
// back.
//
// # The structure
//
//	Column  (Style)
//	├─ Scroll core.Horizontal()     only when MinWidth is set
//	│  └─ Column RoleGrid, Label
//	│     ├─ Row RoleRow            header: RoleColumnHeader cells
//	│     └─ List RoleRowGroup      windowed body, keyed by Key(row)
//	│        └─ Row RoleRow
//	│           ├─ Row  row header "7"   (RowHeaders)
//	│           └─ Row  cell × n
//	├─ Text RoleAlert               the refused commit's message
//	└─ ActionSheet                  a row's menu (OnInsertRow, OnDeleteRow)
//
// A grid owns rows and rows own cells (core.RoleGrid), so the message and the
// sheet are outside the grid container, and the List between the grid and its
// rows is a rowgroup, which is the one container ARIA lets stand there.
//
// A cell's role says what kind of thing it is:
//
//	RoleGridCell   the cell is itself the control: a text or number cell
//	               (press to edit), a bool cell (press to toggle), a row
//	               header with a menu. One of the grid's arrow-key members.
//	RoleCell       the cell holds a native control, or nothing to press: the
//	               cell being edited, a choice cell, a read-only cell, a row
//	               header with no menu. Not a member.
//
// The second is not squeamishness. The web runtime gives every gridcell a
// keydown listener that owns the arrows, Enter and Space, and a key typed in a
// field inside a gridcell bubbles to it: the caret's arrows would move the
// grid's focus and Space would never reach the text. A row may own a plain
// cell, so the structure stays valid.
//
// A cell's spoken name is "<column>, row <n>, <value>", with "read only"
// appended where it applies. Neither native has a grid vocabulary and both
// announce a gridcell as a button, so the name has to carry the position.
//
// # Rows
//
// OnInsertRow and OnDeleteRow put a menu behind each row header ("Insert
// below", "Delete"). The caller performs the change, since the rows are the
// caller's. This is where Key earns its place: rows keyed by index re-pair
// with their neighbours' nodes after a delete, and an open editor stays at an
// index that is now another row.
//
// # What it is not
//
//   - No formulas. A formula engine is a parser, a dependency graph and cycle
//     detection: an application. Format, plus a caller that recomputes derived
//     values in OnChange, covers totals.
//   - No cell ranges, fill handle or column resize by drag: all need
//     pointer-drag positions, which no target reports.
//   - No frozen first column. RowHeaders scroll away with the rest. On a phone
//     the honest advice is few columns.
//   - No multi-cell paste and no undo. The caller holds the data and receives
//     every commit, so an undo stack is one slice in the caller.
//
// # Cost
//
// Each visible cell is a node. core.List windows the rows on both natives, so
// the native cost is visible rows × columns; columns are not windowed. Go's
// cost is not windowed at all: every pass builds every row's nodes and diffs
// them, and every keystroke in the editor is a pass, because the draft is
// state. Measured at about 4µs a cell on a laptop (BenchmarkEditableGrid30x1000:
// 30,000 cells, 119ms a pass, which is far too slow to type into). So the
// supported size is about 5,000 cells, 10 columns by 500 rows, where a pass
// is some 20ms on a laptop and a phone is a few times that. Past it, page the
// rows: hand the grid a window of them and keep OnChange's row index in step.
//
// A List with no height is not lazy, so give the grid one (Style: core.Height
// or core.FlexGrow).
//
// A changed value patches that cell alone. Entering or leaving EDIT does more:
// callback IDs are issued in render order, the editor registers more of them
// than the box it replaces, and every later cell's onClick is re-bound.
//
// # It holds hooks
//
// Two FocusRefs, the editor, the landing cell, the open menu and the blur
// timer. So it has Accordion's rule: render it in a stable position every
// pass rather than inside a core.If.
//
// # Theme roles read
//
//	Lines      Colors.BorderColor: each row is filled with it and shows 1px
//	           between and under its cells, since no target has per-side borders
//	Header     Colors.Surface, Typography.Caption in TextSecondary
//	Cells      Colors.Background, Typography.Body
//	Editing    Colors.Primary border; Colors.Error after a refused commit
//	Read only  Colors.Surface fill, TextSecondary ink
type EditableGrid struct {
	Columns []GridColumn

	// Rows is the caller's data, as text: Rows[r][c]. The grid is controlled
	// and never writes to it.
	Rows [][]string

	// Key is row r's identity across insert and delete. Nil keys rows by
	// index, which is right for a sheet whose rows never move.
	Key func(row int) string

	// OnChange receives one committed cell. It is not called for a commit
	// that leaves the value as it was.
	OnChange func(row, col int, value string)

	// ReadOnly makes single cells read only, over and above a column's flag.
	ReadOnly func(row, col int) bool

	// RowHeaders draws 1, 2, 3 … down the leading side.
	RowHeaders bool

	// OnInsertRow and OnDeleteRow, when either is set, put a menu behind each
	// row header (and turn RowHeaders on, since the header is the trigger).
	// after is the row to insert below.
	OnInsertRow func(after int)
	OnDeleteRow func(row int)

	// Label is the grid's spoken name. Empty means "Grid".
	Label string

	// MinWidth, in px, is the least the grid is drawn at. When set, the grid
	// sits in a horizontal scroll box and a narrow screen scrolls it sideways
	// instead of squeezing its columns. Zero fits the grid to its parent.
	MinWidth float64

	// Compact tightens the cells' padding for a dense sheet.
	Compact bool

	// Style is applied to the outer Column, HeaderStyle to the header row and
	// CellStyle to every body cell, each after its defaults.
	Style       []core.StyleProp
	HeaderStyle []core.StyleProp
	CellStyle   []core.StyleProp
}

// Render builds the grid as drawn in the type doc.
func (g EditableGrid) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	// Every hook, unconditionally and in one order, before anything reads
	// them.
	editorRef := core.UseFocusRef(ctx)
	landingRef := core.UseFocusRef(ctx)
	editState := core.NewState(ctx, gridEdit{})
	landState := core.NewState(ctx, gridSpot{})
	menuState := core.NewState(ctx, gridMenu{})

	s := gridSession{g: g, t: t, edit: editState, land: landState, menu: menuState,
		editorRef: editorRef, landingRef: landingRef}

	edit := editState.Get()
	// Armed per editor (the deps), so a blur on one cell cannot fire into the
	// next cell's edit. fn reads the state afresh when it fires.
	hooks.UseTimeoutWhile(ctx, edit.on && edit.blurred, func() { s.commit(false) },
		gridBlurGrace, edit.key, edit.col)

	if core.IsDebugMode() {
		g.checkConcerns()
	}

	// The editor's row this pass, or -1. A row that was deleted under an open
	// editor resolves to -1 and the editor is simply not drawn; the state is
	// cleared by whatever the reader does next.
	editRow := -1
	if edit.on {
		editRow = g.rowIndex(edit.key, edit.row)
	}
	land := landState.Get()

	grid := make([]core.PropsAndChildren, 0, 8)
	grid = append(grid,
		core.Padding(0),
		core.Gap(0),
		core.BackgroundColor(t.Colors.Background),
		core.BorderColor(t.Colors.BorderColor()),
		core.BorderWidth(1),
		core.FlexGrow(1),
		core.AccessibilityRole(core.RoleGrid),
		core.AccessibilityLabel(g.label()),
	)
	if g.MinWidth > 0 {
		grid = append(grid, core.MinWidth(px(g.MinWidth)))
	}
	grid = append(grid, g.headerRow(t))

	body := make([]core.PropsAndChildren, 0, len(g.Rows)+5)
	body = append(body, core.Padding(0), core.Gap(0), core.FlexGrow(1))
	if len(g.Rows) > 0 {
		// Withheld from an empty body, as DataTable does: a rowgroup that
		// owns no row claims a structure that is not there.
		body = append(body, core.AccessibilityRole(core.RoleRowGroup))
	}
	for r := range g.Rows {
		editCol := -1
		if r == editRow {
			editCol = edit.col
		}
		landCol := -1
		if land.key != "" && land.key == g.rowKey(r) {
			landCol = land.col
		}
		body = append(body, core.Keyed(g.rowKey(r), s.bodyRow(ctx, r, editCol, landCol, edit)))
	}
	grid = append(grid, core.List(body...))

	outer := make([]core.PropsAndChildren, 0, len(g.Style)+6)
	outer = append(outer, core.Padding(0), core.Gap(float64(t.Spacing.XS)))
	outer = append(outer, asProps(g.Style)...)
	if g.MinWidth > 0 {
		// A Scroll, not a Box: Horizontal() on a Box is only a row with
		// overflow:auto, which a browser scrolls and neither native does.
		// Neither reads Overflow beyond "hidden", so on the simulator the
		// Box handed its 460px content width up to the lesson page, which
		// grew wider than the screen and cut every paragraph off at both
		// edges. The grid Column is a grower, which is each native's strip
		// with a grower (GrMobStripContentLayout, GrMobGrowStrip): proposed
		// max(ideal, viewport), so a wide screen still fills.
		outer = append(outer, core.Scroll(core.Horizontal(), core.Padding(0), core.FlexGrow(1),
			core.Column(grid...)))
	} else {
		outer = append(outer, core.Column(grid...))
	}

	// The refused commit's message. Always in the tree, and hidden when there
	// is nothing to say: an alert that is inserted is announced by a browser,
	// and one whose text changes is announced by TalkBack.
	msg := ""
	if editRow >= 0 {
		msg = edit.err
	}
	alert := []core.StyleProp{
		core.UseStyle(t.Typography.Caption),
		core.TextColor(t.Colors.Error),
		core.AccessibilityRole(core.RoleAlert),
	}
	if msg == "" {
		alert = append(alert, core.Display(core.DisplayNone))
	}
	outer = append(outer, core.Text(msg, alert...))

	outer = append(outer, s.rowMenu())
	return core.Column(outer...).Render(ctx)
}

// gridSession is one pass's view of the grid's state, so the builders below
// take one argument and not seven.
type gridSession struct {
	g          EditableGrid
	t          *core.Theme
	edit       core.State[gridEdit]
	land       core.State[gridSpot]
	menu       core.State[gridMenu]
	editorRef  *core.FocusRef
	landingRef *core.FocusRef
}

// label is Label with its default.
func (g EditableGrid) label() string {
	if g.Label == "" {
		return "Grid"
	}
	return g.Label
}

// rowKey is row r's identity: Key's answer, or the index.
func (g EditableGrid) rowKey(r int) string {
	if g.Key != nil {
		return g.Key(r)
	}
	return "r" + itoa(r)
}

// rowIndex finds the row whose key is key, trying hint first. -1 when the row
// is gone.
func (g EditableGrid) rowIndex(key string, hint int) int {
	if hint >= 0 && hint < len(g.Rows) && g.rowKey(hint) == key {
		return hint
	}
	for r := range g.Rows {
		if g.rowKey(r) == key {
			return r
		}
	}
	return -1
}

// value is Rows[r][c], or "" past the end of a short row.
func (g EditableGrid) value(r, c int) string {
	if r < 0 || r >= len(g.Rows) || c < 0 || c >= len(g.Rows[r]) {
		return ""
	}
	return g.Rows[r][c]
}

// readOnly is whether cell (r, c) is a value and not a control.
func (g EditableGrid) readOnly(r, c int) bool {
	if g.Columns[c].ReadOnly {
		return true
	}
	return g.ReadOnly != nil && g.ReadOnly(r, c)
}

// hasRowMenu is whether a row header opens a menu.
func (g EditableGrid) hasRowMenu() bool { return g.OnInsertRow != nil || g.OnDeleteRow != nil }

// showRowHeaders is RowHeaders, forced on by a row menu, whose trigger the
// header is.
func (g EditableGrid) showRowHeaders() bool { return g.RowHeaders || g.hasRowMenu() }

// checkConcerns reports the four misuses. Gated on IsDebugMode by the caller.
func (g EditableGrid) checkConcerns() {
	editable := false
	for c, col := range g.Columns {
		if col.Kind == GridChoice && len(col.Options) == 0 {
			core.ReportConcern(ConcernEditableGridChoiceNoOptions, fmt.Sprintf(
				"EditableGrid column %d (%q) is a GridChoice with no Options", c, col.Title))
		}
		if !col.ReadOnly {
			editable = true
		}
	}
	// A per-cell ReadOnly can close every cell a column flag left open, and
	// asking it of every cell would cost the sheet per pass. A grid that sets
	// it is taken to have meant its cells to differ.
	if g.OnChange == nil && editable && g.ReadOnly == nil && len(g.Columns) > 0 {
		core.ReportConcern(ConcernEditableGridInert,
			"EditableGrid has editable cells and no OnChange, so every commit is discarded; "+
				"set OnChange, or mark the columns ReadOnly")
	}
	for r, row := range g.Rows {
		if len(row) != len(g.Columns) {
			core.ReportConcern(ConcernEditableGridRagged, fmt.Sprintf(
				"EditableGrid row %d has %d values for %d columns", r, len(row), len(g.Columns)))
			break // one report names the shape; a thousand would bury it
		}
	}
	if g.hasRowMenu() && g.Key == nil {
		core.ReportConcern(ConcernEditableGridNoKey,
			"EditableGrid has OnInsertRow or OnDeleteRow and no Key, so rows are keyed by index: "+
				"a delete re-pairs every row below it, and an open editor moves to another row")
	}
}

// ---- layout ---------------------------------------------------------------

// rowHeaderWidth is the row-number column's px width: room for four digits
// in Caption type.
const rowHeaderWidth = 44

// pad is a cell's padding on each axis.
func (g EditableGrid) pad(t *core.Theme) (h, v int) {
	if g.Compact {
		return t.Spacing.XS, t.Spacing.XS
	}
	return t.Spacing.SM, t.Spacing.SM
}

// sizing is the flex share of column c's cell, the same on the header and on
// every body row, which is what keeps a column one width down the sheet.
//
// A weighted cell starts from a zero basis. The natives divide the axis by
// weight and ignore the basis; CSS divides only the leftover, and without the
// zero a long value would widen its own cell on its own row.
func (g EditableGrid) sizing(c int) []core.PropsAndChildren {
	col := g.Columns[c]
	weight := col.Weight
	if weight <= 0 && col.Width <= 0 {
		weight = 1
	}
	if weight > 0 {
		out := []core.PropsAndChildren{core.FlexGrow(weight), core.FlexBasis("0"), core.MinWidth("0")}
		if col.Width > 0 {
			out[2] = core.MinWidth(px(col.Width))
		}
		return out
	}
	return []core.PropsAndChildren{core.Width(px(col.Width)), core.FlexShrink(0)}
}

// fixedSizing is the row-header column's share.
func fixedSizing(w float64) []core.PropsAndChildren {
	return []core.PropsAndChildren{core.Width(px(w)), core.FlexShrink(0)}
}

// align is column c's justification with the Kind's default applied.
func (g EditableGrid) align(c int) core.JustifyContent {
	col := g.Columns[c]
	if col.Align != "" {
		return col.Align
	}
	switch col.Kind {
	case GridNumber:
		return core.JustifyEnd
	case GridBool:
		return core.JustifyCenter
	}
	return core.JustifyStart
}

// ruled is what makes a row draw the grid's lines: no target has per-side
// borders, so a row is filled with the line colour and its cells, which carry
// their own fills, leave 1px of it showing between them and under them.
//
// The lines are the rows' and not the grid's. The first build filled the whole
// grid with the line colour and spaced the rows 1px apart, and a body taller
// than its rows then showed the leftover as a grey band.
func (g EditableGrid) ruled(t *core.Theme) []core.PropsAndChildren {
	return []core.PropsAndChildren{
		core.Padding(0), core.PaddingBottom(1), core.Gap(1),
		core.AlignItemsProp(core.AlignItemsStretch),
		core.BackgroundColor(t.Colors.BorderColor()),
	}
}

// headerRow is the column titles, with a corner over the row headers.
func (g EditableGrid) headerRow(t *core.Theme) core.View {
	h, v := g.pad(t)
	items := make([]core.PropsAndChildren, 0, len(g.Columns)+len(g.HeaderStyle)+5)
	items = append(items, g.ruled(t)...)
	items = append(items, core.AccessibilityRole(core.RoleRow))
	items = append(items, asProps(g.HeaderStyle)...)

	head := func(title, spoken string, sizing []core.PropsAndChildren, align core.JustifyContent) core.View {
		cell := []core.PropsAndChildren{
			core.Padding(0), core.PaddingHorizontal(h), core.PaddingVertical(v),
			core.AlignItemsProp(core.AlignItemsCenter),
			core.Justify(align),
			core.BackgroundColor(t.Colors.Surface),
			core.AccessibilityRole(core.RoleColumnHeader),
		}
		if spoken != "" {
			cell = append(cell, core.AccessibilityLabel(spoken))
		}
		cell = append(cell, sizing...)
		cell = append(cell, core.Text(title,
			core.UseStyle(t.Typography.Caption),
			core.FontWeight(core.Bold),
			core.TextColor(t.Colors.TextSecondary),
			core.MaxLines(1)))
		return core.Row(cell...)
	}
	if g.showRowHeaders() {
		items = append(items, head("#", "Row", fixedSizing(rowHeaderWidth), core.JustifyCenter))
	}
	for c, col := range g.Columns {
		items = append(items, head(col.Title, "", g.sizing(c), g.align(c)))
	}
	return core.Row(items...)
}

// bodyRow is one row of cells. editCol and landCol are the columns of this
// row that hold the editor and the landing ref, or -1.
func (s gridSession) bodyRow(ctx *core.Context, r, editCol, landCol int, edit gridEdit) core.View {
	g := s.g
	items := make([]core.PropsAndChildren, 0, len(g.Columns)+5)
	items = append(items, g.ruled(s.t)...)
	items = append(items, core.AccessibilityRole(core.RoleRow))
	if g.showRowHeaders() {
		items = append(items, s.rowHeader(r))
	}
	for c := range g.Columns {
		switch {
		case c == editCol:
			// Keyed apart from the box it replaces, so the swap is a
			// replacement and not a morph: the web runtime's gridcell
			// listener and tab stop go with the old element.
			items = append(items, core.Keyed("e"+itoa(c), s.editorCell(r, c, edit)))
		default:
			items = append(items, core.Keyed("c"+itoa(c), s.cell(r, c, c == landCol)))
		}
	}
	return core.Row(items...)
}

// cellBase is what every body cell starts from: the padding, the share of the
// row and the fill.
func (s gridSession) cellBase(c int, fill string) []core.PropsAndChildren {
	h, v := s.g.pad(s.t)
	items := make([]core.PropsAndChildren, 0, 12)
	items = append(items,
		core.Padding(0), core.PaddingHorizontal(h), core.PaddingVertical(v),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Justify(s.g.align(c)),
		core.BackgroundColor(fill),
	)
	items = append(items, s.g.sizing(c)...)
	return append(items, asProps(s.g.CellStyle)...)
}

// spoken is a cell's name: "<column>, row <n>, <value>".
func (s gridSession) spoken(r, c int, value string) string {
	if value == "" {
		value = "empty"
	}
	return fmt.Sprintf("%s, row %d, %s", s.g.Columns[c].Title, r+1, value)
}

// cell is one body cell outside EDIT. landing marks the cell that carries the
// landing FocusRef this pass.
func (s gridSession) cell(r, c int, landing bool) core.View {
	g, t := s.g, s.t
	col := g.Columns[c]
	raw := g.value(r, c)
	ro := g.readOnly(r, c)

	fill, ink := t.Colors.Background, t.Colors.TextPrimary
	if ro {
		fill, ink = t.Colors.Surface, t.Colors.TextSecondary
	}
	items := s.cellBase(c, fill)
	if landing {
		items = append(items, core.FocusTarget(s.landingRef))
	}

	shown := raw
	if col.Format != nil {
		shown = col.Format(raw)
	}
	text := func(content string) core.View {
		// A space in an empty cell, so every row is a line of text tall on
		// every host (PINInput's empty box, for the same reason).
		if content == "" {
			content = " "
		}
		return core.Text(content, core.UseStyle(t.Typography.Body), core.TextColor(ink), core.MaxLines(1))
	}

	switch {
	case col.Kind == GridBool:
		on := gridBool(raw)
		glyph, state := "☐", "not checked"
		if on {
			glyph, state = "☑", "checked"
		}
		name := s.spoken(r, c, state)
		if ro {
			items = append(items, core.AccessibilityRole(core.RoleCell),
				core.AccessibilityLabel(name+", read only"))
		} else {
			next := strconv.FormatBool(!on)
			items = append(items,
				core.AccessibilityRole(core.RoleGridCell),
				core.AccessibilityLabel(name),
				core.OnClick(func() { s.toggle(r, c, next) }))
		}
		// Hidden: the glyph's own name ("ballot box with check") would be
		// read after a label that has already said it.
		items = append(items, core.Text(glyph, core.FontSize(t.Typography.Body.FontSize+4),
			core.TextColor(ink), core.AccessibilityHidden()))

	case ro:
		items = append(items, core.AccessibilityRole(core.RoleCell),
			core.AccessibilityLabel(s.spoken(r, c, shown)+", read only"), text(shown))

	case col.Kind == GridChoice:
		// A plain cell round a native picker: see the role table on the type.
		opts := make([]core.SelectOption, 0, len(col.Options)+1)
		if !gridHas(col.Options, raw) {
			// The stored value is not one of the options (an empty new row,
			// usually). Offered as itself, or the picker would draw the first
			// option over a value that is not it.
			opts = append(opts, core.SelectOption{Value: raw, Label: gridChoiceBlank(raw)})
		}
		for _, o := range col.Options {
			opts = append(opts, core.SelectOption{Value: o})
		}
		// The cell is the frame, as it is for the editor: a picker drawn with
		// its own border and padding made every row half again as tall and
		// left room for one letter of the choice.
		items = append(items, core.AccessibilityRole(core.RoleCell),
			core.Select(raw, opts, func(v string) { s.choose(r, c, v) },
				core.Padding(0), core.BorderWidth(0), core.BorderRadius(0),
				core.BackgroundColor(ColorTransparent),
				core.FontSize(t.Typography.Body.FontSize),
				core.FlexGrow(1), core.MinWidth("0"),
				core.AccessibilityLabel(fmt.Sprintf("%s, row %d", col.Title, r+1))))

	default:
		items = append(items,
			core.AccessibilityRole(core.RoleGridCell),
			core.AccessibilityLabel(s.spoken(r, c, shown)),
			core.AccessibilityHint("Edits the cell"),
			core.OnClick(func() { s.begin(r, c) }),
			text(shown))
	}
	return core.Row(items...)
}

// editorCell is the cell in EDIT: the field over the draft, and the ✕.
func (s gridSession) editorCell(r, c int, edit gridEdit) core.View {
	g, t := s.g, s.t
	col := g.Columns[c]

	ring := t.Colors.Primary
	if edit.err != "" {
		ring = t.Colors.Error
	}
	items := s.cellBase(c, t.Colors.Background)
	items = append(items,
		// The ring is drawn inside the cell's box, so the padding gives up
		// what the border takes and the row does not grow by 4px on entry.
		core.BorderColor(ring), core.BorderWidth(2),
		core.Gap(float64(t.Spacing.XS)),
		core.AccessibilityRole(core.RoleCell),
	)
	h, v := g.pad(t)
	items = append(items, core.PaddingHorizontal(maxInt(h-2, 0)), core.PaddingVertical(maxInt(v-2, 0)))

	keyboard := col.Keyboard
	if keyboard == core.KeyboardText && col.Kind == GridNumber {
		keyboard = core.KeyboardDecimal
	}
	name := fmt.Sprintf("%s, row %d", col.Title, r+1)
	field := []core.PropsAndChildren{
		core.FocusTarget(s.editorRef),
		core.Keyboard(keyboard),
		core.OnBlur(func() { s.blurred(true) }),
		core.OnFocus(func() { s.blurred(false) }),
		// The cell is the frame; the field inside it has none, so the text
		// sits where the box's text sat.
		core.Padding(0), core.BorderWidth(0), core.BorderRadius(0),
		core.BackgroundColor(ColorTransparent),
		core.FlexGrow(1), core.MinWidth("0"),
		core.FontSize(t.Typography.Body.FontSize),
		core.AccessibilityLabel(name),
	}
	if edit.err != "" {
		field = append(field, core.AccessibilityHint(edit.err))
	}
	if g.align(c) == core.JustifyEnd {
		field = append(field, core.Align(core.AlignEnd))
	}
	items = append(items, core.InputWithSubmit(edit.draft, "",
		func(typed string) { s.typed(typed) },
		func() { s.commit(true) },
		field...))

	// A Box and not a core.Button: Compose gives a Button a 48dp minimum,
	// which would make the editing row half again as tall as its neighbours.
	items = append(items, core.Box(
		core.Padding(0), core.PaddingHorizontal(t.Spacing.XS),
		core.OnClick(func() { s.discard() }),
		core.AccessibilityRole(core.RoleButton),
		core.AccessibilityLabel("Discard edit"),
		core.Text("✕", core.TextColor(t.Colors.TextSecondary), core.AccessibilityHidden()),
	))
	return core.Row(items...)
}

// rowHeader is the row-number cell, and the row menu's trigger when there is
// a menu.
func (s gridSession) rowHeader(r int) core.View {
	g, t := s.g, s.t
	_, v := g.pad(t)
	items := []core.PropsAndChildren{
		core.Padding(0), core.PaddingVertical(v),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Justify(core.JustifyCenter),
		core.BackgroundColor(t.Colors.Surface),
	}
	items = append(items, fixedSizing(rowHeaderWidth)...)
	if g.hasRowMenu() {
		items = append(items,
			core.AccessibilityRole(core.RoleGridCell),
			core.AccessibilityLabel(fmt.Sprintf("Row %d", r+1)),
			core.AccessibilityHint("Opens the row's menu"),
			core.OnClick(func() { s.openMenu(r) }))
	} else {
		items = append(items, core.AccessibilityRole(core.RoleCell),
			core.AccessibilityLabel(fmt.Sprintf("Row %d", r+1)))
	}
	items = append(items, core.Text(itoa(r+1),
		core.UseStyle(t.Typography.Caption), core.TextColor(t.Colors.TextSecondary),
		core.AccessibilityHidden()))
	return core.Keyed("h", core.Row(items...))
}

// rowMenu is the one sheet every row header opens. Always rendered, because
// ActionSheet is a Modal and a Modal is shown by a prop, not by presence.
func (s gridSession) rowMenu() core.View {
	g := s.g
	m := s.menu.Get()
	row := -1
	if m.open {
		row = g.rowIndex(m.key, m.row)
	}
	shut := func() { s.menu.Set(gridMenu{}) }

	var actions []SheetAction
	if g.OnInsertRow != nil {
		actions = append(actions, SheetAction{Label: "Insert below", OnTap: func() {
			if row >= 0 {
				g.OnInsertRow(row)
			}
		}})
	}
	if g.OnDeleteRow != nil {
		actions = append(actions, SheetAction{Label: "Delete", Variant: VariantError, OnTap: func() {
			if row >= 0 {
				g.OnDeleteRow(row)
			}
		}})
	}
	title := "Row"
	if row >= 0 {
		title = fmt.Sprintf("Row %d", row+1)
	}
	return ActionSheet{
		Visible:   row >= 0 && len(actions) > 0,
		Title:     title,
		Actions:   actions,
		Cancel:    "Cancel",
		OnDismiss: shut,
	}
}

// ---- the state machine ----------------------------------------------------
//
// Every transition reads the state afresh (State.Get) rather than a value
// captured at render, because two of them run in one dispatch (begin commits
// the open editor first) and one runs from a timer.

// begin opens the editor on (r, c), committing an open one first. If that
// commit is refused the reader stays where they were: the message is about
// that cell, and moving on would orphan it.
func (s gridSession) begin(r, c int) {
	if s.edit.Get().on && !s.commit(false) {
		return
	}
	s.edit.Set(gridEdit{on: true, key: s.g.rowKey(r), row: r, col: c, draft: s.g.value(r, c)})
	core.Focus(s.editorRef)
}

// typed takes a keystroke into the draft, and clears a refused commit's
// message: the reader is answering it.
func (s gridSession) typed(text string) {
	e := s.edit.Get()
	if !e.on || (e.draft == text && e.err == "") {
		return
	}
	e.draft, e.err = text, ""
	s.edit.Set(e)
}

// blurred marks or unmarks the editor as having lost focus. See "Why a blur
// waits".
func (s gridSession) blurred(on bool) {
	e := s.edit.Get()
	if !e.on || e.blurred == on {
		return
	}
	e.blurred = on
	s.edit.Set(e)
}

// commit ends the edit with its draft, reporting whether it ended. move is the
// return key's "and go down a row".
func (s gridSession) commit(move bool) bool {
	g := s.g
	e := s.edit.Get()
	if !e.on {
		return true
	}
	r := g.rowIndex(e.key, e.row)
	if r < 0 || e.col >= len(g.Columns) {
		// The row went away under the editor. Nothing to commit to.
		s.edit.Set(gridEdit{})
		return true
	}
	col := g.Columns[e.col]
	draft := e.draft
	if col.Kind == GridNumber {
		draft = strings.TrimSpace(draft)
	}
	if msg := gridValidate(col, draft); msg != "" {
		e.err, e.blurred = msg, false
		s.edit.Set(e)
		if move {
			// The return key took the field's focus with it on Android.
			core.Focus(s.editorRef)
		}
		return false
	}
	s.edit.Set(gridEdit{})
	if g.OnChange != nil && draft != g.value(r, e.col) {
		g.OnChange(r, e.col, draft)
	}
	if move {
		to := r
		if r+1 < len(g.Rows) {
			to = r + 1
		}
		s.land.Set(gridSpot{key: g.rowKey(to), col: e.col})
		core.Focus(s.landingRef)
	}
	return true
}

// discard ends the edit without its draft and hands focus back to the cell.
func (s gridSession) discard() {
	e := s.edit.Get()
	if !e.on {
		return
	}
	s.edit.Set(gridEdit{})
	s.land.Set(gridSpot{key: e.key, col: e.col})
	core.Focus(s.landingRef)
}

// toggle flips a bool cell, committing an open editor first.
func (s gridSession) toggle(r, c int, next string) {
	if s.edit.Get().on && !s.commit(false) {
		return
	}
	if s.g.OnChange != nil {
		s.g.OnChange(r, c, next)
	}
}

// choose takes a choice cell's pick.
func (s gridSession) choose(r, c int, v string) {
	if s.edit.Get().on && !s.commit(false) {
		return
	}
	if s.g.OnChange != nil && v != s.g.value(r, c) {
		s.g.OnChange(r, c, v)
	}
}

// openMenu opens row r's menu, committing an open editor first so an insert
// or delete never runs under a draft.
func (s gridSession) openMenu(r int) {
	if s.edit.Get().on && !s.commit(false) {
		return
	}
	s.menu.Set(gridMenu{open: true, key: s.g.rowKey(r), row: r})
}

// ---- small helpers --------------------------------------------------------

// gridValidate is the Kind's own rule, then the column's.
func gridValidate(col GridColumn, draft string) string {
	if col.Kind == GridNumber && draft != "" {
		if _, err := strconv.ParseFloat(draft, 64); err != nil {
			return fmt.Sprintf("%s: %q is not a number", col.Title, draft)
		}
	}
	if col.Validate != nil {
		return col.Validate(draft)
	}
	return ""
}

// gridBool reads a bool cell's value; anything unparseable is false.
func gridBool(v string) bool {
	on, err := strconv.ParseBool(strings.TrimSpace(v))
	return err == nil && on
}

// gridHas is whether v is one of opts.
func gridHas(opts []string, v string) bool {
	for _, o := range opts {
		if o == v {
			return true
		}
	}
	return false
}

// gridChoiceBlank labels the stand-in option for a value outside Options.
func gridChoiceBlank(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

// maxInt is the larger of two paddings.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
