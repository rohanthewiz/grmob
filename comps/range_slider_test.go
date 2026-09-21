package comps

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// slidersOf collects the Slider leaves, in tree order: minimum, then maximum.
func slidersOf(n *core.Node) []*core.Node {
	var out []*core.Node
	walk(n, func(c *core.Node) {
		if c.Type == "Slider" {
			out = append(out, c)
		}
	})
	return out
}

func dollars(v float64) string { return fmt.Sprintf("$%.0f", v) }

func TestRangeSliderStructureAndAccessibility(t *testing.T) {
	_, n := renderDebug(t, RangeSlider{
		Title: "Price", Max: 200, Step: 5, Low: 20, High: 80,
		OnChange: func(float64, float64) {}, Format: dollars,
	})
	if n.Style.AccessibilityRole != core.RoleGroup || n.Style.AccessibilityLabel != "Price" {
		t.Errorf("role %q label %q", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	words := findText(n, "$20 – $80")
	if words == nil {
		t.Fatal("the title line should state the range in words")
	}
	if !words.Style.AccessibilityHidden {
		t.Error("the range in words repeats the sliders' values and should be hidden from a screen reader")
	}
	s := slidersOf(n)
	if len(s) != 2 {
		t.Fatalf("sliders = %d, want 2", len(s))
	}
	if s[0].Style.AccessibilityLabel != "Minimum" || s[1].Style.AccessibilityLabel != "Maximum" {
		t.Errorf("slider names %q and %q", s[0].Style.AccessibilityLabel, s[1].Style.AccessibilityLabel)
	}
	if s[0].Props["value"] != 20.0 || s[1].Props["value"] != 80.0 {
		t.Errorf("slider values %v and %v, want 20 and 80", s[0].Props["value"], s[1].Props["value"])
	}
	// Both tracks share the one range and step.
	for _, sl := range s {
		if sl.Props["min"] != 0.0 || sl.Props["max"] != 200.0 || sl.Props["step"] != 5.0 {
			t.Errorf("track %v..%v step %v", sl.Props["min"], sl.Props["max"], sl.Props["step"])
		}
	}
}

func TestRangeSliderLabelsOverride(t *testing.T) {
	_, n := renderDebug(t, RangeSlider{
		Title: "Hours", Max: 24, Low: 9, High: 17, Labels: [2]string{"Opens", ""},
		OnChange: func(float64, float64) {},
	})
	s := slidersOf(n)
	if s[0].Style.AccessibilityLabel != "Opens" || s[1].Style.AccessibilityLabel != "Maximum" {
		t.Errorf("names %q and %q, want Opens and the default", s[0].Style.AccessibilityLabel, s[1].Style.AccessibilityLabel)
	}
}

// Each drag end reports the whole pair once, and a thumb dragged past the
// other carries it along, so the pair is always ordered.
func TestRangeSliderThumbsPushEachOther(t *testing.T) {
	cases := []struct {
		name         string
		slider       int // 0 the minimum, 1 the maximum
		to           string
		wantL, wantH float64
	}{
		{"low within the range", 0, "40", 40, 80},
		{"high within the range", 1, "60", 20, 60},
		{"low past high carries high up", 0, "90", 90, 90},
		{"high past low carries low down", 1, "10", 10, 10},
		{"low onto high exactly", 0, "80", 80, 80},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got [][2]float64
			ctx, n := renderDebug(t, RangeSlider{
				Title: "Price", Max: 100, Low: 20, High: 80,
				OnChange: func(l, h float64) { got = append(got, [2]float64{l, h}) },
			})
			ctx.TriggerTextCallback(slidersOf(n)[c.slider].Props["onChangeEnd"].(string), c.to)
			if len(got) != 1 || got[0] != [2]float64{c.wantL, c.wantH} {
				t.Errorf("reports = %v, want exactly [%v %v]", got, c.wantL, c.wantH)
			}
		})
	}
}

// An inverted pair is drawn the right way round and reported, and a pair
// outside the range is clamped as SliderRow clamps one value.
func TestRangeSliderNormalisesWhatItDraws(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := RangeSlider{Title: "Price", Max: 100, Low: 70, High: 30, OnChange: func(float64, float64) {}}.Render(ctx)
	ctx.EndRenderPass()
	if dump := core.DumpConcerns(); !strings.Contains(dump, ConcernRangeSliderInverted) {
		t.Errorf("want %s, got:\n%s", ConcernRangeSliderInverted, dump)
	}
	s := slidersOf(n)
	if s[0].Props["value"] != 30.0 || s[1].Props["value"] != 70.0 {
		t.Errorf("drawn %v and %v, want 30 and 70", s[0].Props["value"], s[1].Props["value"])
	}

	_, n = renderDebug(t, RangeSlider{Title: "Price", Max: 100, Low: -5, High: 140, OnChange: func(float64, float64) {}})
	if findText(n, "0 – 100") == nil {
		t.Error("a pair outside the range should be drawn clamped to it")
	}
}

func TestRangeSliderDisabledAndInert(t *testing.T) {
	called := false
	_, n := renderDebug(t, RangeSlider{
		Title: "Price", Max: 100, Low: 20, High: 80, Disabled: true,
		OnChange: func(float64, float64) { called = true },
	})
	for _, s := range slidersOf(n) {
		if !s.Style.Disabled {
			t.Error("both sliders should be disabled")
		}
		if s.Props["onChangeEnd"] != nil {
			t.Error("a disabled slider registers no report")
		}
	}
	if called {
		t.Error("a disabled range reported")
	}

	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	RangeSlider{Title: "Price", Max: 100, High: 50}.Render(ctx)
	ctx.EndRenderPass()
	if dump := core.DumpConcerns(); !strings.Contains(dump, ConcernRangeSliderInert) {
		t.Errorf("want %s, got:\n%s", ConcernRangeSliderInert, dump)
	}
}
