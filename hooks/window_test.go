package hooks_test

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// UseWindow must subscribe once, request a render on every change, and
// release the subscription when the tree closes.
func TestUseWindowRequestsRenderOnChangeAndUnsubscribesOnClose(t *testing.T) {
	// Leave a phone-sized window behind rather than whatever this test
	// ended on; core's reset is unexported, and nothing else here reads it.
	t.Cleanup(func() { core.ReceiveWindow(core.Window{Width: 1, Height: 1}) })
	ctx := core.NewContext()
	renders := make(chan struct{}, 16)
	ctx.OnStateChange(func() { renders <- struct{}{} })

	var last core.Window
	body := func(ctx *core.Context) { last = hooks.UseWindow(ctx) }

	renderPass(ctx, body)
	renderPass(ctx, body) // a second render must not stack a subscription

	unfolded := core.Window{
		Width: 841, Height: 673, HasFold: true,
		Fold: core.Fold{State: core.FoldHalfOpened, Orientation: core.FoldHorizontal, Separating: true,
			Bounds: core.WindowRect{Y: 336, Width: 841}},
	}
	core.ReceiveWindow(unfolded)
	awaitSignal(t, renders, "render request after a window change")
	assertQuiet(t, renders, 30*time.Millisecond, "second render request (subscription stacked)")

	// A repeat of the same window is not a change.
	core.ReceiveWindow(unfolded)
	assertQuiet(t, renders, 30*time.Millisecond, "render request for a repeated window")

	renderPass(ctx, body)
	if last.Posture() != core.PostureTabletop || last.WidthClass() != core.SizeExpanded {
		t.Errorf("hook returned %+v after the change", last)
	}

	ctx.Close()
	core.ReceiveWindow(core.Window{Width: 360, Height: 780})
	assertQuiet(t, renders, 30*time.Millisecond, "render request after Close")

	renderPass(ctx, body)
	core.ReceiveWindow(unfolded)
	awaitSignal(t, renders, "render request after re-mount")
	ctx.Close()
}
