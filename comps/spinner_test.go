package comps

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// ringOf returns the rotating ring under the status box.
func ringOf(n *core.Node) *core.Node {
	return findFirst(n, func(n *core.Node) bool { return n.Type == "Column" })
}

func TestSpinnerIsANamedStatusWithAHiddenRing(t *testing.T) {
	ctx, n := renderDebug(t, Spinner{})
	defer ctx.Close()

	if n.Type != "Box" || n.Style.AccessibilityRole != core.RoleStatus {
		t.Fatalf("root = %q role %q, want a status Box", n.Type, n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilityLabel != "Loading" {
		t.Errorf("label = %q", n.Style.AccessibilityLabel)
	}
	ring := ringOf(n)
	if ring == nil || !ring.Style.AccessibilityHidden {
		t.Fatal("the ring must exist and be hidden from assistive technology: its steps are not news")
	}
	if ring.Style.Width != "24px" || ring.Style.BorderRadius != 12 {
		t.Errorf("medium ring = %s radius %v, want 24px and 12", ring.Style.Width, ring.Style.BorderRadius)
	}
	if len(ring.Children) != 1 || ring.Children[0].Style.Background != core.DefaultTheme.Colors.Primary {
		t.Error("the ring carries one Primary dot: without it a rotation draws identical pixels")
	}
}

func TestSpinnerSizesReadTheSpacingScale(t *testing.T) {
	for size, want := range map[SpinnerSize]string{SpinnerSmall: "16px", SpinnerLarge: "32px"} {
		ctx, n := renderDebug(t, Spinner{Size: size, Label: "Uploading"})
		if got := ringOf(n).Style.Width; got != want {
			t.Errorf("%q width = %s, want %s", size, got, want)
		}
		if n.Style.AccessibilityLabel != "Uploading" {
			t.Error("Label overrides the announcement")
		}
		ctx.Close()
	}
}

func TestSpinnerHiddenIsDisplayNone(t *testing.T) {
	ctx, n := renderDebug(t, Spinner{Hidden: true})
	defer ctx.Close()
	if n.Style.Display != core.DisplayNone {
		t.Errorf("display = %q, want none", n.Style.Display)
	}
}

// renderSpinnerPasses re-renders on the same context the way a Manager does,
// with the hook cursor reset between passes.
func renderSpinnerPass(ctx *core.Context, s Spinner) *core.Node {
	ctx.Reset()
	ctx.BeginRenderPass()
	n := s.Render(ctx)
	ctx.EndRenderPass()
	return n
}

// A visible spinner advances; a hidden one does not, even across many ticks.
func TestSpinnerStepsWhileVisibleAndHoldsWhileHidden(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	defer func() { core.SetDebugMode(false); core.ClearConcerns() }()

	ctx := core.NewContext()
	defer ctx.Close()

	if got := ringOf(renderSpinnerPass(ctx, Spinner{})).Style.Rotate; got != 0 {
		t.Fatalf("first pass rotate = %v, want 0", got)
	}

	deadline := time.Now().Add(2 * time.Second)
	var moved float64
	for time.Now().Before(deadline) {
		time.Sleep(stepEvery)
		if moved = ringOf(renderSpinnerPass(ctx, Spinner{})).Style.Rotate; moved != 0 {
			break
		}
	}
	if moved == 0 {
		t.Fatal("a visible spinner never advanced")
	}
	if int(moved)%stepDegrees != 0 {
		t.Errorf("rotate = %v, want a multiple of %d", moved, stepDegrees)
	}

	// Hide it, let a tick already past the pause check land, then hold.
	renderSpinnerPass(ctx, Spinner{Hidden: true})
	time.Sleep(2 * stepEvery)
	held := ringOf(renderSpinnerPass(ctx, Spinner{Hidden: true})).Style.Rotate
	time.Sleep(4 * stepEvery)
	if got := ringOf(renderSpinnerPass(ctx, Spinner{Hidden: true})).Style.Rotate; got != held {
		t.Errorf("a hidden spinner moved from %v to %v", held, got)
	}
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("concerns raised across passes:\n%s", dump)
	}
}
