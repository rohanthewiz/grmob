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
  `core.AccessibilityRole` — one no tab names, or one marked
  `AccessibilityHidden` is left unwired, and its tab drops the `aria-controls`
  with it. See
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
- **A heading's tier is `aria-level`,** written only alongside
  `role="heading"` and only for 1–6 — ARIA's own scoping, and a drop rather
  than a clamp. See
  [Styling & Theming](../concepts/styling-and-theming.md#accessibilityheadinglevel).

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
filter bar's extraction into `components.Chip` was byte-identical, and now
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
