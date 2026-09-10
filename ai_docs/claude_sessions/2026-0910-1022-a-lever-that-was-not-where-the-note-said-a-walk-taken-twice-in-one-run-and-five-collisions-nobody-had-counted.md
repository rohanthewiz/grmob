# Session: a lever that was not where the note said, a walk taken twice in one run, and five collisions nobody had counted

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-census-that-corrects-instead-of-walking-a-bound-that-turned-out-to-be-one-and-a-paste-nobody-had-read")

## Ask

"Do all items in the Next list." Six items, all age 0 — one medium, five low.
Two files, no new ones. Three of the six were raised with a hypothesis attached
and two of those hypotheses were wrong, which is most of what this session
found.

| item | where | shape |
|---|---|---|
| 1 | themenearmiss_test.go | three leaves at twelve seconds, and where the cost actually was |
| 2 | inkglyph_test.go | 299 characters and no idea where |
| 3 | themenearmiss_test.go | a printer nobody checked the numbers of |
| 4 | themenearmiss_test.go | one population whose base census ran unheld |
| 5 | themenearmiss_test.go | 557ms to confirm a theorem |
| 6 | themenearmiss_test.go | four name-keyed tables and no census of them |

---

## Item 1 · the lever was one level below where the note put it

The item named a specific cost: the drop census keys its counts by the ending's
own sentence, so every correction is a map lookup on a fifty-character string,
and an ending INDEX would take the per-population cost from 150µs to about 80.

**It is 0.3% of the walk.** Profiled over k = 2:

    themeEditDistance + the allocation under it     most of it
    runtime.lock2 / madvise / usleep                 the allocator, eight
                                                     workers deep
    internal/runtime/maps.putSlotSmallFastStr       0.32%

The thirty windows a drop puts back are thirty calls to `themeLeafSetOf`, which
is the derivation under test and not this file's to make cheaper. The index is
in anyway — it costs nothing and it is what is left once the classifications go
— but it was never the lever.

### What was

The same move `affordedWhole` made, one level further down. A window of kept
indices `{a, b, c}` is put back by EVERY population that drops something
between a and c and keeps all three:

    k = 2   3160 populations × ~30 windows = 190 thousand classifications
            over 3665 distinct index lists
    k = 3   82160 populations, 7.4 million classifications, 8705 lists

The shapes are a function of the POSITIONS and the window bound — the same fact
that lets a fourteen-name family stand as evidence for an eighty-name one — so
they are enumerated before the walk starts and the table is read-only while the
workers run, with no lock between them. A window it does not hold is classified
directly and **counted**, so an enumeration that stopped covering produces the
same numbers more slowly rather than wrong ones; the count is carried out
through the walk's own report and asserted at zero.

                  direct walk    corrected    and memoised
    k = 1                80ms          8ms             3ms
    k = 2               5.3s         463ms            38ms
    k = 3            ~2 hours        12.4s           112ms
    k = 4                        ~4 minutes         685ms
    k = 5                                             8.4s

### So three leaves is a band and not a sentence

112ms buys the three-field edit the same footing every other step has: an
assertion against a family this run's population is a MEMBER of. The measured
band says `band × k` is not settling down —

    its own crowding stopped the search   gaining   224.44%   2.55× the
                                                              two-leaf figure
                                                              6.75× the one-leaf

— so three times the one-leaf band is 99.8% against a measured 224%, and a
scaled bound would call an honest three-field edit a re-sort by a factor of two.

**k = 4 is left off, and that is a judgement with a number under it.** 685ms is
six times what the rest of the test costs and what it buys is the fourth
simultaneous field edit. Five is where the wall is now.

### And the four arms that became one loop

The residual arms were four blocks — long and short, at one leaf and at two —
with the same eight lines of arithmetic and four sets of prose, the second pair
written by copying the first. What varies is the band, the population it was
measured over, and the names that moved. They are one loop over
`affordedBandSteps` now, so **k = 3 arrived with no new arms written for it at
all** — and the log line's band selection went with them, which had a bug worth
naming: the two-leaf LONG case fell through to the one-leaf default, so the arm
asserted against a band the line did not print.

**Break-tests — 6 run, 6 fired.** A gap width dropped from the enumeration
(caught by the miss count). The memo storing the wrong mounting. A span key
truncated so two windows collide. The ending index off by one. This run's base
census taken over the wrong names. And the arms themselves, exercised at k = 1,
2 and 3 by patching the record to a consistent smaller population — which is
what turned up `"SM and XL and XS"`, a join on `" and "` that read correctly
only because the widest step had ever been two.

---

## Item 5 · the walk that was taken twice in one run

The item asked whether a cheaper version of the 557ms sound bound exists and
guessed at a prune: "most of them cannot be the maximum for reasons the one-leaf
band already knows." The cheap bound is far too loose to prune with — the
windows a drop can add are about 60 sets, which is 12% of the large ending and
220% of the small one, both above the maxima being looked for.

The cheaper version was not an algorithm. `affordedSoundTwoStepBound` censused
all 3160 two-drops of the record's names, and `affordedKLeafBand(k=2)` censused
**the same 3160 populations in the same run** — the first reading each against
the whole, the second against the two populations of n−1 it can be reached from.
Same populations, same censuses, one after the other.

`affordedTwoStepBands` takes the census once and reads it three ways: the
measured two-leaf band, the widest second step of any chain, and the sound
composition. What the domination check holds is unchanged, because it was never
holding that the population had been walked twice. The pairs are enumerated i<j,
which is the order `affordedKLeafBand` builds its combinations in at k = 2, so
the measured band names the same widest drop it did.

**Break-tests — 2 run, 2 fired.** The two-leaf band reading against the
second-step scale. A sound bound made to fail the domination it guarantees.

---

## Item 6 · five collisions, in a table the note called an approximation

The item said the four name-keyed tables conflate anything sharing a name and
that "nothing collides today." **Five names do**, and they always did:

    field "chain"      map[string]affordedBand in one struct, float in another
    field "gaining"    the walk's [4]best, and affordedBand's number
    field "losing"     the same
    field "got"        int in one, float in another
    field "measured"   a bracket's count as a float, a comparison's as an int

So the reading a comment described as a limit is a limit in force, on names this
file uses everywhere.

### And the two tables fail differently, which is why it is two arms

    field    a UNION — a name declared float in ANY struct marks every selector
             with it. Order-independent, and always in the direction this census
             calls safe: an entry somebody has to write rather than arithmetic
             nobody sees. Counted, and the count is in the log line.
    method   an ASSIGNMENT — the surviving answer is whichever declaration was
    import   parsed last, and the order is the directory listing. One of the two
             orders makes real float arithmetic invisible, which is the failure
             this whole test exists for. That one fails.

Reported and not resolved: telling two same-named declarations apart is what
go/types is for, and running it over a package that imports half this repository
is the cost this census was written to avoid.

The census also caught the arithmetic this session added — `float64(slot) /
10000`, in item 3's transport check — on its first run, which is the reading
doing its job rather than a finding.

**Break-tests — 2 run, 2 fired.** A method returning a float on one receiver and
an int on another (fails). A new float field colliding with an int one (counted,
6 names not 5, does not fail).

---

## Item 3 · a printer handed numbers nobody read back

The paste test compared KEYS in both directions and nothing else. Both of the
item's suggestions were about measuring: walk the bands in that test (two
seconds for a printer) or check a failing run's own paste (which only ever runs
on a failing run).

Neither is needed, because what is under test is **transport**. The printer
takes the numbers as arguments, so it is handed a set in which every slot is
different — one value per step, per ending, per direction, and a separate range
for the chain and the sound composition, each a distinct four-decimal value at
the precision the printer writes. A printer that transports faithfully, given
this run's measurements by the failing run, writes this run's measurements; and
any mix-up between two slots is two values that are not equal here, where the
recorded bands (four endings at 0.0004 apiece on one of them) would have hidden
it.

The census record's own four counts are checked too — the names were held and
the numbers beside them were not, so a printer that wrote the endings sorted and
the counts in walk order would have produced a paste that compiles, fits the
declaration, carries the right names and records the population under the wrong
sentences.

**Break-tests — 4 run, 4 fired.** Losing printed where gaining goes. The chain
map written under `sound`. Every step writing step 1's band. A census count off
by one.

---

## Item 4 · the base census this run measures its own bands from

Every band a run measures over its own population is a correction of
`affordedWholeOf(names)`'s census, and that census was compared with nothing.
The record's was, twice. A run that has edited core.Theme measures its bands
over a population whose base walk had no second reading at all.

It costs nothing: the test has already walked those sets into `reached`. Read
into an array rather than compared as maps, because an ending no set reached is
absent from one shape and a 0 in the other — the corner `affordedCensus` exists
to remove, and comparing the two shapes directly would have put it back.

**Break-test — 1 run, 1 fired.**

---

## Item 2 · where the BMP's 299 are

96 runs making **8 regions**, and they are not the astral three made larger:

    U+00CC..U+02E2   the accented Latin letters and the Latin extensions
    U+1D2E..U+1ECB   the phonetic modifiers
    U+2071..U+217C   the superscripts and Roman numerals
    U+249D..U+24E3   the circled and parenthesised letters
    U+2C7C           one subscript in Latin Extended-C
    U+3250..U+33FF   the enclosed CJK squares
    U+A7F3           one modifier in Latin Extended-D
    U+FF22..U+FF54   the fullwidth forms

Above the BMP the whole population is three blocks of decorated alphabets; here
NFKD reaches a seed's own letters from every direction the plane has. The runs
were already being computed — the walk is written once and both sides go through
it — and thrown away.

`foldBearingGap` is not choosing the answer on either side, and the margins
differ: 67 inside and 1816 between above the BMP, **162 and 422** on it, so
anything from 163 to 421 gives the same eight. Both gaps are printed for both
planes and both region counts are recorded, so a constant that started choosing
would show up as a count that moved rather than as nothing.

**Break-tests — 2 run, 2 fired.** The region count taken from the wrong plane.
The runs never joined, so the population reads as scattered.

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

Two files, +1360 −547.

    wasm/verify/themenearmiss_test.go  items 1, 3, 4, 5, 6
    wasm/verify/inkglyph_test.go       item 2

`go test ./wasm/verify` is **1.84s against 2.9s**, while adding a step that used
to cost 12.4 seconds. 29 float derivations, 14 of them carrying a comparison.
15 break-tests run; every one fired.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The ending index and the sentence list are held
   together by position and nothing else.** `affordedEndingAt` returns 0, 1, 2,
   3 as literals and `affordedEndingNames` lists the four sentences in that
   order; a reordering of the array silently re-labels every census, band and
   record in the file, and every arm goes on passing because they all read the
   same wrong map. This is a coupling this session INTRODUCED — before it, the
   switch returned the sentence and the order of the test's list was harmless.
   The arm that would hold it is small: build one `themeLeafSet` for each of the
   four branches, ask `affordedEndingOf` for it, and assert the sentence. Two of
   the four are constructible by hand (`cappedByReach` false, `affordedOpen`
   true) and the other two need a set walked out of the population, which the
   test already has.
2. **(age 0 · value low) The one-leaf band over the record's names is measured
   three times a run.** The steps loop takes it, `affordedTwoStepBands` takes it
   again for the composition's b₁, and `affordedChainBoundOf`'s first step takes
   it a third time over the same names — plus an `affordedWholeOf` of the
   record's population inside each. About 15ms, which is why nobody noticed
   while the walk was 2.9 seconds and is a tenth of what the three-leaf step
   costs now that it is 1.8. The band is a pure function of a list of names, so
   the fix is a cache keyed by the list identity rather than any change to what
   is walked.
3. **(age 0 · value low) Five field names are conflated and the count is only
   watched.** The arm reports and does not resolve, which is right for the
   general case, but two of the five are this file's own and are renameable:
   `losing` and `gaining` are `affordedBand`'s two numbers AND the k-leaf walk's
   two arrays of candidate maxima. Renaming the walk's pair takes the census
   from five conflations to three at no cost, and what is left is `chain`, `got`
   and `measured`, which are collisions between unrelated structs and are the
   honest residue.
4. **(age 0 · value low) k = 4 is 685ms and left off on a judgement.** The
   number is written down beside the decision, which is what this file asks of a
   judgement, and the decision could still be wrong: what it buys is the fourth
   simultaneous field edit, and nobody has asked how often core.Theme actually
   moves four leaves at once. The git history has the answer and nothing has
   read it.
5. **(age 0 · value low) The BMP's eight regions are asserted as ranges and
   described in prose.** The eight `U+xxxx..U+xxxx` pairs move with the Unicode
   version and are held by the region count; the sentence naming them — "the
   accented Latin letters, the phonetic modifiers, the superscripts and Roman
   numerals" — is a reading of those ranges that nobody re-derives, and it is
   the half a reader actually uses. Go's `unicode` tables carry block ranges,
   so the block a region falls in is derivable rather than typed.
6. **(age 0 · deliberate non-goal) A sound chain bound at k = 3.**
   `affordedTwoStepBands` exists only for two, and there is no reason to write
   the three-step version: bounding the last step over every chain needs a
   census of every population of size n−k, which is the family the direct
   measurement walks — and the direct measurement is now the cheaper of the two
   at every k this file takes. The composition is kept at k = 2 for the SHAPE it
   holds, not as a route to anything. Listed so it is visibly declined rather
   than quietly missing.
