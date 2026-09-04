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
  `SafeArea` or `List` exports as a flex container along its own axis, whether
  or not its `Style` asks — the same default the WASM runtime plants, from the
  same table (`htmlout/stack.go`). Without it a container carrying nothing but
  padding exported as a block-flow `<div>`, so its children ran down the page
  here and across it on every other target.

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

`examples/todoapp/chip_migration_test.go` uses exactly this to prove the
filter bar's extraction into `components.Chip` was byte-identical.

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
