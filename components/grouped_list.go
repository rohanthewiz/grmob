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

	// StickyHeaders pins each group's band to the top of the viewport while
	// its run scrolls underneath, releasing it when the next band arrives.
	// The reader always knows which month they are looking at, which is the
	// whole reason a feed is banded in the first place.
	//
	//	GroupedList[Sermon]{GroupBy: byMonth, StickyHeaders: true, ...}
	//
	// It is core.StickyHeader on the default GroupHeader, so it does nothing
	// without GroupBy — there are no bands in a flat list to pin.
	//
	// A Header override is handed the Group and builds its own view, which
	// this cannot reach into; such a header pins itself by putting
	// core.StickyHeader() in its own Style. That is the same division
	// HideTrailingCount draws, and for the same reason: a view the caller
	// built is the caller's.
	StickyHeaders bool

	// HeadingLevel places the default bands in the screen's outline. Zero is
	// level 2 — a band is a section of the screen an AppBar's title names at
	// level 1 — and a banded list inside a Card should say 3, while a feed on
	// a screen with no bar at all should say 1.
	//
	// It reaches GroupHeader, so it does nothing without GroupBy and nothing
	// under a Header override, on the same division StickyHeaders draws: a
	// view the caller built is the caller's to place in the outline. See
	// GroupHeader.HeadingLevel.
	HeadingLevel int

	// Dividers inserts a theme hairline between consecutive rows of a group
	// (not after the last row, where the next header or the footer follows).
	Dividers bool

	// Collapse turns the bands into disclosures whose runs the reader can
	// shut, with the state held by the caller. The zero value is the list as
	// it has always been.
	//
	//	Collapse: components.Collapse{
	//	    IsCollapsed: func(g components.Group) bool { return shut.Get()[g.Key] },
	//	    OnToggle:    func(g components.Group) { ... },
	//	}
	//
	// It does nothing without GroupBy — there are no bands in a flat list to
	// shut — on the same division StickyHeaders draws. Unlike StickyHeaders
	// it is *not* ignored under a Header override: the override owns the
	// band, and this owns whether the rows under it are emitted, which is not
	// something a view the caller built can reach. Such a caller renders their
	// own control and calls the same OnToggle.
	//
	// See Collapse for why the state is the caller's and why the two
	// functions are one type, and GroupHeader.Expanded for the band's ARIA
	// shape.
	Collapse Collapse

	// Empty is rendered in place of the rows when Items is empty. Nil renders
	// an empty list.
	Empty core.View
	// Footer is appended after the rows: a LoadMore, a Pagination, a
	// summary. It is rendered whether or not Items is empty, so a pager's
	// error state is still reachable when the first page failed.
	Footer core.View

	// OnEndReached turns the feed into an infinite one: it fires when the
	// reader scrolls within a few rows of the bottom, so the next page is
	// fetched without a tap.
	//
	//	GroupedList[Sermon]{
	//	    Items:        pager.Items,
	//	    OnEndReached: pager.LoadMore,
	//	    Footer:       LoadMore{HasMore: pager.HasMore, Loading: pager.Loading, Err: pager.Err, OnLoadMore: pager.LoadMore},
	//	}
	//
	// # Keep the Footer
	//
	// Auto-loading replaces the *tap*, not the tail. The footer is still
	// where "Loading…" and a failed page's Retry live, and it is still the
	// manual fallback on a target with no way to report the edge (a static
	// export, a browser without IntersectionObserver). A screen that drops
	// its LoadMore for this gains a feed that silently stops at whatever
	// page failed.
	//
	// Passing the pager's load function to both is correct and is the
	// intended shape: core.OnEndReached will not re-ask until the row count
	// changes, so a button tap and a scroll cannot double-load, and a page
	// that came back empty leaves the edge quiet until something else moves.
	//
	// Nil leaves the list exactly as it was — a manual pager, driven by its
	// footer.
	OnEndReached func()

	// Style is applied to the List after its defaults. The defaults shed the
	// theme Column's padding and gap: rows and headers are flush and spacing
	// is theirs to add, so a hairline divider is really one pixel tall.
	Style []core.StyleProp
}

func (g GroupedList[T]) Render(ctx *core.Context) *core.Node {
	items := make([]core.PropsAndChildren, 0, 2*len(g.Items)+len(g.Style)+7)
	items = append(items,
		core.Padding(0),
		core.Gap(0),
	)
	if g.OnEndReached != nil {
		items = append(items, core.OnEndReached(g.OnEndReached))
	}
	for _, sp := range g.Style {
		items = append(items, sp)
	}

	if len(g.Items) == 0 {
		if g.Empty != nil {
			items = append(items, g.Empty)
		}
	} else {
		items = appendRows(ctx, items, rowsSpec[T]{
			Rows:              g.Items,
			Key:               g.Key,
			Row:               g.Row,
			GroupBy:           g.GroupBy,
			Header:            g.Header,
			HideTrailingCount: g.HideTrailingCount,
			StickyHeaders:     g.StickyHeaders,
			HeadingLevel:      g.HeadingLevel,
			Dividers:          g.Dividers,
			Collapse:          g.Collapse,
			// No Wrap: a GroupedList row is the caller's view, emitted as
			// it came back. DataTable is the only decorator.
		})
	}

	if g.Footer != nil {
		items = append(items, g.Footer)
	}
	return core.List(items...).Render(ctx)
}

// rowsSpec is appendRows' parameter list, named. It carries the widget-level
// decisions about how a run of rows is grouped, keyed, banded and divided —
// everything the two collection widgets hand down unchanged — while ctx and
// the child slice stay positional, because those are the call's *subject*
// rather than its knobs.
//
// # Why a struct
//
// The parameters are not independent: they arrived together, they travel
// together, and four of them (HideTrailingCount, StickyHeaders, HeadingLevel,
// Dividers) describe one thing, the default band. As positional arguments
// that was twelve in a row, with two adjacent bools in the middle —
// hideTrailingCount and sticky — that the compiler cannot tell apart. A swap
// there produced a list whose last band hid its count and whose bands did not
// pin, which is two silent visual bugs from one edit, and the only reason it
// was caught is that both flags happen to have a test of their own.
//
// The type does not remove that risk by being a struct; it removes it by
// making the call site name each value. What it *does* remove structurally is
// the next one: a thirteenth knob added to a positional list shifts every
// argument after its insertion point, and a thirteenth field shifts nothing.
//
// # The field names are the widgets' field names
//
// Every field here is spelled exactly as the corresponding field on
// GroupedList and DataTable, so a call site reads as a copy rather than a
// translation — `HideTrailingCount: g.HideTrailingCount` is checkable by eye
// in a way that the eighth positional argument was not. Rows and Row are the
// one place the two widgets diverge (DataTable synthesizes its Row from its
// columns), and they are named for what appendRows does with them rather than
// for either widget's spelling.
type rowsSpec[T any] struct {
	// Rows is the flat, already-ordered run of items. Grouping is by run,
	// not by bucket; see GroupedList's type comment.
	Rows []T

	// Key returns a row's reconciler key. Nil falls back to positional keys.
	Key func(T) string
	// Row draws one item. Required. DataTable passes a closure over its
	// resolved columns rather than a caller-supplied function.
	Row func(T) core.View

	// GroupBy assigns each row to a Group; nil emits a flat run with no
	// bands. Header overrides the default band, and owns everything the
	// three fields below describe — a view the caller built is the caller's.
	GroupBy func(T) Group
	Header  func(Group) core.View

	// The default band's three knobs. Each is ignored under a Header
	// override, and each is documented on the widget field it comes from:
	// GroupedList.HideTrailingCount (why an open run must not publish a
	// count), .StickyHeaders (the pin goes on the band's own Style), and
	// GroupHeader.HeadingLevel (zero meaning the default tier).
	HideTrailingCount bool
	StickyHeaders     bool
	HeadingLevel      int

	// Dividers inserts a hairline between consecutive rows of a run, never
	// after the last one, where a band or the footer follows.
	Dividers bool

	// Collapse is the caller-owned collapse state, if the bands are
	// disclosures. The zero value is "nothing collapses" and is what
	// DataTable passes, for the reason its own type comment gives about band
	// headings and rowgroups.
	//
	// It reaches two places rather than one: the default band, which becomes
	// a button carrying aria-expanded, and the *row emission*, which is the
	// half a Header override cannot own — a caller who builds their own band
	// still gets a run that hides, and wires their control to the same
	// OnToggle.
	//
	// components.CollapseBand is what an override wires it to. It is the
	// disclosure the default band builds, on its own and without the band's
	// chrome, so an override author places a correct control in a layout of
	// their own instead of rebuilding a button, an aria-expanded and a heading
	// wrapper from the argument in components.disclosure.
	Collapse Collapse

	// Wrap decorates each rendered row before it is keyed. DataTable uses it
	// for the tap target and the selection tint; a nil Wrap emits the row as
	// it came back from Row. It is the one field with no widget field behind
	// it — it is how the shared code lets one caller add behavior the other
	// does not have.
	Wrap func(T, core.View) core.View
}

// appendRows emits the grouped, keyed, optionally divided row sequence into
// a container's child list. It is shared with DataTable, which differs only
// in how a row is drawn (a cell Row rather than the caller's view) and in
// wanting each row decorated, so the drawing and the decoration are both
// functions on the spec.
//
// ctx and items stay positional: they are what the call operates on and they
// are the same two arguments at every call site. Everything that varies is in
// spec, where it is named — see rowsSpec for why.
func appendRows[T any](
	ctx *core.Context,
	items []core.PropsAndChildren,
	spec rowsSpec[T],
) []core.PropsAndChildren {
	keyOf := func(i int, item T) string {
		if spec.Key != nil {
			return spec.Key(item)
		}
		return "row:" + itoa(i)
	}
	emit := func(i int, item T, last bool) {
		k := keyOf(i, item)
		v := spec.Row(item)
		if spec.Wrap != nil {
			v = spec.Wrap(item, v)
		}
		items = append(items, core.Keyed(k, v))
		if spec.Dividers && !last {
			items = append(items, core.Keyed("sep:"+k, Separator{}))
		}
	}

	runs := groupRuns(spec.Rows, spec.GroupBy)
	if runs == nil {
		for i, item := range spec.Rows {
			emit(i, item, i == len(spec.Rows)-1)
		}
		return items
	}
	// ri, not i: the row loop below indexes spec.Rows and would shadow it.
	for ri, run := range runs {
		// Copied out of the range variable before the closure below captures
		// it. Correct without the copy under Go 1.22's per-iteration scoping,
		// and written this way anyway: the capture is the kind of thing that
		// gets moved into a helper, where the scoping rule no longer applies.
		group := run.Group

		var h core.View
		if spec.Header != nil {
			// An override owns its own counting *and* its own control: the
			// Group goes through untouched, trailing or not, and a collapsible
			// band built here would be a second control for the same run.
			//
			// components.CollapseBand is how the override builds one that
			// matches — it takes the caller's own Collapse, so the control and
			// the row hiding below answer to one state rather than two.
			h = spec.Header(group)
		} else {
			gh := GroupHeader{
				Group:        group,
				HideCount:    spec.HideTrailingCount && ri == len(runs)-1,
				HeadingLevel: spec.HeadingLevel,
			}
			if spec.Collapse.active() {
				// Expanded, not collapsed: ARIA states the affirmative, and
				// deriving it here rather than at the band keeps the widget's
				// vocabulary ("which groups are shut") and ARIA's ("is this
				// open") from having to agree in two places.
				gh.Expanded = !spec.Collapse.collapsed(group)
				gh.OnToggle = func() { spec.Collapse.OnToggle(group) }
			}
			if spec.StickyHeaders {
				// Onto the band's own Style, not around it in a wrapper: the
				// pin has to be on the node the list sees as its child, and a
				// wrapper would put a plain Box there with the sticky band
				// hidden inside it, where no renderer looks.
				gh.Style = append(gh.Style, core.StickyHeader())
			}
			h = gh
		}
		items = append(items, core.Keyed(groupHeaderKey(group), h))

		// A shut run emits nothing at all — not hidden children, none. The
		// band keeps its key across the toggle, so the reconciler patches the
		// rows away and leaves the header alone, and a re-open costs one
		// render rather than a remount of the band.
		//
		// Checked whether or not the band is the default one, which is the
		// half of Collapse a Header override cannot own: the override draws a
		// control, and this decides what the control is about.
		if spec.Collapse.hides(group) {
			continue
		}
		for i := run.Start; i < run.End; i++ {
			emit(i, spec.Rows[i], i == run.End-1)
		}
	}
	return items
}
