# BibleStudy: tap a verse to dig deeper on Blue Letter Bible

Session: `b07ca730-9cb0-421f-a01e-59a075f7ea01`
**Date:** 2026-10-23 14:28 · **Branch:** master (3e8ecc5, plus this commit)

## Goal

The user asked for "a blueletterbible component that will show a verse or
passage and give the oppor to dig deeper on BLB". `comps.BibleVerse` and
`blb` already did the first half (ce962f8, 2be6c39), and their only way to dig
deeper was the "Read on Blue Letter Bible" link. I offered four options, and
the user chose **2: a separate `BibleStudy` widget** built on BibleVerse,
where each verse can be tapped to open a sheet of BLB study links for that
verse.

## Research: BLB's study-page addresses (checked live with curl)

- The tagger reply's passage link carries `s_<id>`. In every saved fixture
  the id is **BLB's running chapter number × 1000 + verse**:

  | Fixture | Link id | Chapter |
  |---|---|---|
  | Gen 3:1 | `s_3001` | 3 |
  | Psalm 23:1 | `s_501001` | 501 |
  | Psalm 117:1 | `s_595001` | 595 |
  | John 3:16 | `s_1000016` | 1000 |
  | Romans 1:16 | `s_1047016` | 1047 |

- A verse's study page opens on a named tab at
  `/<tr>/<book>/<ch>/<v>/t_<tool>_<id>`. The tab keys come from the page's
  `data-bible-tool` attributes:

  | Key | Tab |
  |---|---|
  | `conc` | Interlinear (the default) |
  | `bibles` | Bibles |
  | `corr` | Cross-Refs |
  | `comms` | Commentaries |
  | `refs` | Dictionaries (`t_dict_` redirects away from the verse) |
  | `misc` | Misc |

  Every valid tab returns the same HTML (200), and the page picks the tab on
  the client.
- `/kjv/jhn/3/17/t_corr_1000017` is titled "John 3 (KJV) - For God sent not
  his", so the id rule holds for a verse no reply named.

## What was built

### `blb` (`blb/blb.go`)
- `Passage.ChapterID`: parsed from the `s_` id as `id / 1000`. The
  `passageRef` regex now captures the id.
- Tool constants: `ToolInterlinear`, `ToolBibles`, `ToolCrossRefs`,
  `ToolCommentaries`, `ToolDictionaries` (`refs`) and `ToolMisc`. They are
  untyped strings, so the `p.StudyURL` method value fits a
  `func(int, string) string` field.
- `Passage.StudyURL(verse, tool)`: the chapter page from `URL`
  (`chapterPage` regex), then the verse, then `t_<tool>_<id>` (or `s_<id>`
  when tool is ""). It returns "" when there is no ChapterID or no
  passage-shaped URL.
- New tests:
  - every fixture's ChapterID, and `StudyURL(v, "")` equal to the reply's own
    link;
  - tabs and other verses;
  - the empty cases;
  - the live test now also fetches John 3:17's cross-refs page;
  - `TestBibleStudyDefaultsAreBLBTools`: blb_test imports comps (comps can't
    import blb because of net/http in wasm), so this holds the two key lists
    together.

### `comps.BibleStudy` (`comps/bible_study.go`)
- **Fields:**
  - `Passage BibleVerse`, a template, as with DatePicker's `Calendar`;
  - `StudyURL func(int, string) string`;
  - `OnStudy func(BibleVerseLine, string)`;
  - `Tools []BibleStudyTool` (default `DefaultBibleStudyTools()`, five tools,
    no Misc);
  - `Prompt`, `HidePrompt`, `VerseHint`, `CancelLabel`.
- **Hooks:** one, the open verse number (0 = closed). The widget must be
  rendered unconditionally.
- **Layout:** `Box(card, ActionSheet)`. The sheet is a sibling, as in
  DatePicker, and stays in the tree on every pass.
- **Rows:** each verse is a Row with RoleButton, the label
  "Verse N, <text>", the hint, and a `Keyed("verse:N")` key. The number is
  hung at the start (`AlignItemsStart`), and the text uses
  `FlexGrow(1)` with `FlexBasis("0")`. The prompt caption is
  `AccessibilityHidden`.
- **Tool tap:** `OnStudy` wins. Otherwise `StudyURL`, falling back to
  `Passage.URL`, then `core.OpenURL`. ActionSheet closes the sheet.
- **Sheet:** titled by `verseRef`, so "John 3:16-18" becomes
  "John 3:17 (KJV)" and "Psalms 23" becomes "Psalms 23:17". The open verse
  is looked up again on every pass, so the sheet closes itself if the
  passage changes underneath it.
- **Concerns:** `ConcernBibleStudyInert` (neither StudyURL nor OnStudy) and
  `ConcernBibleStudyNoVerses` (Text only). Both fall back to a plain
  BibleVerse.
- **`BibleVerse` refactor:** `Render` now calls `card(ctx, body)`, where a
  nil body draws the Paragraph. BibleStudy passes its rows as the body.
  BibleVerse's behaviour is unchanged.
- **Tests:** `comps/bible_study_test.go`, 10 tests. Node `Style` can be nil,
  so the predicates guard `x.Style != nil`.

### Registration and docs
- `internal/apidoc/packages.go`: added to the display topic and its blurb.
  `docs/api/` regenerated.
- `docs/components.md`: a new `## BibleStudy` section after BibleVerse.
- `ai_docs/SKILL.md` widget table (and the installed skill copy under
  `~/.claude/skills/`).
- Go file count 735 → 737 in `wasm/verify/repowalks_test.go` and
  `timings_test.go`.
- **Lesson 4.44**, "Digging deeper on Blue Letter Bible", appended to
  `examples/tutorial/chapter4.go`:
  - `cannedPassage` gained `chapterID` (1000, 501, 595) and a `studyURL`
    method that mirrors blb's;
  - `TestBibleStudyLesson`;
  - the lesson count went 86 → 87 in README (and chapter 4's row went
    43 → 44), `docs/tutorial-interactive.md`, the `wasm/index.html` header
    and `internal/shotclaims`;
  - `docs/images/tutorial-contents.png` retaken with
    `wasm/shots/shoot.sh tutorial-contents`.

## Verification
- `gofmt -l .` is clean. `go vet` passes on comps, blb and tutorial.
  `go test ./...` passes.
- `GRMOB_BLB_LIVE=1 go test ./blb/ -run Live` passes.
- Throwaway shots `zz-study-rows` and `zz-study-sheet` went to the
  scratchpad and were then deleted. Rows show hanging numbers; the sheet
  titled "Psalms 23:2 (KJV)" shows the five tools and Cancel. One cosmetic
  issue: the text start shifts with the number's width (raised as N-127).
- No device or screen-reader run (raised as N-126).

## Next

Closed: None. Declined: None. Raised: N-126, N-127.
Deferred: None. Promoted: None. Moved: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
