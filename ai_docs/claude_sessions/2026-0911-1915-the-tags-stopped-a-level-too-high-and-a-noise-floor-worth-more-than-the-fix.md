# Session: the tags stopped a level too high, and a noise floor worth more than the fix

Session: https://claude.ai/code/session_01AtctQFkHpNuADXAJrdFDJA
Date: 2026-09-11 · Previous:
`2026-0911-1842-the-lever-was-a-struct-tag-and-the-two-items-left-need-hardware.md`

## Ask

> Work 3 and 4

Item 4 (the wire vocabulary) is resolved: sized, declined with the number, and
a free 4% found on the way that the item had not asked about. Item 3
(windowing) is re-sized, and the re-sizing changed both sides of its ledger
enough that the item as written was pricing the wrong thing. Neither needed the
device the last two items are blocked on; the emulator was attached and did get
used, for something more useful than either.

The three sentences worth keeping if the rest is lost:

1. **`omitzero` had been applied one level too high.** Last session's tags made
   an all-zero `Padding` vanish, which hid that a *present* one still wrote all
   six of `core.EdgeInsets`' untagged ints. 51,242 bytes from 53,408 — and the
   reason it went unseen for two sessions is that the parent's tag was right
   there, working.
2. **The fix is invisible to the instrument that motivated it, and that is the
   session's most useful output.** Five cold launches each way: parse+build
   379.8ms → 377.9ms against run-to-run spreads of 17–52ms. So
   `android/device/launch.sh` now has a stated **noise floor of ~8ms, or a 4%
   payload change** — which is what makes the next payload claim checkable.
3. **Item 4's proposal is dominated by item 3's, and neither was priced
   right.** A short wire vocabulary is 17–28% of the payload for a 58-entry
   table in four renderers. Windowing is 66–76% for no change to how a field is
   spelled — and the "protocol change" it was priced with already exists.

## Item 4 — the wire vocabulary

### What the measurement found before it answered the question

Dumped the contents screen and accounted for every byte. The answer to the
item's question was there, but so was something the item had not asked:

    EdgeInsets.Horizontal    77 × 15 bytes   1,155    every one of them zero
    EdgeInsets.Vertical      77 × 13 bytes   1,001    every one of them zero
    ValueRange.Text           1 × 10 bytes      10
                                             ─────
                                             2,166    4.0% of the screen

`core.EdgeInsets` has six untagged ints. `Style.Padding` and `Style.Margin` are
themselves `omitzero`, so an inset that is entirely zero never crosses — but
the 77 that *do* cross wrote all six fields. And the axis pair is zero in every
one of them, because the DSL's side props settle the shorthand into the sides
before writing (`core/padding_sides.go`). The two fields that exist to be a
shorthand were pure overhead on this screen.

### Why it was safe, and why no renderer changed

Stronger than the argument the Style tags rest on. All four renderers resolve a
side by asking whether it is non-zero:

    top = Top != 0 ? Top : Vertical     htmlout.edgeSide
                                        GrMobStyle.kt    parseEdges
                                        GrMobStyle.swift parseEdges
                                        grmob-runtime.js edgeToCSS

A zero side is *already defined* to mean "unset, take the axis" — that lossiness
is documented at length in `htmlout/edges.go` and is deliberate. So a zero field
was never carrying information, and each of the three JSON readers turns a
missing key back into 0 (`optInt(name, 0)`, `?? 0`, `explicit || 0`). htmlout
walks the Go tree and never sees JSON.

Nothing changed in any renderer. The four decoders now each carry a note saying
what the absent key means and warning off the "fix" of trying to recover a
distinction Go no longer sends.

`core.ValueRange`'s four strings got the same tags for ten bytes, which is the
rule and not the bytes.

### The rule is now enforced rather than remembered

`core/wire_omitzero_test.go` — `TestEveryWireFieldOmitsZero` walks every struct
reachable from `core.Node` by reflection and fails any exported field without
`,omitzero`. One named exception (`Node.Type`, which must be written even when
absent would parse). Verified it bites: stripping one tag fails with the field
path that reached it.

It also guards itself — if the walk stops reaching `Style`, `EdgeInsets` or
`ValueRange`, it says so, because every assertion above would otherwise pass
vacuously.

This is the specific failure mode worth catching: the outer tag is visible,
it is doing something, and it hides that the inner fields are not.

### The A/B, which found nothing, usefully

Five cold launches each arm, same emulator, an hour after last session's:

|              | bytes  | parse           | build           | launch        |
|--------------|--------|-----------------|-----------------|---------------|
| untagged     | 53,408 | 201.3 ±16.8 ms  | 178.4 ±51.9 ms  | 3464 ±411 ms  |
| tagged       | 51,242 | 197.4 ±31.1 ms  | 180.5 ± 8.7 ms  | 3233 ±186 ms  |

Parse+build moves **1.9ms** against spreads of 17–52ms. A 4% cut predicts ~8ms
if the parse is linear in length, and 8ms is under the floor.

The 231ms in the launch column is **not** the tags. The untagged arm's five runs
fall 4000, 3764, 3394, 3106, 3058 — that is a machine warming up, and reading it
as a win would have been the easiest mistake available this session.

So the change is in the tree for the bytes and for the rule, and the honest
statement of it is written at all five sites that carry these numbers rather
than the flattering one.

**What that buys is the floor.** `android/device/launch.sh` and its README now
say: five launches on this emulator cannot resolve a payload change smaller
than about 8ms, or 4% of this screen. That is worth more than the 2KB — it is
what stops the next 4% being reported as a win.

### The vocabulary itself — sized, and declined

Of the 51,242 bytes now, **54.1% are key names**:

    Style field names       16,261    31.7%    1,479 fields
    Node field names         8,826    17.2%    Type/Style/Props/Children/Key
    Props key names          2,640     5.2%    "content" ×215, "onClick" ×49
                            ──────
                            27,727    54.1%

Two-character codes: recoding `Style` alone saves 8,866 bytes (17.3%); all three
groups saves 14,552 (28.4%). Against the 378ms of parse-and-build that emulator
measures, **65ms and 107ms** of a ~3,200ms launch — 2% and 3%.

Declined, and the number is only half of it. Verbatim Go field names are
load-bearing: they are why `core.Role`'s ARIA spellings need no mapping table on
either DOM target, why a tree dumped from the bridge is readable, and why
`app_test.go`'s `nodeStyle`, `wasm/verify` and `ios/verify` each decode the half
they care about without a shared schema. A vocabulary puts a 58-entry table in
three readers and a writer and puts every one of those tools behind it.

And it is dominated — see below. Written into the note above `core.Style`, where
the next person lands.

## Item 3 — windowing, re-sized

The item said "re-size it, on the device, not before." The millisecond half
needs a device. The **byte** half does not, and it is decisive.

`TestWhatWindowingWouldSave` in `examples/tutorial` prints the profile so it
moves with the content instead of living in a session doc:

    n   key          type      bytes   cumulative      sent    % sent
    1   title        Column      459          459       681     1.3%
    2   progress     Card        696        1,155     1,377     2.7%
    3   chapter-0    Card      5,011        6,166     6,388    12.5%
    4   chapter-1    Card      5,959       12,125    12,347    24.1%
    5   chapter-2    Card      5,092       17,217    17,439    34.0%
    …
    10  chapter-7    Card      5,301       51,020    51,242   100.0%

### Three things this changes about the item

**The granularity is the chapter, not the lesson.** The List has **ten**
children, not the 49 rows `home.go`'s comment counts — a title, a progress card,
and eight chapter Cards of 5–12KB, with the lesson rows nested *inside* the
cards. So a window cannot be "the rows that fit".

Where the fold actually falls was read off a screenshot rather than assumed: on
1080x2400 it lands inside the fourth child — title, progress, all of chapter-0's
card, and the header of chapter-1's. So the honest window is n=4 (24.1%) and one
card of overscan is n=5 (34.0%). **Windowing is worth 66–76% of the bytes.**

**The protocol change it was priced with already exists.** The item assumed a
new bridge surface for the host to report a visible range. `core.OnHostEvent` /
`mobile.ReportHostEvent` is already a generic host→app channel carrying a name
and a JSON payload, already returns the following pass's patches on the event
path, and is already serialized with render passes by the manager. A visible
range is one more event name. The gomobile surface does not grow.

**What it actually costs is two things nobody had named.** The bootstrap — a
cold launch has no visible range, because the host cannot lay out what it has
not received, so the first render guesses and is corrected. And worse, the
**scroll extent**: a LazyColumn sent four children believes there are four, so
the scrollbar is wrong and the scroll stops short. The honest design is
placeholder children (right key, estimated height, ~40 bytes) rather than absent
ones — which keeps the extent right and still drops ~70%. That is a shape the
measurement supports and the original framing did not describe.

### Why it is still not built

Because the only reading that prices it is that emulator's, whose org.json
spends ~200ms on 51KB where iOS's parser spends **6ms** on the same tree. If a
phone's ART parses this screen in 30ms, a 76% cut is worth 23ms and the
placeholder machinery is not worth owning in four renderers. That is item 1's
open question, unchanged, and this table is what to multiply its answer by.

What is *not* in the way is the instrument. A 66–76% cut is ~250–290ms of
parse-and-build on that emulator — thirty times the 8ms floor this session
established. Whatever a phone says, windowing is the one remaining lever on this
screen whose effect five launches would report unambiguously.

## Verification

    go test ./...                     clean, every package
    go vet ./...                      clean
    gofmt -l .                        clean
    ios/verify/run.sh                 clean
    wasm/verify/run.sh                clean (drives headless Chrome)
    android/verify/run.sh             clean
    xcodebuild -configuration Debug   BUILD SUCCEEDED
    xcodebuild -configuration Release BUILD SUCCEEDED
    gradlew :app:assembleDebug        BUILD SUCCESSFUL
    the emulator                      10 cold launches, 5 per arm, --stages
    the app on that emulator          contents screen and a lesson, screenshotted
                                      — every inset, card, code block, chip and
                                      stepper intact
    TestEveryWireFieldOmitsZero       verified to fail when a tag is removed

Five stale "412 tracked Go files" sentences in `wasm/verify` updated to 413, as
that suite's own arm asked.

## Next

1. **(carried · value medium) `launch.sh`'s numbers are one emulator's, and two
   things now rest on them.** Unchanged from last session except that it has
   gained a second dependent: the windowing profile above is a byte fraction
   that only becomes a decision when multiplied by a device's ms-per-byte. One
   device, five cold launches, `--stages`. The emulator↔simulator gap is 200x on
   the same bytes, so this is not a formality.
2. **(carried · value low) `HeadingSensor.retryIfArmed` is unmeasured.**
   Unchanged. `headingAvailable()` is false on the simulator; needs a physical
   iPhone, one lesson, several taps in Settings.
3. **(carried, re-sized · value low→medium) Windowing `core.List` over the
   bridge.** Re-sized above and materially different from how it was written:
   worth 66–76% of the payload rather than an unsplit "~450ms", needs no new
   bridge surface, and costs a bootstrap guess plus placeholder children to keep
   the scroll extent honest. Still gated on item 1. Raised to medium because it
   is now the only lever left that the emulator could measure unambiguously.
4. **(new · value low) The contents screen's eight chapter cards are all
   expanded, and nothing asked for that.** 97.4% of the payload is inside them.
   Collapsing them by default would take more off the wire than windowing, cost
   no protocol change at all, and is a UX decision about the tutorial rather than
   a framework one — which is exactly why it is written here rather than done.
   Worth one sentence from whoever owns the tutorial's shape.
5. **(closed) A short wire vocabulary.** Sized at 17–28% for a 58-entry table in
   four renderers and declined on the note above `core.Style`, with the numbers
   in hand. Dominated by item 3 at 66–76% for no mapping table. Reopen only if
   the key-name fraction changes shape — it is 54.1% of the payload today.
6. **(declined, non-goal)** Twenty-five entries. See `ai_docs/plans/non_goals.md`.
