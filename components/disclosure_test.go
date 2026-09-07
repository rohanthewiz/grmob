package components

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// disclosureShape is the ARIA arrangement both disclosures must produce,
// read off a rendered tree: a heading naming the control, wrapped around a
// button that carries the state and the same name.
type disclosureShape struct {
	headingLevel int
	headingLabel string
	buttonLabel  string
	expanded     core.ExpandedState
	chevron      string
}

// readDisclosure locates the shape inside a rendered subtree.
//
// By role rather than by child index, on the rule helpers_test.go states: a
// widget that gains a decorative wrapper should not break a test about
// semantics. The button is looked for *inside* the heading, which is the half
// of the arrangement that is easy to get backwards and impossible to see in
// an export that happens to look right.
func readDisclosure(t *testing.T, n *core.Node) disclosureShape {
	t.Helper()

	heading := findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleHeading
	})
	if heading == nil {
		t.Fatal("no heading node")
	}
	button := findFirst(heading, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleButton
	})
	if button == nil {
		t.Fatal("the heading does not contain the button — ARIA's disclosure pattern " +
			"nests a button inside a heading, and the other order announces a heading " +
			"named after a control")
	}

	// Nothing inside the control may claim to be a heading. The tier rides
	// the wrapper precisely because a button's children are presentational:
	// a heading role in here is written into the document, pruned out of the
	// accessibility tree by every browser, and still looks correct in an
	// export — which is the one failure mode a structural test has to catch,
	// since no reader will ever report it.
	if h := findFirst(button, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleHeading
	}); h != nil {
		t.Error("a heading sits inside the button, where every browser prunes it")
	}

	var chevron string
	if glyph := findFirst(button, func(c *core.Node) bool {
		if c.Type != "Text" {
			return false
		}
		s, _ := c.Props["content"].(string)
		return s == "▸" || s == "▾"
	}); glyph != nil {
		chevron, _ = glyph.Props["content"].(string)
	}

	return disclosureShape{
		headingLevel: heading.Style.AccessibilityHeadingLevel,
		headingLabel: heading.Style.AccessibilityLabel,
		buttonLabel:  button.Style.AccessibilityLabel,
		expanded:     button.Style.AccessibilityExpanded,
		chevron:      chevron,
	}
}

// Both disclosures build the same shape, in both states.
//
// This is the test the shared type exists for. Accordion's arrangement was
// argued over three attempts and the two rejected ones both looked correct in
// an export — a named `generic` (dropped by every browser) and a `group` with
// the heading on the words (which cannot carry aria-expanded at all). A
// second widget copying that shape by hand would have been checked only
// against its own expectations, and would have drifted the first time either
// one was touched. Here the two are compared against each other.
func TestBothDisclosuresBuildTheSameShape(t *testing.T) {
	for _, open := range []bool{false, true} {
		state := core.ExpandedClosed
		chevron := "▸"
		if open {
			state = core.ExpandedOpen
			chevron = "▾"
		}

		ctx := core.NewContext()
		ctx.BeginRenderPass()
		acc := readDisclosure(t, Accordion{
			Title:             "January 2026",
			HeadingLevel:      2,
			InitiallyExpanded: open,
			Content:           core.Text("body"),
		}.Render(ctx))

		ctx = core.NewContext()
		ctx.BeginRenderPass()
		band := readDisclosure(t, GroupHeader{
			Group:        Group{Key: "2026-01", Label: "January 2026", Count: 3},
			HeadingLevel: 2,
			Expanded:     open,
			OnToggle:     func() {},
		}.Render(ctx))

		want := disclosureShape{
			headingLevel: 2,
			headingLabel: "January 2026",
			buttonLabel:  "January 2026",
			expanded:     state,
			chevron:      chevron,
		}
		if acc != want {
			t.Errorf("open=%v: Accordion built %+v, want %+v", open, acc, want)
		}
		if band != want {
			t.Errorf("open=%v: GroupHeader built %+v, want %+v", open, band, want)
		}
	}
}

// The state is stated on every pass, shut as well as open.
//
// core.ExpandedClosed is not the same as saying nothing: a collapsed control
// that answers nothing is announced as an ordinary button, and "collapsed" is
// the whole of what tells a reader there is something behind it. The zero
// value of core.ExpandedState is the unstated one, so this is a case a
// straightforward implementation gets wrong by omission.
func TestAShutDisclosureStatesItsState(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	shut := readDisclosure(t, Accordion{Title: "x", Content: core.Text("y")}.Render(ctx))
	if shut.expanded != core.ExpandedClosed {
		t.Errorf("a shut Accordion states %q, want %q", shut.expanded, core.ExpandedClosed)
	}
}

// The chevron is inside the button, where a reader does not reach it.
//
// It used to be deliberately left un-hidden, because it was the only thing
// saying which way the disclosure pointed. aria-expanded says that now, in a
// channel that does not depend on a reader pronouncing "▸" — and the glyph
// being inside a control makes it presentational by construction rather than
// by an aria-hidden somebody has to remember.
func TestTheChevronIsInsideTheControl(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Accordion{Title: "x", Content: core.Text("y")}.Render(ctx)

	button := findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleButton
	})
	if button == nil {
		t.Fatal("no button")
	}
	if findText(button, "▸") == nil {
		t.Error("the chevron is not inside the button — outside it, a reader announces " +
			"the glyph as content")
	}
	// And the heading's name never picks it up: the wrapper is labelled
	// explicitly, which is the whole reason a heading may wrap the control.
	heading := findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleHeading
	})
	if heading == nil || strings.Contains(heading.Style.AccessibilityLabel, "▸") {
		t.Errorf("heading label = %q, want the title alone", heading.Style.AccessibilityLabel)
	}
}

// A custom Accordion header gets the button and no heading, which is the
// division Card.Title and Card.Header draw: a view the caller built is the
// caller's to describe, and a heading named after a Title they replaced would
// announce a row of controls as a section title.
func TestACustomAccordionHeaderGetsNoHeading(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Accordion{
		Title:   "Replaced",
		Header:  core.Text("mine"),
		Content: core.Text("body"),
	}.Render(ctx)

	if h := findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleHeading
	}); h != nil {
		t.Errorf("a custom header was wrapped in a heading named %q", h.Style.AccessibilityLabel)
	}
	if b := findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleButton
	}); b == nil {
		t.Error("a custom header lost the button — the control is the widget's either way")
	}
}
