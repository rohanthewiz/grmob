package comps

import (
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/cart"
	"github.com/rohanthewiz/grmob/core"
)

// cartLineQuantityCeiling bounds a line's Stepper when the item has no cap
// of its own. A Stepper enforces bounds only when Max > Min, and an
// unbounded one would let "−" walk the quantity through zero, so every line
// gets a ceiling: one no phone cart meets, which keeps the "−" stopping at
// one and leaves removal to the Remove button, where it says what it does.
const cartLineQuantityCeiling = 99

// cartThumbSize is the default thumbnail edge: large enough to recognise a
// product, small enough that the label keeps most of a phone's width.
const cartThumbSize = 64

// CartLine is one line of a shopping cart: a thumbnail, the item's name and
// price, a Stepper for its quantity and a Remove button.
//
//	comps.CartLine{
//	    Item:       it,                      // a cart.Item
//	    Currency:   q.Currency,
//	    OnQuantity: func(n int) { store.SetQuantity(it.ID, n) },
//	    OnRemove:   func() { store.Remove(it.ID) },
//	}
//
//	┌ Row  (swipe toward the leading edge removes, when SwipeToRemove) ─┐
//	│ ┌────┐ ┌ Column ──────────────────────────────────────────────┐  │
//	│ │img │ │ ┌ Column  named "Pour-over kettle, Matte black,     ┐ │  │
//	│ │    │ │ │          2 × $45.00, $90.00"                      │ │  │
//	│ └────┘ │ │ Pour-over kettle                        $90.00    │ │  │
//	│ hidden │ │ Matte black                                       │ │  │
//	│        │ │ 2 × $45.00                                        │ │  │
//	│        │ └───────────────────────────────────────────────────┘ │  │
//	│        │ [ − ]  2  [ + ]                            Remove     │  │
//	│        └──────────────────────────────────────────────────────┘  │
//	└───────────────────────────────────────────────────────────────────┘
//
// Usually a CartList draws these; a CartLine on its own is for a screen that
// lays its lines out differently (a grid, a review step with extra columns).
//
// # Controlled, and the controls follow the callbacks
//
// The quantity is the caller's (normally a cart.Store's). The Stepper is
// drawn only when OnQuantity is set and the item can be more than one, and
// the Remove button only when OnRemove is set, so the zero CartLine{Item: it}
// is the read-only line an order-review screen wants, with no inert
// controls to report.
//
// The Stepper runs from one to the item's MaxQuantity (cart.Add fills that
// from the cart's Config), or to 99 when there is none. Its "−" stops at one
// rather than removing the line: a removal deserves a button that says so,
// not a fourth tap on a minus sign.
//
// # Swipe to remove is opt-in
//
// SwipeToRemove removes the line on a swipe toward the leading edge (left in
// a left-to-right layout), the gesture iOS's own lists use. It is off by
// default because nothing on screen hints at it and nothing confirms it: a
// list that deletes on a stray sideways drag needs an undo (a Snackbar that
// Replaces the cart from before), which is the caller's to provide. The
// Remove button stays either way: a screen-reader user cannot swipe a row.
//
// # Accessibility
//
// The item's text is one labelled Column, "Pour-over kettle, Matte black, 2 ×
// $45.00, $90.00", so a reader hears a product and its price together. The
// controls stay outside it, so they remain reachable, and their names carry
// the item: the Stepper is "Quantity, Pour-over kettle" and the button
// "Remove Pour-over kettle". A cart of three lines would otherwise read as
// three identical "Remove" buttons. The thumbnail is hidden; the label says
// what it shows.
//
// # No hooks
//
// May be rendered conditionally.
//
// # Theme roles read
//
//	Label, amount   Typography.Body, Colors.TextPrimary; amount bold
//	Detail, each    Typography.Caption, Colors.TextSecondary
//	Thumbnail       Colors.Surface behind the image, Spacing.SM corner radius
//	Stepper         as Stepper
//	Remove          a ghost Button, Colors.Error ink
type CartLine struct {
	// Item is the line drawn.
	Item cart.Item

	// Currency is the ISO 4217 code amounts are formatted in. Empty is USD.
	Currency string

	// Format renders an amount. Nil is FormatMoney.
	Format func(amount int64, currency string) string

	// OnQuantity receives the new quantity, between one and the cap. Nil
	// draws no Stepper.
	OnQuantity func(int)

	// OnRemove removes the line. Nil draws no Remove button.
	OnRemove func()

	// SwipeToRemove removes the line on a swipe toward the leading edge,
	// when OnRemove is set. See the type doc for why it is off by default.
	SwipeToRemove bool

	// Leading replaces the thumbnail: an icon, a colour swatch, a badge. It
	// wins over Item.ImageURL.
	Leading core.View

	// HideImage leaves the thumbnail out even when Item.ImageURL is set.
	HideImage bool

	// ImageSize is the thumbnail's edge in layout units. Zero is 64.
	ImageSize float64

	// QuantityLabel names the Stepper, before the item's label. Empty is
	// "Quantity".
	QuantityLabel string

	// RemoveLabel is the button's text, and its spoken name before the
	// item's label. Empty is "Remove".
	RemoveLabel string

	// Disabled disables the Stepper and the Remove button: a checkout is in
	// flight and the cart must hold still.
	Disabled bool

	// Style is applied to the outer row after the widget's own props.
	Style []core.StyleProp
}

// Render draws the line. It takes no hook slot.
func (l CartLine) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	it := l.Item
	format, currency := moneyFormat(l.Format, l.Currency)

	// A stored line is always at least one (cart.Add guarantees it); a
	// hand-built Item may not be, and is drawn as the one it would become.
	qty := max(it.Quantity, 1)
	total := format(int64(qty)*it.UnitAmount, currency)
	each := ""
	if qty > 1 {
		each = strconv.Itoa(qty) + " × " + format(it.UnitAmount, currency)
	}

	spoken := []string{it.Label}
	for _, s := range []string{it.Detail, each, total} {
		if s != "" {
			spoken = append(spoken, s)
		}
	}

	body := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		// Grow from a zero basis, so on the web a long label wraps inside
		// the row rather than pushing the amount off its end.
		core.FlexGrow(1),
		core.FlexBasis("0"),
		l.info(t, strings.Join(spoken, ", "), total, each),
	}
	if controls := l.controls(t); controls != nil {
		body = append(body, controls)
	}

	items := make([]core.PropsAndChildren, 0, len(l.Style)+6)
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsStart),
	)
	if remove := l.OnRemove; l.SwipeToRemove && remove != nil && !l.Disabled {
		items = append(items, core.OnSwipe(func(d core.SwipeDirection) {
			if d == core.SwipeLeading {
				remove()
			}
		}))
	}
	items = append(items, asProps(l.Style)...)
	if lead := l.leading(t); lead != nil {
		items = append(items, lead)
	}
	items = append(items, core.Column(body...))
	return core.Row(items...).Render(ctx)
}

// info is the labelled, inert part: the name and total on one row, the
// detail and the per-unit price under them.
func (l CartLine) info(t *core.Theme, spoken, total, each string) core.View {
	col := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(2),
		core.AccessibilityLabel(spoken),
		core.Row(
			core.Padding(0),
			core.Gap(float64(t.Spacing.SM)),
			core.AlignItemsProp(core.AlignItemsStart),
			core.Text(l.Item.Label,
				core.UseStyle(t.Typography.Body),
				core.TextColor(t.Colors.TextPrimary),
				core.FlexGrow(1),
				core.FlexBasis("0"),
			),
			core.Text(total,
				core.UseStyle(t.Typography.Body),
				core.TextColor(t.Colors.TextPrimary),
				core.FontWeight(core.Bold),
				core.FlexShrink(0),
			),
		),
	}
	for _, s := range []string{l.Item.Detail, each} {
		if s != "" {
			col = append(col, core.Text(s,
				core.UseStyle(t.Typography.Caption),
				core.TextColor(t.Colors.TextSecondary),
			))
		}
	}
	return core.Column(col...)
}

// controls is the Stepper and Remove row, or nil when there is neither.
func (l CartLine) controls(t *core.Theme) core.View {
	ceiling := l.Item.MaxQuantity
	if ceiling <= 0 {
		ceiling = cartLineQuantityCeiling
	}
	// A line capped at one has nothing to step, and a Stepper bounded 1..1
	// would not be bounded at all (Stepper enforces Max > Min only).
	showStepper := l.OnQuantity != nil && ceiling > 1
	if !showStepper && l.OnRemove == nil {
		return nil
	}

	row := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
	}
	if showStepper {
		onQty := l.OnQuantity
		row = append(row, Stepper{
			Value:    max(l.Item.Quantity, 1),
			Min:      1,
			Max:      ceiling,
			OnChange: onQty,
			Label:    orDefault(l.QuantityLabel, "Quantity") + ", " + l.Item.Label,
			Disabled: l.Disabled,
		})
	}
	if l.OnRemove != nil {
		// Takes the slack, pinning Remove to the trailing edge whether or
		// not a Stepper is beside it.
		row = append(row, core.Box(core.FlexGrow(1)))
		label := orDefault(l.RemoveLabel, "Remove")
		row = append(row, Button{
			Label:              label,
			OnTap:              l.OnRemove,
			Emphasis:           EmphasisGhost,
			Variant:            VariantError,
			Disabled:           l.Disabled,
			AccessibilityLabel: label + " " + l.Item.Label,
		})
	}
	return core.Row(row...)
}

// leading is the Leading slot, the thumbnail, or nil.
func (l CartLine) leading(t *core.Theme) core.View {
	if l.Leading != nil {
		return l.Leading
	}
	if l.HideImage || l.Item.ImageURL == "" {
		return nil
	}
	size := l.ImageSize
	if size <= 0 {
		size = cartThumbSize
	}
	return core.ImageWithMode(l.Item.ImageURL, core.ContentModeFill,
		core.Width(px(size)),
		core.Height(px(size)),
		core.FlexShrink(0),
		core.BorderRadius(float64(t.Spacing.SM)),
		// Surface behind the picture while it loads, so the slot is visible
		// as a slot rather than a gap the label jumps into.
		core.BackgroundColor(t.Colors.Surface),
		core.AccessibilityHidden(),
	)
}

// moneyFormat resolves a widget's Format and Currency fields: nil is
// FormatMoney, empty is USD, and the code is upper-cased once here so every
// amount on the widget is formatted the same way.
func moneyFormat(format func(int64, string) string, currency string) (func(int64, string) string, string) {
	if format == nil {
		format = FormatMoney
	}
	return format, strings.ToUpper(orDefault(currency, "usd"))
}

// itemCount is the "3 items" phrase CartSummary and CartButton say, through
// the caller's override when there is one.
func itemCount(n int, label func(int) string) string {
	if label != nil {
		return label(n)
	}
	if n == 1 {
		return "1 item"
	}
	return strconv.Itoa(n) + " items"
}
