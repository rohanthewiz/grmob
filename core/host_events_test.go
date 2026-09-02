package core

import "testing"

func TestHostEventFansOutToSubscribersByName(t *testing.T) {
	var got []string
	cancelA := OnHostEvent("thing", func(d map[string]any) { got = append(got, "a:"+d["v"].(string)) })
	cancelB := OnHostEvent("thing", func(d map[string]any) { got = append(got, "b:"+d["v"].(string)) })
	cancelOther := OnHostEvent("other", func(d map[string]any) { got = append(got, "other") })
	defer cancelA()
	defer cancelB()
	defer cancelOther()

	ReceiveHostEvent("thing", map[string]any{"v": "1"})
	if len(got) != 2 {
		t.Fatalf("got %v, want both subscribers and not the other name", got)
	}

	cancelA()
	cancelA() // idempotent
	ReceiveHostEvent("thing", map[string]any{"v": "2"})
	if len(got) != 3 || got[2] != "b:2" {
		t.Errorf("after cancel got %v", got)
	}

	// A nil payload is delivered as an empty map, so subscribers never
	// have to nil-check.
	ReceiveHostEvent("other", nil)
	if got[len(got)-1] != "other" {
		t.Errorf("nil payload was not delivered: %v", got)
	}
}

// Cancelling from inside a handler is the natural "once" pattern and must
// not deadlock on the registry lock.
func TestHostEventSubscriberMayCancelItself(t *testing.T) {
	calls := 0
	var cancel func()
	cancel = OnHostEvent("once", func(map[string]any) {
		calls++
		cancel()
	})
	ReceiveHostEvent("once", nil)
	ReceiveHostEvent("once", nil)
	if calls != 1 {
		t.Errorf("handler ran %d times, want 1", calls)
	}
}

// Core's own consumer runs before app subscribers, so a subscriber to
// "audio_status" already sees the updated record.
func TestHostEventCoreConsumerRunsFirst(t *testing.T) {
	recordSystemEvents(t)
	AudioLoad(AudioTrack{URL: "u"})
	var seenState AudioState
	cancel := OnHostEvent("audio_status", func(map[string]any) {
		seenState = CurrentAudioStatus().State
	})
	defer cancel()
	ReceiveHostEvent("audio_status", map[string]any{"url": "u", "state": "playing", "position": 3})
	if seenState != AudioPlaying {
		t.Errorf("subscriber saw state %q, want playing (core consumer first)", seenState)
	}
	if CurrentAudioStatus().Position != 3 {
		t.Errorf("an int-typed number was not read: %+v", CurrentAudioStatus())
	}
}
