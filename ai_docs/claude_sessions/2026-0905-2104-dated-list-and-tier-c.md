# Session: a list that knows its own age, a skill to rebuild it, and the oldest item on it

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (follows "counted-marks-and-a-wrong-premise")

## Ask

Three, in order, each growing out of the last:

1. "Add a session age param to each item in the Next list and a value param of
   (high, low, or medium)."
2. "Add the ability to sort by value, but by default sort by age. Then let's
   capture this into a global level skill." — then, mid-flight: "The idea is to
   say build a next list from the most recent \<n\> session docs [order by value]."
3. "Now do the oldest item on the Next list."

The third ask is the first two paying for themselves within the hour.

## Part 1 — dating the list

Age is derived, not asserted: grep all 88 session docs for each item's first
appearance, match on the distinctive *symbol* rather than on prose (the wording
drifts between sessions; `AccessibilityNestingLevel` does not), and count
session docs back from the newest.

Two findings the arithmetic turned up on its own:

- **Item 2's age was a lie waiting to happen.** The "on-light tone per palette
  role" item was raised in `2026-0831-1515-button-variants`, dropped off the
  list entirely, and reappeared 62 sessions later labelled "newly raised". Dated
  from the re-raise (that is the live thread) with the history written into the
  item, because a lapse is evidence the item is real *and* evidence the list
  leaks.
- **Two items had been silently conflated.** Item 1's "heading plumbing" is a
  *compass bearing*; item 5's is an accessibility heading level. Nothing in the
  list said so and the words are identical.

Value is payoff, not effort — stated explicitly because the two run opposite
here: Tier C is rated high while being the largest item on the list, and items
7 and 8 are rated low while being nearly free.

## Part 2 — sorting, and the skill

Default **age descending, oldest first**; `by value` is the option. The reason
for that default is the reason the annotation exists: an item nobody has looked
at in seven sessions is the one most likely to have gone stale, and staleness is
the failure the list itself named last session. Value-first answers a different
question — what to pick up next.

The "Carried" / "Newly raised" split is gone. `age 0` says what the heading
said, and one flat list is what makes sorting well-defined at all. A one-line
index in the other key sits under the list so both readings are on the page.

`~/.claude/skills/next-list/SKILL.md`, `/next-list [n] [by value|by age]`,
`n` defaulting to 15 docs. It does not format an existing list — it **rebuilds**
one, which is where its value is:

    open set      the newest doc's Next section is authoritative
    lapsed        raised inside the window, never written up as done, absent
                  from the newest list  ->  surfaced with the doc it fell out of
    age           first appearance in the window; "age >= k" when the window
                  floors it, never a false exact number
    re-raises     dated from the re-raise, with a pointer to the original
    verify        the *why* before the *what*, flagging downstream-adoption
                  items as highest risk

That last step is the one that mattered an hour later.

## Part 3 — Tier C: `Rotate` + heading plumbing + Compass

The oldest item, age 7, value high. ~875 lines across four renderers, three
native hosts, Go, docs and the tutorial.

### The verify step earned its keep three times

**`permission.Heading()` does not exist and cannot.** The plan's step 3 named
it. The `permission` package is dead code left over from the govinci rebrand —
commented-out Portuguese calling an `InvokeNative` that is not in the codebase.
Not needed, as it turned out: the plan's own step 2 offers the better answer
(the browser's permission request is `StartHeading`'s job, since Safari only
grants it inside a user gesture and a separate API is one more thing to forget).
The package was left untouched and is now a Next item in its own right.

**"A `Rotate(-heading)` dial and a fixed needle" is not buildable portably.**
`core` has no z-stacking primitive: `Box` stacks vertically on all four targets
— settled deliberately in `2026-0904-0822-box-overlay-divergence` — and
`Position: absolute` is a declared web-only prop that neither native parses.
A needle drawn over the rose would be a web-only widget wearing a portable
name. The index mark went *above* the circle instead, which costs one glyph of
height and works everywhere.

**The widget exposed a framework-level a11y bug.** More below.

### `core.Rotate(deg)` — B4

One float, clockwise degrees, about the node's own centre. All three platforms
agree on both things that usually need a mapping table (degrees, and
clockwise-positive), which is the argument for one field rather than a
`Transform` type — the moment translate and scale join it, they stop agreeing
on composition order and the type has to say what it means.

    CSS       transform: rotate(Ndeg)        origin 50% 50%
    Compose   Modifier.rotate(N)             layout bounds' centre
    SwiftUI   .rotationEffect(.degrees(N))   anchor: .center

**Layer order is the whole difficulty, and no compiler can see it.** A rotation
is a drawing layer on both natives and a layer turns only what is painted
*after* it. Below the background, the content spins inside a square that stays
put — which compiles, animates, and is still wrong. The two chains run opposite
directions, so the same rule reads backwards on each:

    Compose   margin -> size -> ROTATE -> shadow -> clip -> bg -> border -> gestures -> padding
    SwiftUI   padding -> bg -> gestures -> clip -> border -> shadow -> frame -> ROTATE -> margin

Margin stays outside it on both, matching CSS: a transform turns the border box
and leaves the space reserved around it axis-aligned. Gestures stay inside it on
both, so the touch target turns with the pixels — both platforms transform
pointer coordinates through the layer.

`mobile/verify/rotate_test.go` pins the ordering by index comparison in each
file, with a `t.Fatalf` telling the next person to update the test rather than
delete it if the chain is restructured.

**The angle is not normalised**, and the doc says why at the point of
temptation: 350->370 and 350->10 point the same way and are different
animations. Folding it would take the choice away and pick the wrong one for the
exact case the field was added for — a compass fed folded bearings unwinds the
whole rose every time the user turns past north.

The WASM runtime writes `transform` unconditionally (`""` when zero) because the
patch path reuses the live element; htmlout omits the declaration instead
because it builds a fresh string and has no element to leave stale. An identity
`rotate(0deg)` is not equivalent to omission — it makes the element a containing
block for positioned descendants — so both halves are pinned.

### The sensor

    StartHeading/StopHeading ──"sensor" {kind, command}──▶ host magnetometer
    CurrentHeading ◀── heading record ◀──"heading" host event── host

A sensor is a third kind of thing beside a node and a service — subscribed
rather than commanded, and it costs battery while on — so the interesting design
question is not the payload but who may turn it off.

**Refcounted, not toggled.** The bug a bool makes easy is specific and silent: a
tab-bar badge and a compass screen both start the sensor, the screen is popped,
its Stop turns the sensor off, and the badge stops updating forever with nothing
in any log. Four lines, and that cannot be written. Unbalanced Stops clamp at
zero rather than going negative, or the next Start would only climb back to zero
and start nothing.

**`Received` and `Available` are two fields because they are two answers.** "No
reading yet" is a spinner; "this device has no compass" is a different screen,
and a spinner there spins forever. One boolean could not say both. `available`
defaults to *true* when the key is absent — the one asymmetry in the payload,
because a host sending a reading has demonstrably got a sensor, and three hosts
remembering a boolean on fifteen events a second is a contract that will be got
wrong.

**Notification is filtered; the record is not.** Sub-half-degree drift updates
`CurrentHeading` and wakes nobody, so a 15 Hz stream does not drive 15 full
render passes a second for a movement no eye can see. The filter compares
against the last *notified* value, so a slow turn still gets through; state
changes (availability, error, HasTrue, Active) always do.

**The seam at north is load-bearing arithmetic.** `AngleDelta` returns the
signed shortest turn, so 359->1 is +2 and not -358. It is used by the
notification filter and by Android's smoothing, and a plain subtraction in
either produces a spurious lurch exactly once per rotation. `NormalizeDegrees`
exists because `math.Mod` keeps the sign of its first argument, so -90 comes
back as -90.

Bearings are normalised in `ReceiveHeading` and nowhere else — the one point all
four hosts funnel through, which is what makes "Magnetic is in [0, 360)" an
invariant rather than a hope. Android's host converts radians to degrees (its
smoothing filter needs a continuous number) and deliberately leaves the fold to
Go; `mobile/verify` pins that split.

### `hooks.UseHeading`

Subscribes **before** starting, which costs one redundant render at mount (the
hook hears its own Active transition) and buys the ordering that matters: the
event most likely to land in a start-then-subscribe gap is the one that matters
most — a host with no compass answers `available: false` immediately and then
never speaks again. Missing a heading tick costs 66ms; missing that costs a
screen waiting forever. Pinned, with the reasoning, rather than smoothed over.

The mount/unmount limit hurts more here than for `UseAudio`: a stale
subscription there costs a redundant render, here a running magnetometer. The
doc says the workaround plainly — put a compass on its own route, whose frame
has its own cleanup registry. The tutorial lesson then *is* that arrangement,
which is why it can call the live hook safely.

### `components.Compass`

Flex only, since that is the whole vocabulary all four targets share:

    Column(align center)
     ├─ "▼"                       fixed index mark
     └─ Box(round, Rotate(-H))    JustifyBetween over three rows
          ├─ Row(center)   N
          ├─ Row(between)  W    E
          └─ Row(center)   S

Two spread rules place four letters at four compass points. The rose turns by
*minus* the heading — a navigation compass, not a magnetic one, because the
question a phone user is asking is "which way am I facing", read off the top.

`Heading` is a `float64` and not a `core.Heading`, so the widget also draws a
route leg or a wind reading; the sensor's struct is one caller among others.

### The bug the widget found

The compass announces once as a sentence, because a rose read in tree order is
"N W E S" whatever direction it points. That label sat on a plain container —
and **ARIA forbids an accessible name on a generic element**, so it was dropped
by screen readers on both web targets while both natives read it fine. Exactly
the shape of gap that ships: it works on the two targets the author is most
likely to be testing on.

Fixed by adding `core.RoleImg`, the seventeenth role — `Role.Image` on Compose,
`.isImage` on SwiftUI, verbatim on the web. The role coverage checks in
`mobile/verify/role_test.go` were confirmed to bite by removing the Swift arm
and watching it fail before restoring it.

## Verification

All six paths, green:

    go build / go vet / gofmt / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 8 mjs suites
    ios/verify/run.sh           data layer + view layer + NEW app layer
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

Two harnesses were extended, both because the new code was otherwise unreachable
by any check:

- **`ios/verify/run.sh` now type-checks the app layer.** It checked nothing
  before: `SystemEvents.swift`, `AudioPlayer.swift` and now `HeadingSensor.swift`
  lean on iOS-only frameworks the macOS target cannot see, so they compiled
  nowhere outside Xcode. The new pass targets iOS proper and *skips* rather than
  fails without the iPhoneOS SDK, keeping the script's Go-and-CLT promise.
  `GomobileBridge`/`GrMobApp`/`AppLifecycle` are still out — they import the
  generated `Mobile.xcframework`.
- **`wasm/verify` gained window-level event listeners and a
  `DeviceOrientationEvent` stub**, so the browser sensor's twelve cases are
  reachable: the alpha complement, the relative-event rejection, the throttle,
  the availability timeout and its cancellation, and all three iOS permission
  outcomes (granted, denied, rejected-outside-a-gesture).

New tests:

    core/heading_test.go               normalisation incl. NaN, the seam, the
                                       16-point rose, refcounting, unbalanced
                                       stops, the notification filter, and
                                       unavailable-vs-not-yet
    core/rotate_test.go                the setter clears where the merge cannot
    hooks/heading_test.go              start-once/stop-on-close, the mount
                                       render, and two consumers holding one
                                       sensor
    components/compass_test.go         the rotation's sign, display normalised
                                       while rotation is not, the role+label
                                       pair, and Size as the only knob
    htmlout/export_test.go             transform emitted, omitted at zero,
                                       winding preserved
    wasm/verify/rotate_test.mjs        the live half: returning to zero clears
    wasm/verify/heading_test.mjs       three browsers, twelve cases
    mobile/verify/rotate_test.go       both parsers, both appliers, both layer
                                       orders
    mobile/verify/sensor_test.go       both shells dispatch, both report an
                                       absent compass, both throttle
    examples/tutorial/chapter4_test.go the 359-degree chip and the waiting state

One test-harness detail worth remembering: a `Chip` renders as a `Button`
carrying its caption in the `label` prop, not as a `Text` child, so
`hasTextContaining` cannot find it.

## Docs

`docs/components.md` gained a `## Compass` section;
`docs/concepts/styling-and-theming.md` a `## Rotation` section with the
three-target mapping table; `docs/concepts/state-and-hooks.md` a
`### Sensors: UseHeading`. Tutorial lesson **4.10** ("Sensors: the compass")
drives a Compass from chips — 359 among them, deliberately — and mounts the live
hook beside it. ROADMAP gained three checked lines and the role count moved from
sixteen to seventeen.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 5 · value high) No `tab` / `tablist` role.** Needs a *state*
   (`aria-selected`), which has no home on `Style` and which both natives spell
   differently. Two consumers deep: `Calendar.Deselectable` wants the same slot.
   `RoleImg` landing this session shows the vocabulary half is cheap; it is the
   state half that is missing.
2. **(age 5 · value low) ARIA's `log`** for a chat transcript. `RoleStatus` is
   the nearest thing and is not the same promise. No consumer yet.
3. **(age 4 · value high)** An "on-light" tone per palette role, which both
   Button's outlined treatment and `Chip.ProminenceLoud` are working around.
   Age is from its re-raise; first written down 66 sessions back in
   `2026-0831-1515-button-variants`, then lost from the list entirely.
4. **(age 3 · value low) Heading level 6 is reachable and nothing in the
   framework goes past 2.** Unrelated to Tier C's "heading plumbing", which was
   a compass bearing.
5. **(age 2 · value medium) The themes give `Components.Input` no border.**
   Blocks widening `borderResetTags` to `<input>`/`<textarea>`; a palette
   decision.
6. **(age 2 · value low) `AccessibilityNestingLevel` has no consumer.** Watch
   whether the first nested list downstream reaches for it.
7. **(age 2 · value low) A `<select>` will need the border decision** if a
   picker node type lands. Contingent on a node type that does not exist.
8. **(age 1 · value high) A widget cannot say "this control is on".**
   `Deselectable` joins `tab`/`tablist` in wanting a selected/pressed state on
   `Style`. Same slot as item 1 — they land together or not at all.
9. **(age 1 · value medium) `contrastInk` picks black on `#007AFF`.** A selected
   calendar cell's numeral and dots come out `#000000` on the DefaultTheme blue.
   Same family as item 3.
10. **(age 1 · value low) The carried "Next" list is not audited.** Largely
    addressed: the `next-list` skill now encodes the practice, and its verify
    step caught two wrong premises in the Tier C plan within the hour. Dropped
    to low because what remains is a habit rather than a task — and because the
    skill lives in `~/.claude/skills/`, outside this repo, so nothing here
    records that it exists except this line.
11. **(age 0 · value medium) `core` has no z-stacking primitive.** `Compass`
    wanted a needle over its rose and could not have one: `Box` stacks
    vertically on all four targets and `Position: absolute` is web-only. First
    real consumer. A `core.Stack` node type would be a Compose `Box` / SwiftUI
    `ZStack` / `position: relative` + absolute children — the natives already
    behave this way and had to be talked out of it.
12. **(age 0 · value medium) The `permission` package is dead code.**
    Commented-out Portuguese calling an `InvokeNative` that does not exist. Tier
    C routed around it; Tier D's location work walks straight into it, and
    anyone reaching for `permission.Camera` today finds a type and no function.
13. **(age 0 · value low) Three app-layer Swift files are still checked by
    nothing.** `GomobileBridge`, `GrMobApp` and `AppLifecycle` import the
    generated `Mobile.xcframework`, so the new app-layer typecheck cannot
    include them without a gomobile bind. The other three are covered now.

Read by value instead: **high** 1, 3, 8 · **medium** 5, 9, 11, 12 · **low** 2,
4, 6, 7, 10, 13.
