package components

import "github.com/rohanthewiz/grmob/core"

// Badge is a small non-interactive status pill — a count on a tab, a
// "verified" mark, a state label. For a *selectable* pill, use Chip.
type Badge struct {
	Text string
	// Color is the pill background; defaults to the theme's Primary.
	Color string
	// TextColor is the label ink; defaults to the theme's Background, which
	// reads on Primary in both bundled themes (the same pairing the themes
	// use for Button).
	TextColor string
	// Style is applied after the badge's own pill styling.
	Style []core.StyleProp
}

func (b Badge) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	bg := b.Color
	if bg == "" {
		bg = t.Colors.Primary
	}
	ink := b.TextColor
	if ink == "" {
		ink = t.Colors.Background
	}

	props := make([]core.StyleProp, 0, len(b.Style)+1)
	props = append(props, core.UseStyle(core.Style{
		FontSize:  t.Typography.Caption.FontSize,
		TextColor: ink,
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
