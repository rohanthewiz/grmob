package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// A fixed-size container exports as a flex box with a declared size and nothing
// else, which is what makes the browser's measurement cover this target too.
//
// # The question, and why this file answers only half of it
//
// What a container of a declared size does with a child bigger than it has four
// answers, and until now nothing anywhere had asked for any of them.
// wasm/verify/browser.mjs asks a real Chrome (check 11) and gets a per-axis
// answer sharper than anyone had assumed: the child is SQUEEZED along the
// container's main axis, because it is a flex item whose shrink factor defaults
// to 1 and whose automatic minimum, being empty, is 0 — and it SPILLS across
// the cross axis, because a declared size beats align-items: stretch and nothing
// shrinks a flex item across the line.
//
// That measurement is of the WASM runtime, which is one of this framework's two
// DOM targets. This is the other, and it cannot be measured the same way: an
// export is a string, and a string has no layout. What it can be held to is the
// three declarations the answer follows from — a browser given the same CSS
// lays out the same box — so this is the bridge rather than a second
// measurement, and it is written as one.
//
// # Why "and nothing else" is half the claim
//
// `overflow: hidden` would not change a single rect (it clips paint, not
// layout), and it would change what a reader SEES to something the browser check
// cannot detect. `min-width`/`min-height` would change the main-axis answer
// outright: the automatic minimum being 0 is the whole reason the squeeze
// happens, and an exporter that started writing a floor would make this target
// spill on both axes while the runtime kept squeezing on one.
func TestAFixedSizeBoxExportsTheDeclarationsTheBrowserMeasured(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	out := ExportHTML(core.Box(
		core.Width("120px"), core.Height("40px"),
		core.Box(core.Width("200px"), core.Height("80px")),
	).Render(ctx))

	// The declared size, and the flex context it is declared in. Both matter:
	// a width on a block box is a different question, since block layout has no
	// shrink factor at all and the child would spill on both axes.
	for _, want := range []string{
		"width:120px", "height:40px", "display:flex", "flex-direction:column",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("a fixed-size core.Box exports without %q:\n%s\n\n"+
				"wasm/verify/browser.mjs check 11 measures a real browser laying out "+
				"exactly this box, and this target inherits that answer only while it "+
				"emits the same declarations.", want, out)
		}
	}

	// And nothing that would change the answer. See the header: either of these
	// makes this target behave differently from the one that was measured,
	// silently.
	for _, unwanted := range []string{"overflow", "min-width", "min-height"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("a fixed-size core.Box exports a %s declaration:\n%s\n\n"+
				"The browser check's per-axis answer rests on the child being a flex "+
				"item with no floor and on nothing clipping it. This target no longer "+
				"lays out the box that was measured.", unwanted, out)
		}
	}
}
