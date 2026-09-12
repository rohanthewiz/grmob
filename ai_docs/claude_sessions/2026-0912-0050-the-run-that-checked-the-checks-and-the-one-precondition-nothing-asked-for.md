# Session: the run that checked the checks, and the one precondition nothing asked for

Session: https://claude.ai/code/session_018HHEppckHrYoZNwpi2jgha
Date: 2026-09-12 · Previous:
`2026-0911-2317-the-checks-that-only-ever-ran-here-and-a-remedy-sentence-that-did-not-work.md`

## Ask

A wrap, and then the CI run the previous wrap had just triggered. Nothing was
written this segment; what it produced is a verification and a diagnosis, and
both are worth recording because one of them closes that doc's own top Next
item and the other replaces it.

No code changed. `ef06058` is what is on master.

The three sentences worth keeping:

1. **The previous session's unverifiable half verified.** Its Next item 1 was
   "the three new Android CI steps have never run on a runner, and the failure
   mode to expect is a SKIP where a run was intended, which is quiet." They
   ran. Nothing skipped that was not meant to.
2. **Both CI fixes are confirmed on the machine they failed on.** `go test
   -race` passes in CI for the first time in the history this session could
   see.
3. **Master is red for one reason now instead of three, and the reason is the
   same shape as the rest of this work:** a precondition a check *assumes*
   instead of *asks for*.

---

## What the runner said about the previous session's work

Run 34672699921, commit `ef06058`.

```
iOS runtime conformance              PASS   1m14s
Android AAR and Compose renderer     PASS   4m27s
Go, WASM and the JS runtime          FAIL   2m27s   (one step, see below)
```

### The Android job, step by step

```
Populate the gradle cache android/verify reads
    all 7 compiler jars + foundation-layout's sources jar   46s
Compile the Compose runtime (no NDK, no gomobile)
    OK:   com.grmob.runtime compile against the classpath gradle resolves,
          with the Compose compiler plugin
    SKIP: com.grmob.app (no android/app/libs/grmob.aar)      23s
Build the AAR with the pinned gomobile                       (existing)
Assemble the debug APK                                       (existing)
Run android/verify against the built AAR
    OK: the gate harness counts, discriminates and exits
    OK: both gates answer every precondition, in order
    OK: the Compose census's claims were read out of foundation-layout's sources
    OK: com.grmob.runtime and com.grmob.app compile …
    OK: 15 picker menus match Go's decomposition
    OK: 22 value ranges match Go's reading
```

Four things in that worth keeping:

- **The stage split works as designed.** The early step compiled the runtime
  and named the app stage as skipped, because the `.aar` does not exist yet at
  that point in the job. That is the whole argument for `sources.sh` existing
  separately from `:app:compileDebugKotlin`, and it is the first time anything
  has demonstrated it rather than asserted it.
- **`fetchKotlinCompiler` resolved the same seven jars on Ubuntu that it
  resolved from an empty `GRADLE_USER_HOME` here**, versions included —
  `kotlin-reflect/1.6.10` arrives transitively on both, which is the one entry
  that is not at the build's Kotlin version and is now known to be that on two
  machines rather than one.
- **The Compose census ran.** It had been a permanent SKIP on every CI machine,
  because the sources jar is fetched by no build and the message that says so
  is printed into a log nobody reads. It reads androidx's own arithmetic now.
- **Nothing skipped quietly.** That was the named failure mode, and it did not
  happen.

### And the two fixes, on the machine that had them

```
go vet         PASS
go test -race  PASS      <- both CI-only failures, gone
Build the WASM target    PASS
```

`TestTheCitationSkipsGitAlreadyMakes` and
`TestRetiringAWedgedProcessDoesNotWaitForever` were each fixed against a
reproduction — a fresh clone for the first, a three-way process probe for the
second — and this is the confirmation that the reproductions were of the right
thing.

---

## The one that is left: an antialiasing mode nothing asks for

`./wasm/verify/run.sh` fails, in the `CI` workflow's go job **and** in the
separate `site` workflow's verify job. Same cause in both. In `site` it is
pre-existing — the two prior runs checked carry the identical failure. In the
go job it is newly *reached*: `go test -race` used to fail first, so every step
after it was skipped and this was never run there at all.

### What it is

The band grid's ink scan. Each band declares an antialiasing probe that paints
that band's own face and weight in black, at its own alpha, on white — and
every blend of two greys is a grey, so a coloured pixel in the probe means LCD
subpixel antialiasing, where each channel gets its own coverage.

```
DefaultTheme/a plain band          #87AFE7,  96 channels apart
DefaultTheme+tallCaption/a band    #E7BF87,  96 channels apart
MaterialTheme/a plain band         #1F67BF, 160 channels apart
…
20 of the 20 bands that declare a label did not get their words in their own ink — 0 did.
 8 of the  8 bands that declare a count did not get their counts as digits  — 0 did.
```

Nothing is wrong with the bands. `offSegment` holds every pixel in a label's
box to the line between the fill and the ink on the argument that coverage is
one scalar, and `INK_EPSILON`'s exact match on a stem's interior rests on the
same thing. With subpixel rendering on, **ordinary correct text is off that
line**, and the check says so itself in every one of those messages.

### Why it is the same finding as the rest of this work

The check's message ends:

> Headless Chrome disables it; something about this browser, its flags, or that
> declaration has changed.

That sentence is a belief about a default, and the default differs by platform:
macOS headless Chrome renders grayscale and Linux's does not. So the pass is
green here and red on every runner, which is the third instance this session of
one machine's configuration being mistaken for a fact.

And it is avoidable in the file's own idiom. `browser.mjs` launches Chrome with
eight flags and **every one of them carries an argument for why** —
`--disable-smooth-scrolling` because a smooth scroll makes "did not scroll" and
"has not finished scrolling" the same measurement; `--window-size=800,900`
because the grid's fold budget is that height; the colour-management flags
because the palette check reads hexes back out of a screenshot. Grayscale
rendering is the one precondition in that list that is assumed rather than
requested.

### The fix, not taken

One flag — `--disable-lcd-text` — in that launch block, with its own paragraph,
plus a rewrite of the closing sentence above, which currently sends a reader to
look for something that changed when nothing did.

Left for the user's call, and unverifiable here either way: macOS already
renders grayscale, so adding the flag is a no-op on this machine and only a
runner can say whether it is the right one. That is itself the lesson of the
previous session restated — the instrument for a machine-specific precondition
is a machine that has the fault.

---

## What is checked, and what is not

- Everything the previous session changed: confirmed green on a GitHub runner,
  including the three new Android steps and both CI-only test fixes.
- **Not checked:** whether `--disable-lcd-text` fixes the ink scan. Not
  attempted.
- **Not checked:** anything needing a real caret, keyboard, IME or device. See
  `ai_docs/plans/need_hardware.md`.

---

## Next

1. **(new · value high) The band grid's ink scan fails on every Linux runner,
   and `browser.mjs` does not ask for the rendering mode it needs.** This is
   the only thing red on master now — the `CI` go job and the whole `site`
   workflow. The fix is `--disable-lcd-text` in the launch block plus a
   rewrite of the message that currently blames the browser for changing;
   the flag is a no-op on macOS, so the runner is the only instrument.
2. **(carried · value medium) Nothing checks that a gate's remedy sentence
   works.** Partly exercised this segment — CI's cold cache did take the fetch
   path and it did populate the compiler, so `kotlin_source_verdict`'s new
   sentence is true on one more machine. Still nothing *checks* it, and the
   census sentence and the harness gate's sentence are unexercised. The shape
   is: strip the precondition, run the remedy, run the pass, expect
   not-a-skip. Needs a scratch `GRADLE_USER_HOME` and a network, so it is a
   script somebody runs rather than a test.
3. **(carried · value low) `retire`'s bound is conditional and stated, not
   enforced.** Unchanged. `git cat-file --batch` does not fork, so the Kill
   always reaches the pipe; the Setpgid fix is named above `retire` for
   whoever puts another command behind that reader.
4. **(carried · value low) Windowing's device number.** Unchanged. Declined on
   the payload in `non_goals.md`, so it gates nothing; if a forty-lesson
   chapter ever ships, `TestWhatWindowingWouldSave` asserts its own share and
   is the profile to re-take.
5. **(closed) The CI workflow's three new Android steps have never run on a
   runner.** They ran, green, nothing skipped that was not meant to.
6. **(moved) The four hardware items.** `ai_docs/plans/need_hardware.md`.
7. **(declined, non-goal)** Twenty-seven entries, unchanged this segment. See
   `ai_docs/plans/non_goals.md` and its pointer to `need_hardware.md`.
