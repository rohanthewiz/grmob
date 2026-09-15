package core

import "testing"

// clipboardHost installs a system-event recorder for the test and removes it
// afterwards, returning the events it saw.
func clipboardHost(t *testing.T, answer func(data map[string]any)) *[]map[string]any {
	t.Helper()
	var seen []map[string]any
	SetSystemEventHandler(func(name string, data map[string]any) {
		if name != systemEventClipboard {
			return
		}
		seen = append(seen, data)
		if answer != nil {
			answer(data)
		}
	})
	t.Cleanup(func() {
		SetSystemEventHandler(nil)
		resetClipboardForTest()
	})
	return &seen
}

// A write is one event carrying the text, and an empty string is still sent:
// clearing the clipboard is a real request.
func TestWriteClipboardSendsTheText(t *testing.T) {
	seen := clipboardHost(t, nil)
	WriteClipboard("hello")
	WriteClipboard("")
	if len(*seen) != 2 {
		t.Fatalf("got %d events, want 2", len(*seen))
	}
	for i, want := range []string{"hello", ""} {
		ev := (*seen)[i]
		if ev["command"] != clipboardWrite || ev["text"] != want {
			t.Errorf("event %d = %v, want write %q", i, ev, want)
		}
	}
}

// With nobody to ask, a read answers at once with a refusal rather than
// leaving its caller waiting forever.
func TestReadClipboardWithNoHostRefusesImmediately(t *testing.T) {
	SetSystemEventHandler(nil)
	called := false
	ReadClipboard(func(text string, ok bool) {
		called = true
		if ok || text != "" {
			t.Errorf("headless read = (%q, %v), want (\"\", false)", text, ok)
		}
	})
	if !called {
		t.Fatal("headless read never called back")
	}
}

// A host that answers synchronously, inside SendSystemEvent, still reaches the
// callback: it is registered before the event goes out.
func TestReadClipboardSynchronousReply(t *testing.T) {
	clipboardHost(t, func(data map[string]any) {
		if data["command"] == clipboardRead {
			ReceiveHostEvent(hostEventClipboard, map[string]any{
				"id": data["id"], "text": "pasted", "ok": true,
			})
		}
	})
	var got string
	ReadClipboard(func(text string, ok bool) {
		if ok {
			got = text
		}
	})
	if got != "pasted" {
		t.Errorf("read = %q, want pasted", got)
	}
}

// Two reads in flight get their own answers, in whatever order the host
// replies, and each callback runs once.
func TestReadClipboardCorrelatesById(t *testing.T) {
	seen := clipboardHost(t, nil)
	answers := map[string]string{}
	calls := 0
	ReadClipboard(func(text string, ok bool) { calls++; answers["first"] = text })
	ReadClipboard(func(text string, ok bool) { calls++; answers["second"] = text })
	if len(*seen) != 2 {
		t.Fatalf("got %d read events, want 2", len(*seen))
	}
	firstID := (*seen)[0]["id"]
	secondID := (*seen)[1]["id"]
	if firstID == secondID {
		t.Fatalf("both reads carried id %v", firstID)
	}

	// Reply out of order, then repeat a reply: the duplicate must be dropped.
	ReceiveHostEvent(hostEventClipboard, map[string]any{"id": secondID, "text": "B", "ok": true})
	ReceiveHostEvent(hostEventClipboard, map[string]any{"id": firstID, "text": "A", "ok": true})
	ReceiveHostEvent(hostEventClipboard, map[string]any{"id": firstID, "text": "again", "ok": true})

	if answers["first"] != "A" || answers["second"] != "B" || calls != 2 {
		t.Errorf("answers = %v after %d calls, want first=A second=B in 2", answers, calls)
	}
}

// A reply without ok is a refusal, so a shell that forgets the key pastes
// nothing instead of pasting "" as if the clipboard were empty. A reply with
// no id matches nothing.
func TestReadClipboardMissingOkIsARefusal(t *testing.T) {
	seen := clipboardHost(t, nil)
	var gotOK = true
	calls := 0
	ReadClipboard(func(_ string, ok bool) { calls++; gotOK = ok })

	ReceiveHostEvent(hostEventClipboard, map[string]any{"text": "no id"})
	if calls != 0 {
		t.Fatalf("a reply with no id reached a callback")
	}
	ReceiveHostEvent(hostEventClipboard, map[string]any{"id": (*seen)[0]["id"], "text": "x"})
	if calls != 1 || gotOK {
		t.Errorf("missing ok: calls=%d ok=%v, want 1 call with ok=false", calls, gotOK)
	}
}
