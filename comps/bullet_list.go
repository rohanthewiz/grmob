package comps

import (
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// BulletList is a short run of points, each behind a marker: the key points
// under a lesson, the steps of a recipe, what a plan includes.
//
//	comps.BulletList{Items: []string{"Free delivery", "Cancel any time"}}
//	comps.BulletList{Items: steps, Ordered: true}
//
//	┌ Column  role=list ──────────────────────────────┐
//	│ ┌ Row  listitem ─────────────────────────────┐  │
//	│ │  •   Free delivery on every order over     │  │
//	│ │      twenty pounds                         │  │  wraps under itself
//	│ └────────────────────────────────────────────┘  │
//	│ ┌ Row  listitem ─────────────────────────────┐  │
//	│ │  •   Cancel any time                       │  │
//	│ └────────────────────────────────────────────┘  │
//	└─────────────────────────────────────────────────┘
//
// # The marker column
//
// The marker refuses to shrink and the text grows, so a long item wraps under
// its own first word rather than back under the bullet — the hanging indent
// every word processor draws. Ordered markers ("1.", "2." … "10.") are
// right-aligned in a column sized for the widest of them, so the item text
// starts at one x whatever the number's width. The column is sized from the
// last marker's character count at the body size, because no host reports a
// rendered width; digits are tabular in every bundled face, so the estimate
// only has to cover the widest digit.
//
// # Not core.List
//
// A bullet list is short by construction, and static children need no keys:
// the reason the tutorial's keyPoints were plain Rows before this existed.
// core.List's laziness is for data of unknown length.
//
// # Accessibility
//
// The column is a RoleList and each row a listitem named by its text, so a
// reader hears "list, 3 items" and each point once. The marker is hidden: a
// screen reader states the position itself ("2 of 5"), and "bullet, Free
// delivery" is the marker read as a word.
//
// # Theme roles read
//
//	Marker     Colors.Primary's on-light tone, bold
//	Text       Typography.Body
//	Gap        Spacing.XS between items, Spacing.SM after the marker
type BulletList struct {
	// Items are the points, drawn top to bottom.
	Items []string

	// Ordered numbers the items instead of bulleting them.
	Ordered bool

	// Start is the first number of an ordered list; 0 means 1.
	Start int

	// Marker replaces the bullet of an unordered list; empty gives "•".
	Marker string

	// Label names the list for assistive technology. Empty leaves it unnamed,
	// which is fine under a visible heading.
	Label string

	// Style is applied to the list column after its defaults.
	Style []core.StyleProp
}

// Render builds Column(role=list, Row(listitem, marker, text)…). It takes no
// hook slot.
func (b BulletList) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	start := b.Start
	if start == 0 {
		start = 1
	}
	marker := func(i int) string {
		if b.Ordered {
			return strconv.Itoa(start+i) + "."
		}
		return orDefault(b.Marker, "•")
	}

	// The marker column's width, for ordered lists only: an unordered list's
	// markers are all the same glyph and need no alignment. 0.62 em per
	// character covers a tabular digit and the full stop with room to spare;
	// see "The marker column".
	var markerWidth float64
	if b.Ordered && len(b.Items) > 0 {
		size := t.Typography.Body.FontSize
		if size == 0 {
			size = 16
		}
		widest := len(marker(len(b.Items) - 1))
		if first := len(marker(0)); first > widest {
			// A negative Start can make the first marker the widest ("-10.").
			widest = first
		}
		markerWidth = float64(widest) * 0.62 * size
	}

	items := make([]core.PropsAndChildren, 0, len(b.Style)+len(b.Items)+4)
	items = append(items,
		// The theme's Column base pads a screen; a list inside a screen
		// would be indented a second time.
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		core.AccessibilityRole(core.RoleList),
	)
	if b.Label != "" {
		items = append(items, core.AccessibilityLabel(b.Label))
	}
	items = append(items, asProps(b.Style)...)

	for i, text := range b.Items {
		mark := []core.StyleProp{
			core.UseStyle(t.Typography.Body),
			core.TextColor(VariantDefault.OnLight(t)),
			core.FontWeight(core.Bold),
			core.FlexShrink(0),
			core.AccessibilityHidden(),
		}
		if markerWidth > 0 {
			mark = append(mark,
				core.Width(strconv.FormatFloat(markerWidth, 'f', 1, 64)+"px"),
				core.Align(core.AlignEnd),
			)
		}
		items = append(items, core.Row(
			// The Row base's inset belongs to list rows with a surface; these
			// sit in running content.
			core.Padding(0),
			core.Gap(float64(t.Spacing.SM)),
			// Start, not centre: a wrapped item's marker belongs beside its
			// first line.
			core.AlignItemsProp(core.AlignItemsStart),
			core.AccessibilityRole(core.RoleListItem),
			core.AccessibilityNestingLevel(1),
			core.AccessibilityLabel(text),
			core.Text(marker(i), mark...),
			core.Text(text,
				core.UseStyle(t.Typography.Body),
				core.FlexGrow(1),
				core.FlexShrink(1),
				core.AccessibilityHidden(),
			),
		))
	}
	return core.Column(items...).Render(ctx)
}
