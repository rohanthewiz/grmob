package comps

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
	// # The heading wraps the control, which is ARIA's own accordion shape
	//
	// This is one of the two widgets in the package whose heading is not on
	// the words. The rest put the role on the Text node, because a row also
	// holds a badge or a chevron and a heading spanning the row would be named
	// "▸ What is a hook" rather than "What is a hook". Here the tier rides a
	// Box *around* the header row, and the row is a button:
	//
	//	Box    role=heading, aria-level=3, aria-label="What is a hook"
	//	  Row  role=button,  aria-expanded="false", aria-label="What is a hook"
	//	    Text "▸"        presentational, inside the button
	//	    Text "What is a hook"
	//
	// It took three tries to land there, both of the rejected ones looked
	// right, and the whole argument now lives on comps.disclosure — the
	// shared shape this widget and the collapsible GroupedList band are both
	// built out of. It moved there when the band arrived, because two copies
	// of a four-paragraph argument do not stay in step: the failure is not
	// that the second copy is wrong on the day it lands, it is that a later
	// fix to one leaves the other announcing something else.
	//
	// The wrapper is added only for the default header. A Header replaces the
	// content and is the caller's to describe, on the same division Card.Title
	// and Card.Header draw — so a custom header gets the button and its state
	// and no heading at all.
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

	var headerContent core.View
	if a.Header != nil {
		headerContent = a.Header
	} else {
		// No heading props on the words. The row around them is a button and
		// a button's children are presentational, so a heading role here would
		// be written into the document and pruned out of the accessibility
		// tree by every browser — the worst of both, since it would still look
		// correct in an export. The tier is on the wrapper the disclosure
		// builds; see the HeadingLevel doc.
		headerContent = core.Text(a.Title,
			core.UseStyle(t.Typography.Body),
			core.FontWeight(core.Bold),
			core.FlexGrow(1),
		)
	}

	// The shape itself — heading around button around chevron — is
	// comps.disclosure, shared with the collapsible GroupedList band.
	// The argument for it used to live in the HeadingLevel doc above and now
	// lives on that type; what stays here is the widget's own two decisions.
	//
	// The heading is added only for the default header, which is the first:
	// a Header slot is the caller's to describe, and stamping a heading named
	// by a Title they replaced would announce a row of controls as a section
	// title. The second is the tier — a disclosure sits inside a section, one
	// below a Card title or a GroupedList band.
	headerView := disclosure{
		Label:        a.Title,
		Hint:         "Expands or collapses the section",
		Expanded:     expanded.Get(),
		OnToggle:     func() { expanded.Set(!expanded.Get()) },
		Heading:      a.Header == nil,
		Level:        a.HeadingLevel,
		OwnLevel:     headingLevelSubsection,
		ChevronStyle: []core.StyleProp{core.UseStyle(t.Typography.Body)},
		ControlStyle: []core.StyleProp{core.Gap(float64(t.Spacing.SM))},
		Control:      []core.View{headerContent},
	}.view()

	items := make([]core.PropsAndChildren, 0, len(a.Style)+2)
	for _, sp := range a.Style {
		items = append(items, sp)
	}
	items = append(items, headerView)
	if a.Content != nil {
		items = append(items, core.If(expanded.Get(), a.Content))
	}

	return core.Column(items...).Render(ctx)
}
