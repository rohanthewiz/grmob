package comps

import (
	"strconv"

	"github.com/rohanthewiz/grmob/cart"
	"github.com/rohanthewiz/grmob/core"
)

// CartSummary is the totals of a priced cart: the subtotal, each rule's
// adjustment (discounts, shipping, tax) and the total, with a slot under them
// for the button that moves on.
//
//	q := hooks.UseCart(ctx, store)
//	comps.CartSummary{
//	    Quote:  q,
//	    Footer: comps.Button{Label: "Checkout", OnTap: checkout, FullWidth: true},
//	}
//
//	┌ Card ─────────────────────────────────────────┐
//	│  Order summary          (Title, when set)     │  heading 2
//	│  Subtotal                            $58.00   │  Row, named
//	│  3 items                                      │  "Subtotal, 3 items, $58.00"
//	│  Spring sale                         -$5.80   │
//	│  SPRING10                                     │  the code that gave it
//	│  Shipping                              Free   │  a zero adjustment
//	│  Tax                                  $4.31   │
//	│  ──────────────────────────────────────────── │
//	│  Total                               $56.51   │  Row, named "Total, $56.51"
//	│  Footer                                       │  the caller's slot
//	└───────────────────────────────────────────────┘
//
// # Every number comes from the Quote
//
// The widget adds nothing up. cart.Config.Price produced the subtotal, the
// adjustments and the total together, so the figures drawn here cannot
// disagree with each other or with the Quote the caller sends to its server.
// (The server still prices the order itself; see the cart package doc.)
//
// # Paying with Stripe
//
// StripeCheckoutFromCart turns the same Quote into a comps.StripeCheckout,
// itemised, for the screen that hands the payment to Stripe:
//
//	co := comps.StripeCheckoutFromCart(q)
//	co.OnPay, co.Pending = startCheckout, redirecting.Get()
//
// # Accessibility
//
// Each row is named as one phrase, the label, its detail and its amount
// ("Spring sale, SPRING10, -$5.80"), the line shape StripeCheckout uses,
// which it shares.
//
// # No hooks
//
// May be rendered conditionally. Footer is the caller's view and follows its
// own rules.
//
// # Theme roles read
//
//	Card     the theme's Card base, Spacing.SM between rows
//	Title    Typography.Subtitle, bold
//	Rows     Typography.Body, Colors.TextPrimary; detail Caption, TextSecondary
//	Total    Typography.Subtitle, bold
type CartSummary struct {
	// Quote is the priced cart: normally hooks.UseCart's result.
	Quote cart.Quote

	// Title heads the card. Empty draws none, since a summary usually sits
	// under the cart's own heading.
	Title string

	// HeadingLevel is the Title's tier; zero is 2, as for a Card.
	HeadingLevel int

	// SubtotalLabel and TotalLabel name the two fixed rows. Empty is
	// "Subtotal" and "Total".
	SubtotalLabel, TotalLabel string

	// FreeLabel replaces the amount of a zero adjustment. Empty is "Free".
	FreeLabel string

	// CountLabel phrases the item count under the subtotal ("3 items"). Nil
	// is English; HideCount leaves the count out.
	CountLabel func(n int) string
	HideCount  bool

	// Format renders an amount. Nil is FormatMoney.
	Format func(amount int64, currency string) string

	// Footer is drawn under the total: a checkout button, a note.
	Footer core.View

	// Style is applied to the card after the widget's own props.
	Style []core.StyleProp
}

// Render draws the totals. It takes no hook slot.
func (s CartSummary) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	q := s.Quote
	format, currency := moneyFormat(s.Format, q.Currency)

	items := make([]core.PropsAndChildren, 0, len(s.Style)+len(q.Adjustments)+6)
	items = append(items, core.Gap(float64(t.Spacing.SM)))
	items = append(items, asProps(s.Style)...)

	if s.Title != "" {
		title := []core.StyleProp{core.UseStyle(t.Typography.Subtitle), core.FontWeight(core.Bold)}
		title = append(title, headingProps(s.HeadingLevel, headingLevelSection)...)
		items = append(items, core.Text(s.Title, title...))
	}

	count := ""
	if !s.HideCount {
		count = itemCount(q.Count(), s.CountLabel)
	}
	items = append(items, checkoutLine(t, orDefault(s.SubtotalLabel, "Subtotal"), count,
		format(q.Subtotal, currency), false))

	for i, a := range q.Adjustments {
		amount := format(a.Amount, currency)
		if a.Amount == 0 {
			amount = orDefault(s.FreeLabel, "Free")
		}
		// Keyed by position and label, as StripeCheckout keys its lines: a
		// rule's line appears and disappears (a code applied, shipping
		// crossing its threshold), and the rows after it should patch
		// rather than shift.
		items = append(items, core.Keyed("adj:"+strconv.Itoa(i)+":"+a.Label,
			checkoutLine(t, a.Label, a.Code, amount, false)))
	}

	items = append(items, Separator{},
		checkoutLine(t, orDefault(s.TotalLabel, "Total"), "", format(q.Total, currency), true))
	if s.Footer != nil {
		items = append(items, s.Footer)
	}
	return core.Card(items...).Render(ctx)
}

// StripeCheckoutFromCart is a StripeCheckout filled from q: one CheckoutItem
// per line, one CheckoutAdjustment per rule's line, the currency, and
// Disabled when the cart is empty. The caller adds OnPay (or CheckoutURL)
// and Pending, and anything else it wants to change.
//
// The items are listed, as StripeCheckout always lists them, and the total
// it computes from them equals q.Total, because Price computed q.Total from
// the same lines.
func StripeCheckoutFromCart(q cart.Quote) StripeCheckout {
	items := make([]CheckoutItem, 0, len(q.Cart.Items))
	for _, it := range q.Cart.Items {
		items = append(items, CheckoutItem{
			Label:      it.Label,
			Detail:     it.Detail,
			Quantity:   it.Quantity,
			UnitAmount: it.UnitAmount,
		})
	}
	adjustments := make([]CheckoutAdjustment, 0, len(q.Adjustments))
	for _, a := range q.Adjustments {
		adjustments = append(adjustments, CheckoutAdjustment{Label: a.Label, Amount: a.Amount})
	}
	return StripeCheckout{
		Items:       items,
		Adjustments: adjustments,
		Currency:    q.Currency,
		Disabled:    q.Cart.IsEmpty(),
	}
}
