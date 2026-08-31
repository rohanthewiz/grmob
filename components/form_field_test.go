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
