# Package comps — Screens & structure

```go
import "github.com/rohanthewiz/grmob/comps"
```

Screen, app and bottom bars, tabs, drawers, step indicators, two-pane and foldable layouts, cards, accordions, headings and separators.

One of 7 topic pages of [package comps](comps.md), which has the package overview and an index of every topic. This page documents the declarations in `comps/screen.go`, `comps/app_bar.go`, `comps/bottom_bar.go`, `comps/tabs.go`, `comps/drawer.go`, `comps/step_indicator.go`, `comps/two_pane.go`, `comps/card.go`, `comps/accordion.go`, `comps/disclosure.go`, `comps/heading.go`, `comps/separator.go`.

## Index

- [`type Accordion`](#type-accordion)
    - [`func (Accordion) Render`](#func-accordion-render)
- [`type AppBar`](#type-appbar)
    - [`func (AppBar) Render`](#func-appbar-render)
- [`type BarItem`](#type-baritem)
- [`type BottomBar`](#type-bottombar)
    - [`func (BottomBar) Render`](#func-bottombar-render)
- [`type Card`](#type-card)
    - [`func (Card) Render`](#func-card-render)
- [`type Drawer`](#type-drawer)
    - [`func (Drawer) Render`](#func-drawer-render)
- [`type DrawerItem`](#type-draweritem)
- [`type Screen`](#type-screen)
    - [`func (Screen) Render`](#func-screen-render)
- [`type Separator`](#type-separator)
    - [`func (Separator) Render`](#func-separator-render)
- [`type StepIndicator`](#type-stepindicator)
    - [`func (StepIndicator) Render`](#func-stepindicator-render)
- [`type Tabs`](#type-tabs)
    - [`func (Tabs) Render`](#func-tabs-render)
- [`type TwoPane`](#type-twopane)
    - [`func (TwoPane) Render`](#func-twopane-render)
- [`type TwoPaneCompact`](#type-twopanecompact)

## Types

### type Accordion

```go
type Accordion struct {
	Title string
	// Header replaces the default chevron+Title header content when set.
	// The tap target and toggle behavior stay with the Accordion either way.
	Header  core.View
	Content core.View

	// HeadingLevel is where the Title sits in the screen's outline. Zero is
	// level 3 — a disclosure sits inside a section, one tier below a Card
	// title or a GroupedList band — and a screen built entirely of accordions
	// under an AppBar should say 2.
	//
	// It applies to the default header only. A Header replaces that content
	// and is the caller's to describe, on the same division Card.Title and
	// Card.Header draw.
	//
	// # The heading wraps the control, which is ARIA's own accordion shape
	//
	// This is one of the two widgets in the package whose heading is not on
	// the words. The rest put the role on the Text node, because a row also
	// holds a badge or a chevron and a heading spanning the row would be named
	// "▸ What is a hook" rather than "What is a hook". Here the tier rides a
	// Box *around* the header row, and the row is a button:
	//
	//	Box    role=heading, aria-level=3, aria-label="What is a hook"
	//	  Row  role=button,  aria-expanded="false", aria-label="What is a hook"
	//	    Text "▸"        presentational, inside the button
	//	    Text "What is a hook"
	//
	// It took three tries to land there, both of the rejected ones looked
	// right, and the whole argument now lives on comps.disclosure — the
	// shared shape this widget and the collapsible GroupedList band are both
	// built out of. It moved there when the band arrived, because two copies
	// of a four-paragraph argument do not stay in step: the failure is not
	// that the second copy is wrong on the day it lands, it is that a later
	// fix to one leaves the other announcing something else.
	//
	// The wrapper is added only for the default header. A Header replaces the
	// content and is the caller's to describe, on the same division Card.Title
	// and Card.Header draw — so a custom header gets the button and its state
	// and no heading at all.
	//
	// See headingLevel in heading.go for the package's outline and for how to
	// ask for a heading with no tier at all.
	HeadingLevel int

	// InitiallyExpanded seeds the state on the first pass only; after that
	// the accordion follows the user's taps.
	InitiallyExpanded bool
	// Style is applied to the outer column.
	Style []core.StyleProp
}
```

Accordion is a collapsible section: a tappable header that shows or hides its Content.

It owns its expanded/collapsed state via NewState, which makes it the one widget in this package with hook obligations: render an Accordion unconditionally, in a stable position, every pass — exactly the rules for calling NewState directly (core.SetDebugMode reports violations as cursor-drift concerns). Content, on the other hand, is only rendered while expanded, so it must not contain hooks of its own: they would come and go with the toggle, which is the conditional-hook bug. Interactive, hook-free content (buttons, inputs bound to parent state) is fine — its callbacks re-register on every pass the content is visible.

<small>[comps/accordion.go:17](https://github.com/rohanthewiz/grmob/blob/master/comps/accordion.go#L17)</small>

#### func (Accordion) Render

```go
func (a Accordion) Render(ctx *core.Context) *core.Node
```

<small>[comps/accordion.go:70](https://github.com/rohanthewiz/grmob/blob/master/comps/accordion.go#L70)</small>

### type AppBar

```go
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
	//
	// Android's system back runs it too, whenever the automatic control is
	// drawn: the bar row carries it as core.OnBack, which outranks the plain
	// pop core.Navigator attaches to the route around it. A Leading slot or
	// HideBack draws no control and attaches nothing, so back there is the
	// Navigator's pop.
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
```

AppBar is the title strip at the top of a screen: an optional back affordance, the screen's name, and trailing actions.

	comps.AppBar{Title: "Sermons"}
	comps.AppBar{Title: "Sermon", Actions: []core.View{shareButton}}

	┌ Box ─────────────────────────────────────────────────────────┐
	│ ┌ Row ───────────────────────────────────────────────────────┤
	│ │ [‹]  ┌ Box FlexGrow(1) ──────────┐  [Action] [Action]      │
	│ │      │ Title                     │                         │
	│ │      │ Subtitle                  │                         │
	│ │      └───────────────────────────┘                         │
	│ ├────────────────────────────────────────────────────────────┤
	│ │ Separator                                                  │
	└─┴────────────────────────────────────────────────────────────┘
	       └──── takes all the slack, pinning Actions right ────┘

#### It is a Row, not a platform navigation bar

grmob has no AppBar node, so there is nothing here that floats over the content, collapses on scroll, claims the status bar, or animates a title between screens. This is an ordinary row of ordinary widgets that happens to sit first in a screen's column — which is what makes it identical on all four targets, and what makes comps.Screen's SafeArea, not the bar, responsible for keeping it clear of the notch.

#### The back control appears only when there is somewhere to go

With no Leading and no HideBack, the bar draws a back button exactly when core.CanPop says the navigation stack has a screen underneath. The root of a tab shell therefore gets no arrow without the caller having to say so, and a pushed detail screen gets one without the caller having to wire it — which is the behavior every navigation framework has, reached here through the one question core exposes. See core.CanPop for why asking beats rendering a control that would no-op.

#### Slots beat fields, as everywhere in this package

Content overrides Title/Subtitle and Leading overrides the back control, the same simple-path-plus-slot idiom as Card.Title vs Card.Header. Setting Leading is also how you get a back control the widget would not have drawn — a close button on a modally presented screen, where CanPop is false.

<small>[comps/app_bar.go:47](https://github.com/rohanthewiz/grmob/blob/master/comps/app_bar.go#L47)</small>

#### func (AppBar) Render

```go
func (a AppBar) Render(ctx *core.Context) *core.Node
```

<small>[comps/app_bar.go:109](https://github.com/rohanthewiz/grmob/blob/master/comps/app_bar.go#L109)</small>

### type BarItem

```go
type BarItem struct {
	// Label is the text under the icon and the item's accessible name.
	Label string

	// Icon is an optional glyph or emoji drawn above the label.
	Icon string

	// OnTap is called when the item is pressed.
	OnTap func()

	// AccessibilityLabel replaces Label as the spoken name, for a bar whose
	// labels are abbreviated.
	AccessibilityLabel string
}
```

BarItem is one cell of a BottomBar.

<small>[comps/bottom_bar.go:75](https://github.com/rohanthewiz/grmob/blob/master/comps/bottom_bar.go#L75)</small>

### type BottomBar

```go
type BottomBar struct {
	// Items are the destinations or actions, drawn leading to trailing.
	Items []BarItem

	// Selected is the index of the current destination. A negative value makes
	// the bar an action toolbar with no current item.
	Selected int

	// Style is applied to the bar row after the widget's own props.
	Style []core.StyleProp
}
```

BottomBar is the strip pinned to the bottom of a screen: two to five destinations (Home, Search, Me) or two to five actions (Reply, Archive, Delete), each an icon over a label.

	comps.Screen{
	    Children: []core.View{feed},
	    Footer: comps.BottomBar{
	        Items: []comps.BarItem{
	            {Icon: "🏠", Label: "Home", OnTap: home},
	            {Icon: "🔍", Label: "Search", OnTap: find},
	            {Icon: "👤", Label: "Me", OnTap: profile},
	        },
	        Selected: tab.Get(),
	    },
	}

It belongs in Screen.Footer, which pins it outside the scrolling region; a bar placed among Screen.Children scrolls away with the content.

#### Destinations or actions: the role follows Selected

SegmentedControl is a choice between views of one thing and styles its live segment as a filled chip, which is wrong for a bar. The bar's one design decision is what it announces itself as, and it is made by the field the caller already has to set:

	Selected >= 0   RoleNavigation   a set of destinations, one of them current
	Selected <  0   RoleToolbar      a strip of actions, none of them "current"

Selected's zero value picks the first item, the same rule SegmentedControl and Tabs follow, so an action strip says Selected: -1.

#### Items share the width equally

Each item is a Column with FlexGrow(1), so the tap targets tile the whole bar. Justify(JustifyAround) on content-sized buttons would draw the same spacing but leave dead gaps between targets, which is exactly where a thumb aimed at a small label lands.

#### How the current item is announced

The current cell states core.CurrentPage: aria-current="page" on the web, and the selected state on Compose and SwiftUI, which is how both platforms' own navigation bars announce the destination they show. It used to append ", selected" to its name, because core had no current state and RoleTab would claim a tab panel this bar does not control (examples/social builds that relationship explicitly when it wants it). The name is now the label alone, and stays the same as the selection moves. The Icon is decoration and is hidden from assistive technology, so the Label is what is read.

#### Theme roles read

	Bar background   Colors.Surface
	Current item     Colors.PrimaryOnLightColor(), bold
	Other items      Colors.TextSecondary
	Label text       Typography.Caption; Icon uses Typography.Subtitle
	Padding          Spacing.XS

<small>[comps/bottom_bar.go:62](https://github.com/rohanthewiz/grmob/blob/master/comps/bottom_bar.go#L62)</small>

#### func (BottomBar) Render

```go
func (b BottomBar) Render(ctx *core.Context) *core.Node
```

Render builds Row(Column(icon, label)...) with the role chosen by Selected.

<small>[comps/bottom_bar.go:91](https://github.com/rohanthewiz/grmob/blob/master/comps/bottom_bar.go#L91)</small>

### type Card

```go
type Card struct {
	Title  string
	Header core.View // overrides Title when set
	Body   core.View
	Footer core.View

	// HeadingLevel is where the Title sits in the screen's outline. Zero is
	// level 2 — a card is a section of a screen, the same tier as a
	// GroupedList band — and a card nested inside one of those should say 3.
	//
	// It applies to Title alone. A Header replaces the line entirely, and the
	// view in it is the caller's to describe: this widget cannot know whether
	// it was handed a heading, a row of controls, or an avatar.
	//
	// See headingLevel in heading.go for the package's outline and for how to
	// ask for a heading with no tier at all.
	HeadingLevel int

	// Style is applied to the card container after the theme's Card base.
	Style []core.StyleProp
}
```

Card is a surface with optional header, body, and footer regions, rendered on core.Card (so it inherits the theme's Card base: background, padding, radius, shadow).

Title is the simple path — a themed title line; Header is the escape hatch — an arbitrary view in the same position, taking precedence over Title when both are set. This mirrors element's Card, whose Body (string) and BodyComponent (component) coexist the same way.

<small>[comps/card.go:13](https://github.com/rohanthewiz/grmob/blob/master/comps/card.go#L13)</small>

#### func (Card) Render

```go
func (c Card) Render(ctx *core.Context) *core.Node
```

<small>[comps/card.go:35](https://github.com/rohanthewiz/grmob/blob/master/comps/card.go#L35)</small>

### type Drawer

```go
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
	// four targets, and so is a core.MaxWidth passed through PanelStyle to cap
	// it.
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
```

Drawer is side navigation: a panel of destinations pinned to the leading edge over the screen, opened by a ☰ button and closed by picking a destination, by its ✕, or by a tap on the scrim beside it.

	open := core.NewState(ctx, false)
	closeRef := core.UseFocusRef(ctx)
	comps.Drawer{
	    Open:      open.Get(),
	    OnDismiss: func() { open.Set(false) },
	    Title:     "Notebook",
	    Items: []comps.DrawerItem{
	        {Icon: "📥", Label: "Inbox",   OnTap: func() { section.Set(0) }},
	        {Icon: "⭐", Label: "Starred", OnTap: func() { section.Set(1) }},
	    },
	    Selected: section.Get(),
	    CloseRef: closeRef,
	    Content: comps.Screen{Children: []core.View{
	        comps.AppBar{Title: "Inbox", Leading: comps.Button{
	            Label: "☰", AccessibilityLabel: "Open navigation",
	            Emphasis: comps.EmphasisGhost,
	            OnTap: func() { open.Set(true); core.Focus(closeRef) },
	        }},
	        body,
	    }},
	}

	┌ ZStack  Width 100%, then Style ──────────────────────────────┐
	│ ┌ Box 100% × 100%   AccessibilityHidden while Open ────────┐ │ layer 1:
	│ │ Content                                                  │ │ the screen
	│ └──────────────────────────────────────────────────────────┘ │
	│ ┌ Row 100% × 100%, clipped; inert + hidden while shut ─────┐ │ layer 2:
	│ │ ┌ Column  Width 280px ──┐ ┌ Box FlexGrow(1) ───────────┐ │ │ drawn over
	│ │ │ navigation, "Notebook"│ │ scrim: Backdrop fill,      │ │ │ layer 1
	│ │ │ Notebook          [✕] │ │ tap = OnDismiss,           │ │ │
	│ │ │ 📥 Inbox   (selected) │ │ hidden from assistive tech │ │ │
	│ │ │ ⭐ Starred            │ │                            │ │ │
	│ │ └───────────────────────┘ └────────────────────────────┘ │ │
	│ └──────────────────────────────────────────────────────────┘ │
	└──────────────────────────────────────────────────────────────┘

#### A layer over the content, not a Modal

ActionSheet and Menu present through core.Modal, and a drawer could too. It does not, because a Modal's content lands wherever each host puts a dialog, and only the web puts it anywhere a leading edge could be reached from:

	target    Modal presents as                  a leading panel would be
	───────   ────────────────────────────────   ──────────────────────────
	web ×2    fixed overlay, flex column         possible: a Row filling it
	Compose   Dialog window at platform width,   a panel inside a centred
	          inset from every screen edge       window, off the edge
	SwiftUI   .sheet, medium/large detents       a bottom sheet

A ZStack draws its layers in one box on all four targets (a single-cell grid on the web, a Box on Compose, a stack Layout on SwiftUI), and a layer sized Width/Height "100%" fills that box on each. So the panel is the stack's second layer, and it is a drawer at the leading edge everywhere, with no host change.

What the Modal chassis would have supplied is the cost, and the widget buys back what it can:

  - Screen-reader confinement. A Dialog window and a sheet confine TalkBack and VoiceOver, and the DOM overlay says aria-modal. Here the content layer takes AccessibilityHidden while the drawer is open, which takes the screen behind out of the accessibility tree on every target, so exploration stays in the panel.
  - Focus. Nothing moves it on open, and the ☰ that had it is now inside a hidden layer. CloseRef names the ✕ so the opener can call core.Focus on it, as the example does; OnDismiss can hand focus back to the ☰ through that button's own FocusRef. The widget holds no ref itself, because a ref is a hook (see "No hooks").
  - Keyboard containment on the web. aria-hidden does not stop Tab, so the content layer is also core.Inert while the drawer is open: Tab and Shift-Tab stay in the panel (and the browser's own chrome), and a pointer cannot reach the screen through a gap in the scrim. The phones do not read Inert; a hardware keyboard there can still reach the screen, which Style.Inert records.
  - The Android back button. A Dialog closes on it; a layer does not, so the panel layer carries core.OnBack(OnDismiss) while open. The panel is inside the screen and composed after it, so back closes the drawer before a Navigator pops or an AppBar's back runs; the next back is theirs. With OnDismiss nil there is nothing to call and back falls through to them.

#### It covers its own box, so give it one

The drawer covers the ZStack, not the window, and a ZStack is as big as its largest layer. Both layers ask for 100% of what the parent offers, so where the parent bounds the height (the app root, a Screen with Fill, a pinned Height) the drawer covers exactly that. In a scrolling column nothing bounds it: Compose's fill height is ignored under an unbounded constraint and the panel would sit centred at its own height. Pin a height through Style there, as the tutorial's demo does with core.Height.

#### The panel is off-screen when shut, not removed

The panel layer renders on every pass and stays displayed while shut. The tree is then the same shape open or shut, so opening is a style patch, and any hooks inside Body keep their slots: Body left out of a pass would shift every hook rendered after it, which is Accordion's rule. The content layer is always wrapped in its Box for the same reason. Toggling a prop is a patch; adding a wrapper around the screen would replace the screen.

#### It slides

The panel comes in from the leading edge and goes back out, and the scrim fades with it. Both are core.Transition on a style change, and a transition animates a change to a node that is displayed on both sides of it, so the shut panel is displayed and moved away rather than hidden with Display none (which both natives read as "not composed", and which CSS does not transition out of either):

  - The move is core.Translate("-100%", "") on the shut panel, and none on the open one. The percentage is of the panel's own width, so a percentage Width or a PanelStyle MaxWidth still hides it exactly, and Translate is leading-relative, so a right-to-left layout hides it off the right edge with no branch here.
  - The layer clips (Overflow "hidden", which both natives read as a clip for this), so the shut panel does not draw over whatever sits beside the drawer's box: the page around the tutorial's pinned demo, say.
  - Opening decelerates over 250ms (EaseOut) and closing accelerates over 200ms (EaseIn), the shape Material gives a panel that arrives and leaves. Every target times a change by the Transition of the style being moved to, so the shut style's own Transition times the close.
  - Under the platform's reduce-motion setting both snap, which is core.Transition's rule; the drawer then appears and disappears as it used to.

What a displayed shut panel would otherwise cost, and what buys it back:

  - Touch and pointer. The shut scrim has no fill and no tap handler, and the panel sits clipped away, so nothing on the layer takes a touch on the phones and taps reach the screen beneath. On the web the layer is Inert, which drops its pointer events, so clicks fall through too.
  - Readers and Tab. The whole layer is AccessibilityHidden and Inert while shut, so no reader finds the panel and, on the web, Tab skips it.
  - A hardware keyboard on the phones. The natives do not read Inert (see core.Style.Inert), so a shut panel's rows are composed and reachable by a keyboard's focus traversal on an iPad or a Chromebook, where a Display none panel was not composed at all. Disabled would stop that and is not used: it dims the ✕, a native Button, for the length of the slide out. Recorded rather than approximated, as Inert's own gap is.
  - Composition. The panel's subtree is composed while shut. A handful of rows is cheap; a Body holding a long list would pay for it.

#### Picking a destination closes the drawer

A row calls its item's OnTap, then OnDismiss, as an ActionSheet action does and as Material's navigation drawer does. A caller would otherwise write open.Set(false) into every destination.

The current destination is announced the way BottomBar's is: its row states core.CurrentPage through ListRow.Current, which is aria-current="page" on the web and the selected state on both natives. It used to take ListRow's ", selected" name suffix, before core had a current state. Selected also follows BottomBar and Tabs: the zero value selects the first item, and a negative Selected selects none.

#### No hooks

Open and focus are both the caller's, so Drawer takes no hook slot and is conditional-safe, like Menu.

#### Theme roles read

	Panel      Colors.Background
	Title      Typography.Subtitle, bold, Colors.TextPrimary
	Close      as comps.Button, ghost
	Rows       as comps.ListRow, whose selected look marks the current one
	Icon       Typography.Subtitle
	Scrim      Backdrop, else core.Modal's default #00000088

<small>[comps/drawer.go:177](https://github.com/rohanthewiz/grmob/blob/master/comps/drawer.go#L177)</small>

#### func (Drawer) Render

```go
func (d Drawer) Render(ctx *core.Context) *core.Node
```

Render builds ZStack(content layer, panel layer) as drawn in the type doc.

<small>[comps/drawer.go:265](https://github.com/rohanthewiz/grmob/blob/master/comps/drawer.go#L265)</small>

### type DrawerItem

```go
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
```

DrawerItem is one destination in a Drawer.

<small>[comps/drawer.go:234](https://github.com/rohanthewiz/grmob/blob/master/comps/drawer.go#L234)</small>

### type Screen

```go
type Screen struct {
	// Children are the screen's content, laid out top to bottom in the
	// column. A nil entry is skipped (see above).
	Children []core.View

	// Scroll wraps the column in core.Scroll, making the whole screen
	// scrollable. Leave it false when the screen has its own scrolling region
	// inside it — a core.List, or a Scroll around one section — since a
	// scroll view nested in a scroll view fights for the same drag on both
	// natives.
	//
	// Following that advice with a List costs nothing in layout: the scaffold
	// drops its own inset when the List is the whole content, so the page sits
	// where the scrolled Column drew it. See "A scrolling child is the page".
	Scroll bool

	// KeyboardAware makes that scroll region shrink to sit above the software
	// keyboard, so a focused field near the bottom of a form is scrolled
	// somewhere visible rather than under the keys. See core.KeyboardAware
	// for what each platform does with it and what it deliberately does not
	// cover.
	//
	// Without Scroll it lifts the content column whole instead, which is the
	// behavior a screen with something docked at its bottom wants — chat's
	// composer, a checkout bar — since that bar sits outside any scrolling
	// region and is the one thing the keyboard covers.
	KeyboardAware bool

	// Gap is the uniform vertical spacing between children, in points. Zero
	// means "don't set one", not "zero spacing" — the theme's Column base
	// keeps whatever it had. Use Gap for uniform runs and core.Spacer between
	// specific children when the spacing differs (that rule predates this
	// widget; see examples/fintechapp).
	Gap float64

	// Fill makes the column claim the full height of the safe area
	// (FlexGrow(1)) rather than shrinking to its content. Set it when a child
	// needs to expand into the leftover space — a list that should fill the
	// screen and push a footer to the bottom — because a FlexGrow child can
	// only grow inside a parent that has height to give.
	//
	// Fill with Scroll is legal but unusual: it makes the scrolled content at
	// least as tall as the viewport, which is how you bottom-anchor a footer
	// on a short page. It does not make a scroll view fill anything.
	Fill bool

	// Footer is pinned below the content, outside the scroll region: the slot
	// for a comps.BottomBar, a checkout bar, or a composer that must not scroll
	// away. Nil keeps the scaffold exactly as it is without it.
	//
	// With a Footer the SafeArea holds two children instead of one, and the
	// content region takes the leftover height so the footer sits on the
	// bottom edge even when the content is short:
	//
	//	SafeArea
	//	  ├─ Scroll FlexGrow(1)   (or the Column, growing, when Scroll is false)
	//	  │    └─ Column
	//	  └─ Footer
	//
	// Growing is implied rather than requiring Fill as well, because a footer
	// that floats up under short content is never what a bottom bar means. The
	// footer takes no inset or style from the scaffold: it is the caller's
	// widget, full-width, and draws its own background and padding.
	//
	// KeyboardAware still lands on the scroll region or the column, so a
	// focused field in the content is kept visible. The footer itself is not
	// lifted over the keyboard, which is how both platforms treat a tab bar.
	Footer core.View

	// Style is applied to the column, after Gap and Fill, so a caller can
	// override either — or add padding and a background the scaffold itself
	// has no opinion about.
	Style []core.StyleProp
}
```

Screen is the root scaffold every app in this repo was hand-spelling: the safe-area inset, an optional scroll region, and the vertical column that holds the screen's content.

	SafeArea
	  └─ Scroll            (only when Scroll is true; KeyboardAware rides here)
	       └─ Column       ← Gap / Fill / Style land here
	            ├─ Children[0]
	            └─ …

Five call sites spelled some subset of that by hand, and each picked its own subset — one wrapped in Scroll, two set a Gap, one set FlexGrow(1), and no two agreed on the order the props were written in. The shape is not hard to type; the value of naming it is that there is now one place to hang the things a screen root will eventually need (a pull-to-refresh region, per-platform inset behavior) instead of five — KeyboardAware below is the first of them to arrive.

#### The zero value is exactly SafeArea(Column(children...))

Every field defaults to contributing nothing, so the zero value renders the bare scaffold and no style props at all — the theme's Column base carries through untouched. That is deliberate and it is what let all five migrations below stay byte-identical: a field only speaks when the caller sets it. It applies specifically to Gap, which is applied only when non-zero. An explicit core.Gap(0) would \*overwrite\* a gap the theme's Column base had set (style props mutate the style directly rather than merging into it), so "unset" and "zero" have to mean the same thing here — the absence of a gap, not the imposition of one.

#### Nil children are skipped

Children flows into core.Column's argument list, which skips nil items (the same contract that makes core.MaybeProp work). So the conditional-region idiom this codebase already uses for slots —

	var banner core.View
	if offline {
	    banner = OfflineBanner()
	}
	return comps.Screen{Children: []core.View{banner, body}}

— costs the tree no node at all when the condition is false, rather than the empty Fragment a core.If would leave behind for the reconciler to walk on every pass. (That Fragment draws nothing; the cost is the node, not a gap.)

#### A scrolling child is the page, and is inset once

Screen's column carries the theme's Components.Column base, whose only entry in every bundled theme is the standard 12/16 inset. So does core.List — it is the one other container in the tree built on that same base. A screen whose whole content is a List therefore used to be inset twice, and the doubling is invisible in code because neither inset is written anywhere:

	SafeArea
	  └─ Column   padding 12/16   ← the theme's, via Screen
	       └─ List padding 12/16   ← the theme's, again

Every child of that list drew 16 points further in than the same content in a Scroll'd Column, which is the shape Screen.Scroll's own documentation recommends migrating \*away\* from. So the scaffold now drops its column's padding when its content is a single scrolling page:

	Screen{Children: []core.View{core.List(rows...)}}   inset once, by the List

Three things about the rule are deliberate.

"Only child" is counted after nil entries are skipped, so the conditional-slot idiom above keeps working: a screen holding a nil banner and a List is a single-child screen, exactly as the tree the reconciler walks is. That is why the decision is made on the \*rendered\* child rather than on the Go value — the count that matters is the one core.Column would arrive at, and a wrapper widget (comps.GroupedList) is a List only after it renders.

The set is node types that scroll and arrive pre-inset, which today is core.List alone. core.Scroll is deliberately outside it: a Scroll carries no theme base, so its content is inset once — by this column — and dropping that would move the page rather than unstack it.

Style still wins. The cleared padding is applied ahead of the caller's Style props, so a screen that asks for core.Padding(24) around its list gets 24, and one that wants the old doubled behavior can still spell it. Nothing else about the column changes: a Gap, a Fill and a background all survive, because it is only the inset that was ever duplicated.

<small>[comps/screen.go:90](https://github.com/rohanthewiz/grmob/blob/master/comps/screen.go#L90)</small>

#### func (Screen) Render

```go
func (s Screen) Render(ctx *core.Context) *core.Node
```

<small>[comps/screen.go:215](https://github.com/rohanthewiz/grmob/blob/master/comps/screen.go#L215)</small>

### type Separator

```go
type Separator struct {
	// Color overrides the hairline tint. Empty takes the theme's Border role.
	Color string

	// Thickness is the rule's height in px; 0 means 1. Fractional values are
	// carried through to the platforms as-is (a "0.5px" hairline is a real
	// request on a 2x/3x display, and both renderers parse a float).
	Thickness float64

	// Inset indents the rule from both ends, in px. This is the list idiom
	// where the rule starts under the text rather than under the leading
	// avatar or checkbox, so the leading column reads as one continuous
	// stripe.
	//
	// It is applied as core.MarginHorizontal, which writes the two explicit
	// sides as well as the axis shorthand. The field has now been spelled
	// three ways for one reason each: a Left/Right pair, because the two web
	// targets once read the per-side fields only and dropped the shorthand; a
	// bare EdgeInsets.Horizontal, once every renderer resolved the shorthand
	// into the unset sides; and the prop, because a whole EdgeInsets through
	// UseStyle replaces Margin outright and so clears any vertical gap the
	// caller's Style below asked for. The prop touches one axis and leaves
	// the other alone, which is what this field always meant.
	Inset int

	// Style is applied last, so every default above is overridable.
	Style []core.StyleProp
}
```

Separator is the hairline rule between list rows and between sections.

core.Divider already draws a line, but it takes a literal color and force-applies Margin(8) — an unconditional gap that makes it wrong inside a list, which is presumably why neither example that wanted a rule used it. Separator leaves spacing to the caller and takes its tint from the theme's Border role, so the common case is the zero value:

	comps.Separator{}

The tint reads through ColorPalette.BorderColor rather than off the Border field, so a theme written before that role existed draws a visible default hairline instead of an invisible one. Surface would have been the nearest pre-existing neutral and is the wrong answer: it is a \*fill\* color, so on a Surface-colored panel a Surface hairline disappears.

It is always hidden from assistive technology. A rule is decoration: it carries no information a screen reader can use, and announcing one between every pair of rows in a list turns a 20-row feed into 39 utterances.

#### Horizontal only

There is no Vertical field yet. A vertical rule has to stretch to its row's height, which means cross-axis stretch, and that used to be the blocker: neither renderer mapped AlignItems "stretch", so the field would have advertised something that collapsed to zero height on both platforms.

Both renderers map it today — Compose pins a stretched Row to IntrinsicSize.Max and gives each child fillMaxHeight(), and SwiftUI's GrMobFlexStack proposes the full cross extent to a stretched child — so the field is now a widget change rather than a renderer one, waiting on a caller that wants it. One asymmetry to know when it lands: a Row reads alignItems alone and never the simpler Align fallback (Align is a text-alignment concept and has never applied to a row's vertical axis), so the containing row has to spell out AlignItems "stretch".

<small>[comps/separator.go:44](https://github.com/rohanthewiz/grmob/blob/master/comps/separator.go#L44)</small>

#### func (Separator) Render

```go
func (s Separator) Render(ctx *core.Context) *core.Node
```

<small>[comps/separator.go:73](https://github.com/rohanthewiz/grmob/blob/master/comps/separator.go#L73)</small>

### type StepIndicator

```go
type StepIndicator struct {
	// Steps are the step names, in order.
	Steps []string

	// Current is the zero-based index of the step in progress. It is clamped
	// into the Steps range, so a flow's "finished" index draws every step but
	// the last as done.
	Current int

	// OnTap receives the index of a tapped done step. Nil makes the strip a
	// display of progress with no targets.
	OnTap func(int)

	// Label prefixes the strip's accessible name, for a screen with more than
	// one flow ("Checkout, step 2 of 4: Address").
	Label string

	// StepLabel builds one step's accessible name from its zero-based index,
	// its entry in Steps and whether it is done. Nil gives "Step 2: Address",
	// with ", done" after a done step's. The current step's state is not a
	// parameter: it is announced as core.CurrentStep, in the platform's words.
	StepLabel func(index int, label string, done bool) string

	// PositionLabel builds the strip's accessible name from the zero-based
	// current index, the number of steps and the current step's entry in
	// Steps. Nil gives "Step 2 of 4: Address", prefixed by Label when set.
	// Label is not applied to a caller's PositionLabel, which can include it
	// itself.
	PositionLabel func(current, total int, label string) string

	// Style is applied to the strip after the widget's own props.
	Style []core.StyleProp
}
```

StepIndicator is the "step 2 of 4" header of a multi-screen flow: numbered circles joined by short rules, done steps ticked, the current step filled.

	comps.StepIndicator{
	    Steps:   []string{"Account", "Address", "Payment", "Review"},
	    Current: step.Get(),
	    OnTap:   step.Set, // done steps only
	}

	┌ Scroll Horizontal  role=navigation  "Step 2 of 4: Address" ───────────┐
	│ (✓) Account ── (2) Address ── (3) Payment ── (4) Review               │
	│  done, button    current        upcoming       upcoming              │
	└───────────────────────────────────────────────────────────────────────┘

#### The overflow decision: scroll, not collapse

Five or six labelled steps do not fit a phone's width. The plan offered two answers: collapse to "2 / 4" past a threshold, or let the strip scroll sideways. Collapsing needs a width to compare against, which Go does not have, so any threshold would be a guess that is wrong on some screen. The strip is a core.Scroll with core.Horizontal, which every target already scrolls, and a flow short enough to fit simply does not move.

The Scroll is the strip, as it is in ChipStrip, and nothing wraps it. Its first version sat inside a Row because the web host pages sized every Scroll as a screen's vertical viewport, which collapsed a sideways strip to 0px inside a column. That rule now leaves horizontal Scrolls their content height (see the Scroll rules in wasm/index.html), so the wrapper went away.

#### Which steps can be tapped

Only done steps, and only when OnTap is set. Going back to fix an address is what a flow allows; jumping ahead past a step that has not been validated is not, and a widget that made upcoming steps tappable would push every caller to write that guard. The current step is not tappable either: tapping it would do nothing.

#### Accessibility

The strip is RoleNavigation when OnTap is set, because done steps then are destinations, and RoleGroup when it is not, because a picture of progress is not navigation. Either way its name states the position: "Step 2 of 4: Address", prefixed by Label when one is given. Each step is named ("Step 1: Account, done", "Step 2: Address", "Step 3: Payment") and a tappable one is RoleButton. The circle, its glyph and the rules are drawn for sighted users and hidden from assistive technology, so a step is read once.

The current step states core.CurrentStep: aria-current="step" on the web and the selected state on both natives. It used to be a ", current" name suffix. "done" stays a word in the step's name, because no platform has a completed state: ARIA has no attribute for it, and Compose's and SwiftUI's semantics have no property either. The three ARIA near misses each say something false:

	aria-current    names the one step the flow is on. A done step is exactly
	                the one it is not on, and core.CurrentStep already marks
	                the step that is.
	aria-checked    a checkbox's state. It announces a control the user ticks
	                and unticks, and no step is ticked by being tapped.
	aria-selected   one choice of a set. Every done step is done at once, so
	                there is no set for it to be one of — and a tappable done
	                step is a button, which ARIA does not give aria-selected.

comps.Calendar's ", today" left its name when core.CurrentDate gave the web targets a word for it; ", done" has no such word to move to.

#### Names in the app's language

Since the word has to be in the name, the names are the caller's to write. StepLabel builds each step's name and PositionLabel the strip's; nil gives the English defaults above. This is the seam Calendar's DayLabel and MonthLabel are: the widget knows the facts (index, total, done), and the app knows the language.

#### Theme roles read

	Done circle        Colors.SuccessColor(), ink chosen by Variant.Ink
	Current circle     Colors.Primary, ink chosen by Variant.Ink
	Upcoming circle    Colors.ControlBorder ring (BorderColor() if unset),
	                   Colors.TextSecondary number
	Rule               Colors.SuccessColor() after a done step, else
	                   Colors.BorderColor()
	Labels             Typography.Body; the current step bold, upcoming steps
	                   in Colors.TextSecondary
	Glyphs             Typography.Caption, bold
	Gap                Spacing.XS

<small>[comps/step_indicator.go:97](https://github.com/rohanthewiz/grmob/blob/master/comps/step_indicator.go#L97)</small>

#### func (StepIndicator) Render

```go
func (s StepIndicator) Render(ctx *core.Context) *core.Node
```

Render builds the horizontal Scroll of steps and rules.

<small>[comps/step_indicator.go:144](https://github.com/rohanthewiz/grmob/blob/master/comps/step_indicator.go#L144)</small>

### type Tabs

```go
type Tabs struct {
	Items []core.TabItem // build with core.Tab(label, icon)
	// Selected is the controlled selection index; pair it with OnChange
	// writing to the state it is read from.
	Selected int
	OnChange func(int)
	// Content holds the tab pages. By the core.TabView contract all pages
	// are children of the node; the native side shows the selected one.
	Content []core.View
}
```

Tabs is the named-field facade over core.TabView.

Wrap, not supersede (the open question in the element-lessons plan): core.TabView defines a wire contract — the "TabView" node type with its tabs/selectedIndex/onTabChange props that the native renderers consume — and node-type contracts belong in core, next to the registry of types the renderers know. What TabView lacks is only ergonomics: four positional option props with no record of which is which at a call site. This struct supplies the field names and delegates everything else, so there is exactly one tab implementation to keep in sync with the renderers.

<small>[comps/tabs.go:15](https://github.com/rohanthewiz/grmob/blob/master/comps/tabs.go#L15)</small>

#### func (Tabs) Render

```go
func (t Tabs) Render(ctx *core.Context) *core.Node
```

<small>[comps/tabs.go:26](https://github.com/rohanthewiz/grmob/blob/master/comps/tabs.go#L26)</small>

### type TwoPane

```go
type TwoPane struct {
	// First and Second are the two panes: list and detail, content and
	// controls, left page and right page. First is leading (left in a
	// left-to-right layout) or top.
	First, Second core.View

	// Ratio is the fraction of the width First takes in a ratio split (rule
	// 3). Zero means 0.5; a value outside (0, 1) is clamped to it. A list
	// beside a detail usually wants about 0.4.
	Ratio float64

	// Gap is the space between the panes in a ratio split and in a stacked
	// compact layout. Zero means the theme's MD spacing. It is not applied
	// around a hinge — see "Lining up with the hinge".
	Gap float64

	// SplitAt is the smallest width class that splits side by side without
	// a fold. Zero means core.SizeMedium, which is where an unfolded
	// book-style foldable and a portrait tablet land; core.SizeExpanded keeps
	// medium windows single-pane for content that needs width on both sides.
	SplitAt core.SizeClass

	// Compact is what a compact window with no separating fold shows. Zero is
	// TwoPaneStack.
	Compact TwoPaneCompact

	// Origin is the TwoPane's top-left in window coordinates, for aligning to
	// a hinge when the pane does not start at the window's corner. Only X and
	// Y are read.
	Origin core.WindowRect

	// IgnoreHorizontalFold skips rule 2, for a TwoPane inside a scrolling
	// column. Scrolling moves the pane up and down past a horizontal hinge,
	// so no fixed Origin.Y can say where the hinge falls in it, and a First
	// sized to a stale offset is worse than a ratio split. A vertical hinge
	// is unaffected: vertical scrolling does not move anything sideways.
	IgnoreHorizontalFold bool

	// Style is applied to the container after the widget's own props.
	Style []core.StyleProp
}
```

TwoPane lays two views out side by side when the window has room for both, one after the other (or just one) when it does not, and — on a foldable — on either side of the hinge rather than across it.

	comps.TwoPane{
	    First:   inboxList,
	    Second:  messageDetail,
	    Compact: comps.TwoPaneFirst, // a phone shows the list; a tap navigates
	}

#### The decision, in order

The first rule that applies wins:

 1. a separating fold, vertical     Row:    First | hinge | Second
 2. a separating fold, horizontal   Column: First / hinge / Second
 3. width class ≥ SplitAt           Row:    First | Gap | Second, by Ratio
 4. otherwise (compact)             Compact decides: both stacked, or one

A fold outranks width because it is a physical fact about the glass: an unfolded book-style device is medium or expanded \*and\* has a hinge down the middle, and a ratio split that happened to put a button on the crease is the bug foldable support exists to prevent. A horizontal fold is what the tabletop posture looks like — the natural layout there is content above the hinge and controls below, which is rule 2 with First above.

Only a \*separating\* fold is laid out around. A flat, continuous panel (a Galaxy Z Fold opened flat) reports a fold the platform says content may cross, and falls through to rule 3 so an app is not split in half on a screen with nothing in the middle.

#### Lining up with the hinge

Fold bounds are in window coordinates (see core/window.go) and a TwoPane lays out in its own, so the two agree only when the TwoPane starts where the window does. Origin is the pane's top-left corner in window coordinates, for the layouts where it does not: a TwoPane under an 64dp app bar in tabletop posture passes Origin{Y: statusBar + 64}. A TwoPane that fills the window along the split axis — the usual arrangement for a vertical hinge, where nothing sits to its left — needs none.

First is sized to end exactly at the hinge (a fixed width or height) and Second grows to fill what remains, which means the split is exact on the First side and assumes the TwoPane reaches the window's far edge on the Second. A hinge the pane does not actually contain (First would be zero or negative, or the hinge sits past the window) falls through to the ratio rules rather than drawing a pane with no room.

An occluding hinge (a dual-screen seam with real width) becomes a spacer of that width, so nothing is drawn under it. A non-occluding one is a line and adds nothing; each pane's own padding keeps its content off the crease.

#### One hook

TwoPane calls hooks.UseWindow, so it re-renders itself when the device folds, unfolds or bends, with nothing for the caller to wire. That makes the rule Accordion and Snackbar document apply here too: render a TwoPane in a stable position on every pass rather than conditionally.

#### Filling

The container grows (FlexGrow 1) and stretches its panes along the cross axis. A side-by-side split has to fill the height it is given to look like two panes rather than two cards, and a top/bottom split has to fill the height for the hinge arithmetic to mean anything, so filling is the default and Style can undo it.

<small>[comps/two_pane.go:76](https://github.com/rohanthewiz/grmob/blob/master/comps/two_pane.go#L76)</small>

#### func (TwoPane) Render

```go
func (p TwoPane) Render(ctx *core.Context) *core.Node
```

Render reads the window, resolves the arrangement and builds it.

<small>[comps/two_pane.go:228](https://github.com/rohanthewiz/grmob/blob/master/comps/two_pane.go#L228)</small>

### type TwoPaneCompact

```go
type TwoPaneCompact int
```

TwoPaneCompact chooses what a compact window shows.

<small>[comps/two_pane.go:119](https://github.com/rohanthewiz/grmob/blob/master/comps/two_pane.go#L119)</small>

```go
const (
	// TwoPaneStack shows both panes, First above Second. Right for content
	// and controls that both belong on screen.
	TwoPaneStack TwoPaneCompact = iota
	// TwoPaneFirst shows only First — the list in list–detail, where the
	// caller navigates to the detail on a phone.
	TwoPaneFirst
	// TwoPaneSecond shows only Second — the detail, once something is
	// selected.
	TwoPaneSecond
)
```

