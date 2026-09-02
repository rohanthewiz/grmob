package core

import (
	"strconv"
)

// Slider is a horizontal value control: a thumb on a track, dragged to
// choose a number in [min, max]. A seek bar, a volume, a brightness, a
// price ceiling.
//
//	core.Slider(pos, 0, duration, func(v float64) { scrub.Set(v) },
//	    core.OnSliderChangeEnd(func(v float64) { core.AudioSeek(v) }))
//
// Each platform draws its own: Compose's Material 3 Slider, SwiftUI's
// Slider, and <input type="range"> in the browser and in htmlout.
//
// # Two callbacks, and why the second is the important one
//
// onChange fires continuously while the thumb moves — the value under the
// finger, dozens of times a second. That is right for a label that follows
// the drag and wrong for anything expensive or irreversible: a seek on a
// network stream, a request to a server. OnSliderChangeEnd fires once, when
// the finger lifts, with the final value — and it is the one a seek bar
// acts on. onChange may be nil when only the end matters.
//
// Both cross the bridge as text callbacks carrying the number formatted
// with strconv (the natives' own float formatting is accepted too: "0.5",
// "1.0E-4" and "1e-4" all parse). A value that fails to parse is dropped,
// the same policy NumericInput applies.
//
// # The control is controlled
//
// Like every leaf, the value shown is the one Go rendered — but a drag has
// to feel immediate, and the Go round trip is asynchronous, so the native
// renderers show the finger's value *while dragging* and Go's value
// otherwise (the same compromise the text fields make). A seek bar fed by
// a status tick therefore never snaps the thumb back under the finger.
func Slider(value, min, max float64, onChange func(float64), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		if max <= min {
			// A degenerate range renders as an empty slider rather than a
			// division by zero on the far side. min..min+1 is arbitrary but
			// finite, and the value is pinned to its start.
			max = min + 1
			value = min
		}
		if value < min {
			value = min
		}
		if value > max {
			value = max
		}
		p := map[string]any{
			"value": value,
			"min":   min,
			"max":   max,
		}
		if onChange != nil {
			p["onChange"] = ctx.registerTextCallback(sliderCallback(onChange))
		}
		return leafNode(ctx, "Slider", Style{}, p, props)
	})
}

// OnSliderChangeEnd fires once when the drag ends, with the final value —
// see Slider for why a seek bar wants this rather than onChange.
func OnSliderChangeEnd(fn func(float64)) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if fn == nil {
			return
		}
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["onChangeEnd"] = ctx.registerTextCallback(sliderCallback(fn))
	})
}

// SliderStep snaps the thumb to multiples of step from min. 0 (the default)
// is continuous.
func SliderStep(step float64) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if step <= 0 {
			return
		}
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["step"] = step
	})
}

// sliderCallback adapts a float handler to the text callback channel.
func sliderCallback(fn func(float64)) func(string) {
	return func(s string) {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			fn(v)
		}
	}
}
