package components

import (
	"math"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// fillOf returns the track's fill child.
func fillOf(n *core.Node) *core.Node {
	if len(n.Children) == 0 {
		return nil
	}
	return n.Children[0]
}

func TestProgressBarStructureAndDefaults(t *testing.T) {
	ctx := core.NewContext()
	theme := core.DefaultTheme
	n := ProgressBar{Value: 0.45}.Render(ctx)

	if n.Type != "Row" {
		t.Fatalf("track type = %q, want Row — the fill lays out along the main axis", n.Type)
	}
	if n.Style.Height != "6px" {
		t.Errorf("track Height = %q, want the 6px default", n.Style.Height)
	}
	if n.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("a 6px groove cannot carry the theme Row's 8px inset; got %+v", n.Style.Padding)
	}
	if n.Style.Background != theme.Colors.Surface {
		t.Errorf("track = %q, want theme Surface %q", n.Style.Background, theme.Colors.Surface)
	}
	if n.Style.BorderRadius != 3 {
		t.Errorf("BorderRadius = %v, want half the thickness for pill ends", n.Style.BorderRadius)
	}

	f := fillOf(n)
	if f == nil || f.Type != "Box" {
		t.Fatalf("want a single Box fill, got %+v", n.Children)
	}
	if f.Style.Width != "45%" {
		t.Errorf("fill Width = %q, want %q", f.Style.Width, "45%")
	}
	// A Compose Box and a SwiftUI ZStack both size to content, and the fill
	// has none: without its own height it is a zero-pixel line in a good track.
	if f.Style.Height != "6px" {
		t.Errorf("fill Height = %q, want to match the track", f.Style.Height)
	}
	if f.Style.Background != theme.Colors.Primary {
		t.Errorf("fill = %q, want theme Primary %q", f.Style.Background, theme.Colors.Primary)
	}
}

// The fill must be sized by width, not by a FlexGrow pair. FlexGrow maps onto
// frame(maxWidth: .infinity) on iOS, where two growers split space equally
// regardless of their weights — every bar would sit at 50% on that platform
// with nothing in the tree to show for it.
func TestProgressBarUsesWidthNotFlexWeights(t *testing.T) {
	ctx := core.NewContext()
	n := ProgressBar{Value: 0.3}.Render(ctx)

	if len(n.Children) != 1 {
		t.Fatalf("want exactly one child — a second, weighted remainder box is the "+
			"construction that breaks on iOS; got %d", len(n.Children))
	}
	if got := fillOf(n).Style.FlexGrow; got != 0 {
		t.Errorf("fill FlexGrow = %v, want 0: proportion comes from Width", got)
	}
}

func TestProgressBarValueClamping(t *testing.T) {
	cases := []struct {
		value float64
		want  string
	}{
		{0, "0%"},
		{1, "100%"},
		{0.5, "50%"},
		// Formatting must not leak float64 noise: %g of 0.333*100 is
		// "33.300000000000004".
		{0.333, "33.3%"},
		{0.123456, "12.35%"},
		{-2, "0%"},
		{7, "100%"},
		{math.NaN(), "0%"},
	}

	ctx := core.NewContext()
	for _, c := range cases {
		n := ProgressBar{Value: c.value}.Render(ctx)
		f := fillOf(n)
		if f == nil {
			t.Fatalf("value %v: the fill must render at every value — a constant child count "+
				"keeps progress a style patch instead of a tree edit", c.value)
		}
		if f.Style.Width != c.want {
			t.Errorf("value %v: Width = %q, want %q", c.value, f.Style.Width, c.want)
		}
	}
}

func TestProgressBarAccessibility(t *testing.T) {
	ctx := core.NewContext()

	// No renderer has a progress semantic, so the value has to travel in the
	// label or it is never announced.
	n := ProgressBar{Value: 0.456, AccessibilityLabel: "Upload"}.Render(ctx)
	if n.Style.AccessibilityLabel != "Upload, 46 percent" {
		t.Errorf("label = %q, want the rounded percentage appended", n.Style.AccessibilityLabel)
	}
	if n.Style.AccessibilityHidden {
		t.Error("a labelled bar must not also be hidden")
	}
	// Rounding is for the announcement only.
	if got := fillOf(n).Style.Width; got != "45.6%" {
		t.Errorf("fill Width = %q, want the exact value, not the rounded one", got)
	}

	bare := ProgressBar{Value: 0.5}.Render(ctx)
	if !bare.Style.AccessibilityHidden {
		t.Error("an unlabeled bar announces a number with nothing to attach it to; hide it")
	}
}

func TestProgressBarThicknessAndColorOverrides(t *testing.T) {
	ctx := core.NewContext()
	n := ProgressBar{
		Value:      0.5,
		Thickness:  10,
		Color:      "#00FF00",
		TrackColor: "#EEEEEE",
	}.Render(ctx)

	if n.Style.Height != "10px" || n.Style.BorderRadius != 5 {
		t.Errorf("track = %q/%v, want 10px with a radius of 5", n.Style.Height, n.Style.BorderRadius)
	}
	f := fillOf(n)
	if f.Style.Height != "10px" || f.Style.BorderRadius != 5 {
		t.Errorf("fill must match the track's geometry, got %q/%v", f.Style.Height, f.Style.BorderRadius)
	}
	if n.Style.Background != "#EEEEEE" || f.Style.Background != "#00FF00" {
		t.Errorf("colors not applied: track=%q fill=%q", n.Style.Background, f.Style.Background)
	}
}

func TestProgressBarStyleOverridesDefaults(t *testing.T) {
	ctx := core.NewContext()
	n := ProgressBar{Value: 0.5, Style: []core.StyleProp{
		core.Width("200px"),
		core.BackgroundColor("#000000"),
	}}.Render(ctx)

	if n.Style.Width != "200px" {
		t.Errorf("caller Width = %q, want to reach the track", n.Style.Width)
	}
	if n.Style.Background != "#000000" {
		t.Errorf("caller Style must be applied last and win, got %q", n.Style.Background)
	}
}
