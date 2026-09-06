package core

import "strings"

type Theme struct {
	Colors     ColorPalette
	Typography Typography
	Spacing    SpacingScale
	Components ComponentDefaults
}

// ColorPalette is a theme's semantic color roles. Widgets name the *role*
// they want, never a literal, so one theme swap restyles the whole tree.
//
// The seven original roles (Primary through Error) are set by every theme that
// exists, bundled or user-written, because they predate any of them. The three
// added later — Border, Success, Warning — cannot make that assumption: a
// theme written before they existed leaves them empty, and an empty color is
// not "the default", it is *no color*, which renders as an invisible rule or a
// transparent chip. Read those three through their resolver methods
// (BorderColor, SuccessColor, WarningColor) rather than off the field, and a
// pre-existing theme degrades to the documented fallback instead of to
// nothing. The originals need no such treatment and deliberately have no
// resolvers.
type ColorPalette struct {
	Primary       string
	Secondary     string
	Background    string
	Surface       string
	TextPrimary   string
	TextSecondary string
	Error         string

	// Border is the stroke/hairline role: rules between list rows, card
	// outlines, input borders. It is deliberately distinct from Surface.
	// Surface is a *fill* — the two are near neighbors on a light theme, so a
	// Surface-colored hairline on a Surface-colored panel is invisible.
	//
	// Read via BorderColor.
	Border string

	// Success and Warning complete the status triad with the existing Error,
	// for the "saved" / "expiring" / "failed" progression a status chip,
	// banner or badge variant needs.
	//
	// Success is *not* Secondary, even where a theme happens to give both the
	// same green (DefaultTheme does). Secondary is a brand slot — a theme is
	// free to make it teal or magenta, as MaterialTheme does — while Success
	// carries meaning, and a magenta "saved" badge is a bug.
	//
	// Read via SuccessColor and WarningColor.
	Success string
	Warning string

	// The on-light tones: the same four roles again, dark enough to be read
	// as *ink* on a light surface.
	//
	// A palette role is one hex, and one hex cannot do both jobs a role is
	// asked to do. Spent as a fill with a chosen ink over it, a mid-tone
	// works — components.Variant.Ink picks the more legible of the theme's
	// two inks and a filled Badge or Button clears WCAG AA on both bundled
	// themes. Spent as ink *itself* — an outlined button's label and rule, a
	// loud chip's outline, a banner's leading glyph — the backdrop is
	// whatever the widget was placed on, and a mid-tone loses. The numbers
	// against each bundled theme's own white Background were:
	//
	//	            Default   Material     (WCAG AA for body text is 4.5:1)
	//	primary      4.02:1    7.63:1
	//	success      2.22:1    5.13:1
	//	warning      2.20:1    3.08:1
	//	error        3.55:1    7.33:1
	//
	// Five of those eight fail. This is the fix components.Button and
	// components.Chip both name in their own docs and could not make: the
	// widgets have the number and not the authority. Darkening a role until
	// it passes would repaint a hex the theme author chose — DefaultTheme's
	// 4.02:1 blue is Apple's own system blue, and it is the *default* case —
	// so the second tone is the theme's to declare.
	//
	// # Reading them
	//
	// Through the resolver methods below, or through OnLight when a widget
	// holds a colour rather than a role. An unset tone falls back to the role
	// itself, which is exactly what every widget spent before these existed,
	// so a theme that predates them renders as it always did rather than
	// rendering nothing. That is a softer fallback than Border/Success/
	// Warning get, and deliberately: those degrade to a *visible* default
	// because an empty color is no color, while an absent on-light tone has a
	// perfectly good — merely paler — answer sitting beside it.
	//
	// # "Light" is the theme's Background, not a global assumption
	//
	// The name says which surface the tone is legible on, and for both
	// bundled themes that surface is #FFFFFF. A dark theme's role colours are
	// usually already legible on its dark background, so it leaves these
	// empty and the fallback returns the role — which is the right answer,
	// and the reason these are four extra fields rather than a second
	// palette every theme has to fill in twice.
	PrimaryOnLight string
	SuccessOnLight string
	WarningOnLight string
	ErrorOnLight   string
}

// Fallbacks for the three roles a pre-existing theme can be missing. They are
// DefaultTheme's own values, so a theme that omits a role looks like the
// default theme in that one place rather than disappearing.
//
// Exported because a widget outside this package resolving a role by hand
// (rather than through the methods below) should land on the same value.
const (
	FallbackBorder  = "#E5E5EA" // iOS systemGray5, the hairline both examples had independently picked
	FallbackSuccess = "#34C759" // iOS system green
	FallbackWarning = "#FF9500" // iOS system orange
)

// BorderColor resolves the Border role, falling back to FallbackBorder when
// the theme predates it.
//
// Note this is a *method on the palette* and is unrelated to the
// core.BorderColor style prop, which sets a node's stroke color:
//
//	core.BorderColor(ctx.Theme().Colors.BorderColor())
func (c ColorPalette) BorderColor() string {
	if c.Border != "" {
		return c.Border
	}
	return FallbackBorder
}

// SuccessColor resolves the Success role, falling back to FallbackSuccess.
func (c ColorPalette) SuccessColor() string {
	if c.Success != "" {
		return c.Success
	}
	return FallbackSuccess
}

// WarningColor resolves the Warning role, falling back to FallbackWarning.
func (c ColorPalette) WarningColor() string {
	if c.Warning != "" {
		return c.Warning
	}
	return FallbackWarning
}

// The four on-light resolvers. Each falls back to its own role rather than to
// a constant — see the field docs for why this fallback is softer than
// Border's, and note that each defers to the role's *resolver* where it has
// one, so a theme missing both halves of a role still lands somewhere visible.
func (c ColorPalette) PrimaryOnLightColor() string {
	if c.PrimaryOnLight != "" {
		return c.PrimaryOnLight
	}
	return c.Primary
}

func (c ColorPalette) SuccessOnLightColor() string {
	if c.SuccessOnLight != "" {
		return c.SuccessOnLight
	}
	return c.SuccessColor()
}

func (c ColorPalette) WarningOnLightColor() string {
	if c.WarningOnLight != "" {
		return c.WarningOnLight
	}
	return c.WarningColor()
}

func (c ColorPalette) ErrorOnLightColor() string {
	if c.ErrorOnLight != "" {
		return c.ErrorOnLight
	}
	return c.Error
}

// OnLight returns the ink-weight tone paired with color, when color is one of
// this palette's four toned roles, and color itself otherwise.
//
// The four resolvers above answer for a widget that knows which *role* it is
// spending. This answers for one that knows only a *colour*, which is the
// commoner case than it sounds: components.Chip's accent is read off the
// theme's Button base rather than off Colors.Primary, precisely so that a
// theme whose buttons are not primary-coloured still gets its own look, and a
// widget in that position has a hex and no name for it.
//
// So it is a reverse lookup, and it is honest about being one. A colour that
// is not one of the four roles comes back unchanged, which is the same
// fallback an unset tone gets and the same pixels every widget painted before
// these fields existed.
//
// # First match wins, and two roles may share a hex
//
// DefaultTheme paints Secondary and Success the same green. Only the four
// toned roles are consulted here, and no bundled theme repeats a colour among
// those four — but a theme could, and the arms are ordered Primary, Error,
// Success, Warning so that the answer is at least stable rather than
// map-order dependent. A palette that spends one hex on two of these roles is
// telling the widget the two are indistinguishable, which is a statement
// about the palette rather than a bug here.
//
// Comparison is case-insensitive because "#007AFF" and "#007aff" are the same
// colour to every renderer, and a theme is hand-written.
func (c ColorPalette) OnLight(color string) string {
	switch {
	case strings.EqualFold(color, c.Primary):
		return c.PrimaryOnLightColor()
	case strings.EqualFold(color, c.Error):
		return c.ErrorOnLightColor()
	case strings.EqualFold(color, c.SuccessColor()):
		return c.SuccessOnLightColor()
	case strings.EqualFold(color, c.WarningColor()):
		return c.WarningOnLightColor()
	}
	return color
}

type Typography struct {
	Title    Style
	Subtitle Style
	Body     Style
	Caption  Style
}

type SpacingScale struct {
	XS, SM, MD, LG, XL int
}

type ComponentDefaults struct {
	Button   Style
	Card     Style
	Input    Style
	Column   Style
	Row      Style
	Camera   Style
	CheckBox Style
	TextArea Style
	Text     Style
}

func WithTheme(theme *Theme, children ...View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		newCtx := ctx.WithTheme(theme)
		var rendered []*Node
		for _, child := range children {
			rendered = append(rendered, child.Render(newCtx))
		}
		return &Node{
			Type:     "Theme",
			Props:    map[string]any{},
			Children: rendered,
		}
	})
}

var DefaultTheme = &Theme{
	Colors: ColorPalette{
		Primary:       "#007AFF",   // iOS system blue
		Secondary:     "#34C759",   // iOS system green
		Background:    "#FFFFFF",   // white
		Surface:       "#F2F2F7",   // light gray
		TextPrimary:   "#000000",   // black
		TextSecondary: "#3C3C4399", // secondary label
		Error:         "#FF3B30",   // iOS system red
		Border:        "#E5E5EA",   // iOS systemGray5 — the separator hairline
		Success:       "#34C759",   // iOS system green (same hue as Secondary here; different role)
		Warning:       "#FF9500",   // iOS system orange

		// The ink-weight halves, measured against this theme's own white
		// Background. Apple publishes an accessible variant of each system
		// colour for light mode and three of these are it verbatim, which is
		// the same provenance the rest of this palette has.
		//
		// The green is not. Apple's accessible green is #248A3D, which
		// measures 4.40:1 — under the 4.5:1 AA floor for body text, and this
		// tone's whole job is to be read as body text. So systemGreen is
		// darkened past it instead, which is the one value here without a
		// published source and the one that would otherwise ship a number
		// that looks official and fails.
		PrimaryOnLight: "#0040DD", // Apple accessible blue   — 7.56:1 (from 4.02:1)
		SuccessOnLight: "#1E7A34", // systemGreen, darkened   — 5.40:1 (from 2.22:1)
		WarningOnLight: "#C93400", // Apple accessible orange — 5.28:1 (from 2.20:1)
		ErrorOnLight:   "#D70015", // Apple accessible red    — 5.38:1 (from 3.55:1)
	},
	Typography: Typography{
		Title: Style{
			FontSize:   28,
			FontWeight: Bold,
			TextColor:  "#000000",
			Display:    DisplayBlock,
		},
		Subtitle: Style{
			FontSize:   22,
			FontWeight: Normal,
			TextColor:  "#3C3C4399",
			Display:    DisplayBlock,
		},
		Body: Style{
			FontSize:   17,
			FontWeight: Normal,
			TextColor:  "#000000",
			Display:    DisplayBlock,
		},
		Caption: Style{
			FontSize:   13,
			FontWeight: Normal,
			TextColor:  "#3C3C4399",
			Display:    DisplayBlock,
		},
	},
	Spacing: SpacingScale{
		XS: 4,
		SM: 8,
		MD: 16,
		LG: 24,
		XL: 32,
	},
	Components: ComponentDefaults{
		Button: Style{
			FontSize:     17,
			FontWeight:   Normal,
			TextColor:    "#FFFFFF",
			Background:   "#007AFF",
			Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 16, Right: 16},
			BorderRadius: 8,
			Shadow:       1,
			Align:        AlignCenter,
			Display:      DisplayInline,
		},
		Card: Style{
			Background:   "#FFFFFF",
			Padding:      EdgeInsets{Top: 16, Bottom: 16, Left: 16, Right: 16},
			Margin:       EdgeInsets{Top: 8, Bottom: 8, Left: 8, Right: 8},
			BorderRadius: 12,
			Shadow:       2,
			Display:      DisplayBlock,
		},
		Input: Style{
			FontSize:     17,
			FontWeight:   Normal,
			TextColor:    "#000000",
			Background:   "#FFFFFF",
			Padding:      EdgeInsets{Top: 8, Bottom: 8, Left: 12, Right: 12},
			BorderRadius: 6,
			Shadow:       0,
			Display:      DisplayBlock,
		},
		CheckBox: Style{
			Background:   "#FFFFFF",
			BorderRadius: 6,
			Shadow:       0,
			Display:      DisplayInline,
		},
		TextArea: Style{
			FontSize:     17,
			FontWeight:   Normal,
			TextColor:    "#000000",
			Background:   "#FFFFFF",
			Padding:      EdgeInsets{Top: 12, Bottom: 12, Left: 12, Right: 12},
			BorderRadius: 6,
			Display:      DisplayBlock,
		},

		Column: Style{
			Padding: EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16},
		},
		Row: Style{
			Padding: EdgeInsets{Top: 8, Bottom: 8, Left: 16, Right: 16},
		},
		Camera: Style{
			Background: "#000000",
			Display:    DisplayBlock,
		},
		Text: Style{
			FontSize:     17,
			FontWeight:   Normal,
			TextColor:    "#000000",
			Background:   "#FFFFFF",
			Padding:      EdgeInsets{Top: 12, Bottom: 12, Left: 12, Right: 12},
			BorderRadius: 6,
			Display:      DisplayBlock,
		},
	},
}

var MaterialTheme = &Theme{
	Colors: ColorPalette{
		Primary:       "#6200EE",
		Secondary:     "#03DAC6",
		Background:    "#FFFFFF",
		Surface:       "#F5F5F5",
		TextPrimary:   "#212121",
		TextSecondary: "#757575",
		Error:         "#B00020",
		Border:        "#E0E0E0", // MD grey 300 — black at 12% over white, Material's divider
		Success:       "#2E7D32", // MD green 800, dark enough to carry white label text
		Warning:       "#EF6C00", // MD orange 800

		// Three of Material's four roles are already ink-weight against this
		// theme's white Background, so their on-light tone is the role
		// itself. Stating them rather than leaving them to the fallback is
		// the point: "this role needs no second tone" is a measurement, and a
		// blank field cannot tell it apart from "nobody has looked".
		//
		// Orange is the exception and needs a deeper swatch than its own
		// family carries — MD orange 900 (#E65100) is still only 3.79:1 — so
		// this is deep orange 900, which is Material's own answer for the
		// same problem one hue over.
		PrimaryOnLight: "#6200EE", // = Primary            — 7.63:1, already ink
		SuccessOnLight: "#2E7D32", // = Success            — 5.13:1, already ink
		WarningOnLight: "#BF360C", // MD deep orange 900   — 5.60:1 (from 3.08:1)
		ErrorOnLight:   "#B00020", // = Error              — 7.33:1, already ink
	},
	Typography: Typography{
		Title:    Style{FontSize: 22, FontWeight: Bold, TextColor: "#212121"},
		Subtitle: Style{FontSize: 18, FontWeight: Normal, TextColor: "#424242"},
		Body:     Style{FontSize: 14, FontWeight: Normal, TextColor: "#333333"},
		Caption:  Style{FontSize: 12, FontWeight: Light, TextColor: "#888888"},
	},
	Spacing: SpacingScale{
		XS: 4,
		SM: 8,
		MD: 16,
		LG: 24,
		XL: 32,
	},
	Components: ComponentDefaults{
		Button: Style{
			Background:   "#6200EE",
			TextColor:    "#FFFFFF",
			Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 20, Right: 20},
			BorderRadius: 4,
		},
		Card: Style{
			Background:   "#FFFFFF",
			BorderRadius: 8,
			Shadow:       1,
			Padding:      EdgeInsets{Top: 16, Bottom: 16, Left: 16, Right: 16},
		},
		Input: Style{
			Background: "#FAFAFA",
			Padding:    EdgeInsets{Top: 10, Bottom: 10, Left: 12, Right: 12},
		},
		Column: Style{
			Padding: EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16},
		},
		Row: Style{
			Padding: EdgeInsets{Top: 8, Bottom: 8, Left: 16, Right: 16},
		},
		Camera: Style{
			Background: "#000000",
			Display:    DisplayBlock,
		},

		CheckBox: Style{
			Display:      DisplayInline,
			Margin:       EdgeInsets{Right: 8},
			TextColor:    "#212121",
			BorderRadius: 2,
		},

		TextArea: Style{
			Background:   "#FAFAFA",
			TextColor:    "#212121",
			Padding:      EdgeInsets{Top: 8, Bottom: 8, Left: 12, Right: 12},
			BorderRadius: 4,
		},
	},
}
