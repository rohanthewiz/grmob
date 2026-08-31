package core

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
