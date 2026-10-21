# Package comps — Cart & checkout

```go
import "github.com/rohanthewiz/grmob/comps"
```

Cart lines and lists, the cart summary, the cart button with its count, the promo code field, and the Stripe checkout summary.

One of 8 topic pages of [package comps](comps.md), which has the package overview and an index of every topic. This page documents the declarations in `comps/cart_line.go`, `comps/cart_list.go`, `comps/cart_summary.go`, `comps/cart_button.go`, `comps/promo_code_field.go`, `comps/stripe_checkout.go`.

## Index

- [Constants](#constants) — `ConcernCartButtonInert`, `ConcernCartListDuplicateID`, `ConcernPromoCodeFieldInert`, `ConcernStripeCheckoutInert`, `ConcernStripeCheckoutInsecureURL`
- [`func FormatMoney`](#func-formatmoney)
- [`type CartButton`](#type-cartbutton)
    - [`func (CartButton) Render`](#func-cartbutton-render)
- [`type CartLine`](#type-cartline)
    - [`func (CartLine) Render`](#func-cartline-render)
- [`type CartList`](#type-cartlist)
    - [`func (CartList) Render`](#func-cartlist-render)
- [`type CartSummary`](#type-cartsummary)
    - [`func (CartSummary) Render`](#func-cartsummary-render)
- [`type CheckoutAdjustment`](#type-checkoutadjustment)
- [`type CheckoutItem`](#type-checkoutitem)
- [`type PromoCodeField`](#type-promocodefield)
    - [`func (PromoCodeField) Render`](#func-promocodefield-render)
- [`type StripeCheckout`](#type-stripecheckout)
    - [`func StripeCheckoutFromCart`](#func-stripecheckoutfromcart)
    - [`func (StripeCheckout) Render`](#func-stripecheckout-render)
    - [`func (StripeCheckout) Total`](#func-stripecheckout-total)

## Constants

ConcernCartButtonInert is raised, in debug builds only, when a CartButton has no OnTap. It is announced as a button and counts the cart, and a tap on it goes nowhere: the one control every shopping screen shares, silently broken on all of them.

```go
const ConcernCartButtonInert = "cart-button-inert"
```

<small>[comps/cart_button.go:13](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_button.go#L13)</small>

ConcernCartListDuplicateID is raised, in debug builds only, when two of a CartList's items share an ID. Lines are keyed by ID, so the second would collide with the first in the reconciler, and every cart.Store action addresses a line by ID, so a tap on either line's "+" would change the first. cart.Add never produces this; a hand-built slice can.

```go
const ConcernCartListDuplicateID = "cart-list-duplicate-id"
```

<small>[comps/cart_list.go:13](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_list.go#L13)</small>

ConcernPromoCodeFieldInert is raised, in debug builds only, when a PromoCodeField has no OnApply. Apply clears nothing and applies nothing, and the reader is left believing their discount is on the order.

```go
const ConcernPromoCodeFieldInert = "promo-code-field-inert"
```

<small>[comps/promo_code_field.go:12](https://github.com/rohanthewiz/grmob/blob/master/comps/promo_code_field.go#L12)</small>

ConcernStripeCheckoutInert is raised, in debug builds only, when a StripeCheckout has neither OnPay nor CheckoutURL and is not Disabled. Its Pay button looks ready and does nothing, on the one screen where a reader is most sure what a button should do.

```go
const ConcernStripeCheckoutInert = "stripe-checkout-inert"
```

<small>[comps/stripe_checkout.go:14](https://github.com/rohanthewiz/grmob/blob/master/comps/stripe_checkout.go#L14)</small>

ConcernStripeCheckoutInsecureURL is raised, in debug builds only, when CheckoutURL is not an https:// address. Stripe's Checkout Session and Payment Link URLs always are; anything else is a typo or a URL that did not come from Stripe, and the widget is about to hand it to the browser with the word "Pay" on it.

```go
const ConcernStripeCheckoutInsecureURL = "stripe-checkout-insecure-url"
```

<small>[comps/stripe_checkout.go:21](https://github.com/rohanthewiz/grmob/blob/master/comps/stripe_checkout.go#L21)</small>

## Functions

### func FormatMoney

```go
func FormatMoney(amount int64, currency string) string
```

FormatMoney writes amount, in currency's minor unit, the way an English receipt does: "$1,234.50", "¥4,500", "-€5.00", "SEK 99.00".

The decimals follow Stripe's rules for the currency (two, or none for the yen and fifteen others, or three for the Gulf dinars), so an amount that round-trips through Stripe's API is shown as Stripe will charge it. It is English formatting throughout; StripeCheckout.Format takes a locale-aware replacement.

<small>[comps/stripe_checkout.go:374](https://github.com/rohanthewiz/grmob/blob/master/comps/stripe_checkout.go#L374)</small>

## Types

### type CartButton

```go
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
```

CartButton is the cart glyph with a count beside it, for an app bar: the way to the cart from every screen, saying how full it is.

	q := hooks.UseCart(ctx, store)
	comps.AppBar{Title: "Shop", Actions: []core.View{
	    comps.CartButton{Count: q.Count(), OnTap: openCart},
	}}

	┌ Row  role=button  named "Cart, 3 items" ┐
	│   🛒  ( 3 )                              │  glyph hidden; Badge
	└──────────────────────────────────────────┘

#### The count is drawn beside the glyph, not over its corner

A badge pinned to an icon's corner needs an overlay positioned in absolute terms, which each host lays out differently and which a long count ("99+") pushes off the icon. Beside it, the count is ordinary row layout on every host, and the button's tap target grows to hold it. With nothing in the cart no badge is drawn: a "0" pill reads as a notification.

#### Accessibility

The Row is the button, named as one phrase, "Cart, 3 items", so a reader hears the destination and its state together. The glyph and the badge are inside a button and so presentational. The name changes as the count does; a reader on the button hears the new count when they next focus it, and is not interrupted by every add elsewhere on the screen.

#### No hooks

May be rendered conditionally.

#### Theme roles read

	Glyph    Typography.Title's size
	Badge    as Badge, with Variant
	Insets   Spacing.SM horizontal, Spacing.XS vertical, so the target is
	         larger than the glyph

<small>[comps/cart_button.go:53](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_button.go#L53)</small>

#### func (CartButton) Render

```go
func (b CartButton) Render(ctx *core.Context) *core.Node
```

Render draws the button. It takes no hook slot.

<small>[comps/cart_button.go:82](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_button.go#L82)</small>

### type CartLine

```go
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
```

CartLine is one line of a shopping cart: a thumbnail, the item's name and price, a Stepper for its quantity and a Remove button.

	comps.CartLine{
	    Item:       it,                      // a cart.Item
	    Currency:   q.Currency,
	    OnQuantity: func(n int) { store.SetQuantity(it.ID, n) },
	    OnRemove:   func() { store.Remove(it.ID) },
	}

	┌ Row  (swipe toward the leading edge removes, when SwipeToRemove) ─┐
	│ ┌────┐ ┌ Column ──────────────────────────────────────────────┐  │
	│ │img │ │ ┌ Column  named "Pour-over kettle, Matte black,     ┐ │  │
	│ │    │ │ │          2 × $45.00, $90.00"                      │ │  │
	│ └────┘ │ │ Pour-over kettle                        $90.00    │ │  │
	│ hidden │ │ Matte black                                       │ │  │
	│        │ │ 2 × $45.00                                        │ │  │
	│        │ └───────────────────────────────────────────────────┘ │  │
	│        │ [ − ]  2  [ + ]                            Remove     │  │
	│        └──────────────────────────────────────────────────────┘  │
	└───────────────────────────────────────────────────────────────────┘

Usually a CartList draws these; a CartLine on its own is for a screen that lays its lines out differently (a grid, a review step with extra columns).

#### Controlled, and the controls follow the callbacks

The quantity is the caller's (normally a cart.Store's). The Stepper is drawn only when OnQuantity is set and the item can be more than one, and the Remove button only when OnRemove is set, so the zero CartLine{Item: it} is the read-only line an order-review screen wants, with no inert controls to report.

The Stepper runs from one to the item's MaxQuantity (cart.Add fills that from the cart's Config), or to 99 when there is none. Its "−" stops at one rather than removing the line: a removal deserves a button that says so, not a fourth tap on a minus sign.

#### Swipe to remove is opt-in

SwipeToRemove removes the line on a swipe toward the leading edge (left in a left-to-right layout), the gesture iOS's own lists use. It is off by default because nothing on screen hints at it and nothing confirms it: a list that deletes on a stray sideways drag needs an undo (a Snackbar that Replaces the cart from before), which is the caller's to provide. The Remove button stays either way: a screen-reader user cannot swipe a row.

#### Accessibility

The item's text is one labelled Column, "Pour-over kettle, Matte black, 2 × $45.00, $90.00", so a reader hears a product and its price together. The controls stay outside it, so they remain reachable, and their names carry the item: the Stepper is "Quantity, Pour-over kettle" and the button "Remove Pour-over kettle". A cart of three lines would otherwise read as three identical "Remove" buttons. The thumbnail is hidden; the label says what it shows.

#### No hooks

May be rendered conditionally.

#### Theme roles read

	Label, amount   Typography.Body, Colors.TextPrimary; amount bold
	Detail, each    Typography.Caption, Colors.TextSecondary
	Thumbnail       Colors.Surface behind the image, Spacing.SM corner radius
	Stepper         as Stepper
	Remove          a ghost Button, Colors.Error ink

<small>[comps/cart_line.go:90](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_line.go#L90)</small>

#### func (CartLine) Render

```go
func (l CartLine) Render(ctx *core.Context) *core.Node
```

Render draws the line. It takes no hook slot.

<small>[comps/cart_line.go:138](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_line.go#L138)</small>

### type CartList

```go
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
```

CartList is a shopping cart's lines: a CartLine per item with a separator between them, or an empty state when there are none.

	q := hooks.UseCart(ctx, store)
	comps.CartList{
	    Items:      q.Cart.Items,
	    Currency:   q.Currency,
	    OnQuantity: store.SetQuantity,
	    OnRemove:   store.Remove,
	}

	┌ Column ────────────────────────────────────────┐
	│  CartLine  Pour-over kettle   [−] 1 [+]  $45   │  keyed "line:kettle"
	│  ───────────────────────────────────────────── │  keyed "sep:filters"
	│  CartLine  Filters (100)      [−] 2 [+]  $13   │  keyed "line:filters"
	└────────────────────────────────────────────────┘

	no items:  EmptyState  🛒  "Your cart is empty"   (or the Empty slot)

#### Controlled, with store-shaped callbacks

OnQuantity and OnRemove take the item's ID, the shapes cart.Store's SetQuantity and Remove already have, so the usual wiring is the two method values above. Leave both nil (or set ReadOnly) for an order-review list: the lines draw no controls at all.

#### Configuring a line

The fields shared by every line (Currency, Format, SwipeToRemove, HideImages, the labels, Disabled) are passed through to each CartLine. Line replaces the CartLine altogether, for a row this widget does not draw (a gift message under each item, a subscription interval picker); the list keeps keying, separating and the empty state.

#### Keys

Lines are keyed by item ID, so a removal or a quantity change patches one line and leaves the others, and their Stepper focus, alone. Separators are keyed in their own namespace, by the ID of the line below them, so the separator above a removed line goes with it.

#### No hooks

No hooks of its own; may be rendered conditionally. A Line func that calls hooks makes the list a hook caller with a slot count that varies with the cart, so a Line must be hook-free too.

#### Theme roles read

	Gap     Spacing.SM between a line and a separator
	Lines   as CartLine; separators as Separator; empty as EmptyState

<small>[comps/cart_list.go:66](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_list.go#L66)</small>

#### func (CartList) Render

```go
func (l CartList) Render(ctx *core.Context) *core.Node
```

Render draws the lines, or the empty state. It takes no hook slot.

<small>[comps/cart_list.go:115](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_list.go#L115)</small>

### type CartSummary

```go
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
```

CartSummary is the totals of a priced cart: the subtotal, each rule's adjustment (discounts, shipping, tax) and the total, with a slot under them for the button that moves on.

	q := hooks.UseCart(ctx, store)
	comps.CartSummary{
	    Quote:  q,
	    Footer: comps.Button{Label: "Checkout", OnTap: checkout, FullWidth: true},
	}

	┌ Card ─────────────────────────────────────────┐
	│  Order summary          (Title, when set)     │  heading 2
	│  Subtotal                            $58.00   │  Row, named
	│  3 items                                      │  "Subtotal, 3 items, $58.00"
	│  Spring sale                         -$5.80   │
	│  SPRING10                                     │  the code that gave it
	│  Shipping                              Free   │  a zero adjustment
	│  Tax                                  $4.31   │
	│  ──────────────────────────────────────────── │
	│  Total                               $56.51   │  Row, named "Total, $56.51"
	│  Footer                                       │  the caller's slot
	└───────────────────────────────────────────────┘

#### Every number comes from the Quote

The widget adds nothing up. cart.Config.Price produced the subtotal, the adjustments and the total together, so the figures drawn here cannot disagree with each other or with the Quote the caller sends to its server. (The server still prices the order itself; see the cart package doc.)

#### Paying with Stripe

StripeCheckoutFromCart turns the same Quote into a comps.StripeCheckout, itemised, for the screen that hands the payment to Stripe:

	co := comps.StripeCheckoutFromCart(q)
	co.OnPay, co.Pending = startCheckout, redirecting.Get()

#### Accessibility

Each row is named as one phrase, the label, its detail and its amount ("Spring sale, SPRING10, -$5.80"), the line shape StripeCheckout uses, which it shares.

#### No hooks

May be rendered conditionally. Footer is the caller's view and follows its own rules.

#### Theme roles read

	Card     the theme's Card base, Spacing.SM between rows
	Title    Typography.Subtitle, bold
	Rows     Typography.Body, Colors.TextPrimary; detail Caption, TextSecondary
	Total    Typography.Subtitle, bold

<small>[comps/cart_summary.go:65](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_summary.go#L65)</small>

#### func (CartSummary) Render

```go
func (s CartSummary) Render(ctx *core.Context) *core.Node
```

Render draws the totals. It takes no hook slot.

<small>[comps/cart_summary.go:99](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_summary.go#L99)</small>

### type CheckoutAdjustment

```go
type CheckoutAdjustment struct {
	Label  string
	Amount int64
}
```

CheckoutAdjustment is a line after the items that changes the total: shipping, tax, a discount (a negative Amount). In minor units.

<small>[comps/stripe_checkout.go:52](https://github.com/rohanthewiz/grmob/blob/master/comps/stripe_checkout.go#L52)</small>

### type CheckoutItem

```go
type CheckoutItem struct {
	// Label names the item ("Pour-over kettle").
	Label string

	// Detail is a second, smaller line ("Matte black"). Empty draws the
	// quantity and unit price there when Quantity is above one.
	Detail string

	// Quantity multiplies UnitAmount. Zero or less counts as one, so a line
	// written without it is one of the thing.
	Quantity int

	// UnitAmount is the price of one, in the currency's minor unit (cents),
	// the way Stripe's API states every amount.
	UnitAmount int64
}
```

CheckoutItem is one line of a StripeCheckout's order.

<small>[comps/stripe_checkout.go:24](https://github.com/rohanthewiz/grmob/blob/master/comps/stripe_checkout.go#L24)</small>

### type PromoCodeField

```go
type PromoCodeField struct {
	// OnApply is asked to apply a code (trimmed, never empty) and returns
	// nil, or the reason it was refused. Nil reports
	// ConcernPromoCodeFieldInert.
	OnApply func(code string) error

	// Applied are the codes already on the order, drawn as removable pills.
	Applied []string

	// OnRemove takes an applied code off. Nil draws the pills without ✕.
	OnRemove func(code string)

	// Label names the input for assistive technology. Empty is "Promo code".
	Label string

	// Placeholder is drawn in the empty input. Empty is Label's value.
	Placeholder string

	// ApplyLabel is the button's text. Empty is "Apply".
	ApplyLabel string

	// RemoveLabel prefixes each ✕'s spoken name. Empty is "Remove".
	RemoveLabel string

	// Disabled disables the input, the button and every ✕.
	Disabled bool

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}
```

PromoCodeField is where a reader types a promo code: an input and an Apply button, the reason a code was refused, and the codes already applied, each with its own ✕.

	comps.PromoCodeField{
	    OnApply:  store.ApplyCode,    // func(string) error
	    Applied:  q.Cart.Codes,
	    OnRemove: store.RemoveCode,
	}

	┌ Column ──────────────────────────────────────────────┐
	│ ┌ Row ───────────────────────────────────────────────┐ │
	│ │ [ Promo code                         ]  [ Apply ]  │ │  return applies too
	│ └────────────────────────────────────────────────────┘ │
	│  That code isn't valid                                 │  RoleAlert, when refused
	│ ┌ Row  role=list  wrap ──────────────────────────────┐ │
	│ │ ( SPRING10  ✕ ) ( FREESHIP  ✕ )                    │ │  one listitem each
	│ └────────────────────────────────────────────────────┘ │
	└──────────────────────────────────────────────────────┘

#### OnApply answers with an error

Whether a code is good is not this widget's to decide: it may need the cart (a minimum order) or a server. OnApply returns nil to accept, and the draft clears; or an error, whose text is drawn under the input, and the draft stays so the reader can fix a typo rather than retype the code. cart.Store.ApplyCode has exactly this shape. Typing again clears the error, since it described a code that is no longer in the field.

OnApply runs on the event path and its answer is drawn on the next pass, so it should answer promptly. A server check is better made by accepting the code (Store.Dispatch(cart.ApplyCode{…})) and pricing it once the server agrees, or by an AcceptCode that consults a list fetched earlier.

#### The draft and the error are the widget's

TagInput's test: no application wants a half-typed code, and the refusal is about the last attempt, not about the cart. Both live in hooks here, so the hook rules apply: render a PromoCodeField unconditionally, in a stable position.

#### Accessibility

The input is named by Label. A refusal is a RoleAlert, so a reader who pressed Apply hears why at once. Applied codes are a list of pills whose ✕ is a separate button named "Remove SPRING10", TagInput's shape: a chip with a button inside it is a button inside a button.

#### Theme roles read

	Input    the theme's input
	Apply    an outlined Button
	Error    Typography.Caption, Colors.Error
	Pills    as TagInput's: Colors.Surface fill, ColorPalette.BorderColor hairline

<small>[comps/promo_code_field.go:68](https://github.com/rohanthewiz/grmob/blob/master/comps/promo_code_field.go#L68)</small>

#### func (PromoCodeField) Render

```go
func (p PromoCodeField) Render(ctx *core.Context) *core.Node
```

Render draws the field. It takes two hooks, the draft and the last refusal, before any branch.

<small>[comps/promo_code_field.go:101](https://github.com/rohanthewiz/grmob/blob/master/comps/promo_code_field.go#L101)</small>

### type StripeCheckout

```go
type StripeCheckout struct {
	// Title heads the card. Empty is "Order summary".
	Title string

	// HeadingLevel is the Title's tier; zero is 2, as for a Card.
	HeadingLevel int

	// Items are the order's lines, in order.
	Items []CheckoutItem

	// Adjustments are drawn after the items and added to the total.
	Adjustments []CheckoutAdjustment

	// Currency is the ISO 4217 code, either case ("usd", "EUR"). Empty is USD.
	Currency string

	// CheckoutURL is a Payment Link or Checkout Session url, opened when Pay
	// is tapped and OnPay is nil.
	CheckoutURL string

	// OnPay handles the tap instead of opening CheckoutURL.
	OnPay func()

	// Pending disables the button and relabels it PendingLabel: the app is
	// fetching a Session, or the browser is opening.
	Pending bool

	// PayLabel replaces "Pay <total>".
	PayLabel string

	// PendingLabel replaces "Redirecting to Stripe…".
	PendingLabel string

	// Disabled disables the button: the cart is empty, a form above is
	// incomplete.
	Disabled bool

	// Error is drawn above the button: the last attempt failed. A reader is
	// told at once (RoleAlert), since they are waiting on the outcome.
	Error string

	// Note replaces the line under the button. HideNote leaves it out.
	Note     string
	HideNote bool

	// Variant colours the Pay button. Zero is the theme's primary.
	Variant Variant

	// Format renders an amount. Nil is FormatMoney.
	Format func(amount int64, currency string) string

	// Style is applied to the card after the widget's own props.
	Style []core.StyleProp
}
```

StripeCheckout is the order summary and Pay button in front of a Stripe payment: the items, any shipping, tax or discount lines, the total, and one button that hands the payment to Stripe.

	comps.StripeCheckout{
	    Items: []comps.CheckoutItem{
	        {Label: "Pour-over kettle", Detail: "Matte black", UnitAmount: 4500},
	        {Label: "Filters (100)", Quantity: 2, UnitAmount: 650},
	    },
	    Adjustments: []comps.CheckoutAdjustment{{Label: "Shipping", Amount: 500}},
	    Currency:    "usd",
	    OnPay:       startCheckout, // asks your server for a Session, then core.OpenURL
	    Pending:     redirecting.Get(),
	}

	┌ Card ─────────────────────────────────────────┐
	│  Order summary                    heading 2   │
	│  Pour-over kettle                    $45.00   │  Row, named
	│  Matte black                                  │  "Pour-over kettle, $45.00"
	│  Filters (100)                       $13.00   │
	│  2 × $6.50                                    │
	│  Shipping                             $5.00   │
	│  ──────────────────────────────────────────── │  Separator
	│  Total                               $63.00   │  Row, named "Total, $63.00"
	│  (Error, when set)                            │  Text, RoleAlert
	│  ┌──────────────── Pay $63.00 ─────────────┐  │  Button, full width
	│  └─────────────────────────────────────────┘  │
	│  🔒 Payments are processed securely by Stripe │  Caption
	└───────────────────────────────────────────────┘

#### What it does not do: take a card

There is no card field here, and there will not be. A card number typed into an app's own text field passes through the app's memory, its logs and its crash reports, and that puts the app in PCI DSS scope. Stripe's answer is to collect the card on a page it hosts, which is what Checkout and Payment Links are. So the widget's job ends at the hand-off: it shows what is being bought and sends the reader to Stripe to pay for it.

Two ways to get there:

  - CheckoutURL, a Payment Link ([https://buy.stripe.com/](https://buy.stripe.com/)…) or a Checkout Session's url. A tap opens it with core.OpenURL: the browser on every platform. Right for a fixed product, where the link can be made once in the Stripe dashboard.
  - OnPay, which wins when set. Creating a Checkout Session for a cart needs the secret key, and the secret key lives on a server, never in an app binary, where anyone can read it out. So OnPay is where the app calls its own server, gets the Session's url back, and opens it. Set Pending while that round trip runs.

When the reader comes back (Stripe redirects to the success\_url or cancel\_url the Session was created with; on a phone, a deep link), the app asks its server whether the payment went through. Nothing this widget saw says so.

#### Amounts

Every amount is an int64 in the currency's minor unit, as Stripe's API takes them: 4500 is $45.00, and 4500 yen is ¥4,500, because the yen has no minor unit. FormatMoney knows which currencies Stripe treats as zero- and three-decimal. The total is computed here from the lines, so the button can never disagree with the list above it; it is still the Session, not this number, that decides what is charged.

#### Accessibility

Each line is a row named as one phrase, "Filters (100), 2 × $6.50, $13.00", so a reader hears an item and its price together rather than a column of names and then a column of prices. The Pay button says the total in its label. While Pending the button is disabled and reads "Redirecting to Stripe…", which is what the reader is waiting for.

#### No hooks

Everything is the caller's, so StripeCheckout may be rendered conditionally.

#### Theme roles read

	Card         the theme's Card base, Spacing.SM between the parts
	Title        Typography.Subtitle, bold
	Line         Typography.Body, Colors.TextPrimary; detail Caption, TextSecondary
	Total        Typography.Subtitle, bold
	Error        Typography.Body, Colors.Error
	Button       as Button, with Variant
	Note         Typography.Caption, Colors.TextSecondary

<small>[comps/stripe_checkout.go:144](https://github.com/rohanthewiz/grmob/blob/master/comps/stripe_checkout.go#L144)</small>

#### func StripeCheckoutFromCart

```go
func StripeCheckoutFromCart(q cart.Quote) StripeCheckout
```

StripeCheckoutFromCart is a StripeCheckout filled from q: one CheckoutItem per line, one CheckoutAdjustment per rule's line, the currency, and Disabled when the cart is empty. The caller adds OnPay (or CheckoutURL) and Pending, and anything else it wants to change.

The items are listed, as StripeCheckout always lists them, and the total it computes from them equals q.Total, because Price computed q.Total from the same lines.

<small>[comps/cart_summary.go:150](https://github.com/rohanthewiz/grmob/blob/master/comps/cart_summary.go#L150)</small>

#### func (StripeCheckout) Render

```go
func (s StripeCheckout) Render(ctx *core.Context) *core.Node
```

Render draws the summary. It takes no hook slot.

<small>[comps/stripe_checkout.go:212](https://github.com/rohanthewiz/grmob/blob/master/comps/stripe_checkout.go#L212)</small>

#### func (StripeCheckout) Total

```go
func (s StripeCheckout) Total() int64
```

Total is the sum of the items and adjustments, in minor units.

<small>[comps/stripe_checkout.go:200](https://github.com/rohanthewiz/grmob/blob/master/comps/stripe_checkout.go#L200)</small>

