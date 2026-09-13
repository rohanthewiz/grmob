package comps

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
// is no dedicated Selected entry. How the state reaches a screen reader is the
// next section's subject and depends on Selectable.
//
// # How the state is announced, and why it took a role to do it
//
// The state is scoped by role on both web targets, because ARIA scopes the
// attributes it becomes: a state on an unroled element is dropped by screen
// readers exactly as an accessible name on one is. The *name* half of that has
// since been closed for every widget at once — the two web exporters supply
// core.RoleGroup to a named container that has no role, so an unroled row's
// AccessibilityLabel is now announced on all four targets rather than two. The
// state half could not be closed the same way, because there is no role that
// carries a selection and fits any container (see core.Style.AccessibilitySelected).
// A ListRow is a Box, so announcing a selection properly means giving the row a
// role, and for three versions of this widget every candidate was wrong:
//
//	RoleButton    true only for a tappable row, and a role="button" child
//	              makes the row a *foreign child* of any role="list" it sits
//	              in — the structural rule in core/role.go, which says such a
//	              container may then take no list role at all. One widget's
//	              announcement would cost the enclosing list its shape.
//	RoleListItem  the honest description of a row, and ARIA defines neither
//	              state attribute for it. `aria-selected` is scoped to
//	              gridcell, option, row, tab, columnheader and rowheader, and
//	              a list item is none of them.
//
// So the row wrote ", selected" into its own accessible name instead — the
// true thing said in the weaker of the two places, announced once, inside a
// string that is meant to be stable.
//
// core.RoleOption and core.RoleListBox are the door that was left, and
// Selectable is how a caller walks through it. A row that takes the option
// role carries a real core.AccessibilitySelected, every renderer announces it
// as a state rather than as part of a name, and the suffix does not appear.
//
// A row that is *not* Selectable is unchanged: it still appends the suffix
// when it has both a label and a selection, because it still has no role that
// could carry the state, and saying the true thing weakly beats not saying it.
// The suffix does at least reach a reader now — the group role the exporters
// supply is what makes the name it rides on audible on the web at all, which
// for the three sessions before that role existed it was not.
//
// # Two roles, one row, and the caller picks
//
// NestingLevel and Selectable both give the row a role and the two roles are
// exclusive — see Selectable for the precedence, which is a fact about the
// roles rather than about this widget's willingness.
//
// Both are opt-in, and that is the ownership rule rather than caution: a
// `listitem` with no `list` around it, or an `option` with no `listbox`, is a
// role naming a structure that is not there, which core/role.go calls worse
// than no role at all. A row cannot see its container, so it cannot make that
// true by itself — the caller roles the enclosing collection and marks each
// row to match, and a row that is asked for neither is exactly the unroled Box
// it has always been.
type ListRow struct {
	// Leading whose width is text is worth core.FlexShrink(0), and this is
	// the one trap in the slot.
	//
	// The centre column claims the row's slack, so when the row's content
	// overflows — a long title on a phone — the deficit is shared out among
	// the children that can shrink, and a bare core.Text is the most
	// compressible thing in the row. Every host floors that share at the
	// text's min-content width, so a word is never ground down to a glyph:
	// the web and Compose always did, and iOS does since GrMobMinContent,
	// which a simulator run of examples/tutorial forced after it rendered the
	// lesson numbers as 4 / . / 1 / 2.
	//
	// What the floor does NOT promise is that the text stays on one line. A
	// row number has no break opportunity in it, so its min-content is the
	// whole of it; a two-word label's is its longer word, and a row tight
	// enough will wrap it. FlexShrink(0) is the declaration that says the
	// slot's width is not negotiable at all, and a leading slot whose width
	// is meant to be read at a glance wants it on every host.
	//
	// A fixed-size control — a Checkbox, an icon with a Width — is unaffected,
	// which is why this is a note on the field rather than a wrapper around it:
	// ListRow cannot add a style prop to a View a caller handed it, and
	// wrapping every slot in a pinned Box would be two extra nodes per row of
	// every list to fix the case where the caller passes text.
	//
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

	// Selected drives the row's selected look, and how the state is announced
	// — as a real core.AccessibilitySelected when Selectable is set, and
	// otherwise as a ", selected" suffix on AccessibilityLabel. See "How the
	// state is announced" in the type comment for why there are two answers.
	Selected bool

	// Selectable says this row is one choice in a listbox: it takes
	// core.RoleOption and states core.AccessibilitySelected for *both* values
	// of Selected, so a reader announces "selected" and "not selected" rather
	// than announcing the chosen row and passing silently over the rest. That
	// is core.SelectedOff doing the job it exists for, one widget over from
	// the tab strip whose argument it was written for.
	//
	// # The container is the caller's to role, and must be
	//
	// An `option` is owned by a `listbox` (see "A structural role owns what is
	// inside it" in core/role.go). This widget renders one row and cannot see
	// what it was put in, so set core.RoleListBox on the container yourself,
	// or leave this field false. An orphan `option` is the "table with no
	// rows" failure one row down.
	//
	//	core.List(
	//	    core.AccessibilityRole(core.RoleListBox),
	//	    ListRow{Title: "Weekly",  Selectable: true, Selected: plan == weekly,
	//	            AccessibilityLabel: "Weekly", OnTap: choose(weekly)},
	//	    ListRow{Title: "Monthly", Selectable: true, Selected: plan == monthly,
	//	            AccessibilityLabel: "Monthly", OnTap: choose(monthly)},
	//	)
	//
	// The same foreign-child rule applies as ever: a listbox holding a
	// "Load more" footer or a section heading is not a listbox.
	//
	// # It wins over NestingLevel, and the depth is lost
	//
	// The two fields ask for different roles and a node has one. `option`
	// takes aria-selected and not aria-level; `listitem` takes aria-level and
	// not aria-selected. So a row setting both describes a container that is a
	// list and a listbox at once, which does not exist, and this field is the
	// half that wins: the state is what the row is being tapped to change,
	// and a depth inside a container that has not claimed to be a list is
	// decoration.
	//
	// ARIA does have a role carrying both — `treeitem` inside a `tree` — and
	// core.Role deliberately does not, because a tree is a third pattern with
	// its own expansion state and keyboard contract and nothing here has one.
	// See the RoleListBox block in core/role.go.
	//
	// # What each target does with it
	//
	// The two web targets write role="option" and aria-selected. Neither
	// native names a listbox or an option, but both announce the *state* on
	// any node — a SwiftUI .isSelected trait, a Compose `selected` property —
	// so the row still reads as chosen on device and it is only the
	// container's word that is missing. Setting this therefore adds on every
	// target and costs nothing on any.
	//
	// # What it buys on the keyboard, and what the caller has to do for it
	//
	// A listbox in ARIA's full pattern takes focus, moves an active option
	// with the arrow keys and reports which one through a roving tabindex.
	// The WASM runtime supplies all of it — one tab stop per list, Up/Down
	// within it, Home and End, and Enter or Space running this row's OnTap —
	// but only for rows inside a container that says it is a listbox. That is
	// the same core.RoleListBox the field above already asks the caller for,
	// so the keyboard arrives with the container's role and not with this
	// flag: a Selectable row in an unroled Box is an `option` with nothing to
	// be an option of, and gets no more keyboard than it did before.
	//
	// A static htmlout export writes no tab stops at all, deliberately — see
	// core.RoleListBox. On both phones none of this was ever missing:
	// VoiceOver and TalkBack navigate a collection by swipe.
	Selectable bool

	// NestingLevel is how deep this row sits in a nested collection — 1 for a
	// top-level item, 2 for one inside it, and on down with no ceiling. It
	// makes the row a `listitem` at that depth; zero leaves it the unroled Box
	// it has always been. Ignored when Selectable is set, which takes the
	// row's one role for `option` — see that field.
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
	// gap several of core's roles already have (see
	// core.Style's AccessibilityRole table for which).
	NestingLevel int

	// Style is applied to the row container after ListRow's own defaults
	// (which sit on top of the theme's Row base), so every default here —
	// gap, vertical centering, the theme's row padding — is overridable.
	Style []core.StyleProp
	// SelectedStyle is applied on top when Selected, after Style, so
	// selection wins over the base look. When nil, a theme default is used:
	// a Surface background tint.
	SelectedStyle []core.StyleProp

	// AccessibilityLabel names the whole row for screen readers.
	// AccessibilityHint describes what tapping does.
	//
	// A Selected row with no Selectable gets ", selected" appended, because
	// the name is then the only place the state can be said; a Selectable row
	// leaves the name alone and states the selection properly. See "How the
	// state is announced" in the type comment.
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

	// The row's one role, and whatever second prop that role can carry. Before
	// the label, so the props arrive on the node in the order they are read:
	// what this is, then what it is called. Order does not affect the result —
	// they are all style props — but a reader of this function should meet the
	// role first for the same reason a screen reader does.
	//
	// Each pair travels together, because half of one is inert. A depth with
	// no role is dropped by every target that reads it (ARIA scopes aria-level
	// to three roles and both web exporters switch on exactly those), and a
	// state with no role is dropped for the same reason one attribute over.
	//
	// The branch is an either/or rather than two ifs because a node has one
	// role and the two candidates are exclusive; see the Selectable field for
	// which wins and why the loser is the depth.
	switch {
	case r.Selectable:
		items = append(items,
			core.AccessibilityRole(core.RoleOption),
			// SelectedWhen, not "set it only when on": an option that stays
			// quiet while its neighbour says "selected" is announced as
			// something that cannot be chosen at all.
			core.AccessibilitySelected(core.SelectedWhen(r.Selected)),
		)
	case r.NestingLevel != 0:
		items = append(items,
			core.AccessibilityRole(core.RoleListItem),
			core.AccessibilityNestingLevel(r.NestingLevel),
		)
	}

	if r.AccessibilityLabel != "" {
		label := r.AccessibilityLabel
		// The fallback, and only the fallback. A Selectable row has a real
		// state on it, so appending here would announce the selection twice —
		// once as part of the row's name and once as the control state — and
		// would put a changing word inside a name that is meant to be stable.
		if r.Selected && !r.Selectable {
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
