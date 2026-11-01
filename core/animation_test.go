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

// Glide is Spin's twin on the other paint transform, and it is held to the
// same three facts: the period is written as given (the sign is the
// direction), it merges on non-zero, and it serializes under its own key only
// when set.
func TestGlidePropWritesThePeriodAsGiven(t *testing.T) {
	for _, ms := range []int{12000, -800, 0} {
		s := &Style{Glide: 42}
		Glide(ms).Apply(s)
		if s.Glide != ms {
			t.Errorf("Glide(%d) wrote %d", ms, s.Glide)
		}
	}
}

func TestGlideMergesLikeSpin(t *testing.T) {
	var target Style
	Style{Glide: 12000}.applyTo(&target)
	if target.Glide != 12000 {
		t.Fatalf("merged glide = %d, want 12000", target.Glide)
	}
	Style{}.applyTo(&target)
	if target.Glide != 12000 {
		t.Errorf("a still style cleared the glide: %d", target.Glide)
	}
}

func TestGlideSerializesAsGlide(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := Box(Glide(9000)).Render(ctx)
	raw, err := json.Marshal(n.Style)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"Glide":9000`) {
		t.Errorf("style JSON = %s, want a Glide key", raw)
	}
	still, _ := json.Marshal(Box().Render(ctx).Style)
	if strings.Contains(string(still), "Glide") {
		t.Errorf("a still style serialized a Glide key: %s", still)
	}
}

// The keyframes constant is what both web targets move by, so its shape is
// pinned: the name the animation list uses, a leading-relative distance of
// one full width on the individual translate property, and the same name
// declared again at rest under the reduce-motion media query.
func TestGlideKeyframesShape(t *testing.T) {
	for _, want := range []string{
		"@keyframes grmob-glide{",
		"to{translate:calc(var(--grmob-inline,1)*-100%) 0px}",
		"@media (prefers-reduced-motion:reduce){@keyframes grmob-glide{from,to{translate:0px 0px}}}",
	} {
		if !strings.Contains(GlideKeyframes, want) {
			t.Errorf("GlideKeyframes lacks %q:\n%s", want, GlideKeyframes)
		}
	}
	if strings.Count(GlideKeyframes, "@keyframes grmob-glide") != 2 {
		t.Errorf("the moving rule and its reduced-motion twin should be the only two declarations:\n%s", GlideKeyframes)
	}
}
