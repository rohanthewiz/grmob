package core

// OpenURL asks the host to open a URL outside the app — the platform's own
// browser, mail composer, dialer or map, whichever the scheme names.
//
// It is a system event (see sys_events.go) rather than a node, for the same
// reason ShowToast is: nothing about it is part of the view tree. There is no
// element to reconcile, no state to diff, and the thing it ultimately reaches
// is an OS-level facility the app does not own. So it travels one way, as a
// named payload handed to whatever host is driving the app, and — like a
// toast — it is callable from any goroutine and takes no Context.
//
// Each host maps the event onto its platform's own hand-off:
//
//	Android   Intent(ACTION_VIEW, uri), started with FLAG_ACTIVITY_NEW_TASK
//	iOS       UIApplication.shared.open(url)
//	Browser   window.open(url, "_blank", "noopener,noreferrer")
//	Headless  nothing (no handler registered — see SendSystemEvent)
//
// # Why "outside the app" is the whole contract
//
// The three platforms differ on almost everything about in-app browsing —
// Custom Tabs versus SFSafariViewController versus an iframe — and agree
// completely on handing a URL to the system. So this promises only the part
// that is portable. An app that needs an embedded browser wants a view node,
// which is a different (and much larger) feature.
//
// # Failure is silent, and that is deliberate
//
// A malformed URL, a scheme no app on the device claims (a `tel:` link on a
// tablet with no dialer), or a host that registered no handler at all: none
// of these come back. There is no return channel on a system event, and
// synthesizing one would mean either blocking the caller on an OS round trip
// or inventing a callback protocol for a fire-and-forget gesture. Callers
// that must know whether a link is reachable have to decide that before
// calling — which in practice means not rendering the affordance at all,
// exactly as core.Button does with a nil handler.
//
// An empty url is dropped here rather than sent, since every host would have
// to reject it separately and none could report that it had.
func OpenURL(url string) {
	if url == "" {
		return
	}
	SendSystemEvent("open_url", map[string]any{"url": url})
}
