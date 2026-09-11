# Session: the question asked the other way round, and `main`, which every package has

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · Previous:
`2026-0911-0553-four-levers-nothing-compared-and-a-false-under-twice-in-an-hour.md`

## Ask

`/loop` — work the Next list. **Iteration 5**, working **Next item 1**: invert
the two-copy list — find the shared declarations instead of registering them.

## What was built

`checkSharedNamesAreAccountedFor`, riding the existing two-copy subtest:

    every name both record packages declare is either in a two-copy shape list
    (held identical) or in twoCopyNamesThatDiffer (with the reason it differs)

Three more directions, each a way the arrangement rots:

    an exemption for a name that is not shared   the pair it describes is gone
    an exemption whose two declarations are      the reason has stopped being
    identical                                   true, and the shape is a copy
                                                nothing holds
    a name in a shape list AND the exemption     a contradiction: the walk
    list                                        resolves it as held, and which
                                                was meant is not readable

Supporting: `namesDeclaredBy` (one declaration's names — renamed from my first
choice, which collided with this package's existing `packageLevelNames`), the
walk collecting `name → dirs` for every package-level declaration, and the
subject derived rather than written down — the two-copy packages are *the
packages carrying a timings record*, taken from the walk so a record moving to a
third package is a finding there rather than a silent change of subject here.

## What the item got wrong, and what the measurements decided

**The item said "invert the list" — the list cannot be inverted.** Only an
inclusion list can say that the whole SET is present in both packages, which is
the check that catches a copy losing its twin. The inversion answers a different
question — is every shared name accounted for at all — so it is an addition, not
a replacement. Both lists stay.

**Two measurements chose the design:**

    rendering all 690 package-level declarations in the two packages costs
    ~19ms on top of a ~13ms parse — so the inversion collects NAMES only, and
    the text comparison stays where it already was

    comparing a declaration's own source text (Pos to End, which excludes the
    doc comment) against what go/printer renders: the two readings agree on all
    21 shared names, 0 disagreements. So source text was available as a cheaper
    reading — and it was not taken, because taking it would put TWO readings of
    "are these the same declaration" in one file, which is the shape this
    census exists to find

## What it cost to get right: `main`

The first working version collected the exempted names through the same sets as
the held ones. Every `package main` in the repository declares `main`, which is
an exempted name — so the arm that holds *a package declaring some of the
two-copy shapes to declaring all of them* reported **every package in the
tree**: android/verify, aria/gen, examples, examples/chat, and on.

The fix is a flag on the collected declaration (`excused`) and two separate
memberships: what the lists say must be identical, and what this walk collects.
Conflating those made every exempted name read as a held one. It also exposed a
latent defect in the older arm — its numerator was `len(byDir[dir])`, everything
the walk collected for that directory, against a denominator of the registered
shapes. Correct only while the two sets were the same; now counted properly.

And one message was the same fact twice: a contradicting name reported both the
contradiction and "the walk collected 0 of its declarations rather than 2". The
second is an artefact of the first, and is skipped now.

## Verification

Eleven paths, all green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Break-tests: six, every one read.** A new shared name registered nowhere
(reported, naming both packages); an exemption for a name neither package has;
an exemption whose copies are identical; a name held and exempted at once (one
finding, after the fix); the name collection disabled (the over-nothing arm);
and the same contradiction re-run to confirm it now reports once.

**Cost:** the shared parse reads 0.24–0.31s over five runs with the collection
added, against 0.25–0.27s before it — map inserts against a walk that was
already parsing every file.

**Figures, each package alone, through the placement arm:**

    wasm/verify wholeFile      2.782–2.815s, 10–26% up a 210ms band
    internal/themehistory      2.862–2.918s, in a 400ms band
    wholePackage

All in band. No widening; the floors set two iterations ago hold.

Three files, +398 −8.

## Next

1. **(age 1 · value medium) The two-copy argument is written for two packages
   and the subject is now derived from the records.** `twoCopyPackages` is 2 and
   `coresNoteScanDirs` is 2 and `timingsRecordCopies` is 2, and all three are
   the same fact about the same pair of directories — but the new check takes
   its subject from *where the records are*, which is a fourth way of saying it.
   Measured this iteration: the three constants agree today and nothing holds
   them to agreeing. The useful form is probably one list of the two
   directories that the three counts are read off, which would also let the
   existing "a third package declares the set" trigger name the packages rather
   than count them.
2. **(age 13 · value low) `…TimingsTakenOn` is not the only record shape.**
   Unchanged.
3. **(age 26 · value low) Every Next list in this loop was written by the
   session that would not work it.** Twenty-four iterations. This item was
   wrong in a new way — not incomplete but *inverted*: it proposed replacing a
   list that cannot be replaced, and the iteration's work was to find out why
   and add rather than substitute. Its instruction ("the measurement is the
   argument") paid, as every instruction has.
4. **(declined, non-goal)** Thirteen entries. See `ai_docs/plans/non_goals.md`.
