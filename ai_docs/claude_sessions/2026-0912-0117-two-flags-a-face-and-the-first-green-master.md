# Session: two flags, a face, and the first green master

Session: https://claude.ai/code/session_018HHEppckHrYoZNwpi2jgha
Date: 2026-09-12 · Previous:
`2026-0912-0050-the-run-that-checked-the-checks-and-the-one-precondition-nothing-asked-for.md`

## Ask

> add the flag and push

and then, after what that turned up:

> skip the ink scan when the fonts aren't the calibrated ones

Three commits — `69cf63f`, `faa0b7c`, `e1b11db` — and the result is that
**both workflows are green on master**, which neither has been at any point
this session could see.

```
CI   ✓ Go, WASM and the JS runtime   ✓ iOS   ✓ Android      run 34677613109
site ✓                                                      run 34677613114
```

The four sentences worth keeping:

1. **`--disable-lcd-text` was the right flag, and the runner is what said so.**
   Every antialiasing probe passed on the first Linux run that had it, where
   20 of 20 labels and 8 of 8 counts had been failing.
2. **Fixing it revealed that the flag was never the whole problem.** With the
   probes passing, the ink scan ran further and failed on row selection — the
   scan's numbers are readings of *one face's outline*, and the runner has a
   different face.
3. **A browser that would not start said nothing at all**, because its stderr
   was being discarded. Every startup failure — bad flag, crash, missing
   library, slow machine — produced one identical sentence, on the one class of
   machine nobody here can attach to.
4. **The band census caught the skip immediately**, which is what it is for,
   and exempting it was the one real weakening in this work. The argument is
   written where the change is.

---

## `69cf63f` — the flag, and what it proved

`browser.mjs` launches Chrome with eight flags and every one of them carries an
argument for why. Grayscale rendering was the one precondition in that list
that was **assumed** rather than requested, and the probe verdict said so out
loud in the wrong direction:

> Headless Chrome disables it; something about this browser, its flags, or that
> declaration has changed.

A statement about a default, and the default is not the same everywhere:

```
macOS headless Chrome      grayscale     20 of 20 bands read
Linux headless Chrome      subpixel       0 of 20 bands read
```

`--disable-lcd-text`, with its paragraph, and the verdict rewritten to name the
flag so a future failure points at the flag having stopped working rather than
at a phantom regression.

**Unverifiable here by construction** — macOS already renders grayscale, so the
flag is a no-op on this machine and `wasm/verify/run.sh` was green before and
after, 20 words and 8 counts either way. The next CI run settled it: not one
coloured-fringe message.

---

## `faa0b7c` — a Chrome that will not start, saying why

The first runner to take the flag failed with

```
Error: Chrome never reported a DevTools port
```

and nothing else. That looked like the flag and was not — CI's Chrome started
on the same commit with the same flag, and the probes passed. It was a separate
launch failure in the `site` workflow, and **there was no way to tell from the
log**, because `stdio` was `["ignore", "ignore", "ignore"]`.

One sentence for four different things to do about it. So:

- Chrome's stderr is kept, capped at the last 40 lines, and the error carries it.
- The error names whether the process is still alive and with what status.
- The wait stops as soon as Chrome has exited — checked *after* the port file,
  so a browser that wrote its port and then died still yields a usable answer.
  Before, every startup failure took the full ten seconds to prove something
  already known.

Verified by forcing a launch that cannot work:

```
Chrome never reported a DevTools port: Chrome exited (status 21) without
writing one.

Its last 2 line(s) of stderr:
  ...Failed to create .../SingletonLock: File exists (17)
  ...Failed to create a ProcessSingleton for your profile directory...
```

which is the whole diagnosis, where the old message had none.

**Beyond what was asked**, and said so at the time. The justification is that
the remaining unknown was unreadable without it.

---

## `e1b11db` — the ink scan knows which face it was measured on

With the probes passing, the scan got further and failed differently:

```
MaterialTheme/a plain band: the label's words are scanned on row 468, and one
device row out from it the window is 2.9% glyph — 80.0% above and 2.9% below —
against a floor of 15%.
```

Nothing was wrong with the bands. Every number in the scan is a reading of one
face's outline — the three rows are fractions of its ink band,
`INK_EDGE_CLEARANCE` is one device row of it, `INK_ROW_GLYPH_FLOOR` is what a
row of its glyphs comes to — and a different face puts its baseline and
x-height somewhere else, so a fraction calibrated on one picks a different row
on another. A row on an edge is the exact case `INK_EDGE_CLEARANCE` exists to
avoid, measured, on one face.

### What the face is, and why that is the point

Read off this machine rather than assumed: every band, badge and probe resolves
to **Times**. The grid asks for `core.Theme`'s typography, this Chrome has none
of it, and everything falls through to the default serif. So the recorded name
is a *platform default* rather than a choice — which is precisely the thing that
differs between machines. The runner resolved **Liberation Serif**, Linux's
metric-compatible Times substitute, and the skip named it.

### Skipped, not re-derived

Re-deriving the fractions from whatever face resolved is the fix that looks
principled and is not: the floor, the clearance and the three fractions were
each arrived at by looking at readings on one face, and a formula reproducing
them for Times would be fitted to one point. What this can say honestly is that
it does not know — the stance `ios/verify` takes toward a missing iPhoneOS SDK.

### What still runs

The suppression is the two glyph readings only, as one decision computed beside
`pixelsHeld` rather than a condition repeated at each sampling site.

| Reading | Under an uncalibrated face |
|---|---|
| tap targets, insets | run — rects have no outline to be calibrated on |
| the band fill | runs — a solid colour |
| the antialiasing probes | **run** — they answer about the rendering MODE, which is as true on one face as another |
| the words are in their own ink | skipped |
| the counts are digits | skipped |

The probes staying is load-bearing rather than incidental: they are the
standing confirmation that `--disable-lcd-text` is still working, on the one
machine where it is the only thing keeping them grey.

### The one thing weakened, and the argument for it

The band census caught the skip on the first try — it exists to stop a
suppression from passing as a smaller grid, and it pushed two problems:

```
20 of the 20 bands that declare a label did not get their words in their own ink — 0 did.
 8 of the  8 bands that declare a count did not get their counts as digits — 0 did.
```

Its `words` and `counts` rows are exempted when the face skip is set. That is a
real weakening and the distinction is written where the change is made:

- **Every other suppression there is partial and unintended.** A probe came back
  coloured, a label's node was missing, the grid was clipped at some row. Some
  bands were read and some were not, the difference is the interesting quantity,
  and no single message carries it — which is the gap the census fills.
- **This one is total and declared.** Nothing was read, one `SKIP:` line carries
  the reason *and* the remedy, and the tail refuses to recite a number rather
  than reciting zero. So the silent zero the census exists to prevent cannot
  happen on that path, and counting it would turn a missing font into a red
  pass — the inversion `android/verify/gate.sh` was extracted to make
  impossible, one check over.

### Four arms, because only one is reachable here

```
calibrated                          20 and 8 read, exit 0
uncalibrated face                   SKIP, exit 0, tail says "unread"
calibrated + a real shortfall       FAIL: 1 of the 20 … — 19 did
uncalibrated + a fills shortfall    FAIL — the exemption is exactly two rows wide
```

The last one is the "did I weaken it too far" check, and it is the one worth
keeping: a machine missing a font must not excuse a band that stopped painting.

---

## What is checked, and what is not

- `go test ./...` — green. `wasm/verify/run.sh` — green, both face arms driven.
- **CI and site, both green on `e1b11db`**, which is the first time either has
  been this session. The go job's tail carries the SKIP naming Liberation
  Serif; the Android job's three steps are green for the second run running.
- **Not checked:** why `site`'s Chrome failed to start on `69cf63f`. It has not
  recurred, and `faa0b7c` is the instrument if it does.
- **Not checked:** anything needing a real caret, keyboard, IME or device. See
  `ai_docs/plans/need_hardware.md`.

---

## Next

1. **(new · value medium) The ink scan is unread on every Linux machine, and
   that is now quiet rather than red.** A skip is the honest answer and it is
   not the end state: the two readings it suppresses — that a band's words are
   there in its own ink, that a count is digits — are the ones the band grid
   exists for, and no CI run exercises them any more. The two ways out are
   installing the calibrated face on the runner, or re-measuring `INK_ROWS`,
   `INK_EDGE_CLEARANCE` and `INK_ROW_GLYPH_FLOOR` against a second face and
   recording both. The second is better and is a measurement session, not a fix.
2. **(new · value low) `site`'s Chrome failed to start once, and nothing knows
   why.** It has not recurred. `faa0b7c` makes the next occurrence legible, so
   this is a note to read that output rather than a task.
3. **(carried · value medium) Nothing checks that a gate's remedy sentence
   works.** Unchanged, and this session added another one to the pile: the ink
   skip tells a reader to install a face or re-measure three constants, and
   nobody has done either. The shape is: strip the precondition, run the
   remedy, run the pass, expect not-a-skip.
4. **(carried · value low) `retire`'s bound is conditional and stated, not
   enforced.** Unchanged.
5. **(carried · value low) Windowing's device number.** Unchanged. Declined on
   the payload in `non_goals.md`; `TestWhatWindowingWouldSave` asserts its own
   share.
6. **(closed) The band grid's ink scan fails on every Linux runner.** Done this
   session, in two parts — the antialiasing mode is asked for now, and the face
   is checked rather than assumed.
7. **(moved) The four hardware items.** `ai_docs/plans/need_hardware.md`.
8. **(declined, non-goal)** Twenty-seven entries, unchanged. See
   `ai_docs/plans/non_goals.md`.
