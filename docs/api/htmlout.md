# Package htmlout

```go
import "github.com/rohanthewiz/grmob/htmlout"
```

Package htmlout exports a rendered core.Node tree as a standalone HTML document. It is the demo/inspection path (the example apps print its output); the WASM runtime does not consume it, so readability is favored over compactness.

## Index

- [Constants](#constants) — `OverlayChassis`, `OverlayChildDecl`
- [`func AlignFallbackAxes`](#func-alignfallbackaxes)
- [`func AlignFallbackAxisFor`](#func-alignfallbackaxisfor)
- [`func AlignFallbackTypes`](#func-alignfallbacktypes)
- [`func AriaOrientationDefaults`](#func-ariaorientationdefaults)
- [`func AriaOrientationFor`](#func-ariaorientationfor)
- [`func BorderResetTypes`](#func-borderresettypes)
- [`func CarriesOwnRole`](#func-carriesownrole)
- [`func CrossAxisAlignFor`](#func-crossaxisalignfor)
- [`func CrossAxisAligns`](#func-crossaxisaligns)
- [`func EdgeCSS`](#func-edgecss)
- [`func ExportHTML`](#func-exporthtml)
- [`func GenericTags`](#func-generictags)
- [`func InputTypeFor`](#func-inputtypefor)
- [`func InputTypes`](#func-inputtypes)
- [`func IsGenericTag`](#func-isgenerictag)
- [`func IsOverlay`](#func-isoverlay)
- [`func IsTransparent`](#func-istransparent)
- [`func ModalChassis`](#func-modalchassis)
- [`func ObjectFitFor`](#func-objectfitfor)
- [`func ObjectFits`](#func-objectfits)
- [`func OverlayTypes`](#func-overlaytypes)
- [`func OwnRoleFor`](#func-ownrolefor)
- [`func ResetsUABorder`](#func-resetsuaborder)
- [`func StackAxes`](#func-stackaxes)
- [`func StackAxisFor`](#func-stackaxisfor)
- [`func StackPlacementFor`](#func-stackplacementfor)
- [`func StackPlacements`](#func-stackplacements)
- [`func StackTypes`](#func-stacktypes)
- [`func TagFor`](#func-tagfor)
- [`func Tags`](#func-tags)
- [`func TextAlignFor`](#func-textalignfor)
- [`func TextAligns`](#func-textaligns)
- [`func TransparentTypes`](#func-transparenttypes)

## Constants

The two halves of the overlay's CSS, named here rather than written into styleValue so the WASM runtime has something to be compared against and so the pair reads as one decision.

OverlayChassis goes on the stack itself. \`align-items\` and \`justify-items\` are the grid spellings of "where does an item sit inside its cell", and centre on both axes is the alignment contract core.ZStack documents — the one arrangement a SwiftUI ZStack, a Compose Box and a grid cell all agree on. They are the \*items\* properties, not the \*content\* ones: \`justify-content\` would place the single track inside the container, which on an auto-sized container is a no-op, and that difference is easy to write and impossible to see.

OverlayChildDecl goes on every child, imposed by the parent (see imposed in export.go) because a child has no idea it is a layer. \`grid-area: 1/1\` is the whole of it: row 1, column 1, on all of them. The layer's placement inside that cell is appended after it, from stackPlacements below — the one part of the imposed declaration that differs from sibling to sibling.

```go
const (
	OverlayChassis   = "display:grid; align-items:center; justify-items:center"
	OverlayChildDecl = "grid-area:1/1"
)
```

<small>[htmlout/stack.go:222](https://github.com/rohanthewiz/grmob/blob/master/htmlout/stack.go#L222)</small>

## Functions

### func AlignFallbackAxes

```go
func AlignFallbackAxes() map[string]string
```

AlignFallbackAxes returns a copy of the gate table, for the WASM conformance test — table against table, like CrossAxisAligns.

<small>[htmlout/crossaxis.go:134](https://github.com/rohanthewiz/grmob/blob/master/htmlout/crossaxis.go#L134)</small>

### func AlignFallbackAxisFor

```go
func AlignFallbackAxisFor(nodeType string) string
```

AlignFallbackAxisFor returns the flex axis a node type stacks along, for the types that read the Align cross-axis fallback, and "" for every type that does not.

<small>[htmlout/crossaxis.go:128](https://github.com/rohanthewiz/grmob/blob/master/htmlout/crossaxis.go#L128)</small>

### func AlignFallbackTypes

```go
func AlignFallbackTypes() []string
```

AlignFallbackTypes returns the node types that read the fallback, sorted so a test looping over them reports in a stable order. The behavioral tests range over this rather than a hand-written list, for the reason TransparentTypes exists: a list in a test is the untracked second copy this file removes.

<small>[htmlout/crossaxis.go:145](https://github.com/rohanthewiz/grmob/blob/master/htmlout/crossaxis.go#L145)</small>

### func AriaOrientationDefaults

```go
func AriaOrientationDefaults() map[string]string
```

AriaOrientationDefaults returns a copy of the role → ARIA-default table, for the WASM runtime's conformance test — which compares table against table and so cannot go through AriaOrientationFor one key at a time.

A copy, not the map itself, for the reason StackAxes and Tags return one: a package-level map is reachable and writable by any importer.

<small>[htmlout/orientation.go:89](https://github.com/rohanthewiz/grmob/blob/master/htmlout/orientation.go#L89)</small>

### func AriaOrientationFor

```go
func AriaOrientationFor(role core.Role, nodeType string, dir core.FlexDirection) string
```

AriaOrientationFor answers the aria-orientation value for one node, or "" for a role the attribute is not defined on.

The axis is resolved exactly as styleValue resolves it for the CSS declaration — an explicit FlexDirection over the node type's own stacking direction — so the attribute and the layout cannot disagree. The prefix tests rather than equality are for "row-reverse" and "column-reverse", which run along the same axes as the two they reverse.

grmob-runtime.js restates this as ariaOrientation, and the table it dispatches on is pinned to AriaOrientationDefaults by TestRuntimeOrientationTableMatchesGo.

<small>[htmlout/orientation.go:106](https://github.com/rohanthewiz/grmob/blob/master/htmlout/orientation.go#L106)</small>

### func BorderResetTypes

```go
func BorderResetTypes() []string
```

BorderResetTypes returns those node types, sorted so that a test looping over them reports in a stable order. Exported for the reason GenericTags is: the WASM conformance test has to compare set against set.

<small>[htmlout/tag.go:399](https://github.com/rohanthewiz/grmob/blob/master/htmlout/tag.go#L399)</small>

### func CarriesOwnRole

```go
func CarriesOwnRole(nodeType string) bool
```

CarriesOwnRole reports whether a node type states its own ARIA role, with no core.Style involved. See ownRoles.

Exported because the TabView wiring has to know: the role attribute has one slot per element, and a page whose type already filled it must not be given role="tabpanel" on top.

<small>[htmlout/tag.go:282](https://github.com/rohanthewiz/grmob/blob/master/htmlout/tag.go#L282)</small>

### func CrossAxisAlignFor

```go
func CrossAxisAlignFor(align string) string
```

CrossAxisAlignFor returns the CSS align-items value an alignment falls back to when AlignItems is unset, or "" for one that is absent, unrecognized, or a text-only Alignment.

It takes a string rather than a core.Alignment for the reason TextAlignFor does: the callers that need it most are reading the value back out of a place that has already lost the Go type.

<small>[htmlout/crossaxis.go:71](https://github.com/rohanthewiz/grmob/blob/master/htmlout/crossaxis.go#L71)</small>

### func CrossAxisAligns

```go
func CrossAxisAligns() map[string]string
```

CrossAxisAligns returns a copy of the whole table, keyed by the alignment's string form, for the two callers that must enumerate it rather than query it: the WASM runtime conformance test and the census test.

A copy, not the map itself, for the reason TextAligns hands out one — both callers delete from what they are given as they match rows.

<small>[htmlout/crossaxis.go:81](https://github.com/rohanthewiz/grmob/blob/master/htmlout/crossaxis.go#L81)</small>

### func EdgeCSS

```go
func EdgeCSS(e core.EdgeInsets) string
```

EdgeCSS serializes a core.EdgeInsets into the four-value CSS shorthand ("top right bottom left"), resolving the Horizontal/Vertical shorthand fields the way the native renderers do.

#### The resolution rule

core.EdgeInsets carries six fields, not four: the per-side Top/Right/ Bottom/Left plus a Horizontal/Vertical pair the DSL's PaddingHorizontal / PaddingVertical props write. A side that was not set explicitly takes its value from the shorthand for its axis:

	top    = Top    != 0 ? Top    : Vertical
	bottom = Bottom != 0 ? Bottom : Vertical
	left   = Left   != 0 ? Left   : Horizontal
	right  = Right  != 0 ? Right  : Horizontal

"Set explicitly" means non-zero, which is where the rule is lossy: in a hand-built EdgeInsets a zero Left is indistinguishable from an unset one, so {Horizontal: 16, Left: 0} cannot ask for a zero left inset. Both natives have the same limitation for the same reason (a Go zero value carries no "was it set?" bit), and reproducing it exactly is the point — an inset that resolves one way on device and another on the web is worse than one that is uniformly lossy.

The DSL is not subject to it. core.PaddingLeft and its three siblings dissolve the shorthand into the sides it was standing in for before writing their own, so PaddingHorizontal(16) followed by PaddingLeft(0) arrives here as {Left: 0, Right: 16} and resolves to a real zero. That is a transformation on the Style value, not a change to this rule; see core/padding\_sides.go, and TestSettlingAnAxisPreservesEveryResolvedSide for the proof that it leaves every other side resolving as it did.

#### The rule is also what lets the struct go on the wire sparsely

core.EdgeInsets' six fields are \`json:",omitzero"\`, which is safe precisely because of the rule above: a zero side already means "unset, take the axis", so a field at zero carries nothing and the three JSON readers each turn a missing key back into 0. This function never sees JSON — htmlout walks the Go tree directly — so it is unaffected either way; the note is here because this is where the rule is written down, and the tags rest on it.

This is a restatement of GrMobStyle.swift's parseEdges and GrMobStyle.kt's parseEdges, which have honored the shorthand since they were written. Until this function existed the two web targets read the four per-side fields only, so core.PaddingHorizontal(16) applied cleanly, rendered as 16px of padding on both natives, and as nothing at all in the browser. The WASM runtime's copy is edgeToCSS in wasm/grmob-runtime.js.

<small>[htmlout/edges.go:56](https://github.com/rohanthewiz/grmob/blob/master/htmlout/edges.go#L56)</small>

### func ExportHTML

```go
func ExportHTML(node *core.Node) string
```

ExportHTML renders the node tree into a complete HTML document.

Output is built on the element library rather than hand-assembled strings so that escaping is handled once, in one place: element quote-escapes every attribute value (a raw double quote is the attribute-breakout character), and text content goes through TE(), which entity-escapes it. User-originated strings — Text content, input values, labels, image srcs — therefore cannot re-enter the document as live markup.

<small>[htmlout/export.go:25](https://github.com/rohanthewiz/grmob/blob/master/htmlout/export.go#L25)</small>

### func GenericTags

```go
func GenericTags() []string
```

GenericTags returns the role-free tags, sorted so that a test looping over them reports in a stable order. Exported for the reason Tags and TransparentTypes are: the WASM conformance test has to compare set against set, and a hand-written list there would be exactly the untracked second copy this file exists to remove.

<small>[htmlout/tag.go:190](https://github.com/rohanthewiz/grmob/blob/master/htmlout/tag.go#L190)</small>

### func InputTypeFor

```go
func InputTypeFor(nodeType string) string
```

InputTypeFor returns the HTML \<input> type attribute a node type renders as, or "" for a node that is not an \<input>.

<small>[htmlout/inputtype.go:46](https://github.com/rohanthewiz/grmob/blob/master/htmlout/inputtype.go#L46)</small>

### func InputTypes

```go
func InputTypes() map[string]string
```

InputTypes returns a copy of the whole table, for the callers that must enumerate it rather than query it — currently only the WASM runtime conformance test, which has to compare table against table and so cannot go through InputTypeFor one key at a time.

A copy, not the map itself: a package-level map is reachable and writable by any importer, and a table this small is cheaper to copy than to defend.

<small>[htmlout/inputtype.go:57](https://github.com/rohanthewiz/grmob/blob/master/htmlout/inputtype.go#L57)</small>

### func IsGenericTag

```go
func IsGenericTag(tag string) bool
```

IsGenericTag reports whether a tag's implicit ARIA role is \`generic\`, and so whether a role= attribute may be written onto it. See genericTags.

<small>[htmlout/tag.go:181](https://github.com/rohanthewiz/grmob/blob/master/htmlout/tag.go#L181)</small>

### func IsOverlay

```go
func IsOverlay(nodeType string) bool
```

IsOverlay reports whether a node type draws its children on top of one another. See overlayTypes.

<small>[htmlout/stack.go:188](https://github.com/rohanthewiz/grmob/blob/master/htmlout/stack.go#L188)</small>

### func IsTransparent

```go
func IsTransparent(nodeType string) bool
```

IsTransparent reports whether a node type renders its children directly into the parent, with no element of its own.

<small>[htmlout/tag.go:233](https://github.com/rohanthewiz/grmob/blob/master/htmlout/tag.go#L233)</small>

### func ModalChassis

```go
func ModalChassis() [][2]string
```

ModalChassis returns the fixed declarations of a Modal's overlay look, for the WASM runtime conformance test.

The runtime states the same set in styleFromGrMob, spelled as CSSOM property names with a \`||\` per line so an author's Style wins — which is what this exporter gets from the cascade by writing the chassis first. The two are compared by TestRuntimeModalChassisMatchesGo.

display and background are deliberately not in it. Both are prop-driven, and the two targets get them by different routes: this exporter writes the whole declaration list at once from props it can see, while the runtime's style pass never sees a prop and abstains from display entirely, leaving it to the visible prop's own path.

A copy, not the slice itself, for the reason StackAxes returns one: a package-level slice is reachable and writable by any importer.

<small>[htmlout/export.go:861](https://github.com/rohanthewiz/grmob/blob/master/htmlout/export.go#L861)</small>

### func ObjectFitFor

```go
func ObjectFitFor(mode string) string
```

ObjectFitFor returns the CSS object-fit value a content mode maps to, or "" for a mode that is absent or unrecognized.

It takes a string rather than a core.ContentMode because every caller is reading it back out of a node's Props, where it arrives as one — the same reason InputTypeFor and TagFor take strings.

<small>[htmlout/objectfit.go:52](https://github.com/rohanthewiz/grmob/blob/master/htmlout/objectfit.go#L52)</small>

### func ObjectFits

```go
func ObjectFits() map[string]string
```

ObjectFits returns a copy of the whole table, keyed by the mode's string form, for the callers that must enumerate it rather than query it: the WASM runtime conformance test, which compares table against table, and the census test that holds this table to core.ContentModes().

A copy, not the map itself, for the reason InputTypes and Tags return one. Keyed by string rather than by core.ContentMode because both callers are comparing against something that has already lost the Go type — a parsed JavaScript literal, and a set built from strings.

<small>[htmlout/objectfit.go:65](https://github.com/rohanthewiz/grmob/blob/master/htmlout/objectfit.go#L65)</small>

### func OverlayTypes

```go
func OverlayTypes() []string
```

OverlayTypes returns those node types, sorted so a test looping over them reports in a stable order — the service StackTypes and BorderResetTypes provide, for the reason they give.

<small>[htmlout/stack.go:195](https://github.com/rohanthewiz/grmob/blob/master/htmlout/stack.go#L195)</small>

### func OwnRoleFor

```go
func OwnRoleFor(nodeType string) string
```

OwnRoleFor returns the ARIA role a node type states for itself, or "" for a type that states none. See ownRoles.

Exported for the reason Tags is: the WASM runtime has the same two node types to answer for and cannot ask Go at runtime, so wasm/verify holds its copy against this one rather than against a list written twice.

<small>[htmlout/tag.go:293](https://github.com/rohanthewiz/grmob/blob/master/htmlout/tag.go#L293)</small>

### func ResetsUABorder

```go
func ResetsUABorder(nodeType string) bool
```

ResetsUABorder reports whether a node type needs an explicit "no border" written for it when the style declares none. See borderResetTypes.

<small>[htmlout/tag.go:392](https://github.com/rohanthewiz/grmob/blob/master/htmlout/tag.go#L392)</small>

### func StackAxes

```go
func StackAxes() map[string]string
```

StackAxes returns a copy of the whole table, for the WASM runtime conformance test, which compares table against table and so cannot go through StackAxisFor one key at a time.

A copy, not the map itself, for the reason Tags returns one: a package-level map is reachable and writable by any importer, and the test deletes from what it is given as it matches rows.

<small>[htmlout/stack.go:131](https://github.com/rohanthewiz/grmob/blob/master/htmlout/stack.go#L131)</small>

### func StackAxisFor

```go
func StackAxisFor(nodeType string) string
```

StackAxisFor returns the flex axis a node type stacks its children along, or "" for a type that is not a stack container.

The "" answer is load-bearing in two ways at the call sites: it is what keeps a Text or a Button that happens to set a container prop from being given a stacking default it never asked for, and it is what a caller reads as "this node is only a flex container if its Style makes it one".

<small>[htmlout/stack.go:120](https://github.com/rohanthewiz/grmob/blob/master/htmlout/stack.go#L120)</small>

### func StackPlacementFor

```go
func StackPlacementFor(align core.StackAlignment) string
```

StackPlacementFor returns the CSS declarations that place a layer.

Total over core.StackAlignment's declared values, centre included. A value the table does not name — which can only be a hand-written string, since the type is closed by its const block — falls back to the centre rather than to nothing, so an unrecognised placement renders as the contract's default instead of leaving a layer wherever a stray flex prop put it.

<small>[htmlout/stack.go:289](https://github.com/rohanthewiz/grmob/blob/master/htmlout/stack.go#L289)</small>

### func StackPlacements

```go
func StackPlacements() map[string]string
```

StackPlacements returns a copy of the whole table, keyed by the placement's string form — the service CrossAxisAligns provides, for the reason it gives: the conformance test in wasm/verify compares table against table, and a caller that could mutate the original would be comparing the runtime against whatever the last test left behind.

<small>[htmlout/stack.go:301](https://github.com/rohanthewiz/grmob/blob/master/htmlout/stack.go#L301)</small>

### func StackTypes

```go
func StackTypes() []string
```

StackTypes returns the stack container types, sorted so a test looping over them reports in a stable order — the same service AlignFallbackTypes and TransparentTypes provide, and for the same reason: a hand-written list in a test is the untracked second copy these tables exist to remove.

<small>[htmlout/stack.go:141](https://github.com/rohanthewiz/grmob/blob/master/htmlout/stack.go#L141)</small>

### func TagFor

```go
func TagFor(nodeType string) string
```

TagFor returns the HTML tag a node type renders as.

Fragment and Theme are not in the table and must not be asked: see transparentTypes. The lookup answers \*which element\*, never \*whether an element\*, and a caller that has not made the transparency decision first gets defaultTag — a box those two node types are not supposed to have.

<small>[htmlout/tag.go:130](https://github.com/rohanthewiz/grmob/blob/master/htmlout/tag.go#L130)</small>

### func Tags

```go
func Tags() map[string]string
```

Tags returns a copy of the whole table, for the callers that must enumerate it rather than query it — the WASM runtime conformance test, which has to compare table against table and so cannot go through TagFor one key at a time, and this package's own test that the exporter agrees with it.

A copy, not the map itself, for the reason InputTypes returns one: a package-level map is reachable and writable by any importer.

<small>[htmlout/tag.go:144](https://github.com/rohanthewiz/grmob/blob/master/htmlout/tag.go#L144)</small>

### func TextAlignFor

```go
func TextAlignFor(align string) string
```

TextAlignFor returns the CSS text-align value an alignment maps to, or "" for one that is absent, unrecognized, or a cross-axis-only Alignment.

It takes a string rather than a core.Alignment for the reason ObjectFitFor, InputTypeFor and TagFor do: the callers that need it most are reading the value back out of a place that has already lost the Go type.

<small>[htmlout/textalign.go:71](https://github.com/rohanthewiz/grmob/blob/master/htmlout/textalign.go#L71)</small>

### func TextAligns

```go
func TextAligns() map[string]string
```

TextAligns returns a copy of the whole table, keyed by the alignment's string form, for the two callers that must enumerate it rather than query it: the WASM runtime conformance test, which compares table against table, and the census test that holds this table to core.TextAlignments().

A copy, not the map itself, for the reason ObjectFits hands out one — both callers delete from what they are given as they match rows.

<small>[htmlout/textalign.go:82](https://github.com/rohanthewiz/grmob/blob/master/htmlout/textalign.go#L82)</small>

### func TransparentTypes

```go
func TransparentTypes() []string
```

TransparentTypes returns the transparent node types, sorted so that a test looping over them reports in a stable order. Exported for the same reason Tags is: the WASM conformance test has to know which types are excluded from the tag comparison, and a hand-written list there would be exactly the untracked second copy this file exists to remove.

<small>[htmlout/tag.go:242](https://github.com/rohanthewiz/grmob/blob/master/htmlout/tag.go#L242)</small>

