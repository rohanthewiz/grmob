# Session: a premise priced in pixels, a probe that asked at the wrong size, and a lexer that knew which construct

Session: https://claude.ai/code/session_01MQcEV7mAeEV4h1LywCdQhe
Date: 2026-09-08 (follows "a-canary-nobody-priced-a-generic-nobody-asked-and-a-copy-placed-by-the-wrong-unit")

## Ask

"Do all items in the Next list." Six items, all raised last session, all
closed. Two stages, each verified over ten paths and broken on purpose.

| stage | items | where |
|---|---|---|
| A | 1, 2, 3 | browser.mjs, gen.go, a new inkglyph_test.go |
| B | 4, 5, 6 | themenearmiss, checknumbering, pinfixture |

---

## Stage A — the canary, and a fold that only one end took

### Item 1 · a premise decided by a name

`inkCanvasFallbackFault` asked whether the generic resolved to a face this
grid draws with by comparing the family STRINGS `CSS.getPlatformFontsForNode`
returned. One API, one page, so they agree today — and a platform reporting
one face under two spellings ("Times" for the run, "Times New Roman" for the
generic's node) makes the comparison miss. Silent direction, reached from
inside the reader that exists to close it.

The metric answer was already at the counter. What breaks the canary is not
two nodes sharing a NAME, it is two requests producing the same ADVANCE:

    inkCanvasGenericGap(m)   |named − base|, one canvas, one evaluate, this
                             run's own string at this run's own size

Inside a LayoutUnit the canary has nothing to read. That bound rather than
`INK_CANARY_MARGIN`, because LAYOUT_UNIT is `inkCanvasNameReached`'s own
decision bound — the runs that DID separate are bracketed by the margin in the
census, and the two populations are complements. The name read survives as the
sentence's noun and nothing else; `inkCanaryFaceList` renders it.

The tail no longer claims "which is not a face any of these runs is drawn by".
It recites `asked.canvasGenericNearest`: 1.2246px on this grid, through the
same helper the arm decides with.

    the unread-premise arm  now fires when no joined run produced the pair,
                            which is by construction unreachable — kept as the
                            reader that would fire if that stopped being true,
                            the way inkCanvasFaceFault's `!m.asked` arm is

**Break-tests.** `INK_CANARY_FALLBACK = "Times"`: *"Times measures like the
face 28 of this grid's 28 joined runs are drawn by — no more than 0.000000px
apart"*, with every per-run join reporting silent underneath. Then the same
with the probe's read renamed to "Times New Roman", so one face arrives under
two names: still reported. That second one is the case the string comparison
was missing.

### Item 2 · a probe mounted without a size

One span carrying `font-family` and nothing else, resolving the generic at
whatever the page default was, while every run is measured at its own computed
style. A stack that maps `monospace` differently by size — an optical cut, a
display face — had the probe naming a face no canary is asked at.

The requests are now taken off the joined runs and deduplicated, one probe per
`style|weight|size`, each mounted with that whole `font:` shorthand and the
generic as its family — the same four parts `INK_FONT_SHORTHAND_JS` assembles,
so the two are asking one question. `canvasRuns` moved above the mount to
supply them.

    this grid   normal 400/700 × 12px, 13px, 19px — six requests
    before      one, at the page default: 16px, weight 400, asked by nobody

### Item 3 · a refusal that was ASCII-shaped while the note folded

`inkLigatureNote` folds; `inkGlyphPerCharacter` matched raw bytes and leaned
on the printable-ASCII arm to make that safe. The lean made the REASON wrong —
`"aﬁe"` was refused for holding an invisible character rather than for holding
`"fi"` — and one step further out it fails outright.

`inkGlyphFold` now folds both sides in gen.go, and the pair arm is asked
FIRST, because both arms are true of a precomposed ligature and only one names
what the author wrote.

    inkLigatureForms    U+FB00..FB06 and the long s, from a table — NFKD's
                        answer for the seven characters the seed list is about,
                        written down because this module carries no normaliser
    inkGlyphIgnorable   the same ranges as INK_FOLD_IGNORABLE, spelled as a
                        predicate rather than taken from unicode.Cf, whose
                        membership has moved between versions

Three new tests in `wasm/verify/inkglyph_test.go`. The last one lifts
browser.mjs's real `INK_FOLD_IGNORABLE` and `inkFold` out of the file and runs
them under node — which is what holds the hand-written table against an actual
NFKD, and skips with a sentence when there is no node.

One thing found while writing it: this build's NFKD returns `"st"` for U+FB05
outright, not the `"ſt"` UnicodeData's `<compat> 017F 0074` predicts. The
long-s step is kept in both folds anyway and both comments now say why — a
normaliser that followed the file would leave the long s, and neither
`strings.ToLower` nor `toLowerCase` finishes the job, because that is a case
FOLDING and they are case mappings.

**Break-tests — 4 run, every one failed as it had to**

    ASCII arm first again        "'ﬀ' is outside printable ASCII" — the wrong
                                 suspect, named by the test
    U+FB03 out of the table      the block census
    0x200F off the range         "U+200F is dropped by INK_FOLD_IGNORABLE and
                                 kept by inkGlyphIgnorable"
    gen.go's U+017F removed      "gen.go says "ſ" and browser.mjs says "s""

---

## Stage B

### Item 4 · one floor for four endings 27 to 473 apart

Ten is a bracket under 27 and nothing at all under 473. So the floor moves
with the ending, in INK_OWN_MEASURED_ON's shape: a number taken against ONE
build, recorded as such, and the bracket derived from it.

    affordedMeasuredOn   27 / 473 / 377 / 53, over 80 distinct leaf names at
                         a window of 6
    affordedEndingShare  a third — what separates a population that MOVED
                         (proportional, small) from an arm that went
    affordedMeasuredNote inkOwnBuildNote's clause: the leaf count then and
                         now, so a struct that changed size does not read as
                         a relation coming apart

A reworded ending would find nothing in the map and fall back to the constant
with nobody deciding that, so both directions of the key set are asserted.

**Break-test.** The window narrowed:

    width 5   passes
    width 4   14 of 628 against a floor of 17 — silent before
    width 3   4 of 474 (the old arm) AND 69 of 474 against 125, which is the
              377-ending losing four fifths of itself with nothing to say
    renamed   both arms: no reading for the old sentence, a reading for an
              ending the derivation does not have

### Item 5 · a lender chosen alphabetically

Two exemptions share `.go` and their arguments are nothing alike — one is a
development-mode audit with numbered sections of its own, the other a test
file that quotes citations as examples. The recital named whichever sorted
first, and the `why` that tells them apart was in hand and read by nothing.

`exemptKinds` now carries every reached row of the kind, sorted for the same
stability the single lender was picked for, and the line prints each with its
reason. The reader judges which fits; nothing here pretends to.

**Break-test.** A `README.md` row: 358 → 378, and *"That is the kind one row is
exempted under: README.md, because …"*.

### Item 6 · a shadowed arm lumping three constructs

"A literal, a comment trailing an assertion, or the runaway that blanks the
rest of its own line" — three places to send a reader, one sentence, and the
lexer had already decided. `pinCodeOnly` is a switch with a case per construct
and returned none of it.

    third return   one byte per source byte at the source's own offsets, so a
                   copy's span is a slice of it
    pinBlankKind   'l' 'b' 's' 'r', with 'r' written twice: once as the
                   literal it looked like, and again from its opening quote
                   when the newline closes it — the moment the lexer learns
                   which of the two it was
    pinCopyNote    grouped by construct, so two copies a literal swallowed are
                   one clause and a literal beside a block comment are two

**Break-tests.** The runaway rewrite removed: the note calls it *"a string
literal"* and sends the reader after a quote somebody wrote on purpose. The
construct dropped from the clause: the literal row fails. The fixture gained a
fourth line (a block comment after code, giving the two-construct clause) and
a separate one-line runaway fixture, which is the first thing here that
exercises 'r' at all.

---

## Verification

Ten paths, green, on both stages:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + mjs suites + BROWSER PASS (rc 0)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

Six files, +934 −193 plus one new test file:

    stage A  wasm/verify/browser.mjs           items 1, 2
             wasm/verify/gen.go                item 3
             wasm/verify/inkglyph_test.go      item 3, new, 306 lines
    stage B  wasm/verify/themenearmiss_test.go item 4
             wasm/verify/checknumbering_test.go item 5
             internal/pinfixture/pinfixture_test.go item 6

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) One share for four endings, which is the shape
   complaint one level up.** `affordedEndingShare` is a third and it is a third
   of each ending separately — better than one constant, and still one fraction
   chosen once. INK_OWN_SHAPE_FLOOR's own note says why that is not free: "a
   fraction is still a fraction of one population", and what makes its quarter
   a measurement rather than a taste is the pair of edges it has to separate,
   both in hand on every run. Here nothing names the edges. A third is under
   every ending today and nobody has said what it is above — an ending at 27
   is bracketed by the constant instead, and no reading says which of the four
   the share is actually doing work for.
2. **(age 0 · value medium) A copy straddling two constructs is named by the
   first one.** `pinBlankedBy` returns the first recorded kind in the span, and
   a phrase half inside a string literal and half in the comment trailing it is
   two constructs in one copy — the exact shape the arm was rebuilt to stop
   summarising, one level further in. `pinShadowConstructs` can only see one
   kind per copy, so the two-construct clause cannot describe a straddle at all.
   The bytes are all there; nothing reads past the first.
3. **(age 0 · value low) The generic's face is read six times for a noun.**
   `inkCanaryFaceList` costs a DOM.querySelector and a
   CSS.getPlatformFontsForNode per distinct request — six on this grid — and
   the decision no longer uses any of it. That is a round trip per size for a
   sentence's illustration, inside the window two fingerprints are holding
   open. Either it is worth naming the face at every size a canary is asked at,
   in which case something should say what a DIFFERENCE between those answers
   means, or one read would do.
4. **(age 0 · value low) The two probes' answers are never compared.** The
   mounted probe says which face the generic reached at a request; the metric
   gap says how far that face is from the run's. Nothing checks they agree — a
   probe naming Menlo at 12px while the advances say the generic measures like
   Times at 12px is two readings of one question disagreeing, and both would
   pass. The pair is the same third-party-versus-metric shape inkFaceList's own
   note is about, with nobody holding it.
5. **(age 0 · value low) gen.go's fold is narrower than NFKD and the gap is
   documented, not measured.** `TestTheTwoFoldsAgreeOnTheFormsTheyBothCarry`
   asks only about inputs both sides claim — ASCII, the seven forms, the long
   s, the format characters. An accented letter, a combining mark, a fullwidth
   Latin letter: NFKD decomposes them and `inkGlyphFold` does not, and the
   printable-ASCII arm is what makes that safe. That is the same lean item 3
   removed from the PAIR question, still holding up everything else, with no
   reading of how wide the gap is.
6. **(age 0 · value low) The recital prints every `why` in full.** Item 5's
   line grows with the number of exemptions sharing a kind and is already ~600
   characters at two. That is the right direction — the argument is what tells
   the rows apart — and it has no bound. A third .go exemption makes a passing
   run's log a paragraph, and the thing a reader actually wants is the one row
   whose argument fits the example, which is still their job to work out.
