package cart

import (
	"errors"
	"slices"
	"strings"
)

// ErrUnknownCode is ApplyCode's answer for a code no Rule names, under the
// default Config.AcceptCode.
//
// Its text is lower case, as Go error strings are, so it composes into a
// longer message. comps.PromoCodeField sentence-cases a refusal when it
// draws one ("That code isn't valid"), and its ErrorText takes a
// translation; the error itself stays comparable with errors.Is.
var ErrUnknownCode = errors.New("that code isn't valid")

// Config is everything about a cart that is policy rather than contents: the
// currency, the quantity cap, the pricing rules, which codes are accepted,
// and where the cart is kept between launches. Every field's zero value is
// usable; Config{} is a USD cart with no cap, no rules and no persistence.
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

// currency is the configured currency, upper-cased, defaulting to USD.
func (cfg Config) currency() string {
	if cfg.Currency == "" {
		return "USD"
	}
	return strings.ToUpper(cfg.Currency)
}

// accept runs AcceptCode, or the default policy when it is nil.
func (cfg Config) accept(code string, c Cart) error {
	if cfg.AcceptCode != nil {
		return cfg.AcceptCode(code, c)
	}
	for _, r := range cfg.Rules {
		if r.Code != "" && NormalizeCode(r.Code) == code {
			return nil
		}
	}
	return ErrUnknownCode
}

// NormalizeCode is a promo code as the cart stores and compares it: trimmed
// and upper-cased, so " spring10" typed on a phone matches the "SPRING10" a
// Rule names.
func NormalizeCode(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }

// Action is a change to a cart, applied by Config.Reduce or Store.Dispatch.
// The set is closed (the method is unexported) so Reduce can promise to
// handle every Action there is.
type Action interface{ apply(cfg Config, c Cart) Cart }

// Add puts Item in the cart. A line with the same ID gains Item's quantity
// and takes Item's other fields (the newest label and price win, which is
// what a re-added catalog item means), keeping its place in the order. The
// quantity is clamped to the line's cap.
type Add struct{ Item Item }

// SetQuantity sets the line with ID to Quantity, clamped to its cap. Below
// one removes the line. An unknown ID is ignored.
type SetQuantity struct {
	ID       string
	Quantity int
}

// Remove takes the line with ID out of the cart.
type Remove struct{ ID string }

// ApplyCode adds a promo code. Reduce applies it without asking AcceptCode,
// because a reducer cannot return an error; Store.ApplyCode is the checked
// path the widgets use.
type ApplyCode struct{ Code string }

// RemoveCode takes a promo code off.
type RemoveCode struct{ Code string }

// Clear empties the cart, codes included: what happens after a successful
// payment.
type Clear struct{}

// Replace swaps in a whole cart: loading one the server holds, or undoing a
// removal (keep the Cart from before the Remove and Replace it back, which
// also restores the line's position).
type Replace struct{ Cart Cart }

// Reduce returns the cart after action. It never modifies c: every change
// builds new slices, which is what makes a Cart a value that render code can
// hold without a lock. It has the reducer shape hooks.UseReducer takes:
//
//	c, dispatch := hooks.UseReducer(ctx, cfg.Reduce, cart.Cart{})
//
// A nil action returns c unchanged.
func (cfg Config) Reduce(c Cart, action Action) Cart {
	if action == nil {
		return c
	}
	return action.apply(cfg, c)
}

func (a Add) apply(cfg Config, c Cart) Cart {
	it := a.Item
	if it.Quantity < 1 {
		it.Quantity = 1
	}
	if it.MaxQuantity == 0 {
		it.MaxQuantity = cfg.MaxQuantity
	}
	items := slices.Clone(c.Items)
	if i := c.index(it.ID); i >= 0 {
		it.Quantity += items[i].Quantity
		items[i] = clampItem(it)
	} else {
		items = append(items, clampItem(it))
	}
	c.Items = items
	return c
}

func (a SetQuantity) apply(_ Config, c Cart) Cart {
	i := c.index(a.ID)
	if i < 0 {
		return c
	}
	if a.Quantity < 1 {
		return Remove{ID: a.ID}.apply(Config{}, c)
	}
	items := slices.Clone(c.Items)
	items[i].Quantity = a.Quantity
	items[i] = clampItem(items[i])
	c.Items = items
	return c
}

func (a Remove) apply(_ Config, c Cart) Cart {
	i := c.index(a.ID)
	if i < 0 {
		return c
	}
	// slices.Delete works in place, so it runs on a clone: the caller's
	// backing array is shared with every earlier snapshot.
	c.Items = slices.Delete(slices.Clone(c.Items), i, i+1)
	return c
}

func (a ApplyCode) apply(_ Config, c Cart) Cart {
	code := NormalizeCode(a.Code)
	if code == "" || c.HasCode(code) {
		return c
	}
	c.Codes = append(slices.Clone(c.Codes), code)
	return c
}

func (a RemoveCode) apply(_ Config, c Cart) Cart {
	code := NormalizeCode(a.Code)
	i := slices.Index(c.Codes, code)
	if i < 0 {
		return c
	}
	c.Codes = slices.Delete(slices.Clone(c.Codes), i, i+1)
	return c
}

func (Clear) apply(_ Config, _ Cart) Cart { return Cart{} }

func (a Replace) apply(_ Config, _ Cart) Cart { return a.Cart }

// clampItem folds a line's quantity into [1, MaxQuantity] (no upper bound
// when MaxQuantity is zero or less).
func clampItem(it Item) Item {
	if it.MaxQuantity > 0 && it.Quantity > it.MaxQuantity {
		it.Quantity = it.MaxQuantity
	}
	if it.Quantity < 1 {
		it.Quantity = 1
	}
	return it
}
