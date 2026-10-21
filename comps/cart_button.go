package comps

import (
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernCartButtonInert is raised, in debug builds only, when a CartButton
// has no OnTap. It is announced as a button and counts the cart, and a tap
// on it goes nowhere: the one control every shopping screen shares,
// silently broken on all of them.
const ConcernCartButtonInert = "cart-button-inert"

// CartButton is the cart glyph with a count beside it, for an app bar: the
// way to the cart from every screen, saying how full it is.
//
//	q := hooks.UseCart(ctx, store)
//	comps.AppBar{Title: "Shop", Actions: []core.View{
//	    comps.CartButton{Count: q.Count(), OnTap: openCart},
//	}}
//
//	┌ Row  role=button  named "Cart, 3 items" ┐
//	│   🛒  ( 3 )                              │  glyph hidden; Badge
//	└──────────────────────────────────────────┘
//
// # The count is drawn beside the glyph, not over its corner
//
// A badge pinned to an icon's corner needs an overlay positioned in absolute
// terms, which each host lays out differently and which a long count ("99+")
// pushes off the icon. Beside it, the count is ordinary row layout on every
// host, and the button's tap target grows to hold it. With nothing in the
// cart no badge is drawn: a "0" pill reads as a notification.
//
// # Accessibility
//
// The Row is the button, named as one phrase, "Cart, 3 items", so a reader
// hears the destination and its state together. The glyph and the badge are
// inside a button and so presentational. The name changes as the count
// does; a reader on the button hears the new count when they next focus it,
// and is not interrupted by every add elsewhere on the screen.
//
// # No hooks
//
// May be rendered conditionally.
//
// # Theme roles read
//
//	Glyph    Typography.Title's size
//	Badge    as Badge, with Variant
//	Insets   Spacing.SM horizontal, Spacing.XS vertical, so the target is
//	         larger than the glyph
type CartButton struct {
	// Count is the number of things in the cart (cart.Quote.Count).
	Count int

	// OnTap opens the cart. Nil reports ConcernCartButtonInert.
	OnTap func()

	// Label is the spoken destination. Empty is "Cart".
	Label string

	// Glyph is the drawn mark. Empty is "🛒".
	Glyph string

	// CountLabel phrases the count for the spoken name ("3 items"). Nil is
	// English.
	CountLabel func(n int) string

	// MaxCount is the largest count drawn in full; above it the badge reads
	// "<MaxCount>+". Zero is 99. The spoken name always says the real count.
	MaxCount int

	// Variant colours the badge. Zero is the theme's primary.
	Variant Variant

	// Style is applied to the row after the widget's own props.
	Style []core.StyleProp
}

// Render draws the button. It takes no hook slot.
func (b CartButton) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	if core.IsDebugMode() && b.OnTap == nil {
		core.ReportConcern(ConcernCartButtonInert,
			"CartButton has no OnTap, so it is announced as a button and opens nothing")
	}
	onTap := b.OnTap
	if onTap == nil {
		onTap = func() {}
	}

	items := make([]core.PropsAndChildren, 0, len(b.Style)+10)
	items = append(items,
		// Clears the theme Row's screen inset, then sets the button's own as
		// explicit sides rather than axis shorthands: a shorthand is resolved
		// by each renderer into any side left at zero, so a caller's
		// PaddingLeft(0) would come back as the shorthand's number. Sides
		// are the caller's to clear one at a time (core/padding_sides.go).
		core.Padding(0),
		core.PaddingLeft(t.Spacing.SM),
		core.PaddingRight(t.Spacing.SM),
		core.PaddingTop(t.Spacing.XS),
		core.PaddingBottom(t.Spacing.XS),
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.OnClick(onTap),
		core.AccessibilityRole(core.RoleButton),
		core.AccessibilityLabel(orDefault(b.Label, "Cart")+", "+itemCount(b.Count, b.CountLabel)),
	)
	items = append(items, asProps(b.Style)...)
	items = append(items, core.Text(orDefault(b.Glyph, "🛒"),
		core.FontSize(t.Typography.Title.FontSize),
		core.AccessibilityHidden(),
	))
	if b.Count > 0 {
		items = append(items, Badge{Text: b.badgeText(), Variant: b.Variant})
	}
	return core.Row(items...).Render(ctx)
}

// badgeText is the count as drawn, capped at MaxCount.
func (b CartButton) badgeText() string {
	limit := b.MaxCount
	if limit <= 0 {
		limit = 99
	}
	if b.Count > limit {
		return strconv.Itoa(limit) + "+"
	}
	return strconv.Itoa(b.Count)
}
