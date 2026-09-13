# Package core — Styling: style props

```go
import "github.com/rohanthewiz/grmob/core"
```

The StyleProp constructors: spacing and per-side insets, flex, typography, colour, borders and animation.

One of 11 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/style_props.go`, `core/margin_sides.go`, `core/padding_sides.go`, `core/animation.go`.

## Index

- [Constants](#constants) — `ReducedMotionCSS`, `SpinKeyframes`, `TranslateDirectionCSS`
- [`func AccessibilityControls`](#func-accessibilitycontrols)
- [`func AccessibilityCurrent`](#func-accessibilitycurrent)
- [`func AccessibilityExpanded`](#func-accessibilityexpanded)
- [`func AccessibilityHasPopup`](#func-accessibilityhaspopup)
- [`func AccessibilityHeadingLevel`](#func-accessibilityheadinglevel)
- [`func AccessibilityHidden`](#func-accessibilityhidden)
- [`func AccessibilityHint`](#func-accessibilityhint)
- [`func AccessibilityID`](#func-accessibilityid)
- [`func AccessibilityLabel`](#func-accessibilitylabel)
- [`func AccessibilityNestingLevel`](#func-accessibilitynestinglevel)
- [`func AccessibilityRole`](#func-accessibilityrole)
- [`func AccessibilitySelected`](#func-accessibilityselected)
- [`func AccessibilitySelectionFollowsFocus`](#func-accessibilityselectionfollowsfocus)
- [`func AccessibilityValue`](#func-accessibilityvalue)
- [`func Align`](#func-align)
- [`func AlignItemsProp`](#func-alignitemsprop)
- [`func AlignSelf`](#func-alignself)
- [`func Background`](#func-background)
- [`func BackgroundColor`](#func-backgroundcolor)
- [`func BorderRadius`](#func-borderradius)
- [`func Bottom`](#func-bottom)
- [`func ColumnGap`](#func-columngap)
- [`func Disabled`](#func-disabled)
- [`func Display`](#func-display)
- [`func FlexBasis`](#func-flexbasis)
- [`func FlexDir`](#func-flexdir)
- [`func FlexGrow`](#func-flexgrow)
- [`func FlexShrink`](#func-flexshrink)
- [`func FlexWrap`](#func-flexwrap)
- [`func FontSize`](#func-fontsize)
- [`func FontWeight`](#func-fontweight)
- [`func Gap`](#func-gap)
- [`func Height`](#func-height)
- [`func Inert`](#func-inert)
- [`func Justify`](#func-justify)
- [`func Left`](#func-left)
- [`func LinearGradient`](#func-lineargradient)
- [`func Margin`](#func-margin)
- [`func MarginBottom`](#func-marginbottom)
- [`func MarginHorizontal`](#func-marginhorizontal)
- [`func MarginLeft`](#func-marginleft)
- [`func MarginRight`](#func-marginright)
- [`func MarginTop`](#func-margintop)
- [`func MarginVertical`](#func-marginvertical)
- [`func MaxHeight`](#func-maxheight)
- [`func MaxWidth`](#func-maxwidth)
- [`func MinHeight`](#func-minheight)
- [`func MinWidth`](#func-minwidth)
- [`func Overflow`](#func-overflow)
- [`func Padding`](#func-padding)
- [`func PaddingBottom`](#func-paddingbottom)
- [`func PaddingLeft`](#func-paddingleft)
- [`func PaddingRight`](#func-paddingright)
- [`func PaddingTop`](#func-paddingtop)
- [`func PaddingVertical`](#func-paddingvertical)
- [`func Responsive`](#func-responsive)
- [`func Right`](#func-right)
- [`func Rotate`](#func-rotate)
- [`func RowGap`](#func-rowgap)
- [`func Shadow`](#func-shadow)
- [`func Spin`](#func-spin)
- [`func TextColor`](#func-textcolor)
- [`func Transition`](#func-transition)
- [`func Translate`](#func-translate)
- [`func WhiteSpace`](#func-whitespace)
- [`func Width`](#func-width)
- [`func ZIndex`](#func-zindex)
- [`type Easing`](#type-easing)

## Constants

ReducedMotionCSS is the stylesheet rule both web targets pair with Style.Transition: under the reduce-motion media query, every transition a node declares inline is switched off. See Transition, "Reduced motion".

A stylesheet rule rather than a check in the runtime, for three reasons. The media query is live, so a reader who turns the setting on mid-session is honoured on the next change without a render pass or a listener. It works in an htmlout export, which has no script. And it is the only way to reach an inline declaration from outside: \`!important\` in a sheet beats a normal inline style.

The selector matches on the inline style attribute rather than on \`\*\`, so the rule reaches only elements a grmob renderer gave a transition (a hosting page's own transitions are the page's to manage). Both web targets write the property inline as "transition", and an element with no transition has nothing for the rule to switch off anyway, so the match is exact in the only direction that matters.

```go
const ReducedMotionCSS = `@media (prefers-reduced-motion:reduce){[style*="transition"]{transition:none!important}}`
```

<small>[core/animation.go:94](https://github.com/rohanthewiz/grmob/blob/master/core/animation.go#L94)</small>

SpinKeyframes is the stylesheet rule both web targets pair with Style.Spin. It animates the individual \`rotate\` property rather than \`transform\`, so the spin composes with Style.Rotate's \`transform: rotate()\` instead of replacing it (see Spin). One constant, read by htmlout and restated in the WASM runtime, because the two web targets must name and shape it identically for an export and a live page to turn the same way.

```go
const SpinKeyframes = "@keyframes grmob-spin{from{rotate:0deg}to{rotate:360deg}}"
```

<small>[core/animation.go:75](https://github.com/rohanthewiz/grmob/blob/master/core/animation.go#L75)</small>

TranslateDirectionCSS is the stylesheet rule both web targets pair with a non-zero Style.TranslateX: it sets --grmob-inline to -1 under dir="rtl" and back to 1 under dir="ltr", so the web's physical translate becomes Translate's leading-relative one. Custom properties inherit, so matching the dir attribute where it is written covers every element beneath it, and a nested dir="ltr" inside an RTL page switches back. The attribute rather than :dir(), which also reads only the attribute, because the plain selector is supported everywhere and matches far fewer elements.

```go
const TranslateDirectionCSS = "[dir=rtl]{--grmob-inline:-1}[dir=ltr]{--grmob-inline:1}"
```

<small>[core/style_props.go:235](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L235)</small>

## Functions

### func AccessibilityControls

```go
func AccessibilityControls(id string) StyleProp
```

AccessibilityControls says which element this control switches, by the AccessibilityID that element was given. It becomes aria-controls on both web targets and is deliberately unread on both natives.

	core.Box(
		core.AccessibilityRole(core.RoleTab),
		core.AccessibilitySelected(core.SelectedWhen(tab == "home")),
		core.AccessibilityID("home-tab"),
		core.AccessibilityControls("app-panel"),
		core.Text("Home"),
	)

It is written verbatim and nothing checks that the target exists: an export is one document at a time and a runtime patch is one element at a time, so neither target can see the whole page at the moment the attribute is written. A reference to an id nothing answers to is inert rather than harmful, which is the same trade aria-description makes. See Style.AccessibilityID for why this is the one relationship the vocabulary carries.

<small>[core/style_props.go:621](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L621)</small>

### func AccessibilityCurrent

```go
func AccessibilityCurrent(kind CurrentKind) StyleProp
```

AccessibilityCurrent says this item is the current one of its set.

	core.Column(core.AccessibilityRole(core.RoleButton),
		core.AccessibilityCurrent(core.CurrentPage), …)

aria-current on both web targets, on any role. Both natives announce it as selected unless AccessibilitySelected is stated. See CurrentKind and Style.AccessibilityCurrent.

<small>[core/style_props.go:548](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L548)</small>

### func AccessibilityExpanded

```go
func AccessibilityExpanded(state ExpandedState) StyleProp
```

AccessibilityExpanded says whether this disclosure is open — the accordion section showing its body, the twisty that has been turned.

	core.Button(title, toggle,
		core.AccessibilityExpanded(core.ExpandedWhen(open.Get())),
	)

Use core.ExpandedWhen to convert the bool the widget already holds. Setting only the open case and leaving the shut one unset is the mistake the three-valued type exists to prevent: a closed disclosure that says nothing is announced as an ordinary button, and "collapsed" is the whole of what invites the press.

Paired with a role that can carry it, as a level and a selection both are — and \*not\* the same list a selection takes. aria-expanded is defined for button, link, listbox, row, columnheader and combobox among the roles this framework carries, which drops option and adds link and listbox. A core.Button needs no role of its own, the node type being one; anything else is dropped by both web targets. See Style.AccessibilityExpanded for the full table, for the dialog-shaped near miss it deliberately does not cover, and for why one native maps this and the other cannot.

<small>[core/style_props.go:517](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L517)</small>

### func AccessibilityHasPopup

```go
func AccessibilityHasPopup(kind PopupKind) StyleProp
```

AccessibilityHasPopup says what activating this control opens.

	comps.Button{Label: "⋯", AccessibilityLabel: "Note actions",
		Style: []core.StyleProp{core.AccessibilityHasPopup(core.PopupDialog)}}

It is the relationship AccessibilityExpanded deliberately does not cover: a trigger that presents a core.Modal is not a disclosure, and says so with this instead. Paired with a role ARIA 1.2 defines the attribute on — button, link, tab, columnheader and combobox among core's — or with a core.Button, whose node type is one; anything else is dropped by both web targets. Neither native reads it. See PopupKind and Style.AccessibilityHasPopup.

<small>[core/style_props.go:534](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L534)</small>

### func AccessibilityHeadingLevel

```go
func AccessibilityHeadingLevel(level int) StyleProp
```

AccessibilityHeadingLevel says how deep a heading sits — 1 for the screen's name, 2 for a section inside it, down to 6.

	core.Box(
		core.AccessibilityRole(core.RoleHeading),
		core.AccessibilityHeadingLevel(2),
		core.Text("March"),
	)

Paired with RoleHeading, never alone: the role says the node is a heading and this says which tier, so a level with no role describes the depth of something that is not a heading and every renderer drops it. The two are separate props rather than one because the tier is genuinely optional — every heading written before this existed is still a correct heading, it just does not say where it sits.

What a level buys is the outline. Without one, a screen with a bar title over a run of section bands announces a flat list of peers, and a reader navigating by heading cannot tell the screen's name from the band inside it. comps.AppBar and comps.GroupedList set 1 and 2 for exactly that pair, so the common case needs no call site at all.

See Style.AccessibilityHeadingLevel for the range rule (out-of-range is dropped, not clamped) and for which of the four renderers can express a level — Compose cannot, and that is stated rather than faked.

<small>[core/style_props.go:428](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L428)</small>

### func AccessibilityHidden

```go
func AccessibilityHidden() StyleProp
```

AccessibilityHidden removes the element (and its subtree) from the accessibility tree — for decorative content a screen reader should skip.

<small>[core/style_props.go:629](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L629)</small>

### func AccessibilityHint

```go
func AccessibilityHint(hint string) StyleProp
```

AccessibilityHint describes the \*result\* of activating the element ("Opens the article"). VoiceOver reads it natively; TalkBack has no hint slot, so the Android renderer appends it to the content description.

<small>[core/style_props.go:376](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L376)</small>

### func AccessibilityID

```go
func AccessibilityID(id string) StyleProp
```

AccessibilityID gives this element a document-global name that another element can point at with AccessibilityControls. It becomes the \`id\` attribute on both web targets and is deliberately unread on both natives.

	core.Box(core.AccessibilityID("app-panel"), core.AccessibilityLabel("Home"), …)

Uniqueness is the caller's, as it is in hand-written HTML, and the "grmob-" prefix is reserved for core.TabView's own wiring. See Style.AccessibilityID for the whole argument — including why this and AccessibilityControls are the only two IDREF props in the vocabulary.

<small>[core/style_props.go:596](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L596)</small>

### func AccessibilityLabel

```go
func AccessibilityLabel(label string) StyleProp
```

AccessibilityLabel gives screen readers a name for the element (TalkBack contentDescription, VoiceOver label). Set it on anything non-textual a user can perceive or activate — images, icon buttons, tappable rows.

<small>[core/style_props.go:367](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L367)</small>

### func AccessibilityNestingLevel

```go
func AccessibilityNestingLevel(level int) StyleProp
```

AccessibilityNestingLevel says how deep an item sits inside a nested collection — 1 for a top-level item, 2 for one inside it, upward with no ceiling.

	core.Box(
		core.AccessibilityRole(core.RoleListItem),
		core.AccessibilityNestingLevel(2),
		core.Text("Compline"),
	)

Paired with RoleListItem or RoleRow, never alone, and never with RoleHeading — the depth of a heading is AccessibilityHeadingLevel, which is a different question with a different range. Both become aria-level on the web, and which one is read is decided entirely by the role, so the two can never contend for the attribute.

What a depth buys is the shape of the tree. A flattened nested list — every item a sibling of every other — is what a reader gets from a run of divs, and it is also what it gets from a correctly roled list whose items do not say how deep they are: "list, twelve items" for something the eye reads as three groups of four.

Nothing in the framework sets one. Neither DataTable's rows (a flat table) nor any bundled widget nests a collection inside itself, so unlike the heading pair — which comps.AppBar and comps.GroupedList set for every app without a call site — this is a prop an application reaches for when it builds the nesting itself.

See Style.AccessibilityNestingLevel for why this is a second field rather than a widened first one, and for the two natives that cannot express a depth at all.

<small>[core/style_props.go:465](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L465)</small>

### func AccessibilityRole

```go
func AccessibilityRole(role Role) StyleProp
```

AccessibilityRole says what the element is: a heading, a table cell, a search landmark, a tappable Box that is really a button.

	core.Box(core.AccessibilityRole(core.RoleHeading), core.Text("Sermons"))

It is the third question a screen reader asks, after the name (AccessibilityLabel) and the effect (AccessibilityHint), and the one nothing here could answer until it existed — every container exports as a \<div> and announces as text. See core.Role for the vocabulary and for which of the four renderers honors which value.

Roles are not synthesized from node type or from props: a Box with an OnTap is a button only if it says so. Guessing would mean a widget that wraps a tappable row in a tappable card announcing two nested buttons, and the widget is the only layer that knows which one is the control.

<small>[core/style_props.go:397](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L397)</small>

### func AccessibilitySelected

```go
func AccessibilitySelected(state SelectedState) StyleProp
```

AccessibilitySelected says whether this control is on — the applied filter chip, the tab that is showing, the chosen day in a calendar.

	core.Box(
		core.AccessibilityRole(core.RoleTab),
		core.AccessibilitySelected(core.SelectedWhen(i == current)),
		core.Text(label),
	)

Use core.SelectedWhen to convert the bool a widget already holds. Passing core.SelectedOn alone and leaving the other controls unset is the mistake the three-valued type exists to prevent — see SelectedState.

Paired with a role that can carry a state, exactly as a level is: tab, row and columnheader take aria-selected, a button takes aria-pressed, and a state on anything else is dropped by both web targets because ARIA does not define either attribute there. A core.Button needs no role of its own; the node type is one. See Style.AccessibilitySelected for the two-attribute mapping and for why the natives do not scope it the same way.

<small>[core/style_props.go:490](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L490)</small>

### func AccessibilitySelectionFollowsFocus

```go
func AccessibilitySelectionFollowsFocus() StyleProp
```

AccessibilitySelectionFollowsFocus makes a composite widget choose the member the arrow keys land on, by invoking that member's own OnTap.

Set on the container — the listbox or the tablist — not on the members. See Style.AccessibilitySelectionFollowsFocus for when it is right and when it is the wrong thing to ask for.

A no-arg flag rather than a bool, like AccessibilityHidden and unlike Disabled: a caller does not have this in a variable, and there is no case for forcing it back off — a widget that does not want it writes no prop.

<small>[core/style_props.go:645](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L645)</small>

### func AccessibilityValue

```go
func AccessibilityValue(v ValueRange) StyleProp
```

AccessibilityValue says where a valued control sits inside its range — how far an upload has got, which step a wizard is on.

	core.Row(
		core.AccessibilityRole(core.RoleProgressBar),
		core.AccessibilityLabel("Upload"),
		core.AccessibilityValue(core.ValueOf(45, 0, 100)),
		…
	)

Use core.ValueOf to convert the numbers the widget is already holding, and .WithText when the digits are not what a listener wants to hear ("step 3 of 5"). Leaving it unset beside RoleProgressBar is not an omission — it is ARIA's own spelling of an \*indeterminate\* bar, one that is running with no idea how far.

Paired with a role that can carry it, as the level, the selection and the disclosure all are, and with the narrowest list of the four: aria-valuenow and its bounds are defined for six roles and core carries one of them, progressbar. core.Slider is the near miss and is deliberately outside — it exports as \<input type="range">, which states its own range natively.

ValueRange.Text is the half that is not web-only: it reaches Compose's stateDescription and SwiftUI's accessibilityValue, neither of which asks what the node is. See Style.AccessibilityValue for the guard table and core.ValueRange for why the numbers are strings.

<small>[core/style_props.go:580](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L580)</small>

### func Align

```go
func Align(a Alignment) StyleProp
```

<small>[core/style_props.go:133](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L133)</small>

### func AlignItemsProp

```go
func AlignItemsProp(a AlignItems) StyleProp
```

<small>[core/style_props.go:309](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L309)</small>

### func AlignSelf

```go
func AlignSelf(value AlignItems) StyleProp
```

<small>[core/style_props.go:32](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L32)</small>

### func Background

```go
func Background(w string) StyleProp
```

<small>[core/style_props.go:279](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L279)</small>

### func BackgroundColor

```go
func BackgroundColor(hex string) StyleProp
```

<small>[core/style_props.go:128](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L128)</small>

### func BorderRadius

```go
func BorderRadius(px float64) StyleProp
```

<small>[core/style_props.go:152](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L152)</small>

### func Bottom

```go
func Bottom(v string) StyleProp
```

<small>[core/style_props.go:330](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L330)</small>

### func ColumnGap

```go
func ColumnGap(px float64) StyleProp
```

<small>[core/style_props.go:52](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L52)</small>

### func Disabled

```go
func Disabled(disabled bool) StyleProp
```

Disabled hands the node to the platform's own disabled state: it stops accepting taps, keystrokes and focus, and screen readers announce it as disabled (Compose \`enabled = false\`, SwiftUI \`.disabled(true)\`, the HTML \`disabled\` attribute). See Style.Disabled for the full contract.

It takes the value rather than being a no-arg flag (unlike AccessibilityHidden) because the caller almost always has a bool in hand — \`core.Disabled(sending.Get())\` — and because passing false is the only way to force a node back to enabled: UseStyle's "a zero value means unset" rule means a Style{Disabled: false} cannot clear a flag already on the target.

<small>[core/style_props.go:661](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L661)</small>

### func Display

```go
func Display(mode DisplayMode) StyleProp
```

<small>[core/style_props.go:139](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L139)</small>

### func FlexBasis

```go
func FlexBasis(value string) StyleProp
```

<small>[core/style_props.go:27](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L27)</small>

### func FlexDir

```go
func FlexDir(dir FlexDirection) StyleProp
```

<small>[core/style_props.go:297](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L297)</small>

### func FlexGrow

```go
func FlexGrow(value float64) StyleProp
```

<small>[core/style_props.go:5](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L5)</small>

### func FlexShrink

```go
func FlexShrink(value float64) StyleProp
```

FlexShrink sets a flex item's shrink factor. Zero means "do not shrink", and it is stored as core.ShrinkNone — see that constant for why this one number cannot use the zero-means-unset convention every other number in Style does.

The mapping lives here rather than in Style.Merge because this is the only door into the field: a caller writes core.FlexShrink(0) and a renderer reads Style.ShrinkFactor(), and nothing in between has to know about the sentinel.

<small>[core/style_props.go:18](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L18)</small>

### func FlexWrap

```go
func FlexWrap(enabled bool) StyleProp
```

<small>[core/style_props.go:37](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L37)</small>

### func FontSize

```go
func FontSize(size float64) StyleProp
```

<small>[core/style_props.go:111](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L111)</small>

### func FontWeight

```go
func FontWeight(weight Weight) StyleProp
```

<small>[core/style_props.go:237](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L237)</small>

### func Gap

```go
func Gap(px float64) StyleProp
```

<small>[core/style_props.go:122](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L122)</small>

### func Height

```go
func Height(w string) StyleProp
```

<small>[core/style_props.go:269](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L269)</small>

### func Inert

```go
func Inert(inert bool) StyleProp
```

Inert takes the node and everything inside it out of reach on the web: out of the tab order, out of pointer events and out of the accessibility tree (the HTML \`inert\` attribute). The phones do not read it. See Style.Inert for why it is a flag of its own and for what the natives lack.

	core.Box(core.Inert(drawerOpen), screen)

A bool for Disabled's reason: passing false is the only way to clear a flag UseStyle has already put on the target.

<small>[core/style_props.go:676](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L676)</small>

### func Justify

```go
func Justify(j JustifyContent) StyleProp
```

<small>[core/style_props.go:303](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L303)</small>

### func Left

```go
func Left(v string) StyleProp
```

<small>[core/style_props.go:336](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L336)</small>

### func LinearGradient

```go
func LinearGradient(x, y, z string) string
```

<small>[core/style_props.go:285](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L285)</small>

### func Margin

```go
func Margin(all int) StyleProp
```

<small>[core/style_props.go:289](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L289)</small>

### func MarginBottom

```go
func MarginBottom(px int) StyleProp
```

MarginBottom sets the bottom margin alone. A zero clears.

This is the stacking prop: the gap under one item in a run that a parent's Gap does not describe, which is what examples/chat's message bubble wanted.

	core.Box(core.MarginBottom(8), bubble)

<small>[core/margin_sides.go:80](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L80)</small>

### func MarginHorizontal

```go
func MarginHorizontal(px int) StyleProp
```

MarginHorizontal sets the left and right margins.

This is the inset prop: a rule that stops short of the screen edge, which is what comps.Separator's Inset wanted.

	core.Box(core.MarginHorizontal(16), rule)

It writes the explicit sides as well as the shorthand, for the reason given on PaddingHorizontal.

<small>[core/margin_sides.go:112](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L112)</small>

### func MarginLeft

```go
func MarginLeft(px int) StyleProp
```

MarginLeft sets the left margin alone. A zero clears.

<small>[core/margin_sides.go:88](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L88)</small>

### func MarginRight

```go
func MarginRight(px int) StyleProp
```

MarginRight sets the right margin alone. A zero clears.

<small>[core/margin_sides.go:96](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L96)</small>

### func MarginTop

```go
func MarginTop(px int) StyleProp
```

MarginTop sets the top margin alone, leaving the other three as they were. A zero clears whatever a theme or an earlier prop supplied.

<small>[core/margin_sides.go:67](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L67)</small>

### func MarginVertical

```go
func MarginVertical(px int) StyleProp
```

MarginVertical sets the top and bottom margins. Writes the explicit sides as well as the shorthand, for the reason given on PaddingHorizontal.

<small>[core/margin_sides.go:122](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L122)</small>

### func MaxHeight

```go
func MaxHeight(w string) StyleProp
```

<small>[core/style_props.go:274](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L274)</small>

### func MaxWidth

```go
func MaxWidth(w string) StyleProp
```

MaxWidth caps a node's width: CSS \`max-width\`, honoured on all four targets.

The value is a dimension string: "320px" (or a bare number, in points on the natives), a percentage of the width the parent offers, or "" / "none" for no cap. The cap limits the painted box, padding included, and never the margin around it. It never grows a box — a label narrower than its cap keeps its own width — and it wins over a wider Width: Width("600px") with MaxWidth("520px") draws 520.

A stretched child of a Column (the default for most children) fills the column up to the cap and sits at the start of the line, as in a browser; centre it with the parent's AlignItems. The web targets pass the string through verbatim, so units the natives do not read ("vw", "em") cap only there. One case differs on the natives: a FlexGrow child of a Row whose cap binds keeps its share of the row and leaves the rest empty, where CSS hands the remainder to the other growers.

<small>[core/style_props.go:264](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L264)</small>

### func MinHeight

```go
func MinHeight(value string) StyleProp
```

<small>[core/style_props.go:63](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L63)</small>

### func MinWidth

```go
func MinWidth(value string) StyleProp
```

<small>[core/style_props.go:57](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L57)</small>

### func Overflow

```go
func Overflow(value string) StyleProp
```

<small>[core/style_props.go:68](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L68)</small>

### func Padding

```go
func Padding(all int) StyleProp
```

<small>[core/style_props.go:145](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L145)</small>

### func PaddingBottom

```go
func PaddingBottom(px int) StyleProp
```

PaddingBottom sets the bottom inset alone. A zero clears.

<small>[core/padding_sides.go:143](https://github.com/rohanthewiz/grmob/blob/master/core/padding_sides.go#L143)</small>

### func PaddingLeft

```go
func PaddingLeft(px int) StyleProp
```

PaddingLeft sets the left inset alone. A zero clears.

This is the indent prop: a nested row states its own depth without having to restate the three sides its theme container already got right.

	core.Row(core.PaddingLeft(16*depth), ...)

<small>[core/padding_sides.go:156](https://github.com/rohanthewiz/grmob/blob/master/core/padding_sides.go#L156)</small>

### func PaddingRight

```go
func PaddingRight(px int) StyleProp
```

PaddingRight sets the right inset alone. A zero clears.

<small>[core/padding_sides.go:164](https://github.com/rohanthewiz/grmob/blob/master/core/padding_sides.go#L164)</small>

### func PaddingTop

```go
func PaddingTop(px int) StyleProp
```

PaddingTop sets the top inset alone, leaving the other three as they were. A zero clears whatever the theme or an earlier prop supplied.

<small>[core/padding_sides.go:135](https://github.com/rohanthewiz/grmob/blob/master/core/padding_sides.go#L135)</small>

### func PaddingVertical

```go
func PaddingVertical(px int) StyleProp
```

PaddingVertical sets the top and bottom insets. Writes the explicit sides as well as the shorthand, for the reason given on PaddingHorizontal.

<small>[core/style_props.go:356](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L356)</small>

### func Responsive

```go
func Responsive(breakpoint string, style Style) StyleProp
```

Responsive registers a style variant under a named key in PseudoStates (":hover", ":focus", or a breakpoint name).

The entry is written into a fresh map rather than into whatever map the target already holds. A Style is copied by assignment throughout the framework — containerNode starts each node from a shallow copy of the theme's component Style — so the target's map may well be the theme's own. Writing into it in place would edit the theme for every render afterwards.

<small>[core/style_props.go:101](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L101)</small>

### func Right

```go
func Right(v string) StyleProp
```

<small>[core/style_props.go:342](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L342)</small>

### func Rotate

```go
func Rotate(deg float64) StyleProp
```

Rotate turns the node clockwise by deg degrees about its own centre, without disturbing the layout around it. See Style.Rotate for what each renderer maps it onto and why the angle is not normalised.

Unlike UseStyle, this setter can force zero: Rotate(0) writes the field, which is how a caller clears an angle a theme or role style supplied.

<small>[core/style_props.go:170](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L170)</small>

### func RowGap

```go
func RowGap(px float64) StyleProp
```

<small>[core/style_props.go:46](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L46)</small>

### func Shadow

```go
func Shadow(elevation float64) StyleProp
```

<small>[core/style_props.go:158](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L158)</small>

### func Spin

```go
func Spin(periodMs int) StyleProp
```

Spin turns the node one full revolution every periodMs milliseconds, round and round, for as long as it is displayed. A negative period turns it anticlockwise; zero (the default) holds it still. It is the looping counterpart of Transition, and it exists so that a continuously moving widget — the comps.Spinner ring — is driven by the platform's frame clock rather than by state changes stepped from Go.

	CSS       animation: grmob-spin <ms>ms linear infinite [reverse], with
	          SpinKeyframes on the individual `rotate` property
	Compose   a layout modifier node placing the box in a graphics layer whose
	          rotationZ follows withInfiniteAnimationFrameMillis
	SwiftUI   TimelineView(.animation) turning the box with .rotationEffect,
	          the angle computed from the timeline's date

All three compute the angle from elapsed time modulo the period rather than accumulating per-frame increments, so a dropped frame costs a skipped angle, never a spinner that drifts slower than declared.

#### Why a spin and not a general looping transition

Transition animates \*between\* two values, and the patch supplies both ends: the old style and the new one. A loop has no second patch to supply the other end, so a general loop would need vocabulary of its own — a from-style, a to-style, a repeat count, a direction — mapped onto three animation systems that do not agree on what repeating an arbitrary property means (Compose's infiniteRepeatable is per animated value, SwiftUI's repeatForever rides a transaction, CSS keyframes are a named rule). Rotation is the one property whose cycle closes on its own: 360 degrees draws exactly what 0 does, so the restart is invisible and the loop needs no second endpoint and no alternate-direction mode. It is also the only paint transform core has (see Style.Rotate). A pulse or a shimmer would be the second consumer that justifies the general form; until one exists, the narrow prop is the one all three targets implement identically.

#### Linear, always

There is no easing parameter. An eased revolution slows to a stop at the same angle every turn, which reads as a stutter rather than a spin, and the restart seam that is invisible under linear motion becomes a visible jolt. The period is the only knob.

#### Composition with Rotate

Spin is added to Rotate, not substituted for it. Both turn the box about its own centre, and rotations about one point commute, so each target applies them as two layers and draws the same pixels whichever is outermost.

#### What it costs

Nothing crosses the bridge after the style that declares it: no patches and no render passes. A node with Display none is not composed on either native and runs no CSS animation on the web, so a hidden spinning node draws no frames either.

#### Reduced motion: it keeps turning

A spin is not stopped or slowed when the platform's reduce-motion setting is on, on any target, while a Transition under the same setting snaps (see Transition). The choice, and what it was weighed against:

  - What a spin says. comps.Spinner is the one consumer, and its motion is the message: "still working". A ring frozen at an angle reads as a hung screen or as decoration, and nothing else on the widget says busy. WCAG 2.3.3 (Animation from Interactions) exempts motion that is essential to the information conveyed; an activity indicator is the textbook case of that.
  - What the setting is for. Reduce motion targets vestibular triggers: content sliding across the screen, zooms, parallax, large surfaces moving. A small glyph turning in place is none of those.
  - What the platforms do with their own spinner. UIActivityIndicatorView and SwiftUI's ProgressView keep spinning under Reduce Motion. The web has no built-in spinner, and Bootstrap's slows rather than stops. Android's indeterminate ProgressBar does freeze under "Remove animations", but that switch removes every animator in the system, including the ones apps rely on to show progress, and a frozen ring is the known cost of it rather than a design.
  - Why not slow it. A slower period is the web-library compromise, but the factor would be invented (twice? four times?), it would need a runtime read of the setting on Compose, where the loop is not a scaled animation, and a slow spin is still a spin to anyone it bothers.

So the frame loops stay as they are: Compose's withInfiniteAnimationFrameMillis does not read the animator duration scale, SwiftUI's TimelineView does not read the environment, and ReducedMotionCSS touches \`transition\` only, never \`animation\`. A caller for whom the motion is decoration rather than a status should not use Spin for it.

<small>[core/animation.go:182](https://github.com/rohanthewiz/grmob/blob/master/core/animation.go#L182)</small>

### func TextColor

```go
func TextColor(hex string) StyleProp
```

<small>[core/style_props.go:117](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L117)</small>

### func Transition

```go
func Transition(durationMs int, easing Easing) StyleProp
```

Transition declares that changes to this node's animatable properties — background color, size, padding, list placement — should animate over the given duration instead of snapping. This is the "declare in Go, drive natively" model: Go only ships the declaration in the style; each frame of the animation is produced by the platform's animation system (Compose, SwiftUI, CSS transitions), never by patches over the bridge.

The canonical serialized form is "\<ms>ms \<easing>" (e.g. "250ms ease-in-out"), which the native parsers read; they also tolerate the CSS longhand ("all 0.3s ease") for styles written by hand.

#### Reduced motion

When the platform's reduce-motion setting is on, a Transition snaps: the change lands on the next frame, exactly as it would with no Transition declared. The setting is read by each target rather than passed from Go, so turning it on mid-session applies to the next change without a render.

	CSS       ReducedMotionCSS: under prefers-reduced-motion: reduce, any
	          element whose inline style declares a transition gets
	          transition: none !important
	Compose   nothing to add: Android's "Remove animations" sets the
	          animator duration scale to 0, which Compose's frame clock
	          already reads (MotionDurationScale), and a tween under scale 0
	          plays straight to its end value
	SwiftUI   @Environment(\.accessibilityReduceMotion) swaps the node's
	          Animation for nil

All properties snap, colour included, rather than only the ones that move (size, placement, a translation). A colour fade is not the motion the setting is about, and the web could keep it, but SwiftUI's Animation is scoped to a value and not to a property, so keeping fades there means splitting the box chain into per-property animations. One rule that every target implements the same way beats a finer one that holds on two.

<small>[core/animation.go:56](https://github.com/rohanthewiz/grmob/blob/master/core/animation.go#L56)</small>

### func Translate

```go
func Translate(x, y string) StyleProp
```

Translate shifts the node's painted box by x along the inline axis and y down the block axis, without disturbing the layout around it. Its purpose is motion: paired with core.Transition, changing it slides a node, which is how comps.Drawer brings its panel in from the edge.

	core.Box(core.Transition(250, core.EaseOut), core.Translate("-100%", ""))

Each axis takes "Npx", a bare number (the same px), or "N%" of the node's own box on that axis, so "-100%" moves a panel exactly its own width whatever that width is. "" or any zero leaves the axis where it is; any other form is ignored on all four targets alike.

#### Paint, not layout

Like Rotate, the box keeps the size and place it laid out with and its siblings never reflow; only the pixels, and the touch target with them, move. A node translated out of its parent overflows it and is clipped by the parent's Overflow("hidden"), which both natives read for exactly this.

	CSS       translate: calc(var(--grmob-inline, 1) * x) y
	Compose   a layout modifier placing the box at placeRelative(x, y)
	SwiftUI   a GeometryEffect whose translation is resolved against the
	          view's own size

All three hit-test the moved box where it is drawn. The CSS property is the individual \`translate\`, not \`transform\`, so it composes with Rotate's transform and Spin's \`rotate\` in CSS's fixed order: translate outermost. The natives apply it outside both rotations for the same result.

#### Leading, not left

A positive x moves toward the trailing edge: right in a left-to-right layout, left in a right-to-left one. That is the natives' own rule (Compose's placeRelative mirrors, and SwiftUI mirrors a GeometryEffect's translation under RTL, measured with ImageRenderer), and it is what a widget wants: a drawer on the leading edge hides at "-100%" in both directions. CSS translate is physical, so both web targets multiply x by --grmob-inline, a custom property TranslateDirectionCSS sets to -1 under dir="rtl". Without the rule on the page the multiplier falls back to 1, which is right for every left-to-right document. y has no such question.

Zero on both axes writes no declaration on the web, so an untranslated node never becomes a containing block for its fixed-position descendants, and a transition to zero still animates, because CSS interpolates to none.

<small>[core/style_props.go:220](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L220)</small>

### func WhiteSpace

```go
func WhiteSpace(value string) StyleProp
```

WhiteSpace controls how runs of spaces and line breaks in a Text are treated: "nowrap" keeps a line whole and lets the container scroll or clip it, "pre" additionally preserves the literal spacing, "normal" is the default wrapping behavior.

Style.WhiteSpace has been carried to both web targets since it was added, but nothing could set it — this is the missing constructor, not a new capability. It matters most for text whose columns mean something (code listings, tabular output), where wrapping restarts the continuation at column zero and makes the indentation actively misleading.

It is a no-op on the natives, which have no equivalent knob: SwiftUI and Compose wrap by line-break policy rather than by a CSS-style property.

<small>[core/style_props.go:87](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L87)</small>

### func Width

```go
func Width(w string) StyleProp
```

<small>[core/style_props.go:243](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L243)</small>

### func ZIndex

```go
func ZIndex(v int) StyleProp
```

<small>[core/style_props.go:348](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L348)</small>

## Types

### type Easing

```go
type Easing string
```

Easing names the timing curve of a transition. The values are the CSS keywords — the Style.Transition field predates the native renderers and is CSS-shaped, so the DSL keeps that vocabulary and each renderer maps it onto its own curve type (Compose CubicBezierEasing, SwiftUI Animation). The cubic-bezier control points the CSS spec defines for each keyword are what the native mappings reproduce, so one Go declaration animates identically on Android, iOS, and the web backends.

<small>[core/animation.go:12](https://github.com/rohanthewiz/grmob/blob/master/core/animation.go#L12)</small>

```go
const (
	EaseLinear Easing = "linear"
	Ease       Easing = "ease"
	EaseIn     Easing = "ease-in"
	EaseOut    Easing = "ease-out"
	EaseInOut  Easing = "ease-in-out"
)
```

