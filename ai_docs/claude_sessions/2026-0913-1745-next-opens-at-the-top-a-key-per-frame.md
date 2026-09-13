# Next opens at the top: a key per frame

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-13 17:45
**Branch:** master (592364f → 7c40723)

## 1. The asks

1. `/sl` loaded "DataTable rows, a sweep that never stopped, and a second back".
2. "Fix the Next scroll offset on Android" (last session's Next item 24).
3. `/sw`: this doc, commit, push.

## 2. Commits

| Commit | What | Next item |
|---|---|---|
| 7c40723 | Navigator keys each frame's root: Next opens at the top | 24 |

## 3. Cause

- `core.Navigator` emits no wrapper node, and the tutorial's lesson route
  returns a `comps.Screen` with no key. Lesson 1.3 → 1.4 (a `core.Replace`)
  was therefore the same node type at the same position. `reconcile.Diff`
  diffed one into the other in place.
- On Android, `GrMobRoot` called `RenderNode(root)` with no `key()`. Even a
  whole-tree `replace` would have kept every `remember{}` under that slot,
  `rememberScrollState` included.
- iOS had the same shape: `GrMobRoot` sits outside any `ForEach`, so its
  identity is structural. An in-place diff keeps the node instance and its
  ScrollView offset.
- The web applies a root `replace` with `el.replaceWith`, which does reset
  scroll. It still diffed in place before, because no replace was emitted.

## 4. Fix (7c40723)

- **`core/navigation.go`, `withFrameKey(id, n)`.** Navigator stamps the route's
  root with `routeScopeKey(id)` (`nav:frame:<id>`). An app's own root key is
  kept as a suffix (`nav:frame:3/form`). It writes to a copy, since the node
  may be `core.Cached` (same reasoning as `withSystemBackPop`). It runs before
  `withSystemBackPop`, so a route that claims `onBack` itself (every lesson
  does) is still keyed.
- **`reconcile/patch.go`.** The keyed-mismatch replace moved from the child
  loop to the top of `Diff`, before the type check. It now covers the node
  `Diff` starts at (`"root"`). The loop just recurses. The behaviour for
  children is the same.
- **`Renderer.kt`.** `runtime.store.root?.let { root -> key(root.key) {
  RenderNode(root) } }`. An empty key keys on `""`, as before.
- **`Renderer.swift`.** `.id(root.viewID)` on the root, the same identity rule
  `ForEach` uses for children. **Not built or run.**
- **Tests.**
  - `TestNavigatorKeysTheRouteRootByFrame`: the key is stable across passes and
    differs after Push, after Replace with the same route function, and after
    Pop (back to home's key).
  - `TestNavigatorFrameKeyKeepsTheAppKeyAndLeavesCachedNodesAlone`.
  - `TestDiffKeyedRootMismatchReplacesTheRoot` and
    `TestDiffKeyedRootSameKeyDiffsInPlace`.
- `docs/api/core-navigation.md` regenerated (`go run ./internal/apidoc/gen`).
  Its line references moved.

**Cost.** Every navigation is now a root `replace` carrying the whole new
screen: the initial-mount payload, where it used to be a diff. Passes within
one frame are unchanged, because the key is stable.

## 5. Verification

- `go test ./...`, `android/verify/sources.sh` and `sh wasm/verify/run.sh`
  (exit 0) pass.
- Emulator (checked first that no sweeps or probes were running; the only
  Gradle daemon was this session's `sources.sh`):
  - `android/build.sh ./examples/tutorial`, then `./gradlew installDebug`.
  - `am start -d grmob://lesson/1.3`, eight drags to the bottom, then a tap on
    "Next ›" found by `uiautomator dump`. **1.4 opens at its top:** "‹
    Contents", the chapter chip and the "1.4 Alignment & flex" title are on
    screen.
  - **Regression check:** scrolled 1.4 by 800px, tapped the "Center" demo
    button. Its bounds were identical before and after
    (`[451,1094][582,1147]`), and the title stayed off screen, so an in-frame
    pass still diffs in place.
- Not checked: iOS (not built), the web in a browser, hot reload's scroll
  replay after a root replace (by reading the code it restores by
  `data-node-path` two frames after the route replay, and the root keeps path
  `root`).

## 6. Process notes

- `android/build.sh <pkg>` binds the app package. The default is
  `./examples/mobileapp`, so the tutorial needs `./examples/tutorial`.
- `grmob://lesson/<n.m>` cold-starts straight into a lesson. That is quicker
  than walking the contents screen.
- zsh aborts a command when a glob matches nothing (`--include=*.go` unquoted,
  `ai_docs/.../2026-0913-1*.md`). Quote the pattern.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

1. **(age ≥10 · value high) Tag a release.** v0.3.0 is 92 commits behind with
   this doc. Pushing a tag needs the user's yes. Also still open: `go run
   github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean machine without
   `-replace`.
2. **(age ≥10 · value high) Lessons on hardware, and checks still open.**
   - Real devices for everything (deferred by the user).
   - **Screen readers:**
     - radio and StepIndicator done-step semantics
     - combobox active option
     - "pop-up" on Menu/DatePicker triggers
     - aria-current/selected on current items
     - Calendar's grid on the web
     - CodeEditor toolbar role on Compose
     - SearchableSelect natives reading field + list + status
     - AccessibilityHidden confining TalkBack/VoiceOver behind a Drawer
   - **iOS:**
     - sheet `Dialog` and ActionSheet filler
     - Drawer slide by eye and under Reduce Motion
     - RTL for Drawer and CodeEditor (needs a way to force layout direction)
     - editing in the sideways code editor (caret reveal, selection handles)
     - `banded` and a drag on 4.6's list
   - **Pinning and Drawer:**
     - `Screen.Footer` pinning
     - 100% layers in a pinned-height ZStack
     - the panel's `Height 100%` in an HStack
     - `core.Focus` on a Button
     - a hardware keyboard reaching a shut panel
   - **Android:**
     - predictive back from contents
     - MaxWidth where it binds (tablet)
     - a capped stretched child in RTL
3. **(age ≥10 · value low) ", done" and native ", today".** Natives still
   append English ", today". StepIndicator keeps ", done" (documented: no ARIA
   state fits). PageUp/PageDown do not change month, and composite arrow keys
   are not flipped for RTL.
4. **(age ≥10 · non-goal) Tracked-Go-file count sentences** in `wasm/verify` are
   hand-edited when Go files are added. Still 505 (no Go files added here).
5. **(age ≥10 · non-goal) Windowing.** Declined in `non_goals.md`.
6. **(age ≥10 · non-goal) Rename `docs/components.md` to `comps.md`.**
7. **(age ≥10 · non-goal) Rewrite `components` in the older plans.**
8. **(age ≥10 · non-goal) Trim the copied Android shell's permissions.**
9. **(age ≥10 · non-goal) Replace the iOS usage strings further.**
10. **(age ≥10 · non-goal) C4 `Carousel`** until the host reports a scroll
    offset.
11. **(age 8 · non-goal) A parent that gains `onBack` after its descendants
    outranks them** on Android; the web ranks by document order.
12. **(age 8 · non-goal) An AppBar outside the Navigator** with a custom
    `OnBack` is outranked by the Navigator's pop on Android.
13. **(age 7 · non-goal) Forward does not re-open a screen left by browser
    back.**
14. **(age 6 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose. Give the List a
    Height.
15. **(age 4 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain (`GrMobMotion`). Not profiled.
16. **(age 4 · non-goal) MaxWidth with a growing sibling on the natives.**
17. **(age 4 · non-goal) The typed-hash fold's `history.length` fallback.**
18. **(age 4 · non-goal) A page's own `pushState` while a claim is on screen.**
19. **(age 4 · non-goal) A Drawer's shut panel is composed on the natives.**
20. **(age 2 · value medium) MinWidth/MinHeight on the natives.** Documented as
    web-only, but DatePicker (MinWidth 300), the rich-text link prompt (280)
    and RichTextEditor (MinHeight) rely on them. It has to fit into
    `widthModifier`'s layout lambda on Android; iOS needs the same.
21. **(age 2 · value low) A Row in a horizontal scroll with FlexGrow children**
    has the same zero-weight collapse on Compose. `LocalGrMobUnboundedHeight`
    covers Columns only. Rows inside a bounded LazyColumn with growing children
    too.
22. **(age 2 · value low) Shot-claim field-text check** makes no RTL or
    letter-spacing adjustment (RTL fields use the box check).
23. **(age 2 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android. Give the List a Height to virtualise.
24. **(age 1 · value low) The DataTable tree walk covered initial state only.**
    Open overlays and toggled demos (4.6 Compact, 6.x sheets with growing
    content) were not walked. If a layout bug turns up there, turn the walk
    into a kept test over a few scripted states.
25. **(age 0 · value medium) The iOS root `.id(root.viewID)` is unbuilt.** Build
    with the simulator, repeat the 1.3 → bottom → "Next ›" check, and confirm
    an in-lesson tap keeps its scroll (the id must stay stable within a frame).
26. **(age 0 · value low) Next on the web and hot reload after a root
    replace.** Check in a browser that 1.4 opens at its top, and that `serve
    -dev` still puts the reader back at the same scroll offset.
27. **(age 0 · value low) "Scroll position" in PopToRoot's claims.** The
    `core.PopToRoot` doc and lesson 6's "Unwinding" prose say the root frame
    keeps its scroll position. Hook state survives; the native scroll offset
    does not, because Navigator renders only the top frame, and contents now
    reopens at its top. Either reword both or add a scroll-offset hook the
    host reports into.
