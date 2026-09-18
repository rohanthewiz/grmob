package comps

import "github.com/rohanthewiz/grmob/core"

// BottomBar is the strip pinned to the bottom of a screen: two to five
// destinations (Home, Search, Me) or two to five actions (Reply, Archive,
// Delete), each an icon over a label.
//
//	comps.Screen{
//	    Children: []core.View{feed},
//	    Footer: comps.BottomBar{
//	        Items: []comps.BarItem{
//	            {Icon: "🏠", Label: "Home", OnTap: home},
//	            {Icon: "🔍", Label: "Search", OnTap: find},
//	            {Icon: "👤", Label: "Me", OnTap: profile},
//	        },
//	        Selected: tab.Get(),
//	    },
//	}
//
// It belongs in Screen.Footer, which pins it outside the scrolling region; a
// bar placed among Screen.Children scrolls away with the content.
//
// # Destinations or actions: the role follows Selected
//
// SegmentedControl is a choice between views of one thing and styles its live
// segment as a filled chip, which is wrong for a bar. The bar's one design
// decision is what it announces itself as, and it is made by the field the
// caller already has to set:
//
//	Selected >= 0   RoleNavigation   a set of destinations, one of them current
//	Selected <  0   RoleToolbar      a strip of actions, none of them "current"
//
// Selected's zero value picks the first item, the same rule SegmentedControl
// and Tabs follow, so an action strip says Selected: -1.
//
// # Items share the width equally
//
// Each item is a Column with FlexGrow(1), so the tap targets tile the whole
// bar. Justify(JustifyAround) on content-sized buttons would draw the same
// spacing but leave dead gaps between targets, which is exactly where a thumb
// aimed at a small label lands.
//
// # How the current item is announced
//
// The current cell states core.CurrentPage: aria-current="page" on the web,
// and the selected state on Compose and SwiftUI, which is how both platforms'
// own navigation bars announce the destination they show. It used to append
// ", selected" to its name, because core had no current state and RoleTab
// would claim a tab panel this bar does not control (examples/social builds
// that relationship explicitly when it wants it). The name is now the label
// alone, and stays the same as the selection moves. The Icon is decoration and
// is hidden from assistive technology, so the Label is what is read.
//
// # Badges
//
// BarItem.Badge puts a count or a word on an item — "3" on Inbox — as a
// comps.Badge over the top-end corner of the icon. The icon becomes a
// two-layer core.ZStack: the glyph, centred, and the Badge placed
// core.StackAlignTopEnd.
//
//	┌ ZStack ────────────────┐
//	│            ┌───┐       │   the glyph keeps a margin either side, so
//	│    ┌────┐  │ 3 │       │   the stack is wider than the glyph and the
//	│    │ ✉  │──┴───┘       │   compact badge sits over the glyph's
//	│    └────┘              │   top-end corner rather than across it
//	└────────────────────────┘
//
// The margin is horizontal only, and symmetric. A badge that rose above the
// glyph would need a negative offset, which no target takes portably, and a
// top margin on the glyph to make room would push a badged icon lower than its
// unbadged neighbours; the badge sits level with the glyph's top instead, and
// every icon in the bar stays on one line. An item with no Badge draws the
// icon exactly as before — no stack, no margin — so a bar that never uses one
// renders the tree it always did.
//
// With no Icon the Label wears the badge, by the same two layers.
//
// The count is read as part of the item's name ("Inbox, 3"), because the cell
// is one button with one name and a label on a button replaces the text
// inside it on every target; the badge's own Text is hidden so it is not
// heard twice where it would be. BadgeLabel is the spoken form when the digits
// alone say too little ("3 unread").
//
// # Theme roles read
//
//	Bar background   Colors.Surface
//	Current item     Colors.PrimaryOnLightColor(), bold
//	Other items      Colors.TextSecondary
//	Label text       Typography.Caption; Icon uses Typography.Subtitle
//	Padding          Spacing.XS
//	Badge            VariantError, through Badge — the colour both
//	                 platforms give a notification count — at
//	                 Typography.Caption less two points
type BottomBar struct {
	// Items are the destinations or actions, drawn leading to trailing.
	Items []BarItem

	// Selected is the index of the current destination. A negative value makes
	// the bar an action toolbar with no current item.
	Selected int

	// Style is applied to the bar row after the widget's own props.
	Style []core.StyleProp
}

// BarItem is one cell of a BottomBar.
type BarItem struct {
	// Label is the text under the icon and the item's accessible name.
	Label string

	// Icon is an optional glyph or emoji drawn above the label.
	Icon string

	// OnTap is called when the item is pressed.
	OnTap func()

	// AccessibilityLabel replaces Label as the spoken name, for a bar whose
	// labels are abbreviated.
	AccessibilityLabel string

	// Badge is a count or a short word drawn over the icon's top-end corner;
	// empty draws none. See "Badges" on BottomBar.
	Badge string

	// BadgeLabel is how the badge is read as part of the item's name; empty
	// reads Badge itself. "3 unread" says more than "3".
	BadgeLabel string
}

// Render builds Row(Column(icon, label)...) with the role chosen by Selected.
func (b BottomBar) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	role := core.RoleToolbar
	if b.Selected >= 0 {
		role = core.RoleNavigation
	}

	items := make([]core.PropsAndChildren, 0, len(b.Style)+len(b.Items)+4)
	items = append(items,
		core.AlignItemsProp(core.AlignItemsStretch),
		core.BackgroundColor(t.Colors.Surface),
		core.Padding(t.Spacing.XS),
		core.AccessibilityRole(role),
	)
	for _, sp := range b.Style {
		items = append(items, sp)
	}
	for i, it := range b.Items {
		items = append(items, b.item(t, it, i == b.Selected))
	}
	return core.Row(items...).Render(ctx)
}

// item builds one equal-width cell. The cell, not the label, is the button, so
// the whole tile is the target and is one tab stop on the web.
func (b BottomBar) item(t *core.Theme, it BarItem, current bool) core.View {
	color := t.Colors.TextSecondary
	weight := core.Normal
	if current {
		color = t.Colors.PrimaryOnLightColor()
		weight = core.Bold
	}

	// The name no longer changes with the selection: the current state is
	// said below, as a state. It does change with the badge, which is content
	// rather than state — there is no property on any target that could carry
	// "3 unread" apart from the name.
	name := orDefault(it.AccessibilityLabel, it.Label)
	if it.Badge != "" {
		spoken := orDefault(it.BadgeLabel, it.Badge)
		if name == "" {
			name = spoken
		} else {
			name += ", " + spoken
		}
	}

	cell := []core.PropsAndChildren{
		core.FlexGrow(1),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(2),
		core.Padding(t.Spacing.XS),
		core.AccessibilityRole(core.RoleButton),
	}
	// Only the current cell states it. CurrentNone is ARIA's "false", and a
	// bar that wrote it on every other cell would add nothing a reader says.
	if current {
		cell = append(cell, core.AccessibilityCurrent(core.CurrentPage))
	}
	if name != "" {
		cell = append(cell, core.AccessibilityLabel(name))
	}
	if it.OnTap != nil {
		cell = append(cell, core.OnClick(it.OnTap))
	}
	// The badge rides the first thing drawn: the icon, or the label when
	// there is no icon. Exactly one of the two carries it.
	badgeOnIcon := it.Badge != "" && it.Icon != ""
	badgeOnLabel := it.Badge != "" && it.Icon == ""
	if it.Icon != "" {
		cell = append(cell, b.badged(t, badgeOnIcon, it.Badge, core.Text(it.Icon,
			b.badgedMargin(t, badgeOnIcon,
				core.UseStyle(t.Typography.Subtitle),
				core.TextColor(color),
				core.AccessibilityHidden(),
			)...,
		)))
	}
	if it.Label != "" {
		cell = append(cell, b.badged(t, badgeOnLabel, it.Badge, core.Text(it.Label,
			b.badgedMargin(t, badgeOnLabel,
				core.UseStyle(t.Typography.Caption),
				core.TextColor(color),
				core.FontWeight(weight),
			)...,
		)))
	}
	return core.Column(cell...)
}

// badged returns view unchanged when on is false — the tree every unbadged
// item has always had — and otherwise the two-layer stack of view and a
// Badge placed at its top-end corner. See "Badges" on BottomBar.
func (b BottomBar) badged(t *core.Theme, on bool, badge string, view core.View) core.View {
	if !on {
		return view
	}
	return core.ZStack(
		view,
		Badge{
			Text:    badge,
			Variant: VariantError,
			Style: []core.StyleProp{
				// The compact pill both platforms' bars use for a count, two
				// points under the Caption a free-standing Badge takes. At
				// full size a one-digit badge covered most of an 18px glyph;
				// compact, it sits over the corner and the glyph stays legible.
				core.FontSize(t.Typography.Caption.FontSize - 2),
				core.PaddingVertical(1),
				core.PaddingHorizontal(5),
				core.StackAlign(core.StackAlignTopEnd),
				// The count is already in the cell's name; heard again here
				// it would be "Inbox, 3, 3" wherever a target reads the
				// children of a named button.
				core.AccessibilityHidden(),
			},
		},
	)
}

// badgedMargin adds the symmetric horizontal margin a badged glyph keeps, so
// the stack is wide enough for the badge to sit over the glyph's corner
// rather than on top of all of it. XS + SM (12px on the bundled themes)
// puts a one-digit compact badge over the glyph's top-end corner, overlapping
// it by a few pixels, which is where both platforms' own bars put it; a
// two-digit count reaches further in.
func (b BottomBar) badgedMargin(t *core.Theme, on bool, props ...core.StyleProp) []core.StyleProp {
	if on {
		props = append(props, core.MarginHorizontal(t.Spacing.XS+t.Spacing.SM))
	}
	return props
}
