package comps

import "github.com/rohanthewiz/grmob/core"

// KeyValueList is the label-and-value table of an order summary, a profile
// or an about screen: one row per fact, the name on the leading side and the
// value pinned to the trailing edge.
//
//	comps.KeyValueList{Rows: []comps.KeyValue{
//	    {Key: "Order", Value: "#40121"},
//	    {Key: "Placed", Value: "14 Mar 2026"},
//	    {Key: "Total", Value: "$42.10"},
//	}}
//
//	┌──────────────────────────────────────────┐
//	│ Order                            #40121  │
//	│ Placed                      14 Mar 2026  │
//	│ Total                            $42.10  │
//	└──────────────────────────────────────────┘
//
// Each row is a ListRow — the key is its Leading, the value its Trailing, and
// the empty middle column grows between them — so the value is pinned by
// ListRow's own spine and the list inherits that widget's answer to "how does
// the trailing slot stay at the edge". Nothing here re-solves layout. The key
// is Leading rather than Title so it can be pinned at its width: a long value
// wraps, a key never does.
//
// # The two inks
//
// The key is in the body ink and the value in the secondary one. That is the
// iOS "value" cell and the Material list's supporting text: the key is what
// the eye scans down, and the value is what it stops on once it has found the
// line. The opposite weighting (quiet keys, loud values) reads as a form that
// has been filled in, which is a different screen.
//
// # Accessibility
//
// The column is a RoleList and every row a listitem, so a reader hears "list,
// 3 items" and can step through them — a fact sheet is a list, and saying so
// costs nothing because this widget owns both halves of the structure (the
// ownership rule ListRow.NestingLevel describes, satisfied from inside).
//
// Each row is named "Key, Value". Unlike ListRow, which will not synthesize a
// name because its slots can carry meaning it cannot see, this widget knows
// both strings exactly, which is Avatar's reason for naming itself. The name
// also makes a row one stop on the natives, where two bare Texts would be two
// swipes and a reader would hear "Total" and "$42.10" as unrelated items.
//
// # Theme roles read
//
//	Key        Typography.Body over TextPrimary
//	Value      Typography.Body over TextSecondary
//	Rules      ColorPalette.BorderColor, through Separator, when Dividers
//	Row inset  the theme's Row base, through ListRow
type KeyValueList struct {
	// Rows are the facts, drawn top to bottom.
	Rows []KeyValue

	// Dividers draws a hairline Separator between rows — not above the first
	// or below the last, where the list's own container supplies the edge.
	Dividers bool

	// Label names the list for assistive technology ("Order details"). Empty
	// leaves it unnamed, which is fine under a visible heading.
	Label string

	// Style is applied to the list column after its defaults.
	Style []core.StyleProp
}

// KeyValue is one row of a KeyValueList.
type KeyValue struct {
	// Key names the fact ("Total"); Value states it ("$42.10"). An empty Value
	// draws the key alone, for a fact still loading.
	Key, Value string
}

// Render builds Column(role=list, ListRow(listitem)…).
func (l KeyValueList) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(l.Style)+2*len(l.Rows)+4)
	items = append(items,
		// The theme's Column base pads a screen; the rows already carry the
		// Row base's inset, and a second one around them would indent the
		// list away from every other list on the screen.
		core.Padding(0),
		core.Gap(0),
		core.AccessibilityRole(core.RoleList),
	)
	if l.Label != "" {
		items = append(items, core.AccessibilityLabel(l.Label))
	}
	items = append(items, asProps(l.Style)...)

	for i, kv := range l.Rows {
		// A Separator is AccessibilityHidden, which keeps it out of the
		// list's children as a reader counts them — a visible rule between
		// listitems does not become a foreign child of the list.
		if l.Dividers && i > 0 {
			items = append(items, Separator{})
		}
		items = append(items, l.row(t, kv))
	}
	return core.Column(items...).Render(ctx)
}

// row is one fact as a ListRow at depth 1: the key leading and the value
// trailing. NestingLevel is what gives the row its listitem role; 1 is the
// honest depth of a flat list.
func (l KeyValueList) row(t *core.Theme, kv KeyValue) core.View {
	name := kv.Key
	var trailing core.View
	if kv.Value != "" {
		name += ", " + kv.Value
		trailing = core.Text(kv.Value,
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextSecondary),
			// A long value ("12 Kingfisher Lane, Apt 4") wraps against the
			// trailing edge rather than hanging off it, so it lines up with
			// the short ones above it.
			core.Align(core.AlignEnd),
		)
	}
	return ListRow{
		// The key is the Leading slot, not the Title, so that it can refuse
		// to shrink — ListRow's own advice for a text Leading. As the Title
		// it sat in the growing middle column, and a long value took the
		// row's width from it: "Ship to" broke onto two lines beside a
		// wrapped address. Leading and pinned, the key keeps its line and
		// the value is what wraps; the empty middle still grows, so the
		// value is still pinned to the trailing edge.
		Leading: core.Text(kv.Key,
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextPrimary),
			core.FlexShrink(0),
		),
		Trailing:           trailing,
		NestingLevel:       1,
		AccessibilityLabel: name,
	}
}
