# Session: a lever put where it is read, a prefix with two meanings, and an OVER that was the race suite

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · iteration 8 of a ten-iteration `/loop`
Previous: `2026-0911-0256-three-rules-about-ends…`

## Ask

Next item 1 (age 0, value medium): the band verdict became opt-in last
iteration, so nothing checks the bands on an ordinary run. Put
`GRMOB_BAND_VERDICT=required` where a session will exercise it.

---

## Where it went, which is not where the item said

The item suggested `wasm/verify/run.sh`. That script does not run `go test`
at all — it runs `go run .` and a Node test pass. `build.sh` is a build. The
eleven verification paths are prose in the session docs and not a script
anywhere, so there was no existing place to add a twelfth line to.

So the lever went into **both records' own "Re-taking it" tables**, which is
where a person who needs it is already reading, and the sections are renamed
*Re-taking it, and CHECKING it without re-taking it*:

    GRMOB_BAND_VERDICT=required go test -count=1 -v ./wasm/verify
    GRMOB_BAND_VERDICT=required go test -count=1 -v ./internal/themehistory

Both were run exactly as written. That is cheaper than re-taking a figure —
it is the command already in the table with four more words in front of it —
and it is the answer to "has this gone stale", which is the question the
table previously had no cheap answer to.

## An OVER that was the race suite, and the method line that now says so

The first reading taken with the lever came in at **3.176s, OVER** — taken
immediately after `go test -race ./...` and the three verify scripts. Six
runs later, with nothing else running: **2.684–2.741s, all in band.**

That is the verdict's own message being right — *one reading over a ceiling
is not a regression, take it several times* — and it is now a concrete line
in the method: alone is not sufficient, it also has to be idle.

## A candidate rule declined on a prefix that turned out to mean two things

`GRMOB_[A-Z0-9_]+` looked like the narrow shape that made the test-name rule
work: a prefix this repository owns, screaming case, nothing in English like
it. The rule would hold every `GRMOB_` name in prose to being something the
repository reads.

**Eleven distinct names match it and four are not environment variables.**
`GRMOB_PROGRESS_DETERMINATE`, `…_INDETERMINATE`, `…_UNSTATED` and
`…_EMPTY_RANGE` are Kotlin `const val` declarations holding "determinate",
"indeterminate", "unstated" and "empty-range". 36% of the corpus is a second
meaning — the same failure as the backquoted-name rule, and worse, because
that one is ambiguous across languages and this one is ambiguous inside one
directory of one language.

**And there is nothing to find**: 91 mentions, every one resolving to a name
something reads or declares in whichever of five languages owns it. Zero
stale.

Worth recording separately: the first scan reported **32 unresolved**, and
all 32 were the scan looking only at Go string literals in a polyglot
repository. A fault in the measurement, not a finding — and the kind that
would have produced a rule with a thirty-two-entry exemption table if the
number had been believed instead of read.

Declined in `ai_docs/plans/non_goals.md`.

## Checked and left alone

`verify.test` at the repository root is a 7MB `go test -c` binary from
yesterday. Untracked, and `*.test` is in `.gitignore` — not a defect and not
mine to delete.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

Figures at the end:

    wasm/verify             2.858–2.934s   recorded 2.78–2.97s
    …wholeFileInProcess     2.684–2.741s   recorded 2.55–2.78s  (six runs
                                           after one 3.176s OVER on a busy
                                           machine)
    internal/themehistory   3.060–3.081s   recorded 2.86–3.22s
    …wholeRun, …batchRetire in band when asked

## Next

1. **(age 0 · value medium) Nothing holds the two "Re-taking it" tables to
   being runnable.** Both were run by hand this iteration and both worked,
   but a command in a comment is prose, and this repository has already
   found a `git`-listing command in a comment that nobody could run and four
   comments pointing at tests that were gone. The shape is narrow — a line
   in a tab-indented block inside a `# Re-taking it` section, beginning with
   `go test` or an env assignment — and the check is that the test names in
   it resolve, which the prose-names rule already does for the whole
   repository. So the cheap version may be nothing more than confirming
   those tables are already covered by it, and the finding would be that
   `TestTheWholeWalk` and `TestOneProcessPerObject` in the themehistory
   table are PREFIXES, which that rule accepts by design.
2. **(age 2 · value low) Record which band ends have actually been reached.**
   Sharpened but unchanged.
3. **(age 6 · value low) `…TimingsTakenOn` is not the only record shape** —
   value still low; it unblocks a rule measured at zero.
4. **(age 19 · value low) Every Next list in this loop was written by the
   session that would not work it.** Eight for eight. This item named a file
   that could not host its own suggestion, and the work landed somewhere
   better.
5. **(declined, non-goal)** Three entries now added across this run. See
   `ai_docs/plans/non_goals.md`.
