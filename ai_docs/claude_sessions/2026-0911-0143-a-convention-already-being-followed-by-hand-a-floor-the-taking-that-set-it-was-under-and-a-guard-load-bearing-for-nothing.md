# Session: a convention already being followed by hand, a floor the taking that set it was under, and a guard load-bearing for nothing

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · iteration 1 of a ten-iteration `/loop`
Previous: `2026-0911-0117-the-loop-nine-iterations.md`

## Ask

`/loop 10`. Working the Next list, top item first.

**Item 1 (age 1, value medium):** there is no way to mark a figure as a
quotation of a past reading. Two declined rules failed on the same gap, and
`renamedTestsStillNamed` is a table standing in for it.

---

## What was built

`wasm/verify/quotedprose_test.go` — the convention, stated: **a token in
backquotes is quoted, not claimed.** Two helpers, `quotedRuns` and
`quotedAt`, plus `unwrapRawLiteral`. The test-name rule reads it; the
graveyard table is gone.

The convention was not invented. It is the one `coresAttribution` already
states for the terms it names and `trackedGoFileFigure` restates for a count,
extended to the two shapes a rule actually reads. **It was also already being
followed by hand**, which is the finding that decided the spelling:

    91 duration figures written as a range, in Go prose, 4 of them backquoted
    3 of those 4   quotations of a past reading

Three of the four figures anybody had thought to backquote were exactly the
class the convention names. Nobody was told to do that.

## What it cost, which is the number that decided it

An escape is only worth having if it is narrow, and "backquoted" is a
typographic habit as much as a statement. So it was counted before it was
written:

    356 Test-shaped mentions in Go prose
    13 backquoted
    12 of the 13   in prosenames_test.go, whose mentions were already exempt
    1 of the 13    `func TestMain(` in a tutorial code block

Outside the file that was already exempt, the escape lets through exactly one
mention, and that one is a code sample. It costs no coverage this repository
has.

## What the item got wrong

The item predicted the marker would make **both** rules writable. It made one
of them writable and closed a class in the other.

`renamedTestsStillNamed` is gone, as predicted. The wall-clock rule's residue
of five sentences-that-were-right is now zero — but only two of the five were
resolved by the marker. One was resolved by naming a record field, with the
marker preserving the history the fix would otherwise have deleted; two were
resolved by stating a proportion instead of the two readings it is a
proportion of.

And the rule is **still declined**, on a reason this cleared the way to see:

> **"Lives in a record" is not a span anything can define.** Readings
> legitimately live in a `…TimingsTakenOn` declaration (76 figures), in a
> measurement table in the prose beside one (7), and in the `costs:` field of
> a structure that is a record in everything but name (2).

Plus one figure that is not a reading at all — `hooks/hooks_test.go:26` says
test producers run at `5–20ms` periods, a SPECIFIED duration written as a
range, which is the discriminator's own counterexample, measured at one.

`ai_docs/plans/non_goals.md` carries this as a measurement now rather than as
the prediction it was carrying.

## What was found

**The floor was wrong when it was typed, and the session that typed it can be
seen to have been under it.** `verifyTimingsTakenOn.wholeFile` recorded
2.88–2.97s. Twelve readings this session: 2.840–2.955s, nine of them under
the floor. Nothing had been made faster — one file and one code path had been
ADDED.

The machine was ruled out the way this record always rules it out, by reading
the untouched sibling at the same moment: `internal/themehistory` came back
3.018–3.131s against a recorded 2.92–3.22s, mid-band and slightly *slow*.

The previous session's own closing figures say `2.866–2.946s, in band` — and
2.866 is not in a band that starts at 2.88. The error was a hundredth of a
second, in the record's headline figure, written and quoted in a wrap-up on
the same day, and nothing caught it, because a band is checked by a person
comparing two numbers and that is the one check this repository has no arm
for.

Widened to **2.84–2.97s over thirty-three runs in two sessions** — held rather
than replaced, the same treatment as the `-race` row one package over.

**Two stale counts of this repository's own history.** `internal/themehistory`
prose said `88` `ls-tree` processes twice; the record says 89 and the live arm
prints 89. Same treatment the other fourteen got: the count is not restated,
because it grows. A third figure, `0.87–0.92s`, was a copy of a record field
that says 0.87–0.91s — a hundredth out of step. It now names the field, and
what the line used to say survives in backquotes.

**The last by-hand figure copy.** `verifyTimingsTakenOn`'s re-taking list
named `repowalks_test.go`'s header as "the only one left" that a re-taking
moved by hand, with the reason "prose cannot read a field". That reason is
about mechanism and the sentence needs a *proportion*, which does not drift —
the resolution `internal/themehistory/main.go` had already reached for the
same shape. The list entry now reads "nothing to move any more."

**One asymmetry.** The `func Test…(` sample skip existed in the text path and
not the Go path, so a code sample in a raw string literal was a mention that
resolved only by prefix accident. Both paths have it now.

**One guard load-bearing for nothing, and it stays.** Removing
`unwrapRawLiteral` changes no count — 387 mentions either way, because the one
Test-name in a raw literal is excluded by the sample skip anyway. It stays,
and its doc says so, because its absence is not a missed finding but an
INVISIBLE one: a raw literal would read as one long quotation with nothing in
any message saying so. Same category as `checkProseNamesResolve`'s empty-list
`Fatalf`, which has also never fired. **A rule this repository declines is one
that produces no findings; a mechanism that stops a kept rule lying is a
different thing and is not held to the same count.**

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Five break-tests, every one read rather than counted:**

    unbackquote a dead name          fires, new message read in full
    remove the quotedAt escape       4 findings — the escape is load-bearing
                                     for exactly the corpus
    remove unwrapRawLiteral          no change. Recorded as such
    a stray backquote before a       still CHECKED, not skipped — the safe
    marked name                      direction quotedRuns documents
    the file-count arm               fired on the commit that adds this file,
                                     as it has on every such commit

Figures at the end:

    wasm/verify             2.840–2.955s   recorded 2.84–2.97s (re-taken)
    internal/themehistory   3.018–3.131s   recorded 2.92–3.22s

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, measured from this doc.

1. **(age 0 · value medium) A band is the one check this repository has no arm
   for, and it just cost a wrong headline figure.** The floor above was wrong
   the day it was written and survived a wrap-up that quoted a number outside
   it. A wall clock cannot be asserted — that argument holds — but the
   comparison of a reported range against a recorded one is arithmetic, and
   the wrap-up is where it is done by eye. The cheap form is not an arm over
   prose: it is that the reporting test PRINTS whether its own reading fell in
   its own band, so a person reading a green run sees the word rather than two
   numbers. Both records already print the reading.
2. **(age 0 · value low) `…TimingsTakenOn` is not the only record shape.**
   The wall-clock rule is now blocked on exactly this: `repowalks_test.go`'s
   walk census carries a `costs:` field with its own taking method, and the
   GOMAXPROCS table is a set of readings in prose. If those became records
   proper, "inside a declaration" is a span again and the rule is one
   function. That is a bigger change than the rule is worth on its own, and
   worth doing if either structure is being edited anyway.
3. **(age 12 · value low) Every Next list in this loop was written by the
   session that would not work it.** Ten iterations of evidence now. This
   one's item predicted a marker would make two rules writable; it made one
   writable and re-diagnosed the other. The instruction ("measure before
   writing") paid again; the conclusion was again half wrong.
4. **(declined, non-goal) A wall clock in prose is not held to living in a
   record.** See `ai_docs/plans/non_goals.md` — re-measured this session
   against the convention, and declined on a new and sharper reason.
5. **(declined, non-goal) A backquoted name in prose is not held to being a
   declaration.** See `ai_docs/plans/non_goals.md`, which now records that
   this session gave backquotes the opposite meaning and that the two findings
   are the same one from two sides.
6. **(declined, non-goal) A `t.Run` inside a loop counts as one question.**
7. **(declined, non-goal) `packageLevelCallsTo` cannot tell an initializer
   from a stored function value.**
8. **(declined, non-goal) The diff-and-print remainder stays unmeasured.**
