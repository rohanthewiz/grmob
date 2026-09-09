# Session: two edges for a third, a copy in more than one construct, and a probe that earned its round trips

Session: https://claude.ai/code/session_01MQcEV7mAeEV4h1LywCdQhe
Date: 2026-09-09 (follows "a-premise-priced-in-pixels-a-probe-that-asked-at-the-wrong-size-and-a-lexer-that-knew-which-construct")

## Ask

"Do all items in the Next list." Six items, all raised last session, all
closed. Five files, no new ones — every item landed where its complaint was.

| item | where | shape |
|---|---|---|
| 1 | themenearmiss_test.go | a fraction with no edges |
| 2 | pinfixture_test.go | one construct per copy |
| 3 + 4 | browser.mjs | six reads for a noun, two readings never joined |
| 5 | inkglyph_test.go | a documented gap, unmeasured |
| 6 | checknumbering_test.go | a recital with no bound |

---

## Item 1 · a third, and what it had to sit between

`affordedEndingShare` was a third of each ending's own measurement — better
than one constant for four endings, and still one fraction chosen once.
INK_OWN_SHAPE_FLOOR's note is explicit about what makes its own quarter a
measurement rather than a taste: the pair of populations it has to separate,
both in hand on every run, and both asserted. A third had neither.

The two edges are the two findings the census exists to tell apart, and one of
them was already named in prose and computed by nothing:

    affordedSetCount    2×(leaves−w+1) at each width — the walk as arithmetic,
                        and the walk is asserted against it before anything
                        divides by it
    affordedScale       this run's population over the recorded one. One
                        number, because one walk produces all four endings
    upper edge          measured × scale is what a merely-MOVED ending scores.
                        A floor above that is a floor no healthy census clears
    lower edge          an ending whose share falls under the constant is held
                        to a number that is not a measurement of IT — right for
                        a thin arm, and the derivation gone quiet when it is
                        true of all four

`affordedEndingBracket` now returns a struct carrying which arm set the floor,
and `affordedShortfallCause` decides — rather than hedges — which of the two a
shortfall is: the scale predicts a count, and the count is in hand.

The log line says what each ending was actually held to, by which arm:

    3 of the four on a share of its own measurement and the rest on the stated
    10, over a population at 1.00× the one the record was taken on:
    "…crowding stopped the search" 27/10 (the stated floor),
    "no width crowds these names at all" 473/157 (a third of its own), …

**Break-tests — 4 run, every one fired**

    window 1        every ending's floor above what a moved population would
                    score: "at 0.17×, which puts 473 at about 81"
    window 3        the cause computed: "the population is at 0.51×, which
                    predicts about 192 sets for this ending and 69 arrived.
                    The others moved with the walk; this one went."
    share 100       no ending on the share — "one number holding four endings,
                    which is the state affordedMeasuredOn was recorded to
                    replace, arriving back through the arithmetic"
    formula +1      the walk and affordedSetCount disagree; nothing further is
                    asked, because the scale would be a ratio against a
                    population nobody walked

---

## Item 2 · a copy in more than one construct

`pinBlankedBy` returned the first recorded kind in a copy's span and called
that the construct. Its own comment argued a straddle would be caught by the
multi-construct clause — it cannot: that clause is about two COPIES in two
constructs, and one byte per copy is one construct per copy.

Straddles are reachable, and the reason is the strip walk. The phrase is
matched over the space-stripped source and a newline is whitespace, so the two
transitions that need no delimiter at all — a line comment and a single-line
string both end AT the newline — let a phrase begin in one construct and finish
in whatever the next line starts with.

    x = 1 // pinSame(total,      head: a line comment's
    c.offer) + 2                 tail: code the lexer kept

That copy is half deleted, and the span test called it the surviving assertion
and named line 1 as the line to open.

    pinBlankedAs      every construct the span meets, in order — and a byte is
                      read as kept code only when nothing was recorded against
                      it AND nothing is standing there, because pinCodeOnly
                      leaves newlines alone and a wrapped copy would otherwise
                      straddle its own construct and a newline
    straddling        a fourth bucket beside prose / shadowed / live: neither
                      "blanked" nor "left standing" is true of the whole copy
    pinShadowParts    grouped by the whole RUN of constructs, "inside" for one
                      and "across" for several, names joined with "then"
                      because the order is where the reader looks first

**Break-tests.** The first kind only: *"the copy's own bytes were taken by 1
construct(s)"* — the fixture stops being the straddle it is there to be. The
newline skip removed: the three-construct fixture reports its kinds in the
wrong order AND a block comment that swallowed a phrase across two lines
arrives as a copy half standing in code — the false-straddle flood the skip is
for.

Three fixtures, one of them a comment, a closed string literal and live code in
one copy.

---

## Items 3 and 4 · six reads that now feed a decision

These closed together, and that is the answer to item 3 rather than a
coincidence. `inkCanaryFaceList` cost a DOM.querySelector and a
CSS.getPlatformFontsForNode per distinct request — six on this grid — for a
sentence the premise no longer consults.

`INK_CANARY_REQ_JS` is the request key (style, weight, size), one expression
used by the probe mount AND by the canvas measurement, because a join on two
spellings of a key is a join that silently matches nothing. Each measured row
now carries its `req` and the family it was measured with.

    inkCanaryAgreement   the probe's NAME and the advances' DISTANCE, joined
                         per request. Same face, same request, same canvas,
                         same string: one advance — so one name over two
                         advances is one of the readings being about something
                         else
    the complement       two names over one advance is inkCanvasFallbackFault's
                         blind arm, reported there. Each direction held by the
                         reader that decides it, said once between them
    inkCanaryFaceSpread  mounted / read / distinct answers, recited by the tail

The tail now says what the six reads bought:

    read at 6 of the 6 distinct requests the joined runs make rather than once
    at whatever the page default is, and answering with one face at every one
    of them, which is this stack having no optical cut at these sizes … those
    two readings of one question held against each other on 28 runs where both
    were in hand and agreeing on 28

**Break-tests.** The probe mounted with the run's own family instead of the
generic: *"the probe says Times, which is the family this run is drawn by, and
the same run's own string measures 65.720703px differently under the two
requests"*, over 28 runs. One probe moved to `serif`: caught by the same fault,
because serif reaches Times here — the fault doing real work on a probe asked
the wrong question. One probe moved to `sans-serif`: rc 0, and the tail reads
*"2 different faces across them, which is a family list answering per size and
the reason one probe is not enough"*, with inkCanaryFaceList's long form naming
each request.

One thing found while writing it: a backtick inside a `//` comment that lives
inside a template literal closes the template. `node --check` reports it six
hundred lines later, in the middle of an unrelated call.

---

## Item 5 · how wide the gap is

`TestTheTwoFoldsAgreeOnTheFormsTheyBothCarry` asks only about inputs both sides
claim, and says so. The printable-ASCII arm is what makes the rest safe — the
same lean item 3 of last session removed from the PAIR question, still holding
up everything else, with no reading of how far it reaches.

So the whole plane is walked. browser.mjs's real `inkFold` runs over every BMP
code point under node and each answer is compared with `inkGlyphFold`'s.

    browser.mjs's fold changes 15802 of this plane's code points; gen.go's
    agrees on 748 and is narrower on 15054, 299 of which NFKD turns into a
    letter a seed is spelled with (e.g. U+00CC "Ì"→"ì", where gen.go says "ì")
    — and 0 of it inside printable ASCII

Two edges asserted, and one string end to end. The gap may not reach printable
ASCII, because that is the whole of what the second arm refuses on. gen.go may
not be WIDER anywhere, because a fixture refused for a pair inkLigatureNote
could never report sends its author after a ligature the other end does not
believe is there. And the witness — `"fÌ"`, which the note's fold turns into
`"fì"` and gen.go's does not — is refused, by the ASCII arm, which is what "the
printable-ASCII arm is what makes that safe" means when it is asked of a string.

**Break-tests — 3 run.** The ASCII arm widened to 0xff: *"inkGlyphPerCharacter
accepts \"fÌ\""*. A bogus `'¡': "fi"` in the table: the wider-than-NFKD
arm. `'A': "z"`: the printable-ASCII edge, on U+0041.

---

## Item 6 · a recital with a bound

Every `why` in full was the right direction and grew with the rows: ~600
characters at two exemptions, a paragraph at three, in a PASSING run's log.

    citationLendersNear   directory is the one thing in hand that correlates
                          with content at all — a silent .go file in
                          wasm/verify sits beside a .go exemption written about
                          wasm/verify. Alphabetical within each group, so the
                          stability the single lender was picked for is kept
    citationClipWhy       cut at a word, marked, at 120 characters
    citationLendersShown  two, plus "and N more of this kind, in citationExempt"

Proximity is a guess and the line says so ("nearest this file first"), where
picking whichever sorted first stated nothing at all.

**Break-test.** Two more `.go` rows, one of them in another directory: the line
stays about the same length, both `core/` rows lead, and it ends *"and 2 more
of this kind, in citationExempt"*.

---

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + mjs suites + browser pass (rc 0)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

Five files, +1159 −71:

    wasm/verify/themenearmiss_test.go       item 1
    internal/pinfixture/pinfixture_test.go  item 2
    wasm/verify/browser.mjs                 items 3, 4
    wasm/verify/inkglyph_test.go            item 5
    wasm/verify/checknumbering_test.go      item 6

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The scale is one number over four endings, which is
   the shape complaint one level up again.** `affordedScale` divides this run's
   whole population by the recorded one and every ending is held against that
   single ratio. It is right when the walk changes — a window nudged, leaves
   added — because then all four really do move together. It is wrong in the
   one case that matters: a change to `themeLeafSetOf` that reshapes WHICH
   ending a set reaches moves the four against each other while the population
   is identical and the scale is 1.00. The prediction is then "this ending
   should be where it was", which is the constant argument wearing a
   measurement's clothes, and `affordedShortfallCause` would report a
   redistribution as an arm going.
2. **(age 0 · value medium) The straddle can only be found where a construct
   ends at a newline.** `pinBlankedAs` reads every construct in a span, and the
   only straddles the fixtures can produce are the ones a line comment or a
   single-line string leaves at the line end — everything else needs a
   delimiter in the raw file and a delimiter breaks the phrase match. So the
   multi-construct clause is exercised by exactly one transition shape, and the
   two-and-three-construct paths past it are held by fixtures that all begin
   with `//`. A block comment closing mid-line into a literal is unreachable
   today and nothing says so; the machinery is general and its evidence is not.
3. **(age 0 · value low) The agreement is asked only where the probe named one
   face.** `inkCanaryAgreement` skips a probe whose `faces` has two entries —
   a generic that reached more than one face for "Ag0" — on the argument that
   there is no single name to compare. That is true and it is also the
   population where a name-versus-metric disagreement is most likely, dropped
   without a count. `asked.canvasGenericPaired` says how many runs were joined
   and nothing says how many probes were passed over, so the comparison going
   quiet on this machine would read as 28 agreements out of 28.
4. **(age 0 · value low) The gap census is a fact about node's ICU.** The 15054
   is NFKD as the Node on this machine implements it, compared against a table
   somebody wrote down — the same "measured against ONE build" shape
   INK_OWN_MEASURED_ON records and this does not. A different Node moves every
   number in that log line and the printable-ASCII edge is the only thing
   asserted, so a run whose census had halved would print the new number and
   say nothing. There is no `foldMeasuredOn`.
5. **(age 0 · value low) Two of the six probe reads are now doing nothing.**
   The spread says the generic answers with one face at every request on this
   grid, which is the reading item 3 wanted — and it is also the evidence that
   five of the six round trips bought a sentence. The honest next move is a
   bound: read one, and read a second only where the census says the answers
   could differ. Nothing here knows what that condition is, which is why this
   is worth a measurement rather than an edit.
6. **(age 0 · value low) Proximity is a guess with no reading of how often it
   is right.** `citationLendersNear` puts same-directory rows first, and on
   this repository the example is a `core/` file and both `core/` exemptions
   lead — which looks like the heuristic working and is equally consistent with
   there being two rows and one directory. Nothing counts how many kinds have
   lenders in more than one directory, so the ordering has no population under
   it and the line's "nearest this file first" is a claim about a sort, not
   about a match.
