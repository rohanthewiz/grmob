package core

import (
	"testing"
)

// recordSystemEvents installs a recording system-event handler for the test
// and resets the audio record, so one test's playback never leaks into the
// next (both are package-level).
func recordSystemEvents(t *testing.T) *[]struct {
	Name string
	Data map[string]any
} {
	t.Helper()
	resetAudioForTest()
	var events []struct {
		Name string
		Data map[string]any
	}
	SetSystemEventHandler(func(name string, data map[string]any) {
		events = append(events, struct {
			Name string
			Data map[string]any
		}{name, data})
	})
	t.Cleanup(func() {
		SetSystemEventHandler(nil)
		resetAudioForTest()
	})
	return &events
}

func TestAudioLoadSendsTheCommandAndRecordsTheTrackOptimistically(t *testing.T) {
	events := recordSystemEvents(t)

	track := AudioTrack{URL: "https://x.org/s.mp3", Title: "Grace", Artist: "Pastor A", ArtworkURL: "https://x.org/a.png"}
	AudioLoad(track, AudioStartAt(42), AudioWithRate(1.5), AudioAutoplay(false))

	if len(*events) != 1 || (*events)[0].Name != "audio" {
		t.Fatalf("events = %+v, want one \"audio\" event", *events)
	}
	d := (*events)[0].Data
	if d["command"] != "load" || d["url"] != track.URL || d["title"] != "Grace" ||
		d["artist"] != "Pastor A" || d["artwork"] != track.ArtworkURL {
		t.Errorf("load payload = %v", d)
	}
	if d["autoplay"] != false || d["start"] != 42.0 || d["rate"] != 1.5 {
		t.Errorf("load options = autoplay %v start %v rate %v", d["autoplay"], d["start"], d["rate"])
	}

	s := CurrentAudioStatus()
	if s.State != AudioLoading || s.Track != track || s.Position != 42 || s.Rate != 1.5 {
		t.Errorf("optimistic status = %+v", s)
	}
}

func TestAudioLoadDropsAnEmptyURL(t *testing.T) {
	events := recordSystemEvents(t)
	AudioLoad(AudioTrack{Title: "no url"})
	if len(*events) != 0 {
		t.Fatalf("an empty URL was sent: %+v", *events)
	}
	if CurrentAudioStatus().State != AudioIdle {
		t.Errorf("status changed for an empty URL: %+v", CurrentAudioStatus())
	}
}

// With nothing loaded there is nothing to play, pause or seek, and every
// host would have to ignore the command separately.
func TestAudioTransportCommandsNeedALoadedTrack(t *testing.T) {
	events := recordSystemEvents(t)

	AudioPlay()
	AudioPause()
	AudioSeek(10)
	AudioSkip(30)
	AudioSetRate(2)
	if len(*events) != 0 {
		t.Fatalf("commands sent with nothing loaded: %+v", *events)
	}

	AudioLoad(AudioTrack{URL: "u"})
	*events = nil
	AudioPlay()
	AudioPause()
	AudioSeek(-5) // clamped to 0
	AudioSkip(-10)
	AudioSetRate(0) // ignored
	AudioSetRate(1.25)

	want := []string{"play", "pause", "seek", "skip", "rate"}
	if len(*events) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(*events), len(want), *events)
	}
	for i, w := range want {
		if (*events)[i].Data["command"] != w {
			t.Errorf("event %d command = %v, want %s", i, (*events)[i].Data["command"], w)
		}
	}
	if (*events)[2].Data["position"] != 0.0 {
		t.Errorf("negative seek was not clamped: %v", (*events)[2].Data)
	}
	if (*events)[3].Data["delta"] != -10.0 {
		t.Errorf("skip delta = %v", (*events)[3].Data["delta"])
	}
	if (*events)[4].Data["rate"] != 1.25 {
		t.Errorf("rate = %v", (*events)[4].Data["rate"])
	}
}

func TestAudioToggleFollowsTheReportedState(t *testing.T) {
	events := recordSystemEvents(t)
	AudioLoad(AudioTrack{URL: "u"})

	ReceiveHostEvent("audio_status", map[string]any{"url": "u", "state": "playing"})
	*events = nil
	AudioToggle()
	if (*events)[0].Data["command"] != "pause" {
		t.Errorf("toggle while playing sent %v", (*events)[0].Data)
	}

	ReceiveHostEvent("audio_status", map[string]any{"url": "u", "state": "paused"})
	*events = nil
	AudioToggle()
	if (*events)[0].Data["command"] != "play" {
		t.Errorf("toggle while paused sent %v", (*events)[0].Data)
	}
}

// The host echoes only the URL, so the metadata the app supplied must
// survive every status tick — and a rate the host did not report must not
// reset the one the app set.
func TestAudioStatusFromHostKeepsTrackMetadataAndRate(t *testing.T) {
	recordSystemEvents(t)
	track := AudioTrack{URL: "u", Title: "Grace", Artist: "A"}
	AudioLoad(track, AudioWithRate(1.5))

	var seen []AudioStatus
	cancel := OnAudioStatus(func(s AudioStatus) { seen = append(seen, s) })
	defer cancel()

	ReceiveHostEvent("audio_status", map[string]any{
		"url": "u", "state": "playing", "position": 12.5, "duration": 1800.0,
	})

	s := CurrentAudioStatus()
	if s.Track != track {
		t.Errorf("track metadata lost: %+v", s.Track)
	}
	if s.State != AudioPlaying || s.Position != 12.5 || s.Duration != 1800 {
		t.Errorf("status = %+v", s)
	}
	if s.Rate != 1.5 {
		t.Errorf("rate reset to %v by a tick that did not report one", s.Rate)
	}
	if len(seen) != 1 || seen[0].State != AudioPlaying {
		t.Errorf("subscriber saw %+v", seen)
	}

	// An error state carries its message; leaving it clears the message.
	ReceiveHostEvent("audio_status", map[string]any{"url": "u", "state": "error", "error": "404"})
	if s := CurrentAudioStatus(); s.State != AudioError || s.Error != "404" {
		t.Errorf("error status = %+v", s)
	}
	ReceiveHostEvent("audio_status", map[string]any{"url": "u", "state": "paused", "error": "stale"})
	if s := CurrentAudioStatus(); s.Error != "" {
		t.Errorf("error text survived a non-error state: %+v", s)
	}

	// A different URL keeps only the URL: never someone else's title.
	ReceiveHostEvent("audio_status", map[string]any{"url": "other", "state": "playing"})
	if s := CurrentAudioStatus(); s.Track != (AudioTrack{URL: "other"}) {
		t.Errorf("foreign URL carried the old metadata: %+v", s.Track)
	}

	cancel()
	ReceiveHostEvent("audio_status", map[string]any{"url": "other", "state": "paused"})
	if len(seen) != 4 {
		t.Errorf("cancelled subscriber still notified: %d calls", len(seen))
	}
}

func TestAudioStopReturnsToIdleAndSendsStop(t *testing.T) {
	events := recordSystemEvents(t)
	AudioLoad(AudioTrack{URL: "u", Title: "t"})
	*events = nil

	AudioStop()
	if len(*events) != 1 || (*events)[0].Data["command"] != "stop" {
		t.Fatalf("events = %+v", *events)
	}
	if s := CurrentAudioStatus(); s.State != AudioIdle || s.Loaded() || s.Track.URL != "" {
		t.Errorf("status after stop = %+v", s)
	}
}

func TestAudioProgressIsAClampedFraction(t *testing.T) {
	cases := []struct {
		pos, dur, want float64
	}{
		{0, 0, 0}, {10, 0, 0}, {30, 120, 0.25}, {200, 120, 1}, {-5, 120, 0},
	}
	for _, c := range cases {
		got := AudioStatus{Position: c.pos, Duration: c.dur}.Progress()
		if got != c.want {
			t.Errorf("Progress(%v/%v) = %v, want %v", c.pos, c.dur, got, c.want)
		}
	}
}

// A subscriber that reads status or cancels itself from inside its handler
// must not deadlock: notification runs outside the record lock.
func TestAudioSubscriberMayCancelItself(t *testing.T) {
	recordSystemEvents(t)
	calls := 0
	var cancel func()
	cancel = OnAudioStatus(func(s AudioStatus) {
		calls++
		_ = CurrentAudioStatus()
		cancel()
	})
	AudioLoad(AudioTrack{URL: "u"})
	AudioStop()
	if calls != 1 {
		t.Errorf("self-cancelling subscriber ran %d times, want 1", calls)
	}
}
