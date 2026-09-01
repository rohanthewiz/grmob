package verify

import (
	"strings"
	"testing"
)

// core.OnLongPress must be wired on every node type that can carry it,
// buttons included.
//
// Buttons are the gap this pins. Both renderers read the prop off containers
// and leaves through one shared path — grMobBox's onLongPress argument on
// iOS, gestureModifier on Android — but a Button draws its own control and
// takes neither, so the gesture was documented as wired on both natives while
// each button implementation read nothing but onClick. It was unreachable on
// the one node type that exists to be pressed.
//
// Source text, for the reason doc.go gives: nothing a compiler can see is
// wrong with a renderer that ignores a prop, and neither native runs under
// `go test ./...`. What is checked is that each button implementation reads
// the prop at all — the weakest useful claim, and exactly the one that was
// false.
func TestBothNativeButtonsWireLongPress(t *testing.T) {
	for _, pin := range []struct {
		file string
		// decl anchors the button implementation; read is the prop lookup it
		// must contain; gesture is the platform construct that consumes it.
		decl, read, gesture string
	}{
		{
			file:    swiftRenderer,
			decl:    "private struct GrMobButton",
			read:    `node.stringProp("onLongPress")`,
			gesture: "LongPressGesture(",
		},
		{
			// The long-press variant, not the material3 one: Compose's Button
			// has no long-click slot, so a button that declares the gesture
			// is rendered by a Surface + combinedClickable instead.
			file:    kotlinRenderer,
			decl:    "private fun GrMobLongPressButton(",
			read:    `node.stringProp("onLongPress")`,
			gesture: "onLongClick =",
		},
	} {
		src := declSource(t, pin.file, pin.decl)
		if !strings.Contains(src, pin.read) {
			t.Errorf("%s: %s never reads %s — core.OnLongPress on a Button does nothing on this platform",
				pin.file, pin.decl, pin.read)
		}
		if !strings.Contains(src, pin.gesture) {
			t.Errorf("%s: %s reads onLongPress but has no %s — the prop is read and dropped",
				pin.file, pin.decl, pin.gesture)
		}
	}
}

// The click that follows a long press must not also fire onClick.
//
// Compose gets this from combinedClickable, which decides between its two
// slots itself. SwiftUI does not — a `simultaneousGesture` is by name allowed
// to run alongside the Button's own tap — so the Swift button carries an
// explicit flag, and losing it would silently double-fire every long press as
// a long press plus a click. (The DOM runtime does the same thing with a
// dataset flag; wasm/verify pins that half.)
func TestSwiftButtonSuppressesTheTapAfterALongPress(t *testing.T) {
	src := declSource(t, swiftRenderer, "private struct GrMobButton")
	if !strings.Contains(src, "longPressFired") {
		t.Errorf("%s: GrMobButton no longer tracks whether a long press fired, so releasing "+
			"a long press runs onLongPress and then onClick for one gesture", swiftRenderer)
	}
}
