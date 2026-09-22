package comps

import (
	"math"

	"github.com/rohanthewiz/grmob/core"
)

// Waveform draws a recording's loudness over time as a row of bars mirrored
// about a centre line, with the part already played in the accent colour: the
// strip a voice message or a podcast player shows.
//
// The peaks come from the caller. No host decodes audio for Go, so a
// server or a build step computes them (one number per slice of the
// recording, its loudest sample, scaled to 0–1) and ships them beside the
// file. Without peaks there is no waveform to draw, and this widget does not
// invent one.
//
//	comps.Waveform{Peaks: msg.Peaks, Progress: position / duration}
//
// # Bars are strokes, and that is what keeps them round
//
//	    ╷   ┃ ╷                 one vertical line per bar, stroked BarWidth
//	  ╷ ┃ ╷ ┃ ┃ ╷   ╷           px wide with round caps, mirrored about the
//	──┃─┃─┃─┃─┃─┃─╷─┃─╷──       centre: from mid − peak·reach to mid + peak·reach
//	  ╵ ┃ ╵ ┃ ┃ ╵   ╵
//	    ╵   ┃ ╵
//	 ◀─ played ─▶◀─ rest ─▶
//
// The drawing is stretched across whatever width the layout gives
// (core.CanvasStretch), and a rounded rectangle drawn as a path would stretch
// with it, its corners turning to ellipses. A Canvas stroke is never scaled:
// its width is layout px on every target (see core.Canvas). So a bar is a
// line, its thickness is the stroke's, and its rounded ends are the stroke's
// round caps, all exactly BarWidth px however wide the strip is drawn. Only
// the bars' spacing stretches, which is the part that should.
//
// The viewBox is Height units tall and the canvas Height px, so y is one to
// one and the caps' half a BarWidth of overshoot can be taken off the line's
// reach exactly: no bar is cut by the canvas's edge.
//
// Two shapes carry everything, the played bars and the rest, each one path.
// Progress moving only shifts subpaths from one to the other.
//
// # More peaks than bars
//
// Bars says how many bars to draw. With more peaks than that, each bar takes
// the maximum of its bucket, never the mean: a clap in a quiet passage is one
// sample wide, and averaging would erase the one feature a listener scans the
// strip for. With fewer peaks than Bars, there is a bar per peak.
//
// Go cannot measure the strip, so how many bars suit it is the caller's
// arithmetic, from the width it knows (hooks.UseWindow less its own insets).
// BarsFor does it, as CalendarHeatmap.WeeksFor does for weeks:
//
//	w := comps.Waveform{Peaks: peaks, Progress: p}
//	w.Bars = w.BarsFor(win.Width - 32)
//
// # Display only
//
// Tapping a waveform to seek needs the tap's x position, which no event
// carries to Go. So this is a picture of progress and not a control, and
// AudioPlayer, which offers it through its Waveform field, keeps its slider
// for seeking.
//
// # Accessibility
//
// One image: "Audio waveform, 40 percent played". AccessibilityLabel replaces
// the sentence. Beside a seek bar that already speaks the position, as in
// AudioPlayer, it is decoration, and Decorative hides it.
//
// # Theme roles read
//
//	Played   Colors.Primary
//	Rest     Colors.ControlBorderColor()
type Waveform struct {
	// Peaks are the loudness samples in time order, 0 (silence) to 1 (full
	// scale). Values outside that are clamped; NaN reads as silence.
	Peaks []float64

	// Progress is how much has played, 0 to 1. A bar is played once its
	// centre is behind the playhead.
	Progress float64

	// Bars is how many bars to draw; 0 means one per peak, up to 56. See
	// "More peaks than bars" and BarsFor.
	Bars int

	// Height is the strip's height in px; 0 means 48.
	Height float64

	// BarWidth is each bar's thickness in px (0 means 3), and Gap the space
	// BarsFor leaves between bars (0 means 2). Gap is only BarsFor's: drawn
	// bars are spread evenly over the width the layout gives.
	BarWidth float64
	Gap      float64

	// PlayedColor and RestColor ink the bars behind and ahead of the
	// playhead; empty means the theme's Primary and its control border.
	PlayedColor string
	RestColor   string

	// Decorative hides the strip from screen readers, for a waveform beside
	// a control that already speaks the position.
	Decorative bool

	// Style is applied last, to the canvas.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated sentence.
	AccessibilityLabel string
}

const (
	waveformHeight   = 48.0
	waveformBarWidth = 3.0
	waveformGap      = 2.0

	// waveformMaxBars caps the default bar count. 56 bars at the default
	// pitch are 280 px, which fits the narrowest phone's content width, so
	// an unset Bars never packs the strokes into one solid band.
	waveformMaxBars = 56

	// waveformMinReach is a silent bar's half-length in px: with the round
	// caps it draws a dot, so silence reads as a quiet stretch of the
	// recording and not as a hole in the strip. Not zero, because a
	// zero-length subpath's caps are drawn by SVG and dropped by some
	// platform path APIs.
	waveformMinReach = 0.5
)

// BarsFor is how many bars fit width px at BarWidth and Gap, at least 1.
// n bars take n·BarWidth + (n−1)·Gap.
func (w Waveform) BarsFor(width float64) int {
	bw, gap := orDefaultFloat(w.BarWidth, waveformBarWidth), orDefaultFloat(w.Gap, waveformGap)
	if math.IsNaN(width) || width <= bw {
		return 1
	}
	return max(1, int((width+gap)/(bw+gap)))
}

func (w Waveform) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	h := orDefaultFloat(w.Height, waveformHeight)
	bw := orDefaultFloat(w.BarWidth, waveformBarWidth)
	played := orDefault(w.PlayedColor, t.Colors.Primary)
	rest := orDefault(w.RestColor, t.Colors.ControlBorderColor())

	progress := w.Progress
	if math.IsNaN(progress) {
		progress = 0
	}
	progress = math.Max(0, math.Min(1, progress))

	bars := w.bars()
	n := len(bars)

	// The viewBox is n units wide, a unit per bar, so bar i's centre is
	// i + ½ whatever the drawn width; and h units tall, so y is in px.
	vw := float64(max(n, 1))
	mid := h / 2
	// The longest half-length that keeps the round cap inside the canvas.
	reach := math.Max(waveformMinReach, mid-bw/2)

	paths := [2]*core.Path{core.NewPath(), core.NewPath()}
	for i, peak := range bars {
		x := float64(i) + 0.5
		half := math.Max(waveformMinReach, peak*reach)
		kind := 1
		if x/vw <= progress {
			kind = 0
		}
		paths[kind].MoveTo(x, mid-half).LineTo(x, mid+half)
	}

	props := []core.PropsAndChildren{
		core.CanvasStretch,
		// Played time runs in reading order, as the seek slider beside it
		// does on every target under RTL.
		core.CanvasMirrorsRTL,
		core.Width("100%"),
		core.Height(px(h)),
	}
	if !w.Decorative {
		// A Canvas with no label hides itself, which is what Decorative
		// wants; with one it is an image (see core.Canvas).
		props = append(props, core.AccessibilityLabel(w.label(progress)))
	}
	props = append(props, asProps(w.Style)...)

	return core.Canvas(vw, h, []core.Shape{
		{Path: paths[0], Stroke: played, StrokeWidth: bw, Cap: core.CapRound},
		{Path: paths[1], Stroke: rest, StrokeWidth: bw, Cap: core.CapRound},
	}, props...).Render(ctx)
}

// bars is the peaks cut down to the bar count by bucket maximum, each
// clamped to 0–1. See "More peaks than bars".
func (w Waveform) bars() []float64 {
	n := len(w.Peaks)
	count := w.Bars
	if count <= 0 {
		count = waveformMaxBars
	}
	count = min(count, n)
	out := make([]float64, count)
	for i := range out {
		// Bucket i is peaks [i·n/count, (i+1)·n/count): integer arithmetic,
		// so the buckets tile the peaks with no gap and no overlap, and none
		// is empty because count ≤ n.
		lo, hi := i*n/count, (i+1)*n/count
		for _, v := range w.Peaks[lo:hi] {
			if math.IsNaN(v) {
				continue
			}
			out[i] = math.Max(out[i], math.Min(1, v))
		}
	}
	return out
}

func (w Waveform) label(progress float64) string {
	if w.AccessibilityLabel != "" {
		return w.AccessibilityLabel
	}
	if len(w.Peaks) == 0 {
		return "Audio waveform, no data"
	}
	return "Audio waveform, " + formatValue(math.Round(progress*100)) + " percent played"
}
