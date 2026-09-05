package components

import "github.com/rohanthewiz/grmob/core"

// GroupedList is a virtualized, keyed list of typed items with optional
// run-length group headers and a footer slot for a pager: the shape of an
// archive feed — sermons by month, transactions by day, messages by sender.
//
// # What it settles
//
// Every paged screen in an app builds the same core.List by hand: a keyed
// row per item, an empty note when there are none, and a "Load more" tail.
// The widget owns that assembly so the screen owns only the three things
// that differ — how to key an item, how to draw one, and where its pages
// come from.
//
//	┌ List ──────────────────────────────────┐
//	│ ▒ January 2026                     (3) │  <- GroupHeader, key "group:2026-01"
//	│   Row(item)                            │  <- key Key(item)
//	│   Row(item)                            │
//	│   Row(item)                            │
//	│ ▒ December 2025                    (1) │
//	│   Row(item)                            │
//	│           [ Load more ]                │  <- Footer
//	└────────────────────────────────────────┘
//
// # Grouping is by run, not by bucket
//
// GroupBy is evaluated in Items order and a header is emitted wherever the
// key changes (see groupRuns). Items must therefore arrive already ordered
// by the grouping — which a feed sorted by date, grouped by month, always
// is — and an offset pager that appends pages can only ever grow the last
// group, so nothing above the fold moves on "Load more".
//
// # Identity
//
// core.List keeps row state attached to keys across insertions and reorders,
// so Key must be unique across the whole list and stable across renders. A
// nil Key falls back to positional keys, which is correct for a static list
// and loses row state on reorder for a live one, exactly as core.List
// documents. Group headers take "group:"+Group.Key, so a row key can never
// collide with a header even when a caller keys rows by the same string.
//
// # No hooks
//
// The widget holds no state and calls no hook, so it may be rendered
// conditionally — inside core.IfElse against a pager's loaded flag, say —
// without disturbing the caller's hook cursor.
type GroupedList[T any] struct {
	Items []T

	// Key returns the reconciler key for an item; see the type comment.
	Key func(T) string
	// Row draws one item. Required.
	Row func(T) core.View

	// GroupBy assigns each item to a Group by Key and Label; Count is filled
	// in. Nil renders a flat list.
	GroupBy func(T) Group
	// Header overrides the default GroupHeader for each group.
	Header func(Group) core.View

	// HideTrailingCount drops the count badge from the *last* group's
	// header. Set it from a pager's "there is more" flag.
	//
	// A group's Count counts the rows the list was handed, which under an
	// append-style pager means "the rows loaded so far". Every group but the
	// last is closed — the next group's first row ended it — so its count is
	// final. The last one is still open: the next page can extend it, and a
	// header that says "June 2026 (1)" above a run about to become four is
	// not a stale number, it is a wrong one, and it changes under the reader
	// on a tap they did not think was a question about June.
	//
	//	HideTrailingCount: pager.HasMore
	//
	// So the trailing header shows its label alone until the feed is
	// exhausted, at which point the flag goes false and the count appears.
	// The header keeps its key across that, so the reconciler patches the
	// badge in rather than replacing the band.
	//
	// This is only about the default GroupHeader. A Header override is handed
	// the Group unchanged — its Count included — and owns the decision
	// itself; there is no way for the widget to reach inside a view the
	// caller built.
	HideTrailingCount bool

	// Dividers inserts a theme hairline between consecutive rows of a group
	// (not after the last row, where the next header or the footer follows).
	Dividers bool

	// Empty is rendered in place of the rows when Items is empty. Nil renders
	// an empty list.
	Empty core.View
	// Footer is appended after the rows: a LoadMore, a Pagination, a
	// summary. It is rendered whether or not Items is empty, so a pager's
	// error state is still reachable when the first page failed.
	Footer core.View

	// Style is applied to the List after its defaults. The defaults shed the
	// theme Column's padding and gap: rows and headers are flush and spacing
	// is theirs to add, so a hairline divider is really one pixel tall.
	Style []core.StyleProp
}

func (g GroupedList[T]) Render(ctx *core.Context) *core.Node {
	items := make([]core.PropsAndChildren, 0, 2*len(g.Items)+len(g.Style)+6)
	items = append(items,
		core.Padding(0),
		core.Gap(0),
	)
	for _, sp := range g.Style {
		items = append(items, sp)
	}

	if len(g.Items) == 0 {
		if g.Empty != nil {
			items = append(items, g.Empty)
		}
	} else {
		items = appendRows(ctx, items, g.Items, g.Key, g.Row,
			g.GroupBy, g.Header, g.HideTrailingCount, g.Dividers, nil)
	}

	if g.Footer != nil {
		items = append(items, g.Footer)
	}
	return core.List(items...).Render(ctx)
}

// appendRows emits the grouped, keyed, optionally divided row sequence into
// a container's child list. It is shared with DataTable, which differs only
// in how a row is drawn (a cell Row rather than the caller's view), so it
// takes the row renderer as a function and an optional group-header
// override. wrap, when non-nil, decorates each rendered row (DataTable uses
// it for tap handling and selection tint); a nil wrap emits the row as-is.
// hideTrailingCount suppresses the last run's count badge; see
// GroupedList.HideTrailingCount for why an open run must not publish one.
func appendRows[T any](
	ctx *core.Context,
	items []core.PropsAndChildren,
	rows []T,
	key func(T) string,
	row func(T) core.View,
	groupBy func(T) Group,
	header func(Group) core.View,
	hideTrailingCount bool,
	dividers bool,
	wrap func(T, core.View) core.View,
) []core.PropsAndChildren {
	keyOf := func(i int, item T) string {
		if key != nil {
			return key(item)
		}
		return "row:" + itoa(i)
	}
	emit := func(i int, item T, last bool) {
		k := keyOf(i, item)
		v := row(item)
		if wrap != nil {
			v = wrap(item, v)
		}
		items = append(items, core.Keyed(k, v))
		if dividers && !last {
			items = append(items, core.Keyed("sep:"+k, Separator{}))
		}
	}

	runs := groupRuns(rows, groupBy)
	if runs == nil {
		for i, item := range rows {
			emit(i, item, i == len(rows)-1)
		}
		return items
	}
	// ri, not i: the row loop below indexes rows and would shadow it.
	for ri, run := range runs {
		var h core.View
		if header != nil {
			// An override owns its own counting: the Group goes through
			// untouched, trailing or not.
			h = header(run.Group)
		} else {
			h = GroupHeader{
				Group:     run.Group,
				HideCount: hideTrailingCount && ri == len(runs)-1,
			}
		}
		items = append(items, core.Keyed(groupHeaderKey(run.Group), h))
		for i := run.Start; i < run.End; i++ {
			emit(i, rows[i], i == run.End-1)
		}
	}
	return items
}
