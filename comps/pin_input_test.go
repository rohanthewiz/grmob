package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The harness is select_row_test.go's: one context, one pass per render, the
// protocol a Manager performs. PINInput holds hooks (the field's FocusRef and
// whether it has focus), so its tests cannot call Render bare.

// pinParts is the widget's anatomy: the boxes, in order, and the one field.
type pinParts struct {
	boxes []*core.Node
	field *core.Node
}

func partsOf(t *testing.T, n *core.Node) pinParts {
	t.Helper()
	if n.Type != "ZStack" || len(n.Children) != 2 {
		t.Fatalf("expected a ZStack of the field and the boxes, got %s with %d children", n.Type, len(n.Children))
	}
	// The field is the first layer, under the boxes; see Render.
	field, row := n.Children[0], n.Children[1]
	if row.Type != "Row" || (field.Type != "Input" && field.Type != "InputPassword") {
		t.Fatalf("expected [Input, Row], got [%s, %s]", field.Type, row.Type)
	}
	return pinParts{boxes: row.Children, field: field}
}

// shown is the text box i draws.
func shown(b *core.Node) string {
	return b.Children[0].Props["content"].(string)
}

// pinHarness: a value that the widget's OnChange writes back, so re-rendering
// shows what a real caller would show, plus a record of both callbacks.
type pinHarness struct {
	*rowHarness
	value     string
	changes   []string
	completes []string
}

func newPINHarness(t *testing.T, build func(p *pinHarness) PINInput) *pinHarness {
	t.Helper()
	p := &pinHarness{}
	p.rowHarness = newRowHarness(t, func() core.View {
		in := build(p)
		in.Value = p.value
		in.OnChange = func(v string) { p.value = v; p.changes = append(p.changes, v) }
		if in.OnComplete == nil {
			in.OnComplete = func(v string) { p.completes = append(p.completes, v) }
		}
		return in
	})
	return p
}

// typeField reports the field's whole text, as the host does after an edit.
func typeField(t *testing.T, h *pinHarness, text string) {
	t.Helper()
	id, ok := partsOf(t, h.node).field.Props["onChange"].(string)
	if !ok {
		t.Fatal("the field carries no onChange")
	}
	h.ctx.TriggerTextCallback(id, text)
	h.render()
}

func TestPINInputIsBoxesOverOneField(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4, Label: "SMS code"} })
	root := h.node
	if root.Style.AccessibilityRole != core.RoleGroup || root.Style.AccessibilityLabel != "SMS code" {
		t.Errorf("the widget should be a group named by Label: %+v", root.Style)
	}
	parts := partsOf(t, root)
	if len(parts.boxes) != 4 {
		t.Fatalf("%d boxes, want Length", len(parts.boxes))
	}
	for i, b := range parts.boxes {
		if !b.Style.AccessibilityHidden {
			t.Errorf("box %d should be hidden: the field is the control", i)
		}
		if b.Style.FlexGrow != 1 || b.Style.FlexBasis != "0" {
			t.Errorf("box %d should take an equal share (grow 1, basis 0)", i)
		}
	}
	f := parts.field
	if f.Style.Width != "1px" || f.Style.Height != "1px" || f.Style.TextColor != ColorTransparent {
		t.Errorf("the field should be one invisible point: %+v", f.Style)
	}
	if f.Props["keyboard"] != "digits" {
		t.Errorf("the field should ask for the number pad, keyboard = %v", f.Props["keyboard"])
	}
	if f.Style.AccessibilityLabel != "SMS code, 0 of 4 entered" {
		t.Errorf("field name = %q", f.Style.AccessibilityLabel)
	}
}

func TestPINInputDefaultsToSixBoxesNamedCode(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{} })
	parts := partsOf(t, h.node)
	if len(parts.boxes) != 6 || h.node.Style.AccessibilityLabel != "Code" {
		t.Errorf("got %d boxes named %q", len(parts.boxes), h.node.Style.AccessibilityLabel)
	}
}

// The boxes draw the field's text one character each, and the field holds the
// whole code.
func TestPINInputDrawsTheValueOneCharacterPerBox(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4} })
	typeField(t, h, "41")
	parts := partsOf(t, h.node)
	if parts.field.Props["value"] != "41" {
		t.Errorf("the field should hold the code, got %v", parts.field.Props["value"])
	}
	for i, want := range []string{"4", "1", " ", " "} {
		if got := shown(parts.boxes[i]); got != want {
			t.Errorf("box %d shows %q, want %q", i, got, want)
		}
	}
	if parts.field.Style.AccessibilityLabel != "Code, 2 of 4 entered" {
		t.Errorf("field name = %q", parts.field.Style.AccessibilityLabel)
	}
}

// Every edit is the field's: a key, a paste, a backspace from anywhere. A
// paste longer than the field keeps what fits, and OnComplete fires whenever
// a change leaves the code full, a correction to a full code included.
func TestPINInputEditsAreTheFieldsText(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4} })
	typeField(t, h, "4")
	typeField(t, h, "41729")
	if h.value != "4172" {
		t.Fatalf("a long paste should keep the first four, got %q", h.value)
	}
	typeField(t, h, "417")
	typeField(t, h, "4173")
	want := []string{"4", "4172", "417", "4173"}
	if strings.Join(h.changes, ",") != strings.Join(want, ",") {
		t.Errorf("changes = %v, want %v", h.changes, want)
	}
	if strings.Join(h.completes, ",") != "4172,4173" {
		t.Errorf("completes = %v, want the two full codes", h.completes)
	}
}

// A report of the text already held is an echo: nothing happens.
func TestPINInputIgnoresAnEcho(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4} })
	typeField(t, h, "4172")
	typeField(t, h, "4172")
	typeField(t, h, "41729") // capped to what is held
	if len(h.changes) != 1 || len(h.completes) != 1 {
		t.Errorf("an echo was reported: changes %v, completes %v", h.changes, h.completes)
	}
}

// A tap on the boxes is a tap on the field: it focuses it.
func TestPINInputTapFocusesTheField(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4} })
	row := h.node.Children[1]
	id, _ := row.Props["onClick"].(string)
	if id == "" {
		t.Fatal("the boxes take no tap")
	}
	h.ctx.TriggerCallback(id)
	h.render()
	if got := partsOf(t, h.node).field.Props["focusAction"]; got != "focus" {
		t.Errorf("a tap should focus the field, focusAction = %v", got)
	}
}

// While the field has focus the next box to fill is marked; blurred, none is.
func TestPINInputMarksTheNextBoxWhileFocused(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4} })
	marked := func() int {
		for i, b := range partsOf(t, h.node).boxes {
			if b.Style.BorderWidth == 2 {
				return i
			}
		}
		return -1
	}
	if marked() != -1 {
		t.Fatal("a box is marked before the field has focus")
	}
	f := partsOf(t, h.node).field
	h.ctx.TriggerCallback(f.Props["onFocus"].(string))
	h.render()
	typeField(t, h, "41")
	if marked() != 2 {
		t.Errorf("box %d is marked, want 2, the next to fill", marked())
	}
	typeField(t, h, "4172")
	if marked() != 3 {
		t.Errorf("a full code should mark the last box, got %d", marked())
	}
	h.ctx.TriggerCallback(partsOf(t, h.node).field.Props["onBlur"].(string))
	h.render()
	if marked() != -1 {
		t.Error("a blurred field should mark no box")
	}
}

func TestPINInputSecureMasksTheBoxes(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4, Secure: true} })
	typeField(t, h, "41")
	parts := partsOf(t, h.node)
	if parts.field.Type != "InputPassword" {
		t.Errorf("a secure field should be an InputPassword, got %s", parts.field.Type)
	}
	if shown(parts.boxes[0]) != "•" || shown(parts.boxes[2]) != " " {
		t.Errorf("boxes = %q %q", shown(parts.boxes[0]), shown(parts.boxes[2]))
	}
}

func TestPINInputReportsAFieldNothingCanBeTypedInto(t *testing.T) {
	h := newQuietRowHarness(t, func() core.View {
		return PINInput{Length: 4, Value: "12"}
	})
	if !strings.Contains(core.DumpConcerns(), ConcernPINInputInert) {
		t.Errorf("a PINInput with no OnChange should raise %s, got:\n%s",
			ConcernPINInputInert, core.DumpConcerns())
	}
	// Still draws what it was given: the concern is the report, not a refusal.
	if shown(partsOf(t, h.node).boxes[0]) != "1" {
		t.Error("an inert field should still show its Value")
	}
}

func TestPINInputReportsAValueLongerThanTheField(t *testing.T) {
	h := newQuietRowHarness(t, func() core.View {
		return PINInput{Length: 4, Value: "417293", OnChange: func(string) {}}
	})
	if !strings.Contains(core.DumpConcerns(), ConcernPINValueTooLong) {
		t.Errorf("a Value past the last box should raise %s, got:\n%s",
			ConcernPINValueTooLong, core.DumpConcerns())
	}
	parts := partsOf(t, h.node)
	if len(parts.boxes) != 4 || shown(parts.boxes[3]) != "2" || parts.field.Props["value"] != "4172" {
		t.Errorf("the field should draw and hold the first four: box 3 %q, field %v",
			shown(parts.boxes[3]), parts.field.Props["value"])
	}
}
