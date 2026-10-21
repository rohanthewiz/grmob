# Package cart

```go
import "github.com/rohanthewiz/grmob/cart"
```

Package cart is the model behind a shopping cart: its lines, the rules that price them, and a Store that every screen of an app can share. It draws nothing and imports nothing from core. The comps Cart widgets (CartList, CartLine, CartSummary, CartButton, PromoCodeField) draw it, and hooks.UseCart connects a Store to a screen.

	store := cart.NewStore(cart.Config{
	    Currency:    "usd",
	    MaxQuantity: 10,
	    Rules: []cart.Rule{
	        cart.PercentOff("SPRING10", "Spring sale", cart.Percent(10)),
	        cart.FlatShipping("Shipping", 599, 5000), // free from $50
	        cart.Tax("Tax", cart.Percent(8.25)),
	    },
	})
	store.Add(cart.Item{ID: "kettle/black", Label: "Pour-over kettle", UnitAmount: 4500})

	// in any screen, every pass:
	q := hooks.UseCart(ctx, store)
	comps.CartButton{Count: q.Count(), OnTap: openCart}

## How the pieces fit

	tap ──▶ Store.SetQuantity(id, n) ──▶ Dispatch(SetQuantity{…})
	                                        │
	                   Config.Reduce(cart, action) ──▶ new Cart value
	                                        │
	             ┌──────────────────────────┼─────────────────────┐
	             ▼                          ▼                     ▼
	      Persister.Save            subscribers notified    (old Cart value
	      (write-through)           UseCart → RequestRender  left untouched)
	                                        │
	render ──▶ Store.Quote() ──▶ Config.Price(cart) ──▶ Quote{Subtotal, Adjustments, Total}

The cart is stored unpriced; the Quote is derived on read. Rules can then change (a code expires, a shipping threshold moves) without a stored total going stale, and there is one place, Price, that every total comes from.

## Three layers, each usable alone

  - Config.Reduce is a pure function of (Cart, Action). A screen-local cart with no store at all is hooks.UseReducer(ctx, cfg.Reduce, cart.Cart{}).
  - Config.Price is a pure function of a Cart. Tests and servers can price a cart without any UI.
  - Store adds the two things a reducer cannot: one cart shared by every screen (a badge in the app bar and the cart screen stay in step), and persistence across launches (see Persister and package cartdb).

## Amounts: int64 minor units, one currency per cart

Every amount is an int64 in the currency's minor unit, the convention comps.StripeCheckout and Stripe's API already use: 4500 is $45.00, and 4500 yen is ¥4,500. Integer cents never round a sum; a float64 cannot represent 0.10 exactly. A cart holds one currency (Config.Currency); mixing them in one total has no meaning.

## The price shown is not the price charged

Everything here runs on the reader's device, where every value can be changed by whoever holds it. The Quote is for display. The server that creates the payment must look up each Item.ID's price itself and apply its own rules; send it IDs, quantities and codes, never amounts.

## Snapshots are values

A Cart handed out by a Store, or produced by Reduce, is never modified afterwards: every action builds new slices. Render code may read a snapshot from any goroutine without a lock. The flip side is that a caller must not modify one either; treat Items, Codes and Meta as read-only.

## Index

- [Variables](#variables) — `ErrUnknownCode`
- [`func NormalizeCode`](#func-normalizecode)
- [`type Action`](#type-action)
- [`type Add`](#type-add)
- [`type Adjustment`](#type-adjustment)
- [`type ApplyCode`](#type-applycode)
- [`type Cart`](#type-cart)
    - [`func (Cart) Count`](#func-cart-count)
    - [`func (Cart) Find`](#func-cart-find)
    - [`func (Cart) HasCode`](#func-cart-hascode)
    - [`func (Cart) IsEmpty`](#func-cart-isempty)
    - [`func (Cart) Subtotal`](#func-cart-subtotal)
- [`type Clear`](#type-clear)
- [`type Config`](#type-config)
    - [`func (Config) Price`](#func-config-price)
    - [`func (Config) Reduce`](#func-config-reduce)
- [`type Item`](#type-item)
    - [`func (Item) Amount`](#func-item-amount)
- [`type Persister`](#type-persister)
- [`type Quote`](#type-quote)
    - [`func (Quote) Count`](#func-quote-count)
- [`type Rate`](#type-rate)
    - [`func Percent`](#func-percent)
    - [`func (Rate) Of`](#func-rate-of)
- [`type Remove`](#type-remove)
- [`type RemoveCode`](#type-removecode)
- [`type Replace`](#type-replace)
- [`type Rule`](#type-rule)
    - [`func AmountOff`](#func-amountoff)
    - [`func FlatShipping`](#func-flatshipping)
    - [`func PercentOff`](#func-percentoff)
    - [`func Tax`](#func-tax)
- [`type SetQuantity`](#type-setquantity)
- [`type Store`](#type-store)
    - [`func NewStore`](#func-newstore)
    - [`func (*Store) Add`](#func-store-add)
    - [`func (*Store) ApplyCode`](#func-store-applycode)
    - [`func (*Store) Cart`](#func-store-cart)
    - [`func (*Store) Clear`](#func-store-clear)
    - [`func (*Store) Config`](#func-store-config)
    - [`func (*Store) Dispatch`](#func-store-dispatch)
    - [`func (*Store) Quote`](#func-store-quote)
    - [`func (*Store) Remove`](#func-store-remove)
    - [`func (*Store) RemoveCode`](#func-store-removecode)
    - [`func (*Store) SetQuantity`](#func-store-setquantity)
    - [`func (*Store) Subscribe`](#func-store-subscribe)

## Variables

ErrUnknownCode is ApplyCode's answer for a code no Rule names, under the default Config.AcceptCode.

Its text is lower case, as Go error strings are, so it composes into a longer message. comps.PromoCodeField sentence-cases a refusal when it draws one ("That code isn't valid"), and its ErrorText takes a translation; the error itself stays comparable with errors.Is.

```go
var ErrUnknownCode = errors.New("that code isn't valid")
```

<small>[cart/reduce.go:16](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L16)</small>

## Functions

### func NormalizeCode

```go
func NormalizeCode(code string) string
```

NormalizeCode is a promo code as the cart stores and compares it: trimmed and upper-cased, so " spring10" typed on a phone matches the "SPRING10" a Rule names.

<small>[cart/reduce.go:74](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L74)</small>

## Types

### type Action

```go
type Action interface {
	// contains filtered or unexported methods
}
```

Action is a change to a cart, applied by Config.Reduce or Store.Dispatch. The set is closed (the method is unexported) so Reduce can promise to handle every Action there is.

<small>[cart/reduce.go:79](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L79)</small>

### type Add

```go
type Add struct{ Item Item }
```

Add puts Item in the cart. A line with the same ID gains Item's quantity and takes Item's other fields (the newest label and price win, which is what a re-added catalog item means), keeping its place in the order. The quantity is clamped to the line's cap.

<small>[cart/reduce.go:85](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L85)</small>

### type Adjustment

```go
type Adjustment struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`

	// Code is the promo code that produced the line, when one did, so a
	// widget can relate the discount to the code the reader typed.
	Code string `json:"code,omitempty"`
}
```

Adjustment is one line a Rule adds after the items. A negative Amount is a discount. An Amount of zero is drawn (CartSummary says "Free"), so a rule can show that shipping is free rather than leaving the reader to wonder.

<small>[cart/price.go:31](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L31)</small>

### type ApplyCode

```go
type ApplyCode struct{ Code string }
```

ApplyCode adds a promo code. Reduce applies it without asking AcceptCode, because a reducer cannot return an error; Store.ApplyCode is the checked path the widgets use.

<small>[cart/reduce.go:100](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L100)</small>

### type Cart

```go
type Cart struct {
	Items []Item   `json:"items"`
	Codes []string `json:"codes,omitempty"`
}
```

Cart is the cart's contents: its lines in the order they were added, and the promo codes the reader applied. It holds no prices beyond each line's UnitAmount; see Config.Price.

<small>[cart/cart.go:118](https://github.com/rohanthewiz/grmob/blob/master/cart/cart.go#L118)</small>

#### func (Cart) Count

```go
func (c Cart) Count() int
```

Count is the number of things in the cart, the figure a cart badge shows: two kettles and a filter pack is three.

<small>[cart/cart.go:125](https://github.com/rohanthewiz/grmob/blob/master/cart/cart.go#L125)</small>

#### func (Cart) Find

```go
func (c Cart) Find(id string) (Item, bool)
```

Find returns the line with id, and whether there is one.

<small>[cart/cart.go:147](https://github.com/rohanthewiz/grmob/blob/master/cart/cart.go#L147)</small>

#### func (Cart) HasCode

```go
func (c Cart) HasCode(code string) bool
```

HasCode reports whether code (in any case, with any surrounding spaces) is applied.

<small>[cart/cart.go:156](https://github.com/rohanthewiz/grmob/blob/master/cart/cart.go#L156)</small>

#### func (Cart) IsEmpty

```go
func (c Cart) IsEmpty() bool
```

IsEmpty reports whether the cart has no lines. Applied codes alone do not make a cart non-empty.

<small>[cart/cart.go:135](https://github.com/rohanthewiz/grmob/blob/master/cart/cart.go#L135)</small>

#### func (Cart) Subtotal

```go
func (c Cart) Subtotal() int64
```

Subtotal is the sum of the lines, before any rule.

<small>[cart/cart.go:138](https://github.com/rohanthewiz/grmob/blob/master/cart/cart.go#L138)</small>

### type Clear

```go
type Clear struct{}
```

Clear empties the cart, codes included: what happens after a successful payment.

<small>[cart/reduce.go:107](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L107)</small>

### type Config

```go
type Config struct {
	// Currency is the ISO 4217 code, either case. Empty is USD.
	Currency string

	// MaxQuantity is the per-line cap for an Item added with MaxQuantity
	// zero. Zero is unbounded.
	MaxQuantity int

	// Rules price the cart, in order; see Config.Price.
	Rules []Rule

	// AcceptCode decides whether Store.ApplyCode takes a code, and returns
	// the error to show the reader if not. It is handed the code already
	// normalized (NormalizeCode) and the cart as it stands, so it can say
	// "this code needs a $50 order". Nil accepts exactly the codes some Rule
	// names, and answers ErrUnknownCode otherwise.
	AcceptCode func(code string, c Cart) error

	// Persister keeps the cart between launches. NewStore loads from it and
	// every change is written through to it. Nil keeps the cart in memory.
	Persister Persister

	// OnError receives a Persister's errors. A failed save does not undo the
	// change on screen: losing persistence beats losing the reader's tap.
	// Nil logs them.
	OnError func(error)
}
```

Config is everything about a cart that is policy rather than contents: the currency, the quantity cap, the pricing rules, which codes are accepted, and where the cart is kept between launches. Every field's zero value is usable; Config{} is a USD cart with no cap, no rules and no persistence.

<small>[cart/reduce.go:22](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L22)</small>

#### func (Config) Price

```go
func (cfg Config) Price(c Cart) Quote
```

Price prices c: the subtotal, then each Rule in order, each seeing the running total. A Rule with a Code applies only when c has that code, and its adjustments are stamped with the code. An empty cart is priced with no rules at all, so it shows no shipping and no tax on nothing.

<small>[cart/price.go:69](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L69)</small>

#### func (Config) Reduce

```go
func (cfg Config) Reduce(c Cart, action Action) Cart
```

Reduce returns the cart after action. It never modifies c: every change builds new slices, which is what makes a Cart a value that render code can hold without a lock. It has the reducer shape hooks.UseReducer takes:

	c, dispatch := hooks.UseReducer(ctx, cfg.Reduce, cart.Cart{})

A nil action returns c unchanged.

<small>[cart/reduce.go:121](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L121)</small>

### type Item

```go
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
```

Item is one line of a cart: a product (or one variant of it) and how many.

<small>[cart/cart.go:76](https://github.com/rohanthewiz/grmob/blob/master/cart/cart.go#L76)</small>

#### func (Item) Amount

```go
func (it Item) Amount() int64
```

Amount is the line's total: Quantity × UnitAmount.

<small>[cart/cart.go:113](https://github.com/rohanthewiz/grmob/blob/master/cart/cart.go#L113)</small>

### type Persister

```go
type Persister interface {
	// Load returns the saved cart, or an empty Cart and a nil error when
	// nothing has been saved yet.
	Load() (Cart, error)

	// Save replaces the saved cart with c.
	Save(c Cart) error
}
```

Persister keeps a cart between launches. Package cartdb has one backed by bytdb; anything that can hold a blob (a file, the app's own database, a server) can be one.

<small>[cart/store.go:12](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L12)</small>

### type Quote

```go
type Quote struct {
	// Cart is the cart that was priced.
	Cart Cart

	// Currency is Config.Currency, upper-cased, defaulting to USD, so a
	// widget handed only the Quote can format its amounts.
	Currency string

	// Subtotal is the sum of the lines.
	Subtotal int64

	// Adjustments are the rules' lines, in rule order: discounts (negative),
	// shipping, tax.
	Adjustments []Adjustment

	// Total is Subtotal plus every Adjustment.
	Total int64
}
```

Quote is a cart priced by Config.Price: the snapshot a screen draws.

<small>[cart/price.go:6](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L6)</small>

#### func (Quote) Count

```go
func (q Quote) Count() int
```

Count is the number of things in the quoted cart; see Cart.Count.

<small>[cart/price.go:26](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L26)</small>

### type Rate

```go
type Rate int64
```

Rate is a percentage held in hundredths of a percent (basis points), so a rate like 8.25% is exact and every computation on it stays in integers. Write one with Percent.

<small>[cart/price.go:101](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L101)</small>

#### func Percent

```go
func Percent(p float64) Rate
```

Percent is p percent as a Rate: Percent(8.25) is 825 basis points. Input is rounded to the nearest basis point.

<small>[cart/price.go:105](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L105)</small>

#### func (Rate) Of

```go
func (r Rate) Of(amount int64) int64
```

Of is r applied to amount, rounded half away from zero to a whole minor unit. Integer arithmetic throughout: amount×rate fits an int64 for any cart under ~9×10¹⁴ minor units at a 100% rate.

<small>[cart/price.go:110](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L110)</small>

### type Remove

```go
type Remove struct{ ID string }
```

Remove takes the line with ID out of the cart.

<small>[cart/reduce.go:95](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L95)</small>

### type RemoveCode

```go
type RemoveCode struct{ Code string }
```

RemoveCode takes a promo code off.

<small>[cart/reduce.go:103](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L103)</small>

### type Replace

```go
type Replace struct{ Cart Cart }
```

Replace swaps in a whole cart: loading one the server holds, or undoing a removal (keep the Cart from before the Remove and Replace it back, which also restores the line's position).

<small>[cart/reduce.go:112](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L112)</small>

### type Rule

```go
type Rule struct {
	// Code, when set, ties the rule to a promo code: it applies only while
	// that code is applied to the cart, and naming it here is what lets the
	// default Config.AcceptCode take the code. Compared after NormalizeCode.
	Code string

	// Apply returns the rule's lines for the Quote so far; nil adds none. A
	// nil Apply makes the rule a no-op, which keeps a Rule{} harmless.
	Apply func(q Quote) []Adjustment
}
```

Rule is one step of pricing. Rules run in Config.Rules order and each sees the Quote so far, Total included, so the order is the policy: a discount listed before tax is taxed on the discounted amount, and one listed after is not.

The built-in constructors (PercentOff, AmountOff, FlatShipping, Tax) cover the common cases. Anything else is a Rule written in place:

	cart.Rule{Apply: func(q cart.Quote) []cart.Adjustment {
	    if q.Count() >= 3 {
	        return []cart.Adjustment{{Label: "3 for 2", Amount: -cheapest(q.Cart)}}
	    }
	    return nil
	}}

<small>[cart/price.go:54](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L54)</small>

#### func AmountOff

```go
func AmountOff(code, label string, amount int64) Rule
```

AmountOff takes amount (minor units) off the running total while code is applied, never taking it below zero.

<small>[cart/price.go:133](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L133)</small>

#### func FlatShipping

```go
func FlatShipping(label string, amount, freeOver int64) Rule
```

FlatShipping charges amount on any non-empty cart, and nothing (a zero line, drawn as "Free") once the running total reaches freeOver. A freeOver of zero or less never waives it. The threshold is checked against the running total, so a discount listed before this rule counts against it.

<small>[cart/price.go:147](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L147)</small>

#### func PercentOff

```go
func PercentOff(code, label string, rate Rate) Rule
```

PercentOff takes rate off the running total, while code is applied (an empty code applies always: a storewide sale). The discount never exceeds the running total, so it cannot make an order negative.

<small>[cart/price.go:121](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L121)</small>

#### func Tax

```go
func Tax(label string, rate Rate) Rule
```

Tax adds rate of the running total. List it last to tax the order after discounts and shipping, or before FlatShipping where shipping is not taxed.

<small>[cart/price.go:159](https://github.com/rohanthewiz/grmob/blob/master/cart/price.go#L159)</small>

### type SetQuantity

```go
type SetQuantity struct {
	ID       string
	Quantity int
}
```

SetQuantity sets the line with ID to Quantity, clamped to its cap. Below one removes the line. An unknown ID is ignored.

<small>[cart/reduce.go:89](https://github.com/rohanthewiz/grmob/blob/master/cart/reduce.go#L89)</small>

### type Store

```go
type Store struct {
	// contains filtered or unexported fields
}
```

Store is one cart shared by every screen of an app: the product page that adds to it, the app bar badge that counts it, and the cart screen that edits it all read the same Store, and a change made through any of them re-renders the others (through hooks.UseCart).

	var store = cart.NewStore(cfg)               // one per app, made once

	q := hooks.UseCart(ctx, store)               // in each screen
	comps.CartList{
	    Items:      q.Cart.Items,
	    Currency:   q.Currency,
	    OnQuantity: store.SetQuantity,           // plain setters: the
	    OnRemove:   store.Remove,                // widgets' callback shapes
	}

#### Why a store and not state in a screen

core state lives in a context's hook slots, which belong to one tree position. A cart outlives every screen that shows it, and GrMob has no provider mechanism to hand one value down a navigation stack, so the cart lives here and screens subscribe to it. This is the shape the host-reported values already take (UseLocation, UsePermissionLive): a process-wide source and a hook that re-renders its caller when the source changes.

#### Concurrency

Every method is safe from any goroutine. Changes are serialized by one mutex; the save runs under it, so saves land in the order the changes were made (bytdb's single-row writes take microseconds, the cost the todoapp example already accepts on its tap path). Subscribers are called after the lock is released, so one may read the Store back without deadlocking.

<small>[cart/store.go:52](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L52)</small>

#### func NewStore

```go
func NewStore(cfg Config) *Store
```

NewStore makes a Store with cfg, holding the cart cfg.Persister has saved (or an empty one). A load error goes to cfg.OnError and the store starts empty, for OnError's reason: an app with a broken file still works.

A zero Store is also usable: an in-memory cart with Config{}.

<small>[cart/store.go:66](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L66)</small>

#### func (*Store) Add

```go
func (s *Store) Add(it Item)
```

Add puts it in the cart; see Add.

<small>[cart/store.go:130](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L130)</small>

#### func (*Store) ApplyCode

```go
func (s *Store) ApplyCode(code string) error
```

ApplyCode applies a promo code when Config.AcceptCode takes it, and returns AcceptCode's error otherwise. A code already applied returns nil and changes nothing. Its shape is PromoCodeField.OnApply's, so the field shows the error under the input.

<small>[cart/store.go:148](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L148)</small>

#### func (*Store) Cart

```go
func (s *Store) Cart() Cart
```

Cart is the current cart. See the package doc on snapshots: it is safe to keep and read, and must not be modified.

<small>[cart/store.go:84](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L84)</small>

#### func (*Store) Clear

```go
func (s *Store) Clear()
```

Clear empties the cart, codes included.

<small>[cart/store.go:142](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L142)</small>

#### func (*Store) Config

```go
func (s *Store) Config() Config
```

Config is the Store's configuration, as NewStore was given it.

<small>[cart/store.go:80](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L80)</small>

#### func (*Store) Dispatch

```go
func (s *Store) Dispatch(action Action)
```

Dispatch applies action. When the cart actually changes it is saved and every subscriber is told; an action that changes nothing (a "+" at the cap, removing a line already gone) does neither, so a widget's no-op tap costs no render pass and no write.

<small>[cart/store.go:97](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L97)</small>

#### func (*Store) Quote

```go
func (s *Store) Quote() Quote
```

Quote is the current cart, priced by the Store's rules.

<small>[cart/store.go:91](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L91)</small>

#### func (*Store) Remove

```go
func (s *Store) Remove(id string)
```

Remove takes a line out. Its shape is CartList.OnRemove's.

<small>[cart/store.go:139](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L139)</small>

#### func (*Store) RemoveCode

```go
func (s *Store) RemoveCode(code string)
```

RemoveCode takes a promo code off. Its shape is PromoCodeField.OnRemove's.

<small>[cart/store.go:161](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L161)</small>

#### func (*Store) SetQuantity

```go
func (s *Store) SetQuantity(id string, quantity int)
```

SetQuantity sets a line's quantity; below one removes it. Its shape is CartList.OnQuantity's.

<small>[cart/store.go:134](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L134)</small>

#### func (*Store) Subscribe

```go
func (s *Store) Subscribe(fn func()) (cancel func())
```

Subscribe registers fn to run after every change, and returns the function that unregisters it. fn runs on the goroutine that made the change, after the Store's lock is released. Cancelling twice is harmless.

<small>[cart/store.go:166](https://github.com/rohanthewiz/grmob/blob/master/cart/store.go#L166)</small>

