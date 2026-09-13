package verify

import (
	"path/filepath"
	"strings"
	"testing"
)

// core.OnBack, which is spelled in three places that never compile together:
// core's prop, the Compose renderer that makes it a BackHandler, and the WASM
// runtime that makes it the browser's back button.
//
// # Why source text
//
// Both failures are silent. A renderer that never reads "onBack" compiles, and
// on device the only symptom is that back leaves the app from a pushed screen
// or an open drawer — which is exactly what it did before the prop existed, so
// nothing would look broken to anyone who had not seen it work. A web runtime
// that lets the prop fall to its generic on* branch attaches a listener for a
// "back" DOM event that never fires; no error, no dispatch, and the browser's
// back button leaves the page from every screen.
//
// # What is pinned
//
//	core   the prop key, "onBack" on the wire, with its ID from registerBack
//	kt     RenderNode, the funnel every node passes, reads it and hands it to
//	       BackHandler — not a per-type arm, which would leave every other node
//	       type unable to claim back
//	js     both the create and the update path branch on it before the generic
//	       on* branch, and popstate is what runs it. The behavior itself is
//	       wasm/verify/browserback_test.mjs's; this only keeps the wiring from
//	       vanishing in a run that skips Node.
func TestSystemBackIsWiredOnComposeAndTheWeb(t *testing.T) {
	core := valuesIn(t, filepath.Join("..", "..", "core", "behavioral_props.go"))
	for _, want := range []string{`n.Props["onBack"]`, "registerBackCallback(handler)"} {
		if !strings.Contains(core, want) {
			t.Errorf(`core.OnBack no longer writes %s — the renderers read "onBack", from the back_cb_ sequence`, want)
		}
	}

	node := valuesOf(t, kotlinRenderer, "fun RenderNode(")
	for _, want := range []string{`node.stringProp("onBack")`, "BackHandler {", "runtime.click(onBack)"} {
		if !strings.Contains(node, want) {
			t.Errorf("%s: RenderNode has no %s — system back leaves the app from every screen",
				kotlinRenderer, want)
		}
	}
	if !strings.Contains(valuesIn(t, kotlinRenderer), "import androidx.activity.compose.BackHandler") {
		t.Errorf("%s: BackHandler is not androidx.activity's", kotlinRenderer)
	}

	js := valuesIn(t, filepath.Join("..", "..", "wasm", "grmob-runtime.js"))
	for _, branch := range []string{`key === "onBack"`, `k === "onBack"`} {
		i := strings.Index(js, branch)
		if i < 0 {
			t.Errorf("wasm runtime: no %s branch — onBack falls to the generic on* listener", branch)
			continue
		}
		generic := `key.startsWith("on")`
		if strings.HasPrefix(branch, "k ") {
			generic = `k.startsWith("on")`
		}
		if g := strings.Index(js[i:], generic); g < 0 {
			t.Errorf("wasm runtime: %s is not followed by the generic %s branch it must precede", branch, generic)
		}
	}
	for _, want := range []string{`window.addEventListener("popstate", onBrowserBack)`, "backClaimants.add(el)"} {
		if !strings.Contains(js, want) {
			t.Errorf("wasm runtime: no %s — browser back is not wired to onBack", want)
		}
	}
}
