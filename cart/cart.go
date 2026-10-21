// Package cart is the model behind a shopping cart: its lines, the rules that
// price them, and a Store that every screen of an app can share. It draws
// nothing and imports nothing from core. The comps Cart widgets (CartList,
// CartLine, CartSummary, CartButton, PromoCodeField) draw it, and
// hooks.UseCart connects a Store to a screen.
//
//	store := cart.NewStore(cart.Config{
//	    Currency:    "usd",
//	    MaxQuantity: 10,
//	    Rules: []cart.Rule{
//	        cart.PercentOff("SPRING10", "Spring sale", cart.Percent(10)),
//	        cart.FlatShipping("Shipping", 599, 5000), // free from $50
//	        cart.Tax("Tax", cart.Percent(8.25)),
//	    },
//	})
//	store.Add(cart.Item{ID: "kettle/black", Label: "Pour-over kettle", UnitAmount: 4500})
//
//	// in any screen, every pass:
//	q := hooks.UseCart(ctx, store)
//	comps.CartButton{Count: q.Count(), OnTap: openCart}
//
// # How the pieces fit
//
//	tap ──▶ Store.SetQuantity(id, n) ──▶ Dispatch(SetQuantity{…})
//	                                        │
//	                   Config.Reduce(cart, action) ──▶ new Cart value
//	                                        │
//	             ┌──────────────────────────┼─────────────────────┐
//	             ▼                          ▼                     ▼
//	      Persister.Save            subscribers notified    (old Cart value
//	      (write-through)           UseCart → RequestRender  left untouched)
//	                                        │
//	render ──▶ Store.Quote() ──▶ Config.Price(cart) ──▶ Quote{Subtotal, Adjustments, Total}
//
// The cart is stored unpriced; the Quote is derived on read. Rules can then
// change (a code expires, a shipping threshold moves) without a stored total
// going stale, and there is one place, Price, that every total comes from.
//
// # Three layers, each usable alone
//
//   - Config.Reduce is a pure function of (Cart, Action). A screen-local cart
//     with no store at all is hooks.UseReducer(ctx, cfg.Reduce, cart.Cart{}).
//   - Config.Price is a pure function of a Cart. Tests and servers can price
//     a cart without any UI.
//   - Store adds the two things a reducer cannot: one cart shared by every
//     screen (a badge in the app bar and the cart screen stay in step), and
//     persistence across launches (see Persister and package cartdb).
//
// # Amounts: int64 minor units, one currency per cart
//
// Every amount is an int64 in the currency's minor unit, the convention
// comps.StripeCheckout and Stripe's API already use: 4500 is $45.00, and 4500
// yen is ¥4,500. Integer cents never round a sum; a float64 cannot represent
// 0.10 exactly. A cart holds one currency (Config.Currency); mixing them in
// one total has no meaning.
//
// # The price shown is not the price charged
//
// Everything here runs on the reader's device, where every value can be
// changed by whoever holds it. The Quote is for display. The server that
// creates the payment must look up each Item.ID's price itself and apply its
// own rules; send it IDs, quantities and codes, never amounts.
//
// # Snapshots are values
//
// A Cart handed out by a Store, or produced by Reduce, is never modified
// afterwards: every action builds new slices. Render code may read a
// snapshot from any goroutine without a lock. The flip side is that a
// caller must not modify one either; treat Items, Codes and Meta as
// read-only.
package cart

import "slices"

// Item is one line of a cart: a product (or one variant of it) and how many.
type Item struct {
	// ID identifies the line. Adding an Item whose ID is already in the cart
	// adds to that line rather than starting a second one, so a product sold
	// in variants gives each variant its own ID ("kettle/black").
	ID string `json:"id"`

	// Label names the item ("Pour-over kettle").
	Label string `json:"label"`

	// Detail is a second, smaller line ("Matte black").
	Detail string `json:"detail,omitempty"`

	// ImageURL is the thumbnail CartLine draws. Empty draws none.
	ImageURL string `json:"image,omitempty"`

	// UnitAmount is the price of one, in minor units.
	UnitAmount int64 `json:"unit"`

	// Quantity is how many. Add treats zero or less as one, so an Item
	// written without it is one of the thing; a stored line is always at
	// least one, because a quantity below one removes the line.
	Quantity int `json:"qty"`

	// MaxQuantity caps the line (stock on hand, a per-order limit). Zero
	// takes Config.MaxQuantity when the item is added, so a stored line
	// carries the cap that applies to it and a widget can disable its "+"
	// at that cap without being handed the Config. Zero after that is
	// unbounded.
	MaxQuantity int `json:"max,omitempty"`

	// Meta carries the caller's own data (a product ID, the chosen options)
	// through the cart untouched. Strings only, so a cart always survives a
	// JSON round trip through a Persister unchanged.
	Meta map[string]string `json:"meta,omitempty"`
}

// Amount is the line's total: Quantity × UnitAmount.
func (it Item) Amount() int64 { return int64(it.Quantity) * it.UnitAmount }

// Cart is the cart's contents: its lines in the order they were added, and
// the promo codes the reader applied. It holds no prices beyond each line's
// UnitAmount; see Config.Price.
type Cart struct {
	Items []Item   `json:"items"`
	Codes []string `json:"codes,omitempty"`
}

// Count is the number of things in the cart, the figure a cart badge shows:
// two kettles and a filter pack is three.
func (c Cart) Count() int {
	n := 0
	for _, it := range c.Items {
		n += it.Quantity
	}
	return n
}

// IsEmpty reports whether the cart has no lines. Applied codes alone do not
// make a cart non-empty.
func (c Cart) IsEmpty() bool { return len(c.Items) == 0 }

// Subtotal is the sum of the lines, before any rule.
func (c Cart) Subtotal() int64 {
	var sum int64
	for _, it := range c.Items {
		sum += it.Amount()
	}
	return sum
}

// Find returns the line with id, and whether there is one.
func (c Cart) Find(id string) (Item, bool) {
	if i := c.index(id); i >= 0 {
		return c.Items[i], true
	}
	return Item{}, false
}

// HasCode reports whether code (in any case, with any surrounding spaces) is
// applied.
func (c Cart) HasCode(code string) bool {
	return slices.Contains(c.Codes, NormalizeCode(code))
}

// index is the position of the line with id, or -1. A linear scan: a cart
// is tens of lines, and keeping a map in step with the slice would cost
// more than it saves and break the "a Cart is plain data" property that
// lets it round-trip through JSON.
func (c Cart) index(id string) int {
	for i, it := range c.Items {
		if it.ID == id {
			return i
		}
	}
	return -1
}
