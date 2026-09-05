# Session: The two findings from the first consumer, folded back in

Session: https://claude.ai/code/session_01J7qf4UetyH8cQ3D7Egn9dC
Date: 2026-09-04 (follows "grouped-list-sermons", same session, later)

## Ask

"Fold the trailing-count fix into GroupedList", then "do the datatable sort
caveat too" — items 1 and 2 of the Next list in
`2026-0904-1912-grouped-list-sermons.md`, both found by adopting A1 downstream
in church_mobile's sermons screen. That doc's Next list is now spent except
for A2 and B1–B3.

Commits: `661008b` (trailing count), `ce8d61b` (partial sort). church_mobile
`ddcf5c4` drops the workaround the first one replaced.

## 1. `HideTrailingCount` (661008b)

A group's `Count` counts the rows the widget was handed. Under an append-style
pager that is "the rows loaded so far", and only the *last* group is affected:
every earlier one was closed by the next group's first row, so its count is
final. A band reading "June 2026 (1)" above a run about to become four is not
a stale number, it is a wrong one, and it changes under the reader on a tap
they did not think was a question about June.

    HideTrailingCount: pager.HasMore

### Design decisions

- **A bool, not an `OpenGroup string`.** The widget already knows which run is
  last; the caller only knows whether more rows are coming. Asking for a key
  makes every caller recompute what `groupRuns` just determined — which was
  exactly the paragraph the sermons screen had to write.
- **A `Header` override is left completely alone**, `Count` included.
  Considered and rejected: zeroing `Count` for the trailing run as a sentinel
  (a run is never empty, so 0 is unreachable naturally). An override doing
  `itoa(g.Count)` would then silently print "0", which is worse than a
  provisional number. There is a test pinning the override path.
- **DataTable got the same field.** It shares `appendRows`, so the plumbing
  was one argument, and leaving the twin with the identical latent bug was
  worse than the small scope stretch. Its own test, because it reaches
  `appendRows` down a different path (own cell renderer, own row wrapper)
  where a dropped argument would go unnoticed.
- The header keeps its key across the change, so the reconciler patches the
  badge in rather than replacing the band.
- Renamed the run index to `ri`: the row loop below it also uses `i`.

## 2. `ConcernPartialSort` (ce8d61b)

A column with `Less` is sorted by the table over all of `Rows` and only
`Rows`. Right when that is the whole set — caller-owned, a completed fetch, or
client paging where `Pagination` slices rows the table already holds. Wrong
when `Rows` is a window somebody else chose: sorting the pages an offset pager
accumulated yields the alphabetically-first of *what happens to be loaded*,
under a header claiming the alphabetically-first of the table. Nothing in the
output distinguishes the two. Such a table wants `Sortable` without `Less`, so
`OnSort` goes into the query.

### Design decisions

- **Went past the doc comment the plan called for.** The repo already has the
  machinery for this class of silent bug, and this is one. Flagged to the user
  as a scope call.
- **The trigger is exact, not heuristic:** an active `Sort` on a `Less` column
  *while* `Pagination.PageCount > 0`. A declared PageCount is the caller
  stating the server does the paging, so the combination is unambiguous.
- **The kind constant lives in `components`, not `core`.** It is a
  widget-level contract and core has no business knowing what a DataTable is;
  `core.ReportConcern` is exported for exactly this. First call to the debug
  facility from outside core.
- **The append case is undetectable and was left that way.** A slice of
  accumulated pages is indistinguishable from a complete one. Guessing from
  the presence of a `LoadMore` footer would fire on correct tables, so that
  half stays documentation.
- **Three negative test cases against one positive.** A concern that cries
  wolf is worse than no concern, because the list it lands in is read as
  "these are bugs".

## Method note

Every behavioral claim in both changes was mutation-checked: flip the trailing
run selection to `i == 0`, drop the `HideTrailingCount` argument at
DataTable's call site, remove the `Less == nil` guard — each makes exactly the
test that should fail, fail. Worth keeping up; the assertions here are all of
the form "a badge is absent", which passes for free if the test is looking at
the wrong node.

`findText` in the components tests is an *exact* content match, not a
substring one, which is what makes `findText(header, "2")` a real assertion
even though the label is "Month 2025-11".

## Also touched

- Tutorial lesson 4.6 demonstrates both: its grouped demo was itself the
  append-pager case and had the wrong count; the sort paragraph now gives the
  rule rather than just the mechanism. Two new key points.
- `docs/concepts/debug-mode.md` gets a `partial-sort` row. First draft linked
  to `components/data_table.go`; no other doc links out to source, so that was
  dropped to plain code formatting.
- ROADMAP's existing A1 bullet already covers this at the right altitude — no
  new line.

## Verification

`go test ./...` green in both repos; `GOOS=js GOARCH=wasm go build ./wasm`
builds. church_mobile's goldens did **not** move when the sermons screen
switched from its hand-rolled override to `HideTrailingCount`, which is the
useful confirmation that the widget reproduces what the screen was doing.

## Next

1. Plan A2: the lifted widget bundle (AppBar, Banner, EmptyState, SearchField,
   ChipStrip, Skeleton, StatTile).
2. B1–B3 in one renderer pass (horizontal scroll, `OnEndReached`, sticky
   headers). Sticky headers now has a waiting consumer: the sermons month
   bands, and the events tab's hand-rolled ones.
3. Latent, noticed downstream: `components.Chip`'s selected look (Surface bg,
   Primary ink) reads as the *less* prominent state, so a filter row looks
   inverted. It is what the doc comment says it is, so it is a design call —
   but it surprised a reader of the rendered output.
