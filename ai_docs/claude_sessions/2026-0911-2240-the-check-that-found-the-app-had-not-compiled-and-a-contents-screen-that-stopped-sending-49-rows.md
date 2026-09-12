# Session: the check that found the app had not compiled, and a contents screen that stopped sending 49 rows

Session: https://claude.ai/code/session_018HHEppckHrYoZNwpi2jgha
Date: 2026-09-11 · Previous:
`2026-0911-2124-two-editors-a-lexer-a-document-and-the-hook-that-had-to-be-the-callers.md`

## Ask

> Let's do items 2 - 4 of the next list followed by any other next list item
> that does not need a physical device

Items 2, 3 and 4 of the previous session's `## Next`, then item 8 — the only
remaining one that needs no hardware, and a UX call, so it was put to the user
(they chose "expand the current chapter only"). Items 1, 5 and 6 need devices;
7 is gated on 5.

`go test ./...`, `wasm/verify/run.sh`, `ios/verify/run.sh` and
`android/verify/run.sh` are all green.

The four sentences worth keeping if the rest is lost:

1. **The Android app had not compiled since the editors landed, and the first
   run of the new check said so.** `GrMobCodeEditor.kt` called
   `GrMobNode.isDisabled()`, which `Renderer.kt` declares `private` — and a
   `private` top-level declaration in Kotlin is *file*-private. The textual
   contract tests check that the right calls are made, never that they resolve.
2. **A classpath assembled by guesswork fails on the library rather than on the
   source**, which is the one thing a compile check must never do. The gradle
   cache holds two Compose releases here; newest-wins picks the one the BOM does
   not pin, and the pass then reports `Modifier.animateItemPlacement` missing
   from code that is fine.
3. **The Compose compiler plugin is the check, not a garnish.** Without it
   `@Composable` is an ordinary annotation and `fun Bad() { Text("x") }`
   compiles clean. So its absence is a named SKIP rather than a weaker pass
   wearing the same OK.
4. **A rule that has two doors belongs in neither of them.** The contents
   screen's chapter expansion was wired to `open`, and the deep-link door
   `goTo` never calls it. It lives in `markVisited` now: recording a lesson as
   opened *is* opening its chapter.

---

## Item 2 — `android/verify` grew a classpath

### What was missing

`run.sh` executed the two Kotlin files that import nothing. The other nine
runtime files — the renderer, the style solver, the map, both editors — plus the
whole app layer imported Compose and the Android SDK, and **nothing in this
repository compiled any of them.** They were held to their contracts textually,
by `mobile/verify` searching their source for the calls they are supposed to
make. Real checks of the rules; no check at all of whether the file resolves.

### What landed

| File | What it is |
|---|---|
| `android/verify/sources.sh` | the pass: two stages, its own gate, ~7s |
| `android/verify/kotlinc.sh` | finding a compiler in the gradle cache, shared with `run.sh` |
| `android/verify/gate.sh` | `kotlin_source_verdict`, five preconditions as a function of values |
| `android/app/build.gradle` | `printVerifyClasspath` — the resolution plus AGP's platform jars |

Two stages:

```
com.grmob.runtime   always
com.grmob.app       only when android/app/libs/grmob.aar is present
```

### The three decisions

**It is not `:app:compileDebugKotlin`.** That task needs `app/libs/grmob.aar`,
which is gitignored and produced by `android/build.sh` — gomobile, the NDK and a
Go toolchain. `com.grmob.runtime` imports nothing from that `.aar`,
deliberately, so it compiles on a checkout that has never run gomobile. That is
the difference between a check that runs and a check that needs half an hour of
setup first, and it is why the app layer is a second stage rather than the whole
thing. (The same argument `run.sh` already makes about JUnit one step down.)

**The classpath comes from gradle, not from a glob over its cache.** The glob
was tried first and is wrong here in a way worth recording: this machine's cache
holds Compose 1.6.8, which the BOM pins, *and* 1.10.0, which something else once
resolved. Newest-wins picks 1.10.0 and the pass fails on
`Modifier.animateItemPlacement` — an API 1.6.8 has and 1.10.0 removed. Same
lesson `TestTheComposeCensusClaimsAreWhatTheSourceSays` records about the
sources jar, from the other direction: there a wrong version made a paragraph
quietly stale, here it makes the pass loudly wrong.

**The Compose compiler plugin is required.** Break-tested both ways: without the
plugin a `@Composable` called from a plain function compiles with no diagnostic;
with it, two errors. That is the largest class of Compose-specific mistake and
the plugin is the only thing that knows the rule. It also has to be the
compiler's own version, which is why this pass always uses the cache's matched
pair and never a `kotlinc` on `PATH` — the opposite of what `run.sh` prefers,
and stated as such in both files.

### What it found

```
GrMobCodeEditor.kt:261:33: error: cannot access 'fun GrMobNode.isDisabled(): Boolean':
                          it is private in file.
```

`Renderer.kt:139` declared it `private`. Fixed by making it `internal` —
module-wide is the accurate visibility for a helper the runtime's widget files
share and nothing outside the module should see.

### Two smaller things

`newest_jar` and the six-jar compiler assembly moved out of `run.sh` into
`kotlinc.sh`, because a toolchain lookup written out twice proves only that one
person made the same choice twice. `kotlin_cache_compiler` is consulted through
an `if` rather than `f && x=yes`: whether `set -e` fires on the failing half of
an AND-OR list that is itself the last command differs between shells.

`gate_test.sh` gained thirteen cases — every arm, the order (each case is a
machine on which the *later* answer is also "no", which is exactly when a
misordered gate sends the reader to fix the wrong thing), and all ten pairs of
skips checked distinct. Two mutations were run against it: swapping the SDK and
classpath arms, and turning the plugin arm into a `run:`. Both caught.

---

## Item 3 — `core.Focus` / `core.DismissKeyboard` reach both editors

`CodeEditor` and `RichTextEditor` joined `focusableLeafTypes`. The work was not
the map entry.

**On every other focusable leaf the node the renderer builds *is* the control.
On these two it is a box.** So each renderer resolves the stamp:

| Target | Where the command lands |
|---|---|
| htmlout | nowhere, and it is stated in `isFormControl`: the editors export read-only, so there is no element a browser would focus and `autofocus` on a `<pre>` has nothing to put a caret in |
| WASM | `focusTargetOf` — the `<textarea>` inside the `<pre>` for a CodeEditor; the element itself for a RichTextEditor, which carries `contenteditable`. Returns null rather than falling back to `el`, because focusing the `<pre>` would move focus off whatever legitimately holds it |
| iOS | `GrMobEditorFocus` (new), shared by both coordinators: the hosted `UITextView`'s responder, one async hop, epoch remembered per editor |
| Android | `CodeEditor` gets a `FocusRequester` on the field — not on the Row, which also holds the gutter. `RichTextEditor` drives the `EditText` directly |

**Android's rich-text editor carries an obligation no other node in this runtime
has.** Compose and SwiftUI both raise and lower the soft keyboard as a
consequence of focus; a classic View does not — `requestFocus()` leaves the
keyboard down and `clearFocus()` leaves it up — so the `InputMethodManager` is
asked explicitly on both edges. That is the half `core.DismissKeyboard` exists
for, and it is the thing most likely to be dropped by someone copying the
Compose field's implementation.

The one place the editors' focus epoch differs from their *command* epoch is
deliberate and now stated on both: a focus command **does** fire the first time
it sees a non-zero epoch, because "open a screen with the cursor in its editor"
issues the command one pass before the editor exists. An editor command names a
moment and an edit, and one issued while the editor was off screen was missed.

### Checked

- `core`: `TestBothEditorsTakeFocusCommands`. Neither editor carries a
  `FocusTarget` — that stamps unconditionally and would pass over a set that had
  lost both entries. Mutation-tested by removing each map entry in turn.
- `wasm/verify`: 10 new cases across both editors. Every assertion names the
  element that ended up active rather than merely checking that something did,
  because this DOM's `focus()` is an assignment any element accepts. Mutating
  `focusTargetOf` back to `el` fails two of them.
- `mobile/verify`: textual contracts for all four native arms, including the two
  `InputMethodManager` calls.

---

## Item 4 — measured, and declined with the numbers

The item asked for a profile before anything else. Counted rather than timed,
because a count is the same number in a browser as against the harness DOM and a
millisecond is not:

```
blocks x runs      elements destroyed and recreated
   1 x 1                          2
  10 x 4                         70
 100 x 6                      1,000
 500 x 6                      5,000
2000 x 6                     20,000
```

A patch rewriting only the blocks an edit touched would put ~10 in every row.

**The item's premise was narrower than the truth.** All five callers of
`richTextToDOM` are full rebuilds — a pending mark, a paste, undo/redo, any
toolbar command, and any rewrite arriving from Go. The pending-mark path was
just the one noticed.

Not taken, for two reasons about correctness rather than effort:

1. **A block is not an element here.** Consecutive list blocks share one `<ul>`,
   so the replaceable unit is a maximal run of blocks sharing a container, and
   computing that run correctly on every edit is the whole of the work.
2. **Leaving an element in place is only safe if it still describes its block**,
   and between two calls the *browser* has been editing this DOM —
   `contenteditable` typing splits text nodes and inserts elements. A full
   rebuild normalises that away every time; a partial one would normalise what
   it rewrote and leave the rest in whatever shape the browser left it. That is
   a model/screen drift in an editor, and nothing here can test for it: the
   harness DOM has no Selection API and no `contenteditable` behaviour at all —
   the same limit that made the whole command vocabulary a pure document
   transformation in the first place.

The numbers and the argument are written above `richTextToDOM`, so the next
person starts from the measurement.

---

## Item 8 — the contents screen's chapter cards collapse

Put to the user as a question, because the previous session's note says it is a
UX decision. They chose "expand the current chapter only".

### Where the state lives, and why not `components.Accordion`

`Accordion` owns its expansion and is the right answer for a single section;
`components.Collapse`'s own doc names where that stops, and this is the case it
names. Eight cards means eight independent `NewState`s the screen cannot
address — no way to open the chapter a reader just came back from, and no way to
shut them all. So the map is `tutorial.expanded map[int]bool`, in the session
scope beside `visited`, and each card is a `components.CollapseBand` (the
disclosure control, with the heading/button/aria-expanded shape already argued
out in `components.disclosure`).

Seeded `{0: true}`: a contents screen whose every chapter is shut is eight
headers and nothing else, and a first-time reader has no reason to guess which
to press.

### The two doors

The rule was first wired to `open` — and `goTo`, the inbound deep-link door,
never calls it. `grmob://lesson/2.3` opened a lesson the reader never pressed a
row for, and `‹ Contents` landed them on a shut card. It lives in `markVisited`
now, where it is true by construction: every path that records a lesson as
opened opens its chapter, because recording it *is* opening it. This is the same
drift the file already worries about one function down ("so the address bar
cannot drift from the screen because one path forgot to report").

### The numbers

```
all eight cards open      53,156 bytes    what this screen used to send
one card open             17,366 bytes    what it sends today
every card shut           12,889 bytes    the floor
```

67% off, with no protocol change. Two things worth taking from those rather than
from the percentage:

- A row costs about 800 bytes, so the saving is linear in how many rows are on
  screen and would be the same on any screen that stops drawing rows nobody
  asked for.
- **The floor is a quarter of the original.** Eight disclosures are not free —
  the heading-around-button-around-chevron shape is four nodes with styles.
  Collapsing wins here because 49 rows is far more than eight bands; it would
  not win for a screen with three rows per section.

### What it did to item 7

**Windowing `core.List` over the bridge lost its motivating screen.** It was
sized at 66–76% of the payload and needs a bootstrap guess plus placeholder
children; the collapse is 67% with no protocol change at all. That does not
retire windowing — a reader who opens a forty-lesson chapter is back to forty
rows, and windowing is what keeps that from being paid all at once — but the
tutorial is no longer the screen arguing for it. Recorded in `TestHomeTreeSize`
and in `Home`'s own doc.

### The tests

`openLesson` presses the chapter band and then the row, the way a reader does,
so every lesson test exercises the new control. It presses only when the row is
*already absent*, because pressing an open chapter's band shuts it.

`TestHomeListsEveryLesson` became
`TestHomeListsEveryChapterAndTheOpenChapterSLessons`: the curriculum is still
wholly reachable (the part a reader would notice going missing) and what is on
screen at any moment is the chapters plus one chapter's rows. Both halves,
because a screen that had stopped drawing an open chapter's rows would pass a
check that only counted headers.

`TestTheChapterAReaderCameOutOfIsOpen` drives both doors. Door one routes
through the last lesson of chapter 1 — open already, so nothing about that arm
depends on the helper having expanded anything — then `Next ›` into chapter 2.
Removing the rule from `markVisited` fails both arms.

---

## What is checked, and what is not

- `go test ./...` — green.
- `wasm/verify/run.sh` — green, 10 new editor-focus cases, plus the existing
  replay and browser passes.
- `ios/verify/run.sh` — green; the macOS typecheck and the app-layer pass
  against the iPhoneOS SDK both compile the new `GrMobEditorFocus.swift`.
- `android/verify/run.sh` — green, **and now compiles Kotlin that imports
  Compose**, which is the gap this session closed.
- **Not checked anywhere:** anything needing a real caret or a real keyboard.
  The focus commands are pinned at the stamp (Go), at the resolved element (the
  web shim, where `focus()` is an assignment) and textually on both natives. No
  harness here can say that a soft keyboard actually came up.

---

## Next

1. **(new · value medium) A device pass on the two editors' focus commands.**
   The stamp is checked, the destination is checked, and whether a keyboard
   actually appears is not. Android's rich-text editor is the one to watch: it
   is the only node in this runtime that asks the `InputMethodManager`
   explicitly, and `showSoftInput(..., SHOW_IMPLICIT)` is advisory — the system
   may decline it. iOS's `becomeFirstResponder` after one async hop is the other
   unexercised arm.
2. **(carried · value medium) A device pass on all four editors.** Unchanged
   from the previous session. The IME composing region on Android (both editors
   guard on it and nothing has exercised the guard), hardware-keyboard Tab on
   iPad, and paste from another app into the rich-text editor. Everything else
   has a test; these have an argument.
3. **(new · value low) `android/verify/sources.sh` does not run in CI-shaped
   environments.** It needs an Android SDK and a gradle cache the app build
   filled, and skips cleanly without them — which is the right stance and also
   means a machine that has never built the app checks nothing. The four skip
   sentences each name their own remedy; nobody has run it anywhere but this
   laptop.
4. **(carried · value medium) `launch.sh`'s numbers are one emulator's.**
   Unchanged. One device, five cold launches, `--stages`. The
   emulator↔simulator gap is 200x on the same bytes, so this is not a
   formality. Now also the thing standing between item 5 below and a decision.
5. **(carried · value low, downgraded from medium) Windowing `core.List` over
   the bridge.** Still gated on item 4, and it lost its motivating screen this
   session: the tutorial's contents is 17,366 bytes now, where windowing was
   sized at 66–76% of 53,156. Still the right answer for a single long chapter
   or any other genuinely long list; no longer urgent.
6. **(carried · value low) `HeadingSensor.retryIfArmed` is unmeasured.**
   Unchanged. `headingAvailable()` is false on the simulator; needs a physical
   iPhone, one lesson, several taps in Settings.
7. **(new · value low) `richTextToDOM` rebuilds the whole document on every
   write, and the block-level patch is specified but not built.** Sized this
   session: 20,000 elements for a 2000-block document against ~10 for a patch.
   Declined for two reasons written above the function — list blocks share a
   container, and a partial rewrite stops normalising away what
   `contenteditable` did between calls. Revisitable by anyone with a profile
   from a real document; the design sketch is a group-level diff where a group
   is a maximal run of blocks sharing a container, with a full rebuild whenever
   the group shape changes.
8. **(closed) `core.Focus` and `core.DismissKeyboard` do not reach the
   editors.** Done this session across all four targets.
9. **(closed) Kotlin that imports Compose or Android is compiled by nothing
   here.** Done this session; it found a build break on its first run.
10. **(closed) The contents screen's eight chapter cards are all expanded.**
    Done this session, 67% off the payload.
11. **(closed) A short wire vocabulary.** Unchanged: sized at 17–28% and
    declined with the numbers in hand.
12. **(declined, non-goal)** Twenty-five entries. See
    `ai_docs/plans/non_goals.md`. Unchanged this session: the editors' own v1
    non-goals (collaborative editing, autocomplete/LSP, code folding,
    find-and-replace, images or tables in rich text, full CommonMark, a
    host-side highlighter grammar, per-run font family) stay recorded in
    `ai_docs/plans/components-editors.md` where they were decided.
