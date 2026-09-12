# Package jsonout

```go
import "github.com/rohanthewiz/grmob/jsonout"
```

Package jsonout renders a [core.Node](core.md#type-node) tree to JSON.

It is the smallest of the exporters and the most useful in a test: a tree as a string is something a test can compare, and the output is the same shape the native bridges send across, so a difference visible here is a difference the host would have seen.

For a viewable artifact rather than a comparable one, use htmlout, which renders the same tree as a standalone HTML document.

## Index

- [`func Export`](#func-export)

## Functions

### func Export

```go
func Export(node *core.Node) string
```

<small>[jsonout/export.go:8](https://github.com/rohanthewiz/grmob/blob/master/jsonout/export.go#L8)</small>

