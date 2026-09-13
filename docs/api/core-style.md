# Package core — Styling: the Style struct

```go
import "github.com/rohanthewiz/grmob/core"
```

Style and the value types its fields take: alignment, flex, position, weights and edge insets.

One of 11 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/style.go`.

## Index

- [Constants](#constants) — `AlignItemsCenter`, `AlignItemsEnd`, `AlignItemsStart`, `AlignItemsStretch`, `DisplayFlex`, `FlexColumn`, `FlexRow`, `JustifyAround`, `JustifyBetween`, `JustifyCenter`, `JustifyEnd`, `JustifyEvenly`, and 2 more
- [Variables](#variables) — `TextInputStyle`
- [`func DangerColor`](#func-dangercolor)
- [`func PrimaryColor`](#func-primarycolor)
- [`type AlignItems`](#type-alignitems)
    - [`func (AlignItems) Apply`](#func-alignitems-apply)
- [`type Alignment`](#type-alignment)
- [`type DisplayMode`](#type-displaymode)
- [`type EdgeInsets`](#type-edgeinsets)
- [`type FlexDirection`](#type-flexdirection)
    - [`func (FlexDirection) Apply`](#func-flexdirection-apply)
- [`type JustifyContent`](#type-justifycontent)
    - [`func (JustifyContent) Apply`](#func-justifycontent-apply)
- [`type Position`](#type-position)
- [`type ResponsiveStyle`](#type-responsivestyle)
- [`type Style`](#type-style)
    - [`func (Style) ShrinkFactor`](#func-style-shrinkfactor)
    - [`func (Style) With`](#func-style-with)
- [`type StyleProp`](#type-styleprop)
    - [`func PaddingHorizontal`](#func-paddinghorizontal)
    - [`func RoundedShadowBox`](#func-roundedshadowbox)
    - [`func UseStyle`](#func-usestyle)
- [`type Weight`](#type-weight)

## Constants

```go
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
```

<small>[core/style.go:1283](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1283)</small>

ShrinkNone is what core.FlexShrink(0) stores, and what every renderer must read as a shrink factor of zero.

#### Why a sentinel

Every optional number in Style means "unset" by being zero: Style.Merge copies a field only when it is non-zero, so a component default's Gap survives a caller who did not mention one. That trade is right for every other number here, because their CSS initial value IS zero — an unset Gap and a Gap of 0 lay out identically, so nothing is lost by conflating them.

flex-shrink is the exception. Its CSS initial value is 1, so zero and unset are two different layouts:

	unset       the item shrinks under pressure, in proportion to its base
	zero        the item keeps its size and the container overflows

So core.FlexShrink(0) wrote a zero that Style.Merge read as "nothing was set", htmlout.Export read as "write no declaration", and the WASM runtime read as the empty string — three independent guards, all spelled \`FlexShrink != 0\`, all correct for every other field and all wrong for this one. "Do not shrink this item" was unexpressible, and it failed silently: the prop compiled, applied, serialised and did nothing.

It was found by a break-test that could not break. Mutating a fixture's FlexShrink from 1 to 0 changed no pixel on any target, which is how a declaration nobody can write announces itself.

#### Why -1

CSS forbids a negative flex-shrink — the property's grammar is \<number \[0,∞]> — so no renderer can ever be handed one legitimately, and no author can write one by accident: core.FlexShrink is the only way into the field and it maps 0 here. That makes the sentinel unambiguous in the one place ambiguity would cost the most, which is the JSON that crosses into three other runtimes: core.Style has no field tags, so every renderer sees the number as written and needs exactly one rule to read it.

#### What each target does with it

	htmlout        writes flex-shrink:0
	WASM runtime   writes flexShrink "0"
	SwiftUI        GrMobFlexSolver takes a per-item shrink factor and gives a
	               zero one none of the deficit
	Compose        measures the child with an unbounded main axis and reports
	               its own size, so the Row overflows around it

The Compose arm is the one that needed an argument, and it is worth having here because it is also the limit of what the field means on that target. A Compose Row has no proportional shrink at all — an unweighted child is measured against whatever main-axis space the ones before it did not take — so there is no factor for a FRACTIONAL flex-shrink to be, and Android ignores one. Zero is not a proportion but a refusal, and a refusal is expressible: Modifier.pinMainAxis in Renderer.kt is that, and it is why core.FlexShrink(0) is a declaration that means the same thing on all four targets while core.FlexShrink(0.5) means something on three.

"The same thing on all four targets" is measured rather than argued: internal/pinfixture carries one overflowing Row with the pin in each position and a control with none, and ios/verify/pin.swift solves it through GrMobFlexSolver against a transcription of Compose's own measure loop. The pinned child keeps its base on both; its SIBLINGS do not agree, and that divergence is asserted too. See docs/platforms/native.md.

```go
const ShrinkNone = -1
```

<small>[core/style.go:1382](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1382)</small>

## Variables

```go
var TextInputStyle = UseStyle(Style{
	FontSize:     16,
	TextColor:    "#000000",
	Background:   "#FFFFFF",
	Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 12, Right: 12},
	BorderRadius: 8,
	Shadow:       1,
})
```

<small>[core/style.go:1228](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1228)</small>

## Functions

### func DangerColor

```go
func DangerColor() string
```

<small>[core/style.go:1219](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1219)</small>

### func PrimaryColor

```go
func PrimaryColor() string
```

PrimaryColor and DangerColor are the theme-blind convenience accessors that predate Context.Theme(). They answer for the \*default\* theme's roles and nothing else, so a screen under WithTheme still gets the default palette's hexes from them — which is why nothing in components or core calls either, and why new code should read ctx.Theme().Colors instead.

They read DefaultTheme rather than repeating its literals. Both used to be hard-coded, and the copy was not free: when Colors.Primary moved to Apple's accessible blue (white over systemBlue was 4.02:1, under WCAG AA, and the theme's own Button base declares white), this function kept the old hex — so examples/chat, its one caller, went on painting white on a fill nobody could read it on, in the one place the fix could not reach.

<small>[core/style.go:1218](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1218)</small>

## Types

### type AlignItems

```go
type AlignItems string
```

<small>[core/style.go:1281](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1281)</small>

#### func (AlignItems) Apply

```go
func (a AlignItems) Apply(s *Style)
```

The three named flex-container types are StyleProps in their own right, so the spelling that mirrors core.Align(...) and core.Justify(...) works too:

	core.Column(core.AlignItems(core.AlignItemsCenter), ...)

Without these methods that expression is a type conversion producing a bare string value, which containerNode's PropsAndChildren dispatch cannot recognize and drops — silently outside debug mode. It was the most natural thing to write and it compiled, so an app shipped with every one of its AlignItems lost and its columns left-packed on both natives. Making the value itself apply removes the trap rather than documenting it.

<small>[core/style_props.go:326](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L326)</small>

### type Alignment

```go
type Alignment string
```

<small>[core/style.go:1258](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1258)</small>

```go
const (
	AlignStart    Alignment = "start"
	AlignCenter   Alignment = "center"
	AlignEnd      Alignment = "end"
	AlignStretch  Alignment = "stretch"
	AlignBaseline Alignment = "baseline"
	AlignJustify  Alignment = "justify"
)
```

### type DisplayMode

```go
type DisplayMode string
```

<small>[core/style.go:1269](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1269)</small>

```go
const (
	DisplayVisible DisplayMode = "visible"
	DisplayHidden  DisplayMode = "hidden"
	DisplayNone    DisplayMode = "none"
	DisplayInline  DisplayMode = "inline"
	DisplayBlock   DisplayMode = "block"
)
```

### type EdgeInsets

```go
type EdgeInsets struct {
	Top    int `json:",omitzero"`
	Right  int `json:",omitzero"`
	Bottom int `json:",omitzero"`
	Left   int `json:",omitzero"`

	// Horizontal and Vertical stand in for the pair of sides on their axis,
	// and are read only where that side is zero. They are the two fields the
	// tags above were worth adding for: the DSL writes both a side and its
	// axis (PaddingHorizontal sets Left, Right *and* Horizontal, so a later
	// PaddingLeft(0) can settle it), which leaves the axis fields zero on
	// every inset a side prop built.
	Horizontal int `json:",omitzero"`
	Vertical   int `json:",omitzero"`
}
```

EdgeInsets is a box's inset on each of its four sides, plus the two axis shorthands the PaddingHorizontal / PaddingVertical props write.

#### Why every field is \`omitzero\`, and why the argument is not Style's

Style's fields carry these tags because a zero field tells a renderer nothing it does not already assume (see the note on Style). That argument is about a \*default\*. This struct's is stronger and older: all four renderers resolve a side by asking whether it is non-zero, and take the axis shorthand when it is not —

	top = Top != 0 ? Top : Vertical        htmlout.edgeSide
	                                       GrMobStyle.kt   parseEdges
	                                       GrMobStyle.swift parseEdges
	                                       grmob-runtime.js edgeToCSS

— so a zero side is already \*defined\* to mean "unset, use the axis". A field whose zero means "I said nothing" is precisely a field that can be left off the wire, and every one of the four reads a missing key back as 0 (optInt(name, 0), (obj\[key] as? NSNumber)?.intValue ?? 0, \`explicit || 0\`). htmlout takes the Go value directly and never sees JSON at all.

This is lossy in exactly the way it has always been lossy — a hand-built {Horizontal: 16, Left: 0} cannot ask for a real zero left inset, which htmlout/edges.go documents at length — and the tags neither widen nor narrow that. They only stop writing the fields that were already saying nothing.

#### What it was costing

Six untagged ints wrote all six every time. On the tutorial's contents screen, 77 insets (68 Padding, 9 Margin) crossed the bridge and the axis pair was zero in every one of them, because the DSL's side props settle the shorthand into the sides before writing (core/padding\_sides.go) — so the two fields that exist to be a shorthand were, on this screen, pure overhead:

	"Horizontal":0   77 × 15 bytes   1,155
	"Vertical":0     77 × 13 bytes   1,001
	                                 ─────
	                                 2,156 bytes, 4.0% of the screen

Small next to the 370KB the Style-level tags took off, and free in a way that one was not: no renderer changed, because none of them could tell the difference.

<small>[core/style.go:848](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L848)</small>

### type FlexDirection

```go
type FlexDirection string
```

<small>[core/style.go:1280](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1280)</small>

#### func (FlexDirection) Apply

```go
func (d FlexDirection) Apply(s *Style)
```

<small>[core/style_props.go:328](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L328)</small>

### type JustifyContent

```go
type JustifyContent string
```

<small>[core/style.go:1279](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1279)</small>

#### func (JustifyContent) Apply

```go
func (j JustifyContent) Apply(s *Style)
```

<small>[core/style_props.go:327](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L327)</small>

### type Position

```go
type Position string
```

<small>[core/style.go:1301](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1301)</small>

```go
const (
	PositionRelative Position = "relative"
	PositionAbsolute Position = "absolute"
	PositionFixed    Position = "fixed"
	PositionSticky   Position = "sticky"
)
```

### type ResponsiveStyle

```go
type ResponsiveStyle map[string]Style
```

<small>[core/style.go:1256](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1256)</small>

### type Style

```go
type Style struct {
	FontSize     float64     `json:",omitzero"`
	FontWeight   Weight      `json:",omitzero"`
	TextColor    string      `json:",omitzero"`
	Background   string      `json:",omitzero"`
	Padding      EdgeInsets  `json:",omitzero"`
	Margin       EdgeInsets  `json:",omitzero"`
	BorderRadius float64     `json:",omitzero"`
	Shadow       float64     `json:",omitzero"`
	Align        Alignment   `json:",omitzero"`
	Display      DisplayMode `json:",omitzero"`
	Width        string      `json:",omitzero"`
	Height       string      `json:",omitzero"`
	BorderColor  string      `json:",omitzero"`
	BorderWidth  float64     `json:",omitzero"`
	Position     Position    `json:",omitzero"`
	Top          string      `json:",omitzero"`
	Left         string      `json:",omitzero"`
	Right        string      `json:",omitzero"`
	Bottom       string      `json:",omitzero"`
	ZIndex       int         `json:",omitzero"`
	Overflow     string      `json:",omitzero"` // "hidden", "scroll", "visible"
	WhiteSpace   string      `json:",omitzero"` // "nowrap", "normal", "pre-line"
	LineHeight   int         `json:",omitzero"`
	MaxWidth     string      `json:",omitzero"`
	MaxHeight    string      `json:",omitzero"`
	Gap          float64     `json:",omitzero"`
	Transition   string      `json:",omitzero"` // "all 0.3s ease"
	Animation    string      `json:",omitzero"` // "bounce 2s infinite"

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
	// type — a single matrix field would have to say what order its parts
	// compose in, and the three platforms do not agree on one. TranslateX/Y
	// are separate fields for that reason, and every target applies them
	// outside the rotation (CSS's individual-property order: translate, then
	// rotate), so a turned box slides along the screen's axes, not its own.
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
	Rotate float64 `json:",omitzero"`

	// Spin is a continuous rotation: one revolution every Spin milliseconds,
	// clockwise for a positive period and anticlockwise for a negative one,
	// added to Rotate. Zero holds still. See core.Spin for what each renderer
	// maps it onto and why it is a rotation rather than a general loop.
	Spin int `json:",omitzero"`

	// TranslateX and TranslateY shift the node's painted box, and its touch
	// target with it, without moving anything around it. See core.Translate
	// for the forms, the leading-relative direction rule and each renderer's
	// mapping.
	TranslateX string `json:",omitzero"`
	TranslateY string `json:",omitzero"`

	HoverStyle   *Style           `json:",omitzero"`
	FocusStyle   *Style           `json:",omitzero"`
	PseudoStates map[string]Style `json:",omitzero"` // ":hover", ":focus"

	FlexDirection  FlexDirection  `json:",omitzero"`
	JustifyContent JustifyContent `json:",omitzero"`
	AlignItems     AlignItems     `json:",omitzero"`
	MinHeight      string         `json:",omitzero"`
	MinWidth       string         `json:",omitzero"`
	ColumnGap      float64        `json:",omitzero"`
	RowGap         float64        `json:",omitzero"`
	FlexWrap       string         `json:",omitzero"`
	AlignSelf      AlignItems     `json:",omitzero"`
	FlexBasis      string         `json:",omitzero"`

	// FlexShrink is a flex item's shrink factor, and it is the one number in
	// this struct whose zero is not its own value. Read it through
	// ShrinkFactor rather than off the field; write it through
	// core.FlexShrink, which is what puts the sentinel here.
	//
	// See ShrinkNone for the whole of why.
	FlexShrink float64 `json:",omitzero"`
	FlexGrow   float64 `json:",omitzero"`

	// StackAlign is where this node sits inside the core.ZStack it is a layer
	// of: the per-layer opt-out from the stack's centre-on-both-axes contract.
	// See stack_align.go for the nine values, what each of the four renderers
	// does with one, and why this is a two-axis value of its own rather than a
	// second reading of AlignSelf above.
	//
	// It sits with the flex fields because it is the same kind of thing — a
	// child's say in its own placement — and deliberately not next to them in
	// meaning: those are read by the two DOM targets alone, and this one is
	// honoured on all four.
	StackAlign StackAlignment `json:",omitzero"`

	// Accessibility semantics. These live on Style rather than Props so every
	// builder that takes StyleProps — leaves and containers alike — supports
	// them without a signature change, and so the reconciler's value-compared
	// update-style patches carry changes to them like any visual property.
	// Renderers map them onto the platform's semantics layer: contentDescription
	// / clearAndSetSemantics on Android, accessibilityLabel / accessibilityHint /
	// accessibilityHidden on iOS.
	AccessibilityLabel  string `json:",omitzero"`
	AccessibilityHint   string `json:",omitzero"`
	AccessibilityHidden bool   `json:",omitzero"`

	// AccessibilityRole is what the node *is* — a heading, a table cell, a
	// search landmark — as opposed to what it is called and what tapping it
	// does. See role.go for the vocabulary, what each of the four renderers
	// makes of it, and why fourteen of the twenty-five values do nothing on either
	// native.
	//
	// It sits with the three fields above and travels the same way: on Style
	// rather than in Props, so every builder supports it without a signature
	// change and a change to it patches like any other style property.
	AccessibilityRole Role `json:",omitzero"`

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
	// no level at all, so this is inert on Android — the same honest gap fourteen
	// of the twenty-five roles have, documented in GrMobStyle.kt beside the role
	// dispatch rather than left for the next person to rediscover.
	//
	// Out-of-range values are dropped rather than clamped. 0 is the zero value
	// and means "a heading, tier unstated", which is what every heading in
	// every tree was before this field existed; anything above 6 has no
	// spelling on any of the three targets that can express a level, and
	// silently rewriting a 7 to a 6 would invent a structure the caller did
	// not describe.
	AccessibilityHeadingLevel int `json:",omitzero"`

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
	// gap fourteen of the twenty-seven roles have, and it is written down in
	// GrMobStyle.kt and GrMobStyle.swift beside the role dispatch rather than
	// left for the next person to rediscover.
	//
	// Values below 1 are dropped, as they are for a heading: 0 is the zero
	// value and means "an item, depth unstated", which is what every list item
	// in every tree is unless something says otherwise.
	AccessibilityNestingLevel int `json:",omitzero"`

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
	//	tab, row, columnheader,      aria-selected
	//	option, gridcell
	//	radio                        aria-checked
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
	// option, tab, row, columnheader and gridcell, and picking one of those for a node
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
	AccessibilitySelected SelectedState `json:",omitzero"`

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
	//	gridcell        aria-selected              yes
	//	option          aria-selected              no
	//	link            no                         yes
	//	listbox         no                         yes
	//	combobox        no                         yes, and required
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
	// container and carries a disclosure. comps.Accordion is what happens
	// when a widget takes that seriously — its header row is a button inside a
	// heading, which is ARIA's own accordion shape, rather than a named div
	// with a state a browser throws away.
	//
	// # The near miss: a control that opens a *dialog* is not expanded
	//
	// aria-expanded says the content is here, in the page, and can be shown or
	// hidden. A trigger that opens a modal is a different relationship —
	// ARIA spells that aria-haspopup, which is AccessibilityHasPopup below —
	// so comps.DatePicker's trigger, which looks exactly like a disclosure
	// and even flips a glyph, states a popup and leaves this field unset.
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
	// move comps.Chip's ", selected" name suffix was deleted for. So the
	// key crosses the bridge, is deliberately not parsed, and the note in
	// GrMobStyle.swift says which property it is turning down.
	AccessibilityExpanded ExpandedState `json:",omitzero"`

	// AccessibilityHasPopup is what this control opens — today, only ever a
	// dialog. See PopupKind for the vocabulary and for why ARIA's other six
	// values are not in it.
	//
	// # The near miss above, given its own field
	//
	// AccessibilityExpanded turns down a trigger that opens a modal, because a
	// disclosure's content is in the page and a dialog is a new surface. That
	// left comps.Menu and comps.DatePicker saying nothing at all, which is
	// the silence this closes: aria-haspopup is ARIA's spelling of exactly that
	// relationship, and a reader announces it with the name ("Sort, pop-up
	// button").
	//
	// # The role guard, which is a fifth list
	//
	// ARIA 1.2 defines aria-haspopup for application, button, combobox,
	// gridcell, link, menuitem, slider, tab, textbox and treeitem, lets
	// columnheader inherit it from gridcell, and deprecates it everywhere
	// else. Of those, core.Role carries button, link, tab, columnheader and
	// combobox. Both web exporters write it for exactly the roles the
	// generated fixture gives it among core's own —
	// aria/verify/aria_test.go's state-guard test asks every role in both
	// directions, so the arms of ariaHasPopup cannot drift from the
	// specification. A core.Button needs no role, the node type being one,
	// which is the case comps.Menu's trigger is.
	//
	// comps.DatePicker's trigger is a Row with an OnTap, so it states
	// RoleButton alongside this. Without a role the attribute would have
	// nothing to sit on but ariaRole's `group` fallback, and "group, pop-up"
	// describes nothing a reader can press.
	//
	// # Neither native reads it
	//
	// Compose's SemanticsProperties and SwiftUI's AccessibilityTraits have no
	// popup member. Both platforms present a Modal as a platform dialog that
	// announces itself on opening, so the warning arrives one step later there
	// rather than not at all. The key crosses the bridge and is deliberately
	// unparsed; the notes in GrMobStyle.kt and GrMobStyle.swift say so.
	AccessibilityHasPopup PopupKind `json:",omitzero"`

	// AccessibilityCurrent is whether this item is the current one of its set
	// — the page a bottom bar is showing, the step a wizard is on. See
	// CurrentKind for the vocabulary and for why it is not a selection.
	//
	// # The one state field with no role guard
	//
	// Every other state here is scoped by ARIA to a list of roles, and both
	// web exporters switch on the role to honour it. aria-current is a
	// global: ARIA defines it on every role, including the `group` a named
	// container falls back to. So both web targets write it whenever it is
	// stated and the node is not hidden, and comps.Drawer's rows, which carry
	// no control role, can say it.
	//
	// # Both natives fold it into the selected state
	//
	// Compose's semantics and SwiftUI's traits have no current property. Both
	// platforms' own navigation bars announce the current destination as
	// selected, so a node stating a kind gets `selected = true` on Compose and
	// `.isSelected` on SwiftUI. A stated AccessibilitySelected wins over the
	// fold, because it is the more specific claim and a node carrying both
	// would otherwise be announced twice or contradict itself.
	AccessibilityCurrent CurrentKind `json:",omitzero"`

	// AccessibilityValue is where a valued control sits inside its range —
	// how far an upload has got, which step a wizard is on. See ValueRange
	// for the vocabulary, for why the numbers are strings, and for why the
	// three of them are one field where the two levels are two.
	//
	// # The role guard, and the one role
	//
	// ARIA defines aria-valuenow and its two bounds for meter, progressbar,
	// scrollbar, slider, spinbutton and a focusable separator. core.Role
	// carries exactly one of those, so the guard in both web exporters is a
	// single arm — which is thin, and is ARIA's own scoping rather than a
	// shortlist. The absences are the same absence in every case: no widget
	// here is a meter, a scrollbar or a spinbutton. core.Slider is the near
	// miss and is deliberately outside it, because it exports as
	// <input type="range">, which carries value/min/max natively; an ARIA
	// range written on top would be a second claim about the same fact, free
	// to contradict the first.
	//
	// This is the fourth accessibility state field and it guards like the
	// other three and unlike them:
	//
	//	aria-level        heading, listitem, row     — two fields, one attribute
	//	aria-selected     option, tab, row, columnheader
	//	aria-pressed      button
	//	aria-expanded     button, link, listbox, row, columnheader, tab, combobox
	//	aria-haspopup     button, link, tab, columnheader, combobox
	//	aria-value*       progressbar
	//
	// No two of those lists are the same list, which is the argument for each
	// of these being a type of its own rather than one reused.
	//
	// # Text is the exception, and it is the more useful half on the phones
	//
	// The three numbers are web-only and Compose-only: the DOM writes them
	// verbatim, Compose has progressBarRangeInfo, and SwiftUI has no numeric
	// value property at all. ValueRange.Text is what all four can say — it
	// becomes aria-valuetext, Compose's stateDescription and SwiftUI's
	// accessibilityValue — which makes it the value channel GrMobStyle.swift's
	// AccessibilityExpanded note says the framework has no room for.
	//
	// Both natives honour their equivalent on any node and so do not scope it,
	// the same asymmetry a selection has: the web is strict because ARIA is,
	// not because the framework is.
	AccessibilityValue ValueRange `json:",omitzero"`

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
	AccessibilityID       string `json:",omitzero"`
	AccessibilityControls string `json:",omitzero"`

	// AccessibilitySelectionFollowsFocus makes a composite widget choose the
	// member the arrow keys land on, rather than only focusing it.
	//
	// Set on the *container* — the listbox or the tablist — not on the members.
	// It is a statement about the widget's contract with the keyboard, and a
	// per-member spelling would let a strip disagree with itself.
	//
	// Only a role with a keyboard can have one followed, and which those are is
	// core.KeyboardComposites(). On anything else — a list, a group, a Box
	// whose role was never set — this is inert in the strongest sense: the WASM
	// runtime writes the attribute for any node that asks, so the claim reaches
	// the DOM and nothing ever reads it back. core.AuditTree reports that in
	// debug mode as ConcernInertFollowsFocus, because no exporter can: the one
	// that writes the attribute deliberately does not know the composite tables
	// (see applyAccessibility), and knowing them there would put those tables
	// in two places.
	//
	//	core.Row(core.AccessibilityRole(core.RoleTabList),
	//	    core.AccessibilitySelectionFollowsFocus(),
	//	    …tabs…
	//	)
	//
	// # What it is for
	//
	// ARIA's tabs pattern recommends it outright: "tabs activate automatically
	// when they receive focus as long as their associated tab panels are
	// displayed without noticeable latency". Without it, a keyboard user
	// crossing a three-tab strip presses Right, Right, Enter, and the two
	// panels they arrowed past were never shown — which is a different
	// experience from the one a mouse user gets, in a widget whose whole job is
	// switching between things.
	//
	// ARIA's listbox pattern allows it for a single-select listbox and warns
	// about it for anything expensive, which is why this is a prop and not the
	// default. A strip of tabs over three local views should set it; a list
	// whose selection fires a network request must not, because arrowing from
	// the top of a hundred options to the bottom would fire a hundred.
	//
	// # It is the author's own OnTap that runs
	//
	// The runtime does not write aria-selected and could not: that attribute is
	// rendered from Go state, and a keystroke has no way to reach Go state
	// except through a callback. So this invokes the newly focused member's own
	// OnTap — the same callback Enter and Space already invoke on it — and the
	// selection then arrives the way every other selection does, as a render
	// pass. A member with no handler is focused and nothing else, which is the
	// same rule activation follows.
	//
	// That is worth stating because the obvious reading is that this is a
	// *rendering* feature, and the standing argument against it was that the
	// framework could not make the choice since aria-selected is written from
	// Go. The premise was wrong rather than the conclusion: Enter on a member
	// has always reached Go, and this is the same call on a different key.
	//
	// # Web only, and it is behaviour rather than semantics
	//
	// There is no ARIA attribute for it — it is a description of what a widget's
	// keyboard does, and ARIA describes what a widget IS — so nothing is
	// written into the DOM but a data attribute the runtime reads back.
	//
	// htmlout writes nothing for it, on exactly the argument that keeps the
	// roving tabindex out of the static export: an exporter with no key handler
	// has no focus to follow, so the flag would be a claim about behaviour that
	// does not exist there. Both natives write nothing either, and for the
	// original reason — VoiceOver and TalkBack cross a collection by swipe, so
	// there is no arrow key for a selection to follow.
	AccessibilitySelectionFollowsFocus bool `json:",omitzero"`

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
	// is a palette decision (comps.Button spends Surface/TextSecondary
	// on it); what it *means* is this flag.
	Disabled bool `json:",omitzero"`

	// Inert takes the node and its whole subtree out of reach on the web: no
	// focus, no Tab stop, no pointer events, and nothing in the accessibility
	// tree. It is the HTML `inert` attribute, written by both web targets.
	//
	// # Why a field of its own, and not AccessibilityHidden or Disabled
	//
	// The screen behind comps.Drawer is what asked. The drawer hid it with
	// AccessibilityHidden, and aria-hidden only prunes the accessibility tree;
	// it does not stop Tab. So a keyboard user tabbing past the panel's last
	// control walked into a screen that a reader could not see and the eye
	// could see only dimmed. The three near neighbours each cover part of the
	// job:
	//
	//	                     tree   Tab/focus   pointer   announced as
	//	AccessibilityHidden  gone   kept        kept      nothing
	//	Disabled             kept   gone        gone      "dimmed"/"disabled"
	//	Inert                gone   gone        gone      nothing
	//
	// AccessibilityHidden could not simply start writing `inert`, because it
	// also marks nodes that must stay interactive: a Drawer's scrim is hidden
	// from readers (the ✕ is the accessible way out) and still dismisses on a
	// tap. Disabled is the wrong claim, since a disabled control is announced
	// as one, and a screen behind a drawer is not a disabled screen. So this
	// is its own flag, the one the platform has.
	//
	// # What each target does with it
	//
	// Both web targets write the attribute, and the browser does the rest: it
	// blurs focus already inside the subtree, skips the subtree in sequential
	// navigation, drops pointer events and prunes the accessibility tree.
	//
	// Neither native reads it. On a phone the problem it solves is shaped
	// differently: VoiceOver and TalkBack are kept out by AccessibilityHidden,
	// and touch is kept out by whatever covers the layer (a Drawer's scrim).
	// What remains is a hardware keyboard's focus traversal on an iPad or a
	// Chromebook. SwiftUI's nearest tool, .disabled(true), dims system controls,
	// and Compose's focusProperties cancel entry only on the node they sit on.
	// Neither is the same claim, and neither can be checked here without a
	// device, so the gap is recorded rather than approximated. Set
	// AccessibilityHidden beside it when readers on the phones should be kept
	// out too, as Drawer does.
	Inert bool `json:",omitzero"`
}
```

Style is the visual and semantic description a node carries to whichever renderer is drawing it — Compose, SwiftUI, the DOM, or static HTML.

#### Why every field is \`omitzero\`

The three JSON hosts each decode this struct key by key with a zero default for an absent key ("missing" and "present but zero" have always meant the same thing to GrMobStyle.kt, GrMobStyle.swift and styleFromGrMob), so a field at its zero value carries no information across the wire. Before the tags it was still written out, and on a real screen almost every field is at its zero value: the tutorial's contents screen has 336 nodes and 1,168 non-zero style fields between them — about three and a half per node, out of fifty-eight.

The cost of writing the other fifty-four was not theoretical. That screen serialized to 423,472 bytes, of which 92.4% was Style, and org.json spent 1.6 seconds of a 5-second cold launch parsing it on an emulator:

	                        bytes      Android cold launch
	every field written    423,472     bridge 17ms · parse 1,600ms · build 430ms
	zero fields omitted     53,408     see android/device/launch.sh for the after
	EdgeInsets too          51,242     a later pass, and no reading — see below

The tags are \`omitzero\` rather than \`omitempty\` because two of the fields are structs — Padding and Margin are EdgeInsets, AccessibilityValue is a ValueRange — and \`omitempty\` has never omitted an empty struct. omitzero (Go 1.24) does, and it means the same thing for every other kind here, so one spelling covers the struct instead of two.

#### The one field whose zero is not its value

FlexShrink. Its "unset" and its "explicitly zero" are different states, and the difference is carried by a non-zero sentinel (ShrinkNone) rather than by the field's presence — which is exactly why these tags are safe on it. See ShrinkNone.

#### What this constrains

A future renderer must not read presence as meaning. If some field ever needs to distinguish "the author said nothing" from "the author said zero", it needs a sentinel like FlexShrink's or a pointer type; it cannot get that distinction back from the wire format.

#### The tags stopped one level too high, and the fix is the same one

These tags made an all-zero Padding vanish, which made it easy to miss that a \*present\* one still wrote all six of core.EdgeInsets' untagged ints. The axis pair was zero in all 77 insets on the contents screen — the DSL's side props settle the shorthand before writing — so 2,156 bytes were being spent saying "Horizontal":0. EdgeInsets and core.ValueRange carry the tags now, and TestEveryWireFieldOmitsZero (core/wire\_omitzero\_test.go) walks the whole tree so the next nested struct cannot be added without them.

It bought no measurable time, and that is worth stating rather than eliding: five cold launches each way on the same emulator move parse-and-build by 1.9ms against a run-to-run spread of 17-52ms. A 4% cut predicts ~8ms and 8ms is under that instrument's floor. The bytes are certain; the reading is not available at this size. TestHomeTreeSize carries both arms.

#### What is left, and why the next idea is not a shorter vocabulary

Of the 51,242 bytes now, roughly half are key names:

	Style field names       16,261    31.7%   1,479 fields
	Node field names         8,826    17.2%   Type/Style/Props/Children/Key
	Props key names          2,640     5.2%   "content" ×215, "onClick" ×49
	                        ──────
	                        27,727    54.1%

The obvious move is a short wire vocabulary — two-character codes instead of the Go field names. Sized: recoding Style alone saves 8,866 bytes (17.3% of the payload), and recoding all three groups saves 14,552 (28.4%). Against the 378ms of parse-and-build that emulator actually measures, and assuming the parse is linear in length, that is \*\*65ms and 107ms\*\* of a ~3,200ms cold launch: 2% and 3%.

It is declined, and the number is only half of why. The other half is that verbatim Go field names are load-bearing: they are the reason core.Role's ARIA spellings need no mapping table on either DOM target (see role.go), the reason a tree dumped from the bridge is readable in a debugger, and the reason app\_test.go's nodeStyle, wasm/verify and ios/verify can each decode the half of the tree they care about without a shared schema. A vocabulary puts a 58-entry table in three readers and a writer, and puts every one of those tools behind it.

And it is dominated. The same payload's other lever — sending only the core.List children near the viewport — is worth 66-76% of the bytes rather than 17-28%, on the same screen, with no change to how a field is spelled. See TestWhatWindowingWouldSave in examples/tutorial for that profile and for what it is still waiting on.

<small>[core/style.go:93](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L93)</small>

#### func (Style) ShrinkFactor

```go
func (s Style) ShrinkFactor() (factor float64, declared bool)
```

ShrinkFactor returns the effective flex-shrink and whether one was declared.

The two returns are the two questions a renderer has, and they are separate because a renderer that writes nothing for an undeclared factor is right: the CSS initial value is 1, so an omitted declaration and an explicit 1 lay out the same, and omitting keeps the output the size it was.

	declared == false   nothing was set. Write no declaration.
	declared == true    write the factor, which may be 0.

It exists so the ShrinkNone rule is stated once rather than in each renderer. The two DOM renderers spell their guards independently — that is deliberate elsewhere in this framework — but the mapping from a stored number to a meaning is not a spelling, it is the contract, and three copies of it is how this field got into trouble in the first place.

<small>[core/style.go:1399](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1399)</small>

#### func (Style) With

```go
func (s Style) With(other Style) Style
```

<small>[core/style_props.go:682](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L682)</small>

### type StyleProp

```go
type StyleProp interface {
	Apply(*Style)
}
```

<small>[core/style.go:863](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L863)</small>

#### func PaddingHorizontal

```go
func PaddingHorizontal(px int) StyleProp
```

PaddingHorizontal sets the left and right insets.

It writes the explicit Left/Right sides as well as the Horizontal shorthand. The renderers resolve a side as "the explicit value if non-zero, otherwise the axis shorthand" (see htmlout.EdgeCSS), so a prop that wrote only the shorthand could never override a side that was already set: a theme Column carries Left/Right 16, and PaddingHorizontal(0) after it used to leave the 16 in place — and PaddingHorizontal(24) used to render as 16. Writing the sides too gives this prop the same last-one-wins ordering every other StyleProp has, and a zero clears the theme value in all four renderers without any of them changing their resolution rule.

<small>[core/style.go:1248](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1248)</small>

#### func RoundedShadowBox

```go
func RoundedShadowBox() StyleProp
```

<small>[core/style.go:1220](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1220)</small>

#### func UseStyle

```go
func UseStyle(s Style) StyleProp
```

UseStyle turns a whole Style value into a StyleProp, so a caller can pass a named visual role ("the card surface", "the theme's Body typography") in one argument instead of unpacking it into a dozen individual props.

The merge rule is "a set field wins, an unset field is ignored": every field of s that holds a non-zero value overwrites the target's, and every field left at its zero value leaves the target's alone. That is what makes UseStyle composable — layering role styles onto a theme's component defaults only ever adds, never blanks out what the theme supplied.

The rule's one unavoidable edge is that a zero value is indistinguishable from "not set", so UseStyle cannot \*clear\* a field the target already has: Style{AccessibilityHidden: false} does not un-hide an element, and Style{FontSize: 0} does not reset a font size. Use the individual StyleProp setters (AccessibilityHidden(), FontSize(0)) when the intent is to force a value rather than to layer one.

This merges every field of Style. It previously covered only fourteen of them, which meant Width, Height, the whole flex group, and the accessibility fields were silently dropped — a style value carrying them applied cleanly and did nothing. Any field added to Style must be added here too; TestUseStyleMergesEveryField walks the struct reflectively and fails if one is missed.

<small>[core/style.go:895](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L895)</small>

### type Weight

```go
type Weight int
```

<small>[core/style.go:795](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L795)</small>

```go
const (
	Light  Weight = 200
	Normal Weight = 400
	Bold   Weight = 700
)
```

