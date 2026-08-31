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

func PaddingHorizontal(px int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Padding.Horizontal = px
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
