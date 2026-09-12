package core

// Deep links: the URL the OS hands an app that was opened by tapping a link.
//
//	OS ──▶ shell ──HostEvent("deeplink", {url})──▶ OnDeepLink ──▶ app
//
// This is the inbound half of OpenURL. That function is a system event going
// out — "hand this address to whatever claims it" — and until now there was no
// way back in: an app opened by a link was opened at its front door with the
// address thrown away.
//
// # Why core carries the name and not the parsing
//
// A URL means whatever the app that registered the scheme says it means, so
// the shells forward the string and stop. Anything else would put an app's
// vocabulary in the shell: examples/tutorial reads grmob://lesson/4.12 and
// navigates to a lesson, and a shell that knew the word "lesson" would be a
// shell that only serves the tutorial.
//
// That leaves core with one job worth doing — saying what the payload is, in
// one place, so three shells write the same key. OnDeepLink is a typed wrapper
// over OnHostEvent for exactly that reason; the event is not consumed by core
// (see ReceiveHostEvent's switch), because there is no core state for a URL to
// update.
//
// # What each host does
//
// Android   an <intent-filter> for ACTION_VIEW on the scheme, plus
//           launchMode="singleTop" so a link arriving at a running app reaches
//           onNewIntent instead of starting a second Activity. Without the
//           launch mode the app is recreated and the Go side — a process-wide
//           singleton — is fine, but the back stack grows a duplicate.
// iOS       CFBundleURLTypes in Info.plist and SwiftUI's .onOpenURL. The cold
//           and warm cases are the same callback, which is why the shell needs
//           no equivalent of onNewIntent.
// Browser   nothing here. The page already owns the address bar and translates
//           it, which is the arrangement examples/tutorial's deeplink.go
//           describes: the hash is a "route" host event the page sends. A
//           native shell has no address bar and cannot do that translation,
//           which is the gap this fills.
// Headless  nothing. A Go test calls the app's own handler, or ReceiveHostEvent
//           directly.
//
// # The cost that was worth paying it off
//
// Before this, reaching a particular screen on a device meant scrolling to it.
// The tutorial is 51 lessons and its map lesson is 4.12, so every device or
// simulator run of that lesson began with a bounded scroll loop — and both
// device harnesses in this repository still contain one, because the scroll is
// also how a reader gets there. What changes is that a *run* no longer has to:
//
//	adb shell am start -a android.intent.action.VIEW -d "grmob://lesson/4.12"
//	xcrun simctl openurl booted "grmob://lesson/4.12"

// hostEventDeepLink is the name all three shells report an inbound URL under.
// The payload is one string field, "url", carrying it verbatim — not parsed,
// not validated, and not guaranteed to be a URL at all, since what arrives is
// whatever the OS was asked to open.
const hostEventDeepLink = "deeplink"

// OnDeepLink subscribes fn to inbound URLs. The returned function cancels the
// subscription.
//
//	cancel := core.OnDeepLink(func(url string) {
//	    if id, ok := strings.CutPrefix(url, "grmob://lesson/"); ok {
//	        navigate(id)
//	    }
//	})
//
// fn runs on whichever goroutine delivered the event — a host bridge call —
// and must not block; the usual body parses the URL and asks for a render.
//
// An empty or absent "url" field is dropped rather than delivered as "": a
// subscriber cannot do anything useful with no address, and a shell that
// reported one has a bug this hides less well than it would pass on.
func OnDeepLink(fn func(url string)) (cancel func()) {
	return OnHostEvent(hostEventDeepLink, func(data map[string]any) {
		url, _ := data["url"].(string)
		if url == "" {
			return
		}
		fn(url)
	})
}
