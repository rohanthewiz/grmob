package verify

import (
	"strings"
	"testing"
)

// core.Box stacks its children; it is not an overlay.
//
// This is the divergence the gap sweep exposed rather than a gap bug of its
// own. Box was a Compose `Box` and a SwiftUI `ZStack` — both of which lay
// children on top of one another at the top-start corner — while both DOM
// targets stack them down the page (the WASM runtime lists Box in
// STACK_CONTAINERS). core.Box is documented as one of the flex-style
// containers, sharing Row/Column/Card/List's argument contract and differing
// from Column only in carrying no theme base, so the natives were the outlier
// and two children of one Box drew on top of each other on device.
//
// The pin is that Box shares Column's dispatch arm on both renderers, which
// is the whole of the fix and the thing a well-meaning "Box should be a
// ZStack, that's what Compose calls it" edit would undo.
func TestNativeBoxIsAVerticalStackNotAnOverlay(t *testing.T) {
	for _, pin := range []struct {
		file, arm string
		// overlay is the construct Box must no longer be built with. Both
		// names stay in use elsewhere in each renderer — CameraView is a real
		// overlay on both — so this is checked on the arm, not the file.
		overlay string
	}{
		{swiftRenderer, `case "Column", "Card", "Box":`, "ZStack"},
		{kotlinRenderer, `"Column", "Card", "Box" ->`, "Box("},
	} {
		src := readNative(t, pin.file)
		if !strings.Contains(src, pin.arm) {
			t.Errorf("%s: no %s arm — Box no longer shares Column's dispatch, so it is "+
				"either an overlay again or a third layout of its own", pin.file, pin.arm)
			continue
		}
		// The arm itself must not build the overlay it replaced. Bounded to
		// the rest of the line, since these are single-line dispatch arms.
		line := src[strings.Index(src, pin.arm):]
		if end := strings.IndexByte(line, '\n'); end >= 0 {
			line = line[:end]
		}
		if strings.Contains(line, pin.overlay) {
			t.Errorf("%s: the Box arm builds a %s — its children draw on top of each other, "+
				"while both DOM targets stack them", pin.file, pin.overlay)
		}
	}
}
