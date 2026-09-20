package core

import (
	"math"
	"testing"
)

func freshWindow(t *testing.T) {
	t.Helper()
	resetWindowForTest()
	t.Cleanup(resetWindowForTest)
}

// The payload every shell writes, decoded end to end through the dispatcher,
// so the key spellings in receiveWindow are what is under test and not a
// typed shortcut around them.
func TestWindowHostEventDecodesSizeAndFold(t *testing.T) {
	freshWindow(t)
	ReceiveHostEvent("window", map[string]any{
		"width": 841.0, "height": 673.0,
		"fold": map[string]any{
			"state": "half_opened", "orientation": "vertical",
			"separating": true, "occluding": false,
			"x": 420.5, "y": 0.0, "width": 0.0, "height": 673.0,
		},
	})
	w := CurrentWindow()
	want := Window{
		Width: 841, Height: 673, HasFold: true, Received: true,
		Fold: Fold{
			State: FoldHalfOpened, Orientation: FoldVertical, Separating: true,
			Bounds: WindowRect{X: 420.5, Height: 673},
		},
	}
	if w != want {
		t.Fatalf("got %+v\nwant %+v", w, want)
	}
	if w.Posture() != PostureBook {
		t.Errorf("posture = %q, want book", w.Posture())
	}
	if f, ok := w.SeparatingFold(); !ok || f.Bounds.X != 420.5 {
		t.Errorf("SeparatingFold = %+v, %v", f, ok)
	}
}

// A Go-built payload writes ints, and a fold-less report has no "fold" key.
func TestWindowHostEventWithoutFold(t *testing.T) {
	freshWindow(t)
	ReceiveHostEvent("window", map[string]any{"width": 360, "height": 780})
	w := CurrentWindow()
	if !w.Received || w.HasFold || w.Width != 360 || w.Height != 780 {
		t.Fatalf("got %+v", w)
	}
	if w.Posture() != PostureNormal {
		t.Errorf("posture = %q, want normal", w.Posture())
	}
}

// Before any report the window is unknown, and the layouts that branch on it
// get a phone: compact is the one class that is never wrong enough to break.
func TestWindowBeforeReportIsCompactAndNotReceived(t *testing.T) {
	freshWindow(t)
	w := CurrentWindow()
	if w.Received || w.WidthClass() != SizeCompact || w.HeightClass() != SizeCompact {
		t.Errorf("got %+v (%s × %s)", w, w.WidthClass(), w.HeightClass())
	}
}

// The breakpoints sit exactly on Material's numbers, inclusive at the low
// edge of each bucket.
func TestWindowSizeClassBreakpoints(t *testing.T) {
	for _, c := range []struct {
		width, height float64
		wc, hc        SizeClass
	}{
		{599.9, 479.9, SizeCompact, SizeCompact},
		{600, 480, SizeMedium, SizeMedium},
		{839.9, 899.9, SizeMedium, SizeMedium},
		{840, 900, SizeExpanded, SizeExpanded},
	} {
		w := Window{Width: c.width, Height: c.height}
		if w.WidthClass() != c.wc || w.HeightClass() != c.hc {
			t.Errorf("%v×%v: got %s×%s, want %s×%s", c.width, c.height,
				w.WidthClass(), w.HeightClass(), c.wc, c.hc)
		}
	}
}

func TestWindowPostures(t *testing.T) {
	for _, c := range []struct {
		name string
		w    Window
		want Posture
	}{
		{"no fold", Window{}, PostureNormal},
		{"flat vertical", Window{HasFold: true, Fold: Fold{State: FoldFlat, Orientation: FoldVertical}}, PostureNormal},
		{"half-open horizontal", Window{HasFold: true, Fold: Fold{State: FoldHalfOpened, Orientation: FoldHorizontal}}, PostureTabletop},
		{"half-open vertical", Window{HasFold: true, Fold: Fold{State: FoldHalfOpened, Orientation: FoldVertical}}, PostureBook},
		// A stale fold behind HasFold=false is not a posture.
		{"stale fold", Window{Fold: Fold{State: FoldHalfOpened, Orientation: FoldVertical}}, PostureNormal},
	} {
		if got := c.w.Posture(); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// A flat, continuous panel reports a fold that is not separating; a two-pane
// layout must not split around it.
func TestWindowNonSeparatingFoldIsNotAvoided(t *testing.T) {
	w := Window{HasFold: true, Fold: Fold{State: FoldFlat, Orientation: FoldVertical}}
	if _, ok := w.SeparatingFold(); ok {
		t.Error("a non-separating fold was reported as one to lay out around")
	}
}

// Subscribers hear changes only: Android re-emits the same layout on every
// Activity recreation, and each repeat must cost nothing.
func TestWindowDedupesRepeats(t *testing.T) {
	freshWindow(t)
	var heard []Window
	cancel := OnWindow(func(w Window) { heard = append(heard, w) })
	defer cancel()

	ReceiveWindow(Window{Width: 360, Height: 780})
	ReceiveWindow(Window{Width: 360, Height: 780})
	// Differs only in a Fold hidden behind HasFold=false: normalized away.
	ReceiveWindow(Window{Width: 360, Height: 780, Fold: Fold{State: FoldFlat}})
	ReceiveWindow(Window{Width: 840, Height: 780})
	if len(heard) != 2 {
		t.Fatalf("heard %d reports, want 2: %+v", len(heard), heard)
	}

	cancel()
	cancel() // idempotent
	ReceiveWindow(Window{Width: 360, Height: 780})
	if len(heard) != 2 {
		t.Errorf("a cancelled subscriber still heard a report")
	}
}

// A size that cannot be a window drops the report; the previous one stands.
func TestWindowRejectsInvalidSizes(t *testing.T) {
	freshWindow(t)
	ReceiveWindow(Window{Width: 360, Height: 780})
	for _, bad := range []Window{
		{Width: -1, Height: 780},
		{Width: math.NaN(), Height: 780},
		{Width: 360, Height: math.Inf(1)},
	} {
		ReceiveWindow(bad)
	}
	if w := CurrentWindow(); w.Width != 360 || w.Height != 780 {
		t.Errorf("an invalid report replaced the record: %+v", w)
	}
	// A payload missing a dimension is not a report at all.
	ReceiveHostEvent("window", map[string]any{"width": 500})
	if w := CurrentWindow(); w.Width != 360 {
		t.Errorf("a report with no height was stored: %+v", w)
	}
}

// A fold core does not recognise is dropped on its own; the size it came
// with is still a true measurement and is kept.
func TestWindowUnknownFoldKeepsTheSize(t *testing.T) {
	freshWindow(t)
	ReceiveHostEvent("window", map[string]any{
		"width": 700, "height": 800,
		"fold": map[string]any{"state": "tented", "orientation": "vertical", "separating": true},
	})
	w := CurrentWindow()
	if w.Width != 700 || w.HasFold || w.Fold != (Fold{}) {
		t.Errorf("got %+v, want the size with no fold", w)
	}

	ReceiveHostEvent("window", map[string]any{
		"width": 700, "height": 800,
		"fold": map[string]any{"state": "flat", "orientation": "diagonal"},
	})
	if CurrentWindow().HasFold {
		t.Error("an unknown orientation was stored")
	}
}

// Safe-area insets decode from the same payload, and a shell that predates
// them still reports a window: the key is optional and its absence is zero,
// which is also what a host with no system bars sends.
func TestWindowHostEventDecodesInsets(t *testing.T) {
	freshWindow(t)
	ReceiveHostEvent("window", map[string]any{
		"width": 411, "height": 891,
		"insets": map[string]any{"top": 24.0, "bottom": 48.0, "left": 0.0, "right": 0.0},
	})
	if got := CurrentWindow().Insets; got != (SafeInsets{Top: 24, Bottom: 48}) {
		t.Fatalf("insets = %+v", got)
	}

	ReceiveHostEvent("window", map[string]any{"width": 411, "height": 892})
	if got := CurrentWindow().Insets; got != (SafeInsets{}) {
		t.Errorf("a payload with no insets kept the previous ones: %+v", got)
	}
}

// A change in the insets alone is a change: the bars come and go (immersive
// mode, a rotation that moves the cutout) without the window resizing, and a
// layout that positions against them has to hear it.
func TestWindowInsetsAloneAreAChange(t *testing.T) {
	freshWindow(t)
	var heard int
	cancel := OnWindow(func(Window) { heard++ })
	defer cancel()

	ReceiveWindow(Window{Width: 411, Height: 891, Insets: SafeInsets{Top: 24}})
	ReceiveWindow(Window{Width: 411, Height: 891, Insets: SafeInsets{Top: 24}})
	ReceiveWindow(Window{Width: 411, Height: 891, Insets: SafeInsets{Top: 40}})
	if heard != 2 {
		t.Fatalf("heard %d reports, want 2", heard)
	}
}

// Insets that cannot be insets are dropped on their own, the way an unknown
// fold is: the window they arrived with was measured correctly and is kept.
// A negative edge is the signature of a conversion that divided by a zero
// density, which is exactly when the size is still right.
func TestWindowInvalidInsetsKeepTheSize(t *testing.T) {
	freshWindow(t)
	for _, bad := range []SafeInsets{
		{Top: -1},
		{Bottom: math.NaN()},
		{Left: math.Inf(1)},
		{Right: -0.5},
	} {
		ReceiveWindow(Window{Width: 700, Height: 800, Insets: bad})
		w := CurrentWindow()
		if w.Width != 700 || w.Insets != (SafeInsets{}) {
			t.Errorf("%+v gave %+v, want the size with no insets", bad, w)
		}
	}
}

// A subscriber may read the record and cancel itself from inside its handler
// without deadlocking.
func TestWindowSubscriberMayReadAndCancel(t *testing.T) {
	freshWindow(t)
	var cancel func()
	var seen Window
	cancel = OnWindow(func(Window) {
		seen = CurrentWindow()
		cancel()
	})
	ReceiveWindow(Window{Width: 1, Height: 2})
	if seen.Width != 1 {
		t.Errorf("handler read %+v", seen)
	}
}
