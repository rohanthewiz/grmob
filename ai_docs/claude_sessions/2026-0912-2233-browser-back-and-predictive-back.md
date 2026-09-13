# Browser back, predictive back, and a stale double back

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-12 22:33
**Branch:** master

## 1. The asks

1. `/sl` loaded the "Android system back" session doc.
2. "Let's do item 25, browser back to Navigator along with the two other new
   items on the list": 25 (browser back), 26 (predictive back) and 27 (stale
   callback ID on a double back). Items 28 and 29, also new, are non-goals.
3. `/sw browser-back-and-predictive-back`: save this doc, commit all, push.

## 2. Item 25: browser back runs core.OnBack

### Design: one history entry owned by the runtime

`onBack` was a no-op on the web. It is now a *claim*, and the runtime keeps
history in step with whether any claim is on screen:

```
no claim on screen      history: [ … page ]
a claim appears         history: [ … page, runtime ]   pushState({grmobBack:true})
back pressed            → popstate                     run the innermost claim
claim still on screen   history: [ … page, runtime ]   pushState again
claim ends another way  history.back()                 its popstate ignored,
(an in-app ‹ button)                                    the page's URL carried down
```

- **One entry, not one per Push.** The runtime only knows what the tree says,
  which is enough: back runs the innermost handler, and Go decides whether
  another claim follows. It also stops Forward from replaying a frame whose
  state Pop already discarded.
- **Innermost is the last claimant in document order.** That is Compose's
  registration order for one pass: parent before child, and a later layer such
  as a Drawer panel after the screen it covers. A claimant under `display: none`
  doesn't count, because Compose doesn't compose a hidden node.
- **An open Modal's `onDismiss` counts as a claim**, matching Android's Dialog.
- **The entry is recognised by `history.state.grmobBack`**, which survives a
  reload and a hot reload where a variable would not. So a page that rewrites
  its URL must pass `history.state` through: `wasm/index.html` passed `null`,
  and now doesn't.
- **Opt-out:** `window.GrMobBrowserBack = false` before mount. The runtime then
  never touches history or listens for popstate.

### Two bugs found only in the browser

Both passed the Node harness and showed up in Chrome.

1. **`update-props` with `Changes: null`.** `reconcile/patch.go` sends
   `Changes: new.Props`, and a nil Go map marshals as `null`. Back from a
   lesson (root carries only `onBack`) to the contents screen (root carries
   nothing) was the first transition to send one. `applyEnterKeyHint` threw on
   `null.imeAction`, the batch died partway, and the lesson stayed on screen.
   The fix is a guard at the top of the runtime's `update-props` case. Kotlin
   and Swift already fall back to an empty map.
2. **The unwind landed on a stale URL.** On a deep-linked boot (`#2.3`), the
   page's entry and the runtime's entry share the lesson URL. "‹ Contents"
   cleared the hash on the runtime's entry, and the runtime's `history.back()`
   then landed on the page's entry, still at `#2.3`. The traversal fired
   `hashchange`, and the tutorial re-opened the lesson.

   The fix: the runtime records `location.href` before unwinding and, in the
   unwind's popstate, `replaceState`s it onto the entry it landed on. That runs
   before the traversal's `hashchange` is handled, so the listener reads the
   carried URL.

### The tutorial's lessons claim back themselves

Navigator's pop left `t.current` naming the lesson. The address bar kept its
hash, and a link to the same lesson was ignored as "already on screen". This
was also true of Android back since the last session. `lessonRoute` now wraps
`comps.Screen` in a `ComponentFunc` and applies
`core.OnBack(func() { t.toContents(ctx) })` to the node Screen renders. A claim
on the route root replaces Navigator's, and Screen builds a fresh node every
pass, so writing into it is safe.

## 3. Item 27: back handlers get their own ID sequence

The second of two quick presses is dispatched with the ID the host still holds.
With back handlers numbered among all void callbacks, the pass the first press
caused re-assigns that ID, usually to a tap on the revealed screen:

```
pass 1  lesson     cb_7 = Navigator's Pop (onBack)
press 1 → Pop → pass 2
pass 2  contents   cb_7 = the eighth row's onClick
press 2 → cb_7 (stale) → opens a lesson
```

`callbackRegistry.registerBack` now mints `back_cb_N` into `voidCBs` from its
own `backCounter`. Every host dispatches it through the ordinary void path, so
nothing in Kotlin, Swift or JS changed for this. A stale back ID can only reach
another back handler at the same position (on a three-deep stack, two presses
pop twice) or nothing, once it's purged. `registrationCount`, `beginPass`, and
the ErrorBoundary snapshot and rollback all include the new counter.
`core.OnBack` no longer goes through `On("Back", …)`, but it writes the same
`onBack` key.

## 4. Item 26: predictive back

`android:enableOnBackInvokedCallback="true"` on `<application>`, with a
comment. Every back path in the shell is androidx's (`BackHandler`, and
Compose's Dialog for a Modal), and there is no `onKeyDown` or `onBackPressed`
override that the flag would silence. `-refresh` carries the manifest to
scaffolded apps, and `androidPatches`' `replaceOnce` targets are unaffected.

## 5. What changed

| File | Change |
|---|---|
| `core/event.go` | `backCounter`, `registerBack` (doc with the collision diagram), `registerBackCallback`, and the counter in beginPass, registrationCount and snapshot/rollback |
| `core/behavioral_props.go` | `OnBack` registers through `registerBackCallback`. New "Its own callback IDs" section; Web and Android host notes rewritten |
| `core/navigation_test.go` | `backPass` helper, `TestAStaleBackIDCannotRunTheRevealedScreensTap`, `TestTwoQuickBacksOnADeepStackPopTwice` |
| `wasm/grmob-runtime.js` | Browser-back section above `mount` (claimants set, `backClaimID`, document-order compare, `syncBrowserBack`, `onBrowserBack`, URL carry). `onBack` branches on both paths record the claim; `attachModalDismiss` records one; sync at the end of `mount` and `patch`; null `Changes` guard |
| `wasm/index.html` | `replaceState(history.state, …)` |
| `wasm/verify/browserback_test.mjs` | New: 9 tests over a history model with URLs and a deferred `back()` |
| `mobile/verify/back_test.go` | Renamed `TestSystemBackIsWiredOnComposeAndTheWeb`. Pins `registerBackCallback`, the popstate listener and `backClaimants.add` |
| `examples/tutorial/lesson_screen.go` | Lessons claim back through `toContents` |
| `examples/tutorial/deeplink_test.go` | `TestSystemBackOnALessonTakesTheContentsDoor` |
| `examples/tutorial/chapter4.go`, `chapter6.go` | Key points name browser back |
| `android/app/src/main/AndroidManifest.xml` | Predictive back opt-in |
| Docs | `navigation.md` ("Browser back", "Two quick presses"), `events.md` (System back, the stability note), `wasm.md` ("Browser back" under the host-page contract), regenerated `docs/api/core.md` |

## 6. Verification

- `go test ./...` passes, and so does the full Node runtime suite (fresh
  transcript), including the 9 new browser-back tests.
- **Chrome** (`./build.sh`, `go run ./serve -addr :8093`), fresh load at
  `?v=3#2.3`, with the served runtime confirmed to contain the fix:

  | Step | history.state | hash | Screen | Claims |
  |---|---|---|---|---|
  | boot `#2.3` | grmobBack | `#2.3` | lesson | 1 |
  | ‹ Contents | null | `""` | contents | 0 |
  | route `#4.18` | grmobBack | `#4.18` | lesson | 1 |
  | ☰ opens drawer | grmobBack | `#4.18` | lesson + drawer | 2 |
  | back | grmobBack | `#4.18` | lesson, drawer closed | 1 |
  | back | null | `""` | contents | 0 |

- **Emulator** (Medium_Phone_API_36.1, gesture nav): `android/build.sh
  ./examples/tutorial`, then `gradlew installDebug --offline`. From the
  contents screen, `KEYCODE_BACK` and a left-edge swipe both leave the app, no
  FATAL.
- **Not verified on the emulator:** back on a lesson, the drawer, a double
  back. **Every lesson screen crashes** there:
  `IllegalStateException: Vertically scrollable component was measured with an
  infinity maximum height constraints`. A baseline APK built from HEAD
  (6e0b685, in a scratch worktree, since removed) crashes the same way on
  lesson 2.3, so this session did not cause it. See Next 1.

Process notes: one wasted round-trip ran a `find android ios` from
`wasm/verify` after a `cd` in an earlier command. A synthetic `button.click()`
did reach Go, and the "nothing happened" was bug 2 re-opening the lesson.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 means this doc raised it; `≥ k` means it's older than
the last rebuild's window). *value* is the payoff of doing it, not the effort:
**high** means something is worked around today or a second consumer has
arrived; **medium** means it blocks one named thing or is a visible defect;
**low** means nobody has hit the gap yet. The list is sorted by age, oldest
first, and new items come last.

1. **(age ≥4 · value high) Tag a release.** v0.3.0 is the latest tag, with 25
   commits since once this one lands. Tier A, Tier B, radio roles, the sweep,
   Tier C (Menu, SearchableSelect, Drawer), system back and browser back make a
   natural v0.4.0. Still open: run
   `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean
   machine without `-replace`.
2. **(age ≥4 · value high) Run lessons 6.6, 6.7, 4.15, 4.16, 4.17 and 4.18 on a
   simulator and a device.** Blocked on Android by Next 30 (every lesson
   crashes). Unchecked on real hardware:
   - settings-row one-tap, `Screen.Footer` pinning, the iOS sheet `Dialog`,
     the stepped spinner
   - ActionSheet filler placement on Compose and SwiftUI, Timeline row
     stretch, horizontal StepIndicator scroll, radio semantics
   - CodeEditor's toolbar role on Compose, a Modal inside a ListRow's
     trailing Row (Menu)
   - SearchableSelect's Next action and keyboard dismiss on pick
   - Drawer: 100% layers filling a pinned-height ZStack on Compose and
     SwiftUI, the panel's `Height 100%` in an HStack, `AccessibilityHidden`
     confining TalkBack/VoiceOver, `core.Focus` on a Button on both natives
   - **System back:** back on a lesson returns to contents and clears
     `t.current`; with 4.18's drawer open, back closes it and a second back
     returns to contents; 4.16's AppBar `OnBack` shows its note. The root
     frame is already checked (key and gesture leave the app).
   - **Predictive back:** the swipe previews home from contents and does not
     preview-exit from a lesson or an open drawer.
   - **Double back:** two quick `KEYCODE_BACK` presses from a lesson land on
     contents and open no lesson (the `back_cb_` sequence).
3. **(age ≥4 · value med) Launch the scaffolded app on a simulator and a
   device.**
4. **(age ≥4 · value med) `docs/api/core.md` is one page of ~7,800 lines.**
5. **(age 4 · value med) Try `grmob ios -run`** once with a simulator booted
   and once without, to read the hint (`cmd/grmob/native.go:352`).
6. **(age ≥4 · value low) Migrate hand-rolls to the new comps:** the
   tutorial's `stepper` (`examples/tutorial/widgets.go:206`), `checkRow`
   (`chapter1.go:96`, also chapters 4 and 6.4), confirm flows in
   `todoapp`/`mobileapp`, the signup step header, and hand-rolled "⋯" overflow
   buttons, except `chapter6.go:734`. Retake screenshots afterwards.
7. **(age ≥4 · value low) A looping `Transition`** so `Spinner` can drop its
   stepping and Drawer could slide. `core/animation.go` has no repeat.
8. **(age ≥4 · value low) Current-item semantics.** BottomBar's ", selected",
   StepIndicator's ", done" and ", current", `SheetAction.Checked` and Drawer
   are English fallbacks. The proper shape is an `aria-current`-style state in
   core.
9. **(age ≥4 · value low) The Android remedy drill has never run.** Candidate
   for deletion.
10. **(age ≥4 · value low) `hero.png` is a picture of pictures, stored twice.**
    Candidate for deletion.
11. **(age ≥4 · value low) A third face is still a skip.** Candidate for
    deletion.
12. **(age 4 · value low) No `grmob` command refreshes a scaffolded app's
    `wasm/index.html`.** `-refresh` only re-copies `android/` and `ios/`. The
    template's page needs nothing for browser back (the runtime owns it and
    the template doesn't rewrite URLs), but the next host-page change will.
13. **(age ≥4 · non-goal) Tracked-Go-file count sentences** in `wasm/verify`
    are hand-edited when Go files are added. None were added this session.
14. **(age ≥4 · non-goal) Windowing.** Declined in `non_goals.md`;
    `TestWhatWindowingWouldSave` (`examples/tutorial/app_test.go:863`) is the
    profile.
15. **(age 4 · non-goal) Rename `docs/components.md` to `comps.md`.** The URL
    stays stable.
16. **(age 4 · non-goal) Rewrite `components` in the older plans.** Kept as
    history.
17. **(age 4 · non-goal) Trim the copied Android shell's permission
    declarations.** An app's build-time choice; see `androidPatches`.
18. **(age 4 · non-goal) Replace the iOS usage strings further.** They
    already name the app.
19. **(age 4 · non-goal) C4 `Carousel`** stays blocked until the host can
    report a scroll offset.
20. **(age 3 · value med) ARIA combobox for SearchableSelect.** A
    `RoleComboBox` with aria-expanded/aria-controls in core, plus both web
    exporters, the ARIA fixture and the runtime keyboard.
21. **(age 3 · value low) On the web, focus falls to the page after a keyboard
    pick in SearchableSelect.**
22. **(age 3 · value low) `aria-haspopup` for the Menu and DatePicker
    triggers.**
23. **(age 2 · value med) Web focus containment for Drawer.** Needs an
    `inert`-style prop in core and the web renderers.
24. **(age 2 · value low) `MaxWidth` on the natives.** Nothing in Kotlin or
    Swift reads `core.Style.MaxWidth`.
25. **(age 1 · non-goal) A parent that gains `onBack` after its descendants
    outranks them** on Android. The web ranks by document order and gets this
    case right, so the two hosts can disagree here. The built-ins nest
    correctly.
26. **(age 1 · non-goal) An AppBar placed outside the Navigator** with a
    custom `OnBack` is outranked by the Navigator's pop on Android. On the web
    it wins if it follows the Navigator in document order. Put the AppBar
    inside the route.
27. **(age 0 · value low) The reconciler sends `update-props` with
    `Changes: null`** when a node loses every prop (`reconcile/patch.go:82`).
    All three hosts now tolerate it, but a Go-side `map[string]any{}` would
    remove the trap for a hand-rolled host. Check the transcripts and goldens
    before changing it.
28. **(age 0 · value low) A typed or pasted hash while a claim is on screen
    adds a page entry above the runtime's.** The runtime then pushes a second
    entry, so back takes one extra press to unwind. Rare; revisit if a
    hashchange-routed app reports it.
29. **(age 0 · non-goal) Forward does not re-open a screen left by browser
    back.** The single entry is deliberate: the popped frame's state is gone,
    so there is nothing faithful to replay.
30. **(age 0 · value high) Every tutorial lesson crashes on Android (API
    36.1 emulator).** `IllegalStateException: Vertically scrollable component
    was measured with an infinity maximum height constraints`, thrown from
    `ScrollingLayoutNode.measure` during the first layout of a lesson screen
    (deep links `grmob://lesson/2.3` and `4.18`). The contents screen renders.
    A HEAD build (6e0b685) crashes identically, so it predates this session.
    Suspects: `comps.Screen{Scroll: true}` inside a parent that hands down
    unbounded height (the Navigator route root, or the lesson's ZStack/Fill
    chain), or a nested Scroll in a lesson body. Blocks Next 2 on Android.

Read by value instead: **high** 1, 2, 30 · **medium** 3, 4, 5, 20, 23 ·
**low** 6–12, 21, 22, 24, 27, 28 · **non-goal** 13–19, 25, 26, 29.
