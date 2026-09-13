package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// ringOf returns the spinning ring under the status box.
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
		t.Fatal("the ring must exist and be hidden from assistive technology: it is decoration")
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

// The ring is turned by the platform: one revolution a second declared on the
// ring itself, and no fixed angle left over from the stepped implementation.
// The spin stays declared while hidden, because Display none is what stops the
// frames on every target and a style that flipped Spin as well would be a
// second patch for the same fact.
func TestSpinnerRingSpinsOnThePlatformClock(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		ctx, n := renderDebug(t, Spinner{Hidden: hidden})
		ring := ringOf(n)
		if ring.Style.Spin != spinPeriodMs || spinPeriodMs != 1000 {
			t.Errorf("hidden=%v: ring spin = %d, want %d (one turn a second)", hidden, ring.Style.Spin, 1000)
		}
		if ring.Style.Rotate != 0 {
			t.Errorf("hidden=%v: ring carries a fixed angle %v; the spin needs none", hidden, ring.Style.Rotate)
		}
		if n.Style.Spin != 0 {
			t.Error("the status box must not spin: a caller's Style could size it and the dot would orbit that")
		}
		ctx.Close()
	}
}

// renderSpinnerPass re-renders on the same context the way a Manager does,
// with the hook cursor reset between passes.
func renderSpinnerPass(ctx *core.Context, v core.View) *core.Node {
	ctx.Reset()
	ctx.BeginRenderPass()
	n := v.Render(ctx)
	ctx.EndRenderPass()
	return n
}

// Spinner holds no hooks, so it can come and go between passes ahead of a
// stateful sibling without moving that sibling's slot. With the old stepped
// spinner this shifted a NewState by two slots and debug mode reported it.
func TestSpinnerIsConditionalSafe(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	defer func() { core.SetDebugMode(false); core.ClearConcerns() }()

	ctx := core.NewContext()
	defer ctx.Close()

	type counter struct{ n int }
	view := func(show bool) core.View {
		return core.ComponentFunc(func(ctx *core.Context) *core.Node {
			var spinner core.View
			if show {
				spinner = Spinner{}
			}
			var kids []core.PropsAndChildren
			if spinner != nil {
				kids = append(kids, spinner)
			}
			c := core.NewState(ctx, counter{n: 7})
			kids = append(kids, core.Text("x"))
			if c.Get().n != 7 {
				t.Errorf("sibling state = %+v, want the value it was created with", c.Get())
			}
			return core.Column(kids...).Render(ctx)
		})
	}

	for _, show := range []bool{true, false, true, false} {
		renderSpinnerPass(ctx, view(show))
	}
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("concerns raised as the spinner came and went:\n%s", dump)
	}
}
