package components

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// hairlineColor is the default separator tint.
//
// It is a constant here rather than a theme lookup because ColorPalette has no
// Border entry yet (Primary, Secondary, Background, Surface, TextPrimary,
// TextSecondary, Error — and nothing for a rule). Surface is the nearest
// available neutral but it is a *fill* color: on a Surface-colored panel a
// Surface hairline is invisible. So the value stays literal until the palette
// grows a Border role, at which point this constant becomes its fallback and
// Separator reads ctx.Theme() like every other widget.
//
// The point of pinning it here is that the two examples that draw a hairline
// each hardcoded this same value in their own package. One definition is the
// improvement available today; the themed one needs the palette change.
const hairlineColor = "#E5E5EA"

// Separator is the hairline rule between list rows and between sections.
//
// core.Divider already draws a line, but it takes a literal color and
// force-applies Margin(8) — an unconditional gap that makes it wrong inside a
// list, which is presumably why neither example that wanted a rule used it.
// Separator leaves spacing to the caller and defaults its own color, so the
// common case is the zero value:
//
//	components.Separator{}
//
// It is always hidden from assistive technology. A rule is decoration: it
// carries no information a screen reader can use, and announcing one between
// every pair of rows in a list turns a 20-row feed into 39 utterances.
//
// # Horizontal only
//
// There is deliberately no Vertical field. A vertical rule has to stretch to
// its row's height, which means cross-axis stretch — and neither renderer maps
// AlignItems "stretch" (Compose falls through to Alignment.Top, SwiftUI to
// .top). A vertical separator would therefore collapse to zero height on both
// platforms. Adding the field would be advertising something that does not
// work; it can land with the renderer support.
type Separator struct {
	// Color overrides the hairline tint. Empty uses hairlineColor.
	Color string

	// Thickness is the rule's height in px; 0 means 1. Fractional values are
	// carried through to the platforms as-is (a "0.5px" hairline is a real
	// request on a 2x/3x display, and both renderers parse a float).
	Thickness float64

	// Inset indents the rule from both ends, in px. This is the list idiom
	// where the rule starts under the text rather than under the leading
	// avatar or checkbox, so the leading column reads as one continuous
	// stripe. It is applied as left/right margin rather than
	// EdgeInsets.Horizontal because the HTML exporter only reads the
	// per-side fields.
	Inset int

	// Style is applied last, so every default above is overridable.
	Style []core.StyleProp
}

func (s Separator) Render(ctx *core.Context) *core.Node {
	color := s.Color
	if color == "" {
		color = hairlineColor
	}
	thickness := s.Thickness
	if thickness == 0 {
		thickness = 1
	}

	// Box, not Row or Column: it is the only container with no theme base, so
	// the rule inherits no padding to fight with. A themed container would
	// need Padding(0) to undo its own defaults before it could be 1px tall.
	items := make([]core.PropsAndChildren, 0, len(s.Style)+4)
	items = append(items,
		core.Height(fmt.Sprintf("%gpx", thickness)),
		core.BackgroundColor(color),
		core.AccessibilityHidden(),
	)
	if s.Inset != 0 {
		items = append(items, core.UseStyle(core.Style{
			Margin: core.EdgeInsets{Left: s.Inset, Right: s.Inset},
		}))
	}
	for _, sp := range s.Style {
		items = append(items, sp)
	}

	return core.Box(items...).Render(ctx)
}
