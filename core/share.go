package core

import (
	"strconv"
	"sync"
)

// The system share sheet: hand the platform a text and/or a link, and let the
// person pick where it goes (a message, a mail, another app).
//
//	app ──SendSystemEvent("share", {id: "7", text, url})──▶ host
//	app ◀──ReceiveHostEvent("share", {id: "7", ok})──────── host
//
//	host   sheet
//	web    navigator.share (Web Share API)
//	Android  Intent.ACTION_SEND through Intent.createChooser
//	iOS    UIActivityViewController
//
// # Why there is an answer at all
//
// OpenURL and WriteClipboard are fire-and-forget, and a share could be too,
// except for one host: a desktop browser often has no Web Share API (Firefox
// on a desktop has none), and a share button that silently does nothing there
// is a defect. So every request carries an id, the host echoes it with ok,
// and a caller that has a fallback can run it. comps.Gallery's Share puts
// the text on the clipboard with a toast when ok is false.
//
// It is ReadClipboard's mould (see clipboard.go for why a correlation id
// rather than a shared record).
//
// # What ok means
//
//	ok=true    the sheet was shown. Whether the person then sent it anywhere
//	           or dismissed it is not reported: Android's chooser does not
//	           say, and a caller should not treat a dismissal as a failure
//	           that needs a fallback.
//	ok=false   no sheet could be shown: the browser has no Web Share API or
//	           refused the call (no user gesture, an insecure context), no
//	           app on the device takes the intent, there is no window to
//	           present from, or no host is registered at all.
//
// # Threading and lifetime
//
// As ReadClipboard: done runs on whichever goroutine delivered the host event,
// exactly once, and every shipped host answers every request. A host that
// never answers leaves done pending for the life of the process.

// systemEventShare and hostEventShare share a spelling, as the clipboard's
// do. Held to the shells by mobile/verify's TestShareEventSpellingsAgree.
const (
	systemEventShare = "share"
	hostEventShare   = "share"
)

var (
	shareMu      sync.Mutex
	sharePending = map[string]func(ok bool){}
	shareNext    uint64
)

// Share opens the platform's share sheet with text and url, and calls done
// with whether a sheet was shown: true even if the person then dismisses it,
// false when none could be (a browser without the Web Share API, no app
// to take it, no host). A caller with a fallback, such as copying the link,
// runs it on false.
//
// Either of text and url may be empty, not both: with nothing to share, done
// runs at once with false and nothing is sent. The two travel separately
// because the platforms place them separately (the Web Share API's text and
// url, a link preview above the text on iOS); a host with one slot (Android's
// EXTRA_TEXT) joins them with a newline.
//
// done may be nil, for a caller with no fallback. With no host registered (a
// headless test, htmlout) done runs immediately with false on the caller's
// goroutine, rather than leaving the caller waiting on a reply that cannot
// come.
//
// Call it from a tap handler. A browser shows the sheet only during a user
// gesture, and the natives present it over the screen the person is looking
// at, so a share from a timer can be refused on the web.
func Share(text, url string, done func(ok bool)) {
	if done == nil {
		done = func(bool) {}
	}
	if text == "" && url == "" {
		done(false)
		return
	}
	if !HasSystemEventHandler() {
		done(false)
		return
	}
	// Registered before the event is sent: a host may answer synchronously
	// inside SendSystemEvent, and the reply must find done already waiting.
	shareMu.Lock()
	shareNext++
	id := strconv.FormatUint(shareNext, 10)
	sharePending[id] = done
	shareMu.Unlock()

	SendSystemEvent(systemEventShare, map[string]any{
		"id":   id,
		"text": text,
		"url":  url,
	})
}

// receiveShare decodes the "share" host event, the reply to one request:
//
//	id  string  the id from the request, echoed
//	ok  bool    whether a sheet was shown; absent means false
//
// An id with no pending request (a duplicate reply, or one after a test
// reset) is dropped. A missing ok reads as false, so a shell that forgets the
// key falls back to the caller's alternative rather than doing nothing.
func receiveShare(data map[string]any) {
	id, _ := data["id"].(string)
	if id == "" {
		return
	}
	shareMu.Lock()
	done := sharePending[id]
	delete(sharePending, id)
	shareMu.Unlock()
	if done == nil {
		return
	}
	ok, _ := data["ok"].(bool)
	// Outside the lock: done may share again.
	done(ok)
}

// resetShareForTest drops every pending request so one test's unanswered
// share cannot be answered by the next test's host.
func resetShareForTest() {
	shareMu.Lock()
	defer shareMu.Unlock()
	sharePending = map[string]func(bool){}
}
