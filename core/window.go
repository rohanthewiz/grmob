package core

import (
	"math"
	"sync"
)

// Window metrics: how much room the app has, and whether a fold runs through
// it. This is what lets one tree serve a phone, a tablet and a foldable that
// is both at different moments.
//
// A foldable changes shape under a running app in three ways a fixed-size
// phone never does, and each needs something different from the framework:
//
//	unfold / fold      the window jumps from ~360dp wide to ~840dp or back —
//	                   a size change, answered by size classes
//	half-open          the hinge bends: the device stands like a laptop
//	                   (tabletop) or is held like a book — a posture change,
//	                   answered by Posture
//	hinge / seam       a physical line crosses the window that content
//	                   should not straddle — a geometry fact, answered by
//	                   Fold.Bounds (and laid out by comps.TwoPane)
//
// # Where the numbers come from
//
//	Android   Jetpack WindowManager: WindowMetricsCalculator for the size,
//	          WindowInfoTracker's FoldingFeature for the fold
//	iOS       the root view's size (Split View and Stage Manager resize an
//	          iPad window); no iPhone or iPad folds, so no fold ever
//	Browser   innerWidth/innerHeight; the Viewport Segments API
//	          (window.viewport.segments) for the seam and the Device
//	          Posture API (navigator.devicePosture) for folded vs flat
//	Headless  nothing — the record stays zero and Received stays false
//
// Every length is in the unit the tree is laid out in — dp on Android,
// points on iOS, CSS pixels in a browser — and in *window* coordinates, with
// the origin at the window's top-left corner including the area under the
// system bars. That is the space Android's FoldingFeature reports in and the
// one the viewport segments report in, so the shells convert units and
// nothing else. A component that is not at the window's origin has to know
// its own offset to line up with the hinge; comps.TwoPane takes it as Origin.
//
// # How it arrives
//
// The "window" host event, consumed here the way lifecycle.go consumes
// "lifecycle": the record is updated, then app subscribers to the raw event
// run. Apps read CurrentWindow, subscribe with OnWindow, or call
// hooks.UseWindow from a component.
//
// Hosts report whenever their platform says the window changed, and on
// Android that is often: every Activity recreation (a fold is a
// configuration change) re-emits the same layout. The record dedupes by value
// so subscribers hear changes only.
//
// # Why a record and not a layout primitive
//
// The fold could have been a node the natives lay out themselves, which
// would know its own on-screen position and need no Origin. It is a record
// instead because the decision a foldable forces is almost never "put a gap
// here" — it is "show the list and the detail side by side now", "move the
// video above the hinge", "switch the nav rail on". Those are tree changes,
// which is what Go components already do on a state change, and a record
// every component can read keeps them in Go rather than spread across four
// renderers.

// SizeClass buckets a window dimension into Material's three window size
// classes. The buckets, not the raw width, are what a layout should branch on:
// a Z Fold's inner screen and a small tablet differ by 100dp and want the same
// layout, and a breakpoint shared across apps is one users learn.
type SizeClass string

const (
	// SizeCompact is a phone held upright, or a folded foldable.
	SizeCompact SizeClass = "compact"
	// SizeMedium is an unfolded book-style foldable or a small tablet
	// upright: room for a list beside a narrow detail, not for three panes.
	SizeMedium SizeClass = "medium"
	// SizeExpanded is a tablet in landscape, a desktop browser, or a
	// foldable unfolded in landscape.
	SizeExpanded SizeClass = "expanded"
)

// The breakpoints, in dp, from Material 3's window size classes. Width and
// height use different ones: phones are tall, so a height of 480dp is already
// a landscape phone, while a width of 600dp is already more than any phone.
const (
	widthMediumMin    = 600
	widthExpandedMin  = 840
	heightMediumMin   = 480
	heightExpandedMin = 900
)

// FoldState is how far the hinge is bent.
type FoldState string

const (
	// FoldFlat is fully open — 180°. A hinge that is flat can still separate
	// content (a dual-screen device's seam), which is why Separating is its
	// own field.
	FoldFlat FoldState = "flat"
	// FoldHalfOpened is bent partway, somewhere around a right angle: the
	// device is standing on its own or being held like a book.
	FoldHalfOpened FoldState = "half_opened"
)

// FoldOrientation is the direction the hinge *line* runs across the window.
// A vertical hinge splits the window into left and right; a horizontal one
// into top and bottom.
type FoldOrientation string

const (
	FoldVertical   FoldOrientation = "vertical"
	FoldHorizontal FoldOrientation = "horizontal"
)

// Posture names the two half-opened shapes a layout designs for, derived from
// FoldState and FoldOrientation rather than reported, because every platform
// reports the two underlying facts and none reports these words.
type Posture string

const (
	// PostureNormal is everything that is not half-opened: a phone, a
	// tablet, a foldable fully open or fully closed.
	PostureNormal Posture = "normal"
	// PostureTabletop is half-opened with a horizontal hinge — the device
	// stands like a small laptop. Content goes above the hinge, controls
	// below it.
	PostureTabletop Posture = "tabletop"
	// PostureBook is half-opened with a vertical hinge — held like an open
	// book. Two pages, one each side.
	PostureBook Posture = "book"
)

// WindowRect is an axis-aligned rectangle in window coordinates. Not "Rect":
// that name is the canvas shape constructor (canvas.go).
type WindowRect struct {
	X, Y, Width, Height float64
}

// Fold is one hinge or seam crossing the window.
type Fold struct {
	State       FoldState
	Orientation FoldOrientation

	// Separating reports whether the platform thinks content should not
	// straddle the fold: always for a half-opened hinge, and for a flat
	// one only when it is a physical seam between two screens. A flat
	// Galaxy Fold's inner display is one continuous panel, so its fold is
	// not separating and a layout may run straight across it.
	Separating bool

	// Occluding reports whether the fold hides pixels — a seam between two
	// panels with real width, like the Surface Duo's. Nothing should be
	// drawn in Bounds when it is true. A folding OLED panel hides nothing and
	// reports a zero-width Bounds on the fold line.
	Occluding bool

	// Bounds is the fold's rectangle in window coordinates. For a
	// non-occluding hinge one dimension is zero: it is a line, and the value
	// that matters is where it sits (X for a vertical hinge, Y for a
	// horizontal one).
	Bounds WindowRect
}

// Window is the last report of the app window's size and fold.
type Window struct {
	// Width and Height are the window's size in layout units (see the file
	// comment). Zero before the host reports.
	Width, Height float64

	// HasFold reports whether a fold crosses the window. A foldable that is
	// folded shut reports none — the outer screen is a plain phone — and so
	// does an app in a split-screen half that the hinge does not cross.
	//
	// A bool beside a value rather than a *Fold so that Window stays
	// comparable with ==, which is what lets the record dedupe repeats.
	HasFold bool
	Fold    Fold

	// Received is true once any host has reported. Before that the size is
	// unknown rather than zero, and WidthClass answers compact — a phone is
	// the safest layout to draw into a window of unknown size.
	Received bool
}

// WidthClass is the window's width bucketed into a SizeClass. See the
// breakpoint constants above.
func (w Window) WidthClass() SizeClass {
	switch {
	case w.Width >= widthExpandedMin:
		return SizeExpanded
	case w.Width >= widthMediumMin:
		return SizeMedium
	}
	return SizeCompact
}

// HeightClass is the window's height bucketed into a SizeClass. Most layouts
// only need WidthClass; height is what tells a landscape phone (compact
// height) from a tablet in landscape, which a bottom sheet or a video player
// cares about.
func (w Window) HeightClass() SizeClass {
	switch {
	case w.Height >= heightExpandedMin:
		return SizeExpanded
	case w.Height >= heightMediumMin:
		return SizeMedium
	}
	return SizeCompact
}

// Posture derives the named posture from the fold. See Posture's constants.
func (w Window) Posture() Posture {
	if !w.HasFold || w.Fold.State != FoldHalfOpened {
		return PostureNormal
	}
	if w.Fold.Orientation == FoldHorizontal {
		return PostureTabletop
	}
	return PostureBook
}

// SeparatingFold returns the fold when content should be laid out around it,
// which is the one question a two-pane layout asks. A non-separating fold
// (a flat, continuous panel) is reported as none, since there is nothing to
// avoid.
func (w Window) SeparatingFold() (Fold, bool) {
	if w.HasFold && w.Fold.Separating {
		return w.Fold, true
	}
	return Fold{}, false
}

// hostEventWindow is the host event core consumes into the record below.
// Held to the shells by mobile/verify's TestWindowEventSpellingsAgree.
const hostEventWindow = "window"

var (
	windowMu     sync.RWMutex
	windowRecord Window
	windowSubs   = map[int]func(Window){}
	windowNext   int
)

// CurrentWindow reports the last window the host announced; the zero Window
// (Received false) until it has announced one.
func CurrentWindow() Window {
	windowMu.RLock()
	defer windowMu.RUnlock()
	return windowRecord
}

// OnWindow subscribes fn to window changes. The returned function cancels the
// subscription; calling it more than once is harmless.
//
// Process-wide like OnLifecycle, and for the same reason: one app, one
// window. fn runs on whichever goroutine delivered the event and must not
// block; writing State and calling RequestRender are fine from there.
func OnWindow(fn func(Window)) (cancel func()) {
	windowMu.Lock()
	defer windowMu.Unlock()
	id := windowNext
	windowNext++
	windowSubs[id] = fn
	return func() {
		windowMu.Lock()
		defer windowMu.Unlock()
		delete(windowSubs, id)
	}
}

// ReceiveWindow is the typed entry point for a host that reports in Go (a
// test, an embedder). The JSON hosts arrive through
// ReceiveHostEvent("window", ...), which decodes into this.
//
// Validation is split by what a bad value would do downstream:
//
//   - A negative, NaN or infinite size drops the whole report. A layout that
//     divides by the width or sizes a pane from it would otherwise produce
//     nonsense, and the previous report is a better guess than garbage.
//   - A fold whose state or orientation is not one core knows is dropped on
//     its own and the size kept, the same forward-compatibility stance as
//     ReceiveLifecycle: a newer shell's fifth posture must not reach a
//     switch in an older app that has no arm for it, but the window it
//     measured is still true.
//
// Received is set here, whatever the caller passed, since arriving through
// this function is what receiving means. A repeat of the current window is
// absorbed silently. Subscribers are notified outside the lock.
func ReceiveWindow(w Window) {
	if !validLength(w.Width) || !validLength(w.Height) {
		return
	}
	if w.HasFold && !validFold(w.Fold) {
		w.HasFold = false
	}
	if !w.HasFold {
		// Normalized so two reports that differ only in a stale Fold behind
		// HasFold=false compare equal and dedupe.
		w.Fold = Fold{}
	}
	w.Received = true

	windowMu.Lock()
	if w == windowRecord {
		windowMu.Unlock()
		return
	}
	windowRecord = w
	fns := make([]func(Window), 0, len(windowSubs))
	for _, fn := range windowSubs {
		fns = append(fns, fn)
	}
	windowMu.Unlock()

	for _, fn := range fns {
		fn(w)
	}
}

func validLength(v float64) bool {
	return v >= 0 && !math.IsNaN(v) && !math.IsInf(v, 0)
}

func validFold(f Fold) bool {
	switch f.State {
	case FoldFlat, FoldHalfOpened:
	default:
		return false
	}
	switch f.Orientation {
	case FoldVertical, FoldHorizontal:
	default:
		return false
	}
	b := f.Bounds
	return validLength(b.Width) && validLength(b.Height) &&
		!math.IsNaN(b.X) && !math.IsInf(b.X, 0) &&
		!math.IsNaN(b.Y) && !math.IsInf(b.Y, 0)
}

// receiveWindow decodes the "window" host event. The payload every host
// writes:
//
//	width   number   window width in layout units
//	height  number   window height in layout units
//	fold    object   absent when no fold crosses the window, else:
//	  state        string   "flat" | "half_opened"
//	  orientation  string   "vertical" | "horizontal"
//	  separating   bool
//	  occluding    bool
//	  x, y, width, height   number   the fold's bounds, window coordinates
//
// A report with no width or height is dropped: a host that measured nothing
// has nothing to say, and a zero default would read as a zero-sized window.
func receiveWindow(data map[string]any) {
	width, okW := numberProp(data, "width")
	height, okH := numberProp(data, "height")
	if !okW || !okH {
		return
	}
	w := Window{Width: width, Height: height}
	if f, ok := data["fold"].(map[string]any); ok {
		state, _ := f["state"].(string)
		orient, _ := f["orientation"].(string)
		sep, _ := f["separating"].(bool)
		occ, _ := f["occluding"].(bool)
		x, _ := numberProp(f, "x")
		y, _ := numberProp(f, "y")
		fw, _ := numberProp(f, "width")
		fh, _ := numberProp(f, "height")
		w.HasFold = true
		w.Fold = Fold{
			State:       FoldState(state),
			Orientation: FoldOrientation(orient),
			Separating:  sep,
			Occluding:   occ,
			Bounds:      WindowRect{X: x, Y: y, Width: fw, Height: fh},
		}
	}
	ReceiveWindow(w)
}

// resetWindowForTest returns the record and subscriptions to their initial
// state so one test's reports cannot leak into the next.
func resetWindowForTest() {
	windowMu.Lock()
	defer windowMu.Unlock()
	windowRecord = Window{}
	windowSubs = map[int]func(Window){}
}
