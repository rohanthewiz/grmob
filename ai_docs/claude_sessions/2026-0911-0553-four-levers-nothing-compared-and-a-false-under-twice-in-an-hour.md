# Session: four levers nothing compared, a list named for its first member, and a false UNDER twice in one hour

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · Previous:
`2026-0911-0541-two-floors-found-by-an-arm-in-the-hour-it-existed.md`

## Ask

`/loop` — work the Next list. **Iteration 4**, working **Next item 1**: four
`GRMOB_` levers duplicated across the two packages with nothing holding them in
step.

## What was measured first

A scan of every package-level declaration both `package main` programs declare,
by kind, with whether the rendered text matches:

    21 shared names
    16 identical   of which 12 were registered and 4 were not:
                   bandVerdictEnv, bandVerdictAsked, packageReadingEnv (const)
                   and bandVerdictWanted (func, registrable all along)
    5 differing    both records' own arms, coresAttribution, main, and
                   recordMachineDiffers — each reads its own record, and each
                   states why

So the item's count was right, and the scan also says the other five
differences are all accounted for.

## What was built

    const is now a shape    twoCopyDeclarationsIn read `token.VAR` only, so
    the census can hold     registering a constant reported it MISSING from
                            both packages — a list entry that fails instead of
                            holding. That is why three levers sat unregistered
                            while the argument for registering them had
                            already been written twice, about `var`s
    declarationText renders the keyword the declaration actually used, so a
                            `const` in one package and a `var` of the same
                            name and value in the other is a difference this
                            REPORTS rather than one it renders away
    twoCopyStateShapes →    the list was named for its first and only entry.
    twoCopyValueShapes      `dotImportsReported` is state; the band pattern is
                            a var and is not; three levers are consts and are
                            neither. What unifies them is how the walk finds
                            them — a ValueSpec inside a GenDecl — which is the
                            argument the list's own doc already made

The census now holds **16 shapes — 11 functions and 5 value declarations** —
which is exactly the identical population the scan found. The registered set is
no longer a selection.

## What the levers were risking

Both records' doc comments print `GRMOB_BAND_VERDICT=required go test …` and
each package read it through a constant of its own. Nothing compared the two
strings. A `GRMOB_BANDVERDICT` in one of them would compile, pass every test,
and leave a reader following the documented command with a verdict from one
package and **silence** from the other — and silence here is indistinguishable
from a pass, on the records that have now had four floors found stale between
them.

## What it found: a false UNDER, from this machinery, twice in one hour

Taking the closing figures, both packages reported UNDER:

    wasm/verify        2.755s against a floor of 2.76s
    internal/themehistory  2.816s against a floor of 2.82s

**Both verdicts were wrong.** An end is a reading rounded outward to the
record's two decimals — the record says so — so a floor of `2.76s` stands for
readings down to 2.755s. `bandPlacement` already judged "does this reading reach
an end" that way; the in-band decision compared raw clocks. The two halves of
one sentence disagreed: a reading could be reported outside a band and, by the
placement rule printed in the same line, be AT its floor.

Both copies now switch on `got.Round(step)`. The distance printed is still the
true one — the rounding decides which branch, not what to say once the branch is
chosen. And the "at the floor" and "at the ceiling" branches, which no wall
clock had ever hit, now fire on real readings.

This was found by running the figures, not by a test. The arm built one
iteration earlier is what made it visible: before that, nobody compared these
two numbers at all.

## Verification

Eleven paths, all green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Break-tests: six, every one read.** Three against the const support — a
lever's value changed in one package (reported), a `const` turned into a `var`
(reported, which is what the keyword rendering is for), and a lever renamed in
one package (reported as 15 of 16 shapes). Three against the rounding — half a
step under the floor (in band, at the floor), a full step under (UNDER), and
half a step over the ceiling (in band, at the ceiling).

One of those break-tests caught my own grep rather than the code: break 3 looked
like a silent pass because the pattern I filtered with did not match the message
the census prints. Read the output, not the filter.

**Figures, each package alone, through the placement arm:**

    wasm/verify wholeFile      2.786–2.888s, 12–60% up a 210ms band
    internal/themehistory      2.837–2.911s, 4–22% up a 400ms band
    wholePackage

All in band, and no widening this iteration — the two floors widened an hour ago
now hold.

Four files, +160 −32.

## Next

1. **(age 1 · value medium) Invert the two-copy list: find the shared
   declarations instead of registering them.** The measurement this iteration
   made is the argument. **4 of the 21 shared names were identical and
   unregistered**, and three of them had been for several sessions with the
   reason already written down twice — the list's failure mode is silence, and
   the only thing that found them was a session happening to scan. The
   inversion holds all 16 automatically and needs an exemption list of 5, each
   of which already has its reason in prose. The cost is naming the two
   directories (today `twoCopyPackages` is a count, not a list), collecting
   every package-level declaration in them rather than only the registered
   names, and rendering on demand — measured shape of the corpus: 21 shared
   names out of two packages' worth of declarations, so the comparison is small
   even if the collection is not.
2. **(age 12 · value low) `…TimingsTakenOn` is not the only record shape.**
   Unchanged: `affordedMeasuredOn` and `foldMeasuredOn` are re-derived by the
   run that reads them.
3. **(age 25 · value low) Every Next list in this loop was written by the
   session that would not work it.** Twenty-three iterations. This item was the
   most accurate one yet — it named the four declarations, the reason the walk
   could not hold them, and the fix — and it was still incomplete in the same
   way as the others: it did not predict the rename, and the defect the
   iteration actually spent its last hour on (a false UNDER from rounding) was
   not on any list. It was found by taking the figures, which is the one step
   no item has ever had to name.
4. **(declined, non-goal)** Thirteen entries. See `ai_docs/plans/non_goals.md`.
