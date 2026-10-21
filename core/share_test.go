package core

import "testing"

// shareHost installs a system-event recorder for the test and removes it
// afterwards, returning the share events it saw. answer, when set, replies
// from inside SendSystemEvent, the way a synchronous host would.
func shareHost(t *testing.T, answer func(data map[string]any)) *[]map[string]any {
	t.Helper()
	var seen []map[string]any
	SetSystemEventHandler(func(name string, data map[string]any) {
		if name != systemEventShare {
			return
		}
		seen = append(seen, data)
		if answer != nil {
			answer(data)
		}
	})
	t.Cleanup(func() {
		SetSystemEventHandler(nil)
		resetShareForTest()
	})
	return &seen
}

// A share is one event carrying the id, the text and the url, and the reply
// with that id reaches done, even when the host answers synchronously.
func TestShareSendsTextAndURLAndHearsTheReply(t *testing.T) {
	seen := shareHost(t, func(data map[string]any) {
		ReceiveHostEvent(hostEventShare, map[string]any{"id": data["id"], "ok": true})
	})
	var got []bool
	Share("A sunset", "https://example.com/p/1", func(ok bool) { got = append(got, ok) })
	if len(*seen) != 1 {
		t.Fatalf("got %d events, want 1", len(*seen))
	}
	ev := (*seen)[0]
	if ev["text"] != "A sunset" || ev["url"] != "https://example.com/p/1" || ev["id"] == "" {
		t.Errorf("event = %v", ev)
	}
	if len(got) != 1 || !got[0] {
		t.Errorf("done = %v, want one true", got)
	}
}

// With nobody to ask, or nothing to share, done answers false at once and no
// event goes out: the caller's fallback runs instead of a wait forever.
func TestShareRefusesAtOnceWithNoHostOrNothingToShare(t *testing.T) {
	SetSystemEventHandler(nil)
	calls, gotOK := 0, true
	Share("text", "", func(ok bool) { calls++; gotOK = ok })
	if calls != 1 || gotOK {
		t.Errorf("headless share: calls=%d ok=%v, want one false", calls, gotOK)
	}

	seen := shareHost(t, nil)
	calls, gotOK = 0, true
	Share("", "", func(ok bool) { calls++; gotOK = ok })
	if calls != 1 || gotOK || len(*seen) != 0 {
		t.Errorf("empty share: calls=%d ok=%v events=%d, want one false and none sent",
			calls, gotOK, len(*seen))
	}
	// A nil done is allowed, for a caller with no fallback.
	Share("text", "", nil)
	if len(*seen) != 1 {
		t.Errorf("a share with nil done was not sent")
	}
}

// Two shares in flight get their own answers in whatever order the host
// replies; a duplicate reply, one with no id and one with no ok are handled
// as clipboard.go's are (dropped, dropped, false).
func TestShareCorrelatesById(t *testing.T) {
	seen := shareHost(t, nil)
	answers := map[string]bool{}
	calls := 0
	Share("one", "", func(ok bool) { calls++; answers["first"] = ok })
	Share("two", "", func(ok bool) { calls++; answers["second"] = ok })
	firstID, secondID := (*seen)[0]["id"], (*seen)[1]["id"]
	if firstID == secondID {
		t.Fatalf("both shares carried id %v", firstID)
	}

	ReceiveHostEvent(hostEventShare, map[string]any{"ok": true})
	ReceiveHostEvent(hostEventShare, map[string]any{"id": secondID, "ok": true})
	ReceiveHostEvent(hostEventShare, map[string]any{"id": firstID})
	ReceiveHostEvent(hostEventShare, map[string]any{"id": firstID, "ok": true})

	if calls != 2 || answers["first"] || !answers["second"] {
		t.Errorf("answers = %v after %d calls, want first=false second=true in 2", answers, calls)
	}
}
