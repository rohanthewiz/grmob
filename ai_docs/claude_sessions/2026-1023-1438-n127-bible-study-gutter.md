# N-127: BibleStudy's verse numbers share one gutter

Session: `b07ca730-9cb0-421f-a01e-59a075f7ea01`
**Date:** 2026-10-23 14:38 · **Branch:** master (497de58, plus this commit)

Continues `2026-1023-1428-bible-study-widget` in the same session.

## Goal

The user asked to fix N-127. In `comps.BibleStudy`, each verse row placed its
text right after its number, so the text started at a different x for "1"
and "2" (proportional digits), and again for "9" and "10".

## Fix (`comps/bible_study.go`)

- `verseGutter(t, verses)` returns the number column's width. It counts the
  digits of the passage's widest verse number and allows 0.62 em per digit at
  `Typography.Body.FontSize` (16 when that is zero). This is the same estimate
  `BulletList` uses for ordered markers, because no host reports a rendered
  text width. The default theme's body is 17px, so two digits come to 21.1px.
- When the passage is numbered, every row starts with the gutter:
  - the number Text gets `Width(gutter)`, `FlexShrink(0)` and
    `Align(AlignEnd)`, so the digits right-align against the text;
  - a line with `Number <= 0` gets an empty `Box` of the same width, hidden
    from readers, so its text still lines up.
- The `# Theme roles read` doc entry for Numbers now mentions the gutter.
  `docs/api/comps-display.md` was regenerated.

## Verification

- New test `TestBibleStudyNumbersShareAGutter`: verses 9 and 10 share one
  width, both are right-aligned, the width is 2 × 0.62 × the theme's body
  size, and the unnumbered line keeps an empty gutter of that width. A first
  draft hard-coded 16px and failed against the theme's 17px; the test now
  reads the size from the theme.
- `go test ./...` passes and `gofmt -l .` is clean.
- A throwaway browser shot of lesson 4.44 (Psalm 23:1-2) shows "A Psalm…"
  and "He maketh…" starting at the same x. The script was deleted afterwards.
  The 9 → 10 case is covered only by the test, because no canned passage
  crosses it.

## Next

Closed: N-127. Declined: None. Raised: None.
Deferred: None. Promoted: None. Moved: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
