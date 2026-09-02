package hooks_test

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// UseAudio must subscribe once, request a render on every status change,
// and release the subscription when the tree closes.
func TestUseAudioRequestsRenderOnStatusAndUnsubscribesOnClose(t *testing.T) {
	t.Cleanup(core.AudioStop)
	ctx := core.NewContext()
	renders := make(chan struct{}, 16)
	ctx.OnStateChange(func() { renders <- struct{}{} })

	var last core.AudioStatus
	body := func(ctx *core.Context) { last = hooks.UseAudio(ctx) }

	renderPass(ctx, body)
	renderPass(ctx, body) // a second render must not stack a subscription
	if last.State != core.AudioIdle {
		t.Fatalf("initial status = %+v", last)
	}

	core.ReceiveAudioStatus(core.AudioStatus{Track: core.AudioTrack{URL: "u"}, State: core.AudioPlaying, Position: 7})
	awaitSignal(t, renders, "render request after a status change")
	assertQuiet(t, renders, 30*time.Millisecond, "second render request (subscription stacked)")

	renderPass(ctx, body)
	if last.State != core.AudioPlaying || last.Position != 7 {
		t.Errorf("hook returned %+v after the change", last)
	}

	ctx.Close()
	core.ReceiveAudioStatus(core.AudioStatus{Track: core.AudioTrack{URL: "u"}, State: core.AudioPaused})
	assertQuiet(t, renders, 30*time.Millisecond, "render request after Close")

	// Re-mount over the same context (the WASM host's shape) subscribes again.
	renderPass(ctx, body)
	core.ReceiveAudioStatus(core.AudioStatus{Track: core.AudioTrack{URL: "u"}, State: core.AudioPlaying})
	awaitSignal(t, renders, "render request after re-mount")
	ctx.Close()
}
