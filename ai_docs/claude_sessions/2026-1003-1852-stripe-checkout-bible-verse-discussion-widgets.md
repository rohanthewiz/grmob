# StripeCheckout, BibleVerse (+ blb) and Discussion widgets

Session: `441a9bc5-7b10-410b-93e1-0c5514d728ca`
**Date:** 2026-10-03 18:52 · **Branch:** master (06a6c28 → 6c4b698, plus this doc)

## Ask

From the cats-todo backlog, pasted in: add "a sweet stripe component", "a
bible verse component powered by blueletterbible", and "a discussion
component with threads".

Three answers from the user settled the ambiguity:

- "Sweet stripe" means a **Stripe checkout widget** that hands off to Stripe
  Checkout or a Payment Link. It is not a decorative stripe or a barber-pole
  progress bar.
- The Bible verse gets its text from a **widget plus a separate fetcher
  package**, so `comps` stays free of network code.
- All three go in the **grmob `comps` library**, registered in full, one
  commit per widget.

## Commits

| Commit | What |
|---|---|
| `ce962f8` | `comps.BibleVerse` and the new `blb` package |
| `599e9ec` | `comps.StripeCheckout`, `comps.FormatMoney` |
| `6c4b698` | `comps.Discussion` |

Each commit was made with only its own widget's files in the tree. The other
widgets' files were moved to the scratchpad and back, so each commit's
apidoc census and file-count sentences are correct for that commit. The
tracked-Go-file count went 680 → 684 → 686 → 688 in
`wasm/verify/repowalks_test.go` and `timings_test.go`.

## blb: where Blue Letter Bible's text comes from

BLB has no open JSON API: `api.blueletterbible.org` doesn't resolve, and
`/api/` is a 404. What it does publish for embedding is the **BLB
ScriptTagger** (blueletterbible.org/webtools). Unpacking its packed script
(`eval(function(e,t,a,i,o)…)`, run through node) showed the hover bubble is
loaded from:

    GET https://www.blueletterbible.org/remoteExtensions/toolTip/toolTipRemote.cfm
        ?id=KJV.John.3.16-18&style=par&target=true

The reply is JavaScript: `o.responseText = '...'+'...';` with an HTML
bubble inside.

- **Title:** `<h6>John 3:16-18 (KJV)</h6>`.
- **Verses:** each number is in `<span class="vRef">16</span>` when
  `style=par`. With `style=line` each verse is instead a link whose text is
  the number.
- **Link:** the first passage link is
  `https://www.blueletterbible.org/kjv/jhn/3/16/s_1000016`. The parser drops
  the `s_NNN` segment, so `Passage.URL` is the plain page.
- **Errors:** an unknown reference returns an `<h6>` of "Blue Letter Bible -
  Error" and "Invalid scripture reference.". The parser maps it to
  `ErrInvalidReference`.
- **Escaping:** quotes come through as `\'` inside the JS literals.
- **Markers:** `*` marks footnotes (ESV, NKJV). The KJV puts `[...]` around
  supplied words and `[[...]]` around psalm superscriptions. Both are
  stripped unless `Client.KeepMarkers` is set.
- **Ids:** the format is `TRANS.Book.C.V`. The book has its spaces removed
  (`1John`, `SongofSolomon`), and BLB resolves abbreviations itself. Chapter
  only (`Psa.117`), ranges and lists (`3.16,18`) all work. `PassageID` also
  maps roman ordinals (`II Kings` → `2Kings`), but only when a space follows,
  so "Isaiah" is left alone.
- **Translations checked to answer:** KJV, NKJV, NLT, NIV, ESV, CSB, NASB20,
  NASB95, LSB, AMP, NET, RSV, ASV, YLT, WEB.
- **No CORS headers.** Browser builds need a proxy via `Client.BaseURL`
  (N-092). Native builds and servers are fine.
- **API:** `blb.Fetch(ctx, ref, translation)`, a configurable `Client`
  (HTTPClient, BaseURL, Translation, KeepMarkers), `PassageID`, `SearchURL`
  (BLB's `preSearch.cfm`, a link that needs no fetch), and
  `Passage.Text()`.
- **Fixtures:** `blb/testdata/*.reply` are verbatim replies. They are named
  `.reply`, not `.js`, so no JS tooling picks them up.
- **Live test:** `GRMOB_BLB_LIVE=1 go test ./blb/ -run Live -v` passed.
- **Registration:** `blb` is in `internal/apidoc/packages.go` (group
  Widgets) and the `mkdocs.yml` nav after highlight.
  `TestPackagesCoversEveryPublicPackage` and `TestNavListsEveryPage` require
  both.

## Widget decisions

### BibleVerse (`comps/bible_verse.go`, display topic)

- A card holding a single Paragraph. Verse numbers are bold, secondary-colour
  runs joined to the first word by a no-break space, and drawn only when
  there are two or more verses (`HideNumbers` turns them off).
- `Verses` wins over `Text`.
- `Error` wins over `Loading`. The error is a `RoleStatus`, with an outlined
  Retry button (`AlignSelf` start) when `OnRetry` is set. `Loading` shows a
  three-line Skeleton named "Loading <ref>".
- The link is a `comps.Link` to "Read on Blue Letter Bible". `OnOpen` wins
  over `URL`, and with neither the link is left out.
- The card has no group role or label: with a single member (the link),
  iOS's labelled-container rule would merge the whole card into one element.
- `ConcernBibleVerseEmpty`.
- `BibleVerseLine` has the same shape as `blb.Verse`. `comps` must not import
  `blb`, because that would pull `net/http` into every browser build.

### StripeCheckout (`comps/stripe_checkout.go`, actions topic)

- There is deliberately no card field, because one puts the app in PCI scope.
  `CheckoutURL` (Payment Link or Session url) is opened with `core.OpenURL`.
  `OnPay` wins, and is where the app asks its server for a Session, since the
  secret key stays server-side.
- Amounts are `int64` minor units. `FormatMoney` covers Stripe's zero-decimal
  (16) and three-decimal (5) currency lists, uses English symbols with a code
  fallback, and puts a no-break space after a code.
- `Total()` is computed from the lines. `Quantity` ≤ 0 counts as 1.
- Each line is a Row named as one phrase, like the skill's Tally example
  ("Filters (100), 2 × $6.50, $13.00"). The title gets `headingProps`, level 2.
- `Pending` changes the label to "Redirecting to Stripe…" and disables the
  button. Button's own handling of disabled taps covers the tap race.
- `Error` is a `RoleAlert`. The note has a lock glyph (AccessibilityHidden),
  and `HideNote` removes it.
- `ConcernStripeCheckoutInert` (no OnPay or URL, and not Disabled) and
  `ConcernStripeCheckoutInsecureURL` (not `https://`).

### Discussion (`comps/discussion.go`, display topic)

- **One hook:** the set of thread keys the reader has folded or unfolded,
  relative to `InitiallyCollapsed`. Like Accordion's open state, it is purely
  presentational. The widget must be rendered every pass, and `Composer` must
  be hook-free because it moves between comments.
- A thread holding the `ReplyingTo` comment is always drawn open, so the
  composer can't be folded away.
- **Structure:** each level is a `RoleList` that contains only
  `RoleListItem`s, with `AccessibilityNestingLevel(depth+1)`. A test checks
  that lists hold only items. Composite census: RoleList isn't one of the
  claimed composites in `nested_composite_test.go`, so nothing to add there.
- **Header row:** named "Ben, reply to Ana, 1h". Natives announce that name
  and the web drops it, which is the right split, because the web already
  states the nesting level and the natives can't.
- **Thread line:** a 2px Border-coloured Box with
  `MarginHorizontal(discussionAvatar/2-1)` and Gap 0. That centres it under
  the 28pt avatar and makes the indent exactly one avatar wide. Past
  `MaxDepth` (zero means 4) replies stop indenting, but the nesting level
  keeps counting.
- **Controls:** Like is a Chip (selected state, stable name "Like, N").
  Reply is a ghost Button named "Reply to <author>". The fold is the
  package's `disclosure{Heading: false}`, with `Padding(0)` in ControlStyle
  because the disclosure row is a core.Row.
- `ConcernDiscussionReplyTargetMissing`: the composer is drawn at the top
  instead. Duplicate keys are left to core's `ConcernDuplicateKey`.

## Registration done (per the grmob-component checklist)

- `internal/apidoc/packages.go`: files and blurbs added to the topics, then
  `go run ./internal/apidoc/gen`.
- `docs/components.md`: StripeCheckout goes before InputRow, Discussion
  before ExpandableText, and BibleVerse before QRCode.
- The widget table in `ai_docs/SKILL.md`, and the installed copy at
  `~/.claude/skills/grmob-native-mobile-go/SKILL.md`.
- No canvas, chart, rowsSpec or caller-inset census entries were needed. No
  widget sets a non-zero inset on the node its Style reaches.
- Skipped, optionally: tutorial gallery lessons (N-093).

## Verification

- `gofmt -l .` printed nothing. `go vet ./comps/ ./blb/` passed, and so did
  `go test ./...`.
- Visual check: a throwaway `internal/zzpreview` test (deleted afterwards)
  wrote `htmlout.ExportHTML` of a Screen holding all three widgets, and
  headless Chrome took a screenshot at 420px. All three rendered as designed.
- The cards ran past the right edge, but a stock Card and InputRow did the
  same, so it's the bare export, not the widgets (N-094).

## Next

Closed: None. Declined: None. Raised: N-091, N-092, N-093, N-094.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
