package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestSkeletonZeroValueIsOneBar(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	theme := core.DefaultTheme

	n := Skeleton{}.Render(ctx)

	if len(n.Children) != 1 {
		t.Fatalf("bars = %d, want 1", len(n.Children))
	}
	bar := n.Children[0]
	if bar.Style.Width != "100%" {
		t.Errorf("width = %q, want 100%%", bar.Style.Width)
	}
	// Border, not Surface: Surface is the fill a *panel* uses, so a Surface
	// bar inside a card disappears — the same trap Separator documents.
	if bar.Style.Background != theme.Colors.BorderColor() {
		t.Errorf("bar color = %q, want the theme Border role %q", bar.Style.Background, theme.Colors.BorderColor())
	}
	// About as tall as the text it stands in for, and scaling with the theme.
	if bar.Style.Height != "17px" {
		t.Errorf("height = %q, want the theme body font size", bar.Style.Height)
	}
	if bar.Style.BorderRadius != 4 {
		t.Errorf("radius = %v, want 4", bar.Style.BorderRadius)
	}
}

// A stack reads as a paragraph because the last bar is short. On a single bar
// the last line is the only line, so shortening it would make the simplest
// call surprising.
func TestSkeletonLastLineIsShortOnlyInAStack(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := Skeleton{Lines: 3}.Render(ctx)
	if len(n.Children) != 3 {
		t.Fatalf("bars = %d, want 3", len(n.Children))
	}
	for i := 0; i < 2; i++ {
		if n.Children[i].Style.Width != "100%" {
			t.Errorf("bar %d width = %q, want 100%%", i, n.Children[i].Style.Width)
		}
	}
	if got := n.Children[2].Style.Width; got != "60%" {
		t.Errorf("last bar width = %q, want the ragged 60%%", got)
	}

	if one := (Skeleton{Lines: 1}).Render(ctx); one.Children[0].Style.Width != "100%" {
		t.Errorf("a single bar width = %q, want the full 100%%", one.Children[0].Style.Width)
	}
}

func TestSkeletonOverrides(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := Skeleton{
		Lines: 2, Height: 44, Width: "44px", LastLineWidth: "30px",
		Gap: 2, Radius: 999, Color: "#DDDDDD",
	}.Render(ctx)

	if n.Style.Gap != 2 {
		t.Errorf("gap = %v, want 2", n.Style.Gap)
	}
	bar := n.Children[0]
	if bar.Style.Height != "44px" || bar.Style.Width != "44px" {
		t.Errorf("bar box = %q x %q, want 44px square", bar.Style.Width, bar.Style.Height)
	}
	if bar.Style.BorderRadius != 999 {
		t.Errorf("radius = %v, want 999 — the recipe for a circular avatar placeholder", bar.Style.BorderRadius)
	}
	if bar.Style.Background != "#DDDDDD" {
		t.Errorf("color = %q, want the override", bar.Style.Background)
	}
	if n.Children[1].Style.Width != "30px" {
		t.Errorf("last bar width = %q, want the override", n.Children[1].Style.Width)
	}
}

// A negative or zero Lines is a caller mistake that must still render
// something: a widget standing in for content is the last place to render an
// empty box.
func TestSkeletonLinesIsClampedToOne(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	for _, lines := range []int{0, -3} {
		if n := (Skeleton{Lines: lines}).Render(ctx); len(n.Children) != 1 {
			t.Errorf("Lines %d rendered %d bars, want 1", lines, len(n.Children))
		}
	}
}

func TestSkeletonAccessibility(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := Skeleton{Lines: 4}.Render(ctx)
	if n.Style.AccessibilityLabel != "Loading" {
		t.Errorf("container label = %q, want the default %q", n.Style.AccessibilityLabel, "Loading")
	}
	// A reader walking four unlabeled boxes is worse than silence.
	for i, bar := range n.Children {
		if !bar.Style.AccessibilityHidden {
			t.Errorf("bar %d should be hidden from assistive tech", i)
		}
	}

	n = Skeleton{AccessibilityLabel: "Loading sermons"}.Render(ctx)
	if n.Style.AccessibilityLabel != "Loading sermons" {
		t.Errorf("label = %q, want the caller's", n.Style.AccessibilityLabel)
	}

	// The documented way to say nothing at all: Style lands after the
	// defaults, and AccessibilityHidden is an ordinary style prop.
	n = Skeleton{Style: []core.StyleProp{core.AccessibilityHidden()}}.Render(ctx)
	if !n.Style.AccessibilityHidden {
		t.Error("Style should be able to hide the whole block")
	}
}

// The container paints and insets nothing, so a one-line skeleton sits exactly
// where the row it stands in for would.
func TestSkeletonContainerIsBare(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Skeleton{}.Render(ctx)

	if n.Type != "Box" {
		t.Errorf("container = %q, want Box (no theme base)", n.Type)
	}
	if (n.Style.Padding != core.EdgeInsets{}) {
		t.Errorf("container padding = %+v, want zero", n.Style.Padding)
	}
	if n.Style.Background != "" {
		t.Errorf("container background = %q, want none", n.Style.Background)
	}
}
