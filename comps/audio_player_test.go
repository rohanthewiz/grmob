package comps

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var sermonTrack = core.AudioTrack{URL: "https://example.com/s.mp3", Title: "Sunday, 14 March", Artist: "Pastor Ade"}

// audioHarness renders an AudioPlayer across passes on one Context (it holds
// two hooks), records the "audio" commands it sends, and resets the shared
// player record afterwards so tests do not leak a loaded track into each
// other.
type audioHarness struct {
	*rowHarness
	commands []string
}

func newAudioHarness(t *testing.T, p AudioPlayer, status core.AudioStatus) *audioHarness {
	t.Helper()
	h := &audioHarness{}
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name != "audio" {
			return
		}
		cmd := data["command"].(string)
		switch cmd {
		case "load":
			cmd += " " + data["url"].(string)
		case "skip":
			cmd += fmt.Sprintf(" %g", data["delta"])
		case "seek":
			cmd += fmt.Sprintf(" %g", data["position"])
		case "rate":
			cmd += fmt.Sprintf(" %g", data["rate"])
		}
		h.commands = append(h.commands, cmd)
	})
	core.ReceiveAudioStatus(status)
	t.Cleanup(func() {
		core.SetSystemEventHandler(nil)
		core.ReceiveAudioStatus(core.AudioStatus{State: core.AudioIdle, Rate: 1})
	})
	h.rowHarness = newRowHarness(t, func() core.View { return p })
	return h
}

func (h *audioHarness) button(label string) *core.Node {
	h.t.Helper()
	for _, b := range buttonsOf(h.node) {
		if b.Props["label"] == label {
			return b
		}
	}
	h.t.Fatalf("no button %q", label)
	return nil
}

func (h *audioHarness) tap(label string) {
	h.t.Helper()
	h.ctx.TriggerCallback(h.button(label).Props["onClick"].(string))
	h.render()
}

func (h *audioHarness) slider() *core.Node {
	h.t.Helper()
	s := findFirst(h.node, func(n *core.Node) bool { return n.Type == "Slider" })
	if s == nil {
		h.t.Fatal("no seek bar")
	}
	return s
}

func (h *audioHarness) wantCommands(want ...string) {
	h.t.Helper()
	if strings.Join(h.commands, "|") != strings.Join(want, "|") {
		h.t.Errorf("commands = %q, want %q", h.commands, want)
	}
}

// Nothing loaded: the track is shown idle, Play loads it, and every control
// that would act on a stream is disabled.
func TestAudioPlayerIdleLoadsItsTrack(t *testing.T) {
	h := newAudioHarness(t, AudioPlayer{Track: sermonTrack, ShowStop: true, Rates: []float64{1, 1.5}},
		core.AudioStatus{State: core.AudioIdle, Rate: 1})

	if h.node.Style.AccessibilityRole != core.RoleGroup || h.node.Style.AccessibilityLabel != sermonTrack.Title {
		t.Errorf("group = role %q name %q", h.node.Style.AccessibilityRole, h.node.Style.AccessibilityLabel)
	}
	if findText(h.node, "Pastor Ade") == nil {
		t.Error("the second line is the Artist while idle")
	}
	for _, l := range []string{"−15s", "+15s", "Speed 1×", "Stop"} {
		if !h.button(l).Style.Disabled {
			t.Errorf("%q should be disabled with no track of ours loaded", l)
		}
	}
	if !h.slider().Style.Disabled {
		t.Error("the seek bar should be disabled with nothing to seek")
	}
	if h.button("Play").Style.Disabled {
		t.Fatal("Play is the one live control")
	}
	h.tap("Play")
	h.wantCommands("load " + sermonTrack.URL)
}

// The tree passes the accessibility audit both before the host knows a
// duration and after. Before, the seek bar states no value: a 0-to-0 range
// is one Compose cannot express, and the audit (which only a full render
// manager runs by itself) reports it.
func TestAudioPlayerPassesTheAuditWithAndWithoutADuration(t *testing.T) {
	for _, st := range []core.AudioStatus{
		{State: core.AudioIdle, Rate: 1},
		{Track: sermonTrack, State: core.AudioLoading, Rate: 1},
		{Track: sermonTrack, State: core.AudioPlaying, Position: 5, Duration: 60, Rate: 1},
	} {
		h := newAudioHarness(t, AudioPlayer{Track: sermonTrack}, st)
		core.ClearConcerns()
		core.AuditTree(h.node)
		if dump := core.DumpConcerns(); dump != "" {
			t.Errorf("state %s: audit raised:\n%s", st.State, dump)
		}
		if st.Duration == 0 && h.slider().Style.AccessibilityValue.Max != "" {
			t.Errorf("state %s: a seek bar with no duration should state no range", st.State)
		}
	}
}

// Another screen's track is loaded: this one is still idle, and Play
// replaces the stream rather than toggling somebody else's.
func TestAudioPlayerDoesNotDriveAnotherTrack(t *testing.T) {
	other := core.AudioTrack{URL: "https://example.com/other.mp3"}
	h := newAudioHarness(t, AudioPlayer{Track: sermonTrack},
		core.AudioStatus{Track: other, State: core.AudioPlaying, Position: 30, Duration: 90, Rate: 1})

	if !h.button("−15s").Style.Disabled {
		t.Error("skip would act on the other track")
	}
	h.tap("Play")
	h.wantCommands("load " + sermonTrack.URL)
}

// Our track playing: Pause, live controls, times from the status, and the
// seek bar's value spoken as clock times.
func TestAudioPlayerDrivesItsOwnTrack(t *testing.T) {
	h := newAudioHarness(t, AudioPlayer{Track: sermonTrack, Rates: []float64{1, 1.25, 1.5}},
		core.AudioStatus{Track: sermonTrack, State: core.AudioPlaying, Position: 65, Duration: 3700, Rate: 1.25})

	if findText(h.node, "1:05") == nil || findText(h.node, "1:01:40") == nil {
		t.Error("elapsed 1:05 and total 1:01:40 should be drawn")
	}
	s := h.slider()
	if s.Style.Disabled {
		t.Error("the seek bar is live")
	}
	if v := s.Style.AccessibilityValue; v.Text != "1:05 of 1:01:40" || s.Style.AccessibilityLabel != "Position" {
		t.Errorf("seek bar = name %q value %+v", s.Style.AccessibilityLabel, v)
	}
	if b := h.button("−15s"); b.Style.AccessibilityLabel != "Back 15 seconds" {
		t.Errorf("skip name = %q", b.Style.AccessibilityLabel)
	}

	h.tap("−15s")
	h.tap("+15s")
	h.tap("Speed 1.25×")
	h.tap("Pause")
	h.wantCommands("skip -15", "skip 15", "rate 1.5", "pause")
}

// While the thumb is down the elapsed time follows it; the seek is sent once,
// on release, and the reading returns to the status.
func TestAudioPlayerScrubFollowsTheFingerAndSeeksOnRelease(t *testing.T) {
	h := newAudioHarness(t, AudioPlayer{Track: sermonTrack},
		core.AudioStatus{Track: sermonTrack, State: core.AudioPlaying, Position: 65, Duration: 3700, Rate: 1})

	h.ctx.TriggerTextCallback(h.slider().Props["onChange"].(string), "600")
	h.render()
	if findText(h.node, "10:00") == nil {
		t.Error("the elapsed time should follow the drag")
	}
	h.wantCommands()

	h.ctx.TriggerTextCallback(h.slider().Props["onChangeEnd"].(string), "605")
	h.render()
	h.wantCommands("seek 605")
	if findText(h.node, "1:05") == nil {
		t.Error("after release the reading is the status's again, until the host reports the seek")
	}
}

// Loading and failure replace the Artist line, because they are why the
// controls are not answering.
func TestAudioPlayerSecondLineStates(t *testing.T) {
	h := newAudioHarness(t, AudioPlayer{Track: sermonTrack},
		core.AudioStatus{Track: sermonTrack, State: core.AudioError, Error: "404", Rate: 1})
	if findText(h.node, "Couldn't play: 404") == nil {
		t.Error("an error names itself")
	}
}

// SkipSeconds: a custom step, and negative leaves the buttons out.
func TestAudioPlayerSkipSeconds(t *testing.T) {
	h := newAudioHarness(t, AudioPlayer{Track: sermonTrack, SkipSeconds: 30}, core.AudioStatus{State: core.AudioIdle, Rate: 1})
	h.button("−30s")
	h2 := newAudioHarness(t, AudioPlayer{Track: sermonTrack, SkipSeconds: -1}, core.AudioStatus{State: core.AudioIdle, Rate: 1})
	if n := len(buttonsOf(h2.node)); n != 1 {
		t.Errorf("buttons = %d, want Play alone", n)
	}
}

// No URL: Play disabled and reported.
func TestAudioPlayerWithNoTrackReports(t *testing.T) {
	core.ReceiveAudioStatus(core.AudioStatus{State: core.AudioIdle, Rate: 1})
	h := newQuietRowHarness(t, func() core.View { return AudioPlayer{} })
	if dump := core.DumpConcerns(); !strings.Contains(dump, ConcernAudioPlayerNoTrack) {
		t.Errorf("concerns = %q", dump)
	}
	var play *core.Node
	for _, b := range buttonsOf(h.node) {
		if b.Props["label"] == "Play" {
			play = b
		}
	}
	if play == nil || !play.Style.Disabled {
		t.Error("Play should be disabled with no URL")
	}
}

func TestAudioClockAndRates(t *testing.T) {
	for in, want := range map[float64]string{0: "0:00", 59.6: "1:00", 3599.4: "59:59", 3661: "1:01:01", -3: "0:00"} {
		if got := audioClock(in); got != want {
			t.Errorf("audioClock(%v) = %q, want %q", in, got, want)
		}
	}
	rates := []float64{1, 1.5, 2}
	for _, tc := range [][2]float64{{1, 1.5}, {2, 1}, {1.2500001, 1}, {1.5000001, 2}} {
		if got := nextAudioRate(rates, tc[0]); got != tc[1] {
			t.Errorf("nextAudioRate(%v) = %v, want %v", tc[0], got, tc[1])
		}
	}
}
