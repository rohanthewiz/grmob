# Package core — Layout

```go
import "github.com/rohanthewiz/grmob/core"
```

Rows, columns, stacks, scrolls and lists, and the alignment vocabulary they are placed with.

One of 10 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/layout.go`, `core/list.go`, `core/stack_align.go`, `core/alignment.go`, `core/keyboard.go`, `core/placement_audit.go`.

## Index

- [Constants](#constants) — `ConcernInertPlacement`
- [`func AlignItemsValues`](#func-alignitemsvalues)
- [`func Alignments`](#func-alignments)
- [`func BorderColor`](#func-bordercolor)
- [`func BorderWidth`](#func-borderwidth)
- [`func Box`](#func-box)
- [`func Card`](#func-card)
- [`func Column`](#func-column)
- [`func Divider`](#func-divider)
- [`func Fragment`](#func-fragment)
- [`func GroupingContainers`](#func-groupingcontainers)
- [`func Horizontal`](#func-horizontal)
- [`func JustifyContents`](#func-justifycontents)
- [`func KeyboardAware`](#func-keyboardaware)
- [`func List`](#func-list)
- [`func OnEndReached`](#func-onendreached)
- [`func PlacingContainers`](#func-placingcontainers)
- [`func Row`](#func-row)
- [`func SafeArea`](#func-safearea)
- [`func Scroll`](#func-scroll)
- [`func Spacer`](#func-spacer)
- [`func StackAlign`](#func-stackalign)
- [`func StickyHeader`](#func-stickyheader)
- [`func TextAlignments`](#func-textalignments)
- [`func ZStack`](#func-zstack)
- [`type PropsAndChildren`](#type-propsandchildren)
- [`type StackAlignment`](#type-stackalignment)
    - [`func StackAlignments`](#func-stackalignments)

## Constants

ConcernInertPlacement: a node states core.Style.StackAlign and the container that would place it is not an overlay.

Deliberately inert rather than accidentally so — a layer prop that re-placed a Row's children would be worse than one that did nothing, and align-self already means something else to a flex child — which is what makes this a concern and not a bug to fix in a renderer. The author has written a prop that cannot work where they put it, and the four targets agree silently.

```go
const ConcernInertPlacement = "inert-stack-placement"
```

<small>[core/placement_audit.go:52](https://github.com/rohanthewiz/grmob/blob/master/core/placement_audit.go#L52)</small>

## Functions

### func AlignItemsValues

```go
func AlignItemsValues() []AlignItems
```

AlignItemsValues returns every declared AlignItems, in declaration order.

Named for its values rather than its type because the type name is taken — the same collision AlignItemsProp in style\_props.go had to work around.

Cross-axis placement. AlignItemsStretch is the member that behaves unlike the rest on every native: a stretched child is not \*placed\* differently, it is \*measured\* differently, so neither SwiftUI's alignment nor Compose's Alignment enum can express it and both runtimes handle it off the dispatch (a fill modifier on the child, a solver branch). A stretch arm in those switches is therefore expected to be a no-op — but an explicit no-op arm is how a reader learns the value was considered and handled elsewhere, which is the whole argument for holding a switch to a list.

<small>[core/alignment.go:153](https://github.com/rohanthewiz/grmob/blob/master/core/alignment.go#L153)</small>

### func Alignments

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

### func BorderColor

```go
func BorderColor(hex string) StyleProp
```

<small>[core/layout.go:351](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L351)</small>

### func BorderWidth

```go
func BorderWidth(px float64) StyleProp
```

<small>[core/layout.go:356](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L356)</small>

### func Box

```go
func Box(stylePropsAndChildren ...PropsAndChildren) View
```

Box is the unopinionated container: a Column with no theme base. It stacks its children vertically, honors Gap/JustifyContent/AlignItems on the same axes a Column does, and reads the Align cross-axis fallback the same way — the only difference is that no theme style arrives with it, so nothing insets or paints the box but the caller.

It is not an overlay, on any target. Both natives used to draw it as one (a Compose Box, a SwiftUI ZStack, each pinned to the top-start corner) while the DOM targets stacked its children, so a Box with two children rendered two different pictures. mobile/verify's TestNativeContainersStackTheirChildrenAndDoNotOverlay pins the agreement.

ZStack, below, is the container that does overlay — and it exists because this one stopped. The two are the same argument from both ends: one shape per node type, stated once, rather than a container whose meaning depended on which renderer was reading it.

<small>[core/layout.go:258](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L258)</small>

### func Card

```go
func Card(stylePropsAndChildren ...PropsAndChildren) View
```

<small>[core/layout.go:132](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L132)</small>

### func Column

```go
func Column(stylePropsAndChildren ...PropsAndChildren) View
```

<small>[core/layout.go:236](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L236)</small>

### func Divider

```go
func Divider(height int, color string) View
```

<small>[core/layout.go:344](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L344)</small>

### func Fragment

```go
func Fragment(children ...View) View
```

<small>[core/layout.go:224](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L224)</small>

### func GroupingContainers

```go
func GroupingContainers() []string
```

GroupingContainers returns the transparent node types, sorted. See groupingContainers.

<small>[core/stack_align.go:198](https://github.com/rohanthewiz/grmob/blob/master/core/stack_align.go#L198)</small>

### func Horizontal

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

#### Why a StyleProp and not a Props flag

"Sideways" is spelled entirely in properties Style already carries, and CSS is the spelling both DOM targets already implement:

	flex-direction: row      <- Style.FlexDirection
	overflow: auto           <- Style.Overflow

So the two web renderers need no change at all to honour this — htmlout's styleValue lets an explicit FlexDirection override the node's stacking axis, and grmob-runtime.js's styleFromGrMob does the same. That matters more than it looks: the runtime's style path is \*total\* (an update-style patch carries the whole new Style and every managed property is reassigned), so a flag living in Props would have had to be mirrored onto the element and re-read on every style patch, or the first unrelated re-render would have quietly stood the strip back up on end. A Style field rides the patch channel that was built for exactly this.

The natives, which have no CSS to inherit, read Style.FlexDirection in their Scroll composite and nowhere else — the same narrow contract Style.FlexWrap already has, where only the Row composite reads it.

#### Overflow is supplied, not forced

A vertical Scroll emits no overflow on the web at all: the page scrolls, and the region is just a column. A horizontal one has no such fallback — with no overflow the row is simply clipped or squashed — so this supplies \`auto\` when the caller has not said otherwise. An explicit core.Overflow wins in either argument order, because the value is only defaulted when it is still empty.

(\`overflow: auto\` covers both axes rather than overflow-x alone. CSS forces the other axis to auto anyway the moment one of them is not \`visible\`, so the shorthand says what the browser was going to do; a strip whose children fit its height never shows the vertical bar.)

#### What it is not

It is not a horizontal List. core.List's laziness, its cross-axis stretch and its FlexGrow contract are all written for a vertical main axis on both natives, and nothing yet asks for a lazily-materialized carousel. A strip of chips or a handful of cards is short by construction, which is what Scroll is for.

<small>[core/layout.go:433](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L433)</small>

### func JustifyContents

```go
func JustifyContents() []JustifyContent
```

JustifyContents returns every declared JustifyContent, in declaration order.

Main-axis distribution. The DOM pair emits these verbatim (core's spellings are the CSS ones), so the drift risk is entirely on the natives, where the six values are spread across dispatches that each answer for part of the question: GrMobFlex.swift computes a leading offset in one switch and an inter-item gap in another, and a value absent from \*both\* silently renders as flex-start.

<small>[core/alignment.go:129](https://github.com/rohanthewiz/grmob/blob/master/core/alignment.go#L129)</small>

### func KeyboardAware

```go
func KeyboardAware() BehaviorProp
```

KeyboardAware makes a region yield to the software keyboard instead of being covered by it: while the keyboard is up, the node it is applied to ends where the keyboard begins.

	core.Scroll(
	    core.KeyboardAware(),
	    form,
	)

#### The two shapes it takes

On a \*scrolling\* node (Scroll, List) it shrinks the viewport. The content does not move on its own — but because the viewport now ends above the keyboard, the platform's own "scroll the focused field into view" behavior (Compose's BasicTextField, SwiftUI's ScrollView) lands the field somewhere the user can see, which it cannot do while the viewport still claims the rows the keyboard is sitting on. This is the form case.

On any \*other\* node it lifts that subtree whole. That is the case for a screen with something docked at the bottom — a chat composer, a checkout bar — which is outside the scrolling region by construction and would otherwise be the one thing the keyboard covers. Applied to a whole screen's column, it is the classic "the app resizes for the keyboard" behavior, asked for explicitly and by one screen at a time.

#### What it deliberately does not do

It does not dismiss the keyboard on a tap outside — but that is now a thing an app can ask for directly, which it was not when this prop was written:

	core.Box(
	    core.OnClick(func() { core.DismissKeyboard(ctx) }),
	    form,
	)

Keeping the two separate is the point. This prop is about \*layout\* — which region yields the space the keyboard takes — and dismissal is about focus. A chat composer wants the inset and emphatically does not want a stray tap closing the keyboard between messages; a settings form wants the opposite. Folding one into the other would take that choice away.

(Dragging a keyboard-aware scroll region also dismisses it on iOS — see below — which is the platform's own gesture, not a handler of ours.)

#### What each platform does with it

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

### func List

```go
func List(stylePropsAndChildren ...PropsAndChildren) View
```

List is the virtualized sibling of Column: a vertically scrolling container whose children are laid out lazily by the native renderer (Compose LazyColumn, SwiftUI LazyVStack), so a thousand-row feed composes only the rows on screen. Column + Scroll remains the right choice for short content; List is for long, data-driven collections.

Give every child a stable identity with Keyed(id, ...) — the native lazy containers use the key to keep row state (and recycled views) attached to the same data across insertions, removals, and reorders. Unkeyed children fall back to positional identity, which behaves like Column but loses row state on reorder.

It shares Column's theme base and the standard container argument contract: style props, behavior props (e.g. OnClick on the list surface), and child views in any order.

<small>[core/list.go:20](https://github.com/rohanthewiz/grmob/blob/master/core/list.go#L20)</small>

### func OnEndReached

```go
func OnEndReached(handler func()) BehaviorProp
```

OnEndReached fires when the user scrolls within a few rows of the bottom of a List: the "fetch the next page" edge that turns a manual comps.LoadMore button into an infinite feed.

	core.List(
	    core.OnEndReached(pager.LoadNext),
	    rows...,
	)

#### The debounce, and why it is here rather than in four renderers

The edge is a \*scroll position\*, so every renderer reports it more than once for the same bottom: Compose's snapshot flow emits on each new last-visible index, SwiftUI's .onAppear re-fires when a row is recycled back into view, and an IntersectionObserver fires on entry and on every resize that keeps the sentinel visible. A slow fetch therefore sees two or three calls before its first page lands, and an offset pager answers that by loading page 2 twice.

The fix is one line of state and it belongs on this side of the bridge: remember how many rows the list held when the handler last ran, and refuse to run again until that number changes. A fetch that appends rows unlocks the next fire; a fetch that returns nothing (the feed is exhausted, or it failed) leaves the guard closed, which is exactly right — scrolling at the bottom of a list that just came back empty should not re-ask forever. A caller that wants the retry offers a button; that is what comps.LoadMore's error arm has always been for.

Doing it in Go also means the four renderers each get to be as naive as their platform makes convenient, and none of them has to agree with the others about what "once" means.

The row count is read at \*dispatch\* time off the node this prop was applied to, which by then holds the children the pass rendered. (Behaviour props run before children in containerNode, so there is nothing to count yet when this closure is built — only when it is called.)

#### Where in the argument list it goes

Anywhere. containerNode registers behavior props in argument order but renders children only after that loop has finished, so a List's own callback IDs always precede its rows' — and these two spellings produce the same ID for the same list, at any row count:

	core.List(core.OnEndReached(pager.LoadNext), rows...)
	core.List(append(rows, core.OnEndReached(pager.LoadNext))...)

That is worth saying out loud rather than leaving to be derived, because the guard below is keyed by the ID and a reader who works out what the key is made of is right to wonder whether a page that lengthens the list moves it. Within one List it cannot; TestOnEndReachedIDIsIndependentOfArgumentOrder holds the contract still.

What \*does\* move it is anything earlier in the same pass that registers a varying number of callbacks: a sibling above the List whose own children grow with the page, or a row helper that renders on the spot with view.Render(ctx) rather than returning a View for the List to render. Then the ID slides with the data, each page starts its guard from scratch under a key something else held on the previous pass, and the double-load this prop exists to prevent comes back. Same family as the edge below, and the same identity-keyed IDs close both.

#### The guard's one sharp edge

State is keyed by callback ID, and callback IDs are positional: the Nth void handler registered in a pass is always "cb\_N" (see callbackRegistry). Two different Lists in two different screens can therefore inherit the same ID across a navigation, and the second one's first end-reached is swallowed if it happens to hold exactly as many rows as the first did when the first last fired. That is the same stale-ID window the registry itself documents, and it closes when identity-keyed IDs land; nothing here can close it earlier, because the two lists are indistinguishable from this side.

<small>[core/list.go:154](https://github.com/rohanthewiz/grmob/blob/master/core/list.go#L154)</small>

### func PlacingContainers

```go
func PlacingContainers() []string
```

PlacingContainers returns the node types that place their children, sorted.

Sorted for the reason htmlout's OverlayTypes is: a test looping over a map reports in a different order every run, and a census that names the offender wants one order.

<small>[core/stack_align.go:192](https://github.com/rohanthewiz/grmob/blob/master/core/stack_align.go#L192)</small>

### func Row

```go
func Row(stylePropsAndChildren ...PropsAndChildren) View
```

<small>[core/layout.go:126](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L126)</small>

### func SafeArea

```go
func SafeArea(stylePropsAndChildren ...PropsAndChildren) View
```

SafeArea insets its content from the system bars and the display cutout. It is the root of every screen (comps.Screen builds one) and takes the same mixed argument list as the other containers, so a style can land on the inset box itself.

The one style worth putting there is a background. The inset is padding on this node, so a background here paints under the status bar while the content stays clear of it — whereas a background on the content column stops at the inset and leaves the strip behind the bar in the window's own colour, which on a dark screen is a light band along the top. Each native renderer paints this node's background edge to edge (Compose orders the background before the inset padding; SwiftUI extends it with ignoresSafeArea); the DOM targets have no system bars and treat it as any other container. Padding and margin here are honoured too but rarely wanted, since they inset the whole screen a second time.

Like Scroll it has no theme base: the theme Column's screen padding would otherwise inset the content twice.

Below the inset it is a Column, on every target: children stack and, with no cross-axis alignment set, stretch to its width. Both natives used to draw it as an overlay (a Compose Box, a SwiftUI ZStack), which stacked two children on top of each other and let a lone one — a screen's whole content column, usually — hug its widest child instead of filling the screen.

<small>[core/layout.go:218](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L218)</small>

### func Scroll

```go
func Scroll(stylePropsAndChildren ...PropsAndChildren) View
```

Scroll is a vertically scrolling region: its children are laid out at their natural height and the viewport pans over them.

It takes the standard container argument list — style props, behavior props and child views in any order — rather than the bare ...View it used to, because both native renderers have always applied a Scroll node's style (Compose boxModifier, SwiftUI grMobBox) and Go had no way to set one. The widening is source-compatible: a View is a PropsAndChildren, so every existing core.Scroll(child) call still compiles and, with no props supplied, still renders the same box.

Unlike Column and Row it has no theme base — like Box, it is the unopinionated container, and a scroll region that arrived with the theme Column's screen padding would inset every screen that wraps itself in one.

See KeyboardAware for the software-keyboard behavior.

#### Inside another vertical scroll

A vertical Scroll whose parent is itself a vertical scroll has no viewport to be smaller than. In the DOM an overflow box of auto height is simply as tall as its content and never pans. Compose caps the scroll at the content's intrinsic height when the incoming height is unbounded, which draws the same picture — and that is a guard in the renderer, not Compose's default: a bare verticalScroll throws under an infinite height, and the same guard is what lets a CodeEditor with no Height sit in a comps.Screen{Scroll: true}. SwiftUI needs no guard: the outer ScrollView proposes no height, and an inner ScrollView answers with its content's height. On a simulator the tutorial's Height-less Lists in a scrolled lesson (4.3's outline, 4.6's GroupedLists) came out exactly as tall as their rows, and a drag that starts on one scrolls the page. Give the inner region a Height when its size matters, and avoid the shape where it can: two nested regions that both \*can\* pan fight for the same drag, which is the reason comps.Screen.Scroll gives.

The same shape sideways: a Horizontal() Scroll, a TextGrid or a CodeEditor inside a Horizontal() Scroll. Compose's horizontal scroll throws under an infinite width just as the vertical one does, so the renderer caps that axis too, and on an emulator the inner regions draw at their content width while the outer strip pans. SwiftUI's sideways nesting has not been measured.

<small>[core/layout.go:188](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L188)</small>

### func Spacer

```go
func Spacer(size int) View
```

<small>[core/layout.go:138](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L138)</small>

### func StackAlign

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

### func StickyHeader

```go
func StickyHeader() StyleProp
```

StickyHeader pins a List child to the top of the viewport while the rows it introduces scroll past underneath it, releasing it when the next sticky child arrives to take its place. It is the group band of an archive feed — the month over a run of sermons, the day over a run of transactions.

	core.List(
	    core.Keyed("group:2026-01", core.Row(core.StickyHeader(), monthLabel)),
	    core.Keyed("s1", row), core.Keyed("s2", row),
	)

comps.GroupedList{StickyHeaders: true} is the widget spelling; this is the primitive underneath it.

#### Why a StyleProp, and why no new field

Sticky positioning is not a new idea to this tree: Style.Position already carries PositionSticky, and both DOM targets already emit \`position\`, \`top\` and \`z-index\` verbatim, so the web half of this feature has always worked and needed no code. What was missing was the two natives, which declined Position outright ("no Compose analog at this layer") — true of \`fixed\` and \`absolute\`, and not true of \`sticky\`, which is precisely what a Compose stickyHeader and a SwiftUI pinned Section header are.

So the marker is the CSS one, and the renderers converge on it rather than on a private flag:

	web       position:sticky; top:0; z-index:1
	Compose   LazyColumn { stickyHeader { … } } for the marked rows
	SwiftUI   LazyVStack(pinnedViews: .sectionHeaders) + Section(header:)

Top and ZIndex are \*supplied\* rather than assigned — a caller's own values survive in either argument order — because both are load-bearing on the web and neither has a sensible zero: a sticky box with no offset never sticks, and one at the default stacking level is painted over by the rows that scroll under it.

#### Where it does something

A List child. Both natives implement pinning inside their lazy container and have nowhere to put it otherwise, so a Column child carrying this is sticky in a browser and inert on a phone. That asymmetry is why the name says List's word for the thing ("header") rather than CSS's word for the mechanism.

<small>[core/list.go:69](https://github.com/rohanthewiz/grmob/blob/master/core/list.go#L69)</small>

### func TextAlignments

```go
func TextAlignments() []Alignment
```

TextAlignments returns the Alignments that name a real text alignment, in declaration order: the coverage every renderer's text dispatch owes.

AlignStretch and AlignBaseline are excluded because there is no such thing as stretched or baseline-aligned text — CSS text-align has no such keyword, SwiftUI's TextAlignment has three members, and Compose's TextAlign has no analogue either. They are cross-axis values that share the type; a text dispatch that receives one is right to fall through to its default.

AlignJustify \*is\* included, and is the value that made this list worth writing. Before it existed, justified text rendered on exactly one of the four targets: Renderer.kt mapped it to TextAlign.Justify, Renderer.swift fell through to .leading, htmlout emitted no declaration at all, and the WASM runtime did not read Align in the first place. One value, four behaviors, and nothing anywhere that could notice.

Being on this list does not mean a target can honor the value — SwiftUI genuinely cannot justify text — it means the target has to \*say\* what it does with it. An explicit arm that falls back to leading, with a comment naming the platform limit, is coverage; silence is not.

<small>[core/alignment.go:112](https://github.com/rohanthewiz/grmob/blob/master/core/alignment.go#L112)</small>

### func ZStack

```go
func ZStack(stylePropsAndChildren ...PropsAndChildren) View
```

ZStack overlays its children: every child is drawn in the same box, in tree order, so the last one written is the one on top. It is the framework's only z-axis container, and the one thing Box deliberately is not.

	core.ZStack(
	    core.Width("160px"), core.Height("160px"),
	    rose,          // painted first, underneath
	    indexMark,     // painted second, over it
	)

#### Why this is a node type and not a style

core.Style already carries Position, Top/Right/Bottom/Left and ZIndex, and they are CSS spellings that only the two DOM targets read — Renderer.swift and Renderer.kt consult none of the five. So anything built out of them is a web-only widget wearing a portable name, which is exactly why comps.Compass parked its index mark \*above\* the rose instead of over it. An overlay has a first-class construct on each of the other three targets (a SwiftUI ZStack, a Compose Box, a single-cell CSS grid), and naming the container is what lets each renderer reach for its own.

#### The alignment contract: centred by default

Every child is centred on both axes and keeps its own size. That is the one arrangement all three constructs agree on without argument — SwiftUI's ZStack already defaults to .center, Compose's Box is told to (its own default is TopStart), and the grid cell is given align-items/justify-items centre — and agreeing exactly is worth more than a default that varied, because an overlay that drifted a few points between targets is a bug nobody sees until they hold two phones side by side.

A child that wants to sit somewhere else says so with StackAlign, the per-layer opt-out:

	core.ZStack(
	    core.Width("160px"), core.Height("160px"),
	    rose,
	    core.Text("▼", core.StackAlign(core.StackAlignTop)),
	)

The nine placements and what each target makes of one are in core/stack\_align.go. The centre is the zero value and has no spelling, so a layer that says nothing is placed exactly as every layer was before the property existed.

It arrived a good while after this container did, and the reason is worth recording: while comps.Compass was the only consumer, the escape was to give the layer \*its own box\* — the index mark was a full-height Column justifying its glyph to the start, which lands the mark at top centre while the Column itself is centred like everything else. That works, and one consumer is not a vocabulary. What made it a vocabulary is that all three constructs turned out to have the same nine-value 2D placement enum, so the prop could be portable rather than a CSS property with two renderers ignoring it — which is what Style.AlignSelf beside it still is.

#### What the stack sizes to

The largest child, on every target. A ZStack with no size of its own is as big as the biggest thing in it, which is why the example above states the rose's dimensions on the stack: pinning the box is what keeps a smaller overlay from deciding the size.

That holds on all four targets including a stack with a placed layer, which it did not always. SwiftUI has no per-child ZStack alignment, so the iOS renderer used to place a layer by wrapping it in a frame that filled the stack — and a filling frame is greedy, so an \*unsized\* stack with an aligned layer grew to its parent's proposal there. It now places by coordinate through a custom Layout instead; see core/stack\_align.go for the divergence and what closed it. Pinning a stack's dimensions is still worth doing, and is no longer the difference between two renderings.

Like Box and Scroll it carries no theme base — a theme Column's screen inset applied to an overlay would offset every layer by 16px and change nothing about their relationship.

<small>[core/layout.go:338](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L338)</small>

## Types

### type PropsAndChildren

```go
type PropsAndChildren any
```

<small>[core/layout.go:5](https://github.com/rohanthewiz/grmob/blob/master/core/layout.go#L5)</small>

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

