package social

import "github.com/rohanthewiz/grmob/core"

func TabButton(icon, tab string, selected core.State[string]) core.View {
	isActive := selected.Get() == tab

	return core.Button(icon, func() {
		selected.Set(tab)
	},
		core.FontSize(20),
		core.TextColor(ifThen(isActive, "#007AFF", "#555")),
		core.Padding(12),
		core.Align(core.AlignCenter),
	)
}

func ifThen(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
