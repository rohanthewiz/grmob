package comps

import "github.com/rohanthewiz/grmob/core"

// Breadcrumb is the trail from the root of a hierarchy to the current page —
// Files › Photos › 2026 — where every step but the last is a way back up.
//
//	comps.Breadcrumb{
//	    Items: []string{"Files", "Photos", "2026"},
//	    OnTap: func(i int) { nav.PopTo(i) },
//	}
//
//	┌ Row role=navigation "Breadcrumb" ───────────────────┐
//	│ [Files]  ›  [Photos]  ›  2026                       │
//	└─────────────────────────────────────────────────────┘
//	  ghost       ghost        Text, aria-current="page"
//	  Buttons     Buttons
//
// # The last item is not a button
//
// It is the page the reader is on, so tapping it would go nowhere; drawing it
// as a control would put a dead tab stop at the end of every trail. It is a
// Text in the primary ink carrying core.CurrentPage, which is how every
// target says "you are here" (aria-current on the web, the selected state on
// the natives — see core.AccessibilityCurrent).
//
// # Controlled, and read-only when OnTap is nil
//
// Breadcrumb holds no state. OnTap reports the index of the ancestor tapped
// and the caller navigates; the trail it is handed next is the new truth.
// With OnTap nil every item is drawn as text, which is the trail as a
// location label (a heading's context line) rather than as navigation — a
// legitimate use, and why a nil OnTap is not a concern the way an inert
// picker is: nothing here pretends to be a control.
//
// # Long trails wrap
//
// The row wraps (core.FlexWrap) rather than scrolling or truncating. A deep
// trail on a phone becomes two lines, every step still visible and tappable,
// which is what the reader who went five levels down needs to get back out.
// Collapsing the middle into "…" is a second design (a menu of the hidden
// steps) and a caller that wants it can pass the shortened Items itself.
//
// # Accessibility
//
// The row is RoleNavigation named "Breadcrumb", ARIA's own breadcrumb pattern:
// a landmark a reader can jump to, whose items are its links. The chevrons are
// decoration and hidden, so a reader hears "Files, button; Photos, button;
// 2026, current page" and not the separators between them.
//
// # Theme roles read
//
//	Ancestors  Colors.Primary's on-light tone, through a ghost Button
//	Current    Typography.Body over TextPrimary
//	Chevron    Typography.Body over TextSecondary
//	Gaps       Spacing.XS between every item
type Breadcrumb struct {
	// Items are the steps from the root to the current page, in that order.
	// The last is the current page.
	Items []string

	// OnTap receives the index of the ancestor tapped. It is never called for
	// the last item. Nil draws the whole trail as text.
	OnTap func(i int)

	// Label names the navigation landmark; empty gives "Breadcrumb".
	Label string

	// Separator is the glyph between items; empty gives "›".
	Separator string

	// Style is applied to the row after its defaults.
	Style []core.StyleProp
}

// Render builds Row(role=navigation, Button › Button › … › Text).
func (b Breadcrumb) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(b.Style)+2*len(b.Items)+6)
	items = append(items,
		// A breadcrumb sits above a page's heading, inside whatever inset the
		// screen already has; the theme Row base's padding would push it off
		// the heading's left edge.
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.FlexWrap(true),
		core.AccessibilityRole(core.RoleNavigation),
		core.AccessibilityLabel(orDefault(b.Label, "Breadcrumb")),
	)
	items = append(items, asProps(b.Style)...)

	sep := orDefault(b.Separator, "›")
	last := len(b.Items) - 1
	for i, label := range b.Items {
		if i > 0 {
			items = append(items, core.Text(sep,
				core.UseStyle(t.Typography.Body),
				core.TextColor(t.Colors.TextSecondary),
				core.AccessibilityHidden(),
			))
		}
		items = append(items, b.item(t, i, label, i == last))
	}
	return core.Row(items...).Render(ctx)
}

// item is one step: a ghost Button for an ancestor the reader can go back to,
// or a Text for the current page and for every step of a read-only trail.
func (b Breadcrumb) item(t *core.Theme, i int, label string, current bool) core.View {
	if current || b.OnTap == nil {
		props := []core.StyleProp{core.UseStyle(t.Typography.Body)}
		if current {
			props = append(props,
				core.TextColor(t.Colors.TextPrimary),
				core.AccessibilityCurrent(core.CurrentPage),
			)
		} else {
			props = append(props, core.TextColor(t.Colors.TextSecondary))
		}
		return core.Text(label, props...)
	}
	onTap := b.OnTap
	return Button{
		Label:    label,
		OnTap:    func() { onTap(i) },
		Emphasis: EmphasisGhost,
		// The theme's Button base pads a free-standing button to a
		// comfortable target; between chevrons that reads as a gap twice the
		// size of the one around the current page. The vertical padding is
		// kept, so the target keeps its height.
		Style: []core.StyleProp{core.PaddingHorizontal(t.Spacing.XS)},
	}
}
