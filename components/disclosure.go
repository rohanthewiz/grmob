package components

import "github.com/rohanthewiz/grmob/core"

// disclosure is ARIA's disclosure shape, built once for the widgets that need
// it: a heading wrapping a button that carries aria-expanded.
//
//	Box    role=heading, aria-level=N, aria-label=<Label>
//	  Row  role=button,  aria-expanded=<Expanded>, aria-label=<Label>
//	    Text "▸"        presentational, inside the button
//	    <Control>
//
// # Why a shared shape rather than two copies
//
// components.Accordion arrived at this arrangement over three tries and its
// HeadingLevel doc records all of them, because the two obvious alternatives
// both look right:
//
//   - The row unroled and named. Announced on both natives, dropped on both
//     web targets — ARIA prohibits an accessible name on the `generic` role a
//     <div> carries.
//   - The row as core.RoleGroup with the heading on the words inside it. The
//     name comes back, and the state does not: aria-expanded is defined for
//     button, link, listbox, row, columnheader and tab, and `group` is not
//     among them. So the header could announce what it is called or announce
//     whether it is open, and a reader needs both.
//
// The resolution is to nest the other way. A heading takes its name from its
// content, which is what used to rule this out — a heading around the row
// would be named "▸ January 2026" — and core.AccessibilityLabel overrides
// content-derived naming, so the wrapper is named explicitly and the chevron
// never reaches it. A reader hears the title twice: once as an outline entry
// it can jump to, once as a control it can press and whose state it is told.
//
// That argument is four paragraphs and it was true for one widget. The second
// consumer is what makes it worth stating in one place — the failure mode of
// two copies is not that the second one is wrong on the day it lands, it is
// that a later fix to one of them leaves the other announcing something else.
// TestBothDisclosuresBuildTheSameShape holds the two together.
//
// # The pairing this type enforces
//
// Expanded and OnToggle are both required. A stated expansion with no handler
// is announced on both web targets and is silently nothing on Android, where
// Compose wires its expand/collapse semantics actions to the node's own click
// callback and offers neither without one — which is why core.AuditTree
// reports it as ConcernInertDisclosure. Building the shape from one struct is
// what makes "these two travel together" a fact about the type rather than a
// note in two doc comments.
type disclosure struct {
	// Label names both the heading and the button. One string deliberately:
	// they are the same thing said twice, to two different consumers.
	Label string
	// Hint describes what pressing does ("Expands or collapses the section").
	Hint string

	// Expanded is whether the content behind the control is showing. Stated
	// on every pass, open or shut — core.ExpandedClosed is not the same as
	// saying nothing, and "collapsed" is the whole of what tells a reader
	// there is something behind the control.
	Expanded bool
	// OnToggle flips it. Required; see the type comment.
	OnToggle func()

	// Heading adds the outline wrapper. False builds the button alone, which
	// is what a widget whose header content the caller replaced should do —
	// stamping a heading named by a Title they did not write would announce a
	// row of controls as a section title.
	Heading bool
	// Level is the tier the caller asked for and OwnLevel the widget's own
	// default; a zero Level takes OwnLevel. See headingLevel in heading.go.
	Level, OwnLevel int
	// HeadingStyle is applied to the wrapper. The band uses it to grow.
	HeadingStyle []core.StyleProp

	// ChevronStyle types the glyph. Supplied rather than fixed because the
	// two callers sit at different tiers of the type scale, and a chevron
	// that does not match the words beside it reads as a bullet.
	ChevronStyle []core.StyleProp

	// ControlStyle is applied to the button row, and Control holds its
	// children in order after the chevron.
	//
	// Split rather than one core.PropsAndChildren slice, even though the
	// container's dispatch would sort them out: a slice whose props landed
	// after the chevron read as though the gap applied to something, and the
	// two halves of a container's argument list are the one place in this
	// package where "it works either way" is not a reason to leave it
	// ambiguous.
	ControlStyle []core.StyleProp
	Control      []core.View
}

// view builds the shape.
func (d disclosure) view() core.View {
	// The chevron is presentational by construction rather than by choice —
	// it is inside a button, whose children a reader does not descend into.
	// It carries no aria-hidden of its own and needs none; aria-expanded says
	// which way the disclosure points, in a channel that does not depend on a
	// reader pronouncing "▸".
	chevron := "▸"
	if d.Expanded {
		chevron = "▾"
	}

	button := make([]core.PropsAndChildren, 0, len(d.Control)+len(d.ControlStyle)+6)
	button = append(button,
		core.OnClick(d.OnToggle),
		// Both halves stated together: the state is dropped on any role that
		// is not one of the six ARIA scopes it to, and the role is what makes
		// the row announce as something a reader can press at all.
		core.AccessibilityRole(core.RoleButton),
		core.AccessibilityExpanded(core.ExpandedWhen(d.Expanded)),
		core.AccessibilityLabel(d.Label),
		core.AccessibilityHint(d.Hint),
	)
	for _, sp := range d.ControlStyle {
		button = append(button, sp)
	}
	button = append(button, core.Text(chevron, d.ChevronStyle...))
	for _, child := range d.Control {
		button = append(button, child)
	}
	row := core.Row(button...)

	if !d.Heading {
		return row
	}

	// core.Box and not core.Column — a Column arrives carrying the theme's
	// screen inset, and this wrapper must be geometrically invisible.
	wrapper := headingProps(d.Level, d.OwnLevel)
	// Named explicitly, which is the whole reason the nesting works; see the
	// type comment.
	wrapper = append(wrapper, core.AccessibilityLabel(d.Label))
	wrapper = append(wrapper, d.HeadingStyle...)
	return core.Box(append(asProps(wrapper), row)...)
}
