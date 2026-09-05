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
	// makes of it, and why nine of the sixteen values do nothing on either
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
	// of the sixteen roles have, documented in GrMobStyle.kt beside the role
	// dispatch rather than left for the next person to rediscover.
	//
	// Out-of-range values are dropped rather than clamped. 0 is the zero value
	// and means "a heading, tier unstated", which is what every heading in
	// every tree was before this field existed; anything above 6 has no
	// spelling on any of the three targets that can express a level, and
	// silently rewriting a 7 to a 6 would invent a structure the caller did
	// not describe.
	AccessibilityHeadingLevel int

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
