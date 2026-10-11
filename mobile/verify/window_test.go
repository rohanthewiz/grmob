package verify

import (
	"os"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The window host event is spelled in four places that never compile
// together: core's decoder, the Kotlin WindowManager collector, the Swift
// size reader and the browser's segment reader. A shell that wrote
// "halfOpened" or "HORIZONTAL" would have its fold dropped by core's
// validation — deliberately silent, for forward compatibility — and a
// foldable would lay out as a plain tablet with a button on the crease. This
// holds each shell's spellings to core's exported values and to the keys
// core.receiveWindow reads.
//
// iOS reports no fold (nothing Apple ships folds), so only the size keys are
// required of it.
func TestWindowEventSpellingsAgree(t *testing.T) {
	// The colour scheme rides every shell's report (core.Window.ColorScheme),
	// so its key and both words are part of the size set.
	sizeKeys := []string{`"window"`, `"width"`, `"height"`, `"scheme"`,
		`"` + string(core.ColorSchemeLight) + `"`, `"` + string(core.ColorSchemeDark) + `"`}
	foldWords := []string{
		`"fold"`, `"state"`, `"orientation"`, `"separating"`, `"occluding"`, `"x"`, `"y"`,
		`"` + string(core.FoldFlat) + `"`,
		`"` + string(core.FoldHalfOpened) + `"`,
		`"` + string(core.FoldVertical) + `"`,
		`"` + string(core.FoldHorizontal) + `"`,
	}
	// The safe-area insets, on the two shells whose platform has system
	// bars. The browser sends no insets key at all — a page in a browser
	// window has the whole viewport, and an installed PWA's env() values
	// have no JS reading — and core decodes the absence as zero, which is
	// the same answer.
	insetWords := []string{`"insets"`, `"top"`, `"bottom"`, `"left"`, `"right"`}
	for _, shell := range []struct {
		file         string
		fold, insets bool
	}{
		{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "AppWindow.kt"), true, true},
		{nativeFile("ios", "GrMob", "App", "AppWindow.swift"), false, true},
		{nativeFile("wasm", "grmob-runtime.js"), true, false},
	} {
		raw, err := os.ReadFile(shell.file)
		if err != nil {
			t.Fatalf("reading %s: %v", shell.file, err)
		}
		src := string(raw)
		words := sizeKeys
		if shell.fold {
			words = append(append([]string{}, sizeKeys...), foldWords...)
		}
		if shell.insets {
			words = append(append([]string{}, words...), insetWords...)
		}
		for _, w := range words {
			// The browser runtime writes object keys unquoted (x: aRight),
			// so a key also counts when it appears as `key:`.
			bare := strings.Trim(w, `"`) + ":"
			if !strings.Contains(src, w) && !(strings.HasSuffix(shell.file, ".js") && strings.Contains(src, bare)) {
				t.Errorf("%s never writes %s", shell.file, w)
			}
		}
	}
}

// iOS's insets come from UIKit's window, not from the size reader (N-103).
//
// The size reader ignores the safe area so that it measures the whole window,
// and .ignoresSafeArea() consumes the very insets its proxy would report. For
// as long as the insets were read off that proxy, every iOS report said zero:
// lesson 4.21 showed "top 0, bottom 0, left 0, right 0" in both orientations
// on an iPhone 17 Pro. TutorialWindowUITests now watches a rotation move them;
// this holds the source to the reading that works, so a tidy-up that folds the
// insets back into the GeometryReader is a failing test rather than a quiet
// return to zero.
func TestIOSWindowInsetsAreTheWindowsOwn(t *testing.T) {
	file := nativeFile("ios", "GrMob", "App", "AppWindow.swift")
	src := codeIn(t, file)
	if !strings.Contains(src, "let bars = window.safeAreaInsets") {
		t.Errorf("%s: the insets are not read from the UIWindow's safeAreaInsets", file)
	}
	if strings.Contains(src, "geo.safeAreaInsets") {
		t.Errorf("%s: reads geo.safeAreaInsets, which is zero inside a reader that ignores the safe area", file)
	}
}
