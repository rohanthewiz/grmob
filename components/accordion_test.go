package components

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// renderPass drives one framework-shaped pass over a view, the sequence
// render.Manager performs. Accordion owns state via NewState, so its tests
// must respect the pass protocol rather than calling Render bare.
func renderPass(ctx *core.Context, v core.View) *core.Node {
	ctx.BeginRenderPass()
	ctx.Reset()
	return v.Render(ctx)
}

func TestAccordionTogglesContent(t *testing.T) {
	ctx := core.NewContext()
	acc := Accordion{
		Title:   "Details",
		Content: core.Text("hidden treasure"),
	}

	n := renderPass(ctx, acc)
	if findText(n, "hidden treasure") != nil {
		t.Fatal("content should start collapsed")
	}
	header := findFirst(n, func(n *core.Node) bool { return n.Props["onClick"] != nil })
	if header == nil {
		t.Fatal("header should carry an onClick toggle")
	}
	if findText(n, "▸") == nil {
		t.Error("collapsed accordion should show the closed chevron")
	}

	// Tap the header, then re-render: content appears, chevron flips.
	ctx.TriggerCallback(header.Props["onClick"].(string))
	n = renderPass(ctx, acc)
	if findText(n, "hidden treasure") == nil {
		t.Fatal("content should render after expanding")
	}
	if findText(n, "▾") == nil {
		t.Error("expanded accordion should show the open chevron")
	}

	// Tap again: collapses back.
	header = findFirst(n, func(n *core.Node) bool { return n.Props["onClick"] != nil })
	ctx.TriggerCallback(header.Props["onClick"].(string))
	n = renderPass(ctx, acc)
	if findText(n, "hidden treasure") != nil {
		t.Error("content should collapse on the second tap")
	}
}

func TestAccordionInitiallyExpanded(t *testing.T) {
	ctx := core.NewContext()
	n := renderPass(ctx, Accordion{
		Title:             "Open by default",
		Content:           core.Text("visible"),
		InitiallyExpanded: true,
	})
	if findText(n, "visible") == nil {
		t.Error("InitiallyExpanded should seed the state expanded")
	}
}

func TestAccordionHeaderSlot(t *testing.T) {
	ctx := core.NewContext()
	n := renderPass(ctx, Accordion{
		Title:   "For accessibility",
		Header:  core.Text("custom header"),
		Content: core.Text("content"),
	})
	if findText(n, "custom header") == nil {
		t.Error("Header slot should replace the default header content")
	}
	if findText(n, "For accessibility") != nil {
		t.Error("default title text should be suppressed when Header is set")
	}
}

// The header announces all three of what it is called, what it is, and
// whether it is open.
//
// The shape is ARIA's own accordion pattern — a heading wrapping a button —
// and it is the third arrangement this header has had, so the assertion is
// written as the whole document rather than as a list of attributes: what
// makes it right is which element carries which fact, and every intermediate
// version passed a bag of substrings.
//
//	Box  role=heading  aria-level  aria-label      the outline entry
//	  Row  role=button  aria-expanded  aria-label  the control
//
// Two earlier versions to keep in mind, because both are one edit away.
//
// The row was unroled and named, which put the name on `generic` — prohibited
// by ARIA and pruned by every browser, while VoiceOver and TalkBack read it out
// perfectly. core.RoleGroup, supplied by the exporters, closed that.
//
// Then the row was a group, which is nameable and leaves its children readable
// but is not one of the six roles aria-expanded is defined for. That is what
// this arrangement buys and it is the assertion with no visible effect: a
// group announces a section a reader is never told they can press, and a
// header that cannot say it is shut is a header nobody opens.
func TestTheHeaderAnnouncesItsTierItsRoleAndItsState(t *testing.T) {
	ctx := core.NewContext()
	n := Accordion{Title: "What is a hook", Content: core.Text("state")}.Render(ctx)
	html := htmlout.ExportHTML(n)

	// The two elements, each with its own three facts. Asserted as one
	// substring apiece so that a fact landing on the wrong element fails —
	// which is the only failure mode this shape has, and the one a list of
	// individual attribute checks cannot see.
	for _, want := range []string{
		`role="heading" aria-level="3" aria-label="What is a hook"`,
		`role="button" aria-expanded="false" aria-label="What is a hook" ` +
			`aria-description="Expands or collapses the section"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s:\n%s", want, html)
		}
	}
	// The heading is named rather than being left to take a name from its
	// content. Without the explicit label it would be called "▸ What is a
	// hook", which is what this widget turned the wrapping shape down for
	// until core.AccessibilityLabel was noticed to override it.
	if strings.Contains(html, `aria-label="▸ What is a hook"`) {
		t.Errorf("the heading took its name from its content, chevron included:\n%s", html)
	}
	// And the words inside the button carry no heading of their own. A button's
	// children are presentational, so a role there is written into the document
	// and pruned out of the accessibility tree — correct-looking and inert,
	// which is worse than absent.
	if strings.Count(html, `role="heading"`) != 1 {
		t.Errorf("want exactly one heading in the document:\n%s", html)
	}
}

// The state follows the taps, on every pass, in both directions.
//
// The half that would go silently is ExpandedClosed. A widget that sets the
// open value and leaves the shut one unstated renders identically, toggles
// correctly, and announces a collapsed section as an ordinary button with
// nothing behind it — see core.ExpandedState for why the third value exists.
func TestTheHeaderStatesBothHalvesOfTheDisclosure(t *testing.T) {
	ctx := core.NewContext()
	acc := Accordion{Title: "Details", Content: core.Text("hidden treasure")}

	n := renderPass(ctx, acc)
	if got := headerState(t, n); got != core.ExpandedClosed {
		t.Errorf("collapsed accordion states %q, want %q", got, core.ExpandedClosed)
	}

	header := findFirst(n, func(n *core.Node) bool { return n.Props["onClick"] != nil })
	ctx.TriggerCallback(header.Props["onClick"].(string))
	n = renderPass(ctx, acc)
	if got := headerState(t, n); got != core.ExpandedOpen {
		t.Errorf("expanded accordion states %q, want %q", got, core.ExpandedOpen)
	}
}

// A Header slot gets the control and its state and no heading.
//
// The division is Card.Title/Card.Header's: the caller replaced the content,
// so the widget no longer knows what the line says and will not stamp an
// outline entry named by a Title that is not on screen. What it does still
// know is that the row is a disclosure, which is a fact about the Accordion
// rather than about what was put inside it.
func TestACustomHeaderKeepsTheDisclosureAndDropsTheHeading(t *testing.T) {
	ctx := core.NewContext()
	n := renderPass(ctx, Accordion{
		Title:   "For accessibility",
		Header:  core.Text("custom header"),
		Content: core.Text("content"),
	})

	if got := headerState(t, n); got != core.ExpandedClosed {
		t.Errorf("custom header states %q, want the disclosure state anyway", got)
	}
	if html := htmlout.ExportHTML(n); strings.Contains(html, `role="heading"`) {
		t.Errorf("a replaced header must not be given an outline entry:\n%s", html)
	}
}

// headerState reads the disclosure state off the one node in the tree that
// carries a button role — the header row.
func headerState(t *testing.T, n *core.Node) core.ExpandedState {
	t.Helper()
	node := findFirst(n, func(n *core.Node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == core.RoleButton
	})
	if node == nil {
		t.Fatal("no node carrying the button role: the header is not a control")
	}
	return node.Style.AccessibilityExpanded
}
