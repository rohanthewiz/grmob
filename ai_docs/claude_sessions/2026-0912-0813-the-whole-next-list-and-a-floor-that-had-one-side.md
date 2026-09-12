# Session: the whole Next list, and a floor that had one side

Session: https://claude.ai/code/session_01BxZEsiUvkVxKKe3fZsRN8y
Date: 2026-09-12 · Previous:
`2026-0912-0440-a-readme-with-no-images-and-a-lesson-count-that-said-42-49-and-51.md`

## Ask

> Do the items in the Next list

Nine entries. Three new (the unpinned screenshots, the deleted harness, the
counter with no package), three carried and parked on external blockers, and
three already closed. Five of the six live ones were finished; the sixth is
a fact about a font this machine does not have and was instrumented instead.

The five sentences worth keeping:

1. **A screenshot is a figure in prose that happens to be an image.** The
   previous session's shots were seven facts about seven screens with
   nothing reading them. They are now held by their own text: a manifest
   says what each shows, the app's own package renders it and asserts the
   strings, and the README's caption quotes one of them so the prose and
   the render fail together.
2. **Doing item 3 first made item 2 smaller.** The old harness rewrote
   `wasm/main.go`'s dot-import with `sed` because the counter had no
   package and the mounted app was a compile-time choice. Once
   `examples/counter` existed, the honest fix was a host with a registry
   and a `?app=` parameter — no edits to tracked files, no restore path, one
   build for every shot.
3. **The harness reproduces what is committed, so it was made to.** The
   re-taken shots differed from the old ones by a shadow token and a scroll
   offset. Rather than leave `docs/images/` as something the harness nearly
   produces, the seven files were replaced with its output.
4. **`INK_STEM_REACH` now sits between two measurements.** It had 0.907
   above it and nothing below. Check 13 paints a real pill's own colours at
   five dilutions: 0.75 is refused at 75.1%, 0.85 accepted at 85.0%. The
   comment's claim that a dilution is invisible to `offSegment` is also
   measured now, which is what makes the two catches non-redundant rather
   than belt and braces.
5. **A remedy sentence is the one part of a gate nothing could check.**
   `gate_test.sh` proves the sentences are different and correctly ordered;
   nothing proved any was true, and two of them had once named a gradle task
   that fetches no compiler. The drill runs the remedy on a machine with the
   fault.

---

## Item 3 — `examples/counter`

The README's first app and `docs/getting-started.md`'s first app were two
snippets of a file that did not exist, and they had already drifted: one
carried `core.Padding(24)` and the other did not.

`examples/counter/app.go` is now that file — `package counter`, an `init`
that registers, an `AppName`, and the `App` both documents show.
`TestTheDocumentedCounterIsThisPackage` holds both documents to it on
`examples/todoapp/tutorial_excerpt_test.go`'s rule: every non-blank,
non-comment line of a quoting fence appears in the source, in order.

Two details it forced:

- The README said `package main`. It says `package counter` now, with the
  file path in a leading comment, because the fence is a quotation of a real
  file rather than a sketch.
- The markers that decide "is this fence about this package" are
  `count := core.NewState(ctx, 0)` and
  `func AppName() string { return "Counter" }`, not `func App(`. That
  signature appears in eight documents in this repository and names a
  different app in most of them.

The test has a floor (`found < 2`) for the usual reason: a rename in
`app.go` would quietly make every fence "not about this package", and a
test that checks nothing passes.

---

## Item 1 — the screenshots, held

### The manifest

`internal/shotclaims` states, per image: the app, the state it was driven
to, the strings legible in it, which of those the README's alt text also
quotes, and the test that asserts them.

The strings were read off the images rather than guessed. That mattered
twice:

- `tutorial-lesson.png` is **scrolled**, so the lesson's own title bar is
  above the frame. Claiming "1.1  Hello, GrMob" would have been a claim
  this test could hold and no reader could check against the picture. The
  claim is the paragraph, the highlighted code and the TRY IT panel.
- `todo.png`'s four titles are Buy oat milk / Read the GrMob docs / Ship
  the beta / Call Ada, the second and fourth ticked. They are in the claim
  because the picture shows them, which makes the shot reproducible.

`Quoted` is the join. Each entry must be in `Shows` **and** in that image's
alt text, so `0 of 51 lessons opened` is asserted against a rendered tree
and stated in prose, and a lesson added anywhere breaks both.

### What each side checks

| where | question |
|---|---|
| `internal/shotclaims` (test) | every image is claimed and every claim has an image; each row is one of the two shapes; captions quote what the screen shows; every claim has an action script that drives the app it names |
| `examples/*/screenshot_test.go` | the app, driven to the state, still renders the strings |

The manifest cannot render an app and an example package cannot see the set;
that split is why there are two.

`shotclaims.Shown` recurses into nested prop values. Stopping at top-level
strings silently lost the picker's `Free` (a list of option objects) and
every line of every code block (a grid of runs) — a check that passes by
looking away, which is the failure mode being fixed one level up.

`Test` is a string in the manifest and is therefore held for free: the
shared repository parse reads every string literal in every tracked Go file
for Test-shaped names and fails when one names a test that does not exist.

### Two small corrections the tests forced

- `todoapp`'s rows carry their title in a **Style** field
  (`AccessibilityLabel`), not a prop, so the row is found structurally: the
  innermost node that both shows the title and holds a checkbox. Every
  ancestor also satisfies both, so the search is post-order.
- `todoapp`'s store outlives every test in the package, so the screenshot
  test empties the list through the app's own delete button first, and
  again on the way out.

---

## Item 2 — `wasm/shots`

```
wasm/shots/
  host/main.go   js/wasm host; a registry of five example apps, `?app=` picks one
  index.html     the bezel, sized by ?w=/?h=, sets window.__grmobReady
  shot.mjs       a CDP client: mount, drive, clip-capture at scale 2
  hero.html      the composite page, three finished PNGs on one ground
  scripts/*.js   one per shot: a `// grmob-shot: {…}` header and the actions
  shoot.sh       builds into a temp dir, serves it, runs each shot
```

The design difference from the deleted version is the registry. `sed` on a
tracked file has an opinion about what happens if the process dies between
the edit and the restore; a query parameter has none. It costs a bigger
module, paid by a local harness and nobody else — which is also why this is
not `wasm/main.go` with a parameter: that file is what `./build.sh` ships.

`--local` is gone with it. That arm existed because the counter could not be
a package; it is one now.

Things worth knowing:

- **The size is in the script, not in a table.** `// grmob-shot: {"app":
  "todoapp", "w": 414, "h": 600}` — a counter wants a short frame and the
  tutorial a tall one, and that is as much a property of the shot as the
  taps are.
- **The clip is rounded to nearest, not outward.** Outward folded the
  bezel's fractional *position* (centred in a window with an odd remainder)
  into its *size*: a frame asked for at 260 came back 261.
- **`--out -` is the probe.** The action script is the body of an async
  function, so a `return` prints instead of writing. That is how the DOM is
  read at all — the browser-automation tooling refuses this page's content.
- **`toggle` needed `within`.** A checkbox carries the listener and has no
  text, so "the element with the listener whose text includes X" matches
  nothing. `within` finds the innermost element showing the text that
  *contains* such a holder.

`shotclaims_test.go`'s fourth question holds the manifest and the harness
together: every claim needs a script named after its image, every script
needs a claim, and each script's header must name the app the claim says the
picture is of.

Verified by running it: the seven shots were re-taken and committed, so
`docs/images/` is now literally what `./shoot.sh` produces.

---

## Item 4 — the ink floor's other side

`INK_STEM_REACH` is 0.8. It had one reading above it — the thinnest stem the
band grid draws, 1.000 here and 0.907 on Liberation Serif — and none below.
Its own comment said what a dimmer ink *would* read as, which was arithmetic
nobody had asked a browser.

**Check 13** paints a real count pill's own two colours at five known
dilutions and reads what the same `scanInk` says:

| dilution | read as | verdict |
|---|---|---|
| 1.000 | 100.0% | accepted |
| 0.850 | 85.0% | accepted — the lowest reading over the floor |
| 0.750 | 75.1% | refused — the highest under it |
| 0.700 | 70.0% | refused |
| 0.500 | 50.0% | refused |

Three design points:

1. **The digits are set large and bold.** Thinness is the *other*
   experiment. A dilution measured on a thin stem is two effects in one
   number and neither is readable out of it.
2. **The colours come from a real band**, not a pair of hexes written in the
   check. `gen.go` already reads the pill's fill and ink off the rendered
   node; a literal pair would be this check agreeing with itself about a
   widget it had stopped describing.
3. **`INK_FLOOR_BRACKET` is what makes it a pin.** Both halves of the
   verdict derive from `INK_STEM_REACH`, so without a width the check is
   true of any floor — moving it to 0.6 moved the expectation with it.
   Verified: with the floor at 0.6 the nearest pair is 20.2 points apart and
   the drill fails, naming `INK_DILUTIONS` as the thing that has to move.

It also measures why the floor is needed beside the stronger catch: a
dilution is a point *on* the segment between fill and ink, so `offSegment`
cannot see it. That had been reasoning; the boxes come back on the line to
within a fraction of a channel.

The tail now recites both ends on every run. Adding a check meant the
header's numbered list, both prose tallies (`three about paint`, `Thirteen
claims`), and `thirteen` in `checknumbering_test.go`'s word map.

---

## Item 6 — the remedy drills

`internal/gateharness/remedy.sh` is the shape, beside `harness.sh` and for
the same reason: four steps — **strip**, **fault**, **remedy**, **pass** —
with the counting, the substring matching and the footer factored out.

It runs the *pass*, not the verdict function. `gate_test.sh` already calls
the verdict function with hand-set arguments; the subject here is whether a
machine in that state really prints the sentence and whether doing what it
says really changes the answer, and only the pass on such a machine can say
either.

**`ios/verify/remedy.sh` — executed, and mutation-tested.** The remedy is
"install Xcode", which is not a command, so the executable form is the other
direction: an `xcrun` shim on PATH that answers `--sdk iphoneos
--show-sdk-path` with nothing and delegates everything else. The pass skips
with the sentence naming Xcode; the shim comes off; the app layer
type-checks. Checked for vacuity by changing the expected sentence, which
fails — and the failure prints the real skip, which is the right one.

The shim delegates rather than refusing everything because `swiftc` reaches
`xcrun` too, and a shim that refused all of it would strip far more than the
precondition.

**`android/verify/remedy.sh` — written, and it skips here.** Its fault is a
`GRADLE_USER_HOME` that has never been filled, and it walks the gate's own
order: the cold cache fails the classpath question, so the drill meets that
sentence, runs `:app:printVerifyClasspath`, is then failed by the *next*
question with the compiler sentence, runs `:app:fetchKotlinCompiler`, and
requires the pass to run. The second sentence is only reachable because the
first remedy worked, which is the drill's own evidence.

It needs an SDK and a network, so it takes its own skip on this machine and
runs in CI. Both drills are wired into `.github/workflows/ci.yml` — the
Android one **before** the step that warms the real cache, since the fault
it arranges is exactly the state that step exists to leave.

What is not drilled, and the script says so: the JVM harness gate's own
pass. Its sentence names the same `fetchKotlinCompiler`, so the command is
drilled; running that pass too would be arranging two machines at once.

---

## Item 5 — instrumented, not closed

This needs a machine resolving a serif that is neither Times nor Liberation
Serif, which is hardware. What *was* checked is the claim attached to it:
that such a machine gets the skip plus the table it needs.

Drilled by temporarily removing `Times` from `INK_CALIBRATED_ON`. The skip
fires with the right sentence and prints the calibration report — and the
brackets it measures for Times both clear their constants:

```
INK_ROW_ROUNDING is 0.15 …  over 10.0% and at or under 33.3%
INK_ASCENDER_SEPARATION is 0.43 …  over 12.8% and at or under 65.8%
```

So the remedy that skip names is executable and the report is usable. The
`WHICH DOES NOT SEPARATE` arm is still unreached, which is what the item
said it was.

---

## The file count, again

`TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn` failed on
the first full run with five sentences saying 441 against a tree of 450 —
nine new Go files. Two of the five are written across a line break, so a
`sed` fixed three and the other two needed the joined form. This is the arm
working: it fires on a commit that adds Go files and nothing else, and the
message says exactly what to edit.

Final state: `gofmt` clean, `go vet ./...` clean, `go test ./...` exit 0,
`wasm/verify/run.sh` OK (browser pass included, 13 checks), `ios/verify`
OK, `ios/verify/remedy.sh` OK.

---

## Next

1. **(new · value low) The Android remedy drill has never run.** It is
   syntax-checked, it takes its own skip correctly on a machine with no SDK,
   and every executable line of it is unexecuted — the fault it arranges
   needs an SDK, a network and a cold gradle cache. The first CI run on a
   branch that touches it is its first real test, and a spurious failure
   there costs a job rather than a release. Worth watching on the next push;
   worth reverting rather than debugging in place if it is noisy.
2. **(new · value low) `hero.png` is a picture of pictures and its parts
   are held twice.** The composite carries no strings of its own, on the
   argument that the three shots under it are already asserted. That is
   right today and stops being right the moment the hero is cropped, since a
   crop can remove text the parts still claim. Nothing reads the composite's
   own pixels.
3. **(carried · value low) A third face is still a skip.** Any machine
   resolving something other than Times or Liberation Serif gets the skip
   plus the calibration table, and this session confirmed both by drilling
   it: the table's brackets for Times clear their constants. What nobody has
   is a face where the readings do NOT clear — the report's "WHICH DOES NOT
   SEPARATE" arm has fired exactly once, on Liberation Serif under the old
   fractions, and is unreachable by any face this project has met.
4. **(carried · value low) Windowing as a proposal.** Declined on the
   payload in `non_goals.md`; `TestWhatWindowingWouldSave` asserts its own
   share and is the profile to re-take if a forty-lesson chapter ever ships.
   The device number it was once waiting on is `need_hardware.md`'s. Not a
   Next item any more except as this line.
5. **(moved) The four hardware items.** `ai_docs/plans/need_hardware.md`.
6. **(declined, non-goal)** Twenty-seven entries, unchanged. See
   `ai_docs/plans/non_goals.md`.
