package comps

import (
	"fmt"
	"math"

	"github.com/rohanthewiz/grmob/core"
)

// Timeline is a vertical list of events joined by a line down the leading
// edge, a dot per event: order tracking, an activity feed, a changelog.
//
//	comps.Timeline{
//	    Label: "Order history",
//	    Events: []comps.TimelineEvent{
//	        {Time: "09:12", Title: "Order placed"},
//	        {Time: "11:40", Title: "Packed", Subtitle: "Warehouse 3"},
//	        {Time: "14:05", Title: "Out for delivery", Variant: comps.VariantSuccess},
//	    },
//	}
//
// # The line is drawn per row, and that is the hard part
//
// No renderer draws a line that spans siblings, so each row draws its own
// piece of it. The row is a Row with AlignItems stretch, so its leading rail
// column is exactly as tall as the event's text, and the rail is three parts:
//
//	┌ Row  AlignItems(stretch)  role=listitem ──────────────────┐
//	│ ┌ rail Column ┐ ┌ body Column FlexGrow(1) ───────────────┐ │
//	│ │  │ top      │ │ 11:40                   (Time)         │ │
//	│ │  ●  dot     │ │ Packed                  (Title)        │ │
//	│ │  │          │ │ Warehouse 3             (Subtitle)     │ │
//	│ │  │ bottom   │ │ [Content]                              │ │
//	│ │  │ FlexGrow │ │                    PaddingBottom(MD)   │ │
//	│ └─────────────┘ └────────────────────────────────────────┘ │
//	└────────────────────────────────────────────────────────────┘
//
// The top segment is a fixed height that puts the dot's centre on the centre
// of the body's first line. The bottom segment grows to the row's full
// height, and the row's spacing below an event is the body's bottom padding,
// not a gap between rows, so the bottom segment runs through it and meets the
// next row's top segment with no break. The first row's top segment and the
// last row's bottom segment keep their size and paint nothing, so every dot
// sits at the same offset.
//
// # Why not ListRow
//
// The plan sketched this as ListRow with a stretched leading slot. ListRow
// centres its slots by default (overridable) and, more to the point, carries
// the theme's Row padding above and below. That padding sits outside the
// rail, so the line would break between every pair of rows. The rows here are
// plain Rows with Padding(0), and the spacing moves inside the body where the
// rail can run through it.
//
// Cross-axis stretch on a Row is honoured on every target: the web by
// align-items, Compose through stretchRowHeight's intrinsic measurement and
// fillMaxHeight, and iOS through the flex layout's stretch. The native half
// comes from reading Renderer.kt and GrMobFlex.swift, not from a device run.
//
// # Accessibility
//
// The Timeline is RoleList and each event RoleListItem, so a reader announces
// "list, 3 items" and moves event by event; Label names the list. The rail is
// drawing and is hidden from assistive technology. A list item holds no
// button role, so an event is read, not operated; put a control in Content if
// an event needs one.
//
// # Theme roles read
//
//	Line       Colors.BorderColor()
//	Dot        Variant.Color — Colors.Primary unless the event sets a Variant
//	Time       Typography.Caption
//	Title      Typography.Body, bold
//	Subtitle   Typography.Caption
//	Spacing    Spacing.SM between rail and body, Spacing.MD below an event
type Timeline struct {
	// Events are drawn top to bottom, oldest or newest first as the caller
	// orders them.
	Events []TimelineEvent

	// Label is the list's accessible name.
	Label string

	// Style is applied to the list column after the widget's own props.
	Style []core.StyleProp
}

// TimelineEvent is one row of a Timeline.
type TimelineEvent struct {
	// Time is an optional caption above the title: "09:12", "Yesterday".
	Time string

	// Title is the event's primary line.
	Title string

	// Subtitle is the quieter line under the title.
	Subtitle string

	// Content is an optional view under the text: a thumbnail, a quote, a
	// button.
	Content core.View

	// Variant colours the dot. The zero value is Primary.
	Variant Variant
}

// timelineDot is the dot's diameter; timelineLine is the line's width.
const (
	timelineDot  = 12
	timelineLine = 2
)

// Render builds Column(list) > Row(listitem)... as drawn in the type doc.
func (tl Timeline) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(tl.Style)+len(tl.Events)+4)
	items = append(items,
		core.Padding(0),
		core.Gap(0),
		core.AccessibilityRole(core.RoleList),
	)
	if tl.Label != "" {
		items = append(items, core.AccessibilityLabel(tl.Label))
	}
	for _, sp := range tl.Style {
		items = append(items, sp)
	}
	last := len(tl.Events) - 1
	for i, ev := range tl.Events {
		items = append(items, tl.row(t, ev, i == 0, i == last))
	}
	return core.Column(items...).Render(ctx)
}

// row builds one event: the rail and the body, stretched to one height.
func (tl Timeline) row(t *core.Theme, ev TimelineEvent, first, last bool) core.View {
	return core.Row(
		// Stretch is what makes the rail as tall as the body. A Row's cross
		// axis is read from AlignItems alone on every target, so it is spelled
		// out rather than left to a default.
		core.AlignItemsProp(core.AlignItemsStretch),
		core.Gap(float64(t.Spacing.SM)),
		// Padding(0): the theme Row's padding would sit outside the rail and
		// break the line between rows. See "Why not ListRow".
		core.Padding(0),
		core.AccessibilityRole(core.RoleListItem),
		tl.rail(t, ev, first, last),
		tl.body(t, ev, last),
	)
}

// rail builds the leading column: top segment, dot, growing bottom segment.
func (tl Timeline) rail(t *core.Theme, ev TimelineEvent, first, last bool) core.View {
	line := t.Colors.BorderColor()
	px := func(v int) string { return fmt.Sprintf("%dpx", v) }

	top := []core.PropsAndChildren{core.Width(px(timelineLine)), core.Height(px(tl.railTop(t, ev)))}
	if !first {
		top = append(top, core.BackgroundColor(line))
	}
	bottom := []core.PropsAndChildren{core.Width(px(timelineLine)), core.FlexGrow(1)}
	if !last {
		bottom = append(bottom, core.BackgroundColor(line))
	}

	return core.Column(
		core.Width(px(timelineDot)),
		core.FlexShrink(0),
		core.Padding(0),
		core.Gap(0),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.AccessibilityHidden(),
		core.Box(top...),
		core.Box(
			core.Width(px(timelineDot)),
			core.Height(px(timelineDot)),
			core.FlexShrink(0),
			core.BorderRadius(timelineDot/2),
			core.BackgroundColor(ev.Variant.Color(t)),
		),
		core.Box(bottom...),
	)
}

// railTop is the top segment's height: half the body's first line, less half
// the dot, so the dot is centred on that line. The first line is the Time
// caption when there is one, else the title. A style with no LineHeight is
// measured at 1.2 times its font size, the browsers' "normal" and close to
// both natives' default.
func (tl Timeline) railTop(t *core.Theme, ev TimelineEvent) int {
	first := t.Typography.Body
	if ev.Time != "" {
		first = t.Typography.Caption
	}
	lh := float64(first.LineHeight)
	if lh == 0 {
		lh = first.FontSize * 1.2
	}
	return max(0, int(math.Round(lh/2-timelineDot/2.0)))
}

// body builds the growing text column. Its bottom padding is the space below
// the event, inside the row so the rail runs through it; the last event needs
// none.
func (tl Timeline) body(t *core.Theme, ev TimelineEvent, last bool) core.View {
	items := []core.PropsAndChildren{
		core.FlexGrow(1),
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
	}
	if !last {
		items = append(items, core.PaddingBottom(t.Spacing.MD))
	}
	if ev.Time != "" {
		items = append(items, core.Text(ev.Time, core.UseStyle(t.Typography.Caption)))
	}
	if ev.Title != "" {
		items = append(items, core.Text(ev.Title,
			core.UseStyle(t.Typography.Body),
			core.FontWeight(core.Bold),
		))
	}
	if ev.Subtitle != "" {
		items = append(items, core.Text(ev.Subtitle, core.UseStyle(t.Typography.Caption)))
	}
	if ev.Content != nil {
		items = append(items, ev.Content)
	}
	return core.Column(items...)
}
