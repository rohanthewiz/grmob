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
// selected tab intact, because the tab state lives in a slot that the pushed
// route never touches. See DetailsPage for what "never touches" requires.
func App(ctx *Context) View {
	return Navigator(func(ctx *Context) View {
		currentTab := NewState(ctx, "home")

		return Column(
			Match(currentTab.Get(),
				Case("home", HomePage(ctx)),
				Case("search", SearchPage(ctx)),
				Case("profile", ProfilePage(ctx)),
			),
			Row( // tab bar
				TabButton("🏠", "home", currentTab),
				TabButton("🔍", "search", currentTab),
				TabButton("👤", "profile", currentTab),
			),
		)
	})
}
