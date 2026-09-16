package hooks

import (
	"sync"
	"time"

	"github.com/rohanthewiz/grmob/alarm"
	"github.com/rohanthewiz/grmob/core"
)

// AlarmOptions configures UseAlarms. Every field has a usable zero value.
type AlarmOptions struct {
	// OnRing is called once when an alarm starts ringing, including a snoozed
	// one coming back. It runs on the alarm goroutine, not in a render: write
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
}

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
}

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
// # In-app only
//
// This rings while the app is running and in the foreground, and not
// otherwise. iOS suspends a backgrounded app within seconds and Android cuts
// its network and may stop it (see core/notifications.go), and nothing here
// asks the OS to wake the app at a time. An alarm that must ring with the app
// closed needs a scheduled OS notification or the platform's alarm API, which
// core does not have yet.
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

	done := make(chan struct{})
	ctx.OnClose(func() {
		close(done)
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
