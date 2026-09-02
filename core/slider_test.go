package core

import "testing"

func TestSliderRendersBoundsAndClampsTheValue(t *testing.T) {
	ctx := NewContext()
	n := Slider(150, 0, 100, nil).Render(ctx)
	if n.Type != "Slider" {
		t.Fatalf("type = %q", n.Type)
	}
	if n.Props["value"] != 100.0 || n.Props["min"] != 0.0 || n.Props["max"] != 100.0 {
		t.Errorf("props = %v", n.Props)
	}
	if _, has := n.Props["onChange"]; has {
		t.Errorf("nil onChange registered a callback: %v", n.Props)
	}

	// A degenerate range becomes a finite one with the value pinned.
	n = Slider(5, 10, 10, nil).Render(ctx)
	if n.Props["min"] != 10.0 || n.Props["max"] != 11.0 || n.Props["value"] != 10.0 {
		t.Errorf("degenerate range props = %v", n.Props)
	}
}

func TestSliderCallbacksParseFloats(t *testing.T) {
	ctx := NewContext()
	var changed, ended []float64
	n := Slider(0.25, 0, 1, func(v float64) { changed = append(changed, v) },
		OnSliderChangeEnd(func(v float64) { ended = append(ended, v) }),
		SliderStep(0.05),
	).Render(ctx)

	if n.Props["step"] != 0.05 {
		t.Errorf("step = %v", n.Props["step"])
	}
	onChange := n.Props["onChange"].(string)
	onEnd := n.Props["onChangeEnd"].(string)
	if onChange == onEnd {
		t.Fatalf("both callbacks share an ID %q", onChange)
	}

	// Go's own formatting, Kotlin's, Swift's, and garbage.
	for _, s := range []string{"0.5", "1.0E-4", "1e-4", "nope"} {
		ctx.TriggerTextCallback(onChange, s)
	}
	ctx.TriggerTextCallback(onEnd, "0.75")

	if len(changed) != 3 || changed[0] != 0.5 || changed[1] != 1e-4 || changed[2] != 1e-4 {
		t.Errorf("onChange saw %v", changed)
	}
	if len(ended) != 1 || ended[0] != 0.75 {
		t.Errorf("onChangeEnd saw %v", ended)
	}
}
