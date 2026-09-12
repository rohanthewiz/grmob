package comps

import "github.com/rohanthewiz/grmob/core"

// Group identifies one run of rows in a grouped collection. GroupBy callbacks
// return Key and Label; the grouping engine fills Count. Key is what decides
// group identity and becomes the header's reconciler key, so it should be
// stable and comparable ("2026-01"), while Label is what people read
// ("January 2026").
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

// Collapse is caller-owned collapse state for a banded collection: which
// groups are shut, and what to do when one is pressed.
//
// # Why the caller holds it
//
// GroupedList calls no hook, which is what lets it be rendered conditionally
// — inside a core.IfElse against a pager's loaded flag, say — without
// disturbing the caller's hook cursor. Owning collapse state would end that,
// and the widget is the wrong place for it anyway: which months are shut is
// screen state, it usually wants to survive a pager reload, and a screen that
// wants "collapse all" has no way to reach inside a widget's NewState.
//
// So the screen keeps a set and answers two questions:
//
//	shut := core.NewState(ctx, map[string]bool{})
//	GroupedList[Sermon]{
//	    GroupBy: byMonth,
//	    Collapse: comps.Collapse{
//	        IsCollapsed: func(g comps.Group) bool { return shut.Get()[g.Key] },
//	        OnToggle: func(g comps.Group) {
//	            next := maps.Clone(shut.Get())
//	            next[g.Key] = !next[g.Key]
//	            shut.Set(next)
//	        },
//	    },
//	}
//
// comps.Accordion is the other answer to the same question and stays the
// right one for a single section: it owns its state, and the hook obligations
// that come with it are documented on the widget. A list of twenty bands is
// where owning the state stops being a convenience — twenty independent
// NewStates that a reorder cannot move, and no way to shut them all.
//
// # One type, two functions
//
// They are two halves of one fact and are useless apart, which is the same
// argument core.ValueRange makes for its three numbers. As two fields on
// GroupedList a caller could supply either alone: IsCollapsed without OnToggle
// is a list with rows nobody can bring back, and OnToggle without IsCollapsed
// is a control that announces a state it does not have. The zero value is
// "nothing collapses", which is what every list that has never heard of this
// keeps doing.
type Collapse struct {
	// IsCollapsed reports whether a group's run is hidden. Nil means no.
	IsCollapsed func(Group) bool
	// OnToggle is called with the group whose band was pressed. Nil leaves
	// the bands as plain headings — see GroupHeader.Expanded.
	OnToggle func(Group)
}

// active reports whether the bands should be built as disclosures. Keyed on
// OnToggle rather than on IsCollapsed: a band with a handler and no predicate
// is a disclosure that is always open, which is odd but coherent, while a
// band with a predicate and no handler is a control nobody can operate.
func (c Collapse) active() bool { return c.OnToggle != nil }

// collapsed reports whether the predicate says this group is shut. Only
// meaningful together with active; use hides for the question the row loop
// asks.
func (c Collapse) collapsed(g Group) bool {
	return c.IsCollapsed != nil && c.IsCollapsed(g)
}

// hides reports whether this group's run should be withheld — the predicate
// says shut *and* there is a control that can bring it back.
//
// Both halves, in one place, because the two callers would otherwise have to
// agree: a predicate with no handler would hide rows behind a band with no
// control, which is a feed that silently loses items and offers no way to
// find them. The band's own state is derived from the same pair, so a run
// that is hidden always has a control announcing it as collapsed.
func (c Collapse) hides(g Group) bool { return c.active() && c.collapsed(g) }

// CollapseBand is the disclosure control the default band builds, on its own: a
// heading wrapping a button that carries aria-expanded and toggles one group's
// run. No Surface, no padding, no count badge — the chrome is the caller's.
//
// # The gap it closes
//
// GroupedList.Collapse reaches past a Header override for the row emission, so
// an override's run still hides, and it stops at the override for the *control*,
// because a band the widget also built would be a second control for the same
// run. That division is right and it left the override author holding three
// things at once: a button, an aria-expanded that has to be stated on every pass
// open or shut, and a heading wrapper whose nesting order — heading around
// button, named explicitly so the chevron never reaches it — is four paragraphs
// of argument in comps.disclosure, which is unexported.
//
// GroupHeader is still the answer when the whole default band will do; it takes
// Expanded and OnToggle and builds all of it. This is the answer when it will
// not:
//
//	Header: func(g comps.Group) core.View {
//	    return core.Row(
//	        core.PaddingHorizontal(16),
//	        comps.CollapseBand{Collapse: shut, Group: g},
//	        Avatar{Name: leader[g.Key]},
//	        Badge{Text: strconv.Itoa(g.Count)},
//	    )
//	},
//
// The Collapse passed here is the caller's own — the same value handed to
// GroupedList — which is what keeps the control and the row hiding answering to
// one state. An override that built its control from a second Collapse would
// have a chevron pointing one way and a run obeying the other.
//
// # An inactive Collapse builds a heading and no control
//
// The zero Collapse — and one with IsCollapsed and no OnToggle — produces the
// label in a plain heading, exactly as GroupHeader's own non-disclosure branch
// does. Not a button with a dead handler: an expansion stated with nothing to
// toggle it is announced on both web targets, is silently nothing on Android,
// and is what core.AuditTree reports as ConcernInertDisclosure. Building one
// here would be building the thing the audit exists to find.
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
}

func (b CollapseBand) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	// The same three declarations GroupHeader gives its label, so a control
	// lifted out of the default band into a custom row does not change weight
	// or ink on the way.
	label := []core.StyleProp{
		core.UseStyle(t.Typography.Caption),
		core.FontWeight(core.Bold),
		core.TextColor(t.Colors.TextSecondary),
	}

	content := b.Content
	if len(content) == 0 {
		content = []core.View{core.Text(b.Group.Label, label...)}
	}

	if !b.Collapse.active() {
		// The heading props ride the words here rather than a wrapper, which
		// is the division GroupHeader draws too: there is no button for the
		// tier to sit outside of.
		plain := append(label, headingProps(b.HeadingLevel, headingLevelSection)...)
		box := make([]core.PropsAndChildren, 0, len(b.Style)+len(b.ControlStyle)+len(content)+1)
		for _, sp := range b.Style {
			box = append(box, sp)
		}
		// After Style, so a caller who set both gets the control's chrome
		// last — the same order the disclosure branch below applies them in.
		for _, sp := range b.ControlStyle {
			box = append(box, sp)
		}
		if len(b.Content) == 0 {
			box = append(box, core.Text(b.Group.Label, plain...))
		} else {
			// A caller who supplied their own content owns its typing; the
			// tier still has to land somewhere, so it goes on the wrapper.
			for _, sp := range headingProps(b.HeadingLevel, headingLevelSection) {
				box = append(box, sp)
			}
			box = append(box, core.AccessibilityLabel(b.Group.Label))
			for _, child := range content {
				box = append(box, child)
			}
		}
		return core.Box(box...).Render(ctx)
	}

	return disclosure{
		Label:        b.Group.Label,
		Hint:         "Expands or collapses the group",
		Expanded:     !b.Collapse.collapsed(b.Group),
		OnToggle:     func() { b.Collapse.OnToggle(b.Group) },
		Heading:      true,
		Level:        b.HeadingLevel,
		OwnLevel:     headingLevelSection,
		HeadingStyle: b.Style,
		ChevronStyle: []core.StyleProp{core.UseStyle(t.Typography.Caption)},
		ControlStyle: append([]core.StyleProp{
			core.Gap(float64(t.Spacing.SM)),
			core.AlignItemsProp(core.AlignItemsCenter),
		}, b.ControlStyle...),
		Control: content,
	}.view().Render(ctx)
}

// groupRun is one contiguous slice of items sharing a group key: items
// [Start, End) belong to Group.
type groupRun[T any] struct {
	Group      Group
	Start, End int
}

// groupRuns partitions items into runs of equal group key, in display order.
//
// # Why run-length and not a map
//
// Grouping is a property of the *order the caller hands us*, not of the data.
// Two things follow:
//
//   - A sorted input stays in one pass and allocates no buckets, and the
//     order of headers is exactly the order of first appearance, which is the
//     order the caller already chose (a date-desc sermon feed gets date-desc
//     month headers without anyone re-sorting).
//   - An append-only pager (offset-based "Load more") can only ever extend
//     the *last* run or start new runs after it. Earlier headers never move,
//     so loading the next page never reshuffles what is already on screen.
//
// The cost is that an unsorted input yields repeated headers for the same
// key. That is the honest rendering of the caller's order: a table sorted by
// teacher and grouped by month legitimately shows a month more than once.
// Callers who want one header per key sort by the group key first.
//
//	items:  Jan Jan Feb Feb Feb Jan
//	runs:   [Jan 0..2] [Feb 2..5] [Jan 5..6]
func groupRuns[T any](items []T, by func(T) Group) []groupRun[T] {
	if by == nil || len(items) == 0 {
		return nil
	}
	var runs []groupRun[T]
	for i, item := range items {
		g := by(item)
		if n := len(runs); n > 0 && runs[n-1].Group.Key == g.Key {
			runs[n-1].End = i + 1
			runs[n-1].Group.Count++
			continue
		}
		g.Count = 1
		runs = append(runs, groupRun[T]{Group: g, Start: i, End: i + 1})
	}
	// The last run is the one an append pager extends, which is a fact about
	// the walk rather than about any one item — so it is stamped here, where
	// the walk ends, and not at the band, where it would be an index
	// comparison repeated at every call site that grouped anything.
	runs[len(runs)-1].Group.Trailing = true
	return runs
}

// trailingRun returns the last group in items — the run an append-style pager
// extends — or false when there is no grouping or nothing to group.
//
// # Why it walks backwards instead of calling groupRuns
//
// The caller wants one run out of a list that can be thousands of rows long,
// and it wants it on every render. groupRuns is a full pass that allocates a
// slice of every run in the list; this is a walk over the last run alone,
// which for a month-banded feed is a handful of items regardless of how many
// pages have loaded.
//
// The subtlety is that it must agree with groupRuns about what the last run's
// Group *is*, or the widget would ask its Collapse predicate about a group the
// band never showed. groupRuns takes the Group from the run's first item and
// counts the rest, so this does the same after finding the boundary: a GroupBy
// that returns the same Key with a different Label for two items — a
// "March" and a "March 2026" — is answered here exactly as the band answers it.
func trailingRun[T any](items []T, by func(T) Group) (Group, bool) {
	if by == nil || len(items) == 0 {
		return Group{}, false
	}
	key := by(items[len(items)-1]).Key
	start := len(items) - 1
	for start > 0 && by(items[start-1]).Key == key {
		start--
	}
	g := by(items[start])
	g.Count = len(items) - start
	// The same stamp groupRuns puts on its last run, so the Group this hands
	// to a Collapse predicate is shaped exactly like the one the band will.
	// Group.AutoLoadWithheld is deliberately *not* set: it is the answer this
	// walk is being run to compute, and a probe that pre-announced its own
	// result would be asking the predicate a leading question.
	g.Trailing = true
	return g, true
}

// groupHeaderKey is the reconciler key for a group's header row. Prefixed so
// it cannot collide with a row key that happens to equal a group key.
func groupHeaderKey(g Group) string {
	return "group:" + g.Key
}

// GroupHeader is the default band rendered above each group in GroupedList
// and DataTable: the label in bold caption ink on the theme's Surface, with
// the row count as a badge pinned to the trailing edge. It is exported so a
// caller can render it with a different Count or Label from inside a Header
// override, or reuse it in a hand-built list.
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

// bandInsets is the band's chrome, expressed as padding on the control rather
// than on the row that holds it.
//
// # Why the insets are not on the band Row
//
// They used to be, and the shape that produced was a header whose tap target
// was a strip in the middle of it:
//
//	 Row ────────────────────────────────────────────────
//	│        ┌───────────────────────┐          ┌───┐    │   before
//	│  16px  │ ▸ January 2026        │   8px    │ 3 │ 16 │   the button is the
//	│        └───────────────────────┘          └───┘    │   box; the insets
//	 ────────────────────────────────────────────────────    are dead space
//
//	 Row ────────────────────────────────────────────────
//	│┌──────────────────────────────────────┐  ┌───┐     │   after
//	││  ▸ January 2026                      │  │ 3 │ 16px│   the button owns
//	│└──────────────────────────────────────┘  └───┘     │   the insets and
//	 ────────────────────────────────────────────────────    the gap
//
// Nothing moves. The pixels the reader sees are identical in both — the same
// 16 before the chevron, the same 8 before the badge, the same 4 above and
// below — because padding on a stretched child fills exactly the space the
// same padding on its parent held. What changes is which node it belongs to,
// and therefore what a finger landing on it hits.
//
// The band Row keeps its Padding(0) (to shed the theme Row's own recipe), its
// fill and its cross-axis centering, which is what matters for the two things
// that must not move: core.StickyHeader goes on this node because the list
// sees it as its child, and the fill has to span the full width or the pinned
// band would show the rows scrolling through its margins.
//
// # Where the tap target still stops
//
// At the badge, and that is deliberate rather than residual. The count is
// content — it is announced separately, which is the whole reason it sits
// outside the button (see GroupHeader.Expanded) — so a press on the number is
// a press on a thing, not on the control beside it. The gap before it belongs
// to the button, because the gap is chrome.
//
// trailing is therefore the caller's answer to "what comes next": the gap
// when a badge follows, the band's own trailing inset when nothing does.
// trailing is 0 for "whatever the leading inset is", which is what a band
// with nothing after its control wants.
//
// # The picture used to assume the disclosure branch, and now it is measured
//
// On the plain branch the node carrying these insets *is* the Row's growing
// child, so "the button owns the insets" is a main-axis fact and the flex
// arithmetic ios/verify runs settles it. On the disclosure branch it is not:
// the growing child is a heading wrapper with no chrome of its own, and the
// button sits inside it with no weight at all (see disclosure.view). Whether
// the target reaches the wrapper's edges is a CROSS-axis question — a
// non-growing child of a vertical container is stretched to it, which is the
// fallback core.Box and core.SafeArea both document — and a main-axis
// distributor has no answer to it.
//
// wasm/verify/browser.mjs now mounts real bands of both branches in every
// bundled theme and measures the rects. The button does fill the wrapper, so
// the picture above is right about the disclosure too, and it is right because
// of a cross-axis default rather than because of anything this widget declares.
// A wrapper that stopped stretching its child would leave the tap target at the
// label's own width with the rest of the band dead, which is the shape the move
// was made to end.
func bandInsets(t *core.Theme, trailing int) []core.StyleProp {
	insets := []core.StyleProp{
		core.PaddingHorizontal(t.Spacing.MD),
		core.PaddingVertical(t.Spacing.XS),
	}
	if trailing > 0 {
		// A side beside the shorthand rather than two explicit sides. The
		// horizontal step is the band's own recipe and the trailing value is
		// an override for what follows it, and spelling it this way keeps the
		// two separable — core/padding_sides.go settles the pair the same way
		// for every renderer, and a caller clearing one side through
		// ControlStyle lands on the same machinery.
		insets = append(insets, core.PaddingRight(trailing))
	}
	return insets
}

func (h GroupHeader) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(h.Style)+8)
	// Padding(0) to shed the theme Row's own padding, and Gap(0) because the
	// one gap this band has is the button's trailing inset now. The band's
	// tighter recipe — a full row of horizontal breathing room and the finest
	// vertical step, so the band reads as a divider rather than a row — is on
	// the control instead; see bandInsets.
	items = append(items,
		core.Padding(0),
		core.Gap(0),
		core.BackgroundColor(t.Colors.Surface),
		core.AlignItemsProp(core.AlignItemsCenter),
	)
	// The one inset that cannot move onto the control: the badge's own
	// trailing breathing room, which is past the button's right edge. With no
	// badge there is nothing on that side, so the inset is the button's and
	// arrives through trailing below.
	trailing := 0
	if !h.HideCount {
		trailing = t.Spacing.SM
		items = append(items, core.PaddingRight(t.Spacing.MD))
	}
	for _, sp := range h.Style {
		items = append(items, sp)
	}

	// A band titles a run of rows, which is what a heading is — and on a long
	// banded feed it is the thing a reader navigating by heading wants to move
	// between, since the screen's own title scrolled away several pages ago.
	//
	// On the label rather than on the band: the band also holds the count
	// badge, and a heading whose name is "March 12" reads worse than one whose
	// name is "March". The count is still announced, as the separate thing it
	// is.
	//
	// Level 2 by default — a band is a section *of* the screen whose name the
	// AppBar's title carries at level 1, which is the outline
	// core.Style.AccessibilityHeadingLevel exists for — and whatever
	// HeadingLevel says otherwise.
	label := []core.StyleProp{
		core.UseStyle(t.Typography.Caption),
		core.FontWeight(core.Bold),
		core.TextColor(t.Colors.TextSecondary),
	}

	// The label grows so the badge sits hard against the trailing edge —
	// the same FlexGrow-not-JustifyBetween pinning ListRow settled on.
	//
	// Which node grows differs between the two shapes and the reason is the
	// same in both: it has to be the band Row's own direct child, or the flex
	// line has nothing to distribute. Plain, that is the Box around the label;
	// as a disclosure, it is the heading wrapper the shape builds, and the
	// growth is passed in as HeadingStyle for exactly that reason.
	if h.OnToggle == nil {
		label = append(label, headingProps(h.HeadingLevel, headingLevelSection)...)
		// The same insets the disclosure branch puts on its button. There is
		// no tap target here to enlarge — the point is that the two branches
		// are the same band geometrically, so a caller who adds OnToggle to a
		// GroupHeader gets a control and not a relayout.
		box := asProps(append(append(bandInsets(t, trailing), core.FlexGrow(1)),
			h.ControlStyle...))
		items = append(items, core.Box(append(box,
			core.Text(h.Group.Label, label...),
		)...))
	} else {
		// No heading props on the words here: the tier rides the wrapper, and
		// a heading inside a button is written into the document and pruned
		// out of the accessibility tree by every browser. Same division
		// Accordion draws, and for the same reason.
		items = append(items, disclosure{
			Label:        h.Group.Label,
			Hint:         "Expands or collapses the group",
			Expanded:     h.Expanded,
			OnToggle:     h.OnToggle,
			Heading:      true,
			Level:        h.HeadingLevel,
			OwnLevel:     headingLevelSection,
			HeadingStyle: []core.StyleProp{core.FlexGrow(1)},
			ChevronStyle: []core.StyleProp{core.UseStyle(t.Typography.Caption)},
			// The band's insets, on the button. See bandInsets for the
			// before/after and for why the badge is where the target stops.
			ControlStyle: append(append(bandInsets(t, trailing),
				core.Gap(float64(t.Spacing.SM)),
				core.AlignItemsProp(core.AlignItemsCenter),
			), h.ControlStyle...),
			Control: []core.View{core.Text(h.Group.Label, label...)},
		}.view())
	}
	if !h.HideCount {
		items = append(items, Badge{Text: itoa(h.Group.Count)})
	}
	return core.Row(items...).Render(ctx)
}

// itoa is strconv.Itoa without the import in every widget file.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
