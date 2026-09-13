# Next-list sweep: disabled radios, a code toolbar, and a reference without aria/spec

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-12 20:44
**Branch:** master

## 1. The asks

1. `/sl` — load the Tier B session doc.
2. "Before implementing Tier C, let's address what items we can in the Next list."
3. Decisions on the items handed back: "#12 don't rename docs/components.md.
   #13 only update the two most recent plans that say components. #22 yeah move
   the aria/spec out of the user-facing ref. Carry over #25 and #26. Finally one
   commit is fine."
4. `/sw`.

## 2. Commits

| Commit | What |
|---|---|
| 0e028e9 | Next-list sweep: disabled radios, a CodeEditor toolbar, ios -run, and a reference without aria/spec |

`go test ./...`, `wasm/verify/run.sh` and `node --test wasm/verify/keynav_test.mjs`
(72 tests) green before the commit.

## 3. What was addressed (previous doc's Next numbering)

### #5 Arrowing onto a disabled radio — fixed in the runtime
- `compositeMembers` in `wasm/grmob-runtime.js` now drops a member whose role is
  `radio` and which carries `aria-disabled="true"`. A listbox still keeps its
  aria-disabled options reachable: landing on an option chooses nothing, but
  `moveCompositeFocus` always checks a radio it lands on, so a disabled radio
  was a stop whose check Go refused. The walk's header table gained the row.
- New `keynav_test.mjs` test "an arrow steps over a disabled radio" (focus and
  `cb_2` dispatch skip the middle radio; it gets no tabindex).
- Chrome, lesson 4.16 (Shipping step): Standard → Express → wraps to Standard;
  "Pick up in store" has `aria-disabled="true"` and no tabindex.

### #7 CodeEditor's toolbar — now a toolbar
- `comps/code_editor.go`: the command row carries `core.RoleToolbar` and
  `AccessibilityLabel(orDefault(c.ToolbarLabel, "Editing"))`; new
  `CodeEditor.ToolbarLabel`. Doc comment on `toolbar()` says why a closed widget
  may declare it (members built by the widget, no `core.View` field).
- `comps/nested_composite_test.go`: `code_editor.go` added to `closedComposites`.
- `docs/components.md` CodeEditor section: "The toolbar is a toolbar." paragraph.
- Chrome, lesson 4.13: one `role=toolbar` named "Editing", three buttons, one
  tab stop; ArrowRight moves focus and stop from Indent to Outdent.

### #6 Stale prose
- `docs/components.md` ChipStrip paragraph now says the runtime supplies the
  toolbar keyboard once the caller declares the role.
- `comps/list_row.go` no longer says "nine of core's twenty roles"; it points
  at `core.Style`'s AccessibilityRole table instead of carrying a count.

### #3 Old host CSS in scaffolded apps
- README `## Upgrading` gained "Apps scaffolded with v0.3.0: two host-page
  rules" (above the v0.3.0 rename section) with both CSS rules and why a
  horizontal Scroll collapses without them. No refresh command was added.

### #15 `grmob ios -run`
- `cmd/grmob/native.go`: new `-run` flag (variable `launch`, since `run` is the
  exec helper) runs `xcrun simctl install booted <app>` then
  `xcrun simctl launch booted <id>`; install failure wraps a hint
  (`xcrun simctl list devices booted`, `open -a Simulator`). Without `-run` the
  manual line is still printed. "booted" rather than choosing a device, so no
  model/runtime is guessed and no silent slow boot happens.
- README quick-start line and `docs/platforms/native.md` mention `-install` /
  `-run`. **Not executed** — needs a full `grmob ios` build and a simulator.

### #16 Copied shells' comments cite `grmob://`
- `androidPatches`: `-d "grmob://lesson/4.12"` → `-d "<scheme>://"`.
- `iosPatches`: "OS hands it grmob:// links." and
  `openurl booted "grmob://lesson/4.12"` rewritten to the app's scheme (lesson
  path dropped — it is the demo's).
- `TestShellPatchesStillApply` refuse lists now include `grmob://` for both.

### #18 `insertLineBefore` duplicated `uniqueLine`
- Now calls `uniqueLine` with a contains-matcher. Error wording changes to
  `uniqueLine`'s; nothing tested the old wording.

### #21 Links from `docs/api/` into the narrative pages
- `TestEveryGeneratedLinkResolves` (`internal/apidoc/apidoc_test.go`) follows
  `../` links: resolved with `path.Join` against the page's dir, must stay under
  `docs/`, file must exist, fragment must match a heading id from `headings()`
  on that file (cached per page). Only one such link exists today
  (`index.md` → `../concepts/architecture.md`), and it resolves.

### #22 `aria/spec` in the user-facing reference — removed
- Entry dropped from `apidoc.Packages`; the "Left out deliberately" comment gains
  `aria/...` with its reason; `excluded()` gains `rel == "aria" || aria/...`.
- `docs/api/aria-spec.md` deleted, `mkdocs.yml` nav line removed, `index.md`
  regenerated (13 pages, no "Tools" group). Two comments in `packages.go` that
  used "aria/spec" as the nested-dir example now say "a/b".

### #13 Plans still saying `components`
- The two most recent plans that name the *package*: `non_goals.md` (5 lines)
  and `components-editors.md` (11 lines). `comps-low-hanging-fruit.md` is more
  recent but its hits are all still-valid names (`docs/components.md`, a plan
  filename, `examples/components.go`).
- Rewritten: `components.X` → `comps.X`, `components/<file>.go` →
  `comps/<file>.go`, `` `components` `` → `` `comps` ``. Kept:
  `docs/components.md`, plan filenames, `examples/components.go`,
  `"components:link"`, "Next components" titles. Older plans left as history.

### #23 The skill's pointer URL — already live
- `https://raw.githubusercontent.com/rohanthewiz/grmob/master/ai_docs/SKILL.md`
  returns 200.

### Decided, not done
- #12: `docs/components.md` keeps its name (URL stability) — now a non-goal.
- #17 / #19: already deliberate in `native.go` (Android declarations carry no
  user text and removing one makes `permission.Request` answer "unavailable";
  iOS keys must exist or the app is terminated, and the strings are already
  rewritten to name the app). Recorded as non-goals.

## 4. Verification notes

- Shots harness in the scratchpad: `GOOS=js GOARCH=wasm go build -o
  <www>/main.wasm ./wasm/shots/host`, copy `wasm/shots/index.html`,
  `wasm/grmob-runtime.js`, `$(go env GOROOT)/lib/wasm/wasm_exec.js`, then
  `node wasm/shots/shot.mjs --dir <www> --out - probe.js`.
- **Gotcha:** the tutorial's contents screen shows only chapter 1, so
  `tap("<lesson title>")` fails for later chapters. Open a lesson with
  `GrMobWASM.HostEvent("route", JSON.stringify({ lesson: "4.16" }))` — the
  payload must be a **JSON string**; a plain object reaches Go with no
  `lesson` field and nothing happens. `scrollTo` matches visible text only, so
  an aria-label like "Indent" is not findable with it.
- Lesson 4.16's RadioGroup renders only on the Shipping step: `tap("Next")`
  first.

## Next

1. **(carried, partly done · value high) Tag a release.** v0.3.0 is tagged.
   Tier A + Tier B + radio roles + this sweep are a natural v0.4.0. Still open:
   run `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean
   machine without `-replace`.
2. **(carried, widened · value high) Run lessons 6.6, 6.7, 4.15 and 4.16 on a
   simulator and a device.** Unverified on hardware: settings-row one-tap,
   `Screen.Footer` pinning, iOS sheet `Dialog`, stepped spinner, ActionSheet
   filler placement on Compose and SwiftUI, Timeline row stretch, horizontal
   StepIndicator scroll, the radio semantics (Compose
   `selectableGroup`/`RadioButton`), and CodeEditor's toolbar role on Compose.
3. **(new · value med) Exercise `grmob ios -run`** on a machine with a booted
   simulator, and once with none booted to read the hint.
4. **(new · value med) Tier C of the plan:** C1 `Menu` (ActionSheet with a
   trigger slot), C3 `SearchableSelect` (focus behaviour first), C4 `Carousel`
   stays blocked on a scroll-offset signal.
5. **(carried · value med) Launch the scaffolded app on a simulator and a
   device.**
6. **(carried · value med) `docs/api/core.md` is one 7,700-line page.**
7. **(carried · value low) Migrate hand-rolls to the new comps:** the
   tutorial's `stepper` helper, chapter 6.4's and chapter 1's `checkRow`,
   confirm flows in `todoapp`/`mobileapp`, and the signup flow's step header.
   Retakes screenshots.
8. **(carried · value low) A looping `Transition`** so `Spinner` can drop its
   stepping.
9. **(carried · value low) `BottomBar` current-item semantics.** ", selected"
   (also StepIndicator's ", done"/", current") is an English fallback; an
   `aria-current`-style state in core would be the proper shape.
10. **(new · value low) No `grmob` command refreshes a scaffolded app's
    `wasm/index.html`.** The README note covers the Scroll rules; the next host
    CSS change will need another note unless one exists.
11. **(carried · value low) The Android remedy drill has never run.**
12. **(carried · value low) `hero.png` is a picture of pictures and its parts
    are held twice.**
13. **(carried · value low) A third face is still a skip.**
14. **(carried · non-goal) Tracked-Go-file count sentences** in `wasm/verify`
    are hand-edited on every commit that adds Go files, and count untracked
    files. Working as designed.
15. **(non-goal) Rename `docs/components.md` to `comps.md`.** Declined this
    session: the page keeps its name for URL stability.
16. **(non-goal) Rewrite `components` in the older plans.** Only the two most
    recent were updated; the rest stay as history.
17. **(non-goal) Trim the copied Android shell's permission declarations.**
    Removing one is a build-time choice the app should make; see
    `androidPatches`.
18. **(non-goal) Replace the iOS usage strings further.** They already name the
    app; the app must say why before shipping (`docs/platforms/native.md`).
19. **(non-goal) Windowing.** Declined in `non_goals.md`;
    `TestWhatWindowingWouldSave` is the profile to re-take.
20. **(non-goal) Drawer (plan C2)** stays skipped until asked for; `BottomBar`
    covers the same navigation need with no overlay.

Closed since the previous doc: "Apps scaffolded before 1397c8d have the old
host CSS" (README note), "Arrowing onto a disabled radio", "`docs/components.md`
ChipStrip paragraph is stale", "CodeEditor's toolbar", "`grmob ios -run` prints
a simctl line", "The copied shells' comments still cite `grmob://`",
"`insertLineBefore` duplicates `uniqueLine`'s search", "Nothing checks links
from `docs/api/` out into the narrative pages", "`aria/spec` in a user-facing
reference", and "The skill's pointer URL is dead until merge".
