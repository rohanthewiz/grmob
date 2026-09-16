# Package comps — Lists & tables

```go
import "github.com/rohanthewiz/grmob/comps"
```

List, settings and input rows, grouped and paged lists, data tables and timelines.

One of 7 topic pages of [package comps](comps.md), which has the package overview and an index of every topic. This page documents the declarations in `comps/list_row.go`, `comps/settings_row.go`, `comps/input_row.go`, `comps/grouped_list.go`, `comps/grouping.go`, `comps/paging.go`, `comps/data_table.go`, `comps/timeline.go`.

## Index

- [Constants](#constants) — `ConcernPartialSort`
- [`type CheckboxRow`](#type-checkboxrow)
    - [`func (CheckboxRow) Render`](#func-checkboxrow-render)
- [`type Collapse`](#type-collapse)
- [`type CollapseBand`](#type-collapseband)
    - [`func (CollapseBand) Render`](#func-collapseband-render)
- [`type Column`](#type-column)
- [`type DataTable`](#type-datatable)
    - [`func (DataTable) Render`](#func-datatable-render)
- [`type Group`](#type-group)
- [`type GroupHeader`](#type-groupheader)
    - [`func (GroupHeader) Render`](#func-groupheader-render)
- [`type GroupedList`](#type-groupedlist)
    - [`func (GroupedList) AutoLoadWithheld`](#func-groupedlist-autoloadwithheld)
    - [`func (GroupedList) Render`](#func-groupedlist-render)
- [`type InputRow`](#type-inputrow)
    - [`func (InputRow) Render`](#func-inputrow-render)
- [`type ListRow`](#type-listrow)
    - [`func (ListRow) Render`](#func-listrow-render)
- [`type LoadMore`](#type-loadmore)
    - [`func (LoadMore) Render`](#func-loadmore-render)
- [`type Pagination`](#type-pagination)
    - [`func (Pagination) Render`](#func-pagination-render)
- [`type Sort`](#type-sort)
- [`type SwitchRow`](#type-switchrow)
    - [`func (SwitchRow) Render`](#func-switchrow-render)
- [`type Timeline`](#type-timeline)
    - [`func (Timeline) Render`](#func-timeline-render)
- [`type TimelineEvent`](#type-timelineevent)

## Constants

ConcernPartialSort: a DataTable sorted client-side (the active Sort names a column with a Less) while its Pagination declares a PageCount — which is the caller saying the server chooses which rows arrive. The table can only order the window it was handed, so the header claims an ordering over the whole table and delivers one over one page of it. The fix is to drop the column's Less and keep Sortable, letting OnSort go into the query.

Reported through core.ReportConcern rather than detected in core: this is a widget-level contract, and core has no business knowing what a DataTable is. Debug mode only, like every other concern.

```go
const ConcernPartialSort = "partial-sort"
```

<small>[comps/data_table.go:80](https://github.com/rohanthewiz/grmob/blob/master/comps/data_table.go#L80)</small>

## Types

### type CheckboxRow

```go
type CheckboxRow struct {
	// Title is the choice's text. It is also the checkbox's accessible label.
	Title string

	// Subtitle is the secondary line under the title, and the checkbox's
	// accessibility hint.
	Subtitle string

	// Leading is an optional icon or avatar before the text.
	Leading core.View

	// Checked is the caller's current value.
	Checked bool

	// OnToggle receives the new value. It is a setter; see SwitchRow.OnToggle.
	OnToggle func(checked bool)

	// Disabled disables both the checkbox and the row.
	Disabled bool

	// Style is passed to the underlying ListRow and so beats its defaults.
	Style []core.StyleProp
}
```

CheckboxRow is SwitchRow with a core.Checkbox on the trailing edge: a named option a form or a later action will read — "Also delete attachments", a filter, a debug flag. Everything in SwitchRow's doc applies unchanged, including OnToggle being a setter.

	comps.CheckboxRow{Title: "Also delete attachments", Checked: purge.Get(), OnToggle: purge.Set}

#### Trailing here, leading in a ListRow

A checkbox goes on one edge or the other depending on what the row is, and grmob's examples keep to one rule:

	the row is…                          checkbox   build it with
	──────────────────────────────────   ────────   ──────────────────────────
	an option with a name (a setting,    trailing   CheckboxRow
	a filter, "also do X")
	the thing being marked (a task       leading    ListRow{Leading: core.Checkbox}
	done, a sentence agreed to)

An option trails because it shares its list with SwitchRows, and a column of controls on one edge is what lets a reader scan a settings list; the title is the label and the control its value. A marked item leads because the mark is read before the content, as a form's "I agree to the terms" box is on every platform, and a task list's ticks line up down the side the eye starts from. Tutorial lesson 2.6, "Two booleans: checkbox and switch", draws the leading form; the demo toggles across the tutorial are options and use this row.

Inside a comps.FormField the field owns the label, so leave Title empty and put the sentence in the field's Label, or keep Title and give the field no label; the row renders the same either way.

<small>[comps/settings_row.go:137](https://github.com/rohanthewiz/grmob/blob/master/comps/settings_row.go#L137)</small>

#### func (CheckboxRow) Render

```go
func (r CheckboxRow) Render(ctx *core.Context) *core.Node
```

Render builds the ListRow described in SwitchRow's doc.

<small>[comps/settings_row.go:162](https://github.com/rohanthewiz/grmob/blob/master/comps/settings_row.go#L162)</small>

### type Collapse

```go
type Collapse struct {
	// IsCollapsed reports whether a group's run is hidden. Nil means no.
	IsCollapsed func(Group) bool
	// OnToggle is called with the group whose band was pressed. Nil leaves
	// the bands as plain headings — see GroupHeader.Expanded.
	OnToggle func(Group)
}
```

Collapse is caller-owned collapse state for a banded collection: which groups are shut, and what to do when one is pressed.

#### Why the caller holds it

GroupedList calls no hook, which is what lets it be rendered conditionally — inside a core.IfElse against a pager's loaded flag, say — without disturbing the caller's hook cursor. Owning collapse state would end that, and the widget is the wrong place for it anyway: which months are shut is screen state, it usually wants to survive a pager reload, and a screen that wants "collapse all" has no way to reach inside a widget's NewState.

So the screen keeps a set and answers two questions:

	shut := core.NewState(ctx, map[string]bool{})
	GroupedList[Sermon]{
	    GroupBy: byMonth,
	    Collapse: comps.Collapse{
	        IsCollapsed: func(g comps.Group) bool { return shut.Get()[g.Key] },
	        OnToggle: func(g comps.Group) {
	            next := maps.Clone(shut.Get())
	            next[g.Key] = !next[g.Key]
	            shut.Set(next)
	        },
	    },
	}

comps.Accordion is the other answer to the same question and stays the right one for a single section: it owns its state, and the hook obligations that come with it are documented on the widget. A list of twenty bands is where owning the state stops being a convenience — twenty independent NewStates that a reorder cannot move, and no way to shut them all.

#### One type, two functions

They are two halves of one fact and are useless apart, which is the same argument core.ValueRange makes for its three numbers. As two fields on GroupedList a caller could supply either alone: IsCollapsed without OnToggle is a list with rows nobody can bring back, and OnToggle without IsCollapsed is a control that announces a state it does not have. The zero value is "nothing collapses", which is what every list that has never heard of this keeps doing.

<small>[comps/grouping.go:122](https://github.com/rohanthewiz/grmob/blob/master/comps/grouping.go#L122)</small>

### type CollapseBand

```go
type CollapseBand struct {
	// Collapse is the caller's own collapse state — the same value given to
	// GroupedList. The zero value builds a plain heading; see above.
	Collapse Collapse

	// Group is the run this control is about. Its Label names both the heading
	// and the button unless Content replaces the words.
	Group Group

	// HeadingLevel is where the band sits in the screen's outline. Zero is
	// level 2, the same default GroupHeader takes and for the same reason —
	// see GroupHeader.HeadingLevel, which is the field this mirrors.
	HeadingLevel int

	// Content goes inside the button, after the chevron. Empty takes the
	// group's Label in the band's own type, which is the common case and the
	// reason this is a slice rather than a required field.
	//
	// Whatever goes here is presentational: a reader does not descend into a
	// button's children, and the button's name comes from Group.Label. So an
	// icon needs no aria-hidden and a count put here stops being announced —
	// which is why the default band keeps its badge outside the control.
	Content []core.View

	// Style is applied to the heading wrapper, which is the node a caller's
	// layout sees. GroupedList's own band puts core.FlexGrow(1) here so the
	// count badge sits hard against the trailing edge; a caller arranging
	// their own row decides that for themselves.
	Style []core.StyleProp

	// ControlStyle is applied to the button, which is the node a finger lands
	// on. This is where a caller's own chrome belongs.
	//
	// # The question it answers
	//
	// A band's padding on the row that holds the control is dead space: the
	// button fills the box it was given and the insets are outside it, so the
	// 16px before the chevron is a place a press does nothing. GroupHeader
	// used to be built that way and is not any more (see bandInsets), and a
	// caller writing their own row inherits the same question one level out:
	//
	//	core.Row(                                    core.Row(
	//	    core.PaddingHorizontal(16),                  CollapseBand{
	//	    CollapseBand{...},              becomes          Collapse: shut, Group: g,
	//	    Badge{Text: count},                              ControlStyle: []core.StyleProp{
	//	)                                                        core.PaddingLeft(16),
	//	                                                         core.PaddingRight(8)},
	//	                                                 },
	//	                                                 Badge{Text: count},
	//	                                                 core.PaddingRight(16),
	//	                                             )
	//
	// Same pixels; a target that reaches the leading edge rather than starting
	// 16px into it.
	//
	// On an inactive Collapse there is no button and this lands on the plain
	// heading, for the same reason GroupHeader.ControlStyle does: the two
	// branches must be the same size, or a band changes shape on the day
	// somebody gives it a handler.
	ControlStyle []core.StyleProp

	// ChevronStyle types the ▸/▾ glyph, applied after the band's own Caption
	// type so a caller's declaration wins.
	//
	// # Why a band needs it and GroupHeader does not
	//
	// The Caption default is right for the default band, whose words are
	// Caption too — disclosure's ChevronStyle doc says why a chevron has to
	// match the tier beside it. A band given Content has chosen its own words
	// and their size, and the glyph cannot follow them there: at Caption next to
	// a Body-sized title it is a 6px speck that reads as a bullet, which is what
	// the tutorial's chapter cards showed. So the caller who picked the words
	// picks the glyph's tier too.
	//
	// An inactive Collapse draws no chevron, so this has nothing to land on
	// there and is ignored.
	ChevronStyle []core.StyleProp
}
```

CollapseBand is the disclosure control the default band builds, on its own: a heading wrapping a button that carries aria-expanded and toggles one group's run. No Surface, no padding, no count badge — the chrome is the caller's.

#### The gap it closes

GroupedList.Collapse reaches past a Header override for the row emission, so an override's run still hides, and it stops at the override for the \*control\*, because a band the widget also built would be a second control for the same run. That division is right and it left the override author holding three things at once: a button, an aria-expanded that has to be stated on every pass open or shut, and a heading wrapper whose nesting order — heading around button, named explicitly so the chevron never reaches it — is four paragraphs of argument in comps.disclosure, which is unexported.

GroupHeader is still the answer when the whole default band will do; it takes Expanded and OnToggle and builds all of it. This is the answer when it will not:

	Header: func(g comps.Group) core.View {
	    return core.Row(
	        core.PaddingHorizontal(16),
	        comps.CollapseBand{Collapse: shut, Group: g},
	        Avatar{Name: leader[g.Key]},
	        Badge{Text: strconv.Itoa(g.Count)},
	    )
	},

The Collapse passed here is the caller's own — the same value handed to GroupedList — which is what keeps the control and the row hiding answering to one state. An override that built its control from a second Collapse would have a chevron pointing one way and a run obeying the other.

#### An inactive Collapse builds a heading and no control

The zero Collapse — and one with IsCollapsed and no OnToggle — produces the label in a plain heading, exactly as GroupHeader's own non-disclosure branch does. Not a button with a dead handler: an expansion stated with nothing to toggle it is announced on both web targets, is silently nothing on Android, and is what core.AuditTree reports as ConcernInertDisclosure. Building one here would be building the thing the audit exists to find.

<small>[comps/grouping.go:194](https://github.com/rohanthewiz/grmob/blob/master/comps/grouping.go#L194)</small>

#### func (CollapseBand) Render

```go
func (b CollapseBand) Render(ctx *core.Context) *core.Node
```

<small>[comps/grouping.go:273](https://github.com/rohanthewiz/grmob/blob/master/comps/grouping.go#L273)</small>

### type Column

```go
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

	// Width fixes the column's content width in px, the same on every row, so
	// a column of short values ("Mar 1", "Mar 22") stays aligned rather than
	// hugging each row's own text. The cell's padding is outside it. 0 leaves
	// the column to Weight or to its content.
	//
	// With Weight also set, Width is the least the column takes and the
	// weight shares out the slack above it. A weightless column with a Width
	// does not shrink either, so a narrow row keeps every fixed column whole
	// and squeezes the weighted ones.
	//
	// It is a width on an inner, unpadded box rather than on the cell, because
	// the cell carries padding and the targets do not agree on whether a
	// padded box's width includes it (CSS's content-box default says no); an
	// unpadded box has one width on all four.
	Width float64

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
```

Column describes one column of a DataTable: its header, how a row's value is drawn, how much of the row's width it takes, and whether it sorts.

<small>[comps/data_table.go:12](https://github.com/rohanthewiz/grmob/blob/master/comps/data_table.go#L12)</small>

### type DataTable

```go
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

	// StickyHeaders pins each group band while its run scrolls under it, as
	// in GroupedList.
	//
	// It pins the *group* bands, not the column header. The column header is
	// a sibling of the body list rather than a row inside it — that is what
	// keeps it from scrolling away in the first place — and both natives
	// implement pinning inside their lazy container only, so there is nothing
	// there for the marker to mean.
	StickyHeaders bool

	// HeadingLevel places the default group bands in the screen's outline, as
	// in GroupedList. Zero is level 2; a table nested inside a Card should say
	// 3. It reaches GroupHeader, so it does nothing without GroupBy and
	// nothing under a Header override.
	//
	// It says nothing about the *column* header, whose cells carry
	// core.RoleColumnHeader and no level — aria-level is defined for heading,
	// listitem and row, and pointedly not for columnheader.
	HeadingLevel int

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
```

DataTable is a header row over a virtualized, keyed body of cell rows, with controlled sorting, optional grouping, optional pagination and a compact mode for narrow screens.

	┌ Column ────────────────────────────────────────────┐
	│ ┌ Row (header) ────────────────────────────────┐   │
	│ │ Title ▲          Teacher         Date        │   │  <- tap sorts
	│ └──────────────────────────────────────────────┘   │
	│ ┌ List (body) ─────────────────────────────────┐   │
	│ │ ▒ January 2026                          (2)  │   │  <- GroupHeader
	│ │ Grace and Truth   Pastor Ray     Jan 12      │   │  <- key Key(row)
	│ │ ─────────────────────────────────────────    │   │  <- Dividers
	│ │ The Vine          Guest          Jan 5       │   │
	│ └──────────────────────────────────────────────┘   │
	│ ‹ Prev            Page 1 of 4            Next ›    │  <- Pagination
	└────────────────────────────────────────────────────┘

#### Order of operations

Rows are sorted, then paged, then grouped: the sort decides the order, paging takes a window of it, and grouping reads runs off that window. Grouping last is what makes a client-side page's headers agree with its rows, and sorting first is what makes group runs follow the sort (a table sorted by teacher and grouped by month shows a month once per teacher run — the honest rendering; see groupRuns).

#### Sorting sorts the rows you hand it

A column with Less is sorted by the table, over Rows — all of Rows, and only Rows. That is right whenever Rows is the whole set: a slice the caller holds, or a fetch that completed, including the client-paged case where Pagination slices rows the table already has.

It is wrong when Rows is a window somebody else chose. An offset pager's "the three pages loaded so far", or a server-paged table declaring PageCount, is a subset selected under some other ordering. Sorting that yields the alphabetically-first of the rows that happen to be loaded, and puts it under a header claiming to be the alphabetically-first rows. The result carries no sign of which it is, which is exactly what makes the mistake worth naming: a partial sort looks like a working one, and the rows that disprove it are the ones not fetched.

For that table set Sortable instead of Less. The header still taps and OnSort still fires; the sort goes into the next query, and the server returns a window of the ordering the reader actually asked for.

One shape of this is detectable and is reported as a debug concern: Sort active on a Less column while Pagination declares a PageCount, since a declared PageCount is the caller saying the server does the paging. See ConcernPartialSort. The append-style case has no such signal — a slice of accumulated pages is indistinguishable from a complete one — so that half is left to this documentation.

#### Who owns what

Everything is controlled. The table holds no state and calls no hook: Sort is read from the caller and OnSort reports the tap, Pagination.Page is read and OnChange reports the step. Only the \*work\* is optionally the table's: a column with Less is sorted here, a Pagination with PageSize and no PageCount is sliced here. Both are pure functions of the inputs.

#### Compact

A phone is too narrow for a five-column table. Compact drops every Narrow column and leaves the rest; the header and cells stay in step because both walk the same filtered column list. The caller decides when the table is compact — from a breakpoint, a settings toggle, an orientation event.

<small>[comps/data_table.go:149](https://github.com/rohanthewiz/grmob/blob/master/comps/data_table.go#L149)</small>

#### func (DataTable) Render

```go
func (d DataTable[T]) Render(ctx *core.Context) *core.Node
```

<small>[comps/data_table.go:229](https://github.com/rohanthewiz/grmob/blob/master/comps/data_table.go#L229)</small>

### type Group

```go
type Group struct {
	Key   string
	Label string
	Count int

	// Trailing reports whether this is the last run in the collection — the
	// one an append-style pager extends, and the only one whose Count is
	// still open (see GroupedList.HideTrailingCount for why that matters).
	//
	// Filled in by the widget, like Count: a value a GroupBy callback sets is
	// overwritten. The two grouping walks agree about it — groupRuns marks its
	// last run and trailingRun marks the run it went looking for — so a band
	// and the auto-load decision are answering the same question.
	//
	// It exists for the Header override. HideTrailingCount is documented as a
	// decision an override "owns itself", and until this field an override
	// could not make it: the rule is *don't publish an open run's count*, and
	// nothing handed to the override said which run was open. Deriving it
	// meant re-walking Items with the same GroupBy the widget had just walked.
	//
	//	Header: func(g comps.Group) core.View {
	//	    label := g.Label
	//	    if !g.Trailing || !pager.HasMore {
	//	        label += " (" + strconv.Itoa(g.Count) + ")"
	//	    }
	//	    return ...
	//	}
	Trailing bool

	// AutoLoadWithheld reports that this run being shut is why the list is
	// rendering without its edge sensor — the state
	// GroupedList.OnEndReached's "a shut trailing group withholds it" section
	// describes, attached to the run that caused it.
	//
	// True on at most one group of a list, and always a Trailing one. False
	// throughout a list with no OnEndReached, since there is no sensor to
	// withhold; false on every band of a DataTable, which has no edge sensor
	// at all.
	//
	// # Why the widget states it rather than the band deriving it
	//
	// GroupedList.AutoLoadWithheld answers the *caller*, who can then build a
	// footer. A Header override is a different reader in a different place: it
	// is handed a Group and nothing else, so a band that wanted to say
	// "collapsed — auto-load is off here" had to close over Items, GroupBy and
	// Collapse and re-derive an answer the widget had just computed.
	//
	//	Header: func(g comps.Group) core.View {
	//	    band := core.Row(comps.CollapseBand{Collapse: shut, Group: g})
	//	    if g.AutoLoadWithheld {
	//	        band = core.Row(band, comps.Badge{Text: "paused"})
	//	    }
	//	    return band
	//	},
	//
	// Deriving it in the override is also easy to get subtly wrong, which is
	// the stronger half of the argument. The composite is Trailing *and* the
	// run is hidden *and* a sensor was given — and "the run is hidden" is
	// Collapse.hides, which needs OnToggle as well as IsCollapsed: a caller
	// with a predicate and no handler hides nothing, so their bands would
	// announce a pause the list is not taking. One statement, from the value
	// that made the decision.
	//
	// The default GroupHeader ignores it. What a band says about a paused feed
	// is a wording decision, and the default band's vocabulary is a label and a
	// count; this is the fact an override needs to make that decision, not a
	// decision the widget makes for it.
	AutoLoadWithheld bool
}
```

Group identifies one run of rows in a grouped collection. GroupBy callbacks return Key and Label; the grouping engine fills Count. Key is what decides group identity and becomes the header's reconciler key, so it should be stable and comparable ("2026-01"), while Label is what people read ("January 2026").

<small>[comps/grouping.go:10](https://github.com/rohanthewiz/grmob/blob/master/comps/grouping.go#L10)</small>

### type GroupHeader

```go
type GroupHeader struct {
	Group Group

	// HideCount drops the trailing badge; the label stands alone.
	HideCount bool

	// HeadingLevel is where the band sits in the screen's outline. Zero is
	// level 2 — a band is a section of the screen whose name an AppBar's
	// title carries at level 1 — and a banded list nested inside a Card (also
	// level 2) should say 3.
	//
	// The field is also how a screen with no bar stops lying. A bandless feed
	// used to start its outline at 2 with no 1 above it, which is a soft lint
	// on the web and nothing at all to either native; the alternative was
	// announcing a screen's name and its March band as peers, so 2 stayed as
	// the lesser of two wrongs. Now the caller in that position writes 1.
	//
	// See headingLevel in heading.go for the package's outline and for how to
	// ask for a heading with no tier at all.
	HeadingLevel int

	// Expanded and OnToggle turn the band into a disclosure: the label becomes
	// a button carrying aria-expanded, with a chevron ahead of it, and the run
	// beneath is the caller's to hide.
	//
	// # Both or neither
	//
	// OnToggle nil is the ordinary band, and Expanded is then ignored. That is
	// not a silent drop of a stated fact — it is what core.ExpandedUnset
	// means, and a caller who states one without the other is reported by
	// core.SetDebugMode as ConcernInertDisclosure, because an expansion with
	// no handler is announced on both web targets and is silently nothing on
	// Android. See comps.disclosure, which is where the pairing and the
	// ARIA shape are argued.
	//
	// # The badge stays outside the button
	//
	// A button's children are presentational — a reader does not descend into
	// them — so a count inside the control would stop being announced, and the
	// count is real content rather than chrome. It therefore sits beside the
	// button rather than within it, which is also what keeps the heading named
	// "January 2026" instead of "January 2026 3".
	//
	// The cost is that the badge is not part of the tap target: the button
	// runs from the band's leading edge and stops where the count begins.
	// That is the ordinary shape of a header row with a trailing badge, and
	// the alternative — folding the count into the button's accessible name —
	// would be assembling an English phrase in the renderer, which is the move
	// Chip's ", selected" suffix was deleted for.
	//
	// The band's own insets used to be excluded too, which was not the same
	// kind of cost: the count is content a press should not toggle, and the
	// padding is chrome. They are on the control now — see bandInsets — so
	// what the target excludes is exactly the thing that is not the control.
	Expanded bool
	OnToggle func()

	// Style is applied to the band Row after its defaults: its fill, its
	// margins, core.StickyHeader, a width.
	//
	// Not its insets. The band's padding lives on the control now — see
	// bandInsets for why and for the picture — so a caller who wants the
	// label flush left writes it in ControlStyle. Putting a
	// core.PaddingLeft(0) here still reaches the Row, where it is a no-op on
	// every side but the badge's own trailing inset.
	Style []core.StyleProp

	// ControlStyle is applied to the band's control after the insets: the
	// node a finger lands on, and therefore where the band's own padding is.
	//
	// A caller indenting a nested band's label, or shipping a denser feed,
	// reaches for this rather than Style — and gets a tap target that moves
	// with the chrome instead of a strip in the middle of it.
	//
	// On a band with no OnToggle there is no control, and this lands on the
	// plain heading that stands in for one. That is deliberate: the two
	// branches are the same band geometrically, and a knob that silently did
	// nothing on one of them would be a relayout the first time a caller
	// added a handler.
	//
	// "The same band geometrically" is a claim about the chrome, and it is
	// exact: browser.mjs measures both branches in every bundled theme and the
	// control's leading and trailing edges are identical. The band's HEIGHT is
	// not, by one point — the disclosure's button holds a chevron the plain
	// band does not, a control is as tall as its tallest child plus its own
	// insets, and that glyph's line box exceeds the caption's in the font stacks
	// Chrome resolves. That is content, not chrome, and it is checked as the
	// equation it is rather than waved at: the difference between the two bands
	// must equal the chevron's overhang over the words exactly, so chrome
	// drifting between the branches still fails.
	ControlStyle []core.StyleProp
}
```

GroupHeader is the default band rendered above each group in GroupedList and DataTable: the label in bold caption ink on the theme's Surface, with the row count as a badge pinned to the trailing edge. It is exported so a caller can render it with a different Count or Label from inside a Header override, or reuse it in a hand-built list.

<small>[comps/grouping.go:442](https://github.com/rohanthewiz/grmob/blob/master/comps/grouping.go#L442)</small>

#### func (GroupHeader) Render

```go
func (h GroupHeader) Render(ctx *core.Context) *core.Node
```

<small>[comps/grouping.go:616](https://github.com/rohanthewiz/grmob/blob/master/comps/grouping.go#L616)</small>

### type GroupedList

```go
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
	//	Collapse: comps.Collapse{
	//	    IsCollapsed: func(g comps.Group) bool { return shut.Get()[g.Key] },
	//	    OnToggle:    func(g comps.Group) { ... },
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
	//
	// It interacts with OnEndReached, which is the one place two of this
	// widget's features are in tension: a shut *trailing* group withholds the
	// edge sensor, because a page fetched into a hidden run appears nowhere
	// and silences the guard behind it. See OnEndReached.
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
	//
	// # A shut trailing group withholds it
	//
	// The two features are in tension and neither one is wrong. An append
	// pager can only ever extend the *last* run (see the type comment), and a
	// shut run emits no rows at all — so when the reader has collapsed the
	// group at the bottom of the feed, a page that arrives is a page that
	// appears nowhere.
	//
	// Left alone, that is worse than it sounds. core.OnEndReached will not
	// re-ask until the List's child count changes, and a page absorbed into a
	// hidden run changes nothing, so the *first* auto-load spends a page and
	// every one after it is refused. The feed reads as exhausted, the pager's
	// offset has moved, and nothing anywhere says so.
	//
	//	shut trailing group, auto-load left on
	//	  scroll to bottom -> fetch page 3 -> 20 rows into a hidden run
	//	  -> child count unchanged -> the guard closes -> the feed is over
	//
	// So the prop is withheld while that group is shut, and comes back the
	// moment it is opened. Suppressing rather than starving is the honest
	// half: the reader has said they do not want to see this run, and
	// fetching more of it in the background is work nobody asked for and
	// nobody can look at.
	//
	// What stays reachable is the Footer, which is why "keep the Footer"
	// above is not only about static targets. comps.LoadMore is a button
	// that calls the same function directly, so a reader who wants the next
	// page while the last group is shut has one — and pressing it is a
	// deliberate act, where a scroll is not.
	//
	// This is a fact about the *last* run only. A shut group anywhere above
	// it hides its rows and changes nothing about the edge, because the pager
	// was never going to extend it.
	//
	// AutoLoadWithheld is how a caller finds out it has happened. Without it
	// the withholding is invisible from outside — a feed that has stopped
	// fetching and a feed that has run out look identical — and a screen that
	// hides its footer on the strength of auto-loading has no way out at all.
	OnEndReached func()

	// Style is applied to the List after its defaults. The defaults shed the
	// theme Column's padding and gap: rows and headers are flush and spacing
	// is theirs to add, so a hairline divider is really one pixel tall.
	Style []core.StyleProp
}
```

GroupedList is a virtualized, keyed list of typed items with optional run-length group headers and a footer slot for a pager: the shape of an archive feed — sermons by month, transactions by day, messages by sender.

#### What it settles

Every paged screen in an app builds the same core.List by hand: a keyed row per item, an empty note when there are none, and a "Load more" tail. The widget owns that assembly so the screen owns only the three things that differ — how to key an item, how to draw one, and where its pages come from.

	┌ List ──────────────────────────────────┐
	│ ▒ January 2026                     (3) │  <- GroupHeader, key "group:2026-01"
	│   Row(item)                            │  <- key Key(item)
	│   Row(item)                            │
	│   Row(item)                            │
	│ ▒ December 2025                    (1) │
	│   Row(item)                            │
	│           [ Load more ]                │  <- Footer
	└────────────────────────────────────────┘

#### Grouping is by run, not by bucket

GroupBy is evaluated in Items order and a header is emitted wherever the key changes (see groupRuns). Items must therefore arrive already ordered by the grouping — which a feed sorted by date, grouped by month, always is — and an offset pager that appends pages can only ever grow the last group, so nothing above the fold moves on "Load more".

#### Identity

core.List keeps row state attached to keys across insertions and reorders, so Key must be unique across the whole list and stable across renders. A nil Key falls back to positional keys, which is correct for a static list and loses row state on reorder for a live one, exactly as core.List documents. Group headers take "group:"+Group.Key, so a row key can never collide with a header even when a caller keys rows by the same string.

#### No hooks

The widget holds no state and calls no hook, so it may be rendered conditionally — inside core.IfElse against a pager's loaded flag, say — without disturbing the caller's hook cursor.

<small>[comps/grouped_list.go:49](https://github.com/rohanthewiz/grmob/blob/master/comps/grouped_list.go#L49)</small>

#### func (GroupedList) AutoLoadWithheld

```go
func (g GroupedList[T]) AutoLoadWithheld() bool
```

AutoLoadWithheld reports whether this list is about to render \*without\* the edge sensor it was given — the state OnEndReached's "a shut trailing group withholds it" section describes.

##### Why a caller needs to be able to ask

Withholding the sensor is the right call and it is invisible. The reader collapses the last group, scrolling quietly stops fetching, and the only cue anywhere is the absence of new rows — which is exactly what a feed that has genuinely run out looks like. Nothing in the tree says which of the two it is, and until this method nothing outside Render could work it out either: the answer is composed from Items, GroupBy and Collapse, three fields the caller holds separately and none of which means anything alone.

The footer is where that matters, because the footer is the way out. A screen whose Footer is a bare LoadMore is already fine — the button is visible and calls the same function — and the shape that is not fine is the common one:

	Footer: hasMore ? LoadMore{...} : nil     // "the feed is over"

With the last group shut, hasMore is true, so that one is fine too. The trap is the other direction — a footer hidden while auto-load is believed to be doing the work:

	list := comps.GroupedList[Sermon]{
	    Items: pager.Items, GroupBy: byMonth, Collapse: shut,
	    OnEndReached: pager.LoadMore,
	}
	// Shown when there is more to fetch *and* nothing is fetching it.
	if pager.HasMore && list.AutoLoadWithheld() {
	    list.Footer = comps.LoadMore{HasMore: true, OnLoadMore: pager.LoadMore, ...}
	}

A method rather than a second field, because it is derived: a field would be a copy of a fact the widget already computes, free to disagree with it the moment either the items or the collapse state moved. The value has to be built before it can be asked, which is why the example assigns Footer after the literal — the same ordering any derived-from-itself decision takes.

It answers false when OnEndReached is nil. There is no sensor to withhold on a manual pager, and a caller asking this question is asking whether the automatic path is off \*right now\*, not whether the last group happens to be shut. Collapse.hides is the field to ask for that.

<small>[comps/grouped_list.go:282](https://github.com/rohanthewiz/grmob/blob/master/comps/grouped_list.go#L282)</small>

#### func (GroupedList) Render

```go
func (g GroupedList[T]) Render(ctx *core.Context) *core.Node
```

<small>[comps/grouped_list.go:286](https://github.com/rohanthewiz/grmob/blob/master/comps/grouped_list.go#L286)</small>

### type InputRow

```go
type InputRow struct {
	// Value is the field's text. The field is fully controlled: it displays
	// exactly this, and OnChange is the only way it changes.
	Value string

	// Placeholder is the empty-state text inside the field. It is the field's
	// only label — InputRow has no caption slot; wrap it in a FormField when
	// one is needed.
	Placeholder string

	// OnChange fires on every keystroke and is what keeps Value in step with
	// what the user typed. Without it the field is read-only in practice: the
	// next render paints Value back over the keystrokes.
	OnChange func(string)

	// OnSubmit is the commit action: the keyboard's return key (iOS) or IME
	// done action (Android), and the trailing button's tap.
	//
	// When it is nil the field is built without a submit path at all rather
	// than with one wired to a no-op, so neither platform advertises a submit
	// affordance the row would ignore.
	OnSubmit func()

	// Button is the trailing commit button. Its zero value renders nothing;
	// a Button with a Label renders with OnTap defaulted to OnSubmit. Every
	// other Button field — Variant, Emphasis, Disabled, Style, the
	// accessibility pair — works as it does anywhere else.
	Button Button

	// Gap is the horizontal spacing between the field and the button, in
	// points. Zero means the theme's SM step, not zero spacing; see above.
	Gap float64

	// Style is applied to the row after Gap, so a caller can override it —
	// or add the background and padding that make the row read as a docked
	// bar, which the widget itself has no opinion about.
	Style []core.StyleProp
}
```

InputRow is the composer: a single-line text field that fills the row, and an optional trailing button that commits it.

	Row (Gap)
	  ├─ Input   ← FlexGrow(1), value / placeholder / onChange / onSubmit
	  └─ Button  ← only when Button.Label is set; OnTap defaults to OnSubmit

Two call sites spelled this out by hand — chat's message composer and todoapp's entry row — and they were near-identical down to the Gap(8) and the FlexGrow(1). What they also shared was the wiring, which is the part that is easy to get subtly wrong: one commit action reached by three paths (the keyboard's return/done key, the trailing button, and — through onChange — the Go state that both of them read).

#### The three paths, and why OnSubmit drives two of them

A composer is not "a field and a button that happen to sit together": the button \*is\* the field's submit, rendered as a tap target for the case where the keyboard's return key is not obvious or not reachable. Both hand-written sites therefore named the same helper twice, once as core.InputWithSubmit's onSubmit and once as the button's OnTap. Here OnSubmit is the commit action and the button inherits it, so the two cannot drift apart:

	comps.InputRow{
	    Value:       draft.Get(),
	    Placeholder: "What needs doing?",
	    OnChange:    func(v string) { draft.Set(v) },
	    OnSubmit:    addTodo,
	    Button:      comps.Button{Label: "Add"},
	}

Setting Button.OnTap explicitly still wins — a "Send anyway" that skips a validation the keyboard path performs is a real shape — but it has to be said out loud.

#### Gap defaults to the theme's step, unlike Screen's

Screen's Gap treats zero as "don't set one", because the spacing between a screen's sections is the app's decision and a theme's Column base may already carry one. The gap here is the opposite kind of thing: it is the widget's own internal layout — the field and the button must not touch — so InputRow owns it the way FormField owns the spacing between its label and its input. Zero therefore means "the theme's SM step" (8pt in both bundled themes, which is exactly what both hand-written sites had picked).

A caller that genuinely wants no gap says so through Style, which is applied after: Style: \[]core.StyleProp{core.Gap(0)}.

#### The trailing button is optional

A zero Button renders nothing at all — no node, not an empty one — so a search field or a filter box that commits on the return key alone is the same widget with one less field set. Absence is keyed on Label because a button with no visible label is not a button; a glyph button spells its glyph ("✕", "→") in Label and its meaning in AccessibilityLabel.

#### The input is owned, not slotted

Unlike FormField, which takes whatever input it is given, InputRow builds its own — the wiring above \*is\* the widget, and a slot would hand it back to the caller. The consequence is that the field itself takes no per-call styling; a composer that needs to restyle its input has outgrown this and should go back to core.Row + core.InputWithSubmit.

<small>[comps/input_row.go:68](https://github.com/rohanthewiz/grmob/blob/master/comps/input_row.go#L68)</small>

#### func (InputRow) Render

```go
func (r InputRow) Render(ctx *core.Context) *core.Node
```

<small>[comps/input_row.go:107](https://github.com/rohanthewiz/grmob/blob/master/comps/input_row.go#L107)</small>

### type ListRow

```go
type ListRow struct {
	// Leading whose width is text is worth core.FlexShrink(0), and this is
	// the one trap in the slot.
	//
	// The centre column claims the row's slack, so when the row's content
	// overflows — a long title on a phone — the deficit is shared out among
	// the children that can shrink, and a bare core.Text is the most
	// compressible thing in the row. Every host floors that share at the
	// text's min-content width, so a word is never ground down to a glyph:
	// the web and Compose always did, and iOS does since GrMobMinContent,
	// which a simulator run of examples/tutorial forced after it rendered the
	// lesson numbers as 4 / . / 1 / 2.
	//
	// What the floor does NOT promise is that the text stays on one line. A
	// row number has no break opportunity in it, so its min-content is the
	// whole of it; a two-word label's is its longer word, and a row tight
	// enough will wrap it. FlexShrink(0) is the declaration that says the
	// slot's width is not negotiable at all, and a leading slot whose width
	// is meant to be read at a glance wants it on every host.
	//
	// A fixed-size control — a Checkbox, an icon with a Width — is unaffected,
	// which is why this is a note on the field rather than a wrapper around it:
	// ListRow cannot add a style prop to a View a caller handed it, and
	// wrapping every slot in a pinned Box would be two extra nodes per row of
	// every list to fix the case where the caller passes text.
	//
	// Leading is the control at the start of the row: a checkbox, an icon,
	// an avatar. Nil renders nothing and costs no node.
	Leading core.View

	// Title is the row's primary line, Subtitle the quieter second line.
	Title    string
	Subtitle string
	// Content is the escape hatch for the middle: an arbitrary view in the
	// growing slot, taking precedence over Title/Subtitle when set. Same
	// simple-path-plus-slot idiom as Card.Title vs Card.Header.
	Content core.View

	// Trailing is the action or value pinned to the end of the row: a badge,
	// an amount, a delete button, a chevron.
	Trailing core.View

	// OnTap and OnLongPress make the whole row a target. They are wired only
	// when non-nil, so a purely presentational row carries no callback and
	// no gesture recognizer on any platform. Both may be set at once: the
	// renderers wire them as a single recognizer, so a long press never also
	// fires the tap.
	OnTap       func()
	OnLongPress func()

	// Selected drives the row's selected look, and how the state is announced
	// — as a real core.AccessibilitySelected when Selectable is set, and
	// otherwise as a ", selected" suffix on AccessibilityLabel. See "How the
	// state is announced" in the type comment for why there are two answers.
	Selected bool

	// Current says this row is the current item of its set — the destination
	// a navigation list is showing. It is written as core.AccessibilityCurrent,
	// which ARIA defines on every role, so it needs none of the role
	// negotiation Selected goes through, and a row stating it takes no
	// ", selected" suffix. comps.Drawer sets it on its current destination.
	// It changes nothing a sighted user sees; Selected still draws the tint.
	Current core.CurrentKind

	// Selectable says this row is one choice in a listbox: it takes
	// core.RoleOption and states core.AccessibilitySelected for *both* values
	// of Selected, so a reader announces "selected" and "not selected" rather
	// than announcing the chosen row and passing silently over the rest. That
	// is core.SelectedOff doing the job it exists for, one widget over from
	// the tab strip whose argument it was written for.
	//
	// # The container is the caller's to role, and must be
	//
	// An `option` is owned by a `listbox` (see "A structural role owns what is
	// inside it" in core/role.go). This widget renders one row and cannot see
	// what it was put in, so set core.RoleListBox on the container yourself,
	// or leave this field false. An orphan `option` is the "table with no
	// rows" failure one row down.
	//
	//	core.List(
	//	    core.AccessibilityRole(core.RoleListBox),
	//	    ListRow{Title: "Weekly",  Selectable: true, Selected: plan == weekly,
	//	            AccessibilityLabel: "Weekly", OnTap: choose(weekly)},
	//	    ListRow{Title: "Monthly", Selectable: true, Selected: plan == monthly,
	//	            AccessibilityLabel: "Monthly", OnTap: choose(monthly)},
	//	)
	//
	// The same foreign-child rule applies as ever: a listbox holding a
	// "Load more" footer or a section heading is not a listbox.
	//
	// # It wins over NestingLevel, and the depth is lost
	//
	// The two fields ask for different roles and a node has one. `option`
	// takes aria-selected and not aria-level; `listitem` takes aria-level and
	// not aria-selected. So a row setting both describes a container that is a
	// list and a listbox at once, which does not exist, and this field is the
	// half that wins: the state is what the row is being tapped to change,
	// and a depth inside a container that has not claimed to be a list is
	// decoration.
	//
	// ARIA does have a role carrying both — `treeitem` inside a `tree` — and
	// core.Role deliberately does not, because a tree is a third pattern with
	// its own expansion state and keyboard contract and nothing here has one.
	// See the RoleListBox block in core/role.go.
	//
	// # What each target does with it
	//
	// The two web targets write role="option" and aria-selected. Neither
	// native names a listbox or an option, but both announce the *state* on
	// any node — a SwiftUI .isSelected trait, a Compose `selected` property —
	// so the row still reads as chosen on device and it is only the
	// container's word that is missing. Setting this therefore adds on every
	// target and costs nothing on any.
	//
	// # What it buys on the keyboard, and what the caller has to do for it
	//
	// A listbox in ARIA's full pattern takes focus, moves an active option
	// with the arrow keys and reports which one through a roving tabindex.
	// The WASM runtime supplies all of it — one tab stop per list, Up/Down
	// within it, Home and End, and Enter or Space running this row's OnTap —
	// but only for rows inside a container that says it is a listbox. That is
	// the same core.RoleListBox the field above already asks the caller for,
	// so the keyboard arrives with the container's role and not with this
	// flag: a Selectable row in an unroled Box is an `option` with nothing to
	// be an option of, and gets no more keyboard than it did before.
	//
	// A static htmlout export writes no tab stops at all, deliberately — see
	// core.RoleListBox. On both phones none of this was ever missing:
	// VoiceOver and TalkBack navigate a collection by swipe.
	Selectable bool

	// NestingLevel is how deep this row sits in a nested collection — 1 for a
	// top-level item, 2 for one inside it, and on down with no ceiling. It
	// makes the row a `listitem` at that depth; zero leaves it the unroled Box
	// it has always been. Ignored when Selectable is set, which takes the
	// row's one role for `option` — see that field.
	//
	// # What it is for
	//
	// A tree flattened into one list. That is the case ARIA defines aria-level
	// on `listitem` for: the rows are siblings in the markup because a list is
	// a flat run of children — which is also what core.List's virtualization
	// requires — so the depth a reader needs has nowhere else to live. Without
	// it an outline is announced as a flat run of items and every indent is
	// pixels only.
	//
	//	core.List(
	//	    core.AccessibilityRole(core.RoleList),
	//	    ListRow{Title: "Gospels",  NestingLevel: 1},
	//	    ListRow{Title: "Matthew",  NestingLevel: 2, Style: indent(1)},
	//	    ListRow{Title: "Sermon on the Mount", NestingLevel: 3, Style: indent(2)},
	//	)
	//
	// # The container is the caller's to role, and must be
	//
	// A `listitem` is owned by a `list` (see "A structural role owns what is
	// inside it" in core/role.go). This widget renders one row and cannot see
	// what it was put in, so it cannot supply the other half — set RoleList on
	// the container yourself, or leave this field at zero. An orphan
	// `listitem` is the "table with no rows" failure one row down.
	//
	// The corollary is worth stating because it costs something: a row inside
	// a role="list" must not also be a role="button", so a tappable row in an
	// outline announces as an item at a depth and not as a control. That is
	// the same foreign-child rule that kept the selected state off this widget,
	// read from the other side — and it is not a regression, because a ListRow
	// has never carried RoleButton.
	//
	// # What each target does with it
	//
	// The web writes aria-level. Neither native has a nesting-depth property
	// at all, so the role goes out and the depth does not, which is the honest
	// gap several of core's roles already have (see
	// core.Style's AccessibilityRole table for which).
	NestingLevel int

	// Style is applied to the row container after ListRow's own defaults
	// (which sit on top of the theme's Row base), so every default here —
	// gap, vertical centering, the theme's row padding — is overridable.
	Style []core.StyleProp
	// SelectedStyle is applied on top when Selected, after Style, so
	// selection wins over the base look. When nil, a theme default is used:
	// a Surface background tint.
	SelectedStyle []core.StyleProp

	// AccessibilityLabel names the whole row for screen readers.
	// AccessibilityHint describes what tapping does.
	//
	// A Selected row with no Selectable gets ", selected" appended, because
	// the name is then the only place the state can be said; a Selectable row
	// leaves the name alone and states the selection properly. See "How the
	// state is announced" in the type comment.
	//
	// No label is synthesized from Title: a row is a compound control whose
	// slots (a badge's amount, a trailing control's own name) carry meaning
	// the widget cannot see, and labelling the container overrides how those
	// children are announced. Naming the row is therefore the caller's call,
	// exactly as it is for Chip.
	AccessibilityLabel string
	AccessibilityHint  string
}
```

ListRow is the leading-control / flexible-title / trailing-action shape that every list in the examples hand-rolls: a checkbox and a task with a delete button, an avatar and a name with a chevron, a label and an amount.

#### Why the widget exists

The shape was written five times across the examples and the instances did not agree on how the trailing slot gets pinned to the trailing edge. Some used Justify(JustifyBetween) on the row; others used FlexGrow(1) on the middle Text. The two are not equivalent:

	JustifyBetween  distributes slack *between every pair* of children, so a
	                row with no trailing slot pushes leading and title apart.
	FlexGrow(1)     gives all the slack to one child, so leading and trailing
	                stay hard against the row's edges in every configuration.

ListRow settles it on FlexGrow: the middle column is the row's spine and it always grows. That is also why the middle column is rendered even when Title, Subtitle and Content are all empty — unlike Card, which omits empty regions. Here the middle is structure, not content: making it conditional would make the pinning conditional too, which is precisely the inconsistency this widget exists to remove.

	┌ Row ─────────────────────────────────────────────────────────┐
	│ [Leading]  ┌ Column FlexGrow(1) ────────────┐    [Trailing]  │
	│            │ Title                          │                │
	│            │ Subtitle                       │                │
	│            └────────────────────────────────┘                │
	└──────────────────────────────────────────────────────────────┘
	            └──────── takes all the slack ────┘

#### Selection

Selection is controlled by the caller, as with Chip: ListRow holds no state, it renders Selected and reports taps. Selected rows take the theme's Surface as a background tint — the palette's only muted \*fill\*, and still the right one now that Border exists, since Border is a stroke role; there is no dedicated Selected entry. How the state reaches a screen reader is the next section's subject and depends on Selectable.

#### How the state is announced, and why it took a role to do it

The state is scoped by role on both web targets, because ARIA scopes the attributes it becomes: a state on an unroled element is dropped by screen readers exactly as an accessible name on one is. The \*name\* half of that has since been closed for every widget at once — the two web exporters supply core.RoleGroup to a named container that has no role, so an unroled row's AccessibilityLabel is now announced on all four targets rather than two. The state half could not be closed the same way, because there is no role that carries a selection and fits any container (see core.Style.AccessibilitySelected). A ListRow is a Box, so announcing a selection properly means giving the row a role, and for three versions of this widget every candidate was wrong:

	RoleButton    true only for a tappable row, and a role="button" child
	              makes the row a *foreign child* of any role="list" it sits
	              in — the structural rule in core/role.go, which says such a
	              container may then take no list role at all. One widget's
	              announcement would cost the enclosing list its shape.
	RoleListItem  the honest description of a row, and ARIA defines neither
	              state attribute for it. `aria-selected` is scoped to
	              gridcell, option, row, tab, columnheader and rowheader, and
	              a list item is none of them.

So the row wrote ", selected" into its own accessible name instead — the true thing said in the weaker of the two places, announced once, inside a string that is meant to be stable.

core.RoleOption and core.RoleListBox are the door that was left, and Selectable is how a caller walks through it. A row that takes the option role carries a real core.AccessibilitySelected, every renderer announces it as a state rather than as part of a name, and the suffix does not appear.

A row that is \*not\* Selectable is unchanged: it still appends the suffix when it has both a label and a selection, because it still has no role that could carry the state, and saying the true thing weakly beats not saying it. The suffix does at least reach a reader now — the group role the exporters supply is what makes the name it rides on audible on the web at all, which for the three sessions before that role existed it was not.

#### Two roles, one row, and the caller picks

NestingLevel and Selectable both give the row a role and the two roles are exclusive — see Selectable for the precedence, which is a fact about the roles rather than about this widget's willingness.

Both are opt-in, and that is the ownership rule rather than caution: a \`listitem\` with no \`list\` around it, or an \`option\` with no \`listbox\`, is a role naming a structure that is not there, which core/role.go calls worse than no role at all. A row cannot see its container, so it cannot make that true by itself — the caller roles the enclosing collection and marks each row to match, and a row that is asked for neither is exactly the unroled Box it has always been.

<small>[comps/list_row.go:97](https://github.com/rohanthewiz/grmob/blob/master/comps/list_row.go#L97)</small>

#### func (ListRow) Render

```go
func (r ListRow) Render(ctx *core.Context) *core.Node
```

<small>[comps/list_row.go:309](https://github.com/rohanthewiz/grmob/blob/master/comps/list_row.go#L309)</small>

### type LoadMore

```go
type LoadMore struct {
	HasMore bool
	Loading bool
	Err     error

	// OnLoadMore fetches the next page. OnRetry re-runs a failed fetch; when
	// nil, Retry falls back to OnLoadMore, which is the right answer for a
	// pager whose load call is idempotent about the offset.
	OnLoadMore func()
	OnRetry    func()

	// Label, LoadingLabel and RetryLabel override the copy. ErrorText
	// replaces err.Error() — for an app that maps transport errors to
	// something a person should read.
	Label        string
	LoadingLabel string
	RetryLabel   string
	ErrorText    string

	// Style is applied to the tail's container after its defaults.
	Style []core.StyleProp
}
```

LoadMore is the append-style tail of an offset-paged list: the "Load more" button, the "Loading…" note while a page is in flight, and the retry strip when a page failed. It exists because every paged screen in an app hand-rolls exactly this state machine at the bottom of its list.

	HasMore  Loading  Err   renders
	false    false    nil   nothing (the list is complete)
	true     false    nil   [Load more]
	*        true     *     Loading…
	*        false    set   message  [Retry]

Err wins over HasMore because a failed fetch says nothing about whether more rows exist; Loading wins over Err because a retry in flight has superseded the failure it is retrying.

<small>[comps/paging.go:108](https://github.com/rohanthewiz/grmob/blob/master/comps/paging.go#L108)</small>

#### func (LoadMore) Render

```go
func (l LoadMore) Render(ctx *core.Context) *core.Node
```

<small>[comps/paging.go:131](https://github.com/rohanthewiz/grmob/blob/master/comps/paging.go#L131)</small>

### type Pagination

```go
type Pagination struct {
	// Page is the 0-based current page.
	Page int
	// PageSize is rows per page; only read by a collection doing its own
	// slicing (see the type comment).
	PageSize int
	// PageCount is the total number of pages, 0 when unknown.
	PageCount int
	OnChange  func(page int)

	// PrevLabel and NextLabel override the button text. Empty takes the
	// defaults below.
	PrevLabel string
	NextLabel string

	// Style is applied to the footer row after its defaults.
	Style []core.StyleProp
}
```

Pagination is the numbered-page footer: "‹ Prev  Page 2 of 7  Next ›".

It serves two paging models with one struct, told apart by PageCount:

  - PageCount > 0: the caller owns the pages. Whatever rows it hands the collection are the current page, and OnChange asks it to fetch another. This is the server-side shape.
  - PageCount == 0 and PageSize > 0: the collection owns the pages. It slices the full row set itself and derives PageCount from len(rows). This is the client-side shape; DataTable implements it, GroupedList does not (a grouped feed pages by "Load more", not by number).
  - PageCount == 0 and PageSize == 0: open-ended. The label shows only the current page and Next is never disabled, for a caller that learns there is no next page only by asking.

Selection is controlled, as everywhere in this package: Page is read from the caller's state and OnChange writes it back.

<small>[comps/paging.go:22](https://github.com/rohanthewiz/grmob/blob/master/comps/paging.go#L22)</small>

#### func (Pagination) Render

```go
func (p Pagination) Render(ctx *core.Context) *core.Node
```

<small>[comps/paging.go:41](https://github.com/rohanthewiz/grmob/blob/master/comps/paging.go#L41)</small>

### type Sort

```go
type Sort struct {
	Column int
	Desc   bool
}
```

Sort names the active sort column and direction. DataTable.Sort is a pointer so that "no sort" is nil rather than an ambiguous column 0.

<small>[comps/data_table.go:65](https://github.com/rohanthewiz/grmob/blob/master/comps/data_table.go#L65)</small>

### type SwitchRow

```go
type SwitchRow struct {
	// Title is the setting's name. It is also the control's accessible label.
	Title string

	// Subtitle is the secondary line under the title, and the control's
	// accessibility hint.
	Subtitle string

	// Leading is an optional icon or avatar before the text, as in ListRow.
	Leading core.View

	// On is the caller's current value.
	On bool

	// OnToggle receives the new value. It is a setter — apply the value; do
	// not invert your own state — because on the web the row and the switch
	// can both report one tap (see the type doc).
	OnToggle func(on bool)

	// Disabled disables both the switch and the row: no taps reach OnToggle,
	// and the control is announced as disabled.
	Disabled bool

	// Style is passed to the underlying ListRow and so beats its defaults.
	Style []core.StyleProp
}
```

SwitchRow is the settings-screen row: a title, an optional subtitle, a switch on the trailing edge, and the whole row tappable rather than only the switch.

	comps.SwitchRow{Title: "Notifications", Subtitle: "Push and email",
	    On: notify.Get(), OnToggle: notify.Set}

It is ListRow with the trailing slot fixed to core.Switch and OnTap wired to the same setter. CheckboxRow, below, is the same row with a core.Checkbox: the switch for a setting that takes effect on the tap, the checkbox for a value a form collects (see core.Switch for why the two are different controls).

#### One tap, one state change

The row and the control are both tappable, and the targets disagree about what a tap on the control does:

	target    tap on the control                      handlers that fire
	───────   ─────────────────────────────────────   ──────────────────────
	Compose   Switch's toggleable consumes the press   control only
	SwiftUI   Toggle's gesture wins over the row's     control only
	web       native toggle, then the click BUBBLES    row (click), then
	          to the row <div>, then `change` fires    control (change)

On the web one tap therefore reaches Go twice. Go cannot tell which element the click landed on, so the row cannot skip its handler for taps "on the control". Two other ways out were considered and rejected:

  - Render the control Disabled and let the row be the only handler. That is what the web runtime would need (a disabled control gets pointer-events:none, so the click falls through), but Disabled is the platform's disabled state, not a hit-test flag: Material, SwiftUI and the browser all draw the control greyed, and every screen reader says "dimmed". A settings screen of disabled-looking switches is broken.
  - Give the control a no-op handler. On both natives the control consumes its own press, so tapping the switch itself would do nothing.

What makes the double dispatch harmless is that both handlers \*set\* a value rather than flip one, and both go through one guard:

	set(v) = if v != On { OnToggle(v) }

The row calls set(!On) with the On it was rendered with. The control calls set(v) with the value the platform reports. On the web the row's click is dispatched first, Go re-renders, and the registry now holds closures over the new On (callback IDs are positional per pass and a pass overwrites each handler; see core's callbackRegistry). The \`change\` that follows reports the same v, which now equals On, so the guard drops it. If the two events ever arrived within one pass, both would compute the same target value and a setter is idempotent. Either way OnToggle sees exactly one change per tap.

OnToggle is therefore documented as a setter: it receives the new value, and a caller should apply it rather than invert its own state.

#### Accessibility

The control carries AccessibilityLabel(Title) and, when present, AccessibilityHint(Subtitle), so a reader landing on it hears "Notifications, switch, on" rather than an unnamed switch; a platform control has no label of its own (core.Switch leaves it to the caller). The row takes no role: ListRow's rule is that a container is not relabelled, and the switch or checkbox is the control a reader is looking for.

#### Theme roles read

Everything ListRow reads (Spacing.SM gap, Typography.Body and Caption for the text), plus Components.CheckBox through the control.

<small>[comps/settings_row.go:73](https://github.com/rohanthewiz/grmob/blob/master/comps/settings_row.go#L73)</small>

#### func (SwitchRow) Render

```go
func (r SwitchRow) Render(ctx *core.Context) *core.Node
```

Render builds the ListRow described in the type doc.

<small>[comps/settings_row.go:101](https://github.com/rohanthewiz/grmob/blob/master/comps/settings_row.go#L101)</small>

### type Timeline

```go
type Timeline struct {
	// Events are drawn top to bottom, oldest or newest first as the caller
	// orders them.
	Events []TimelineEvent

	// Label is the list's accessible name.
	Label string

	// Style is applied to the list column after the widget's own props.
	Style []core.StyleProp
}
```

Timeline is a vertical list of events joined by a line down the leading edge, a dot per event: order tracking, an activity feed, a changelog.

	comps.Timeline{
	    Label: "Order history",
	    Events: []comps.TimelineEvent{
	        {Time: "09:12", Title: "Order placed"},
	        {Time: "11:40", Title: "Packed", Subtitle: "Warehouse 3"},
	        {Time: "14:05", Title: "Out for delivery", Variant: comps.VariantSuccess},
	    },
	}

#### The line is drawn per row, and that is the hard part

No renderer draws a line that spans siblings, so each row draws its own piece of it. The row is a Row with AlignItems stretch, so its leading rail column is exactly as tall as the event's text, and the rail is three parts:

	┌ Row  AlignItems(stretch)  role=listitem ──────────────────┐
	│ ┌ rail Column ┐ ┌ body Column FlexGrow(1) ───────────────┐ │
	│ │  │ top      │ │ 11:40                   (Time)         │ │
	│ │  ●  dot     │ │ Packed                  (Title)        │ │
	│ │  │          │ │ Warehouse 3             (Subtitle)     │ │
	│ │  │ bottom   │ │ [Content]                              │ │
	│ │  │ FlexGrow │ │                    PaddingBottom(MD)   │ │
	│ └─────────────┘ └────────────────────────────────────────┘ │
	└────────────────────────────────────────────────────────────┘

The top segment is a fixed height that puts the dot's centre on the centre of the body's first line. The bottom segment grows to the row's full height, and the row's spacing below an event is the body's bottom padding, not a gap between rows, so the bottom segment runs through it and meets the next row's top segment with no break. The first row's top segment and the last row's bottom segment keep their size and paint nothing, so every dot sits at the same offset.

#### Why not ListRow

The plan sketched this as ListRow with a stretched leading slot. ListRow centres its slots by default (overridable) and, more to the point, carries the theme's Row padding above and below. That padding sits outside the rail, so the line would break between every pair of rows. The rows here are plain Rows with Padding(0), and the spacing moves inside the body where the rail can run through it.

Cross-axis stretch on a Row is honoured on every target: the web by align-items, Compose through stretchRowHeight's intrinsic measurement and fillMaxHeight, and iOS through the flex layout's stretch. The native half comes from reading Renderer.kt and GrMobFlex.swift, not from a device run.

#### Accessibility

The Timeline is RoleList and each event RoleListItem, so a reader announces "list, 3 items" and moves event by event; Label names the list. The rail is drawing and is hidden from assistive technology. A list item holds no button role, so an event is read, not operated; put a control in Content if an event needs one.

#### Theme roles read

	Line       Colors.BorderColor()
	Dot        Variant.Color — Colors.Primary unless the event sets a Variant
	Time       Typography.Caption
	Title      Typography.Body, bold
	Subtitle   Typography.Caption
	Spacing    Spacing.SM between rail and body, Spacing.MD below an event

<small>[comps/timeline.go:76](https://github.com/rohanthewiz/grmob/blob/master/comps/timeline.go#L76)</small>

#### func (Timeline) Render

```go
func (tl Timeline) Render(ctx *core.Context) *core.Node
```

Render builds Column(list) > Row(listitem)... as drawn in the type doc.

<small>[comps/timeline.go:114](https://github.com/rohanthewiz/grmob/blob/master/comps/timeline.go#L114)</small>

### type TimelineEvent

```go
type TimelineEvent struct {
	// Time is an optional caption above the title: "09:12", "Yesterday".
	Time string

	// Title is the event's primary line.
	Title string

	// Subtitle is the quieter line under the title.
	Subtitle string

	// Content is an optional view under the text: a thumbnail, a quote, a
	// button.
	Content core.View

	// Variant colours the dot. The zero value is Primary.
	Variant Variant
}
```

TimelineEvent is one row of a Timeline.

<small>[comps/timeline.go:89](https://github.com/rohanthewiz/grmob/blob/master/comps/timeline.go#L89)</small>

