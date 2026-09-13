# Package core — Theming

```go
import "github.com/rohanthewiz/grmob/core"
```

Themes, palettes, typography and spacing scales, and per-component defaults.

One of 11 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/theme.go`.

## Index

- [Constants](#constants) — `FallbackBorder`, `FallbackControlBorder`, `FallbackSuccess`, `FallbackWarning`
- [Variables](#variables) — `AmberTheme`, `DefaultTheme`, `MaterialTheme`
- [`func BundledThemes`](#func-bundledthemes)
- [`func WithTheme`](#func-withtheme)
- [`type ColorPalette`](#type-colorpalette)
    - [`func (ColorPalette) BorderColor`](#func-colorpalette-bordercolor)
    - [`func (ColorPalette) ControlBorderColor`](#func-colorpalette-controlbordercolor)
    - [`func (ColorPalette) ErrorOnLightColor`](#func-colorpalette-erroronlightcolor)
    - [`func (ColorPalette) OnLight`](#func-colorpalette-onlight)
    - [`func (ColorPalette) PrimaryOnLightColor`](#func-colorpalette-primaryonlightcolor)
    - [`func (ColorPalette) SuccessColor`](#func-colorpalette-successcolor)
    - [`func (ColorPalette) SuccessOnLightColor`](#func-colorpalette-successonlightcolor)
    - [`func (ColorPalette) WarningColor`](#func-colorpalette-warningcolor)
    - [`func (ColorPalette) WarningOnLightColor`](#func-colorpalette-warningonlightcolor)
- [`type ComponentDefaults`](#type-componentdefaults)
- [`type SpacingScale`](#type-spacingscale)
- [`type Theme`](#type-theme)
- [`type Typography`](#type-typography)

## Constants

Fallbacks for the three roles a pre-existing theme can be missing. They are DefaultTheme's own values, so a theme that omits a role looks like the default theme in that one place rather than disappearing.

Exported because a widget outside this package resolving a role by hand (rather than through the methods below) should land on the same value.

```go
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
```

<small>[core/theme.go:240](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L240)</small>

## Variables

AmberTheme is the third bundled palette, and the one whose brand colour is not a colour you can put white on.

#### Why a third theme exists at all

Two rules in this framework had no bundled evidence. inkOn reads the theme's declared fill/ink pair \*before\* measuring contrast, and ColorPalette.OnLight moves a role to its ink-weight tone — and under both palettes above, an implementation that deleted either would paint identical pixels. Their only witness was a test fixture (components' midTonePrimaryTheme), which is a thing a palette edit can quietly leave holding the whole thread.

This palette witnesses both, and it does so the way a real brand does rather than by being contrived:

	the declaration outranks the measurement
	    Components.Button below states brown 900 over the amber fill. Measured
	    against this theme's own two inks, the winner is TextPrimary (8.39:1
	    against brown 900's 6.77:1) — so an inkOn that only measured would paint
	    a different, and slightly *higher*-contrast, label. Deleting declaredInk
	    moves pixels here.

	OnLight moves the Primary role
	    Amber 700 is 2.04:1 on white. It is an excellent fill and cannot be ink,
	    which is the whole argument for the on-light tones and the case neither
	    palette above still makes for Primary.

#### The escape that made this shippable, which is worth recording

The obvious third theme — a mid-tone brand with white declared over it — is not shippable, and the reason is arithmetic rather than taste. contrastInk picks the \*higher\*-contrast of the theme's two ink roles, and against any fill the white ratio and the black ratio multiply to at most 21. So a declaration that loses the measurement can be at most sqrt(21) = 4.58:1, while components' AA check demands 4.5:1 of every variant's ink. The band a pole-flipping bundled theme would have to sit in is \[4.50, 4.58] — 1.8% wide, at the very bottom of the legibility scale.

The squeeze binds only when the declared ink is one of the two poles being measured. A brand ink that is \*neither\* — brown 900 over amber, where the page's ink is MD3's near-black — is a different colour from the measurement's answer with three points of headroom on both. That is also the ordinary real-world decision ("our label is warm, not the page's black"), so the theme carries the rule by being a normal theme rather than by being tuned to a gap. components' TestThePoleFlipBandIsTooNarrowToShip states the arithmetic above as a test, and the pole flip itself stays with the fixture — now provably rather than accidentally.

#### Provenance

Material Design published values throughout, as MaterialTheme's are, with one exception stated at the field: the amber family carries no swatch dark enough to be read as ink on white (amber 900 is 2.79:1), so PrimaryOnLight is amber 700 scaled to 56% brightness — the same hue at 37.8 degrees, and the same move DefaultTheme's SuccessOnLight makes one hue over for the same reason.

```go
var AmberTheme = &Theme{
	Colors: ColorPalette{

		Primary:       "#FFA000",
		Secondary:     "#00897B",
		Background:    "#FFFFFF",
		Surface:       "#FFF8E1",
		TextPrimary:   "#1C1B1F",
		TextSecondary: "#616161",
		Error:         "#B00020",
		Border:        "#E0E0E0",
		Success:       "#2E7D32",
		Warning:       "#EF6C00",

		ControlBorder: "#8D6E63",

		PrimaryOnLight: "#8F5A00",
		SuccessOnLight: "#2E7D32",
		WarningOnLight: "#BF360C",
		ErrorOnLight:   "#B00020",
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

		Button: Style{
			FontSize:     16,
			FontWeight:   Bold,
			TextColor:    "#3E2723",
			Background:   "#FFA000",
			Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 20, Right: 20},
			BorderRadius: 20,
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

			Background:   "#FFF8E1",
			Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 12, Right: 12},
			BorderColor:  "#8D6E63",
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
			BorderColor:  "#8D6E63",
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
	},
}
```

<small>[core/theme.go:847](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L847)</small>

```go
var DefaultTheme = &Theme{
	Colors: ColorPalette{

		Primary:       "#0040DD",
		Secondary:     "#34C759",
		Background:    "#FFFFFF",
		Surface:       "#F2F2F7",
		TextPrimary:   "#000000",
		TextSecondary: "#3C3C4399",
		Error:         "#FF3B30",
		Border:        "#E5E5EA",
		Success:       "#34C759",
		Warning:       "#FF9500",

		ControlBorder: "#89898E",

		PrimaryOnLight: "#0040DD",
		SuccessOnLight: "#1E7A34",
		WarningOnLight: "#C93400",
		ErrorOnLight:   "#D70015",
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

			TextColor:    "#FFFFFF",
			Background:   "#0040DD",
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

			BorderColor:  "#89898E",
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

			BorderColor:  "#89898E",
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
	},
}
```

<small>[core/theme.go:496](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L496)</small>

```go
var MaterialTheme = &Theme{
	Colors: ColorPalette{
		Primary:       "#6200EE",
		Secondary:     "#03DAC6",
		Background:    "#FFFFFF",
		Surface:       "#F5F5F5",
		TextPrimary:   "#212121",
		TextSecondary: "#757575",
		Error:         "#B00020",
		Border:        "#E0E0E0",
		Success:       "#2E7D32",
		Warning:       "#EF6C00",

		ControlBorder: "#757575",

		PrimaryOnLight: "#6200EE",
		SuccessOnLight: "#2E7D32",
		WarningOnLight: "#BF360C",
		ErrorOnLight:   "#B00020",
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

			BorderColor: "#757575",
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
			BorderColor:  "#757575",
			BorderWidth:  1,
			BorderRadius: 4,
		},
	},
}
```

<small>[core/theme.go:691](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L691)</small>

## Functions

### func BundledThemes

```go
func BundledThemes() map[string]*Theme
```

BundledThemes returns every theme this package ships, keyed by its Go identifier.

#### Why this exists rather than a list at each call site

It used to be a hand-written map in every test that asks a question of "the bundled themes" — the palette censuses in components, the role and frame pins in this package's own tests, several widget tests. There were a dozen of them, all spelling the same two entries, and the failure mode is the one TestBundledThemesSetEveryColorRole was written reflectively to avoid one level down: a third theme is added, a census keeps its two-entry literal, and the new palette is simply never asked the question. Nothing fails. The rule goes on being true of the themes somebody remembered.

So the list is one list, and TestBundledThemesListIsExhaustive derives it from this file's own source rather than trusting it — a package-level \*Theme var that is not in this map fails there.

A fresh map each call, for the reason ColorPalette's resolvers exist: a package-level map is reachable and writable by any importer, and a test that deleted an entry would silently narrow every census at once.

<small>[core/theme.go:1003](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L1003)</small>

### func WithTheme

```go
func WithTheme(theme *Theme, children ...View) View
```

<small>[core/theme.go:481](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L481)</small>

## Types

### type ColorPalette

```go
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
	// component defaults spent one. comps.Chip is what made that false:
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
	// TestEveryControlBoundaryPairIsAccountedFor in comps/variant_test.go
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
	// itself (comps.DatePicker does), because it wants the radius and the
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
	// works — comps.Variant.Ink picks the more legible of the theme's
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
	// Five of those eight fail. This is the fix comps.Button and
	// comps.Chip both name in their own docs and could not make: the
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
	//
	// # Overriding a role means releasing its tone
	//
	// These are *measurements*, taken against the role colour beside them. An
	// app that brands a theme by copying DefaultTheme and assigning
	// Colors.Primary therefore inherits a tone measured against a colour that
	// is no longer there:
	//
	//	theme := *core.DefaultTheme
	//	theme.Colors.Primary = siteColor   // and PrimaryOnLight is still #0040DD
	//
	// The result is a half-branded app, and a quiet one — filled controls take
	// the new colour (they read Primary, or the Button base) while every
	// unfilled one keeps the default's blue, because Outlined, Ghost, Chip and
	// the calendar's month arrows all spend the *tone*. The first downstream
	// app to brand a theme shipped exactly that.
	//
	// So an override sets the pair or clears it:
	//
	//	theme.Colors.Primary = siteColor
	//	theme.Colors.PrimaryOnLight = ""   // no measurement, use the role
	//
	// Clearing is the honest default. The fallback then returns siteColor,
	// which is the same treatment every widget gave before these fields
	// existed; writing siteColor into the tone renders identically but claims
	// a contrast check nobody ran. Either way, what is not available is
	// leaving the old number in place.
	PrimaryOnLight string
	SuccessOnLight string
	WarningOnLight string
	ErrorOnLight   string
}
```

ColorPalette is a theme's semantic color roles. Widgets name the \*role\* they want, never a literal, so one theme swap restyles the whole tree.

The seven original roles (Primary through Error) are set by every theme that exists, bundled or user-written, because they predate any of them. The three added later — Border, Success, Warning — cannot make that assumption: a theme written before they existed leaves them empty, and an empty color is not "the default", it is \*no color\*, which renders as an invisible rule or a transparent chip. Read those three through their resolver methods (BorderColor, SuccessColor, WarningColor) rather than off the field, and a pre-existing theme degrades to the documented fallback instead of to nothing. The originals need no such treatment and deliberately have no resolvers.

<small>[core/theme.go:25](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L25)</small>

#### func (ColorPalette) BorderColor

```go
func (c ColorPalette) BorderColor() string
```

BorderColor resolves the Border role, falling back to FallbackBorder when the theme predates it.

Note this is a \*method on the palette\* and is unrelated to the core.BorderColor style prop, which sets a node's stroke color:

	core.BorderColor(ctx.Theme().Colors.BorderColor())

<small>[core/theme.go:260](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L260)</small>

#### func (ColorPalette) ControlBorderColor

```go
func (c ColorPalette) ControlBorderColor() string
```

ControlBorderColor resolves the ControlBorder role, falling back to FallbackControlBorder when the theme predates it.

Note it does \*not\* fall back to BorderColor(). The two roles are near neighbours in the struct and opposites in intent — see the field docs — and a theme that has one and not the other is a theme that has only the divider, which is precisely the value this must not return.

<small>[core/theme.go:274](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L274)</small>

#### func (ColorPalette) ErrorOnLightColor

```go
func (c ColorPalette) ErrorOnLightColor() string
```

<small>[core/theme.go:322](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L322)</small>

#### func (ColorPalette) OnLight

```go
func (c ColorPalette) OnLight(color string) string
```

OnLight returns the ink-weight tone paired with color, when color is one of this palette's four toned roles, and color itself otherwise.

The four resolvers above answer for a widget that knows which \*role\* it is spending. This answers for one that knows only a \*colour\*, which is the commoner case than it sounds: comps.Chip's accent is read off the theme's Button base rather than off Colors.Primary, precisely so that a theme whose buttons are not primary-coloured still gets its own look, and a widget in that position has a hex and no name for it.

So it is a reverse lookup, and it is honest about being one. A colour that is not one of the four roles comes back unchanged, which is the same fallback an unset tone gets and the same pixels every widget painted before these fields existed.

##### First match wins, and two roles may share a hex

DefaultTheme paints Secondary and Success the same green. Only the four toned roles are consulted here, and no bundled theme repeats a colour among those four — but a theme could, and the arms are ordered Primary, Error, Success, Warning so that the answer is at least stable rather than map-order dependent. A palette that spends one hex on two of these roles is telling the widget the two are indistinguishable, which is a statement about the palette rather than a bug here.

Comparison is case-insensitive because "#007AFF" and "#007aff" are the same colour to every renderer, and a theme is hand-written.

##### Most of these arms are identities under most bundled themes

A role whose colour is already ink-weight sets its tone equal to itself, deliberately (see each palette for why "this role needs no second tone" is written out rather than left blank). Under DefaultTheme only Primary is in that position; under MaterialTheme, Primary, Success and Error are; under AmberTheme, Success and Error. An arm that is an identity for \*every\* bundled theme could be deleted with no bundled pixel moving.

Primary's used to be exactly that, and AmberTheme is what put it back: amber 700 is a fill that cannot be ink (2.04:1 on white), so its role and its tone are genuinely different colours and deleting this arm repaints every outlined button and loud chip in that theme.

Which arms have a bundled witness and which rest on a test fixture is recorded and checked per role by TestEveryPaletteRuleStillHasAWitness in comps/palette\_witness\_test.go, so a retint that leaves an arm with no evidence anywhere is reported rather than merely true.

<small>[core/theme.go:375](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L375)</small>

#### func (ColorPalette) PrimaryOnLightColor

```go
func (c ColorPalette) PrimaryOnLightColor() string
```

The four on-light resolvers. Each falls back to its own role rather than to a constant — see the field docs for why this fallback is softer than Border's, and note that each defers to the role's \*resolver\* where it has one, so a theme missing both halves of a role still lands somewhere visible.

<small>[core/theme.go:301](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L301)</small>

#### func (ColorPalette) SuccessColor

```go
func (c ColorPalette) SuccessColor() string
```

SuccessColor resolves the Success role, falling back to FallbackSuccess.

<small>[core/theme.go:282](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L282)</small>

#### func (ColorPalette) SuccessOnLightColor

```go
func (c ColorPalette) SuccessOnLightColor() string
```

<small>[core/theme.go:308](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L308)</small>

#### func (ColorPalette) WarningColor

```go
func (c ColorPalette) WarningColor() string
```

WarningColor resolves the Warning role, falling back to FallbackWarning.

<small>[core/theme.go:290](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L290)</small>

#### func (ColorPalette) WarningOnLightColor

```go
func (c ColorPalette) WarningOnLightColor() string
```

<small>[core/theme.go:315](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L315)</small>

### type ComponentDefaults

```go
type ComponentDefaults struct {
	Button   Style `notbackdrop:"a control's own fill, not a surface: Colors.Primary. A bordered control is never drawn on top of a filled button — an outline Button draws its own edge over whatever is behind it, which is the page or a panel, and both of those are already measured. Excluded because the pair is unreachable, not because it is close"`
	Card     Style `backdrop:"a panel: a Card is a container, so anything a screen puts inside one is drawn on this fill. comps.FormField's Input inside a comps.Card is the commonest screen this framework builds, and its frame is a control boundary against exactly this colour"`
	Input    Style `backdrop:"a field's own interior, enclosed by its own frame: the pair here is a boundary against the fill it encircles rather than one control on top of another. Reachable by construction, not by composition — every Input that states a BorderColor builds it, and there is no arrangement of widgets that avoids it"`
	Column   Style `backdrop:"a layout container. It states no fill in any bundled theme, so it contributes no pair today — that is a fact about the themes and not about the geometry. A theme that fills its Column has made it a page region, and every control laid out in one is then drawn on it"`
	Row      Style `backdrop:"a layout container, on the other axis and for the same reason as Column. comps.GroupHeader's band is a filled Row with a bordered control in it the moment a caller styles one, which is the shape that makes this a real pair rather than a hypothetical"`
	Camera   Style `notbackdrop:"a viewfinder: its fill is black in every theme because it is what shows for the frame before the first camera frame arrives, and nothing draws a control boundary on top of a preview. Excluded by name rather than by a lightness test, because a rule that skipped dark fills would also skip a dark theme's page"`
	CheckBox Style `backdrop:"the box's own interior, enclosed by its own boundary — the same by-construction pair as Input. Two of the three bundled themes state a fill here and the third does not, so the pair exists in some palettes and not others, which is what a derived census handles and a hand-written list does not"`
	TextArea Style `backdrop:"a field's own interior, as Input, one tag over. The two are separate fields because a theme may want a taller field to read differently, and they are separate rows in the census for the same reason"`
}
```

ComponentDefaults is the per-component base Style a theme supplies. Every widget merges the caller's props over the field named for it.

#### The notbackdrop tag

A field's Background is, by default, a fill a bordered control can be drawn on — and core.ColorPalette.ControlBorder has WCAG 1.4.11's 3:1 floor against every such fill. internal/palette derives that list by reflecting over this struct, precisely so that adding a field (a Sheet, a Popover) adds a backdrop with nobody having to remember, and comps/variant\_test.go measures the pair.

Every field carries exactly one of two tags, and both are claims about the \*geometry\* of the framework rather than about any number:

	backdrop      what draws a control boundary on this fill
	notbackdrop   why nothing ever does

The value is the argument in both cases and it is required. An exclusion with no reason is a census defeating itself — every excluded field below fails the 3:1 floor in all three bundled themes and would otherwise look like a failure somebody made go away — and an inclusion with no reason is the thing the pair was introduced to end.

#### Why both, and not just the exclusion

The exclusion came first and everything else was measured because it was left over, which had two costs. A pair nothing builds was measured beside a pair three widgets build, so a shortfall in either read the same way to whoever had to fix it — and, worse, a \`notbackdrop\` tag could be \*deleted\* with no consequence but a pair quietly joining the census. Drop Camera's and the added pair clears 6:1, so the run stays green while a geometry claim has been thrown away.

With both tags mandatory a field carrying neither is a hard failure, so deleting either one is now the same kind of event as deleting a field's name. palette.Untagged is that reading and TestEveryComponentFillIsClassified is where it fails.

They live here rather than in a list one package over because this is where a theme author works. "No widget puts a bordered control on this surface" is knowable at the field and is not knowable from a name in internal/palette, and a name in a list is invisible in the diff that adds a field beside it.

palette.IsABackdrop and palette.NotABackdrop read these tags and are their only readers; the reachability claim travels on into the census, which prints it when a pair falls short.

<small>[core/theme.go:447](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L447)</small>

### type SpacingScale

```go
type SpacingScale struct {
	XS, SM, MD, LG, XL int
}
```

<small>[core/theme.go:396](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L396)</small>

### type Theme

```go
type Theme struct {
	Colors     ColorPalette
	Typography Typography
	Spacing    SpacingScale
	Components ComponentDefaults
}
```

<small>[core/theme.go:5](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L5)</small>

### type Typography

```go
type Typography struct {
	Title    Style
	Subtitle Style
	Body     Style
	Caption  Style
}
```

<small>[core/theme.go:389](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L389)</small>

