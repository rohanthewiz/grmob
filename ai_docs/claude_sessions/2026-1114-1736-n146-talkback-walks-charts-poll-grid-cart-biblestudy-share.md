# N-146 and an emulator TalkBack pass: charts, Poll, grid, cart, BibleStudy, Share

Session: `94ba3d17-9a13-41bc-96af-d1aadbbafc95`

The third batch of an unattended `/work-next-list` run. The earlier docs are
`2026-1114-1650-…` and `2026-1114-1702-…`.

## 1. Setup

- **Emulator:** GrMob_Foldable (API 36), booted headless with
  `emulator -avd GrMob_Foldable -no-window -no-audio`.
  - The installed tutorial was from 00:01 today. The first launch showed a
    "System UI isn't responding" dialog; "Wait" cleared it.
- **Rebuilt from HEAD:** `android/build.sh ./examples/tutorial`, then
  `gradlew installDebug --offline`. This is the first device run of N-138's
  one-pass-per-event change. Every lesson walked behaved normally, but that
  is a smoke test, not a measurement.
- **TalkBack harness** (memory `emulator-talkback-harness`):
  - Google TTS disabled, so utterances land in logcat.
  - A virtual HID boot keyboard through `adb shell hid -` on a FIFO.
  - Alt+Right (modifier 0x04, usage 0x4F) steps swipe order; Alt+Enter
    (0x28) activates.
  - Scratch scripts in the session scratchpad: `hidkb.py`, `walk.sh`,
    `utter.py`.
- **Restored afterwards:** services `null`, `accessibility_enabled 0`, TTS
  enabled, keyboard unregistered.

## 2. N-146: not an app defect

- The walk reproduced the "silent" stop after "Copy code" on 4.40.
  `uiautomator dump` showed the code's `HorizontalScrollView` node holding
  the whole snippet as `text`. A screenshot put TalkBack's focus box on the
  code block.
- **First theory:** TalkBack does not voice a scroll view's text. I split
  the Kotlin so the scroll box and the text were separate nodes, and the
  raw log at that stop showed the snippet spoken.
- **Real cause:** reinstalling the unchanged Kotlin showed the snippet
  spoken there too. The utterance's text holds newlines, so it spans many
  logcat lines. The parser matched `text:…, earcons:` one line at a time,
  found nothing, and printed "silent".
- The Kotlin and test edits were reverted, and N-146 is closed as a harness
  artifact.
- A second parser trap was found on 4.36: a text holding `%` breaks
  TalkBack's own debug logger. The line then appears as `LogUtils: Bad
  formatting string: …` with a `MissingFormatArgumentException`, and no
  `SpeechControllerImpl` line. `utter.py` reads both now.
- Both traps are recorded in the memory.

## 3. What was heard (TalkBack, emulator)

- **N-070, charts (4.36):** all four sentences.
  - Candlestick: "ACME: 10 periods, …; 8 up, 2 down".
  - Funnel: "Checkout: … 62% of the step before; … 8% overall".
  - Radar: "Player: Ade, Speed 8, …".
  - Waveform: "Audio waveform, 30 percent played, Image".
- **N-062, Poll (4.34):** one stop per option while voting, and one per
  result in the closed poll ("Tabs, 34 percent, 1 vote"). No ProgressBar or
  text stops.
- **N-072, EditableGrid (4.37):**
  - The editor: "Editing, 310.5, Edit box, Amount, row 2".
  - Typing -5 and pressing return: "Amount, row 2. An amount cannot be
    negative." then the alert again on its own.
  - The row menu: a "Row 1" sheet with Insert below, Delete and Cancel.
- **N-123, cart (4.43):**
  - Lines read as one phrase each: "Pour-over kettle, Matte black, $45.00".
  - "1, Quantity, Pour-over kettle", "Remove Pour-over kettle",
    "Cart, 3 items, Button".
  - The refusal alert was not triggered.
- **N-126, BibleStudy (4.44):**
  - Rows read "Verse 2, He maketh me… still waters.. Opens study tools on
    Blue Letter Bible, Button". The double period is the verse's full stop
    plus the hint's join, and is not voiced.
  - Alt+Enter opened the sheet "Psalms 23:2 (KJV)" with its five tools and
    Cancel.
- **N-114, Share (4.41), with TalkBack off:** after flipping the card,
  Share opened `com.android.intentresolver`'s chooser, "Sharing text", with
  "River valley" and the picsum link together.

## 4. Raised

- **N-152 (Validate).** While 4.34's TypingIndicator showed, each step's log
  began with "Ana is typing". Idle for 12 s, nothing was logged. This may be
  an artifact of TTS being off, so it needs hearing with real speech.
- **N-153 (Open).** Row 2's Category cell in 4.37 is skipped by swipe order
  on every walk; rows 1, 3 and 4 are read. Row 2's node looks like the
  others in uiautomator. The grid's sideways scroll on entering row 2 is a
  suspect.

## Next

Closed: N-146. Declined: None. Raised: N-152, N-153. Deferred: None.
Promoted: None. Moved: None. Updated: N-062, N-070, N-072, N-114, N-123,
N-126. Full list: `ai_docs/todo/next-list.md`.
