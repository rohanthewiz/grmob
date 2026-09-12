# Session: five Next items, and a gutter that took the author's padding

**Session ID:** session_01HbRxrY6havcS6djbXtapZN
**Date:** 2026-09-12, ~16:27
**Branch:** master

## Goal

Load the previous session (`2026-0912-1519-grmob-new-and-the-shorthands-a-plain-object-could-not-see.md`),
list its `## Next`, then do items 4–8 of it:

4. The CodeEditor gutter clears an author's left padding (unconfirmed; test first).
5. `FULL_STYLE` is kept by hand.
6. Re-check the CSSOM table in a browser automatically.
7. Scaffold hygiene (x/mobile churn, demo strings in the iOS project, GrMobUITests).
8. The shots host has no `Shutdown`.

Then `/sw`.

## 4 — The gutter bug was real

`syncCodeGutter` wrote `el.style.paddingLeft = ""` whenever line numbers were
off. It runs after the style pass, which had just assigned the author's
`padding` shorthand, so the `""` removed the left side of the author's Padding —
the overflowX bug's shape again. htmlout's `codeEditorPadding` writes nothing
when the numbers are off, so the two DOM targets disagreed.

- **Test first:** `codeeditor_test.mjs` "the gutter hands padding-left back to
  the author's Style when the numbers go" — mount with `Padding.Left: 7`,
  numbers off → `7px`; on → `3ch` (top untouched); off → `7px`; restyle to 9
  with numbers off, toggle on/off → `9px`. Against the unfixed runtime it failed
  on the first assertion (`''` vs `'7px'`).
- **Fix (`grmob-runtime.js`):** `applyStyle` records
  `el.dataset.basePaddingLeft = css.padding ? el.style.paddingLeft : ""` for a
  CodeEditor (same pattern as `baseDisplay`: refreshed on every style patch);
  the off branch of `syncCodeGutter` writes that value back instead of `""`.
  While numbers are on, the gutter's inset still wins, matching htmlout.
- **Sweep:** `totality_test.mjs`'s `NODE_TYPES` claimed to be "every node type
  the runtime draws" and was missing five from `tagForType`: `Switch`,
  `CodeEditor`, `RichTextEditor`, `MapView`, `Marker`. All added; all pass.
  With HEAD's runtime swapped back in, the erasure sweep fails with
  `CodeEditor on creation` — so the sweep now catches this bug by itself.

## 5 — `FULL_STYLE` held to the source

New test "FULL_STYLE is every Style field the mapping reads": slices
`styleFromGrMob` from its declaration to the first `\n    }\n`, strips `//`
comments (several comments mention `style.X`), collects `style\.([A-Z]\w*)`,
and asserts both directions, with a floor of 30 reads so a broken parse can't
pass.

- **Missing, now added:** Display (`"hidden"` — not `"none"`, which would mask
  a stale `display:flex` and never reach `visibility`), FlexDirection (`"row"`),
  Transition, Animation, MinWidth, MinHeight, MaxWidth, MaxHeight, WhiteSpace.
- **Stale, removed:** `Opacity` and `ObjectFit`. Neither is a `core.Style` field
  (confirmed with rg over `core/`) and nothing in the runtime reads them; they
  looked like coverage and were not.
- All sweeps pass with the extended Style (65/65 in totality + shorthand +
  codeeditor).

## 6 — Browser check 14: the CSSOM table replayed in Chrome

- The table moved from `shorthand_test.mjs` into `cssstyle.mjs` as exported
  `CSSOM_READS` (browser.mjs can't import a `_test.mjs`: node:test would
  register its tests). `shorthand_test.mjs` imports it; its header now points
  at check 14 instead of "paste it into a console".
- A 20th row added for the gutter sequence: `padding "1px 2px 3px 7px"` →
  `paddingLeft "3ch"` → `paddingLeft "7px"` reads `padding "1px 2px 3px 7px"`.
  Not hand-taken — check 14 confirmed it in Chrome.
- `browser.mjs`: header entry 14, body marker 14 (after check 13, before
  `finally`), one `evaluate` that runs every row on a fresh detached
  `div.style`, per-property failure messages, `asked.cssomRows`, and a clause in
  the OK tail ("20 sequences of inline-style assignments read their shorthands
  back the way cssstyle.mjs's table says").
- Tallies: opening sentence gains a fifth kind ("and one about how it reads a
  CSS shorthand back"), "Thirteen claims" → "Fourteen".
  `checknumbering_test.go`: `tallyByKind` regex takes five counts (fourth kind's
  phrase matched as `[^,]+`), `"fourteen": 14` added, "four kinds" → "five".
  `run.sh` prose updated (it also said "two about paint", stale; now three).
- **Proven able to fail:** changed the `overflow: "auto hidden"` expectation to
  `"hidden auto"`, ran `browser.mjs` → `FAIL … this browser reads overflow as
  "auto hidden", and CSSOM_READS … says "hidden auto"`. Restored and confirmed.

## 7 — Scaffold hygiene (`cmd/grmob/native.go`)

- **x/mobile churn:** `ensureGomobile` now runs `go get -tool` for
  `cmd/gomobile` and `cmd/gobind` (the bind package needs no directive; it is in
  the same module). A `tool` directive is a requirement tidy keeps — the same
  arrangement grmob's own go.mod uses.
- **URL scheme:** new `urlScheme(cfg)` = lower-cased application ID. Patches:
  Android `<data android:scheme="grmob" />`, iOS `CFBundleURLSchemes: [grmob]`
  and `CFBundleURLName: com.grmob.deeplink` → `<id>.deeplink`. No Swift/Kotlin
  code checks the scheme (only the manifest and project.yml name it).
- **iOS usage descriptions:** the four `NS*UsageDescription` lines are replaced
  (new `replaceLine` helper) with `"<Name> uses the camera when you allow it."`
  etc. Keys kept: iOS terminates an app that requests a permission without one.
- **GrMobUITests:** added to `iosShell.skip`; new `removeLines(from, through)`
  helper (prefix-of-trimmed anchors, because "test:" also occurs inside a
  comment) removes the target block (`# Simulator smoke test` … `- target:
  GrMobApp`), the scheme's `GrMobUITests: [test]`, and its `test:` action.
  Ordering comment in `iosPatches` explains why the target block goes first.
- **Android permissions left as they are**, deliberately: declarations carry no
  user-visible text, and removing one makes `permission.Request` answer
  "unavailable" — an app's choice, not a scaffold's.
- `TestShellPatchesStillApply`: expectations are now lists, plus a `refuse` map
  (`GrMobUITests`, `Demonstrates permission`, `[grmob]`, `com.grmob.deeplink`,
  `android:scheme="grmob"`), an upper-case ID to exercise the lowering, and the
  exact project.yml text that must remain after the UITests removal.
- `docs/platforms/native.md`: what the copy changes beyond the name, and the
  tool directives.

**End to end** (scratchpad app `hygiene`, `-replace` → checkout):
`grmob android` → APK built; `grmob ios` → xcodegen + simulator xcodebuild OK
with no UITests target; go.mod has the `tool` block and `go mod tidy` leaves it
unchanged; built `Info.plist` has `CFBundleURLSchemes = [com.example.hygiene]`
and "Hygiene App uses the camera when you allow it."

## 8 — Shots host `Shutdown`

`wasm/shots/host/main.go` gained `done`/`doneOnce`, a `shutdown` binding
mirroring wasm/main.go's (close manager, then release main), and `<-done` in
place of `select {}`. `webhost_test.go`'s `hosts` is now a plain path list and
every host must install exactly `webhost.Bindings`; the subset allowance was
removed with its reason. `GOOS=js go vet` clean; `go test ./webhost` ok.

## Verification at the end

- `go test ./...`: all ok.
- `wasm/verify/run.sh`: `EXIT=0` — Node suites and the headless-Chrome pass,
  including check 14 (20 rows held).
- Native end to end as above.

## Files

Changed: `wasm/grmob-runtime.js`, `wasm/verify/{codeeditor_test.mjs,totality_test.mjs,cssstyle.mjs,shorthand_test.mjs,browser.mjs,checknumbering_test.go,run.sh}`,
`cmd/grmob/{native.go,new_test.go}`, `docs/platforms/native.md`,
`wasm/shots/host/main.go`, `webhost/webhost_test.go`.

## Next

1. **(carried · value high) Tag a release.** `@latest` is v0.2.4, which has
   neither `cmd/grmob` nor the gap fix, so the README's first command fails on
   any other machine until a tag is pushed. Then run
   `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean machine
   without `-replace` — every end-to-end run so far (including this session's
   native build) used a replace to the checkout, so the module-cache path is
   exercised only by reasoning.
2. **(carried · value med) Launch the scaffolded app on a simulator and a
   device.** This session built Android and iOS again (now with the new scheme,
   strings and no UITests) but launched neither.
3. **(carried · value med) `grmob ios -run`.** Still prints a `simctl
   install/launch` line instead of doing it.
4. **(new · value low) The copied shells' comments still cite `grmob://`.**
   The manifest and project.yml comments show `grmob://lesson/4.12` examples
   next to a scheme that is now the app's ID. Harmless, misleading.
5. **(new · value low) The copied Android shell declares every demo
   permission.** Left deliberately (see §7); a `grmob android` note or a
   doctor-style hint could tell an app to prune what it does not request
   before a Play submission.
6. **(new · value low) `insertLineBefore` duplicates `uniqueLine`'s search.**
   Left as written; could be expressed through the new helper.
7. **(new · value low) The iOS usage strings are placeholders.** Documented in
   native.md as "rewrite before shipping"; nothing enforces it.
8. **(carried · value med) `docs/api/core.md` is one 7,700-line page.**
   Splitting would need the flat-sibling link scheme reconsidered.
9. **(carried · value med) Nothing checks links from `docs/api/` out into the
   narrative pages.** `TestEveryGeneratedLinkResolves` skips `../`
   destinations.
10. **(carried · value low) `aria/spec` in a user-facing reference.**
11. **(carried · value low) The skill's pointer URL is dead until merge.**
12. **(carried · value low) The Android remedy drill has never run.**
13. **(carried · value low) `hero.png` is a picture of pictures and its parts
    are held twice.**
14. **(carried · value low) A third face is still a skip.**
15. **(carried · value low) Windowing as a proposal.** `TestWhatWindowingWouldSave`
    is the profile to re-take.
16. **(moved) The four hardware items.** `ai_docs/plans/need_hardware.md`.
17. **(declined, non-goal)** Twenty-seven entries, unchanged. See
    `ai_docs/plans/non_goals.md`.

Done this session and dropped from the list: the gutter padding bug, the
hand-kept `FULL_STYLE`, the manual CSSOM table, scaffold hygiene (x/mobile,
iOS strings/scheme, UITests), the shots host's missing `Shutdown`.
