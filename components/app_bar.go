package components

import "github.com/rohanthewiz/grmob/core"

// AppBar is the title strip at the top of a screen: an optional back
// affordance, the screen's name, and trailing actions.
//
//	components.AppBar{Title: "Sermons"}
//	components.AppBar{Title: "Sermon", Actions: []core.View{shareButton}}
//
//	┌ Box ─────────────────────────────────────────────────────────┐
//	│ ┌ Row ───────────────────────────────────────────────────────┤
//	│ │ [‹]  ┌ Box FlexGrow(1) ──────────┐  [Action] [Action]      │
//	│ │      │ Title                     │                         │
//	│ │      │ Subtitle                  │                         │
//	│ │      └───────────────────────────┘                         │
//	│ ├────────────────────────────────────────────────────────────┤
//	│ │ Separator                                                  │
//	└─┴────────────────────────────────────────────────────────────┘
//	       └──── takes all the slack, pinning Actions right ────┘
//
// # It is a Row, not a platform navigation bar
//
// grmob has no AppBar node, so there is nothing here that floats over the
// content, collapses on scroll, claims the status bar, or animates a title
// between screens. This is an ordinary row of ordinary widgets that happens
// to sit first in a screen's column — which is what makes it identical on all
// four targets, and what makes components.Screen's SafeArea, not the bar,
// responsible for keeping it clear of the notch.
//
// # The back control appears only when there is somewhere to go
//
// With no Leading and no HideBack, the bar draws a back button exactly when
// core.CanPop says the navigation stack has a screen underneath. The root of
// a tab shell therefore gets no arrow without the caller having to say so,
// and a pushed detail screen gets one without the caller having to wire it —
// which is the behavior every navigation framework has, reached here through
// the one question core exposes. See core.CanPop for why asking beats
// rendering a control that would no-op.
//
// # Slots beat fields, as everywhere in this package
//
// Content overrides Title/Subtitle and Leading overrides the back control,
// the same simple-path-plus-slot idiom as Card.Title vs Card.Header. Setting
// Leading is also how you get a back control the widget would not have drawn
// — a close button on a modally presented screen, where CanPop is false.
type AppBar struct {
	// Title is the screen's name. Subtitle is a quieter second line under it
	// — a count, a date, a connection state.
	Title    string
	Subtitle string

	// Content is the escape hatch for the growing middle: an arbitrary view
	// in place of the Title/Subtitle stack. Takes precedence when set.
	//
	// The middle is rendered either way, empty or not: it is the flexible
	// slot that pins Actions to the trailing edge, so making it conditional
	// would make the pinning conditional. Same rule, and the same reason, as
	// ListRow's middle column.
	Content core.View

	// Leading replaces the automatic back control entirely. A nil Leading
	// with HideBack unset draws the back button when core.CanPop is true.
	Leading core.View

	// HideBack suppresses the automatic back control on a screen that can pop
	// but should not offer it — a wizard step that must be completed or
	// abandoned through its own buttons.
	HideBack bool

	// OnBack replaces core.Pop as what the automatic back control does. Set
	// it to confirm before leaving, or to pop more than one frame; call
	// core.Pop yourself from inside it when the answer is yes.
	OnBack func()

	// BackGlyph is the back control's label. Empty is "‹". It is the one
	// piece of the automatic control worth a field of its own — swapping "‹"
	// for "←" or "Back" otherwise costs the caller the whole Leading slot,
	// CanPop test and Pop wiring included.
	BackGlyph string

	// Actions are the trailing controls, in leading-to-trailing order. Nil
	// entries are skipped, so a conditional action can be a nil variable
	// rather than a filtered slice.
	Actions []core.View

	// HideSeparator drops the hairline under the bar.
	//
	// The rule is on by default because the zero value has to work on the
	// zero-value screen: an unstyled bar sits on the same Background as the
	// content below it, and with nothing between them the title reads as the
	// first line of the page. A bar given its own fill through Style
	// separates itself and will usually want this set.
	HideSeparator bool

	// Style is applied to the bar row, after the widget's own defaults and
	// before the children — so padding, background and alignment are all
	// overridable. It does not reach the separator; use a Leading/Content
	// slot or your own Separator for that.
	Style []core.StyleProp
}

func (a AppBar) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(a.Actions)+len(a.Style)+5)
	// The bar's own defaults first so Style can override every one of them.
	// The theme's Row base already supplies the padding; what a row of mixed
	// heights adds is a shared centre line and a gap between slots — the same
	// two ListRow adds, for the same reason.
	items = append(items,
		core.BackgroundColor(t.Colors.Background),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
	)
	// The banner landmark: the region a reader jumps to for "where am I and
	// what can I do from here". It goes on the row rather than on the Box the
	// separator case wraps around it, so that it lands on the same element in
	// both shapes — the alternative moves the role between two elements
	// depending on a cosmetic flag, and only one of them is what Style
	// reaches. What the row leaves out is the hairline, which is decoration.
	//
	// Before Style, so a caller can override it: a second bar on a screen that
	// already has a banner, or a bar that is really a section header, says so
	// with core.AccessibilityRole in Style.
	items = append(items, core.AccessibilityRole(core.RoleBanner))
	for _, sp := range a.Style {
		items = append(items, sp)
	}

	if lead := a.leading(ctx); lead != nil {
		items = append(items, lead)
	}
	items = append(items, a.middle(t))
	for _, action := range a.Actions {
		if action != nil {
			items = append(items, action)
		}
	}

	bar := core.Row(items...)
	if a.HideSeparator {
		return bar.Render(ctx)
	}
	// Box, not Column: it is the only container with no theme base, so the
	// bar and its rule stack with nothing inserted between or around them.
	// A Column here would contribute the theme's screen-level padding and
	// gap, both of which would have to be undone. (Separator's doc makes the
	// same choice for the same reason.)
	return core.Box(bar, Separator{}).Render(ctx)
}

// leading resolves the start of the bar: the caller's slot, the automatic
// back control, or nothing at all.
func (a AppBar) leading(ctx *core.Context) core.View {
	if a.Leading != nil {
		return a.Leading
	}
	if a.HideBack || !core.CanPop(ctx) {
		return nil
	}

	glyph := a.BackGlyph
	if glyph == "" {
		glyph = "‹"
	}
	onBack := a.OnBack
	if onBack == nil {
		// Captured rather than passed: Pop needs the context, and the
		// callback outlives this render pass in the registry.
		onBack = func() { core.Pop(ctx) }
	}
	return Button{
		Label:    glyph,
		Emphasis: EmphasisGhost,
		OnTap:    onBack,
		// A screen reader announcing "‹" announces nothing. Every glyph
		// control in this package carries a real name; see Button.
		AccessibilityLabel: "Back",
		Style: []core.StyleProp{
			// Larger than the surrounding text because a single chevron at
			// body size is a small target and a smaller mark; the tightened
			// padding keeps the taller glyph from stretching the bar.
			core.FontSize(26),
			core.PaddingHorizontal(6),
			core.PaddingVertical(0),
		},
	}
}

// middle builds the growing centre — the slot that takes the bar's slack and
// thereby pins Actions to the trailing edge.
func (a AppBar) middle(t *core.Theme) core.View {
	if a.Content != nil {
		return core.Box(core.FlexGrow(1), a.Content)
	}

	items := make([]core.PropsAndChildren, 0, 4)
	items = append(items, core.FlexGrow(1), core.Gap(float64(t.Spacing.XS)))

	if a.Title != "" {
		// Subtitle's size, not Title's. Typography.Title is the screen's
		// large heading (28pt under DefaultTheme) and does not fit in a bar;
		// Subtitle is 22 and 18 in the two bundled themes, which is the size
		// a bar title wants and scales with the theme rather than with a
		// literal here. Its weight and ink are overridden after it: a bar
		// title is the screen's name, so it takes Bold and the primary ink,
		// where Subtitle's own Normal + secondary ink says "supporting line".
		items = append(items, core.Text(a.Title,
			core.UseStyle(t.Typography.Subtitle),
			core.FontWeight(core.Bold),
			core.TextColor(t.Colors.TextPrimary),
			// The screen's name is the screen's heading — the thing a reader
			// navigating by heading expects to land on first, and the one
			// role in this widget that is true on all four targets rather
			// than on the web alone. Only the Title takes it: a Subtitle is a
			// supporting line, and a Content slot is the caller's to describe.
			core.AccessibilityRole(core.RoleHeading),
			// Level 1, because a bar title is the top of the screen's outline
			// by construction: an AppBar is the screen's own bar, so there is
			// nothing above it for it to be a section of. This and
			// GroupedList's bands (level 2) are the pair that made the field
			// worth having — without a tier the two announce as peers, and a
			// reader navigating by heading cannot tell the screen's name from
			// a band inside it. Set by the widget rather than asked for,
			// exactly like the role it accompanies.
			core.AccessibilityHeadingLevel(1),
		))
	}
	if a.Subtitle != "" {
		items = append(items, core.Text(a.Subtitle, core.UseStyle(t.Typography.Caption)))
	}
	return core.Box(items...)
}
