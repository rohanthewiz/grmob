package core

type Style struct {
	FontSize     float64
	FontWeight   Weight
	TextColor    string
	Background   string
	Padding      EdgeInsets
	Margin       EdgeInsets
	BorderRadius float64
	Shadow       float64
	Align        Alignment
	Display      DisplayMode
	Width        string
	Height       string
	BorderColor  string
	BorderWidth  float64
	Position     Position
	Top          string
	Left         string
	Right        string
	Bottom       string
	ZIndex       int
	Overflow     string // "hidden", "scroll", "visible"
	WhiteSpace   string // "nowrap", "normal", "pre-line"
	LineHeight   int
	MaxWidth     string
	MaxHeight    string
	Gap          float64
	Transition   string // "all 0.3s ease"
	Animation    string // "bounce 2s infinite"

	// Rotate turns the node clockwise by this many degrees about its own
	// centre. It is a paint-time transform on all four targets, not a layout
	// one: the box keeps the size and position it laid out with, and only its
	// pixels are turned. A rotated node therefore never reflows its siblings,
	// and a rotated node whose corners now stick out of its parent is clipped
	// by that parent's Overflow like any other overflowing paint.
	//
	//	CSS       transform: rotate(Ndeg)     about transform-origin: 50% 50%
	//	Compose   Modifier.rotate(N)          about the layout bounds' centre
	//	SwiftUI   .rotationEffect(.degrees(N)) about .center
	//
	// All four agree on the two things that would otherwise need a mapping
	// table: degrees (not radians or turns), and positive meaning clockwise
	// on screen. That agreement is why this is one float and not a Transform
	// type — the moment translate and scale join it, the three platforms stop
	// agreeing on composition order and the type has to say what it means.
	//
	// # Centre only
	//
	// There is no transform-origin. A caller who needs to swing a node about
	// some other point wraps it in a box whose centre is that point, which
	// costs one node and works identically on every target; exposing an
	// origin would cost a second field on every renderer to express the same
	// thing less portably (Compose takes a TransformOrigin fraction, SwiftUI
	// a UnitPoint, CSS a length-or-percentage pair).
	//
	// # Winding, and why nothing here normalises it
	//
	// 370 and 10 look identical and -90 and 270 look identical, and the value
	// is passed through as written rather than folded into [0, 360). Without
	// a Transition the two spellings are indistinguishable, since each frame
	// simply draws where it was told. With one they are not, and the choice
	// belongs to the caller: 350 → 370 sweeps 20 degrees forwards, 350 → 10
	// sweeps 340 degrees back the other way.
	//
	// Normalising here would take that choice away and pick the wrong one for
	// the case this field was added for — an animated compass fed bearings
	// folded into [0, 360) unwinds the whole rose backwards every time the
	// user turns past north. core.AngleDelta (heading.go) is the arithmetic
	// for accumulating an unwrapped angle when a caller wants one.
	Rotate float64

	HoverStyle   *Style
	FocusStyle   *Style
	PseudoStates map[string]Style // ":hover", ":focus"

	FlexDirection  FlexDirection
	JustifyContent JustifyContent
	AlignItems     AlignItems
	MinHeight      string
	MinWidth       string
	ColumnGap      float64
	RowGap         float64
	FlexWrap       string
	AlignSelf      AlignItems
	FlexBasis      string
	FlexShrink     float64
	FlexGrow       float64

	// Accessibility semantics. These live on Style rather than Props so every
	// builder that takes StyleProps — leaves and containers alike — supports
	// them without a signature change, and so the reconciler's value-compared
	// update-style patches carry changes to them like any visual property.
	// Renderers map them onto the platform's semantics layer: contentDescription
	// / clearAndSetSemantics on Android, accessibilityLabel / accessibilityHint /
	// accessibilityHidden on iOS.
	AccessibilityLabel  string
	AccessibilityHint   string
	AccessibilityHidden bool

	// AccessibilityRole is what the node *is* — a heading, a table cell, a
	// search landmark — as opposed to what it is called and what tapping it
	// does. See role.go for the vocabulary, what each of the four renderers
	// makes of it, and why nine of the twenty values do nothing on either
	// native.
	//
	// It sits with the three fields above and travels the same way: on Style
	// rather than in Props, so every builder supports it without a signature
	// change and a change to it patches like any other style property.
	AccessibilityRole Role

	// AccessibilityHeadingLevel is the tier of a heading — 1 for the screen's
	// own name, 2 for a section within it, and so on to 6.
	//
	// It is read only when AccessibilityRole is RoleHeading. A level is a
	// property *of* a heading, and ARIA scopes aria-level the same way (the
	// attribute is defined for heading, listitem and row, and notably not for
	// columnheader, which is why a DataTable's column headers take the role
	// and no level).
	//
	// # Why a field and not RoleHeading2
	//
	// The obvious alternative was six more Role constants. Three things are
	// wrong with it, and the first is decisive: core.Role's values are ARIA's
	// own spellings, which is the entire reason the two DOM renderers need no
	// mapping table (see role.go). There is no `role="heading2"`, so the
	// moment the enum carries one, both web targets need the table the
	// vocabulary was chosen to avoid.
	//
	// The second is that "every renderer names every role" would then cost
	// twelve new arms — six on each native — every one of which would map to
	// the heading primitive the plain `heading` arm already maps to. The
	// coverage checks would be pinning six spellings of one fact.
	//
	// The third is that level and role are genuinely independent questions. A
	// reader asks "what is this" once and "where does it sit" separately, and
	// a caller that wants a heading without committing to a tier — which is
	// every caller that existed before this field — should be able to say so
	// by leaving a field alone rather than by picking from a lettered set.
	//
	// # What each target does with it
	//
	// The web emits aria-level. SwiftUI has accessibilityHeading, whose
	// AccessibilityHeadingLevel is the same 1-6 idea, so the level survives to
	// VoiceOver's heading rotor. Compose's heading() takes no argument and has
	// no level at all, so this is inert on Android — the same honest gap nine
	// of the twenty roles have, documented in GrMobStyle.kt beside the role
	// dispatch rather than left for the next person to rediscover.
	//
	// Out-of-range values are dropped rather than clamped. 0 is the zero value
	// and means "a heading, tier unstated", which is what every heading in
	// every tree was before this field existed; anything above 6 has no
	// spelling on any of the three targets that can express a level, and
	// silently rewriting a 7 to a 6 would invent a structure the caller did
	// not describe.
	AccessibilityHeadingLevel int

	// AccessibilityNestingLevel is how deep an item sits inside a nested
	// collection — 1 for a top-level item, 2 for one inside it, and so on with
	// no ceiling.
	//
	// It is read only when AccessibilityRole is RoleListItem or RoleRow.
	// That is the rest of ARIA's own scoping for aria-level, whose three roles
	// are heading, listitem and row; the heading third is
	// AccessibilityHeadingLevel above, and the two fields are mutually
	// exclusive by construction because a node has exactly one role.
	//
	// # Why a second field rather than a wider first one
	//
	// Both become the same attribute, so one field named AccessibilityLevel
	// reading all three roles was the obvious alternative. What it cannot
	// carry is that the two levels are validated differently, and not by
	// accident:
	//
	//	heading   1-6      HTML has h1-h6 and SwiftUI's
	//	                   AccessibilityHeadingLevel has .h1-.h6; a 7 has no
	//	                   spelling on any target that can express a tier
	//	nesting   1 and up ARIA requires only "an integer greater than or
	//	                   equal to 1", and a deeply nested tree is not
	//	                   malformed at depth 7
	//
	// One field would need one rule, and either rule is wrong for the other
	// half: capping nesting at 6 would flatten a legitimate tree, and lifting
	// the heading cap would export an aria-level no target can honor. The
	// names are the other half of it — a caller reaching for "the level" on a
	// list item should not have to read a doc to learn that the field is
	// spelled for headings.
	//
	// # What each target does with it
	//
	// The web emits aria-level. Neither native does anything: SwiftUI has no
	// nesting-depth property at all, and Compose's nearest thing —
	// collectionItemInfo — describes an item's index and span within one
	// collection rather than its depth within nested ones, so mapping onto it
	// would state something the field does not mean. This is the same honest
	// gap nine of the twenty roles have, and it is written down in
	// GrMobStyle.kt and GrMobStyle.swift beside the role dispatch rather than
	// left for the next person to rediscover.
	//
	// Values below 1 are dropped, as they are for a heading: 0 is the zero
	// value and means "an item, depth unstated", which is what every list item
	// in every tree is unless something says otherwise.
	AccessibilityNestingLevel int

	// AccessibilitySelected is whether this control is *on* — the applied
	// filter chip, the tab that is showing, the chosen calendar day. See
	// SelectedState for the vocabulary and for why "off" and "not selectable"
	// are two values rather than one bool.
	//
	// # One field, two attributes — the mirror of the level pair
	//
	// AccessibilityHeadingLevel and AccessibilityNestingLevel are two fields
	// that become one attribute, resolved by a switch on the role. This is
	// the same problem reflected: one field that becomes two attributes,
	// resolved by the same switch, in the same two functions.
	//
	//	role                         attribute
	//	-------------------------    -------------------------------------
	//	tab, row, columnheader       aria-selected
	//	button (or a core.Button)    aria-pressed
	//	anything else                nothing at all
	//
	// ARIA has two words because it draws a real distinction. Selection is
	// *one of these*: a tab among tabs, a row among rows, and choosing one
	// unchooses the rest. Pressed is *this one, on or off*: a toggle that
	// answers only for itself. A filter chip is pressed; a tab is selected;
	// and a widget that says the wrong one is announced as a member of a set
	// that does not exist.
	//
	// The natives have one spelling each and so need no switch: Compose sets
	// its `selected` semantics property, SwiftUI adds `.isSelected`. That
	// asymmetry is why the switch lives in the two web exporters rather than
	// in core — the field means one thing, and only ARIA needs to know which
	// word to say it with.
	//
	// # The role guard is ARIA's, not this framework's
	//
	// aria-selected on a plain container is dropped by screen readers, for
	// the same reason an accessible name on one is: neither attribute is
	// defined for a generic element. So a widget states the role alongside the
	// state, and the two web exporters write nothing when it has not.
	//
	// The name half of that pair is no longer the author's problem — the two
	// web exporters supply RoleGroup to a named container that has no role of
	// its own, which makes the name legal without claiming anything (see
	// core.RoleGroup). A state gets no such rescue, and the asymmetry is the
	// point rather than an omission. `group` is a role that fits any
	// container, so supplying it invents nothing; there is no role that
	// carries a selection and fits any container — aria-selected is scoped to
	// option, tab, row and columnheader, and picking one of those for a node
	// would be deciding what the node is. A name is a fact about the node the
	// author already stated; a role is not.
	//
	// The one case that needs no role is a core.Button, whose node type
	// already is one — the "roles a node type carries for itself" rule in
	// role.go. Both web exporters read the node type beside the role for
	// exactly that reason, as they already do for a Modal's dialog.
	//
	// # Neither native scopes it, and that is not a divergence to fix
	//
	// Compose will set `selected` on any node and VoiceOver will honour
	// `.isSelected` on any view, so a state on an unroled node reaches both
	// natives and neither web target. The web is the strict one because ARIA
	// is; each platform says the truest thing it can, which is the same rule
	// the nine unmapped roles follow.
	AccessibilitySelected SelectedState

	// AccessibilityExpanded is whether this disclosure is *open* — the
	// accordion section showing its body, the twisty that has been turned. See
	// ExpandedState for the vocabulary, for why it is a separate type from
	// SelectedState, and for why "closed" and "not a disclosure" are two
	// values rather than one bool.
	//
	// # One field, one attribute, and a role guard that is not the same one
	//
	// This is the simplest of the three accessibility state fields on the web
	// — it becomes aria-expanded and nothing else, where a level resolves two
	// fields onto one attribute and a selection resolves one field onto two.
	// What it does *not* share with the selection is the role list, and the
	// difference is the whole of the guard:
	//
	//	                aria-selected / -pressed   aria-expanded
	//	button          aria-pressed               yes
	//	tab             aria-selected              yes
	//	row             aria-selected              yes
	//	columnheader    aria-selected              yes
	//	option          aria-selected              no
	//	link            no                         yes
	//	listbox         no                         yes
	//
	// Both lists are ARIA's own scoping rather than a shortlist of what seemed
	// useful, and the two disagree at both ends. So the two fields cannot
	// share a guard even though they look like they should, which is one of
	// the two reasons ExpandedState is a type of its own.
	//
	// A core.Button needs no role beside it, on the rule that gives a Modal
	// its dialog role and a Chip its aria-pressed: the node type already is a
	// button. That is not an optimisation here either — ARIA's own disclosure
	// pattern *is* a button, so the node type that most wants this attribute
	// would otherwise be the one that could not carry it.
	//
	// Anything else writes nothing. An unroled container is `generic`, ARIA
	// does not define aria-expanded there, and a reader drops it. This gets no
	// RoleGroup-shaped rescue for the reason a selection does not: `group` is
	// not among the roles above, so there is no role that both fits any
	// container and carries a disclosure. components.Accordion is what happens
	// when a widget takes that seriously — its header row is a button inside a
	// heading, which is ARIA's own accordion shape, rather than a named div
	// with a state a browser throws away.
	//
	// # The near miss: a control that opens a *dialog* is not expanded
	//
	// aria-expanded says the content is here, in the page, and can be shown or
	// hidden. A trigger that opens a modal is a different relationship —
	// ARIA spells that aria-haspopup, which this vocabulary does not carry —
	// so components.DatePicker's trigger, which looks exactly like a
	// disclosure and even flips a glyph, deliberately sets nothing.
	//
	// # One native maps it and one cannot, which is the reverse of usual
	//
	// Compose has expand()/collapse() semantics actions, so a collapsed
	// disclosure offers TalkBack an "expand" action and an open one offers
	// "collapse". They are actions rather than a state, which means they need
	// something to perform: Renderer.kt wires them to the node's own click
	// callback, and a node with a state but no handler gets neither. That is
	// the honest shape — an expand action nothing can perform is worse than
	// none.
	//
	// SwiftUI has nothing. There is no expanded trait, and its own
	// DisclosureGroup announces the state by writing a localized accessibility
	// *value* — a string SwiftUI supplies and this framework has no channel
	// for. Emitting an English "expanded" from the renderer would be the same
	// move components.Chip's ", selected" name suffix was deleted for. So the
	// key crosses the bridge, is deliberately not parsed, and the note in
	// GrMobStyle.swift says which property it is turning down.
	AccessibilityExpanded ExpandedState

	// AccessibilityID names this element so another one can point at it, and
	// AccessibilityControls is the pointing. Both are web-only, and they are
	// the vocabulary's one pair of *references* rather than values.
	//
	// # The rule that keeps this from becoming a second ARIA
	//
	// Half of ARIA is IDREF-shaped — aria-labelledby, aria-describedby,
	// aria-controls, aria-owns, aria-activedescendant — and a framework that
	// added all of them would be asking every app author to mint and track
	// document-global ids for things it already has a shorter way to say. The
	// line drawn here:
	//
	//	a reference prop earns its place only when what it points at cannot
	//	be said as a value.
	//
	// aria-labelledby points at *text*, and AccessibilityLabel already carries
	// text. aria-describedby points at text, and AccessibilityHint already
	// carries text (as aria-description, which is that idea in value form —
	// see accessibilityAttrs in htmlout/export.go). Neither reference buys an
	// author anything except a saved copy of a string they are holding.
	//
	// aria-controls is different in kind: what it points at is *another
	// element*, and there is no string that can stand in for one. That is why
	// this pair exists and the other three do not.
	//
	// # What asked for it
	//
	// A tab strip built by hand. core.TabView mints its own ids and writes the
	// whole tab/panel wiring from the node type (see htmlout/tabview.go), so
	// the wired case needed nothing; a strip assembled out of chips or buttons
	// — which is how the social example's bottom bar is built, and how anyone
	// who wants a different-looking strip has to build one — could say
	// role="tab" and role="tablist" and then had no way at all to say which
	// region each tab shows. A reader that cannot follow that relationship
	// announces three tabs controlling nothing.
	//
	//	// the strip
	//	core.Row(core.AccessibilityRole(core.RoleTabList),
	//	    Chip{Label: "Home", Style: []core.StyleProp{
	//	        core.AccessibilityRole(core.RoleTab),
	//	        core.AccessibilitySelected(core.SelectedWhen(tab == "home")),
	//	        core.AccessibilityID("home-tab"),
	//	        core.AccessibilityControls("app-panel"),
	//	    }},
	//	)
	//	// the region it switches
	//	core.Box(core.AccessibilityID("app-panel"), core.AccessibilityLabel("Home"), …)
	//
	// # Ids are the author's to keep unique, with one prefix reserved
	//
	// An id is document-global and this framework does not rewrite the string,
	// so two elements given the same AccessibilityID are two elements with the
	// same id — invalid HTML, and a reference that resolves to whichever the
	// browser saw first. That is the author's to avoid, exactly as it is in
	// hand-written HTML.
	//
	// The one reservation is the "grmob-" prefix, which is where core.TabView's
	// own minted ids live (tabScope in htmlout/tabview.go and its twin in
	// grmob-runtime.js). An author id colliding with one of those would break a
	// TabView's wiring rather than their own.
	//
	// A TabView page that carries an AccessibilityID of its own is left
	// unwired, on the same rule an authored role follows there: the author has
	// claimed the slot, and the wiring does not take it back.
	//
	// # Neither native has an equivalent, and neither is given one
	//
	// There is no relationship of this kind in SwiftUI's or Compose's semantics
	// vocabulary — a reader on either phone navigates a tab strip by swiping to
	// the next element, not by following a reference — so both keys cross the
	// bridge and are deliberately not parsed. Not parsed rather than parsed and
	// ignored, for the reason AccessibilityNestingLevel is: a field silently
	// dropped inside a renderer is indistinguishable from one nobody had heard
	// of. mobile/verify/idref_test.go pins that.
	//
	// The near miss worth naming is accessibilityIdentifier (iOS) and testTag
	// (Compose). Both are element identities and neither is an accessibility
	// relationship: they are what a UI test selects by, they are not exposed to
	// VoiceOver or TalkBack, and filling them from an ARIA wiring string would
	// silently make every hand-built tab a test selector.
	AccessibilityID       string
	AccessibilityControls string

	// Disabled marks the node inert: the renderers hand it to the platform's
	// own disabled state rather than emulating one, so the control stops
	// accepting input, loses focus eligibility, and — the part an emulation
	// cannot buy — announces itself as disabled to the screen reader
	// (Compose's `enabled = false`, SwiftUI's `.disabled(true)`, the HTML
	// `disabled` attribute).
	//
	// It lives on Style for the same reason the accessibility fields do:
	// every builder already takes StyleProps, so Button, the inputs, the
	// checkbox and any tappable container support it without a signature
	// change, and a change to it patches like any other style property.
	//
	// Disabling is *not* the same as dropping the handler. Go must keep
	// registering the callback (a nil handler in the registry panics when a
	// native tap races the patch that disabled the control), and the
	// renderers must additionally refuse to dispatch — a platform disabled
	// state already does that, which is what closes the race properly.
	//
	// Visual muting is deliberately not implied. What "disabled" looks like
	// is a palette decision (components.Button spends Surface/TextSecondary
	// on it); what it *means* is this flag.
	Disabled bool
}

type Weight int

const (
	Light  Weight = 200
	Normal Weight = 400
	Bold   Weight = 700
)

type EdgeInsets struct {
	Top, Right, Bottom, Left int
	Horizontal, Vertical     int
}
type StyleProp interface {
	Apply(*Style)
}
type styleFunc func(*Style)

func (f styleFunc) Apply(s *Style) {
	f(s)
}

// UseStyle turns a whole Style value into a StyleProp, so a caller can pass a
// named visual role ("the card surface", "the theme's Body typography") in one
// argument instead of unpacking it into a dozen individual props.
//
// The merge rule is "a set field wins, an unset field is ignored": every field
// of s that holds a non-zero value overwrites the target's, and every field
// left at its zero value leaves the target's alone. That is what makes
// UseStyle composable — layering role styles onto a theme's component defaults
// only ever adds, never blanks out what the theme supplied.
//
// The rule's one unavoidable edge is that a zero value is indistinguishable
// from "not set", so UseStyle cannot *clear* a field the target already has:
// Style{AccessibilityHidden: false} does not un-hide an element, and
// Style{FontSize: 0} does not reset a font size. Use the individual StyleProp
// setters (AccessibilityHidden(), FontSize(0)) when the intent is to force a
// value rather than to layer one.
//
// This merges every field of Style. It previously covered only fourteen of
// them, which meant Width, Height, the whole flex group, and the accessibility
// fields were silently dropped — a style value carrying them applied cleanly
// and did nothing. Any field added to Style must be added here too;
// TestUseStyleMergesEveryField walks the struct reflectively and fails if one
// is missed.
func UseStyle(s Style) StyleProp {
	return styleFunc(s.applyTo)
}

// applyTo is the merge itself, split out from UseStyle so the nested style
// fields (HoverStyle, FocusStyle, PseudoStates entries) can recurse into it
// and get the same field-by-field semantics as the top level.
func (s Style) applyTo(target *Style) {
	// Typography and color.
	if s.FontSize != 0 {
		target.FontSize = s.FontSize
	}
	if s.FontWeight != 0 {
		target.FontWeight = s.FontWeight
	}
	if s.TextColor != "" {
		target.TextColor = s.TextColor
	}
	if s.Background != "" {
		target.Background = s.Background
	}
	if s.LineHeight != 0 {
		target.LineHeight = s.LineHeight
	}
	if s.WhiteSpace != "" {
		target.WhiteSpace = s.WhiteSpace
	}
	if s.Align != "" {
		target.Align = s.Align
	}

	// Box model. EdgeInsets is a comparable struct, so the whole inset set is
	// one field: a partially-filled EdgeInsets replaces the target's outright
	// rather than merging edge by edge. Layering one edge onto another is what
	// the PaddingTop / PaddingHorizontal props are for.
	if s.Padding != (EdgeInsets{}) {
		target.Padding = s.Padding
	}
	if s.Margin != (EdgeInsets{}) {
		target.Margin = s.Margin
	}
	if s.Width != "" {
		target.Width = s.Width
	}
	if s.Height != "" {
		target.Height = s.Height
	}
	if s.MinWidth != "" {
		target.MinWidth = s.MinWidth
	}
	if s.MinHeight != "" {
		target.MinHeight = s.MinHeight
	}
	if s.MaxWidth != "" {
		target.MaxWidth = s.MaxWidth
	}
	if s.MaxHeight != "" {
		target.MaxHeight = s.MaxHeight
	}
	if s.Overflow != "" {
		target.Overflow = s.Overflow
	}

	// Borders, corners, elevation.
	if s.BorderRadius != 0 {
		target.BorderRadius = s.BorderRadius
	}
	if s.BorderColor != "" {
		target.BorderColor = s.BorderColor
	}
	if s.BorderWidth != 0 {
		target.BorderWidth = s.BorderWidth
	}
	if s.Shadow != 0 {
		target.Shadow = s.Shadow
	}
	// Rotate merges on the same "non-zero wins" rule as the rest, which means
	// a role style cannot merge a node back to zero degrees. That is the
	// documented UseStyle edge, not a special case here: use the Rotate()
	// prop to force an angle, including 0.
	if s.Rotate != 0 {
		target.Rotate = s.Rotate
	}

	// Positioning. Top was the odd one out before: Bottom, Left and Right were
	// merged and Top was not, so an absolutely-positioned style value lost its
	// top offset alone.
	if s.Position != "" {
		target.Position = s.Position
	}
	if s.Top != "" {
		target.Top = s.Top
	}
	if s.Right != "" {
		target.Right = s.Right
	}
	if s.Bottom != "" {
		target.Bottom = s.Bottom
	}
	if s.Left != "" {
		target.Left = s.Left
	}
	if s.ZIndex != 0 {
		target.ZIndex = s.ZIndex
	}

	// Flex container and item properties.
	if s.Display != "" {
		target.Display = s.Display
	}
	if s.FlexDirection != "" {
		target.FlexDirection = s.FlexDirection
	}
	if s.JustifyContent != "" {
		target.JustifyContent = s.JustifyContent
	}
	if s.AlignItems != "" {
		target.AlignItems = s.AlignItems
	}
	if s.AlignSelf != "" {
		target.AlignSelf = s.AlignSelf
	}
	if s.FlexWrap != "" {
		target.FlexWrap = s.FlexWrap
	}
	if s.FlexBasis != "" {
		target.FlexBasis = s.FlexBasis
	}
	if s.FlexGrow != 0 {
		target.FlexGrow = s.FlexGrow
	}
	if s.FlexShrink != 0 {
		target.FlexShrink = s.FlexShrink
	}
	if s.Gap != 0 {
		target.Gap = s.Gap
	}
	if s.RowGap != 0 {
		target.RowGap = s.RowGap
	}
	if s.ColumnGap != 0 {
		target.ColumnGap = s.ColumnGap
	}

	// Motion.
	if s.Transition != "" {
		target.Transition = s.Transition
	}
	if s.Animation != "" {
		target.Animation = s.Animation
	}

	// Accessibility semantics.
	if s.AccessibilityLabel != "" {
		target.AccessibilityLabel = s.AccessibilityLabel
	}
	if s.AccessibilityHint != "" {
		target.AccessibilityHint = s.AccessibilityHint
	}
	if s.AccessibilityHidden {
		target.AccessibilityHidden = true
	}
	if s.AccessibilityRole != RoleNone {
		target.AccessibilityRole = s.AccessibilityRole
	}
	// Merged independently of the role, not alongside it: a theme's Style can
	// carry the tier a widget's own Style then names the role for, and the
	// pair only has to agree at export time.
	if s.AccessibilityHeadingLevel != 0 {
		target.AccessibilityHeadingLevel = s.AccessibilityHeadingLevel
	}
	// Independently of the role and of each other. The two levels are never
	// both read — a node has one role, and the two fields answer to disjoint
	// sets of them — but merging is not the layer that knows that, and a
	// merge that dropped one because the other was set would make the result
	// depend on which Style in the chain happened to name the role.
	if s.AccessibilityNestingLevel != 0 {
		target.AccessibilityNestingLevel = s.AccessibilityNestingLevel
	}
	// Independently of the role for the same reason, and note that
	// SelectedOff is a *stated* value and so merges: only SelectedUnset, the
	// zero value, leaves the target alone. A widget layering an unselected
	// state over a selected one has to be able to turn it off, which is the
	// whole of what a strip does when the selection moves.
	if s.AccessibilityID != "" {
		target.AccessibilityID = s.AccessibilityID
	}
	if s.AccessibilityControls != "" {
		target.AccessibilityControls = s.AccessibilityControls
	}
	if s.AccessibilitySelected != SelectedUnset {
		target.AccessibilitySelected = s.AccessibilitySelected
	}
	// And on the same terms: ExpandedClosed is a stated value and merges, so
	// that a layer describing a shut disclosure can close one a layer below it
	// had opened. Only ExpandedUnset leaves the target alone.
	if s.AccessibilityExpanded != ExpandedUnset {
		target.AccessibilityExpanded = s.AccessibilityExpanded
	}
	if s.Disabled {
		target.Disabled = true
	}

	// Nested styles. These three are reference types, and a Style value gets
	// copied by assignment all over the framework — containerNode starts from
	// `style := &base` where base is a *copy of the theme's* Style, which
	// shares the theme's pointer and map. Writing through either would reach
	// back into the theme (or into the shared closure a package-level
	// StyleProp holds) and corrupt every later render. So both branches below
	// build fresh values and never store s's own pointer or map.
	if s.HoverStyle != nil {
		target.HoverStyle = mergedStylePtr(target.HoverStyle, *s.HoverStyle)
	}
	if s.FocusStyle != nil {
		target.FocusStyle = mergedStylePtr(target.FocusStyle, *s.FocusStyle)
	}
	if len(s.PseudoStates) > 0 {
		// Merged per key rather than replaced wholesale: each entry is an
		// independent state, so a style value that only describes ":hover"
		// must not delete a ":focus" the target already carries.
		merged := make(map[string]Style, len(target.PseudoStates)+len(s.PseudoStates))
		for k, v := range target.PseudoStates {
			merged[k] = v
		}
		for k, v := range s.PseudoStates {
			if base, ok := merged[k]; ok {
				v.applyTo(&base)
				merged[k] = base
			} else {
				merged[k] = v
			}
		}
		target.PseudoStates = merged
	}
}

// mergedStylePtr merges src onto a copy of *target (or onto the zero Style
// when target is nil) and returns a pointer to that copy. Always allocating
// keeps the result unaliased from both operands.
func mergedStylePtr(target *Style, src Style) *Style {
	var merged Style
	if target != nil {
		merged = *target
	}
	src.applyTo(&merged)
	return &merged
}
func PrimaryColor() string { return "#007AFF" }
func DangerColor() string  { return "#FF3B30" }
func RoundedShadowBox() StyleProp {
	return UseStyle(Style{
		BorderRadius: 12,
		Shadow:       2,
		Background:   "#FFFFFF",
	})
}

var TextInputStyle = UseStyle(Style{
	FontSize:     16,
	TextColor:    "#000000",
	Background:   "#FFFFFF",
	Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 12, Right: 12},
	BorderRadius: 8,
	Shadow:       1,
})

func PaddingTop(px int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Padding.Top = px
	})
}

// PaddingHorizontal sets the left and right insets.
//
// It writes the explicit Left/Right sides as well as the Horizontal shorthand.
// The renderers resolve a side as "the explicit value if non-zero, otherwise
// the axis shorthand" (see htmlout.EdgeCSS), so a prop that wrote only the
// shorthand could never override a side that was already set: a theme Column
// carries Left/Right 16, and PaddingHorizontal(0) after it used to leave the
// 16 in place — and PaddingHorizontal(24) used to render as 16. Writing the
// sides too gives this prop the same last-one-wins ordering every other
// StyleProp has, and a zero clears the theme value in all four renderers
// without any of them changing their resolution rule.
func PaddingHorizontal(px int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Padding.Horizontal = px
		s.Padding.Left = px
		s.Padding.Right = px
	})
}

type ResponsiveStyle map[string]Style // "mobile", "tablet", "desktop"

type Alignment string

const (
	AlignStart    Alignment = "start"
	AlignCenter   Alignment = "center"
	AlignEnd      Alignment = "end"
	AlignStretch  Alignment = "stretch"
	AlignBaseline Alignment = "baseline"
	AlignJustify  Alignment = "justify"
)

type DisplayMode string

const (
	DisplayVisible DisplayMode = "visible"
	DisplayHidden  DisplayMode = "hidden"
	DisplayNone    DisplayMode = "none"
	DisplayInline  DisplayMode = "inline"
	DisplayBlock   DisplayMode = "block"
)

type JustifyContent string
type FlexDirection string
type AlignItems string

const (
	JustifyStart   JustifyContent = "flex-start"
	JustifyCenter  JustifyContent = "center"
	JustifyEnd     JustifyContent = "flex-end"
	JustifyBetween JustifyContent = "space-between"
	JustifyAround  JustifyContent = "space-around"
	JustifyEvenly  JustifyContent = "space-evenly"

	AlignItemsStart   AlignItems = "flex-start"
	AlignItemsCenter  AlignItems = "center"
	AlignItemsEnd     AlignItems = "flex-end"
	AlignItemsStretch AlignItems = "stretch"

	FlexRow     FlexDirection = "row"
	FlexColumn  FlexDirection = "column"
	DisplayFlex               = "flex"
)

type Position string

const (
	PositionRelative Position = "relative"
	PositionAbsolute Position = "absolute"
	PositionFixed    Position = "fixed"
	PositionSticky   Position = "sticky"
)

//func Responsive(breakpoint string, style Style) StyleProp {
//	return styleFunc(func(s *Style) {
//		if s.Responsive == nil {
//			s.Responsive = make(ResponsiveStyle)
//		}
//		s.Responsive[breakpoint] = style
//	})
//}
