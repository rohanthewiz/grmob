package comps

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
	// the theme's Button base, with a rule in the control-boundary role.
	theme := core.DefaultTheme
	if n.Style.Background != theme.Colors.Surface {
		t.Errorf("unselected chip background = %q, want theme Surface %q", n.Style.Background, theme.Colors.Surface)
	}
	if n.Style.TextColor != theme.Colors.TextPrimary {
		t.Errorf("unselected chip ink = %q, want theme TextPrimary %q", n.Style.TextColor, theme.Colors.TextPrimary)
	}
	if n.Style.BorderWidth != 1 || n.Style.BorderColor != theme.Colors.ControlBorderColor() {
		t.Errorf("unselected chip rule = %vpx %q, want 1px %q",
			n.Style.BorderWidth, n.Style.BorderColor, theme.Colors.ControlBorderColor())
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

	// The name is the name in both states. It used to carry ", selected" on
	// the chosen chip, which meant the reader heard a *different control*
	// after a tap rather than the same one in a new state.
	for _, c := range []struct {
		what string
		node *core.Node
	}{{"unselected", unselected}, {"selected", selected}} {
		if got := c.node.Style.AccessibilityLabel; got != "Show all tasks" {
			t.Errorf("%s label = %q, want the plain name — the state is not part of it",
				c.what, got)
		}
	}
	// Both chips state a selection, and the unselected one is the half that
	// matters: a strip where only the chosen chip answers announces the rest
	// as plain buttons. See core.SelectedState.
	if got := unselected.Style.AccessibilitySelected; got != core.SelectedOff {
		t.Errorf("unselected state = %q, want %q — an unselected chip has to say so",
			got, core.SelectedOff)
	}
	if got := selected.Style.AccessibilitySelected; got != core.SelectedOn {
		t.Errorf("selected state = %q, want %q", got, core.SelectedOn)
	}
	if got := selected.Style.AccessibilityHint; got != "Filters the task list" {
		t.Errorf("hint = %q", got)
	}
}

// A chip with no accessibility label still states its selection.
//
// The two used to be one branch — the state was appended to the name, so a
// chip that was not named announced nothing about being chosen — and that is
// the case most chips in a tree are in: the caption usually names them well
// enough that nobody sets a label. Splitting the state out is what fixes it,
// and this is the half of the split with no caller to notice it was broken.
func TestChipStatesItsSelectionWithoutALabel(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := Chip{Label: "Active", Selected: true, OnTap: func() {}}.Render(ctx)
	if got := n.Style.AccessibilitySelected; got != core.SelectedOn {
		t.Errorf("selected state = %q, want %q", got, core.SelectedOn)
	}
	if got := n.Style.AccessibilityLabel; got != "" {
		t.Errorf("label = %q, want none — the caption names an unlabelled chip", got)
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
// # Why this runs under a fixture theme and not DefaultTheme
//
// The load-bearing assertion is that the outline is the accent's *tone* and
// not the accent, and it can only say that where the two are different hexes.
// DefaultTheme's Primary was darkened to a value that is ink-weight on its own
// (Colors.PrimaryOnLight now equals Colors.Primary), so under it the two
// implementations paint the same pixels and this test would pass on either.
// midTonePrimaryTheme (variant_test.go) is DefaultTheme as it stood before
// that move: systemBlue with a separate accessible tone.
func TestChipProminenceLoudIsAnOutlineNotAFill(t *testing.T) {
	theme := midTonePrimaryTheme()
	ctx := core.NewContext().WithTheme(theme)
	ctx.BeginRenderPass()

	accent := theme.Components.Button.Background
	// The outline is drawn in the accent's ink-weight tone, looked up by
	// colour because the accent is a hex the widget read off the Button base
	// rather than a role it named.
	ink := theme.Colors.OnLight(accent)
	if ink == accent {
		t.Fatalf("fixture no longer exercises the split: the Button base fill %q has no "+
			"separate on-light tone", accent)
	}

	c := Chip{Label: "$25", Prominence: ProminenceLoud, OnTap: func() {}}
	unselected := c.Render(ctx)
	c.Selected = true
	selected := c.Render(ctx)

	if unselected.Style.Background != ColorTransparent {
		t.Errorf("loud unselected fill = %q, want the transparent hole %q",
			unselected.Style.Background, ColorTransparent)
	}
	if unselected.Style.TextColor != ink || unselected.Style.BorderColor != ink {
		t.Errorf("loud unselected ink/rule = %q/%q, want the accent's on-light tone %q",
			unselected.Style.TextColor, unselected.Style.BorderColor, ink)
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
//
// It falls back to Primary and is then toned like any other accent, which is
// the composition worth pinning: the two steps are independent — one answers
// "which colour", the other "how dark" — and a fallback that skipped the
// second would spend the raw role colour on exactly the themes that had said
// least about their own colours. Under a theme whose Primary is a mid-tone
// that is a label nobody can read; under the bundled two it currently is not,
// because their Primary is ink-weight on its own.
func TestChipProminenceLoudFallsBackToPrimaryWithoutAButtonFill(t *testing.T) {
	fillless := &core.Theme{Colors: core.DefaultTheme.Colors}
	ctx := core.NewContext().WithTheme(fillless)
	ctx.BeginRenderPass()

	n := Chip{Label: "$25", Prominence: ProminenceLoud}.Render(ctx)

	want := fillless.Colors.PrimaryOnLightColor()
	if n.Style.TextColor != want || n.Style.BorderColor != want {
		t.Errorf("ink/rule = %q/%q, want the Primary role's on-light tone %q",
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

// The quiet chip's ring is the theme's control-boundary role, not its divider.
//
// This is the assertion the widget shipped without, and the reason it shipped
// is that nothing about a chip painted in Colors.Border *looks* broken in a
// tree dump: the border is set, it is a themed value, and it renders. What was
// wrong was the number, and a number is only wrong against a backdrop — which
// is why the second half measures rather than compares hexes.
func TestChipQuietRingIsTheBoundaryRoleAndNotTheDivider(t *testing.T) {
	for name, theme := range core.BundledThemes() {
		ctx := core.NewContext().WithTheme(theme)
		ctx.BeginRenderPass()

		n := Chip{Label: "2024"}.Render(ctx)

		if got, want := n.Style.BorderColor, theme.Colors.ControlBorderColor(); got != want {
			t.Errorf("%s: quiet ring = %q, want the boundary role %q", name, got, want)
		}
		if n.Style.BorderColor == theme.Colors.BorderColor() {
			t.Errorf("%s: quiet ring is the divider role — a filter chip drawn in a "+
				"hairline is chrome nobody can find", name)
		}
	}
}

// And the ring clears WCAG 1.4.11's floor against *both* backdrops a chip has.
//
// 3:1, not the 4.5:1 the ink assertions in this package use: this is non-text
// contrast, and the thing being identified is a control rather than read as
// words.
//
// # Why the inner backdrop is asserted now
//
// A chip sits on the page and is filled with Surface, so its ring has an
// outer edge and an inner one. This measured only the outer one for three
// sessions, and said so at length: the outer edge is what a reader picks the
// pill out by, because the fill is 1.12:1 against the page under DefaultTheme
// and identifies nothing on its own, while the inner pair is a boundary
// between two parts of one control. That was the argument for a 2.92:1 inner
// pair, and the last sentence of the comment that stood here was the tell —
// asserting the inner edge "would fail on a value that is Apple's own
// systemGray and correct".
//
// A test shaped around the value it must not fail on is a test that has
// stopped being able to find anything. The tone moved (see
// core.ColorPalette.ControlBorder), both pairs clear, and the loop measures
// both — so the next tone that clears the page and not the fill is caught
// here, at the widget, and not only in the palette census.
func TestChipQuietRingClearsTheNonTextContrastFloor(t *testing.T) {
	const wcagNonText = 3.0

	for name, theme := range core.BundledThemes() {
		ctx := core.NewContext().WithTheme(theme)
		ctx.BeginRenderPass()

		// The rendered node, not the palette role: this is the pair the
		// widget actually draws, fill and ring together.
		quiet := Chip{Label: "2024"}.Render(ctx).Style
		ringLum, ok := relativeLuminance(quiet.BorderColor)
		if !ok {
			t.Fatalf("%s: quiet ring %q is not a parseable hex", name, quiet.BorderColor)
		}

		for _, backdrop := range []struct{ what, hex string }{
			{"the page", theme.Colors.Background},
			{"its own fill", quiet.Background},
		} {
			lum, ok := relativeLuminance(backdrop.hex)
			if !ok {
				t.Fatalf("%s: %s %q is not a parseable hex", name, backdrop.what, backdrop.hex)
			}
			if r := contrastRatio(ringLum, lum); r < wcagNonText {
				t.Errorf("%s: quiet ring %q is %.2f:1 against %s (%q), want at least %.1f:1 "+
					"(WCAG 1.4.11 — the edge that identifies a control)",
					name, quiet.BorderColor, r, backdrop.what, backdrop.hex, wcagNonText)
			}
		}
	}
}
