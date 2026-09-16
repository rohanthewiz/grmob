package verify

import (
	"strings"
	"testing"
)

// core.AccessibilityKeyShortcuts on the natives: a page-global chord (Control,
// Alt or Meta held, or an F-key) presses the control that declares it from a
// hardware keyboard.
//
// # Why pins and not a run
//
// Neither half can be driven from here. SwiftUI's keyboardShortcut fires from
// a UIKit key command on a device or simulator, and Compose's path starts at
// an Activity's dispatchKeyEvent. Both are driven on the simulator and the
// emulator by lesson 2.2's button (ios/GrMobUITests/TutorialKeyShortcutsUITests
// and an adb key combination); these pins hold the wiring between runs. What can fail silently is the wiring, and
// every link below fails that way: a field nobody parses, a Button that never
// asks for its chord, an Activity that never offers the event, or a walk that
// stops honouring display none and would press a control on a screen behind
// a modal.
//
//	SwiftUI   str("AccessibilityKeyShortcuts") ─► GrMobButton
//	          .grMobKeyShortcut ─► grMobKeyChords ─► .keyboardShortcut on the
//	          Button (first chord), and on an invisible Button behind it (each
//	          further chord); a tappable box's GrMobGestures
//	          .grMobBoxKeyShortcuts puts every chord on an invisible Button
//	iOS       F-keys, which SwiftUI does not deliver: GrMobApp.init ─►
//	          GrMobFunctionKeys (GCKeyboard.keyChangedHandler) ─►
//	          GrMobRuntime.pressFunctionKey ─► functionKeyTarget ─► click
//	Compose   optString("AccessibilityKeyShortcuts") ─► MainActivity
//	          .dispatchKeyEvent ─► GrMobRuntime.handleKeyEvent
//	          ─► findKeyShortcut ─► GrMobKeyChord.pageGlobal/matches ─► click
func TestBothNativesPressADeclaredPageGlobalChord(t *testing.T) {
	swiftRenderer := nativeFile("ios", "GrMob", "Runtime", "Renderer.swift")
	swiftRuntime := nativeFile("ios", "GrMob", "Runtime", "GrMobRuntime.swift")
	swiftFunctionKeys := nativeFile("ios", "GrMob", "App", "GrMobFunctionKeys.swift")
	swiftApp := nativeFile("ios", "GrMob", "App", "GrMobApp.swift")
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
		{swiftStyle, `if grMobFunctionKeyNumber(name) != nil { continue }`,
			"an F-key chord is left to GrMobFunctionKeys, since a KeyEquivalent for one never fires"},
		{swiftFunctionKeys, `keyboard.keyboardInput?.keyChangedHandler = {`, "F-key presses are heard with nothing focused"},
		{swiftFunctionKeys, `MainActor.assumeIsolated { _ = runtime?.pressFunctionKey(chord) }`,
			"each F-key press is offered to the tree"},
		{swiftApp, `GrMobFunctionKeys.attach(to: runtime)`, "the listener is installed at all"},
		{swiftRuntime, `if style?.display == "none" || style?.accessibilityHidden == true { return nil }`,
			"a hidden subtree is not a candidate for an F-key either"},
		{swiftRuntime, `if !target.disabled { click(target.node.stringProp("onClick")) }`,
			"a disabled F-key match takes the key and clicks nothing"},
		{swiftStyle, `modifier(GrMobVisibleChord(key: first.key, modifiers: first.modifiers))`,
			"the SwiftUI shortcut itself, through the modifier that can ask whether the subtree is hidden"},
		{swiftStyle, `content.keyboardShortcut(key, modifiers: modifiers)`,
			"and GrMobVisibleChord attaching it"},
		{swiftStyle, `.keyboardShortcut(chord.key, modifiers: chord.modifiers)`,
			"every further chord, which a Button's one keyboardShortcut cannot hold"},
		{swiftRenderer, `.grMobKeyShortcut(s?.accessibilityKeyShortcuts ?? "", press: press)`,
			"GrMobButton asks for its chords, with the action a tap runs"},
		{swiftStyle, `keyShortcuts: s?.accessibilityKeyShortcuts ?? ""))`,
			"a tappable box is handed its chords, as the web and Compose press any node with an onClick"},
		{swiftStyle, `.grMobBoxKeyShortcuts(onTap.isEmpty ? "" : keyShortcuts) { dispatch?(onTap) }`,
			"each of a box's chords presses its tap, and only past the disabled branch"},
		{swiftStyle, `background { GrMobShortcutButtons(chords: chords, press: press) }`,
			"a box has no keyboardShortcut slot, so every chord gets an invisible Button"},
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

// A chord inside a core.AccessibilityHidden subtree does not fire on iOS.
//
// # The gap this closes
//
// `.accessibilityHidden(true)` prunes a subtree from VoiceOver and from
// nothing else, so a chord declared inside one stayed live on a hardware
// keyboard: a screen behind a modal, or a shut Drawer panel, answering a
// shortcut for a control the reader could not reach. The web skips a hidden
// subtree when it routes a chord, Compose does, and this runtime's own F-key
// walk does (the arm above holds that line) — SwiftUI's keyboardShortcut was
// the one route left, because SwiftUI resolves it rather than a walk of ours.
//
// # Why the environment is the subject
//
// The fact is about ANCESTORS, and a renderer building one node's view has the
// node and not the path above it. The environment is what SwiftUI already
// propagates down a view tree, so the check is that the same branch which
// calls accessibilityHidden also sets it, and that both chord routes read it:
//
//	grMobAccessibility   sets grMobAccessibilityHidden beside accessibilityHidden
//	GrMobVisibleChord    a Button's FIRST chord — a View extension cannot read
//	                     the environment, so the shortcut goes on through a
//	                     ViewModifier that can
//	GrMobShortcutButtons a Button's further chords, and every chord of a
//	                     tappable box
//
// Each route failing alone is silent: the Button's first chord is the common
// one and the further chords are the rarer, so a fix to either half on its own
// looks like a fix.
func TestASwiftUIChordBehindAHiddenSubtreeDoesNotFire(t *testing.T) {
	swiftStyle := nativeFile("ios", "GrMob", "Runtime", "GrMobStyle.swift")
	code := codeIn(t, swiftStyle)
	for _, c := range []struct{ expr, why string }{
		{`.environment(\.grMobAccessibilityHidden, true)`,
			"the hidden branch of grMobAccessibility publishes the fact to its subtree"},
		{`@Environment(\.grMobAccessibilityHidden) private var subtreeHidden`,
			"a chord route reads it"},
		{`modifier(GrMobVisibleChord(key: first.key, modifiers: first.modifiers))`,
			"a Button's first chord goes through the modifier that can ask"},
	} {
		if !strings.Contains(code, c.expr) {
			t.Errorf("%s: no %s — %s.\n\nWithout it a shortcut declared behind a "+
				"modal, or inside a shut Drawer panel, still answers a hardware "+
				"keyboard while nothing in the subtree can be reached.",
				swiftStyle, c.expr, c.why)
		}
	}
	// Both routes, not one. Counted rather than matched once: the two reads
	// are identical text, and a single one would satisfy a Contains.
	if n := strings.Count(code, `@Environment(\.grMobAccessibilityHidden) private var subtreeHidden`); n != 2 {
		t.Errorf("%s: %d reader(s) of grMobAccessibilityHidden, want 2 — "+
			"GrMobVisibleChord for a Button's first chord and "+
			"GrMobShortcutButtons for every other one. One of the two routes is "+
			"still live behind a hidden subtree.", swiftStyle, n)
	}
	// And the gate is the absence of the shortcut, not a .disabled: a
	// disabled Button is greyed out for everybody, which is a visible change
	// to a screen that is merely hidden from a reader.
	guard := codeOf(t, swiftStyle, "struct GrMobVisibleChord: ViewModifier {")
	if strings.Contains(guard, ".disabled(") {
		t.Errorf("%s: GrMobVisibleChord turns the chord off with .disabled, which "+
			"also greys the control out for sighted users. Not attaching the "+
			"shortcut is the change that is invisible to everyone but the "+
			"keyboard.", swiftStyle)
	}
}
