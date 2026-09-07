# Session: a strip in the middle of a band, a page that landed nowhere, and a palette nobody had looked at

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-fixture-that-was-wrong-and-a-check-that-could-only-pass")

## Ask

"Do the oldest 5 items in the Next list" — the five at age 3.

| # | age·value | item |
|---|---|---|
| 1 | 3·low | A collapsible band's tap target excludes its own padding and its badge |
| 2 | 3·low | A collapsed run is invisible to `OnEndReached` |
| 3 | 3·low | The gomobile stub's *doc comment* is still prose |
| 4 | 3·medium | The `ControlBorder` retint is not visible on any screenshot |
| 5 | 3·low | `boundaryBackdrops` is a list of fills somebody remembered |

All five closed. Two of them turned out to be the same shape from opposite
ends — items 4 and 5 are both "the census knows a number and has never looked
at the thing the number is about" — and items 1 and 3 are both "the fact is
correct and is written on the wrong node."

## Phase 1 — the census derives its backdrops (item 5)

`boundaryBackdrops` was five fills named by hand with `Camera` left out by
name. Every entry was right; the *shape* was wrong, and the item had already
said why: the set is a consequence of `core.ComponentDefaults`, not a decision.
A new field carrying a `Background` — a `Sheet`, a `Popover` — is a fill a
control can sit on, and a hand-written list does not know.

`internal/palette` reflects over the struct instead. Two things fell out:

- **Deriving found two pairs the list had never measured.** `CheckBox fill`
  (Default and Amber) and `Text fill` (all three). Both clear, which is the
  ordinary outcome and not the point: they were unmeasured, and nothing said
  so.
- **The exclusions had to become data with arguments.** `Camera` was already
  argued in prose; `Button` was never excluded because it was never in the
  list, and reflection put it there. Its fill is `Colors.Primary` and it fails
  the floor in all three themes — which is exactly why the exclusion has to be
  a *claim*: an entry that merely made a failure go away would be the census
  defeating itself. The reason it carries is geometric — a bordered control is
  never drawn on top of a filled button; an outline `Button` draws its own edge
  over the page or a panel, and both of those are already measured.

`TestTheBackdropExclusionsNameRealFills` holds each entry to the struct in two
directions, and `TestEveryStatedFillIsMeasuredOrExcluded` is the closure that
matters: it walks `ComponentFields` and `Fill` *directly* rather than through
`Backdrops`, so a derivation that quietly stopped reflecting reports pairs by
name instead of shrinking to a smaller green run.

## Phase 2 — the band's insets are its tap target (item 1)

The item said the fix was not obvious, and named both obstacles: moving the
chrome onto the button would leave the badge without its trailing inset, and
the band `Row` is the node `StickyHeader` has to sit on. Both are real and both
are avoidable, because they are about *different insets*.

```
 Row ────────────────────────────────────      Row ───────────────────────────
│      ┌──────────────────┐     ┌───┐    │    │┌──────────────────────┐ ┌───┐ │
│ 16px │ ▸ January 2026   │ 8px │ 3 │ 16 │    ││  ▸ January 2026      │ │ 3 │ │
│      └──────────────────┘     └───┘    │    │└──────────────────────┘ └───┘ │
 ────────────────────────────────────────      ───────────────────────────────
 the insets are dead space                     the control owns them
```

Nothing moves on screen: padding on a stretched child fills exactly the space
the same padding on its parent held. The `Row` keeps its fill, its cross-axis
centering, `StickyHeader` — and `PaddingRight(MD)`, the badge's own breathing
room, which is the one inset past the control's right edge and therefore the
one that cannot move.

Where the target still stops is at the badge, and that is a decision rather
than a leftover: the count is content, announced separately, so a press on the
number is a press on a thing. The gap before it is chrome and belongs to the
button.

### The knob that had to come with it

`GroupHeader.Style` used to reach the insets and now does not. That is a real
loss and the fix is `ControlStyle` on both bands, which is also the answer to
the half of the item about `CollapseBand`: a caller placing the control in
their own row inherits the same question one level out, and `ControlStyle` is
where their chrome goes.

Both branches take it — the disclosure's button and the plain band's stand-in
Box — because a knob that worked on one would be a relayout arriving the day
somebody added a handler. `TestABandIsTheSameSizeWithAndWithoutItsControl`
pins that.

### The test that noticed first

`TestACallerStylePropOutranksAWidgetsOwnInsets` failed immediately, with its
own guard: "the widget's own Left is already 0 — this case no longer has a
default to reach past." **`GroupHeader` was the only widget in the package
with a live horizontal padding shorthand on its root**, so the naive fix
(`PaddingLeft`/`PaddingRight`) would have deleted that shape from the
repository's coverage entirely.

So `bandInsets` is written as a shorthand with an override beside it —
`PaddingHorizontal(MD)`, then `PaddingRight(trailing)` only when something
follows — which is both the honest expression of the rule ("a full row of
breathing room; less on the side the badge is on") and what keeps the settle
case alive. The two cases now read the control node and clear their side
through `ControlStyle`, so they cover the new field as well.

## Phase 3 — a page that landed nowhere (item 2)

The failure is worse than the item's own framing, and the difference is what
decided the fix. It is not that the next page never loads:

```
shut trailing group, auto-load left on
  scroll to bottom -> fetch page 3 -> 20 rows into a hidden run
  -> the List's child count is unchanged -> the guard closes -> the feed is over
```

`core.OnEndReached` will not re-ask until the row count changes, and a page
absorbed into a shut run changes nothing. So the *first* auto-load spends a
page, every one after it is refused, and the feed reads as exhausted while the
pager's offset has moved on.

The item offered two options — a note, or suppression. Suppression, because
starving is the worse half: the reader has said they do not want to see this
run, and fetching more of it in the background is work nobody asked for and
nobody can look at. The `Footer` stays reachable, which is the other reason
"keep the Footer" was already in that field's doc.

`trailingRun` walks back from the end rather than calling `groupRuns`, since
this runs on every render over a list that can be thousands of rows. The
subtlety is that it must *agree* with `groupRuns` about what the last run's
Group is, or the widget asks its predicate about a group the band never showed
— so it takes the Group from the run's first item and counts the rest, exactly
as the partition does, and `TestTrailingRunAgreesWithTheFullPartition` compares
the two derivations directly.

Three shapes keep the prop and are asserted: a flat feed, an empty one, and a
`Collapse` with a predicate and no handler — the last because `hides` requires
both halves, so those rows are on screen and the edge must follow them.

## Phase 4 — the stub's comment, in rows (item 3)

The declarations in `gomobile_stub.swift` were pinned character-for-character
and the comment above them was a hand-written description of the rules doing
the pinning. It agreed with `gobindSwiftTypes`, `swiftType` and `swiftResult`
because it had been written from them — which is the same copy-that-drifts the
whole stand-in exists to refuse, one level up. It had already started: the
comment's provenance line still said "follows the generated header
(`Headers/Mobile.objc.h`)", which stopped being where the facts come from last
session.

The load-bearing half is a delimited block now:

```
	--- checked against mobile/verify/gomobilestub_test.go ---
	gobind   v0.0.0-20251021151156-188f512ec823
	prefix   Mobile
	suffix   Protocol
	type     string        String?                 String
	type     <interface>   Mobile<Name>Protocol?   Mobile<Name>Protocol?
	results  2   refused: maps a (T, error) pair onto a Swift
	results  3   refused: refuses more than two outright
	--- end ---
```

Three things make it a mechanism rather than a second transcription:

- **Every row is checked against the thing it describes**, never against a copy
  in the test. The version goes to `go.mod`; the suffix is recovered by asking
  `swiftType` about a real interface and subtracting the name; each type row
  calls `swiftType` in both positions; each `results` row builds a signature of
  that arity and asserts what `swiftResult` actually does with it — including
  that a refusal's *message* still contains the phrase the comment sends the
  reader to.
- **The interface row is written with placeholders** and checked by asking
  about an interface literally called `<Name>`. Not a trick: the spelling is a
  pure function of the name, so substituting the placeholder is the general
  case, and a row naming a real interface would go stale on a rename.
- **The rows are counted.** A rule deleted from the comment is verified by
  nothing, which is how it drifted the first time.

The prose around the block is deliberately unchecked. A paragraph explaining
*why* nullability is asymmetric cannot be wrong the way `string String? String`
can.

## Phase 5 — the palette, on a screenshot (item 4)

The item's own words were the hard part: "what is still missing is what to
compare it against, which is the whole question a palette poses." The answer
is not a golden image. It is **the arithmetic the census already performs**.

`components/variant_test.go` proves `#89898E` is 3.12:1 on `#F2F2F7`. It cannot
prove either colour reaches a screen, and everything between the palette and
the pixel is invisible to it: the runtime's style mapping, CSS shorthand
parsing, an alpha channel, an opacity in the composite, a colour profile, a
frame a guard dropped. Every one of those leaves the number true and the
control unreadable.

So `browser.mjs` paints one swatch per pair — a box filled with the backdrop,
holding a smaller box with a 1px frame in the tone, the inner box filled with
the *same* colour so the border has the backdrop on both sides — takes a
`Page.captureScreenshot`, decodes the PNG and reads the pixels back.

**What it asserts is equality with the hex, not a ratio.** The ratio travels
with the table from Go, and recomputing it in JavaScript would be a second WCAG
implementation in one module, which is the one thing a contrast floor cannot
survive: both copies would look right and only one would be. So `Luminance` and
`Ratio` moved out of `components/variant.go` into `internal/palette` (with
one-line forwarders left behind, so all forty-odd call sites are untouched) and
the number is carried across as data.

`wasm/verify/palette.mjs` is pinned to `core.BundledThemes()` in both
directions by `palette_test.go`, which prints the whole corrected table on
failure — regenerating twenty-two rows by hand is how a transcription gets made
in the first place.

### The PNG decoder is eighty lines and that is the price of the promise

`run.sh` promises Go and Node and nothing else. `node:zlib` is built in and
Chrome's screenshots are 8-bit non-interlaced truecolour, so the five scanline
filters are the whole of the format that matters. It refuses everything else
loudly rather than guessing: this check's entire output is "the pixel is
#89898E", and a wrong answer is worse than no answer, because it is a contrast
claim about a colour nothing painted.

### One mutation changed no pixels, and that is recorded rather than hidden

Drawing the 1px frame at `0.4px` was **not** caught. Chrome snaps a solid
sub-pixel border up to one full-strength device pixel at dpr 1, so the
screenshot is identical and the check correctly reports nothing. What the
border sample catches is a *blended* edge — a translucent colour, an opacity, a
dropped frame — and the comment now says exactly that instead of the broader
claim it started with.

## The break-tests

**Thirty run, thirty caught, zero MISSED, zero restore failures.** Plus the
0.4px mutation above, which is not a miss — it changes nothing on screen, and
saying so is the finding.

Same discipline as the last eight sessions: snapshot by hash, one exact-string
mutation, run the suite, restore, assert the restore by hashing, and an anchor
that does not appear exactly once is a `SKIP`.

    item 1   the band's insets and ControlStyle    6
    item 2   the withheld edge                     4
    item 3   the stub's checked block              8
    item 4   the palette pin and the browser       7
    item 5   the derived backdrops                 6

Worth recording what caught what.

- **The palette table cannot break-test itself.** Changing a hex in
  `palette.mjs` changes both the paint and the expectation, and the check
  passes — correctly. The mutations that matter are in the *path between*
  them: backgrounds at 80% alpha, a border colour with an alpha suffix, the
  border guard turned off, and a deliberately wrong Paeth arm in the decoder.
  All four caught, and they are the only shape of break-test this check
  admits.
- **`caller_style_insets_test.go` caught the first draft of the band fix** with
  a guard about its own subject, and the guard was right: the naive spelling
  would have deleted the horizontal-shorthand case from the package.
- **A predicate with no handler withholding the edge** was caught only after
  the case was added — `hides` versus `collapsed` is a one-word difference and
  the original three cases could not tell them apart.
- **`Backdrops` returning only the two palette roles** was caught by the
  closure test and by nothing else. The census itself would have gone green
  with fewer pairs, which is the failure a table that measures things cannot
  afford.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 17 mjs suites + BROWSER PASS
                                (4 keyboard checks + 22 palette swatches)
    ios/verify/run.sh           flex + stack solvers + 14 picker menus
                                + replay + view + app
    android/verify/run.sh       14 picker menus, on a JVM
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New files:

    internal/palette/palette.go        the derived backdrops, the exclusions,
                                       and the WCAG arithmetic
    wasm/verify/palette.mjs            the bundled palettes as the browser paints them
    wasm/verify/palette_test.go        that table, pinned to core

Changed:

    components/grouping.go             bandInsets, ControlStyle on both bands,
                                       trailingRun
    components/grouped_list.go         trailingRunIsShut and the withheld edge
    components/variant.go              two forwarders to internal/palette
    components/variant_test.go         the derived census, the exclusion test,
                                       the closure test
    components/collapse_band_test.go   four tests about where the insets are
    components/grouped_list_test.go    three tests about the withheld edge
    components/caller_style_insets_test.go  two cases moved onto the control
    ios/verify/gomobile_stub.swift     the checked block; stale provenance fixed
    mobile/verify/gomobilestub_test.go the block's parser and three checkers
    wasm/verify/browser.mjs            a PNG decoder, the swatch grid, check 5
    wasm/verify/run.sh                 what the browser pass now covers

## Docs

`docs/components.md`: the band's insets with the before/after picture,
`ControlStyle` on both bands, and the `Collapse`/`OnEndReached` tension.
`docs/concepts/styling-and-theming.md`: the backdrops are derived, the two
exclusions and their arguments, and the pairs are painted.
`docs/platforms/wasm.md`: a new section for the palette check — what it
asserts, what it catches, and the one thing it does not.
`docs/platforms/native.md`: the stub's header comment as a checked block.
`ROADMAP.md`: five new entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 4 · value low) The wrapper test is a rule about a slice nobody
   returns.** `appendRows` appends into the caller's `[]core.PropsAndChildren`
   and returns it, so "can this knob be done to the result" is answered against
   a slice that also holds the container's own props. The opacity assertion
   measures the children it emitted, which is the right subject, but a real
   wrapper would have to find them among the props first.
2. **(age 3 · value medium) The overlay `Layout` is measured and has never been
   mounted.** `GrMobStackSolver` is checked hard, and `GrMobStackLayout` — the
   part that proposes sizes to subviews and reads the `LayoutValueKey` — is
   only type-checked. Three things it does are judgement calls a simulator
   would settle in a minute and no test can: measuring children with the
   incoming proposal rather than an unspecified one, *not* clamping the
   container to the proposal, and re-proposing `bounds.size` at placement.
3. **(age 3 · value low) `AmberTheme` is not used by any example app.** Three
   palettes ship and the tutorial's theme chapter still lists two by hand. This
   session made the gap narrower and did not close it: the browser pass now
   *paints* all three palettes, so the hexes are no longer un-looked-at — but a
   swatch grid is not a screen, and the question a bundled theme exists to
   answer is whether a real one looks right in it.
4. **(age 3 · value low) `wantWitnesses` is a hand-transcribed table that a
   fourth theme silently widens.** The census derives its theme list from
   `core.BundledThemes()`, so a new palette is asked every rule — and its
   answers land as a diff against a map written by hand. What is not stated is
   what a new row should look like when somebody is adding a theme rather than
   debugging one: "witnessed by nobody" and "witnessed by everybody" are both
   legal and mean opposite things. Directly comparable to the exclusion table
   this session wrote, which took the other road.
5. **(age 2 · value medium) `grMobValue`'s three-way branch is still checked
   by `strings.Contains`.** The runner exists and the branch is out of its
   reach, because `grMobValue` is a `SemanticsPropertyReceiver` extension and
   the harness has no Compose. The move is the one `GrMobSelectMenu.kt` made:
   split the branch into a pure function in a file that imports nothing. What
   that needs first is a Go authority to compare against, and there is none —
   htmlout passes the three ARIA attributes straight through, which is a
   different mapping from Compose's three-way one.
6. **(age 2 · value low) A `Spacer`'s size prop clobbers its own Style.**
   `applySpacerSize` writes width, height and flex-shrink, and `renderNode`
   calls it *after* `createElement` — so a Spacer carrying a `Width` in its
   Style loses it to a prop that is not exempt from anything. Nothing in
   `core` builds such a node; a hand-assembled one reaches it.
7. **(age 2 · value low) A heading with no options under it is still
   inexpressible.** A `Group` is a string on an option, so a section that
   exists to say "nothing here yet" has no way to be written. The blocker is
   authoring rather than rendering: it needs a heading declared independently
   of an option, which means a second list and a matching problem the run-based
   reading was chosen to avoid.
8. **(age 2 · value low) `browser.mjs` still asks one page four questions.**
   Advanced again rather than closed: it grew a fifth check, and that check is
   the first one that is not about the keyboard, which is the real move. What
   is still absent is anything about *layout* — whether a focus ring is visible
   on a themed control, whether an `aria-live` region announces, whether a
   `<select>`'s menu opens where the theme's frame says it should. Adding one
   is a function call, and there is now a screenshot decoder to build it on.
9. **(age 1 · value medium) A toolbar containing a composite is two tab stops,
   and only a comment says so.** `focusableMembers` stops at a nested
   composite and does not take it as a member, so a `tablist` inside a
   `toolbar` keeps its own roving tabindex. Every control stays reachable,
   which is why this shape was chosen, and it is not what ARIA describes — the
   pattern makes the nested widget's *current* member the outer widget's
   member. Doing that means two composites writing tabindex on one element,
   which needs an owner rule the section does not have.
10. **(age 1 · value medium) `CONTROL_ROLES` is two roles, hand-written, and
    core can grow a third.** The runtime treats a `Box` as one of a toolbar's
    controls when it carries `role="button"` or `role="link"` *and* an OnTap.
    Those are exactly the two `core.Role` documents as "a tappable container",
    and `wasm/verify/keynav_test.go` pins the pair — but the pin is against a
    hand-written list, not against the property. A future `RoleCheckbox` or
    `RoleSwitch` would be a control on the same argument and would silently not
    be a toolbar member. What is missing is a name in `core` for "roles that
    make a container a control".
11. **(age 1 · value low) The refusals table's `Shape` is prose again, with
    braces around it.** `Blocked` is derived from `core.Roles()` and checked,
    which is what makes the table a mechanism; `Shape` — "two dimensions", "a
    submenu with its own Escape" — is a sentence nothing can contradict. There
    is no obvious authority for it, which is the honest reason it is not
    checked, and saying so is not the same as checking it.
12. **(age 1 · value low) Selection-follows-focus has no browser check.** It is
    covered thoroughly in `keynav_test.mjs`, where `focus()` is an assignment
    and a dispatched callback is an array push. The two facts a browser would
    add are that the *real* focus move fires it exactly once and that a
    `<button>` receiving both an arrow and its own click behaviour does not
    double-fire. `browser.mjs` already records dispatches in
    `window.__dispatched` and asserts nothing about them.
13. **(age 1 · value low) `aria/gen` is not on any verification path, and a
    stale download regenerates differently.** The conformance test compares the
    fixture against whatever HTML is on disk, so a checkout holding an ARIA 1.1
    copy would fail with a diff that looks like a fixture error rather than a
    stale-download one. The spec's own version string is in the document and
    nothing reads it.
14. **(age 1 · value low) `CollapseBand` is not used by any example.** Its only
    readers are its tests. This session gave it a second field (`ControlStyle`)
    whose whole justification is how a real custom band is assembled, which
    makes the absence of one a slightly bigger hole than it was.
15. **(age 1 · value low) The `data-grmob-selection-follows-focus` attribute is
    written on any node that asks.** `applyAccessibility` does not consult the
    composite tables, deliberately — knowing them there would put the tables in
    two places — so the flag on a `list` or a plain `Box` is an attribute nobody
    reads. That is also a claim in the DOM that is not true of the element. The
    audit is the natural place to report it and has no rule for it.
16. **(age 0 · value medium) The band's insets are on the control and no
    renderer but the web has been asked about it.** The change is padding moved
    from a parent to a stretched child, which is exactly identical in CSS flex
    and is an *assumption* everywhere else: SwiftUI's `padding` on a view inside
    an `HStack` and Compose's `Modifier.padding` both behave the same way, and
    neither was measured. `ios/verify`'s stack solver and `browser.mjs` between
    them could settle it — the solver knows about proposals and the browser can
    now read pixels — and nothing currently asks either.
17. **(age 0 · value low) A `Header` override still cannot be told the edge was
    withheld.** `GroupedList` withholds `OnEndReached` when the trailing run is
    shut, and the reader's only cue is that scrolling stops loading. A feed
    whose `Footer` is a bare `LoadMore` shows "Load more" and works; one that
    hides the footer when `HasMore` is false shows nothing and looks finished.
    The widget knows which case it is in and says nothing to anybody.
18. **(age 0 · value low) The palette check paints swatches, not widgets.** The
    pairs measured are `(ControlBorder, fill)` from the theme, drawn by
    `browser.mjs` itself. What a real `Input` or a quiet `Chip` puts on screen
    goes through `components`, which the browser pass does not run — so a widget
    that stopped reading `Colors.ControlBorder` would keep passing this check
    while drawing an edge nobody measured. `TestBundledFieldFramesAreTheControlBorderRole`
    covers the two named widgets in Go; the general case is unheld.
19. **(age 0 · value low) `internal/palette` reflects and `core` cannot tell
    it not to.** A `ComponentDefaults` field whose `Background` is a fill no
    control is ever drawn on has to be excluded by name in a package the theme
    author does not edit. A struct tag would put the fact next to the field,
    where somebody adding one would see it — at the cost of a tag `core` reads
    for a test's benefit, which is a trade nobody has argued yet.
20. **(age 0 · value low) The two-result gobind arm is still transcribed
    nowhere.** `swiftResult` now names which of the three arms a refusal is
    about and the stub's comment is checked against that naming, so the *report*
    is right — and a `(T, error)` bridge function would still be refused rather
    than declared. The mapping is read and understood (`funcSummary`); what is
    missing is a bridge function of that shape to make it worth writing.
