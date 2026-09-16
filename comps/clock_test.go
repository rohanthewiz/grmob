package comps

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

func renderView(t *testing.T, v core.View) *core.Node {
	t.Helper()
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	return v.Render(ctx)
}

// at is a fixed local instant, so the tests do not depend on the machine's zone.
func at(h, m, s int) time.Time {
	return time.Date(2026, 9, 16, h, m, s, 0, time.UTC)
}

func TestClockDigitsFormats(t *testing.T) {
	cases := []struct {
		tm              time.Time
		hour24, seconds bool
		digits, marker  string
	}{
		{at(22, 42, 7), false, false, "10:42", "PM"},
		{at(9, 5, 0), false, true, "9:05:00", "AM"},
		{at(9, 5, 0), true, false, "09:05", ""},
		{at(0, 0, 0), false, false, "12:00", "AM"},
	}
	for _, c := range cases {
		d, m := clockDigits(c.tm, c.hour24, c.seconds)
		if d != c.digits || m != c.marker {
			t.Errorf("clockDigits(%v, 24h=%v, sec=%v) = %q %q, want %q %q",
				c.tm.Format(time.TimeOnly), c.hour24, c.seconds, d, m, c.digits, c.marker)
		}
	}
}

// The widget speaks once, as a sentence, and its parts are hidden.
func TestDigitalClockIsOneAccessibleElement(t *testing.T) {
	n := renderView(t, DigitalClock{Time: at(22, 42, 7), ShowSeconds: true, ShowDate: true})
	if got := n.Style.AccessibilityLabel; got != "10:42:07 PM, Wednesday, September 16" {
		t.Errorf("label = %q", got)
	}
	if n.Style.AccessibilityRole != core.RoleImg {
		t.Errorf("role = %q, want img", n.Style.AccessibilityRole)
	}
	for _, s := range []string{"10:42:07", "PM", "Wednesday, September 16"} {
		if findText(n, s) == nil {
			t.Errorf("text %q missing", s)
		}
	}
}

func TestDigitalClock24HourHasNoMarker(t *testing.T) {
	n := renderView(t, DigitalClock{Time: at(22, 42, 7), Hour24: true})
	if findText(n, "22:42") == nil || findText(n, "PM") != nil {
		t.Errorf("24-hour clock drew the wrong parts")
	}
}

// The angles are unwrapped across the day, so a Transition never sweeps back
// at the top of a minute or hour.
func TestClockAnglesAreUnwrapped(t *testing.T) {
	h, m, s := clockAngles(at(15, 30, 0))
	if h != 465 || m != 5580 || s != 334800 {
		t.Errorf("15:30:00 → %g %g %g", h, m, s)
	}
	_, m1, s1 := clockAngles(at(10, 0, 59))
	_, m2, s2 := clockAngles(at(10, 1, 0))
	if s2-s1 != 6 {
		t.Errorf("second hand moved %g° across a minute boundary, want 6", s2-s1)
	}
	if abs(m2-m1-0.1) > 1e-9 {
		t.Errorf("minute hand moved %g° in a second, want 0.1", m2-m1)
	}
}

// handRotations collects the layers that carry a hand: a two-child column
// (spacer, stick) whose stick is rounded. Ticks are square, so they are skipped.
func handRotations(n *core.Node) []*core.Node {
	var out []*core.Node
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n.Type == "Column" && len(n.Children) == 2 {
			stick := n.Children[1]
			if stick.Style != nil && stick.Style.BorderRadius > 0 {
				out = append(out, n)
				return
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

func TestAnalogClockHands(t *testing.T) {
	n := renderView(t, AnalogClock{Time: at(3, 0, 30), ShowSeconds: true, Smooth: true})
	hands := handRotations(n)
	if len(hands) != 3 {
		t.Fatalf("found %d hands, want 3", len(hands))
	}
	want := []float64{10830.0 / 120, 1083, 64980}
	for i, h := range hands {
		if h.Style.Rotate != want[i] {
			t.Errorf("hand %d rotated %g, want %g", i, h.Style.Rotate, want[i])
		}
		if h.Style.Transition == "" {
			t.Errorf("hand %d has no Transition under Smooth", i)
		}
	}
	if got := n.Style.AccessibilityLabel; got != "3:00:30 AM" {
		t.Errorf("label = %q", got)
	}

	if len(handRotations(renderView(t, AnalogClock{Time: at(3, 0, 30)}))) != 2 {
		t.Error("second hand drawn without ShowSeconds")
	}
}

// The pass at midnight drops the Transition, so the hands jump to zero rather
// than unwinding a day's worth of turns.
func TestAnalogClockMidnightJumps(t *testing.T) {
	for _, h := range handRotations(renderView(t, AnalogClock{Time: at(0, 0, 0), ShowSeconds: true, Smooth: true})) {
		if h.Style.Transition != "" {
			t.Errorf("hand animates across midnight: %q", h.Style.Transition)
		}
	}
}

// A hand's stick ends at the centre plus its tail: spacer + stick = size/2 + tail.
func TestAnalogClockHandsPivotAtTheCentre(t *testing.T) {
	const size = 200
	n := renderView(t, AnalogClock{Time: at(3, 0, 30), ShowSeconds: true, Size: size})
	tails := []float64{0, 0, size * clockSecondTailLen}
	for i, h := range handRotations(n) {
		sum := parsePx(t, h.Children[0].Style.Height) + parsePx(t, h.Children[1].Style.Height)
		if want := size/2 + tails[i]; abs(sum-want) > 1e-9 {
			t.Errorf("hand %d reaches %gpx down its layer, want %g", i, sum, want)
		}
	}
}

func parsePx(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 64)
	if err != nil || !strings.HasSuffix(s, "px") {
		t.Fatalf("not a px length: %q", s)
	}
	return v
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
