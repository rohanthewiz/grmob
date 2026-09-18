package comps

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// ConcernCountdownUntilUnset is raised, in debug builds only, when Until is
// the zero time.Time. The countdown is then permanently expired: it draws
// 0:00 and fires OnDone on its first pass, which on screen is exactly what a
// timer that has just finished looks like. So a Countdown rendered before its
// deadline was assigned — a struct built from a half-filled record, a field
// spelled differently in the caller — would otherwise announce itself as a
// completed timer and nobody would go looking.
const ConcernCountdownUntilUnset = "countdown-until-unset"

// ConcernStopwatchSinceUnset is raised, in debug builds only, when Running is
// true and Since is the zero time.Time. The elapsed time is then measured
// from year 1, which reads as a seventeen-million-hour stopwatch — visibly
// wrong, but only if somebody is looking at the digits rather than at a
// screenshot, and silently wrong in the accessible label.
const ConcernStopwatchSinceUnset = "stopwatch-since-unset"

// Countdown draws the time left until a deadline, one tick a second, and
// reports once when it runs out.
//
//	comps.Countdown{Until: expiresAt, OnDone: func() { code.Set("") }}
//
//	  2:59      the digits, in DigitalClock's face
//
// It is the half of the clock family that watches a *duration* rather than an
// instant: an alarm row that wants to say how long until it rings, a
// one-time-code field that wants to say how long the code is good for, a
// rest timer between sets. Stopwatch, below, is the same widget counting the
// other way.
//
// # It holds hooks, so it is not conditional-safe
//
// Unlike DigitalClock — which takes a time.Time and holds nothing — a
// countdown has to know what "now" is, so it owns a tick. That makes it a
// hook caller, with the rule Accordion and Snackbar document: render it in a
// stable position on every pass and drive Hidden, rather than wrapping it in
// a core.If. A hidden countdown is Display none, so it costs no pixels.
//
// # What the tick is, and what it is not
//
// The tick is hooks.UseIntervalWhile with an empty callback: the widget reads
// the clock itself in Render, so all a tick has to do is bring the render
// back. It runs only while there is a reason for it:
//
//	state                       ticking
//	────────────────────────    ───────
//	counting, visible           yes
//	counting, Hidden            only if OnDone is set
//	finished                    no
//	Hidden and no OnDone        no
//
// The middle row is the one worth stating. Hiding a countdown removes the
// first reason to tick (nothing to draw) but not the second (somebody is
// waiting to be told it ran out), so a hidden countdown that owes an OnDone
// keeps counting. A hidden one that owes nothing stops dead.
//
// # Why not hooks.UseNow
//
// UseNow is the clock hook and aligns its ticks to the wall clock, so a
// DigitalClock changes its seconds digit when the phone's status bar does.
// A countdown has no such phase to share: its own boundaries fall at Until
// minus a whole number of seconds, which is a phase nothing else on the
// screen is on. There being nothing to align to, the cheaper hook wins — and
// UseNow cannot be paused, which the table above needs.
//
// The visible consequence is that the deadline is noticed on the first tick
// at or after it, so the digits reach 0:00 up to a second late. The reading
// is rounded *up* to compensate, which makes the lag conservative rather than
// arbitrary: a Countdown never tells you that you have less time left than
// you do.
//
// # OnDone comes from the effect, not from the render
//
// A render pass is not a place to run a handler — it may run more than once
// for one state, it runs while the tree is being built, and a handler that
// set state from inside it would re-enter the renderer. So OnDone is a
// hooks.UseEffect keyed on whether the deadline has passed, which gives it
// exactly the semantics the name implies:
//
//	remaining  5s ──── 4s ──── … ──── 1s ──── 0 ──── 0 ──── 0
//	deps       false   false         false   true   true   true
//	OnDone      ·       ·             ·      fire    ·      ·
//
// Once per crossing, on the tick that crosses, off the render goroutine. Two
// consequences follow from "per crossing" rather than "per widget":
//
//   - A Countdown whose Until is already in the past when it first renders
//     fires immediately. That is the correct reading of a deadline restored
//     from disk while the app was closed, and it is why the zero Until is a
//     concern rather than a quiet no-op.
//   - Moving Until forward re-arms it. A restart is Until: time.Now().Add(d)
//     and nothing else; the widget needs no reset call.
//
// OnDone is a display-grade signal and not a scheduler. It only fires while
// the widget is rendered and the app is running, and it is late by up to one
// tick. Something that must happen at a time whether or not anyone is
// looking belongs in the alarm package and hooks.UseAlarms.
//
// # Accessibility
//
// The digits are one element with RoleImg and a spoken label — "4 minutes 12
// seconds remaining", "Time is up" — for the reason DigitalClock gives: read
// as text, "4:12" is punctuation, and RoleImg is what makes a label survive
// on the web. The label is not a live region, so a screen reader is not told
// the new number every second; a caller who wants the announcement puts the
// Countdown beside its own core.RoleStatus text.
//
// # Theme roles read
//
//	Digits   Colors.TextPrimary, unless Color says otherwise
type Countdown struct {
	// Until is the deadline. The widget draws the time from now to here,
	// clamped at zero.
	Until time.Time

	// OnDone is called once when the deadline passes, from the hook's effect
	// and so on its own goroutine. Nil is a display-only countdown, which
	// also stops ticking the moment it is hidden.
	OnDone func()

	// Format writes the digits. It receives the remaining time already
	// rounded up to a whole second — the same number the default writes — so
	// a custom format cannot disagree with the widget about which second it
	// is showing. Nil uses the phone-timer format: M:SS under an hour,
	// H:MM:SS at or over one.
	Format func(time.Duration) string

	// Size is the digits' font size in px; 0 means 40, as in DigitalClock.
	Size float64

	// Color inks the digits; empty uses the theme's TextPrimary.
	Color string

	// Hidden removes the widget from display, which also stops its tick
	// unless OnDone is still owed. Prefer it to leaving the widget out of the
	// tree; see "It holds hooks".
	Hidden bool

	// AccessibilityLabel overrides the spoken remaining time.
	AccessibilityLabel string

	// Style is applied last, to the digits.
	Style []core.StyleProp
}

// Render reads the clock once, arms the tick and the effect, and draws the
// digits.
func (c Countdown) Render(ctx *core.Context) *core.Node {
	// One clock read for the whole pass, and it happens before the hooks
	// because the hooks' arguments depend on it: whether to keep ticking and
	// whether the deadline has passed are both answers about this instant.
	// The hook cannot both report the instant and be told, in the same call,
	// what the instant means.
	remaining := time.Until(c.Until)
	done := remaining <= 0

	// Unconditional and in a fixed order, as every hook caller must be. The
	// callback is empty on purpose: see "What the tick is".
	hooks.UseIntervalWhile(ctx, c.ticking(done), timerTick, time.Second)
	// done is the whole of the deps, so the effect runs on the pass where it
	// changes and on no other.
	hooks.UseEffect(ctx, func() {
		if done && c.OnDone != nil {
			c.OnDone()
		}
	}, done)

	if core.IsDebugMode() && c.Until.IsZero() {
		core.ReportConcern(ConcernCountdownUntilUnset,
			"Countdown has no Until, so it is already finished: it draws 0:00 and fires OnDone on its first pass")
	}

	shown := ceilSecond(remaining)
	return timerText(ctx.Theme(), c.digits(shown), c.label(shown),
		c.Size, c.Color, c.Hidden, c.Style).Render(ctx)
}

// ticking answers the table in the type doc.
func (c Countdown) ticking(done bool) bool {
	switch {
	case done:
		// Nothing left to count and the crossing has already been reported.
		return false
	case !c.Hidden:
		return true
	default:
		// Out of sight, but somebody is waiting to hear that it ran out.
		return c.OnDone != nil
	}
}

// digits writes the reading.
func (c Countdown) digits(remaining time.Duration) string {
	if c.Format != nil {
		return c.Format(remaining)
	}
	return timerDigits(remaining)
}

// label is the sentence the widget announces.
func (c Countdown) label(remaining time.Duration) string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	if remaining <= 0 {
		// Not "0 seconds remaining", which is a reading of a number rather
		// than the fact the listener is waiting for.
		return "Time is up"
	}
	return spokenDuration(remaining) + " remaining"
}

// Stopwatch draws time elapsed, one tick a second, and can be paused and
// resumed without losing what it has counted.
//
//	comps.Stopwatch{Since: startedAt.Get(), Elapsed: banked.Get(), Running: running.Get()}
//
// It is Countdown counting the other way, and it is the simpler of the two:
// there is no deadline, so there is nothing to report and no OnDone.
//
// # The caller holds the two numbers, and why there are two
//
// A stopwatch that knew only when it started could not be paused: the instant
// the finger lifts is not recorded anywhere, so on the next render the widget
// would either keep counting or forget everything. So the state is the pair
// every stopwatch keeps — the time banked from earlier runs, and the start of
// the current one — and the reading is their sum:
//
//	Elapsed + (Running ? now − Since : 0)
//
// which makes the four moves assignments the caller can write inline:
//
//	start    Since = time.Now();  Elapsed = 0;                     Running = true
//	pause    Elapsed += time.Since(Since);                         Running = false
//	resume   Since = time.Now();                                   Running = true
//	reset    Elapsed = 0;                                          Running = false
//
// The pair lives with the caller rather than in the widget for the reason
// SliderRow's draft does: state held here would be state the app cannot save,
// restore or show anywhere else, and a stopwatch is exactly the thing an app
// wants to keep running across a screen change.
//
// # No hundredths
//
// Real stopwatches show hundredths and this one shows seconds, because a
// core.State change requests a render of the whole tree: a centisecond
// stopwatch would charge the app a hundred render passes a second to animate
// two digits. A lap timer that genuinely needs them wants a renderer-side
// clock, which is what core.Spin is for animation and what no core primitive
// offers for text.
//
// The reading is rounded *down*, the opposite of Countdown's rounding and for
// the same reason: a stopwatch never claims more elapsed time than has
// actually passed, so 0:00 covers the first second exactly as a phone's does.
//
// # Ticking
//
// hooks.UseIntervalWhile again, active while Running and not Hidden. There is
// no exception for a hidden one, because unlike Countdown it owes nobody a
// callback — a hidden stopwatch has nothing to do but keep its arithmetic,
// which is the caller's two fields and needs no ticks at all. It therefore
// holds a hook and is not conditional-safe; see Countdown.
//
// # Theme roles read
//
//	Digits   Colors.TextPrimary, unless Color says otherwise
type Stopwatch struct {
	// Since is when the current run started. Read only while Running.
	Since time.Time

	// Elapsed is time banked from earlier runs, and the whole reading while
	// paused. Zero for a stopwatch that has never been paused.
	Elapsed time.Duration

	// Running says whether the current run is counting.
	Running bool

	// Format writes the digits. It receives the elapsed time already
	// truncated to a whole second. Nil uses the phone-timer format: M:SS
	// under an hour, H:MM:SS at or over one.
	Format func(time.Duration) string

	// Size is the digits' font size in px; 0 means 40, as in DigitalClock.
	Size float64

	// Color inks the digits; empty uses the theme's TextPrimary.
	Color string

	// Hidden removes the widget from display and stops its tick.
	Hidden bool

	// AccessibilityLabel overrides the spoken elapsed time.
	AccessibilityLabel string

	// Style is applied last, to the digits.
	Style []core.StyleProp
}

// Render arms the tick and draws the digits.
func (s Stopwatch) Render(ctx *core.Context) *core.Node {
	// Read before the hook, as in Countdown, and once for the whole pass so
	// the digits and the spoken label cannot describe two different instants.
	elapsed := s.total()

	hooks.UseIntervalWhile(ctx, s.Running && !s.Hidden, timerTick, time.Second)

	if core.IsDebugMode() && s.Running && s.Since.IsZero() {
		core.ReportConcern(ConcernStopwatchSinceUnset,
			"Stopwatch is Running with no Since, so it is counting from the zero time")
	}

	// Truncate rather than round: see "No hundredths".
	shown := elapsed.Truncate(time.Second)
	return timerText(ctx.Theme(), s.digits(shown), s.label(shown),
		s.Size, s.Color, s.Hidden, s.Style).Render(ctx)
}

// total is the reading: banked time plus the current run.
func (s Stopwatch) total() time.Duration {
	d := s.Elapsed
	if s.Running {
		d += time.Since(s.Since)
	}
	if d < 0 {
		// A Since in the future, or a negative Elapsed. Neither is meaningful
		// and a negative reading would format as a nonsense one.
		return 0
	}
	return d
}

// digits writes the reading.
func (s Stopwatch) digits(elapsed time.Duration) string {
	if s.Format != nil {
		return s.Format(elapsed)
	}
	return timerDigits(elapsed)
}

// label is the sentence the widget announces.
func (s Stopwatch) label(elapsed time.Duration) string {
	if s.AccessibilityLabel != "" {
		return s.AccessibilityLabel
	}
	return spokenDuration(elapsed) + " elapsed"
}

// timerDigitSize is the default face size, shared with DigitalClock so a
// countdown and a clock on one screen are the same size without either
// caller saying so.
const timerDigitSize = 40

// timerTick is the pump's callback. It is empty because both widgets read the
// clock in their own Render: hooks.UseIntervalWhile calls fn and then requests
// a render, so an empty fn is precisely "come back and look again".
func timerTick() {}

// timerText is the one node both widgets draw. One node, because there is one
// thing to draw — DigitalClock needs a column only because it has an AM/PM
// marker and a date line to place, and neither exists here.
//
// The caller's Style goes last, so it beats every default above it, including
// the font size a Size field also sets.
func timerText(t *core.Theme, digits, label string, size float64, color string,
	hidden bool, style []core.StyleProp) core.View {

	if size <= 0 {
		size = timerDigitSize
	}
	if color == "" {
		color = t.Colors.TextPrimary
	}

	props := make([]core.StyleProp, 0, len(style)+6)
	props = append(props,
		core.FontSize(size),
		// Light, for DigitalClock's reason: a display-size numeral at Normal
		// or Bold weight reads as a heading rather than as a clock face.
		core.FontWeight(core.Light),
		core.TextColor(color),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(label),
	)
	if hidden {
		props = append(props, core.Display(core.DisplayNone))
	}
	props = append(props, style...)
	return core.Text(digits, props...)
}

// ceilSecond rounds a duration up to a whole second, and clamps a spent one
// at zero. Countdown rounds this way so its reading is never smaller than the
// time actually left; see "Why not hooks.UseNow".
func ceilSecond(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	if rem := d % time.Second; rem > 0 {
		return d + time.Second - rem
	}
	return d
}

// timerDigits writes a whole-second duration the way a phone's timer does:
// minutes and seconds, with the hours field appearing only when there are
// hours to show.
//
//	   0:07     under a minute — the minutes field is still there, because a
//	            bare "7" would read as seven of something unnamed
//	   9:05     minutes are not zero-padded; seconds always are
//	  59:59     the last reading before the hours field appears
//	1:00:00     and the first one after it
//	26:00:00    days are hours, so the field can only grow and the reading
//	            never needs a unit the format has not shown before
//
// Crossing an hour changes the string's width, which moves a centred widget's
// digits. That is what every phone timer does and is not worth a fixed-width
// format that would show "00:00:07" for a seven-second timer.
func timerDigits(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int64(d / time.Second)
	h, m, s := total/3600, (total%3600)/60, total%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// spokenDuration writes a duration as words, for the accessible label. Only
// the non-zero fields appear, so 2h05s is "2 hours 5 seconds" rather than
// "2 hours 0 minutes 5 seconds" — a listener counts the words they are given,
// and a zero field is one more word to discard.
//
// Anything under a second is "0 seconds": the widgets clamp and round before
// they get here, so this is reached only by a genuinely spent timer, and a
// reading of "" would leave the label as the bare " remaining".
func spokenDuration(d time.Duration) string {
	if d < time.Second {
		return "0 seconds"
	}
	total := int64(d / time.Second)
	parts := make([]string, 0, 3)
	if h := total / 3600; h > 0 {
		parts = append(parts, spokenCount(h, "hour"))
	}
	if m := (total % 3600) / 60; m > 0 {
		parts = append(parts, spokenCount(m, "minute"))
	}
	if s := total % 60; s > 0 {
		parts = append(parts, spokenCount(s, "second"))
	}
	return strings.Join(parts, " ")
}

// spokenCount writes one field of spokenDuration, pluralised.
func spokenCount(n int64, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return strconv.FormatInt(n, 10) + " " + unit + "s"
}
