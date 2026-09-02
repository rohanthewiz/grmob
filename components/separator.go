package components

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// Separator is the hairline rule between list rows and between sections.
//
// core.Divider already draws a line, but it takes a literal color and
// force-applies Margin(8) — an unconditional gap that makes it wrong inside a
// list, which is presumably why neither example that wanted a rule used it.
// Separator leaves spacing to the caller and takes its tint from the theme's
// Border role, so the common case is the zero value:
//
//	components.Separator{}
//
// The tint reads through ColorPalette.BorderColor rather than off the Border
// field, so a theme written before that role existed draws a visible default
// hairline instead of an invisible one. Surface would have been the nearest
// pre-existing neutral and is the wrong answer: it is a *fill* color, so on a
// Surface-colored panel a Surface hairline disappears.
//
// It is always hidden from assistive technology. A rule is decoration: it
// carries no information a screen reader can use, and announcing one between
// every pair of rows in a list turns a 20-row feed into 39 utterances.
//
// # Horizontal only
//
// There is no Vertical field yet. A vertical rule has to stretch to its row's
// height, which means cross-axis stretch, and that used to be the blocker:
// neither renderer mapped AlignItems "stretch", so the field would have
// advertised something that collapsed to zero height on both platforms.
//
// Both renderers map it today — Compose pins a stretched Row to
// IntrinsicSize.Max and gives each child fillMaxHeight(), and SwiftUI's
// GrMobFlexStack proposes the full cross extent to a stretched child — so the
// field is now a widget change rather than a renderer one, waiting on a caller
// that wants it. One asymmetry to know when it lands: a Row reads alignItems
// alone and never the simpler Align fallback (Align is a text-alignment
// concept and has never applied to a row's vertical axis), so the containing
// row has to spell out AlignItems "stretch".
type Separator struct {
	// Color overrides the hairline tint. Empty takes the theme's Border role.
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
		color = ctx.Theme().Colors.BorderColor()
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
