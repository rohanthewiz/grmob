package social

import (
	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
)

// TabButton is one glyph in the bottom tab bar.
//
// It is a ghost button, and that fixes a real bug rather than restyling
// anything. This used to be a bare core.Button with a TextColor override and
// no Background override — which meant it inherited the theme's Button base,
// a solid #007AFF pill. The active tab therefore rendered #007AFF on #007AFF,
// an invisible glyph at 1:1 contrast, and the inactive one #555 on #007AFF at
// about 2.6:1. Setting only half of a color pair is exactly the mistake the
// widget's emphasis axis exists to prevent: EmphasisGhost punches the fill out
// instead of leaving it to whatever the theme put there.
//
// Wrapped in a ComponentFunc so the inactive tint can come from the palette.
// The function returns a View before any render happens, so there is no theme
// to read until a Context arrives.
func TabButton(icon, tab string, selected core.State[string]) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		b := components.Button{
			Label:    icon,
			OnTap:    func() { selected.Set(tab) },
			Emphasis: components.EmphasisGhost,
			Style: []core.StyleProp{
				core.FontSize(20),
				core.Padding(12),
				core.Align(core.AlignCenter),
			},
		}
		// Ghost already inks the label with the variant's color — the theme's
		// Primary — which is the selected look. Only the unselected tab needs
		// to say anything, and it dims to the palette's secondary ink rather
		// than to the literal grey this carried before.
		if selected.Get() != tab {
			b.Style = append(b.Style, core.TextColor(ctx.Theme().Colors.TextSecondary))
		}
		return b.Render(ctx)
	})
}
