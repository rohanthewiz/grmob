# Session: B1–B3, seen from the consumer

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (follows "tier-b-collection-props")

## Ask

"Adopt B1-B3 downstream in church_mobile" — item 5 of the previous session's
Next list.

**No grmob code changed.** The whole diff is in `../church/church_mobile`
(branch `roh/use-grmob`, commit `03bf45e`), whose own doc
`ai_docs/claude_sessions/2026-0905-0435-adopt-b1-b3-downstream.md` carries the
detail. What is worth recording here is what the three props looked like from
the other side of the API.

## The scorecard

    GroupedList.StickyHeaders  -> the sermons archive's month bands
    GroupedList.OnEndReached   -> the sermons pager
    ChipStrip.Scrollable       -> the sermons year filter
    core.OnEndReached          -> pagedList (articles, prayer wall, giving
                                  history) and eventsList, hand-written

Three one-line field assignments on the named screen. B2 then spread on its
own to every other paged feed in the app, because a single screen that loads
itself while the rest still need a tap reads as a bug.

## What the API got right

- **The `StickyHeaders`/`GroupHeader` pairing.** B3 cost the adopter nothing
  because the widget's band already paints `Colors.Surface`. That was not
  designed for stickiness — it predates the prop — but it is the difference
  between a pinned band and one with rows sliding visibly through it. The
  app's *hand-rolled* transparent band (events' `monthHeader`) is the control
  group: it could not take the prop without first being given a background,
  and was left unpinned with the reason written on it.
- **`OnEndReached` alongside the Footer, not instead of it.** The doc's "Keep
  the Footer" section was followed verbatim at all five sites. Handing the
  same `pager.LoadMore` to both is what the app already wanted to write, and
  the double guard (core's row-count debounce plus the app pager's in-flight
  drop) meant the adopter had to reason about the interaction exactly once.
- **`ChipStrip.Scrollable` as a field rather than a second widget.** The app
  now has two thin helpers over one widget (`chipRow`, `scrollingChipRow`),
  which is the right split *for the app* — the choice follows from what the
  chips are, not from how they look — and it is a split the app got to make
  itself because the widget did not make it first.

## Friction found

- **`core.OnEndReached`'s positional callback ID is a live trap for
  hand-written lists.** `containerNode` sorts arguments by type, so a reader
  reasonably believes the prop can go anywhere in the argument list. It
  cannot: registered *after* the rows, the handler takes a new ID every time a
  page lengthens the list, so its debounce ledger restarts each page under an
  ID a row held on the previous pass. `GroupedList` gets this right by
  construction; anyone writing `core.List(...)` by hand has to know.

  The prop's doc explains the stale-ID window but does not say **emit this
  before the children**. That sentence is nearly free and belongs next to
  "The guard's one sharp edge". Two of the four adoption sites here are
  hand-written lists, so this is not a hypothetical adopter.

- **The `Chip` selected look is now unmissable.** Surface background, Primary
  ink — the *less* prominent treatment on the *selected* chip. Latent since
  A1, reported after the bundle adoption, and `Scrollable` has now put the
  app's whole year filter on one panning line where the inversion runs the
  width of the screen. It has been the top cosmetic item for three sessions.

- **No way for a scrolling strip to say what it is.** Same
  `Role`/`AccessibilityRole` gap as the bundle inherited; a panning filter bar
  is one more thing that cannot announce itself.

## Downstream verification

`go build`, `go vet`, `gofmt`, `go test ./...`, `go test ./... -race` — clean.
Four goldens moved, and the sermons diff is the whole visual delta of the
three props: `flex-wrap:wrap` → `overflow:auto` on the filter strip,
`position:sticky; top:0; z-index:1` on both bands, `data-onendreached` on the
list.

The new downstream test dispatches the end-reached edge **twice back to back**
before anything lands — which is what an observer re-firing, a recycled row
and a per-index snapshot flow actually do — and pins that page two appears
once. That is the first test anywhere that exercises the debounce from an
app's side rather than core's.

No native shell work was needed: the Kotlin and Swift renderers live here and
already carry B1–B3.

## Next

1. Plan A3: Calendar / DatePicker.
2. `components.Chip`'s selected look — see above; third session running.
3. A `Role`/`AccessibilityRole` prop in core, serving the screen-furniture
   bundle and `DataTable`'s `<table>` semantics together.
4. Small: say "emit `OnEndReached` before the children" in its doc, for
   hand-written `core.List` callers. Found by this adoption.
5. Small: add the mid-list busy case to `EmptyState`'s doc (carried over).
