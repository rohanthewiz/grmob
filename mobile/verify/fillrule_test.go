package verify

import (
	"strings"
	"testing"
)

// core.FillEvenOdd on both native canvases. Read off the shape's props inside
// the drawing loop, which neither native test harness runs (the geometry
// fixtures stop at the decoded calls), so a renderer that dropped the rule
// would fill every ring solid with nothing failing.
func TestBothNativeCanvasesApplyTheFillRule(t *testing.T) {
	swift := valuesIn(t, swiftRenderer)
	if !strings.Contains(swift, `FillStyle(eoFill: props["fillRule"] as? String == "evenodd")`) {
		t.Errorf("%s: GrMobCanvas does not pass the fill rule to ctx.fill", swiftRenderer)
	}
	kotlin := valuesIn(t, nativeFile("android", "app", "src", "main", "java", "com", "grmob",
		"runtime", "GrMobCanvas.kt"))
	if !strings.Contains(kotlin, `if (props["fillRule"] == "evenodd") path.fillType = PathFillType.EvenOdd`) {
		t.Error("GrMobCanvas.kt does not set PathFillType.EvenOdd from the shape's fillRule")
	}
}
