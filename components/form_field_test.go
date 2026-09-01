package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestFormFieldLabelInputHint(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := FormField{
		Label: "Email",
		Hint:  "We never share it",
		Input: core.Input("", "you@example.com", func(string) {}),
	}.Render(ctx)

	if n.Type != "Column" {
		t.Fatalf("field frame should be a Column, got %q", n.Type)
	}
	label := findText(n, "Email")
	if label == nil {
		t.Fatal("label should render")
	}
	if label.Style.FontWeight != core.Bold || label.Style.TextColor != core.DefaultTheme.Colors.TextPrimary {
		t.Errorf("label should be bold primary ink, got %+v", label.Style)
	}
	if findFirst(n, func(n *core.Node) bool { return n.Type == "Input" }) == nil {
		t.Error("Input slot should render")
	}
	if findText(n, "We never share it") == nil {
		t.Error("hint should render")
	}
	// The frame must hug its content: the theme Column's screen padding is
	// deliberately zeroed so label/hint read as attached to the input.
	if n.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("field frame should have no padding, got %+v", n.Style.Padding)
	}
}

func TestFormFieldErrorReplacesHint(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := FormField{
		Label: "Email",
		Hint:  "We never share it",
		Error: "Not a valid address",
		Input: core.Input("nope", "", func(string) {}),
	}.Render(ctx)

	if findText(n, "We never share it") != nil {
		t.Error("hint must be replaced while an error is present")
	}
	errText := findText(n, "Not a valid address")
	if errText == nil {
		t.Fatal("error should render")
	}
	if errText.Style.TextColor != core.DefaultTheme.Colors.Error {
		t.Errorf("error ink = %q, want theme Error %q", errText.Style.TextColor, core.DefaultTheme.Colors.Error)
	}
}

func TestFormFieldRequiredMarker(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := FormField{
		Label:    "Email",
		Required: true,
		Input:    core.Input("", "you@example.com", func(string) {}),
	}.Render(ctx)

	marker := findText(n, "*")
	if marker == nil {
		t.Fatal("a required field should draw the marker")
	}
	if marker.Style.TextColor != core.DefaultTheme.Colors.Error {
		t.Errorf("marker ink = %q, want theme Error %q", marker.Style.TextColor, core.DefaultTheme.Colors.Error)
	}
	// The marker is a node of its own precisely so it can say what it means:
	// a screen reader announcing "star" tells the user nothing.
	if marker.Style.AccessibilityLabel != "required" {
		t.Errorf("marker accessibility label = %q, want %q", marker.Style.AccessibilityLabel, "required")
	}

	// The label keeps its own ink — the marker is the only thing in the error
	// color, since a required field is not a field in error.
	label := findText(n, "Email")
	if label == nil {
		t.Fatal("label should still render beside the marker")
	}
	if label.Style.TextColor != core.DefaultTheme.Colors.TextPrimary {
		t.Errorf("label ink = %q, want primary %q", label.Style.TextColor, core.DefaultTheme.Colors.TextPrimary)
	}

	// The pairing Row is the widget's own, so it must not inherit the theme
	// Row's screen-level padding and push the label off the input.
	row := findFirst(n, func(n *core.Node) bool { return n.Type == "Row" })
	if row == nil {
		t.Fatal("label and marker should share a Row")
	}
	if row.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("label row should have no padding, got %+v", row.Style.Padding)
	}
}

func TestFormFieldWithoutRequiredIsUnchanged(t *testing.T) {
	// The Row exists only on the required branch: an ordinary field renders
	// the same bare label it did before the option existed, with no wrapper
	// for a native renderer to lay out.
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := FormField{Label: "Nickname", Input: core.Input("", "", func(string) {})}.Render(ctx)

	if findText(n, "*") != nil {
		t.Error("an optional field must not draw the marker")
	}
	if findFirst(n, func(n *core.Node) bool { return n.Type == "Row" }) != nil {
		t.Error("an optional field should not gain a wrapper Row")
	}
}

func TestFormFieldRequiredWithoutLabelDrawsNothing(t *testing.T) {
	// There is nothing to mark. The checkbox row in examples/signup is this
	// case: its title belongs to the ListRow, not to the field.
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := FormField{
		Required: true,
		Error:    "Please accept the terms",
		Input:    core.Input("", "", func(string) {}),
	}.Render(ctx)

	if findText(n, "*") != nil {
		t.Error("a marker with no label to sit beside must not render")
	}
}
