# Session: GroupedList's first consumer (A1 adopted downstream)

Session: https://claude.ai/code/session_01J7qf4UetyH8cQ3D7Egn9dC
Date: 2026-09-04 (follows "grouped-list-and-data-table")

## Ask

"Apply the Grouped datatable to church mobile sermons." — item 1 of the
previous session's Next list.

**No grmob code changed this session.** The work landed in
`../church/church_mobile` (branch `roh/use-grmob`); its own session doc is
`ai_docs/claude_sessions/2026-0904-1912-grouped-list-sermons.md` over there.
This doc records what the first real consumer proved about the widgets, so the
grmob thread stays unbroken.

## What the adoption used

`components.GroupedList[api.Sermon]` with `Key`, `Row`, `GroupBy`, a `Header`
override, `Empty`, and `components.LoadMore` as the `Footer`, fed straight
from the app's `usePager`. `DataTable` was not the right widget for the screen
that commissioned it — worth knowing:

- A phone-width sermon row is a card (glyph, title, subtitle), not a tuple.
- DataTable sorts the slice it is handed. Under an offset pager that slice is
  "the pages loaded so far", so a sort control would reorder a prefix of the
  archive and present it as the archive. **DataTable's Sort belongs with
  caller-owned or fully-loaded data; a server-paged feed sorts in the query.**
  Worth a line in DataTable's doc comment.

## What held up

- **Run-length grouping did exactly what it was designed for.** June straddled
  the page boundary; "Load more" extended the open June run rather than
  emitting a second June header, and nothing above the fold moved. This is the
  behavior the groupRuns comment claims, now exercised against a real offset
  pager rather than a table-driven test.
- **`Header` override + `GroupHeader{HideCount:}`** turned out to be load
  bearing, and for a reason the widget did not anticipate: a run count counts
  *loaded* rows, so under an append pager the trailing group's count is a
  number about to be wrong. The screen suppresses it while `HasMore`.
  **This is general to every append-paged GroupedList** — a candidate for the
  widget itself (an `OpenGroup string` or `HideTrailingCount bool` field), or
  at minimum a paragraph in the GroupedList doc comment.
- **Footer-renders-even-when-empty** is right for page two onward and wrong
  for page one: a first-page failure wants the screen, not a hairline strip
  under nothing. The consumer resolved `!Loaded` before reaching the widget.
  That split is fine, but it means "GroupedList replaces your list body" is
  only true after the first successful page — worth saying out loud in the
  docs.
- **`LoadMore.ErrorText` earned its existence immediately**; the default
  `err.Error()` was a URL and a status code.
- **The keyed/unkeyed row split** the widget forces (Key is the widget's job,
  so Row must hand back an unkeyed view) is a small but real migration cost
  for any app whose row helpers already return `core.Keyed`. Cheap to do,
  worth a sentence in the GroupedList comment so the next adopter expects it.

## Incidental

Ten of church_mobile's eleven snapshot goldens were already stale against
current grmob before this change: Column nodes now emit explicit
`display:flex; flex-direction:column` (the block-flow work). Purely additive,
no content change — but a reminder that grmob's htmlout output is a downstream
golden-file contract, and a change to it silently reddens every consumer's
snapshot suite.

Also noticed downstream: `components.Chip`'s selected look (Surface bg,
Primary ink) reads as the *less* prominent of the two states, so a filter row
looks inverted — the unselected chips are solid primary. It is what the doc
comment says it is, so it is a design call rather than a bug, but it surprised
a reader of the rendered output.

## Next

1. Fold the two findings above back into the widgets: a trailing-count story
   for append-paged GroupedList, and DataTable's sort-vs-server-paging caveat.
2. Plan A2: the lifted widget bundle (AppBar, Banner, EmptyState, SearchField,
   ChipStrip, Skeleton, StatTile).
3. B1-B3 in one renderer pass (horizontal scroll, `OnEndReached`, sticky
   headers). Sticky headers now have a consumer waiting.
