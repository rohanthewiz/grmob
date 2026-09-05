package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestChipRendersAsButton(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	tapped := false
	n := Chip{Label: "Active", OnTap: func() { tapped = true }}.Render(ctx)

	if n.Type != "Button" || n.Props["label"] != "Active" {
		t.Fatalf("chip should be a themed Button, got %q %v", n.Type, n.Props)
	}
	id, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatal("chip should register an onClick callback")
	}
	ctx.TriggerCallback(id)
	if !tapped {
		t.Error("tapping the chip should invoke OnTap")
	}
	// Unselected is the quiet state: a Surface fill and TextPrimary ink over
	// the theme's Button base, with a hairline rule.
	theme := core.DefaultTheme
	if n.Style.Background != theme.Colors.Surface {
		t.Errorf("unselected chip background = %q, want theme Surface %q", n.Style.Background, theme.Colors.Surface)
	}
	if n.Style.TextColor != theme.Colors.TextPrimary {
		t.Errorf("unselected chip ink = %q, want theme TextPrimary %q", n.Style.TextColor, theme.Colors.TextPrimary)
	}
	if n.Style.BorderWidth != 1 || n.Style.BorderColor != theme.Colors.BorderColor() {
		t.Errorf("unselected chip rule = %vpx %q, want 1px %q",
			n.Style.BorderWidth, n.Style.BorderColor, theme.Colors.BorderColor())
	}
}

// The selected chip is the loud one, and it is loud by *not* painting: the
// theme's Components.Button carries the fill and the ink through untouched,
// so a theme whose buttons are not primary-coloured keeps its own look. The
// only thing the default contributes is the ring, and the ring is the fill,
// so it cannot be seen — it is there to hold the same box as the unselected
// chip's visible rule.
func TestChipSelectedThemeDefault(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Chip{Label: "Done", Selected: true, OnTap: func() {}}.Render(ctx)

	theme := core.DefaultTheme
	base := theme.Components.Button
	if n.Style.Background != base.Background {
		t.Errorf("selected default background = %q, want the Button base %q", n.Style.Background, base.Background)
	}
	if n.Style.TextColor != base.TextColor {
		t.Errorf("selected default ink = %q, want the Button base %q", n.Style.TextColor, base.TextColor)
	}
	if n.Style.BorderWidth != 1 || n.Style.BorderColor != base.Background {
		t.Errorf("selected ring = %vpx %q, want 1px of the fill %q",
			n.Style.BorderWidth, n.Style.BorderColor, base.Background)
	}
}

// A theme whose Button base has no fill of its own has nothing for the ring
// to hide against, so the ring goes transparent rather than being guessed at
// — it still holds the pixel, which is all it is for.
func TestChipSelectedRingIsTransparentWithoutAButtonFill(t *testing.T) {
	fillless := &core.Theme{Colors: core.DefaultTheme.Colors}
	ctx := core.NewContext().WithTheme(fillless)
	ctx.BeginRenderPass()
	n := Chip{Label: "Done", Selected: true}.Render(ctx)

	if n.Style.BorderWidth != 1 || n.Style.BorderColor != ColorTransparent {
		t.Errorf("ring = %vpx %q, want 1px %q", n.Style.BorderWidth, n.Style.BorderColor, ColorTransparent)
	}
}

// UnselectedStyle is the other half of the pair, and it distinguishes nil
// from empty: nil takes the default, an allocated empty slice drops it, which
// is how a caller gets the pre-inversion look back.
func TestChipUnselectedStyleOverride(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	custom := Chip{Label: "All", UnselectedStyle: []core.StyleProp{core.BackgroundColor("#FFF8E1")}}.Render(ctx)
	if custom.Style.Background != "#FFF8E1" {
		t.Errorf("UnselectedStyle should replace the default: %q", custom.Style.Background)
	}
	if custom.Style.BorderWidth != 0 {
		t.Error("an override replaces the whole default, the rule included")
	}

	bare := Chip{Label: "All", UnselectedStyle: []core.StyleProp{}}.Render(ctx)
	if bare.Style.Background != core.DefaultTheme.Components.Button.Background {
		t.Errorf("an empty (non-nil) UnselectedStyle should apply nothing, leaving the "+
			"Button base: %q", bare.Style.Background)
	}
}

// Style is shared across both states and the state wins where they collide —
// otherwise one Style handed to a whole strip would flatten the distinction
// the strip exists to draw.
func TestChipStateStyleBeatsSharedStyle(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	c := Chip{
		Label: "All",
		Style: []core.StyleProp{core.BackgroundColor("#123456"), core.FontSize(13)},
	}
	unselected := c.Render(ctx)
	c.Selected = true
	selected := c.Render(ctx)

	if unselected.Style.Background != core.DefaultTheme.Colors.Surface {
		t.Errorf("unselected background = %q, want the state default to win", unselected.Style.Background)
	}
	if selected.Style.Background != core.DefaultTheme.Components.Button.Background {
		t.Errorf("selected background = %q, want the Button base to win", selected.Style.Background)
	}
	// The fields the state says nothing about still come through.
	if unselected.Style.FontSize != 13 || selected.Style.FontSize != 13 {
		t.Error("Style's non-colliding fields should survive in both states")
	}
}

func TestChipSelectedStyleOverride(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Chip{
		Label:    "Done",
		Selected: true,
		OnTap:    func() {},
		SelectedStyle: []core.StyleProp{
			core.BackgroundColor("#E8F0FE"),
			core.TextColor("#0B57D0"),
		},
	}.Render(ctx)

	if n.Style.Background != "#E8F0FE" || n.Style.TextColor != "#0B57D0" {
		t.Errorf("SelectedStyle should replace the theme default: %+v", n.Style)
	}
}

func TestChipAccessibilityAnnouncesSelection(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	base := Chip{Label: "All", OnTap: func() {}, AccessibilityLabel: "Show all tasks", AccessibilityHint: "Filters the task list"}
	unselected := base.Render(ctx)
	base.Selected = true
	selected := base.Render(ctx)

	if got := unselected.Style.AccessibilityLabel; got != "Show all tasks" {
		t.Errorf("unselected label = %q", got)
	}
	if got := selected.Style.AccessibilityLabel; got != "Show all tasks, selected" {
		t.Errorf("selected label = %q, want the state appended", got)
	}
	if got := selected.Style.AccessibilityHint; got != "Filters the task list" {
		t.Errorf("hint = %q", got)
	}
}

// TestChipNilOnTapDoesNotPanic pins the nil-handler guard. A decorative chip
// (a tag, a status pill) is a reasonable thing to write, and Chip handed a nil
// OnTap straight to core.Button, which registers whatever it is given; the
// registry then invoked it unguarded on the first tap.
func TestChipNilOnTapDoesNotPanic(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := Chip{Label: "Tag"}.Render(ctx)
	id, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatal("chip should still register an onClick callback with no OnTap")
	}
	if err := core.Guard(func() { ctx.TriggerCallback(id) }); err != nil {
		t.Fatalf("tapping a chip with no OnTap panicked: %v", err.Value)
	}
}

// ProminenceLoud draws the unselected chip as an outline in the chip's own
// accent — the fill the selected state paints — rather than in a Surface grey.
//
// The half worth pinning is that it is *not* a return to the pre-inversion
// look: that one painted every unselected chip a solid fill and left the
// chosen one pale. Here the fill is transparent, so the selected chip is still
// the only solid pill in the row, and a row of suggestions is exactly the
// shape that would hide a quiet slide back.
func TestChipProminenceLoudIsAnOutlineNotAFill(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	theme := core.DefaultTheme
	accent := theme.Components.Button.Background

	c := Chip{Label: "$25", Prominence: ProminenceLoud, OnTap: func() {}}
	unselected := c.Render(ctx)
	c.Selected = true
	selected := c.Render(ctx)

	if unselected.Style.Background != ColorTransparent {
		t.Errorf("loud unselected fill = %q, want the transparent hole %q",
			unselected.Style.Background, ColorTransparent)
	}
	if unselected.Style.TextColor != accent || unselected.Style.BorderColor != accent {
		t.Errorf("loud unselected ink/rule = %q/%q, want the chip's accent %q",
			unselected.Style.TextColor, unselected.Style.BorderColor, accent)
	}
	if unselected.Style.BorderWidth != 1 {
		t.Errorf("loud unselected rule width = %v, want 1", unselected.Style.BorderWidth)
	}

	// The selected chip is untouched by the field: it is the theme's Button
	// base in both prominences, and it has to stay the loudest thing here.
	if selected.Style.Background != accent {
		t.Errorf("loud selected background = %q, want the Button base %q — Prominence "+
			"must not reach the selected state", selected.Style.Background, accent)
	}
	if selected.Style.TextColor != theme.Components.Button.TextColor {
		t.Errorf("loud selected ink = %q, want the Button base %q",
			selected.Style.TextColor, theme.Components.Button.TextColor)
	}
}

// The zero value is the quiet state, so adding the field restyles nothing that
// already exists. Pinned against the quiet chip rendered beside it rather than
// against a copy of the palette, so the two can only agree.
func TestChipProminenceZeroValueIsTheQuietDefault(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	implicit := Chip{Label: "2024"}.Render(ctx)
	explicit := Chip{Label: "2024", Prominence: ProminenceQuiet}.Render(ctx)

	if implicit.Style.Background != explicit.Style.Background ||
		implicit.Style.TextColor != explicit.Style.TextColor ||
		implicit.Style.BorderColor != explicit.Style.BorderColor {
		t.Errorf("the zero Prominence should be the quiet default: %+v vs %+v",
			implicit.Style, explicit.Style)
	}
	if implicit.Style.Background != core.DefaultTheme.Colors.Surface {
		t.Errorf("quiet fill = %q, want theme Surface", implicit.Style.Background)
	}
}

// UnselectedStyle is the more specific of the two knobs and is reached first;
// otherwise a caller who set both would find the override unreachable.
func TestChipUnselectedStyleBeatsProminence(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := Chip{
		Label:           "$25",
		Prominence:      ProminenceLoud,
		UnselectedStyle: []core.StyleProp{core.BackgroundColor("#FFF8E1")},
	}.Render(ctx)

	if n.Style.Background != "#FFF8E1" {
		t.Errorf("background = %q, want the explicit override to win over Prominence",
			n.Style.Background)
	}
	if n.Style.BorderWidth != 0 {
		t.Error("an override replaces the whole treatment, the loud rule included")
	}
}

// A theme whose Button base carries no fill has no accent to read, and the
// fallback here is the palette's Primary rather than chipRing's transparent:
// this colour is ink and a visible rule, and transparent ink is an invisible
// chip.
func TestChipProminenceLoudFallsBackToPrimaryWithoutAButtonFill(t *testing.T) {
	fillless := &core.Theme{Colors: core.DefaultTheme.Colors}
	ctx := core.NewContext().WithTheme(fillless)
	ctx.BeginRenderPass()

	n := Chip{Label: "$25", Prominence: ProminenceLoud}.Render(ctx)

	want := fillless.Colors.Primary
	if n.Style.TextColor != want || n.Style.BorderColor != want {
		t.Errorf("ink/rule = %q/%q, want the palette's Primary %q",
			n.Style.TextColor, n.Style.BorderColor, want)
	}
}

// Style still applies to both states and still loses to the treatment, loud
// included — the rule the widget already holds for the quiet default.
func TestChipLoudTreatmentBeatsSharedStyle(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := Chip{
		Label:      "$25",
		Prominence: ProminenceLoud,
		Style:      []core.StyleProp{core.BackgroundColor("#123456"), core.FontSize(13)},
	}.Render(ctx)

	if n.Style.Background != ColorTransparent {
		t.Errorf("background = %q, want the loud treatment to win over Style", n.Style.Background)
	}
	if n.Style.FontSize != 13 {
		t.Error("Style's non-colliding fields should still survive")
	}
}
