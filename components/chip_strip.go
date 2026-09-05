package components

import "github.com/rohanthewiz/grmob/core"

// ChipStrip lays out a run of Chips that wraps onto as many lines as it needs
// — a filter bar, the tags on an article, the scripture references on a
// sermon, the quick amounts on a giving form.
//
//	components.ChipStrip{Chips: []components.Chip{
//	    {Label: "All",      Selected: f == "",      OnTap: func() { filter.Set("") }},
//	    {Label: "Sermons",  Selected: f == "sermon", OnTap: func() { filter.Set("sermon") }},
//	    {Label: "Articles", Selected: f == "article", OnTap: func() { filter.Set("article") }},
//	}}
//
// # Why the field is []Chip and not a parallel vocabulary
//
// The tempting API is Labels []string plus Selected func(string) bool plus
// OnTap func(int) — and it is a second way to describe a chip, which then has
// to grow its own Style, its own AccessibilityLabel, its own everything as
// Chip does. Taking []Chip means the strip adds layout and nothing else: a
// chip configured here is configured exactly as a chip configured anywhere,
// and a caller who needs one of Chip's knobs already has it.
//
// # ChipStrip is not SegmentedControl
//
// SegmentedControl is one-of-N: a fixed, exhaustive set where exactly one
// option is live, drawn as a single joined control. ChipStrip is the loose
// case — any number selected including none, a set that comes from data and
// changes length, entries that may not be selectable at all. Reach for the
// segmented control when the options are a closed choice, this when they are
// a collection.
//
// # Wrapping or scrolling
//
// The default is to wrap: a strip with more chips than fit takes a second
// line. That is the right shape for a set the reader should see all of —
// the tags on an article, the scripture references on a sermon — and it
// costs the layout nothing.
//
// Scrollable is the other shape. A long filter bar reads better as one line
// that pans sideways, because a bar that grows to three lines pushes the
// content it filters off the screen, and the chips past the fold are a hint
// that there is more rather than a queue demanding to be read.
//
//	components.ChipStrip{Scrollable: true, Chips: years}
//
// The two are exclusive by construction — a scrolling strip is one line, so
// there is nothing to wrap — and Scrollable wins when both are asked for.
type ChipStrip struct {
	// Chips are the strip's contents, in order.
	Chips []Chip

	// Children is the escape hatch: arbitrary views in place of Chips, taking
	// precedence when set. For a strip mixing chips with something else — a
	// trailing "+ Add", a Badge among the tags — or for chips already built
	// by a helper of the caller's own. Nil entries are skipped.
	Children []core.View

	// Gap is the spacing between chips, both across and between lines. Zero
	// takes the theme's SM step.
	Gap float64

	// Scrollable makes the strip one line that pans sideways instead of a
	// block that wraps; see the type comment for when each is right.
	//
	// It is a core.Scroll carrying core.Horizontal, not a Row with an
	// overflow: the natives have no CSS to fall back on and implement
	// sideways panning in their scroll composites alone (Compose
	// horizontalScroll, SwiftUI ScrollView(.horizontal)), so the node type
	// has to change, not just a style. Style still lands on the strip
	// itself either way — the Scroll *is* the strip when this is set, so
	// there is no extra box to configure.
	Scrollable bool

	// Style is applied after the widget's own defaults, so the wrap, the gap
	// and the (removed) row padding are all overridable.
	Style []core.StyleProp
}

func (c ChipStrip) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	gap := c.Gap
	if gap == 0 {
		gap = float64(t.Spacing.SM)
	}

	items := make([]core.PropsAndChildren, 0, len(c.Chips)+len(c.Children)+len(c.Style)+3)
	if c.Scrollable {
		// core.Horizontal supplies the row axis and the overflow; the wrap is
		// deliberately absent, since a strip that pans has one line by
		// definition and `flex-wrap: wrap` inside a horizontal scroller is a
		// contradiction the browser resolves by wrapping, not by scrolling.
		items = append(items, core.Horizontal(), core.Gap(gap))
	} else {
		items = append(items, core.FlexWrap(true), core.Gap(gap))
	}
	items = append(items,
		// Assigns over the theme's Row inset rather than merging with it. A
		// strip of chips is a run of content inside a screen that is already
		// padded, not a row with margins of its own; leaving the base padding
		// on double-insets it against every sibling in the column.
		core.Padding(0),
	)
	for _, sp := range c.Style {
		items = append(items, sp)
	}

	if c.Children != nil {
		for _, child := range c.Children {
			if child != nil {
				items = append(items, child)
			}
		}
	} else {
		for _, chip := range c.Chips {
			items = append(items, chip)
		}
	}

	// A Scroll carries no theme base, so the Padding(0) above is a no-op on
	// that branch rather than a correction — kept unconditional because it is
	// the same statement about the strip either way, and a reader comparing
	// the two branches should find only the axis differing between them.
	if c.Scrollable {
		return core.Scroll(items...).Render(ctx)
	}
	return core.Row(items...).Render(ctx)
}
