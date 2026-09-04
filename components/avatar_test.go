package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestAvatarImageBranch(t *testing.T) {
	ctx := core.NewContext()
	n := Avatar{Src: "https://example.com/ada.jpg", Name: "Ada Lovelace", Size: 64}.Render(ctx)

	if n.Type != "Image" {
		t.Fatalf("root type = %q, want Image when Src is set", n.Type)
	}
	if n.Props["src"] != "https://example.com/ada.jpg" {
		t.Errorf("src = %v", n.Props["src"])
	}
	if n.Style.Width != "64px" || n.Style.Height != "64px" {
		t.Errorf("avatar must be square: Width=%q Height=%q", n.Style.Width, n.Style.Height)
	}
	// Radius tracks the diameter rather than being a fixed oversized value, so
	// Size stays the single knob that shapes the widget.
	if n.Style.BorderRadius != 32 {
		t.Errorf("BorderRadius = %v, want half of Size (32)", n.Style.BorderRadius)
	}
	if n.Style.AccessibilityLabel != "Ada Lovelace" {
		t.Errorf("AccessibilityLabel = %q, want the Name", n.Style.AccessibilityLabel)
	}
	// core.Image's theme base is Components.Camera, whose background is black:
	// unoverridden, an avatar is a black disc until the image downloads.
	if n.Style.Background != core.DefaultTheme.Colors.Surface {
		t.Errorf("loading placeholder = %q, want the neutral Surface %q",
			n.Style.Background, core.DefaultTheme.Colors.Surface)
	}
}

func TestAvatarBackgroundOverrideAppliesToBothBranches(t *testing.T) {
	ctx := core.NewContext()
	for _, a := range []Avatar{{Src: "x.jpg", Background: "#123456"}, {Initials: "AL", Background: "#123456"}} {
		if n := a.Render(ctx); n.Style.Background != "#123456" {
			t.Errorf("%s branch: Background = %q, want the override", n.Type, n.Style.Background)
		}
	}
}

func TestAvatarFallbackDisc(t *testing.T) {
	ctx := core.NewContext()
	theme := core.DefaultTheme
	n := Avatar{Name: "Ada Lovelace"}.Render(ctx)

	if n.Type != "Row" {
		t.Fatalf("root type = %q — the disc must be a Row, which centres its child on both "+
			"axes through Justify and AlignItems with no theme padding to undo", n.Type)
	}
	if n.Style.JustifyContent != core.JustifyCenter || n.Style.AlignItems != core.AlignItemsCenter {
		t.Errorf("initials must be centred on both axes; got justify=%q align=%q",
			n.Style.JustifyContent, n.Style.AlignItems)
	}
	if n.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("the theme Row's padding must be zeroed or the disc exceeds Size; got %+v",
			n.Style.Padding)
	}
	if n.Style.Width != "40px" || n.Style.Height != "40px" {
		t.Errorf("default size should be 40px square; got %q x %q", n.Style.Width, n.Style.Height)
	}
	if n.Style.Background != theme.Colors.Primary {
		t.Errorf("disc fill = %q, want the theme Primary %q", n.Style.Background, theme.Colors.Primary)
	}

	txt := findText(n, "AL")
	if txt == nil {
		t.Fatal(`want initials "AL" on the disc`)
	}
	if txt.Style.TextColor != theme.Colors.Background {
		t.Errorf("ink = %q, want the palette's on-Primary color %q",
			txt.Style.TextColor, theme.Colors.Background)
	}
	// Proportional so the disc scales as one piece.
	if txt.Style.FontSize != 16 {
		t.Errorf("FontSize = %v, want 0.4 * 40", txt.Style.FontSize)
	}
}

func TestAvatarInitialsDerivation(t *testing.T) {
	cases := []struct {
		name, initials, want string
	}{
		{name: "Ada Lovelace", want: "AL"},
		// First and last, not first two: "Ada King Lovelace" is AL, not AK.
		{name: "Ada King Lovelace", want: "AL"},
		{name: "prince", want: "P"},
		{name: "  ada   lovelace  ", want: "AL"},
		{name: "", want: ""},
		// Explicit Initials win over anything derivable.
		{name: "Ada Lovelace", initials: "♥", want: "♥"},
		// Rune-based, so a multi-byte name keeps whole characters.
		{name: "Ольга Ладыженская", want: "ОЛ"},
		{name: "李 白", want: "李白"},
	}

	ctx := core.NewContext()
	for _, c := range cases {
		got := Avatar{Name: c.name, Initials: c.initials}.initials()
		if got != c.want {
			t.Errorf("initials(Name=%q, Initials=%q) = %q, want %q", c.name, c.initials, got, c.want)
		}
		// An empty derivation must still render a valid, childless disc rather
		// than panicking or emitting an empty Text node.
		if c.want == "" {
			n := Avatar{Name: c.name}.Render(ctx)
			if len(n.Children) != 0 {
				t.Errorf("a nameless avatar should render a bare disc, got %d children", len(n.Children))
			}
		}
	}
}

// An avatar with no name is decoration beside text that already names the
// person; unlabeled, a screen reader announces it as "image" or reads its URL.
func TestAvatarAccessibility(t *testing.T) {
	ctx := core.NewContext()

	explicit := Avatar{Src: "x.jpg", Name: "Ada", AccessibilityLabel: "Ada's photo"}.Render(ctx)
	if explicit.Style.AccessibilityLabel != "Ada's photo" {
		t.Errorf("explicit label must win over Name, got %q", explicit.Style.AccessibilityLabel)
	}
	if explicit.Style.AccessibilityHidden {
		t.Error("a labelled avatar must not also be hidden")
	}

	for _, a := range []Avatar{{Src: "x.jpg"}, {Initials: "AL"}} {
		n := a.Render(ctx)
		if !n.Style.AccessibilityHidden {
			t.Errorf("%+v: an unnamed avatar must be hidden from assistive tech", a)
		}
		if n.Style.AccessibilityLabel != "" {
			t.Errorf("%+v: hidden avatar should carry no label, got %q", a, n.Style.AccessibilityLabel)
		}
	}
}

func TestAvatarStyleAppliesToBothBranches(t *testing.T) {
	ctx := core.NewContext()
	style := []core.StyleProp{core.BorderRadius(4), core.BorderWidth(2), core.BorderColor("#FFFFFF")}

	img := Avatar{Src: "x.jpg", Style: style}.Render(ctx)
	disc := Avatar{Initials: "AL", Style: style}.Render(ctx)

	for _, n := range []*core.Node{img, disc} {
		if n.Style.BorderRadius != 4 {
			t.Errorf("%s: caller Style must be applied last and win, got radius %v",
				n.Type, n.Style.BorderRadius)
		}
		if n.Style.BorderWidth != 2 || n.Style.BorderColor != "#FFFFFF" {
			t.Errorf("%s: ring props dropped (BorderWidth/BorderColor reach the node only "+
				"because UseStyle-independent props are applied directly)", n.Type)
		}
	}
}
