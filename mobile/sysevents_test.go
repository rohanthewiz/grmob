package mobile_test

import (
	"encoding/json"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/mobile"
)

// recordingListener is the shape a native shell implements: one method, two
// strings. Events are appended rather than replaced so a test can assert on
// ordering as well as content.
type recordingListener struct {
	events []struct{ Name, Payload string }
}

func (r *recordingListener) OnSystemEvent(name string, payload string) {
	r.events = append(r.events, struct{ Name, Payload string }{name, payload})
}

// attach installs l and guarantees detachment: core's handler is
// package-level, so a listener left registered would receive events from
// every later test in this binary.
func attach(t *testing.T, l mobile.SystemEventListener) {
	t.Helper()
	mobile.SetSystemEventListener(l)
	t.Cleanup(func() { mobile.SetSystemEventListener(nil) })
}

// The bound path end to end: a Go-side OpenURL reaches the shell's listener
// with the event name and a JSON payload it can parse.
func TestSystemEventListenerReceivesOpenURLAsJSON(t *testing.T) {
	rec := &recordingListener{}
	attach(t, rec)

	core.OpenURL("https://example.org/give")

	if len(rec.events) != 1 {
		t.Fatalf("listener saw %d events, want 1", len(rec.events))
	}
	if rec.events[0].Name != "open_url" {
		t.Errorf("name = %q, want %q", rec.events[0].Name, "open_url")
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(rec.events[0].Payload), &payload); err != nil {
		t.Fatalf("payload is not valid JSON (%q): %v", rec.events[0].Payload, err)
	}
	if payload.URL != "https://example.org/give" {
		t.Errorf("payload.url = %q, want the URL passed in", payload.URL)
	}
}

// Toasts travel the same channel — this is the event the natives were
// dropping outright before a listener existed, so it is worth pinning that
// they now arrive with the duration the config carries.
func TestSystemEventListenerReceivesToasts(t *testing.T) {
	rec := &recordingListener{}
	attach(t, rec)

	core.ShowToast("Gift received", core.Duration(3500))

	if len(rec.events) != 1 {
		t.Fatalf("listener saw %d events, want 1", len(rec.events))
	}
	if rec.events[0].Name != "toast" {
		t.Errorf("name = %q, want %q", rec.events[0].Name, "toast")
	}
	var payload struct {
		Message  string `json:"message"`
		Duration int    `json:"duration"`
	}
	if err := json.Unmarshal([]byte(rec.events[0].Payload), &payload); err != nil {
		t.Fatalf("payload is not valid JSON (%q): %v", rec.events[0].Payload, err)
	}
	if payload.Message != "Gift received" || payload.Duration != 3500 {
		t.Errorf("payload = %+v, want {Gift received 3500}", payload)
	}
}

// Passing nil detaches. A shell tearing down must be able to stop receiving
// without the events starting to panic on a nil interface value.
func TestSetSystemEventListenerNilDetaches(t *testing.T) {
	rec := &recordingListener{}
	mobile.SetSystemEventListener(rec)
	mobile.SetSystemEventListener(nil)
	t.Cleanup(func() { mobile.SetSystemEventListener(nil) })

	core.OpenURL("https://example.org")

	if len(rec.events) != 0 {
		t.Fatalf("detached listener still saw %d events", len(rec.events))
	}
}

// A second registration replaces the first rather than fanning out: one
// process has one screen, and two sinks would mean one tap producing two
// toasts.
func TestSetSystemEventListenerReplacesRatherThanFansOut(t *testing.T) {
	first, second := &recordingListener{}, &recordingListener{}
	mobile.SetSystemEventListener(first)
	attach(t, second)

	core.OpenURL("https://example.org")

	if len(first.events) != 0 {
		t.Errorf("replaced listener still received %d events", len(first.events))
	}
	if len(second.events) != 1 {
		t.Errorf("current listener received %d events, want 1", len(second.events))
	}
}
