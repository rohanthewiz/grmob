package comps

import (
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernPromoCodeFieldInert is raised, in debug builds only, when a
// PromoCodeField has no OnApply. Apply clears nothing and applies nothing,
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
// so it should answer promptly. A server check is better made by accepting
// the code (Store.Dispatch(cart.ApplyCode{…})) and pricing it once the
// server agrees, or by an AcceptCode that consults a list fetched earlier.
//
// # The draft and the error are the widget's
//
// TagInput's test: no application wants a half-typed code, and the refusal
// is about the last attempt, not about the cart. Both live in hooks here, so
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

// Render draws the field. It takes two hooks, the draft and the last
// refusal, before any branch.
func (p PromoCodeField) Render(ctx *core.Context) *core.Node {
	draft := core.NewState(ctx, "")
	refusal := core.NewState(ctx, "")
	t := ctx.Theme()

	if core.IsDebugMode() && p.OnApply == nil {
		core.ReportConcern(ConcernPromoCodeFieldInert,
			"PromoCodeField has no OnApply, so Apply does nothing with the code typed")
	}

	label := orDefault(p.Label, "Promo code")
	apply := func() {
		code := strings.TrimSpace(draft.Get())
		if code == "" || p.OnApply == nil || p.Disabled {
			return
		}
		if err := p.OnApply(code); err != nil {
			refusal.Set(err.Error())
			return
		}
		draft.Set("")
		refusal.Set("")
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
	if p.Disabled {
		input = append(input, core.Disabled(true))
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
			Label:    orDefault(p.ApplyLabel, "Apply"),
			OnTap:    apply,
			Emphasis: EmphasisOutlined,
			// Disabled while there is nothing to apply, so the button says
			// what a tap would do; apply still guards an empty draft for a
			// tap racing this patch.
			Disabled: p.Disabled || strings.TrimSpace(draft.Get()) == "",
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
