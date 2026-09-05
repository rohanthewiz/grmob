package todoapp

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// legacyFilterBar is the filter bar hand-rolled, as it was before the
// extraction into components.Chip. It is the acceptance check from the
// element-lessons plan (Workstream 3): the components package must be able to
// reproduce what apps were building by hand, byte for byte.
//
// It was verbatim from the pre-migration app until Chip's two state defaults
// were swapped — selected is now the theme's Button base and unselected the
// quiet Surface pill, where it used to be the other way around. That moved
// this bar's markup on purpose, so the reference moved with it: the
// unselected branch below is new, and it is the hand-written spelling of
// Chip's current unselected default.
//
// What the comparison proves is therefore narrower than it was, and still the
// half worth having: the widget builds exactly the row this app would build by
// hand today. It no longer says anything about the migration, which happened
// four widget releases ago and cannot be re-run.
func legacyFilterBar(active int, onSelect func(int)) core.View {
	return core.Row(
		core.Gap(8),
		core.For(filterLabels, func(label string, i int) core.View {
			// []core.PropsAndChildren, not []core.StyleProp: core.Button's
			// argument list widened, and Go will not spread the narrower
			// slice into the wider variadic. The elements are unchanged, so
			// the markup this builds — which is the whole point of the
			// comparison below — is byte-for-byte what it always was.
			styles := []core.PropsAndChildren{
				core.FontSize(13),
				core.Transition(200, core.EaseInOut),
				core.AccessibilityHint("Filters the task list"),
			}
			accLabel := "Show " + strings.ToLower(label) + " tasks"
			// The two state treatments, spelled out. Selected is the app's
			// own palette (the SelectedStyle the real bar passes); unselected
			// is Chip's default, read off the same theme the widget reads so
			// the comparison is against the palette and not against a hex
			// copied out of it.
			if i == active {
				styles = append(styles,
					core.BackgroundColor(colorAccent),
					core.TextColor(colorAccentInk),
				)
				accLabel += ", selected"
			} else {
				palette := core.DefaultTheme.Colors
				styles = append(styles,
					core.BackgroundColor(palette.Surface),
					core.TextColor(palette.TextPrimary),
					core.BorderWidth(1),
					core.BorderColor(palette.BorderColor()),
				)
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
