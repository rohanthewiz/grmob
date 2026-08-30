package main

import (
	"os"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

func AppLayoutExample() core.View {
	return core.WithTheme(core.DefaultTheme,
		core.SafeArea(
			core.Column(
				Header(),
				core.Spacer(8),
				BodySection(),
				core.Spacer(8),
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
			core.Text("Welcome, Ismael!", core.FontSize(18)),
			core.Spacer(8),
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
	node := AppLayoutExample().Render(ctx)
	html := htmlout.ExportHTML(node)
	_ = os.WriteFile("layout.html", []byte(html), 0644)
}
