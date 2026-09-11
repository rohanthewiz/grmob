# Session: the lever was a struct tag, and the two items left need hardware

Session: https://claude.ai/code/session_01AtctQFkHpNuADXAJrdFDJA
Date: 2026-09-11 · Previous:
`2026-0911-1810-two-metrics-that-printed-confident-nonsense-and-a-win-that-was-twenty-nine-percent.md`

## Ask

> work the Next list items in reasonable batches

Item 1 is done, and it did not end where it pointed. Items 2 and 3 both need a
physical device and neither is attached; nothing was done on them and nothing
could be.

The three sentences worth keeping if the rest is lost:

1. **The 2516ms was org.json, and the payload it was parsing was 92.4%
   zero-valued `core.Style` fields.** `json:",omitzero"` on `core.Style` and
   `core.Node` took the contents screen from 423,472 bytes to 53,408 and the
   Android cold launch from 4850ms to 3530ms — 27% of the whole launch, from
   struct tags.
2. **Windowing was the plan and the measurement said not to build it.** The
   item asked for sizing first; sizing found the cost was in a stage nobody had
   timed, and a probe killed the obvious alternative fix too — `JsonReader`
   consuming the same string is no faster than org.json.
3. **The same payload costs 6ms on the iOS simulator and cost 1666ms on the
   Android emulator.** That is a parser, not a machine, and it is why one host's
   lever was the view layer and the other's was the wire.

## Item 1 — Android's lever

### The instrument the item actually needed

`GrMobRuntime.start()` now takes three clock reads and `TreeStore.mount`
returns `MountStats`; `Startup.kt` prints one line per launch, gated on the
platform's own per-tag switch so a library does not narrate every launch of
every app built on it:

    adb shell setprop log.tag.GrMobStartup DEBUG
    android/device/launch.sh 5 --stages          # sets it for you

Three stages, because they respond to different fixes:

    bridge   Mobile.renderInitial() — Go's render, the marshal, and the
             gomobile crossing that copies the result into a Java String
    parse    org.json turning that string into JSONObjects
    build    GrMobNode.parse walking those into the snapshot-state tree

Plus `Process.getStartUptimeMillis()`, which is what makes the three a fraction
of something rather than three numbers in isolation.

### What it said

Five cold launches, the emulator from last session (`sdk_gphone64_arm64`,
1080x2400, Debug both halves):

    bridge   17 ms
    parse  1666 ms
    build   427 ms
             ────
            2110 ms of the 2516ms the four-arm A/B had attributed to the tree

**Neither Go nor the FFI is in it.** The JNI crossing of 423KB is a rounding
error, which retires the phrase "everything that happens to those bytes" that
three files were carrying.

### Two things the probe ruled out before anything was built

A temporary probe in the same process, reverted afterwards:

    org.json, passes 2 and 3      1113 / 1178 ms   so it is the parser, not a
                                                   cold JIT — the cold pass was
                                                   1.6s and the warm one 1.1s
    JsonReader, whole token walk   1398 / 1063 /   so a streaming parser buys
                                    943 ms         nothing; this is the floor
                                                   for one and it is not lower

So the payload was the lever, not the parser and not the protocol.

### The payload was mostly nothing

336 nodes, 423,472 bytes — 1,260 bytes a node. Of that:

    Style   391,406 bytes   92.4%
    Props    11,666          2.8%
    Type      2,087
    Key       1,250

`core.Style` has 58 fields and carried no json tags, so every node wrote all 58.
Across all 336 nodes there were **1,168 non-zero style fields** — three and a
half a node.

### The fix, and why `omitzero` rather than `omitempty`

`json:",omitzero"` on every field of `core.Style`, and on everything but `Type`
on `core.Node`. Not `omitempty`, because two of the fields are structs —
`Padding` and `Margin` are `EdgeInsets`, `AccessibilityValue` is a `ValueRange`
— and `omitempty` has never omitted an empty struct. `omitzero` (Go 1.24) does,
and it means the same thing for every other kind here, so one spelling covers
the struct instead of two.

**What made it safe was already true.** All three JSON hosts decode this struct
key by key with a zero default for an absent key — Kotlin's `optDouble(k, 0.0)`,
Swift's `?? 0`, the runtime's `style.FontSize ? … : ""`. "Missing" and "present
but zero" have meant the same thing to them since before the tags existed; the
tags just stopped writing the second one. The doc comments on both native
decoders that read *"because core.Style carries no json tags"* now say what the
tags are and why the equivalence is the contract they rest on.

`FlexShrink` is the one field whose zero is not its value, and it is safe for
the same reason it has always been odd: the distinction is carried by a
non-zero sentinel (`ShrinkNone`), not by presence.

### The reading

Same emulator, same hour, both arms with the stage clocks compiled in:

|                        | bytes   | parse   | build  | cold launch |
|------------------------|---------|---------|--------|-------------|
| every field written    | 423,472 | 1666 ms | 427 ms | 4850 ms     |
| zero fields omitted    |  53,408 |  249 ms | 185 ms | 3530 ms     |

**−1320ms, or 27% of the whole launch.** The screen's own cost over the
near-empty control is about 1000ms now rather than 2516ms.

(4850 rather than last session's 5045 for the before because both arms of *this*
A/B were measured with the clocks in and on a busier machine. Compare within a
table, not across them.)

Verified in the app, not only in the numbers: `ui.sh texts` on the contents
screen and on a lesson, a tap through a lesson and back, and a screenshot —
cards, shadows, borders, the badge, the progress bar and every inset intact.

### What windowing is now worth

It was Next item 1's proposal and it is still the only lever left on the tree,
but it is attacking ~450ms of parse-and-build inside a 3530ms launch rather
than 2110ms inside 4850ms, and it costs a protocol change (the host has to
report a visible range). That is the answer the item asked for: sized, and not
worth building yet.

## What fell out of it

### iOS gains nothing, and the reason is the interesting part

`LiveMapUITests` has recorded for some time that the same tree parses and builds
in **6ms** on the simulator. Android's org.json spent 1666ms on the identical
bytes. Two hundred times is not a device-speed difference; it is a parser, and
it is the whole reason iOS's lever was SwiftUI's view-per-node and Android's was
the wire. That line in the iOS test now says so, since it was previously read as
"already accounted for" and it was the load-bearing number for the other host.

### The two web targets disagreed, and now do not

Every scalar line in the runtime's `styleFromGrMob` was already blind to
present-versus-absent (`style.FontSize ? … : ""` answers `""` to both). Two
lines were not: an `EdgeInsets` full of zeros is truthy in JavaScript, so
`out.padding` and `out.margin` answered `"0px 0px 0px 0px"` where an absent one
answers `""`.

`htmlout` has always omitted the declaration for a zero inset
(`if s.Padding != (core.EdgeInsets{})`). So static HTML let a `<button>` keep the
UA's padding and the wasm runtime forced it to zero — a divergence between the
two web targets that nothing had noticed. They agree now, and there is a comment
at that site saying not to "fix" it back.

Checked in a browser rather than argued: every GrMob-rendered button across six
lessons carries an explicit inline `10px 16px` from the theme, so the UA default
never applies; the only controls with no inline padding are checkboxes, whose UA
padding is `0px`. Visually inert, and now consistent.

## Items 2 and 3 — not done, and not doable here

Both need hardware that is not attached. `adb devices` lists one emulator;
`xcrun xctrace list devices` lists the Mac and simulators.

- **Item 2** (do the emulator's numbers hold on a phone) is *more* worth asking
  now, not less: the whole finding rests on org.json being catastrophically slow
  on this emulator, and a phone's parse could be fast enough to change which
  lever matters. One device run settles it.
- **Item 3** (`HeadingSensor.retryIfArmed`) is unchanged from last session:
  `headingAvailable()` is false on the simulator, so the arm cannot be reached
  without a physical iPhone.

Nothing was faked and nothing was partially attempted.

## Verification

    go test ./...                    clean, every package
    gofmt -l .                       clean
    ios/verify/run.sh                clean
    wasm/verify/run.sh               clean
    android/verify/run.sh            clean
    xcodebuild -configuration Debug  BUILD SUCCEEDED
    xcodebuild -configuration Release BUILD SUCCEEDED
    gradlew :app:assembleDebug       BUILD SUCCESSFUL
    LiveMapUITests                   2 passed
    an emulator                      5 cold launches before, 5 after, plus
                                     3 exercising the new --stages flag and
                                     1 confirming the default run is unchanged
    the app on that emulator         contents screen, a lesson, a tap and back
    the wasm build in a browser      the contents screen, and the padding
                                     question asked of every form control on
                                     six lessons

## Next

1. **(carried · value medium, raised from low) `launch.sh`'s numbers are one
   emulator's, and now something rests on them.** Last session this was a
   footnote about absolute numbers. It is load-bearing now: the finding above is
   that org.json costs 1666ms where iOS's parser costs 6ms, and a physical
   phone's ART could be fast enough that the parse was never the largest stage
   there — which would not undo the fix (53KB is better than 423KB on any
   device) but would change what the *next* lever is. One device run, five cold
   launches, `--stages`.
2. **(carried · value low) `HeadingSensor.retryIfArmed` is unmeasured.** The
   compass has the arm, the teardown and the named message `LocationSensor`
   needed, and none of it has been observed: `headingAvailable()` is false on
   the simulator. A physical iPhone, one lesson, several taps in Settings.
3. **(new · value low) Windowing `core.List` over the bridge, re-sized.** Still
   the only lever left on the tree, now worth ~450ms of a 3530ms emulator launch
   rather than 2110ms of 4850ms, and still a protocol change. Re-size it after
   item 1 above, on the device — not before.
4. **(new · value low) 53% of the remaining payload is still `Style`.** 28,227
   bytes carrying 1,168 non-zero fields, which is ~24 bytes each and mostly the
   key names (`AccessibilityLabel` is 18 characters before its value). A short
   wire vocabulary would roughly halve it, and it would cost a mapping table in
   four renderers that the verbatim-Go-field-names convention exists to avoid.
   Probably not worth it; written down so the next person can decline it with
   the number in hand rather than re-deriving it.
5. **(declined, non-goal)** Twenty-five entries. See `ai_docs/plans/non_goals.md`.
