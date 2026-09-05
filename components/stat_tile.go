package components

import "github.com/rohanthewiz/grmob/core"

// StatTile is one figure with its name and, optionally, its movement — the
// unit a dashboard, a profile header, or an account summary is made of.
//
//	core.Row(core.Gap(12),
//	    components.StatTile{Label: "Attendance", Value: "412", Fill: true,
//	        Delta: "+18 vs last week", DeltaVariant: components.VariantSuccess},
//	    components.StatTile{Label: "Giving", Value: "MZN 42,750", Fill: true},
//	)
//
// # It has no frame, deliberately
//
// "Tile" names the content, not a card. The widget renders a text stack and
// paints nothing: no background, no border, no padding of its own. That is
// what lets the two common arrangements both be composition rather than
// configuration — three tiles inside one core.Card, or three tiles each in
// their own — instead of a Framed bool that is wrong half the time. The
// fintech example's balance block was already exactly this shape inside a
// card; it is the card that was doing the framing, and it still is.
//
// # Order: label, then figure
//
// The label sits above the value. In a row of tiles that keeps the labels on
// one line at the top and the figures on another below them, which survives
// labels of different lengths; the other order ("412" over "Attendance")
// makes the figures ragged the moment one label wraps to two lines. For a
// centered arrangement, pass core.AlignItemsProp(core.AlignItemsCenter) in
// Style — each line then shrinks to its content and centers.
//
// # The delta's zero value is neutral, not Primary
//
// Everywhere else in this package VariantDefault resolves to the theme's
// Primary — that is what Badge and Button do, and what makes their zero value
// a no-op. Here it resolves to the secondary text ink instead, and the reason
// is that a delta is a measurement rather than a status. Whether a number
// going up is good is entirely the caller's domain: attendance up is success,
// expenses up is not, and latency up is an incident. A widget that guessed —
// by coloring on the sign, or by defaulting to the brand color as if any
// movement were noteworthy — would be confidently wrong on half the tiles a
// real screen carries. So the default says nothing, and a caller who knows
// what the movement means says it with DeltaVariant.
type StatTile struct {
	// Label is what the figure measures. Value is the figure itself,
	// pre-formatted — the widget does no number formatting, because currency,
	// grouping and locale are the caller's, not a layout widget's.
	Label string
	Value string

	// Delta is the movement line under the value: "+18 vs last week", "−3%",
	// "unchanged". Empty renders nothing.
	Delta string

	// DeltaVariant colors the delta with a semantic role. The zero value is
	// the theme's secondary ink — neutral, saying nothing about whether the
	// movement is good. See the type comment for why this one field departs
	// from the package's usual reading of VariantDefault.
	//
	// The same contrast caveat applies here as to an outlined or ghost
	// Button, and for the same reason: the role color is laid on a backdrop
	// the widget cannot see, so the theme's own numbers are what you get.
	// Against each theme's Background, Success is 2.22:1 under DefaultTheme
	// and Warning 2.20:1 — well under the 4.5:1 body-text floor, i.e. a delta
	// that is decoration rather than text. Under MaterialTheme the same two
	// are 5.13:1 and 3.08:1. So on the default palette a colored delta needs
	// to be reinforcement for wording that already says it ("+18, best week
	// this term"), never the only place the direction appears; a caller who
	// needs the color itself to be legible wants a Badge, which owns its fill
	// and picks its ink by contrast.
	DeltaVariant Variant

	// Fill makes the tile take an equal share of a row rather than hugging
	// its content, which is what a row of two or three tiles wants.
	//
	// It sets FlexGrow *and* a zero FlexBasis, which looks like belt and
	// braces and is actually what makes the four targets agree. Compose's
	// Modifier.weight and SwiftUI's equivalent divide the whole axis by
	// weight, so equal grows are already equal widths there; CSS flex-grow
	// divides only the *leftover* space, so on the web a tile with a longer
	// value would come out wider. Zeroing the basis is what makes CSS
	// distribute the whole axis too. The natives ignore FlexBasis, so the one
	// prop that is inert on two targets is precisely the one that converges
	// the other two.
	Fill bool

	// OnTap makes the whole tile a target — a KPI that opens the report
	// behind it. Wired only when non-nil, so a presentational tile registers
	// no callback.
	OnTap func()

	// AccessibilityLabel names the tile as one thing, which is usually what
	// you want: a reader moving through three tiles should hear "Attendance,
	// 412, up 18 from last week", not six fragments. Nothing is synthesized
	// when it is empty — the three lines are then announced separately, which
	// is correct but wordier.
	AccessibilityLabel string
	AccessibilityHint  string

	// Style is applied after the widget's own defaults.
	Style []core.StyleProp
}

func (s StatTile) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(s.Style)+8)
	// XS between the three lines: they are one block, not three stacked
	// items. Same tight-stack recipe ListRow's middle column uses.
	items = append(items, core.Gap(float64(t.Spacing.XS)))

	if s.Fill {
		items = append(items, core.FlexGrow(1), core.FlexBasis("0"))
	}
	for _, sp := range s.Style {
		items = append(items, sp)
	}
	if s.AccessibilityLabel != "" {
		items = append(items, core.AccessibilityLabel(s.AccessibilityLabel))
	}
	if s.AccessibilityHint != "" {
		items = append(items, core.AccessibilityHint(s.AccessibilityHint))
	}
	if s.OnTap != nil {
		items = append(items, core.OnClick(s.OnTap))
	}

	if s.Label != "" {
		items = append(items, core.Text(s.Label, core.UseStyle(t.Typography.Caption)))
	}
	if s.Value != "" {
		// The theme's Title role, ink included: the figure is the tile's
		// content and takes the theme's own heading color, not a brand accent
		// applied on top of it.
		items = append(items, core.Text(s.Value, core.UseStyle(t.Typography.Title)))
	}
	if s.Delta != "" {
		items = append(items, core.Text(s.Delta,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(s.deltaInk(t)),
		))
	}

	// Box, not Column: the tile paints and insets nothing, so it drops into a
	// card, a row or a list row without a padding of its own to undo. See the
	// type comment on why there is no frame here.
	return core.Box(items...).Render(ctx)
}

// deltaInk resolves the movement line's color: the role color for a set
// variant, the theme's secondary ink for the zero value.
func (s StatTile) deltaInk(t *core.Theme) string {
	if s.DeltaVariant == VariantDefault {
		return t.Colors.TextSecondary
	}
	return s.DeltaVariant.Color(t)
}
