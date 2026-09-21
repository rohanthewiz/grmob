package comps

import (
	"math"
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernColorSwatchInert is raised, in debug builds only, when a
// ColorSwatchPicker has no OnChange and is not Disabled: a grid of targets
// that select nothing.
const ConcernColorSwatchInert = "color-swatch-inert"

// ConcernColorSwatchUnnamed is raised, in debug builds only, when a swatch the
// caller supplied has no Name. It is then spoken by a name guessed from its
// hue ("blue"), which cannot tell "Brand blue" from "Link blue" and knows
// nothing of what the colour means in the application. The caller does.
const ConcernColorSwatchUnnamed = "color-swatch-unnamed"

// ConcernColorSwatchBadHex is raised, in debug builds only, when a swatch's
// Hex is not "#rgb" or "#rrggbb". Such a swatch is not drawn: a fill no target
// can parse paints nothing, and an invisible target that still selects is
// worse than a missing one.
const ConcernColorSwatchBadHex = "color-swatch-bad-hex"

const (
	// defaultSwatchColumns suits a phone's width at the swatch height below:
	// six across leaves each swatch a little wider than it is tall.
	defaultSwatchColumns = 6

	// swatchHeight is a swatch's drawn height in points, inside its ring. With
	// the ring it is a 50 point target, above every platform's minimum.
	swatchHeight = 44

	// swatchRing is the width of the selection ring, and of the transparent
	// border every unselected swatch carries in its place so that selecting
	// one moves nothing.
	swatchRing = 2

	// swatchPad is the space between the ring and the colour, so the ring
	// reads as a ring round the swatch and not as its border.
	swatchPad = 2
)

// Swatch is one colour on offer.
type Swatch struct {
	// Hex is the colour, "#rgb" or "#rrggbb", and the value reported.
	Hex string

	// Name is what a screen reader says: "Brand blue", "Sunset". Empty falls
	// back to a name guessed from the hue and reports
	// ConcernColorSwatchUnnamed.
	Name string
}

// ColorSwatchPicker chooses one colour from a set: a label's colour, a
// calendar's, an avatar's background.
//
//	comps.ColorSwatchPicker{
//	    Label:    "Label colour",
//	    Value:    colour.Get(),
//	    OnChange: colour.Set,
//	}
//
//	┌ Column ───────────────────────────────────────┐
//	│ ┌ Column  role=radiogroup ──────────────────┐ │
//	│ │ ┌ Row ──────────────────────────────────┐ │ │
//	│ │ │ ▇▇  ▇▇  ┏▇✓┓  ▇▇  ▇▇  ▇▇              │ │ │  each a role=radio;
//	│ │ └───────────────────────────────────────┘ │ │  the chosen one ringed
//	│ │ ┌ Row ──────────────────────────────────┐ │ │  and checked
//	│ │ │ ▇▇  ▇▇  ··  ··  ··  ··                │ │ │  blanks keep the columns
//	│ │ └───────────────────────────────────────┘ │ │
//	│ └───────────────────────────────────────────┘ │
//	│ ┌ Row ─ only with AllowCustom ──────────────┐ │
//	│ │ ▇▇  [ #RRGGBB                           ] │ │
//	│ └───────────────────────────────────────────┘ │
//	└───────────────────────────────────────────────┘
//
// # The default palette
//
// With no Colors the picker offers the theme's chart colours
// (ColorPalette.ChartColors). They are already chosen to be told apart from
// one another and validated against the theme's surface, which is exactly
// what a set of label colours needs, and a second hard-coded palette here
// would be a second list to keep in step with every theme. They are named
// from their hues, which for eight deliberately distinct colours is enough.
//
// # What selected looks like
//
// A ring in PrimaryOnLight round the swatch, and a check on it. The check's
// ink is whichever of black and white has the better contrast against that
// swatch (contrastInk), so it reads on yellow and on navy alike. Colour alone
// never carries the state: the ring has a shape and the check is a glyph.
// Every swatch carries the ring's border, transparent when unselected, so a
// selection never shifts its neighbours.
//
// # Custom colours
//
// AllowCustom adds a hex field under the grid. It commits the moment its text
// is a whole six-digit colour ("#2A78D6", with or without the #) and ignores
// everything else, so a half-typed value never reaches the caller. The short
// form ("#2a7") commits on the keyboard's return action only: every six-digit
// colour passes through a valid three-digit one on its way, and committing
// that would hand the caller a colour nobody chose, mid-word. The
// half-typed text is the widget's own (TagInput's reasoning: nobody wants
// "#2A7" in their state as a colour), which makes this a hook-owning widget:
// render it unconditionally, in a stable position. The hook is taken whether
// or not AllowCustom is set, so flipping the field cannot shift a slot.
//
// A Value that is none of the swatches is shown in the preview beside the
// field, selected, so a custom colour chosen earlier is still visibly the
// choice.
//
// # What it is not
//
// A hue and saturation square. That needs the position of a touch on a
// Canvas, and no event carries one (the round-four plan's Phase 6).
//
// # Accessibility
//
// The grid is a core.RoleRadioGroup named by Label and each swatch a
// core.RoleRadio with AccessibilitySelected, RadioGroup's pair. The rows
// between them are layout only. The hex field is outside the group (a
// radiogroup's members are radios) and named CustomLabel.
//
// # Theme roles read
//
//	Ring          Colors.PrimaryOnLight
//	Swatch edge   ColorPalette.BorderColor hairline, so white shows on white
//	Check         black or white, by contrast with the swatch
//	Gaps          Spacing.XS between swatches, Spacing.SM above the field
type ColorSwatchPicker struct {
	// Colors are the swatches on offer. Empty means the theme's chart
	// colours; see "The default palette".
	Colors []Swatch

	// Value is the chosen colour. It is compared without regard to case or
	// to the short form, so "#2a7" selects a "#22AA77" swatch.
	Value string

	// OnChange receives the chosen colour as "#RRGGBB". Nil reports
	// ConcernColorSwatchInert unless Disabled.
	OnChange func(hex string)

	// Columns is the number of swatches across. Zero means six.
	Columns int

	// Label is the group's accessible name. Empty means "Colour".
	Label string

	// AllowCustom adds the hex field under the grid.
	AllowCustom bool

	// CustomLabel is the hex field's accessible name. Empty means
	// "Custom colour, hex".
	CustomLabel string

	// Disabled greys the swatches and the field and drops their reports.
	Disabled bool

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}

// Render draws the grid, and the hex field when AllowCustom.
func (p ColorSwatchPicker) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	swatches := p.swatches(t)
	value, _ := normalizeHex(p.Value)
	custom := value != "" && !hasSwatch(swatches, value)

	// Before any branch. It starts as the Value when that is a custom colour,
	// so a restored screen shows the hex it was given.
	initial := ""
	if custom {
		initial = value
	}
	draft := core.NewState(ctx, initial)

	if core.IsDebugMode() && p.OnChange == nil && !p.Disabled {
		core.ReportConcern(ConcernColorSwatchInert,
			"ColorSwatchPicker has no OnChange and is not Disabled, so no swatch can be chosen")
	}

	choose := func(hex string) {
		if p.Disabled || p.OnChange == nil || hex == value {
			return
		}
		p.OnChange(hex)
	}

	items := make([]core.PropsAndChildren, 0, len(p.Style)+5)
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.Width("100%"),
	)
	items = append(items, asProps(p.Style)...)
	items = append(items, p.grid(t, swatches, value, func(hex string) {
		// A swatch replaces a custom colour, so the field must not go on
		// showing the hex that is no longer the choice.
		draft.Set("")
		choose(hex)
	}))

	if p.AllowCustom {
		items = append(items, p.customRow(t, draft.Get(), value, custom,
			func(typed string) {
				draft.Set(typed)
				// Six digits only while typing; see "Custom colours".
				if hex, ok := normalizeHex(typed); ok && len(strings.TrimPrefix(strings.TrimSpace(typed), "#")) == 6 {
					choose(hex)
				}
			},
			func() {
				if hex, ok := normalizeHex(draft.Get()); ok {
					choose(hex)
				}
			}))
	}
	return core.Column(items...).Render(ctx)
}

// swatches resolves Colors: the caller's, with unparseable ones dropped and
// unnamed ones named, or the theme's chart colours.
func (p ColorSwatchPicker) swatches(t *core.Theme) []Swatch {
	if len(p.Colors) == 0 {
		// Deduplicated as chartPalette does, without its alpha repeats: a
		// tint of a colour already offered is not a second choice.
		seen := map[string]bool{}
		names := map[string]int{}
		var out []Swatch
		for _, c := range t.Colors.ChartColors() {
			hex, ok := normalizeHex(c)
			if !ok || seen[hex] {
				continue
			}
			seen[hex] = true
			// Two blues in one theme become "blue" and "blue 2", so every
			// radio in the group has a name of its own.
			name := colorName(hex)
			names[name]++
			if k := names[name]; k > 1 {
				name += " " + strconv.Itoa(k)
			}
			out = append(out, Swatch{Hex: hex, Name: name})
		}
		return out
	}

	out := make([]Swatch, 0, len(p.Colors))
	seen := map[string]bool{}
	for _, s := range p.Colors {
		hex, ok := normalizeHex(s.Hex)
		if !ok {
			if core.IsDebugMode() {
				core.ReportConcern(ConcernColorSwatchBadHex,
					"ColorSwatchPicker swatch \""+s.Hex+"\" is not #rgb or #rrggbb and is not drawn")
			}
			continue
		}
		// A colour listed twice is offered once. Both copies would report the
		// same value and both would draw as selected, and the swatches are
		// keyed by their hex, which siblings may not share.
		if seen[hex] {
			continue
		}
		seen[hex] = true
		name := s.Name
		if name == "" {
			if core.IsDebugMode() {
				core.ReportConcern(ConcernColorSwatchUnnamed,
					"ColorSwatchPicker swatch "+hex+" has no Name, so it is spoken by a guess from its hue")
			}
			name = colorName(hex)
		}
		out = append(out, Swatch{Hex: hex, Name: name})
	}
	return out
}

// grid is the radiogroup: rows of Columns equal shares.
func (p ColorSwatchPicker) grid(t *core.Theme, swatches []Swatch, value string, choose func(string)) core.View {
	cols := p.Columns
	if cols <= 0 {
		cols = defaultSwatchColumns
	}
	gap := float64(t.Spacing.XS)

	rows := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(gap),
		core.Width("100%"),
		core.AccessibilityRole(core.RoleRadioGroup),
		core.AccessibilityLabel(orDefault(p.Label, "Colour")),
	}
	for start := 0; start < len(swatches); start += cols {
		row := []core.PropsAndChildren{core.Padding(0), core.Gap(gap), core.Width("100%")}
		for i := start; i < start+cols; i++ {
			if i < len(swatches) {
				s := swatches[i]
				row = append(row, core.Keyed(s.Hex, p.swatch(t, s, s.Hex == value, func() { choose(s.Hex) })))
			} else {
				// A short last row is padded, so its swatches are the width
				// of the ones above and not stretched across the row.
				// It carries a swatch's padding and ring width as well: on
				// the web a share is the zero basis plus the cell's own
				// padding and border, and a bare box made the last row's
				// swatches wider than the rest (seen in headless Chrome).
				row = append(row, core.Box(core.FlexGrow(1), core.FlexBasis("0"),
					core.Padding(swatchPad), core.BorderWidth(swatchRing),
					core.BorderColor(ColorTransparent), core.AccessibilityHidden()))
			}
		}
		rows = append(rows, core.Row(row...))
	}
	return core.Column(rows...)
}

// swatch is one radio: a ring box holding the colour.
func (p ColorSwatchPicker) swatch(t *core.Theme, s Swatch, selected bool, onTap func()) core.View {
	ring := ColorTransparent
	if selected {
		ring = t.Colors.PrimaryOnLightColor()
	}
	fill := []core.PropsAndChildren{
		core.Width("100%"),
		core.Height(strconv.Itoa(swatchHeight) + "px"),
		core.Padding(0),
		core.BorderRadius(6),
		core.BackgroundColor(s.Hex),
		// A hairline, so a swatch the colour of the page still has an edge.
		core.BorderWidth(1),
		core.BorderColor(t.Colors.BorderColor()),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Justify(core.JustifyCenter),
		core.AccessibilityHidden(),
	}
	if selected {
		fill = append(fill, core.Text("✓",
			core.FontSize(18),
			core.FontWeight(core.Bold),
			core.TextColor(contrastInk(s.Hex, "#000000", "#FFFFFF")),
		))
	}

	items := []core.PropsAndChildren{
		// Equal shares: PINInput's note on why the basis is zero.
		core.FlexGrow(1),
		core.FlexBasis("0"),
		core.Padding(swatchPad),
		core.BorderRadius(10),
		core.BorderWidth(swatchRing),
		core.BorderColor(ring),
		core.AccessibilityRole(core.RoleRadio),
		core.AccessibilitySelected(core.SelectedWhen(selected)),
		core.AccessibilityLabel(s.Name),
		// Registered even when disabled, the contract core.Style.Disabled
		// states; choose() is the guard.
		core.OnClick(onTap),
	}
	if p.Disabled {
		items = append(items, core.Disabled(true))
	}
	// A Column, because the check is centred with the flex alignment props,
	// which a Box does not lay out on every target (RadioGroup's ring).
	items = append(items, core.Column(fill...))
	return core.Box(items...)
}

// customRow is the preview and the hex field.
func (p ColorSwatchPicker) customRow(t *core.Theme, draft, value string, custom bool, onChange func(string), onSubmit func()) core.View {
	// The preview shows the custom choice, or the hairline box of "none yet".
	preview := []core.PropsAndChildren{
		core.Width(strconv.Itoa(swatchHeight) + "px"),
		core.Height(strconv.Itoa(swatchHeight) + "px"),
		core.FlexShrink(0),
		core.Padding(0),
		core.BorderRadius(6),
		core.BorderWidth(1),
		core.BorderColor(t.Colors.BorderColor()),
		core.AccessibilityHidden(),
	}
	if custom {
		preview = append(preview, core.BackgroundColor(value))
	}

	input := []core.PropsAndChildren{
		core.FlexGrow(1),
		core.FlexBasis("0"),
		core.AccessibilityLabel(orDefault(p.CustomLabel, "Custom colour, hex")),
	}
	if p.Disabled {
		input = append(input, core.Disabled(true))
	}
	return core.Row(
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.Width("100%"),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Box(preview...),
		core.InputWithSubmit(draft, "#RRGGBB", onChange, onSubmit, input...),
	)
}

// hasSwatch reports whether hex (normalised) is one of the swatches.
func hasSwatch(swatches []Swatch, hex string) bool {
	for _, s := range swatches {
		if s.Hex == hex {
			return true
		}
	}
	return false
}

// normalizeHex reads "#rgb", "#rrggbb", or either without the #, and writes
// "#RRGGBB". Anything else, including a colour with an alpha byte, is refused:
// a picker of opaque swatches has no way to show what an alpha would mean.
func normalizeHex(s string) (string, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = strings.Repeat(s[0:1], 2) + strings.Repeat(s[1:2], 2) + strings.Repeat(s[2:3], 2)
	}
	if len(s) != 6 {
		return "", false
	}
	if _, err := strconv.ParseUint(s, 16, 32); err != nil {
		return "", false
	}
	return "#" + strings.ToUpper(s), true
}

// colorName guesses a spoken name for a normalised "#RRGGBB" from its hue,
// saturation and lightness: "blue", "dark green", "light grey".
//
// It is a fallback, and a coarse one on purpose. Eight hue bands is about
// what people agree on without a colour chart in front of them; a finer table
// would produce names ("chartreuse") that say less to most listeners than
// "green" does. The bands, in degrees of hue:
//
//	red <12 · orange <38 · yellow <70 · green <150 · teal <195
//	blue <240 · purple <290 · pink <345 · red
//
// Below 12% saturation a colour has no hue worth naming and is a grey; at the
// ends of the lightness range it is black or white whatever its hue.
func colorName(hex string) string {
	v, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil {
		return hex
	}
	r, g, b := float64(v>>16&0xFF)/255, float64(v>>8&0xFF)/255, float64(v&0xFF)/255
	hi, lo := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l := (hi + lo) / 2
	d := hi - lo

	sat := 0.0
	if d > 0 {
		sat = d / (1 - math.Abs(2*l-1))
	}
	switch {
	case l < 0.08:
		return "black"
	case l > 0.95:
		return "white"
	}

	name := "grey"
	if sat >= 0.12 {
		var hue float64
		switch hi {
		case r:
			hue = math.Mod((g-b)/d, 6)
		case g:
			hue = (b-r)/d + 2
		default:
			hue = (r-g)/d + 4
		}
		hue *= 60
		if hue < 0 {
			hue += 360
		}
		switch {
		case hue < 12:
			name = "red"
		case hue < 38:
			name = "orange"
		case hue < 70:
			name = "yellow"
		case hue < 150:
			name = "green"
		case hue < 195:
			name = "teal"
		case hue < 240:
			name = "blue"
		case hue < 290:
			name = "purple"
		case hue < 345:
			name = "pink"
		default:
			name = "red"
		}
	}
	switch {
	case l < 0.25:
		return "dark " + name
	case l > 0.8:
		return "light " + name
	}
	return name
}
