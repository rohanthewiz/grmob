# Package core — Controls

```go
import "github.com/rohanthewiz/grmob/core"
```

Buttons, text inputs, switches, sliders, selects, images, tab views, text grids and vector canvases.

One of 11 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/button.go`, `core/input.go`, `core/switch.go`, `core/slider.go`, `core/select_menu.go`, `core/image.go`, `core/tabview.go`, `core/textgrid.go`, `core/canvas.go`.

## Index

- [Constants](#constants) — `GridBold`, `GridDim`, `GridItalic`, `GridStrike`, `GridUnderline`, `PathClose`, `PathCubic`, `PathLine`, `PathMove`
- [`func Button`](#func-button)
- [`func ButtonWithEvent`](#func-buttonwithevent)
- [`func Canvas`](#func-canvas)
- [`func CanvasMapping`](#func-canvasmapping)
- [`func Checkbox`](#func-checkbox)
- [`func Image`](#func-image)
- [`func ImageWithMode`](#func-imagewithmode)
- [`func Input`](#func-input)
- [`func InputPassword`](#func-inputpassword)
- [`func InputWithSubmit`](#func-inputwithsubmit)
- [`func NumericInput`](#func-numericinput)
- [`func OnSliderChangeEnd`](#func-onsliderchangeend)
- [`func Select`](#func-select)
- [`func Slider`](#func-slider)
- [`func SliderStep`](#func-sliderstep)
- [`func Switch`](#func-switch)
- [`func TabView`](#func-tabview)
- [`func TextArea`](#func-textarea)
- [`func TextGrid`](#func-textgrid)
- [`type CanvasScale`](#type-canvasscale)
    - [`func (CanvasScale) Apply`](#func-canvasscale-apply)
- [`type ContentMode`](#type-contentmode)
    - [`func ContentModes`](#func-contentmodes)
- [`type FillRule`](#type-fillrule)
- [`type GridRow`](#type-gridrow)
- [`type GridRun`](#type-gridrun)
- [`type LineCap`](#type-linecap)
- [`type LineJoin`](#type-linejoin)
- [`type Path`](#type-path)
    - [`func Circle`](#func-circle)
    - [`func Line`](#func-line)
    - [`func NewPath`](#func-newpath)
    - [`func Polyline`](#func-polyline)
    - [`func Rect`](#func-rect)
    - [`func Sector`](#func-sector)
    - [`func (*Path) Arc`](#func-path-arc)
    - [`func (*Path) Close`](#func-path-close)
    - [`func (*Path) CubicTo`](#func-path-cubicto)
    - [`func (*Path) LineTo`](#func-path-lineto)
    - [`func (*Path) MoveTo`](#func-path-moveto)
    - [`func (*Path) QuadTo`](#func-path-quadto)
- [`type SelectMenuItem`](#type-selectmenuitem)
- [`type SelectMenuSection`](#type-selectmenusection)
    - [`func SelectMenuSections`](#func-selectmenusections)
    - [`func (SelectMenuSection) First`](#func-selectmenusection-first)
- [`type SelectOption`](#type-selectoption)
    - [`func Option`](#func-option)
- [`type Shape`](#type-shape)
- [`type TabItem`](#type-tabitem)
    - [`func Tab`](#func-tab)
- [`type TabViewNode`](#type-tabviewnode)
- [`type TabViewProp`](#type-tabviewprop)
    - [`func Content`](#func-content)
    - [`func OnTabChange`](#func-ontabchange)
    - [`func SelectedIndex`](#func-selectedindex)
    - [`func Tabs`](#func-tabs)

## Constants

Wire opcodes. The numbers after each are its operands:

	0 x y                   move to
	1 x y                   line to
	2 x1 y1 x2 y2 x y       cubic Bézier to, via two control points
	3                       close the subpath

Held to the renderers by the canvas tests in mobile/verify and wasm/verify.

```go
const (
	PathMove  = 0
	PathLine  = 1
	PathCubic = 2
	PathClose = 3
)
```

<small>[core/canvas.go:290](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L290)</small>

GridRun attribute bits. A renderer without a native spelling for one may drop it (there is no dim on the web's font-weight scale, say, so the DOM targets fake it with opacity), but must never fail the row.

```go
const (
	GridBold      = 1 << iota // heavier weight
	GridDim                   // reduced intensity
	GridItalic                // slanted
	GridUnderline             // a line below
	GridStrike                // a line through
)
```

<small>[core/textgrid.go:67](https://github.com/rohanthewiz/grmob/blob/master/core/textgrid.go#L67)</small>

## Functions

### func Button

```go
func Button(label string, onClick func(), props ...PropsAndChildren) View
```

Button takes the same mixed argument list the inputs do — style props and behavior props in any order — rather than the \`...StyleProp\` it took originally:

	core.Button("Delete", onDelete,
	    core.BackgroundColor(ctx.Theme().Colors.Error),
	    core.OnLongPress(confirmDestructive),
	)

It was the last leaf that could not carry a behavior prop, which meant OnLongPress — a gesture a button is the most natural home for — was unreachable on the one node type that exists to be pressed. Widening it through leafNode closes that and puts every leaf on one argument contract.

The renderers had the matching half of that gap: both natives read the gesture off containers and leaves but not off a Button, because a Button draws its own control and does not go through the generic gesture path. Both now wire it on the button itself (a Surface + combinedClickable on Compose, a simultaneousGesture on SwiftUI), as does the DOM runtime, which synthesizes the gesture from pointer events.

The widening is source-compatible for the same reason the inputs' was: a StyleProp is a PropsAndChildren, so every existing core.Button(label, fn, core.Padding(8)) call compiles untouched. The shape it does break is forwarding — a \[]StyleProp cannot be spread into a ...PropsAndChildren — so a wrapper that collected style props into a slice has to widen its own slice to \[]core.PropsAndChildren. comps.Button and comps.Chip are the two in this tree that did.

See leafNode for the ordering and nil contracts, and for why a View passed here is a debug-mode concern rather than a silent no-op.

Button is deliberately absent from focusableLeafTypes: a phone does not give a button keyboard focus, so it carries no focus-command stamp and a core.Focus aimed at one would do nothing. FocusTarget still applies if an app wants the stamp anyway — see core/focus.go.

<small>[core/button.go:39](https://github.com/rohanthewiz/grmob/blob/master/core/button.go#L39)</small>

### func ButtonWithEvent

```go
func ButtonWithEvent(label string, event string, handler func(), props ...PropsAndChildren) View
```

ButtonWithEvent is Button with the event name chosen by the caller, for the gestures that have no dedicated builder. It is largely superseded by the widening above — core.Button(label, fn, core.On("LongPress", g)) says the same thing and keeps the click — but it stays because it is the only way to build a button whose \*only\* wiring is a non-click event.

<small>[core/button.go:58](https://github.com/rohanthewiz/grmob/blob/master/core/button.go#L58)</small>

### func Canvas

```go
func Canvas(w, h float64, shapes []Shape, props ...PropsAndChildren) View
```

Canvas draws vector shapes — lines, curves, arcs, filled polygons — in a coordinate space of its own, scaled into whatever box the layout gives it.

	p := core.NewPath().MoveTo(0, 80).LineTo(40, 20).LineTo(100, 50)

	core.Canvas(100, 100, []core.Shape{
	    {Path: core.Circle(50, 50, 48), Fill: t.Colors.Surface},
	    {Path: p, Stroke: t.Colors.Primary, StrokeWidth: 2, Cap: core.CapRound},
	}, core.Width("100%"))

It is the one primitive in core that can join two arbitrary points, which is what line, area, pie and scatter charts are made of. Everything else in the vocabulary is a flex box, and a box can only fake a diagonal by turning.

#### The coordinate space is not the size

w × h is a viewBox: the units the shapes are written in, with the origin at the top-left and y growing downwards (screen convention, which is what all three platforms' drawing APIs use). The node's size on screen comes from its Style like any other node's, and the drawing is mapped onto that box by the CanvasScale prop:

	CanvasFit      one scale for both axes, centred — a clock face stays round
	               in a wide box. The default. SVG `xMidYMid meet`.
	CanvasStretch  each axis scaled on its own — a line chart spans a wide box.
	               SVG `none`.

A Canvas whose Style gives no Height takes the viewBox's aspect ratio, and one that gives no Width fills its parent's width, so the common case — a chart as wide as the screen — needs no sizing props at all.

#### Strokes are in layout units, never scaled

A StrokeWidth of 2 is 2 px (dp, pt) whatever the scale, on every target. Scaled strokes would make a stretched chart's lines fat on one axis and thin on the other, and would make the same chart's lines change weight between a phone and a tablet. SVG says this with vector-effect: non-scaling-stroke; the natives transform the path's points and stroke the result untransformed, which is the same thing.

#### Four opcodes on the wire

Path offers moves, lines, quadratic and cubic Béziers, and circular arcs, but what reaches a renderer is only move, line, cubic and close. Quadratics are raised to cubics exactly, and arcs are approximated by cubic segments of at most 90° (error under 0.03% of the radius) — in Go, once.

The alternative was teaching three languages four arc conventions: SVG's endpoint form with its large-arc and sweep flags, Compose's bounding rectangle with degrees, SwiftUI's centre with radians and a clockwise flag that is flipped in a y-down space. Each is a place for the three targets to disagree by a sign, and a disagreement there draws a plausible wrong picture rather than failing. Every renderer already has moveTo, lineTo, cubicTo and close with identical meaning, so that is the whole contract.

#### One child per shape

A Canvas is a container of CanvasShape nodes, the shape TextGrid has with its rows and for the same reason: the reconciler pairs children by index and compares props by value, so a pass that moves one hand of a clock sends one update-props patch and leaves the face alone. Shapes are drawn in order, so a later shape paints over an earlier one.

Shapes take no props of their own and are never built directly by app code. Behavior props (OnClick, ...) apply to the canvas as a whole.

#### Accessibility

A drawing has no text a reader could find in it. A Canvas without an AccessibilityLabel is treated as decoration and hidden; one with a label is a single image element that speaks it. A chart should always be given one that states what the chart shows, not that it is a chart.

#### Not in v1

Text inside the drawing (lay labels out around it as Text nodes), gradients, clipping and per-shape hit-testing. Fills use the nonzero rule, every target's default, unless a shape asks for FillEvenOdd.

<small>[core/canvas.go:83](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L83)</small>

### func CanvasMapping

```go
func CanvasMapping(w, h, boxW, boxH float64, scale CanvasScale) (sx, sy, ox, oy float64)
```

CanvasMapping is how a w × h viewBox lands in a boxW × boxH box under a scale: a viewBox point (x, y) is drawn at (x·sx + ox, y·sy + oy).

	CanvasStretch  sx = boxW/w, sy = boxH/h, no offset
	CanvasFit      sx = sy = the smaller of the two, and the slack on the
	               other axis split evenly — SVG's "xMidYMid meet"

The web targets never call this; they hand the viewBox to SVG, which applies the same rule itself. It is the statement the two native renderers restate (canvasViewport in GrMobCanvasGeometry.kt, the Swift equivalent), and internal/canvasfixture holds them to it. A non-positive viewBox side reads as 100, which is what Canvas writes for one.

<small>[core/canvas.go:154](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L154)</small>

### func Checkbox

```go
func Checkbox(checked bool, onToggle func(bool), props ...PropsAndChildren) View
```

<small>[core/input.go:50](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L50)</small>

### func Image

```go
func Image(src string, styleProps ...StyleProp) View
```

Image renders a remote or bundled image at the default content mode (ContentModeFit). Use ImageWithMode to choose another.

<small>[core/image.go:92](https://github.com/rohanthewiz/grmob/blob/master/core/image.go#L92)</small>

### func ImageWithMode

```go
func ImageWithMode(src string, mode ContentMode, styleProps ...StyleProp) View
```

ImageWithMode is Image plus an explicit ContentMode.

A separate builder rather than a variadic change to Image, matching InputWithSubmit: every existing Image call site keeps compiling and keeps its current rendering, and the mode stays a required, visible argument at the sites that care rather than an option buried in a style list.

<small>[core/image.go:102](https://github.com/rohanthewiz/grmob/blob/master/core/image.go#L102)</small>

### func Input

```go
func Input(value string, placeholder string, onChange func(string), props ...PropsAndChildren) View
```

<small>[core/input.go:23](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L23)</small>

### func InputPassword

```go
func InputPassword(value string, placeholder string, onChange func(string), props ...PropsAndChildren) View
```

<small>[core/input.go:59](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L59)</small>

### func InputWithSubmit

```go
func InputWithSubmit(value string, placeholder string, onChange func(string), onSubmit func(), props ...PropsAndChildren) View
```

InputWithSubmit is Input plus a submit action: pressing the keyboard's return key (iOS) or IME done action (Android) dispatches onSubmit. The submit rides the existing void-callback channel — the renderers read the "onSubmit" prop and dispatch it exactly like a Button's onClick — so the bridge surface is unchanged. A separate builder rather than a variadic change to Input keeps every existing call site compiling untouched.

<small>[core/input.go:39](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L39)</small>

### func NumericInput

```go
func NumericInput(value int, onChange func(int), props ...PropsAndChildren) View
```

<small>[core/input.go:69](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L69)</small>

### func OnSliderChangeEnd

```go
func OnSliderChangeEnd(fn func(float64)) BehaviorProp
```

OnSliderChangeEnd fires once when the drag ends, with the final value — see Slider for why a seek bar wants this rather than onChange.

<small>[core/slider.go:67](https://github.com/rohanthewiz/grmob/blob/master/core/slider.go#L67)</small>

### func Select

```go
func Select(value string, options []SelectOption, onChange func(string), props ...PropsAndChildren) View
```

Select is the picker: one value chosen from a fixed list.

	core.Select(form.Country, []core.SelectOption{
	    {Value: "us", Label: "United States"},
	    {Value: "pt", Label: "Portugal"},
	}, func(v string) { form.Country = v })

Controlled, like every other input here: the value shown is always the one Go passed, and a change goes up as an event. onChange carries the option's \*Value\*, never its label or its index — the index is the one identity that changes when the list is reordered, and a label is written to be read.

#### What each target draws

	web       a <select>, whose options are a prop rather than child nodes
	iOS       a Menu whose label is the chosen option's text
	Android   a Box anchored to a DropdownMenu, same shape

The natives are deliberately \*not\* built from a platform picker control (SwiftUI's .pickerStyle(.menu), Material's ExposedDropdownMenuBox). Both of those draw a frame of their own, and the whole rule this widget lands under is that the Go style owns the frame — see borderResetTypes in htmlout/tag.go, which the web half joins for the same reason. A control whose edge came from the platform on two targets and from the theme on two others is the divergence that rule exists to prevent.

#### It reads the theme's Input base

A picker is a field: it sits in a form beside text inputs, and a picker that did not match the fields around it would look like a mistake. Reading Components.Input is also how it inherits the frame those fields grew — the same move comps.DatePicker makes for the same reason, and the reason this widget needs no palette role of its own.

#### Options are a prop, not children

A \<select>'s options are elements, but they are not \*nodes\*: no patch is ever addressed to one, they carry no style, and they cannot hold a subtree. Sending them as children would put four renderers in the business of deciding which child is chrome, which is the complication core.TabView's tabs prop already avoids one node type over. The web renderer builds the \<option> elements from the prop; both natives read the same list.

#### Grouped and disabled options

SelectOption carries a Group, a Disabled and a GroupDisabled beside its two required fields; see the type. All three are drawn by every target — an \<optgroup>, a disabled \<option> and \<optgroup disabled> on the web; a Section and a disabled Button in the iOS menu; a heading item and a disabled item in the Android dropdown.

What a heading still cannot carry is an icon, and that is a decision rather than a gap: an \<optgroup>'s label is an attribute, so the web can hold text and nothing else. A heading with an icon on two targets and without one on the other two is the divergence this widget refuses everywhere else — the same argument that keeps it off the platform picker controls, one property down.

Neither reaches this function as anything but a map key, which is the point: the flattening below is the one place that knows what a SelectOption is, and the four renderers each read a list of flat string maps. A fifth field would land here and nowhere else.

What the renderers do \*not\* each decide is which options form which run. SelectMenuSections (select\_menu.go) takes the flattened list and answers that once; htmlout calls it, and the two natives carry transliterations that ios/verify checks against it.

<small>[core/input.go:280](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L280)</small>

### func Slider

```go
func Slider(value, min, max float64, onChange func(float64), props ...PropsAndChildren) View
```

Slider is a horizontal value control: a thumb on a track, dragged to choose a number in \[min, max]. A seek bar, a volume, a brightness, a price ceiling.

	core.Slider(pos, 0, duration, func(v float64) { scrub.Set(v) },
	    core.OnSliderChangeEnd(func(v float64) { core.AudioSeek(v) }))

Each platform draws its own: Compose's Material 3 Slider, SwiftUI's Slider, and \<input type="range"> in the browser and in htmlout.

#### Two callbacks, and why the second is the important one

onChange fires continuously while the thumb moves — the value under the finger, dozens of times a second. That is right for a label that follows the drag and wrong for anything expensive or irreversible: a seek on a network stream, a request to a server. OnSliderChangeEnd fires once, when the finger lifts, with the final value — and it is the one a seek bar acts on. onChange may be nil when only the end matters.

Both cross the bridge as text callbacks carrying the number formatted with strconv (the natives' own float formatting is accepted too: "0.5", "1.0E-4" and "1e-4" all parse). A value that fails to parse is dropped, the same policy NumericInput applies.

#### The control is controlled

Like every leaf, the value shown is the one Go rendered — but a drag has to feel immediate, and the Go round trip is asynchronous, so the native renderers show the finger's value \*while dragging\* and Go's value otherwise (the same compromise the text fields make). A seek bar fed by a status tick therefore never snaps the thumb back under the finger.

<small>[core/slider.go:38](https://github.com/rohanthewiz/grmob/blob/master/core/slider.go#L38)</small>

### func SliderStep

```go
func SliderStep(step float64) BehaviorProp
```

SliderStep snaps the thumb to multiples of step from min. 0 (the default) is continuous.

<small>[core/slider.go:81](https://github.com/rohanthewiz/grmob/blob/master/core/slider.go#L81)</small>

### func Switch

```go
func Switch(on bool, onToggle func(bool), props ...PropsAndChildren) View
```

Switch is the instant-effect boolean: a track and a thumb, flipped to turn one thing on or off right now. Airplane mode, notifications, dark theme.

	core.Switch(settings.Notify, func(on bool) { settings.SetNotify(on) })

Each platform draws its own — Material 3's Switch on Android, SwiftUI's Toggle on iOS, and an \<input type="checkbox" switch role="switch"> in the browser and in htmlout.

#### Why this is a node type and not a flag on Checkbox

The two controls look similar in a props list and are not interchangeable on screen, and the difference is \*when the choice takes effect\*. A checkbox collects a value that something else will act on — a form's "remember me", a row's selection, a terms box above a Submit button — so a tick that sits there unacted-on is the expected state. A switch acts on the tap: there is no Submit, and a switch that needed one would be read as broken.

That is a platform-idiom difference rather than a styling one, which is what makes it a type. Both natives draw \*both\* controls, and they are different controls there (Compose's Checkbox and Switch; on iOS the platform has no checkbox at all and Toggle stands in for one — see GrMobCheckbox in the SwiftUI renderer). A bool prop on Checkbox would have the reconciler swapping one platform control for another inside one node's update-props patch, which is exactly the work a node type does properly: a changed type is a replace, and a replace is how a control is exchanged.

#### The state crosses the wire as "checked"

Go says \`on\` because a switch is on, and the wire says \`checked\` because that is what the DOM calls it — the same split core.SelectedState makes when it spells a bool "true" in a prop map.

It is not only tidiness. Both web renderers already carry a \`checked\` prop: htmlout writes the boolean attribute from it and the WASM runtime assigns el.checked from it on create \*and\* on update-props. Naming the prop \`on\` would have meant a second spelling of both halves in both renderers — four new branches whose only job is to mean what an existing branch already means — and the update half is the one that would have been forgotten, because a switch drawn correctly on the first render and frozen thereafter looks like a working widget until somebody changes its value from Go.

#### What announces it, and why the role is not a core.Role

HTML has no switch control. It has a \*switch attribute\* on a checkbox (WHATWG HTML; Safari draws it, most browsers do not yet), and it has role="switch", which tells a screen reader what this is in every browser regardless. Both are written, so the control announces itself correctly everywhere and is drawn correctly where the browser can — and where it cannot, it degrades to a checkbox, which is the same bool in the same state.

That role is written from the \*node type\*, with no Style involved, which makes this the second such node after Modal's dialog — see htmlout.CarriesOwnRole. It is deliberately not a value in the Role vocabulary: every Role a caller can spell obliges all four renderers to grow an arm for it (core.Roles() is held against each renderer's dispatch in mobile/verify), and there is nothing for the natives to do here. A Material Switch and a SwiftUI Toggle announce themselves as switches already. \`switch\` therefore joins aria/spec.NearMisses for the reason \`dialog\` is there: a role this framework emits and does not name.

#### It reads the theme's CheckBox base

The same base the other boolean control reads, and not a field of its own. What that style actually contributes is geometry and display — both natives read only margin and size off a control's style (marginAndSize in the Compose renderer, marginAndSizeOnly in SwiftUI), and on the web a control drawn by the user agent ignores a fill. A Components.Switch field would therefore be a palette entry no palette could spend, measured by the contrast census as though some surface were drawn from it.

Like Checkbox it carries no label: a control's label is the caller's, and comps.FormField and comps.InputRow already own that slot.

#### Keyboard focus

Not in focusableLeafTypes, for Checkbox's reason one file over: neither native renderer gives one keyboard focus, so the stamp would emit a patch per focus command that nothing on the far side reads. A browser focuses the \<input> for free, as it does a checkbox's.

<small>[core/switch.go:84](https://github.com/rohanthewiz/grmob/blob/master/core/switch.go#L84)</small>

### func TabView

```go
func TabView(props ...TabViewProp) View
```

<small>[core/tabview.go:19](https://github.com/rohanthewiz/grmob/blob/master/core/tabview.go#L19)</small>

### func TextArea

```go
func TextArea(value string, onChange func(string), rows int, props ...PropsAndChildren) View
```

<small>[core/input.go:334](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L334)</small>

### func TextGrid

```go
func TextGrid(rows []GridRow, props ...PropsAndChildren) View
```

TextGrid is a monospace grid of styled text: a terminal pane, a log tail, a hex dump. Rows are given in order; each row is a run of styled spans that the renderer lays out in a fixed-pitch font with no wrapping.

	core.TextGrid(rows, core.FontSize(12), core.Background("#000"))

#### Why a node type and not a Column of Text

Nothing else in core can draw this. Style has no font family, so a Text cannot ask for a fixed pitch, and a row of Text nodes has no way to keep its glyphs on a cell grid across styled runs. Emulating a grid from Row and Text would also put every run in the node tree as its own element, which for an 80×24 pane at terminal diff rate is a patch stream measured in thousands of nodes per second.

#### Rows are children, so a changed row is one patch

The grid renders as a container node with one GridRow child per row, and each row's runs are one prop on that child. The reconciler pairs children by index and compares props by value, so a pass that changes three rows of twenty-four emits three update-props patches and nothing else; an unchanged row costs a DeepEqual on its runs and no traffic. A renderer therefore repaints a row, never the grid. No caching is needed to get this; it falls out of the node shape.

Each platform draws its own: Compose an AnnotatedString in a monospace Text per row, SwiftUI an AttributedString with the monospaced design, the browser and htmlout a \<pre> of \<div> rows holding \<span> runs.

The Style applies to the grid as a whole (FontSize, TextColor and Background are the ones that matter; a run's own colours override the grid's). Behavior props apply to the grid too, so a tap on any cell is a tap on the grid.

<small>[core/textgrid.go:36](https://github.com/rohanthewiz/grmob/blob/master/core/textgrid.go#L36)</small>

## Types

### type CanvasScale

```go
type CanvasScale string
```

CanvasScale says how a Canvas's viewBox is mapped onto its box. It is passed among a Canvas's props like a StyleProp:

	core.Canvas(200, 100, shapes, core.CanvasStretch, core.Height("160px"))

<small>[core/canvas.go:118](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L118)</small>

```go
const (
	// CanvasFit scales both axes by the same factor, the largest that fits
	// the whole drawing in the box, and centres it. The default.
	CanvasFit CanvasScale = "fit"

	// CanvasStretch scales each axis independently so the viewBox exactly
	// fills the box. Shapes distort; strokes do not (see Canvas).
	CanvasStretch CanvasScale = "stretch"
)
```

#### func (CanvasScale) Apply

```go
func (c CanvasScale) Apply(_ *Context, n *Node)
```

Apply makes a CanvasScale a BehaviorProp: it writes a node prop rather than a Style field, because it means nothing to any node but a Canvas.

<small>[core/canvas.go:132](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L132)</small>

### type ContentMode

```go
type ContentMode string
```

ContentMode says how an image's intrinsic aspect ratio is reconciled with the box the layout gave it. It is the one image property that is genuinely not styling: every renderer expresses it through the image view's own API (SwiftUI's content mode, Compose's ContentScale, CSS object-fit), not through the box modifiers, so it travels as a node prop.

The four values are the intersection all three targets can express exactly:

	         | fits inside | fills box | ratio kept
	---------+-------------+-----------+-----------
	Fit      | yes         | no        | yes
	Fill     | no (crops)  | yes       | yes
	Stretch  | no          | yes       | no
	Center   | no          | no        | yes (1:1 pixels)

Fit is the default — it is what core.Image has always rendered as, so an existing call site keeps its layout — and is also the safe default: it is the only mode that never crops and never distorts.

<small>[core/image.go:21](https://github.com/rohanthewiz/grmob/blob/master/core/image.go#L21)</small>

```go
const (
	// ContentModeFit scales the image down until it fits entirely inside the
	// box, preserving the aspect ratio and leaving empty space on the axis
	// that ran out first. CSS `object-fit: contain`.
	ContentModeFit ContentMode = "fit"

	// ContentModeFill scales the image up until it covers the box, preserving
	// the aspect ratio and cropping the overflow on the longer axis. The mode
	// for avatars, hero images and thumbnails — anything where empty space
	// would be worse than losing an edge. CSS `object-fit: cover`.
	//
	// The crop is real on every target: CSS object-fit clips, Compose's
	// ContentScale.Crop clips, and the SwiftUI path adds an explicit
	// .clipped() — an unclipped image would paint over its siblings.
	ContentModeFill ContentMode = "fill"

	// ContentModeStretch distorts the image to exactly the box's dimensions,
	// ignoring the aspect ratio. CSS `object-fit: fill`. Rarely what a design
	// wants; included because the platforms offer it and because a Stretch
	// spelled out beats an app pre-scaling its assets.
	ContentModeStretch ContentMode = "stretch"

	// ContentModeCenter draws the image at its intrinsic size, centered, with
	// no scaling in either direction — larger than the box means it is
	// cropped, smaller means it is surrounded by space. CSS `object-fit:
	// none`. For pixel-exact assets (icons, QR codes) that scaling would blur.
	ContentModeCenter ContentMode = "center"
)
```

#### func ContentModes

```go
func ContentModes() []ContentMode
```

ContentModes returns every declared ContentMode, in declaration order.

Go cannot enumerate the constants of a named string type at run time, so the set has to be written out a second time — and a second copy of a list is exactly the thing that goes stale. This one is pinned to the const block above by TestContentModesMatchTheDeclaredConstants, which reads them out of this file's syntax tree, so adding a constant without adding it here fails \`go test ./...\` rather than silently shrinking the set.

It exists because four renderers each map these modes onto their own vocabulary — CSS object-fit in htmlout and the WASM runtime, SwiftUI scaling in Renderer.swift, Compose's ContentScale in Renderer.kt — and none of them can be asked "did you cover every mode?" without a list to check against. All four are now held to it:

	htmlout.ObjectFits          htmlout/objectfit_test.go
	the WASM runtime's copy     wasm/verify/objectfit_test.go (via htmlout)
	Renderer.swift              mobile/verify/contentmode_test.go
	Renderer.kt                 mobile/verify/contentmode_test.go

The first two are table comparisons — both sides map a mode onto the same CSS keyword, so the values can be compared as well as the keys. The natives map onto SwiftUI and Compose vocabularies that share nothing with CSS or with each other, so only the key set is comparable; those two checks read the arms out of the native source and check coverage alone.

A fresh slice per call rather than a package-level var: a var of slice type is writable by any importer, and four elements are cheaper to build than to defend.

<small>[core/image.go:81](https://github.com/rohanthewiz/grmob/blob/master/core/image.go#L81)</small>

### type FillRule

```go
type FillRule string
```

FillRule is how a fill decides whether a point is inside a path whose subpaths overlap or wind around each other.

	nonzero   inside when the path winds around the point a nonzero number
	          of times, counting direction: a ring whose two circles run the
	          same way fills solid, and one whose inner circle runs backwards
	          has a hole
	evenodd   inside when a ray from the point crosses the path an odd number
	          of times, ignoring direction: any inner subpath is a hole

Even-odd is the one to reach for when a shape has holes and its subpaths come from code that does not track direction, as core.Path's Arc and a data-built outline do not. Every target has both:

	SVG       fill-rule="nonzero" | "evenodd"
	Compose   PathFillType.NonZero | PathFillType.EvenOdd
	SwiftUI   FillStyle(eoFill: false | true)

<small>[core/canvas.go:275](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L275)</small>

```go
const (
	FillNonZero FillRule = "nonzero"
	FillEvenOdd FillRule = "evenodd"
)
```

### type GridRow

```go
type GridRow []GridRun
```

GridRow is one row of a TextGrid: its runs, in order, left to right.

<small>[core/textgrid.go:48](https://github.com/rohanthewiz/grmob/blob/master/core/textgrid.go#L48)</small>

### type GridRun

```go
type GridRun struct {
	Text string `json:"t"`
	Fg   string `json:"fg,omitempty"`
	Bg   string `json:"bg,omitempty"`
	Attr int    `json:"a,omitempty"`
}
```

GridRun is a span of one row drawn in one style. Text is the glyphs; Fg and Bg are CSS colours ("#rrggbb"), each "" to inherit the grid's; Attr is a bitmask of the Grid\* attributes.

The json tags are the wire shape the renderers read. They are short because a full pane is a few thousand runs a second at diff rate, and the key names are the part of a run that is not content.

<small>[core/textgrid.go:57](https://github.com/rohanthewiz/grmob/blob/master/core/textgrid.go#L57)</small>

### type LineCap

```go
type LineCap string
```

LineCap is how an open stroke's ends are drawn.

<small>[core/canvas.go:170](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L170)</small>

```go
const (
	CapButt   LineCap = "butt" // flush with the end point; the default
	CapRound  LineCap = "round"
	CapSquare LineCap = "square"
)
```

### type LineJoin

```go
type LineJoin string
```

LineJoin is how a stroke turns a corner. A miter longer than 4× half the stroke width is cut to a bevel on every target — SVG's and Compose's default miter limit, pinned explicitly on iOS, whose own default is 10.

<small>[core/canvas.go:181](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L181)</small>

```go
const (
	JoinMiter LineJoin = "miter" // the default
	JoinRound LineJoin = "round"
	JoinBevel LineJoin = "bevel"
)
```

### type Path

```go
type Path struct {
	// contains filtered or unexported fields
}
```

Path is a sequence of subpaths built by chained calls. The builder methods mutate and return the receiver, so a Path should be finished before it is handed to a Shape: the node takes a copy when the Canvas renders, and a change after that is invisible (see Node immutability).

<small>[core/canvas.go:301](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L301)</small>

#### func Circle

```go
func Circle(cx, cy, r float64) *Path
```

Circle is a closed circle, drawn clockwise from three o'clock.

<small>[core/canvas.go:468](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L468)</small>

#### func Line

```go
func Line(x1, y1, x2, y2 float64) *Path
```

Line is a single straight segment.

<small>[core/canvas.go:448](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L448)</small>

#### func NewPath

```go
func NewPath() *Path
```

NewPath returns an empty path.

<small>[core/canvas.go:312](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L312)</small>

#### func Polyline

```go
func Polyline(xy ...float64) *Path
```

Polyline joins the points (x0, y0, x1, y1, ...) with straight segments. An odd trailing coordinate is ignored.

<small>[core/canvas.go:454](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L454)</small>

#### func Rect

```go
func Rect(x, y, w, h float64) *Path
```

Rect is a closed rectangle with its top-left corner at (x, y).

<small>[core/canvas.go:463](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L463)</small>

#### func Sector

```go
func Sector(cx, cy, inner, outer, startDeg, sweepDeg float64) *Path
```

Sector is a closed ring segment between radii inner and outer — a pie wedge when inner is 0, a donut segment otherwise. Angles are as for Path.Arc.

<small>[core/canvas.go:475](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L475)</small>

#### func (*Path) Arc

```go
func (p *Path) Arc(cx, cy, r, startDeg, sweepDeg float64) *Path
```

Arc draws part of a circle centred on (cx, cy) with radius r, starting at startDeg and sweeping sweepDeg. Angles are degrees clockwise from the positive x-axis (three o'clock) — clockwise on screen, the same sense as core.Rotate — and a negative sweep goes anticlockwise.

If the path has a current point, a straight line joins it to the arc's start, so a pie wedge is MoveTo(centre).Arc(...).Close(). Otherwise the arc starts a new subpath.

##### How it is approximated

The sweep is split into segments of at most 90°, and each becomes the cubic whose control points lie on the tangents at its ends, at distance k = 4/3 · tan(θ/4) · r. That k makes the curve's midpoint lie exactly on the circle; the worst radial error for a 90° segment is about 0.027% of r, below a pixel for any radius a phone can show.

	P1 ──k── C1
	           ╲         C1 = P1 + k · tangent(P1)
	            C2       C2 = P2 − k · tangent(P2)
	             │
	             P2

<small>[core/canvas.go:378](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L378)</small>

#### func (*Path) Close

```go
func (p *Path) Close() *Path
```

Close joins the current point back to the start of the subpath. A later LineTo continues from that start, as it does on every platform.

<small>[core/canvas.go:416](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L416)</small>

#### func (*Path) CubicTo

```go
func (p *Path) CubicTo(x1, y1, x2, y2, x, y float64) *Path
```

CubicTo draws a cubic Bézier to (x, y) via control points (x1, y1) and (x2, y2).

<small>[core/canvas.go:335](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L335)</small>

#### func (*Path) LineTo

```go
func (p *Path) LineTo(x, y float64) *Path
```

LineTo draws a straight segment to (x, y). With no current point it moves there instead, which is what every platform's API does and what makes a polyline a single loop body.

<small>[core/canvas.go:324](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L324)</small>

#### func (*Path) MoveTo

```go
func (p *Path) MoveTo(x, y float64) *Path
```

MoveTo starts a new subpath at (x, y).

<small>[core/canvas.go:315](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L315)</small>

#### func (*Path) QuadTo

```go
func (p *Path) QuadTo(qx, qy, x, y float64) *Path
```

QuadTo draws a quadratic Bézier to (x, y) via control point (qx, qy). It is sent as the cubic that traces the identical curve: each cubic control point sits two thirds of the way from an end point to the quadratic one.

<small>[core/canvas.go:347](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L347)</small>

### type SelectMenuItem

```go
type SelectMenuItem struct {
	Index    int
	Value    string
	Label    string
	Disabled bool
}
```

SelectMenuItem is one choosable row of a picker's menu: an option, resolved out of the flat wire map into the three things every renderer asks it.

Index is the option's position in the original list. It is carried because a menu row needs an identity that survives two options sharing a label — which core.Select explicitly allows, since the Value is the identity and the Label is written to be read — and because a renderer that keys its rows on the map itself cannot: a map is not hashable in Swift and not comparable in Go.

<small>[core/select_menu.go:51](https://github.com/rohanthewiz/grmob/blob/master/core/select_menu.go#L51)</small>

### type SelectMenuSection

```go
type SelectMenuSection struct {
	Heading string

	// Disabled marks the whole run unavailable — core.SelectOption's
	// GroupDisabled, resolved. See that field for what states it and why any
	// one option in the run is enough.
	//
	// Every Item of a disabled section is itself Disabled, which is not a
	// convenience: it is the only mechanism two of the four targets have.
	// SwiftUI puts `.disabled` on the Button and never on the Section (see
	// grMobMenuItems), Material's dropdown has no section construct at all,
	// and on the web a run with no heading has no <optgroup> to carry the
	// attribute. So this field is what a renderer reads to grey the *heading*
	// — and, on the web, to write <optgroup disabled> once instead of the
	// attribute N times — while the refusal itself always rides on the items.
	Disabled bool

	Items []SelectMenuItem
}
```

SelectMenuSection is one run of consecutive options sharing a heading.

Heading is empty for the options that stand on their own at the top level, and such a run is a real section rather than an absence of one: every renderer needs somewhere to put those options, and giving them a section with no heading means the drawing code is one loop over sections rather than a loop with a special case in it.

<small>[core/select_menu.go:65](https://github.com/rohanthewiz/grmob/blob/master/core/select_menu.go#L65)</small>

#### func SelectMenuSections

```go
func SelectMenuSections(options []map[string]string) []SelectMenuSection
```

SelectMenuSections splits a picker's flattened options into the runs core.SelectOption.Group describes.

Runs, not a gather: consecutive options sharing a heading are one section, in the order they were written, and the same heading either side of a different one is two sections. The field's own doc carries the argument — the list's order is the caller's, and no renderer could undo a reordering.

The loop closes a run when the next option names a different heading, and flushes the last one after the loop, which is the only bookkeeping the rule needs. An empty list gives an empty slice rather than nil, so a renderer can range over the result without a guard.

##### Why the flush is now more than an append

A run's Disabled is a property of the \*whole\* run — any option carrying core.SelectOption.GroupDisabled sets it — so it is not known until the run is closed, and closing it has to walk back over the items already collected to disable them. That is the second thing the flush does, and it is why closing a run is a named step here rather than an inline append: a rule with two halves that can be half-remembered is exactly the shape this file exists to hold in one place.

<small>[core/select_menu.go:122](https://github.com/rohanthewiz/grmob/blob/master/core/select_menu.go#L122)</small>

#### func (SelectMenuSection) First

```go
func (s SelectMenuSection) First() int
```

First is the index of the section's first option, which is what identifies the section to a renderer that needs a key.

The heading cannot be that key: core.SelectOption.Group allows the same heading either side of a different one, and that is two sections. The index can, because a section is a contiguous run and no two runs start in the same place. A section with no items cannot occur — SelectMenuSections opens one only when it has an option to put in it — so the read is total.

<small>[core/select_menu.go:93](https://github.com/rohanthewiz/grmob/blob/master/core/select_menu.go#L93)</small>

### type SelectOption

```go
type SelectOption struct {
	Value string
	Label string

	// Group is the heading this option is filed under: a country's continent,
	// a font's family, "Recently used" above the rest. Empty means the option
	// stands on its own at the top level of the list, which is what every
	// option did before this field existed.
	//
	// # Consecutive options with the same Group form one section
	//
	// Runs, not a gather. Two options naming "Europe" with an American one
	// between them make *two* Europe sections, in the order they were written.
	//
	// That is the honest reading and the only one this widget can offer. The
	// list's order is the caller's — it is what a person sees and what the
	// keyboard walks — and a gather would silently reorder it to suit the
	// headings, which is a bigger change than the one being asked for and one
	// no renderer could undo. Sorting a list into its sections is a line of Go
	// at the call site; un-sorting one is not.
	//
	// Each target draws a run as its own construct: an <optgroup> on the web,
	// a Section in the iOS menu, a heading item in the Android dropdown. All
	// three are labels rather than options — none of them is selectable, and
	// none of them carries a Value.
	//
	// Which options form which run is decided once, by SelectMenuSections
	// (select_menu.go), and not by each renderer — see that file for what four
	// copies of this rule cost.
	//
	// # A heading with nothing under it cannot be written
	//
	// This field is a property of an *option*, so a section with no options
	// has nothing to declare it: a run exists because some option named it.
	// SelectMenuSections therefore never produces an empty section, which
	// SelectMenuSection.First relies on and TestAnEmptySectionIsUnreachable
	// pins.
	//
	// That is a limit rather than an oversight. Declaring a heading
	// independently means a second list beside the options, and then a rule
	// for matching the two — which headings are in use, what a heading with no
	// matching option does, what an option naming a heading that is not in the
	// list does. The run-based reading was chosen precisely to have no
	// matching problem in it, and an empty section is the one thing that
	// reading cannot express. Nothing has asked for it: every real request has
	// been "this category is empty, say so", which is not an empty section at
	// all.
	//
	// What to write instead is a placeholder option, disabled:
	//
	//	{Group: "Archive", Label: "Nothing archived yet", Disabled: true}
	//
	// It is better than an empty section on every target rather than merely
	// possible: an <optgroup> with no <option> in it, a SwiftUI Section with
	// no Button and a Compose heading with no rows are each a label a screen
	// reader announces and a pointer cannot reach, and none of them says why
	// the category is empty. A disabled row says it in the caller's own words,
	// in the place a person is already looking. internal/menufixture carries
	// the shape, so all four picker menus are checked against it.
	Group string

	// Disabled greys this option out: visible, announced, and not choosable.
	// The plan a caller has outgrown, the size that is out of stock, the
	// timezone their region does not offer.
	//
	// Distinct from leaving the option out, which is the alternative and is
	// usually worse: an option that vanishes takes its explanation with it,
	// and a list that changes length between renders is one a person has to
	// re-read. A disabled option says *this exists and you cannot have it*.
	//
	// It does not stop Go from being handed the value. Every target refuses
	// the tap or the click, so nothing reaches onChange through the control —
	// but a Select is controlled, and an app that sets its own state to a
	// disabled option's value will find the widget showing it, because the
	// value shown is always the one Go passed. That is the same contract an
	// out-of-list value lands under; see Select.
	Disabled bool

	// GroupDisabled marks this option's whole *run* unavailable: the paid
	// plans on a free account, a shipping tier this address cannot use, a
	// "Coming soon" family that is worth showing and not worth offering.
	//
	// # Any option in the run is enough
	//
	// The declaration is read off every option, not off the first one. A
	// caller writing it on the second entry of a run and getting nothing would
	// have no way to find that out — a menu is drawn behind a tap, there is no
	// error channel here, and the option would look exactly like an option
	// that had been read. Making any one of them decide is the reading with no
	// silent failure in it.
	//
	// The cost is that a run's state is not known until the run is closed,
	// which is real bookkeeping: SelectMenuSections walks back over the run's
	// items when it flushes one. That is a cost paid once, in the authority,
	// which is the reason the authority exists.
	//
	// # It is not the same as disabling every option by hand
	//
	// Marking each option Disabled refuses each tap and says nothing about the
	// heading, which stays as legible as the ones above it. GroupDisabled
	// carries to the section — SelectMenuSection.Disabled — so the *heading*
	// can be greyed too, and so the web can write <optgroup disabled> once
	// rather than an attribute per option.
	//
	// Every item of a disabled run is still marked Disabled on its way out, so
	// the refusal reaches the two targets that have no section-level control
	// at all. See SelectMenuSection.Disabled.
	//
	// On an ungrouped run (Group empty) there is no heading to grey and, on
	// the web, no <optgroup> to carry the attribute — so it degrades to
	// exactly "every option in the run is disabled", which is the honest
	// answer rather than a special case.
	GroupDisabled bool
}
```

SelectOption is one choice in a Select: the value the app works in, and the label a person reads.

Two fields rather than a bare string because the two are different things often enough to be worth the type — a country code and a country name, a status enum and a sentence — and a widget that took only strings would push every caller into keeping a parallel slice. An empty Label means "the value is readable enough", which is the common small case (a list of sizes, a list of years) and keeps that case a one-word literal.

<small>[core/input.go:93](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L93)</small>

#### func Option

```go
func Option(value, label string) SelectOption
```

Option builds a SelectOption, mirroring Tab's constructor next door.

<small>[core/input.go:209](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L209)</small>

### type Shape

```go
type Shape struct {
	Path *Path

	// Fill and Stroke are CSS colours ("#rrggbb", "#rrggbbaa"); "" for none.
	Fill   string
	Stroke string

	// FillRule decides which regions of a self-overlapping path Fill paints;
	// the zero value is FillNonZero. See FillRule.
	FillRule FillRule

	// StrokeWidth is in layout units, not viewBox units; 0 means 1.
	StrokeWidth float64

	Cap  LineCap
	Join LineJoin

	// Dash alternates dash and gap lengths, in layout units like StrokeWidth.
	// Nil for a solid line.
	Dash []float64
}
```

Shape is one path drawn once: filled, stroked, or both (fill first, then the stroke over it, on every target). A shape with neither colour draws nothing and still occupies its child slot, which keeps the slots of the shapes after it stable while it comes and goes.

<small>[core/canvas.go:193](https://github.com/rohanthewiz/grmob/blob/master/core/canvas.go#L193)</small>

### type TabItem

```go
type TabItem struct {
	Label string
	Icon  string
}
```

<small>[core/tabview.go:10](https://github.com/rohanthewiz/grmob/blob/master/core/tabview.go#L10)</small>

#### func Tab

```go
func Tab(label string, icon string) TabItem
```

<small>[core/tabview.go:80](https://github.com/rohanthewiz/grmob/blob/master/core/tabview.go#L80)</small>

### type TabViewNode

```go
type TabViewNode struct {
	SelectedIndex int
	OnTabChange   func(int)
	Tabs          []TabItem
	Content       []View
}
```

<small>[core/tabview.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/tabview.go#L3)</small>

### type TabViewProp

```go
type TabViewProp interface {
	Apply(*TabViewNode)
}
```

<small>[core/tabview.go:15](https://github.com/rohanthewiz/grmob/blob/master/core/tabview.go#L15)</small>

#### func Content

```go
func Content(views ...View) TabViewProp
```

<small>[core/tabview.go:74](https://github.com/rohanthewiz/grmob/blob/master/core/tabview.go#L74)</small>

#### func OnTabChange

```go
func OnTabChange(fn func(int)) TabViewProp
```

<small>[core/tabview.go:62](https://github.com/rohanthewiz/grmob/blob/master/core/tabview.go#L62)</small>

#### func SelectedIndex

```go
func SelectedIndex(i int) TabViewProp
```

<small>[core/tabview.go:56](https://github.com/rohanthewiz/grmob/blob/master/core/tabview.go#L56)</small>

#### func Tabs

```go
func Tabs(tabs ...TabItem) TabViewProp
```

<small>[core/tabview.go:68](https://github.com/rohanthewiz/grmob/blob/master/core/tabview.go#L68)</small>

