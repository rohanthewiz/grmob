# Session: a README with no images, and a lesson count that said 42, 49 and 51

Session: https://claude.ai/code/session_01BxZEsiUvkVxKKe3fZsRN8y
Date: 2026-09-12 · Previous:
`2026-0912-0201-five-open-items-and-a-quarter-that-was-one-faces-number.md`

## Ask

> Rework the README to include examples with screenshots providing a gentle
> intro to GrMob. At the end point to the full tutorial for more.

> fix the 49 lesson counts in index.html and the tutorial doc

> yes, fix those two

The repository had no images at all — not one `.png`, `.jpg` or `.svg` under
any directory. "Include screenshots" was therefore not a writing task with a
lookup in front of it; the screenshots had to be produced before the README
could reference them, and producing them meant driving the apps.

The four sentences worth keeping:

1. **A framework that renders to the DOM can photograph itself.** The browser
   build is already the honest preview — `wasm/main.go` dot-imports one
   example and the host wiring is the same the deployed site uses. Swapping
   that one import is the whole mechanism for pointing a camera at a
   different app.
2. **The apps were tapped, not posed.** Every screenshot's state was reached
   through `GoInvokeCallback` — the same entry point `grmob-runtime.js` calls
   — so the todo list has four rows because four titles were typed and
   committed, and the signup form shows one error because a password was
   mistyped by one character. Nothing was seeded, faked, or hand-edited into
   the DOM.
3. **The lesson count was wrong in three different directions at once.** The
   README said 42 in its first line and "Forty-nine" further down; the site
   page said 49 in two places; two Go comments said 49. The app says 51. Six
   sentences, each written while it was true.
4. **One of the seven 49s was correct and had to stay.** The distinction is
   tense, not value: `app_test.go:357` says the contents screen *used to*
   list all 49 lessons at once, which is a fact about the era before the
   cards collapsed. Rewriting that one to 51 would have been the only edit in
   the set that introduced an error.

---

## Part 1 — the screenshot harness

### What existed to build on

`wasm/main.go` is host wiring only: JS bindings, a `render.Manager`, the
event bridge. The app it mounts is a dot-import, and the file says so:

```go
// The mounted app. Point this dot-import at any package in examples/
// that exports an App root view — examples/social was the previous
// occupant — and rebuild to switch what the browser shows.
. "github.com/rohanthewiz/grmob/examples/tutorial"
```

That comment is the harness's whole design. `.shots/shoot.sh` rewrote the
import with `sed`, built to a private `www/`, and left `wasm/` untouched:

```sh
.shots/shoot.sh todoapp             # mount examples/todoapp
.shots/shoot.sh --local counter.go  # mount an App from a standalone file
```

The `--local` arm exists because of a Go-tool detail. A directory whose name
begins with `.` is invisible to `./...` but buildable when named directly, so
`./.shots/host` compiled — but `github.com/rohanthewiz/grmob/.shots/counter`
is not a legal import path. The counter therefore could not be a sibling
package; it had to be a second file *in* the host package, with the
dot-import dropped by `grep -v`.

### The page

A copy of `wasm/index.html` with the tutorial header, the deep-link
translation layer and the Leaflet tags removed — the phone bezel, the `#app`
mount rules, and the `Scroll` overflow rule that makes the node a viewport.
Two additions:

- `?h=` / `?w=` set the bezel's size, so a three-control counter is
  photographed in a short frame instead of a screenful of blank paper.
- `window.__grmobReady = true` after `GrMob.mount`, because `main.wasm` is
  ~7MB and the first paint is the "Loading…" placeholder. Waiting on `load`
  would have photographed that.

### The driver

`shot.mjs`, a CDP client over Node 22's built-in `WebSocket`, launching the
Chrome that `wasm/verify/browser.mjs` already knows how to find. Two
decisions carried the image quality:

- **Clip to the element, not the viewport.** `Page.captureScreenshot` with
  `clip` set to the `.phone` bounding rect means the window's own size never
  reaches the PNG, so every shot is framed identically regardless of what
  Chrome decided the window was. The MCP browser tools downscale their
  captures (1745 CSS px arriving as a 1311px JPEG); CDP at `scale: 2` does
  not.
- **`--hide-scrollbars`.** A scrollbar inside the bezel reads as app chrome
  rather than as the browser's.

An action script runs between mount and capture, with a prelude of helpers
that speak the runtime's own vocabulary:

| helper | attribute it reads | payload |
|---|---|---|
| `tap` | `data-listener_on-click` | `{}` |
| `typeInto` | `data-listener_on-change` | `{value}` |
| `toggle` | `data-listener_on-toggle` | `{value: bool}` |
| `selectTab` | `data-listener_on-tab-change` | `{value: index}` |

`tap` needed one correction that is worth writing down: a container's
`textContent` equals its only child's, so `byText("Add")` matched the `Row`
wrapping the button as readily as the button. The fix is to prefer the
candidate that actually carries the handler, which is always the innermost
one.

`out == "-"` turns a shot into a probe — the action script's return value is
printed instead of a PNG being written. That is how the DOM was read at all,
because `javascript_tool` against the same page returned
`[BLOCKED: Cookie/query string data]`.

### What was photographed

Seven files, 708K, in `docs/images/`:

| file | app | state it was driven to |
|---|---|---|
| `counter.png` | a standalone `App` | three taps of `+` |
| `todo.png` | `examples/todoapp` | four titles typed and added, two ticked |
| `signup.png` | `examples/signup` | filled, one transposed character in confirm, submitted |
| `tabs-list.png` | `examples/mobileapp` | Feed tab, row 3 selected |
| `tutorial-contents.png` | `examples/tutorial` | as it opens |
| `tutorial-lesson.png` | `examples/tutorial` | lesson 1.1, scrolled to the TRY IT panel |
| `hero.png` | — | the three app shots composited |

`examples/social` was tried and dropped: it is a Portuguese-language
navigation demo (`🏠 Página Inicial`), not the profile screen the old
README's example code described. That code does not correspond to anything
in the repository, and it is the one section of the old README that this
rework did not carry forward.

### The counter has no home

`docs/getting-started.md` shows a counter; no `examples/counter` package
exists. The README's first app is therefore complete and compiling but
lives only in the README. It was verified rather than asserted:

```
$ diff -w scratch/counter.go readme-extracted.go
$ gofmt -e readme-extracted.go > /dev/null
```

Whitespace-only difference (the README's code blocks use spaces, as they
did before), and it parses. The screenshot is of that source.

---

## Part 2 — the count

### What the app says

Driven rather than counted by hand, through the same probe mechanism:

```
{"total": "51", "sum": 51, "chapters": [
  {"n":1,"title":"Views & Layout","lessons":5},
  {"n":2,"title":"State, Events & Lists","lessons":6},
  {"n":3,"title":"Hooks & Effects","lessons":5},
  {"n":4,"title":"The Widget Library","lessons":14},
  {"n":5,"title":"Forms & Validation","lessons":6},
  {"n":6,"title":"Navigation & Overlays","lessons":5},
  {"n":7,"title":"Theming & Styling","lessons":5},
  {"n":8,"title":"Robustness","lessons":5}]}
```

The total and the sum of the parts agree, and the tests derive theirs from
`len(flatLessons)` — which is exactly why nothing caught the prose. The
count is computed everywhere it is *checked* and transcribed everywhere it
is *stated*.

### The seven sentences

| where | said | now |
|---|---|---|
| `README.md` header | 42 | 51 |
| `README.md` tutorial section | "Forty-nine" | rewritten |
| `wasm/index.html:7` (meta description) | 49 | 51 |
| `wasm/index.html:143` (page header) | 49 | 51 |
| `docs/tutorial-interactive.md:5` | 49 | 51 |
| `core/deeplink.go:47` | 49 | 51 |
| `examples/tutorial/app_test.go:682` | 49 | 51 |
| `examples/tutorial/app_test.go:357` | 49 | **unchanged** |

The meta description was not in the ask — it was found by grepping the file
rather than editing the named line, and it is the one a search result or a
link preview shows.

`app_test.go:357` is the counter-example to the whole exercise:

> It used to list all 49 lessons at once and this test used to say so. The
> cards collapse now, so the claim splits in two…

A sentence about a past state, correct as written, sitting three characters
away from the same digits that were wrong everywhere else.

### The claim next door

`core/deeplink.go:47` states two facts in one sentence — "The tutorial is 49
lessons and its map lesson is 4.12". Editing the first is an occasion to
check the second, since a tutorial that grew by two lessons could have
renumbered chapter 4. It did not: chapter 4's fourteen lesson titles
enumerate to 4.1 Buttons … 4.11 Maps: a picture and a hand-off, **4.12 Live
maps: markers and the echo guard**. That agrees with the `grmob://lesson/4.12`
invocations below it, with `core/location.go:77`, and with the iOS sensor
and map-view files. Only the count was stale.

---

## The test that caught the harness

`TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn` failed on the
first full run of the session, reporting 442 tracked Go files against five
sentences that say 441. `git ls-files '*.go'` says 441. The discrepancy was
`.shots/host/main.go` — the walk counts untracked `.go` files too.

Moving the harness out and re-running turned it green, which established
that the baseline was clean and that the failure was self-inflicted. Worth
knowing as a property of the walk: **any scratch `.go` file anywhere in the
tree fails that test**, and its message will name five innocent sentences
in `repowalks_test.go` and `timings_test.go` as the thing to edit. The
harness was deleted before the final verification for exactly this reason.

Final state: `go vet` clean on the two edited packages, `go test ./...`
exit 0, 30 packages ok.

---

## The README

Reordered from a feature list into a progression: hero → a complete
twenty-line counter with its screenshot → how to run it → a three-screen
tour (lists and derived state, forms and reveal policies, tabs and widgets)
→ conditional rendering and events → the render-loop diagram → the four
targets → the reference material → the tutorial, with a chapter table, at
the end.

Nothing was dropped except the `examples/social` profile snippet noted
above. The verify-harness section, the ARIA fixture explanation, the
architecture map and the hooks list are all still there, after the
introduction rather than interleaved with it.

Every relative link and image path in the finished file was resolved against
the filesystem — 22 of them, all present.

---

## Next

1. **(new · value medium) The screenshots are unpinned prose in image
   form.** They are facts about seven screens, written down while true, with
   nothing that reads them — the same shape as the six lesson counts this
   session fixed, and the repository's whole argument is that such a number
   is worth nothing. A widget restyle, a theme change or a renamed label
   silently makes `docs/images/` a picture of a version that no longer
   exists. The cheap arm is a test that asserts the *text* each shot claims
   to show still appears in that app's rendered tree ("What needs doing?",
   "The two passwords differ", "0 of 51 lessons opened"), which would fail on
   the drift without anyone re-photographing anything.
2. **(new · value medium) The harness that took them is gone.** It was
   deleted rather than committed because an untracked `.go` file in the tree
   fails the shared-parse walk (above), and `.shots/` would have needed a
   `.gitignore` entry plus a decision about whether `shot.mjs` belongs beside
   `wasm/verify/browser.mjs`. As it stands, re-taking a screenshot means
   rebuilding the import-swap script, the bezel page and the CDP driver from
   this document. The mechanism is worth ~150 lines; the question is whether
   it lives in `wasm/` as a sibling of the verify driver, which already
   finds Chrome and speaks CDP.
3. **(new · value low) The README's first app has no package.** The counter
   compiles and is what `counter.png` shows, but it exists only inside a
   fenced block, and `docs/getting-started.md` carries a near-identical one.
   An `examples/counter` would make the first screenshot reproducible by
   `./build.sh` and give the two documents one source. Declined in-session as
   scope; the reason to want it is that a new example is the one kind of file
   the repowalk tests actively check.
4. **(carried · value medium) `INK_STEM_REACH` is a floor with one side
   measured.** 0.8 sits under a single reading of 0.907 and over nothing at
   all. The other bracket would come from painting a count in a dimmer
   version of its own declared ink and reading what the scan says — a
   fixture, not a grid change. The tail recites the thinnest stem on every
   run, so the margin cannot be spent silently in the meantime.
5. **(carried · value low) A third face is still a skip, and now a cheaper
   one.** Any machine resolving something other than Times or Liberation
   Serif gets the skip plus the table it needs. What nobody has is a face
   where the readings do NOT clear — the report's "WHICH DOES NOT SEPARATE"
   arm has fired exactly once, on Liberation Serif under the old fractions,
   and is now unreachable by any face this project has met.
6. **(carried · value medium) The toolchain gates' remedy sentences are
   still unexecuted.** `android/verify`'s Gradle-cache remedies and
   `ios/verify`'s SDK one. The ink gate's was drilled two sessions ago and
   the shape generalises — strip the precondition, run the remedy, run the
   pass, expect not-a-skip — but those two need a network, a Gradle cache and
   an SDK, so it is a CI job rather than a test.
7. **(carried · value low) Windowing as a proposal.** Declined on the
   payload in `non_goals.md`; `TestWhatWindowingWouldSave` asserts its own
   share and is the profile to re-take if a forty-lesson chapter ever ships.
   The device number it was once waiting on is `need_hardware.md`'s. Not a
   Next item any more except as this line.
8. **(moved) The four hardware items.** `ai_docs/plans/need_hardware.md`.
9. **(declined, non-goal)** Twenty-seven entries, unchanged. See
   `ai_docs/plans/non_goals.md`.
