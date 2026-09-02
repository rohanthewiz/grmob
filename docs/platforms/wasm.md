# WebAssembly

The WASM target runs the same app in a browser: Go (compiled to
WebAssembly) renders and diffs; a small JS runtime applies patches to the
DOM. It is the fastest way to *see* an app during development — no
simulator, instant reload.

## Building

The entry point is the `wasm` package, which mounts a registered app (it
imports the app package for its `init` side effect — edit the import to
switch apps):

```bash
GOOS=js GOARCH=wasm go build -o main.wasm ./wasm
```

Serve `main.wasm` alongside Go's `wasm_exec.js` (from
`$(go env GOROOT)/lib/wasm/`) and a host page.

## The host-page contract

The Go side registers a `GrMobWASM` global with four functions:

| Function | Purpose |
|---|---|
| `GrMobWASM.RenderInitial()` | Mounts (or re-mounts) the app; returns the full tree JSON. Re-mounting closes the previous manager first, so timers from the old instance can't leak |
| `GrMobWASM.ReceiveEvent(id, payloadJSON)` | Delivers a user event: `payloadJSON` is `{"value": ...}` and the value's type picks the callback kind |
| `GrMobWASM.RenderAgain()` | Re-renders and returns the diff — the polling path |
| `GrMobWASM.IsDirty()` | Whether state changed since the last render — poll this to know when `RenderAgain` is worth calling |

And it looks for one global the page provides:

- **`GrMobApplyPatches(patchesJSON)`** — if defined as a function at mount
  time, async state changes (timers, goroutines) are **pushed** to it as
  patch JSON, on the state write's own schedule.

  The shipped `wasm/grmob-runtime.js` always defines it (at page level,
  before the module is instantiated, so the host's startup check finds it),
  so every page on the shipped runtime is on the push path. It is optional
  only in the *protocol* sense: a hand-rolled host may omit it and fall back
  to the `IsDirty` poll, and nothing is lost either way — the manager never
  consumes a diff unless a listener is attached.

  The fallback is a genuine downgrade, not just extra work. The poll rides
  `requestAnimationFrame`, which is fully suspended in a hidden tab, so a
  `UseInterval` clock freezes the moment the tab loses visibility even
  though the Go ticker keeps running. The push channel does not.

```mermaid
flowchart LR
    subgraph page["Host page (JS)"]
        RT["runtime.js<br/>mount · apply patches · wire DOM events"]
    end
    subgraph go["main.wasm (Go)"]
        W["GrMobWASM<br/>RenderInitial · ReceiveEvent<br/>RenderAgain · IsDirty"]
        M["render.Manager"]
    end
    RT -->|"ReceiveEvent(id, payload)"| W
    W --> M
    M -->|"push: GrMobApplyPatches(json)"| RT
    RT -->|"poll: IsDirty → RenderAgain"| W
```

Event wiring on the DOM side uses the callback-ID attributes the tree
carries (`data-onclick`, `data-onchange`, `data-ontoggle`): the runtime
listens for interactions, reads the ID, and calls `ReceiveEvent` with it.


**Host events.** The runtime reports host→app traffic that answers no
callback — the audio player's status ticks — through `GrMobWASM.HostEvent(name,
payloadJSON)`, which `wasm/main.go` installs beside `ReceiveEvent`. A page
that copies the runtime needs nothing more: the `"audio"` system event is
handled inside `grmob-runtime.js` (`GrMob.audio`, an `HTMLAudioElement` plus
the Media Session API), and consumers' state writes reach the screen through
the push channel. See [Native — Audio](native.md#audio) for the command and
status shapes.

### Node types and tags

Which element a node becomes is one table, stated once in Go
(`htmlout/tag.go`) and restated in `grmob-runtime.js` because the runtime is
the side that calls `createElement`. `Text` is a `<span>`, `Button` a
`<button>`, `Image` an `<img>`, `TextArea` a `<textarea>`, the four form
inputs an `<input>`, and every container — `Row`, `Column`, `Card`, `Box`,
`Scroll`, `SafeArea`, `List`, `Modal`, `TabView`, `Spacer`, `CameraView` — a
`<div>`. What distinguishes a `Row` from a `Column` is the flex declarations,
not the element, which is why the runtime keeps the Go type in
`data-node-type` instead of reading it back off the tag.

The two copies are compared by `TestRuntimeTagsMatchGo` in `wasm/verify`, the
same way the `<input>` type table below is, so a row added on one side fails
`go test ./...` until it is added on both.

**One deliberate divergence.** `Fragment` and `Theme` are grouping nodes with
no box of their own, and `htmlout` renders them transparently — their children
land directly in the parent — as do both natives. This runtime boxes them in a
`<div>`, because patches are addressed positionally (`TargetID` is
`"root/1/0"`, resolved against the `data-node-path` attributes written while
walking `node.Children`), so its DOM has to stay isomorphic to the node tree.
The cost is real: inside a flex parent that `<div>` becomes the single flex
item and swallows the gap and alignment meant for the children. Closing it
means teaching the addressing scheme about nodes with no element. The
divergence is named in `htmlout/tag.go` and pinned by the same test, so it
reads as a decision rather than as drift.

### Form controls

Four Go node types share the `<input>` tag, so the runtime writes a `type`
attribute to tell them apart — the only thing that makes a checkbox draw as a
checkbox rather than a text box:

| Node type | Rendered as | State prop |
|---|---|---|
| `Input` | `<input type="text">` | `value` |
| `InputPassword` | `<input type="password">` | `value` |
| `NumericInput` | `<input type="number">` | `value` |
| `Checkbox` | `<input type="checkbox">` | `checked` |
| `TextArea` | `<textarea>` | `value`, `rows` |

Go states that table once, in `htmlout/inputtype.go`; the runtime restates it
in JavaScript because it is the side that actually sets the attribute and
cannot call into Go to ask. The two are not kept in step by hand — a Go test
in `wasm/verify` parses the runtime's literal out of `grmob-runtime.js` and
compares it against `htmlout.InputTypes()`, so a change to either side fails
`go test ./...` until it is made to both.

All of the runtime's lookup tables — tags, `<input>` types, the `object-fit`
and `text-align` values below, and the cross-axis pair that follows them — are
parsed by the same helper, which is why they are written in the same shape: a
flat object literal in a named function, subscripted by that function's own
argument, with a `|| "<fallback>"` default that the parse checks too.

### Image content modes

`core.ContentMode` maps onto CSS `object-fit`: `fit` → `contain`, `fill` →
`cover`, `stretch` → `fill`, `center` → `none`. `htmlout/objectfit.go` is the
authority and `TestRuntimeObjectFitsMatchGo` pins the runtime's copy to it.

Go's table holds the bare value rather than the whole declaration, because
that is the half the two sides share: `htmlout` joins `object-fit:` onto it
for a style attribute, the runtime assigns it to `el.style.objectFit`.

An absent or unrecognized mode yields `""`, and the runtime assigns that,
**clearing** the property — which is what a patch removing an Image's
`contentMode` needs, so the image falls back to the browser's default instead
of keeping the last mode it was handed.

Coverage is checkable here in a way the tag table's is not: `ContentMode` is a
named type with four declared constants, and `core.ContentModes()` — itself
pinned to that `const` block by a test that reads the file's syntax tree —
gives `TestObjectFitsCoversEveryContentMode` a list to check against. The tag
table has no equivalent, because node types are string literals scattered
across core's construction sites.

That same list now holds all four renderers, not just this pair. The natives
map `ContentMode` onto SwiftUI and Compose vocabularies with no CSS in them, so
they cannot be compared as tables; `mobile/verify/contentmode_test.go` reads
their `switch`/`when` arms out of the source and checks coverage alone. See
[Native platforms](native.md#contentmode-on-image). (`replay_test.mjs` holds a third
copy on purpose: a conformance test has to state the rule independently, or
it only proves the implementation agrees with itself.)

### Text alignment

`core.Alignment` maps onto CSS `text-align`: `start` → `start`, `center` →
`center`, `end` → `end`, `justify` → `justify`. `htmlout/textalign.go` is the
authority and `TestRuntimeTextAlignsMatchGo` pins the runtime's copy to it.

This was the first table added to **close** a gap rather than to pin a copy
that already existed (the cross-axis pair below is the second). Until it did,
this runtime did not read `style.Align` at all, in any form — so every
`core.Align` on the web target was silently dropped, while `htmlout` emitted a
declaration for three of the six values and both natives set one. Four
renderers, three behaviors, and one of them was "nothing".

Only four of the six `Alignment`s are in the table. `AlignStretch` and
`AlignBaseline` name a cross-axis placement rather than a text alignment, and
CSS `text-align` has no such keyword; they reach the property through
`Style.Align`'s *other* role — the fallback a vertical-stacking container
reads when `AlignItems` is unset, the next section's table — and fall through
to `""`, which clears it.
`core.TextAlignments()` is the list that says so, and both natives are held to
the same one (see [Native platforms](native.md#alignment-justifycontent-and-alignitems)).

`start` and `end` are CSS's direction-aware keywords, matching the spelling
both natives use (SwiftUI `.leading`/`.trailing`, Compose
`TextAlign.Start`/`.End`). The exporter originally emitted the physical
`left`/`right`, which rendered identically in LTR documents but left-aligned
in RTL locales while both natives trailing-aligned — from the same
`core.AlignStart`. No table comparison can see which spelling is right: the
two DOM copies agree with each other under either one, so the choice lives in
`htmlout/textalign.go`'s doc and here, not in a test. The table maps every
text alignment to itself, but it still earns its keep as a filter — the
identity must not extend to the two cross-axis values.

`justify-content` and `align-items` themselves need no table. Core's spellings
*are* the CSS ones, so both DOM renderers pass them through verbatim and
neither can be wrong about a value it never interprets.

### The cross-axis fallback

`Style.Align`'s second role has a pair of tables of its own, and they closed
the last alignment behavior the DOM pair did not share with the natives. When
`AlignItems` is unset on a vertical-stacking container — `Column`, `Card`, or
`List` — both natives fall back to `Align` for cross-axis placement
(`crossAxisValue` in `Renderer.swift`, the `alignItems.ifEmpty { align }`
reads in `Renderer.kt`), and until these tables existed neither DOM target
did: `Align: "center"` centered the children on device and only the *text* on
the web, and `Align: "stretch"` filled rows on device while the web agreed
only wherever block flow happened to produce the same picture.

`htmlout/crossaxis.go` is the authority for both halves. `crossAxisAlignFor`
maps the four cross-axis `Alignment`s onto the `AlignItems` spellings
(`start` → `flex-start`, and so on), because the fallback means "behave as if
that `AlignItems` had been set" — its census holds the values to exactly
`core.AlignItemsValues()`. `alignFallbackAxisFor` is the gate saying which
node types consult the fallback at all: exactly the three containers the
natives read it for, and pointedly not `Row`, whose vertical cross axis
`Align` has never applied to on any target. `TestRuntimeCrossAxisAlignsMatchGo`
and `TestRuntimeAlignFallbackAxesMatchGo` pin the runtime's copies, and a
source pin holds `styleFromGrMob` to actually reading them, with `AlignItems`
taking precedence.

`justify` and `baseline` have no rows. No native cross-axis dispatch answers
for either (`baseline` falls through to start-packing), so a row — and CSS
`align-items` genuinely has a `baseline` keyword someone could be tempted to
"complete" the table with — would move two targets out of four.

`checked` and `rows` are set as element *properties*, not attributes. A
`checked` attribute is only the control's default state — the browser stops
consulting it the moment the user touches the box — and the live property is
what Go is describing. `rows` is limited to positive numbers in the DOM, so
a non-positive count leaves the browser's own default rather than being
assigned; `core.TextArea` always supplies a positive one.

## Testing without a browser

`wasm/verify/run.sh` is the WASM analog of `ios/verify`, and needs only Go
and Node — no npm, no lockfile, no `node_modules`, no network.

It does two things. `gen.go` drives real example apps through
`render.Manager` and records the initial tree, every patch batch, and the
final tree; Node then mounts that transcript through the **actual**
`wasm/grmob-runtime.js` (loaded with `node:vm` against a minimal DOM in
`dom.mjs`), applies the batches, walks the resulting DOM back into a tree and
compares it with Go's final render. Alongside it, unit tests cover the
per-element logic no transcript reaches — the return key's Enter filter, the
void envelope it sends, `enterkeyhint`, the form-control types and state
above, and the focus command's frame deferral and epoch guard.

What it cannot answer is anything that needs real rendering: whether
`enterkeyhint` actually relabels a soft keyboard, whether `focus()` opens
one, or anything about layout. Those still need a browser, exactly as the
iOS view layer still needs a simulator.

```
$ sh wasm/verify/run.sh
..................................
```

## Permissions

Hardware permission requests (camera, microphone, geolocation) route through
an optional `GrMobRequestPermission(name, callback)` page global, letting
the host page bridge to browser permission APIs.

## Same engine, same rules

The WASM runtime drives the very same `render.Manager` the native shells
use — pass boundaries, callback purging, and [debug mode](../concepts/debug-mode.md)
all behave identically. An app that runs clean in the browser preview is
running the same Go code it will run on the phone; only the renderer
differs.
