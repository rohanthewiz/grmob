# Needs hardware

Things this repository has decided it **cannot check here**, with what is
unchecked, why the harness cannot reach it, and what a physical device would
have to do to close it.

## Why this file exists

`ai_docs/plans/non_goals.md` exists because an item that will never be done sits
in a Next list forever, re-read at the top of every session and re-declined at
the bottom of it. This file is the same argument for a different shape of item:
one that **will** be done, that nobody can do at a desk, and that therefore
carries session after session with `unchanged` written next to it.

The difference matters. A non-goal is closed. These are open, and the reason
they are open is a missing iPhone or a missing hour with a cable — not a missing
decision and not a missing design. Left in the Next list they look like work
that keeps being deferred; they are work that keeps being *blocked*, and the two
read identically in a list and want opposite things from a reader.

**Each entry states:** what is unchecked · where the code is · why nothing here
can answer it · the pass that would · and what the pass would settle. The last
line is the point: an entry whose pass would tell you nothing is not worth the
device.

## How an entry leaves

By being run. Write the reading into the code it is about — the way
`android/device/launch.sh` carries its own numbers in its header rather than in
a session doc — and delete the entry. A reading that lives only in a session doc
is a reading the next person will not find.

An entry may also leave by being re-argued into `non_goals.md`, if it turns out
the thing it wanted to check is not worth checking. That has not happened yet.

---

## The two editors' focus commands have never raised a keyboard

*Raised: 2026-09-11 · Code: `ios/GrMob/Runtime/GrMobEditorFocus.swift`,
`android/.../GrMobRichText.kt` (`applyFocus`),
`android/.../GrMobCodeEditor.kt` (the `FocusRequester` on the field)*

**What is unchecked.** `core.Focus` and `core.DismissKeyboard` reach
`CodeEditor` and `RichTextEditor` on all four targets. Three things about that
are checked and one is not:

| Layer | Checked by | Says |
|---|---|---|
| the stamp | `core.TestBothEditorsTakeFocusCommands` | the command is written onto the node |
| the destination | `wasm/verify`, 10 cases | the command resolves to the right element |
| the native calls | `mobile/verify`, textual | the right method is named in the right file |
| **a keyboard actually appearing** | **nothing** | — |

**Why nothing here can answer it.** The web shim's `focus()` is an assignment
any element accepts, so the WASM pass can say which element ended up active and
nothing about a caret. The two native passes are textual: `mobile/verify` greps
the source for the calls the contract requires, which proves the call is written
and not that the platform honoured it. And the platform is exactly what is in
doubt — `showSoftInput(view, SHOW_IMPLICIT)` is **advisory**. The system may
decline it, and a declined request looks identical to a granted one from inside
the process.

**The arm most likely to be wrong.** Android's `RichTextEditor` is the only node
in this runtime that talks to the `InputMethodManager` at all. Every other
focusable leaf is a Compose or SwiftUI control, where raising and lowering the
keyboard is a consequence of focus; a classic `EditText` inside an `AndroidView`
is not — `requestFocus()` leaves the keyboard down and `clearFocus()` leaves it
up — so both edges are asked for explicitly. Two explicit calls that no test
exercises, in the one file where a reader copying the Compose field's
implementation would delete them.

iOS's is the async hop: `GrMobEditorFocus` calls `becomeFirstResponder` one
runloop turn after the command arrives, because the hosted `UITextView` may not
be in a window yet on the pass that carries the stamp. One hop is a guess about
how late "late enough" is.

**The pass.** One Android device and one iPhone, the tutorial's editor lessons.
Per editor: a screen that opens with the command already stamped (the
mounts-while-targeted case, which is the arm whose epoch rule differs from every
other command's), then a `Focus` issued while the screen is up, then a
`DismissKeyboard`. Watch for the keyboard, not for the caret — a caret can
appear with no keyboard behind it, and that is the failure this is looking for.

**What it would settle.** Whether `SHOW_IMPLICIT` is enough on the rich-text
editor, and whether one hop is enough on iOS. Both have a known next move if
not: `SHOW_FORCED` is the wrong answer and `windowInsetsController` is the right
one; on iOS the hop becomes a wait on the view's window.

---

## Four editors, three behaviours that only exist on a device

*Raised: 2026-09-11 · Code:
`{ios,android}/.../GrMob{CodeEditor,RichText}.{swift,kt}`*

**What is unchecked.** Three behaviours, each of which only exists when a real
input method or a real second app is in the picture:

1. **The IME composing region.** Both Android editors are *designed around* it.
   `GrMobCodeEditor` puts Go's colours in a `VisualTransformation` rather than
   in the state precisely because writing a new `TextFieldValue`
   mid-composition cancels the composition — the whole reason the decoration is
   a pure function of the buffer instead of a rewrite of it. Nothing has ever
   composed a character against either editor. The design's central claim is
   an argument.
2. **Hardware-keyboard Tab.** Both code editors take Tab away from the platform
   — Compose via `onPreviewKeyEvent`, UIKit via
   `shouldChangeTextIn` — and both files say why in the same words: *a soft
   keyboard has no Tab.* Which also means a simulator with the host's keyboard
   disabled has no Tab, and that is the configuration every run so far has been.
   Needs an iPad with a keyboard, or a physical Android keyboard.
3. **Paste from another app into the rich-text editor.** iOS's code editor
   states the boundary — single characters only, so a paste carrying newlines or
   tabs is inserted verbatim — and the rich-text editor's span handling names
   "a paste or a system control" as the platform-owned case it has to survive.
   A clipboard crossing an app boundary carries styles this runtime never wrote.

**Why nothing here can answer it.** All three are the input stack, and the input
stack is the one part of these files no harness in this repository instantiates.
`go test` sees a document transformation; `wasm/verify`'s DOM has no Selection
API and no `contenteditable` behaviour; `android/verify/sources.sh` compiles the
Kotlin and says nothing about what an IME does to it; `ios/verify` typechecks.
The editors are held to their contracts everywhere and exercised nowhere.

**The pass.** A device each. Compose a word with a real IME over each Android
editor and watch whether the composition survives a colour arriving from Go
mid-word (the failure is the underline disappearing, or the word committing
early). Tab against both code editors from a hardware keyboard. Paste styled
text from a mail client into the rich-text editor.

**What it would settle.** Whether the transformation-not-state design does what
it was chosen for. That is the single largest unverified claim in either
editor, and it is load-bearing for both.

---

## `launch.sh`'s numbers are one emulator's

*Raised: 2026-09-11 · Code:
`android/device/launch.sh` — the header carries every reading*

**What is unchecked.** Every startup number this project has for Android came
off one emulator (`sdk_gphone64_arm64`, 1080x2400), Debug builds of both halves,
five cold launches per arm. The header says so in its own last paragraph. What
is sound about them is the **A/B**: each table's arms ran on the same machine
within the same hour, so the differences are real. What is not established is
any absolute, and two of the conclusions drawn here are shaped like absolutes.

Specifically:

```
the parse was 1666ms of a 4850ms launch      -> the payload was the lever
the noise floor is about 8ms, or 4%          -> below this, do not report a win
```

The first decided that `json:",omitzero"` on `core.Style` was worth doing, and
it was (7.9x off the wire, -27% of the launch). The second is now the rule by
which *later* payload changes are accepted or rejected, and it was derived from
the spread of five runs on this one machine. A noise floor is a property of the
measuring apparatus, and the apparatus here is an emulator on a laptop that was
doing other things.

**Why nothing here can answer it.** An emulator's CPU is the host's, its I/O is
the host's filesystem, and its JIT warms differently. The iOS side of this same
question already has the cautionary number: the identical payload parses in 6ms
on the simulator and took 1666ms through `org.json` on the emulator. Two hundred
times is not a device-speed difference — it is a different parser — but it is a
standing reminder that this project has twice found the host and the target to
disagree by more than the effect being measured.

**The pass.** One physical Android device. Five cold launches of each arm with
`--stages`, which prints the bridge / parse / build split that `Startup.kt`
times. The arms are already written and the control arm (title + progress card
only, same APK) is the one that makes the rest mean anything.

**What it would settle.** Whether the parse is still the largest thing in a
launch on real hardware, and what the noise floor actually is.

It no longer gates a decision. Windowing `core.List` was waiting on it and has
since been declined on the payload instead — the chapter collapse took the same
bytes a window would have, so what is left is ~8KB and about 55ms on this
emulator (`non_goals.md`, and `TestWhatWindowingWouldSave` for the re-taken
profile). A device would only make that number smaller, since this emulator's
`org.json` is the slowest parser this runtime meets. So the pass is now about
the instrument rather than about any pending build: every startup figure this
project quotes is one emulator's, and two of them — the parse being the lever,
and the 8ms floor below which a payload change cannot be reported as a win —
are used as if they were absolutes.

---

## `HeadingSensor.retryIfArmed` has never run

*Raised: 2026-09-11 · Code:
`ios/GrMob/App/HeadingSensor.swift:164`*

**What is unchecked.** The arm that re-starts a compass after the user withdraws
a refusal. `retryIfArmed` guards on `CLLocationManager.headingAvailable()`,
which is **false on the simulator** — there is no magnetometer — so every run of
this code on every machine here has returned at the guard. The body below it
has never executed.

The body is three lines and one of them is load-bearing:

```swift
manager.stopUpdatingHeading()   // torn down before it is built up
manager.startUpdatingHeading()
```

The teardown is there for a reason measured one file over
(`LocationSensor.beginUpdates`): starting a manager that is already in a
delivery session is a no-op on top of that session's state, so a manager the
platform has just refused **stays refused and delivers nothing**. That is
exactly the state `retryIfArmed` is reached in, and it is the only path that can
reach `beginUpdates` with a session the platform has already killed. The fix is
copied from a sibling and verified on the sibling; on this sensor it is
inference.

**Why nothing here can answer it.** `headingAvailable()` is hardware. There is
no simulator setting that makes it true, and the arm is behind it by design —
re-asking rather than assuming, because this path is only reachable on a device
that had a magnetometer and was refused.

**The pass.** A physical iPhone, one compass lesson. Open it, deny location,
leave the app, grant it in Settings, come back. The compass should start
pointing. Then the same again without leaving the lesson, since the
foreground-return path and the in-place grant reach `retryIfArmed` from
different callers.

**What it would settle.** Whether the stop-then-start actually revives a refused
heading session, which is the whole of this function. Cheap to run — one lesson,
several taps in Settings — and it is the last unexercised arm in either sensor.
