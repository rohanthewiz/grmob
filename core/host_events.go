package core

import (
	"log"
	"sync"
)

// Host events are the reverse of system events: host→app notifications that
// are not answers to any callback the app registered in its tree. A system
// event (sys_events.go) is the app telling the platform to do something it
// owns — show a toast, open a URL, play a file. A host event is the platform
// telling the app something happened on that side — the audio player's
// position moved, a track finished, a file failed to load — with no button
// tap or input change behind it that a callback ID could name.
//
//	app ──SendSystemEvent("audio", {command: "play"})──▶ host
//	app ◀──ReceiveHostEvent("audio_status", {...})────── host
//
// The channel is generic on purpose. Audio status was its first traffic and
// the app lifecycle (lifecycle.go) its second, but the shape — a name and a
// JSON-ish payload, delivered to whoever asked for that name — is the same
// one every later platform bridge on the roadmap needs (a keystore result,
// a location fix), and
// adding a bridge function per feature would grow the gomobile surface for
// no gain. Each host therefore exposes exactly one entry point
// (mobile.ReportHostEvent on the natives, GrMobWASM.HostEvent in the
// browser) and this file fans the events out.
//
// Dispatch order is fixed: names core owns are consumed here first (the
// audio status feeds core's own status record, which apps read through
// CurrentAudioStatus and hooks.UseAudio; the lifecycle state likewise
// through CurrentLifecycle and hooks.UseLifecycle), and then every app
// subscriber for
// the name runs. That lets an app observe a core event without core having
// to expose its internals, and lets an app define entirely private event
// names for a shell it has extended itself.
//
// # Threading
//
// ReceiveHostEvent runs on the caller's goroutine. On the natives that is
// the bridge call made from the shell's events thread, serialized with
// render passes by render.Manager.Dispatch; in the browser it is the JS
// callback's goroutine. Subscribers may write State and call RequestRender
// freely — both are safe from any goroutine — but must not block.
var (
	hostEventsMu  sync.RWMutex
	hostEventSubs = map[string]map[int]func(map[string]any){}
	hostEventNext int
)

// OnHostEvent subscribes fn to host events named name. The returned function
// cancels the subscription; calling it more than once is harmless.
//
// Subscriptions are process-wide, like the system-event handler and for the
// same reason: the thing on the far side of the channel is one physical
// device with one audio output, one keystore, one location, so there is no
// context tree to scope them to. A component that subscribes during render
// must therefore guard against subscribing again on the next pass —
// hooks.UseAudio shows the pattern (a hook slot remembers that it did).
func OnHostEvent(name string, fn func(data map[string]any)) (cancel func()) {
	hostEventsMu.Lock()
	defer hostEventsMu.Unlock()
	id := hostEventNext
	hostEventNext++
	subs := hostEventSubs[name]
	if subs == nil {
		subs = map[int]func(map[string]any){}
		hostEventSubs[name] = subs
	}
	subs[id] = fn
	return func() {
		hostEventsMu.Lock()
		defer hostEventsMu.Unlock()
		if subs := hostEventSubs[name]; subs != nil {
			delete(subs, id)
			if len(subs) == 0 {
				delete(hostEventSubs, name)
			}
		}
	}
}

// ReceiveHostEvent delivers one event from the host. Names core owns are
// consumed first; then every subscriber for the name runs, outside the
// registry lock so a subscriber may subscribe or cancel from inside its own
// handler without deadlocking.
//
// An event nobody consumes is logged rather than dropped silently — unlike
// an unknown system event, which a host drops because a newer app may
// legitimately send what an older shell does not understand, an unknown
// host event means the shell is sending traffic the app never asked for,
// which is worth a line in the log during development.
func ReceiveHostEvent(name string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	consumed := false
	switch name {
	case hostEventAudioStatus:
		receiveAudioStatus(data)
		consumed = true
	case hostEventLifecycle:
		receiveLifecycle(data)
		consumed = true
	}

	hostEventsMu.RLock()
	subs := hostEventSubs[name]
	fns := make([]func(map[string]any), 0, len(subs))
	for _, fn := range subs {
		fns = append(fns, fn)
	}
	hostEventsMu.RUnlock()

	for _, fn := range fns {
		fn(data)
	}
	if !consumed && len(fns) == 0 {
		log.Printf("grmob: host event %q has no consumer", name)
	}
}

// hostEventAudioStatus is the first of the host events core consumes itself;
// see audio.go for the payload, and lifecycle.go for the other one.
const hostEventAudioStatus = "audio_status"

// numberProp reads a JSON number out of a decoded payload. JSON has one
// number type, so a decoded map carries float64 — but a payload built in Go
// (a test, an embedder) naturally writes int, and both should work.
func numberProp(data map[string]any, key string) (float64, bool) {
	switch v := data[key].(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}
