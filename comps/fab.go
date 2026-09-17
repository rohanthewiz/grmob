package comps

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// FAB is the floating action button: the one primary action of a screen,
// drawn as a raised disc that sits over the content rather than in it.
//
//	comps.FAB{Icon: "+", AccessibilityLabel: "New note", OnTap: create}
//	comps.FAB{Icon: "✎", Label: "Compose", OnTap: compose}   // the extended form
//
// It is a comps.Button in a circle, and nothing more: the treatment
// (Variant × Emphasis, disabled dimming, the caller's Style landing after the
// widget's own) is Button's, so a FAB and a Button on the same screen cannot
// disagree about what "primary" looks like.
//
// # Where it floats
//
// The widget does not place itself. A floating thing needs a layer to float
// on, and the only container that draws one child over another is
// core.ZStack, which sizes to its largest layer — so a FAB that wrapped its
// own screen would have to know how big the screen is. The screen already
// knows: Screen.Floating takes the FAB and places it bottom-end over the
// content, above the Footer, and that is the intended way to use it.
//
//	comps.Screen{
//	    Scroll:   true,
//	    Children: []core.View{list},
//	    Floating: comps.FAB{Icon: "+", AccessibilityLabel: "Add", OnTap: add},
//	    Footer:   comps.BottomBar{...},
//	}
//
// A FAB placed anywhere else is an ordinary round button in the flow, which
// is also fine — a toolbar can hold one — it just does not float.
//
// # Two shapes
//
// Icon-only is a disc: Width and Height equal, BorderRadius half of that, no
// padding so the glyph centres. With Label set it is the *extended* FAB, a
// pill wide enough for the glyph and the word; the height is the same so the
// two forms sit at the same place on a screen and a label can be added
// without the button moving.
//
//	┌──────┐        ┌─────────────────┐
//	│  +   │        │  ✎  Compose     │
//	└──────┘        └─────────────────┘
//	  disc              extended
//
// The sizes are fixed points rather than the theme's spacing scale, because a
// FAB is a *touch target* first: 56 is the platform norm for the regular size
// and 40 for the small one, and neither sits on a scale whose steps are 4, 8,
// 16, 24, 32. Spacing does still set the extended form's horizontal padding.
//
// # Accessibility
//
// A "+" is a glyph, not a name, so an icon-only FAB needs AccessibilityLabel
// and the widget gives it nothing to fall back on: it would rather a screen
// reader say "plus" than say a wrong name the widget invented. The extended
// form is named by its Label like any button, and AccessibilityLabel replaces
// that where the word on the pill is too short to explain the action.
//
// # Theme roles read
//
//	Fill and ink   as comps.Button: the theme's Button base for the zero
//	               Variant, the variant's colour and contrast-picked ink
//	               otherwise
//	Padding        Spacing.MD, on the extended form's sides
type FAB struct {
	// Icon is the glyph on the disc, and leads the Label on the extended
	// form. A single character or emoji; the widget does not size an image.
	Icon string

	// Label turns the disc into a pill with the word beside the glyph.
	Label string

	// OnTap is the action. Nil leaves the button in place and inert.
	OnTap func()

	// Variant picks the fill. The zero value is the theme's own Button
	// pairing, which on every bundled theme is the primary fill.
	Variant Variant

	// Emphasis is passed through to the Button. The zero value is filled,
	// which is what a FAB is; an outlined or ghost FAB is unusual but not
	// forbidden.
	Emphasis Emphasis

	// Size picks the diameter. The zero value is FABRegular.
	Size FABSize

	// Disabled draws the button muted and drops taps, as Button does.
	Disabled bool

	// AccessibilityLabel is the spoken name. Required for an icon-only FAB;
	// on the extended form it replaces the Label as the name.
	AccessibilityLabel string

	// AccessibilityHint says what happens on activation, as on Button.
	AccessibilityHint string

	// Style is applied after the widget's own props and before Disabled, so
	// a look can be overridden and inertness cannot.
	Style []core.StyleProp

	// FocusRef lets the caller move focus to the button.
	FocusRef *core.FocusRef
}

// FABSize selects a FAB's height, and its width when it is a disc.
type FABSize string

const (
	// FABRegular is 56 points, the platform default; the zero value.
	FABRegular FABSize = ""
	// FABSmall is 40 points, for a secondary action beside a regular one or
	// a FAB in a dense layout.
	FABSmall FABSize = "small"
)

// The two diameters, in points. See "Two shapes" on the type for why these
// are constants rather than theme spacing steps.
const (
	fabRegularPt = 56.0
	fabSmallPt   = 40.0
)

// diameter returns the height in points, which is also the width of a disc.
func (f FAB) diameter() float64 {
	if f.Size == FABSmall {
		return fabSmallPt
	}
	return fabRegularPt
}

// Render builds the round (or pill) Button.
func (f FAB) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	d := f.diameter()
	px := func(v float64) string { return fmt.Sprintf("%gpx", v) }

	// The widget's own look, ahead of the caller's Style so any of it can be
	// overridden; Button appends the caller's Style after these.
	styles := make([]core.StyleProp, 0, len(f.Style)+8)
	styles = append(styles,
		core.Height(px(d)),
		core.BorderRadius(d/2),
		// The disc is raised: that is what makes it read as floating rather
		// than as a round button painted on the content. Every target reads
		// Style.Shadow.
		core.Shadow(fabElevation),
	)
	label := f.Icon
	if f.Label == "" {
		// A disc: as wide as it is tall, with no padding so the glyph sits in
		// the middle. Padding(0) matters because the theme's Button base
		// carries a horizontal inset that would push a fixed-width disc's
		// glyph off centre. The glyph is sized to the disc rather than to the
		// theme's button text — a body-sized "+" in a 56-point disc looks
		// lost. Only the disc: the extended form's word is button text and
		// keeps the theme's size, since the glyph and the word are one label
		// and cannot be sized apart.
		styles = append(styles, core.Width(px(d)), core.Padding(0), core.FontSize(d*fabGlyphRatio))
	} else {
		// Extended: the pill takes its width from its content, inset by the
		// theme's medium step on each side. Vertical padding is zeroed so the
		// stated Height, not the padding, is what fixes the pill's height.
		styles = append(styles, core.Padding(0), core.PaddingHorizontal(t.Spacing.MD))
		if f.Icon != "" {
			label = f.Icon + "  " + f.Label
		} else {
			label = f.Label
		}
	}
	styles = append(styles, f.Style...)

	return Button{
		Label:              label,
		OnTap:              f.OnTap,
		Variant:            f.Variant,
		Emphasis:           f.Emphasis,
		Disabled:           f.Disabled,
		AccessibilityLabel: f.AccessibilityLabel,
		AccessibilityHint:  f.AccessibilityHint,
		Style:              styles,
		FocusRef:           f.FocusRef,
	}.Render(ctx)
}

// fabElevation is the shadow depth. Higher than a Card's RoundedShadowBox (2)
// because the disc floats over content that may itself carry cards.
const fabElevation = 6

// fabGlyphRatio is the glyph's font size as a fraction of the diameter: 24
// points in a 56-point disc, which is the icon size the platform norm pairs
// with that disc.
const fabGlyphRatio = 24.0 / 56.0
