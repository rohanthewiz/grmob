package comps

import "github.com/rohanthewiz/grmob/core"

// RadioGroup is a vertical set of mutually exclusive options, each a row with
// a ring on the leading edge: shipping speed, a plan, a theme.
//
//	comps.RadioGroup{
//	    Label: "Shipping",
//	    Options: []comps.RadioOption{
//	        {Value: "std", Label: "Standard", Subtitle: "3–5 days"},
//	        {Value: "exp", Label: "Express", Subtitle: "Next day"},
//	    },
//	    Value:    ship.Get(),
//	    OnChange: ship.Set,
//	}
//
// # Where it sits among the choice widgets
//
//	core.Select         compact; the options are hidden until opened
//	SegmentedControl    horizontal, short labels, two to four options
//	RadioGroup          vertical, every option visible, room for a subtitle
//
// # The role decision: listbox and option, not radiogroup and radio
//
// core.Role has no RoleRadio or RoleRadioGroup. The plan weighed two answers:
// add the pair to core (core/role.go, both web exporters' role tables and
// the trait mapping on both natives), or use RoleListBox and RoleOption with
// AccessibilitySelected, which every target already announces. This ships
// the second, through ListRow.Selectable, and records the first as a
// follow-up. What the choice costs is the word: a reader says "option, 2 of
// 3, selected" where a radio group would say "radio button, checked". What it
// keeps is everything that matters for operating it: one choice is announced
// as selected and the rest as not selected (SelectedWhen, never unset), and
// the WASM runtime's listbox keyboard (one tab stop, Up/Down, Home/End,
// Enter or Space to choose) arrives with the container's role.
//
// # A closed composite: do not put it inside another
//
// RadioGroup declares its own listbox, which makes it one of the two widgets
// in this package that declare a keyboard container role (BottomBar's toolbar
// is the other). It holds no core.View, so nothing can be nested inside it.
// Placing it inside a container you roled as a listbox, tablist or toolbar
// yourself is the one way to nest it, and core.AuditTree reports that as a
// nested composite in debug mode. comps/nested_composite_test.go holds the
// "closed" half of this.
//
// # The whole row is the target, and there is no second control
//
// Unlike SwitchRow, the ring is drawn, not a platform control: a Column with a
// border and, when selected, a filled dot, hidden from assistive technology.
// The row's OnTap is therefore the only handler, and the web's double
// dispatch that SwitchRow guards against cannot happen here. Tapping the
// selected option does nothing; OnChange fires only for a change, the rule
// Stepper and Rating follow.
//
// The selected row takes no background tint. ListRow's Surface tint is its
// only selection cue; here the ring already says it, and a tinted row on top
// reads as a second, different state.
//
// # Theme roles read
//
//	Ring, selected   Colors.PrimaryOnLightColor() — border and dot
//	Ring, other      Colors.ControlBorder (Colors.BorderColor() if unset):
//	                 the ring is a control boundary, so it takes the stroke
//	                 role that clears 3:1
//	Ring, disabled   Colors.TextSecondary
//	Row text         everything ListRow reads
type RadioGroup struct {
	// Options are drawn top to bottom.
	Options []RadioOption

	// Value is the selected option's Value. A Value matching no option draws
	// every ring empty.
	Value string

	// OnChange receives the tapped option's Value, only when it differs from
	// Value. Nil draws a display-only group with no handlers.
	OnChange func(string)

	// Label is the group's accessible name. A listbox with no name is
	// announced as a bare list of options, so set it unless a visible heading
	// directly above names the group.
	Label string

	// Disabled disables every option.
	Disabled bool

	// Style is applied to the group column after the widget's own props.
	Style []core.StyleProp
}

// RadioOption is one choice in a RadioGroup.
type RadioOption struct {
	// Value identifies the option to OnChange and is compared with
	// RadioGroup.Value.
	Value string

	// Label is the row's title and its accessible name.
	Label string

	// Subtitle is the quieter second line and the row's accessibility hint.
	Subtitle string

	// Disabled greys this option and drops its taps.
	Disabled bool
}

// ringSize is the ring's diameter in points. 20 matches the platform radio
// controls on both natives closely enough that a RadioGroup beside a
// core.Checkbox does not look like a different family.
const ringSize = 20

// Render builds Column(listbox) > ListRow(option)... as described in the type
// doc.
func (g RadioGroup) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(g.Style)+len(g.Options)+4)
	items = append(items,
		// Padding(0) and no gap: each row already carries the theme's row
		// padding, and a gap between rows would leave dead strips between tap
		// targets.
		core.Padding(0),
		core.Gap(0),
		core.AccessibilityRole(core.RoleListBox),
	)
	if g.Label != "" {
		items = append(items, core.AccessibilityLabel(g.Label))
	}
	for _, sp := range g.Style {
		items = append(items, sp)
	}
	for _, opt := range g.Options {
		items = append(items, g.row(t, opt))
	}
	return core.Column(items...).Render(ctx)
}

// row builds one option. A disabled row keeps its handler registered and
// carries core.Disabled, the contract core.Style.Disabled documents (a native
// tap racing the disabling patch must find a handler); the guard inside the
// handler is what keeps OnChange from firing on a target that dispatches
// anyway.
func (g RadioGroup) row(t *core.Theme, opt RadioOption) ListRow {
	selected := opt.Value == g.Value
	disabled := g.Disabled || opt.Disabled

	var style []core.StyleProp
	if disabled {
		style = []core.StyleProp{core.Disabled(true)}
	}

	var onTap func()
	if g.OnChange != nil {
		onTap = func() {
			if disabled || selected {
				return
			}
			g.OnChange(opt.Value)
		}
	}

	return ListRow{
		Leading:            ring(t, selected, disabled),
		Title:              opt.Label,
		Subtitle:           opt.Subtitle,
		OnTap:              onTap,
		Selected:           selected,
		Selectable:         true,
		SelectedStyle:      []core.StyleProp{}, // non-nil: no Surface tint; see the type doc
		Style:              style,
		AccessibilityLabel: opt.Label,
		AccessibilityHint:  opt.Subtitle,
	}
}

// ring draws the radio glyph: a bordered circle, with a centred dot when
// selected. A Column rather than a Box because the dot is centred with the
// flex alignment props, which a Box does not lay out on every target.
func ring(t *core.Theme, selected, disabled bool) core.View {
	stroke := t.Colors.ControlBorder
	if stroke == "" {
		stroke = t.Colors.BorderColor()
	}
	if selected {
		stroke = t.Colors.PrimaryOnLightColor()
	}
	if disabled {
		stroke = t.Colors.TextSecondary
	}

	items := []core.PropsAndChildren{
		core.Width("20px"),
		core.Height("20px"),
		core.FlexShrink(0),
		core.Padding(0),
		core.Gap(0),
		core.BorderRadius(ringSize / 2),
		core.BorderWidth(2),
		core.BorderColor(stroke),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Justify(core.JustifyCenter),
		core.AccessibilityHidden(),
	}
	if selected {
		items = append(items, core.Box(
			core.Width("10px"),
			core.Height("10px"),
			core.BorderRadius(5),
			core.BackgroundColor(stroke),
		))
	}
	return core.Column(items...)
}
