package comps

import (
	"github.com/rohanthewiz/grmob/cart"
	"github.com/rohanthewiz/grmob/core"
)

// ConcernCartListDuplicateID is raised, in debug builds only, when two of a
// CartList's items share an ID. Lines are keyed by ID, so the second would
// collide with the first in the reconciler, and every cart.Store action
// addresses a line by ID, so a tap on either line's "+" would change the
// first. cart.Add never produces this; a hand-built slice can.
const ConcernCartListDuplicateID = "cart-list-duplicate-id"

// CartList is a shopping cart's lines: a CartLine per item with a separator
// between them, or an empty state when there are none.
//
//	q := hooks.UseCart(ctx, store)
//	comps.CartList{
//	    Items:      q.Cart.Items,
//	    Currency:   q.Currency,
//	    OnQuantity: store.SetQuantity,
//	    OnRemove:   store.Remove,
//	}
//
//	┌ Column ────────────────────────────────────────┐
//	│  CartLine  Pour-over kettle   [−] 1 [+]  $45   │  keyed "line:kettle"
//	│  ───────────────────────────────────────────── │  keyed "sep:filters"
//	│  CartLine  Filters (100)      [−] 2 [+]  $13   │  keyed "line:filters"
//	└────────────────────────────────────────────────┘
//
//	no items:  EmptyState  🛒  "Your cart is empty"   (or the Empty slot)
//
// # Controlled, with store-shaped callbacks
//
// OnQuantity and OnRemove take the item's ID, the shapes cart.Store's
// SetQuantity and Remove already have, so the usual wiring is the two method
// values above. Leave both nil (or set ReadOnly) for an order-review list:
// the lines draw no controls at all.
//
// # Configuring a line
//
// The fields shared by every line (Currency, Format, SwipeToRemove,
// HideImages, the labels, Disabled) are passed through to each CartLine. Line
// replaces the CartLine altogether, for a row this widget does not draw (a
// gift message under each item, a subscription interval picker); the list
// keeps keying, separating and the empty state.
//
// # Keys
//
// Lines are keyed by item ID, so a removal or a quantity change patches one
// line and leaves the others, and their Stepper focus, alone. Separators are
// keyed in their own namespace, by the ID of the line below them, so the
// separator above a removed line goes with it.
//
// # No hooks
//
// No hooks of its own; may be rendered conditionally. A Line func that calls
// hooks makes the list a hook caller with a slot count that varies with the
// cart, so a Line must be hook-free too.
//
// # Theme roles read
//
//	Gap     Spacing.SM between a line and a separator
//	Lines   as CartLine; separators as Separator; empty as EmptyState
type CartList struct {
	// Items are the lines, in order: normally a Quote's Cart.Items.
	Items []cart.Item

	// Currency is the ISO 4217 code amounts are formatted in. Empty is USD.
	Currency string

	// Format renders an amount. Nil is FormatMoney.
	Format func(amount int64, currency string) string

	// OnQuantity receives an item's ID and its new quantity. Nil draws no
	// Steppers.
	OnQuantity func(id string, quantity int)

	// OnRemove receives the ID of the item to remove. Nil draws no Remove
	// buttons.
	OnRemove func(id string)

	// ReadOnly draws no controls even when the callbacks are set, so a
	// review step can share its wiring with the editable cart.
	ReadOnly bool

	// SwipeToRemove passes through to every CartLine; see CartLine.
	SwipeToRemove bool

	// HideImages leaves every line's thumbnail out.
	HideImages bool

	// Line replaces the CartLine for each item. It must be hook-free.
	Line func(item cart.Item) core.View

	// Empty replaces the empty state drawn when there are no items.
	Empty core.View

	// EmptyTitle and EmptyHint are the built empty state's lines. Empty
	// title is "Your cart is empty"; empty hint draws none.
	EmptyTitle, EmptyHint string

	// QuantityLabel and RemoveLabel pass through to every CartLine.
	QuantityLabel, RemoveLabel string

	// Disabled disables every line's controls.
	Disabled bool

	// Style is applied to the outer column after the widget's own props.
	Style []core.StyleProp
}

// Render draws the lines, or the empty state. It takes no hook slot.
func (l CartList) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(l.Style)+2*len(l.Items)+2)
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
	)
	items = append(items, asProps(l.Style)...)

	if len(l.Items) == 0 {
		empty := l.Empty
		if empty == nil {
			empty = EmptyState{
				Glyph: "🛒",
				Title: orDefault(l.EmptyTitle, "Your cart is empty"),
				Hint:  l.EmptyHint,
			}
		}
		items = append(items, empty)
		return core.Column(items...).Render(ctx)
	}

	seen := make(map[string]bool, len(l.Items))
	for i, it := range l.Items {
		if core.IsDebugMode() && seen[it.ID] {
			core.ReportConcern(ConcernCartListDuplicateID,
				"CartList has two items with ID \""+it.ID+"\"; lines are keyed and addressed by ID, so the second collides with the first")
		}
		seen[it.ID] = true
		if i > 0 {
			items = append(items, core.Keyed("sep:"+it.ID, Separator{}))
		}
		items = append(items, core.Keyed("line:"+it.ID, l.line(it)))
	}
	return core.Column(items...).Render(ctx)
}

// line is the view for one item: the caller's Line, or a CartLine bound to
// it. The closures capture id per item (a fresh variable per iteration
// since Go 1.22), so each line's handlers address their own item.
func (l CartList) line(it cart.Item) core.View {
	if l.Line != nil {
		return l.Line(it)
	}
	line := CartLine{
		Item:          it,
		Currency:      l.Currency,
		Format:        l.Format,
		SwipeToRemove: l.SwipeToRemove,
		HideImage:     l.HideImages,
		QuantityLabel: l.QuantityLabel,
		RemoveLabel:   l.RemoveLabel,
		Disabled:      l.Disabled,
	}
	if !l.ReadOnly {
		id := it.ID
		if onQty := l.OnQuantity; onQty != nil {
			line.OnQuantity = func(n int) { onQty(id, n) }
		}
		if onRemove := l.OnRemove; onRemove != nil {
			line.OnRemove = func() { onRemove(id) }
		}
	}
	return line
}
