# Exporters

Two packages turn a rendered `Node` tree into text — for previews, tests,
golden comparisons, and tooling. Neither is a runtime: they take a tree and
return a string.

## htmlout — HTML export

```go
ctx := core.NewContext()
ctx.BeginRenderPass()
node := myView.Render(ctx)
html := htmlout.ExportHTML(node)
```

`ExportHTML` walks the tree onto an
[element](https://github.com/rohanthewiz/element) builder and returns
indented HTML. Properties worth relying on:

- **User content is escaped.** Text content and labels are entity-escaped;
  attribute values (input `value`/`placeholder`, image `src`) are
  quote-escaped. `core.Text("<script>…")` exports as inert text — the
  exporter is safe to point at untrusted data.
- **Callback IDs are preserved** as `data-onclick` / `data-onchange` /
  `data-ontoggle` attributes — the same contract the WASM runtime reads —
  so exported HTML also documents the event surface of a tree.
- **Deterministic output**: stable attribute order and formatting, which is
  what makes string comparison between exports meaningful.
- **Containers are stacks.** A `Row`, `Column`, `Card`, `Box`, `Scroll`,
  `SafeArea`, `List` or `TabView` exports as a flex container along its own
  axis, whether or not its `Style` asks — the same default the WASM runtime
  plants, from the same table (`htmlout/stack.go`). Without it a container
  carrying nothing but padding exported as a block-flow `<div>`, so its
  children ran down the page here and across it on every other target.
- **A `ZStack` is an overlay,** and the one container that is not a flex one:
  a single-cell grid, centred on both axes, with every child imposed into row
  1 / column 1. A grid rather than absolute positioning because an
  out-of-flow child contributes nothing to its parent's size, and both natives
  size an overlay to its largest child. The container props that promote any
  other box to flex (`Gap`, `JustifyContent`, `AlignItems`, `FlexDirection`)
  are inert here and deliberately do not promote it — that would cost it the
  overlay outright. The same imposed declaration carries each layer's
  `core.StackAlign` as a `justify-self`/`align-self` pair — the one thing the
  channel writes that differs from sibling to sibling — including for the
  layers that ask for nothing, since `align-self` is a property a flex child
  can set for itself and an unstated layer would otherwise be movable by it.
  See [WASM — The overlay](wasm.md#the-overlay).
- **A `Select` builds its own options.** They come from the `options` prop
  rather than from child nodes, the chosen one carrying `selected="selected"`
  — `<select>` has no value attribute, so the selection lives on the options.
  A value matching nothing marks nothing, which is what a live `<select>` does
  with an out-of-list value. Both halves of every option are escaped: the
  value through the attribute path, the label through the text one. Which
  options share an `<optgroup>` is not this exporter's decision but
  `core.SelectMenuSections`', the one statement of the rule that all four
  renderers follow — an `<optgroup>` per section that names a heading, and the
  options written straight into the `<select>` for the section that does not.
  A run marked `GroupDisabled` becomes `<optgroup disabled>`, which is the one
  thing this target can say that the two natives cannot: it greys the heading
  as well as refusing the options. The per-option `disabled` is still written
  beside it, because `core` resolves a disabled run onto its items for the
  renderers that have no section-level control at all.
- **A `TabView` gets its bar and its selection.** The `tabs` prop becomes a
  `role="tablist"` strip of `role="tab"` buttons ahead of the pages, carrying
  `data-ontabchange` and a `data-tab-index` per tab in the same spirit as the
  `data-onclick` family — the ID and the argument, with the wiring left to
  whatever loads the document. Pages other than the selected one are hidden
  with `display:none` rather than dropped, so the export still holds every
  screen. Each tab also carries an `id` and an `aria-controls` naming the page
  it governs, and that page a matching `role="tabpanel"` and `aria-labelledby`
  — the ids derived from the node path (`grmob-root-1-tab-0`), so they are the
  *same strings* the WASM runtime writes for the same tree rather than merely
  the same shape. A page whose element already has a role — from the browser
  (`<button>`, `<img>`, an input) or from the author's own
  `core.AccessibilityRole` — one carrying its own `core.AccessibilityID`, one
  no tab names, or one marked `AccessibilityHidden` is left unwired, and its
  tab drops the `aria-controls` with it. `core.RoleGroup` is the one authored
  role that is *not* theft to replace, since a `tabpanel` says everything a
  group says and one thing more — and it has to be, because the exporter
  supplies a group to any named page whether the author asked or not.
  `core.RoleTabPanel` is *not* an exception to the authored-role rule: a page
  that roles itself has taken the slot, and a page with a role and no id is a
  panel nothing points at. A wired panel also carries `data-grmob-panel`, the
  marker that says the role is the framework's rather than the author's; it has
  no reader here, and is written so the two web targets emit one document —
  the runtime is where it does work, since until `core.RoleTabPanel` existed
  that target told its own writes from an author's by the *value*, which made
  the absence of the constant load-bearing. See
  `htmlout/tabview.go` and [WASM — Tab views](wasm.md#tab-views), which draws
  the same chrome live and states the reasoning for the pair.
- **A `Modal` is a dialog.** The overlay carries `role="dialog"` and
  `aria-modal="true"` alongside its fixed-position chassis, because
  `core.ModalNode` has no `Style` for a `core.AccessibilityRole` to ride on and
  both natives get the same semantics free from their platform dialog. A closed
  modal is `display:none`, so the claim never reaches a reader it could
  mislead; an author's own role on a hand-built Modal node wins, the way their
  style already outranks the chassis. `htmlout.CarriesOwnRole` is how the tab
  wiring knows to leave such a page alone rather than writing a second `role`
  onto it.
- **A `Switch` is a switch.** HTML has no switch element, so `core.Switch`
  exports as `<input type="checkbox">` with HTML's own `switch` attribute (which
  Safari draws as a track and a thumb, and other engines ignore, drawing the
  box) and `role="switch"`, which is what a reader announces on every browser
  either way. The role comes from the node type, like a Modal's — both are in
  `htmlout.ownRoles`, both supply a default an author's `AccessibilityRole`
  outranks, and neither is a `core.Role`, because the natives announce their own
  platform control. A bare `switch` attribute is spelled `switch="switch"` here
  for the reason `checked="checked"` is: `element` emits `key="value"` pairs.
- **A map exports as a placeholder that has not lost its region.**
  `core.MapView` has no engine in a static document and no tiles to fetch, so it
  exports as a `<div>` — as `CameraView` does — carrying
  `data-lat`/`data-lng`/`data-zoom`, and each `core.Marker` child as a childless
  `<div>` with `data-marker-id`, its coordinates and its title. A grey box that
  does not say where it was pointing is a worse snapshot than one that does, and
  it makes the export *upgradeable*: the WASM runtime reads those same attributes
  off the same div, so a page that loads Leaflet and wires the exported callback
  IDs turns the snapshot into the live node. See
  [WASM — Live maps](wasm.md#live-maps).
- **A named container is given `role="group"`.** ARIA prohibits an accessible
  name on the `generic` role a `<div>` and a `<span>` carry, and browsers prune
  it, so an `AccessibilityLabel` on any layout node was announced by both
  natives and by nothing on either web target. A node with a name, no role of
  its own and a generic tag is written `role="group"` — the smallest role that
  makes a name legal. An author's own role always wins. See
  [Styling & Theming](../concepts/styling-and-theming.md#rolegroup-and-the-one-role-you-get-without-asking).
- **An `id` and an `aria-controls`** come from `core.Style.AccessibilityID` and
  `core.Style.AccessibilityControls`, verbatim in both directions, and nothing
  checks that the target of one exists — an export is a snapshot of one tree
  and has no index of the document it lands in. A dangling IDREF is inert, the
  same trade `aria-description` makes — and it is now *reported* rather than
  merely tolerated: `core.SetDebugMode` walks the finished tree, which is the
  one place a whole document is visible, and flags a reference nothing answers
  to, an id claimed twice, and an id that is not a usable HTML id. See
  [State & Hooks](../concepts/state-and-hooks.md#debug-mode). They are what
  lets a hand-built tab strip say which region each tab governs, together with
  `core.RoleTabPanel` on the region itself; see
  [Styling & Theming](../concepts/styling-and-theming.md#accessibilityid-and-accessibilitycontrols).
- **A heading's tier is `aria-level`,** written only alongside
  `role="heading"` and only for 1–6 — ARIA's own scoping, and a drop rather
  than a clamp. See
  [Styling & Theming](../concepts/styling-and-theming.md#accessibilityheadinglevel).
- **A nested item's depth is `aria-level` too,** written alongside
  `role="listitem"` or `role="row"` and with no ceiling. One exporter function
  (`ariaLevel`) switches on the role, so the two `core.Style` level fields are
  mutually exclusive by construction rather than by a precedence rule. See
  [Styling & Theming](../concepts/styling-and-theming.md#accessibilitynestinglevel).
- **A control state is `aria-selected` / `aria-pressed` / `aria-expanded`,**
  each written only for the roles ARIA scopes it to. Two `core.Style` fields
  and three attributes: `ariaSelected` picks between the first two by role
  (selection is one of a set, pressed is a toggle answering for itself), and
  `ariaExpanded` answers a *third* role list that is neither of theirs — it
  drops `option` and adds `link` and `listbox`. A `core.Button` carries all
  three with no role at all, the node type being one, which is what lets a
  `comps.Chip` be pressed and an `comps.Accordion` header be a
  disclosure. There is no `role="group"`-shaped rescue for either state, and
  that asymmetry with the name is deliberate. See
  [Styling & Theming](../concepts/styling-and-theming.md#accessibilityexpanded).
- **A composite's axis is `aria-orientation`,** written for the three roles
  ARIA defines it on that `core.Role` carries — `listbox`, `tablist` and
  `toolbar` — with the node's own layout axis as its value and ARIA's per-role
  default as the fallback for a role on something that is not a stack. It is
  pure semantics and so belongs on both web targets equally: an export of a
  vertical tab strip described it exactly as wrongly as the live one did, since
  ARIA's default for a `tablist` is horizontal. `htmlout/orientation.go` is the
  Go authority both targets read. See
  [WASM — Which way a composite runs](wasm.md#which-way-a-composite-runs).
- **A valued control's position is the `aria-value*` family,** written only
  alongside `role="progressbar"` — the one role of the six ARIA scopes it to
  that this vocabulary carries. Each of the four is written only when stated,
  so a bare position reads as a percentage over ARIA's implicit `0..100`, and a
  role with no range at all is ARIA's own spelling of an *indeterminate* bar
  rather than an omission to be defaulted away. `core.Slider` is deliberately
  outside: it exports as `<input type="range">` and states its own range. See
  [WASM — The value of a valued control](wasm.md#the-value-of-a-valued-control).
- **A form control is told it has no border** when the style declares none.
  The border guard is "a width *and* a color" on all four targets; emitting
  nothing on the web left the user agent's own rule standing, which is what
  gave `comps.Button`'s ghost emphasis an outline the natives never drew.
  `htmlout.ResetsUABorder` is the set, keyed by **node type** rather than by
  tag: `Button`, the three text inputs, `TextArea` and `Select` are in, and
  `Checkbox`, `Switch` and `Slider` — which share `<input>` with the text
  fields — are pointedly out, because the browser draws those controls in their
  entirety and their border *is* the control. See
  [WASM — The user-agent border](wasm.md#the-user-agent-border-and-the-third-value-totality-needs).

- **No `tabindex`, on any node, ever.** A `listbox` and a `tablist` are ARIA
  *controls*, and the pattern each one names includes a roving `tabindex` that
  puts the widget's one tab stop on its active member. The WASM runtime writes
  that and moves it with the arrow keys; this exporter deliberately writes
  neither half. A roving tab stop without the handler that moves it is strictly
  worse than nothing — it takes every member but one out of the tab order and
  supplies no way to reach the rest — so `tabindex` is behaviour rather than
  semantics, and this is not a runtime. Every *semantic* half of both patterns
  (the roles, `aria-selected`, the `aria-controls` wiring) is written here as
  usual, which is exactly what lets the runtime supply the rest without a new
  prop. `wasm/verify/keynav_test.go` holds the line in both directions. See
  [WASM — Composite widgets are operable](wasm.md#composite-widgets-are-operable).

### Testing with htmlout

The high-leverage pattern is **equivalence pinning**: render two trees on
fresh contexts and compare their exported HTML. Fresh contexts matter —
callback IDs are per-pass sequence numbers, so two trees that register the
same handlers in the same order carry identical IDs, and any diff in the
output is a real structural or style difference.

```go
render := func(v core.View) string {
    ctx := core.NewContext()
    ctx.BeginRenderPass()
    return htmlout.ExportHTML(v.Render(ctx))
}
if got, want := render(newImpl), render(oldImpl); got != want {
    t.Errorf("refactor changed rendered output:\n%s\nvs\n%s", got, want)
}
```

`examples/todoapp/chip_migration_test.go` uses exactly this: it proved the
filter bar's extraction into `comps.Chip` was byte-identical, and now
holds the widget's output against the same bar written by hand.

## jsonout — JSON export

```go
jsonout.Export(node)
```

Serializes the tree as JSON — the same shape `render.Manager.RenderInitial`
returns. Useful for inspecting what a renderer actually receives, and for
snapshot-style assertions where you care about the tree rather than its
HTML projection.

## Which to use when

| Need | Use |
|---|---|
| Human-readable preview of a view | `htmlout.ExportHTML` |
| Prove a refactor didn't change output | `htmlout` equivalence pinning |
| Inspect exact props/styles a renderer sees | `jsonout.Export` |
| Full render-loop behavior (events, diffs) | Not an exporter — drive `render.Manager` (see [Getting Started](../getting-started.md#1-drive-it-from-a-test-no-simulator-no-browser)) |
