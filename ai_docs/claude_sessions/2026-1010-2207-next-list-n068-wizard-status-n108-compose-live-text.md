# Next list, part 9 (final): N-068's Wizard status line, N-108, and what is left

Session: `1e5d7494-420f-4e59-bc6e-2acf4044d880`
**Date:** 2026-10-10 22:07 · **Branch:** master (2d2193d → f8a4a5f, plus this doc)

## Ask

The standing ask from part 1: do everything in the Next list that does not
need the user, commit and push per item, and wrap every two or three items.
This is the last wrap of the run.

## 1. N-068: the Wizard's status line on TalkBack, which found N-108

- **The probe.** Temporary, reverted: `hooks.UseInterval` advancing 4.35's
  wizard every 5s, with the emulator's TalkBack harness listening (TTS
  disabled, so utterances log).
- **Before the fix:** "Step 2: Gift note", "Selected", "Step 2: Gift note,
  done". That is the step strip item TalkBack had focused, re-read as its
  state changed. The "Step N of 3" status line was never spoken.
- **Isolating it:**
  - A lone `core.Text` with RoleStatus counting every 5s at the top of 1.4:
    silent for 32s.
  - The same Text with its words also as its AccessibilityLabel: every
    change announced, "Probe status 4" … "8".
  - So TalkBack hears a live region whose content description changes, and
    not a Text whose text changes. That is N-108. It affects every
    RoleStatus, RoleAlert or RoleLog Text without a label: the Wizard line,
    BibleVerse's error line and the like.
- **The fix:** GrMobText gives such a Text its content as its content
  description (`grMobLiveRoles`). The words are the same, so it is read once.
- **After:**
  - the plain probe was announced at every change;
  - the Wizard's line was heard at every step: "Step 3 of 3", "Step 1 of
    3", "Step 2 of 3, optional".
- **Checks:** android/verify passes; `mobile/verify/liveregion_test.go`
  pins it; wasm/verify's figure is 701.
- RenderNode's comments that still said the provider is "skipped" (stale
  since N-106) are corrected.
- Commit `f8a4a5f`.

## 2. N-002: `core.Focus` on a Button (iOS), tried

A scratch test opened and closed 4.18's drawer (CloseRef focuses the ✕, and
OnDismiss focuses the ☰ back), with and without a hardware chord first.
XCUITest reported `hasFocus` false for every element throughout, and no
focused element in the whole tree. The simulator cannot see this focus. It
stays open for a real iPad keyboard (N-004's route).

## The whole run, parts 1–9

| part | doc stem (2026-1010-) | items |
|---|---|---|
| 1 | `1919-…-n096-…-n092-…-n021-…` | N-096, N-092, N-021 closed |
| 2 | `1955-…-n043-…-n091-…-n100-…` | N-043 → Validate; N-091, N-100 closed; N-101 raised |
| 3 | `2011-…-n090-…-n102-…` | N-090, N-102 closed |
| 4 | `2027-…-n097-…-n028-…-n103-…` | N-103 closed; N-097, N-028 updated |
| 5 | `2039-…-n004-…-n065-…-n064-n104-…` | N-104 closed; N-004, N-064, N-065 updated |
| 6 | `2059-…-n070-…-n064-…-n062-…` | N-062, N-064, N-070 updated |
| 7 | `2139-…-n002-…-n106-…` | N-106 closed; N-002 updated |
| 8 | `2154-…-n002-…-n107-…` | N-107 closed; N-002 updated |
| 9 | this doc | N-108 closed; N-002, N-068 updated |

Code fixes landed, one commit each:
- N-096: the export's viewport.
- N-092: `blb.Proxy`.
- N-021: the iOS image floor.
- N-100: AlignSelf on the natives.
- N-102: the iOS keyboard reveal.
- N-103: iOS window insets.
- N-104: the Compose shadow under Opacity.
- N-106: node identity across flag flips, on both natives.
- N-107: Compose's centring of a capped Modal child.
- N-108: Compose live-region Text.
- Test-only commits: N-043 and N-004.

### What is left, and why I did not do it

- **Decisions for the user:**
  - Standing "leave it" or decline recommendations: N-013, N-015, N-027.
  - Non-goal candidates: N-045 (by design), N-051 (moot).
  - API decisions:
    - N-098: options are now written into the item;
    - N-024: needs host measurement of text;
    - N-099: SwiftUI Inert; core's doc declined `.disabled`.
  - N-009: AlarmKit or full-screen intents, which also means a permission
    and Play-policy decision.
- **Contingent:** N-003 (the next hook change), N-047 (a Compose BOM past
  foundation 1.10).
- **Hardware or a person:**
  - the Fold6: N-030, N-058's HID route, N-101, TalkBack swipe checks in
    N-062, N-066, N-068, N-070 and N-072;
  - the Mi Max 3 or an API 29 image: N-049;
  - a real iPad keyboard: N-004, N-005, N-040, and `core.Focus` on a Button;
  - a real iPhone: N-043, N-065;
  - a person holding a phone: N-006;
  - VoiceOver, which the simulator does not have, in several items;
  - hardware-backed keys: N-097.
- **Downloads I did not start unasked:** an iOS 17 runtime for N-044
  (several GB), and an API 29 system image plus a new AVD for N-049.
  N-031 shows the user decides about AVDs.
- **N-058:** diagnosing TalkBack's Tab order on the emulator. The pattern
  ("Tab read prose that Compose's focus never visits") looks like TalkBack's
  own keyboard navigation rather than the app's. Part 6 showed the
  emulator's TalkBack is drivable only by live regions, not by navigation,
  so it needs the Fold6 harness.

## Restored at the end

- **Emulator:** TalkBack off, TTS enabled, animator scale 1, rotation 0 with
  auto-rotate 1, no per-app locale, `show_ime_with_hard_keyboard` 0, Backup
  Manager disabled on the GMS transport, the tutorial installed clean, and
  POST_NOTIFICATIONS and SCHEDULE_EXACT_ALARM granted.
- **Simulator:** the iPhone 17 Pro booted (the iPhone 17 shut down),
  hardware keyboard on, Reduce Motion off, the tutorial framework built
  clean.
- No scratch tests or probes remain in the tree.

## Files touched in this part

- `android/app/src/main/java/com/grmob/runtime/Renderer.kt`
- `mobile/verify/liveregion_test.go` (new)
- `wasm/verify/{repowalks,timings}_test.go` (701)
- `ai_docs/todo/next-list.md`, this doc

## Next

Closed: N-108. Declined: None. Raised: N-108. Deferred: None. Promoted: None.
Moved: None. Updated: N-002, N-068. Full list: `ai_docs/todo/next-list.md`.
