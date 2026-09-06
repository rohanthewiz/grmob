package social

import (
	. "github.com/rohanthewiz/grmob/core"
)

// App is the root view of the social example: a Navigator whose initial route
// is a tabbed shell (home / search / profile), with the pages themselves in
// pages.go.
//
// It lives here rather than in the host (wasm/main.go) so the example is a
// complete application in one package — importable by a test, a WASM shell, or
// a native shell without any of them re-deriving the wiring. A host's job is
// to own the transport (JS bindings, the render.Manager, event plumbing); the
// app tree is the example's job.
//
// The whole shell is one route so Push/Pop layer *over* the tab bar: pushing
// DetailsPage replaces the tabbed screen entirely, and Pop restores it with the
// selected tab intact — `currentTab` lives in the root frame's own scope, and
// the root frame never leaves the stack. See DetailsPage for the history of
// that guarantee.
//
// Where state must outlive a frame, put it above the Navigator. The `ctx`
// parameter here is the context the Navigator renders into, so
// `ctx.Scope("session")` (captured by the route closures below) would survive
// a Reset that discarded every screen — the shape to reach for when something
// like a draft message or an upload queue must not die with the screen that
// started it.
func App(ctx *Context) View {
	return Navigator(func(ctx *Context) View {
		currentTab := NewState(ctx, "home")

		return Column(
			// The region the strip switches, wrapped so it has an element of
			// its own to be named by. Match returns whichever page is showing
			// and a page is not the strip's to annotate, so the id and the
			// name live on a Box around it — the same reason core.TabView
			// imposes its panel attributes on the page's element rather than
			// asking the page to carry them.
			//
			// The name changes with the tab, because what the region *is*
			// changes with the tab: a reader following a tab's aria-controls
			// should arrive somewhere that says which page it landed on. It is
			// aria-label rather than TabView's aria-labelledby for the reason
			// core.Style.AccessibilityID gives — a reference earns its place
			// only when what it points at cannot be said as a value, and this
			// can.
			//
			// The Box also picks up role="group" from both web exporters,
			// which is what makes the name audible; see core.RoleGroup.
			Box(
				AccessibilityID(TabPanelID),
				AccessibilityLabel(tabNames[currentTab.Get()]),
				Match(currentTab.Get(),
					Case("home", HomePage(ctx)),
					Case("search", SearchPage(ctx)),
					Case("profile", ProfilePage(ctx)),
				),
			),
			Row( // tab bar
				// A strip of three tabs and nothing else, which is what lets
				// it take the role at all: role="tablist" claims its children
				// are tabs, and a bar that also held a count or a "+" would be
				// making a claim it could not keep. See "A structural role
				// owns what is inside it" in core/role.go.
				AccessibilityRole(RoleTabList),
				TabButton("🏠", "home", tabNames["home"], currentTab),
				TabButton("🔍", "search", tabNames["search"], currentTab),
				TabButton("👤", "profile", tabNames["profile"], currentTab),
			),
		)
	})
}

// tabNames is what each tab and its region are called, in one place, because
// the two ends of the relationship have to agree: the tab announces "Início,
// tab, selected" and a reader following its aria-controls arrives at a region
// named "Início". Two literals would let those drift into naming the same
// thing differently, which is worse than either name alone.
var tabNames = map[string]string{
	"home":    "Início",
	"search":  "Pesquisa",
	"profile": "Perfil",
}
