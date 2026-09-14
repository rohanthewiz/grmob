package verify

import (
	"strings"
	"testing"
)

// core.AccessibilityKeyShortcuts on the natives: a page-global chord (Control,
// Alt or Meta held) presses the control that declares it from a hardware
// keyboard.
//
// # Why pins and not a run
//
// Neither half can be driven from here. SwiftUI's keyboardShortcut fires from
// a UIKit key command on a device or simulator, and Compose's path starts at
// an Activity's dispatchKeyEvent. What can fail silently is the wiring, and
// every link below fails that way: a field nobody parses, a Button that never
// asks for its chord, an Activity that never offers the event, or a walk that
// stops honouring display none and would press a control on a screen behind
// a modal.
//
//	SwiftUI   str("AccessibilityKeyShortcuts") ─► GrMobButton
//	          .grMobKeyShortcut ─► grMobKeyChord ─► .keyboardShortcut
//	Compose   optString("AccessibilityKeyShortcuts") ─► MainActivity
//	          .dispatchKeyEvent ─► GrMobRuntime.handleKeyEvent
//	          ─► findKeyShortcut ─► GrMobKeyChord.pageGlobal/matches ─► click
func TestBothNativesPressADeclaredPageGlobalChord(t *testing.T) {
	swiftRenderer := nativeFile("ios", "GrMob", "Runtime", "Renderer.swift")
	kotlinRuntime := nativeFile("android", "app", "src", "main", "java", "com", "grmob",
		"runtime", "GrMobRuntime.kt")
	kotlinChord := nativeFile("android", "app", "src", "main", "java", "com", "grmob",
		"runtime", "GrMobKeyChord.kt")
	activity := nativeFile("android", "app", "src", "main", "java", "com", "grmob",
		"app", "MainActivity.kt")

	for _, pin := range []struct{ file, expr, why string }{
		{swiftStyle, `str("AccessibilityKeyShortcuts")`, "the field is parsed"},
		{swiftStyle, `func grMobKeyChord(_ spec: String) -> (key: KeyEquivalent, modifiers: EventModifiers)? {`,
			"the chord parser"},
		{swiftStyle, `guard !modifiers.subtracting(.shift).isEmpty else { continue }`,
			"a bare key (PageDown on a calendar arrow) is not made page-global"},
		{swiftStyle, `keyboardShortcut(chord.key, modifiers: chord.modifiers)`, "the SwiftUI shortcut itself"},
		{swiftRenderer, `.grMobKeyShortcut(s?.accessibilityKeyShortcuts ?? "")`, "GrMobButton asks for its chord"},
		{kotlinStyle, `optString("AccessibilityKeyShortcuts")`, "the field is parsed"},
		{kotlinChord, `get() = control || alt || meta || FUNCTION_KEY.matches(key)`,
			"the page-global rule, the same as the WASM runtime's isPageGlobalChord"},
		{kotlinRuntime, `fun handleKeyEvent(event: KeyEvent): Boolean {`, "the runtime's entry point"},
		{kotlinRuntime, `if (style?.display == "none" || style?.accessibilityHidden == true) return null`,
			"a hidden subtree is not a candidate"},
		{kotlinRuntime, `it.pageGlobal && it.matches(key, control, alt, meta, shift)`,
			"only page-global chords are answered"},
		{kotlinRuntime, `if (!target.disabled) click(target.node.stringProp("onClick"))`,
			"a disabled match takes the key and clicks nothing"},
		{kotlinRuntime, `val disabled = ancestorDisabled || style?.disabled == true`,
			"a control under a disabled ancestor counts as disabled, as LocalGrMobDisabled does"},
		{activity, `if (runtime?.handleKeyEvent(event) == true) return true`,
			"the Activity offers every key to the runtime"},
	} {
		if !strings.Contains(valuesIn(t, pin.file), pin.expr) {
			t.Errorf("%s: %q not found — %s. A declared shortcut would do nothing on "+
				"this platform, and nothing would say so", pin.file, pin.expr, pin.why)
		}
	}
}
