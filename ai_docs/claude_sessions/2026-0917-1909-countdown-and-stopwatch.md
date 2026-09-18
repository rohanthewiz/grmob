# D3: Countdown and Stopwatch, and a tick with nothing to do

**Session:** bdb32502-39ac-42d6-bee0-b1ed4d8817b6
**Date:** 2026-09-17 19:09 (follows "select-row-and-slider-row")
**Branch:** master (34ee0e6 → this commit)

## 1. The ask

"start D3" — the next item in `ai_docs/plans/comps-low-hanging-fruit-2.md`,
fourth in the plan's own order, and the first one whose widgets have to know
what time it is.

## 2. What the family looked like before

`comps/clock.go` holds `DigitalClock` and `AnalogClock`, and its doc spends a
section on why neither of them ticks: a widget that called `hooks.UseNow`
itself would be a hook caller, could not be tested at a fixed instant, and
could not show anything but "now here" — a world clock wants a different time.
So the clocks take a `time.Time` and hold nothing.

D3 is the two widgets that cannot take that deal. A countdown's whole subject
is the distance between now and something else, so one of the two ends has to
be read rather than handed over. That makes them the first hook-owning widgets
in the display half of the package, and everything below follows from it.

## 3. `Countdown`

```go
comps.Countdown{Until: expiresAt, OnDone: func() { code.Set("") }}
```

One `core.Text` in `DigitalClock`'s face — `RoleImg` with a spoken label, for
the reason that doc already gives: read as text, "4:12" is punctuation.

**The tick is `hooks.UseIntervalWhile` with an empty callback**, and the two
alternatives are both worth writing down because both looked better first.

`hooks.UseNow` is the obvious one, and the plan's sketch named it. It aligns
its ticks to wall-clock boundaries, which is exactly right for a clock — a
`DigitalClock` changes its seconds digit when the phone's status bar does. A
countdown has no such phase to share: its own boundaries fall at `Until` minus
a whole number of seconds, which nothing else on the screen is on. There being
nothing to align to, the more expensive hook buys nothing. And `UseNow` cannot
be paused, which the plan's own "the tick pauses while hidden or done"
requires.

Adding `hooks.UseNowWhile` — the pausable `UseNow`, as `UseIntervalWhile` is
to `UseInterval` — was the second candidate and was dropped for the same
reason: it would have been a new public hook whose only advantage over the
existing one is an alignment these widgets have no use for.

So the tick is the pump and nothing else. `UseIntervalWhile` calls `fn` and
then requests a render, so an empty `fn` is precisely "come back and look
again", and the widget reads the clock in its own `Render`.

**The clock is read before the hooks, not by them.** Whether to keep ticking
and whether the deadline has passed are both answers about the same instant,
and the hook cannot both report the instant and be told, in the same call,
what the instant means. One `time.Until` at the top of `Render` settles it,
and the digits and the spoken label cannot then describe two different
instants.

**"Pauses while hidden" needed an exception, and it is the interesting one.**
The tick has two reasons to exist: there is something to draw, and there is
somebody waiting to be told it ran out. Hiding the widget removes the first
and not the second. So:

| state | ticking |
| --- | --- |
| counting, visible | yes |
| counting, `Hidden` | only if `OnDone` is set |
| finished | no |
| `Hidden` with no `OnDone` | no |

A hidden countdown that owes an `OnDone` keeps counting; one that owes nothing
stops dead. There is no way to observe the pump's pause from a rendered tree,
so `ticking` is a method and a table test is what pins the table.

**`OnDone` comes from the effect**, which was the plan's one stated decision,
and the deps are what make it more than "not from the render pass":

```
remaining  5s ──── 4s ──── … ──── 1s ──── 0 ──── 0 ──── 0
deps       false   false         false   true   true   true
OnDone      ·       ·             ·      fire    ·      ·
```

Keyed on the *crossing* rather than on the widget, which settles two things
the sketch did not ask about. A `Countdown` whose `Until` is already past on
its first pass fires immediately — the correct reading of a deadline restored
from disk while the app was closed, and the reason a zero `Until` is a concern
rather than a quiet no-op. And moving `Until` forward re-arms it, so a restart
is `Until: time.Now().Add(d)` and nothing else; the widget needs no reset call
and has no reset method.

The doc says plainly what it is not: a display-grade signal, late by up to one
tick, firing only while the widget is rendered. Something that must happen
whether or not anyone is looking belongs in `alarm` and `hooks.UseAlarms`.

## 4. `Stopwatch`, and the field the sketch did not have

```go
comps.Stopwatch{Since: startedAt.Get(), Elapsed: banked.Get(), Running: running.Get()}
```

The plan wrote `Stopwatch{Since time.Time, Running bool, ...}`, and `Since`
alone cannot be paused: the instant the finger lifts is recorded nowhere, so
on the next render the widget would either keep counting or forget everything.
The state is therefore the pair every stopwatch keeps — time banked from
earlier runs, and the start of the current one — and the reading is their sum,
which makes the four moves assignments the caller writes inline:

```go
start   since.Set(time.Now());                                running.Set(true)
pause   banked.Set(banked.Get() + time.Since(since.Get()));   running.Set(false)
resume  since.Set(time.Now());                                running.Set(true)
reset   banked.Set(0);                                        running.Set(false)
```

Held in the widget it would be state the app cannot save, restore or show
anywhere else, and a running stopwatch is exactly the thing an app wants to
keep across a screen change. That is `SliderRow`'s argument from last session
arriving at a different widget, which is why the doc points at it.

No exception to the pause rule here, and the asymmetry with `Countdown` is the
point: a stopwatch owes nobody a callback, so a hidden one has nothing to do
but keep arithmetic that is the caller's two fields and needs no ticks at all.

## 5. The two roundings

Opposite directions, both conservative, and each one is a claim a test pins:

- **`Countdown` rounds up.** The deadline is noticed on the first tick at or
  after it, so the digits reach 0:00 up to a second late. Rounding up makes
  that lag conservative rather than arbitrary — a countdown never tells you
  that you have less time left than you do.
- **`Stopwatch` truncates.** It never claims more elapsed time than has
  passed, so its first second reads 0:00, as a phone's does.

Neither shows hundredths. A `core.State` change requests a render of the whole
tree, so a centisecond stopwatch would charge the app a hundred passes a
second to animate two digits.

`Format` receives the duration *already rounded the widget's way*, so a custom
format cannot disagree with the widget about which second is on screen. The
default is the phone timer's: `M:SS` under an hour, `H:MM:SS` at or over one,
days folded into hours (`26:00:00`), so the field can only grow and the
reading never needs a unit the format has not already shown.

## 6. Two concerns

Both are states that are permanently wrong and look ordinary on screen, which
is the bar `ConcernSelectRowValueNotAnOption` set last session.

- `ConcernCountdownUntilUnset` — a zero `Until` is a countdown that is already
  spent: it draws 0:00 and fires `OnDone` on its first pass, which is exactly
  what a timer that just finished looks like.
- `ConcernStopwatchSinceUnset` — `Running` with a zero `Since` counts from
  year 1. Visibly absurd in the digits, silently wrong in the label.

## 7. Lesson 4.24

`examples/tutorial/chapter4.go`, appended for the reason 4.15, 4.22 and 4.23
were — lesson numbers in deep links do not move. Chapter 4 rather than
anywhere else because 4.19 is where the clocks are taught.

The demo runs both widgets at once, because the interesting thing is the
difference: one holds a deadline and reports, the other holds nothing and
reports nothing. A "Ran out N times" caption makes `OnDone`'s once-per-crossing
visible rather than asserted.

`TestTimersLessonRestartsTheDeadlineAndBanksTheStopwatch` drives the two
claims a tree can answer without waiting out ten real seconds: moving `Until`
forward *is* the restart (the reading returns to 0:10 with no reset call
anywhere), and the stopwatch's controls are the caller's three assignments.
The crossing itself is pinned in `comps`, against a deadline a test controls.

## 8. Fallout

- `docs/components.md`: `## Countdown & Stopwatch` between Compass and QRCode.
  `docs/api/` regenerated; `timers.go` registered under "Data display & maps".
- "0 of 63 lessons opened" → 64 in README.md, docs/tutorial-interactive.md,
  `examples/tutorial/screenshot_test.go`, `internal/shotclaims/*`;
  `docs/images/tutorial-contents.png` re-taken with `wasm/shots/shoot.sh` and
  read back to confirm it says 64.
- The file census in `wasm/verify` went 564 → 566 across five sentences, the
  two new `comps` files.
- The plan marked D3 landed, with the four decisions written up.

18 new tests in `comps`, one in the tutorial; full suite, `go vet` and
`gofmt` green.

## 9. Not verified

The two natives. A single `Text` node carrying `RoleImg`, a label and a font
size is the shape `DigitalClock` already ships inside its column, so the risk
is low — but neither timer has been seen off the DOM, and nothing here was run
on a device.

## 10. Next

D5 `PINInput`, D6 `DateRangePicker` and D7 `TimePicker`, in any order per the
plan. `PINInput` carries the two decisions already settled in the sketch: a
pasted full code arrives in the first cell's `OnChange` as a multi-character
string and is spread across the cells, and a backspace on an empty cell cannot
be seen because there are no key events, so a cleared cell stays focused.
