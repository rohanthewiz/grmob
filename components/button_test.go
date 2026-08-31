package components

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The load-bearing property of both new axes: their zero values contribute
// nothing, so the widget's output is the theme's Button base byte for byte.
func TestButtonZeroValueIsExactlyCoreButton(t *testing.T) {
	for name, theme := range map[string]*core.Theme{
		"Default":  core.DefaultTheme,
		"Material": core.MaterialTheme,
	} {
		t.Run(name, func(t *testing.T) {
			ctx := core.NewContext().WithTheme(theme)
			ctx.BeginRenderPass()
			want := core.Button("Save", func() {}).Render(ctx)

			ctx = core.NewContext().WithTheme(theme)
			ctx.BeginRenderPass()
			got := Button{Label: "Save", OnTap: func() {}}.Render(ctx)

			// reflect.DeepEqual, not ==: Style carries a PseudoStates map and
			// is therefore not comparable.
			if !reflect.DeepEqual(*got.Style, *want.Style) {
				t.Fatalf("zero-value Button restyles the theme base:\n got %+v\nwant %+v",
					*got.Style, *want.Style)
			}
			if got.Props["label"] != want.Props["label"] {
				t.Fatalf("label = %v, want %v", got.Props["label"], want.Props["label"])
			}
		})
	}
}

// The previous test cannot tell "apply nothing" from "reapply the palette",
// because both bundled themes pair Components.Button with Colors.Primary and
// Colors.Background — so the two implementations agree there by coincidence.
//
// This is the case that separates them: a theme whose Button base deliberately
// differs from its palette roles, which is legal and is exactly what a
// re-deriving implementation would silently overwrite. It is also the shape a
// third-party theme is most likely to have, since Components.Button is where a
// theme expresses a house button style.
func TestButtonZeroValueDoesNotRederiveFromThePalette(t *testing.T) {
	// Palette says blue-on-white; the Button base says charcoal-on-amber. A
	// widget that "applies the default variant" lands on the palette pair.
	theme := &core.Theme{
		Colors: core.ColorPalette{
			Primary:    "#0000FF",
			Background: "#FFFFFF",
			Surface:    "#EEEEEE",
			Error:      "#FF0000",
		},
		Components: core.ComponentDefaults{
			Button: core.Style{Background: "#FFBF00", TextColor: "#222222"},
		},
	}

	ctx := core.NewContext().WithTheme(theme)
	ctx.BeginRenderPass()
	n := Button{Label: "Save", OnTap: func() {}}.Render(ctx)

	if n.Style.Background != "#FFBF00" || n.Style.TextColor != "#222222" {
		t.Fatalf("zero-value Button rendered %q on %q, want the theme's own Button "+
			"base (#222222 on #FFBF00) — the palette was re-derived over it",
			n.Style.TextColor, n.Style.Background)
	}
}

// Filled owns both the fill and the label, so it is the one emphasis whose
// contrast the widget can promise. This is Badge's guard applied through the
// widget rather than to Variant directly: a naive white ink would ship
// DefaultTheme's success and warning at ~2.2:1.
//
// VariantDefault is excluded for the same reason it is excluded from
// TestVariantInkIsLegibleOnEveryThemeAndVariant, and the exclusion is more
// pointed here. Its pairing is not the widget's: it is the theme's own
// Components.Button, which under DefaultTheme is white on #007AFF at 4.02:1 —
// below AA. Button applies no color props at all in that case, so failing it
// here would be reporting a palette decision as a widget defect, and "fixing"
// it would mean the zero value silently repaints every button in every tree.
// Recorded as a backlog item against the theme instead.
func TestButtonFilledStatusVariantsAreLegibleOnEveryTheme(t *testing.T) {
	const wcagAA = 4.5
	for themeName, theme := range map[string]*core.Theme{
		"Default":  core.DefaultTheme,
		"Material": core.MaterialTheme,
	} {
		for _, v := range []Variant{VariantSuccess, VariantWarning, VariantError} {
			ctx := core.NewContext().WithTheme(theme)
			ctx.BeginRenderPass()
			n := Button{Label: "Act", Variant: v, OnTap: func() {}}.Render(ctx)

			bgLum, ok := relativeLuminance(n.Style.Background)
			if !ok {
				t.Fatalf("%s/%s: unparseable background %q", themeName, v, n.Style.Background)
			}
			inkLum, ok := relativeLuminance(n.Style.TextColor)
			if !ok {
				t.Fatalf("%s/%s: unparseable ink %q", themeName, v, n.Style.TextColor)
			}
			if r := contrastRatio(bgLum, inkLum); r < wcagAA {
				t.Errorf("%s/%s: ink %q on %q is %.2f:1, below WCAG AA",
					themeName, v, n.Style.TextColor, n.Style.Background, r)
			}
		}
	}
}

// Outlined and Ghost both punch a real hole rather than omitting the fill: an
// empty Background inherits the theme's solid Button base, which is the
// opposite of the intent. Their label is the variant color verbatim — the
// documented contract, and the reason the type doc carries a contrast table
// instead of a guarantee.
func TestButtonOutlinedAndGhostAreTransparentWithVariantInk(t *testing.T) {
	theme := core.DefaultTheme
	for _, v := range []Variant{VariantDefault, VariantError} {
		t.Run(string("v="+v), func(t *testing.T) {
			ctx := core.NewContext().WithTheme(theme)
			ctx.BeginRenderPass()

			out := Button{Label: "Cancel", Variant: v, Emphasis: EmphasisOutlined}.Render(ctx)
			if out.Style.Background != ColorTransparent {
				t.Errorf("outlined fill = %q, want transparent", out.Style.Background)
			}
			if want := v.Color(theme); out.Style.TextColor != want {
				t.Errorf("outlined ink = %q, want the variant color %q", out.Style.TextColor, want)
			}
			if out.Style.BorderWidth == 0 || out.Style.BorderColor != v.Color(theme) {
				t.Errorf("outlined rule = %vpx %q, want 1px in the variant color",
					out.Style.BorderWidth, out.Style.BorderColor)
			}

			ghost := Button{Label: "Skip", Variant: v, Emphasis: EmphasisGhost}.Render(ctx)
			if ghost.Style.Background != ColorTransparent {
				t.Errorf("ghost fill = %q, want transparent", ghost.Style.Background)
			}
			if ghost.Style.BorderWidth != 0 {
				t.Errorf("ghost drew a rule of %vpx", ghost.Style.BorderWidth)
			}
			if want := v.Color(theme); ghost.Style.TextColor != want {
				t.Errorf("ghost ink = %q, want the variant color %q", ghost.Style.TextColor, want)
			}
		})
	}
}

// ColorTransparent has to survive every export path or the hole becomes an
// opaque black rectangle. The natives parse #RRGGBBAA in their own parseColor;
// this pins the HTML half, which emits the string verbatim.
func TestTransparentFillSurvivesTheHTMLExport(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Button{Label: "Skip", Emphasis: EmphasisGhost}.Render(ctx)
	if n.Style.Background != "#00000000" {
		t.Fatalf("ColorTransparent = %q; the 8-digit CSS byte order (alpha last) "+
			"is what both native parseColor implementations expect", n.Style.Background)
	}
}

func TestButtonStyleOverridesTheVariant(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Button{
		Label:   "Delete",
		Variant: VariantError,
		Style:   []core.StyleProp{core.BackgroundColor("#123456")},
	}.Render(ctx)

	if n.Style.Background != "#123456" {
		t.Fatalf("explicit Style lost to the variant: %q", n.Style.Background)
	}
}

// FullWidth needs the block display as well as the width: both bundled themes
// give Button an inline display, and width has no effect on an inline box.
func TestButtonFullWidthSetsWidthAndBlockDisplay(t *testing.T) {
	if core.DefaultTheme.Components.Button.Display != core.DisplayInline {
		t.Skip("theme Button base is no longer inline; the block half may be redundant")
	}
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Button{Label: "Continue", FullWidth: true}.Render(ctx)

	if n.Style.Width != "100%" {
		t.Errorf("width = %q, want 100%%", n.Style.Width)
	}
	if n.Style.Display != core.DisplayBlock {
		t.Errorf("display = %q, want block — width does nothing on an inline box",
			n.Style.Display)
	}
}

func TestButtonDisabled(t *testing.T) {
	theme := core.DefaultTheme

	t.Run("swallows taps", func(t *testing.T) {
		ctx := core.NewContext().WithTheme(theme)
		ctx.BeginRenderPass()
		tapped := false
		n := Button{Label: "Send", Disabled: true, OnTap: func() { tapped = true }}.Render(ctx)

		id, ok := n.Props["onClick"].(string)
		if !ok || id == "" {
			t.Fatalf("no onClick registered: %#v", n.Props)
		}
		// Dispatched rather than merely inspected: the handler must be a no-op
		// and not a nil func, which the registry would happily store and then
		// panic on when a late native tap arrives.
		ctx.TriggerCallback(id)
		if tapped {
			t.Fatal("a disabled button ran its handler")
		}
	})

	t.Run("nil OnTap is also safe to dispatch", func(t *testing.T) {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		n := Button{Label: "Inert"}.Render(ctx)
		ctx.TriggerCallback(n.Props["onClick"].(string)) // must not panic
	})

	t.Run("muted treatment overrides the variant", func(t *testing.T) {
		ctx := core.NewContext().WithTheme(theme)
		ctx.BeginRenderPass()
		n := Button{Label: "Delete", Variant: VariantError, Disabled: true}.Render(ctx)

		if n.Style.Background == theme.Colors.Error {
			t.Error("a disabled button still reads as danger")
		}
		if n.Style.Background != theme.Colors.Surface {
			t.Errorf("background = %q, want the palette's muted Surface", n.Style.Background)
		}
	})

	t.Run("announces the state", func(t *testing.T) {
		ctx := core.NewContext()
		ctx.BeginRenderPass()

		// No explicit label: the visible one is promoted so the suffix has
		// something to attach to.
		n := Button{Label: "Send", Disabled: true}.Render(ctx)
		if got := n.Style.AccessibilityLabel; got != "Send, disabled" {
			t.Errorf("label = %q, want %q", got, "Send, disabled")
		}

		// With one: the suffix rides the explicit name, which is the case
		// that matters for a glyph button.
		n = Button{Label: "✕", AccessibilityLabel: "Delete task", Disabled: true}.Render(ctx)
		if got := n.Style.AccessibilityLabel; got != "Delete task, disabled" {
			t.Errorf("label = %q, want %q", got, "Delete task, disabled")
		}

		// Enabled buttons gain no suffix and no synthesized label.
		n = Button{Label: "Send"}.Render(ctx)
		if n.Style.AccessibilityLabel != "" {
			t.Errorf("enabled button synthesized a label: %q", n.Style.AccessibilityLabel)
		}
	})

	t.Run("the suffix cannot be clobbered by Style", func(t *testing.T) {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		n := Button{
			Label:    "Send",
			Disabled: true,
			Style:    []core.StyleProp{core.AccessibilityLabel("Send")},
		}.Render(ctx)
		if !strings.HasSuffix(n.Style.AccessibilityLabel, ", disabled") {
			t.Errorf("label = %q; the state announcement must outlast a caller style",
				n.Style.AccessibilityLabel)
		}
	})
}

func TestButtonAccessibilityHint(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Button{Label: "Add", AccessibilityHint: "Adds the task typed in the field"}.Render(ctx)
	if n.Style.AccessibilityHint != "Adds the task typed in the field" {
		t.Fatalf("hint = %q", n.Style.AccessibilityHint)
	}
}
