package comps

import (
	"strings"
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernPromoCodeFieldInert is raised, in debug builds only, when a
// PromoCodeField has neither OnApply nor OnApplyAsync. Apply clears nothing and applies nothing,
// and the reader is left believing their discount is on the order.
const ConcernPromoCodeFieldInert = "promo-code-field-inert"

// PromoCodeField is where a reader types a promo code: an input and an Apply
// button, the reason a code was refused, and the codes already applied, each
// with its own ✕.
//
//	comps.PromoCodeField{
//	    OnApply:  store.ApplyCode,    // func(string) error
//	    Applied:  q.Cart.Codes,
//	    OnRemove: store.RemoveCode,
//	}
//
//	┌ Column ──────────────────────────────────────────────┐
//	│ ┌ Row ───────────────────────────────────────────────┐ │
//	│ │ [ Promo code                         ]  [ Apply ]  │ │  return applies too
//	│ └────────────────────────────────────────────────────┘ │
//	│  That code isn't valid                                 │  RoleAlert, when refused
//	│ ┌ Row  role=list  wrap ──────────────────────────────┐ │
//	│ │ ( SPRING10  ✕ ) ( FREESHIP  ✕ )                    │ │  one listitem each
//	│ └────────────────────────────────────────────────────┘ │
//	└──────────────────────────────────────────────────────┘
//
// # OnApply answers with an error
//
// Whether a code is good is not this widget's to decide: it may need the
// cart (a minimum order) or a server. OnApply returns nil to accept, and the
// draft clears; or an error, whose text is drawn under the input, and the
// draft stays so the reader can fix a typo rather than retype the code.
// cart.Store.ApplyCode has exactly this shape. Typing again clears the
// error, since it described a code that is no longer in the field.
//
// OnApply runs on the event path and its answer is drawn on the next pass,
// so it must answer promptly: a network call there freezes the screen.
//
// # A code only the server can check: OnApplyAsync
//
// OnApplyAsync is handed the code and a done func, and returns at once. The
// field is pending until done is called: the input and the button are
// disabled and the button reads PendingLabel ("Checking…"), which is also
// what a screen reader hears on it. done(nil) clears the draft; done(err)
// shows err, as OnApply's answers do. done may be called from any
// goroutine, and only its first call counts.
//
//	comps.PromoCodeField{
//	    OnApplyAsync: func(code string, done func(error)) {
//	        go func() {
//	            err := api.CheckCode(code)             // your server
//	            if err == nil {
//	                store.Dispatch(cart.ApplyCode{Code: code})
//	            }
//	            done(err)
//	        }()
//	    },
//	    Applied:  q.Cart.Codes,
//	    OnRemove: store.RemoveCode,
//	}
//
// Dispatch rather than Store.ApplyCode, because the server has already
// accepted the code and the Config's AcceptCode would ask again. The rule
// that prices the code still has to be in Config.Rules.
//
// A callback rather than a Pending field the caller drives: the pending
// state belongs to one attempt, and a done func ties the answer to the
// attempt that asked, with nothing for the caller to keep in step. The
// input is locked while an attempt is out, so there is only ever one.
// OnApplyAsync wins when both are set, as StripeCheckout's OnPay wins over
// its CheckoutURL.
//
// # The draft and the error are the widget's
//
// TagInput's test: no application wants a half-typed code, and the refusal
// is about the last attempt, not about the cart. They, and the pending
// attempt, live in hooks here, so
// the hook rules apply: render a PromoCodeField unconditionally, in a
// stable position.
//
// # Accessibility
//
// The input is named by Label. A refusal is a RoleAlert, so a reader who
// pressed Apply hears why at once. Applied codes are a list of pills whose
// ✕ is a separate button named "Remove SPRING10", TagInput's shape: a chip
// with a button inside it is a button inside a button.
//
// # Theme roles read
//
//	Input    the theme's input
//	Apply    an outlined Button
//	Error    Typography.Caption, Colors.Error
//	Pills    as TagInput's: Colors.Surface fill, ColorPalette.BorderColor hairline
type PromoCodeField struct {
	// OnApply is asked to apply a code (trimmed, never empty) and returns
	// nil, or the reason it was refused. Nil reports
	// ConcernPromoCodeFieldInert.
	OnApply func(code string) error

	// OnApplyAsync is OnApply for an answer that takes time: it is handed the
	// code and a done func to call with nil or the refusal, from any
	// goroutine. It wins over OnApply. See the type doc.
	OnApplyAsync func(code string, done func(error))

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

	// PendingLabel is the button's text while an OnApplyAsync attempt is
	// pending. Empty is "Checking…".
	PendingLabel string

	// RemoveLabel prefixes each ✕'s spoken name. Empty is "Remove".
	RemoveLabel string

	// Disabled disables the input, the button and every ✕.
	Disabled bool

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}

// Render draws the field. It takes three hooks before any branch: the draft,
// the last refusal, and whether an async attempt is pending.
func (p PromoCodeField) Render(ctx *core.Context) *core.Node {
	draft := core.NewState(ctx, "")
	refusal := core.NewState(ctx, "")
	pending := core.NewState(ctx, false)
	t := ctx.Theme()

	if core.IsDebugMode() && p.OnApply == nil && p.OnApplyAsync == nil {
		core.ReportConcern(ConcernPromoCodeFieldInert,
			"PromoCodeField has neither OnApply nor OnApplyAsync, so Apply does nothing with the code typed")
	}

	label := orDefault(p.Label, "Promo code")
	// answer settles an attempt the same way whichever path produced it.
	answer := func(err error) {
		if err != nil {
			refusal.Set(err.Error())
			return
		}
		draft.Set("")
		refusal.Set("")
	}
	apply := func() {
		code := strings.TrimSpace(draft.Get())
		if code == "" || p.Disabled || pending.Get() {
			return
		}
		switch {
		case p.OnApplyAsync != nil:
			pending.Set(true)
			refusal.Set("")
			// Only the first call counts: a caller that answers from a
			// timeout and a response both must not clear a later draft.
			var once sync.Once
			p.OnApplyAsync(code, func(err error) {
				once.Do(func() {
					pending.Set(false)
					answer(err)
				})
			})
		case p.OnApply != nil:
			answer(p.OnApply(code))
		}
	}
	onChange := func(s string) {
		draft.Set(s)
		if refusal.Get() != "" {
			refusal.Set("")
		}
	}

	input := []core.PropsAndChildren{
		core.FlexGrow(1),
		core.FlexBasis("0"),
		core.AccessibilityLabel(label),
	}
	// Pending disables the input too: a code edited while its check is out
	// would leave the answer describing text no longer in the field.
	busy := pending.Get()
	if p.Disabled || busy {
		input = append(input, core.Disabled(true))
	}
	applyLabel := orDefault(p.ApplyLabel, "Apply")
	if busy {
		applyLabel = orDefault(p.PendingLabel, "Checking…")
	}

	items := make([]core.PropsAndChildren, 0, len(p.Style)+5)
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
	)
	items = append(items, asProps(p.Style)...)
	items = append(items, core.Row(
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.InputWithSubmit(draft.Get(), orDefault(p.Placeholder, label), onChange, apply, input...),
		Button{
			Label:    applyLabel,
			OnTap:    apply,
			Emphasis: EmphasisOutlined,
			// Disabled while there is nothing to apply, or an attempt is
			// out, so the button says what a tap would do; apply still
			// guards both for a tap racing this patch.
			Disabled: p.Disabled || busy || strings.TrimSpace(draft.Get()) == "",
		},
	))

	if msg := refusal.Get(); msg != "" {
		items = append(items, core.Text(msg,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.Error),
			core.AccessibilityRole(core.RoleAlert),
		))
	}

	if len(p.Applied) > 0 {
		strip := make([]core.PropsAndChildren, 0, len(p.Applied)+4)
		strip = append(strip,
			core.Padding(0),
			core.FlexWrap(true),
			core.Gap(float64(t.Spacing.XS)),
			core.AccessibilityRole(core.RoleList),
		)
		removeLabel := orDefault(p.RemoveLabel, "Remove")
		for _, code := range p.Applied {
			var remove func()
			if onRemove := p.OnRemove; onRemove != nil {
				remove = func() { onRemove(code) }
			}
			strip = append(strip, core.Keyed("code:"+code,
				removablePill(t, code, removeLabel+" "+code, p.Disabled, remove)))
		}
		items = append(items, core.Row(strip...))
	}

	return core.Column(items...).Render(ctx)
}
