package mobile_test

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
	"github.com/rohanthewiz/grmob/mobile"
)

// audioApp shows the player's state, so a status tick reported over the
// bridge must come back as a patch on the text.
func audioApp(ctx *core.Context) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		s := hooks.UseAudio(ctx)
		return core.Column(core.Text("state: " + string(s.State))).Render(ctx)
	})
}

// The bound path end to end: the shell reports a host event as JSON text
// and receives the patches of the render it caused, exactly as it does for
// a Trigger* call.
func TestReportHostEventReturnsThePatchesItCaused(t *testing.T) {
	t.Cleanup(core.AudioStop)
	mobile.Register(core.NewContext(), audioApp)
	if !strings.Contains(mobile.RenderInitial(), "state: idle") {
		t.Fatalf("initial render should show idle")
	}
	core.AudioLoad(core.AudioTrack{URL: "u"})

	out := mobile.ReportHostEvent("audio_status", `{"url":"u","state":"playing","position":1.5,"duration":60}`)
	if !strings.Contains(out, "update-props") || !strings.Contains(out, "state: playing") {
		t.Errorf("ReportHostEvent should return the status-change patches, got: %s", out)
	}
	if s := core.CurrentAudioStatus(); s.Position != 1.5 || s.Duration != 60 {
		t.Errorf("status = %+v", s)
	}
}

func TestReportHostEventDropsMalformedJSON(t *testing.T) {
	t.Cleanup(core.AudioStop)
	mobile.Register(core.NewContext(), audioApp)
	mobile.RenderInitial()
	if out := mobile.ReportHostEvent("audio_status", `{not json`); out != "[]" {
		t.Errorf("malformed payload returned %q, want []", out)
	}
	// An empty payload is a valid empty object, not an error.
	if out := mobile.ReportHostEvent("audio_status", ""); out == "" {
		t.Errorf("empty payload returned nothing")
	}
}
