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
	// outlines, the ring around a compass rose. It is deliberately distinct
	// from Surface. Surface is a *fill* — the two are near neighbors on a
	// light theme, so a Surface-colored hairline on a Surface-colored panel
	// is invisible.
	//
	// # It is a divider, not a control boundary
	//
	// This role used to name input borders too, and no longer does. A rule
	// *between* things is decoration — nothing about the page becomes
	// unusable if a reader cannot make it out — and every bundled theme spends
	// a very pale hex on it accordingly: #E5E5EA measures 1.26:1 against
	// white and #E0E0E0 1.32:1 (AmberTheme spends the second of those). The
	// edge that says *this rectangle is a field
	// you can type in* is the opposite case: it is the only thing identifying
	// a control, which WCAG 1.4.11 (Non-text Contrast) puts a 3:1 floor
	// under. One hex cannot be both, for the same reason a role's fill tone
	// cannot also be its ink — see the on-light tones below.
	//
	// So this role keeps the dividers and ControlBorder below carries the
	// boundary. That split used to have no second field in it — the frames
	// lived in Components.Input and Components.TextArea alone, and the note
	// here said no palette role was needed because nothing outside those two
	// component defaults spent one. components.Chip is what made that false:
	// a quiet chip's hairline is not a rule *between* things, it is the only
	// edge a filter control has, and it was drawing it out of this role.
	//
	// Read via BorderColor.
	Border string

	// ControlBorder is the boundary tone: the edge that says *this rectangle
	// is a control*. A text field's frame, a quiet chip's ring — anything a
	// reader has to make out before they can know there is something here to
	// operate.
	//
	// It is Border's other half and exists because one hex cannot do both
	// jobs, which is the same shape the on-light tones' argument has one
	// property over. A divider is decoration and a pale one is a legitimate
	// choice; a boundary that identifies a control carries WCAG 1.4.11's 3:1
	// floor. Every bundled theme spends 1.26:1 or 1.32:1 on Border
	// accordingly, so a control drawn in it is close to invisible *as a
	// control*:
	//
	//	                    Default            Material           Amber
	//	Border              #E5E5EA  1.26:1    #E0E0E0  1.32:1    #E0E0E0  1.32:1
	//	ControlBorder       #89898E  3.48:1    #757575  4.61:1    #8D6E63  4.62:1
	//
	// (against each theme's own white Background; see the themes for the
	// second backdrop each measures against.)
	//
	// # A control has more than one backdrop, and the census is the list
	//
	// The page is only the first of them. A boundary drawn on a Surface
	// panel, a Card or a field's own fill is measured against that fill, and
	// the tone is one hex for all of them — so "ControlBorder clears 3:1" is
	// not a property of the tone, it is a property of a *pair*.
	//
	// Every pair a bundled theme can produce is enumerated and measured by
	// TestEveryControlBoundaryPairIsAccountedFor in components/variant_test.go
	// (the arithmetic lives there, beside the on-light census, for the reason
	// that one gives). Every pair clears, and the census's
	// knownBoundaryShortfalls table — the place a defended shortfall would be
	// recorded as a fact rather than as prose — is empty.
	//
	// It was not always. DefaultTheme's tone was iOS systemGray #8E8E93,
	// which is 3.26:1 on that theme's page and 2.92:1 on its Surface, and the
	// quiet chip's ring is drawn on Surface. The shortfall was argued and
	// exempted for two sessions — the edge that identifies the pill is the
	// outer one, so the inner pair is a boundary between two parts of one
	// control — and the argument was sound. What retired it is that the
	// census made the alternative cheap: with every pair measured, "does this
	// candidate hex clear all four backdrops" is one test run, and #89898E
	// clears them at 3.12:1 and above while differing from systemGray by
	// 5/255 per channel. Defending a shortfall costs a paragraph that every
	// future boundary has to be read against; this cost a retint.
	//
	// # Why this arrived a session after the frames did
	//
	// Components.Input and Components.TextArea state their frame as a literal
	// and still do — a Style is a value, so a component default cannot call a
	// resolver — and while those two were the only spenders, a role would have
	// been a name with one call site. The second spender is what a role is
	// for. The two must not drift, so a bundled theme's Input and TextArea
	// frames are pinned to its ControlBorder by TestBundledFieldFramesAreThe
	// ControlBorderRole rather than by the type system.
	//
	// A widget that wants to look like a text field still reads the Input base
	// itself (components.DatePicker does), because it wants the radius and the
	// fill too. This role is for a widget that wants only the edge.
	//
	// Read via ControlBorderColor.
	ControlBorder string

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
	// two inks and a filled Badge or Button clears WCAG AA on every bundled
	// theme. Spent as ink *itself* — an outlined button's label and rule, a
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
	// it passes would repaint a hex the theme author chose, so the second
	// tone is the theme's to declare.
	//
	// # One of those five was later fixed at the role, and why that is not a
	// contradiction
	//
	// DefaultTheme's primary is no longer 4.02:1. The role itself moved to
	// Apple's accessible blue #0040DD (7.56:1), because that role is not read
	// as ink only — it is also the *fill* under every filled Button and under
	// Calendar's selected day, and the white the theme declares over that fill
	// was the thing failing. A second tone cannot reach a declared pairing;
	// only the role can. So the table above is the state these fields were
	// introduced to answer for, and four of the eight still fail today.
	//
	// Which gives the rule from the other side. Darken the *role* when the
	// role is spent as a fill and the ink declared over it is the problem;
	// add a *tone* when the role is a perfectly good fill and only fails as
	// ink. Success and Warning are the second case — DefaultTheme's green and
	// orange carry black text at ~9.5:1 — and Primary turned out to be the
	// first.
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

	// FallbackControlBorder is DefaultTheme's boundary tone, which is also
	// what its Input and TextArea frames are painted in. A theme predating
	// this role degrades to a *visible* edge rather than to Border's hairline:
	// falling back to the divider is the levelling-down the role was split to
	// prevent, and it would be indistinguishable from the bug.
	FallbackControlBorder = "#89898E" // systemGray, darkened — 3.48:1 on white
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

// ControlBorderColor resolves the ControlBorder role, falling back to
// FallbackControlBorder when the theme predates it.
//
// Note it does *not* fall back to BorderColor(). The two roles are near
// neighbours in the struct and opposites in intent — see the field docs — and
// a theme that has one and not the other is a theme that has only the divider,
// which is precisely the value this must not return.
func (c ColorPalette) ControlBorderColor() string {
	if c.ControlBorder != "" {
		return c.ControlBorder
	}
	return FallbackControlBorder
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
//
// # Most of these arms are identities under most bundled themes
//
// A role whose colour is already ink-weight sets its tone equal to itself,
// deliberately (see each palette for why "this role needs no second tone" is
// written out rather than left blank). Under DefaultTheme only Primary is in
// that position; under MaterialTheme, Primary, Success and Error are; under
// AmberTheme, Success and Error. An arm that is an identity for *every*
// bundled theme could be deleted with no bundled pixel moving.
//
// Primary's used to be exactly that, and AmberTheme is what put it back: amber
// 700 is a fill that cannot be ink (2.04:1 on white), so its role and its tone
// are genuinely different colours and deleting this arm repaints every outlined
// button and loud chip in that theme.
//
// Which arms have a bundled witness and which rest on a test fixture is
// recorded and checked per role by TestEveryPaletteRuleStillHasAWitness in
// components/palette_witness_test.go, so a retint that leaves an arm with no
// evidence anywhere is reported rather than merely true.
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
		// Apple's *accessible* system blue, not plain systemBlue (#007AFF).
		//
		// This role is spent as a fill far more often than as ink — every
		// filled Button, Badge and Avatar, Calendar's selected day, the
		// compass needle, ProgressBar's fill — and Components.Button below
		// declares white as the ink over it. White on #007AFF is 4.02:1,
		// under WCAG AA's 4.5:1 for body text, so the framework's own default
		// theme was shipping an unreadable label on its commonest control.
		//
		// Apple publishes an accessible variant of each system colour for
		// light mode and this is it verbatim, which is the provenance the
		// rest of this palette has. White over it is 7.56:1.
		//
		// The alternative was to darken Components.Button alone and leave the
		// role at systemBlue, and it is worth saying why that is wrong rather
		// than merely narrower: the button base is what declares the ink for
		// *this* fill (see components.declaredInk). Move one without the
		// other and Primary becomes a fill the theme has paired nothing with,
		// so Calendar's selected day falls back to measurement and picks
		// black on system blue — exactly the disagreement inkOn was written
		// to end.
		Primary:       "#0040DD",   // Apple accessible blue — 7.56:1 under white
		Secondary:     "#34C759",   // iOS system green
		Background:    "#FFFFFF",   // white
		Surface:       "#F2F2F7",   // light gray
		TextPrimary:   "#000000",   // black
		TextSecondary: "#3C3C4399", // secondary label
		Error:         "#FF3B30",   // iOS system red
		Border:        "#E5E5EA",   // iOS systemGray5 — the separator hairline
		Success:       "#34C759",   // iOS system green (same hue as Secondary here; different role)
		Warning:       "#FF9500",   // iOS system orange

		// The boundary tone, and the same hex Components.Input and
		// Components.TextArea below paint their frames in — which is a
		// requirement rather than a coincidence, pinned in theme_test.go.
		//
		// Measured against both backdrops a control has here: the page and a
		// field's fill are the same white, and a quiet Chip's fill is Surface.
		// Both clear WCAG 1.4.11's 3:1 floor, which is the whole census in
		// components/variant_test.go for this theme.
		//
		// This is *not* Apple's systemGray, and the two digits are the only
		// value in this palette that leaves its published source. systemGray
		// is #8E8E93 and measures 2.92:1 on this theme's Surface — under the
		// floor, by 0.08. The pair was defended rather than fixed while the
		// only way to check a replacement was an audit; the census turned that
		// into one test run, and a tone five steps darker clears every backdrop
		// at a distance an eye cannot tell from systemGray. See
		// ColorPalette.ControlBorder for the argument that was retired.
		ControlBorder: "#89898E", // systemGray, darkened — 3.48:1 on #FFFFFF, 3.12:1 on #F2F2F7

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
		//
		// Blue is stated and equal to its role, on MaterialTheme's pattern:
		// Primary moved to the accessible variant for the fill's sake, so the
		// role is now ink-weight on its own and needs no second tone. Written
		// out rather than left to PrimaryOnLightColor's fallback because
		// "this role needs no second tone" is a measurement, and a blank
		// field cannot be told apart from "nobody has looked".
		PrimaryOnLight: "#0040DD", // = Primary               — 7.56:1, already ink
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
			FontSize:   17,
			FontWeight: Normal,
			// The declared pair, and the only place this theme states a fill
			// and an ink together — which is what components.declaredInk
			// reads back for every widget that paints Primary. Two things
			// have to hold and neither is expressible in the type system:
			// the fill stays Colors.Primary (TestBundledButtonFillsAreThe
			// PrimaryRole, next door) and the ink stays legible over it
			// (components' TestVariantInkIsLegibleOnEveryThemeAndVariant,
			// which owns the WCAG arithmetic).
			TextColor:    "#FFFFFF",
			Background:   "#0040DD", // = Colors.Primary — white over it is 7.56:1
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
			FontSize:   17,
			FontWeight: Normal,
			TextColor:  "#000000",
			Background: "#FFFFFF",
			Padding:    EdgeInsets{Top: 8, Bottom: 8, Left: 12, Right: 12},
			// The field frame. This theme paints a white field on a white
			// page, so the border is the whole of what says a control is
			// here — and it is why the border is not Colors.Border: that
			// role's #E5E5EA is a 1.26:1 divider, and a boundary that
			// identifies a control has a 3:1 floor under it (WCAG 1.4.11).
			// systemGray is Apple's own tone at that weight, and this is that
			// tone darkened five steps for the Surface backdrop a chip's ring
			// needs (see Colors.ControlBorder). It measures 3.48:1 against
			// this theme's Background, which is also roughly what the
			// browser's own input border was drawing before borderResetTypes
			// took it away. See ColorPalette.Border.
			BorderColor:  "#89898E", // = Colors.ControlBorder — 3.48:1 on #FFFFFF
			BorderWidth:  1,
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
			FontSize:   17,
			FontWeight: Normal,
			TextColor:  "#000000",
			Background: "#FFFFFF",
			Padding:    EdgeInsets{Top: 12, Bottom: 12, Left: 12, Right: 12},
			// The same frame as Input, and the same reasoning: a <textarea>
			// is the other tag whose user-agent border the web used to draw
			// for free. A multi-line field that differed from a single-line
			// one by its edge alone would look like two controls.
			BorderColor:  "#89898E", // = Colors.ControlBorder — 3.48:1 on #FFFFFF
			BorderWidth:  1,
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

		// The boundary tone, and the hex this theme's Input and TextArea
		// frames already carried. Material's own text-field outline colour,
		// and it clears 3:1 against every backdrop a control sits on here:
		// the white page, the #FAFAFA field fill, and the #F5F5F5 Surface a
		// quiet Chip is filled with.
		ControlBorder: "#757575", // MD grey 600 — 4.61:1 on #FFFFFF, 4.23:1 on #F5F5F5

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
			// grey 600, not the grey 300 this theme's Border role spends: a
			// divider may be 1.32:1 and a control boundary may not (WCAG
			// 1.4.11 asks 3:1). It is the same hex as TextSecondary, which is
			// Material's own medium-emphasis weight, and it measures 4.61:1
			// against both this theme's Background and the field's own fill.
			BorderColor: "#757575", // MD grey 600 — 4.61:1 on #FFFFFF
			BorderWidth: 1,
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
			BorderColor:  "#757575", // MD grey 600 — 4.61:1 on #FFFFFF
			BorderWidth:  1,
			BorderRadius: 4,
		},
	},
}

// AmberTheme is the third bundled palette, and the one whose brand colour is
// not a colour you can put white on.
//
// # Why a third theme exists at all
//
// Two rules in this framework had no bundled evidence. inkOn reads the theme's
// declared fill/ink pair *before* measuring contrast, and ColorPalette.OnLight
// moves a role to its ink-weight tone — and under both palettes above, an
// implementation that deleted either would paint identical pixels. Their only
// witness was a test fixture (components' midTonePrimaryTheme), which is a
// thing a palette edit can quietly leave holding the whole thread.
//
// This palette witnesses both, and it does so the way a real brand does rather
// than by being contrived:
//
//	the declaration outranks the measurement
//	    Components.Button below states brown 900 over the amber fill. Measured
//	    against this theme's own two inks, the winner is TextPrimary (8.39:1
//	    against brown 900's 6.77:1) — so an inkOn that only measured would paint
//	    a different, and slightly *higher*-contrast, label. Deleting declaredInk
//	    moves pixels here.
//
//	OnLight moves the Primary role
//	    Amber 700 is 2.04:1 on white. It is an excellent fill and cannot be ink,
//	    which is the whole argument for the on-light tones and the case neither
//	    palette above still makes for Primary.
//
// # The escape that made this shippable, which is worth recording
//
// The obvious third theme — a mid-tone brand with white declared over it — is
// not shippable, and the reason is arithmetic rather than taste. contrastInk
// picks the *higher*-contrast of the theme's two ink roles, and against any
// fill the white ratio and the black ratio multiply to at most 21. So a
// declaration that loses the measurement can be at most sqrt(21) = 4.58:1,
// while components' AA check demands 4.5:1 of every variant's ink. The band a
// pole-flipping bundled theme would have to sit in is [4.50, 4.58] — 1.8%
// wide, at the very bottom of the legibility scale.
//
// The squeeze binds only when the declared ink is one of the two poles being
// measured. A brand ink that is *neither* — brown 900 over amber, where the
// page's ink is MD3's near-black — is a different colour from the measurement's
// answer with three points of headroom on both. That is also the ordinary
// real-world decision ("our label is warm, not the page's black"), so the theme
// carries the rule by being a normal theme rather than by being tuned to a
// gap. components' TestThePoleFlipBandIsTooNarrowToShip states the arithmetic
// above as a test, and the pole flip itself stays with the fixture — now
// provably rather than accidentally.
//
// # Provenance
//
// Material Design published values throughout, as MaterialTheme's are, with one
// exception stated at the field: the amber family carries no swatch dark enough
// to be read as ink on white (amber 900 is 2.79:1), so PrimaryOnLight is amber
// 700 scaled to 56% brightness — the same hue at 37.8 degrees, and the same
// move DefaultTheme's SuccessOnLight makes one hue over for the same reason.
var AmberTheme = &Theme{
	Colors: ColorPalette{
		// MD amber 700. A fill and not an ink: white over it is 2.04:1 and the
		// page's own near-black is 8.39:1, which is why this role has a
		// separate on-light tone below and why the button declares a dark
		// label. Both bundled palettes above are the other case — a role dark
		// enough to be either — so this is the only place in the repository a
		// reader can see the two halves of a role come apart in a shipped
		// theme.
		Primary:       "#FFA000",
		Secondary:     "#00897B", // MD teal 600 — a brand slot, deliberately not the status green
		Background:    "#FFFFFF", // white; the warmth is in Surface, not the page
		Surface:       "#FFF8E1", // MD amber 50 — the panel tint, one step off the page
		TextPrimary:   "#1C1B1F", // MD3 on-surface — 17.13:1 on the page
		TextSecondary: "#616161", // MD grey 700 — 6.19:1 on the page, 5.83:1 on Surface
		Error:         "#B00020", // MD error, as MaterialTheme
		Border:        "#E0E0E0", // MD grey 300 — 1.32:1, a divider and nothing else
		Success:       "#2E7D32", // MD green 800
		Warning:       "#EF6C00", // MD orange 800

		// MD brown 300, chosen at Material's own field-frame *weight* rather
		// than for its family: grey 600 measures 4.61:1 on white and this
		// measures 4.62:1, so the frame reads as the same thickness of edge in
		// a warm palette. It clears WCAG 1.4.11's 3:1 on every backdrop a
		// control sits on here — the white page and Card, and the amber-50
		// Surface a quiet Chip and both field fills use, at 4.35:1.
		ControlBorder: "#8D6E63",

		// Amber 700 darkened until it can be read as ink: (255,160,0) scaled
		// to 56% is (143,90,0), which is the same hue (37.8 degrees) at 5.78:1.
		// The amber family stops at 900 (#FF6F00, 2.79:1) and has nothing
		// darker, so this is the one value in this palette without a published
		// source — the same position, and the same remedy, as DefaultTheme's
		// SuccessOnLight.
		//
		// This is the pair no bundled theme could show before: a role that is
		// a good fill and cannot be ink.
		PrimaryOnLight: "#8F5A00", // amber 700 at 56% — 5.78:1 (from 2.04:1)
		SuccessOnLight: "#2E7D32", // = Success            — 5.13:1, already ink
		WarningOnLight: "#BF360C", // MD deep orange 900   — 5.60:1 (from 3.08:1)
		ErrorOnLight:   "#B00020", // = Error              — 7.33:1, already ink
	},
	Typography: Typography{
		Title:    Style{FontSize: 24, FontWeight: Bold, TextColor: "#1C1B1F", Display: DisplayBlock},
		Subtitle: Style{FontSize: 19, FontWeight: Normal, TextColor: "#3A3A3E", Display: DisplayBlock},
		Body:     Style{FontSize: 16, FontWeight: Normal, TextColor: "#1C1B1F", Display: DisplayBlock},
		Caption:  Style{FontSize: 13, FontWeight: Normal, TextColor: "#616161", Display: DisplayBlock},
	},
	Spacing: SpacingScale{
		XS: 4,
		SM: 8,
		MD: 16,
		LG: 24,
		XL: 32,
	},
	Components: ComponentDefaults{
		// The declaration this theme exists to carry.
		//
		// Brown 900 over amber 700 at 6.77:1 — comfortably legible, and *not*
		// what measurement would choose. contrastInk compares this theme's two
		// ink roles against the fill and prefers TextPrimary at 8.39:1, so an
		// implementation that dropped declaredInk from inkOn would repaint
		// every filled Button, Badge, Avatar and Calendar selection in this
		// theme with the page's near-black.
		//
		// A brand ink rather than the page's ink is an ordinary decision and
		// the reason it is the *shippable* one is in this theme's own doc: an
		// ink that is one of the two measured poles would have had to sit in a
		// 1.8%-wide band at the AA floor.
		Button: Style{
			FontSize:     16,
			FontWeight:   Bold,
			TextColor:    "#3E2723", // MD brown 900 — 6.77:1 over the amber fill
			Background:   "#FFA000", // = Colors.Primary
			Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 20, Right: 20},
			BorderRadius: 20, // a pill, which is what a single strong brand colour wants
			Shadow:       1,
			Align:        AlignCenter,
			Display:      DisplayInline,
		},
		Card: Style{
			Background:   "#FFFFFF",
			Padding:      EdgeInsets{Top: 16, Bottom: 16, Left: 16, Right: 16},
			Margin:       EdgeInsets{Top: 8, Bottom: 8, Left: 8, Right: 8},
			BorderRadius: 12,
			Shadow:       1,
			Display:      DisplayBlock,
		},
		Input: Style{
			FontSize:   16,
			FontWeight: Normal,
			TextColor:  "#1C1B1F",
			// The field fill is the Surface tint rather than the page, so the
			// frame has two different backdrops here as it does under
			// MaterialTheme — which is the shape the boundary census wants at
			// least one bundled theme to have.
			Background:   "#FFF8E1",
			Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 12, Right: 12},
			BorderColor:  "#8D6E63", // = Colors.ControlBorder — 4.62:1 on the page, 4.35:1 on this fill
			BorderWidth:  1,
			BorderRadius: 8,
			Shadow:       0,
			Display:      DisplayBlock,
		},
		CheckBox: Style{
			Background:   "#FFFFFF",
			TextColor:    "#1C1B1F",
			BorderRadius: 4,
			Margin:       EdgeInsets{Right: 8},
			Display:      DisplayInline,
		},
		TextArea: Style{
			FontSize:     16,
			FontWeight:   Normal,
			TextColor:    "#1C1B1F",
			Background:   "#FFF8E1",
			Padding:      EdgeInsets{Top: 12, Bottom: 12, Left: 12, Right: 12},
			BorderColor:  "#8D6E63", // = Colors.ControlBorder, as Input's — one edge, two tags
			BorderWidth:  1,
			BorderRadius: 8,
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
			FontSize:     16,
			FontWeight:   Normal,
			TextColor:    "#1C1B1F",
			Background:   "#FFFFFF",
			Padding:      EdgeInsets{Top: 12, Bottom: 12, Left: 12, Right: 12},
			BorderRadius: 8,
			Display:      DisplayBlock,
		},
	},
}

// BundledThemes returns every theme this package ships, keyed by its Go
// identifier.
//
// # Why this exists rather than a list at each call site
//
// It used to be a hand-written map in every test that asks a question of "the
// bundled themes" — the palette censuses in components, the role and frame pins
// in this package's own tests, several widget tests. There were a dozen of
// them, all spelling the same two entries, and the failure mode is the one
// TestBundledThemesSetEveryColorRole was written reflectively to avoid one
// level down: a third theme is added, a census keeps its two-entry literal, and
// the new palette is simply never asked the question. Nothing fails. The rule
// goes on being true of the themes somebody remembered.
//
// So the list is one list, and TestBundledThemesListIsExhaustive derives it
// from this file's own source rather than trusting it — a package-level
// *Theme var that is not in this map fails there.
//
// A fresh map each call, for the reason ColorPalette's resolvers exist: a
// package-level map is reachable and writable by any importer, and a test that
// deleted an entry would silently narrow every census at once.
func BundledThemes() map[string]*Theme {
	return map[string]*Theme{
		"DefaultTheme":  DefaultTheme,
		"MaterialTheme": MaterialTheme,
		"AmberTheme":    AmberTheme,
	}
}
