---
name: grmob-component
description: Create a reusable GrMob component — a view an app composes from core primitives, or a widget contributed to grmob's comps library — with theme-driven styling, caller Style overrides, accessibility, nil-safe callbacks, the hook rules for widget-owned state, debug-mode tests, and (inside the grmob repository) every census test, API-doc topic and doc page a new widget must be registered in. Use when asked to build, extract or add a component, widget or control for a GrMob app or for comps.
---

# Creating a GrMob component

> A component is a View written in Go entirely out of other Views: a value
> with `Render(ctx *core.Context) *core.Node`. It needs no registration, no
> renderer code and no platform files.

The canonical definition is the doc comment on `core.View` (`core/view.go`).
`docs/concepts/components.md` is the same material for human readers. Use
the vocabulary exactly as defined, in code comments and docs alike:

| Word | Means |
|---|---|
| **View** | the interface `Render(ctx) *Node`; each of the other three is one |
| **Primitive** | a core constructor whose node type a host draws itself (`Text`, `Row`, `Button`, `Canvas`, …); adding one touches every host |
| **Component** | a View written in Go from other Views; an app's screens are components |
| **Widget** | a component built for reuse: a struct with named fields that keeps all six rules below; every `comps` type is one |

**The contract.** Rules 1–3 are about correctness, and every component keeps
them, screens included. Rules 4–6 make a component reusable, and every
widget keeps all six. Each rule is expanded in the section noted.

1. `Render` builds a fresh tree and never mutates a Node after returning it.
2. Hooks: none, or all of them before any branch, with the component then
   rendered every pass. Own state only when it is purely presentational (§6).
3. Never register a nil callback, and call the caller's callbacks nil-safely (§5).
4. The look comes from `ctx.Theme()`, and the caller's `Style` is applied last (§3).
5. Role, name and state are on the nodes, and misuse is reported with
   `core.ReportConcern` under debug mode (§4).
6. Every field's zero value means something sensible (§2).

Anything that cannot be written under these rules is a gap in core's
primitives, not a component (§7).

This skill assumes the app-level rules in the **grmob-native-mobile-go** skill
(`ai_docs/SKILL.md`): positional hooks, immutable nodes, `State.Set` as the
only update path. It covers what changes when the code is a reusable piece
rather than a screen.

There are two destinations, and most of this file applies to both:

| You are… | Code goes in | Extra obligations |
|---|---|---|
| building an app | the app's own package (e.g. `app/ui/`) | none beyond tests |
| contributing to grmob | `comps/<snake_name>.go` in the grmob repo | [the registration checklist](#contributing-a-widget-to-comps) |

## 1. Pick the smallest shape that works

```
needs its own named knobs / slots, reused across screens? ──no──▶ a plain func
        │yes
needs a node type, style field or role core lacks? ──yes──▶ a core gap (§7), not a widget
        │no
a struct implementing core.View  (the comps idiom)
```

- **A plain function** of its data, for a screen-local piece. No hooks, so it
  is safe to create and discard inside `core.For` or `core.If`:

  ```go
  func todoRow(t Todo, setDone func(int, bool)) core.View {
      return core.Keyed(fmt.Sprintf("todo-%d", t.ID), core.Row(
          core.Checkbox(t.Done, func(v bool) { setDone(t.ID, v) }),
          core.Text(t.Title, core.FlexGrow(1)),
      ))
  }
  ```

- **A struct with named fields** for anything reusable. Named fields let the
  widget grow a knob without breaking a call site, and a `core.View` field is
  a composition slot.
- **`core.ComponentFunc(func(ctx) *core.Node {...})`** only when a piece needs
  a render function value of its own; a struct's `Render` already is one.

Before writing anything, check the existing library: `docs/api/comps-*.md`
lists every widget and its fields, and there are over a hundred of them.
Composing existing widgets is preferred. `comps.Stepper` builds its buttons
from `comps.Button{Emphasis: comps.EmphasisOutlined}` rather than drawing its own.

## 2. Anatomy of a struct widget

A complete leaf widget. It compiles as written against the current API:

```go
package ui

import (
    "strconv"

    "github.com/rohanthewiz/grmob/comps"
    "github.com/rohanthewiz/grmob/core"
)

// Tally is a labelled count: "Unread  12" with the number in a pill.
//
//	ui.Tally{Label: "Unread", Count: n, Variant: comps.VariantWarning}
//
//	┌ Row  (label, value read as one phrase) ───┐
//	│  Unread                         ( 12 )    │
//	└───────────────────────────────────────────┘
//
// # Controlled, no hooks
//
// The count belongs to the caller, so Tally holds no state and may be
// rendered conditionally (inside core.If).
//
// # Theme roles read
//
//	Label        Typography.Body, Colors.TextPrimary
//	Pill         Variant.Color fill, Variant.Ink label, Spacing.XS / SM insets
type Tally struct {
    // Label is drawn before the count and names it for screen readers.
    Label string
    // Count is the number in the pill.
    Count int
    // Variant colours the pill. The zero value is the theme's primary.
    // It reinforces the label and never replaces it: nothing announces "warning".
    Variant comps.Variant
    // Format renders the count. Nil uses strconv.Itoa.
    Format func(int) string
    // Style is applied to the row after the widget's own props.
    Style []core.StyleProp
}

func (w Tally) Render(ctx *core.Context) *core.Node {
    t := ctx.Theme()

    text := strconv.Itoa(w.Count)
    if w.Format != nil {
        text = w.Format(w.Count)
    }
    fill := w.Variant.Color(t)

    // Defaults first, the caller's Style last, so a caller override wins.
    items := make([]core.PropsAndChildren, 0, len(w.Style)+6)
    items = append(items,
        // The theme pads every Row for use as a screen band. A control sits
        // inside a padded row or field already, so it clears that inset.
        core.Padding(0),
        core.Gap(float64(t.Spacing.SM)),
        core.AlignItemsProp(core.AlignItemsCenter),
        // One stable phrase for readers: "Unread, 12".
        core.AccessibilityLabel(w.Label+", "+text),
    )
    for _, sp := range w.Style {
        items = append(items, sp)
    }
    items = append(items,
        core.Text(w.Label, core.UseStyle(t.Typography.Body),
            core.TextColor(t.Colors.TextPrimary), core.FlexGrow(1)),
        core.Text(text,
            core.UseStyle(t.Typography.Caption),
            core.BackgroundColor(fill),
            core.TextColor(w.Variant.Ink(t, fill)),
            core.PaddingHorizontal(t.Spacing.SM),
            core.PaddingVertical(t.Spacing.XS),
            core.BorderRadius(999),
        ),
    )
    return core.Row(items...).Render(ctx)
}
```

### Field conventions

| Field | Convention |
|---|---|
| `Value` + `OnChange func(T)` | controlled pair; the widget never keeps its own copy of the value |
| `OnTap func()` | a tap callback (core's prop is `core.OnClick`; there is no `OnTap` prop) |
| `Header`, `Body`, `Footer`, `Leading`, `Trailing` `core.View` | slots; when a simple field and a slot are both set (`Title` vs `Header`), **the slot wins** |
| `Style []core.StyleProp` | caller overrides; the doc comment says which node they land on |
| `Variant comps.Variant` | semantic colour role; **the zero value must be the existing look** |
| `Disabled bool` | applied with `core.Disabled(true)`, after the caller's Style |
| `Label` / `AccessibilityLabel` / `AccessibilityHint` | spoken name that is not drawn |
| `DecreaseLabel`, `RevealLabel`, … | localisable spoken defaults, filled with an `orDefault(v, "Decrease")` helper |
| `HeadingLevel int` | when the widget titles a section; zero = the widget's own tier |
| `Initially*` | seeds widget-owned state on the first pass only |
| `Format func(T) string` | display formatting; nil = a sensible default |
| `FocusRef *core.FocusRef` | when callers need to move focus to it |

**Zero values must mean something sensible**, because Go has no "unset". If a
zero would be a trap, rename the field so that zero is safe. `Stepper`
enforces `Min`/`Max` only when `Max > Min`. `Poll` uses `Mine bool` rather
than `Voted int`, and `Wizard` uses `Blocked` rather than `CanAdvance`.

**The doc comment is part of the widget.** Include a one-line usage snippet, an
ASCII diagram of the node tree, a statement about hooks (either "No hooks" or
the obligation), a `# Accessibility` section where anything is non-obvious, and
a `# Theme roles read` table.

## 3. Styling: theme in, caller last

- **Every look comes from `t := ctx.Theme()`**, never a hex value or a magic
  number:
  - colours: `t.Colors.{Primary, Secondary, Background, Surface, TextPrimary, TextSecondary, Error, Border, ControlBorder}`, and `t.Colors.SuccessColor()`, `WarningColor()`, `ChartColors()`, `SequentialColors()`
  - spacing: `t.Spacing.{XS, SM, MD, LG, XL}` are ints. Padding props take ints (`core.PaddingHorizontal(t.Spacing.SM)`), while `core.Gap` and `core.BorderRadius` take float64 (`core.Gap(float64(t.Spacing.SM))`)
  - text styles: `core.UseStyle(t.Typography.{Title, Subtitle, Body, Caption})`
  - `Variant.Color(t)` for a fill, `Variant.Ink(t, bg)` for the label on it, `Variant.AsInk(t)` when the colour is the ink itself (outlined or ghost)
- **Merge order: the widget's defaults go first, then the caller's `Style`.** A
  caller's `PaddingLeft(0)` must beat the widget's `PaddingHorizontal`; a wide
  prop first and side props after is how inset props settle. The exception is
  behaviour that must not be overridable. Button appends `core.Disabled(true)`
  after the caller's Style, so a styling tweak cannot re-enable it.
- **`core.Row` and `core.Column` carry the theme's screen inset.** A widget
  that is a control inside something else adds `core.Padding(0)`, or uses
  `core.HBox` (a Row with no theme base) in place of the Row. Use
  `core.Box` for a wrapper that must be geometrically invisible, since Box has
  no theme base. Box does not centre its child on the natives; centre with a
  Row plus `Justify` / `AlignItemsProp`.
- **`core.UseStyle` adds and never clears**: a zero field is treated as unset.
  To force zero, use the individual setter (`core.FontSize(0)`).
- On the web, flex shares include padding: a "fill the rest" child wants
  `core.FlexGrow(1)` with `core.FlexBasis("0")`. To pin something to the end
  of a row, give a sibling `FlexGrow(1)` instead of using `JustifyBetween`.
- Physical padding does not mirror under RTL on the web; indent with a leading
  spacer.

## 4. Accessibility

- **Don't restate what the node type owns.** `core.Button` is already a
  button, and `core.Disabled` already announces "dimmed/unavailable", so a
  `", disabled"` suffix gets heard twice.
- **A tappable Box or Row** needs `core.OnClick(fn)` plus
  `core.AccessibilityRole(core.RoleButton)` plus a label. Without the role,
  every renderer draws it as inert scenery.
- **Glyph buttons need spoken names**: "−" gets "Decrease", "×" gets "Close".
- **A toggle keeps one stable name** and states its state separately with
  `core.AccessibilitySelected(core.SelectedWhen(on))`. Don't encode the state
  in the name ("Show password" / "Hide password").
- **A grouped control** (stepper-like) is `RoleGroup` plus a label plus
  `core.AccessibilityValue(core.ValueOf(now, min, max).WithText(text))`. Give
  numbers only when the range is real.
- **An expandable control states both halves**:
  `core.AccessibilityExpanded(core.ExpandedWhen(open))` and an `OnClick` on
  the same node. Either half alone is `ConcernInertDisclosure`.
- **Headings go on the words** (the Text node), except inside a button.
  A button's children are presentational, so the heading goes on a `core.Box`
  wrapper that has its own `AccessibilityLabel` (`comps.Accordion`'s shape).
  The comps outline is AppBar 1, GroupHeader and Card.Title 2, Accordion 3.
- **Chart-like canvases**: the root is `RoleImg` with a summary label, and
  tick and legend Text is `core.AccessibilityHidden()`.
- **Container roles that claim their children** (`RoleListBox`,
  `RoleRadioGroup`, `RoleGrid`, `RoleTabList`, `RoleToolbar`) are for *closed*
  widgets only, ones with no `core.View` slot a caller could fill with
  something else.
- **Misuse is reported in debug mode**, not by panicking. Export a
  `const ConcernMyWidgetInert = "my-widget-inert"` and report it with
  `if core.IsDebugMode() && cond { core.ReportConcern(ConcernMyWidgetInert, "what is wrong and what it costs") }`.

## 5. Callbacks

- **Never register a nil func.** A nil handler in the registry panics when a
  native tap arrives. Substitute a no-op, and report a concern if a missing
  callback makes the widget inert:

  ```go
  onTap := b.OnTap
  if b.Disabled || onTap == nil {
      onTap = func() {}
  }
  ```

- **Call the user's callback nil-safely**, and only on a real change:
  `if next != s.Value && s.OnChange != nil { s.OnChange(next) }`.
- **Clamp, dedupe and validate inside the widget**, so the caller's handler can
  be a plain setter (`OnChange: qty.Set`).
- **A disabled control keeps its handler registered** and adds
  `core.Disabled(true)`. The handler is still guarded, because a tap can race
  the disabling patch.
- **Key list rows** with `core.Keyed(stableID, row)`. This scopes their
  callback IDs too. Keep separator and header keys in a different namespace
  from row keys (`"sep:"+k`, `"row:"+k`).

## 6. Hooks inside a widget

A widget's `Render` gets the **caller's** context, so every hook it calls
takes a positional slot in the caller's sequence.

- **The bar: own state only when it is purely about the widget's own
  presentation** and no application would want to read, drive or persist it.
  An accordion's open/closed or a password field's reveal qualifies. A
  calendar's visible month does not (a screen may need to open on the month
  of its next event), so `Calendar` takes no hooks and `DatePicker` packages
  that state for the form case.
- **Take every hook first, before any branch**, so the slot count never varies
  between passes.
- **A widget with hooks must itself be rendered unconditionally** — never
  inside `core.If`, a conditional, or a varying `For`. Say so in its doc.
  Hook-free widgets say "No hooks; may be rendered conditionally."
- **Content rendered only while open must be hook-free.** Alternatively, the
  caller wraps it in its own scope (`ctx.Scope(key)`).
- **Timers**: use `hooks.UseIntervalWhile` / `hooks.UseTimeoutWhile` so a
  stopped widget stops ticking. A plain `UseInterval` re-renders forever.

A hook-owning widget, built on the expandable-control rules from §4:

```go
// Spoiler hides Content behind a tappable title until the user reveals it.
//
// It takes one hook (revealed), so it must be rendered unconditionally,
// every pass; Content is rendered only while revealed and must be hook-free.
type Spoiler struct {
    Title   string
    Content core.View
    // InitiallyRevealed seeds the state on the first pass only.
    InitiallyRevealed bool
    // Style is applied to the outer column.
    Style []core.StyleProp
}

func (s Spoiler) Render(ctx *core.Context) *core.Node {
    // Before any branch: the slot is claimed on every pass.
    revealed := core.NewState(ctx, s.InitiallyRevealed)
    t := ctx.Theme()

    chevron := "▸"
    if revealed.Get() {
        chevron = "▾"
    }
    header := core.Row(
        core.Padding(0),
        core.Gap(float64(t.Spacing.SM)),
        // Role, state and handler on the same node: a stated expansion with
        // no click handler is announced on the web and inert on Android.
        core.OnClick(func() { revealed.Set(!revealed.Get()) }),
        core.AccessibilityRole(core.RoleButton),
        core.AccessibilityExpanded(core.ExpandedWhen(revealed.Get())),
        core.AccessibilityLabel(s.Title),
        core.Text(chevron, core.UseStyle(t.Typography.Body)),
        core.Text(s.Title, core.UseStyle(t.Typography.Body), core.FontWeight(core.Bold)),
    )

    items := []core.PropsAndChildren{core.Padding(0), core.Gap(float64(t.Spacing.XS))}
    for _, sp := range s.Style {
        items = append(items, sp)
    }
    items = append(items, header)
    if s.Content != nil {
        items = append(items, core.If(revealed.Get(), s.Content))
    }
    return core.Column(items...).Render(ctx)
}
```

## 7. When it is a core gap, not a widget

A widget never needs Kotlin, Swift, JavaScript or htmlout changes. If it
does, what is missing is a core primitive, and that is a separate,
larger piece of work:

| Missing | Touches |
|---|---|
| a node type | `core`, `htmlout/tag.go` (`tags`, `TransparentTypes`), the wasm runtime's tag table, `android/.../runtime/Renderer.kt`, `ios/GrMob/Runtime/Renderer.swift`. Guarded by `TestRuntimeTagsMatchGo` and `TestBothNativeRenderersDispatchEveryNodeType`; a missed type renders as an empty Column. |
| a `core.Style` field | `UseStyle`'s merge (`TestUseStyleMergesEveryField`), htmlout, the wasm runtime, `GrMobStyle.kt` / `Renderer.kt`, `GrMobStyle.swift`, and the verify tests |
| a role | core, both web exporters, both natives |

Check whether a primitive already does the job before reaching for one.
`core.Canvas` draws anything a Text node cannot; `comps.Rating` draws its
half-star on a canvas because SwiftUI truncates squeezed text. In an app,
report a missing primitive upstream rather than working around it in
platform code.

Canvas widgets have their own rules:
- Draw on `core.Canvas(w, h, shapes, core.CanvasStretch, …)`.
- Keep the shape count constant, using empty `core.Shape{}` placeholders, so
  toggling a feature patches one slot instead of shifting every later one.
- Use round-capped zero-length strokes for dots, because circles distort
  under stretch.
- Decide on mirroring. Pass `core.CanvasMirrorsRTL` when x is time or
  category order. Omit it for real-world geometry such as a QR code or a map.

## 8. Testing

Render under debug mode **and** run the whole-tree audit. A bare `Render`
skips `core.AuditTree`, and that audit is where unusable value ranges,
duplicate IDs and inert disclosures show up:

```go
package ui

import (
    "testing"

    "github.com/rohanthewiz/grmob/core"
)

// renderDebug renders v the way render.Manager does and fails on any concern.
func renderDebug(t *testing.T, v core.View) (*core.Context, *core.Node) {
    t.Helper()
    core.SetDebugMode(true)
    core.ClearConcerns()
    t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
    ctx := core.NewContext()
    ctx.BeginRenderPass()
    n := v.Render(ctx)
    ctx.EndRenderPass()
    core.AuditTree(n)
    if dump := core.DumpConcerns(); dump != "" {
        t.Errorf("concerns raised:\n%s", dump)
    }
    return ctx, n
}

// renderPass drives a later pass on the same context, for widgets with hooks.
func renderPass(ctx *core.Context, v core.View) *core.Node {
    ctx.BeginRenderPass()
    ctx.Reset() // the test is the render driver here, so this is its call
    return v.Render(ctx)
}

// findFirst locates a node by predicate; never assert on child indexes alone.
func findFirst(n *core.Node, pred func(*core.Node) bool) *core.Node {
    if n == nil || pred(n) {
        return n
    }
    for _, c := range n.Children {
        if f := findFirst(c, pred); f != nil {
            return f
        }
    }
    return nil
}

func findText(n *core.Node, s string) *core.Node {
    return findFirst(n, func(x *core.Node) bool { return x.Type == "Text" && x.Props["content"] == s })
}

func TestSpoilerRevealsOnTap(t *testing.T) {
    sp := Spoiler{Title: "Ending", Content: core.Text("they were the butler")}
    ctx, n := renderDebug(t, sp)
    if findText(n, "they were the butler") != nil {
        t.Fatal("content starts hidden")
    }
    header := findFirst(n, func(x *core.Node) bool { return x.Style.AccessibilityRole == core.RoleButton })
    ctx.TriggerCallback(header.Props["onClick"].(string)) // a tap, through the real registry

    n = renderPass(ctx, sp)
    if findText(n, "they were the butler") == nil {
        t.Fatal("a tap reveals the content")
    }
}
```

What to cover:
- **Structure and accessibility**: node type, role, label, value, and the
  spoken names of glyph buttons.
- **Behaviour**: fire callbacks with `ctx.TriggerCallback` /
  `TriggerTextCallback` / `TriggerBoolCallback` / `TriggerIntCallback`. Assert
  what `OnChange` received, including that it was *not* called on a no-op.
- **Zero-value configuration** renders without concerns.
- **Caller `Style` outranks the widget's defaults.**
- **Every bundled theme**: loop over `core.BundledThemes()` with
  `core.NewContext().WithTheme(th)` when the widget computes colours.
- **Snapshots, when extracting from app code**: pin
  `htmlout.ExportHTML(node)` before and after the extraction.

Inside the grmob repo, `comps/helpers_test.go` already has `findFirst` and
`findText`, `comps/stepper_test.go` has `renderDebug`, and
`comps/accordion_test.go` has `renderPass`. Reuse them rather than redefining
them.

## Contributing a widget to comps

Within the grmob repository, a new widget file is wired into several places,
and most of them fail the build if missed. Work through these in order:

1. **`comps/<snake_name>.go`**, following §2–§6. Use the package's helpers
   rather than re-deriving them:
   - `asProps`, `orDefault`
   - `headingProps(asked, own)` with `headingLevelSection` / `headingLevelSubsection`
   - `disclosure{...}.view()`, for any expandable header
   - `appendRows` / `rowsSpec`, for keyed row lists (shared by GroupedList and DataTable)
   - `bandInsets`
   - `chart.go`: `niceScale`, `cartesianFrame`, `legend`, `chartPalette(t)`
   - `px`
2. **`comps/<snake_name>_test.go`**, using the shared harness (§8).
3. **Census tests.** These are hand-maintained lists, so nothing will discover
   an unlisted widget for you:
   - **It draws a `core.Canvas`**: add it to `mirrored` or `fixed` in
     `comps/canvas_mirror_test.go` (`TestCanvasMirrorCensus`).
   - **It is a chart that speaks a summary**: add a case to
     `TestEveryChartCarriesItsData` in `comps/chart_data_test.go`. The root
     is a labelled `RoleImg` carrying `Props["chartData"]`.
   - **It references a composite container role**: add it to
     `closedComposites` in `comps/nested_composite_test.go`, with a reason.
     This list *is* enforced by parsing; a listed file must have no
     `core.View` fields, except those in `viewsOutsideTheComposite`.
   - **It sets its own non-zero padding or margin on the node `Style`
     reaches**: add a case to `TestACallerStylePropOutranksAWidgetsOwnInsets`
     in `comps/caller_style_insets_test.go`.
   - **It adds a knob to `rowsSpec`**: update the `comps/rows_spec_test.go`
     censuses.
4. **API reference topic.** Add the file name to the right topic's `Files` in
   `internal/apidoc/packages.go` (the `comps` entry). The topics are
   `structure`, `lists`, `inputs`, `actions`, `overlays`, `display` and
   `charts`. Also add the widget's name to that topic's `Blurb`. Then
   regenerate:

   ```sh
   go run ./internal/apidoc/gen      # -> docs/api/comps-<topic>.md
   ```

   `TestTopicsPartitionTheirPackage` fails on a file no topic lists.
   `TestGeneratedPagesAreUpToDate` fails when `docs/api/` is stale, so
   regenerate after every exported-signature or doc-comment change.
5. **Tracked Go file count.** Five sentences in `wasm/verify/repowalks_test.go`
   and `wasm/verify/timings_test.go` quote the repository's Go file count, and
   the check in `wasm/verify/prosefigures_test.go` fails when a new file moves
   it. Run `go test ./wasm/verify/`. The failure message names the new count
   and each sentence that quotes the old one; edit those numbers. BSD sed has
   no `\b`, so use `perl -pi -e 's/\bOLD\b/NEW/g' <files>`.
6. **Prose docs.** Add a `## WidgetName` section to `docs/components.md`, near
   its topical neighbours. It has a paragraph saying what the widget is, a
   fenced `go` example, bold-lead paragraphs for its rules (controlled,
   bounds, accessibility), and an "Other notes:" list.
7. **Widget lists that no test enforces.** Add the name to the topic table in
   `ai_docs/SKILL.md` (and the installed copy of that skill). The widget
   lists in `docs/index.md` and `README.md` are optional.
8. **Optional gallery lesson.** Append a lesson to the end of
   `examples/tutorial/chapter4.go`, never in the middle: lesson deep-link IDs
   (`grmob://lesson/4.N`) come from position. Then follow the lesson-count
   tests as they fail:
   - `examples/tutorial/readme_counts_test.go`
   - `examples/tutorial/pagecount_test.go`, which checks the `wasm/index.html` header
   - the "N lessons across M chapters" sentences in `README.md` and `docs/tutorial-interactive.md`
   - `internal/shotclaims`
   - the contents screenshot, via `wasm/shots/shoot.sh tutorial-contents`
9. **Verify**:

   ```sh
   gofmt -l .            # must print nothing: CI and .githooks/pre-push stop here first
   go vet ./comps/
   go test ./...
   ```

   Never `go build ./examples/<x>` without `-o /dev/null`, because it
   overwrites a tracked binary. For a visual check, run `./build.sh && go run
   ./serve`, or use a throwaway `wasm/shots/scripts/zz-*.js` with
   `GRMOB_SHOTS_OUT=<scratch> wasm/shots/shoot.sh zz-<name>`, and delete the
   script afterwards.
10. **Commit**, one widget per commit, with `docs/api/` regenerated in the
    same commit. If the work came from `ai_docs/todo/next-list.md`, update
    that item, and write a session doc (`/sess-save`).

## Pitfalls, in the order they bite

1. **A hook after a branch**, or a hook-owning widget placed inside `core.If`.
2. **Hard-coded colours or sizes** instead of theme roles. They break under
   `DarkTheme` and every custom theme.
3. **Caller `Style` applied before the defaults**, so the override loses.
4. **A nested Row or Column without `core.Padding(0)`** (or built as
   `core.HBox` / `core.Box`), which shows as a double inset.
5. **A nil callback registered**, which panics on the first native tap.
6. **A zero value that means something surprising.** Rename the field so zero
   is safe.
7. **State the app would want to own kept inside the widget.** Make it
   controlled instead.
8. **A test that calls `Render` bare**, so the `AuditTree` findings are never
   seen.
9. **A new file missing from `internal/apidoc/packages.go`, or a stale
   `docs/api/`**, either of which turns the build red.
10. **A canvas or chart left out of its census.** Nothing fails; the guard
    simply never checks it.
