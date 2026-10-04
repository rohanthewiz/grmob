# N-093: gallery lessons for StripeCheckout, BibleVerse and Discussion

Session: `54e9d84e-f02f-4aea-a066-c92959aab421`
**Date:** 2026-10-03 20:08 · **Branch:** master (6c43114 → one commit with this doc)

## Ask

N-093 from the next-list, pasted in from the cats-todo backlog: the three
widgets added in `2026-1003-1852-…` are in `docs/components.md` and the API
pages but have no tutorial lessons. Adding lessons moves the lesson counts, so
the README, the `wasm/index.html` header, `internal/shotclaims` and the
contents screenshot move too. BibleVerse needs canned verses because the
browser tutorial cannot fetch from BLB (N-092).

## What landed

### Three lessons, appended to chapter 4

They are appended to the end of the chapter "for the reason 4.25 was", like
every round-four lesson. That keeps the earlier lessons' numbers stable.

| # | Title | Function | Test |
|---|---|---|---|
| 4.38 | Checkout with Stripe | `lessonStripeCheckout` | `TestStripeCheckoutLesson` |
| 4.39 | Quoting a Bible passage | `lessonBibleVerse` | `TestBibleVerseLesson` |
| 4.40 | Threaded comments | `lessonDiscussion` | `TestDiscussionLesson` |

- **4.38.** A `checkoutPhase` state machine (ready → redirecting → paid, or
  back to ready with an error after a decline). While Pending, two buttons
  stand in for Stripe's hosted page and the server's answer: "Payment
  succeeded" and "Card declined". The demo never opens a real Payment Link,
  because a tutorial that did could take money. A currency control
  (USD/JPY/KWD) changes the currency without converting the minor-unit
  amounts, so 5800 reads `$58.00`, `¥5,800` and `KWD 5.800`.
  `FormatMoney` puts a no-break space (U+00A0) after a currency code, and the
  test asserts it as ` `.
- **4.39.** `tutorialPassages` holds three canned KJV passages in
  `blb.Passage`'s shape: one verse, two verses, and a whole short chapter
  (Psalm 117). One verse draws no verse numbers; two or more do. The
  references use BLB's spelling ("Psalms 23:1-2"), and the URLs come from the
  `blb/testdata` fixtures with the `s_` segment dropped. A second control plays
  the fetch (Loaded/Loading/Failed), and Retry returns to Loaded. The tutorial
  **does not import `blb`**: it would put `net/http` in the wasm build for
  three fixed strings. The KJV is used because it is public domain, and the
  prose says so.
- **4.40.** A working thread held in lesson state. `editComment` copies the
  whole tree rather than just the path, so the state slot never shares slices
  with the previous pass's closures. Each new comment's key is `you-<count>`;
  the lesson only ever adds comments, so a count is never reused. The
  composer is a hook-free `core.Column` holding an InputRow, plus a "Cancel
  reply" button while replying. `MaxDepth: 2` lets a phone show the indent
  cap after a single reply to Chen.

### Counts: 80 → 83 lessons, chapter 4 37 → 40

`README.md` (alt text, sentence, chapter table), `docs/tutorial-interactive.md`,
`wasm/index.html`, `internal/shotclaims/shotclaims.go`, and the comments in
`pagecount_test.go`, `readme_counts_test.go` and `screenshot_test.go`.
`TestPageHeaderCountsTheLessons`, `TestReadmeChapterTableMatchesTheCurriculum`,
`TestDocsLessonTotalsMatchTheCurriculum` and the contents-screenshot claim test
all failed first, and passed after the change. No Go files were added, so the
tracked-file counts in `wasm/verify` did not move.

`docs/images/tutorial-contents.png` was retaken with
`wasm/shots/shoot.sh tutorial-contents`.

### Fix 1: the shot harness pins light mode (`wasm/shots/shot.mjs`)

The first retake came out **dark**. This Mac is in dark mode
(`AppleInterfaceStyle` = Dark), headless Chrome reports
`prefers-color-scheme: dark`, and since N-050 the tutorial follows the system
setting. `shot.mjs` now sends `Emulation.setEmulatedMedia` with
`prefers-color-scheme: light` before navigating. The retake then matches the
old shot, except for "83".

### Fix 2: `comps.Discussion` overflowed on a phone

Found by screenshotting 4.40 at 414px: after "Reply to Ben", the composer
(input plus Post) and the whole level of Ana's replies, Dev's too, ran past
the demo panel's right edge. A DOM probe found the cause: the replies column
beside the thread line was `flex: 1 1 0` with `min-width: auto`, so it would
not shrink below its content's min-content width. That content was the
input's intrinsic width (about 212px) plus the button: a 318px column in a
296px parent, with scrollWidth 344. It now has `core.MinWidth("0")`, the same
idiom `EditableGrid` uses. After the fix every ancestor's scrollWidth equals
its width. `TestDiscussionIndentedRepliesMayShrink` in
`comps/discussion_test.go` fails without the fix and passes with it.

### Docs

`docs/components.md` gets a one-line pointer to each widget's lesson (4.38,
4.39, 4.40).

## How it was seen

Screenshots in headless Chrome at 414px wide, using `shot.mjs` with scratch
action scripts and a scratch www dir built the way `shoot.sh` builds one.
States captured:

- 4.38: after a decline, with the discount on.
- 4.39: Psalm 23 numbered.
- 4.40: as it opens, and after posting a reply to Chen with the composer
  moved under Ben.

`shot.mjs`'s `tap` matches on textContent, so the Reply buttons (text
"Reply", aria-label "Reply to Chen") were reached with
`document.querySelector('[aria-label=…]')` and `invoke`.

**Not seen** on Android or iOS, or on a device.

Full `go test ./...` passes. `go vet` is clean on `comps` and `examples/tutorial`.

## Next

Closed: N-093. Declined: None. Raised: N-095. Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
