package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// findInput finds the field itself. Both core.Input and core.InputWithSubmit
// emit the same "Input" node type — the submit rides the existing callback
// channel as an extra prop rather than as a new node — so one predicate covers
// the widget's two paths.
func findInput(n *core.Node) *core.Node {
	return findFirst(n, func(n *core.Node) bool { return n.Type == "Input" })
}

func TestSearchFieldIsControlled(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	var got string
	n := SearchField{Value: "gra", OnChange: func(s string) { got = s }}.Render(ctx)

	in := findInput(n)
	if in == nil {
		t.Fatal("no input")
	}
	if in.Props["value"] != "gra" {
		t.Errorf("value = %v, want the caller's", in.Props["value"])
	}
	if in.Props["placeholder"] != "Search" {
		t.Errorf("placeholder = %v, want the default %q", in.Props["placeholder"], "Search")
	}
	ctx.TriggerTextCallback(in.Props["onChange"].(string), "grace")
	if got != "grace" {
		t.Errorf("OnChange received %q, want %q", got, "grace")
	}
}

// The row around the field is the frame, so the theme's Input base — its own
// background, corners and inset — has to come off, or there are two boxes.
func TestSearchFieldFlattensTheInput(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := SearchField{}.Render(ctx)

	if n.Style.Background != core.DefaultTheme.Colors.Surface {
		t.Errorf("row background = %q, want the theme Surface", n.Style.Background)
	}
	in := findInput(n)
	if in.Style.Background != ColorTransparent {
		t.Errorf("input background = %q, want transparent", in.Style.Background)
	}
	if in.Style.BorderRadius != 0 {
		t.Errorf("input radius = %v, want 0 — the row carries the corners", in.Style.BorderRadius)
	}
	if in.Style.FlexGrow != 1 {
		t.Errorf("input FlexGrow = %v, want 1", in.Style.FlexGrow)
	}
	if (in.Style.Padding != core.EdgeInsets{}) {
		t.Errorf("input padding = %+v, want zero — the theme inset is from a frame that is gone", in.Style.Padding)
	}
}

func TestSearchFieldClearAppearsOnlyWhenThereIsSomethingToClear(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	if n := (SearchField{OnChange: func(string) {}}).Render(ctx); findButton(n, "✕") != nil {
		t.Error("an empty field should draw no clear button")
	}

	var got string
	cleared := false
	n := SearchField{Value: "grace", OnChange: func(s string) { got = s; cleared = true }}.Render(ctx)

	clear := findButton(n, "✕")
	if clear == nil {
		t.Fatal("no clear button")
	}
	if clear.Style.AccessibilityLabel != "Clear Search" {
		t.Errorf("clear label = %q, want it to name what it clears", clear.Style.AccessibilityLabel)
	}
	// Last in the row, so its arrival never shifts the input among its
	// siblings — row children are unkeyed, so identity there is positional.
	if n.Children[len(n.Children)-1].Props["label"] != "✕" {
		t.Error("the clear button should be the row's last child")
	}
	ctx.TriggerCallback(clear.Props["onClick"].(string))
	if !cleared || got != "" {
		t.Errorf("clearing should send an empty value through OnChange, got %q (fired=%v)", got, cleared)
	}
}

func TestSearchFieldOnClearOverridesTheDefault(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	changed, custom := false, false
	n := SearchField{
		Value:    "grace",
		OnChange: func(string) { changed = true },
		OnClear:  func() { custom = true },
	}.Render(ctx)

	ctx.TriggerCallback(findButton(n, "✕").Props["onClick"].(string))
	if !custom {
		t.Error("OnClear should run")
	}
	if changed {
		t.Error("OnClear should replace the default OnChange(\"\"), not run alongside it")
	}
}

func TestSearchFieldSubmitIsOptional(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	// No OnSubmit: no submit callback registered at all, which keeps a
	// search-as-you-type field off that path entirely.
	if in := findInput(SearchField{}.Render(ctx)); in.Props["onSubmit"] != nil {
		t.Error("a field with no OnSubmit should register no submit callback")
	}

	submitted := false
	n := SearchField{OnSubmit: func() { submitted = true }}.Render(ctx)
	in := findInput(n)
	id, ok := in.Props["onSubmit"].(string)
	if !ok {
		t.Fatal("OnSubmit should register a submit callback")
	}
	ctx.TriggerCallback(id)
	if !submitted {
		t.Error("the return key should invoke OnSubmit")
	}
}

// A placeholder is not a label on any platform — it vanishes on the first
// keystroke — so the field is named even when the caller says nothing.
func TestSearchFieldAccessibilityLabelFallsBackToThePlaceholder(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	if in := findInput(SearchField{}.Render(ctx)); in.Style.AccessibilityLabel != "Search" {
		t.Errorf("default label = %q, want the resolved placeholder", in.Style.AccessibilityLabel)
	}
	n := SearchField{Placeholder: "Find a sermon"}.Render(ctx)
	if in := findInput(n); in.Style.AccessibilityLabel != "Find a sermon" {
		t.Errorf("label = %q, want the placeholder", in.Style.AccessibilityLabel)
	}
	n = SearchField{Placeholder: "Find a sermon", AccessibilityLabel: "Sermon search"}.Render(ctx)
	if in := findInput(n); in.Style.AccessibilityLabel != "Sermon search" {
		t.Errorf("label = %q, want the explicit one", in.Style.AccessibilityLabel)
	}
}

func TestSearchFieldGlyph(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := SearchField{}.Render(ctx)
	g := findText(n, "🔍")
	if g == nil {
		t.Fatal("no default glyph")
	}
	// The field beside it is already named; announcing the magnifier too
	// names the same control twice.
	if !g.Style.AccessibilityHidden {
		t.Error("the glyph should be hidden from assistive tech")
	}

	if n := (SearchField{Glyph: "🔎"}).Render(ctx); findText(n, "🔎") == nil {
		t.Error("Glyph should override the default")
	}
	if n := (SearchField{NoGlyph: true}).Render(ctx); findText(n, "🔍") != nil {
		t.Error("NoGlyph should drop the mark")
	}
}

// core.Input registers whatever it is given and the dispatcher invokes it
// unguarded, so a nil OnChange has to become a no-op rather than reach the
// registry — the same panic Chip's nil OnTap used to have.
func TestSearchFieldNilOnChangeDoesNotPanic(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := SearchField{Value: "x"}.Render(ctx)

	ctx.TriggerTextCallback(findInput(n).Props["onChange"].(string), "typed")
	// And the clear button, whose default path also runs through OnChange.
	ctx.TriggerCallback(findButton(n, "✕").Props["onClick"].(string))
}

// The widget holds no state and calls no hook, which is what lets a search box
// live in a header that appears and disappears.
func TestSearchFieldConsumesNoHookSlot(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	before := ctx.Cursor
	SearchField{Value: "x", OnChange: func(string) {}}.Render(ctx)
	if ctx.Cursor != before {
		t.Errorf("cursor moved from %d to %d — SearchField must consume no hook slot", before, ctx.Cursor)
	}
}

// The search landmark goes on the row rather than on the input: the landmark
// is the region a reader jumps to, and landing on the field alone would put
// them past the clear button. The field keeps its own name either way — a role
// says what a region is, a label says what a control is called, and neither
// substitutes for the other.
func TestSearchFieldIsASearchLandmark(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := SearchField{Value: "grace", OnChange: func(string) {}}.Render(ctx)

	if n.Style.AccessibilityRole != core.RoleSearch {
		t.Errorf("the strip = %q, want search", n.Style.AccessibilityRole)
	}
	input := findFirst(n, func(n *core.Node) bool { return n.Type == "Input" })
	if input == nil {
		t.Fatal("no input in the search field")
	}
	if input.Style.AccessibilityLabel == "" {
		t.Error("the field should still be named by its own label")
	}
	if input.Style.AccessibilityRole != core.RoleNone {
		t.Errorf("the input is a control inside the region, not the region: %q",
			input.Style.AccessibilityRole)
	}
}
