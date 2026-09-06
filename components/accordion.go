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

	// HeadingLevel is where the Title sits in the screen's outline. Zero is
	// level 3 — a disclosure sits inside a section, one tier below a Card
	// title or a GroupedList band — and a screen built entirely of accordions
	// under an AppBar should say 2.
	//
	// It applies to the default header only. A Header replaces that content
	// and is the caller's to describe, on the same division Card.Title and
	// Card.Header draw.
	//
	// # The heading rides the title, not the row
	//
	// The row is the tap target and the title is the words, and the role goes
	// on the words for the reason GroupHeader's does: a band's row also holds
	// a count badge, and this row also holds a chevron, so a heading spanning
	// either would be named "▸ What is a hook" rather than "What is a hook".
	//
	// ARIA's own disclosure pattern nests them the other way — a heading
	// element *wrapping* a button that carries aria-expanded — and this is not
	// that, for two reasons worth separating. The wrapping is a shape core
	// could express (a Box around the Row) and it would not help: a heading
	// takes its name from its content, so a heading around this row is named
	// "▸ What is a hook" again, which is the thing putting the role on the
	// words avoids. The aria-expanded half is simply missing — core has no
	// vocabulary for an expanded state — so the header announces what it is
	// and not yet what state it is in.
	//
	// See headingLevel in heading.go for the package's outline and for how to
	// ask for a heading with no tier at all.
	HeadingLevel int

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
		title := []core.StyleProp{
			core.UseStyle(t.Typography.Body),
			core.FontWeight(core.Bold),
			core.FlexGrow(1),
		}
		title = append(title, headingProps(a.HeadingLevel, headingLevelSubsection)...)
		headerContent = core.Text(a.Title, title...)
	}

	header := core.Row(
		core.OnClick(func() { expanded.Set(!expanded.Get()) }),
		core.Gap(float64(t.Spacing.SM)),
		core.AccessibilityLabel(a.Title),
		core.AccessibilityHint("Expands or collapses the section"),
		// The chevron is deliberately *not* hidden, though it is decoration by
		// every other measure in this package. It is the only thing in the row
		// that says whether the section is open: core has no vocabulary for an
		// expanded state, so a reader that lost the glyph would be told a
		// heading and a hint and nothing about which way the disclosure is
		// pointing. Revisit when aria-expanded has a field.
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
