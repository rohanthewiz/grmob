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
	sizeKeys := []string{`"window"`, `"width"`, `"height"`}
	foldWords := []string{
		`"fold"`, `"state"`, `"orientation"`, `"separating"`, `"occluding"`, `"x"`, `"y"`,
		`"` + string(core.FoldFlat) + `"`,
		`"` + string(core.FoldHalfOpened) + `"`,
		`"` + string(core.FoldVertical) + `"`,
		`"` + string(core.FoldHorizontal) + `"`,
	}
	for _, shell := range []struct {
		file string
		fold bool
	}{
		{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "AppWindow.kt"), true},
		{nativeFile("ios", "GrMob", "App", "AppWindow.swift"), false},
		{nativeFile("wasm", "grmob-runtime.js"), true},
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
