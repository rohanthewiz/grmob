package comps

import (
	"fmt"
	"strconv"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// DigitalClock draws a time as digits, with an optional date line under them.
//
//	now := hooks.UseNow(ctx, time.Second)
//	comps.DigitalClock{Time: now, ShowSeconds: true, ShowDate: true}
//
//	    10:42:07 PM         digits, with the AM/PM marker set smaller
//	 Wednesday, September 16    optional date
//
// # A time, not a ticker
//
// The widget draws whatever Time it is handed and holds no hooks, so it can
// be rendered conditionally, tested at a fixed instant, and fed a time that is
// not "now here" (a world clock is Time: now.In(tokyo)). hooks.UseNow is the
// ticker; its doc explains why it aligns to the wall clock rather than to
// mount.
//
// # Accessibility
//
// Read in tree order the parts are "10:42:07", "PM", "Wednesday, September
// 16" — three stops for one fact. So the whole widget is one element that
// announces a sentence, with its parts hidden, the same shape Compass uses and
// for the same reason: RoleImg is what makes the label survive on the web (see
// Compass's Accessibility section).
//
// # Known limit
//
// core.Style has no font family, so the digits are the platform's
// proportional ones and the line's width can change by a pixel or two as the
// digits do. Centring (the default alignment here) keeps that from reading as
// a jitter at either edge.
type DigitalClock struct {
	// Time is the instant to draw, in the location it should be read in.
	Time time.Time

	// Hour24 draws 22:42 instead of 10:42 PM.
	Hour24 bool

	// ShowSeconds adds :07 to the digits. Off by default: a clock that only
	// changes once a minute can be driven by hooks.UseNow(ctx, time.Minute).
	ShowSeconds bool

	// ShowDate adds a line with the weekday, month and day.
	ShowDate bool

	// Size is the digits' font size in px; 0 means 40. The AM/PM marker and
	// the date line scale from it.
	Size float64

	// Color inks the digits; empty uses the theme's TextPrimary. The date line
	// always uses TextSecondary, so it reads as subordinate.
	Color string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel overrides the spoken time.
	AccessibilityLabel string
}

func (c DigitalClock) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	size := c.Size
	if size <= 0 {
		size = 40
	}
	ink := c.Color
	if ink == "" {
		ink = t.Colors.TextPrimary
	}

	digits, marker := clockDigits(c.Time, c.Hour24, c.ShowSeconds)

	// The digits row. AlignItemsEnd sits the smaller marker on the digits'
	// bottom edge, which is the nearest to a shared baseline that flex offers
	// on all four targets.
	row := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(size * 0.15),
		core.AlignItemsProp(core.AlignItemsEnd),
		core.AccessibilityHidden(),
		// Light, because a display-size numeral at Normal or Bold weight reads
		// as a heading rather than as a clock face.
		core.Text(digits, core.FontSize(size), core.FontWeight(core.Light), core.TextColor(ink)),
	}
	if marker != "" {
		row = append(row, core.Text(marker,
			core.FontSize(size*0.4),
			core.TextColor(ink),
			// Lifts the marker off the row's bottom edge by roughly the digits'
			// descender space, so it sits on the numerals' baseline rather than
			// below it.
			core.MarginBottom(int(size*0.14)),
		))
	}

	items := make([]core.PropsAndChildren, 0, 8+len(c.Style))
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(c.label(digits, marker)),
	)
	for _, sp := range c.Style {
		items = append(items, sp)
	}
	items = append(items, core.Row(row...))

	if c.ShowDate {
		items = append(items, core.Text(c.Time.Format("Monday, January 2"),
			core.FontSize(size*0.35),
			core.TextColor(t.Colors.TextSecondary),
			core.AccessibilityHidden(),
		))
	}

	return core.Column(items...).Render(ctx)
}

// label is the sentence the widget announces: the digits and marker as one
// phrase, then the date when it is drawn.
func (c DigitalClock) label(digits, marker string) string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	s := digits
	if marker != "" {
		s += " " + marker
	}
	if c.ShowDate {
		s += ", " + c.Time.Format("Monday, January 2")
	}
	return s
}

// clockDigits formats the time's digits and, on a 12-hour clock, its AM/PM
// marker separately, because the two are drawn at different sizes. The hour
// is not zero-padded on a 12-hour clock ("9:05", as every phone draws it) and
// is on a 24-hour one ("09:05", the ISO habit that 24-hour readers expect).
func clockDigits(tm time.Time, hour24, seconds bool) (digits, marker string) {
	layout := "3:04"
	if hour24 {
		layout = "15:04"
	}
	if seconds {
		layout += ":05"
	}
	digits = tm.Format(layout)
	if !hour24 {
		marker = tm.Format("PM")
	}
	return digits, marker
}

// AnalogClock draws a time as a clock face with hands.
//
//	now := hooks.UseNow(ctx, time.Second)
//	comps.AnalogClock{Time: now, ShowSeconds: true, Numerals: true}
//
// # How a hand pivots at the centre
//
// core.Rotate turns a node about its own centre and deliberately has no
// transform-origin; its doc says to wrap the thing in a box whose centre is
// the pivot. So every hand, tick and numeral is its own layer of a ZStack, and
// each layer is exactly the size of the dial. The layer is what turns, and its
// centre is the dial's centre:
//
//	┌───────────────┐   one layer, size × size, Rotate(angle)
//	│               │
//	│   (spacer)    │   height = size/2 − length
//	│       ┃       │
//	│       ┃       │   the hand: length + tail tall, horizontally centred
//	│       ● ──────┼── the layer's centre, which the hand's lower end passes
//	│       ┃       │   tail (second hand only)
//	│               │
//	└───────────────┘
//
// Everything outside the stick is transparent, so a stack of such layers
// draws as a face with hands. The cost is one node per layer, which on every
// target is a view with no content, and it uses only primitives that already
// agree on all four (see Style.Rotate's table) — no shape primitive needed.
//
// # Angles, and why they do not wrap
//
// All three hands are derived from the seconds elapsed in the local day:
//
//	hour   = s / 120     0 … 720°   (two turns a day)
//	minute = s / 10      0 … 8640°
//	second = s × 6       0 … 518400°
//
// Unwrapped because Smooth animates each change with a Transition, and a
// transition from 354° to 0° sweeps the long way back. Rotate passes its value
// through unnormalised for exactly this reason. The largest value is well
// within Compose's Float precision (about 0.03° at that magnitude).
//
// The day does roll over. At midnight every hand's angle falls to zero, which
// under a Transition would spin the second hand backwards 1,440 turns; so the
// pass whose time is in the first second of the day omits the Transition and
// the hands jump. A caller that skips that exact second (an app suspended over
// midnight) sees one backwards sweep, and only with Smooth.
//
// # Accessibility
//
// One element that says the time, as DigitalClock does; the face is hidden.
type AnalogClock struct {
	// Time is the instant to draw, in the location it should be read in.
	Time time.Time

	// Size is the dial's diameter in px; 0 means 200.
	Size float64

	// ShowSeconds draws the second hand.
	ShowSeconds bool

	// Numerals draws 1–12 inside the ticks.
	Numerals bool

	// MinuteTicks adds a fine tick for every minute between the hour ticks.
	// Off by default: it is 48 more layers, and at small sizes the ticks run
	// together into a ring.
	MinuteTicks bool

	// Smooth animates each step of the hands (a short ease-out) instead of
	// jumping like a quartz movement. See "Angles" above for the midnight
	// caveat that comes with it.
	Smooth bool

	// Face fills the dial and Ink draws the ticks, numerals and the hour and
	// minute hands; empty uses the theme's Surface and TextPrimary.
	// SecondColor draws the second hand and hub, and defaults to Primary so
	// the fastest-moving part is the one the eye can find.
	Face        string
	Ink         string
	SecondColor string

	// Style is applied last, to the stack.
	Style []core.StyleProp

	// AccessibilityLabel overrides the spoken time.
	AccessibilityLabel string
}

// Proportions of the dial, as fractions of its diameter. Kept together so the
// face can be re-balanced in one place; the relationships that matter are that
// the hour hand ends inside the numerals, the minute hand reaches the ticks,
// and the second hand overlaps them slightly.
const (
	clockTickInset      = 0.03  // rim to the outer end of a tick
	clockHourTickLen    = 0.07  // an hour tick
	clockQuarterTickLen = 0.1   // the 12, 3, 6 and 9 ticks
	clockMinuteTickLen  = 0.03  // a minute tick
	clockNumeralInset   = 0.15  // rim to the top of a numeral
	clockHourHandLen    = 0.25  // centre to tip
	clockMinuteHandLen  = 0.37  // centre to tip
	clockSecondHandLen  = 0.42  // centre to tip
	clockSecondTailLen  = 0.08  // centre to the counterweight end
	clockHubDiameter    = 0.055 // the cap over the pivot
)

func (c AnalogClock) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	size := c.Size
	if size <= 0 {
		size = 200
	}
	face := c.Face
	if face == "" {
		face = t.Colors.Surface
	}
	ink := c.Ink
	if ink == "" {
		ink = t.Colors.TextPrimary
	}
	accent := c.SecondColor
	if accent == "" {
		accent = t.Colors.Primary
	}

	hourDeg, minuteDeg, secondDeg := clockAngles(c.Time)

	// See "Angles": no Transition on the pass that crosses midnight.
	smooth := c.Smooth && secondsOfDay(c.Time) >= 1

	dim := px(size)
	items := make([]core.PropsAndChildren, 0, 80+len(c.Style))
	items = append(items,
		core.Width(dim),
		core.Height(dim),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(c.label()),
	)
	for _, sp := range c.Style {
		items = append(items, sp)
	}

	// The face: a square Box made round.
	items = append(items, core.Box(
		core.Width(dim),
		core.Height(dim),
		core.Padding(0),
		core.BorderRadius(size/2),
		core.BackgroundColor(face),
		core.BorderWidth(1),
		core.BorderColor(t.Colors.BorderColor()),
		core.AccessibilityHidden(),
	))

	// Ticks. Hour ticks every 30°, the four quarters longer and heavier so
	// the face can be read at a glance without numerals.
	for i := 0; i < 60; i++ {
		var length, width float64
		switch {
		case i%15 == 0:
			length, width = clockQuarterTickLen, max(2, size*0.015)
		case i%5 == 0:
			length, width = clockHourTickLen, max(1.5, size*0.01)
		case c.MinuteTicks:
			length, width = clockMinuteTickLen, max(1, size*0.005)
		default:
			continue
		}
		items = append(items, dialLayer(size, float64(i*6), false,
			core.Box(core.Padding(0), core.Height(px(size*clockTickInset))),
			core.Box(
				core.Padding(0),
				core.Width(px(width)),
				core.Height(px(size*length)),
				core.BackgroundColor(ink),
			),
		))
	}

	if c.Numerals {
		numeral := max(10, size*0.08)
		for h := 1; h <= 12; h++ {
			deg := float64(h * 30)
			items = append(items, dialLayer(size, deg, false,
				core.Box(core.Padding(0), core.Height(px(size*clockNumeralInset-numeral*0.3))),
				// Counter-rotated about its own centre, so the numeral stays
				// upright while its position goes round with the layer.
				core.Text(strconv.Itoa(h),
					core.FontSize(numeral),
					core.TextColor(ink),
					core.Rotate(-deg),
				),
			))
		}
	}

	items = append(items,
		handLayer(size, clockHourHandLen, 0, max(3, size*0.035), ink, hourDeg, smooth),
		handLayer(size, clockMinuteHandLen, 0, max(2, size*0.022), ink, minuteDeg, smooth),
	)
	if c.ShowSeconds {
		items = append(items,
			handLayer(size, clockSecondHandLen, clockSecondTailLen, max(1, size*0.008), accent, secondDeg, smooth))
	}

	// The hub, over the hands' inner ends. A ZStack centres a layer that says
	// nothing about its alignment, which is exactly the pivot.
	hub := size * clockHubDiameter
	items = append(items, core.Box(
		core.Padding(0),
		core.Width(px(hub)),
		core.Height(px(hub)),
		core.BorderRadius(hub/2),
		core.BackgroundColor(accent),
		core.AccessibilityHidden(),
	))

	return core.ZStack(items...).Render(ctx)
}

// label is the spoken time. Seconds are included only when the second hand
// is drawn: a listener asking a face without one has not been shown them.
func (c AnalogClock) label() string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	digits, marker := clockDigits(c.Time, false, c.ShowSeconds)
	return digits + " " + marker
}

// dialLayer is one full-dial layer turned by deg, holding children stacked
// from the top edge down and centred horizontally. animate adds the hands'
// Transition. See the AnalogClock doc for why
// every part of the face is drawn this way.
func dialLayer(size, deg float64, animate bool, children ...core.PropsAndChildren) core.View {
	items := make([]core.PropsAndChildren, 0, 8+len(children))
	items = append(items,
		core.Width(px(size)),
		core.Height(px(size)),
		core.Padding(0),
		core.Gap(0),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Rotate(deg),
		core.AccessibilityHidden(),
	)
	if animate {
		// Shorter than the one-second step so the hand settles before the
		// next tick, and ease-out so it snaps forward like a sprung movement
		// rather than drifting.
		items = append(items, core.Transition(250, core.EaseOut))
	}
	items = append(items, children...)
	return core.Column(items...)
}

// handLayer draws one hand: length and tail are fractions of the diameter,
// width is in px. The spacer pushes the stick down so its lower end passes the
// centre by tail.
func handLayer(size, length, tail, width float64, color string, deg float64, animate bool) core.View {
	return dialLayer(size, deg, animate,
		core.Box(core.Padding(0), core.Height(px(size/2-size*length))),
		core.Box(
			core.Padding(0),
			core.Width(px(width)),
			core.Height(px(size*(length+tail))),
			core.BorderRadius(width/2),
			core.BackgroundColor(color),
		),
	)
}

// clockAngles returns the three hands' angles, in degrees clockwise from
// twelve, unwrapped across the local day. See "Angles" on AnalogClock.
func clockAngles(tm time.Time) (hour, minute, second float64) {
	s := secondsOfDay(tm)
	return s / 120, s / 10, s * 6
}

// secondsOfDay is whole seconds since local midnight. Whole, because a hand
// that carried the fraction would sit a few degrees past its mark whenever a
// tick arrived late, and a quartz second hand points at marks.
//
// Computed from the wall-clock fields rather than tm.Sub(midnight), which on a
// daylight-saving day is an hour off from what the clock face should show.
func secondsOfDay(tm time.Time) float64 {
	h, m, s := tm.Clock()
	return float64(h*3600 + m*60 + s)
}

// px formats a length for the Width/Height setters.
func px(v float64) string {
	return fmt.Sprintf("%gpx", v)
}
