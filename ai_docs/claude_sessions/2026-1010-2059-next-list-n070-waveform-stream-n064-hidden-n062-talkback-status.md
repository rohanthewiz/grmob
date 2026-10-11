# Next list, part 6: N-070 waveform on a stream, N-064 DisplayHidden, N-062 TalkBack and the status line

Session: `1e5d7494-420f-4e59-bc6e-2acf4044d880`
**Date:** 2026-10-10 20:59 · **Branch:** master (ddc1bbf → this doc; no code commits in this part)

## Ask

The standing ask from part 1: do everything in the Next list that does not
need the user, commit and push per item, and wrap every two or three items.
This is the sixth wrap. Every check in this part used a temporary probe in
the tutorial, reverted before anything was committed, so the only commit is
this doc and the list.

## 1. N-070: `AudioPlayer.Waveform` with a real stream, and silent bars

- **The probe.** Lesson 4.30's player got `Waveform: tutorialPeaks()`; no
  bundled screen gives a live player a waveform. The stream is SoundHelix
  Song 1 (6:13, HTTP 200).
- **Android, the API 36 emulator:**
  - Play, then screenshots: at 0:13 the first bars are blue.
  - An `adb shell input swipe` along the seek bar over 5s, with a screencap
    mid-drag: elapsed 2:31, and the strip blue up to the thumb.
  - After release, playback resumes from 3:34 (0.575 × 6:13), with the strip
    filled to it.
  - Stop: the media session's state is NONE.
- **iOS, the 26.5 simulator:**
  - At 0:09 the first bar is blue.
  - `press(forDuration:thenDragTo:withVelocity:thenHoldForDuration: 6)`
    holds the drag. XCUITest cannot screenshot during its own gesture, so the
    shell watched the test log for a "dragging" marker and took
    `xcrun simctl io … screenshot` 3.5s later. Elapsed 3:30, and the strip
    follows the glass thumb.
  - After release: 3:51.
  - The first run's extra swipe pushed the player under the status bar; it
    was rerun without it.
- **Silent bars.** Peaks 70–79 set to 0. Both natives draw the run as a row
  of round dots (four, after bucketing).

## 2. N-064: DisplayHidden watched on Compose

- **The probe.** Three 90×50 Boxes, each with a fill, a 3px black border,
  radius 8 and `Shadow(8)`. The middle one is `Display(DisplayHidden)`.
- **Result.** On the emulator the middle box drew nothing (no fill, border
  or shadow) and kept its space. This also exercises part 5's new shadow arm
  at alpha 0.

## 3. N-062: TypingIndicator's status line on TalkBack

- **The probe.** 4.34's `typing` state flipped every 4s by `hooks.UseInterval`,
  since TalkBack swallows injected taps, so the checkbox cannot be pressed
  under it.
- **The harness** ([[emulator-talkback-harness]]):
  - Google TTS disabled, so each utterance logs as "TTS is not ready".
  - TalkBack's service set, then `accessibility_enabled 1`.
  - Restored after every run: services null, enabled 0, TTS on again.
- **The runs:**
  1. One fixed name, about 60s: "TalkBack on", "Ana is typing", "GrMob",
     "Heading", then silence while the indicator kept flipping. The flipping
     was confirmed by sampling the checkbox row's pixel across frames.
  2. **A wrong turn.** I took run 1 as "a live region that arrives from
     Display none is not announced". I rebuilt the widget so its status row
     stayed in the tree, nameless while hidden, with only the visuals Display
     none, and updated its tests and the chat example's. That build was also
     heard once.
  3. Names alternating Ana and Ben, new widget, 40s: "Ben is typing", then
     later "Ana is typing".
  4. The same probe on the **old** widget (stashed), 40s: the identical
     pattern.
- **Conclusion.** The old widget's appearance is announced. TalkBack does not
  re-announce text it has just spoken from that region, which explains run
  1. The widget change bought nothing, so it was reverted with its test
  edits, and the doc's "announced when it appears" stands for TalkBack.
- **Restored:** accessibility 0, TTS enabled, and the tutorial rebuilt clean.

## What went wrong

- **The wrong turn in N-062 cost two build-and-listen cycles.** Run 1 had two
  explanations (node appearance, or repeat suppression), and the A/B
  comparison of old and new widgets with varying text should have come
  first. The edits never left the working tree.
- **My first checkbox pixel sample was in the wrong place.** The scaled
  screenshot coordinates were not converted. Two frames that happened to fall
  in the hidden phase briefly suggested the probe had stopped.

## Files touched

- `ai_docs/todo/next-list.md`, this doc. Nothing else: every probe and the
  dropped widget change were reverted.

## Next

Closed: None. Declined: None. Raised: None. Deferred: None. Promoted: None.
Moved: None. Updated: N-062, N-064, N-070. Full list:
`ai_docs/todo/next-list.md`.
