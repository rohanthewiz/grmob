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
//
// # Which state is the loud one
//
// Selected is the prominent state: the theme's Button base — a solid fill with
// the base's own label colour. Unselected is the quiet one: a Surface fill,
// TextPrimary ink and a hairline rule.
//
// That is the reverse of what this widget shipped with, and the reversal is
// the whole of the change. The original default painted the *selected* chip
// Surface-on-Primary and left the unselected chips on the solid Button base,
// so a filter row read inverted — the four options the reader had not chosen
// shouted, and the one they had chosen receded. It was what the doc comment
// said it was, so it was a design call rather than a bug, but three separate
// consumers reported the same surprise on first sight of the rendered output,
// which is the point at which a design call is wrong.
//
// A caller who wants the old look back has it in one field:
//
//	components.Chip{Label: l, Selected: sel, OnTap: f,
//	    SelectedStyle: []core.StyleProp{
//	        core.BackgroundColor(t.Colors.Surface), core.TextColor(t.Colors.Primary),
//	    },
//	    UnselectedStyle: []core.StyleProp{}, // empty, not nil: "apply nothing"
//	}
//
// # The selected default restates the theme's own Button colors
//
// It sets the fill and the ink to the values Components.Button already
// carries, so the look is the theme's, not a re-derivation of it. That is the
// rule Button's zero value follows, for the reason it gives: a theme's Button
// base is allowed to differ from Colors.Primary, and a widget rebuilding the
// fill out of the palette would silently overwrite that choice in a way the
// bundled themes — where the two agree — cannot show.
//
// Restating rather than simply letting the base show through is what makes
// "the state wins over Style" true on this side as well. A color the selected
// default never set could not beat one in Style, so a strip handed a single
// shared Style{BackgroundColor(x)} would paint x on the selected chip and the
// quiet fill on all the others — the inversion again, by a different route.
//
// The border is there for geometry, not for decoration. Only the unselected
// chip wants a visible rule, but a rule on one state alone makes that state
// 2px wider and taller wherever box-sizing is content-box (the static export
// sets no reset, so it is), and a pill that grows when you tap it is a worse
// artifact than the one this fixes. So both states carry a 1px border and the
// selected one paints it in its own fill, where it cannot be seen.
type Chip struct {
	Label    string
	Selected bool
	OnTap    func()

	// Style is applied to every chip, selected or not, before the state's
	// own styling. The state wins on any field both set — otherwise one
	// Style shared across a strip would flatten the very distinction the
	// strip exists to draw. Use it for the properties that are the same in
	// both states (radius, padding, font) and the two state fields below for
	// the ones that are not.
	Style []core.StyleProp

	// SelectedStyle replaces the selected default described above; nil takes
	// the default. UnselectedStyle is its counterpart for the other state.
	//
	// Both distinguish nil from empty: a nil slice means "use the theme
	// default", an allocated but empty one means "apply nothing", which is
	// how a caller drops a default rather than overriding it.
	SelectedStyle   []core.StyleProp
	UnselectedStyle []core.StyleProp

	// AccessibilityLabel names the chip for screen readers; when Selected,
	// ", selected" is appended so state is announced with the name.
	// AccessibilityHint describes the effect of tapping.
	AccessibilityLabel string
	AccessibilityHint  string
}

func (c Chip) Render(ctx *core.Context) *core.Node {
	state := c.stateStyle(ctx.Theme())

	styles := make([]core.StyleProp, 0, len(c.Style)+len(state)+2)
	styles = append(styles, c.Style...)
	if c.AccessibilityHint != "" {
		styles = append(styles, core.AccessibilityHint(c.AccessibilityHint))
	}
	styles = append(styles, state...)

	if c.AccessibilityLabel != "" {
		label := c.AccessibilityLabel
		if c.Selected {
			label += ", selected"
		}
		styles = append(styles, core.AccessibilityLabel(label))
	}

	// A nil OnTap becomes an explicit no-op rather than being handed to
	// core.Button as-is: the registry stores whatever it is given and
	// TriggerCallback invokes it unguarded, so a decorative Chip{Label: "x"}
	// panicked the moment it was tapped. Same guard Button and
	// SegmentedControl already apply.
	onTap := c.OnTap
	if onTap == nil {
		onTap = func() {}
	}

	return core.Button(c.Label, onTap, asProps(styles)...).Render(ctx)
}

// stateStyle resolves the one state's styling: the caller's override when it
// supplied one, otherwise the theme default for that state.
func (c Chip) stateStyle(t *core.Theme) []core.StyleProp {
	if c.Selected {
		if c.SelectedStyle != nil {
			return c.SelectedStyle
		}
		// The base's own two colours, restated rather than simply left to
		// show through. Leaving them to show through draws the same pixels
		// and is what this did first, but a colour that is never *set* cannot
		// win against Style: a strip handed one shared
		// Style{BackgroundColor(x)} got x on the selected chip and the quiet
		// fill on every other, inverting the row again by another route.
		// Restating makes "the state wins" true on both sides.
		//
		// Restating is not re-deriving: these read the base itself, so a
		// theme whose buttons are not primary-coloured still gets its own
		// look. A field the base leaves empty is skipped entirely — passing
		// "" to either prop would *clear* the base's value rather than
		// inherit it, since the prop setters assign unconditionally.
		base := t.Components.Button
		state := make([]core.StyleProp, 0, 4)
		if base.Background != "" {
			state = append(state, core.BackgroundColor(base.Background))
		}
		if base.TextColor != "" {
			state = append(state, core.TextColor(base.TextColor))
		}
		return append(state, core.BorderWidth(1), core.BorderColor(chipRing(t)))
	}
	if c.UnselectedStyle != nil {
		return c.UnselectedStyle
	}
	return []core.StyleProp{
		core.BackgroundColor(t.Colors.Surface),
		core.TextColor(t.Colors.TextPrimary),
		core.BorderWidth(1),
		core.BorderColor(t.Colors.BorderColor()),
	}
}

// chipRing is the color of the selected chip's invisible border: its own
// fill, read off the theme's Button base rather than re-derived from
// Colors.Primary, so a theme whose buttons are not primary-colored still gets
// a ring that disappears into the pill.
//
// A base with no fill of its own falls back to a fully transparent border
// rather than to a guessed color. Transparent is what "no ring" means, and it
// still occupies the pixel, which is the only thing this border is for; an
// empty BorderColor would instead be dropped by the exporters and take the
// pixel with it.
func chipRing(t *core.Theme) string {
	if fill := t.Components.Button.Background; fill != "" {
		return fill
	}
	return ColorTransparent
}
