package verify

import (
	"strings"
	"testing"
)

// A horizontal Scroll with a FlexGrow child on SwiftUI: a ScrollView proposes
// its content an unbounded width, so the grower had no free space and lesson
// 4.8's footer count sat right after its chip on the simulator, where the web
// and Compose (GrMobGrowStrip) push it to the far edge. GrMobScroll's
// horizontal body lays such a strip out as a flex row proposed
// max(ideal, viewport) wide. The measurement is ios/GrMobUITests'
// TutorialFooterStripUITests; these pins hold the wiring between runs.
func TestSwiftUIStripWithAGrowerDividesTheViewport(t *testing.T) {
	renderer := nativeFile("ios", "GrMob", "Runtime", "Renderer.swift")
	for _, c := range []struct{ decl, expr, why string }{
		{"@ViewBuilder private var horizontal: some View {",
			"let grows = node.children.contains { ($0.style?.flexGrow ?? 0) > 0 }",
			"only a strip with a grower changes layout; the rest keep the HStack"},
		{"@ViewBuilder private var horizontal: some View {", "GrMobStripContentLayout(viewport: viewport) {",
			"the row is proposed at least the viewport width"},
		{"@ViewBuilder private var horizontal: some View {", "GrMobFlexStack(axis: .horizontal, style: node.style) {",
			"a flex row, which divides free space among the growers and reads AlignItems"},
		{"@ViewBuilder private var horizontal: some View {", "viewport = grows ? width : 0",
			"the viewport width is read off the ScrollView itself, by onGeometryChange (a preference arrived as 0)"},
		{"private func stripWidth(", "return max(ideal, viewport)",
			"free space only when the row is shorter than the viewport; a longer one scrolls"},
	} {
		if !strings.Contains(codeOf(t, renderer, c.decl), c.expr) {
			t.Errorf("%s: %s has no %q — %s", renderer, c.decl, c.expr, c.why)
		}
	}
}
