# Session: the Chip inversion, and a role prop

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (follows "calendar-adopted-downstream")

## Ask

Items 1 and 2 of the previous session's Next list, together: `components.Chip`'s
selected look (open for four sessions) and a `Role`/`AccessibilityRole` prop in
core (three consumers deep).

33 files changed, ~1000 lines added. All of it in grmob; nothing downstream.

## 1. Chip: the two state defaults, swapped

Selected is now the loud state — the theme's `Components.Button` fill and ink.
Unselected is the quiet one — Surface fill, `TextPrimary` ink, a hairline rule.
That is the whole of the change, and it is the reversal three separate
consumers reported on first sight of the rendered output.

Three things that were not obvious until the code was written:

- **Both states carry a 1px border.** Only the unselected chip wants a visible
  rule, but a rule on one state alone makes that state 2px wider and taller
  wherever `box-sizing` is content-box — and the static export ships no reset,
  so it is. A pill that grows when you tap it is a worse artifact than the one
  being fixed. The selected chip paints its ring in its own fill, where it
  cannot be seen; a theme whose Button base has no fill gets
  `ColorTransparent`, which still holds the pixel where an empty `BorderColor`
  would be dropped by the exporters and take the pixel with it.

- **The selected default restates the base's two colours rather than letting
  them show through.** The first version contributed only the ring, on the
  "Button's zero value applies nothing" principle — and a test caught what that
  costs: a colour that is never *set* cannot win against `Style`, so a strip
  handed one shared `Style{BackgroundColor(x)}` painted x on the selected chip
  and the quiet fill on every other. The inversion again, by another route.
  Restating (reading `t.Components.Button`, not re-deriving from
  `Colors.Primary`) makes "the state wins over Style" true on both sides.
  Empty base fields are skipped, since the prop setters assign unconditionally
  and `BackgroundColor("")` would *clear* the base rather than inherit it.

- **`UnselectedStyle`, and nil-vs-empty.** Both state fields read `nil` as "use
  the default" and an allocated-but-empty slice as "apply nothing" — which is
  how a caller drops a default instead of overriding it, and how the
  pre-inversion look comes back in two fields.

`examples/todoapp` keeps its pale-accent palette and is right-way-up by
consequence: its `SelectedStyle` tint now sits against neutral chips instead of
solid blue ones. Its migration test's "legacy" reference had to move with the
default, so the doc comment says that plainly rather than keeping a
byte-identity claim that no longer means what it did.

## 2. `core.AccessibilityRole`

`core/role.go`: a `Role` string type, fifteen values, `Roles()` census pinned to
its own const blocks by `role_enum_test.go` (a syntax-tree parse, the
`ContentModes`/`Alignments` pattern). `RoleNone` is declared but excluded from
the census, so the pin runs against `append(Roles(), RoleNone)` — every
declared constant accounted for, and no renderer asked to implement "unset".

The vocabulary is ARIA's own spellings, which is the load-bearing decision: the
two DOM targets then need no table at all, and the natives were going to need
one whichever vocabulary core picked.

    target              | mapping
    --------------------+--------------------------------------------------
    htmlout + runtime   | role= verbatim
    SwiftUI             | .isHeader / .isButton / .isSearchField; the other
                        | eleven named as explicit no-op arms
    Compose             | heading(), Role.Button, liveRegion Polite/Assertive;
                        | the other ten named

`mobile/verify/role_test.go` holds both native dispatches against `Roles()` in
both directions, plus the key-is-parsed and primitive-is-reached pins the tier
B props already had. Mutation-tested: dropping one arm from each native fails
by name.

### The collision worth recording

`role` now has two writers, because TabView already stamps `role="tabpanel"`.
Left alone, an update-style on a tab page would have taken the panel role off
it (self-healing at batch end via `syncTouchedTabViews`, but only by luck), and
an author's role would have been silently replaced by the wiring's.

The fix is the judgment the wiring already makes for a `<button>` page: the
author's role wins, and such a page is not wired at all. The runtime cannot
read the Style — it runs long after — so it reads the attribute back and tells
the two writers apart **by value**: `tabpanel` is not one of `core.Role`'s
spellings, so an element carrying it can only have got it from the wiring.
`TestNoRoleCollidesWithTheTabPanelWiring` in htmlout keeps that true, and it is
the one test here whose failure message explains a bug that has not happened
yet (unwire/rewire on alternate syncs).

`wireTabPanel` correspondingly clears only its own mark rather than the
attribute — total in the sense that matters, without reaching past what it
wrote.

### Adopted, since a prop with no consumer is unproven

- **DataTable** — `table > rowgroup > row > cell`, `columnheader` on the header
  cells. The rowgroup is the one that looks like padding and is not: the body
  is a `core.List`, and an unroled element between a table and its rows breaks
  the ownership that makes them its rows. It is *withheld* when the body holds
  a busy or empty placeholder instead of rows — "table, no rows" is the truth
  and an absent rowgroup already says it. `cell()` grew a role parameter rather
  than being forked in two.
- **AppBar** — banner on the bar row (not on the Box the separator adds, so the
  role lands on the same element in both shapes and is the element `Style`
  reaches), heading on the Title alone.
- **Banner** — alert when the variant is Error, status otherwise. The split the
  variant already draws visually, in the one vocabulary that has a word for
  "interrupt" and a word for "mention at the next pause".
- **SearchField** — search on the row, not the input: the landmark is the
  region a reader jumps to, and landing on the field alone puts them past the
  clear button.
- **Calendar** — button on all 42 day cells, the inert ones included. A
  disabled button is still a button, and a cell that dropped its role out of
  range would change kind as the reader paged.
- **ChipStrip** — deliberately claims nothing. A filter bar is a toolbar and
  the tags on an article are not, the widget cannot tell them apart, and it
  implements none of the roving focus a toolbar implies. Its doc says how to
  opt in.

## Decisions worth keeping

- **Nine of the fifteen roles do nothing on either native, on purpose.** That
  is why both dispatches spell out the roles they drop: on those platforms
  there is no visible difference between "deliberately inert" and "nobody has
  heard of it", and the difference is the whole of what the next reader needs.
  Same lesson `grMobScaled`'s ContentMode arms already carry.
- **Roles are never inferred.** A Box with an `OnTap` is a button only if it
  says so — guessing would have a widget wrapping a tappable row in a tappable
  card announce two nested buttons.
- **`RoleRowGroup` was added late**, when adopting in DataTable showed the
  other four describe a table with *no rows* without it. Worth noting that the
  adoption, not the design, is what found it.

## Known limits, stated rather than hidden

- A grouped table's band headings sit inside the rowgroup and are not rows,
  which ARIA has no reading for. The fix is a rowgroup per band, which needs
  the grouping engine to hand back bands rather than a flat run of children —
  a change to `appendRows` that GroupedList shares, so not one to make on the
  way past.
- The Kotlin is reviewed, not compiled: there is no Kotlin toolchain here. The
  Swift *is* compiled — `ios/verify/run.sh` type-checks the view layer for
  real, and it caught nothing, which is worth as much as if it had.

## Verification

`go build`, `go vet`, `gofmt`, `go test ./...`, `go test ./... -race`,
`wasm/verify/run.sh` (transcript replay + unit tests), `ios/verify/run.sh`
(replay + view-layer type-check), `GOOS=js GOARCH=wasm go build ./wasm` — all
clean.

New tests: role coverage for both natives (mutation-tested), the census pin,
every-role export tests on both web targets, the two tab-panel coexistence
cases on both, the five DataTable roles and the withheld rowgroup, AppBar's
pair, Banner's variant split and its override, SearchField's landmark,
Calendar's 42 cells, and four new Chip tests including the shared-Style
precedence case that found the restate-vs-show-through bug.

## Next

1. Adopt downstream in `../church/church_mobile`: the Chip inversion lands on
   its year filter for free, and the role prop wants its hand-written
   containers to say what they are. That is also the check on whether the
   fifteen values are the right fifteen.
2. The `<button>` user-agent border divergence on the two DOM renderers
   (`EmphasisGhost` draws a rule on the web and not on the natives).
3. `Calendar.Deselectable` and a counted `Marked func(time.Time) int` — both
   found by the calendar adoption, both cheap now and awkward later.
4. Small: say "emit `OnEndReached` before the children" in its doc, for
   hand-written `core.List` callers (carried over).
5. Small: add the mid-list busy case to `EmptyState`'s doc (carried over).
6. Then Tier C: heading plumbing + `Rotate` + Compass.
7. Someday, and now cheap to state: a rowgroup per group band, and
   `collectionInfo` on Compose if the counts ever become available to it.
