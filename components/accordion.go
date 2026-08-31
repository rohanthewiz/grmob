package components

import "github.com/rohanthewiz/grmob/core"

// Accordion is a collapsible section: a tappable header that shows or hides
// its Content.
//
// It owns its expanded/collapsed state via NewState, which makes it the one
// widget in this package with hook obligations: render an Accordion
// unconditionally, in a stable position, every pass — exactly the rules for
// calling NewState directly (core.SetDebugMode reports violations as
// cursor-drift concerns). Content, on the other hand, is only rendered while
// expanded, so it must not contain hooks of its own: they would come and go
// with the toggle, which is the conditional-hook bug. Interactive,
// hook-free content (buttons, inputs bound to parent state) is fine — its
// callbacks re-register on every pass the content is visible.
type Accordion struct {
	Title string
	// Header replaces the default chevron+Title header content when set.
	// The tap target and toggle behavior stay with the Accordion either way.
	Header  core.View
	Content core.View
	// InitiallyExpanded seeds the state on the first pass only; after that
	// the accordion follows the user's taps.
	InitiallyExpanded bool
	// Style is applied to the outer column.
	Style []core.StyleProp
}

func (a Accordion) Render(ctx *core.Context) *core.Node {
	expanded := core.NewState(ctx, a.InitiallyExpanded)
	t := ctx.Theme()

	chevron := "▸"
	if expanded.Get() {
		chevron = "▾"
	}

	var headerContent core.View
	if a.Header != nil {
		headerContent = a.Header
	} else {
		headerContent = core.Text(a.Title,
			core.UseStyle(t.Typography.Body),
			core.FontWeight(core.Bold),
			core.FlexGrow(1),
		)
	}

	header := core.Row(
		core.OnClick(func() { expanded.Set(!expanded.Get()) }),
		core.Gap(float64(t.Spacing.SM)),
		core.AccessibilityLabel(a.Title),
		core.AccessibilityHint("Expands or collapses the section"),
		core.Text(chevron, core.UseStyle(t.Typography.Body)),
		headerContent,
	)

	items := make([]core.PropsAndChildren, 0, len(a.Style)+2)
	for _, sp := range a.Style {
		items = append(items, sp)
	}
	items = append(items, header)
	if a.Content != nil {
		items = append(items, core.If(expanded.Get(), a.Content))
	}

	return core.Column(items...).Render(ctx)
}
