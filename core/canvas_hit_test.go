package core

import (
	"reflect"
	"testing"
)

// tapCanvas renders a canvas and returns a function that taps it at a box
// point, as a host would, plus the rendered node.
func tapCanvas(t *testing.T, v View) (*Node, func(at string)) {
	t.Helper()
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := v.Render(ctx)
	return n, func(at string) {
		t.Helper()
		id, ok := n.Props["onShapeTap"].(string)
		if !ok {
			t.Fatalf("canvas has no onShapeTap: %v", n.Props)
		}
		ctx.TriggerTextCallback(id, at)
	}
}

// The topmost shape with a handler wins; a shape without one does not take
// the tap, a miss falls to the canvas's own OnClick, and a fill, a stroke and
// a clip each decide what counts as "on the shape".
func TestShapeOnClickHitTest(t *testing.T) {
	var got []string
	hit := func(name string) func() { return func() { got = append(got, name) } }

	// 100×100 viewBox in a 200×200 box: every viewBox unit is 2 layout units.
	_, tap := tapCanvas(t, Canvas(100, 100, []Shape{
		// Bottom: a big square.
		{Path: Rect(0, 0, 100, 100), Fill: "#eee", OnClick: hit("back")},
		// A filled circle over its middle.
		{Path: Circle(50, 50, 20), Fill: "#f00", OnClick: hit("dot")},
		// A gridline across the dot, with no handler: transparent to taps.
		{Path: Line(0, 50, 100, 50), Stroke: "#000", StrokeWidth: 10},
		// An unfilled 4-unit-wide stroked line near the top.
		{Path: Line(10, 10, 90, 10), Stroke: "#00f", StrokeWidth: 4, OnClick: hit("rule")},
		// A filled square at the bottom clipped to its left half.
		{Path: Rect(60, 70, 30, 20), Fill: "#0f0", Clip: Rect(60, 70, 15, 20), OnClick: hit("clipped")},
	}, OnClick(hit("canvas"))))

	for _, c := range []struct {
		at, want string
	}{
		{"100,100,200,200", "dot"},     // centre, under the gridline
		{"100,141,200,200", "back"},    // (50, 70.5): just outside the circle
		{"100,21,200,200", "rule"},     // y 10.5 in viewBox, 1 layout unit from the line
		{"100,23.5,200,200", "back"},   // 3.5 layout units away: past half the 4 width
		{"130,160,200,200", "clipped"}, // (65, 80): inside the clip
		{"170,160,200,200", "back"},    // (85, 80): outside the clip
	} {
		got = nil
		tap(c.at)
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("tap %s ran %v, want [%s]", c.at, got, c.want)
		}
	}

	// A canvas whose only handled shape is missed runs its own OnClick.
	_, tap = tapCanvas(t, Canvas(100, 100, []Shape{
		{Path: Circle(50, 50, 10), Fill: "#f00", OnClick: hit("dot")},
	}, OnClick(hit("canvas"))))
	got = nil
	tap("5,5,200,200")
	if !reflect.DeepEqual(got, []string{"canvas"}) {
		t.Errorf("miss ran %v, want the canvas's OnClick", got)
	}
}

// The tap is in the box, so the mapping (fit's letterbox, stretch's unequal
// axes) has to be undone the way the renderers applied it.
func TestShapeOnClickFollowsTheMapping(t *testing.T) {
	var got []string
	shapes := []Shape{
		{Path: Rect(0, 0, 50, 50), Fill: "#f00", OnClick: func() { got = append(got, "left") }},
		{Path: Rect(50, 0, 50, 50), Fill: "#00f", OnClick: func() { got = append(got, "right") }},
	}
	// Fit: a 100×50 drawing in a 400×400 box is 4× and letterboxed by 100
	// above and below, so y = 50 is above the drawing and hits nothing.
	_, tap := tapCanvas(t, Canvas(100, 50, shapes))
	for _, c := range []struct{ at, want string }{
		{"100,200,400,400", "left"},
		{"300,200,400,400", "right"},
		{"100,50,400,400", ""},
	} {
		got = nil
		tap(c.at)
		if want := []string{c.want}; c.want == "" && len(got) != 0 || c.want != "" && !reflect.DeepEqual(got, want) {
			t.Errorf("fit tap %s ran %v, want %q", c.at, got, c.want)
		}
	}
	// Stretch: the same drawing fills a 400×40 box, so (300, 20) is the right
	// half's middle.
	_, tap = tapCanvas(t, Canvas(100, 50, shapes, CanvasStretch))
	got = nil
	tap("300,20,400,40")
	if !reflect.DeepEqual(got, []string{"right"}) {
		t.Errorf("stretch tap ran %v, want [right]", got)
	}
}

// The fill rule decides a ring's hole for the hit-test as it does for the
// paint.
func TestShapeOnClickFillRule(t *testing.T) {
	ring := NewPath().Arc(50, 50, 40, 0, 360).Close().Arc(50, 50, 20, 0, 360).Close()
	for _, c := range []struct {
		rule FillRule
		hole bool
	}{{FillNonZero, false}, {FillEvenOdd, true}} {
		hits := 0
		_, tap := tapCanvas(t, Canvas(100, 100, []Shape{
			{Path: ring, Fill: "#000", FillRule: c.rule, OnClick: func() { hits++ }},
		}))
		tap("50,50,100,100") // the centre, in both circles
		tap("50,15,100,100") // between them
		if want := map[bool]int{true: 1, false: 2}[c.hole]; hits != want {
			t.Errorf("%s ring: %d hits, want %d", c.rule, hits, want)
		}
	}
}

// Only a canvas with a tappable shape carries onShapeTap, and a disabled one
// carries none: hosts attach nothing for it.
func TestOnShapeTapIsWrittenOnlyWhenAShapeCanBeHit(t *testing.T) {
	for _, c := range []struct {
		what string
		v    View
		want bool
	}{
		{"no handlers", Canvas(10, 10, []Shape{{Path: Rect(0, 0, 5, 5), Fill: "#000"}}), false},
		{"a text shape's handler", Canvas(10, 10, []Shape{{Text: &CanvasText{Content: "a"}, Fill: "#000", OnClick: func() {}}}), false},
		{"disabled", Canvas(10, 10, []Shape{{Path: Rect(0, 0, 5, 5), Fill: "#000", OnClick: func() {}}}, Disabled(true)), false},
		{"a handler", Canvas(10, 10, []Shape{{Path: Rect(0, 0, 5, 5), Fill: "#000", OnClick: func() {}}}), true},
	} {
		n := renderCanvas(c.v)
		if _, ok := n.Props["onShapeTap"]; ok != c.want {
			t.Errorf("%s: onShapeTap present = %v, want %v", c.what, ok, c.want)
		}
	}
}

// A malformed tap is dropped, not guessed at.
func TestParseCanvasTap(t *testing.T) {
	if x, y, w, h, ok := parseCanvasTap("1.5, 2,300,40"); !ok || x != 1.5 || y != 2 || w != 300 || h != 40 {
		t.Errorf("good tap = %v %v %v %v %v", x, y, w, h, ok)
	}
	for _, bad := range []string{"", "1,2,3", "1,2,0,4", "a,2,3,4", "1,2,3,NaN", "1,2,3,4,5"} {
		if _, _, _, _, ok := parseCanvasTap(bad); ok {
			t.Errorf("parseCanvasTap(%q) accepted", bad)
		}
	}
}

// A text shape is a CanvasText node with only its set fields on the wire.
func TestCanvasTextWire(t *testing.T) {
	n := renderCanvas(Canvas(100, 50, []Shape{
		{Fill: "#111", Text: &CanvasText{X: 33.333333, Y: 10, Content: "Jan"}},
		{Fill: "#222", Clip: Rect(0, 0, 50, 50), Text: &CanvasText{
			X: 50, Y: 25, Content: "Mid", Size: 16, Bold: true,
			Align: CanvasAlignMiddle, VAlign: CanvasVAlignTop,
		}},
		{Text: &CanvasText{Content: "x", Align: "sideways", VAlign: CanvasVAlignMiddle}},
	}))
	want := []map[string]any{
		{"text": "Jan", "at": []float64{33.33, 10}, "size": 12.0, "fill": "#111"},
		{"text": "Mid", "at": []float64{50, 25}, "size": 16.0, "fill": "#222", "bold": true,
			"align": "middle", "valign": "top", "clip": Rect(0, 0, 50, 50).wire(2)},
		{"text": "x", "at": []float64{0, 0}, "size": 12.0},
	}
	for i, c := range n.Children {
		if c.Type != "CanvasText" {
			t.Errorf("child %d type = %s", i, c.Type)
		}
		if !reflect.DeepEqual(c.Props, want[i]) {
			t.Errorf("child %d props =\n %#v\nwant\n %#v", i, c.Props, want[i])
		}
	}
}

// A clip crosses the wire as path ops; an empty one as an empty list, which
// hides the shape rather than leaving it unclipped.
func TestClipWire(t *testing.T) {
	n := renderCanvas(Canvas(10, 10, []Shape{
		{Path: Rect(0, 0, 10, 10), Fill: "#000"},
		{Path: Rect(0, 0, 10, 10), Fill: "#000", Clip: Rect(0, 0, 5, 5)},
		{Path: Rect(0, 0, 10, 10), Fill: "#000", Clip: NewPath()},
	}))
	if _, ok := n.Children[0].Props["clip"]; ok {
		t.Error("an unclipped shape carries clip")
	}
	if got := n.Children[1].Props["clip"]; !reflect.DeepEqual(got, Rect(0, 0, 5, 5).wire(3)) {
		t.Errorf("clip = %v", got)
	}
	if got, ok := n.Children[2].Props["clip"].([]float64); !ok || got == nil || len(got) != 0 {
		t.Errorf("empty clip = %#v, want an empty non-nil list", n.Children[2].Props["clip"])
	}
}
