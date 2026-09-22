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
// By default it rounds to the nearest whole glyph (4.5 draws five). Halves
// draws the half-glyph this field was typed for: the value is rounded to the
// nearest half instead, and a caller storing an average never changed type.
// OnChange reports whole numbers either way, because a tap lands on a whole
// glyph.
//
// # Halves are drawn, not typed
//
// Half a "★" would be a text glyph clipped by a half-width box, and that is
// the one thing a text node cannot be trusted to do: SwiftUI truncates a
// Text proposed less than its width to "…" rather than letting it overflow
// to be clipped, and Compose would wrap or squeeze it. So with Halves set
// every position is a small core.Canvas star — the same shape for full,
// empty and half, so a half sits beside its neighbours as one drawing — and
// the half is exact geometry rather than a clip: a regular five-point star
// is symmetric about its vertical axis, which runs through the top point and
// the bottom inner vertex, so its left half is the polygon of the vertices on
// that side.
//
//	        0  (top point, on the axis)
//	       ╱╲
//	8 ────9  1──── 2        left half = 0 → 9 → 8 → 7 → 6 → 5 → 0
//	   ╲        ╱
//	    7      3            k: angle −90° + 36°·k, radius R (even k)
//	   ╱   5    ╲               or R·0.382 (odd k)
//	  6 ╱    ╲   4
//	         (5 is the bottom inner vertex, on the axis)
//
// Glyph and EmptyGlyph do not apply to a Halves rating, since its glyphs are
// drawings. Without Halves the tree is exactly what it was before the field
// existed.
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
	// Value is the score, from 0 to Max. It is rounded to whole glyphs, or
	// to halves with Halves.
	Value float64

	// Halves rounds Value to the nearest half and draws a half star for it.
	// See "Halves are drawn, not typed".
	Halves bool

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

	// With Halves, the score in half steps: whole is the number of full
	// stars, half whether one more is drawn half full. Without it these are
	// filled and false, and nothing below changes.
	whole, half := filled, false
	valueText := fmt.Sprintf("%d of %d", filled, maxN)
	valueNow := float64(filled)
	if r.Halves {
		halves := min(max(int(math.Round(r.Value*2)), 0), 2*maxN)
		whole, half = halves/2, halves%2 == 1
		valueNow = float64(halves) / 2
		valueText = fmt.Sprintf("%g of %d", valueNow, maxN)
	}

	items := make([]core.PropsAndChildren, 0, len(r.Style)+maxN+6)
	items = append(items,
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Padding(0),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(orDefault(r.Label, "Rating")),
		core.AccessibilityValue(core.ValueOf(valueNow, 0, float64(maxN)).
			WithText(valueText)),
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

		// The drawn star, for Halves; see "Halves are drawn, not typed". A
		// Canvas with no label is decoration, hidden like the text glyph.
		var star core.View
		if r.Halves {
			fill := 0.0
			if i < whole {
				fill = 1
			} else if i == whole && half {
				fill = 0.5
			}
			star = ratingStar(t, fill)
		}

		if !interactive {
			if star != nil {
				items = append(items, star)
				continue
			}
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
			glyphOr(star, core.Text(glyph, text...)),
		))
	}
	return core.Row(items...).Render(ctx)
}

// glyphOr returns the drawn star when there is one, else the text glyph.
func glyphOr(star, text core.View) core.View {
	if star != nil {
		return star
	}
	return text
}

// ratingStarInner is the inner radius of a regular five-point star as a
// fraction of the outer: sin 18° / sin 126°, the ratio at which the star's
// edges are collinear in pairs (the pentagram).
const ratingStarInner = 0.381966

// ratingStar draws one star at the text glyph's size: the outline always,
// and the fill over all of it (fill 1), its left half (0.5) or none (0).
func ratingStar(t *core.Theme, fill float64) core.View {
	const view, cx, cy, outer = 24.0, 12.0, 12.8, 11.0
	point := func(k int) (float64, float64) {
		r := outer
		if k%2 == 1 {
			r = outer * ratingStarInner
		}
		a := (-90 + 36*float64(k)) * math.Pi / 180
		return cx + r*math.Cos(a), cy + r*math.Sin(a)
	}
	polygon := func(ks ...int) *core.Path {
		p := core.NewPath()
		for i, k := range ks {
			x, y := point(k)
			if i == 0 {
				p.MoveTo(x, y)
			} else {
				p.LineTo(x, y)
			}
		}
		return p.Close()
	}

	on, off := t.Colors.WarningOnLightColor(), t.Colors.TextSecondary
	whole := polygon(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	var shapes []core.Shape
	switch {
	case fill >= 1:
		shapes = []core.Shape{{Path: whole, Fill: on, Stroke: on, StrokeWidth: 1.2, Join: core.JoinRound}}
	case fill > 0:
		shapes = []core.Shape{
			{Path: polygon(0, 9, 8, 7, 6, 5), Fill: on},
			{Path: whole, Stroke: on, StrokeWidth: 1.2, Join: core.JoinRound},
		}
	default:
		shapes = []core.Shape{{Path: whole, Stroke: off, StrokeWidth: 1.2, Join: core.JoinRound}}
	}

	size := t.Typography.Subtitle.FontSize
	if size == 0 {
		size = 22
	}
	return core.Canvas(view, view, shapes,
		// A half star fills its leading half: the left in LTR, the right
		// under RTL, where the row of stars also runs from the right.
		core.CanvasMirrorsRTL,
		core.Width(fmt.Sprintf("%gpx", size)),
		core.Height(fmt.Sprintf("%gpx", size)),
		core.FlexShrink(0),
	)
}
