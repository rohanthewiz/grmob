package comps

import (
	"math"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestWaveformIsAnImageThatSaysItsProgress(t *testing.T) {
	_, n := renderDebug(t, Waveform{Peaks: []float64{0.2, 0.8, 0.5, 1, 0.1}, Progress: 0.4})
	if n.Type != "Canvas" || n.Style.AccessibilityRole != core.RoleImg {
		t.Fatalf("want a Canvas image, got %s role %q", n.Type, n.Style.AccessibilityRole)
	}
	if want := "Audio waveform, 40 percent played"; n.Style.AccessibilityLabel != want {
		t.Errorf("label = %q, want %q", n.Style.AccessibilityLabel, want)
	}
	if len(n.Children) != 2 {
		t.Fatalf("shapes = %d, want played and rest", len(n.Children))
	}
	played, rest := n.Children[0], n.Children[1]
	if played.Props["stroke"] != core.DefaultTheme.Colors.Primary {
		t.Errorf("played stroke = %v, want Primary", played.Props["stroke"])
	}
	// Bar centres are at 10%, 30%, … of the width: two are behind 40%.
	if p, r := subpaths(played), subpaths(rest); p != 2 || r != 3 {
		t.Errorf("played %d + rest %d bars, want 2 + 3", p, r)
	}
	for _, s := range n.Children {
		if s.Props["cap"] != string(core.CapRound) || s.Props["strokeWidth"] != waveformBarWidth {
			t.Errorf("bars are %v px strokes with %v caps, want %v and round",
				s.Props["strokeWidth"], s.Props["cap"], waveformBarWidth)
		}
	}
}

// A bar is mirrored about the centre line, and the tallest stops half a
// stroke short of the canvas's edge so its round cap is not cut.
func TestWaveformBarsAreMirroredAndKeepTheirCaps(t *testing.T) {
	_, n := renderDebug(t, Waveform{Peaks: []float64{1, 0.5, 0, math.NaN(), 7}, Height: 40, BarWidth: 4})
	ys := pathYs(n.Children[1])
	reach := 40.0/2 - 4.0/2
	want := []float64{
		20 - reach, 20 + reach, // full scale
		20 - reach/2, 20 + reach/2, // half
		20 - waveformMinReach, 20 + waveformMinReach, // silence: a dot
		20 - waveformMinReach, 20 + waveformMinReach, // NaN reads as silence
		20 - reach, 20 + reach, // over 1 is clamped
	}
	if len(ys) != len(want) {
		t.Fatalf("got %d y values, want %d", len(ys), len(want))
	}
	for i := range want {
		if math.Abs(ys[i]-want[i]) > 0.011 {
			t.Errorf("y[%d] = %v, want %v", i, ys[i], want[i])
		}
	}
}

// Downsampling keeps each bucket's maximum, so a one-sample transient
// survives where a mean would have flattened it.
func TestWaveformDownsamplesByBucketMaximum(t *testing.T) {
	peaks := make([]float64, 100)
	for i := range peaks {
		peaks[i] = 0.1
	}
	peaks[37] = 1
	bars := Waveform{Peaks: peaks, Bars: 10}.bars()
	if len(bars) != 10 {
		t.Fatalf("bars = %d, want 10", len(bars))
	}
	for i, v := range bars {
		want := 0.1
		if i == 3 {
			want = 1
		}
		if v != want {
			t.Errorf("bar %d = %v, want %v", i, v, want)
		}
	}
	// Buckets of unequal size still tile every peak: the last one's spike
	// is in the last bar.
	odd := make([]float64, 7)
	odd[6] = 0.9
	if got := (Waveform{Peaks: odd, Bars: 3}).bars(); got[2] != 0.9 {
		t.Errorf("uneven buckets = %v, want the spike in the last", got)
	}
	// Fewer peaks than Bars: a bar per peak. Unset: capped.
	if got := len(Waveform{Peaks: odd, Bars: 50}.bars()); got != 7 {
		t.Errorf("bars = %d, want one per peak", got)
	}
	if got := len(Waveform{Peaks: peaks}.bars()); got != waveformMaxBars {
		t.Errorf("default bars = %d, want the cap of %d", got, waveformMaxBars)
	}
}

func TestWaveformBarsFor(t *testing.T) {
	w := Waveform{}
	// n bars take 3n + 2(n−1): 58 bars are 288 px and 59 are 293.
	for _, c := range []struct {
		width float64
		want  int
	}{{290, 58}, {293, 59}, {3, 1}, {0, 1}, {-5, 1}, {math.NaN(), 1}} {
		if got := w.BarsFor(c.width); got != c.want {
			t.Errorf("BarsFor(%v) = %d, want %d", c.width, got, c.want)
		}
	}
	if got := (Waveform{BarWidth: 4, Gap: 4}).BarsFor(100); got != 13 {
		t.Errorf("BarsFor(100) at 4 + 4 = %d, want 13", got)
	}
}

// Decorative, empty and out-of-range inputs.
func TestWaveformEdges(t *testing.T) {
	_, n := renderDebug(t, Waveform{Peaks: []float64{0.5}, Progress: 3, Decorative: true})
	if !n.Style.AccessibilityHidden || n.Style.AccessibilityLabel != "" {
		t.Error("Decorative should leave the canvas unlabelled and hidden")
	}
	if subpaths(n.Children[0]) != 1 {
		t.Error("Progress over 1 is all played")
	}
	_, n = renderDebug(t, Waveform{})
	if n.Style.AccessibilityLabel != "Audio waveform, no data" {
		t.Errorf("empty label = %q", n.Style.AccessibilityLabel)
	}
	_, n = renderDebug(t, Waveform{Peaks: []float64{1}, AccessibilityLabel: "Voice message"})
	if n.Style.AccessibilityLabel != "Voice message" {
		t.Errorf("label = %q", n.Style.AccessibilityLabel)
	}
}

// AudioPlayer draws its peaks above the seek bar, filled to the position and
// silent to a screen reader, and draws nothing without them.
func TestAudioPlayerWaveform(t *testing.T) {
	status := core.AudioStatus{Track: sermonTrack, State: core.AudioPlaying, Position: 925, Duration: 3700, Rate: 1}
	h := newAudioHarness(t, AudioPlayer{Track: sermonTrack, Waveform: []float64{0.2, 0.9, 0.4, 0.6}}, status)
	canvas := canvasOf(h.node)
	if canvas == nil {
		t.Fatal("no waveform drawn")
	}
	if !canvas.Style.AccessibilityHidden {
		t.Error("the slider speaks the position; the strip should be hidden")
	}
	// A quarter played: bar centres at 12.5%, 37.5%, … so one is behind.
	if got := subpaths(canvas.Children[0]); got != 1 {
		t.Errorf("played bars = %d, want 1", got)
	}

	h = newAudioHarness(t, AudioPlayer{Track: sermonTrack}, status)
	if canvasOf(h.node) != nil {
		t.Error("no peaks, no waveform")
	}
}
