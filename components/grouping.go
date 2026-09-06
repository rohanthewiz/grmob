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
	label = append(label, headingProps(h.HeadingLevel, headingLevelSection)...)

	// The label grows so the badge sits hard against the trailing edge —
	// the same FlexGrow-not-JustifyBetween pinning ListRow settled on.
	items = append(items, core.Box(
		core.FlexGrow(1),
		core.Text(h.Group.Label, label...),
	))
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
