# Session: TabView, the last stacking divergence and the bar behind it

Session: https://claude.ai/code/session_01Mv1iFwxr7pCAdxCFMKT8Pk
Date: 2026-09-04 (follows "block-flow-and-syntax-highlighting", whose
"Still outstanding" list named the first half of this as its first item)

## Ask

Two turns, the second only reachable once the first landed:

1. "Now fix the TabView block flow divergence too."
2. "Now do the tab bar and selection pass."

---

# Part 1 — TabView joins the stack table

## What it was

The previous session added `stackAxes` — the one table saying which node types
lay their children out along an axis whether or not a Style asks — and left
TabView out of it, on the recorded grounds that "neither DOM target has ever
defaulted it to flex, so the two web targets agree about it today, which is
the property this file exists to protect."

That reasoning does not survive contact with the third target. Both natives
build a TabView from a vertical stack — `Renderer.kt` a Compose `Column`
holding a `TabRow` and the page, `Renderer.swift` a SwiftUI `VStack` holding a
hand-rolled bar and the same page — so the two web targets were agreeing on
the wrong layout. One row, `"TabView": "column"`, in Go and in the runtime's
copy.

## The pin that let it go in on evidence

`mobile/verify/TestNativeTabViewIsAColumnStack`. The table row is a *claim
about the two renderers*, so it is checked against them rather than asserted
in prose.

**Positional, not merely present.** Both composites hold a horizontal stack —
the tab bar — inside the vertical one, so "builds a VStack" would pass just as
well on a renderer that had the two the wrong way round, and that renderer
would be drawing the bar beside the page instead of above it. Comparing the
two offsets is what says which is the outer box, and that box's axis is what
the table states. The bar is located by its own construct (`HStack(`,
`TabRow(`) rather than by a second vertical stack, so a missing bar fails as a
missing bar instead of leaving the ordering check with nothing to compare.

`dispatchArm` was reused to bound a whole composite declaration rather than a
dispatch arm; its doc and failure text widened to be true of both. Two new
boundary regexps were needed because a Swift `struct` is bounded by neither
the arm regexps nor `switchlabels_test.go`'s `declStart`, which stops only at
a function.

## Not in the other table

TabView is deliberately absent from `alignFallbackAxes`. That gate is the
types whose native stack reads `Style.Align` for cross-axis placement, and
neither TabView composite does — Compose's `Column` is given no
`horizontalAlignment`, SwiftUI's `VStack` no alignment argument. Scroll is the
standing precedent for a type that stacks and reads no fallback; the two
tables answer different questions and are not required to agree.

---

# Part 2 — the tab bar and the selection

## The real size of the gap

Part 1 fixed the axis. The audit it forced turned up the rest: **neither DOM
target read any of `core.TabView`'s four wire props.** Not `tabs`, not
`selectedIndex`, not `onTabChange`, and not "one child per page" — a TabView
exported as a bare box holding *every* page at once, with no bar and no way to
switch. An app whose navigation is a TabView had no navigation at all on the
web, and its screens stacked one under the other.

`onTabChange` was worse than unread. It fell through the generic `on*` branch,
which derived a `"tabchange"` DOM event, attached a listener nothing could
fire, and marked the slot taken — the same shape `onDismiss` and `onLongPress`
each needed their own branch to escape.

## The markup, identical on both targets

```html
<div data-node-type="TabView" data-node-path="root/1" style="display:flex; flex-direction:column">
  <div data-grmob-chrome="tabbar" role="tablist" data-ontabchange="int_cb_0">
    <button type="button" role="tab" aria-selected="true"  data-tab-index="0">Home</button>
    <button type="button" role="tab" aria-selected="false" data-tab-index="1">Search</button>
  </div>
  <div data-node-path="root/1/0">…</div>
  <div data-node-path="root/1/1" style="…; display:none">…</div>
</div>
```

One callback ID on the bar and an index per tab, because one handler serves
every tab and the argument is what distinguishes them — which is exactly the
shape `core.OnTabChange` has, and the same spirit as the `data-onclick` family:
the ID, not the behavior.

## The bar is chrome, not a node

It carries no `data-node-path`, no patch is ever addressed to it, and it is
marked `data-grmob-chrome`. That mark is load-bearing in three separate
places:

- `chromeOffset` converts a **node** child index into a **DOM** child index,
  which the two positional patch paths need. `add` resolves
  `parent.children[index + chromeOffset(parent)]`; `add-child` derives the new
  child's path from `el.children.length - chromeOffset(el)`. Without the
  shift a page added to a TabView lands in front of the bar and every path
  after it is one slot out of step with the element answering to it.
- `wasm/verify`'s replay walks the DOM back into a flat description and
  compares it with Go's final tree; chrome is skipped whole.
- The offset is **counted, not derived from the node type** — a TabView with
  no `tabs` prop has no bar, and "TabView" alone would answer 1 for an element
  whose first child is really its first page.

It is the same trick a TextGrid row's runs use, one step harder: those spans
sit under a node with no children of its own, so nothing ever had to count
past them.

The replay's skip is narrowed the way the tag table's exemption is: skipping
on the marker *and* asserting the marked element has no path. Skipping on
"has no data-node-path" would also swallow a real node whose path went
missing, which is one of the exact failures the replay exists to catch.

## Hidden, not dropped

Where both DOM targets differ from the natives. The runtime cannot drop a
page — `TargetID`s are positional, so its DOM has to stay isomorphic to the
node tree. `htmlout` could, being a static snapshot with no patches to
address, but an export that silently lost every screen but one is a worse
document and a divergence between the two web targets for no gain.

## The hardest part: the selection is derived state

`styleFromGrMob` is *total* — it assigns every property it manages on every
pass — so an `update-style` landing on a hidden page rewrites its `display` and
reveals it. Three unrelated patch shapes invalidate the selection:

```
update-props on the TabView   a new selectedIndex — which is what a switch IS
update-style on a page        totality overwrites the hiding
add / remove / replace        the set of children changed
```

Rather than teach each of those cases about tabs, `patch()` collects the
elements the batch reached, walks up from each, and recomputes every TabView
at or above them once the batch has landed. Nothing paints in between: a batch
is one synchronous run. `syncTabView` is idempotent and reads only the
element, so calling it more often than strictly needed is safe — that is the
design, not a concession.

**`data-base-display`** is the other half. Restoring a page means putting back
the display the style pass computed, not clearing the declaration: a Column
page cleared to `""` comes back in block flow. It is recorded at the two
places this runtime decides a display — the stack default in `createElement`
and `applyStyle` — and because `applyStyle` is total the record can never go
stale.

## Rebuilt only when the strip changed

`buildTabBar` compares a signature of the labels. `selectedIndex` changes on
every switch, and a switch is exactly when a keyboard user has one of these
buttons focused; rebuilding unconditionally would throw that focus away on
every tab press. The selection is applied by mutating the buttons in place
instead.

## htmlout's one structural change

The hiding decision belongs to the parent while the style attribute is
assembled in the child, so `renderNode` gained an `extraDecl` parameter —
appended last, so it outranks the `display:flex` a stack container gets
unconditionally. It is forwarded through the transparent branch (a Fragment
used as a page has no box, so what the parent meant has to reach the children
standing in for it) and through the Spacer early return, which was the one
node type that would otherwise have silently ignored it.

## Two deliberate omissions

Both would make the web the outlier rather than close a gap:

- **The icon** half of a `core.TabItem` is drawn by no target. Compose's `Tab`
  is built with `text = { Text(label) }`, the SwiftUI bar with a `Text` of the
  label.
- **An out-of-range `selectedIndex` is not clamped**: no tab marked, no page
  shown. That is `children.indices.contains` in Swift, `getOrNull` in Kotlin,
  and the Swift bar's plain `i == selected`.

## What is pinned across the two targets, and what is not

The chassis is authored twice — a declaration list is a Go string on one side
and a property object on the other, and neither target can call into the
other. That is the arrangement the Modal chassis has always had.

What *is* pinned is the half that is a contract rather than a look:
`TestRuntimeDrawsTheSameTabChrome` holds the runtime to the roles, the ARIA
state and the data attributes htmlout writes, and
`TestRuntimeShiftsChildIndicesPastTheChrome` holds it to the two index sites.
A drift in either is silent: the bar still draws, still switches, and simply
stops being the same thing on the two web targets.

## Verified in a browser, both targets

Not just in the harness, which has no CSS engine:

- The **exported HTML** served over localhost: three equal-width tabs, the
  selected one at full opacity with a `currentColor` underline, the other two
  dimmed, one page visible.
- The **live runtime**, mounted in a real page with a stub `GoInvokeCallback`
  that echoes the click back as the `update-props` patch Go would send. A
  click on the third tab moved both the indicator and the visible page — the
  full round trip.

---

## Changes

**Part 1**
- `htmlout/stack.go` — the `"TabView": "column"` row; the "deliberately
  absent" bullet replaced by a section on why it is in, what its row does not
  fix, and why it is not in `alignFallbackAxes`.
- `wasm/grmob-runtime.js` — the same row; two comments that used TabView as an
  example of a non-stack type rewritten.
- `mobile/verify/stacking_test.go` — `TestNativeTabViewIsAColumnStack`, two
  composite-boundary regexps, and `dispatchArm`'s doc and failure text widened.
- `htmlout/stack_test.go` — TabView dropped from the block-flow cases.

**Part 2**
- `htmlout/tabview.go` (new) — the chassis constants, `renderTabView`,
  `renderTabBar`, `tabSelectedIndex`, `tabLabels`.
- `htmlout/export.go` — `renderNode`'s `extraDecl` parameter and its four call
  sites, the Spacer forward, the `case "TabView"` arm.
- `wasm/grmob-runtime.js` — `buildTabBar`, `syncTabView`, `chromeOffset`,
  `syncTouchedTabViews`, the style constants; `onTabChange`/`selectedIndex`
  branches on both prop paths; `baseDisplay` at the two display sites; the two
  index shifts; the end-of-batch sync.
- `wasm/verify/dom.mjs` — a real `insertBefore` (the runtime's `add` path had
  been calling one the harness did not have), and the member census updated.
- `wasm/verify/replay_test.mjs` — `isChrome`, skipping chrome subtrees.
- `htmlout/tabview_test.go` (new, 14 tests), `wasm/verify/tabview_test.mjs`
  (new, 17 tests), `wasm/verify/tabchrome_test.go` (new, the cross-target pin).
- `docs/platforms/wasm.md` (a "Tab views" section), `docs/platforms/exporters.md`,
  `docs/components.md`.

## Still outstanding

- **`element.PrettyHTML` mangles whitespace-significant content.** Unchanged
  from the previous session: worked around in htmlout rather than fixed
  upstream, the dependency being pinned at v0.7.0.
- **A TabView's page has no `role="tabpanel"` and no `aria-controls` wiring.**
  The bar is a well-formed tablist on its own, but the relationship between a
  tab and its panel is not expressed. Adding it means putting ARIA attributes
  on *node* elements, which is a different kind of change from anything in
  this pass — every attribute on a node element is one an author's Style could
  also be asked to set.
- **CameraView stays an overlay** on both natives, which is what it is for.
  Modal is one on Android for the same reason.

## Verification

`gofmt`, `go vet ./...`, `go test ./...`, `sh wasm/verify/run.sh`,
`sh ios/verify/run.sh`, and `:app:compileDebugKotlin` (`--rerun-tasks`) — all
clean. The Android SDK path is not in the environment; the compile was run
with `ANDROID_HOME="$HOME/Library/Android/sdk"`.

Three mutation checks, each confirming a new test fails for the reason it
claims: swapping the expected outer/inner stacks in the native pin, dropping
the `extraDecl` forward from the Spacer path, and the browser round trip
above standing in for the harness's missing CSS engine.
