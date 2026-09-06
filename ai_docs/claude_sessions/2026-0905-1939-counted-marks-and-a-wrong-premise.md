# Session: dots that count, a deselect, and a carried item whose premise was wrong

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (items 1, 2 and 3 from "button-border-and-nesting-level")

## Ask

"Let's do items 1, 2, and 3 in the Next list." That is `Calendar.Deselectable`
plus a counted `Marked`, the `OnEndReached` ordering sentence, and the mid-list
busy case in `EmptyState`'s doc. Two of the three were the small ones. The
small ones were not the same size.

## Part 1 — the Calendar

### `Marked func(time.Time) bool` → `func(time.Time) int`

Two services on one Sunday and one service on one Sunday are different facts
about the day, and the bool could state neither. The signature was flagged as
"cheap now and awkward later" three sessions running; this is the now.

A count cannot be one box's colour, so the single dot became a row of them —
which is where the interesting part is. The dot that was there before was
*always in the tree and sometimes transparent*, for two stated reasons: the day
numbers keep one baseline whether or not their day is marked, and toggling a
mark is a colour patch rather than a child insertion in the middle of a 42-cell
grid. Both had to survive.

Three shapes were considered and two rejected:

    fixed 3 slots, ink left to right     never structural, but a lone dot sits
                                         ~7px left of the number it belongs to
    fixed 3 slots, ink middle outward    n=1 and n=3 centre, n=2 reads as a
                                         3-cluster with its middle missing
    ≥1 slot, add the 2nd and 3rd         chosen

The last one keeps the old invariant *exactly* for the case the old code
handled: nothing↔one is still a colour patch on a single always-present box, so
every cell of a yes/no calendar and most cells of any other never change shape.
Only the second and third dots are structural, and only on the cells whose
count reaches them — the price of counting, charged to the cells doing it.

The row is `core.Row(Padding(0), Gap(3), AccessibilityHidden(), dots...)`. No
`Justify`: the cell's own `AlignItemsCenter` centres it on every target, which
is what centred the single dot before. Verified in exported HTML —
`flex-direction:row; gap:3px`, `aria-hidden="true"`, and 3/2/1/1-transparent
dots on March 1/15/22/8.

`calendarMaxDots = 3`, and the cap is not arbitrary: 3 dots and 2 gaps is 21px
against roughly 45px of cell on a narrow phone, and past three the answer a
reader takes away is "several" rather than a number — which is exactly what a
capped cluster says. `calendarDotGap = 3` is deliberately **not** a theme step:
the cluster is glyph-scale furniture inside a cell, not part of the screen's
layout rhythm, and a theme that loosened its spacing would otherwise push three
dots wider than the cell holding them. (XS is 4px — already most of a dot.)

The count is clamped, not trusted. A caller's `len(x) - 1` slip asking for a
negative number of children is the failure that clamping is cheaper than.

### `Deselectable bool`

`Selected` already spells "nothing is chosen" as the zero time, so a second tap
on the chosen day reports that same zero back through `OnSelect`. The value
makes a round trip through the caller's own state and there is no `OnDeselect`
— two callbacks setting one piece of state is two things for every consumer to
keep in step, and a screen that wants to tell them apart can test the zero it
was handed.

Off by default, and the default is the half worth writing down. A picker asking
which day the appointment is has no "no day" to offer; there a stray second tap
that quietly emptied the field loses an answer nobody asked to lose. So the
widget does not guess which of the two it is in — a filter opts in.

**`DatePicker` forces it off**, and this is the sharpest bit of the part. A
form hands `OnSelect` to whatever holds its date. A deselecting tap would feed
that setter the zero time — a clear arriving through the callback whose entire
job is to fill the field, indistinguishable from a pick. Emptying is `OnClear`'s,
and whether a field may be emptied *at all* is the form's question, which is
why `OnClear` is nil-able. It is the one template field the picker overrides
for a reason other than "the picker computes it", so it is pinned by its own
test rather than folded into the pass-through one.

### The one thing left open, deliberately

A deselectable selected cell still announces `", selected"` and nothing about
what activating it now does. That is *state* — ARIA would spell it
`aria-pressed` — and `core.Style` has no slot for one. Same missing slot as the
carried "no `tab`/`tablist` role" item, so it is recorded against that rather
than worked around with a one-off English hint on one of 42 cells. Pinned from
the other side too: `TestCalendarDeselectableDoesNotChangeHowTheCellReads`
holds the fill and the spoken name identical with and without the flag, because
the tempting shortcut — drawing a deselectable selection differently — would
make the grid say two things about one state.

## Part 2 — the `OnEndReached` item, which was wrong

The carried sentence was: *emit `OnEndReached` before the children*, because a
prop registered after the rows takes a new callback ID every time a page
lengthens the list, restarting the debounce ledger each page.

It was tested before it was written down. **Argument order makes no
difference.** `containerNode` registers behavior props during its argument loop
and renders children only *after* that loop, so the List's own ID is assigned
before any row can take one:

    rows=2  propFirst=cb_0  propLast=cb_0
    rows=5  propFirst=cb_0  propLast=cb_0
    rows=9  propFirst=cb_0  propLast=cb_0

That contract is written on `containerNode` itself and lands in `d4f90dc`
(Aug 31) — five days *before* the friction report that assumed otherwise. The
adopter downstream diagnosed a real symptom and named the wrong cause.

The real cause is one step out. Anything earlier in the same pass that
registers a *varying* number of callbacks moves the ID:

    eager rows=2 -> cb_2
    eager rows=5 -> cb_5
    eager rows=9 -> cb_9

— a row helper that calls `view.Render(ctx)` itself instead of returning a
`View`, or a sibling above the list whose own children grow with the page. Then
each page starts its guard afresh under a key something else held last pass and
the double-load comes back. Same stale-ID family as the edge the doc already
described, and the same identity-keyed IDs close both.

So the doc says the opposite of what the item asked for, and says it under a
heading a reader will actually look under ("Where in the argument list it
goes" — "Anywhere."). Worth stating rather than leaving to be derived: the
guard is keyed by an ID, IDs are positional, and a reader who works that out is
*right* to wonder. The answer just happens to be reassuring.

`TestOnEndReachedIDIsIndependentOfArgumentOrder` pins it at two row counts and
in both spellings, because the failure it guards is not "the two spellings
differ" but "one of them drifts as the list grows" — and neither fails loudly.

## Part 3 — `EmptyState`'s mid-list busy case

Recorded as a rule rather than a second example, since the rule is what the
next screen needs:

> errors and empties speak in the primary line; a wait speaks there only when
> it is the whole screen

Screen-wide wait → `Title`. List tail → `Hint`, `Title` left empty, padding
usually brought down to `Spacing.SM` (Style is applied after the defaults). The
forcing case is the one the downstream adoption found: under the last row of a
paged list, body-sized primary ink reads as one more row, and the reader tries
to parse "Loading sermons…" as content.

The widget cannot apply the rule itself — it is handed a slot and never learns
whether that slot is a screen's middle or a list's end — so this is the
caller's call, which is the whole reason it belongs in a doc.

## Verification

All six paths, green:

    go build / go vet / gofmt / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 6 mjs suites
    ios/verify/run.sh           data layer + view-layer typecheck
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

No native work, and that is the whole story this time: the change is Go-side
widget composition over props all four renderers already carry. Nothing in
`mobile/verify` had a claim to make.

New and changed tests:

    core/list_props_test.go            the ordering contract, two row counts,
                                       both spellings
    components/calendar_test.go        the cluster (renamed from the dot),
                                       counts across the cap including a
                                       negative, the nil-Marked baseline, the
                                       hidden row and its shed inset, both
                                       halves of Deselectable, and the cell
                                       reading identically either way
    components/date_picker_test.go     the forced override, driven through the
                                       real callback
    examples/tutorial/chapter4_test.go three dots on March 1 down to a
                                       transparent placeholder on March 8, and
                                       a second tap clearing the caption

The tutorial needed a fixture of its own: the 4.6 archive has at most one entry
on any day, so a count taken over it alone could never draw more than one dot —
and a `Marked` that cannot show two is the exact signature lesson 4.9 now
exists to explain. `tutorialAlsoOn` adds two things to March 1 and one to March
15 without touching `archive`, which four other lessons read.

## Docs

`components/calendar.go` gained a "The zero time goes both ways" section and
rewrote `Marked`'s field doc; the ASCII grid legend now reads `·n = one mark,
··n = two`. `core/list.go` gained "Where in the argument list it goes".
`components/empty_state.go` gained "The busy line moves when the wait is not
the whole screen". `docs/components.md` (EmptyState) and
`docs/concepts/views.md` (the ordering note) carry the same in prose. ROADMAP
has two new checked lines.

## Next

Carried, minus the three done here:

1. Then Tier C: heading plumbing + `Rotate` + Compass.
2. An "on-light" tone per palette role, which both Button's outlined treatment
   and `Chip.ProminenceLoud` are working around.
3. **No `tab` / `tablist` role.** Needs a *state* (`aria-selected`), which has
   no home on `Style` and which both natives spell differently. Now two
   consumers deep: `Calendar.Deselectable` wants the same slot to say that
   activating a selected cell clears it.
4. **ARIA's `log`** for a chat transcript. `RoleStatus` is the nearest thing
   and is not the same promise.
5. **Heading level 6 is reachable and nothing in the framework goes past 2.**
6. **The themes give `Components.Input` no border.** Blocks widening
   `borderResetTags` to `<input>`/`<textarea>`; a palette decision.
7. **`AccessibilityNestingLevel` has no consumer.** Watch whether the first
   nested list downstream reaches for it or invents something else.
8. **A `<select>` will need the border decision** if a picker node type lands.

Newly raised here:

- **The carried "Next" list is not audited.** One of its three items had been
  wrong for eight sessions and was carried verbatim through six of them,
  because a small item reads as too small to re-check. The cost was low here —
  the item was cheap enough to test in two minutes — but the same list carries
  items whose premises are bigger. Worth spending a moment on the *why* of an
  item before the *what*, especially one inherited from a downstream adoption
  where the reporter saw a symptom and guessed a cause.
- **A widget cannot say "this control is on".** `Deselectable` joins
  `tab`/`tablist` in wanting a selected/pressed state on `Style`. Two
  independent consumers is usually the bar for building the thing.
- **`contrastInk` picks black on `#007AFF`.** Visible in the exported grid: a
  selected cell's numeral and its dots come out `#000000` on the DefaultTheme
  blue. Untouched here — it is pre-existing and orthogonal — but it is the same
  family as the "on-light tone per palette role" item above, and this is now a
  place it is easy to look at.
