package todoapp

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// legacyFilterBar is the filter bar exactly as it was hand-rolled before the
// extraction into components.Chip — kept verbatim so the test below can
// prove the migration changed nothing observable. This is the acceptance
// check from the element-lessons plan (Workstream 3): the components package
// must be able to reproduce what apps were building by hand, byte for byte.
func legacyFilterBar(active int, onSelect func(int)) core.View {
	return core.Row(
		core.Gap(8),
		core.For(filterLabels, func(label string, i int) core.View {
			styles := []core.StyleProp{
				core.FontSize(13),
				core.Transition(200, core.EaseInOut),
				core.AccessibilityHint("Filters the task list"),
			}
			accLabel := "Show " + strings.ToLower(label) + " tasks"
			if i == active {
				styles = append(styles,
					core.BackgroundColor(colorAccent),
					core.TextColor(colorAccentInk),
				)
				accLabel += ", selected"
			}
			styles = append(styles, core.AccessibilityLabel(accLabel))
			return core.Keyed("filter-"+label,
				core.Button(label, func() { onSelect(i) }, styles...),
			)
		}),
	)
}

// TestFilterBarMatchesLegacyMarkup renders the Chip-based bar and the legacy
// hand-rolled bar on fresh contexts and compares their exported HTML.
// Fresh contexts matter: callback IDs are per-pass sequence numbers, so two
// bars that register the same handlers in the same order carry identical
// data-onclick attributes — any divergence in the output is a real
// structural or style difference, not ID noise.
func TestFilterBarMatchesLegacyMarkup(t *testing.T) {
	for active := range filterLabels {
		render := func(bar core.View) string {
			ctx := core.NewContext()
			ctx.BeginRenderPass()
			return htmlout.ExportHTML(bar.Render(ctx))
		}
		got := render(filterBar(active, func(int) {}))
		want := render(legacyFilterBar(active, func(int) {}))
		if got != want {
			t.Errorf("active=%d: Chip-based bar diverges from legacy markup\n--- chip ---\n%s\n--- legacy ---\n%s", active, got, want)
		}
	}
}
