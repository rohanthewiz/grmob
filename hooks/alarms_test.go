package hooks

import (
	"strconv"
	"strings"
	"sync"
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

// notificationHost records the notification system events core sends.
func notificationHost(t *testing.T) *[]map[string]any {
	t.Helper()
	var seen []map[string]any
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name == "notification" {
			seen = append(seen, data)
		}
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })
	return &seen
}

// Going to the background with Notify schedules the week's occurrences, soonest
// first, a one-time alarm once and a snooze by its own time; coming back
// cancels exactly those.
func TestAlarmsNotifyScheduleWhileAwayAndCancelOnReturn(t *testing.T) {
	seen := notificationHost(t)
	// 2026-09-16 is a Wednesday.
	weekdays := alarm.Alarm{ID: "wake", Hour: 6, Minute: 30, Days: alarm.Weekdays, Enabled: true, Label: "Wake up"}
	once := alarm.Alarm{ID: "once", Hour: 13, Enabled: true}
	off := alarm.Alarm{ID: "off", Hour: 14}
	r := newRecord(at(12, 0, 0), weekdays, once, off)
	r.opts.Notify = true
	r.snoozes = []snoozedAlarm{{alarm: weekdays, at: at(12, 5, 0)}}

	rang := r.setAway(true, at(12, 0, 0))
	if len(rang) != 0 {
		t.Errorf("reported %v on the way out", rang)
	}
	var posts []map[string]any
	for _, e := range *seen {
		if e["command"] == "post" {
			posts = append(posts, e)
		}
	}
	// Snooze 12:05, once 13:00, then wake on Thu, Fri, Mon, Tue, Wed (the
	// week ends at next Wednesday 12:00, after its 06:30).
	want := []string{
		"grmob.alarm.wake." + itoa(at(12, 5, 0)),
		"grmob.alarm.once." + itoa(at(13, 0, 0)),
	}
	for _, d := range []int{17, 18, 21, 22, 23} {
		want = append(want, "grmob.alarm.wake."+itoa(time.Date(2026, 9, d, 6, 30, 0, 0, time.UTC)))
	}
	if len(posts) != len(want) {
		t.Fatalf("posted %d, want %d: %v", len(posts), len(want), posts)
	}
	for i, p := range posts {
		if p["id"] != want[i] {
			t.Errorf("post %d id = %v, want %v", i, p["id"], want[i])
		}
	}
	if posts[0]["title"] != "Wake up" || posts[0]["body"] != "6:30 AM" || posts[1]["title"] != "Alarm" {
		t.Errorf("text: %v / %v", posts[0], posts[1])
	}

	// While away nothing rings in the app, and the snooze that came due is
	// dropped (the OS rang it).
	if rang, changed := r.check(at(13, 0, 1)); rang != nil || changed || len(r.snoozes) != 0 {
		t.Errorf("rang %v in the background; snoozes %v", rang, r.snoozes)
	}

	*seen = nil
	rang = r.setAway(false, at(13, 30, 0))
	cancels := 0
	for _, e := range *seen {
		if e["command"] != "cancel" {
			t.Errorf("unexpected %v on return", e)
		}
		cancels++
	}
	if cancels != len(want) {
		t.Errorf("cancelled %d, want %d", cancels, len(want))
	}
	// The one-time alarm came due while away and is reported, so the app can
	// switch it off; nothing is rung again by the next check.
	if len(rang) != 1 || rang[0].ID != "once" {
		t.Errorf("reported %v, want the one-time alarm", rang)
	}
	if rang, _ := r.check(at(13, 30, 1)); rang != nil {
		t.Errorf("rang %v again after returning", rang)
	}
}

// Without Notify, the background changes nothing: no OS calls, and the in-app
// ringer keeps its old behaviour.
func TestAlarmsWithoutNotifyIgnoreTheBackground(t *testing.T) {
	seen := notificationHost(t)
	r := newRecord(at(6, 59, 59), alarm.Alarm{ID: "wake", Hour: 7, Enabled: true})
	r.setAway(true, at(6, 59, 59))
	if rang, _ := r.check(at(7, 0, 0)); rang == nil {
		t.Error("did not ring in the background without Notify")
	}
	r.setAway(false, at(7, 0, 1))
	if len(*seen) != 0 {
		t.Errorf("sent %v without Notify", *seen)
	}
}

func itoa(t time.Time) string { return strconv.FormatInt(t.Unix(), 10) }

// sweepHost answers every notification sweep inline with fired, the way a
// native host may answer inside SendSystemEvent's call, and counts the sweeps.
func sweepHost(t *testing.T, fired ...string) *[]string {
	t.Helper()
	var prefixes []string
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name != "notification" || data["command"] != "sweep" {
			return
		}
		prefixes = append(prefixes, data["prefix"].(string))
		core.ReceiveHostEvent("notification_swept", map[string]any{
			"request": data["request"],
			"fired":   fired,
		})
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })
	return &prefixes
}

// Mounting in the foreground sweeps the previous process's alarm
// notifications, and what the host says fired reaches OnRing: each alarm still
// in the list once, soonest first. Ids for deleted alarms or of another shape
// are skipped, and an alarm id containing dots still parses.
func TestUseAlarmsSweepsAndReportsWhatFiredWhileClosed(t *testing.T) {
	prefixes := sweepHost(t,
		"grmob.alarm.wake.200",
		"grmob.alarm.wake.100", // the same alarm, earlier: reported once, at 100
		"grmob.alarm.nap.v2.150",
		"grmob.alarm.deleted.50",
		"grmob.alarm.nodot",
		"grmob.alarm.wake.notanumber",
		"someone.else.10",
	)
	ctx := core.NewContext()
	defer ctx.Close()

	var mu sync.Mutex
	var rang []string
	UseAlarms(ctx, []alarm.Alarm{
		{ID: "wake", Hour: 6, Enabled: true},
		{ID: "nap.v2", Hour: 14, Enabled: true},
	}, AlarmOptions{Notify: true, OnRing: func(a alarm.Alarm) {
		mu.Lock()
		rang = append(rang, a.ID)
		mu.Unlock()
	}})

	if len(*prefixes) != 1 || (*prefixes)[0] != "grmob.alarm." {
		t.Fatalf("sweeps = %v, want one of grmob.alarm.", *prefixes)
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(rang, ",") != "wake,nap.v2" {
		t.Errorf("OnRing heard %v, want [wake nap.v2]", rang)
	}

	// A later render does not sweep again.
	ctx.Reset()
	UseAlarms(ctx, nil, AlarmOptions{Notify: true})
	if len(*prefixes) != 1 {
		t.Errorf("a second render swept again: %v", *prefixes)
	}
}

// A record that mounted in the background leaves the OS's alarms alone until
// the first return to the foreground, and sweeps then, once.
func TestAlarmsMountedAwaySweepOnFirstReturn(t *testing.T) {
	prefixes := sweepHost(t, "grmob.alarm.wake.100")
	wake := alarm.Alarm{ID: "wake", Hour: 6, Enabled: true}
	var rang []string
	r := newRecord(at(12, 0, 0), wake)
	r.opts = AlarmOptions{Notify: true, OnRing: func(a alarm.Alarm) { rang = append(rang, a.ID) }}
	r.away, r.sweepOnReturn = true, true

	r.setAway(false, at(12, 1, 0))
	if len(*prefixes) != 1 || len(rang) != 1 || rang[0] != "wake" {
		t.Fatalf("first return: sweeps %v, rang %v", *prefixes, rang)
	}
	r.setAway(true, at(12, 2, 0))
	r.setAway(false, at(12, 3, 0))
	if len(*prefixes) != 1 {
		t.Errorf("swept again on a later return: %v", *prefixes)
	}
}
