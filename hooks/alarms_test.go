package hooks

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/alarm"
	"github.com/rohanthewiz/grmob/core"
)

func at(h, m, s int) time.Time { return time.Date(2026, 9, 16, h, m, s, 0, time.UTC) }

func newRecord(start time.Time, alarms ...alarm.Alarm) *alarmsRecord {
	return &alarmsRecord{alarms: alarms, last: start}
}

// Late ticks still ring, a ringing alarm reports once, and it times out.
func TestAlarmsCheckRingsOnceAndTimesOut(t *testing.T) {
	wake := alarm.Alarm{ID: "wake", Hour: 7, Enabled: true}
	r := newRecord(at(6, 59, 58), wake)
	r.opts.RingFor = time.Minute

	if rang, changed := r.check(at(6, 59, 59)); rang != nil || changed {
		t.Fatal("rang before its time")
	}
	// Four seconds late.
	rang, changed := r.check(at(7, 0, 3))
	if rang == nil || rang.ID != "wake" || !changed {
		t.Fatalf("late tick: rang=%v changed=%v", rang, changed)
	}
	if rang, changed := r.check(at(7, 0, 4)); rang != nil || changed {
		t.Error("a ringing alarm reported again")
	}
	if _, changed := r.check(at(7, 1, 3)); !changed || r.ringing != nil {
		t.Error("did not time out after RingFor")
	}
}

// A second alarm due while one rings waits, and rings when the first stops.
func TestAlarmsQueueWhileRinging(t *testing.T) {
	a := alarm.Alarm{ID: "a", Hour: 7, Enabled: true}
	b := alarm.Alarm{ID: "b", Hour: 7, Minute: 1, Enabled: true}
	r := newRecord(at(6, 59, 59), a, b)
	r.check(at(7, 0, 0))
	r.check(at(7, 1, 0))
	if r.ringing.ID != "a" || len(r.queue) != 1 {
		t.Fatalf("ringing %v, queue %v", r.ringing, r.queue)
	}
	ringer := AlarmRinger{rec: r, ctx: core.NewContext()}
	ringer.Dismiss()
	if rang, _ := r.check(at(7, 1, 1)); rang == nil || rang.ID != "b" {
		t.Errorf("queued alarm did not ring after dismissal: %v", rang)
	}
}

// Snooze silences now and rings again after the snooze length.
func TestAlarmsSnooze(t *testing.T) {
	r := newRecord(at(6, 59, 59), alarm.Alarm{ID: "wake", Hour: 7, Enabled: true})
	r.opts.Snooze = 5 * time.Minute
	r.check(at(7, 0, 0))
	ringer := AlarmRinger{rec: r, ctx: core.NewContext()}
	ringer.Snooze()
	if _, ok := ringer.Ringing(); ok {
		t.Fatal("still ringing after Snooze")
	}
	a, when, ok := ringer.Snoozed()
	if !ok || a.ID != "wake" {
		t.Fatalf("Snoozed = %v %v %v", a, when, ok)
	}
	// Snooze stamps from the real clock, so move the snooze onto the test's.
	r.snoozes[0].at = at(7, 5, 0)
	if rang, _ := r.check(at(7, 4, 59)); rang != nil {
		t.Error("snooze came back early")
	}
	if rang, _ := r.check(at(7, 5, 0)); rang == nil || rang.ID != "wake" {
		t.Error("snooze did not come back")
	}
}

// A disabled alarm and an edit take effect through the list the render passes.
func TestUseAlarmsRereadsTheListEachRender(t *testing.T) {
	ctx := core.NewContext()
	defer ctx.Close()
	pass := func(alarms ...alarm.Alarm) AlarmRinger {
		ctx.Reset()
		return UseAlarms(ctx, alarms, AlarmOptions{})
	}
	pass(alarm.Alarm{ID: "x", Hour: 7, Enabled: true})
	ringer := pass(alarm.Alarm{ID: "x", Hour: 7, Enabled: false})
	ringer.rec.mu.Lock()
	enabled := ringer.rec.alarms[0].Enabled
	ringer.rec.mu.Unlock()
	if enabled {
		t.Error("the second render's list did not replace the first")
	}
}
