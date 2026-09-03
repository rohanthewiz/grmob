package hooks_test

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// UseLifecycle must subscribe once, request a render on every transition,
// and release the subscription when the tree closes.
func TestUseLifecycleRequestsRenderOnTransitionAndUnsubscribesOnClose(t *testing.T) {
	t.Cleanup(func() { core.ReceiveLifecycle(core.LifecycleActive) })
	ctx := core.NewContext()
	renders := make(chan struct{}, 16)
	ctx.OnStateChange(func() { renders <- struct{}{} })

	var last core.LifecycleState
	body := func(ctx *core.Context) { last = hooks.UseLifecycle(ctx) }

	renderPass(ctx, body)
	renderPass(ctx, body) // a second render must not stack a subscription
	if last != core.LifecycleActive {
		t.Fatalf("initial state = %q", last)
	}

	core.ReceiveLifecycle(core.LifecycleBackground)
	awaitSignal(t, renders, "render request after a transition")
	assertQuiet(t, renders, 30*time.Millisecond, "second render request (subscription stacked)")

	renderPass(ctx, body)
	if last != core.LifecycleBackground {
		t.Errorf("hook returned %q after the transition", last)
	}

	ctx.Close()
	core.ReceiveLifecycle(core.LifecycleActive)
	assertQuiet(t, renders, 30*time.Millisecond, "render request after Close")

	// Re-mount over the same context (the WASM host's shape) subscribes again.
	renderPass(ctx, body)
	core.ReceiveLifecycle(core.LifecycleBackground)
	awaitSignal(t, renders, "render request after re-mount")
	ctx.Close()
}
