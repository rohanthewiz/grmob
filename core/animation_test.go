package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTransitionPropCanonicalForm(t *testing.T) {
	cases := []struct {
		ms     int
		easing Easing
		want   string
	}{
		{250, EaseInOut, "250ms ease-in-out"},
		{300, "", "300ms ease"}, // easing defaults to CSS "ease"
		{1000, EaseLinear, "1000ms linear"},
		{0, EaseIn, ""},  // non-positive duration clears the transition
		{-5, EaseIn, ""}, // (a zero-duration "transition" is just a snap)
	}
	for _, c := range cases {
		s := &Style{Transition: "stale"}
		Transition(c.ms, c.easing).Apply(s)
		if s.Transition != c.want {
			t.Errorf("Transition(%d, %q) = %q, want %q", c.ms, c.easing, s.Transition, c.want)
		}
	}
}

func TestTransitionRidesTheNodeStyle(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := Row(Transition(250, EaseInOut), Text("x")).Render(ctx)
	if n.Style.Transition != "250ms ease-in-out" {
		t.Fatalf("node style transition = %q", n.Style.Transition)
	}
}

// Spin is written through as given: the sign is the direction, and Spin(0) is
// the one way to force a node still over a role style that spins it.
func TestSpinPropWritesThePeriodAsGiven(t *testing.T) {
	for _, ms := range []int{1000, -750, 0} {
		s := &Style{Spin: 42}
		Spin(ms).Apply(s)
		if s.Spin != ms {
			t.Errorf("Spin(%d) wrote %d", ms, s.Spin)
		}
	}
}

// Merge follows "non-zero wins", like Rotate: a style with a spin passes it on,
// and a still style does not stop one already there.
func TestSpinMergesLikeRotate(t *testing.T) {
	target := Style{}
	Style{Spin: 1200}.applyTo(&target)
	if target.Spin != 1200 {
		t.Fatalf("merged spin = %d, want 1200", target.Spin)
	}
	Style{Rotate: 5}.applyTo(&target)
	if target.Spin != 1200 {
		t.Errorf("a still style cleared the spin: %d", target.Spin)
	}
}

// The spin crosses the bridge under the key every host parses.
func TestSpinSerializesAsSpin(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := Box(Spin(900)).Render(ctx)
	raw, err := json.Marshal(n.Style)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"Spin":900`) {
		t.Errorf("style JSON = %s, want a Spin key", raw)
	}
	still, _ := json.Marshal(Style{})
	if strings.Contains(string(still), "Spin") {
		t.Errorf("a still style serialized a Spin key: %s", still)
	}
}
