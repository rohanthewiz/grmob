package core

// Local notifications: a banner the OS draws outside the app, posted by the
// app itself rather than pushed from a server.
//
//	app ──SendSystemEvent("notification", {command: "post", id, title, body})──▶ host
//	app ──SendSystemEvent("notification", {command: "cancel", id})───────────▶ host
//	app ◀──ReceiveHostEvent("notification_tap", {id})───────────────────────── host
//
// Posting and cancelling are system events in OpenURL's mould: one way,
// fire-and-forget, no Context. The tap is a host event with a typed wrapper,
// in OnDeepLink's mould — core keeps no state for it, it forwards the id to
// whoever asked.
//
// # Why the id is required
//
// Every platform keys a notification by an identifier the app chooses, and all
// three things an app does after posting need it: posting again under the same
// id replaces the banner in place (an agent that blocks twice is one banner,
// not two), cancel names what to take down, and the tap says which banner the
// user touched. A notification without one could be posted and never touched
// again, which is never what an app wants, so PostNotification drops it rather
// than inventing an id nobody holds.
//
// # Permission is the app's to ask for, not this file's
//
// Android 13+, iOS and every browser refuse to show a notification the user
// has not allowed, and they refuse silently — the post is simply not drawn.
// Asking belongs to the permission package (permission.Notifications), which
// core cannot import (permission imports core), and asking at the moment of
// posting would be the wrong moment anyway: a prompt that appears because an
// agent blocked while the user was looking elsewhere is a prompt that gets
// denied. An app asks from a screen, with a reason on it, and posts later.
//
//	Android   NotificationCompat on a "grmob" channel; POST_NOTIFICATIONS on 13+
//	iOS       UNUserNotificationCenter, shown in the foreground too
//	Browser   new Notification(title, {body, tag: id})
//	Headless  nothing (no handler registered — see SendSystemEvent)
//
// # What a tap means on each host
//
// Tapping brings the app forward on every platform — that part needs no code
// — and then reports the id. The report arrives after the shell has started
// the runtime (Android reports from MainActivity after start(); iOS's
// notification delegate is set at launch, before the first tap can be
// delivered), so a subscriber registered during the first render hears a tap
// that cold-launched the app.
//
// # Backgrounded apps
//
// A post only happens while the Go side is running *and can hear about
// whatever it is posting about*, and both natives take that away within
// seconds of the app leaving the screen:
//
//	iOS       the process is suspended shortly after it leaves the foreground
//	Android   the process keeps running, but the platform's background
//	          firewall cuts the app's network after a short grace period —
//	          measured at about 5 s on an API 36 emulator (dumpsys netpolicy:
//	          effective=APP_BACKGROUND). A socket-driven app hears the first
//	          event in that window and nothing after it, cancels included,
//	          until it is back on screen.
//	Browser   a hidden tab keeps running and keeps its network
//
// So this is a way to tell the user something the app learned while it could,
// not a replacement for server push. An app that must alert from the
// background needs push (APNs/FCM) or, on Android, a foreground service.

// LocalNotification is one banner to post. ID is required; Title and Body may
// each be empty but not both.
type LocalNotification struct {
	// ID identifies the notification for replacement, cancellation and taps.
	// Posting a second notification with the same ID replaces the first.
	ID    string
	Title string
	Body  string
}

// The two event names and the two commands, which are the whole wire
// contract. Held to the shells by mobile/verify's
// TestNotificationEventSpellingsAgree.
const (
	systemEventNotification  = "notification"
	hostEventNotificationTap = "notification_tap"
	notificationPost         = "post"
	notificationCancel       = "cancel"
)

// PostNotification asks the host to show n, replacing any notification already
// showing under the same ID. Dropped without an ID or without any text; see
// the file comment for why the ID is required and for permissions.
func PostNotification(n LocalNotification) {
	if n.ID == "" || (n.Title == "" && n.Body == "") {
		return
	}
	SendSystemEvent(systemEventNotification, map[string]any{
		"command": notificationPost,
		"id":      n.ID,
		"title":   n.Title,
		"body":    n.Body,
	})
}

// CancelNotification takes down the notification posted under id, whether it
// is still on screen or already in the notification list. Cancelling one that
// is not there is harmless on every host.
func CancelNotification(id string) {
	if id == "" {
		return
	}
	SendSystemEvent(systemEventNotification, map[string]any{
		"command": notificationCancel,
		"id":      id,
	})
}

// OnNotificationTap subscribes fn to taps on the app's notifications; fn
// receives the ID the tapped notification was posted under. The returned
// function cancels the subscription.
//
// Like OnDeepLink, a typed wrapper over OnHostEvent and nothing more: core
// keeps no record of taps, because a tap is an instruction ("show me this")
// rather than a state anyone reads later. fn runs on the goroutine that
// delivered the host event and must not block. An empty or absent id is
// dropped — a subscriber cannot route a tap it cannot identify.
func OnNotificationTap(fn func(id string)) (cancel func()) {
	return OnHostEvent(hostEventNotificationTap, func(data map[string]any) {
		id, _ := data["id"].(string)
		if id == "" {
			return
		}
		fn(id)
	})
}
