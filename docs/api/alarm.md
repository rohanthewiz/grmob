# Package alarm

```go
import "github.com/rohanthewiz/grmob/alarm"
```

Package alarm is the time arithmetic of an alarm clock: when an alarm next rings, and whether it came due between two instants.

	a := alarm.Alarm{ID: "wake", Hour: 6, Minute: 30, Days: alarm.Weekdays, Enabled: true}
	a.Next(now)               // the next 06:30 on a Monday–Friday, after now
	a.Due(lastCheck, now)     // did one fall in (lastCheck, now]?

It holds no clock and no goroutine. hooks.UseAlarms is what checks the alarms every second and rings them, and comps.AlarmRow / comps.AlarmRinging draw them; this package is the part of all that which can be tested with two time.Time values and nothing else.

## Due is an interval, not an equality

A check of "is it 06:30:00 now" misses an alarm whenever the tick that would have seen it arrives late or not at all — a garbage-collection pause, a render that took longer than a second, a process the OS paused for a moment. Asking whether the ring time fell in (last check, now] cannot miss one: a late tick finds the ring time inside a longer interval, and the next check starts where this one ended, so no instant is in two intervals and no alarm rings twice.

	last check        now
	    │                │
	────(────────●───────]────▶ time
	          06:30:00   └ Due: yes, however late this tick is

## Wall-clock times, in the caller's location

Hour and Minute are a time of day on the wall clock of whatever location the instants passed in carry, which is what a person setting an alarm means: 06:30 stays 06:30 after a flight or a daylight-saving change. On a spring-forward day a 02:30 alarm that does not exist rings at 03:30, just after the gap (see Alarm.on); on a fall-back day a 01:30 that happens twice rings the first time only.

## Index

- [Variables](#variables) — `Weekdays`, `Weekend`
- [`type Alarm`](#type-alarm)
    - [`func (Alarm) DaysLabel`](#func-alarm-dayslabel)
    - [`func (Alarm) Due`](#func-alarm-due)
    - [`func (Alarm) Next`](#func-alarm-next)
    - [`func (Alarm) Once`](#func-alarm-once)
    - [`func (Alarm) TimeLabel`](#func-alarm-timelabel)
    - [`func (Alarm) Valid`](#func-alarm-valid)

## Variables

Weekdays and Weekend are the two repeat sets alarm apps offer as presets.

```go
var (
	Weekdays = []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
	Weekend  = []time.Weekday{time.Saturday, time.Sunday}
)
```

<small>[alarm/alarm.go:66](https://github.com/rohanthewiz/grmob/blob/master/alarm/alarm.go#L66)</small>

## Types

### type Alarm

```go
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
```

Alarm is one alarm as a user set it.

<small>[alarm/alarm.go:44](https://github.com/rohanthewiz/grmob/blob/master/alarm/alarm.go#L44)</small>

#### func (Alarm) DaysLabel

```go
func (a Alarm) DaysLabel() string
```

DaysLabel is the repeat set as an alarm list shows it: "Once", "Every day", "Weekdays", "Weekends", or the days' short names in week order starting on Monday ("Mon Wed Fri").

<small>[alarm/alarm.go:144](https://github.com/rohanthewiz/grmob/blob/master/alarm/alarm.go#L144)</small>

#### func (Alarm) Due

```go
func (a Alarm) Due(from, to time.Time) bool
```

Due reports whether the alarm's ring time fell in (from, to]. See the package comment for why this is an interval.

<small>[alarm/alarm.go:122](https://github.com/rohanthewiz/grmob/blob/master/alarm/alarm.go#L122)</small>

#### func (Alarm) Next

```go
func (a Alarm) Next(after time.Time) time.Time
```

Next returns the first instant strictly after \`after\` at which the alarm rings, in after's location. It is the zero time for a disabled or invalid alarm.

Eight days are enough to search: a repeating alarm rings on at least one weekday, and every weekday recurs within seven days of any date; the eighth covers today's ring time having already passed.

<small>[alarm/alarm.go:86](https://github.com/rohanthewiz/grmob/blob/master/alarm/alarm.go#L86)</small>

#### func (Alarm) Once

```go
func (a Alarm) Once() bool
```

Once reports whether the alarm rings a single time rather than repeating.

<small>[alarm/alarm.go:72](https://github.com/rohanthewiz/grmob/blob/master/alarm/alarm.go#L72)</small>

#### func (Alarm) TimeLabel

```go
func (a Alarm) TimeLabel(hour24 bool) string
```

TimeLabel is the ring time on a 12- or 24-hour clock ("6:30 AM", "06:30").

<small>[alarm/alarm.go:187](https://github.com/rohanthewiz/grmob/blob/master/alarm/alarm.go#L187)</small>

#### func (Alarm) Valid

```go
func (a Alarm) Valid() bool
```

Valid reports whether Hour and Minute name a time of day.

<small>[alarm/alarm.go:75](https://github.com/rohanthewiz/grmob/blob/master/alarm/alarm.go#L75)</small>

