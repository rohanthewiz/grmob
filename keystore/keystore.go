// Package keystore keeps small secrets — a bearer token, a refresh token, an
// API key — in the platform's own secure store rather than in the app's files:
// the Keychain on iOS, and on Android values sealed by a key that lives inside
// the Android Keystore and never leaves it.
//
// It is for credentials, not data. A value is one string, written and read
// whole; an app's records belong in bytdb (see examples/todoapp), and the
// token that lets the app fetch them belongs here.
//
//	keystore.Save("token", tok, func(err error) { … })
//	keystore.Get("token", func(tok string, found bool, err error) { … })
//	keystore.Delete("token", func(err error) { … })
//
// # What kind of thing this is
//
// permission's package comment draws the lines between what crosses the
// app/host boundary — a node is reconciled, a service is commanded, a sensor
// is subscribed, a permission is asked. A keystore call is a fifth thing, a
// *question with its own answer*, and it has the shape core.ReadClipboard
// already uses for that:
//
//	Save/Get/Delete ──"keystore" system event {command, id, key, value}──▶ host
//	done(…)         ◀──"keystore" host event  {id, ok, found, value, error}── host
//
// Each call carries an id, the host echoes it, and the reply runs exactly the
// callback that asked. A record — permission's shape, one process-wide value
// that every screen reads — would be wrong twice over: the answer to "did
// *this* save land" belongs to the code that saved, and a secret copied into
// a package-level map would outlive every caller that wanted it.
//
// # Why only callbacks, never a blocking Get
//
// The reply comes back through the same serial path a render pass runs on —
// render.Manager.Dispatch on the natives, the page's event loop in the
// browser. A Get that blocked until its answer arrived, called from a tap
// handler or a render, would hold that path while waiting for a reply that
// can only be delivered once it lets go: a deadlock on the natives, a frozen
// page in the browser. So a screen that needs a secret at launch asks on
// mount and renders a placeholder until the callback sets its state, which
// is also the honest picture of a read that may touch secure hardware.
//
// # Where the bytes live
//
//	iOS       Keychain generic-password items: service "grmob.keystore",
//	          account = key, accessible AfterFirstUnlockThisDeviceOnly.
//	Android   AES-256-GCM in a private SharedPreferences file, under a key
//	          generated inside AndroidKeyStore (hardware-backed where the
//	          device has a TEE). The entry's name is the cipher's associated
//	          data, so one entry's ciphertext cannot be replayed as another's.
//	Browser   Refused: every call fails with ErrUnavailable. See below.
//	Headless  No host to ask (a test, htmlout): ErrUnavailable at once, on
//	          the caller's goroutine.
//
// AfterFirstUnlock rather than WhenUnlocked because a credential is exactly
// what a background launch needs — an audio app resumed after a reboot, a
// notification action — and ThisDeviceOnly because a secret restored onto a
// different phone from a backup is one the user never put there. Android's
// key is device-bound by construction, so the two platforms agree: a value
// never survives a move to new hardware.
//
// They also agree on uninstall. Android deletes the preferences file and the
// key with the app. iOS keeps Keychain items after an app is deleted, so its
// host wipes this service's items the first time it runs in a fresh install
// (it keeps an "installed" marker in UserDefaults, which uninstall does
// delete). Without that, a user who deleted the app to sign out would find
// themselves signed in again on reinstall — on one platform only.
//
// # Why the browser refuses rather than falling back
//
// A page has no secure store. localStorage, IndexedDB and cookies readable
// from script are all open to any script on the origin, which is precisely
// the cross-site-scripting threat a keystore exists to keep a token away
// from. Quietly storing there would make Save "succeed" while delivering none
// of what its name promises, so the browser runtime answers every call with
// ErrUnavailable and the app chooses its own fallback, knowing what it is
// choosing. A web build that wants a session should get it from an HttpOnly
// cookie, which script cannot read at all.
//
// # Errors
//
//	nil             the call did what it says; Get's found reports whether
//	                the key held a value
//	ErrUnavailable  this host has no secure store (the browser, a headless
//	                run). Expected, permanent for the process, and the cue to
//	                fall back to whatever the app did before it had a keystore
//	anything else   the store exists but this call failed — a Keychain error,
//	                a key the Android Keystore would not use. The message
//	                carries the host's reason and the key, never the value
//
// Android can lose its key out from under the data: a backup restored onto a
// reinstall brings back the preferences file but not the key that sealed it.
// Its host notices when it has to create a fresh key, discards the entries
// the old one sealed, and Get then answers found=false — the same answer as
// a store that never held the key, which is what the app should treat it as.
//
// # Threading and lifetime
//
// done runs on whichever goroutine delivered the host event: on the natives
// that is inside render.Manager.Dispatch, so a State.Set from it is rendered
// by the pass that follows. It must not block. ErrUnavailable from a headless
// run is the one answer delivered synchronously, on the caller's goroutine.
//
// Every shipped host answers every call, failures included, so there is no
// timeout — the same contract, and the same reasoning, as core.ReadClipboard:
// a timeout would mean a goroutine per call to cover a host bug.
// mobile/verify holds the three shells to this file's spellings.
package keystore

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// ErrUnavailable is the answer from a host with no secure store: the browser,
// which refuses on purpose (see the package comment), and any process with no
// host registered at all. Compare with errors.Is.
var ErrUnavailable = errors.New("keystore: no secure store on this host")

// ErrEmptyKey is returned, without asking the host, for a call with key "".
// Both platforms would accept it — an empty Keychain account, an empty
// preferences name — but it is always a caller's bug, and one that would
// otherwise surface as two unrelated features sharing a slot.
var ErrEmptyKey = errors.New("keystore: empty key")

// The wire contract. The request and its reply share an event name for the
// reason core's clipboard gives: they are one feature, and a shell author
// looking for either finds both.
const (
	systemEventName = "keystore"
	hostEventName   = "keystore"

	commandSave   = "save"
	commandGet    = "get"
	commandDelete = "delete"

	// reasonUnavailable is the one reply reason with a meaning of its own: a
	// host that has no secure store says so with this, and Go maps it to
	// ErrUnavailable so callers can tell "not here" from "failed here". Any
	// other reason is free text for a log.
	reasonUnavailable = "unavailable"
)

// call is one request waiting for its reply. command and key are kept only
// to build an error message that says which call failed; the value is never
// kept, so a pending save does not hold a second copy of the secret.
type call struct {
	command string
	key     string
	answer  func(r reply)
}

// reply is a decoded host answer, before it is spread over the caller's
// callback parameters.
type reply struct {
	found bool
	value string
	err   error
}

var (
	mu      sync.Mutex
	pending = map[string]call{}
	next    uint64
)

// init subscribes to the host's replies.
//
// At package init rather than on first call, matching permission: a reply
// can only answer a request this package sent, so there is no early traffic
// to catch, but subscribing once here keeps send free of a sync.Once and
// keeps the subscription out of every caller's path.
func init() {
	core.OnHostEvent(hostEventName, receive)
}

// Save stores value under key, replacing whatever the key held, and calls
// done with the outcome exactly once.
//
// done may be nil when the caller has nothing to do with the answer; a
// failure is then logged (with the key, never the value) rather than lost.
// The empty string is a value like any other: Save("k", "") stores it, and
// Get then reports found=true with "".
func Save(key, value string, done func(err error)) {
	if done == nil {
		done = logFailure(commandSave)
	}
	send(commandSave, key, value, func(r reply) { done(r.err) })
}

// Get reads the value stored under key and calls done exactly once: with the
// value and found=true when there is one, with ("", false, nil) when the key
// holds nothing, and with a non-nil err when the store could not be asked or
// could not answer (see the package comment's Errors section).
//
// A nil done makes the call a no-op: a read whose answer nobody receives has
// nothing to do, and sending it anyway would only move the secret across the
// bridge for no one.
func Get(key string, done func(value string, found bool, err error)) {
	if done == nil {
		return
	}
	send(commandGet, key, "", func(r reply) { done(r.value, r.found, r.err) })
}

// Delete removes key and calls done exactly once. Deleting a key that holds
// nothing succeeds: sign-out clears the token whether or not one was saved,
// and a caller should not have to Get first to avoid an error.
//
// done may be nil; a failure is then logged, as for Save.
func Delete(key string, done func(err error)) {
	if done == nil {
		done = logFailure(commandDelete)
	}
	send(commandDelete, key, "", func(r reply) { done(r.err) })
}

// send registers the pending call and emits the request, or answers at once
// when there is nothing to ask.
//
// The callback is registered before the event goes out because a host may
// reply synchronously, inside SendSystemEvent — the same ordering
// core.ReadClipboard keeps, for the same reason. The value is put on the
// wire only for a save; a get or delete has none to carry.
func send(command, key, value string, answer func(reply)) {
	if key == "" {
		answer(reply{err: ErrEmptyKey})
		return
	}
	if !core.HasSystemEventHandler() {
		// The headless case: no platform at all, so a caller waiting on a
		// reply that can never come would be a hang rather than a
		// degradation. Answered on the caller's goroutine.
		answer(reply{err: ErrUnavailable})
		return
	}

	mu.Lock()
	next++
	// A decimal string rather than a number: it crosses JSON twice, and each
	// host's number type differs, which a string survives byte for byte.
	id := strconv.FormatUint(next, 10)
	pending[id] = call{command: command, key: key, answer: answer}
	mu.Unlock()

	data := map[string]any{
		"command": command,
		"id":      id,
		"key":     key,
	}
	if command == commandSave {
		data["value"] = value
	}
	core.SendSystemEvent(systemEventName, data)
}

// receive decodes one "keystore" host event, the reply to one call. The
// payload every host writes:
//
//	id     string  the id from the request, echoed
//	ok     bool    the call succeeded; absent means false
//	found  bool    get only: the key held a value
//	value  string  get only, when found: the stored value
//	error  string  when !ok: the reason, or "unavailable" for a host with no
//	               secure store
//
// An id with no pending call — a duplicate reply, or one arriving after a
// test reset — is dropped. A missing ok is a failure rather than a success,
// so a shell that forgets the key fails visibly (the app sees an error)
// instead of reporting a save that never happened.
func receive(data map[string]any) {
	id, _ := data["id"].(string)
	if id == "" {
		return
	}
	mu.Lock()
	c, ok := pending[id]
	delete(pending, id)
	mu.Unlock()
	if !ok {
		return
	}
	// Outside the lock: the callback may start another call.
	c.answer(decode(c, data))
}

// decode turns one reply payload into the caller's answer.
//
// found and value are read only for a get, and only when the call succeeded:
// a host that echoes stray keys on a save or a failure must not be able to
// hand a caller a value it never asked for.
func decode(c call, data map[string]any) reply {
	if ok, _ := data["ok"].(bool); !ok {
		reason, _ := data["error"].(string)
		switch reason {
		case reasonUnavailable:
			return reply{err: ErrUnavailable}
		case "":
			reason = "the host gave no reason"
		}
		return reply{err: fmt.Errorf("keystore: %s %q: %s", c.command, c.key, reason)}
	}
	if c.command != commandGet {
		return reply{}
	}
	if found, _ := data["found"].(bool); !found {
		return reply{}
	}
	value, _ := data["value"].(string)
	return reply{found: true, value: value}
}

// logFailure is the done a nil callback becomes: silent on success and on
// ErrUnavailable (a headless run or the browser, where every call fails the
// same expected way and a line per call would only be noise), one log line on
// a real failure. A host failure's message already names the command and the
// key (see decode); the empty-key error does not, so the command is added.
func logFailure(command string) func(error) {
	return func(err error) {
		if err != nil && !errors.Is(err, ErrUnavailable) {
			log.Printf("grmob: %s: %v", command, err)
		}
	}
}

// resetForTest drops every pending call so one test's unanswered request
// cannot be answered by the next test's host.
func resetForTest() {
	mu.Lock()
	defer mu.Unlock()
	pending = map[string]call{}
}
