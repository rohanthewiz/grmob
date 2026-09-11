# Session: a figure in prose is a copy, a word that filters 386 files to eleven, and a budget that was advice

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-line-that-was-its-own-comment-twice-a-call-in-no-function-and-six-figures-nothing-held")

## Ask

`/loop` — work the Next list, five iterations or until it is empty. This is
iteration 1, and it took the top item and one of the cheap ones.

The top item was the cold read's own finding turned into work: five sentences
priced a repository walk against how many Go files it parses, all five had
drifted by four at once, and nothing held any of them. The item came with the
measurement to take first — what `parser.ParseComments` costs on a walk that
already parses — and the answer moved the design twice.

| item | value | outcome |
|---|---|---|
| 1 | high | the count is armed; a fourth question on the copies walk |
| 4 | low | `filesIn`'s budget says out loud that it is advice |
| 2, 3, 5 | — | carried, unchanged |

---

## Item 1 · a figure in prose is a copy

### Where it went, and why there

`copies_test.go` is the file about shapes this repository keeps two copies of.
A sentence saying `386 tracked Go files` is a copy of a number that lives in
the tree, kept in step by somebody remembering to — which makes it the purest
instance of that file's subject and the only one whose second copy is not
source. It is a fourth subtest on the existing repository-wide walk, not a walk
of its own: `repositoryParseBudget` is 4 and a fifth parse is the decision that
budget exists to force.

The walk already had both halves. The enumeration above it is the count; the
parse below it is the prose. The question costs the walk nothing it was not
already paying.

### It can be an arm, and the wall clocks beside it cannot

The same paragraphs carry timings, and this package's standing rule is that a
timing cannot be an arm — asserting one fails on every computer that is not the
one it was taken on. A file count is a different kind of fact: it is about the
TREE, so a figure that disagrees with the tree disagrees for everybody. That is
the whole argument for the asymmetry, and it is now written in both files.

The cost is that adding a Go file fails a test until five sentences are edited.
That is the point rather than a side effect — the alternative is what they
were — and the failure names each line so the fix is one pass.

### The form is fixed, because a pronoun is not a number

The five sentences wrote the count three ways:

    386 tracked Go files      the number and what it counts in one breath
    386 of them               a number only because of the clause before it
    386 where                 the same

Holding the last two means guessing how far back a pronoun reaches, in
paragraphs that also quote wall clocks, core counts and budgets — a check that
invents findings. So the convention is the first form, and all five sentences
were rewritten into it. It is the same rule `coresAttribution` states one file
over for the terms it names: backquotes say a word is meant as a declaration's
name, and a term named in prose is neither claimed nor checked.

What is held is therefore the FORM and not the practice. Written down as a
limit rather than resolved.

### The line break that makes a phrase two phrases

    // … a reading of a repository on a day — 386
    // tracked Go files where verifyTimingsTakenOn was taken — and it …

One phrase to a reader, two to anything matching raw lines — and it is how the
first of the five is actually written. So a comment group is collapsed to
single spaces before the pattern runs.

Collapsing loses the line, and a comment group here is routinely forty lines
long: a finding that pointed at the top of a section and left somebody to find
the sentence would have thrown away most of what the message is for. `prose`
keeps a line number per word and attributes a match to the word it starts at,
which is what makes `repowalks_test.go:30` the answer instead of `:16`.

The same type takes the literals of a `+` chain, so a figure inside
`coresAttribution` is attributed to the literal it is written in rather than to
the head of the chain. `stringLiteralProse` walks exactly the shapes
`stringLiteralValue` evaluates — a sentence one could read and the other could
not would be a finding whose line number pointed somewhere else.

### The measurement, which was the item's actual question, and then a second one

The item said to measure `ParseComments` before deciding. Over this walk's own
386 files, five runs apiece:

    SkipObjectResolution              0.040s
    SkipObjectResolution|ParseComments  0.044s

Four milliseconds. Affordable, so the question went on the existing walk.

That was the wrong number to have stopped at. The comments are the cheap half;
the SCAN is not. Isolating the subtest:

    with the figure scan       0.477–0.511s
    without it                 0.387–0.419s

**0.09s** — 20% of that walk and 3% of the package, for a question with five
answers in it, because it walked every node of every syntax tree in the
repository and built a run of text out of every comment group and every string
constant in it.

The fix is the move the walk census already makes one file over: a byte scan
before the parse, which cannot produce a false negative because a figure in the
held form contains the word whatever else the file says. Filtering on `tracked`
takes 386 files to **11**, and the scan's cost back into the noise:

    filtered                   0.382–0.400s

The bytes are read in the loop now and handed to `parser.ParseFile`, which
reads the file itself when passed nil — the same read moved rather than a
second one, doing two jobs.

**The record did not need touching.** Twelve runs of the package afterwards:
**2.815–2.924s**, inside the recorded 2.78–2.93s. Twelve runs before the filter
were 2.904–2.996s, which is mostly outside it — so the reduction is the
difference between a change that costs nothing and one that would have needed
the band re-taken.

### What it reports

    386 tracked Go file(s) by git ls-files; 5 sentence(s) quote that count:
      wasm/verify/repowalks_test.go:30 says 386, in a comment
      wasm/verify/repowalks_test.go:472 says 386, in a string constant
      wasm/verify/repowalks_test.go:669 says 386, in a comment
      wasm/verify/timings_test.go:259 says 386, in a comment
      wasm/verify/timings_test.go:318 says 386, in a string constant

Logged whether or not anything fails, because the useful moment for the number
is BEFORE somebody edits prose: it is how a person adding a walk finds the count
to write into the sentence they are about to add.

Comment or string constant is not decoration. A string constant here can be
something another package's test reads without running this one —
`coresAttribution` is exactly that — so editing one has a second consequence a
comment does not, and somebody handed a list of lines wants to know which kind
each is before they start.

### The self-skip is now load-bearing

The walk skips `copies_test.go` by name so that a check cannot read its own
explanation as a finding. That skip was a convenience; it is now structural,
because this file's header quotes a figure as an example and would otherwise be
a finding about itself. Said so in the header rather than left to be
rediscovered by whoever removes the skip.

---

## Item 4 · a budget that was advice

`packageSource.filesIn` reports over `coresNoteScanDirs` with `t.Errorf` and
then lists the directory anyway. Deliberate, and written down nowhere, which
made the bound read as enforced.

The argument is the one the `ReadDir` failure below it already makes: a
truncated scan produces the wrong finding in front of the wrong person. Every
term the cores note names would read as one this package no longer has, because
the files declaring them were never listed — one true finding about a cache and
a dozen invented ones about a note, with nothing saying which was which.

So it is a TRIGGER and not a limit. What it bounds is how long a growing cache
can go unremarked, which is one run — not how much memory it may hold, which is
bounded by the repository.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Two break-tests for the new arm, both re-run after the byte filter was added
to prove the filter changed no answer:**

    a stale figure     timings_test.go's string constant set to 385. Names
                       that line, that kind, and the count to write
    the form is gone   `tracked Go files` → `tracked Golang files` throughout.
                       The reaching arm fires: no sentence quotes a count, and
                       the message says the two things that can mean

**Figures.** Package **2.815–2.924s** over twelve, inside the recorded
2.78–2.93s.

## What this iteration is evidence of

The item carried a measurement to take and the measurement was right — but it
was a measurement of the wrong half. `ParseComments` was 4ms and the thing it
enabled was 90ms, and nothing in the item said to time the second one, because
when it was written the scan did not exist to be timed.

Which is the same shape the arm itself is about, one level up: the item priced
the work against the number it had, and the number it had was not the number
that mattered. The package's own answer applies — take the reading, then attach
it to the thing it is a reading of.

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, counted in `ai_docs/claude_sessions/`
and measured from this doc, so `age 0` means it was raised here.

1. **(age 1 · value medium) The duplicated comment line argues for a
   whitespace arm.** Nothing in eleven verification paths reads a comment as
   text, and a line that is one comment concatenated with itself survived at
   least a session. Cheap to detect over this directory's bytes, which the walk
   census has already read — no new walk, no parse. The question is what else
   belongs in it: a trailing-whitespace rule and a tab-inside-a-line rule are
   the same scan, and deciding the SET is the work. Note that this session
   built machinery for exactly this shape — `prose` collapses a comment group
   and keeps a line per word — but over the whole repository rather than this
   directory, and the whitespace question is about raw lines rather than
   collapsed ones, so it is a neighbour and not the same scan.
2. **(age 2 · value low) Every Next list in this loop was written by the
   session that would not work it.** The previous session's cold read found
   three things five lists had not. Still one pass by one reader; the next one
   would be reading files this session has just rewritten.
3. **(age 0 · value low) The figure scan reads 11 files of 386 and the 11 is a
   reading.** `bytes.Contains(raw, "tracked")` is what makes the fourth
   question free, and the count of files it lets through is quoted in a comment
   beside it — which is a figure in prose priced against a walk, i.e. exactly
   the shape this session armed, in the arm. It is not held, twice over: the
   figure sits in copies_test.go, which the walk skips by name, and it is a
   count of files containing a word rather than a tracked-Go-file count, so
   the form would not read it either. Either widen the form and find it
   somewhere the walk looks, or accept that the arm's own cost note is the one
   unattributed figure left in the package and say so beside it.
4. **(age 1 · value low) `packageLevelCallsTo` cannot tell an initializer from
   a function value a `var` holds.** Stated in the message rather than
   resolved, because resolving it is data flow and these walks decline type
   information. Worth revisiting only if the case ever occurs — today no
   package-level declaration in this directory calls anything either caller
   asks about.
5. **(declined, non-goal) The diff-and-print remainder stays unmeasured.** See
   `ai_docs/plans/non_goals.md` — what was declined, the argument, and the two
   conditions that would change it.
