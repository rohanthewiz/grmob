# Session: Chip.Prominence — the second question about the quiet state

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (item 2 from "roles-seen-from-the-consumer", same session)

## Ask

"Let's do the Chip.Prominence field", then "now adopt it downstream in
church_mobile". Both halves are here; the downstream doc
`ai_docs/claude_sessions/2026-0905-1738-chip-prominence-adopted.md` carries the
consumer's side.

## The problem, restated from the last doc

The Chip inversion settled *which* state is the loud one — the selected one,
and that is not negotiable, it was the bug. It did not settle *how much*
quieter the other one is, and the first app to adopt the widget wanted two
different answers in two rows:

    the sermons year filter   chrome above the list it filters. A loud row of
                              years competes with the archive. Quiet is right.

    the giving suggestions    four ways to answer the screen's only question,
                              and the fast path most gifts take. Grey pills
                              over an empty amount field do not read as "tap
                              one of these". Quiet is wrong.

The second row spelled its own treatment in four style props through
`UnselectedStyle`. This field is that block, promoted.

## The design

`Prominence` is a package-level string enum, `ProminenceQuiet` (zero) and
`ProminenceLoud`. Loud is the outlined treatment: transparent fill, the accent
as ink and as a 1px rule.

**It touches the unselected state alone.** The selected chip is the theme's
Button base under both values. There is nothing louder to give it, and the
whole inversion depends on it staying the loudest thing in the row — a
`Prominence` that reached both states could walk the widget straight back into
the bug it just climbed out of. Loud is emphatically *not* the pre-inversion
look: that one gave every unselected chip a **solid** fill and left the chosen
one pale; here the fill is transparent, so the selected chip is still the only
solid pill in the row.

**Not Button's `Emphasis`, and the reasons are not stylistic.** Emphasis's zero
value is `EmphasisFilled` — the *loud* one, because a Button with no opinion is
a solid button. Chip's zero has to stay quiet, or the field restyles every chip
already in a tree. Sharing the type would mean the same zero value meaning
opposite things in two widgets a page apart. Separately, Emphasis describes a
whole control where this describes *one of a chip's two states*, and
`EmphasisGhost` has no meaning here at all: a chip with no box is a run of text.

**The accent is the Button base, not `Colors.Primary`.** `chipAccent(t)` reads
`t.Components.Button.Background` — the exact fill the *selected* chip paints —
so an outline and what it becomes when tapped are the same hue on any theme.
Re-deriving from the palette would split them apart on any theme whose buttons
are not primary-coloured, which is the same rule the selected default and
`chipRing` already follow.

Its fallback deliberately differs from `chipRing`'s, and the two now
cross-reference each other on the point. The ring falls back to transparent
because a ring nobody can see is exactly what it wants when there is no fill to
hide against. This colour is *ink*, and transparent ink is an invisible chip,
so it falls back to `Colors.Primary`.

**Precedence: `UnselectedStyle` is checked before `Prominence`.** They are the
same knob at different resolutions — Prominence picks among the widget's
treatments, UnselectedStyle replaces them — so the more specific one has to be
reached first or it could never be reached at all.

## Contrast: measured, then written down

The loud accent over Background is **4.02:1** under DefaultTheme (`#007AFF` on
white) and **7.63:1** under MaterialTheme. Those are not new numbers: they are
literally Button's outlined `default` row, because it is the same colour on the
same backdrop. Only Material clears WCAG AA at the theme's Button font size.

The first draft of the comment said the accent is "normally dark enough to read
on Background". That was an assertion, so it got measured, and it was wrong for
half the bundled themes. The code and the docs now carry the two numbers and
name the override, following Button's own precedent of documenting a bad ratio
rather than silently darkening a hex the theme author chose. The durable fix is
the same one Button names and neither widget can make: a second palette value
per role, an "on-light" tone, which is a theme's decision.

## What was left out

**No strip-level `Prominence` on ChipStrip.** Tempting — a loud strip is the
obvious ask — and refused for the rule ChipStrip's doc already states: it takes
`[]Chip` so that it adds layout and nothing else. A strip-level field is a
second place to configure a chip, and every field added there is one the two
types then have to keep in step. A strip whose chips are all loud sets the
field in the loop that already sets Label and Selected. Written into the type
doc so the next person to want it finds the argument.

`SegmentedControl` needed nothing: its `Segment` is a whole `Chip`, so
`Segment: Chip{Prominence: ProminenceLoud}` already carries to every segment.

## The adoption, and the result worth stating

`app/giving.go` drops the four hand-written props for
`Prominence: components.ProminenceLoud`. **`app/testdata/giving.html` did not
change** — the widget's treatment renders byte-identical to the block it
replaces, so the adoption is a pure reduction with no visual delta to review.

The downstream test moved its assertions to the accent the widget actually
spends, and gained a guard pinning that `themeFor` keeps
`Components.Button.Background` and `Colors.Primary` in step. The app spelled
its outline in `Colors.Primary`; the widget reads the Button base; `themeFor`
writes the brand into both, which is why the hex is identical. If that ever
stops, the test now names which half moved rather than quietly repainting the
row.

## The scorecard, checked rather than assumed

    giving suggestions      loud. The one adopter.
    sermons year filter     quiet. The canonical filter row the doc cites.
    article categories      quiet. Inert labels — a loud outline would
                            advertise a tap that does not exist.
    sermon scripture refs   quiet. A mixed row: some open a browser, some are
                            topical notes with no destination.

One consumer, three correct abstentions. The near-miss is the scripture refs:
loud on *only* the tappable ones would be a real affordance signal for a row
that currently separates them by ARIA role alone. Left alone — it is a design
change to a detail screen, not part of adopting a field — but it is the case
that would first ask for prominence to vary *within* one strip, which the API
already allows precisely because the field sits on the chip.

## Verification

Upstream: `go build`, `go vet`, `gofmt`, `go test ./...`, `-race`,
`wasm/verify/run.sh`, `ios/verify/run.sh`, `GOOS=js` build — all clean. Five
new tests in `chip_test.go`: loud is an outline and not a fill *and* leaves the
selected state untouched; the zero value is the quiet default, pinned against a
quiet chip rendered beside it rather than against a copy of the palette;
`UnselectedStyle` beats `Prominence`; the fill-less-theme fallback is
`Colors.Primary`; the loud treatment still beats a shared `Style`.

Downstream: the same Go set plus `-race`. All 15 screen goldens unchanged.

## Next

Unchanged from the last doc, minus item 2:

1. `core.Role` doc: state the container-mixes-items-with-chrome rule once,
   where both `DataTable` and a hand-written list can find it.
2. Decide the heading-level question before a consumer invents one.
3. `core.Modal` could carry `dialog` itself, the way Button carries `button`.
4. The `<button>` user-agent border divergence on the two DOM renderers
   (`EmphasisGhost` draws a rule on the web and not on the natives).
5. `Calendar.Deselectable` and a counted `Marked` (carried over).
6. Small: "emit `OnEndReached` before the children" in its doc (carried over).
7. Small: the mid-list busy case in `EmptyState`'s doc (carried over).
8. Then Tier C: heading plumbing + `Rotate` + Compass.

Newly raised here, unranked: an "on-light" tone per palette role, which is what
both Button's outlined treatment and this one are working around.
