# Session: a row that said three, a rule declined on nine measurements, and an arm that fired on its own new file

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-figure-in-prose-is-a-copy-a-word-that-filters-386-files-to-eleven-and-a-budget-that-was-advice")

## Ask

`/loop` iteration 2. The list's top item was the whitespace arm; the iteration
opened with something that was not on the list at all.

| item | value | outcome |
|---|---|---|
| — | high | a row the last iteration made wrong, and the arm that counts it |
| 1 | medium | the comment-text arm, its SET decided by measurement |
| 3 | low | two unheld figures removed rather than armed |

---

## The row that said three

`repositoryWalks` lists what each repository-wide walk asks. Its own doc says
why the field is a LIST:

> A paragraph is not countable. A list is: a row with four entries is a walk
> that has quietly become four censuses sharing a parse, and the number is
> printed on every green run rather than being something a reader notices by
> finding the field long.

Iteration 1 added a fourth question to the copies walk and left the row saying
three. Nothing failed. The field written to be countable was counted by nobody,
and it went wrong inside one session of that sentence being written.

### A question is a subtest

`subtestSitesIn` counts the `t.Run` sites a walk's body opens at its own level,
and the row's `asks` must have that many entries. The identity is not a
convention somebody could drop — it is copies_test.go's own argument: each
question ends in a `t.Fatalf` over a walk that reached nothing, a Fatalf ends
the goroutine, and three questions in one function is three questions any one
of which can silence the other two. That had already happened once.

Measured across all seven decided walks before writing it, the rule holds
exactly:

    checkCitationsResolve                             0 subtests, 1 ask
    TestTheCitationSkipsGitAlreadyMakes               0 subtests, 1 ask
    TestEveryGitListingAsksForNulSeparatedPaths       0 subtests, 1 ask
    TestEveryGitListingInAScriptAsksForNul…           0 subtests, 1 ask
    TestTheShortLeversAreTheOnes…                     0 subtests, 1 ask
    TestTheShapesThisRepositoryKeepsTwoCopies…        4 subtests, 4 asks
    TestTheDottedVersionParsers…                      0 subtests, 1 ask

Three things it declines to count, each for a reason:

    a t.Run in a function literal   a subtest of a subtest, on the inner `t`
                                    whatever that is spelled
    a t.Run in a `for`              one site, as many subtests as the loop is
                                    long — the same understatement `runs`
                                    already makes for a walk in a loop
    `Run` on anything else          the receiver is matched against the
                                    function's own *testing.T PARAMETER NAME,
                                    read off the signature rather than assumed
                                    to be `t`, for the reason
                                    importnames_test.go reads a qualifier off
                                    the import

---

## The comment-text arm, and deciding the set

The item said the work was deciding WHICH rules belong. Two of the obvious
candidates were declined on measurement rather than on taste.

### Trailing whitespace — gofmt already holds it

Checked rather than assumed: `gofmt` strips trailing whitespace from comment
lines, and `gofmt -l` is one of the eleven verification paths. A rule for it
would be a second statement of something already held.

### A doubled word — nine hits, not one of them a defect

Measured over every Go comment in the repository:

    what it is is ARIA's                         correct English
    a subtest that fails fails its parent        correct English
    the row that moved moved DOWN                correct English
    deleting the catch-all all land here         correct English
    an ordinary literal; what it actually is is  correct English
    the k-step scale … a million million         deliberate: 10^12
    Jan Jan Feb Feb Feb Jan                      deliberate: example data
    "sermons sermons sermons"                    deliberate: test data
    a List of rows rows                          names the `rows` parameter

The last one is the point. It read like the one real duplication in the set,
and it is a sentence naming the parameter declared on the next line. A rule
that needs a nine-row exemption table on the day it is written, for a shape
that has never once been wrong here, produces noise rather than findings.

### The two that survived

    written twice     an interior `//` whose two halves are the same sentence.
                      The shape repowalks_test.go:395 actually had
    an interior tab   a tab anywhere but in the leading indent. Nobody types
                      one; it is a join, a paste out of aligned output, or an
                      editor — and the incident's two copies were held
                      together by exactly this

Both at **zero across every Go file in the repository**, so neither arrives
with an exemption table. The general form of the first — any second `//` on a
comment line — is unusable: forty-odd legitimate instances repo-wide, mostly
annotated code samples inside doc comments, five of them in this directory.
The rule is the shape the incident had, not the shape it belongs to.

### Why it parses, which the item did not expect

The item said "no new walk, no parse — cheap to detect over this directory's
bytes." It cannot be done over bytes. **A line beginning with `//` is not
necessarily a comment** — it can be a line inside a raw string literal — and a
scanner cannot track raw strings by counting backquotes either, because the
comments here are full of backquoted names and a comment with an odd number of
them drops the scanner into a string that is not there. That was found by
writing the byte scanner and watching it report most of gitquoting_test.go's
header.

So go/parser says where the comments are and `ast.Comment.Text` says what each
says, which is the source as written — the parser does not normalise a
comment's interior, which is the whole reason this works. One directory,
**0.01s** over 12,125 comment lines. Not a repository walk: no budget moves.

### Scope, and the price of widening

This directory. The incident was here, and this is the package that is mostly
prose. A repository-wide version would be a fifth question on the copies walk
— the only arm holding every Go file's bytes and its parse — and a garbled
comment is not a shape kept in two copies. Putting it there because the bytes
are in scope is how a walk becomes a place to put things. Written down, with
what widening would actually cost.

### The file the rules cannot read

Its header quotes the incident verbatim, so it breaks both rules and is a
finding about itself twice over. Skipped by name, like gitquoting_test.go and
copies_test.go, and the trade stated rather than buried: the one file the rules
do not cover is the one somebody editing the rules will be editing. The
alternative was to describe the shape instead of quoting it, and a shape
described in words is one the next reader has to reconstruct — which for two
rules about invisible characters is most of what the header is for.

---

## The arm fired on the file this session added

`commenttext_test.go` is a tracked Go file, so the tree went to 387 and
iteration 1's arm failed the run, naming all five sentences and the number to
write:

    5 sentence(s) quote a tracked-Go-file count this repository does not have.
    git ls-files enumerated 387: repowalks_test.go:30 says 386, in a comment;
    repowalks_test.go:500 says 386, in a string constant; … 

Which is the trade being paid on the first commit after it was made, by the
session that made it, exactly as its header says it would be.

## Two figures removed rather than armed

Last session's Next list raised one: the byte filter's own comment said `11 of
these 386 files`, a figure in prose priced against a walk, in the arm, in the
one file the walk skips. A second appeared this session — the new header's
`38 files`.

Neither is now a number. The filter's note says *eleven files in the tree
contain the word at all*, which is what it was there to say, and the comment
scan logs its count on every run instead of writing it down. Deleting a figure
is the cheaper half of the same decision: a figure is worth having when a
reader needs it to price something, and worth removing when the sentence works
without it.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Five break-tests:**

    asks says three         the row's fourth entry removed. Names the file,
                            the line, 4 subtests against 3 questions, and both
                            directions of what that can mean
    a fifth subtest         a t.Run nobody wrote a row for: 5 against 4
    a nested t.Run          added inside a subtest closure. Count unchanged,
                            no finding — the FuncLit stop works
    written twice           a duplicated comment line: named with its text
    an interior tab         a tab mid-comment: named with its text

**Figures.** Package **2.830–2.914s** over twelve, inside the recorded
2.78–2.93s. The comment scan is 0.01s of it.

## What this iteration is evidence of

Both halves of it came from measurement overturning what the list said.

The Next item predicted a byte scan and it had to be a parse. The obvious rule
set had four rules in it and two of them died on their own numbers — one to a
`gofmt` invocation, one to nine grep hits, and the strongest evidence in the
whole iteration was the single hit that looked real and was not.

And the item that was not on the list at all — a row saying three — was there
because the previous iteration put it there. A list written by the session
that will not work it does not contain what that session broke.

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, counted in `ai_docs/claude_sessions/`
and measured from this doc, so `age 0` means it was raised here.

1. **(age 0 · value medium) The two comment rules are this directory only.**
   The shape they catch is repository-wide and so is the class it belongs to;
   what stopped the widening is that the only arm holding every file's bytes
   is about something else. The decision to make is whether a sixth repository
   walk against a budget of six is worth a rule that has fired once — the
   budget exists to force exactly that conversation, so have it rather than
   inheriting the scope by default.
2. **(age 3 · value low) Every Next list in this loop was written by the
   session that would not work it.** Iteration 2 is new evidence and not the
   same evidence: the item it opened with was a regression the previous
   iteration introduced, which no list written by that iteration could have
   contained. The cold read finds what an author cannot see; this finds what an
   author has just done.
3. **(age 0 · value low) `subtestSitesIn` counts a `t.Run` in a loop as one.**
   Stated in its doc rather than resolved — the loop's length is a run-time
   fact no parse has, which is the same limit `priceCalls` already writes up
   for a walk in a loop. No such site exists in this package today. Worth
   revisiting only if a table-driven walk test appears, where one site would be
   several questions and the row would say one.
4. **(age 2 · value low) `packageLevelCallsTo` cannot tell an initializer from
   a function value a `var` holds.** Stated in the message rather than
   resolved, because resolving it is data flow and these walks decline type
   information. Worth revisiting only if the case ever occurs — today no
   package-level declaration in this directory calls anything either caller
   asks about.
5. **(declined, non-goal) The diff-and-print remainder stays unmeasured.** See
   `ai_docs/plans/non_goals.md` — what was declined, the argument, and the two
   conditions that would change it.
