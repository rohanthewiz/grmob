package comps

import (
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernStripeCheckoutInert is raised, in debug builds only, when a
// StripeCheckout has neither OnPay nor CheckoutURL and is not Disabled. Its
// Pay button looks ready and does nothing, on the one screen where a reader
// is most sure what a button should do.
const ConcernStripeCheckoutInert = "stripe-checkout-inert"

// ConcernStripeCheckoutInsecureURL is raised, in debug builds only, when
// CheckoutURL is not an https:// address. Stripe's Checkout Session and
// Payment Link URLs always are; anything else is a typo or a URL that did not
// come from Stripe, and the widget is about to hand it to the browser with
// the word "Pay" on it.
const ConcernStripeCheckoutInsecureURL = "stripe-checkout-insecure-url"

// CheckoutItem is one line of a StripeCheckout's order.
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

func (it CheckoutItem) qty() int {
	if it.Quantity < 1 {
		return 1
	}
	return it.Quantity
}

func (it CheckoutItem) amount() int64 { return int64(it.qty()) * it.UnitAmount }

// CheckoutAdjustment is a line after the items that changes the total:
// shipping, tax, a discount (a negative Amount). In minor units.
type CheckoutAdjustment struct {
	Label  string
	Amount int64
}

// StripeCheckout is the order summary and Pay button in front of a Stripe
// payment: the items, any shipping, tax or discount lines, the total, and one
// button that hands the payment to Stripe.
//
//	comps.StripeCheckout{
//	    Items: []comps.CheckoutItem{
//	        {Label: "Pour-over kettle", Detail: "Matte black", UnitAmount: 4500},
//	        {Label: "Filters (100)", Quantity: 2, UnitAmount: 650},
//	    },
//	    Adjustments: []comps.CheckoutAdjustment{{Label: "Shipping", Amount: 500}},
//	    Currency:    "usd",
//	    OnPay:       startCheckout, // asks your server for a Session, then core.OpenURL
//	    Pending:     redirecting.Get(),
//	}
//
//	┌ Card ─────────────────────────────────────────┐
//	│  Order summary                    heading 2   │
//	│  Pour-over kettle                    $45.00   │  Row, named
//	│  Matte black                                  │  "Pour-over kettle, $45.00"
//	│  Filters (100)                       $13.00   │
//	│  2 × $6.50                                    │
//	│  Shipping                             $5.00   │
//	│  ──────────────────────────────────────────── │  Separator
//	│  Total                               $63.00   │  Row, named "Total, $63.00"
//	│  (Error, when set)                            │  Text, RoleAlert
//	│  ┌──────────────── Pay $63.00 ─────────────┐  │  Button, full width
//	│  └─────────────────────────────────────────┘  │
//	│  🔒 Payments are processed securely by Stripe │  Caption
//	└───────────────────────────────────────────────┘
//
// # What it does not do: take a card
//
// There is no card field here, and there will not be. A card number typed
// into an app's own text field passes through the app's memory, its logs and
// its crash reports, and that puts the app in PCI DSS scope. Stripe's answer
// is to collect the card on a page it hosts, which is what Checkout and
// Payment Links are. So the widget's job ends at the hand-off: it shows what
// is being bought and sends the reader to Stripe to pay for it.
//
// Two ways to get there:
//
//   - CheckoutURL, a Payment Link (https://buy.stripe.com/…) or a Checkout
//     Session's url. A tap opens it with core.OpenURL: the browser on every
//     platform. Right for a fixed product, where the link can be made once in
//     the Stripe dashboard.
//   - OnPay, which wins when set. Creating a Checkout Session for a cart needs
//     the secret key, and the secret key lives on a server, never in an app
//     binary, where anyone can read it out. So OnPay is where the app calls
//     its own server, gets the Session's url back, and opens it. Set Pending
//     while that round trip runs.
//
// When the reader comes back (Stripe redirects to the success_url or
// cancel_url the Session was created with; on a phone, a deep link), the app
// asks its server whether the payment went through. Nothing this widget saw
// says so.
//
// # Amounts
//
// Every amount is an int64 in the currency's minor unit, as Stripe's API
// takes them: 4500 is $45.00, and 4500 yen is ¥4,500, because the yen has no
// minor unit. FormatMoney knows which currencies Stripe treats as zero- and
// three-decimal. The total is computed here from the lines, so the button can
// never disagree with the list above it; it is still the Session, not this
// number, that decides what is charged.
//
// # Accessibility
//
// Each line is a row named as one phrase, "Filters (100), 2 × $6.50,
// $13.00", so a reader hears an item and its price together rather than a
// column of names and then a column of prices. The Pay button says the total
// in its label. While Pending the button is disabled and reads
// "Redirecting to Stripe…", which is what the reader is waiting for.
//
// # No hooks
//
// Everything is the caller's, so StripeCheckout may be rendered
// conditionally.
//
// # Theme roles read
//
//	Card         the theme's Card base, Spacing.SM between the parts
//	Title        Typography.Subtitle, bold
//	Line         Typography.Body, Colors.TextPrimary; detail Caption, TextSecondary
//	Total        Typography.Subtitle, bold
//	Error        Typography.Body, Colors.Error
//	Button       as Button, with Variant
//	Note         Typography.Caption, Colors.TextSecondary
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

// Total is the sum of the items and adjustments, in minor units.
func (s StripeCheckout) Total() int64 {
	var total int64
	for _, it := range s.Items {
		total += it.amount()
	}
	for _, a := range s.Adjustments {
		total += a.Amount
	}
	return total
}

// Render draws the summary. It takes no hook slot.
func (s StripeCheckout) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	currency := strings.ToUpper(orDefault(s.Currency, "usd"))
	format := s.Format
	if format == nil {
		format = FormatMoney
	}
	total := format(s.Total(), currency)

	if core.IsDebugMode() {
		if s.OnPay == nil && s.CheckoutURL == "" && !s.Disabled {
			core.ReportConcern(ConcernStripeCheckoutInert,
				"StripeCheckout has neither OnPay nor CheckoutURL and is not Disabled, so its Pay button does nothing")
		}
		if s.CheckoutURL != "" && !strings.HasPrefix(s.CheckoutURL, "https://") {
			core.ReportConcern(ConcernStripeCheckoutInsecureURL,
				"StripeCheckout.CheckoutURL \""+s.CheckoutURL+"\" is not https://; Stripe's checkout and payment-link URLs always are")
		}
	}

	items := make([]core.PropsAndChildren, 0, len(s.Style)+len(s.Items)+len(s.Adjustments)+8)
	items = append(items, core.Gap(float64(t.Spacing.SM)))
	items = append(items, asProps(s.Style)...)

	title := []core.StyleProp{core.UseStyle(t.Typography.Subtitle), core.FontWeight(core.Bold)}
	title = append(title, headingProps(s.HeadingLevel, headingLevelSection)...)
	items = append(items, core.Text(orDefault(s.Title, "Order summary"), title...))

	for i, it := range s.Items {
		detail := it.Detail
		if detail == "" && it.qty() > 1 {
			detail = strconv.Itoa(it.qty()) + " × " + format(it.UnitAmount, currency)
		}
		// Keyed by position and label: a cart is reordered rarely, and a
		// label alone could repeat ("Gift card" twice at two prices).
		items = append(items, core.Keyed("item:"+strconv.Itoa(i)+":"+it.Label,
			checkoutLine(t, it.Label, detail, format(it.amount(), currency), false)))
	}
	for i, a := range s.Adjustments {
		items = append(items, core.Keyed("adj:"+strconv.Itoa(i)+":"+a.Label,
			checkoutLine(t, a.Label, "", format(a.Amount, currency), false)))
	}
	items = append(items, Separator{}, checkoutLine(t, "Total", "", total, true))

	if s.Error != "" {
		items = append(items, core.Text(s.Error,
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.Error),
			core.AccessibilityRole(core.RoleAlert),
		))
	}

	label := orDefault(s.PayLabel, "Pay "+total)
	if s.Pending {
		label = orDefault(s.PendingLabel, "Redirecting to Stripe…")
	}
	pay := s.OnPay
	hint := ""
	if pay == nil && s.CheckoutURL != "" {
		url := s.CheckoutURL
		pay = func() { core.OpenURL(url) }
		hint = "Opens Stripe Checkout in your browser"
	}
	// Button swaps a nil or disabled handler for a no-op itself, and keeps
	// it registered, so a tap racing the disabling patch is still caught.
	items = append(items, Button{
		Label:             label,
		OnTap:             pay,
		Variant:           s.Variant,
		FullWidth:         true,
		Disabled:          s.Disabled || s.Pending,
		AccessibilityHint: hint,
	})

	if !s.HideNote {
		note := orDefault(s.Note, "Payments are processed securely by Stripe.")
		items = append(items, core.Row(
			core.Padding(0),
			core.Gap(float64(t.Spacing.XS)),
			core.Justify(core.JustifyCenter),
			// The lock is decoration: the sentence says the same thing.
			core.Text("🔒", core.UseStyle(t.Typography.Caption), core.AccessibilityHidden()),
			core.Text(note, core.UseStyle(t.Typography.Caption), core.TextColor(t.Colors.TextSecondary)),
		))
	}

	return core.Card(items...).Render(ctx)
}

// checkoutLine is one row of the summary: the label (and a detail line under
// it) taking the room, the amount pinned to the end. strong is the total.
func checkoutLine(t *core.Theme, label, detail, amount string, strong bool) core.View {
	text := t.Typography.Body
	if strong {
		text = t.Typography.Subtitle
	}
	labelProps := []core.StyleProp{core.UseStyle(text), core.TextColor(t.Colors.TextPrimary)}
	amountProps := []core.StyleProp{core.UseStyle(text), core.TextColor(t.Colors.TextPrimary), core.FlexShrink(0)}
	if strong {
		labelProps = append(labelProps, core.FontWeight(core.Bold))
		amountProps = append(amountProps, core.FontWeight(core.Bold))
	}

	spoken := label
	if detail != "" {
		spoken += ", " + detail
	}
	spoken += ", " + amount

	left := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(2),
		// Grow from a zero basis: on the web a flex share includes padding,
		// and a long label must wrap rather than push the amount off the row.
		core.FlexGrow(1),
		core.FlexBasis("0"),
		core.Text(label, labelProps...),
	}
	if detail != "" {
		left = append(left, core.Text(detail,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextSecondary),
		))
	}
	return core.Row(
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsStart),
		core.AccessibilityLabel(spoken),
		core.Column(left...),
		core.Text(amount, amountProps...),
	)
}

// Stripe's zero- and three-decimal currencies: the ones whose minor unit is
// not a hundredth. From Stripe's "Supported currencies" list.
var (
	zeroDecimal = map[string]bool{
		"BIF": true, "CLP": true, "DJF": true, "GNF": true, "JPY": true, "KMF": true,
		"KRW": true, "MGA": true, "PYG": true, "RWF": true, "UGX": true, "VND": true,
		"VUV": true, "XAF": true, "XOF": true, "XPF": true,
	}
	threeDecimal = map[string]bool{"BHD": true, "JOD": true, "KWD": true, "OMR": true, "TND": true}

	// currencySymbols are the prefixes English readers expect. A currency
	// not listed is written with its code, which is never wrong, merely
	// plainer.
	currencySymbols = map[string]string{
		"USD": "$", "EUR": "€", "GBP": "£", "JPY": "¥", "INR": "₹", "KRW": "₩",
		"CAD": "CA$", "AUD": "A$", "NZD": "NZ$", "MXN": "MX$", "BRL": "R$",
		"CNY": "CN¥", "HKD": "HK$", "SGD": "S$",
	}
)

// FormatMoney writes amount, in currency's minor unit, the way an English
// receipt does: "$1,234.50", "¥4,500", "-€5.00", "SEK 99.00".
//
// The decimals follow Stripe's rules for the currency (two, or none for the
// yen and fifteen others, or three for the Gulf dinars), so an amount that
// round-trips through Stripe's API is shown as Stripe will charge it. It is
// English formatting throughout; StripeCheckout.Format takes a locale-aware
// replacement.
func FormatMoney(amount int64, currency string) string {
	code := strings.ToUpper(currency)
	decimals := 2
	switch {
	case zeroDecimal[code]:
		decimals = 0
	case threeDecimal[code]:
		decimals = 3
	}

	sign := ""
	if amount < 0 {
		sign, amount = "-", -amount
	}
	unit := int64(1)
	for range decimals {
		unit *= 10
	}
	whole, frac := amount/unit, amount%unit

	// Thousands separators, from the right.
	digits := strconv.FormatInt(whole, 10)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	num := b.String()
	if decimals > 0 {
		f := strconv.FormatInt(frac, 10)
		num += "." + strings.Repeat("0", decimals-len(f)) + f
	}

	if sym, ok := currencySymbols[code]; ok {
		return sign + sym + num
	}
	// A code is a word, so it needs a space; a no-break one keeps it on the
	// same line as the number.
	return sign + code + " " + num
}
