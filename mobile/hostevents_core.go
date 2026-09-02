package mobile

import "github.com/rohanthewiz/grmob/core"

// coreReceiveHostEvent is the seam hostevents.go dispatches through. A var
// rather than a direct call so the test can observe delivery without a
// registered app; the production value is core's own entry point.
var coreReceiveHostEvent = core.ReceiveHostEvent
