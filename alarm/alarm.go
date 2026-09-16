// Package alarm is the time arithmetic of an alarm clock: when an alarm next
// rings, and whether it came due between two instants.
//
//	a := alarm.Alarm{ID: "wake", Hour: 6, Minute: 30, Days: alarm.Weekdays, Enabled: true}
//	a.Next(now)               // the next 06:30 on a Monday–Friday, after now
//	a.Due(lastCheck, now)     // did one fall in (lastCheck, now]?
//
// It holds no clock and no goroutine. hooks.UseAlarms is what checks the
// alarms every second and rings them, and comps.AlarmRow / comps.AlarmRinging
// draw them; this package is the part of all that which can be tested with two
// time.Time values and nothing else.
//
// # Due is an interval, not an equality
//
// A check of "is it 06:30:00 now" misses an alarm whenever the tick that would
// have seen it arrives late or not at all — a garbage-collection pause, a
// render that took longer than a second, a process the OS paused for a moment.
// Asking whether the ring time fell in (last check, now] cannot miss one: a
// late tick finds the ring time inside a longer interval, and the next check
// starts where this one ended, so no instant is in two intervals and no alarm
// rings twice.
//
//	last check        now
//	    │                │
//	────(────────●───────]────▶ time
//	          06:30:00   └ Due: yes, however late this tick is
//
// # Wall-clock times, in the caller's location
//
// Hour and Minute are a time of day on the wall clock of whatever location the
// instants passed in carry, which is what a person setting an alarm means: 06:30
// stays 06:30 after a flight or a daylight-saving change. On a spring-forward
// day a 02:30 alarm that does not exist rings at 03:30, just after the gap (see
// Alarm.on); on a fall-back day a 01:30 that happens twice rings the first time
// only.
package alarm

import (
	"fmt"
	"time"
)

// Alarm is one alarm as a user set it.
type Alarm struct {
	// ID identifies the alarm across edits, for keyed rows and for telling a
	// ringing alarm's buttons which one they act on.
	ID string

	// Hour (0–23) and Minute (0–59) are the wall-clock time it rings at.
	Hour, Minute int

	// Label is what the ringing screen shows, and may be empty.
	Label string

	// Days are the weekdays it repeats on. Empty means once: it rings at the
	// next occurrence of the time, and the app is expected to turn it off
	// when it does (hooks.UseAlarms reports a ring; it does not edit the
	// app's list).
	Days []time.Weekday

	// Enabled is the switch beside the alarm. A disabled alarm is never due.
	Enabled bool
}

// Weekdays and Weekend are the two repeat sets alarm apps offer as presets.
var (
	Weekdays = []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
	Weekend  = []time.Weekday{time.Saturday, time.Sunday}
)

// Once reports whether the alarm rings a single time rather than repeating.
func (a Alarm) Once() bool { return len(a.Days) == 0 }

// Valid reports whether Hour and Minute name a time of day.
func (a Alarm) Valid() bool {
	return a.Hour >= 0 && a.Hour < 24 && a.Minute >= 0 && a.Minute < 60
}

// Next returns the first instant strictly after `after` at which the alarm
// rings, in after's location. It is the zero time for a disabled or invalid
// alarm.
//
// Eight days are enough to search: a repeating alarm rings on at least one
// weekday, and every weekday recurs within seven days of any date; the eighth
// covers today's ring time having already passed.
func (a Alarm) Next(after time.Time) time.Time {
	if !a.Enabled || !a.Valid() {
		return time.Time{}
	}
	y, m, d := after.Date()
	for offset := 0; offset <= 7; offset++ {
		t := a.on(y, m, d+offset, after.Location())
		if !t.After(after) || !a.ringsOn(t.Weekday()) {
			continue
		}
		return t
	}
	return time.Time{}
}

// on is the ring time on one calendar day.
//
// time.Date does not promise which side of a daylight-saving gap it resolves a
// wall time that does not exist to, and in practice picks the earlier offset —
// 02:30 on a spring-forward day comes back as 01:30, an hour before the clock
// ever reads the alarm's time. So a result whose wall clock is not the one
// asked for is recomputed as elapsed time from midnight, which lands a
// nonexistent 02:30 at 03:30: the first moment after the gap, where the alarm
// would have been had the clock not jumped. Only then, because elapsed time
// from midnight is wrong on every other hour of a transition day.
func (a Alarm) on(y int, m time.Month, d int, loc *time.Location) time.Time {
	t := time.Date(y, m, d, a.Hour, a.Minute, 0, 0, loc)
	if t.Hour() == a.Hour && t.Minute() == a.Minute {
		return t
	}
	midnight := time.Date(y, m, d, 0, 0, 0, 0, loc)
	return midnight.Add(time.Duration(a.Hour)*time.Hour + time.Duration(a.Minute)*time.Minute)
}

// Due reports whether the alarm's ring time fell in (from, to]. See the package
// comment for why this is an interval.
func (a Alarm) Due(from, to time.Time) bool {
	next := a.Next(from)
	return !next.IsZero() && !next.After(to)
}

// ringsOn reports whether a repeating alarm includes the weekday; a one-time
// alarm rings on any.
func (a Alarm) ringsOn(w time.Weekday) bool {
	if a.Once() {
		return true
	}
	for _, d := range a.Days {
		if d == w {
			return true
		}
	}
	return false
}

// DaysLabel is the repeat set as an alarm list shows it: "Once", "Every day",
// "Weekdays", "Weekends", or the days' short names in week order starting on
// Monday ("Mon Wed Fri").
func (a Alarm) DaysLabel() string {
	if a.Once() {
		return "Once"
	}
	var set [7]bool
	for _, d := range a.Days {
		if d >= time.Sunday && d <= time.Saturday {
			set[d] = true
		}
	}
	all, weekdays, weekend := true, true, true
	for d := time.Sunday; d <= time.Saturday; d++ {
		isWeekend := d == time.Saturday || d == time.Sunday
		all = all && set[d]
		if isWeekend {
			weekend = weekend && set[d]
			weekdays = weekdays && !set[d]
		} else {
			weekdays = weekdays && set[d]
			weekend = weekend && !set[d]
		}
	}
	switch {
	case all:
		return "Every day"
	case weekdays:
		return "Weekdays"
	case weekend:
		return "Weekends"
	}
	out := ""
	for _, d := range []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday} {
		if set[d] {
			if out != "" {
				out += " "
			}
			out += d.String()[:3]
		}
	}
	return out
}

// TimeLabel is the ring time on a 12- or 24-hour clock ("6:30 AM", "06:30").
func (a Alarm) TimeLabel(hour24 bool) string {
	if hour24 {
		return fmt.Sprintf("%02d:%02d", a.Hour, a.Minute)
	}
	h, marker := a.Hour%12, "AM"
	if h == 0 {
		h = 12
	}
	if a.Hour >= 12 {
		marker = "PM"
	}
	return fmt.Sprintf("%d:%02d %s", h, a.Minute, marker)
}
