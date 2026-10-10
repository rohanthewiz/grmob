package keystore

import (
	"errors"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// fakeHost installs a system-event recorder for the test and removes it
// afterwards, returning the requests it saw. answer, when non-nil, runs inside
// SendSystemEvent — the synchronous-reply case a native host may produce.
func fakeHost(t *testing.T, answer func(req map[string]any)) *[]map[string]any {
	t.Helper()
	var seen []map[string]any
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name != systemEventName {
			return
		}
		seen = append(seen, data)
		if answer != nil {
			answer(data)
		}
	})
	t.Cleanup(func() {
		core.SetSystemEventHandler(nil)
		resetForTest()
	})
	return &seen
}

// memoryHost is a fakeHost that behaves like a working platform store: it
// answers every command from a map, synchronously, over the real host-event
// path. It is what the round-trip tests run against.
func memoryHost(t *testing.T) map[string]string {
	t.Helper()
	store := map[string]string{}
	fakeHost(t, func(req map[string]any) {
		id, _ := req["id"].(string)
		key, _ := req["key"].(string)
		reply := map[string]any{"id": id, "ok": true}
		switch req["command"] {
		case commandSave:
			store[key], _ = req["value"].(string)
		case commandGet:
			if v, ok := store[key]; ok {
				reply["found"] = true
				reply["value"] = v
			} else {
				reply["found"] = false
			}
		case commandDelete:
			delete(store, key)
		}
		core.ReceiveHostEvent(hostEventName, reply)
	})
	return store
}

// Save, Get and Delete against a working store: the value comes back, a
// missing key is found=false with no error, and delete of a missing key is
// not an error either.
func TestRoundTrip(t *testing.T) {
	store := memoryHost(t)

	var saveErr error = errors.New("unset")
	Save("token", "s3cret", func(err error) { saveErr = err })
	if saveErr != nil {
		t.Fatalf("save: %v", saveErr)
	}
	if store["token"] != "s3cret" {
		t.Fatalf("host stored %q, want s3cret", store["token"])
	}

	var got string
	var found bool
	Get("token", func(v string, f bool, err error) {
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		got, found = v, f
	})
	if !found || got != "s3cret" {
		t.Fatalf("get = (%q, %v), want (s3cret, true)", got, found)
	}

	var delErr error = errors.New("unset")
	Delete("token", func(err error) { delErr = err })
	if delErr != nil {
		t.Fatalf("delete: %v", delErr)
	}
	Delete("token", func(err error) { delErr = err })
	if delErr != nil {
		t.Fatalf("second delete: %v", delErr)
	}

	found = true
	Get("token", func(v string, f bool, err error) {
		if err != nil {
			t.Fatalf("get after delete: %v", err)
		}
		got, found = v, f
	})
	if found || got != "" {
		t.Errorf("get after delete = (%q, %v), want (\"\", false)", got, found)
	}
}

// The empty string is a value: it is sent, stored, and read back as found.
func TestEmptyValueIsAValue(t *testing.T) {
	memoryHost(t)
	Save("k", "", nil)
	found := false
	Get("k", func(v string, f bool, err error) { found = f && v == "" && err == nil })
	if !found {
		t.Error("an empty value read back as not found")
	}
}

// The request carries exactly what the host needs: the value only on a save,
// so a get or a delete never moves a secret across the bridge.
func TestRequestShape(t *testing.T) {
	seen := fakeHost(t, nil)
	Save("token", "s3cret", nil)
	Get("token", func(string, bool, error) {})
	Delete("token", nil)
	if len(*seen) != 3 {
		t.Fatalf("got %d requests, want 3", len(*seen))
	}
	for i, want := range []string{commandSave, commandGet, commandDelete} {
		req := (*seen)[i]
		if req["command"] != want || req["key"] != "token" || req["id"] == "" {
			t.Errorf("request %d = %v, want %s of token with an id", i, req, want)
		}
		_, hasValue := req["value"]
		if hasValue != (want == commandSave) {
			t.Errorf("request %d (%s) carries value=%v", i, want, hasValue)
		}
	}
	if (*seen)[0]["value"] != "s3cret" {
		t.Errorf("save carried value %v", (*seen)[0]["value"])
	}
}

// With nobody to ask, every call answers at once with ErrUnavailable rather
// than leaving its caller waiting forever.
func TestNoHostIsUnavailableImmediately(t *testing.T) {
	core.SetSystemEventHandler(nil)
	calls := 0
	Save("k", "v", func(err error) {
		calls++
		if !errors.Is(err, ErrUnavailable) {
			t.Errorf("headless save err = %v", err)
		}
	})
	Get("k", func(v string, found bool, err error) {
		calls++
		if !errors.Is(err, ErrUnavailable) || found || v != "" {
			t.Errorf("headless get = (%q, %v, %v)", v, found, err)
		}
	})
	Delete("k", func(err error) {
		calls++
		if !errors.Is(err, ErrUnavailable) {
			t.Errorf("headless delete err = %v", err)
		}
	})
	if calls != 3 {
		t.Fatalf("%d of 3 headless calls answered", calls)
	}
}

// An empty key is refused in Go, before any host is asked.
func TestEmptyKeyNeverReachesTheHost(t *testing.T) {
	seen := fakeHost(t, nil)
	var err error
	Save("", "v", func(e error) { err = e })
	if !errors.Is(err, ErrEmptyKey) {
		t.Errorf("save with empty key err = %v", err)
	}
	Get("", func(_ string, _ bool, e error) { err = e })
	if !errors.Is(err, ErrEmptyKey) {
		t.Errorf("get with empty key err = %v", err)
	}
	if len(*seen) != 0 {
		t.Errorf("the host saw %d requests for an empty key", len(*seen))
	}
}

// The browser's refusal is ErrUnavailable, so an app can tell "no keystore
// here" from "the keystore failed" — and only the reserved reason maps to it.
func TestReasonUnavailableIsTheSentinel(t *testing.T) {
	fakeHost(t, func(req map[string]any) {
		reply := map[string]any{"id": req["id"], "ok": false, "error": reasonUnavailable}
		if req["key"] == "broken" {
			reply["error"] = "errSecInteractionNotAllowed (-25308)"
		}
		core.ReceiveHostEvent(hostEventName, reply)
	})
	var err error
	Save("token", "v", func(e error) { err = e })
	if err != ErrUnavailable {
		t.Errorf("refusal err = %v, want ErrUnavailable itself", err)
	}
	Save("broken", "s3cret", func(e error) { err = e })
	if err == nil || errors.Is(err, ErrUnavailable) {
		t.Fatalf("host failure err = %v, want a non-sentinel error", err)
	}
	// The message names the call and the host's reason, and never the value.
	msg := err.Error()
	for _, want := range []string{"save", `"broken"`, "-25308"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not mention %s", msg, want)
		}
	}
	if strings.Contains(msg, "s3cret") {
		t.Errorf("error %q leaks the value", msg)
	}
}

// A reply without ok is a failure, so a shell that forgets the key reports an
// error instead of a save that never happened.
func TestMissingOkIsAFailure(t *testing.T) {
	seen := fakeHost(t, nil)
	var err error
	calls := 0
	Save("k", "v", func(e error) { calls++; err = e })
	core.ReceiveHostEvent(hostEventName, map[string]any{"id": (*seen)[0]["id"]})
	if calls != 1 || err == nil {
		t.Errorf("missing ok: calls=%d err=%v, want one call with an error", calls, err)
	}
}

// Two calls in flight get their own answers in whatever order the host
// replies; a duplicate reply and a reply with no id reach no one.
func TestRepliesCorrelateById(t *testing.T) {
	seen := fakeHost(t, nil)
	answers := map[string]string{}
	calls := 0
	Get("a", func(v string, _ bool, _ error) { calls++; answers["a"] = v })
	Get("b", func(v string, _ bool, _ error) { calls++; answers["b"] = v })
	if len(*seen) != 2 {
		t.Fatalf("got %d requests, want 2", len(*seen))
	}
	idA, idB := (*seen)[0]["id"], (*seen)[1]["id"]
	if idA == idB {
		t.Fatalf("both calls carried id %v", idA)
	}

	core.ReceiveHostEvent(hostEventName, map[string]any{"value": "no id", "ok": true, "found": true})
	core.ReceiveHostEvent(hostEventName, map[string]any{"id": idB, "ok": true, "found": true, "value": "B"})
	core.ReceiveHostEvent(hostEventName, map[string]any{"id": idA, "ok": true, "found": true, "value": "A"})
	core.ReceiveHostEvent(hostEventName, map[string]any{"id": idA, "ok": true, "found": true, "value": "again"})

	if answers["a"] != "A" || answers["b"] != "B" || calls != 2 {
		t.Errorf("answers = %v after %d calls, want a=A b=B in 2", answers, calls)
	}
}

// found and value are believed only on a successful get: a save's reply that
// echoes a value, or a failed get that carries one, hands the caller nothing.
func TestStrayValuesAreIgnored(t *testing.T) {
	fakeHost(t, func(req map[string]any) {
		core.ReceiveHostEvent(hostEventName, map[string]any{
			"id": req["id"], "ok": req["command"] != commandGet,
			"found": true, "value": "stray", "error": "locked",
		})
	})
	var got string
	var found bool
	var err error
	Get("k", func(v string, f bool, e error) { got, found, err = v, f, e })
	if err == nil || found || got != "" {
		t.Errorf("failed get = (%q, %v, %v), want (\"\", false, error)", got, found, err)
	}
	Save("k", "v", func(e error) { err = e })
	if err != nil {
		t.Errorf("save with stray keys err = %v", err)
	}
}

// A nil done is allowed for the writes and makes Get a no-op — a read whose
// answer nobody receives is not sent at all.
func TestNilCallbacks(t *testing.T) {
	seen := fakeHost(t, func(req map[string]any) {
		core.ReceiveHostEvent(hostEventName, map[string]any{"id": req["id"], "ok": false, "error": "boom"})
	})
	Save("k", "v", nil)
	Delete("k", nil)
	Get("k", nil)
	if len(*seen) != 2 {
		t.Errorf("host saw %d requests, want 2 (save and delete; get with nil done is not sent)", len(*seen))
	}
}
