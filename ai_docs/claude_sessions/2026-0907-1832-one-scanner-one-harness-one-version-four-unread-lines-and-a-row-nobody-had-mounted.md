# Session: one scanner, one harness, one version, four unread lines, and a Row nobody had mounted

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "two-numberings-of-one-sequence-three-controls-nobody-could-delete-and-a-refusal-that-had-stopped-applying")

## Ask

"take the oldest 5 items in the Next list".

Three duplications and two unread readings. Four of the five were exactly what
their entries said; the fifth's proposed subject turned out to be inert, which is
the second session running that a break-test has corrected the item rather than
confirming it.

| # | age·value | item |
|---|---|---|
| 1 | 3·low | `maskNonCode` and `matchingBrace` answer the same question twice |
| 2 | 3·low | The two `gate_test.sh` scripts share a shape and no code |
| 3 | 2·low | `ext.composeLayoutVersion` is a second spelling of a derived fact |
| 4 | 1·high | The Compose transcription is the one link in the pin's chain nothing checks |
| 5 | 1·medium | The pinned Row is not mounted in a browser |

## Item 1 — the counting moved onto the mask

Two files, four arms each, same order, same words. `maskNonCode` blanks comments
and literals; `matchingBrace` skipped them while counting braces. Each carried a
comment asking the other to agree.

The merge is one line of insight: **blanking preserves offsets and length**, so
counting braces *over the mask* is counting braces in code.

```
   maskNonCode      the four not-code arms          ← one answer
   matchingBrace    the two brace arms, over it     ← the part never shared
   returns          src[1:i], not code[1:i]         ← callers read literals
```

The one thing the counter needed that a mask did not was the loud refusal of an
unterminated `/*` or `"""`. That is a second return now, which turned a
`t.Fatalf` buried in a scanner into a value — so the fault is reachable by
handing over two strings instead of by owning a source file with it in.

`TestEverySwiftTypeCutComesBackBalanced` was quietly a *fourth* answer
(`stringLiteral.ReplaceAllString(stripLineComments(body))`, which misses block
comments and `"""`); it counts over the mask now, so the cut is checked against
its own idea of code rather than against a different one.

The new test says the agreement holds. Each row hides the same two things in the
same construct — a `}` the counter must not count, an anchor the mask must not
find. The multi-line row needed an **unpaired quote** to discriminate: without
one, a scanner that has lost its `"""` arm pairs the delimiters off two at a time
and blanks the content by accident.

## Item 2 — a harness, and a harness for the harness

`internal/gateharness/harness.sh` now holds the count, the prefix match,
`gate_distinct` and the FAIL/OK footer. Each `gate_test.sh` keeps its table,
which is the only part that is about iOS or about the JVM.

The interesting half is one layer out. A shared helper that silently stopped
counting would take *both* gate tests green with it — strictly worse than the two
untested copies it replaced, and the exact regress `gate.sh` exists to end. So
`harness_test.sh` runs each helper in a subshell, judges it by what it printed
and returned, and both `run.sh` run it before their own gate.

## Item 3 — there is one version now

`ext.composeLayoutVersion = '1.6.8'` is gone. The Compose BOM sits on the
`composeLayoutSources` configuration and resolves the sources jar the way it
resolves every other Compose artifact in the file.

Three details make it work, and **each was the previous arrangement's stated
reason for believing it impossible**:

```
   the platform on THIS configuration    a BOM constrains where it is declared
   transitive = false on the DEPENDENCY  on the configuration it also drops
                                         the BOM's constraints — which is what
                                         made this look impossible
   an artifact block, not `:sources@jar` artifact-only notation skips the
                                         metadata the BOM constrains
```

The check that remains is narrow on purpose. Take the platform off, or move
`transitive = false`, or go back to `@jar`, and gradle fails immediately and says
why. Writing a version back into the coordinate is the *one* change that is
silent: it resolves perfectly and is held to nothing. That is what
`TestTheComposeSourcesTakeTheirVersionFromTheBOM` asserts — reading the gradle
file through `maskComments`, because the paragraph explaining the change quotes
the declaration it replaced.

## Item 4 — the chain had a link drawn in

`internal/pinfixture`'s header lists the chain each transcribed branch hangs
from, and it was accurate about everything except itself: **only
`mainAxisMax - fixedSpace` was read out of the sources jar.** The rest were
quoted in a Go comment.

Six readings now, up from two:

```
   the floor              (mainAxisMax - fixedSpace).coerceAtLeast(0).toInt()
   the cleared minimum    mainAxisMin = 0,
   the elided branch      if (mainAxisMax == Constraints.Infinity)
   the gap clamp          spaceAfterLastNoWeight = min(…)
   the trailing gap       fixedSpace -= spaceAfterLastNoWeight
   the missing bound      no mainAxisMax, no coerceAtMost   ← a notWant
   the unclamped report   return layout(placeable.width, placeable.height)
```

The transcription's own quote had silently dropped the `Constraints.Infinity`
arm. It is quoted whole now and the elision is named — a quote with a branch
taken out of it is how a reader ends up believing the else arm is the whole rule.

The `notWant` is the interesting row: an absence cannot be a substring, so it is
a claim about a *window*, and the window has to be drawn honestly. The first one
was sized to the expression (142 bytes) and was useless — a clamp appended lands
one byte past the end of what it was measuring, and the break-test that appended
one passed. 220, which is the expression plus room for the shortest plausible
spelling of the thing it refuses.

## Item 5 — the reasoning became a measurement

The pin census pairs a Compose column with a CSS one, and the CSS column was
`GrMobFlexSolver`'s. Check 9 had just caught that solver disagreeing with a real
Chrome under overflow when a child has padding — and the fixture's answer was
"these children have none, so the two rules coincide". True, and asked of nobody.

Check 12 mounts the four Rows. A browser lays them out at

```
   no pin       24, 80, 16      pin middle   0, 200, 0
   pin first   200,  0,  0      pin last     0,   0, 200
```

which is the census's CSS column exactly. It **recomputes no flex line** — that
would make the check about whether two transcriptions agree — so every claim is
one the fixture states, held against pixels: the pinned child keeps its base,
the extents match Compose precisely where `mainsAgreeWithCSS` says they do, and a
child's width does not depend on where it sits. Those three pin three of the four
rows to their exact numbers.

### The subject that was not one

`pinTree` carries `min-width: 0`, and the item's shape said that was its control.
A break-test removed it and **nothing moved**: CSS's automatic minimum is
min(specified size, content size) and these children are empty boxes, so it is
already 0. It is bandTree's *control* the declaration matters to, because that
child wraps a label.

So it is kept as intent, named as inert, and `controls_test.go` does not pin it —
a control with no subject is the shape that file exists to refuse. What it pins
instead is arithmetic over the fixture: that both an agreeing and a diverging case
exist, which is what check 12's own both-arms guard is about.

### And a stale address the mechanism could not see

`htmlout/fixedsize_test.go` cited the fixed-size check as `check 10` in two
places after it became 11. It was outside `checknumbering_test.go`'s hand-listed
file set; it and `controls_test.go` are in it now.

Worth being exact about what that bought, because a break-test said so: the scan
checks that a citation is **in range**, and `check 10` is in range. A reader found
this one. The list now catches a citation to a check that does not exist, which is
what a renumbering that shortens the sequence produces, and the header says so
rather than implying more.

## The break-tests

**27 run, 27 caught.**

    item 1  a lost `"""` arm, a lost `//` arm, the mask returned instead
            of the source, the unterminated report dropped                    4
    item 2  gate_expect stops counting, gate_verdict exits 0, gate_distinct
            stops noticing, and both real gates regressed through the
            shared harness                                                    5
    item 3  a version back in the coordinate, the BOM off the configuration,
            ext.composeLayoutVersion declared again                           3
    item 4  six mutations of a sources jar in a fake gradle cache             6
    item 5  the flex minimum removed, the pin spelled as a literal zero,
            a width that drifts with position, the fixture claiming
            agreement everywhere and nowhere, a floor on the children,
            two out-of-range citations                                        9

Item 1's line-comment mutation is the one worth noting: **one edit failed four
tests across three consumers**, including the new gradle check, which is what
being one scanner means.

Item 4's six needed a mutated sources jar, so the checks were run against a fake
`GRADLE_USER_HOME` holding a rewritten copy — which also exercised the cache
lookup. Two of them found real weaknesses rather than confirming strength: the
`notWant` window was too tight, and `go test` caching had to be defeated with
`-count=1` before any of them meant anything.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       THE GATE HARNESS + gate + flex + stack + 3 bands
                            + 4 pinned Rows + 15 menus + replay + view
                            + importer + app
    android/verify/run.sh   THE GATE HARNESS + gate + the Compose census's
                            source half + 15 picker menus + 22 value ranges
    wasm/verify/run.sh      replay + 19 mjs suites + BROWSER PASS (12 checks)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    internal/gateharness/harness.sh       the shape both gate tests are
    internal/gateharness/harness_test.sh  and its own arms, reached

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 2 · value medium) The pin census table's CSS column is unheld.**
   `TestTheCensusTableIsTheFixture` compares only the Compose column. Two things
   now produce the CSS one — `ios/verify` computes it and check 12 measures it —
   and the table in `internal/pinfixture`'s header is still prose beside them. The
   control row's `24, 80, 16` is the sharpest case: it is the one row nothing
   derives from a stated claim, so a reader who trusts those three numbers is
   trusting a comment.
2. **(age 2 · value medium) Which mask a check gets is decided per file, not per
   question.** `codeOf` is used by `shrink_test.go` and nowhere else, yet "does
   the renderer call this" is what most of `mobile/verify` asks — so a string
   literal naming a call still satisfies those checks. The scanner is one now;
   which of its two levels a call site wants is still a decision nobody makes
   deliberately.
3. **(age 2 · value low) `GRMOB_COMPOSE_SOURCES=required` is exercised by no
   pass.** The arm that turns a skip into a failure runs only in its own unit
   test.
4. **(age 2 · value medium) The importer reading runs only where Swift does.**
   `mobile/verify` holds the two type tables to what `importer.swift` *says*;
   `swiftc` is what holds that file to the truth, and it runs in `ios/verify` — a
   Mac. On any other machine the pairing is verified and the reading behind it is
   not, silently.
5. **(age 2 · value medium) `[]byte`'s refusal is now answerable and still
   stands.** Its stated reason is that no golden exercises `([]byte, error)` —
   but `NSData*` is `_Nullable` in gobind's own golden, so it takes the same arm
   `string` does and stays the return rather than moving into an out-pointer.
   That is one reading away from being a row, and the refusal text still sends the
   reader to a Mac.
6. **(age 2 · value low) `checknumbering_test.go` scans a hand-listed set of
   files for citations.** Eight paths now, written out, and the two added this
   session were added because somebody noticed rather than because anything
   pointed at them. Walking the repository would close it, and would have to
   decide what to do about `core/debug.go`, whose "Check 1..3" are its own
   sections.
7. **(age 2 · value low) The band grid samples one pixel per band.** The fill,
   and nothing else. The widget grid reads a fill *and* both boundary edges,
   because either single edge is consistent with a correct frame; a band's label
   ink and its badge are unread, so a band painting its fill over an invisible
   label would pass.
8. **(age 2 · value low) The band grid's fold guard cannot be reached.** Nine
   bands fit the viewport comfortably, and the only way to trip the guard is to
   add themes. It is written out and stated rather than hidden, but nothing has
   ever executed it.
9. **(age 0 · value medium) Nothing measures the gap collapse.** The census says
   `spaceAfterLastNoWeight` is `min(spacing, what is left)`, so an overflowing Row
   inserts no spacing after the child that spent it. That line is read out of the
   jar and `MeasureCompose` implements it — and every case in `internal/pinfixture`
   has `gap: 0`, so no cross-target comparison has ever reached it. One case with
   a gap would put a browser and the solver behind a sentence three documents
   repeat.
10. **(age 0 · value low) The gate harness's self-test never runs under `go test
    ./...`.** It runs from the two `run.sh`, which is where its subject is used —
    but that is the same shape as item 4 above, and `checknumbering_test.go` is
    the precedent for the other answer: a check about a shell file, kept where
    anyone with a Go toolchain runs it.
11. **(age 0 · value low) `regionOf` cuts `browser.mjs` between two anchors.**
    A coarse cut into a JavaScript file, which is what `swiftTypeBody` was before
    it got a scanner — and this session's whole first item was about not having a
    second answer to "where does this declaration end". It is defensible (what is
    counted inside it is one exact literal) and it is stated, but a `MinWidth: "0"`
    added to whatever declaration sits between the anchors would be counted as
    `bandTree`'s.
12. **(age 0 · value low) The claims table's windows are byte counts measured
    against androidx's current formatting.** 1500, 200, 220, 1600 — each derived
    by measuring the file as it is shipped today. A reformat of
    `RowColumnMeasurementHelper.kt` that moved a line moves them, and the
    `notWant` row is the one where being wrong is silent in the safe direction:
    too tight a window finds no clamp and passes.
13. **(age 0 · value low) `composeLayoutVersion` re-derives what gradle
    resolves.** It reads the BOM's pom the way gradle does. If the two ever
    disagreed — a different entry winning, a platform enforced somewhere else —
    the Go side would look for a jar that is not there and *skip*, which is the
    quiet answer rather than the loud one.
