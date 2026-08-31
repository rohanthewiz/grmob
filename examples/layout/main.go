package main

import (
	"os"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

func AppLayoutExample() core.View {
	return core.WithTheme(core.DefaultTheme,
		core.SafeArea(
			// Gap on the container replaces interleaved Spacers: the spacing
			// here is uniform, so one prop expresses it instead of N-1 filler
			// views — fewer nodes to diff, and no way to add a child and
			// forget its separator.
			core.Column(
				core.Gap(8),
				Header(),
				BodySection(),
				Footer(),
			),
		),
	)
}

func Header() core.View {
	return core.Row(
		core.BackgroundColor("#6200EE"),
		core.Padding(16),
		core.Text("My App", core.FontSize(20), core.TextColor("#FFFFFF")),
	)
}

func BodySection() core.View {
	return core.Row(
		core.Column(
			core.BackgroundColor("#F5F5F5"),
			core.Padding(16),
			core.Gap(8),
			core.Text("Welcome, Ismael!", core.FontSize(18)),
			core.Text("Here's your dashboard overview."),
		),
	)
}

func Footer() core.View {
	return core.Row(
		core.BackgroundColor("#EEEEEE"),
		core.Padding(12),
		core.Align(core.AlignCenter),
		core.Text("© 2025 GrMob Labs", core.FontSize(12), core.TextColor("#666")),
	)
}
func main() {
	ctx := core.NewContext()
	// core.Render is the documented entry point: it resets the hook cursor
	// before rendering, so it is safe to call on every pass, not just the
	// first. This exporter renders once, but copying view.Render(ctx) into a
	// host that re-renders is exactly how state starts reading the wrong slot.
	node := core.Render(ctx, AppLayoutExample())
	html := htmlout.ExportHTML(node)
	_ = os.WriteFile("layout.html", []byte(html), 0644)
}
