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
// name.
//
// # Why this row still spells its state into the name
//
// Chip and Calendar used to do the same and no longer do: core.Style has
// core.AccessibilitySelected now, which every renderer announces as a control
// being on. This row deliberately did not follow them, and the reason is that
// a row is not a control.
//
// The state is scoped by role on both web targets, because ARIA scopes the
// attributes it becomes: a state on an unroled element is dropped by screen
// readers exactly as an accessible name on one is — the failure core.RoleImg
// exists to close. A ListRow is a Box. So adopting the field here means
// giving every row a role, and both candidates are wrong:
//
//	RoleButton    true only for a tappable row, and a role="button" child
//	              makes the row a *foreign child* of any role="list" it sits
//	              in — the structural rule in core/role.go, which says such a
//	              container may then take no list role at all. One widget's
//	              announcement would cost the enclosing list its shape.
//	RoleListItem  the honest description of a row, and ARIA defines neither
//	              state attribute for it. A selectable item in a collection is
//	              an `option` inside a `listbox`, and core.Role carries
//	              neither — nor the roving focus a listbox promises.
//
// So the suffix stays until the vocabulary has the pair that fits. It still
// says the true thing; it says it in the weaker of the two places.
//
// # Depth, which the same table says yes to
//
// NestingLevel is the other half of that table, and it lands where the
// selection could not. A depth's role is `listitem` — one of the three ARIA
// defines aria-level for — and `listitem` is the row's honest description; the
// selection's role is `option`, which core.Role does not carry. Two fields, one
// widget, opposite answers, and the reason is a property of the roles rather
// than of this widget's willingness.
//
// It is opt-in, and that is the ownership rule rather than caution: a
// `listitem` with no `list` around it is a role naming a structure that is not
// there, which core/role.go calls worse than no role at all. A row cannot see
// its container, so it cannot make that true by itself — the caller sets
// RoleList on the enclosing list and the depth on each row, and a row that is
// asked for neither is exactly the unroled Box it has always been.
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

	// NestingLevel is how deep this row sits in a nested collection — 1 for a
	// top-level item, 2 for one inside it, and on down with no ceiling. It
	// makes the row a `listitem` at that depth; zero leaves it the unroled Box
	// it has always been.
	//
	// # What it is for
	//
	// A tree flattened into one list. That is the case ARIA defines aria-level
	// on `listitem` for: the rows are siblings in the markup because a list is
	// a flat run of children — which is also what core.List's virtualization
	// requires — so the depth a reader needs has nowhere else to live. Without
	// it an outline is announced as a flat run of items and every indent is
	// pixels only.
	//
	//	core.List(
	//	    core.AccessibilityRole(core.RoleList),
	//	    ListRow{Title: "Gospels",  NestingLevel: 1},
	//	    ListRow{Title: "Matthew",  NestingLevel: 2, Style: indent(1)},
	//	    ListRow{Title: "Sermon on the Mount", NestingLevel: 3, Style: indent(2)},
	//	)
	//
	// # The container is the caller's to role, and must be
	//
	// A `listitem` is owned by a `list` (see "A structural role owns what is
	// inside it" in core/role.go). This widget renders one row and cannot see
	// what it was put in, so it cannot supply the other half — set RoleList on
	// the container yourself, or leave this field at zero. An orphan
	// `listitem` is the "table with no rows" failure one row down.
	//
	// The corollary is worth stating because it costs something: a row inside
	// a role="list" must not also be a role="button", so a tappable row in an
	// outline announces as an item at a depth and not as a control. That is
	// the same foreign-child rule that kept the selected state off this widget,
	// read from the other side — and it is not a regression, because a ListRow
	// has never carried RoleButton.
	//
	// # What each target does with it
	//
	// The web writes aria-level. Neither native has a nesting-depth property
	// at all, so the role goes out and the depth does not, which is the honest
	// gap nine of core's twenty roles already have.
	NestingLevel int

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

	// Before the label, so the two arrive on the node in the order they are
	// read: what this is, then what it is called. Both are style props and
	// order does not affect the result, but a reader of this function should
	// meet the role first for the same reason a screen reader does.
	//
	// The pair travels together — a depth with no role is dropped by every
	// target that reads it, since ARIA scopes aria-level to three roles and
	// both web exporters switch on exactly those.
	if r.NestingLevel != 0 {
		items = append(items,
			core.AccessibilityRole(core.RoleListItem),
			core.AccessibilityNestingLevel(r.NestingLevel),
		)
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
