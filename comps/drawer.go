package comps

import "github.com/rohanthewiz/grmob/core"

// Drawer is side navigation: a panel of destinations pinned to the leading
// edge over the screen, opened by a ☰ button and closed by picking a
// destination, by its ✕, or by a tap on the scrim beside it.
//
//	open := core.NewState(ctx, false)
//	closeRef := core.UseFocusRef(ctx)
//	comps.Drawer{
//	    Open:      open.Get(),
//	    OnDismiss: func() { open.Set(false) },
//	    Title:     "Notebook",
//	    Items: []comps.DrawerItem{
//	        {Icon: "📥", Label: "Inbox",   OnTap: func() { section.Set(0) }},
//	        {Icon: "⭐", Label: "Starred", OnTap: func() { section.Set(1) }},
//	    },
//	    Selected: section.Get(),
//	    CloseRef: closeRef,
//	    Content: comps.Screen{Children: []core.View{
//	        comps.AppBar{Title: "Inbox", Leading: comps.Button{
//	            Label: "☰", AccessibilityLabel: "Open navigation",
//	            Emphasis: comps.EmphasisGhost,
//	            OnTap: func() { open.Set(true); core.Focus(closeRef) },
//	        }},
//	        body,
//	    }},
//	}
//
//	┌ ZStack  Width 100%, then Style ──────────────────────────────┐
//	│ ┌ Box 100% × 100%   AccessibilityHidden while Open ────────┐ │ layer 1:
//	│ │ Content                                                  │ │ the screen
//	│ └──────────────────────────────────────────────────────────┘ │
//	│ ┌ Row 100% × 100%   Display none while shut ───────────────┐ │ layer 2:
//	│ │ ┌ Column  Width 280px ──┐ ┌ Box FlexGrow(1) ───────────┐ │ │ drawn over
//	│ │ │ navigation, "Notebook"│ │ scrim: Backdrop fill,      │ │ │ layer 1
//	│ │ │ Notebook          [✕] │ │ tap = OnDismiss,           │ │ │
//	│ │ │ 📥 Inbox   (selected) │ │ hidden from assistive tech │ │ │
//	│ │ │ ⭐ Starred            │ │                            │ │ │
//	│ │ └───────────────────────┘ └────────────────────────────┘ │ │
//	│ └──────────────────────────────────────────────────────────┘ │
//	└──────────────────────────────────────────────────────────────┘
//
// # A layer over the content, not a Modal
//
// ActionSheet and Menu present through core.Modal, and a drawer could too. It
// does not, because a Modal's content lands wherever each host puts a dialog,
// and only the web puts it anywhere a leading edge could be reached from:
//
//	target    Modal presents as                  a leading panel would be
//	───────   ────────────────────────────────   ──────────────────────────
//	web ×2    fixed overlay, flex column         possible: a Row filling it
//	Compose   Dialog window at platform width,   a panel inside a centred
//	          inset from every screen edge       window, off the edge
//	SwiftUI   .sheet, medium/large detents       a bottom sheet
//
// A ZStack draws its layers in one box on all four targets (a single-cell grid
// on the web, a Box on Compose, a stack Layout on SwiftUI), and a layer sized
// Width/Height "100%" fills that box on each. So the panel is the stack's
// second layer, and it is a drawer at the leading edge everywhere, with no
// host change.
//
// What the Modal chassis would have supplied is the cost, and the widget buys
// back what it can:
//
//   - Screen-reader confinement. A Dialog window and a sheet confine TalkBack
//     and VoiceOver, and the DOM overlay says aria-modal. Here the content
//     layer takes AccessibilityHidden while the drawer is open, which takes
//     the screen behind out of the accessibility tree on every target, so
//     exploration stays in the panel.
//   - Focus. Nothing moves it on open, and the ☰ that had it is now inside a
//     hidden layer. CloseRef names the ✕ so the opener can call core.Focus on
//     it, as the example does; OnDismiss can hand focus back to the ☰ through
//     that button's own FocusRef. The widget holds no ref itself, because a
//     ref is a hook (see "No hooks").
//   - Keyboard containment on the web. aria-hidden does not stop Tab and core
//     has no inert, so Tab past the panel's last control still reaches the
//     hidden screen. Recorded, not fixed: it needs a renderer.
//   - The Android back button. A Dialog closes on it; a layer does not, so
//     the panel layer carries core.OnBack(OnDismiss) while open. The panel is
//     inside the screen and composed after it, so back closes the drawer
//     before a Navigator pops or an AppBar's back runs; the next back is
//     theirs. With OnDismiss nil there is nothing to call and back falls
//     through to them.
//
// # It covers its own box, so give it one
//
// The drawer covers the ZStack, not the window, and a ZStack is as big as its
// largest layer. Both layers ask for 100% of what the parent offers, so where
// the parent bounds the height (the app root, a Screen with Fill, a pinned
// Height) the drawer covers exactly that. In a scrolling column nothing bounds
// it: Compose's fill height is ignored under an unbounded constraint and the
// panel would sit centred at its own height. Pin a height through Style there,
// as the tutorial's demo does with core.Height.
//
// # The panel is hidden when shut, not removed
//
// The panel layer renders on every pass and takes Display none while shut,
// which both natives read as "do not compose" and the web as display:none. The
// tree is then the same shape open or shut, so opening is a style patch, and
// any hooks inside Body keep their slots: Body left out of a pass would shift
// every hook rendered after it, which is Spinner's rule. The content layer is
// always wrapped in its Box for the same reason. Toggling a prop is a patch;
// adding a wrapper around the screen would replace the screen.
//
// # Picking a destination closes the drawer
//
// A row calls its item's OnTap, then OnDismiss, as an ActionSheet action does
// and as Material's navigation drawer does. A caller would otherwise write
// open.Set(false) into every destination.
//
// The current destination is announced the way BottomBar's is: ListRow's
// ", selected" name suffix, because core has no current-page state. Selected
// also follows BottomBar and Tabs: the zero value selects the first item, and
// a negative Selected selects none.
//
// # No hooks
//
// Open and focus are both the caller's, so Drawer takes no hook slot and is
// conditional-safe, like Menu.
//
// # Theme roles read
//
//	Panel      Colors.Background
//	Title      Typography.Subtitle, bold, Colors.TextPrimary
//	Close      as comps.Button, ghost
//	Rows       as comps.ListRow, whose selected look marks the current one
//	Icon       Typography.Subtitle
//	Scrim      Backdrop, else core.Modal's default #00000088
type Drawer struct {
	// Open is the caller's open/shut state.
	Open bool

	// OnDismiss is called for the ✕, for a scrim tap, after any destination,
	// and for Android's system back while open. Nil draws no ✕, leaves the
	// scrim inert and claims no back, so the drawer closes only when the
	// caller flips Open.
	OnDismiss func()

	// Title names the panel: its heading and the navigation landmark's
	// accessible name. Empty draws no heading and leaves the landmark
	// unnamed; set it.
	Title string

	// Items are the destinations, top to bottom, each a full-width ListRow.
	Items []DrawerItem

	// Selected is the index of the current destination. Zero is the first
	// item; negative marks none.
	Selected int

	// Body replaces the Items rows with arbitrary content under the heading:
	// grouped destinations, an account row. Picking from it closes nothing
	// unless its own handlers call OnDismiss.
	Body core.View

	// Content is the screen the drawer is drawn over.
	Content core.View

	// CloseLabel is the ✕'s accessible name. Empty is "Close " + Title, or
	// "Close navigation" with no Title.
	CloseLabel string

	// CloseRef names the ✕ for core.Focus. It has no node to name when
	// OnDismiss is nil, since the ✕ is not drawn.
	CloseRef *core.FocusRef

	// Width is the panel's width. Empty is "280px", which leaves a tappable
	// scrim beside it on a 320-point phone. A percentage is honoured on all
	// four targets; core.MaxWidth is read by the web targets only, so cap a
	// percentage through PanelStyle for the browser alone.
	Width string

	// Backdrop overrides the scrim colour. Empty is core.Modal's default.
	Backdrop string

	// PanelStyle is applied to the panel column after the widget's props,
	// so it can repaint the panel or replace the landmark's name.
	PanelStyle []core.StyleProp

	// Style is applied to the ZStack after its Width, so it can pin the
	// height the drawer covers (see the type doc).
	Style []core.StyleProp
}

// DrawerItem is one destination in a Drawer.
type DrawerItem struct {
	// Icon is an optional glyph or emoji before the label. It is decoration
	// and hidden from assistive technology.
	Icon string

	// Label is the row's title and its accessible name.
	Label string

	// Subtitle is an optional quieter second line, such as a count.
	Subtitle string

	// OnTap is called before the drawer's OnDismiss.
	OnTap func()
}

// drawerDefaultBackdrop is core.Modal's default scrim, restated because a
// ZStack layer has no Modal to inherit it from. The two are one decision: a
// drawer's scrim and a dialog's should dim the screen by the same amount.
const drawerDefaultBackdrop = "#00000088"

// Render builds ZStack(content layer, panel layer) as drawn in the type doc.
func (d Drawer) Render(ctx *core.Context) *core.Node {
	stack := make([]core.PropsAndChildren, 0, len(d.Style)+3)
	// Width first so a caller's Style can replace it.
	stack = append(stack, core.Width("100%"))
	for _, sp := range d.Style {
		stack = append(stack, sp)
	}
	stack = append(stack, d.contentLayer(), d.panelLayer(ctx.Theme()))
	return core.ZStack(stack...).Render(ctx)
}

// contentLayer wraps Content in the Box that is hidden from assistive
// technology while the drawer is open. The Box is there open or shut; see
// "The panel is hidden when shut, not removed".
func (d Drawer) contentLayer() core.View {
	items := []core.PropsAndChildren{core.Width("100%"), core.Height("100%")}
	if d.Open {
		items = append(items, core.AccessibilityHidden())
	}
	// A nil Content is skipped by the container, leaving an empty layer.
	items = append(items, d.Content)
	return core.Box(items...)
}

// panelLayer builds Row(panel, scrim), shut with Display none.
func (d Drawer) panelLayer(t *core.Theme) core.View {
	items := []core.PropsAndChildren{
		// Padding and gap zeroed because a theme Row carries the screen
		// inset, which would pull the panel off the leading edge. Stretch so
		// the panel and scrim both run the layer's full height.
		core.Padding(0),
		core.Gap(0),
		core.AlignItemsProp(core.AlignItemsStretch),
		core.Width("100%"),
		core.Height("100%"),
	}
	if !d.Open {
		// Only the shut state writes a Display. Open writes none, so the
		// Row's own flex display comes back on the web's total style pass
		// rather than being overwritten by a block or inline keyword.
		items = append(items, core.Display(core.DisplayNone))
	} else if d.OnDismiss != nil {
		// System back, on the layer rather than the ✕ because comps.Button
		// takes style props only, and the layer is the node Display toggles.
		// Open only: a shut panel is not composed on Android anyway, but the
		// prop left on would register a callback every pass for a drawer
		// nobody can see, and any other reader of the tree would see back
		// claimed by a closed drawer.
		items = append(items, core.OnBack(d.OnDismiss))
	}
	items = append(items, d.panel(t), d.scrim())
	return core.Row(items...)
}

// panel builds the navigation column: the heading row, then Body or the rows.
func (d Drawer) panel(t *core.Theme) core.View {
	width := orDefault(d.Width, "280px")
	items := make([]core.PropsAndChildren, 0, len(d.PanelStyle)+len(d.Items)+8)
	items = append(items,
		core.Width(width),
		// Height as well as the Row's stretch: SwiftUI's HStack does not
		// stretch a child that has not asked to fill.
		core.Height("100%"),
		core.BackgroundColor(t.Colors.Background),
		core.Padding(t.Spacing.SM),
		core.Gap(float64(t.Spacing.XS)),
		core.AccessibilityRole(core.RoleNavigation),
	)
	if d.Title != "" {
		items = append(items, core.AccessibilityLabel(d.Title))
	}
	for _, sp := range d.PanelStyle {
		items = append(items, sp)
	}
	items = append(items, d.header(t))
	if d.Body != nil {
		items = append(items, d.Body)
	} else {
		for i, it := range d.Items {
			items = append(items, d.row(t, it, i == d.Selected))
		}
	}
	return core.Column(items...)
}

// header builds the Title and the ✕, or nil when there is neither.
func (d Drawer) header(t *core.Theme) core.View {
	if d.Title == "" && d.OnDismiss == nil {
		return nil
	}
	items := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
	}
	// The title box grows even when empty, so the ✕ stays on the trailing
	// edge of the panel in both cases.
	title := []core.PropsAndChildren{core.FlexGrow(1), core.Padding(0)}
	if d.Title != "" {
		title = append(title, core.Text(d.Title,
			core.UseStyle(t.Typography.Subtitle),
			core.FontWeight(core.Bold),
			// Primary ink stated, because the Subtitle scale can carry a
			// secondary tone and the panel's name should not read as a
			// caption under the rows it names.
			core.TextColor(t.Colors.TextPrimary),
		))
	}
	items = append(items, core.Box(title...))
	if d.OnDismiss != nil {
		name := d.CloseLabel
		if name == "" {
			name = "Close navigation"
			if d.Title != "" {
				name = "Close " + d.Title
			}
		}
		items = append(items, Button{
			Label:              "✕",
			AccessibilityLabel: name,
			OnTap:              d.OnDismiss,
			Emphasis:           EmphasisGhost,
			FocusRef:           d.CloseRef,
		})
	}
	return core.Row(items...)
}

// row builds one destination as a ListRow whose tap picks and then dismisses.
func (d Drawer) row(t *core.Theme, it DrawerItem, current bool) core.View {
	var leading core.View
	if it.Icon != "" {
		leading = core.Text(it.Icon,
			core.UseStyle(t.Typography.Subtitle),
			core.AccessibilityHidden(),
		)
	}
	return ListRow{
		Leading:  leading,
		Title:    it.Label,
		Subtitle: it.Subtitle,
		OnTap:    d.pick(it),
		Selected: current,
		// ListRow appends its ", selected" suffix to an explicit name only,
		// so the label is restated here to get the current destination
		// announced. The subtitle stays out of the name, as on any ListRow
		// whose name is set: a count that changes would change the name.
		AccessibilityLabel: it.Label,
	}
}

// pick returns the handler for one destination: its own OnTap, then the
// drawer's OnDismiss. Taken per item so each closure holds its own item.
func (d Drawer) pick(it DrawerItem) func() {
	return func() {
		if it.OnTap != nil {
			it.OnTap()
		}
		if d.OnDismiss != nil {
			d.OnDismiss()
		}
	}
}

// scrim is the growing, dimmed area beside the panel. A tap on it dismisses;
// it is hidden from assistive technology because the ✕ is the accessible way
// out, as it is for ActionSheet's filler.
func (d Drawer) scrim() core.View {
	items := []core.PropsAndChildren{
		core.FlexGrow(1),
		core.Height("100%"),
		core.BackgroundColor(orDefault(d.Backdrop, drawerDefaultBackdrop)),
		core.AccessibilityHidden(),
	}
	// Guarded like ActionSheet's filler: a registered no-op would make an
	// inert scrim look tappable to a host.
	if d.OnDismiss != nil {
		items = append(items, core.OnClick(d.OnDismiss))
	}
	return core.Box(items...)
}
