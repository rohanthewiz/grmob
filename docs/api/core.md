# Package core

```go
import "github.com/rohanthewiz/grmob/core"
```

Package core is the vocabulary a grmob app is written in: views, the tree they render to, the state they read, and the styling and accessibility properties they carry. Everything else in the framework is layered on this package's public surface — the widget library, the validation layer, the two web exporters and both native bridges import core and add nothing to it.

## The shape of an app

A view is anything with a Render method ([View](#type-view)), and what it returns is a [Node](#type-node) — a plain data description of one element and its children. An app is a function from a [Context](#type-context) to a view, and a frame is one call of it:

	func App(ctx *core.Context) core.View {
	    return core.Column(
	        core.Text("Hello"),
	        core.Button("Tap", func() { /* ... */ }),
	    )
	}

Nothing here draws anything. Render produces a tree; the render loop in package render diffs that tree against the previous one and hands the difference to whichever host is attached. The same app function therefore runs unchanged on Android, on iOS, in the browser and in a test.

## State and the cursor

[NewState](#func-newstate) allocates a slot on the context and returns typed accessors for it. Slots are identified by the order the calls happen in, not by name:

	pass 1:  NewState(ctx, 0)   NewState(ctx, "")   ->  slot 0, slot 1
	pass 2:  NewState(ctx, 0)   NewState(ctx, "")   ->  slot 0, slot 1
	         ^ the same two slots, because the same two calls in the same order

That is what makes the rules of hooks rules rather than advice: a state call inside a conditional shifts every later call onto a neighbour's slot. Debug mode detects the shift and reports it as a concern (see [SetDebugMode](#func-setdebugmode) and [DumpConcerns](#func-dumpconcerns)) rather than leaving it to be found as a display bug.

[State.Set](#func-state-set) is safe to call from any goroutine. It writes the slot and nudges the render loop, which coalesces a burst of writes into a single pass.

## Nodes are frozen

A [Node](#type-node) is immutable once the pass that produced it returns. The reconciler treats pointer equality as proof that a subtree is unchanged, which is what makes [Cached](#func-cached) worth having and what makes a post-render mutation invisible rather than merely unusual — the diff would skip the subtree it changed.

## Where to read next

The narrative documentation covers the parts a reference cannot: the architecture's pass boundaries, the rules of hooks in full, the styling resolution order, and the reconciler's cost model. This page is the exact surface; those pages are why it has the shape it does.

## Index

- [Constants](#constants) — `AlignItemsCenter`, `AlignItemsEnd`, `AlignItemsStart`, `AlignItemsStretch`, `ConcernCachedCallbacks`, `ConcernCachedHooks`, `ConcernCursorDrift`, `ConcernDanglingReference`, `ConcernDuplicateAccessibilityID`, `ConcernDuplicateKey`, `ConcernHandlerPanic`, `ConcernInertDisclosure`, and 39 more
- [Variables](#variables) — `AmberTheme`, `DefaultTheme`, `MaterialTheme`, `TextInputStyle`
- [`func AngleDelta`](#func-angledelta)
- [`func AudioLoad`](#func-audioload)
- [`func AudioPause`](#func-audiopause)
- [`func AudioPlay`](#func-audioplay)
- [`func AudioSeek`](#func-audioseek)
- [`func AudioSetRate`](#func-audiosetrate)
- [`func AudioSkip`](#func-audioskip)
- [`func AudioStop`](#func-audiostop)
- [`func AudioToggle`](#func-audiotoggle)
- [`func AuditTree`](#func-audittree)
- [`func BundledThemes`](#func-bundledthemes)
- [`func CanPop`](#func-canpop)
- [`func Cardinal`](#func-cardinal)
- [`func ClearConcerns`](#func-clearconcerns)
- [`func CompositeWalkStopsAt`](#func-compositewalkstopsat)
- [`func DangerColor`](#func-dangercolor)
- [`func DismissKeyboard`](#func-dismisskeyboard)
- [`func DistanceMeters`](#func-distancemeters)
- [`func DumpConcerns`](#func-dumpconcerns)
- [`func EditBlock`](#func-editblock)
- [`func EditLink`](#func-editlink)
- [`func Focus`](#func-focus)
- [`func FocusNext`](#func-focusnext)
- [`func FocusPrevious`](#func-focusprevious)
- [`func FormatLatLng`](#func-formatlatlng)
- [`func FormatRegion`](#func-formatregion)
- [`func GroupingContainers`](#func-groupingcontainers)
- [`func HasSystemEventHandler`](#func-hassystemeventhandler)
- [`func HeadingActive`](#func-headingactive)
- [`func IsDebugMode`](#func-isdebugmode)
- [`func LinearGradient`](#func-lineargradient)
- [`func LocationAcquiring`](#func-locationacquiring)
- [`func LocationActive`](#func-locationactive)
- [`func NormalizeDegrees`](#func-normalizedegrees)
- [`func OnAudioStatus`](#func-onaudiostatus)
- [`func OnDeepLink`](#func-ondeeplink)
- [`func OnHeading`](#func-onheading)
- [`func OnHostEvent`](#func-onhostevent)
- [`func OnLifecycle`](#func-onlifecycle)
- [`func OnLocation`](#func-onlocation)
- [`func OpenURL`](#func-openurl)
- [`func ParseLatLng`](#func-parselatlng)
- [`func PlacingContainers`](#func-placingcontainers)
- [`func Pop`](#func-pop)
- [`func PopToRoot`](#func-poptoroot)
- [`func PrimaryColor`](#func-primarycolor)
- [`func Push`](#func-push)
- [`func ReceiveAudioStatus`](#func-receiveaudiostatus)
- [`func ReceiveHeading`](#func-receiveheading)
- [`func ReceiveHostEvent`](#func-receivehostevent)
- [`func ReceiveLifecycle`](#func-receivelifecycle)
- [`func ReceiveLocation`](#func-receivelocation)
- [`func Replace`](#func-replace)
- [`func ReportConcern`](#func-reportconcern)
- [`func Reset`](#func-reset)
- [`func RunEditorCommand`](#func-runeditorcommand)
- [`func SendSystemEvent`](#func-sendsystemevent)
- [`func SetDebugMode`](#func-setdebugmode)
- [`func SetSystemEventHandler`](#func-setsystemeventhandler)
- [`func ShowToast`](#func-showtoast)
- [`func StackDepth`](#func-stackdepth)
- [`func StartHeading`](#func-startheading)
- [`func StartLocation`](#func-startlocation)
- [`func StopHeading`](#func-stopheading)
- [`func StopLocation`](#func-stoplocation)
- [`func UseFocusOrder`](#func-usefocusorder)
- [`func WithConfigOpt`](#func-withconfigopt)
- [`func WithThemeOpt`](#func-withthemeopt)
- [`func WrapLongitude`](#func-wraplongitude)
- [`type AlignItems`](#type-alignitems)
    - [`func AlignItemsValues`](#func-alignitemsvalues)
    - [`func (AlignItems) Apply`](#func-alignitems-apply)
- [`type Alignment`](#type-alignment)
    - [`func Alignments`](#func-alignments)
    - [`func TextAlignments`](#func-textalignments)
- [`type AppConfig`](#type-appconfig)
- [`type AudioOpt`](#type-audioopt)
    - [`func AudioAutoplay`](#func-audioautoplay)
    - [`func AudioStartAt`](#func-audiostartat)
    - [`func AudioWithRate`](#func-audiowithrate)
- [`type AudioState`](#type-audiostate)
- [`type AudioStatus`](#type-audiostatus)
    - [`func CurrentAudioStatus`](#func-currentaudiostatus)
    - [`func (AudioStatus) Loaded`](#func-audiostatus-loaded)
    - [`func (AudioStatus) Progress`](#func-audiostatus-progress)
- [`type AudioTrack`](#type-audiotrack)
- [`type BehaviorProp`](#type-behaviorprop)
    - [`func CommentPrefix`](#func-commentprefix)
    - [`func EditorTarget`](#func-editortarget)
    - [`func FocusTarget`](#func-focustarget)
    - [`func KeyboardAware`](#func-keyboardaware)
    - [`func LineNumbers`](#func-linenumbers)
    - [`func On`](#func-on)
    - [`func OnBack`](#func-onback)
    - [`func OnBlur`](#func-onblur)
    - [`func OnClick`](#func-onclick)
    - [`func OnEndReached`](#func-onendreached)
    - [`func OnFocus`](#func-onfocus)
    - [`func OnLongPress`](#func-onlongpress)
    - [`func OnMapTap`](#func-onmaptap)
    - [`func OnMarkerTap`](#func-onmarkertap)
    - [`func OnRegionChange`](#func-onregionchange)
    - [`func OnRichSelectionChange`](#func-onrichselectionchange)
    - [`func OnSelectionChange`](#func-onselectionchange)
    - [`func OnSliderChangeEnd`](#func-onsliderchangeend)
    - [`func OnTouch`](#func-ontouch)
    - [`func Placeholder`](#func-placeholder)
    - [`func ReadOnly`](#func-readonly)
    - [`func ShowUserLocation`](#func-showuserlocation)
    - [`func SliderStep`](#func-sliderstep)
    - [`func TabSize`](#func-tabsize)
- [`type CameraNode`](#type-cameranode)
- [`type CameraProp`](#type-cameraprop)
    - [`func OnCapture`](#func-oncapture)
    - [`func OnError`](#func-onerror)
    - [`func SetFacing`](#func-setfacing)
    - [`func WithFlash`](#func-withflash)
    - [`func WithOverlay`](#func-withoverlay)
    - [`func WithStyle`](#func-withstyle)
- [`type ColorPalette`](#type-colorpalette)
    - [`func (ColorPalette) BorderColor`](#func-colorpalette-bordercolor)
    - [`func (ColorPalette) ControlBorderColor`](#func-colorpalette-controlbordercolor)
    - [`func (ColorPalette) ErrorOnLightColor`](#func-colorpalette-erroronlightcolor)
    - [`func (ColorPalette) OnLight`](#func-colorpalette-onlight)
    - [`func (ColorPalette) PrimaryOnLightColor`](#func-colorpalette-primaryonlightcolor)
    - [`func (ColorPalette) SuccessColor`](#func-colorpalette-successcolor)
    - [`func (ColorPalette) SuccessOnLightColor`](#func-colorpalette-successonlightcolor)
    - [`func (ColorPalette) WarningColor`](#func-colorpalette-warningcolor)
    - [`func (ColorPalette) WarningOnLightColor`](#func-colorpalette-warningonlightcolor)
- [`type ComponentDefaults`](#type-componentdefaults)
- [`type ComponentFunc`](#type-componentfunc)
    - [`func (ComponentFunc) Render`](#func-componentfunc-render)
- [`type CompositeWalk`](#type-compositewalk)
    - [`func CompositeWalkAt`](#func-compositewalkat)
    - [`func (CompositeWalk) String`](#func-compositewalk-string)
- [`type Concern`](#type-concern)
    - [`func Concerns`](#func-concerns)
- [`type ContentMode`](#type-contentmode)
    - [`func ContentModes`](#func-contentmodes)
- [`type Context`](#type-context)
    - [`func NewContext`](#func-newcontext)
    - [`func UseChildContext`](#func-usechildcontext)
    - [`func (*Context) BeginRenderPass`](#func-context-beginrenderpass)
    - [`func (*Context) ClearDirty`](#func-context-cleardirty)
    - [`func (*Context) Close`](#func-context-close)
    - [`func (*Context) Config`](#func-context-config)
    - [`func (*Context) EndRenderPass`](#func-context-endrenderpass)
    - [`func (*Context) IsDirty`](#func-context-isdirty)
    - [`func (*Context) MarkDirty`](#func-context-markdirty)
    - [`func (*Context) NewChildContext`](#func-context-newchildcontext)
    - [`func (*Context) OnClose`](#func-context-onclose)
    - [`func (*Context) OnStateChange`](#func-context-onstatechange)
    - [`func (*Context) PurgeUnusedCallbacks`](#func-context-purgeunusedcallbacks)
    - [`func (*Context) ReceiveEventPayload`](#func-context-receiveeventpayload)
    - [`func (*Context) RequestRender`](#func-context-requestrender)
    - [`func (*Context) Reset`](#func-context-reset)
    - [`func (*Context) Scope`](#func-context-scope)
    - [`func (*Context) Theme`](#func-context-theme)
    - [`func (*Context) TriggerBoolCallback`](#func-context-triggerboolcallback)
    - [`func (*Context) TriggerCallback`](#func-context-triggercallback)
    - [`func (*Context) TriggerIntCallback`](#func-context-triggerintcallback)
    - [`func (*Context) TriggerTextCallback`](#func-context-triggertextcallback)
    - [`func (*Context) With`](#func-context-with)
    - [`func (*Context) WithConfig`](#func-context-withconfig)
    - [`func (*Context) WithTheme`](#func-context-withtheme)
- [`type DisplayMode`](#type-displaymode)
- [`type Easing`](#type-easing)
- [`type EdgeInsets`](#type-edgeinsets)
- [`type EditorRef`](#type-editorref)
    - [`func UseEditorRef`](#func-useeditorref)
- [`type ExpandedState`](#type-expandedstate)
    - [`func ExpandedStates`](#func-expandedstates)
    - [`func ExpandedWhen`](#func-expandedwhen)
- [`type FlexDirection`](#type-flexdirection)
    - [`func (FlexDirection) Apply`](#func-flexdirection-apply)
- [`type FocusRef`](#type-focusref)
    - [`func UseFocusRef`](#func-usefocusref)
- [`type GridRow`](#type-gridrow)
- [`type GridRun`](#type-gridrun)
- [`type Heading`](#type-heading)
    - [`func CurrentHeading`](#func-currentheading)
    - [`func (Heading) Cardinal`](#func-heading-cardinal)
- [`type JustifyContent`](#type-justifycontent)
    - [`func JustifyContents`](#func-justifycontents)
    - [`func (JustifyContent) Apply`](#func-justifycontent-apply)
- [`type LifecycleState`](#type-lifecyclestate)
    - [`func CurrentLifecycle`](#func-currentlifecycle)
- [`type Location`](#type-location)
    - [`func CurrentLocation`](#func-currentlocation)
- [`type MatchCase`](#type-matchcase)
    - [`func Case`](#func-case)
    - [`func Default`](#func-default)
- [`type ModalNode`](#type-modalnode)
- [`type ModalProp`](#type-modalprop)
    - [`func Backdrop`](#func-backdrop)
    - [`func ModalContent`](#func-modalcontent)
    - [`func OnDismiss`](#func-ondismiss)
    - [`func Visible`](#func-visible)
- [`type Node`](#type-node)
    - [`func Render`](#func-render)
- [`type Position`](#type-position)
- [`type Progress`](#type-progress)
- [`type ProgressReading`](#type-progressreading)
- [`type PropsAndChildren`](#type-propsandchildren)
    - [`func MaybeProp`](#func-maybeprop)
- [`type Region`](#type-region)
    - [`func ParseRegion`](#func-parseregion)
- [`type RenderError`](#type-rendererror)
    - [`func Guard`](#func-guard)
    - [`func (*RenderError) Error`](#func-rendererror-error)
    - [`func (*RenderError) Unwrap`](#func-rendererror-unwrap)
- [`type RenderManager`](#type-rendermanager)
    - [`func NewRenderManager`](#func-newrendermanager)
    - [`func (*RenderManager) TriggerRender`](#func-rendermanager-triggerrender)
- [`type ResponsiveStyle`](#type-responsivestyle)
- [`type RichSelection`](#type-richselection)
    - [`func (RichSelection) HasSelection`](#func-richselection-hasselection)
- [`type Role`](#type-role)
    - [`func CompositeMemberRole`](#func-compositememberrole)
    - [`func KeyboardComposites`](#func-keyboardcomposites)
    - [`func Roles`](#func-roles)
    - [`func TappableContainerRoles`](#func-tappablecontainerroles)
- [`type SelectMenuItem`](#type-selectmenuitem)
- [`type SelectMenuSection`](#type-selectmenusection)
    - [`func SelectMenuSections`](#func-selectmenusections)
    - [`func (SelectMenuSection) First`](#func-selectmenusection-first)
- [`type SelectOption`](#type-selectoption)
    - [`func Option`](#func-option)
- [`type SelectedState`](#type-selectedstate)
    - [`func SelectedStates`](#func-selectedstates)
    - [`func SelectedWhen`](#func-selectedwhen)
- [`type SpacingScale`](#type-spacingscale)
- [`type StackAlignment`](#type-stackalignment)
    - [`func StackAlignments`](#func-stackalignments)
- [`type State`](#type-state)
    - [`func NewState`](#func-newstate)
    - [`func (*State) Get`](#func-state-get)
    - [`func (*State) Set`](#func-state-set)
- [`type Style`](#type-style)
    - [`func (Style) ShrinkFactor`](#func-style-shrinkfactor)
    - [`func (Style) With`](#func-style-with)
- [`type StyleProp`](#type-styleprop)
    - [`func AccessibilityControls`](#func-accessibilitycontrols)
    - [`func AccessibilityExpanded`](#func-accessibilityexpanded)
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
    - [`func BorderColor`](#func-bordercolor)
    - [`func BorderRadius`](#func-borderradius)
    - [`func BorderWidth`](#func-borderwidth)
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
    - [`func Horizontal`](#func-horizontal)
    - [`func Justify`](#func-justify)
    - [`func Left`](#func-left)
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
    - [`func PaddingHorizontal`](#func-paddinghorizontal)
    - [`func PaddingLeft`](#func-paddingleft)
    - [`func PaddingRight`](#func-paddingright)
    - [`func PaddingTop`](#func-paddingtop)
    - [`func PaddingVertical`](#func-paddingvertical)
    - [`func Responsive`](#func-responsive)
    - [`func Right`](#func-right)
    - [`func Rotate`](#func-rotate)
    - [`func RoundedShadowBox`](#func-roundedshadowbox)
    - [`func RowGap`](#func-rowgap)
    - [`func Shadow`](#func-shadow)
    - [`func StackAlign`](#func-stackalign)
    - [`func StickyHeader`](#func-stickyheader)
    - [`func TextColor`](#func-textcolor)
    - [`func Transition`](#func-transition)
    - [`func UseStyle`](#func-usestyle)
    - [`func WhiteSpace`](#func-whitespace)
    - [`func Width`](#func-width)
    - [`func ZIndex`](#func-zindex)
- [`type TabItem`](#type-tabitem)
    - [`func Tab`](#func-tab)
- [`type TabViewNode`](#type-tabviewnode)
- [`type TabViewProp`](#type-tabviewprop)
    - [`func Content`](#func-content)
    - [`func OnTabChange`](#func-ontabchange)
    - [`func SelectedIndex`](#func-selectedindex)
    - [`func Tabs`](#func-tabs)
- [`type Theme`](#type-theme)
- [`type ToastConfig`](#type-toastconfig)
- [`type ToastOpt`](#type-toastopt)
    - [`func Duration`](#func-duration)
    - [`func UseToastStyle`](#func-usetoaststyle)
- [`type Typography`](#type-typography)
- [`type ValueRange`](#type-valuerange)
    - [`func ValueOf`](#func-valueof)
    - [`func (ValueRange) Progress`](#func-valuerange-progress)
    - [`func (ValueRange) Stated`](#func-valuerange-stated)
    - [`func (ValueRange) Unparsed`](#func-valuerange-unparsed)
    - [`func (ValueRange) WithText`](#func-valuerange-withtext)
- [`type View`](#type-view)
    - [`func Box`](#func-box)
    - [`func Button`](#func-button)
    - [`func ButtonWithEvent`](#func-buttonwithevent)
    - [`func Cached`](#func-cached)
    - [`func CameraView`](#func-cameraview)
    - [`func Card`](#func-card)
    - [`func Checkbox`](#func-checkbox)
    - [`func CodeEditor`](#func-codeeditor)
    - [`func Column`](#func-column)
    - [`func DefaultErrorFallback`](#func-defaulterrorfallback)
    - [`func Divider`](#func-divider)
    - [`func ErrorBoundary`](#func-errorboundary)
    - [`func For`](#func-for)
    - [`func Fragment`](#func-fragment)
    - [`func If`](#func-if)
    - [`func IfElse`](#func-ifelse)
    - [`func Image`](#func-image)
    - [`func ImageWithMode`](#func-imagewithmode)
    - [`func Input`](#func-input)
    - [`func InputPassword`](#func-inputpassword)
    - [`func InputWithSubmit`](#func-inputwithsubmit)
    - [`func Keyed`](#func-keyed)
    - [`func List`](#func-list)
    - [`func MapView`](#func-mapview)
    - [`func Marker`](#func-marker)
    - [`func Match`](#func-match)
    - [`func MatchBool`](#func-matchbool)
    - [`func Modal`](#func-modal)
    - [`func Navigator`](#func-navigator)
    - [`func NumericInput`](#func-numericinput)
    - [`func RichTextEditor`](#func-richtexteditor)
    - [`func Row`](#func-row)
    - [`func SafeArea`](#func-safearea)
    - [`func SafeRender`](#func-saferender)
    - [`func Scroll`](#func-scroll)
    - [`func Select`](#func-select)
    - [`func Slider`](#func-slider)
    - [`func Spacer`](#func-spacer)
    - [`func Switch`](#func-switch)
    - [`func TabView`](#func-tabview)
    - [`func Text`](#func-text)
    - [`func TextArea`](#func-textarea)
    - [`func TextGrid`](#func-textgrid)
    - [`func WithTheme`](#func-withtheme)
    - [`func ZStack`](#func-zstack)
- [`type Weight`](#type-weight)
- [`type WhenClause`](#type-whenclause)
    - [`func Otherwise`](#func-otherwise)
    - [`func When`](#func-when)

## Constants

Concern kinds for the accessibility audit. Declared here rather than beside the others in debug.go so the seven arrive with the walk that produces them. The walk's eighth finding, ConcernInertPlacement, is declared in placement\_audit.go for the same reason: beside the argument for it.

```go
const (
	// ConcernDuplicateAccessibilityID: two elements in one tree carry the
	// same core.Style.AccessibilityID. Ids are document-global and nothing
	// rewrites the string, so both are written verbatim — which is invalid
	// HTML, and which makes every aria-controls pointing at that id resolve
	// to whichever element the browser parsed first.
	ConcernDuplicateAccessibilityID = "duplicate-accessibility-id"

	// ConcernDanglingReference: a core.Style.AccessibilityControls names an
	// id no element in the tree claims. The attribute is written anyway (an
	// exporter has no index to check against) and a reader following it finds
	// nothing, so the control announces as governing a region that is not
	// there.
	ConcernDanglingReference = "dangling-aria-reference"

	// ConcernInvalidAccessibilityID: an AccessibilityID that is not a usable
	// HTML id — one containing whitespace, or one starting with the "grmob-"
	// prefix core.TabView's own minted ids live in.
	ConcernInvalidAccessibilityID = "invalid-accessibility-id"

	// ConcernInertDisclosure: a node states core.Style.AccessibilityExpanded
	// and carries no handler to act on it. The web writes aria-expanded
	// regardless, and Compose does not: its expand()/collapse() are semantics
	// *actions*, and Renderer.kt wires them to the node's own click callback,
	// so a node with a state and nothing to perform gets neither. That is the
	// right behaviour — an action nothing can perform is worse than none —
	// and it is a rule stated only in a comment, where the equivalent web rule
	// (a state on an unroled node is dropped) has a test on both targets.
	ConcernInertDisclosure = "inert-disclosure"

	// ConcernInertFollowsFocus: a node states
	// core.Style.AccessibilitySelectionFollowsFocus and carries no role that
	// has a keyboard for a selection to follow. The WASM runtime writes the
	// data attribute on any node that asks — deliberately, since consulting
	// the composite tables where an attribute is written would put those
	// tables in two places — so the flag lands in the DOM and is read by
	// nothing, and no other target writes anything for it at all.
	ConcernInertFollowsFocus = "inert-follows-focus"

	// ConcernUnusableValueRange: a node states a core.Style.AccessibilityValue
	// whose numbers no platform can use as written — a position or a bound
	// that is not a number, or a range whose Max is at or below its Min. Every
	// target resolves it and no two of them resolve it the same way, so the
	// bar announces a different wrong number on each. See checkValueRange.
	ConcernUnusableValueRange = "unusable-value-range"

	// ConcernNestedComposite: a container with an ARIA keyboard pattern sits
	// inside another one. Both keep their own roving tabindex, so the pair is
	// two tab stops where ARIA describes one — the outer widget's arrows step
	// over the inner widget whole. It is deliberate and it is a divergence;
	// see checkNestedComposite.
	ConcernNestedComposite = "nested-composite"
)
```

<small>[core/a11y_audit.go:73](https://github.com/rohanthewiz/grmob/blob/master/core/a11y_audit.go#L73)</small>

Concern kinds. Each names one class of silent bug the debug checks detect.

```go
const (
	// ConcernCursorDrift: a context's hook cursor ended a pass out of step
	// with its slot count or with the previous pass — some NewState /
	// UseChildContext call is conditional or loop-varying, so later slots
	// are (or will be) read by the wrong component.
	ConcernCursorDrift = "cursor-drift"

	// ConcernDuplicateKey: two siblings in one container carry the same
	// non-empty Key, defeating keyed reconciliation for that sibling list.
	ConcernDuplicateKey = "duplicate-key"

	// ConcernCachedHooks: a Cached view consumed hook slots during its
	// render. In production the view renders once and never again, so those
	// slots vanish on later passes and shift every component after it.
	ConcernCachedHooks = "cached-hooks"

	// ConcernCachedCallbacks: a Cached view registered event callbacks. In
	// production its handlers are purged after the first pass it skips, and
	// the un-consumed counter slots shift the callback IDs of everything
	// registered after it.
	ConcernCachedCallbacks = "cached-callbacks"

	// ConcernUnknownItem: a container (Row, Column, Card, Box, List) was
	// handed an argument that is neither a StyleProp, a BehaviorProp nor a
	// View. PropsAndChildren is an alias for any, so the compiler accepts
	// anything and containerNode drops what it cannot classify — the symptom
	// is a style or handler that simply never took effect. An untyped nil is
	// exempt: that is MaybeProp's false path, not a mistake.
	ConcernUnknownItem = "unknown-container-item"

	// ConcernRenderPanic: an ErrorBoundary caught a panic escaping a
	// component's Render and swapped in its fallback. The app kept running —
	// that is the boundary doing its job — which is exactly why this needs
	// reporting: a boundary placed high in the tree can hide a component that
	// has been dead for weeks behind a plausible-looking "unavailable" panel.
	// The detail carries the panic value; the full stack goes to the
	// fallback, not here.
	ConcernRenderPanic = "render-panic"

	// ConcernHandlerPanic: an event handler panicked and the render driver
	// recovered it. Distinct from ConcernRenderPanic because the failure is
	// in a different phase with a different blast radius: a render panic
	// costs a subtree's frame, while a handler panic abandons the handler
	// partway, so the app's state may be half-updated in a way no fallback
	// can describe.
	ConcernHandlerPanic = "handler-panic"
)
```

<small>[core/debug.go:40](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L40)</small>

The commands a core.CodeEditor understands. Spelled as constants so a toolbar and a renderer cannot disagree about a string literal, and so the census of what v1 supports is one list rather than four.

Each acts on the host's current selection, or on the line the caret is in when the selection is empty — which is what makes "indent" useful without a selection, and is the behavior every code editor has.

```go
const (
	// EditIndent inserts one indent at the start of every line the selection
	// touches. One indent is tabSize spaces, or a tab when tabSize is 0.
	EditIndent = "indent"
	// EditOutdent removes one indent's worth of leading white space from every
	// line the selection touches, and leaves a line that has none alone.
	EditOutdent = "outdent"
	// EditCommentLine toggles the commentPrefix on every line the selection
	// touches: it comments them all when any is uncommented, and uncomments
	// them when every one is already commented. The toggle is decided for the
	// whole run rather than per line, so a partially-commented block becomes
	// fully commented rather than inverting line by line.
	EditCommentLine = "commentLine"
	// EditSelectAll selects the whole buffer.
	EditSelectAll = "selectAll"
)
```

<small>[core/editor.go:191](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L191)</small>

The commands a core.RichTextEditor understands. Each toggles on the current selection, or sets the typing attributes when the selection is empty — which is what makes "press bold, then type" work.

```go
const (
	EditBold      = "bold"
	EditItalic    = "italic"
	EditUnderline = "underline"
	EditStrike    = "strike"
	// EditCode is the inline mark — a monospace span inside a sentence. The
	// block-level one is EditBlock(richtext.BlockCode).
	EditCode = "code"
	// EditUnlink removes the link from the selection, leaving its text.
	EditUnlink = "unlink"
	// EditUndo and EditRedo are the editing history. Each host uses its own
	// (UIKit's UndoManager, EditText's), except the web, where the runtime keeps
	// a stack of Docs: a browser's native history does not survive the
	// programmatic attribute edits the other commands make.
	EditUndo = "undo"
	EditRedo = "redo"
)
```

<small>[core/richtext.go:88](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L88)</small>

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

<small>[core/style.go:1132](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1132)</small>

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

Fallbacks for the three roles a pre-existing theme can be missing. They are DefaultTheme's own values, so a theme that omits a role looks like the default theme in that one place rather than disappearing.

Exported because a widget outside this package resolving a role by hand (rather than through the methods below) should land on the same value.

```go
const (
	FallbackBorder  = "#E5E5EA" // iOS systemGray5, the hairline both examples had independently picked
	FallbackSuccess = "#34C759" // iOS system green
	FallbackWarning = "#FF9500" // iOS system orange

	// FallbackControlBorder is DefaultTheme's boundary tone, which is also
	// what its Input and TextArea frames are painted in. A theme predating
	// this role degrades to a *visible* edge rather than to Border's hairline:
	// falling back to the divider is the levelling-down the role was split to
	// prevent, and it would be indistinguishable from the bug.
	FallbackControlBorder = "#89898E" // systemGray, darkened — 3.48:1 on white
)
```

<small>[core/theme.go:240](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L240)</small>

ConcernInertPlacement: a node states core.Style.StackAlign and the container that would place it is not an overlay.

Deliberately inert rather than accidentally so — a layer prop that re-placed a Row's children would be worse than one that did nothing, and align-self already means something else to a flex child — which is what makes this a concern and not a bug to fix in a renderer. The author has written a prop that cannot work where they put it, and the four targets agree silently.

```go
const ConcernInertPlacement = "inert-stack-placement"
```

<small>[core/placement_audit.go:52](https://github.com/rohanthewiz/grmob/blob/master/core/placement_audit.go#L52)</small>

DefaultMapZoom is the scale a Region with no Zoom is drawn at: a neighbourhood, which is close enough to read street names and wide enough to hold more than one marker.

14 rather than comps.DefaultMapZoom's 15, and the difference is the difference between the two widgets. A static map answers "where is this one place"; a live map is usually showing a set, and one level out is about four times the area.

```go
const DefaultMapZoom = 14.0
```

<small>[core/mapview.go:182](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L182)</small>

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

<small>[core/style.go:1231](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1231)</small>

## Variables

AmberTheme is the third bundled palette, and the one whose brand colour is not a colour you can put white on.

#### Why a third theme exists at all

Two rules in this framework had no bundled evidence. inkOn reads the theme's declared fill/ink pair \*before\* measuring contrast, and ColorPalette.OnLight moves a role to its ink-weight tone — and under both palettes above, an implementation that deleted either would paint identical pixels. Their only witness was a test fixture (components' midTonePrimaryTheme), which is a thing a palette edit can quietly leave holding the whole thread.

This palette witnesses both, and it does so the way a real brand does rather than by being contrived:

	the declaration outranks the measurement
	    Components.Button below states brown 900 over the amber fill. Measured
	    against this theme's own two inks, the winner is TextPrimary (8.39:1
	    against brown 900's 6.77:1) — so an inkOn that only measured would paint
	    a different, and slightly *higher*-contrast, label. Deleting declaredInk
	    moves pixels here.

	OnLight moves the Primary role
	    Amber 700 is 2.04:1 on white. It is an excellent fill and cannot be ink,
	    which is the whole argument for the on-light tones and the case neither
	    palette above still makes for Primary.

#### The escape that made this shippable, which is worth recording

The obvious third theme — a mid-tone brand with white declared over it — is not shippable, and the reason is arithmetic rather than taste. contrastInk picks the \*higher\*-contrast of the theme's two ink roles, and against any fill the white ratio and the black ratio multiply to at most 21. So a declaration that loses the measurement can be at most sqrt(21) = 4.58:1, while components' AA check demands 4.5:1 of every variant's ink. The band a pole-flipping bundled theme would have to sit in is \[4.50, 4.58] — 1.8% wide, at the very bottom of the legibility scale.

The squeeze binds only when the declared ink is one of the two poles being measured. A brand ink that is \*neither\* — brown 900 over amber, where the page's ink is MD3's near-black — is a different colour from the measurement's answer with three points of headroom on both. That is also the ordinary real-world decision ("our label is warm, not the page's black"), so the theme carries the rule by being a normal theme rather than by being tuned to a gap. components' TestThePoleFlipBandIsTooNarrowToShip states the arithmetic above as a test, and the pole flip itself stays with the fixture — now provably rather than accidentally.

#### Provenance

Material Design published values throughout, as MaterialTheme's are, with one exception stated at the field: the amber family carries no swatch dark enough to be read as ink on white (amber 900 is 2.79:1), so PrimaryOnLight is amber 700 scaled to 56% brightness — the same hue at 37.8 degrees, and the same move DefaultTheme's SuccessOnLight makes one hue over for the same reason.

```go
var AmberTheme = &Theme{
	Colors: ColorPalette{

		Primary:       "#FFA000",
		Secondary:     "#00897B",
		Background:    "#FFFFFF",
		Surface:       "#FFF8E1",
		TextPrimary:   "#1C1B1F",
		TextSecondary: "#616161",
		Error:         "#B00020",
		Border:        "#E0E0E0",
		Success:       "#2E7D32",
		Warning:       "#EF6C00",

		ControlBorder: "#8D6E63",

		PrimaryOnLight: "#8F5A00",
		SuccessOnLight: "#2E7D32",
		WarningOnLight: "#BF360C",
		ErrorOnLight:   "#B00020",
	},
	Typography: Typography{
		Title:    Style{FontSize: 24, FontWeight: Bold, TextColor: "#1C1B1F", Display: DisplayBlock},
		Subtitle: Style{FontSize: 19, FontWeight: Normal, TextColor: "#3A3A3E", Display: DisplayBlock},
		Body:     Style{FontSize: 16, FontWeight: Normal, TextColor: "#1C1B1F", Display: DisplayBlock},
		Caption:  Style{FontSize: 13, FontWeight: Normal, TextColor: "#616161", Display: DisplayBlock},
	},
	Spacing: SpacingScale{
		XS: 4,
		SM: 8,
		MD: 16,
		LG: 24,
		XL: 32,
	},
	Components: ComponentDefaults{

		Button: Style{
			FontSize:     16,
			FontWeight:   Bold,
			TextColor:    "#3E2723",
			Background:   "#FFA000",
			Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 20, Right: 20},
			BorderRadius: 20,
			Shadow:       1,
			Align:        AlignCenter,
			Display:      DisplayInline,
		},
		Card: Style{
			Background:   "#FFFFFF",
			Padding:      EdgeInsets{Top: 16, Bottom: 16, Left: 16, Right: 16},
			Margin:       EdgeInsets{Top: 8, Bottom: 8, Left: 8, Right: 8},
			BorderRadius: 12,
			Shadow:       1,
			Display:      DisplayBlock,
		},
		Input: Style{
			FontSize:   16,
			FontWeight: Normal,
			TextColor:  "#1C1B1F",

			Background:   "#FFF8E1",
			Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 12, Right: 12},
			BorderColor:  "#8D6E63",
			BorderWidth:  1,
			BorderRadius: 8,
			Shadow:       0,
			Display:      DisplayBlock,
		},
		CheckBox: Style{
			Background:   "#FFFFFF",
			TextColor:    "#1C1B1F",
			BorderRadius: 4,
			Margin:       EdgeInsets{Right: 8},
			Display:      DisplayInline,
		},
		TextArea: Style{
			FontSize:     16,
			FontWeight:   Normal,
			TextColor:    "#1C1B1F",
			Background:   "#FFF8E1",
			Padding:      EdgeInsets{Top: 12, Bottom: 12, Left: 12, Right: 12},
			BorderColor:  "#8D6E63",
			BorderWidth:  1,
			BorderRadius: 8,
			Display:      DisplayBlock,
		},
		Column: Style{
			Padding: EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16},
		},
		Row: Style{
			Padding: EdgeInsets{Top: 8, Bottom: 8, Left: 16, Right: 16},
		},
		Camera: Style{
			Background: "#000000",
			Display:    DisplayBlock,
		},
	},
}
```

<small>[core/theme.go:847](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L847)</small>

```go
var DefaultTheme = &Theme{
	Colors: ColorPalette{

		Primary:       "#0040DD",
		Secondary:     "#34C759",
		Background:    "#FFFFFF",
		Surface:       "#F2F2F7",
		TextPrimary:   "#000000",
		TextSecondary: "#3C3C4399",
		Error:         "#FF3B30",
		Border:        "#E5E5EA",
		Success:       "#34C759",
		Warning:       "#FF9500",

		ControlBorder: "#89898E",

		PrimaryOnLight: "#0040DD",
		SuccessOnLight: "#1E7A34",
		WarningOnLight: "#C93400",
		ErrorOnLight:   "#D70015",
	},
	Typography: Typography{
		Title: Style{
			FontSize:   28,
			FontWeight: Bold,
			TextColor:  "#000000",
			Display:    DisplayBlock,
		},
		Subtitle: Style{
			FontSize:   22,
			FontWeight: Normal,
			TextColor:  "#3C3C4399",
			Display:    DisplayBlock,
		},
		Body: Style{
			FontSize:   17,
			FontWeight: Normal,
			TextColor:  "#000000",
			Display:    DisplayBlock,
		},
		Caption: Style{
			FontSize:   13,
			FontWeight: Normal,
			TextColor:  "#3C3C4399",
			Display:    DisplayBlock,
		},
	},
	Spacing: SpacingScale{
		XS: 4,
		SM: 8,
		MD: 16,
		LG: 24,
		XL: 32,
	},
	Components: ComponentDefaults{
		Button: Style{
			FontSize:   17,
			FontWeight: Normal,

			TextColor:    "#FFFFFF",
			Background:   "#0040DD",
			Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 16, Right: 16},
			BorderRadius: 8,
			Shadow:       1,
			Align:        AlignCenter,
			Display:      DisplayInline,
		},
		Card: Style{
			Background:   "#FFFFFF",
			Padding:      EdgeInsets{Top: 16, Bottom: 16, Left: 16, Right: 16},
			Margin:       EdgeInsets{Top: 8, Bottom: 8, Left: 8, Right: 8},
			BorderRadius: 12,
			Shadow:       2,
			Display:      DisplayBlock,
		},
		Input: Style{
			FontSize:   17,
			FontWeight: Normal,
			TextColor:  "#000000",
			Background: "#FFFFFF",
			Padding:    EdgeInsets{Top: 8, Bottom: 8, Left: 12, Right: 12},

			BorderColor:  "#89898E",
			BorderWidth:  1,
			BorderRadius: 6,
			Shadow:       0,
			Display:      DisplayBlock,
		},
		CheckBox: Style{
			Background:   "#FFFFFF",
			BorderRadius: 6,
			Shadow:       0,
			Display:      DisplayInline,
		},
		TextArea: Style{
			FontSize:   17,
			FontWeight: Normal,
			TextColor:  "#000000",
			Background: "#FFFFFF",
			Padding:    EdgeInsets{Top: 12, Bottom: 12, Left: 12, Right: 12},

			BorderColor:  "#89898E",
			BorderWidth:  1,
			BorderRadius: 6,
			Display:      DisplayBlock,
		},

		Column: Style{
			Padding: EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16},
		},
		Row: Style{
			Padding: EdgeInsets{Top: 8, Bottom: 8, Left: 16, Right: 16},
		},
		Camera: Style{
			Background: "#000000",
			Display:    DisplayBlock,
		},
	},
}
```

<small>[core/theme.go:496](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L496)</small>

```go
var MaterialTheme = &Theme{
	Colors: ColorPalette{
		Primary:       "#6200EE",
		Secondary:     "#03DAC6",
		Background:    "#FFFFFF",
		Surface:       "#F5F5F5",
		TextPrimary:   "#212121",
		TextSecondary: "#757575",
		Error:         "#B00020",
		Border:        "#E0E0E0",
		Success:       "#2E7D32",
		Warning:       "#EF6C00",

		ControlBorder: "#757575",

		PrimaryOnLight: "#6200EE",
		SuccessOnLight: "#2E7D32",
		WarningOnLight: "#BF360C",
		ErrorOnLight:   "#B00020",
	},
	Typography: Typography{
		Title:    Style{FontSize: 22, FontWeight: Bold, TextColor: "#212121"},
		Subtitle: Style{FontSize: 18, FontWeight: Normal, TextColor: "#424242"},
		Body:     Style{FontSize: 14, FontWeight: Normal, TextColor: "#333333"},
		Caption:  Style{FontSize: 12, FontWeight: Light, TextColor: "#888888"},
	},
	Spacing: SpacingScale{
		XS: 4,
		SM: 8,
		MD: 16,
		LG: 24,
		XL: 32,
	},
	Components: ComponentDefaults{
		Button: Style{
			Background:   "#6200EE",
			TextColor:    "#FFFFFF",
			Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 20, Right: 20},
			BorderRadius: 4,
		},
		Card: Style{
			Background:   "#FFFFFF",
			BorderRadius: 8,
			Shadow:       1,
			Padding:      EdgeInsets{Top: 16, Bottom: 16, Left: 16, Right: 16},
		},
		Input: Style{
			Background: "#FAFAFA",
			Padding:    EdgeInsets{Top: 10, Bottom: 10, Left: 12, Right: 12},

			BorderColor: "#757575",
			BorderWidth: 1,
		},
		Column: Style{
			Padding: EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16},
		},
		Row: Style{
			Padding: EdgeInsets{Top: 8, Bottom: 8, Left: 16, Right: 16},
		},
		Camera: Style{
			Background: "#000000",
			Display:    DisplayBlock,
		},

		CheckBox: Style{
			Display:      DisplayInline,
			Margin:       EdgeInsets{Right: 8},
			TextColor:    "#212121",
			BorderRadius: 2,
		},

		TextArea: Style{
			Background:   "#FAFAFA",
			TextColor:    "#212121",
			Padding:      EdgeInsets{Top: 8, Bottom: 8, Left: 12, Right: 12},
			BorderColor:  "#757575",
			BorderWidth:  1,
			BorderRadius: 4,
		},
	},
}
```

<small>[core/theme.go:691](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L691)</small>

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

<small>[core/style.go:1077](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1077)</small>

## Functions

### func AngleDelta

```go
func AngleDelta(a, b float64) float64
```

AngleDelta is the signed shortest turn from a to b, in (-180, 180]: positive clockwise, negative counter-clockwise.

This is the arithmetic that makes a compass behave at the seam. Plain subtraction says the step from 359 degrees to 1 degree is -358, which is wrong by every measure that matters: it is a two-degree nudge, not most of a lap, and code that treats the difference as a magnitude (a change threshold, a smoothing filter, an animation) gets a spurious lurch once per rotation.

<small>[core/heading.go:163](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L163)</small>

### func AudioLoad

```go
func AudioLoad(track AudioTrack, opts ...AudioOpt)
```

AudioLoad replaces whatever is loaded with track and, by default, starts playing it. An empty URL is dropped here rather than sent, since every host would have to reject it separately and none could report that it had (the same rule OpenURL applies).

The status record is updated optimistically — see the package comment — and subscribers are notified before the command leaves, so a screen that re-renders on the notification already sees the new track.

<small>[core/audio.go:180](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L180)</small>

### func AudioPause

```go
func AudioPause()
```

AudioPause pauses the loaded track, keeping its position.

<small>[core/audio.go:219](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L219)</small>

### func AudioPlay

```go
func AudioPlay()
```

AudioPlay resumes the loaded track. After AudioEnded it starts over from the beginning. With nothing loaded it does nothing.

<small>[core/audio.go:211](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L211)</small>

### func AudioSeek

```go
func AudioSeek(seconds float64)
```

AudioSeek moves playback to the given second. The host clamps it to \[0, duration].

<small>[core/audio.go:239](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L239)</small>

### func AudioSetRate

```go
func AudioSetRate(rate float64)
```

AudioSetRate changes the playback speed; 1 is normal, 1.5 is the podcast listener's favorite. Non-positive rates are ignored — 0 would be a pause spelled confusingly, and the hosts reject it anyway.

<small>[core/audio.go:263](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L263)</small>

### func AudioSkip

```go
func AudioSkip(delta float64)
```

AudioSkip moves playback by delta seconds relative to where it actually is — negative to go back. The host does the arithmetic, not core: the position core knows is up to one status tick old, and a "+30s" computed from a stale number lands somewhere subtly wrong.

<small>[core/audio.go:253](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L253)</small>

### func AudioStop

```go
func AudioStop()
```

AudioStop unloads the track and releases the media session: the lock-screen controls disappear and the status returns to idle. Pause is what a user usually wants; Stop is for "sign out", "this content is no longer available", and the like.

<small>[core/audio.go:274](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L274)</small>

### func AudioToggle

```go
func AudioToggle()
```

AudioToggle plays when paused and pauses when playing — the one-button transport control. Anything else (loading, ended, error) is treated as "please play", which is what a user tapping the button means.

<small>[core/audio.go:229](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L229)</small>

### func AuditTree

```go
func AuditTree(root *Node)
```

AuditTree runs the whole-tree checks over a finished render tree: the seven accessibility findings above and the placement finding beside them.

Called by the render driver after the pass that produced the tree, beside EndRenderPass and for the same reason: both describe a \*completed\* pass, and running either over a half-built tree reports nothing except that something went wrong earlier. A no-op (one atomic load) when debug mode is off.

Hosts that drive passes through render.Manager get this for free. A hand-rolled pass loop should call it with the tree it just rendered.

nil is not an error: a pass that produced no tree has nothing to audit, which is the state a host is in before its first render.

<small>[core/a11y_audit.go:140](https://github.com/rohanthewiz/grmob/blob/master/core/a11y_audit.go#L140)</small>

### func BundledThemes

```go
func BundledThemes() map[string]*Theme
```

BundledThemes returns every theme this package ships, keyed by its Go identifier.

#### Why this exists rather than a list at each call site

It used to be a hand-written map in every test that asks a question of "the bundled themes" — the palette censuses in components, the role and frame pins in this package's own tests, several widget tests. There were a dozen of them, all spelling the same two entries, and the failure mode is the one TestBundledThemesSetEveryColorRole was written reflectively to avoid one level down: a third theme is added, a census keeps its two-entry literal, and the new palette is simply never asked the question. Nothing fails. The rule goes on being true of the themes somebody remembered.

So the list is one list, and TestBundledThemesListIsExhaustive derives it from this file's own source rather than trusting it — a package-level \*Theme var that is not in this map fails there.

A fresh map each call, for the reason ColorPalette's resolvers exist: a package-level map is reachable and writable by any importer, and a test that deleted an entry would silently narrow every census at once.

<small>[core/theme.go:1003](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L1003)</small>

### func CanPop

```go
func CanPop(ctx *Context) bool
```

CanPop reports whether there is a screen to go back to, which is what a back button or a hardware-back handler needs in order to decide between popping and exiting the app. Pop is a safe no-op when this is false; the point of asking first is to avoid rendering a control that does nothing.

<small>[core/navigation.go:356](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L356)</small>

### func Cardinal

```go
func Cardinal(deg float64) string
```

Cardinal names any bearing in degrees, normalising it first so a caller can pass an unwrapped or negative angle.

Each of the sixteen sectors is 22.5 degrees wide and \*centred\* on its point, which is the half worth stating: north is 348.75 through 11.25, not 0 through 22.5, so a bearing one degree west of north reads "N" rather than "NNW". The +11.25 before the divide is what shifts the sector boundaries off the points and onto the gaps between them.

<small>[core/heading.go:134](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L134)</small>

### func ClearConcerns

```go
func ClearConcerns()
```

ClearConcerns drops all recorded concerns. Tests call it between cases; apps can call it after acting on a dump.

<small>[core/debug.go:161](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L161)</small>

### func CompositeWalkStopsAt

```go
func CompositeWalkStopsAt(outer, inner Role) bool
```

CompositeWalkStopsAt is CompositeWalkAt for a caller that only wants the bool, and it is kept because that is what the two runtimes' walks and the audit's sentence actually ask: "does my rotation reach inside this node".

It answers true for both non-descending values, which is exactly the conflation CompositeWalkAt exists to undo — so this is safe only for a caller that has already established both roles are composites, and every caller in this repository has (AuditTree tests hasKeyboard on both ends before it asks, and wasm/verify's pins iterate KeyboardComposites()). A caller that has not should ask CompositeWalkAt and handle the third value, because for a role with no walk this returns a confident \`true\` about a rotation that does not exist.

<small>[core/role.go:845](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L845)</small>

### func DangerColor

```go
func DangerColor() string
```

<small>[core/style.go:1068](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1068)</small>

### func DismissKeyboard

```go
func DismissKeyboard(ctx *Context)
```

DismissKeyboard releases the input focus, putting the software keyboard away, as of the next render pass.

This is the other half of the old backlog item OnFocus/OnBlur opened: a tap on the background of a form can now actually close the keyboard.

	core.Box(
	    core.OnClick(func() { core.DismissKeyboard(ctx) }),
	    form,
	)

It takes a Context where Focus does not because a dismiss names no node, and therefore has no ref to carry one. Reaching for a package-level global instead would recreate exactly the bug Context's shared-pointer block documents: two apps in one process sharing one keyboard.

Unlike Focus this reaches every focusable leaf, because the field the user tapped into is one Go was never told about — the framework does not wire OnFocus unless an app asks for it. Each leaf stamps "blur" and each renderer releases focus only if that leaf actually holds it, so exactly one of them does anything.

<small>[core/focus.go:274](https://github.com/rohanthewiz/grmob/blob/master/core/focus.go#L274)</small>

### func DistanceMeters

```go
func DistanceMeters(lat1, lng1, lat2, lng2 float64) float64
```

DistanceMeters is the great-circle distance between two coordinates, by the haversine formula.

Haversine rather than the flat approximation (scale the longitude by cos(lat), then Pythagoras) because the flat one is wrong in exactly the case a map application cares about: it degrades with latitude, and it breaks completely across the antimeridian, where two points a kilometre apart are 360 degrees of longitude apart on paper. The notification filter in this file is a consumer — a device crossing 180° must not be told it has travelled 40,000km — and so is any "within n metres of here" an app writes.

Haversine rather than Vincenty, which is the next step up: Vincenty solves on the ellipsoid and is accurate to millimetres, at the cost of an iterative solver that fails to converge for antipodal points. A third of a percent is already well inside a GPS fix.

<small>[core/location.go:544](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L544)</small>

### func DumpConcerns

```go
func DumpConcerns() string
```

DumpConcerns renders the recorded concerns as a human-readable block, one line per finding. Empty string when there is nothing to report.

<small>[core/debug.go:169](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L169)</small>

### func EditBlock

```go
func EditBlock(kind richtext.BlockKind) string
```

EditBlock is the command that makes every block the selection touches the given kind.

The kind's wire value \*is\* richtext.BlockKind's, which is why that type is a string: a toolbar naming a heading and a document holding one use the same token, so there is no second table to keep in step.

<small>[core/richtext.go:129](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L129)</small>

### func EditLink

```go
func EditLink(url string) string
```

EditLink is the command that makes the selection a link to url.

A function rather than a constant because the command carries an argument, and the argument rides in the string — the command channel is one prop and widening it to a map would change the shape all four hosts read for the sake of one command. "link:" is the prefix; everything after the first colon is the URL, so a URL containing colons (every one of them does) is intact.

An empty url is EditUnlink's job and is refused here rather than sent as a link to nowhere.

<small>[core/richtext.go:116](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L116)</small>

### func Focus

```go
func Focus(ref *FocusRef)
```

Focus puts the input focus — and with it the software keyboard — on ref's node, as of the next render pass.

Called from an event handler, as the imperative counterpart to OnFocus:

	core.Button("Next", func() { core.Focus(password) })

Calling it for a ref whose node is not currently in the tree does nothing visible: no node stamps "focus", so no renderer acts. The command still consumes an epoch, which is correct — it happened, it simply had no target on screen.

A nil ref is a no-op rather than a panic, matching FocusTarget.

<small>[core/focus.go:236](https://github.com/rohanthewiz/grmob/blob/master/core/focus.go#L236)</small>

### func FocusNext

```go
func FocusNext(ref *FocusRef)
```

FocusNext moves the input focus to the field after ref in its order.

This is what the keyboard's Next action runs, and it is callable directly for the cases a keyboard cannot reach — a "Next" button drawn above the keyboard, a barcode scan that fills one field and should land in the next:

	core.Button("Next", func() { core.FocusNext(current) })

It takes the field to move \*from\* rather than reading "the focused field", because Go does not reliably know which field that is: the framework wires OnFocus only where an app asked for it, so the field the user tapped into is one Go was never told about (see focus.go). Naming the source is honest about that, and every caller has it — the keyboard action is stamped on a known field, and an app-drawn toolbar tracks the current field with OnFocus if it wants one.

At the end of the order, or for a ref in no order at all, this does nothing. A nil ref is a no-op, matching Focus and FocusTarget.

<small>[core/focus_order.go:236](https://github.com/rohanthewiz/grmob/blob/master/core/focus_order.go#L236)</small>

### func FocusPrevious

```go
func FocusPrevious(ref *FocusRef)
```

FocusPrevious moves the input focus to the field before ref in its order. See FocusNext for the shape and for why the source field is named.

It has no keyboard action behind it on any platform here: neither the Android IME nor the iOS keyboard offers a "previous" key, and SwiftUI gives no input-accessory toolbar for free. This exists for the toolbar an app draws itself, above a KeyboardAware region, which is where a back-and-forth pair of arrows actually belongs.

<small>[core/focus_order.go:248](https://github.com/rohanthewiz/grmob/blob/master/core/focus_order.go#L248)</small>

### func FormatLatLng

```go
func FormatLatLng(lat, lng float64) string
```

FormatLatLng writes the wire form of a point, "lat,lng". See FormatRegion.

<small>[core/mapview.go:411](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L411)</small>

### func FormatRegion

```go
func FormatRegion(r Region) string
```

FormatRegion writes the wire form of a region, "lat,lng,zoom".

Nothing in Go sends one today — the hosts are the writers, in Swift, Kotlin and JavaScript — and this exists so that the format has a Go statement for the harnesses to check those three against, and so a Go-side host (a test, an embedder driving the tree directly) has the same spelling available rather than inventing one.

'f' with -1 precision: the shortest form that round-trips, so 38.7223 stays "38.7223" and a whole degree stays "38". Deliberately not 'g', which switches to exponent form for small numbers — "1e-05" is a valid float in Go and is not what a hand-written host parser expects to find in a comma-separated coordinate.

<small>[core/mapview.go:406](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L406)</small>

### func GroupingContainers

```go
func GroupingContainers() []string
```

GroupingContainers returns the transparent node types, sorted. See groupingContainers.

<small>[core/stack_align.go:198](https://github.com/rohanthewiz/grmob/blob/master/core/stack_align.go#L198)</small>

### func HasSystemEventHandler

```go
func HasSystemEventHandler() bool
```

HasSystemEventHandler reports whether a host has registered a sink.

It exists for the one caller that has to behave differently when nobody is listening rather than merely have its event dropped: the permission package, where "there is no platform to ask" is a real answer a screen must draw (permission.Unavailable) and not the same thing as "the user has not decided yet". Every other sender is fire-and-forget and correctly does not care — a toast with no screen to draw on is a no-op, not a state.

<small>[core/sys_events.go:42](https://github.com/rohanthewiz/grmob/blob/master/core/sys_events.go#L42)</small>

### func HeadingActive

```go
func HeadingActive() bool
```

HeadingActive reports whether the sensor is running. Mostly useful to tests and to a debug overlay; a screen wants Heading.Active, which travels with the reading it belongs to.

<small>[core/heading.go:265](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L265)</small>

### func IsDebugMode

```go
func IsDebugMode() bool
```

IsDebugMode reports whether debug checks are active.

<small>[core/debug.go:35](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L35)</small>

### func LinearGradient

```go
func LinearGradient(x, y, z string) string
```

<small>[core/style_props.go:208](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L208)</small>

### func LocationAcquiring

```go
func LocationAcquiring()
```

LocationAcquiring is the host saying the sensor has just started — or started again — and has nothing to report yet.

#### The window it exists for

A refused start stays armed on both natives, because the grant usually arrives after the screen that wants it: the tap that asks is on that screen. When it does arrive the host re-arms and the GPS begins working, but the last thing Go was told is still the refusal, and a cold first fix is tens of seconds away. A screen written to the documented shape — \`case !loc.Available: EmptyState{Hint: loc.Error}\` — therefore prints "location permission not granted" for the whole of that window, about a sensor that is running.

The refusal is the host's statement about a run, so only the host can withdraw it, which is why this is an event a host sends rather than something core could infer: nothing in Go knows that a re-arm happened.

#### Why it is not a field on Location

Because the state it produces is one the record could already express and never reached: \*\*Active and not Received\*\* is a sensor that is running and has said nothing, which is exactly "acquiring". What was missing was a way to get BACK to it after a refusal, not a way to describe it. So this resets the report — the coordinates, the accuracy, the altitude, Received, Error — leaving Available true, because a sensor that accepted the start is one that can try and nobody yet knows better.

The reset is what makes a screen that has never heard of this improve without being touched: with Error cleared and Available true, the arm that used to print the refusal no longer matches, and the arm that draws a spinner does.

Ignored when nothing is running. A host reporting a re-arm with no consumer is a host bug, and acting on it would mean a record moving for a sensor nobody asked for.

<small>[core/location.go:380](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L380)</small>

### func LocationActive

```go
func LocationActive() bool
```

LocationActive reports whether the sensor is running. Mostly useful to tests and to a debug overlay; a screen wants Location.Active, which travels with the fix it belongs to.

<small>[core/location.go:270](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L270)</small>

### func NormalizeDegrees

```go
func NormalizeDegrees(deg float64) float64
```

NormalizeDegrees folds any angle into \[0, 360). Negative angles and angles past a full turn both come back on the circle, so -90 is 270 and 730 is 10.

math.Mod alone is not enough: it keeps the sign of its first argument, so -90 comes back as -90 rather than 270.

<small>[core/heading.go:144](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L144)</small>

### func OnAudioStatus

```go
func OnAudioStatus(fn func(AudioStatus)) (cancel func())
```

OnAudioStatus subscribes fn to every status change. The returned function cancels the subscription. fn runs on whichever goroutine delivered the change — a host bridge call, or the app's own AudioLoad — and must not block; the usual body is a State write or a RequestRender.

Most screens want hooks.UseAudio instead, which subscribes once per component and re-renders on each change.

<small>[core/audio.go:293](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L293)</small>

### func OnDeepLink

```go
func OnDeepLink(fn func(url string)) (cancel func())
```

OnDeepLink subscribes fn to inbound URLs. The returned function cancels the subscription.

	cancel := core.OnDeepLink(func(url string) {
	    if id, ok := strings.CutPrefix(url, "grmob://lesson/"); ok {
	        navigate(id)
	    }
	})

fn runs on whichever goroutine delivered the event — a host bridge call — and must not block; the usual body parses the URL and asks for a render.

An empty or absent "url" field is dropped rather than delivered as "": a subscriber cannot do anything useful with no address, and a shell that reported one has a bug this hides less well than it would pass on.

<small>[core/deeplink.go:76](https://github.com/rohanthewiz/grmob/blob/master/core/deeplink.go#L76)</small>

### func OnHeading

```go
func OnHeading(fn func(Heading)) (cancel func())
```

OnHeading subscribes fn to heading changes. The returned function cancels the subscription.

Subscribing does not start the sensor: the two are separate on purpose, because a screen that wants to \*display\* a heading and a screen that wants the sensor \*on\* are not always the same screen (a background service, a second view of one reading). hooks.UseHeading does both, which is what most callers want.

fn runs on whichever goroutine delivered the reading — a host bridge call — and must not block; the usual body is a RequestRender.

<small>[core/heading.go:291](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L291)</small>

### func OnHostEvent

```go
func OnHostEvent(name string, fn func(data map[string]any)) (cancel func())
```

OnHostEvent subscribes fn to host events named name. The returned function cancels the subscription; calling it more than once is harmless.

Subscriptions are process-wide, like the system-event handler and for the same reason: the thing on the far side of the channel is one physical device with one audio output, one keystore, one location, so there is no context tree to scope them to. A component that subscribes during render must therefore guard against subscribing again on the next pass — hooks.UseAudio shows the pattern (a hook slot remembers that it did).

<small>[core/host_events.go:60](https://github.com/rohanthewiz/grmob/blob/master/core/host_events.go#L60)</small>

### func OnLifecycle

```go
func OnLifecycle(fn func(LifecycleState)) (cancel func())
```

OnLifecycle subscribes fn to lifecycle transitions. The returned function cancels the subscription; calling it more than once is harmless.

Like OnAudioStatus and OnHostEvent, the subscription is process-wide — there is one app and one screen, so there is no context tree to scope it to. A component that subscribes during render must guard against subscribing again on the next pass; hooks.UseLifecycle does that.

fn runs on whichever goroutine delivered the event (see the threading note in host\_events.go) and must not block. Writing State and calling RequestRender are fine from there.

<small>[core/lifecycle.go:92](https://github.com/rohanthewiz/grmob/blob/master/core/lifecycle.go#L92)</small>

### func OnLocation

```go
func OnLocation(fn func(Location)) (cancel func())
```

OnLocation subscribes fn to location changes. The returned function cancels the subscription.

Subscribing does not start the sensor, for the reason OnHeading does not: wanting to \*see\* a position and wanting the GPS \*on\* are not always the same screen. hooks.UseLocation does both.

fn runs on whichever goroutine delivered the fix — a host bridge call — and must not block; the usual body is a RequestRender.

<small>[core/location.go:293](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L293)</small>

### func OpenURL

```go
func OpenURL(url string)
```

OpenURL asks the host to open a URL outside the app — the platform's own browser, mail composer, dialer or map, whichever the scheme names.

It is a system event (see sys\_events.go) rather than a node, for the same reason ShowToast is: nothing about it is part of the view tree. There is no element to reconcile, no state to diff, and the thing it ultimately reaches is an OS-level facility the app does not own. So it travels one way, as a named payload handed to whatever host is driving the app, and — like a toast — it is callable from any goroutine and takes no Context.

Each host maps the event onto its platform's own hand-off:

	Android   Intent(ACTION_VIEW, uri), started with FLAG_ACTIVITY_NEW_TASK
	iOS       UIApplication.shared.open(url)
	Browser   window.open(url, "_blank", "noopener,noreferrer")
	Headless  nothing (no handler registered — see SendSystemEvent)

#### Why "outside the app" is the whole contract

The three platforms differ on almost everything about in-app browsing — Custom Tabs versus SFSafariViewController versus an iframe — and agree completely on handing a URL to the system. So this promises only the part that is portable. An app that needs an embedded browser wants a view node, which is a different (and much larger) feature.

#### Failure is silent, and that is deliberate

A malformed URL, a scheme no app on the device claims (a \`tel:\` link on a tablet with no dialer), or a host that registered no handler at all: none of these come back. There is no return channel on a system event, and synthesizing one would mean either blocking the caller on an OS round trip or inventing a callback protocol for a fire-and-forget gesture. Callers that must know whether a link is reachable have to decide that before calling — which in practice means not rendering the affordance at all, exactly as core.Button does with a nil handler.

An empty url is dropped here rather than sent, since every host would have to reject it separately and none could report that it had.

<small>[core/openurl.go:41](https://github.com/rohanthewiz/grmob/blob/master/core/openurl.go#L41)</small>

### func ParseLatLng

```go
func ParseLatLng(s string) (lat, lng float64, ok bool)
```

ParseLatLng reads a host's "lat,lng" payload, for OnMapTap. Same contract as ParseRegion: false rather than zeros.

<small>[core/mapview.go:380](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L380)</small>

### func PlacingContainers

```go
func PlacingContainers() []string
```

PlacingContainers returns the node types that place their children, sorted.

Sorted for the reason htmlout's OverlayTypes is: a test looping over a map reports in a different order every run, and a census that names the offender wants one order.

<small>[core/stack_align.go:192](https://github.com/rohanthewiz/grmob/blob/master/core/stack_align.go#L192)</small>

### func Pop

```go
func Pop(ctx *Context)
```

Pop removes the top route, discarding its state, and reveals the one below. It is a no-op at the root — the stack is never left empty, because Navigator has nothing to render then.

<small>[core/navigation.go:240](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L240)</small>

### func PopToRoot

```go
func PopToRoot(ctx *Context) bool
```

PopToRoot unwinds to the bottom of the stack, discarding the state of every frame above it, and returns whether anything was popped.

It differs from Reset in exactly one way, and it is the way that matters: the root frame is the one already there, state and all. Reset(ctx, root) would look identical on screen and quietly reset the root's scroll position, selected tab and form contents. Reach for PopToRoot to escape a deep drill-down ("Done" out of a five-level settings tree), and for Reset to end a session.

<small>[core/navigation.go:321](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L321)</small>

### func PrimaryColor

```go
func PrimaryColor() string
```

PrimaryColor and DangerColor are the theme-blind convenience accessors that predate Context.Theme(). They answer for the \*default\* theme's roles and nothing else, so a screen under WithTheme still gets the default palette's hexes from them — which is why nothing in components or core calls either, and why new code should read ctx.Theme().Colors instead.

They read DefaultTheme rather than repeating its literals. Both used to be hard-coded, and the copy was not free: when Colors.Primary moved to Apple's accessible blue (white over systemBlue was 4.02:1, under WCAG AA, and the theme's own Button base declares white), this function kept the old hex — so examples/chat, its one caller, went on painting white on a fill nobody could read it on, in the one place the fix could not reach.

<small>[core/style.go:1067](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1067)</small>

### func Push

```go
func Push(ctx *Context, route func(*Context) View)
```

Push adds a route on top of the stack. The screen underneath keeps its state and is restored intact by the matching Pop.

Like every mutation here it ends in RequestRender rather than a bare MarkDirty, which these used to do. Marking alone is enough only when a pass is already guaranteed to follow — true for a tap, since the native dispatch path re-renders on the way out, and false for a navigation triggered from anywhere else: an effect goroutine resolving a deep link, a timeout dismissing a splash screen, a websocket pushing the user to a call screen. Those marked the tree dirty and then waited for an unrelated event to notice.

<small>[core/navigation.go:230](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L230)</small>

### func ReceiveAudioStatus

```go
func ReceiveAudioStatus(s AudioStatus)
```

ReceiveAudioStatus is the typed entry point for a host that builds the status in Go (a test, an embedder). The JSON hosts arrive through ReceiveHostEvent("audio\_status", ...) instead, which decodes into this.

Track metadata is not something a host reports — it only ever echoes the URL — so the incoming Track's URL is matched against the loaded track: the same URL keeps the title, artist and artwork the app supplied; a different one (a host playing something the app did not load, which should not happen but must not corrupt the record) keeps only the URL.

<small>[core/audio.go:315](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L315)</small>

### func ReceiveHeading

```go
func ReceiveHeading(h Heading)
```

ReceiveHeading is the typed entry point for a host that builds the reading in Go (a test, an embedder). The JSON hosts arrive through ReceiveHostEvent("heading", ...) instead, which decodes into this.

Bearings are normalised here rather than trusted: a host that reports 360.0 at north, or a negative azimuth (which Android's getOrientation returns for half the circle — it answers in radians over -pi..pi), would otherwise leak an out-of-range angle into every consumer's arithmetic. This is the one place all four hosts funnel through, so it is the one place the invariant "Magnetic is in \[0, 360)" can actually be established.

Active is core's bookkeeping and is overwritten from the reference count, not taken from the caller: a host does not know how many screens asked.

<small>[core/heading.go:317](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L317)</small>

### func ReceiveHostEvent

```go
func ReceiveHostEvent(name string, data map[string]any)
```

ReceiveHostEvent delivers one event from the host. Names core owns are consumed first; then every subscriber for the name runs, outside the registry lock so a subscriber may subscribe or cancel from inside its own handler without deadlocking.

An event nobody consumes is logged rather than dropped silently — unlike an unknown system event, which a host drops because a newer app may legitimately send what an older shell does not understand, an unknown host event means the shell is sending traffic the app never asked for, which is worth a line in the log during development.

<small>[core/host_events.go:93](https://github.com/rohanthewiz/grmob/blob/master/core/host_events.go#L93)</small>

### func ReceiveLifecycle

```go
func ReceiveLifecycle(s LifecycleState)
```

ReceiveLifecycle is the typed entry point for a host that reports in Go (a test, an embedder). The JSON hosts arrive through ReceiveHostEvent("lifecycle", ...), which decodes into this.

A state that is not one of the three is dropped rather than stored: a newer shell reporting a fourth state to an older app must not leave CurrentLifecycle answering something no switch in that app has an arm for. A repeat of the current state is absorbed silently (see the file comment). Subscribers are notified outside the lock, so one may read CurrentLifecycle, subscribe or cancel from inside its handler.

<small>[core/lifecycle.go:115](https://github.com/rohanthewiz/grmob/blob/master/core/lifecycle.go#L115)</small>

### func ReceiveLocation

```go
func ReceiveLocation(l Location)
```

ReceiveLocation is the typed entry point for a host that builds the fix in Go (a test, an embedder). The JSON hosts arrive through ReceiveHostEvent("location", ...), which decodes into this.

Coordinates are normalised here rather than trusted, which is this function's reason for existing beyond plumbing: it is the one place all four hosts funnel through, so it is the only place the invariant can be established. Latitude is clamped to ±90 and longitude wrapped into (-180, 180] — two rules, because they are two different facts about the sphere. A latitude past the pole is not a place; a longitude past the antimeridian is the same meridian spelled the long way round, and clamping it would move the device to the far side of the Pacific.

Active is core's bookkeeping and is overwritten from the reference count rather than taken from the caller: a host does not know how many screens asked.

<small>[core/location.go:322](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L322)</small>

### func Replace

```go
func Replace(ctx *Context, route func(*Context) View)
```

Replace swaps the top route for another without changing the stack depth, discarding the outgoing route's state. Use it for a step that should not be returned to — the "logged in" screen after a login form, so Back skips the form rather than showing it again.

<small>[core/navigation.go:264](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L264)</small>

### func ReportConcern

```go
func ReportConcern(kind, detail string)
```

ReportConcern records a concern from outside this package.

The detection sites for most concern kinds live in core, so they call upsertConcern directly; the panic guards in the render driver do not, and a recovered panic is exactly the sort of silent-by-design event the concern list exists to surface. Callers should gate on IsDebugMode themselves — detail strings usually cost a Sprintf to build, and there is no reason to pay for one in a release build.

Deduplicated on kind+detail like every other concern, so a failure that repeats every frame occupies one entry with a rising count.

<small>[core/debug.go:137](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L137)</small>

### func Reset

```go
func Reset(ctx *Context, route func(*Context) View)
```

Reset discards the entire stack and starts over with route as the only frame. This is the log-out / onboarding-complete operation: every frame's hook state is thrown away and every background resource its hooks started is stopped, so nothing from the previous session survives to be re-displayed.

The new root is a fresh frame even when route is the same function the old root ran, which is the point — resetting to the login screen must not show the previous tenant's half-filled form.

What Reset does not touch is state the app deliberately kept outside the stack: hooks on the context hosting the Navigator, package-level stores, the database. Those outlive navigation by construction, and clearing them is the app's call, not the router's.

<small>[core/navigation.go:300](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L300)</small>

### func RunEditorCommand

```go
func RunEditorCommand(ref *EditorRef, command string)
```

RunEditorCommand sends one command to ref's editor, to be applied to whatever the host's own selection is at the time it lands.

Called from an event handler, as a toolbar button's whole body:

	core.Button("Bold", func() { core.RunEditorCommand(ref, core.EditBold) })

The command strings each editor understands are its own; see the Edit\* constants below for the census. A command an editor does not know is deliberately a no-op on every host rather than an error — the alternative is a screen that crashes because a toolbar outgrew its editor.

Issuing a command for a ref whose editor is not currently in the tree does nothing visible: no node stamps it, so no host acts. The command still consumes an epoch, which is correct — it happened, it simply had no target on screen.

A nil ref is a no-op rather than a panic, matching EditorTarget.

<small>[core/editor.go:168](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L168)</small>

### func SendSystemEvent

```go
func SendSystemEvent(name string, data map[string]any)
```

SendSystemEvent delivers one event to the host, synchronously on the caller's goroutine. The read is under RLock so senders never contend with each other, only with the (rare) handler swap.

<small>[core/sys_events.go:51](https://github.com/rohanthewiz/grmob/blob/master/core/sys_events.go#L51)</small>

### func SetDebugMode

```go
func SetDebugMode(on bool)
```

SetDebugMode turns the debug checks on or off. Zero overhead when off: every check site guards with IsDebugMode before doing any work.

<small>[core/debug.go:30](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L30)</small>

### func SetSystemEventHandler

```go
func SetSystemEventHandler(fn func(name string, data map[string]any))
```

SetSystemEventHandler installs the host's sink for system events. Passing nil detaches it, after which SendSystemEvent drops events silently — the right behavior for a headless run, where there is no screen to draw on.

<small>[core/sys_events.go:28](https://github.com/rohanthewiz/grmob/blob/master/core/sys_events.go#L28)</small>

### func ShowToast

```go
func ShowToast(msg string, opts ...ToastOpt)
```

<small>[core/toast.go:14](https://github.com/rohanthewiz/grmob/blob/master/core/toast.go#L14)</small>

### func StackDepth

```go
func StackDepth(ctx *Context) int
```

StackDepth reports how many frames are on the stack.

Before the Navigator's first render it counts only what the app itself pushed — 0 for an app that has not navigated yet, because the initial route is installed lazily by that first render. Afterwards it is at least 1.

<small>[core/navigation.go:346](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L346)</small>

### func StartHeading

```go
func StartHeading()
```

StartHeading asks the host to begin reporting compass headings, and is balanced by StopHeading — see "Reference counting" in the package comment.

On a browser this is also the permission moment: Safari's DeviceOrientationEvent.requestPermission must be called from inside a user gesture, and the host runtime makes that call here rather than exposing a separate permission API. A start that happens outside a gesture is refused by the browser and comes back as an "available: false" event carrying the reason, which is why Heading.Error exists.

<small>[core/heading.go:217](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L217)</small>

### func StartLocation

```go
func StartLocation()
```

StartLocation asks the host to begin reporting position fixes, and is balanced by StopLocation — the reference counting heading.go describes, and for the same reason twice over: GPS is the most expensive sensor on the device, and two screens that each want the user's position must each be able to let go without blinding the other.

A host that needs an OS permission asks for one here if it has to, and a refusal arrives as an \`available: false\` event with a reason. An app that wants to control when that dialog appears asks first — see the type comment.

<small>[core/location.go:213](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L213)</small>

### func StopHeading

```go
func StopHeading()
```

StopHeading releases one Start. The sensor is turned off when the last holder lets go; extra Stops are ignored rather than driving the count negative.

<small>[core/heading.go:239](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L239)</small>

### func StopLocation

```go
func StopLocation()
```

StopLocation releases one Start. The sensor is turned off when the last holder lets go; extra Stops are ignored rather than driving the count negative.

<small>[core/location.go:244](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L244)</small>

### func UseFocusOrder

```go
func UseFocusOrder(ctx *Context, refs ...*FocusRef)
```

UseFocusOrder declares the order the input focus walks through a set of fields. Call it in the component that renders those fields, above them:

	core.UseFocusOrder(ctx, email, password, confirm)

Two things follow from it. core.FocusNext and core.FocusPrevious can move relative to any ref in the list; and every field but the last advertises the keyboard's "next" action, which advances to the field after it. The last field advertises nothing new, so a form's final field keeps its own submit action — see FocusTarget for the rule when a field has both.

#### Why "above them" is not merely style

Membership is read while a field's props are stamped, so a field rendered before this call runs sees the \*previous\* pass's membership. Hooks belong at the top of a render function anyway, and the fields of a form are rendered by the component that owns their refs, so the natural shape is already the correct one — but a call moved into a child component that renders after the fields would stamp a form that never advances.

#### It reserves no hook slot

Despite the name, this is not a hook: everything it records lives on the refs, which are slot-stable already (see UseFocusRef), so there is nothing for a slot to hold. The Use prefix says where it belongs — inside a render function, on every pass — which is the part a caller has to get right. The consequence of not being a hook is only ever permissive: calling it conditionally is safe, where a real hook would drift the cursor.

A nil ref in the list is skipped rather than panicking, so \`core.UseFocusOrder(ctx, email, maybeRef, confirm)\` degrades to the order without it — the same tolerance MaybeProp and FocusTarget have. A ref belonging to a different app's context is skipped for the same reason focusState is per-app: two apps in one process must not share an order.

Listing a ref twice is allowed and the last position wins; there is no meaningful "field visited twice" and no reason to panic over a typo.

<small>[core/focus_order.go:101](https://github.com/rohanthewiz/grmob/blob/master/core/focus_order.go#L101)</small>

### func WithConfigOpt

```go
func WithConfigOpt(c *AppConfig) func(*Context)
```

<small>[core/context.go:326](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L326)</small>

### func WithThemeOpt

```go
func WithThemeOpt(t *Theme) func(*Context)
```

<small>[core/context.go:320](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L320)</small>

### func WrapLongitude

```go
func WrapLongitude(lng float64) float64
```

WrapLongitude folds a longitude into (-180, 180] by going round rather than by stopping at the edge: 190° east is 170° west, the same meridian, and a clamp would move the point to the antimeridian instead.

Exported because every consumer of a coordinate needs it and getting it wrong is silent — a map centred 20 degrees from where it was asked to be still looks like a map. comps.StaticMap does the same arithmetic for the same reason.

math.Mod keeps the sign of its first argument, so a negative input stays west, and the two adjustments are what carry a value past ±180 round to the other side.

<small>[core/location.go:508](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L508)</small>

## Types

### type AlignItems

```go
type AlignItems string
```

<small>[core/style.go:1130](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1130)</small>

#### func AlignItemsValues

```go
func AlignItemsValues() []AlignItems
```

AlignItemsValues returns every declared AlignItems, in declaration order.

Named for its values rather than its type because the type name is taken — the same collision AlignItemsProp in style\_props.go had to work around.

Cross-axis placement. AlignItemsStretch is the member that behaves unlike the rest on every native: a stretched child is not \*placed\* differently, it is \*measured\* differently, so neither SwiftUI's alignment nor Compose's Alignment enum can express it and both runtimes handle it off the dispatch (a fill modifier on the child, a solver branch). A stretch arm in those switches is therefore expected to be a no-op — but an explicit no-op arm is how a reader learns the value was considered and handled elsewhere, which is the whole argument for holding a switch to a list.

<small>[core/alignment.go:153](https://github.com/rohanthewiz/grmob/blob/master/core/alignment.go#L153)</small>

#### func (AlignItems) Apply

```go
func (a AlignItems) Apply(s *Style)
```

The three named flex-container types are StyleProps in their own right, so the spelling that mirrors core.Align(...) and core.Justify(...) works too:

	core.Column(core.AlignItems(core.AlignItemsCenter), ...)

Without these methods that expression is a type conversion producing a bare string value, which containerNode's PropsAndChildren dispatch cannot recognize and drops — silently outside debug mode. It was the most natural thing to write and it compiled, so an app shipped with every one of its AlignItems lost and its columns left-packed on both natives. Making the value itself apply removes the trap rather than documenting it.

<small>[core/style_props.go:249](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L249)</small>

### type Alignment

```go
type Alignment string
```

<small>[core/style.go:1107](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1107)</small>

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

#### func Alignments

```go
func Alignments() []Alignment
```

Alignments returns every declared Alignment, in declaration order.

This is the census list — the one that must equal the const block exactly — and it is deliberately the only one of the four that no renderer is held to. Alignment carries two roles that no single dispatch serves:

	value    | text-align role      | cross-axis role
	---------+----------------------+---------------------------------
	start    | leading edge         | items packed to the start edge
	center   | centered             | items centered on the cross axis
	end      | trailing edge        | items packed to the end edge
	justify  | justified text       | none
	stretch  | none                 | items filled to the cross extent
	baseline | none                 | items aligned on their baselines

Style.Align feeds both: it is the text alignment of a Text node, and it is also the fallback every renderer's vertical-stacking containers read when AlignItems is unset (crossAxisValue in Renderer.swift; htmlout/crossaxis.go states the DOM pair's version). So a text dispatch that was required to answer for "stretch", or a cross-axis dispatch required to answer for "justify", would be made to write an arm that can never mean anything. TextAlignments is the subset that is a real text alignment; AlignItemsValues is the vocabulary the cross-axis dispatches actually dispatch on.

The two roles are not split into two Go types because that would be a breaking change for any caller already passing AlignStretch to Align(), and because the split is real only at the point of \*use\* — Style has one Align field, and which role it plays depends on the node it lands on.

<small>[core/alignment.go:81](https://github.com/rohanthewiz/grmob/blob/master/core/alignment.go#L81)</small>

#### func TextAlignments

```go
func TextAlignments() []Alignment
```

TextAlignments returns the Alignments that name a real text alignment, in declaration order: the coverage every renderer's text dispatch owes.

AlignStretch and AlignBaseline are excluded because there is no such thing as stretched or baseline-aligned text — CSS text-align has no such keyword, SwiftUI's TextAlignment has three members, and Compose's TextAlign has no analogue either. They are cross-axis values that share the type; a text dispatch that receives one is right to fall through to its default.

AlignJustify \*is\* included, and is the value that made this list worth writing. Before it existed, justified text rendered on exactly one of the four targets: Renderer.kt mapped it to TextAlign.Justify, Renderer.swift fell through to .leading, htmlout emitted no declaration at all, and the WASM runtime did not read Align in the first place. One value, four behaviors, and nothing anywhere that could notice.

Being on this list does not mean a target can honor the value — SwiftUI genuinely cannot justify text — it means the target has to \*say\* what it does with it. An explicit arm that falls back to leading, with a comment naming the platform limit, is coverage; silence is not.

<small>[core/alignment.go:112](https://github.com/rohanthewiz/grmob/blob/master/core/alignment.go#L112)</small>

### type AppConfig

```go
type AppConfig struct {
	Name        string
	Description string
	Version     string
	Locale      string
	Author      string
	Meta        map[string]string
}
```

<small>[core/context.go:121](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L121)</small>

### type AudioOpt

```go
type AudioOpt interface {
	Apply(*audioLoadConfig)
}
```

AudioOpt configures AudioLoad.

<small>[core/audio.go:130](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L130)</small>

#### func AudioAutoplay

```go
func AudioAutoplay(on bool) AudioOpt
```

AudioAutoplay controls whether AudioLoad starts playing as soon as the host can. The default is true: a tap on "play" that only buffered would need a second tap, and the browser only allows autoplay from inside a user gesture anyway — which a tap handler is.

<small>[core/audio.go:148](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L148)</small>

#### func AudioStartAt

```go
func AudioStartAt(seconds float64) AudioOpt
```

AudioStartAt begins playback at the given second instead of at 0 — the "resume where you left off" option. The host clamps it to the track.

<small>[core/audio.go:154](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L154)</small>

#### func AudioWithRate

```go
func AudioWithRate(rate float64) AudioOpt
```

AudioWithRate sets the initial playback speed; 1 is normal. Non-positive values are ignored.

<small>[core/audio.go:164](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L164)</small>

### type AudioState

```go
type AudioState string
```

AudioState is the player's phase, as the host last reported it (or as core set optimistically; see the package comment).

<small>[core/audio.go:55](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L55)</small>

```go
const (
	AudioIdle    AudioState = "idle"    // nothing loaded
	AudioLoading AudioState = "loading" // buffering, or waiting for the host to answer a Load
	AudioPlaying AudioState = "playing"
	AudioPaused  AudioState = "paused"
	AudioEnded   AudioState = "ended" // played to the end; Play or Seek starts it again
	AudioError   AudioState = "error" // see AudioStatus.Error
)
```

### type AudioStatus

```go
type AudioStatus struct {
	Track    AudioTrack
	State    AudioState
	Position float64
	Duration float64
	Rate     float64
	Error    string // set when State is AudioError
}
```

AudioStatus is everything the app can know about playback. Position and Duration are seconds; Duration is 0 until the host has learned it (a streamed file reports it once the headers arrive). Rate is the playback speed, 1 being normal.

<small>[core/audio.go:82](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L82)</small>

#### func CurrentAudioStatus

```go
func CurrentAudioStatus() AudioStatus
```

CurrentAudioStatus returns the last known status. Safe from any goroutine.

<small>[core/audio.go:280](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L280)</small>

#### func (AudioStatus) Loaded

```go
func (s AudioStatus) Loaded() bool
```

Loaded reports whether a track is loaded, in any state but idle.

<small>[core/audio.go:92](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L92)</small>

#### func (AudioStatus) Progress

```go
func (s AudioStatus) Progress() float64
```

Progress is Position as a fraction of Duration, 0 while Duration is unknown — what a seek slider wants.

<small>[core/audio.go:96](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L96)</small>

### type AudioTrack

```go
type AudioTrack struct {
	URL        string
	Title      string
	Artist     string // the speaker, the band, the podcast host
	Album      string // the series, the show
	ArtworkURL string
}
```

AudioTrack names what to play and how the platform should describe it on the lock screen. Only URL is required; the rest is metadata the media session shows (and, on a phone, is the difference between a notification reading "Unknown" and one reading the sermon's title).

<small>[core/audio.go:70](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L70)</small>

### type BehaviorProp

```go
type BehaviorProp interface {
	Apply(*Context, *Node)
}
```

BehaviorProp attaches event behavior to a node. Apply takes the rendering Context (unlike StyleProp) because registering the handler needs the context's callback registry — the registry is per-app state on the context tree, not a package global.

<small>[core/behavioral_props.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L7)</small>

#### func CommentPrefix

```go
func CommentPrefix(prefix string) BehaviorProp
```

CommentPrefix sets the line-comment marker EditCommentLine toggles — "//" for Go, "#" for shell and Python, "--" for SQL.

An empty prefix makes EditCommentLine a no-op, which is the right answer for a language that has no line comments (JSON) rather than inserting a marker that would make the document invalid.

<small>[core/codeeditor.go:168](https://github.com/rohanthewiz/grmob/blob/master/core/codeeditor.go#L168)</small>

#### func EditorTarget

```go
func EditorTarget(ref *EditorRef) BehaviorProp
```

EditorTarget marks the editing surface it is applied to as ref's.

A nil ref returns a nil prop rather than panicking — leafNode skips a nil item — so \`core.EditorTarget(maybeRef)\` degrades to an editor no toolbar can command instead of crashing a render pass.

Applying it to anything but a CodeEditor or a RichTextEditor is harmless and pointless: the stamp lands and no renderer reads it.

<small>[core/editor.go:115](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L115)</small>

#### func FocusTarget

```go
func FocusTarget(ref *FocusRef) BehaviorProp
```

FocusTarget marks the node it is applied to as ref's node.

It is an ordinary BehaviorProp, so it composes with OnFocus, OnBlur and everything else in the same argument list:

	core.Input(v, "", onChange, core.FocusTarget(email), core.OnBlur(...))

A nil ref returns a nil prop rather than panicking — leafNode and containerNode both skip a nil item (MaybeProp's contract), so \`core.FocusTarget(maybeRef)\` degrades to an unnamed field instead of crashing a render pass.

Applying it to a node the platform never focuses (a Row, a Checkbox) is harmless but pointless: the stamp lands and no renderer reads it.

<small>[core/focus.go:184](https://github.com/rohanthewiz/grmob/blob/master/core/focus.go#L184)</small>

#### func KeyboardAware

```go
func KeyboardAware() BehaviorProp
```

KeyboardAware makes a region yield to the software keyboard instead of being covered by it: while the keyboard is up, the node it is applied to ends where the keyboard begins.

	core.Scroll(
	    core.KeyboardAware(),
	    form,
	)

##### The two shapes it takes

On a \*scrolling\* node (Scroll, List) it shrinks the viewport. The content does not move on its own — but because the viewport now ends above the keyboard, the platform's own "scroll the focused field into view" behavior (Compose's BasicTextField, SwiftUI's ScrollView) lands the field somewhere the user can see, which it cannot do while the viewport still claims the rows the keyboard is sitting on. This is the form case.

On any \*other\* node it lifts that subtree whole. That is the case for a screen with something docked at the bottom — a chat composer, a checkout bar — which is outside the scrolling region by construction and would otherwise be the one thing the keyboard covers. Applied to a whole screen's column, it is the classic "the app resizes for the keyboard" behavior, asked for explicitly and by one screen at a time.

##### What it deliberately does not do

It does not dismiss the keyboard on a tap outside — but that is now a thing an app can ask for directly, which it was not when this prop was written:

	core.Box(
	    core.OnClick(func() { core.DismissKeyboard(ctx) }),
	    form,
	)

Keeping the two separate is the point. This prop is about \*layout\* — which region yields the space the keyboard takes — and dismissal is about focus. A chat composer wants the inset and emphatically does not want a stray tap closing the keyboard between messages; a settings form wants the opposite. Folding one into the other would take that choice away.

(Dragging a keyboard-aware scroll region also dismisses it on iOS — see below — which is the platform's own gesture, not a handler of ours.)

##### What each platform does with it

	Android   Modifier.imePadding() on the node, injected at the one funnel
	          every node passes through. It needs the window to have stopped
	          fitting the system windows itself, which the demo activity does
	          (enableEdgeToEdge plus windowSoftInputMode="adjustResize"); an
	          app that skips both gets the platform's whole-window resize
	          instead and this prop then reads a consumed, zero-height inset.
	iOS       SwiftUI treats the keyboard as its own safe-area region and
	          insets for it by itself, so the shrink is the platform default
	          with or without the flag. What the flag adds is
	          .scrollDismissesKeyboard(.interactively) on the two scrolling
	          node types: dragging the region puts the keyboard away.
	HTML/WASM Nothing. A browser has no software keyboard to inset for, and
	          the exported page scrolls the focused field into view natively.

That asymmetry is why this is a flag and not simply what Scroll always does: on iOS the shrink is free, on Android it costs a window-level opt-in and a per-region decision about which thing should move, and a Go app should be able to name the region without knowing either.

It is also why SafeArea does not carry it. The safe area on Android is WindowInsets.safeDrawing, which bundles the IME in with the system bars — applied there it would resize every screen whole and, worse, consume the inset so that a Scroll asking for it received nothing. The renderer subtracts the IME from that set for exactly this reason, leaving the keyboard to whichever node asked for it: the same split SwiftUI makes.

<small>[core/keyboard.go:74](https://github.com/rohanthewiz/grmob/blob/master/core/keyboard.go#L74)</small>

#### func LineNumbers

```go
func LineNumbers() BehaviorProp
```

LineNumbers turns on the gutter.

The gutter is drawn by the host rather than being part of the buffer, which is the only arrangement that works: numbers inside the text would be selectable, copyable and editable, and a buffer whose first four columns are not the user's is not the buffer.

<small>[core/codeeditor.go:134](https://github.com/rohanthewiz/grmob/blob/master/core/codeeditor.go#L134)</small>

#### func On

```go
func On(event string, handler func()) BehaviorProp
```

<small>[core/behavioral_props.go:16](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L16)</small>

#### func OnBack

```go
func OnBack(handler func()) BehaviorProp
```

OnBack claims the platform's system back — Android's back button and back gesture — for as long as the node carrying it is on screen. While any such node is, a back press runs the innermost one's handler instead of the platform default; when none is, back does what the platform would have done anyway, which on Android is to leave the app.

core.Navigator attaches one to the route it shows whenever core.CanPop is true, so a pushed screen pops on back with no app code. comps.AppBar attaches its back arrow's action, and comps.Drawer its OnDismiss while open.

##### A prop, not a host event

The other things a shell reports without a callback — lifecycle, a deep link — arrive as host events. Back cannot, because Android has to know before the press whether the app will take it: OnBackPressedDispatcher consults its callbacks' enabled flags synchronously on the UI thread, and falls through to finishing the Activity if none is enabled. A host event reaches Go after that decision, so a shell built on one would need a second, Go→host "enabled" signal kept in step with the app's state. A prop already is that signal: its presence in the tree is the enabled flag, and the tree diff keeps the shell's copy current with no channel of its own.

##### Innermost wins

Compose's BackHandler gives priority to the handler registered most recently. Handlers composed in one pass register parent before child, so the innermost wins; one composed later — a drawer that has just opened — outranks every handler already on screen.

	Navigator route root  onBack = Pop            outermost, runs last
	  AppBar row          onBack = AppBar.OnBack
	  Drawer panel layer  onBack = OnDismiss      while Open; runs first

The one ordering this gets wrong is a parent that gains the prop after its descendants already hold one: it registers last and outranks them. Keep OnBack on nodes whose lifetimes nest the way their handlers should, which the three above do.

##### Its own callback IDs

The handler is an ordinary void callback on the wire, but its ID comes from a sequence of its own ("back\_cb\_N" rather than "cb\_N"). A second quick press is dispatched with the ID from before the first press's patches landed, and a shared sequence would have re-assigned that ID to a tap on the screen the first press revealed. See callbackRegistry.registerBack.

##### Hosts

	Android  RenderNode wraps the node in androidx's BackHandler. A node
	         hidden with Display none is not composed, so its handler is
	         inactive while hidden. A Modal needs none: the Dialog window
	         reports back through the Modal's own onDismiss. The manifest
	         opts in to predictive back, which BackHandler supports.
	iOS      nothing. There is no system back; the edge swipe belongs to a
	         UINavigationController, which the SwiftUI renderer does not use.
	Web      the browser's back button. While any node carrying the prop is
	         on screen, the runtime keeps one history entry of its own above
	         the page's; a back press consumes it and runs the innermost
	         handler (the last claimant in document order, which is the same
	         nesting Compose ranks by), and the entry is pushed again if a
	         claim is still on screen afterwards. An open Modal's onDismiss
	         counts as a claim, matching Android's Dialog. A page that owns
	         its history sets window.GrMobBrowserBack = false. htmlout does
	         not export the prop.

<small>[core/behavioral_props.go:124](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L124)</small>

#### func OnBlur

```go
func OnBlur(handler func()) BehaviorProp
```

OnBlur fires when the node loses input focus. See OnFocus for the pairing and the ordering caveat.

<small>[core/behavioral_props.go:166](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L166)</small>

#### func OnClick

```go
func OnClick(handler func()) BehaviorProp
```

<small>[core/behavioral_props.go:25](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L25)</small>

#### func OnEndReached

```go
func OnEndReached(handler func()) BehaviorProp
```

OnEndReached fires when the user scrolls within a few rows of the bottom of a List: the "fetch the next page" edge that turns a manual comps.LoadMore button into an infinite feed.

	core.List(
	    core.OnEndReached(pager.LoadNext),
	    rows...,
	)

##### The debounce, and why it is here rather than in four renderers

The edge is a \*scroll position\*, so every renderer reports it more than once for the same bottom: Compose's snapshot flow emits on each new last-visible index, SwiftUI's .onAppear re-fires when a row is recycled back into view, and an IntersectionObserver fires on entry and on every resize that keeps the sentinel visible. A slow fetch therefore sees two or three calls before its first page lands, and an offset pager answers that by loading page 2 twice.

The fix is one line of state and it belongs on this side of the bridge: remember how many rows the list held when the handler last ran, and refuse to run again until that number changes. A fetch that appends rows unlocks the next fire; a fetch that returns nothing (the feed is exhausted, or it failed) leaves the guard closed, which is exactly right — scrolling at the bottom of a list that just came back empty should not re-ask forever. A caller that wants the retry offers a button; that is what comps.LoadMore's error arm has always been for.

Doing it in Go also means the four renderers each get to be as naive as their platform makes convenient, and none of them has to agree with the others about what "once" means.

The row count is read at \*dispatch\* time off the node this prop was applied to, which by then holds the children the pass rendered. (Behaviour props run before children in containerNode, so there is nothing to count yet when this closure is built — only when it is called.)

##### Where in the argument list it goes

Anywhere. containerNode registers behavior props in argument order but renders children only after that loop has finished, so a List's own callback IDs always precede its rows' — and these two spellings produce the same ID for the same list, at any row count:

	core.List(core.OnEndReached(pager.LoadNext), rows...)
	core.List(append(rows, core.OnEndReached(pager.LoadNext))...)

That is worth saying out loud rather than leaving to be derived, because the guard below is keyed by the ID and a reader who works out what the key is made of is right to wonder whether a page that lengthens the list moves it. Within one List it cannot; TestOnEndReachedIDIsIndependentOfArgumentOrder holds the contract still.

What \*does\* move it is anything earlier in the same pass that registers a varying number of callbacks: a sibling above the List whose own children grow with the page, or a row helper that renders on the spot with view.Render(ctx) rather than returning a View for the List to render. Then the ID slides with the data, each page starts its guard from scratch under a key something else held on the previous pass, and the double-load this prop exists to prevent comes back. Same family as the edge below, and the same identity-keyed IDs close both.

##### The guard's one sharp edge

State is keyed by callback ID, and callback IDs are positional: the Nth void handler registered in a pass is always "cb\_N" (see callbackRegistry). Two different Lists in two different screens can therefore inherit the same ID across a navigation, and the second one's first end-reached is swallowed if it happens to hold exactly as many rows as the first did when the first last fired. That is the same stale-ID window the registry itself documents, and it closes when identity-keyed IDs land; nothing here can close it earlier, because the two lists are indistinguishable from this side.

<small>[core/list.go:154](https://github.com/rohanthewiz/grmob/blob/master/core/list.go#L154)</small>

#### func OnFocus

```go
func OnFocus(handler func()) BehaviorProp
```

OnFocus fires when the node becomes the input focus — a text field the user has tapped into, with the software keyboard on its way up.

OnBlur fires when that focus leaves. The pair is deliberately \*not\* a single bool-carrying handler: the two edges are almost never handled together (a form reveals errors on blur and does nothing on focus; a search box does the opposite), and one prop per edge lets a node carry only the edge it cares about instead of registering a callback to ignore half its calls.

Both ride the void-callback channel, exactly like OnClick — the edge itself is the whole payload, and "which node" is already answered by which callback ID the platform dispatches.

Focus is a leaf concern in practice: the renderers wire these on the text input node types, which are the only things a mobile platform gives focus to. The props are attachable to any node because BehaviorProp is uniform, but a Row carrying OnFocus will simply never hear from the native renderers.

Ordering note: the framework guarantees the edges are dispatched in the order they happened, but \*not\* that a blur on the field being left arrives before the focus on the field being entered — that ordering is the platform's, and Android and iOS do not agree on it. Handlers must therefore be independent: read the field the callback belongs to, never "the field that is focused now".

<small>[core/behavioral_props.go:160](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L160)</small>

#### func OnLongPress

```go
func OnLongPress(handler func()) BehaviorProp
```

OnLongPress fires after 500ms of held press without the finger lifting — the default on all three targets (UILongPressGestureRecognizer, Android's ViewConfiguration, and the DOM runtime's own timer).

A node may carry both OnClick and OnLongPress, and one gesture produces exactly one handler call: Compose's combinedClickable splits them natively, while SwiftUI and the DOM each suppress the tap that follows a fired long press. Which handler runs is decided by how long the press was held, never by both running.

Wired on all three targets, containers and leaves alike, including Button — which needs its own wiring on both natives, since a Button draws its own control rather than going through the generic gesture path.

<small>[core/behavioral_props.go:56](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L56)</small>

#### func OnMapTap

```go
func OnMapTap(fn func(lat, lng float64)) BehaviorProp
```

OnMapTap reports where the user tapped on the map, in degrees — the prop a "choose a place" screen is built on.

It does not fire for a tap that hit a marker: that is OnMarkerTap's event, and a host that sent both would make every marker tap also drop a pin. Each host suppresses it the way its own map does — an annotation's hit test runs first on all three.

Like OnRegionChange it crosses as text, "lat,lng", and an unparseable payload is dropped rather than delivered as 0,0.

<small>[core/mapview.go:286](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L286)</small>

#### func OnMarkerTap

```go
func OnMarkerTap(fn func(id string)) BehaviorProp
```

OnMarkerTap reports the id of the marker the user tapped — the id given to core.Marker, unchanged.

An id rather than the marker's coordinates, because the id is what the app has an index of. Two markers can share a position (a building with two tenants, a rounded coordinate) and no app wants to identify a row by comparing floats.

<small>[core/mapview.go:264](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L264)</small>

#### func OnRegionChange

```go
func OnRegionChange(fn func(Region)) BehaviorProp
```

OnRegionChange reports where the user moved the map to, after they stop moving it.

"After" is the contract and the hosts enforce it, because a pan is a stream: a finger dragging across a map generates a region per frame, and each one that crossed the bridge would be a full Go render pass. Every host therefore throttles — it reports on the gesture's end, and at a bounded rate during a sustained one — which is the same arrangement the sensors have, for the same reason, and is checked in the same places.

The region arrives through the text callback channel as "lat,lng,zoom", which is how Slider's float crosses and why no new bridge channel was added for this node. A payload that does not parse is dropped rather than delivered as zeros: 0,0 is a real place, and an app that centred on it because of a formatting bug in one host would be looking at the Gulf of Guinea with no error anywhere.

<small>[core/mapview.go:241](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L241)</small>

#### func OnRichSelectionChange

```go
func OnRichSelectionChange(handler func(RichSelection)) BehaviorProp
```

OnRichSelectionChange reports a rich-text editor's caret and the formatting active at it.

A separate builder from OnSelectionChange rather than an overload, because the two carry different things and the difference is the point: a code editor's selection is two offsets, and a rich editor's is the state a toolbar has to draw. They share the prop name on the wire ("onSelectionChange") and the text channel; what differs is the payload each node type sends and the parse core does before app code sees it.

<small>[core/richtext.go:237](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L237)</small>

#### func OnSelectionChange

```go
func OnSelectionChange(handler func(start, end int)) BehaviorProp
```

OnSelectionChange reports the editing surface's selection as it moves.

The handler receives byte offsets into the UTF-8 value, half-open as every range in Go is: start == end is a caret with nothing selected.

	core.CodeEditor(src, onChange, rows,
	    core.OnSelectionChange(func(start, end int) { status.Set(start, end) }),
	)

It rides the text channel carrying "start:end" rather than needing a fifth bridge channel; see this file's doc. A payload that is not two integers is dropped rather than delivered as a zero range.

Applied to a node that is not an editing surface it is inert, like every other behavior prop on a node type that does not read it.

<small>[core/editor.go:253](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L253)</small>

#### func OnSliderChangeEnd

```go
func OnSliderChangeEnd(fn func(float64)) BehaviorProp
```

OnSliderChangeEnd fires once when the drag ends, with the final value — see Slider for why a seek bar wants this rather than onChange.

<small>[core/slider.go:67](https://github.com/rohanthewiz/grmob/blob/master/core/slider.go#L67)</small>

#### func OnTouch

```go
func OnTouch(handler func()) BehaviorProp
```

OnTouch fires the moment a finger (or pen, or mouse button) goes down on the node, before the press has resolved into a tap or a long press. Use it for immediate feedback — a sound, a highlight — not for the action itself: a press that slides off the node still fired this.

Web only today. The DOM runtime maps it to pointerdown; neither native renderer reads the prop, so a node carrying it on iOS or Android simply never hears from it. (It used to be worse: the DOM's event-name fallback derived "touch", which is not a DOM event either, so the prop attached a listener nothing could ever fire on any target.)

<small>[core/behavioral_props.go:39](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L39)</small>

#### func Placeholder

```go
func Placeholder(text string) BehaviorProp
```

Placeholder is the prompt an empty editing surface shows.

core.Input and core.TextArea take theirs positionally, because they had one before the mixed argument list existed; the editors take it as a prop, which is the shape every option added since uses.

<small>[core/editor.go:292](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L292)</small>

#### func ReadOnly

```go
func ReadOnly() BehaviorProp
```

ReadOnly makes an editing surface show a caret and allow selection while refusing every edit.

It is deliberately not core.Disabled. A disabled control is inert and greyed and is skipped by assistive technology's traversal; a read-only code block is \*content\* — the user is meant to read it, select it and copy out of it — and on every platform that is a different state with a different look. The two are not interchangeable and a widget that wants the greyed-out reading can still ask for it.

<small>[core/editor.go:278](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L278)</small>

#### func ShowUserLocation

```go
func ShowUserLocation() BehaviorProp
```

ShowUserLocation asks the host's map to draw the user's position — the blue dot, with the platform's own styling and its own accuracy halo.

It is the host's location, not core.Location's. Every map SDK has this built in, reads the OS permission itself, and tells Go nothing about where anybody is. A screen that needs the coordinates wants hooks.UseLocation, which is a separate feature that happens to need the same permission — see core.Location's "What this is not".

The dot needs that permission. On a platform where it has not been granted, every one of these hosts draws the map and no dot, with nothing reported: a map is still a map. An app that wants to explain the absence has to check the permission itself, which is what permission.Location is for.

<small>[core/mapview.go:216](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L216)</small>

#### func SliderStep

```go
func SliderStep(step float64) BehaviorProp
```

SliderStep snaps the thumb to multiples of step from min. 0 (the default) is continuous.

<small>[core/slider.go:81](https://github.com/rohanthewiz/grmob/blob/master/core/slider.go#L81)</small>

#### func TabSize

```go
func TabSize(spaces int) BehaviorProp
```

TabSize sets how many spaces one indent is worth — what the Tab key inserts, and what EditIndent adds and EditOutdent removes.

Zero means a literal tab character instead of spaces, which is what Go source wants. A negative size is clamped to zero rather than refused: the honest reading of "minus two spaces" is "no spaces", and a render pass is not a place to panic over an argument.

<small>[core/codeeditor.go:150](https://github.com/rohanthewiz/grmob/blob/master/core/codeeditor.go#L150)</small>

### type CameraNode

```go
type CameraNode struct {
	OnCapture func(string)
	OnError   func(string)
	Active    bool
	Flash     bool
	Facing    string
	Overlay   View
	Style     Style
}
```

<small>[core/camera.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L7)</small>

### type CameraProp

```go
type CameraProp interface {
	Apply(*CameraNode)
}
```

<small>[core/camera.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L3)</small>

#### func OnCapture

```go
func OnCapture(fn func(string)) CameraProp
```

<small>[core/camera.go:78](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L78)</small>

#### func OnError

```go
func OnError(fn func(string)) CameraProp
```

<small>[core/camera.go:72](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L72)</small>

#### func SetFacing

```go
func SetFacing(facing string) CameraProp
```

<small>[core/camera.go:66](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L66)</small>

#### func WithFlash

```go
func WithFlash(enabled bool) CameraProp
```

<small>[core/camera.go:60](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L60)</small>

#### func WithOverlay

```go
func WithOverlay(view View) CameraProp
```

<small>[core/camera.go:84](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L84)</small>

#### func WithStyle

```go
func WithStyle(style Style) CameraProp
```

<small>[core/camera.go:90](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L90)</small>

### type ColorPalette

```go
type ColorPalette struct {
	Primary       string
	Secondary     string
	Background    string
	Surface       string
	TextPrimary   string
	TextSecondary string
	Error         string

	// Border is the stroke/hairline role: rules between list rows, card
	// outlines, the ring around a compass rose. It is deliberately distinct
	// from Surface. Surface is a *fill* — the two are near neighbors on a
	// light theme, so a Surface-colored hairline on a Surface-colored panel
	// is invisible.
	//
	// # It is a divider, not a control boundary
	//
	// This role used to name input borders too, and no longer does. A rule
	// *between* things is decoration — nothing about the page becomes
	// unusable if a reader cannot make it out — and every bundled theme spends
	// a very pale hex on it accordingly: #E5E5EA measures 1.26:1 against
	// white and #E0E0E0 1.32:1 (AmberTheme spends the second of those). The
	// edge that says *this rectangle is a field
	// you can type in* is the opposite case: it is the only thing identifying
	// a control, which WCAG 1.4.11 (Non-text Contrast) puts a 3:1 floor
	// under. One hex cannot be both, for the same reason a role's fill tone
	// cannot also be its ink — see the on-light tones below.
	//
	// So this role keeps the dividers and ControlBorder below carries the
	// boundary. That split used to have no second field in it — the frames
	// lived in Components.Input and Components.TextArea alone, and the note
	// here said no palette role was needed because nothing outside those two
	// component defaults spent one. comps.Chip is what made that false:
	// a quiet chip's hairline is not a rule *between* things, it is the only
	// edge a filter control has, and it was drawing it out of this role.
	//
	// Read via BorderColor.
	Border string

	// ControlBorder is the boundary tone: the edge that says *this rectangle
	// is a control*. A text field's frame, a quiet chip's ring — anything a
	// reader has to make out before they can know there is something here to
	// operate.
	//
	// It is Border's other half and exists because one hex cannot do both
	// jobs, which is the same shape the on-light tones' argument has one
	// property over. A divider is decoration and a pale one is a legitimate
	// choice; a boundary that identifies a control carries WCAG 1.4.11's 3:1
	// floor. Every bundled theme spends 1.26:1 or 1.32:1 on Border
	// accordingly, so a control drawn in it is close to invisible *as a
	// control*:
	//
	//	                    Default            Material           Amber
	//	Border              #E5E5EA  1.26:1    #E0E0E0  1.32:1    #E0E0E0  1.32:1
	//	ControlBorder       #89898E  3.48:1    #757575  4.61:1    #8D6E63  4.62:1
	//
	// (against each theme's own white Background; see the themes for the
	// second backdrop each measures against.)
	//
	// # A control has more than one backdrop, and the census is the list
	//
	// The page is only the first of them. A boundary drawn on a Surface
	// panel, a Card or a field's own fill is measured against that fill, and
	// the tone is one hex for all of them — so "ControlBorder clears 3:1" is
	// not a property of the tone, it is a property of a *pair*.
	//
	// Every pair a bundled theme can produce is enumerated and measured by
	// TestEveryControlBoundaryPairIsAccountedFor in comps/variant_test.go
	// (the arithmetic lives there, beside the on-light census, for the reason
	// that one gives). Every pair clears, and the census's
	// knownBoundaryShortfalls table — the place a defended shortfall would be
	// recorded as a fact rather than as prose — is empty.
	//
	// It was not always. DefaultTheme's tone was iOS systemGray #8E8E93,
	// which is 3.26:1 on that theme's page and 2.92:1 on its Surface, and the
	// quiet chip's ring is drawn on Surface. The shortfall was argued and
	// exempted for two sessions — the edge that identifies the pill is the
	// outer one, so the inner pair is a boundary between two parts of one
	// control — and the argument was sound. What retired it is that the
	// census made the alternative cheap: with every pair measured, "does this
	// candidate hex clear all four backdrops" is one test run, and #89898E
	// clears them at 3.12:1 and above while differing from systemGray by
	// 5/255 per channel. Defending a shortfall costs a paragraph that every
	// future boundary has to be read against; this cost a retint.
	//
	// # Why this arrived a session after the frames did
	//
	// Components.Input and Components.TextArea state their frame as a literal
	// and still do — a Style is a value, so a component default cannot call a
	// resolver — and while those two were the only spenders, a role would have
	// been a name with one call site. The second spender is what a role is
	// for. The two must not drift, so a bundled theme's Input and TextArea
	// frames are pinned to its ControlBorder by TestBundledFieldFramesAreThe
	// ControlBorderRole rather than by the type system.
	//
	// A widget that wants to look like a text field still reads the Input base
	// itself (comps.DatePicker does), because it wants the radius and the
	// fill too. This role is for a widget that wants only the edge.
	//
	// Read via ControlBorderColor.
	ControlBorder string

	// Success and Warning complete the status triad with the existing Error,
	// for the "saved" / "expiring" / "failed" progression a status chip,
	// banner or badge variant needs.
	//
	// Success is *not* Secondary, even where a theme happens to give both the
	// same green (DefaultTheme does). Secondary is a brand slot — a theme is
	// free to make it teal or magenta, as MaterialTheme does — while Success
	// carries meaning, and a magenta "saved" badge is a bug.
	//
	// Read via SuccessColor and WarningColor.
	Success string
	Warning string

	// The on-light tones: the same four roles again, dark enough to be read
	// as *ink* on a light surface.
	//
	// A palette role is one hex, and one hex cannot do both jobs a role is
	// asked to do. Spent as a fill with a chosen ink over it, a mid-tone
	// works — comps.Variant.Ink picks the more legible of the theme's
	// two inks and a filled Badge or Button clears WCAG AA on every bundled
	// theme. Spent as ink *itself* — an outlined button's label and rule, a
	// loud chip's outline, a banner's leading glyph — the backdrop is
	// whatever the widget was placed on, and a mid-tone loses. The numbers
	// against each bundled theme's own white Background were:
	//
	//	            Default   Material     (WCAG AA for body text is 4.5:1)
	//	primary      4.02:1    7.63:1
	//	success      2.22:1    5.13:1
	//	warning      2.20:1    3.08:1
	//	error        3.55:1    7.33:1
	//
	// Five of those eight fail. This is the fix comps.Button and
	// comps.Chip both name in their own docs and could not make: the
	// widgets have the number and not the authority. Darkening a role until
	// it passes would repaint a hex the theme author chose, so the second
	// tone is the theme's to declare.
	//
	// # One of those five was later fixed at the role, and why that is not a
	// contradiction
	//
	// DefaultTheme's primary is no longer 4.02:1. The role itself moved to
	// Apple's accessible blue #0040DD (7.56:1), because that role is not read
	// as ink only — it is also the *fill* under every filled Button and under
	// Calendar's selected day, and the white the theme declares over that fill
	// was the thing failing. A second tone cannot reach a declared pairing;
	// only the role can. So the table above is the state these fields were
	// introduced to answer for, and four of the eight still fail today.
	//
	// Which gives the rule from the other side. Darken the *role* when the
	// role is spent as a fill and the ink declared over it is the problem;
	// add a *tone* when the role is a perfectly good fill and only fails as
	// ink. Success and Warning are the second case — DefaultTheme's green and
	// orange carry black text at ~9.5:1 — and Primary turned out to be the
	// first.
	//
	// # Reading them
	//
	// Through the resolver methods below, or through OnLight when a widget
	// holds a colour rather than a role. An unset tone falls back to the role
	// itself, which is exactly what every widget spent before these existed,
	// so a theme that predates them renders as it always did rather than
	// rendering nothing. That is a softer fallback than Border/Success/
	// Warning get, and deliberately: those degrade to a *visible* default
	// because an empty color is no color, while an absent on-light tone has a
	// perfectly good — merely paler — answer sitting beside it.
	//
	// # "Light" is the theme's Background, not a global assumption
	//
	// The name says which surface the tone is legible on, and for both
	// bundled themes that surface is #FFFFFF. A dark theme's role colours are
	// usually already legible on its dark background, so it leaves these
	// empty and the fallback returns the role — which is the right answer,
	// and the reason these are four extra fields rather than a second
	// palette every theme has to fill in twice.
	//
	// # Overriding a role means releasing its tone
	//
	// These are *measurements*, taken against the role colour beside them. An
	// app that brands a theme by copying DefaultTheme and assigning
	// Colors.Primary therefore inherits a tone measured against a colour that
	// is no longer there:
	//
	//	theme := *core.DefaultTheme
	//	theme.Colors.Primary = siteColor   // and PrimaryOnLight is still #0040DD
	//
	// The result is a half-branded app, and a quiet one — filled controls take
	// the new colour (they read Primary, or the Button base) while every
	// unfilled one keeps the default's blue, because Outlined, Ghost, Chip and
	// the calendar's month arrows all spend the *tone*. The first downstream
	// app to brand a theme shipped exactly that.
	//
	// So an override sets the pair or clears it:
	//
	//	theme.Colors.Primary = siteColor
	//	theme.Colors.PrimaryOnLight = ""   // no measurement, use the role
	//
	// Clearing is the honest default. The fallback then returns siteColor,
	// which is the same treatment every widget gave before these fields
	// existed; writing siteColor into the tone renders identically but claims
	// a contrast check nobody ran. Either way, what is not available is
	// leaving the old number in place.
	PrimaryOnLight string
	SuccessOnLight string
	WarningOnLight string
	ErrorOnLight   string
}
```

ColorPalette is a theme's semantic color roles. Widgets name the \*role\* they want, never a literal, so one theme swap restyles the whole tree.

The seven original roles (Primary through Error) are set by every theme that exists, bundled or user-written, because they predate any of them. The three added later — Border, Success, Warning — cannot make that assumption: a theme written before they existed leaves them empty, and an empty color is not "the default", it is \*no color\*, which renders as an invisible rule or a transparent chip. Read those three through their resolver methods (BorderColor, SuccessColor, WarningColor) rather than off the field, and a pre-existing theme degrades to the documented fallback instead of to nothing. The originals need no such treatment and deliberately have no resolvers.

<small>[core/theme.go:25](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L25)</small>

#### func (ColorPalette) BorderColor

```go
func (c ColorPalette) BorderColor() string
```

BorderColor resolves the Border role, falling back to FallbackBorder when the theme predates it.

Note this is a \*method on the palette\* and is unrelated to the core.BorderColor style prop, which sets a node's stroke color:

	core.BorderColor(ctx.Theme().Colors.BorderColor())

<small>[core/theme.go:260](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L260)</small>

#### func (ColorPalette) ControlBorderColor

```go
func (c ColorPalette) ControlBorderColor() string
```

ControlBorderColor resolves the ControlBorder role, falling back to FallbackControlBorder when the theme predates it.

Note it does \*not\* fall back to BorderColor(). The two roles are near neighbours in the struct and opposites in intent — see the field docs — and a theme that has one and not the other is a theme that has only the divider, which is precisely the value this must not return.

<small>[core/theme.go:274](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L274)</small>

#### func (ColorPalette) ErrorOnLightColor

```go
func (c ColorPalette) ErrorOnLightColor() string
```

<small>[core/theme.go:322](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L322)</small>

#### func (ColorPalette) OnLight

```go
func (c ColorPalette) OnLight(color string) string
```

OnLight returns the ink-weight tone paired with color, when color is one of this palette's four toned roles, and color itself otherwise.

The four resolvers above answer for a widget that knows which \*role\* it is spending. This answers for one that knows only a \*colour\*, which is the commoner case than it sounds: comps.Chip's accent is read off the theme's Button base rather than off Colors.Primary, precisely so that a theme whose buttons are not primary-coloured still gets its own look, and a widget in that position has a hex and no name for it.

So it is a reverse lookup, and it is honest about being one. A colour that is not one of the four roles comes back unchanged, which is the same fallback an unset tone gets and the same pixels every widget painted before these fields existed.

##### First match wins, and two roles may share a hex

DefaultTheme paints Secondary and Success the same green. Only the four toned roles are consulted here, and no bundled theme repeats a colour among those four — but a theme could, and the arms are ordered Primary, Error, Success, Warning so that the answer is at least stable rather than map-order dependent. A palette that spends one hex on two of these roles is telling the widget the two are indistinguishable, which is a statement about the palette rather than a bug here.

Comparison is case-insensitive because "#007AFF" and "#007aff" are the same colour to every renderer, and a theme is hand-written.

##### Most of these arms are identities under most bundled themes

A role whose colour is already ink-weight sets its tone equal to itself, deliberately (see each palette for why "this role needs no second tone" is written out rather than left blank). Under DefaultTheme only Primary is in that position; under MaterialTheme, Primary, Success and Error are; under AmberTheme, Success and Error. An arm that is an identity for \*every\* bundled theme could be deleted with no bundled pixel moving.

Primary's used to be exactly that, and AmberTheme is what put it back: amber 700 is a fill that cannot be ink (2.04:1 on white), so its role and its tone are genuinely different colours and deleting this arm repaints every outlined button and loud chip in that theme.

Which arms have a bundled witness and which rest on a test fixture is recorded and checked per role by TestEveryPaletteRuleStillHasAWitness in comps/palette\_witness\_test.go, so a retint that leaves an arm with no evidence anywhere is reported rather than merely true.

<small>[core/theme.go:375](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L375)</small>

#### func (ColorPalette) PrimaryOnLightColor

```go
func (c ColorPalette) PrimaryOnLightColor() string
```

The four on-light resolvers. Each falls back to its own role rather than to a constant — see the field docs for why this fallback is softer than Border's, and note that each defers to the role's \*resolver\* where it has one, so a theme missing both halves of a role still lands somewhere visible.

<small>[core/theme.go:301](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L301)</small>

#### func (ColorPalette) SuccessColor

```go
func (c ColorPalette) SuccessColor() string
```

SuccessColor resolves the Success role, falling back to FallbackSuccess.

<small>[core/theme.go:282](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L282)</small>

#### func (ColorPalette) SuccessOnLightColor

```go
func (c ColorPalette) SuccessOnLightColor() string
```

<small>[core/theme.go:308](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L308)</small>

#### func (ColorPalette) WarningColor

```go
func (c ColorPalette) WarningColor() string
```

WarningColor resolves the Warning role, falling back to FallbackWarning.

<small>[core/theme.go:290](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L290)</small>

#### func (ColorPalette) WarningOnLightColor

```go
func (c ColorPalette) WarningOnLightColor() string
```

<small>[core/theme.go:315](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L315)</small>

### type ComponentDefaults

```go
type ComponentDefaults struct {
	Button   Style `notbackdrop:"a control's own fill, not a surface: Colors.Primary. A bordered control is never drawn on top of a filled button — an outline Button draws its own edge over whatever is behind it, which is the page or a panel, and both of those are already measured. Excluded because the pair is unreachable, not because it is close"`
	Card     Style `backdrop:"a panel: a Card is a container, so anything a screen puts inside one is drawn on this fill. comps.FormField's Input inside a comps.Card is the commonest screen this framework builds, and its frame is a control boundary against exactly this colour"`
	Input    Style `backdrop:"a field's own interior, enclosed by its own frame: the pair here is a boundary against the fill it encircles rather than one control on top of another. Reachable by construction, not by composition — every Input that states a BorderColor builds it, and there is no arrangement of widgets that avoids it"`
	Column   Style `backdrop:"a layout container. It states no fill in any bundled theme, so it contributes no pair today — that is a fact about the themes and not about the geometry. A theme that fills its Column has made it a page region, and every control laid out in one is then drawn on it"`
	Row      Style `backdrop:"a layout container, on the other axis and for the same reason as Column. comps.GroupHeader's band is a filled Row with a bordered control in it the moment a caller styles one, which is the shape that makes this a real pair rather than a hypothetical"`
	Camera   Style `notbackdrop:"a viewfinder: its fill is black in every theme because it is what shows for the frame before the first camera frame arrives, and nothing draws a control boundary on top of a preview. Excluded by name rather than by a lightness test, because a rule that skipped dark fills would also skip a dark theme's page"`
	CheckBox Style `backdrop:"the box's own interior, enclosed by its own boundary — the same by-construction pair as Input. Two of the three bundled themes state a fill here and the third does not, so the pair exists in some palettes and not others, which is what a derived census handles and a hand-written list does not"`
	TextArea Style `backdrop:"a field's own interior, as Input, one tag over. The two are separate fields because a theme may want a taller field to read differently, and they are separate rows in the census for the same reason"`
}
```

ComponentDefaults is the per-component base Style a theme supplies. Every widget merges the caller's props over the field named for it.

#### The notbackdrop tag

A field's Background is, by default, a fill a bordered control can be drawn on — and core.ColorPalette.ControlBorder has WCAG 1.4.11's 3:1 floor against every such fill. internal/palette derives that list by reflecting over this struct, precisely so that adding a field (a Sheet, a Popover) adds a backdrop with nobody having to remember, and comps/variant\_test.go measures the pair.

Every field carries exactly one of two tags, and both are claims about the \*geometry\* of the framework rather than about any number:

	backdrop      what draws a control boundary on this fill
	notbackdrop   why nothing ever does

The value is the argument in both cases and it is required. An exclusion with no reason is a census defeating itself — every excluded field below fails the 3:1 floor in all three bundled themes and would otherwise look like a failure somebody made go away — and an inclusion with no reason is the thing the pair was introduced to end.

#### Why both, and not just the exclusion

The exclusion came first and everything else was measured because it was left over, which had two costs. A pair nothing builds was measured beside a pair three widgets build, so a shortfall in either read the same way to whoever had to fix it — and, worse, a \`notbackdrop\` tag could be \*deleted\* with no consequence but a pair quietly joining the census. Drop Camera's and the added pair clears 6:1, so the run stays green while a geometry claim has been thrown away.

With both tags mandatory a field carrying neither is a hard failure, so deleting either one is now the same kind of event as deleting a field's name. palette.Untagged is that reading and TestEveryComponentFillIsClassified is where it fails.

They live here rather than in a list one package over because this is where a theme author works. "No widget puts a bordered control on this surface" is knowable at the field and is not knowable from a name in internal/palette, and a name in a list is invisible in the diff that adds a field beside it.

palette.IsABackdrop and palette.NotABackdrop read these tags and are their only readers; the reachability claim travels on into the census, which prints it when a pair falls short.

<small>[core/theme.go:447](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L447)</small>

### type ComponentFunc

```go
type ComponentFunc func(ctx *Context) *Node
```

<small>[core/view.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/view.go#L7)</small>

#### func (ComponentFunc) Render

```go
func (f ComponentFunc) Render(ctx *Context) *Node
```

<small>[core/view.go:9](https://github.com/rohanthewiz/grmob/blob/master/core/view.go#L9)</small>

### type CompositeWalk

```go
type CompositeWalk int
```

CompositeWalk is what an outer composite's member walk does at a nested composite: one of two answers, or the value that says the question does not apply.

#### Why the third value exists

This started as CompositeWalkStopsAt alone, returning a bool. Two of its three arms returned true — a container with no keyboard, and a toolbar whose walk genuinely does stop — and the two are different facts: one is "the arrows step over this node", the other is "there are no arrows". The distinction was real in the code, with a paragraph on each arm saying so, and invisible from outside: a caller holding the \`true\` could not tell which it had, and the safe reading of a non-composite ("stops") is a confident statement about a walk that does not exist.

Making it a value rather than a doc note is the same move CompositeMemberRole made one function up when its two empty answers became (member, composite): the fact is put where the compiler and the caller can both see it, instead of in a sentence asking the caller to have already checked something.

<small>[core/role.go:868](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L868)</small>

```go
const (
	// CompositeWalkNotApplicable: the outer role has no keyboard, so it has no
	// member walk and neither of the other two values is true of it.
	//
	// Zero so that a CompositeWalk nobody assigned reads as "no claim" rather
	// than as one of the answers — the same trade every unset field in
	// core.Style makes, and it is safe here for the reason it is not safe for
	// flex-shrink (see ShrinkNone): the useful default really is the empty one.
	CompositeWalkNotApplicable CompositeWalk = iota

	// CompositeWalkStops: the outer widget's arrows step over the inner one
	// whole. Its members are still its own; what it loses is any element of
	// its member role buried inside the nested widget.
	CompositeWalkStops

	// CompositeWalkDescends: the outer widget's walk carries on through the
	// inner one, so an element of the outer's member role inside the nested
	// widget's subtree is pooled into the outer's rotation and the arrows can
	// land inside a widget they are not steering.
	CompositeWalkDescends
)
```

#### func CompositeWalkAt

```go
func CompositeWalkAt(outer, inner Role) CompositeWalk
```

CompositeWalkAt reports what an outer composite's member walk does when it meets a nested composite: stop there, descend through it, or — the case that is not an answer — nothing, because the outer role has no walk.

##### Why core states a rule about a walk in another language

AuditTree is the reader. ConcernNestedComposite is a finding about a pair of containers, and \*what goes wrong\* is not the same for every pair — so a report that did not know this rule could only describe one of the two outcomes, and would describe the other one wrongly. That is the same argument KeyboardComposites makes one level up: the audit is the pass whose job is telling a Go author what their finished tree amounts to, and it cannot get the answer from a runtime written in JavaScript.

##### The rule

	outer has no keyboard          not applicable. There is no member walk, so
	(a heading, a Box)             neither answer is true of it — see
	                               CompositeWalkNotApplicable for why that is a
	                               value rather than a `true` chosen for safety.

	outer names no member role     stop. Nothing says whose a plain button is,
	(toolbar)                      so a control inside a nested composite
	                               belongs to the nested one.

	inner has the same members     stop. Two listboxes pooling their options
	                               would let one widget's arrows walk out into
	                               the other's rows.

	otherwise                      descend. An option below a tablist is still
	                               the listbox's option — the roles say whose
	                               it is, and stopping would lose a member the
	                               vocabulary has already assigned.

Member roles are unique per container, so the middle case is the same set of pairs as \`inner == outer\`; it is written against the member role because that is the question the runtime's walk actually asks (\`compositeMemberRole(child) === memberRole\`), and a fourth pattern sharing a member role with a third would land here rather than in a surprise.

##### What descending costs, and why it is still right

A descending pair is still two tab stops — both containers keep a roving tabindex either way, which is the part of the finding that never varies. What differs is the reach: where a stopping pair's outer arrows step over the inner widget whole, a descending pair's outer arrows can land \*inside\* it, on any element of the outer's member role buried in the inner's subtree. Neither is what ARIA describes for nested composites, and the framework's refusal to guess is documented at ConcernNestedComposite.

<small>[core/role.go:809](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L809)</small>

#### func (CompositeWalk) String

```go
func (w CompositeWalk) String() string
```

String names the value for a message. The three spellings are the words the audit's finding and this file's docs already use, so a report built from a %v and a report written by hand read the same.

<small>[core/role.go:895](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L895)</small>

### type Concern

```go
type Concern struct {
	Kind   string
	Detail string
	Count  int
}
```

Concern is one detected issue. Kind is one of the Concern\* constants; Detail is human-readable specifics; Count is how many times this exact (Kind, Detail) pair fired — checks run every pass, so a persistent bug increments its count rather than flooding the collector with duplicates.

<small>[core/debug.go:92](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L92)</small>

#### func Concerns

```go
func Concerns() []Concern
```

Concerns returns a snapshot of all recorded concerns, sorted by kind then detail so test assertions and dumps are deterministic.

<small>[core/debug.go:143](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L143)</small>

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

### type Context

```go
type Context struct {
	Cursor int
	// contains filtered or unexported fields
}
```

<small>[core/context.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L7)</small>

#### func NewContext

```go
func NewContext() *Context
```

<small>[core/context.go:130](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L130)</small>

#### func UseChildContext

```go
func UseChildContext(ctx *Context) *Context
```

<small>[core/context.go:161](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L161)</small>

#### func (*Context) BeginRenderPass

```go
func (ctx *Context) BeginRenderPass()
```

BeginRenderPass starts a callback ID pass for this context tree; see callbackRegistry.beginPass for the stability contract.

<small>[core/event.go:316](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L316)</small>

#### func (*Context) ClearDirty

```go
func (ctx *Context) ClearDirty()
```

<small>[core/context.go:115](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L115)</small>

#### func (*Context) Close

```go
func (ctx *Context) Close()
```

Close stops every background resource registered on this context tree since the last Close (see the drain semantics on cleanupRegistry). The tree itself remains renderable afterwards; a subsequent render pass simply re-registers whatever resources it still needs.

<small>[core/cleanup.go:120](https://github.com/rohanthewiz/grmob/blob/master/core/cleanup.go#L120)</small>

#### func (*Context) Config

```go
func (ctx *Context) Config() *AppConfig
```

<small>[core/context.go:198](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L198)</small>

#### func (*Context) EndRenderPass

```go
func (ctx *Context) EndRenderPass()
```

EndRenderPass closes a render pass for debug purposes: in debug mode it walks the context tree and flags any context whose hook usage this pass is inconsistent. Hosts that drive passes through render.Manager get this for free (the manager calls it after each pass); hand-rolled pass loops should call it after rendering, paired with BeginRenderPass/Reset before.

The pairing with the rest of the pass boundary:

	BeginRenderPass()  — callback ID counters restart
	Reset()            — hook cursors restart
	root.Render(ctx)   — components consume slots, cursor advances
	EndRenderPass()    — cursors audited against slots + previous pass  ← here

A no-op (single atomic load) when debug mode is off.

<small>[core/debug.go:198](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L198)</small>

#### func (*Context) IsDirty

```go
func (ctx *Context) IsDirty() bool
```

IsDirty reports whether the tree has changes no pass has consumed yet. It answers for the whole app, not for the context it is called on.

<small>[core/context.go:109](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L109)</small>

#### func (*Context) MarkDirty

```go
func (ctx *Context) MarkDirty()
```

MarkDirty records that the tree needs re-rendering, without notifying anyone. Callers that want a render to actually happen want RequestRender, which does this and nudges the render manager; MarkDirty alone is for paths where a pass is already guaranteed to follow.

<small>[core/context.go:101](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L101)</small>

#### func (*Context) NewChildContext

```go
func (ctx *Context) NewChildContext() *Context
```

<small>[core/context.go:144](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L144)</small>

#### func (*Context) OnClose

```go
func (ctx *Context) OnClose(fn func())
```

OnClose registers fn to run when this context tree is closed. Hooks use it to hand ownership of their background resources to whoever drives the app's lifecycle (normally render.Manager, whose Close closes its context).

"This context tree" is the registry the context carries, which for most contexts is the app-wide one. A context inside a navigation stack frame carries that frame's registry instead, so its resources also stop when the frame leaves the stack — earlier than the app's own shutdown.

<small>[core/cleanup.go:109](https://github.com/rohanthewiz/grmob/blob/master/core/cleanup.go#L109)</small>

#### func (*Context) OnStateChange

```go
func (ctx *Context) OnStateChange(fn func())
```

OnStateChange registers fn to run whenever state anywhere in this context tree is written (State.Set, or anything else calling RequestRender). Only one handler is held: a render driver like render.Manager owns re-rendering for the whole app, so later registrations replace earlier ones rather than fanning out duplicate render passes.

fn is invoked on a fresh goroutine per notification (see TriggerRender), so it must be safe to call concurrently and should be cheap — the intended pattern is a non-blocking nudge into a coalescing channel, not a render.

<small>[core/render_manager.go:52](https://github.com/rohanthewiz/grmob/blob/master/core/render_manager.go#L52)</small>

#### func (*Context) PurgeUnusedCallbacks

```go
func (ctx *Context) PurgeUnusedCallbacks()
```

PurgeUnusedCallbacks drops handlers not re-registered in the current pass; see callbackRegistry.purge.

OnEndReached's debounce ledger is trimmed in the same breath and against the registry's own survivors, so the two can never disagree about which lists are still on screen — a guard outliving its handler would silently suppress the first page fetch of whatever list next inherits the ID.

<small>[core/event.go:327](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L327)</small>

#### func (*Context) ReceiveEventPayload

```go
func (ctx *Context) ReceiveEventPayload(payload map[string]any)
```

ReceiveEventPayload dispatches a loosely typed event envelope ({"callback": id, "value": ...}) by sniffing the value's type — the shape the WASM host sends. Typed hosts should call the Trigger\* methods directly.

<small>[core/event.go:365](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L365)</small>

#### func (*Context) RequestRender

```go
func (ctx *Context) RequestRender()
```

RequestRender marks the tree dirty and notifies the registered render driver. This is the one entry point for "state changed, the UI should re-render" — used by State.Set and by async sources such as timers, so changes that happen outside a native event (where no bridge call is pending a response) can still reach the screen via the push channel.

<small>[core/render_manager.go:63](https://github.com/rohanthewiz/grmob/blob/master/core/render_manager.go#L63)</small>

#### func (*Context) Reset

```go
func (ctx *Context) Reset()
```

<small>[core/context.go:332](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L332)</small>

#### func (*Context) Scope

```go
func (ctx *Context) Scope(key string) *Context
```

Scope returns a stable child context under key, creating it on first use. The child owns its own hook slots, which is what lets a subtree be rendered conditionally — a tab that is only drawn when selected, a navigation frame — without shifting the positional slots of everything around it.

##### Theme and config are re-inherited on every call

A child copies its parent's theme and config when it is built, and a scope is then cached for the life of the app. So a theme that changes \*after\* the scope's first render — the common shape for an app whose palette arrives from the network — would otherwise never reach anything inside it, while everything outside repainted. Nothing in the tree could explain the difference, because the scope is invisible at the call site.

Refreshing here is cheap (two pointer assignments) and safe: Scope is called during a render pass, which render.Manager serializes, and the theme is only ever read during a pass.

It is also the correct semantics. theme and config are \*inherited\* state, not state the scope owns; hook slots are what the scope owns, and those are deliberately left alone.

<small>[core/context.go:382](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L382)</small>

#### func (*Context) Theme

```go
func (ctx *Context) Theme() *Theme
```

<small>[core/context.go:191](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L191)</small>

#### func (*Context) TriggerBoolCallback

```go
func (ctx *Context) TriggerBoolCallback(id string, val bool)
```

TriggerBoolCallback dispatches a bool-carrying event (e.g. a toggle).

<small>[core/event.go:349](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L349)</small>

#### func (*Context) TriggerCallback

```go
func (ctx *Context) TriggerCallback(id string)
```

TriggerCallback dispatches a void event (e.g. a button tap) by callback ID. Unknown IDs are silent no-ops: a late native event racing a purge is expected traffic, not an error.

<small>[core/event.go:335](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L335)</small>

#### func (*Context) TriggerIntCallback

```go
func (ctx *Context) TriggerIntCallback(id string, val int)
```

TriggerIntCallback dispatches an int-carrying event (e.g. tab selection).

<small>[core/event.go:356](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L356)</small>

#### func (*Context) TriggerTextCallback

```go
func (ctx *Context) TriggerTextCallback(id string, val string)
```

TriggerTextCallback dispatches a string-carrying event (e.g. input change).

<small>[core/event.go:342](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L342)</small>

#### func (*Context) With

```go
func (ctx *Context) With(opts ...func(*Context)) *Context
```

<small>[core/context.go:313](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L313)</small>

#### func (*Context) WithConfig

```go
func (ctx *Context) WithConfig(cfg *AppConfig) *Context
```

<small>[core/context.go:219](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L219)</small>

#### func (*Context) WithTheme

```go
func (ctx *Context) WithTheme(theme *Theme) *Context
```

<small>[core/context.go:243](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L243)</small>

### type DisplayMode

```go
type DisplayMode string
```

<small>[core/style.go:1118](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1118)</small>

```go
const (
	DisplayVisible DisplayMode = "visible"
	DisplayHidden  DisplayMode = "hidden"
	DisplayNone    DisplayMode = "none"
	DisplayInline  DisplayMode = "inline"
	DisplayBlock   DisplayMode = "block"
)
```

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

<small>[core/style.go:723](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L723)</small>

### type EditorRef

```go
type EditorRef struct {
	// contains filtered or unexported fields
}
```

EditorRef names one editing surface so that a toolbar can send it commands.

It is the editor half of FocusRef and is used the same way: a hook makes one that is stable across render passes, EditorTarget puts it on the editor, and RunEditorCommand sends to it.

	ref := core.UseEditorRef(ctx)

	core.CodeEditor(src, onChange, rows, core.EditorTarget(ref))
	core.Button("Indent", func() { core.RunEditorCommand(ref, core.EditIndent) })

Unlike FocusRef it carries its own command state rather than pointing at the app's — see the file doc for why one editor's commands are nobody else's.

<small>[core/editor.go:76](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L76)</small>

#### func UseEditorRef

```go
func UseEditorRef(ctx *Context) *EditorRef
```

UseEditorRef returns an EditorRef that is stable for the lifetime of this hook slot, which is what makes the ref usable as an identity.

A hook rather than a bare constructor for exactly FocusRef's reason: a ref built inline in a render function is a new pointer every pass, so EditorTarget would stamp one identity and the toolbar's handler would bump another — and the command would silently never reach a node. NewState both pins the pointer and reserves the cursor slot properly.

A widget that calls this consumes a positional hook slot on the caller's context and must therefore be rendered unconditionally, like any other hook user; comps.CodeEditor says so in its own doc.

<small>[core/editor.go:98](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L98)</small>

### type ExpandedState

```go
type ExpandedState string
```

ExpandedState is whether a disclosure is \*open\* — the accordion section showing its body, the twisty that has been turned.

It is the third of the state types, after SelectedState, and it exists because a control can be on and open at the same time. comps.Accordion is the widget that asked for it: it is the one stateful widget in the components package, its header is the only thing on screen that knows whether the section is showing, and until this type existed the only thing that said so was a chevron glyph — which a screen reader announces as "black right-pointing small triangle" or, more often, not at all.

#### Why this is a second type and not SelectedState reused

The two carry the same three values for the same reason (see below), and reusing the type would have compiled, exported and rendered. It is turned down on two grounds.

\*They are independent facts about one node.\* A control can be selected and expanded at once — a menu button that is both the current tab and showing its submenu — so they are two fields, and two fields typed the same are two fields a caller can transpose. core.AccessibilitySelected(ExpandedOpen) is nonsense that would type-check.

\*They are scoped differently.\* ARIA defines aria-selected for four of this framework's roles and aria-pressed for a fifth; aria-expanded is defined for a sixth set that overlaps but does not match. A shared type would suggest a shared guard, and the guards are what make each state legal where it is written.

#### Why three values and not a bool

The same argument SelectedState makes, arriving from the opposite end. There "off" and "not selectable" were two facts a bool has one spelling for; here it is "closed" and "not a disclosure".

	ExpandedUnset    a Box, a heading, a row of text. Makes no claim, which is
	                 what every node in every tree was before this field, so
	                 the zero value is a no-op.
	ExpandedOpen     the section that is showing.
	ExpandedClosed   a disclosure that could be open and is not.

The third is again the one that would be lost, and losing it is worse here than it is for a selection. A closed accordion that says nothing is announced as an ordinary button: a reader is told they can press it and not that there is anything behind it. "Collapsed" is the whole of what invites the press.

#### Why the values are ARIA's own spellings

Because both web targets write them into the attribute verbatim and need no mapping table, which is the trade core.Role and SelectedState both made. The two natives map what they can, which here is one of them — see Style.AccessibilityExpanded.

<small>[core/expanded.go:56](https://github.com/rohanthewiz/grmob/blob/master/core/expanded.go#L56)</small>

```go
const (
	// ExpandedUnset is the zero value: this node is not a disclosure and says
	// nothing about being open. Every renderer writes no attribute and offers
	// no action.
	ExpandedUnset ExpandedState = ""

	// ExpandedOpen — the disclosure is showing its content.
	ExpandedOpen ExpandedState = "true"

	// ExpandedClosed — the disclosure is a disclosure and is shut. See the
	// type doc for why this is not the same as ExpandedUnset.
	ExpandedClosed ExpandedState = "false"
)
```

#### func ExpandedStates

```go
func ExpandedStates() []ExpandedState
```

ExpandedStates returns both stated values, in declaration order.

ExpandedUnset is excluded for the reason SelectedStates() excludes its own zero value: it is the absence of a claim rather than one of the states, and a coverage check that demanded a renderer arm for it would be asking each renderer to implement "unstated".

Pinned to the const block above by expanded\_enum\_test.go, and consumed by the exporters' round-trip checks the way SelectedStates() is.

<small>[core/expanded.go:95](https://github.com/rohanthewiz/grmob/blob/master/core/expanded.go#L95)</small>

#### func ExpandedWhen

```go
func ExpandedWhen(open bool) ExpandedState
```

ExpandedWhen turns the bool a disclosure already holds into the stated pair.

The twin of SelectedWhen, and it earns its place the same way: the widget owns a \`expanded bool\` (comps.Accordion holds one in NewState), so the conversion would otherwise be written by hand at each call site, and the tempting hand-rolled version — set ExpandedOpen when open, leave it alone otherwise — is exactly the silence the third value exists to prevent.

<small>[core/expanded.go:79](https://github.com/rohanthewiz/grmob/blob/master/core/expanded.go#L79)</small>

### type FlexDirection

```go
type FlexDirection string
```

<small>[core/style.go:1129](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1129)</small>

#### func (FlexDirection) Apply

```go
func (d FlexDirection) Apply(s *Style)
```

<small>[core/style_props.go:251](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L251)</small>

### type FocusRef

```go
type FocusRef struct {
	// contains filtered or unexported fields
}
```

FocusRef names one focusable node so an app can put the cursor in it later.

It is deliberately opaque and carries no state of its own beyond the context it belongs to: identity is the whole point, and identity is the pointer. That is also why it must be stable across render passes — see UseFocusRef.

<small>[core/focus.go:130](https://github.com/rohanthewiz/grmob/blob/master/core/focus.go#L130)</small>

#### func UseFocusRef

```go
func UseFocusRef(ctx *Context) *FocusRef
```

UseFocusRef returns a FocusRef that is stable for the lifetime of this hook slot, which is what makes the ref usable as an identity:

	email := core.UseFocusRef(ctx)

	core.Input(v, "you@example.com", onChange,
	    core.FocusTarget(email),
	)
	core.Button("Next", func() { core.Focus(email) })

A hook rather than a bare constructor because a ref built inline in a render function is a new pointer every pass: FocusTarget would stamp one identity and the click handler would compare against another, so Focus would silently never match a node. Going through NewState both pins the pointer and reserves the cursor slot properly.

It lives in core rather than hooks because hooks imports core and not the reverse; the focus state it closes over is on Context.

<small>[core/focus.go:160](https://github.com/rohanthewiz/grmob/blob/master/core/focus.go#L160)</small>

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

### type Heading

```go
type Heading struct {
	// Magnetic is the bearing relative to magnetic north, in [0, 360).
	Magnetic float64

	// True is the bearing relative to *geographic* north, in [0, 360). The two
	// differ by the local magnetic declination, which is a fraction of a degree
	// in some places and more than 15 degrees in others, so a map application
	// wants this one and a "which way am I facing" readout does not care.
	//
	// It requires the host to know where it is: iOS reports trueHeading only
	// with location authorization, and neither Android's rotation vector nor
	// the browser's orientation events carry it at all. HasTrue says whether
	// the number is real; True is 0 when it is not, and 0 is also a perfectly
	// good northward bearing, which is why the bool exists rather than a
	// sentinel.
	True    float64
	HasTrue bool

	// Accuracy is the reading's error margin in degrees, or -1 when the host
	// does not say. Android reports a bucketed sensor accuracy, iOS a
	// headingAccuracy in degrees, and the browser nothing at all; a large value
	// is the cue to show the platform's figure-eight calibration prompt.
	Accuracy float64

	// Available reports whether this device can produce headings. False before
	// the first event and false forever on a desktop browser; see Received.
	Available bool

	// Received is true once any heading event has arrived, which is what
	// separates "this device has no compass" from "the first reading has not
	// landed yet". A spinner is right for the second and wrong for the first.
	Received bool

	// Active is true while the sensor is running — that is, while the
	// reference count is above zero. It is core's own bookkeeping rather than
	// the host's word, so it flips on the Start call rather than a round trip
	// later.
	Active bool

	// Error is the host's message when it could not start the sensor: a
	// browser motion permission refused, a magnetometer that failed to open.
	// Set alongside Available: false.
	Error string
}
```

Heading is one compass reading. Degrees increase clockwise, so 0 is north, 90 is east, 180 south, 270 west — the convention all three platform APIs and every paper compass share.

<small>[core/heading.go:66](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L66)</small>

#### func CurrentHeading

```go
func CurrentHeading() Heading
```

CurrentHeading returns the last reading, exactly as it arrived — the notification filter described on headingNotifyEpsilon does not apply here. Safe from any goroutine.

<small>[core/heading.go:274](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L274)</small>

#### func (Heading) Cardinal

```go
func (h Heading) Cardinal() string
```

Cardinal returns the 16-point compass abbreviation for the magnetic bearing — "N", "NNE", "NE", ... — which is what a compact readout shows beside (or instead of) the number.

Sixteen points rather than eight or thirty-two: eight is coarse enough that a bearing can sit 22 degrees from the label naming it, and thirty-two needs four-letter names ("NbE") that no reader outside sailing recognises.

<small>[core/heading.go:118](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L118)</small>

### type JustifyContent

```go
type JustifyContent string
```

<small>[core/style.go:1128](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1128)</small>

#### func JustifyContents

```go
func JustifyContents() []JustifyContent
```

JustifyContents returns every declared JustifyContent, in declaration order.

Main-axis distribution. The DOM pair emits these verbatim (core's spellings are the CSS ones), so the drift risk is entirely on the natives, where the six values are spread across dispatches that each answer for part of the question: GrMobFlex.swift computes a leading offset in one switch and an inter-item gap in another, and a value absent from \*both\* silently renders as flex-start.

<small>[core/alignment.go:129](https://github.com/rohanthewiz/grmob/blob/master/core/alignment.go#L129)</small>

#### func (JustifyContent) Apply

```go
func (j JustifyContent) Apply(s *Style)
```

<small>[core/style_props.go:250](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L250)</small>

### type LifecycleState

```go
type LifecycleState string
```

LifecycleState is where the app sits in the platform's foreground / background lifecycle. See the package comment above for the three values.

<small>[core/lifecycle.go:53](https://github.com/rohanthewiz/grmob/blob/master/core/lifecycle.go#L53)</small>

```go
const (
	LifecycleActive     LifecycleState = "active"
	LifecycleInactive   LifecycleState = "inactive"
	LifecycleBackground LifecycleState = "background"
)
```

#### func CurrentLifecycle

```go
func CurrentLifecycle() LifecycleState
```

CurrentLifecycle reports the last state the host announced; active until it has announced anything.

<small>[core/lifecycle.go:75](https://github.com/rohanthewiz/grmob/blob/master/core/lifecycle.go#L75)</small>

### type Location

```go
type Location struct {
	// Lat and Lng are degrees, WGS-84 — the datum every platform API and every
	// tile provider here uses, so no conversion happens anywhere in this
	// framework.
	Lat, Lng float64

	// Accuracy is the horizontal radius of the fix in metres: the device
	// believes it is somewhere inside this circle. -1 when the host does not
	// say, which in practice no host does.
	//
	// It is not an error bar to be ignored. 5 metres is a GPS fix outdoors,
	// 50 is a fix through a roof, 2000 is a guess from the cell tower or the
	// IP address, and the last one is what a desktop browser reports while
	// looking exactly like the first to any code that reads only Lat and Lng.
	Accuracy float64

	// Altitude is metres above the WGS-84 ellipsoid, and HasAltitude says
	// whether the number is real. 0 is a perfectly good altitude — it is most
	// of the world's coastline — which is why the bool exists rather than a
	// sentinel, exactly as Heading.HasTrue does one file over.
	//
	// Vertical accuracy is deliberately not carried. It is reported by two of
	// the three hosts, it is a different and much larger number than the
	// horizontal one, and nothing has asked for it; a field no screen reads is
	// three hosts remembering to fill it in for nothing.
	Altitude    float64
	HasAltitude bool

	// Available reports whether this device can produce a fix. False before
	// the first event, and false after a refusal or a hardware failure; see
	// Received for the difference between "no" and "not yet".
	//
	// It is the best answer anyone has rather than a guarantee: while the
	// sensor is running and has not reported yet — Active with Received false,
	// which is what LocationAcquiring puts the record back into — it is true,
	// because a sensor that accepted the start is one that can try.
	Available bool

	// Received is true once any location event has arrived. A first fix can
	// take tens of seconds on cold GPS, so this is the flag that separates
	// "still acquiring" — which is a spinner, and a long one — from "this
	// device will never tell you", which is a different screen.
	//
	// Reset to false when a run begins, which is the half of it that took a
	// second emulator run to find: a refused start that is later granted
	// re-arms on both natives, and without the reset the record still carried
	// the refusal all the way to the first fix. Active && !Received is the
	// acquiring state, and LocationAcquiring is how a host gets back to it.
	Received bool

	// Active is true while the sensor is running, which is core's own
	// reference count rather than the host's word: it flips on the Start call
	// rather than a round trip later.
	Active bool

	// Error is the host's message when it could not start or keep the sensor:
	// a permission refused, location services switched off system-wide, a
	// browser with no geolocation. Set alongside Available: false, and cleared
	// whenever a run begins — the reason a previous attempt failed is not a
	// statement about the one now running.
	Error string
}
```

Location is one position fix.

<small>[core/location.go:88](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L88)</small>

#### func CurrentLocation

```go
func CurrentLocation() Location
```

CurrentLocation returns the last fix, exactly as it arrived — the notification filter does not apply here. Safe from any goroutine.

<small>[core/location.go:278](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L278)</small>

### type MatchCase

```go
type MatchCase[T comparable] struct {
	Value   T
	View    View
	Default bool
}
```

MatchCase Generic Match for comparable values

<small>[core/conditionals.go:53](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L53)</small>

#### func Case

```go
func Case[T comparable](val T, view View) MatchCase[T]
```

<small>[core/conditionals.go:59](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L59)</small>

#### func Default

```go
func Default[T comparable](view View) MatchCase[T]
```

<small>[core/conditionals.go:63](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L63)</small>

### type ModalNode

```go
type ModalNode struct {
	Visible   bool
	OnDismiss func()
	Backdrop  string
	Content   []View
}
```

<small>[core/modal.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L7)</small>

### type ModalProp

```go
type ModalProp interface {
	Apply(*ModalNode)
}
```

<small>[core/modal.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L3)</small>

#### func Backdrop

```go
func Backdrop(color string) ModalProp
```

<small>[core/modal.go:60](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L60)</small>

#### func ModalContent

```go
func ModalContent(children ...View) ModalProp
```

ModalContent sets the views drawn inside the overlay. It appends rather than replaces, so content may be assembled across several props if a caller finds that clearer; order is render order, top to bottom.

The content renders every pass regardless of Visible — a Modal hides, it does not unmount. Visible is an ordinary prop the host maps to visibility (display on the web), so toggling it is a cheap prop patch, not a subtree add/remove, and any state hooks inside the content survive a close. That makes the trade-off against navigation explicit: a dismissed modal reopens exactly as it was left, where a popped Navigator frame starts fresh. A dialog whose state must NOT survive dismissal should reset it in OnDismiss, where the intent is recorded.

<small>[core/modal.go:78](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L78)</small>

#### func OnDismiss

```go
func OnDismiss(fn func()) ModalProp
```

<small>[core/modal.go:54](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L54)</small>

#### func Visible

```go
func Visible(v bool) ModalProp
```

<small>[core/modal.go:48](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L48)</small>

### type Node

```go
type Node struct {
	Type string
	// Everything but Type is omitted from JSON when it is at its zero value,
	// for the reason core.Style's fields are — see the note above that struct.
	// Type is not, because a node without one is not a node, and a renderer
	// reading an absent Type would fall through its dispatch to whatever its
	// default arm is rather than say what is wrong.
	Key      string         `json:",omitzero"`
	Props    map[string]any `json:",omitzero"`
	Style    *Style         `json:",omitzero"`
	Children []*Node        `json:",omitzero"`
}
```

Node is the retained render-tree element the reconciler diffs.

Immutability contract: a Node is frozen once its render pass returns it. Builders may assemble a node freely while constructing it (Keyed sets Key, containerNode applies behavior props), but after render nothing may write to it — the reconciler only reads, and renderers must also only read. The contract is what makes sharing safe: Cached returns the same \*Node every pass and Diff treats pointer equality as proof the subtree is unchanged, so a post-render mutation would silently never reach the screen.

<small>[core/node.go:12](https://github.com/rohanthewiz/grmob/blob/master/core/node.go#L12)</small>

#### func Render

```go
func Render(ctx *Context, view View) *Node
```

Render renders view into ctx after restarting ctx's hook cursors. It is the entry point for a host driving passes by hand; render.Manager does the same two steps itself (with the debug pass boundary around them) and does not call this.

<small>[core/navigation.go:364](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L364)</small>

### type Position

```go
type Position string
```

<small>[core/style.go:1150](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1150)</small>

```go
const (
	PositionRelative Position = "relative"
	PositionAbsolute Position = "absolute"
	PositionFixed    Position = "fixed"
	PositionSticky   Position = "sticky"
)
```

### type Progress

```go
type Progress struct {
	Reading       ProgressReading
	Now, Min, Max float64
}
```

Progress is a ValueRange's numbers, resolved.

Now, Min and Max are meaningful when Reading is ProgressDeterminate. For ProgressEmptyRange they are the numbers as stated, unclamped, so a caller reporting the problem can name them; for the other two readings they are zero, which is not a position.

<small>[core/value.go:192](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L192)</small>

### type ProgressReading

```go
type ProgressReading string
```

ProgressReading is what a ValueRange's three numbers amount to once somebody has to act on them.

#### Why core owns this and the web exporters do not use it

The two DOM targets hand aria-valuenow/-min/-max to a browser verbatim, and a browser applies ARIA's rules itself: the implicit 0..100, the reading of a missing position as an indeterminate bar. That pass-through is right and it is also why this reading had nowhere to live — the only code in the repository applying those rules was Kotlin, in a three-way branch inside a Compose semantics lambda, checked by looking for substrings in the file.

The rules are ARIA's and this type's, though, not Compose's: both of them are already stated in prose on the fields below, and the Kotlin is a transliteration of that prose. Naming them here makes the transliteration comparable — android/verify runs GrMobProgress.kt against this function over internal/valuefixture's table — which is the same relationship core.SelectMenuSections has with the four picker menus.

Text has no part in it. The words are a separate claim on a separate property (aria-valuetext, stateDescription, accessibilityValue) and they are announced on nodes that carry no range at all, so a reading about the numbers must not depend on them.

<small>[core/value.go:160](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L160)</small>

```go
const (
	// Nothing numeric was stated. A `text` on an ordinary node must not turn
	// it into a progress bar, so this is the reading that leaves a platform's
	// range property untouched.
	ProgressUnstated ProgressReading = "unstated"

	// Bounds and no position: a bar that is running with no idea how far.
	// ARIA spells it by omitting aria-valuenow; Compose has a name for it.
	ProgressIndeterminate ProgressReading = "indeterminate"

	// A position inside a real range. Min and Max carry ARIA's own defaults
	// of 0 and 100 when unstated, which is what makes a bare Now announce as
	// a percentage, and Now is clamped into the range.
	ProgressDeterminate ProgressReading = "determinate"

	// A position inside a range that is not one — Max at or below Min. It is
	// separated from Unstated because the two are different mistakes and a
	// platform may want to treat them differently: Compose cannot express it
	// at all (ProgressBarRangeInfo requires a non-empty range and throws), so
	// it drops the property rather than crashing a render over an
	// accessibility annotation.
	ProgressEmptyRange ProgressReading = "empty-range"
)
```

### type PropsAndChildren

```go
type PropsAndChildren any
```

<small>[core/layout.go:5](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L5)</small>

#### func MaybeProp

```go
func MaybeProp(cond bool, prop PropsAndChildren) PropsAndChildren
```

MaybeProp conditionally contributes one item to a container's argument list: Row, Column, Card, Box and List, the variadic ...PropsAndChildren builders. It returns prop when cond holds and an untyped nil otherwise, and containerNode skips a nil item, so a false condition costs the tree nothing at all — no node, no slot, no style.

It exists because core.If cannot do this job, in two separate ways:

 1. If(false, view) returns Fragment(), and an empty Fragment is still a real child node: the reconciler walks and diffs it on every pass, and it occupies a child index, so anything addressing children by position counts it. If earns its place where the alternative is a whole branch of the tree; it is the wrong tool for one optional item in a row of three.

    It does not, however, draw anything. A grouping node with no children renders no box on any of the three targets — that is what both native renderers have always done, and what the HTML exporter now does too. An earlier version of this note claimed the empty Fragment took a flex slot and opened a stray Gap; that was true only of the exporter, which wrapped every grouping node in a div, and it is fixed. The cost is a node, not a gap.

 2. If is typed View -> View. There is no If for a StyleProp or a BehaviorProp, so "apply this padding only when selected" or "attach OnClick only when a handler was supplied" had no expression form at all. MaybeProp takes PropsAndChildren, so it covers all three item kinds with one helper.

Together those replace the accumulate-into-a-slice idiom this codebase kept reaching for:

	items := make([]core.PropsAndChildren, 0, 3)
	items = append(items, core.UseStyle(bubble))
	if !mine {
	    items = append(items, core.Text(from))
	}
	items = append(items, core.Text(body))
	return core.Column(items...)

	// becomes
	return core.Column(
	    core.UseStyle(bubble),
	    core.MaybeProp(!mine, core.Text(from)),
	    core.Text(body),
	)

Two limits, both deliberate:

prop is evaluated eagerly, like any Go argument — the condition does not guard it. That is safe for the prop constructors, which only build values (core.Text returns a closure; nothing renders until the container renders it), but MaybeProp is not a substitute for an if statement around an expression that would panic or do real work on the false path.

The return type is PropsAndChildren (i.e. any), so this is only valid in the container builders' argument lists. Text and Button take typed variadics (...StyleProp), which will not accept it — and must not, since their loops call Apply on every element and would panic on a nil.

<small>[core/conditionals.go:135](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L135)</small>

### type Region

```go
type Region struct {
	// Lat and Lng are the centre of the view, in degrees, WGS-84.
	Lat, Lng float64

	// Zoom is the slippy-tile zoom level every one of these engines speaks: 0
	// is the whole world, 19 is a building. Fractional values are allowed and
	// meaningful — a pinch lands between levels, and the hosts report what the
	// user actually reached rather than rounding it.
	//
	// A zero Zoom means DefaultMapZoom rather than "the whole world", which is
	// the one legitimate value this type spends on a default. It is the same
	// trade comps.StaticMap.Zoom makes and for the same reason: a map with
	// no zoom stated is a map somebody forgot to scale, and the world is never
	// what they meant.
	Zoom float64
}
```

Region is a place and a scale: where a map is looking and how closely.

One struct rather than three floats at every call site, and the same struct in both directions — MapView takes one and OnRegionChange hands one back, so an app that echoes the user's pan into its own state is storing the type it renders from. Two types here (a "MapRegion" in and a "RegionChange" out) would differ in nothing and convert at every seam.

<small>[core/mapview.go:157](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L157)</small>

#### func ParseRegion

```go
func ParseRegion(s string) (Region, bool)
```

ParseRegion reads a host's "lat,lng,zoom" payload. The bool is false for anything that does not parse, and a caller must not substitute zeros: see OnRegionChange.

Exported because all three hosts format this string and a test in each harness has to read one back. Keeping the parse in one place is also what makes the wire format a single fact rather than three.

<small>[core/mapview.go:359](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L359)</small>

### type RenderError

```go
type RenderError struct {
	// Value is exactly what was passed to panic(): usually an error or a
	// string, but it can be any type.
	Value any

	// Stack is debug.Stack() captured inside the deferred recover, so it
	// still contains the frames being unwound — the component that actually
	// panicked is in here, which is the whole point of keeping it. Nil only
	// if a RenderError was constructed by hand.
	Stack []byte
}
```

RenderError is a panic that escaped a component's Render and was caught by an ErrorBoundary (or by the render driver's top-level guard).

It carries the raw panic value rather than just a message because the panic may well be a real error worth inspecting: a \*net.OpError, a wrapped sentinel, a custom type the app wants to switch on. Unwrap exposes it to errors.Is/errors.As when it is one, so

	if errors.Is(err, sql.ErrNoRows) { ... }

works inside a fallback even though the value travelled through panic().

<small>[core/error_boundary.go:19](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L19)</small>

#### func Guard

```go
func Guard(fn func()) (rerr *RenderError)
```

Guard runs fn and converts a panic escaping it into a \*RenderError, returning nil when fn completes normally.

Exported because the render driver needs the same guard around a whole pass that ErrorBoundary needs around a subtree, and render is a separate package. It deliberately takes a func() rather than a View: the driver must also cover the root-view \*construction\* call, not only Render.

Guard restores nothing — it is the bare recover. Callers that intend to keep rendering after the failure are responsible for repairing whatever the half-finished work left behind; see renderRecovered for what that means inside a boundary.

<small>[core/error_boundary.go:63](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L63)</small>

#### func (*RenderError) Error

```go
func (e *RenderError) Error() string
```

Error renders the panic value as a message. The "panic during render" prefix is deliberate: a RenderError frequently ends up in a log line next to ordinary application errors, and without it a bare "index out of range" gives no hint that it came from a render pass.

<small>[core/error_boundary.go:35](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L35)</small>

#### func (*RenderError) Unwrap

```go
func (e *RenderError) Unwrap() error
```

Unwrap exposes the panicked value to errors.Is/As when it is an error, and returns nil otherwise (a panic("boom") wraps nothing).

<small>[core/error_boundary.go:44](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L44)</small>

### type RenderManager

```go
type RenderManager struct {
	// contains filtered or unexported fields
}
```

RenderManager is the app's "state changed" notification point: one registered handler (see OnStateChange), invoked whenever anything in the context tree calls RequestRender. One instance per NewContext root, shared by pointer with every derived context.

It is keyed by string rather than holding a bare func because it once carried a second, parallel registration API — RegisterRender, which minted "render\_N" ids, and SubscribeRender, which called it and threw the id away. Nothing ever triggered those ids: State.Set has always notified the hardcoded "default" key, so every SubscribeRender handler was unreachable while the map grew by one entry per call. Both are gone; OnStateChange is the registration side that actually completes the circuit.

<small>[core/render_manager.go:19](https://github.com/rohanthewiz/grmob/blob/master/core/render_manager.go#L19)</small>

#### func NewRenderManager

```go
func NewRenderManager() *RenderManager
```

<small>[core/render_manager.go:24](https://github.com/rohanthewiz/grmob/blob/master/core/render_manager.go#L24)</small>

#### func (*RenderManager) TriggerRender

```go
func (r *RenderManager) TriggerRender(id string)
```

TriggerRender invokes the handler registered under id, if any.

<small>[core/render_manager.go:31](https://github.com/rohanthewiz/grmob/blob/master/core/render_manager.go#L31)</small>

### type ResponsiveStyle

```go
type ResponsiveStyle map[string]Style
```

<small>[core/style.go:1105](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1105)</small>

### type RichSelection

```go
type RichSelection struct {
	Start, End int

	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Code      bool

	// Link is the URL under the caret, or "" when there is none. A toolbar uses
	// it to decide between offering "Link" and offering "Unlink", and to
	// pre-fill the prompt when editing one.
	Link string

	// Block is the kind of the block the caret is in. An empty value means the
	// host had nothing to say, which happens on a document with no blocks yet.
	Block richtext.BlockKind
}
```

RichSelection is what a rich-text editor reports about its caret: where it is, and what formatting is active there.

The marks matter more than the offsets, and that is the reason this is a struct rather than the two ints a CodeEditor reports. A toolbar has to show its bold button as \*on\* when the caret is inside bold text, and nothing in Go can work that out — the document is Go's, but where the caret is inside it is the host's.

Start and End are byte offsets into the document's plain text (richtext.Doc.PlainText), which is the one coordinate system all four hosts can produce and which is stable across the marks. They are there for a status line and for "is anything selected"; a command never needs them, because every command acts on the host's own selection.

<small>[core/richtext.go:147](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L147)</small>

#### func (RichSelection) HasSelection

```go
func (s RichSelection) HasSelection() bool
```

HasSelection reports whether anything is actually selected, as opposed to a bare caret. The distinction is what a "Link" button needs: linking an empty selection has nothing to attach to.

<small>[core/richtext.go:169](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L169)</small>

### type Role

```go
type Role string
```

Role is what a node \*is\* to assistive technology, as distinct from what it is called (AccessibilityLabel) or what tapping it does (AccessibilityHint).

A screen reader announces "Sermons, heading" or "March, column header" because something told it the element's kind. Nothing in this framework could say that until now: every container is a Box or a Row, every one of them exports as a \<div>, and a screen built entirely out of them is a flat run of text to VoiceOver and TalkBack no matter how carefully it is labelled. Three widgets hit the wall independently — DataTable wanting to be a table, the screen-furniture bundle wanting a banner, a heading and a search landmark, and Calendar's forty-two tappable day cells wanting to be buttons — which is what turned "an ARIA role prop, some day" into this file.

#### Why the values are spelled in ARIA

The set has to be \*some\* vocabulary, and the four renderers do not share one. ARIA is the only candidate that is a published standard with a name for every case here; SwiftUI's AccessibilityTraits and Compose's SemanticsProperties are small, partly overlapping sets that would each need a mapping table whichever vocabulary core picked. Choosing ARIA means the two DOM targets need no table at all — the value is the attribute — and the two natives map what they can, which is the same work they already do for ContentMode and the alignments.

#### What each target does with a role

	role          | DOM (both)      | SwiftUI trait  | Compose semantics
	--------------+-----------------+----------------+------------------------
	heading       | role="heading"  | .isHeader      | heading()
	columnheader  | role=…          | .isHeader      | heading()
	button        | role="button"   | .isButton      | role = Role.Button
	link          | role="link"     | .isLink        | —
	search        | role="search"   | .isSearchField | —
	img           | role="img"      | .isImage       | role = Role.Image
	tab           | role="tab"      | —              | role = Role.Tab
	tablist       | role="tablist"  | .isTabBar      | —
	radiogroup    | role=…          | —              | selectableGroup()
	radio         | role="radio"    | —              | role = Role.RadioButton
	status        | role="status"   | —              | liveRegion = Polite
	alert         | role="alert"    | —              | liveRegion = Assertive
	log           | role="log"      | —              | liveRegion = Polite
	progressbar   | role=…          | —              | — (but see below)
	the other 13  | role=…          | —              | —

The other thirteen are table, rowgroup, row, cell, list, listitem, listbox, option, tabpanel, banner, navigation, toolbar and group — the tabular set, both collection pairs, the region a tab shows, the landmarks, and the naming role.

progressbar has a row of its own because its dashes mean less than the others'. The \*role\* maps to nothing on either phone — neither has a word for what a progress bar is — while the value beside it maps to Compose's progressBarRangeInfo, which is one of the better mappings in this framework: TalkBack turns the numbers into a percentage it localizes itself. So the thing a reader most wants to hear does arrive on one native; it arrives through Style.AccessibilityValue rather than through this field. See core.ValueRange.

The tab pair is the one row of that table where the two natives disagree about \*which half\* they can say, and it is a useful illustration of why the vocabulary is ARIA's rather than either platform's. Compose has a Role.Tab for the control and nothing for the strip around it; SwiftUI has .isTabBar for the strip and nothing for the control. Neither could have supplied the pair, and a caller marking up a tab strip sets both and gets whichever half each platform knows.

Fourteen of the twenty-seven do nothing on either native, and that is the honest state of those platforms rather than a gap to be filled later: neither has a tabular semantics vocabulary a role can be mapped onto (Compose has collectionInfo, which describes counts and indices this prop does not carry), neither has a listbox in its semantics vocabulary (both spell a chosen item as a \*state\* instead, which is why the selectable pair costs them nothing to leave out — see RoleListBox), and neither has landmarks at all — VoiceOver's rotor navigates by heading, not by banner.

RoleGroup is the one empty pair in that fourteen that is empty for the opposite reason, and it is worth telling apart. The other thirteen are silent because the platform has no way to say the thing; \`group\` is silent because neither platform \*needs\* it — both honour an accessibility label on any node at all, and making that label legal is the whole of what the role does. See its own block below.

A role that maps to nothing is still worth setting. The web is a first-class target here, the mapping can improve later without the call sites changing, and a role that is right on one platform and inert on two is strictly better than a div.

#### A structural role owns what is inside it

The tabular five and the three collection pairs are not labels on a container — they are claims about what the container holds. role="list" says its children are listitems; role="listbox" says its children are options; role="table" says its children are rows, or rowgroups holding rows. A reader acts on the claim rather than re-deriving it: it announces the count ("list, five items"), it offers item-by-item navigation, and it reads the structure instead of the text.

A container can fail that claim in two directions, and both produce something worse than no role at all.

\*A gap in the chain.\* An unroled element between the container and its items breaks the ownership: role="table" wrapping a plain div wrapping the rows reports a table with no rows. This is what RoleRowGroup exists to close, and its comment below has the detail.

\*A foreign child.\* A container that holds something which is not an item — a footer, a heading, a spinner — is claiming a structure it does not have. ARIA specifies the children a role requires and not what to do with any others, so what a reader makes of the odd child is an implementation's choice rather than a promise: it may be counted, skipped, or announced as the item it is not.

So: \*\*a container that mixes items with chrome cannot take a structural role.\*\* Either the chrome moves outside the container, or the container stays roleless — in which case its contents are announced as the text they are, which is what everything was before this type existed and is not a regression. Reaching for the role anyway is the one move that makes the screen worse.

The rule is easy to meet by accident, because the shapes that hit it are the ordinary ones. DataTable meets it three times in one widget: it puts a rowgroup on its body list to close a gap, \*withholds\* that rowgroup when the list holds a placeholder instead of rows, and documents a grouped table's band headings — foreign children it cannot move — as a limit it cannot close from where it sits. A paged list in the first app to adopt these roles met it a fourth time without having read any of that: its "Load more" footer sits inside the core.List, so the list cannot be a list.

The landmarks, the live regions and the content roles carry no such promise and are not subject to this. A banner, a navigation region or a log owns whatever it likes; RoleHeading, RoleButton, RoleLink, RoleImg and RoleTab describe the node itself. Five of the const blocks below hold a role that makes a claim about its children — the tabular set, the three collection pairs, and the tablist half of the tab pair — which is where to look rather than here if a role is ever added to any of them. (RoleTab itself does not: it describes one control, the way RoleButton does, and only the strip around it claims what it contains.)

#### Roles a node type carries for itself

Some semantics are not the author's to state, because the node type already knows them. core.Button exports as a \<button> and builds a real control on both natives, so nobody has to say \`button\` — RoleButton exists for the \*other\* case, a Box or a Row with an OnTap, which every renderer draws as inert scenery.

core.Modal is the same shape and has no vocabulary entry at all. The two natives already present it through a platform dialog (a SwiftUI sheet, a Compose Dialog), each of which announces itself; the two DOM renderers drew a plain div, so the overlay was the one target where a dialog was not a dialog. Both now write role="dialog" and aria-modal="true" as part of the Modal chassis, next to the fixed-overlay rules — semantics the node type owns, not a value a caller passes.

So there is deliberately no RoleDialog. Adding one would put the burden back on the author for something three of the four targets already do unasked, and would cost two more native arms that could only be empty — which in this vocabulary means "this platform cannot say it", the opposite of the truth here. An author who overrides a Modal's role with a core.AccessibilityRole still wins, on the same principle the chassis follows for style: the framework's default goes first.

RoleTabPanel is \*not\* a third case of that shape, and for five sessions it was recorded as one. The argument that kept it out was that a tab panel is not really a role but one end of a \*relationship\* — the announcement a reader gives ("tab 2 of 3, Sermons, tab panel") comes from aria-controls and aria-labelledby pointing between two elements, and both are IDREFs, which Style does not carry. That was true when it was written and stopped being true the moment AccessibilityControls landed: the pointing half exists now, and the constant was the only piece still missing from a hand-built strip.

So the division is not "does the node type know it" but "can an author say it". core.TabView still owns its own wiring end to end — it mints the ids, writes this role, and keeps aria-selected in step — exactly as core.Modal owns its dialog role; the difference from RoleDialog is that a hand-built tab strip is a shape people actually build, and a hand-built modal is not. examples/social's bottom bar is one, and until this constant its regions were \`group\`s that three tabs claimed to control.

#### What used to block it, and what replaced the block

The WASM runtime has to tell a panel it wired itself from a role an author wrote, or it unwires and rewires the same element on alternate syncs. It did that by the value — "tabpanel" could only have come from wireTabPanel, because no core.Role spelled it — which made the \*absence of this constant\* load-bearing, and which is why the entry sat.

The discriminator is now a data-grmob-panel marker that both web targets write, in the channel data-grmob-chrome already uses for the same kind of fact: this element is something the framework put here. That is a better answer than the old one even setting the constant aside, because it says what it means — the old test asked "is this value one no author could have written", which is a fact about the vocabulary standing in for a fact about the element.

RoleTab and RoleTabList were never in the same position and were always present: they say what a control and a strip \*are\*, which is a claim about one element.

#### Every renderer names every role

Both natives dispatch on the string, arm by arm, so a role with no arm falls into a catch-all and is silently inert — the same failure ContentMode has, where a mode nobody taught the natives about draws as \`fit\` on device and as the browser default on the web with no error anywhere. So each native spells out the roles it does \*not\* implement alongside the ones it does, and mobile/verify/role\_test.go holds both dispatches against Roles(). Adding a constant below without adding it there fails \`go test ./...\`.

<small>[core/role.go:212](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L212)</small>

Tabular structure. The five together describe a table to a screen reader — separately they describe nothing, since a cell outside a row outside a table is not a thing ARIA recognizes. DataTable sets all five.

RoleRowGroup is the one that looks like padding and is not. A table's rows have to be \*owned\* by the table, and an unroled container between the two breaks the ownership: role="table" wrapping a plain div wrapping the rows reports a table with no rows. DataTable has exactly that shape — its body is a core.List, which is a div — so without a rowgroup on it the other four would describe an empty table, which is worse than describing nothing.

```go
const (
	RoleTable        Role = "table"
	RoleRowGroup     Role = "rowgroup"
	RoleRow          Role = "row"
	RoleColumnHeader Role = "columnheader"
	RoleCell         Role = "cell"
)
```

Collections. The looser cousin of the table pair, for a run of items that is a list rather than a grid — GroupedList's bands, a strip of cards.

RoleList makes the same claim about its children that RoleTable does, and loses it to chrome just as easily — the rule is structural, not tabular: a container holding items \*and\* a "Load more" footer, a section heading or a spinner is not a list, whatever it looks like. See "A structural role owns what is inside it" above — that is the section to read before putting one of these on a container that was not built to hold items alone.

```go
const (
	RoleList     Role = "list"
	RoleListItem Role = "listitem"
)
```

The selectable collection: a run of choices, and one choice in it. The fourth structural block, and the pair \`list\`/\`listitem\` above cannot stand in for.

##### Why a second collection pair rather than a state on the first

Because ARIA will not carry it. \`aria-selected\` is defined for gridcell, option, row, tab and columnheader — not for \`listitem\` — so a list item that says it is chosen says it into a void on both web targets: the attribute is written, the DOM inspector shows it, and no reader announces anything. A list is \*content\* and a listbox is a \*control\*, and the state only exists on the control side.

comps.ListRow is what asked. Its selected row spelled the state into its own accessible name (", selected") because both other doors were shut: \`listitem\` cannot carry the state, and \`button\` — which carries the neighbouring \`aria-pressed\` — would make the row a foreign child of any role="list" around it, costing the whole list its shape for one row's announcement. This pair is the door that was left.

##### What a listbox promises, and who keeps the promise

A listbox is a real control in ARIA's model, and the pattern that goes with it is larger than two attributes: the container takes keyboard focus, the arrow keys move an active option, and the reader is told which option is active through a roving tabindex or aria-activedescendant.

None of that is \*here\*, and it does not need to be. This type is a vocabulary — it says what a node is, and nothing in core stamps a tabindex or reads an arrow key (core/focus.go is about putting the cursor in a named field, which is a different question and one that costs a render pass per keystroke). What closed the gap instead was noticing that the vocabulary already says everything the pattern needs:

	what is a member of what   this pair, and the structural rule above that
	                           makes a listbox's contents its own
	which one is chosen        Style.AccessibilitySelected
	which way the arrows go    the container's own layout axis
	what activation means      the OnTap the author already wrote

So the WASM runtime supplies the whole keyboard half from what crosses the wire, with no new prop and no source change in any screen that had already said the above — see "Composite widgets are operable" in docs/platforms/wasm.md. htmlout deliberately writes none of it: a roving tabindex without the handler that moves it takes every option but one out of the tab order and reaches none of them, so the attribute is behaviour rather than semantics and a static export must not carry it.

On the two phones there was never a gap — VoiceOver and TalkBack navigate a collection by swipe, not by arrow key — which is also why neither native has a listbox in its semantics vocabulary at all: both spell a chosen item as the \`selected\` state this pair exists to make \*legal\*, and they honour that state on any node without being told what contains it.

##### The depth question, answered the other way

\`listitem\` carries aria-level and \`option\` does not, so a row cannot be both a choice and a depth: the two roles are exclusive and only one of them takes a level. ARIA does have a role for an item that is both — \`treeitem\` inside a \`tree\`, which supports aria-level and aria-selected together — and it is deliberately not here, because a tree is a third pattern with its own expansion state and its own keyboard contract, and nothing in this repository has one. See comps.ListRow.Selectable, which is where the two fields meet and where the precedence is written down.

```go
const (
	RoleListBox Role = "listbox"
	RoleOption  Role = "option"
)
```

The radio pair: a set of mutually exclusive choices that are all on screen, and one choice in it. The third collection pair, and the third structural block that makes a claim about its children — role="radiogroup" says the things inside it are radios. See "A structural role owns what is inside it" above.

##### Why a pair of its own when listbox and option already carry a choice

Because the two patterns promise different things, and a widget that wears the wrong one is announced as the wrong control. An option is \*selected\*; a radio is \*checked\*. A listbox is one tab stop whose arrows move a highlight and leave the choice alone unless the widget asks otherwise (AccessibilitySelectionFollowsFocus); a radio group is one tab stop whose arrows move the check itself, always. comps.RadioGroup shipped as a listbox for want of this pair and said "option, selected" for "radio button, checked" — true, and the weaker of the two words.

##### One state field, a third attribute

The state is Style.AccessibilitySelected, the field every other choice uses, and not a new AccessibilityChecked. The two web exporters write it as aria-checked for a radio, beside aria-selected for an option and a tab and aria-pressed for a button; the natives have one spelling of "this one is on" each and use it for all of them. A second field would let a radio carry both a selection and a check, which is a state no control has.

##### What each target does with it

	web       role="radiogroup" / role="radio" and aria-checked; the WASM
	          runtime adds the keyboard (one tab stop on the checked radio,
	          the arrows move and check), htmlout writes no tabindex, as for
	          every composite
	Compose   radiogroup is selectableGroup(), radio is Role.RadioButton, and
	          the state is the `selected` property grMobSelected sets
	SwiftUI   no trait for either; the state arrives as .isSelected, the same
	          loss the listbox pair has on this platform

```go
const (
	RoleRadioGroup Role = "radiogroup"
	RoleRadio      Role = "radio"
)
```

The tab pair: a strip of controls that switches what the screen is showing, and one control in it. The third structural block, and it makes the same claim about its children the other two do — role="tablist" says the things inside it are tabs, so a strip that also holds a "+" button or a count is not a tablist. See "A structural role owns what is inside it" above.

Both are for a \*hand-built\* strip. core.TabView needs neither: it writes these two, plus the tabpanel half and the aria-controls/aria-labelledby wiring between them, from the node type — see "Roles a node type carries for itself" above for why there is no RoleTabPanel to complete the set.

A tab is the vocabulary's second selectable control, after RoleButton, and the state it wants is Style.AccessibilitySelected. The pair is what makes the strip legible: a reader announces "tab, selected" only when the tab says so, and a tablist in which \*no\* tab says so announces every one of them as unselected. So a strip sets the state on every tab, not just the live one — SelectedOff is a value with a job here, not a way of saying nothing.

It is also what the arrow keys move between. The WASM runtime reads this pair the same way it reads the listbox one and supplies ARIA's keyboard half — one tab stop for the strip, Left/Right within it (Up/Down for a strip laid out as a column), Home and End to the ends — from the roles and the selection alone. A hand-built strip gets it by saying what it is; see the listbox pair above for the argument, and docs/platforms/wasm.md for what each target does.

```go
const (
	RoleTab     Role = "tab"
	RoleTabList Role = "tablist"
)
```

Landmarks: the regions of a screen a reader jumps between rather than reads through. AppBar is a banner, a tab strip is navigation, SearchField is a search, ChipStrip is a toolbar.

```go
const (
	RoleBanner     Role = "banner"
	RoleNavigation Role = "navigation"
	RoleSearch     Role = "search"
	RoleToolbar    Role = "toolbar"
)
```

Live regions: content that changes on its own and should be announced when it does, without the reader having to be looking at it.

The first two differ in how rudely they interrupt. Status waits for a pause — "saved", "3 new items". Alert cuts in — a failure, an expiry, anything the reader must hear before continuing. Banner picks between them by variant, which is the distinction its Variant already draws visually.

RoleLog is the third, and it is not a politeness level: log and status are both polite, and on Android they are the same call. What differs is the \*shape of the content\*, which is a promise about the element rather than about the interruption.

	status   one advisory that is replaced. "Saved", "3 new items". A reader
	         announces the region's new state, and the old text is gone.
	log      a record that is appended to and whose order is meaningful — a
	         chat transcript, a console, a running import. A reader announces
	         what arrived, and what came before it is still there to be read
	         back.

The distinction is the difference between "the region now says this" and "this was added at the end", which is why a transcript marked \`status\` announces correctly and reads back wrong: the whole conversation is one region that has, as far as the reader is concerned, just changed entirely.

It maps to Compose's polite live region, the same call \`status\` makes, because that is the whole of what Compose can say — an honest collapse rather than a second spelling of one fact. On the web the two are different roles with different reading behaviour, which is where the field earns its place.

```go
const (
	RoleStatus Role = "status"
	RoleAlert  Role = "alert"
	RoleLog    Role = "log"
)
```

Content roles: what a node is when it is not a region.

RoleButton is for a tappable container — a Box or a Row with an OnTap, which every renderer draws as inert scenery and every screen reader announces as text. A core.Button needs none of this; it is already a \<button> on the web and a real control on both natives.

RoleLink is the other half of that pair, and the distinction is not cosmetic: a button does something \*here\* and a link goes somewhere else. A reader deciding whether to follow a control needs to know which, and the framework has no node type that carries the difference — core.OpenURL is a callback like any other, so a row that dials a phone number and a row that files a form are the same tappable Box until one of them says otherwise. RoleHeading is the one value here with a second question attached: how deep the heading sits. That is Style.AccessibilityHeadingLevel, a separate int rather than a lettered set of constants — see its doc for why six more spellings of "heading" would cost both DOM renderers the mapping table this vocabulary exists to avoid.

RoleImg is for a node that is a \*picture\* — something whose meaning is carried by its arrangement rather than by any text inside it, and which therefore needs one text alternative standing in for the whole thing. comps.Compass is the case that asked for it: a rose read in tree order is "N W E S" whatever direction it is pointing, so the widget hides its parts and speaks once.

It was also, for a while, the only role that made such a label \*work at all\* on the web: ARIA forbids an accessible name on a generic element, so an AccessibilityLabel on a plain container was dropped by screen readers rather than announced. RoleGroup below is now the general answer to that, and the division between the two is what the node is rather than what it needs — an img stands in for its parts and should hide them, a group names them and leaves them readable. Reach for this one only when the picture reading is true.

A node with this role should hide its children, or the reader gets the alternative \*and\* the parts it was standing in for.

```go
const (
	RoleHeading Role = "heading"
	RoleButton  Role = "button"
	RoleLink    Role = "link"
	RoleImg     Role = "img"
)
```

The naming role: the least a container can be, and the only thing that makes an accessible name on one legal at all.

##### The silence it closes

Every layout node in this framework exports as a \<div> or a \<span>, and both tags carry the implicit ARIA role \`generic\`. ARIA prohibits an accessible name on \`generic\` — aria-label and aria-labelledby are listed under "roles which cannot be named" — and browsers enforce it by pruning the name from the accessibility tree. So:

	core.Box(core.AccessibilityLabel("Unread messages"), …)

wrote a correct-looking attribute that no screen reader on either web target announced, while VoiceOver and TalkBack read it out perfectly, because a SwiftUI accessibilityLabel and a Compose contentDescription are honoured on any node without asking what it is. Two targets silent, two fine — which is what let it ship: the two that work are the two a developer is most likely to be testing on.

RoleImg was the first door out and it is the wrong shape for most rows. It says the node is a \*picture\* whose parts should be hidden behind one alternative, which is true of comps.Compass and false of a list row, a disclosure header or a stat tile — all of which want their contents read as well as their name.

##### Why \`group\` and not one of the louder candidates

	region     also nameable, and a landmark. A reader adds every region to
	           the list it jumps between, so naming six rows would put six
	           entries in a screen's table of contents.
	button     claims a control, makes its children presentational (a heading
	           inside one stops being a heading), and is a foreign child of any
	           list around it — see comps.ListRow, which turned it down
	           for exactly that.
	group      "a set of user interface objects", nameable, not a landmark,
	           and with no required children and no presentational-children
	           rule. It says these things belong together and this is what they
	           are called, and nothing else.

That "nothing else" is the whole recommendation. A role is a claim, and the structural rule above says a claim a container cannot keep is worse than no role at all; \`group\` is the one value in this vocabulary that promises nothing about what it holds, so it can be given to a container nobody has looked inside.

##### It is also a fallback, not only a constant

Because the silence is a framework bug rather than an author's mistake, the two web exporters supply this role themselves: a node with an accessible name, no role of its own, and a generic tag is written role="group" so the name is heard. An author who says anything more specific wins — the fallback only ever fills an empty slot. See accessibilityAttrs in htmlout/export.go and applyAccessibility in wasm/grmob-runtime.js, which restate one rule.

The fallback cannot make anything worse, which is the argument for doing it silently. Before it, the name was invalid ARIA that was dropped; after it, the name is valid ARIA that is announced. The one thing it could disturb is a structural container's claim about its children — but a generic div inside a role="list" was never a \`listitem\` either, so a \`group\` there is the same foreign child it already was, one attribute louder.

##### Both natives leave it empty for the opposite of the usual reason

Nine of the roles here are inert on SwiftUI and Compose because those platforms have no way to say the thing. This one is inert because they have no need to: both already announce a label on any node, so the role that makes the label legal buys them nothing. Setting it therefore costs nothing anywhere and closes a two-target silence.

```go
const RoleGroup Role = "group"
```

The zero value. A node that never sets a role has none, which is what every node had before this type existed: the renderers emit no attribute, add no trait and set no semantics.

```go
const RoleNone Role = ""
```

The one valued role: a control that is somewhere between two ends.

It is the only value in this vocabulary that reads Style.AccessibilityValue, and that pairing is ARIA's own scoping rather than a shortlist. aria-valuenow and its two bounds are defined for meter, progressbar, scrollbar, slider, spinbutton and a focusable separator; of those, this is the only one core has a role for, and the absences are all the same absence — no widget here is a meter, a scrollbar or a spinbutton, and core.Slider is a node type that exports as \<input type="range">, which carries the whole range natively and would have a second, contradicting claim written onto it by an ARIA one.

##### What it closes

comps.ProgressBar had no way to say it was a progress bar or how far along it was, so it said both into its accessible \*name\*: "Upload, 45 percent". That is the move Chip's ", selected" suffix was deleted for — a name is meant to be stable, so a bar ticking from 44 to 45 re-announced the whole thing, and nothing could act on a number buried in a string. With the role and the range, a reader announces the name once and the value as it moves.

##### A determinate bar and an indeterminate one are the same role

ARIA spells the difference by \*omitting\* aria-valuenow: a progressbar with a range is a bar with a known position, and one without is a spinner that is running. So an indeterminate bar is this role and a zero ValueRange, which falls out of the vocabulary rather than needing a value of its own.

##### Both natives

Compose has progressBarRangeInfo, which takes the numbers and announces a percentage TalkBack localizes — one of the better-mapped values here, and the reason grMobValue exists beside grMobRole. SwiftUI has no numeric equivalent at all; it takes the ValueRange's Text through accessibilityValue and nothing else, which is the honest half. See core.ValueRange.

```go
const RoleProgressBar Role = "progressbar"
```

The region a tab shows: the third member of the tab family, and the one that only makes sense with a reference beside it.

A tabpanel on its own says almost nothing — it is a section of a page — and what makes it announce as "tab panel, Sermons" is being pointed at. So this is the one role in the vocabulary that is not much use without Style.AccessibilityID and a tab's Style.AccessibilityControls, and the two arrived in the opposite order: the pointing existed for a session before the thing it points at could say what it was.

	// the strip
	core.Row(core.AccessibilityRole(core.RoleTabList),
	    Chip{Label: "Home", Style: []core.StyleProp{
	        core.AccessibilityRole(core.RoleTab),
	        core.AccessibilitySelected(core.SelectedWhen(tab == "home")),
	        core.AccessibilityControls("app-panel"),
	    }},
	)
	// the region it switches
	core.Box(core.AccessibilityRole(core.RoleTabPanel),
	    core.AccessibilityID("app-panel"), core.AccessibilityLabel("Home"), …)

It makes no claim about its children, unlike the tablist half — a panel holds whatever a screen holds — so it is not subject to the structural rule above and can go on any container.

core.TabView writes it from the node type and needs no author to. See "Roles a node type carries for itself" for why that is not a reason to leave the constant out, and for the marker that lets the runtime tell its own writes from an author's.

```go
const RoleTabPanel Role = "tabpanel"
```

#### func CompositeMemberRole

```go
func CompositeMemberRole(container Role) (member Role, composite bool)
```

CompositeMemberRole returns the role ARIA gives the members of a composite container, or "" for a composite whose members ARIA does not name.

	listbox      option
	radiogroup   radio
	tablist      tab
	toolbar      ""      ARIA defines no `toolbaritem`

It is the second half of what KeyboardComposites states — that list says which containers have a keyboard, this says how each one recognises the things the arrows move between — and it is here for the same reason the list is: a caller (and AuditTree) asks core what a role means, and the alternative is the fact living only in the WASM runtime's COMPOSITE\_MEMBERS, where no Go reader can consult it.

##### Two different empty answers, and why the second return exists

The empty answer is a real answer and not a "not found". A toolbar has a keyboard; what it does not have is a role that says "this is one of my members", which is exactly why the runtime has to supply a membership rule of its own (TappableContainerRoles is the Go half of that rule).

A RoleHeading has neither — no keyboard, and so no members to name — and it used to get the same "" back. The doc said callers separated the two by asking KeyboardComposites first, which is a contract a doc comment cannot enforce and which the one caller inside core got right by accident of never being handed a non-composite. CompositeWalkStopsAt reads this answer and returns "the walk stops here" for an empty one; handed a RoleHeading it produced a confident statement about a walk that does not exist.

So the two cases are separated where they are made:

	CompositeMemberRole(RoleListBox)  -> RoleOption, true
	CompositeMemberRole(RoleToolbar)  -> "",         true   a keyboard, no
	                                                        member role
	CompositeMemberRole(RoleHeading)  -> "",         false  no keyboard at all

\`composite\` is exactly membership of KeyboardComposites, and role\_control\_test.go holds the two to each other — a container added to that list and not here would report false for a role that has a keyboard, which is the same class of quiet wrong answer one table over.

<small>[core/role.go:746](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L746)</small>

#### func KeyboardComposites

```go
func KeyboardComposites() []Role
```

KeyboardComposites returns the container roles that get an ARIA keyboard pattern — a roving tabindex, arrow movement, Home and End — from the WASM runtime.

##### Why core states a fact about one target's runtime

It is not a list of what the runtime happens to implement; it is the list of roles for which \*setting a composite-keyboard style prop means anything at all\*, and that is a question a caller asks of core. AuditTree is the first reader: core.AccessibilitySelectionFollowsFocus is a statement about what a widget's keyboard does, and on a role with no keyboard it is a claim about nothing — which no exporter can notice, because knowing these four roles where an attribute is written would put the list in two places.

The runtime keeps the same four in two tables split by a different question (whether ARIA names the members), and wasm/verify holds their union to this function. So a fourth pattern is one edit here and a failing check there, rather than a role that quietly gains a keyboard the audit still calls inert.

##### Why these four and not the rest of ARIA's patterns

\`listbox\`, \`radiogroup\` and \`tablist\` are the three ARIA structures that both name their members and own their children, so the runtime can find a container's members by role. A radio group differs from a listbox in one behaviour rather than in its walk: its arrows move the check itself, so the runtime follows focus inside one without being asked, and AccessibilitySelectionFollowsFocus on it states what it already does. \`toolbar\` names no member role — ARIA defines no \`toolbaritem\` — and is here anyway because the pattern is real and the runtime supplies the membership rule itself: a toolbar's controls are the natively focusable tags plus the containers that say they are controls.

\`menu\`, \`menubar\`, \`tree\`, \`treegrid\` and \`grid\` are the patterns ARIA describes that this framework refuses, each for a stated reason — aria/verify/refusals\_test.go holds every refusal to what the pattern actually requires. \`list\` is deliberately absent and is the near miss worth naming: it is content rather than a control, and ARIA gives it no keyboard at all.

Container order matches Roles(); the members are not here, because being a member is a fact about a role's parent rather than about the role.

<small>[core/role.go:701](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L701)</small>

#### func Roles

```go
func Roles() []Role
```

Roles returns every declared Role except RoleNone, in declaration order.

RoleNone is excluded because it is the absence of a role rather than one of them: it is the field's zero value, no renderer has an arm for it, and a coverage check that demanded one would be asking each renderer to implement "unset". Everything downstream that iterates roles — the native dispatch pins, the DOM export test — wants the twenty-seven that do something.

A fresh slice per call rather than a package-level var, which any importer could write to. Twenty-seven elements are cheaper to build than to defend.

Pinned to the const blocks above by role\_enum\_test.go, which reads this file's syntax tree: adding a constant without adding it here should fail \`go test ./...\` rather than silently shrink the set every renderer's coverage check rests on.

<small>[core/role.go:645](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L645)</small>

#### func TappableContainerRoles

```go
func TappableContainerRoles() []Role
```

TappableContainerRoles returns the roles whose whole purpose is to make an ordinary container announce itself as a control.

These are the two the "Content roles" block above argues for in as many words: a Box or a Row with an OnTap, which every renderer draws as inert scenery and every screen reader announces as text until one of these says otherwise. A core.Button needs neither — it is already a \<button> on the web and a real control on both natives.

##### Why it is a list and who reads it

The WASM runtime's toolbar keyboard needs it. A toolbar's members are named by no role (ARIA defines no \`toolbaritem\`), so the runtime has to be told what a control is, and its answer is two rules: a natively focusable tag, or a container carrying one of \*these\* roles together with an OnTap. That second rule was a pair of bare strings in the runtime pinned against a pair of constants hand-written in a test — three copies of one fact, none of which was the fact itself.

The fact is here now, and role\_control\_test.go is what makes it a property rather than a fourth copy: every role core declares is either in this list or in a table saying why it is not one, so a new role cannot be added without somebody deciding. That was the actual hole — a future RoleCheckbox would be a tappable container by exactly the argument above, and would silently not be a toolbar member.

##### Why not "every role a screen reader calls a widget"

Because the question is narrower than it looks: not "is this thing interactive" but "does putting this role on a plain container make it a control the browser should give a tab stop to". RoleOption and RoleTab are interactive and are \*not\* here — they are members of a composite, whose tab stop belongs to their container and not to them, and taking one as a toolbar's control would put a second keyboard on a widget that has one.

<small>[core/role.go:944](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L944)</small>

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

### type SelectedState

```go
type SelectedState string
```

SelectedState is whether a control is \*on\* — the tab that is showing, the filter chip that is applied, the calendar day that is chosen.

It is the state half of core.Role. A role says what a control is and the label says what it is called; neither can say that this one of five chips is the one in effect, and until this type existed nothing in the framework could. What a widget did instead was write the state into the name — comps.Chip appended ", selected" to its AccessibilityLabel — which announces once, in the wrong place (a name is meant to be stable, and a reader that re-announces the control after a tap says the whole altered name rather than the changed state), and which no platform can act on.

#### Why three values and not a bool

Because "off" and "not a thing that can be on" are different facts and a bool has one spelling for both.

	SelectedUnset   a Box, a heading, a row of text. The node makes no claim,
	                which is the state every node in every tree was in before
	                this field existed, so the zero value is a no-op.
	SelectedOn      the chip that is applied, the tab that is showing.
	SelectedOff     a control that *could* be on and is not.

The third is the one a bool loses, and losing it is not cosmetic. A tablist in which only the live tab carries a state is malformed: a reader counting "tab 2 of 5" needs all five to say something, and the four that stay quiet are announced as plain tabs while the fifth is announced as selected — a strip that reads as though it has one tab and four pieces of furniture. So a strip sets the state on every control it draws, and SelectedOff is a value with a job rather than a way of saying nothing.

#### Why the values are ARIA's own spellings

For the reason core.Role's are: the two DOM targets write the value into the attribute verbatim and need no mapping table, and the two natives map what they can — which is the same trade the vocabulary made and the same place it pays off. \`mixed\`, ARIA's third value for aria-pressed and aria-checked, is deliberately absent: it describes a control governing a partially-selected set, no widget here has one, and a value nobody can produce is a fourth arm on every renderer for nothing.

<small>[core/selected.go:43](https://github.com/rohanthewiz/grmob/blob/master/core/selected.go#L43)</small>

```go
const (
	// SelectedUnset is the zero value: this node says nothing about being on
	// or off. Every renderer writes no attribute, adds no trait and sets no
	// semantics property, which is what every node did before this type.
	SelectedUnset SelectedState = ""

	// SelectedOn — the control is on.
	SelectedOn SelectedState = "true"

	// SelectedOff — the control can be on and is not. See the type doc for
	// why this is not the same as SelectedUnset.
	SelectedOff SelectedState = "false"
)
```

#### func SelectedStates

```go
func SelectedStates() []SelectedState
```

SelectedStates returns both stated values, in declaration order.

SelectedUnset is excluded for the reason RoleNone is excluded from Roles(): it is the field's zero value rather than one of the states, no renderer has an arm for it, and a coverage check that demanded one would be asking each renderer to implement "unstated".

Pinned to the const block above by selected\_enum\_test.go, and consumed by the exporters' round-trip checks the way Roles() is.

<small>[core/selected.go:85](https://github.com/rohanthewiz/grmob/blob/master/core/selected.go#L85)</small>

#### func SelectedWhen

```go
func SelectedWhen(on bool) SelectedState
```

SelectedWhen turns a widget's plain bool into the stated pair.

Every widget that has one of these holds a \`Selected bool\` — the caller owns the selection and the widget renders it — so the conversion would otherwise be three lines at each of them, and the tempting two-line version (\`if on { … SelectedOn }\` and nothing else) is exactly the bug the type doc warns about: it leaves the unselected controls silent.

It is a function rather than a method so that the bool reads as the subject: core.AccessibilitySelected(core.SelectedWhen(c.Selected)).

<small>[core/selected.go:69](https://github.com/rohanthewiz/grmob/blob/master/core/selected.go#L69)</small>

### type SpacingScale

```go
type SpacingScale struct {
	XS, SM, MD, LG, XL int
}
```

<small>[core/theme.go:396](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L396)</small>

### type StackAlignment

```go
type StackAlignment string
```

StackAlignment is where one layer of a ZStack sits inside the stack.

Every layer of a ZStack is centred, which is the alignment contract the node type documents and the one arrangement a SwiftUI ZStack, a Compose Box and a single-cell CSS grid all agree on without argument. This is the opt-out, per layer:

	   top-start        top       top-end
	       start      (center)      end
	bottom-start     bottom    bottom-end

The centre of that grid is the zero value and has no spelling of its own — StackAlignCenter is "" — so a layer that says nothing is placed exactly as every layer was before this type existed.

#### Why a two-axis value rather than Style.AlignSelf

AlignSelf is CSS's flexbox align-self: one axis, and \*which\* axis depends on the container's flex direction. A layer needs both axes named at once, and a stack has no main axis for the other one to be the cross of. Reusing the field would also have given it two meanings dispatched on the parent's node type — a stack's child means one thing by it and a Row's child another — which is the shape core.Box and core.ZStack were split apart to stop.

It is also why AlignSelf could not simply be taught to the natives: it is honoured by the two DOM targets only, and a portable-looking prop that works on the web is precisely what core.ZStack exists instead of.

#### What each target does with it

	web       justify-self / align-self on the grid item, imposed by the
	          stack (see htmlout's imposed) rather than written by the layer,
	          because align-self means something else on a flex child
	iOS       a coordinate handed to GrMobStackLayout, a custom SwiftUI
	          Layout — SwiftUI has no per-child ZStack alignment
	Android   Modifier.align(Alignment.*) in the Box's scope

The three constructs each have a nine-value 2D placement vocabulary and they agree value for value, which is what makes this portable where a flexbox property would not have been.

#### The divergence this used to carry, and what closed it

SwiftUI's spelling was the odd one: a frame that fills the stack, with the layer placed inside it. That is SwiftUI's own idiom for a job it has no direct spelling for, and a filling frame is \*greedy\* — so on iOS a stack that stated no size of its own grew to whatever its parent offered as soon as one layer was aligned, where a Compose Box and a CSS grid track both stay the size of their largest child.

It was documented in four places and avoided by pinning the stack's box, which core.ZStack asks for anyway. What it was not was \*pinned\*: ios/verify type-checks and replays a transcript, and neither of those measures a size.

The frame is gone. The iOS renderer places a layer by coordinate through a custom SwiftUI Layout (GrMobStackLayout), so nothing is wrapped and nothing is greedy, and the container reports the largest child on each axis like the other three. The arithmetic lives in GrMobStack.swift as pure CoreGraphics — split out for the reason GrMobFlexSolver was — and ios/verify measures both halves of it: what the stack sizes to, and where each of the nine anchors puts a layer.

Pinning a stack's dimensions is still good advice, and for the reason it always had: "top-start" of a box with no size is wherever the largest layer happens to end. It is no longer the difference between two renderings.

<small>[core/stack_align.go:70](https://github.com/rohanthewiz/grmob/blob/master/core/stack_align.go#L70)</small>

```go
const (
	// StackAlignCenter is the zero value: centred on both axes, which is
	// core.ZStack's contract and what every layer did before this type. It has
	// no spelling so that an unset Style.StackAlign *is* it — the same reason
	// SelectedUnset and VariantDefault are the empty string.
	StackAlignCenter StackAlignment = ""

	StackAlignTopStart StackAlignment = "top-start"
	StackAlignTop      StackAlignment = "top"
	StackAlignTopEnd   StackAlignment = "top-end"

	// StackAlignStart and StackAlignEnd are the middle row: the named edge
	// horizontally, centred vertically. Spelled without a "center-" prefix to
	// match StackAlignTop and StackAlignBottom, which are the same shape one
	// axis over.
	StackAlignStart StackAlignment = "start"
	StackAlignEnd   StackAlignment = "end"

	StackAlignBottomStart StackAlignment = "bottom-start"
	StackAlignBottom      StackAlignment = "bottom"
	StackAlignBottomEnd   StackAlignment = "bottom-end"
)
```

#### func StackAlignments

```go
func StackAlignments() []StackAlignment
```

StackAlignments returns the eight stated placements, in declaration order.

StackAlignCenter is excluded for the reason SelectedStates() excludes SelectedUnset: it is the field's zero value, every node in every tree carries it, and no renderer has — or should have — an arm for "the default". A coverage check that demanded one would be asking each renderer to implement doing nothing.

Pinned to the const block above by stack\_align\_enum\_test.go, and consumed by mobile/verify's native coverage checks and by htmlout's placement table, so a census that quietly stopped listing a value would quietly stop requiring an arm for it on three renderers at once.

<small>[core/stack_align.go:107](https://github.com/rohanthewiz/grmob/blob/master/core/stack_align.go#L107)</small>

### type State

```go
type State[T any] struct {
	// contains filtered or unexported fields
}
```

<small>[core/context.go:178](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L178)</small>

#### func NewState

```go
func NewState[T any](ctx *Context, initial T) State[T]
```

NewState allocates (or on re-render, re-binds) the hook slot at the current cursor position and returns typed accessors for it.

Slot access is guarded by ctx.lock because reads and writes come from different goroutines: renders run on the manager/pump goroutine (or a native event thread), while Set may be called from timers, network handlers, or any goroutine the app spawns. Render passes themselves are serialized by render.Manager, so the lock's job is only to make individual slot accesses atomic against concurrent Sets — a Set landing mid-render yields a tree mixing old and new values for one pass, which is benign: the Set also nudges the pump, so a follow-up pass renders the settled state.

<small>[core/context.go:273](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L273)</small>

#### func (*State) Get

```go
func (s *State[T]) Get() T
```

<small>[core/context.go:183](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L183)</small>

#### func (*State) Set

```go
func (s *State[T]) Set(val T)
```

<small>[core/context.go:187](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L187)</small>

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
	Rotate float64 `json:",omitzero"`

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
	//	option
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
	// container and carries a disclosure. comps.Accordion is what happens
	// when a widget takes that seriously — its header row is a button inside a
	// heading, which is ARIA's own accordion shape, rather than a named div
	// with a state a browser throws away.
	//
	// # The near miss: a control that opens a *dialog* is not expanded
	//
	// aria-expanded says the content is here, in the page, and can be shown or
	// hidden. A trigger that opens a modal is a different relationship —
	// ARIA spells that aria-haspopup, which this vocabulary does not carry —
	// so comps.DatePicker's trigger, which looks exactly like a
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
	// move comps.Chip's ", selected" name suffix was deleted for. So the
	// key crosses the bridge, is deliberately not parsed, and the note in
	// GrMobStyle.swift says which property it is turning down.
	AccessibilityExpanded ExpandedState `json:",omitzero"`

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
	//	aria-expanded     button, link, listbox, row, columnheader, tab
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

<small>[core/style.go:1248](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1248)</small>

#### func (Style) With

```go
func (s Style) With(other Style) Style
```

<small>[core/style_props.go:559](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L559)</small>

### type StyleProp

```go
type StyleProp interface {
	Apply(*Style)
}
```

<small>[core/style.go:738](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L738)</small>

#### func AccessibilityControls

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

<small>[core/style_props.go:513](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L513)</small>

#### func AccessibilityExpanded

```go
func AccessibilityExpanded(state ExpandedState) StyleProp
```

AccessibilityExpanded says whether this disclosure is open — the accordion section showing its body, the twisty that has been turned.

	core.Button(title, toggle,
		core.AccessibilityExpanded(core.ExpandedWhen(open.Get())),
	)

Use core.ExpandedWhen to convert the bool the widget already holds. Setting only the open case and leaving the shut one unset is the mistake the three-valued type exists to prevent: a closed disclosure that says nothing is announced as an ordinary button, and "collapsed" is the whole of what invites the press.

Paired with a role that can carry it, as a level and a selection both are — and \*not\* the same list a selection takes. aria-expanded is defined for button, link, listbox, row and columnheader among the roles this framework carries, which drops option and adds link and listbox. A core.Button needs no role of its own, the node type being one; anything else is dropped by both web targets. See Style.AccessibilityExpanded for the full table, for the dialog-shaped near miss it deliberately does not cover, and for why one native maps this and the other cannot.

<small>[core/style_props.go:440](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L440)</small>

#### func AccessibilityHeadingLevel

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

<small>[core/style_props.go:351](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L351)</small>

#### func AccessibilityHidden

```go
func AccessibilityHidden() StyleProp
```

AccessibilityHidden removes the element (and its subtree) from the accessibility tree — for decorative content a screen reader should skip.

<small>[core/style_props.go:521](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L521)</small>

#### func AccessibilityHint

```go
func AccessibilityHint(hint string) StyleProp
```

AccessibilityHint describes the \*result\* of activating the element ("Opens the article"). VoiceOver reads it natively; TalkBack has no hint slot, so the Android renderer appends it to the content description.

<small>[core/style_props.go:299](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L299)</small>

#### func AccessibilityID

```go
func AccessibilityID(id string) StyleProp
```

AccessibilityID gives this element a document-global name that another element can point at with AccessibilityControls. It becomes the \`id\` attribute on both web targets and is deliberately unread on both natives.

	core.Box(core.AccessibilityID("app-panel"), core.AccessibilityLabel("Home"), …)

Uniqueness is the caller's, as it is in hand-written HTML, and the "grmob-" prefix is reserved for core.TabView's own wiring. See Style.AccessibilityID for the whole argument — including why this and AccessibilityControls are the only two IDREF props in the vocabulary.

<small>[core/style_props.go:488](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L488)</small>

#### func AccessibilityLabel

```go
func AccessibilityLabel(label string) StyleProp
```

AccessibilityLabel gives screen readers a name for the element (TalkBack contentDescription, VoiceOver label). Set it on anything non-textual a user can perceive or activate — images, icon buttons, tappable rows.

<small>[core/style_props.go:290](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L290)</small>

#### func AccessibilityNestingLevel

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

<small>[core/style_props.go:388](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L388)</small>

#### func AccessibilityRole

```go
func AccessibilityRole(role Role) StyleProp
```

AccessibilityRole says what the element is: a heading, a table cell, a search landmark, a tappable Box that is really a button.

	core.Box(core.AccessibilityRole(core.RoleHeading), core.Text("Sermons"))

It is the third question a screen reader asks, after the name (AccessibilityLabel) and the effect (AccessibilityHint), and the one nothing here could answer until it existed — every container exports as a \<div> and announces as text. See core.Role for the vocabulary and for which of the four renderers honors which value.

Roles are not synthesized from node type or from props: a Box with an OnTap is a button only if it says so. Guessing would mean a widget that wraps a tappable row in a tappable card announcing two nested buttons, and the widget is the only layer that knows which one is the control.

<small>[core/style_props.go:320](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L320)</small>

#### func AccessibilitySelected

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

<small>[core/style_props.go:413](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L413)</small>

#### func AccessibilitySelectionFollowsFocus

```go
func AccessibilitySelectionFollowsFocus() StyleProp
```

AccessibilitySelectionFollowsFocus makes a composite widget choose the member the arrow keys land on, by invoking that member's own OnTap.

Set on the container — the listbox or the tablist — not on the members. See Style.AccessibilitySelectionFollowsFocus for when it is right and when it is the wrong thing to ask for.

A no-arg flag rather than a bool, like AccessibilityHidden and unlike Disabled: a caller does not have this in a variable, and there is no case for forcing it back off — a widget that does not want it writes no prop.

<small>[core/style_props.go:537](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L537)</small>

#### func AccessibilityValue

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

<small>[core/style_props.go:472](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L472)</small>

#### func Align

```go
func Align(a Alignment) StyleProp
```

<small>[core/style_props.go:133](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L133)</small>

#### func AlignItemsProp

```go
func AlignItemsProp(a AlignItems) StyleProp
```

<small>[core/style_props.go:232](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L232)</small>

#### func AlignSelf

```go
func AlignSelf(value AlignItems) StyleProp
```

<small>[core/style_props.go:32](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L32)</small>

#### func Background

```go
func Background(w string) StyleProp
```

<small>[core/style_props.go:202](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L202)</small>

#### func BackgroundColor

```go
func BackgroundColor(hex string) StyleProp
```

<small>[core/style_props.go:128](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L128)</small>

#### func BorderColor

```go
func BorderColor(hex string) StyleProp
```

<small>[core/layout.go:328](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L328)</small>

#### func BorderRadius

```go
func BorderRadius(px float64) StyleProp
```

<small>[core/style_props.go:152](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L152)</small>

#### func BorderWidth

```go
func BorderWidth(px float64) StyleProp
```

<small>[core/layout.go:333](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L333)</small>

#### func Bottom

```go
func Bottom(v string) StyleProp
```

<small>[core/style_props.go:253](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L253)</small>

#### func ColumnGap

```go
func ColumnGap(px float64) StyleProp
```

<small>[core/style_props.go:52](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L52)</small>

#### func Disabled

```go
func Disabled(disabled bool) StyleProp
```

Disabled hands the node to the platform's own disabled state: it stops accepting taps, keystrokes and focus, and screen readers announce it as disabled (Compose \`enabled = false\`, SwiftUI \`.disabled(true)\`, the HTML \`disabled\` attribute). See Style.Disabled for the full contract.

It takes the value rather than being a no-arg flag (unlike AccessibilityHidden) because the caller almost always has a bool in hand — \`core.Disabled(sending.Get())\` — and because passing false is the only way to force a node back to enabled: UseStyle's "a zero value means unset" rule means a Style{Disabled: false} cannot clear a flag already on the target.

<small>[core/style_props.go:553](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L553)</small>

#### func Display

```go
func Display(mode DisplayMode) StyleProp
```

<small>[core/style_props.go:139](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L139)</small>

#### func FlexBasis

```go
func FlexBasis(value string) StyleProp
```

<small>[core/style_props.go:27](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L27)</small>

#### func FlexDir

```go
func FlexDir(dir FlexDirection) StyleProp
```

<small>[core/style_props.go:220](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L220)</small>

#### func FlexGrow

```go
func FlexGrow(value float64) StyleProp
```

<small>[core/style_props.go:5](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L5)</small>

#### func FlexShrink

```go
func FlexShrink(value float64) StyleProp
```

FlexShrink sets a flex item's shrink factor. Zero means "do not shrink", and it is stored as core.ShrinkNone — see that constant for why this one number cannot use the zero-means-unset convention every other number in Style does.

The mapping lives here rather than in Style.Merge because this is the only door into the field: a caller writes core.FlexShrink(0) and a renderer reads Style.ShrinkFactor(), and nothing in between has to know about the sentinel.

<small>[core/style_props.go:18](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L18)</small>

#### func FlexWrap

```go
func FlexWrap(enabled bool) StyleProp
```

<small>[core/style_props.go:37](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L37)</small>

#### func FontSize

```go
func FontSize(size float64) StyleProp
```

<small>[core/style_props.go:111](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L111)</small>

#### func FontWeight

```go
func FontWeight(weight Weight) StyleProp
```

<small>[core/style_props.go:176](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L176)</small>

#### func Gap

```go
func Gap(px float64) StyleProp
```

<small>[core/style_props.go:122](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L122)</small>

#### func Height

```go
func Height(w string) StyleProp
```

<small>[core/style_props.go:192](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L192)</small>

#### func Horizontal

```go
func Horizontal() StyleProp
```

Horizontal turns a Scroll on its side: its children lay out in a row and the viewport pans across them instead of down them.

	core.Scroll(
	    core.Horizontal(),
	    core.Gap(8),
	    chips...,
	)

It is the chip strip, the tab strip and the card carousel — a short, wider-than-the-screen row that must not wrap and must not be clipped.

##### Why a StyleProp and not a Props flag

"Sideways" is spelled entirely in properties Style already carries, and CSS is the spelling both DOM targets already implement:

	flex-direction: row      <- Style.FlexDirection
	overflow: auto           <- Style.Overflow

So the two web renderers need no change at all to honour this — htmlout's styleValue lets an explicit FlexDirection override the node's stacking axis, and grmob-runtime.js's styleFromGrMob does the same. That matters more than it looks: the runtime's style path is \*total\* (an update-style patch carries the whole new Style and every managed property is reassigned), so a flag living in Props would have had to be mirrored onto the element and re-read on every style patch, or the first unrelated re-render would have quietly stood the strip back up on end. A Style field rides the patch channel that was built for exactly this.

The natives, which have no CSS to inherit, read Style.FlexDirection in their Scroll composite and nowhere else — the same narrow contract Style.FlexWrap already has, where only the Row composite reads it.

##### Overflow is supplied, not forced

A vertical Scroll emits no overflow on the web at all: the page scrolls, and the region is just a column. A horizontal one has no such fallback — with no overflow the row is simply clipped or squashed — so this supplies \`auto\` when the caller has not said otherwise. An explicit core.Overflow wins in either argument order, because the value is only defaulted when it is still empty.

(\`overflow: auto\` covers both axes rather than overflow-x alone. CSS forces the other axis to auto anyway the moment one of them is not \`visible\`, so the shorthand says what the browser was going to do; a strip whose children fit its height never shows the vertical bar.)

##### What it is not

It is not a horizontal List. core.List's laziness, its cross-axis stretch and its FlexGrow contract are all written for a vertical main axis on both natives, and nothing yet asks for a lazily-materialized carousel. A strip of chips or a handful of cards is short by construction, which is what Scroll is for.

<small>[core/layout.go:410](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L410)</small>

#### func Justify

```go
func Justify(j JustifyContent) StyleProp
```

<small>[core/style_props.go:226](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L226)</small>

#### func Left

```go
func Left(v string) StyleProp
```

<small>[core/style_props.go:259](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L259)</small>

#### func Margin

```go
func Margin(all int) StyleProp
```

<small>[core/style_props.go:212](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L212)</small>

#### func MarginBottom

```go
func MarginBottom(px int) StyleProp
```

MarginBottom sets the bottom margin alone. A zero clears.

This is the stacking prop: the gap under one item in a run that a parent's Gap does not describe, which is what examples/chat's message bubble wanted.

	core.Box(core.MarginBottom(8), bubble)

<small>[core/margin_sides.go:80](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L80)</small>

#### func MarginHorizontal

```go
func MarginHorizontal(px int) StyleProp
```

MarginHorizontal sets the left and right margins.

This is the inset prop: a rule that stops short of the screen edge, which is what comps.Separator's Inset wanted.

	core.Box(core.MarginHorizontal(16), rule)

It writes the explicit sides as well as the shorthand, for the reason given on PaddingHorizontal.

<small>[core/margin_sides.go:112](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L112)</small>

#### func MarginLeft

```go
func MarginLeft(px int) StyleProp
```

MarginLeft sets the left margin alone. A zero clears.

<small>[core/margin_sides.go:88](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L88)</small>

#### func MarginRight

```go
func MarginRight(px int) StyleProp
```

MarginRight sets the right margin alone. A zero clears.

<small>[core/margin_sides.go:96](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L96)</small>

#### func MarginTop

```go
func MarginTop(px int) StyleProp
```

MarginTop sets the top margin alone, leaving the other three as they were. A zero clears whatever a theme or an earlier prop supplied.

<small>[core/margin_sides.go:67](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L67)</small>

#### func MarginVertical

```go
func MarginVertical(px int) StyleProp
```

MarginVertical sets the top and bottom margins. Writes the explicit sides as well as the shorthand, for the reason given on PaddingHorizontal.

<small>[core/margin_sides.go:122](https://github.com/rohanthewiz/grmob/blob/master/core/margin_sides.go#L122)</small>

#### func MaxHeight

```go
func MaxHeight(w string) StyleProp
```

<small>[core/style_props.go:197](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L197)</small>

#### func MaxWidth

```go
func MaxWidth(w string) StyleProp
```

<small>[core/style_props.go:187](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L187)</small>

#### func MinHeight

```go
func MinHeight(value string) StyleProp
```

<small>[core/style_props.go:63](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L63)</small>

#### func MinWidth

```go
func MinWidth(value string) StyleProp
```

<small>[core/style_props.go:57](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L57)</small>

#### func Overflow

```go
func Overflow(value string) StyleProp
```

<small>[core/style_props.go:68](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L68)</small>

#### func Padding

```go
func Padding(all int) StyleProp
```

<small>[core/style_props.go:145](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L145)</small>

#### func PaddingBottom

```go
func PaddingBottom(px int) StyleProp
```

PaddingBottom sets the bottom inset alone. A zero clears.

<small>[core/padding_sides.go:143](https://github.com/rohanthewiz/grmob/blob/master/core/padding_sides.go#L143)</small>

#### func PaddingHorizontal

```go
func PaddingHorizontal(px int) StyleProp
```

PaddingHorizontal sets the left and right insets.

It writes the explicit Left/Right sides as well as the Horizontal shorthand. The renderers resolve a side as "the explicit value if non-zero, otherwise the axis shorthand" (see htmlout.EdgeCSS), so a prop that wrote only the shorthand could never override a side that was already set: a theme Column carries Left/Right 16, and PaddingHorizontal(0) after it used to leave the 16 in place — and PaddingHorizontal(24) used to render as 16. Writing the sides too gives this prop the same last-one-wins ordering every other StyleProp has, and a zero clears the theme value in all four renderers without any of them changing their resolution rule.

<small>[core/style.go:1097](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1097)</small>

#### func PaddingLeft

```go
func PaddingLeft(px int) StyleProp
```

PaddingLeft sets the left inset alone. A zero clears.

This is the indent prop: a nested row states its own depth without having to restate the three sides its theme container already got right.

	core.Row(core.PaddingLeft(16*depth), ...)

<small>[core/padding_sides.go:156](https://github.com/rohanthewiz/grmob/blob/master/core/padding_sides.go#L156)</small>

#### func PaddingRight

```go
func PaddingRight(px int) StyleProp
```

PaddingRight sets the right inset alone. A zero clears.

<small>[core/padding_sides.go:164](https://github.com/rohanthewiz/grmob/blob/master/core/padding_sides.go#L164)</small>

#### func PaddingTop

```go
func PaddingTop(px int) StyleProp
```

PaddingTop sets the top inset alone, leaving the other three as they were. A zero clears whatever the theme or an earlier prop supplied.

<small>[core/padding_sides.go:135](https://github.com/rohanthewiz/grmob/blob/master/core/padding_sides.go#L135)</small>

#### func PaddingVertical

```go
func PaddingVertical(px int) StyleProp
```

PaddingVertical sets the top and bottom insets. Writes the explicit sides as well as the shorthand, for the reason given on PaddingHorizontal.

<small>[core/style_props.go:279](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L279)</small>

#### func Responsive

```go
func Responsive(breakpoint string, style Style) StyleProp
```

Responsive registers a style variant under a named key in PseudoStates (":hover", ":focus", or a breakpoint name).

The entry is written into a fresh map rather than into whatever map the target already holds. A Style is copied by assignment throughout the framework — containerNode starts each node from a shallow copy of the theme's component Style — so the target's map may well be the theme's own. Writing into it in place would edit the theme for every render afterwards.

<small>[core/style_props.go:101](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L101)</small>

#### func Right

```go
func Right(v string) StyleProp
```

<small>[core/style_props.go:265](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L265)</small>

#### func Rotate

```go
func Rotate(deg float64) StyleProp
```

Rotate turns the node clockwise by deg degrees about its own centre, without disturbing the layout around it. See Style.Rotate for what each renderer maps it onto and why the angle is not normalised.

Unlike UseStyle, this setter can force zero: Rotate(0) writes the field, which is how a caller clears an angle a theme or role style supplied.

<small>[core/style_props.go:170](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L170)</small>

#### func RoundedShadowBox

```go
func RoundedShadowBox() StyleProp
```

<small>[core/style.go:1069](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L1069)</small>

#### func RowGap

```go
func RowGap(px float64) StyleProp
```

<small>[core/style_props.go:46](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L46)</small>

#### func Shadow

```go
func Shadow(elevation float64) StyleProp
```

<small>[core/style_props.go:158](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L158)</small>

#### func StackAlign

```go
func StackAlign(value StackAlignment) StyleProp
```

StackAlign places this node inside the core.ZStack it is a layer of.

	core.ZStack(
	    core.Width("160px"), core.Height("160px"),
	    rose,
	    core.Text("N", core.StackAlign(core.StackAlignTop)),
	)

It says nothing anywhere else. A stack is the only container that places its children in two dimensions at once, so this prop on a child of a Column, a Row or a Box is inert on every target — deliberately, and not merely as an accident of the implementations: on the web the value never reaches the element at all (the stack imposes the declaration, see htmlout's imposed), because align-self \*does\* mean something to a flex child and a layer prop that silently re-placed a row's children would be worse than one that did nothing.

Inert is the right behaviour and \*silent\* is not, so the tree walk says so: with debug mode on, core.AuditTree reports a placement no container will read as ConcernInertPlacement, naming the node path and the container that was going to place it. That is the only diagnostic any target produces, and the argument for putting it there rather than in a renderer is in placement\_audit.go.

<small>[core/stack_align.go:143](https://github.com/rohanthewiz/grmob/blob/master/core/stack_align.go#L143)</small>

#### func StickyHeader

```go
func StickyHeader() StyleProp
```

StickyHeader pins a List child to the top of the viewport while the rows it introduces scroll past underneath it, releasing it when the next sticky child arrives to take its place. It is the group band of an archive feed — the month over a run of sermons, the day over a run of transactions.

	core.List(
	    core.Keyed("group:2026-01", core.Row(core.StickyHeader(), monthLabel)),
	    core.Keyed("s1", row), core.Keyed("s2", row),
	)

comps.GroupedList{StickyHeaders: true} is the widget spelling; this is the primitive underneath it.

##### Why a StyleProp, and why no new field

Sticky positioning is not a new idea to this tree: Style.Position already carries PositionSticky, and both DOM targets already emit \`position\`, \`top\` and \`z-index\` verbatim, so the web half of this feature has always worked and needed no code. What was missing was the two natives, which declined Position outright ("no Compose analog at this layer") — true of \`fixed\` and \`absolute\`, and not true of \`sticky\`, which is precisely what a Compose stickyHeader and a SwiftUI pinned Section header are.

So the marker is the CSS one, and the renderers converge on it rather than on a private flag:

	web       position:sticky; top:0; z-index:1
	Compose   LazyColumn { stickyHeader { … } } for the marked rows
	SwiftUI   LazyVStack(pinnedViews: .sectionHeaders) + Section(header:)

Top and ZIndex are \*supplied\* rather than assigned — a caller's own values survive in either argument order — because both are load-bearing on the web and neither has a sensible zero: a sticky box with no offset never sticks, and one at the default stacking level is painted over by the rows that scroll under it.

##### Where it does something

A List child. Both natives implement pinning inside their lazy container and have nowhere to put it otherwise, so a Column child carrying this is sticky in a browser and inert on a phone. That asymmetry is why the name says List's word for the thing ("header") rather than CSS's word for the mechanism.

<small>[core/list.go:69](https://github.com/rohanthewiz/grmob/blob/master/core/list.go#L69)</small>

#### func TextColor

```go
func TextColor(hex string) StyleProp
```

<small>[core/style_props.go:117](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L117)</small>

#### func Transition

```go
func Transition(durationMs int, easing Easing) StyleProp
```

Transition declares that changes to this node's animatable properties — background color, size, padding, list placement — should animate over the given duration instead of snapping. This is the "declare in Go, drive natively" model: Go only ships the declaration in the style; each frame of the animation is produced by the platform's animation system (Compose, SwiftUI, CSS transitions), never by patches over the bridge.

The canonical serialized form is "\<ms>ms \<easing>" (e.g. "250ms ease-in-out"), which the native parsers read; they also tolerate the CSS longhand ("all 0.3s ease") for styles written by hand.

<small>[core/animation.go:32](https://github.com/rohanthewiz/grmob/blob/master/core/animation.go#L32)</small>

#### func UseStyle

```go
func UseStyle(s Style) StyleProp
```

UseStyle turns a whole Style value into a StyleProp, so a caller can pass a named visual role ("the card surface", "the theme's Body typography") in one argument instead of unpacking it into a dozen individual props.

The merge rule is "a set field wins, an unset field is ignored": every field of s that holds a non-zero value overwrites the target's, and every field left at its zero value leaves the target's alone. That is what makes UseStyle composable — layering role styles onto a theme's component defaults only ever adds, never blanks out what the theme supplied.

The rule's one unavoidable edge is that a zero value is indistinguishable from "not set", so UseStyle cannot \*clear\* a field the target already has: Style{AccessibilityHidden: false} does not un-hide an element, and Style{FontSize: 0} does not reset a font size. Use the individual StyleProp setters (AccessibilityHidden(), FontSize(0)) when the intent is to force a value rather than to layer one.

This merges every field of Style. It previously covered only fourteen of them, which meant Width, Height, the whole flex group, and the accessibility fields were silently dropped — a style value carrying them applied cleanly and did nothing. Any field added to Style must be added here too; TestUseStyleMergesEveryField walks the struct reflectively and fails if one is missed.

<small>[core/style.go:770](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L770)</small>

#### func WhiteSpace

```go
func WhiteSpace(value string) StyleProp
```

WhiteSpace controls how runs of spaces and line breaks in a Text are treated: "nowrap" keeps a line whole and lets the container scroll or clip it, "pre" additionally preserves the literal spacing, "normal" is the default wrapping behavior.

Style.WhiteSpace has been carried to both web targets since it was added, but nothing could set it — this is the missing constructor, not a new capability. It matters most for text whose columns mean something (code listings, tabular output), where wrapping restarts the continuation at column zero and makes the indentation actively misleading.

It is a no-op on the natives, which have no equivalent knob: SwiftUI and Compose wrap by line-break policy rather than by a CSS-style property.

<small>[core/style_props.go:87](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L87)</small>

#### func Width

```go
func Width(w string) StyleProp
```

<small>[core/style_props.go:182](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L182)</small>

#### func ZIndex

```go
func ZIndex(v int) StyleProp
```

<small>[core/style_props.go:271](https://github.com/rohanthewiz/grmob/blob/master/core/style_props.go#L271)</small>

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

### type Theme

```go
type Theme struct {
	Colors     ColorPalette
	Typography Typography
	Spacing    SpacingScale
	Components ComponentDefaults
}
```

<small>[core/theme.go:5](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L5)</small>

### type ToastConfig

```go
type ToastConfig struct {
	Duration int // ms
	Style    *Style
}
```

<small>[core/toast.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/toast.go#L7)</small>

### type ToastOpt

```go
type ToastOpt interface {
	Apply(*ToastConfig)
}
```

<small>[core/toast.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/toast.go#L3)</small>

#### func Duration

```go
func Duration(ms int) ToastOpt
```

<small>[core/toast.go:31](https://github.com/rohanthewiz/grmob/blob/master/core/toast.go#L31)</small>

#### func UseToastStyle

```go
func UseToastStyle(s Style) ToastOpt
```

<small>[core/toast.go:37](https://github.com/rohanthewiz/grmob/blob/master/core/toast.go#L37)</small>

### type Typography

```go
type Typography struct {
	Title    Style
	Subtitle Style
	Body     Style
	Caption  Style
}
```

<small>[core/theme.go:389](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L389)</small>

### type ValueRange

```go
type ValueRange struct {
	// Now is the current position, as ARIA's aria-valuenow. Empty means
	// unstated, which for a progressbar is ARIA's own spelling of
	// "indeterminate" — a bar that is running with no idea how far.
	Now string `json:",omitzero"`

	// Min and Max are the ends of the range, aria-valuemin and aria-valuemax.
	// Empty on both means ARIA's defaults, which are 0 and 100 — so a bare
	// Now reads as a percentage, which is what a progress fraction wants and
	// is why ValueOf's two-argument sibling would have been a trap: "3" with
	// no range announces as 3%, not as step 3.
	Min string `json:",omitzero"`
	Max string `json:",omitzero"`

	// Text replaces the number in the announcement when the digits are not
	// what a listener wants to hear — "3 of 5", "medium", "£12.50". ARIA says
	// a reader announces this *instead of* Now, so a Text that disagrees with
	// the number is the version the user gets.
	Text string `json:",omitzero"`
}
```

ValueRange is where a valued control sits inside its range — the fraction an upload has finished, the step a wizard is on.

It is the fourth of core's accessibility state vocabularies, after SelectedState (is this control on), ExpandedState (is this disclosure open) and the two level ints (how deep does this sit). Those three answer yes/no or a single integer; this one answers "how far along, out of what", which is three numbers and cannot be one field.

#### What asked for it

comps.ProgressBar, which had no way to say any of it. Its accessible name was built as "Upload, 45 percent" — the value spelled into the \*name\* channel, which is the exact move comps.Chip's ", selected" suffix was deleted for. A name is meant to be stable: a reader that re-announces a control says the whole altered name rather than the changed part, so a bar ticking from 44 to 45 re-announced "Upload, 45 percent" instead of "45 percent", and no platform could act on the number because no platform could find it.

#### Why the numbers are strings

For the reason Role's values and SelectedState's are ARIA's own spellings: the two DOM targets write them into the attribute verbatim and need no mapping table. The Style struct already carries numbers this way wherever zero is a legal value — Width, Height and MaxWidth are all strings — and that is the deciding reason here rather than a stylistic one:

	a float64 Now of 0 is a bar at the start of an upload, and it is also the
	zero value of the field. Style merges on "non-zero wins", so a stated 0
	would be indistinguishable from an unstated one and would be dropped by
	every merge in the chain.

SelectedState solved the same problem the same way — SelectedOff is the stated string "false" and SelectedUnset is "" — and ValueOf is the constructor that keeps a caller from having to think about it.

#### Why it is one field and not four

Style's two level ints merge independently \*on purpose\*: they answer disjoint questions, and dropping one because the other was set would make the result depend on which Style in the chain happened to name the role. This is the opposite case. Now, Min and Max are one fact in three parts — "45" means 45% under ARIA's implicit 0..100 and means nothing at all without knowing whether the range is 0..100 or 1..5 — so two Styles each merging half a range would produce a claim neither of them made. Merging as a unit is what makes that unwritable: a stated range replaces a stated range whole.

#### Text is the value channel, and it is not only for a range

Text becomes aria-valuetext on the web, Compose's stateDescription and SwiftUI's accessibilityValue. Those last two are honoured on any node, which makes this the value channel GrMobStyle.swift's AccessibilityExpanded note says the framework does not have. The difference that makes it safe now is whose words they are: a renderer emitting the literal "expanded" would be inventing English for every app in every locale, where this string is the app's own — the same line AccessibilityLabel and AccessibilityHint sit on.

The web is stricter than the natives here, exactly as it is for a selection: ARIA scopes aria-valuetext to the range roles, so a Text on an unroled container is dropped by a browser and announced by both phones. Each platform says the truest thing it can.

#### Why every field is \`omitzero\`

The same rule Style and EdgeInsets carry, and here it is the rule rather than the bytes: an empty field on this struct already means "unstated" by construction — that is the whole reason the three numbers are strings, per the argument above — and all four readers turn a missing key back into the empty string (optString, \`as? String ?? ""\`, \`v.Now || ""\`, and htmlout, which reads the Go value and never sees JSON). So presence carries nothing and the tags cost nothing.

The saving on the tutorial's contents screen is ten bytes, because one node on it states a range. That is not the point. The point is that "every struct that crosses this bridge omits its zeros" is now true without exception, which is what TestEveryWireFieldOmitsZero pins — an untagged field added to a wire struct is the failure mode the Style tags were worth 370KB catching late.

<small>[core/value.go:87](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L87)</small>

#### func ValueOf

```go
func ValueOf(now, min, max float64) ValueRange
```

ValueOf states a range from the three numbers a caller is holding.

The formatting is 'f' with the shortest round-tripping precision rather than %g, and that is not cosmetic: %g switches to scientific notation past six digits, so a byte counter would emit aria-valuenow="1.048576e+06" — which is not a number ARIA accepts and which no reader announces. -1 precision is what keeps a whole value short ("45", not "45.000000").

<small>[core/value.go:115](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L115)</small>

#### func (ValueRange) Progress

```go
func (v ValueRange) Progress() Progress
```

Progress resolves the three numbers into the one claim they make.

The parse is deliberately strict about what counts as a number: an empty string is unstated, and so is anything that does not parse — including the non-finite spellings ("NaN", "Inf") that Go's and Kotlin's parsers both accept and that no range property on any platform can hold. A bar whose position failed to parse is a bar with no position, which is a state ARIA already has a meaning for.

<small>[core/value.go:205](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L205)</small>

#### func (ValueRange) Stated

```go
func (v ValueRange) Stated() bool
```

Stated reports whether this range says anything at all. The zero value says nothing, which is what every node in every tree carried before this type existed, and is the condition Style.Merge and both web exporters test.

<small>[core/value.go:298](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L298)</small>

#### func (ValueRange) Unparsed

```go
func (v ValueRange) Unparsed() []string
```

Unparsed names the stated numeric fields that are not numbers this vocabulary can use, in the order the type declares them.

##### Why the reading alone is not enough to report one

Progress answers what the three strings amount to, and it answers it in ARIA's own terms: a position that does not parse is a bar with no position, which is a state ARIA already has a meaning for, and a bound that does not parse falls back to ARIA's default. Both are the right resolution and neither is what the author wrote — and from the outside the two cases are indistinguishable from the ones where nothing was stated at all.

	ValueRange{Now: "half", Min: "0", Max: "100"}   reads as indeterminate
	ValueRange{Min: "0", Max: "100"}                reads as indeterminate

The first is a mistake and the second is a bar that is genuinely running with no idea how far. So the fields are named separately from the reading, and core.AuditTree is what reports the first without reporting the second.

##### And it is not a harmless mistake

The three targets that parse these strings do not agree about a string that is not a number. Compose and this package read it as absent; a browser reads aria-valuenow="half" as 0 and pins the bar at the start of its range, and reads aria-valuemax="lots" as 0 and then clamps the position down to it. So an unparseable number is not "no number" — it is a different wrong answer on each platform, with nothing anywhere saying so. wasm/verify's browser pass holds Chrome to that divergence case by case.

Text has no part in it: it is words by design, and any string is a legal one.

<small>[core/value.go:266](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L266)</small>

#### func (ValueRange) WithText

```go
func (v ValueRange) WithText(text string) ValueRange
```

WithText adds the spoken form to a range, for a control whose number is not what a listener wants to hear.

	core.ValueOf(3, 1, 5).WithText("step 3 of 5")

A method rather than a fourth argument to ValueOf, because most ranges do not want one: a percentage announces perfectly well as a percentage, in whatever language the reader is set to, and supplying an English string would take that localization away.

<small>[core/value.go:132](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L132)</small>

### type View

```go
type View interface {
	Render(ctx *Context) *Node
}
```

<small>[core/view.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/view.go#L3)</small>

#### func Box

```go
func Box(stylePropsAndChildren ...PropsAndChildren) View
```

Box is the unopinionated container: a Column with no theme base. It stacks its children vertically, honors Gap/JustifyContent/AlignItems on the same axes a Column does, and reads the Align cross-axis fallback the same way — the only difference is that no theme style arrives with it, so nothing insets or paints the box but the caller.

It is not an overlay, on any target. Both natives used to draw it as one (a Compose Box, a SwiftUI ZStack, each pinned to the top-start corner) while the DOM targets stacked its children, so a Box with two children rendered two different pictures. mobile/verify's TestNativeContainersStackTheirChildrenAndDoNotOverlay pins the agreement.

ZStack, below, is the container that does overlay — and it exists because this one stopped. The two are the same argument from both ends: one shape per node type, stated once, rather than a container whose meaning depended on which renderer was reading it.

<small>[core/layout.go:235](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L235)</small>

#### func Button

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

#### func ButtonWithEvent

```go
func ButtonWithEvent(label string, event string, handler func(), props ...PropsAndChildren) View
```

ButtonWithEvent is Button with the event name chosen by the caller, for the gestures that have no dedicated builder. It is largely superseded by the widening above — core.Button(label, fn, core.On("LongPress", g)) says the same thing and keeps the click — but it stays because it is the only way to build a button whose \*only\* wiring is a non-click event.

<small>[core/button.go:58](https://github.com/rohanthewiz/grmob/blob/master/core/button.go#L58)</small>

#### func Cached

```go
func Cached(view View) View
```

Cached returns a View that renders view on first use and replays the same \*Node on every later pass. See the type comment for the constraints on what may be cached.

<small>[core/cached.go:58](https://github.com/rohanthewiz/grmob/blob/master/core/cached.go#L58)</small>

#### func CameraView

```go
func CameraView(props ...CameraProp) View
```

<small>[core/camera.go:17](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L17)</small>

#### func Card

```go
func Card(stylePropsAndChildren ...PropsAndChildren) View
```

<small>[core/layout.go:132](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L132)</small>

#### func Checkbox

```go
func Checkbox(checked bool, onToggle func(bool), props ...PropsAndChildren) View
```

<small>[core/input.go:50](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L50)</small>

#### func CodeEditor

```go
func CodeEditor(value string, onChange func(string), rows []GridRow, props ...PropsAndChildren) View
```

CodeEditor is an editable monospace buffer with syntax colour, a line-number gutter and the keyboard behaviour a programmer's editor has.

	core.CodeEditor(state.Src, onChange, highlight.Go().Rows(state.Src, highlight.Darcula),
	    core.LineNumbers(),
	    core.TabSize(4),
	    core.OnSelectionChange(func(start, end int) { ... }),
	    core.Height("240px"),
	)

comps.CodeEditor is the widget over it — it runs the highlighter, wires a toolbar and picks a scheme from the theme — and is what application code should reach for. This is the primitive it is built on.

##### Why a node type rather than a composition

The obvious pure-Go construction is a transparent core.TextArea in a ZStack over a core.TextGrid: Go colours the grid, the user types into the invisible field above it, and the two line up. They do not line up, and cannot: core.Style has no font-family, so the TextArea is drawn in the platform's proportional UI face while the grid is monospace, and no amount of styling from outside can pitch-match a SwiftUI TextEditor or a Compose BasicTextField to a separate text view. The overlay has to be built by something that owns \*both\* elements, which is the renderer. That is the whole argument for this being a node type, and it is the same argument TextGrid makes one step earlier.

##### The three rules every host implements

 1. Echo guard, unchanged from TextArea. \`value\` from Go is applied only when it is not an echo of the host's own last onChange. The buffer is the host's while focused and Go's otherwise — see GrMobTextField's pendingEchoes in either native renderer for the bookkeeping, which this node reuses verbatim rather than restating.

 2. Decoration is advisory and per line. The rows are GridRow children, exactly as core.TextGrid builds them, and a host applies row N's styling only if that row's concatenated text equals the host's current line N. A line that disagrees — Go is a keystroke behind, which it is for a few milliseconds after every keypress — is drawn in plain ink until the next patch. Never the other way round: decoration never rewrites the buffer, so a lexer that is wrong can make the screen ugly and can never make it lose text.

 3. Commands are epoch-stamped props. See core/editor.go.

##### The behaviour the contract tests pin

Monospace, no wrapping, horizontal scroll. Tab inserts an indent rather than moving focus. Enter copies the previous line's leading white space. Autocorrect, autocapitalization and smart quotes are off — every one of them corrupts source. A readOnly buffer is still selectable and still shows a caret, because a code block the user cannot copy out of is a screenshot.

##### Focus commands

core.Focus and core.DismissKeyboard reach an editor: CodeEditor is in focusableLeafTypes, so every command stamps it like any other text control and a background tap puts its keyboard away.

The one thing worth knowing is where the command lands. This node is a \*box\* — a scroll container holding a gutter and a buffer — where an Input is the control itself, so each renderer resolves the stamp to the buffer rather than applying it where it arrived. The gutter is chrome and never takes the caret. htmlout applies it nowhere at all: its editor is a read-only snapshot with no editable element to autofocus.

##### Known gaps in v1

Nothing that needs a caret is exercised by any harness here — the IME composing region, a hardware Tab on iPad, and paste from another app are arguments rather than tests.

##### value and rows are two facts about one buffer, and they can disagree

value is the text; rows are a \*decoration of that text\*, computed in Go from that same text. They always agree at the moment Go builds them, and are allowed to disagree with the host mid-keystroke, which is what rule 2 is about. A caller that computes rows from something other than value has not broken anything — the rows simply never match and the buffer is drawn plain.

<small>[core/codeeditor.go:83](https://github.com/rohanthewiz/grmob/blob/master/core/codeeditor.go#L83)</small>

#### func Column

```go
func Column(stylePropsAndChildren ...PropsAndChildren) View
```

<small>[core/layout.go:213](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L213)</small>

#### func DefaultErrorFallback

```go
func DefaultErrorFallback(err error) View
```

DefaultErrorFallback is the fallback ErrorBoundary uses when given nil: a bordered card in the theme's Error role.

The detail line is gated on debug mode on purpose. A panic message is developer-facing text — "runtime error: index out of range \[7] with length 3" tells a user nothing and quietly leaks internals into a screenshot — so a release build shows only the generic line, while a debug build shows the message that identifies the bug. The full \*RenderError, stack included, is always available to a custom fallback either way.

<small>[core/error_boundary.go:208](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L208)</small>

#### func Divider

```go
func Divider(height int, color string) View
```

<small>[core/layout.go:321](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L321)</small>

#### func ErrorBoundary

```go
func ErrorBoundary(child View, fallback func(err error) View) View
```

ErrorBoundary renders child, and if child's render panics, renders fallback(err) in its place instead of letting the panic reach the render driver — where, on a native host, it would take the whole app down.

Pass a nil fallback to get DefaultErrorFallback.

	core.ErrorBoundary(
	    ProfilePanel(user),
	    func(err error) core.View {
	        log.Printf("profile panel failed: %v", err)
	        return core.Text("Profile unavailable")
	    },
	)

##### The fallback is also the notification hook

ErrorBoundary logs nothing itself. fallback is called on every pass in which the child fails, receives the full \*RenderError (stack included), and is the intended place to log, report, or degrade. Note "every pass": a component that panics deterministically panics again next frame, so a fallback that logs unconditionally will log at frame rate. Rate-limit, or log from a boundary placed high enough that failures are rare.

##### It does not latch

React's error boundaries stay in the fallback until explicitly reset, because there the failed subtree's instances are unrecoverable. Nothing of the sort is true here: the tree is rebuilt from scratch every pass, so a child that panicked on a stale slice index simply renders normally on the next pass once the state settles. Latching would turn a one-frame glitch into a permanent dead panel, so the boundary retries every pass and heals on its own. The cost is the repeated-panic case above, which is the right trade — a stuck fallback is worse than a noisy one.

##### What it repairs, and why it needs its own contexts

A panic partway through a render leaves two pieces of per-pass bookkeeping half-advanced, and both are positional, so leaving them where they fell would corrupt \*unrelated\* components rendered later in the same pass:

	hook slots      parent ctx.Cursor sits between the child's hooks, so
	                every later sibling reads the wrong slots — sibling
	                state visibly swaps
	callback IDs    the registry counters sit past the handlers the child
	                managed to register, so every later sibling's IDs shift
	                and taps land on the wrong handler

The hook half is solved structurally rather than by rollback: the boundary takes two child contexts (one for child, one for the fallback) and renders into those. A panic can then only strand a cursor inside the child's own context, and the boundary consumes exactly two parent slots whether the child succeeds, fails early, or fails late.

	parent ctx slots:   [ ... | childCtx | fallbackCtx | ... ]
	                             ^ panic strands the cursor in here only

The callback half is a genuine rollback: renderRecovered snapshots the registry counters before the child renders and rewinds them after a panic, so the boundary's ID footprint equals the fallback's footprint and does not depend on how far the failed render got.

##### Consequence: the child gets its own hook namespace

Because child renders into a child context, its hook slots and its ctx.Scope table are its own rather than the parent's. State is keyed by position within a context, so this is transparent for the child itself — but a component that reaches for ctx.Scope("x") expecting to share a scope with something \*outside\* the boundary will get a different scope. Shared app state (navigation, callbacks, theme, config) lives on pointers copied into every derived context and is unaffected.

<small>[core/error_boundary.go:146](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L146)</small>

#### func For

```go
func For[T any](items []T, render func(item T, index int) View) View
```

<small>[core/conditionals.go:10](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L10)</small>

#### func Fragment

```go
func Fragment(children ...View) View
```

<small>[core/layout.go:201](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L201)</small>

#### func If

```go
func If(condition bool, view View) View
```

<small>[core/conditionals.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L3)</small>

#### func IfElse

```go
func IfElse(condition bool, thenView View, elseView View) View
```

<small>[core/conditionals.go:23](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L23)</small>

#### func Image

```go
func Image(src string, styleProps ...StyleProp) View
```

Image renders a remote or bundled image at the default content mode (ContentModeFit). Use ImageWithMode to choose another.

<small>[core/image.go:92](https://github.com/rohanthewiz/grmob/blob/master/core/image.go#L92)</small>

#### func ImageWithMode

```go
func ImageWithMode(src string, mode ContentMode, styleProps ...StyleProp) View
```

ImageWithMode is Image plus an explicit ContentMode.

A separate builder rather than a variadic change to Image, matching InputWithSubmit: every existing Image call site keeps compiling and keeps its current rendering, and the mode stays a required, visible argument at the sites that care rather than an option buried in a style list.

<small>[core/image.go:102](https://github.com/rohanthewiz/grmob/blob/master/core/image.go#L102)</small>

#### func Input

```go
func Input(value string, placeholder string, onChange func(string), props ...PropsAndChildren) View
```

<small>[core/input.go:23](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L23)</small>

#### func InputPassword

```go
func InputPassword(value string, placeholder string, onChange func(string), props ...PropsAndChildren) View
```

<small>[core/input.go:59](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L59)</small>

#### func InputWithSubmit

```go
func InputWithSubmit(value string, placeholder string, onChange func(string), onSubmit func(), props ...PropsAndChildren) View
```

InputWithSubmit is Input plus a submit action: pressing the keyboard's return key (iOS) or IME done action (Android) dispatches onSubmit. The submit rides the existing void-callback channel — the renderers read the "onSubmit" prop and dispatch it exactly like a Button's onClick — so the bridge surface is unchanged. A separate builder rather than a variadic change to Input keeps every existing call site compiling untouched.

<small>[core/input.go:39](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L39)</small>

#### func Keyed

```go
func Keyed(key string, child View) View
```

<small>[core/view.go:13](https://github.com/rohanthewiz/grmob/blob/master/core/view.go#L13)</small>

#### func List

```go
func List(stylePropsAndChildren ...PropsAndChildren) View
```

List is the virtualized sibling of Column: a vertically scrolling container whose children are laid out lazily by the native renderer (Compose LazyColumn, SwiftUI LazyVStack), so a thousand-row feed composes only the rows on screen. Column + Scroll remains the right choice for short content; List is for long, data-driven collections.

Give every child a stable identity with Keyed(id, ...) — the native lazy containers use the key to keep row state (and recycled views) attached to the same data across insertions, removals, and reorders. Unkeyed children fall back to positional identity, which behaves like Column but loses row state on reorder.

It shares Column's theme base and the standard container argument contract: style props, behavior props (e.g. OnClick on the list surface), and child views in any order.

<small>[core/list.go:20](https://github.com/rohanthewiz/grmob/blob/master/core/list.go#L20)</small>

#### func MapView

```go
func MapView(region Region, props ...PropsAndChildren) View
```

MapView is a live map: the platform's own map widget, panned and zoomed by the user, with markers as child nodes.

	core.MapView(core.Region{Lat: 38.7223, Lng: -9.1393, Zoom: 14},
	    core.Width("100%"), core.Height("280px"),
	    core.ShowUserLocation(),
	    core.OnMarkerTap(func(id string) { open(id) }),
	    core.Marker("hall", 38.7223, -9.1393, "The hall"),
	    core.Marker("annex", 38.7251, -9.1402, "The annex"),
	)

Each host draws its platform's map:

	iOS       MapKit, which is free and needs no key
	Android   osmdroid over OpenStreetMap tiles — no key, and no dependency on
	          Play Services being present on the device
	Browser   Leaflet over OpenStreetMap tiles, loaded by the host page
	htmlout   a placeholder box, as CameraView is: a static snapshot has no
	          engine to run and no tiles to fetch

##### When to use comps.StaticMap instead

Almost always, if the question is "where is this". A static map is an image and a hand-off to the platform's maps app: no engine, no tile budget, no key, nothing to keep in step, and the directions the user actually wanted come from the app that has their home address in it.

This node is for the cases that need the map to be \*part of\* the screen: a set of markers to compare, a region the user explores, a position they pick by tapping. Those are interactions, and an image cannot have them.

##### Markers are children, not a prop

The same decision core.TextGrid makes about its rows, for the same reason. A marker set sent as one prop means every marker is re-read whenever any of them moves: the reconciler sees one changed value and the host rebuilds the annotation layer, which on every platform is a visible flicker and on two of them loses the selected callout.

As children they are ordinary nodes. The reconciler pairs them by key, emits an update-props patch for the one marker that moved, an add for the one that appeared and a remove for the one that left — and each host's annotation bookkeeping is the patch handling it already has.

Marker keys itself from its id, so a caller writing core.Marker in a loop gets stable identity without having to remember core.Keyed. See Marker.

##### The map is controlled, with the echo guard a drag needs

Region is Go's statement of where the map should be, and it is applied to the host widget \*only when it changes\*. It is not re-asserted on every patch.

That sounds like a detail and it is the whole usability of the node. A map is the one widget whose value the user changes continuously by touching it: if every render re-centred the host map on Go's Region, then an app that does not echo OnRegionChange back into its own state would snap the map back under the user's finger on the next unrelated re-render — and an app that does echo it would fight its own round trip, because the echo arrives a frame late and moves the map again.

So each host remembers the Region it last applied and compares: Go moving the map is an instruction, and Go merely re-rendering is not. It is the same compromise the text fields and the Slider make — the value shown is Go's except where the finger is the authority — and it has to be implemented the same way in all three live hosts, which mobile/verify and wasm/verify check.

An app that wants the map pinned to its own state does nothing special: it echoes OnRegionChange into state, and every Region it renders is one it chose. An app that wants "show me this place, then let the user wander" renders a constant Region, which is applied once.

##### Two memories, and the one thing that cannot be said

Each host keeps the Region \*Go\* last asked for and, separately, where the \*map\* last came to rest. They are the same value until somebody touches the map, and each direction reads the one that answers its own question — Go changing its mind is an instruction; the map already being there is not.

Folding those into one slot is the bug this contract exists to prevent, and it shipped in all three hosts: a pan wrote the user's Region into the slot the apply path reads, so Go's \*unchanged\* Region read as a change and the next patch to reach the map — a pin dropped, a marker moved, any unrelated re-render — snapped the map back. It survived a unit test because a fake map can be panned without firing the event a real one always fires, and it was found by opening a browser.

The consequence a caller can see is this: re-rendering the \*same\* Region is never a re-centre. An app that pans away and then wants the opening view back cannot get it by handing the same numbers over again, because from here that is indistinguishable from the unrelated re-render above. The remedy is the one this node already recommends — echo OnRegionChange into state, so the Region an app renders tracks where the map is and a "back to the start" button is a genuine change. There is deliberately no imperative recentre command; adding one is a host feature in three languages, and the echo costs one line.

##### Tiles are somebody else's bandwidth

Two of the three hosts draw OpenStreetMap tiles from the project's own servers, which have a usage policy: identify your app, do not bulk download, and expect to be blocked if you send a million tile requests a day. An app shipping this to a real user base should point its host at a tile provider it pays for. That decision lives in each host rather than in this node — it is a URL template in osmdroid's configuration and in Leaflet's layer — because it is a deployment fact rather than a property of the view.

<small>[core/mapview.go:114](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L114)</small>

#### func Marker

```go
func Marker(id string, lat, lng float64, title string, props ...PropsAndChildren) View
```

Marker is one pin on a MapView: a child node, keyed by its id.

	core.Marker("hall", 38.7223, -9.1393, "The hall")

##### It keys itself

The key is "marker:" + id, written here rather than left to the caller, which is a departure from how every other keyed child in this framework works — core.List's rows are the caller's to key, and forgetting is a documented mistake with a debug-mode concern behind it.

The difference is that a marker already carries its identity. The id is required, it is what OnMarkerTap reports, and there is no sensible second answer to "which marker is this" — so a caller writing core.Keyed around one would be restating the id, and a caller forgetting to would get the failure keys exist to prevent (a marker layer rebuilt on every change) in the one place it is most expensive.

An empty id is allowed and keys nothing, which is the honest answer for the single unnamed marker a "you are here" view draws: there is nothing to tell it apart from, and nothing will ever report a tap on it by name.

##### A marker is data, not a box

It carries no style and draws no element of its own. Every host reads its props and creates a native annotation; the node exists so the reconciler can address it. Both DOM renderers give it a hidden element, because a patch path is positional and a node with no element would put every later patch in the wrong place — the same reason htmlout and the WASM runtime disagree about Fragment (see htmlout's transparentTypes).

Title is the callout the platform shows when a marker is tapped, and may be empty. It is not an accessibility label: a map's annotations are announced by each platform's own map accessibility, which this framework does not reach into.

<small>[core/mapview.go:337](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L337)</small>

#### func Match

```go
func Match[T comparable](input T, cases ...MatchCase[T]) View
```

<small>[core/conditionals.go:67](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L67)</small>

#### func MatchBool

```go
func MatchBool(clauses ...WhenClause) View
```

<small>[core/conditionals.go:43](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L43)</small>

#### func Modal

```go
func Modal(props ...ModalProp) View
```

<small>[core/modal.go:14](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L14)</small>

#### func Navigator

```go
func Navigator(initial func(*Context) View) View
```

Navigator renders the top of the route stack, seeding the stack with initial the first time it renders. It emits no wrapper node of its own — the tree it returns is the route's tree — so a Navigator can sit anywhere a view can.

Each frame renders into its own scope of the host context, which has three consequences worth knowing:

  - Routes may use hooks freely. NewState in a pushed route claims slot 0 of that frame, not slot 0 of whatever screen is underneath it.
  - A route's state, and any background resource its hooks started, is discarded when its frame leaves the stack (Pop, Replace, Reset).
  - State that must outlive a frame belongs above the Navigator. Routes are closures, so the usual move is to capture the context the Navigator itself renders into and keep the state in a scope of that.

Note that Navigator does not call ctx.Reset(): cursors are restarted once per pass by the render driver, before the root render. A second, partial Reset from inside the tree would rewind the cursor of every context at or below this one mid-pass — harmless when the Navigator is the root view and silently corrupting when it is not, since siblings rendered before it have already consumed slots that the rewind hands out again.

<small>[core/navigation.go:161](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L161)</small>

#### func NumericInput

```go
func NumericInput(value int, onChange func(int), props ...PropsAndChildren) View
```

<small>[core/input.go:69](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L69)</small>

#### func RichTextEditor

```go
func RichTextEditor(doc richtext.Doc, onChange func(richtext.Doc), props ...PropsAndChildren) View
```

RichTextEditor is an editable formatted document: bold, italics, headings, lists, quotes, links.

	core.RichTextEditor(state.Doc, func(d richtext.Doc) { state.Doc = d },
	    core.Placeholder("Write something…"),
	    core.EditorTarget(ref),
	    core.OnRichSelectionChange(func(sel core.RichSelection) { bar.Set(sel) }),
	)

comps.RichTextEditor is the widget over it — it builds the toolbar and wires the link prompt — and is what application code should reach for.

##### The value is a document, and the document is Go's

richtext.Doc crosses the wire as JSON and each host maps it to and from its own text representation. That is the whole architectural decision here and the package doc for richtext carries the argument: an NSAttributedString, a Spannable and a contenteditable's innerHTML are three vocabularies, and an app that stored whichever one the user happened to type on would have a database its other two targets could not read.

##### The rules it shares with CodeEditor, and the one it does not

The echo guard is unchanged — the doc JSON is compared exactly as a TextArea's value is, so Go's echo of the host's own last onChange never resets the caret. Commands are epoch-stamped props, same mechanism, same adopt-on-first-sight rule (see core/editor.go).

The stale-line rule is \*not\* here and does not need to be. A CodeEditor has two facts about one buffer — the text and a decoration of it computed separately — which can disagree for a frame. Here the doc \*is\* the styled buffer: there is nothing to compare it against, because the formatting and the characters arrive together.

##### Known gaps in v1

core.Focus and core.DismissKeyboard reach an editor — the type is in focusableLeafTypes, exactly as CodeEditor is. On both phones the control is a classic text view hosted inside the declarative framework (a UITextView, an EditText), so neither renderer can hand the command to its platform's own focus system and each drives the responder directly; see core/focus.go.

Collaborative editing, images, tables and per-run fonts are non-goals; each is a driver away and none changes the design above.

<small>[core/richtext.go:54](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L54)</small>

#### func Row

```go
func Row(stylePropsAndChildren ...PropsAndChildren) View
```

<small>[core/layout.go:126](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L126)</small>

#### func SafeArea

```go
func SafeArea(stylePropsAndChildren ...PropsAndChildren) View
```

SafeArea insets its content from the system bars and the display cutout. It is the root of every screen (comps.Screen builds one) and takes the same mixed argument list as the other containers, so a style can land on the inset box itself.

The one style worth putting there is a background. The inset is padding on this node, so a background here paints under the status bar while the content stays clear of it — whereas a background on the content column stops at the inset and leaves the strip behind the bar in the window's own colour, which on a dark screen is a light band along the top. Each native renderer paints this node's background edge to edge (Compose orders the background before the inset padding; SwiftUI extends it with ignoresSafeArea); the DOM targets have no system bars and treat it as any other container. Padding and margin here are honoured too but rarely wanted, since they inset the whole screen a second time.

Like Scroll it has no theme base: the theme Column's screen padding would otherwise inset the content twice.

Below the inset it is a Column, on every target: children stack and, with no cross-axis alignment set, stretch to its width. Both natives used to draw it as an overlay (a Compose Box, a SwiftUI ZStack), which stacked two children on top of each other and let a lone one — a screen's whole content column, usually — hug its widest child instead of filling the screen.

<small>[core/layout.go:195](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L195)</small>

#### func SafeRender

```go
func SafeRender(child View) View
```

SafeRender is ErrorBoundary with the built-in fallback — the one-liner for wrapping a subtree you merely want to survive, with no opinion about what replaces it.

<small>[core/error_boundary.go:195](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L195)</small>

#### func Scroll

```go
func Scroll(stylePropsAndChildren ...PropsAndChildren) View
```

Scroll is a vertically scrolling region: its children are laid out at their natural height and the viewport pans over them.

It takes the standard container argument list — style props, behavior props and child views in any order — rather than the bare ...View it used to, because both native renderers have always applied a Scroll node's style (Compose boxModifier, SwiftUI grMobBox) and Go had no way to set one. The widening is source-compatible: a View is a PropsAndChildren, so every existing core.Scroll(child) call still compiles and, with no props supplied, still renders the same box.

Unlike Column and Row it has no theme base — like Box, it is the unopinionated container, and a scroll region that arrived with the theme Column's screen padding would inset every screen that wraps itself in one.

See KeyboardAware for the software-keyboard behavior.

<small>[core/layout.go:165](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L165)</small>

#### func Select

```go
func Select(value string, options []SelectOption, onChange func(string), props ...PropsAndChildren) View
```

Select is the picker: one value chosen from a fixed list.

	core.Select(form.Country, []core.SelectOption{
	    {Value: "us", Label: "United States"},
	    {Value: "pt", Label: "Portugal"},
	}, func(v string) { form.Country = v })

Controlled, like every other input here: the value shown is always the one Go passed, and a change goes up as an event. onChange carries the option's \*Value\*, never its label or its index — the index is the one identity that changes when the list is reordered, and a label is written to be read.

##### What each target draws

	web       a <select>, whose options are a prop rather than child nodes
	iOS       a Menu whose label is the chosen option's text
	Android   a Box anchored to a DropdownMenu, same shape

The natives are deliberately \*not\* built from a platform picker control (SwiftUI's .pickerStyle(.menu), Material's ExposedDropdownMenuBox). Both of those draw a frame of their own, and the whole rule this widget lands under is that the Go style owns the frame — see borderResetTypes in htmlout/tag.go, which the web half joins for the same reason. A control whose edge came from the platform on two targets and from the theme on two others is the divergence that rule exists to prevent.

##### It reads the theme's Input base

A picker is a field: it sits in a form beside text inputs, and a picker that did not match the fields around it would look like a mistake. Reading Components.Input is also how it inherits the frame those fields grew — the same move comps.DatePicker makes for the same reason, and the reason this widget needs no palette role of its own.

##### Options are a prop, not children

A \<select>'s options are elements, but they are not \*nodes\*: no patch is ever addressed to one, they carry no style, and they cannot hold a subtree. Sending them as children would put four renderers in the business of deciding which child is chrome, which is the complication core.TabView's tabs prop already avoids one node type over. The web renderer builds the \<option> elements from the prop; both natives read the same list.

##### Grouped and disabled options

SelectOption carries a Group, a Disabled and a GroupDisabled beside its two required fields; see the type. All three are drawn by every target — an \<optgroup>, a disabled \<option> and \<optgroup disabled> on the web; a Section and a disabled Button in the iOS menu; a heading item and a disabled item in the Android dropdown.

What a heading still cannot carry is an icon, and that is a decision rather than a gap: an \<optgroup>'s label is an attribute, so the web can hold text and nothing else. A heading with an icon on two targets and without one on the other two is the divergence this widget refuses everywhere else — the same argument that keeps it off the platform picker controls, one property down.

Neither reaches this function as anything but a map key, which is the point: the flattening below is the one place that knows what a SelectOption is, and the four renderers each read a list of flat string maps. A fifth field would land here and nowhere else.

What the renderers do \*not\* each decide is which options form which run. SelectMenuSections (select\_menu.go) takes the flattened list and answers that once; htmlout calls it, and the two natives carry transliterations that ios/verify checks against it.

<small>[core/input.go:280](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L280)</small>

#### func Slider

```go
func Slider(value, min, max float64, onChange func(float64), props ...PropsAndChildren) View
```

Slider is a horizontal value control: a thumb on a track, dragged to choose a number in \[min, max]. A seek bar, a volume, a brightness, a price ceiling.

	core.Slider(pos, 0, duration, func(v float64) { scrub.Set(v) },
	    core.OnSliderChangeEnd(func(v float64) { core.AudioSeek(v) }))

Each platform draws its own: Compose's Material 3 Slider, SwiftUI's Slider, and \<input type="range"> in the browser and in htmlout.

##### Two callbacks, and why the second is the important one

onChange fires continuously while the thumb moves — the value under the finger, dozens of times a second. That is right for a label that follows the drag and wrong for anything expensive or irreversible: a seek on a network stream, a request to a server. OnSliderChangeEnd fires once, when the finger lifts, with the final value — and it is the one a seek bar acts on. onChange may be nil when only the end matters.

Both cross the bridge as text callbacks carrying the number formatted with strconv (the natives' own float formatting is accepted too: "0.5", "1.0E-4" and "1e-4" all parse). A value that fails to parse is dropped, the same policy NumericInput applies.

##### The control is controlled

Like every leaf, the value shown is the one Go rendered — but a drag has to feel immediate, and the Go round trip is asynchronous, so the native renderers show the finger's value \*while dragging\* and Go's value otherwise (the same compromise the text fields make). A seek bar fed by a status tick therefore never snaps the thumb back under the finger.

<small>[core/slider.go:38](https://github.com/rohanthewiz/grmob/blob/master/core/slider.go#L38)</small>

#### func Spacer

```go
func Spacer(size int) View
```

<small>[core/layout.go:138](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L138)</small>

#### func Switch

```go
func Switch(on bool, onToggle func(bool), props ...PropsAndChildren) View
```

Switch is the instant-effect boolean: a track and a thumb, flipped to turn one thing on or off right now. Airplane mode, notifications, dark theme.

	core.Switch(settings.Notify, func(on bool) { settings.SetNotify(on) })

Each platform draws its own — Material 3's Switch on Android, SwiftUI's Toggle on iOS, and an \<input type="checkbox" switch role="switch"> in the browser and in htmlout.

##### Why this is a node type and not a flag on Checkbox

The two controls look similar in a props list and are not interchangeable on screen, and the difference is \*when the choice takes effect\*. A checkbox collects a value that something else will act on — a form's "remember me", a row's selection, a terms box above a Submit button — so a tick that sits there unacted-on is the expected state. A switch acts on the tap: there is no Submit, and a switch that needed one would be read as broken.

That is a platform-idiom difference rather than a styling one, which is what makes it a type. Both natives draw \*both\* controls, and they are different controls there (Compose's Checkbox and Switch; on iOS the platform has no checkbox at all and Toggle stands in for one — see GrMobCheckbox in the SwiftUI renderer). A bool prop on Checkbox would have the reconciler swapping one platform control for another inside one node's update-props patch, which is exactly the work a node type does properly: a changed type is a replace, and a replace is how a control is exchanged.

##### The state crosses the wire as "checked"

Go says \`on\` because a switch is on, and the wire says \`checked\` because that is what the DOM calls it — the same split core.SelectedState makes when it spells a bool "true" in a prop map.

It is not only tidiness. Both web renderers already carry a \`checked\` prop: htmlout writes the boolean attribute from it and the WASM runtime assigns el.checked from it on create \*and\* on update-props. Naming the prop \`on\` would have meant a second spelling of both halves in both renderers — four new branches whose only job is to mean what an existing branch already means — and the update half is the one that would have been forgotten, because a switch drawn correctly on the first render and frozen thereafter looks like a working widget until somebody changes its value from Go.

##### What announces it, and why the role is not a core.Role

HTML has no switch control. It has a \*switch attribute\* on a checkbox (WHATWG HTML; Safari draws it, most browsers do not yet), and it has role="switch", which tells a screen reader what this is in every browser regardless. Both are written, so the control announces itself correctly everywhere and is drawn correctly where the browser can — and where it cannot, it degrades to a checkbox, which is the same bool in the same state.

That role is written from the \*node type\*, with no Style involved, which makes this the second such node after Modal's dialog — see htmlout.CarriesOwnRole. It is deliberately not a value in the Role vocabulary: every Role a caller can spell obliges all four renderers to grow an arm for it (core.Roles() is held against each renderer's dispatch in mobile/verify), and there is nothing for the natives to do here. A Material Switch and a SwiftUI Toggle announce themselves as switches already. \`switch\` therefore joins aria/spec.NearMisses for the reason \`dialog\` is there: a role this framework emits and does not name.

##### It reads the theme's CheckBox base

The same base the other boolean control reads, and not a field of its own. What that style actually contributes is geometry and display — both natives read only margin and size off a control's style (marginAndSize in the Compose renderer, marginAndSizeOnly in SwiftUI), and on the web a control drawn by the user agent ignores a fill. A Components.Switch field would therefore be a palette entry no palette could spend, measured by the contrast census as though some surface were drawn from it.

Like Checkbox it carries no label: a control's label is the caller's, and comps.FormField and comps.InputRow already own that slot.

##### Keyboard focus

Not in focusableLeafTypes, for Checkbox's reason one file over: neither native renderer gives one keyboard focus, so the stamp would emit a patch per focus command that nothing on the far side reads. A browser focuses the \<input> for free, as it does a checkbox's.

<small>[core/switch.go:84](https://github.com/rohanthewiz/grmob/blob/master/core/switch.go#L84)</small>

#### func TabView

```go
func TabView(props ...TabViewProp) View
```

<small>[core/tabview.go:19](https://github.com/rohanthewiz/grmob/blob/master/core/tabview.go#L19)</small>

#### func Text

```go
func Text(content string, styleProps ...StyleProp) View
```

<small>[core/text.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/text.go#L3)</small>

#### func TextArea

```go
func TextArea(value string, onChange func(string), rows int, props ...PropsAndChildren) View
```

<small>[core/input.go:334](https://github.com/rohanthewiz/grmob/blob/master/core/input.go#L334)</small>

#### func TextGrid

```go
func TextGrid(rows []GridRow, props ...PropsAndChildren) View
```

TextGrid is a monospace grid of styled text: a terminal pane, a log tail, a hex dump. Rows are given in order; each row is a run of styled spans that the renderer lays out in a fixed-pitch font with no wrapping.

	core.TextGrid(rows, core.FontSize(12), core.Background("#000"))

##### Why a node type and not a Column of Text

Nothing else in core can draw this. Style has no font family, so a Text cannot ask for a fixed pitch, and a row of Text nodes has no way to keep its glyphs on a cell grid across styled runs. Emulating a grid from Row and Text would also put every run in the node tree as its own element, which for an 80×24 pane at terminal diff rate is a patch stream measured in thousands of nodes per second.

##### Rows are children, so a changed row is one patch

The grid renders as a container node with one GridRow child per row, and each row's runs are one prop on that child. The reconciler pairs children by index and compares props by value, so a pass that changes three rows of twenty-four emits three update-props patches and nothing else; an unchanged row costs a DeepEqual on its runs and no traffic. A renderer therefore repaints a row, never the grid. No caching is needed to get this; it falls out of the node shape.

Each platform draws its own: Compose an AnnotatedString in a monospace Text per row, SwiftUI an AttributedString with the monospaced design, the browser and htmlout a \<pre> of \<div> rows holding \<span> runs.

The Style applies to the grid as a whole (FontSize, TextColor and Background are the ones that matter; a run's own colours override the grid's). Behavior props apply to the grid too, so a tap on any cell is a tap on the grid.

<small>[core/textgrid.go:36](https://github.com/rohanthewiz/grmob/blob/master/core/textgrid.go#L36)</small>

#### func WithTheme

```go
func WithTheme(theme *Theme, children ...View) View
```

<small>[core/theme.go:481](https://github.com/rohanthewiz/grmob/blob/master/core/theme.go#L481)</small>

#### func ZStack

```go
func ZStack(stylePropsAndChildren ...PropsAndChildren) View
```

ZStack overlays its children: every child is drawn in the same box, in tree order, so the last one written is the one on top. It is the framework's only z-axis container, and the one thing Box deliberately is not.

	core.ZStack(
	    core.Width("160px"), core.Height("160px"),
	    rose,          // painted first, underneath
	    indexMark,     // painted second, over it
	)

##### Why this is a node type and not a style

core.Style already carries Position, Top/Right/Bottom/Left and ZIndex, and they are CSS spellings that only the two DOM targets read — Renderer.swift and Renderer.kt consult none of the five. So anything built out of them is a web-only widget wearing a portable name, which is exactly why comps.Compass parked its index mark \*above\* the rose instead of over it. An overlay has a first-class construct on each of the other three targets (a SwiftUI ZStack, a Compose Box, a single-cell CSS grid), and naming the container is what lets each renderer reach for its own.

##### The alignment contract: centred by default

Every child is centred on both axes and keeps its own size. That is the one arrangement all three constructs agree on without argument — SwiftUI's ZStack already defaults to .center, Compose's Box is told to (its own default is TopStart), and the grid cell is given align-items/justify-items centre — and agreeing exactly is worth more than a default that varied, because an overlay that drifted a few points between targets is a bug nobody sees until they hold two phones side by side.

A child that wants to sit somewhere else says so with StackAlign, the per-layer opt-out:

	core.ZStack(
	    core.Width("160px"), core.Height("160px"),
	    rose,
	    core.Text("▼", core.StackAlign(core.StackAlignTop)),
	)

The nine placements and what each target makes of one are in core/stack\_align.go. The centre is the zero value and has no spelling, so a layer that says nothing is placed exactly as every layer was before the property existed.

It arrived a good while after this container did, and the reason is worth recording: while comps.Compass was the only consumer, the escape was to give the layer \*its own box\* — the index mark was a full-height Column justifying its glyph to the start, which lands the mark at top centre while the Column itself is centred like everything else. That works, and one consumer is not a vocabulary. What made it a vocabulary is that all three constructs turned out to have the same nine-value 2D placement enum, so the prop could be portable rather than a CSS property with two renderers ignoring it — which is what Style.AlignSelf beside it still is.

##### What the stack sizes to

The largest child, on every target. A ZStack with no size of its own is as big as the biggest thing in it, which is why the example above states the rose's dimensions on the stack: pinning the box is what keeps a smaller overlay from deciding the size.

That holds on all four targets including a stack with a placed layer, which it did not always. SwiftUI has no per-child ZStack alignment, so the iOS renderer used to place a layer by wrapping it in a frame that filled the stack — and a filling frame is greedy, so an \*unsized\* stack with an aligned layer grew to its parent's proposal there. It now places by coordinate through a custom Layout instead; see core/stack\_align.go for the divergence and what closed it. Pinning a stack's dimensions is still worth doing, and is no longer the difference between two renderings.

Like Box and Scroll it carries no theme base — a theme Column's screen inset applied to an overlay would offset every layer by 16px and change nothing about their relationship.

<small>[core/layout.go:315](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L315)</small>

### type Weight

```go
type Weight int
```

<small>[core/style.go:670](https://github.com/rohanthewiz/grmob/blob/master/core/style.go#L670)</small>

```go
const (
	Light  Weight = 200
	Normal Weight = 400
	Bold   Weight = 700
)
```

### type WhenClause

```go
type WhenClause struct {
	Condition bool
	View      View
}
```

<small>[core/conditionals.go:30](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L30)</small>

#### func Otherwise

```go
func Otherwise(view View) WhenClause
```

<small>[core/conditionals.go:39](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L39)</small>

#### func When

```go
func When(cond bool, view View) WhenClause
```

<small>[core/conditionals.go:35](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L35)</small>

