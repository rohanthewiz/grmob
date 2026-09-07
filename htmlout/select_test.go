package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// selectNode builds a picker node in the wire shape core.Select produces.
func selectNode(value string, style *core.Style, opts ...map[string]string) *core.Node {
	return &core.Node{
		Type:  "Select",
		Style: style,
		Props: map[string]any{"value": value, "options": opts},
	}
}

func opt(value, label string) map[string]string {
	return map[string]string{"value": value, "label": label}
}

// The options come from the props, and the chosen one is marked on the option
// rather than on the element — <select> has no value attribute, so a value
// written there would be inert and the picker would open on its first entry.
func TestSelectWritesItsOptionsAndMarksTheChosenOne(t *testing.T) {
	out := ExportHTML(selectNode("pt", nil,
		opt("us", "United States"),
		opt("pt", "Portugal"),
	))

	if !strings.Contains(out, "<select") {
		t.Fatalf("not exported as a <select>:\n%s", out)
	}
	if !strings.Contains(out, `<option value="pt" selected="selected">Portugal</option>`) {
		t.Errorf("the chosen option is not marked:\n%s", out)
	}
	if strings.Contains(out, `<option value="us" selected="selected"`) {
		t.Errorf("an unchosen option is marked:\n%s", out)
	}
	if strings.Contains(out, `value="pt"`) && strings.Contains(out, `<select value=`) {
		t.Errorf("a value attribute was written on the element, where it is inert:\n%s", out)
	}
}

// A value matching nothing marks nothing, which is what a live <select> does
// with an out-of-list value: it shows the first option. Degrading the same way
// on both web targets matters more than inventing a better answer on one.
func TestSelectWithAnUnmatchedValueMarksNothing(t *testing.T) {
	out := ExportHTML(selectNode("zz", nil, opt("us", "United States"), opt("pt", "Portugal")))
	if strings.Contains(out, "selected") {
		t.Errorf("an unmatched value marked an option:\n%s", out)
	}
}

// Option labels and values are as user-originated as any other content here —
// a country list read from a server is the normal case — so both halves go out
// escaped. The value travels the attribute path (a double quote is the
// breakout character) and the label the text path.
func TestSelectOptionsAreEscaped(t *testing.T) {
	out := ExportHTML(selectNode("", nil,
		opt(`" onfocus="steal()`, `<script>alert(1)</script>`),
	))
	if strings.Contains(out, `" onfocus="`) {
		t.Errorf("an option value broke out of its attribute:\n%s", out)
	}
	if strings.Contains(out, "<script>") {
		t.Errorf("an option label rendered as live markup:\n%s", out)
	}
}

// The picker is a form control, which is what makes HTML's own disabled
// attribute legal on it — the pointer-events fallback a disabled container
// gets would leave the control focusable and silently unannounced.
func TestADisabledSelectIsDisabledTheHTMLWay(t *testing.T) {
	out := ExportHTML(selectNode("", &core.Style{Disabled: true}, opt("a", "A")))
	if !strings.Contains(out, `disabled="disabled"`) {
		t.Errorf("no disabled attribute:\n%s", out)
	}
	if strings.Contains(out, "pointer-events:none") {
		t.Errorf("the container fallback was used on a form control:\n%s", out)
	}
}

// The border decision this node type was added to settle: the Go style owns
// the frame, so a picker with no border in its style is talked out of the
// browser's, exactly as a text field is.
//
// Both directions, because a reset that swallowed a real border would be the
// opposite bug and just as silent.
func TestSelectFrameIsTheStylesAndNotTheBrowsers(t *testing.T) {
	bare := ExportHTML(selectNode("", nil, opt("a", "A")))
	if !strings.Contains(bare, "border:none") {
		t.Errorf("a picker with no border in its style keeps the browser's:\n%s", bare)
	}

	framed := ExportHTML(selectNode("", &core.Style{BorderWidth: 1, BorderColor: "#8E8E93"}, opt("a", "A")))
	if !strings.Contains(framed, "border:1px solid #8E8E93") {
		t.Errorf("the style's own frame was swallowed:\n%s", framed)
	}
	if strings.Contains(framed, "border:none") {
		t.Errorf("the reset fired on a picker that asked for a frame:\n%s", framed)
	}
}

// A themed picker draws the field frame the theme states, which is the whole
// premise of the reset above: there is something to reset *to*. This is the
// end-to-end version — a real core.Select rendered against a real theme —
// where the tests above build nodes by hand.
func TestAThemedPickerWearsTheThemesFieldFrame(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	out := ExportHTML(core.Select("pt", []core.SelectOption{
		{Value: "pt", Label: "Portugal"},
	}, func(string) {}).Render(ctx))

	base := core.DefaultTheme.Components.Input
	if !strings.Contains(out, "border:1px solid "+base.BorderColor) {
		t.Errorf("a themed picker does not wear the theme's field frame %q:\n%s", base.BorderColor, out)
	}
	if strings.Contains(out, "border:none") {
		t.Errorf("the reset fired on a themed picker:\n%s", out)
	}
	// And the change callback is recorded, like every other control's.
	if !strings.Contains(out, "data-onchange=") {
		t.Errorf("no change callback ID was exported:\n%s", out)
	}
}

// Consecutive options sharing a group become one <optgroup>, and an option
// with none stays at the top level of the list.
//
// Both halves are asserted because either alone would pass on a picker that
// got the nesting wrong: a renderer that wrapped *every* option in a group of
// its own produces the same set of labels, and one that never closed a run
// produces the same set of options.
func TestSelectGroupsConsecutiveOptionsIntoOptgroups(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	out := ExportHTML(core.Select("pt", []core.SelectOption{
		{Value: "none", Label: "Pick one"},
		{Value: "pt", Label: "Portugal", Group: "Europe"},
		{Value: "es", Label: "Spain", Group: "Europe"},
		{Value: "us", Label: "United States", Group: "Americas"},
		{Value: "zz", Label: "Elsewhere"},
	}, func(string) {}).Render(ctx))

	if n := strings.Count(out, "<optgroup"); n != 2 {
		t.Errorf("%d optgroups, want 2:\n%s", n, out)
	}
	if n := strings.Count(out, "</optgroup>"); n != 2 {
		t.Errorf("%d optgroups closed, want 2 — an unclosed run swallows every option "+
			"after it:\n%s", n, out)
	}
	for _, want := range []string{`<optgroup label="Europe">`, `<optgroup label="Americas">`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s:\n%s", want, out)
		}
	}
	// The ungrouped pair must sit outside any group. Checked by position:
	// "Pick one" before the first <optgroup, "Elsewhere" after the last
	// </optgroup>.
	if strings.Index(out, "Pick one") > strings.Index(out, "<optgroup") {
		t.Errorf("the leading ungrouped option was pulled into a group:\n%s", out)
	}
	if strings.LastIndex(out, "Elsewhere") < strings.LastIndex(out, "</optgroup>") {
		t.Errorf("the trailing ungrouped option was left inside a group:\n%s", out)
	}
}

// A group that ends the list is closed.
//
// Separated from the test above rather than folded into it, because that one
// cannot see this: its last group is followed by an ungrouped option, so the
// run is closed by the *next* option's heading changing and the explicit close
// after the loop is never reached. Deleting that close passed every other
// assertion in this file — the markup it produced was a `<select>` closed
// inside an open `<optgroup>`, i.e. a document a browser repairs and a parser
// does not.
//
// The nesting is what is checked, not the tag count: a stray `</optgroup>` in
// the wrong place would balance the count and still be wrong.
func TestSelectClosesAGroupThatEndsTheList(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	out := ExportHTML(core.Select("", []core.SelectOption{
		{Value: "a", Label: "A", Group: "G"},
		{Value: "b", Label: "B", Group: "G"},
	}, nil).Render(ctx))

	closeGroup := strings.Index(out, "</optgroup>")
	closeSelect := strings.Index(out, "</select>")
	if closeGroup < 0 {
		t.Fatalf("the trailing group is never closed:\n%s", out)
	}
	if closeSelect < 0 {
		t.Fatalf("no </select> at all:\n%s", out)
	}
	if closeGroup > closeSelect {
		t.Errorf("the picker is closed inside an open group:\n%s", out)
	}
	// Both options are inside it, which is the other half: a group closed too
	// early is as wrong as one never closed.
	inner := out[strings.Index(out, "<optgroup"):closeGroup]
	for _, want := range []string{`value="a"`, `value="b"`} {
		if !strings.Contains(inner, want) {
			t.Errorf("option %s is outside the group that should hold it:\n%s", want, out)
		}
	}
}

// The same heading either side of a different one is two groups, in the order
// written. See core.SelectOption.Group: a gather would silently reorder the
// list, which is a bigger change than the one being asked for and one no
// renderer could undo.
func TestSelectDoesNotGatherASplitGroup(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	out := ExportHTML(core.Select("", []core.SelectOption{
		{Value: "a", Label: "A", Group: "One"},
		{Value: "b", Label: "B", Group: "Two"},
		{Value: "c", Label: "C", Group: "One"},
	}, nil).Render(ctx))

	if n := strings.Count(out, "<optgroup"); n != 3 {
		t.Errorf("%d optgroups, want 3 — the runs were gathered:\n%s", n, out)
	}
	// And the order is the list's, not the headings'.
	if strings.Index(out, `label="Two"`) < strings.Index(out, `label="One"`) {
		t.Errorf("the groups were reordered:\n%s", out)
	}
}

// A disabled option is written, announced and unselectable. `disabled` is the
// spec-blessed spelling of the bare boolean attribute, as `selected` is on the
// chosen option and `checked` is on a Checkbox — element emits key="value"
// pairs only.
func TestSelectMarksADisabledOptionWithoutDroppingIt(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	out := ExportHTML(core.Select("s", []core.SelectOption{
		{Value: "s", Label: "Small"},
		{Value: "l", Label: "Large", Disabled: true},
	}, nil).Render(ctx))

	if !strings.Contains(out, `<option value="l" disabled="disabled">Large</option>`) {
		t.Errorf("the disabled option is not written as one:\n%s", out)
	}
	if strings.Contains(out, `<option value="s" disabled`) {
		t.Errorf("an option nobody disabled was disabled anyway:\n%s", out)
	}
	// The picker itself is untouched: disabling one choice is not disabling
	// the control, which is core.Style.Disabled's job.
	if strings.Contains(out, "<select disabled") || strings.Contains(out, `<select style="`) &&
		strings.Contains(out, `aria-disabled`) {
		t.Errorf("the control itself was disabled:\n%s", out)
	}
}

// The <optgroup> label goes through the attribute path, so a heading carrying
// markup cannot re-enter the document as markup. The options' own escaping is
// pinned by TestSelectOptionsAreEscaped above; this is the third string the
// picker writes and the one that arrived last.
func TestSelectGroupLabelsAreEscaped(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	out := ExportHTML(core.Select("", []core.SelectOption{
		{Value: "a", Label: "A", Group: `Europe" onmouseover="x`},
	}, nil).Render(ctx))

	if strings.Contains(out, `onmouseover="x"`) {
		t.Errorf("a group label escaped its attribute:\n%s", out)
	}
}

// core.SelectOption.GroupDisabled: <optgroup disabled>, plus the per-option
// attribute core resolves for the targets that have no section-level control.
//
// The web is the only one of the four that can refuse a whole run in one
// place, and it still writes both — because the two are not alternatives here.
// The attribute on the group is what greys the *heading*; the attribute on
// each option is what core wrote for SwiftUI and Compose and what this
// exporter has no reason to strip back out.
func TestSelectWritesADisabledRunAsADisabledOptgroup(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	out := ExportHTML(core.Select("free", []core.SelectOption{
		{Value: "free", Label: "Free", Group: "Free"},
		{Value: "pro", Label: "Pro", Group: "Paid"},
		// Declared on the run's last option, which is where a renderer that
		// reads the flag as it opens a group would miss it.
		{Value: "max", Label: "Max", Group: "Paid", GroupDisabled: true},
	}, nil).Render(ctx))

	if !strings.Contains(out, `<optgroup label="Paid" disabled="disabled">`) {
		t.Errorf("the disabled run's optgroup carries no disabled attribute:\n%s", out)
	}
	if strings.Contains(out, `<optgroup label="Free" disabled`) {
		t.Errorf("a run nobody disabled was disabled anyway:\n%s", out)
	}
	// Both of the run's options, including the one that made no declaration
	// of its own.
	for _, want := range []string{
		`<option value="pro" disabled="disabled">Pro</option>`,
		`<option value="max" disabled="disabled">Max</option>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, `<option value="free" disabled`) {
		t.Errorf("the declaration reached the run before it:\n%s", out)
	}
}

// An ungrouped run has no <optgroup> to carry the attribute, so the
// declaration degrades to exactly "every option in it is disabled". That is
// the case the per-option propagation exists for, and the one a reader is most
// likely to assume is unsupported.
func TestADisabledUngroupedRunFallsBackToItsOptions(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	out := ExportHTML(core.Select("", []core.SelectOption{
		{Value: "a", Label: "A", GroupDisabled: true},
		{Value: "b", Label: "B"},
	}, nil).Render(ctx))

	if strings.Contains(out, "<optgroup") {
		t.Errorf("a headingless run grew a wrapper:\n%s", out)
	}
	for _, want := range []string{
		`<option value="a" disabled="disabled">A</option>`,
		`<option value="b" disabled="disabled">B</option>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s:\n%s", want, out)
		}
	}
}
