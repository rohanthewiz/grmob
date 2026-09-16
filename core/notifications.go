package core

import (
	"strconv"
	"sync"
	"time"
)

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

// # Scheduled notifications
//
// LocalNotification.At hands the OS a time instead of asking for the banner
// now, and that is the one way around the paragraph above: the OS holds the
// request, so it is drawn at At whether the app is on screen, suspended or
// closed.
//
//	              wire: "at" = Unix milliseconds, absent for "now"
//	Android   AlarmManager → NotificationAlarmReceiver, which posts it.
//	          Exact (setExactAndAllowWhileIdle) where the app may schedule
//	          exact alarms — SCHEDULE_EXACT_ALARM, which Android 14+ does not
//	          grant new installs by default — and otherwise inexact
//	          (setAndAllowWhileIdle), which Doze may defer by minutes.
//	          The shell keeps a list of what it scheduled and re-arms it
//	          after a reboot or an app update (a boot receiver) and at the
//	          next launch after a force stop; one whose time passed while it
//	          could not fire is posted then, late rather than never.
//	iOS       UNCalendarNotificationTrigger on the local wall clock; iOS
//	          keeps at most 64 pending requests per app.
//	Browser   a timer in the page, so only while the tab is open — a page
//	          cannot ask the browser to post for it later.
//	Headless  nothing.
//
// A time that has already passed posts now, as a zero At does. Cancel takes a
// scheduled notification down before it fires, and posting again under the
// same ID replaces the pending request with the new time.

// LocalNotification is one banner to post. ID is required; Title and Body may
// each be empty but not both.
type LocalNotification struct {
	// ID identifies the notification for replacement, cancellation and taps.
	// Posting a second notification with the same ID replaces the first,
	// scheduled or shown.
	ID    string
	Title string
	Body  string

	// At schedules the notification for a moment instead of posting it now.
	// The zero value, or any time not after the moment of posting, posts
	// immediately. See "Scheduled notifications" above for what each host
	// promises.
	At time.Time
}

// The two event names and the two commands, which are the whole wire
// contract. Held to the shells by mobile/verify's
// TestNotificationEventSpellingsAgree.
const (
	systemEventNotification  = "notification"
	hostEventNotificationTap = "notification_tap"
	notificationPost         = "post"
	notificationCancel       = "cancel"
	notificationAt           = "at"

	notificationSweep          = "sweep"
	hostEventNotificationSwept = "notification_swept"
)

// PostNotification asks the host to show n — now, or at n.At — replacing any
// notification already showing or scheduled under the same ID. Dropped without an ID or without any text; see
// the file comment for why the ID is required and for permissions.
func PostNotification(n LocalNotification) {
	if n.ID == "" || (n.Title == "" && n.Body == "") {
		return
	}
	payload := map[string]any{
		"command": notificationPost,
		"id":      n.ID,
		"title":   n.Title,
		"body":    n.Body,
	}
	// Milliseconds since the epoch, the one time representation all three
	// hosts construct a date from without a parser (Date(ms), Date(timeInterval
	// SinceReferenceDate…), System.currentTimeMillis). A past time is sent as
	// "now" by leaving the key out, so no host has to decide what a stale
	// schedule means.
	if !n.At.IsZero() && n.At.After(notificationNow()) {
		payload[notificationAt] = n.At.UnixMilli()
	}
	SendSystemEvent(systemEventNotification, payload)
}

// notificationNow is time.Now, replaceable by tests that need a fixed instant.
var notificationNow = time.Now

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

// # Sweeping by prefix
//
// CancelNotification needs the id, and an id is only as durable as the memory
// holding it. hooks.UseAlarms names every notification it schedules
// "grmob.alarm.<alarm id>.<unix seconds>" and keeps that list in memory, so a
// process that dies with notifications pending leaves them in the OS with
// nobody left who can name them: an alarm switched off after a relaunch would
// still ring. A sweep asks the host instead, because the host is the one
// thing that still knows.
//
//	app ──SendSystemEvent("notification", {command: "sweep", prefix, request})──▶ host
//	app ◀──ReceiveHostEvent("notification_swept", {request, fired: [ids]})────── host
//
// The host cancels every notification it scheduled or shows under the prefix
// and answers with the ids among the scheduled ones whose time has come — the
// ones the OS has drawn, or will draw late — which is how a relaunched app
// learns what rang while it was not running. `request` is a correlation id in
// ReadClipboard's mould (see clipboard.go for why one is needed).
//
// Each host answers from a record of its own, because no platform lists what
// it has already delivered in a form that survives the user clearing it:
//
//	Android   the scheduled-post store Notifications.kt already keeps for
//	          re-arming; a fired entry is marked rather than removed, so the
//	          sweep can report it
//	iOS       a UserDefaults map of id → time, written when a post is scheduled
//	Browser   the page's own timers; a closed tab took them with it, so a
//	          fresh page has nothing to report, and nothing to cancel either
//	Headless  nothing: fn runs at once with no ids
//
// A host keeps fired entries for a week (the horizon UseAlarms schedules
// within) and prunes them after that, so an app that never sweeps does not
// grow the record without bound.

var (
	sweepMu      sync.Mutex
	sweepPending = map[string]func(fired []string){}
	sweepNext    uint64
)

// SweepNotifications cancels every notification posted or scheduled under an
// ID beginning with prefix, and calls fn once with the IDs among the scheduled
// ones whose time had arrived (see "Sweeping by prefix"). fn may be nil.
//
// An empty prefix is refused (fn runs with no ids and nothing is sent):
// sweeping everything would take down notifications this caller never
// posted. With no host registered fn runs at once on the caller's goroutine,
// as ReadClipboard's does; otherwise on the goroutine that delivers the reply.
func SweepNotifications(prefix string, fn func(fired []string)) {
	if fn == nil {
		fn = func([]string) {}
	}
	if prefix == "" || !HasSystemEventHandler() {
		fn(nil)
		return
	}
	sweepMu.Lock()
	sweepNext++
	request := strconv.FormatUint(sweepNext, 10)
	// Registered before sending: a native host may answer inside the call.
	sweepPending[request] = fn
	sweepMu.Unlock()

	SendSystemEvent(systemEventNotification, map[string]any{
		"command": notificationSweep,
		"prefix":  prefix,
		"request": request,
	})
}

// receiveNotificationSwept decodes the reply to one sweep. The payload every
// host writes:
//
//	request  string    the request id from the sweep, echoed
//	fired    []string  ids whose scheduled time had arrived; absent means none
//
// An unknown request is dropped, as a duplicate clipboard reply is. Ids that
// are not strings are skipped rather than failing the whole reply.
func receiveNotificationSwept(data map[string]any) {
	request, _ := data["request"].(string)
	if request == "" {
		return
	}
	sweepMu.Lock()
	fn := sweepPending[request]
	delete(sweepPending, request)
	sweepMu.Unlock()
	if fn == nil {
		return
	}
	var fired []string
	// A decoded JSON array is []any; a payload built in Go may be []string.
	switch list := data["fired"].(type) {
	case []any:
		for _, v := range list {
			if id, ok := v.(string); ok && id != "" {
				fired = append(fired, id)
			}
		}
	case []string:
		for _, id := range list {
			if id != "" {
				fired = append(fired, id)
			}
		}
	}
	fn(fired)
}

// resetSweepsForTest drops every pending sweep, for the reason
// resetClipboardForTest exists.
func resetSweepsForTest() {
	sweepMu.Lock()
	defer sweepMu.Unlock()
	sweepPending = map[string]func([]string){}
}
