package components

import "github.com/rohanthewiz/grmob/core"

// Card is a surface with optional header, body, and footer regions, rendered
// on core.Card (so it inherits the theme's Card base: background, padding,
// radius, shadow).
//
// Title is the simple path — a themed title line; Header is the escape hatch
// — an arbitrary view in the same position, taking precedence over Title when
// both are set. This mirrors element's Card, whose Body (string) and
// BodyComponent (component) coexist the same way.
type Card struct {
	Title  string
	Header core.View // overrides Title when set
	Body   core.View
	Footer core.View
	// Style is applied to the card container after the theme's Card base.
	Style []core.StyleProp
}

func (c Card) Render(ctx *core.Context) *core.Node {
	// Build the mixed style/children argument list core.Card expects. Order
	// matters only for children (header, body, footer top to bottom);
	// containerNode applies style props to the container wherever they
	// appear.
	items := make([]core.PropsAndChildren, 0, len(c.Style)+3)
	for _, sp := range c.Style {
		items = append(items, sp)
	}

	switch {
	case c.Header != nil:
		items = append(items, c.Header)
	case c.Title != "":
		// The theme's Subtitle role, not Title: a card heading is a section
		// heading within a screen, one level below the screen's own title.
		items = append(items, core.Text(c.Title,
			core.UseStyle(ctx.Theme().Typography.Subtitle),
			core.FontWeight(core.Bold),
		))
	}
	if c.Body != nil {
		items = append(items, c.Body)
	}
	if c.Footer != nil {
		items = append(items, c.Footer)
	}

	return core.Card(items...).Render(ctx)
}
