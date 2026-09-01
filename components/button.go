package components

import "github.com/rohanthewiz/grmob/core"

// ColorTransparent is a fully transparent fill, written in the CSS byte order
// (#RRGGBBAA) that every target parses: htmlout emits it verbatim as a CSS
// Color 4 hex, and both native parseColor implementations handle the 8-digit
// form with alpha last.
//
// It exists because core.Style has no "clear" or "unset" for a color, and an
// *empty* Background is not transparent — it means "inherit the theme's
// Button base", which is a solid Primary fill. Outlined and Ghost need an
// actual hole, not an omission.
const ColorTransparent = "#00000000"

// Emphasis is how strongly a button asserts itself: how much of the variant's
// color it spends. It is the second of Button's two color axes.
//
// # Why two axes and not one enum
//
// The obvious API is a single enum — Primary | Secondary | Danger | Ghost —
// and it is what the component gap analysis first sketched. It was dropped
// because it conflates two independent questions: *which* color (the meaning)
// and *how much* of it (the visual weight). A flat enum cannot express an
// outlined destructive button, the ordinary shape of a "Delete" confirmation,
// without a fifth value, and then a ghost destructive needs a sixth.
//
// Splitting them also lets Button reuse Variant verbatim, so a danger Button
// and an error Badge are the same red by construction rather than by two
// palettes agreeing. See Variant, which Badge already uses.
type Emphasis string

const (
	// EmphasisFilled is the zero value: a solid fill in the variant's color
	// with a contrast-picked label. This is the look core.Button has always
	// had, which is what makes the field's zero value a no-op.
	EmphasisFilled Emphasis = ""

	// EmphasisOutlined is a transparent fill, a 1px rule and a label both in
	// the variant's color — a secondary action that still names its meaning.
	EmphasisOutlined Emphasis = "outlined"

	// EmphasisGhost is EmphasisOutlined without the rule: label only. For a
	// tertiary action, a toolbar glyph, or a tab that must not look like a
	// pill.
	EmphasisGhost Emphasis = "ghost"
)

// Button is a themed action button with two orthogonal color axes and no
// per-call hex.
//
//	components.Button{Label: "Save", OnTap: save}                                // theme Button base
//	components.Button{Label: "Delete", Variant: components.VariantError, OnTap: rm}
//	components.Button{Label: "Cancel", Emphasis: components.EmphasisOutlined, OnTap: back}
//	components.Button{Label: "Skip", Emphasis: components.EmphasisGhost, OnTap: skip}
//
// # The zero value applies nothing
//
// Button{Label: l, OnTap: f} renders exactly core.Button(l, f) — not "core.Button
// re-derived from the palette". With both axes at their zero values the widget
// contributes no color props at all, so the theme's own Components.Button
// carries the look through untouched. That matters because a theme's Button
// base is allowed to differ from Colors.Primary; re-deriving would silently
// overwrite that choice, and the difference is invisible in the bundled themes
// where the two happen to agree.
//
// # Secondary is deliberately not a Variant
//
// Variant carries *meaning* — success, warning, error. Secondary is a brand
// slot a theme may set to anything (MaterialTheme makes it teal), so a
// "Variant: Secondary" would put a brand color where a reader expects a status.
// A button that wants the brand's second color says so through Style, which is
// the escape hatch for exactly the case the semantic roles do not cover:
//
//	components.Button{Label: "Recharge", Emphasis: components.EmphasisOutlined,
//	    Style: []core.StyleProp{core.TextColor(t.Colors.Secondary), core.BorderColor(t.Colors.Secondary)}}
//
// # Contrast, and the limit of what this widget can promise
//
// EmphasisFilled owns both the fill and the label, so it picks the label by
// contrast against the fill (Variant.Ink) and is tested to clear WCAG AA under
// both bundled themes.
//
// Outlined and Ghost own neither: the fill is transparent, so the label's real
// backdrop is whatever the button was placed on, which the widget cannot see.
// Their label is the variant's color verbatim — the theme author's choice, not
// a synthesized shade. Their legibility is therefore the palette's
// responsibility, and several bundled combinations are genuinely poor.
// Measured against each theme's own Background (both are #FFFFFF):
//
//	            Default   Material
//	default      4.02:1    7.63:1
//	success      2.22:1    5.13:1
//	warning      2.20:1    3.08:1
//	error        3.55:1    7.33:1
//
// Only Material's default, success and error clear WCAG AA (4.5:1). So prefer
// EmphasisFilled for a status action — it is tested to clear AA on both themes
// — and override TextColor when placing an outlined button where the numbers
// above do not hold.
//
// Darkening the role color until it passes was considered and rejected. It
// would repaint DefaultTheme's own brand blue (4.02:1, the value Apple ships
// and the one both themes pair with Button), i.e. the *default* case, and a
// widget silently altering a hex the theme author chose is worse than a
// documented number. The durable fix is a second palette value per role — an
// "on-light" tone, as Material carries alongside each container color — which
// is a palette decision, not a Button one.
type Button struct {
	Label string
	OnTap func()

	// Variant selects the semantic color role: Success, Warning, Error, or
	// the zero value for the theme's Primary.
	Variant Variant

	// Emphasis selects how that color is spent: filled (zero), outlined, or
	// ghost.
	Emphasis Emphasis

	// FullWidth stretches the button across its parent instead of hugging its
	// label. It sets both Width and a block Display: the bundled themes give
	// Button an inline display, and width has no effect on an inline box in
	// CSS. The natives read Display only for "hidden", so the block half is
	// inert there and the width alone does the work.
	FullWidth bool

	// Disabled renders the muted treatment and marks the control inert.
	//
	// Three independent things have to be true, and the widget does not get to
	// pick two:
	//
	//   - The platform must refuse to dispatch. That is core.Disabled, which
	//     every renderer now maps onto the native state (Compose's
	//     `enabled = false`, SwiftUI's `.disabled(true)`, the HTML disabled
	//     attribute). It is also what makes the control announce itself as
	//     disabled to a screen reader — VoiceOver says "dimmed", TalkBack
	//     reads the Disabled property — which is why the label no longer
	//     carries a hand-written ", disabled" suffix. Doing both would
	//     announce the state twice.
	//   - The handler must still be registered, replaced with a no-op rather
	//     than dropped: core.Button registers whatever it is given, and a nil
	//     func() in the registry panics if a native tap arrives in the window
	//     between the user pressing and the disabling patch landing.
	//   - It must look inert, which is the colorProps treatment below.
	Disabled bool

	// Style is applied after the variant treatment, so any single property
	// here overrides it.
	Style []core.StyleProp

	// AccessibilityLabel replaces the visible label for screen readers — use
	// it when Label is a glyph ("✕"). AccessibilityHint describes the effect
	// of tapping.
	AccessibilityLabel string
	AccessibilityHint  string
}

func (b Button) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	styles := make([]core.StyleProp, 0, len(b.Style)+8)
	styles = append(styles, b.colorProps(t)...)

	if b.FullWidth {
		styles = append(styles, core.Width("100%"), core.Display(core.DisplayBlock))
	}

	// Caller styles land after the treatment, so a one-off override wins over
	// the variant, and before the two props below that are not looks at all.
	styles = append(styles, b.Style...)

	// After the caller's styles: whether the control is inert is not a look,
	// so a one-off Style override must not be able to re-enable it.
	if b.Disabled {
		styles = append(styles, core.Disabled(true))
	}

	if b.AccessibilityLabel != "" {
		styles = append(styles, core.AccessibilityLabel(b.AccessibilityLabel))
	}
	if b.AccessibilityHint != "" {
		styles = append(styles, core.AccessibilityHint(b.AccessibilityHint))
	}

	onTap := b.OnTap
	if b.Disabled || onTap == nil {
		onTap = func() {}
	}

	return core.Button(b.Label, onTap, asProps(styles)...).Render(ctx)
}

// colorProps resolves the two axes into style props, or into nothing at all.
func (b Button) colorProps(t *core.Theme) []core.StyleProp {
	if b.Disabled {
		// One muted treatment for every variant: a disabled control has no
		// meaning left to signal, and a dimmed red still reads as "danger" to
		// a user who cannot tell it is inert. Surface + TextSecondary is the
		// palette's own muted pair, so this retints with the theme.
		return []core.StyleProp{
			core.BackgroundColor(t.Colors.Surface),
			core.TextColor(t.Colors.TextSecondary),
			core.BorderWidth(0),
		}
	}

	fill := b.Variant.Color(t)

	switch b.Emphasis {
	case EmphasisOutlined:
		return []core.StyleProp{
			core.BackgroundColor(ColorTransparent),
			core.TextColor(fill),
			core.BorderColor(fill),
			core.BorderWidth(1),
		}
	case EmphasisGhost:
		return []core.StyleProp{
			core.BackgroundColor(ColorTransparent),
			core.TextColor(fill),
		}
	}

	// Filled. The zero value of both axes contributes nothing so the theme's
	// Button base shows through verbatim; see the type doc.
	if b.Variant == VariantDefault {
		return nil
	}
	return []core.StyleProp{
		core.BackgroundColor(fill),
		core.TextColor(b.Variant.Ink(t, fill)),
	}
}

// asProps converts a collected []core.StyleProp into the []core.PropsAndChildren
// core.Button now takes.
//
// The conversion exists because Go will not spread a []StyleProp into a
// ...PropsAndChildren even though every element satisfies the wider type —
// the slice headers are different types, so the copy is unavoidable. It is
// the one call shape core.Button's widening broke, and the widgets here are
// the two places in the tree that hit it.
//
// The widgets' own Style fields stay []core.StyleProp on purpose rather than
// widening to match. A Button{} or Chip{} places its styles at a documented
// point in a treatment order it controls, and it has nowhere sensible to put
// a behavior prop; accepting one would mean silently deciding whether it
// lands before or after the variant colors. Callers that need a behavior prop
// on a button should reach for core.Button directly, which now takes them.
func asProps(styles []core.StyleProp) []core.PropsAndChildren {
	out := make([]core.PropsAndChildren, len(styles))
	for i, s := range styles {
		out[i] = s
	}
	return out
}
