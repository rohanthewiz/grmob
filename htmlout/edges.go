package htmlout

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// EdgeCSS serializes a core.EdgeInsets into the four-value CSS shorthand
// ("top right bottom left"), resolving the Horizontal/Vertical shorthand
// fields the way the native renderers do.
//
// # The resolution rule
//
// core.EdgeInsets carries six fields, not four: the per-side Top/Right/
// Bottom/Left plus a Horizontal/Vertical pair the DSL's PaddingHorizontal /
// PaddingVertical props write. A side that was not set explicitly takes its
// value from the shorthand for its axis:
//
//	top    = Top    != 0 ? Top    : Vertical
//	bottom = Bottom != 0 ? Bottom : Vertical
//	left   = Left   != 0 ? Left   : Horizontal
//	right  = Right  != 0 ? Right  : Horizontal
//
// "Set explicitly" means non-zero, which is the one place the rule is lossy:
// PaddingHorizontal(16) plus PaddingLeft(0) cannot ask for a zero left inset,
// because a zero Left is indistinguishable from an unset one. Both natives
// have the same limitation for the same reason (a Go zero value carries no
// "was it set?" bit), and reproducing it exactly is the point — an inset that
// resolves one way on device and another on the web is worse than one that is
// uniformly lossy.
//
// This is a restatement of GrMobStyle.swift's parseEdges and GrMobStyle.kt's
// parseEdges, which have honored the shorthand since they were written. Until
// this function existed the two web targets read the four per-side fields
// only, so core.PaddingHorizontal(16) applied cleanly, rendered as 16px of
// padding on both natives, and as nothing at all in the browser. The WASM
// runtime's copy is edgeToCSS in wasm/grmob-runtime.js.
func EdgeCSS(e core.EdgeInsets) string {
	return fmt.Sprintf("%dpx %dpx %dpx %dpx",
		edgeSide(e.Top, e.Vertical),
		edgeSide(e.Right, e.Horizontal),
		edgeSide(e.Bottom, e.Vertical),
		edgeSide(e.Left, e.Horizontal),
	)
}

// edgeSide picks one side's value: the explicit per-side field when it is set,
// otherwise the shorthand for that side's axis.
func edgeSide(explicit, shorthand int) int {
	if explicit != 0 {
		return explicit
	}
	return shorthand
}
