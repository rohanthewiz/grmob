package tutorial

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// Light and dark: the tutorial's colour scheme, chosen by the host page.
//
// # Who decides
//
// The host page, for the reason it decides the layout (split.go): only it
// knows the reader's OS preference and what they picked in its header. The
// scheme arrives as a host event on the same generic channel, and the app
// holds it as session state:
//
//	page ──HostEvent("theme", {scheme: "dark" | "light"})──▶ app
//
// The page resolves "follow the system" itself and sends only the answer, so
// the app has two states and no media queries. No other host sends the event,
// so the natives, the headless tests and the screenshot host all keep
// DefaultTheme by construction.
//
// # How the palette is swapped
//
// withScheme renders the whole app — Navigator, layout and all — through
// ctx.WithTheme(darkTheme). That copy delegates every hook slot to the
// context it was made from (core.Context.hookOwner), and every scope and
// navigation frame re-inherits the theme on each pass (core.Context.Scope),
// so a toggle mid-lesson repaints in place: no slot moves, no demo loses its
// state, and a frame pushed before the toggle picks the new palette up on the
// next pass like everything else.
//
// It is a context swap rather than core.WithTheme, which wraps its children
// in a "Theme" node. That node would sit above the root only while dark, so
// every toggle would move the whole tree down a level and every host would
// rebuild it, scroll positions included. In the light scheme withScheme is
// transparent: the tree is byte for byte what it was before this file.
//
// # What the page still owns
//
// The screens paint no background of their own (the host's surface shows
// through, as on device), so the guide pane's and the phone glass's fill and
// default ink are wasm/index.html's, in its --screen-bg / --screen-fg tokens,
// which it keeps equal to darkTheme's Background and TextPrimary.

// themeEvent is the host event's name. Its payload is one string field,
// "scheme": "dark" selects darkTheme, anything else DefaultTheme.
const themeEvent = "theme"

// bootTheme is the scheme the page asked for before the app's first render,
// for bootLayout's reasons (split.go): a dark boot then draws its first frame
// dark instead of mounting a light tree and patching every colour in it. The
// same tree counting keeps a scheme sent to one app from becoming the boot
// scheme of the next one built in the same process.
var bootTheme struct {
	mu    sync.Mutex
	trees int // live useColorScheme subscriptions
	dark  bool
}

func init() {
	core.OnHostEvent(themeEvent, func(data map[string]any) {
		scheme, _ := data["scheme"].(string)
		bootTheme.mu.Lock()
		defer bootTheme.mu.Unlock()
		if bootTheme.trees == 0 {
			bootTheme.dark = scheme == "dark"
		}
	})
}

// bootDark is the scheme an app starts in: what the page sent before the
// first render, or light.
func bootDark() bool {
	bootTheme.mu.Lock()
	defer bootTheme.mu.Unlock()
	return bootTheme.dark
}

// themeRecord is useColorScheme's hook-slot memory: layoutRecord's shape, for
// its reason (the subscription is taken once per context tree).
type themeRecord struct {
	mu         sync.Mutex
	subscribed bool
}

// useColorScheme subscribes the app to the page's theme event, once per
// context tree, and releases it when the tree closes. Called on the session
// scope, above the Navigator, because the scheme outlives every frame.
func (t *tutorial) useColorScheme(ctx *core.Context) {
	slot := core.NewState(ctx, &themeRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	already := rec.subscribed
	rec.subscribed = true
	rec.mu.Unlock()
	if already {
		return
	}

	bootTheme.mu.Lock()
	bootTheme.trees++
	bootTheme.mu.Unlock()
	cancel := core.OnHostEvent(themeEvent, func(data map[string]any) {
		scheme, _ := data["scheme"].(string)
		dark := scheme == "dark"
		// Only a change is a Set: the page restates the scheme at boot and
		// whenever the OS preference flips, and an unchanged Set would still
		// request a pass.
		if t.dark.Get() != dark {
			t.dark.Set(dark)
		}
	})
	ctx.OnClose(func() {
		cancel()
		rec.mu.Lock()
		rec.subscribed = false
		rec.mu.Unlock()
		bootTheme.mu.Lock()
		bootTheme.trees--
		if bootTheme.trees == 0 {
			bootTheme.dark = false
		}
		bootTheme.mu.Unlock()
	})
}

// withScheme renders v under darkTheme while the page has asked for it, and
// unchanged otherwise. See the file comment for why this swaps the context
// rather than wrapping the tree in core.WithTheme.
func (t *tutorial) withScheme(v core.View) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		if t.dark.Get() {
			ctx = ctx.WithTheme(darkTheme)
		}
		return v.Render(ctx)
	})
}

// darkTheme is DefaultTheme's dark counterpart: the same type scale, spacing
// and component geometry, repainted in Apple's dark-mode system colours.
//
// It is the tutorial's and not a bundled core theme on purpose. Core's
// palette censuses (the on-light tones, the control-boundary pairs) are
// written against light pages, and a bundled dark theme is a framework
// decision with its own review; this one only has to make the tutorial
// readable at night.
//
// Measured against its own page, #1C1C1E (iOS secondarySystemBackground
// dark, chosen over pure black so the guide reads as a lit sheet next to the
// page chrome rather than a hole in it):
//
//	role              hex        on #1C1C1E   on Surface #2C2C2E
//	TextPrimary       #FFFFFF    17.0:1       13.9:1
//	TextSecondary     #AEAEB2     7.7:1        6.3:1
//	Primary (ink)     #409CFF     6.0:1        4.9:1
//	ControlBorder     #8E8E93     5.2:1        4.3:1   (3:1 floor, WCAG 1.4.11)
//	Border            #38383A     1.5:1        —       (a divider, by design)
//
// # The "on-light" tones are the ink tones here
//
// ColorPalette's four *OnLight fields are named for the light themes they
// were introduced for, but what widgets read them for is "this role, legible
// as ink on the page". On a dark page that is the light end of each hue, so
// they are set to Apple's dark-mode accessible variants rather than left to
// fall back to the fills.
//
// # Primary is a light fill with dark ink
//
// A blue light enough to be read as ink on #1C1C1E (#409CFF) cannot also
// carry white text (2.8:1). Components.Button declares black over it instead
// (7.4:1), and comps.declaredInk carries that pairing to every Badge, filled
// Button and selection that paints Primary. That's the same arrangement
// AmberTheme makes for its light fill.
var darkTheme = newDarkTheme()

func newDarkTheme() *core.Theme {
	// A copy of DefaultTheme, so type sizes, paddings and radii stay in step
	// with it; only colours are restated below. Style and ColorPalette are
	// value types, so assigning their fields never writes DefaultTheme. The
	// two slices are replaced outright rather than edited, for the same
	// reason.
	th := *core.DefaultTheme

	const (
		bg        = "#1C1C1E" // iOS secondarySystemBackground (dark)
		surface   = "#2C2C2E" // iOS tertiarySystemBackground (dark)
		ink       = "#FFFFFF"
		inkDim    = "#AEAEB2" // iOS systemGray2 (dark), opaque so contrast math is exact
		primary   = "#409CFF" // iOS accessible blue (dark)
		control   = "#8E8E93" // iOS systemGray
		separator = "#38383A" // iOS opaqueSeparator (dark)
	)

	th.Colors = core.ColorPalette{
		Primary:       primary,
		Secondary:     "#30D158", // iOS systemGreen (dark)
		Background:    bg,
		Surface:       surface,
		TextPrimary:   ink,
		TextSecondary: inkDim,
		Error:         "#FF453A", // iOS systemRed (dark)
		Border:        separator,
		ControlBorder: control,
		Success:       "#30D158", // iOS systemGreen (dark)
		Warning:       "#FF9F0A", // iOS systemOrange (dark)

		// Ink-weight tones on the dark page; see the doc above.
		PrimaryOnLight: primary,   // already ink: 6.0:1
		SuccessOnLight: "#30DB5B", // accessible green (dark)  — 9.2:1
		WarningOnLight: "#FFB340", // accessible orange (dark) — 9.5:1
		ErrorOnLight:   "#FF6961", // accessible red (dark)    — 6.0:1

		// The dark-page series and quantity lists core already ships for
		// exactly this (their first bundled consumer).
		Chart:      core.DefaultDarkChartColors(),
		Sequential: core.DefaultDarkSequentialColors(),
	}

	th.Typography.Title.TextColor = ink
	th.Typography.Subtitle.TextColor = inkDim
	th.Typography.Body.TextColor = ink
	th.Typography.Caption.TextColor = inkDim

	th.Components.Button.Background = primary
	th.Components.Button.TextColor = "#000000" // 7.4:1 over #409CFF
	// Cards are lifted one step off the page, as on iOS, where a grouped
	// cell is lighter than the background it sits on.
	th.Components.Card.Background = surface
	// Fields keep DefaultTheme's shape: the page's own fill, framed by the
	// boundary tone.
	th.Components.Input.Background = bg
	th.Components.Input.TextColor = ink
	th.Components.Input.BorderColor = control
	th.Components.TextArea.Background = bg
	th.Components.TextArea.TextColor = ink
	th.Components.TextArea.BorderColor = control
	th.Components.CheckBox.Background = bg

	return &th
}
