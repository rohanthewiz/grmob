package core

import "testing"

func notificationHost(t *testing.T) *[]map[string]any {
	t.Helper()
	var seen []map[string]any
	SetSystemEventHandler(func(name string, data map[string]any) {
		if name == systemEventNotification {
			seen = append(seen, data)
		}
	})
	t.Cleanup(func() { SetSystemEventHandler(nil) })
	return &seen
}

// A post carries every field; a cancel carries the id.
func TestPostAndCancelNotificationSendTheWireShape(t *testing.T) {
	seen := notificationHost(t)
	PostNotification(LocalNotification{ID: "w1:p3", Title: "claude is blocked", Body: "w1:p3"})
	CancelNotification("w1:p3")

	if len(*seen) != 2 {
		t.Fatalf("got %d events, want 2", len(*seen))
	}
	post := (*seen)[0]
	if post["command"] != notificationPost || post["id"] != "w1:p3" ||
		post["title"] != "claude is blocked" || post["body"] != "w1:p3" {
		t.Errorf("post = %v", post)
	}
	cancel := (*seen)[1]
	if cancel["command"] != notificationCancel || cancel["id"] != "w1:p3" {
		t.Errorf("cancel = %v", cancel)
	}
}

// Without an id, or with no text at all, nothing is sent: a notification
// nobody can replace, cancel or route a tap from is never what an app meant.
func TestPostNotificationDropsWhatCannotBeAddressed(t *testing.T) {
	seen := notificationHost(t)
	PostNotification(LocalNotification{Title: "no id"})
	PostNotification(LocalNotification{ID: "x"})
	CancelNotification("")
	if len(*seen) != 0 {
		t.Errorf("sent %v, want nothing", *seen)
	}

	// A title alone or a body alone is enough.
	PostNotification(LocalNotification{ID: "a", Title: "t"})
	PostNotification(LocalNotification{ID: "b", Body: "b"})
	if len(*seen) != 2 {
		t.Errorf("sent %d events for two addressable posts", len(*seen))
	}
}

// A tap reaches the subscriber with its id; a tap without one does not.
func TestOnNotificationTapDeliversTheID(t *testing.T) {
	var got []string
	cancel := OnNotificationTap(func(id string) { got = append(got, id) })
	defer cancel()

	ReceiveHostEvent(hostEventNotificationTap, map[string]any{"id": "w1:p3"})
	ReceiveHostEvent(hostEventNotificationTap, map[string]any{"id": ""})
	ReceiveHostEvent(hostEventNotificationTap, map[string]any{"id": 7})
	if len(got) != 1 || got[0] != "w1:p3" {
		t.Errorf("taps = %v, want [w1:p3]", got)
	}

	cancel()
	ReceiveHostEvent(hostEventNotificationTap, map[string]any{"id": "late"})
	if len(got) != 1 {
		t.Errorf("a cancelled subscriber heard %v", got)
	}
}
