package components

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
//	    Collapse: components.Collapse{
//	        IsCollapsed: func(g components.Group) bool { return shut.Get()[g.Key] },
//	        OnToggle: func(g components.Group) {
//	            next := maps.Clone(shut.Get())
//	            next[g.Key] = !next[g.Key]
//	            shut.Set(next)
//	        },
//	    },
//	}
//
// components.Accordion is the other answer to the same question and stays the
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
	return runs
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
	// Android. See components.disclosure, which is where the pairing and the
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
	// The cost is that the badge and the band's own padding are not part of
	// the tap target: the button fills the space between the insets and stops
	// where the count begins. That is the ordinary shape of a header row with
	// a trailing badge, and the alternative — folding the count into the
	// button's accessible name — would be assembling an English phrase in the
	// renderer, which is the move Chip's ", selected" suffix was deleted for.
	Expanded bool
	OnToggle func()

	// Style is applied to the band after its defaults.
	Style []core.StyleProp
}

func (h GroupHeader) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(h.Style)+8)
	// Padding(0) first to shed the theme Row's own padding, then the band's
	// tighter recipe: a full row of horizontal breathing room and the
	// finest vertical step, so the band reads as a divider, not a row.
	items = append(items,
		core.Padding(0),
		core.PaddingHorizontal(t.Spacing.MD),
		core.PaddingVertical(t.Spacing.XS),
		core.BackgroundColor(t.Colors.Surface),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
	)
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
		items = append(items, core.Box(
			core.FlexGrow(1),
			core.Text(h.Group.Label, label...),
		))
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
			ControlStyle: []core.StyleProp{
				core.Gap(float64(t.Spacing.SM)),
				core.AlignItemsProp(core.AlignItemsCenter),
			},
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
