# Session: closing core gap 3 — `core.MaybeProp`

**Session ID:** session_01Ej9aSqY1yefk9am5LZG367
**Date:** 2026-08-31, ~15:00
**Branch:** master
**Follows:** `2026-0831-1445-palette-roles-and-badge-variants.md`

## Goal

The last of the three core gaps: the accumulate-into-a-slice idiom the backlog
had flagged at `chat/main.go:147` and `mobileapp/app.go:146`. The note said a
`core.MaybeProp(cond, prop)` deletes it.

Only the chat site was still live — `mobileapp:146` had already been migrated
to `components.ListRow` two sessions ago and survives only as a comment
describing the shape it used to have.

## The helper

```go
func MaybeProp(cond bool, prop PropsAndChildren) PropsAndChildren {
	if cond {
		return prop
	}
	return nil
}
```

Four lines. The value is entirely in *why* and in what holds the contract up.

### `core.If` cannot do this job, in two separate ways

**1. `If(false, v)` returns `Fragment()`, which is a real child node.** Inside a
flex container it takes a slot, so it opens a stray `Gap` and shifts a
`Justify`. This was already written down in `chat/main.go` as the reason the
slice existed — the codebase had diagnosed the problem and worked around it
rather than fixing it.

**2. `If` is typed `View -> View`.** There was *no* expression form at all for
an optional `StyleProp` or `BehaviorProp` — "apply this padding only when
selected", "attach `OnClick` only when a handler was supplied". This is the
half the backlog note did not spell out and it is the more valuable half:
`MaybeProp` takes `PropsAndChildren`, so one helper covers all three item
kinds. `mobileapp`'s comment had independently hit this exact wall ("core.If
emits a real child node, so it was not usable for a *style* prop").

Guidance recorded in the doc comment and in `views.md`: `If` where the
alternative is a whole branch of the tree, `MaybeProp` for a single optional
item among siblings.

### Two limits, both deliberate and both documented

**Eager evaluation.** `prop` is an ordinary Go argument — the condition does
not guard it. Free for the prop constructors (`core.Text` returns a
`ComponentFunc`; nothing renders until the container renders it), but not a
substitute for an `if` around an expression that would panic or do real work on
the false path.

**Return type is `PropsAndChildren` (i.e. `any`),** so it only fits the five
container builders. `Text` and `Button` take `...StyleProp` and will not accept
it — and *must not*, since their loops call `Apply` on every element and would
panic on a nil. Worth stating explicitly because "why can't I use this in
`Text`?" is the obvious next question and the answer is a crash, not taste.

## The nil contract needed a guard — this is the finding of the session

`MaybeProp` rests entirely on `containerNode`'s type switch dropping a nil.
That was **accidental**, not stated: the switch had three cases and no default,
so anything unmatched fell through silently. Building a public API on an
unstated fallthrough is how a later "let's add a default case for diagnostics"
quietly breaks it.

So the switch now says both halves out loud:

```go
case nil:
	// A nil item is a contract, not an accident: MaybeProp(false, ...)
	// returns an untyped nil so a dropped prop costs the tree nothing.
default:
	// Anything else silently vanished. Debug mode names the type.
```

### The default case is worth having on its own

`PropsAndChildren` is an alias for `any`, so a container accepts *literally
anything* at compile time and drops what it cannot classify. The symptom is a
style or handler that simply never took effect — no compile error, no runtime
error, nothing in the tree. New concern kind `ConcernUnknownItem`
(`unknown-container-item`), reported in debug mode only:

```
[unknown-container-item] ×3 Row: argument of type core.Style is not a
StyleProp, BehaviorProp or View and was ignored
```

The three realistic causes: a bare `core.Style` where `core.UseStyle(style)`
was meant; a `core.WhenClause` that never reached `MatchBool`; a `*core.Node`
where a `core.View` was wanted.

The two changes reinforce each other, which is the point. `nil` is cased
**ahead** of the default precisely so `MaybeProp`'s false path can never be
mistaken for the footgun it sits next to. Neither change is fully safe alone:
without the default, the nil contract is invisible; without the explicit nil
case, adding the default would have started flagging every `MaybeProp(false,
...)` as a bug.

## Tests — `core/conditionals_test.go` (new, 5 tests)

`MaybeProp`'s whole value is what it does *not* leave behind, so the tests are
mostly assertions about absence.

1. **`TestMaybePropLeavesNoNodeWhereIfLeavesAnEmptyFragment`** — the headline
   difference, both helpers in one test so the contrast is the assertion.
2. **`TestMaybePropTrueKeepsTheItemInPlace`** — the true path is a
   pass-through: the item lands in the position it would have had inline, since
   sibling order is what the renderers lay out and what keyed reconciliation
   diffs.
3. **`TestMaybePropCoversStyleAndBehaviorProps`** — the second half of the
   rationale. Includes the case that actually matters: a false `MaybeProp` must
   leave an *earlier* style prop standing rather than applying a zero value
   over it.
4. **`TestMaybePropWorksInEveryContainer`** — all five builders. They share
   `containerNode` today, so a future container that hand-rolls its own item
   loop is exactly what a Column-only test would miss.
5. **`TestUnknownContainerItemIsReportedButNilIsNot`** — the two halves of the
   switch, asserted together.

### Self-guard

Test 1 asserts the `If(false)` baseline *first*, and fails loudly if `If` ever
stops emitting a child. Same move as `TestVariantDefaultKeepsTheThemePairing`
last session: without it, a future change to `If` would make the whole
rationale test pass vacuously.

### Both guards verified to bite

```
# MaybeProp reverted to returning Fragment() on false
--- FAIL: TestMaybePropLeavesNoNodeWhereIfLeavesAnEmptyFragment
--- FAIL: TestMaybePropWorksInEveryContainer/{Row,Column,Card,Box,List}
        Row has 2 children, want 1

# `case nil:` deleted so nil falls to the default
--- FAIL: TestUnknownContainerItemIsReportedButNilIsNot/nil_is_silent
    MaybeProp's false path raised concerns: [{Kind:unknown-container-item
    Detail:Column: argument of type <nil> ... was ignored Count:1}]
```

## The migration

`chat/main.go`'s `MessageBubble` — the slice, the three appends and the
five-line apology comment collapse into one expression:

```go
core.Column(
	core.UseStyle(bubble),
	core.MaybeProp(!m.Mine(),
		core.Text(m.From, core.FontSize(12), core.FontWeight(core.Bold))),
	core.Text(m.Text, core.FontSize(15)),
)
```

Verified **byte-identical HTML export** for both the mine and theirs cases: a
throwaway `package main` snapshot test in `examples/chat`, run against the new
code and then against the old via `git stash push examples/chat/main.go`, and
diffed. Clean, then deleted. (The example is `package main`, so an external
harness cannot import it — the test has to live in the package briefly.)

## Deliberately not migrated

The seven `components/*.go` builders that also `make([]core.PropsAndChildren,
...)`. Their slices exist to absorb the caller's variadic `Style` slice, which
`MaybeProp` cannot remove — converting their `if x != nil { append }` guards
would be churn at equal line count, not a simplification. `MaybeProp` earns its
place in an *inline* argument list, which is where the slice is otherwise pure
overhead.

Several of those branches also compute a value before appending
(`list_row.go`'s accessibility label with its `", selected"` suffix), which the
eager-evaluation limit rules out anyway.

## Verification

`go build ./...`, `go vet ./...`, full `go test ./...` green.
`examples/todoapp/store.go` remains the only `gofmt` offender, unchanged.

## Files touched

**New**
- `core/conditionals_test.go` (5 tests)

**Modified**
- `core/conditionals.go` — `MaybeProp`
- `core/layout.go` — explicit `case nil:`, `default:` concern, nil contract on
  `containerNode`'s doc comment
- `core/debug.go` — `ConcernUnknownItem`
- `examples/chat/main.go` — the migration
- `examples/mobileapp/app.go` — one clause added to the historical comment so
  it no longer implies the style-prop conditional has no expression form
- `docs/concepts/views.md` — "One optional item: `MaybeProp`" + a debug tip
- `docs/concepts/debug-mode.md` — table row + "Unknown container items"
- `README.md`, `docs/index.md` — helper named alongside `If`/`Match`/`For`

## Backlog after this session

- **All three core gaps are now closed.** Tier 1 widgets are the front of the
  queue: `Button` variants (5 instances), `Screen` scaffold (5).
- **Tier 2:** `InputRow`/composer, `StatTile`, `EmptyState`,
  `SegmentedControl`.
- **`Variant` consumers:** `Alert`/banner is the obvious second (no new core
  work — it calls `Variant.Color` / `Variant.Ink` exactly as Badge does), then
  `Chip{Variant:}`. A `Neutral`/muted variant mapping to Surface is the one
  addition a status set usually wants next.
- **`MaybeProp` has one consumer.** Any new widget with an optional
  style/behavior prop is the natural second, and it is now the reason `Button`
  variants and `Screen` should be cheap to write.
- **Renderer work, none of it blocking, unchanged:**
  1. Proportional flex weights on iOS (custom SwiftUI `Layout`) — lets
     `ProgressBar` move off percentage widths.
  2. `AlignItems: "stretch"` on both renderers — unblocks
     `Separator.Vertical`.
  3. A `ContentMode` prop on `Image` — unblocks avatar images that fill rather
     than letterbox.
- **Still true from four sessions ago:** `ROADMAP.md` lists `UseMemo` and
  `UseReducer` as done; neither identifier exists in the tree.
