# N-083: `core.HBox`, a Row with no theme base

Session: `ca6ca295-dc54-4e57-95ba-f0c109e7604b`
**Date:** 2026-10-07 01:57 · **Branch:** master (04aec20 → one commit with this doc)

## Ask

N-083 from the next-list, pasted in from the cats-todo backlog: the theme's
Row (8/16) and Column (12/16) padding lands on every plain stack, which is
right for a screen's outer column and wrong for nearly every nested one, so
the tutorial cancels it with `core.Padding(0)` all over. Marked "value low
(API decision)". The 2026-09-29 recommendation was to decline, for two
reasons: changing the theme default moves every app's layout, and a second
paddingless constructor would double the stack API.

## What the premise check found

- **The vertical half already existed.** `core.Box` is documented as "a
  Column with no theme base". Only a paddingless Row was missing, so the
  "doubles the stack API" argument was half wrong.
- **The theme base is applied in Go, not by node type.** `Row` and `Column`
  pass `ctx.Theme().Components.Row` / `.Column` as the base style to
  `containerNode`, and it travels as ordinary inline style on the node. No
  renderer (Compose, SwiftUI, htmlout, the wasm runtime) looks a theme up by
  type. (`examples/tutorial/split.go` already recorded the related fact that
  the inline style beat the host page's CSS.)
- **The counts were bigger than the item said, but mostly not on stacks.**
  A grep found 256 non-test Go lines with `Padding(0)`, 212 of them in comps,
  and 42 in examples/tutorial. A throwaway go/ast counter (in the scratchpad,
  not committed) counted `core.Row`/`core.Column` calls with a literal
  `Padding(0)` argument:

  | area              | Row (zeroed / total) | Column (zeroed / total) |
  |-------------------|----------------------|-------------------------|
  | comps             | 20 / 98              | 22 / 97                 |
  | examples/tutorial | 15 / 73              | 7 / 142                 |
  | examples (other)  | 0 / 6                | 2 / 15                  |

  That makes 66 of about 435. Slice-built argument lists (`append(...,
  core.Padding(0))`, `paneStack`) are not seen by it, so the true figure is
  somewhat higher.

## Decision

Asked the user to choose between four options:

1. decline, with the counts corrected;
2. add a paddingless Row, additive only;
3. add one and migrate the `Padding(0)` sites;
4. change the theme default.

They chose **2**: add the Row twin of `Box`, and migrate nothing.

## What landed

- `core/layout.go`:
  - `HBox(stylePropsAndChildren ...PropsAndChildren) View` is
    `containerNode(ctx, "Row", Style{}, …)`. It emits an ordinary **`Row`
    node**, so every per-type Row rule on every target applies unchanged: flex
    direction, cross-axis fallback, min-content, labelled-container sets,
    `RowOfferFillers`. A new node type would have needed an arm in each of
    those tables on each target. The doc comment explains why `Box` has a type
    of its own anyway: it began as an overlay on the natives. It also carries
    a small diagram:

        Row(...)  ──► Node{Type: "Row", Style: theme.Components.Row ⊕ props}
        HBox(...) ──► Node{Type: "Row", Style: Style{}             ⊕ props}

  - `Row` and `Column` got their first doc comments, each naming its
    paddingless twin (`HBox` and `Box`).
- `core/hbox_test.go`:
  - `TestHBoxIsARowNode` checks the node type, the children, a style prop and
    an OnClick that fires.
  - `TestHBoxCarriesNoThemeBase` checks that no theme padding arrives, that a
    control `Row` still gets the theme's inset, and that `HBox(Padding(4))`
    still takes the caller's padding.
- Docs:
  - `docs/api/*` regenerated (`go run ./internal/apidoc/gen`): 696 exported
    functions and methods.
  - `docs/concepts/views.md`: an `HBox` bullet, and `HBox` added to both
    container lists.
  - `docs/concepts/components.md` and `ai_docs/SKILL.md`: the nested-inset
    rule now names `HBox` / `Box` alongside `Padding(0)`.
  - `ai_docs/SKILL-component.md`: the same, plus its common-mistakes entry.
  - Code samples were left as they are, since nothing was migrated.

## Checks

- `go build ./...` and `gofmt -l core/` are clean.
- `go test` passes for `./core`, `./internal/apidoc` (the docs are not stale),
  `./htmlout`, `./comps` and `./reconcile`.
- Not run on a device or in a browser: no renderer changed, and an `HBox`
  node is byte-for-byte a `Row` node with an empty base.

## Loose ends

- The installed skills in `~/.claude/skills/grmob-native-mobile-go` and
  `~/.claude/skills/grmob-component` already differed from the repo's
  `ai_docs/SKILL*.md` before this session, and were not updated.
- Another session's uncommitted N-031 move (Open → Non-goals) was in
  `ai_docs/todo/next-list.md`. Only this session's N-083 hunks are committed.

## Next

Closed: N-083. Declined: None. Raised: None.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
