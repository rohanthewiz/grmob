package components

import (
	"fmt"
	"sort"

	"github.com/rohanthewiz/grmob/core"
)

// Column describes one column of a DataTable: its header, how a row's value
// is drawn, how much of the row's width it takes, and whether it sorts.
type Column[T any] struct {
	Title string

	// Text is the simple path: the cell shows this string in body type.
	// Cell is the slot: any view, taking precedence when both are set — the
	// same simple-path-plus-slot idiom as Card.Title vs Card.Header.
	Text func(T) string
	Cell func(T) core.View

	// Weight is the column's FlexGrow share of the row's slack. 0 hugs the
	// cell's content, which is right for a short code or a glyph column;
	// give the column that carries the row's meaning a weight so it takes
	// what the fixed ones leave.
	Weight float64

	// Align positions the cell's content on the row axis; the zero value is
	// the leading edge. Numbers want JustifyEnd.
	Align core.JustifyContent

	// Narrow marks a column the table drops in Compact mode: a phone shows
	// title and date, a tablet adds teacher and place.
	Narrow bool

	// Less orders two rows for a client-side sort on this column; setting it
	// also makes the header tappable. Sortable makes the header tappable
	// without a comparator, for a table whose caller sorts server-side and
	// only needs to hear which column was asked for.
	//
	// Which of the two to set is not a matter of convenience: it depends on
	// whether DataTable.Rows is the whole set or a window of it. See the
	// "Sorting sorts the rows you hand it" section on DataTable.
	Less     func(a, b T) bool
	Sortable bool
}

// Sort names the active sort column and direction. DataTable.Sort is a
// pointer so that "no sort" is nil rather than an ambiguous column 0.
type Sort struct {
	Column int
	Desc   bool
}

// ConcernPartialSort: a DataTable sorted client-side (the active Sort names
// a column with a Less) while its Pagination declares a PageCount — which is
// the caller saying the server chooses which rows arrive. The table can only
// order the window it was handed, so the header claims an ordering over the
// whole table and delivers one over one page of it. The fix is to drop the
// column's Less and keep Sortable, letting OnSort go into the query.
//
// Reported through core.ReportConcern rather than detected in core: this is
// a widget-level contract, and core has no business knowing what a DataTable
// is. Debug mode only, like every other concern.
const ConcernPartialSort = "partial-sort"

// DataTable is a header row over a virtualized, keyed body of cell rows,
// with controlled sorting, optional grouping, optional pagination and a
// compact mode for narrow screens.
//
//	┌ Column ────────────────────────────────────────────┐
//	│ ┌ Row (header) ────────────────────────────────┐   │
//	│ │ Title ▲          Teacher         Date        │   │  <- tap sorts
//	│ └──────────────────────────────────────────────┘   │
//	│ ┌ List (body) ─────────────────────────────────┐   │
//	│ │ ▒ January 2026                          (2)  │   │  <- GroupHeader
//	│ │ Grace and Truth   Pastor Ray     Jan 12      │   │  <- key Key(row)
//	│ │ ─────────────────────────────────────────    │   │  <- Dividers
//	│ │ The Vine          Guest          Jan 5       │   │
//	│ └──────────────────────────────────────────────┘   │
//	│ ‹ Prev            Page 1 of 4            Next ›    │  <- Pagination
//	└────────────────────────────────────────────────────┘
//
// # Order of operations
//
// Rows are sorted, then paged, then grouped: the sort decides the order,
// paging takes a window of it, and grouping reads runs off that window.
// Grouping last is what makes a client-side page's headers agree with its
// rows, and sorting first is what makes group runs follow the sort (a table
// sorted by teacher and grouped by month shows a month once per teacher run
// — the honest rendering; see groupRuns).
//
// # Sorting sorts the rows you hand it
//
// A column with Less is sorted by the table, over Rows — all of Rows, and
// only Rows. That is right whenever Rows is the whole set: a slice the
// caller holds, or a fetch that completed, including the client-paged case
// where Pagination slices rows the table already has.
//
// It is wrong when Rows is a window somebody else chose. An offset pager's
// "the three pages loaded so far", or a server-paged table declaring
// PageCount, is a subset selected under some other ordering. Sorting that
// yields the alphabetically-first of the rows that happen to be loaded, and
// puts it under a header claiming to be the alphabetically-first rows. The
// result carries no sign of which it is, which is exactly what makes the
// mistake worth naming: a partial sort looks like a working one, and the
// rows that disprove it are the ones not fetched.
//
// For that table set Sortable instead of Less. The header still taps and
// OnSort still fires; the sort goes into the next query, and the server
// returns a window of the ordering the reader actually asked for.
//
// One shape of this is detectable and is reported as a debug concern: Sort
// active on a Less column while Pagination declares a PageCount, since a
// declared PageCount is the caller saying the server does the paging. See
// ConcernPartialSort. The append-style case has no such signal — a slice of
// accumulated pages is indistinguishable from a complete one — so that half
// is left to this documentation.
//
// # Who owns what
//
// Everything is controlled. The table holds no state and calls no hook: Sort
// is read from the caller and OnSort reports the tap, Pagination.Page is
// read and OnChange reports the step. Only the *work* is optionally the
// table's: a column with Less is sorted here, a Pagination with PageSize and
// no PageCount is sliced here. Both are pure functions of the inputs.
//
// # Compact
//
// A phone is too narrow for a five-column table. Compact drops every Narrow
// column and leaves the rest; the header and cells stay in step because both
// walk the same filtered column list. The caller decides when the table is
// compact — from a breakpoint, a settings toggle, an orientation event.
type DataTable[T any] struct {
	Columns []Column[T]
	Rows    []T

	// Key returns the reconciler key for a row; nil falls back to positional
	// keys (see GroupedList for the trade-off).
	Key func(T) string

	// GroupBy, Header and HideTrailingCount work as in GroupedList; a group
	// header spans the full row.
	//
	// HideTrailingCount is for the same case there and here: a table fed by
	// an append-style pager, whose last group can still grow. It has no work
	// to do for a table that paginates through Pagination, where every page
	// shows a complete slice of rows the table already holds.
	GroupBy           func(T) Group
	Header            func(Group) core.View
	HideTrailingCount bool

	// Sort is the active sort, nil for none. OnSort receives the requested
	// sort when a sortable header is tapped: the same column toggles
	// direction, a different column starts ascending.
	//
	// Whether the table then does the sorting depends on the column: see
	// "Sorting sorts the rows you hand it" above before giving a column a
	// Less on a table whose Rows come a page at a time.
	Sort   *Sort
	OnSort func(Sort)

	// Pagination, when non-nil, renders the numbered footer and — when it
	// carries a PageSize and no PageCount — slices Rows to the current page.
	Pagination *Pagination

	// OnRowTap makes every body row a target.
	OnRowTap func(T)
	// Selected tints a row with the theme's Surface, ListRow's convention.
	Selected func(T) bool

	// Compact drops Narrow columns.
	Compact bool
	// Dividers inserts a hairline between consecutive body rows.
	Dividers bool

	// Empty is rendered in the body when there are no rows; Loading is
	// rendered in the body instead of the rows whenever it is non-nil, so a
	// refresh keeps its header and footer in place.
	Empty   core.View
	Loading core.View

	// Footer is an arbitrary view after the body, before the Pagination
	// footer: a LoadMore for a table that appends rather than pages.
	Footer core.View

	// Style is applied to the outer Column; HeaderStyle to the header row;
	// RowStyle to every body row, after each one's defaults.
	Style       []core.StyleProp
	HeaderStyle []core.StyleProp
	RowStyle    []core.StyleProp
}

func (d DataTable[T]) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	cols := d.visibleColumns()

	if core.IsDebugMode() {
		d.checkPartialSort()
	}

	rows := d.sorted()
	pager := d.Pagination
	if pager != nil && pager.PageCount == 0 && pager.PageSize > 0 {
		// Client-side paging: slice here and hand the footer a copy that
		// knows the page count, so "Page 2 of 7" is derived, not declared.
		var page Pagination
		page, rows = d.slice(*pager, rows)
		pager = &page
	}

	items := make([]core.PropsAndChildren, 0, len(d.Style)+6)
	items = append(items, core.Padding(0), core.Gap(0))
	for _, sp := range d.Style {
		items = append(items, sp)
	}

	items = append(items, d.headerRow(ctx, cols))

	body := make([]core.PropsAndChildren, 0, 2*len(rows)+4)
	body = append(body, core.Padding(0), core.Gap(0), core.FlexGrow(1))
	switch {
	case d.Loading != nil:
		body = append(body, d.Loading)
	case len(rows) == 0:
		if d.Empty != nil {
			body = append(body, d.Empty)
		}
	default:
		body = appendRows(ctx, body, rows, d.Key,
			func(row T) core.View { return d.cellRow(t, cols, row) },
			d.GroupBy, d.Header, d.HideTrailingCount, d.Dividers, d.wrapRow)
	}
	items = append(items, core.List(body...))

	if d.Footer != nil {
		items = append(items, d.Footer)
	}
	if pager != nil {
		items = append(items, *pager)
	}
	return core.Column(items...).Render(ctx)
}

// visibleColumns is Columns with the Narrow ones removed in Compact mode.
// The original index is kept alongside so Sort.Column and OnSort keep
// speaking in terms of the caller's column list, not the filtered one —
// otherwise toggling Compact would silently re-point the active sort.
type visibleColumn[T any] struct {
	Column[T]
	index int
}

func (d DataTable[T]) visibleColumns() []visibleColumn[T] {
	out := make([]visibleColumn[T], 0, len(d.Columns))
	for i, c := range d.Columns {
		if d.Compact && c.Narrow {
			continue
		}
		out = append(out, visibleColumn[T]{Column: c, index: i})
	}
	return out
}

// checkPartialSort reports the one shape of a partial sort the table can see:
// sorting here, over rows the caller has said the server pages. See
// ConcernPartialSort.
//
// Gated by the caller on IsDebugMode, per ReportConcern's contract — the
// detail string costs a Sprintf and a release build should not pay for one.
// The concern collector deduplicates on kind+detail, so a table that renders
// every frame occupies one entry with a rising count.
func (d DataTable[T]) checkPartialSort() {
	if d.Sort == nil || d.Pagination == nil || d.Pagination.PageCount <= 0 {
		return
	}
	col := d.Sort.Column
	if col < 0 || col >= len(d.Columns) || d.Columns[col].Less == nil {
		return
	}
	core.ReportConcern(ConcernPartialSort, fmt.Sprintf(
		"column %q has a Less, so it is sorted over the %d rows given, but "+
			"Pagination.PageCount is %d: the server chose those rows. Use "+
			"Sortable without Less and sort in the query.",
		d.Columns[col].Title, len(d.Rows), d.Pagination.PageCount))
}

// sorted returns Rows ordered by the active sort when that column has a
// comparator, and Rows itself otherwise. The sort is stable and works on a
// copy: Rows is the caller's slice and a widget must not reorder it under
// them.
func (d DataTable[T]) sorted() []T {
	if d.Sort == nil || d.Sort.Column < 0 || d.Sort.Column >= len(d.Columns) {
		return d.Rows
	}
	less := d.Columns[d.Sort.Column].Less
	if less == nil {
		return d.Rows
	}
	out := make([]T, len(d.Rows))
	copy(out, d.Rows)
	desc := d.Sort.Desc
	sort.SliceStable(out, func(i, j int) bool {
		if desc {
			return less(out[j], out[i])
		}
		return less(out[i], out[j])
	})
	return out
}

// slice returns the pager with PageCount filled in and Page clamped, and the
// window of rows it names. A stale Page past the end (rows were removed)
// lands on the last page rather than on an empty one.
func (d DataTable[T]) slice(p Pagination, rows []T) (Pagination, []T) {
	n := len(rows)
	p.PageCount = (n + p.PageSize - 1) / p.PageSize
	if p.PageCount == 0 {
		p.PageCount = 1
	}
	if p.Page < 0 {
		p.Page = 0
	}
	if p.Page > p.PageCount-1 {
		p.Page = p.PageCount - 1
	}
	start := p.Page * p.PageSize
	end := start + p.PageSize
	if end > n {
		end = n
	}
	return p, rows[start:end]
}

// headerRow is the column titles, each a tappable cell when its column
// sorts, with a direction glyph on the active one.
func (d DataTable[T]) headerRow(ctx *core.Context, cols []visibleColumn[T]) core.View {
	t := ctx.Theme()
	items := make([]core.PropsAndChildren, 0, len(cols)+len(d.HeaderStyle)+6)
	items = append(items,
		core.Padding(0),
		core.Gap(0),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.BackgroundColor(t.Colors.Surface),
		core.BorderColor(t.Colors.BorderColor()),
		core.BorderWidth(1),
	)
	for _, sp := range d.HeaderStyle {
		items = append(items, sp)
	}

	for _, c := range cols {
		title := c.Title
		sortable := d.OnSort != nil && (c.Less != nil || c.Sortable)
		active := d.Sort != nil && d.Sort.Column == c.index
		if active {
			if d.Sort.Desc {
				title += " ▼"
			} else {
				title += " ▲"
			}
		}

		cell := []core.PropsAndChildren{
			core.Text(title,
				core.UseStyle(t.Typography.Caption),
				core.FontWeight(core.Bold),
				core.TextColor(t.Colors.TextSecondary)),
		}
		if sortable {
			// The next sort this tap asks for: toggle on the active column,
			// start ascending on any other. Computed here, in the render,
			// so the callback captures a value and not the table.
			next := Sort{Column: c.index}
			if active {
				next.Desc = !d.Sort.Desc
			}
			cell = append(cell,
				core.OnClick(func() { d.OnSort(next) }),
				core.AccessibilityHint("Sorts by "+c.Title),
			)
		}
		items = append(items, d.cell(t, c.Column, cell...))
	}
	return core.Row(items...)
}

// cellRow is one body row: a cell per visible column.
func (d DataTable[T]) cellRow(t *core.Theme, cols []visibleColumn[T], row T) core.View {
	items := make([]core.PropsAndChildren, 0, len(cols)+len(d.RowStyle)+4)
	items = append(items,
		core.Padding(0),
		core.Gap(0),
		core.AlignItemsProp(core.AlignItemsCenter),
	)
	for _, sp := range d.RowStyle {
		items = append(items, sp)
	}
	for _, c := range cols {
		var content core.View
		switch {
		case c.Cell != nil:
			content = c.Cell(row)
		case c.Text != nil:
			content = core.Text(c.Text(row), core.UseStyle(t.Typography.Body))
		default:
			content = core.Fragment()
		}
		items = append(items, d.cell(t, c.Column, content))
	}
	return core.Row(items...)
}

// cell is the sized, padded box every header and body cell sits in. A Row
// rather than a Box so Align can use flex justification, which every target
// implements, rather than text alignment, which only text nodes have.
func (d DataTable[T]) cell(t *core.Theme, c Column[T], content ...core.PropsAndChildren) core.View {
	items := make([]core.PropsAndChildren, 0, len(content)+6)
	items = append(items,
		core.Padding(0),
		core.PaddingHorizontal(t.Spacing.SM),
		core.PaddingVertical(t.Spacing.SM),
		core.AlignItemsProp(core.AlignItemsCenter),
	)
	if c.Weight > 0 {
		items = append(items, core.FlexGrow(c.Weight))
	}
	if c.Align != "" {
		items = append(items, core.Justify(c.Align))
	}
	items = append(items, content...)
	return core.Row(items...)
}

// wrapRow adds the per-row behavior that is the table's rather than the
// cells': the tap target and the selection tint. Applied by appendRows around
// cellRow's output so the keyed node is the one carrying the handler.
func (d DataTable[T]) wrapRow(row T, v core.View) core.View {
	tap := d.OnRowTap != nil
	selected := d.Selected != nil && d.Selected(row)
	if !tap && !selected {
		return v
	}
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		n := v.Render(ctx)
		if selected {
			n.Style.Background = ctx.Theme().Colors.Surface
		}
		if tap {
			// Guarded like ListRow: a presentational table registers no
			// callback per row.
			core.OnClick(func() { d.OnRowTap(row) }).Apply(ctx, n)
		}
		return n
	})
}
