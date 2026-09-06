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
	// # The heading wraps the control, which is ARIA's own accordion shape
	//
	// This is the one widget in the package whose heading is not on the words.
	// The rest put the role on the Text node, because a row also holds a badge
	// or a chevron and a heading spanning the row would be named "▸ What is a
	// hook" rather than "What is a hook". Here the tier rides a Box *around*
	// the header row, and the row is a button.
	//
	// It took three tries to land there and the middle one is worth keeping,
	// because it looked right.
	//
	// The row started unroled and named, which meant its name was announced on
	// both natives and dropped on both web targets — ARIA prohibits an
	// accessible name on the `generic` role a <div> carries. The exporters'
	// supplied core.RoleGroup closed that, and the heading stayed on the words
	// because a group leaves its children readable where role="button" — ARIA's
	// own disclosure control — makes them presentational.
	//
	// What that arrangement could not do is say whether the section is open.
	// aria-expanded is not defined for `group`; it is defined for button, link,
	// listbox, row, columnheader and tab, and of those only button is a
	// disclosure. So the choice was between a header that announces its name
	// and a header that announces its state, and a reader needs both: a
	// "group" is not something a reader is told they can press at all.
	//
	// ARIA's disclosure pattern resolves it by nesting the other way — a
	// heading element wrapping a button — and the objection this widget used to
	// raise against that was a naming one: a heading takes its name from its
	// content, so a heading around this row would be named "▸ What is a hook".
	// That is true of a heading with no name of its own. core.AccessibilityLabel
	// overrides content-derived naming, and the Title is right there. So the
	// wrapper is named explicitly and the chevron never reaches it.
	//
	//	Box    role=heading, aria-level=3, aria-label="What is a hook"
	//	  Row  role=button,  aria-expanded="false", aria-label="What is a hook"
	//	    Text "▸"        presentational, inside the button
	//	    Text "What is a hook"
	//
	// A reader now hears the title twice — once as an outline entry it can jump
	// to, once as a control it can press and whose state it is told. That is
	// what every accessible accordion on the web does, and it is what the APG
	// pattern produces for the same reason: the two facts belong to two
	// different elements.
	//
	// The wrapper is added only for the default header. A Header replaces the
	// content and is the caller's to describe, on the same division Card.Title
	// and Card.Header draw — so a custom header gets the button and its state
	// and no heading at all, which is exactly what it got before.
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
		// No heading props on the words any more. The row around them is a
		// button and a button's children are presentational, so a heading role
		// here would be written into the document and pruned out of the
		// accessibility tree by every browser — the worst of both, since it
		// would still look correct in an export. The tier is on the wrapper
		// below; see the HeadingLevel doc.
		headerContent = core.Text(a.Title,
			core.UseStyle(t.Typography.Body),
			core.FontWeight(core.Bold),
			core.FlexGrow(1),
		)
	}

	header := core.Row(
		core.OnClick(func() { expanded.Set(!expanded.Get()) }),
		core.Gap(float64(t.Spacing.SM)),
		// The row is the control: it is the tap target, and ARIA's disclosure
		// pattern is a button carrying aria-expanded. Both halves have to be
		// stated together — the state is dropped on any role that is not one
		// of the six ARIA scopes it to, and `group`, which this row used to
		// take, is not among them.
		core.AccessibilityRole(core.RoleButton),
		// Stated on every pass, open or shut. core.ExpandedClosed is not the
		// same as saying nothing: a collapsed section that answers nothing is
		// announced as an ordinary button, and "collapsed" is the whole of
		// what tells a reader there is something behind it.
		core.AccessibilityExpanded(core.ExpandedWhen(expanded.Get())),
		core.AccessibilityLabel(a.Title),
		core.AccessibilityHint("Expands or collapses the section"),
		// The chevron is presentational now, and by construction rather than
		// by choice — it is inside a button, whose children a reader does not
		// descend into. It used to be deliberately left un-hidden because it
		// was the only thing on the screen that said which way the disclosure
		// was pointing; aria-expanded says that now, and says it in a channel
		// that does not depend on a reader pronouncing "▸".
		core.Text(chevron, core.UseStyle(t.Typography.Body)),
		headerContent,
	)

	// The outline entry, wrapped around the control. Only for the default
	// header: a Header slot is the caller's to describe, and stamping a
	// heading named by a Title they replaced would announce a row of controls
	// as a section title.
	//
	// core.Box and not core.Column — a Column arrives carrying the theme's
	// screen inset, and this wrapper must be geometrically invisible.
	headerView := core.View(header)
	if a.Header == nil {
		wrapper := headingProps(a.HeadingLevel, headingLevelSubsection)
		// Named explicitly, which is the whole reason this shape works. A
		// heading with no name of its own takes one from its content, and its
		// content here is the chevron plus the title.
		wrapper = append(wrapper, core.AccessibilityLabel(a.Title))
		headerView = core.Box(append(asProps(wrapper), header)...)
	}

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
