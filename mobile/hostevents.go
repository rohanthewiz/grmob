package mobile

import (
	"encoding/json"
	"log"
)

// ReportHostEvent is the host→app half of the bridge for traffic that is
// not an answer to a registered callback: the audio player's status ticks
// today, and whatever the shells report next (see core/host_events.go for
// the channel and why it is generic).
//
// It mirrors the Trigger* functions exactly. name is the event kind
// ("audio_status"); payload is its data as a JSON object, crossing as text
// for the reason system events do — a Go map is not a bindable type. The
// return value is the patches of the render pass that follows, delivered on
// the event path, so a shell dispatches it on the same serial executor as
// its Trigger* calls and applies the result the same way. Nothing else is
// needed for a status tick to reach the screen.
//
// A malformed payload is dropped here with a log line and "[]" returned —
// the shell built the JSON, so the bug is on that side and a Go-side panic
// would only obscure it. An empty payload is a valid empty object.
func ReportHostEvent(name string, payload string) string {
	var data map[string]any
	if payload != "" {
		if err := json.Unmarshal([]byte(payload), &data); err != nil {
			log.Printf("grmob/mobile: dropping host event %q: %v", name, err)
			return "[]"
		}
	}
	return mgr.Dispatch("host event "+name, func() {
		coreReceiveHostEvent(name, data)
	})
}
