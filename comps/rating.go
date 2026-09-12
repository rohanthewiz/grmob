package comps

import (
	"fmt"
	"math"

	"github.com/rohanthewiz/grmob/core"
)

// Rating is a row of stars (or any glyph), read-only or tappable: a review
// score on a product row, or the "how was it?" prompt after an order.
//
//	comps.Rating{Value: 3, OnChange: stars.Set}              // interactive
//	comps.Rating{Value: 4.5, ReadOnly: true, Label: "Score"} // display only
//
//	┌ Row  role=group  label="Rating"  value="3 of 5" ─┐
//	│   ★     ★     ★     ☆     ☆                      │
//	│  btn   btn   btn   btn   btn   (interactive only) │
//	└───────────────────────────────────────────────────┘
//
// # Value is a float so half-stars need no signature change
//
// v1 rounds to the nearest whole glyph (4.5 draws five). A future half-glyph
// renders from the same field, so a caller storing an average today does not
// change type when that lands. OnChange reports whole numbers, because a tap
// lands on a whole glyph.
//
// # Interactive glyphs are buttons; read-only glyphs are decoration
//
// An interactive glyph is a Box with RoleButton, named "3 of 5", so each star
// is a separate, labelled tab stop and activation target. Tapping the star that
// is already the value does nothing, the same "report only changes" rule
// Stepper follows. A read-only rating registers no callbacks and hides the
// glyphs from assistive technology: the group's value ("4 of 5") is the one
// announcement, instead of five "black star" readings.
//
// # Accessibility
//
// The row is RoleGroup with Label (default "Rating") as its name and the
// rounded score as an AccessibilityValue. As with Stepper, the natives read
// the value's text on any node and the web scopes aria-value* to progressbar,
// so on the web the read-only group is announced by name and the per-star
// labels carry the score for the interactive one.
//
// # Theme roles read
//
//	Filled glyph   Colors.WarningOnLightColor() — amber that holds contrast on
//	               a light surface, where Colors.Warning is about 2:1
//	Empty glyph    Colors.TextSecondary
//	Glyph size     Typography.Subtitle
//	Gap            Spacing.XS
type Rating struct {
	// Value is the score, from 0 to Max. It is rounded to whole glyphs.
	Value float64

	// Max is the number of glyphs. Zero means 5.
	Max int

	// OnChange receives the tapped glyph's 1-based position. Nil, like
	// ReadOnly, draws a display-only rating.
	OnChange func(int)

	// ReadOnly drops the handlers and the per-glyph buttons.
	ReadOnly bool

	// Glyph and EmptyGlyph draw filled and unfilled positions. Empty uses
	// "★" and "☆".
	Glyph, EmptyGlyph string

	// Label is the group's accessible name. Empty uses "Rating".
	Label string

	// Style is applied to the row after the widget's own props.
	Style []core.StyleProp
}

// Render builds the glyph row described in the type doc.
func (r Rating) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	maxN := r.Max
	if maxN <= 0 {
		maxN = 5
	}
	// Rounded and clamped once, so the drawn stars, the announced value and
	// the "tap the current value" guard all agree on one integer.
	filled := int(math.Round(r.Value))
	filled = min(max(filled, 0), maxN)
	interactive := !r.ReadOnly && r.OnChange != nil

	items := make([]core.PropsAndChildren, 0, len(r.Style)+maxN+6)
	items = append(items,
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Padding(0),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(orDefault(r.Label, "Rating")),
		core.AccessibilityValue(core.ValueOf(float64(filled), 0, float64(maxN)).
			WithText(fmt.Sprintf("%d of %d", filled, maxN))),
	)
	for _, sp := range r.Style {
		items = append(items, sp)
	}

	full, empty := orDefault(r.Glyph, "★"), orDefault(r.EmptyGlyph, "☆")
	for i := range maxN {
		glyph, color := empty, t.Colors.TextSecondary
		if i < filled {
			glyph, color = full, t.Colors.WarningOnLightColor()
		}
		text := []core.StyleProp{core.UseStyle(t.Typography.Subtitle), core.TextColor(color)}

		if !interactive {
			items = append(items, core.Text(glyph, append(text, core.AccessibilityHidden())...))
			continue
		}
		pos := i + 1 // captured per glyph: the handler outlives this loop pass
		items = append(items, core.Box(
			core.AccessibilityRole(core.RoleButton),
			core.AccessibilityLabel(fmt.Sprintf("%d of %d", pos, maxN)),
			core.OnClick(func() {
				if pos != filled {
					r.OnChange(pos)
				}
			}),
			core.Text(glyph, text...),
		))
	}
	return core.Row(items...).Render(ctx)
}
