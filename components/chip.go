package components

import "github.com/rohanthewiz/grmob/core"

// Chip is a selectable pill — a filter toggle, a tag picker entry. It renders
// as a themed Button whose look shifts with Selected, so switching selection
// patches two style fields instead of restructuring the row (the pattern the
// todoapp filter bar established; this widget is that bar's extraction).
//
// Selection is controlled by the caller: Chip holds no state, it just renders
// Selected and reports taps through OnTap. A group of chips is therefore one
// piece of parent state plus a loop.
type Chip struct {
	Label    string
	Selected bool
	OnTap    func()

	// Style is applied to every chip, selected or not, after the theme's
	// Button base.
	Style []core.StyleProp
	// SelectedStyle is applied on top when Selected. When nil, a theme
	// default is used: Surface background with Primary ink — the muted
	// look that reads as "pressed in" next to the solid Button base.
	SelectedStyle []core.StyleProp

	// AccessibilityLabel names the chip for screen readers; when Selected,
	// ", selected" is appended so state is announced with the name.
	// AccessibilityHint describes the effect of tapping.
	AccessibilityLabel string
	AccessibilityHint  string
}

func (c Chip) Render(ctx *core.Context) *core.Node {
	styles := make([]core.StyleProp, 0, len(c.Style)+len(c.SelectedStyle)+3)
	styles = append(styles, c.Style...)
	if c.AccessibilityHint != "" {
		styles = append(styles, core.AccessibilityHint(c.AccessibilityHint))
	}

	if c.Selected {
		sel := c.SelectedStyle
		if sel == nil {
			t := ctx.Theme()
			sel = []core.StyleProp{
				core.BackgroundColor(t.Colors.Surface),
				core.TextColor(t.Colors.Primary),
			}
		}
		styles = append(styles, sel...)
	}

	if c.AccessibilityLabel != "" {
		label := c.AccessibilityLabel
		if c.Selected {
			label += ", selected"
		}
		styles = append(styles, core.AccessibilityLabel(label))
	}

	return core.Button(c.Label, c.OnTap, asProps(styles)...).Render(ctx)
}
