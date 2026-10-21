package cart

import "math"

// Quote is a cart priced by Config.Price: the snapshot a screen draws.
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

// Count is the number of things in the quoted cart; see Cart.Count.
func (q Quote) Count() int { return q.Cart.Count() }

// Adjustment is one line a Rule adds after the items. A negative Amount is a
// discount. An Amount of zero is drawn (CartSummary says "Free"), so a rule
// can show that shipping is free rather than leaving the reader to wonder.
type Adjustment struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`

	// Code is the promo code that produced the line, when one did, so a
	// widget can relate the discount to the code the reader typed.
	Code string `json:"code,omitempty"`
}

// Rule is one step of pricing. Rules run in Config.Rules order and each sees
// the Quote so far, Total included, so the order is the policy: a discount
// listed before tax is taxed on the discounted amount, and one listed after
// is not.
//
// The built-in constructors (PercentOff, AmountOff, FlatShipping, Tax) cover
// the common cases. Anything else is a Rule written in place:
//
//	cart.Rule{Apply: func(q cart.Quote) []cart.Adjustment {
//	    if q.Count() >= 3 {
//	        return []cart.Adjustment{{Label: "3 for 2", Amount: -cheapest(q.Cart)}}
//	    }
//	    return nil
//	}}
type Rule struct {
	// Code, when set, ties the rule to a promo code: it applies only while
	// that code is applied to the cart, and naming it here is what lets the
	// default Config.AcceptCode take the code. Compared after NormalizeCode.
	Code string

	// Apply returns the rule's lines for the Quote so far; nil adds none. A
	// nil Apply makes the rule a no-op, which keeps a Rule{} harmless.
	Apply func(q Quote) []Adjustment
}

// Price prices c: the subtotal, then each Rule in order, each seeing the
// running total. A Rule with a Code applies only when c has that code, and
// its adjustments are stamped with the code. An empty cart is priced with no
// rules at all, so it shows no shipping and no tax on nothing.
func (cfg Config) Price(c Cart) Quote {
	q := Quote{Cart: c, Currency: cfg.currency(), Subtotal: c.Subtotal()}
	q.Total = q.Subtotal
	if c.IsEmpty() {
		return q
	}
	for _, r := range cfg.Rules {
		if r.Apply == nil {
			continue
		}
		code := NormalizeCode(r.Code)
		if code != "" && !c.HasCode(code) {
			continue
		}
		for _, adj := range r.Apply(q) {
			if code != "" && adj.Code == "" {
				adj.Code = code
			}
			// Only what a rule returns is kept. Apply got a copy of q, so a
			// rule that appended to its copy's Adjustments wrote at most into
			// spare capacity past this Quote's length, which the next append
			// here overwrites.
			q.Adjustments = append(q.Adjustments, adj)
			q.Total += adj.Amount
		}
	}
	return q
}

// Rate is a percentage held in hundredths of a percent (basis points), so a
// rate like 8.25% is exact and every computation on it stays in integers.
// Write one with Percent.
type Rate int64

// Percent is p percent as a Rate: Percent(8.25) is 825 basis points. Input is
// rounded to the nearest basis point.
func Percent(p float64) Rate { return Rate(math.Round(p * 100)) }

// Of is r applied to amount, rounded half away from zero to a whole minor
// unit. Integer arithmetic throughout: amount×rate fits an int64 for any
// cart under ~9×10¹⁴ minor units at a 100% rate.
func (r Rate) Of(amount int64) int64 {
	n := amount * int64(r)
	if n >= 0 {
		return (n + 5000) / 10000
	}
	return -((-n + 5000) / 10000)
}

// PercentOff takes rate off the running total, while code is applied (an
// empty code applies always: a storewide sale). The discount never exceeds
// the running total, so it cannot make an order negative.
func PercentOff(code, label string, rate Rate) Rule {
	return Rule{Code: code, Apply: func(q Quote) []Adjustment {
		off := min(rate.Of(q.Total), max(q.Total, 0))
		if off <= 0 {
			return nil
		}
		return []Adjustment{{Label: label, Amount: -off}}
	}}
}

// AmountOff takes amount (minor units) off the running total while code is
// applied, never taking it below zero.
func AmountOff(code, label string, amount int64) Rule {
	return Rule{Code: code, Apply: func(q Quote) []Adjustment {
		off := min(amount, max(q.Total, 0))
		if off <= 0 {
			return nil
		}
		return []Adjustment{{Label: label, Amount: -off}}
	}}
}

// FlatShipping charges amount on any non-empty cart, and nothing (a zero line,
// drawn as "Free") once the running total reaches freeOver. A freeOver of
// zero or less never waives it. The threshold is checked against the running
// total, so a discount listed before this rule counts against it.
func FlatShipping(label string, amount, freeOver int64) Rule {
	return Rule{Apply: func(q Quote) []Adjustment {
		if freeOver > 0 && q.Total >= freeOver {
			return []Adjustment{{Label: label, Amount: 0}}
		}
		return []Adjustment{{Label: label, Amount: amount}}
	}}
}

// Tax adds rate of the running total. List it last to tax the order after
// discounts and shipping, or before FlatShipping where shipping is not
// taxed.
func Tax(label string, rate Rate) Rule {
	return Rule{Apply: func(q Quote) []Adjustment {
		tax := rate.Of(max(q.Total, 0))
		if tax == 0 {
			return nil
		}
		return []Adjustment{{Label: label, Amount: tax}}
	}}
}
