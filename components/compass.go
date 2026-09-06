package components

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// Compass draws a bearing as a compass rose: a round card lettered N/E/S/W
// that turns so its N points the way north actually is, under a fixed index
// mark at the top showing where the device is pointing.
//
//	h := hooks.UseHeading(ctx)
//	components.Compass{Heading: h.Magnetic, ShowDegrees: true}
//
//	      ┌──▼───┐        index mark — fixed, over the rose's rim
//	      │   N  │        the rose turns by -Heading, so N stays
//	      │ W   E│        pointing at north while the device turns
//	      │   S  │
//	      └──────┘
//	      312° NW         optional readout
//
// # Which half turns
//
// Two conventions exist and they are not interchangeable. A magnetic compass
// has a fixed card and a needle that swings to north. A navigation compass —
// what every phone ships — turns the whole card and keeps a fixed index at
// twelve o'clock, so the bearing is read where the index crosses the rose.
// This is the second one, because the second one answers the question a phone
// user is asking ("which way am I facing", read off the top) rather than the
// question a hiker with a paper map is asking ("where is north").
//
// The rose therefore rotates by *minus* the heading. Turning the device
// clockwise increases the heading, and the rose must turn counter-clockwise by
// the same amount to keep pointing at the same piece of the world.
//
// # Where the index mark sits, and where it used to
//
// On the rose's rim, at twelve o'clock, drawn over it — which is where a
// compass index belongs and where this one could not go for as long as core
// had no z-axis container. Box stacks its children vertically on all four
// targets (settled deliberately — see the "Box overlay divergence" note in
// core.Box), and CSS absolute positioning is a declared web-only prop neither
// native renderer reads, so anything drawn *over* the rose would have been a
// web-only widget wearing a portable name. The mark was parked in the row
// above instead, costing a glyph of height and reading as a separate thing
// pointing at the dial rather than as part of it.
//
// core.ZStack is that container, and this is its first consumer. The dial is
// a stack of two layers — the rose, then a full-height column that justifies
// the mark to the top — because a ZStack centres every layer and a layer that
// wants to be somewhere else says so with its own box. See core.ZStack for
// why the alignment is a fixed centre rather than a prop.
//
// The rose's inset went from half a letter to a whole one to make room. The
// mark's glyph is three quarters of a letter tall, so a ring that deep is what
// keeps N clear of it at heading zero — the one bearing where the two
// deliberately coincide, since an index pointing at N is exactly what facing
// north looks like.
//
// # A float, not a core.Heading
//
// The field is a bearing in degrees rather than the sensor's own struct, so
// the widget also draws a bearing that has nothing to do with the compass —
// the direction of a route leg, a wind reading, the way a photograph was
// taken. hooks.UseHeading hands over the sensor's; anything else can hand over
// its own, and neither has to know about the other.
//
// # Accessibility
//
// The rose is four letters whose positions carry the meaning, which is exactly
// the content a screen reader cannot convey: read aloud in tree order it is
// "N W E S", turning or not. So the whole widget announces once, as a spoken
// bearing ("Heading 312 degrees, northwest"), and every part inside it is
// hidden. AccessibilityLabel overrides the sentence for a caller whose bearing
// means something more specific than a heading.
//
// The container takes core.RoleImg, and it is still the right value now that
// the two web exporters supply core.RoleGroup to any named container that has
// none (see core.RoleGroup). Both make the label legal — ARIA forbids an
// accessible name on a generic element, which is what a plain core layout node
// exports as, so the label alone was dropped by screen readers on both web
// targets while both natives read it out. What only `img` says is the part
// this widget depends on: a picture standing in for one fact, whose parts a
// reader should not read. `group` names its children and leaves them readable,
// which for a rose is "Heading 312 degrees, northwest" followed by "N W E S".
type Compass struct {
	// Heading is the bearing to draw, in degrees clockwise from north. Any
	// value works: it is normalised for the readout and the spoken label, and
	// applied as-is to the rotation (see core.Style.Rotate on winding).
	Heading float64

	// Size is the rose's diameter in px; 0 means 160.
	Size float64

	// ShowDegrees adds the numeric readout under the rose — "312° NW". Off by
	// default: a compass beside a map is a picture, and the number is noise
	// until something asks for it.
	ShowDegrees bool

	// Background is the rose's fill and Color its lettering and border; empty
	// uses the theme's Surface and TextSecondary. NorthColor inks the N and
	// the index mark, and defaults to the theme's Primary — north is the one
	// letter a reader looks for, and the accent is what makes it findable
	// while the rose is turning.
	Background string
	Color      string
	NorthColor string

	// Style is applied last, to the outer column, so a caller can space the
	// widget or give the whole thing a margin without reaching inside it.
	Style []core.StyleProp

	// AccessibilityLabel overrides the spoken bearing. See the type comment.
	AccessibilityLabel string
}

func (c Compass) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	size := c.Size
	if size <= 0 {
		size = 160
	}
	ink := c.Color
	if ink == "" {
		ink = t.Colors.TextSecondary
	}
	north := c.NorthColor
	if north == "" {
		north = t.Colors.Primary
	}
	bg := c.Background
	if bg == "" {
		bg = t.Colors.Surface
	}

	// Letters scale with the rose so one Size is the only knob, the same rule
	// Avatar's initials follow. An eighth of the diameter keeps the four
	// letters clear of each other at 96px and stops them looking lost at 240.
	// Floored at 10 because below that the glyph stops being legible on a
	// phone and a caller asking for a 60px compass wants a small compass, not
	// an unreadable one.
	letter := size / 8
	if letter < 10 {
		letter = 10
	}

	// Degrees for display are normalised; the rotation below is not. A
	// readout of "-48°" or "412°" is wrong in a way the rotation's winding is
	// not — a bearing shown to a person is a point on the circle, and only
	// the animation cares which way it got there.
	shown := core.NormalizeDegrees(c.Heading)

	// The rose. A square Box turned into a circle by a radius of half its
	// width, holding three rows spread top/middle/bottom: N, then W and E at
	// the edges, then S.
	//
	// Three rows and not a grid because flex is the whole vocabulary that
	// works on all four targets. JustifyBetween on the column pins the first
	// row to the top and the last to the bottom, and the middle row's own
	// JustifyBetween pushes W and E to the sides — which is what places four
	// letters at four compass points using nothing but two spread rules.
	//
	// Padding keeps the letters off the border; without it W and E sit on the
	// stroke, and at a small Size they cross it.
	rose := core.Box(
		core.Width(fmt.Sprintf("%gpx", size)),
		core.Height(fmt.Sprintf("%gpx", size)),
		core.BorderRadius(size/2),
		core.BackgroundColor(bg),
		core.BorderWidth(1),
		core.BorderColor(t.Colors.BorderColor()),
		// A whole letter, not half of one: the index mark is drawn over this
		// box's rim now (see the type doc), and the ring is what keeps N out
		// from under it.
		core.Padding(int(letter)),
		core.Justify(core.JustifyBetween),
		core.AlignItemsProp(core.AlignItemsCenter),
		// Minus the heading: see "Which half turns" on the type.
		core.Rotate(-c.Heading),
		// The whole rose is decorative; the container speaks for it.
		core.AccessibilityHidden(),

		roseRow(core.Text("N", core.FontSize(letter), core.TextColor(north),
			core.FontWeight(core.Bold))),
		core.Row(
			core.Padding(0),
			core.Width("100%"),
			core.Justify(core.JustifyBetween),
			core.AlignItemsProp(core.AlignItemsCenter),
			core.Text("W", core.FontSize(letter), core.TextColor(ink)),
			core.Text("E", core.FontSize(letter), core.TextColor(ink)),
		),
		roseRow(core.Text("S", core.FontSize(letter), core.TextColor(ink))),
	)

	// The dial: the rose with the index mark laid over it.
	//
	// "▼" rather than a drawn triangle: core has no shape primitive, and a CSS
	// border trick would be web-only. The glyph is in every system font on all
	// four targets. Sized below the letters so it reads as a mark against the
	// rose rather than a fifth cardinal point.
	//
	// The mark is wrapped in a full-height column rather than laid in the
	// stack bare, because a ZStack centres its layers: a bare glyph would sit
	// in the middle of the rose. The column is as tall as the rose and
	// justifies its one child to the start, which puts the mark on the rim —
	// the escape core.ZStack documents in place of a per-child alignment prop.
	dial := core.ZStack(
		core.Width(fmt.Sprintf("%gpx", size)),
		core.Height(fmt.Sprintf("%gpx", size)),
		// Hidden as a whole. Both layers are decorative — the container
		// speaks for the widget — and hiding the stack takes the subtree with
		// it rather than relying on each layer to hide itself.
		core.AccessibilityHidden(),
		rose,
		core.Column(
			core.Padding(0),
			core.Height(fmt.Sprintf("%gpx", size)),
			core.Justify(core.JustifyStart),
			core.AlignItemsProp(core.AlignItemsCenter),
			core.Text("▼",
				core.FontSize(letter*0.75),
				core.TextColor(north),
			),
		),
	)

	// The outer column: dial, optional readout.
	items := make([]core.PropsAndChildren, 0, 8+len(c.Style))
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		// Role and label together; see "Accessibility" on the type for why
		// the label does not survive on the web without the role.
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(c.label(shown)),
	)
	for _, sp := range c.Style {
		items = append(items, sp)
	}

	items = append(items, dial)

	if c.ShowDegrees {
		// Rounded to whole degrees. A magnetometer's accuracy is measured in
		// degrees, so a decimal place would be a digit of noise that changes
		// fifteen times a second.
		items = append(items, core.Text(
			fmt.Sprintf("%.0f° %s", shown, core.Cardinal(shown)),
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextPrimary),
			core.AccessibilityHidden(),
		))
	}

	return core.Column(items...).Render(ctx)
}

// roseRow centres one letter on its own row. The rose's column centres its
// children as a whole, but a bare Text child would size to its glyph and sit
// wherever the column's cross-axis rule put it; wrapping it in a full-width
// row that centres its own content places N and S over the circle's vertical
// axis whatever the glyph's width.
func roseRow(child core.View) core.View {
	return core.Row(
		core.Padding(0),
		core.Width("100%"),
		core.Justify(core.JustifyCenter),
		child,
	)
}

// label is the sentence the whole widget announces. Degrees are rounded and
// the point is spelled out in full — "north-northwest", not "NNW", which a
// screen reader pronounces as three letters.
func (c Compass) label(shown float64) string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	return fmt.Sprintf("Heading %.0f degrees, %s", shown, spokenPoint(shown))
}

// spokenPointNames are the sixteen rose points written the way they are said,
// indexed to match core.Cardinal's abbreviations.
var spokenPointNames = map[string]string{
	"N": "north", "NNE": "north-northeast", "NE": "northeast", "ENE": "east-northeast",
	"E": "east", "ESE": "east-southeast", "SE": "southeast", "SSE": "south-southeast",
	"S": "south", "SSW": "south-southwest", "SW": "southwest", "WSW": "west-southwest",
	"W": "west", "WNW": "west-northwest", "NW": "northwest", "NNW": "north-northwest",
}

func spokenPoint(deg float64) string {
	abbrev := core.Cardinal(deg)
	if full, ok := spokenPointNames[abbrev]; ok {
		return full
	}
	return abbrev
}
