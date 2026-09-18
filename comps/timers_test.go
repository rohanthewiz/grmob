package comps

import (
	"strings"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// The harness is select_row_test.go's: one context, one pass per render, the
// protocol a Manager performs. Countdown and Stopwatch hold hooks for the same
// reason SelectRow does, so their tests cannot call Render bare either.

func TestCountdownIsOneLabelledTextInTheClockFace(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return Countdown{Until: time.Now().Add(90 * time.Second)}
	})
	defer h.ctx.Close()

	n := h.node
	if n.Type != "Text" {
		t.Fatalf("root = %q, want the single Text the type doc describes", n.Type)
	}
	if n.Props["content"] != "1:30" {
		t.Errorf("digits = %v, want 1:30", n.Props["content"])
	}
	// RoleImg with a label, DigitalClock's shape: read as text, "1:30" is
	// punctuation.
	if n.Style.AccessibilityRole != core.RoleImg {
		t.Errorf("role = %q, want img so the label survives on the web", n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilityLabel != "1 minute 30 seconds remaining" {
		t.Errorf("label = %q, want the spoken remaining time", n.Style.AccessibilityLabel)
	}
	if n.Style.FontSize != timerDigitSize || n.Style.FontWeight != core.Light {
		t.Errorf("face = %v px weight %v, want DigitalClock's %v px Light",
			n.Style.FontSize, n.Style.FontWeight, timerDigitSize)
	}
	if n.Style.TextColor != core.DefaultTheme.Colors.TextPrimary {
		t.Errorf("ink = %q, want the theme's TextPrimary", n.Style.TextColor)
	}
}

// The rounding claim in the type doc: the reading is never smaller than the
// time actually left, so a tick that arrives late shows a second too many
// rather than a second too few.
func TestCountdownRoundsUpSoItNeverUnderstatesTheTimeLeft(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return Countdown{Until: time.Now().Add(1500 * time.Millisecond)}
	})
	defer h.ctx.Close()

	if got := h.node.Props["content"]; got != "0:02" {
		t.Errorf("1.5s remaining reads %v, want 0:02", got)
	}
}

// Format is handed the same rounded number the default writes, so the two can
// never disagree about which second is on screen.
func TestCountdownFormatReceivesTheRoundedRemainder(t *testing.T) {
	var seen time.Duration
	h := newRowHarness(t, func() core.View {
		return Countdown{
			Until:  time.Now().Add(1500 * time.Millisecond),
			Format: func(d time.Duration) string { seen = d; return "soon" },
		}
	})
	defer h.ctx.Close()

	if seen != 2*time.Second {
		t.Errorf("Format saw %v, want the rounded 2s", seen)
	}
	if h.node.Props["content"] != "soon" {
		t.Errorf("digits = %v, want the caller's format", h.node.Props["content"])
	}
}

// A spent countdown says the fact, not a reading of zero.
func TestCountdownSpentReadsZeroAndSaysTimeIsUp(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return Countdown{Until: time.Now().Add(-time.Minute)}
	})
	defer h.ctx.Close()

	if h.node.Props["content"] != "0:00" {
		t.Errorf("digits = %v, want 0:00", h.node.Props["content"])
	}
	if h.node.Style.AccessibilityLabel != "Time is up" {
		t.Errorf("label = %q, want the fact rather than %q",
			h.node.Style.AccessibilityLabel, "0 seconds remaining")
	}
}

func TestCountdownHiddenIsDisplayNone(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return Countdown{Until: time.Now().Add(time.Minute), Hidden: true}
	})
	defer h.ctx.Close()

	if h.node.Style.Display != core.DisplayNone {
		t.Errorf("hidden countdown display = %q, want none", h.node.Style.Display)
	}
}

// OnDone fires once, on the pass that finds the deadline passed, and not on
// the passes either side of it. The effect runs on its own goroutine, so the
// count is collected through a channel rather than read from a variable.
func TestCountdownFiresOnDoneOnceOnTheCrossing(t *testing.T) {
	fired := make(chan struct{}, 8)
	until := time.Now().Add(time.Hour)
	h := newRowHarness(t, func() core.View {
		return Countdown{Until: until, OnDone: func() { fired <- struct{}{} }}
	})
	defer h.ctx.Close()

	if n := drain(fired, 60*time.Millisecond); n != 0 {
		t.Fatalf("OnDone fired %d times while counting", n)
	}

	// The deadline passes. Two more passes follow it, standing in for the
	// ticks that keep arriving while the widget is on screen.
	until = time.Now().Add(-time.Second)
	h.render()
	h.render()
	if n := drain(fired, 120*time.Millisecond); n != 1 {
		t.Errorf("OnDone fired %d times across the crossing, want exactly 1", n)
	}

	// Moving the deadline forward re-arms it: that is what a restart is.
	until = time.Now().Add(time.Hour)
	h.render()
	if n := drain(fired, 60*time.Millisecond); n != 0 {
		t.Fatalf("re-arming fired OnDone %d times", n)
	}
	until = time.Now().Add(-time.Second)
	h.render()
	if n := drain(fired, 120*time.Millisecond); n != 1 {
		t.Errorf("second crossing fired %d times, want 1", n)
	}
}

// A deadline already past on the first pass is a deadline that passed while
// the app was closed, and the widget says so rather than waiting for a
// crossing that has already happened.
func TestCountdownAlreadyPastFiresOnItsFirstPass(t *testing.T) {
	fired := make(chan struct{}, 4)
	h := newRowHarness(t, func() core.View {
		return Countdown{Until: time.Now().Add(-time.Minute), OnDone: func() { fired <- struct{}{} }}
	})
	defer h.ctx.Close()

	if n := drain(fired, 150*time.Millisecond); n != 1 {
		t.Errorf("OnDone fired %d times on mount, want 1", n)
	}
}

// The table in Countdown's type doc, pinned. There is no way to observe the
// pump's pause from a rendered tree, so the decision itself is the subject.
func TestCountdownTicksOnlyWhileItHasAReasonTo(t *testing.T) {
	onDone := func() {}
	cases := []struct {
		name string
		c    Countdown
		done bool
		want bool
	}{
		{"counting, visible", Countdown{}, false, true},
		{"counting, visible, no OnDone", Countdown{}, false, true},
		{"counting, hidden, owes an OnDone", Countdown{Hidden: true, OnDone: onDone}, false, true},
		{"counting, hidden, owes nothing", Countdown{Hidden: true}, false, false},
		{"finished, visible", Countdown{OnDone: onDone}, true, false},
		{"finished, hidden", Countdown{Hidden: true, OnDone: onDone}, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.c.ticking(c.done); got != c.want {
				t.Errorf("ticking(done=%v) = %v, want %v", c.done, got, c.want)
			}
		})
	}
}

// A Countdown with no deadline is permanently spent, which on screen is
// indistinguishable from one that has just finished.
func TestCountdownWithNoDeadlineIsAConcern(t *testing.T) {
	h := newQuietRowHarness(t, func() core.View { return Countdown{} })
	defer h.ctx.Close()

	dump := core.DumpConcerns()
	if !strings.Contains(dump, ConcernCountdownUntilUnset) {
		t.Errorf("want %s in the concerns, got:\n%s", ConcernCountdownUntilUnset, dump)
	}
	core.ClearConcerns()
}

func TestStopwatchSumsBankedTimeAndTheCurrentRun(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return Stopwatch{
			Elapsed: 65 * time.Second,
			Since:   time.Now().Add(-10 * time.Second),
			Running: true,
		}
	})
	defer h.ctx.Close()

	if got := h.node.Props["content"]; got != "1:15" {
		t.Errorf("digits = %v, want 1:15: 65s banked plus 10s running", got)
	}
	if h.node.Style.AccessibilityLabel != "1 minute 15 seconds elapsed" {
		t.Errorf("label = %q, want the spoken elapsed time", h.node.Style.AccessibilityLabel)
	}
}

// Paused, the banked time is the whole reading — and Since is not read, which
// is what makes "pause" a single assignment for the caller.
func TestStopwatchPausedReadsTheBankedTimeAndIgnoresSince(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return Stopwatch{Elapsed: 65 * time.Second}
	})
	defer h.ctx.Close()

	if got := h.node.Props["content"]; got != "1:05" {
		t.Errorf("paused digits = %v, want the banked 1:05", got)
	}
}

// The mirror of Countdown's rounding: a stopwatch never claims more elapsed
// time than has passed, so its first second reads 0:00.
func TestStopwatchTruncatesSoItNeverOverstatesElapsed(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return Stopwatch{Elapsed: 1900 * time.Millisecond}
	})
	defer h.ctx.Close()

	if got := h.node.Props["content"]; got != "0:01" {
		t.Errorf("1.9s elapsed reads %v, want 0:01", got)
	}
}

// A run that started at the zero time reads as seventeen million hours, which
// is only obviously wrong to somebody reading the digits.
func TestStopwatchRunningWithNoSinceIsAConcern(t *testing.T) {
	h := newQuietRowHarness(t, func() core.View { return Stopwatch{Running: true} })
	defer h.ctx.Close()

	dump := core.DumpConcerns()
	if !strings.Contains(dump, ConcernStopwatchSinceUnset) {
		t.Errorf("want %s in the concerns, got:\n%s", ConcernStopwatchSinceUnset, dump)
	}
	core.ClearConcerns()
}

// A paused stopwatch with no Since is the ordinary reset state and says
// nothing: the harness's own concern assertion is the test.
func TestStopwatchPausedWithNoSinceIsQuiet(t *testing.T) {
	h := newRowHarness(t, func() core.View { return Stopwatch{} })
	defer h.ctx.Close()

	if got := h.node.Props["content"]; got != "0:00" {
		t.Errorf("reset digits = %v, want 0:00", got)
	}
}

func TestStopwatchHiddenIsDisplayNone(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return Stopwatch{Elapsed: time.Minute, Hidden: true}
	})
	defer h.ctx.Close()

	if h.node.Style.Display != core.DisplayNone {
		t.Errorf("hidden stopwatch display = %q, want none", h.node.Style.Display)
	}
}

// Style is applied after everything the widget sets, including the face Size
// sets, which is the promise every widget in this package makes.
func TestTimersApplyTheCallerStyleLast(t *testing.T) {
	for _, c := range []struct {
		name string
		view core.View
	}{
		{"Countdown", Countdown{Until: time.Now().Add(time.Minute), Size: 40,
			Style: []core.StyleProp{core.FontSize(18), core.TextColor("#abcdef")}}},
		{"Stopwatch", Stopwatch{Elapsed: time.Minute, Size: 40,
			Style: []core.StyleProp{core.FontSize(18), core.TextColor("#abcdef")}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newRowHarness(t, func() core.View { return c.view })
			defer h.ctx.Close()
			if h.node.Style.FontSize != 18 || h.node.Style.TextColor != "#abcdef" {
				t.Errorf("style = %v px %q, want the caller's 18 px #abcdef",
					h.node.Style.FontSize, h.node.Style.TextColor)
			}
		})
	}
}

func TestTimerDigitsFollowThePhoneTimerFormat(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{-time.Second, "0:00"}, // clamped, so a negative can never print a "-"
		{0, "0:00"},
		{7 * time.Second, "0:07"},
		{65 * time.Second, "1:05"},
		{3599 * time.Second, "59:59"},   // the last reading before the hours field
		{3600 * time.Second, "1:00:00"}, // and the first one after it
		{26 * time.Hour, "26:00:00"},    // days are hours: the field only grows
	}
	for _, c := range cases {
		if got := timerDigits(c.d); got != c.want {
			t.Errorf("timerDigits(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestCeilSecondRoundsUpAndClampsAtZero(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want time.Duration
	}{
		{-time.Hour, 0},
		{0, 0},
		{time.Nanosecond, time.Second},
		{time.Second, time.Second}, // already whole: not bumped to two
		{1500 * time.Millisecond, 2 * time.Second},
	}
	for _, c := range cases {
		if got := ceilSecond(c.d); got != c.want {
			t.Errorf("ceilSecond(%v) = %v, want %v", c.d, got, c.want)
		}
	}
}

func TestSpokenDurationNamesOnlyTheFieldsThatArePresent(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0 seconds"},
		{500 * time.Millisecond, "0 seconds"},
		{time.Second, "1 second"},
		{90 * time.Second, "1 minute 30 seconds"},
		{time.Hour, "1 hour"},
		{2*time.Hour + 5*time.Second, "2 hours 5 seconds"}, // no "0 minutes"
		{25*time.Hour + 61*time.Second, "25 hours 1 minute 1 second"},
	}
	for _, c := range cases {
		if got := spokenDuration(c.d); got != c.want {
			t.Errorf("spokenDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

// drain counts what arrives on ch within window. The effect hook runs on its
// own goroutine, so "fired once" is a claim about a window rather than about
// the moment the render returned; the window has to outlast a goroutine
// start, and the zero cases have to be long enough that a late fire would
// have been caught.
func drain(ch <-chan struct{}, window time.Duration) int {
	n := 0
	deadline := time.After(window)
	for {
		select {
		case <-ch:
			n++
		case <-deadline:
			return n
		}
	}
}
