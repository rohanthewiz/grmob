package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The harness is select_row_test.go's: one context, one pass per render, the
// protocol a Manager performs. PINInput holds hooks — a FocusRef per cell —
// for the same reason SelectRow does, so its tests cannot call Render bare
// either, and a focus command issued by a handler is only visible on the pass
// after it.

// cellsOf returns the field's cells in order. Both node types are collected so
// one helper serves the Secure case too.
func cellsOf(n *core.Node) []*core.Node {
	var out []*core.Node
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n == nil {
			return
		}
		if n.Type == "Input" || n.Type == "InputPassword" {
			out = append(out, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

// typeInto dispatches cell i's change with what the field would now report,
// then re-renders so a focus command issued by the handler lands on nodes.
func typeInto(t *testing.T, h *pinHarness, i int, text string) {
	t.Helper()
	cells := cellsOf(h.node)
	if i >= len(cells) {
		t.Fatalf("cell %d of %d", i, len(cells))
	}
	id, ok := cells[i].Props["onChange"].(string)
	if !ok {
		t.Fatalf("cell %d carries no onChange", i)
	}
	h.ctx.TriggerTextCallback(id, text)
	h.render()
}

// focused reports which cell the last focus command named, or -1 when no
// command has been issued at all. "focus" is the action exactly one cell
// carries; see core/focus.go for why the others are told "" rather than
// false.
func focused(t *testing.T, h *pinHarness) int {
	t.Helper()
	for i, c := range cellsOf(h.node) {
		if c.Props["focusAction"] == "focus" {
			return i
		}
	}
	return -1
}

// pinHarness is the shape nearly every test here wants: a value that the
// widget's OnChange writes back, so re-rendering shows what a real caller
// would show, plus a record of what the two callbacks were handed.
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

func TestPINInputIsAGroupOfEquallyDividedNamedCells(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput {
		return PINInput{Length: 4, Label: "One-time code"}
	})

	row := h.node
	if row.Type != "Row" {
		t.Fatalf("root = %q, want the Row of cells", row.Type)
	}
	if row.Style.AccessibilityRole != core.RoleGroup {
		t.Errorf("row role = %q, want group so the name below is legal on the web",
			row.Style.AccessibilityRole)
	}
	if row.Style.AccessibilityLabel != "One-time code" {
		t.Errorf("row label = %q, want Label", row.Style.AccessibilityLabel)
	}

	cells := cellsOf(row)
	if len(cells) != 4 {
		t.Fatalf("%d cells, want Length", len(cells))
	}
	for i, c := range cells {
		// The pair that makes the four targets agree on equal shares rather
		// than on equal shares of the leftovers.
		if c.Style.FlexGrow != 1 || c.Style.FlexBasis != "0" {
			t.Errorf("cell %d divides the row as grow %v basis %q, want 1 and \"0\"",
				i, c.Style.FlexGrow, c.Style.FlexBasis)
		}
		if c.Style.Align != core.AlignCenter {
			t.Errorf("cell %d align = %q, want the glyph centred", i, c.Style.Align)
		}
		if want := "One-time code, " + string(rune('1'+i)) + " of 4"; c.Style.AccessibilityLabel != want {
			t.Errorf("cell %d label = %q, want %q", i, c.Style.AccessibilityLabel, want)
		}
	}
}

// Length's default and Label's, in the widget a caller writes with neither.
func TestPINInputDefaultsToSixCellsNamedCode(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{} })

	cells := cellsOf(h.node)
	if len(cells) != defaultPINLength {
		t.Errorf("%d cells, want the %d a one-time code has", len(cells), defaultPINLength)
	}
	if h.node.Style.AccessibilityLabel != "Code" {
		t.Errorf("row label = %q, want the default name", h.node.Style.AccessibilityLabel)
	}
}

// The value is a prefix: cell i draws character i and the rest are empty.
func TestPINInputDrawsTheValueOneCharacterPerCell(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 6} })
	h.value = "417"
	h.render()

	want := []string{"4", "1", "7", "", "", ""}
	for i, c := range cellsOf(h.node) {
		if c.Props["value"] != want[i] {
			t.Errorf("cell %d = %v, want %q", i, c.Props["value"], want[i])
		}
	}
}

func TestPINInputTypingFillsACellAndMovesTheCursorOn(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4} })

	if focused(t, h) != -1 {
		t.Fatal("no focus command should exist before anything is typed")
	}
	typeInto(t, h, 0, "4")

	if h.value != "4" {
		t.Errorf("value = %q, want the character in the first cell", h.value)
	}
	if len(h.changes) != 1 {
		t.Errorf("OnChange fired %d times, want exactly one per keystroke", len(h.changes))
	}
	if got := focused(t, h); got != 1 {
		t.Errorf("cursor at cell %d, want cell 1: the point of the widget", got)
	}
	if len(h.completes) != 0 {
		t.Error("OnComplete fired on an unfinished code")
	}
}

// A paste and an overflowing keystroke are the same event, and this is the
// paste half: the whole code arrives in the first cell and is spread.
func TestPINInputSpreadsAPastedCodeAcrossTheCells(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 6} })

	typeInto(t, h, 0, "417293")

	if h.value != "417293" {
		t.Errorf("value = %q, want the whole pasted code", h.value)
	}
	for i, c := range cellsOf(h.node) {
		if want := string("417293"[i]); c.Props["value"] != want {
			t.Errorf("cell %d = %v, want %q", i, c.Props["value"], want)
		}
	}
	if len(h.completes) != 1 || h.completes[0] != "417293" {
		t.Errorf("OnComplete got %v, want one call with the full code", h.completes)
	}
	// Nothing to advance to, so no command is issued at all — which is what
	// keeps the cursor in the last cell rather than nowhere.
	if got := focused(t, h); got != -1 {
		t.Errorf("a full paste moved the cursor to cell %d, want no command", got)
	}
}

// The other half: a second character in a cell that already holds one arrives
// as both of them, which is the same spread starting at that cell.
func TestPINInputSecondCharacterInAFullCellLandsInTheNextOne(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4} })
	h.value = "1"
	h.render()

	typeInto(t, h, 0, "12")

	if h.value != "12" {
		t.Errorf("value = %q, want the old character kept and the new one after it", h.value)
	}
	// Two cells were filled, so the cursor lands after the second of them.
	if got := focused(t, h); got != 2 {
		t.Errorf("cursor at cell %d, want cell 2 — after the last cell filled", got)
	}
}

// Characters past the last cell are dropped rather than growing the value.
func TestPINInputDropsWhatWillNotFit(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4} })

	typeInto(t, h, 0, "417293")

	if h.value != "4172" {
		t.Errorf("value = %q, want the code clipped to the field", h.value)
	}
	if len(h.completes) != 1 || h.completes[0] != "4172" {
		t.Errorf("OnComplete got %v, want the clipped code once", h.completes)
	}
}

// The asymmetric edit, stated in the type doc: a cleared cell takes the tail
// with it, because a string cannot hold the gap that keeping it would need.
func TestPINInputClearingACellDropsEverythingAfterIt(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 6} })
	h.value = "4172"
	h.render()

	typeInto(t, h, 1, "")

	if h.value != "4" {
		t.Errorf("value = %q, want everything from the cleared cell on dropped", h.value)
	}
	// Clearing is where a person is going backwards; moving them forwards
	// would fight them.
	if got := focused(t, h); got != -1 {
		t.Errorf("clearing moved the cursor to cell %d, want it left where it is", got)
	}
}

// Typing over one cell of a full code replaces that cell and keeps the rest,
// and the corrected code reports again — which is the whole reason
// OnComplete is not keyed on a crossing.
func TestPINInputOverwritesOneCellAndReportsTheCorrectedCode(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 6} })
	h.value = "417293"
	h.render()

	typeInto(t, h, 2, "9")

	if h.value != "419293" {
		t.Errorf("value = %q, want only the third character changed", h.value)
	}
	if len(h.completes) != 1 || h.completes[0] != "419293" {
		t.Errorf("OnComplete got %v, want the corrected code", h.completes)
	}
}

// A report of the text Go just handed the field is not an edit.
func TestPINInputIgnoresAnEchoOfTheValueItAlreadyHas(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 6} })
	h.value = "417"
	h.render()

	typeInto(t, h, 0, "4")

	if len(h.changes) != 0 {
		t.Errorf("OnChange fired %v on an echo", h.changes)
	}
	if len(h.completes) != 0 {
		t.Errorf("OnComplete fired %v on an echo", h.completes)
	}
	if got := focused(t, h); got != -1 {
		t.Errorf("an echo moved the cursor to cell %d", got)
	}
}

// There are no holes, so a cell past the end of the code has no position of
// its own and an edit aimed at one is an edit at the end.
func TestPINInputTypingPastTheEndLandsAtTheEnd(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 6} })
	h.value = "4"
	h.render()

	typeInto(t, h, 3, "7")

	if h.value != "47" {
		t.Errorf("value = %q, want the character appended rather than stranded", h.value)
	}
	if got := focused(t, h); got != 2 {
		t.Errorf("cursor at cell %d, want cell 2 — after the cell actually filled", got)
	}
}

// UseFocusOrder's half of the wiring: the keyboard's action key advances,
// and the last cell keeps nothing to advance to.
func TestPINInputCellsAdvertiseTheKeyboardsNextAction(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput { return PINInput{Length: 4} })

	cells := cellsOf(h.node)
	for i, c := range cells[:len(cells)-1] {
		if c.Props["imeAction"] != "next" {
			t.Errorf("cell %d advertises %v, want next", i, c.Props["imeAction"])
		}
		if _, ok := c.Props["onSubmit"].(string); !ok {
			t.Errorf("cell %d has no onSubmit for its Next key to dispatch", i)
		}
	}
	if last := cells[len(cells)-1]; last.Props["imeAction"] != "" {
		t.Errorf("the last cell advertises %v, want nothing to advance to", last.Props["imeAction"])
	}
}

func TestPINInputSecureMasksTheCells(t *testing.T) {
	h := newPINHarness(t, func(*pinHarness) PINInput {
		return PINInput{Length: 4, Secure: true}
	})

	for i, c := range cellsOf(h.node) {
		if c.Type != "InputPassword" {
			t.Errorf("cell %d = %q, want the masked field", i, c.Type)
		}
	}
}

// The hook decision, pinned where it can actually fail: a state allocated
// after the widget must keep its slot when Length shrinks. Without the
// high-water mark the sentinel below would land on a FocusRef's slot and the
// typed accessor would panic.
func TestPINInputKeepsItsHookSlotsWhenLengthShrinks(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })

	length := 6
	ctx := core.NewContext()
	var sentinel string
	pass := func() {
		ctx.BeginRenderPass()
		ctx.Reset()
		PINInput{Length: length, Value: "", OnChange: func(string) {}}.Render(ctx)
		// Allocated after the widget, exactly as a component's own state
		// would be if it rendered a PINInput above it.
		slot := core.NewState(ctx, "kept")
		sentinel = slot.Get()
		ctx.EndRenderPass()
	}

	pass()
	length = 4
	pass()

	if sentinel != "kept" {
		t.Errorf("the state after the widget reads %q, want its own value", sentinel)
	}
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("shrinking Length raised concerns:\n%s", dump)
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
	if cellsOf(h.node)[0].Props["value"] != "1" {
		t.Error("an inert field should still show its Value")
	}
}

func TestPINInputReportsAValueLongerThanTheField(t *testing.T) {
	h := newQuietRowHarness(t, func() core.View {
		return PINInput{Length: 4, Value: "417293", OnChange: func(string) {}}
	})

	if !strings.Contains(core.DumpConcerns(), ConcernPINValueTooLong) {
		t.Errorf("a Value past the last cell should raise %s, got:\n%s",
			ConcernPINValueTooLong, core.DumpConcerns())
	}
	cells := cellsOf(h.node)
	if len(cells) != 4 {
		t.Fatalf("%d cells, want Length", len(cells))
	}
	if cells[3].Props["value"] != "2" {
		t.Errorf("last cell = %v, want the fourth character: the rest are undrawable",
			cells[3].Props["value"])
	}
}
