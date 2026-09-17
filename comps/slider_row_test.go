package comps

import (
	"fmt"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// sliderOf digs the track out of a rendered SliderRow.
func sliderOf(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	s := findFirst(n, func(n *core.Node) bool { return n.Type == "Slider" })
	if s == nil {
		t.Fatal("the row has no Slider in it")
	}
	return s
}

// The shape the type doc draws: the title line and the track share the row's
// growing middle column, which is what aligns the track with the title rather
// than with the leading icon.
func TestSliderRowPutsTheTrackUnderTheTitleInTheGrowingColumn(t *testing.T) {
	_, n := renderDebug(t, SliderRow{
		Title: "Brightness", Subtitle: "Screen", Leading: core.Text("🔆"),
		Value: 72, Max: 100, Step: 1, OnChange: func(float64) {},
	})

	if n.Type != "Row" {
		t.Fatalf("root = %q, want ListRow's Row", n.Type)
	}
	// Leading, then the middle column.
	middle := n.Children[len(n.Children)-1]
	if middle.Type != "Column" || middle.Style.FlexGrow != 1 {
		t.Fatalf("middle = %q grow=%v, want ListRow's growing column", middle.Type, middle.Style.FlexGrow)
	}
	box := middle.Children[0]
	if box.Type != "Box" || len(box.Children) != 2 {
		t.Fatalf("content = %q with %d children, want the header and the track",
			box.Type, len(box.Children))
	}
	if box.Children[0].Type != "Row" {
		t.Errorf("first content child = %q, want the header row", box.Children[0].Type)
	}
	if box.Children[1].Type != "Slider" {
		t.Errorf("second content child = %q, want the track", box.Children[1].Type)
	}
	if findText(n, "Brightness") == nil || findText(n, "Screen") == nil {
		t.Error("the header carries the title and subtitle")
	}
}

func TestSliderRowTrackFillsTheColumnAndCarriesTheRange(t *testing.T) {
	_, n := renderDebug(t, SliderRow{
		Title: "Brightness", Subtitle: "Screen",
		Value: 72, Min: 10, Max: 100, Step: 5, OnChange: func(float64) {},
	})

	s := sliderOf(t, n)
	if s.Style.Width != "100%" {
		t.Errorf("track width = %q, want 100%%: a Compose child hugs its content otherwise", s.Style.Width)
	}
	for key, want := range map[string]float64{"value": 72, "min": 10, "max": 100, "step": 5} {
		if s.Props[key] != want {
			t.Errorf("track %s = %v, want %v", key, s.Props[key], want)
		}
	}
	// The control is what a reader is looking for, so it is the thing named.
	if s.Style.AccessibilityLabel != "Brightness" || s.Style.AccessibilityHint != "Screen" {
		t.Errorf("track a11y = label %q hint %q, want the row's text",
			s.Style.AccessibilityLabel, s.Style.AccessibilityHint)
	}
	// core.Slider exports as <input type="range">, which carries its own
	// range natively; an ARIA one on top would contradict it.
	if s.Style.AccessibilityRole != "" {
		t.Errorf("track role = %q, want none", s.Style.AccessibilityRole)
	}
}

// The row is not a tap target: there is no second value for a tap to mean.
func TestSliderRowIsNotTappableAndTakesNoRole(t *testing.T) {
	_, n := renderDebug(t, SliderRow{Title: "Brightness", Max: 100, OnChange: func(float64) {}})

	if n.Props["onClick"] != nil {
		t.Error("the row must carry no OnTap: a tap has no value to mean on a slider")
	}
	if n.Style.AccessibilityRole != "" {
		t.Errorf("row role = %q, want none: the slider is the control", n.Style.AccessibilityRole)
	}
}

// OnChange is wired to the end of the drag, and nothing is wired to the
// continuous channel unless OnDrag asks for it.
func TestSliderRowReportsAtTheEndOfTheDragOnly(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })

	value, ends, drags := 20.0, 0, 0
	ctx := core.NewContext()
	render := func(withDrag bool) *core.Node {
		ctx.BeginRenderPass()
		ctx.Reset()
		row := SliderRow{
			Title: "Brightness", Value: value, Max: 100, Step: 1,
			OnChange: func(v float64) { value = v; ends++ },
		}
		if withDrag {
			row.OnDrag = func(float64) { drags++ }
		}
		n := row.Render(ctx)
		ctx.EndRenderPass()
		return n
	}

	s := sliderOf(t, render(false))
	if s.Props["onChange"] != nil {
		t.Error("no OnDrag means no continuous callback in the registry at all")
	}
	id, ok := s.Props["onChangeEnd"].(string)
	if !ok {
		t.Fatal("OnChange must ride on the drag's end")
	}
	ctx.TriggerTextCallback(id, "55")
	if ends != 1 || value != 55 {
		t.Errorf("after one drag end: %d reports, value %v; want 1 and 55", ends, value)
	}

	s = sliderOf(t, render(true))
	dragID, ok := s.Props["onChange"].(string)
	if !ok {
		t.Fatal("OnDrag must wire the continuous callback")
	}
	ctx.TriggerTextCallback(dragID, "56")
	if drags != 1 {
		t.Errorf("OnDrag fired %d times, want one per tick", drags)
	}
	if ends != 1 {
		t.Errorf("a drag tick must not also report an end (%d ends)", ends)
	}
}

func TestSliderRowDisabledGreysTheRowAndDropsItsReports(t *testing.T) {
	calls := 0
	_, n := renderDebug(t, SliderRow{
		Title: "Brightness", Value: 20, Max: 100, Disabled: true,
		OnChange: func(float64) { calls++ },
		OnDrag:   func(float64) { calls++ },
	})

	if !n.Style.Disabled {
		t.Error("a disabled row must carry core.Disabled")
	}
	s := sliderOf(t, n)
	if !s.Style.Disabled {
		t.Error("the track must be disabled with the row")
	}
	if s.Props["onChange"] != nil || s.Props["onChangeEnd"] != nil {
		t.Error("a disabled row registers neither callback")
	}
	if calls != 0 {
		t.Errorf("handlers ran %d times on a disabled row", calls)
	}
}

func TestSliderRowReadoutDefaultsToThePrecisionTheStepImplies(t *testing.T) {
	cases := []struct {
		name             string
		row              SliderRow
		want             string
		wantNoReadoutFor bool
	}{
		{name: "a percentage steps in whole numbers",
			row:  SliderRow{Title: "Brightness", Value: 72, Max: 100, Step: 1},
			want: "72"},
		{name: "a half-step keeps its half",
			row:  SliderRow{Title: "Rating", Value: 3.5, Max: 5, Step: 0.5},
			want: "3.5"},
		{name: "a continuous unit range reads in hundredths",
			row:  SliderRow{Title: "Volume", Value: 0.725, Max: 1},
			want: "0.72"},
		{name: "a continuous wide range reads whole",
			row:  SliderRow{Title: "Year", Value: 1994.4, Min: 1900, Max: 2026},
			want: "1994"},
		{name: "Format wins outright",
			row: SliderRow{Title: "Brightness", Value: 72, Max: 100, Step: 1,
				Format: func(v float64) string { return fmt.Sprintf("%.0f%%", v) }},
			want: "72%"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := tc.row
			row.OnChange = func(float64) {}
			_, n := renderDebug(t, row)
			if findText(n, tc.want) == nil {
				t.Errorf("readout should read %q", tc.want)
			}
		})
	}
}

// Returning "" is how a row hides a number that would mean nothing to read.
func TestSliderRowFormatCanDrawNoReadoutAtAll(t *testing.T) {
	_, n := renderDebug(t, SliderRow{
		Title: "Contrast", Value: 0.4, Max: 1, OnChange: func(float64) {},
		Format: func(float64) string { return "" },
	})

	box := findFirst(n, func(n *core.Node) bool { return n.Type == "Box" })
	head := box.Children[0]
	// Title column only: no trailing readout node.
	if len(head.Children) != 1 {
		t.Errorf("header has %d children, want the title column alone", len(head.Children))
	}
}

// A degenerate range is resolved the same way in the readout and in the
// control, so the number beside the title cannot disagree with the thumb.
func TestSliderRowZeroValueIsAPinnedUnitRange(t *testing.T) {
	_, n := renderDebug(t, SliderRow{Title: "Amount", OnChange: func(float64) {}})

	s := sliderOf(t, n)
	if s.Props["min"] != 0.0 || s.Props["max"] != 1.0 || s.Props["value"] != 0.0 {
		t.Errorf("degenerate range = %v..%v at %v, want core.Slider's 0..1 pinned to the start",
			s.Props["min"], s.Props["max"], s.Props["value"])
	}
	if findText(n, "0.00") == nil {
		t.Error("the readout reads the same range the track does")
	}
}

func TestSliderRowClampsOutOfRangeValues(t *testing.T) {
	_, n := renderDebug(t, SliderRow{Title: "Amount", Value: 140, Max: 100, Step: 1, OnChange: func(float64) {}})
	if s := sliderOf(t, n); s.Props["value"] != 100.0 {
		t.Errorf("value = %v, want the maximum", s.Props["value"])
	}
	if findText(n, "100") == nil {
		t.Error("the readout reports the clamped value, not the one it was handed")
	}
}

func TestSliderRowDecimalsTable(t *testing.T) {
	cases := []struct {
		step, span float64
		want       int
	}{
		{step: 1, span: 100, want: 0},
		{step: 5, span: 100, want: 0},
		{step: 0.5, span: 5, want: 1},
		{step: 0.25, span: 1, want: 2},
		{step: 0.001, span: 1, want: 3},
		{step: 0, span: 100, want: 0},
		{step: 0, span: 10, want: 0},
		{step: 0, span: 5, want: 1},
		{step: 0, span: 2, want: 1},
		{step: 0, span: 1, want: 2},
		{step: 0, span: 0.5, want: 2},
	}
	for _, c := range cases {
		if got := sliderRowDecimals(c.step, c.span); got != c.want {
			t.Errorf("sliderRowDecimals(%v, %v) = %d, want %d", c.step, c.span, got, c.want)
		}
	}
}
