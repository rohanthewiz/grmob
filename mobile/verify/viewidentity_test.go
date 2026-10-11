package verify

import (
	"strings"
	"testing"
)

// Both natives keep a node's identity when a prop toggles (N-106).
//
// A @ViewBuilder branch is part of a SwiftUI view's structural identity, so a
// modifier that writes `if flag { content } else { content.something() }`
// rebuilds the whole subtree whenever the flag flips: state is dropped and no
// Transition can animate, because the old view is gone and a new one arrives
// already in place. comps.Drawer flips two such flags on every open and close,
// AccessibilityHidden on both layers and an OnEscape claim on the panel
// layer. On the iOS 26.5 simulator the panel went from shut to open within one
// 60fps frame, where Drawer documents a 250ms slide. Each fix alone left it
// that way, measured on a screen recording; both together give the slide
// (seven intermediate positions over about 300ms), and Reduce Motion still
// snaps it. These pins keep both flags arguments rather than branches.
//
// Compose had the same fault in another form: RenderNode called its content
// at one call site when no CompositionLocal changed and at another inside
// CompositionLocalProvider when one did, and Compose keys state by call site.
// The Drawer's layers flip Inert, which is one of those locals, so the panel's
// animation state was composed again at its target. At ten times the
// animator scale the emulator showed it fully open in the first frame.
func TestTogglesKeepTheNodesIdentityOnBothNatives(t *testing.T) {
	style := codeIn(t, swiftStyle)
	if !strings.Contains(style, ".accessibilityHidden(hidden)") {
		t.Errorf("%s: grMobAccessibility no longer passes hidden as an argument", swiftStyle)
	}
	if strings.Contains(style, "if s?.accessibilityHidden == true {") {
		t.Errorf("%s: grMobAccessibility branches on AccessibilityHidden again; a node "+
			"that is hidden or shown is rebuilt instead of updated, and a drawer opens without its slide",
			swiftStyle)
	}
	renderer := codeIn(t, swiftRenderer)
	if !strings.Contains(renderer, "content.background {\n            if !id.isEmpty {") {
		t.Errorf("%s: GrMobEscapeClaim must keep its condition inside the background; "+
			"a branch around content rebuilds a node that gains or loses its claim", swiftRenderer)
	}

	kotlin := codeIn(t, kotlinRenderer)
	if !strings.Contains(kotlin, "CompositionLocalProvider(*provided.toTypedArray()) { RenderNodeContent(node, mods) }") {
		t.Errorf("%s: RenderNode no longer composes its content through the one provider call", kotlinRenderer)
	}
	if strings.Contains(kotlin, "if (!disable && !bound && !boundWidth && !named && !inert") {
		t.Errorf("%s: RenderNode calls its content from two places again; a node whose "+
			"local flips (Inert, Disabled) is composed again and loses its state and its animation",
			kotlinRenderer)
	}
}
