package tutorial

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
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
// the app has two states and no media queries.
//
// # Until a page says, the system does
//
// No other host sends the event. The natives, which have no page, follow the
// system's own dark mode instead: every host reports it on the window record
// (core.Window.ColorScheme), and the scheme is the window's until the first
// "theme" event arrives. After that the page's choice wins for the rest of
// the session, which is what lets its Light button hold on a dark OS.
//
//	page sent "theme"?   scheme
//	──────────────────   ───────────────────────────────
//	yes                  the page's (Light, Dark, or the OS as it resolved it)
//	no                   hooks.UseWindow's ColorScheme; light when unreported
//
// The headless tests and the screenshot host report neither, so they keep
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
	said  bool // a "theme" event arrived; the page's choice, not the window's
}

func init() {
	core.OnHostEvent(themeEvent, func(data map[string]any) {
		scheme, _ := data["scheme"].(string)
		bootTheme.mu.Lock()
		defer bootTheme.mu.Unlock()
		if bootTheme.trees == 0 {
			bootTheme.dark = scheme == "dark"
			bootTheme.said = true
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

// bootSaid is whether that scheme came from a page at all, rather than being
// the default. See "Until a page says, the system does".
func bootSaid() bool {
	bootTheme.mu.Lock()
	defer bootTheme.mu.Unlock()
	return bootTheme.said
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
		if !t.pageSaid.Get() {
			t.pageSaid.Set(true)
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
			bootTheme.said = false
		}
		bootTheme.mu.Unlock()
	})
}

// withScheme renders v under darkTheme while the page has asked for it, or,
// before any page has, while the system is dark; unchanged otherwise. See the
// file comment for why this swaps the context rather than wrapping the tree
// in core.WithTheme.
//
// hooks.UseWindow is called on every pass whichever source wins, so the hook
// slot never moves, and a dark-mode switch on a native re-renders through it.
// On Android the switch recreates the Activity and the first frame after it
// is drawn before the new report lands, so it can show the old scheme for a
// frame; the Go app outlives the Activity, so that is a repaint, not a
// restart.
func (t *tutorial) withScheme(v core.View) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		fromPage := t.pageSaid.Get()
		dark := hooks.UseWindow(ctx).Dark()
		if fromPage {
			dark = t.dark.Get()
		}
		if !dark {
			return v.Render(ctx)
		}
		n := v.Render(ctx.WithTheme(darkTheme))
		if fromPage {
			return n
		}
		return paintPage(n, darkTheme.Colors.Background, darkTheme.Colors.TextPrimary)
	})
}

// paintPage gives a native's root the dark page colour and ink the web page
// sets in CSS. The screens paint no background of their own (the file comment), so
// on a native the shell's surface shows through, and the shells' surfaces are
// light: a dark-mode Android drew darkTheme's white ink on a near-white
// window (seen on the emulator). A style on the root the tree already has,
// not a wrapper node, for withScheme's reason: a wrapper present only while
// dark would move the whole tree down a level on every switch.
//
// Only for the window's scheme. A page that sends "theme" paints its own
// panes (--screen-bg), and the split's root is not a page, so the web's tree
// stays what it was. A copy, because withLayout remembers the lesson node it
// was handed and a shared Style would be painted in the light tree too.
func paintPage(n *core.Node, color, ink string) *core.Node {
	if n == nil {
		return nil
	}
	var st core.Style
	if n.Style != nil {
		st = *n.Style
	}
	if st.Background != "" {
		return n
	}
	st.Background = color
	// The page's ink too, which every Text that states none inherits
	// (core.TextColor): the web page's --screen-fg. Without it a lesson's
	// plain core.Text drew Compose's default black on the dark page.
	if st.TextColor == "" {
		st.TextColor = ink
	}
	painted := *n
	painted.Style = &st
	return &painted
}

// darkTheme is core.DarkTheme. It was this file's own until 2026-10-03, when
// it moved into core as a bundled theme (N-074) with its figures measured by
// the palette censuses rather than written out here; the name stays so the
// rest of the tutorial, and wasm/index.html's tokens that track its page
// colour and ink, read as they did.
var darkTheme = core.DarkTheme
