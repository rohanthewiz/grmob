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
	// a ring in the theme's control-boundary tone. Right for a filter row,
	// which is chrome above the content it filters.
	//
	// "Quiet" is about the fill and the ink. The ring is not part of what
	// recedes — it is the only thing that says the pill is a control, and it
	// used to be drawn in the divider role, which made a chip that receded
	// out of sight rather than into the background. See stateStyle.
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
// TextPrimary ink and a ring in the theme's control-boundary tone.
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
// Neither answer is "invisible". Both treatments draw a 1px ring at
// control-boundary weight — the loud one in the chip's own accent, the quiet
// one in Colors.ControlBorder — because WCAG 1.4.11 puts a 3:1 floor under the
// edge that identifies a control, and a chip's fill clears it in neither
// state. See stateStyle for the numbers.
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

	// AccessibilityLabel names the chip for screen readers.
	// AccessibilityHint describes the effect of tapping.
	//
	// The name no longer carries the state. Until core.Style had a slot for
	// one, this widget appended ", selected" to whatever named the chip,
	// because a name was the only channel it had; the state now goes out as
	// core.AccessibilitySelected and reaches every target as the platform's
	// own idea of a control being on (aria-pressed on both web targets, a
	// `selected` semantics property in Compose, the .isSelected trait in
	// SwiftUI).
	//
	// Both would announce it twice, which is the reasoning Button's Disabled
	// field already gives for dropping its own ", disabled" suffix. It is
	// also the better half to keep: a name is meant to be stable, so a reader
	// re-announcing the control after a tap read out the whole altered name,
	// where a state change is announced as a state change.
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

	// Stated on every chip, selected or not. SelectedOff is not the same as
	// saying nothing: a strip in which only the chosen chip answers announces
	// its neighbours as plain buttons, so the reader hears one toggle among
	// several pieces of furniture rather than one of five options. See
	// core.SelectedState.
	//
	// It needs no core.AccessibilityRole beside it because a Chip renders as
	// a core.Button, and a <button> already is one — the node type carries
	// the role, which is what both web exporters check when the style names
	// none. A caller who makes a strip of these into a tab bar sets
	// core.RoleTab through Style, and the same state then goes out as
	// aria-selected instead; nothing here has to know which.
	styles = append(styles, core.AccessibilitySelected(core.SelectedWhen(c.Selected)))

	if c.AccessibilityLabel != "" {
		styles = append(styles, core.AccessibilityLabel(c.AccessibilityLabel))
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
		// widget cannot see. So the outline is drawn in the accent's
		// *on-light* tone — the palette's second value per role, dark enough
		// to be read as ink on a light surface.
		//
		// The bundled numbers are Button's outlined "default" row, since it
		// is the same colour on the same backdrop: 7.56:1 under DefaultTheme
		// (up from 4.02:1, which missed WCAG AA at this font size) and
		// 7.63:1 under MaterialTheme, whose blue needed no second tone. A
		// theme that declares none falls back to the accent itself, which is
		// what this painted before the palette had one.
		//
		// The lookup is by *colour*, not by role, and that is forced by the
		// line above: the accent is read off the theme's Button base rather
		// than off Colors.Primary, precisely so a theme whose buttons are not
		// primary-coloured keeps its own look — which leaves this widget
		// holding a hex and no name for it. Colors.OnLight is the reverse
		// lookup for that position, and a base fill that is not one of the
		// palette's toned roles comes back unchanged, which is the same
		// fallback and the same pixels as before.
		//
		// Both the label and the rule take it, rather than tinting the rule
		// separately. See Button's colorProps for why that is a coherence
		// argument and not a contrast one — and note it means the outline and
		// the fill the *selected* chip paints are now two weights of one hue
		// rather than the same value. They are still the same hue by
		// construction, which is what stopped the two from drifting apart on
		// a theme whose buttons are not primary-coloured.
		accent := t.Colors.OnLight(chipAccent(t))
		return []core.StyleProp{
			core.BackgroundColor(ColorTransparent),
			core.TextColor(accent),
			core.BorderWidth(1),
			core.BorderColor(accent),
		}
	}
	// The quiet treatment: a Surface fill, TextPrimary ink, and a ring in the
	// theme's *boundary* tone rather than its divider.
	//
	// The ring used to be Colors.Border, and the whole pill was very close to
	// invisible as a control because of it. A quiet chip has two channels for
	// saying "there is something here to tap" and both were spent on hairline
	// values:
	//
	//	                    Default          Material
	//	fill vs page        1.12:1           1.09:1
	//	old ring vs page    1.26:1           1.32:1
	//	new ring vs page    3.26:1           4.61:1
	//
	// WCAG 1.4.11 (Non-text Contrast) is the line, and it is the same one the
	// field frames were moved for one session earlier: a rule *between* things
	// is decoration, while the edge that identifies a control carries a 3:1
	// floor. A filter row is chrome and is *meant* to recede — that is what
	// ProminenceQuiet means — but receding is a matter of how loud the fill
	// and the ink are, not of whether the control can be found at all.
	//
	// Colors.ControlBorderColor, not Components.Input.BorderColor. The two
	// hold the same hex in both bundled themes and reading the Input base
	// would have got the right pixels today, at the cost of tying a chip's
	// edge to a text field's: a theme that restyled its fields would have
	// silently restyled its chips. The role is the thing both of them name.
	//
	// One number under the floor, stated rather than rounded off. A chip has
	// two backdrops — the page behind it and its own Surface fill — and under
	// DefaultTheme the ring is 3.26:1 against the first and 2.92:1 against the
	// second. The edge that identifies the pill is the outer one: the fill is
	// 1.12:1 against the page and identifies nothing, so what a reader picks
	// the control out by is the ring against the page, which clears. The
	// inner edge is the boundary between two parts of one control. Closing
	// that last 0.08 would mean darkening the theme's boundary tone past
	// Apple's own systemGray, which is the same repaint-the-theme's-choice
	// move the on-light tones were added to avoid.
	return []core.StyleProp{
		core.BackgroundColor(t.Colors.Surface),
		core.TextColor(t.Colors.TextPrimary),
		core.BorderWidth(1),
		core.BorderColor(t.Colors.ControlBorderColor()),
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
