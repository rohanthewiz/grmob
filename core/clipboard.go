package core

import (
	"strconv"
	"sync"
)

// The system clipboard: write text to it, or ask for the text on it.
//
// Writing is a plain system event, the same one-way shape as OpenURL — the
// app hands the platform something to hold and has nothing to hear back.
// Reading is the first call in core that needs an answer to a specific
// question, and neither existing channel shape fits it on its own:
//
//	app ──SendSystemEvent("clipboard", {command: "read", id: "7"})──▶ host
//	app ◀──ReceiveHostEvent("clipboard", {id: "7", text, ok})──────── host
//
// # Why a correlation id, when the permission package has none
//
// permission answers into a record: the status of the camera is one fact
// about the device, a second Check overwrites the first with the same truth,
// and every screen reading the record wants the latest answer. A clipboard
// read is not a fact to cache — the contents change underneath the app, from
// other apps, with no event — and the caller wants *its* answer delivered to
// *its* continuation (the composer that pressed Paste), not a shared value it
// has to poll. So each read carries an id, the host echoes it, and the
// answer runs exactly the callback that asked. Two reads in flight at once
// (a double tap) each get their own reply and never each other's.
//
// The id is a string rather than a number because it crosses JSON twice and
// every host's number type differs; a decimal string survives all of them
// byte for byte.
//
// # What ok means
//
//	ok=true,  text=""    the clipboard holds no text (empty, or an image)
//	ok=true,  text="…"   the text on the clipboard
//	ok=false             the platform refused or cannot read: the browser
//	                     denied the permission or has no async clipboard, the
//	                     user declined iOS's paste prompt, or no host is
//	                     registered at all
//
// A caller that only wants to paste can treat both "" cases the same way;
// the distinction exists so a screen can explain a refusal instead of
// silently pasting nothing.
//
// # Threading and lifetime
//
// The callback runs on whichever goroutine delivered the host event (see
// host_events.go): on the natives that is inside render.Manager.Dispatch, so
// a State.Set from it is rendered by the pass that follows. It must not
// block. A host that never answers leaves its callback pending for the life
// of the process — every shipped host answers every read, including with
// ok=false on failure, so there is no timeout; adding one would mean a
// goroutine per read to cover a host bug.

// systemEventClipboard and hostEventClipboard share a spelling: the request
// and its reply are one feature, and a shell author looking for either finds
// both. Held to the shells by mobile/verify's TestClipboardEventSpellingsAgree.
const (
	systemEventClipboard = "clipboard"
	hostEventClipboard   = "clipboard"
	clipboardWrite       = "write"
	clipboardRead        = "read"
)

var (
	clipboardMu      sync.Mutex
	clipboardPending = map[string]func(text string, ok bool){}
	clipboardNext    uint64
)

// WriteClipboard puts text on the system clipboard. Fire-and-forget, like
// OpenURL: callable from any goroutine, silent when no host is registered.
//
// An empty string is sent rather than dropped (unlike OpenURL's empty url):
// clearing the clipboard — after copying a one-time code, say — is a
// legitimate thing to ask for.
func WriteClipboard(text string) {
	SendSystemEvent(systemEventClipboard, map[string]any{
		"command": clipboardWrite,
		"text":    text,
	})
}

// ReadClipboard asks the host for the clipboard's text and calls fn with the
// answer, exactly once. See the file comment for what ok means.
//
// With no host registered (a headless test, htmlout) fn runs immediately
// with ("", false) on the caller's goroutine: there is no platform to ask,
// and a caller waiting on a reply that can never come would be a hang, not
// a degradation.
//
// fn is registered before the event is sent, because a native host may
// answer synchronously inside SendSystemEvent's call and the reply must find
// its callback already waiting.
func ReadClipboard(fn func(text string, ok bool)) {
	if fn == nil {
		return
	}
	if !HasSystemEventHandler() {
		fn("", false)
		return
	}
	clipboardMu.Lock()
	clipboardNext++
	id := strconv.FormatUint(clipboardNext, 10)
	clipboardPending[id] = fn
	clipboardMu.Unlock()

	SendSystemEvent(systemEventClipboard, map[string]any{
		"command": clipboardRead,
		"id":      id,
	})
}

// receiveClipboard decodes the "clipboard" host event, the reply to one
// read. The payload is the contract every host writes:
//
//	id    string  the id from the read request, echoed
//	text  string  the clipboard's text; absent or "" when it holds none
//	ok    bool    false when the platform refused; absent means false
//
// An id with no pending read — a duplicate reply, or one arriving after a
// test reset — is dropped. A missing ok is read as a refusal rather than a
// success so a shell that forgets the key fails visibly (nothing pastes)
// instead of pasting "" as if the clipboard were empty.
func receiveClipboard(data map[string]any) {
	id, _ := data["id"].(string)
	if id == "" {
		return
	}
	clipboardMu.Lock()
	fn := clipboardPending[id]
	delete(clipboardPending, id)
	clipboardMu.Unlock()
	if fn == nil {
		return
	}
	text, _ := data["text"].(string)
	ok, _ := data["ok"].(bool)
	// Outside the lock: the callback may start another read.
	fn(text, ok)
}

// resetClipboardForTest drops every pending read so one test's unanswered
// request cannot be answered by the next test's host.
func resetClipboardForTest() {
	clipboardMu.Lock()
	defer clipboardMu.Unlock()
	clipboardPending = map[string]func(string, bool){}
}
