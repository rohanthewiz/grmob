# Session: the tab set says which region each tab governs

Session: https://claude.ai/code/session_01Mv1iFwxr7pCAdxCFMKT8Pk
Date: 2026-09-04 (follows "tabview-stacking-and-tab-bar", whose "Still
outstanding" list named this as its second item)

## Ask

"Now wire up role=tabpanel and aria-controls."

The previous session left it as a named gap with a reason attached: *"The bar
is a well-formed tablist on its own, but the relationship between a tab and
its panel is not expressed. Adding it means putting ARIA attributes on **node**
elements, which is a different kind of change from anything in this pass —
every attribute on a node element is one an author's Style could also be asked
to set."* That caveat turned out to be the whole shape of the work.

---

## What a tablist does not say

`role="tablist"` around `role="tab"`s, each with `aria-selected`, is a
well-formed strip. It announces "tab, 1 of 4, selected". It says nothing about
**which region of the screen that tab governs** — which is the entire point of
a tab set. A screen-reader user could hear the strip and have no way to know
the content below it was the thing the strip controlled.

That relationship is `aria-controls` (tab → panel) and `aria-labelledby`
(panel → tab), and both are **IDREFs**. So the wiring cannot be expressed at
all without element ids, and ids are document-global.

## The hardest decision: where the ids come from

Two TabViews in one document must not both call their first tab `tab-0`. Three
candidate identities:

| | |
|---|---|
| a per-document counter | unique by construction, but a *different string* on each target — the two web targets would agree only in shape |
| the tab labels | not unique |
| **the node path** | already unique per element, already what every patch is resolved against |

The path won, and the reason is the one this pair keeps returning to: it makes
the ids **identical across the two web targets**, not merely well-formed. A
TabView at `root/1` names its first tab `grmob-root-1-tab-0` in the exported
document *and* in the live DOM. The contract the pair shares is the literal
string.

The uniqueness is not a new claim. It is exactly the uniqueness the addressing
scheme already rests on: if two live elements could share a `data-node-path`,
every patch aimed at either is already going to the wrong one, and duplicate
ids would be the least of it.

Slashes become dashes (`root/1` → `grmob-root-1`) so an id is usable as a CSS
selector fragment without escaping. Nothing is lost — a path is `root`
followed by digits, so no dash can appear in one and no two paths can collapse
onto one scope.

### The cost of that choice, in htmlout

The runtime already had a path on every element. `htmlout` had none — it never
needed one, being a static snapshot with nothing to address. So `renderNode`
grew a `path` parameter, walked by child index exactly as `reconcile.Patch`
builds its TargetIDs. **It is never emitted.** It exists only so a TabView can
derive the same ids the runtime does.

`childPath` is the one place the shape is spelled, so the exporter, the
reconciler and the runtime stay one convention rather than three that happen
to agree.

### The second parent-imposed channel

The previous session added `extraDecl` — a CSS declaration list the *parent*
imposes on a child, because the hiding decision belongs to the TabView while
the style attribute is assembled in the page. The panel wiring is the same
shape of problem with a different payload: attributes, not declarations.

Rather than a fifth parameter, `extraDecl` became a struct:

```go
type imposed struct {
	decl  string   // appended last, so display:none outranks display:flex
	attrs []string // appended to the node's own attribute list
}
```

Both travel through the transparent branch together, and both through the
Spacer early return — which remains the one node type that would otherwise
silently ignore what its parent meant, now for two reasons instead of one.

## The three ways a page opts out

This is where the previous session's caveat about node elements landed. Each
case is one where wiring the page would make the document say something false:

| The page… | Why it is left alone |
|---|---|
| has no tab at its index | a `tabpanel` outside a tab set, with nothing for the `aria-controls` to sit on |
| renders as an element that already has a role (`<button>`, `<img>`, an input) | `role="tabpanel"` **replaces** the role the browser gave it |
| is `AccessibilityHidden` | the author severed the relationship on purpose |

The second is the one worth dwelling on. The whole purpose of this pass is
accessibility, so it must not *cost* any: stamping `tabpanel` onto a `<button>`
page would take away the button role in the name of adding a panel role. The
eligible set is therefore the tags whose implicit ARIA role is `generic` —
`div`, `span`, `pre` — stated as `genericTags` in `htmlout/tag.go`, restated as
`GENERIC_TAGS` in the runtime, and compared by `TestRuntimeGenericTagsMatchGo`
under a plain `go test ./...`. A whitelist rather than a blacklist, because the
safe answer for a tag nobody has considered yet is "not eligible".

**In every opt-out case the tab drops its `aria-controls` too.** A dangling
IDREF is worse than an absent one: a screen reader announcing a tab that
controls a region, and finding no such region, is a lie about the document,
whereas a tab with no `aria-controls` is merely one that has not said what it
governs.

## Deferring to the author twice

The two places an author's intent could have been overridden, both left to
them:

- **A page that names itself keeps its name.** `aria-labelledby` wins over
  `aria-label` in the accessible-name calculation, so writing it
  unconditionally would silently discard a `core.Style.AccessibilityLabel` the
  app author chose. The tab still points *at* the panel; only the naming is
  theirs.
- **A page they hid stays hidden.** Read off the *element* (`aria-hidden`)
  rather than out of the Style, because that is where `applyAccessibility` put
  it and the sync runs long after.

Nothing else collides: no `core.Style` field maps onto `id`, `role` or
`aria-labelledby`.

## The runtime needed no new machinery

The wiring rides on `syncTabView`, the pass the previous session built for the
selection — and the reason is the same reason that pass exists. Every one of
the three patch shapes that invalidates the *selection* also invalidates the
*wiring*:

```
update-style on a page      can set AccessibilityHidden -> no longer eligible
update-props on the TabView a shrinking tabs strip -> a page with no tab
replace                     a <div> page becomes a <button> -> a roled element
```

So it is written **or removed** on every sync, for the reason `styleFromGrMob`
is total: a guarded write would leave a role and a dangling reference standing
after the reason for them was gone. `wireTabPanel` is four `setOrRemove` calls
and nothing else.

One ordering note: `buildTabBar` runs inside `createElement`, *before* the
children exist, so it cannot see the pages. `syncTabView` runs after them on
both the create and the patch path, which is why the wiring lives there and not
in the builder.

## Where the two web targets genuinely differ

`htmlout` has to ask one question the runtime cannot need: **which element
stands in for the page.** It drops the box for a `Fragment` or a `Theme` (the
transparency exemption), so:

- a single-child transparent page — `core.WithTheme` wrapping a screen is the
  realistic case — is wired on the one element standing in for it;
- a multi-child one is not wired at all, because the id would have to be
  written twice (invalid) or nowhere.

The runtime boxes both node types, so page *i* is always exactly the element in
child slot *i*. This is a new instance of the divergence `transparentTypes`
already documents, not a new divergence.

## Two deliberate omissions

- **No `tabindex` on the panel**, which the ARIA authoring practices suggest
  for a panel holding nothing focusable. `tabindex` changes the page's real tab
  order — a behavioral change to an app author's node rather than a statement
  about it — and this pass is deliberately semantic only.
- The **icon** half of a `core.TabItem`, unchanged: drawn by no target.

## Verified against a real app, both targets

Not only in the harnesses:

- `examples/mobileapp` exported through `htmlout`: 8 ids, no duplicates, all 8
  `aria-controls`/`aria-labelledby` references resolve to an element in the
  document.
- The same app's recorded patch transcript replayed through the live runtime
  ends with **exactly those eight ids** — `grmob-root-0-1-tab-0` …
  `grmob-root-0-1-panel-3`. That is the cross-target id equality demonstrated
  end to end on a real tree, rather than asserted.

No browser pass this time, and the reason is stated rather than skipped: the
change writes no CSS, and the harness models attribute bookkeeping faithfully.
The previous session's browser round trip was needed because that one turned on
the cascade.

---

## Changes

- `htmlout/export.go` — the `imposed` struct replacing the `extraDecl`
  parameter; the `path` parameter and `childPath`; both channels forwarded
  through the transparent branch and the Spacer early return; the six call
  sites.
- `htmlout/tag.go` — `genericTags`, `IsGenericTag`, `GenericTags()`.
- `htmlout/tabview.go` — `tabScope`, `tabID`, `panelID`, `tabPanelBoxes`,
  `tabPanelBox`, `tabPanelAttrs`; `renderTabView` and `renderTabBar` widened;
  two new sections in the file header.
- `wasm/grmob-runtime.js` — `GENERIC_TAGS`, `tabScope`, `tabId`, `panelId`,
  `canBeTabPanel`, `wireTabPanel`; `syncTabView` restructured around the tab
  count and the scope.
- `htmlout/tabview_test.go` — 12 new tests plus the `panelTag` helper;
  `tabTags` rewritten to filter on `role="tab"` rather than pin a tab's
  attribute order.
- `wasm/verify/tabview_test.mjs` — 10 new tests.
- `wasm/verify/tabchrome_test.go` — `TestRuntimeWiresTabsToPanels`,
  `TestRuntimeGenericTagsMatchGo`.
- `wasm/verify/replay_test.mjs` — an id-uniqueness invariant at the end of every
  transcript.
- `docs/platforms/wasm.md` (the markup sample, the wiring section, the opt-out
  table), `docs/platforms/exporters.md`, `docs/components.md`.

## Still outstanding

- **`element.PrettyHTML` mangles whitespace-significant content.** Unchanged
  across three sessions now: worked around in `htmlout` rather than fixed
  upstream, the dependency being pinned at v0.7.0.
- **No roving `tabindex` on the bar.** Arrow-key navigation between tabs is the
  other half of the ARIA tab pattern and is not implemented; the tabs are
  ordinary buttons in the document's tab order. Unlike the panel `tabindex`
  above, this one is chrome the exporter owns, so it would not be reaching into
  an author's nodes — it is simply a separate piece of work.
- **A `remove-child` followed by an `add-child` can duplicate a node path.**
  Pre-existing and untouched: paths are assigned at render and not renumbered
  on removal, so the surviving siblings' paths go stale. It already breaks
  patch addressing, and now would also duplicate an id. Fixing it means
  renumbering on removal, which is a change to the addressing scheme.
- **CameraView stays an overlay** on both natives, which is what it is for.

## Verification

`gofmt`, `go vet ./...`, `go test ./...`, `sh wasm/verify/run.sh`, and
`sh ios/verify/run.sh` — all clean. The Android Kotlin compile was **not** run
and does not apply: no Kotlin or Swift file was touched.

Four mutation checks, each confirming a new test fails for the reason it
claims: writing `aria-controls` unconditionally (caught by the roled-page and
aria-hidden tests), replacing the path-derived scope with a constant (caught by
the literal-id and nested-scope tests), dropping the `IsGenericTag` guard, and
dropping the imposed-attrs forward from the transparent branch.
