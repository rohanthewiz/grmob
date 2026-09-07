# Session: a command nobody ran, two columns nobody reached, and a browser nobody asked

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-void-that-lost-its-style-a-conversion-nobody-ran-and-a-download-nobody-dated")

## Ask

"Do the oldest 5 Next list items", then mid-session: "Once complete continue
with the next 4 oldest items in the Next list. At the end wrap the session."

| # | age·value | item |
|---|---|---|
| 1 | 3·low | `aria/gen` runs on no verification path |
| 2 | 3·low | The two exception tables for themes are empty |
| 3 | 3·low | The sticky check writes `core.StickyHeader()`'s declarations by hand |
| 4 | 3·low | `ValueRange.Progress` has no Go consumer |
| 5 | 3·medium | `internal/valuefixture` is compared on one platform |
| 6 | 2·low | A `Header` override cannot be told the edge was withheld |
| 7 | 2·low | The palette check paints swatches, not widgets |
| 8 | 2·low | `internal/palette` reflects and `core` cannot tell it not to |
| 9 | 2·low | The two-result gobind arm is transcribed nowhere |

All nine closed. Five of them (1, 2, 4, 5, 9) turned out to be the same shape:
**an arm that cannot run against the data the repository actually has**, which
is a different problem from an arm that is wrong, and wants a different fix in
each case — a synthetic input, a pure function, a new consumer, a browser, or a
correction to the prose that stood in for the arm. Items 4 and 5 met in the
middle and reinforced each other: the Go audit finding and the browser check
are two reports of one divergence, and each is the other's justification.

## Phase 1 — the command with no test (item 1)

`aria/gen` is four things aria/spec is not: a walk up to the module root, two
paths, the error a missing download produces, and the write. `go test ./...`
reached none of them.

Every failure available to it is quiet, which is the argument the floors in
`spec.Parse` already rest on:

```
   the wrong path written   a fixture regenerated where nothing reads it, and
                            a stale aria.json every guard still passes against
   the wrong path read      LocalPath and FixturePath are siblings under aria/
   a partial run            an error after the write leaves a fixture from a
                            document the command then refused
   the missing download     the one error a human actually meets
```

`syntheticSpec` builds a document satisfying three things at once — both of
Parse's floors, the version guard, and Scope's requirement that every scoped
role have a section — and the tests run `run` from **three working
directories**, which is what makes `repoRoot` the subject rather than an
implementation detail. The expected bytes are never written out: each test
compares against `spec.Scope` and `Fixture.Render` on the same document.

`run` took an `io.Writer` so the summary line's two counts became checkable.
`main()` is now the one line here nothing covers, which is the right line for
that to be.

## Phase 2 — the two columns nothing could reach (item 2)

`quietThemes` and `universalThemes` are empty and have been since they were
written. That is the honest state — every bundled palette witnesses some rules
and not others — and it means **five of the six arms reading them had never
run**. An empty map cannot hold a stale name or a missing reason, and neither
degenerate column is reachable from three ordinary palettes.

The fix is not an invented entry. The classification moved into
`landingComplaints(carried, rules, quiet, universal) []string`, and
`TestTheLandingArmsEachReportTheirCase` drives every arm over constructed theme
sets, each "must complain" case written beside the "must stay silent" one:

```
   quiet + unrecorded      complains        quiet + a reason        silent
   universal + unrecorded  complains        universal + a reason    silent
   ordinary + recorded     complains        ordinary + neither      silent
   an entry for a theme that is gone / an entry with no reason      complains
```

The real tables stay empty. One thing the refactor turned up: with
`paletteRules` emptied, every theme is both quiet and universal, and the switch
takes the quiet arm — so a census whose rules had all been deleted would report
every palette as quiet, a true statement about a table that no longer says
anything. `TestEveryKnownThemeLandsSomewhereStated` now refuses that before it
counts anything, and the arm ordering it depends on is pinned separately.

## Phase 3 — three declarations, written out (item 3)

`browser.mjs` is the only pass that can tell a pinned band from a band with
`position:sticky` written on it, and its fixture stated the three declarations
itself. `palette.mjs` is in the same position with the bundled hexes and is
pinned; this one was not.

`STICKY_DECLARATIONS` is the one statement now, spread into the band's Style,
and `sticky_test.go` holds it to `core.StickyHeader()` applied to an **empty**
`core.Style` — so the comparison is the *set of fields the prop touches*, not
three names typed on each side:

```
   value check only   passes a prop that grew a fourth declaration
   field-set check    fails, which is the change that matters — the browser
                      would be pinning a band the framework no longer builds
```

`core.Style` marshals every field including the zeros, so the diff against a
zero Style is total.

## Phase 4 — a reading with no reader (item 4)

`core.ValueRange.Progress` had two readers: a test and a Kotlin
transliteration. `components.ProgressBar` states all three numbers itself and is
determinate by construction, so nothing in Go had ever needed the reading.

`core.AuditTree` needs it. `ConcernUnusableValueRange` is the seventh
accessibility finding, and it is the one shape in this vocabulary that fails
**differently on every target**:

```
                        Now: "half"          Min: "9", Max: "1"
   web (Chrome)         a bar pinned at 0    a bar pinned at 9
   Compose              indeterminate        the property is dropped
   SwiftUI              nothing either way   nothing either way
   core.Progress        indeterminate        empty-range
```

None of those is what was written, nothing errors, and the bar looks correct on
screen in every case — the fill is drawn from the caller's own float, which
never went through this vocabulary at all.

The check needed a second question `Progress` could not answer. From outside, a
position that failed to parse and one that was never stated are the same
reading, and only one of them is a mistake — so `ValueRange.Unparsed()` names
the stated fields that are not numbers, and the audit uses both: `Unparsed` for
the parse failures, `Progress().Reading` for the empty range. `Text` is outside
both, by design.

The role is deliberately not part of the guard. A range on an unroled node is
dropped by both exporters by design and is not a finding; a *broken* range on
that node is the author's own numbers either way.

## Phase 5 — asking the browser (item 5)

`internal/valuefixture` reached one target: android/verify's JVM pass. And the
web is not a target that merely transliterates this rule — it is the one that
**does not implement it**. Both DOM exporters write the attributes verbatim on
the argument that a browser applies ARIA's rules itself, which is right, and
which means nothing here had ever watched one do it.

`valuerange.mjs` carries the table (pinned to the fixture and `core` by
`valuerange_test.go`, both directions), and `browser.mjs` mounts one
`progressbar` per row and reads the answer out of **Chrome's own accessibility
tree** — not the DOM, because the attributes are what this runtime wrote and the
question is what the browser made of them.

What agreement means is per reading, and each difference is the reading's own
meaning:

```
   determinate    all three numbers
   indeterminate  no position. The bounds are NOT compared: a progressbar
   unstated       reports 0..100 whether or not one was written, so comparing
                  there would pass for the wrong reason
   empty-range    the bounds as stated. Chrome clamps the position to whichever
                  end it can reach and Progress reports it unclamped; both are
                  honest answers to a range that is not one
```

The browser agrees on everything that parses: the implicit 0..100, one stated
bound leaving the other at ARIA's default, indeterminate by omission, clamping
in both directions.

**And it disagrees, exactly where `Unparsed` says it will.** Chrome reads
`aria-valuenow="half"` as 0 and pins the bar at the start; it reads
`aria-valuemax="lots"` as 0, inverting the range, and then clamps a bar at 45%
into announcing as complete. So the table carries `parses` and the two halves
are asserted in opposite directions — a row that parses must agree, a row that
does not must **not**. The second is a pinned divergence: a browser that started
applying ARIA's defaults there fails this pass, which is when somebody should
hear about it.

One collapse worth recording: in a browser, `unstated` and `indeterminate` are
the same node. Both report no value, because ARIA spells indeterminate by
omission and an unstated range writes no attributes at all.

## Phase 6 — a silence with no way to ask about it (item 6)

`GroupedList` withholds `OnEndReached` while the trailing run is shut, and that
is right. It is also invisible: a feed that stopped fetching and a feed that ran
out produce the same tree and the same experience.

`AutoLoadWithheld()` is the question. A method rather than a field because it is
derived — `Items`, `GroupBy` and `Collapse`, none of which means anything alone
— and the value has to be built before it can be asked, which is why a
conditional `Footer` is assigned after the literal.

It answers false with no `OnEndReached`: there is no sensor to withhold on a
manual pager, and `Collapse.IsCollapsed` is the question about the run itself.
The test's real claim is not the four answers but that the method **agrees with
the tree** — an answer that disagreed would send a screen into showing two ways
to load one page, or hiding the only one.

## Phase 7 — a model of a boundary, and the real thing (item 7)

The swatch grid answers everything about the route from a hex to a pixel and
nothing about the route from the palette role to the hex, because that route
runs through `components` — which is Go, and `browser.mjs` mounts JSON.

`gen.go` renders one quiet `components.Chip` per bundled theme on a page painted
in that theme's own `Colors.Background`, and reads three colours off the
**rendered node**. Those trees ride in the transcript, which is why `run.sh` now
hands `browser.mjs` `GRMOB_TRANSCRIPT` and why the script refuses to run
without it — a missing transcript is a fact about the invocation, not about the
machine, so it is a failure rather than a skip.

Two sampling lessons, both found by the check failing:

```
   the fill    read at the chip's CENTRE -> #8D8D90 out of an #F2F2F7 chip.
               That is the label. Sampled in the leading padding instead.
   the ring    scanned across the LEFT edge, as the square swatches are ->
               #907267 out of an #8D6E63 ring. A pill's leftmost point is the
               apex of a curve. The top edge at mid-width is straight at any
               radius.
```

What the browser cannot check is whether those three hexes are the census's, so
`widget_test.go` is that half in Go: the ring must be
`Colors.ControlBorderColor()`, both backdrops must be fills `internal/palette`
derives from `ComponentDefaults`, and the ratios carried across must be
`palette.Ratio`'s.

## Phase 8 — the exclusion moves to the field (item 8)

`internal/palette` derives the backdrop list from `core.ComponentDefaults`
precisely so a new field cannot go unmeasured — and the *exception* to that
derivation was a map in a package the theme author never opens, invisible in the
diff that adds a field beside it.

The `notbackdrop` struct tag is the fact in its own place, its value the
argument. `palette.NotABackdrop()` is the reading. Two failure modes stop being
checked and become impossible: a tag cannot name a field that does not exist,
and it cannot drift from the field it names.

Long single-line tags are the cost, and they are the right cost: the argument
travelled verbatim rather than being summarised into a shorter one somewhere
else.

## Phase 9 — reading the generator to the end (item 9)

The two-result arm was refused because no bridge function had the shape. Reading
the pinned `bind/genobjc.go` through turned up a second obstacle, and it is the
one that decides the spelling:

```
   genFuncH:  g.Printf("FOUNDATION_EXPORT %s;\n", s.asFunc(g))
```

Every symbol this bridge exports is a package-level func, so every one is a
plain **C function**. The Swift `throws` spelling is the Objective-C *method*
convention — Clang rewrites a trailing `NSError**` into `throws` for methods and
does not for C functions, which have none without an explicit `swift_error`
attribute. gobind emits none. So the old message named the wrong thing:

```
   func F() (string, error)  ->
   FOUNDATION_EXPORT NSString* _Nonnull MobileF(NSError* _Nullable* _Nullable error);
```

and what Swift makes of *that* is the one fact not in the module cache. It is in
the importer. The refusal now says so, and says the fix is one `gomobile bind`
rather than a header hunt.

`TestTheResultArmsAreReadOffThePinnedGobind` turns the rest from prose into a
check: it reads `bind/genobjc.go` and `bind/types.go` out of the module cache
and holds all four claims to them — the C-function emission, the
`isNullableType` split, the `BOOL` arm, the three-result refusal. It skips when
the cache has no copy, which is the ios/verify stance; `x/mobile` is held in
go.mod by the tool block and by nothing else, so `go build ./...` never fetches
it.

## The break-tests

**35 caught, 4 misses (3 by design, 1 a real limit), 0 restore failures.**

    item 1   the two paths, the walk, the two errors, the ordering   6
    item 2   five arms, the empty census both ways                   6
    item 3   the fixture, the prop's fourth field, the spread        4
    item 4   the call, both arms, three Unparsed rules               6
    item 5   the table both ways, the switch, both assert halves     5
    item 6   the method three ways, and Render's use of it           4
    item 7   the border, the alpha, the ratio, the theme, the fill   5
    item 8   the key, the reason, the skip, both tags                4
    item 9   two readings, and the stub header                       3

Four misses, and they are not the same kind.

- **`Camera`'s exclusion can be dropped silently.** Removing its `notbackdrop`
  tag makes it a backdrop, the census measures the pair — and it *passes*:
  `#89898E` on black is 6.03:1. `Button`'s exclusion is load-bearing (Primary
  as a fill is 2.17:1 and the census fails), so the two tags are not the same
  kind of claim. This was equally true of the map it replaced; the tag neither
  fixed nor worsened it. Named below.
- **`checkValueRange`'s `!v.Stated()` early return is an optimization.** A zero
  range produces no `Unparsed` fields and reads as `Unstated`, so deleting the
  guard changes no behaviour. Correct as written; recorded so nobody reads the
  miss as a hole.
- **The empty-census guard cannot be mutated, only its subject.** Disabling it
  while six rules exist is unobservable; emptying `paletteRules` is caught.
  Same shape as last session's stale-download skip.
- **The browser's divergence half is unfalsifiable by mutation today.** Deleting
  the `!row.parses && agrees` report changes nothing while Chrome disagrees.
  Its subject is somebody else's software, and the half that *can* fail (the
  `parses` rows) is caught.

One test was weakened by discovery and says so: `TestTheWidgetSwatchRingIsThe
PaletteRole` cannot catch a chip that read `Components.Input.BorderColor`
instead of the role, because the two hold the same hex in all three bundled
themes. That is a difference in provenance with no observable consequence until
somebody restyles their fields — which is the day `chipRing`'s argument is
about. The doc comment states the limit rather than implying the name.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 17 mjs suites + BROWSER PASS
                                (22 swatches, 3 real widgets, a sticky band,
                                 22 value ranges through Chrome's AX tree)
    ios/verify/run.sh           flex + stack solver + 15 picker menus
                                + replay + view + app
    android/verify/run.sh       15 picker menus and 22 value ranges, on a JVM
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    aria/gen/main_test.go              the synthetic spec and the four failures
    wasm/verify/sticky_test.go         the field-set pin
    wasm/verify/valuerange.mjs         the fixture as the browser mounts it
    wasm/verify/valuerange_test.go     its pin, and both sides of the divergence
    wasm/verify/widget_test.go         the swatches held to the census

Changed:

    aria/gen/main.go                   run(io.Writer), dest
    components/palette_witness_test.go landingComplaints and its arms
    components/variant_test.go         the tag reading, in both tests
    components/grouped_list{,_test}.go AutoLoadWithheld
    core/value.go                      Unparsed
    core/value_test.go                 its cases and the Progress property
    core/a11y_audit{,_test}.go         ConcernUnusableValueRange
    core/theme.go                      the notbackdrop tags and their doc
    internal/palette/palette.go        NotABackdrop() as a reading
    ios/verify/gomobile_stub.swift     the results-2 rule
    mobile/verify/gomobilestub_test.go the corrected refusal, the cache reader
    wasm/verify/gen.go                 widgetCase, widgetCases, ratioBetween
    wasm/verify/browser.mjs            two new checks, the transcript, the
                                       sticky constant
    wasm/verify/run.sh                 GRMOB_TRANSCRIPT for the browser pass

## Docs

`docs/platforms/wasm.md`: two new sections (a real widget's paint, with both
sampling lessons; a browser's own ARIA value rules, with the divergence table),
plus the sticky pin. `docs/concepts/debug-mode.md`: the
`unusable-value-range` row. `docs/concepts/styling-and-theming.md`: the
`notbackdrop` tag and the widget swatches. `docs/components.md`:
`AutoLoadWithheld` with the conditional-footer shape. `ROADMAP.md`: nine new
entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 3 · value low) A `Header` override still cannot be told the edge was
   withheld.** `AutoLoadWithheld` answers the *caller*, who can then build a
   footer. A `Header` override is a view the widget hands a `Group` to, and it
   is handed no such fact — so a band that wanted to say "collapsed; auto-load
   is off" has to close over the same three fields the caller just asked the
   widget about. `Group` growing a field is the obvious move and is a wider
   change than it looks: it is on the path of every band in every list.
2. **(age 3 · value low) `internal/palette` reflects and `core` still cannot
   tell it a fill is *reachable*.** The `notbackdrop` tag says a fill is never
   drawn on. What no tag says is which widget draws on which fill, so the census
   is a full cross product — every tone against every fill — and a pair nothing
   builds is measured beside a pair three widgets build. The widget swatches
   (phase 7) are the first evidence of a *real* pair; three of them against
   twenty-two arithmetic ones.
3. **(age 2 · value low) The palette check paints one widget per theme.** Item 7
   closed the "no widget at all" hole with a quiet `components.Chip`. An `Input`
   frame is the other half of `ControlBorder`'s job and is not painted — it
   exports as `<input>`, whose user-agent border is a second declaration on the
   same edge, which is exactly why it wants a browser rather than a Go test.
4. **(age 2 · value low) The two-result gobind arm is still transcribed
   nowhere**, and now for a stated reason: the Swift importer's treatment of a C
   function with a trailing `NSError**`. One `gomobile bind` on a Mac settles
   it, and the readings around it are checked as of this session.
5. **(age 1 · value medium) The band's insets are on the control and no renderer
   but the web has been asked about it.** Padding moved from a parent to a
   stretched child, which is exactly identical in CSS flex and is an
   *assumption* everywhere else. `ios/verify`'s stack solver could settle the
   SwiftUI half and nothing asks it.
6. **(age 1 · value medium) A hand-assembled `Spacer`'s children are dropped on
   both natives.** Both DOM renderers emit a Spacer's children like any other
   element's; a Compose `Spacer` and a `Color.clear` are leaves.
7. **(age 1 · value medium) The nested-composite finding is reported in Go and
   asserted in JavaScript, and the two are not held together.**
   `core.KeyboardComposites()` is pinned to the runtime's tables; the *walk's
   stopping rule* is not.
8. **(age 1 · value low) `swiftTypeBody`'s cut is "the first closing brace in
   column one".** True of every Swift type in `Renderer.swift` and not true of
   Swift.
9. **(age 1 · value low) `notTappable`'s reasons are prose by construction.**
   The four kinds it sorts roles into are almost a derivable property, the way
   the refusals table's `Nesting` now is.
10. **(age 1 · value low) The stale-download skip is a guard nothing
    exercises.** Its subject — `spec.Parse` refusing 1.1 — is covered; the skip
    itself has never run.
11. **(age 0 · value medium) A dropped `notbackdrop` tag is silent when the pair
    it exempted happens to pass.** Removing `Camera`'s tag adds a pair that
    clears 6:1 and nothing says a geometry claim was deleted; removing
    `Button`'s fails the census, because that one's number is 2.17:1. So half
    the exclusions are load-bearing by accident. What would close it is the
    reachability fact item 2 above wants — an exclusion is a claim about which
    widget draws where, and nothing can check it against a list of widgets that
    does not exist.
12. **(age 0 · value low) The browser's value-range divergence half cannot fail
    on this machine.** A row with an unparseable number must *not* agree with
    `core.Progress`, and it does not — so the assertion has never fired and a
    mutation of it is invisible. It is a claim about somebody else's software,
    which is the point; what it means is that the half of the check that reads
    as a discovery is exactly the half no local mutation can defend.
13. **(age 0 · value low) `TestTheWidgetSwatchRingIsThePaletteRole` cannot see
    the swap it is named for.** `Components.Input.BorderColor` holds the same
    hex as the role in all three bundled themes, so a chip reading the field
    base passes everything. A fourth bundled theme that split the two would make
    the whole family of checks sharper — and would be a palette added for a
    test, which is the argument against.
14. **(age 0 · value low) `browser.mjs` now requires the transcript and skips on
    no Chrome.** Two different preconditions with two different stances, both
    right and neither obvious from outside: `run.sh` always supplies the first,
    and a standalone `node browser.mjs` fails with a sentence rather than
    checking two thirds of what it says it checks.
15. **(age 0 · value low) The widget swatches mount one tree per theme and
    screenshot each.** Three Chrome round trips where the swatch grid does one,
    because each page paints its own background. A single page carrying all
    three would halve the browser pass's slowest section, at the cost of the
    per-theme page fill being a sibling rather than the document.
