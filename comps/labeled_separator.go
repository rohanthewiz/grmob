package comps

import "github.com/rohanthewiz/grmob/core"

// LabeledSeparator is the rule with a word in it — the "or" between "Sign in
// with Google" and the email form, the "Today" over a day's messages.
//
//	comps.LabeledSeparator{Label: "or"}
//
//	──────────────── or ────────────────
//
// It is a Row of two Separators that each grow to half the slack, with the
// label between them, so the word stays centred at any width and the rules
// are the theme's own hairline rather than a second one drawn here. An empty
// Label draws one unbroken rule, which is just a Separator with more nodes;
// it is allowed so a label that is sometimes empty does not need a branch.
//
// # Accessibility
//
// The rules are hidden, as every Separator is. The label is not: "or" between
// two ways to sign in is part of what the screen says, and a reader who skips
// it hears two sign-in buttons with nothing to say they are alternatives.
//
// # Theme roles read
//
//	Rules   ColorPalette.BorderColor, through Separator
//	Label   Typography.Caption over TextSecondary
//	Gap     Spacing.SM either side of the label
type LabeledSeparator struct {
	// Label is the word in the rule. Empty draws the rule unbroken.
	Label string

	// Color overrides the rules' tint, as Separator.Color does.
	Color string

	// Style is applied to the row after its defaults.
	Style []core.StyleProp
}

// Render builds Row(Separator grow, Text, Separator grow).
func (s LabeledSeparator) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	// Each rule grows by the same factor, so the label sits at the centre of
	// the row whatever its width. FlexGrow on a Separator is enough because a
	// Separator is a Box with a height and no width of its own.
	rule := Separator{Color: s.Color, Style: []core.StyleProp{core.FlexGrow(1)}}

	items := make([]core.PropsAndChildren, 0, len(s.Style)+6)
	items = append(items,
		// Vertical breathing room is the caller's, as it is for a Separator;
		// the theme Row base's padding would be a gap nobody asked for.
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
	)
	items = append(items, asProps(s.Style)...)

	items = append(items, rule)
	if s.Label != "" {
		items = append(items,
			core.Text(s.Label,
				core.UseStyle(t.Typography.Caption),
				core.TextColor(t.Colors.TextSecondary),
				// The rules give way first when the row is tight; the word
				// never does.
				core.FlexShrink(0),
			),
			rule,
		)
	}
	return core.Row(items...).Render(ctx)
}
