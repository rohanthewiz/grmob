package comps

import (
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// radiosOf collects the swatch radios in tree order.
func radiosOf(n *core.Node) []*core.Node {
	var out []*core.Node
	walk(n, func(c *core.Node) {
		if c.Style.AccessibilityRole == core.RoleRadio {
			out = append(out, c)
		}
	})
	return out
}

func radioGroupOf(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	var g *core.Node
	walk(n, func(c *core.Node) {
		if c.Style.AccessibilityRole == core.RoleRadioGroup {
			g = c
		}
	})
	if g == nil {
		t.Fatal("no radiogroup in the tree")
	}
	return g
}

var brand = []Swatch{
	{Hex: "#2A78D6", Name: "Brand blue"},
	{Hex: "#fd0", Name: "Sunshine"},
	{Hex: "#101418", Name: "Ink"},
}

func TestNormalizeHex(t *testing.T) {
	for in, want := range map[string]string{
		"#2a78d6": "#2A78D6", "2A78D6": "#2A78D6", "#fd0": "#FFDD00", " #FFF ": "#FFFFFF",
		"": "", "#12": "", "#12345": "", "#2A78D680": "", "#ggg": "", "blue": "",
	} {
		got, ok := normalizeHex(in)
		if got != want || ok != (want != "") {
			t.Errorf("normalizeHex(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
}

// The default chart colours must come out as eight different names, since
// each is the only thing a screen reader has to tell its radio by.
func TestColorNames(t *testing.T) {
	for hex, want := range map[string]string{
		"#2A78D6": "blue", "#EB6834": "orange", "#1BAF7A": "teal", "#EDA100": "yellow",
		"#E87BA4": "pink", "#008300": "green", "#4A3AA7": "purple", "#E34948": "red",
		"#000000": "black", "#FFFFFF": "white", "#888888": "grey", "#DDDDDD": "light grey",
		"#0B2A5C": "dark blue", "#CFE3FF": "light blue",
	} {
		if got := colorName(hex); got != want {
			t.Errorf("colorName(%s) = %q, want %q", hex, got, want)
		}
	}
}

func TestColorSwatchPickerDefaultsToTheThemesChartColours(t *testing.T) {
	_, n := renderDebug(t, ColorSwatchPicker{OnChange: func(string) {}})

	g := radioGroupOf(t, n)
	if g.Style.AccessibilityLabel != "Colour" {
		t.Errorf("group label %q, want Colour", g.Style.AccessibilityLabel)
	}
	radios := radiosOf(n)
	want := core.DefaultChartColors()
	if len(radios) != len(want) {
		t.Fatalf("swatches = %d, want the theme's %d chart colours", len(radios), len(want))
	}
	names := map[string]bool{}
	for _, r := range radios {
		if r.Style.AccessibilityLabel == "" || names[r.Style.AccessibilityLabel] {
			t.Errorf("swatch name %q is empty or repeated", r.Style.AccessibilityLabel)
		}
		names[r.Style.AccessibilityLabel] = true
		if r.Style.AccessibilitySelected != core.SelectedOff {
			t.Errorf("%s is %q with no Value set", r.Style.AccessibilityLabel, r.Style.AccessibilitySelected)
		}
	}
	// Eight swatches at six across: two rows, the second padded with four
	// hidden blanks so its swatches are the width of the ones above.
	if len(g.Children) != 2 || len(g.Children[1].Children) != defaultSwatchColumns {
		t.Fatalf("rows = %d, last row cells = %d", len(g.Children), len(g.Children[1].Children))
	}
	if blank := g.Children[1].Children[5]; !blank.Style.AccessibilityHidden || blank.Style.FlexGrow != 1 {
		t.Error("the last row should be padded with hidden equal-share blanks")
	}
}

// Selected is a ring and a check, and the check's ink follows the swatch.
func TestColorSwatchPickerMarksTheChoice(t *testing.T) {
	th := core.DefaultTheme
	for _, c := range []struct{ value, name, ink string }{
		{"#ffdd00", "Sunshine", "#000000"}, // the short form and the case both match
		{"#101418", "Ink", "#FFFFFF"},
	} {
		_, n := renderDebug(t, ColorSwatchPicker{Colors: brand, Value: c.value, OnChange: func(string) {}})
		for _, r := range radiosOf(n) {
			chosen := r.Style.AccessibilityLabel == c.name
			if got := r.Style.AccessibilitySelected == core.SelectedOn; got != chosen {
				t.Errorf("value %s: %s selected = %v", c.value, r.Style.AccessibilityLabel, got)
			}
			check := findText(r, "✓")
			if (check != nil) != chosen {
				t.Errorf("value %s: %s check drawn = %v", c.value, r.Style.AccessibilityLabel, check != nil)
			}
			// The ring's border is always there, so nothing moves on select.
			if r.Style.BorderWidth != swatchRing {
				t.Errorf("%s border width %v, want %v selected or not", r.Style.AccessibilityLabel, r.Style.BorderWidth, swatchRing)
			}
			if chosen {
				if check.Style.TextColor != c.ink {
					t.Errorf("check on %s is %s, want %s", c.name, check.Style.TextColor, c.ink)
				}
				if r.Style.BorderColor != th.Colors.PrimaryOnLightColor() {
					t.Errorf("ring is %s, want PrimaryOnLight", r.Style.BorderColor)
				}
			} else if r.Style.BorderColor != ColorTransparent {
				t.Errorf("an unselected ring is %s, want transparent", r.Style.BorderColor)
			}
		}
	}
}

// A tap reports the swatch as #RRGGBB, once; a tap on the choice already made
// is not a change.
func TestColorSwatchPickerTapReports(t *testing.T) {
	var got []string
	ctx, n := renderDebug(t, ColorSwatchPicker{
		Colors: brand, Value: "#2A78D6", OnChange: func(h string) { got = append(got, h) },
	})
	r := radiosOf(n)
	ctx.TriggerCallback(r[1].Props["onClick"].(string))
	ctx.TriggerCallback(r[0].Props["onClick"].(string)) // already chosen
	if !slices.Equal(got, []string{"#FFDD00"}) {
		t.Errorf("reports = %v, want exactly [#FFDD00]", got)
	}
}

// The hex field commits only a whole colour, keeps the half-typed text as its
// own, and shows a custom choice in the preview.
func TestColorSwatchPickerCustomHex(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })

	value := "#2A78D6"
	var reports []string
	ctx := core.NewContext()
	render := func() *core.Node {
		ctx.BeginRenderPass()
		ctx.Reset()
		n := ColorSwatchPicker{
			Colors: brand, Value: value, AllowCustom: true,
			OnChange: func(h string) { value = h; reports = append(reports, h) },
		}.Render(ctx)
		ctx.EndRenderPass()
		core.AuditTree(n)
		return n
	}
	field := func(n *core.Node) *core.Node {
		var in *core.Node
		walk(n, func(c *core.Node) {
			if c.Type == "Input" {
				in = c
			}
		})
		if in == nil {
			t.Fatal("AllowCustom should draw a hex field")
		}
		return in
	}

	n := render()
	in := field(n)
	if in.Style.AccessibilityLabel != "Custom colour, hex" || in.Props["value"] != "" {
		t.Errorf("field name %q value %q", in.Style.AccessibilityLabel, in.Props["value"])
	}
	// The field is not a radio, so it must sit outside the radiogroup.
	walk(radioGroupOf(t, n), func(c *core.Node) {
		if c.Type == "Input" {
			t.Error("the hex field is inside the radiogroup")
		}
	})

	// Every step on the way to a six-digit colour, "#7B2" included: a valid
	// short colour that nobody chose, and which must not be reported.
	for _, typed := range []string{"#", "#7", "#7B", "#7B2", "#7B2F", "#7B2FF"} {
		ctx.TriggerTextCallback(field(n).Props["onChange"].(string), typed)
		n = render()
		if got := field(n).Props["value"]; got != typed {
			t.Fatalf("the draft should be kept as typed: %q, want %q", got, typed)
		}
	}
	if len(reports) != 0 {
		t.Fatalf("a half-typed colour was reported: %v", reports)
	}
	ctx.TriggerTextCallback(field(n).Props["onChange"].(string), "#7b2ff0")
	n = render()
	if !slices.Equal(reports, []string{"#7B2FF0"}) {
		t.Errorf("reports = %v, want exactly [#7B2FF0]", reports)
	}
	for _, r := range radiosOf(n) {
		if r.Style.AccessibilitySelected == core.SelectedOn {
			t.Errorf("%s is selected while the choice is a custom colour", r.Style.AccessibilityLabel)
		}
	}

	// The short form is a whole colour once the reader says so, with return.
	ctx.TriggerTextCallback(field(n).Props["onChange"].(string), "#fd0")
	n = render()
	if value != "#7B2FF0" {
		t.Fatalf("the short form committed while typing: %s", value)
	}
	ctx.TriggerCallback(field(n).Props["onSubmit"].(string))
	n = render()
	if value != "#FFDD00" {
		t.Errorf("after return: value %s, want #FFDD00", value)
	}
	// #FFDD00 is one of the swatches, so it is that swatch which is selected.
	if r := radiosOf(n)[1]; r.Style.AccessibilitySelected != core.SelectedOn {
		t.Error("a typed colour that is a swatch should select the swatch")
	}
	value = "#7B2FF0"
	n = render()

	// A swatch replaces the custom colour and clears the field.
	ctx.TriggerCallback(radiosOf(n)[2].Props["onClick"].(string))
	n = render()
	if value != "#101418" || field(n).Props["value"] != "" {
		t.Errorf("after a swatch: value %s, field %q", value, field(n).Props["value"])
	}
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("concerns raised:\n%s", dump)
	}
}

func TestColorSwatchPickerConcerns(t *testing.T) {
	for _, c := range []struct {
		name string
		p    ColorSwatchPicker
		want string
	}{
		{"no OnChange", ColorSwatchPicker{Colors: brand}, ConcernColorSwatchInert},
		{"an unnamed swatch", ColorSwatchPicker{Colors: []Swatch{{Hex: "#2A78D6"}}, OnChange: func(string) {}}, ConcernColorSwatchUnnamed},
		{"a colour no target parses", ColorSwatchPicker{Colors: []Swatch{{Hex: "cornflower", Name: "Cornflower"}}, OnChange: func(string) {}}, ConcernColorSwatchBadHex},
	} {
		t.Run(c.name, func(t *testing.T) {
			core.SetDebugMode(true)
			core.ClearConcerns()
			t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
			ctx := core.NewContext()
			ctx.BeginRenderPass()
			c.p.Render(ctx)
			ctx.EndRenderPass()
			if dump := core.DumpConcerns(); !strings.Contains(dump, c.want) {
				t.Errorf("want %s, got:\n%s", c.want, dump)
			}
		})
	}
}

// A colour listed twice is one swatch: the radios are keyed by hex.
func TestColorSwatchPickerDropsDuplicates(t *testing.T) {
	_, n := renderDebug(t, ColorSwatchPicker{
		Colors:   []Swatch{{Hex: "#fd0", Name: "Sunshine"}, {Hex: "#FFDD00", Name: "Again"}},
		OnChange: func(string) {},
	})
	if got := len(radiosOf(n)); got != 1 {
		t.Errorf("swatches = %d, want 1", got)
	}
}
