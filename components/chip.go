package components

import "github.com/rohanthewiz/grmob/core"

// Prominence is how loudly a Chip's *unselected* state asserts itself. It is
// the answer to a question the widget shipped with one answer to and which
// turns out to have two, both right.
//
//	the sermons year filter    a set of options, most of them not chosen. The
//	                           row is chrome above the list it filters, and a
//	                           loud row of years competes with the archive.
//	                           Quiet.
//
//	a giving form's suggested  four ways to answer the screen's only question,
//	amounts                    and the fast path most gifts take. A row of grey
//	                           pills over an empty amount field does not read
//	                           as "tap one of these". Loud.
//
// Material draws exactly this distinction — a *filter* chip against a
// *suggestion* chip — with different default prominence for each. This field
// is that distinction, arriving here because the first two consumers of the
// widget wanted one each and the second had to spell its treatment by hand.
//
// # Why this is a new type rather than Button's Emphasis
//
// Emphasis is the nearest thing in the package and is deliberately not reused,
// for two reasons that both bite.
//
// Its zero value is EmphasisFilled — the loud one — because a Button with no
// opinion is a solid button. Chip's zero has to stay quiet, because that is
// the look every chip in an existing tree already has and the field must be a
// no-op. Sharing the type would mean the same zero value meaning opposite
// things in two widgets a page apart.
//
// And Emphasis describes a whole control, where this describes *one of a
// chip's two states*. EmphasisGhost has no meaning here: a chip with no box is
// indistinguishable from a run of text, and the selected/unselected pair is
// exactly what a chip exists to draw.
type Prominence string

const (
	// ProminenceQuiet is the zero value: a Surface fill, TextPrimary ink and
	// a hairline rule. Right for a filter row, which is chrome above the
	// content it filters.
	ProminenceQuiet Prominence = ""

	// ProminenceLoud draws the unselected chip as an outline in the chip's
	// own accent — the accent as ink and as a 1px rule over a transparent
	// fill, which is what EmphasisOutlined spends on a secondary button.
	//
	// It is not a return to the pre-inversion look, and the difference is the
	// whole point: the old default painted every unselected chip a *solid*
	// fill and left the chosen one pale. Here the selected chip keeps its
	// solid fill while its neighbours are outlines, so the row says both
	// "pick one of these" and "this is the one you picked".
	ProminenceLoud Prominence = "loud"
)

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
// # How loud the quiet state is, is a second question
//
// Which state is louder is settled above and is not negotiable — that was the
// bug. *How much* quieter the other one is has two right answers, and
// Prominence is the field that picks: quiet (the default, a Surface fill) for
// a filter row, loud (an outline in the chip's own accent) for a row of
// suggestions the reader is meant to reach into. See Prominence.
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

	// Prominence tunes the unselected state alone: quiet (zero) or loud. It
	// says nothing about the selected chip, which is the theme's Button base
	// in both — there is nothing louder to give it, and the row must keep
	// reading as "this is the one you picked" either way.
	//
	// UnselectedStyle still wins where it is set, being the more specific of
	// the two: Prominence picks between the widget's own treatments, that
	// replaces them.
	Prominence Prominence

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
	// The caller's own override is checked before Prominence, not after: the
	// two are the same knob at different resolutions — Prominence picks among
	// the widget's treatments, UnselectedStyle replaces them — so the more
	// specific one has to be reached first or it could never be reached at
	// all.
	if c.UnselectedStyle != nil {
		return c.UnselectedStyle
	}
	if c.Prominence == ProminenceLoud {
		// The outlined treatment, in the chip's own accent rather than in
		// Colors.Primary: the accent is the fill the *selected* chip paints,
		// so the outline and the fill it turns into are the same hue by
		// construction. Re-deriving from the palette would split them apart
		// on any theme whose buttons are not primary-coloured.
		//
		// Legibility here is the palette's, not the widget's, and for the
		// reason Button's doc gives at length: a transparent fill means the
		// label's real backdrop is whatever the chip was placed on, which the
		// widget cannot see. The accent is the theme author's own hex, spent
		// verbatim rather than darkened until it passes.
		//
		// The bundled numbers are exactly Button's outlined "default" row,
		// since it is the same colour on the same backdrop — 4.02:1 under
		// DefaultTheme (#007AFF on white) and 7.63:1 under MaterialTheme. So
		// the second clears WCAG AA at this font size and the first does not,
		// and a screen leaning on loud chips under a DefaultTheme-like
		// palette wants a darker TextColor through UnselectedStyle. The
		// durable fix is the same one Button names: a second palette value
		// per role, an "on-light" tone, which is a theme's decision and not
		// this widget's.
		accent := chipAccent(t)
		return []core.StyleProp{
			core.BackgroundColor(ColorTransparent),
			core.TextColor(accent),
			core.BorderWidth(1),
			core.BorderColor(accent),
		}
	}
	return []core.StyleProp{
		core.BackgroundColor(t.Colors.Surface),
		core.TextColor(t.Colors.TextPrimary),
		core.BorderWidth(1),
		core.BorderColor(t.Colors.BorderColor()),
	}
}

// chipAccent is the chip's own colour: the fill its selected state paints,
// read off the theme's Button base for the same reason the selected default
// reads it rather than rebuilding it from Colors.Primary.
//
// Its fallback differs from chipRing's, and deliberately. That one falls back
// to transparent because a ring nobody can see is precisely what it wants when
// there is no fill to hide against. This one is *ink*, and transparent ink is
// an invisible chip, so a theme with no Button fill of its own falls back to
// the palette's Primary — the same colour Button's VariantDefault spends.
func chipAccent(t *core.Theme) string {
	if fill := t.Components.Button.Background; fill != "" {
		return fill
	}
	return t.Colors.Primary
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
// pixel with it. See chipAccent, which reads the same base for a rule that is
// meant to be seen and so cannot take that fallback.
func chipRing(t *core.Theme) string {
	if fill := t.Components.Button.Background; fill != "" {
		return fill
	}
	return ColorTransparent
}
