package components

import "github.com/rohanthewiz/grmob/core"

// Badge is a small non-interactive status pill — a count on a tab, a
// "verified" mark, a state label. For a *selectable* pill, use Chip.
//
// # Variants
//
// Variant names the badge's meaning and takes its colors from the palette's
// status roles, so a status pill stops carrying literal hex:
//
//	components.Badge{Text: "Paid", Variant: components.VariantSuccess}
//	components.Badge{Text: "Expiring", Variant: components.VariantWarning}
//	components.Badge{Text: "Failed", Variant: components.VariantError}
//
// The zero value is VariantDefault — the theme's Primary — which is the look
// every badge written before this field had, so adding it restyles nothing.
//
// The label ink is chosen per variant by contrast against the resolved
// background rather than fixed, because the palette pairs no ink with a status
// role and the right answer flips between themes. See Variant.Ink; the short
// version is that a fixed white ink would render DefaultTheme's Success and
// Warning badges at ~2.2:1, which is unreadable.
//
// # Color is not the message
//
// A variant is *reinforcement* for Text, never a substitute for it. Nothing
// here announces "warning" to a screen reader, and a reader who cannot
// distinguish the tints sees only the label — so the label has to say it
// ("Overdue", not "!"). That is WCAG 1.4.1, and it is why Variant deliberately
// does not synthesize an accessibility label the way Avatar does: Avatar has
// one obvious thing to say, a badge's meaning is already in its Text.
type Badge struct {
	Text string

	// Variant selects the semantic color role: Success, Warning, Error, or
	// the zero value for the theme's Primary.
	Variant Variant

	// Color is the pill background. Empty takes the Variant's role color; a
	// literal here overrides the variant, since an explicit color is the more
	// specific instruction.
	Color string

	// TextColor is the label ink. Empty is resolved by Variant.Ink: the
	// theme's Background for VariantDefault, and for a status variant
	// whichever of the theme's two ink roles reads better on the fill.
	TextColor string

	// Style is applied after the badge's own pill styling.
	Style []core.StyleProp
}

func (b Badge) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	bg := b.Color
	if bg == "" {
		bg = b.Variant.Color(t)
	}
	ink := b.TextColor
	if ink == "" {
		// Resolved against bg, not against the variant's role color, so an
		// explicit Color still gets a legible ink picked for it.
		ink = b.Variant.Ink(t, bg)
	}

	props := make([]core.StyleProp, 0, len(b.Style)+1)
	props = append(props, core.UseStyle(core.Style{
		FontSize:   t.Typography.Caption.FontSize,
		TextColor:  ink,
		Background: bg,
		// Oversized radius clamps to a stadium/pill shape at any height, so
		// the badge never needs to know its own size.
		BorderRadius: 999,
		Padding:      core.EdgeInsets{Top: 2, Bottom: 2, Left: 8, Right: 8},
		Display:      core.DisplayInline,
	}))
	props = append(props, b.Style...)

	return core.Text(b.Text, props...).Render(ctx)
}
