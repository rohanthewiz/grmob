package alarm

import (
	"testing"
	"time"
)

// Wednesday 16 September 2026.
func wed(h, m, s int) time.Time { return time.Date(2026, 9, 16, h, m, s, 0, time.UTC) }

func TestNext(t *testing.T) {
	cases := []struct {
		name  string
		a     Alarm
		after time.Time
		want  time.Time
	}{
		{"once, later today", Alarm{Hour: 7, Minute: 0, Enabled: true}, wed(6, 0, 0), wed(7, 0, 0)},
		{"once, already passed today", Alarm{Hour: 7, Minute: 0, Enabled: true}, wed(8, 0, 0), wed(7, 0, 0).AddDate(0, 0, 1)},
		{"exactly at the ring time is not after it", Alarm{Hour: 7, Minute: 0, Enabled: true}, wed(7, 0, 0), wed(7, 0, 0).AddDate(0, 0, 1)},
		{"weekdays from Friday evening skips the weekend", Alarm{Hour: 6, Minute: 30, Days: Weekdays, Enabled: true},
			time.Date(2026, 9, 18, 20, 0, 0, 0, time.UTC), time.Date(2026, 9, 21, 6, 30, 0, 0, time.UTC)},
		{"a single weekday a week out", Alarm{Hour: 9, Minute: 0, Days: []time.Weekday{time.Wednesday}, Enabled: true},
			wed(10, 0, 0), time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)},
		{"disabled", Alarm{Hour: 7, Enabled: false}, wed(6, 0, 0), time.Time{}},
		{"invalid", Alarm{Hour: 24, Enabled: true}, wed(6, 0, 0), time.Time{}},
	}
	for _, c := range cases {
		if got := c.a.Next(c.after); !got.Equal(c.want) {
			t.Errorf("%s: Next = %v, want %v", c.name, got, c.want)
		}
	}
}

// A late or skipped tick still finds the alarm, and consecutive intervals ring
// it exactly once.
func TestDueIsAnInterval(t *testing.T) {
	a := Alarm{Hour: 7, Minute: 0, Enabled: true}
	if !a.Due(wed(6, 59, 58), wed(7, 0, 3)) {
		t.Error("a tick three seconds late missed the alarm")
	}
	rings := 0
	for s := 0; s < 10; s++ {
		from := wed(6, 59, 55).Add(time.Duration(s) * time.Second)
		if a.Due(from, from.Add(time.Second)) {
			rings++
		}
	}
	if rings != 1 {
		t.Errorf("rang %d times across consecutive one-second checks, want 1", rings)
	}
	if a.Due(wed(7, 0, 0), wed(7, 0, 1)) {
		t.Error("an interval starting at the ring time rang again")
	}
}

// On a spring-forward day the nonexistent time rings when time.Date puts it.
func TestNextAcrossDaylightSaving(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no tz database")
	}
	// 8 March 2026: clocks go from 02:00 to 03:00.
	a := Alarm{Hour: 2, Minute: 30, Enabled: true}
	got := a.Next(time.Date(2026, 3, 8, 1, 0, 0, 0, ny))
	if h, m, _ := got.Clock(); h != 3 || m != 30 {
		t.Errorf("spring-forward 02:30 rang at %v", got)
	}
}

func TestLabels(t *testing.T) {
	for days, want := range map[string]struct {
		d    []time.Weekday
		want string
	}{
		"once":     {nil, "Once"},
		"all":      {[]time.Weekday{0, 1, 2, 3, 4, 5, 6}, "Every day"},
		"weekdays": {Weekdays, "Weekdays"},
		"weekend":  {Weekend, "Weekends"},
		"mixed":    {[]time.Weekday{time.Friday, time.Monday, time.Sunday}, "Mon Fri Sun"},
	} {
		if got := (Alarm{Days: want.d}).DaysLabel(); got != want.want {
			t.Errorf("%s: DaysLabel = %q, want %q", days, got, want.want)
		}
	}
	for _, c := range []struct {
		h, m   int
		h24    bool
		expect string
	}{{0, 5, false, "12:05 AM"}, {12, 0, false, "12:00 PM"}, {18, 30, false, "6:30 PM"}, {6, 30, true, "06:30"}} {
		if got := (Alarm{Hour: c.h, Minute: c.m}).TimeLabel(c.h24); got != c.expect {
			t.Errorf("TimeLabel(%d:%d) = %q, want %q", c.h, c.m, got, c.expect)
		}
	}
}

// On a fall-back day an ordinary hour after the transition is still its own
// wall-clock time, and the repeated hour rings once.
func TestNextOnFallBack(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no tz database")
	}
	// 1 November 2026: clocks go from 02:00 back to 01:00.
	day := time.Date(2026, 11, 1, 0, 0, 0, 0, ny)
	if got := (Alarm{Hour: 5, Enabled: true}).Next(day); got.Hour() != 5 {
		t.Errorf("05:00 on fall-back day rang at %v", got)
	}
	a := Alarm{Hour: 1, Minute: 30, Enabled: true}
	first := a.Next(day)
	if again := a.Next(first); again.Day() == first.Day() {
		t.Errorf("repeated 01:30 rang twice in one day: %v then %v", first, again)
	}
}
