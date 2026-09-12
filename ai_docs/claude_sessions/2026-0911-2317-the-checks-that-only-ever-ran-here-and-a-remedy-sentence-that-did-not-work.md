# Session: the checks that only ever ran here, and a remedy sentence that did not work

Session: https://claude.ai/code/session_018HHEppckHrYoZNwpi2jgha
Date: 2026-09-11 · Previous:
`2026-0911-2240-the-check-that-found-the-app-had-not-compiled-and-a-contents-screen-that-stopped-sending-49-rows.md`

## Ask

> Let's do items 3, 5 and 7 of the next list, but first move items 1-7 to
> ai_docs/plans/need_hardware.md

Put to the user, because items 3, 5 and 7 are the three that need no hardware:
which of 1-7 belong in the new file. They chose 1, 2, 4 and 6 — the
hardware-blocked ones only.

Then items 3, 5 and 7. Items 5 and 7 had both been *declined with numbers* last
session, so "do them" was put to the user as well; they chose to re-profile
and re-decide item 5, and to take the safe half of item 7.

`go test ./...`, `go test -race ./...`, `wasm/verify/run.sh`,
`ios/verify/run.sh` and `android/verify/run.sh` are all green — and the two Go
commands are green **on a fresh clone**, which is where they were not.

The five sentences worth keeping if the rest is lost:

1. **CI had been red on master since before the editors landed, and nothing
   here could have said so.** Two failures, both deterministic, both invisible
   on this laptop. The Android job was green the whole time: `assembleDebug`
   already compiles every Kotlin file, so the gap the previous session closed
   was a *laptop* gap, not a repository one.
2. **`git check-ignore` answers "not ignored" for a directory that does not
   exist.** `build/` is a directory-only pattern and git decides directory-ness
   from the filesystem. `TestTheCitationSkipsGitAlreadyMakes` therefore passed
   here, where the build dirs exist, and failed on every machine that had never
   built. A trailing slash on the query states the kind and fixes it.
3. **A gate consulted after the first executable line is not a gate.**
   `android/verify/run.sh` ran `go run .` *above* the gate, so a machine with a
   JDK and no Go died at exit 127 — the same stance inversion `gate.sh`'s whole
   doc is about, in the same file, two toolchains later.
4. **Both compiler skips named a task that does not fetch a compiler.** A
   reader who met the skip, ran what it said, and tried again met the same
   skip. A skip sentence is the one part of a gate no arm of `gate_test.sh` can
   check: the table proves the five sentences are *different* and nothing
   proves any of them is *true*.
5. **Windowing `core.List` was being priced from a table the chapter collapse
   had already falsified.** 66-76% of 51,242 bytes became 42-50% of 17,366 —
   34-39KB to 7-9KB — because the collapse and the window were aimed at the
   same 45 lesson rows. The test printed and asserted nothing, so it went stale
   in silence.

---

## First: `ai_docs/plans/need_hardware.md`

`non_goals.md`'s opposite number. A non-goal is closed; these are open, and the
reason they are open is a missing iPhone or a missing hour with a cable — not a
missing decision and not a missing design. Left in a Next list the two read
identically and want opposite things from a reader.

Four entries, each stating what is unchecked · where the code is · why nothing
here can answer it · the pass that would · and what the pass would settle:

| Entry | The arm most likely to be wrong |
|---|---|
| the two editors' focus commands | Android's `RichTextEditor` — the only node in this runtime that asks the `InputMethodManager`, and `SHOW_IMPLICIT` is advisory |
| four editors, three device-only behaviours | the IME composing region, which both Android editors are *designed around* and which nothing has ever exercised |
| `launch.sh`'s numbers are one emulator's | the 8ms noise floor, which is a property of the apparatus and is used as an absolute |
| `HeadingSensor.retryIfArmed` | the stop-then-start, copied from a sibling and verified on the sibling |

An entry leaves by being run, with the reading written into the code it is
about — the way `android/device/launch.sh` carries its own numbers — not into a
session doc. `non_goals.md` gained a pointer to it.

---

## Item 3 — the checks that had only ever run on this laptop

The item said `android/verify/sources.sh` needs an SDK and a warm gradle cache
and "nobody has run it anywhere but this laptop". Running it elsewhere is what
the whole session turned on.

### CI was red, and had been

`gh run list` first. Every run on master failing, back at least four commits.
Android green, iOS green, **Go red** — and the previous session's headline
(the app had not compiled) was about a gap CI did not have: `assembleDebug`
compiles all of `com.grmob.app`, so CI would have caught the `private
isDisabled()`. What `sources.sh` buys is the *laptop*, and a fast Kotlin signal
with no NDK.

Two deterministic failures.

### 1. `TestTheCitationSkipsGitAlreadyMakes` — a test about one working tree

```
citationSkipDirs calls android/build gitIgnores and `git check-ignore`
says no rule excludes it.
```

`android/.gitignore` says `build/`. A **directory-only** pattern matches only a
path git believes is a directory — for a path that exists git asks the
filesystem, and for one that does not it can only believe what the query says.
So:

```
this laptop, android/build on disk     ignored       passed
a fresh checkout, nothing built        not ignored   failed
```

Proved in a three-line scratch repo before touching anything. The fix is a
trailing slash on the query — every prefix in `citationSkipDirs` is a directory
and a citation walk descends into nothing else. Verified both ways (fresh clone
fails before, passes after; laptop still passes) and mutation-tested by deleting
`build/` from `android/.gitignore` on the clean checkout, which the fixed
assertion still catches.

The old comment claimed `check-ignore` "answers for a path that does not exist —
which is the case that matters". It does, and not for these patterns; that is
now written out with the table.

### 2. `TestRetiringAWedgedProcessDoesNotWaitForever` — a test about which shell

10.00s against a 150ms grace, every run. No Linux available here, so the
mechanism was proved rather than reproduced: a probe running `retire`'s exact
shape against three children.

```
sh -c 'sleep 60'      (execs)    Kill reached the pipe, 151ms
sh -c 'sleep 60; :'   (forks)    WEDGED: pipe still open 5s after Kill
sleep 60              (no shell) Kill reached the pipe, 151ms
```

A shell given one simple command may exec itself away or may fork and wait. If
it forks, `Kill` reaches the shell and the **grandchild** keeps the stdout write
end, so the drain never ends — and `retire`'s `<-done` after the Kill is
unbounded. macOS's `/bin/sh` execs; Ubuntu's did not.

So the test reported "retire has no deadline" — a true symptom with the wrong
cause, from a fixture the assertion never named. The child is `sleep` with no
shell now: one process by construction, which is the shape the doc always meant.

And what that leaves said about `retire` is now said on `retire`: the bound is
conditional on the child not forking, `git cat-file --batch` does not (not even
a filter process — `--batch` hands back raw blobs and `--filters` is never
passed), and the process-group defence is named as the change to make if a
forking child ever ends up on those pipes.

### 3. A red pass from a missing Go — the same inversion, two toolchains later

Found by running `android/verify/run.sh` with a stripped PATH:

```
SKIP: the Compose census's source half (no go on PATH to run it with)
android/verify/run.sh: line 119: go: command not found
exit=127
```

The script announced the skip for the census, then ran `go run .` for the case
table anyway. `set -e` turned a missing optional toolchain into a failing pass —
which is word for word what `gate.sh`'s header says the kotlinc-before-java
order used to do.

`jvm_harness_verdict` takes a `have-go` now, **asked first**, and the gate moved
above `go run .`. The ordering argument is not the usual one — Go, java and
kotlinc are independent `command -v` facts and nothing forces an order between
them. What forces it is where the first executable line sits. A gate consulted
after it is not a gate.

Two mutations, both caught: swapping the go and java arms, and removing the go
arm entirely (3 failures).

After: a clean `SKIP` and `exit=0`, with `sources.sh` still running and passing
above it — which is the right partition, since the source compile needs no Go.

### 4. The remedy sentences, checked

`kotlin_source_verdict`'s compiler skip said to run `:app:printVerifyClasspath`.
Measured against an empty `GRADLE_USER_HOME`:

```
kotlin-compiler-embeddable                0
kotlin-stdlib                             2
kotlin-reflect                            1
kotlin-daemon-embeddable                  0
kotlinx-coroutines-core-jvm               2
annotations                               1
kotlin-compose-compiler-plugin-embeddable 0
```

Four of six, and **not** the two that are nobody's compile dependency. Four of
six is the worst shape for this: enough that the cache looks populated, not
enough to compile a line. The harness gate's sentence said `compileDebugKotlin`,
which does pull one — and needs `libs/grmob.aar`, the gomobile build this pass
exists to not require.

So `:app:fetchKotlinCompiler` was added, in the idiom `fetchComposeLayoutSources`
already set: a configuration of its own so a compiler is never on the app's
compile classpath, two coordinates (the compiler and the Compose plugin) at the
build's own Kotlin version, and the other four jars left to transitivity —
naming them would pin versions this repository has no opinion about. The Kotlin
version is `ext.kotlinVersion` in `android/build.gradle` now, where it was
spelled twice.

Verified from empty: the task fetches all seven, and then

```
sources.sh   OK: com.grmob.runtime and com.grmob.app compile …
run.sh       OK … OK … OK: 15 picker menus … OK: 22 value ranges
```

on a cache filled by exactly two documented gradle commands. No `assembleDebug`,
no gomobile, no NDK.

**The lesson is written into `gate.sh`:** a skip sentence is the only thing a
reader on a broken machine ever sees, and it is the one part of a gate no arm of
`gate_test.sh` can check. The table proves the five sentences are *different*;
nothing proves any is *true*. The only instrument for that is running the remedy
on a machine that has the fault.

### 5. CI runs android/verify now

```
Install the compile SDK                        (existing)
Populate the gradle cache android/verify reads (new, online, ~20s)
Compile the Compose runtime (no NDK, gomobile) (new, ~7s — the fast fail)
Build the AAR with the pinned gomobile         (existing, ~2.5min)
Assemble the debug APK                         (existing)
Run android/verify against the built AAR       (new)
```

The fetch is a step of its own so the scripts keep their `--offline` promise:
that stance is exactly what makes a cold cache — every first CI run — skip every
Kotlin check with a sentence nobody reads. The early compile sits before the
bind because `com.grmob.runtime` imports nothing from the `.aar`, and a broken
renderer or editor is the commonest way that job goes red; after the bind it
would be reported two and a half minutes later, having built a binding nobody
was going to use. The pass at the end adds the gate tests, the Compose census
and the JVM harness, none of which `assembleDebug` can say, plus the app stage
the early step could not do.

**Not verified:** the workflow itself. The YAML parses and every command in it
was run here, but nothing confirms it on a GitHub runner until it is pushed.

---

## Item 5 — re-profiled, and declined with the new number

The item was gated on a device pass that just moved to `need_hardware.md`. What
was *not* gated is the payload, which is the same number on every machine — and
it had moved.

`TestWhatWindowingWouldSave` printed a table taken at 51,242 bytes and asserted
nothing. The chapter collapse took the screen to 17,366 the session before, and
the table did not change and did not complain.

```
 n  child              bytes   cumulative     sent   % sent
 1  title                459          459      681     3.9%
 2  progress card        696        1,155    1,377     7.9%
 3  chapter-0 card     5,872        7,027    7,249    41.7%
 4  chapter-1 band     1,436        8,463    8,685    50.0%
 5  chapter-2 band     1,427        9,890   10,112    58.2%
10  chapter-7 band     1,429       17,144   17,366   100.0%
```

The fold has not moved — children 1 to 3 occupy the pixels they always did,
because chapter-0 is the open card — so the window is still n=4, with n=5 as one
card of overscan. What moved is the prize:

```
                     payload    a window of 4-5 removes
before the collapse   51,242    33,800 - 38,900 bytes   (66-76%)
today                 17,366     7,254 -  8,681 bytes   (42-50%)
```

A quarter. The seven shut chapters are bands of ~1,430 bytes where they were
cards of ~5,300 — **the collapse and the window were aimed at the same 45 lesson
rows, and the collapse got there first, with no protocol change.**

Priced on the emulator's own attribution (~377ms of parse-and-build for 51,242
bytes, so ~128ms for 17,366 if the parse is linear in length — the assumption
the 8ms floor was derived under), a window saves **54-64ms**. Still seven times
the floor, so the effect is real and that instrument would see it. It is 1.7% of
a 3,200ms launch where the first sizing made it 7.7%, and the cost is unchanged:
a visible-range event, a bootstrap guess, and placeholder children in four
renderers. And that emulator is the machine most favourable to the argument —
its `org.json` spends ~200ms on 51KB where iOS's parser spends 6ms on the same
tree, so a device makes the prize smaller, not larger.

Declined, in `non_goals.md`, with the shape fully written down (it needs no new
bridge surface: `core.OnHostEvent` already carries a name and a payload). What
would bring it back is a long list — forty lessons in one open chapter, or any
app on this framework with a genuinely long `core.List`. Windowing was always a
framework answer; what changed is that the tutorial stopped arguing for it.

**And the test asserts now.** The share at n=4, in a 40-60% band — the share
rather than the bytes, because the share is what the decision turns on and is
what stays still while lessons are added; a band rather than an equality,
because this is not a budget. Mutation-tested by re-expanding all eight
chapters, which reproduces the pre-collapse screen and reports 23.4% against the
24.1% the old table recorded.

---

## Item 7 — the safe half, and the proposal written down

The partial rewrite stays declined, and for the reason it was declined on:
leaving an element in place is only safe if it still describes its block, and
between two calls the *browser* has been editing this DOM. A full rebuild
normalises that away every time; a partial one would normalise what it rewrote.
Nothing here can test for the difference — the harness DOM has no Selection API
and no `contenteditable` behaviour at all.

What was taken is the half that carries none of that risk. The element count is
untouched — every element is still new on every call, so the normalisation
argument stands word for word — but the *insertions into the live tree* are not:

```
blocks x runs      appendChild onto an attached node
   1 x 1                          1
 100 x 6                        101      (100 blocks + their <ul>)
2000 x 6                      2,001
```

and it is 1 for every row now. Each block's runs already went into a detached
box; what did not was the box itself and the `<ul>`/`<ol>` a run of list blocks
shares. Both go into a `DocumentFragment`, which is spliced in by one
`appendChild` — the one case where that call is not "put this node here".

**Not for layout.** Browsers batch layout and nothing here reads geometry back,
so there was never a forced reflow per block to remove. For the editing
machinery: `el` carries `contenteditable`, so the browser's own editing
implementation is watching this subtree, and so is any MutationObserver an
embedder attached. Building incrementally showed both of them ~n intermediate
states of a document mid-edit — partially rewritten, briefly missing every block
after the one being appended. The clear moved to the end for the same reason:
until the fragment is ready, `el` still holds the document the reader was
looking at.

The harness DOM grew a `DocumentFragment` — the 27th member it models. Modelled
rather than approximated, because a harness that appended a fragment as if it
were an element would put a node in the tree that no browser ever shows, and
every shape assertion in that file would be describing a document the browser
does not have.

Two tests. One counts `appendChild` calls landing on a node attached to the
document (attached, not "on the editor": an append into a `<ul>` already in the
editor is just as visible), asserted as the constant 1 at 1, 100 and 2000
blocks — because the regression to guard is not a slowdown but one more
`el.appendChild` inside the loop, which every other test in that file would go
on passing since they all assert shape and the shape is identical either way.
Mutation-tested by putting the blocks back into `el`: 14 failures. The other
checks the tree is unchanged and that no fragment survived into it.

---

## What is checked, and what is not

- `go test ./...` and `go test -race ./...` — green, **and green on a fresh
  clone**, which is the condition both CI failures needed.
- `wasm/verify/run.sh` — green here and on the fresh clone; 2 new richtext
  cases.
- `ios/verify/run.sh` — green.
- `android/verify/run.sh` — green, and green from a gradle cache that started
  empty and was filled by the two documented fetch tasks.
- Gates: 2 mutations against the new go arm, 1 against the citation fix, 1
  against the windowing share, 1 against the live-append budget. All caught.
- **Not checked:** the CI workflow on a GitHub runner. Every command in it ran
  here; the YAML parses; nothing confirms the runner until it is pushed.
- **Not checked:** anything needing a real caret, keyboard, IME or device. See
  `ai_docs/plans/need_hardware.md`, which is now where that list lives.

---

## Next

1. **(new · value high) The CI workflow's three new Android steps have never
   run on a runner.** Every command in them was run here and the YAML parses,
   but the gradle cache warms differently on a hosted image and
   `setup-java`'s cache may restore a `GRADLE_USER_HOME` that changes which
   arm the gates take. Watch the first push; the failure mode to expect is a
   SKIP where a run was intended, which is quiet.
2. **(new · value medium) Nothing checks that a gate's remedy sentence works.**
   Both compiler skips named tasks that do not fetch a compiler, for as long as
   they have existed, and `gate_test.sh` could not have known — it proves the
   sentences differ, not that any is true. Three gates in this repo now carry
   remedy sentences (`android/verify` x2, the Compose census). The shape of a
   check is: strip the precondition, run the remedy, run the pass, expect not-a-
   skip. It needs a scratch `GRADLE_USER_HOME` and a network, so it is a script
   somebody runs, not a test.
3. **(new · value low) `retire`'s bound is conditional and stated, not
   enforced.** `git cat-file --batch` does not fork, so the Kill always reaches
   the pipe. If a forking child ever ends up on those pipes the fix is Setpgid
   at Start and a negative Kill; the paragraph above `retire` says so. Worth
   revisiting only if another command is put behind that reader.
4. **(carried · value low) Windowing's device number.** Declined this session on
   the payload (`non_goals.md`), so this is no longer a gate on anything — but
   if a forty-lesson chapter ever ships, the profile to re-take is
   `TestWhatWindowingWouldSave`, which asserts its own share now.
5. **(moved) A device pass on the two editors' focus commands.** Now
   `ai_docs/plans/need_hardware.md`.
6. **(moved) A device pass on all four editors.** Now
   `ai_docs/plans/need_hardware.md`.
7. **(moved) `launch.sh`'s numbers are one emulator's.** Now
   `ai_docs/plans/need_hardware.md`.
8. **(moved) `HeadingSensor.retryIfArmed` is unmeasured.** Now
   `ai_docs/plans/need_hardware.md`.
9. **(closed) `android/verify/sources.sh` does not run in CI-shaped
   environments.** Done this session. It found two red CI failures, a red pass
   from a missing Go, and two remedy sentences that did not work.
10. **(closed) Windowing `core.List` over the bridge.** Re-profiled and
    declined; see `ai_docs/plans/non_goals.md`.
11. **(closed) `richTextToDOM` rebuilds the whole document on every write.**
    The partial patch is declined in `ai_docs/plans/non_goals.md`; the detached
    build was taken and is pinned.
12. **(declined, non-goal)** Twenty-seven entries now — the two added this
    session are windowing and the rich-text block patch. See
    `ai_docs/plans/non_goals.md`, which has gained a pointer to its sibling
    `need_hardware.md`: a non-goal is decided, a hardware item is blocked, and
    the two read identically in a Next list while wanting opposite things from
    a reader.
