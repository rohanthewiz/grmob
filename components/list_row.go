package components

import "github.com/rohanthewiz/grmob/core"

// ListRow is the leading-control / flexible-title / trailing-action shape that
// every list in the examples hand-rolls: a checkbox and a task with a delete
// button, an avatar and a name with a chevron, a label and an amount.
//
// # Why the widget exists
//
// The shape was written five times across the examples and the instances did
// not agree on how the trailing slot gets pinned to the trailing edge. Some
// used Justify(JustifyBetween) on the row; others used FlexGrow(1) on the
// middle Text. The two are not equivalent:
//
//	JustifyBetween  distributes slack *between every pair* of children, so a
//	                row with no trailing slot pushes leading and title apart.
//	FlexGrow(1)     gives all the slack to one child, so leading and trailing
//	                stay hard against the row's edges in every configuration.
//
// ListRow settles it on FlexGrow: the middle column is the row's spine and it
// always grows. That is also why the middle column is rendered even when
// Title, Subtitle and Content are all empty — unlike Card, which omits empty
// regions. Here the middle is structure, not content: making it conditional
// would make the pinning conditional too, which is precisely the
// inconsistency this widget exists to remove.
//
//	┌ Row ─────────────────────────────────────────────────────────┐
//	│ [Leading]  ┌ Column FlexGrow(1) ────────────┐    [Trailing]  │
//	│            │ Title                          │                │
//	│            │ Subtitle                       │                │
//	│            └────────────────────────────────┘                │
//	└──────────────────────────────────────────────────────────────┘
//	            └──────── takes all the slack ────┘
//
// # Selection
//
// Selection is controlled by the caller, as with Chip: ListRow holds no
// state, it renders Selected and reports taps. Selected rows take the theme's
// Surface as a background tint — the palette's only muted *fill*, and still
// the right one now that Border exists, since Border is a stroke role; there
// is no dedicated Selected entry — and, when an AccessibilityLabel
// is set, get ", selected" appended so the state is announced along with the
// name. That suffix convention is the same one Chip owns internally.
type ListRow struct {
	// Leading is the control at the start of the row: a checkbox, an icon,
	// an avatar. Nil renders nothing and costs no node.
	Leading core.View

	// Title is the row's primary line, Subtitle the quieter second line.
	Title    string
	Subtitle string
	// Content is the escape hatch for the middle: an arbitrary view in the
	// growing slot, taking precedence over Title/Subtitle when set. Same
	// simple-path-plus-slot idiom as Card.Title vs Card.Header.
	Content core.View

	// Trailing is the action or value pinned to the end of the row: a badge,
	// an amount, a delete button, a chevron.
	Trailing core.View

	// OnTap and OnLongPress make the whole row a target. They are wired only
	// when non-nil, so a purely presentational row carries no callback and
	// no gesture recognizer on any platform. Both may be set at once: the
	// renderers wire them as a single recognizer, so a long press never also
	// fires the tap.
	OnTap       func()
	OnLongPress func()

	// Selected drives the row's selected look and its accessibility suffix.
	Selected bool

	// Style is applied to the row container after ListRow's own defaults
	// (which sit on top of the theme's Row base), so every default here —
	// gap, vertical centering, the theme's row padding — is overridable.
	Style []core.StyleProp
	// SelectedStyle is applied on top when Selected, after Style, so
	// selection wins over the base look. When nil, a theme default is used:
	// a Surface background tint.
	SelectedStyle []core.StyleProp

	// AccessibilityLabel names the whole row for screen readers; when
	// Selected, ", selected" is appended. AccessibilityHint describes what
	// tapping does.
	//
	// No label is synthesized from Title: a row is a compound control whose
	// slots (a badge's amount, a trailing control's own name) carry meaning
	// the widget cannot see, and labelling the container overrides how those
	// children are announced. Naming the row is therefore the caller's call,
	// exactly as it is for Chip.
	AccessibilityLabel string
	AccessibilityHint  string
}

func (r ListRow) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(r.Style)+len(r.SelectedStyle)+9)

	// ListRow's own defaults first, so the caller's Style can override any of
	// them. The theme's Row base already supplies the padding; what a row of
	// mixed-height controls additionally needs is a gap between slots and a
	// shared centre line — a checkbox and a two-line title otherwise sit on
	// their tops.
	items = append(items,
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
	)

	for _, sp := range r.Style {
		items = append(items, sp)
	}

	if r.Selected {
		sel := r.SelectedStyle
		if sel == nil {
			sel = []core.StyleProp{core.BackgroundColor(t.Colors.Surface)}
		}
		for _, sp := range sel {
			items = append(items, sp)
		}
	}

	if r.AccessibilityLabel != "" {
		label := r.AccessibilityLabel
		if r.Selected {
			label += ", selected"
		}
		items = append(items, core.AccessibilityLabel(label))
	}
	if r.AccessibilityHint != "" {
		items = append(items, core.AccessibilityHint(r.AccessibilityHint))
	}

	// Guarded so a presentational row registers no callback at all: an
	// unconditional OnClick would put a live handler ID in the props of
	// every row in a list, and every one of them would survive the render
	// pass's callback sweep for nothing.
	if r.OnTap != nil {
		items = append(items, core.OnClick(r.OnTap))
	}
	if r.OnLongPress != nil {
		items = append(items, core.OnLongPress(r.OnLongPress))
	}

	if r.Leading != nil {
		items = append(items, r.Leading)
	}
	items = append(items, r.middle(t))
	if r.Trailing != nil {
		items = append(items, r.Trailing)
	}

	return core.Row(items...).Render(ctx)
}

// middle builds the growing centre column — the slot that takes the row's
// slack and thereby pins Trailing to the trailing edge (see the type comment).
func (r ListRow) middle(t *core.Theme) core.View {
	items := make([]core.PropsAndChildren, 0, 5)

	// Padding(0) assigns rather than merges, so this drops the theme
	// Column's screen-level padding: the title stack must read as one block
	// inside the row, not as a nested panel. Rows of the stack are separated
	// by the finest spacing step, the same tight-stack recipe FormField uses.
	items = append(items,
		core.FlexGrow(1),
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
	)

	if r.Content != nil {
		items = append(items, r.Content)
		return core.Column(items...)
	}

	if r.Title != "" {
		items = append(items, core.Text(r.Title, core.UseStyle(t.Typography.Body)))
	}
	if r.Subtitle != "" {
		// Caption: smaller and in secondary ink, so the second line reads as
		// support for the title rather than a competing one.
		items = append(items, core.Text(r.Subtitle, core.UseStyle(t.Typography.Caption)))
	}
	return core.Column(items...)
}
