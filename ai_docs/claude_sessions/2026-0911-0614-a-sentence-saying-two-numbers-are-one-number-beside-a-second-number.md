# Session: a sentence saying two numbers are one number, beside a second number

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · Previous:
`2026-0911-0606-the-question-asked-the-other-way-round-and-main-which-every-package-has.md`

## Ask

`/loop` — work the Next list. **Iteration 6**, working **Next item 1**: three
constants that are all `2` about the same pair of directories, with nothing
holding them to agreeing.

## What was measured

    27 integer constants across the two record packages
    10 of them literals whose doc comment names another constant
    1  of those 10 was actually a copy

The one is `twoCopyPackages`. It was `2`, and its own doc said *"it is the same
constant as `timingsRecordCopies`"* — a sentence claiming two numbers are one
number, sitting beside a second number. `coresNoteScanDirs` had got this right
two files over, is defined as `timingsRecordCopies`, and says why: "a bare 2 here
would be a second number to keep in step by hand."

The other nine name a constant to point at a **pattern**, not a value.
`gitWrapperAcceptRules` says `timingsRecordCopies` "is the same shape of constant
for the same kind of reason" — about acceptance rules in a git-wrapper census,
and `2` by coincidence. Eight of the nine have a different value from the
constant they name.

## What was built

    twoCopyPackages = timingsRecordCopies   one line, and the argument for the
                                            separate NAME kept: it is where the
                                            second answer goes if somebody
                                            decides the two should part

    the copy count now fires BOTH ways      it only asked whether there were too
                                            MANY packages declaring the set.
                                            Too FEW is the copies quietly
                                            becoming one — every shape still
                                            found, every comparison still run,
                                            and a comparison of one declaration
                                            against nothing passes. Reachable
                                            without a compile error: the
                                            helpers are used by the censuses in
                                            the file that declares them, so
                                            deleting the file and its callers
                                            together builds fine and halves the
                                            copies

## What the item got wrong, twice

**"One list of the two directories" is worse than what is there.** The subject is
already derived — iteration 5 made it *the packages carrying a timings record*,
taken from the walk. A hardcoded list would be a fourth statement of the same
fact and the only one that is a copy of something computed on every run. The
counts exist to force a decision when the pair grows; what NAMES the pair is the
source. Written down where the derivation is.

**"Let the trigger name the packages rather than count them" was already done.**
Both trigger messages already join the directories they found.

So the item's one correct half was the part it stated as a measurement — the
three constants agree and nothing holds them — and even that was two-thirds
wrong: `coresNoteScanDirs` was already derived and argued. Only one of the three
was unheld.

## What was declined, with numbers

One entry added to `ai_docs/plans/non_goals.md`, which now holds **fourteen**:

    a constant whose doc names another     1 defect in 10 candidates, and the
    constant is not held to being          distinction a rule would need — "is
    defined as it                          the same constant as X" against "is
                                           the same shape of constant as X" —
                                           is reading English, which is what
                                           the cores-note census learned not to
                                           do. Eight of the nine innocents have
                                           a different value, so a value
                                           comparison would not separate them
                                           either

## Verification

Eleven paths, all green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Break-tests: three, every one read.** One package dropping the whole set
(reported: "1 package(s) declare the whole set of 16 and the copy argument is
written for 2"); the derived constant forced to 3 (the downward arm fires, which
is what says the derivation is load-bearing); and a third package declaring all
sixteen shapes, copied verbatim so the texts match (the upward arm fires as
before).

**And a process failure worth recording.** Restoring a break-test edit from a
scratch copy taken two iterations earlier silently reverted iteration 4's
rounding fix — 39 lines, including the comment explaining the false UNDER. Caught
by `git status` showing a file this iteration had no business touching, and
restored from the commit. Last iteration's lesson was "don't revert with `git
checkout` when the file has uncommitted work"; this iteration's is the other
half: **a scratch copy is only valid for the edit it was taken for.** Take it in
the same command as the break.

**Figures, each package alone, through the placement arm:**

    wasm/verify wholeFile      2.758–2.821s, all in band
    internal/themehistory      2.863–2.925s, in band
    wholePackage

The 2.758s reading is the iteration-4 rounding fix earning its place: it rounds
to 2.76, which is the floor, and a raw comparison would have called it UNDER.

Three files, +86 −5.

## Next

1. **(age 14 · value low) `…TimingsTakenOn` is not the only record shape.**
   `affordedMeasuredOn` and `foldMeasuredOn` carry bands too, and both are
   re-derived by the run that reads them, so a number that moved is a finding
   about the data rather than a stale record. This is now the oldest live item
   and its value has not moved in six iterations; the next session that reaches
   it should consider measuring it once and moving it to `non_goals.md` rather
   than carrying it further.
2. **(age 1 · value low) The band machinery is now eight declarations across two
   packages and its own prose is the only map of it.** `recordedBand`,
   `recordedBandForm`, `bandPlacement`, `againstBandGiven`, `againstBand`,
   `recordMachineDiffers`, the three levers and two arms — held identical where
   they should be, and findable only by reading. Measured this iteration while
   break-testing: renaming eight of them by hand to simulate a dropped copy took
   one regexp pass and produced a correct finding, so the census covers the
   identity question; what is missing is nothing mechanical, only a reader's
   entry point. Cheapest honest form is probably one paragraph in
   `band_test.go`'s header naming the set and what each member is for — which is
   prose, and therefore a thing that goes stale, so weigh it against leaving the
   census to be the map.
3. **(age 27 · value low) Every Next list in this loop was written by the
   session that would not work it.** Twenty-five iterations. This item is the
   first whose *measurement* was wrong rather than its conclusion — it said
   three constants were unheld and two of them were already right. Measuring
   before building caught it in the first five minutes, which is the instruction
   that has now paid in every iteration of two runs.
4. **(declined, non-goal)** Fourteen entries. See `ai_docs/plans/non_goals.md`.
