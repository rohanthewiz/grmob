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
// There is no aria-current in core's vocabulary and RoleTab would claim a tab
// panel this bar does not control (examples/social builds that relationship
// explicitly when it wants it). So the current item takes ListRow's fallback:
// its accessible name gains ", selected". The Icon is decoration and is hidden
// from assistive technology, so the Label is what is read.
//
// # Theme roles read
//
//	Bar background   Colors.Surface
//	Current item     Colors.PrimaryOnLightColor(), bold
//	Other items      Colors.TextSecondary
//	Label text       Typography.Caption; Icon uses Typography.Subtitle
//	Padding          Spacing.XS
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

	name := orDefault(it.AccessibilityLabel, it.Label)
	if current && name != "" {
		name += ", selected"
	}

	cell := []core.PropsAndChildren{
		core.FlexGrow(1),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(2),
		core.Padding(t.Spacing.XS),
		core.AccessibilityRole(core.RoleButton),
	}
	if name != "" {
		cell = append(cell, core.AccessibilityLabel(name))
	}
	if it.OnTap != nil {
		cell = append(cell, core.OnClick(it.OnTap))
	}
	if it.Icon != "" {
		cell = append(cell, core.Text(it.Icon,
			core.UseStyle(t.Typography.Subtitle),
			core.TextColor(color),
			core.AccessibilityHidden(),
		))
	}
	if it.Label != "" {
		cell = append(cell, core.Text(it.Label,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(color),
			core.FontWeight(weight),
		))
	}
	return core.Column(cell...)
}
