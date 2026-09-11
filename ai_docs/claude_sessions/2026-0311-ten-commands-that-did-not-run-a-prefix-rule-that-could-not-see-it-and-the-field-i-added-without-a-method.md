# Session: ten commands that did not run, a prefix rule that could not see it, and the field I added without a method

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · iteration 9 of a ten-iteration `/loop`
Previous: `2026-0911-0303-a-lever-put-where-it-is-read…`

## Ask

Next item 1 (age 0, value medium): nothing holds the two "Re-taking it"
tables to being runnable. The item predicted the cheap version might be
"confirming those tables are already covered by the prose-names rule".

They were not covered, and the reason is the interesting part.

---

## Five of the ten commands did not run

    internal/themehistory   three lines wrote `./internal/…`
    wasm/verify             the three walk depths were FRAGMENTS — no
                            `go test`, no package path, and two of the three
                            test names cut off at an ellipsis

All shortened to keep a column aligned. There is no line-width census in
this repository and comment lines of 195 characters exist elsewhere in it,
so the abbreviation bought tidiness and cost runnability.

The record already names this fault in its own history. `wholeRun`'s field
doc says a figure that could only be re-derived by a recipe was "a recipe
rather than a command, and it made this the only figure in either record a
person could not re-derive by running something". A table of recipes is the
same defect one level up.

Both tables are now one command per field, written out. **All ten were run.**

## Why the rule that reads those tables could not catch it

The prose-names rule holds every `Test`-shaped name in prose to being a test
this repository has, and it reads these tables on every run. It passed on
`TestEveryGitListingInAScriptAsksForNul…`.

Because the regexp stops at the ellipsis, leaving `…AsksForNul`, which is a
**prefix** of the real test — and a prefix resolves *by design*. That rule's
own header argues for it: `go test -run X` runs everything X begins, so a
prefix still takes a reader where they are going. It is right, and it means
an elided command is invisible to it.

A narrow check does exist for half the fault — a tab-indented comment line
containing `go test` and a typographic `…`, one shape with one meaning since
`./...` is three ASCII dots and Go's own wildcard. **Measured: 2 lines, both
real.** Both now fixed, so it would score zero, and it misses the three
fragments that had no ellipsis at all. Declined in
`ai_docs/plans/non_goals.md`, with "a third instance" as what would change it.

## The field I added without a method

Checking every band field against its table turned up one gap:

    nine of ten band fields   have a taking command
    wholeFileInProcess        does not

Which is the field **this run added six iterations ago**. A field arrives
with a band and the method it was taken by stays in the head of whoever took
it. Now written out — it is the verdict command read for its reading rather
than its verdict, since the line says what the run took before it says where
that fell.

That check is a candidate arm and a good-looking one: the fields are AST, the
labels are lines matching `^\t<name>$` in the record's doc comment, one shape
and one meaning, and `checkTimingsRecordCopies` already walks both records
off the shared parse so it would need no second copy. **1 real defect in a
corpus of 10 on its first run, and it would have caught this session's own
omission in the session that made it.** Not built here — iteration 9 is not
the place to start an arm that needs a budget raise, a cores-note entry and a
census registration — and it is the top item with its numbers.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Plus every command in both tables, run as written:** wholePackage,
batchRetire, wholeRun, perObjectRun, wholeFile, wholeFileInProcess, foldWalk,
walkEnumerate, walkRead, walkParse — ten of ten.

Figures at the end:

    wasm/verify             2.859–2.929s   recorded 2.78–2.97s
    …wholeFileInProcess     2.694s         recorded 2.55–2.78s
    internal/themehistory   3.044–3.118s   recorded 2.86–3.22s

## Next

1. **(age 0 · value medium) Every band field should be held to having a
   taking command.** Measured at 1 real of 10 on the first run, on the field
   this run itself added. The arm reads fields off the AST and labels off the
   record's doc comment; `checkTimingsRecordCopies` already visits both
   records on the shared parse, so it needs no second copy. What it costs is
   what any new question on that walk costs: a `walkQuestionBudget` raise
   with its argument, and the registrations three censuses will ask for —
   which is two iterations' worth of the evidence from this run.
2. **(age 3 · value low) Record which band ends have actually been reached.**
3. **(age 7 · value low) `…TimingsTakenOn` is not the only record shape** —
   value still low; it unblocks a rule measured at zero.
4. **(age 20 · value low) Every Next list in this loop was written by the
   session that would not work it.** Nine for nine. This item guessed the
   tables were already covered; they were not, and the reason they were not —
   a prefix resolving by design — is a better piece of knowledge than the
   item was asking for.
5. **(declined, non-goal)** Four entries added across this run. See
   `ai_docs/plans/non_goals.md`.
