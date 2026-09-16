package hooks

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rohanthewiz/grmob/alarm"
	"github.com/rohanthewiz/grmob/core"
)

// AlarmOptions configures UseAlarms. Every field has a usable zero value.
type AlarmOptions struct {
	// OnRing is called once when an alarm starts ringing, including a snoozed
	// one coming back. With Notify, an alarm the OS rang while the app was in
	// the background is reported here when the app returns to the foreground,
	// and one it rang while the app was closed shortly after the app next
	// starts (see "Off screen" on UseAlarms). It runs on the alarm goroutine
	// or the host-event goroutine, never in a render: write
	// state with State.Set (which is goroutine-safe) rather than touching
	// anything a render owns. A one-time alarm (alarm.Alarm.Once) is the
	// app's to switch off here — the hook reports rings and never edits the
	// app's list.
	OnRing func(alarm.Alarm)

	// Sound is played through core's audio player while an alarm rings, and
	// restarted each time it reaches its end. An empty URL rings silently.
	//
	// core has one player (see core.AudioLoad), so a ringing alarm replaces
	// whatever the app was playing; it is not resumed afterwards.
	Sound core.AudioTrack

	// Haptics pulses the device every other second while an alarm rings.
	Haptics bool

	// Snooze is how long Snooze silences an alarm for; 0 means 9 minutes, the
	// length alarm clocks settled on when snooze was a mechanical cam.
	Snooze time.Duration

	// RingFor is how long an alarm rings unanswered before it stops by itself;
	// 0 means 10 minutes. An alarm that stops this way is dismissed, not
	// snoozed.
	RingFor time.Duration

	// Notify hands alarms to the OS while the app is off screen, so they ring
	// as notifications with the app suspended or closed. See "Off screen"
	// on UseAlarms. The app asks for permission.Notifications itself; without
	// it the OS drops the notifications silently.
	Notify bool

	// NotifyText writes a scheduled notification's title and body. Nil uses
	// the alarm's Label (or "Alarm") and its 12-hour time.
	NotifyText func(a alarm.Alarm) (title, body string)
}

// notifyText resolves NotifyText.
func (o AlarmOptions) notifyText(a alarm.Alarm) (title, body string) {
	if o.NotifyText != nil {
		return o.NotifyText(a)
	}
	title = a.Label
	if title == "" {
		title = "Alarm"
	}
	return title, a.TimeLabel(false)
}

// Limits on what Notify schedules at once. iOS keeps at most 64 pending
// notification requests per app and silently drops the rest, so the total
// stays under that with room for the app's own. A week ahead covers every
// repeat pattern alarm.Alarm can express at least once, and an app that
// stays closed longer than that is rescheduled the next time it runs.
const (
	alarmNotifyMax     = 60
	alarmNotifyHorizon = 7 * 24 * time.Hour
)

// AlarmRinger is what UseAlarms returns: the ringing alarm, if any, and the two
// things a person can do about it.
type AlarmRinger struct {
	rec *alarmsRecord
	ctx *core.Context
}

// Ringing returns the alarm that is ringing now, and whether one is.
func (r AlarmRinger) Ringing() (alarm.Alarm, bool) {
	r.rec.mu.Lock()
	defer r.rec.mu.Unlock()
	if r.rec.ringing == nil {
		return alarm.Alarm{}, false
	}
	return *r.rec.ringing, true
}

// Snoozed returns the soonest snoozed alarm and when it will ring again, and
// whether there is one — for a "Snoozed until 6:39" line.
func (r AlarmRinger) Snoozed() (alarm.Alarm, time.Time, bool) {
	r.rec.mu.Lock()
	defer r.rec.mu.Unlock()
	if len(r.rec.snoozes) == 0 {
		return alarm.Alarm{}, time.Time{}, false
	}
	first := r.rec.snoozes[0]
	for _, s := range r.rec.snoozes[1:] {
		if s.at.Before(first.at) {
			first = s
		}
	}
	return first.alarm, first.at, true
}

// Snooze silences the ringing alarm and rings it again after the snooze
// length. Nothing happens if no alarm is ringing.
func (r AlarmRinger) Snooze() {
	r.rec.mu.Lock()
	if r.rec.ringing != nil {
		r.rec.snoozes = append(r.rec.snoozes, snoozedAlarm{alarm: *r.rec.ringing, at: time.Now().Add(r.rec.opts.snooze())})
	}
	r.rec.silence()
	r.rec.mu.Unlock()
	r.ctx.RequestRender()
}

// Dismiss silences the ringing alarm until its next scheduled time. Nothing
// happens if no alarm is ringing.
func (r AlarmRinger) Dismiss() {
	r.rec.mu.Lock()
	r.rec.silence()
	r.rec.mu.Unlock()
	r.ctx.RequestRender()
}

// snoozedAlarm is one snoozed ring waiting to come back.
type snoozedAlarm struct {
	alarm alarm.Alarm
	at    time.Time
}

// alarmsRecord is the per-slot state of one UseAlarms. It lives in the
// hook-slot array for the reasons intervalRecord documents.
//
//	render pass                    alarm goroutine (1 s, wall-aligned)
//	   │ alarms, opts ──[mu]──▶       │ check(now): due? → ring
//	   │                              │   ringing: pulse, loop sound, time out
//	   │ Ringing() ◀──[mu]── ringing  │ RequestRender on any change
//	handlers: Snooze/Dismiss ──[mu]──▶ ringing = nil, snoozes += …
//
// mu guards every field. The goroutine, every render, and the ringer's
// methods (called from handlers) all read and write it.
type alarmsRecord struct {
	mu      sync.Mutex
	started bool
	alarms  []alarm.Alarm
	opts    AlarmOptions

	// last is the instant the previous check covered up to. Each check asks
	// about (last, now], which is what makes a late tick unable to miss an
	// alarm or ring one twice (see package alarm).
	last time.Time

	ringing   *alarm.Alarm
	ringSince time.Time
	// queue holds alarms that came due while another was ringing. They ring
	// in turn, each after the one before it is answered, rather than being
	// dropped: two alarms set a minute apart are two things to wake for.
	queue   []alarm.Alarm
	snoozes []snoozedAlarm

	// away is true while the host reports the app in the background. With
	// Notify set, the OS rings for the app then, so check neither rings nor
	// queues anything (see "Off screen" on UseAlarms).
	away bool
	// scheduled holds the ids of the notifications handed to the OS on the
	// last move to the background, for cancelling on the way back, and
	// awaySince is when that move happened, for reporting what they rang.
	scheduled []string
	awaySince time.Time

	// sweepOnReturn is set when the hook mounted in the background: the
	// sweep of a previous process's notifications waits for the first return
	// to the foreground, because sweeping while away would cancel the alarms
	// the OS is meant to ring right now (see "Off screen").
	sweepOnReturn bool
}

// alarmNotifyPrefix begins every notification id UseAlarms schedules; see
// notifications for the whole spelling and UseAlarms for the sweep.
const alarmNotifyPrefix = "grmob.alarm."

func (o AlarmOptions) snooze() time.Duration {
	if o.Snooze > 0 {
		return o.Snooze
	}
	return 9 * time.Minute
}

func (o AlarmOptions) ringFor() time.Duration {
	if o.RingFor > 0 {
		return o.RingFor
	}
	return 10 * time.Minute
}

// UseAlarms checks alarms every second while the app runs, and rings the ones
// that come due:
//
//	alarms := core.NewState(ctx, []alarm.Alarm{...})
//	ringer := hooks.UseAlarms(ctx, alarms.Get(), hooks.AlarmOptions{
//	    Sound: core.AudioTrack{URL: "alarm.mp3", Title: "Alarm"},
//	    Haptics: true,
//	})
//	if a, ok := ringer.Ringing(); ok {
//	    return comps.AlarmRinging{Alarm: a, OnSnooze: ringer.Snooze, OnDismiss: ringer.Dismiss}
//	}
//
// # Off screen
//
// Without Notify this rings while the app is running and in the foreground,
// and not otherwise: iOS suspends a backgrounded app within seconds and
// Android may stop it (see core/notifications.go).
//
// With Notify the OS takes over while the app is away:
//
//	host reports     the hook
//	───────────────  ─────────────────────────────────────────────────────
//	background       schedules a core.LocalNotification{At} for every
//	                 enabled alarm's occurrences in the next week, and for
//	                 each pending snooze; the in-app ringer stands down
//	active           cancels them all, and skips whatever came due while
//	                 away — the OS already rang it, and ringing it again on
//	                 return would wake the user for an alarm they answered
//
// Standing down while away matters on Android, where the process (and this
// goroutine) keeps running in the background: without it the alarm would
// ring twice at once, as a notification and as a sound from a hidden app.
//
// A notification is one banner with the platform's notification sound, not
// a ringing screen with Snooze: tapping it opens the app, which is not
// ringing. OnRing hears about each alarm the OS rang when the app returns,
// so a one-time alarm is switched off the same way as one rung in the app.
//
// An app that was closed rather than backgrounded lost that record with its
// process, so the hook asks the host instead: when it mounts (or, if it
// mounts in the background, when the app first comes forward) it sweeps every
// "grmob.alarm." notification with core.SweepNotifications. That cancels what
// the dead process left scheduled — an alarm switched off after a relaunch
// must not still ring — and the host's reply names the ones that already
// fired, which reach OnRing for every alarm still in the list, soonest first,
// once each. The reply is asynchronous, so those calls arrive shortly after
// the first render rather than during it. Only one UseAlarms with Notify per
// app: the prefix is the hook's, not the call's, so a second one would sweep
// the first one's notifications. Occurrences are scheduled a week ahead and
// capped at 60 in all (iOS's pending limit is 64); an app away for longer is
// rescheduled the next time it runs and leaves the screen.
//
// # What a check does
//
// Once a second, aligned to the wall clock as UseNow is, the goroutine asks
// every enabled alarm and every snooze whether it fell due since the last
// check. The first due one starts ringing: OnRing is called, the sound starts,
// and a render is requested so Ringing() reports it. While one rings, others
// that come due wait their turn. A ringing alarm times out after RingFor.
//
// A render is requested only when the ringing state changes, so an app with
// alarms set and none ringing costs one goroutine wake a second and no render
// passes.
//
// The alarm list is re-read from each render, so editing, adding or
// disabling an alarm takes effect at the next check. Like the other timer
// hooks, the goroutine stops when the context tree closes, not when the
// component leaves the view.
func UseAlarms(ctx *core.Context, alarms []alarm.Alarm, opts AlarmOptions) AlarmRinger {
	slot := core.NewState(ctx, &alarmsRecord{})
	rec := slot.Get()
	ringer := AlarmRinger{rec: rec, ctx: ctx}

	rec.mu.Lock()
	// A copy, so the app mutating its slice in place (which it should not,
	// but can) cannot race the goroutine reading it.
	rec.alarms = append(rec.alarms[:0:0], alarms...)
	rec.opts = opts
	alreadyRunning := rec.started
	rec.started = true
	if !alreadyRunning {
		// An alarm whose time fell before the app started is not rung late:
		// checks begin from mount.
		rec.last = time.Now()
	}
	rec.mu.Unlock()

	if alreadyRunning {
		return ringer
	}

	// The lifecycle subscription is taken whether or not Notify is set now,
	// because opts is re-read every render and Notify may be switched on
	// later. With it off, awayChanged only records the state.
	rec.mu.Lock()
	rec.away = core.CurrentLifecycle() == core.LifecycleBackground
	rec.sweepOnReturn = rec.away
	sweepNow := !rec.away
	rec.mu.Unlock()
	// Before the lifecycle subscription, so the sweep reaches the host ahead
	// of anything this record schedules: hosts handle commands in order, and
	// a sweep that ran after this record's own posts would cancel them.
	if sweepNow {
		rec.sweepPrevious()
	}
	stopLifecycle := core.OnLifecycle(func(state core.LifecycleState) {
		rec.awayChanged(state == core.LifecycleBackground, time.Now())
	})

	done := make(chan struct{})
	ctx.OnClose(func() {
		close(done)
		stopLifecycle()
		rec.mu.Lock()
		rec.started = false
		rec.silence()
		rec.mu.Unlock()
	})

	go func() {
		timer := time.NewTimer(untilNextBoundary(time.Now(), time.Second))
		defer timer.Stop()
		for {
			select {
			case <-done:
				return
			case <-timer.C:
				now := time.Now()
				if rang, changed := rec.check(now); changed {
					if rang != nil {
						rec.mu.Lock()
						onRing := rec.opts.OnRing
						rec.mu.Unlock()
						if onRing != nil {
							onRing(*rang)
						}
					}
					ctx.RequestRender()
				}
				timer.Reset(untilNextBoundary(now, time.Second))
			}
		}
	}()

	return ringer
}

// check advances the record to now. It returns the alarm that started ringing
// on this check, if one did, and whether anything a render shows changed.
//
// Separated from the goroutine so tests can drive it with chosen instants.
// The side effects that leave the process — sound and haptics — go through
// core, which is a no-op with no host attached.
func (r *alarmsRecord) check(now time.Time) (rang *alarm.Alarm, changed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	from := r.last
	r.last = now

	// Away with Notify on: the OS notifications scheduled on the way out ring
	// for everything due now, so nothing here rings or queues. Snoozes that
	// have come due are dropped for the same reason. `last` still advances,
	// so the return to the foreground has nothing stale to catch up on.
	if r.away && r.opts.Notify {
		kept := r.snoozes[:0]
		for _, s := range r.snoozes {
			if s.at.After(now) {
				kept = append(kept, s)
			}
		}
		r.snoozes = kept
		return nil, false
	}

	// Collect everything that came due in (from, now]: snoozes first, since a
	// snoozed alarm has already been waited for once.
	var due []alarm.Alarm
	kept := r.snoozes[:0]
	for _, s := range r.snoozes {
		if !s.at.After(now) {
			due = append(due, s.alarm)
		} else {
			kept = append(kept, s)
		}
	}
	r.snoozes = kept
	for _, a := range r.alarms {
		if a.Due(from, now) {
			due = append(due, a)
		}
	}
	r.queue = append(r.queue, due...)

	if r.ringing != nil {
		if now.Sub(r.ringSince) >= r.opts.ringFor() {
			r.silence()
			changed = true
		} else {
			r.keepRinging(now)
			return nil, false
		}
	}

	if len(r.queue) == 0 {
		return nil, changed
	}
	next := r.queue[0]
	r.queue = r.queue[1:]
	r.ringing = &next
	r.ringSince = now
	if r.opts.Sound.URL != "" {
		core.AudioLoad(r.opts.Sound, core.AudioAutoplay(true))
	}
	if r.opts.Haptics {
		core.Haptic(core.HapticHeavy)
	}
	return &next, true
}

// keepRinging is the once-a-second upkeep of a ringing alarm: restart the sound
// when it ends (core's player does not loop), and pulse every other second.
// Called with mu held.
func (r *alarmsRecord) keepRinging(now time.Time) {
	if r.opts.Sound.URL != "" && core.CurrentAudioStatus().State == core.AudioEnded {
		core.AudioPlay()
	}
	if r.opts.Haptics && int(now.Sub(r.ringSince).Seconds())%2 == 0 {
		core.Haptic(core.HapticHeavy)
	}
}

// silence stops the ringing alarm, if any. Called with mu held. The queue is
// left alone: an alarm waiting behind this one rings on the next check.
func (r *alarmsRecord) silence() {
	if r.ringing == nil {
		return
	}
	r.ringing = nil
	if r.opts.Sound.URL != "" {
		core.AudioStop()
	}
}

// awayChanged records a move into or out of the background and, with Notify
// set, hands the alarms to the OS or takes them back. See "Off screen" on
// UseAlarms.
//
// The OS calls happen outside mu: core's system events call into the host,
// and nothing about the host should be able to wait on this hook's lock.
func (r *alarmsRecord) awayChanged(away bool, now time.Time) {
	rang := r.setAway(away, now)
	if len(rang) == 0 {
		return
	}
	// opts is written by every render, so it is read under mu like the
	// goroutine reads it.
	r.mu.Lock()
	onRing := r.opts.OnRing
	r.mu.Unlock()
	if onRing == nil {
		return
	}
	for _, a := range rang {
		onRing(a)
	}
}

// setAway is awayChanged without the OnRing calls, which it returns instead:
// the alarms the OS rang while the app was away, in the order they came due.
// Separate so tests can see the list, and so OnRing (app code, which may
// write State) runs with no lock of this hook's held.
func (r *alarmsRecord) setAway(away bool, now time.Time) (rangAway []alarm.Alarm) {
	r.mu.Lock()
	if r.away == away {
		r.mu.Unlock()
		return nil
	}
	r.away = away
	// Read before the early return below: a record that mounted away sweeps
	// on its first return whether or not Notify is on now, because the
	// notifications it is sweeping belong to a process that may have had it on.
	sweep := !away && r.sweepOnReturn
	if sweep {
		r.sweepOnReturn = false
	}
	if !r.opts.Notify && len(r.scheduled) == 0 {
		r.mu.Unlock()
		if sweep {
			r.sweepPrevious()
		}
		return nil
	}
	cancel := r.scheduled
	r.scheduled = nil
	var post []core.LocalNotification
	if away {
		// A ringing alarm is silenced on the way out: its sound would play
		// from a hidden app (Android) or freeze mid-ring (iOS). Its
		// notification was the OS's to post and the app was on screen, so it
		// is simply answered.
		r.silence()
		r.queue = nil
		post = r.notifications(now)
		for _, n := range post {
			r.scheduled = append(r.scheduled, n.ID)
		}
		r.awaySince = now
	} else {
		// Back on screen: anything that came due while away was rung by the
		// OS. A frozen goroutine (iOS) would otherwise ring it on its first
		// tick after resuming.
		r.last = now
		kept := r.snoozes[:0]
		for _, s := range r.snoozes {
			if s.at.After(now) {
				kept = append(kept, s)
			}
		}
		r.snoozes = kept
		// Only what was actually handed to the OS counts as rung: an app
		// that switched Notify on while away scheduled nothing.
		if len(cancel) > 0 && !r.awaySince.IsZero() {
			rangAway = r.dueBetween(r.awaySince, now)
		}
	}
	r.mu.Unlock()

	if sweep {
		// Ahead of the cancels below only by convention: this record
		// scheduled nothing before its first return, so there is nothing of
		// its own under the prefix for the sweep to take.
		r.sweepPrevious()
	}

	for _, id := range cancel {
		core.CancelNotification(id)
	}
	for _, n := range post {
		core.PostNotification(n)
	}
	return rangAway
}

// dueBetween lists the alarms whose notifications fired in (from, to]: each
// alarm once however many times it came due, soonest first. Snoozes dropped
// by check while away are not in it — a snooze is an alarm that has already
// been reported. Called with mu held.
func (r *alarmsRecord) dueBetween(from, to time.Time) []alarm.Alarm {
	type hit struct {
		a  alarm.Alarm
		at time.Time
	}
	var hits []hit
	for _, a := range r.alarms {
		if next := a.Next(from); !next.IsZero() && !next.After(to) {
			hits = append(hits, hit{a, next})
		}
	}
	slices.SortStableFunc(hits, func(x, y hit) int { return x.at.Compare(y.at) })
	out := make([]alarm.Alarm, len(hits))
	for i, h := range hits {
		out[i] = h.a
	}
	return out
}

// notifications lists what Notify schedules at now: each pending snooze, then
// every enabled alarm's occurrences within alarmNotifyHorizon, soonest first
// and at most alarmNotifyMax in all. Called with mu held.
//
// An id names the alarm and the instant, so two occurrences of one alarm are
// two requests and a reschedule of the same instant replaces rather than
// duplicates.
func (r *alarmsRecord) notifications(now time.Time) []core.LocalNotification {
	type due struct {
		a  alarm.Alarm
		at time.Time
	}
	var all []due
	for _, s := range r.snoozes {
		if s.at.After(now) {
			all = append(all, due{s.alarm, s.at})
		}
	}
	end := now.Add(alarmNotifyHorizon)
	for _, a := range r.alarms {
		for next := a.Next(now); !next.IsZero() && !next.After(end); next = a.Next(next) {
			all = append(all, due{a, next})
			// A one-time alarm rings at its next occurrence only; Next
			// would otherwise find the same time tomorrow, and the day after.
			if a.Once() {
				break
			}
		}
	}
	slices.SortStableFunc(all, func(x, y due) int { return x.at.Compare(y.at) })
	if len(all) > alarmNotifyMax {
		all = all[:alarmNotifyMax]
	}
	out := make([]core.LocalNotification, 0, len(all))
	for _, d := range all {
		title, body := r.opts.notifyText(d.a)
		out = append(out, core.LocalNotification{
			ID:    alarmNotifyPrefix + d.a.ID + "." + strconv.FormatInt(d.at.Unix(), 10),
			Title: title,
			Body:  body,
			At:    d.at,
		})
	}
	return out
}

// sweepPrevious cancels every alarm notification a previous process left with
// the OS and reports the ones that fired to OnRing. Called without mu held:
// the host may answer inside the call, and the answer takes mu.
func (r *alarmsRecord) sweepPrevious() {
	core.SweepNotifications(alarmNotifyPrefix, func(fired []string) {
		r.mu.Lock()
		rang := r.firedAlarms(fired)
		onRing := r.opts.OnRing
		r.mu.Unlock()
		if onRing == nil {
			return
		}
		for _, a := range rang {
			onRing(a)
		}
	})
}

// firedAlarms maps the ids a sweep reported back to alarms in the current
// list: each alarm once, soonest ring first. An id whose alarm is no longer
// in the list is skipped (OnRing takes an alarm.Alarm, and there is none to
// hand over), as is one that does not parse — it was not this hook's.
// Called with mu held.
//
// The id is "grmob.alarm.<alarm id>.<unix seconds>". An alarm id may itself
// contain dots, so the time is split off at the last one.
func (r *alarmsRecord) firedAlarms(fired []string) []alarm.Alarm {
	type hit struct {
		a  alarm.Alarm
		at int64
	}
	byID := map[string]alarm.Alarm{}
	for _, a := range r.alarms {
		byID[a.ID] = a
	}
	first := map[string]int64{}
	for _, id := range fired {
		rest, ok := strings.CutPrefix(id, alarmNotifyPrefix)
		if !ok {
			continue
		}
		dot := strings.LastIndexByte(rest, '.')
		if dot <= 0 {
			continue
		}
		at, err := strconv.ParseInt(rest[dot+1:], 10, 64)
		if err != nil {
			continue
		}
		alarmID := rest[:dot]
		if _, known := byID[alarmID]; !known {
			continue
		}
		if prev, seen := first[alarmID]; !seen || at < prev {
			first[alarmID] = at
		}
	}
	hits := make([]hit, 0, len(first))
	for id, at := range first {
		hits = append(hits, hit{byID[id], at})
	}
	// Ties broken by id so the order does not depend on map iteration.
	slices.SortFunc(hits, func(x, y hit) int {
		if x.at != y.at {
			return cmp.Compare(x.at, y.at)
		}
		return strings.Compare(x.a.ID, y.a.ID)
	})
	out := make([]alarm.Alarm, len(hits))
	for i, h := range hits {
		out[i] = h.a
	}
	return out
}
